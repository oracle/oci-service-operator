/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sslciphersuite

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
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	sslCipherSuiteLoadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	sslCipherSuiteNameValue      = "example_ssl_cipher_suite"
)

type fakeGeneratedSSLCipherSuiteOCIClient struct {
	createRequests         []loadbalancersdk.CreateSSLCipherSuiteRequest
	getRequests            []loadbalancersdk.GetSSLCipherSuiteRequest
	listRequests           []loadbalancersdk.ListSSLCipherSuitesRequest
	updateRequests         []loadbalancersdk.UpdateSSLCipherSuiteRequest
	deleteRequests         []loadbalancersdk.DeleteSSLCipherSuiteRequest
	getWorkRequestRequests []loadbalancersdk.GetWorkRequestRequest

	getErr            error
	createErr         error
	listErr           error
	updateErr         error
	deleteErr         error
	getWorkRequestErr error

	createWorkRequestID string
	updateWorkRequestID string
	deleteWorkRequestID string
	keepAfterDelete     bool
	sslCipherSuites     map[string]loadbalancersdk.SslCipherSuite
	workRequests        map[string]loadbalancersdk.WorkRequest
}

func (f *fakeGeneratedSSLCipherSuiteOCIClient) CreateSSLCipherSuite(_ context.Context, request loadbalancersdk.CreateSSLCipherSuiteRequest) (loadbalancersdk.CreateSSLCipherSuiteResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return loadbalancersdk.CreateSSLCipherSuiteResponse{}, f.createErr
	}
	f.ensureSSLCipherSuites()
	sslCipherSuite := sslCipherSuiteFromCreateDetails(request.CreateSslCipherSuiteDetails)
	f.sslCipherSuites[stringValue(sslCipherSuite.Name)] = sslCipherSuite
	return loadbalancersdk.CreateSSLCipherSuiteResponse{OpcWorkRequestId: common.String(f.createWorkRequestID)}, nil
}

func (f *fakeGeneratedSSLCipherSuiteOCIClient) GetSSLCipherSuite(_ context.Context, request loadbalancersdk.GetSSLCipherSuiteRequest) (loadbalancersdk.GetSSLCipherSuiteResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getErr != nil {
		return loadbalancersdk.GetSSLCipherSuiteResponse{}, f.getErr
	}
	sslCipherSuite, ok := f.sslCipherSuites[stringValue(request.Name)]
	if !ok {
		return loadbalancersdk.GetSSLCipherSuiteResponse{}, errortest.NewServiceError(404, "NotFound", "missing SSL cipher suite")
	}
	return loadbalancersdk.GetSSLCipherSuiteResponse{SslCipherSuite: sslCipherSuite}, nil
}

func (f *fakeGeneratedSSLCipherSuiteOCIClient) ListSSLCipherSuites(_ context.Context, request loadbalancersdk.ListSSLCipherSuitesRequest) (loadbalancersdk.ListSSLCipherSuitesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return loadbalancersdk.ListSSLCipherSuitesResponse{}, f.listErr
	}
	names := make([]string, 0, len(f.sslCipherSuites))
	for name := range f.sslCipherSuites {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]loadbalancersdk.SslCipherSuite, 0, len(names))
	for _, name := range names {
		items = append(items, f.sslCipherSuites[name])
	}
	return loadbalancersdk.ListSSLCipherSuitesResponse{Items: items}, nil
}

func (f *fakeGeneratedSSLCipherSuiteOCIClient) UpdateSSLCipherSuite(_ context.Context, request loadbalancersdk.UpdateSSLCipherSuiteRequest) (loadbalancersdk.UpdateSSLCipherSuiteResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return loadbalancersdk.UpdateSSLCipherSuiteResponse{}, f.updateErr
	}
	f.ensureSSLCipherSuites()
	name := stringValue(request.Name)
	existing := f.sslCipherSuites[name]
	f.sslCipherSuites[name] = sslCipherSuiteFromUpdateDetails(name, request.UpdateSslCipherSuiteDetails, existing)
	return loadbalancersdk.UpdateSSLCipherSuiteResponse{OpcWorkRequestId: common.String(f.updateWorkRequestID)}, nil
}

