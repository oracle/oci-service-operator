/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package autonomousdatabase

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"unicode"
	"unsafe"

	"github.com/oracle/oci-go-sdk/v65/common"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const quickAutonomousDatabaseOCID = "ocid1.autonomousdatabase.oc1..quick"

type quickAutonomousDatabaseCreateRequest struct {
	CreateAutonomousDatabaseDetails any
	CreateAutonomousDatabaseBase    any
}

type quickAutonomousDatabaseGetRequest struct {
	AutonomousDatabaseId *string
}

type quickAutonomousDatabaseDeleteRequest struct {
	AutonomousDatabaseId *string
}

type quickAutonomousDatabaseListRequest struct {
	CompartmentId string
	DisplayName   string
}

type quickAutonomousDatabaseState struct {
	Id               string `json:"id,omitempty"`
	CompartmentId    string `json:"compartmentId,omitempty"`
	DisplayName      string `json:"displayName,omitempty"`
	LifecycleState   string `json:"lifecycleState,omitempty"`
	LifecycleDetails string `json:"lifecycleDetails,omitempty"`
}

type quickAutonomousDatabaseCreateResponse struct {
	AutonomousDatabase quickAutonomousDatabaseState `presentIn:"body"`
}

type quickAutonomousDatabaseGetResponse struct {
	AutonomousDatabase quickAutonomousDatabaseState `presentIn:"body"`
}

type quickAutonomousDatabaseCollection struct {
	Items []quickAutonomousDatabaseState `json:"items,omitempty"`
}

type quickAutonomousDatabaseListResponse struct {
	Collection quickAutonomousDatabaseCollection `presentIn:"body"`
}

type quickAutonomousDatabaseDeleteCase struct {
	deleted bool
	states  []string
}

func TestAutonomousDatabaseRuntimeLifecycleClassificationQuick(t *testing.T) {
	t.Parallel()

	cfg := quickAutonomousDatabaseRuntimeConfig(t)
	states := append([]string{}, cfg.Semantics.Lifecycle.ProvisioningStates...)
	states = append(states, cfg.Semantics.Lifecycle.UpdatingStates...)
	states = append(states, cfg.Semantics.Lifecycle.ActiveStates...)
	if len(states) == 0 {
		t.Fatal("AutonomousDatabase lifecycle states should not be empty")
	}

	property := func(stateIndex uint8, caseMask uint64) bool {
		lifecycleState := quickAutonomousDatabaseCaseMask(pickQuickState(states, stateIndex), caseMask)
		wantCondition := quickAutonomousDatabaseLifecycleCondition(cfg.Semantics, lifecycleState)
		client := newQuickAutonomousDatabaseCreateClient(t, cfg, lifecycleState)
		resource := &databasev1beta1.AutonomousDatabase{}

		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if err != nil {
			t.Logf("CreateOrUpdate(%q) returned error: %v", lifecycleState, err)
			return false
		}
		if !response.IsSuccessful {
			t.Logf("CreateOrUpdate(%q) reported unsuccessful reconcile", lifecycleState)
			return false
		}
		if response.ShouldRequeue != (wantCondition != shared.Active) {
			t.Logf("CreateOrUpdate(%q) requeue = %t, want %t", lifecycleState, response.ShouldRequeue, wantCondition != shared.Active)
			return false
		}
		if got := quickAutonomousDatabaseLastCondition(resource); got != wantCondition {
			t.Logf("CreateOrUpdate(%q) condition = %s, want %s", lifecycleState, got, wantCondition)
			return false
		}
		if got := string(resource.Status.OsokStatus.Ocid); got != quickAutonomousDatabaseOCID {
			t.Logf("CreateOrUpdate(%q) status.ocid = %q, want %q", lifecycleState, got, quickAutonomousDatabaseOCID)
			return false
		}
		if got := resource.Status.LifecycleState; got != lifecycleState {
			t.Logf("CreateOrUpdate(%q) lifecycleState = %q, want %q", lifecycleState, got, lifecycleState)
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 96}); err != nil {
		t.Fatal(err)
	}
}

func TestAutonomousDatabaseDeleteConfirmationQuick(t *testing.T) {
	t.Parallel()

	cfg := quickAutonomousDatabaseRuntimeConfig(t)
	cases := quickAutonomousDatabaseDeleteCases(t, cfg)

	property := func(caseIndex uint8, stateIndex uint8, caseMask uint64) bool {
		return quickAutonomousDatabaseDeleteConfirmationProperty(t, cfg, cases, caseIndex, stateIndex, caseMask)
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 96}); err != nil {
		t.Fatal(err)
	}
}

