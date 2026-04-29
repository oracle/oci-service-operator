/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odaprivateendpoint

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestOdaPrivateEndpointRuntimeHooksConfigureModernSemantics(t *testing.T) {
	hooks := newOdaPrivateEndpointDefaultRuntimeHooks(odasdk.ManagementClient{})
	applyOdaPrivateEndpointRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed lifecycle semantics")
	}
	if hooks.Semantics.FormalService != "oda" || hooks.Semantics.FormalSlug != "odaprivateendpoint" {
		t.Fatalf("formal binding = %s/%s, want oda/odaprivateendpoint", hooks.Semantics.FormalService, hooks.Semantics.FormalSlug)
	}
	if hooks.Semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", hooks.Semantics.FinalizerPolicy)
	}
	if hooks.Semantics.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", hooks.Semantics.Delete.Policy)
	}
	if !slices.Contains(hooks.Semantics.Lifecycle.ActiveStates, string(odasdk.OdaPrivateEndpointLifecycleStateActive)) {
		t.Fatalf("ActiveStates = %#v, want ACTIVE", hooks.Semantics.Lifecycle.ActiveStates)
	}
	for _, field := range []string{"displayName", "description", "nsgIds", "freeformTags", "definedTags"} {
		if !slices.Contains(hooks.Semantics.Mutation.Mutable, field) {
			t.Fatalf("Mutation.Mutable = %#v, want %s", hooks.Semantics.Mutation.Mutable, field)
		}
	}
	for _, field := range []string{"compartmentId", "subnetId"} {
		if !slices.Contains(hooks.Semantics.Mutation.ForceNew, field) {
			t.Fatalf("Mutation.ForceNew = %#v, want %s", hooks.Semantics.Mutation.ForceNew, field)
		}
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("Identity.GuardExistingBeforeCreate = nil, want bounded create-or-bind guard")
	}

	decision, err := hooks.Identity.GuardExistingBeforeCreate(context.Background(), newOdaPrivateEndpointResource(""))
	if err != nil {
		t.Fatalf("GuardExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedSkip {
		t.Fatalf("GuardExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedSkip)
	}

	decision, err = hooks.Identity.GuardExistingBeforeCreate(context.Background(), newOdaPrivateEndpointResource("private-endpoint"))
	if err != nil {
		t.Fatalf("GuardExistingBeforeCreate(named resource) error = %v", err)
	}
	if decision != generatedAllow {
		t.Fatalf("GuardExistingBeforeCreate(named resource) = %q, want %q", decision, generatedAllow)
	}
}

func TestOdaPrivateEndpointCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newOdaPrivateEndpointResource("private-endpoint")
	client := &fakeOdaPrivateEndpointOCIClient{
		listResponses: []odasdk.ListOdaPrivateEndpointsResponse{
			{OdaPrivateEndpointCollection: odasdk.OdaPrivateEndpointCollection{}},
		},
		createResponses: []odasdk.CreateOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.created", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, odasdk.OdaPrivateEndpointLifecycleStateCreating)},
		},
		getResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.created", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, odasdk.OdaPrivateEndpointLifecycleStateActive)},
		},
	}

	response, err := newOdaPrivateEndpointServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = true, want false after ACTIVE follow-up")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("Create requests = %d, want 1", len(client.createRequests))
	}

	createRequest := client.createRequests[0]
	requireStringPointer(t, "create compartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
	requireStringPointer(t, "create subnetId", createRequest.SubnetId, resource.Spec.SubnetId)
	requireStringPointer(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	if createRequest.OpcRetryToken == nil || strings.TrimSpace(*createRequest.OpcRetryToken) == "" {
		t.Fatal("Create request OpcRetryToken is empty")
	}
	requireProjectedOdaPrivateEndpoint(t, resource, "ocid1.ope.created", odasdk.OdaPrivateEndpointLifecycleStateActive)
	requireLastCondition(t, resource, shared.Active)
}

func TestOdaPrivateEndpointCreateOrUpdateBindsExistingResourceBeforeCreate(t *testing.T) {
	resource := newOdaPrivateEndpointResource("existing-private-endpoint")
	client := &fakeOdaPrivateEndpointOCIClient{
		listResponses: []odasdk.ListOdaPrivateEndpointsResponse{
			{
				OdaPrivateEndpointCollection: odasdk.OdaPrivateEndpointCollection{
					Items: []odasdk.OdaPrivateEndpointSummary{
						{
							Id:             common.String("ocid1.ope.existing"),
							DisplayName:    common.String(resource.Spec.DisplayName),
							CompartmentId:  common.String(resource.Spec.CompartmentId),
							LifecycleState: odasdk.OdaPrivateEndpointLifecycleStateActive,
						},
					},
				},
			},
		},
		getResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.existing", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, odasdk.OdaPrivateEndpointLifecycleStateActive)},
		},
	}

	response, err := newOdaPrivateEndpointServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("Create requests = %d, want 0 for existing bind", len(client.createRequests))
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("List requests = %d, want 1", len(client.listRequests))
	}
	requireStringPointer(t, "list compartmentId", client.listRequests[0].CompartmentId, resource.Spec.CompartmentId)
	requireStringPointer(t, "list displayName", client.listRequests[0].DisplayName, resource.Spec.DisplayName)
	requireProjectedOdaPrivateEndpoint(t, resource, "ocid1.ope.existing", odasdk.OdaPrivateEndpointLifecycleStateActive)
}

