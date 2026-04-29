/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package esxihost

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocvpsdk "github.com/oracle/oci-go-sdk/v65/ocvp"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestEsxiHostCreateOrUpdateRejectsCreateWithoutDisplayNameWhenUntracked(t *testing.T) {
	t.Parallel()

	resource := newEsxiHostTestResource()
	resource.Spec.DisplayName = ""
	createCalled := false
	listCalled := false

	manager := newEsxiHostRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.EsxiHost]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateEsxiHostRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when displayName is missing")
				return nil, nil
			},
			Fields: esxiHostCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListEsxiHostsRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				listCalled = true
				t.Fatal("List() should not be called when displayName is missing")
				return nil, nil
			},
			Fields: esxiHostListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want displayName requirement failure")
	}
	if err.Error() != "EsxiHost spec.displayName is required when no OCI identifier is recorded" {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit displayName requirement", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when displayName is missing")
	}
	if createCalled {
		t.Fatal("Create() should not be called when displayName is missing")
	}
	if listCalled {
		t.Fatal("List() should not be called when displayName is missing")
	}
	requireEsxiHostCondition(t, resource, shared.Failed)
}

func TestEsxiHostCreateOrUpdateCreatesAndRequeuesFromLifecycleFollowUp(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.esxihost.oc1..created"

	resource := newEsxiHostTestResource()
	calls := make([]string, 0, 3)
	var createRequest ocvpsdk.CreateEsxiHostRequest
	var listRequest ocvpsdk.ListEsxiHostsRequest
	listCalls := 0

	manager := newEsxiHostRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.EsxiHost]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateEsxiHostRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				calls = append(calls, "create")
				createRequest = *request.(*ocvpsdk.CreateEsxiHostRequest)
				return ocvpsdk.CreateEsxiHostResponse{
					OpcWorkRequestId: common.String("ocid1.workrequest.oc1..create"),
				}, nil
			},
			Fields: esxiHostCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListEsxiHostsRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				calls = append(calls, "list")
				listRequest = *request.(*ocvpsdk.ListEsxiHostsRequest)
				listCalls++
				if listCalls == 1 {
					return ocvpsdk.ListEsxiHostsResponse{
						EsxiHostCollection: ocvpsdk.EsxiHostCollection{
							Items: nil,
						},
					}, nil
				}
				return ocvpsdk.ListEsxiHostsResponse{
					EsxiHostCollection: ocvpsdk.EsxiHostCollection{
						Items: []ocvpsdk.EsxiHostSummary{
							observedEsxiHostSummaryFromSpec(createdID, resource.Spec, "CREATING"),
						},
					},
				}, nil
			},
			Fields: esxiHostListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when create follow-up observes CREATING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle remains CREATING")
	}
	if len(calls) != 3 || calls[0] != "list" || calls[1] != "create" || calls[2] != "list" {
		t.Fatalf("call order = %v, want [list create list]", calls)
	}
	requireEsxiHostStringPointer(t, "create request clusterId", createRequest.ClusterId, resource.Spec.ClusterId)
	requireEsxiHostStringPointer(t, "create request displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireEsxiHostStringPointer(t, "create request computeAvailabilityDomain", createRequest.ComputeAvailabilityDomain, resource.Spec.ComputeAvailabilityDomain)
	requireEsxiHostStringPointer(t, "create request hostShapeName", createRequest.HostShapeName, resource.Spec.HostShapeName)
	requireEsxiHostFloat32Pointer(t, "create request hostOcpuCount", createRequest.HostOcpuCount, resource.Spec.HostOcpuCount)
	if createRequest.CurrentCommitment != ocvpsdk.CommitmentEnum(resource.Spec.CurrentCommitment) {
		t.Fatalf("create request currentCommitment = %q, want %q", createRequest.CurrentCommitment, resource.Spec.CurrentCommitment)
	}
	if createRequest.NextCommitment != ocvpsdk.CommitmentEnum(resource.Spec.NextCommitment) {
		t.Fatalf("create request nextCommitment = %q, want %q", createRequest.NextCommitment, resource.Spec.NextCommitment)
	}
	if createRequest.FreeformTags["env"] != "dev" {
		t.Fatalf("create request freeformTags = %#v, want env=dev", createRequest.FreeformTags)
	}
	requireEsxiHostStringPointer(t, "list request clusterId", listRequest.ClusterId, resource.Spec.ClusterId)
	requireEsxiHostStringPointer(t, "list request displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	requireEsxiHostCondition(t, resource, shared.Provisioning)
	requireEsxiHostOCID(t, resource, createdID)
	if resource.Status.LifecycleState != "CREATING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "CREATING")
	}
}

