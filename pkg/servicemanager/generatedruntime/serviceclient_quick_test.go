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

	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
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

const (
	streamMutationQuickCompartmentID    = "ocid1.compartment.oc1..match"
	streamMutationQuickExistingID       = "ocid1.thing.oc1..existing"
	streamMutationQuickCreatedID        = "ocid1.thing.oc1..created"
	streamMutationQuickResourceName     = "wanted"
	streamMutationQuickLiveDisplayName  = "steady-name"
	streamMutationQuickDriftDisplayName = "desired-name"
	streamMutationQuickLiveRetention    = 24
	streamMutationQuickDriftRetention   = 48

	streamStaleQuickCompartmentID   = "ocid1.compartment.oc1..match"
	streamStaleQuickStaleID         = "ocid1.thing.oc1..stale"
	streamStaleQuickReplacementID   = "ocid1.thing.oc1..replacement"
	streamStaleQuickCreatedID       = "ocid1.thing.oc1..created"
	streamStaleQuickResourceName    = "wanted"
	streamStaleQuickCreatedResource = "created-name"
)

type streamMutationQuickEnv struct {
	tc               streamMutationQuickCase
	resource         *fakeResource
	desiredDisplay   string
	desiredRetention int
	createCalls      int
	getCalls         int
	listCalls        int
	updateCalls      int
}

func evaluateStreamMutationQuickCase(tc streamMutationQuickCase) error {
	env := newStreamMutationQuickEnv(tc)
	response, err := env.newClient().CreateOrUpdate(context.Background(), env.resource, ctrl.Request{})
	if err := env.assertCommonCalls(); err != nil {
		return err
	}
	return env.assertOutcome(response, err)
}

func newStreamMutationQuickEnv(tc streamMutationQuickCase) *streamMutationQuickEnv {
	env := &streamMutationQuickEnv{
		tc:               tc,
		desiredDisplay:   streamMutationQuickLiveDisplayName,
		desiredRetention: streamMutationQuickLiveRetention,
	}
	if tc.MutableDrift {
		env.desiredDisplay = streamMutationQuickDriftDisplayName
	}
	if tc.ForceNewDrift {
		env.desiredRetention = streamMutationQuickDriftRetention
	}

	env.resource = &fakeResource{
		Spec: fakeSpec{
			CompartmentId:    streamMutationQuickCompartmentID,
			Name:             streamMutationQuickResourceName,
			DisplayName:      env.desiredDisplay,
			RetentionInHours: env.desiredRetention,
		},
	}
	if !tc.PreCreateReuse {
		env.resource.Status = fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: streamMutationQuickExistingID},
			Id:         streamMutationQuickExistingID,
		}
	}
	return env
}

func (env *streamMutationQuickEnv) newClient() ServiceClient[*fakeResource] {
	return NewServiceClient[*fakeResource](Config[*fakeResource]{
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
			Call:       env.createThing,
			Fields: []RequestField{
				{FieldName: "FakeCreateThingDetails", Contribution: "body"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call:       env.getThing,
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call:       env.listThing,
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call:       env.updateThing,
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
				{FieldName: "FakeUpdateThingDetails", Contribution: "body"},
			},
		},
	})
}

func (env *streamMutationQuickEnv) createThing(_ context.Context, _ any) (any, error) {
	env.createCalls++
	return fakeCreateThingResponse{
		Thing: fakeThing{
			Id:               streamMutationQuickCreatedID,
			Name:             streamMutationQuickResourceName,
			CompartmentId:    streamMutationQuickCompartmentID,
			DisplayName:      env.desiredDisplay,
			RetentionInHours: env.desiredRetention,
			LifecycleState:   "ACTIVE",
		},
	}, nil
}

func (env *streamMutationQuickEnv) getThing(_ context.Context, _ any) (any, error) {
	env.getCalls++
	if env.tc.PreCreateReuse && env.tc.LiveGetMisses {
		return nil, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "thing not found")
	}
	return fakeGetThingResponse{
		Thing: fakeThing{
			Id:               streamMutationQuickExistingID,
			Name:             streamMutationQuickResourceName,
			CompartmentId:    streamMutationQuickCompartmentID,
			DisplayName:      streamMutationQuickLiveDisplayName,
			RetentionInHours: streamMutationQuickLiveRetention,
			LifecycleState:   "ACTIVE",
		},
	}, nil
}

