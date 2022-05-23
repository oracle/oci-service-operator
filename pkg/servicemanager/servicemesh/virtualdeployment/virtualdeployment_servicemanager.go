/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshConversions "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceManager struct {
	client            client.Client
	serviceMeshClient services.ServiceMeshClient
	log               loggerutil.OSOKLogger
	referenceResolver references.Resolver
}

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualdeployments/finalizers,verbs=update

func NewVirtualDeploymentResourceManager(client client.Client, serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger, referenceResolver references.Resolver) manager.CustomResourceHandler {
	return &ResourceManager{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
		referenceResolver: referenceResolver,
	}
}

func (m *ResourceManager) GetResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return err
	}
	if len(virtualDeployment.Status.VirtualDeploymentId) > 0 {
		sdkVirtualDeployment, err := m.serviceMeshClient.GetVirtualDeployment(ctx, &virtualDeployment.Status.VirtualDeploymentId)
		if err != nil {
			return err
		}
		details.VdDetails.SdkVd = sdkVirtualDeployment
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "virtualDeployment did not sync to the control plane", "name", virtualDeployment.ObjectMeta.Name)
	return nil
}

func (m *ResourceManager) CreateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) (bool, error) {
	if details.VdDetails.SdkVd == nil {
		m.log.InfoLogWithFixedMessage(ctx, "creating virtualDeployment..", "name", object.GetName(), "OpcRetryToken", *details.OpcRetryToken)
		sdkVirtualDeployment, err := m.serviceMeshClient.CreateVirtualDeployment(ctx, details.VdDetails.BuildSdkVd, details.OpcRetryToken)
		if err != nil {
			return false, err
		}
		details.VdDetails.SdkVd = sdkVirtualDeployment
		return false, nil
	}
	return true, nil
}

func (m *ResourceManager) UpdateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	m.log.InfoLogWithFixedMessage(ctx, "updating virtualDeployment...", "name", object.GetName())
	details.VdDetails.SdkVd.LifecycleState = sdk.VirtualDeploymentLifecycleStateUpdating
	return m.serviceMeshClient.UpdateVirtualDeployment(ctx, details.VdDetails.BuildSdkVd)
}

func (m *ResourceManager) ChangeCompartment(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return err
	}
	m.log.InfoLogWithFixedMessage(ctx, "Moving virtualDeployment to new compartment", "Name", object.GetName())
	// sdkVirtualDeployment state would be Active here, hence update the state to updating for correct status
	details.VdDetails.SdkVd.LifecycleState = sdk.VirtualDeploymentLifecycleStateUpdating
	return m.serviceMeshClient.ChangeVirtualDeploymentCompartment(ctx, &virtualDeployment.Status.VirtualDeploymentId, &virtualDeployment.Spec.CompartmentId)
}

func (m *ResourceManager) DeleteResource(ctx context.Context, object client.Object) error {
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return err
	}
	if len(virtualDeployment.Status.VirtualDeploymentId) == 0 {
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "Deleting virtual deployment", "Name", object.GetName())
	return m.serviceMeshClient.DeleteVirtualDeployment(ctx, &virtualDeployment.Status.VirtualDeploymentId)
}

func (m *ResourceManager) VerifyEntityType(object runtime.Object) (client.Object, error) {
	virtualDeployment, ok := object.(*servicemeshapi.VirtualDeployment)
	if !ok {
		return nil, errors.New("object is not a virtual deployment")
	}
	return virtualDeployment, nil
}

func (m *ResourceManager) VerifyResourceStatus(details *manager.ResourceDetails) (bool, error) {
	if details.VdDetails.SdkVd == nil || details.VdDetails.SdkVd.LifecycleState == sdk.VirtualDeploymentLifecycleStateActive {
		return true, nil
	}

	state := details.VdDetails.SdkVd.LifecycleState
	// Terminate the reconcile request if resource in the control plane is deleted or failed
	if state == sdk.VirtualDeploymentLifecycleStateDeleted || state == sdk.VirtualDeploymentLifecycleStateFailed {
		return false, meshErrors.NewDoNotRequeueError(errors.New("virtual deployment in the control plane is deleted or failed"))
	}

	return false, meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
}

func (m *ResourceManager) GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return nil, err
	}
	return &virtualDeployment.Status, nil
}

func (m *ResourceManager) GetConditionStatus(details *manager.ResourceDetails) metav1.ConditionStatus {
	state := details.VdDetails.SdkVd.LifecycleState
	return meshCommons.GetConditionStatus(string(state))
}

func (m *ResourceManager) UpdateStatus(object client.Object, details *manager.ResourceDetails) (bool, error) {
	needsUpdate := false
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return false, err
	}
	if virtualDeployment.Status.VirtualServiceId == "" {
		virtualDeployment.Status.VirtualServiceId = api.OCID(*details.VdDetails.SdkVd.VirtualServiceId)
		virtualDeployment.Status.VirtualServiceName = details.VdDetails.VsRef.Name
		virtualDeployment.Status.MeshId = details.VdDetails.VsRef.MeshId
		needsUpdate = true
	}

	if virtualDeployment.Status.VirtualDeploymentId != api.OCID(*details.VdDetails.SdkVd.Id) {
		virtualDeployment.Status.VirtualDeploymentId = api.OCID(*details.VdDetails.SdkVd.Id)
		needsUpdate = true
	}

	// update operator time iff cp has different updated time
	// (we cannot change cp updated time)
	// so change operator updated time to match cp, so resource isnt reconsidered during next reconciliation
	cpTime := details.VdDetails.SdkVd.TimeUpdated
	operatorTime := virtualDeployment.Status.LastUpdatedTime
	if manager.DoTimeStampsDiffer(cpTime, operatorTime) {
		virtualDeployment.Status.LastUpdatedTime = (*metav1.Time)(details.VdDetails.SdkVd.TimeUpdated)
		needsUpdate = true
	}

	return needsUpdate, nil
}

