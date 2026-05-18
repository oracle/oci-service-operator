/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package auditarchiveretrieval

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAuditArchiveRetrievalID = "ocid1.auditarchiveretrieval.oc1..test"
	testCompartmentID           = "ocid1.compartment.oc1..test"
	testTargetID                = "ocid1.datasafetarget.oc1..test"
	testStartDate               = "2026-01-01T00:00:00Z"
	testEndDate                 = "2026-01-31T00:00:00Z"
)

func TestAuditArchiveRetrievalRuntimeSemantics(t *testing.T) {
	t.Parallel()

	semantics := newAuditArchiveRetrievalRuntimeSemantics()
	if semantics == nil {
		t.Fatal("newAuditArchiveRetrievalRuntimeSemantics() = nil")
	}
	assertAuditArchiveRetrievalStringEqual(t, "FormalService", semantics.FormalService, "datasafe")
	assertAuditArchiveRetrievalStringEqual(t, "FormalSlug", semantics.FormalSlug, "auditarchiveretrieval")
	assertAuditArchiveRetrievalStringEqual(t, "FinalizerPolicy", semantics.FinalizerPolicy, "retain-until-confirmed-delete")
	assertAuditArchiveRetrievalStringEqual(t, "Async.Strategy", semantics.Async.Strategy, "lifecycle")
	assertAuditArchiveRetrievalStringEqual(t, "Async.Runtime", semantics.Async.Runtime, "generatedruntime")
	assertAuditArchiveRetrievalStringEqual(t, "Delete.Policy", semantics.Delete.Policy, "required")
	assertAuditArchiveRetrievalStringEqual(t, "CreateFollowUp.Strategy", semantics.CreateFollowUp.Strategy, "read-after-write")
	assertAuditArchiveRetrievalStringEqual(t, "UpdateFollowUp.Strategy", semantics.UpdateFollowUp.Strategy, "read-after-write")
	assertAuditArchiveRetrievalStringEqual(t, "DeleteFollowUp.Strategy", semantics.DeleteFollowUp.Strategy, "confirm-delete")

	assertAuditArchiveRetrievalStringSliceEqual(t, "Lifecycle.ProvisioningStates", semantics.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertAuditArchiveRetrievalStringSliceEqual(t, "Lifecycle.UpdatingStates", semantics.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertAuditArchiveRetrievalStringSliceEqual(t, "Lifecycle.ActiveStates", semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertAuditArchiveRetrievalStringSliceEqual(t, "Delete.PendingStates", semantics.Delete.PendingStates, []string{"DELETING"})
	assertAuditArchiveRetrievalStringSliceEqual(t, "Delete.TerminalStates", semantics.Delete.TerminalStates, []string{"DELETED"})
	assertAuditArchiveRetrievalStringSliceEqual(t, "List.MatchFields", semantics.List.MatchFields, []string{
		"compartmentId",
		"targetId",
		"startDate",
		"endDate",
		"displayName",
	})
	assertAuditArchiveRetrievalStringSliceEqual(t, "Mutation.Mutable", semantics.Mutation.Mutable, []string{
		"displayName",
		"description",
		"freeformTags",
		"definedTags",
	})
	assertAuditArchiveRetrievalStringSliceEqual(t, "Mutation.ForceNew", semantics.Mutation.ForceNew, []string{
		"compartmentId",
		"targetId",
		"startDate",
		"endDate",
	})

	hooks := newAuditArchiveRetrievalRuntimeTestHooks()
	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed AuditArchiveRetrieval semantics")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("hooks.WrapGeneratedClient = empty, want delete pre-read wrapper")
	}
}

func TestAuditArchiveRetrievalCreateProjectsLifecycleAndRequestIDs(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		var createRequest datasafesdk.CreateAuditArchiveRetrievalRequest
		hooks.Create.Call = func(_ context.Context, request datasafesdk.CreateAuditArchiveRetrievalRequest) (datasafesdk.CreateAuditArchiveRetrievalResponse, error) {
			createRequest = request
			return datasafesdk.CreateAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateCreating),
				OpcRequestId:          common.String("opc-create-1"),
				OpcWorkRequestId:      common.String("wr-create-1"),
			}, nil
		}
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			requireAuditArchiveRetrievalStringPtr(t, "get auditArchiveRetrievalId", request.AuditArchiveRetrievalId, testAuditArchiveRetrievalID)
			assertAuditArchiveRetrievalCreateRequest(t, createRequest, resource)
			return datasafesdk.GetAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateCreating),
				OpcRequestId:          common.String("opc-get-1"),
			}, nil
		}
		hooks.List.Call = func(context.Context, datasafesdk.ListAuditArchiveRetrievalsRequest) (datasafesdk.ListAuditArchiveRetrievalsResponse, error) {
			return datasafesdk.ListAuditArchiveRetrievalsResponse{}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while AuditArchiveRetrieval is creating")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while AuditArchiveRetrieval is creating")
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.id", resource.Status.Id, testAuditArchiveRetrievalID)
	assertAuditArchiveRetrievalStringEqual(t, "status.status.ocid", string(resource.Status.OsokStatus.Ocid), testAuditArchiveRetrievalID)
	assertAuditArchiveRetrievalStringEqual(t, "status.lifecycleState", resource.Status.LifecycleState, "CREATING")
	assertAuditArchiveRetrievalStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	requireAuditArchiveRetrievalAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "CREATING", shared.OSOKAsyncClassPending, "wr-create-1")
}

