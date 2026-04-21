/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"testing"
)

func TestServiceClientCreateOrUpdateFallsBackToList(t *testing.T) {
	t.Parallel()
	var listRequest fakeListThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		listRequest = *request.(*fakeListThingRequest)
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..other", DisplayName: "other", LifecycleState: "ACTIVE"}, {Id: "ocid1.thing.oc1..match", DisplayName: "wanted", LifecycleState: "ACTIVE"}}}}, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{DisplayName: "wanted"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list finds a matching resource")
	}
	if listRequest.DisplayName != "wanted" {
		t.Fatalf("list request displayName = %q, want wanted", listRequest.DisplayName)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..match" {
		t.Fatalf("status.ocid = %q, want matched OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCreateOrUpdateUsesFormalListMatching(t *testing.T) {
	t.Parallel()
	var listRequest fakeListThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Resources", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		listRequest = *request.(*fakeListThingRequest)
		return fakeNamedListThingResponse{Collection: fakeNamedThingCollection{Resources: []fakeThingSummary{{Id: "ocid1.thing.oc1..other", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..other", LifecycleState: "ACTIVE"}, {Id: "ocid1.thing.oc1..match", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if listRequest.CompartmentId != "ocid1.compartment.oc1..match" {
		t.Fatalf("list request compartmentId = %q, want ocid1.compartment.oc1..match", listRequest.CompartmentId)
	}
	if listRequest.Name != "wanted" {
		t.Fatalf("list request name = %q, want wanted", listRequest.Name)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..match" {
		t.Fatalf("status.ocid = %q, want matched OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCurrentIDIgnoresSpecCompartmentReference(t *testing.T) {
	t.Parallel()
	createCalled := false
	getCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Compartment", SDKName: "Compartment", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{ForceNew: []string{"compartmentId"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createCalled = true
		createRequest := request.(*fakeCreateThingRequest)
		if createRequest.CompartmentId != "ocid1.compartment.oc1..parent" {
			t.Fatalf("create request compartmentId = %q, want parent compartment ID", createRequest.CompartmentId)
		}
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.compartment.oc1..child", CompartmentId: "ocid1.compartment.oc1..parent", DisplayName: "created-name", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "FakeCreateThingDetails", Contribution: "body"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		t.Fatalf("Get() should not be called before a tracked OCID exists, got request=%+v", request)
		return nil, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..parent", DisplayName: "created-name"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if !createCalled {
		t.Fatal("Create() should be called when only a parent compartment reference exists in spec")
	}
	if getCalled {
		t.Fatal("Get() should not be called before the created child OCID is tracked")
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.compartment.oc1..child" {
		t.Fatalf("status.ocid = %q, want created child OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCompartmentCreateIgnoresParentListItem(t *testing.T) {
	t.Parallel()
	createCalled := false
	getCalled := false
	var listRequest fakeListThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Compartment", SDKName: "Compartment", Semantics: &Semantics{Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}}, Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, List: &ListSemantics{ResponseItemsField: "Resources", MatchFields: []string{"compartmentId", "lifecycleState", "name"}}, Mutation: MutationSemantics{Mutable: []string{"description", "name"}, ForceNew: []string{"compartmentId"}}, CreateFollowUp: FollowUpSemantics{Strategy: "read-after-write"}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createCalled = true
		createRequest := request.(*fakeCreateThingRequest)
		if createRequest.CompartmentId != "ocid1.compartment.oc1..parent" {
			t.Fatalf("create request compartmentId = %q, want parent compartment ID", createRequest.CompartmentId)
		}
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.compartment.oc1..child", CompartmentId: "ocid1.compartment.oc1..parent", Name: "codex-identity-compartment-20260403083600", DisplayName: "codex-identity-compartment-20260403083600", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "FakeCreateThingDetails", Contribution: "body"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if !createCalled {
			t.Fatalf("Get() should not be called before Create(), got request=%+v", request)
		}
		getRequest := request.(*fakeGetThingRequest)
		if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.compartment.oc1..child" {
			t.Fatalf("Get() should only follow the created child OCID, got request=%+v", request)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.compartment.oc1..child", CompartmentId: "ocid1.compartment.oc1..parent", Name: "codex-identity-compartment-20260403083600", DisplayName: "codex-identity-compartment-20260403083600", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		listRequest = *request.(*fakeListThingRequest)
		return fakeNamedListThingResponse{Collection: fakeNamedThingCollection{Resources: []fakeThingSummary{{Id: "ocid1.compartment.oc1..parent", CompartmentId: "ocid1.tenancy.oc1..tenancy", Name: "vdittaka", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}, {FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..parent", Name: "codex-identity-compartment-20260403083600", DisplayName: "codex-identity-compartment-20260403083600"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if listRequest.CompartmentId != "ocid1.compartment.oc1..parent" {
		t.Fatalf("list request compartmentId = %q, want parent compartment ID", listRequest.CompartmentId)
	}
	if listRequest.Name != "codex-identity-compartment-20260403083600" {
		t.Fatalf("list request name = %q, want sample compartment name", listRequest.Name)
	}
	if !createCalled {
		t.Fatal("Create() should be called when the only listed compartment is the parent")
	}
	if !getCalled {
		t.Fatal("Get() should be called for create follow-up using the created child OCID")
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.compartment.oc1..child" {
		t.Fatalf("status.ocid = %q, want created child OCID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.CompartmentId != "ocid1.compartment.oc1..parent" {
		t.Fatalf("status.compartmentId = %q, want parent compartment ID", resource.Status.CompartmentId)
	}
}

func TestServiceClientCreateOrUpdateKeepsTrackedCurrentIDWhenPreCreateLookupMisses(t *testing.T) {
	t.Parallel()
	createCalled := false
	listCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Bucket", SDKName: "Bucket", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		createCalled = true
		t.Fatal("Create() should not be called when a tracked resource already exists")
		return nil, nil
	}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		listCalled = true
		listRequest := request.(*fakeListThingRequest)
		if listRequest.Name != "bucket-new" {
			t.Fatalf("list request name = %q, want desired spec name", listRequest.Name)
		}
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.bucket.oc1..other", Name: "bucket-old", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called when immutable drift is detected on a tracked resource")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "bucket-new", DisplayName: "steady-name"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.bucket.oc1..existing"}, Id: "ocid1.bucket.oc1..existing", CompartmentId: "ocid1.compartment.oc1..match", Name: "bucket-old", DisplayName: "steady-name"}}
	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "reject unsupported update drift for name") {
		t.Fatalf("CreateOrUpdate() error = %v, want docs-denied name drift failure", err)
	}
	if !listCalled {
		t.Fatal("List() should be called during pre-create resolution")
	}
	if createCalled {
		t.Fatal("Create() should not be called when immutable drift is detected on a tracked resource")
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.bucket.oc1..existing" {
		t.Fatalf("status.ocid = %q, want tracked OCID preserved", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCreateOrUpdateUsesUppercaseSpecIDAlias(t *testing.T) {
	t.Parallel()
	var getRequest fakeGetThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getRequest = *request.(*fakeGetThingRequest)
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..bound", DisplayName: "bound-name", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Spec: fakeSpec{Id: "ocid1.thing.oc1..bound"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when spec.Id binds an existing resource")
	}
	if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..bound" {
		t.Fatalf("get request thingId = %v, want bound OCID", getRequest.ThingId)
	}
}

func TestServiceClientCreateOrUpdateReusesExistingListMatchBeforeCreate(t *testing.T) {
	t.Parallel()
	createCalled := false
	var listRequest fakeListThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		createCalled = true
		t.Fatal("Create() should not be called when list lookup finds a reusable resource")
		return nil, nil
	}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		listRequest = *request.(*fakeListThingRequest)
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..existing", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list lookup reuses an existing resource")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list lookup finds a reusable resource")
	}
	if listRequest.CompartmentId != "ocid1.compartment.oc1..match" {
		t.Fatalf("list request compartmentId = %q, want ocid1.compartment.oc1..match", listRequest.CompartmentId)
	}
	if listRequest.Name != "wanted" {
		t.Fatalf("list request name = %q, want wanted", listRequest.Name)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..existing" {
		t.Fatalf("status.ocid = %q, want reused OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCreateOrUpdateSkipsPreCreateListResolutionWhenRequested(t *testing.T) {
	t.Parallel()
	createCalled := false
	listCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		createCalled = true
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..created", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}, nil
	}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		listCalled = true
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..existing", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted"}}
	response, err := client.CreateOrUpdate(WithSkipExistingBeforeCreate(context.Background()), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	if listCalled {
		t.Fatal("CreateOrUpdate() should skip list-before-create resolution when requested")
	}
	if !createCalled {
		t.Fatal("CreateOrUpdate() should call Create() when list-before-create resolution is skipped")
	}
	requireStatusOCID(t, resource, "ocid1.thing.oc1..created")
}

func TestServiceClientCreateOrUpdateSkipsUpdateAfterListReuseWhenMutableStateMatches(t *testing.T) {
	t.Parallel()
	createCalled := false
	getCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}}, Mutation: MutationSemantics{Mutable: []string{"displayName"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		createCalled = true
		t.Fatal("Create() should not be called when list lookup finds a reusable resource")
		return nil, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
			t.Fatalf("get request thingId = %v, want reused OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..existing", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", DisplayName: "steady-name", LifecycleState: "ACTIVE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..existing", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}, Update: &Operation{NewRequest: func() any {
		return &fakeUpdateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		t.Fatal("Update() should not be called when live mutable state already matches")
		return nil, nil
	}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted", DisplayName: "steady-name"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list reuse finds matching mutable state")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list lookup finds a reusable resource")
	}
	if !getCalled {
		t.Fatal("Get() should be called after list reuse to compare mutable fields")
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..existing" {
		t.Fatalf("status.ocid = %q, want reused OCID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.DisplayName != "steady-name" {
		t.Fatalf("status.displayName = %q, want steady-name", resource.Status.DisplayName)
	}
}

func TestServiceClientCreateOrUpdateSkipsGetWithoutOcidBeforeListReuse(t *testing.T) {
	t.Parallel()
	createCalled := false
	getCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		createCalled = true
		t.Fatal("Create() should not be called when list lookup finds a reusable resource")
		return nil, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if request.(*fakeGetThingRequest).ThingId == nil {
			t.Fatal("Get() should not be called without a resource OCID")
		}
		return fakeGetThingResponse{}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..existing", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list lookup reuses an existing resource")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list lookup finds a reusable resource")
	}
	if getCalled {
		t.Fatal("Get() should be skipped when no resource OCID is recorded")
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..existing" {
		t.Fatalf("status.ocid = %q, want reused OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCreateOrUpdateCreatesWhenLiveGetMissesAfterListReuse(t *testing.T) {
	t.Parallel()
	createCalled := false
	var createRequest fakeCreateThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, Mutation: MutationSemantics{ForceNew: []string{"retentionInHours"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createCalled = true
		createRequest = *request.(*fakeCreateThingRequest)
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..created", DisplayName: "created-name", LifecycleState: "ACTIVE"}}, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return nil, errortest.NewServiceError(404, "NotFound", "thing not found")
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..stale", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted", DisplayName: "created-name", RetentionInHours: 48}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should create when the live Get no longer finds the list match")
	}
	if !createCalled {
		t.Fatal("Create() should be called when the live Get no longer finds the list match")
	}
	if createRequest.DisplayName != "created-name" {
		t.Fatalf("create request displayName = %q, want created-name", createRequest.DisplayName)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.Id != "ocid1.thing.oc1..created" {
		t.Fatalf("status.id = %q, want created OCID", resource.Status.Id)
	}
}

func TestServiceClientCreateOrUpdateRebindsWhenTrackedStatusIDIsStale(t *testing.T) {
	t.Parallel()
	createCalled := false
	var listRequest fakeListThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"id", "name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		createCalled = true
		t.Fatal("Create() should not be called when list fallback rebinds a replacement resource")
		return nil, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..stale" {
			t.Fatalf("get request thingId = %v, want stale tracked OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return nil, errortest.NewServiceError(404, "NotFound", "thing not found")
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		listRequest = *request.(*fakeListThingRequest)
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..replacement", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Id", RequestName: "id", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..stale"}, Id: "ocid1.thing.oc1..stale", CompartmentId: "ocid1.compartment.oc1..match"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should rebind when the tracked OCID is stale")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list fallback rebinds a replacement resource")
	}
	if listRequest.Id != "" {
		t.Fatalf("list request id = %q, want empty after stale tracked ID fallback", listRequest.Id)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..replacement" {
		t.Fatalf("status.ocid = %q, want replacement OCID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.Id != "ocid1.thing.oc1..replacement" {
		t.Fatalf("status.id = %q, want replacement OCID", resource.Status.Id)
	}
}

func TestServiceClientCreateOrUpdateCreatesWhenTrackedStatusIDIsStaleAndNoReplacementExists(t *testing.T) {
	t.Parallel()
	var createRequest fakeCreateThingRequest
	var listRequest fakeListThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"id", "name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createRequest = *request.(*fakeCreateThingRequest)
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..created", CompartmentId: "ocid1.compartment.oc1..match", DisplayName: "created-name", LifecycleState: "ACTIVE"}}, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..stale" {
			t.Fatalf("get request thingId = %v, want stale tracked OCID", request.(*fakeGetThingRequest).ThingId)
		}
		return nil, errortest.NewServiceError(404, "NotFound", "thing not found")
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		listRequest = *request.(*fakeListThingRequest)
		return fakeListThingResponse{Collection: fakeThingCollection{Items: nil}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Id", RequestName: "id", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", DisplayName: "created-name", Name: "wanted"}, Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..stale"}, Id: "ocid1.thing.oc1..stale", CompartmentId: "ocid1.compartment.oc1..match"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should recreate when the tracked OCID is stale and no replacement exists")
	}
	if listRequest.Id != "" {
		t.Fatalf("list request id = %q, want empty after stale tracked ID fallback", listRequest.Id)
	}
	if createRequest.DisplayName != "created-name" {
		t.Fatalf("create request displayName = %q, want created-name", createRequest.DisplayName)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.Id != "ocid1.thing.oc1..created" {
		t.Fatalf("status.id = %q, want created OCID", resource.Status.Id)
	}
}

func TestServiceClientCreateOrUpdateIgnoresDeleteCandidatesDuringPreCreateLookup(t *testing.T) {
	t.Parallel()
	createCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createCalled = true
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..created", DisplayName: request.(*fakeCreateThingRequest).DisplayName, LifecycleState: "ACTIVE"}}, nil
	}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..deleted", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "DELETED"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted", DisplayName: "created-name"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when create replaces a deleted list match")
	}
	if !createCalled {
		t.Fatal("Create() should be called when only delete-phase list entries match")
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", resource.Status.OsokStatus.Ocid)
	}
}
