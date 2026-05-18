/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sdmmaskingpolicydifference

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSdmMaskingPolicyDifferenceID               = "ocid1.sdmmaskingpolicydifference.oc1..resource"
	testSdmMaskingPolicyDifferenceCompartmentID    = "ocid1.compartment.oc1..resource"
	testSdmMaskingPolicyDifferenceMaskingPolicyID  = "ocid1.datasafemaskingpolicy.oc1..policy"
	testSdmMaskingPolicyDifferenceSensitiveModelID = "ocid1.datasafesensitivedatamodel.oc1..model"
)

func TestSdmMaskingPolicyDifferenceRuntimeSemantics(t *testing.T) {
	t.Parallel()

	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooks()
	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil")
	}
	if hooks.DeleteHooks.ConfirmRead == nil {
		t.Fatal("hooks.DeleteHooks.ConfirmRead = nil")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil")
	}
	if hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("hooks.DeleteHooks.ApplyOutcome = nil")
	}

	assertSdmMaskingPolicyDifferenceStringEqual(t, "formal service", hooks.Semantics.FormalService, "datasafe")
	assertSdmMaskingPolicyDifferenceStringEqual(t, "formal slug", hooks.Semantics.FormalSlug, "sdmmaskingpolicydifference")
	assertSdmMaskingPolicyDifferenceStringEqual(t, "delete policy", hooks.Semantics.Delete.Policy, "required")
	assertSdmMaskingPolicyDifferenceStringSliceEqual(t, "mutable fields", hooks.Semantics.Mutation.Mutable, []string{
		"displayName",
		"freeformTags",
		"definedTags",
	})
	assertSdmMaskingPolicyDifferenceStringSliceEqual(t, "force-new fields", hooks.Semantics.Mutation.ForceNew, []string{
		"maskingPolicyId",
		"compartmentId",
		"differenceType",
	})
}

func TestSdmMaskingPolicyDifferenceCreateProjectsLifecycleAndRequestIDs(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	var createRequest datasafesdk.CreateSdmMaskingPolicyDifferenceRequest
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.List.Call = func(context.Context, datasafesdk.ListSdmMaskingPolicyDifferencesRequest) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error) {
			return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{}, nil
		}
		hooks.Create.Call = func(_ context.Context, request datasafesdk.CreateSdmMaskingPolicyDifferenceRequest) (datasafesdk.CreateSdmMaskingPolicyDifferenceResponse, error) {
			createRequest = request
			return datasafesdk.CreateSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					resource.Spec,
					datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateCreating,
				),
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
			}, nil
		}
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			requireSdmMaskingPolicyDifferenceStringPtr(t, "get sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, testSdmMaskingPolicyDifferenceID)
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					resource.Spec,
					datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateCreating,
				),
			}, nil
		}
	})

	response, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while CREATING")
	}
	assertSdmMaskingPolicyDifferenceCreateRequest(t, createRequest, resource)
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.id", resource.Status.Id, testSdmMaskingPolicyDifferenceID)
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.ocid", string(resource.Status.OsokStatus.Ocid), testSdmMaskingPolicyDifferenceID)
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.lifecycleState", resource.Status.LifecycleState, "CREATING")
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	requireSdmMaskingPolicyDifferenceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "CREATING", shared.OSOKAsyncClassPending, "wr-create-1")
}

