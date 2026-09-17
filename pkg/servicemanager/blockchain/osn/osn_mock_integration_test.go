/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package osn

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

func TestMockIntegrationOsnCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &blockchainv1beta1.Osn{}
	ocimock.InitializeResource(resource, "mock-osn")
	resource.Spec = ocimock.MustJSONFixture[blockchainv1beta1.OsnSpec](t, `{
  "blockchainPlatformId": "<ocid:1>",
  "ad": "AD1",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 1.0
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"ocpuAllocationParam":{"ocpuAllocationNumber":2.0}}`)
	createRequest := ocimock.MustJSONFixture[blockchainsdk.CreateOsnDetails](t, `{
  "ad": "AD1",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 1.0
  }
}`)
	updateRequest := ocimock.MustJSONFixture[blockchainsdk.UpdateOsnDetails](t, `{
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 2.0
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[blockchainsdk.Osn](t, `{
  "ad": "AD1",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 1.0
  },
  "osnKey": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[blockchainsdk.Osn](t, `{
  "ad": "AD1",
  "ocpuAllocationParam": {
    "ocpuAllocationNumber": 2.0
  },
  "osnKey": "resource-key",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[blockchainsdk.Osn, blockchainsdk.CreateOsnDetails, blockchainsdk.UpdateOsnDetails]{
		CollectionPath: "/20191010/blockchainPlatforms/<ocid:1>/osns", ItemPath: "/20191010/blockchainPlatforms/<ocid:1>/osns/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ blockchainsdk.CreateOsnDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-create"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-delete"}},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/wr-create", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", osnWorkRequest("wr-create", "CREATED"))},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/wr-update", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", osnWorkRequest("wr-update", "UPDATED"))},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20191010/workRequests/wr-delete", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", osnWorkRequest("wr-delete", "DELETED"))},
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
	client := newOsnServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*blockchainv1beta1.Osn]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *blockchainv1beta1.Osn) error {
			if current.Status.Ad != resource.Spec.Ad || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Osn status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *blockchainv1beta1.Osn) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *blockchainv1beta1.Osn) error {
			if current.Status.OcpuAllocationParam.OcpuAllocationNumber != current.Spec.OcpuAllocationParam.OcpuAllocationNumber {
				return fmt.Errorf("updated Osn status = %+v", current.Status)
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

func osnWorkRequest(id string, action string) blockchainsdk.WorkRequest {
	return blockchainsdk.WorkRequest{
		Id: common.String(id), OperationType: blockchainsdk.WorkRequestOperationTypeUpdatePlatform,
		Status: blockchainsdk.WorkRequestStatusSucceeded, CompartmentId: common.String("<ocid:2>"), PercentComplete: func() *float32 { value := float32(100); return &value }(),
		Resources: []blockchainsdk.WorkRequestResource{{EntityType: common.String("osn"), ActionType: blockchainsdk.WorkRequestResourceActionTypeEnum(action), Identifier: common.String("resource-key")}},
	}
}
