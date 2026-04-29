/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitalassistant

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
	testDigitalAssistantOdaInstanceID = "ocid1.odainstance.oc1..digitalassistantparent"
	testDigitalAssistantID            = "ocid1.digitalassistant.oc1..digitalassistant"
)

type fakeDigitalAssistantOCIClient struct {
	createFunc      func(context.Context, odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error)
	getFunc         func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error)
	listFunc        func(context.Context, odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error)
	updateFunc      func(context.Context, odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error)
	deleteFunc      func(context.Context, odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error)
	workRequestFunc func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error)

	createRequests      []odasdk.CreateDigitalAssistantRequest
	getRequests         []odasdk.GetDigitalAssistantRequest
	listRequests        []odasdk.ListDigitalAssistantsRequest
	updateRequests      []odasdk.UpdateDigitalAssistantRequest
	deleteRequests      []odasdk.DeleteDigitalAssistantRequest
	workRequestRequests []odasdk.GetWorkRequestRequest
}

func (f *fakeDigitalAssistantOCIClient) CreateDigitalAssistant(ctx context.Context, request odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return odasdk.CreateDigitalAssistantResponse{}, nil
}

func (f *fakeDigitalAssistantOCIClient) GetDigitalAssistant(ctx context.Context, request odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return odasdk.GetDigitalAssistantResponse{}, nil
}

func (f *fakeDigitalAssistantOCIClient) ListDigitalAssistants(ctx context.Context, request odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return odasdk.ListDigitalAssistantsResponse{}, nil
}

func (f *fakeDigitalAssistantOCIClient) UpdateDigitalAssistant(ctx context.Context, request odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return odasdk.UpdateDigitalAssistantResponse{}, nil
}

func (f *fakeDigitalAssistantOCIClient) DeleteDigitalAssistant(ctx context.Context, request odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return odasdk.DeleteDigitalAssistantResponse{}, nil
}

func (f *fakeDigitalAssistantOCIClient) GetWorkRequest(ctx context.Context, request odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFunc != nil {
		return f.workRequestFunc(ctx, request)
	}
	return odasdk.GetWorkRequestResponse{}, nil
}

