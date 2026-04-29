/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opsiconfiguration

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeOpsiConfigurationOCIClient struct {
	createFn      func(context.Context, opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationResponse, error)
	getFn         func(context.Context, opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error)
	listFn        func(context.Context, opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error)
	updateFn      func(context.Context, opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error)
	deleteFn      func(context.Context, opsisdk.DeleteOpsiConfigurationRequest) (opsisdk.DeleteOpsiConfigurationResponse, error)
	workRequestFn func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeOpsiConfigurationOCIClient) CreateOpsiConfiguration(ctx context.Context, req opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return opsisdk.CreateOpsiConfigurationResponse{}, fmt.Errorf("unexpected CreateOpsiConfiguration call")
}

func (f *fakeOpsiConfigurationOCIClient) GetOpsiConfiguration(ctx context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return opsisdk.GetOpsiConfigurationResponse{}, fmt.Errorf("unexpected GetOpsiConfiguration call")
}

func (f *fakeOpsiConfigurationOCIClient) ListOpsiConfigurations(ctx context.Context, req opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return opsisdk.ListOpsiConfigurationsResponse{}, fmt.Errorf("unexpected ListOpsiConfigurations call")
}

func (f *fakeOpsiConfigurationOCIClient) UpdateOpsiConfiguration(ctx context.Context, req opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return opsisdk.UpdateOpsiConfigurationResponse{}, fmt.Errorf("unexpected UpdateOpsiConfiguration call")
}

func (f *fakeOpsiConfigurationOCIClient) DeleteOpsiConfiguration(ctx context.Context, req opsisdk.DeleteOpsiConfigurationRequest) (opsisdk.DeleteOpsiConfigurationResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return opsisdk.DeleteOpsiConfigurationResponse{}, fmt.Errorf("unexpected DeleteOpsiConfiguration call")
}

func (f *fakeOpsiConfigurationOCIClient) GetWorkRequest(ctx context.Context, req opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return opsisdk.GetWorkRequestResponse{}, fmt.Errorf("unexpected GetWorkRequest call")
}

//nolint:gocyclo // This single contract test keeps the generatedruntime semantic surface together.
func TestOpsiConfigurationRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := newOpsiConfigurationRuntimeSemantics()
	if got == nil {
		t.Fatal("newOpsiConfigurationRuntimeSemantics() = nil")
	}
	if got.FormalService != "opsi" {
		t.Fatalf("FormalService = %q, want opsi", got.FormalService)
	}
	if got.FormalSlug != "opsiconfiguration" {
		t.Fatalf("FormalSlug = %q, want opsiconfiguration", got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest", got.Async)
	}
	if got.Async.WorkRequest == nil || got.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest = %#v, want service-sdk", got.Async.WorkRequest)
	}
	assertOpsiConfigurationStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	assertOpsiConfigurationStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertOpsiConfigurationStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertOpsiConfigurationStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertOpsiConfigurationStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertOpsiConfigurationStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertOpsiConfigurationStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "opsiConfigType"})
	assertOpsiConfigurationStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags", "systemTags", "jsonData"})
	assertOpsiConfigurationStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "opsiConfigType"})
}

