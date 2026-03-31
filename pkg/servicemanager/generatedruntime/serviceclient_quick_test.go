/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type streamMutationQuickCase struct {
	PreCreateReuse bool
	LiveGetMisses  bool
	ForceNewDrift  bool
	MutableDrift   bool
}

func (streamMutationQuickCase) Generate(r *rand.Rand, _ int) reflect.Value {
	tc := streamMutationQuickCase{
		PreCreateReuse: r.Intn(2) == 0,
		ForceNewDrift:  r.Intn(2) == 0,
		MutableDrift:   r.Intn(2) == 0,
	}
	if tc.PreCreateReuse {
		tc.LiveGetMisses = r.Intn(3) == 0
	}
	return reflect.ValueOf(tc)
}

type streamStaleTrackedIDQuickCase struct {
	ReplacementExists bool
	StatusIDSource    uint8
}

func (streamStaleTrackedIDQuickCase) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(streamStaleTrackedIDQuickCase{
		ReplacementExists: r.Intn(2) == 0,
		StatusIDSource:    uint8(r.Intn(3)),
	})
}

func TestServiceClientCreateOrUpdateQuickHonorsStreamMutationDecisionMatrix(t *testing.T) {
	t.Parallel()

	var evalErr error
	if err := quick.Check(func(tc streamMutationQuickCase) bool {
		evalErr = evaluateStreamMutationQuickCase(tc)
		return evalErr == nil
	}, streamQuickConfig(1774907911310273)); err != nil {
		t.Fatalf("stream mutation property failed: %v: %v", err, evalErr)
	}
}

func TestServiceClientCreateOrUpdateQuickClearsStatusOnlyTrackedIDsAfter404(t *testing.T) {
	t.Parallel()

	var evalErr error
	if err := quick.Check(func(tc streamStaleTrackedIDQuickCase) bool {
		evalErr = evaluateStreamStaleTrackedIDQuickCase(tc)
		return evalErr == nil
	}, streamQuickConfig(1774907911310274)); err != nil {
		t.Fatalf("stale tracked ID property failed: %v: %v", err, evalErr)
	}
}

