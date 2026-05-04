/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package governancerule

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	governancerulescontrolplanesdk "github.com/oracle/oci-go-sdk/v65/governancerulescontrolplane"
	governancerulescontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/governancerulescontrolplane/v1beta1"
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
	testGovernanceRuleID     = "ocid1.governancerule.oc1..example"
	testGovernanceRuleOther  = "ocid1.governancerule.oc1..other"
	testGovernanceRuleCompID = "ocid1.tenancy.oc1..example"
)

type governanceRuleOCIClient interface {
	CreateGovernanceRule(context.Context, governancerulescontrolplanesdk.CreateGovernanceRuleRequest) (governancerulescontrolplanesdk.CreateGovernanceRuleResponse, error)
	GetGovernanceRule(context.Context, governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error)
	ListGovernanceRules(context.Context, governancerulescontrolplanesdk.ListGovernanceRulesRequest) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error)
	UpdateGovernanceRule(context.Context, governancerulescontrolplanesdk.UpdateGovernanceRuleRequest) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error)
	DeleteGovernanceRule(context.Context, governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error)
	GetWorkRequest(context.Context, governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error)
}

type fakeGovernanceRuleOCIClient struct {
	createFn      func(context.Context, governancerulescontrolplanesdk.CreateGovernanceRuleRequest) (governancerulescontrolplanesdk.CreateGovernanceRuleResponse, error)
	getFn         func(context.Context, governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error)
	listFn        func(context.Context, governancerulescontrolplanesdk.ListGovernanceRulesRequest) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error)
	updateFn      func(context.Context, governancerulescontrolplanesdk.UpdateGovernanceRuleRequest) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error)
	deleteFn      func(context.Context, governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error)
	workRequestFn func(context.Context, governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error)
}

func (f *fakeGovernanceRuleOCIClient) CreateGovernanceRule(
	ctx context.Context,
	request governancerulescontrolplanesdk.CreateGovernanceRuleRequest,
) (governancerulescontrolplanesdk.CreateGovernanceRuleResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return governancerulescontrolplanesdk.CreateGovernanceRuleResponse{}, nil
}

func (f *fakeGovernanceRuleOCIClient) GetGovernanceRule(
	ctx context.Context,
	request governancerulescontrolplanesdk.GetGovernanceRuleRequest,
) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return governancerulescontrolplanesdk.GetGovernanceRuleResponse{}, nil
}

func (f *fakeGovernanceRuleOCIClient) ListGovernanceRules(
	ctx context.Context,
	request governancerulescontrolplanesdk.ListGovernanceRulesRequest,
) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return governancerulescontrolplanesdk.ListGovernanceRulesResponse{}, nil
}

func (f *fakeGovernanceRuleOCIClient) UpdateGovernanceRule(
	ctx context.Context,
	request governancerulescontrolplanesdk.UpdateGovernanceRuleRequest,
) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return governancerulescontrolplanesdk.UpdateGovernanceRuleResponse{}, nil
}

func (f *fakeGovernanceRuleOCIClient) DeleteGovernanceRule(
	ctx context.Context,
	request governancerulescontrolplanesdk.DeleteGovernanceRuleRequest,
) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return governancerulescontrolplanesdk.DeleteGovernanceRuleResponse{}, nil
}

func (f *fakeGovernanceRuleOCIClient) GetWorkRequest(
	ctx context.Context,
	request governancerulescontrolplanesdk.GetWorkRequestRequest,
) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return governancerulescontrolplanesdk.GetWorkRequestResponse{}, nil
}

func testGovernanceRuleClient(fake *fakeGovernanceRuleOCIClient) GovernanceRuleServiceClient {
	manager := &GovernanceRuleServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newGovernanceRuleRuntimeHooksWithOCIClient(fake)
	applyGovernanceRuleRuntimeHooks(&hooks, fake, nil)
	delegate := defaultGovernanceRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*governancerulescontrolplanev1beta1.GovernanceRule](
			buildGovernanceRuleGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapGovernanceRuleGeneratedClient(hooks, delegate)
}