func TestSdmMaskingPolicyDifferenceBindsExistingMatchAcrossListPages(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.sdmmaskingpolicydifference.oc1..existing"
	resource := newSdmMaskingPolicyDifferenceTestResource()
	listCalls := 0
	createCalled := false
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Create.Call = func(context.Context, datasafesdk.CreateSdmMaskingPolicyDifferenceRequest) (datasafesdk.CreateSdmMaskingPolicyDifferenceResponse, error) {
			createCalled = true
			t.Fatal("CreateSdmMaskingPolicyDifference() should not be called when a reusable list match exists")
			return datasafesdk.CreateSdmMaskingPolicyDifferenceResponse{}, nil
		}
		hooks.List.Call = func(_ context.Context, request datasafesdk.ListSdmMaskingPolicyDifferencesRequest) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error) {
			listCalls++
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list maskingPolicyId", request.MaskingPolicyId, resource.Spec.MaskingPolicyId)
			switch listCalls {
			case 1:
				if request.Page != nil {
					t.Fatalf("first list page = %q, want nil", *request.Page)
				}
				otherSpec := resource.Spec
				otherSpec.DisplayName = "other-difference"
				return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{
					SdmMaskingPolicyDifferenceCollection: datasafesdk.SdmMaskingPolicyDifferenceCollection{
						Items: []datasafesdk.SdmMaskingPolicyDifferenceSummary{
							observedSdmMaskingPolicyDifferenceSummaryFromSpec("ocid1.sdmmaskingpolicydifference.oc1..other", otherSpec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireSdmMaskingPolicyDifferenceStringPtr(t, "second list page", request.Page, "page-2")
				return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{
					SdmMaskingPolicyDifferenceCollection: datasafesdk.SdmMaskingPolicyDifferenceCollection{
						Items: []datasafesdk.SdmMaskingPolicyDifferenceSummary{
							observedSdmMaskingPolicyDifferenceSummaryFromSpec(existingID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected ListSdmMaskingPolicyDifferences() call %d", listCalls)
				return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{}, nil
			}
		}
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			requireSdmMaskingPolicyDifferenceStringPtr(t, "get sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, existingID)
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(existingID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive),
			}, nil
		}
	})

	response, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if createCalled {
		t.Fatal("CreateSdmMaskingPolicyDifference() was called unexpectedly")
	}
	if listCalls != 2 {
		t.Fatalf("ListSdmMaskingPolicyDifferences() calls = %d, want 2", listCalls)
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.ocid", string(resource.Status.OsokStatus.Ocid), existingID)
}

func TestSdmMaskingPolicyDifferenceNoopObserveDoesNotUpdateWhenSpecMatches(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	seedSdmMaskingPolicyDifferenceStatus(resource, testSdmMaskingPolicyDifferenceID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive)
	updateCalled := false
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			requireSdmMaskingPolicyDifferenceStringPtr(t, "get sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, testSdmMaskingPolicyDifferenceID)
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					resource.Spec,
					datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive,
				),
			}, nil
		}
		hooks.Update.Call = func(context.Context, datasafesdk.UpdateSdmMaskingPolicyDifferenceRequest) (datasafesdk.UpdateSdmMaskingPolicyDifferenceResponse, error) {
			updateCalled = true
			t.Fatal("UpdateSdmMaskingPolicyDifference() should not be called when desired and live state match")
			return datasafesdk.UpdateSdmMaskingPolicyDifferenceResponse{}, nil
		}
	})

	response, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observe", response)
	}
	if updateCalled {
		t.Fatal("UpdateSdmMaskingPolicyDifference() was called unexpectedly")
	}
	requireSdmMaskingPolicyDifferenceCondition(t, resource, shared.Active)
}

