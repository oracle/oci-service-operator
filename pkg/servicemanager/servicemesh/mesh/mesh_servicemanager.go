/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mesh

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshConversions "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	pkgErrors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceManager struct {
	client            client.Client
	serviceMeshClient services.ServiceMeshClient
	log               loggerutil.OSOKLogger
}

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=meshes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=meshes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=meshes/finalizers,verbs=update

func NewMeshResourceManager(client client.Client, serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger) manager.CustomResourceHandler {
	return &ResourceManager{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
	}
}

func (m *ResourceManager) GetResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	mesh, err := getMesh(object)
	if err != nil {
		return err
	}
	if len(mesh.Status.MeshId) > 0 {
		sdkMesh, err := m.serviceMeshClient.GetMesh(ctx, &mesh.Status.MeshId)
		if err != nil {
			return err
		}
		details.MeshDetails.SdkMesh = sdkMesh
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "mesh did not sync to the control plane", "name", mesh.ObjectMeta.Name)
	return nil
}

func (m *ResourceManager) CreateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) (bool, error) {
	if details.MeshDetails.SdkMesh == nil {
		m.log.InfoLogWithFixedMessage(ctx, "creating mesh...", "name", object.GetName(), "OpcRetryToken", *details.OpcRetryToken)
		sdkMesh, err := m.serviceMeshClient.CreateMesh(ctx, details.MeshDetails.BuildSdkMesh, details.OpcRetryToken)
		if err != nil {
			return false, err
		}
		details.MeshDetails.SdkMesh = sdkMesh
		return false, nil
	}
	return true, nil
}

func (m *ResourceManager) UpdateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	m.log.InfoLogWithFixedMessage(ctx, "updating mesh...", "name", object.GetName())
	details.MeshDetails.SdkMesh.LifecycleState = sdk.MeshLifecycleStateUpdating
	return m.serviceMeshClient.UpdateMesh(ctx, details.MeshDetails.BuildSdkMesh)
}

func (m *ResourceManager) GetTimeUpdated(details *manager.ResourceDetails) *common.SDKTime {
	return details.MeshDetails.SdkMesh.TimeUpdated
}

func (m *ResourceManager) ChangeCompartment(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	mesh, err := getMesh(object)
	if err != nil {
		return err
	}
	m.log.InfoLogWithFixedMessage(ctx, "Moving mesh to new compartment", "Name", object.GetName())
	// sdkMesh state would be Active here, hence update the state to updating for correct status
	details.MeshDetails.SdkMesh.LifecycleState = sdk.MeshLifecycleStateUpdating
	return m.serviceMeshClient.ChangeMeshCompartment(ctx, &mesh.Status.MeshId, &mesh.Spec.CompartmentId)
}

func (m *ResourceManager) DeleteResource(ctx context.Context, object client.Object) error {
	mesh, err := getMesh(object)
	if err != nil {
		return err
	}
	if len(mesh.Status.MeshId) == 0 {
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "Deleting mesh", "Name", object.GetName())
	return m.serviceMeshClient.DeleteMesh(ctx, &mesh.Status.MeshId)
}

func (m *ResourceManager) VerifyEntityType(object runtime.Object) (client.Object, error) {
	mesh, ok := object.(*servicemeshapi.Mesh)
	if !ok {
		return nil, errors.New("object is not a mesh")
	}
	return mesh, nil
}

func (m *ResourceManager) VerifyResourceStatus(details *manager.ResourceDetails) (bool, error) {
	if details.MeshDetails.SdkMesh == nil || details.MeshDetails.SdkMesh.LifecycleState == sdk.MeshLifecycleStateActive {
		return true, nil
	}

	state := details.MeshDetails.SdkMesh.LifecycleState
	// Terminate the reconcile request if resource in the control plane is deleted or failed
	if state == sdk.MeshLifecycleStateDeleted || state == sdk.MeshLifecycleStateFailed {
		return false, meshErrors.NewDoNotRequeueError(errors.New("mesh in the control plane is deleted or failed"))
	}

	return false, meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
}

func (m *ResourceManager) GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	mesh, err := getMesh(object)
	if err != nil {
		return nil, err
	}
	return &mesh.Status, nil
}

func (m *ResourceManager) GetConditionStatus(details *manager.ResourceDetails) metav1.ConditionStatus {
	state := details.MeshDetails.SdkMesh.LifecycleState
	return meshCommons.GetConditionStatus(string(state))
}

func (m *ResourceManager) UpdateStatus(object client.Object, details *manager.ResourceDetails) (bool, error) {
	needsUpdate := false
	mesh, err := getMesh(object)
	if err != nil {
		return false, err
	}
	// Update MeshId
	if mesh.Status.MeshId != api.OCID(*details.MeshDetails.SdkMesh.Id) {
		mesh.Status.MeshId = api.OCID(*details.MeshDetails.SdkMesh.Id)
		needsUpdate = true
	}

	mtls, err := meshConversions.ConvertSdkMeshMTlsToCrdMeshMTls(details.MeshDetails.SdkMesh.Mtls)
	if err != nil {
		return false, err
	}

	if mtls != nil && !cmp.Equal(mesh.Status.MeshMtls, mtls) {
		mesh.Status.MeshMtls = mtls
		needsUpdate = true
	}

	// update operator time iff cp has different updated time
	// (we cannot change cp updated time)
	// so change operator updated time to match cp, so resource isnt reconsidered during next reconciliation
	cpTime := details.MeshDetails.SdkMesh.TimeUpdated
	operatorTime := mesh.Status.LastUpdatedTime

	if manager.DoTimeStampsDiffer(cpTime, operatorTime) {
		mesh.Status.LastUpdatedTime = (*metav1.Time)(details.MeshDetails.SdkMesh.TimeUpdated)
		needsUpdate = true
	}

	return needsUpdate, nil
}