func TestAuditArchiveRetrievalBindsExistingMatchAcrossListPages(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.auditarchiveretrieval.oc1..existing"
	resource := newAuditArchiveRetrievalTestResource()
	listCalls := 0
	createCalled := false
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Create.Call = func(context.Context, datasafesdk.CreateAuditArchiveRetrievalRequest) (datasafesdk.CreateAuditArchiveRetrievalResponse, error) {
			createCalled = true
			t.Fatal("CreateAuditArchiveRetrieval() should not be called when list returns a reusable match")
			return datasafesdk.CreateAuditArchiveRetrievalResponse{}, nil
		}
		hooks.List.Call = func(_ context.Context, request datasafesdk.ListAuditArchiveRetrievalsRequest) (datasafesdk.ListAuditArchiveRetrievalsResponse, error) {
			listCalls++
			requireAuditArchiveRetrievalStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireAuditArchiveRetrievalStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			requireAuditArchiveRetrievalStringPtr(t, "list targetId", request.TargetId, resource.Spec.TargetId)
			switch listCalls {
			case 1:
				if request.Page != nil {
					t.Fatalf("first list page = %q, want nil", *request.Page)
				}
				otherSpec := resource.Spec
				otherSpec.TargetId = "ocid1.datasafetarget.oc1..other"
				return datasafesdk.ListAuditArchiveRetrievalsResponse{
					AuditArchiveRetrievalCollection: datasafesdk.AuditArchiveRetrievalCollection{
						Items: []datasafesdk.AuditArchiveRetrievalSummary{
							observedAuditArchiveRetrievalSummaryFromSpec(t, "ocid1.auditarchiveretrieval.oc1..other", otherSpec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireAuditArchiveRetrievalStringPtr(t, "second list page", request.Page, "page-2")
				return datasafesdk.ListAuditArchiveRetrievalsResponse{
					AuditArchiveRetrievalCollection: datasafesdk.AuditArchiveRetrievalCollection{
						Items: []datasafesdk.AuditArchiveRetrievalSummary{
							observedAuditArchiveRetrievalSummaryFromSpec(t, existingID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected ListAuditArchiveRetrievals() call %d", listCalls)
				return datasafesdk.ListAuditArchiveRetrievalsResponse{}, nil
			}
		}
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			requireAuditArchiveRetrievalStringPtr(t, "get auditArchiveRetrievalId", request.AuditArchiveRetrievalId, existingID)
			return datasafesdk.GetAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, existingID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive),
			}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after binding an ACTIVE AuditArchiveRetrieval")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after binding an ACTIVE AuditArchiveRetrieval")
	}
	if createCalled {
		t.Fatal("CreateAuditArchiveRetrieval() was called unexpectedly")
	}
	if listCalls != 2 {
		t.Fatalf("ListAuditArchiveRetrievals() calls = %d, want 2", listCalls)
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.status.ocid", string(resource.Status.OsokStatus.Ocid), existingID)
}

func TestAuditArchiveRetrievalNoopObserveDoesNotUpdateWhenSpecMatches(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	seedAuditArchiveRetrievalStatus(resource, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive)
	updateCalled := false
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			requireAuditArchiveRetrievalStringPtr(t, "get auditArchiveRetrievalId", request.AuditArchiveRetrievalId, testAuditArchiveRetrievalID)
			return datasafesdk.GetAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive),
			}, nil
		}
		hooks.Update.Call = func(context.Context, datasafesdk.UpdateAuditArchiveRetrievalRequest) (datasafesdk.UpdateAuditArchiveRetrievalResponse, error) {
			updateCalled = true
			t.Fatal("UpdateAuditArchiveRetrieval() should not be called when desired and live state match")
			return datasafesdk.UpdateAuditArchiveRetrievalResponse{}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observe", response)
	}
	if updateCalled {
		t.Fatal("UpdateAuditArchiveRetrieval() was called unexpectedly")
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.status.ocid", string(resource.Status.OsokStatus.Ocid), testAuditArchiveRetrievalID)
}

func TestAuditArchiveRetrievalMutableUpdateRefreshesObservedState(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	previousSpec := resource.Spec
	previousSpec.DisplayName = "previous-retrieval"
	previousSpec.Description = "old description"
	previousSpec.FreeformTags = map[string]string{"team": "legacy"}
	previousSpec.DefinedTags = map[string]shared.MapValue{
		"Operations": {
			"CostCenter": "7",
		},
	}
	seedAuditArchiveRetrievalStatus(resource, testAuditArchiveRetrievalID, previousSpec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive)

	getCalls := 0
	var updateRequest datasafesdk.UpdateAuditArchiveRetrievalRequest
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			getCalls++
			requireAuditArchiveRetrievalStringPtr(t, "get auditArchiveRetrievalId", request.AuditArchiveRetrievalId, testAuditArchiveRetrievalID)
			spec := previousSpec
			if getCalls > 1 {
				spec = resource.Spec
			}
			return datasafesdk.GetAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, testAuditArchiveRetrievalID, spec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive),
			}, nil
		}
		hooks.Update.Call = func(_ context.Context, request datasafesdk.UpdateAuditArchiveRetrievalRequest) (datasafesdk.UpdateAuditArchiveRetrievalResponse, error) {
			updateRequest = request
			return datasafesdk.UpdateAuditArchiveRetrievalResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue update after ACTIVE follow-up", response)
	}
	requireAuditArchiveRetrievalStringPtr(t, "update auditArchiveRetrievalId", updateRequest.AuditArchiveRetrievalId, testAuditArchiveRetrievalID)
	requireAuditArchiveRetrievalStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	requireAuditArchiveRetrievalStringPtr(t, "update description", updateRequest.Description, resource.Spec.Description)
	if got := updateRequest.FreeformTags["team"]; got != "security" {
		t.Fatalf("update freeformTags[team] = %q, want security", got)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 42", got)
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.displayName", resource.Status.DisplayName, resource.Spec.DisplayName)
	assertAuditArchiveRetrievalStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
}

func TestAuditArchiveRetrievalRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	previousSpec := resource.Spec
	previousSpec.TargetId = "ocid1.datasafetarget.oc1..previous"
	seedAuditArchiveRetrievalStatus(resource, testAuditArchiveRetrievalID, previousSpec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive)
	updateCalled := false
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Get.Call = func(context.Context, datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			return datasafesdk.GetAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, testAuditArchiveRetrievalID, previousSpec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive),
			}, nil
		}
		hooks.Update.Call = func(context.Context, datasafesdk.UpdateAuditArchiveRetrievalRequest) (datasafesdk.UpdateAuditArchiveRetrievalResponse, error) {
			updateCalled = true
			t.Fatal("UpdateAuditArchiveRetrieval() should not be called after create-only targetId drift")
			return datasafesdk.UpdateAuditArchiveRetrievalResponse{}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "targetId") {
		t.Fatalf("CreateOrUpdate() error = %v, want targetId create-only drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	if updateCalled {
		t.Fatal("UpdateAuditArchiveRetrieval() was called unexpectedly")
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.status.reason", resource.Status.OsokStatus.Reason, string(shared.Failed))
}

func TestAuditArchiveRetrievalDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	seedAuditArchiveRetrievalStatus(resource, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive)

	getCalls := 0
	deleteCalls := 0
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Get.Call = func(_ context.Context, request datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			getCalls++
			requireAuditArchiveRetrievalStringPtr(t, "get auditArchiveRetrievalId", request.AuditArchiveRetrievalId, testAuditArchiveRetrievalID)
			state := datasafesdk.AuditArchiveRetrievalLifecycleStateActive
			if getCalls == 3 {
				state = datasafesdk.AuditArchiveRetrievalLifecycleStateDeleting
			}
			return datasafesdk.GetAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, testAuditArchiveRetrievalID, resource.Spec, state),
			}, nil
		}
		hooks.Delete.Call = func(_ context.Context, request datasafesdk.DeleteAuditArchiveRetrievalRequest) (datasafesdk.DeleteAuditArchiveRetrievalResponse, error) {
			deleteCalls++
			requireAuditArchiveRetrievalStringPtr(t, "delete auditArchiveRetrievalId", request.AuditArchiveRetrievalId, testAuditArchiveRetrievalID)
			return datasafesdk.DeleteAuditArchiveRetrievalResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle remains DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAuditArchiveRetrieval() calls = %d, want 1", deleteCalls)
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.lifecycleState", resource.Status.LifecycleState, "DELETING")
	assertAuditArchiveRetrievalStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-delete-1")
	requireAuditArchiveRetrievalAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "", shared.OSOKAsyncClassPending, "wr-delete-1")
}