func evaluateStreamMutationQuickCase(tc streamMutationQuickCase) error {
	const (
		compartmentID    = "ocid1.compartment.oc1..match"
		existingID       = "ocid1.thing.oc1..existing"
		createdID        = "ocid1.thing.oc1..created"
		resourceName     = "wanted"
		liveDisplayName  = "steady-name"
		driftDisplayName = "desired-name"
		liveRetention    = 24
		driftRetention   = 48
	)

	listCalls := 0
	getCalls := 0
	createCalls := 0
	updateCalls := 0

	desiredDisplay := liveDisplayName
	if tc.MutableDrift {
		desiredDisplay = driftDisplayName
	}
	desiredRetention := liveRetention
	if tc.ForceNewDrift {
		desiredRetention = driftRetention
	}

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId:    compartmentID,
			Name:             resourceName,
			DisplayName:      desiredDisplay,
			RetentionInHours: desiredRetention,
		},
	}
	if !tc.PreCreateReuse {
		resource.Status = fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: existingID},
			Id:         existingID,
		}
	}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable:  []string{"displayName"},
				ForceNew: []string{"retentionInHours"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalls++
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:               createdID,
						Name:             resourceName,
						CompartmentId:    compartmentID,
						DisplayName:      desiredDisplay,
						RetentionInHours: desiredRetention,
						LifecycleState:   "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "FakeCreateThingDetails", Contribution: "body"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				getCalls++
				if tc.PreCreateReuse && tc.LiveGetMisses {
					return nil, fakeServiceError{
						code:       "NotAuthorizedOrNotFound",
						message:    "thing not found",
						statusCode: 404,
						opcID:      "opc-test",
					}
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:               existingID,
						Name:             resourceName,
						CompartmentId:    compartmentID,
						DisplayName:      liveDisplayName,
						RetentionInHours: liveRetention,
						LifecycleState:   "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				listCalls++
				if !tc.PreCreateReuse {
					return fakeListThingResponse{Collection: fakeThingCollection{}}, nil
				}
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             existingID,
								Name:           resourceName,
								CompartmentId:  compartmentID,
								LifecycleState: "ACTIVE",
							},
						},
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				updateCalls++
				return fakeUpdateThingResponse{
					Thing: fakeThing{
						Id:               existingID,
						Name:             resourceName,
						CompartmentId:    compartmentID,
						DisplayName:      desiredDisplay,
						RetentionInHours: liveRetention,
						LifecycleState:   "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
				{FieldName: "FakeUpdateThingDetails", Contribution: "body"},
			},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	wantListCalls := 0
	if tc.PreCreateReuse {
		wantListCalls = 1
	}
	if listCalls != wantListCalls {
		return fmt.Errorf("listCalls=%d, want %d for %+v", listCalls, wantListCalls, tc)
	}
	if getCalls != 1 {
		return fmt.Errorf("getCalls=%d, want 1 for %+v", getCalls, tc)
	}

	switch {
	case tc.PreCreateReuse && tc.LiveGetMisses:
		if err != nil {
			return fmt.Errorf("CreateOrUpdate() error=%v, want create fallback success for %+v", err, tc)
		}
		if !response.IsSuccessful || response.ShouldRequeue {
			return fmt.Errorf("response=%+v, want immediate success for %+v", response, tc)
		}
		if createCalls != 1 || updateCalls != 0 {
			return fmt.Errorf("createCalls=%d updateCalls=%d, want create=1 update=0 for %+v", createCalls, updateCalls, tc)
		}
		if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
			return fmt.Errorf("status.ocid=%q, want %q for %+v", got, createdID, tc)
		}
		return nil
	case tc.ForceNewDrift:
		if err == nil || !strings.Contains(err.Error(), "require replacement when retentionInHours changes") {
			return fmt.Errorf("CreateOrUpdate() error=%v, want live force-new replacement failure for %+v", err, tc)
		}
		if createCalls != 0 || updateCalls != 0 {
			return fmt.Errorf("createCalls=%d updateCalls=%d, want create=0 update=0 for %+v", createCalls, updateCalls, tc)
		}
		if !tc.PreCreateReuse {
			if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
				return fmt.Errorf("status.ocid=%q, want %q for %+v", got, existingID, tc)
			}
		} else if resource.Status.Id != existingID {
			return fmt.Errorf("status.id=%q, want %q for %+v", resource.Status.Id, existingID, tc)
		}
		if resource.Status.RetentionInHours != liveRetention {
			return fmt.Errorf("status.retentionInHours=%d, want %d from live GET for %+v", resource.Status.RetentionInHours, liveRetention, tc)
		}
		return nil
	case tc.MutableDrift:
		if err != nil {
			return fmt.Errorf("CreateOrUpdate() error=%v, want update success for %+v", err, tc)
		}
		if !response.IsSuccessful || response.ShouldRequeue {
			return fmt.Errorf("response=%+v, want update success without requeue for %+v", response, tc)
		}
		if createCalls != 0 || updateCalls != 1 {
			return fmt.Errorf("createCalls=%d updateCalls=%d, want create=0 update=1 for %+v", createCalls, updateCalls, tc)
		}
		if resource.Status.DisplayName != desiredDisplay {
			return fmt.Errorf("status.displayName=%q, want %q for %+v", resource.Status.DisplayName, desiredDisplay, tc)
		}
		return nil
	default:
		if err != nil {
			return fmt.Errorf("CreateOrUpdate() error=%v, want observe success for %+v", err, tc)
		}
		if !response.IsSuccessful || response.ShouldRequeue {
			return fmt.Errorf("response=%+v, want observe success without requeue for %+v", response, tc)
		}
		if createCalls != 0 || updateCalls != 0 {
			return fmt.Errorf("createCalls=%d updateCalls=%d, want create=0 update=0 for %+v", createCalls, updateCalls, tc)
		}
		if resource.Status.DisplayName != liveDisplayName {
			return fmt.Errorf("status.displayName=%q, want %q for %+v", resource.Status.DisplayName, liveDisplayName, tc)
		}
		return nil
	}
}

