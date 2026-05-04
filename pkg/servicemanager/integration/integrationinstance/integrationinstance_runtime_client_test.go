/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package integrationinstance

import (
	"context"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	integrationsdk "github.com/oracle/oci-go-sdk/v65/integration"
	integrationv1beta1 "github.com/oracle/oci-service-operator/api/integration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestIntegrationInstanceRuntimeSemanticsEncodesModernLifecycleContract(t *testing.T) {
	t.Parallel()

	semantics := newIntegrationInstanceRuntimeSemantics()
	if semantics == nil {
		t.Fatal("newIntegrationInstanceRuntimeSemantics() = nil")
	}
	assertIntegrationInstanceStringEqual(t, "FormalService", semantics.FormalService, "integration")
	assertIntegrationInstanceStringEqual(t, "FormalSlug", semantics.FormalSlug, "integrationinstance")
	assertIntegrationInstanceStringEqual(t, "FinalizerPolicy", semantics.FinalizerPolicy, "retain-until-confirmed-delete")
	assertIntegrationInstanceStringEqual(t, "Delete.Policy", semantics.Delete.Policy, "required")
	assertIntegrationInstanceStringEqual(t, "CreateFollowUp.Strategy", semantics.CreateFollowUp.Strategy, "read-after-write")
	assertIntegrationInstanceStringEqual(t, "UpdateFollowUp.Strategy", semantics.UpdateFollowUp.Strategy, "read-after-write")
	assertIntegrationInstanceStringEqual(t, "DeleteFollowUp.Strategy", semantics.DeleteFollowUp.Strategy, "confirm-delete")

	assertIntegrationInstanceStringSliceEqual(t, "Lifecycle.ProvisioningStates", semantics.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertIntegrationInstanceStringSliceEqual(t, "Lifecycle.UpdatingStates", semantics.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertIntegrationInstanceStringSliceEqual(t, "Lifecycle.ActiveStates", semantics.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertIntegrationInstanceStringSliceEqual(t, "Delete.PendingStates", semantics.Delete.PendingStates, []string{"DELETING"})
	assertIntegrationInstanceStringSliceEqual(t, "Delete.TerminalStates", semantics.Delete.TerminalStates, []string{"DELETED"})
	assertIntegrationInstanceStringSliceEqual(t, "List.MatchFields", semantics.List.MatchFields, []string{"compartmentId", "displayName"})
	assertIntegrationInstanceStringSliceEqual(t, "Mutation.Mutable", semantics.Mutation.Mutable, []string{
		"displayName",
		"integrationInstanceType",
		"isByol",
		"messagePacks",
		"freeformTags",
		"definedTags",
		"securityAttributes",
		"isFileServerEnabled",
		"isVisualBuilderEnabled",
		"customEndpoint",
		"alternateCustomEndpoints",
		"networkEndpointDetails",
	})
	assertIntegrationInstanceStringSliceEqual(t, "Mutation.ForceNew", semantics.Mutation.ForceNew, []string{
		"compartmentId",
		"idcsAt",
		"consumptionModel",
		"isDisasterRecoveryEnabled",
		"shape",
		"domainId",
	})

	hooks := newIntegrationInstanceRuntimeTestHooks()
	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed IntegrationInstance semantics")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want safe bind guard")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create body builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update body builder")
	}
}

func TestIntegrationInstanceCreateUsesManualBodyAndListFollowUp(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.integrationinstance.oc1..created"
	resource := newIntegrationInstanceTestResource()
	resource.Spec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		NetworkEndpointType:         string(integrationsdk.NetworkEndpointTypePublic),
		AllowlistedHttpIps:          []string{"203.0.113.10"},
		IsIntegrationVcnAllowlisted: true,
		Runtime: integrationv1beta1.IntegrationInstanceNetworkEndpointDetailsRuntime{
			AllowlistedHttpIps: []string{"10.0.0.0/24"},
		},
	}
	hooks := newIntegrationInstanceRuntimeTestHooks()

	var createRequest integrationsdk.CreateIntegrationInstanceRequest
	getCalls := 0
	listCalls := 0
	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.CreateIntegrationInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*integrationsdk.CreateIntegrationInstanceRequest)
				return integrationsdk.CreateIntegrationInstanceResponse{
					OpcRequestId:     common.String("opc-create-1"),
					OpcWorkRequestId: common.String("wr-create-1"),
				}, nil
			},
			Fields: hooks.Create.Fields,
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				getCalls++
				t.Fatal("GetIntegrationInstance() should not be called before create follow-up resolves an OCID")
				return nil, nil
			},
			Fields: hooks.Get.Fields,
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.ListIntegrationInstancesRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listCalls++
				listRequest := request.(*integrationsdk.ListIntegrationInstancesRequest)
				requireIntegrationInstanceStringPtr(t, "list compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
				requireIntegrationInstanceStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
				if listCalls == 1 {
					return integrationsdk.ListIntegrationInstancesResponse{}, nil
				}
				return integrationsdk.ListIntegrationInstancesResponse{
					Items: []integrationsdk.IntegrationInstanceSummary{
						observedIntegrationInstanceSummaryFromSpec(createdID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateCreating),
					},
				}, nil
			},
			Fields: hooks.List.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while IntegrationInstance is creating")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while IntegrationInstance is creating")
	}
	if getCalls != 0 {
		t.Fatalf("GetIntegrationInstance() calls = %d, want 0", getCalls)
	}
	assertIntegrationInstanceCreateRequest(t, createRequest, resource)
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if resource.Status.LifecycleState != string(integrationsdk.IntegrationInstanceLifecycleStateCreating) {
		t.Fatalf("status.lifecycleState = %q, want CREATING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", resource.Status.OsokStatus.OpcRequestID)
	}
	requireIntegrationInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "CREATING", shared.OSOKAsyncClassPending, "wr-create-1")
}