func TestDigitalAssistantRuntimeSemanticsEncodesWorkRequestLifecycleContract(t *testing.T) {
	fake := &fakeDigitalAssistantOCIClient{}
	hooks := newDigitalAssistantRuntimeHooksWithOCIClient(fake)
	applyDigitalAssistantRuntimeHooks(
		&DigitalAssistantServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
		&hooks,
		fake,
		nil,
	)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics is nil")
	}
	if got := hooks.Semantics.FormalService; got != "oda" {
		t.Fatalf("FormalService = %q, want oda", got)
	}
	if got := hooks.Semantics.FormalSlug; got != "digitalassistant" {
		t.Fatalf("FormalSlug = %q, want digitalassistant", got)
	}
	if got := hooks.Semantics.Async.Runtime; got != "handwritten" {
		t.Fatalf("Async.Runtime = %q, want handwritten", got)
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got)
	}
	assertDigitalAssistantStringSliceEqual(t, "Lifecycle.ProvisioningStates", hooks.Semantics.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertDigitalAssistantStringSliceEqual(t, "Lifecycle.UpdatingStates", hooks.Semantics.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertDigitalAssistantStringSliceEqual(t, "Lifecycle.ActiveStates", hooks.Semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertDigitalAssistantStringSliceEqual(t, "Delete.PendingStates", hooks.Semantics.Delete.PendingStates, []string{"DELETING"})
	assertDigitalAssistantStringSliceEqual(t, "Delete.TerminalStates", hooks.Semantics.Delete.TerminalStates, []string{"DELETED"})
	assertDigitalAssistantStringSliceContains(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "description")
	assertDigitalAssistantStringSliceContains(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "jsonData")
	assertDigitalAssistantStringSliceContains(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "displayName")
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want 1", len(hooks.WrapGeneratedClient))
	}
}

func TestDigitalAssistantRequiresOdaInstanceAnnotation(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Annotations = nil
	fake := &fakeDigitalAssistantOCIClient{
		createFunc: func(context.Context, odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error) {
			t.Fatal("CreateDigitalAssistant should not be called without parent annotation")
			return odasdk.CreateDigitalAssistantResponse{}, nil
		},
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), digitalAssistantOdaInstanceIDAnnotation) {
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

func TestDigitalAssistantCreateTracksPendingWorkRequest(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Spec.FreeformTags = map[string]string{"managed-by": "osok"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	fake := &fakeDigitalAssistantOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error) {
		return odasdk.ListDigitalAssistantsResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testDigitalAssistantOdaInstanceID {
			t.Fatalf("create odaInstanceId = %q, want %q", got, testDigitalAssistantOdaInstanceID)
		}
		if request.OpcRetryToken == nil || stringValue(request.OpcRetryToken) == "" {
			t.Fatal("create opcRetryToken is empty, want deterministic token")
		}
		details, ok := request.CreateDigitalAssistantDetails.(odasdk.CreateNewDigitalAssistantDetails)
		if !ok {
			t.Fatalf("create details = %T, want CreateNewDigitalAssistantDetails", request.CreateDigitalAssistantDetails)
		}
		if got := stringValue(details.Name); got != resource.Spec.Name {
			t.Fatalf("create name = %q, want %q", got, resource.Spec.Name)
		}
		if got := stringValue(details.DisplayName); got != resource.Spec.DisplayName {
			t.Fatalf("create displayName = %q, want %q", got, resource.Spec.DisplayName)
		}
		if got := details.FreeformTags["managed-by"]; got != "osok" {
			t.Fatalf("create freeform managed-by = %q, want osok", got)
		}
		if got := details.DefinedTags["Operations"]["CostCenter"]; got != "42" {
			t.Fatalf("create defined tag CostCenter = %#v, want 42", got)
		}
		return odasdk.CreateDigitalAssistantResponse{
			OpcWorkRequestId: common.String("wr-create"),
			OpcRequestId:     common.String("create-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKDigitalAssistantWorkRequest(
				"wr-create",
				odasdk.WorkRequestRequestActionCreateDigitalAssistant,
				odasdk.WorkRequestStatusInProgress,
				"",
			),
		}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful requeue", response)
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want 1", len(fake.listRequests))
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request polls = %d, want 1", len(fake.workRequestRequests))
	}
	if len(fake.getRequests) != 0 {
		t.Fatalf("get requests = %d, want 0 while create work request is pending", len(fake.getRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseCreate || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.current = %#v, want pending create work request", current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "create-request" {
		t.Fatalf("opcRequestId = %q, want create-request", got)
	}
}

func TestDigitalAssistantCreateUsesJsonDataBody(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Spec = odav1beta1.DigitalAssistantSpec{
		JsonData: `{"kind":"NEW","name":"JsonAssistant","displayName":"JSON Assistant","version":"1.0","category":"dev","description":"from json","platformVersion":"22.04","multilingualMode":"native","primaryLanguageTag":"en","nativeLanguageTags":["en","es"],"freeformTags":{"source":"json"},"definedTags":{"Operations":{"CostCenter":"42"}}}`,
	}
	fake := &fakeDigitalAssistantOCIClient{}
	fake.listFunc = func(context.Context, odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error) {
		return odasdk.ListDigitalAssistantsResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request odasdk.CreateDigitalAssistantRequest) (odasdk.CreateDigitalAssistantResponse, error) {
		details, ok := request.CreateDigitalAssistantDetails.(odasdk.CreateNewDigitalAssistantDetails)
		if !ok {
			t.Fatalf("create details = %T, want CreateNewDigitalAssistantDetails", request.CreateDigitalAssistantDetails)
		}
		if got := stringValue(details.Name); got != "JsonAssistant" {
			t.Fatalf("create jsonData name = %q, want JsonAssistant", got)
		}
		if got := stringValue(details.DisplayName); got != "JSON Assistant" {
			t.Fatalf("create jsonData displayName = %q, want JSON Assistant", got)
		}
		if got := details.MultilingualMode; got != odasdk.BotMultilingualModeNative {
			t.Fatalf("create jsonData multilingualMode = %q, want %q", got, odasdk.BotMultilingualModeNative)
		}
		if got := details.FreeformTags["source"]; got != "json" {
			t.Fatalf("create jsonData freeform source = %q, want json", got)
		}
		if got := details.DefinedTags["Operations"]["CostCenter"]; got != "42" {
			t.Fatalf("create jsonData defined tag CostCenter = %#v, want 42", got)
		}
		return odasdk.CreateDigitalAssistantResponse{
			OpcWorkRequestId: common.String("wr-create"),
			OpcRequestId:     common.String("create-request"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKDigitalAssistantWorkRequest(
				"wr-create",
				odasdk.WorkRequestRequestActionCreateDigitalAssistant,
				odasdk.WorkRequestStatusInProgress,
				"",
			),
		}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request polls = %d, want 1", len(fake.workRequestRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "create-request" {
		t.Fatalf("opcRequestId = %q, want create-request", got)
	}
}

func TestDigitalAssistantResumesSucceededCreateWorkRequest(t *testing.T) {
	resource := makeDigitalAssistantResource()
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		RawStatus:       string(odasdk.WorkRequestStatusInProgress),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
	fake := &fakeDigitalAssistantOCIClient{}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKDigitalAssistantWorkRequest(
				"wr-create",
				odasdk.WorkRequestRequestActionCreateDigitalAssistant,
				odasdk.WorkRequestStatusSucceeded,
				testDigitalAssistantID,
			),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testDigitalAssistantOdaInstanceID {
			t.Fatalf("get odaInstanceId = %q, want %q", got, testDigitalAssistantOdaInstanceID)
		}
		if got := stringValue(request.DigitalAssistantId); got != testDigitalAssistantID {
			t.Fatalf("get digitalAssistantId = %q, want %q", got, testDigitalAssistantID)
		}
		return odasdk.GetDigitalAssistantResponse{
			DigitalAssistant: makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive),
		}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 while resuming work request", len(fake.createRequests))
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request polls = %d, want 1", len(fake.workRequestRequests))
	}
	assertDigitalAssistantActiveStatus(t, resource)
}

func TestDigitalAssistantBindsExistingWithoutCreate(t *testing.T) {
	resource := makeDigitalAssistantResource()
	fake := &fakeDigitalAssistantOCIClient{}
	fake.listFunc = func(_ context.Context, request odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testDigitalAssistantOdaInstanceID {
			t.Fatalf("list odaInstanceId = %q, want %q", got, testDigitalAssistantOdaInstanceID)
		}
		if got := stringValue(request.Name); got != resource.Spec.Name {
			t.Fatalf("list name = %q, want %q", got, resource.Spec.Name)
		}
		if got := stringValue(request.Version); got != resource.Spec.Version {
			t.Fatalf("list version = %q, want %q", got, resource.Spec.Version)
		}
		return odasdk.ListDigitalAssistantsResponse{
			DigitalAssistantCollection: odasdk.DigitalAssistantCollection{
				Items: []odasdk.DigitalAssistantSummary{
					makeSDKDigitalAssistantSummary(testDigitalAssistantID, resource, odasdk.LifecycleStateActive),
				},
			},
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		return odasdk.GetDigitalAssistantResponse{
			DigitalAssistant: makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive),
		}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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
	assertDigitalAssistantActiveStatus(t, resource)
}

func TestDigitalAssistantBindsExistingWithJsonDataIdentity(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Spec = odav1beta1.DigitalAssistantSpec{
		JsonData: `{"kind":"NEW","name":"JsonAssistant","displayName":"JSON Assistant","version":"1.0","category":"dev","description":"from json","platformVersion":"22.04","multilingualMode":"native","primaryLanguageTag":"en","nativeLanguageTags":["en","es"],"freeformTags":{"source":"json"},"definedTags":{"Operations":{"CostCenter":"42"}}}`,
	}
	current := makeSDKDigitalAssistantFromJSONFixture(odasdk.LifecycleStateActive, "JSON Assistant")
	fake := &fakeDigitalAssistantOCIClient{}
	fake.listFunc = func(_ context.Context, request odasdk.ListDigitalAssistantsRequest) (odasdk.ListDigitalAssistantsResponse, error) {
		if got := stringValue(request.Name); got != "JsonAssistant" {
			t.Fatalf("list name = %q, want JsonAssistant", got)
		}
		if got := stringValue(request.Version); got != "1.0" {
			t.Fatalf("list version = %q, want 1.0", got)
		}
		return odasdk.ListDigitalAssistantsResponse{
			DigitalAssistantCollection: odasdk.DigitalAssistantCollection{
				Items: []odasdk.DigitalAssistantSummary{
					{
						Id:             common.String(testDigitalAssistantID),
						Name:           common.String("JsonAssistant"),
						Version:        common.String("1.0"),
						DisplayName:    common.String("JSON Assistant"),
						LifecycleState: odasdk.LifecycleStateActive,
					},
				},
			},
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		return odasdk.GetDigitalAssistantResponse{DigitalAssistant: current}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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
	if got := resource.Status.Name; got != "JsonAssistant" {
		t.Fatalf("status.name = %q, want JsonAssistant", got)
	}
	if got := resource.Status.DisplayName; got != "JSON Assistant" {
		t.Fatalf("status.displayName = %q, want JSON Assistant", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Active)
	}
}

func TestDigitalAssistantUpdatesSupportedMutableDrift(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Status.Id = testDigitalAssistantID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalAssistantID)
	resource.Spec.Category = "prod"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	current := makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive)
	current.Category = common.String("dev")
	current.Description = common.String("old description")
	current.FreeformTags = map[string]string{"env": "dev"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "41"}}
	updated := makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive)

	fake := &fakeDigitalAssistantOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		switch len(fake.getRequests) {
		case 1:
			return odasdk.GetDigitalAssistantResponse{DigitalAssistant: current}, nil
		default:
			return odasdk.GetDigitalAssistantResponse{DigitalAssistant: updated}, nil
		}
	}
	fake.updateFunc = func(_ context.Context, request odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testDigitalAssistantOdaInstanceID {
			t.Fatalf("update odaInstanceId = %q, want %q", got, testDigitalAssistantOdaInstanceID)
		}
		if got := stringValue(request.DigitalAssistantId); got != testDigitalAssistantID {
			t.Fatalf("update digitalAssistantId = %q, want %q", got, testDigitalAssistantID)
		}
		return odasdk.UpdateDigitalAssistantResponse{
			DigitalAssistant: updated,
			OpcRequestId:     common.String("update-request"),
		}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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
	details := fake.updateRequests[0].UpdateDigitalAssistantDetails
	if got := stringValue(details.Category); got != resource.Spec.Category {
		t.Fatalf("updated category = %q, want %q", got, resource.Spec.Category)
	}
	if got := stringValue(details.Description); got != resource.Spec.Description {
		t.Fatalf("updated description = %q, want %q", got, resource.Spec.Description)
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
	assertDigitalAssistantActiveStatus(t, resource)
}

func TestDigitalAssistantRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Status.Id = testDigitalAssistantID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalAssistantID)
	resource.Spec.Name = "DifferentAssistant"
	current := makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive)
	current.Name = common.String("TestAssistant")
	fake := &fakeDigitalAssistantOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		return odasdk.GetDigitalAssistantResponse{DigitalAssistant: current}, nil
	}
	fake.updateFunc = func(context.Context, odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
		t.Fatal("UpdateDigitalAssistant should not be called for create-only drift")
		return odasdk.UpdateDigitalAssistantResponse{}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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

func TestDigitalAssistantRejectsJsonDataDriftBeforeActive(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Spec = odav1beta1.DigitalAssistantSpec{
		JsonData: `{"kind":"NEW","name":"JsonAssistant","displayName":"Changed JSON Assistant","version":"1.0","category":"dev","description":"from json","platformVersion":"22.04","multilingualMode":"native","primaryLanguageTag":"en","nativeLanguageTags":["en","es"],"freeformTags":{"source":"json"},"definedTags":{"Operations":{"CostCenter":"42"}}}`,
	}
	resource.Status.Id = testDigitalAssistantID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalAssistantID)
	current := makeSDKDigitalAssistantFromJSONFixture(odasdk.LifecycleStateActive, "JSON Assistant")
	fake := &fakeDigitalAssistantOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		return odasdk.GetDigitalAssistantResponse{DigitalAssistant: current}, nil
	}
	fake.updateFunc = func(context.Context, odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
		t.Fatal("UpdateDigitalAssistant should not be called for jsonData drift")
		return odasdk.UpdateDigitalAssistantResponse{}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create-only field drift") || !strings.Contains(err.Error(), "jsonData.displayName") {
		t.Fatalf("CreateOrUpdate error = %v, want jsonData displayName drift", err)
	}
	if strings.Contains(err.Error(), "jsonData.multilingualMode") {
		t.Fatalf("CreateOrUpdate error = %v, did not want normalized multilingualMode drift", err)
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

func TestDigitalAssistantNormalizesMultilingualModeBeforeCreateOnlyDriftComparison(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Status.Id = testDigitalAssistantID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalAssistantID)
	resource.Spec.MultilingualMode = "native"
	current := makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive)
	current.MultilingualMode = odasdk.BotMultilingualModeNative
	fake := &fakeDigitalAssistantOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		return odasdk.GetDigitalAssistantResponse{DigitalAssistant: current}, nil
	}
	fake.updateFunc = func(context.Context, odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
		t.Fatal("UpdateDigitalAssistant should not be called for normalized multilingualMode")
		return odasdk.UpdateDigitalAssistantResponse{}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Active)
	}
}

