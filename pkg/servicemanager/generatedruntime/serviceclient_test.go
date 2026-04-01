/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
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
	CompartmentId        string `json:"compartmentId,omitempty"`
	DisplayName          string `json:"displayName,omitempty"`
	Name                 string `json:"name,omitempty"`
	Enabled              bool   `json:"enabled,omitempty"`
	DataStorageSizeInGBs int    `json:"dataStorageSizeInGBs,omitempty"`
}

type fakeStatus struct {
	OsokStatus           shared.OSOKStatus `json:"status"`
	Id                   string            `json:"id,omitempty"`
	CompartmentId        string            `json:"compartmentId,omitempty"`
	DisplayName          string            `json:"displayName,omitempty"`
	LifecycleState       string            `json:"lifecycleState,omitempty"`
	DataStorageSizeInGBs int               `json:"dataStorageSizeInGBs,omitempty"`
}

type fakeSecretResource struct {
	Name      string           `json:"-"`
	Namespace string           `json:"-"`
	Spec      fakeSecretSpec   `json:"spec,omitempty"`
	Status    fakeSecretStatus `json:"status,omitempty"`
}

type fakeSecretSpec struct {
	DisplayName   string                `json:"displayName,omitempty"`
	AdminUsername shared.UsernameSource `json:"adminUsername,omitempty"`
	AdminPassword shared.PasswordSource `json:"adminPassword,omitempty"`
}

type fakeSecretStatus struct {
	OsokStatus     shared.OSOKStatus     `json:"status"`
	Id             string                `json:"id,omitempty"`
	LifecycleState string                `json:"lifecycleState,omitempty"`
	AdminUsername  shared.UsernameSource `json:"adminUsername,omitempty"`
	AdminPassword  shared.PasswordSource `json:"adminPassword,omitempty"`
}

type fakeThing struct {
	Id                   string `json:"id,omitempty"`
	CompartmentId        string `json:"compartmentId,omitempty"`
	DisplayName          string `json:"displayName,omitempty"`
	Name                 string `json:"name,omitempty"`
	LifecycleState       string `json:"lifecycleState,omitempty"`
	DataStorageSizeInGBs int    `json:"dataStorageSizeInGBs,omitempty"`
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
	DisplayName          string `json:"displayName,omitempty"`
	Enabled              bool   `json:"enabled,omitempty"`
	DataStorageSizeInGBs int    `json:"dataStorageSizeInGBs,omitempty"`
}

type fakeUpdateThingRequest struct {
	ThingId                *string `contributesTo:"path" name:"thingId"`
	FakeUpdateThingDetails `contributesTo:"body"`
}

type fakeUpdateThingResponse struct {
	Thing fakeThing `presentIn:"body"`
}

type FakeCreateSecretThingDetails struct {
	DisplayName   string  `json:"displayName,omitempty"`
	AdminUsername *string `json:"adminUsername,omitempty"`
	AdminPassword *string `json:"adminPassword,omitempty"`
}

type fakeCreateSecretThingRequest struct {
	FakeCreateSecretThingDetails `contributesTo:"body"`
}

type fakeCreateSecretThingResponse struct {
	Thing fakeThing `presentIn:"body"`
}

type fakeDeleteThingRequest struct {
	ThingId *string `contributesTo:"path" name:"thingId"`
}

type fakeDeleteThingResponse struct{}

