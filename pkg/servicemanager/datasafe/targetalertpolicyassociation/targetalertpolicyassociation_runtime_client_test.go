/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package targetalertpolicyassociation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testTargetAlertPolicyAssociationID            = "ocid1.datasafetargetalertpolicyassociation.oc1..assoc"
	testTargetAlertPolicyAssociationCompartmentID = "ocid1.compartment.oc1..datasafe"
	testTargetAlertPolicyAssociationPolicyID      = "ocid1.datasafealertpolicy.oc1..policy"
	testTargetAlertPolicyAssociationTargetID      = "ocid1.datasafetargetdatabase.oc1..target"
	testTargetAlertPolicyAssociationDisplayName   = "customer-target-alert-policy"
	testTargetAlertPolicyAssociationDescription   = "customer target alert policy association"
)

type fakeTargetAlertPolicyAssociationOCI struct {
	createRequests      []datasafesdk.CreateTargetAlertPolicyAssociationRequest
	getRequests         []datasafesdk.GetTargetAlertPolicyAssociationRequest
	listRequests        []datasafesdk.ListTargetAlertPolicyAssociationsRequest
	updateRequests      []datasafesdk.UpdateTargetAlertPolicyAssociationRequest
	deleteRequests      []datasafesdk.DeleteTargetAlertPolicyAssociationRequest
	workRequestRequests []datasafesdk.GetWorkRequestRequest

	create         func(context.Context, datasafesdk.CreateTargetAlertPolicyAssociationRequest) (datasafesdk.CreateTargetAlertPolicyAssociationResponse, error)
	get            func(context.Context, datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error)
	list           func(context.Context, datasafesdk.ListTargetAlertPolicyAssociationsRequest) (datasafesdk.ListTargetAlertPolicyAssociationsResponse, error)
	update         func(context.Context, datasafesdk.UpdateTargetAlertPolicyAssociationRequest) (datasafesdk.UpdateTargetAlertPolicyAssociationResponse, error)
	delete         func(context.Context, datasafesdk.DeleteTargetAlertPolicyAssociationRequest) (datasafesdk.DeleteTargetAlertPolicyAssociationResponse, error)
	getWorkRequest func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

func (f *fakeTargetAlertPolicyAssociationOCI) CreateTargetAlertPolicyAssociation(
	ctx context.Context,
	request datasafesdk.CreateTargetAlertPolicyAssociationRequest,
) (datasafesdk.CreateTargetAlertPolicyAssociationResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create != nil {
		return f.create(ctx, request)
	}
	return datasafesdk.CreateTargetAlertPolicyAssociationResponse{}, nil
}

func (f *fakeTargetAlertPolicyAssociationOCI) GetTargetAlertPolicyAssociation(
	ctx context.Context,
	request datasafesdk.GetTargetAlertPolicyAssociationRequest,
) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get != nil {
		return f.get(ctx, request)
	}
	return datasafesdk.GetTargetAlertPolicyAssociationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
}

func (f *fakeTargetAlertPolicyAssociationOCI) ListTargetAlertPolicyAssociations(
	ctx context.Context,
	request datasafesdk.ListTargetAlertPolicyAssociationsRequest,
) (datasafesdk.ListTargetAlertPolicyAssociationsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list != nil {
		return f.list(ctx, request)
	}
	return datasafesdk.ListTargetAlertPolicyAssociationsResponse{}, nil
}

func (f *fakeTargetAlertPolicyAssociationOCI) UpdateTargetAlertPolicyAssociation(
	ctx context.Context,
	request datasafesdk.UpdateTargetAlertPolicyAssociationRequest,
) (datasafesdk.UpdateTargetAlertPolicyAssociationResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update != nil {
		return f.update(ctx, request)
	}
	return datasafesdk.UpdateTargetAlertPolicyAssociationResponse{}, nil
}

func (f *fakeTargetAlertPolicyAssociationOCI) DeleteTargetAlertPolicyAssociation(
	ctx context.Context,
	request datasafesdk.DeleteTargetAlertPolicyAssociationRequest,
) (datasafesdk.DeleteTargetAlertPolicyAssociationResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete != nil {
		return f.delete(ctx, request)
	}
	return datasafesdk.DeleteTargetAlertPolicyAssociationResponse{}, nil
}