func TestIntegrationInstanceCreateDecodesNetworkEndpointJSONData(t *testing.T) {
	t.Parallel()

	resource := newIntegrationInstanceTestResource()
	resource.Spec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		JsonData: `{"networkEndpointType":"PUBLIC","allowlistedHttpIps":["198.51.100.7"],"isIntegrationVcnAllowlisted":true}`,
	}

	body, err := buildIntegrationInstanceCreateBody(context.Background(), resource, "")
	if err != nil {
		t.Fatalf("buildIntegrationInstanceCreateBody() error = %v", err)
	}
	details := body.(integrationsdk.CreateIntegrationInstanceDetails)
	publicEndpoint, ok := details.NetworkEndpointDetails.(integrationsdk.PublicEndpointDetails)
	if !ok {
		t.Fatalf("networkEndpointDetails type = %T, want PublicEndpointDetails", details.NetworkEndpointDetails)
	}
	if len(publicEndpoint.AllowlistedHttpIps) != 1 || publicEndpoint.AllowlistedHttpIps[0] != "198.51.100.7" {
		t.Fatalf("allowlistedHttpIps = %#v, want [198.51.100.7]", publicEndpoint.AllowlistedHttpIps)
	}
	if publicEndpoint.IsIntegrationVcnAllowlisted == nil || !*publicEndpoint.IsIntegrationVcnAllowlisted {
		t.Fatalf("isIntegrationVcnAllowlisted = %#v, want true", publicEndpoint.IsIntegrationVcnAllowlisted)
	}
}

func TestIntegrationInstanceCreateRejectsUnsupportedNetworkEndpointJSONData(t *testing.T) {
	t.Parallel()

	resource := newIntegrationInstanceTestResource()
	resource.Spec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		JsonData: `{"networkEndpointType":"PRIVATE"}`,
	}

	_, err := buildIntegrationInstanceCreateBody(context.Background(), resource, "")
	if err == nil || !strings.Contains(err.Error(), `networkEndpointDetails.networkEndpointType "PRIVATE" is not supported`) {
		t.Fatalf("buildIntegrationInstanceCreateBody() error = %v, want unsupported PRIVATE endpoint failure", err)
	}
}

func TestIntegrationInstanceBindsExistingDisplayNameMatchWithoutCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..existing"
	resource := newIntegrationInstanceTestResource()
	hooks := newIntegrationInstanceRuntimeTestHooks()

	createCalled := false
	getCalls := 0
	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.CreateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				createCalled = true
				t.Fatal("CreateIntegrationInstance() should not be called when list returns a reusable match")
				return nil, nil
			},
			Fields: hooks.Create.Fields,
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				requireIntegrationInstanceStringPtr(t, "get integrationInstanceId", request.(*integrationsdk.GetIntegrationInstanceRequest).IntegrationInstanceId, existingID)
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.ListIntegrationInstancesRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listRequest := request.(*integrationsdk.ListIntegrationInstancesRequest)
				requireIntegrationInstanceStringPtr(t, "list compartmentId", listRequest.CompartmentId, resource.Spec.CompartmentId)
				requireIntegrationInstanceStringPtr(t, "list displayName", listRequest.DisplayName, resource.Spec.DisplayName)
				return integrationsdk.ListIntegrationInstancesResponse{
					Items: []integrationsdk.IntegrationInstanceSummary{
						observedIntegrationInstanceSummaryFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
					},
				}, nil
			},
			Fields: hooks.List.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after binding an ACTIVE IntegrationInstance")
	}
	if createCalled {
		t.Fatal("CreateIntegrationInstance() was called unexpectedly")
	}
	if getCalls != 1 {
		t.Fatalf("GetIntegrationInstance() calls = %d, want 1 live read after list bind", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestIntegrationInstanceBindFollowsListPagination(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..laterpage"
	resource := newIntegrationInstanceTestResource()
	hooks := newIntegrationInstanceRuntimeTestHooksWithOperations(func(hooks *IntegrationInstanceRuntimeHooks) {
		hooks.Create.Call = func(context.Context, integrationsdk.CreateIntegrationInstanceRequest) (integrationsdk.CreateIntegrationInstanceResponse, error) {
			t.Fatal("CreateIntegrationInstance() should not be called when a later list page returns a reusable match")
			return integrationsdk.CreateIntegrationInstanceResponse{}, nil
		}
		hooks.Get.Call = func(_ context.Context, request integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
			requireIntegrationInstanceStringPtr(t, "get integrationInstanceId", request.IntegrationInstanceId, existingID)
			return integrationsdk.GetIntegrationInstanceResponse{
				IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
			}, nil
		}
		listCalls := 0
		hooks.List.Call = func(_ context.Context, request integrationsdk.ListIntegrationInstancesRequest) (integrationsdk.ListIntegrationInstancesResponse, error) {
			listCalls++
			requireIntegrationInstanceStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireIntegrationInstanceStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			switch listCalls {
			case 1:
				if request.Page != nil {
					t.Fatalf("first list page token = %v, want nil", request.Page)
				}
				nonmatchingSpec := resource.Spec
				nonmatchingSpec.DisplayName = "other-integration"
				return integrationsdk.ListIntegrationInstancesResponse{
					Items: []integrationsdk.IntegrationInstanceSummary{
						observedIntegrationInstanceSummaryFromSpec("ocid1.integrationinstance.oc1..other", nonmatchingSpec, integrationsdk.IntegrationInstanceLifecycleStateActive),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireIntegrationInstanceStringPtr(t, "second list page token", request.Page, "page-2")
				return integrationsdk.ListIntegrationInstancesResponse{
					Items: []integrationsdk.IntegrationInstanceSummary{
						observedIntegrationInstanceSummaryFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
					},
				}, nil
			default:
				t.Fatalf("unexpected ListIntegrationInstances() call %d", listCalls)
				return integrationsdk.ListIntegrationInstancesResponse{}, nil
			}
		}
	})

	response, err := newIntegrationInstanceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful later-page bind", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestIntegrationInstanceListPaginationRejectsDuplicateMatches(t *testing.T) {
	t.Parallel()

	resource := newIntegrationInstanceTestResource()
	hooks := newIntegrationInstanceRuntimeTestHooksWithOperations(func(hooks *IntegrationInstanceRuntimeHooks) {
		hooks.Create.Call = func(context.Context, integrationsdk.CreateIntegrationInstanceRequest) (integrationsdk.CreateIntegrationInstanceResponse, error) {
			t.Fatal("CreateIntegrationInstance() should not be called when list pagination finds duplicate matches")
			return integrationsdk.CreateIntegrationInstanceResponse{}, nil
		}
		hooks.Get.Call = func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
			t.Fatal("GetIntegrationInstance() should not be called when list pagination finds duplicate matches")
			return integrationsdk.GetIntegrationInstanceResponse{}, nil
		}
		listCalls := 0
		hooks.List.Call = func(_ context.Context, request integrationsdk.ListIntegrationInstancesRequest) (integrationsdk.ListIntegrationInstancesResponse, error) {
			listCalls++
			switch listCalls {
			case 1:
				return integrationsdk.ListIntegrationInstancesResponse{
					Items: []integrationsdk.IntegrationInstanceSummary{
						observedIntegrationInstanceSummaryFromSpec("ocid1.integrationinstance.oc1..duplicate1", resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireIntegrationInstanceStringPtr(t, "second list page token", request.Page, "page-2")
				return integrationsdk.ListIntegrationInstancesResponse{
					Items: []integrationsdk.IntegrationInstanceSummary{
						observedIntegrationInstanceSummaryFromSpec("ocid1.integrationinstance.oc1..duplicate2", resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
					},
				}, nil
			default:
				t.Fatalf("unexpected ListIntegrationInstances() call %d", listCalls)
				return integrationsdk.ListIntegrationInstancesResponse{}, nil
			}
		}
	})

	response, err := newIntegrationInstanceRuntimeTestClient(hooks).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate list match failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful duplicate match response", response)
	}
}

func TestIntegrationInstanceNoopReconcileObservesActiveResourceWithoutUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..steady"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	hooks := newIntegrationInstanceRuntimeTestHooks()

	updateCalled := false
	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				updateCalled = true
				t.Fatal("UpdateIntegrationInstance() should not be called when observed state matches desired state")
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful steady observe", response)
	}
	if updateCalled {
		t.Fatal("UpdateIntegrationInstance() was called unexpectedly")
	}
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
}

func TestIntegrationInstanceNetworkEndpointJSONDataMatchingReadbackDoesNotUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..jsondata-match"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	resource.Spec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		JsonData: `{"networkEndpointType":"PUBLIC","allowlistedHttpIps":["198.51.100.7"],"isIntegrationVcnAllowlisted":true}`,
	}
	hooks := newIntegrationInstanceRuntimeTestHooks()

	updateCalled := false
	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				updateCalled = true
				t.Fatal("UpdateIntegrationInstance() should not be called when networkEndpointDetails jsonData matches readback")
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful steady observe", response)
	}
	if updateCalled {
		t.Fatal("UpdateIntegrationInstance() was called unexpectedly")
	}
}

func TestIntegrationInstanceNetworkEndpointJSONDataDriftUsesChangeAction(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..jsondata-change"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	currentSpec := resource.Spec
	currentSpec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		JsonData: `{"networkEndpointType":"PUBLIC","allowlistedHttpIps":["198.51.100.7"],"isIntegrationVcnAllowlisted":true}`,
	}
	resource.Spec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		JsonData: `{"networkEndpointType":"PUBLIC","allowlistedHttpIps":["198.51.100.8"],"isIntegrationVcnAllowlisted":true}`,
	}
	client, changeRequest := newIntegrationInstanceEndpointChangeTestClient(
		t,
		resource,
		existingID,
		currentSpec,
		integrationsdk.IntegrationInstanceLifecycleStateUpdating,
		"opc-change-1",
		"wr-change-1",
		"UpdateIntegrationInstance() should not be called for networkEndpointDetails-only drift",
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireIntegrationInstanceResponse(t, response, true, "successful pending network endpoint change")
	assertIntegrationInstanceNetworkEndpointChangeRequest(t, *changeRequest, existingID, "198.51.100.8")
	assertIntegrationInstanceStringEqual(t, "status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-change-1")
	requireIntegrationInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "UPDATING", shared.OSOKAsyncClassPending, "wr-change-1")
}