type fakeListThingRequest struct {
	CompartmentId string `contributesTo:"query" name:"compartmentId"`
	DisplayName   string `contributesTo:"query" name:"displayName"`
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

type fakeCredentialClient struct {
	secrets   map[string]map[string][]byte
	readNames []string
}

var _ credhelper.CredentialClient = (*fakeCredentialClient)(nil)

type quickLifecycleBindingCase struct {
	LifecycleState string
	CompartmentID  string
	Name           string
}

type quickMutableAliasDriftCase struct {
	Current int
	Desired int
}

type quickUnsupportedUpdateDriftCase struct {
	CurrentCompartmentID string
	DesiredCompartmentID string
}

func (quickLifecycleBindingCase) Generate(rand *rand.Rand, _ int) reflect.Value {
	lifecycleStates := []string{"ACTIVE", "CREATING", "UPDATING", "DELETING", "DELETED", "FAILED", "", "CUSTOM"}
	lifecycleState := lifecycleStates[rand.Intn(len(lifecycleStates))]
	if lifecycleState == "CUSTOM" {
		lifecycleState = fmt.Sprintf("custom-%d", rand.Intn(1_000_000))
	}

	return reflect.ValueOf(quickLifecycleBindingCase{
		LifecycleState: randomizeCase(rand, lifecycleState),
		CompartmentID:  fmt.Sprintf("ocid1.compartment.oc1..%06d", rand.Intn(1_000_000)),
		Name:           fmt.Sprintf("wanted-%06d", rand.Intn(1_000_000)),
	})
}

func (quickMutableAliasDriftCase) Generate(rand *rand.Rand, _ int) reflect.Value {
	current := rand.Intn(4096) + 1
	desired := rand.Intn(4096) + 1
	if rand.Intn(4) == 0 {
		desired = current
	}

	return reflect.ValueOf(quickMutableAliasDriftCase{
		Current: current,
		Desired: desired,
	})
}

func (quickUnsupportedUpdateDriftCase) Generate(rand *rand.Rand, _ int) reflect.Value {
	current := fmt.Sprintf("ocid1.compartment.oc1..current%06d", rand.Intn(1_000_000))
	desired := fmt.Sprintf("ocid1.compartment.oc1..desired%06d", rand.Intn(1_000_000))
	if rand.Intn(4) == 0 {
		desired = current
	}

	return reflect.ValueOf(quickUnsupportedUpdateDriftCase{
		CurrentCompartmentID: current,
		DesiredCompartmentID: desired,
	})
}

func randomizeCase(rand *rand.Rand, value string) string {
	if value == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value))
	for index := 0; index < len(value); index++ {
		ch := value[index]
		switch {
		case ch >= 'a' && ch <= 'z':
			if rand.Intn(2) == 0 {
				ch = ch - ('a' - 'A')
			}
		case ch >= 'A' && ch <= 'Z':
			if rand.Intn(2) == 0 {
				ch = ch + ('a' - 'A')
			}
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func (f *fakeCredentialClient) CreateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return false, nil
}

func (f *fakeCredentialClient) DeleteSecret(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (f *fakeCredentialClient) GetSecret(_ context.Context, name, namespace string) (map[string][]byte, error) {
	f.readNames = append(f.readNames, namespace+"/"+name)
	secret, ok := f.secrets[name]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
	}
	return secret, nil
}

func (f *fakeCredentialClient) UpdateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return false, nil
}

//nolint:gocyclo // The tableless assertions intentionally verify the full projected status surface end to end.
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

