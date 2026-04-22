/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sddc

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

func TestSddcCreateOrUpdateRejectsCreateWithoutDisplayNameWhenUntracked(t *testing.T) {
	t.Parallel()

	resource := newSddcTestResource()
	resource.Spec.DisplayName = ""
	createCalled := false
	listCalled := false

	manager := newSddcRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Sddc]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateSddcRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when displayName is missing")
				return nil, nil
			},
			Fields: sddcCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListSddcsRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				listCalled = true
				t.Fatal("List() should not be called when displayName is missing")
				return nil, nil
			},
			Fields: sddcListFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want displayName requirement failure")
	}
	if err.Error() != "Sddc spec.displayName is required when no OCI identifier is recorded" {
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
	requireSddcCondition(t, resource, shared.Failed)
}

func TestSddcCreateOrUpdateCreatesAndRequeuesFromLifecycleFollowUp(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.sddc.oc1..created"

	resource := newSddcTestResource()
	calls := make([]string, 0, 3)
	var createRequest ocvpsdk.CreateSddcRequest
	var listRequest ocvpsdk.ListSddcsRequest
	listCalls := 0

	manager := newSddcRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Sddc]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateSddcRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				calls = append(calls, "create")
				createRequest = *request.(*ocvpsdk.CreateSddcRequest)
				return ocvpsdk.CreateSddcResponse{
					OpcWorkRequestId: common.String("ocid1.workrequest.oc1..create"),
				}, nil
			},
			Fields: sddcCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListSddcsRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				calls = append(calls, "list")
				listRequest = *request.(*ocvpsdk.ListSddcsRequest)
				listCalls++
				if listCalls < 3 {
					return ocvpsdk.ListSddcsResponse{
						SddcCollection: ocvpsdk.SddcCollection{
							Items: nil,
						},
					}, nil
				}
				return ocvpsdk.ListSddcsResponse{
					SddcCollection: ocvpsdk.SddcCollection{
						Items: []ocvpsdk.SddcSummary{
							observedSddcSummaryFromSpec(createdID, resource.Spec, "CREATING"),
						},
					},
				}, nil
			},
			Fields: sddcListFields(),
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
	if len(calls) != 4 || calls[0] != "list" || calls[1] != "list" || calls[2] != "create" || calls[3] != "list" {
		t.Fatalf("call order = %v, want [list list create list]", calls)
	}
	requireSddcStringPointer(t, "create request displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireSddcStringPointer(t, "create request compartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
	requireSddcStringPointer(t, "create request vmwareSoftwareVersion", createRequest.VmwareSoftwareVersion, resource.Spec.VmwareSoftwareVersion)
	requireSddcStringPointer(t, "create request sshAuthorizedKeys", createRequest.SshAuthorizedKeys, resource.Spec.SshAuthorizedKeys)
	if createRequest.HcxMode != ocvpsdk.HcxModesEnum(resource.Spec.HcxMode) {
		t.Fatalf("create request hcxMode = %q, want %q", createRequest.HcxMode, resource.Spec.HcxMode)
	}
	if createRequest.InitialConfiguration == nil || len(createRequest.InitialConfiguration.InitialClusterConfigurations) != 1 {
		t.Fatalf("create request initialConfiguration = %#v, want one initial cluster configuration", createRequest.InitialConfiguration)
	}
	requireSddcStringPointer(t, "list request displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	requireSddcStringPointer(t, "list request compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireSddcCondition(t, resource, shared.Provisioning)
	requireSddcOCID(t, resource, createdID)
	if resource.Status.LifecycleState != "CREATING" {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, "CREATING")
	}
}