func TestDigitalAssistantClassifiesLifecycleStates(t *testing.T) {
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
			resource := makeDigitalAssistantResource()
			resource.Status.Id = testDigitalAssistantID
			resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalAssistantID)
			fake := &fakeDigitalAssistantOCIClient{}
			fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
				return odasdk.GetDigitalAssistantResponse{
					DigitalAssistant: makeSDKDigitalAssistant(testDigitalAssistantID, resource, tt.state),
				}, nil
			}
			fake.updateFunc = func(context.Context, odasdk.UpdateDigitalAssistantRequest) (odasdk.UpdateDigitalAssistantResponse, error) {
				t.Fatal("UpdateDigitalAssistant should not be called for lifecycle classification")
				return odasdk.UpdateDigitalAssistantResponse{}, nil
			}
			client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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

func TestDigitalAssistantDeleteWaitsForDeletingConfirmation(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Status.Id = testDigitalAssistantID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalAssistantID)
	fake := &fakeDigitalAssistantOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetDigitalAssistantResponse{
				DigitalAssistant: makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetDigitalAssistantResponse{
			DigitalAssistant: makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateDeleting),
		}, nil
	}
	fake.deleteFunc = func(_ context.Context, request odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error) {
		if got := stringValue(request.OdaInstanceId); got != testDigitalAssistantOdaInstanceID {
			t.Fatalf("delete odaInstanceId = %q, want %q", got, testDigitalAssistantOdaInstanceID)
		}
		if got := stringValue(request.DigitalAssistantId); got != testDigitalAssistantID {
			t.Fatalf("delete digitalAssistantId = %q, want %q", got, testDigitalAssistantID)
		}
		return odasdk.DeleteDigitalAssistantResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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

func TestDigitalAssistantDeletePreservesPendingCreateWorkRequest(t *testing.T) {
	resource := makeDigitalAssistantResource()
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		RawStatus:       string(odasdk.WorkRequestStatusInProgress),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
	fake := &fakeDigitalAssistantOCIClient{}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKDigitalAssistantWorkRequest(
				"wr-create",
				odasdk.WorkRequestRequestActionCreateDigitalAssistant,
				odasdk.WorkRequestStatusInProgress,
				"",
			),
		}, nil
	}
	fake.deleteFunc = func(context.Context, odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error) {
		t.Fatal("DeleteDigitalAssistant should not be called while create work request is pending")
		return odasdk.DeleteDigitalAssistantResponse{}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Fatal("Delete returned deleted=true, want false while create work request is pending")
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request polls = %d, want 1", len(fake.workRequestRequests))
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0", len(fake.deleteRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseCreate || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.current = %#v, want pending create work request", current)
	}
}

