/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odainstance

import (
	"context"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestOdaInstanceRuntimeSemanticsEncodesModernLifecycleContract(t *testing.T) {
	t.Parallel()

	semantics := newOdaInstanceRuntimeSemantics()
	if semantics == nil {
		t.Fatal("newOdaInstanceRuntimeSemantics() = nil")
	}
	if semantics.FormalService != "oda" {
		t.Fatalf("FormalService = %q, want oda", semantics.FormalService)
	}
	if semantics.FormalSlug != "odainstance" {
		t.Fatalf("FormalSlug = %q, want odainstance", semantics.FormalSlug)
	}
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", semantics.FinalizerPolicy)
	}
	if semantics.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", semantics.Delete.Policy)
	}
	if semantics.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", semantics.CreateFollowUp.Strategy)
	}
	if semantics.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", semantics.UpdateFollowUp.Strategy)
	}
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", semantics.DeleteFollowUp.Strategy)
	}

	assertOdaInstanceStringSliceEqual(t, "Lifecycle.ProvisioningStates", semantics.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertOdaInstanceStringSliceEqual(t, "Lifecycle.UpdatingStates", semantics.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertOdaInstanceStringSliceEqual(t, "Lifecycle.ActiveStates", semantics.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertOdaInstanceStringSliceEqual(t, "Delete.PendingStates", semantics.Delete.PendingStates, []string{"DELETING"})
	assertOdaInstanceStringSliceEqual(t, "Delete.TerminalStates", semantics.Delete.TerminalStates, []string{"DELETED"})
	assertOdaInstanceStringSliceEqual(t, "List.MatchFields", semantics.List.MatchFields, []string{"compartmentId", "displayName"})
	assertOdaInstanceStringSliceEqual(t, "Mutation.Mutable", semantics.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags"})
	assertOdaInstanceStringSliceEqual(t, "Mutation.ForceNew", semantics.Mutation.ForceNew, []string{"compartmentId", "shapeName", "isRoleBasedAccess", "identityDomain"})

	hooks := newOdaInstanceDefaultRuntimeHooks(odasdk.OdaClient{})
	applyOdaInstanceRuntimeHooks(&hooks)
	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed OdaInstance semantics")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want safe bind guard")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update body builder")
	}
}

func TestOdaInstanceCreateProjectsPendingLifecycleAndStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.odainstance.oc1..created"
	resource := newOdaInstanceTestResource()
	hooks := newOdaInstanceRuntimeTestHooks()

	var createRequest odasdk.CreateOdaInstanceRequest
	getCalls := 0
	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.CreateOdaInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*odasdk.CreateOdaInstanceRequest)
				return odasdk.CreateOdaInstanceResponse{
					OdaInstance:      observedOdaInstanceFromSpec(createdID, resource.Spec, odasdk.OdaInstanceLifecycleStateCreating),
					OpcRequestId:     common.String("opc-create-1"),
					OpcWorkRequestId: common.String("wr-create-1"),
				}, nil
			},
			Fields: hooks.Create.Fields,
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireOdaInstanceStringPtr(t, "get odaInstanceId", request.(*odasdk.GetOdaInstanceRequest).OdaInstanceId, createdID)
				return odasdk.GetOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(createdID, resource.Spec, odasdk.OdaInstanceLifecycleStateCreating),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while OdaInstance is creating")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while OdaInstance is creating")
	}
	requireOdaInstanceStringPtr(t, "create compartmentId", createRequest.CreateOdaInstanceDetails.CompartmentId, resource.Spec.CompartmentId)
	if createRequest.CreateOdaInstanceDetails.ShapeName != odasdk.CreateOdaInstanceDetailsShapeNameEnum(resource.Spec.ShapeName) {
		t.Fatalf("create shapeName = %q, want %q", createRequest.CreateOdaInstanceDetails.ShapeName, resource.Spec.ShapeName)
	}
	if getCalls != 1 {
		t.Fatalf("GetOdaInstance() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if resource.Status.LifecycleState != string(odasdk.OdaInstanceLifecycleStateCreating) {
		t.Fatalf("status.lifecycleState = %q, want CREATING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", resource.Status.OsokStatus.OpcRequestID)
	}
	requireOdaInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "CREATING", shared.OSOKAsyncClassPending, "wr-create-1")
}

