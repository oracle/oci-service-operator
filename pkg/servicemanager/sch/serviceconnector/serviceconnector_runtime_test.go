/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package serviceconnector

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	schsdk "github.com/oracle/oci-go-sdk/v65/sch"
	schv1beta1 "github.com/oracle/oci-service-operator/api/sch/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testServiceConnectorID            = "ocid1.serviceconnector.oc1..serviceconnector"
	testServiceConnectorOtherID       = "ocid1.serviceconnector.oc1..other"
	testServiceConnectorCompartmentID = "ocid1.compartment.oc1..serviceconnector"
	testServiceConnectorDisplayName   = "service-connector-sample"
	testServiceConnectorDescription   = "desired connector"
	testServiceConnectorTopicID       = "ocid1.onstopic.oc1..serviceconnector"
	testServiceConnectorLogGroupID    = "_Audit"
	testServiceConnectorLogID         = "ocid1.log.oc1..serviceconnector"
)

func TestServiceConnectorRuntimeSemantics(t *testing.T) {
	t.Parallel()

	hooks := newServiceConnectorRuntimeHooksWithOCIClient(&fakeServiceConnectorOCIClient{})
	applyServiceConnectorRuntimeHooks(
		&ServiceConnectorServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
		&hooks,
		&fakeServiceConnectorOCIClient{},
		nil,
	)

	assertServiceConnectorBaseSemantics(t, hooks)
	assertServiceConnectorMutationSemantics(t, hooks)
	assertServiceConnectorCustomHooks(t, hooks)
}

func assertServiceConnectorBaseSemantics(t *testing.T, hooks ServiceConnectorRuntimeHooks) {
	t.Helper()

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed ServiceConnector semantics")
	}
	if got := hooks.Semantics.FormalService; got != "sch" {
		t.Fatalf("FormalService = %q, want sch", got)
	}
	if got := hooks.Semantics.FormalSlug; got != "serviceconnector" {
		t.Fatalf("FormalSlug = %q, want serviceconnector", got)
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "workrequest" || hooks.Semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest semantics", hooks.Semantics.Async)
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	if got := hooks.Semantics.DeleteFollowUp.Strategy; got != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got)
	}
}

func assertServiceConnectorMutationSemantics(t *testing.T, hooks ServiceConnectorRuntimeHooks) {
	t.Helper()

	assertServiceConnectorContainsAll(t, hooks.Semantics.List.MatchFields, "compartmentId", "displayName")
	assertServiceConnectorContainsAll(t, hooks.Semantics.Mutation.Mutable, "displayName", "description", "source", "target", "tasks", "freeformTags", "definedTags")
	assertServiceConnectorContainsAll(t, hooks.Semantics.Mutation.ForceNew, "compartmentId")
}

func assertServiceConnectorCustomHooks(t *testing.T, hooks ServiceConnectorRuntimeHooks) {
	t.Helper()

	if hooks.BuildCreateBody == nil {
		t.Fatal("BuildCreateBody = nil, want resource-local create request shaping")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("BuildUpdateBody = nil, want resource-local update request shaping")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("WrapGeneratedClient count = %d, want delete confirmation wrapper", len(hooks.WrapGeneratedClient))
	}
}

func TestServiceConnectorBuildCreateBodyUsesPolymorphicDetails(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Spec.Target.EnableFormattedMessaging = false

	details, err := buildServiceConnectorCreateDetails(context.Background(), &ServiceConnectorServiceManager{}, resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildServiceConnectorCreateDetails() error = %v", err)
	}

	source, ok := details.Source.(schsdk.LoggingSourceDetails)
	if !ok {
		t.Fatalf("details.Source = %T, want sch.LoggingSourceDetails", details.Source)
	}
	if len(source.LogSources) != 1 {
		t.Fatalf("source.LogSources length = %d, want 1", len(source.LogSources))
	}
	requireServiceConnectorStringPtr(t, "source logGroupId", source.LogSources[0].LogGroupId, testServiceConnectorLogGroupID)

	target, ok := details.Target.(schsdk.NotificationsTargetDetails)
	if !ok {
		t.Fatalf("details.Target = %T, want sch.NotificationsTargetDetails", details.Target)
	}
	requireServiceConnectorStringPtr(t, "target topicId", target.TopicId, testServiceConnectorTopicID)
	if target.EnableFormattedMessaging == nil || *target.EnableFormattedMessaging {
		t.Fatalf("target.EnableFormattedMessaging = %v, want explicit false pointer", target.EnableFormattedMessaging)
	}

	if len(details.Tasks) != 1 {
		t.Fatalf("details.Tasks length = %d, want 1", len(details.Tasks))
	}
	task, ok := details.Tasks[0].(schsdk.LogRuleTaskDetails)
	if !ok {
		t.Fatalf("details.Tasks[0] = %T, want sch.LogRuleTaskDetails", details.Tasks[0])
	}
	requireServiceConnectorStringPtr(t, "task condition", task.Condition, "logContent='ERROR'")
}

