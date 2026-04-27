/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package suppression

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	emailsdk "github.com/oracle/oci-go-sdk/v65/email"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSuppressionID    = "ocid1.suppression.oc1..example"
	testCompartmentID    = "ocid1.tenancy.oc1..example"
	testSuppressionEmail = "recipient@example.com"
)

type suppressionOCIClient interface {
	CreateSuppression(context.Context, emailsdk.CreateSuppressionRequest) (emailsdk.CreateSuppressionResponse, error)
	GetSuppression(context.Context, emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error)
	ListSuppressions(context.Context, emailsdk.ListSuppressionsRequest) (emailsdk.ListSuppressionsResponse, error)
	DeleteSuppression(context.Context, emailsdk.DeleteSuppressionRequest) (emailsdk.DeleteSuppressionResponse, error)
}

type fakeSuppressionOCIClient struct {
	createFn func(context.Context, emailsdk.CreateSuppressionRequest) (emailsdk.CreateSuppressionResponse, error)
	getFn    func(context.Context, emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error)
	listFn   func(context.Context, emailsdk.ListSuppressionsRequest) (emailsdk.ListSuppressionsResponse, error)
	deleteFn func(context.Context, emailsdk.DeleteSuppressionRequest) (emailsdk.DeleteSuppressionResponse, error)
}

func (f *fakeSuppressionOCIClient) CreateSuppression(ctx context.Context, req emailsdk.CreateSuppressionRequest) (emailsdk.CreateSuppressionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return emailsdk.CreateSuppressionResponse{}, nil
}

func (f *fakeSuppressionOCIClient) GetSuppression(ctx context.Context, req emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return emailsdk.GetSuppressionResponse{}, nil
}

func (f *fakeSuppressionOCIClient) ListSuppressions(ctx context.Context, req emailsdk.ListSuppressionsRequest) (emailsdk.ListSuppressionsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return emailsdk.ListSuppressionsResponse{}, nil
}

func (f *fakeSuppressionOCIClient) DeleteSuppression(ctx context.Context, req emailsdk.DeleteSuppressionRequest) (emailsdk.DeleteSuppressionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return emailsdk.DeleteSuppressionResponse{}, nil
}

func TestSuppressionRuntimeSemanticsEncodesStateFreeContract(t *testing.T) {
	t.Parallel()

	got := newSuppressionRuntimeSemantics()
	if got == nil {
		t.Fatal("newSuppressionRuntimeSemantics() = nil")
	}
	if got.FormalService != "email" || got.FormalSlug != "suppression" {
		t.Fatalf("formal binding = %s/%s, want email/suppression", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "none" || got.Async.FormalClassification != "none" {
		t.Fatalf("async semantics = %#v, want explicit state-free none semantics", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertSuppressionStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "emailAddress"})
	assertSuppressionStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "emailAddress"})
	if len(got.Mutation.Mutable) != 0 {
		t.Fatalf("Mutation.Mutable = %#v, want no mutable fields because OCI Suppression has no update API", got.Mutation.Mutable)
	}
}

