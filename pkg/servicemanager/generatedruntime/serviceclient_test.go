/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

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
	CompartmentId string `json:"compartmentId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	Name          string `json:"name,omitempty"`
	Enabled       bool   `json:"enabled,omitempty"`
}

type fakeStatus struct {
	OsokStatus     shared.OSOKStatus `json:"status"`
	Id             string            `json:"id,omitempty"`
	CompartmentId  string            `json:"compartmentId,omitempty"`
	DisplayName    string            `json:"displayName,omitempty"`
	LifecycleState string            `json:"lifecycleState,omitempty"`
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
	Id             string `json:"id,omitempty"`
	CompartmentId  string `json:"compartmentId,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	Name           string `json:"name,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
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
