/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"testing"
)

func TestServiceClientHasMutableDriftDetectsTagAddition(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Mutation: MutationSemantics{Mutable: []string{"freeformTags"}}}})
	resource := &fakeResource{Spec: fakeSpec{FreeformTags: map[string]string{"scenario": "e2e"}}}
	current := fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", LifecycleState: "ACTIVE"}}
	drift, err := client.hasMutableDrift(resource, current)
	if err != nil {
		t.Fatalf("hasMutableDrift() error = %v", err)
	}
	if !drift {
		t.Fatal("hasMutableDrift() = false, want true when spec adds a mutable tag")
	}
}

func TestServiceClientHasMutableDriftIgnoresIdenticalTags(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Mutation: MutationSemantics{Mutable: []string{"freeformTags"}}}})
	resource := &fakeResource{Spec: fakeSpec{FreeformTags: map[string]string{"scenario": "e2e"}}}
	current := fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", FreeformTags: map[string]string{"scenario": "e2e"}, LifecycleState: "ACTIVE"}}
	drift, err := client.hasMutableDrift(resource, current)
	if err != nil {
		t.Fatalf("hasMutableDrift() error = %v", err)
	}
	if drift {
		t.Fatal("hasMutableDrift() = true, want false when mutable tags already match")
	}
}

func TestServiceClientCreateOrUpdateUpdatesExistingResource(t *testing.T) {
	t.Parallel()
	var updateRequest fakeUpdateThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		updateRequest = *request.(*fakeUpdateThingRequest)
		return fakeUpdateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "updated-name", LifecycleState: "UPDATING"}}, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "updated-name", LifecycleState: "UPDATING"}}, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{DisplayName: "updated-name", Enabled: true}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while update is in progress")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle is UPDATING")
	}
	if updateRequest.ThingId == nil || *updateRequest.ThingId != "ocid1.thing.oc1..existing" {
		t.Fatalf("update request thingId = %v, want existing OCID", updateRequest.ThingId)
	}
	if updateRequest.DisplayName != "updated-name" {
		t.Fatalf("update request displayName = %q, want updated-name", updateRequest.DisplayName)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Updating {
		t.Fatalf("status conditions = %#v, want trailing Updating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	t.Parallel()
	getCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
			t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "steady-name", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called when mutable fields already match")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{DisplayName: "steady-name"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when mutable fields already match")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if !getCalled {
		t.Fatal("Get() should be called to compare the current mutable state")
	}
	if resource.Status.DisplayName != "steady-name" {
		t.Fatalf("status.displayName = %q, want steady-name", resource.Status.DisplayName)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Active {
		t.Fatalf("status conditions = %#v, want trailing Active condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientCreateOrUpdateBuildsMinimalUpdateBodyFromChangedMutableFields(t *testing.T) {
	t.Parallel()
	var updateRequest fakeUpdateThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"displayName", "freeformTags"}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		updateRequest = *request.(*fakeUpdateThingRequest)
		return fakeUpdateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "steady-name", FreeformTags: map[string]string{"scenario": "update", "run": "123"}, LifecycleState: "ACTIVE"}}, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "steady-name", FreeformTags: map[string]string{"scenario": "create", "run": "123"}, ShapeConfig: &fakeShapeConfig{Ocpus: 1, MemoryInGBs: 16, Vcpus: 2}, LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Spec: fakeSpec{DisplayName: "steady-name", FreeformTags: map[string]string{"scenario": "update", "run": "123"}, ShapeConfig: &fakeShapeConfig{Ocpus: 1, MemoryInGBs: 16}}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing"}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if updateRequest.ThingId == nil || *updateRequest.ThingId != "ocid1.thing.oc1..existing" {
		t.Fatalf("update request thingId = %v, want existing OCID", updateRequest.ThingId)
	}
	if updateRequest.DisplayName != "" {
		t.Fatalf("update request displayName = %q, want unchanged field omitted", updateRequest.DisplayName)
	}
	if updateRequest.ShapeConfig != nil {
		t.Fatalf("update request shapeConfig = %#v, want non-mutable field omitted", updateRequest.ShapeConfig)
	}
	if got := updateRequest.FreeformTags; !valuesEqual(got, map[string]string{"scenario": "update", "run": "123"}) {
		t.Fatalf("update request freeformTags = %#v, want only changed mutable tags", got)
	}
}

