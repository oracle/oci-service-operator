package quota

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	limitssdk "github.com/oracle/oci-go-sdk/v65/limits"
	limitsv1beta1 "github.com/oracle/oci-service-operator/api/limits/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testQuotaID          = "ocid1.quota.oc1..runtime"
	testQuotaCompartment = "ocid1.compartment.oc1..runtime"
	testQuotaName        = "quota-runtime"
)

func TestApplyQuotaRuntimeHooksOverridesGeneratedDefaults(t *testing.T) {
	t.Parallel()

	hooks := newQuotaDefaultRuntimeHooks(limitssdk.QuotasClient{})
	applyQuotaRuntimeHooks(nil, &hooks)

	assertQuotaRuntimeHooksConfigured(t, hooks)
	assertQuotaUpdateBodyNoopsOnMatchingState(t, hooks)
}

func TestQuotaCreateRequestProjectsBodyAndRequestID(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	resource.Spec.Locks = []limitsv1beta1.QuotaLock{{
		Type:              "DELETE",
		RelatedResourceId: "ocid1.locksource.oc1..runtime",
		Message:           "managed by quota test",
	}}

	fake := &fakeQuotaOCIClient{}
	fake.createFunc = func(_ context.Context, request limitssdk.CreateQuotaRequest) (limitssdk.CreateQuotaResponse, error) {
		if request.OpcRetryToken == nil || *request.OpcRetryToken == "" {
			t.Fatal("create opc retry token is empty, want generated deterministic retry token")
		}
		return limitssdk.CreateQuotaResponse{
			Quota:        quotaFromSpec(testQuotaID, resource.Spec),
			OpcRequestId: common.String("opc-create-1"),
		}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	assertQuotaCreateRequest(t, fake.createRequests[0])
	assertQuotaStatusIDAndRequestID(t, resource, testQuotaID, "opc-create-1")
}

func TestQuotaBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	fake := &fakeQuotaOCIClient{}
	fake.listFunc = paginatedQuotaListFunc(t, fake, resource)
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		return limitssdk.GetQuotaResponse{Quota: quotaFromSpec(testQuotaID, resource.Spec)}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	assertQuotaStatusOCID(t, resource, testQuotaID)
}

func TestQuotaRejectsDuplicateMatchesAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	fake := &fakeQuotaOCIClient{}
	fake.listFunc = func(_ context.Context, request limitssdk.ListQuotasRequest) (limitssdk.ListQuotasResponse, error) {
		switch len(fake.listRequests) {
		case 1:
			return limitssdk.ListQuotasResponse{
				Items:       []limitssdk.QuotaSummary{quotaSummaryFromSpec(testQuotaID, resource.Spec)},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			return limitssdk.ListQuotasResponse{
				Items: []limitssdk.QuotaSummary{quotaSummaryFromSpec("ocid1.quota.oc1..duplicate", resource.Spec)},
			}, nil
		default:
			t.Fatalf("unexpected list request count %d with page %q", len(fake.listRequests), stringValue(request.Page))
			return limitssdk.ListQuotasResponse{}, nil
		}
	}
	client := newQuotaRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match error", err)
	}
	if response.IsSuccessful {
		t.Fatal("response.IsSuccessful = true, want false")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
}

func TestQuotaNoOpReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)

	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		return limitssdk.GetQuotaResponse{Quota: quotaFromSpec(testQuotaID, resource.Spec)}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestQuotaUpdatesSupportedMutableDrift(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)
	resource.Spec.Description = "updated description"
	resource.Spec.Statements = []string{"zero database quotas in compartment runtime"}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	oldSpec := newQuotaRuntimeTestResource().Spec
	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		return limitssdk.GetQuotaResponse{Quota: quotaFromSpec(testQuotaID, oldSpec)}, nil
	}
	fake.updateFunc = func(_ context.Context, request limitssdk.UpdateQuotaRequest) (limitssdk.UpdateQuotaResponse, error) {
		return limitssdk.UpdateQuotaResponse{
			Quota:        quotaFromSpec(testQuotaID, resource.Spec),
			OpcRequestId: common.String("opc-update-1"),
		}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	assertQuotaUpdateRequest(t, fake.updateRequests[0], resource.Spec)
	assertQuotaStatusIDAndRequestID(t, resource, testQuotaID, "opc-update-1")
}

func TestQuotaRejectsCreateOnlyLocksDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)
	resource.Spec.Locks = []limitsv1beta1.QuotaLock{{Type: "FULL", Message: "desired"}}

	current := quotaFromSpec(testQuotaID, resource.Spec)
	current.Locks = []limitssdk.ResourceLock{{Type: limitssdk.ResourceLockTypeDelete, Message: common.String("current")}}

	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		return limitssdk.GetQuotaResponse{Quota: current}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "locks") {
		t.Fatalf("CreateOrUpdate() error = %v, want locks create-only drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("response.IsSuccessful = true, want false")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestQuotaDeleteRetainsFinalizerUntilReadbackConfirmsDeletion(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)

	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		switch len(fake.getRequests) {
		case 1:
			return limitssdk.GetQuotaResponse{Quota: quotaFromSpec(testQuotaID, resource.Spec)}, nil
		case 2:
			return limitssdk.GetQuotaResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "quota deleted")
		default:
			t.Fatalf("unexpected get request count %d", len(fake.getRequests))
			return limitssdk.GetQuotaResponse{}, nil
		}
	}
	fake.deleteFunc = func(context.Context, limitssdk.DeleteQuotaRequest) (limitssdk.DeleteQuotaResponse, error) {
		return limitssdk.DeleteQuotaResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after post-delete not-found confirmation")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestQuotaDeleteWaitsWhenReadbackStillFindsResource(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)

	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		return limitssdk.GetQuotaResponse{Quota: quotaFromSpec(testQuotaID, resource.Spec)}, nil
	}
	fake.deleteFunc = func(context.Context, limitssdk.DeleteQuotaRequest) (limitssdk.DeleteQuotaResponse, error) {
		return limitssdk.DeleteQuotaResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback still returns quota")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil before delete confirmation", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
}

func TestQuotaDeleteRecordsPreDeleteGetRequestIDOnSurfacedError(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)

	getErr := errortest.NewServiceError(500, "InternalError", "get quota failed")
	getErr.OpcRequestID = "opc-get-predelete-1"
	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		return limitssdk.GetQuotaResponse{}, getErr
	}
	client := newQuotaRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want surfaced pre-delete GetQuota error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for surfaced pre-delete GetQuota error")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after pre-delete GetQuota error", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-get-predelete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-get-predelete-1", got)
	}
}

func TestQuotaDeleteRecordsPostDeleteGetRequestIDOnSurfacedError(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)

	getErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	getErr.OpcRequestID = "opc-get-postdelete-1"
	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		switch len(fake.getRequests) {
		case 1:
			return limitssdk.GetQuotaResponse{Quota: quotaFromSpec(testQuotaID, resource.Spec)}, nil
		case 2:
			return limitssdk.GetQuotaResponse{}, getErr
		default:
			t.Fatalf("unexpected get request count %d", len(fake.getRequests))
			return limitssdk.GetQuotaResponse{}, nil
		}
	}
	fake.deleteFunc = func(context.Context, limitssdk.DeleteQuotaRequest) (limitssdk.DeleteQuotaResponse, error) {
		return limitssdk.DeleteQuotaResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newQuotaRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative post-delete auth-shaped 404 error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for surfaced post-delete GetQuota error")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1 before post-delete confirmation error", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-get-postdelete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-get-postdelete-1", got)
	}
}

func TestQuotaDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	trackQuotaID(resource, testQuotaID)

	fake := &fakeQuotaOCIClient{}
	fake.getFunc = func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
		return limitssdk.GetQuotaResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
	client := newQuotaRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped 404 error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after ambiguous pre-delete read", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
}

