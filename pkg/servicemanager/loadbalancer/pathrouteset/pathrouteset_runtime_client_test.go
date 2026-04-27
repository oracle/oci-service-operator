/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pathrouteset

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
	pathRouteSetLoadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	pathRouteSetNameValue      = "example_path_route_set"
)

type fakeGeneratedPathRouteSetOCIClient struct {
	createRequests []loadbalancersdk.CreatePathRouteSetRequest
	getRequests    []loadbalancersdk.GetPathRouteSetRequest
	listRequests   []loadbalancersdk.ListPathRouteSetsRequest
	updateRequests []loadbalancersdk.UpdatePathRouteSetRequest
	deleteRequests []loadbalancersdk.DeletePathRouteSetRequest

	getErr    error
	createErr error
	listErr   error
	updateErr error
	deleteErr error

	keepAfterDelete bool
	pathRouteSets   map[string]loadbalancersdk.PathRouteSet
}

func (f *fakeGeneratedPathRouteSetOCIClient) CreatePathRouteSet(_ context.Context, request loadbalancersdk.CreatePathRouteSetRequest) (loadbalancersdk.CreatePathRouteSetResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return loadbalancersdk.CreatePathRouteSetResponse{}, f.createErr
	}
	f.ensurePathRouteSets()
	pathRouteSet := pathRouteSetFromCreateDetails(request.CreatePathRouteSetDetails)
	f.pathRouteSets[stringValue(pathRouteSet.Name)] = pathRouteSet
	return loadbalancersdk.CreatePathRouteSetResponse{}, nil
}

func (f *fakeGeneratedPathRouteSetOCIClient) GetPathRouteSet(_ context.Context, request loadbalancersdk.GetPathRouteSetRequest) (loadbalancersdk.GetPathRouteSetResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getErr != nil {
		return loadbalancersdk.GetPathRouteSetResponse{}, f.getErr
	}
	pathRouteSet, ok := f.pathRouteSets[stringValue(request.PathRouteSetName)]
	if !ok {
		return loadbalancersdk.GetPathRouteSetResponse{}, errortest.NewServiceError(404, "NotFound", "missing path route set")
	}
	return loadbalancersdk.GetPathRouteSetResponse{PathRouteSet: pathRouteSet}, nil
}

func (f *fakeGeneratedPathRouteSetOCIClient) ListPathRouteSets(_ context.Context, request loadbalancersdk.ListPathRouteSetsRequest) (loadbalancersdk.ListPathRouteSetsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return loadbalancersdk.ListPathRouteSetsResponse{}, f.listErr
	}
	names := make([]string, 0, len(f.pathRouteSets))
	for name := range f.pathRouteSets {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]loadbalancersdk.PathRouteSet, 0, len(names))
	for _, name := range names {
		items = append(items, f.pathRouteSets[name])
	}
	return loadbalancersdk.ListPathRouteSetsResponse{Items: items}, nil
}

func (f *fakeGeneratedPathRouteSetOCIClient) UpdatePathRouteSet(_ context.Context, request loadbalancersdk.UpdatePathRouteSetRequest) (loadbalancersdk.UpdatePathRouteSetResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return loadbalancersdk.UpdatePathRouteSetResponse{}, f.updateErr
	}
	f.ensurePathRouteSets()
	name := stringValue(request.PathRouteSetName)
	existing := f.pathRouteSets[name]
	f.pathRouteSets[name] = pathRouteSetFromUpdateDetails(name, request.UpdatePathRouteSetDetails, existing)
	return loadbalancersdk.UpdatePathRouteSetResponse{}, nil
}

func (f *fakeGeneratedPathRouteSetOCIClient) DeletePathRouteSet(_ context.Context, request loadbalancersdk.DeletePathRouteSetRequest) (loadbalancersdk.DeletePathRouteSetResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return loadbalancersdk.DeletePathRouteSetResponse{}, f.deleteErr
	}
	if !f.keepAfterDelete {
		delete(f.pathRouteSets, stringValue(request.PathRouteSetName))
	}
	return loadbalancersdk.DeletePathRouteSetResponse{}, nil
}

func (f *fakeGeneratedPathRouteSetOCIClient) ensurePathRouteSets() {
	if f.pathRouteSets == nil {
		f.pathRouteSets = map[string]loadbalancersdk.PathRouteSet{}
	}
}