func (env *streamMutationQuickEnv) listThing(_ context.Context, _ any) (any, error) {
	env.listCalls++
	if !env.tc.PreCreateReuse {
		return fakeListThingResponse{Collection: fakeThingCollection{}}, nil
	}
	return fakeListThingResponse{
		Collection: fakeThingCollection{
			Items: []fakeThingSummary{
				{
					Id:             streamMutationQuickExistingID,
					Name:           streamMutationQuickResourceName,
					CompartmentId:  streamMutationQuickCompartmentID,
					LifecycleState: "ACTIVE",
				},
			},
		},
	}, nil
}

func (env *streamMutationQuickEnv) updateThing(_ context.Context, _ any) (any, error) {
	env.updateCalls++
	return fakeUpdateThingResponse{
		Thing: fakeThing{
			Id:               streamMutationQuickExistingID,
			Name:             streamMutationQuickResourceName,
			CompartmentId:    streamMutationQuickCompartmentID,
			DisplayName:      env.desiredDisplay,
			RetentionInHours: streamMutationQuickLiveRetention,
			LifecycleState:   "ACTIVE",
		},
	}, nil
}

func (env *streamMutationQuickEnv) assertCommonCalls() error {
	wantListCalls := 0
	if env.tc.PreCreateReuse {
		wantListCalls = 1
	}
	if env.listCalls != wantListCalls {
		return fmt.Errorf("listCalls=%d, want %d for %+v", env.listCalls, wantListCalls, env.tc)
	}
	if env.getCalls != 1 {
		return fmt.Errorf("getCalls=%d, want 1 for %+v", env.getCalls, env.tc)
	}
	return nil
}

func (env *streamMutationQuickEnv) assertOutcome(response servicemanager.OSOKResponse, err error) error {
	switch {
	case env.tc.PreCreateReuse && env.tc.LiveGetMisses:
		return env.assertCreateFallback(response, err)
	case env.tc.ForceNewDrift:
		return env.assertForceNewFailure(err)
	case env.tc.MutableDrift:
		return env.assertUpdate(response, err)
	default:
		return env.assertObserve(response, err)
	}
}

func (env *streamMutationQuickEnv) assertCreateFallback(response servicemanager.OSOKResponse, err error) error {
	if err := assertQuickImmediateSuccess(response, err, env.tc, "create fallback success"); err != nil {
		return err
	}
	if err := env.assertCreateUpdateCalls(1, 0); err != nil {
		return err
	}
	if got := string(env.resource.Status.OsokStatus.Ocid); got != streamMutationQuickCreatedID {
		return fmt.Errorf("status.ocid=%q, want %q for %+v", got, streamMutationQuickCreatedID, env.tc)
	}
	return nil
}

func (env *streamMutationQuickEnv) assertForceNewFailure(err error) error {
	if err == nil || !strings.Contains(err.Error(), "require replacement when retentionInHours changes") {
		return fmt.Errorf("CreateOrUpdate() error=%v, want live force-new replacement failure for %+v", err, env.tc)
	}
	if err := env.assertCreateUpdateCalls(0, 0); err != nil {
		return err
	}
	if err := env.assertExistingStatusID(); err != nil {
		return err
	}
	if env.resource.Status.RetentionInHours != streamMutationQuickLiveRetention {
		return fmt.Errorf(
			"status.retentionInHours=%d, want %d from live GET for %+v",
			env.resource.Status.RetentionInHours,
			streamMutationQuickLiveRetention,
			env.tc,
		)
	}
	return nil
}

func (env *streamMutationQuickEnv) assertUpdate(response servicemanager.OSOKResponse, err error) error {
	if err := assertQuickImmediateSuccess(response, err, env.tc, "update success without requeue"); err != nil {
		return err
	}
	if err := env.assertCreateUpdateCalls(0, 1); err != nil {
		return err
	}
	if env.resource.Status.DisplayName != env.desiredDisplay {
		return fmt.Errorf("status.displayName=%q, want %q for %+v", env.resource.Status.DisplayName, env.desiredDisplay, env.tc)
	}
	return nil
}

