/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odaprivateendpointattachment

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestOdaPrivateEndpointAttachmentRuntimeHooksConfigureModernSemantics(t *testing.T) {
	hooks := newOdaPrivateEndpointAttachmentDefaultRuntimeHooks(odasdk.ManagementClient{})
	applyOdaPrivateEndpointAttachmentRuntimeHookConfig(&hooks, nil, nil)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed lifecycle semantics")
	}
	if hooks.Semantics.FormalService != "oda" || hooks.Semantics.FormalSlug != "odaprivateendpointattachment" {
		t.Fatalf("formal binding = %s/%s, want oda/odaprivateendpointattachment", hooks.Semantics.FormalService, hooks.Semantics.FormalSlug)
	}
	if hooks.Semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", hooks.Semantics.FinalizerPolicy)
	}
	if hooks.Semantics.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", hooks.Semantics.Delete.Policy)
	}
	if !slices.Contains(hooks.Semantics.Lifecycle.ActiveStates, string(odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive)) {
		t.Fatalf("ActiveStates = %#v, want ACTIVE", hooks.Semantics.Lifecycle.ActiveStates)
	}
	for _, field := range []string{"odaInstanceId", "odaPrivateEndpointId"} {
		if !slices.Contains(hooks.Semantics.Mutation.ForceNew, field) {
			t.Fatalf("Mutation.ForceNew = %#v, want %s", hooks.Semantics.Mutation.ForceNew, field)
		}
		if !slices.Contains(hooks.Semantics.List.MatchFields, field) {
			t.Fatalf("List.MatchFields = %#v, want %s", hooks.Semantics.List.MatchFields, field)
		}
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("BuildCreateBody = nil, want resource-local create request body")
	}
	if hooks.Identity.Resolve == nil || hooks.Identity.RecordPath == nil || hooks.Identity.LookupExisting == nil {
		t.Fatalf("Identity hooks incomplete: %#v", hooks.Identity)
	}
}

func TestOdaPrivateEndpointAttachmentCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newOdaPrivateEndpointAttachmentResource()
	client := &fakeOdaPrivateEndpointAttachmentOCIClient{
		parentGetResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: odasdk.OdaPrivateEndpoint{
				Id:            common.String(resource.Spec.OdaPrivateEndpointId),
				CompartmentId: common.String("ocid1.compartment.oc1..aaaa"),
			}},
		},
		listResponses: []odasdk.ListOdaPrivateEndpointAttachmentsResponse{
			{OdaPrivateEndpointAttachmentCollection: odasdk.OdaPrivateEndpointAttachmentCollection{}},
		},
		createResponses: []odasdk.CreateOdaPrivateEndpointAttachmentResponse{
			{OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachment("ocid1.attachment.created", resource, odasdk.OdaPrivateEndpointAttachmentLifecycleStateCreating)},
		},
		getResponses: []odasdk.GetOdaPrivateEndpointAttachmentResponse{
			{OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachment("ocid1.attachment.created", resource, odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive)},
		},
	}

	response, err := newOdaPrivateEndpointAttachmentServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if len(client.parentGetRequests) != 1 {
		t.Fatalf("Parent get requests = %d, want 1", len(client.parentGetRequests))
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("List requests = %d, want 1", len(client.listRequests))
	}
	requireStringPointer(t, "list compartmentId", client.listRequests[0].CompartmentId, "ocid1.compartment.oc1..aaaa")
	requireStringPointer(t, "list odaPrivateEndpointId", client.listRequests[0].OdaPrivateEndpointId, resource.Spec.OdaPrivateEndpointId)
	if len(client.createRequests) != 1 {
		t.Fatalf("Create requests = %d, want 1", len(client.createRequests))
	}
	createRequest := client.createRequests[0]
	requireStringPointer(t, "create odaInstanceId", createRequest.OdaInstanceId, resource.Spec.OdaInstanceId)
	requireStringPointer(t, "create odaPrivateEndpointId", createRequest.OdaPrivateEndpointId, resource.Spec.OdaPrivateEndpointId)
	if createRequest.OpcRetryToken == nil || strings.TrimSpace(*createRequest.OpcRetryToken) == "" {
		t.Fatal("Create request OpcRetryToken is empty")
	}
	requireProjectedOdaPrivateEndpointAttachment(t, resource, "ocid1.attachment.created", odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive)
	requireLastCondition(t, resource, shared.Active)
}

