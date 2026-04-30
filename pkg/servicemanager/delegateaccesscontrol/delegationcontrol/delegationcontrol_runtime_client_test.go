/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package delegationcontrol

import (
	"context"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	delegateaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol"
	delegateaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/delegateaccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDelegationControlOCIClient struct {
	createFn      func(context.Context, delegateaccesscontrolsdk.CreateDelegationControlRequest) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error)
	getFn         func(context.Context, delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error)
	listFn        func(context.Context, delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error)
	updateFn      func(context.Context, delegateaccesscontrolsdk.UpdateDelegationControlRequest) (delegateaccesscontrolsdk.UpdateDelegationControlResponse, error)
	deleteFn      func(context.Context, delegateaccesscontrolsdk.DeleteDelegationControlRequest) (delegateaccesscontrolsdk.DeleteDelegationControlResponse, error)
	workRequestFn func(context.Context, delegateaccesscontrolsdk.GetWorkRequestRequest) (delegateaccesscontrolsdk.GetWorkRequestResponse, error)
}

func (f *fakeDelegationControlOCIClient) CreateDelegationControl(
	ctx context.Context,
	req delegateaccesscontrolsdk.CreateDelegationControlRequest,
) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return delegateaccesscontrolsdk.CreateDelegationControlResponse{}, nil
}

func (f *fakeDelegationControlOCIClient) GetDelegationControl(
	ctx context.Context,
	req delegateaccesscontrolsdk.GetDelegationControlRequest,
) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return delegateaccesscontrolsdk.GetDelegationControlResponse{}, errortest.NewServiceError(404, "NotFound", "missing DelegationControl")
}

func (f *fakeDelegationControlOCIClient) ListDelegationControls(
	ctx context.Context,
	req delegateaccesscontrolsdk.ListDelegationControlsRequest,
) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return delegateaccesscontrolsdk.ListDelegationControlsResponse{}, nil
}

func (f *fakeDelegationControlOCIClient) UpdateDelegationControl(
	ctx context.Context,
	req delegateaccesscontrolsdk.UpdateDelegationControlRequest,
) (delegateaccesscontrolsdk.UpdateDelegationControlResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return delegateaccesscontrolsdk.UpdateDelegationControlResponse{}, nil
}

func (f *fakeDelegationControlOCIClient) DeleteDelegationControl(
	ctx context.Context,
	req delegateaccesscontrolsdk.DeleteDelegationControlRequest,
) (delegateaccesscontrolsdk.DeleteDelegationControlResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return delegateaccesscontrolsdk.DeleteDelegationControlResponse{}, nil
}

func (f *fakeDelegationControlOCIClient) GetWorkRequest(
	ctx context.Context,
	req delegateaccesscontrolsdk.GetWorkRequestRequest,
) (delegateaccesscontrolsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return delegateaccesscontrolsdk.GetWorkRequestResponse{}, nil
}

func TestReviewedDelegationControlRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedDelegationControlRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedDelegationControlRuntimeSemantics() = nil")
	}

	if got.FormalService != "delegateaccesscontrol" {
		t.Fatalf("FormalService = %q, want delegateaccesscontrol", got.FormalService)
	}
	if got.FormalSlug != "delegationcontrol" {
		t.Fatalf("FormalSlug = %q, want delegationcontrol", got.FormalSlug)
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
	assertDelegationControlStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertDelegationControlStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertDelegationControlStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertDelegationControlStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertDelegationControlStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertDelegationControlStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertDelegationControlStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "resourceType", "resourceIds"})
	assertDelegationControlStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"definedTags",
		"delegationSubscriptionIds",
		"description",
		"displayName",
		"freeformTags",
		"isAutoApproveDuringMaintenance",
		"notificationMessageFormat",
		"notificationTopicId",
		"numApprovalsRequired",
		"preApprovedServiceProviderActionNames",
		"resourceIds",
	})
	assertDelegationControlStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "resourceType", "vaultId", "vaultKeyId"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetDelegationControl" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetDelegationControl", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetDelegationControl" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetDelegationControl", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetDelegationControl/ListDelegationControls confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardDelegationControlExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeDelegationControlResource()
	resource.Spec.ResourceIds = nil

	decision, err := guardDelegationControlExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardDelegationControlExistingBeforeCreate(empty resourceIds) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardDelegationControlExistingBeforeCreate(empty resourceIds) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.ResourceIds = []string{"ocid1.vmcluster.oc1..two", "ocid1.vmcluster.oc1..one"}
	decision, err = guardDelegationControlExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardDelegationControlExistingBeforeCreate(complete identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardDelegationControlExistingBeforeCreate(complete identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildDelegationControlCreateBodyValidatesVaultConstraintAndPreservesFalseBool(t *testing.T) {
	t.Parallel()

	t.Run("cloudvmcluster requires vault inputs", func(t *testing.T) {
		resource := makeDelegationControlResource()
		resource.Spec.ResourceType = string(delegateaccesscontrolsdk.DelegationControlResourceTypeCloudvmcluster)
		resource.Spec.VaultId = ""
		resource.Spec.VaultKeyId = ""

		_, err := buildDelegationControlCreateBody(context.Background(), resource, "default")
		if err == nil {
			t.Fatal("buildDelegationControlCreateBody() error = nil, want missing vault validation")
		}
		if !strings.Contains(err.Error(), "requires both vaultId and vaultKeyId") {
			t.Fatalf("buildDelegationControlCreateBody() error = %v, want missing vault detail", err)
		}
	})

	t.Run("vmcluster rejects vault inputs", func(t *testing.T) {
		resource := makeDelegationControlResource()
		resource.Spec.ResourceType = string(delegateaccesscontrolsdk.DelegationControlResourceTypeVmcluster)
		resource.Spec.VaultId = "ocid1.vault.oc1..unexpected"
		resource.Spec.VaultKeyId = "ocid1.key.oc1..unexpected"

		_, err := buildDelegationControlCreateBody(context.Background(), resource, "default")
		if err == nil {
			t.Fatal("buildDelegationControlCreateBody() error = nil, want unsupported vault validation")
		}
		if !strings.Contains(err.Error(), "only supported when resourceType is CLOUDVMCLUSTER") {
			t.Fatalf("buildDelegationControlCreateBody() error = %v, want unsupported vault detail", err)
		}
	})

	t.Run("explicit false bool survives create body projection", func(t *testing.T) {
		resource := makeDelegationControlResource()
		resource.Spec.IsAutoApproveDuringMaintenance = false

		body, err := buildDelegationControlCreateBody(context.Background(), resource, "default")
		if err != nil {
			t.Fatalf("buildDelegationControlCreateBody() error = %v", err)
		}
		requireDelegationControlBoolPtr(t, "details.isAutoApproveDuringMaintenance", body.IsAutoApproveDuringMaintenance, false)
	})
}

func TestBuildDelegationControlUpdateBodyPreservesClearSemanticsAndFalseBool(t *testing.T) {
	t.Parallel()

	currentResource := makeDelegationControlResource()
	desired := makeDelegationControlResource()
	desired.Spec.Description = ""
	desired.Spec.DelegationSubscriptionIds = []string{}
	desired.Spec.ResourceIds = []string{}
	desired.Spec.PreApprovedServiceProviderActionNames = []string{}
	desired.Spec.IsAutoApproveDuringMaintenance = false
	desired.Spec.NotificationTopicId = "ocid1.onstopic.oc1..updated"
	desired.Spec.NotificationMessageFormat = string(delegateaccesscontrolsdk.DelegationControlNotificationMessageFormatHtml)
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildDelegationControlUpdateBody(
		desired,
		delegateaccesscontrolsdk.GetDelegationControlResponse{
			DelegationControl: makeSDKDelegationControl(
				"ocid1.delegationcontrol.oc1..existing",
				currentResource,
				delegateaccesscontrolsdk.DelegationControlLifecycleStateActive,
			),
		},
	)
	if err != nil {
		t.Fatalf("buildDelegationControlUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildDelegationControlUpdateBody() updateNeeded = false, want true")
	}

	requireDelegationControlStringPtr(t, "details.description", body.Description, "")
	requireDelegationControlStringPtr(t, "details.notificationTopicId", body.NotificationTopicId, desired.Spec.NotificationTopicId)
	if body.NotificationMessageFormat != delegateaccesscontrolsdk.DelegationControlNotificationMessageFormatHtml {
		t.Fatalf("details.NotificationMessageFormat = %q, want HTML", body.NotificationMessageFormat)
	}
	requireDelegationControlBoolPtr(t, "details.isAutoApproveDuringMaintenance", body.IsAutoApproveDuringMaintenance, false)
	if len(body.DelegationSubscriptionIds) != 0 {
		t.Fatalf("details.DelegationSubscriptionIds = %#v, want empty slice for clear", body.DelegationSubscriptionIds)
	}
	if len(body.ResourceIds) != 0 {
		t.Fatalf("details.ResourceIds = %#v, want empty slice for clear", body.ResourceIds)
	}
	if len(body.PreApprovedServiceProviderActionNames) != 0 {
		t.Fatalf("details.PreApprovedServiceProviderActionNames = %#v, want empty slice for clear", body.PreApprovedServiceProviderActionNames)
	}
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
}

func TestLookupExistingDelegationControlRequiresMatchingVaultIdentity(t *testing.T) {
	t.Parallel()

	resource := makeDelegationControlResource()
	resource.Spec.ResourceType = string(delegateaccesscontrolsdk.DelegationControlResourceTypeCloudvmcluster)
	resource.Spec.ResourceIds = []string{"ocid1.cloudvmcluster.oc1..two", "ocid1.cloudvmcluster.oc1..one"}
	resource.Spec.VaultId = "ocid1.vault.oc1..desired"
	resource.Spec.VaultKeyId = "ocid1.key.oc1..desired"

	identity, err := resolveDelegationControlIdentity(resource)
	if err != nil {
		t.Fatalf("resolveDelegationControlIdentity() error = %v", err)
	}

	client := &fakeDelegationControlOCIClient{
		listFn: func(_ context.Context, req delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
			requireDelegationControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireDelegationControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			requireDelegationControlStringPtr(t, "list resourceId", req.ResourceId, "ocid1.cloudvmcluster.oc1..one")
			return delegateaccesscontrolsdk.ListDelegationControlsResponse{
				DelegationControlSummaryCollection: delegateaccesscontrolsdk.DelegationControlSummaryCollection{
					Items: []delegateaccesscontrolsdk.DelegationControlSummary{
						makeSDKDelegationControlSummary("ocid1.delegationcontrol.oc1..candidate", resource, delegateaccesscontrolsdk.DelegationControlLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
			requireDelegationControlStringPtr(t, "get delegationControlId", req.DelegationControlId, "ocid1.delegationcontrol.oc1..candidate")
			candidate := makeSDKDelegationControl("ocid1.delegationcontrol.oc1..candidate", resource, delegateaccesscontrolsdk.DelegationControlLifecycleStateActive)
			candidate.VaultId = common.String("ocid1.vault.oc1..other")
			candidate.VaultKeyId = common.String("ocid1.key.oc1..other")
			return delegateaccesscontrolsdk.GetDelegationControlResponse{DelegationControl: candidate}, nil
		},
	}

	got, err := lookupExistingDelegationControl(context.Background(), client, nil, identity)
	if err != nil {
		t.Fatalf("lookupExistingDelegationControl() error = %v", err)
	}
	if got != nil {
		t.Fatalf("lookupExistingDelegationControl() = %#v, want nil when create-only vault identity differs", got)
	}
}

func TestDelegationControlCreateOrUpdateRejectsAmbiguousReuse(t *testing.T) {
	t.Parallel()

	resource := makeDelegationControlResource()
	createCalls := 0
	getCalls := 0

	client := newTestDelegationControlClient(&fakeDelegationControlOCIClient{
		listFn: func(_ context.Context, req delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
			requireDelegationControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireDelegationControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			requireDelegationControlStringPtr(t, "list resourceId", req.ResourceId, "ocid1.vmcluster.oc1..one")
			if req.ResourceType != delegateaccesscontrolsdk.ListDelegationControlsResourceTypeEnum(resource.Spec.ResourceType) {
				t.Fatalf("list resourceType = %q, want %q", req.ResourceType, resource.Spec.ResourceType)
			}
			return delegateaccesscontrolsdk.ListDelegationControlsResponse{
				DelegationControlSummaryCollection: delegateaccesscontrolsdk.DelegationControlSummaryCollection{
					Items: []delegateaccesscontrolsdk.DelegationControlSummary{
						makeSDKDelegationControlSummary("ocid1.delegationcontrol.oc1..first", resource, delegateaccesscontrolsdk.DelegationControlLifecycleStateActive),
						makeSDKDelegationControlSummary("ocid1.delegationcontrol.oc1..second", resource, delegateaccesscontrolsdk.DelegationControlLifecycleStateUpdating),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
			getCalls++
			return delegateaccesscontrolsdk.GetDelegationControlResponse{
				DelegationControl: makeSDKDelegationControl(
					stringValue(req.DelegationControlId),
					resource,
					delegateaccesscontrolsdk.DelegationControlLifecycleStateActive,
				),
			}, nil
		},
		createFn: func(_ context.Context, _ delegateaccesscontrolsdk.CreateDelegationControlRequest) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error) {
			createCalls++
			return delegateaccesscontrolsdk.CreateDelegationControlResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want ambiguous exact match failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful result", response)
	}
	if !strings.Contains(err.Error(), "multiple exact matches") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match failure", err)
	}
	if getCalls != 2 {
		t.Fatalf("GetDelegationControl() calls = %d, want 2 candidate rereads", getCalls)
	}
	if createCalls != 0 {
		t.Fatalf("CreateDelegationControl() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
}

func TestDelegationControlCreateOrUpdateDoesNotReuseDifferentResourceIdentity(t *testing.T) {
	t.Parallel()

	const workRequestID = "wr-delegationcontrol-create-different-resource"

	resource := makeDelegationControlResource()
	createCalls := 0
	getCalls := 0
	listCalls := 0

	client := newTestDelegationControlClient(&fakeDelegationControlOCIClient{
		listFn: func(_ context.Context, req delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
			listCalls++
			requireDelegationControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireDelegationControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.ResourceType != delegateaccesscontrolsdk.ListDelegationControlsResourceTypeEnum(resource.Spec.ResourceType) {
				t.Fatalf("list resourceType = %q, want %q", req.ResourceType, resource.Spec.ResourceType)
			}
			if listCalls == 1 {
				requireDelegationControlStringPtr(t, "strict lookup resourceId", req.ResourceId, "ocid1.vmcluster.oc1..one")
			} else if req.ResourceId != nil {
				t.Fatalf("generic fallback resourceId = %v, want nil", req.ResourceId)
			}
			return delegateaccesscontrolsdk.ListDelegationControlsResponse{
				DelegationControlSummaryCollection: delegateaccesscontrolsdk.DelegationControlSummaryCollection{
					Items: []delegateaccesscontrolsdk.DelegationControlSummary{
						makeSDKDelegationControlSummary(
							"ocid1.delegationcontrol.oc1..different-resource",
							resource,
							delegateaccesscontrolsdk.DelegationControlLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
			getCalls++
			requireDelegationControlStringPtr(t, "get delegationControlId", req.DelegationControlId, "ocid1.delegationcontrol.oc1..different-resource")
			candidate := makeSDKDelegationControl(
				"ocid1.delegationcontrol.oc1..different-resource",
				resource,
				delegateaccesscontrolsdk.DelegationControlLifecycleStateActive,
			)
			candidate.ResourceIds = []string{"ocid1.vmcluster.oc1..different"}
			return delegateaccesscontrolsdk.GetDelegationControlResponse{DelegationControl: candidate}, nil
		},
		createFn: func(_ context.Context, req delegateaccesscontrolsdk.CreateDelegationControlRequest) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error) {
			createCalls++
			requireDelegationControlStringPtr(t, "create compartmentId", req.CreateDelegationControlDetails.CompartmentId, resource.Spec.CompartmentId)
			return delegateaccesscontrolsdk.CreateDelegationControlResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-delegationcontrol-different-resource"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req delegateaccesscontrolsdk.GetWorkRequestRequest) (delegateaccesscontrolsdk.GetWorkRequestResponse, error) {
			requireDelegationControlStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return delegateaccesscontrolsdk.GetWorkRequestResponse{
				WorkRequest: makeDelegationControlWorkRequest(
					workRequestID,
					delegateaccesscontrolsdk.OperationTypeCreateDelegationControl,
					delegateaccesscontrolsdk.OperationStatusInProgress,
					delegateaccesscontrolsdk.ActionTypeInProgress,
					"",
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
	if getCalls != 1 {
		t.Fatalf("GetDelegationControl() calls = %d, want 1 strict reread before create", getCalls)
	}
	if listCalls != 2 {
		t.Fatalf("ListDelegationControls() calls = %d, want strict lookup plus generic fallback", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDelegationControl() calls = %d, want 1 when existing resource identity differs", createCalls)
	}
	requireDelegationControlAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
}

func TestDelegationControlServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.delegationcontrol.oc1..created"
		workRequestID = "wr-delegationcontrol-create"
	)

	resource := makeDelegationControlResource()
	resource.Spec.IsAutoApproveDuringMaintenance = false
	workRequests := map[string]delegateaccesscontrolsdk.WorkRequest{
		workRequestID: makeDelegationControlWorkRequest(
			workRequestID,
			delegateaccesscontrolsdk.OperationTypeCreateDelegationControl,
			delegateaccesscontrolsdk.OperationStatusInProgress,
			delegateaccesscontrolsdk.ActionTypeInProgress,
			"",
		),
	}

	var createRequest delegateaccesscontrolsdk.CreateDelegationControlRequest
	getCalls := 0
	listCalls := 0

	client := newTestDelegationControlClient(&fakeDelegationControlOCIClient{
		listFn: func(_ context.Context, req delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
			listCalls++
			requireDelegationControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireDelegationControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.ResourceId != nil {
				requireDelegationControlStringPtr(t, "list resourceId", req.ResourceId, "ocid1.vmcluster.oc1..one")
			}
			return delegateaccesscontrolsdk.ListDelegationControlsResponse{}, nil
		},
		createFn: func(_ context.Context, req delegateaccesscontrolsdk.CreateDelegationControlRequest) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error) {
			createRequest = req
			return delegateaccesscontrolsdk.CreateDelegationControlResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-delegationcontrol"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req delegateaccesscontrolsdk.GetWorkRequestRequest) (delegateaccesscontrolsdk.GetWorkRequestResponse, error) {
			requireDelegationControlStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return delegateaccesscontrolsdk.GetWorkRequestResponse{
				WorkRequest: workRequests[workRequestID],
			}, nil
		},
		getFn: func(_ context.Context, req delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
			getCalls++
			requireDelegationControlStringPtr(t, "get delegationControlId", req.DelegationControlId, createdID)
			return delegateaccesscontrolsdk.GetDelegationControlResponse{
				DelegationControl: makeSDKDelegationControl(
					createdID,
					resource,
					delegateaccesscontrolsdk.DelegationControlLifecycleStateActive,
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
	requireDelegationControlStringPtr(t, "create compartmentId", createRequest.CreateDelegationControlDetails.CompartmentId, resource.Spec.CompartmentId)
	requireDelegationControlStringPtr(t, "create notificationTopicId", createRequest.CreateDelegationControlDetails.NotificationTopicId, resource.Spec.NotificationTopicId)
	requireDelegationControlBoolPtr(t, "create isAutoApproveDuringMaintenance", createRequest.CreateDelegationControlDetails.IsAutoApproveDuringMaintenance, false)
	if listCalls == 0 {
		t.Fatal("ListDelegationControls() calls = 0, want pre-create lookup before create")
	}
	if getCalls != 0 {
		t.Fatalf("GetDelegationControl() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireDelegationControlAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-delegationcontrol" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-delegationcontrol", got)
	}

	workRequests[workRequestID] = makeDelegationControlWorkRequest(
		workRequestID,
		delegateaccesscontrolsdk.OperationTypeCreateDelegationControl,
		delegateaccesscontrolsdk.OperationStatusSucceeded,
		delegateaccesscontrolsdk.ActionTypeCreated,
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
		t.Fatalf("GetDelegationControl() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(delegateaccesscontrolsdk.DelegationControlLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDelegationControlDeleteResolvesPendingCreateWithoutTrackedID(t *testing.T) {
	t.Parallel()

	const (
		createdID         = "ocid1.delegationcontrol.oc1..created"
		createWorkRequest = "wr-delegationcontrol-create"
		deleteWorkRequest = "wr-delegationcontrol-delete"
	)

	resource := makeDelegationControlResource()
	listCalls := 0
	deleteCalls := 0

	client := newTestDelegationControlClient(&fakeDelegationControlOCIClient{
		listFn: func(_ context.Context, req delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
			listCalls++
			requireDelegationControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireDelegationControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			if req.ResourceType != delegateaccesscontrolsdk.ListDelegationControlsResourceTypeEnum(resource.Spec.ResourceType) {
				t.Fatalf("list resourceType = %q, want %q", req.ResourceType, resource.Spec.ResourceType)
			}
			switch listCalls {
			case 1, 2:
				return delegateaccesscontrolsdk.ListDelegationControlsResponse{}, nil
			case 3:
				return delegateaccesscontrolsdk.ListDelegationControlsResponse{
					DelegationControlSummaryCollection: delegateaccesscontrolsdk.DelegationControlSummaryCollection{
						Items: []delegateaccesscontrolsdk.DelegationControlSummary{
							makeSDKDelegationControlSummary(createdID, resource, delegateaccesscontrolsdk.DelegationControlLifecycleStateCreating),
						},
					},
				}, nil
			default:
				t.Fatalf("ListDelegationControls() unexpected call %d", listCalls)
				return delegateaccesscontrolsdk.ListDelegationControlsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
			requireDelegationControlStringPtr(t, "get delegationControlId", req.DelegationControlId, createdID)
			return delegateaccesscontrolsdk.GetDelegationControlResponse{
				DelegationControl: makeSDKDelegationControl(
					createdID,
					resource,
					delegateaccesscontrolsdk.DelegationControlLifecycleStateCreating,
				),
			}, nil
		},
		createFn: func(_ context.Context, _ delegateaccesscontrolsdk.CreateDelegationControlRequest) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error) {
			return delegateaccesscontrolsdk.CreateDelegationControlResponse{
				OpcWorkRequestId: common.String(createWorkRequest),
				OpcRequestId:     common.String("opc-create-delegationcontrol"),
			}, nil
		},
		deleteFn: func(_ context.Context, req delegateaccesscontrolsdk.DeleteDelegationControlRequest) (delegateaccesscontrolsdk.DeleteDelegationControlResponse, error) {
			deleteCalls++
			requireDelegationControlStringPtr(t, "delete delegationControlId", req.DelegationControlId, createdID)
			if req.Description != nil {
				t.Fatalf("delete description = %v, want nil", req.Description)
			}
			return delegateaccesscontrolsdk.DeleteDelegationControlResponse{
				OpcWorkRequestId: common.String(deleteWorkRequest),
				OpcRequestId:     common.String("opc-delete-delegationcontrol"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req delegateaccesscontrolsdk.GetWorkRequestRequest) (delegateaccesscontrolsdk.GetWorkRequestResponse, error) {
			switch stringValue(req.WorkRequestId) {
			case createWorkRequest:
				return delegateaccesscontrolsdk.GetWorkRequestResponse{
					WorkRequest: makeDelegationControlWorkRequest(
						createWorkRequest,
						delegateaccesscontrolsdk.OperationTypeCreateDelegationControl,
						delegateaccesscontrolsdk.OperationStatusInProgress,
						delegateaccesscontrolsdk.ActionTypeInProgress,
						"",
					),
				}, nil
			case deleteWorkRequest:
				return delegateaccesscontrolsdk.GetWorkRequestResponse{
					WorkRequest: makeDelegationControlWorkRequest(
						deleteWorkRequest,
						delegateaccesscontrolsdk.OperationTypeDeleteDelegationControl,
						delegateaccesscontrolsdk.OperationStatusInProgress,
						delegateaccesscontrolsdk.ActionTypeInProgress,
						createdID,
					),
				}, nil
			default:
				t.Fatalf("GetWorkRequest() unexpected work request id %q", stringValue(req.WorkRequestId))
				return delegateaccesscontrolsdk.GetWorkRequestResponse{}, nil
			}
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	if resource.Status.Id != "" {
		t.Fatalf("status.id = %q, want empty before create work request bind resolves", resource.Status.Id)
	}
	requireDelegationControlAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, createWorkRequest, shared.OSOKAsyncClassPending)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want pending delete while delete work request is in progress")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDelegationControl() calls = %d, want 1 after no-ID delete resolution", deleteCalls)
	}
	if listCalls != 3 {
		t.Fatalf("ListDelegationControls() calls = %d, want 3 across create lookup, create fallback, and delete resolution", listCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q after delete resolved the OCI identifier", got, createdID)
	}
	requireDelegationControlAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, deleteWorkRequest, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-delegationcontrol" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-delegationcontrol", got)
	}
}

func TestDelegationControlDeleteTreatsNoTrackedIDAndNoRemoteMatchAsDeleted(t *testing.T) {
	t.Parallel()

	resource := makeDelegationControlResource()
	deleteCalls := 0
	listCalls := 0

	client := newTestDelegationControlClient(&fakeDelegationControlOCIClient{
		listFn: func(_ context.Context, req delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
			listCalls++
			requireDelegationControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireDelegationControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			requireDelegationControlStringPtr(t, "list resourceId", req.ResourceId, "ocid1.vmcluster.oc1..one")
			if req.ResourceType != delegateaccesscontrolsdk.ListDelegationControlsResourceTypeEnum(resource.Spec.ResourceType) {
				t.Fatalf("list resourceType = %q, want %q", req.ResourceType, resource.Spec.ResourceType)
			}
			return delegateaccesscontrolsdk.ListDelegationControlsResponse{}, nil
		},
		deleteFn: func(_ context.Context, _ delegateaccesscontrolsdk.DeleteDelegationControlRequest) (delegateaccesscontrolsdk.DeleteDelegationControlResponse, error) {
			deleteCalls++
			return delegateaccesscontrolsdk.DeleteDelegationControlResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want already-absent resource to clear the finalizer path")
	}
	if listCalls != 1 {
		t.Fatalf("ListDelegationControls() calls = %d, want 1 delete-time strict lookup", listCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteDelegationControl() calls = %d, want 0 when no remote match exists", deleteCalls)
	}
	if got := resource.Status.OsokStatus.Message; got != "OCI resource no longer exists" {
		t.Fatalf("status.message = %q, want OCI resource no longer exists", got)
	}
}

func TestDelegationControlDeleteIgnoresSpecDescriptionField(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.delegationcontrol.oc1..existing"

	resource := newExistingDelegationControlResource(existingID)
	resource.Spec.Description = "resource description must not become delete reason"

	client := newTestDelegationControlClient(&fakeDelegationControlOCIClient{
		deleteFn: func(_ context.Context, req delegateaccesscontrolsdk.DeleteDelegationControlRequest) (delegateaccesscontrolsdk.DeleteDelegationControlResponse, error) {
			requireDelegationControlStringPtr(t, "delete delegationControlId", req.DelegationControlId, existingID)
			if req.Description != nil {
				t.Fatalf("delete description = %v, want nil", req.Description)
			}
			return delegateaccesscontrolsdk.DeleteDelegationControlResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after OCI reports NotFound")
	}
}

func TestDelegationControlCreateOrUpdateClassifiesNeedsAttentionAsFailed(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.delegationcontrol.oc1..existing"

	resource := newExistingDelegationControlResource(existingID)

	client := newTestDelegationControlClient(&fakeDelegationControlOCIClient{
		getFn: func(_ context.Context, req delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
			requireDelegationControlStringPtr(t, "get delegationControlId", req.DelegationControlId, existingID)
			return delegateaccesscontrolsdk.GetDelegationControlResponse{
				DelegationControl: makeSDKDelegationControl(
					existingID,
					resource,
					delegateaccesscontrolsdk.DelegationControlLifecycleStateNeedsAttention,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want nil lifecycle classification result", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful result", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want terminal failure without requeue", response)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Failed)
	}
	if resource.Status.LifecycleState != string(delegateaccesscontrolsdk.DelegationControlLifecycleStateNeedsAttention) {
		t.Fatalf("status.lifecycleState = %q, want NEEDS_ATTENTION", resource.Status.LifecycleState)
	}
}

func newTestDelegationControlClient(client *fakeDelegationControlOCIClient) DelegationControlServiceClient {
	if client == nil {
		client = &fakeDelegationControlOCIClient{}
	}
	return newDelegationControlServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDelegationControlResource() *delegateaccesscontrolv1beta1.DelegationControl {
	return &delegateaccesscontrolv1beta1.DelegationControl{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delegation-control-sample",
			Namespace: "default",
		},
		Spec: delegateaccesscontrolv1beta1.DelegationControlSpec{
			CompartmentId: "ocid1.compartment.oc1..delegateaccesscontrolexample",
			DisplayName:   "delegation-control-sample",
			DelegationSubscriptionIds: []string{
				"ocid1.delegationsubscription.oc1..subscriptiontwo",
				"ocid1.delegationsubscription.oc1..subscriptionone",
			},
			ResourceIds: []string{
				"ocid1.vmcluster.oc1..two",
				"ocid1.vmcluster.oc1..one",
			},
			ResourceType:                          string(delegateaccesscontrolsdk.DelegationControlResourceTypeVmcluster),
			NotificationTopicId:                   "ocid1.onstopic.oc1..delegateaccesscontrol",
			NotificationMessageFormat:             string(delegateaccesscontrolsdk.DelegationControlNotificationMessageFormatJson),
			Description:                           "delegate managed access to an Exadata resource",
			NumApprovalsRequired:                  2,
			PreApprovedServiceProviderActionNames: []string{"PATCH_CLUSTER", "VIEW_CLUSTER"},
			IsAutoApproveDuringMaintenance:        true,
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

func newExistingDelegationControlResource(id string) *delegateaccesscontrolv1beta1.DelegationControl {
	resource := makeDelegationControlResource()
	resource.Status = delegateaccesscontrolv1beta1.DelegationControlStatus{
		OsokStatus:                            resource.Status.OsokStatus,
		Id:                                    id,
		DisplayName:                           resource.Spec.DisplayName,
		CompartmentId:                         resource.Spec.CompartmentId,
		ResourceType:                          resource.Spec.ResourceType,
		DelegationSubscriptionIds:             append([]string(nil), resource.Spec.DelegationSubscriptionIds...),
		ResourceIds:                           append([]string(nil), resource.Spec.ResourceIds...),
		NotificationTopicId:                   resource.Spec.NotificationTopicId,
		NotificationMessageFormat:             resource.Spec.NotificationMessageFormat,
		Description:                           resource.Spec.Description,
		NumApprovalsRequired:                  resource.Spec.NumApprovalsRequired,
		PreApprovedServiceProviderActionNames: append([]string(nil), resource.Spec.PreApprovedServiceProviderActionNames...),
		IsAutoApproveDuringMaintenance:        resource.Spec.IsAutoApproveDuringMaintenance,
		VaultId:                               resource.Spec.VaultId,
		VaultKeyId:                            resource.Spec.VaultKeyId,
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	return resource
}

func makeSDKDelegationControl(
	id string,
	resource *delegateaccesscontrolv1beta1.DelegationControl,
	state delegateaccesscontrolsdk.DelegationControlLifecycleStateEnum,
) delegateaccesscontrolsdk.DelegationControl {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	current := delegateaccesscontrolsdk.DelegationControl{
		Id:                                    common.String(id),
		CompartmentId:                         common.String(resource.Spec.CompartmentId),
		DisplayName:                           common.String(resource.Spec.DisplayName),
		ResourceType:                          delegateaccesscontrolsdk.DelegationControlResourceTypeEnum(resource.Spec.ResourceType),
		Description:                           optionalString(resource.Spec.Description),
		NumApprovalsRequired:                  optionalInt(resource.Spec.NumApprovalsRequired),
		PreApprovedServiceProviderActionNames: append([]string(nil), resource.Spec.PreApprovedServiceProviderActionNames...),
		DelegationSubscriptionIds:             append([]string(nil), resource.Spec.DelegationSubscriptionIds...),
		IsAutoApproveDuringMaintenance:        common.Bool(resource.Spec.IsAutoApproveDuringMaintenance),
		ResourceIds:                           append([]string(nil), resource.Spec.ResourceIds...),
		NotificationTopicId:                   common.String(resource.Spec.NotificationTopicId),
		NotificationMessageFormat:             delegateaccesscontrolsdk.DelegationControlNotificationMessageFormatEnum(resource.Spec.NotificationMessageFormat),
		VaultId:                               optionalString(resource.Spec.VaultId),
		VaultKeyId:                            optionalString(resource.Spec.VaultKeyId),
		LifecycleState:                        state,
		LifecycleStateDetails:                 common.String("reviewed runtime"),
		TimeCreated:                           &now,
		TimeUpdated:                           &now,
		FreeformTags:                          maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:                           delegationControlDefinedTagsFromSpec(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
	return current
}

func makeSDKDelegationControlSummary(
	id string,
	resource *delegateaccesscontrolv1beta1.DelegationControl,
	state delegateaccesscontrolsdk.DelegationControlLifecycleStateEnum,
) delegateaccesscontrolsdk.DelegationControlSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return delegateaccesscontrolsdk.DelegationControlSummary{
		Id:                    common.String(id),
		DisplayName:           common.String(resource.Spec.DisplayName),
		CompartmentId:         common.String(resource.Spec.CompartmentId),
		ResourceType:          delegateaccesscontrolsdk.DelegationControlResourceTypeEnum(resource.Spec.ResourceType),
		TimeCreated:           &now,
		TimeUpdated:           &now,
		LifecycleState:        state,
		LifecycleStateDetails: common.String("reviewed runtime"),
		FreeformTags:          maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:           delegationControlDefinedTagsFromSpec(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeDelegationControlWorkRequest(
	id string,
	operation delegateaccesscontrolsdk.OperationTypeEnum,
	status delegateaccesscontrolsdk.OperationStatusEnum,
	action delegateaccesscontrolsdk.ActionTypeEnum,
	resourceID string,
) delegateaccesscontrolsdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return delegateaccesscontrolsdk.WorkRequest{
		OperationType: operation,
		Status:        status,
		Id:            common.String(id),
		CompartmentId: common.String("ocid1.compartment.oc1..delegateaccesscontrolexample"),
		Resources: []delegateaccesscontrolsdk.WorkRequestResource{
			{
				EntityType: common.String("DelegationControl"),
				ActionType: action,
				Identifier: optionalString(resourceID),
			},
		},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
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

func assertDelegationControlStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireDelegationControlStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireDelegationControlBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %t", name, got, want)
	}
}

func requireDelegationControlAsyncCurrent(
	t *testing.T,
	resource *delegateaccesscontrolv1beta1.DelegationControl,
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