func newTestPathRouteSetRuntimeClient(client *fakeGeneratedPathRouteSetOCIClient) PathRouteSetServiceClient {
	hooks := newPathRouteSetRuntimeHooksWithOCIClient(client)
	applyPathRouteSetRuntimeHooks(&hooks)
	config := buildPathRouteSetGeneratedRuntimeConfig(&PathRouteSetServiceManager{}, hooks)
	return defaultPathRouteSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.PathRouteSet](config),
	}
}

func TestPathRouteSetRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := newPathRouteSetRuntimeSemantics()
	if got == nil {
		t.Fatal("newPathRouteSetRuntimeSemantics() = nil")
	}
	if got.FormalService != "loadbalancer" {
		t.Fatalf("FormalService = %q, want loadbalancer", got.FormalService)
	}
	if got.FormalSlug != "pathrouteset" {
		t.Fatalf("FormalSlug = %q, want pathrouteset", got.FormalSlug)
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

	assertPathRouteSetStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertPathRouteSetStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertPathRouteSetStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"name"})
	assertPathRouteSetStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"pathRoutes"})
	assertPathRouteSetStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"name"})
}

func TestPathRouteSetRequestFieldsKeepOperationsScopedToRecordedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{
			name: "create",
			got:  pathRouteSetCreateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "CreatePathRouteSetDetails",
					RequestName:  "CreatePathRouteSetDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "get",
			got:  pathRouteSetGetFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "PathRouteSetName",
					RequestName:  "pathRouteSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
			},
		},
		{
			name: "list",
			got:  pathRouteSetListFields(),
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
			got:  pathRouteSetUpdateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "PathRouteSetName",
					RequestName:  "pathRouteSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
				{
					FieldName:    "UpdatePathRouteSetDetails",
					RequestName:  "UpdatePathRouteSetDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "delete",
			got:  pathRouteSetDeleteFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "PathRouteSetName",
					RequestName:  "pathRouteSetName",
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

func TestCreateOrUpdateRejectsMissingPathRouteSetLoadBalancerAnnotation(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedPathRouteSetResource()
	resource.Annotations = nil
	client := &fakeGeneratedPathRouteSetOCIClient{}

	response, err := newTestPathRouteSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), pathRouteSetLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing %s annotation", err, pathRouteSetLoadBalancerIDAnnotation)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.createRequests) != 0 || len(client.getRequests) != 0 || len(client.listRequests) != 0 {
		t.Fatalf("OCI calls = create:%d get:%d list:%d, want none", len(client.createRequests), len(client.getRequests), len(client.listRequests))
	}
}

func TestCreateOrUpdateCreatesThenObservesPathRouteSet(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedPathRouteSetOCIClient{
		pathRouteSets: map[string]loadbalancersdk.PathRouteSet{},
	}
	serviceClient := newTestPathRouteSetRuntimeClient(client)
	resource := makeUntrackedPathRouteSetResource()

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
	assertPathRouteSetPathIdentity(t, client.createRequests[0].LoadBalancerId, common.String(pathRouteSetNameValue), pathRouteSetLoadBalancerID, pathRouteSetNameValue)
	if got := stringValue(client.createRequests[0].CreatePathRouteSetDetails.Name); got != pathRouteSetNameValue {
		t.Fatalf("CreatePathRouteSetDetails.Name = %q, want %q", got, pathRouteSetNameValue)
	}
	assertPathRouteSetSDKPathRoutes(t, "create path routes", client.createRequests[0].CreatePathRouteSetDetails.PathRoutes, sdkPathRouteSetRoutes("/example/video", "video_backend", "PREFIX_MATCH"))
	assertPathRouteSetTrackedStatus(t, resource, pathRouteSetLoadBalancerID, pathRouteSetNameValue, resource.Spec.PathRoutes)

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
	assertPathRouteSetTrackedStatus(t, resource, pathRouteSetLoadBalancerID, pathRouteSetNameValue, resource.Spec.PathRoutes)
}

func TestCreateOrUpdateBindsExistingPathRouteSet(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedPathRouteSetResource()
	client := &fakeGeneratedPathRouteSetOCIClient{
		pathRouteSets: map[string]loadbalancersdk.PathRouteSet{
			pathRouteSetNameValue: sdkPathRouteSet(resource.Spec.PathRoutes),
		},
	}

	response, err := newTestPathRouteSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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
	assertPathRouteSetPathIdentity(t, client.getRequests[0].LoadBalancerId, client.getRequests[0].PathRouteSetName, pathRouteSetLoadBalancerID, pathRouteSetNameValue)
	assertPathRouteSetTrackedStatus(t, resource, pathRouteSetLoadBalancerID, pathRouteSetNameValue, resource.Spec.PathRoutes)
}