func TestDigitalAssistantDeleteContinuesAfterSucceededCreateWorkRequest(t *testing.T) {
	resource := makeDigitalAssistantResource()
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		RawStatus:       string(odasdk.WorkRequestStatusInProgress),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
	fake := &fakeDigitalAssistantOCIClient{}
	fake.workRequestFunc = func(context.Context, odasdk.GetWorkRequestRequest) (odasdk.GetWorkRequestResponse, error) {
		return odasdk.GetWorkRequestResponse{
			WorkRequest: makeSDKDigitalAssistantWorkRequest(
				"wr-create",
				odasdk.WorkRequestRequestActionCreateDigitalAssistant,
				odasdk.WorkRequestStatusSucceeded,
				testDigitalAssistantID,
			),
		}, nil
	}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		if len(fake.getRequests) <= 2 {
			return odasdk.GetDigitalAssistantResponse{
				DigitalAssistant: makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetDigitalAssistantResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "digital assistant not found")
	}
	fake.deleteFunc = func(_ context.Context, request odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error) {
		if got := stringValue(request.DigitalAssistantId); got != testDigitalAssistantID {
			t.Fatalf("delete digitalAssistantId = %q, want %q", got, testDigitalAssistantID)
		}
		return odasdk.DeleteDigitalAssistantResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete returned deleted=false, want true after create work request resolves and delete is confirmed")
	}
	if len(fake.workRequestRequests) != 1 {
		t.Fatalf("work request polls = %d, want 1", len(fake.workRequestRequests))
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want timestamp after deletion confirmation")
	}
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("async.current = %#v, want nil after delete confirmation", current)
	}
}

