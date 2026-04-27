/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authenticationprovider

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOdaInstanceID            = "ocid1.odainstance.oc1..test"
	testAuthenticationProviderID = "ocid1.authenticationprovider.oc1..test"
)

type fakeAuthenticationProviderOCIClient struct {
	createFunc func(context.Context, odasdk.CreateAuthenticationProviderRequest) (odasdk.CreateAuthenticationProviderResponse, error)
	getFunc    func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error)
	listFunc   func(context.Context, odasdk.ListAuthenticationProvidersRequest) (odasdk.ListAuthenticationProvidersResponse, error)
	updateFunc func(context.Context, odasdk.UpdateAuthenticationProviderRequest) (odasdk.UpdateAuthenticationProviderResponse, error)
	deleteFunc func(context.Context, odasdk.DeleteAuthenticationProviderRequest) (odasdk.DeleteAuthenticationProviderResponse, error)

	createRequests []odasdk.CreateAuthenticationProviderRequest
	getRequests    []odasdk.GetAuthenticationProviderRequest
	listRequests   []odasdk.ListAuthenticationProvidersRequest
	updateRequests []odasdk.UpdateAuthenticationProviderRequest
	deleteRequests []odasdk.DeleteAuthenticationProviderRequest
}

func (f *fakeAuthenticationProviderOCIClient) CreateAuthenticationProvider(ctx context.Context, request odasdk.CreateAuthenticationProviderRequest) (odasdk.CreateAuthenticationProviderResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return odasdk.CreateAuthenticationProviderResponse{}, nil
}

func (f *fakeAuthenticationProviderOCIClient) GetAuthenticationProvider(ctx context.Context, request odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return odasdk.GetAuthenticationProviderResponse{}, nil
}

func (f *fakeAuthenticationProviderOCIClient) ListAuthenticationProviders(ctx context.Context, request odasdk.ListAuthenticationProvidersRequest) (odasdk.ListAuthenticationProvidersResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return odasdk.ListAuthenticationProvidersResponse{}, nil
}

func (f *fakeAuthenticationProviderOCIClient) UpdateAuthenticationProvider(ctx context.Context, request odasdk.UpdateAuthenticationProviderRequest) (odasdk.UpdateAuthenticationProviderResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return odasdk.UpdateAuthenticationProviderResponse{}, nil
}

func (f *fakeAuthenticationProviderOCIClient) DeleteAuthenticationProvider(ctx context.Context, request odasdk.DeleteAuthenticationProviderRequest) (odasdk.DeleteAuthenticationProviderResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return odasdk.DeleteAuthenticationProviderResponse{}, nil
}

func TestAuthenticationProviderRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	hooks := newAuthenticationProviderRuntimeHooksWithOCIClient(&fakeAuthenticationProviderOCIClient{})
	applyAuthenticationProviderRuntimeHooks(
		&AuthenticationProviderServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
		&hooks,
	)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics is nil")
	}
	if got := hooks.Semantics.FormalService; got != "oda" {
		t.Fatalf("FormalService = %q, want oda", got)
	}
	if got := hooks.Semantics.FormalSlug; got != "authenticationprovider" {
		t.Fatalf("FormalSlug = %q, want authenticationprovider", got)
	}
	if got := hooks.Semantics.Async.Runtime; got != "handwritten" {
		t.Fatalf("Async.Runtime = %q, want handwritten", got)
	}
	assertStringSliceEqual(t, "Lifecycle.ProvisioningStates", hooks.Semantics.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertStringSliceEqual(t, "Lifecycle.UpdatingStates", hooks.Semantics.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertStringSliceEqual(t, "Lifecycle.ActiveStates", hooks.Semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertStringSliceEqual(t, "Delete.PendingStates", hooks.Semantics.Delete.PendingStates, []string{"DELETING"})
	assertStringSliceEqual(t, "Delete.TerminalStates", hooks.Semantics.Delete.TerminalStates, []string{"DELETED"})
	assertStringSliceContains(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "tokenEndpointUrl")
	assertStringSliceContains(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "name")
	if len(hooks.Semantics.Unsupported) != 1 || hooks.Semantics.Unsupported[0].Category != "clientSecret-update-drift" {
		t.Fatalf("Unsupported = %#v, want clientSecret-update-drift entry", hooks.Semantics.Unsupported)
	}
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want 1", len(hooks.WrapGeneratedClient))
	}
}

func TestAuthenticationProviderRequiresOdaInstanceAnnotation(t *testing.T) {
	resource := makeAuthenticationProviderResource()
	resource.Annotations = nil
	fake := &fakeAuthenticationProviderOCIClient{
		createFunc: func(context.Context, odasdk.CreateAuthenticationProviderRequest) (odasdk.CreateAuthenticationProviderResponse, error) {
			t.Fatal("CreateAuthenticationProvider should not be called without parent annotation")
			return odasdk.CreateAuthenticationProviderResponse{}, nil
		},
	}
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), authenticationProviderOdaInstanceIDAnnotation) {
		t.Fatalf("CreateOrUpdate error = %v, want missing annotation error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
}

func TestAuthenticationProviderCreateProjectsStatus(t *testing.T) {
	resource := makeAuthenticationProviderResource()
	resource.Spec.IsVisible = true
	fake := &fakeAuthenticationProviderOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListAuthenticationProvidersRequest) (odasdk.ListAuthenticationProvidersResponse, error) {
		return odasdk.ListAuthenticationProvidersResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request odasdk.CreateAuthenticationProviderRequest) (odasdk.CreateAuthenticationProviderResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("create odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.CreateAuthenticationProviderDetails.Name); got != resource.Spec.Name {
			t.Fatalf("create name = %q, want %q", got, resource.Spec.Name)
		}
		if got := stringValue(request.CreateAuthenticationProviderDetails.ClientSecret); got != resource.Spec.ClientSecret {
			t.Fatalf("create clientSecret = %q, want spec secret", got)
		}
		if request.CreateAuthenticationProviderDetails.IsVisible == nil || !*request.CreateAuthenticationProviderDetails.IsVisible {
			t.Fatalf("create isVisible = %#v, want true", request.CreateAuthenticationProviderDetails.IsVisible)
		}
		return odasdk.CreateAuthenticationProviderResponse{
			AuthenticationProvider: makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateCreating),
			OpcRequestId:           common.String("create-request"),
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
		return odasdk.GetAuthenticationProviderResponse{
			AuthenticationProvider: makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive),
		}, nil
	}
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want 1", len(fake.listRequests))
	}
	if got := stringValue(fake.listRequests[0].OdaInstanceId); got != testOdaInstanceID {
		t.Fatalf("list odaInstanceId = %q, want %q", got, testOdaInstanceID)
	}
	if got := fake.listRequests[0].IdentityProvider; got != odasdk.ListAuthenticationProvidersIdentityProviderGeneric {
		t.Fatalf("list identityProvider = %q, want GENERIC", got)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	assertAuthenticationProviderActiveStatus(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "create-request" {
		t.Fatalf("opcRequestId = %q, want create-request", got)
	}
}

func TestAuthenticationProviderBindsExistingWithoutCreate(t *testing.T) {
	resource := makeAuthenticationProviderResource()
	fake := &fakeAuthenticationProviderOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListAuthenticationProvidersRequest) (odasdk.ListAuthenticationProvidersResponse, error) {
		return odasdk.ListAuthenticationProvidersResponse{
			AuthenticationProviderCollection: odasdk.AuthenticationProviderCollection{
				Items: []odasdk.AuthenticationProviderSummary{
					makeSDKAuthenticationProviderSummary(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive),
				},
			},
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
		return odasdk.GetAuthenticationProviderResponse{
			AuthenticationProvider: makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive),
		}, nil
	}
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	assertAuthenticationProviderActiveStatus(t, resource)
}

