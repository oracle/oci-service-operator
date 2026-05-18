package zprpolicy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	zprsdk "github.com/oracle/oci-go-sdk/v65/zpr"
	zprv1beta1 "github.com/oracle/oci-service-operator/api/zpr/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const testZprPolicyID = "ocid1.zprpolicy.oc1..runtime"

type fakeZprPolicyOCIClient struct {
	createFn         func(context.Context, zprsdk.CreateZprPolicyRequest) (zprsdk.CreateZprPolicyResponse, error)
	getFn            func(context.Context, zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error)
	listFn           func(context.Context, zprsdk.ListZprPoliciesRequest) (zprsdk.ListZprPoliciesResponse, error)
	updateFn         func(context.Context, zprsdk.UpdateZprPolicyRequest) (zprsdk.UpdateZprPolicyResponse, error)
	deleteFn         func(context.Context, zprsdk.DeleteZprPolicyRequest) (zprsdk.DeleteZprPolicyResponse, error)
	getWorkRequestFn func(context.Context, zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error)
}

func (f *fakeZprPolicyOCIClient) CreateZprPolicy(ctx context.Context, req zprsdk.CreateZprPolicyRequest) (zprsdk.CreateZprPolicyResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return zprsdk.CreateZprPolicyResponse{}, nil
}

func (f *fakeZprPolicyOCIClient) GetZprPolicy(ctx context.Context, req zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return zprsdk.GetZprPolicyResponse{}, nil
}

func (f *fakeZprPolicyOCIClient) ListZprPolicies(ctx context.Context, req zprsdk.ListZprPoliciesRequest) (zprsdk.ListZprPoliciesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return zprsdk.ListZprPoliciesResponse{}, nil
}

func (f *fakeZprPolicyOCIClient) UpdateZprPolicy(ctx context.Context, req zprsdk.UpdateZprPolicyRequest) (zprsdk.UpdateZprPolicyResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return zprsdk.UpdateZprPolicyResponse{}, nil
}

func (f *fakeZprPolicyOCIClient) DeleteZprPolicy(ctx context.Context, req zprsdk.DeleteZprPolicyRequest) (zprsdk.DeleteZprPolicyResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return zprsdk.DeleteZprPolicyResponse{}, nil
}

func (f *fakeZprPolicyOCIClient) GetZprPolicyWorkRequest(ctx context.Context, req zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return zprsdk.GetZprPolicyWorkRequestResponse{}, nil
}

func newTestZprPolicyRuntimeHooks(client zprPolicyOCIClient) ZprPolicyRuntimeHooks {
	if client == nil {
		client = &fakeZprPolicyOCIClient{}
	}

	hooks := ZprPolicyRuntimeHooks{
		Semantics: newZprPolicyRuntimeSemantics(),
		Create: runtimeOperationHooks[zprsdk.CreateZprPolicyRequest, zprsdk.CreateZprPolicyResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateZprPolicyDetails", RequestName: "CreateZprPolicyDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request zprsdk.CreateZprPolicyRequest) (zprsdk.CreateZprPolicyResponse, error) {
				return client.CreateZprPolicy(ctx, request)
			},
		},
		Get: runtimeOperationHooks[zprsdk.GetZprPolicyRequest, zprsdk.GetZprPolicyResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ZprPolicyId", RequestName: "zprPolicyId", Contribution: "path", PreferResourceID: true},
			},
			Call: func(ctx context.Context, request zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error) {
				return client.GetZprPolicy(ctx, request)
			},
		},
		List: runtimeOperationHooks[zprsdk.ListZprPoliciesRequest, zprsdk.ListZprPoliciesResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
				{FieldName: "Id", RequestName: "id", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
			},
			Call: func(ctx context.Context, request zprsdk.ListZprPoliciesRequest) (zprsdk.ListZprPoliciesResponse, error) {
				return client.ListZprPolicies(ctx, request)
			},
		},
		Update: runtimeOperationHooks[zprsdk.UpdateZprPolicyRequest, zprsdk.UpdateZprPolicyResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ZprPolicyId", RequestName: "zprPolicyId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateZprPolicyDetails", RequestName: "UpdateZprPolicyDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request zprsdk.UpdateZprPolicyRequest) (zprsdk.UpdateZprPolicyResponse, error) {
				return client.UpdateZprPolicy(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[zprsdk.DeleteZprPolicyRequest, zprsdk.DeleteZprPolicyResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ZprPolicyId", RequestName: "zprPolicyId", Contribution: "path", PreferResourceID: true},
			},
			Call: func(ctx context.Context, request zprsdk.DeleteZprPolicyRequest) (zprsdk.DeleteZprPolicyResponse, error) {
				return client.DeleteZprPolicy(ctx, request)
			},
		},
	}

	applyZprPolicyRuntimeHooks(&hooks, client, nil)
	hooks.WrapGeneratedClient = nil
	return hooks
}