func TestIntegrationInstanceNetworkEndpointMatchingReadbackSkipsChangeAction(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..endpoint-noop"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	resource.Spec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		NetworkEndpointType: string(integrationsdk.NetworkEndpointTypePublic),
		AllowlistedHttpIps:  []string{"198.51.100.7"},
	}
	hooks := newIntegrationInstanceRuntimeTestHooks()

	getIntegrationInstance := func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
		return integrationsdk.GetIntegrationInstanceResponse{
			IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
		}, nil
	}
	changeCalled := false
	client := newIntegrationInstanceNetworkEndpointActionTestClient(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return getIntegrationInstance(ctx, *request.(*integrationsdk.GetIntegrationInstanceRequest))
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal("UpdateIntegrationInstance() should not be called when networkEndpointDetails matches")
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	}, getIntegrationInstance, func(context.Context, integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error) {
		changeCalled = true
		t.Fatal("ChangeIntegrationInstanceNetworkEndpoint() should not be called when networkEndpointDetails matches")
		return integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse{}, nil
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if changeCalled {
		t.Fatal("ChangeIntegrationInstanceNetworkEndpoint() was called unexpectedly")
	}
}

func TestIntegrationInstanceNetworkEndpointDefaultPublicReadbackMatchesClearedSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		endpoint integrationsdk.NetworkEndpointDetails
	}{
		{
			name:     "public discriminator only",
			endpoint: integrationsdk.PublicEndpointDetails{},
		},
		{
			name: "public discriminator with false integration VCN allowlist",
			endpoint: integrationsdk.PublicEndpointDetails{
				IsIntegrationVcnAllowlisted: common.Bool(false),
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.integrationinstance.oc1..endpoint-default-public"
			resource := newExistingIntegrationInstanceTestResource(existingID)
			hooks := newIntegrationInstanceRuntimeTestHooks()

			getIntegrationInstance := func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
				observed := observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive)
				observed.NetworkEndpointDetails = tc.endpoint
				return integrationsdk.GetIntegrationInstanceResponse{IntegrationInstance: observed}, nil
			}
			changeCalled := false
			client := newIntegrationInstanceNetworkEndpointActionTestClient(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
					Call: func(ctx context.Context, request any) (any, error) {
						return getIntegrationInstance(ctx, *request.(*integrationsdk.GetIntegrationInstanceRequest))
					},
					Fields: hooks.Get.Fields,
				},
				Update: &generatedruntime.Operation{
					NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
					Call: func(context.Context, any) (any, error) {
						t.Fatal("UpdateIntegrationInstance() should not be called when only default networkEndpointDetails are observed")
						return nil, nil
					},
					Fields: hooks.Update.Fields,
				},
			}, getIntegrationInstance, func(context.Context, integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error) {
				changeCalled = true
				t.Fatal("ChangeIntegrationInstanceNetworkEndpoint() should not be called for default public endpoint readback")
				return integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse{}, nil
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			requireIntegrationInstanceResponse(t, response, false, "successful default endpoint no-op")
			if changeCalled {
				t.Fatal("ChangeIntegrationInstanceNetworkEndpoint() was called unexpectedly")
			}
			if integrationInstanceNetworkEndpointMeaningful(resource.Status.NetworkEndpointDetails) {
				t.Fatalf("status.networkEndpointDetails = %#v, want normalized empty endpoint", resource.Status.NetworkEndpointDetails)
			}
		})
	}
}