func assertQuotaRuntimeHooksConfigured(t *testing.T, hooks QuotaRuntimeHooks) {
	t.Helper()

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.Semantics.List == nil || len(hooks.Semantics.List.MatchFields) != 2 {
		t.Fatalf("hooks.Semantics.List = %#v, want compartment/name matching", hooks.Semantics.List)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("hooks.ParityHooks.ValidateCreateOnlyDrift = nil, want create-only drift guard")
	}
	if hooks.Read.List == nil {
		t.Fatal("hooks.Read.List = nil, want paginated list read operation")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
}

func assertQuotaUpdateBodyNoopsOnMatchingState(t *testing.T, hooks QuotaRuntimeHooks) {
	t.Helper()

	resource := newQuotaRuntimeTestResource()
	body, updateNeeded, err := hooks.BuildUpdateBody(
		context.Background(),
		resource,
		resource.Namespace,
		quotaFromSpec(testQuotaID, newQuotaRuntimeTestResource().Spec),
	)
	if err != nil {
		t.Fatalf("BuildUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("BuildUpdateBody() updateNeeded = true with matching current state; body = %#v", body)
	}
}

func assertQuotaCreateRequest(t *testing.T, request limitssdk.CreateQuotaRequest) {
	t.Helper()

	if got := stringValue(request.CompartmentId); got != testQuotaCompartment {
		t.Fatalf("create compartmentId = %q, want %q", got, testQuotaCompartment)
	}
	if got := stringValue(request.Name); got != testQuotaName {
		t.Fatalf("create name = %q, want %q", got, testQuotaName)
	}
	if len(request.Locks) != 1 || request.Locks[0].Type != limitssdk.AddLockDetailsTypeDelete {
		t.Fatalf("create locks = %#v, want DELETE lock", request.Locks)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
}

func assertQuotaStatusIDAndRequestID(t *testing.T, resource *limitsv1beta1.Quota, id, requestID string) {
	t.Helper()

	assertQuotaStatusOCID(t, resource, id)
	if got := resource.Status.OsokStatus.OpcRequestID; got != requestID {
		t.Fatalf("status.opcRequestId = %q, want %q", got, requestID)
	}
}

func assertQuotaStatusOCID(t *testing.T, resource *limitsv1beta1.Quota, id string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.ocid = %q, want %q", got, id)
	}
}

func paginatedQuotaListFunc(
	t *testing.T,
	fake *fakeQuotaOCIClient,
	resource *limitsv1beta1.Quota,
) func(context.Context, limitssdk.ListQuotasRequest) (limitssdk.ListQuotasResponse, error) {
	t.Helper()

	return func(_ context.Context, request limitssdk.ListQuotasRequest) (limitssdk.ListQuotasResponse, error) {
		assertQuotaListRequest(t, request)
		switch len(fake.listRequests) {
		case 1:
			assertQuotaListPage(t, request, "")
			return limitssdk.ListQuotasResponse{OpcNextPage: common.String("page-2")}, nil
		case 2:
			assertQuotaListPage(t, request, "page-2")
			return limitssdk.ListQuotasResponse{
				Items: []limitssdk.QuotaSummary{quotaSummaryFromSpec(testQuotaID, resource.Spec)},
			}, nil
		default:
			t.Fatalf("unexpected list request count %d", len(fake.listRequests))
			return limitssdk.ListQuotasResponse{}, nil
		}
	}
}

func assertQuotaListRequest(t *testing.T, request limitssdk.ListQuotasRequest) {
	t.Helper()

	if got := stringValue(request.CompartmentId); got != testQuotaCompartment {
		t.Fatalf("list compartmentId = %q, want %q", got, testQuotaCompartment)
	}
	if got := stringValue(request.Name); got != testQuotaName {
		t.Fatalf("list name = %q, want %q", got, testQuotaName)
	}
}

func assertQuotaListPage(t *testing.T, request limitssdk.ListQuotasRequest, page string) {
	t.Helper()

	if page == "" {
		if request.Page != nil {
			t.Fatalf("list page = %q, want nil", stringValue(request.Page))
		}
		return
	}
	if got := stringValue(request.Page); got != page {
		t.Fatalf("list page = %q, want %q", got, page)
	}
}

func assertQuotaUpdateRequest(t *testing.T, request limitssdk.UpdateQuotaRequest, spec limitsv1beta1.QuotaSpec) {
	t.Helper()

	if got := stringValue(request.QuotaId); got != testQuotaID {
		t.Fatalf("update quotaId = %q, want %q", got, testQuotaID)
	}
	if request.IsLockOverride != nil {
		t.Fatalf("update isLockOverride = %#v, want omitted", request.IsLockOverride)
	}
	if got := stringValue(request.Description); got != spec.Description {
		t.Fatalf("update description = %q, want %q", got, spec.Description)
	}
	if got := request.Statements; len(got) != 1 || got[0] != spec.Statements[0] {
		t.Fatalf("update statements = %#v, want %#v", got, spec.Statements)
	}
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("update freeformTags.env = %q, want prod", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 84", got)
	}
}

func newQuotaRuntimeTestClient(fake *fakeQuotaOCIClient) QuotaServiceClient {
	hooks := newQuotaDefaultRuntimeHooks(limitssdk.QuotasClient{})
	hooks.Create.Call = fake.CreateQuota
	hooks.Get.Call = fake.GetQuota
	hooks.List.Call = fake.ListQuotas
	hooks.Update.Call = fake.UpdateQuota
	hooks.Delete.Call = fake.DeleteQuota
	applyQuotaRuntimeHooks(nil, &hooks)

	manager := &QuotaServiceManager{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}
	delegate := defaultQuotaServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*limitsv1beta1.Quota](
			buildQuotaGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapQuotaGeneratedClient(hooks, delegate)
}

func newQuotaRuntimeTestResource() *limitsv1beta1.Quota {
	return &limitsv1beta1.Quota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "quota-sample",
			Namespace: "default",
		},
		Spec: limitsv1beta1.QuotaSpec{
			CompartmentId: testQuotaCompartment,
			Name:          testQuotaName,
			Description:   "runtime quota",
			Statements:    []string{"zero compute-core quotas in compartment runtime"},
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func trackQuotaID(resource *limitsv1beta1.Quota, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func quotaFromSpec(id string, spec limitsv1beta1.QuotaSpec) limitssdk.Quota {
	return limitssdk.Quota{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Name:           common.String(spec.Name),
		Statements:     append([]string(nil), spec.Statements...),
		Description:    common.String(spec.Description),
		Locks:          quotaResourceLocksFromSpec(spec.Locks),
		LifecycleState: limitssdk.QuotaLifecycleStateActive,
		FreeformTags:   mapsClone(spec.FreeformTags),
		DefinedTags:    quotaDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func quotaSummaryFromSpec(id string, spec limitsv1beta1.QuotaSpec) limitssdk.QuotaSummary {
	return limitssdk.QuotaSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Name:           common.String(spec.Name),
		Description:    common.String(spec.Description),
		Locks:          quotaResourceLocksFromSpec(spec.Locks),
		LifecycleState: limitssdk.QuotaSummaryLifecycleStateActive,
		FreeformTags:   mapsClone(spec.FreeformTags),
		DefinedTags:    quotaDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func quotaResourceLocksFromSpec(locks []limitsv1beta1.QuotaLock) []limitssdk.ResourceLock {
	converted := make([]limitssdk.ResourceLock, 0, len(locks))
	for _, lock := range locks {
		converted = append(converted, limitssdk.ResourceLock{
			Type:              limitssdk.ResourceLockTypeEnum(lock.Type),
			RelatedResourceId: optionalString(lock.RelatedResourceId),
			Message:           optionalString(lock.Message),
		})
	}
	return converted
}

func mapsClone(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	cloned := make(map[string]string, len(value))
	for key, child := range value {
		cloned[key] = child
	}
	return cloned
}

type fakeQuotaOCIClient struct {
	createFunc func(context.Context, limitssdk.CreateQuotaRequest) (limitssdk.CreateQuotaResponse, error)
	getFunc    func(context.Context, limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error)
	listFunc   func(context.Context, limitssdk.ListQuotasRequest) (limitssdk.ListQuotasResponse, error)
	updateFunc func(context.Context, limitssdk.UpdateQuotaRequest) (limitssdk.UpdateQuotaResponse, error)
	deleteFunc func(context.Context, limitssdk.DeleteQuotaRequest) (limitssdk.DeleteQuotaResponse, error)

	createRequests []limitssdk.CreateQuotaRequest
	getRequests    []limitssdk.GetQuotaRequest
	listRequests   []limitssdk.ListQuotasRequest
	updateRequests []limitssdk.UpdateQuotaRequest
	deleteRequests []limitssdk.DeleteQuotaRequest
}

func (f *fakeQuotaOCIClient) CreateQuota(ctx context.Context, request limitssdk.CreateQuotaRequest) (limitssdk.CreateQuotaResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return limitssdk.CreateQuotaResponse{}, nil
}

func (f *fakeQuotaOCIClient) GetQuota(ctx context.Context, request limitssdk.GetQuotaRequest) (limitssdk.GetQuotaResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return limitssdk.GetQuotaResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "quota not found")
}

func (f *fakeQuotaOCIClient) ListQuotas(ctx context.Context, request limitssdk.ListQuotasRequest) (limitssdk.ListQuotasResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return limitssdk.ListQuotasResponse{}, nil
}

func (f *fakeQuotaOCIClient) UpdateQuota(ctx context.Context, request limitssdk.UpdateQuotaRequest) (limitssdk.UpdateQuotaResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return limitssdk.UpdateQuotaResponse{}, nil
}

func (f *fakeQuotaOCIClient) DeleteQuota(ctx context.Context, request limitssdk.DeleteQuotaRequest) (limitssdk.DeleteQuotaResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return limitssdk.DeleteQuotaResponse{}, nil
}

var _ servicemanager.OSOKServiceManager = (*QuotaServiceManager)(nil)