func (env *streamMutationQuickEnv) assertObserve(response servicemanager.OSOKResponse, err error) error {
	if err := assertQuickImmediateSuccess(response, err, env.tc, "observe success without requeue"); err != nil {
		return err
	}
	if err := env.assertCreateUpdateCalls(0, 0); err != nil {
		return err
	}
	if env.resource.Status.DisplayName != streamMutationQuickLiveDisplayName {
		return fmt.Errorf(
			"status.displayName=%q, want %q for %+v",
			env.resource.Status.DisplayName,
			streamMutationQuickLiveDisplayName,
			env.tc,
		)
	}
	return nil
}

func (env *streamMutationQuickEnv) assertCreateUpdateCalls(wantCreate int, wantUpdate int) error {
	if env.createCalls != wantCreate || env.updateCalls != wantUpdate {
		return fmt.Errorf(
			"createCalls=%d updateCalls=%d, want create=%d update=%d for %+v",
			env.createCalls,
			env.updateCalls,
			wantCreate,
			wantUpdate,
			env.tc,
		)
	}
	return nil
}

func (env *streamMutationQuickEnv) assertExistingStatusID() error {
	if !env.tc.PreCreateReuse {
		if got := string(env.resource.Status.OsokStatus.Ocid); got != streamMutationQuickExistingID {
			return fmt.Errorf("status.ocid=%q, want %q for %+v", got, streamMutationQuickExistingID, env.tc)
		}
		return nil
	}
	if env.resource.Status.Id != streamMutationQuickExistingID {
		return fmt.Errorf("status.id=%q, want %q for %+v", env.resource.Status.Id, streamMutationQuickExistingID, env.tc)
	}
	return nil
}

type streamStaleTrackedIDQuickEnv struct {
	tc          streamStaleTrackedIDQuickCase
	resource    *fakeResource
	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	getRequest  fakeGetThingRequest
	listRequest fakeListThingRequest
}

func evaluateStreamStaleTrackedIDQuickCase(tc streamStaleTrackedIDQuickCase) error {
	env := newStreamStaleTrackedIDQuickEnv(tc)
	response, err := env.newClient().CreateOrUpdate(context.Background(), env.resource, ctrl.Request{})
	if err := assertQuickImmediateSuccess(response, err, tc, "immediate success"); err != nil {
		return err
	}
	if err := env.assertCommonCalls(); err != nil {
		return err
	}
	if err := env.assertRequests(); err != nil {
		return err
	}
	return env.assertStatus()
}

func newStreamStaleTrackedIDQuickEnv(tc streamStaleTrackedIDQuickCase) *streamStaleTrackedIDQuickEnv {
	env := &streamStaleTrackedIDQuickEnv{
		tc: tc,
		resource: &fakeResource{
			Spec: fakeSpec{
				CompartmentId: streamStaleQuickCompartmentID,
				Name:          streamStaleQuickResourceName,
				DisplayName:   streamStaleQuickCreatedResource,
			},
		},
	}
	env.seedTrackedStatusID()
	return env
}

func (env *streamStaleTrackedIDQuickEnv) seedTrackedStatusID() {
	switch env.tc.StatusIDSource {
	case 0:
		env.resource.Status.OsokStatus.Ocid = streamStaleQuickStaleID
	case 1:
		env.resource.Status.Id = streamStaleQuickStaleID
	default:
		env.resource.Status.OsokStatus.Ocid = streamStaleQuickStaleID
		env.resource.Status.Id = streamStaleQuickStaleID
	}
}

func (env *streamStaleTrackedIDQuickEnv) newClient() ServiceClient[*fakeResource] {
	return NewServiceClient[*fakeResource](Config[*fakeResource]{
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
			Call:       env.createThing,
			Fields: []RequestField{
				{FieldName: "FakeCreateThingDetails", Contribution: "body"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call:       env.getThing,
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call:       env.listThing,
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Id", RequestName: "id", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call:       env.updateThing,
		},
	})
}

func (env *streamStaleTrackedIDQuickEnv) createThing(_ context.Context, _ any) (any, error) {
	env.createCalls++
	return fakeCreateThingResponse{
		Thing: fakeThing{
			Id:             streamStaleQuickCreatedID,
			Name:           streamStaleQuickResourceName,
			CompartmentId:  streamStaleQuickCompartmentID,
			DisplayName:    streamStaleQuickCreatedResource,
			LifecycleState: "ACTIVE",
		},
	}, nil
}

func (env *streamStaleTrackedIDQuickEnv) getThing(_ context.Context, request any) (any, error) {
	env.getCalls++
	env.getRequest = *request.(*fakeGetThingRequest)
	return nil, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "thing not found")
}