func TestIntegrationInstanceNetworkEndpointClearUsesEmptyChangePayload(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..endpoint-clear"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	currentSpec := resource.Spec
	currentSpec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		NetworkEndpointType: string(integrationsdk.NetworkEndpointTypePublic),
		AllowlistedHttpIps:  []string{"198.51.100.7"},
	}
	resource.Status.NetworkEndpointDetails = currentSpec.NetworkEndpointDetails
	client, changeRequest := newIntegrationInstanceEndpointChangeTestClient(
		t,
		resource,
		existingID,
		currentSpec,
		integrationsdk.IntegrationInstanceLifecycleStateActive,
		"opc-clear-1",
		"wr-clear-1",
		"UpdateIntegrationInstance() should not be called to clear networkEndpointDetails",
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireIntegrationInstanceResponse(t, response, false, "successful endpoint clear")
	assertIntegrationInstanceNetworkEndpointClearRequest(t, *changeRequest, existingID)
	assertIntegrationInstanceNetworkEndpointCleared(t, resource)
}

func TestIntegrationInstanceNetworkEndpointClearAcceptsDefaultPublicFollowUp(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..endpoint-clear-default-public"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	currentSpec := resource.Spec
	currentSpec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		NetworkEndpointType: string(integrationsdk.NetworkEndpointTypePublic),
		AllowlistedHttpIps:  []string{"198.51.100.7"},
	}
	resource.Status.NetworkEndpointDetails = currentSpec.NetworkEndpointDetails
	hooks := newIntegrationInstanceRuntimeTestHooks()

	getCalls := 0
	getIntegrationInstance := func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
		getCalls++
		observed := observedIntegrationInstanceFromSpec(existingID, currentSpec, integrationsdk.IntegrationInstanceLifecycleStateActive)
		if getCalls > 1 {
			observed = observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive)
			observed.NetworkEndpointDetails = integrationsdk.PublicEndpointDetails{
				IsIntegrationVcnAllowlisted: common.Bool(false),
			}
		}
		return integrationsdk.GetIntegrationInstanceResponse{IntegrationInstance: observed}, nil
	}

	changeRequest := &integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest{}
	client := newIntegrationInstanceNetworkEndpointActionTestClient(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return getIntegrationInstance(ctx, *request.(*integrationsdk.GetIntegrationInstanceRequest))
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal("UpdateIntegrationInstance() should not be called to clear networkEndpointDetails")
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	}, getIntegrationInstance, func(_ context.Context, request integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error) {
		*changeRequest = request
		return integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse{
			OpcRequestId:     common.String("opc-clear-1"),
			OpcWorkRequestId: common.String("wr-clear-1"),
		}, nil
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireIntegrationInstanceResponse(t, response, false, "successful endpoint clear with default public readback")
	assertIntegrationInstanceNetworkEndpointClearRequest(t, *changeRequest, existingID)
	assertIntegrationInstanceNetworkEndpointCleared(t, resource)
}

func TestIntegrationInstanceNetworkEndpointAllowlistDriftFromDefaultPublicUsesChangeAction(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..endpoint-default-public-drift"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	resource.Spec.NetworkEndpointDetails = integrationv1beta1.IntegrationInstanceNetworkEndpointDetails{
		NetworkEndpointType: string(integrationsdk.NetworkEndpointTypePublic),
		AllowlistedHttpIps:  []string{"198.51.100.8"},
	}
	hooks := newIntegrationInstanceRuntimeTestHooks()

	getCalls := 0
	getIntegrationInstance := func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
		getCalls++
		observed := observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive)
		if getCalls == 1 {
			observed.NetworkEndpointDetails = integrationsdk.PublicEndpointDetails{
				IsIntegrationVcnAllowlisted: common.Bool(false),
			}
		} else {
			observed.LifecycleState = integrationsdk.IntegrationInstanceLifecycleStateUpdating
		}
		return integrationsdk.GetIntegrationInstanceResponse{IntegrationInstance: observed}, nil
	}

	changeRequest := &integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest{}
	client := newIntegrationInstanceNetworkEndpointActionTestClient(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return getIntegrationInstance(ctx, *request.(*integrationsdk.GetIntegrationInstanceRequest))
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal("UpdateIntegrationInstance() should not be called for networkEndpointDetails drift")
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	}, getIntegrationInstance, func(_ context.Context, request integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error) {
		*changeRequest = request
		return integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse{
			OpcRequestId:     common.String("opc-change-1"),
			OpcWorkRequestId: common.String("wr-change-1"),
		}, nil
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireIntegrationInstanceResponse(t, response, true, "successful pending network endpoint change")
	assertIntegrationInstanceNetworkEndpointChangeRequest(t, *changeRequest, existingID, "198.51.100.8")
	requireIntegrationInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "UPDATING", shared.OSOKAsyncClassPending, "wr-change-1")
}