func (f *fakeTargetAlertPolicyAssociationOCI) GetWorkRequest(
	ctx context.Context,
	request datasafesdk.GetWorkRequestRequest,
) (datasafesdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.getWorkRequest != nil {
		return f.getWorkRequest(ctx, request)
	}
	workRequestID := targetAlertPolicyAssociationStringValue(request.WorkRequestId)
	phase := shared.OSOKAsyncPhaseCreate
	switch {
	case strings.Contains(workRequestID, "update"):
		phase = shared.OSOKAsyncPhaseUpdate
	case strings.Contains(workRequestID, "delete"):
		phase = shared.OSOKAsyncPhaseDelete
	}
	return datasafesdk.GetWorkRequestResponse{
		WorkRequest: targetAlertPolicyAssociationWorkRequest(workRequestID, phase, datasafesdk.WorkRequestStatusInProgress),
	}, nil
}

func newTestTargetAlertPolicyAssociationClient(fake *fakeTargetAlertPolicyAssociationOCI) TargetAlertPolicyAssociationServiceClient {
	hooks := newTargetAlertPolicyAssociationDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	hooks.Create.Call = fake.CreateTargetAlertPolicyAssociation
	hooks.Get.Call = fake.GetTargetAlertPolicyAssociation
	hooks.List.Call = fake.ListTargetAlertPolicyAssociations
	hooks.Update.Call = fake.UpdateTargetAlertPolicyAssociation
	hooks.Delete.Call = fake.DeleteTargetAlertPolicyAssociation
	applyTargetAlertPolicyAssociationRuntimeHooks(&hooks, fake, nil)

	manager := &TargetAlertPolicyAssociationServiceManager{}
	delegate := defaultTargetAlertPolicyAssociationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.TargetAlertPolicyAssociation](
			buildTargetAlertPolicyAssociationGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapTargetAlertPolicyAssociationGeneratedClient(hooks, delegate)
}

func TestTargetAlertPolicyAssociationRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newTargetAlertPolicyAssociationDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyTargetAlertPolicyAssociationRuntimeHooks(&hooks, &fakeTargetAlertPolicyAssociationOCI{}, nil)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.RecordPath", ok: hooks.Identity.RecordPath != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "StatusHooks.ClearProjectedStatus", ok: hooks.StatusHooks.ClearProjectedStatus != nil},
		{name: "StatusHooks.RestoreStatus", ok: hooks.StatusHooks.RestoreStatus != nil},
		{name: "StatusHooks.ProjectStatus", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "Async.GetWorkRequest", ok: hooks.Async.GetWorkRequest != nil},
		{name: "Async.RecoverResourceID", ok: hooks.Async.RecoverResourceID != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}

	resource := newTargetAlertPolicyAssociationResource()
	resource.Spec.IsEnabled = false
	body, err := hooks.BuildCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateTargetAlertPolicyAssociationDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateTargetAlertPolicyAssociationDetails", body)
	}
	requireTargetAlertPolicyAssociationStringPtr(t, "create policyId", details.PolicyId, testTargetAlertPolicyAssociationPolicyID)
	requireTargetAlertPolicyAssociationStringPtr(t, "create targetId", details.TargetId, testTargetAlertPolicyAssociationTargetID)
	requireTargetAlertPolicyAssociationStringPtr(t, "create compartmentId", details.CompartmentId, testTargetAlertPolicyAssociationCompartmentID)
	requireTargetAlertPolicyAssociationBoolPtr(t, "create isEnabled", details.IsEnabled, false)
}