func TestAuthenticationProviderUpdatesSupportedMutableDrift(t *testing.T) {
	resource := makeAuthenticationProviderResource()
	resource.Status.Id = testAuthenticationProviderID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAuthenticationProviderID)
	resource.Spec.TokenEndpointUrl = "https://idp.example.com/new-token"
	resource.Spec.ClientId = "new-client"
	resource.Spec.Scopes = "openid email profile"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	current := makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive)
	current.TokenEndpointUrl = common.String("https://idp.example.com/old-token")
	current.ClientId = common.String("old-client")
	current.Scopes = common.String("openid")
	current.FreeformTags = map[string]string{"env": "dev"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}

	updated := makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive)
	fake := &fakeAuthenticationProviderOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
		switch len(fake.getRequests) {
		case 1:
			return odasdk.GetAuthenticationProviderResponse{AuthenticationProvider: current}, nil
		default:
			return odasdk.GetAuthenticationProviderResponse{AuthenticationProvider: updated}, nil
		}
	}
	fake.updateFunc = func(_ context.Context, request odasdk.UpdateAuthenticationProviderRequest) (odasdk.UpdateAuthenticationProviderResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("update odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AuthenticationProviderId); got != testAuthenticationProviderID {
			t.Fatalf("update authProviderId = %q, want %q", got, testAuthenticationProviderID)
		}
		return odasdk.UpdateAuthenticationProviderResponse{
			AuthenticationProvider: updated,
			OpcRequestId:           common.String("update-request"),
		}, nil
	}
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	details := fake.updateRequests[0].UpdateAuthenticationProviderDetails
	if got := stringValue(details.TokenEndpointUrl); got != resource.Spec.TokenEndpointUrl {
		t.Fatalf("updated tokenEndpointUrl = %q, want %q", got, resource.Spec.TokenEndpointUrl)
	}
	if got := stringValue(details.ClientId); got != resource.Spec.ClientId {
		t.Fatalf("updated clientId = %q, want %q", got, resource.Spec.ClientId)
	}
	if got := stringValue(details.Scopes); got != resource.Spec.Scopes {
		t.Fatalf("updated scopes = %q, want %q", got, resource.Spec.Scopes)
	}
	if details.ClientSecret != nil {
		t.Fatalf("update clientSecret = %#v, want nil because OCI does not return secret drift", details.ClientSecret)
	}
	if got := details.FreeformTags["env"]; got != "prod" {
		t.Fatalf("updated freeform env = %q, want prod", got)
	}
	if got := details.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("updated defined tag CostCenter = %#v, want 42", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "update-request" {
		t.Fatalf("opcRequestId = %q, want update-request", got)
	}
	assertAuthenticationProviderActiveStatus(t, resource)
}

func TestAuthenticationProviderRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := makeAuthenticationProviderResource()
	resource.Status.Id = testAuthenticationProviderID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAuthenticationProviderID)
	current := makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive)
	current.Name = common.String("different-name")
	fake := &fakeAuthenticationProviderOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
		return odasdk.GetAuthenticationProviderResponse{AuthenticationProvider: current}, nil
	}
	fake.updateFunc = func(context.Context, odasdk.UpdateAuthenticationProviderRequest) (odasdk.UpdateAuthenticationProviderResponse, error) {
		t.Fatal("UpdateAuthenticationProvider should not be called for create-only drift")
		return odasdk.UpdateAuthenticationProviderResponse{}, nil
	}
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create-only field drift") || !strings.Contains(err.Error(), "name") {
		t.Fatalf("CreateOrUpdate error = %v, want create-only name drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
}

