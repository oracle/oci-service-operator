/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	databasesdk "github.com/oracle/oci-go-sdk/v65/database"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeResource struct {
	Name      string     `json:"-"`
	Namespace string     `json:"-"`
	UID       string     `json:"-"`
	Spec      fakeSpec   `json:"spec,omitempty"`
	Status    fakeStatus `json:"status,omitempty"`
}

type fakeSpec struct {
	Id                   string                `json:"Id,omitempty"`
	CompartmentId        string                `json:"compartmentId,omitempty"`
	DisplayName          string                `json:"displayName,omitempty"`
	Name                 string                `json:"name,omitempty"`
	FreeformTags         map[string]string     `json:"freeformTags,omitempty"`
	ShapeConfig          *fakeShapeConfig      `json:"shapeConfig,omitempty"`
	AdminUsername        shared.UsernameSource `json:"adminUsername,omitempty"`
	AdminPassword        shared.PasswordSource `json:"adminPassword,omitempty"`
	DataStorageSizeInGBs int                   `json:"dataStorageSizeInGBs,omitempty"`
	Partitions           int                   `json:"partitions,omitempty"`
	RetentionInHours     int                   `json:"retentionInHours,omitempty"`
	Enabled              bool                  `json:"enabled,omitempty"`
}

type fakeStatus struct {
	OsokStatus           shared.OSOKStatus     `json:"status"`
	Id                   string                `json:"id,omitempty"`
	CompartmentId        string                `json:"compartmentId,omitempty"`
	Name                 string                `json:"name,omitempty"`
	DisplayName          string                `json:"displayName,omitempty"`
	FreeformTags         map[string]string     `json:"freeformTags,omitempty"`
	ShapeConfig          *fakeShapeConfig      `json:"shapeConfig,omitempty"`
	AdminUsername        shared.UsernameSource `json:"adminUsername,omitempty"`
	AdminPassword        shared.PasswordSource `json:"adminPassword,omitempty"`
	DataStorageSizeInGBs int                   `json:"dataStorageSizeInGBs,omitempty"`
	Partitions           int                   `json:"partitions,omitempty"`
	RetentionInHours     int                   `json:"retentionInHours,omitempty"`
	LifecycleState       string                `json:"lifecycleState,omitempty"`
}

type fakeShapeConfig struct {
	Ocpus       int `json:"ocpus,omitempty"`
	MemoryInGBs int `json:"memoryInGBs,omitempty"`
	Vcpus       int `json:"vcpus,omitempty"`
}

type fakeThing struct {
	Id                   string            `json:"id,omitempty"`
	CompartmentId        string            `json:"compartmentId,omitempty"`
	DisplayName          string            `json:"displayName,omitempty"`
	Name                 string            `json:"name,omitempty"`
	FreeformTags         map[string]string `json:"freeformTags,omitempty"`
	ShapeConfig          *fakeShapeConfig  `json:"shapeConfig,omitempty"`
	DataStorageSizeInGBs int               `json:"dataStorageSizeInGBs,omitempty"`
	Partitions           int               `json:"partitions,omitempty"`
	RetentionInHours     int               `json:"retentionInHours,omitempty"`
	LifecycleState       string            `json:"lifecycleState,omitempty"`
}

type fakeThingSummary struct {
	Id             string `json:"id,omitempty"`
	CompartmentId  string `json:"compartmentId,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	Name           string `json:"name,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
}