func TestOdaPrivateEndpointCreateOrUpdateUpdatesMutableDrift(t *testing.T) {
	resource := newOdaPrivateEndpointResource("private-endpoint-new")
	resource.Spec.Description = "new description"
	resource.Spec.NsgIds = []string{"nsg-new"}
	resource.Spec.FreeformTags = map[string]string{"env": "test"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"namespace": {"key": "value"}}
	recordOdaPrivateEndpointID(resource, "ocid1.ope.update")

	client := &fakeOdaPrivateEndpointOCIClient{
		getResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{
				OdaPrivateEndpoint: odasdk.OdaPrivateEndpoint{
					Id:             common.String("ocid1.ope.update"),
					DisplayName:    common.String("private-endpoint-old"),
					CompartmentId:  common.String(resource.Spec.CompartmentId),
					SubnetId:       common.String(resource.Spec.SubnetId),
					Description:    common.String("old description"),
					NsgIds:         []string{"nsg-old"},
					FreeformTags:   map[string]string{"env": "old"},
					DefinedTags:    map[string]map[string]interface{}{"namespace": {"old": "value"}},
					LifecycleState: odasdk.OdaPrivateEndpointLifecycleStateActive,
				},
			},
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.update", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, odasdk.OdaPrivateEndpointLifecycleStateActive)},
		},
		updateResponses: []odasdk.UpdateOdaPrivateEndpointResponse{{}},
	}
	client.getResponses[1].OdaPrivateEndpoint.Description = common.String(resource.Spec.Description)
	client.getResponses[1].OdaPrivateEndpoint.NsgIds = append([]string(nil), resource.Spec.NsgIds...)
	client.getResponses[1].OdaPrivateEndpoint.FreeformTags = map[string]string{"env": "test"}
	client.getResponses[1].OdaPrivateEndpoint.DefinedTags = map[string]map[string]interface{}{"namespace": {"key": "value"}}

	response, err := newOdaPrivateEndpointServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue update", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("Update requests = %d, want 1", len(client.updateRequests))
	}

	updateRequest := client.updateRequests[0]
	requireStringPointer(t, "update id", updateRequest.OdaPrivateEndpointId, "ocid1.ope.update")
	requireStringPointer(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	requireStringPointer(t, "update description", updateRequest.Description, resource.Spec.Description)
	if !slices.Equal(updateRequest.NsgIds, resource.Spec.NsgIds) {
		t.Fatalf("update nsgIds = %#v, want %#v", updateRequest.NsgIds, resource.Spec.NsgIds)
	}
	if updateRequest.FreeformTags["env"] != "test" {
		t.Fatalf("update freeformTags = %#v, want env=test", updateRequest.FreeformTags)
	}
	if updateRequest.DefinedTags["namespace"]["key"] != "value" {
		t.Fatalf("update definedTags = %#v, want namespace.key=value", updateRequest.DefinedTags)
	}
	requireProjectedOdaPrivateEndpoint(t, resource, "ocid1.ope.update", odasdk.OdaPrivateEndpointLifecycleStateActive)
}