func TestAutonomousDatabaseDeleteLookupByListQuick(t *testing.T) {
	t.Parallel()

	cfg := quickAutonomousDatabaseRuntimeConfig(t)

	property := func(seed uint32, rawDisplayName string) bool {
		return quickAutonomousDatabaseDeleteLookupByListProperty(t, cfg, seed, rawDisplayName)
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 64}); err != nil {
		t.Fatal(err)
	}
}

func quickAutonomousDatabaseDeleteCases(
	t *testing.T,
	cfg generatedruntime.Config[*databasev1beta1.AutonomousDatabase],
) []quickAutonomousDatabaseDeleteCase {
	t.Helper()

	cases := []quickAutonomousDatabaseDeleteCase{
		{deleted: false, states: cfg.Semantics.Delete.PendingStates},
		{deleted: true, states: cfg.Semantics.Delete.TerminalStates},
	}
	for _, testCase := range cases {
		if len(testCase.states) == 0 {
			t.Fatal("delete confirmation runtime states should not be empty")
		}
	}

	return cases
}

func quickAutonomousDatabaseDeleteConfirmationProperty(
	t *testing.T,
	cfg generatedruntime.Config[*databasev1beta1.AutonomousDatabase],
	cases []quickAutonomousDatabaseDeleteCase,
	caseIndex uint8,
	stateIndex uint8,
	caseMask uint64,
) bool {
	t.Helper()

	testCase := cases[int(caseIndex)%len(cases)]
	lifecycleState := quickAutonomousDatabaseCaseMask(pickQuickState(testCase.states, stateIndex), caseMask)
	client := newQuickAutonomousDatabaseDeleteClient(t, cfg, lifecycleState)
	resource := &databasev1beta1.AutonomousDatabase{}
	resource.Status.OsokStatus.Ocid = shared.OCID(quickAutonomousDatabaseOCID)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Logf("Delete(%q) returned error: %v", lifecycleState, err)
		return false
	}

	return quickAutonomousDatabaseValidateDeleteConfirmation(t, resource, lifecycleState, deleted, testCase.deleted)
}

func quickAutonomousDatabaseValidateDeleteConfirmation(
	t *testing.T,
	resource *databasev1beta1.AutonomousDatabase,
	lifecycleState string,
	deleted bool,
	wantDeleted bool,
) bool {
	t.Helper()

	if deleted != wantDeleted {
		t.Logf("Delete(%q) deleted = %t, want %t", lifecycleState, deleted, wantDeleted)
		return false
	}
	if got := quickAutonomousDatabaseLastCondition(resource); got != shared.Terminating {
		t.Logf("Delete(%q) condition = %s, want %s", lifecycleState, got, shared.Terminating)
		return false
	}
	if wantDeleted && resource.Status.OsokStatus.DeletedAt == nil {
		t.Logf("Delete(%q) should set deletedAt", lifecycleState)
		return false
	}
	if !wantDeleted && resource.Status.OsokStatus.DeletedAt != nil {
		t.Logf("Delete(%q) should not set deletedAt while OCI remains pending", lifecycleState)
		return false
	}
	if got := resource.Status.LifecycleState; got != lifecycleState {
		t.Logf("Delete(%q) lifecycleState = %q, want %q", lifecycleState, got, lifecycleState)
		return false
	}

	return true
}

