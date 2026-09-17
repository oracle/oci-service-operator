/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cluster

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockClusterID = "ocid1.cluster.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/containerengine/cluster and formal/imports/containerengine/cluster.json
//   - resource runtime: cluster_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/containerengine
func TestMockIntegrationClusterLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &containerenginev1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-cluster", Namespace: "default", UID: types.UID("mock-cluster-uid")},
		Spec: containerenginev1beta1.ClusterSpec{
			Name:              "mock-cluster",
			CompartmentId:     "ocid1.compartment.oc1..mock",
			VcnId:             "ocid1.vcn.oc1..mock",
			KubernetesVersion: "v1.36.1",
			EndpointConfig: containerenginev1beta1.ClusterEndpointConfig{
				SubnetId:          "ocid1.subnet.oc1..endpoint",
				IsPublicIpEnabled: false,
			},
			Options: containerenginev1beta1.ClusterOptions{
				ServiceLbSubnetIds: []string{"ocid1.subnet.oc1..service-lb"},
			},
			Type:         "BASIC_CLUSTER",
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newClusterMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://containerengine.mock.invalid", BasePath: "20180222", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Cluster OCI mock: %v", err)
		}
	})

	client := newMockClusterClient(containerenginesdk.ContainerEngineClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*containerenginev1beta1.Cluster]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *containerenginev1beta1.Cluster) error {
			if current.Status.Id != mockClusterID ||
				current.Status.Name != "mock-cluster" ||
				current.Status.Type != "BASIC_CLUSTER" ||
				current.Status.LifecycleState != string(containerenginesdk.ClusterLifecycleStateActive) {
				return fmt.Errorf("created Cluster status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *containerenginev1beta1.Cluster) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *containerenginev1beta1.Cluster) error {
			if current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(containerenginesdk.ClusterLifecycleStateActive) {
				return fmt.Errorf("updated Cluster status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newClusterMockResponder(resource *containerenginev1beta1.Cluster) (*ocimock.CRUDResponder[containerenginesdk.Cluster], error) {
	createReadObserved := false
	updateReadObserved := false
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[containerenginesdk.Cluster]{
		CollectionPath:         "/20180222/clusters",
		ItemPath:               "/20180222/clusters/" + mockClusterID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		List: func(request ocimock.Request, present bool, state containerenginesdk.Cluster) (ocimock.Response, error) {
			if request.URL.Query().Get("compartmentId") != resource.Spec.CompartmentId ||
				request.URL.Query().Get("name") != resource.Spec.Name {
				return ocimock.Response{}, fmt.Errorf("unexpected ListClusters query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []containerenginesdk.Cluster{})
			}
			return ocimock.JSONResponse(http.StatusOK, []containerenginesdk.Cluster{state})
		},
		Create: func(request ocimock.Request) (containerenginesdk.Cluster, ocimock.Response, error) {
			var details containerenginesdk.CreateClusterDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return containerenginesdk.Cluster{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return containerenginesdk.Cluster{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.VcnId == nil || *details.VcnId != resource.Spec.VcnId ||
				details.KubernetesVersion == nil || *details.KubernetesVersion != resource.Spec.KubernetesVersion ||
				details.EndpointConfig == nil || details.EndpointConfig.SubnetId == nil ||
				*details.EndpointConfig.SubnetId != resource.Spec.EndpointConfig.SubnetId ||
				details.Type != containerenginesdk.ClusterTypeBasicCluster {
				return containerenginesdk.Cluster{}, ocimock.Response{}, fmt.Errorf("unexpected CreateCluster details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return containerenginesdk.Cluster{}, ocimock.Response{}, fmt.Errorf("CreateCluster opc-retry-token is empty")
			}
			state := containerenginesdk.Cluster{
				Id:                common.String(mockClusterID),
				Name:              details.Name,
				CompartmentId:     details.CompartmentId,
				VcnId:             details.VcnId,
				KubernetesVersion: details.KubernetesVersion,
				EndpointConfig: &containerenginesdk.ClusterEndpointConfig{
					SubnetId:          details.EndpointConfig.SubnetId,
					IsPublicIpEnabled: details.EndpointConfig.IsPublicIpEnabled,
				},
				Options:        details.Options,
				Type:           details.Type,
				FreeformTags:   details.FreeformTags,
				LifecycleState: containerenginesdk.ClusterLifecycleStateCreating,
			}
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{
				"Opc-Request-Id":      []string{"mock-cluster-create-request"},
				"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-cluster-create"},
			}
			return state, response, nil
		},
		ReadTransition: func(request ocimock.Request, state containerenginesdk.Cluster) (containerenginesdk.Cluster, ocimock.Response, error) {
			if request.URL.Query().Get("shouldIncludeOidcConfigFile") != "true" {
				return containerenginesdk.Cluster{}, ocimock.Response{}, fmt.Errorf("GetCluster omitted shouldIncludeOidcConfigFile=true")
			}
			switch state.LifecycleState {
			case containerenginesdk.ClusterLifecycleStateCreating:
				if createReadObserved {
					state.LifecycleState = containerenginesdk.ClusterLifecycleStateActive
				} else {
					createReadObserved = true
				}
			case containerenginesdk.ClusterLifecycleStateUpdating:
				if updateReadObserved {
					state.LifecycleState = containerenginesdk.ClusterLifecycleStateActive
				} else {
					updateReadObserved = true
				}
			case containerenginesdk.ClusterLifecycleStateDeleting:
				if deleteReadObserved {
					state.LifecycleState = containerenginesdk.ClusterLifecycleStateDeleted
				} else {
					deleteReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state containerenginesdk.Cluster) (containerenginesdk.Cluster, ocimock.Response, error) {
			var details containerenginesdk.UpdateClusterDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return containerenginesdk.Cluster{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" {
				return containerenginesdk.Cluster{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateCluster details: %+v", details)
			}
			state.FreeformTags = details.FreeformTags
			state.LifecycleState = containerenginesdk.ClusterLifecycleStateUpdating
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{
				"Opc-Request-Id":      []string{"mock-cluster-update-request"},
				"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-cluster-update"},
			}
			return state, response, nil
		},
		DeleteTransition: func(_ ocimock.Request, state containerenginesdk.Cluster) (containerenginesdk.Cluster, ocimock.Response, error) {
			state.LifecycleState = containerenginesdk.ClusterLifecycleStateDeleting
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{
				"Opc-Request-Id":      []string{"mock-cluster-delete-request"},
				"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-cluster-delete"},
			}
			return state, response, nil
		},
	})
}