func TestBuildOpsiConfigurationCreateBodyUsesUXJsonDataWithSpecDefaults(t *testing.T) {
	t.Parallel()

	resource := makeOpsiConfigurationResource()
	resource.Spec.JsonData = `{"opsiConfigType":"UX_CONFIGURATION","configItems":[{"configItemType":"BASIC","name":"ui.theme","value":"dark"}]}`

	body, err := buildOpsiConfigurationCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("buildOpsiConfigurationCreateBody() error = %v", err)
	}
	details, ok := body.(opsisdk.CreateOpsiUxConfigurationDetails)
	if !ok {
		t.Fatalf("create body type = %T, want CreateOpsiUxConfigurationDetails", body)
	}
	requireOpsiConfigurationStringPtr(t, "compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireOpsiConfigurationStringPtr(t, "displayName", details.DisplayName, resource.Spec.DisplayName)
	requireOpsiConfigurationStringPtr(t, "description", details.Description, resource.Spec.Description)
	if got := details.FreeformTags["env"]; got != "test" {
		t.Fatalf("freeformTags.env = %q, want test", got)
	}
	if got := fmt.Sprint(details.DefinedTags["Operations"]["CostCenter"]); got != "42" {
		t.Fatalf("definedTags.Operations.CostCenter = %q, want 42", got)
	}
	if len(details.ConfigItems) != 1 {
		t.Fatalf("configItems len = %d, want 1", len(details.ConfigItems))
	}
	item, ok := details.ConfigItems[0].(opsisdk.CreateBasicConfigurationItemDetails)
	if !ok {
		t.Fatalf("configItems[0] type = %T, want CreateBasicConfigurationItemDetails", details.ConfigItems[0])
	}
	requireOpsiConfigurationStringPtr(t, "configItems[0].name", item.Name, "ui.theme")
	requireOpsiConfigurationStringPtr(t, "configItems[0].value", item.Value, "dark")

	payload, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("json.Marshal(create details) error = %v", err)
	}
	if !strings.Contains(string(payload), `"opsiConfigType":"UX_CONFIGURATION"`) {
		t.Fatalf("create details JSON = %s, want UX discriminator", payload)
	}
}

func TestBuildOpsiConfigurationCreateBodyRejectsJsonDataSpecConflict(t *testing.T) {
	t.Parallel()

	resource := makeOpsiConfigurationResource()
	resource.Spec.JsonData = `{"opsiConfigType":"UX_CONFIGURATION","compartmentId":"ocid1.compartment.oc1..other"}`

	_, err := buildOpsiConfigurationCreateBody(context.Background(), resource, "default")
	if err == nil || !strings.Contains(err.Error(), "jsonData conflicts") || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("buildOpsiConfigurationCreateBody() error = %v, want compartment conflict", err)
	}
}