func TestServiceClientCreateOrUpdateResolvesSecretInputsAndStampsStatus(t *testing.T) {
	t.Parallel()

	var createRequest fakeCreateSecretThingRequest
	credentialClient := &fakeCredentialClient{
		secrets: map[string]map[string][]byte{
			"admin-secret": {
				"username": []byte("admin"),
				"password": []byte("S3cr3t!"),
			},
		},
	}

	client := NewServiceClient[*fakeSecretResource](Config[*fakeSecretResource]{
		Kind:             "Thing",
		SDKName:          "Thing",
		CredentialClient: credentialClient,
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateSecretThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*fakeCreateSecretThingRequest)
				return fakeCreateSecretThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..create",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
	})

	resource := &fakeSecretResource{
		Namespace: "default",
		Spec: fakeSecretSpec{
			DisplayName:   "desired-name",
			AdminUsername: shared.UsernameSource{Secret: shared.SecretSource{SecretName: "admin-secret"}},
			AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-secret"}},
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assertSuccessfulSecretCreate(t, response, err)
	assertSecretCreateRequestResolved(t, createRequest)
	assertSecretStatusStamped(t, resource, credentialClient)
}

func assertSuccessfulSecretCreate(t *testing.T, response servicemanager.OSOKResponse, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
}

func assertSecretCreateRequestResolved(t *testing.T, request fakeCreateSecretThingRequest) {
	t.Helper()

	if request.DisplayName != "desired-name" {
		t.Fatalf("create request displayName = %q, want desired-name", request.DisplayName)
	}
	if request.AdminUsername == nil || *request.AdminUsername != "admin" {
		t.Fatalf("create request adminUsername = %v, want admin", request.AdminUsername)
	}
	if request.AdminPassword == nil || *request.AdminPassword != "S3cr3t!" {
		t.Fatalf("create request adminPassword = %v, want resolved secret value", request.AdminPassword)
	}
}

func assertSecretStatusStamped(t *testing.T, resource *fakeSecretResource, credentialClient *fakeCredentialClient) {
	t.Helper()

	if got := resource.Status.AdminUsername.Secret.SecretName; got != "admin-secret" {
		t.Fatalf("status.adminUsername.secret.secretName = %q, want admin-secret", got)
	}
	if got := resource.Status.AdminPassword.Secret.SecretName; got != "admin-secret" {
		t.Fatalf("status.adminPassword.secret.secretName = %q, want admin-secret", got)
	}
	if len(credentialClient.readNames) != 2 {
		t.Fatalf("credential reads = %v, want username and password lookups", credentialClient.readNames)
	}
}

func TestServiceClientCreateOrUpdateOmitsEmptySecretInputs(t *testing.T) {
	t.Parallel()

	var createRequest fakeCreateSecretThingRequest
	credentialClient := &fakeCredentialClient{secrets: map[string]map[string][]byte{}}

	client := NewServiceClient[*fakeSecretResource](Config[*fakeSecretResource]{
		Kind:             "Thing",
		SDKName:          "Thing",
		CredentialClient: credentialClient,
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateSecretThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*fakeCreateSecretThingRequest)
				return fakeCreateSecretThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..create",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
	})

	resource := &fakeSecretResource{
		Namespace: "default",
		Spec: fakeSecretSpec{
			DisplayName: "desired-name",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createRequest.AdminUsername != nil {
		t.Fatalf("create request adminUsername = %v, want omitted nil pointer", createRequest.AdminUsername)
	}
	if createRequest.AdminPassword != nil {
		t.Fatalf("create request adminPassword = %v, want omitted nil pointer", createRequest.AdminPassword)
	}
	payload, err := json.Marshal(createRequest.FakeCreateSecretThingDetails)
	if err != nil {
		t.Fatalf("json.Marshal(create request) error = %v", err)
	}
	if strings.Contains(string(payload), "adminUsername") || strings.Contains(string(payload), "adminPassword") {
		t.Fatalf("create request payload = %s, want admin secret fields omitted", string(payload))
	}
	if len(credentialClient.readNames) != 0 {
		t.Fatalf("credential reads = %v, want no secret lookups", credentialClient.readNames)
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

func TestServiceClientCreateOrUpdateBindsExistingResourceBeforeCreate(t *testing.T) {
	t.Parallel()

	createCalled := false
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
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeNamedListThingResponse{
					Collection: fakeNamedThingCollection{
						Resources: []fakeThingSummary{
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
	if createCalled {
		t.Fatal("Create() should not be called when formal list lookup binds an existing resource")
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

func TestServiceClientCreateOrUpdateCreatesAfterFormalListMiss(t *testing.T) {
	t.Parallel()

	var createRequest fakeCreateThingRequest
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
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
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
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeNamedListThingResponse{
					Collection: fakeNamedThingCollection{Resources: nil},
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
			DisplayName:   "created-name",
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
	if createRequest.DisplayName != "created-name" {
		t.Fatalf("create request displayName = %q, want created-name", createRequest.DisplayName)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCreateOrUpdateDoesNotBindFormalListWithoutIdentifyingCriteria(t *testing.T) {
	t.Parallel()

	createCalled := false
	var createRequest fakeCreateThingRequest
	var listRequest fakeListThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Resources",
				MatchFields:        []string{"displayName", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				TerminalStates: []string{"DELETED"},
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
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*fakeListThingRequest)
				return fakeNamedListThingResponse{
					Collection: fakeNamedThingCollection{
						Resources: []fakeThingSummary{
							{Id: "ocid1.thing.oc1..existing", CompartmentId: "ocid1.compartment.oc1..match", DisplayName: "existing-name", LifecycleState: "ACTIVE"},
						},
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !createCalled {
		t.Fatal("Create() should be called when formal list lookup lacks identifying criteria")
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if listRequest.CompartmentId != "ocid1.compartment.oc1..match" {
		t.Fatalf("list request compartmentId = %q, want ocid1.compartment.oc1..match", listRequest.CompartmentId)
	}
	if listRequest.DisplayName != "" {
		t.Fatalf("list request displayName = %q, want omitted displayName", listRequest.DisplayName)
	}
	if createRequest.DisplayName != "" {
		t.Fatalf("create request displayName = %q, want omitted displayName", createRequest.DisplayName)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.thing.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestServiceClientCreateOrUpdateFormalListBindingRespectsReusableLifecycleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		lifecycleState string
		wantCreate     bool
		wantRequeue    bool
		wantOcid       string
	}{
		{name: "active", lifecycleState: "ACTIVE", wantCreate: false, wantRequeue: false, wantOcid: "ocid1.thing.oc1..existing"},
		{name: "creating", lifecycleState: "CREATING", wantCreate: false, wantRequeue: true, wantOcid: "ocid1.thing.oc1..existing"},
		{name: "updating", lifecycleState: "UPDATING", wantCreate: false, wantRequeue: true, wantOcid: "ocid1.thing.oc1..existing"},
		{name: "deleting", lifecycleState: "DELETING", wantCreate: true, wantRequeue: false, wantOcid: "ocid1.thing.oc1..created"},
		{name: "deleted", lifecycleState: "DELETED", wantCreate: true, wantRequeue: false, wantOcid: "ocid1.thing.oc1..created"},
		{name: "failed", lifecycleState: "FAILED", wantCreate: true, wantRequeue: false, wantOcid: "ocid1.thing.oc1..created"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runFormalListBindingLifecycleCase(t, tt.lifecycleState, tt.wantCreate, tt.wantRequeue, tt.wantOcid)
		})
	}
}

func TestServiceClientCreateOrUpdateFormalListBindingRespectsReusableLifecycleStatesQuick(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(bindingCase quickLifecycleBindingCase) bool {
		return checkLifecycleBindingQuickCase(t, bindingCase)
	}, &quick.Config{MaxCount: 64})
	if err != nil {
		t.Fatal(err)
	}
}

func checkLifecycleBindingQuickCase(t *testing.T, bindingCase quickLifecycleBindingCase) bool {
	t.Helper()

	wantCreate, wantRequeue, wantOcid := expectedLifecycleBindingOutcome(bindingCase.LifecycleState)
	client, createCalled, listRequest, createRequest := newFormalListBindingLifecycleClient(
		bindingCase.LifecycleState,
		bindingCase.Name,
		bindingCase.CompartmentID,
	)
	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: bindingCase.CompartmentID,
			DisplayName:   "created-name",
			Name:          bindingCase.Name,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Logf("CreateOrUpdate(%q) error = %v", bindingCase.LifecycleState, err)
		return false
	}
	if !response.IsSuccessful {
		t.Logf("CreateOrUpdate(%q) reported unsuccessful response", bindingCase.LifecycleState)
		return false
	}
	if *createCalled != wantCreate {
		t.Logf("Create() called = %t, want %t for lifecycle %q", *createCalled, wantCreate, bindingCase.LifecycleState)
		return false
	}
	if response.ShouldRequeue != wantRequeue {
		t.Logf("response.ShouldRequeue = %t, want %t for lifecycle %q", response.ShouldRequeue, wantRequeue, bindingCase.LifecycleState)
		return false
	}
	if listRequest.CompartmentId != bindingCase.CompartmentID {
		t.Logf("list request compartmentId = %q, want %q", listRequest.CompartmentId, bindingCase.CompartmentID)
		return false
	}
	if listRequest.Name != bindingCase.Name {
		t.Logf("list request name = %q, want %q", listRequest.Name, bindingCase.Name)
		return false
	}
	if wantCreate && createRequest.DisplayName != "created-name" {
		t.Logf("create request displayName = %q, want created-name", createRequest.DisplayName)
		return false
	}
	if string(resource.Status.OsokStatus.Ocid) != wantOcid {
		t.Logf("status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, wantOcid)
		return false
	}
	return true
}

func expectedLifecycleBindingOutcome(lifecycleState string) (wantCreate, wantRequeue bool, wantOcid string) {
	switch strings.ToUpper(lifecycleState) {
	case "ACTIVE":
		return false, false, "ocid1.thing.oc1..existing"
	case "CREATING", "UPDATING":
		return false, true, "ocid1.thing.oc1..existing"
	default:
		return true, false, "ocid1.thing.oc1..created"
	}
}

func runFormalListBindingLifecycleCase(t *testing.T, lifecycleState string, wantCreate, wantRequeue bool, wantOcid string) {
	t.Helper()

	client, createCalled, listRequest, createRequest := newFormalListBindingLifecycleClient(
		lifecycleState,
		"wanted",
		"ocid1.compartment.oc1..match",
	)
	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			DisplayName:   "created-name",
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
	if *createCalled != wantCreate {
		t.Fatalf("Create() called = %t, want %t for lifecycle %q", *createCalled, wantCreate, lifecycleState)
	}
	if response.ShouldRequeue != wantRequeue {
		t.Fatalf("response.ShouldRequeue = %t, want %t for lifecycle %q", response.ShouldRequeue, wantRequeue, lifecycleState)
	}
	if listRequest.CompartmentId != "ocid1.compartment.oc1..match" {
		t.Fatalf("list request compartmentId = %q, want ocid1.compartment.oc1..match", listRequest.CompartmentId)
	}
	if listRequest.Name != "wanted" {
		t.Fatalf("list request name = %q, want wanted", listRequest.Name)
	}
	if wantCreate && createRequest.DisplayName != "created-name" {
		t.Fatalf("create request displayName = %q, want created-name", createRequest.DisplayName)
	}
	if string(resource.Status.OsokStatus.Ocid) != wantOcid {
		t.Fatalf("status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, wantOcid)
	}
}

func newFormalListBindingLifecycleClient(lifecycleState, name, compartmentID string) (ServiceClient[*fakeResource], *bool, *fakeListThingRequest, *fakeCreateThingRequest) {
	createCalled := false
	createRequest := fakeCreateThingRequest{}
	listRequest := fakeListThingRequest{}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Resources",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*fakeCreateThingRequest)
				createCalled = true
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						DisplayName:    "created-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
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
								Id:             "ocid1.thing.oc1..existing",
								Name:           name,
								CompartmentId:  compartmentID,
								LifecycleState: lifecycleState,
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

	return client, &createCalled, &listRequest, &createRequest
}

func TestServiceClientCreateOrUpdateSkipsUpdateWhenNoSupportedDriftRemains(t *testing.T) {
	t.Parallel()

	updateCalled := false
	var getRequest fakeGetThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				TerminalStates: []string{"DELETED"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				updateCalled = true
				return fakeUpdateThingResponse{}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getRequest = *request.(*fakeGetThingRequest)
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "same-name",
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
			DisplayName: "same-name",
			Enabled:     true,
		},
		Status: fakeStatus{
			OsokStatus:  shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:          "ocid1.thing.oc1..existing",
			DisplayName: "same-name",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if updateCalled {
		t.Fatal("Update() should not be called when no supported mutable drift remains")
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..existing" {
		t.Fatalf("get request thingId = %v, want existing OCID", getRequest.ThingId)
	}
}

func TestServiceClientAllowsMutableDriftThroughInGBsAlias(t *testing.T) {
	t.Parallel()

	updateCalled := false

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				TerminalStates: []string{"DELETED"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"dataStorageSizeInGb"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateCalled = true
				updateRequest := *request.(*fakeUpdateThingRequest)
				if updateRequest.DataStorageSizeInGBs != 20 {
					t.Fatalf("update request dataStorageSizeInGBs = %d, want 20", updateRequest.DataStorageSizeInGBs)
				}
				return fakeUpdateThingResponse{
					Thing: fakeThing{
						Id:                   "ocid1.thing.oc1..existing",
						DataStorageSizeInGBs: 20,
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
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !updateCalled {
		t.Fatal("Update() should be called when only the InGBs alias differs")
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if resource.Status.DataStorageSizeInGBs != 20 {
		t.Fatalf("status.dataStorageSizeInGBs = %d, want 20", resource.Status.DataStorageSizeInGBs)
	}
}

func TestServiceClientAllowsMutableDriftThroughInGBsAliasQuick(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(driftCase quickMutableAliasDriftCase) bool {
		wantUpdate := driftCase.Current != driftCase.Desired
		client, updateCalled, updateRequest := newMutableAliasUpdateClient()
		resource := &fakeResource{
			Spec: fakeSpec{
				DataStorageSizeInGBs: driftCase.Desired,
			},
			Status: fakeStatus{
				OsokStatus:           shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
				Id:                   "ocid1.thing.oc1..existing",
				DataStorageSizeInGBs: driftCase.Current,
			},
		}

		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if err != nil {
			t.Logf("CreateOrUpdate() error = %v for current=%d desired=%d", err, driftCase.Current, driftCase.Desired)
			return false
		}
		if !response.IsSuccessful {
			t.Logf("CreateOrUpdate() reported unsuccessful response for current=%d desired=%d", driftCase.Current, driftCase.Desired)
			return false
		}
		if response.ShouldRequeue {
			t.Logf("response.ShouldRequeue = true, want false for current=%d desired=%d", driftCase.Current, driftCase.Desired)
			return false
		}
		if *updateCalled != wantUpdate {
			t.Logf("Update() called = %t, want %t for current=%d desired=%d", *updateCalled, wantUpdate, driftCase.Current, driftCase.Desired)
			return false
		}
		if wantUpdate && updateRequest.DataStorageSizeInGBs != driftCase.Desired {
			t.Logf("update request dataStorageSizeInGBs = %d, want %d", updateRequest.DataStorageSizeInGBs, driftCase.Desired)
			return false
		}
		if resource.Status.DataStorageSizeInGBs != driftCase.Desired {
			t.Logf("status.dataStorageSizeInGBs = %d, want %d", resource.Status.DataStorageSizeInGBs, driftCase.Desired)
			return false
		}
		return true
	}, &quick.Config{MaxCount: 64})
	if err != nil {
		t.Fatal(err)
	}
}

func newMutableAliasUpdateClient() (ServiceClient[*fakeResource], *bool, *fakeUpdateThingRequest) {
	updateCalled := false
	updateRequest := fakeUpdateThingRequest{}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				TerminalStates: []string{"DELETED"},
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

	return client, &updateCalled, &updateRequest
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

func TestServiceClientRejectsUnsupportedUpdateDriftQuick(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(driftCase quickUnsupportedUpdateDriftCase) bool {
		return checkUnsupportedUpdateDriftQuickCase(t, driftCase)
	}, &quick.Config{MaxCount: 64})
	if err != nil {
		t.Fatal(err)
	}
}

func checkUnsupportedUpdateDriftQuickCase(t *testing.T, driftCase quickUnsupportedUpdateDriftCase) bool {
	t.Helper()

	wantReject := driftCase.CurrentCompartmentID != driftCase.DesiredCompartmentID
	client, updateCalled := newUnsupportedUpdateDriftClient()
	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: driftCase.DesiredCompartmentID,
			DisplayName:   "same-name",
		},
		Status: fakeStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:            "ocid1.thing.oc1..existing",
			CompartmentId: driftCase.CurrentCompartmentID,
			DisplayName:   "same-name",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if wantReject {
		return checkRejectedUnsupportedUpdateDriftCase(t, driftCase, updateCalled, err)
	}
	return checkAllowedUnsupportedUpdateDriftCase(t, driftCase, updateCalled, response, err)
}

func checkRejectedUnsupportedUpdateDriftCase(t *testing.T, driftCase quickUnsupportedUpdateDriftCase, updateCalled *bool, err error) bool {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), "reject unsupported update drift for compartmentId") {
		t.Logf("CreateOrUpdate() error = %v, want unsupported drift failure for current=%q desired=%q", err, driftCase.CurrentCompartmentID, driftCase.DesiredCompartmentID)
		return false
	}
	if *updateCalled {
		t.Logf("Update() called for rejected drift current=%q desired=%q", driftCase.CurrentCompartmentID, driftCase.DesiredCompartmentID)
		return false
	}
	return true
}

func checkAllowedUnsupportedUpdateDriftCase(
	t *testing.T,
	driftCase quickUnsupportedUpdateDriftCase,
	updateCalled *bool,
	response servicemanager.OSOKResponse,
	err error,
) bool {
	t.Helper()

	if err != nil {
		t.Logf("CreateOrUpdate() error = %v for equal current=%q desired=%q", err, driftCase.CurrentCompartmentID, driftCase.DesiredCompartmentID)
		return false
	}
	if !response.IsSuccessful {
		t.Logf("CreateOrUpdate() reported unsuccessful response for equal current=%q desired=%q", driftCase.CurrentCompartmentID, driftCase.DesiredCompartmentID)
		return false
	}
	if response.ShouldRequeue {
		t.Logf("response.ShouldRequeue = true, want false for equal current=%q desired=%q", driftCase.CurrentCompartmentID, driftCase.DesiredCompartmentID)
		return false
	}
	if *updateCalled {
		t.Logf("Update() called for equal current=%q desired=%q", driftCase.CurrentCompartmentID, driftCase.DesiredCompartmentID)
		return false
	}
	return true
}

func newUnsupportedUpdateDriftClient() (ServiceClient[*fakeResource], *bool) {
	updateCalled := false

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
				updateCalled = true
				return fakeUpdateThingResponse{}, nil
			},
		},
	})

	return client, &updateCalled
}

func TestServiceClientRejectsForceNewSecretSourceMutation(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeSecretResource](Config[*fakeSecretResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Mutation: MutationSemantics{
				ForceNew: []string{"adminUsername", "adminPassword"},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				t.Fatal("Update() should not be called when a secret-backed force-new field changes")
				return nil, nil
			},
		},
	})

	resource := &fakeSecretResource{
		Spec: fakeSecretSpec{
			AdminUsername: shared.UsernameSource{Secret: shared.SecretSource{SecretName: "new-user-secret"}},
			AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "old-password-secret"}},
		},
		Status: fakeSecretStatus{
			OsokStatus:    shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:            "ocid1.thing.oc1..existing",
			AdminUsername: shared.UsernameSource{Secret: shared.SecretSource{SecretName: "old-user-secret"}},
			AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "old-password-secret"}},
		},
	}

	if _, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "require replacement when adminUsername changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new replacement failure for adminUsername", err)
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
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..delete" {
		t.Fatalf("delete request thingId = %v, want existing OCID", deleteRequest.ThingId)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteUsesFormalListLookupWhenTrackedIDMissing(t *testing.T) {
	t.Parallel()

	var deleteRequest fakeDeleteThingRequest
	var listRequest fakeListThingRequest
	var getRequest fakeGetThingRequest

	client := newFormalDeleteListLookupClient(t, &deleteRequest, &listRequest, &getRequest)

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
		t.Fatal("Delete() should report completion after list lookup resolves and delete confirmation returns DELETED")
	}
	assertDeleteLookupRequests(t, listRequest, deleteRequest, getRequest)
}

func newFormalDeleteListLookupClient(t *testing.T, deleteRequest *fakeDeleteThingRequest, listRequest *fakeListThingRequest, getRequest *fakeGetThingRequest) ServiceClient[*fakeResource] {
	t.Helper()

	return NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Resources",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				*deleteRequest = *request.(*fakeDeleteThingRequest)
				return fakeDeleteThingResponse{}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				*getRequest = *request.(*fakeGetThingRequest)
				ensureResolvedThingID(t, *getRequest)
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..matched",
						LifecycleState: "DELETED",
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
				*listRequest = *request.(*fakeListThingRequest)
				return fakeNamedListThingResponse{
					Collection: fakeNamedThingCollection{
						Resources: []fakeThingSummary{
							{Id: "ocid1.thing.oc1..matched", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "ACTIVE"},
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
}

func ensureResolvedThingID(t *testing.T, request fakeGetThingRequest) {
	t.Helper()

	if request.ThingId == nil || *request.ThingId == "" {
		t.Fatal("Get() should not be called without a resolved OCI identifier")
	}
}

func assertDeleteLookupRequests(t *testing.T, listRequest fakeListThingRequest, deleteRequest fakeDeleteThingRequest, getRequest fakeGetThingRequest) {
	t.Helper()

	if listRequest.CompartmentId != "ocid1.compartment.oc1..match" {
		t.Fatalf("list request compartmentId = %q, want ocid1.compartment.oc1..match", listRequest.CompartmentId)
	}
	if listRequest.Name != "wanted" {
		t.Fatalf("list request name = %q, want wanted", listRequest.Name)
	}
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..matched" {
		t.Fatalf("delete request thingId = %v, want matched OCID", deleteRequest.ThingId)
	}
	if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..matched" {
		t.Fatalf("get request thingId = %v, want matched OCID", getRequest.ThingId)
	}
}