func TestOdaInstanceBindsExistingDisplayNameMatchWithoutCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.odainstance.oc1..existing"
	resource := newOdaInstanceTestResource()
	hooks := newOdaInstanceRuntimeTestHooks()

	createCalled := false
	getCalls := 0
	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.CreateOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				createCalled = true
				t.Fatal("CreateOdaInstance() should not be called when list returns a reusable match")
				return nil, nil
			},
			Fields: hooks.Create.Fields,
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireOdaInstanceStringPtr(t, "get odaInstanceId", request.(*odasdk.GetOdaInstanceRequest).OdaInstanceId, existingID)
				return odasdk.GetOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(existingID, resource.Spec, odasdk.OdaInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.ListOdaInstancesRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest := request.(*odasdk.ListOdaInstancesRequest)
				requireOdaInstanceStringPtr(t, "list compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
				requireOdaInstanceStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
				return odasdk.ListOdaInstancesResponse{
					Items: []odasdk.OdaInstanceSummary{
						observedOdaInstanceSummaryFromSpec(existingID, resource.Spec, odasdk.OdaInstanceLifecycleStateActive),
					},
				}, nil
			},
			Fields: hooks.List.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after binding an ACTIVE OdaInstance")
	}
	if createCalled {
		t.Fatal("CreateOdaInstance() was called unexpectedly")
	}
	if getCalls != 1 {
		t.Fatalf("GetOdaInstance() calls = %d, want 1 live read after list bind", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestOdaInstanceCreateSkipsUnsafeListBindWhenDisplayNameIsEmpty(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.odainstance.oc1..created-empty-name"
	resource := newOdaInstanceTestResource()
	resource.Spec.DisplayName = ""
	hooks := newOdaInstanceRuntimeTestHooks()

	listCalled := false
	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.CreateOdaInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				if request.(*odasdk.CreateOdaInstanceRequest).CreateOdaInstanceDetails.DisplayName != nil {
					t.Fatalf("create displayName = %#v, want nil for empty displayName", request.(*odasdk.CreateOdaInstanceRequest).CreateOdaInstanceDetails.DisplayName)
				}
				return odasdk.CreateOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(createdID, resource.Spec, odasdk.OdaInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Create.Fields,
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return odasdk.GetOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(createdID, resource.Spec, odasdk.OdaInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.ListOdaInstancesRequest{} },
			Call: func(context.Context, any) (any, error) {
				listCalled = true
				t.Fatal("ListOdaInstances() should be skipped without displayName match criteria")
				return nil, nil
			},
			Fields: hooks.List.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful steady create", response)
	}
	if listCalled {
		t.Fatal("ListOdaInstances() was called unexpectedly")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
}

func TestOdaInstanceUpdateProjectsOnlySupportedMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.odainstance.oc1..update"
	resource := newExistingOdaInstanceTestResource(existingID)
	resource.Spec.Description = "updated ODA description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod", "owner": "osok"}
	hooks := newOdaInstanceRuntimeTestHooks()

	getCalls := 0
	var updateRequest odasdk.UpdateOdaInstanceRequest
	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireOdaInstanceStringPtr(t, "get odaInstanceId", request.(*odasdk.GetOdaInstanceRequest).OdaInstanceId, existingID)
				spec := newOdaInstanceTestResource().Spec
				lifecycle := odasdk.OdaInstanceLifecycleStateActive
				if getCalls > 1 {
					spec = resource.Spec
					lifecycle = odasdk.OdaInstanceLifecycleStateUpdating
				}
				return odasdk.GetOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(existingID, spec, lifecycle),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.UpdateOdaInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*odasdk.UpdateOdaInstanceRequest)
				return odasdk.UpdateOdaInstanceResponse{
					OdaInstance:  observedOdaInstanceFromSpec(existingID, resource.Spec, odasdk.OdaInstanceLifecycleStateUpdating),
					OpcRequestId: common.String("opc-update-1"),
				}, nil
			},
			Fields: hooks.Update.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while OdaInstance is updating")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while OdaInstance is updating")
	}
	requireOdaInstanceStringPtr(t, "update odaInstanceId", updateRequest.OdaInstanceId, existingID)
	if updateRequest.UpdateOdaInstanceDetails.Description == nil || *updateRequest.UpdateOdaInstanceDetails.Description != resource.Spec.Description {
		t.Fatalf("update description = %#v, want %q", updateRequest.UpdateOdaInstanceDetails.Description, resource.Spec.Description)
	}
	if updateRequest.UpdateOdaInstanceDetails.DisplayName != nil {
		t.Fatalf("update displayName = %#v, want nil for unchanged displayName", updateRequest.UpdateOdaInstanceDetails.DisplayName)
	}
	if !maps.Equal(updateRequest.UpdateOdaInstanceDetails.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", updateRequest.UpdateOdaInstanceDetails.FreeformTags, resource.Spec.FreeformTags)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
	requireOdaInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "UPDATING", shared.OSOKAsyncClassPending, "")
}