func TestOdaPrivateEndpointAttachmentCreateOrUpdateBindsExistingResourceBeforeCreate(t *testing.T) {
	resource := newOdaPrivateEndpointAttachmentResource()
	client := &fakeOdaPrivateEndpointAttachmentOCIClient{
		parentGetResponses: []odasdk.GetOdaPrivateEndpointResponse{
			{OdaPrivateEndpoint: odasdk.OdaPrivateEndpoint{
				Id:            common.String(resource.Spec.OdaPrivateEndpointId),
				CompartmentId: common.String("ocid1.compartment.oc1..aaaa"),
			}},
		},
		listResponses: []odasdk.ListOdaPrivateEndpointAttachmentsResponse{
			{
				OdaPrivateEndpointAttachmentCollection: odasdk.OdaPrivateEndpointAttachmentCollection{
					Items: []odasdk.OdaPrivateEndpointAttachmentSummary{
						newOdaPrivateEndpointAttachmentSummary("ocid1.attachment.existing", resource, odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive),
					},
				},
			},
		},
		getResponses: []odasdk.GetOdaPrivateEndpointAttachmentResponse{
			{OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachment("ocid1.attachment.existing", resource, odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive)},
		},
	}

	response, err := newOdaPrivateEndpointAttachmentServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("Create requests = %d, want 0 for existing bind", len(client.createRequests))
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("Attachment get requests = %d, want 1", len(client.getRequests))
	}
	requireStringPointer(t, "get id", client.getRequests[0].OdaPrivateEndpointAttachmentId, "ocid1.attachment.existing")
	requireProjectedOdaPrivateEndpointAttachment(t, resource, "ocid1.attachment.existing", odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive)
	requireLastCondition(t, resource, shared.Active)
}

func TestOdaPrivateEndpointAttachmentCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	resource := newOdaPrivateEndpointAttachmentResource()
	resource.Spec.OdaInstanceId = "ocid1.odainstance.oc1..new"
	recordOdaPrivateEndpointAttachmentID(resource, "ocid1.attachment.force-new")
	client := &fakeOdaPrivateEndpointAttachmentOCIClient{
		getResponses: []odasdk.GetOdaPrivateEndpointAttachmentResponse{
			{
				OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachmentWithIDs(
					"ocid1.attachment.force-new",
					"ocid1.odainstance.oc1..old",
					resource.Spec.OdaPrivateEndpointId,
					"ocid1.compartment.oc1..aaaa",
					odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive,
				),
			},
		},
	}

	response, err := newOdaPrivateEndpointAttachmentServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when odaInstanceId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want odaInstanceId force-new drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("Create requests = %d, want 0 after force-new rejection", len(client.createRequests))
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestOdaPrivateEndpointAttachmentCreateOrUpdateMapsLifecycleStates(t *testing.T) {
	tests := []struct {
		name          string
		state         odasdk.OdaPrivateEndpointAttachmentLifecycleStateEnum
		wantCondition shared.OSOKConditionType
		wantRequeue   bool
		wantSuccess   bool
	}{
		{name: "creating", state: odasdk.OdaPrivateEndpointAttachmentLifecycleStateCreating, wantCondition: shared.Provisioning, wantRequeue: true, wantSuccess: true},
		{name: "updating", state: odasdk.OdaPrivateEndpointAttachmentLifecycleStateUpdating, wantCondition: shared.Updating, wantRequeue: true, wantSuccess: true},
		{name: "active", state: odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive, wantCondition: shared.Active, wantRequeue: false, wantSuccess: true},
		{name: "deleting", state: odasdk.OdaPrivateEndpointAttachmentLifecycleStateDeleting, wantCondition: shared.Terminating, wantRequeue: true, wantSuccess: true},
		{name: "failed", state: odasdk.OdaPrivateEndpointAttachmentLifecycleStateFailed, wantCondition: shared.Failed, wantRequeue: false, wantSuccess: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resource := newOdaPrivateEndpointAttachmentResource()
			recordOdaPrivateEndpointAttachmentID(resource, "ocid1.attachment.lifecycle")
			client := &fakeOdaPrivateEndpointAttachmentOCIClient{
				getResponses: []odasdk.GetOdaPrivateEndpointAttachmentResponse{
					{OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachment("ocid1.attachment.lifecycle", resource, tc.state)},
				},
			}

			response, err := newOdaPrivateEndpointAttachmentServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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

func TestOdaPrivateEndpointAttachmentDeleteRetainsFinalizerUntilDeleteIsConfirmed(t *testing.T) {
	resource := newOdaPrivateEndpointAttachmentResource()
	recordOdaPrivateEndpointAttachmentID(resource, "ocid1.attachment.delete")
	client := &fakeOdaPrivateEndpointAttachmentOCIClient{
		getResponses: []odasdk.GetOdaPrivateEndpointAttachmentResponse{
			{OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachment("ocid1.attachment.delete", resource, odasdk.OdaPrivateEndpointAttachmentLifecycleStateActive)},
			{OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachment("ocid1.attachment.delete", resource, odasdk.OdaPrivateEndpointAttachmentLifecycleStateDeleting)},
		},
		deleteResponses: []odasdk.DeleteOdaPrivateEndpointAttachmentResponse{{}},
	}

	deleted, err := newOdaPrivateEndpointAttachmentServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("Delete requests = %d, want 1", len(client.deleteRequests))
	}
	requireStringPointer(t, "delete id", client.deleteRequests[0].OdaPrivateEndpointAttachmentId, "ocid1.attachment.delete")
	requireLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("DeletedAt is set before delete confirmation")
	}
}

func TestOdaPrivateEndpointAttachmentDeleteReleasesFinalizerAfterTerminalDelete(t *testing.T) {
	resource := newOdaPrivateEndpointAttachmentResource()
	recordOdaPrivateEndpointAttachmentID(resource, "ocid1.attachment.deleted")
	client := &fakeOdaPrivateEndpointAttachmentOCIClient{
		getResponses: []odasdk.GetOdaPrivateEndpointAttachmentResponse{
			{OdaPrivateEndpointAttachment: newOdaPrivateEndpointAttachment("ocid1.attachment.deleted", resource, odasdk.OdaPrivateEndpointAttachmentLifecycleStateDeleted)},
		},
	}

	deleted, err := newOdaPrivateEndpointAttachmentServiceClientWithOCIClient(client).Delete(context.Background(), resource)
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

type fakeOdaPrivateEndpointAttachmentOCIClient struct {
	parentGetRequests []odasdk.GetOdaPrivateEndpointRequest
	createRequests    []odasdk.CreateOdaPrivateEndpointAttachmentRequest
	getRequests       []odasdk.GetOdaPrivateEndpointAttachmentRequest
	listRequests      []odasdk.ListOdaPrivateEndpointAttachmentsRequest
	deleteRequests    []odasdk.DeleteOdaPrivateEndpointAttachmentRequest

	parentGetResponses []odasdk.GetOdaPrivateEndpointResponse
	createResponses    []odasdk.CreateOdaPrivateEndpointAttachmentResponse
	getResponses       []odasdk.GetOdaPrivateEndpointAttachmentResponse
	listResponses      []odasdk.ListOdaPrivateEndpointAttachmentsResponse
	deleteResponses    []odasdk.DeleteOdaPrivateEndpointAttachmentResponse
}

func (c *fakeOdaPrivateEndpointAttachmentOCIClient) GetOdaPrivateEndpoint(_ context.Context, request odasdk.GetOdaPrivateEndpointRequest) (odasdk.GetOdaPrivateEndpointResponse, error) {
	c.parentGetRequests = append(c.parentGetRequests, request)
	if len(c.parentGetResponses) == 0 {
		return odasdk.GetOdaPrivateEndpointResponse{}, fmt.Errorf("unexpected GetOdaPrivateEndpoint call")
	}
	response := c.parentGetResponses[0]
	c.parentGetResponses = c.parentGetResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointAttachmentOCIClient) CreateOdaPrivateEndpointAttachment(_ context.Context, request odasdk.CreateOdaPrivateEndpointAttachmentRequest) (odasdk.CreateOdaPrivateEndpointAttachmentResponse, error) {
	c.createRequests = append(c.createRequests, request)
	if len(c.createResponses) == 0 {
		return odasdk.CreateOdaPrivateEndpointAttachmentResponse{}, fmt.Errorf("unexpected CreateOdaPrivateEndpointAttachment call")
	}
	response := c.createResponses[0]
	c.createResponses = c.createResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointAttachmentOCIClient) GetOdaPrivateEndpointAttachment(_ context.Context, request odasdk.GetOdaPrivateEndpointAttachmentRequest) (odasdk.GetOdaPrivateEndpointAttachmentResponse, error) {
	c.getRequests = append(c.getRequests, request)
	if len(c.getResponses) == 0 {
		return odasdk.GetOdaPrivateEndpointAttachmentResponse{}, fmt.Errorf("unexpected GetOdaPrivateEndpointAttachment call")
	}
	response := c.getResponses[0]
	c.getResponses = c.getResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointAttachmentOCIClient) ListOdaPrivateEndpointAttachments(_ context.Context, request odasdk.ListOdaPrivateEndpointAttachmentsRequest) (odasdk.ListOdaPrivateEndpointAttachmentsResponse, error) {
	c.listRequests = append(c.listRequests, request)
	if len(c.listResponses) == 0 {
		return odasdk.ListOdaPrivateEndpointAttachmentsResponse{}, fmt.Errorf("unexpected ListOdaPrivateEndpointAttachments call")
	}
	response := c.listResponses[0]
	c.listResponses = c.listResponses[1:]
	return response, nil
}

func (c *fakeOdaPrivateEndpointAttachmentOCIClient) DeleteOdaPrivateEndpointAttachment(_ context.Context, request odasdk.DeleteOdaPrivateEndpointAttachmentRequest) (odasdk.DeleteOdaPrivateEndpointAttachmentResponse, error) {
	c.deleteRequests = append(c.deleteRequests, request)
	if len(c.deleteResponses) == 0 {
		return odasdk.DeleteOdaPrivateEndpointAttachmentResponse{}, fmt.Errorf("unexpected DeleteOdaPrivateEndpointAttachment call")
	}
	response := c.deleteResponses[0]
	c.deleteResponses = c.deleteResponses[1:]
	return response, nil
}

func newOdaPrivateEndpointAttachmentResource() *odav1beta1.OdaPrivateEndpointAttachment {
	return &odav1beta1.OdaPrivateEndpointAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "private-endpoint-attachment",
			Namespace: "default",
			UID:       types.UID("uid-oda-private-endpoint-attachment"),
		},
		Spec: odav1beta1.OdaPrivateEndpointAttachmentSpec{
			OdaInstanceId:        "ocid1.odainstance.oc1..aaaa",
			OdaPrivateEndpointId: "ocid1.odaprivateendpoint.oc1..aaaa",
		},
	}
}