//nolint:gocognit,gocyclo // The create test exercises one two-step work-request flow end to end.
func TestOpsiConfigurationCreateUsesWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.opsiconfiguration.oc1..created"
		workRequestID = "wr-opsiconfiguration-create"
	)
	resource := makeOpsiConfigurationResource()
	workRequests := map[string]opsisdk.WorkRequest{
		workRequestID: makeOpsiConfigurationWorkRequest(workRequestID, opsisdk.OperationTypeCreateOpsiConfiguration, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, ""),
	}
	var createRequest opsisdk.CreateOpsiConfigurationRequest
	getCalls := 0

	client := newTestOpsiConfigurationClient(&fakeOpsiConfigurationOCIClient{
		listFn: func(_ context.Context, req opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error) {
			requireOpsiConfigurationStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireOpsiConfigurationStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if len(req.OpsiConfigType) != 0 {
				t.Fatalf("list opsiConfigType = %#v, want empty so filtering stays in formal match", req.OpsiConfigType)
			}
			return opsisdk.ListOpsiConfigurationsResponse{}, nil
		},
		createFn: func(_ context.Context, req opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationResponse, error) {
			createRequest = req
			return opsisdk.CreateOpsiConfigurationResponse{
				OpsiConfiguration: makeSDKOpsiConfiguration(createdID, resource, opsisdk.OpsiConfigurationLifecycleStateCreating),
				OpcWorkRequestId:  common.String(workRequestID),
				OpcRequestId:      common.String("opc-create-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireOpsiConfigurationStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return opsisdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			getCalls++
			requireOpsiConfigurationStringPtr(t, "get opsiConfigurationId", req.OpsiConfigurationId, createdID)
			requireOpsiConfigurationGetResponseFields(t, req)
			return opsisdk.GetOpsiConfigurationResponse{
				OpsiConfiguration: makeSDKOpsiConfiguration(createdID, resource, opsisdk.OpsiConfigurationLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while work request is pending", response)
	}
	details, ok := createRequest.CreateOpsiConfigurationDetails.(opsisdk.CreateOpsiUxConfigurationDetails)
	if !ok {
		t.Fatalf("create body type = %T, want CreateOpsiUxConfigurationDetails", createRequest.CreateOpsiConfigurationDetails)
	}
	requireOpsiConfigurationStringPtr(t, "create compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireOpsiConfigurationStringPtr(t, "create displayName", details.DisplayName, resource.Spec.DisplayName)
	if createRequest.OpcRetryToken == nil || strings.TrimSpace(*createRequest.OpcRetryToken) == "" {
		t.Fatal("create opc retry token is empty")
	}
	requireOpsiConfigurationCreateResponseFields(t, createRequest)
	requireOpsiConfigurationAsyncCurrent(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
	if getCalls != 0 {
		t.Fatalf("GetOpsiConfiguration() calls = %d, want 0 while work request is pending", getCalls)
	}

	workRequests[workRequestID] = makeOpsiConfigurationWorkRequest(workRequestID, opsisdk.OperationTypeCreateOpsiConfiguration, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeCreated, createdID)
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want successful no requeue", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetOpsiConfiguration() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(opsisdk.OpsiConfigurationLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil", resource.Status.OsokStatus.Async.Current)
	}
}

//nolint:gocyclo // Pagination, bind, and no-create assertions are coupled in this regression.
func TestOpsiConfigurationBindsExistingFromPaginatedList(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.opsiconfiguration.oc1..existing"
	resource := makeOpsiConfigurationResource()
	listCalls := 0
	createCalled := false
	updateCalled := false

	client := newTestOpsiConfigurationClient(&fakeOpsiConfigurationOCIClient{
		listFn: func(_ context.Context, req opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error) {
			listCalls++
			requireOpsiConfigurationStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireOpsiConfigurationStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			switch listCalls {
			case 1:
				if req.Page != nil {
					t.Fatalf("first list page token = %q, want nil", *req.Page)
				}
				return opsisdk.ListOpsiConfigurationsResponse{
					OpsiConfigurationsCollection: opsisdk.OpsiConfigurationsCollection{
						Items: []opsisdk.OpsiConfigurationSummary{
							makeSDKOpsiConfigurationSummary("ocid1.opsiconfiguration.oc1..other", "different", resource, opsisdk.OpsiConfigurationLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireOpsiConfigurationStringPtr(t, "second list page token", req.Page, "page-2")
				return opsisdk.ListOpsiConfigurationsResponse{
					OpsiConfigurationsCollection: opsisdk.OpsiConfigurationsCollection{
						Items: []opsisdk.OpsiConfigurationSummary{
							makeSDKOpsiConfigurationSummary(existingID, resource.Spec.DisplayName, resource, opsisdk.OpsiConfigurationLifecycleStateActive),
						},
					},
				}, nil
			default:
				t.Fatalf("ListOpsiConfigurations() call %d, want at most 2", listCalls)
				return opsisdk.ListOpsiConfigurationsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			requireOpsiConfigurationStringPtr(t, "get opsiConfigurationId", req.OpsiConfigurationId, existingID)
			return opsisdk.GetOpsiConfigurationResponse{
				OpsiConfiguration: makeSDKOpsiConfiguration(existingID, resource, opsisdk.OpsiConfigurationLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationResponse, error) {
			createCalled = true
			return opsisdk.CreateOpsiConfigurationResponse{}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error) {
			updateCalled = true
			return opsisdk.UpdateOpsiConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if createCalled {
		t.Fatal("CreateOpsiConfiguration() should not be called when paginated list finds a match")
	}
	if updateCalled {
		t.Fatal("UpdateOpsiConfiguration() should not be called for matching existing resource")
	}
	if listCalls != 2 {
		t.Fatalf("ListOpsiConfigurations() calls = %d, want 2", listCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestOpsiConfigurationNoOpDoesNotUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.opsiconfiguration.oc1..noop"
	resource := makeOpsiConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID

	client := newTestOpsiConfigurationClient(&fakeOpsiConfigurationOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			requireOpsiConfigurationStringPtr(t, "get opsiConfigurationId", req.OpsiConfigurationId, existingID)
			return opsisdk.GetOpsiConfigurationResponse{
				OpsiConfiguration: makeSDKOpsiConfiguration(existingID, resource, opsisdk.OpsiConfigurationLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
}

func TestOpsiConfigurationProjectsObservedJsonDataConfigItems(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.opsiconfiguration.oc1..observed-json"
	resource := makeOpsiConfigurationResource()
	resource.Status.OsokStatus.OpcRequestID = "opc-before"

	if err := opsiConfigurationStatusFromResponse(resource, opsisdk.GetOpsiConfigurationResponse{
		OpsiConfiguration: makeSDKOpsiConfiguration(existingID, resource, opsisdk.OpsiConfigurationLifecycleStateActive),
	}); err != nil {
		t.Fatalf("opsiConfigurationStatusFromResponse() error = %v", err)
	}

	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-before" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-before", got)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.OpsiConfigType; got != opsiConfigurationTypeUX {
		t.Fatalf("status.opsiConfigType = %q, want %q", got, opsiConfigurationTypeUX)
	}
	assertOpsiConfigurationStatusJSONData(t, resource.Status.JsonData, "ui.theme", "dark")
}

func TestOpsiConfigurationMutableUpdateStartsWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.opsiconfiguration.oc1..update"
		workRequestID = "wr-opsiconfiguration-update"
	)
	resource := makeOpsiConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	var updateRequest opsisdk.UpdateOpsiConfigurationRequest

	client := newTestOpsiConfigurationClient(&fakeOpsiConfigurationOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			requireOpsiConfigurationStringPtr(t, "get opsiConfigurationId", req.OpsiConfigurationId, existingID)
			current := makeSDKOpsiConfiguration(existingID, resource, opsisdk.OpsiConfigurationLifecycleStateActive)
			current.Description = common.String("old description")
			current.FreeformTags = map[string]string{"env": "old"}
			return opsisdk.GetOpsiConfigurationResponse{OpsiConfiguration: current}, nil
		},
		updateFn: func(_ context.Context, req opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error) {
			updateRequest = req
			return opsisdk.UpdateOpsiConfigurationResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireOpsiConfigurationStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeOpsiConfigurationWorkRequest(workRequestID, opsisdk.OperationTypeUpdateOpsiConfiguration, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, existingID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while update work request is pending", response)
	}
	requireOpsiConfigurationStringPtr(t, "update opsiConfigurationId", updateRequest.OpsiConfigurationId, existingID)
	details, ok := updateRequest.UpdateOpsiConfigurationDetails.(opsisdk.UpdateOpsiUxConfigurationDetails)
	if !ok {
		t.Fatalf("update body type = %T, want UpdateOpsiUxConfigurationDetails", updateRequest.UpdateOpsiConfigurationDetails)
	}
	requireOpsiConfigurationStringPtr(t, "update description", details.Description, resource.Spec.Description)
	if got := details.FreeformTags["env"]; got != "test" {
		t.Fatalf("update freeformTags.env = %q, want test", got)
	}
	requireOpsiConfigurationAsyncCurrent(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseUpdate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestOpsiConfigurationRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.opsiconfiguration.oc1..force-new"
	resource := makeOpsiConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	updateCalled := false

	client := newTestOpsiConfigurationClient(&fakeOpsiConfigurationOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			requireOpsiConfigurationStringPtr(t, "get opsiConfigurationId", req.OpsiConfigurationId, existingID)
			current := makeSDKOpsiConfiguration(existingID, resource, opsisdk.OpsiConfigurationLifecycleStateActive)
			current.CompartmentId = common.String("ocid1.compartment.oc1..old")
			return opsisdk.GetOpsiConfigurationResponse{OpsiConfiguration: current}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error) {
			updateCalled = true
			return opsisdk.UpdateOpsiConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement") || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only compartment drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalled {
		t.Fatal("UpdateOpsiConfiguration() should not be called after create-only drift rejection")
	}
}

func TestOpsiConfigurationRejectsCreateOnlyDriftFromJsonData(t *testing.T) {
	t.Parallel()

	resource := makeOpsiConfigurationResource()
	resource.Spec.CompartmentId = ""
	resource.Spec.JsonData = `{"opsiConfigType":"UX_CONFIGURATION","compartmentId":"ocid1.compartment.oc1..new","displayName":"opsi-configuration"}`

	err := validateOpsiConfigurationCreateOnlyDrift(resource, opsisdk.GetOpsiConfigurationResponse{
		OpsiConfiguration: opsisdk.OpsiUxConfiguration{
			Id:             common.String("ocid1.opsiconfiguration.oc1..jsondrift"),
			CompartmentId:  common.String("ocid1.compartment.oc1..old"),
			DisplayName:    common.String(resource.Spec.DisplayName),
			LifecycleState: opsisdk.OpsiConfigurationLifecycleStateActive,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "jsonData.compartmentId") {
		t.Fatalf("validateOpsiConfigurationCreateOnlyDrift() error = %v, want jsonData compartment drift", err)
	}
}

func TestBuildOpsiConfigurationUpdateBodyShapesJsonDataConfigItems(t *testing.T) {
	t.Parallel()

	resource := makeOpsiConfigurationResource()
	resource.Spec.JsonData = `{"opsiConfigType":"UX_CONFIGURATION","compartmentId":"ocid1.compartment.oc1..opsi","displayName":"opsi-configuration","configItems":[{"configItemType":"BASIC","name":"ui.theme","value":"light"}]}`
	current := opsisdk.GetOpsiConfigurationResponse{
		OpsiConfiguration: opsisdk.OpsiUxConfiguration{
			Id:             common.String("ocid1.opsiconfiguration.oc1..configitems"),
			CompartmentId:  common.String(resource.Spec.CompartmentId),
			DisplayName:    common.String(resource.Spec.DisplayName),
			LifecycleState: opsisdk.OpsiConfigurationLifecycleStateActive,
			ConfigItems: []opsisdk.OpsiConfigurationConfigurationItemSummary{
				opsisdk.OpsiConfigurationBasicConfigurationItemSummary{
					Name:  common.String("ui.theme"),
					Value: common.String("dark"),
				},
			},
		},
	}

	body, updateNeeded, err := buildOpsiConfigurationUpdateBody(context.Background(), resource, "default", current)
	if err != nil {
		t.Fatalf("buildOpsiConfigurationUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildOpsiConfigurationUpdateBody() updateNeeded = false, want true for config item drift")
	}
	details, ok := body.(opsisdk.UpdateOpsiUxConfigurationDetails)
	if !ok {
		t.Fatalf("update body type = %T, want UpdateOpsiUxConfigurationDetails", body)
	}
	if len(details.ConfigItems) != 1 {
		t.Fatalf("update configItems len = %d, want 1", len(details.ConfigItems))
	}
	item, ok := details.ConfigItems[0].(opsisdk.UpdateBasicConfigurationItemDetails)
	if !ok {
		t.Fatalf("update configItems[0] type = %T, want UpdateBasicConfigurationItemDetails", details.ConfigItems[0])
	}
	requireOpsiConfigurationStringPtr(t, "update configItems[0].name", item.Name, "ui.theme")
	requireOpsiConfigurationStringPtr(t, "update configItems[0].value", item.Value, "light")
}

//nolint:gocyclo // The delete test keeps pending and confirmed work-request passes in one flow.
func TestOpsiConfigurationDeleteRetainsFinalizerUntilWorkRequestConfirmsDeleted(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.opsiconfiguration.oc1..delete"
		workRequestID = "wr-opsiconfiguration-delete"
	)
	resource := makeOpsiConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	workRequests := map[string]opsisdk.WorkRequest{
		workRequestID: makeOpsiConfigurationWorkRequest(workRequestID, opsisdk.OperationTypeDeleteOpsiConfiguration, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, existingID),
	}
	getDeleted := false
	deleteCalls := 0

	client := newTestOpsiConfigurationClient(&fakeOpsiConfigurationOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			requireOpsiConfigurationStringPtr(t, "get opsiConfigurationId", req.OpsiConfigurationId, existingID)
			state := opsisdk.OpsiConfigurationLifecycleStateActive
			if getDeleted {
				state = opsisdk.OpsiConfigurationLifecycleStateDeleted
			}
			return opsisdk.GetOpsiConfigurationResponse{
				OpsiConfiguration: makeSDKOpsiConfiguration(existingID, resource, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req opsisdk.DeleteOpsiConfigurationRequest) (opsisdk.DeleteOpsiConfigurationResponse, error) {
			deleteCalls++
			requireOpsiConfigurationStringPtr(t, "delete opsiConfigurationId", req.OpsiConfigurationId, existingID)
			return opsisdk.DeleteOpsiConfigurationResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireOpsiConfigurationStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return opsisdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while delete work request is pending")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteOpsiConfiguration() calls = %d, want 1", deleteCalls)
	}
	requireOpsiConfigurationAsyncCurrent(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseDelete, workRequestID, shared.OSOKAsyncClassPending)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil while delete confirmation is pending")
	}

	getDeleted = true
	workRequests[workRequestID] = makeOpsiConfigurationWorkRequest(workRequestID, opsisdk.OperationTypeDeleteOpsiConfiguration, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, existingID)
	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() after work request success error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after work request and DELETED readback")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteOpsiConfiguration() calls = %d, want no duplicate delete after work request", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after confirmed delete", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOpsiConfigurationDeleteRejectsAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.opsiconfiguration.oc1..ambiguous"
	resource := makeOpsiConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID

	client := newTestOpsiConfigurationClient(&fakeOpsiConfigurationOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			requireOpsiConfigurationStringPtr(t, "get opsiConfigurationId", req.OpsiConfigurationId, existingID)
			return opsisdk.GetOpsiConfigurationResponse{
				OpsiConfiguration: makeSDKOpsiConfiguration(existingID, resource, opsisdk.OpsiConfigurationLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req opsisdk.DeleteOpsiConfigurationRequest) (opsisdk.DeleteOpsiConfigurationResponse, error) {
			requireOpsiConfigurationStringPtr(t, "delete opsiConfigurationId", req.OpsiConfigurationId, existingID)
			return opsisdk.DeleteOpsiConfigurationResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous delete")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 rejection", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false for auth-shaped not found")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil for ambiguous 404")
	}
}

func newTestOpsiConfigurationClient(client opsiConfigurationOCIClient) OpsiConfigurationServiceClient {
	hooks := newOpsiConfigurationRuntimeHooksWithOCIClient(client)
	applyOpsiConfigurationRuntimeHooks(&hooks, client, nil)
	config := buildOpsiConfigurationGeneratedRuntimeConfig(&OpsiConfigurationServiceManager{Log: loggerutil.OSOKLogger{}}, hooks)
	return defaultOpsiConfigurationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.OpsiConfiguration](config),
	}
}

func makeOpsiConfigurationResource() *opsiv1beta1.OpsiConfiguration {
	return &opsiv1beta1.OpsiConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opsiconfiguration-sample",
			Namespace: "default",
			UID:       types.UID("uid-opsiconfiguration"),
		},
		Spec: opsiv1beta1.OpsiConfigurationSpec{
			CompartmentId:  "ocid1.compartment.oc1..opsi",
			DisplayName:    "opsi-configuration",
			Description:    "initial description",
			FreeformTags:   map[string]string{"env": "test"},
			DefinedTags:    map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			SystemTags:     map[string]shared.MapValue{"orcl-cloud": {"free-tier-retained": "true"}},
			OpsiConfigType: opsiConfigurationTypeUX,
		},
	}
}

func makeSDKOpsiConfiguration(
	id string,
	resource *opsiv1beta1.OpsiConfiguration,
	state opsisdk.OpsiConfigurationLifecycleStateEnum,
) opsisdk.OpsiUxConfiguration {
	return opsisdk.OpsiUxConfiguration{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      common.String(resource.Spec.DisplayName),
		Description:      common.String(resource.Spec.Description),
		FreeformTags:     cloneOpsiConfigurationStringMap(resource.Spec.FreeformTags),
		DefinedTags:      opsiConfigurationDefinedTags(resource.Spec.DefinedTags),
		SystemTags:       opsiConfigurationDefinedTags(resource.Spec.SystemTags),
		LifecycleState:   state,
		LifecycleDetails: common.String(""),
		ConfigItems: []opsisdk.OpsiConfigurationConfigurationItemSummary{
			opsisdk.OpsiConfigurationBasicConfigurationItemSummary{
				Name:  common.String("ui.theme"),
				Value: common.String("dark"),
			},
		},
	}
}

func makeSDKOpsiConfigurationSummary(
	id string,
	displayName string,
	resource *opsiv1beta1.OpsiConfiguration,
	state opsisdk.OpsiConfigurationLifecycleStateEnum,
) opsisdk.OpsiUxConfigurationSummary {
	return opsisdk.OpsiUxConfigurationSummary{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      common.String(displayName),
		Description:      common.String(resource.Spec.Description),
		FreeformTags:     cloneOpsiConfigurationStringMap(resource.Spec.FreeformTags),
		DefinedTags:      opsiConfigurationDefinedTags(resource.Spec.DefinedTags),
		SystemTags:       opsiConfigurationDefinedTags(resource.Spec.SystemTags),
		LifecycleState:   state,
		LifecycleDetails: common.String(""),
	}
}

func makeOpsiConfigurationWorkRequest(
	id string,
	operation opsisdk.OperationTypeEnum,
	status opsisdk.OperationStatusEnum,
	action opsisdk.ActionTypeEnum,
	resourceID string,
) opsisdk.WorkRequest {
	return opsisdk.WorkRequest{
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..opsi"),
		OperationType:   operation,
		Status:          status,
		PercentComplete: common.Float32(50),
		TimeAccepted:    &common.SDKTime{Time: metav1.Now().Time},
		TimeStarted:     &common.SDKTime{Time: metav1.Now().Time},
		TimeFinished:    &common.SDKTime{Time: metav1.Now().Time},
		Resources:       []opsisdk.WorkRequestResource{makeOpsiConfigurationWorkRequestResource(action, resourceID)},
	}
}

func makeOpsiConfigurationWorkRequestResource(action opsisdk.ActionTypeEnum, resourceID string) opsisdk.WorkRequestResource {
	return opsisdk.WorkRequestResource{
		EntityType: common.String("opsiConfiguration"),
		ActionType: action,
		Identifier: common.String(resourceID),
		EntityUri:  common.String("/opsiconfigurations/" + resourceID),
	}
}

func requireOpsiConfigurationStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireOpsiConfigurationAsyncCurrent(
	t *testing.T,
	resource *opsiv1beta1.OpsiConfiguration,
	source shared.OSOKAsyncSource,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.async.current = nil, want %s %s", source, phase)
	}
	if current.Source != source || current.Phase != phase || current.WorkRequestID != workRequestID || current.NormalizedClass != class {
		t.Fatalf("status.async.current = %#v, want source=%q phase=%q workRequestID=%q class=%q", current, source, phase, workRequestID, class)
	}
}

func assertOpsiConfigurationStatusJSONData(t *testing.T, raw string, wantName string, wantValue string) {
	t.Helper()

	var payload struct {
		OpsiConfigType string           `json:"opsiConfigType"`
		ConfigItems    []map[string]any `json:"configItems"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode status.jsonData %q error = %v", raw, err)
	}
	if payload.OpsiConfigType != opsiConfigurationTypeUX {
		t.Fatalf("status.jsonData.opsiConfigType = %q, want %q", payload.OpsiConfigType, opsiConfigurationTypeUX)
	}
	if len(payload.ConfigItems) != 1 {
		t.Fatalf("status.jsonData.configItems len = %d, want 1", len(payload.ConfigItems))
	}
	item := payload.ConfigItems[0]
	if got := item["configItemType"]; got != "BASIC" {
		t.Fatalf("status.jsonData.configItems[0].configItemType = %#v, want BASIC", got)
	}
	if got := item["name"]; got != wantName {
		t.Fatalf("status.jsonData.configItems[0].name = %#v, want %q", got, wantName)
	}
	if got := item["value"]; got != wantValue {
		t.Fatalf("status.jsonData.configItems[0].value = %#v, want %q", got, wantValue)
	}
}

func requireOpsiConfigurationCreateResponseFields(t *testing.T, request opsisdk.CreateOpsiConfigurationRequest) {
	t.Helper()
	if !containsOpsiConfigurationCreateField(request.OpsiConfigField, opsisdk.CreateOpsiConfigurationOpsiConfigFieldConfigitems) {
		t.Fatalf("create OpsiConfigField = %#v, want configItems", request.OpsiConfigField)
	}
	if !containsOpsiConfigurationCreateConfigItemStatus(request.ConfigItemCustomStatus, opsisdk.CreateOpsiConfigurationConfigItemCustomStatusCustomized) ||
		!containsOpsiConfigurationCreateConfigItemStatus(request.ConfigItemCustomStatus, opsisdk.CreateOpsiConfigurationConfigItemCustomStatusNoncustomized) {
		t.Fatalf("create ConfigItemCustomStatus = %#v, want customized and noncustomized", request.ConfigItemCustomStatus)
	}
	if !containsOpsiConfigurationCreateConfigItemField(request.ConfigItemField, opsisdk.CreateOpsiConfigurationConfigItemFieldName) ||
		!containsOpsiConfigurationCreateConfigItemField(request.ConfigItemField, opsisdk.CreateOpsiConfigurationConfigItemFieldValue) {
		t.Fatalf("create ConfigItemField = %#v, want name and value", request.ConfigItemField)
	}
}

func requireOpsiConfigurationGetResponseFields(t *testing.T, request opsisdk.GetOpsiConfigurationRequest) {
	t.Helper()
	if !containsOpsiConfigurationGetField(request.OpsiConfigField, opsisdk.GetOpsiConfigurationOpsiConfigFieldConfigitems) {
		t.Fatalf("get OpsiConfigField = %#v, want configItems", request.OpsiConfigField)
	}
	if !containsOpsiConfigurationGetConfigItemStatus(request.ConfigItemCustomStatus, opsisdk.GetOpsiConfigurationConfigItemCustomStatusCustomized) ||
		!containsOpsiConfigurationGetConfigItemStatus(request.ConfigItemCustomStatus, opsisdk.GetOpsiConfigurationConfigItemCustomStatusNoncustomized) {
		t.Fatalf("get ConfigItemCustomStatus = %#v, want customized and noncustomized", request.ConfigItemCustomStatus)
	}
	if !containsOpsiConfigurationGetConfigItemField(request.ConfigItemField, opsisdk.GetOpsiConfigurationConfigItemFieldName) ||
		!containsOpsiConfigurationGetConfigItemField(request.ConfigItemField, opsisdk.GetOpsiConfigurationConfigItemFieldValue) {
		t.Fatalf("get ConfigItemField = %#v, want name and value", request.ConfigItemField)
	}
}

func containsOpsiConfigurationCreateField(values []opsisdk.CreateOpsiConfigurationOpsiConfigFieldEnum, want opsisdk.CreateOpsiConfigurationOpsiConfigFieldEnum) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsOpsiConfigurationCreateConfigItemStatus(values []opsisdk.CreateOpsiConfigurationConfigItemCustomStatusEnum, want opsisdk.CreateOpsiConfigurationConfigItemCustomStatusEnum) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsOpsiConfigurationCreateConfigItemField(values []opsisdk.CreateOpsiConfigurationConfigItemFieldEnum, want opsisdk.CreateOpsiConfigurationConfigItemFieldEnum) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsOpsiConfigurationGetField(values []opsisdk.GetOpsiConfigurationOpsiConfigFieldEnum, want opsisdk.GetOpsiConfigurationOpsiConfigFieldEnum) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsOpsiConfigurationGetConfigItemStatus(values []opsisdk.GetOpsiConfigurationConfigItemCustomStatusEnum, want opsisdk.GetOpsiConfigurationConfigItemCustomStatusEnum) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsOpsiConfigurationGetConfigItemField(values []opsisdk.GetOpsiConfigurationConfigItemFieldEnum, want opsisdk.GetOpsiConfigurationConfigItemFieldEnum) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertOpsiConfigurationStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