func TestOdaInstanceRejectsCreateOnlyShapeDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.odainstance.oc1..shape-drift"
	resource := newExistingOdaInstanceTestResource(existingID)
	resource.Spec.ShapeName = string(odasdk.OdaInstanceShapeNameProduction)
	hooks := newOdaInstanceRuntimeTestHooks()

	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return odasdk.GetOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(existingID, newOdaInstanceTestResource().Spec, odasdk.OdaInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.UpdateOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal("UpdateOdaInstance() should not be called for create-only shapeName drift")
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	})

	_, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when shapeName changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want shapeName replacement failure", err)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
}

func TestOdaInstanceFailedLifecycleProjectsFailureCondition(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.odainstance.oc1..failed"
	resource := newExistingOdaInstanceTestResource(existingID)
	hooks := newOdaInstanceRuntimeTestHooks()

	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				observed := observedOdaInstanceFromSpec(existingID, resource.Spec, odasdk.OdaInstanceLifecycleStateFailed)
				observed.StateMessage = common.String("ODA instance provisioning failed")
				return odasdk.GetOdaInstanceResponse{OdaInstance: observed}, nil
			},
			Fields: hooks.Get.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report unsuccessful for FAILED lifecycle")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after FAILED lifecycle")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
	if resource.Status.LifecycleState != string(odasdk.OdaInstanceLifecycleStateFailed) {
		t.Fatalf("status.lifecycleState = %q, want FAILED", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for observed FAILED steady-state read", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOdaInstanceDeleteWaitsForLifecycleConfirmation(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.odainstance.oc1..delete"
	resource := newExistingOdaInstanceTestResource(existingID)
	hooks := newOdaInstanceRuntimeTestHooks()

	getCalls := 0
	var deleteRequest odasdk.DeleteOdaInstanceRequest
	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				getCalls++
				lifecycle := odasdk.OdaInstanceLifecycleStateActive
				if getCalls > 1 {
					lifecycle = odasdk.OdaInstanceLifecycleStateDeleting
				}
				return odasdk.GetOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(existingID, resource.Spec, lifecycle),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.DeleteOdaInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*odasdk.DeleteOdaInstanceRequest)
				return odasdk.DeleteOdaInstanceResponse{
					OpcRequestId:     common.String("opc-delete-1"),
					OpcWorkRequestId: common.String("wr-delete-1"),
				}, nil
			},
			Fields: hooks.Delete.Fields,
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI delete is pending")
	}
	requireOdaInstanceStringPtr(t, "delete odaInstanceId", deleteRequest.OdaInstanceId, existingID)
	if deleteRequest.RetentionTime != nil {
		t.Fatalf("delete retentionTime = %#v, want nil because CRD does not expose it", deleteRequest.RetentionTime)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	if resource.Status.LifecycleState != string(odasdk.OdaInstanceLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", resource.Status.OsokStatus.OpcRequestID)
	}
	requireOdaInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "", shared.OSOKAsyncClassPending, "wr-delete-1")
}