func TestCreateOrUpdateUpdatesPathRouteSetRoutes(t *testing.T) {
	t.Parallel()

	resource := makeTrackedPathRouteSetResource()
	desiredRoutes := []loadbalancerv1beta1.PathRouteSetPathRoute{
		pathRouteSetPathRoute("/example/video", "new_video_backend", "PREFIX_MATCH"),
	}
	resource.Spec.PathRoutes = desiredRoutes
	client := &fakeGeneratedPathRouteSetOCIClient{
		pathRouteSets: map[string]loadbalancersdk.PathRouteSet{
			pathRouteSetNameValue: sdkPathRouteSet([]loadbalancerv1beta1.PathRouteSetPathRoute{
				pathRouteSetPathRoute("/example/video", "old_video_backend", "PREFIX_MATCH"),
			}),
		},
	}

	response, err := newTestPathRouteSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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
	assertPathRouteSetPathIdentity(t, client.updateRequests[0].LoadBalancerId, client.updateRequests[0].PathRouteSetName, pathRouteSetLoadBalancerID, pathRouteSetNameValue)
	assertPathRouteSetSDKPathRoutes(t, "update path routes", client.updateRequests[0].UpdatePathRouteSetDetails.PathRoutes, sdkPathRoutesFromSpec(desiredRoutes))
	assertPathRouteSetTrackedStatus(t, resource, pathRouteSetLoadBalancerID, pathRouteSetNameValue, desiredRoutes)
}

func TestCreateOrUpdateRejectsPathRouteSetForceNewNameDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedPathRouteSetResource()
	resource.Spec.Name = "replacement_path_route_set"
	client := &fakeGeneratedPathRouteSetOCIClient{
		pathRouteSets: map[string]loadbalancersdk.PathRouteSet{
			pathRouteSetNameValue: sdkPathRouteSet(resource.Status.PathRoutes),
		},
	}

	response, err := newTestPathRouteSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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

func TestCreateOrUpdateRejectsPathRouteSetLoadBalancerAnnotationDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedPathRouteSetResource()
	resource.Annotations[pathRouteSetLoadBalancerIDAnnotation] = "ocid1.loadbalancer.oc1..replacement"
	client := &fakeGeneratedPathRouteSetOCIClient{}

	response, err := newTestPathRouteSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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

func TestDeleteConfirmsPathRouteSetRemoval(t *testing.T) {
	t.Parallel()

	resource := makeTrackedPathRouteSetResource()
	resource.Spec.Name = "replacement_path_route_set"
	client := &fakeGeneratedPathRouteSetOCIClient{
		pathRouteSets: map[string]loadbalancersdk.PathRouteSet{
			pathRouteSetNameValue: sdkPathRouteSet(resource.Status.PathRoutes),
		},
	}

	deleted, err := newTestPathRouteSetRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want confirmed delete")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	assertPathRouteSetPathIdentity(t, client.deleteRequests[0].LoadBalancerId, client.deleteRequests[0].PathRouteSetName, pathRouteSetLoadBalancerID, pathRouteSetNameValue)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want confirmed deletion timestamp")
	}
}

func TestDeleteRetainsPathRouteSetFinalizerWhileReadbackStillExists(t *testing.T) {
	t.Parallel()

	resource := makeTrackedPathRouteSetResource()
	client := &fakeGeneratedPathRouteSetOCIClient{
		keepAfterDelete: true,
		pathRouteSets: map[string]loadbalancersdk.PathRouteSet{
			pathRouteSetNameValue: sdkPathRouteSet(resource.Status.PathRoutes),
		},
	}

	deleted, err := newTestPathRouteSetRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained while readback still returns PathRouteSet")
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

func makeUntrackedPathRouteSetResource() *loadbalancerv1beta1.PathRouteSet {
	return &loadbalancerv1beta1.PathRouteSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: pathRouteSetNameValue,
			Annotations: map[string]string{
				pathRouteSetLoadBalancerIDAnnotation: pathRouteSetLoadBalancerID,
			},
		},
		Spec: loadbalancerv1beta1.PathRouteSetSpec{
			Name: pathRouteSetNameValue,
			PathRoutes: []loadbalancerv1beta1.PathRouteSetPathRoute{
				pathRouteSetPathRoute("/example/video", "video_backend", "PREFIX_MATCH"),
			},
		},
	}
}

