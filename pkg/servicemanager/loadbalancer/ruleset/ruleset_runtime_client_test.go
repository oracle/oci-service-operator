/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ruleset

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ruleSetLoadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	ruleSetNameValue      = "example_rule_set"
)

type fakeGeneratedRuleSetOCIClient struct {
	createRequests []loadbalancersdk.CreateRuleSetRequest
	getRequests    []loadbalancersdk.GetRuleSetRequest
	listRequests   []loadbalancersdk.ListRuleSetsRequest
	updateRequests []loadbalancersdk.UpdateRuleSetRequest
	deleteRequests []loadbalancersdk.DeleteRuleSetRequest

	getErr    error
	createErr error
	listErr   error
	updateErr error
	deleteErr error

	keepAfterDelete bool
	ruleSets        map[string]loadbalancersdk.RuleSet
}

func (f *fakeGeneratedRuleSetOCIClient) CreateRuleSet(_ context.Context, request loadbalancersdk.CreateRuleSetRequest) (loadbalancersdk.CreateRuleSetResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return loadbalancersdk.CreateRuleSetResponse{}, f.createErr
	}
	f.ensureRuleSets()
	ruleSet := ruleSetFromCreateDetails(request.CreateRuleSetDetails)
	f.ruleSets[stringValue(ruleSet.Name)] = ruleSet
	return loadbalancersdk.CreateRuleSetResponse{}, nil
}

func (f *fakeGeneratedRuleSetOCIClient) GetRuleSet(_ context.Context, request loadbalancersdk.GetRuleSetRequest) (loadbalancersdk.GetRuleSetResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getErr != nil {
		return loadbalancersdk.GetRuleSetResponse{}, f.getErr
	}
	ruleSet, ok := f.ruleSets[stringValue(request.RuleSetName)]
	if !ok {
		return loadbalancersdk.GetRuleSetResponse{}, errortest.NewServiceError(404, "NotFound", "missing rule set")
	}
	return loadbalancersdk.GetRuleSetResponse{RuleSet: ruleSet}, nil
}

func (f *fakeGeneratedRuleSetOCIClient) ListRuleSets(_ context.Context, request loadbalancersdk.ListRuleSetsRequest) (loadbalancersdk.ListRuleSetsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return loadbalancersdk.ListRuleSetsResponse{}, f.listErr
	}
	names := make([]string, 0, len(f.ruleSets))
	for name := range f.ruleSets {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]loadbalancersdk.RuleSet, 0, len(names))
	for _, name := range names {
		items = append(items, f.ruleSets[name])
	}
	return loadbalancersdk.ListRuleSetsResponse{Items: items}, nil
}

func (f *fakeGeneratedRuleSetOCIClient) UpdateRuleSet(_ context.Context, request loadbalancersdk.UpdateRuleSetRequest) (loadbalancersdk.UpdateRuleSetResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return loadbalancersdk.UpdateRuleSetResponse{}, f.updateErr
	}
	f.ensureRuleSets()
	name := stringValue(request.RuleSetName)
	existing := f.ruleSets[name]
	f.ruleSets[name] = ruleSetFromUpdateDetails(name, request.UpdateRuleSetDetails, existing)
	return loadbalancersdk.UpdateRuleSetResponse{}, nil
}

func (f *fakeGeneratedRuleSetOCIClient) DeleteRuleSet(_ context.Context, request loadbalancersdk.DeleteRuleSetRequest) (loadbalancersdk.DeleteRuleSetResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return loadbalancersdk.DeleteRuleSetResponse{}, f.deleteErr
	}
	if !f.keepAfterDelete {
		delete(f.ruleSets, stringValue(request.RuleSetName))
	}
	return loadbalancersdk.DeleteRuleSetResponse{}, nil
}

func (f *fakeGeneratedRuleSetOCIClient) ensureRuleSets() {
	if f.ruleSets == nil {
		f.ruleSets = map[string]loadbalancersdk.RuleSet{}
	}
}

func newTestRuleSetRuntimeClient(client *fakeGeneratedRuleSetOCIClient) RuleSetServiceClient {
	hooks := newRuleSetRuntimeHooksWithOCIClient(client)
	applyRuleSetRuntimeHooks(&hooks)
	config := buildRuleSetGeneratedRuntimeConfig(&RuleSetServiceManager{}, hooks)
	return defaultRuleSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.RuleSet](config),
	}
}

func TestRuleSetRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := newRuleSetRuntimeSemantics()
	if got == nil {
		t.Fatal("newRuleSetRuntimeSemantics() = nil")
	}
	if got.FormalService != "loadbalancer" {
		t.Fatalf("FormalService = %q, want loadbalancer", got.FormalService)
	}
	if got.FormalSlug != "ruleset" {
		t.Fatalf("FormalSlug = %q, want ruleset", got.FormalSlug)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}

	assertRuleSetStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertRuleSetStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertRuleSetStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"name"})
	assertRuleSetStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"items"})
	assertRuleSetStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"name"})
}

func TestRuleSetRequestFieldsKeepOperationsScopedToRecordedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{
			name: "create",
			got:  ruleSetCreateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "CreateRuleSetDetails",
					RequestName:  "CreateRuleSetDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "get",
			got:  ruleSetGetFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "RuleSetName",
					RequestName:  "ruleSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
			},
		},
		{
			name: "list",
			got:  ruleSetListFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
			},
		},
		{
			name: "update",
			got:  ruleSetUpdateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "RuleSetName",
					RequestName:  "ruleSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
				{
					FieldName:    "UpdateRuleSetDetails",
					RequestName:  "UpdateRuleSetDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "delete",
			got:  ruleSetDeleteFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "RuleSetName",
					RequestName:  "ruleSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Fatalf("%s fields = %#v, want %#v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestBuildRuleSetBodiesConvertSupportedRuleShapes(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedRuleSetResource()
	resource.Spec.Items = []loadbalancerv1beta1.RuleSetItem{
		ruleSetAddRequestHeaderRule("x-request-id", "enabled"),
		{
			Action: string(loadbalancersdk.RuleActionRedirect),
			Conditions: []loadbalancerv1beta1.RuleSetItemCondition{
				ruleSetPathCondition("/old", "PREFIX_MATCH"),
			},
			ResponseCode: 301,
			RedirectUri: loadbalancerv1beta1.RuleSetItemRedirectUri{
				Protocol: "HTTPS",
				Host:     "example.com",
				Path:     "/new",
				Query:    "source=osok",
			},
		},
		{
			Action:                      string(loadbalancersdk.RuleActionHttpHeader),
			AreInvalidCharactersAllowed: true,
			HttpLargeHeaderSizeInKB:     32,
		},
	}

	createBody, err := buildRuleSetCreateBody(resource)
	if err != nil {
		t.Fatalf("buildRuleSetCreateBody() error = %v", err)
	}

	want := []loadbalancersdk.Rule{
		loadbalancersdk.AddHttpRequestHeaderRule{
			Header: common.String("x-request-id"),
			Value:  common.String("enabled"),
		},
		loadbalancersdk.RedirectRule{
			Conditions: []loadbalancersdk.RuleCondition{
				loadbalancersdk.PathMatchCondition{
					AttributeValue: common.String("/old"),
					Operator:       loadbalancersdk.PathMatchConditionOperatorPrefixMatch,
				},
			},
			ResponseCode: common.Int(301),
			RedirectUri: &loadbalancersdk.RedirectUri{
				Protocol: common.String("HTTPS"),
				Host:     common.String("example.com"),
				Path:     common.String("/new"),
				Query:    common.String("source=osok"),
			},
		},
		loadbalancersdk.HttpHeaderRule{
			AreInvalidCharactersAllowed: common.Bool(true),
			HttpLargeHeaderSizeInKB:     common.Int(32),
		},
	}
	assertRuleSetSDKRules(t, "create items", createBody.Items, want)

	updateBody, updateNeeded, err := buildRuleSetUpdateBody(resource, loadbalancersdk.RuleSet{Name: common.String(ruleSetNameValue), Items: want[:1]})
	if err != nil {
		t.Fatalf("buildRuleSetUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildRuleSetUpdateBody() updateNeeded = false, want true")
	}
	assertRuleSetSDKRules(t, "update items", updateBody.Items, want)
}

func TestCreateOrUpdateRejectsMissingRuleSetLoadBalancerAnnotation(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedRuleSetResource()
	resource.Annotations = nil
	client := &fakeGeneratedRuleSetOCIClient{}

	response, err := newTestRuleSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), ruleSetLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing %s annotation", err, ruleSetLoadBalancerIDAnnotation)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.createRequests) != 0 || len(client.getRequests) != 0 || len(client.listRequests) != 0 {
		t.Fatalf("OCI calls = create:%d get:%d list:%d, want none", len(client.createRequests), len(client.getRequests), len(client.listRequests))
	}
}

