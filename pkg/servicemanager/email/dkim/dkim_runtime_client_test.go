/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dkim

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	emailsdk "github.com/oracle/oci-go-sdk/v65/email"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDkimID        = "ocid1.dkim.oc1..example"
	testEmailDomainID = "ocid1.emaildomain.oc1..example"
	testDkimName      = "selector-phx-20260427"
)

type fakeDkimOCIClient struct {
	createFn func(context.Context, emailsdk.CreateDkimRequest) (emailsdk.CreateDkimResponse, error)
	getFn    func(context.Context, emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error)
	listFn   func(context.Context, emailsdk.ListDkimsRequest) (emailsdk.ListDkimsResponse, error)
	updateFn func(context.Context, emailsdk.UpdateDkimRequest) (emailsdk.UpdateDkimResponse, error)
	deleteFn func(context.Context, emailsdk.DeleteDkimRequest) (emailsdk.DeleteDkimResponse, error)
}

func (f *fakeDkimOCIClient) CreateDkim(ctx context.Context, req emailsdk.CreateDkimRequest) (emailsdk.CreateDkimResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return emailsdk.CreateDkimResponse{}, nil
}

func (f *fakeDkimOCIClient) GetDkim(ctx context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return emailsdk.GetDkimResponse{}, nil
}

func (f *fakeDkimOCIClient) ListDkims(ctx context.Context, req emailsdk.ListDkimsRequest) (emailsdk.ListDkimsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return emailsdk.ListDkimsResponse{}, nil
}

func (f *fakeDkimOCIClient) UpdateDkim(ctx context.Context, req emailsdk.UpdateDkimRequest) (emailsdk.UpdateDkimResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return emailsdk.UpdateDkimResponse{}, nil
}

func (f *fakeDkimOCIClient) DeleteDkim(ctx context.Context, req emailsdk.DeleteDkimRequest) (emailsdk.DeleteDkimResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return emailsdk.DeleteDkimResponse{}, nil
}

func testDkimClient(fake *fakeDkimOCIClient) defaultDkimServiceClient {
	if fake == nil {
		fake = &fakeDkimOCIClient{}
	}

	hooks := DkimRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*emailv1beta1.Dkim]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*emailv1beta1.Dkim]{},
		StatusHooks:     generatedruntime.StatusHooks[*emailv1beta1.Dkim]{},
		ParityHooks:     generatedruntime.ParityHooks[*emailv1beta1.Dkim]{},
		Async:           generatedruntime.AsyncHooks[*emailv1beta1.Dkim]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*emailv1beta1.Dkim]{},
		Create: runtimeOperationHooks[emailsdk.CreateDkimRequest, emailsdk.CreateDkimResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateDkimDetails", RequestName: "CreateDkimDetails", Contribution: "body", PreferResourceID: false}},
			Call:   fake.CreateDkim,
		},
		Get: runtimeOperationHooks[emailsdk.GetDkimRequest, emailsdk.GetDkimResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DkimId", RequestName: "dkimId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.GetDkim,
		},
		List: runtimeOperationHooks[emailsdk.ListDkimsRequest, emailsdk.ListDkimsResponse]{
			Fields: dkimListFields(),
			Call:   fake.ListDkims,
		},
		Update: runtimeOperationHooks[emailsdk.UpdateDkimRequest, emailsdk.UpdateDkimResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "DkimId", RequestName: "dkimId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateDkimDetails", RequestName: "UpdateDkimDetails", Contribution: "body", PreferResourceID: false},
			},
			Call: fake.UpdateDkim,
		},
		Delete: runtimeOperationHooks[emailsdk.DeleteDkimRequest, emailsdk.DeleteDkimResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DkimId", RequestName: "dkimId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.DeleteDkim,
		},
	}
	applyDkimRuntimeHooks(&hooks)

	manager := &DkimServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
	return defaultDkimServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*emailv1beta1.Dkim](buildDkimGeneratedRuntimeConfig(manager, hooks)),
	}
}

func makeDkimResource() *emailv1beta1.Dkim {
	return &emailv1beta1.Dkim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dkim-sample",
			Namespace: "default",
		},
		Spec: emailv1beta1.DkimSpec{
			EmailDomainId: testEmailDomainID,
			Name:          testDkimName,
			Description:   "DKIM sample",
		},
	}
}

func makeSDKDkim(id, name, emailDomainID, description string, state emailsdk.DkimLifecycleStateEnum) emailsdk.Dkim {
	return emailsdk.Dkim{
		Id:             common.String(id),
		Name:           common.String(name),
		EmailDomainId:  common.String(emailDomainID),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		Description:    common.String(description),
		LifecycleState: state,
	}
}