func (env *streamStaleTrackedIDQuickEnv) listThing(_ context.Context, request any) (any, error) {
	env.listCalls++
	env.listRequest = *request.(*fakeListThingRequest)
	if !env.tc.ReplacementExists {
		return fakeListThingResponse{Collection: fakeThingCollection{}}, nil
	}
	return fakeListThingResponse{
		Collection: fakeThingCollection{
			Items: []fakeThingSummary{
				{
					Id:             streamStaleQuickReplacementID,
					Name:           streamStaleQuickResourceName,
					CompartmentId:  streamStaleQuickCompartmentID,
					LifecycleState: "ACTIVE",
				},
			},
		},
	}, nil
}

func (env *streamStaleTrackedIDQuickEnv) updateThing(_ context.Context, _ any) (any, error) {
	env.updateCalls++
	return fakeUpdateThingResponse{}, nil
}

func (env *streamStaleTrackedIDQuickEnv) assertCommonCalls() error {
	if env.getCalls != 1 {
		return fmt.Errorf("getCalls=%d, want 1 for %+v", env.getCalls, env.tc)
	}
	if env.listCalls != 1 {
		return fmt.Errorf("listCalls=%d, want 1 for %+v", env.listCalls, env.tc)
	}
	if env.updateCalls != 0 {
		return fmt.Errorf("updateCalls=%d, want 0 for %+v", env.updateCalls, env.tc)
	}
	return nil
}

func (env *streamStaleTrackedIDQuickEnv) assertRequests() error {
	if env.getRequest.ThingId == nil || *env.getRequest.ThingId != streamStaleQuickStaleID {
		return fmt.Errorf("get request thingId=%v, want stale ID %q for %+v", env.getRequest.ThingId, streamStaleQuickStaleID, env.tc)
	}
	if env.listRequest.Id != "" {
		return fmt.Errorf("list request id=%q, want stale tracked ID cleared for %+v", env.listRequest.Id, env.tc)
	}
	if env.listRequest.Name != streamStaleQuickResourceName || env.listRequest.CompartmentId != streamStaleQuickCompartmentID {
		return fmt.Errorf(
			"list request=%+v, want name=%q compartmentId=%q for %+v",
			env.listRequest,
			streamStaleQuickResourceName,
			streamStaleQuickCompartmentID,
			env.tc,
		)
	}
	return nil
}

func (env *streamStaleTrackedIDQuickEnv) assertStatus() error {
	wantID := streamStaleQuickReplacementID
	wantCreates := 0
	if !env.tc.ReplacementExists {
		wantID = streamStaleQuickCreatedID
		wantCreates = 1
	}
	if env.createCalls != wantCreates {
		return fmt.Errorf("createCalls=%d, want %d for %+v", env.createCalls, wantCreates, env.tc)
	}
	if got := string(env.resource.Status.OsokStatus.Ocid); got != wantID {
		return fmt.Errorf("status.ocid=%q, want %q for %+v", got, wantID, env.tc)
	}
	if env.resource.Status.Id != wantID {
		return fmt.Errorf("status.id=%q, want %q for %+v", env.resource.Status.Id, wantID, env.tc)
	}
	return nil
}

func assertQuickImmediateSuccess(response servicemanager.OSOKResponse, err error, tc any, description string) error {
	if err != nil {
		return fmt.Errorf("CreateOrUpdate() error=%v, want %s for %+v", err, description, tc)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		return fmt.Errorf("response=%+v, want %s for %+v", response, description, tc)
	}
	return nil
}

func streamQuickConfig(seed int64) *quick.Config {
	return &quick.Config{
		MaxCount: 96,
		Rand:     rand.New(rand.NewSource(seed)),
	}
}