func TestTargetAlertPolicyAssociationCreateStartsWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.create = func(_ context.Context, request datasafesdk.CreateTargetAlertPolicyAssociationRequest) (datasafesdk.CreateTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationCreateRequest(t, request, resource)
		return datasafesdk.CreateTargetAlertPolicyAssociationResponse{
			TargetAlertPolicyAssociation: targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateCreating),
			OpcRequestId:                 common.String("opc-create"),
			OpcWorkRequestId:             common.String("wr-create"),
		}, nil
	}

	response, err := newTestTargetAlertPolicyAssociationClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want work request requeue")
	}
	requireTargetAlertPolicyAssociationCallCount(t, "ListTargetAlertPolicyAssociations", len(fake.listRequests), 1)
	requireTargetAlertPolicyAssociationCallCount(t, "CreateTargetAlertPolicyAssociation", len(fake.createRequests), 1)
	requireTargetAlertPolicyAssociationCallCount(t, "GetWorkRequest", len(fake.workRequestRequests), 1)
	requireTargetAlertPolicyAssociationRecordedID(t, resource, testTargetAlertPolicyAssociationID)
	requireTargetAlertPolicyAssociationString(t, "status.lifecycleState", resource.Status.LifecycleState, "CREATING")
	requireTargetAlertPolicyAssociationString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-create")
	requireTargetAlertPolicyAssociationWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", "IN_PROGRESS")
}

func TestTargetAlertPolicyAssociationBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	existing := targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive)
	var pages []string
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.list = func(_ context.Context, request datasafesdk.ListTargetAlertPolicyAssociationsRequest) (datasafesdk.ListTargetAlertPolicyAssociationsResponse, error) {
		pages = append(pages, targetAlertPolicyAssociationStringValue(request.Page))
		requireTargetAlertPolicyAssociationStringPtr(t, "list compartmentId", request.CompartmentId, testTargetAlertPolicyAssociationCompartmentID)
		requireTargetAlertPolicyAssociationStringPtr(t, "list alertPolicyId", request.AlertPolicyId, testTargetAlertPolicyAssociationPolicyID)
		requireTargetAlertPolicyAssociationStringPtr(t, "list targetId", request.TargetId, testTargetAlertPolicyAssociationTargetID)
		if request.Page == nil {
			return datasafesdk.ListTargetAlertPolicyAssociationsResponse{
				TargetAlertPolicyAssociationCollection: datasafesdk.TargetAlertPolicyAssociationCollection{
					Items: []datasafesdk.TargetAlertPolicyAssociationSummary{
						targetAlertPolicyAssociationSummaryWithTarget(resource, "ocid1.datasafetargetalertpolicyassociation.oc1..other", "ocid1.datasafetargetdatabase.oc1..other", datasafesdk.AlertPolicyLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		return datasafesdk.ListTargetAlertPolicyAssociationsResponse{
			TargetAlertPolicyAssociationCollection: datasafesdk.TargetAlertPolicyAssociationCollection{
				Items: []datasafesdk.TargetAlertPolicyAssociationSummary{
					targetAlertPolicyAssociationSummary(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive),
				},
			},
		}, nil
	}
	fake.get = func(_ context.Context, request datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "get id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{TargetAlertPolicyAssociation: existing}, nil
	}
	fake.create = func(context.Context, datasafesdk.CreateTargetAlertPolicyAssociationRequest) (datasafesdk.CreateTargetAlertPolicyAssociationResponse, error) {
		t.Fatal("CreateTargetAlertPolicyAssociation() called despite existing list match")
		return datasafesdk.CreateTargetAlertPolicyAssociationResponse{}, nil
	}

	response, err := newTestTargetAlertPolicyAssociationClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListTargetAlertPolicyAssociations() pages = %q, want \",page-2\"", got)
	}
	requireTargetAlertPolicyAssociationCallCount(t, "CreateTargetAlertPolicyAssociation", len(fake.createRequests), 0)
	requireTargetAlertPolicyAssociationRecordedID(t, resource, testTargetAlertPolicyAssociationID)
}

func TestTargetAlertPolicyAssociationNoopWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTargetAlertPolicyAssociationID)
	current := targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive)
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "get id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{TargetAlertPolicyAssociation: current}, nil
	}
	fake.update = func(context.Context, datasafesdk.UpdateTargetAlertPolicyAssociationRequest) (datasafesdk.UpdateTargetAlertPolicyAssociationResponse, error) {
		t.Fatal("UpdateTargetAlertPolicyAssociation() called during no-op reconcile")
		return datasafesdk.UpdateTargetAlertPolicyAssociationResponse{}, nil
	}

	response, err := newTestTargetAlertPolicyAssociationClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	requireTargetAlertPolicyAssociationCallCount(t, "GetTargetAlertPolicyAssociation", len(fake.getRequests), 1)
	requireTargetAlertPolicyAssociationCallCount(t, "UpdateTargetAlertPolicyAssociation", len(fake.updateRequests), 0)
	requireTargetAlertPolicyAssociationLastCondition(t, resource, shared.Active)
}