func makeSDKDkimSummary(id, name, emailDomainID, description string, state emailsdk.DkimLifecycleStateEnum) emailsdk.DkimSummary {
	return emailsdk.DkimSummary{
		Id:             common.String(id),
		Name:           common.String(name),
		EmailDomainId:  common.String(emailDomainID),
		CompartmentId:  common.String("ocid1.compartment.oc1..example"),
		Description:    common.String(description),
		LifecycleState: state,
	}
}

func TestDkimRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := dkimRuntimeSemantics()
	if got == nil {
		t.Fatal("dkimRuntimeSemantics() = nil")
	}
	if got.FormalService != "email" || got.FormalSlug != "dkim" {
		t.Fatalf("formal binding = %s/%s, want email/dkim", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	assertDkimStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertDkimStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertDkimStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertDkimStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"emailDomainId", "name"})
	assertDkimStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "freeformTags"})
	assertDkimStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"emailDomainId", "name"})
}

func TestDkimServiceClientCreatesAndProjectsActiveStatus(t *testing.T) {
	t.Parallel()

	resource := makeDkimResource()
	var createRequest emailsdk.CreateDkimRequest
	getCalls := 0
	listCalls := 0

	client := testDkimClient(&fakeDkimOCIClient{
		listFn: func(_ context.Context, req emailsdk.ListDkimsRequest) (emailsdk.ListDkimsResponse, error) {
			listCalls++
			requireDkimStringPtr(t, "list emailDomainId", req.EmailDomainId, testEmailDomainID)
			requireDkimStringPtr(t, "list name", req.Name, testDkimName)
			if req.LifecycleState != "" {
				t.Fatalf("list lifecycleState = %q, want empty", req.LifecycleState)
			}
			return emailsdk.ListDkimsResponse{}, nil
		},
		createFn: func(_ context.Context, req emailsdk.CreateDkimRequest) (emailsdk.CreateDkimResponse, error) {
			createRequest = req
			return emailsdk.CreateDkimResponse{
				Dkim:             makeSDKDkim(testDkimID, testDkimName, testEmailDomainID, resource.Spec.Description, emailsdk.DkimLifecycleStateCreating),
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				ContentLocation:  common.String("https://example.invalid/dkim"),
				Location:         common.String("https://example.invalid/dkim"),
			}, nil
		},
		getFn: func(_ context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
			getCalls++
			requireDkimStringPtr(t, "get dkimId", req.DkimId, testDkimID)
			return emailsdk.GetDkimResponse{
				Dkim: makeSDKDkim(testDkimID, testDkimName, testEmailDomainID, resource.Spec.Description, emailsdk.DkimLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListDkims() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDkim() calls = %d, want 1 follow-up read", getCalls)
	}
	requireDkimStringPtr(t, "create emailDomainId", createRequest.CreateDkimDetails.EmailDomainId, testEmailDomainID)
	requireDkimStringPtr(t, "create name", createRequest.CreateDkimDetails.Name, testDkimName)
	requireDkimStringPtr(t, "create description", createRequest.CreateDkimDetails.Description, resource.Spec.Description)
	if got := string(resource.Status.OsokStatus.Ocid); got != testDkimID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDkimID)
	}
	if got := resource.Status.Id; got != testDkimID {
		t.Fatalf("status.id = %q, want %q", got, testDkimID)
	}
	if got := resource.Status.LifecycleState; got != string(emailsdk.DkimLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := lastDkimConditionType(t, resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestDkimServiceClientCreatesWithoutListWhenSelectorIsGenerated(t *testing.T) {
	t.Parallel()

	resource := makeDkimResource()
	resource.Spec.Name = ""
	const generatedName = "generated-selector"

	client := testDkimClient(&fakeDkimOCIClient{
		listFn: func(context.Context, emailsdk.ListDkimsRequest) (emailsdk.ListDkimsResponse, error) {
			t.Fatal("ListDkims() should not run when spec.name is empty")
			return emailsdk.ListDkimsResponse{}, nil
		},
		createFn: func(_ context.Context, req emailsdk.CreateDkimRequest) (emailsdk.CreateDkimResponse, error) {
			requireDkimStringPtr(t, "create emailDomainId", req.CreateDkimDetails.EmailDomainId, testEmailDomainID)
			if req.CreateDkimDetails.Name != nil {
				t.Fatalf("create name = %v, want nil so OCI generates the selector", req.CreateDkimDetails.Name)
			}
			return emailsdk.CreateDkimResponse{
				Dkim:         makeSDKDkim(testDkimID, generatedName, testEmailDomainID, resource.Spec.Description, emailsdk.DkimLifecycleStateCreating),
				OpcRequestId: common.String("opc-create-generated"),
			}, nil
		},
		getFn: func(_ context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
			requireDkimStringPtr(t, "get dkimId", req.DkimId, testDkimID)
			return emailsdk.GetDkimResponse{
				Dkim: makeSDKDkim(testDkimID, generatedName, testEmailDomainID, resource.Spec.Description, emailsdk.DkimLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if got := resource.Status.Name; got != generatedName {
		t.Fatalf("status.name = %q, want generated selector %q", got, generatedName)
	}
}

func TestDkimServiceClientBindsExistingByEmailDomainAndName(t *testing.T) {
	t.Parallel()

	resource := makeDkimResource()
	createCalled := false
	getCalls := 0

	client := testDkimClient(&fakeDkimOCIClient{
		listFn: func(_ context.Context, req emailsdk.ListDkimsRequest) (emailsdk.ListDkimsResponse, error) {
			requireDkimStringPtr(t, "list emailDomainId", req.EmailDomainId, testEmailDomainID)
			requireDkimStringPtr(t, "list name", req.Name, testDkimName)
			return emailsdk.ListDkimsResponse{
				DkimCollection: emailsdk.DkimCollection{
					Items: []emailsdk.DkimSummary{
						makeSDKDkimSummary("ocid1.dkim.oc1..other", "other", testEmailDomainID, "other", emailsdk.DkimLifecycleStateActive),
						makeSDKDkimSummary(testDkimID, testDkimName, testEmailDomainID, resource.Spec.Description, emailsdk.DkimLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
			getCalls++
			requireDkimStringPtr(t, "get dkimId", req.DkimId, testDkimID)
			return emailsdk.GetDkimResponse{
				Dkim: makeSDKDkim(testDkimID, testDkimName, testEmailDomainID, resource.Spec.Description, emailsdk.DkimLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, emailsdk.CreateDkimRequest) (emailsdk.CreateDkimResponse, error) {
			createCalled = true
			return emailsdk.CreateDkimResponse{}, nil
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
		t.Fatal("CreateDkim() should not be called when ListDkims finds a matching DKIM")
	}
	if getCalls != 1 {
		t.Fatalf("GetDkim() calls = %d, want 1 live follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDkimID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDkimID)
	}
}

func TestDkimServiceClientUpdatesMutableDescription(t *testing.T) {
	t.Parallel()

	resource := makeDkimResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDkimID)
	resource.Status.Id = testDkimID
	resource.Spec.Description = "Updated DKIM description"
	resource.Spec.FreeformTags = map[string]string{"managed-by": "osok"}
	var updateRequest emailsdk.UpdateDkimRequest
	getCalls := 0
	updateCalls := 0

	client := testDkimClient(&fakeDkimOCIClient{
		getFn: func(_ context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
			getCalls++
			requireDkimStringPtr(t, "get dkimId", req.DkimId, testDkimID)
			description := "DKIM sample"
			freeformTags := map[string]string{"managed-by": "legacy"}
			if getCalls > 1 {
				description = resource.Spec.Description
				freeformTags = resource.Spec.FreeformTags
			}
			dkim := makeSDKDkim(testDkimID, testDkimName, testEmailDomainID, description, emailsdk.DkimLifecycleStateActive)
			dkim.FreeformTags = freeformTags
			return emailsdk.GetDkimResponse{
				Dkim: dkim,
			}, nil
		},
		updateFn: func(_ context.Context, req emailsdk.UpdateDkimRequest) (emailsdk.UpdateDkimResponse, error) {
			updateCalls++
			updateRequest = req
			return emailsdk.UpdateDkimResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue update", response)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDkim() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetDkim() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	requireDkimStringPtr(t, "update dkimId", updateRequest.DkimId, testDkimID)
	requireDkimStringPtr(t, "update description", updateRequest.UpdateDkimDetails.Description, resource.Spec.Description)
	if got := updateRequest.UpdateDkimDetails.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("update freeformTags[managed-by] = %q, want osok", got)
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
	if got := resource.Status.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("status.freeformTags[managed-by] = %q, want osok", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestDkimServiceClientRejectsSelectorDriftBeforeMutation(t *testing.T) {
	t.Parallel()

	resource := makeDkimResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDkimID)
	resource.Status.Id = testDkimID
	resource.Spec.Name = "replacement-selector"
	updateCalled := false

	client := testDkimClient(&fakeDkimOCIClient{
		getFn: func(_ context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
			requireDkimStringPtr(t, "get dkimId", req.DkimId, testDkimID)
			return emailsdk.GetDkimResponse{
				Dkim: makeSDKDkim(testDkimID, testDkimName, testEmailDomainID, resource.Spec.Description, emailsdk.DkimLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, emailsdk.UpdateDkimRequest) (emailsdk.UpdateDkimResponse, error) {
			updateCalled = true
			return emailsdk.UpdateDkimResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required selector drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateDkim() should not be called when selector drift requires replacement")
	}
}

func TestDkimServiceClientClassifiesLifecycleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		state         emailsdk.DkimLifecycleStateEnum
		wantSuccess   bool
		wantRequeue   bool
		wantCondition shared.OSOKConditionType
	}{
		{name: "active", state: emailsdk.DkimLifecycleStateActive, wantSuccess: true, wantCondition: shared.Active},
		{name: "creating", state: emailsdk.DkimLifecycleStateCreating, wantSuccess: true, wantRequeue: true, wantCondition: shared.Provisioning},
		{name: "updating", state: emailsdk.DkimLifecycleStateUpdating, wantSuccess: true, wantRequeue: true, wantCondition: shared.Updating},
		{name: "deleting", state: emailsdk.DkimLifecycleStateDeleting, wantSuccess: true, wantRequeue: true, wantCondition: shared.Terminating},
		{name: "failed", state: emailsdk.DkimLifecycleStateFailed, wantCondition: shared.Failed},
		{name: "needs attention", state: emailsdk.DkimLifecycleStateNeedsAttention, wantCondition: shared.Failed},
		{name: "inactive", state: emailsdk.DkimLifecycleStateInactive, wantCondition: shared.Failed},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeDkimResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(testDkimID)
			resource.Status.Id = testDkimID
			client := testDkimClient(&fakeDkimOCIClient{
				getFn: func(_ context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
					requireDkimStringPtr(t, "get dkimId", req.DkimId, testDkimID)
					return emailsdk.GetDkimResponse{
						Dkim: makeSDKDkim(testDkimID, testDkimName, testEmailDomainID, resource.Spec.Description, tt.state),
					}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tt.wantSuccess {
				t.Fatalf("CreateOrUpdate() success = %t, want %t", response.IsSuccessful, tt.wantSuccess)
			}
			if response.ShouldRequeue != tt.wantRequeue {
				t.Fatalf("CreateOrUpdate() shouldRequeue = %t, want %t", response.ShouldRequeue, tt.wantRequeue)
			}
			if got := lastDkimConditionType(t, resource); got != tt.wantCondition {
				t.Fatalf("last condition = %q, want %q", got, tt.wantCondition)
			}
		})
	}
}

func TestDkimServiceClientDeleteRetainsUntilTerminalConfirmation(t *testing.T) {
	t.Parallel()

	resource := makeDkimResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDkimID)
	resource.Status.Id = testDkimID
	getCalls := 0
	deleteCalls := 0

	client := testDkimClient(&fakeDkimOCIClient{
		getFn: func(_ context.Context, req emailsdk.GetDkimRequest) (emailsdk.GetDkimResponse, error) {
			getCalls++
			requireDkimStringPtr(t, "get dkimId", req.DkimId, testDkimID)
			state := emailsdk.DkimLifecycleStateActive
			if getCalls == 2 {
				state = emailsdk.DkimLifecycleStateDeleting
			}
			if getCalls > 2 {
				state = emailsdk.DkimLifecycleStateDeleted
			}
			return emailsdk.GetDkimResponse{
				Dkim: makeSDKDkim(testDkimID, testDkimName, testEmailDomainID, resource.Spec.Description, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req emailsdk.DeleteDkimRequest) (emailsdk.DeleteDkimResponse, error) {
			deleteCalls++
			requireDkimStringPtr(t, "delete dkimId", req.DkimId, testDkimID)
			return emailsdk.DeleteDkimResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call = true, want finalizer retained while DKIM is DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDkim() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.LifecycleState; got != string(emailsdk.DkimLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", got)
	}
	if got := lastDkimConditionType(t, resource); got != shared.Terminating {
		t.Fatalf("last condition after first delete = %q, want Terminating", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call = false, want terminal delete confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDkim() calls after second delete = %d, want still 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetDkim() calls = %d, want 3", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.LifecycleState; got != string(emailsdk.DkimLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState after second delete = %q, want DELETED", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func requireDkimStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func lastDkimConditionType(t *testing.T, resource *emailv1beta1.Dkim) shared.OSOKConditionType {
	t.Helper()
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		t.Fatal("status.status.conditions is empty")
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}

func assertDkimStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}