func (f *fakeGeneratedSSLCipherSuiteOCIClient) DeleteSSLCipherSuite(_ context.Context, request loadbalancersdk.DeleteSSLCipherSuiteRequest) (loadbalancersdk.DeleteSSLCipherSuiteResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return loadbalancersdk.DeleteSSLCipherSuiteResponse{}, f.deleteErr
	}
	if !f.keepAfterDelete {
		delete(f.sslCipherSuites, stringValue(request.Name))
	}
	return loadbalancersdk.DeleteSSLCipherSuiteResponse{OpcWorkRequestId: common.String(f.deleteWorkRequestID)}, nil
}

func (f *fakeGeneratedSSLCipherSuiteOCIClient) GetWorkRequest(_ context.Context, request loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequestErr != nil {
		return loadbalancersdk.GetWorkRequestResponse{}, f.getWorkRequestErr
	}
	workRequest, ok := f.workRequests[stringValue(request.WorkRequestId)]
	if !ok {
		return loadbalancersdk.GetWorkRequestResponse{}, errortest.NewServiceError(404, "NotFound", "missing work request")
	}
	return loadbalancersdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
}

func (f *fakeGeneratedSSLCipherSuiteOCIClient) ensureSSLCipherSuites() {
	if f.sslCipherSuites == nil {
		f.sslCipherSuites = map[string]loadbalancersdk.SslCipherSuite{}
	}
}

func newTestSSLCipherSuiteRuntimeClient(client *fakeGeneratedSSLCipherSuiteOCIClient) SSLCipherSuiteServiceClient {
	hooks := newSSLCipherSuiteRuntimeHooksWithOCIClient(client)
	applySSLCipherSuiteRuntimeHooks(&hooks, client, nil, loggerutil.OSOKLogger{})
	config := buildSSLCipherSuiteGeneratedRuntimeConfig(&SSLCipherSuiteServiceManager{}, hooks)
	delegate := defaultSSLCipherSuiteServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.SSLCipherSuite](config),
	}
	return wrapSSLCipherSuiteGeneratedClient(hooks, delegate)
}

func TestSSLCipherSuiteRuntimeSemanticsEncodesWorkRequestLifecycleContract(t *testing.T) {
	t.Parallel()

	got := newSSLCipherSuiteRuntimeSemantics()
	if got == nil {
		t.Fatal("newSSLCipherSuiteRuntimeSemantics() = nil")
	}
	if got.FormalService != "loadbalancer" {
		t.Fatalf("FormalService = %q, want loadbalancer", got.FormalService)
	}
	if got.FormalSlug != "sslciphersuite" {
		t.Fatalf("FormalSlug = %q, want sslciphersuite", got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest semantics", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "workrequest" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want workrequest", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "workrequest" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want workrequest", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}

	assertSSLCipherSuiteStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertSSLCipherSuiteStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"name"})
	assertSSLCipherSuiteStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"ciphers"})
	assertSSLCipherSuiteStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"name"})
}

func TestSSLCipherSuiteRequestFieldsKeepOperationsScopedToRecordedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{
			name: "create",
			got:  sslCipherSuiteCreateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "CreateSslCipherSuiteDetails",
					RequestName:  "CreateSslCipherSuiteDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "get",
			got:  sslCipherSuiteGetFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "Name",
					RequestName:  "name",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
			},
		},
		{
			name: "list",
			got:  sslCipherSuiteListFields(),
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
			got:  sslCipherSuiteUpdateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "Name",
					RequestName:  "name",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
				{
					FieldName:    "UpdateSslCipherSuiteDetails",
					RequestName:  "UpdateSslCipherSuiteDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "delete",
			got:  sslCipherSuiteDeleteFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "Name",
					RequestName:  "name",
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

func TestCreateOrUpdateRejectsMissingSSLCipherSuiteLoadBalancerAnnotation(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedSSLCipherSuiteResource()
	resource.Annotations = nil
	client := &fakeGeneratedSSLCipherSuiteOCIClient{}

	response, err := newTestSSLCipherSuiteRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), sslCipherSuiteLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing %s annotation", err, sslCipherSuiteLoadBalancerIDAnnotation)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.createRequests) != 0 || len(client.getRequests) != 0 || len(client.listRequests) != 0 {
		t.Fatalf("OCI calls = create:%d get:%d list:%d, want none", len(client.createRequests), len(client.getRequests), len(client.listRequests))
	}
}

