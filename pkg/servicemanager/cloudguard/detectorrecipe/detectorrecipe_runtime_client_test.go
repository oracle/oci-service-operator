/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package detectorrecipe

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDetectorRecipeCompartmentID = "ocid1.compartment.oc1..detectorrecipe"
	testDetectorRecipeID            = "ocid1.detectorrecipe.oc1..detectorrecipe"
	testSourceDetectorRecipeID      = "ocid1.detectorrecipe.oc1..source"
	testDetectorRecipeDisplayName   = "detector-recipe"
	testDetectorRuleID              = "detector-rule-1"
)

type fakeDetectorRecipeOCIClient struct {
	createFunc func(context.Context, cloudguardsdk.CreateDetectorRecipeRequest) (cloudguardsdk.CreateDetectorRecipeResponse, error)
	getFunc    func(context.Context, cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error)
	listFunc   func(context.Context, cloudguardsdk.ListDetectorRecipesRequest) (cloudguardsdk.ListDetectorRecipesResponse, error)
	updateFunc func(context.Context, cloudguardsdk.UpdateDetectorRecipeRequest) (cloudguardsdk.UpdateDetectorRecipeResponse, error)
	deleteFunc func(context.Context, cloudguardsdk.DeleteDetectorRecipeRequest) (cloudguardsdk.DeleteDetectorRecipeResponse, error)

	createRequests []cloudguardsdk.CreateDetectorRecipeRequest
	getRequests    []cloudguardsdk.GetDetectorRecipeRequest
	listRequests   []cloudguardsdk.ListDetectorRecipesRequest
	updateRequests []cloudguardsdk.UpdateDetectorRecipeRequest
	deleteRequests []cloudguardsdk.DeleteDetectorRecipeRequest
}

func (f *fakeDetectorRecipeOCIClient) CreateDetectorRecipe(
	ctx context.Context,
	request cloudguardsdk.CreateDetectorRecipeRequest,
) (cloudguardsdk.CreateDetectorRecipeResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return cloudguardsdk.CreateDetectorRecipeResponse{}, nil
}

func (f *fakeDetectorRecipeOCIClient) GetDetectorRecipe(
	ctx context.Context,
	request cloudguardsdk.GetDetectorRecipeRequest,
) (cloudguardsdk.GetDetectorRecipeResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return cloudguardsdk.GetDetectorRecipeResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakeDetectorRecipeOCIClient) ListDetectorRecipes(
	ctx context.Context,
	request cloudguardsdk.ListDetectorRecipesRequest,
) (cloudguardsdk.ListDetectorRecipesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return cloudguardsdk.ListDetectorRecipesResponse{}, nil
}

func (f *fakeDetectorRecipeOCIClient) UpdateDetectorRecipe(
	ctx context.Context,
	request cloudguardsdk.UpdateDetectorRecipeRequest,
) (cloudguardsdk.UpdateDetectorRecipeResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return cloudguardsdk.UpdateDetectorRecipeResponse{}, nil
}

func (f *fakeDetectorRecipeOCIClient) DeleteDetectorRecipe(
	ctx context.Context,
	request cloudguardsdk.DeleteDetectorRecipeRequest,
) (cloudguardsdk.DeleteDetectorRecipeResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return cloudguardsdk.DeleteDetectorRecipeResponse{}, nil
}