func TestOdaPrivateEndpointCreateOrUpdateRejectsForceNewDriftBeforeUpdate(t *testing.T) {
	resource := newOdaPrivateEndpointResource("private-endpoint")
	resource.Spec.SubnetId = "ocid1.subnet.new"
	recordOdaPrivateEndpointID(resource, "ocid1.ope.force-new")
	client := &fakeOdaPrivateEndpointOCIClient{
		getResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.force-new", resource.Spec.DisplayName, resource.Spec.CompartmentId, "ocid1.subnet.old", odasdk.OdaPrivateEndpointLifecycleStateActive)},
		},
	}

	response, err := newOdaPrivateEndpointServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when subnetId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want subnetId force-new drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("Update requests = %d, want 0 after force-new rejection", len(client.updateRequests))
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestOdaPrivateEndpointCreateOrUpdateMapsLifecycleStates(t *testing.T) {
	tests := []struct {
		name          string
		state         odasdk.OdaPrivateEndpointLifecycleStateEnum
		wantCondition shared.OSOKConditionType
		wantRequeue   bool
		wantSuccess   bool
	}{
		{name: "creating", state: odasdk.OdaPrivateEndpointLifecycleStateCreating, wantCondition: shared.Provisioning, wantRequeue: true, wantSuccess: true},
		{name: "updating", state: odasdk.OdaPrivateEndpointLifecycleStateUpdating, wantCondition: shared.Updating, wantRequeue: true, wantSuccess: true},
		{name: "active", state: odasdk.OdaPrivateEndpointLifecycleStateActive, wantCondition: shared.Active, wantRequeue: false, wantSuccess: true},
		{name: "failed", state: odasdk.OdaPrivateEndpointLifecycleStateFailed, wantCondition: shared.Failed, wantRequeue: false, wantSuccess: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := newOdaPrivateEndpointResource("private-endpoint")
			recordOdaPrivateEndpointID(resource, "ocid1.ope.lifecycle")
			client := &fakeOdaPrivateEndpointOCIClient{
				getResponses: []odasdk.GetOdaPrivateEndpointResponse{
					{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.lifecycle", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, tc.state)},
				},
			}

			response, err := newOdaPrivateEndpointServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccess {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t", response.IsSuccessful, tc.wantSuccess)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			requireLastCondition(t, resource, tc.wantCondition)
		})
	}
}

func TestOdaPrivateEndpointDeleteRetainsFinalizerUntilDeleteIsConfirmed(t *testing.T) {
	resource := newOdaPrivateEndpointResource("private-endpoint")
	recordOdaPrivateEndpointID(resource, "ocid1.ope.delete")
	client := &fakeOdaPrivateEndpointOCIClient{
		getResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.delete", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, odasdk.OdaPrivateEndpointLifecycleStateActive)},
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.delete", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, odasdk.OdaPrivateEndpointLifecycleStateDeleting)},
		},
		deleteResponses: []odasdk.DeleteOdaPrivateEndpointResponse{{}},
	}

	deleted, err := newOdaPrivateEndpointServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("Delete requests = %d, want 1", len(client.deleteRequests))
	}
	requireStringPointer(t, "delete id", client.deleteRequests[0].OdaPrivateEndpointId, "ocid1.ope.delete")
	requireLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("DeletedAt is set before delete confirmation")
	}
}

func TestOdaPrivateEndpointDeleteReleasesFinalizerAfterTerminalDelete(t *testing.T) {
	resource := newOdaPrivateEndpointResource("private-endpoint")
	recordOdaPrivateEndpointID(resource, "ocid1.ope.deleted")
	client := &fakeOdaPrivateEndpointOCIClient{
		getResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: newOdaPrivateEndpoint("ocid1.ope.deleted", resource.Spec.DisplayName, resource.Spec.CompartmentId, resource.Spec.SubnetId, odasdk.OdaPrivateEndpointLifecycleStateDeleted)},
		},
	}

	deleted, err := newOdaPrivateEndpointServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED confirmation")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("Delete requests = %d, want 0 when delete is already terminal", len(client.deleteRequests))
	}
	requireLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("DeletedAt is nil after terminal delete confirmation")
	}
}

type fakeOdaPrivateEndpointOCIClient struct {
	createRequests  []odasdk.CreateOdaPrivateEndpointRequest
	getRequests     []odasdk.GetOdaPrivateEndpointRequest
	listRequests    []odasdk.ListOdaPrivateEndpointsRequest
	updateRequests  []odasdk.UpdateOdaPrivateEndpointRequest
	deleteRequests  []odasdk.DeleteOdaPrivateEndpointRequest
	createResponses []odasdk.CreateOdaPrivateEndpointResponse
	getResponses    []odasdk.GetOdaPrivateEndpointResponse
	listResponses   []odasdk.ListOdaPrivateEndpointsResponse
	updateResponses []odasdk.UpdateOdaPrivateEndpointResponse
	deleteResponses []odasdk.DeleteOdaPrivateEndpointResponse
}