func TestSdmMaskingPolicyDifferenceMutableUpdateRefreshesObservedState(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	previousSpec := resource.Spec
	previousSpec.DisplayName = "previous-difference"
	previousSpec.FreeformTags = map[string]string{"team": "legacy"}
	previousSpec.DefinedTags = map[string]shared.MapValue{
		"Operations": {
			"CostCenter": "7",
		},
	}
	seedSdmMaskingPolicyDifferenceStatus(resource, testSdmMaskingPolicyDifferenceID, previousSpec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive)

	getCalls := 0
	var updateRequest datasafesdk.UpdateSdmMaskingPolicyDifferenceRequest
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			getCalls++
			requireSdmMaskingPolicyDifferenceStringPtr(t, "get sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, testSdmMaskingPolicyDifferenceID)
			spec := previousSpec
			if getCalls > 1 {
				spec = resource.Spec
			}
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					spec,
					datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive,
				),
			}, nil
		}
		hooks.Update.Call = func(_ context.Context, request datasafesdk.UpdateSdmMaskingPolicyDifferenceRequest) (datasafesdk.UpdateSdmMaskingPolicyDifferenceResponse, error) {
			updateRequest = request
			return datasafesdk.UpdateSdmMaskingPolicyDifferenceResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		}
	})

	response, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue update", response)
	}
	requireSdmMaskingPolicyDifferenceStringPtr(t, "update sdmMaskingPolicyDifferenceId", updateRequest.SdmMaskingPolicyDifferenceId, testSdmMaskingPolicyDifferenceID)
	requireSdmMaskingPolicyDifferenceStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	if got := updateRequest.FreeformTags["team"]; got != "security" {
		t.Fatalf("update freeformTags[team] = %q, want security", got)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 42", got)
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.displayName", resource.Status.DisplayName, resource.Spec.DisplayName)
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
}

func TestSdmMaskingPolicyDifferenceRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	previousSpec := resource.Spec
	previousSpec.MaskingPolicyId = "ocid1.datasafemaskingpolicy.oc1..previous"
	seedSdmMaskingPolicyDifferenceStatus(resource, testSdmMaskingPolicyDifferenceID, previousSpec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive)
	updateCalled := false
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Get.Call = func(context.Context, datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					previousSpec,
					datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive,
				),
			}, nil
		}
		hooks.Update.Call = func(context.Context, datasafesdk.UpdateSdmMaskingPolicyDifferenceRequest) (datasafesdk.UpdateSdmMaskingPolicyDifferenceResponse, error) {
			updateCalled = true
			t.Fatal("UpdateSdmMaskingPolicyDifference() should not be called after create-only maskingPolicyId drift")
			return datasafesdk.UpdateSdmMaskingPolicyDifferenceResponse{}, nil
		}
	})

	response, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "maskingPolicyId") {
		t.Fatalf("CreateOrUpdate() error = %v, want maskingPolicyId create-only drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	if updateCalled {
		t.Fatal("UpdateSdmMaskingPolicyDifference() was called unexpectedly")
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.reason", resource.Status.OsokStatus.Reason, string(shared.Failed))
}

func TestSdmMaskingPolicyDifferenceDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	seedSdmMaskingPolicyDifferenceStatus(resource, testSdmMaskingPolicyDifferenceID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive)

	getCalls := 0
	deleteCalls := 0
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			getCalls++
			requireSdmMaskingPolicyDifferenceStringPtr(t, "get sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, testSdmMaskingPolicyDifferenceID)
			state := datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive
			if getCalls > 1 {
				state = datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateDeleting
			}
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					resource.Spec,
					state,
				),
			}, nil
		}
		hooks.Delete.Call = func(_ context.Context, request datasafesdk.DeleteSdmMaskingPolicyDifferenceRequest) (datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse, error) {
			deleteCalls++
			requireSdmMaskingPolicyDifferenceStringPtr(t, "delete sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, testSdmMaskingPolicyDifferenceID)
			return datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		}
	})

	deleted, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle remains DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteSdmMaskingPolicyDifference() calls = %d, want 1", deleteCalls)
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.lifecycleState", resource.Status.LifecycleState, "DELETING")
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-delete-1")
	requireSdmMaskingPolicyDifferenceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "", shared.OSOKAsyncClassPending, "wr-delete-1")
}

func TestSdmMaskingPolicyDifferenceDeleteConfirmsDeletedWithoutCallingDelete(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	seedSdmMaskingPolicyDifferenceStatus(resource, testSdmMaskingPolicyDifferenceID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateDeleting)
	deleteCalled := false
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Get.Call = func(context.Context, datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					resource.Spec,
					datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateDeleted,
				),
			}, nil
		}
		hooks.Delete.Call = func(context.Context, datasafesdk.DeleteSdmMaskingPolicyDifferenceRequest) (datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteSdmMaskingPolicyDifference() should not be called after DELETED readback")
			return datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse{}, nil
		}
	})

	deleted, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED readback")
	}
	if deleteCalled {
		t.Fatal("DeleteSdmMaskingPolicyDifference() was called unexpectedly")
	}
}

