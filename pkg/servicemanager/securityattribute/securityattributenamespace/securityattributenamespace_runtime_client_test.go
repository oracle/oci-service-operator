/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securityattributenamespace

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	securityattributesdk "github.com/oracle/oci-go-sdk/v65/securityattribute"
	securityattributev1beta1 "github.com/oracle/oci-service-operator/api/securityattribute/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestSecurityAttributeNamespaceRuntimeSemantics(t *testing.T) {
	t.Parallel()

	hooks := newSecurityAttributeNamespaceRuntimeHooksWithOCIClient(nil)
	applySecurityAttributeNamespaceRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed runtime semantics")
	}
	assertContainsAll(t, "Lifecycle.ActiveStates", hooks.Semantics.Lifecycle.ActiveStates, "ACTIVE", "INACTIVE")
	assertContainsAll(t, "Delete.PendingStates", hooks.Semantics.Delete.PendingStates, "DELETING")
	assertContainsAll(t, "Delete.TerminalStates", hooks.Semantics.Delete.TerminalStates, "DELETED")
	assertContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "compartmentId", "name")
	assertContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "description", "isRetired", "freeformTags", "definedTags")
	assertContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId", "name")
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("BuildCreateBody/BuildUpdateBody = nil, want resource-specific body builders")
	}
	if hooks.DeleteHooks.HandleError == nil || hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("DeleteHooks are incomplete, want conservative delete handling")
	}
}

func TestSecurityAttributeNamespaceServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.listFunc = func(context.Context, securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
		return securityattributesdk.ListSecurityAttributeNamespacesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request securityattributesdk.CreateSecurityAttributeNamespaceRequest) (securityattributesdk.CreateSecurityAttributeNamespaceResponse, error) {
		return securityattributesdk.CreateSecurityAttributeNamespaceResponse{
			SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
			OpcRequestId:               common.String("opc-create"),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request securityattributesdk.GetSecurityAttributeNamespaceRequest) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error) {
		if got, want := stringValue(request.SecurityAttributeNamespaceId), "namespace-id"; got != want {
			t.Fatalf("GetSecurityAttributeNamespace id = %q, want %q", got, want)
		}
		return securityattributesdk.GetSecurityAttributeNamespaceResponse{
			SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
			OpcRequestId:               common.String("opc-get"),
		}, nil
	}

	response, err := newTestSecurityAttributeNamespaceServiceClient(fake).CreateOrUpdate(context.Background(), resource, requestFor(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateSecurityAttributeNamespace calls = %d, want 1", len(fake.createRequests))
	}
	createDetails := fake.createRequests[0].CreateSecurityAttributeNamespaceDetails
	if got, want := stringValue(createDetails.CompartmentId), resource.Spec.CompartmentId; got != want {
		t.Fatalf("create compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(createDetails.Name), resource.Spec.Name; got != want {
		t.Fatalf("create name = %q, want %q", got, want)
	}
	if got, want := stringValue(createDetails.Description), resource.Spec.Description; got != want {
		t.Fatalf("create description = %q, want %q", got, want)
	}
	assertSecurityAttributeNamespaceTrackedID(t, resource, "namespace-id")
	assertSecurityAttributeNamespaceCondition(t, resource, shared.Active)
	assertSecurityAttributeNamespaceOpcRequestID(t, resource, "opc-create")
}

func TestSecurityAttributeNamespaceServiceClientBindsFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.listFunc = func(_ context.Context, request securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
		switch page := stringValue(request.Page); page {
		case "":
			return securityattributesdk.ListSecurityAttributeNamespacesResponse{
				Items: []securityattributesdk.SecurityAttributeNamespaceSummary{
					sdkSecurityAttributeNamespaceSummary("other-id", resource.Spec.CompartmentId, "other", false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return securityattributesdk.ListSecurityAttributeNamespacesResponse{
				Items: []securityattributesdk.SecurityAttributeNamespaceSummary{
					sdkSecurityAttributeNamespaceSummary("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
				},
			}, nil
		default:
			t.Fatalf("ListSecurityAttributeNamespaces page = %q, want empty or page-2", page)
			return securityattributesdk.ListSecurityAttributeNamespacesResponse{}, nil
		}
	}
	fake.getFunc = func(_ context.Context, request securityattributesdk.GetSecurityAttributeNamespaceRequest) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error) {
		if got, want := stringValue(request.SecurityAttributeNamespaceId), "namespace-id"; got != want {
			t.Fatalf("GetSecurityAttributeNamespace id = %q, want %q", got, want)
		}
		return securityattributesdk.GetSecurityAttributeNamespaceResponse{
			SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
		}, nil
	}

	response, err := newTestSecurityAttributeNamespaceServiceClient(fake).CreateOrUpdate(context.Background(), resource, requestFor(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateSecurityAttributeNamespace calls = %d, want 0 for bind", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListSecurityAttributeNamespaces calls = %d, want 2", len(fake.listRequests))
	}
	assertSecurityAttributeNamespaceTrackedID(t, resource, "namespace-id")
}

func TestSecurityAttributeNamespaceServiceClientNoOpReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	resource.Status.OsokStatus.Ocid = shared.OCID("namespace-id")
	resource.Status.Id = "namespace-id"
	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.getResponses = []securityattributesdk.GetSecurityAttributeNamespaceResponse{
		{
			SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
		},
	}

	response, err := newTestSecurityAttributeNamespaceServiceClient(fake).CreateOrUpdate(context.Background(), resource, requestFor(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateSecurityAttributeNamespace calls = %d, want 0", len(fake.updateRequests))
	}
	assertSecurityAttributeNamespaceCondition(t, resource, shared.Active)
}

func TestSecurityAttributeNamespaceServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	resource.Status.OsokStatus.Ocid = shared.OCID("namespace-id")
	resource.Status.Id = "namespace-id"
	resource.Spec.Description = "new description"
	resource.Spec.IsRetired = true
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.getResponses = []securityattributesdk.GetSecurityAttributeNamespaceResponse{
		{
			SecurityAttributeNamespace: securityattributesdk.SecurityAttributeNamespace{
				Id:             common.String("namespace-id"),
				CompartmentId:  common.String(resource.Spec.CompartmentId),
				Name:           common.String(resource.Spec.Name),
				Description:    common.String("old description"),
				IsRetired:      common.Bool(false),
				FreeformTags:   map[string]string{"old": "tag"},
				DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "17"}},
				LifecycleState: securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive,
			},
		},
		{
			SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, true, securityattributesdk.SecurityAttributeNamespaceLifecycleStateInactive),
		},
	}
	fake.updateFunc = func(_ context.Context, request securityattributesdk.UpdateSecurityAttributeNamespaceRequest) (securityattributesdk.UpdateSecurityAttributeNamespaceResponse, error) {
		if got, want := stringValue(request.SecurityAttributeNamespaceId), "namespace-id"; got != want {
			t.Fatalf("UpdateSecurityAttributeNamespace id = %q, want %q", got, want)
		}
		return securityattributesdk.UpdateSecurityAttributeNamespaceResponse{
			SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, true, securityattributesdk.SecurityAttributeNamespaceLifecycleStateInactive),
			OpcRequestId:               common.String("opc-update"),
		}, nil
	}

	response, err := newTestSecurityAttributeNamespaceServiceClient(fake).CreateOrUpdate(context.Background(), resource, requestFor(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateSecurityAttributeNamespace calls = %d, want 1", len(fake.updateRequests))
	}
	assertSecurityAttributeNamespaceMutableUpdate(t, fake.updateRequests[0].UpdateSecurityAttributeNamespaceDetails, resource.Spec)
	assertSecurityAttributeNamespaceCondition(t, resource, shared.Active)
	assertSecurityAttributeNamespaceOpcRequestID(t, resource, "opc-update")
}