func TestEsxiHostCreateOrUpdateReusesMatchingListEntryWhenDisplayNamePresent(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.esxihost.oc1..existing"

	resource := newEsxiHostTestResource()
	createCalled := false
	getCalled := false
	var listRequest ocvpsdk.ListEsxiHostsRequest

	manager := newEsxiHostRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.EsxiHost]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateEsxiHostRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list lookup finds a reusable esxihost")
				return nil, nil
			},
			Fields: esxiHostCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListEsxiHostsRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*ocvpsdk.ListEsxiHostsRequest)
				return ocvpsdk.ListEsxiHostsResponse{
					EsxiHostCollection: ocvpsdk.EsxiHostCollection{
						Items: []ocvpsdk.EsxiHostSummary{
							observedEsxiHostSummaryFromSpec(existingID, resource.Spec, "ACTIVE"),
						},
					},
				}, nil
			},
			Fields: esxiHostListFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetEsxiHostRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				requireEsxiHostStringPointer(t, "get request esxiHostId", request.(*ocvpsdk.GetEsxiHostRequest).EsxiHostId, existingID)
				return ocvpsdk.GetEsxiHostResponse{
					EsxiHost: observedEsxiHostFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
			Fields: esxiHostGetFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list lookup reuses an existing esxihost")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE list reuse")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list lookup finds a reusable esxihost")
	}
	if !getCalled {
		t.Fatal("Get() should be called after the identity lookup resolves an existing esxihost ID")
	}
	requireEsxiHostStringPointer(t, "list request clusterId", listRequest.ClusterId, resource.Spec.ClusterId)
	requireEsxiHostStringPointer(t, "list request displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	requireEsxiHostOCID(t, resource, existingID)
	if resource.Status.Id != existingID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, existingID)
	}
	requireEsxiHostCondition(t, resource, shared.Active)
}

func TestEsxiHostCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.esxihost.oc1..existing"

	resource := newExistingEsxiHostTestResource(existingID)
	resource.Spec.DisplayName = "esxihost-renamed"
	resource.Spec.NextCommitment = string(ocvpsdk.CommitmentOneYear)
	resource.Spec.BillingDonorHostId = "ocid1.esxihost.oc1..donor"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	getCalls := 0
	var updateRequest ocvpsdk.UpdateEsxiHostRequest

	manager := newEsxiHostRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.EsxiHost]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetEsxiHostRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireEsxiHostStringPointer(t, "get request esxiHostId", request.(*ocvpsdk.GetEsxiHostRequest).EsxiHostId, existingID)
				live := observedEsxiHostFromSpec(existingID, newEsxiHostTestResource().Spec, "ACTIVE")
				if getCalls > 1 {
					live = observedEsxiHostFromSpec(existingID, resource.Spec, "ACTIVE")
				}
				return ocvpsdk.GetEsxiHostResponse{EsxiHost: live}, nil
			},
			Fields: esxiHostGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.UpdateEsxiHostRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*ocvpsdk.UpdateEsxiHostRequest)
				return ocvpsdk.UpdateEsxiHostResponse{
					EsxiHost: observedEsxiHostFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
			Fields: esxiHostUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when mutable EsxiHost fields change")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when the update follow-up sees ACTIVE state")
	}
	if getCalls != 2 {
		t.Fatalf("get calls = %d, want 2", getCalls)
	}
	requireEsxiHostStringPointer(t, "update request esxiHostId", updateRequest.EsxiHostId, existingID)
	requireEsxiHostStringPointer(t, "update request displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	requireEsxiHostStringPointer(t, "update request billingDonorHostId", updateRequest.BillingDonorHostId, resource.Spec.BillingDonorHostId)
	if updateRequest.NextCommitment != ocvpsdk.CommitmentEnum(resource.Spec.NextCommitment) {
		t.Fatalf("update request nextCommitment = %q, want %q", updateRequest.NextCommitment, resource.Spec.NextCommitment)
	}
	if updateRequest.FreeformTags["env"] != "prod" {
		t.Fatalf("update request freeformTags = %#v, want env=prod", updateRequest.FreeformTags)
	}
	requireEsxiHostOCID(t, resource, existingID)
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
	requireEsxiHostCondition(t, resource, shared.Active)
}

func TestEsxiHostCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.esxihost.oc1..existing"

	resource := newExistingEsxiHostTestResource(existingID)
	resource.Spec.ClusterId = "ocid1.cluster.oc1..desired"

	updateCalled := false
	manager := newEsxiHostRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.EsxiHost]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetEsxiHostRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				requireEsxiHostStringPointer(t, "get request esxiHostId", request.(*ocvpsdk.GetEsxiHostRequest).EsxiHostId, existingID)
				live := observedEsxiHostFromSpec(existingID, newEsxiHostTestResource().Spec, "ACTIVE")
				live.ClusterId = common.String("ocid1.cluster.oc1..live")
				return ocvpsdk.GetEsxiHostResponse{EsxiHost: live}, nil
			},
			Fields: esxiHostGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.UpdateEsxiHostRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				updateCalled = true
				t.Fatal("Update() should not be called when create-only EsxiHost fields drift")
				return nil, nil
			},
			Fields: esxiHostUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "EsxiHost formal semantics require replacement when clusterId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit clusterId replacement failure", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when create-only EsxiHost fields drift")
	}
	if updateCalled {
		t.Fatal("Update() should not be called when create-only EsxiHost fields drift")
	}
	requireEsxiHostCondition(t, resource, shared.Failed)
}

func TestEsxiHostCreateOrUpdateMapsFailedLifecycleToFailure(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.esxihost.oc1..existing"

	resource := newExistingEsxiHostTestResource(existingID)
	manager := newEsxiHostRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.EsxiHost]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetEsxiHostRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				requireEsxiHostStringPointer(t, "get request esxiHostId", request.(*ocvpsdk.GetEsxiHostRequest).EsxiHostId, existingID)
				return ocvpsdk.GetEsxiHostResponse{
					EsxiHost: observedEsxiHostFromSpec(existingID, resource.Spec, "FAILED"),
				}, nil
			},
			Fields: esxiHostGetFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when OCI returns FAILED")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue terminal failed lifecycle states")
	}
	requireEsxiHostCondition(t, resource, shared.Failed)
	if resource.Status.LifecycleState != "FAILED" {
		t.Fatalf("status.lifecycleState = %q, want FAILED", resource.Status.LifecycleState)
	}
}