func TestAuthenticationProviderClassifiesLifecycleStates(t *testing.T) {
	tests := []struct {
		name          string
		state         odasdk.LifecycleStateEnum
		wantReason    shared.OSOKConditionType
		wantRequeue   bool
		wantSuccess   bool
		wantAsync     bool
		wantAsyncNorm shared.OSOKAsyncNormalizedClass
	}{
		{name: "creating", state: odasdk.LifecycleStateCreating, wantReason: shared.Provisioning, wantRequeue: true, wantSuccess: true, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassPending},
		{name: "updating", state: odasdk.LifecycleStateUpdating, wantReason: shared.Updating, wantRequeue: true, wantSuccess: true, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassPending},
		{name: "deleting", state: odasdk.LifecycleStateDeleting, wantReason: shared.Terminating, wantRequeue: true, wantSuccess: true, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassPending},
		{name: "active", state: odasdk.LifecycleStateActive, wantReason: shared.Active, wantRequeue: false, wantSuccess: true, wantAsync: false},
		{name: "failed", state: odasdk.LifecycleStateFailed, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassUnknown},
		{name: "inactive", state: odasdk.LifecycleStateInactive, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := makeAuthenticationProviderResource()
			resource.Status.Id = testAuthenticationProviderID
			resource.Status.OsokStatus.Ocid = shared.OCID(testAuthenticationProviderID)
			fake := &fakeAuthenticationProviderOCIClient{}
			fake.getFunc = func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
				return odasdk.GetAuthenticationProviderResponse{
					AuthenticationProvider: makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, tt.state),
				}, nil
			}
			fake.updateFunc = func(context.Context, odasdk.UpdateAuthenticationProviderRequest) (odasdk.UpdateAuthenticationProviderResponse, error) {
				t.Fatal("UpdateAuthenticationProvider should not be called for lifecycle classification")
				return odasdk.UpdateAuthenticationProviderResponse{}, nil
			}
			client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate returned error: %v", err)
			}
			if response.IsSuccessful != tt.wantSuccess {
				t.Fatalf("IsSuccessful = %v, want %v", response.IsSuccessful, tt.wantSuccess)
			}
			if response.ShouldRequeue != tt.wantRequeue {
				t.Fatalf("ShouldRequeue = %v, want %v", response.ShouldRequeue, tt.wantRequeue)
			}
			if got := resource.Status.OsokStatus.Reason; got != string(tt.wantReason) {
				t.Fatalf("status.reason = %q, want %q", got, tt.wantReason)
			}
			if tt.wantAsync {
				if resource.Status.OsokStatus.Async.Current == nil {
					t.Fatalf("async.current = nil, want %s", tt.wantAsyncNorm)
				}
				if got := resource.Status.OsokStatus.Async.Current.NormalizedClass; got != tt.wantAsyncNorm {
					t.Fatalf("async.normalizedClass = %q, want %q", got, tt.wantAsyncNorm)
				}
			} else if resource.Status.OsokStatus.Async.Current != nil {
				t.Fatalf("async.current = %#v, want nil", resource.Status.OsokStatus.Async.Current)
			}
		})
	}
}

func TestAuthenticationProviderDeleteWaitsForDeletingConfirmation(t *testing.T) {
	resource := makeAuthenticationProviderResource()
	resource.Status.Id = testAuthenticationProviderID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAuthenticationProviderID)
	fake := &fakeAuthenticationProviderOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetAuthenticationProviderResponse{
				AuthenticationProvider: makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetAuthenticationProviderResponse{
			AuthenticationProvider: makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateDeleting),
		}, nil
	}
	fake.deleteFunc = func(_ context.Context, request odasdk.DeleteAuthenticationProviderRequest) (odasdk.DeleteAuthenticationProviderResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testOdaInstanceID {
			t.Fatalf("delete odaInstanceId = %q, want %q", got, testOdaInstanceID)
		}
		if got := stringValue(request.AuthenticationProviderId); got != testAuthenticationProviderID {
			t.Fatalf("delete authProviderId = %q, want %q", got, testAuthenticationProviderID)
		}
		return odasdk.DeleteAuthenticationProviderResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want false while OCI is DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("DeletedAt = %#v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "delete-request" {
		t.Fatalf("opcRequestId = %q, want delete-request", got)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("async.current = %#v, want pending delete async", current)
	}
}

func TestAuthenticationProviderDeleteConfirmsReadNotFound(t *testing.T) {
	resource := makeAuthenticationProviderResource()
	resource.Status.Id = testAuthenticationProviderID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAuthenticationProviderID)
	fake := &fakeAuthenticationProviderOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetAuthenticationProviderRequest) (odasdk.GetAuthenticationProviderResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetAuthenticationProviderResponse{
				AuthenticationProvider: makeSDKAuthenticationProvider(testAuthenticationProviderID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetAuthenticationProviderResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "authentication provider not found")
	}
	fake.deleteFunc = func(context.Context, odasdk.DeleteAuthenticationProviderRequest) (odasdk.DeleteAuthenticationProviderResponse, error) {
		return odasdk.DeleteAuthenticationProviderResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newAuthenticationProviderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete returned deleted=false, want true after confirm read not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want timestamp after deletion confirmation")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("async.current = %#v, want nil after delete confirmation", current)
	}
}