func TestTargetAlertPolicyAssociationMutableUpdateStartsWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTargetAlertPolicyAssociationID)
	resource.Spec.IsEnabled = false
	current := targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive)
	current.IsEnabled = common.Bool(true)
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "get id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{TargetAlertPolicyAssociation: current}, nil
	}
	fake.update = func(_ context.Context, request datasafesdk.UpdateTargetAlertPolicyAssociationRequest) (datasafesdk.UpdateTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "update id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		requireTargetAlertPolicyAssociationBoolPtr(t, "update isEnabled", request.IsEnabled, false)
		return datasafesdk.UpdateTargetAlertPolicyAssociationResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}

	response, err := newTestTargetAlertPolicyAssociationClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want update work request requeue")
	}
	requireTargetAlertPolicyAssociationCallCount(t, "UpdateTargetAlertPolicyAssociation", len(fake.updateRequests), 1)
	requireTargetAlertPolicyAssociationString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-update")
	requireTargetAlertPolicyAssociationWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update", "IN_PROGRESS")
}

func TestTargetAlertPolicyAssociationMutableUpdateClearsStrings(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	resource.Status.Id = testTargetAlertPolicyAssociationID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTargetAlertPolicyAssociationID)
	resource.Status.DisplayName = "stale display name"
	resource.Status.Description = "stale description"
	resource.Spec.DisplayName = ""
	resource.Spec.Description = ""
	current := targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive)
	current.DisplayName = common.String("stale display name")
	current.Description = common.String("stale description")
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "get id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{TargetAlertPolicyAssociation: current}, nil
	}
	fake.update = func(_ context.Context, request datasafesdk.UpdateTargetAlertPolicyAssociationRequest) (datasafesdk.UpdateTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "update id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		requireTargetAlertPolicyAssociationStringPtr(t, "update displayName", request.DisplayName, "")
		requireTargetAlertPolicyAssociationStringPtr(t, "update description", request.Description, "")
		return datasafesdk.UpdateTargetAlertPolicyAssociationResponse{
			OpcRequestId:     common.String("opc-update-clear-strings"),
			OpcWorkRequestId: common.String("wr-update-clear-strings"),
		}, nil
	}

	response, err := newTestTargetAlertPolicyAssociationClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want update work request requeue")
	}
	requireTargetAlertPolicyAssociationCallCount(t, "UpdateTargetAlertPolicyAssociation", len(fake.updateRequests), 1)
	requireTargetAlertPolicyAssociationString(t, "status.displayName", resource.Status.DisplayName, "")
	requireTargetAlertPolicyAssociationString(t, "status.description", resource.Status.Description, "")
	requireTargetAlertPolicyAssociationString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-update-clear-strings")
	requireTargetAlertPolicyAssociationWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-clear-strings", "IN_PROGRESS")
}

func TestTargetAlertPolicyAssociationForceNewDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTargetAlertPolicyAssociationID)
	current := targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive)
	current.PolicyId = common.String("ocid1.datasafealertpolicy.oc1..old")
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.get = func(context.Context, datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{TargetAlertPolicyAssociation: current}, nil
	}
	fake.update = func(context.Context, datasafesdk.UpdateTargetAlertPolicyAssociationRequest) (datasafesdk.UpdateTargetAlertPolicyAssociationResponse, error) {
		t.Fatal("UpdateTargetAlertPolicyAssociation() called despite policyId drift")
		return datasafesdk.UpdateTargetAlertPolicyAssociationResponse{}, nil
	}

	_, err := newTestTargetAlertPolicyAssociationClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want force-new drift rejection")
	}
	if !strings.Contains(err.Error(), "policyId") {
		t.Fatalf("CreateOrUpdate() error = %v, want policyId detail", err)
	}
	requireTargetAlertPolicyAssociationCallCount(t, "UpdateTargetAlertPolicyAssociation", len(fake.updateRequests), 0)
	requireTargetAlertPolicyAssociationLastCondition(t, resource, shared.Failed)
}