func TestCreateOrUpdateCreatesThenObservesRuleSet(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedRuleSetOCIClient{
		ruleSets: map[string]loadbalancersdk.RuleSet{},
	}
	serviceClient := newTestRuleSetRuntimeClient(client)
	resource := makeUntrackedRuleSetResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful create response", response)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want create fallback to requeue")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	assertRuleSetPathIdentity(t, client.createRequests[0].LoadBalancerId, common.String(ruleSetNameValue), ruleSetLoadBalancerID, ruleSetNameValue)
	if got := stringValue(client.createRequests[0].CreateRuleSetDetails.Name); got != ruleSetNameValue {
		t.Fatalf("CreateRuleSetDetails.Name = %q, want %q", got, ruleSetNameValue)
	}
	assertRuleSetSDKRules(t, "create items", client.createRequests[0].CreateRuleSetDetails.Items, sdkRuleSetAddRequestHeaderRules("x-osok", "enabled"))
	assertRuleSetTrackedStatus(t, resource, ruleSetLoadBalancerID, ruleSetNameValue, resource.Spec.Items)

	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("observe CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("observe CreateOrUpdate() response = %#v, want successful observe response", response)
	}
	if response.ShouldRequeue {
		t.Fatal("observe CreateOrUpdate() ShouldRequeue = true, want active observation")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests after observe = %d, want 1", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests after observe = %d, want 0", len(client.updateRequests))
	}
	assertRuleSetTrackedStatus(t, resource, ruleSetLoadBalancerID, ruleSetNameValue, resource.Spec.Items)
}

func TestCreateOrUpdateBindsExistingRuleSet(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedRuleSetResource()
	client := &fakeGeneratedRuleSetOCIClient{
		ruleSets: map[string]loadbalancersdk.RuleSet{
			ruleSetNameValue: sdkRuleSet(resource.Spec.Items),
		},
	}

	response, err := newTestRuleSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful bind response", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want active bind response")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind path", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for no-drift bind path", len(client.updateRequests))
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1 for bind path", len(client.getRequests))
	}
	assertRuleSetPathIdentity(t, client.getRequests[0].LoadBalancerId, client.getRequests[0].RuleSetName, ruleSetLoadBalancerID, ruleSetNameValue)
	assertRuleSetTrackedStatus(t, resource, ruleSetLoadBalancerID, ruleSetNameValue, resource.Spec.Items)
}

func TestCreateOrUpdateUpdatesRuleSetItems(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRuleSetResource()
	desiredItems := []loadbalancerv1beta1.RuleSetItem{
		ruleSetAddRequestHeaderRule("x-osok", "updated"),
	}
	resource.Spec.Items = desiredItems
	client := &fakeGeneratedRuleSetOCIClient{
		ruleSets: map[string]loadbalancersdk.RuleSet{
			ruleSetNameValue: sdkRuleSet([]loadbalancerv1beta1.RuleSetItem{
				ruleSetAddRequestHeaderRule("x-osok", "enabled"),
			}),
		},
	}

	response, err := newTestRuleSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want update fallback to requeue")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for update path", len(client.createRequests))
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	assertRuleSetPathIdentity(t, client.updateRequests[0].LoadBalancerId, client.updateRequests[0].RuleSetName, ruleSetLoadBalancerID, ruleSetNameValue)
	assertRuleSetSDKRules(t, "update items", client.updateRequests[0].UpdateRuleSetDetails.Items, sdkRuleSetAddRequestHeaderRules("x-osok", "updated"))
	assertRuleSetTrackedStatus(t, resource, ruleSetLoadBalancerID, ruleSetNameValue, desiredItems)
}