func makeAuthenticationProviderResource() *odav1beta1.AuthenticationProvider {
	return &odav1beta1.AuthenticationProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-auth-provider",
			Namespace: "default",
			Annotations: map[string]string{
				authenticationProviderOdaInstanceIDAnnotation: testOdaInstanceID,
			},
		},
		Spec: odav1beta1.AuthenticationProviderSpec{
			GrantType:        string(odasdk.AuthenticationGrantTypeClientCredentials),
			IdentityProvider: string(odasdk.AuthenticationIdentityProviderGeneric),
			Name:             "test-auth-provider",
			TokenEndpointUrl: "https://idp.example.com/token",
			ClientId:         "client-id",
			ClientSecret:     "client-secret",
			Scopes:           "openid",
		},
	}
}

func makeSDKAuthenticationProvider(id string, resource *odav1beta1.AuthenticationProvider, state odasdk.LifecycleStateEnum) odasdk.AuthenticationProvider {
	return odasdk.AuthenticationProvider{
		Id:                                common.String(id),
		GrantType:                         odasdk.AuthenticationGrantTypeEnum(resource.Spec.GrantType),
		IdentityProvider:                  odasdk.AuthenticationIdentityProviderEnum(resource.Spec.IdentityProvider),
		Name:                              common.String(resource.Spec.Name),
		TokenEndpointUrl:                  common.String(resource.Spec.TokenEndpointUrl),
		ClientId:                          common.String(resource.Spec.ClientId),
		Scopes:                            common.String(resource.Spec.Scopes),
		IsVisible:                         common.Bool(resource.Spec.IsVisible),
		LifecycleState:                    state,
		AuthorizationEndpointUrl:          optionalString(resource.Spec.AuthorizationEndpointUrl),
		ShortAuthorizationCodeRequestUrl:  optionalString(resource.Spec.ShortAuthorizationCodeRequestUrl),
		RevokeTokenEndpointUrl:            optionalString(resource.Spec.RevokeTokenEndpointUrl),
		SubjectClaim:                      optionalString(resource.Spec.SubjectClaim),
		RefreshTokenRetentionPeriodInDays: optionalInt(resource.Spec.RefreshTokenRetentionPeriodInDays),
		RedirectUrl:                       optionalString(resource.Spec.RedirectUrl),
		FreeformTags:                      cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:                       authenticationProviderDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKAuthenticationProviderSummary(id string, resource *odav1beta1.AuthenticationProvider, state odasdk.LifecycleStateEnum) odasdk.AuthenticationProviderSummary {
	return odasdk.AuthenticationProviderSummary{
		Id:               common.String(id),
		GrantType:        odasdk.AuthenticationGrantTypeEnum(resource.Spec.GrantType),
		IdentityProvider: odasdk.AuthenticationIdentityProviderEnum(resource.Spec.IdentityProvider),
		Name:             common.String(resource.Spec.Name),
		LifecycleState:   state,
	}
}

func assertAuthenticationProviderActiveStatus(t *testing.T, resource *odav1beta1.AuthenticationProvider) {
	t.Helper()
	if got := resource.Status.Id; got != testAuthenticationProviderID {
		t.Fatalf("status.id = %q, want %q", got, testAuthenticationProviderID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testAuthenticationProviderID {
		t.Fatalf("status.ocid = %q, want %q", got, testAuthenticationProviderID)
	}
	if got := resource.Status.LifecycleState; got != string(odasdk.LifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Active)
	}
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("async.current = %#v, want nil for ACTIVE", current)
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return common.Int(value)
}

func assertStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}

func assertStringSliceContains(t *testing.T, name string, got []string, want string) {
	t.Helper()
	for _, value := range got {
		if value == want {
			return
		}
	}
	t.Fatalf("%s = %#v, want entry %q", name, got, want)
}