func TestServiceConnectorWorkRequestRecoveryIgnoresUnrelatedEmptyEntityType(t *testing.T) {
	t.Parallel()

	workRequest := makeServiceConnectorWorkRequest(
		"wr-create",
		schsdk.OperationTypeCreateServiceConnector,
		schsdk.OperationStatusSucceeded,
		schsdk.ActionTypeCreated,
		"",
	)
	workRequest.Resources = []schsdk.WorkRequestResource{
		{
			ActionType: schsdk.ActionTypeCreated,
			Identifier: common.String("ocid1.log.oc1..notserviceconnector"),
		},
		{
			ActionType: schsdk.ActionTypeCreated,
			Identifier: common.String(testServiceConnectorID),
		},
	}

	got, err := resolveServiceConnectorIDFromWorkRequest(workRequest, schsdk.ActionTypeCreated)
	if err != nil {
		t.Fatalf("resolveServiceConnectorIDFromWorkRequest() error = %v", err)
	}
	if got != testServiceConnectorID {
		t.Fatalf("resolved service connector ID = %q, want %q", got, testServiceConnectorID)
	}
}

func TestServiceConnectorCreateTracksPendingWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	fake := newFakeServiceConnectorPendingCreateClient(t, resource)

	response, err := newTestServiceConnectorClient(fake).CreateOrUpdate(context.Background(), resource, serviceConnectorReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertServiceConnectorPendingCreateResponse(t, response, resource, fake)
}

func newFakeServiceConnectorPendingCreateClient(t *testing.T, resource *schv1beta1.ServiceConnector) *fakeServiceConnectorOCIClient {
	t.Helper()

	fake := &fakeServiceConnectorOCIClient{}
	fake.listFunc = func(context.Context, schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error) {
		return schsdk.ListServiceConnectorsResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request schsdk.CreateServiceConnectorRequest) (schsdk.CreateServiceConnectorResponse, error) {
		assertServiceConnectorPendingCreateRequest(t, request, resource)
		return schsdk.CreateServiceConnectorResponse{
			OpcWorkRequestId: common.String("wr-create"),
			OpcRequestId:     common.String("opc-create"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
		return schsdk.GetWorkRequestResponse{
			WorkRequest: makeServiceConnectorWorkRequest(
				"wr-create",
				schsdk.OperationTypeCreateServiceConnector,
				schsdk.OperationStatusInProgress,
				schsdk.ActionTypeCreated,
				"",
			),
		}, nil
	}
	return fake
}

func assertServiceConnectorPendingCreateRequest(
	t *testing.T,
	request schsdk.CreateServiceConnectorRequest,
	resource *schv1beta1.ServiceConnector,
) {
	t.Helper()

	requireServiceConnectorStringPtr(t, "create opcRetryToken", request.OpcRetryToken, string(resource.UID))
	target, ok := request.Target.(schsdk.NotificationsTargetDetails)
	if !ok {
		t.Fatalf("create target = %T, want sch.NotificationsTargetDetails", request.Target)
	}
	if target.EnableFormattedMessaging == nil || *target.EnableFormattedMessaging {
		t.Fatalf("create target EnableFormattedMessaging = %v, want explicit false pointer", target.EnableFormattedMessaging)
	}
}

func assertServiceConnectorPendingCreateResponse(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *schv1beta1.ServiceConnector,
	fake *fakeServiceConnectorOCIClient,
) {
	t.Helper()

	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	assertServiceConnectorPendingCreateCalls(t, fake)
	assertServiceConnectorPendingCreateAsync(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
}

func assertServiceConnectorPendingCreateCalls(t *testing.T, fake *fakeServiceConnectorOCIClient) {
	t.Helper()

	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateServiceConnector calls = %d, want 1", len(fake.createRequests))
	}
	if len(fake.getRequests) != 0 {
		t.Fatalf("GetServiceConnector calls = %d, want 0 while create work request is pending", len(fake.getRequests))
	}
}

func assertServiceConnectorPendingCreateAsync(t *testing.T, resource *schv1beta1.ServiceConnector) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseCreate || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.current = %#v, want pending create work request", current)
	}
	if got := current.WorkRequestID; got != "wr-create" {
		t.Fatalf("async.current.workRequestId = %q, want wr-create", got)
	}
}