func (c *fakeOdaPrivateEndpointOCIClient) CreateOdaPrivateEndpoint(_ context.Context, request odasdk.CreateOdaPrivateEndpointRequest) (odasdk.CreateOdaPrivateEndpointResponse, error) {
	c.createRequests = append(c.createRequests, request)
	if len(c.createResponses) == 0 {
		return odasdk.CreateOdaPrivateEndpointResponse{}, fmt.Errorf("unexpected CreateOdaPrivateEndpoint call")
	}
	response := c.createResponses[0]
	c.createResponses = c.createResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointOCIClient) GetOdaPrivateEndpoint(_ context.Context, request odasdk.GetOdaPrivateEndpointRequest) (odasdk.GetOdaPrivateEndpointResponse, error) {
	c.getRequests = append(c.getRequests, request)
	if len(c.getResponses) == 0 {
		return odasdk.GetOdaPrivateEndpointResponse{}, fmt.Errorf("unexpected GetOdaPrivateEndpoint call")
	}
	response := c.getResponses[0]
	c.getResponses = c.getResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointOCIClient) ListOdaPrivateEndpoints(_ context.Context, request odasdk.ListOdaPrivateEndpointsRequest) (odasdk.ListOdaPrivateEndpointsResponse, error) {
	c.listRequests = append(c.listRequests, request)
	if len(c.listResponses) == 0 {
		return odasdk.ListOdaPrivateEndpointsResponse{}, fmt.Errorf("unexpected ListOdaPrivateEndpoints call")
	}
	response := c.listResponses[0]
	c.listResponses = c.listResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointOCIClient) UpdateOdaPrivateEndpoint(_ context.Context, request odasdk.UpdateOdaPrivateEndpointRequest) (odasdk.UpdateOdaPrivateEndpointResponse, error) {
	c.updateRequests = append(c.updateRequests, request)
	if len(c.updateResponses) == 0 {
		return odasdk.UpdateOdaPrivateEndpointResponse{}, fmt.Errorf("unexpected UpdateOdaPrivateEndpoint call")
	}
	response := c.updateResponses[0]
	c.updateResponses = c.updateResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointOCIClient) DeleteOdaPrivateEndpoint(_ context.Context, request odasdk.DeleteOdaPrivateEndpointRequest) (odasdk.DeleteOdaPrivateEndpointResponse, error) {
	c.deleteRequests = append(c.deleteRequests, request)
	if len(c.deleteResponses) == 0 {
		return odasdk.DeleteOdaPrivateEndpointResponse{}, fmt.Errorf("unexpected DeleteOdaPrivateEndpoint call")
	}
	response := c.deleteResponses[0]
	c.deleteResponses = c.deleteResponses[1:]
	return response, nil
}

func newOdaPrivateEndpointResource(displayName string) *odav1beta1.OdaPrivateEndpoint {
	return &odav1beta1.OdaPrivateEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "private-endpoint",
			Namespace: "default",
			UID:       types.UID("uid-oda-private-endpoint"),
		},
		Spec: odav1beta1.OdaPrivateEndpointSpec{
			CompartmentId: "ocid1.compartment.oc1..aaaa",
			SubnetId:      "ocid1.subnet.oc1..aaaa",
			DisplayName:   displayName,
		},
	}
}

func newOdaPrivateEndpoint(id string, displayName string, compartmentID string, subnetID string, state odasdk.OdaPrivateEndpointLifecycleStateEnum) odasdk.OdaPrivateEndpoint {
	return odasdk.OdaPrivateEndpoint{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		SubnetId:       common.String(subnetID),
		LifecycleState: state,
	}
}

func recordOdaPrivateEndpointID(resource *odav1beta1.OdaPrivateEndpoint, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func requireProjectedOdaPrivateEndpoint(t *testing.T, resource *odav1beta1.OdaPrivateEndpoint, id string, state odasdk.OdaPrivateEndpointLifecycleStateEnum) {
	t.Helper()
	if resource.Status.Id != id {
		t.Fatalf("Status.Id = %q, want %q", resource.Status.Id, id)
	}
	if string(resource.Status.OsokStatus.Ocid) != id {
		t.Fatalf("Status.OsokStatus.Ocid = %q, want %q", resource.Status.OsokStatus.Ocid, id)
	}
	if resource.Status.LifecycleState != string(state) {
		t.Fatalf("Status.LifecycleState = %q, want %q", resource.Status.LifecycleState, state)
	}
}

func requireStringPointer(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireLastCondition(t *testing.T, resource *odav1beta1.OdaPrivateEndpoint, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status conditions empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s; conditions=%#v", got, want, conditions)
	}
}

const (
	generatedAllow = generatedruntime.ExistingBeforeCreateDecisionAllow
	generatedSkip  = generatedruntime.ExistingBeforeCreateDecisionSkip
)