func TestAuditArchiveRetrievalDeleteConfirmsDeletedWithoutCallingDelete(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	seedAuditArchiveRetrievalStatus(resource, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateDeleting)
	deleteCalled := false
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Get.Call = func(context.Context, datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			return datasafesdk.GetAuditArchiveRetrievalResponse{
				AuditArchiveRetrieval: observedAuditArchiveRetrievalFromSpec(t, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateDeleted),
			}, nil
		}
		hooks.Delete.Call = func(context.Context, datasafesdk.DeleteAuditArchiveRetrievalRequest) (datasafesdk.DeleteAuditArchiveRetrievalResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteAuditArchiveRetrieval() should not be called after DELETED readback")
			return datasafesdk.DeleteAuditArchiveRetrievalResponse{}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED readback")
	}
	if deleteCalled {
		t.Fatal("DeleteAuditArchiveRetrieval() was called unexpectedly")
	}
}

func TestAuditArchiveRetrievalDeleteRejectsAuthShapedPreReadWithoutCallingDelete(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	seedAuditArchiveRetrievalStatus(resource, testAuditArchiveRetrievalID, resource.Spec, datasafesdk.AuditArchiveRetrievalLifecycleStateActive)
	deleteCalled := false
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Get.Call = func(context.Context, datasafesdk.GetAuditArchiveRetrievalRequest) (datasafesdk.GetAuditArchiveRetrievalResponse, error) {
			return datasafesdk.GetAuditArchiveRetrievalResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		}
		hooks.Delete.Call = func(context.Context, datasafesdk.DeleteAuditArchiveRetrievalRequest) (datasafesdk.DeleteAuditArchiveRetrievalResponse, error) {
			deleteCalled = true
			t.Fatal("DeleteAuditArchiveRetrieval() should not be called after ambiguous pre-read")
			return datasafesdk.DeleteAuditArchiveRetrievalResponse{}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "refusing to call delete") {
		t.Fatalf("Delete() error = %v, want ambiguous pre-read refusal", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false after ambiguous pre-read")
	}
	if deleteCalled {
		t.Fatal("DeleteAuditArchiveRetrieval() was called unexpectedly")
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
}

func TestAuditArchiveRetrievalCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := newAuditArchiveRetrievalTestResource()
	hooks := newAuditArchiveRetrievalRuntimeTestHooksWithOperations(func(hooks *AuditArchiveRetrievalRuntimeHooks) {
		hooks.Create.Call = func(context.Context, datasafesdk.CreateAuditArchiveRetrievalRequest) (datasafesdk.CreateAuditArchiveRetrievalResponse, error) {
			return datasafesdk.CreateAuditArchiveRetrievalResponse{}, errortest.NewServiceError(500, "InternalServerError", "create failed")
		}
		hooks.List.Call = func(context.Context, datasafesdk.ListAuditArchiveRetrievalsRequest) (datasafesdk.ListAuditArchiveRetrievalsResponse, error) {
			return datasafesdk.ListAuditArchiveRetrievalsResponse{}, nil
		}
	})
	client := newAuditArchiveRetrievalRuntimeTestClient(hooks)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful OCI failure", response)
	}
	assertAuditArchiveRetrievalStringEqual(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
	assertAuditArchiveRetrievalStringEqual(t, "status.status.reason", resource.Status.OsokStatus.Reason, string(shared.Failed))
}

func newAuditArchiveRetrievalRuntimeTestHooks() AuditArchiveRetrievalRuntimeHooks {
	hooks := newAuditArchiveRetrievalDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyAuditArchiveRetrievalRuntimeHooks(&hooks)
	return hooks
}

func newAuditArchiveRetrievalRuntimeTestHooksWithOperations(
	mutate func(*AuditArchiveRetrievalRuntimeHooks),
) AuditArchiveRetrievalRuntimeHooks {
	hooks := newAuditArchiveRetrievalDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	if mutate != nil {
		mutate(&hooks)
	}
	applyAuditArchiveRetrievalRuntimeHooks(&hooks)
	return hooks
}

func newAuditArchiveRetrievalRuntimeTestClient(hooks AuditArchiveRetrievalRuntimeHooks) AuditArchiveRetrievalServiceClient {
	manager := &AuditArchiveRetrievalServiceManager{}
	delegate := defaultAuditArchiveRetrievalServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.AuditArchiveRetrieval](
			buildAuditArchiveRetrievalGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapAuditArchiveRetrievalGeneratedClient(hooks, delegate)
}

func newAuditArchiveRetrievalTestResource() *datasafev1beta1.AuditArchiveRetrieval {
	return &datasafev1beta1.AuditArchiveRetrieval{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "audit-archive-retrieval",
			Namespace: "default",
			UID:       "audit-archive-retrieval-uid",
		},
		Spec: datasafev1beta1.AuditArchiveRetrievalSpec{
			CompartmentId: testCompartmentID,
			TargetId:      testTargetID,
			StartDate:     testStartDate,
			EndDate:       testEndDate,
			DisplayName:   "audit-retrieval",
			Description:   "retrieve archived audit events",
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

func seedAuditArchiveRetrievalStatus(
	resource *datasafev1beta1.AuditArchiveRetrieval,
	id string,
	spec datasafev1beta1.AuditArchiveRetrievalSpec,
	state datasafesdk.AuditArchiveRetrievalLifecycleStateEnum,
) {
	resource.Status.Id = id
	resource.Status.CompartmentId = spec.CompartmentId
	resource.Status.TargetId = spec.TargetId
	resource.Status.StartDate = spec.StartDate
	resource.Status.EndDate = spec.EndDate
	resource.Status.DisplayName = spec.DisplayName
	resource.Status.Description = spec.Description
	resource.Status.FreeformTags = spec.FreeformTags
	resource.Status.DefinedTags = spec.DefinedTags
	resource.Status.LifecycleState = string(state)
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func observedAuditArchiveRetrievalFromSpec(
	t *testing.T,
	id string,
	spec datasafev1beta1.AuditArchiveRetrievalSpec,
	state datasafesdk.AuditArchiveRetrievalLifecycleStateEnum,
) datasafesdk.AuditArchiveRetrieval {
	t.Helper()
	return datasafesdk.AuditArchiveRetrieval{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		StartDate:      mustAuditArchiveRetrievalSDKTime(t, spec.StartDate),
		EndDate:        mustAuditArchiveRetrievalSDKTime(t, spec.EndDate),
		TargetId:       common.String(spec.TargetId),
		LifecycleState: state,
		Description:    common.String(spec.Description),
		FreeformTags:   spec.FreeformTags,
		DefinedTags:    auditArchiveRetrievalDefinedTags(spec.DefinedTags),
	}
}

func observedAuditArchiveRetrievalSummaryFromSpec(
	t *testing.T,
	id string,
	spec datasafev1beta1.AuditArchiveRetrievalSpec,
	state datasafesdk.AuditArchiveRetrievalLifecycleStateEnum,
) datasafesdk.AuditArchiveRetrievalSummary {
	t.Helper()
	return datasafesdk.AuditArchiveRetrievalSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		StartDate:      mustAuditArchiveRetrievalSDKTime(t, spec.StartDate),
		EndDate:        mustAuditArchiveRetrievalSDKTime(t, spec.EndDate),
		TargetId:       common.String(spec.TargetId),
		LifecycleState: state,
		Description:    common.String(spec.Description),
		FreeformTags:   spec.FreeformTags,
		DefinedTags:    auditArchiveRetrievalDefinedTags(spec.DefinedTags),
	}
}

func auditArchiveRetrievalDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if len(tags) == 0 {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func mustAuditArchiveRetrievalSDKTime(t *testing.T, value string) *common.SDKTime {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse SDK time %q: %v", value, err)
	}
	return &common.SDKTime{Time: parsed}
}

func assertAuditArchiveRetrievalCreateRequest(
	t *testing.T,
	request datasafesdk.CreateAuditArchiveRetrievalRequest,
	resource *datasafev1beta1.AuditArchiveRetrieval,
) {
	t.Helper()
	requireAuditArchiveRetrievalStringPtr(t, "create compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireAuditArchiveRetrievalStringPtr(t, "create targetId", request.TargetId, resource.Spec.TargetId)
	requireAuditArchiveRetrievalStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	requireAuditArchiveRetrievalStringPtr(t, "create description", request.Description, resource.Spec.Description)
	if got := request.StartDate.Format(time.RFC3339); got != resource.Spec.StartDate {
		t.Fatalf("create startDate = %q, want %q", got, resource.Spec.StartDate)
	}
	if got := request.EndDate.Format(time.RFC3339); got != resource.Spec.EndDate {
		t.Fatalf("create endDate = %q, want %q", got, resource.Spec.EndDate)
	}
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

func requireAuditArchiveRetrievalStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireAuditArchiveRetrievalAsyncCurrent(
	t *testing.T,
	resource *datasafev1beta1.AuditArchiveRetrieval,
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

func assertAuditArchiveRetrievalStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

func assertAuditArchiveRetrievalStringEqual(t *testing.T, label string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}
