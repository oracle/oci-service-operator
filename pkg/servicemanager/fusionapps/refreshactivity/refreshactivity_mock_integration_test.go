/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package refreshactivity

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	fusionappssdk "github.com/oracle/oci-go-sdk/v65/fusionapps"
	fusionappsv1beta1 "github.com/oracle/oci-service-operator/api/fusionapps/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationRefreshActivityCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &fusionappsv1beta1.RefreshActivity{}
	ocimock.InitializeResource(resource, "mock-refreshactivity")
	resource.Spec = ocimock.MustJSONFixture[fusionappsv1beta1.RefreshActivitySpec](t, `{
  "fusionEnvironmentId": "<ocid:1>",
  "sourceFusionEnvironmentId": "<ocid:2>",
  "isDataMaskingOpted": false,
  "timeScheduledStart": "2026-09-08T12:00:00Z"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"timeScheduledStart":"2026-09-09T12:00:00Z"}`)
	createRequest := ocimock.MustJSONFixture[fusionappssdk.CreateRefreshActivityDetails](t, `{
  "sourceFusionEnvironmentId": "<ocid:2>",
  "timeScheduledStart": "2026-09-08T12:00:00Z"
}`)
	updateRequest := ocimock.MustJSONFixture[fusionappssdk.UpdateRefreshActivityDetails](t, `{
  "timeScheduledStart": "2026-09-09T12:00:00Z"
}`)
	createdState := ocimock.MustOCIResponseFixture[fusionappssdk.RefreshActivity](t, `{
  "sourceFusionEnvironmentId": "<ocid:2>",
  "isDataMaskingOpted": false,
  "timeScheduledStart": "2026-09-08T12:00:00Z",
  "id": "resource-key",
  "lifecycleState": "SUCCEEDED"
}`)
	updatedState := ocimock.MustOCIResponseFixture[fusionappssdk.RefreshActivity](t, `{
  "sourceFusionEnvironmentId": "<ocid:2>",
  "isDataMaskingOpted": false,
  "timeScheduledStart": "2026-09-09T12:00:00Z",
  "id": "resource-key",
  "lifecycleState": "SUCCEEDED"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[fusionappssdk.RefreshActivity, fusionappssdk.CreateRefreshActivityDetails, fusionappssdk.UpdateRefreshActivityDetails]{
		CollectionPath: "/20211201/fusionEnvironments/<ocid:1>/refreshActivities", ItemPath: "/20211201/fusionEnvironments/<ocid:1>/refreshActivities/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ fusionappssdk.CreateRefreshActivityDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-create"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-delete"}},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20211201/workRequests/wr-create", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", refreshActivityMockWorkRequest("wr-create", fusionappssdk.WorkRequestResourceActionTypeCreated))},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20211201/workRequests/wr-delete", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", refreshActivityMockWorkRequest("wr-delete", fusionappssdk.WorkRequestResourceActionTypeDeleted))},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://fusionapps.mock.invalid", BasePath: "20211201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := fusionappssdk.FusionApplicationsClient{BaseClient: session.BaseClient()}
	client := newRefreshActivityServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*fusionappsv1beta1.RefreshActivity]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *fusionappsv1beta1.RefreshActivity) error {
			if current.Status.TimeScheduledStart != resource.Spec.TimeScheduledStart || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created RefreshActivity status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *fusionappsv1beta1.RefreshActivity) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *fusionappsv1beta1.RefreshActivity) error {
			if current.Status.TimeScheduledStart != current.Spec.TimeScheduledStart {
				return fmt.Errorf("updated RefreshActivity status = %+v", current.Status)
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

func refreshActivityMockWorkRequest(id string, action fusionappssdk.WorkRequestResourceActionTypeEnum) fusionappssdk.WorkRequest {
	complete := float32(100)
	return fusionappssdk.WorkRequest{
		OperationType:   fusionappssdk.WorkRequestOperationTypeRefreshFusionEnvironment,
		Status:          fusionappssdk.WorkRequestStatusSucceeded,
		Id:              common.String(id),
		CompartmentId:   common.String("<ocid:3>"),
		PercentComplete: &complete,
		Resources: []fusionappssdk.WorkRequestResource{{
			EntityType: common.String("RefreshActivity"), ActionType: action, Identifier: common.String("resource-key"),
		}},
	}
}