func TestSdmMaskingPolicyDifferenceDeleteResolvesMissingStatusIDFromPaginatedList(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.sdmmaskingpolicydifference.oc1..deleteexisting"
	resource := newSdmMaskingPolicyDifferenceTestResource()
	listCalls := 0
	getCalls := 0
	deleteCalls := 0
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.List.Call = func(_ context.Context, request datasafesdk.ListSdmMaskingPolicyDifferencesRequest) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error) {
			listCalls++
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list maskingPolicyId", request.MaskingPolicyId, resource.Spec.MaskingPolicyId)
			switch listCalls {
			case 1:
				if request.Page != nil {
					t.Fatalf("first list page = %q, want nil", *request.Page)
				}
				otherSpec := resource.Spec
				otherSpec.DifferenceType = string(datasafesdk.SdmMaskingPolicyDifferenceDifferenceTypeNew)
				return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{
					SdmMaskingPolicyDifferenceCollection: datasafesdk.SdmMaskingPolicyDifferenceCollection{
						Items: []datasafesdk.SdmMaskingPolicyDifferenceSummary{
							observedSdmMaskingPolicyDifferenceSummaryFromSpec("ocid1.sdmmaskingpolicydifference.oc1..other", otherSpec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireSdmMaskingPolicyDifferenceStringPtr(t, "second list page", request.Page, "page-2")
				return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{
					SdmMaskingPolicyDifferenceCollection: datasafesdk.SdmMaskingPolicyDifferenceCollection{
						Items: []datasafesdk.SdmMaskingPolicyDifferenceSummary{
							observedSdmMaskingPolicyDifferenceSummaryFromSpec(existingID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected ListSdmMaskingPolicyDifferences() call %d", listCalls)
				return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{}, nil
			}
		}
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			getCalls++
			requireSdmMaskingPolicyDifferenceStringPtr(t, "get sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, existingID)
			state := datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive
			if getCalls > 1 {
				state = datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateDeleting
			}
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(existingID, resource.Spec, state),
			}, nil
		}
		hooks.Delete.Call = func(_ context.Context, request datasafesdk.DeleteSdmMaskingPolicyDifferenceRequest) (datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse, error) {
			deleteCalls++
			requireSdmMaskingPolicyDifferenceStringPtr(t, "delete sdmMaskingPolicyDifferenceId", request.SdmMaskingPolicyDifferenceId, existingID)
			return datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse{
				OpcRequestId:     common.String("opc-delete-list"),
				OpcWorkRequestId: common.String("wr-delete-list"),
			}, nil
		}
	})

	deleted, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while recovered resource is DELETING")
	}
	if listCalls != 2 {
		t.Fatalf("ListSdmMaskingPolicyDifferences() calls = %d, want 2", listCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteSdmMaskingPolicyDifference() calls = %d, want 1", deleteCalls)
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.id", resource.Status.Id, existingID)
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.ocid", string(resource.Status.OsokStatus.Ocid), existingID)
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.lifecycleState", resource.Status.LifecycleState, "DELETING")
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-delete-list")
}

func TestSdmMaskingPolicyDifferenceDeleteWithNoStatusIDAndNoListMatchConfirmsDeleted(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	getCalled := false
	deleteCalled := false
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.List.Call = func(_ context.Context, request datasafesdk.ListSdmMaskingPolicyDifferencesRequest) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error) {
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list maskingPolicyId", request.MaskingPolicyId, resource.Spec.MaskingPolicyId)
			return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{}, nil
		}
		hooks.Get.Call = func(context.Context, datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			getCalled = true
			t.Fatal("GetSdmMaskingPolicyDifference() should not be called when no list match exists")
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{}, nil
		}
		hooks.Delete.Call = func(context.Context, datasafesdk.DeleteSdmMaskingPolicyDifferenceRequest) (datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteSdmMaskingPolicyDifference() should not be called when no list match exists")
			return datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse{}, nil
		}
	})

	deleted, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when no OCI resource is found")
	}
	if getCalled {
		t.Fatal("GetSdmMaskingPolicyDifference() was called unexpectedly")
	}
	if deleteCalled {
		t.Fatal("DeleteSdmMaskingPolicyDifference() was called unexpectedly")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestSdmMaskingPolicyDifferenceDeleteWithNoStatusIDRejectsAuthShapedListFailure(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-list-confirm-1"

	resource := newSdmMaskingPolicyDifferenceTestResource()
	getCalled := false
	deleteCalled := false
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.List.Call = func(_ context.Context, request datasafesdk.ListSdmMaskingPolicyDifferencesRequest) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error) {
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			requireSdmMaskingPolicyDifferenceStringPtr(t, "list maskingPolicyId", request.MaskingPolicyId, resource.Spec.MaskingPolicyId)
			return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{}, serviceErr
		}
		hooks.Get.Call = func(context.Context, datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			getCalled = true
			t.Fatal("GetSdmMaskingPolicyDifference() should not be called when no status OCID exists and list confirmation is ambiguous")
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{}, nil
		}
		hooks.Delete.Call = func(context.Context, datasafesdk.DeleteSdmMaskingPolicyDifferenceRequest) (datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteSdmMaskingPolicyDifference() should not be called after ambiguous list confirmation")
			return datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse{}, nil
		}
	})

	deleted, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous list confirmation refusal", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false after ambiguous list confirmation")
	}
	if getCalled {
		t.Fatal("GetSdmMaskingPolicyDifference() was called unexpectedly")
	}
	if deleteCalled {
		t.Fatal("DeleteSdmMaskingPolicyDifference() was called unexpectedly")
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-list-confirm-1")
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer-retaining delete status")
	}
}