func TestTargetAlertPolicyAssociationDeleteStartsWorkRequestAndRetainsFinalizer(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	resource.Status.Id = testTargetAlertPolicyAssociationID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTargetAlertPolicyAssociationID)
	active := targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive)
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "get id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{TargetAlertPolicyAssociation: active}, nil
	}
	fake.delete = func(_ context.Context, request datasafesdk.DeleteTargetAlertPolicyAssociationRequest) (datasafesdk.DeleteTargetAlertPolicyAssociationResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "delete id", request.TargetAlertPolicyAssociationId, testTargetAlertPolicyAssociationID)
		return datasafesdk.DeleteTargetAlertPolicyAssociationResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}

	deleted, err := newTestTargetAlertPolicyAssociationClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	requireTargetAlertPolicyAssociationCallCount(t, "DeleteTargetAlertPolicyAssociation", len(fake.deleteRequests), 1)
	requireTargetAlertPolicyAssociationString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-delete")
	requireTargetAlertPolicyAssociationWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", "IN_PROGRESS")
}

func TestTargetAlertPolicyAssociationDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	resource.Status.Id = testTargetAlertPolicyAssociationID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTargetAlertPolicyAssociationID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	authErr.OpcRequestID = "opc-auth"
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.get = func(context.Context, datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{}, authErr
	}
	fake.delete = func(context.Context, datasafesdk.DeleteTargetAlertPolicyAssociationRequest) (datasafesdk.DeleteTargetAlertPolicyAssociationResponse, error) {
		t.Fatal("DeleteTargetAlertPolicyAssociation() called after ambiguous pre-delete read")
		return datasafesdk.DeleteTargetAlertPolicyAssociationResponse{}, nil
	}

	deleted, err := newTestTargetAlertPolicyAssociationClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	requireTargetAlertPolicyAssociationCallCount(t, "DeleteTargetAlertPolicyAssociation", len(fake.deleteRequests), 0)
	requireTargetAlertPolicyAssociationString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-auth")
}

func TestTargetAlertPolicyAssociationDeleteRejectsAuthShapedConfirmationAfterDeleteWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	resource.Status.Id = testTargetAlertPolicyAssociationID
	resource.Status.OsokStatus.Ocid = shared.OCID(testTargetAlertPolicyAssociationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		RawStatus:       "IN_PROGRESS",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	active := targetAlertPolicyAssociationBody(resource, testTargetAlertPolicyAssociationID, datasafesdk.AlertPolicyLifecycleStateActive)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous after work request")
	authErr.OpcRequestID = "opc-auth-after-work-request"
	getCalls := 0
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.get = func(context.Context, datasafesdk.GetTargetAlertPolicyAssociationRequest) (datasafesdk.GetTargetAlertPolicyAssociationResponse, error) {
		getCalls++
		if getCalls == 1 {
			return datasafesdk.GetTargetAlertPolicyAssociationResponse{TargetAlertPolicyAssociation: active}, nil
		}
		return datasafesdk.GetTargetAlertPolicyAssociationResponse{}, authErr
	}
	fake.getWorkRequest = func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
		requireTargetAlertPolicyAssociationStringPtr(t, "work request id", request.WorkRequestId, "wr-delete")
		return datasafesdk.GetWorkRequestResponse{
			WorkRequest: targetAlertPolicyAssociationWorkRequest("wr-delete", shared.OSOKAsyncPhaseDelete, datasafesdk.WorkRequestStatusSucceeded),
		}, nil
	}
	fake.delete = func(context.Context, datasafesdk.DeleteTargetAlertPolicyAssociationRequest) (datasafesdk.DeleteTargetAlertPolicyAssociationResponse, error) {
		t.Fatal("DeleteTargetAlertPolicyAssociation() called after ambiguous post-work-request confirmation")
		return datasafesdk.DeleteTargetAlertPolicyAssociationResponse{}, nil
	}

	deleted, err := newTestTargetAlertPolicyAssociationClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped confirmation error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	requireTargetAlertPolicyAssociationCallCount(t, "GetTargetAlertPolicyAssociation", len(fake.getRequests), 2)
	requireTargetAlertPolicyAssociationCallCount(t, "GetWorkRequest", len(fake.workRequestRequests), 1)
	requireTargetAlertPolicyAssociationCallCount(t, "DeleteTargetAlertPolicyAssociation", len(fake.deleteRequests), 0)
	requireTargetAlertPolicyAssociationString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-auth-after-work-request")
}