func TestServiceConnectorCreateSucceededWorkRequestReadsObservedState(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	fake := &fakeServiceConnectorOCIClient{}
	fake.listFunc = func(context.Context, schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error) {
		return schsdk.ListServiceConnectorsResponse{}, nil
	}
	fake.createFunc = func(context.Context, schsdk.CreateServiceConnectorRequest) (schsdk.CreateServiceConnectorResponse, error) {
		return schsdk.CreateServiceConnectorResponse{
			OpcWorkRequestId: common.String("wr-create"),
			OpcRequestId:     common.String("opc-create"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
		return schsdk.GetWorkRequestResponse{
			WorkRequest: makeServiceConnectorWorkRequest(
				"wr-create",
				schsdk.OperationTypeCreateServiceConnector,
				schsdk.OperationStatusSucceeded,
				schsdk.ActionTypeCreated,
				testServiceConnectorID,
			),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		requireServiceConnectorStringPtr(t, "get serviceConnectorId", request.ServiceConnectorId, testServiceConnectorID)
		return schsdk.GetServiceConnectorResponse{
			ServiceConnector: makeSDKServiceConnector(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName, testServiceConnectorDescription, schsdk.LifecycleStateActive),
		}, nil
	}

	response, err := newTestServiceConnectorClient(fake).CreateOrUpdate(context.Background(), resource, serviceConnectorReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success", response)
	}
	if got := resource.Status.Id; got != testServiceConnectorID {
		t.Fatalf("status.id = %q, want %q", got, testServiceConnectorID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testServiceConnectorID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testServiceConnectorID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async.current = %#v, want nil after succeeded create", resource.Status.OsokStatus.Async.Current)
	}
	assertServiceConnectorTrailingCondition(t, resource, shared.Active)
}

func TestServiceConnectorCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	fake := &fakeServiceConnectorOCIClient{}
	fake.listFunc = pagedServiceConnectorList(t, []serviceConnectorListPage{
		{
			items: []schsdk.ServiceConnectorSummary{
				makeSDKServiceConnectorSummary(testServiceConnectorOtherID, testServiceConnectorCompartmentID, "other-connector"),
			},
			nextPage: "page-2",
		},
		{
			wantPage: "page-2",
			items: []schsdk.ServiceConnectorSummary{
				makeSDKServiceConnectorSummary(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName),
			},
		},
	})
	fake.getFunc = func(_ context.Context, request schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		requireServiceConnectorStringPtr(t, "get serviceConnectorId", request.ServiceConnectorId, testServiceConnectorID)
		return schsdk.GetServiceConnectorResponse{
			ServiceConnector: makeSDKServiceConnector(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName, testServiceConnectorDescription, schsdk.LifecycleStateActive),
		}, nil
	}

	response, err := newTestServiceConnectorClient(fake).CreateOrUpdate(context.Background(), resource, serviceConnectorReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListServiceConnectors calls = %d, want 2", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateServiceConnector calls = %d, want 0", len(fake.createRequests))
	}
	if got := resource.Status.Id; got != testServiceConnectorID {
		t.Fatalf("status.id = %q, want %q", got, testServiceConnectorID)
	}
	assertServiceConnectorTrailingCondition(t, resource, shared.Active)
}

func TestServiceConnectorNoopDoesNotUpdateWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	fake := &fakeServiceConnectorOCIClient{}
	fake.getFunc = func(_ context.Context, request schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		requireServiceConnectorStringPtr(t, "get serviceConnectorId", request.ServiceConnectorId, testServiceConnectorID)
		return schsdk.GetServiceConnectorResponse{
			ServiceConnector: makeSDKServiceConnector(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName, testServiceConnectorDescription, schsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, schsdk.UpdateServiceConnectorRequest) (schsdk.UpdateServiceConnectorResponse, error) {
		t.Fatal("UpdateServiceConnector should not be called when desired state matches readback")
		return schsdk.UpdateServiceConnectorResponse{}, nil
	}

	response, err := newTestServiceConnectorClient(fake).CreateOrUpdate(context.Background(), resource, serviceConnectorReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateServiceConnector calls = %d, want 0", len(fake.updateRequests))
	}
	assertServiceConnectorTrailingCondition(t, resource, shared.Active)
}

func TestServiceConnectorMutableUpdateUsesWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Spec.Description = "updated description"
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	fake := &fakeServiceConnectorOCIClient{}
	getCalls := 0
	fake.getFunc = func(_ context.Context, request schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		requireServiceConnectorStringPtr(t, "get serviceConnectorId", request.ServiceConnectorId, testServiceConnectorID)
		getCalls++
		description := testServiceConnectorDescription
		if getCalls > 1 {
			description = resource.Spec.Description
		}
		return schsdk.GetServiceConnectorResponse{
			ServiceConnector: makeSDKServiceConnector(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName, description, schsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(_ context.Context, request schsdk.UpdateServiceConnectorRequest) (schsdk.UpdateServiceConnectorResponse, error) {
		requireServiceConnectorStringPtr(t, "update serviceConnectorId", request.ServiceConnectorId, testServiceConnectorID)
		requireServiceConnectorStringPtr(t, "update description", request.Description, resource.Spec.Description)
		if request.Source != nil {
			t.Fatalf("update Source = %#v, want nil for unchanged source", request.Source)
		}
		return schsdk.UpdateServiceConnectorResponse{
			OpcWorkRequestId: common.String("wr-update"),
			OpcRequestId:     common.String("opc-update"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
		return schsdk.GetWorkRequestResponse{
			WorkRequest: makeServiceConnectorWorkRequest(
				"wr-update",
				schsdk.OperationTypeUpdateServiceConnector,
				schsdk.OperationStatusSucceeded,
				schsdk.ActionTypeUpdated,
				testServiceConnectorID,
			),
		}, nil
	}

	response, err := newTestServiceConnectorClient(fake).CreateOrUpdate(context.Background(), resource, serviceConnectorReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want updating success", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateServiceConnector calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	assertServiceConnectorTrailingCondition(t, resource, shared.Active)
}

func TestServiceConnectorCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	fake := &fakeServiceConnectorOCIClient{}
	fake.getFunc = func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		return schsdk.GetServiceConnectorResponse{
			ServiceConnector: makeSDKServiceConnector(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName, testServiceConnectorDescription, schsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, schsdk.UpdateServiceConnectorRequest) (schsdk.UpdateServiceConnectorResponse, error) {
		t.Fatal("UpdateServiceConnector should not be called when compartmentId drift is create-only")
		return schsdk.UpdateServiceConnectorResponse{}, nil
	}

	response, err := newTestServiceConnectorClient(fake).CreateOrUpdate(context.Background(), resource, serviceConnectorReconcileRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateServiceConnector calls = %d, want 0", len(fake.updateRequests))
	}
	assertServiceConnectorTrailingCondition(t, resource, shared.Failed)
}

func TestServiceConnectorDeleteWaitsForPendingCreateOrUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range []serviceConnectorPendingDeleteWorkRequestCase{
		{
			name:   "create",
			phase:  shared.OSOKAsyncPhaseCreate,
			op:     schsdk.OperationTypeCreateServiceConnector,
			action: schsdk.ActionTypeCreated,
		},
		{
			name:   "update",
			phase:  shared.OSOKAsyncPhaseUpdate,
			op:     schsdk.OperationTypeUpdateServiceConnector,
			action: schsdk.ActionTypeUpdated,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runServiceConnectorDeleteWaitsForPendingWorkRequest(t, tc)
		})
	}
}

type serviceConnectorPendingDeleteWorkRequestCase struct {
	name   string
	phase  shared.OSOKAsyncPhase
	op     schsdk.OperationTypeEnum
	action schsdk.ActionTypeEnum
}

func runServiceConnectorDeleteWaitsForPendingWorkRequest(t *testing.T, tc serviceConnectorPendingDeleteWorkRequestCase) {
	t.Helper()

	resource := newServiceConnectorDeletingResource(tc.phase, "wr-"+tc.name)
	fake := newFakeServiceConnectorPendingWorkRequestBeforeDeleteClient(t, tc)

	deleted, err := newTestServiceConnectorClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertServiceConnectorPendingWorkRequestBeforeDelete(t, deleted, resource, fake, tc)
}

func newServiceConnectorDeletingResource(phase shared.OSOKAsyncPhase, workRequestID string) *schv1beta1.ServiceConnector {
	resource := newServiceConnectorResource()
	resource.Finalizers = []string{"osok-finalizer"}
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	seedServiceConnectorCurrentWorkRequest(resource, phase, workRequestID)
	return resource
}

func newFakeServiceConnectorPendingWorkRequestBeforeDeleteClient(
	t *testing.T,
	tc serviceConnectorPendingDeleteWorkRequestCase,
) *fakeServiceConnectorOCIClient {
	t.Helper()

	workRequestID := "wr-" + tc.name
	fake := &fakeServiceConnectorOCIClient{}
	fake.workRequestFunc = func(_ context.Context, request schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
		requireServiceConnectorStringPtr(t, "workRequestId", request.WorkRequestId, workRequestID)
		return schsdk.GetWorkRequestResponse{
			WorkRequest: makeServiceConnectorWorkRequest(
				workRequestID,
				tc.op,
				schsdk.OperationStatusInProgress,
				tc.action,
				testServiceConnectorID,
			),
		}, nil
	}
	fake.getFunc = func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		t.Fatal("GetServiceConnector should not be called while write work request is pending before delete")
		return schsdk.GetServiceConnectorResponse{}, nil
	}
	fake.deleteFunc = func(context.Context, schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error) {
		t.Fatal("DeleteServiceConnector should not be called while write work request is pending")
		return schsdk.DeleteServiceConnectorResponse{}, nil
	}
	return fake
}

func assertServiceConnectorPendingWorkRequestBeforeDelete(
	t *testing.T,
	deleted bool,
	resource *schv1beta1.ServiceConnector,
	fake *fakeServiceConnectorOCIClient,
	tc serviceConnectorPendingDeleteWorkRequestCase,
) {
	t.Helper()

	if deleted {
		t.Fatal("Delete() deleted = true, want false while write work request is pending")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteServiceConnector calls = %d, want 0 before write work request finishes", len(fake.deleteRequests))
	}
	if len(fake.getRequests) != 0 {
		t.Fatalf("GetServiceConnector calls = %d, want 0 before write work request finishes", len(fake.getRequests))
	}
	if len(resource.Finalizers) != 1 || resource.Finalizers[0] != "osok-finalizer" {
		t.Fatalf("finalizers = %#v, want retained osok-finalizer", resource.Finalizers)
	}
	requireServiceConnectorCurrentWorkRequest(t, resource, tc.phase, "wr-"+tc.name, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.Async.Current.Message; !strings.Contains(got, "waiting before delete") {
		t.Fatalf("async.current.message = %q, want waiting-before-delete detail", got)
	}
}

func TestServiceConnectorDeletePendingWorkRequestKeepsFinalizer(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	fake := &fakeServiceConnectorOCIClient{}
	fake.getFunc = func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		return schsdk.GetServiceConnectorResponse{
			ServiceConnector: makeSDKServiceConnector(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName, testServiceConnectorDescription, schsdk.LifecycleStateActive),
		}, nil
	}
	fake.deleteFunc = func(_ context.Context, request schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error) {
		requireServiceConnectorStringPtr(t, "delete serviceConnectorId", request.ServiceConnectorId, testServiceConnectorID)
		return schsdk.DeleteServiceConnectorResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("opc-delete"),
		}, nil
	}
	fake.workRequestFunc = func(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
		return schsdk.GetWorkRequestResponse{
			WorkRequest: makeServiceConnectorWorkRequest(
				"wr-delete",
				schsdk.OperationTypeDeleteServiceConnector,
				schsdk.OperationStatusAccepted,
				schsdk.ActionTypeDeleted,
				testServiceConnectorID,
			),
		}, nil
	}

	deleted, err := newTestServiceConnectorClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteServiceConnector calls = %d, want 1", len(fake.deleteRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.current = %#v, want pending delete work request", current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
}

func TestServiceConnectorDeleteRejectsAuthShapedDeleteCall(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	fake := &fakeServiceConnectorOCIClient{}
	fake.getFunc = func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		return schsdk.GetServiceConnectorResponse{
			ServiceConnector: makeSDKServiceConnector(testServiceConnectorID, testServiceConnectorCompartmentID, testServiceConnectorDisplayName, testServiceConnectorDescription, schsdk.LifecycleStateActive),
		}, nil
	}
	fake.deleteFunc = func(context.Context, schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error) {
		return schsdk.DeleteServiceConnectorResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newTestServiceConnectorClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped delete 404")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteServiceConnector calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id from OCI error", got)
	}
}

func TestServiceConnectorDeleteSucceededWorkRequestConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeServiceConnectorOCIClient{}
	fake.getFunc = func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		return schsdk.GetServiceConnectorResponse{}, errortest.NewServiceError(404, "NotFound", "service connector deleted")
	}
	fake.workRequestFunc = func(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
		return schsdk.GetWorkRequestResponse{
			WorkRequest: makeServiceConnectorWorkRequest(
				"wr-delete",
				schsdk.OperationTypeDeleteServiceConnector,
				schsdk.OperationStatusSucceeded,
				schsdk.ActionTypeDeleted,
				testServiceConnectorID,
			),
		}, nil
	}

	deleted, err := newTestServiceConnectorClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after succeeded work request and unambiguous not-found")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteServiceConnector calls = %d, want 0 while resuming tracked delete work request", len(fake.deleteRequests))
	}
	assertServiceConnectorTrailingCondition(t, resource, shared.Terminating)
}