func newDetectorRecipeRuntimeTestClient(fake *fakeDetectorRecipeOCIClient) DetectorRecipeServiceClient {
	if fake == nil {
		fake = &fakeDetectorRecipeOCIClient{}
	}

	hooks := newDetectorRecipeDefaultRuntimeHooks(cloudguardsdk.CloudGuardClient{})
	hooks.Create.Call = func(ctx context.Context, request cloudguardsdk.CreateDetectorRecipeRequest) (cloudguardsdk.CreateDetectorRecipeResponse, error) {
		return fake.CreateDetectorRecipe(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		return fake.GetDetectorRecipe(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request cloudguardsdk.ListDetectorRecipesRequest) (cloudguardsdk.ListDetectorRecipesResponse, error) {
		return fake.ListDetectorRecipes(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request cloudguardsdk.UpdateDetectorRecipeRequest) (cloudguardsdk.UpdateDetectorRecipeResponse, error) {
		return fake.UpdateDetectorRecipe(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request cloudguardsdk.DeleteDetectorRecipeRequest) (cloudguardsdk.DeleteDetectorRecipeResponse, error) {
		return fake.DeleteDetectorRecipe(ctx, request)
	}
	applyDetectorRecipeRuntimeHooks(&hooks)

	config := buildDetectorRecipeGeneratedRuntimeConfig(&DetectorRecipeServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}, hooks)
	delegate := defaultDetectorRecipeServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.DetectorRecipe](config),
	}
	return wrapDetectorRecipeGeneratedClient(hooks, delegate)
}

func newDetectorRecipeRuntimeTestResource() *cloudguardv1beta1.DetectorRecipe {
	return &cloudguardv1beta1.DetectorRecipe{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDetectorRecipeDisplayName,
			Namespace: "default",
			UID:       "detector-recipe-uid",
		},
		Spec: cloudguardv1beta1.DetectorRecipeSpec{
			CompartmentId:          testDetectorRecipeCompartmentID,
			DisplayName:            testDetectorRecipeDisplayName,
			Description:            "runtime detector recipe",
			Detector:               string(cloudguardsdk.DetectorEnumIaasActivityDetector),
			SourceDetectorRecipeId: testSourceDetectorRecipeID,
			DetectorRules: []cloudguardv1beta1.DetectorRecipeDetectorRuleFields{
				{
					DetectorRuleId: testDetectorRuleID,
					Details: cloudguardv1beta1.DetectorRecipeDetectorRuleDetails{
						IsEnabled: false,
						RiskLevel: string(cloudguardsdk.RiskLevelHigh),
						Labels:    []string{"security"},
					},
				},
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func newDetectorRecipeRuntimeTestResourceWithoutRules() *cloudguardv1beta1.DetectorRecipe {
	resource := newDetectorRecipeRuntimeTestResource()
	resource.Spec.DetectorRules = nil
	return resource
}

func detectorRecipeFromSpec(
	t *testing.T,
	id string,
	spec cloudguardv1beta1.DetectorRecipeSpec,
	state cloudguardsdk.LifecycleStateEnum,
) cloudguardsdk.DetectorRecipe {
	t.Helper()

	return cloudguardsdk.DetectorRecipe{
		Id:                     common.String(id),
		CompartmentId:          common.String(spec.CompartmentId),
		DisplayName:            common.String(spec.DisplayName),
		Description:            common.String(spec.Description),
		Detector:               cloudguardsdk.DetectorEnumEnum(spec.Detector),
		Owner:                  cloudguardsdk.OwnerTypeCustomer,
		SourceDetectorRecipeId: common.String(spec.SourceDetectorRecipeId),
		DetectorRecipeType:     cloudguardsdk.DetectorRecipeEnumStandard,
		DetectorRules:          detectorRecipeSDKRulesFromSpec(t, spec.DetectorRules),
		LifecycleState:         state,
		FreeformTags:           cloneStringMap(spec.FreeformTags),
		DefinedTags:            detectorRecipeDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:             map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": true}},
	}
}

func detectorRecipeSummaryFromSpec(
	t *testing.T,
	id string,
	spec cloudguardv1beta1.DetectorRecipeSpec,
	state cloudguardsdk.LifecycleStateEnum,
) cloudguardsdk.DetectorRecipeSummary {
	t.Helper()

	return cloudguardsdk.DetectorRecipeSummary{
		Id:                     common.String(id),
		CompartmentId:          common.String(spec.CompartmentId),
		DisplayName:            common.String(spec.DisplayName),
		Description:            common.String(spec.Description),
		Detector:               cloudguardsdk.DetectorEnumEnum(spec.Detector),
		Owner:                  cloudguardsdk.OwnerTypeCustomer,
		SourceDetectorRecipeId: common.String(spec.SourceDetectorRecipeId),
		DetectorRecipeType:     cloudguardsdk.DetectorRecipeEnumStandard,
		DetectorRules:          detectorRecipeSDKRulesFromSpec(t, spec.DetectorRules),
		LifecycleState:         state,
		FreeformTags:           cloneStringMap(spec.FreeformTags),
		DefinedTags:            detectorRecipeDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:             map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": true}},
	}
}

func detectorRecipeSDKRulesFromSpec(
	t *testing.T,
	rules []cloudguardv1beta1.DetectorRecipeDetectorRuleFields,
) []cloudguardsdk.DetectorRecipeDetectorRule {
	t.Helper()
	if rules == nil {
		return nil
	}
	payload, err := json.Marshal(rules)
	if err != nil {
		t.Fatalf("marshal detector rules: %v", err)
	}
	var converted []cloudguardsdk.DetectorRecipeDetectorRule
	if err := json.Unmarshal(payload, &converted); err != nil {
		t.Fatalf("convert detector rules: %v", err)
	}
	return converted
}

func requireDetectorRecipeRuntimeHooksConfigured(t *testing.T, hooks DetectorRecipeRuntimeHooks) {
	t.Helper()

	wantMatchFields := []string{"compartmentId", "displayName", "detector", "sourceDetectorRecipeId", "id"}
	checks := []struct {
		name string
		ok   bool
	}{
		{name: "reviewed semantics", ok: hooks.Semantics != nil},
		{name: "stable bind fields", ok: hooks.Semantics != nil &&
			hooks.Semantics.List != nil && reflect.DeepEqual(hooks.Semantics.List.MatchFields, wantMatchFields)},
		{name: "create body builder", ok: hooks.BuildCreateBody != nil},
		{name: "update body builder", ok: hooks.BuildUpdateBody != nil},
		{name: "pre-create guard", ok: hooks.Identity.GuardExistingBeforeCreate != nil},
		{name: "paginated list read", ok: hooks.Read.List != nil},
		{name: "tag-safe status projection", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "conservative delete errors", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "delete confirmation guard", ok: len(hooks.WrapGeneratedClient) != 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("%s not configured", check.name)
		}
	}
}

func assertDetectorRecipeCreateRequest(
	t *testing.T,
	request cloudguardsdk.CreateDetectorRecipeRequest,
	spec cloudguardv1beta1.DetectorRecipeSpec,
) {
	t.Helper()

	requireStringPtr(t, "CreateDetectorRecipeRequest.CompartmentId", request.CompartmentId, spec.CompartmentId)
	requireStringPtr(t, "CreateDetectorRecipeRequest.DisplayName", request.DisplayName, spec.DisplayName)
	requireStringPtr(t, "CreateDetectorRecipeRequest.Description", request.Description, spec.Description)
	if got := string(request.Detector); got != spec.Detector {
		t.Fatalf("CreateDetectorRecipeRequest.Detector = %q, want %q", got, spec.Detector)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateDetectorRecipeRequest.OpcRetryToken is empty, want deterministic retry token")
	}
	assertDetectorRecipeRequestRuleDisablesRule(t, request.DetectorRules)
}

func assertDetectorRecipeCreatedStatus(t *testing.T, resource *cloudguardv1beta1.DetectorRecipe) {
	t.Helper()

	if got := resource.Status.Id; got != testDetectorRecipeID {
		t.Fatalf("status.id = %q, want %q", got, testDetectorRecipeID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDetectorRecipeID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDetectorRecipeID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := resource.Status.LifecycleState; got != string(cloudguardsdk.LifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.SystemTags["orcl-cloud"]["free-tier-retained"]; got != "true" {
		t.Fatalf("status.systemTags.orcl-cloud.free-tier-retained = %q, want true", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil for ACTIVE", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDetectorRecipeRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newDetectorRecipeDefaultRuntimeHooks(cloudguardsdk.CloudGuardClient{})
	applyDetectorRecipeRuntimeHooks(&hooks)

	requireDetectorRecipeRuntimeHooksConfigured(t, hooks)
}

func TestDetectorRecipeCreateOrUpdateCreatesRecipeAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResource()
	fake := &fakeDetectorRecipeOCIClient{}
	fake.createFunc = func(_ context.Context, request cloudguardsdk.CreateDetectorRecipeRequest) (cloudguardsdk.CreateDetectorRecipeResponse, error) {
		assertDetectorRecipeCreateRequest(t, request, resource.Spec)
		return cloudguardsdk.CreateDetectorRecipeResponse{
			OpcRequestId:   common.String("opc-create-1"),
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateCreating),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		requireStringPtr(t, "GetDetectorRecipeRequest.DetectorRecipeId", request.DetectorRecipeId, testDetectorRecipeID)
		return cloudguardsdk.GetDetectorRecipeResponse{
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE follow-up")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateDetectorRecipe() calls = %d, want 1", len(fake.createRequests))
	}
	assertDetectorRecipeCreatedStatus(t, resource)
}

func TestDetectorRecipeCreateOrUpdateBindsExistingRecipeFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResourceWithoutRules()
	fake := &fakeDetectorRecipeOCIClient{}
	fake.listFunc = func(_ context.Context, request cloudguardsdk.ListDetectorRecipesRequest) (cloudguardsdk.ListDetectorRecipesResponse, error) {
		requireStringPtr(t, "ListDetectorRecipesRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		requireStringPtr(t, "ListDetectorRecipesRequest.DisplayName", request.DisplayName, resource.Spec.DisplayName)
		if request.LifecycleState != cloudguardsdk.ListDetectorRecipesLifecycleStateActive {
			return cloudguardsdk.ListDetectorRecipesResponse{}, nil
		}
		switch page := stringValue(request.Page); page {
		case "":
			otherSpec := resource.Spec
			otherSpec.DisplayName = "other-recipe"
			return cloudguardsdk.ListDetectorRecipesResponse{
				DetectorRecipeCollection: cloudguardsdk.DetectorRecipeCollection{
					Items: []cloudguardsdk.DetectorRecipeSummary{
						detectorRecipeSummaryFromSpec(t, "ocid1.detectorrecipe.oc1..other", otherSpec, cloudguardsdk.LifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return cloudguardsdk.ListDetectorRecipesResponse{
				DetectorRecipeCollection: cloudguardsdk.DetectorRecipeCollection{
					Items: []cloudguardsdk.DetectorRecipeSummary{
						detectorRecipeSummaryFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("ListDetectorRecipesRequest.Page = %q, want empty or page-2", page)
			return cloudguardsdk.ListDetectorRecipesResponse{}, nil
		}
	}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		requireStringPtr(t, "GetDetectorRecipeRequest.DetectorRecipeId", request.DetectorRecipeId, testDetectorRecipeID)
		return cloudguardsdk.GetDetectorRecipeResponse{
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.createFunc = func(context.Context, cloudguardsdk.CreateDetectorRecipeRequest) (cloudguardsdk.CreateDetectorRecipeResponse, error) {
		t.Fatal("CreateDetectorRecipe() called; want bind to existing recipe")
		return cloudguardsdk.CreateDetectorRecipeResponse{}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateDetectorRecipe() calls = %d, want 0", len(fake.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDetectorRecipeID {
		t.Fatalf("status.status.ocid = %q, want bound recipe ID", got)
	}
	if !detectorRecipeSawListPage(fake.listRequests, cloudguardsdk.ListDetectorRecipesLifecycleStateActive, "page-2") {
		t.Fatalf("ListDetectorRecipes() requests = %#v, want ACTIVE page-2 lookup", fake.listRequests)
	}
}

func TestDetectorRecipeCreateOrUpdateSkipsUpdateWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResourceWithoutRules()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDetectorRecipeID)
	fake := &fakeDetectorRecipeOCIClient{}
	fake.getFunc = func(context.Context, cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		return cloudguardsdk.GetDetectorRecipeResponse{
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, cloudguardsdk.UpdateDetectorRecipeRequest) (cloudguardsdk.UpdateDetectorRecipeResponse, error) {
		t.Fatal("UpdateDetectorRecipe() called; want no-op reconcile")
		return cloudguardsdk.UpdateDetectorRecipeResponse{}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateDetectorRecipe() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestDetectorRecipeCreateOrUpdateUpdatesMutableFieldsAndClearsTags(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResource()
	resource.Spec.DisplayName = "detector-recipe-updated"
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Status.OsokStatus.Ocid = shared.OCID(testDetectorRecipeID)

	currentSpec := newDetectorRecipeRuntimeTestResource().Spec
	currentSpec.DisplayName = testDetectorRecipeDisplayName
	currentSpec.FreeformTags = map[string]string{"env": "old"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "41"}}
	currentSpec.DetectorRules[0].Details.IsEnabled = true

	fake := &fakeDetectorRecipeOCIClient{}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		requireStringPtr(t, "GetDetectorRecipeRequest.DetectorRecipeId", request.DetectorRecipeId, testDetectorRecipeID)
		if len(fake.updateRequests) == 0 {
			return cloudguardsdk.GetDetectorRecipeResponse{
				DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, currentSpec, cloudguardsdk.LifecycleStateActive),
			}, nil
		}
		return cloudguardsdk.GetDetectorRecipeResponse{
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(_ context.Context, request cloudguardsdk.UpdateDetectorRecipeRequest) (cloudguardsdk.UpdateDetectorRecipeResponse, error) {
		assertDetectorRecipeMutableUpdateRequest(t, request, resource.Spec)
		return cloudguardsdk.UpdateDetectorRecipeResponse{
			OpcRequestId:   common.String("opc-update-1"),
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateUpdating),
		}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateDetectorRecipe() calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want updated displayName", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestDetectorRecipeCreateOrUpdateClearsDescription(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResourceWithoutRules()
	resource.Spec.Description = ""
	resource.Status.OsokStatus.Ocid = shared.OCID(testDetectorRecipeID)

	currentSpec := resource.Spec
	currentSpec.Description = "runtime detector recipe"
	fake := &fakeDetectorRecipeOCIClient{}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		requireStringPtr(t, "GetDetectorRecipeRequest.DetectorRecipeId", request.DetectorRecipeId, testDetectorRecipeID)
		if len(fake.updateRequests) == 0 {
			return cloudguardsdk.GetDetectorRecipeResponse{
				DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, currentSpec, cloudguardsdk.LifecycleStateActive),
			}, nil
		}
		return cloudguardsdk.GetDetectorRecipeResponse{
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(_ context.Context, request cloudguardsdk.UpdateDetectorRecipeRequest) (cloudguardsdk.UpdateDetectorRecipeResponse, error) {
		assertDetectorRecipeClearDescriptionRequest(t, request)
		return cloudguardsdk.UpdateDetectorRecipeResponse{
			OpcRequestId:   common.String("opc-clear-description"),
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateDetectorRecipe() calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.Description; got != "" {
		t.Fatalf("status.description = %q, want cleared description", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-clear-description" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-clear-description", got)
	}
}

func TestDetectorRecipeCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResourceWithoutRules()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDetectorRecipeID)

	currentSpec := resource.Spec
	currentSpec.CompartmentId = testDetectorRecipeCompartmentID
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	fake := &fakeDetectorRecipeOCIClient{}
	fake.getFunc = func(context.Context, cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		return cloudguardsdk.GetDetectorRecipeResponse{
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, currentSpec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, cloudguardsdk.UpdateDetectorRecipeRequest) (cloudguardsdk.UpdateDetectorRecipeResponse, error) {
		t.Fatal("UpdateDetectorRecipe() called; want immutable drift rejected before OCI update")
		return cloudguardsdk.UpdateDetectorRecipeResponse{}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	if !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement message", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateDetectorRecipe() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestDetectorRecipeDeleteRetainsFinalizerWhileDeleteIsPending(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResourceWithoutRules()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDetectorRecipeID)

	fake := &fakeDetectorRecipeOCIClient{}
	fake.getFunc = func(context.Context, cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		state := cloudguardsdk.LifecycleStateActive
		if len(fake.deleteRequests) > 0 {
			state = cloudguardsdk.LifecycleStateDeleting
		}
		return cloudguardsdk.GetDetectorRecipeResponse{
			DetectorRecipe: detectorRecipeFromSpec(t, testDetectorRecipeID, resource.Spec, state),
		}, nil
	}
	fake.deleteFunc = func(_ context.Context, request cloudguardsdk.DeleteDetectorRecipeRequest) (cloudguardsdk.DeleteDetectorRecipeResponse, error) {
		requireStringPtr(t, "DeleteDetectorRecipeRequest.DetectorRecipeId", request.DetectorRecipeId, testDetectorRecipeID)
		return cloudguardsdk.DeleteDetectorRecipeResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI delete is pending")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteDetectorRecipe() calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.LifecycleState; got != string(cloudguardsdk.LifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete operation")
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current.phase = %q, want delete", got)
	}
}

func TestDetectorRecipeDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResourceWithoutRules()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDetectorRecipeID)

	fake := &fakeDetectorRecipeOCIClient{}
	fake.getFunc = func(context.Context, cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
		return cloudguardsdk.GetDetectorRecipeResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "not authorized or not found")
	}
	fake.deleteFunc = func(context.Context, cloudguardsdk.DeleteDetectorRecipeRequest) (cloudguardsdk.DeleteDetectorRecipeResponse, error) {
		t.Fatal("DeleteDetectorRecipe() called; want auth-shaped pre-delete read to retain finalizer")
		return cloudguardsdk.DeleteDetectorRecipeResponse{}, nil
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want conservative auth-shaped readback error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained on ambiguous readback")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want NotAuthorizedOrNotFound context", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want service error opc request id", got)
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteDetectorRecipe() calls = %d, want 0", len(fake.deleteRequests))
	}
}

func TestDetectorRecipeCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := newDetectorRecipeRuntimeTestResourceWithoutRules()
	fake := &fakeDetectorRecipeOCIClient{}
	fake.createFunc = func(context.Context, cloudguardsdk.CreateDetectorRecipeRequest) (cloudguardsdk.CreateDetectorRecipeResponse, error) {
		return cloudguardsdk.CreateDetectorRecipeResponse{}, errortest.NewServiceError(500, "InternalServerError", "create failed")
	}
	client := newDetectorRecipeRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful OCI error", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want service error opc request id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want Failed", got)
	}
}

func assertDetectorRecipeMutableUpdateRequest(
	t *testing.T,
	request cloudguardsdk.UpdateDetectorRecipeRequest,
	spec cloudguardv1beta1.DetectorRecipeSpec,
) {
	t.Helper()

	requireStringPtr(t, "UpdateDetectorRecipeRequest.DetectorRecipeId", request.DetectorRecipeId, testDetectorRecipeID)
	requireStringPtr(t, "UpdateDetectorRecipeRequest.DisplayName", request.DisplayName, spec.DisplayName)
	if request.FreeformTags == nil || len(request.FreeformTags) != 0 {
		t.Fatalf("UpdateDetectorRecipeRequest.FreeformTags = %#v, want empty map to clear tags", request.FreeformTags)
	}
	if request.DefinedTags == nil || len(request.DefinedTags) != 0 {
		t.Fatalf("UpdateDetectorRecipeRequest.DefinedTags = %#v, want empty map to clear tags", request.DefinedTags)
	}
	assertDetectorRecipeRequestRuleDisablesRule(t, request.DetectorRules)
}

func assertDetectorRecipeClearDescriptionRequest(
	t *testing.T,
	request cloudguardsdk.UpdateDetectorRecipeRequest,
) {
	t.Helper()

	requireStringPtr(t, "UpdateDetectorRecipeRequest.DetectorRecipeId", request.DetectorRecipeId, testDetectorRecipeID)
	if request.Description == nil {
		t.Fatal("UpdateDetectorRecipeRequest.Description = nil, want empty string pointer")
	}
	if got := *request.Description; got != "" {
		t.Fatalf("UpdateDetectorRecipeRequest.Description = %q, want empty string", got)
	}
	if request.DisplayName != nil {
		t.Fatalf("UpdateDetectorRecipeRequest.DisplayName = %q, want unset", stringValue(request.DisplayName))
	}
	if len(request.DetectorRules) != 0 {
		t.Fatalf("UpdateDetectorRecipeRequest.DetectorRules length = %d, want 0", len(request.DetectorRules))
	}
	if request.FreeformTags != nil {
		t.Fatalf("UpdateDetectorRecipeRequest.FreeformTags = %#v, want unset", request.FreeformTags)
	}
	if request.DefinedTags != nil {
		t.Fatalf("UpdateDetectorRecipeRequest.DefinedTags = %#v, want unset", request.DefinedTags)
	}
}

func assertDetectorRecipeRequestRuleDisablesRule(
	t *testing.T,
	rules []cloudguardsdk.UpdateDetectorRecipeDetectorRule,
) {
	t.Helper()

	if len(rules) != 1 {
		t.Fatalf("DetectorRules length = %d, want 1", len(rules))
	}
	details := rules[0].Details
	if details == nil {
		t.Fatal("DetectorRules[0].Details = nil, want explicit details")
	}
	if details.IsEnabled == nil {
		t.Fatal("DetectorRules[0].Details.IsEnabled = nil, want explicit false")
	}
	if *details.IsEnabled {
		t.Fatalf("DetectorRules[0].Details.IsEnabled = %t, want false", *details.IsEnabled)
	}
}

func detectorRecipeSawListPage(
	requests []cloudguardsdk.ListDetectorRecipesRequest,
	state cloudguardsdk.ListDetectorRecipesLifecycleStateEnum,
	page string,
) bool {
	for _, request := range requests {
		if request.LifecycleState == state && stringValue(request.Page) == page {
			return true
		}
	}
	return false
}

func requireStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", field, *got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