func TestIntegrationInstanceUpdateProjectsOnlySupportedMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..update"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	resource.Spec.DisplayName = "updated-integration"
	resource.Spec.IsByol = false
	resource.Spec.MessagePacks = 2
	resource.Spec.FreeformTags = map[string]string{"env": "prod", "owner": "osok"}
	resource.Spec.SecurityAttributes = map[string]shared.MapValue{
		"Oracle-ZPR": {
			"MaxEgressCount": "42",
		},
	}
	resource.Spec.CustomEndpoint = integrationv1beta1.IntegrationInstanceCustomEndpoint{
		Hostname:            "oic.example.com",
		CertificateSecretId: "ocid1.vaultsecret.oc1..cert",
	}
	hooks := newIntegrationInstanceRuntimeTestHooks()

	getCalls := 0
	var updateRequest integrationsdk.UpdateIntegrationInstanceRequest
	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				getCalls++
				spec := newIntegrationInstanceTestResource().Spec
				spec.IsByol = true
				spec.MessagePacks = 1
				lifecycle := integrationsdk.IntegrationInstanceLifecycleStateActive
				if getCalls > 1 {
					spec = resource.Spec
					lifecycle = integrationsdk.IntegrationInstanceLifecycleStateUpdating
				}
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, spec, lifecycle),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*integrationsdk.UpdateIntegrationInstanceRequest)
				return integrationsdk.UpdateIntegrationInstanceResponse{
					OpcRequestId:     common.String("opc-update-1"),
					OpcWorkRequestId: common.String("wr-update-1"),
				}, nil
			},
			Fields: hooks.Update.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success while IntegrationInstance is updating")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while IntegrationInstance is updating")
	}
	requireIntegrationInstanceStringPtr(t, "update integrationInstanceId", updateRequest.IntegrationInstanceId, existingID)
	assertIntegrationInstanceUpdateRequest(t, updateRequest, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
	requireIntegrationInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, "UPDATING", shared.OSOKAsyncClassPending, "wr-update-1")
}

func TestIntegrationInstanceRejectsCreateOnlyShapeDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..shape-drift"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	resource.Spec.Shape = string(integrationsdk.IntegrationInstanceShapeProduction)
	hooks := newIntegrationInstanceRuntimeTestHooks()

	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, newIntegrationInstanceTestResource().Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal("UpdateIntegrationInstance() should not be called for create-only shape drift")
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	})

	_, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when shape changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want shape replacement failure", err)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
}

func TestIntegrationInstanceDeleteWaitsForLifecycleConfirmation(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..delete"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	hooks := newIntegrationInstanceRuntimeTestHooks()

	getCalls := 0
	var deleteRequest integrationsdk.DeleteIntegrationInstanceRequest
	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				getCalls++
				lifecycle := integrationsdk.IntegrationInstanceLifecycleStateActive
				if getCalls > 1 {
					lifecycle = integrationsdk.IntegrationInstanceLifecycleStateDeleting
				}
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, lifecycle),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.DeleteIntegrationInstanceRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*integrationsdk.DeleteIntegrationInstanceRequest)
				return integrationsdk.DeleteIntegrationInstanceResponse{
					OpcRequestId:     common.String("opc-delete-1"),
					OpcWorkRequestId: common.String("wr-delete-1"),
				}, nil
			},
			Fields: hooks.Delete.Fields,
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI delete is pending")
	}
	requireIntegrationInstanceStringPtr(t, "delete integrationInstanceId", deleteRequest.IntegrationInstanceId, existingID)
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Terminating)
	}
	if resource.Status.LifecycleState != string(integrationsdk.IntegrationInstanceLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", resource.Status.OsokStatus.OpcRequestID)
	}
	requireIntegrationInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "", shared.OSOKAsyncClassPending, "wr-delete-1")
}

func TestIntegrationInstanceDeleteReleasesFinalizerAfterTerminalReadback(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..deleted"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	hooks := newIntegrationInstanceRuntimeTestHooks()

	deleteCalled := false
	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateDeleted),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.DeleteIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				deleteCalled = true
				t.Fatal("DeleteIntegrationInstance() should not be called after terminal readback")
				return nil, nil
			},
			Fields: hooks.Delete.Fields,
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after DELETED readback")
	}
	if deleteCalled {
		t.Fatal("DeleteIntegrationInstance() was called unexpectedly")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

func TestIntegrationInstanceDeleteRejectsAuthShapedDeleteNotFound(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..auth-delete"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-delete-auth-1"

	hooks := newIntegrationInstanceRuntimeTestHooksWithOperations(func(hooks *IntegrationInstanceRuntimeHooks) {
		hooks.Get.Call = func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
			return integrationsdk.GetIntegrationInstanceResponse{
				IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
			}, nil
		}
		hooks.Delete.Call = func(context.Context, integrationsdk.DeleteIntegrationInstanceRequest) (integrationsdk.DeleteIntegrationInstanceResponse, error) {
			return integrationsdk.DeleteIntegrationInstanceResponse{}, serviceErr
		}
	})

	deleted, err := newIntegrationInstanceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped delete failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped delete error")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped delete error", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-auth-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-auth-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestIntegrationInstanceDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..auth-confirm"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-auth-1"

	hooks := newIntegrationInstanceRuntimeTestHooksWithOperations(func(hooks *IntegrationInstanceRuntimeHooks) {
		getCalls := 0
		hooks.Get.Call = func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
			getCalls++
			if getCalls < 3 {
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, resource.Spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
				}, nil
			}
			return integrationsdk.GetIntegrationInstanceResponse{}, serviceErr
		}
		hooks.Delete.Call = func(context.Context, integrationsdk.DeleteIntegrationInstanceRequest) (integrationsdk.DeleteIntegrationInstanceResponse, error) {
			return integrationsdk.DeleteIntegrationInstanceResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		}
	})

	deleted, err := newIntegrationInstanceRuntimeTestClient(hooks).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped confirm-read failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm-read error")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped confirm-read error", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-confirm-auth-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-confirm-auth-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestIntegrationInstanceOCIErrorRecordsRequestID(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.integrationinstance.oc1..error"
	resource := newExistingIntegrationInstanceTestResource(existingID)
	resource.Spec.DisplayName = "update-fails"
	hooks := newIntegrationInstanceRuntimeTestHooks()

	manager := newIntegrationInstanceRuntimeTestManager(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				spec := newIntegrationInstanceTestResource().Spec
				return integrationsdk.GetIntegrationInstanceResponse{
					IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, spec, integrationsdk.IntegrationInstanceLifecycleStateActive),
				}, nil
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				return integrationsdk.UpdateIntegrationInstanceResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "update failed")
			},
			Fields: hooks.Update.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI update failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report unsuccessful after OCI update failure")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", got, shared.Failed)
	}
}