func newGovernanceRuleRuntimeHooksWithOCIClient(client governanceRuleOCIClient) GovernanceRuleRuntimeHooks {
	return GovernanceRuleRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*governancerulescontrolplanev1beta1.GovernanceRule]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*governancerulescontrolplanev1beta1.GovernanceRule]{},
		StatusHooks:     generatedruntime.StatusHooks[*governancerulescontrolplanev1beta1.GovernanceRule]{},
		ParityHooks:     generatedruntime.ParityHooks[*governancerulescontrolplanev1beta1.GovernanceRule]{},
		Async:           generatedruntime.AsyncHooks[*governancerulescontrolplanev1beta1.GovernanceRule]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*governancerulescontrolplanev1beta1.GovernanceRule]{},
		Create: runtimeOperationHooks[governancerulescontrolplanesdk.CreateGovernanceRuleRequest, governancerulescontrolplanesdk.CreateGovernanceRuleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateGovernanceRuleDetails", RequestName: "CreateGovernanceRuleDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.CreateGovernanceRuleRequest) (governancerulescontrolplanesdk.CreateGovernanceRuleResponse, error) {
				return client.CreateGovernanceRule(ctx, request)
			},
		},
		Get: runtimeOperationHooks[governancerulescontrolplanesdk.GetGovernanceRuleRequest, governancerulescontrolplanesdk.GetGovernanceRuleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "GovernanceRuleId", RequestName: "governanceRuleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
				return client.GetGovernanceRule(ctx, request)
			},
		},
		List: runtimeOperationHooks[governancerulescontrolplanesdk.ListGovernanceRulesRequest, governancerulescontrolplanesdk.ListGovernanceRulesResponse]{
			Fields: governanceRuleListFields(),
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.ListGovernanceRulesRequest) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error) {
				return client.ListGovernanceRules(ctx, request)
			},
		},
		Update: runtimeOperationHooks[governancerulescontrolplanesdk.UpdateGovernanceRuleRequest, governancerulescontrolplanesdk.UpdateGovernanceRuleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "GovernanceRuleId", RequestName: "governanceRuleId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateGovernanceRuleDetails", RequestName: "UpdateGovernanceRuleDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.UpdateGovernanceRuleRequest) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error) {
				return client.UpdateGovernanceRule(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[governancerulescontrolplanesdk.DeleteGovernanceRuleRequest, governancerulescontrolplanesdk.DeleteGovernanceRuleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "GovernanceRuleId", RequestName: "governanceRuleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
				return client.DeleteGovernanceRule(ctx, request)
			},
		},
		WrapGeneratedClient: []func(GovernanceRuleServiceClient) GovernanceRuleServiceClient{},
	}
}

func makeGovernanceRuleResource() *governancerulescontrolplanev1beta1.GovernanceRule {
	return &governancerulescontrolplanev1beta1.GovernanceRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rule-alpha",
			Namespace: "default",
			UID:       types.UID("governance-rule-uid"),
		},
		Spec: governancerulescontrolplanev1beta1.GovernanceRuleSpec{
			CompartmentId:  testGovernanceRuleCompID,
			DisplayName:    "rule-alpha",
			Type:           string(governancerulescontrolplanesdk.GovernanceRuleTypeQuota),
			CreationOption: string(governancerulescontrolplanesdk.CreationOptionTemplate),
			Template: governancerulescontrolplanev1beta1.GovernanceRuleTemplate{
				DisplayName: "quota-alpha",
				Description: "quota template",
				Statements:  []string{"set compute-core quota standard-e4-core-count to 10 in tenancy"},
			},
			Description:  "governance quota rule",
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKGovernanceRule(
	id string,
	compartmentID string,
	displayName string,
	description string,
	state governancerulescontrolplanesdk.GovernanceRuleLifecycleStateEnum,
	template governancerulescontrolplanesdk.Template,
) governancerulescontrolplanesdk.GovernanceRule {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return governancerulescontrolplanesdk.GovernanceRule{
		Id:                common.String(id),
		CompartmentId:     common.String(compartmentID),
		DisplayName:       common.String(displayName),
		Type:              governancerulescontrolplanesdk.GovernanceRuleTypeQuota,
		CreationOption:    governancerulescontrolplanesdk.CreationOptionTemplate,
		Template:          template,
		TimeCreated:       &created,
		TimeUpdated:       &created,
		LifecycleState:    state,
		FreeformTags:      map[string]string{"env": "dev"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		Description:       common.String(description),
		RelatedResourceId: nil,
	}
}

func makeSDKGovernanceRuleSummary(
	id string,
	compartmentID string,
	displayName string,
	state governancerulescontrolplanesdk.GovernanceRuleLifecycleStateEnum,
) governancerulescontrolplanesdk.GovernanceRuleSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return governancerulescontrolplanesdk.GovernanceRuleSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		Type:           governancerulescontrolplanesdk.GovernanceRuleTypeQuota,
		CreationOption: governancerulescontrolplanesdk.CreationOptionTemplate,
		TimeCreated:    &created,
		TimeUpdated:    &created,
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKQuotaTemplate(displayName string, description string, statements []string) governancerulescontrolplanesdk.QuotaTemplate {
	return governancerulescontrolplanesdk.QuotaTemplate{
		DisplayName: common.String(displayName),
		Description: common.String(description),
		Statements:  append([]string(nil), statements...),
	}
}

func makeGovernanceRuleWorkRequest(
	id string,
	status governancerulescontrolplanesdk.OperationStatusEnum,
	operation governancerulescontrolplanesdk.OperationTypeEnum,
	action governancerulescontrolplanesdk.ActionTypeEnum,
) governancerulescontrolplanesdk.WorkRequest {
	percent := float32(50)
	return governancerulescontrolplanesdk.WorkRequest{
		Id:              common.String(id),
		Status:          status,
		OperationType:   operation,
		CompartmentId:   common.String(testGovernanceRuleCompID),
		PercentComplete: common.Float32(percent),
		Resources: []governancerulescontrolplanesdk.WorkRequestResource{
			{
				EntityType: common.String("GovernanceRule"),
				ActionType: action,
				Identifier: common.String(testGovernanceRuleID),
			},
		},
	}
}

func TestGovernanceRuleRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := governanceRuleRuntimeSemantics()
	if got == nil {
		t.Fatal("governanceRuleRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	requireGovernanceRuleStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "type", "id"})
	requireGovernanceRuleStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "description", "template", "relatedResourceId", "freeformTags", "definedTags"})
	requireGovernanceRuleStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "type", "creationOption"})
}