func TestCreateOrUpdateCreatesThenObservesSSLCipherSuiteWorkRequest(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedSSLCipherSuiteOCIClient{
		createWorkRequestID: "wr-create-1",
		sslCipherSuites:     map[string]loadbalancersdk.SslCipherSuite{},
		workRequests: map[string]loadbalancersdk.WorkRequest{
			"wr-create-1": sslCipherSuiteWorkRequest("wr-create-1", "CreateSSLCipherSuite", loadbalancersdk.WorkRequestLifecycleStateInProgress),
		},
	}
	serviceClient := newTestSSLCipherSuiteRuntimeClient(client)
	resource := makeUntrackedSSLCipherSuiteResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful create response", response)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want pending work request requeue")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	assertSSLCipherSuitePathIdentity(t, client.createRequests[0].LoadBalancerId, common.String(sslCipherSuiteNameValue), sslCipherSuiteLoadBalancerID, sslCipherSuiteNameValue)
	if got := stringValue(client.createRequests[0].CreateSslCipherSuiteDetails.Name); got != sslCipherSuiteNameValue {
		t.Fatalf("CreateSslCipherSuiteDetails.Name = %q, want %q", got, sslCipherSuiteNameValue)
	}
	assertSSLCipherSuiteStringSliceEqual(t, "create ciphers", client.createRequests[0].CreateSslCipherSuiteDetails.Ciphers, []string{"ECDHE-RSA-AES256-GCM-SHA384"})
	requireSSLCipherSuiteAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1", shared.OSOKAsyncClassPending)

	client.workRequests["wr-create-1"] = sslCipherSuiteWorkRequest("wr-create-1", "CreateSSLCipherSuite", loadbalancersdk.WorkRequestLifecycleStateSucceeded)
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
	if len(client.getRequests) == 0 {
		t.Fatal("get requests after work request success = 0, want readback")
	}
	assertSSLCipherSuiteTrackedStatus(t, resource, sslCipherSuiteLoadBalancerID, sslCipherSuiteNameValue, resource.Spec.Ciphers)
}

func TestCreateOrUpdateBindsExistingSSLCipherSuite(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedSSLCipherSuiteResource()
	client := &fakeGeneratedSSLCipherSuiteOCIClient{
		sslCipherSuites: map[string]loadbalancersdk.SslCipherSuite{
			sslCipherSuiteNameValue: sdkSSLCipherSuite(resource.Spec.Ciphers),
		},
	}

	response, err := newTestSSLCipherSuiteRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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
	assertSSLCipherSuitePathIdentity(t, client.getRequests[0].LoadBalancerId, client.getRequests[0].Name, sslCipherSuiteLoadBalancerID, sslCipherSuiteNameValue)
	assertSSLCipherSuiteTrackedStatus(t, resource, sslCipherSuiteLoadBalancerID, sslCipherSuiteNameValue, resource.Spec.Ciphers)
}