func newIntegrationInstanceRuntimeTestHooks() IntegrationInstanceRuntimeHooks {
	hooks := newIntegrationInstanceDefaultRuntimeHooks(integrationsdk.IntegrationInstanceClient{})
	applyIntegrationInstanceRuntimeHooks(&hooks)
	return hooks
}

func newIntegrationInstanceRuntimeTestHooksWithOperations(
	mutate func(*IntegrationInstanceRuntimeHooks),
) IntegrationInstanceRuntimeHooks {
	hooks := newIntegrationInstanceDefaultRuntimeHooks(integrationsdk.IntegrationInstanceClient{})
	if mutate != nil {
		mutate(&hooks)
	}
	applyIntegrationInstanceRuntimeHooks(&hooks)
	return hooks
}

func newIntegrationInstanceRuntimeTestClient(hooks IntegrationInstanceRuntimeHooks) IntegrationInstanceServiceClient {
	manager := &IntegrationInstanceServiceManager{}
	delegate := defaultIntegrationInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*integrationv1beta1.IntegrationInstance](
			buildIntegrationInstanceGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapIntegrationInstanceGeneratedClient(hooks, delegate)
}

func newIntegrationInstanceNetworkEndpointActionTestClient(
	cfg generatedruntime.Config[*integrationv1beta1.IntegrationInstance],
	getIntegrationInstance func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error),
	changeNetworkEndpoint func(context.Context, integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error),
) IntegrationInstanceServiceClient {
	manager := newIntegrationInstanceRuntimeTestManager(cfg)
	return newIntegrationInstanceNetworkEndpointActionClient(
		manager.client,
		getIntegrationInstance,
		changeNetworkEndpoint,
		manager.Log,
	)
}

func newIntegrationInstanceEndpointChangeTestClient(
	t *testing.T,
	resource *integrationv1beta1.IntegrationInstance,
	existingID string,
	currentSpec integrationv1beta1.IntegrationInstanceSpec,
	followUpLifecycle integrationsdk.IntegrationInstanceLifecycleStateEnum,
	opcRequestID string,
	workRequestID string,
	updateFatalMessage string,
) (IntegrationInstanceServiceClient, *integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) {
	t.Helper()

	hooks := newIntegrationInstanceRuntimeTestHooks()
	getCalls := 0
	getIntegrationInstance := func(context.Context, integrationsdk.GetIntegrationInstanceRequest) (integrationsdk.GetIntegrationInstanceResponse, error) {
		getCalls++
		spec := currentSpec
		lifecycle := integrationsdk.IntegrationInstanceLifecycleStateActive
		if getCalls > 1 {
			spec = resource.Spec
			lifecycle = followUpLifecycle
		}
		return integrationsdk.GetIntegrationInstanceResponse{
			IntegrationInstance: observedIntegrationInstanceFromSpec(existingID, spec, lifecycle),
		}, nil
	}

	changeRequest := &integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest{}
	client := newIntegrationInstanceNetworkEndpointActionTestClient(generatedruntime.Config[*integrationv1beta1.IntegrationInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.GetIntegrationInstanceRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return getIntegrationInstance(ctx, *request.(*integrationsdk.GetIntegrationInstanceRequest))
			},
			Fields: hooks.Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &integrationsdk.UpdateIntegrationInstanceRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal(updateFatalMessage)
				return nil, nil
			},
			Fields: hooks.Update.Fields,
		},
	}, getIntegrationInstance, func(_ context.Context, request integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest) (integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse, error) {
		*changeRequest = request
		return integrationsdk.ChangeIntegrationInstanceNetworkEndpointResponse{
			OpcRequestId:     common.String(opcRequestID),
			OpcWorkRequestId: common.String(workRequestID),
		}, nil
	})

	return client, changeRequest
}

func newIntegrationInstanceRuntimeTestManager(
	cfg generatedruntime.Config[*integrationv1beta1.IntegrationInstance],
) *IntegrationInstanceServiceManager {
	hooks := newIntegrationInstanceRuntimeTestHooks()
	if cfg.Kind == "" {
		cfg.Kind = "IntegrationInstance"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "IntegrationInstance"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = hooks.Semantics
	}
	if cfg.Identity.GuardExistingBeforeCreate == nil {
		cfg.Identity.GuardExistingBeforeCreate = hooks.Identity.GuardExistingBeforeCreate
	}
	if cfg.BuildCreateBody == nil {
		cfg.BuildCreateBody = hooks.BuildCreateBody
	}
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = hooks.BuildUpdateBody
	}
	if cfg.ParityHooks.NormalizeDesiredState == nil {
		cfg.ParityHooks.NormalizeDesiredState = hooks.ParityHooks.NormalizeDesiredState
	}
	if cfg.ParityHooks.ValidateCreateOnlyDrift == nil {
		cfg.ParityHooks.ValidateCreateOnlyDrift = hooks.ParityHooks.ValidateCreateOnlyDrift
	}

	return &IntegrationInstanceServiceManager{
		client: defaultIntegrationInstanceServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*integrationv1beta1.IntegrationInstance](cfg),
		},
	}
}

func newIntegrationInstanceTestResource() *integrationv1beta1.IntegrationInstance {
	return &integrationv1beta1.IntegrationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-sample",
			Namespace: "default",
		},
		Spec: integrationv1beta1.IntegrationInstanceSpec{
			DisplayName:             "integration-sample",
			CompartmentId:           "ocid1.compartment.oc1..example",
			IntegrationInstanceType: string(integrationsdk.CreateIntegrationInstanceDetailsIntegrationInstanceTypeEnterprise),
			IsByol:                  false,
			MessagePacks:            1,
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
			IsVisualBuilderEnabled: true,
			IsFileServerEnabled:    true,
			Shape:                  string(integrationsdk.CreateIntegrationInstanceDetailsShapeDevelopment),
		},
	}
}