func TestOdaInstanceDeleteReleasesFinalizerAfterTerminalReadback(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.odainstance.oc1..deleted"
	resource := newExistingOdaInstanceTestResource(existingID)
	hooks := newOdaInstanceRuntimeTestHooks()

	deleteCalled := false
	manager := newOdaInstanceRuntimeTestManager(generatedruntime.Config[*odav1beta1.OdaInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.GetOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return odasdk.GetOdaInstanceResponse{
					OdaInstance: observedOdaInstanceFromSpec(existingID, resource.Spec, odasdk.OdaInstanceLifecycleStateDeleted),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &odasdk.DeleteOdaInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				deleteCalled = true
				t.Fatal("DeleteOdaInstance() should not be called after terminal readback")
				return nil, nil
			},
			Fields: hooks.Delete.Fields,
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after DELETED readback")
	}
	if deleteCalled {
		t.Fatal("DeleteOdaInstance() was called unexpectedly")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

func newOdaInstanceRuntimeTestHooks() OdaInstanceRuntimeHooks {
	hooks := newOdaInstanceDefaultRuntimeHooks(odasdk.OdaClient{})
	applyOdaInstanceRuntimeHooks(&hooks)
	return hooks
}

func newOdaInstanceRuntimeTestManager(
	cfg generatedruntime.Config[*odav1beta1.OdaInstance],
) *OdaInstanceServiceManager {
	hooks := newOdaInstanceRuntimeTestHooks()
	if cfg.Kind == "" {
		cfg.Kind = "OdaInstance"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "OdaInstance"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = hooks.Semantics
	}
	if cfg.Identity.GuardExistingBeforeCreate == nil {
		cfg.Identity.GuardExistingBeforeCreate = hooks.Identity.GuardExistingBeforeCreate
	}
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = hooks.BuildUpdateBody
	}

	return &OdaInstanceServiceManager{
		client: defaultOdaInstanceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.OdaInstance](cfg),
		},
	}
}

func newOdaInstanceTestResource() *odav1beta1.OdaInstance {
	return &odav1beta1.OdaInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oda-sample",
			Namespace: "default",
		},
		Spec: odav1beta1.OdaInstanceSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			ShapeName:     string(odasdk.OdaInstanceShapeNameDevelopment),
			DisplayName:   "oda-sample",
			Description:   "ODA description",
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func newExistingOdaInstanceTestResource(existingID string) *odav1beta1.OdaInstance {
	resource := newOdaInstanceTestResource()
	resource.Status = odav1beta1.OdaInstanceStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedOdaInstanceFromSpec(
	id string,
	spec odav1beta1.OdaInstanceSpec,
	lifecycleState odasdk.OdaInstanceLifecycleStateEnum,
) odasdk.OdaInstance {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return odasdk.OdaInstance{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		ShapeName:         odasdk.OdaInstanceShapeNameEnum(spec.ShapeName),
		DisplayName:       pointerOrNil(spec.DisplayName),
		Description:       pointerOrNil(spec.Description),
		TimeCreated:       now,
		TimeUpdated:       now,
		LifecycleState:    lifecycleState,
		StateMessage:      common.String(string(lifecycleState)),
		FreeformTags:      cloneStringMap(spec.FreeformTags),
		DefinedTags:       odaInstanceDefinedTagsFromSpec(spec.DefinedTags),
		IsRoleBasedAccess: common.Bool(spec.IsRoleBasedAccess),
		IdentityDomain:    pointerOrNil(spec.IdentityDomain),
		WebAppUrl:         common.String("https://oda.example.com"),
		ConnectorUrl:      common.String("https://connector.example.com"),
	}
}

func observedOdaInstanceSummaryFromSpec(
	id string,
	spec odav1beta1.OdaInstanceSpec,
	lifecycleState odasdk.OdaInstanceLifecycleStateEnum,
) odasdk.OdaInstanceSummary {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return odasdk.OdaInstanceSummary{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		ShapeName:         odasdk.OdaInstanceSummaryShapeNameEnum(spec.ShapeName),
		DisplayName:       pointerOrNil(spec.DisplayName),
		Description:       pointerOrNil(spec.Description),
		TimeCreated:       now,
		TimeUpdated:       now,
		LifecycleState:    odasdk.OdaInstanceSummaryLifecycleStateEnum(lifecycleState),
		StateMessage:      common.String(string(lifecycleState)),
		FreeformTags:      cloneStringMap(spec.FreeformTags),
		DefinedTags:       odaInstanceDefinedTagsFromSpec(spec.DefinedTags),
		IsRoleBasedAccess: common.Bool(spec.IsRoleBasedAccess),
		IdentityDomain:    pointerOrNil(spec.IdentityDomain),
	}
}

func requireOdaInstanceStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func requireOdaInstanceAsyncCurrent(
	t *testing.T,
	resource *odav1beta1.OdaInstance,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want lifecycle tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func assertOdaInstanceStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", label, got, want)
		}
	}
}

func pointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	return maps.Clone(values)
}
