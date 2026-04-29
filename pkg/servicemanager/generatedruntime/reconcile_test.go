/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestServiceClientCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()
	var createRequest fakeCreateThingRequest
	var getRequest fakeGetThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createRequest = *request.(*fakeCreateThingRequest)
		return fakeCreateThingResponse{OpcRequestId: stringPtr("opc-create-1"), Thing: fakeThing{Id: "ocid1.thing.oc1..create", DisplayName: "created-name", LifecycleState: "ACTIVE"}}, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getRequest = *request.(*fakeGetThingRequest)
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..create", DisplayName: "created-name", LifecycleState: "ACTIVE"}}, nil
	}}})
	resource := &fakeResource{Name: "thing-sample", Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..example", DisplayName: "desired-name", Enabled: true}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, false, "CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	requireCreateThingRequestMatchesSpec(t, createRequest, resource.Spec)
	requireThingIDRequest(t, "get", getRequest.ThingId, "ocid1.thing.oc1..create")
	requireStatusOCID(t, resource, "ocid1.thing.oc1..create")
	requireStatusOpcRequestID(t, resource, "opc-create-1")
	requireStringEqual(t, "status.id", resource.Status.Id, "ocid1.thing.oc1..create")
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "created-name")
	requireStringEqual(t, "status.lifecycleState", resource.Status.LifecycleState, "ACTIVE")
	requireCreatedAt(t, resource)
	requireTrailingCondition(t, resource, shared.Active)
}

func TestServiceClientCreateOrUpdateUsesExplicitRequestFieldsAndFormalLifecycle(t *testing.T) {
	t.Parallel()
	var createRequest fakeCreateThingRequest
	var getRequest fakeGetThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{FormalService: "identity", FormalSlug: "user", Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, ActiveStates: []string{"AVAILABLE"}}, Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, CreateFollowUp: FollowUpSemantics{Strategy: "read-after-write", Hooks: []Hook{{Helper: "tfresource.CreateResource"}}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createRequest = *request.(*fakeCreateThingRequest)
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..create", DisplayName: "created-name", LifecycleState: "CREATING"}}, nil
	}, Fields: []RequestField{{FieldName: "FakeCreateThingDetails", Contribution: "body"}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getRequest = *request.(*fakeGetThingRequest)
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..create", DisplayName: "created-name", LifecycleState: "AVAILABLE"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..example", DisplayName: "desired-name", Enabled: true}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for AVAILABLE lifecycle")
	}
	if createRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create request compartmentId = %q, want %q", createRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..create" {
		t.Fatalf("get request thingId = %v, want created OCID", getRequest.ThingId)
	}
	if resource.Status.LifecycleState != "AVAILABLE" {
		t.Fatalf("status.lifecycleState = %q, want AVAILABLE", resource.Status.LifecycleState)
	}
}