func TestDigitalAssistantDeleteConfirmsReadNotFound(t *testing.T) {
	resource := makeDigitalAssistantResource()
	resource.Status.Id = testDigitalAssistantID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalAssistantID)
	fake := &fakeDigitalAssistantOCIClient{}
	fake.getFunc = func(context.Context, odasdk.GetDigitalAssistantRequest) (odasdk.GetDigitalAssistantResponse, error) {
		if len(fake.getRequests) == 1 {
			return odasdk.GetDigitalAssistantResponse{
				DigitalAssistant: makeSDKDigitalAssistant(testDigitalAssistantID, resource, odasdk.LifecycleStateActive),
			}, nil
		}
		return odasdk.GetDigitalAssistantResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "digital assistant not found")
	}
	fake.deleteFunc = func(context.Context, odasdk.DeleteDigitalAssistantRequest) (odasdk.DeleteDigitalAssistantResponse, error) {
		return odasdk.DeleteDigitalAssistantResponse{OpcRequestId: common.String("delete-request")}, nil
	}
	client := newDigitalAssistantServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)

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

func makeDigitalAssistantResource() *odav1beta1.DigitalAssistant {
	return &odav1beta1.DigitalAssistant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-digitalassistant",
			Namespace: "default",
			Annotations: map[string]string{
				digitalAssistantOdaInstanceIDAnnotation: testDigitalAssistantOdaInstanceID,
			},
		},
		Spec: odav1beta1.DigitalAssistantSpec{
			Kind:               "NEW",
			Name:               "TestAssistant",
			DisplayName:        "Test Assistant",
			Version:            "1.0",
			Category:           "dev",
			Description:        "test digital assistant",
			PlatformVersion:    "22.04",
			MultilingualMode:   string(odasdk.BotMultilingualModeNative),
			PrimaryLanguageTag: "en",
			NativeLanguageTags: []string{"en"},
			FreeformTags:       map[string]string{},
			DefinedTags:        map[string]shared.MapValue{},
		},
	}
}