func newExistingIntegrationInstanceTestResource(existingID string) *integrationv1beta1.IntegrationInstance {
	resource := newIntegrationInstanceTestResource()
	resource.Status = integrationv1beta1.IntegrationInstanceStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func observedIntegrationInstanceFromSpec(
	id string,
	spec integrationv1beta1.IntegrationInstanceSpec,
	lifecycleState integrationsdk.IntegrationInstanceLifecycleStateEnum,
) integrationsdk.IntegrationInstance {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}

	return integrationsdk.IntegrationInstance{
		Id:                        common.String(id),
		DisplayName:               common.String(spec.DisplayName),
		CompartmentId:             common.String(spec.CompartmentId),
		IntegrationInstanceType:   integrationsdk.IntegrationInstanceIntegrationInstanceTypeEnum(spec.IntegrationInstanceType),
		IsByol:                    common.Bool(spec.IsByol),
		InstanceUrl:               common.String("https://integration.example.com"),
		MessagePacks:              common.Int(spec.MessagePacks),
		TimeCreated:               now,
		TimeUpdated:               now,
		LifecycleState:            lifecycleState,
		LifecycleDetails:          common.String(string(lifecycleState)),
		StateMessage:              common.String(string(lifecycleState)),
		FreeformTags:              maps.Clone(spec.FreeformTags),
		DefinedTags:               integrationInstanceMapValueMap(spec.DefinedTags),
		SecurityAttributes:        integrationInstanceMapValueMap(spec.SecurityAttributes),
		IsFileServerEnabled:       common.Bool(spec.IsFileServerEnabled),
		IsVisualBuilderEnabled:    common.Bool(spec.IsVisualBuilderEnabled),
		CustomEndpoint:            observedIntegrationInstanceCustomEndpoint(spec.CustomEndpoint),
		AlternateCustomEndpoints:  observedIntegrationInstanceAlternateCustomEndpoints(spec.AlternateCustomEndpoints),
		ConsumptionModel:          integrationsdk.IntegrationInstanceConsumptionModelEnum(spec.ConsumptionModel),
		NetworkEndpointDetails:    observedIntegrationInstanceNetworkEndpoint(spec.NetworkEndpointDetails),
		Shape:                     integrationsdk.IntegrationInstanceShapeEnum(spec.Shape),
		IsDisasterRecoveryEnabled: common.Bool(spec.IsDisasterRecoveryEnabled),
	}
}

func observedIntegrationInstanceSummaryFromSpec(
	id string,
	spec integrationv1beta1.IntegrationInstanceSpec,
	lifecycleState integrationsdk.IntegrationInstanceLifecycleStateEnum,
) integrationsdk.IntegrationInstanceSummary {
	observed := observedIntegrationInstanceFromSpec(id, spec, lifecycleState)
	return integrationsdk.IntegrationInstanceSummary{
		Id:                        observed.Id,
		DisplayName:               observed.DisplayName,
		CompartmentId:             observed.CompartmentId,
		IntegrationInstanceType:   integrationsdk.IntegrationInstanceSummaryIntegrationInstanceTypeEnum(observed.IntegrationInstanceType),
		IsByol:                    observed.IsByol,
		InstanceUrl:               observed.InstanceUrl,
		MessagePacks:              observed.MessagePacks,
		TimeCreated:               observed.TimeCreated,
		TimeUpdated:               observed.TimeUpdated,
		LifecycleState:            integrationsdk.IntegrationInstanceSummaryLifecycleStateEnum(observed.LifecycleState),
		LifecycleDetails:          observed.LifecycleDetails,
		StateMessage:              observed.StateMessage,
		FreeformTags:              observed.FreeformTags,
		DefinedTags:               observed.DefinedTags,
		SecurityAttributes:        observed.SecurityAttributes,
		IsFileServerEnabled:       observed.IsFileServerEnabled,
		IsVisualBuilderEnabled:    observed.IsVisualBuilderEnabled,
		CustomEndpoint:            observed.CustomEndpoint,
		AlternateCustomEndpoints:  observed.AlternateCustomEndpoints,
		ConsumptionModel:          integrationsdk.IntegrationInstanceSummaryConsumptionModelEnum(observed.ConsumptionModel),
		NetworkEndpointDetails:    observed.NetworkEndpointDetails,
		Shape:                     integrationsdk.IntegrationInstanceSummaryShapeEnum(observed.Shape),
		IsDisasterRecoveryEnabled: observed.IsDisasterRecoveryEnabled,
	}
}

func observedIntegrationInstanceCustomEndpoint(
	spec integrationv1beta1.IntegrationInstanceCustomEndpoint,
) *integrationsdk.CustomEndpointDetails {
	if !integrationInstanceCustomEndpointMeaningful(spec) {
		return nil
	}
	return &integrationsdk.CustomEndpointDetails{
		Hostname:            common.String(spec.Hostname),
		CertificateSecretId: integrationInstanceStringPtr(spec.CertificateSecretId),
	}
}

func observedIntegrationInstanceAlternateCustomEndpoints(
	spec []integrationv1beta1.IntegrationInstanceAlternateCustomEndpoint,
) []integrationsdk.CustomEndpointDetails {
	if len(spec) == 0 {
		return nil
	}
	endpoints := make([]integrationsdk.CustomEndpointDetails, 0, len(spec))
	for _, endpoint := range spec {
		endpoints = append(endpoints, integrationsdk.CustomEndpointDetails{
			Hostname:            common.String(endpoint.Hostname),
			CertificateSecretId: integrationInstanceStringPtr(endpoint.CertificateSecretId),
		})
	}
	return endpoints
}

func observedIntegrationInstanceNetworkEndpoint(
	spec integrationv1beta1.IntegrationInstanceNetworkEndpointDetails,
) integrationsdk.NetworkEndpointDetails {
	endpoint, err := integrationInstanceNetworkEndpointDetails(spec)
	if err != nil {
		return nil
	}
	return endpoint
}

