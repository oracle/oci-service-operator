/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nodepool

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

const mockNodePoolID = "ocid1.nodepool.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/containerengine/nodepool and formal/imports/containerengine/nodepool.json
//   - resource runtime: nodepool_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/containerengine
func TestMockIntegrationNodePoolLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &containerenginev1beta1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-node-pool", Namespace: "default", UID: types.UID("mock-node-pool-uid")},
		Spec: containerenginev1beta1.NodePoolSpec{
			CompartmentId:     "ocid1.compartment.oc1..mock",
			ClusterId:         "ocid1.cluster.oc1..mock",
			Name:              "mock-node-pool",
			NodeShape:         "VM.Standard.E3.Flex",
			KubernetesVersion: "v1.36.1",
			NodeMetadata: map[string]string{
				"areLegacyImdsEndpointsDisabled": "true",
			},
			NodeSourceDetails: containerenginev1beta1.NodePoolNodeSourceDetails{
				SourceType:          "IMAGE",
				ImageId:             "ocid1.image.oc1..mock",
				BootVolumeSizeInGBs: 50,
			},
			NodeShapeConfig: containerenginev1beta1.NodePoolNodeShapeConfig{
				Ocpus:       1,
				MemoryInGBs: 16,
			},
			NodeConfigDetails: containerenginev1beta1.NodePoolNodeConfigDetails{
				Size: 1,
				PlacementConfigs: []containerenginev1beta1.NodePoolNodeConfigDetailsPlacementConfig{{
					AvailabilityDomain: "mock:AD-1",
					SubnetId:           "ocid1.subnet.oc1..nodes",
				}},
				NodePoolPodNetworkOptionDetails: containerenginev1beta1.NodePoolNodeConfigDetailsNodePoolPodNetworkOptionDetails{
					CniType:      "OCI_VCN_IP_NATIVE",
					PodSubnetIds: []string{"ocid1.subnet.oc1..pods"},
				},
			},
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newNodePoolMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://containerengine.mock.invalid", BasePath: "20180222", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close NodePool OCI mock: %v", err)
		}
	})

	client := newMockNodePoolClient(containerenginesdk.ContainerEngineClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*containerenginev1beta1.NodePool]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *containerenginev1beta1.NodePool) error {
			if current.Status.Id != mockNodePoolID ||
				current.Status.Name != "mock-node-pool" ||
				current.Status.NodeMetadata["areLegacyImdsEndpointsDisabled"] != "true" ||
				current.Status.LifecycleState != string(containerenginesdk.NodePoolLifecycleStateActive) {
				return fmt.Errorf("created NodePool status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *containerenginev1beta1.NodePool) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *containerenginev1beta1.NodePool) error {
			if current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.NodeMetadata["areLegacyImdsEndpointsDisabled"] != "true" ||
				current.Status.LifecycleState != string(containerenginesdk.NodePoolLifecycleStateActive) {
				return fmt.Errorf("updated NodePool status = %+v", current.Status)
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

func newNodePoolMockResponder(resource *containerenginev1beta1.NodePool) (*ocimock.CRUDResponder[containerenginesdk.NodePool], error) {
	createReadObserved := false
	updateReadObserved := false
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[containerenginesdk.NodePool]{
		CollectionPath:         "/20180222/nodePools",
		ItemPath:               "/20180222/nodePools/" + mockNodePoolID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		List: func(request ocimock.Request, present bool, state containerenginesdk.NodePool) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("compartmentId") != resource.Spec.CompartmentId ||
				query.Get("clusterId") != resource.Spec.ClusterId ||
				query.Get("name") != resource.Spec.Name {
				return ocimock.Response{}, fmt.Errorf("unexpected ListNodePools query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []containerenginesdk.NodePool{})
			}
			return ocimock.JSONResponse(http.StatusOK, []containerenginesdk.NodePool{state})
		},
		Create: func(request ocimock.Request) (containerenginesdk.NodePool, ocimock.Response, error) {
			var details containerenginesdk.CreateNodePoolDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return containerenginesdk.NodePool{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return containerenginesdk.NodePool{}, ocimock.Response{}, err
			}
			source, ok := details.NodeSourceDetails.(containerenginesdk.NodeSourceViaImageDetails)
			if !ok ||
				source.ImageId == nil || *source.ImageId != resource.Spec.NodeSourceDetails.ImageId ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.ClusterId == nil || *details.ClusterId != resource.Spec.ClusterId ||
				details.Name == nil || *details.Name != resource.Spec.Name ||
				details.NodeShape == nil || *details.NodeShape != resource.Spec.NodeShape ||
				details.NodeMetadata["areLegacyImdsEndpointsDisabled"] != "true" ||
				details.NodeConfigDetails == nil || details.NodeConfigDetails.Size == nil ||
				*details.NodeConfigDetails.Size != 1 || len(details.NodeConfigDetails.PlacementConfigs) != 1 ||
				len(details.SubnetIds) != 0 {
				return containerenginesdk.NodePool{}, ocimock.Response{}, fmt.Errorf("unexpected CreateNodePool details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return containerenginesdk.NodePool{}, ocimock.Response{}, fmt.Errorf("CreateNodePool opc-retry-token is empty")
			}
			placement := details.NodeConfigDetails.PlacementConfigs
			podNetwork := containerenginesdk.OciVcnIpNativeNodePoolPodNetworkOptionDetails{
				PodSubnetIds: append([]string(nil), resource.Spec.NodeConfigDetails.NodePoolPodNetworkOptionDetails.PodSubnetIds...),
			}
			state := containerenginesdk.NodePool{
				Id:                common.String(mockNodePoolID),
				CompartmentId:     details.CompartmentId,
				ClusterId:         details.ClusterId,
				Name:              details.Name,
				KubernetesVersion: details.KubernetesVersion,
				NodeMetadata:      details.NodeMetadata,
				InitialNodeLabels: []containerenginesdk.KeyValue{},
				NodeShape:         details.NodeShape,
				NodeShapeConfig: &containerenginesdk.NodeShapeConfig{
					Ocpus:       details.NodeShapeConfig.Ocpus,
					MemoryInGBs: details.NodeShapeConfig.MemoryInGBs,
				},
				NodeSourceDetails: source,
				NodeConfigDetails: &containerenginesdk.NodePoolNodeConfigDetails{
					Size:                            details.NodeConfigDetails.Size,
					PlacementConfigs:                placement,
					IsPvEncryptionInTransitEnabled:  common.Bool(false),
					NodePoolPodNetworkOptionDetails: podNetwork,
				},
				NodeEvictionNodePoolSettings: &containerenginesdk.NodeEvictionNodePoolSettings{
					IsForceDeleteAfterGraceDuration: common.Bool(false),
					IsForceActionAfterGraceDuration: common.Bool(false),
				},
				NodePoolCyclingDetails: &containerenginesdk.NodePoolCyclingDetails{
					IsNodeCyclingEnabled: common.Bool(false),
				},
				FreeformTags:   details.FreeformTags,
				LifecycleState: containerenginesdk.NodePoolLifecycleStateCreating,
			}
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{
				"Opc-Request-Id":      []string{"mock-node-pool-create-request"},
				"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-node-pool-create"},
			}
			return state, response, nil
		},
		ReadTransition: func(_ ocimock.Request, state containerenginesdk.NodePool) (containerenginesdk.NodePool, ocimock.Response, error) {
			switch state.LifecycleState {
			case containerenginesdk.NodePoolLifecycleStateCreating:
				if createReadObserved {
					state.LifecycleState = containerenginesdk.NodePoolLifecycleStateActive
				} else {
					createReadObserved = true
				}
			case containerenginesdk.NodePoolLifecycleStateUpdating:
				if updateReadObserved {
					state.LifecycleState = containerenginesdk.NodePoolLifecycleStateActive
				} else {
					updateReadObserved = true
				}
			case containerenginesdk.NodePoolLifecycleStateDeleting:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state containerenginesdk.NodePool) (containerenginesdk.NodePool, ocimock.Response, error) {
			var details containerenginesdk.UpdateNodePoolDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return containerenginesdk.NodePool{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" ||
				details.NodeMetadata["areLegacyImdsEndpointsDisabled"] != "true" ||
				details.NodeConfigDetails == nil ||
				len(details.SubnetIds) != 0 {
				return containerenginesdk.NodePool{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateNodePool details: %+v", details)
			}
			state.FreeformTags = details.FreeformTags
			state.NodeMetadata = details.NodeMetadata
			state.LifecycleState = containerenginesdk.NodePoolLifecycleStateUpdating
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{
				"Opc-Request-Id":      []string{"mock-node-pool-update-request"},
				"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-node-pool-update"},
			}
			return state, response, nil
		},
		DeleteTransition: func(_ ocimock.Request, state containerenginesdk.NodePool) (containerenginesdk.NodePool, ocimock.Response, error) {
			state.LifecycleState = containerenginesdk.NodePoolLifecycleStateDeleting
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{
				"Opc-Request-Id":      []string{"mock-node-pool-delete-request"},
				"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-node-pool-delete"},
			}
			return state, response, nil
		},
	})
}