func evaluateStreamStaleTrackedIDQuickCase(tc streamStaleTrackedIDQuickCase) error {
	const (
		compartmentID   = "ocid1.compartment.oc1..match"
		staleID         = "ocid1.thing.oc1..stale"
		replacementID   = "ocid1.thing.oc1..replacement"
		createdID       = "ocid1.thing.oc1..created"
		resourceName    = "wanted"
		createdResource = "created-name"
	)

	listCalls := 0
	getCalls := 0
	createCalls := 0
	updateCalls := 0
	var getRequest fakeGetThingRequest
	var listRequest fakeListThingRequest

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: compartmentID,
			Name:          resourceName,
			DisplayName:   createdResource,
		},
	}
	switch tc.StatusIDSource {
	case 0:
		resource.Status.OsokStatus.Ocid = staleID
	case 1:
		resource.Status.Id = staleID
	default:
		resource.Status.OsokStatus.Ocid = staleID
		resource.Status.Id = staleID
	}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"id", "name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalls++
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             createdID,
						Name:           resourceName,
						CompartmentId:  compartmentID,
						DisplayName:    createdResource,
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "FakeCreateThingDetails", Contribution: "body"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				getRequest = *request.(*fakeGetThingRequest)
				return nil, fakeServiceError{
					code:       "NotAuthorizedOrNotFound",
					message:    "thing not found",
					statusCode: 404,
					opcID:      "opc-test",
				}
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listCalls++
				listRequest = *request.(*fakeListThingRequest)
				if !tc.ReplacementExists {
					return fakeListThingResponse{Collection: fakeThingCollection{}}, nil
				}
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             replacementID,
								Name:           resourceName,
								CompartmentId:  compartmentID,
								LifecycleState: "ACTIVE",
							},
						},
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Id", RequestName: "id", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				updateCalls++
				return fakeUpdateThingResponse{}, nil
			},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		return fmt.Errorf("CreateOrUpdate() error=%v for %+v", err, tc)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		return fmt.Errorf("response=%+v, want immediate success for %+v", response, tc)
	}
	if getCalls != 1 {
		return fmt.Errorf("getCalls=%d, want 1 for %+v", getCalls, tc)
	}
	if listCalls != 1 {
		return fmt.Errorf("listCalls=%d, want 1 for %+v", listCalls, tc)
	}
	if updateCalls != 0 {
		return fmt.Errorf("updateCalls=%d, want 0 for %+v", updateCalls, tc)
	}
	if getRequest.ThingId == nil || *getRequest.ThingId != staleID {
		return fmt.Errorf("get request thingId=%v, want stale ID %q for %+v", getRequest.ThingId, staleID, tc)
	}
	if listRequest.Id != "" {
		return fmt.Errorf("list request id=%q, want stale tracked ID cleared for %+v", listRequest.Id, tc)
	}
	if listRequest.Name != resourceName || listRequest.CompartmentId != compartmentID {
		return fmt.Errorf("list request=%+v, want name=%q compartmentId=%q for %+v", listRequest, resourceName, compartmentID, tc)
	}

	wantID := replacementID
	wantCreates := 0
	if !tc.ReplacementExists {
		wantID = createdID
		wantCreates = 1
	}
	if createCalls != wantCreates {
		return fmt.Errorf("createCalls=%d, want %d for %+v", createCalls, wantCreates, tc)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		return fmt.Errorf("status.ocid=%q, want %q for %+v", got, wantID, tc)
	}
	if resource.Status.Id != wantID {
		return fmt.Errorf("status.id=%q, want %q for %+v", resource.Status.Id, wantID, tc)
	}
	return nil
}

func streamQuickConfig(seed int64) *quick.Config {
	return &quick.Config{
		MaxCount: 96,
		Rand:     rand.New(rand.NewSource(seed)),
	}
}