func TestSdmMaskingPolicyDifferenceDeleteRejectsAuthShapedPreReadWithoutCallingDelete(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	seedSdmMaskingPolicyDifferenceStatus(resource, testSdmMaskingPolicyDifferenceID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive)
	deleteCalled := false
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Get.Call = func(context.Context, datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		}
		hooks.Delete.Call = func(context.Context, datasafesdk.DeleteSdmMaskingPolicyDifferenceRequest) (datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteSdmMaskingPolicyDifference() should not be called after ambiguous pre-read")
			return datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse{}, nil
		}
	})

	deleted, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous pre-read refusal", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false after ambiguous pre-read")
	}
	if deleteCalled {
		t.Fatal("DeleteSdmMaskingPolicyDifference() was called unexpectedly")
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
}

func TestSdmMaskingPolicyDifferenceDeleteRejectsAuthShapedDeleteError(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	seedSdmMaskingPolicyDifferenceStatus(resource, testSdmMaskingPolicyDifferenceID, resource.Spec, datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive)
	deleteCalls := 0
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.Get.Call = func(context.Context, datasafesdk.GetSdmMaskingPolicyDifferenceRequest) (datasafesdk.GetSdmMaskingPolicyDifferenceResponse, error) {
			return datasafesdk.GetSdmMaskingPolicyDifferenceResponse{
				SdmMaskingPolicyDifference: observedSdmMaskingPolicyDifferenceFromSpec(
					testSdmMaskingPolicyDifferenceID,
					resource.Spec,
					datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateActive,
				),
			}, nil
		}
		hooks.Delete.Call = func(context.Context, datasafesdk.DeleteSdmMaskingPolicyDifferenceRequest) (datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse, error) {
			deleteCalls++
			return datasafesdk.DeleteSdmMaskingPolicyDifferenceResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		}
	})

	deleted, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous delete refusal", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false after ambiguous delete error")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteSdmMaskingPolicyDifference() calls = %d, want 1", deleteCalls)
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
}

func TestSdmMaskingPolicyDifferenceCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	hooks := newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(func(hooks *SdmMaskingPolicyDifferenceRuntimeHooks) {
		hooks.List.Call = func(context.Context, datasafesdk.ListSdmMaskingPolicyDifferencesRequest) (datasafesdk.ListSdmMaskingPolicyDifferencesResponse, error) {
			return datasafesdk.ListSdmMaskingPolicyDifferencesResponse{}, nil
		}
		hooks.Create.Call = func(context.Context, datasafesdk.CreateSdmMaskingPolicyDifferenceRequest) (datasafesdk.CreateSdmMaskingPolicyDifferenceResponse, error) {
			return datasafesdk.CreateSdmMaskingPolicyDifferenceResponse{}, errortest.NewServiceError(500, "InternalServerError", "create failed")
		}
	})

	response, err := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful OCI failure", response)
	}
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
	assertSdmMaskingPolicyDifferenceStringEqual(t, "status.status.reason", resource.Status.OsokStatus.Reason, string(shared.Failed))
}

func newSdmMaskingPolicyDifferenceRuntimeTestHooks() SdmMaskingPolicyDifferenceRuntimeHooks {
	hooks := newSdmMaskingPolicyDifferenceDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applySdmMaskingPolicyDifferenceRuntimeHooks(&hooks)
	return hooks
}

func newSdmMaskingPolicyDifferenceRuntimeTestHooksWithOperations(
	mutate func(*SdmMaskingPolicyDifferenceRuntimeHooks),
) SdmMaskingPolicyDifferenceRuntimeHooks {
	hooks := newSdmMaskingPolicyDifferenceDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	if mutate != nil {
		mutate(&hooks)
	}
	applySdmMaskingPolicyDifferenceRuntimeHooks(&hooks)
	return hooks
}

func newSdmMaskingPolicyDifferenceRuntimeTestClient(
	hooks SdmMaskingPolicyDifferenceRuntimeHooks,
) SdmMaskingPolicyDifferenceServiceClient {
	manager := &SdmMaskingPolicyDifferenceServiceManager{}
	delegate := defaultSdmMaskingPolicyDifferenceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SdmMaskingPolicyDifference](
			buildSdmMaskingPolicyDifferenceGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSdmMaskingPolicyDifferenceGeneratedClient(hooks, delegate)
}