func newOdaPrivateEndpointAttachment(id string, resource *odav1beta1.OdaPrivateEndpointAttachment, state odasdk.OdaPrivateEndpointAttachmentLifecycleStateEnum) odasdk.OdaPrivateEndpointAttachment {
	return newOdaPrivateEndpointAttachmentWithIDs(id, resource.Spec.OdaInstanceId, resource.Spec.OdaPrivateEndpointId, "ocid1.compartment.oc1..aaaa", state)
}

func newOdaPrivateEndpointAttachmentWithIDs(
	id string,
	odaInstanceID string,
	odaPrivateEndpointID string,
	compartmentID string,
	state odasdk.OdaPrivateEndpointAttachmentLifecycleStateEnum,
) odasdk.OdaPrivateEndpointAttachment {
	return odasdk.OdaPrivateEndpointAttachment{
		Id:                   common.String(id),
		OdaInstanceId:        common.String(odaInstanceID),
		OdaPrivateEndpointId: common.String(odaPrivateEndpointID),
		CompartmentId:        common.String(compartmentID),
		LifecycleState:       state,
	}
}

func newOdaPrivateEndpointAttachmentSummary(id string, resource *odav1beta1.OdaPrivateEndpointAttachment, state odasdk.OdaPrivateEndpointAttachmentLifecycleStateEnum) odasdk.OdaPrivateEndpointAttachmentSummary {
	return odasdk.OdaPrivateEndpointAttachmentSummary{
		Id:                   common.String(id),
		OdaInstanceId:        common.String(resource.Spec.OdaInstanceId),
		OdaPrivateEndpointId: common.String(resource.Spec.OdaPrivateEndpointId),
		CompartmentId:        common.String("ocid1.compartment.oc1..aaaa"),
		LifecycleState:       state,
	}
}

func recordOdaPrivateEndpointAttachmentID(resource *odav1beta1.OdaPrivateEndpointAttachment, id string) {
	resource.Status.Id = id
	resource.Status.OdaInstanceId = resource.Spec.OdaInstanceId
	resource.Status.OdaPrivateEndpointId = resource.Spec.OdaPrivateEndpointId
	resource.Status.CompartmentId = "ocid1.compartment.oc1..aaaa"
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func requireProjectedOdaPrivateEndpointAttachment(t *testing.T, resource *odav1beta1.OdaPrivateEndpointAttachment, id string, state odasdk.OdaPrivateEndpointAttachmentLifecycleStateEnum) {
	t.Helper()
	if resource.Status.Id != id {
		t.Fatalf("Status.Id = %q, want %q", resource.Status.Id, id)
	}
	if string(resource.Status.OsokStatus.Ocid) != id {
		t.Fatalf("Status.OsokStatus.Ocid = %q, want %q", resource.Status.OsokStatus.Ocid, id)
	}
	if resource.Status.OdaInstanceId != resource.Spec.OdaInstanceId {
		t.Fatalf("Status.OdaInstanceId = %q, want %q", resource.Status.OdaInstanceId, resource.Spec.OdaInstanceId)
	}
	if resource.Status.OdaPrivateEndpointId != resource.Spec.OdaPrivateEndpointId {
		t.Fatalf("Status.OdaPrivateEndpointId = %q, want %q", resource.Status.OdaPrivateEndpointId, resource.Spec.OdaPrivateEndpointId)
	}
	if resource.Status.CompartmentId != "ocid1.compartment.oc1..aaaa" {
		t.Fatalf("Status.CompartmentId = %q, want ocid1.compartment.oc1..aaaa", resource.Status.CompartmentId)
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

func requireLastCondition(t *testing.T, resource *odav1beta1.OdaPrivateEndpointAttachment, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status conditions empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s; conditions=%#v", got, want, conditions)
	}
}