func TestSuppressionServiceClientCreatesAndProjectsStateFreeActive(t *testing.T) {
	t.Parallel()

	resource := makeSuppressionResource()
	resource.Spec.CompartmentId = " " + testCompartmentID + " "
	resource.Spec.EmailAddress = "Recipient@Example.COM"
	getCalls := 0
	listCalls := 0
	var createRequest emailsdk.CreateSuppressionRequest

	client := newTestSuppressionClient(&fakeSuppressionOCIClient{
		listFn: func(_ context.Context, req emailsdk.ListSuppressionsRequest) (emailsdk.ListSuppressionsResponse, error) {
			listCalls++
			requireStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list emailAddress", req.EmailAddress, testSuppressionEmail)
			return emailsdk.ListSuppressionsResponse{}, nil
		},
		createFn: func(_ context.Context, req emailsdk.CreateSuppressionRequest) (emailsdk.CreateSuppressionResponse, error) {
			createRequest = req
			return emailsdk.CreateSuppressionResponse{
				Suppression:  makeSDKSuppression(testSuppressionID, resource),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error) {
			getCalls++
			requireStringPtr(t, "get suppressionId", req.SuppressionId, testSuppressionID)
			return emailsdk.GetSuppressionResponse{Suppression: makeSDKSuppression(testSuppressionID, resource)}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue state-free create", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListSuppressions() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetSuppression() calls = %d, want 1 follow-up read", getCalls)
	}
	requireStringPtr(t, "create compartmentId", createRequest.CreateSuppressionDetails.CompartmentId, testCompartmentID)
	requireStringPtr(t, "create emailAddress", createRequest.CreateSuppressionDetails.EmailAddress, testSuppressionEmail)
	if got := resource.Spec.EmailAddress; got != testSuppressionEmail {
		t.Fatalf("spec.emailAddress = %q, want normalized %q", got, testSuppressionEmail)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testSuppressionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSuppressionID)
	}
	if got := resource.Status.Id; got != testSuppressionID {
		t.Fatalf("status.id = %q, want %q", got, testSuppressionID)
	}
	if got := resource.Status.EmailAddress; got != testSuppressionEmail {
		t.Fatalf("status.emailAddress = %q, want %q", got, testSuppressionEmail)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := lastSuppressionConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestSuppressionServiceClientBindsExistingByCompartmentAndEmail(t *testing.T) {
	t.Parallel()

	resource := makeSuppressionResource()
	createCalled := false
	getCalls := 0

	client := newTestSuppressionClient(&fakeSuppressionOCIClient{
		listFn: func(_ context.Context, req emailsdk.ListSuppressionsRequest) (emailsdk.ListSuppressionsResponse, error) {
			requireStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list emailAddress", req.EmailAddress, testSuppressionEmail)
			other := makeSuppressionResource()
			other.Spec.EmailAddress = "other@example.com"
			return emailsdk.ListSuppressionsResponse{
				Items: []emailsdk.SuppressionSummary{
					makeSDKSuppressionSummary("ocid1.suppression.oc1..other", other),
					makeSDKSuppressionSummary(testSuppressionID, resource),
				},
			}, nil
		},
		getFn: func(_ context.Context, req emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error) {
			getCalls++
			requireStringPtr(t, "get suppressionId", req.SuppressionId, testSuppressionID)
			return emailsdk.GetSuppressionResponse{Suppression: makeSDKSuppression(testSuppressionID, resource)}, nil
		},
		createFn: func(context.Context, emailsdk.CreateSuppressionRequest) (emailsdk.CreateSuppressionResponse, error) {
			createCalled = true
			return emailsdk.CreateSuppressionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if createCalled {
		t.Fatal("CreateSuppression() should not be called when list lookup finds a matching suppression")
	}
	if getCalls != 1 {
		t.Fatalf("GetSuppression() calls = %d, want 1 live follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testSuppressionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSuppressionID)
	}
}

func TestSuppressionServiceClientRejectsCreateOnlyDriftBeforeMutation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mutateDesired func(*emailv1beta1.Suppression)
		mutateCurrent func(*emailsdk.Suppression)
		wantField     string
	}{
		{
			name: "compartment",
			mutateDesired: func(resource *emailv1beta1.Suppression) {
				resource.Spec.CompartmentId = "ocid1.tenancy.oc1..new"
			},
			mutateCurrent: func(current *emailsdk.Suppression) {
				current.CompartmentId = common.String(testCompartmentID)
			},
			wantField: "compartmentId",
		},
		{
			name: "email",
			mutateDesired: func(resource *emailv1beta1.Suppression) {
				resource.Spec.EmailAddress = "new-recipient@example.com"
			},
			mutateCurrent: func(current *emailsdk.Suppression) {
				current.EmailAddress = common.String(testSuppressionEmail)
			},
			wantField: "emailAddress",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeSuppressionResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(testSuppressionID)
			resource.Status.Id = testSuppressionID
			tt.mutateDesired(resource)
			createCalled := false

			client := newTestSuppressionClient(&fakeSuppressionOCIClient{
				getFn: func(_ context.Context, req emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error) {
					requireStringPtr(t, "get suppressionId", req.SuppressionId, testSuppressionID)
					current := makeSDKSuppression(testSuppressionID, makeSuppressionResource())
					tt.mutateCurrent(&current)
					return emailsdk.GetSuppressionResponse{Suppression: current}, nil
				},
				createFn: func(context.Context, emailsdk.CreateSuppressionRequest) (emailsdk.CreateSuppressionResponse, error) {
					createCalled = true
					return emailsdk.CreateSuppressionResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil || !strings.Contains(err.Error(), "require replacement when "+tt.wantField+" changes") {
				t.Fatalf("CreateOrUpdate() error = %v, want %s force-new rejection", err, tt.wantField)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
			}
			if createCalled {
				t.Fatal("CreateSuppression() should not be called after create-only drift rejection")
			}
			if got := lastSuppressionConditionType(resource); got != shared.Failed {
				t.Fatalf("last condition = %q, want Failed", got)
			}
		})
	}
}

func TestSuppressionServiceClientDeleteRetainsFinalizerUntilReadNotFound(t *testing.T) {
	t.Parallel()

	resource := makeSuppressionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSuppressionID)
	resource.Status.Id = testSuppressionID
	getCalls := 0
	deleteCalls := 0

	client := newTestSuppressionClient(&fakeSuppressionOCIClient{
		getFn: func(_ context.Context, req emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error) {
			getCalls++
			requireStringPtr(t, "get suppressionId", req.SuppressionId, testSuppressionID)
			if getCalls >= 4 {
				return emailsdk.GetSuppressionResponse{}, errortest.NewServiceError(404, "NotFound", "Suppression deleted")
			}
			return emailsdk.GetSuppressionResponse{Suppression: makeSDKSuppression(testSuppressionID, resource)}, nil
		},
		deleteFn: func(_ context.Context, req emailsdk.DeleteSuppressionRequest) (emailsdk.DeleteSuppressionResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete suppressionId", req.SuppressionId, testSuppressionID)
			return emailsdk.DeleteSuppressionResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback still finds the suppression")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteSuppression() calls = %d, want 1", deleteCalls)
	}
	if got := lastSuppressionConditionType(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete ||
		resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending delete", resource.Status.OsokStatus.Async.Current)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("second Delete() deleted = true, want finalizer retained while readback still finds the suppression")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteSuppression() calls after second delete = %d, want still 1", deleteCalls)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("third Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("third Delete() deleted = false, want finalizer release after NotFound confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteSuppression() calls after final confirmation = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp after confirmation")
	}
}

func newTestSuppressionClient(client suppressionOCIClient) SuppressionServiceClient {
	if client == nil {
		client = &fakeSuppressionOCIClient{}
	}
	hooks := newSuppressionRuntimeHooksWithOCIClient(client)
	applySuppressionRuntimeHooks(&hooks)
	delegate := defaultSuppressionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*emailv1beta1.Suppression](
			buildSuppressionGeneratedRuntimeConfig(
				&SuppressionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}},
				hooks,
			),
		),
	}
	return wrapSuppressionGeneratedClient(hooks, delegate)
}

func newSuppressionRuntimeHooksWithOCIClient(client suppressionOCIClient) SuppressionRuntimeHooks {
	return SuppressionRuntimeHooks{
		Create: runtimeOperationHooks[emailsdk.CreateSuppressionRequest, emailsdk.CreateSuppressionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateSuppressionDetails", RequestName: "CreateSuppressionDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request emailsdk.CreateSuppressionRequest) (emailsdk.CreateSuppressionResponse, error) {
				return client.CreateSuppression(ctx, request)
			},
		},
		Get: runtimeOperationHooks[emailsdk.GetSuppressionRequest, emailsdk.GetSuppressionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SuppressionId", RequestName: "suppressionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request emailsdk.GetSuppressionRequest) (emailsdk.GetSuppressionResponse, error) {
				return client.GetSuppression(ctx, request)
			},
		},
		List: runtimeOperationHooks[emailsdk.ListSuppressionsRequest, emailsdk.ListSuppressionsResponse]{
			Fields: suppressionListFields(),
			Call: func(ctx context.Context, request emailsdk.ListSuppressionsRequest) (emailsdk.ListSuppressionsResponse, error) {
				return client.ListSuppressions(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[emailsdk.DeleteSuppressionRequest, emailsdk.DeleteSuppressionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "SuppressionId", RequestName: "suppressionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request emailsdk.DeleteSuppressionRequest) (emailsdk.DeleteSuppressionResponse, error) {
				return client.DeleteSuppression(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SuppressionServiceClient) SuppressionServiceClient{},
	}
}

func makeSuppressionResource() *emailv1beta1.Suppression {
	return &emailv1beta1.Suppression{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-suppression", Namespace: "default"},
		Spec: emailv1beta1.SuppressionSpec{
			CompartmentId: testCompartmentID,
			EmailAddress:  testSuppressionEmail,
		},
	}
}

func makeSDKSuppression(id string, resource *emailv1beta1.Suppression) emailsdk.Suppression {
	return emailsdk.Suppression{
		Id:            common.String(id),
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		EmailAddress:  common.String(normalizeSuppressionEmail(resource.Spec.EmailAddress)),
		Reason:        emailsdk.SuppressionReasonManual,
	}
}

func makeSDKSuppressionSummary(id string, resource *emailv1beta1.Suppression) emailsdk.SuppressionSummary {
	return emailsdk.SuppressionSummary{
		Id:            common.String(id),
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		EmailAddress:  common.String(normalizeSuppressionEmail(resource.Spec.EmailAddress)),
		Reason:        emailsdk.SuppressionSummaryReasonManual,
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

func lastSuppressionConditionType(resource *emailv1beta1.Suppression) shared.OSOKConditionType {
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return ""
	}
	return conditions[len(conditions)-1].Type
}

func assertSuppressionStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}
