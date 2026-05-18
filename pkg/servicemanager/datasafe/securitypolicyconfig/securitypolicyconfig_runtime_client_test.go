/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitypolicyconfig

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSecurityPolicyConfigID      = "ocid1.securitypolicyconfig.oc1..existing"
	testSecurityPolicyConfigUpdated = "ocid1.securitypolicyconfig.oc1..updated"
	testSecurityPolicyID            = "ocid1.securitypolicy.oc1..policy"
	testOtherSecurityPolicyID       = "ocid1.securitypolicy.oc1..other"
	testSecurityPolicyCompartmentID = "ocid1.compartment.oc1..datasafe"
	testMovedCompartmentID          = "ocid1.compartment.oc1..moved"
)

type fakeSecurityPolicyConfigOCIClient struct {
	resources map[string]datasafesdk.SecurityPolicyConfig

	createRequests      []datasafesdk.CreateSecurityPolicyConfigRequest
	getRequests         []datasafesdk.GetSecurityPolicyConfigRequest
	listRequests        []datasafesdk.ListSecurityPolicyConfigsRequest
	updateRequests      []datasafesdk.UpdateSecurityPolicyConfigRequest
	deleteRequests      []datasafesdk.DeleteSecurityPolicyConfigRequest
	changeRequests      []datasafesdk.ChangeSecurityPolicyConfigCompartmentRequest
	workRequestRequests []datasafesdk.GetWorkRequestRequest

	getResults    []securityPolicyConfigGetResult
	listResponses []datasafesdk.ListSecurityPolicyConfigsResponse
	workRequests  map[string]datasafesdk.WorkRequest

	createErr error
	updateErr error
	deleteErr error
	changeErr error

	createWorkRequestID string
	updateWorkRequestID string
	deleteWorkRequestID string
	changeWorkRequestID string
}

type securityPolicyConfigGetResult struct {
	response datasafesdk.GetSecurityPolicyConfigResponse
	err      error
}

func (f *fakeSecurityPolicyConfigOCIClient) CreateSecurityPolicyConfig(
	_ context.Context,
	request datasafesdk.CreateSecurityPolicyConfigRequest,
) (datasafesdk.CreateSecurityPolicyConfigResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return datasafesdk.CreateSecurityPolicyConfigResponse{}, f.createErr
	}
	resource := securityPolicyConfigFromCreateDetails(testSecurityPolicyConfigID, request.CreateSecurityPolicyConfigDetails)
	f.ensureResources()[testSecurityPolicyConfigID] = resource
	return datasafesdk.CreateSecurityPolicyConfigResponse{
		SecurityPolicyConfig: resource,
		OpcWorkRequestId:     common.String(f.createWorkRequestID),
		OpcRequestId:         common.String("opc-create-1"),
	}, nil
}

func (f *fakeSecurityPolicyConfigOCIClient) GetSecurityPolicyConfig(
	_ context.Context,
	request datasafesdk.GetSecurityPolicyConfigRequest,
) (datasafesdk.GetSecurityPolicyConfigResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) > 0 {
		result := f.getResults[0]
		f.getResults = f.getResults[1:]
		return result.response, result.err
	}

	id := securityPolicyConfigStringValue(request.SecurityPolicyConfigId)
	resource, ok := f.resources[id]
	if !ok {
		return datasafesdk.GetSecurityPolicyConfigResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	return datasafesdk.GetSecurityPolicyConfigResponse{
		SecurityPolicyConfig: resource,
		OpcRequestId:         common.String("opc-get-1"),
	}, nil
}