func TestSddcCreateOrUpdateReusesMatchingListEntryWhenDisplayNamePresent(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.sddc.oc1..existing"

	resource := newSddcTestResource()
	createCalled := false
	getCalled := false
	var listRequest ocvpsdk.ListSddcsRequest

	manager := newSddcRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Sddc]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.CreateSddcRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				t.Fatal("Create() should not be called when list lookup finds a reusable sddc")
				return nil, nil
			},
			Fields: sddcCreateFields(),
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.ListSddcsRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest = *request.(*ocvpsdk.ListSddcsRequest)
				return ocvpsdk.ListSddcsResponse{
					SddcCollection: ocvpsdk.SddcCollection{
						Items: []ocvpsdk.SddcSummary{
							observedSddcSummaryFromSpec(existingID, resource.Spec, "ACTIVE"),
						},
					},
				}, nil
			},
			Fields: sddcListFields(),
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetSddcRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalled = true
				requireSddcStringPointer(t, "get request sddcId", request.(*ocvpsdk.GetSddcRequest).SddcId, existingID)
				return ocvpsdk.GetSddcResponse{
					Sddc: observedSddcFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
			Fields: sddcGetFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when list lookup reuses an existing sddc")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE list reuse")
	}
	if createCalled {
		t.Fatal("Create() should not be called when list lookup finds a reusable sddc")
	}
	if !getCalled {
		t.Fatal("Get() should be called after the identity lookup resolves an existing sddc ID")
	}
	requireSddcStringPointer(t, "list request displayName", listRequest.DisplayName, resource.Spec.DisplayName)
	requireSddcStringPointer(t, "list request compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
	requireSddcOCID(t, resource, existingID)
	if resource.Status.Id != existingID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, existingID)
	}
	requireSddcCondition(t, resource, shared.Active)
}

func TestSddcCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.sddc.oc1..existing"

	resource := newExistingSddcTestResource(existingID)
	resource.Spec.DisplayName = "sddc-renamed"

	getCalls := 0
	var updateRequest ocvpsdk.UpdateSddcRequest

	manager := newSddcRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Sddc]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetSddcRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireSddcStringPointer(t, "get request sddcId", request.(*ocvpsdk.GetSddcRequest).SddcId, existingID)
				live := observedSddcFromSpec(existingID, newSddcTestResource().Spec, "ACTIVE")
				if getCalls > 1 {
					live.DisplayName = common.String(resource.Spec.DisplayName)
				}
				return ocvpsdk.GetSddcResponse{Sddc: live}, nil
			},
			Fields: sddcGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.UpdateSddcRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*ocvpsdk.UpdateSddcRequest)
				return ocvpsdk.UpdateSddcResponse{
					Sddc: observedSddcFromSpec(existingID, resource.Spec, "ACTIVE"),
				}, nil
			},
			Fields: sddcUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should succeed when a mutable Sddc field changes")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when the update follow-up sees ACTIVE state")
	}
	if getCalls != 2 {
		t.Fatalf("get calls = %d, want 2", getCalls)
	}
	requireSddcStringPointer(t, "update request sddcId", updateRequest.SddcId, existingID)
	requireSddcStringPointer(t, "update request displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	requireSddcOCID(t, resource, existingID)
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
	requireSddcCondition(t, resource, shared.Active)
}

func TestSddcCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.sddc.oc1..existing"

	resource := newExistingSddcTestResource(existingID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"

	updateCalled := false
	manager := newSddcRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Sddc]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.GetSddcRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				requireSddcStringPointer(t, "get request sddcId", request.(*ocvpsdk.GetSddcRequest).SddcId, existingID)
				live := observedSddcFromSpec(existingID, newSddcTestResource().Spec, "ACTIVE")
				live.CompartmentId = common.String("ocid1.compartment.oc1..live")
				return ocvpsdk.GetSddcResponse{Sddc: live}, nil
			},
			Fields: sddcGetFields(),
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &ocvpsdk.UpdateSddcRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				updateCalled = true
				t.Fatal("Update() should not be called when create-only Sddc fields drift")
				return nil, nil
			},
			Fields: sddcUpdateFields(),
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "Sddc formal semantics require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit compartmentId replacement failure", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when create-only Sddc fields drift")
	}
	if updateCalled {
		t.Fatal("Update() should not be called when create-only Sddc fields drift")
	}
	requireSddcCondition(t, resource, shared.Failed)
}

