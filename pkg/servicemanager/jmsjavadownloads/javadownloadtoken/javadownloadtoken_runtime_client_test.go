/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package javadownloadtoken

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	jmsjavadownloadssdk "github.com/oracle/oci-go-sdk/v65/jmsjavadownloads"
	jmsjavadownloadsv1beta1 "github.com/oracle/oci-service-operator/api/jmsjavadownloads/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeJavaDownloadTokenOCIClient struct {
	createFn      func(context.Context, jmsjavadownloadssdk.CreateJavaDownloadTokenRequest) (jmsjavadownloadssdk.CreateJavaDownloadTokenResponse, error)
	getFn         func(context.Context, jmsjavadownloadssdk.GetJavaDownloadTokenRequest) (jmsjavadownloadssdk.GetJavaDownloadTokenResponse, error)
	listFn        func(context.Context, jmsjavadownloadssdk.ListJavaDownloadTokensRequest) (jmsjavadownloadssdk.ListJavaDownloadTokensResponse, error)
	updateFn      func(context.Context, jmsjavadownloadssdk.UpdateJavaDownloadTokenRequest) (jmsjavadownloadssdk.UpdateJavaDownloadTokenResponse, error)
	deleteFn      func(context.Context, jmsjavadownloadssdk.DeleteJavaDownloadTokenRequest) (jmsjavadownloadssdk.DeleteJavaDownloadTokenResponse, error)
	workRequestFn func(context.Context, jmsjavadownloadssdk.GetWorkRequestRequest) (jmsjavadownloadssdk.GetWorkRequestResponse, error)
}

func (f *fakeJavaDownloadTokenOCIClient) CreateJavaDownloadToken(
	ctx context.Context,
	req jmsjavadownloadssdk.CreateJavaDownloadTokenRequest,
) (jmsjavadownloadssdk.CreateJavaDownloadTokenResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return jmsjavadownloadssdk.CreateJavaDownloadTokenResponse{}, nil
}

func (f *fakeJavaDownloadTokenOCIClient) GetJavaDownloadToken(
	ctx context.Context,
	req jmsjavadownloadssdk.GetJavaDownloadTokenRequest,
) (jmsjavadownloadssdk.GetJavaDownloadTokenResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return jmsjavadownloadssdk.GetJavaDownloadTokenResponse{}, nil
}

func (f *fakeJavaDownloadTokenOCIClient) ListJavaDownloadTokens(
	ctx context.Context,
	req jmsjavadownloadssdk.ListJavaDownloadTokensRequest,
) (jmsjavadownloadssdk.ListJavaDownloadTokensResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return jmsjavadownloadssdk.ListJavaDownloadTokensResponse{}, nil
}

func (f *fakeJavaDownloadTokenOCIClient) UpdateJavaDownloadToken(
	ctx context.Context,
	req jmsjavadownloadssdk.UpdateJavaDownloadTokenRequest,
) (jmsjavadownloadssdk.UpdateJavaDownloadTokenResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return jmsjavadownloadssdk.UpdateJavaDownloadTokenResponse{}, nil
}

func (f *fakeJavaDownloadTokenOCIClient) DeleteJavaDownloadToken(
	ctx context.Context,
	req jmsjavadownloadssdk.DeleteJavaDownloadTokenRequest,
) (jmsjavadownloadssdk.DeleteJavaDownloadTokenResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return jmsjavadownloadssdk.DeleteJavaDownloadTokenResponse{}, nil
}

func (f *fakeJavaDownloadTokenOCIClient) GetWorkRequest(
	ctx context.Context,
	req jmsjavadownloadssdk.GetWorkRequestRequest,
) (jmsjavadownloadssdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return jmsjavadownloadssdk.GetWorkRequestResponse{}, nil
}