func TestEsxiHostDeleteConfirmsDeleteOutcomes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		confirmedState   string
		wantDeleted      bool
		wantMessage      string
		wantDeletedStamp bool
	}{
		{
			name:             "deleting remains pending",
			confirmedState:   "DELETING",
			wantDeleted:      false,
			wantMessage:      "OCI resource delete is in progress",
			wantDeletedStamp: false,
		},
		{
			name:             "deleted confirms removal",
			confirmedState:   "DELETED",
			wantDeleted:      true,
			wantMessage:      "OCI resource deleted",
			wantDeletedStamp: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.esxihost.oc1..existing"

			resource := newExistingEsxiHostTestResource(existingID)
			getCalls := 0
			var deleteRequest ocvpsdk.DeleteEsxiHostRequest

			manager := newEsxiHostRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.EsxiHost]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &ocvpsdk.GetEsxiHostRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getCalls++
						requireEsxiHostStringPointer(t, "get request esxiHostId", request.(*ocvpsdk.GetEsxiHostRequest).EsxiHostId, existingID)
						state := "ACTIVE"
						if getCalls > 1 {
							state = tc.confirmedState
						}
						return ocvpsdk.GetEsxiHostResponse{
							EsxiHost: observedEsxiHostFromSpec(existingID, resource.Spec, state),
						}, nil
					},
					Fields: esxiHostGetFields(),
				},
				Delete: &generatedruntime.Operation{
					NewRequest: func() any { return &ocvpsdk.DeleteEsxiHostRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						deleteRequest = *request.(*ocvpsdk.DeleteEsxiHostRequest)
						return ocvpsdk.DeleteEsxiHostResponse{
							OpcWorkRequestId: common.String("ocid1.workrequest.oc1..delete"),
						}, nil
					},
					Fields: esxiHostDeleteFields(),
				},
			})

			deleted, err := manager.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted != tc.wantDeleted {
				t.Fatalf("Delete() deleted = %t, want %t", deleted, tc.wantDeleted)
			}
			if getCalls != 2 {
				t.Fatalf("get calls = %d, want 2", getCalls)
			}
			requireEsxiHostStringPointer(t, "delete request esxiHostId", deleteRequest.EsxiHostId, existingID)
			requireEsxiHostCondition(t, resource, shared.Terminating)
			if resource.Status.LifecycleState != tc.confirmedState {
				t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, tc.confirmedState)
			}
			if resource.Status.OsokStatus.Message != tc.wantMessage {
				t.Fatalf("status.message = %q, want %q", resource.Status.OsokStatus.Message, tc.wantMessage)
			}
			if tc.wantDeletedStamp {
				if resource.Status.OsokStatus.DeletedAt == nil {
					t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
				}
			} else if resource.Status.OsokStatus.DeletedAt != nil {
				t.Fatalf("status.deletedAt = %v, want nil while delete is still pending", resource.Status.OsokStatus.DeletedAt)
			}
		})
	}
}

func newEsxiHostRuntimeTestManager(cfg generatedruntime.Config[*ocvpv1beta1.EsxiHost]) *EsxiHostServiceManager {
	manager := &EsxiHostServiceManager{}
	hooks := EsxiHostRuntimeHooks{
		Semantics:       newEsxiHostRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*ocvpv1beta1.EsxiHost]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*ocvpv1beta1.EsxiHost]{},
		StatusHooks:     generatedruntime.StatusHooks[*ocvpv1beta1.EsxiHost]{},
		ParityHooks:     generatedruntime.ParityHooks[*ocvpv1beta1.EsxiHost]{},
		Create: esxiHostRuntimeTestOperation[ocvpsdk.CreateEsxiHostRequest, ocvpsdk.CreateEsxiHostResponse](
			esxiHostCreateFields(),
			cfg.Create,
		),
		Get: esxiHostRuntimeTestOperation[ocvpsdk.GetEsxiHostRequest, ocvpsdk.GetEsxiHostResponse](
			esxiHostGetFields(),
			cfg.Get,
		),
		List: esxiHostRuntimeTestOperation[ocvpsdk.ListEsxiHostsRequest, ocvpsdk.ListEsxiHostsResponse](
			esxiHostListFields(),
			cfg.List,
		),
		Update: esxiHostRuntimeTestOperation[ocvpsdk.UpdateEsxiHostRequest, ocvpsdk.UpdateEsxiHostResponse](
			esxiHostUpdateFields(),
			cfg.Update,
		),
		Delete: esxiHostRuntimeTestOperation[ocvpsdk.DeleteEsxiHostRequest, ocvpsdk.DeleteEsxiHostResponse](
			esxiHostDeleteFields(),
			cfg.Delete,
		),
		WrapGeneratedClient: []func(EsxiHostServiceClient) EsxiHostServiceClient{},
	}
	if cfg.Semantics != nil {
		hooks.Semantics = cfg.Semantics
	}
	applyEsxiHostRuntimeHooks(&hooks)
	config := buildEsxiHostGeneratedRuntimeConfig(manager, hooks)
	config.InitError = cfg.InitError
	if cfg.Create == nil {
		config.Create = nil
	}
	if cfg.Get == nil {
		config.Get = nil
		config.Read.Get = nil
	}
	if cfg.List == nil {
		config.List = nil
		config.Read.List = nil
	}
	if cfg.Update == nil {
		config.Update = nil
	}
	if cfg.Delete == nil {
		config.Delete = nil
	}
	delegate := defaultEsxiHostServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*ocvpv1beta1.EsxiHost](config),
	}
	manager.client = wrapEsxiHostGeneratedClient(hooks, delegate)
	return manager
}