func TestCreateOrUpdateSkipsRuleSetUpdateForServiceDefaultedRuleFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRuleSetResource()
	resource.Spec.Items = []loadbalancerv1beta1.RuleSetItem{
		{
			Action: string(loadbalancersdk.RuleActionRedirect),
			Conditions: []loadbalancerv1beta1.RuleSetItemCondition{
				ruleSetPathCondition("/old", "PREFIX_MATCH"),
			},
			RedirectUri: loadbalancerv1beta1.RuleSetItemRedirectUri{
				Protocol: "HTTPS",
				Host:     "example.com",
				Path:     "/new",
			},
		},
		{
			Action:         string(loadbalancersdk.RuleActionControlAccessUsingHttpMethods),
			AllowedMethods: []string{"GET", "POST"},
		},
		{
			Action: string(loadbalancersdk.RuleActionHttpHeader),
		},
	}
	resource.Status.Items = append([]loadbalancerv1beta1.RuleSetItem(nil), resource.Spec.Items...)
	client := &fakeGeneratedRuleSetOCIClient{
		ruleSets: map[string]loadbalancersdk.RuleSet{
			ruleSetNameValue: {
				Name: common.String(ruleSetNameValue),
				Items: []loadbalancersdk.Rule{
					loadbalancersdk.RedirectRule{
						Conditions: []loadbalancersdk.RuleCondition{
							loadbalancersdk.PathMatchCondition{
								AttributeValue: common.String("/old"),
								Operator:       loadbalancersdk.PathMatchConditionOperatorPrefixMatch,
							},
						},
						ResponseCode: common.Int(ruleSetDefaultRedirectResponseCode),
						RedirectUri: &loadbalancersdk.RedirectUri{
							Protocol: common.String("HTTPS"),
							Host:     common.String("example.com"),
							Path:     common.String("/new"),
						},
					},
					loadbalancersdk.ControlAccessUsingHttpMethodsRule{
						AllowedMethods: []string{"GET", "POST"},
						StatusCode:     common.Int(ruleSetDefaultControlAccessStatusCode),
					},
					loadbalancersdk.HttpHeaderRule{
						AreInvalidCharactersAllowed: common.Bool(ruleSetDefaultInvalidHeaderCharsAllowed),
					},
				},
			},
		},
	}

	response, err := newTestRuleSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful observe response", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want active observation")
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for service-defaulted fields", len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsRuleSetForceNewNameDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRuleSetResource()
	resource.Spec.Name = "replacement_rule_set"
	client := &fakeGeneratedRuleSetOCIClient{
		ruleSets: map[string]loadbalancersdk.RuleSet{
			ruleSetNameValue: sdkRuleSet(resource.Status.Items),
		},
	}

	response, err := newTestRuleSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want name replacement error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 on force-new drift", len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsRuleSetLoadBalancerAnnotationDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRuleSetResource()
	resource.Annotations[ruleSetLoadBalancerIDAnnotation] = "ocid1.loadbalancer.oc1..replacement"
	client := &fakeGeneratedRuleSetOCIClient{}

	response, err := newTestRuleSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "changed from recorded loadBalancerId") {
		t.Fatalf("CreateOrUpdate() error = %v, want annotation drift error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.createRequests) != 0 || len(client.getRequests) != 0 || len(client.updateRequests) != 0 {
		t.Fatalf("OCI calls = create:%d get:%d update:%d, want none", len(client.createRequests), len(client.getRequests), len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsUnsupportedRuleSetActionBeforeOCI(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedRuleSetResource()
	resource.Spec.Items = []loadbalancerv1beta1.RuleSetItem{{Action: "UNSUPPORTED"}}
	client := &fakeGeneratedRuleSetOCIClient{}

	response, err := newTestRuleSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "unsupported RuleSet item action") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported action error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.createRequests) != 0 || len(client.updateRequests) != 0 {
		t.Fatalf("OCI mutation calls = create:%d update:%d, want none", len(client.createRequests), len(client.updateRequests))
	}
}

func TestDeleteConfirmsRuleSetRemoval(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRuleSetResource()
	resource.Spec.Name = "replacement_rule_set"
	client := &fakeGeneratedRuleSetOCIClient{
		ruleSets: map[string]loadbalancersdk.RuleSet{
			ruleSetNameValue: sdkRuleSet(resource.Status.Items),
		},
	}

	deleted, err := newTestRuleSetRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want confirmed delete")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	assertRuleSetPathIdentity(t, client.deleteRequests[0].LoadBalancerId, client.deleteRequests[0].RuleSetName, ruleSetLoadBalancerID, ruleSetNameValue)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want confirmed deletion timestamp")
	}
}

func TestDeleteRetainsRuleSetFinalizerWhileReadbackStillExists(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRuleSetResource()
	client := &fakeGeneratedRuleSetOCIClient{
		keepAfterDelete: true,
		ruleSets: map[string]loadbalancersdk.RuleSet{
			ruleSetNameValue: sdkRuleSet(resource.Status.Items),
		},
	}

	deleted, err := newTestRuleSetRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained while readback still returns RuleSet")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want empty while delete is still pending")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending delete async operation", current)
	}
}