func (f *fakeSecurityPolicyConfigOCIClient) ListSecurityPolicyConfigs(
	_ context.Context,
	request datasafesdk.ListSecurityPolicyConfigsRequest,
) (datasafesdk.ListSecurityPolicyConfigsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResponses) > 0 {
		response := f.listResponses[0]
		f.listResponses = f.listResponses[1:]
		return response, nil
	}

	var items []datasafesdk.SecurityPolicyConfigSummary
	for _, resource := range f.resources {
		if securityPolicyConfigMatchesListRequest(resource, request) {
			items = append(items, securityPolicyConfigSummaryFromSDK(resource))
		}
	}
	return datasafesdk.ListSecurityPolicyConfigsResponse{
		SecurityPolicyConfigCollection: datasafesdk.SecurityPolicyConfigCollection{Items: items},
		OpcRequestId:                   common.String("opc-list-1"),
	}, nil
}

func (f *fakeSecurityPolicyConfigOCIClient) UpdateSecurityPolicyConfig(
	_ context.Context,
	request datasafesdk.UpdateSecurityPolicyConfigRequest,
) (datasafesdk.UpdateSecurityPolicyConfigResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return datasafesdk.UpdateSecurityPolicyConfigResponse{}, f.updateErr
	}
	return datasafesdk.UpdateSecurityPolicyConfigResponse{
		OpcWorkRequestId: common.String(f.updateWorkRequestID),
		OpcRequestId:     common.String("opc-update-1"),
	}, nil
}

func (f *fakeSecurityPolicyConfigOCIClient) DeleteSecurityPolicyConfig(
	_ context.Context,
	request datasafesdk.DeleteSecurityPolicyConfigRequest,
) (datasafesdk.DeleteSecurityPolicyConfigResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return datasafesdk.DeleteSecurityPolicyConfigResponse{}, f.deleteErr
	}
	return datasafesdk.DeleteSecurityPolicyConfigResponse{
		OpcWorkRequestId: common.String(f.deleteWorkRequestID),
		OpcRequestId:     common.String("opc-delete-1"),
	}, nil
}

func (f *fakeSecurityPolicyConfigOCIClient) ChangeSecurityPolicyConfigCompartment(
	_ context.Context,
	request datasafesdk.ChangeSecurityPolicyConfigCompartmentRequest,
) (datasafesdk.ChangeSecurityPolicyConfigCompartmentResponse, error) {
	f.changeRequests = append(f.changeRequests, request)
	if f.changeErr != nil {
		return datasafesdk.ChangeSecurityPolicyConfigCompartmentResponse{}, f.changeErr
	}
	return datasafesdk.ChangeSecurityPolicyConfigCompartmentResponse{
		OpcWorkRequestId: common.String(f.changeWorkRequestID),
		OpcRequestId:     common.String("opc-change-1"),
	}, nil
}

func (f *fakeSecurityPolicyConfigOCIClient) GetWorkRequest(
	_ context.Context,
	request datasafesdk.GetWorkRequestRequest,
) (datasafesdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	workRequest, ok := f.workRequests[securityPolicyConfigStringValue(request.WorkRequestId)]
	if !ok {
		return datasafesdk.GetWorkRequestResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "work request not found")
	}
	return datasafesdk.GetWorkRequestResponse{
		WorkRequest:  workRequest,
		OpcRequestId: common.String("opc-work-request-1"),
	}, nil
}

func (f *fakeSecurityPolicyConfigOCIClient) ensureResources() map[string]datasafesdk.SecurityPolicyConfig {
	if f.resources == nil {
		f.resources = map[string]datasafesdk.SecurityPolicyConfig{}
	}
	return f.resources
}

func TestSecurityPolicyConfigRuntimeHooks(t *testing.T) {
	hooks := newSecurityPolicyConfigRuntimeHooksWithOCIClient(&fakeSecurityPolicyConfigOCIClient{})
	applySecurityPolicyConfigRuntimeHooks(&SecurityPolicyConfigServiceManager{}, &hooks, nil, nil)

	assertSecurityPolicyConfigRuntimeHooks(t, hooks)
	assertSecurityPolicyConfigCreateBody(t, hooks)
}