func TestServiceConnectorDeleteSucceededWorkRequestRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorDeletingResource(shared.OSOKAsyncPhaseDelete, "wr-delete")
	fake := &fakeServiceConnectorOCIClient{}
	fake.workRequestFunc = func(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
		return schsdk.GetWorkRequestResponse{
			WorkRequest: makeServiceConnectorWorkRequest(
				"wr-delete",
				schsdk.OperationTypeDeleteServiceConnector,
				schsdk.OperationStatusSucceeded,
				schsdk.ActionTypeDeleted,
				testServiceConnectorID,
			),
		}, nil
	}
	fake.getFunc = func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		return schsdk.GetServiceConnectorResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
	fake.deleteFunc = func(context.Context, schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error) {
		t.Fatal("DeleteServiceConnector should not be called after ambiguous delete work request confirm read")
		return schsdk.DeleteServiceConnectorResponse{}, nil
	}

	deleted, err := newTestServiceConnectorClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteServiceConnector calls = %d, want 0", len(fake.deleteRequests))
	}
	if len(resource.Finalizers) != 1 || resource.Finalizers[0] != "osok-finalizer" {
		t.Fatalf("finalizers = %#v, want retained osok-finalizer", resource.Finalizers)
	}
	requireServiceConnectorCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassFailed)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id from OCI error", got)
	}
}

func TestServiceConnectorDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newServiceConnectorResource()
	resource.Status.Id = testServiceConnectorID
	resource.Status.OsokStatus.Ocid = shared.OCID(testServiceConnectorID)
	fake := &fakeServiceConnectorOCIClient{}
	fake.getFunc = func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
		return schsdk.GetServiceConnectorResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
	fake.deleteFunc = func(context.Context, schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error) {
		t.Fatal("DeleteServiceConnector should not be called after ambiguous confirm read")
		return schsdk.DeleteServiceConnectorResponse{}, nil
	}

	deleted, err := newTestServiceConnectorClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteServiceConnector calls = %d, want 0", len(fake.deleteRequests))
	}
}

func newTestServiceConnectorClient(fake *fakeServiceConnectorOCIClient) ServiceConnectorServiceClient {
	return newServiceConnectorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func newServiceConnectorResource() *schv1beta1.ServiceConnector {
	return &schv1beta1.ServiceConnector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServiceConnectorDisplayName,
			Namespace: "default",
			UID:       k8stypes.UID("service-connector-uid"),
		},
		Spec: schv1beta1.ServiceConnectorSpec{
			DisplayName:   testServiceConnectorDisplayName,
			CompartmentId: testServiceConnectorCompartmentID,
			Description:   testServiceConnectorDescription,
			Source: schv1beta1.ServiceConnectorSource{
				Kind: "logging",
				LogSources: []schv1beta1.ServiceConnectorSourceLogSource{
					{
						CompartmentId: testServiceConnectorCompartmentID,
						LogGroupId:    testServiceConnectorLogGroupID,
						LogId:         testServiceConnectorLogID,
					},
				},
			},
			Target: schv1beta1.ServiceConnectorTarget{
				Kind:                     "notifications",
				TopicId:                  testServiceConnectorTopicID,
				EnableFormattedMessaging: false,
			},
			Tasks: []schv1beta1.ServiceConnectorTask{
				{
					Kind:      "logRule",
					Condition: "logContent='ERROR'",
				},
			},
			FreeformTags: map[string]string{"managed-by": "osok"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func serviceConnectorReconcileRequest(resource *schv1beta1.ServiceConnector) ctrl.Request {
	return ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func seedServiceConnectorCurrentWorkRequest(resource *schv1beta1.ServiceConnector, phase shared.OSOKAsyncPhase, workRequestID string) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func makeSDKServiceConnector(
	id string,
	compartmentID string,
	displayName string,
	description string,
	state schsdk.LifecycleStateEnum,
) schsdk.ServiceConnector {
	return schsdk.ServiceConnector{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: state,
		Description:    common.String(description),
		Source: schsdk.LoggingSourceDetailsResponse{
			LogSources: []schsdk.LogSource{
				{
					CompartmentId: common.String(testServiceConnectorCompartmentID),
					LogGroupId:    common.String(testServiceConnectorLogGroupID),
					LogId:         common.String(testServiceConnectorLogID),
				},
			},
		},
		Target: schsdk.NotificationsTargetDetailsResponse{
			TopicId:                  common.String(testServiceConnectorTopicID),
			EnableFormattedMessaging: common.Bool(false),
		},
		Tasks: []schsdk.TaskDetailsResponse{
			schsdk.LogRuleTaskDetailsResponse{Condition: common.String("logContent='ERROR'")},
		},
		FreeformTags: map[string]string{"managed-by": "osok"},
		DefinedTags:  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:   map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func makeSDKServiceConnectorSummary(id string, compartmentID string, displayName string) schsdk.ServiceConnectorSummary {
	return schsdk.ServiceConnectorSummary{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: schsdk.LifecycleStateActive,
		Description:    common.String(testServiceConnectorDescription),
		FreeformTags:   map[string]string{"managed-by": "osok"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeServiceConnectorWorkRequest(
	id string,
	operationType schsdk.OperationTypeEnum,
	status schsdk.OperationStatusEnum,
	action schsdk.ActionTypeEnum,
	serviceConnectorID string,
) schsdk.WorkRequest {
	percentComplete := float32(42)
	workRequest := schsdk.WorkRequest{
		Id:              common.String(id),
		CompartmentId:   common.String(testServiceConnectorCompartmentID),
		OperationType:   operationType,
		Status:          status,
		PercentComplete: &percentComplete,
	}
	if serviceConnectorID != "" {
		workRequest.Resources = []schsdk.WorkRequestResource{
			{
				EntityType: common.String("serviceConnector"),
				ActionType: action,
				Identifier: common.String(serviceConnectorID),
				EntityUri:  common.String("/20200901/serviceConnectors/" + serviceConnectorID),
			},
		}
	}
	return workRequest
}

type serviceConnectorListPage struct {
	wantPage string
	items    []schsdk.ServiceConnectorSummary
	nextPage string
}

func pagedServiceConnectorList(t *testing.T, pages []serviceConnectorListPage) func(context.Context, schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error) {
	t.Helper()
	index := 0
	return func(_ context.Context, request schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error) {
		if index >= len(pages) {
			t.Fatalf("ListServiceConnectors call %d beyond configured pages", index+1)
		}
		page := pages[index]
		index++
		if got := stringValue(request.Page); got != page.wantPage {
			t.Fatalf("ListServiceConnectors page = %q, want %q", got, page.wantPage)
		}
		requireServiceConnectorStringPtr(t, "list compartmentId", request.CompartmentId, testServiceConnectorCompartmentID)
		requireServiceConnectorStringPtr(t, "list displayName", request.DisplayName, testServiceConnectorDisplayName)
		response := schsdk.ListServiceConnectorsResponse{
			ServiceConnectorCollection: schsdk.ServiceConnectorCollection{Items: page.items},
		}
		if page.nextPage != "" {
			response.OpcNextPage = common.String(page.nextPage)
		}
		return response, nil
	}
}

type fakeServiceConnectorOCIClient struct {
	createFunc      func(context.Context, schsdk.CreateServiceConnectorRequest) (schsdk.CreateServiceConnectorResponse, error)
	getFunc         func(context.Context, schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error)
	listFunc        func(context.Context, schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error)
	updateFunc      func(context.Context, schsdk.UpdateServiceConnectorRequest) (schsdk.UpdateServiceConnectorResponse, error)
	deleteFunc      func(context.Context, schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error)
	workRequestFunc func(context.Context, schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error)

	createRequests      []schsdk.CreateServiceConnectorRequest
	getRequests         []schsdk.GetServiceConnectorRequest
	listRequests        []schsdk.ListServiceConnectorsRequest
	updateRequests      []schsdk.UpdateServiceConnectorRequest
	deleteRequests      []schsdk.DeleteServiceConnectorRequest
	workRequestRequests []schsdk.GetWorkRequestRequest
}

func (f *fakeServiceConnectorOCIClient) CreateServiceConnector(ctx context.Context, request schsdk.CreateServiceConnectorRequest) (schsdk.CreateServiceConnectorResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return schsdk.CreateServiceConnectorResponse{}, nil
}

func (f *fakeServiceConnectorOCIClient) GetServiceConnector(ctx context.Context, request schsdk.GetServiceConnectorRequest) (schsdk.GetServiceConnectorResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return schsdk.GetServiceConnectorResponse{}, nil
}

func (f *fakeServiceConnectorOCIClient) ListServiceConnectors(ctx context.Context, request schsdk.ListServiceConnectorsRequest) (schsdk.ListServiceConnectorsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return schsdk.ListServiceConnectorsResponse{}, nil
}

func (f *fakeServiceConnectorOCIClient) UpdateServiceConnector(ctx context.Context, request schsdk.UpdateServiceConnectorRequest) (schsdk.UpdateServiceConnectorResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return schsdk.UpdateServiceConnectorResponse{}, nil
}

func (f *fakeServiceConnectorOCIClient) DeleteServiceConnector(ctx context.Context, request schsdk.DeleteServiceConnectorRequest) (schsdk.DeleteServiceConnectorResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return schsdk.DeleteServiceConnectorResponse{}, nil
}

func (f *fakeServiceConnectorOCIClient) GetWorkRequest(ctx context.Context, request schsdk.GetWorkRequestRequest) (schsdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFunc != nil {
		return f.workRequestFunc(ctx, request)
	}
	return schsdk.GetWorkRequestResponse{}, nil
}

func requireServiceConnectorStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertServiceConnectorTrailingCondition(t *testing.T, resource *schv1beta1.ServiceConnector, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %s, want %s", got, want)
	}
}

func requireServiceConnectorCurrentWorkRequest(
	t *testing.T,
	resource *schv1beta1.ServiceConnector,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.async.current = nil, want %s work request %s", phase, workRequestID)
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != phase ||
		current.WorkRequestID != workRequestID ||
		current.NormalizedClass != class {
		t.Fatalf(
			"status.async.current = %#v, want source=%s phase=%s workRequestID=%q class=%s",
			current,
			shared.OSOKAsyncSourceWorkRequest,
			phase,
			workRequestID,
			class,
		)
	}
}

func assertServiceConnectorContainsAll(t *testing.T, got []string, want ...string) {
	t.Helper()
	seen := make(map[string]bool, len(got))
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("slice %#v missing %q", got, item)
		}
	}
}