func makeSDKDigitalAssistant(id string, resource *odav1beta1.DigitalAssistant, state odasdk.LifecycleStateEnum) odasdk.DigitalAssistant {
	return odasdk.DigitalAssistant{
		Id:                 common.String(id),
		Name:               common.String(resource.Spec.Name),
		Version:            common.String(resource.Spec.Version),
		DisplayName:        common.String(resource.Spec.DisplayName),
		LifecycleState:     state,
		LifecycleDetails:   odasdk.BotPublishStateDraft,
		PlatformVersion:    common.String(resource.Spec.PlatformVersion),
		Category:           common.String(resource.Spec.Category),
		Description:        common.String(resource.Spec.Description),
		Namespace:          common.String("default"),
		DialogVersion:      common.String("1.0"),
		MultilingualMode:   odasdk.BotMultilingualModeEnum(resource.Spec.MultilingualMode),
		PrimaryLanguageTag: common.String(resource.Spec.PrimaryLanguageTag),
		NativeLanguageTags: append([]string(nil), resource.Spec.NativeLanguageTags...),
		FreeformTags:       cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:        digitalAssistantDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKDigitalAssistantSummary(id string, resource *odav1beta1.DigitalAssistant, state odasdk.LifecycleStateEnum) odasdk.DigitalAssistantSummary {
	return odasdk.DigitalAssistantSummary{
		Id:               common.String(id),
		Name:             common.String(resource.Spec.Name),
		Version:          common.String(resource.Spec.Version),
		DisplayName:      common.String(resource.Spec.DisplayName),
		Namespace:        common.String("default"),
		Category:         common.String(resource.Spec.Category),
		LifecycleState:   state,
		LifecycleDetails: odasdk.BotPublishStateDraft,
		PlatformVersion:  common.String(resource.Spec.PlatformVersion),
		FreeformTags:     cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:      digitalAssistantDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKDigitalAssistantFromJSONFixture(state odasdk.LifecycleStateEnum, displayName string) odasdk.DigitalAssistant {
	return odasdk.DigitalAssistant{
		Id:                 common.String(testDigitalAssistantID),
		Name:               common.String("JsonAssistant"),
		Version:            common.String("1.0"),
		DisplayName:        common.String(displayName),
		LifecycleState:     state,
		LifecycleDetails:   odasdk.BotPublishStateDraft,
		PlatformVersion:    common.String("22.04"),
		Category:           common.String("dev"),
		Description:        common.String("from json"),
		Namespace:          common.String("default"),
		DialogVersion:      common.String("1.0"),
		MultilingualMode:   odasdk.BotMultilingualModeNative,
		PrimaryLanguageTag: common.String("en"),
		NativeLanguageTags: []string{"en", "es"},
		FreeformTags:       map[string]string{"source": "json"},
		DefinedTags:        map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKDigitalAssistantWorkRequest(
	id string,
	action odasdk.WorkRequestRequestActionEnum,
	status odasdk.WorkRequestStatusEnum,
	resourceID string,
) odasdk.WorkRequest {
	var percent float32 = 50
	if status == odasdk.WorkRequestStatusSucceeded {
		percent = 100
	}
	workRequest := odasdk.WorkRequest{
		Id:              common.String(id),
		OdaInstanceId:   common.String(testDigitalAssistantOdaInstanceID),
		ResourceId:      common.String(resourceID),
		RequestAction:   action,
		Status:          status,
		PercentComplete: common.Float32(percent),
	}
	if resourceID != "" {
		workRequest.Resources = []odasdk.WorkRequestResource{
			{
				ResourceAction: odasdk.WorkRequestResourceResourceActionCreate,
				ResourceType:   common.String("digitalassistant"),
				ResourceId:     common.String(resourceID),
				Status:         odasdk.WorkRequestResourceStatusSucceeded,
			},
		}
	}
	return workRequest
}

func assertDigitalAssistantActiveStatus(t *testing.T, resource *odav1beta1.DigitalAssistant) {
	t.Helper()
	if got := resource.Status.Id; got != testDigitalAssistantID {
		t.Fatalf("status.id = %q, want %q", got, testDigitalAssistantID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalAssistantID {
		t.Fatalf("status.ocid = %q, want %q", got, testDigitalAssistantID)
	}
	if got := resource.Status.Name; got != resource.Spec.Name {
		t.Fatalf("status.name = %q, want %q", got, resource.Spec.Name)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
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

func assertDigitalAssistantStringSliceEqual(t *testing.T, name string, got []string, want []string) {
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

func assertDigitalAssistantStringSliceContains(t *testing.T, name string, got []string, want string) {
	t.Helper()
	for _, value := range got {
		if value == want {
			return
		}
	}
	t.Fatalf("%s = %#v, want entry %q", name, got, want)
}
