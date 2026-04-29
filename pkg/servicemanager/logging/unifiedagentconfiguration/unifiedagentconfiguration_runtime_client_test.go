/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package unifiedagentconfiguration

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeUnifiedAgentConfigurationOCIClient struct {
	createFn      func(context.Context, loggingsdk.CreateUnifiedAgentConfigurationRequest) (loggingsdk.CreateUnifiedAgentConfigurationResponse, error)
	getFn         func(context.Context, loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error)
	listFn        func(context.Context, loggingsdk.ListUnifiedAgentConfigurationsRequest) (loggingsdk.ListUnifiedAgentConfigurationsResponse, error)
	updateFn      func(context.Context, loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error)
	deleteFn      func(context.Context, loggingsdk.DeleteUnifiedAgentConfigurationRequest) (loggingsdk.DeleteUnifiedAgentConfigurationResponse, error)
	workRequestFn func(context.Context, loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error)
}

func (f *fakeUnifiedAgentConfigurationOCIClient) CreateUnifiedAgentConfiguration(ctx context.Context, req loggingsdk.CreateUnifiedAgentConfigurationRequest) (loggingsdk.CreateUnifiedAgentConfigurationResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return loggingsdk.CreateUnifiedAgentConfigurationResponse{}, nil
}

func (f *fakeUnifiedAgentConfigurationOCIClient) GetUnifiedAgentConfiguration(ctx context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return loggingsdk.GetUnifiedAgentConfigurationResponse{}, nil
}

func (f *fakeUnifiedAgentConfigurationOCIClient) ListUnifiedAgentConfigurations(ctx context.Context, req loggingsdk.ListUnifiedAgentConfigurationsRequest) (loggingsdk.ListUnifiedAgentConfigurationsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return loggingsdk.ListUnifiedAgentConfigurationsResponse{}, nil
}

func (f *fakeUnifiedAgentConfigurationOCIClient) UpdateUnifiedAgentConfiguration(ctx context.Context, req loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return loggingsdk.UpdateUnifiedAgentConfigurationResponse{}, nil
}

func (f *fakeUnifiedAgentConfigurationOCIClient) DeleteUnifiedAgentConfiguration(ctx context.Context, req loggingsdk.DeleteUnifiedAgentConfigurationRequest) (loggingsdk.DeleteUnifiedAgentConfigurationResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return loggingsdk.DeleteUnifiedAgentConfigurationResponse{}, nil
}

func (f *fakeUnifiedAgentConfigurationOCIClient) GetWorkRequest(ctx context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return loggingsdk.GetWorkRequestResponse{}, nil
}

func TestUnifiedAgentConfigurationRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := newUnifiedAgentConfigurationRuntimeSemantics()
	if got == nil {
		t.Fatal("newUnifiedAgentConfigurationRuntimeSemantics() = nil")
	}

	if got.FormalService != "logging" {
		t.Fatalf("FormalService = %q, want logging", got.FormalService)
	}
	if got.FormalSlug != "unifiedagentconfiguration" {
		t.Fatalf("FormalSlug = %q, want unifiedagentconfiguration", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertUnifiedAgentConfigurationStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName"})
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "serviceConfiguration"})
	assertUnifiedAgentConfigurationStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
}