func quickAutonomousDatabaseDeleteLookupByListProperty(
	t *testing.T,
	cfg generatedruntime.Config[*databasev1beta1.AutonomousDatabase],
	seed uint32,
	rawDisplayName string,
) bool {
	t.Helper()

	compartmentID := fmt.Sprintf("ocid1.compartment.oc1..%08x", seed)
	displayName := quickAutonomousDatabaseDisplayName(rawDisplayName)
	var deletedID string
	listCalls := 0

	client := newQuickAutonomousDatabaseListDeleteClient(t, cfg, compartmentID, displayName, &deletedID, &listCalls)
	resource := &databasev1beta1.AutonomousDatabase{
		Spec: databasev1beta1.AutonomousDatabaseSpec{
			CompartmentId: compartmentID,
			DisplayName:   displayName,
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Logf("Delete(list lookup %q/%q) returned error: %v", compartmentID, displayName, err)
		return false
	}

	return quickAutonomousDatabaseValidateDeleteLookupByList(
		t,
		resource,
		compartmentID,
		displayName,
		deleted,
		deletedID,
		listCalls,
	)
}

func quickAutonomousDatabaseValidateDeleteLookupByList(
	t *testing.T,
	resource *databasev1beta1.AutonomousDatabase,
	compartmentID string,
	displayName string,
	deleted bool,
	deletedID string,
	listCalls int,
) bool {
	t.Helper()

	if !deleted {
		t.Logf("Delete(list lookup %q/%q) should complete once the follow-up list returns no items", compartmentID, displayName)
		return false
	}
	if deletedID != quickAutonomousDatabaseOCID {
		t.Logf("Delete(list lookup %q/%q) id = %q, want %q", compartmentID, displayName, deletedID, quickAutonomousDatabaseOCID)
		return false
	}
	if listCalls != 2 {
		t.Logf("Delete(list lookup %q/%q) list calls = %d, want 2", compartmentID, displayName, listCalls)
		return false
	}
	if got := resource.Status.Id; got != quickAutonomousDatabaseOCID {
		t.Logf("Delete(list lookup %q/%q) status.id = %q, want %q", compartmentID, displayName, got, quickAutonomousDatabaseOCID)
		return false
	}
	if got := resource.Status.CompartmentId; got != compartmentID {
		t.Logf("Delete(list lookup %q/%q) status.compartmentId = %q, want %q", compartmentID, displayName, got, compartmentID)
		return false
	}
	if got := resource.Status.DisplayName; got != displayName {
		t.Logf("Delete(list lookup %q/%q) status.displayName = %q, want %q", compartmentID, displayName, got, displayName)
		return false
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Logf("Delete(list lookup %q/%q) should set deletedAt", compartmentID, displayName)
		return false
	}

	return true
}

func quickAutonomousDatabaseRuntimeConfig(t *testing.T) generatedruntime.Config[*databasev1beta1.AutonomousDatabase] {
	t.Helper()

	client, ok := newAutonomousDatabaseServiceClient(&AutonomousDatabaseServiceManager{
		Provider: common.NewRawConfigurationProvider("", "", "", "", "", nil),
	}).(defaultAutonomousDatabaseServiceClient)
	if !ok {
		t.Fatalf("newAutonomousDatabaseServiceClient() returned %T, want %T", client, defaultAutonomousDatabaseServiceClient{})
	}

	cfg := cloneGeneratedRuntimeConfig(t, client.ServiceClient)
	cfg.InitError = nil
	if cfg.Semantics == nil {
		t.Fatal("AutonomousDatabase generated runtime should define formal semantics")
	}
	if cfg.Semantics.FormalService != "database" {
		t.Fatalf("formal service = %q, want %q", cfg.Semantics.FormalService, "database")
	}
	if cfg.Semantics.FormalSlug != "databaseautonomousdatabase" {
		t.Fatalf("formal slug = %q, want %q", cfg.Semantics.FormalSlug, "databaseautonomousdatabase")
	}
	if cfg.Semantics.Delete.Policy != "required" {
		t.Fatalf("delete policy = %q, want %q", cfg.Semantics.Delete.Policy, "required")
	}
	if cfg.Semantics.List == nil {
		t.Fatal("AutonomousDatabase list semantics should not be nil")
	}
	if got := cfg.Semantics.List.ResponseItemsField; got != "Items" {
		t.Fatalf("list response items field = %q, want %q", got, "Items")
	}
	if got := cfg.Semantics.List.MatchFields; !reflect.DeepEqual(got, []string{"compartmentId", "displayName"}) {
		t.Fatalf("list match fields = %v, want %v", got, []string{"compartmentId", "displayName"})
	}

	return cfg
}

// generatedruntime.ServiceClient keeps its config unexported; clone it so these
// tests stay pinned to the checked-in AutonomousDatabase generator output.
func cloneGeneratedRuntimeConfig[T any](t *testing.T, client generatedruntime.ServiceClient[T]) generatedruntime.Config[T] {
	t.Helper()

	value := reflect.ValueOf(&client).Elem().FieldByName("config")
	if !value.IsValid() || !value.CanAddr() {
		t.Fatal("generated runtime config field was not addressable")
	}

	return *(*generatedruntime.Config[T])(unsafe.Pointer(value.UnsafeAddr()))
}

func newQuickAutonomousDatabaseCreateClient(
	t *testing.T,
	baseCfg generatedruntime.Config[*databasev1beta1.AutonomousDatabase],
	lifecycleState string,
) generatedruntime.ServiceClient[*databasev1beta1.AutonomousDatabase] {
	t.Helper()

	cfg := baseCfg
	cfg.Create = &generatedruntime.Operation{
		NewRequest: func() any { return &quickAutonomousDatabaseCreateRequest{} },
		Fields:     baseCfg.Create.Fields,
		Call: func(_ context.Context, _ any) (any, error) {
			return quickAutonomousDatabaseCreateResponse{
				AutonomousDatabase: quickAutonomousDatabaseState{
					Id:             quickAutonomousDatabaseOCID,
					LifecycleState: lifecycleState,
					DisplayName:    "quick-created",
				},
			}, nil
		},
	}
	cfg.Get = &generatedruntime.Operation{
		NewRequest: func() any { return &quickAutonomousDatabaseGetRequest{} },
		Fields:     baseCfg.Get.Fields,
		Call: func(_ context.Context, request any) (any, error) {
			req := request.(*quickAutonomousDatabaseGetRequest)
			if req.AutonomousDatabaseId == nil || *req.AutonomousDatabaseId != quickAutonomousDatabaseOCID {
				return nil, fmt.Errorf("get request id = %v, want %q", req.AutonomousDatabaseId, quickAutonomousDatabaseOCID)
			}

			return quickAutonomousDatabaseGetResponse{
				AutonomousDatabase: quickAutonomousDatabaseState{
					Id:               quickAutonomousDatabaseOCID,
					LifecycleState:   lifecycleState,
					LifecycleDetails: "quick create follow-up",
				},
			}, nil
		},
	}
	cfg.List = nil
	cfg.Update = nil
	cfg.Delete = nil

	return generatedruntime.NewServiceClient[*databasev1beta1.AutonomousDatabase](cfg)
}

func newQuickAutonomousDatabaseDeleteClient(
	t *testing.T,
	baseCfg generatedruntime.Config[*databasev1beta1.AutonomousDatabase],
	lifecycleState string,
) generatedruntime.ServiceClient[*databasev1beta1.AutonomousDatabase] {
	t.Helper()

	cfg := baseCfg
	cfg.Create = nil
	cfg.Update = nil
	cfg.Get = &generatedruntime.Operation{
		NewRequest: func() any { return &quickAutonomousDatabaseGetRequest{} },
		Fields:     baseCfg.Get.Fields,
		Call: func(_ context.Context, request any) (any, error) {
			req := request.(*quickAutonomousDatabaseGetRequest)
			if req.AutonomousDatabaseId == nil || *req.AutonomousDatabaseId != quickAutonomousDatabaseOCID {
				return nil, fmt.Errorf("get request id = %v, want %q", req.AutonomousDatabaseId, quickAutonomousDatabaseOCID)
			}

			return quickAutonomousDatabaseGetResponse{
				AutonomousDatabase: quickAutonomousDatabaseState{
					Id:               quickAutonomousDatabaseOCID,
					LifecycleState:   lifecycleState,
					LifecycleDetails: "quick delete confirmation",
				},
			}, nil
		},
	}
	cfg.List = nil
	cfg.Delete = &generatedruntime.Operation{
		NewRequest: func() any { return &quickAutonomousDatabaseDeleteRequest{} },
		Fields:     baseCfg.Delete.Fields,
		Call: func(_ context.Context, request any) (any, error) {
			req := request.(*quickAutonomousDatabaseDeleteRequest)
			if req.AutonomousDatabaseId == nil || *req.AutonomousDatabaseId != quickAutonomousDatabaseOCID {
				return nil, fmt.Errorf("delete request id = %v, want %q", req.AutonomousDatabaseId, quickAutonomousDatabaseOCID)
			}
			return struct{}{}, nil
		},
	}

	return generatedruntime.NewServiceClient[*databasev1beta1.AutonomousDatabase](cfg)
}

func newQuickAutonomousDatabaseListDeleteClient(
	t *testing.T,
	baseCfg generatedruntime.Config[*databasev1beta1.AutonomousDatabase],
	compartmentID string,
	displayName string,
	deletedID *string,
	listCalls *int,
) generatedruntime.ServiceClient[*databasev1beta1.AutonomousDatabase] {
	t.Helper()

	cfg := baseCfg
	cfg.Create = nil
	cfg.Update = nil
	cfg.Get = nil
	cfg.List = &generatedruntime.Operation{
		NewRequest: func() any { return &quickAutonomousDatabaseListRequest{} },
		Fields:     baseCfg.List.Fields,
		Call: func(_ context.Context, request any) (any, error) {
			req := request.(*quickAutonomousDatabaseListRequest)
			if req.CompartmentId != compartmentID {
				return nil, fmt.Errorf("list request compartmentId = %q, want %q", req.CompartmentId, compartmentID)
			}
			if req.DisplayName != displayName {
				return nil, fmt.Errorf("list request displayName = %q, want %q", req.DisplayName, displayName)
			}

			*listCalls = *listCalls + 1
			if *listCalls == 1 {
				return quickAutonomousDatabaseListResponse{
					Collection: quickAutonomousDatabaseCollection{
						Items: []quickAutonomousDatabaseState{
							{
								Id:            "ocid1.autonomousdatabase.oc1..distractor-display",
								CompartmentId: compartmentID,
								DisplayName:   displayName + "-other",
							},
							{
								Id:             quickAutonomousDatabaseOCID,
								CompartmentId:  compartmentID,
								DisplayName:    displayName,
								LifecycleState: "AVAILABLE",
							},
							{
								Id:            "ocid1.autonomousdatabase.oc1..distractor-compartment",
								CompartmentId: compartmentID + "-other",
								DisplayName:   displayName,
							},
						},
					},
				}, nil
			}

			return quickAutonomousDatabaseListResponse{
				Collection: quickAutonomousDatabaseCollection{},
			}, nil
		},
	}
	cfg.Delete = &generatedruntime.Operation{
		NewRequest: func() any { return &quickAutonomousDatabaseDeleteRequest{} },
		Fields:     baseCfg.Delete.Fields,
		Call: func(_ context.Context, request any) (any, error) {
			req := request.(*quickAutonomousDatabaseDeleteRequest)
			if req.AutonomousDatabaseId == nil {
				return nil, fmt.Errorf("delete request id was nil")
			}
			*deletedID = *req.AutonomousDatabaseId
			return struct{}{}, nil
		},
	}

	return generatedruntime.NewServiceClient[*databasev1beta1.AutonomousDatabase](cfg)
}

func quickAutonomousDatabaseLastCondition(resource *databasev1beta1.AutonomousDatabase) shared.OSOKConditionType {
	if len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}

	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}

