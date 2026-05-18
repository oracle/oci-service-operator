/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opainstance

import (
	"context"
	"io"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	opasdk "github.com/oracle/oci-go-sdk/v65/opa"
	opav1beta1 "github.com/oracle/oci-service-operator/api/opa/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeOpaInstanceOCIClient struct {
	createFn      func(context.Context, opasdk.CreateOpaInstanceRequest) (opasdk.CreateOpaInstanceResponse, error)
	getFn         func(context.Context, opasdk.GetOpaInstanceRequest) (opasdk.GetOpaInstanceResponse, error)
	listFn        func(context.Context, opasdk.ListOpaInstancesRequest) (opasdk.ListOpaInstancesResponse, error)
	updateFn      func(context.Context, opasdk.UpdateOpaInstanceRequest) (opasdk.UpdateOpaInstanceResponse, error)
	deleteFn      func(context.Context, opasdk.DeleteOpaInstanceRequest) (opasdk.DeleteOpaInstanceResponse, error)
	workRequestFn func(context.Context, opasdk.GetWorkRequestRequest) (opasdk.GetWorkRequestResponse, error)
}

func (f *fakeOpaInstanceOCIClient) CreateOpaInstance(
	ctx context.Context,
	req opasdk.CreateOpaInstanceRequest,
) (opasdk.CreateOpaInstanceResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return opasdk.CreateOpaInstanceResponse{}, nil
}

func (f *fakeOpaInstanceOCIClient) GetOpaInstance(
	ctx context.Context,
	req opasdk.GetOpaInstanceRequest,
) (opasdk.GetOpaInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return opasdk.GetOpaInstanceResponse{}, errortest.NewServiceError(404, "NotFound", "missing OpaInstance")
}

func (f *fakeOpaInstanceOCIClient) ListOpaInstances(
	ctx context.Context,
	req opasdk.ListOpaInstancesRequest,
) (opasdk.ListOpaInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return opasdk.ListOpaInstancesResponse{}, nil
}

func (f *fakeOpaInstanceOCIClient) UpdateOpaInstance(
	ctx context.Context,
	req opasdk.UpdateOpaInstanceRequest,
) (opasdk.UpdateOpaInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return opasdk.UpdateOpaInstanceResponse{}, nil
}

func (f *fakeOpaInstanceOCIClient) DeleteOpaInstance(
	ctx context.Context,
	req opasdk.DeleteOpaInstanceRequest,
) (opasdk.DeleteOpaInstanceResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return opasdk.DeleteOpaInstanceResponse{}, nil
}

func (f *fakeOpaInstanceOCIClient) GetWorkRequest(
	ctx context.Context,
	req opasdk.GetWorkRequestRequest,
) (opasdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return opasdk.GetWorkRequestResponse{}, nil
}

type opaInstanceRequestBodyBuilder interface {
	HTTPRequest(string, string, *common.OCIReadSeekCloser, map[string]string) (http.Request, error)
}

func TestReviewedOpaInstanceRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedOpaInstanceRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedOpaInstanceRuntimeSemantics() = nil")
	}
	if got.FormalService != "opa" {
		t.Fatalf("FormalService = %q, want opa", got.FormalService)
	}
	if got.FormalSlug != "opainstance" {
		t.Fatalf("FormalSlug = %q, want opainstance", got.FormalSlug)
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
	assertOpaInstanceStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertOpaInstanceStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertOpaInstanceStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertOpaInstanceStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertOpaInstanceStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertOpaInstanceStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName"})
	assertOpaInstanceStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags"})
	assertOpaInstanceStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "consumptionModel", "idcsAt", "isBreakglassEnabled", "meteringType", "shapeName"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetOpaInstance" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetOpaInstance", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetOpaInstance" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetOpaInstance", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetOpaInstance/ListOpaInstances confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}

	hooks := newOpaInstanceDefaultRuntimeHooks(opasdk.OpaInstanceClient{})
	applyOpaInstanceRuntimeHooks(&hooks, &fakeOpaInstanceOCIClient{}, nil)
	if hooks.ParityHooks.NormalizeDesiredState == nil {
		t.Fatal("hooks.ParityHooks.NormalizeDesiredState = nil, want create-time normalization hook")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want bounded reuse guard")
	}
	if hooks.Async.GetWorkRequest == nil || hooks.Async.ResolvePhase == nil || hooks.Async.RecoverResourceID == nil {
		t.Fatalf("hooks.Async = %#v, want populated workrequest hooks", hooks.Async)
	}
}

func TestGuardOpaInstanceExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeOpaInstanceResource()
	resource.Spec.DisplayName = ""

	decision, err := guardOpaInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardOpaInstanceExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardOpaInstanceExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "opa-instance"
	resource.Spec.CompartmentId = ""
	decision, err = guardOpaInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardOpaInstanceExistingBeforeCreate(empty compartmentId) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardOpaInstanceExistingBeforeCreate(empty compartmentId) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.CompartmentId = "ocid1.compartment.oc1..opaexample"
	decision, err = guardOpaInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardOpaInstanceExistingBeforeCreate(complete identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardOpaInstanceExistingBeforeCreate(complete identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildOpaInstanceUpdateBodyPreservesClearsAndRename(t *testing.T) {
	t.Parallel()

	currentResource := makeOpaInstanceResource()
	desired := makeOpaInstanceResource()
	desired.Spec.DisplayName = "opa-instance-updated"
	desired.Spec.Description = ""
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildOpaInstanceUpdateBody(
		desired,
		opasdk.GetOpaInstanceResponse{
			OpaInstance: makeSDKOpaInstance(
				"ocid1.opainstance.oc1..existing",
				currentResource,
				opasdk.OpaInstanceLifecycleStateActive,
			),
		},
	)
	if err != nil {
		t.Fatalf("buildOpaInstanceUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildOpaInstanceUpdateBody() updateNeeded = false, want true")
	}

	requireOpaInstanceStringPtr(t, "details.displayName", body.DisplayName, desired.Spec.DisplayName)
	requireOpaInstanceStringPtr(t, "details.description", body.Description, "")
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map clear", body.DefinedTags)
	}

	requestBody := opaInstanceSerializedRequestBody(t, opasdk.UpdateOpaInstanceRequest{
		OpaInstanceId:            common.String("ocid1.opainstance.oc1..existing"),
		UpdateOpaInstanceDetails: body,
	}, http.MethodPut, "/opaInstances/ocid1.opainstance.oc1..existing")
	for _, want := range []string{
		`"displayName":"opa-instance-updated"`,
		`"description":""`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(requestBody, want) {
			t.Fatalf("request body %s does not contain %s", requestBody, want)
		}
	}
}

func TestOpaInstanceCreateOrUpdateRejectsAmbiguousDisplayNameReuse(t *testing.T) {
	t.Parallel()

	resource := makeOpaInstanceResource()
	createCalls := 0

	client := newTestOpaInstanceClient(&fakeOpaInstanceOCIClient{
		listFn: func(_ context.Context, req opasdk.ListOpaInstancesRequest) (opasdk.ListOpaInstancesResponse, error) {
			requireOpaInstanceStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireOpaInstanceStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.LifecycleState != "" {
				t.Fatalf("list lifecycleState = %q, want empty reviewed lookup filter", req.LifecycleState)
			}
			return opasdk.ListOpaInstancesResponse{
				OpaInstanceCollection: opasdk.OpaInstanceCollection{
					Items: []opasdk.OpaInstanceSummary{
						makeSDKOpaInstanceSummary("ocid1.opainstance.oc1..first", resource, opasdk.OpaInstanceLifecycleStateActive),
						makeSDKOpaInstanceSummary("ocid1.opainstance.oc1..second", resource, opasdk.OpaInstanceLifecycleStateUpdating),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ opasdk.CreateOpaInstanceRequest) (opasdk.CreateOpaInstanceResponse, error) {
			createCalls++
			return opasdk.CreateOpaInstanceResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want ambiguous list match failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful result", response)
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match failure", err)
	}
	if createCalls != 0 {
		t.Fatalf("CreateOpaInstance() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
}

func TestOpaInstanceCreateOrUpdateIgnoresUnobservedIdcsAtAfterCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.opainstance.oc1..existing"

	resource := newExistingOpaInstanceResource(existingID)
	resource.Spec.IdcsAt = "rotated-idcs-token"
	updateCalled := false

	client := newTestOpaInstanceClient(&fakeOpaInstanceOCIClient{
		getFn: func(_ context.Context, req opasdk.GetOpaInstanceRequest) (opasdk.GetOpaInstanceResponse, error) {
			requireOpaInstanceStringPtr(t, "get opaInstanceId", req.OpaInstanceId, existingID)
			return opasdk.GetOpaInstanceResponse{
				OpaInstance: makeSDKOpaInstance(
					existingID,
					makeOpaInstanceResource(),
					opasdk.OpaInstanceLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, _ opasdk.UpdateOpaInstanceRequest) (opasdk.UpdateOpaInstanceResponse, error) {
			updateCalled = true
			t.Fatal("UpdateOpaInstance() should not be called for create-only idcsAt drift")
			return opasdk.UpdateOpaInstanceResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for steady ACTIVE state")
	}
	if updateCalled {
		t.Fatal("UpdateOpaInstance() was called unexpectedly")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Active)
	}
}

func TestOpaInstanceServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.opainstance.oc1..created"
		workRequestID = "wr-opainstance-create"
	)

	resource := makeOpaInstanceResource()
	workRequests := map[string]opasdk.WorkRequest{
		workRequestID: makeOpaInstanceWorkRequest(
			workRequestID,
			opasdk.OperationTypeCreateOpaInstance,
			opasdk.OperationStatusInProgress,
			opasdk.ActionTypeInProgress,
			"",
		),
	}

	var createRequest opasdk.CreateOpaInstanceRequest
	var listRequest opasdk.ListOpaInstancesRequest
	getCalls := 0

	client := newTestOpaInstanceClient(&fakeOpaInstanceOCIClient{
		listFn: func(_ context.Context, req opasdk.ListOpaInstancesRequest) (opasdk.ListOpaInstancesResponse, error) {
			listRequest = req
			return opasdk.ListOpaInstancesResponse{}, nil
		},
		createFn: func(_ context.Context, req opasdk.CreateOpaInstanceRequest) (opasdk.CreateOpaInstanceResponse, error) {
			createRequest = req
			return opasdk.CreateOpaInstanceResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-opa"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req opasdk.GetWorkRequestRequest) (opasdk.GetWorkRequestResponse, error) {
			requireOpaInstanceStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return opasdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req opasdk.GetOpaInstanceRequest) (opasdk.GetOpaInstanceResponse, error) {
			getCalls++
			requireOpaInstanceStringPtr(t, "get opaInstanceId", req.OpaInstanceId, createdID)
			return opasdk.GetOpaInstanceResponse{
				OpaInstance: makeSDKOpaInstance(createdID, resource, opasdk.OpaInstanceLifecycleStateActive),
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
	requireOpaInstanceStringPtr(t, "list compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireOpaInstanceStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	if listRequest.LifecycleState != "" {
		t.Fatalf("list lifecycleState = %q, want empty reviewed lookup filter", listRequest.LifecycleState)
	}
	requireOpaInstanceStringPtr(t, "create compartmentId", createRequest.CreateOpaInstanceDetails.CompartmentId, resource.Spec.CompartmentId)
	requireOpaInstanceStringPtr(t, "create displayName", createRequest.CreateOpaInstanceDetails.DisplayName, resource.Spec.DisplayName)
	if createRequest.CreateOpaInstanceDetails.ShapeName != opasdk.OpaInstanceShapeNameEnum(resource.Spec.ShapeName) {
		t.Fatalf("create shapeName = %q, want %q", createRequest.CreateOpaInstanceDetails.ShapeName, resource.Spec.ShapeName)
	}
	if getCalls != 0 {
		t.Fatalf("GetOpaInstance() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireOpaInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-opa" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-opa", got)
	}

	workRequests[workRequestID] = makeOpaInstanceWorkRequest(
		workRequestID,
		opasdk.OperationTypeCreateOpaInstance,
		opasdk.OperationStatusSucceeded,
		opasdk.ActionTypeCreated,
		createdID,
	)

	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetOpaInstance() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(opasdk.OpaInstanceLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOpaInstanceDeleteStartsWorkRequestAndWaitsForConfirmation(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.opainstance.oc1..existing"
		workRequestID = "wr-opainstance-delete"
	)

	resource := newExistingOpaInstanceResource(existingID)
	getCalls := 0
	var deleteRequest opasdk.DeleteOpaInstanceRequest

	client := newTestOpaInstanceClient(&fakeOpaInstanceOCIClient{
		getFn: func(_ context.Context, req opasdk.GetOpaInstanceRequest) (opasdk.GetOpaInstanceResponse, error) {
			getCalls++
			requireOpaInstanceStringPtr(t, "get opaInstanceId", req.OpaInstanceId, existingID)
			return opasdk.GetOpaInstanceResponse{
				OpaInstance: makeSDKOpaInstance(existingID, resource, opasdk.OpaInstanceLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req opasdk.DeleteOpaInstanceRequest) (opasdk.DeleteOpaInstanceResponse, error) {
			deleteRequest = req
			return opasdk.DeleteOpaInstanceResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-delete-opa"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req opasdk.GetWorkRequestRequest) (opasdk.GetWorkRequestResponse, error) {
			requireOpaInstanceStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return opasdk.GetWorkRequestResponse{
				WorkRequest: makeOpaInstanceWorkRequest(
					workRequestID,
					opasdk.OperationTypeDeleteOpaInstance,
					opasdk.OperationStatusInProgress,
					opasdk.ActionTypeInProgress,
					existingID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending delete confirmation")
	}
	requireOpaInstanceStringPtr(t, "delete opaInstanceId", deleteRequest.OpaInstanceId, existingID)
	requireOpaInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-opa" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-opa", got)
	}
	if getCalls != 0 {
		t.Fatalf("GetOpaInstance() calls = %d, want 0 when the tracked OCID is already available", getCalls)
	}
}

func newTestOpaInstanceClient(client *fakeOpaInstanceOCIClient) OpaInstanceServiceClient {
	if client == nil {
		client = &fakeOpaInstanceOCIClient{}
	}
	return newOpaInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		client,
	)
}

func makeOpaInstanceResource() *opav1beta1.OpaInstance {
	return &opav1beta1.OpaInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opa-instance-sample",
			Namespace: "default",
		},
		Spec: opav1beta1.OpaInstanceSpec{
			DisplayName:         "opa-instance",
			CompartmentId:       "ocid1.compartment.oc1..opaexample",
			ShapeName:           "DEVELOPMENT",
			Description:         "opa instance description",
			ConsumptionModel:    string(opasdk.OpaInstanceConsumptionModelUcm),
			MeteringType:        string(opasdk.OpaInstanceMeteringTypeUsers),
			IdcsAt:              "opaque-idcs-token",
			IsBreakglassEnabled: true,
			FreeformTags: map[string]string{
				"environment": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func newExistingOpaInstanceResource(existingID string) *opav1beta1.OpaInstance {
	resource := makeOpaInstanceResource()
	resource.Status = opav1beta1.OpaInstanceStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func makeSDKOpaInstance(
	id string,
	resource *opav1beta1.OpaInstance,
	state opasdk.OpaInstanceLifecycleStateEnum,
) opasdk.OpaInstance {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return opasdk.OpaInstance{
		Id:                  common.String(id),
		DisplayName:         common.String(resource.Spec.DisplayName),
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		ShapeName:           opasdk.OpaInstanceShapeNameEnum(resource.Spec.ShapeName),
		TimeCreated:         &now,
		LifecycleState:      state,
		Description:         common.String(resource.Spec.Description),
		InstanceUrl:         common.String("https://opa.example.com"),
		ConsumptionModel:    opasdk.OpaInstanceConsumptionModelEnum(resource.Spec.ConsumptionModel),
		MeteringType:        opasdk.OpaInstanceMeteringTypeEnum(resource.Spec.MeteringType),
		TimeUpdated:         &now,
		IsBreakglassEnabled: common.Bool(resource.Spec.IsBreakglassEnabled),
		FreeformTags:        maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:         sdkOpaInstanceDefinedTags(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeSDKOpaInstanceSummary(
	id string,
	resource *opav1beta1.OpaInstance,
	state opasdk.OpaInstanceLifecycleStateEnum,
) opasdk.OpaInstanceSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return opasdk.OpaInstanceSummary{
		Id:                  common.String(id),
		DisplayName:         common.String(resource.Spec.DisplayName),
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		ShapeName:           opasdk.OpaInstanceShapeNameEnum(resource.Spec.ShapeName),
		TimeCreated:         &now,
		LifecycleState:      state,
		Description:         common.String(resource.Spec.Description),
		InstanceUrl:         common.String("https://opa.example.com"),
		ConsumptionModel:    opasdk.OpaInstanceConsumptionModelEnum(resource.Spec.ConsumptionModel),
		MeteringType:        opasdk.OpaInstanceMeteringTypeEnum(resource.Spec.MeteringType),
		TimeUpdated:         &now,
		IsBreakglassEnabled: common.Bool(resource.Spec.IsBreakglassEnabled),
		FreeformTags:        maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:         sdkOpaInstanceDefinedTags(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeOpaInstanceWorkRequest(
	id string,
	operation opasdk.OperationTypeEnum,
	status opasdk.OperationStatusEnum,
	action opasdk.ActionTypeEnum,
	resourceID string,
) opasdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return opasdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..opaexample"),
		Resources:       []opasdk.WorkRequestResource{{EntityType: common.String("OpaInstance"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func sdkOpaInstanceDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		out[namespace] = converted
	}
	return out
}

func opaInstanceSerializedRequestBody(
	t *testing.T,
	request opaInstanceRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request body) error = %v", err)
	}
	return string(body)
}

func assertOpaInstanceStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireOpaInstanceStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireOpaInstanceAsyncCurrent(
	t *testing.T,
	resource *opav1beta1.OpaInstance,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}