func TestServiceClientFilteredUpdateBodyOmitsDocsDeniedNameChange(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Bucket", SDKName: "Bucket", Semantics: &Semantics{Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}}})
	resource := &fakeResource{Spec: fakeSpec{Name: "bucket-new", DisplayName: "display-new"}}
	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{CurrentResponse: fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.bucket.oc1..existing", Name: "bucket-old", DisplayName: "display-old", LifecycleState: "ACTIVE"}}})
	if err != nil {
		t.Fatalf("filteredUpdateBody() error = %v", err)
	}
	if !ok {
		t.Fatal("filteredUpdateBody() = false, want mutable displayName change to produce an update body")
	}
	bodyMap, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("filteredUpdateBody() body = %T, want map[string]any", body)
	}
	if _, found := bodyMap["name"]; found {
		t.Fatalf("filteredUpdateBody() body = %#v, want docs-denied name omitted", bodyMap)
	}
	if got, found := bodyMap["displayName"]; !found || got != "display-new" {
		t.Fatalf("filteredUpdateBody() body = %#v, want displayName only", bodyMap)
	}
}

func TestServiceClientCreateOrUpdateSkipsUpdateWhileLifecycleProvisioning(t *testing.T) {
	t.Parallel()
	getCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
			t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "creating-name", LifecycleState: "CREATING"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called while lifecycle is still provisioning")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{DisplayName: "desired-name"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed while observing provisioning lifecycle")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle is provisioning")
	}
	if !getCalled {
		t.Fatal("Get() should be called to observe current provisioning lifecycle")
	}
	if resource.Status.LifecycleState != "CREATING" {
		t.Fatalf("status.lifecycleState = %q, want CREATING", resource.Status.LifecycleState)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Provisioning {
		t.Fatalf("status conditions = %#v, want trailing Provisioning condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientCreateOrUpdateSkipsUpdateWhenMutableFieldIsNotReturnedByService(t *testing.T) {
	t.Parallel()
	getCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"adminPassword"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
			t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "steady-name", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called when a mutable field is write-only in the service response")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-password"}}}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when write-only mutable fields are not returned")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if !getCalled {
		t.Fatal("Get() should be called to observe the current resource")
	}
}

func TestServiceClientCreateOrUpdateUpdatesWhenMutableStateDiffers(t *testing.T) {
	t.Parallel()
	getCalled := false
	var updateRequest fakeUpdateThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
			t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "old-name", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		updateRequest = *request.(*fakeUpdateThingRequest)
		return fakeUpdateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DisplayName: "new-name", LifecycleState: "ACTIVE"}}, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{DisplayName: "new-name"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, false, "CreateOrUpdate() should not requeue for ACTIVE update result")
	requireTrue(t, getCalled, "Get() should be called before comparing mutable fields")
	requireThingIDRequest(t, "update", updateRequest.ThingId, "ocid1.thing.oc1..existing")
	requireStringEqual(t, "update request displayName", updateRequest.DisplayName, "new-name")
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "new-name")
}

func TestServiceClientCreateOrUpdateAllowsMutableDriftThroughInGBsAlias(t *testing.T) {
	t.Parallel()
	updateCalled := false
	var updateRequest fakeUpdateThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"dataStorageSizeInGb"}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		updateCalled = true
		updateRequest = *request.(*fakeUpdateThingRequest)
		return fakeUpdateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", DataStorageSizeInGBs: updateRequest.DataStorageSizeInGBs, LifecycleState: "ACTIVE"}}, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{DataStorageSizeInGBs: 20}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing", DataStorageSizeInGBs: 10}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, false, "CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	if !updateCalled {
		t.Fatal("Update() should be called when only the InGBs alias differs")
	}
	if updateRequest.DataStorageSizeInGBs != 20 {
		t.Fatalf("update request dataStorageSizeInGBs = %d, want 20", updateRequest.DataStorageSizeInGBs)
	}
	if resource.Status.DataStorageSizeInGBs != 20 {
		t.Fatalf("status.dataStorageSizeInGBs = %d, want 20", resource.Status.DataStorageSizeInGBs)
	}
}

func TestServiceClientRejectsUnsupportedUpdateDrift(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called when update drift is outside the supported mutable surface")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..new", DisplayName: "same-name"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing", CompartmentId: "ocid1.compartment.oc1..old", DisplayName: "same-name"}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "reject unsupported update drift for compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported update drift failure", err)
	}
}

func TestServiceClientRejectsForceNewMutationChanges(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Mutation: MutationSemantics{ForceNew: []string{"compartmentId"}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		t.Fatal("Update() should not be called when a force-new field changes")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..new", DisplayName: "updated-name"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing", CompartmentId: "ocid1.compartment.oc1..old"}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new replacement failure", err)
	}
}