func TestSddcDeleteConfirmsDeleteOutcomes(t *testing.T) {
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

			const existingID = "ocid1.sddc.oc1..existing"

			resource := newExistingSddcTestResource(existingID)
			getCalls := 0
			var deleteRequest ocvpsdk.DeleteSddcRequest

			manager := newSddcRuntimeTestManager(generatedruntime.Config[*ocvpv1beta1.Sddc]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &ocvpsdk.GetSddcRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						getCalls++
						requireSddcStringPointer(t, "get request sddcId", request.(*ocvpsdk.GetSddcRequest).SddcId, existingID)
						state := "ACTIVE"
						if getCalls > 1 {
							state = tc.confirmedState
						}
						return ocvpsdk.GetSddcResponse{
							Sddc: observedSddcFromSpec(existingID, resource.Spec, state),
						}, nil
					},
					Fields: sddcGetFields(),
				},
				Delete: &generatedruntime.Operation{
					NewRequest: func() any { return &ocvpsdk.DeleteSddcRequest{} },
					Call: func(_ context.Context, request any) (any, error) {
						deleteRequest = *request.(*ocvpsdk.DeleteSddcRequest)
						return ocvpsdk.DeleteSddcResponse{
							OpcWorkRequestId: common.String("ocid1.workrequest.oc1..delete"),
						}, nil
					},
					Fields: sddcDeleteFields(),
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
			requireSddcStringPointer(t, "delete request sddcId", deleteRequest.SddcId, existingID)
			requireSddcCondition(t, resource, shared.Terminating)
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

func newSddcRuntimeTestManager(cfg generatedruntime.Config[*ocvpv1beta1.Sddc]) *SddcServiceManager {
	manager := &SddcServiceManager{}
	hooks := SddcRuntimeHooks{
		Semantics:       newSddcRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*ocvpv1beta1.Sddc]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*ocvpv1beta1.Sddc]{},
		StatusHooks:     generatedruntime.StatusHooks[*ocvpv1beta1.Sddc]{},
		ParityHooks:     generatedruntime.ParityHooks[*ocvpv1beta1.Sddc]{},
		Create: sddcRuntimeTestOperation[ocvpsdk.CreateSddcRequest, ocvpsdk.CreateSddcResponse](
			sddcCreateFields(),
			cfg.Create,
		),
		Get: sddcRuntimeTestOperation[ocvpsdk.GetSddcRequest, ocvpsdk.GetSddcResponse](
			sddcGetFields(),
			cfg.Get,
		),
		List: sddcRuntimeTestOperation[ocvpsdk.ListSddcsRequest, ocvpsdk.ListSddcsResponse](
			sddcListFields(),
			cfg.List,
		),
		Update: sddcRuntimeTestOperation[ocvpsdk.UpdateSddcRequest, ocvpsdk.UpdateSddcResponse](
			sddcUpdateFields(),
			cfg.Update,
		),
		Delete: sddcRuntimeTestOperation[ocvpsdk.DeleteSddcRequest, ocvpsdk.DeleteSddcResponse](
			sddcDeleteFields(),
			cfg.Delete,
		),
		WrapGeneratedClient: []func(SddcServiceClient) SddcServiceClient{},
	}
	if cfg.Semantics != nil {
		hooks.Semantics = cfg.Semantics
	}
	applySddcRuntimeHooks(&hooks)
	config := buildSddcGeneratedRuntimeConfig(manager, hooks)
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
	delegate := defaultSddcServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*ocvpv1beta1.Sddc](config),
	}
	manager.client = wrapSddcGeneratedClient(hooks, delegate)
	return manager
}

func sddcRuntimeTestOperation[Req any, Resp any](
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
			return zero, fmt.Errorf("sddc test operation returned %T, want %T", response, zero)
		}
		return typed, nil
	}
	return op
}

func newSddcTestResource() *ocvpv1beta1.Sddc {
	return &ocvpv1beta1.Sddc{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sddc-sample",
			Namespace: "default",
		},
		Spec: ocvpv1beta1.SddcSpec{
			DisplayName:           "sddc-sample",
			VmwareSoftwareVersion: "7.0.0",
			CompartmentId:         "ocid1.compartment.oc1..example",
			HcxMode:               "DISABLED",
			SshAuthorizedKeys:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCexampleReplaceMe user@example",
			EsxiSoftwareVersion:   "7.0.0",
			IsSingleHostSddc:      false,
			InitialConfiguration:  newSddcInitialConfiguration(),
			FreeformTags:          map[string]string{"env": "dev"},
		},
	}
}