type FakeCreateThingDetails struct {
	CompartmentId string `json:"compartmentId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	Enabled       bool   `json:"enabled,omitempty"`
}

type fakeCreateThingRequest struct {
	OpcRetryToken          *string `contributesTo:"header" name:"opc-retry-token"`
	FakeCreateThingDetails `contributesTo:"body"`
}

type fakeCreateThingResponse struct {
	Thing fakeThing `presentIn:"body"`
}

type FakeCreateThingWithSecretDetails struct {
	DisplayName   string `json:"displayName,omitempty"`
	AdminUsername string `json:"adminUsername,omitempty"`
	AdminPassword string `json:"adminPassword,omitempty"`
}

type fakeCreateThingWithSecretRequest struct {
	FakeCreateThingWithSecretDetails `contributesTo:"body"`
}

type fakeGetThingRequest struct {
	ThingId *string `contributesTo:"path" name:"thingId"`
}

type fakeGetThingResponse struct {
	Thing fakeThing `presentIn:"body"`
}

type FakeUpdateThingDetails struct {
	DisplayName          string            `json:"displayName,omitempty"`
	Enabled              bool              `json:"enabled,omitempty"`
	DataStorageSizeInGBs int               `json:"dataStorageSizeInGBs,omitempty"`
	FreeformTags         map[string]string `json:"freeformTags,omitempty"`
	ShapeConfig          *fakeShapeConfig  `json:"shapeConfig,omitempty"`
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
	CompartmentId string `contributesTo:"query" name:"compartmentId"`
	DisplayName   string `contributesTo:"query" name:"displayName"`
	Id            string `contributesTo:"query" name:"id"`
	Name          string `contributesTo:"query" name:"name"`
}

type fakeThingCollection struct {
	Items []fakeThingSummary `json:"items,omitempty"`
}

type fakeListThingResponse struct {
	Collection fakeThingCollection `presentIn:"body"`
}

type fakeNamedThingCollection struct {
	Resources []fakeThingSummary `json:"resources,omitempty"`
}

type fakeNamedListThingResponse struct {
	Collection fakeNamedThingCollection `presentIn:"body"`
}

type fakeCredentialClient struct {
	secrets    map[string]map[string][]byte
	getCalls   []string
	namespaces []string
}

func (f *fakeCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(_ context.Context, name, namespace string) (map[string][]byte, error) {
	f.getCalls = append(f.getCalls, name)
	f.namespaces = append(f.namespaces, namespace)
	secret, ok := f.secrets[name]
	if !ok {
		return nil, nil
	}
	return secret, nil
}

func (f *fakeCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
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
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, false, "CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	requireCreateThingRequestMatchesSpec(t, createRequest, resource.Spec)
	requireThingIDRequest(t, "get", getRequest.ThingId, "ocid1.thing.oc1..create")
	requireStatusOCID(t, resource, "ocid1.thing.oc1..create")
	requireStringEqual(t, "status.id", resource.Status.Id, "ocid1.thing.oc1..create")
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "created-name")
	requireStringEqual(t, "status.lifecycleState", resource.Status.LifecycleState, "ACTIVE")
	requireCreatedAt(t, resource)
	requireTrailingCondition(t, resource, shared.Active)
}

func TestApplySuccessSetsLifecycleAsyncTrackerWhilePending(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				UpdatingStates: []string{"UPDATING"},
				ActiveStates:   []string{"ACTIVE"},
			},
		},
	})

	resource := &fakeResource{}
	response, err := client.applySuccess(resource, fakeGetThingResponse{
		Thing: fakeThing{
			Id:             "ocid1.thing.oc1..pending",
			DisplayName:    "pending-thing",
			LifecycleState: "UPDATING",
		},
	}, shared.Updating)

	if err != nil {
		t.Fatalf("applySuccess() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("response.ShouldRequeue = false, want true")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatalf("status.async.current = nil, want lifecycle tracker")
	}
	if resource.Status.OsokStatus.Async.Current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", resource.Status.OsokStatus.Async.Current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseUpdate)
	}
	if resource.Status.OsokStatus.Async.Current.RawStatus != "UPDATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "UPDATING")
	}
	if resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", resource.Status.OsokStatus.Async.Current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func TestApplySuccessClearsLifecycleAsyncTrackerWhenActive(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				UpdatingStates: []string{"UPDATING"},
				ActiveStates:   []string{"ACTIVE"},
			},
		},
	})

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{
				Async: shared.OSOKAsyncTracker{
					Current: &shared.OSOKAsyncOperation{
						Source:          shared.OSOKAsyncSourceLifecycle,
						Phase:           shared.OSOKAsyncPhaseUpdate,
						RawStatus:       "UPDATING",
						NormalizedClass: shared.OSOKAsyncClassPending,
					},
				},
			},
		},
	}

	response, err := client.applySuccess(resource, fakeGetThingResponse{
		Thing: fakeThing{
			Id:             "ocid1.thing.oc1..active",
			DisplayName:    "active-thing",
			LifecycleState: "ACTIVE",
		},
	}, shared.Active)

	if err != nil {
		t.Fatalf("applySuccess() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatalf("response.ShouldRequeue = true, want false")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Active)
	}
}

func TestApplySuccessPrefersObservedLifecyclePhaseTransition(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				UpdatingStates: []string{"UPDATING"},
				ActiveStates:   []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				PendingStates: []string{"DELETING"},
			},
		},
	})

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{
				Async: shared.OSOKAsyncTracker{
					Current: &shared.OSOKAsyncOperation{
						Source:          shared.OSOKAsyncSourceLifecycle,
						Phase:           shared.OSOKAsyncPhaseUpdate,
						RawStatus:       "UPDATING",
						NormalizedClass: shared.OSOKAsyncClassPending,
					},
				},
			},
		},
	}

	response, err := client.applySuccess(resource, fakeGetThingResponse{
		Thing: fakeThing{
			Id:             "ocid1.thing.oc1..deleting",
			DisplayName:    "deleting-thing",
			LifecycleState: "DELETING",
		},
	}, shared.Updating)

	if err != nil {
		t.Fatalf("applySuccess() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("response.ShouldRequeue = false, want true")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatalf("status.async.current = nil, want lifecycle tracker")
	}
	if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
	}
	if resource.Status.OsokStatus.Async.Current.RawStatus != "DELETING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "DELETING")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
}

func TestApplySuccessHeuristicDeleteTerminalKeepsAsyncTrackerWithoutSemantics(t *testing.T) {
	t.Parallel()

	for _, lifecycleState := range []string{"DELETED", "TERMINATED"} {
		lifecycleState := lifecycleState
		t.Run(lifecycleState, func(t *testing.T) {
			t.Parallel()

			client := NewServiceClient[*fakeResource](Config[*fakeResource]{
				Kind:    "Thing",
				SDKName: "Thing",
			})

			resource := &fakeResource{
				Status: fakeStatus{
					OsokStatus: shared.OSOKStatus{
						Async: shared.OSOKAsyncTracker{
							Current: &shared.OSOKAsyncOperation{
								Source:          shared.OSOKAsyncSourceLifecycle,
								Phase:           shared.OSOKAsyncPhaseUpdate,
								RawStatus:       "UPDATING",
								NormalizedClass: shared.OSOKAsyncClassPending,
							},
						},
					},
				},
			}

			response, err := client.applySuccess(resource, fakeGetThingResponse{
				Thing: fakeThing{
					Id:             "ocid1.thing.oc1..deleted",
					DisplayName:    "deleted-thing",
					LifecycleState: lifecycleState,
				},
			}, shared.Active)
			if err != nil {
				t.Fatalf("applySuccess() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatalf("response.IsSuccessful = false, want true")
			}
			if !response.ShouldRequeue {
				t.Fatalf("response.ShouldRequeue = false, want true")
			}
			if resource.Status.OsokStatus.Async.Current == nil {
				t.Fatalf("status.async.current = nil, want delete tracker")
			}
			if resource.Status.OsokStatus.Async.Current.Source != shared.OSOKAsyncSourceLifecycle {
				t.Fatalf("status.async.current.source = %q, want %q", resource.Status.OsokStatus.Async.Current.Source, shared.OSOKAsyncSourceLifecycle)
			}
			if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
				t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
			}
			if resource.Status.OsokStatus.Async.Current.RawStatus != lifecycleState {
				t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, lifecycleState)
			}
			if resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
				t.Fatalf("status.async.current.normalizedClass = %q, want %q", resource.Status.OsokStatus.Async.Current.NormalizedClass, shared.OSOKAsyncClassSucceeded)
			}
			if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
				t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
			}
		})
	}
}

func TestApplySuccessFormalDeleteTerminalKeepsAsyncTrackerWithSemantics(t *testing.T) {
	t.Parallel()

	for _, lifecycleState := range []string{"DELETED", "TERMINATED"} {
		lifecycleState := lifecycleState
		t.Run(lifecycleState, func(t *testing.T) {
			t.Parallel()

			client := NewServiceClient[*fakeResource](Config[*fakeResource]{
				Kind:    "Thing",
				SDKName: "Thing",
				Semantics: &Semantics{
					Lifecycle: LifecycleSemantics{
						UpdatingStates: []string{"UPDATING"},
						ActiveStates:   []string{"ACTIVE"},
					},
					Delete: DeleteSemantics{
						TerminalStates: []string{"DELETED", "TERMINATED"},
					},
				},
			})

			resource := &fakeResource{
				Status: fakeStatus{
					OsokStatus: shared.OSOKStatus{
						Async: shared.OSOKAsyncTracker{
							Current: &shared.OSOKAsyncOperation{
								Source:          shared.OSOKAsyncSourceLifecycle,
								Phase:           shared.OSOKAsyncPhaseUpdate,
								RawStatus:       "UPDATING",
								NormalizedClass: shared.OSOKAsyncClassPending,
							},
						},
					},
				},
			}

			response, err := client.applySuccess(resource, fakeGetThingResponse{
				Thing: fakeThing{
					Id:             "ocid1.thing.oc1..deleted",
					DisplayName:    "deleted-thing",
					LifecycleState: lifecycleState,
				},
			}, shared.Active)
			if err != nil {
				t.Fatalf("applySuccess() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatalf("response.IsSuccessful = false, want true")
			}
			if !response.ShouldRequeue {
				t.Fatalf("response.ShouldRequeue = false, want true")
			}
			if resource.Status.OsokStatus.Async.Current == nil {
				t.Fatalf("status.async.current = nil, want delete tracker")
			}
			if resource.Status.OsokStatus.Async.Current.Source != shared.OSOKAsyncSourceLifecycle {
				t.Fatalf("status.async.current.source = %q, want %q", resource.Status.OsokStatus.Async.Current.Source, shared.OSOKAsyncSourceLifecycle)
			}
			if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
				t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
			}
			if resource.Status.OsokStatus.Async.Current.RawStatus != lifecycleState {
				t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, lifecycleState)
			}
			if resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
				t.Fatalf("status.async.current.normalizedClass = %q, want %q", resource.Status.OsokStatus.Async.Current.NormalizedClass, shared.OSOKAsyncClassSucceeded)
			}
			if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
				t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
			}
		})
	}
}

func TestServiceClientHasMutableDriftDetectsTagAddition(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				Mutable: []string{"freeformTags"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			FreeformTags: map[string]string{"scenario": "e2e"},
		},
	}
	current := fakeGetThingResponse{
		Thing: fakeThing{
			Id:             "ocid1.thing.oc1..existing",
			LifecycleState: "ACTIVE",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				Mutable: []string{"freeformTags"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			FreeformTags: map[string]string{"scenario": "e2e"},
		},
	}
	current := fakeGetThingResponse{
		Thing: fakeThing{
			Id:             "ocid1.thing.oc1..existing",
			FreeformTags:   map[string]string{"scenario": "e2e"},
			LifecycleState: "ACTIVE",
		},
	}

	drift, err := client.hasMutableDrift(resource, current)
	if err != nil {
		t.Fatalf("hasMutableDrift() error = %v", err)
	}
	if drift {
		t.Fatal("hasMutableDrift() = true, want false when mutable tags already match")
	}
}

func TestServiceClientCreateOrUpdateResolvesSecretBackedBodyFields(t *testing.T) {
	t.Parallel()

	var createRequest fakeCreateThingWithSecretRequest
	credClient := &fakeCredentialClient{
		secrets: map[string]map[string][]byte{
			"admin-user": {
				"username": []byte("dbadmin"),
			},
			"admin-password": {
				"password": []byte("SuperSecret123"),
			},
		},
	}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:             "Thing",
		SDKName:          "Thing",
		CredentialClient: credClient,
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingWithSecretRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*fakeCreateThingWithSecretRequest)
				return fakeCreateThingResponse{
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
		Namespace: "database-system",
		Spec: fakeSpec{
			DisplayName:   "created-name",
			AdminUsername: shared.UsernameSource{Secret: shared.SecretSource{SecretName: "admin-user"}},
			AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-password"}},
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireStringEqual(t, "create request displayName", createRequest.DisplayName, "created-name")
	requireStringEqual(t, "create request adminUsername", createRequest.AdminUsername, "dbadmin")
	requireStringEqual(t, "create request adminPassword", createRequest.AdminPassword, "SuperSecret123")
	requireStringEqual(t, "GetSecret() names", strings.Join(credClient.getCalls, ","), "admin-user,admin-password")
	for _, namespace := range credClient.namespaces {
		requireStringEqual(t, "GetSecret() namespace", namespace, "database-system")
	}
	requireStringEqual(t, "status.adminUsername.secret.secretName", resource.Status.AdminUsername.Secret.SecretName, "admin-user")
	requireStringEqual(t, "status.adminPassword.secret.secretName", resource.Status.AdminPassword.Secret.SecretName, "admin-password")
}

func TestServiceClientCreateOrUpdateFailsWhenSecretKeyIsMissing(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		CredentialClient: &fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-password": {
					"not-password": []byte("missing"),
				},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingWithSecretRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeCreateThingResponse{}, nil
			},
		},
	})

	resource := &fakeResource{
		Namespace: "database-system",
		Spec: fakeSpec{
			AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-password"}},
		},
	}

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), `password key in secret "admin-password" is not found`) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing password key failure", err)
	}
}

func TestGeneratedCredentialSourceFieldsOmitZeroJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    any
		unwanted []string
	}{
		{
			name: "autonomous database spec",
			value: databasev1beta1.AutonomousDatabaseSpec{
				CompartmentId: "ocid1.compartment.oc1..adb",
				DisplayName:   "adb-sample",
			},
			unwanted: []string{`"adminPassword"`},
		},
		{
			name: "mysql dbsystem spec",
			value: mysqlv1beta1.DbSystemSpec{
				CompartmentId: "ocid1.compartment.oc1..mysql",
				ShapeName:     "MySQL.VM.Standard.E4.1.8GB",
				SubnetId:      "ocid1.subnet.oc1..mysql",
			},
			unwanted: []string{`"adminUsername"`, `"adminPassword"`},
		},
		{
			name:     "mysql dbsystem status",
			value:    mysqlv1beta1.DbSystemStatus{},
			unwanted: []string{`"adminUsername"`, `"adminPassword"`},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			for _, token := range tc.unwanted {
				if strings.Contains(string(payload), token) {
					t.Fatalf("json.Marshal() = %s, unexpected token %s", payload, token)
				}
			}
		})
	}
}

//nolint:gocognit,gocyclo // Table-driven coverage keeps the generated polymorphic create-body variants together.
func TestBuildRequestPopulatesAutonomousDatabasePolymorphicCreateBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		spec   databasev1beta1.AutonomousDatabaseSpec
		assert func(*testing.T, databasesdk.CreateAutonomousDatabaseBase)
	}{
		{
			name: "default create details",
			spec: databasev1beta1.AutonomousDatabaseSpec{
				CompartmentId: "ocid1.compartment.oc1..create",
				DisplayName:   "adb-create",
			},
			assert: func(t *testing.T, body databasesdk.CreateAutonomousDatabaseBase) {
				t.Helper()

				details, ok := body.(databasesdk.CreateAutonomousDatabaseDetails)
				if !ok {
					t.Fatalf("create body type = %T, want %T", body, databasesdk.CreateAutonomousDatabaseDetails{})
				}
				if details.CompartmentId == nil || *details.CompartmentId != "ocid1.compartment.oc1..create" {
					t.Fatalf("details.compartmentId = %v, want ocid1.compartment.oc1..create", details.CompartmentId)
				}
				if details.DisplayName == nil || *details.DisplayName != "adb-create" {
					t.Fatalf("details.displayName = %v, want adb-create", details.DisplayName)
				}
			},
		},
		{
			name: "clone details",
			spec: databasev1beta1.AutonomousDatabaseSpec{
				CompartmentId: "ocid1.compartment.oc1..clone",
				DisplayName:   "adb-clone",
				Source:        "DATABASE",
				SourceId:      "ocid1.autonomousdatabase.oc1..source",
			},
			assert: func(t *testing.T, body databasesdk.CreateAutonomousDatabaseBase) {
				t.Helper()

				details, ok := body.(databasesdk.CreateAutonomousDatabaseCloneDetails)
				if !ok {
					t.Fatalf("create body type = %T, want %T", body, databasesdk.CreateAutonomousDatabaseCloneDetails{})
				}
				if details.SourceId == nil || *details.SourceId != "ocid1.autonomousdatabase.oc1..source" {
					t.Fatalf("details.sourceId = %v, want ocid1.autonomousdatabase.oc1..source", details.SourceId)
				}
				if details.DisplayName == nil || *details.DisplayName != "adb-clone" {
					t.Fatalf("details.displayName = %v, want adb-clone", details.DisplayName)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := &databasesdk.CreateAutonomousDatabaseRequest{}
			resource := &databasev1beta1.AutonomousDatabase{Spec: tc.spec}
			values, err := lookupValues(resource)
			if err != nil {
				t.Fatalf("lookupValues() error = %v", err)
			}

			err = buildRequest(
				request,
				resource,
				values,
				"",
				[]RequestField{{FieldName: "CreateAutonomousDatabaseDetails", RequestName: "createAutonomousDatabaseDetails", Contribution: "body"}},
				nil,
				requestBuildOptions{Context: context.Background()},
				nil,
				false,
			)
			if err != nil {
				t.Fatalf("buildRequest() error = %v", err)
			}
			if request.CreateAutonomousDatabaseDetails == nil {
				t.Fatal("buildRequest() should populate CreateAutonomousDatabaseDetails")
			}

			tc.assert(t, request.CreateAutonomousDatabaseDetails)
		})
	}
}

func TestBuildRequestPopulatesLaunchInstancePolymorphicSourceDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		spec   corev1beta1.InstanceSpec
		assert func(*testing.T, coresdk.LaunchInstanceDetails)
	}{
		{
			name: "image source details",
			spec: corev1beta1.InstanceSpec{
				CompartmentId:      "ocid1.compartment.oc1..instance",
				AvailabilityDomain: "AD-1",
				SubnetId:           "ocid1.subnet.oc1..instance",
				DisplayName:        "instance-sample",
				Shape:              "VM.Standard.E4.Flex",
				SourceDetails: corev1beta1.InstanceSourceDetails{
					SourceType:          "image",
					ImageId:             "ocid1.image.oc1..image",
					BootVolumeSizeInGBs: 50,
				},
			},
			assert: func(t *testing.T, details coresdk.LaunchInstanceDetails) {
				t.Helper()

				if details.CompartmentId == nil || *details.CompartmentId != "ocid1.compartment.oc1..instance" {
					t.Fatalf("details.compartmentId = %v, want ocid1.compartment.oc1..instance", details.CompartmentId)
				}
				if details.AvailabilityDomain == nil || *details.AvailabilityDomain != "AD-1" {
					t.Fatalf("details.availabilityDomain = %v, want AD-1", details.AvailabilityDomain)
				}
				source, ok := details.SourceDetails.(coresdk.InstanceSourceViaImageDetails)
				if !ok {
					t.Fatalf("details.sourceDetails type = %T, want %T", details.SourceDetails, coresdk.InstanceSourceViaImageDetails{})
				}
				if source.ImageId == nil || *source.ImageId != "ocid1.image.oc1..image" {
					t.Fatalf("image source imageId = %v, want ocid1.image.oc1..image", source.ImageId)
				}
				if source.BootVolumeSizeInGBs == nil || *source.BootVolumeSizeInGBs != 50 {
					t.Fatalf("image source bootVolumeSizeInGBs = %v, want 50", source.BootVolumeSizeInGBs)
				}
			},
		},
		{
			name: "boot volume source details",
			spec: corev1beta1.InstanceSpec{
				CompartmentId:      "ocid1.compartment.oc1..boot",
				AvailabilityDomain: "AD-2",
				Shape:              "VM.Standard.E4.Flex",
				SourceDetails: corev1beta1.InstanceSourceDetails{
					SourceType:   "bootVolume",
					BootVolumeId: "ocid1.bootvolume.oc1..boot",
				},
			},
			assert: func(t *testing.T, details coresdk.LaunchInstanceDetails) {
				t.Helper()

				source, ok := details.SourceDetails.(coresdk.InstanceSourceViaBootVolumeDetails)
				if !ok {
					t.Fatalf("details.sourceDetails type = %T, want %T", details.SourceDetails, coresdk.InstanceSourceViaBootVolumeDetails{})
				}
				if source.BootVolumeId == nil || *source.BootVolumeId != "ocid1.bootvolume.oc1..boot" {
					t.Fatalf("boot volume source bootVolumeId = %v, want ocid1.bootvolume.oc1..boot", source.BootVolumeId)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := &coresdk.LaunchInstanceRequest{}
			resource := &corev1beta1.Instance{Spec: tc.spec}
			values, err := lookupValues(resource)
			if err != nil {
				t.Fatalf("lookupValues() error = %v", err)
			}

			err = buildRequest(
				request,
				resource,
				values,
				"",
				[]RequestField{{FieldName: "LaunchInstanceDetails", RequestName: "LaunchInstanceDetails", Contribution: "body"}},
				nil,
				requestBuildOptions{Context: context.Background()},
				nil,
				false,
			)
			if err != nil {
				t.Fatalf("buildRequest() error = %v", err)
			}

			tc.assert(t, request.LaunchInstanceDetails)
		})
	}
}

func TestBuildRequestOmitsUnsetGeneratedAdminCredentialSources(t *testing.T) {
	t.Parallel()

	mysqlRequest := &mysqlsdk.CreateDbSystemRequest{}
	mysqlResource := &mysqlv1beta1.DbSystem{
		Spec: mysqlv1beta1.DbSystemSpec{
			CompartmentId: "ocid1.compartment.oc1..mysql",
			ShapeName:     "MySQL.VM.Standard.E4.1.8GB",
			SubnetId:      "ocid1.subnet.oc1..mysql",
		},
	}
	mysqlValues, err := lookupValues(mysqlResource)
	if err != nil {
		t.Fatalf("lookupValues(mysql) error = %v", err)
	}

	if err := buildRequest(mysqlRequest, mysqlResource, mysqlValues, "", nil, nil, requestBuildOptions{Context: context.Background()}, nil, false); err != nil {
		t.Fatalf("buildRequest(mysql) error = %v", err)
	}
	if mysqlRequest.AdminUsername != nil {
		t.Fatalf("mysql adminUsername = %v, want nil when secret source is omitted", mysqlRequest.AdminUsername)
	}
	if mysqlRequest.AdminPassword != nil {
		t.Fatalf("mysql adminPassword = %v, want nil when secret source is omitted", mysqlRequest.AdminPassword)
	}

	adbRequest := &databasesdk.CreateAutonomousDatabaseRequest{}
	adbResource := &databasev1beta1.AutonomousDatabase{
		Spec: databasev1beta1.AutonomousDatabaseSpec{
			CompartmentId: "ocid1.compartment.oc1..adb",
			DisplayName:   "adb-sample",
		},
	}
	adbValues, err := lookupValues(adbResource)
	if err != nil {
		t.Fatalf("lookupValues(adb) error = %v", err)
	}

	if err := buildRequest(
		adbRequest,
		adbResource,
		adbValues,
		"",
		[]RequestField{{FieldName: "CreateAutonomousDatabaseDetails", RequestName: "createAutonomousDatabaseDetails", Contribution: "body"}},
		nil,
		requestBuildOptions{Context: context.Background()},
		nil,
		false,
	); err != nil {
		t.Fatalf("buildRequest(adb) error = %v", err)
	}
	if adbRequest.CreateAutonomousDatabaseDetails == nil {
		t.Fatal("buildRequest(adb) should populate CreateAutonomousDatabaseDetails")
	}
	adbDetails, ok := adbRequest.CreateAutonomousDatabaseDetails.(databasesdk.CreateAutonomousDatabaseDetails)
	if !ok {
		t.Fatalf("create body type = %T, want %T", adbRequest.CreateAutonomousDatabaseDetails, databasesdk.CreateAutonomousDatabaseDetails{})
	}
	if adbDetails.AdminPassword != nil {
		t.Fatalf("autonomous database adminPassword = %v, want nil when secret source is omitted", adbDetails.AdminPassword)
	}
}

func TestBuildRequestOmitsZeroValueAutonomousDatabaseNestedStructs(t *testing.T) {
	t.Parallel()

	adbRequest := &databasesdk.CreateAutonomousDatabaseRequest{}
	adbResource := &databasev1beta1.AutonomousDatabase{
		Spec: databasev1beta1.AutonomousDatabaseSpec{
			CompartmentId:        "ocid1.compartment.oc1..adb",
			DisplayName:          "adb-sample",
			DbName:               "adbsample",
			DbWorkload:           "OLTP",
			IsDedicated:          false,
			DbVersion:            "19c",
			DataStorageSizeInTBs: 1,
			CpuCoreCount:         1,
			LicenseModel:         "LICENSE_INCLUDED",
			IsAutoScalingEnabled: true,
		},
	}
	adbValues, err := lookupValues(adbResource)
	if err != nil {
		t.Fatalf("lookupValues(adb) error = %v", err)
	}

	if err := buildRequest(
		adbRequest,
		adbResource,
		adbValues,
		"",
		[]RequestField{{FieldName: "CreateAutonomousDatabaseDetails", RequestName: "createAutonomousDatabaseDetails", Contribution: "body"}},
		nil,
		requestBuildOptions{Context: context.Background()},
		nil,
		false,
	); err != nil {
		t.Fatalf("buildRequest(adb) error = %v", err)
	}
	if adbRequest.CreateAutonomousDatabaseDetails == nil {
		t.Fatal("buildRequest(adb) should populate CreateAutonomousDatabaseDetails")
	}

	httpRequest, err := adbRequest.HTTPRequest("POST", "/autonomousDatabases", nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest(adb create body) error = %v", err)
	}
	payload, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("io.ReadAll(adb create body) error = %v", err)
	}
	body := string(payload)

	for _, unwanted := range []string{`"resourcePoolSummary"`, `"longTermBackupSchedule"`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("adb create body = %s, unexpected token %s", body, unwanted)
		}
	}
	for _, wanted := range []string{`"dbVersion":"19c"`, `"cpuCoreCount":1`, `"dataStorageSizeInTBs":1`, `"isAutoScalingEnabled":true`} {
		if !strings.Contains(body, wanted) {
			t.Fatalf("adb create body = %s, missing token %s", body, wanted)
		}
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

func TestServiceClientCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	t.Parallel()

	getCalled := false

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
					t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "steady-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called when mutable fields already match")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "steady-name",
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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName", "freeformTags"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*fakeUpdateThingRequest)
				return fakeUpdateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "steady-name",
						FreeformTags:   map[string]string{"scenario": "update", "run": "123"},
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:           "ocid1.thing.oc1..existing",
						DisplayName:  "steady-name",
						FreeformTags: map[string]string{"scenario": "create", "run": "123"},
						ShapeConfig: &fakeShapeConfig{
							Ocpus:       1,
							MemoryInGBs: 16,
							Vcpus:       2,
						},
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName:  "steady-name",
			FreeformTags: map[string]string{"scenario": "update", "run": "123"},
			ShapeConfig: &fakeShapeConfig{
				Ocpus:       1,
				MemoryInGBs: 16,
			},
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:         "ocid1.thing.oc1..existing",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Bucket",
		SDKName: "Bucket",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			Name:        "bucket-new",
			DisplayName: "display-new",
		},
	}

	body, ok, err := client.filteredUpdateBody(resource, requestBuildOptions{
		CurrentResponse: fakeGetThingResponse{
			Thing: fakeThing{
				Id:             "ocid1.bucket.oc1..existing",
				Name:           "bucket-old",
				DisplayName:    "display-old",
				LifecycleState: "ACTIVE",
			},
		},
	})
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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
					t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "creating-name",
						LifecycleState: "CREATING",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called while lifecycle is still provisioning")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "desired-name",
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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"adminPassword"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
					t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "steady-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called when a mutable field is write-only in the service response")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			AdminPassword: shared.PasswordSource{
				Secret: shared.SecretSource{SecretName: "admin-password"},
			},
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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
					t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "old-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*fakeUpdateThingRequest)
				return fakeUpdateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "new-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "new-name",
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:         "ocid1.thing.oc1..existing",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"dataStorageSizeInGb"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateCalled = true
				updateRequest = *request.(*fakeUpdateThingRequest)
				return fakeUpdateThingResponse{
					Thing: fakeThing{
						Id:                   "ocid1.thing.oc1..existing",
						DataStorageSizeInGBs: updateRequest.DataStorageSizeInGBs,
						LifecycleState:       "ACTIVE",
					},
				}, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DataStorageSizeInGBs: 20,
		},
		Status: fakeStatus{
			OsokStatus:           shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:                   "ocid1.thing.oc1..existing",
			DataStorageSizeInGBs: 10,
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called when update drift is outside the supported mutable surface")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..new",
			DisplayName:   "same-name",
		},
		Status: fakeStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:            "ocid1.thing.oc1..existing",
			CompartmentId: "ocid1.compartment.oc1..old",
			DisplayName:   "same-name",
		},
	}

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "reject unsupported update drift for compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported update drift failure", err)
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
				return nil, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "thing not found")
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

func TestServiceClientCreateOrUpdateUsesExplicitRequestFieldsAndFormalLifecycle(t *testing.T) {
	t.Parallel()

	var createRequest fakeCreateThingRequest
	var getRequest fakeGetThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			FormalService: "identity",
			FormalSlug:    "user",
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				ActiveStates:       []string{"AVAILABLE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			CreateFollowUp: FollowUpSemantics{
				Strategy: "read-after-write",
				Hooks:    []Hook{{Helper: "tfresource.CreateResource"}},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*fakeCreateThingRequest)
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..create",
						DisplayName:    "created-name",
						LifecycleState: "CREATING",
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
				getRequest = *request.(*fakeGetThingRequest)
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..create",
						DisplayName:    "created-name",
						LifecycleState: "AVAILABLE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
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

func TestServiceClientCreateOrUpdateUsesFormalListMatching(t *testing.T) {
	t.Parallel()

	var listRequest fakeListThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Resources",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				TerminalStates: []string{"DELETED"},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeNamedListThingResponse{
					Collection: fakeNamedThingCollection{
						Resources: []fakeThingSummary{
							{Id: "ocid1.thing.oc1..other", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..other", LifecycleState: "ACTIVE"},
							{Id: "ocid1.thing.oc1..match", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"},
						},
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Compartment",
		SDKName: "Compartment",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				ForceNew: []string{"compartmentId"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createCalled = true
				createRequest := request.(*fakeCreateThingRequest)
				if createRequest.CompartmentId != "ocid1.compartment.oc1..parent" {
					t.Fatalf("create request compartmentId = %q, want parent compartment ID", createRequest.CompartmentId)
				}
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.compartment.oc1..child",
						CompartmentId:  "ocid1.compartment.oc1..parent",
						DisplayName:    "created-name",
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
				getCalled = true
				t.Fatalf("Get() should not be called before a tracked OCID exists, got request=%+v", request)
				return nil, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..parent",
			DisplayName:   "created-name",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Compartment",
		SDKName: "Compartment",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				ActiveStates:       []string{"ACTIVE", "INACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			List: &ListSemantics{
				ResponseItemsField: "Resources",
				MatchFields:        []string{"compartmentId", "lifecycleState", "name"},
			},
			Mutation: MutationSemantics{
				Mutable:  []string{"description", "name"},
				ForceNew: []string{"compartmentId"},
			},
			CreateFollowUp: FollowUpSemantics{
				Strategy: "read-after-write",
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createCalled = true
				createRequest := request.(*fakeCreateThingRequest)
				if createRequest.CompartmentId != "ocid1.compartment.oc1..parent" {
					t.Fatalf("create request compartmentId = %q, want parent compartment ID", createRequest.CompartmentId)
				}
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.compartment.oc1..child",
						CompartmentId:  "ocid1.compartment.oc1..parent",
						Name:           "codex-identity-compartment-20260403083600",
						DisplayName:    "codex-identity-compartment-20260403083600",
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
				getCalled = true
				if !createCalled {
					t.Fatalf("Get() should not be called before Create(), got request=%+v", request)
				}
				getRequest := request.(*fakeGetThingRequest)
				if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.compartment.oc1..child" {
					t.Fatalf("Get() should only follow the created child OCID, got request=%+v", request)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.compartment.oc1..child",
						CompartmentId:  "ocid1.compartment.oc1..parent",
						Name:           "codex-identity-compartment-20260403083600",
						DisplayName:    "codex-identity-compartment-20260403083600",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeNamedListThingResponse{
					Collection: fakeNamedThingCollection{
						Resources: []fakeThingSummary{
							{
								Id:             "ocid1.compartment.oc1..parent",
								CompartmentId:  "ocid1.tenancy.oc1..tenancy",
								Name:           "vdittaka",
								LifecycleState: "ACTIVE",
							},
						},
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..parent",
			Name:          "codex-identity-compartment-20260403083600",
			DisplayName:   "codex-identity-compartment-20260403083600",
		},
	}

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

func TestServiceClientRejectsForceNewMutationChanges(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				ForceNew: []string{"compartmentId"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				t.Fatal("Update() should not be called when a force-new field changes")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..new",
			DisplayName:   "updated-name",
		},
		Status: fakeStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:            "ocid1.thing.oc1..existing",
			CompartmentId: "ocid1.compartment.oc1..old",
		},
	}

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new replacement failure", err)
	}
}

func TestServiceClientRejectsConflictingMutationFields(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				ConflictsWith: map[string][]string{
					"name": {"displayName"},
				},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			Name:        "wanted",
			DisplayName: "conflicting",
		},
	}

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "forbid setting name with displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want conflictsWith failure", err)
	}
}

func TestServiceClientCreateOrUpdateRejectsDocsDeniedNameDrift(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Bucket",
		SDKName: "Bucket",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.bucket.oc1..existing" {
					t.Fatalf("get request thingId = %v, want existing OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.bucket.oc1..existing",
						Name:           "bucket-old",
						DisplayName:    "display-old",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called when Bucket.name drift is outside the conservative mutable surface")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			Name:        "bucket-new",
			DisplayName: "display-old",
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.bucket.oc1..existing"},
			Id:         "ocid1.bucket.oc1..existing",
		},
	}

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "reject unsupported update drift for name") {
		t.Fatalf("CreateOrUpdate() error = %v, want docs-denied name drift failure", err)
	}
}

func TestServiceClientCreateOrUpdateKeepsTrackedCurrentIDWhenPreCreateLookupMisses(t *testing.T) {
	t.Parallel()

	createCalled := false
	listCalled := false

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Bucket",
		SDKName: "Bucket",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when a tracked resource already exists")
				return nil, nil
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listCalled = true
				listRequest := request.(*fakeListThingRequest)
				if listRequest.Name != "bucket-new" {
					t.Fatalf("list request name = %q, want desired spec name", listRequest.Name)
				}
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.bucket.oc1..other",
								Name:           "bucket-old",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
				t.Fatal("Update() should not be called when immutable drift is detected on a tracked resource")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "bucket-new",
			DisplayName:   "steady-name",
		},
		Status: fakeStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.bucket.oc1..existing"},
			Id:            "ocid1.bucket.oc1..existing",
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "bucket-old",
			DisplayName:   "steady-name",
		},
	}

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

func TestServiceClientRejectsOpenFormalGapsAtInit(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Unsupported: []UnsupportedSemantic{
				{Category: "legacy-adapter", StopCondition: "keep manual adapter until gaps close"},
			},
		},
	})

	if _, err := client.CreateOrUpdate(context.Background(), &fakeResource{}, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "open formal gap legacy-adapter") {
		t.Fatalf("CreateOrUpdate() error = %v, want init failure for open formal gap", err)
	}
}

func TestServiceClientDeleteUsesFormalRequiredConfirmation(t *testing.T) {
	t.Parallel()

	var deleteRequest fakeDeleteThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*fakeDeleteThingRequest)
				return fakeDeleteThingResponse{}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..delete",
						LifecycleState: "DELETING",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
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
	if deleted {
		t.Fatal("Delete() should keep waiting while lifecycle is DELETING")
	}
	if deleteRequest.ThingId != nil {
		t.Fatalf("delete request thingId = %v, want no delete request once confirm-delete already reports DELETING", deleteRequest.ThingId)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteConflictStillConfirmsFormalPendingState(t *testing.T) {
	t.Parallel()

	var deleteRequest fakeDeleteThingRequest
	getCalls := 0

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"TERMINATING"},
				TerminalStates: []string{"TERMINATED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*fakeDeleteThingRequest)
				return nil, errortest.NewServiceError(409, "IncorrectState", "delete is still settling")
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				lifecycleState := "ACTIVE"
				if getCalls > 1 {
					lifecycleState = "TERMINATING"
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..delete",
						LifecycleState: lifecycleState,
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
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
	if deleted {
		t.Fatal("Delete() should keep waiting while the confirmed lifecycle remains TERMINATING")
	}
	if getCalls != 2 {
		t.Fatalf("Get() calls = %d, want pre-delete read plus one follow-up after conflict", getCalls)
	}
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..delete" {
		t.Fatalf("delete request thingId = %v, want existing OCID", deleteRequest.ThingId)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty while delete remains pending")
	}
	if resource.Status.LifecycleState != "TERMINATING" {
		t.Fatalf("status.lifecycleState = %q, want TERMINATING", resource.Status.LifecycleState)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteSkipsReissuingDeleteWhenFormalStateAlreadyPending(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	getCalls := 0

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				deleteCalls++
				t.Fatal("Delete() should not be called once delete confirmation already reports DELETING")
				return nil, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				getRequest := request.(*fakeGetThingRequest)
				if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..delete" {
					t.Fatalf("get request thingId = %v, want existing OCID", getRequest.ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..delete",
						LifecycleState: "DELETING",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
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
	if deleted {
		t.Fatal("Delete() should keep waiting while the confirmed lifecycle remains DELETING")
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() calls = %d, want 0 once delete is already pending", deleteCalls)
	}
	if getCalls != 1 {
		t.Fatalf("Get() calls = %d, want 1 confirmation read", getCalls)
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteConflictStillConfirmsFormalTerminalState(t *testing.T) {
	t.Parallel()

	getCalls := 0

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"TERMINATING"},
				TerminalStates: []string{"TERMINATED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, errortest.NewServiceError(409, "IncorrectState", "delete is still settling")
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				lifecycleState := "ACTIVE"
				if getCalls > 1 {
					lifecycleState = "TERMINATED"
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..delete",
						LifecycleState: lifecycleState,
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
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
		t.Fatal("Delete() should succeed once the conflict follow-up confirms TERMINATED")
	}
	if getCalls != 2 {
		t.Fatalf("Get() calls = %d, want pre-delete read plus one follow-up after conflict", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed terminal delete")
	}
	if resource.Status.LifecycleState != "TERMINATED" {
		t.Fatalf("status.lifecycleState = %q, want TERMINATED", resource.Status.LifecycleState)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientCreateOrUpdateUsesUppercaseSpecIDAlias(t *testing.T) {
	t.Parallel()

	var getRequest fakeGetThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest = *request.(*fakeGetThingRequest)
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..bound",
						DisplayName:    "bound-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			Id: "ocid1.thing.oc1..bound",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list lookup finds a reusable resource")
				return nil, nil
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..existing",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						Name:           "wanted",
						CompartmentId:  "ocid1.compartment.oc1..match",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				listCalled = true
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..existing",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
		},
	}

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
				Mutable: []string{"displayName"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list lookup finds a reusable resource")
				return nil, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..existing" {
					t.Fatalf("get request thingId = %v, want reused OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						Name:           "wanted",
						CompartmentId:  "ocid1.compartment.oc1..match",
						DisplayName:    "steady-name",
						LifecycleState: "ACTIVE",
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
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..existing",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
				t.Fatal("Update() should not be called when live mutable state already matches")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
			DisplayName:   "steady-name",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list lookup finds a reusable resource")
				return nil, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				if request.(*fakeGetThingRequest).ThingId == nil {
					t.Fatal("Get() should not be called without a resource OCID")
				}
				return fakeGetThingResponse{}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..existing",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
		},
	}

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

func TestServiceClientCreateOrUpdateReusesDeterministicCreateRetryToken(t *testing.T) {
	t.Parallel()

	var tokens []string

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				token := request.(*fakeCreateThingRequest).OpcRetryToken
				if token == nil || strings.TrimSpace(*token) == "" {
					t.Fatal("create request should set opc retry token")
				}
				tokens = append(tokens, *token)
				return fakeCreateThingResponse{}, nil
			},
		},
	})

	resource := &fakeResource{
		Name:      "thing-sample",
		Namespace: "default",
		UID:       "123e4567-e89b-12d3-a456-426614174000",
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..sample",
			DisplayName:   "sample-name",
		},
	}

	for i := 0; i < 2; i++ {
		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		requireCreateOrUpdateSuccess(t, response, err)
	}

	if len(tokens) != 2 {
		t.Fatalf("create calls = %d, want 2", len(tokens))
	}
	if tokens[0] != resource.UID || tokens[1] != resource.UID {
		t.Fatalf("create retry tokens = %v, want both %q", tokens, resource.UID)
	}
}

func TestServiceClientCreateOrUpdateForcesLiveGetForForceNewValidationAfterListReuse(t *testing.T) {
	t.Parallel()

	createCalled := false
	var getRequest fakeGetThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			Mutation: MutationSemantics{
				ForceNew: []string{"retentionInHours"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when live force-new validation fails")
				return nil, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest = *request.(*fakeGetThingRequest)
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:               "ocid1.thing.oc1..existing",
						Name:             "wanted",
						CompartmentId:    "ocid1.compartment.oc1..match",
						RetentionInHours: 24,
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
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..existing",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId:    "ocid1.compartment.oc1..match",
			Name:             "wanted",
			RetentionInHours: 48,
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				ForceNew: []string{"shapeConfig"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest := *request.(*fakeGetThingRequest)
				if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..existing" {
					t.Fatalf("get request thingId = %v, want reused OCID", getRequest.ThingId)
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id: "ocid1.thing.oc1..existing",
						ShapeConfig: &fakeShapeConfig{
							Ocpus:       1,
							MemoryInGBs: 16,
							Vcpus:       2,
						},
						LifecycleState: "CREATING",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			ShapeConfig: &fakeShapeConfig{
				Ocpus:       1,
				MemoryInGBs: 16,
			},
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:         "ocid1.thing.oc1..existing",
		},
	}

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

	spec := map[string]any{
		"imageId":    "ocid1.image.oc1..example",
		"sourceType": "image",
		"instanceSourceImageFilterDetails": map[string]any{
			"compartmentId": "",
		},
	}
	current := map[string]any{
		"imageId":                          "ocid1.image.oc1..example",
		"sourceType":                       "image",
		"instanceSourceImageFilterDetails": nil,
	}

	if !forceNewValuesEqual(spec, current) {
		t.Fatalf("forceNewValuesEqual() = false, want true when only meaningless nested maps differ")
	}
}

func TestUnsupportedUpdateDriftPathsIgnoresMeaninglessNestedMaps(t *testing.T) {
	t.Parallel()

	spec := map[string]any{
		"displayName": "example",
		"preemptibleInstanceConfig": map[string]any{
			"preemptionAction": map[string]any{
				"jsonData": "",
				"type":     "",
			},
		},
	}
	current := map[string]any{
		"displayName": "example",
	}

	paths := unsupportedUpdateDriftPaths(spec, current, MutationSemantics{
		Mutable: []string{"displayName"},
	})
	if len(paths) != 0 {
		t.Fatalf("unsupportedUpdateDriftPaths() = %v, want no drift for meaningless nested maps", paths)
	}
}

func TestServiceClientCreateOrUpdateCreatesWhenLiveGetMissesAfterListReuse(t *testing.T) {
	t.Parallel()

	createCalled := false
	var createRequest fakeCreateThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			Mutation: MutationSemantics{
				ForceNew: []string{"retentionInHours"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createCalled = true
				createRequest = *request.(*fakeCreateThingRequest)
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						DisplayName:    "created-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, errortest.NewServiceError(404, "NotFound", "thing not found")
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..stale",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId:    "ocid1.compartment.oc1..match",
			Name:             "wanted",
			DisplayName:      "created-name",
			RetentionInHours: 48,
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"id", "name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list fallback rebinds a replacement resource")
				return nil, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..stale" {
					t.Fatalf("get request thingId = %v, want stale tracked OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return nil, errortest.NewServiceError(404, "NotFound", "thing not found")
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..replacement",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
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
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
		},
		Status: fakeStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.thing.oc1..stale"},
			Id:            "ocid1.thing.oc1..stale",
			CompartmentId: "ocid1.compartment.oc1..match",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"id", "name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*fakeCreateThingRequest)
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						CompartmentId:  "ocid1.compartment.oc1..match",
						DisplayName:    "created-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				if request.(*fakeGetThingRequest).ThingId == nil || *request.(*fakeGetThingRequest).ThingId != "ocid1.thing.oc1..stale" {
					t.Fatalf("get request thingId = %v, want stale tracked OCID", request.(*fakeGetThingRequest).ThingId)
				}
				return nil, errortest.NewServiceError(404, "NotFound", "thing not found")
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: nil,
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Id", RequestName: "id", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			DisplayName:   "created-name",
			Name:          "wanted",
		},
		Status: fakeStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.thing.oc1..stale"},
			Id:            "ocid1.thing.oc1..stale",
			CompartmentId: "ocid1.compartment.oc1..match",
		},
	}

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

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createCalled = true
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						DisplayName:    request.(*fakeCreateThingRequest).DisplayName,
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..deleted",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
								LifecycleState: "DELETED",
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
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
			DisplayName:   "created-name",
		},
	}

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

func TestServiceClientRejectsForceNewChangesAgainstLiveOCIState(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				ForceNew: []string{"compartmentId"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						CompartmentId:  "ocid1.compartment.oc1..live",
						DisplayName:    "updated-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called when live force-new validation fails")
				return nil, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..desired",
			DisplayName:   "updated-name",
		},
		Status: fakeStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:            "ocid1.thing.oc1..existing",
			CompartmentId: "ocid1.compartment.oc1..desired",
		},
	}

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want live force-new replacement failure", err)
	}
}

func TestServiceClientDeleteResolvesDeletePhaseListMatchWithoutOcid(t *testing.T) {
	t.Parallel()

	var deleteRequest fakeDeleteThingRequest
	getCalled := false

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "none",
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				if request.(*fakeGetThingRequest).ThingId == nil {
					t.Fatal("Get() should not be called without a resource OCID")
				}
				return fakeGetThingResponse{}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeListThingResponse{
					Collection: fakeThingCollection{
						Items: []fakeThingSummary{
							{
								Id:             "ocid1.thing.oc1..deleting",
								Name:           "wanted",
								CompartmentId:  "ocid1.compartment.oc1..match",
								LifecycleState: "DELETING",
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
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*fakeDeleteThingRequest)
				return fakeDeleteThingResponse{}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			Name:          "wanted",
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should resolve delete-phase list matches without a recorded OCID")
	}
	if getCalled {
		t.Fatal("Get() should be skipped when delete resolution has no recorded OCID")
	}
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..deleting" {
		t.Fatalf("delete request thingId = %v, want delete-phase OCID", deleteRequest.ThingId)
	}
}

func TestValidateFormalSemanticsRejectsWorkRequestHelperWithoutExplicitAsyncContract(t *testing.T) {
	t.Parallel()

	err := validateFormalSemantics("RedisCluster", &Semantics{
		Delete: DeleteSemantics{
			Policy: "best-effort",
		},
		CreateFollowUp: FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []Hook{
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"},
			},
		},
	})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want explicit async helper failure")
	}
	if !strings.Contains(err.Error(), `workrequest helper requires explicit async strategy "workrequest"`) {
		t.Fatalf("validateFormalSemantics() error = %v, want helper/strategy failure", err)
	}
}

func TestValidateFormalSemanticsRejectsLifecycleAsyncWithWorkRequestHelper(t *testing.T) {
	t.Parallel()

	err := validateFormalSemantics("OpensearchCluster", &Semantics{
		Async: &AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Delete: DeleteSemantics{
			Policy: "best-effort",
		},
		CreateFollowUp: FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []Hook{
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"},
			},
		},
	})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want explicit async helper failure")
	}
	if !strings.Contains(err.Error(), `workrequest helper requires explicit async strategy "workrequest"`) {
		t.Fatalf("validateFormalSemantics() error = %v, want helper/strategy failure", err)
	}
}

func TestValidateFormalSemanticsRejectsInvalidAsyncMetadataEnums(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		semantics *Semantics
		wantErr   string
	}{
		{
			name: "unknown strategy",
			semantics: &Semantics{
				Async: &AsyncSemantics{
					Strategy:             "eventual",
					Runtime:              "generatedruntime",
					FormalClassification: "lifecycle",
				},
				Delete: DeleteSemantics{Policy: "best-effort"},
			},
			wantErr: `async.strategy "eventual" must be one of "lifecycle", "workrequest", or "none"`,
		},
		{
			name: "unknown formal classification",
			semantics: &Semantics{
				Async: &AsyncSemantics{
					Strategy:             "lifecycle",
					Runtime:              "generatedruntime",
					FormalClassification: "eventual",
				},
				Delete: DeleteSemantics{Policy: "best-effort"},
			},
			wantErr: `async.formalClassification "eventual" must be one of "lifecycle", "workrequest", or "none"`,
		},
		{
			name: "unknown workrequest source",
			semantics: &Semantics{
				Async: &AsyncSemantics{
					Strategy:             "workrequest",
					Runtime:              "handwritten",
					FormalClassification: "workrequest",
					WorkRequest: &WorkRequestSemantics{
						Source: "custom-source",
						Phases: []string{"create"},
					},
				},
				Delete: DeleteSemantics{Policy: "best-effort"},
			},
			wantErr: `async.workRequest.source "custom-source" must be one of "service-sdk", "workrequests-service", or "provider-helper"`,
		},
		{
			name: "invalid workrequest phase",
			semantics: &Semantics{
				Async: &AsyncSemantics{
					Strategy:             "workrequest",
					Runtime:              "handwritten",
					FormalClassification: "workrequest",
					WorkRequest: &WorkRequestSemantics{
						Source: "service-sdk",
						Phases: []string{"reconcile"},
					},
				},
				Delete: DeleteSemantics{Policy: "best-effort"},
			},
			wantErr: `async.workRequest.phases[0] "reconcile" must be one of "create", "update", or "delete"`,
		},
		{
			name: "duplicate workrequest phase",
			semantics: &Semantics{
				Async: &AsyncSemantics{
					Strategy:             "workrequest",
					Runtime:              "handwritten",
					FormalClassification: "workrequest",
					WorkRequest: &WorkRequestSemantics{
						Source: "service-sdk",
						Phases: []string{"create", "create"},
					},
				},
				Delete: DeleteSemantics{Policy: "best-effort"},
			},
			wantErr: `async.workRequest.phases contains duplicate phase "create"`,
		},
		{
			name: "blank legacy bridge field",
			semantics: &Semantics{
				Async: &AsyncSemantics{
					Strategy:             "workrequest",
					Runtime:              "handwritten",
					FormalClassification: "workrequest",
					WorkRequest: &WorkRequestSemantics{
						Source: "service-sdk",
						Phases: []string{"create"},
						LegacyFieldBridge: &WorkRequestLegacyFieldBridge{
							Update: "   ",
						},
					},
				},
				Delete: DeleteSemantics{Policy: "best-effort"},
			},
			wantErr: `async.workRequest.legacyFieldBridge.update must not be blank`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validateFormalSemantics("Thing", test.semantics)
			if err == nil {
				t.Fatal("validateFormalSemantics() error = nil, want invalid async metadata failure")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateFormalSemantics() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestValidateFormalSemanticsRejectsExplicitHandwrittenAsyncRuntime(t *testing.T) {
	t.Parallel()

	err := validateFormalSemantics("Queue", &Semantics{
		Async: &AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "handwritten",
			FormalClassification: "workrequest",
			WorkRequest: &WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "delete"},
			},
		},
		Delete: DeleteSemantics{
			Policy: "best-effort",
		},
	})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want handwritten runtime failure")
	}
	if !strings.Contains(err.Error(), `generatedruntime cannot honor explicit async runtime "handwritten"`) {
		t.Fatalf("validateFormalSemantics() error = %v, want handwritten-runtime detail", err)
	}
}

func TestValidateFormalSemanticsBlocksAuxiliaryOperations(t *testing.T) {
	t.Parallel()

	err := validateFormalSemantics("Vcn", &Semantics{
		Delete: DeleteSemantics{
			Policy: "best-effort",
		},
		AuxiliaryOperations: []AuxiliaryOperation{
			{Phase: "list", MethodName: "ListVcns"},
		},
	})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want auxiliary-operation failure")
	}
	if !strings.Contains(err.Error(), "unsupported list auxiliary operation ListVcns") {
		t.Fatalf("validateFormalSemantics() error = %v, want auxiliary-operation detail", err)
	}
}

func requireCreateOrUpdateSuccess(t *testing.T, response servicemanager.OSOKResponse, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
}

func requireRequeueState(t *testing.T, response servicemanager.OSOKResponse, want bool, message string) {
	t.Helper()

	if response.ShouldRequeue != want {
		t.Fatal(message)
	}
}

func requireCreateThingRequestMatchesSpec(t *testing.T, request fakeCreateThingRequest, spec fakeSpec) {
	t.Helper()

	requireStringEqual(t, "create request compartmentId", request.CompartmentId, spec.CompartmentId)
	requireStringEqual(t, "create request displayName", request.DisplayName, spec.DisplayName)
	requireTrue(t, request.Enabled, "create request enabled flag was not projected from spec")
}

func requireThingIDRequest(t *testing.T, operation string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s request thingId = %v, want %s", operation, got, want)
	}
}

func requireStatusOCID(t *testing.T, resource *fakeResource, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
}

func requireCreatedAt(t *testing.T, resource *fakeResource) {
	t.Helper()

	if resource.Status.OsokStatus.CreatedAt == nil {
		t.Fatal("status.createdAt should be set after create")
	}
}

func requireTrailingCondition(t *testing.T, resource *fakeResource, want shared.OSOKConditionType) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 || conditions[len(conditions)-1].Type != want {
		t.Fatalf("status conditions = %#v, want trailing %s condition", conditions, want)
	}
}

func requireStringEqual(t *testing.T, fieldName string, got string, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("%s = %q, want %q", fieldName, got, want)
	}
}

func requireTrue(t *testing.T, got bool, message string) {
	t.Helper()

	if !got {
		t.Fatal(message)
	}
}