func assertSecurityPolicyConfigRuntimeHooks(t *testing.T, hooks SecurityPolicyConfigRuntimeHooks) {
	t.Helper()
	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed runtime semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got)
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	assertContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "compartmentId", "displayName", "description", "firewallConfig", "unifiedAuditPolicyConfig", "freeformTags", "definedTags")
	assertContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "securityPolicyId")
	assertContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "compartmentId", "securityPolicyId", "id")
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("BuildCreateBody/BuildUpdateBody = nil, want resource-specific body builders")
	}
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("Async.GetWorkRequest = nil, want work-request tracking")
	}
	if hooks.DeleteHooks.ConfirmRead == nil || hooks.DeleteHooks.HandleError == nil || hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("DeleteHooks incomplete, want conservative delete confirmation")
	}
}

func assertSecurityPolicyConfigCreateBody(t *testing.T, hooks SecurityPolicyConfigRuntimeHooks) {
	t.Helper()
	body, err := hooks.BuildCreateBody(context.Background(), securityPolicyConfigResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	createBody := body.(datasafesdk.CreateSecurityPolicyConfigDetails)
	if got := securityPolicyConfigStringValue(createBody.CompartmentId); got != testSecurityPolicyCompartmentID {
		t.Fatalf("create compartmentId = %q, want %q", got, testSecurityPolicyCompartmentID)
	}
	if createBody.FirewallConfig == nil || createBody.FirewallConfig.Status != datasafesdk.FirewallConfigDetailsStatusEnabled {
		t.Fatalf("create firewallConfig = %#v, want enabled firewall", createBody.FirewallConfig)
	}
	if createBody.UnifiedAuditPolicyConfig == nil ||
		createBody.UnifiedAuditPolicyConfig.ExcludeDatasafeUser != datasafesdk.UnifiedAuditPolicyConfigDetailsExcludeDatasafeUserEnabled {
		t.Fatalf("create unifiedAuditPolicyConfig = %#v, want enabled excludeDatasafeUser", createBody.UnifiedAuditPolicyConfig)
	}
}

func TestSecurityPolicyConfigCreateOrUpdateCreatesAndTracksWorkRequest(t *testing.T) {
	created := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	client := &fakeSecurityPolicyConfigOCIClient{
		createWorkRequestID: "wr-create-1",
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-create-1": securityPolicyConfigWorkRequest(
				"wr-create-1",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeCreateSecurityPolicyConfig,
				datasafesdk.WorkRequestResourceActionTypeCreated,
				testSecurityPolicyConfigID,
			),
		},
		getResults: []securityPolicyConfigGetResult{{response: securityPolicyConfigGetResponse(created)}},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	createBody := client.createRequests[0].CreateSecurityPolicyConfigDetails
	if got := securityPolicyConfigStringValue(createBody.SecurityPolicyId); got != testSecurityPolicyID {
		t.Fatalf("create securityPolicyId = %q, want %q", got, testSecurityPolicyID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testSecurityPolicyConfigID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSecurityPolicyConfigID)
	}
	if got := resource.Status.Id; got != testSecurityPolicyConfigID {
		t.Fatalf("status.id = %q, want %q", got, testSecurityPolicyConfigID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after succeeded work request readback", resource.Status.OsokStatus.Async.Current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestSecurityPolicyConfigCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	existing := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	client := &fakeSecurityPolicyConfigOCIClient{
		resources: map[string]datasafesdk.SecurityPolicyConfig{testSecurityPolicyConfigID: existing},
		listResponses: []datasafesdk.ListSecurityPolicyConfigsResponse{
			{
				SecurityPolicyConfigCollection: datasafesdk.SecurityPolicyConfigCollection{},
				OpcNextPage:                    common.String("page-2"),
			},
			{
				SecurityPolicyConfigCollection: datasafesdk.SecurityPolicyConfigCollection{
					Items: []datasafesdk.SecurityPolicyConfigSummary{securityPolicyConfigSummaryFromSDK(existing)},
				},
			},
		},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 for pagination", len(client.listRequests))
	}
	if got := securityPolicyConfigStringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if got := resource.Status.Id; got != testSecurityPolicyConfigID {
		t.Fatalf("status.id = %q, want bound ID %q", got, testSecurityPolicyConfigID)
	}
}

func TestSecurityPolicyConfigCreateOrUpdateNoopUsesTrackedGet(t *testing.T) {
	current := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	client := &fakeSecurityPolicyConfigOCIClient{
		resources: map[string]datasafesdk.SecurityPolicyConfig{testSecurityPolicyConfigID: current},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Status.Id = testSecurityPolicyConfigID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSecurityPolicyConfigID)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for tracked resource", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for matching readback", len(client.updateRequests))
	}
}

