/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package peer

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	blockchainsdk "github.com/oracle/oci-go-sdk/v65/blockchain"
	"github.com/oracle/oci-go-sdk/v65/common"
	blockchainv1beta1 "github.com/oracle/oci-service-operator/api/blockchain/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationPeerCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &blockchainv1beta1.Peer{}
	ocimock.InitializeResource(resource, "mock-peer")
	resource.Spec = ocimock.MustJSONFixture[blockchainv1beta1.PeerSpec](t, `{
  "blockchainPlatformId": "<ocid:1>",
  "role": "MEMBER",
  "ad": "AD1",
  "alias": "peer-create",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 1.0
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"ocpuAllocationParam":{"ocpuAllocationNumber":2.0}}`)
	createRequest := ocimock.MustJSONFixture[blockchainsdk.CreatePeerDetails](t, `{
  "role": "MEMBER",
  "ad": "AD1",
  "alias": "peer-create",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 1.0
  }
}`)
	updateRequest := ocimock.MustJSONFixture[blockchainsdk.UpdatePeerDetails](t, `{
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 2.0
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[blockchainsdk.Peer](t, `{
  "role": "MEMBER",
  "ad": "AD1",
  "alias": "peer-create",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 1.0
  },
  "peerKey": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[blockchainsdk.Peer](t, `{
  "role": "MEMBER",
  "ad": "AD1",
  "alias": "peer-create",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 2.0
  },
  "peerKey": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[blockchainsdk.Peer, blockchainsdk.CreatePeerDetails, blockchainsdk.UpdatePeerDetails]{
		CollectionPath: "/20191010/blockchainPlatforms/<ocid:1>/peers", ItemPath: "/20191010/blockchainPlatforms/<ocid:1>/peers/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ blockchainsdk.CreatePeerDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-create"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-delete"}},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/wr-create", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", peerWorkRequest("wr-create", "CREATED"))},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/wr-update", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", peerWorkRequest("wr-update", "UPDATED"))},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/wr-delete", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", peerWorkRequest("wr-delete", "DELETED"))},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://blockchain.mock.invalid", BasePath: "20191010", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := blockchainsdk.BlockchainPlatformClient{BaseClient: session.BaseClient()}
	client := newPeerServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*blockchainv1beta1.Peer]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *blockchainv1beta1.Peer) error {
			if current.Status.Alias != resource.Spec.Alias || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Peer status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *blockchainv1beta1.Peer) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *blockchainv1beta1.Peer) error {
			if current.Status.OcpuAllocationParam.OcpuAllocationNumber != current.Spec.OcpuAllocationParam.OcpuAllocationNumber {
				return fmt.Errorf("updated Peer status = %+v", current.Status)
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

func peerWorkRequest(id string, action string) blockchainsdk.WorkRequest {
	return blockchainsdk.WorkRequest{
		Id: common.String(id), OperationType: blockchainsdk.WorkRequestOperationTypeUpdatePlatform,
		Status: blockchainsdk.WorkRequestStatusSucceeded, CompartmentId: common.String("<ocid:2>"), PercentComplete: func() *float32 { value := float32(100); return &value }(),
		Resources: []blockchainsdk.WorkRequestResource{{EntityType: common.String("peer"), ActionType: blockchainsdk.WorkRequestResourceActionTypeEnum(action), Identifier: common.String("resource-key")}},
	}
}