func TestUnifiedAgentConfigurationServiceClientCreatesAndTracksWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeUnifiedAgentConfigurationResource()
	var createRequest loggingsdk.CreateUnifiedAgentConfigurationRequest
	getCalled := false

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		listFn: func(_ context.Context, req loggingsdk.ListUnifiedAgentConfigurationsRequest) (loggingsdk.ListUnifiedAgentConfigurationsResponse, error) {
			requireStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return loggingsdk.ListUnifiedAgentConfigurationsResponse{}, nil
		},
		createFn: func(_ context.Context, req loggingsdk.CreateUnifiedAgentConfigurationRequest) (loggingsdk.CreateUnifiedAgentConfigurationResponse, error) {
			createRequest = req
			return loggingsdk.CreateUnifiedAgentConfigurationResponse{
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-1")
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeUnifiedAgentConfigurationWorkRequest(
					"wr-create-1",
					loggingsdk.OperationTypesCreateConfiguration,
					loggingsdk.OperationStatusInProgress,
					loggingsdk.ActionTypesInProgress,
					"ocid1.unifiedagentconfiguration.oc1..created",
				),
			}, nil
		},
		getFn: func(context.Context, loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			getCalled = true
			return loggingsdk.GetUnifiedAgentConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while create work request is pending")
	}
	if getCalled {
		t.Fatal("GetUnifiedAgentConfiguration() should not run while create work request is pending")
	}
	requireStringPtr(t, "create compartmentId", createRequest.CreateUnifiedAgentConfigurationDetails.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "create displayName", createRequest.CreateUnifiedAgentConfigurationDetails.DisplayName, resource.Spec.DisplayName)
	if createRequest.CreateUnifiedAgentConfigurationDetails.IsEnabled == nil || !*createRequest.CreateUnifiedAgentConfigurationDetails.IsEnabled {
		t.Fatalf("create isEnabled = %v, want true", createRequest.CreateUnifiedAgentConfigurationDetails.IsEnabled)
	}
	if _, ok := createRequest.CreateUnifiedAgentConfigurationDetails.ServiceConfiguration.(loggingsdk.UnifiedAgentLoggingConfiguration); !ok {
		t.Fatalf("create serviceConfiguration = %T, want logging.UnifiedAgentLoggingConfiguration", createRequest.CreateUnifiedAgentConfigurationDetails.ServiceConfiguration)
	}
	payload, err := json.Marshal(createRequest.CreateUnifiedAgentConfigurationDetails)
	if err != nil {
		t.Fatalf("marshal create details: %v", err)
	}
	if strings.Contains(string(payload), "jsonData") {
		t.Fatalf("create details leaked jsonData helper field: %s", payload)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, "wr-create-1", shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestUnifiedAgentConfigurationServiceClientResumesCreateWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.unifiedagentconfiguration.oc1..created"
	resource := makeUnifiedAgentConfigurationResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create-success",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-success")
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeUnifiedAgentConfigurationWorkRequest(
					"wr-create-success",
					loggingsdk.OperationTypesCreateConfiguration,
					loggingsdk.OperationStatusSucceeded,
					loggingsdk.ActionTypesCreated,
					createdID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, createdID)
			return loggingsdk.GetUnifiedAgentConfigurationResponse{
				UnifiedAgentConfiguration: makeSDKUnifiedAgentConfiguration(createdID, resource, loggingsdk.LogLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want converged success", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(loggingsdk.LogLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
	requireTrailingCondition(t, resource, shared.Active)
}

func TestUnifiedAgentConfigurationServiceClientBindsExistingWithoutCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..existing"
	resource := makeUnifiedAgentConfigurationResource()
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		listFn: func(_ context.Context, req loggingsdk.ListUnifiedAgentConfigurationsRequest) (loggingsdk.ListUnifiedAgentConfigurationsResponse, error) {
			requireStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return loggingsdk.ListUnifiedAgentConfigurationsResponse{
				UnifiedAgentConfigurationCollection: loggingsdk.UnifiedAgentConfigurationCollection{
					Items: []loggingsdk.UnifiedAgentConfigurationSummary{
						makeSDKUnifiedAgentConfigurationSummary(existingID, resource, loggingsdk.LogLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			getCalls++
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			return loggingsdk.GetUnifiedAgentConfigurationResponse{
				UnifiedAgentConfiguration: makeSDKUnifiedAgentConfiguration(existingID, resource, loggingsdk.LogLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, loggingsdk.CreateUnifiedAgentConfigurationRequest) (loggingsdk.CreateUnifiedAgentConfigurationResponse, error) {
			createCalled = true
			return loggingsdk.CreateUnifiedAgentConfigurationResponse{}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateUnifiedAgentConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want converged success", response)
	}
	if createCalled {
		t.Fatal("CreateUnifiedAgentConfiguration() should not be called when list finds a reusable match")
	}
	if updateCalled {
		t.Fatal("UpdateUnifiedAgentConfiguration() should not be called when mutable state already matches")
	}
	if getCalls != 1 {
		t.Fatalf("GetUnifiedAgentConfiguration() calls = %d, want 1 live assessment read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
	requireTrailingCondition(t, resource, shared.Active)
}

func TestUnifiedAgentConfigurationServiceClientCreatesWhenDisplayNameEmptyInsteadOfBindingByCompartment(t *testing.T) {
	t.Parallel()

	resource := makeUnifiedAgentConfigurationResource()
	resource.Spec.DisplayName = ""
	listCalled := false
	createCalled := false

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		listFn: func(context.Context, loggingsdk.ListUnifiedAgentConfigurationsRequest) (loggingsdk.ListUnifiedAgentConfigurationsResponse, error) {
			listCalled = true
			return loggingsdk.ListUnifiedAgentConfigurationsResponse{
				UnifiedAgentConfigurationCollection: loggingsdk.UnifiedAgentConfigurationCollection{
					Items: []loggingsdk.UnifiedAgentConfigurationSummary{
						{
							Id:             common.String("ocid1.unifiedagentconfiguration.oc1..unrelated"),
							CompartmentId:  common.String(resource.Spec.CompartmentId),
							DisplayName:    common.String("unrelated-uac"),
							LifecycleState: loggingsdk.LogLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getFn: func(context.Context, loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			t.Fatal("GetUnifiedAgentConfiguration() should not run when empty displayName skips pre-create reuse")
			return loggingsdk.GetUnifiedAgentConfigurationResponse{}, nil
		},
		createFn: func(_ context.Context, req loggingsdk.CreateUnifiedAgentConfigurationRequest) (loggingsdk.CreateUnifiedAgentConfigurationResponse, error) {
			createCalled = true
			if req.CreateUnifiedAgentConfigurationDetails.DisplayName != nil {
				t.Fatalf("create displayName = %q, want nil for omitted spec.displayName", *req.CreateUnifiedAgentConfigurationDetails.DisplayName)
			}
			return loggingsdk.CreateUnifiedAgentConfigurationResponse{
				OpcWorkRequestId: common.String("wr-create-no-display-name"),
				OpcRequestId:     common.String("opc-create-no-display-name"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-no-display-name")
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeUnifiedAgentConfigurationWorkRequest(
					"wr-create-no-display-name",
					loggingsdk.OperationTypesCreateConfiguration,
					loggingsdk.OperationStatusInProgress,
					loggingsdk.ActionTypesInProgress,
					"ocid1.unifiedagentconfiguration.oc1..created-without-display-name",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want pending create success", response)
	}
	if listCalled {
		t.Fatal("ListUnifiedAgentConfigurations() should not be called when spec.displayName is empty")
	}
	if !createCalled {
		t.Fatal("CreateUnifiedAgentConfiguration() was not called")
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, "wr-create-no-display-name", shared.OSOKAsyncClassPending)
}

func TestUnifiedAgentConfigurationServiceClientUpdatesSupportedDisplayNameDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..update"
	resource := makeUnifiedAgentConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = "old-uac"
	resource.Spec.DisplayName = "new-uac"

	var updateRequest loggingsdk.UpdateUnifiedAgentConfigurationRequest
	getCalls := 0

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			getCalls++
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			current := makeSDKUnifiedAgentConfiguration(existingID, resource, loggingsdk.LogLifecycleStateActive)
			if getCalls == 1 {
				current.DisplayName = common.String("old-uac")
			}
			return loggingsdk.GetUnifiedAgentConfigurationResponse{UnifiedAgentConfiguration: current}, nil
		},
		updateFn: func(_ context.Context, req loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
			updateRequest = req
			return loggingsdk.UpdateUnifiedAgentConfigurationResponse{
				OpcWorkRequestId: common.String("wr-update-1"),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-update-1")
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeUnifiedAgentConfigurationWorkRequest(
					"wr-update-1",
					loggingsdk.OperationTypesUpdateConfiguration,
					loggingsdk.OperationStatusSucceeded,
					loggingsdk.ActionTypesUpdated,
					existingID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want converged success", response)
	}
	if getCalls != 2 {
		t.Fatalf("GetUnifiedAgentConfiguration() calls = %d, want live read and post-work-request read", getCalls)
	}
	requireStringPtr(t, "update unifiedAgentConfigurationId", updateRequest.UnifiedAgentConfigurationId, existingID)
	requireStringPtr(t, "update displayName", updateRequest.UpdateUnifiedAgentConfigurationDetails.DisplayName, resource.Spec.DisplayName)
	if updateRequest.UpdateUnifiedAgentConfigurationDetails.IsEnabled == nil || !*updateRequest.UpdateUnifiedAgentConfigurationDetails.IsEnabled {
		t.Fatalf("update isEnabled = %v, want current true value", updateRequest.UpdateUnifiedAgentConfigurationDetails.IsEnabled)
	}
	if updateRequest.UpdateUnifiedAgentConfigurationDetails.ServiceConfiguration == nil {
		t.Fatal("update serviceConfiguration = nil, want current value preserved for SDK mandatory field")
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
	requireTrailingCondition(t, resource, shared.Active)
}

func TestUnifiedAgentConfigurationCreateDetailsPreservesExplicitJSONDataFalseAndZeroValues(t *testing.T) {
	t.Parallel()

	resource := makeJSONDataUnifiedAgentConfigurationResource()

	details, err := buildUnifiedAgentConfigurationCreateDetails(resource)
	if err != nil {
		t.Fatalf("buildUnifiedAgentConfigurationCreateDetails() error = %v", err)
	}
	configuration, ok := details.ServiceConfiguration.(loggingsdk.UnifiedAgentLoggingConfiguration)
	if !ok {
		t.Fatalf("serviceConfiguration = %T, want logging.UnifiedAgentLoggingConfiguration", details.ServiceConfiguration)
	}
	source, ok := configuration.Sources[0].(loggingsdk.UnifiedAgentTailLogSource)
	if !ok {
		t.Fatalf("source = %T, want logging.UnifiedAgentTailLogSource", configuration.Sources[0])
	}
	parser, ok := source.Parser.(loggingsdk.UnifiedAgentCriParser)
	if !ok {
		t.Fatalf("parser = %T, want logging.UnifiedAgentCriParser", source.Parser)
	}
	if parser.IsMergeCriFields == nil || *parser.IsMergeCriFields {
		t.Fatalf("parser.isMergeCriFields = %v, want explicit false", parser.IsMergeCriFields)
	}
	if parser.TimeoutInMilliseconds == nil || *parser.TimeoutInMilliseconds != 0 {
		t.Fatalf("parser.timeoutInMilliseconds = %v, want explicit 0", parser.TimeoutInMilliseconds)
	}
	payload, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal create details: %v", err)
	}
	if !strings.Contains(string(payload), `"isMergeCriFields":false`) {
		t.Fatalf("create details omitted explicit false jsonData field: %s", payload)
	}
	if !strings.Contains(string(payload), `"timeoutInMilliseconds":0`) {
		t.Fatalf("create details omitted explicit zero jsonData field: %s", payload)
	}
	if strings.Contains(string(payload), "jsonData") {
		t.Fatalf("create details leaked jsonData helper field: %s", payload)
	}
}

func TestUnifiedAgentConfigurationServiceClientAcceptsMatchingJSONDataReadbackWithoutDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..jsondata"
	resource := makeJSONDataUnifiedAgentConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	updateCalled := false

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			current := makeSDKUnifiedAgentConfiguration(existingID, makeUnifiedAgentConfigurationResource(), loggingsdk.LogLifecycleStateActive)
			details, err := buildUnifiedAgentConfigurationCreateDetails(resource)
			if err != nil {
				t.Fatalf("build create details for matching readback: %v", err)
			}
			current.ServiceConfiguration = details.ServiceConfiguration
			return loggingsdk.GetUnifiedAgentConfigurationResponse{UnifiedAgentConfiguration: current}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateUnifiedAgentConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want matching jsonData readback accepted", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want converged success", response)
	}
	if updateCalled {
		t.Fatal("UpdateUnifiedAgentConfiguration() should not be called for matching jsonData readback")
	}
	requireTrailingCondition(t, resource, shared.Active)
}

func TestUnifiedAgentConfigurationServiceClientRejectsJSONDataMismatchBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..jsondata-mismatch"
	resource := makeJSONDataUnifiedAgentConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	updateCalled := false

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			current := makeSDKUnifiedAgentConfiguration(existingID, makeUnifiedAgentConfigurationResource(), loggingsdk.LogLifecycleStateActive)
			details, err := buildUnifiedAgentConfigurationCreateDetails(resource)
			if err != nil {
				t.Fatalf("build create details for mismatched readback: %v", err)
			}
			configuration, ok := details.ServiceConfiguration.(loggingsdk.UnifiedAgentLoggingConfiguration)
			if !ok {
				t.Fatalf("serviceConfiguration = %T, want logging.UnifiedAgentLoggingConfiguration", details.ServiceConfiguration)
			}
			configuration.Destination.LogObjectId = common.String("ocid1.log.oc1..different")
			current.ServiceConfiguration = configuration
			return loggingsdk.GetUnifiedAgentConfigurationResponse{UnifiedAgentConfiguration: current}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateUnifiedAgentConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want jsonData serviceConfiguration drift rejection")
	}
	if !strings.Contains(err.Error(), "reject unsupported update drift for serviceConfiguration") {
		t.Fatalf("CreateOrUpdate() error = %v, want serviceConfiguration drift detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failure", response)
	}
	if updateCalled {
		t.Fatal("UpdateUnifiedAgentConfiguration() should not be called after jsonData serviceConfiguration drift rejection")
	}
}

func TestUnifiedAgentConfigurationCreateDetailsStripsEmptyTypedHelperFields(t *testing.T) {
	t.Parallel()

	resource := makeUnifiedAgentConfigurationResource()
	resource.Spec.ServiceConfiguration.Sources[0].Parser = loggingv1beta1.UnifiedAgentConfigurationServiceConfigurationSourceParser{
		ParserType: string(loggingsdk.UnifiedAgentParserParserTypeCri),
	}

	details, err := buildUnifiedAgentConfigurationCreateDetails(resource)
	if err != nil {
		t.Fatalf("buildUnifiedAgentConfigurationCreateDetails() error = %v", err)
	}
	configuration, ok := details.ServiceConfiguration.(loggingsdk.UnifiedAgentLoggingConfiguration)
	if !ok {
		t.Fatalf("serviceConfiguration = %T, want logging.UnifiedAgentLoggingConfiguration", details.ServiceConfiguration)
	}
	source, ok := configuration.Sources[0].(loggingsdk.UnifiedAgentTailLogSource)
	if !ok {
		t.Fatalf("source = %T, want logging.UnifiedAgentTailLogSource", configuration.Sources[0])
	}
	parser, ok := source.Parser.(loggingsdk.UnifiedAgentCriParser)
	if !ok {
		t.Fatalf("parser = %T, want logging.UnifiedAgentCriParser", source.Parser)
	}
	if parser.IsMergeCriFields != nil {
		t.Fatalf("parser.isMergeCriFields = %v, want nil for empty typed helper field", parser.IsMergeCriFields)
	}
	if parser.TimeoutInMilliseconds != nil {
		t.Fatalf("parser.timeoutInMilliseconds = %v, want nil for empty typed helper field", parser.TimeoutInMilliseconds)
	}
}

func TestUnifiedAgentConfigurationServiceClientRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..replace"
	resource := makeUnifiedAgentConfigurationResource()
	originalCompartment := resource.Spec.CompartmentId
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.CompartmentId = originalCompartment
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	updateCalled := false

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			current := makeSDKUnifiedAgentConfiguration(existingID, resource, loggingsdk.LogLifecycleStateActive)
			current.CompartmentId = common.String(originalCompartment)
			return loggingsdk.GetUnifiedAgentConfigurationResponse{UnifiedAgentConfiguration: current}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateUnifiedAgentConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want compartment replacement rejection")
	}
	if !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartment replacement detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failure", response)
	}
	if updateCalled {
		t.Fatal("UpdateUnifiedAgentConfiguration() should not be called after create-only drift rejection")
	}
}

func TestUnifiedAgentConfigurationServiceClientRejectsServiceConfigurationDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..serviceconfig"
	currentResource := makeUnifiedAgentConfigurationResource()
	resource := makeUnifiedAgentConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Spec.ServiceConfiguration.Destination.LogObjectId = "ocid1.log.oc1..different"
	updateCalled := false

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			return loggingsdk.GetUnifiedAgentConfigurationResponse{
				UnifiedAgentConfiguration: makeSDKUnifiedAgentConfiguration(existingID, currentResource, loggingsdk.LogLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateUnifiedAgentConfigurationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want serviceConfiguration drift rejection")
	}
	if !strings.Contains(err.Error(), "reject unsupported update drift for serviceConfiguration") {
		t.Fatalf("CreateOrUpdate() error = %v, want serviceConfiguration drift detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failure", response)
	}
	if updateCalled {
		t.Fatal("UpdateUnifiedAgentConfiguration() should not be called after serviceConfiguration drift rejection")
	}
}

func TestUnifiedAgentConfigurationServiceClientClassifiesLifecycleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		state       loggingsdk.LogLifecycleStateEnum
		wantReason  shared.OSOKConditionType
		wantRequeue bool
		wantSuccess bool
	}{
		{name: "creating", state: loggingsdk.LogLifecycleStateCreating, wantReason: shared.Provisioning, wantRequeue: true, wantSuccess: true},
		{name: "updating", state: loggingsdk.LogLifecycleStateUpdating, wantReason: shared.Updating, wantRequeue: true, wantSuccess: true},
		{name: "deleting", state: loggingsdk.LogLifecycleStateDeleting, wantReason: shared.Terminating, wantRequeue: true, wantSuccess: true},
		{name: "active", state: loggingsdk.LogLifecycleStateActive, wantReason: shared.Active, wantRequeue: false, wantSuccess: true},
		{name: "failed", state: loggingsdk.LogLifecycleStateFailed, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false},
		{name: "inactive", state: loggingsdk.LogLifecycleStateInactive, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.unifiedagentconfiguration.oc1..lifecycle"
			resource := makeUnifiedAgentConfigurationResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
			resource.Status.Id = existingID

			client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
				getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
					requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
					return loggingsdk.GetUnifiedAgentConfigurationResponse{
						UnifiedAgentConfiguration: makeSDKUnifiedAgentConfiguration(existingID, resource, tc.state),
					}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if tc.wantSuccess && err != nil {
				t.Fatalf("CreateOrUpdate() error = %v, want success", err)
			}
			if response.IsSuccessful != tc.wantSuccess {
				t.Fatalf("CreateOrUpdate() success = %t, want %t (err=%v)", response.IsSuccessful, tc.wantSuccess, err)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() requeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if got := resource.Status.OsokStatus.Reason; got != string(tc.wantReason) {
				t.Fatalf("status.reason = %q, want %s", got, tc.wantReason)
			}
			requireTrailingCondition(t, resource, tc.wantReason)
		})
	}
}

func TestUnifiedAgentConfigurationDeleteTracksWorkRequestUntilPendingCompletes(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..delete"
	resource := makeUnifiedAgentConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	deleteCalls := 0

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			return loggingsdk.GetUnifiedAgentConfigurationResponse{
				UnifiedAgentConfiguration: makeSDKUnifiedAgentConfiguration(existingID, resource, loggingsdk.LogLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req loggingsdk.DeleteUnifiedAgentConfigurationRequest) (loggingsdk.DeleteUnifiedAgentConfigurationResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			return loggingsdk.DeleteUnifiedAgentConfigurationResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-1")
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeUnifiedAgentConfigurationWorkRequest(
					"wr-delete-1",
					loggingsdk.OperationTypesDeleteConfiguration,
					loggingsdk.OperationStatusInProgress,
					loggingsdk.ActionTypesInProgress,
					existingID,
				),
			}, nil
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
		t.Fatalf("DeleteUnifiedAgentConfiguration() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseDelete, "wr-delete-1", shared.OSOKAsyncClassPending)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil while delete work request is pending")
	}
}

func TestUnifiedAgentConfigurationDeleteConfirmsWorkRequestReadNotFound(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.unifiedagentconfiguration.oc1..delete-gone"
	resource := makeUnifiedAgentConfigurationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete-success",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := newTestUnifiedAgentConfigurationClient(&fakeUnifiedAgentConfigurationOCIClient{
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-success")
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeUnifiedAgentConfigurationWorkRequest(
					"wr-delete-success",
					loggingsdk.OperationTypesDeleteConfiguration,
					loggingsdk.OperationStatusSucceeded,
					loggingsdk.ActionTypesDeleted,
					existingID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
			requireStringPtr(t, "get unifiedAgentConfigurationId", req.UnifiedAgentConfigurationId, existingID)
			return loggingsdk.GetUnifiedAgentConfigurationResponse{}, errortest.NewServiceError(404, "NotFound", "UnifiedAgentConfiguration deleted")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after work request succeeds and read reports NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func newTestUnifiedAgentConfigurationClient(client unifiedAgentConfigurationOCIClient) defaultUnifiedAgentConfigurationServiceClient {
	if client == nil {
		client = &fakeUnifiedAgentConfigurationOCIClient{}
	}
	hooks := newUnifiedAgentConfigurationRuntimeHooksWithOCIClient(client)
	applyUnifiedAgentConfigurationRuntimeHooks(
		&UnifiedAgentConfigurationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}},
		&hooks,
		client,
		nil,
	)
	return defaultUnifiedAgentConfigurationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loggingv1beta1.UnifiedAgentConfiguration](
			buildUnifiedAgentConfigurationGeneratedRuntimeConfig(
				&UnifiedAgentConfigurationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}},
				hooks,
			),
		),
	}
}

func newUnifiedAgentConfigurationRuntimeHooksWithOCIClient(client unifiedAgentConfigurationOCIClient) UnifiedAgentConfigurationRuntimeHooks {
	return UnifiedAgentConfigurationRuntimeHooks{
		Semantics: newUnifiedAgentConfigurationRuntimeSemantics(),
		Create: runtimeOperationHooks[loggingsdk.CreateUnifiedAgentConfigurationRequest, loggingsdk.CreateUnifiedAgentConfigurationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateUnifiedAgentConfigurationDetails", RequestName: "CreateUnifiedAgentConfigurationDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request loggingsdk.CreateUnifiedAgentConfigurationRequest) (loggingsdk.CreateUnifiedAgentConfigurationResponse, error) {
				return client.CreateUnifiedAgentConfiguration(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loggingsdk.GetUnifiedAgentConfigurationRequest, loggingsdk.GetUnifiedAgentConfigurationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "UnifiedAgentConfigurationId", RequestName: "unifiedAgentConfigurationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error) {
				return client.GetUnifiedAgentConfiguration(ctx, request)
			},
		},
		List: runtimeOperationHooks[loggingsdk.ListUnifiedAgentConfigurationsRequest, loggingsdk.ListUnifiedAgentConfigurationsResponse]{
			Fields: unifiedAgentConfigurationListFields(),
			Call: func(ctx context.Context, request loggingsdk.ListUnifiedAgentConfigurationsRequest) (loggingsdk.ListUnifiedAgentConfigurationsResponse, error) {
				return client.ListUnifiedAgentConfigurations(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loggingsdk.UpdateUnifiedAgentConfigurationRequest, loggingsdk.UpdateUnifiedAgentConfigurationResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "UnifiedAgentConfigurationId", RequestName: "unifiedAgentConfigurationId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateUnifiedAgentConfigurationDetails", RequestName: "UpdateUnifiedAgentConfigurationDetails", Contribution: "body", PreferResourceID: false},
			},
			Call: func(ctx context.Context, request loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error) {
				return client.UpdateUnifiedAgentConfiguration(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loggingsdk.DeleteUnifiedAgentConfigurationRequest, loggingsdk.DeleteUnifiedAgentConfigurationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "UnifiedAgentConfigurationId", RequestName: "unifiedAgentConfigurationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request loggingsdk.DeleteUnifiedAgentConfigurationRequest) (loggingsdk.DeleteUnifiedAgentConfigurationResponse, error) {
				return client.DeleteUnifiedAgentConfiguration(ctx, request)
			},
		},
		WrapGeneratedClient: []func(UnifiedAgentConfigurationServiceClient) UnifiedAgentConfigurationServiceClient{},
	}
}

func makeUnifiedAgentConfigurationResource() *loggingv1beta1.UnifiedAgentConfiguration {
	return &loggingv1beta1.UnifiedAgentConfiguration{
		Spec: loggingv1beta1.UnifiedAgentConfigurationSpec{
			IsEnabled:     true,
			CompartmentId: "ocid1.compartment.oc1..uac",
			DisplayName:   "osok-uac",
			Description:   "example unified agent configuration",
			FreeformTags:  map[string]string{"managed-by": "osok"},
			ServiceConfiguration: loggingv1beta1.UnifiedAgentConfigurationServiceConfiguration{
				ConfigurationType: string(loggingsdk.UnifiedAgentServiceConfigurationTypesLogging),
				Sources: []loggingv1beta1.UnifiedAgentConfigurationServiceConfigurationSource{
					{
						Name:       "application",
						SourceType: string(loggingsdk.UnifiedAgentLoggingSourceSourceTypeLogTail),
						Paths:      []string{"/var/log/app.log"},
						Parser: loggingv1beta1.UnifiedAgentConfigurationServiceConfigurationSourceParser{
							ParserType: string(loggingsdk.UnifiedAgentParserParserTypeNone),
						},
					},
				},
				Destination: loggingv1beta1.UnifiedAgentConfigurationServiceConfigurationDestination{
					LogObjectId: "ocid1.log.oc1..destination",
				},
			},
			GroupAssociation: loggingv1beta1.UnifiedAgentConfigurationGroupAssociation{
				GroupList: []string{"ocid1.dynamicgroup.oc1..agent"},
			},
		},
	}
}

func makeJSONDataUnifiedAgentConfigurationResource() *loggingv1beta1.UnifiedAgentConfiguration {
	resource := makeUnifiedAgentConfigurationResource()
	resource.Spec.ServiceConfiguration = loggingv1beta1.UnifiedAgentConfigurationServiceConfiguration{
		JsonData: `{"configurationType":"LOGGING","sources":[{"sourceType":"LOG_TAIL","name":"container","paths":["/var/log/containers/*.log"],"parser":{"parserType":"CRI","isMergeCriFields":false,"timeoutInMilliseconds":0}}],"destination":{"logObjectId":"ocid1.log.oc1..destination"}}`,
	}
	return resource
}

func makeSDKUnifiedAgentConfiguration(
	id string,
	resource *loggingv1beta1.UnifiedAgentConfiguration,
	state loggingsdk.LogLifecycleStateEnum,
) loggingsdk.UnifiedAgentConfiguration {
	return loggingsdk.UnifiedAgentConfiguration{
		Id:                   common.String(id),
		CompartmentId:        common.String(resource.Spec.CompartmentId),
		DisplayName:          common.String(resource.Spec.DisplayName),
		LifecycleState:       state,
		IsEnabled:            common.Bool(resource.Spec.IsEnabled),
		ConfigurationState:   loggingsdk.UnifiedAgentServiceConfigurationStatesValid,
		ServiceConfiguration: makeSDKUnifiedAgentServiceConfiguration(resource),
		GroupAssociation:     &loggingsdk.GroupAssociationDetails{GroupList: append([]string(nil), resource.Spec.GroupAssociation.GroupList...)},
		Description:          common.String(resource.Spec.Description),
		FreeformTags:         cloneUnifiedAgentConfigurationStringMap(resource.Spec.FreeformTags),
	}
}

func makeSDKUnifiedAgentConfigurationSummary(
	id string,
	resource *loggingv1beta1.UnifiedAgentConfiguration,
	state loggingsdk.LogLifecycleStateEnum,
) loggingsdk.UnifiedAgentConfigurationSummary {
	return loggingsdk.UnifiedAgentConfigurationSummary{
		Id:                 common.String(id),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		DisplayName:        common.String(resource.Spec.DisplayName),
		LifecycleState:     state,
		IsEnabled:          common.Bool(resource.Spec.IsEnabled),
		ConfigurationType:  loggingsdk.UnifiedAgentServiceConfigurationTypesLogging,
		ConfigurationState: loggingsdk.UnifiedAgentServiceConfigurationStatesValid,
		Description:        common.String(resource.Spec.Description),
		FreeformTags:       cloneUnifiedAgentConfigurationStringMap(resource.Spec.FreeformTags),
	}
}

func makeSDKUnifiedAgentServiceConfiguration(resource *loggingv1beta1.UnifiedAgentConfiguration) loggingsdk.UnifiedAgentLoggingConfiguration {
	source := resource.Spec.ServiceConfiguration.Sources[0]
	return loggingsdk.UnifiedAgentLoggingConfiguration{
		Sources: []loggingsdk.UnifiedAgentLoggingSource{
			loggingsdk.UnifiedAgentTailLogSource{
				Name:   common.String(source.Name),
				Paths:  append([]string(nil), source.Paths...),
				Parser: loggingsdk.UnifiedAgentNoneParser{},
			},
		},
		Destination: &loggingsdk.UnifiedAgentLoggingDestination{
			LogObjectId: common.String(resource.Spec.ServiceConfiguration.Destination.LogObjectId),
		},
	}
}

func makeUnifiedAgentConfigurationWorkRequest(
	id string,
	operation loggingsdk.OperationTypesEnum,
	status loggingsdk.OperationStatusEnum,
	action loggingsdk.ActionTypesEnum,
	resourceID string,
) loggingsdk.WorkRequest {
	percent := float32(50)
	if status == loggingsdk.OperationStatusSucceeded {
		percent = 100
	}
	return loggingsdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operation,
		Status:          status,
		CompartmentId:   common.String("ocid1.compartment.oc1..uac"),
		PercentComplete: common.Float32(percent),
		Resources: []loggingsdk.WorkRequestResource{
			{
				EntityType: common.String("configuration"),
				ActionType: action,
				Identifier: common.String(resourceID),
			},
		},
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireCurrentAsync(
	t *testing.T,
	resource *loggingv1beta1.UnifiedAgentConfiguration,
	wantSource shared.OSOKAsyncSource,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want async operation")
	}
	if current.Source != wantSource {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, wantSource)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
}

func requireTrailingCondition(
	t *testing.T,
	resource *loggingv1beta1.UnifiedAgentConfiguration,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 || conditions[len(conditions)-1].Type != want {
		t.Fatalf("status.conditions = %#v, want trailing %s condition", conditions, want)
	}
}

func assertUnifiedAgentConfigurationStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}