func TestCreateOrUpdateUpdatesSSLCipherSuiteCiphers(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSSLCipherSuiteResource()
	desiredCiphers := []string{"ECDHE-RSA-AES128-GCM-SHA256"}
	resource.Spec.Ciphers = desiredCiphers
	client := &fakeGeneratedSSLCipherSuiteOCIClient{
		updateWorkRequestID: "wr-update-1",
		sslCipherSuites: map[string]loadbalancersdk.SslCipherSuite{
			sslCipherSuiteNameValue: sdkSSLCipherSuite([]string{"ECDHE-RSA-AES256-GCM-SHA384"}),
		},
		workRequests: map[string]loadbalancersdk.WorkRequest{
			"wr-update-1": sslCipherSuiteWorkRequest("wr-update-1", "UpdateSSLCipherSuite", loadbalancersdk.WorkRequestLifecycleStateSucceeded),
		},
	}

	response, err := newTestSSLCipherSuiteRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want completed work request readback")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for update path", len(client.createRequests))
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	assertSSLCipherSuitePathIdentity(t, client.updateRequests[0].LoadBalancerId, client.updateRequests[0].Name, sslCipherSuiteLoadBalancerID, sslCipherSuiteNameValue)
	assertSSLCipherSuiteStringSliceEqual(t, "update ciphers", client.updateRequests[0].UpdateSslCipherSuiteDetails.Ciphers, desiredCiphers)
	assertSSLCipherSuiteTrackedStatus(t, resource, sslCipherSuiteLoadBalancerID, sslCipherSuiteNameValue, desiredCiphers)
}

func TestCreateOrUpdateRejectsSSLCipherSuiteForceNewNameDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSSLCipherSuiteResource()
	resource.Spec.Name = "replacement_ssl_cipher_suite"
	client := &fakeGeneratedSSLCipherSuiteOCIClient{
		sslCipherSuites: map[string]loadbalancersdk.SslCipherSuite{
			sslCipherSuiteNameValue: sdkSSLCipherSuite(resource.Status.Ciphers),
		},
	}

	response, err := newTestSSLCipherSuiteRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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

func TestCreateOrUpdateRejectsSSLCipherSuiteLoadBalancerAnnotationDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSSLCipherSuiteResource()
	resource.Annotations[sslCipherSuiteLoadBalancerIDAnnotation] = "ocid1.loadbalancer.oc1..replacement"
	client := &fakeGeneratedSSLCipherSuiteOCIClient{}

	response, err := newTestSSLCipherSuiteRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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

func TestDeleteConfirmsSSLCipherSuiteRemovalAfterWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSSLCipherSuiteResource()
	client := &fakeGeneratedSSLCipherSuiteOCIClient{
		deleteWorkRequestID: "wr-delete-1",
		sslCipherSuites: map[string]loadbalancersdk.SslCipherSuite{
			sslCipherSuiteNameValue: sdkSSLCipherSuite(resource.Status.Ciphers),
		},
		workRequests: map[string]loadbalancersdk.WorkRequest{
			"wr-delete-1": sslCipherSuiteWorkRequest("wr-delete-1", "DeleteSSLCipherSuite", loadbalancersdk.WorkRequestLifecycleStateSucceeded),
		},
	}

	deleted, err := newTestSSLCipherSuiteRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want confirmed delete")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	assertSSLCipherSuitePathIdentity(t, client.deleteRequests[0].LoadBalancerId, client.deleteRequests[0].Name, sslCipherSuiteLoadBalancerID, sslCipherSuiteNameValue)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want confirmed deletion timestamp")
	}
}