func pickQuickState(states []string, index uint8) string {
	return states[int(index)%len(states)]
}

func quickAutonomousDatabaseCaseMask(text string, mask uint64) string {
	var builder strings.Builder
	builder.Grow(len(text))

	for i, r := range text {
		if !unicode.IsLetter(r) {
			builder.WriteRune(r)
			continue
		}
		if mask&(1<<uint(i%64)) != 0 {
			builder.WriteRune(unicode.ToLower(r))
			continue
		}
		builder.WriteRune(unicode.ToUpper(r))
	}

	return builder.String()
}

func quickAutonomousDatabaseDisplayName(raw string) string {
	var builder strings.Builder
	for _, r := range raw {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
		case r == '-', r == '_':
			builder.WriteRune(r)
		}
		if builder.Len() >= 24 {
			break
		}
	}

	if builder.Len() == 0 {
		return "adb-quick"
	}
	return builder.String()
}

func quickAutonomousDatabaseLifecycleCondition(semantics *generatedruntime.Semantics, lifecycleState string) shared.OSOKConditionType {
	state := strings.ToUpper(lifecycleState)
	switch {
	case quickAutonomousDatabaseContains(semantics.Lifecycle.ProvisioningStates, state):
		return shared.Provisioning
	case quickAutonomousDatabaseContains(semantics.Lifecycle.UpdatingStates, state):
		return shared.Updating
	case quickAutonomousDatabaseContains(semantics.Lifecycle.ActiveStates, state):
		return shared.Active
	default:
		return shared.Failed
	}
}

func quickAutonomousDatabaseContains(states []string, state string) bool {
	for _, candidate := range states {
		if candidate == state {
			return true
		}
	}
	return false
}
