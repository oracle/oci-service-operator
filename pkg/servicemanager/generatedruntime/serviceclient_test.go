/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"testing"

	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeResource struct {
	Name      string     `json:"-"`
	Namespace string     `json:"-"`
	Spec      fakeSpec   `json:"spec,omitempty"`
	Status    fakeStatus `json:"status,omitempty"`
}

type fakeSpec struct {
	CompartmentId string `json:"compartmentId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	Enabled       bool   `json:"enabled,omitempty"`
}

type fakeStatus struct {
	OsokStatus     shared.OSOKStatus `json:"status"`
	Id             string            `json:"id,omitempty"`
	DisplayName    string            `json:"displayName,omitempty"`
	LifecycleState string            `json:"lifecycleState,omitempty"`
}

type fakeThing struct {
	Id             string `json:"id,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
}

type fakeThingSummary struct {
	Id             string `json:"id,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
}

type FakeCreateThingDetails struct {
	CompartmentId string `json:"compartmentId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	Enabled       bool   `json:"enabled,omitempty"`
}

type fakeCreateThingRequest struct {
	FakeCreateThingDetails `contributesTo:"body"`
}

type fakeCreateThingResponse struct {
	Thing fakeThing `presentIn:"body"`
}

type fakeGetThingRequest struct {
	ThingId *string `contributesTo:"path" name:"thingId"`
}

type fakeGetThingResponse struct {
	Thing fakeThing `presentIn:"body"`
}

type FakeUpdateThingDetails struct {
	DisplayName string `json:"displayName,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
}

type fakeUpdateThingRequest struct {
	ThingId                *string `contributesTo:"path" name:"thingId"`
	FakeUpdateThingDetails `contributesTo:"body"`
}

type fakeUpdateThingResponse struct {
	Thing fakeThing `presentIn:"body"`
}

type fakeDeleteThingRequest struct {
	ThingId *string `contributesTo:"path" name:"thingId"`
}

type fakeDeleteThingResponse struct{}

type fakeListThingRequest struct {
	DisplayName string `contributesTo:"query" name:"displayName"`
}

type fakeThingCollection struct {
	Items []fakeThingSummary `json:"items,omitempty"`
}

type fakeListThingResponse struct {
	Collection fakeThingCollection `presentIn:"body"`
}

type fakeServiceError struct {
	code       string
	message    string
	statusCode int
	opcID      string
}

func (f fakeServiceError) Error() string          { return f.message }
func (f fakeServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeServiceError) GetMessage() string     { return f.message }
func (f fakeServiceError) GetCode() string        { return f.code }
func (f fakeServiceError) GetOpcRequestID() string {
	return f.opcID
}

func TestServiceClientCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest fakeCreateThingRequest
	var getRequest fakeGetThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*fakeCreateThingRequest)
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..create",
						DisplayName:    "created-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest = *request.(*fakeGetThingRequest)
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..create",
						DisplayName:    "created-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
	})

	resource := &fakeResource{
		Name: "thing-sample",
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "desired-name",
			Enabled:       true,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if createRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create request compartmentId = %q, want %q", createRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("create request displayName = %q, want %q", createRequest.DisplayName, resource.Spec.DisplayName)
	}
	if !createRequest.Enabled {
		t.Fatal("create request enabled flag was not projected from spec")
	}
	if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..create" {
		t.Fatalf("get request thingId = %v, want created OCID", getRequest.ThingId)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..create" {
		t.Fatalf("status.ocid = %q, want created OCID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.Id != "ocid1.thing.oc1..create" {
		t.Fatalf("status.id = %q, want created OCID", resource.Status.Id)
	}
	if resource.Status.DisplayName != "created-name" {
		t.Fatalf("status.displayName = %q, want created-name", resource.Status.DisplayName)
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.CreatedAt == nil {
		t.Fatal("status.createdAt should be set after create")
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Active {
		t.Fatalf("status conditions = %#v, want trailing Active condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientCreateOrUpdateUpdatesExistingResource(t *testing.T) {
	t.Parallel()

	var updateRequest fakeUpdateThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*fakeUpdateThingRequest)
				return fakeUpdateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "updated-name",
						LifecycleState: "UPDATING",
					},
				}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "updated-name",
						LifecycleState: "UPDATING",
					},
				}, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "updated-name",
			Enabled:     true,
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:         "ocid1.thing.oc1..existing",
		},
	}

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

func TestServiceClientCreateOrUpdateFallsBackToList(t *testing.T) {
	t.Parallel()

	var listRequest fakeListThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{Id: "ocid1.thing.oc1..other", DisplayName: "other", LifecycleState: "ACTIVE"},
							{Id: "ocid1.thing.oc1..match", DisplayName: "wanted", LifecycleState: "ACTIVE"},
						},
					},
				}, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{DisplayName: "wanted"},
	}

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

func TestServiceClientDeleteTreatsNotFoundAsDeleted(t *testing.T) {
	t.Parallel()

	var deleteRequest fakeDeleteThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*fakeDeleteThingRequest)
				return fakeDeleteThingResponse{}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, fakeServiceError{
					code:       "NotAuthorizedOrNotFound",
					message:    "thing not found",
					statusCode: 404,
					opcID:      "opc-test",
				}
			},
		},
	})

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"},
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success when OCI returns not found")
	}
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..delete" {
		t.Fatalf("delete request thingId = %v, want existing OCID", deleteRequest.ThingId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
}
