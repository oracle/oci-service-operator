/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package fleet

import (
	"context"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	jmssdk "github.com/oracle/oci-go-sdk/v65/jms"
	jmsv1beta1 "github.com/oracle/oci-service-operator/api/jms/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeFleetOCIClient struct {
	createFn      func(context.Context, jmssdk.CreateFleetRequest) (jmssdk.CreateFleetResponse, error)
	getFn         func(context.Context, jmssdk.GetFleetRequest) (jmssdk.GetFleetResponse, error)
	listFn        func(context.Context, jmssdk.ListFleetsRequest) (jmssdk.ListFleetsResponse, error)
	updateFn      func(context.Context, jmssdk.UpdateFleetRequest) (jmssdk.UpdateFleetResponse, error)
	deleteFn      func(context.Context, jmssdk.DeleteFleetRequest) (jmssdk.DeleteFleetResponse, error)
	workRequestFn func(context.Context, jmssdk.GetWorkRequestRequest) (jmssdk.GetWorkRequestResponse, error)
}

func (f *fakeFleetOCIClient) CreateFleet(
	ctx context.Context,
	req jmssdk.CreateFleetRequest,
) (jmssdk.CreateFleetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return jmssdk.CreateFleetResponse{}, nil
}

func (f *fakeFleetOCIClient) GetFleet(
	ctx context.Context,
	req jmssdk.GetFleetRequest,
) (jmssdk.GetFleetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return jmssdk.GetFleetResponse{}, errortest.NewServiceError(404, "NotFound", "missing Fleet")
}

func (f *fakeFleetOCIClient) ListFleets(
	ctx context.Context,
	req jmssdk.ListFleetsRequest,
) (jmssdk.ListFleetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return jmssdk.ListFleetsResponse{}, nil
}

func (f *fakeFleetOCIClient) UpdateFleet(
	ctx context.Context,
	req jmssdk.UpdateFleetRequest,
) (jmssdk.UpdateFleetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return jmssdk.UpdateFleetResponse{}, nil
}

func (f *fakeFleetOCIClient) DeleteFleet(
	ctx context.Context,
	req jmssdk.DeleteFleetRequest,
) (jmssdk.DeleteFleetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return jmssdk.DeleteFleetResponse{}, nil
}

func (f *fakeFleetOCIClient) GetWorkRequest(
	ctx context.Context,
	req jmssdk.GetWorkRequestRequest,
) (jmssdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return jmssdk.GetWorkRequestResponse{}, nil
}

func TestReviewedFleetRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedFleetRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedFleetRuntimeSemantics() = nil")
	}

	if got.FormalService != "jms" {
		t.Fatalf("FormalService = %q, want jms", got.FormalService)
	}
	if got.FormalSlug != "fleet" {
		t.Fatalf("FormalSlug = %q, want fleet", got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest semantics", got.Async)
	}
	assertFleetStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertFleetStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertFleetStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertFleetStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "NEEDS_ATTENTION"})
	assertFleetStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertFleetStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertFleetStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	assertFleetStringSliceEqual(
		t,
		"Mutation.Mutable",
		got.Mutation.Mutable,
		[]string{"definedTags", "description", "displayName", "freeformTags", "inventoryLog", "isAdvancedFeaturesEnabled", "operationLog"},
	)
	assertFleetStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetFleet" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetFleet", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetFleet" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetFleet", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetFleet/ListFleets confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want GetWorkRequest -> GetFleet/ListFleets confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardFleetExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeFleetResource()
	resource.Spec.DisplayName = ""

	decision, err := guardFleetExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardFleetExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardFleetExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "fleet-sample"
	decision, err = guardFleetExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardFleetExistingBeforeCreate(complete identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardFleetExistingBeforeCreate(complete identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildFleetCreateBodyRequiresInventoryLog(t *testing.T) {
	t.Parallel()

	resource := makeFleetResource()
	resource.Spec.InventoryLog = jmsv1beta1.FleetInventoryLog{}

	_, err := buildFleetCreateBody(resource)
	if err == nil {
		t.Fatal("buildFleetCreateBody() error = nil, want missing inventoryLog validation")
	}
	if !strings.Contains(err.Error(), "inventoryLog") {
		t.Fatalf("buildFleetCreateBody() error = %v, want inventoryLog detail", err)
	}
}

func TestBuildFleetUpdateBodyPreservesClearsAndBooleanFalse(t *testing.T) {
	t.Parallel()

	currentResource := makeFleetResource()
	currentResource.Spec.Description = "current fleet description"
	currentResource.Spec.InventoryLog = jmsv1beta1.FleetInventoryLog{
		LogGroupId: "ocid1.loggroup.oc1..currentinventory",
		LogId:      "ocid1.log.oc1..currentinventory",
	}
	currentResource.Spec.OperationLog = jmsv1beta1.FleetOperationLog{
		LogGroupId: "ocid1.loggroup.oc1..currentoperation",
		LogId:      "ocid1.log.oc1..currentoperation",
	}
	currentResource.Spec.IsAdvancedFeaturesEnabled = true

	desired := makeFleetResource()
	desired.Spec.Description = ""
	desired.Spec.InventoryLog = jmsv1beta1.FleetInventoryLog{
		LogGroupId: "ocid1.loggroup.oc1..desiredinventory",
		LogId:      "ocid1.log.oc1..desiredinventory",
	}
	desired.Spec.OperationLog = jmsv1beta1.FleetOperationLog{
		LogGroupId: "ocid1.loggroup.oc1..desiredoperation",
		LogId:      "ocid1.log.oc1..desiredoperation",
	}
	desired.Spec.IsAdvancedFeaturesEnabled = false
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildFleetUpdateBody(
		desired,
		jmssdk.GetFleetResponse{
			Fleet: makeSDKFleet("ocid1.fleet.oc1..existing", currentResource, jmssdk.LifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildFleetUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildFleetUpdateBody() updateNeeded = false, want true")
	}

	requireFleetStringPtr(t, "details.description", body.Description, "")
	requireFleetCustomLog(t, "details.inventoryLog", body.InventoryLog, desired.Spec.InventoryLog.LogGroupId, desired.Spec.InventoryLog.LogId)
	requireFleetCustomLog(t, "details.operationLog", body.OperationLog, desired.Spec.OperationLog.LogGroupId, desired.Spec.OperationLog.LogId)
	requireFleetBoolPtr(t, "details.isAdvancedFeaturesEnabled", body.IsAdvancedFeaturesEnabled, false)
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
}

func TestFleetCreateOrUpdateStartsWorkRequestTracking(t *testing.T) {
	t.Parallel()

	resource := makeFleetResource()
	createCalls := 0

	client := newTestFleetClient(&fakeFleetOCIClient{
		listFn: func(_ context.Context, req jmssdk.ListFleetsRequest) (jmssdk.ListFleetsResponse, error) {
			requireFleetStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireFleetStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil before the resource is tracked", req.Id)
			}
			return jmssdk.ListFleetsResponse{
				FleetCollection: jmssdk.FleetCollection{
					Items: []jmssdk.FleetSummary{},
				},
			}, nil
		},
		createFn: func(_ context.Context, req jmssdk.CreateFleetRequest) (jmssdk.CreateFleetResponse, error) {
			createCalls++
			requireFleetStringPtr(t, "create displayName", req.CreateFleetDetails.DisplayName, resource.Spec.DisplayName)
			requireFleetStringPtr(t, "create compartmentId", req.CreateFleetDetails.CompartmentId, resource.Spec.CompartmentId)
			requireFleetCustomLog(t, "create inventoryLog", req.CreateFleetDetails.InventoryLog, resource.Spec.InventoryLog.LogGroupId, resource.Spec.InventoryLog.LogId)
			return jmssdk.CreateFleetResponse{
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req jmssdk.GetWorkRequestRequest) (jmssdk.GetWorkRequestResponse, error) {
			requireFleetStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-1")
			return jmssdk.GetWorkRequestResponse{
				WorkRequest: makeFleetWorkRequest(
					"wr-create-1",
					jmssdk.OperationTypeCreateFleet,
					jmssdk.OperationStatusAccepted,
					jmssdk.ActionTypeCreated,
					"ocid1.fleet.oc1..created",
				),
			}, nil
		},
		getFn: func(_ context.Context, _ jmssdk.GetFleetRequest) (jmssdk.GetFleetResponse, error) {
			t.Fatal("GetFleet() should not run while the create work request is pending")
			return jmssdk.GetFleetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateFleet() calls = %d, want 1", createCalls)
	}
	requireFleetAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1", shared.OSOKAsyncClassPending)
}

func TestFleetCreateOrUpdateReusesNeedsAttentionFleet(t *testing.T) {
	t.Parallel()

	resource := makeFleetResource()
	existingID := "ocid1.fleet.oc1..existing"
	createCalls := 0

	client := newTestFleetClient(&fakeFleetOCIClient{
		listFn: func(_ context.Context, req jmssdk.ListFleetsRequest) (jmssdk.ListFleetsResponse, error) {
			requireFleetStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireFleetStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return jmssdk.ListFleetsResponse{
				FleetCollection: jmssdk.FleetCollection{
					Items: []jmssdk.FleetSummary{
						makeSDKFleetSummary(existingID, resource, jmssdk.LifecycleStateNeedsAttention),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ jmssdk.CreateFleetRequest) (jmssdk.CreateFleetResponse, error) {
			createCalls++
			return jmssdk.CreateFleetResponse{}, nil
		},
		getFn: func(_ context.Context, req jmssdk.GetFleetRequest) (jmssdk.GetFleetResponse, error) {
			requireFleetStringPtr(t, "get fleetId", req.FleetId, existingID)
			return jmssdk.GetFleetResponse{
				Fleet: makeSDKFleet(existingID, resource, jmssdk.LifecycleStateNeedsAttention),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful settled result", response)
	}
	if createCalls != 0 {
		t.Fatalf("CreateFleet() calls = %d, want 0 when a reusable Fleet exists", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
	if got := resource.Status.LifecycleState; got != string(jmssdk.LifecycleStateNeedsAttention) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, jmssdk.LifecycleStateNeedsAttention)
	}
}

func TestFleetDeleteConfirmsRemovalAfterWorkRequest(t *testing.T) {
	t.Parallel()

	existingID := "ocid1.fleet.oc1..delete"
	resource := newExistingFleetResource(existingID)

	client := newTestFleetClient(&fakeFleetOCIClient{
		deleteFn: func(_ context.Context, req jmssdk.DeleteFleetRequest) (jmssdk.DeleteFleetResponse, error) {
			requireFleetStringPtr(t, "delete fleetId", req.FleetId, existingID)
			return jmssdk.DeleteFleetResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req jmssdk.GetWorkRequestRequest) (jmssdk.GetWorkRequestResponse, error) {
			requireFleetStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-1")
			return jmssdk.GetWorkRequestResponse{
				WorkRequest: makeFleetWorkRequest(
					"wr-delete-1",
					jmssdk.OperationTypeDeleteFleet,
					jmssdk.OperationStatusSucceeded,
					jmssdk.ActionTypeDeleted,
					existingID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req jmssdk.GetFleetRequest) (jmssdk.GetFleetResponse, error) {
			requireFleetStringPtr(t, "get fleetId", req.FleetId, existingID)
			return jmssdk.GetFleetResponse{}, errortest.NewServiceError(404, "NotFound", "missing Fleet")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want deleted after work request success and NotFound confirmation")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after delete", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

func newTestFleetClient(client *fakeFleetOCIClient) FleetServiceClient {
	if client == nil {
		client = &fakeFleetOCIClient{}
	}
	return newFleetServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeFleetResource() *jmsv1beta1.Fleet {
	return &jmsv1beta1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-sample",
			Namespace: "default",
		},
		Spec: jmsv1beta1.FleetSpec{
			DisplayName:   "fleet-sample",
			CompartmentId: "ocid1.compartment.oc1..fleetexample",
			InventoryLog: jmsv1beta1.FleetInventoryLog{
				LogGroupId: "ocid1.loggroup.oc1..inventory",
				LogId:      "ocid1.log.oc1..inventory",
			},
			Description: "managed java fleet",
			FreeformTags: map[string]string{
				"environment": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func newExistingFleetResource(existingID string) *jmsv1beta1.Fleet {
	resource := makeFleetResource()
	resource.Status = jmsv1beta1.FleetStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func makeSDKFleet(
	id string,
	resource *jmsv1beta1.Fleet,
	state jmssdk.LifecycleStateEnum,
) jmssdk.Fleet {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return jmssdk.Fleet{
		Id:                                   common.String(id),
		DisplayName:                          common.String(resource.Spec.DisplayName),
		Description:                          common.String(resource.Spec.Description),
		CompartmentId:                        common.String(resource.Spec.CompartmentId),
		ApproximateJreCount:                  common.Int(1),
		ApproximateInstallationCount:         common.Int(2),
		ApproximateApplicationCount:          common.Int(3),
		ApproximateManagedInstanceCount:      common.Int(4),
		ApproximateJavaServerCount:           common.Int(5),
		ApproximateLibraryCount:              common.Int(6),
		ApproximateLibraryVulnerabilityCount: common.Int(7),
		TimeCreated:                          &now,
		LifecycleState:                       state,
		InventoryLog:                         sdkFleetCustomLog(resource.Spec.InventoryLog.LogGroupId, resource.Spec.InventoryLog.LogId),
		OperationLog:                         sdkFleetCustomLog(resource.Spec.OperationLog.LogGroupId, resource.Spec.OperationLog.LogId),
		IsAdvancedFeaturesEnabled:            common.Bool(resource.Spec.IsAdvancedFeaturesEnabled),
		IsExportSettingEnabled:               common.Bool(true),
		DefinedTags:                          sdkFleetDefinedTags(resource.Spec.DefinedTags),
		FreeformTags:                         maps.Clone(resource.Spec.FreeformTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeSDKFleetSummary(
	id string,
	resource *jmsv1beta1.Fleet,
	state jmssdk.LifecycleStateEnum,
) jmssdk.FleetSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return jmssdk.FleetSummary{
		Id:                                   common.String(id),
		DisplayName:                          common.String(resource.Spec.DisplayName),
		Description:                          common.String(resource.Spec.Description),
		CompartmentId:                        common.String(resource.Spec.CompartmentId),
		ApproximateJreCount:                  common.Int(1),
		ApproximateInstallationCount:         common.Int(2),
		ApproximateApplicationCount:          common.Int(3),
		ApproximateManagedInstanceCount:      common.Int(4),
		ApproximateJavaServerCount:           common.Int(5),
		ApproximateLibraryCount:              common.Int(6),
		ApproximateLibraryVulnerabilityCount: common.Int(7),
		TimeCreated:                          &now,
		LifecycleState:                       state,
		InventoryLog:                         sdkFleetCustomLog(resource.Spec.InventoryLog.LogGroupId, resource.Spec.InventoryLog.LogId),
		OperationLog:                         sdkFleetCustomLog(resource.Spec.OperationLog.LogGroupId, resource.Spec.OperationLog.LogId),
		IsAdvancedFeaturesEnabled:            common.Bool(resource.Spec.IsAdvancedFeaturesEnabled),
		IsExportSettingEnabled:               common.Bool(true),
		DefinedTags:                          sdkFleetDefinedTags(resource.Spec.DefinedTags),
		FreeformTags:                         maps.Clone(resource.Spec.FreeformTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeFleetWorkRequest(
	id string,
	operation jmssdk.OperationTypeEnum,
	status jmssdk.OperationStatusEnum,
	action jmssdk.ActionTypeEnum,
	resourceID string,
) jmssdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return jmssdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..fleetexample"),
		Resources:       []jmssdk.WorkRequestResource{{EntityType: common.String("Fleet"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func sdkFleetCustomLog(logGroupID string, logID string) *jmssdk.CustomLog {
	logGroupID = strings.TrimSpace(logGroupID)
	logID = strings.TrimSpace(logID)
	if logGroupID == "" || logID == "" {
		return nil
	}
	return &jmssdk.CustomLog{
		LogGroupId: common.String(logGroupID),
		LogId:      common.String(logID),
	}
}

func sdkFleetDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		out[namespace] = converted
	}
	return out
}

func assertFleetStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireFleetStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireFleetBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %t", name, got, want)
	}
}

func requireFleetCustomLog(t *testing.T, name string, got *jmssdk.CustomLog, wantLogGroupID string, wantLogID string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want logGroupId=%q logId=%q", name, wantLogGroupID, wantLogID)
	}
	requireFleetStringPtr(t, name+".logGroupId", got.LogGroupId, wantLogGroupID)
	requireFleetStringPtr(t, name+".logId", got.LogId, wantLogID)
}

func requireFleetAsyncCurrent(
	t *testing.T,
	resource *jmsv1beta1.Fleet,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}