func TestTargetAlertPolicyAssociationCreateErrorRecordsOpcRequestID(t *testing.T) {
	t.Parallel()

	resource := newTargetAlertPolicyAssociationResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	fake := &fakeTargetAlertPolicyAssociationOCI{}
	fake.create = func(context.Context, datasafesdk.CreateTargetAlertPolicyAssociationRequest) (datasafesdk.CreateTargetAlertPolicyAssociationResponse, error) {
		return datasafesdk.CreateTargetAlertPolicyAssociationResponse{}, createErr
	}

	_, err := newTestTargetAlertPolicyAssociationClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	requireTargetAlertPolicyAssociationString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-create-error")
	requireTargetAlertPolicyAssociationLastCondition(t, resource, shared.Failed)
}

func newTargetAlertPolicyAssociationResource() *datasafev1beta1.TargetAlertPolicyAssociation {
	return &datasafev1beta1.TargetAlertPolicyAssociation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "target-alert-policy-association",
			Namespace: "default",
		},
		Spec: datasafev1beta1.TargetAlertPolicyAssociationSpec{
			PolicyId:      testTargetAlertPolicyAssociationPolicyID,
			TargetId:      testTargetAlertPolicyAssociationTargetID,
			CompartmentId: testTargetAlertPolicyAssociationCompartmentID,
			IsEnabled:     true,
			DisplayName:   testTargetAlertPolicyAssociationDisplayName,
			Description:   testTargetAlertPolicyAssociationDescription,
			FreeformTags:  map[string]string{"owner": "runtime"},
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func targetAlertPolicyAssociationBody(
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	id string,
	lifecycleState datasafesdk.AlertPolicyLifecycleStateEnum,
) datasafesdk.TargetAlertPolicyAssociation {
	created := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 5, 0, 0, time.UTC)}
	return datasafesdk.TargetAlertPolicyAssociation{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		TimeCreated:      &created,
		TimeUpdated:      &updated,
		LifecycleState:   lifecycleState,
		DisplayName:      common.String(resource.Spec.DisplayName),
		Description:      common.String(resource.Spec.Description),
		PolicyId:         common.String(resource.Spec.PolicyId),
		TargetId:         common.String(resource.Spec.TargetId),
		IsEnabled:        common.Bool(resource.Spec.IsEnabled),
		LifecycleDetails: common.String("lifecycle detail"),
		FreeformTags:     targetAlertPolicyAssociationStringMap(resource.Spec.FreeformTags),
		DefinedTags:      targetAlertPolicyAssociationDefinedTags(resource.Spec.DefinedTags),
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}},
	}
}

func targetAlertPolicyAssociationSummary(
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	id string,
	lifecycleState datasafesdk.AlertPolicyLifecycleStateEnum,
) datasafesdk.TargetAlertPolicyAssociationSummary {
	return targetAlertPolicyAssociationSummaryWithTarget(resource, id, resource.Spec.TargetId, lifecycleState)
}