func newZprPolicyTestClient(client zprPolicyOCIClient) ZprPolicyServiceClient {
	hooks := newTestZprPolicyRuntimeHooks(client)
	delegate := defaultZprPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*zprv1beta1.ZprPolicy](
			buildZprPolicyGeneratedRuntimeConfig(
				&ZprPolicyServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
				hooks,
			),
		),
	}
	return wrapZprPolicyGeneratedClient(hooks, delegate)
}

func newZprPolicyTestResource() *zprv1beta1.ZprPolicy {
	return &zprv1beta1.ZprPolicy{
		Spec: zprv1beta1.ZprPolicySpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			Name:          "policy-alpha",
			Description:   "policy description",
			Statements: []string{
				"allow any-user to inspect zpr-policies in tenancy",
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func trackZprPolicy(resource *zprv1beta1.ZprPolicy, policyID string) {
	resource.Status.Id = policyID
	resource.Status.OsokStatus.Ocid = shared.OCID(policyID)
}

func makeSDKZprPolicy(id, name, description string, state zprsdk.ZprPolicyLifecycleStateEnum) zprsdk.ZprPolicy {
	return zprsdk.ZprPolicy{
		Id:               common.String(id),
		Name:             common.String(name),
		Description:      common.String(description),
		CompartmentId:    common.String("ocid1.compartment.oc1..example"),
		Statements:       []string{"allow any-user to inspect zpr-policies in tenancy"},
		LifecycleState:   state,
		TimeCreated:      &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:      &common.SDKTime{Time: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		LifecycleDetails: common.String("ready"),
	}
}

func makeZprWorkRequest(
	id string,
	status zprsdk.WorkRequestStatusEnum,
	operation zprsdk.OperationTypeEnum,
	action zprsdk.ActionTypeEnum,
	policyID string,
) zprsdk.WorkRequest {
	return zprsdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operation,
		CompartmentId: common.String("ocid1.compartment.oc1..example"),
		Resources: []zprsdk.WorkRequestResource{
			{
				EntityType: common.String("zprpolicy"),
				ActionType: action,
				Identifier: common.String(policyID),
				EntityUri:  common.String("/zprPolicies/" + policyID),
			},
		},
		PercentComplete: common.Float32(50),
		TimeAccepted:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:     &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 1, 0, 0, time.UTC)},
	}
}

func TestApplyZprPolicyRuntimeHooksConfiguresGeneratedWorkRequestHooks(t *testing.T) {
	t.Parallel()

	hooks := newZprPolicyDefaultRuntimeHooks(zprsdk.ZprClient{})
	applyZprPolicyRuntimeHooks(&hooks, &fakeZprPolicyOCIClient{}, nil)

	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want create body builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want update body builder")
	}
	if hooks.StatusHooks.ProjectStatus == nil {
		t.Fatal("hooks.StatusHooks.ProjectStatus = nil, want status projection")
	}
	if hooks.TrackedRecreate.ClearTrackedIdentity == nil {
		t.Fatal("hooks.TrackedRecreate.ClearTrackedIdentity = nil, want tracked identity clear hook")
	}
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("hooks.Async.GetWorkRequest = nil, want work request getter")
	}
	if hooks.Async.ResolveAction == nil {
		t.Fatal("hooks.Async.ResolveAction = nil, want work request action resolver")
	}
	if hooks.Async.ResolvePhase == nil {
		t.Fatal("hooks.Async.ResolvePhase = nil, want work request phase resolver")
	}
	if hooks.Async.RecoverResourceID == nil {
		t.Fatal("hooks.Async.RecoverResourceID = nil, want work request resource-ID recovery")
	}
}