func TestSecurityPolicyConfigCreateOrUpdateMutableUpdate(t *testing.T) {
	current := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	current.DisplayName = common.String("old-name")
	current.Description = common.String("old description")
	current.FirewallConfig.Status = datasafesdk.FirewallConfigStatusEnabled

	updated := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	updated.DisplayName = common.String("updated-name")
	updated.Description = common.String("")
	updated.FirewallConfig.Status = datasafesdk.FirewallConfigStatusDisabled
	updated.UnifiedAuditPolicyConfig.ExcludeDatasafeUser = datasafesdk.UnifiedAuditPolicyConfigExcludeDatasafeUserDisabled
	updated.FreeformTags = map[string]string{"env": "prod"}

	client := &fakeSecurityPolicyConfigOCIClient{
		updateWorkRequestID: "wr-update-1",
		getResults: []securityPolicyConfigGetResult{
			{response: securityPolicyConfigGetResponse(current)},
			{response: securityPolicyConfigGetResponse(updated)},
		},
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-update-1": securityPolicyConfigWorkRequest(
				"wr-update-1",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeUpdateSecurityPolicyConfig,
				datasafesdk.WorkRequestResourceActionTypeUpdated,
				testSecurityPolicyConfigID,
			),
		},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Status.Id = testSecurityPolicyConfigID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSecurityPolicyConfigID)
	resource.Spec.DisplayName = "updated-name"
	resource.Spec.Description = ""
	resource.Spec.FirewallConfig.Status = string(datasafesdk.FirewallConfigStatusDisabled)
	resource.Spec.UnifiedAuditPolicyConfig.ExcludeDatasafeUser = string(datasafesdk.UnifiedAuditPolicyConfigExcludeDatasafeUserDisabled)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	assertSecurityPolicyConfigMutableUpdateRequest(t, client.updateRequests[0])
	if got := resource.Status.DisplayName; got != "updated-name" {
		t.Fatalf("status.displayName = %q, want updated-name", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after update work request", resource.Status.OsokStatus.Async.Current)
	}
}

func assertSecurityPolicyConfigMutableUpdateRequest(
	t *testing.T,
	request datasafesdk.UpdateSecurityPolicyConfigRequest,
) {
	t.Helper()
	updateBody := request.UpdateSecurityPolicyConfigDetails
	if got := securityPolicyConfigStringValue(updateBody.DisplayName); got != "updated-name" {
		t.Fatalf("update displayName = %q, want updated-name", got)
	}
	if updateBody.Description == nil || *updateBody.Description != "" {
		t.Fatalf("update description = %#v, want explicit empty string", updateBody.Description)
	}
	if updateBody.FirewallConfig == nil || updateBody.FirewallConfig.Status != datasafesdk.FirewallConfigDetailsStatusDisabled {
		t.Fatalf("update firewallConfig = %#v, want disabled firewall", updateBody.FirewallConfig)
	}
}

func TestSecurityPolicyConfigCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	current := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	current.SecurityPolicyId = common.String(testOtherSecurityPolicyID)
	client := &fakeSecurityPolicyConfigOCIClient{
		resources: map[string]datasafesdk.SecurityPolicyConfig{testSecurityPolicyConfigID: current},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Status.Id = testSecurityPolicyConfigID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSecurityPolicyConfigID)

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable securityPolicyId drift")
	}
	if !strings.Contains(err.Error(), "securityPolicyId") {
		t.Fatalf("CreateOrUpdate() error = %v, want securityPolicyId drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after create-only drift", len(client.updateRequests))
	}
}

