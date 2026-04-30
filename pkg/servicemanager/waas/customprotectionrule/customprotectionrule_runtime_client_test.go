/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package customprotectionrule

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCustomProtectionRuleID          = "ocid1.customprotectionrule.oc1..example"
	testCustomProtectionRuleOtherID     = "ocid1.customprotectionrule.oc1..other"
	testCustomProtectionRuleCompartment = "ocid1.compartment.oc1..example"
)

type fakeCustomProtectionRuleOCIClient struct {
	createFn func(context.Context, waassdk.CreateCustomProtectionRuleRequest) (waassdk.CreateCustomProtectionRuleResponse, error)
	getFn    func(context.Context, waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error)
	listFn   func(context.Context, waassdk.ListCustomProtectionRulesRequest) (waassdk.ListCustomProtectionRulesResponse, error)
	updateFn func(context.Context, waassdk.UpdateCustomProtectionRuleRequest) (waassdk.UpdateCustomProtectionRuleResponse, error)
	deleteFn func(context.Context, waassdk.DeleteCustomProtectionRuleRequest) (waassdk.DeleteCustomProtectionRuleResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeCustomProtectionRuleOCIClient) CreateCustomProtectionRule(
	ctx context.Context,
	request waassdk.CreateCustomProtectionRuleRequest,
) (waassdk.CreateCustomProtectionRuleResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return waassdk.CreateCustomProtectionRuleResponse{}, nil
}

func (f *fakeCustomProtectionRuleOCIClient) GetCustomProtectionRule(
	ctx context.Context,
	request waassdk.GetCustomProtectionRuleRequest,
) (waassdk.GetCustomProtectionRuleResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return waassdk.GetCustomProtectionRuleResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "custom protection rule missing")
}

func (f *fakeCustomProtectionRuleOCIClient) ListCustomProtectionRules(
	ctx context.Context,
	request waassdk.ListCustomProtectionRulesRequest,
) (waassdk.ListCustomProtectionRulesResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return waassdk.ListCustomProtectionRulesResponse{}, nil
}

func (f *fakeCustomProtectionRuleOCIClient) UpdateCustomProtectionRule(
	ctx context.Context,
	request waassdk.UpdateCustomProtectionRuleRequest,
) (waassdk.UpdateCustomProtectionRuleResponse, error) {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return waassdk.UpdateCustomProtectionRuleResponse{}, nil
}

func (f *fakeCustomProtectionRuleOCIClient) DeleteCustomProtectionRule(
	ctx context.Context,
	request waassdk.DeleteCustomProtectionRuleRequest,
) (waassdk.DeleteCustomProtectionRuleResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return waassdk.DeleteCustomProtectionRuleResponse{}, nil
}

func testCustomProtectionRuleClient(fake *fakeCustomProtectionRuleOCIClient) CustomProtectionRuleServiceClient {
	if fake == nil {
		fake = &fakeCustomProtectionRuleOCIClient{}
	}
	hooks := testCustomProtectionRuleRuntimeHooks(fake)
	applyCustomProtectionRuleRuntimeHooks(&hooks)
	manager := &CustomProtectionRuleServiceManager{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}
	delegate := defaultCustomProtectionRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waasv1beta1.CustomProtectionRule](
			buildCustomProtectionRuleGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapCustomProtectionRuleGeneratedClient(hooks, delegate)
}