func makeTrackedPathRouteSetResource() *loadbalancerv1beta1.PathRouteSet {
	resource := makeUntrackedPathRouteSetResource()
	resource.Status = loadbalancerv1beta1.PathRouteSetStatus{
		Name:       pathRouteSetNameValue,
		PathRoutes: append([]loadbalancerv1beta1.PathRouteSetPathRoute(nil), resource.Spec.PathRoutes...),
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(pathRouteSetLoadBalancerID),
		},
	}
	return resource
}

func pathRouteSetPathRoute(path, backendSetName, matchType string) loadbalancerv1beta1.PathRouteSetPathRoute {
	return loadbalancerv1beta1.PathRouteSetPathRoute{
		Path: path,
		PathMatchType: loadbalancerv1beta1.PathRouteSetPathRoutePathMatchType{
			MatchType: matchType,
		},
		BackendSetName: backendSetName,
	}
}

func sdkPathRouteSet(routes []loadbalancerv1beta1.PathRouteSetPathRoute) loadbalancersdk.PathRouteSet {
	return loadbalancersdk.PathRouteSet{
		Name:       common.String(pathRouteSetNameValue),
		PathRoutes: sdkPathRoutesFromSpec(routes),
	}
}

func sdkPathRouteSetRoutes(path, backendSetName, matchType string) []loadbalancersdk.PathRoute {
	return []loadbalancersdk.PathRoute{
		{
			Path: common.String(path),
			PathMatchType: &loadbalancersdk.PathMatchType{
				MatchType: loadbalancersdk.PathMatchTypeMatchTypeEnum(matchType),
			},
			BackendSetName: common.String(backendSetName),
		},
	}
}

func sdkPathRoutesFromSpec(routes []loadbalancerv1beta1.PathRouteSetPathRoute) []loadbalancersdk.PathRoute {
	converted := make([]loadbalancersdk.PathRoute, 0, len(routes))
	for _, route := range routes {
		converted = append(converted, loadbalancersdk.PathRoute{
			Path: common.String(route.Path),
			PathMatchType: &loadbalancersdk.PathMatchType{
				MatchType: loadbalancersdk.PathMatchTypeMatchTypeEnum(route.PathMatchType.MatchType),
			},
			BackendSetName: common.String(route.BackendSetName),
		})
	}
	return converted
}

func pathRouteSetFromCreateDetails(details loadbalancersdk.CreatePathRouteSetDetails) loadbalancersdk.PathRouteSet {
	return loadbalancersdk.PathRouteSet{
		Name:       details.Name,
		PathRoutes: details.PathRoutes,
	}
}

func pathRouteSetFromUpdateDetails(name string, details loadbalancersdk.UpdatePathRouteSetDetails, existing loadbalancersdk.PathRouteSet) loadbalancersdk.PathRouteSet {
	pathRouteSet := existing
	pathRouteSet.Name = common.String(name)
	pathRouteSet.PathRoutes = details.PathRoutes
	return pathRouteSet
}

func assertPathRouteSetPathIdentity(t *testing.T, loadBalancerID, pathRouteSetName *string, wantLoadBalancerID, wantPathRouteSetName string) {
	t.Helper()
	if got := stringValue(loadBalancerID); got != wantLoadBalancerID {
		t.Fatalf("LoadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(pathRouteSetName); got != wantPathRouteSetName {
		t.Fatalf("PathRouteSetName = %q, want %q", got, wantPathRouteSetName)
	}
}

func assertPathRouteSetTrackedStatus(t *testing.T, resource *loadbalancerv1beta1.PathRouteSet, wantLoadBalancerID, wantPathRouteSetName string, wantPathRoutes []loadbalancerv1beta1.PathRouteSetPathRoute) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil, want PathRouteSet")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantLoadBalancerID {
		t.Fatalf("status.status.ocid = %q, want recorded loadBalancerId %q", got, wantLoadBalancerID)
	}
	if got := resource.Status.Name; got != wantPathRouteSetName {
		t.Fatalf("status.name = %q, want %q", got, wantPathRouteSetName)
	}
	if !reflect.DeepEqual(resource.Status.PathRoutes, wantPathRoutes) {
		t.Fatalf("status.pathRoutes = %#v, want %#v", resource.Status.PathRoutes, wantPathRoutes)
	}
}

func assertPathRouteSetSDKPathRoutes(t *testing.T, name string, got, want []loadbalancersdk.PathRoute) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func assertPathRouteSetStringSliceEqual(t *testing.T, name string, got, want []string) {
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