func TestSecurityPolicyConfigCreateOrUpdateMovesCompartment(t *testing.T) {
	current := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	current.CompartmentId = common.String(testSecurityPolicyCompartmentID)
	client := &fakeSecurityPolicyConfigOCIClient{
		resources:           map[string]datasafesdk.SecurityPolicyConfig{testSecurityPolicyConfigID: current},
		changeWorkRequestID: "wr-change-1",
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Spec.CompartmentId = testMovedCompartmentID
	resource.Status.Id = testSecurityPolicyConfigID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSecurityPolicyConfigID)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.changeRequests) != 1 {
		t.Fatalf("change compartment requests = %d, want 1", len(client.changeRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 while compartment move is handled separately", len(client.updateRequests))
	}
	changeBody := client.changeRequests[0].ChangeSecurityPolicyConfigCompartmentDetails
	if got := securityPolicyConfigStringValue(changeBody.CompartmentId); got != testMovedCompartmentID {
		t.Fatalf("change compartmentId = %q, want %q", got, testMovedCompartmentID)
	}
	assertCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, "wr-change-1")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-change-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-change-1", got)
	}
}

func TestSecurityPolicyConfigDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	client := &fakeSecurityPolicyConfigOCIClient{
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-create-pending": securityPolicyConfigWorkRequest(
				"wr-create-pending",
				datasafesdk.WorkRequestStatusInProgress,
				datasafesdk.WorkRequestOperationTypeCreateSecurityPolicyConfig,
				datasafesdk.WorkRequestResourceActionTypeInProgress,
				testSecurityPolicyConfigID,
			),
		},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:        shared.OSOKAsyncSourceWorkRequest,
		Phase:         shared.OSOKAsyncPhaseCreate,
		WorkRequestID: "wr-create-pending",
	}

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while create work request is pending")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 while create work request is pending", len(client.deleteRequests))
	}
	assertCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-pending")
}

func TestSecurityPolicyConfigDeleteWaitsForWorkRequestAndConfirmsNotFound(t *testing.T) {
	active := securityPolicyConfigSDK(testSecurityPolicyConfigID, datasafesdk.SecurityPolicyConfigLifecycleStateActive)
	client := &fakeSecurityPolicyConfigOCIClient{
		deleteWorkRequestID: "wr-delete-1",
		getResults: []securityPolicyConfigGetResult{
			{response: securityPolicyConfigGetResponse(active)},
			{err: errortest.NewServiceError(404, errorutil.NotFound, "deleted")},
		},
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-delete-1": securityPolicyConfigWorkRequest(
				"wr-delete-1",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeDeleteSecurityPolicyConfig,
				datasafesdk.WorkRequestResourceActionTypeDeleted,
				testSecurityPolicyConfigID,
			),
		},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Status.Id = testSecurityPolicyConfigID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSecurityPolicyConfigID)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after confirmed not found")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestSecurityPolicyConfigDeleteKeepsFinalizerForAuthShapedPreDeleteRead(t *testing.T) {
	client := &fakeSecurityPolicyConfigOCIClient{
		getResults: []securityPolicyConfigGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		}},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Finalizers = []string{"osok.oracle.com/finalizer"}
	resource.Status.Id = testSecurityPolicyConfigID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSecurityPolicyConfigID)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after auth-shaped pre-delete read", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for auth-shaped pre-delete read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if got, want := resource.Finalizers, []string{"osok.oracle.com/finalizer"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("finalizers = %#v, want %#v", got, want)
	}
}