func (m *ResourceManager) HasCompartmentIdChanged(object client.Object, details *manager.ResourceDetails) (bool, error) {
	mesh, err := getMesh(object)
	if err != nil {
		return false, err
	}
	return mesh.Spec.CompartmentId != api.OCID(*details.MeshDetails.SdkMesh.CompartmentId), nil
}

func (m *ResourceManager) BuildSdk(object client.Object, details *manager.ResourceDetails) error {
	mesh, err := getMesh(object)
	if err != nil {
		return err
	}
	buildSdkMesh := &sdk.Mesh{}
	if details.MeshDetails.SdkMesh != nil {
		buildSdkMesh.FreeformTags = details.MeshDetails.SdkMesh.FreeformTags
		buildSdkMesh.DefinedTags = details.MeshDetails.SdkMesh.DefinedTags
	}
	conversionErr := meshConversions.ConvertCrdMeshToSdkMesh(mesh, buildSdkMesh)
	if conversionErr != nil {
		return conversionErr
	}
	details.MeshDetails.BuildSdkMesh = buildSdkMesh
	return nil
}

func (m *ResourceManager) GetMessage(details *manager.ResourceDetails) string {
	state := details.MeshDetails.SdkMesh.LifecycleState
	return string(meshCommons.GetMessage(string(state)))
}

func (m *ResourceManager) GetLifecycleState(details *manager.ResourceDetails) string {
	return string(details.MeshDetails.SdkMesh.LifecycleState)
}

func (m *ResourceManager) ResolveDependencies(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	return nil
}

func (m *ResourceManager) GetFinalizer() string {
	return meshCommons.MeshFinalizer
}

func (m *ResourceManager) Finalize(ctx context.Context, object client.Object) error {
	mesh, err := getMesh(object)
	if err != nil {
		return err
	}
	var message []string
	hasVsMembers, err := m.hasVirtualServiceMembers(ctx, mesh)
	if err != nil {
		return err
	}

	if hasVsMembers {
		message = append(message, "virtualServices")
	}

	hasApMembers, err := m.hasAccessPolicyMembers(ctx, mesh)
	if err != nil {
		return err
	}

	if hasApMembers {
		message = append(message, "accessPolicies")
	}

	hasIgMembers, err := m.hasIngressGatewayMembers(ctx, mesh)
	if err != nil {
		return err
	}

	if hasIgMembers {
		message = append(message, "ingressGateways")
	}

	if len(message) > 0 {
		return fmt.Errorf("mesh has pending subresources to be deleted: %s", strings.Join(message, ", "))
	}

	return nil
}

func (m *ResourceManager) hasVirtualServiceMembers(ctx context.Context, mesh *servicemeshapi.Mesh) (bool, error) {
	vsList := &servicemeshapi.VirtualServiceList{}
	hasVsMembers := false

	if err := m.client.List(ctx, vsList); err != nil {
		return false, pkgErrors.Wrapf(err, "unable to list virtualServices associated with mesh: %s for namespace: %s", mesh.Name, mesh.Namespace)
	}

	for _, vs := range vsList.Items {
		if vs.Status.MeshId == mesh.Status.MeshId {
			hasVsMembers = true
			break
		}
	}

	return hasVsMembers, nil
}

func (m *ResourceManager) hasAccessPolicyMembers(ctx context.Context, mesh *servicemeshapi.Mesh) (bool, error) {
	apList := &servicemeshapi.AccessPolicyList{}
	hasApMembers := false

	if err := m.client.List(ctx, apList); err != nil {
		return false, pkgErrors.Wrapf(err, "unable to list accessPolicies associated with mesh: %s for namespace: %s", mesh.Name, mesh.Namespace)
	}

	for _, ap := range apList.Items {
		if ap.Status.MeshId == mesh.Status.MeshId {
			hasApMembers = true
			break
		}
	}

	return hasApMembers, nil
}

func (m *ResourceManager) hasIngressGatewayMembers(ctx context.Context, mesh *servicemeshapi.Mesh) (bool, error) {
	igList := &servicemeshapi.IngressGatewayList{}
	hasIgMembers := false

	if err := m.client.List(ctx, igList); err != nil {
		return false, pkgErrors.Wrapf(err, "unable to list ingressGateways associated with mesh: %s for namespace: %s", mesh.Name, mesh.Namespace)
	}

	for _, ig := range igList.Items {
		if ig.Status.MeshId == mesh.Status.MeshId {
			hasIgMembers = true
			break
		}
	}

	return hasIgMembers, nil
}

func getMesh(object client.Object) (*servicemeshapi.Mesh, error) {
	mesh, ok := object.(*servicemeshapi.Mesh)
	if !ok {
		return nil, errors.New("object is not a mesh")
	}
	return mesh, nil
}

func (m *ResourceManager) HasSdk(details *manager.ResourceDetails) bool {
	return details.MeshDetails.SdkMesh != nil
}