func TestSecurityAttributeNamespaceServiceClientRecordsMutatingOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.listFunc = func(context.Context, securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
		return securityattributesdk.ListSecurityAttributeNamespacesResponse{}, nil
	}
	fake.createFunc = func(context.Context, securityattributesdk.CreateSecurityAttributeNamespaceRequest) (securityattributesdk.CreateSecurityAttributeNamespaceResponse, error) {
		return securityattributesdk.CreateSecurityAttributeNamespaceResponse{}, createErr
	}

	response, err := newTestSecurityAttributeNamespaceServiceClient(fake).CreateOrUpdate(context.Background(), resource, requestFor(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	assertSecurityAttributeNamespaceCondition(t, resource, shared.Failed)
	assertSecurityAttributeNamespaceOpcRequestID(t, resource, "opc-create-error")
}

func TestSecurityAttributeNamespaceServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	resource.Status.OsokStatus.Ocid = shared.OCID("namespace-id")
	resource.Status.Id = "namespace-id"
	resource.Spec.Description = "new description"
	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.getResponses = []securityattributesdk.GetSecurityAttributeNamespaceResponse{
		{
			SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, "renamed-in-oci", "old description", false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive),
		},
	}

	response, err := newTestSecurityAttributeNamespaceServiceClient(fake).CreateOrUpdate(context.Background(), resource, requestFor(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("CreateOrUpdate() error = %q, want name drift", err.Error())
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateSecurityAttributeNamespace calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestSecurityAttributeNamespaceServiceClientDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	resource.Status.OsokStatus.Ocid = shared.OCID("namespace-id")
	resource.Status.Id = "namespace-id"
	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.getResponses = []securityattributesdk.GetSecurityAttributeNamespaceResponse{
		{SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive)},
		{SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive)},
		{SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleting)},
		{SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleting)},
		{SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateDeleted)},
	}
	fake.deleteFunc = func(context.Context, securityattributesdk.DeleteSecurityAttributeNamespaceRequest) (securityattributesdk.DeleteSecurityAttributeNamespaceResponse, error) {
		return securityattributesdk.DeleteSecurityAttributeNamespaceResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	client := newTestSecurityAttributeNamespaceServiceClient(fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while OCI lifecycle is DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteSecurityAttributeNamespace calls after first delete = %d, want 1", len(fake.deleteRequests))
	}
	assertSecurityAttributeNamespaceCondition(t, resource, shared.Terminating)
	assertSecurityAttributeNamespaceOpcRequestID(t, resource, "opc-delete")

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after terminal lifecycle")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteSecurityAttributeNamespace calls after second delete = %d, want still 1", len(fake.deleteRequests))
	}
}

func TestSecurityAttributeNamespaceServiceClientRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	resource.Status.OsokStatus.Ocid = shared.OCID("namespace-id")
	resource.Status.Id = "namespace-id"
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	authErr.OpcRequestID = "opc-auth-predelete"
	fake := &fakeSecurityAttributeNamespaceOCIClient{
		getErrors: []error{authErr},
	}

	deleted, err := newTestSecurityAttributeNamespaceServiceClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirmation rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteSecurityAttributeNamespace calls = %d, want 0", len(fake.deleteRequests))
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 context", err.Error())
	}
	assertSecurityAttributeNamespaceOpcRequestID(t, resource, "opc-auth-predelete")
}

func TestSecurityAttributeNamespaceServiceClientRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newTestSecurityAttributeNamespace()
	resource.Status.OsokStatus.Ocid = shared.OCID("namespace-id")
	resource.Status.Id = "namespace-id"
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	authErr.OpcRequestID = "opc-auth-postdelete"
	fake := &fakeSecurityAttributeNamespaceOCIClient{}
	fake.getResponses = []securityattributesdk.GetSecurityAttributeNamespaceResponse{
		{SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive)},
		{SecurityAttributeNamespace: sdkSecurityAttributeNamespace("namespace-id", resource.Spec.CompartmentId, resource.Spec.Name, resource.Spec.Description, false, securityattributesdk.SecurityAttributeNamespaceLifecycleStateActive)},
	}
	fake.getErrors = []error{nil, nil, authErr}
	fake.deleteFunc = func(context.Context, securityattributesdk.DeleteSecurityAttributeNamespaceRequest) (securityattributesdk.DeleteSecurityAttributeNamespaceResponse, error) {
		return securityattributesdk.DeleteSecurityAttributeNamespaceResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestSecurityAttributeNamespaceServiceClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete confirmation rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteSecurityAttributeNamespace calls = %d, want 1", len(fake.deleteRequests))
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 context", err.Error())
	}
	assertSecurityAttributeNamespaceOpcRequestID(t, resource, "opc-auth-postdelete")
}