func TestGovernanceRuleCreateDetailsBuildsTagTemplateAndPreservesFalse(t *testing.T) {
	t.Parallel()

	spec := makeGovernanceRuleResource().Spec
	spec.Type = string(governancerulescontrolplanesdk.GovernanceRuleTypeTag)
	spec.Template = governancerulescontrolplanev1beta1.GovernanceRuleTemplate{
		Name:        "tag-namespace",
		Description: "tag namespace",
		Tags: []governancerulescontrolplanev1beta1.GovernanceRuleTemplateTag{
			{
				Name:           "CostCenter",
				IsCostTracking: false,
				Validator: governancerulescontrolplanev1beta1.GovernanceRuleTemplateTagValidator{
					ValidatorType: string(governancerulescontrolplanesdk.BaseTagDefinitionValidatorValidatorTypeEnumvalue),
					Values:        []string{"42", "84"},
				},
			},
		},
		TagDefaults: []governancerulescontrolplanev1beta1.GovernanceRuleTemplateTagDefault{
			{TagName: "CostCenter", Value: "42", IsRequired: false},
		},
	}

	body, err := governanceRuleCreateDetailsFromSpec(spec)
	if err != nil {
		t.Fatalf("governanceRuleCreateDetailsFromSpec() error = %v", err)
	}
	tagTemplate, ok := body.Template.(governancerulescontrolplanesdk.TagTemplate)
	if !ok {
		t.Fatalf("Template = %T, want TagTemplate", body.Template)
	}
	if len(tagTemplate.Tags) != 1 {
		t.Fatalf("tag count = %d, want 1", len(tagTemplate.Tags))
	}
	requireGovernanceRuleBoolPtr(t, "tag isCostTracking", tagTemplate.Tags[0].IsCostTracking, false)
	validator, ok := tagTemplate.Tags[0].Validator.(governancerulescontrolplanesdk.EnumTagDefinitionValidator)
	if !ok {
		t.Fatalf("tag validator = %T, want EnumTagDefinitionValidator", tagTemplate.Tags[0].Validator)
	}
	if !reflect.DeepEqual(validator.Values, []string{"42", "84"}) {
		t.Fatalf("validator values = %#v, want [42 84]", validator.Values)
	}
	requireGovernanceRuleBoolPtr(t, "tag default isRequired", tagTemplate.TagDefaults[0].IsRequired, false)
}

func TestGovernanceRuleCreateDetailsAllowsCloneFromRelatedResourceID(t *testing.T) {
	t.Parallel()

	spec := makeGovernanceRuleResource().Spec
	spec.CreationOption = string(governancerulescontrolplanesdk.CreationOptionClone)
	spec.RelatedResourceId = testGovernanceRuleOther
	spec.Template = governancerulescontrolplanev1beta1.GovernanceRuleTemplate{}

	body, err := governanceRuleCreateDetailsFromSpec(spec)
	if err != nil {
		t.Fatalf("governanceRuleCreateDetailsFromSpec() error = %v", err)
	}
	if body.CreationOption != governancerulescontrolplanesdk.CreationOptionClone {
		t.Fatalf("create creationOption = %q, want CLONE", body.CreationOption)
	}
	requireGovernanceRuleStringPtr(t, "create relatedResourceId", body.RelatedResourceId, testGovernanceRuleOther)
	if body.Template != nil {
		t.Fatalf("create template = %T, want nil for relatedResourceId-only CLONE", body.Template)
	}
}

func TestGovernanceRuleServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	template := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)
	var listRequest governancerulescontrolplanesdk.ListGovernanceRulesRequest
	var createRequest governancerulescontrolplanesdk.CreateGovernanceRuleRequest
	var getRequest governancerulescontrolplanesdk.GetGovernanceRuleRequest
	var workRequestRequest governancerulescontrolplanesdk.GetWorkRequestRequest
	listCalls := 0
	createCalls := 0
	getCalls := 0
	workRequestCalls := 0

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		listFn: func(_ context.Context, request governancerulescontrolplanesdk.ListGovernanceRulesRequest) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error) {
			listCalls++
			listRequest = request
			return governancerulescontrolplanesdk.ListGovernanceRulesResponse{
				GovernanceRuleCollection: governancerulescontrolplanesdk.GovernanceRuleCollection{},
				OpcRequestId:             common.String("opc-list"),
			}, nil
		},
		createFn: func(_ context.Context, request governancerulescontrolplanesdk.CreateGovernanceRuleRequest) (governancerulescontrolplanesdk.CreateGovernanceRuleResponse, error) {
			createCalls++
			createRequest = request
			return governancerulescontrolplanesdk.CreateGovernanceRuleResponse{
				GovernanceRule: makeSDKGovernanceRule(
					testGovernanceRuleID,
					testGovernanceRuleCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
					template,
				),
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			workRequestRequest = request
			return governancerulescontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: makeGovernanceRuleWorkRequest(
					"wr-create",
					governancerulescontrolplanesdk.OperationStatusSucceeded,
					governancerulescontrolplanesdk.OperationTypeCreateGovernanceRule,
					governancerulescontrolplanesdk.ActionTypeCreated,
				),
			}, nil
		},
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			getRequest = request
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{
				GovernanceRule: makeSDKGovernanceRule(
					testGovernanceRuleID,
					testGovernanceRuleCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
					template,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 1 || createCalls != 1 || workRequestCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/create/workrequest/get = %d/%d/%d/%d, want 1/1/1/1", listCalls, createCalls, workRequestCalls, getCalls)
	}
	assertGovernanceRuleCreateRequest(t, resource, listRequest, createRequest)
	requireGovernanceRuleStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	requireGovernanceRuleStringPtr(t, "workRequestId", workRequestRequest.WorkRequestId, "wr-create")
	requireGovernanceRuleStringPtr(t, "get governanceRuleId", getRequest.GovernanceRuleId, testGovernanceRuleID)
	assertGovernanceRuleStatus(t, resource, testGovernanceRuleID, shared.Active)
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestGovernanceRuleServiceClientBindsFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	template := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)
	listCalls := 0
	getCalls := 0
	var pages []string

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		listFn: func(_ context.Context, request governancerulescontrolplanesdk.ListGovernanceRulesRequest) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error) {
			listCalls++
			pages = append(pages, governanceRuleStringValue(request.Page))
			if listCalls == 1 {
				return governancerulescontrolplanesdk.ListGovernanceRulesResponse{
					GovernanceRuleCollection: governancerulescontrolplanesdk.GovernanceRuleCollection{
						Items: []governancerulescontrolplanesdk.GovernanceRuleSummary{
							makeSDKGovernanceRuleSummary(
								testGovernanceRuleOther,
								testGovernanceRuleCompID,
								"other-rule",
								governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return governancerulescontrolplanesdk.ListGovernanceRulesResponse{
				GovernanceRuleCollection: governancerulescontrolplanesdk.GovernanceRuleCollection{
					Items: []governancerulescontrolplanesdk.GovernanceRuleSummary{
						makeSDKGovernanceRuleSummary(
							testGovernanceRuleID,
							testGovernanceRuleCompID,
							resource.Spec.DisplayName,
							governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{
				GovernanceRule: makeSDKGovernanceRule(
					testGovernanceRuleID,
					testGovernanceRuleCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
					template,
				),
			}, nil
		},
		createFn: func(context.Context, governancerulescontrolplanesdk.CreateGovernanceRuleRequest) (governancerulescontrolplanesdk.CreateGovernanceRuleResponse, error) {
			t.Fatal("CreateGovernanceRule() called for existing governance rule")
			return governancerulescontrolplanesdk.CreateGovernanceRuleResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 2 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 2/1", listCalls, getCalls)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("list pages = %#v, want %#v", pages, want)
	}
	assertGovernanceRuleStatus(t, resource, testGovernanceRuleID, shared.Active)
}

func TestGovernanceRuleServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	template := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)
	getCalls := 0
	updateCalls := 0

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{
				GovernanceRule: makeSDKGovernanceRule(
					testGovernanceRuleID,
					testGovernanceRuleCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
					template,
				),
			}, nil
		},
		updateFn: func(context.Context, governancerulescontrolplanesdk.UpdateGovernanceRuleRequest) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error) {
			updateCalls++
			t.Fatal("UpdateGovernanceRule() called when observed state matches")
			return governancerulescontrolplanesdk.UpdateGovernanceRuleResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts get/update = %d/%d, want 1/0", getCalls, updateCalls)
	}
	requireGovernanceRuleLastCondition(t, resource, shared.Active)
}

func TestGovernanceRuleServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	resource.Spec.DisplayName = "rule-beta"
	resource.Spec.Description = "updated governance quota rule"
	resource.Spec.Template.Statements = []string{"set compute-core quota standard-e4-core-count to 20 in tenancy"}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0
	var updateRequest governancerulescontrolplanesdk.UpdateGovernanceRuleRequest
	var workRequestRequest governancerulescontrolplanesdk.GetWorkRequestRequest
	oldTemplate := makeSDKQuotaTemplate("quota-alpha", "quota template", []string{"set compute-core quota standard-e4-core-count to 10 in tenancy"})
	updatedTemplate := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			currentTemplate := oldTemplate
			displayName := "rule-alpha"
			description := "governance quota rule"
			freeformTags := map[string]string{"env": "dev"}
			if getCalls > 1 {
				currentTemplate = updatedTemplate
				displayName = resource.Spec.DisplayName
				description = resource.Spec.Description
				freeformTags = map[string]string{"env": "prod"}
			}
			current := makeSDKGovernanceRule(
				testGovernanceRuleID,
				testGovernanceRuleCompID,
				displayName,
				description,
				governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
				currentTemplate,
			)
			current.FreeformTags = freeformTags
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{GovernanceRule: current}, nil
		},
		updateFn: func(_ context.Context, request governancerulescontrolplanesdk.UpdateGovernanceRuleRequest) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error) {
			updateCalls++
			updateRequest = request
			return governancerulescontrolplanesdk.UpdateGovernanceRuleResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			workRequestRequest = request
			return governancerulescontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: makeGovernanceRuleWorkRequest(
					"wr-update",
					governancerulescontrolplanesdk.OperationStatusSucceeded,
					governancerulescontrolplanesdk.OperationTypeUpdateGovernanceRule,
					governancerulescontrolplanesdk.ActionTypeUpdated,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 2 || updateCalls != 1 || workRequestCalls != 1 {
		t.Fatalf("call counts get/update/workrequest = %d/%d/%d, want 2/1/1", getCalls, updateCalls, workRequestCalls)
	}
	assertGovernanceRuleUpdateRequest(t, resource, updateRequest)
	requireGovernanceRuleStringPtr(t, "workRequestId", workRequestRequest.WorkRequestId, "wr-update")
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
	requireGovernanceRuleLastCondition(t, resource, shared.Active)
}

func TestGovernanceRuleServiceClientResumesPendingUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   "wr-update-pending",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(context.Context, governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			t.Fatal("GetGovernanceRule() called before pending update work request completes")
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{}, nil
		},
		updateFn: func(context.Context, governancerulescontrolplanesdk.UpdateGovernanceRuleRequest) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error) {
			updateCalls++
			t.Fatal("UpdateGovernanceRule() called while update work request is pending")
			return governancerulescontrolplanesdk.UpdateGovernanceRuleResponse{}, nil
		},
		workRequestFn: func(_ context.Context, request governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireGovernanceRuleStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update-pending")
			return governancerulescontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: makeGovernanceRuleWorkRequest(
					"wr-update-pending",
					governancerulescontrolplanesdk.OperationStatusInProgress,
					governancerulescontrolplanesdk.OperationTypeUpdateGovernanceRule,
					governancerulescontrolplanesdk.ActionTypeUpdated,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if getCalls != 0 || updateCalls != 0 || workRequestCalls != 1 {
		t.Fatalf("call counts get/update/workrequest = %d/%d/%d, want 0/0/1", getCalls, updateCalls, workRequestCalls)
	}
	assertGovernanceRuleAsyncCurrent(t, resource, "wr-update-pending", shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
}

func TestGovernanceRuleServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	resource.Spec.CompartmentId = "ocid1.tenancy.oc1..changed"
	template := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)
	updateCalls := 0

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{
				GovernanceRule: makeSDKGovernanceRule(
					testGovernanceRuleID,
					testGovernanceRuleCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
					template,
				),
			}, nil
		},
		updateFn: func(context.Context, governancerulescontrolplanesdk.UpdateGovernanceRuleRequest) (governancerulescontrolplanesdk.UpdateGovernanceRuleResponse, error) {
			updateCalls++
			t.Fatal("UpdateGovernanceRule() called despite create-only drift")
			return governancerulescontrolplanesdk.UpdateGovernanceRuleResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "compartmentId") ||
		!strings.Contains(err.Error(), "require replacement") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement rejection", err)
	}
	requireGovernanceRuleLastCondition(t, resource, shared.Failed)
}