func testCustomProtectionRuleRuntimeHooks(fake *fakeCustomProtectionRuleOCIClient) CustomProtectionRuleRuntimeHooks {
	return CustomProtectionRuleRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*waasv1beta1.CustomProtectionRule]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*waasv1beta1.CustomProtectionRule]{},
		StatusHooks:     generatedruntime.StatusHooks[*waasv1beta1.CustomProtectionRule]{},
		ParityHooks:     generatedruntime.ParityHooks[*waasv1beta1.CustomProtectionRule]{},
		Async:           generatedruntime.AsyncHooks[*waasv1beta1.CustomProtectionRule]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*waasv1beta1.CustomProtectionRule]{},
		Create: runtimeOperationHooks[waassdk.CreateCustomProtectionRuleRequest, waassdk.CreateCustomProtectionRuleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateCustomProtectionRuleDetails", RequestName: "CreateCustomProtectionRuleDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request waassdk.CreateCustomProtectionRuleRequest) (waassdk.CreateCustomProtectionRuleResponse, error) {
				return fake.CreateCustomProtectionRule(ctx, request)
			},
		},
		Get: runtimeOperationHooks[waassdk.GetCustomProtectionRuleRequest, waassdk.GetCustomProtectionRuleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CustomProtectionRuleId", RequestName: "customProtectionRuleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
				return fake.GetCustomProtectionRule(ctx, request)
			},
		},
		List: runtimeOperationHooks[waassdk.ListCustomProtectionRulesRequest, waassdk.ListCustomProtectionRulesResponse]{
			Fields: customProtectionRuleListFields(),
			Call: func(ctx context.Context, request waassdk.ListCustomProtectionRulesRequest) (waassdk.ListCustomProtectionRulesResponse, error) {
				return fake.ListCustomProtectionRules(ctx, request)
			},
		},
		Update: runtimeOperationHooks[waassdk.UpdateCustomProtectionRuleRequest, waassdk.UpdateCustomProtectionRuleResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CustomProtectionRuleId", RequestName: "customProtectionRuleId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateCustomProtectionRuleDetails", RequestName: "UpdateCustomProtectionRuleDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request waassdk.UpdateCustomProtectionRuleRequest) (waassdk.UpdateCustomProtectionRuleResponse, error) {
				return fake.UpdateCustomProtectionRule(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[waassdk.DeleteCustomProtectionRuleRequest, waassdk.DeleteCustomProtectionRuleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CustomProtectionRuleId", RequestName: "customProtectionRuleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request waassdk.DeleteCustomProtectionRuleRequest) (waassdk.DeleteCustomProtectionRuleResponse, error) {
				return fake.DeleteCustomProtectionRule(ctx, request)
			},
		},
		WrapGeneratedClient: []func(CustomProtectionRuleServiceClient) CustomProtectionRuleServiceClient{},
	}
}

func makeCustomProtectionRuleResource() *waasv1beta1.CustomProtectionRule {
	return &waasv1beta1.CustomProtectionRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-protection-rule-sample",
			Namespace: "default",
			UID:       types.UID("custom-protection-rule-uid"),
		},
		Spec: waasv1beta1.CustomProtectionRuleSpec{
			CompartmentId: testCustomProtectionRuleCompartment,
			DisplayName:   "custom-rule",
			Template:      `SecRule REQUEST_HEADERS "example" "id: {{id_1}}, ctl:ruleEngine={{mode}}"`,
			Description:   "blocks example input",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKCustomProtectionRule(
	id string,
	resource *waasv1beta1.CustomProtectionRule,
	state waassdk.LifecycleStatesEnum,
) waassdk.CustomProtectionRule {
	return waassdk.CustomProtectionRule{
		Id:                 common.String(id),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		DisplayName:        common.String(resource.Spec.DisplayName),
		Description:        common.String(resource.Spec.Description),
		ModSecurityRuleIds: []string{"100001"},
		Template:           common.String(resource.Spec.Template),
		LifecycleState:     state,
		FreeformTags:       cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:        customProtectionRuleDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKCustomProtectionRuleSummary(
	id string,
	compartmentID string,
	displayName string,
	state waassdk.LifecycleStatesEnum,
) waassdk.CustomProtectionRuleSummary {
	return waassdk.CustomProtectionRuleSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
	}
}

func reconcileRequest(resource *waasv1beta1.CustomProtectionRule) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func TestCustomProtectionRuleRuntimeSemantics(t *testing.T) {
	hooks := CustomProtectionRuleRuntimeHooks{}
	applyCustomProtectionRuleRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed CustomProtectionRule semantics")
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("Async = %#v, want lifecycle semantics", hooks.Semantics.Async)
	}
	if hooks.Semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", hooks.Semantics.FinalizerPolicy)
	}
	if hooks.Semantics.Delete.Policy != "required" || hooks.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", hooks.Semantics.Delete, hooks.Semantics.DeleteFollowUp)
	}
	assertStringSliceContainsAll(t, "Lifecycle.ActiveStates", hooks.Semantics.Lifecycle.ActiveStates, "ACTIVE")
	assertStringSliceContainsAll(t, "Delete.PendingStates", hooks.Semantics.Delete.PendingStates, "DELETING")
	assertStringSliceContainsAll(t, "Delete.TerminalStates", hooks.Semantics.Delete.TerminalStates, "DELETED")
	assertStringSliceContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "compartmentId", "displayName", "id")
	assertStringSliceContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "definedTags", "description", "displayName", "freeformTags", "template")
	assertStringSliceContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId")
	if hooks.BuildUpdateBody == nil {
		t.Fatal("BuildUpdateBody = nil, want explicit update builder")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}
	if len(customProtectionRuleListFields()) != 2 {
		t.Fatalf("customProtectionRuleListFields() = %#v, want compartment/page only to avoid scalar-to-slice filters", customProtectionRuleListFields())
	}
}

func TestCustomProtectionRuleCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	var listRequest waassdk.ListCustomProtectionRulesRequest
	var createRequest waassdk.CreateCustomProtectionRuleRequest
	var getRequest waassdk.GetCustomProtectionRuleRequest
	fake := &fakeCustomProtectionRuleOCIClient{
		listFn: func(_ context.Context, request waassdk.ListCustomProtectionRulesRequest) (waassdk.ListCustomProtectionRulesResponse, error) {
			listRequest = request
			return waassdk.ListCustomProtectionRulesResponse{}, nil
		},
		createFn: func(_ context.Context, request waassdk.CreateCustomProtectionRuleRequest) (waassdk.CreateCustomProtectionRuleResponse, error) {
			createRequest = request
			return waassdk.CreateCustomProtectionRuleResponse{
				CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesActive),
				OpcRequestId:         common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
			getRequest = request
			return waassdk.GetCustomProtectionRuleResponse{
				CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesActive),
			}, nil
		},
	}

	response, err := testCustomProtectionRuleClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful active create", response)
	}
	requireStringPtr(t, "list compartmentId", listRequest.CompartmentId, testCustomProtectionRuleCompartment)
	if listRequest.DisplayName != nil || listRequest.Id != nil {
		t.Fatalf("list filters displayName=%#v id=%#v, want nil slice filters", listRequest.DisplayName, listRequest.Id)
	}
	requireStringPtr(t, "create compartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "create template", createRequest.Template, resource.Spec.Template)
	requireStringPtr(t, "create description", createRequest.Description, resource.Spec.Description)
	requireStringPtr(t, "get customProtectionRuleId", getRequest.CustomProtectionRuleId, testCustomProtectionRuleID)
	if got := string(resource.Status.OsokStatus.Ocid); got != testCustomProtectionRuleID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testCustomProtectionRuleID)
	}
	if got := resource.Status.Id; got != testCustomProtectionRuleID {
		t.Fatalf("status.id = %q, want %q", got, testCustomProtectionRuleID)
	}
	if got := resource.Status.LifecycleState; got != string(waassdk.LifecycleStatesActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestCustomProtectionRuleCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	fake := &fakeCustomProtectionRuleOCIClient{}
	fake.listFn = func(_ context.Context, request waassdk.ListCustomProtectionRulesRequest) (waassdk.ListCustomProtectionRulesResponse, error) {
		switch fake.listCalls {
		case 1:
			if request.Page != nil {
				t.Fatalf("first ListCustomProtectionRules page = %q, want nil", *request.Page)
			}
			return waassdk.ListCustomProtectionRulesResponse{
				Items: []waassdk.CustomProtectionRuleSummary{
					makeSDKCustomProtectionRuleSummary(testCustomProtectionRuleOtherID, testCustomProtectionRuleCompartment, "other-rule", waassdk.LifecycleStatesActive),
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			requireStringPtr(t, "second ListCustomProtectionRules page", request.Page, "page-2")
			return waassdk.ListCustomProtectionRulesResponse{
				Items: []waassdk.CustomProtectionRuleSummary{
					makeSDKCustomProtectionRuleSummary(testCustomProtectionRuleID, testCustomProtectionRuleCompartment, resource.Spec.DisplayName, waassdk.LifecycleStatesActive),
				},
			}, nil
		default:
			t.Fatalf("unexpected ListCustomProtectionRules call %d", fake.listCalls)
			return waassdk.ListCustomProtectionRulesResponse{}, nil
		}
	}
	fake.getFn = func(_ context.Context, request waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
		requireStringPtr(t, "get customProtectionRuleId", request.CustomProtectionRuleId, testCustomProtectionRuleID)
		return waassdk.GetCustomProtectionRuleResponse{
			CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.createFn = func(context.Context, waassdk.CreateCustomProtectionRuleRequest) (waassdk.CreateCustomProtectionRuleResponse, error) {
		t.Fatal("CreateCustomProtectionRule should not be called when list lookup binds an existing rule")
		return waassdk.CreateCustomProtectionRuleResponse{}, nil
	}

	response, err := testCustomProtectionRuleClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful active bind", response)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListCustomProtectionRules calls = %d, want 2", fake.listCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testCustomProtectionRuleID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testCustomProtectionRuleID)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestCustomProtectionRuleCreateOrUpdateSkipsNoOpUpdate(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testCustomProtectionRuleID)
	resource.Status.Id = testCustomProtectionRuleID
	fake := &fakeCustomProtectionRuleOCIClient{
		getFn: func(_ context.Context, request waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
			requireStringPtr(t, "get customProtectionRuleId", request.CustomProtectionRuleId, testCustomProtectionRuleID)
			return waassdk.GetCustomProtectionRuleResponse{
				CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesActive),
			}, nil
		},
		updateFn: func(context.Context, waassdk.UpdateCustomProtectionRuleRequest) (waassdk.UpdateCustomProtectionRuleResponse, error) {
			t.Fatal("UpdateCustomProtectionRule should not be called when desired and observed state match")
			return waassdk.UpdateCustomProtectionRuleResponse{}, nil
		},
	}

	response, err := testCustomProtectionRuleClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateCustomProtectionRule calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestCustomProtectionRuleCreateOrUpdateAppliesMutableUpdateAndClearsOptionalFields(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testCustomProtectionRuleID)
	resource.Status.Id = testCustomProtectionRuleID
	resource.Spec.Description = ""
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Spec.Template = resource.Spec.Template + "\nSecRule REQUEST_URI \"second\" \"id: {{id_2}}, ctl:ruleEngine={{mode}}\""
	var updateRequest waassdk.UpdateCustomProtectionRuleRequest
	fake := &fakeCustomProtectionRuleOCIClient{}
	fake.getFn = func(_ context.Context, _ waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
		current := makeSDKCustomProtectionRule(testCustomProtectionRuleID, makeCustomProtectionRuleResource(), waassdk.LifecycleStatesActive)
		if fake.getCalls >= 2 {
			current = makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesActive)
		}
		return waassdk.GetCustomProtectionRuleResponse{CustomProtectionRule: current}, nil
	}
	fake.updateFn = func(_ context.Context, request waassdk.UpdateCustomProtectionRuleRequest) (waassdk.UpdateCustomProtectionRuleResponse, error) {
		updateRequest = request
		return waassdk.UpdateCustomProtectionRuleResponse{
			CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesUpdating),
			OpcRequestId:         common.String("opc-update"),
		}, nil
	}

	response, err := testCustomProtectionRuleClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful active update after follow-up read", response)
	}
	requireStringPtr(t, "update customProtectionRuleId", updateRequest.CustomProtectionRuleId, testCustomProtectionRuleID)
	requireStringPtr(t, "update description", updateRequest.Description, "")
	requireStringPtr(t, "update template", updateRequest.Template, resource.Spec.Template)
	requireEmptyStringMap(t, "update freeformTags", updateRequest.FreeformTags)
	requireEmptyDefinedTags(t, "update definedTags", updateRequest.DefinedTags)
	if got := resource.Status.Description; got != "" {
		t.Fatalf("status.description = %q, want cleared", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestCustomProtectionRuleCreateOrUpdateRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testCustomProtectionRuleID)
	resource.Status.Id = testCustomProtectionRuleID
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
	fake := &fakeCustomProtectionRuleOCIClient{
		getFn: func(_ context.Context, _ waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
			currentResource := makeCustomProtectionRuleResource()
			return waassdk.GetCustomProtectionRuleResponse{
				CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, currentResource, waassdk.LifecycleStatesActive),
			}, nil
		},
		updateFn: func(context.Context, waassdk.UpdateCustomProtectionRuleRequest) (waassdk.UpdateCustomProtectionRuleResponse, error) {
			t.Fatal("UpdateCustomProtectionRule should not be called after create-only compartment drift")
			return waassdk.UpdateCustomProtectionRuleResponse{}, nil
		},
	}

	response, err := testCustomProtectionRuleClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartment replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateCustomProtectionRule calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestCustomProtectionRuleCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	fake := &fakeCustomProtectionRuleOCIClient{
		createFn: func(context.Context, waassdk.CreateCustomProtectionRuleRequest) (waassdk.CreateCustomProtectionRuleResponse, error) {
			return waassdk.CreateCustomProtectionRuleResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	}

	response, err := testCustomProtectionRuleClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestCustomProtectionRuleDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testCustomProtectionRuleID)
	resource.Status.Id = testCustomProtectionRuleID
	fake := &fakeCustomProtectionRuleOCIClient{}
	fake.getFn = func(_ context.Context, request waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
		requireStringPtr(t, "get customProtectionRuleId", request.CustomProtectionRuleId, testCustomProtectionRuleID)
		state := waassdk.LifecycleStatesActive
		if fake.getCalls >= 3 {
			state = waassdk.LifecycleStatesDeleting
		}
		return waassdk.GetCustomProtectionRuleResponse{
			CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, state),
		}, nil
	}
	fake.deleteFn = func(_ context.Context, request waassdk.DeleteCustomProtectionRuleRequest) (waassdk.DeleteCustomProtectionRuleResponse, error) {
		requireStringPtr(t, "delete customProtectionRuleId", request.CustomProtectionRuleId, testCustomProtectionRuleID)
		return waassdk.DeleteCustomProtectionRuleResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := testCustomProtectionRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI lifecycle is DELETING")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("DeleteCustomProtectionRule calls = %d, want 1", fake.deleteCalls)
	}
	if got := resource.Status.LifecycleState; got != string(waassdk.LifecycleStatesDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending delete lifecycle operation", current)
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestCustomProtectionRuleDeleteReleasesFinalizerAfterUnambiguousNotFound(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testCustomProtectionRuleID)
	resource.Status.Id = testCustomProtectionRuleID
	fake := &fakeCustomProtectionRuleOCIClient{}
	fake.getFn = func(_ context.Context, _ waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
		if fake.getCalls >= 3 {
			return waassdk.GetCustomProtectionRuleResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		}
		return waassdk.GetCustomProtectionRuleResponse{
			CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.deleteFn = func(context.Context, waassdk.DeleteCustomProtectionRuleRequest) (waassdk.DeleteCustomProtectionRuleResponse, error) {
		return waassdk.DeleteCustomProtectionRuleResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := testCustomProtectionRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after unambiguous NotFound")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestCustomProtectionRuleDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testCustomProtectionRuleID)
	resource.Status.Id = testCustomProtectionRuleID
	fake := &fakeCustomProtectionRuleOCIClient{
		getFn: func(context.Context, waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
			return waassdk.GetCustomProtectionRuleResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, waassdk.DeleteCustomProtectionRuleRequest) (waassdk.DeleteCustomProtectionRuleResponse, error) {
			t.Fatal("DeleteCustomProtectionRule should not be called after ambiguous pre-delete confirm read")
			return waassdk.DeleteCustomProtectionRuleResponse{}, nil
		},
	}

	deleted, err := testCustomProtectionRuleClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained after ambiguous confirm read")
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("DeleteCustomProtectionRule calls = %d, want 0", fake.deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestCustomProtectionRuleDeleteRejectsAuthShapedDeleteError(t *testing.T) {
	resource := makeCustomProtectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testCustomProtectionRuleID)
	resource.Status.Id = testCustomProtectionRuleID
	fake := &fakeCustomProtectionRuleOCIClient{
		getFn: func(context.Context, waassdk.GetCustomProtectionRuleRequest) (waassdk.GetCustomProtectionRuleResponse, error) {
			return waassdk.GetCustomProtectionRuleResponse{
				CustomProtectionRule: makeSDKCustomProtectionRule(testCustomProtectionRuleID, resource, waassdk.LifecycleStatesActive),
			}, nil
		},
		deleteFn: func(context.Context, waassdk.DeleteCustomProtectionRuleRequest) (waassdk.DeleteCustomProtectionRuleResponse, error) {
			return waassdk.DeleteCustomProtectionRuleResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	}

	deleted, err := testCustomProtectionRuleClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained after ambiguous delete error")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
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

func requireEmptyStringMap(t *testing.T, name string, got map[string]string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want explicit empty map clear", name)
	}
	if len(got) != 0 {
		t.Fatalf("%s = %#v, want explicit empty map clear", name, got)
	}
}

func requireEmptyDefinedTags(t *testing.T, name string, got map[string]map[string]interface{}) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want explicit empty map clear", name)
	}
	if len(got) != 0 {
		t.Fatalf("%s = %#v, want explicit empty map clear", name, got)
	}
}

func assertTrailingCondition(t *testing.T, resource *waasv1beta1.CustomProtectionRule, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = nil, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %q, want %q", got, want)
	}
}

func assertStringSliceContainsAll(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	values := map[string]bool{}
	for _, value := range got {
		values[value] = true
	}
	for _, value := range want {
		if !values[value] {
			t.Fatalf("%s = %#v, want %q", name, got, value)
		}
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func TestCustomProtectionRuleDefinedTagsFromSpec(t *testing.T) {
	tags := customProtectionRuleDefinedTagsFromSpec(map[string]shared.MapValue{"Operations": {"CostCenter": "42"}})
	want := map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("customProtectionRuleDefinedTagsFromSpec() = %#v, want %#v", tags, want)
	}
}