type fakeSecurityAttributeNamespaceOCIClient struct {
	createFunc func(context.Context, securityattributesdk.CreateSecurityAttributeNamespaceRequest) (securityattributesdk.CreateSecurityAttributeNamespaceResponse, error)
	getFunc    func(context.Context, securityattributesdk.GetSecurityAttributeNamespaceRequest) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error)
	listFunc   func(context.Context, securityattributesdk.ListSecurityAttributeNamespacesRequest) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error)
	updateFunc func(context.Context, securityattributesdk.UpdateSecurityAttributeNamespaceRequest) (securityattributesdk.UpdateSecurityAttributeNamespaceResponse, error)
	deleteFunc func(context.Context, securityattributesdk.DeleteSecurityAttributeNamespaceRequest) (securityattributesdk.DeleteSecurityAttributeNamespaceResponse, error)

	createRequests []securityattributesdk.CreateSecurityAttributeNamespaceRequest
	getRequests    []securityattributesdk.GetSecurityAttributeNamespaceRequest
	listRequests   []securityattributesdk.ListSecurityAttributeNamespacesRequest
	updateRequests []securityattributesdk.UpdateSecurityAttributeNamespaceRequest
	deleteRequests []securityattributesdk.DeleteSecurityAttributeNamespaceRequest

	getResponses []securityattributesdk.GetSecurityAttributeNamespaceResponse
	getErrors    []error
}

func (f *fakeSecurityAttributeNamespaceOCIClient) CreateSecurityAttributeNamespace(
	ctx context.Context,
	request securityattributesdk.CreateSecurityAttributeNamespaceRequest,
) (securityattributesdk.CreateSecurityAttributeNamespaceResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc == nil {
		return securityattributesdk.CreateSecurityAttributeNamespaceResponse{}, fmt.Errorf("unexpected CreateSecurityAttributeNamespace call")
	}
	return f.createFunc(ctx, request)
}

func (f *fakeSecurityAttributeNamespaceOCIClient) GetSecurityAttributeNamespace(
	ctx context.Context,
	request securityattributesdk.GetSecurityAttributeNamespaceRequest,
) (securityattributesdk.GetSecurityAttributeNamespaceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	index := len(f.getRequests) - 1
	if index < len(f.getErrors) && f.getErrors[index] != nil {
		return securityattributesdk.GetSecurityAttributeNamespaceResponse{}, f.getErrors[index]
	}
	if index < len(f.getResponses) {
		return f.getResponses[index], nil
	}
	return securityattributesdk.GetSecurityAttributeNamespaceResponse{}, fmt.Errorf("unexpected GetSecurityAttributeNamespace call %d", index+1)
}

func (f *fakeSecurityAttributeNamespaceOCIClient) ListSecurityAttributeNamespaces(
	ctx context.Context,
	request securityattributesdk.ListSecurityAttributeNamespacesRequest,
) (securityattributesdk.ListSecurityAttributeNamespacesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc == nil {
		return securityattributesdk.ListSecurityAttributeNamespacesResponse{}, fmt.Errorf("unexpected ListSecurityAttributeNamespaces call")
	}
	return f.listFunc(ctx, request)
}

func (f *fakeSecurityAttributeNamespaceOCIClient) UpdateSecurityAttributeNamespace(
	ctx context.Context,
	request securityattributesdk.UpdateSecurityAttributeNamespaceRequest,
) (securityattributesdk.UpdateSecurityAttributeNamespaceResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc == nil {
		return securityattributesdk.UpdateSecurityAttributeNamespaceResponse{}, fmt.Errorf("unexpected UpdateSecurityAttributeNamespace call")
	}
	return f.updateFunc(ctx, request)
}

