package occcapacityrequest

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	capacitymanagementsdk "github.com/oracle/oci-go-sdk/v65/capacitymanagement"
	"github.com/oracle/oci-go-sdk/v65/common"
	capacitymanagementv1beta1 "github.com/oracle/oci-service-operator/api/capacitymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeOccCapacityRequestOCIClient struct {
	createFn func(context.Context, capacitymanagementsdk.CreateOccCapacityRequestRequest) (capacitymanagementsdk.CreateOccCapacityRequestResponse, error)
	getFn    func(context.Context, capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error)
	listFn   func(context.Context, capacitymanagementsdk.ListOccCapacityRequestsRequest) (capacitymanagementsdk.ListOccCapacityRequestsResponse, error)
	updateFn func(context.Context, capacitymanagementsdk.UpdateOccCapacityRequestRequest) (capacitymanagementsdk.UpdateOccCapacityRequestResponse, error)
	deleteFn func(context.Context, capacitymanagementsdk.DeleteOccCapacityRequestRequest) (capacitymanagementsdk.DeleteOccCapacityRequestResponse, error)
}

func (f *fakeOccCapacityRequestOCIClient) CreateOccCapacityRequest(
	ctx context.Context,
	req capacitymanagementsdk.CreateOccCapacityRequestRequest,
) (capacitymanagementsdk.CreateOccCapacityRequestResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return capacitymanagementsdk.CreateOccCapacityRequestResponse{}, nil
}

func (f *fakeOccCapacityRequestOCIClient) GetOccCapacityRequest(
	ctx context.Context,
	req capacitymanagementsdk.GetOccCapacityRequestRequest,
) (capacitymanagementsdk.GetOccCapacityRequestResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return capacitymanagementsdk.GetOccCapacityRequestResponse{}, errortest.NewServiceError(404, "NotFound", "missing OccCapacityRequest")
}

func (f *fakeOccCapacityRequestOCIClient) ListOccCapacityRequests(
	ctx context.Context,
	req capacitymanagementsdk.ListOccCapacityRequestsRequest,
) (capacitymanagementsdk.ListOccCapacityRequestsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return capacitymanagementsdk.ListOccCapacityRequestsResponse{}, nil
}

func (f *fakeOccCapacityRequestOCIClient) UpdateOccCapacityRequest(
	ctx context.Context,
	req capacitymanagementsdk.UpdateOccCapacityRequestRequest,
) (capacitymanagementsdk.UpdateOccCapacityRequestResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return capacitymanagementsdk.UpdateOccCapacityRequestResponse{}, nil
}

func (f *fakeOccCapacityRequestOCIClient) DeleteOccCapacityRequest(
	ctx context.Context,
	req capacitymanagementsdk.DeleteOccCapacityRequestRequest,
) (capacitymanagementsdk.DeleteOccCapacityRequestResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return capacitymanagementsdk.DeleteOccCapacityRequestResponse{}, nil
}

func TestReviewedOccCapacityRequestRuntimeSemanticsEncodesNoAdoptionContract(t *testing.T) {
	t.Parallel()

	got := reviewedOccCapacityRequestRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedOccCapacityRequestRuntimeSemantics() = nil")
	}
	if got.FormalService != "capacitymanagement" {
		t.Fatalf("FormalService = %q, want capacitymanagement", got.FormalService)
	}
	if got.FormalSlug != "occcapacityrequest" {
		t.Fatalf("FormalSlug = %q, want occcapacityrequest", got.FormalSlug)
	}
	if got.List != nil {
		t.Fatalf("List = %#v, want nil because untracked list adoption is disabled", got.List)
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" {
		t.Fatalf("Async = %#v, want lifecycle semantics", got.Async)
	}
	if !slices.Equal(got.Lifecycle.ProvisioningStates, []string{"CREATING"}) {
		t.Fatalf("Lifecycle.ProvisioningStates = %#v, want [CREATING]", got.Lifecycle.ProvisioningStates)
	}
	if !slices.Equal(got.Lifecycle.UpdatingStates, []string{"UPDATING"}) {
		t.Fatalf("Lifecycle.UpdatingStates = %#v, want [UPDATING]", got.Lifecycle.UpdatingStates)
	}
	if !slices.Equal(got.Lifecycle.ActiveStates, []string{"ACTIVE"}) {
		t.Fatalf("Lifecycle.ActiveStates = %#v, want [ACTIVE]", got.Lifecycle.ActiveStates)
	}
	if !slices.Equal(got.Delete.PendingStates, []string{"DELETING"}) {
		t.Fatalf("Delete.PendingStates = %#v, want [DELETING]", got.Delete.PendingStates)
	}
	if !slices.Equal(got.Mutation.Mutable, []string{"definedTags", "displayName", "freeformTags", "requestState"}) {
		t.Fatalf("Mutation.Mutable = %#v, want reviewed mutable fields", got.Mutation.Mutable)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
}