func (m *ResourceManager) GetMessage(details *manager.ResourceDetails) string {
	state := details.VdDetails.SdkVd.LifecycleState
	return string(meshCommons.GetMessage(string(state)))
}

func (m *ResourceManager) BuildSdk(object client.Object, details *manager.ResourceDetails) error {
	buildSdkVirtualDeployment := &sdk.VirtualDeployment{}
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return err
	}
	meshConversions.ConvertCrdVirtualDeploymentToSdkVirtualDeployment(virtualDeployment, buildSdkVirtualDeployment, &details.VdDetails.VsRef.Id)
	details.VdDetails.BuildSdkVd = buildSdkVirtualDeployment
	return nil
}

func (m *ResourceManager) HasCompartmentIdChanged(object client.Object, details *manager.ResourceDetails) (bool, error) {
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return false, err
	}
	return virtualDeployment.Spec.CompartmentId != api.OCID(*details.VdDetails.SdkVd.CompartmentId), nil
}

func (m *ResourceManager) GetLifecycleState(details *manager.ResourceDetails) string {
	return string(details.VdDetails.SdkVd.LifecycleState)
}

func (m *ResourceManager) ResolveDependencies(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return err
	}
	details.VdDetails.VsRef = &meshCommons.ResourceRef{
		Id:     virtualDeployment.Status.VirtualServiceId,
		Name:   virtualDeployment.Status.VirtualServiceName,
		MeshId: virtualDeployment.Status.MeshId,
	}

	if details.VdDetails.VsRef.Id == "" {
		virtualServiceRef, err := m.referenceResolver.ResolveVirtualServiceIdAndName(ctx, &virtualDeployment.Spec.VirtualService, &virtualDeployment.ObjectMeta)
		details.VdDetails.VsRef = virtualServiceRef
		return err
	}
	return nil
}

func (m *ResourceManager) GetTimeUpdated(details *manager.ResourceDetails) *common.SDKTime {
	return details.VdDetails.SdkVd.TimeUpdated
}

func (m *ResourceManager) GetFinalizer() string {
	return meshCommons.VirtualDeploymentFinalizer
}

func (m *ResourceManager) Finalize(ctx context.Context, object client.Object) error {
	virtualDeployment, err := getVirtualDeployment(object)
	if err != nil {
		return err
	}
	if hasDependencies, err := m.hasVirtualDeploymentBindings(ctx, virtualDeployment); err != nil {
		return err
	} else if hasDependencies {
		return errors.New("cannot delete virtual deployment when there are virtual deployment binding resources associated")
	}

	if hasDependencies, err := m.hasVirtualServiceRouteTables(ctx, virtualDeployment); err != nil {
		return err
	} else if hasDependencies {
		return errors.New("cannot delete virtual deployment when there are virtual service route table resources associated")
	}

	return nil
}

func (m *ResourceManager) hasVirtualDeploymentBindings(ctx context.Context, virtualDeployment *servicemeshapi.VirtualDeployment) (bool, error) {
	vdbList := &servicemeshapi.VirtualDeploymentBindingList{}

	if err := m.client.List(ctx, vdbList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list virtualDeploymentBindings associated with virtualDeployment", "name", virtualDeployment.Name, "namespace", virtualDeployment.Namespace)
		return false, err
	}

	for _, vdb := range vdbList.Items {
		if vdb.Status.VirtualDeploymentId == virtualDeployment.Status.VirtualDeploymentId {
			return true, nil
		}
	}

	return false, nil
}

func (m *ResourceManager) hasVirtualServiceRouteTables(ctx context.Context, virtualDeployment *servicemeshapi.VirtualDeployment) (bool, error) {
	vsrtList := &servicemeshapi.VirtualServiceRouteTableList{}

	if err := m.client.List(ctx, vsrtList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list virtualServiceRouteTables for virtualDeployment", "name", virtualDeployment.Name, "namespace", virtualDeployment.Namespace)
		return false, err
	}

	for _, vsrt := range vsrtList.Items {
		if virtualDeployment.Status.VirtualServiceId == vsrt.Status.VirtualServiceId {
			for _, rule := range vsrt.Status.VirtualDeploymentIdForRules {
				for _, virtualDeploymentId := range rule {
					if virtualDeployment.Status.VirtualDeploymentId == virtualDeploymentId {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func getVirtualDeployment(object client.Object) (*servicemeshapi.VirtualDeployment, error) {
	virtualDeployment, ok := object.(*servicemeshapi.VirtualDeployment)
	if !ok {
		return nil, errors.New("object is not a virtual deployment")
	}
	return virtualDeployment, nil
}

func (m *ResourceManager) HasSdk(details *manager.ResourceDetails) bool {
	return details.VdDetails.SdkVd != nil
}