func TestGovernanceRuleDeleteWaitsOnPendingWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	template := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)
	getCalls := 0
	deleteCalls := 0
	workRequestCalls := 0

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{
				GovernanceRule: makeSDKGovernanceRule(
					testGovernanceRuleID,
					testGovernanceRuleCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
					template,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
			deleteCalls++
			requireGovernanceRuleStringPtr(t, "delete governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.DeleteGovernanceRuleResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireGovernanceRuleStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return governancerulescontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: makeGovernanceRuleWorkRequest(
					"wr-delete",
					governancerulescontrolplanesdk.OperationStatusInProgress,
					governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule,
					governancerulescontrolplanesdk.ActionTypeDeleted,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if getCalls != 1 || deleteCalls != 1 || workRequestCalls != 1 {
		t.Fatalf("call counts get/delete/workrequest = %d/%d/%d, want 1/1/1", getCalls, deleteCalls, workRequestCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	assertGovernanceRuleAsyncCurrent(t, resource, "wr-delete", shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
	requireGovernanceRuleLastCondition(t, resource, shared.Terminating)
}

func TestGovernanceRuleDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	template := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)
	getCalls := 0
	workRequestCalls := 0

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			if getCalls == 1 {
				return governancerulescontrolplanesdk.GetGovernanceRuleResponse{
					GovernanceRule: makeSDKGovernanceRule(
						testGovernanceRuleID,
						testGovernanceRuleCompID,
						resource.Spec.DisplayName,
						resource.Spec.Description,
						governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
						template,
					),
				}, nil
			}
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "governance rule deleted")
		},
		deleteFn: func(_ context.Context, request governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
			requireGovernanceRuleStringPtr(t, "delete governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.DeleteGovernanceRuleResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireGovernanceRuleStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return governancerulescontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: makeGovernanceRuleWorkRequest(
					"wr-delete",
					governancerulescontrolplanesdk.OperationStatusSucceeded,
					governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule,
					governancerulescontrolplanesdk.ActionTypeDeleted,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
	if getCalls != 2 || workRequestCalls != 1 {
		t.Fatalf("call counts get/workrequest = %d/%d, want 2/1", getCalls, workRequestCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestGovernanceRuleDeleteResumesPendingWorkRequestWithoutDuplicateDelete(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete-pending",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	getCalls := 0
	deleteCalls := 0
	workRequestCalls := 0

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(context.Context, governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			getCalls++
			t.Fatal("GetGovernanceRule() called before pending delete work request completes")
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{}, nil
		},
		deleteFn: func(context.Context, governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
			deleteCalls++
			t.Fatal("DeleteGovernanceRule() called while delete work request is pending")
			return governancerulescontrolplanesdk.DeleteGovernanceRuleResponse{}, nil
		},
		workRequestFn: func(_ context.Context, request governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireGovernanceRuleStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete-pending")
			return governancerulescontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: makeGovernanceRuleWorkRequest(
					"wr-delete-pending",
					governancerulescontrolplanesdk.OperationStatusInProgress,
					governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule,
					governancerulescontrolplanesdk.ActionTypeDeleted,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if getCalls != 0 || deleteCalls != 0 || workRequestCalls != 1 {
		t.Fatalf("call counts get/delete/workrequest = %d/%d/%d, want 0/0/1", getCalls, deleteCalls, workRequestCalls)
	}
	requireGovernanceRuleLastCondition(t, resource, shared.Terminating)
}

func TestGovernanceRuleDeleteSucceededWorkRequestRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	const workRequestID = "wr-delete"
	resource := makeGovernanceRulePendingDeleteResource(workRequestID)
	probe := newGovernanceRuleAuthShapedConfirmReadProbe(t, workRequestID)

	deleted, err := probe.client.Delete(context.Background(), resource)
	assertGovernanceRuleAuthShapedConfirmReadRejected(t, resource, deleted, err, probe)
}

func makeGovernanceRulePendingDeleteResource(workRequestID string) *governancerulescontrolplanev1beta1.GovernanceRule {
	resource := makeGovernanceRuleResource()
	resource.Finalizers = []string{"osok-finalizer"}
	resource.Status.Id = testGovernanceRuleID
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	return resource
}

type governanceRuleAuthShapedConfirmReadProbe struct {
	client           GovernanceRuleServiceClient
	workRequestID    string
	getCalls         int
	deleteCalls      int
	workRequestCalls int
}

func newGovernanceRuleAuthShapedConfirmReadProbe(
	t *testing.T,
	workRequestID string,
) *governanceRuleAuthShapedConfirmReadProbe {
	t.Helper()
	probe := &governanceRuleAuthShapedConfirmReadProbe{workRequestID: workRequestID}
	probe.client = testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn:         probe.authShapedGetNotFound(t),
		deleteFn:      probe.unexpectedDelete(t),
		workRequestFn: probe.succeededDeleteWorkRequest(t),
	})
	return probe
}

func (p *governanceRuleAuthShapedConfirmReadProbe) authShapedGetNotFound(
	t *testing.T,
) func(context.Context, governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
	t.Helper()
	return func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
		p.getCalls++
		requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
		return governancerulescontrolplanesdk.GetGovernanceRuleResponse{}, errortest.NewServiceError(
			404,
			errorutil.NotAuthorizedOrNotFound,
			"not authorized or not found",
		)
	}
}

func (p *governanceRuleAuthShapedConfirmReadProbe) unexpectedDelete(
	t *testing.T,
) func(context.Context, governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
	t.Helper()
	return func(context.Context, governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
		p.deleteCalls++
		t.Fatal("DeleteGovernanceRule() called after ambiguous succeeded delete work request confirmation")
		return governancerulescontrolplanesdk.DeleteGovernanceRuleResponse{}, nil
	}
}

func (p *governanceRuleAuthShapedConfirmReadProbe) succeededDeleteWorkRequest(
	t *testing.T,
) func(context.Context, governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, request governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error) {
		p.workRequestCalls++
		requireGovernanceRuleStringPtr(t, "workRequestId", request.WorkRequestId, p.workRequestID)
		return governancerulescontrolplanesdk.GetWorkRequestResponse{
			WorkRequest: makeGovernanceRuleWorkRequest(
				p.workRequestID,
				governancerulescontrolplanesdk.OperationStatusSucceeded,
				governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule,
				governancerulescontrolplanesdk.ActionTypeDeleted,
			),
		}, nil
	}
}

func assertGovernanceRuleAuthShapedConfirmReadRejected(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	deleted bool,
	err error,
	probe *governanceRuleAuthShapedConfirmReadProbe,
) {
	t.Helper()
	requireGovernanceRuleAmbiguousDeleteError(t, deleted, err)
	requireGovernanceRuleConfirmReadCallCounts(t, probe)
	requireGovernanceRuleFinalizerRetained(t, resource)
	requireGovernanceRuleUnconfirmedDeleteStatus(t, resource, probe.workRequestID)
}

func requireGovernanceRuleAmbiguousDeleteError(t *testing.T, deleted bool, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 rejection")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
}

func requireGovernanceRuleConfirmReadCallCounts(
	t *testing.T,
	probe *governanceRuleAuthShapedConfirmReadProbe,
) {
	t.Helper()
	if probe.getCalls != 1 || probe.deleteCalls != 0 || probe.workRequestCalls != 1 {
		t.Fatalf(
			"call counts get/delete/workrequest = %d/%d/%d, want 1/0/1",
			probe.getCalls,
			probe.deleteCalls,
			probe.workRequestCalls,
		)
	}
}

func requireGovernanceRuleFinalizerRetained(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
) {
	t.Helper()
	if len(resource.Finalizers) != 1 {
		t.Fatalf("finalizers = %#v, want exactly one osok-finalizer", resource.Finalizers)
	}
	if resource.Finalizers[0] != "osok-finalizer" {
		t.Fatalf("finalizers = %#v, want retained osok-finalizer", resource.Finalizers)
	}
}

func requireGovernanceRuleUnconfirmedDeleteStatus(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	workRequestID string,
) {
	t.Helper()
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil for ambiguous not found", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
	assertGovernanceRuleAsyncCurrent(t, resource, workRequestID, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassFailed)
	requireGovernanceRuleLastCondition(t, resource, shared.Failed)
}

func TestGovernanceRuleDeleteKeepsFinalizerOnAmbiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeGovernanceRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testGovernanceRuleID)
	template := makeSDKQuotaTemplate(resource.Spec.Template.DisplayName, resource.Spec.Template.Description, resource.Spec.Template.Statements)

	client := testGovernanceRuleClient(&fakeGovernanceRuleOCIClient{
		getFn: func(_ context.Context, request governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error) {
			requireGovernanceRuleStringPtr(t, "get governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.GetGovernanceRuleResponse{
				GovernanceRule: makeSDKGovernanceRule(
					testGovernanceRuleID,
					testGovernanceRuleCompID,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive,
					template,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request governancerulescontrolplanesdk.DeleteGovernanceRuleRequest) (governancerulescontrolplanesdk.DeleteGovernanceRuleResponse, error) {
			requireGovernanceRuleStringPtr(t, "delete governanceRuleId", request.GovernanceRuleId, testGovernanceRuleID)
			return governancerulescontrolplanesdk.DeleteGovernanceRuleResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous not found")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil for ambiguous not found", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func assertGovernanceRuleCreateRequest(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	listRequest governancerulescontrolplanesdk.ListGovernanceRulesRequest,
	createRequest governancerulescontrolplanesdk.CreateGovernanceRuleRequest,
) {
	t.Helper()
	requireGovernanceRuleStringPtr(t, "list compartmentId", listRequest.CompartmentId, testGovernanceRuleCompID)
	requireGovernanceRuleStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	if listRequest.GovernanceRuleType != governancerulescontrolplanesdk.ListGovernanceRulesGovernanceRuleTypeQuota {
		t.Fatalf("list governanceRuleType = %q, want QUOTA", listRequest.GovernanceRuleType)
	}
	createDetails := createRequest.CreateGovernanceRuleDetails
	requireGovernanceRuleStringPtr(t, "create compartmentId", createDetails.CompartmentId, testGovernanceRuleCompID)
	requireGovernanceRuleStringPtr(t, "create displayName", createDetails.DisplayName, resource.Spec.DisplayName)
	requireGovernanceRuleStringPtr(t, "create description", createDetails.Description, resource.Spec.Description)
	if createDetails.Type != governancerulescontrolplanesdk.GovernanceRuleTypeQuota {
		t.Fatalf("create type = %q, want QUOTA", createDetails.Type)
	}
	if createDetails.CreationOption != governancerulescontrolplanesdk.CreationOptionTemplate {
		t.Fatalf("create creationOption = %q, want TEMPLATE", createDetails.CreationOption)
	}
	template, ok := createDetails.Template.(governancerulescontrolplanesdk.QuotaTemplate)
	if !ok {
		t.Fatalf("create template = %T, want QuotaTemplate", createDetails.Template)
	}
	requireGovernanceRuleStringPtr(t, "create template displayName", template.DisplayName, resource.Spec.Template.DisplayName)
	if !reflect.DeepEqual(template.Statements, resource.Spec.Template.Statements) {
		t.Fatalf("create template statements = %#v, want %#v", template.Statements, resource.Spec.Template.Statements)
	}
	if !reflect.DeepEqual(createDetails.FreeformTags, map[string]string{"env": "dev"}) {
		t.Fatalf("create freeformTags = %#v, want env=dev", createDetails.FreeformTags)
	}
}

func assertGovernanceRuleUpdateRequest(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	updateRequest governancerulescontrolplanesdk.UpdateGovernanceRuleRequest,
) {
	t.Helper()
	requireGovernanceRuleStringPtr(t, "update governanceRuleId", updateRequest.GovernanceRuleId, testGovernanceRuleID)
	requireGovernanceRuleStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	requireGovernanceRuleStringPtr(t, "update description", updateRequest.Description, resource.Spec.Description)
	if _, ok := updateRequest.Template.(governancerulescontrolplanesdk.QuotaTemplate); !ok {
		t.Fatalf("update template = %T, want QuotaTemplate", updateRequest.Template)
	}
	if !reflect.DeepEqual(updateRequest.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateRequest.FreeformTags)
	}
}

func assertGovernanceRuleStatus(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	wantID string,
	wantCondition shared.OSOKConditionType,
) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if resource.Status.Id != wantID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, wantID)
	}
	if resource.Status.LifecycleState != string(governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	requireGovernanceRuleLastCondition(t, resource, wantCondition)
}

func assertGovernanceRuleAsyncCurrent(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	wantWorkRequestID string,
	wantPhase shared.OSOKAsyncPhase,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status async current = nil, want %s %s", wantPhase, wantWorkRequestID)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status async current workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status async current phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status async current class = %q, want %q", current.NormalizedClass, wantClass)
	}
}

func requireGovernanceRuleLastCondition(
	t *testing.T,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions is empty, want last condition %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireGovernanceRuleStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireGovernanceRuleBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}

func requireGovernanceRuleStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