func newSdmMaskingPolicyDifferenceTestResource() *datasafev1beta1.SdmMaskingPolicyDifference {
	return &datasafev1beta1.SdmMaskingPolicyDifference{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sdm-masking-policy-difference",
			Namespace: "default",
			UID:       "sdm-masking-policy-difference-uid",
		},
		Spec: datasafev1beta1.SdmMaskingPolicyDifferenceSpec{
			MaskingPolicyId: testSdmMaskingPolicyDifferenceMaskingPolicyID,
			CompartmentId:   testSdmMaskingPolicyDifferenceCompartmentID,
			DifferenceType:  string(datasafesdk.SdmMaskingPolicyDifferenceDifferenceTypeAll),
			DisplayName:     "sdm-masking-policy-difference",
			FreeformTags: map[string]string{
				"team": "security",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func seedSdmMaskingPolicyDifferenceStatus(
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	id string,
	spec datasafev1beta1.SdmMaskingPolicyDifferenceSpec,
	state datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateEnum,
) {
	resource.Status.Id = id
	resource.Status.CompartmentId = spec.CompartmentId
	resource.Status.DifferenceType = spec.DifferenceType
	resource.Status.DisplayName = spec.DisplayName
	resource.Status.LifecycleState = string(state)
	resource.Status.SensitiveDataModelId = testSdmMaskingPolicyDifferenceSensitiveModelID
	resource.Status.MaskingPolicyId = spec.MaskingPolicyId
	resource.Status.FreeformTags = spec.FreeformTags
	resource.Status.DefinedTags = spec.DefinedTags
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func observedSdmMaskingPolicyDifferenceFromSpec(
	id string,
	spec datasafev1beta1.SdmMaskingPolicyDifferenceSpec,
	state datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateEnum,
) datasafesdk.SdmMaskingPolicyDifference {
	return datasafesdk.SdmMaskingPolicyDifference{
		Id:                   common.String(id),
		CompartmentId:        common.String(spec.CompartmentId),
		DifferenceType:       datasafesdk.SdmMaskingPolicyDifferenceDifferenceTypeEnum(spec.DifferenceType),
		DisplayName:          common.String(spec.DisplayName),
		LifecycleState:       state,
		SensitiveDataModelId: common.String(testSdmMaskingPolicyDifferenceSensitiveModelID),
		MaskingPolicyId:      common.String(spec.MaskingPolicyId),
		FreeformTags:         spec.FreeformTags,
		DefinedTags:          sdmMaskingPolicyDifferenceDefinedTags(spec.DefinedTags),
	}
}

func observedSdmMaskingPolicyDifferenceSummaryFromSpec(
	id string,
	spec datasafev1beta1.SdmMaskingPolicyDifferenceSpec,
	state datasafesdk.SdmMaskingPolicyDifferenceLifecycleStateEnum,
) datasafesdk.SdmMaskingPolicyDifferenceSummary {
	return datasafesdk.SdmMaskingPolicyDifferenceSummary{
		Id:                   common.String(id),
		CompartmentId:        common.String(spec.CompartmentId),
		DisplayName:          common.String(spec.DisplayName),
		SensitiveDataModelId: common.String(testSdmMaskingPolicyDifferenceSensitiveModelID),
		MaskingPolicyId:      common.String(spec.MaskingPolicyId),
		LifecycleState:       state,
		DifferenceType:       datasafesdk.SdmMaskingPolicyDifferenceDifferenceTypeEnum(spec.DifferenceType),
		FreeformTags:         spec.FreeformTags,
		DefinedTags:          sdmMaskingPolicyDifferenceDefinedTags(spec.DefinedTags),
	}
}

func assertSdmMaskingPolicyDifferenceCreateRequest(
	t *testing.T,
	request datasafesdk.CreateSdmMaskingPolicyDifferenceRequest,
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
) {
	t.Helper()
	requireSdmMaskingPolicyDifferenceStringPtr(t, "create maskingPolicyId", request.MaskingPolicyId, resource.Spec.MaskingPolicyId)
	requireSdmMaskingPolicyDifferenceStringPtr(t, "create compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	if request.DifferenceType != datasafesdk.SdmMaskingPolicyDifferenceDifferenceTypeEnum(resource.Spec.DifferenceType) {
		t.Fatalf("create differenceType = %q, want %q", request.DifferenceType, resource.Spec.DifferenceType)
	}
	requireSdmMaskingPolicyDifferenceStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	if got := request.FreeformTags["team"]; got != "security" {
		t.Fatalf("create freeformTags[team] = %q, want security", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("create opc-retry-token is empty, want deterministic token")
	}
}

func requireSdmMaskingPolicyDifferenceStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireSdmMaskingPolicyDifferenceAsyncCurrent(
	t *testing.T,
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil")
	}
	if current.Phase != phase {
		t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if rawStatus != "" && current.RawStatus != rawStatus {
		t.Fatalf("status.status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func requireSdmMaskingPolicyDifferenceCondition(
	t *testing.T,
	resource *datasafev1beta1.SdmMaskingPolicyDifference,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = empty, want %s", want)
	}
	got := conditions[len(conditions)-1].Type
	if got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func assertSdmMaskingPolicyDifferenceStringEqual(t *testing.T, label string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}

func assertSdmMaskingPolicyDifferenceStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", label, got, want)
		}
	}
}