func TestGuardOccCapacityRequestExistingBeforeCreateAlwaysSkips(t *testing.T) {
	t.Parallel()

	decision, err := guardOccCapacityRequestExistingBeforeCreate(context.Background(), newOccCapacityRequestTestResource())
	if err != nil {
		t.Fatalf("guardOccCapacityRequestExistingBeforeCreate() error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardOccCapacityRequestExistingBeforeCreate() = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}
}

func TestNormalizeOccCapacityRequestDesiredStateClearsServerManagedFields(t *testing.T) {
	t.Parallel()

	resource := newOccCapacityRequestTestResource()
	resource.Spec.LifecycleDetails = "user supplied"
	resource.Spec.RequestState = "CANCELLED"

	current := observedOccCapacityRequestFromSpec("ocid1.occcapacityrequest.oc1..existing", resource.Spec, "ACTIVE", "COMPLETED")
	normalizeOccCapacityRequestDesiredState(resource, current)

	if resource.Spec.LifecycleDetails != "" {
		t.Fatalf("Spec.LifecycleDetails = %q, want empty after normalization", resource.Spec.LifecycleDetails)
	}
	if resource.Spec.RequestState != "" {
		t.Fatalf("Spec.RequestState = %q, want empty after request settles beyond mutable states", resource.Spec.RequestState)
	}
}

func TestBuildOccCapacityRequestUpdateBodySupportsTagClearsAndRequestState(t *testing.T) {
	t.Parallel()

	currentResource := newOccCapacityRequestTestResource()
	desired := newOccCapacityRequestTestResource()
	desired.Spec.DisplayName = "capacity-request-updated"
	desired.Spec.RequestState = string(capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateCancelled)
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildOccCapacityRequestUpdateBody(
		desired,
		observedOccCapacityRequestFromSpec(
			"ocid1.occcapacityrequest.oc1..existing",
			currentResource.Spec,
			"ACTIVE",
			string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
		),
	)
	if err != nil {
		t.Fatalf("buildOccCapacityRequestUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildOccCapacityRequestUpdateBody() updateNeeded = false, want true")
	}
	if body.DisplayName == nil || *body.DisplayName != desired.Spec.DisplayName {
		t.Fatalf("UpdateOccCapacityRequestDetails.DisplayName = %#v, want %q", body.DisplayName, desired.Spec.DisplayName)
	}
	if body.RequestState != capacitymanagementsdk.UpdateOccCapacityRequestDetailsRequestStateCancelled {
		t.Fatalf("UpdateOccCapacityRequestDetails.RequestState = %q, want CANCELLED", body.RequestState)
	}
	if len(body.FreeformTags) != 0 {
		t.Fatalf("UpdateOccCapacityRequestDetails.FreeformTags = %#v, want empty map clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("UpdateOccCapacityRequestDetails.DefinedTags = %#v, want empty map clear", body.DefinedTags)
	}
}

func TestOccCapacityRequestCreateOrUpdateCreatesWithoutListReuse(t *testing.T) {
	t.Parallel()

	resource := newOccCapacityRequestTestResource()
	listCalls := 0
	createCalls := 0

	client := newTestOccCapacityRequestClient(&fakeOccCapacityRequestOCIClient{
		listFn: func(_ context.Context, _ capacitymanagementsdk.ListOccCapacityRequestsRequest) (capacitymanagementsdk.ListOccCapacityRequestsResponse, error) {
			listCalls++
			return capacitymanagementsdk.ListOccCapacityRequestsResponse{}, nil
		},
		getFn: func(_ context.Context, req capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error) {
			if req.OccCapacityRequestId == nil || *req.OccCapacityRequestId != "ocid1.occcapacityrequest.oc1..created" {
				t.Fatalf("GetOccCapacityRequestRequest.OccCapacityRequestId = %#v, want created OCID", req.OccCapacityRequestId)
			}
			return capacitymanagementsdk.GetOccCapacityRequestResponse{
				OccCapacityRequest: observedOccCapacityRequestFromSpec(
					"ocid1.occcapacityrequest.oc1..created",
					resource.Spec,
					"ACTIVE",
					string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
				),
			}, nil
		},
		createFn: func(_ context.Context, req capacitymanagementsdk.CreateOccCapacityRequestRequest) (capacitymanagementsdk.CreateOccCapacityRequestResponse, error) {
			createCalls++
			if req.CreateOccCapacityRequestDetails.CompartmentId == nil || *req.CreateOccCapacityRequestDetails.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("CreateOccCapacityRequestDetails.CompartmentId = %#v, want %q", req.CreateOccCapacityRequestDetails.CompartmentId, resource.Spec.CompartmentId)
			}
			return capacitymanagementsdk.CreateOccCapacityRequestResponse{
				OccCapacityRequest: observedOccCapacityRequestFromSpec(
					"ocid1.occcapacityrequest.oc1..created",
					resource.Spec,
					"ACTIVE",
					string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
				),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = true, want false")
	}
	if createCalls != 1 {
		t.Fatalf("CreateOccCapacityRequest() calls = %d, want 1", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListOccCapacityRequests() calls = %d, want 0 because untracked adoption is disabled", listCalls)
	}
}

func TestOccCapacityRequestCreateOrUpdateRecreatesAfterStaleTrackedIDWithoutListAdoption(t *testing.T) {
	t.Parallel()

	resource := newExistingOccCapacityRequestTestResource("ocid1.occcapacityrequest.oc1..stale")
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := newTestOccCapacityRequestClient(&fakeOccCapacityRequestOCIClient{
		listFn: func(_ context.Context, _ capacitymanagementsdk.ListOccCapacityRequestsRequest) (capacitymanagementsdk.ListOccCapacityRequestsResponse, error) {
			listCalls++
			return capacitymanagementsdk.ListOccCapacityRequestsResponse{}, nil
		},
		getFn: func(_ context.Context, req capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error) {
			getCalls++
			if req.OccCapacityRequestId == nil {
				t.Fatal("GetOccCapacityRequestRequest.OccCapacityRequestId = nil")
			}
			switch *req.OccCapacityRequestId {
			case "ocid1.occcapacityrequest.oc1..stale":
				return capacitymanagementsdk.GetOccCapacityRequestResponse{}, errortest.NewServiceError(404, "NotFound", "missing OccCapacityRequest")
			case "ocid1.occcapacityrequest.oc1..created":
				return capacitymanagementsdk.GetOccCapacityRequestResponse{
					OccCapacityRequest: observedOccCapacityRequestFromSpec(
						"ocid1.occcapacityrequest.oc1..created",
						resource.Spec,
						"ACTIVE",
						string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
					),
				}, nil
			default:
				t.Fatalf("unexpected get ID %q", *req.OccCapacityRequestId)
				return capacitymanagementsdk.GetOccCapacityRequestResponse{}, nil
			}
		},
		createFn: func(_ context.Context, _ capacitymanagementsdk.CreateOccCapacityRequestRequest) (capacitymanagementsdk.CreateOccCapacityRequestResponse, error) {
			createCalls++
			return capacitymanagementsdk.CreateOccCapacityRequestResponse{
				OccCapacityRequest: observedOccCapacityRequestFromSpec(
					"ocid1.occcapacityrequest.oc1..created",
					resource.Spec,
					"ACTIVE",
					string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if createCalls != 1 {
		t.Fatalf("CreateOccCapacityRequest() calls = %d, want 1 after stale tracked ID miss", createCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetOccCapacityRequest() calls = %d, want 2 (stale ID miss + created follow-up)", getCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListOccCapacityRequests() calls = %d, want 0 because stale tracked IDs must not adopt another live request", listCalls)
	}
}

func TestOccCapacityRequestCreatePendingUsesRetryAfter(t *testing.T) {
	t.Parallel()

	resource := newOccCapacityRequestTestResource()

	client := newTestOccCapacityRequestClient(&fakeOccCapacityRequestOCIClient{
		getFn: func(_ context.Context, req capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error) {
			if req.OccCapacityRequestId == nil || *req.OccCapacityRequestId != "ocid1.occcapacityrequest.oc1..created" {
				t.Fatalf("GetOccCapacityRequestRequest.OccCapacityRequestId = %#v, want created OCID", req.OccCapacityRequestId)
			}
			return capacitymanagementsdk.GetOccCapacityRequestResponse{
				OccCapacityRequest: observedOccCapacityRequestFromSpec(
					"ocid1.occcapacityrequest.oc1..created",
					resource.Spec,
					"CREATING",
					string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
				),
			}, nil
		},
		createFn: func(_ context.Context, _ capacitymanagementsdk.CreateOccCapacityRequestRequest) (capacitymanagementsdk.CreateOccCapacityRequestResponse, error) {
			return capacitymanagementsdk.CreateOccCapacityRequestResponse{
				OccCapacityRequest: observedOccCapacityRequestFromSpec(
					"ocid1.occcapacityrequest.oc1..created",
					resource.Spec,
					"CREATING",
					string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
				),
				RetryAfter:   common.Int(37),
				OpcRequestId: common.String("opc-create-pending"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if response.RequeueDuration != 37*time.Second {
		t.Fatalf("CreateOrUpdate() RequeueDuration = %v, want 37s", response.RequeueDuration)
	}
}

func TestOccCapacityRequestUpdatePendingUsesRetryAfter(t *testing.T) {
	t.Parallel()

	resource := newExistingOccCapacityRequestTestResource("ocid1.occcapacityrequest.oc1..existing")
	resource.Spec.DisplayName = "capacity-request-updated"
	getCalls := 0

	client := newTestOccCapacityRequestClient(&fakeOccCapacityRequestOCIClient{
		getFn: func(_ context.Context, req capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error) {
			getCalls++
			if req.OccCapacityRequestId == nil {
				t.Fatal("GetOccCapacityRequestRequest.OccCapacityRequestId = nil")
			}
			switch getCalls {
			case 1:
				currentSpec := newOccCapacityRequestTestResource().Spec
				return capacitymanagementsdk.GetOccCapacityRequestResponse{
					OccCapacityRequest: observedOccCapacityRequestFromSpec(
						"ocid1.occcapacityrequest.oc1..existing",
						currentSpec,
						"ACTIVE",
						string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
					),
				}, nil
			case 2:
				return capacitymanagementsdk.GetOccCapacityRequestResponse{
					OccCapacityRequest: observedOccCapacityRequestFromSpec(
						"ocid1.occcapacityrequest.oc1..existing",
						resource.Spec,
						"UPDATING",
						string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
					),
				}, nil
			default:
				t.Fatalf("GetOccCapacityRequest() calls = %d, want 2", getCalls)
				return capacitymanagementsdk.GetOccCapacityRequestResponse{}, nil
			}
		},
		updateFn: func(_ context.Context, req capacitymanagementsdk.UpdateOccCapacityRequestRequest) (capacitymanagementsdk.UpdateOccCapacityRequestResponse, error) {
			if req.UpdateOccCapacityRequestDetails.DisplayName == nil || *req.UpdateOccCapacityRequestDetails.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("UpdateOccCapacityRequestDetails.DisplayName = %#v, want %q", req.UpdateOccCapacityRequestDetails.DisplayName, resource.Spec.DisplayName)
			}
			return capacitymanagementsdk.UpdateOccCapacityRequestResponse{
				OccCapacityRequest: observedOccCapacityRequestFromSpec(
					"ocid1.occcapacityrequest.oc1..existing",
					resource.Spec,
					"UPDATING",
					string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
				),
				RetryAfter:   common.Int(19),
				OpcRequestId: common.String("opc-update-pending"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if response.RequeueDuration != 19*time.Second {
		t.Fatalf("CreateOrUpdate() RequeueDuration = %v, want 19s", response.RequeueDuration)
	}
}

func TestOccCapacityRequestDeleteWithResultUsesRetryAfter(t *testing.T) {
	t.Parallel()

	resource := newExistingOccCapacityRequestTestResource("ocid1.occcapacityrequest.oc1..existing")
	getCalls := 0
	client := newTestOccCapacityRequestClient(&fakeOccCapacityRequestOCIClient{
		deleteFn: func(_ context.Context, req capacitymanagementsdk.DeleteOccCapacityRequestRequest) (capacitymanagementsdk.DeleteOccCapacityRequestResponse, error) {
			if req.OccCapacityRequestId == nil || *req.OccCapacityRequestId != "ocid1.occcapacityrequest.oc1..existing" {
				t.Fatalf("DeleteOccCapacityRequestRequest.OccCapacityRequestId = %#v, want existing OCID", req.OccCapacityRequestId)
			}
			return capacitymanagementsdk.DeleteOccCapacityRequestResponse{
				RetryAfter:   common.Int(23),
				OpcRequestId: common.String("opc-delete-pending"),
			}, nil
		},
		getFn: func(_ context.Context, req capacitymanagementsdk.GetOccCapacityRequestRequest) (capacitymanagementsdk.GetOccCapacityRequestResponse, error) {
			getCalls++
			if req.OccCapacityRequestId == nil || *req.OccCapacityRequestId != "ocid1.occcapacityrequest.oc1..existing" {
				t.Fatalf("GetOccCapacityRequestRequest.OccCapacityRequestId = %#v, want existing OCID", req.OccCapacityRequestId)
			}
			switch getCalls {
			case 1:
				return capacitymanagementsdk.GetOccCapacityRequestResponse{
					OccCapacityRequest: observedOccCapacityRequestFromSpec(
						"ocid1.occcapacityrequest.oc1..existing",
						resource.Spec,
						"ACTIVE",
						string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
					),
				}, nil
			case 2:
				return capacitymanagementsdk.GetOccCapacityRequestResponse{
					OccCapacityRequest: observedOccCapacityRequestFromSpec(
						"ocid1.occcapacityrequest.oc1..existing",
						resource.Spec,
						"DELETING",
						string(capacitymanagementsdk.OccCapacityRequestRequestStateSubmitted),
					),
				}, nil
			default:
				t.Fatalf("GetOccCapacityRequest() calls = %d, want 2", getCalls)
				return capacitymanagementsdk.GetOccCapacityRequestResponse{}, nil
			}
		},
	})

	deleteClient, ok := client.(occCapacityRequestDeleteResultClient)
	if !ok {
		t.Fatalf("client type %T does not implement occCapacityRequestDeleteResultClient", client)
	}

	result, err := deleteClient.DeleteWithResult(context.Background(), resource)
	if err != nil {
		t.Fatalf("DeleteWithResult() error = %v", err)
	}
	if result.Deleted {
		t.Fatal("DeleteWithResult() Deleted = true, want false while OCI is DELETING")
	}
	if result.RequeueDuration != 23*time.Second {
		t.Fatalf("DeleteWithResult() RequeueDuration = %v, want 23s", result.RequeueDuration)
	}
}

func newTestOccCapacityRequestClient(client occCapacityRequestOCIClient) OccCapacityRequestServiceClient {
	return newOccCapacityRequestServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		client,
	)
}

func newOccCapacityRequestTestResource() *capacitymanagementv1beta1.OccCapacityRequest {
	return &capacitymanagementv1beta1.OccCapacityRequest{
		Spec: capacitymanagementv1beta1.OccCapacityRequestSpec{
			CompartmentId:                "ocid1.tenancy.oc1..tenancy",
			Namespace:                    "COMPUTE",
			Region:                       "us-phoenix-1",
			DisplayName:                  "capacity-request",
			DateExpectedCapacityHandover: "2026-06-01T00:00:00Z",
			Details: []capacitymanagementv1beta1.OccCapacityRequestDetail{
				{
					ResourceType:   "SERVER",
					WorkloadType:   "GENERIC",
					ResourceName:   "BM.Standard.E5.192",
					DemandQuantity: 2,
				},
			},
			OccAvailabilityCatalogId: "ocid1.occavailabilitycatalog.oc1..catalog",
			RequestType:              "NEW",
			Description:              "initial request",
			FreeformTags:             map[string]string{"env": "dev"},
			DefinedTags:              map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			AvailabilityDomain:       "Uocm:PHX-AD-1",
			RequestState:             "SUBMITTED",
		},
	}
}

func newExistingOccCapacityRequestTestResource(id string) *capacitymanagementv1beta1.OccCapacityRequest {
	resource := newOccCapacityRequestTestResource()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	return resource
}

func observedOccCapacityRequestFromSpec(
	id string,
	spec capacitymanagementv1beta1.OccCapacityRequestSpec,
	lifecycleState string,
	requestState string,
) capacitymanagementsdk.OccCapacityRequest {
	return capacitymanagementsdk.OccCapacityRequest{
		Id:                           common.String(id),
		CompartmentId:                common.String(spec.CompartmentId),
		OccAvailabilityCatalogId:     stringPtr(spec.OccAvailabilityCatalogId),
		DisplayName:                  common.String(spec.DisplayName),
		Namespace:                    capacitymanagementsdk.NamespaceEnum(spec.Namespace),
		OccCustomerGroupId:           common.String("ocid1.occcustomergroup.oc1..group"),
		Region:                       common.String(spec.Region),
		AvailabilityDomain:           stringPtr(spec.AvailabilityDomain),
		DateExpectedCapacityHandover: sdkTimePtr(spec.DateExpectedCapacityHandover),
		RequestState:                 capacitymanagementsdk.OccCapacityRequestRequestStateEnum(requestState),
		TimeCreated:                  sdkTimePtr("2026-05-01T00:00:00Z"),
		TimeUpdated:                  sdkTimePtr("2026-05-02T00:00:00Z"),
		LifecycleState:               capacitymanagementsdk.OccCapacityRequestLifecycleStateEnum(lifecycleState),
		Details:                      sdkOccCapacityRequestDetails(spec.Details),
		Description:                  stringPtr(spec.Description),
		RequestType:                  capacitymanagementsdk.OccCapacityRequestRequestTypeEnum(spec.RequestType),
		LifecycleDetails:             common.String("current lifecycle details"),
		FreeformTags:                 maps.Clone(spec.FreeformTags),
		DefinedTags:                  occCapacityRequestDefinedTagsFromSpec(spec.DefinedTags),
		SystemTags:                   map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}},
	}
}

func sdkOccCapacityRequestDetails(spec []capacitymanagementv1beta1.OccCapacityRequestDetail) []capacitymanagementsdk.OccCapacityRequestBaseDetails {
	if spec == nil {
		return nil
	}

	out := make([]capacitymanagementsdk.OccCapacityRequestBaseDetails, 0, len(spec))
	for _, detail := range spec {
		out = append(out, capacitymanagementsdk.OccCapacityRequestBaseDetails{
			ResourceType:             common.String(detail.ResourceType),
			WorkloadType:             common.String(detail.WorkloadType),
			ResourceName:             common.String(detail.ResourceName),
			DemandQuantity:           common.Int64(detail.DemandQuantity),
			SourceWorkloadType:       stringPtr(detail.SourceWorkloadType),
			ExpectedHandoverQuantity: common.Int64(detail.ExpectedHandoverQuantity),
			DateExpectedHandover:     sdkTimePtr(detail.DateExpectedHandover),
			ActualHandoverQuantity:   common.Int64(detail.ActualHandoverQuantity),
			DateActualHandover:       sdkTimePtr(detail.DateActualHandover),
			AvailabilityDomain:       stringPtr(detail.AvailabilityDomain),
		})
	}
	return out
}

func sdkTimePtr(value string) *common.SDKTime {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &common.SDKTime{Time: parsed.UTC()}
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}