func TestServiceClientRejectsConflictingMutationFields(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Mutation: MutationSemantics{ConflictsWith: map[string][]string{"name": {"displayName"}}}}})
	resource := &fakeResource{Spec: fakeSpec{Name: "wanted", DisplayName: "conflicting"}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "forbid setting name with displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want conflictsWith failure", err)
	}
}

func TestServiceClientCreateOrUpdateRejectsDocsDeniedNameDrift(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Bucket", SDKName: "Bucket", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.bucket.oc1..existing" {
			t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.bucket.oc1..existing", Name: "bucket-old", DisplayName: "display-old", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called when Bucket.name drift is outside the conservative mutable surface")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{Name: "bucket-new", DisplayName: "display-old"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.bucket.oc1..existing"}, Id: "ocid1.bucket.oc1..existing"}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "reject unsupported update drift for name") {
		t.Fatalf("CreateOrUpdate() error = %v, want docs-denied name drift failure", err)
	}
}

func TestServiceClientCreateOrUpdateForcesLiveGetForForceNewValidationAfterListReuse(t *testing.T) {
	t.Parallel()
	createCalled := false
	var getRequest fakeGetThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, Mutation: MutationSemantics{ForceNew: []string{"retentionInHours"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		createCalled = true
		t.Fatal("Create() should not be called when live force-new validation fails")
		return nil, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getRequest = *request.(*fakeGetThingRequest)
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", RetentionInHours: 24, LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..existing", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted", RetentionInHours: 48}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "require replacement when retentionInHours changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want live force-new replacement failure", err)
	}
	if createCalled {
		t.Fatal("Create() should not be called when live force-new validation fails")
	}
	if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..existing" {
		t.Fatalf("get request thingId = %v, want reused OCID", getRequest.ThingId)
	}
	if resource.Status.RetentionInHours != 24 {
		t.Fatalf("status.retentionInHours = %d, want 24 from live Get", resource.Status.RetentionInHours)
	}
}

func TestServiceClientCreateOrUpdateIgnoresProviderManagedNestedForceNewFields(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{ForceNew: []string{"shapeConfig"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getRequest := *request.(*fakeGetThingRequest)
		if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..existing" {
			t.Fatalf("get request thingId = %v, want reused OCID", getRequest.ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", ShapeConfig: &fakeShapeConfig{Ocpus: 1, MemoryInGBs: 16, Vcpus: 2}, LifecycleState: "CREATING"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Spec: fakeSpec{ShapeConfig: &fakeShapeConfig{Ocpus: 1, MemoryInGBs: 16}}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want nil", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while live resource is provisioning", response)
	}
	if resource.Status.ShapeConfig == nil || resource.Status.ShapeConfig.Vcpus != 2 {
		t.Fatalf("status.shapeConfig = %#v, want live provider-managed fields merged", resource.Status.ShapeConfig)
	}
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 || conditions[len(conditions)-1].Type != shared.Provisioning {
		t.Fatalf("status conditions = %#v, want trailing Provisioning condition", conditions)
	}
}

func TestForceNewValuesEqualIgnoresMeaninglessNestedMaps(t *testing.T) {
	t.Parallel()
	spec := map[string]any{"imageId": "ocid1.image.oc1..example", "sourceType": "image", "instanceSourceImageFilterDetails": map[string]any{"compartmentId": ""}}
	current := map[string]any{"imageId": "ocid1.image.oc1..example", "sourceType": "image", "instanceSourceImageFilterDetails": nil}
	if !forceNewValuesEqual(spec, current) {
		t.Fatalf("forceNewValuesEqual() = false, want true when only meaningless nested maps differ")
	}
}

func TestUnsupportedUpdateDriftPathsIgnoresMeaninglessNestedMaps(t *testing.T) {
	t.Parallel()
	spec := map[string]any{"displayName": "example", "preemptibleInstanceConfig": map[string]any{"preemptionAction": map[string]any{"jsonData": "", "type": ""}}}
	current := map[string]any{"displayName": "example"}
	paths := unsupportedUpdateDriftPaths(spec, current, MutationSemantics{Mutable: []string{"displayName"}})
	if len(paths) != 0 {
		t.Fatalf("unsupportedUpdateDriftPaths() = %v, want no drift for meaningless nested maps", paths)
	}
}

func TestServiceClientRejectsForceNewChangesAgainstLiveOCIState(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Mutation: MutationSemantics{ForceNew: []string{"compartmentId"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", CompartmentId: "ocid1.compartment.oc1..live", DisplayName: "updated-name", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called when live force-new validation fails")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..desired", DisplayName: "updated-name"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"}, Id: "ocid1.thing.oc1..existing", CompartmentId: "ocid1.compartment.oc1..desired"}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want live force-new replacement failure", err)
	}
}