func makeUntrackedRuleSetResource() *loadbalancerv1beta1.RuleSet {
	return &loadbalancerv1beta1.RuleSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: ruleSetNameValue,
			Annotations: map[string]string{
				ruleSetLoadBalancerIDAnnotation: ruleSetLoadBalancerID,
			},
		},
		Spec: loadbalancerv1beta1.RuleSetSpec{
			Name: ruleSetNameValue,
			Items: []loadbalancerv1beta1.RuleSetItem{
				ruleSetAddRequestHeaderRule("x-osok", "enabled"),
			},
		},
	}
}

func makeTrackedRuleSetResource() *loadbalancerv1beta1.RuleSet {
	resource := makeUntrackedRuleSetResource()
	resource.Status = loadbalancerv1beta1.RuleSetStatus{
		Name:  ruleSetNameValue,
		Items: append([]loadbalancerv1beta1.RuleSetItem(nil), resource.Spec.Items...),
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(ruleSetLoadBalancerID),
		},
	}
	return resource
}

func ruleSetAddRequestHeaderRule(header, value string) loadbalancerv1beta1.RuleSetItem {
	return loadbalancerv1beta1.RuleSetItem{
		Action: string(loadbalancersdk.RuleActionAddHttpRequestHeader),
		Header: header,
		Value:  value,
	}
}

func ruleSetPathCondition(path, operator string) loadbalancerv1beta1.RuleSetItemCondition {
	return loadbalancerv1beta1.RuleSetItemCondition{
		AttributeName:  string(loadbalancersdk.RuleConditionAttributeNamePath),
		AttributeValue: path,
		Operator:       operator,
	}
}

func sdkRuleSet(items []loadbalancerv1beta1.RuleSetItem) loadbalancersdk.RuleSet {
	return loadbalancersdk.RuleSet{
		Name:  common.String(ruleSetNameValue),
		Items: sdkRulesFromSpec(items),
	}
}

func sdkRuleSetAddRequestHeaderRules(header, value string) []loadbalancersdk.Rule {
	return []loadbalancersdk.Rule{
		loadbalancersdk.AddHttpRequestHeaderRule{
			Header: common.String(header),
			Value:  common.String(value),
		},
	}
}

func sdkRulesFromSpec(items []loadbalancerv1beta1.RuleSetItem) []loadbalancersdk.Rule {
	rules, err := ruleSetSDKRules(items)
	if err != nil {
		panic(err)
	}
	return rules
}

func ruleSetFromCreateDetails(details loadbalancersdk.CreateRuleSetDetails) loadbalancersdk.RuleSet {
	return loadbalancersdk.RuleSet{
		Name:  details.Name,
		Items: details.Items,
	}
}

func ruleSetFromUpdateDetails(name string, details loadbalancersdk.UpdateRuleSetDetails, existing loadbalancersdk.RuleSet) loadbalancersdk.RuleSet {
	ruleSet := existing
	ruleSet.Name = common.String(name)
	ruleSet.Items = details.Items
	return ruleSet
}

func assertRuleSetPathIdentity(t *testing.T, loadBalancerID, ruleSetName *string, wantLoadBalancerID, wantRuleSetName string) {
	t.Helper()
	if got := stringValue(loadBalancerID); got != wantLoadBalancerID {
		t.Fatalf("LoadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(ruleSetName); got != wantRuleSetName {
		t.Fatalf("RuleSetName = %q, want %q", got, wantRuleSetName)
	}
}

func assertRuleSetTrackedStatus(t *testing.T, resource *loadbalancerv1beta1.RuleSet, wantLoadBalancerID, wantRuleSetName string, wantItems []loadbalancerv1beta1.RuleSetItem) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil, want RuleSet")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantLoadBalancerID {
		t.Fatalf("status.status.ocid = %q, want recorded loadBalancerId %q", got, wantLoadBalancerID)
	}
	if got := resource.Status.Name; got != wantRuleSetName {
		t.Fatalf("status.name = %q, want %q", got, wantRuleSetName)
	}
	if !reflect.DeepEqual(resource.Status.Items, wantItems) {
		t.Fatalf("status.items = %#v, want %#v", resource.Status.Items, wantItems)
	}
}

func assertRuleSetSDKRules(t *testing.T, name string, got, want []loadbalancersdk.Rule) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func assertRuleSetStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