func TestDeleteRetainsSSLCipherSuiteFinalizerWhileReadbackStillExists(t *testing.T) {
	t.Parallel()

	resource := makeTrackedSSLCipherSuiteResource()
	client := &fakeGeneratedSSLCipherSuiteOCIClient{
		deleteWorkRequestID: "wr-delete-1",
		keepAfterDelete:     true,
		sslCipherSuites: map[string]loadbalancersdk.SslCipherSuite{
			sslCipherSuiteNameValue: sdkSSLCipherSuite(resource.Status.Ciphers),
		},
		workRequests: map[string]loadbalancersdk.WorkRequest{
			"wr-delete-1": sslCipherSuiteWorkRequest("wr-delete-1", "DeleteSSLCipherSuite", loadbalancersdk.WorkRequestLifecycleStateSucceeded),
		},
	}

	deleted, err := newTestSSLCipherSuiteRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained while readback still returns SSLCipherSuite")
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

func makeUntrackedSSLCipherSuiteResource() *loadbalancerv1beta1.SSLCipherSuite {
	return &loadbalancerv1beta1.SSLCipherSuite{
		ObjectMeta: metav1.ObjectMeta{
			Name: sslCipherSuiteNameValue,
			Annotations: map[string]string{
				sslCipherSuiteLoadBalancerIDAnnotation: sslCipherSuiteLoadBalancerID,
			},
		},
		Spec: loadbalancerv1beta1.SSLCipherSuiteSpec{
			Name:    sslCipherSuiteNameValue,
			Ciphers: []string{"ECDHE-RSA-AES256-GCM-SHA384"},
		},
	}
}

func makeTrackedSSLCipherSuiteResource() *loadbalancerv1beta1.SSLCipherSuite {
	resource := makeUntrackedSSLCipherSuiteResource()
	resource.Status = loadbalancerv1beta1.SSLCipherSuiteStatus{
		Name:    sslCipherSuiteNameValue,
		Ciphers: append([]string(nil), resource.Spec.Ciphers...),
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(sslCipherSuiteLoadBalancerID),
		},
	}
	return resource
}

func sdkSSLCipherSuite(ciphers []string) loadbalancersdk.SslCipherSuite {
	return loadbalancersdk.SslCipherSuite{
		Name:    common.String(sslCipherSuiteNameValue),
		Ciphers: append([]string(nil), ciphers...),
	}
}

func sslCipherSuiteFromCreateDetails(details loadbalancersdk.CreateSslCipherSuiteDetails) loadbalancersdk.SslCipherSuite {
	return loadbalancersdk.SslCipherSuite{
		Name:    details.Name,
		Ciphers: append([]string(nil), details.Ciphers...),
	}
}

func sslCipherSuiteFromUpdateDetails(name string, details loadbalancersdk.UpdateSslCipherSuiteDetails, existing loadbalancersdk.SslCipherSuite) loadbalancersdk.SslCipherSuite {
	sslCipherSuite := existing
	sslCipherSuite.Name = common.String(name)
	sslCipherSuite.Ciphers = append([]string(nil), details.Ciphers...)
	return sslCipherSuite
}

func sslCipherSuiteWorkRequest(id string, operationType string, state loadbalancersdk.WorkRequestLifecycleStateEnum) loadbalancersdk.WorkRequest {
	return loadbalancersdk.WorkRequest{
		Id:             common.String(id),
		LoadBalancerId: common.String(sslCipherSuiteLoadBalancerID),
		Type:           common.String(operationType),
		LifecycleState: state,
		Message:        common.String(operationType + " " + string(state)),
	}
}

func assertSSLCipherSuitePathIdentity(t *testing.T, loadBalancerID, name *string, wantLoadBalancerID, wantName string) {
	t.Helper()
	if got := stringValue(loadBalancerID); got != wantLoadBalancerID {
		t.Fatalf("LoadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(name); got != wantName {
		t.Fatalf("Name = %q, want %q", got, wantName)
	}
}

func assertSSLCipherSuiteTrackedStatus(t *testing.T, resource *loadbalancerv1beta1.SSLCipherSuite, wantLoadBalancerID, wantName string, wantCiphers []string) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil, want SSLCipherSuite")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantLoadBalancerID {
		t.Fatalf("status.status.ocid = %q, want recorded loadBalancerId %q", got, wantLoadBalancerID)
	}
	if got := resource.Status.Name; got != wantName {
		t.Fatalf("status.name = %q, want %q", got, wantName)
	}
	if !reflect.DeepEqual(resource.Status.Ciphers, wantCiphers) {
		t.Fatalf("status.ciphers = %#v, want %#v", resource.Status.Ciphers, wantCiphers)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Active)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func requireSSLCipherSuiteAsync(t *testing.T, resource *loadbalancerv1beta1.SSLCipherSuite, wantPhase shared.OSOKAsyncPhase, wantWorkRequestID string, wantClass shared.OSOKAsyncNormalizedClass) {
	t.Helper()
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want populated tracker")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
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

func assertSSLCipherSuiteStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}