func newExistingSddcTestResource(existingID string) *ocvpv1beta1.Sddc {
	resource := newSddcTestResource()
	resource.Status = ocvpv1beta1.SddcStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func newSddcInitialConfiguration() ocvpv1beta1.SddcInitialConfiguration {
	return ocvpv1beta1.SddcInitialConfiguration{
		InitialClusterConfigurations: []ocvpv1beta1.SddcInitialConfigurationInitialClusterConfiguration{
			{
				DisplayName:               "management-cluster",
				VsphereType:               "MANAGEMENT",
				ComputeAvailabilityDomain: "Uocm:PHX-AD-1",
				EsxiHostsCount:            3,
				NetworkConfiguration: ocvpv1beta1.SddcInitialConfigurationInitialClusterConfigurationNetworkConfiguration{
					ProvisioningSubnetId: "ocid1.subnet.oc1..example",
					VmotionVlanId:        "ocid1.vlan.oc1..vmotion",
					VsanVlanId:           "ocid1.vlan.oc1..vsan",
					NsxVTepVlanId:        "ocid1.vlan.oc1..nsx-vtep",
					NsxEdgeVTepVlanId:    "ocid1.vlan.oc1..nsx-edge-vtep",
					VsphereVlanId:        "ocid1.vlan.oc1..vsphere",
					NsxEdgeUplink1VlanId: "ocid1.vlan.oc1..nsx-edge-uplink-1",
					NsxEdgeUplink2VlanId: "ocid1.vlan.oc1..nsx-edge-uplink-2",
				},
			},
		},
	}
}

func observedSddcFromSpec(id string, spec ocvpv1beta1.SddcSpec, lifecycleState string) ocvpsdk.Sddc {
	return ocvpsdk.Sddc{
		Id:                    common.String(id),
		DisplayName:           common.String(spec.DisplayName),
		VmwareSoftwareVersion: common.String(spec.VmwareSoftwareVersion),
		CompartmentId:         common.String(spec.CompartmentId),
		ClustersCount:         common.Int(len(spec.InitialConfiguration.InitialClusterConfigurations)),
		VcenterFqdn:           common.String("vcenter-sddc.example.oraclecloud.com"),
		NsxManagerFqdn:        common.String("nsx-sddc.example.oraclecloud.com"),
		VcenterPrivateIpId:    common.String("ocid1.privateip.oc1..vcenter"),
		NsxManagerPrivateIpId: common.String("ocid1.privateip.oc1..nsx"),
		SshAuthorizedKeys:     common.String(spec.SshAuthorizedKeys),
		HcxMode:               ocvpsdk.HcxModesEnum(spec.HcxMode),
		InitialConfiguration:  sdkInitialConfigurationFromSpec(spec.InitialConfiguration),
		FreeformTags:          map[string]string{"env": "dev"},
		DefinedTags:           map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		EsxiSoftwareVersion:   common.String(spec.EsxiSoftwareVersion),
		IsSingleHostSddc:      common.Bool(spec.IsSingleHostSddc),
		LifecycleState:        ocvpsdk.LifecycleStatesEnum(lifecycleState),
	}
}

func observedSddcSummaryFromSpec(id string, spec ocvpv1beta1.SddcSpec, lifecycleState string) ocvpsdk.SddcSummary {
	return ocvpsdk.SddcSummary{
		Id:                    common.String(id),
		DisplayName:           common.String(spec.DisplayName),
		VmwareSoftwareVersion: common.String(spec.VmwareSoftwareVersion),
		CompartmentId:         common.String(spec.CompartmentId),
		ClustersCount:         common.Int(len(spec.InitialConfiguration.InitialClusterConfigurations)),
		FreeformTags:          map[string]string{"env": "dev"},
		DefinedTags:           map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		HcxMode:               ocvpsdk.HcxModesEnum(spec.HcxMode),
		VcenterFqdn:           common.String("vcenter-sddc.example.oraclecloud.com"),
		NsxManagerFqdn:        common.String("nsx-sddc.example.oraclecloud.com"),
		LifecycleState:        ocvpsdk.LifecycleStatesEnum(lifecycleState),
		IsSingleHostSddc:      common.Bool(spec.IsSingleHostSddc),
	}
}

func sdkInitialConfigurationFromSpec(spec ocvpv1beta1.SddcInitialConfiguration) *ocvpsdk.InitialConfiguration {
	configurations := make([]ocvpsdk.InitialClusterConfiguration, 0, len(spec.InitialClusterConfigurations))
	for _, cfg := range spec.InitialClusterConfigurations {
		configurations = append(configurations, sdkInitialClusterConfigurationFromSpec(cfg))
	}
	return &ocvpsdk.InitialConfiguration{
		InitialClusterConfigurations: configurations,
	}
}

func sdkInitialClusterConfigurationFromSpec(spec ocvpv1beta1.SddcInitialConfigurationInitialClusterConfiguration) ocvpsdk.InitialClusterConfiguration {
	cluster := ocvpsdk.InitialClusterConfiguration{
		VsphereType:               ocvpsdk.VsphereTypesEnum(spec.VsphereType),
		ComputeAvailabilityDomain: common.String(spec.ComputeAvailabilityDomain),
		EsxiHostsCount:            common.Int(spec.EsxiHostsCount),
		NetworkConfiguration:      sdkNetworkConfigurationFromSpec(spec.NetworkConfiguration),
	}
	if spec.DisplayName != "" {
		cluster.DisplayName = common.String(spec.DisplayName)
	}
	if spec.InstanceDisplayNamePrefix != "" {
		cluster.InstanceDisplayNamePrefix = common.String(spec.InstanceDisplayNamePrefix)
	}
	if spec.InitialCommitment != "" {
		cluster.InitialCommitment = ocvpsdk.CommitmentEnum(spec.InitialCommitment)
	}
	if spec.WorkloadNetworkCidr != "" {
		cluster.WorkloadNetworkCidr = common.String(spec.WorkloadNetworkCidr)
	}
	if spec.InitialHostShapeName != "" {
		cluster.InitialHostShapeName = common.String(spec.InitialHostShapeName)
	}
	if spec.InitialHostOcpuCount != 0 {
		cluster.InitialHostOcpuCount = float32Ptr(spec.InitialHostOcpuCount)
	}
	if spec.IsShieldedInstanceEnabled {
		cluster.IsShieldedInstanceEnabled = common.Bool(spec.IsShieldedInstanceEnabled)
	}
	if spec.CapacityReservationId != "" {
		cluster.CapacityReservationId = common.String(spec.CapacityReservationId)
	}
	if len(spec.Datastores) > 0 {
		cluster.Datastores = make([]ocvpsdk.DatastoreInfo, 0, len(spec.Datastores))
		for _, datastore := range spec.Datastores {
			cluster.Datastores = append(cluster.Datastores, ocvpsdk.DatastoreInfo{
				BlockVolumeIds: append([]string(nil), datastore.BlockVolumeIds...),
				DatastoreType:  ocvpsdk.DatastoreTypesEnum(datastore.DatastoreType),
			})
		}
	}
	return cluster
}

func sdkNetworkConfigurationFromSpec(spec ocvpv1beta1.SddcInitialConfigurationInitialClusterConfigurationNetworkConfiguration) *ocvpsdk.NetworkConfiguration {
	network := &ocvpsdk.NetworkConfiguration{
		ProvisioningSubnetId: common.String(spec.ProvisioningSubnetId),
		VmotionVlanId:        common.String(spec.VmotionVlanId),
		VsanVlanId:           common.String(spec.VsanVlanId),
		NsxVTepVlanId:        common.String(spec.NsxVTepVlanId),
		NsxEdgeVTepVlanId:    common.String(spec.NsxEdgeVTepVlanId),
	}
	if spec.VsphereVlanId != "" {
		network.VsphereVlanId = common.String(spec.VsphereVlanId)
	}
	if spec.NsxEdgeUplink1VlanId != "" {
		network.NsxEdgeUplink1VlanId = common.String(spec.NsxEdgeUplink1VlanId)
	}
	if spec.NsxEdgeUplink2VlanId != "" {
		network.NsxEdgeUplink2VlanId = common.String(spec.NsxEdgeUplink2VlanId)
	}
	if spec.ReplicationVlanId != "" {
		network.ReplicationVlanId = common.String(spec.ReplicationVlanId)
	}
	if spec.ProvisioningVlanId != "" {
		network.ProvisioningVlanId = common.String(spec.ProvisioningVlanId)
	}
	if spec.HcxVlanId != "" {
		network.HcxVlanId = common.String(spec.HcxVlanId)
	}
	return network
}

func sddcCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSddcDetails", RequestName: "CreateSddcDetails", Contribution: "body"},
	}
}

func sddcGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SddcId", RequestName: "sddcId", Contribution: "path", PreferResourceID: true},
	}
}

func sddcListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "ComputeAvailabilityDomain", RequestName: "computeAvailabilityDomain", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func sddcUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SddcId", RequestName: "sddcId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSddcDetails", RequestName: "UpdateSddcDetails", Contribution: "body"},
	}
}

func sddcDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SddcId", RequestName: "sddcId", Contribution: "path", PreferResourceID: true},
	}
}

func requireSddcCondition(t *testing.T, resource *ocvpv1beta1.Sddc, want shared.OSOKConditionType) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 || conditions[len(conditions)-1].Type != want {
		t.Fatalf("status.conditions = %#v, want trailing %s condition", conditions, want)
	}
}

func requireSddcOCID(t *testing.T, resource *ocvpv1beta1.Sddc, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
}

func requireSddcStringPointer(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func float32Ptr(v float32) *float32 {
	return &v
}