func targetAlertPolicyAssociationSummaryWithTarget(
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	id string,
	targetID string,
	lifecycleState datasafesdk.AlertPolicyLifecycleStateEnum,
) datasafesdk.TargetAlertPolicyAssociationSummary {
	created := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 5, 0, 0, time.UTC)}
	return datasafesdk.TargetAlertPolicyAssociationSummary{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		TimeCreated:      &created,
		TimeUpdated:      &updated,
		LifecycleState:   lifecycleState,
		DisplayName:      common.String(resource.Spec.DisplayName),
		Description:      common.String(resource.Spec.Description),
		PolicyId:         common.String(resource.Spec.PolicyId),
		TargetId:         common.String(targetID),
		IsEnabled:        common.Bool(resource.Spec.IsEnabled),
		LifecycleDetails: common.String("lifecycle detail"),
		FreeformTags:     targetAlertPolicyAssociationStringMap(resource.Spec.FreeformTags),
		DefinedTags:      targetAlertPolicyAssociationDefinedTags(resource.Spec.DefinedTags),
	}
}

func targetAlertPolicyAssociationWorkRequest(
	id string,
	phase shared.OSOKAsyncPhase,
	status datasafesdk.WorkRequestStatusEnum,
) datasafesdk.WorkRequest {
	percent := float32(10)
	if status == datasafesdk.WorkRequestStatusSucceeded {
		percent = 100
	}
	action := datasafesdk.WorkRequestResourceActionTypeInProgress
	if status == datasafesdk.WorkRequestStatusSucceeded {
		switch phase {
		case shared.OSOKAsyncPhaseUpdate:
			action = datasafesdk.WorkRequestResourceActionTypeUpdated
		case shared.OSOKAsyncPhaseDelete:
			action = datasafesdk.WorkRequestResourceActionTypeDeleted
		default:
			action = datasafesdk.WorkRequestResourceActionTypeCreated
		}
	}
	return datasafesdk.WorkRequest{
		OperationType:   datasafesdk.WorkRequestOperationTypeTargetAlertPolicyAssociation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String(testTargetAlertPolicyAssociationCompartmentID),
		PercentComplete: common.Float32(percent),
		Resources: []datasafesdk.WorkRequestResource{
			{
				EntityType: common.String("target_alert_policy_association"),
				ActionType: action,
				Identifier: common.String(testTargetAlertPolicyAssociationID),
				EntityUri:  common.String("/targetAlertPolicyAssociations/" + testTargetAlertPolicyAssociationID),
			},
		},
	}
}

func requireTargetAlertPolicyAssociationCreateRequest(
	t *testing.T,
	request datasafesdk.CreateTargetAlertPolicyAssociationRequest,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
) {
	t.Helper()
	requireTargetAlertPolicyAssociationStringPtr(t, "create policyId", request.PolicyId, resource.Spec.PolicyId)
	requireTargetAlertPolicyAssociationStringPtr(t, "create targetId", request.TargetId, resource.Spec.TargetId)
	requireTargetAlertPolicyAssociationStringPtr(t, "create compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireTargetAlertPolicyAssociationBoolPtr(t, "create isEnabled", request.IsEnabled, resource.Spec.IsEnabled)
	requireTargetAlertPolicyAssociationStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	requireTargetAlertPolicyAssociationStringPtr(t, "create description", request.Description, resource.Spec.Description)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateTargetAlertPolicyAssociationRequest.OpcRetryToken is empty")
	}
}

func requireTargetAlertPolicyAssociationStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireTargetAlertPolicyAssociationBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}

func requireTargetAlertPolicyAssociationString(t *testing.T, name string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}

func requireTargetAlertPolicyAssociationCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func requireTargetAlertPolicyAssociationRecordedID(
	t *testing.T,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	want string,
) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func requireTargetAlertPolicyAssociationWorkRequestAsync(
	t *testing.T,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantRawStatus string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want work request tracker")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.status.async.current.source = %q, want workrequest", current.Source)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
	if current.RawStatus != wantRawStatus {
		t.Fatalf("status.status.async.current.rawStatus = %q, want %q", current.RawStatus, wantRawStatus)
	}
}

func requireTargetAlertPolicyAssociationLastCondition(
	t *testing.T,
	resource *datasafev1beta1.TargetAlertPolicyAssociation,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}