func TestReviewedJavaDownloadTokenRuntimeSemanticsEncodesSafeWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedJavaDownloadTokenRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedJavaDownloadTokenRuntimeSemantics() = nil")
	}

	if got.FormalService != "jmsjavadownloads" {
		t.Fatalf("FormalService = %q, want jmsjavadownloads", got.FormalService)
	}
	if got.FormalSlug != "javadownloadtoken" {
		t.Fatalf("FormalSlug = %q, want javadownloadtoken", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	assertJavaDownloadTokenStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertJavaDownloadTokenStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertJavaDownloadTokenStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertJavaDownloadTokenStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertJavaDownloadTokenStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertJavaDownloadTokenStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED", "NOT_FOUND"})
	assertJavaDownloadTokenStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	assertJavaDownloadTokenStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags", "isDefault", "licenseType", "timeExpires"})
	assertJavaDownloadTokenStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "javaVersion"})
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetJavaDownloadToken" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetJavaDownloadToken", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetJavaDownloadToken" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetJavaDownloadToken", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetJavaDownloadToken/ListJavaDownloadTokens confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.Unsupported) != 0 {
		t.Fatalf("Unsupported = %#v, want no runtime-blocking gaps in reviewed semantics", got.Unsupported)
	}
}

func TestGuardJavaDownloadTokenExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeJavaDownloadTokenResource()
	resource.Spec.DisplayName = ""

	decision, err := guardJavaDownloadTokenExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardJavaDownloadTokenExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardJavaDownloadTokenExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "token"
	resource.Spec.CompartmentId = ""
	decision, err = guardJavaDownloadTokenExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardJavaDownloadTokenExistingBeforeCreate(empty compartmentId) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardJavaDownloadTokenExistingBeforeCreate(empty compartmentId) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.CompartmentId = "ocid1.compartment.oc1..example"
	decision, err = guardJavaDownloadTokenExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardJavaDownloadTokenExistingBeforeCreate(non-empty identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardJavaDownloadTokenExistingBeforeCreate(non-empty identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestJavaDownloadTokenListFieldsOmitSensitiveValueFilter(t *testing.T) {
	t.Parallel()

	for _, field := range javaDownloadTokenListFields() {
		switch field.RequestName {
		case "value", "familyVersion", "searchByUser":
			t.Fatalf("javaDownloadTokenListFields() exposed unsafe query field %q", field.RequestName)
		}
	}
}

func TestBuildJavaDownloadTokenUpdateBodyPreservesFalseAndTagClears(t *testing.T) {
	t.Parallel()

	currentResource := makeJavaDownloadTokenResource()
	desired := makeJavaDownloadTokenResource()
	desired.Spec.DisplayName = "updated-token"
	desired.Spec.Description = "updated description"
	desired.Spec.IsDefault = false
	desired.Spec.TimeExpires = "2031-01-02T03:04:05Z"
	desired.Spec.LicenseType = []string{"BCL"}
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildJavaDownloadTokenUpdateBody(
		desired,
		jmsjavadownloadssdk.GetJavaDownloadTokenResponse{
			JavaDownloadToken: makeSDKJavaDownloadToken("ocid1.javadownloadtoken.oc1..existing", currentResource, jmsjavadownloadssdk.LifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildJavaDownloadTokenUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildJavaDownloadTokenUpdateBody() updateNeeded = false, want true")
	}

	requireJavaDownloadTokenStringPtr(t, "details.displayName", body.DisplayName, desired.Spec.DisplayName)
	requireJavaDownloadTokenStringPtr(t, "details.description", body.Description, desired.Spec.Description)
	requireJavaDownloadTokenBoolPtr(t, "details.isDefault", body.IsDefault, false)
	requireJavaDownloadTokenStringPtr(t, "details.timeExpires", sdkTimePtrString(body.TimeExpires), desired.Spec.TimeExpires)
	if !reflect.DeepEqual(body.LicenseType, []jmsjavadownloadssdk.LicenseTypeEnum{jmsjavadownloadssdk.LicenseTypeBcl}) {
		t.Fatalf("details.LicenseType = %#v, want [BCL]", body.LicenseType)
	}
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
}

func TestResolveAndRecoverJavaDownloadTokenWorkRequest(t *testing.T) {
	t.Parallel()

	workRequest := makeJavaDownloadTokenWorkRequest(
		"wr-token",
		jmsjavadownloadssdk.OperationTypeUpdateJavaDownloadToken,
		jmsjavadownloadssdk.OperationStatusInProgress,
		jmsjavadownloadssdk.ActionTypeUpdated,
		"ocid1.javadownloadtoken.oc1..resource",
	)

	action, err := resolveJavaDownloadTokenGeneratedWorkRequestAction(workRequest)
	if err != nil {
		t.Fatalf("resolveJavaDownloadTokenGeneratedWorkRequestAction() error = %v", err)
	}
	if action != string(jmsjavadownloadssdk.OperationTypeUpdateJavaDownloadToken) {
		t.Fatalf("resolveJavaDownloadTokenGeneratedWorkRequestAction() = %q, want %q", action, jmsjavadownloadssdk.OperationTypeUpdateJavaDownloadToken)
	}

	phase, ok, err := resolveJavaDownloadTokenGeneratedWorkRequestPhase(workRequest)
	if err != nil {
		t.Fatalf("resolveJavaDownloadTokenGeneratedWorkRequestPhase() error = %v", err)
	}
	if !ok || phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("resolveJavaDownloadTokenGeneratedWorkRequestPhase() = (%q, %t), want (%q, true)", phase, ok, shared.OSOKAsyncPhaseUpdate)
	}

	id, err := recoverJavaDownloadTokenIDFromGeneratedWorkRequest(makeJavaDownloadTokenResource(), workRequest, shared.OSOKAsyncPhaseUpdate)
	if err != nil {
		t.Fatalf("recoverJavaDownloadTokenIDFromGeneratedWorkRequest() error = %v", err)
	}
	if id != "ocid1.javadownloadtoken.oc1..resource" {
		t.Fatalf("recoverJavaDownloadTokenIDFromGeneratedWorkRequest() = %q, want token OCID", id)
	}
}

func TestJavaDownloadTokenStatusOmitsSensitiveValue(t *testing.T) {
	t.Parallel()

	statusType := reflect.TypeOf(jmsjavadownloadsv1beta1.JavaDownloadTokenStatus{})
	if _, ok := statusType.FieldByName("Value"); ok {
		t.Fatal("JavaDownloadTokenStatus.Value exists, want sensitive token value omitted from status")
	}
}

func TestJavaDownloadTokenCreateOrUpdateStartsWorkRequestWithoutValueLookup(t *testing.T) {
	t.Parallel()

	resource := makeJavaDownloadTokenResource()
	listCalls := 0

	client := newJavaDownloadTokenServiceClientWithOCIClient(loggerutil.OSOKLogger{}, &fakeJavaDownloadTokenOCIClient{
		listFn: func(_ context.Context, req jmsjavadownloadssdk.ListJavaDownloadTokensRequest) (jmsjavadownloadssdk.ListJavaDownloadTokensResponse, error) {
			listCalls++
			requireJavaDownloadTokenStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireJavaDownloadTokenStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.Value != nil {
				t.Fatalf("list value filter = %v, want nil", req.Value)
			}
			if req.FamilyVersion != nil {
				t.Fatalf("list familyVersion filter = %v, want nil", req.FamilyVersion)
			}
			if req.SearchByUser != nil {
				t.Fatalf("list searchByUser filter = %v, want nil", req.SearchByUser)
			}
			return jmsjavadownloadssdk.ListJavaDownloadTokensResponse{}, nil
		},
		createFn: func(_ context.Context, req jmsjavadownloadssdk.CreateJavaDownloadTokenRequest) (jmsjavadownloadssdk.CreateJavaDownloadTokenResponse, error) {
			requireJavaDownloadTokenStringPtr(t, "create displayName", req.CreateJavaDownloadTokenDetails.DisplayName, resource.Spec.DisplayName)
			requireJavaDownloadTokenStringPtr(t, "create compartmentId", req.CreateJavaDownloadTokenDetails.CompartmentId, resource.Spec.CompartmentId)
			return jmsjavadownloadssdk.CreateJavaDownloadTokenResponse{
				JavaDownloadToken: makeSDKJavaDownloadToken("ocid1.javadownloadtoken.oc1..created", resource, jmsjavadownloadssdk.LifecycleStateCreating),
				OpcWorkRequestId:  common.String("wr-create-token"),
				OpcRequestId:      common.String("opc-create-token"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req jmsjavadownloadssdk.GetWorkRequestRequest) (jmsjavadownloadssdk.GetWorkRequestResponse, error) {
			requireJavaDownloadTokenStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-token")
			return jmsjavadownloadssdk.GetWorkRequestResponse{
				WorkRequest: makeJavaDownloadTokenWorkRequest(
					"wr-create-token",
					jmsjavadownloadssdk.OperationTypeCreateJavaDownloadToken,
					jmsjavadownloadssdk.OperationStatusInProgress,
					jmsjavadownloadssdk.ActionTypeCreated,
					"ocid1.javadownloadtoken.oc1..created",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListJavaDownloadTokens() calls = %d, want 1", listCalls)
	}
	requireJavaDownloadTokenAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-token", shared.OSOKAsyncClassPending)
}

func makeJavaDownloadTokenResource() *jmsjavadownloadsv1beta1.JavaDownloadToken {
	return &jmsjavadownloadsv1beta1.JavaDownloadToken{
		Spec: jmsjavadownloadsv1beta1.JavaDownloadTokenSpec{
			DisplayName:   "token",
			Description:   "token description",
			CompartmentId: "ocid1.compartment.oc1..example",
			TimeExpires:   "2030-01-01T00:00:00Z",
			JavaVersion:   "17",
			LicenseType:   []string{"NFTC"},
			IsDefault:     true,
			FreeformTags: map[string]string{
				"managed-by": "osok",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"costCenter": "42",
				},
			},
		},
	}
}

func makeSDKJavaDownloadToken(
	id string,
	resource *jmsjavadownloadsv1beta1.JavaDownloadToken,
	lifecycleState jmsjavadownloadssdk.LifecycleStateEnum,
) jmsjavadownloadssdk.JavaDownloadToken {
	return jmsjavadownloadssdk.JavaDownloadToken{
		Id:             common.String(id),
		DisplayName:    common.String(resource.Spec.DisplayName),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		Description:    common.String(resource.Spec.Description),
		TimeCreated:    sdkTimePtr("2029-01-01T00:00:00Z"),
		TimeExpires:    sdkTimePtr(resource.Spec.TimeExpires),
		JavaVersion:    common.String(resource.Spec.JavaVersion),
		LifecycleState: lifecycleState,
		LicenseType:    []jmsjavadownloadssdk.LicenseTypeEnum{jmsjavadownloadssdk.LicenseTypeNftc},
		IsDefault:      common.Bool(resource.Spec.IsDefault),
		FreeformTags:   resource.Spec.FreeformTags,
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {
				"costCenter": "42",
			},
		},
	}
}

func makeJavaDownloadTokenWorkRequest(
	workRequestID string,
	operationType jmsjavadownloadssdk.OperationTypeEnum,
	status jmsjavadownloadssdk.OperationStatusEnum,
	actionType jmsjavadownloadssdk.ActionTypeEnum,
	resourceID string,
) jmsjavadownloadssdk.WorkRequest {
	resources := []jmsjavadownloadssdk.WorkRequestResource{}
	if resourceID != "" {
		resources = append(resources, jmsjavadownloadssdk.WorkRequestResource{
			EntityType: common.String("JavaDownloadToken"),
			ActionType: actionType,
			Identifier: common.String(resourceID),
		})
	}
	return jmsjavadownloadssdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(workRequestID),
		CompartmentId:   common.String("ocid1.compartment.oc1..example"),
		Resources:       resources,
		PercentComplete: common.Float32(25),
		TimeAccepted:    sdkTimePtr("2030-01-01T00:00:00Z"),
	}
}

func sdkTimePtr(value string) *common.SDKTime {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &common.SDKTime{Time: parsed}
}

func sdkTimePtrString(value *common.SDKTime) *string {
	if value == nil {
		return nil
	}
	formatted := value.Time.Format(time.RFC3339)
	return &formatted
}

func requireJavaDownloadTokenStringPtr(t *testing.T, field string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if got := *value; got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func requireJavaDownloadTokenBoolPtr(t *testing.T, field string, value *bool, want bool) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %t", field, want)
	}
	if got := *value; got != want {
		t.Fatalf("%s = %t, want %t", field, got, want)
	}
}

func requireJavaDownloadTokenAsyncCurrent(
	t *testing.T,
	resource *jmsjavadownloadsv1beta1.JavaDownloadToken,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want pending workrequest")
	}
	current := resource.Status.OsokStatus.Async.Current
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

func assertJavaDownloadTokenStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}
