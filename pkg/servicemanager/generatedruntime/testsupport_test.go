/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"testing"
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
	CreateWorkRequestId  string                `json:"createWorkRequestId,omitempty"`
	UpdateWorkRequestId  string                `json:"updateWorkRequestId,omitempty"`
	DeleteWorkRequestId  string                `json:"deleteWorkRequestId,omitempty"`
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
	OpcRequestId     *string   `presentIn:"header" name:"opc-request-id"`
	OpcWorkRequestId *string   `presentIn:"header" name:"opc-work-request-id"`
	Thing            fakeThing `presentIn:"body"`
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
	OpcRequestId *string   `presentIn:"header" name:"opc-request-id"`
	Thing        fakeThing `presentIn:"body"`
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
	OpcRequestId     *string   `presentIn:"header" name:"opc-request-id"`
	OpcWorkRequestId *string   `presentIn:"header" name:"opc-work-request-id"`
	Thing            fakeThing `presentIn:"body"`
}

type fakeDeleteThingRequest struct {
	ThingId *string `contributesTo:"path" name:"thingId"`
}

type fakeDeleteThingResponse struct {
	OpcRequestId     *string `presentIn:"header" name:"opc-request-id"`
	OpcWorkRequestId *string `presentIn:"header" name:"opc-work-request-id"`
}

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

type fakePathIdentity struct {
	parentID    string
	thingName   string
	syntheticID string
}

type fakeNestedGetThingRequest struct {
	ParentId  *string `contributesTo:"path" name:"parentId"`
	ThingName *string `contributesTo:"path" name:"thingName"`
}

type fakeNestedListThingRequest struct {
	ParentId *string `contributesTo:"path" name:"parentId"`
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

func requireStatusOpcRequestID(t *testing.T, resource *fakeResource, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
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

func requireCurrentWorkRequestID(t *testing.T, resource *fakeResource, want string) {
	t.Helper()
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want populated tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.WorkRequestID; got != want {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", got, want)
	}
}

func requireCurrentAsyncSource(t *testing.T, resource *fakeResource, want shared.OSOKAsyncSource) {
	t.Helper()
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want populated tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.Source; got != want {
		t.Fatalf("status.async.current.source = %q, want %q", got, want)
	}
}

func requireCurrentAsyncPhase(t *testing.T, resource *fakeResource, want shared.OSOKAsyncPhase) {
	t.Helper()
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want populated tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != want {
		t.Fatalf("status.async.current.phase = %q, want %q", got, want)
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

func stringPtr(value string) *string {
	return &value
}