func esxiHostRuntimeTestOperation[Req any, Resp any](
	defaultFields []generatedruntime.RequestField,
	override *generatedruntime.Operation,
) runtimeOperationHooks[Req, Resp] {
	op := runtimeOperationHooks[Req, Resp]{
		Fields: append([]generatedruntime.RequestField(nil), defaultFields...),
		Call: func(context.Context, Req) (Resp, error) {
			var zero Resp
			return zero, nil
		},
	}
	if override == nil {
		return op
	}
	if len(override.Fields) != 0 {
		op.Fields = append([]generatedruntime.RequestField(nil), override.Fields...)
	}
	op.Call = func(ctx context.Context, request Req) (Resp, error) {
		var zero Resp
		if override.Call == nil {
			return zero, nil
		}
		response, err := override.Call(ctx, &request)
		if err != nil {
			return zero, err
		}
		if response == nil {
			return zero, nil
		}
		typed, ok := response.(Resp)
		if !ok {
			return zero, fmt.Errorf("esxihost test operation returned %T, want %T", response, zero)
		}
		return typed, nil
	}
	return op
}

func newEsxiHostTestResource() *ocvpv1beta1.EsxiHost {
	return &ocvpv1beta1.EsxiHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "esxihost-sample",
			Namespace: "default",
		},
		Spec: ocvpv1beta1.EsxiHostSpec{
			ClusterId:                 "ocid1.cluster.oc1..example",
			DisplayName:               "esxihost-sample",
			BillingDonorHostId:        "ocid1.esxihost.oc1..billingdonor",
			CurrentCommitment:         string(ocvpsdk.CommitmentMonth),
			NextCommitment:            string(ocvpsdk.CommitmentMonth),
			ComputeAvailabilityDomain: "Uocm:PHX-AD-1",
			HostShapeName:             "BM.DenseIO2.52",
			HostOcpuCount:             32,
			CapacityReservationId:     "ocid1.capacityreservation.oc1..example",
			EsxiSoftwareVersion:       "7.0.0",
			FreeformTags:              map[string]string{"env": "dev"},
			DefinedTags:               map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func newExistingEsxiHostTestResource(existingID string) *ocvpv1beta1.EsxiHost {
	resource := newEsxiHostTestResource()
	resource.Status = ocvpv1beta1.EsxiHostStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedEsxiHostFromSpec(id string, spec ocvpv1beta1.EsxiHostSpec, lifecycleState string) ocvpsdk.EsxiHost {
	return ocvpsdk.EsxiHost{
		Id:                              common.String(id),
		DisplayName:                     common.String(spec.DisplayName),
		SddcId:                          common.String("ocid1.sddc.oc1..example"),
		ClusterId:                       common.String(spec.ClusterId),
		CurrentCommitment:               ocvpsdk.CommitmentEnum(spec.CurrentCommitment),
		NextCommitment:                  ocvpsdk.CommitmentEnum(spec.NextCommitment),
		VmwareSoftwareVersion:           common.String("vmware-version"),
		ComputeAvailabilityDomain:       common.String(spec.ComputeAvailabilityDomain),
		HostShapeName:                   common.String(spec.HostShapeName),
		FreeformTags:                    map[string]string{"env": spec.FreeformTags["env"]},
		DefinedTags:                     map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		CompartmentId:                   common.String("ocid1.compartment.oc1..example"),
		ComputeInstanceId:               common.String("ocid1.instance.oc1..example"),
		LifecycleState:                  ocvpsdk.LifecycleStatesEnum(lifecycleState),
		BillingDonorHostId:              common.String(spec.BillingDonorHostId),
		SwapBillingHostId:               common.String("ocid1.esxihost.oc1..swap"),
		IsBillingContinuationInProgress: common.Bool(false),
		IsBillingSwappingInProgress:     common.Bool(false),
		FailedEsxiHostId:                common.String(""),
		ReplacementEsxiHostId:           common.String(""),
		EsxiSoftwareVersion:             common.String(spec.EsxiSoftwareVersion),
		NonUpgradedEsxiHostId:           common.String(""),
		UpgradedReplacementEsxiHostId:   common.String(""),
		HostOcpuCount:                   common.Float32(spec.HostOcpuCount),
		CapacityReservationId:           common.String(spec.CapacityReservationId),
	}
}

func observedEsxiHostSummaryFromSpec(id string, spec ocvpv1beta1.EsxiHostSpec, lifecycleState string) ocvpsdk.EsxiHostSummary {
	return ocvpsdk.EsxiHostSummary{
		Id:                              common.String(id),
		SddcId:                          common.String("ocid1.sddc.oc1..example"),
		ClusterId:                       common.String(spec.ClusterId),
		CurrentCommitment:               ocvpsdk.CommitmentEnum(spec.CurrentCommitment),
		NextCommitment:                  ocvpsdk.CommitmentEnum(spec.NextCommitment),
		VmwareSoftwareVersion:           common.String("vmware-version"),
		ComputeAvailabilityDomain:       common.String(spec.ComputeAvailabilityDomain),
		HostShapeName:                   common.String(spec.HostShapeName),
		FreeformTags:                    map[string]string{"env": spec.FreeformTags["env"]},
		DefinedTags:                     map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		DisplayName:                     common.String(spec.DisplayName),
		CompartmentId:                   common.String("ocid1.compartment.oc1..example"),
		ComputeInstanceId:               common.String("ocid1.instance.oc1..example"),
		LifecycleState:                  ocvpsdk.LifecycleStatesEnum(lifecycleState),
		FailedEsxiHostId:                common.String(""),
		ReplacementEsxiHostId:           common.String(""),
		NonUpgradedEsxiHostId:           common.String(""),
		UpgradedReplacementEsxiHostId:   common.String(""),
		HostOcpuCount:                   common.Float32(spec.HostOcpuCount),
		BillingDonorHostId:              common.String(spec.BillingDonorHostId),
		SwapBillingHostId:               common.String("ocid1.esxihost.oc1..swap"),
		IsBillingContinuationInProgress: common.Bool(false),
		IsBillingSwappingInProgress:     common.Bool(false),
	}
}

func esxiHostCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateEsxiHostDetails", RequestName: "CreateEsxiHostDetails", Contribution: "body"},
	}
}

func esxiHostGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "EsxiHostId", RequestName: "esxiHostId", Contribution: "path", PreferResourceID: true},
	}
}

func esxiHostListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SddcId", RequestName: "sddcId", Contribution: "query"},
		{FieldName: "ClusterId", RequestName: "clusterId", Contribution: "query"},
		{FieldName: "ComputeInstanceId", RequestName: "computeInstanceId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "IsBillingDonorsOnly", RequestName: "isBillingDonorsOnly", Contribution: "query"},
		{FieldName: "IsSwapBillingOnly", RequestName: "isSwapBillingOnly", Contribution: "query"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
	}
}

func esxiHostUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "EsxiHostId", RequestName: "esxiHostId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateEsxiHostDetails", RequestName: "UpdateEsxiHostDetails", Contribution: "body"},
	}
}

func esxiHostDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "EsxiHostId", RequestName: "esxiHostId", Contribution: "path", PreferResourceID: true},
	}
}

func requireEsxiHostCondition(t *testing.T, resource *ocvpv1beta1.EsxiHost, want shared.OSOKConditionType) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 || conditions[len(conditions)-1].Type != want {
		t.Fatalf("status.conditions = %#v, want trailing %s condition", conditions, want)
	}
}

func requireEsxiHostOCID(t *testing.T, resource *ocvpv1beta1.EsxiHost, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
}

func requireEsxiHostStringPointer(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func requireEsxiHostFloat32Pointer(t *testing.T, label string, got *float32, want float32) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %f", label, got, want)
	}
}