func TestSecurityPolicyConfigDeleteWorkRequestAuthShapedReadbackKeepsFinalizer(t *testing.T) {
	client := &fakeSecurityPolicyConfigOCIClient{
		getResults: []securityPolicyConfigGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		}},
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-delete-done": securityPolicyConfigWorkRequest(
				"wr-delete-done",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeDeleteSecurityPolicyConfig,
				datasafesdk.WorkRequestResourceActionTypeDeleted,
				testSecurityPolicyConfigID,
			),
		},
	}
	serviceClient := newSecurityPolicyConfigServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := securityPolicyConfigResource()
	resource.Status.Id = testSecurityPolicyConfigID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSecurityPolicyConfigID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete-done",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped work-request readback error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped work-request readback")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 while resuming tracked delete work request", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for auth-shaped delete work-request readback")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	assertCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete-done")
}

func securityPolicyConfigResource() *datasafev1beta1.SecurityPolicyConfig {
	return &datasafev1beta1.SecurityPolicyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "security-policy-config",
			Namespace: "default",
			UID:       types.UID("security-policy-config-uid"),
		},
		Spec: datasafev1beta1.SecurityPolicyConfigSpec{
			CompartmentId:    testSecurityPolicyCompartmentID,
			SecurityPolicyId: testSecurityPolicyID,
			DisplayName:      "security-policy-config",
			Description:      "config description",
			FirewallConfig: datasafev1beta1.SecurityPolicyConfigFirewallConfig{
				Status:                string(datasafesdk.FirewallConfigStatusEnabled),
				ViolationLogAutoPurge: string(datasafesdk.FirewallConfigViolationLogAutoPurgeDisabled),
				ExcludeJob:            string(datasafesdk.FirewallConfigExcludeJobIncluded),
			},
			UnifiedAuditPolicyConfig: datasafev1beta1.SecurityPolicyConfigUnifiedAuditPolicyConfig{
				ExcludeDatasafeUser: string(datasafesdk.UnifiedAuditPolicyConfigExcludeDatasafeUserEnabled),
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func securityPolicyConfigSDK(
	id string,
	state datasafesdk.SecurityPolicyConfigLifecycleStateEnum,
) datasafesdk.SecurityPolicyConfig {
	return datasafesdk.SecurityPolicyConfig{
		Id:               common.String(id),
		CompartmentId:    common.String(testSecurityPolicyCompartmentID),
		DisplayName:      common.String("security-policy-config"),
		SecurityPolicyId: common.String(testSecurityPolicyID),
		LifecycleState:   state,
		Description:      common.String("config description"),
		FirewallConfig: &datasafesdk.FirewallConfig{
			Status:                datasafesdk.FirewallConfigStatusEnabled,
			ViolationLogAutoPurge: datasafesdk.FirewallConfigViolationLogAutoPurgeDisabled,
			ExcludeJob:            datasafesdk.FirewallConfigExcludeJobIncluded,
		},
		UnifiedAuditPolicyConfig: &datasafesdk.UnifiedAuditPolicyConfig{
			ExcludeDatasafeUser: datasafesdk.UnifiedAuditPolicyConfigExcludeDatasafeUserEnabled,
		},
		FreeformTags: map[string]string{"env": "dev"},
		DefinedTags:  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func securityPolicyConfigFromCreateDetails(
	id string,
	details datasafesdk.CreateSecurityPolicyConfigDetails,
) datasafesdk.SecurityPolicyConfig {
	return datasafesdk.SecurityPolicyConfig{
		Id:                       common.String(id),
		CompartmentId:            details.CompartmentId,
		DisplayName:              details.DisplayName,
		SecurityPolicyId:         details.SecurityPolicyId,
		LifecycleState:           datasafesdk.SecurityPolicyConfigLifecycleStateCreating,
		Description:              details.Description,
		FirewallConfig:           securityPolicyConfigFirewallFromDetails(details.FirewallConfig),
		UnifiedAuditPolicyConfig: securityPolicyConfigUnifiedAuditFromDetails(details.UnifiedAuditPolicyConfig),
		FreeformTags:             details.FreeformTags,
		DefinedTags:              details.DefinedTags,
	}
}

func securityPolicyConfigFirewallFromDetails(details *datasafesdk.FirewallConfigDetails) *datasafesdk.FirewallConfig {
	if details == nil {
		return nil
	}
	return &datasafesdk.FirewallConfig{
		Status:                datasafesdk.FirewallConfigStatusEnum(details.Status),
		ViolationLogAutoPurge: datasafesdk.FirewallConfigViolationLogAutoPurgeEnum(details.ViolationLogAutoPurge),
		ExcludeJob:            datasafesdk.FirewallConfigExcludeJobEnum(details.ExcludeJob),
	}
}

func securityPolicyConfigUnifiedAuditFromDetails(
	details *datasafesdk.UnifiedAuditPolicyConfigDetails,
) *datasafesdk.UnifiedAuditPolicyConfig {
	if details == nil {
		return nil
	}
	return &datasafesdk.UnifiedAuditPolicyConfig{
		ExcludeDatasafeUser: datasafesdk.UnifiedAuditPolicyConfigExcludeDatasafeUserEnum(details.ExcludeDatasafeUser),
	}
}

func securityPolicyConfigGetResponse(
	resource datasafesdk.SecurityPolicyConfig,
) datasafesdk.GetSecurityPolicyConfigResponse {
	return datasafesdk.GetSecurityPolicyConfigResponse{
		SecurityPolicyConfig: resource,
		OpcRequestId:         common.String("opc-get-1"),
	}
}

func securityPolicyConfigSummaryFromSDK(
	resource datasafesdk.SecurityPolicyConfig,
) datasafesdk.SecurityPolicyConfigSummary {
	return datasafesdk.SecurityPolicyConfigSummary(resource)
}

func securityPolicyConfigMatchesListRequest(
	resource datasafesdk.SecurityPolicyConfig,
	request datasafesdk.ListSecurityPolicyConfigsRequest,
) bool {
	if request.CompartmentId != nil && securityPolicyConfigStringValue(resource.CompartmentId) != securityPolicyConfigStringValue(request.CompartmentId) {
		return false
	}
	if request.SecurityPolicyConfigId != nil && securityPolicyConfigStringValue(resource.Id) != securityPolicyConfigStringValue(request.SecurityPolicyConfigId) {
		return false
	}
	if request.SecurityPolicyId != nil && securityPolicyConfigStringValue(resource.SecurityPolicyId) != securityPolicyConfigStringValue(request.SecurityPolicyId) {
		return false
	}
	return true
}

func securityPolicyConfigWorkRequest(
	id string,
	status datasafesdk.WorkRequestStatusEnum,
	operation datasafesdk.WorkRequestOperationTypeEnum,
	action datasafesdk.WorkRequestResourceActionTypeEnum,
	resourceID string,
) datasafesdk.WorkRequest {
	return datasafesdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operation,
		Resources: []datasafesdk.WorkRequestResource{{
			EntityType: common.String("SecurityPolicyConfig"),
			ActionType: action,
			Identifier: common.String(resourceID),
		}},
	}
}

func assertContainsAll(t *testing.T, label string, got []string, wants ...string) {
	t.Helper()
	values := make(map[string]struct{}, len(got))
	for _, value := range got {
		values[value] = struct{}{}
	}
	for _, want := range wants {
		if _, ok := values[want]; !ok {
			t.Fatalf("%s = %#v, missing %q", label, got, want)
		}
	}
}

func assertCurrentWorkRequest(
	t *testing.T,
	resource *datasafev1beta1.SecurityPolicyConfig,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.async.current = nil, want %s %s", phase, class)
	}
	if current.Phase != phase || current.NormalizedClass != class || current.WorkRequestID != workRequestID {
		t.Fatalf(
			"status.async.current = phase %q class %q workRequestID %q, want phase %q class %q workRequestID %q",
			current.Phase,
			current.NormalizedClass,
			current.WorkRequestID,
			phase,
			class,
			workRequestID,
		)
	}
}