func TestZprPolicyRuntimeCreateTracksPendingWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newZprPolicyTestResource()
	client := newZprPolicyTestClient(&fakeZprPolicyOCIClient{
		createFn: func(_ context.Context, req zprsdk.CreateZprPolicyRequest) (zprsdk.CreateZprPolicyResponse, error) {
			if got := stringValue(req.CreateZprPolicyDetails.Name); got != resource.Spec.Name {
				t.Fatalf("CreateZprPolicy request name = %q, want %q", got, resource.Spec.Name)
			}
			return zprsdk.CreateZprPolicyResponse{
				ZprPolicy:        makeSDKZprPolicy(testZprPolicyID, resource.Spec.Name, resource.Spec.Description, zprsdk.ZprPolicyLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("req-create-1"),
				ContentLocation:  common.String("/zprPolicies/" + testZprPolicyID),
				Location:         common.String("/zprPolicies/" + testZprPolicyID),
			}, nil
		},
		getFn: func(_ context.Context, _ zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error) {
			t.Fatal("GetZprPolicy should not run while the create work request is pending")
			return zprsdk.GetZprPolicyResponse{}, nil
		},
		getWorkRequestFn: func(_ context.Context, req zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error) {
			if got := stringValue(req.WorkRequestId); got != "wr-create-1" {
				t.Fatalf("GetZprPolicyWorkRequest request ID = %q, want %q", got, "wr-create-1")
			}
			return zprsdk.GetZprPolicyWorkRequestResponse{
				WorkRequest: makeZprWorkRequest(
					"wr-create-1",
					zprsdk.WorkRequestStatusInProgress,
					zprsdk.OperationTypeCreateZprPolicy,
					zprsdk.ActionTypeCreated,
					testZprPolicyID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true while work request is pending")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while work request is pending")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want pending work request tracker")
	}

	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, shared.OSOKAsyncPhaseCreate)
	}
	if current.WorkRequestID != "wr-create-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, "wr-create-1")
	}
	if current.RawStatus != "IN_PROGRESS" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, "IN_PROGRESS")
	}
	if current.RawOperationType != "CREATE_ZPR_POLICY" {
		t.Fatalf("status.async.current.rawOperationType = %q, want %q", current.RawOperationType, "CREATE_ZPR_POLICY")
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func TestZprPolicyRuntimeCreateSucceededWorkRequestReadsPolicy(t *testing.T) {
	t.Parallel()

	resource := newZprPolicyTestResource()
	client := newZprPolicyTestClient(&fakeZprPolicyOCIClient{
		createFn: func(_ context.Context, _ zprsdk.CreateZprPolicyRequest) (zprsdk.CreateZprPolicyResponse, error) {
			return zprsdk.CreateZprPolicyResponse{
				ZprPolicy:        makeSDKZprPolicy(testZprPolicyID, resource.Spec.Name, resource.Spec.Description, zprsdk.ZprPolicyLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-2"),
			}, nil
		},
		getFn: func(_ context.Context, req zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error) {
			if got := stringValue(req.ZprPolicyId); got != testZprPolicyID {
				t.Fatalf("GetZprPolicy request ID = %q, want %q", got, testZprPolicyID)
			}
			return zprsdk.GetZprPolicyResponse{
				ZprPolicy: makeSDKZprPolicy(testZprPolicyID, resource.Spec.Name, resource.Spec.Description, zprsdk.ZprPolicyLifecycleStateActive),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error) {
			if got := stringValue(req.WorkRequestId); got != "wr-create-2" {
				t.Fatalf("GetZprPolicyWorkRequest request ID = %q, want %q", got, "wr-create-2")
			}
			return zprsdk.GetZprPolicyWorkRequestResponse{
				WorkRequest: makeZprWorkRequest(
					"wr-create-2",
					zprsdk.WorkRequestStatusSucceeded,
					zprsdk.OperationTypeCreateZprPolicy,
					zprsdk.ActionTypeCreated,
					testZprPolicyID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true after create converges")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false after ACTIVE readback")
	}
	if resource.Status.Id != testZprPolicyID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testZprPolicyID)
	}
	if resource.Status.Name != resource.Spec.Name {
		t.Fatalf("status.name = %q, want %q", resource.Status.Name, resource.Spec.Name)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after convergence", resource.Status.OsokStatus.Async.Current)
	}
}

func TestZprPolicyRuntimeUpdateNeedsAttentionFails(t *testing.T) {
	t.Parallel()

	resource := newZprPolicyTestResource()
	resource.Spec.Description = "updated description"
	trackZprPolicy(resource, testZprPolicyID)

	client := newZprPolicyTestClient(&fakeZprPolicyOCIClient{
		getFn: func(_ context.Context, req zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error) {
			if got := stringValue(req.ZprPolicyId); got != testZprPolicyID {
				t.Fatalf("GetZprPolicy request ID = %q, want %q", got, testZprPolicyID)
			}
			return zprsdk.GetZprPolicyResponse{
				ZprPolicy: makeSDKZprPolicy(testZprPolicyID, resource.Spec.Name, "old description", zprsdk.ZprPolicyLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req zprsdk.UpdateZprPolicyRequest) (zprsdk.UpdateZprPolicyResponse, error) {
			if got := stringValue(req.ZprPolicyId); got != testZprPolicyID {
				t.Fatalf("UpdateZprPolicy request ID = %q, want %q", got, testZprPolicyID)
			}
			if got := stringValue(req.UpdateZprPolicyDetails.Description); got != resource.Spec.Description {
				t.Fatalf("UpdateZprPolicy description = %q, want %q", got, resource.Spec.Description)
			}
			return zprsdk.UpdateZprPolicyResponse{
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error) {
			if got := stringValue(req.WorkRequestId); got != "wr-update-1" {
				t.Fatalf("GetZprPolicyWorkRequest request ID = %q, want %q", got, "wr-update-1")
			}
			return zprsdk.GetZprPolicyWorkRequestResponse{
				WorkRequest: makeZprWorkRequest(
					"wr-update-1",
					zprsdk.WorkRequestStatusNeedsAttention,
					zprsdk.OperationTypeUpdateZprPolicy,
					zprsdk.ActionTypeUpdated,
					testZprPolicyID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want needs-attention work request failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want failure for needs-attention work request")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want terminal work request tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.NormalizedClass; got != shared.OSOKAsyncClassAttention {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", got, shared.OSOKAsyncClassAttention)
	}
	if !strings.Contains(err.Error(), "finished with status NEEDS_ATTENTION") {
		t.Fatalf("CreateOrUpdate() error = %q, want NEEDS_ATTENTION detail", err.Error())
	}
}

func TestZprPolicyRuntimeDeleteSucceededWorkRequestConfirmsDisappearance(t *testing.T) {
	t.Parallel()

	resource := newZprPolicyTestResource()
	trackZprPolicy(resource, testZprPolicyID)

	client := newZprPolicyTestClient(&fakeZprPolicyOCIClient{
		deleteFn: func(_ context.Context, req zprsdk.DeleteZprPolicyRequest) (zprsdk.DeleteZprPolicyResponse, error) {
			if got := stringValue(req.ZprPolicyId); got != testZprPolicyID {
				t.Fatalf("DeleteZprPolicy request ID = %q, want %q", got, testZprPolicyID)
			}
			return zprsdk.DeleteZprPolicyResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
		getFn: func(_ context.Context, req zprsdk.GetZprPolicyRequest) (zprsdk.GetZprPolicyResponse, error) {
			if got := stringValue(req.ZprPolicyId); got != testZprPolicyID {
				t.Fatalf("GetZprPolicy request ID = %q, want %q", got, testZprPolicyID)
			}
			return zprsdk.GetZprPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "policy deleted")
		},
		getWorkRequestFn: func(_ context.Context, req zprsdk.GetZprPolicyWorkRequestRequest) (zprsdk.GetZprPolicyWorkRequestResponse, error) {
			if got := stringValue(req.WorkRequestId); got != "wr-delete-1" {
				t.Fatalf("GetZprPolicyWorkRequest request ID = %q, want %q", got, "wr-delete-1")
			}
			return zprsdk.GetZprPolicyWorkRequestResponse{
				WorkRequest: makeZprWorkRequest(
					"wr-delete-1",
					zprsdk.WorkRequestStatusSucceeded,
					zprsdk.OperationTypeDeleteZprPolicy,
					zprsdk.ActionTypeDeleted,
					testZprPolicyID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after successful delete confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after delete", resource.Status.OsokStatus.Async.Current)
	}
}
