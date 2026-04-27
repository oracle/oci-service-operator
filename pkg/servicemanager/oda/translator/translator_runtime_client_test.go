/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package translator

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
	testTranslatorOdaInstanceID = "ocid1.odainstance.oc1..translatorparent"
	testTranslatorID            = "ocid1.translator.oc1..translator"
)

type fakeTranslatorOCIClient struct {
	createFunc func(context.Context, odasdk.CreateTranslatorRequest) (odasdk.CreateTranslatorResponse, error)
	getFunc    func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error)
	listFunc   func(context.Context, odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error)
	updateFunc func(context.Context, odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error)
	deleteFunc func(context.Context, odasdk.DeleteTranslatorRequest) (odasdk.DeleteTranslatorResponse, error)

	createRequests []odasdk.CreateTranslatorRequest
	getRequests    []odasdk.GetTranslatorRequest
	listRequests   []odasdk.ListTranslatorsRequest
	updateRequests []odasdk.UpdateTranslatorRequest
	deleteRequests []odasdk.DeleteTranslatorRequest
}

func (f *fakeTranslatorOCIClient) CreateTranslator(ctx context.Context, request odasdk.CreateTranslatorRequest) (odasdk.CreateTranslatorResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return odasdk.CreateTranslatorResponse{}, nil
}

func (f *fakeTranslatorOCIClient) GetTranslator(ctx context.Context, request odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return odasdk.GetTranslatorResponse{}, nil
}

func (f *fakeTranslatorOCIClient) ListTranslators(ctx context.Context, request odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return odasdk.ListTranslatorsResponse{}, nil
}

func (f *fakeTranslatorOCIClient) UpdateTranslator(ctx context.Context, request odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return odasdk.UpdateTranslatorResponse{}, nil
}

func (f *fakeTranslatorOCIClient) DeleteTranslator(ctx context.Context, request odasdk.DeleteTranslatorRequest) (odasdk.DeleteTranslatorResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return odasdk.DeleteTranslatorResponse{}, nil
}

func TestTranslatorRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	hooks := newTranslatorRuntimeHooksWithOCIClient(&fakeTranslatorOCIClient{})
	applyTranslatorRuntimeHooks(
		&TranslatorServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
		&hooks,
	)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics is nil")
	}
	if got := hooks.Semantics.FormalService; got != "oda" {
		t.Fatalf("FormalService = %q, want oda", got)
	}
	if got := hooks.Semantics.FormalSlug; got != "translator" {
		t.Fatalf("FormalSlug = %q, want translator", got)
	}
	if got := hooks.Semantics.Async.Runtime; got != "handwritten" {
		t.Fatalf("Async.Runtime = %q, want handwritten", got)
	}
	assertTranslatorStringSliceEqual(t, "Lifecycle.ProvisioningStates", hooks.Semantics.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertTranslatorStringSliceEqual(t, "Lifecycle.UpdatingStates", hooks.Semantics.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertTranslatorStringSliceEqual(t, "Lifecycle.ActiveStates", hooks.Semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertTranslatorStringSliceEqual(t, "Delete.PendingStates", hooks.Semantics.Delete.PendingStates, []string{"DELETING"})
	assertTranslatorStringSliceEqual(t, "Delete.TerminalStates", hooks.Semantics.Delete.TerminalStates, []string{"DELETED"})
	assertTranslatorStringSliceContains(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "baseUrl")
	assertTranslatorStringSliceContains(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "type")
	if len(hooks.Semantics.Unsupported) != 1 || hooks.Semantics.Unsupported[0].Category != "authToken-update-drift" {
		t.Fatalf("Unsupported = %#v, want authToken-update-drift entry", hooks.Semantics.Unsupported)
	}
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want 1", len(hooks.WrapGeneratedClient))
	}
}

func TestTranslatorRequiresOdaInstanceAnnotation(t *testing.T) {
	resource := makeTranslatorResource()
	resource.Annotations = nil
	fake := &fakeTranslatorOCIClient{
		createFunc: func(context.Context, odasdk.CreateTranslatorRequest) (odasdk.CreateTranslatorResponse, error) {
			t.Fatal("CreateTranslator should not be called without parent annotation")
			return odasdk.CreateTranslatorResponse{}, nil
		},
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), translatorOdaInstanceIDAnnotation) {
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

func TestTranslatorCreateProjectsStatus(t *testing.T) {
	resource := makeTranslatorResource()
	resource.Spec.Properties = map[string]string{"model": "nmt"}
	resource.Spec.FreeformTags = map[string]string{"managed-by": "osok"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	fake := &fakeTranslatorOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error) {
		return odasdk.ListTranslatorsResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request odasdk.CreateTranslatorRequest) (odasdk.CreateTranslatorResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testTranslatorOdaInstanceID {
			t.Fatalf("create odaInstanceId = %q, want %q", got, testTranslatorOdaInstanceID)
		}
		if got := request.CreateTranslatorDetails.Type; got != odasdk.TranslationServiceGoogle {
			t.Fatalf("create type = %q, want GOOGLE", got)
		}
		if got := stringValue(request.CreateTranslatorDetails.BaseUrl); got != resource.Spec.BaseUrl {
			t.Fatalf("create baseUrl = %q, want %q", got, resource.Spec.BaseUrl)
		}
		if got := stringValue(request.CreateTranslatorDetails.AuthToken); got != resource.Spec.AuthToken {
			t.Fatalf("create authToken = %q, want spec token", got)
		}
		if got := request.CreateTranslatorDetails.Properties["model"]; got != "nmt" {
			t.Fatalf("create property model = %q, want nmt", got)
		}
		if got := request.CreateTranslatorDetails.FreeformTags["managed-by"]; got != "osok" {
			t.Fatalf("create freeform managed-by = %q, want osok", got)
		}
		if got := request.CreateTranslatorDetails.DefinedTags["Operations"]["CostCenter"]; got != "42" {
			t.Fatalf("create defined tag CostCenter = %#v, want 42", got)
		}
		return odasdk.CreateTranslatorResponse{
			Translator:   makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateCreating),
			OpcRequestId: common.String("create-request"),
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		return odasdk.GetTranslatorResponse{
			Translator: makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive),
		}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	assertTranslatorActiveStatus(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "create-request" {
		t.Fatalf("opcRequestId = %q, want create-request", got)
	}
}

func TestTranslatorBindsExistingWithoutCreate(t *testing.T) {
	resource := makeTranslatorResource()
	fake := &fakeTranslatorOCIClient{}
	fake.listFunc = func(_ context.Context, request odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testTranslatorOdaInstanceID {
			t.Fatalf("list odaInstanceId = %q, want %q", got, testTranslatorOdaInstanceID)
		}
		if got := request.Type; got != odasdk.ListTranslatorsTypeGoogle {
			t.Fatalf("list type = %q, want GOOGLE", got)
		}
		return odasdk.ListTranslatorsResponse{
			TranslatorCollection: odasdk.TranslatorCollection{
				Items: []odasdk.TranslatorSummary{
					makeSDKTranslatorSummary(testTranslatorID, resource, odasdk.LifecycleStateActive),
				},
			},
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		return odasdk.GetTranslatorResponse{
			Translator: makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive),
		}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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
	assertTranslatorActiveStatus(t, resource)
}

func TestTranslatorBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeTranslatorResource()
	fake := &fakeTranslatorOCIClient{}
	fake.listFunc = func(_ context.Context, request odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testTranslatorOdaInstanceID {
			t.Fatalf("list odaInstanceId = %q, want %q", got, testTranslatorOdaInstanceID)
		}
		if got := request.Type; got != odasdk.ListTranslatorsTypeGoogle {
			t.Fatalf("list type = %q, want GOOGLE", got)
		}
		switch len(fake.listRequests) {
		case 1:
			if request.Page != nil {
				t.Fatalf("first list page token = %q, want nil", stringValue(request.Page))
			}
			return odasdk.ListTranslatorsResponse{
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			if got := stringValue(request.Page); got != "page-2" {
				t.Fatalf("second list page token = %q, want page-2", got)
			}
			return odasdk.ListTranslatorsResponse{
				TranslatorCollection: odasdk.TranslatorCollection{
					Items: []odasdk.TranslatorSummary{
						makeSDKTranslatorSummary(testTranslatorID, resource, odasdk.LifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list request count %d", len(fake.listRequests))
			return odasdk.ListTranslatorsResponse{}, nil
		}
	}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		return odasdk.GetTranslatorResponse{
			Translator: makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive),
		}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	assertTranslatorActiveStatus(t, resource)
}

func TestTranslatorRejectsDuplicateMatchesAcrossListPages(t *testing.T) {
	resource := makeTranslatorResource()
	fake := &fakeTranslatorOCIClient{}
	fake.listFunc = func(_ context.Context, request odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error) {
		switch len(fake.listRequests) {
		case 1:
			if request.Page != nil {
				t.Fatalf("first list page token = %q, want nil", stringValue(request.Page))
			}
			return odasdk.ListTranslatorsResponse{
				TranslatorCollection: odasdk.TranslatorCollection{
					Items: []odasdk.TranslatorSummary{
						makeSDKTranslatorSummary(testTranslatorID, resource, odasdk.LifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			if got := stringValue(request.Page); got != "page-2" {
				t.Fatalf("second list page token = %q, want page-2", got)
			}
			return odasdk.ListTranslatorsResponse{
				TranslatorCollection: odasdk.TranslatorCollection{
					Items: []odasdk.TranslatorSummary{
						makeSDKTranslatorSummary("ocid1.translator.oc1..duplicate", resource, odasdk.LifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list request count %d", len(fake.listRequests))
			return odasdk.ListTranslatorsResponse{}, nil
		}
	}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		t.Fatal("GetTranslator should not be called when list resolution finds duplicate matches")
		return odasdk.GetTranslatorResponse{}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "multiple OCI Translators matched type") {
		t.Fatalf("CreateOrUpdate error = %v, want duplicate match error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2", len(fake.listRequests))
	}
	if len(fake.getRequests) != 0 {
		t.Fatalf("get requests = %d, want 0", len(fake.getRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
}

func TestTranslatorUpdatesSupportedMutableDrift(t *testing.T) {
	resource := makeTranslatorResource()
	resource.Status.Id = testTranslatorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTranslatorID)
	resource.Spec.BaseUrl = "https://translation.example.com/new"
	resource.Spec.Properties = map[string]string{"region": "us"}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	current := makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive)
	current.BaseUrl = common.String("https://translation.example.com/old")
	current.Properties = map[string]string{"region": "eu"}
	current.FreeformTags = map[string]string{"env": "dev"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
	updated := makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive)

	fake := &fakeTranslatorOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		switch len(fake.getRequests) {
		case 1:
			return odasdk.GetTranslatorResponse{Translator: current}, nil
		default:
			return odasdk.GetTranslatorResponse{Translator: updated}, nil
		}
	}
	fake.updateFunc = func(_ context.Context, request odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testTranslatorOdaInstanceID {
			t.Fatalf("update odaInstanceId = %q, want %q", got, testTranslatorOdaInstanceID)
		}
		if got := stringValue(request.TranslatorId); got != testTranslatorID {
			t.Fatalf("update translatorId = %q, want %q", got, testTranslatorID)
		}
		return odasdk.UpdateTranslatorResponse{
			Translator:   updated,
			OpcRequestId: common.String("update-request"),
		}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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
	details := fake.updateRequests[0].UpdateTranslatorDetails
	if got := stringValue(details.BaseUrl); got != resource.Spec.BaseUrl {
		t.Fatalf("updated baseUrl = %q, want %q", got, resource.Spec.BaseUrl)
	}
	if got := stringValue(details.AuthToken); got != resource.Spec.AuthToken {
		t.Fatalf("updated authToken = %q, want spec token when another mutable field changed", got)
	}
	if got := details.Properties["region"]; got != "us" {
		t.Fatalf("updated property region = %q, want us", got)
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
	assertTranslatorActiveStatus(t, resource)
}

func TestTranslatorDoesNotLoopOnWriteOnlyAuthToken(t *testing.T) {
	resource := makeTranslatorResource()
	resource.Status.Id = testTranslatorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTranslatorID)
	current := makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive)
	fake := &fakeTranslatorOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		return odasdk.GetTranslatorResponse{Translator: current}, nil
	}
	fake.updateFunc = func(context.Context, odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error) {
		t.Fatal("UpdateTranslator should not be called for authToken-only drift because OCI does not expose the current token")
		return odasdk.UpdateTranslatorResponse{}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestTranslatorRejectsCreateOnlyTypeDriftBeforeUpdate(t *testing.T) {
	resource := makeTranslatorResource()
	resource.Status.Id = testTranslatorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTranslatorID)
	resource.Spec.Type = string(odasdk.TranslationServiceMicrosoft)
	current := makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive)
	current.Type = odasdk.TranslationServiceGoogle
	fake := &fakeTranslatorOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		return odasdk.GetTranslatorResponse{Translator: current}, nil
	}
	fake.updateFunc = func(context.Context, odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error) {
		t.Fatal("UpdateTranslator should not be called for create-only type drift")
		return odasdk.UpdateTranslatorResponse{}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create-only field drift") || !strings.Contains(err.Error(), "type") {
		t.Fatalf("CreateOrUpdate error = %v, want create-only type drift", err)
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

func TestTranslatorClassifiesLifecycleStates(t *testing.T) {
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
		{name: "failed", state: odasdk.LifecycleStateFailed, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassFailed},
		{name: "inactive", state: odasdk.LifecycleStateInactive, wantReason: shared.Failed, wantRequeue: false, wantSuccess: false, wantAsync: true, wantAsyncNorm: shared.OSOKAsyncClassFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := makeTranslatorResource()
			resource.Status.Id = testTranslatorID
			resource.Status.OsokStatus.Ocid = shared.OCID(testTranslatorID)
			fake := &fakeTranslatorOCIClient{}
			fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
				return odasdk.GetTranslatorResponse{
					Translator: makeSDKTranslator(testTranslatorID, resource, tt.state),
				}, nil
			}
			fake.updateFunc = func(context.Context, odasdk.UpdateTranslatorRequest) (odasdk.UpdateTranslatorResponse, error) {
				t.Fatal("UpdateTranslator should not be called for lifecycle classification")
				return odasdk.UpdateTranslatorResponse{}, nil
			}
			client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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

func TestTranslatorDeleteWaitsForDeletingConfirmation(t *testing.T) {
	resource := makeTranslatorResource()
	resource.Status.Id = testTranslatorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTranslatorID)
	fake := &fakeTranslatorOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetTranslatorResponse{
				Translator: makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetTranslatorResponse{
			Translator: makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateDeleting),
		}, nil
	}
	fake.deleteFunc = func(_ context.Context, request odasdk.DeleteTranslatorRequest) (odasdk.DeleteTranslatorResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testTranslatorOdaInstanceID {
			t.Fatalf("delete odaInstanceId = %q, want %q", got, testTranslatorOdaInstanceID)
		}
		if got := stringValue(request.TranslatorId); got != testTranslatorID {
			t.Fatalf("delete translatorId = %q, want %q", got, testTranslatorID)
		}
		return odasdk.DeleteTranslatorResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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

func TestTranslatorDeleteResolvesByListWhenStatusIdentityMissing(t *testing.T) {
	resource := makeTranslatorResource()
	fake := &fakeTranslatorOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListTranslatorsRequest) (odasdk.ListTranslatorsResponse, error) {
		return odasdk.ListTranslatorsResponse{
			TranslatorCollection: odasdk.TranslatorCollection{
				Items: []odasdk.TranslatorSummary{
					makeSDKTranslatorSummary(testTranslatorID, resource, odasdk.LifecycleStateActive),
				},
			},
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetTranslatorResponse{
				Translator: makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetTranslatorResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "translator not found")
	}
	fake.deleteFunc = func(context.Context, odasdk.DeleteTranslatorRequest) (odasdk.DeleteTranslatorResponse, error) {
		return odasdk.DeleteTranslatorResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete returned deleted=false, want true after confirm read not found")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want 1", len(fake.listRequests))
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if got := stringValue(fake.deleteRequests[0].TranslatorId); got != testTranslatorID {
		t.Fatalf("delete translatorId = %q, want %q", got, testTranslatorID)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want timestamp after deletion confirmation")
	}
}

func TestTranslatorDeleteConfirmsReadNotFound(t *testing.T) {
	resource := makeTranslatorResource()
	resource.Status.Id = testTranslatorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTranslatorID)
	fake := &fakeTranslatorOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetTranslatorRequest) (odasdk.GetTranslatorResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetTranslatorResponse{
				Translator: makeSDKTranslator(testTranslatorID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetTranslatorResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "translator not found")
	}
	fake.deleteFunc = func(context.Context, odasdk.DeleteTranslatorRequest) (odasdk.DeleteTranslatorResponse, error) {
		return odasdk.DeleteTranslatorResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newTranslatorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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

func makeTranslatorResource() *odav1beta1.Translator {
	return &odav1beta1.Translator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-translator",
			Namespace: "default",
			Annotations: map[string]string{
				translatorOdaInstanceIDAnnotation: testTranslatorOdaInstanceID,
			},
		},
		Spec: odav1beta1.TranslatorSpec{
			Type:      string(odasdk.TranslationServiceGoogle),
			BaseUrl:   "https://translation.example.com/base",
			AuthToken: "translator-token",
		},
	}
}

func makeSDKTranslator(id string, resource *odav1beta1.Translator, state odasdk.LifecycleStateEnum) odasdk.Translator {
	translatorType, _ := normalizedTranslatorType(resource.Spec.Type)
	return odasdk.Translator{
		Id:             common.String(id),
		Type:           odasdk.TranslationServiceEnum(translatorType),
		Name:           common.String("Google"),
		BaseUrl:        common.String(resource.Spec.BaseUrl),
		LifecycleState: state,
		Properties:     cloneStringMap(resource.Spec.Properties),
		FreeformTags:   cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:    translatorDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKTranslatorSummary(id string, resource *odav1beta1.Translator, state odasdk.LifecycleStateEnum) odasdk.TranslatorSummary {
	translatorType, _ := normalizedTranslatorType(resource.Spec.Type)
	return odasdk.TranslatorSummary{
		Id:             common.String(id),
		Type:           odasdk.TranslationServiceEnum(translatorType),
		Name:           common.String("Google"),
		LifecycleState: state,
	}
}

func assertTranslatorActiveStatus(t *testing.T, resource *odav1beta1.Translator) {
	t.Helper()
	if got := resource.Status.Id; got != testTranslatorID {
		t.Fatalf("status.id = %q, want %q", got, testTranslatorID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testTranslatorID {
		t.Fatalf("status.ocid = %q, want %q", got, testTranslatorID)
	}
	if got := resource.Status.Type; got != string(odasdk.TranslationServiceGoogle) {
		t.Fatalf("status.type = %q, want GOOGLE", got)
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

func assertTranslatorStringSliceEqual(t *testing.T, name string, got []string, want []string) {
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

func assertTranslatorStringSliceContains(t *testing.T, name string, got []string, want string) {
	t.Helper()
	for _, value := range got {
		if value == want {
			return
		}
	}
	t.Fatalf("%s = %#v, want entry %q", name, got, want)
}