func (f *fakeSecurityAttributeNamespaceOCIClient) DeleteSecurityAttributeNamespace(
	ctx context.Context,
	request securityattributesdk.DeleteSecurityAttributeNamespaceRequest,
) (securityattributesdk.DeleteSecurityAttributeNamespaceResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc == nil {
		return securityattributesdk.DeleteSecurityAttributeNamespaceResponse{}, fmt.Errorf("unexpected DeleteSecurityAttributeNamespace call")
	}
	return f.deleteFunc(ctx, request)
}

func newTestSecurityAttributeNamespaceServiceClient(fake *fakeSecurityAttributeNamespaceOCIClient) SecurityAttributeNamespaceServiceClient {
	return newSecurityAttributeNamespaceServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
}

func newTestSecurityAttributeNamespace() *securityattributev1beta1.SecurityAttributeNamespace {
	return &securityattributev1beta1.SecurityAttributeNamespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "namespace-sample",
			Namespace: "default",
		},
		Spec: securityattributev1beta1.SecurityAttributeNamespaceSpec{
			CompartmentId: "tenancy-id",
			Name:          "sample-namespace",
			Description:   "sample namespace",
		},
	}
}

func requestFor(resource *securityattributev1beta1.SecurityAttributeNamespace) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func sdkSecurityAttributeNamespace(
	id string,
	compartmentID string,
	name string,
	description string,
	retired bool,
	lifecycleState securityattributesdk.SecurityAttributeNamespaceLifecycleStateEnum,
) securityattributesdk.SecurityAttributeNamespace {
	return securityattributesdk.SecurityAttributeNamespace{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		Name:           common.String(name),
		Description:    common.String(description),
		IsRetired:      common.Bool(retired),
		LifecycleState: lifecycleState,
	}
}

func sdkSecurityAttributeNamespaceSummary(
	id string,
	compartmentID string,
	name string,
	retired bool,
	lifecycleState securityattributesdk.SecurityAttributeNamespaceLifecycleStateEnum,
) securityattributesdk.SecurityAttributeNamespaceSummary {
	return securityattributesdk.SecurityAttributeNamespaceSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		Name:           common.String(name),
		Description:    common.String("summary description"),
		IsRetired:      common.Bool(retired),
		LifecycleState: lifecycleState,
	}
}

func assertContainsAll(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	present := map[string]bool{}
	for _, value := range got {
		present[value] = true
	}
	for _, value := range want {
		if !present[value] {
			t.Fatalf("%s = %#v, want it to contain %q", label, got, value)
		}
	}
}

func assertSecurityAttributeNamespaceTrackedID(
	t *testing.T,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	want string,
) {
	t.Helper()
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func assertSecurityAttributeNamespaceCondition(
	t *testing.T,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions is empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}

func assertSecurityAttributeNamespaceOpcRequestID(
	t *testing.T,
	resource *securityattributev1beta1.SecurityAttributeNamespace,
	want string,
) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertSecurityAttributeNamespaceMutableUpdate(
	t *testing.T,
	updateDetails securityattributesdk.UpdateSecurityAttributeNamespaceDetails,
	spec securityattributev1beta1.SecurityAttributeNamespaceSpec,
) {
	t.Helper()
	if got, want := stringValue(updateDetails.Description), spec.Description; got != want {
		t.Fatalf("update description = %q, want %q", got, want)
	}
	if updateDetails.IsRetired == nil || !*updateDetails.IsRetired {
		t.Fatalf("update isRetired = %#v, want true", updateDetails.IsRetired)
	}
	if updateDetails.FreeformTags == nil || len(updateDetails.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want explicit empty map", updateDetails.FreeformTags)
	}
	if got, want := updateDetails.DefinedTags["Operations"]["CostCenter"], "42"; got != want {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want %q", got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