func assertIntegrationInstanceCreateRequest(
	t *testing.T,
	createRequest integrationsdk.CreateIntegrationInstanceRequest,
	resource *integrationv1beta1.IntegrationInstance,
) {
	t.Helper()

	requireIntegrationInstanceStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireIntegrationInstanceStringPtr(t, "create compartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
	if createRequest.IsByol == nil || *createRequest.IsByol != resource.Spec.IsByol {
		t.Fatalf("create isByol = %#v, want %t", createRequest.IsByol, resource.Spec.IsByol)
	}
	if createRequest.NetworkEndpointDetails == nil {
		t.Fatal("create networkEndpointDetails = nil, want public endpoint details")
	}
	publicEndpoint, ok := createRequest.NetworkEndpointDetails.(integrationsdk.PublicEndpointDetails)
	if !ok {
		t.Fatalf("create networkEndpointDetails type = %T, want PublicEndpointDetails", createRequest.NetworkEndpointDetails)
	}
	if len(publicEndpoint.AllowlistedHttpIps) != 1 || publicEndpoint.AllowlistedHttpIps[0] != "203.0.113.10" {
		t.Fatalf("create allowlistedHttpIps = %#v, want [203.0.113.10]", publicEndpoint.AllowlistedHttpIps)
	}
	if publicEndpoint.Runtime == nil || len(publicEndpoint.Runtime.AllowlistedHttpIps) != 1 || publicEndpoint.Runtime.AllowlistedHttpIps[0] != "10.0.0.0/24" {
		t.Fatalf("create runtime allowlist = %#v, want 10.0.0.0/24", publicEndpoint.Runtime)
	}
}

func assertIntegrationInstanceUpdateRequest(
	t *testing.T,
	updateRequest integrationsdk.UpdateIntegrationInstanceRequest,
	resource *integrationv1beta1.IntegrationInstance,
) {
	t.Helper()

	requireIntegrationInstanceStringPtr(t, "update displayName", updateRequest.DisplayName, resource.Spec.DisplayName)
	if updateRequest.IsByol == nil || *updateRequest.IsByol != resource.Spec.IsByol {
		t.Fatalf("update isByol = %#v, want %t", updateRequest.IsByol, resource.Spec.IsByol)
	}
	if updateRequest.MessagePacks == nil || *updateRequest.MessagePacks != resource.Spec.MessagePacks {
		t.Fatalf("update messagePacks = %#v, want %d", updateRequest.MessagePacks, resource.Spec.MessagePacks)
	}
	if !maps.Equal(updateRequest.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", updateRequest.FreeformTags, resource.Spec.FreeformTags)
	}
	if updateRequest.CustomEndpoint == nil {
		t.Fatal("update customEndpoint = nil, want configured endpoint")
	}
	requireIntegrationInstanceStringPtr(t, "update customEndpoint.hostname", updateRequest.CustomEndpoint.Hostname, resource.Spec.CustomEndpoint.Hostname)
}

func requireIntegrationInstanceResponse(
	t *testing.T,
	response servicemanager.OSOKResponse,
	wantRequeue bool,
	wantDescription string,
) {
	t.Helper()

	if !response.IsSuccessful || response.ShouldRequeue != wantRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want %s", response, wantDescription)
	}
}

func assertIntegrationInstanceNetworkEndpointChangeRequest(
	t *testing.T,
	changeRequest integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest,
	existingID string,
	wantAllowlistedIP string,
) {
	t.Helper()

	requireIntegrationInstanceStringPtr(t, "change integrationInstanceId", changeRequest.IntegrationInstanceId, existingID)
	requireIntegrationInstanceRetryToken(t, "change", changeRequest.OpcRetryToken)
	publicEndpoint, ok := changeRequest.NetworkEndpointDetails.(integrationsdk.PublicEndpointDetails)
	if !ok {
		t.Fatalf("change networkEndpointDetails type = %T, want PublicEndpointDetails", changeRequest.NetworkEndpointDetails)
	}
	if len(publicEndpoint.AllowlistedHttpIps) != 1 || publicEndpoint.AllowlistedHttpIps[0] != wantAllowlistedIP {
		t.Fatalf("change allowlistedHttpIps = %#v, want [%s]", publicEndpoint.AllowlistedHttpIps, wantAllowlistedIP)
	}
}

func assertIntegrationInstanceNetworkEndpointClearRequest(
	t *testing.T,
	changeRequest integrationsdk.ChangeIntegrationInstanceNetworkEndpointRequest,
	existingID string,
) {
	t.Helper()

	requireIntegrationInstanceStringPtr(t, "clear integrationInstanceId", changeRequest.IntegrationInstanceId, existingID)
	requireIntegrationInstanceRetryToken(t, "clear", changeRequest.OpcRetryToken)
	if changeRequest.NetworkEndpointDetails != nil {
		t.Fatalf("clear networkEndpointDetails = %#v, want nil empty change payload", changeRequest.NetworkEndpointDetails)
	}
}

func assertIntegrationInstanceNetworkEndpointCleared(
	t *testing.T,
	resource *integrationv1beta1.IntegrationInstance,
) {
	t.Helper()

	assertIntegrationInstanceStringEqual(t, "status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-clear-1")
	if integrationInstanceNetworkEndpointMeaningful(resource.Status.NetworkEndpointDetails) {
		t.Fatalf("status.networkEndpointDetails = %#v, want cleared", resource.Status.NetworkEndpointDetails)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared after active readback", resource.Status.OsokStatus.Async.Current)
	}
}

func requireIntegrationInstanceRetryToken(t *testing.T, label string, retryToken *string) {
	t.Helper()
	if retryToken == nil || strings.TrimSpace(*retryToken) == "" {
		t.Fatalf("%s opcRetryToken is empty, want deterministic retry token", label)
	}
}

func requireIntegrationInstanceStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func requireIntegrationInstanceAsyncCurrent(
	t *testing.T,
	resource *integrationv1beta1.IntegrationInstance,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want lifecycle tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func assertIntegrationInstanceStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("%s = %#v, want %#v", label, got, want)
		}
	}
}

func assertIntegrationInstanceStringEqual(t *testing.T, label string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}
