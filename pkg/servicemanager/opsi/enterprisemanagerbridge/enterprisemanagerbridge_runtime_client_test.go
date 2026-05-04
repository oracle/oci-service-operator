/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package enterprisemanagerbridge

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testEnterpriseManagerBridgeID          = "ocid1.enterprisemanagerbridge.oc1..example"
	testEnterpriseManagerBridgeCompartment = "ocid1.compartment.oc1..example"
	testEnterpriseManagerBridgeName        = "em-bridge"
	testEnterpriseManagerBridgeBucket      = "em-bucket"
)

type fakeEnterpriseManagerBridgeOCIClient struct {
	createRequests      []opsisdk.CreateEnterpriseManagerBridgeRequest
	getRequests         []opsisdk.GetEnterpriseManagerBridgeRequest
	listRequests        []opsisdk.ListEnterpriseManagerBridgesRequest
	updateRequests      []opsisdk.UpdateEnterpriseManagerBridgeRequest
	deleteRequests      []opsisdk.DeleteEnterpriseManagerBridgeRequest
	workRequestRequests []opsisdk.GetWorkRequestRequest

	createResponse  opsisdk.CreateEnterpriseManagerBridgeResponse
	createErr       error
	getResponses    map[string]opsisdk.EnterpriseManagerBridge
	getErrs         map[string]error
	listResponses   []opsisdk.ListEnterpriseManagerBridgesResponse
	listErr         error
	updateResponse  opsisdk.UpdateEnterpriseManagerBridgeResponse
	updateErr       error
	deleteResponse  opsisdk.DeleteEnterpriseManagerBridgeResponse
	deleteErr       error
	workRequests    map[string]opsisdk.WorkRequest
	workRequestErrs map[string]error
}

func (f *fakeEnterpriseManagerBridgeOCIClient) CreateEnterpriseManagerBridge(_ context.Context, request opsisdk.CreateEnterpriseManagerBridgeRequest) (opsisdk.CreateEnterpriseManagerBridgeResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return opsisdk.CreateEnterpriseManagerBridgeResponse{}, f.createErr
	}
	if f.createResponse.OpcWorkRequestId != nil || f.createResponse.Id != nil {
		return f.createResponse, nil
	}
	return opsisdk.CreateEnterpriseManagerBridgeResponse{
		EnterpriseManagerBridge: makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, request.DisplayName, request.ObjectStorageBucketName, opsisdk.LifecycleStateCreating),
		OpcWorkRequestId:        common.String("wr-create"),
		OpcRequestId:            common.String("opc-create"),
	}, nil
}

func (f *fakeEnterpriseManagerBridgeOCIClient) GetEnterpriseManagerBridge(_ context.Context, request opsisdk.GetEnterpriseManagerBridgeRequest) (opsisdk.GetEnterpriseManagerBridgeResponse, error) {
	f.getRequests = append(f.getRequests, request)
	resourceID := stringValue(request.EnterpriseManagerBridgeId)
	if err := f.getErrs[resourceID]; err != nil {
		return opsisdk.GetEnterpriseManagerBridgeResponse{}, err
	}
	if response, ok := f.getResponses[resourceID]; ok {
		return opsisdk.GetEnterpriseManagerBridgeResponse{EnterpriseManagerBridge: response}, nil
	}
	return opsisdk.GetEnterpriseManagerBridgeResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "EnterpriseManagerBridge not found")
}

func (f *fakeEnterpriseManagerBridgeOCIClient) ListEnterpriseManagerBridges(_ context.Context, request opsisdk.ListEnterpriseManagerBridgesRequest) (opsisdk.ListEnterpriseManagerBridgesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return opsisdk.ListEnterpriseManagerBridgesResponse{}, f.listErr
	}
	if len(f.listResponses) == 0 {
		return opsisdk.ListEnterpriseManagerBridgesResponse{}, nil
	}
	index := len(f.listRequests) - 1
	if index >= len(f.listResponses) {
		index = len(f.listResponses) - 1
	}
	return f.listResponses[index], nil
}

func (f *fakeEnterpriseManagerBridgeOCIClient) UpdateEnterpriseManagerBridge(_ context.Context, request opsisdk.UpdateEnterpriseManagerBridgeRequest) (opsisdk.UpdateEnterpriseManagerBridgeResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return opsisdk.UpdateEnterpriseManagerBridgeResponse{}, f.updateErr
	}
	if f.updateResponse.OpcWorkRequestId != nil {
		return f.updateResponse, nil
	}
	return opsisdk.UpdateEnterpriseManagerBridgeResponse{OpcWorkRequestId: common.String("wr-update"), OpcRequestId: common.String("opc-update")}, nil
}

func (f *fakeEnterpriseManagerBridgeOCIClient) DeleteEnterpriseManagerBridge(_ context.Context, request opsisdk.DeleteEnterpriseManagerBridgeRequest) (opsisdk.DeleteEnterpriseManagerBridgeResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return opsisdk.DeleteEnterpriseManagerBridgeResponse{}, f.deleteErr
	}
	if f.deleteResponse.OpcWorkRequestId != nil {
		return f.deleteResponse, nil
	}
	return opsisdk.DeleteEnterpriseManagerBridgeResponse{OpcWorkRequestId: common.String("wr-delete"), OpcRequestId: common.String("opc-delete")}, nil
}

func (f *fakeEnterpriseManagerBridgeOCIClient) GetWorkRequest(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	workRequestID := stringValue(request.WorkRequestId)
	if err := f.workRequestErrs[workRequestID]; err != nil {
		return opsisdk.GetWorkRequestResponse{}, err
	}
	if workRequest, ok := f.workRequests[workRequestID]; ok {
		return opsisdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
	}
	return opsisdk.GetWorkRequestResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "work request not found")
}

func TestEnterpriseManagerBridgeCreateStartsWorkRequest(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		createResponse: opsisdk.CreateEnterpriseManagerBridgeResponse{
			EnterpriseManagerBridge: makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateCreating),
			OpcWorkRequestId:        common.String("wr-create"),
			OpcRequestId:            common.String("opc-create"),
		},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-create": makeEnterpriseManagerBridgeWorkRequest("wr-create", opsisdk.OperationTypeCreateEnterpriseManagerBridge, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, testEnterpriseManagerBridgeID),
		},
	}

	response, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while create work request is pending")
	}
	requireEnterpriseManagerBridgeCreateRequest(t, fakeClient.createRequests)
	requireEnterpriseManagerBridgeOpcRequestID(t, resource, "opc-create")
	requireEnterpriseManagerBridgeOCID(t, resource, testEnterpriseManagerBridgeID)
	requireEnterpriseManagerBridgeAsync(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create")
}

func TestEnterpriseManagerBridgeCompletesCreateWorkRequestWithReadback(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{
			testEnterpriseManagerBridgeID: makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive),
		},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-create": makeEnterpriseManagerBridgeWorkRequest("wr-create", opsisdk.OperationTypeCreateEnterpriseManagerBridge, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeCreated, testEnterpriseManagerBridgeID),
		},
	}

	response, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false after successful readback")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status async current = %#v, want nil after successful readback", resource.Status.OsokStatus.Async.Current)
	}
	if got := resource.Status.Id; got != testEnterpriseManagerBridgeID {
		t.Fatalf("status id = %q, want %q", got, testEnterpriseManagerBridgeID)
	}
	if got := lastConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestEnterpriseManagerBridgeBindUsesPaginatedListAndReadback(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	next := "page-2"
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{
			testEnterpriseManagerBridgeID: makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive),
		},
		listResponses: []opsisdk.ListEnterpriseManagerBridgesResponse{
			{
				EnterpriseManagerBridgeCollection: opsisdk.EnterpriseManagerBridgeCollection{
					Items: []opsisdk.EnterpriseManagerBridgeSummary{
						makeEnterpriseManagerBridgeSummary("ocid1.enterprisemanagerbridge.oc1..other", "other", testEnterpriseManagerBridgeBucket, opsisdk.LifecycleStateActive),
					},
				},
				OpcNextPage: common.String(next),
			},
			{
				EnterpriseManagerBridgeCollection: opsisdk.EnterpriseManagerBridgeCollection{
					Items: []opsisdk.EnterpriseManagerBridgeSummary{
						makeEnterpriseManagerBridgeSummary(testEnterpriseManagerBridgeID, testEnterpriseManagerBridgeName, testEnterpriseManagerBridgeBucket, opsisdk.LifecycleStateActive),
					},
				},
			},
		},
	}

	_, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(fakeClient.createRequests) != 0 {
		t.Fatalf("create calls = %d, want 0 for bind", len(fakeClient.createRequests))
	}
	if len(fakeClient.listRequests) != 2 {
		t.Fatalf("list calls = %d, want 2 pages", len(fakeClient.listRequests))
	}
	if got := stringValue(fakeClient.listRequests[1].Page); got != next {
		t.Fatalf("second list page = %q, want %q", got, next)
	}
	if len(fakeClient.getRequests) == 0 {
		t.Fatal("get calls = 0, want full readback after list bind")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testEnterpriseManagerBridgeID {
		t.Fatalf("status ocid = %q, want %q", got, testEnterpriseManagerBridgeID)
	}
}

func TestEnterpriseManagerBridgeNoOpReconcileSkipsUpdate(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testEnterpriseManagerBridgeID)
	current := makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive)
	current.Description = common.String(resource.Spec.Description)
	current.FreeformTags = map[string]string{"env": "prod"}
	current.DefinedTags = map[string]map[string]interface{}{"ns": {"key": "value"}}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"key": "value"}}
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{testEnterpriseManagerBridgeID: current},
	}

	_, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(fakeClient.updateRequests) != 0 {
		t.Fatalf("update calls = %d, want 0 when OCI state matches spec", len(fakeClient.updateRequests))
	}
	if got := lastConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestEnterpriseManagerBridgeOmittedDescriptionDoesNotClearReadback(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testEnterpriseManagerBridgeID)
	resource.Spec.Description = ""
	current := makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive)
	current.Description = common.String("external description")
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{testEnterpriseManagerBridgeID: current},
	}

	_, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(fakeClient.updateRequests) != 0 {
		t.Fatalf("update calls = %d, want 0 when spec.description is omitted", len(fakeClient.updateRequests))
	}
	if got := resource.Status.Description; got != "external description" {
		t.Fatalf("status description = %q, want observed description", got)
	}
	if got := lastConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestEnterpriseManagerBridgeMutableUpdateStartsWorkRequest(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testEnterpriseManagerBridgeID)
	resource.Spec.DisplayName = "renamed"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"key": "new"}}
	current := makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive)
	current.Description = common.String("old description")
	current.FreeformTags = map[string]string{"env": "dev"}
	current.DefinedTags = map[string]map[string]interface{}{"ns": {"key": "old"}}
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{testEnterpriseManagerBridgeID: current},
		updateResponse: opsisdk.UpdateEnterpriseManagerBridgeResponse{
			OpcWorkRequestId: common.String("wr-update"),
			OpcRequestId:     common.String("opc-update"),
		},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-update": makeEnterpriseManagerBridgeWorkRequest("wr-update", opsisdk.OperationTypeEnum("UPDATE_ENTERPRISE_MANAGER_BRIDGE"), opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, testEnterpriseManagerBridgeID),
		},
	}

	response, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true while update work request is pending")
	}
	if len(fakeClient.updateRequests) != 1 {
		t.Fatalf("update calls = %d, want 1", len(fakeClient.updateRequests))
	}
	update := fakeClient.updateRequests[0]
	if got := stringValue(update.EnterpriseManagerBridgeId); got != testEnterpriseManagerBridgeID {
		t.Fatalf("update EnterpriseManagerBridgeId = %q, want %q", got, testEnterpriseManagerBridgeID)
	}
	if got := stringValue(update.DisplayName); got != "renamed" {
		t.Fatalf("update displayName = %q, want renamed", got)
	}
	if got := update.FreeformTags["env"]; got != "prod" {
		t.Fatalf("update freeformTags[env] = %q, want prod", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status opcRequestId = %q, want opc-update", got)
	}
}

func TestEnterpriseManagerBridgeRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testEnterpriseManagerBridgeID)
	resource.Spec.ObjectStorageBucketName = "different-bucket"
	current := makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive)
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{testEnterpriseManagerBridgeID: current},
	}

	_, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "objectStorageBucketName") {
		t.Fatalf("CreateOrUpdate() error = %q, want objectStorageBucketName drift", err.Error())
	}
	if len(fakeClient.updateRequests) != 0 {
		t.Fatalf("update calls = %d, want 0 after create-only drift rejection", len(fakeClient.updateRequests))
	}
}

func TestEnterpriseManagerBridgeDeleteRetainsFinalizerUntilWorkRequestConfirmed(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testEnterpriseManagerBridgeID)
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{
			testEnterpriseManagerBridgeID: makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive),
		},
		deleteResponse: opsisdk.DeleteEnterpriseManagerBridgeResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("opc-delete"),
		},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-delete": makeEnterpriseManagerBridgeWorkRequest("wr-delete", opsisdk.OperationTypeDeleteEnterpriseManagerBridge, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, testEnterpriseManagerBridgeID),
		},
	}
	client := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first pass error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first pass deleted = true, want false while work request is pending")
	}
	if len(fakeClient.deleteRequests) != 1 {
		t.Fatalf("delete calls after first pass = %d, want 1", len(fakeClient.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status deletedAt is set while delete work request is pending")
	}

	fakeClient.workRequests["wr-delete"] = makeEnterpriseManagerBridgeWorkRequest("wr-delete", opsisdk.OperationTypeDeleteEnterpriseManagerBridge, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testEnterpriseManagerBridgeID)
	fakeClient.getErrs = map[string]error{
		testEnterpriseManagerBridgeID: errortest.NewServiceError(404, errorutil.NotFound, "deleted"),
	}
	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() confirmation pass error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() confirmation pass deleted = false, want true after unambiguous not found")
	}
	if len(fakeClient.deleteRequests) != 1 {
		t.Fatalf("delete calls after confirmation = %d, want original delete only", len(fakeClient.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status deletedAt is nil after confirmed delete")
	}
}

func TestEnterpriseManagerBridgeDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testEnterpriseManagerBridgeID)
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getResponses: map[string]opsisdk.EnterpriseManagerBridge{
			testEnterpriseManagerBridgeID: makeEnterpriseManagerBridge(testEnterpriseManagerBridgeID, common.String(testEnterpriseManagerBridgeName), common.String(testEnterpriseManagerBridgeBucket), opsisdk.LifecycleStateActive),
		},
		deleteErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
	}

	deleted, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 message", err.Error())
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status deletedAt is set after ambiguous delete")
	}
}

func TestEnterpriseManagerBridgeDeleteRejectsAuthShapedReadbackAfterSucceededWorkRequest(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testEnterpriseManagerBridgeID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		getErrs: map[string]error{
			testEnterpriseManagerBridgeID: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-delete": makeEnterpriseManagerBridgeWorkRequest("wr-delete", opsisdk.OperationTypeDeleteEnterpriseManagerBridge, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testEnterpriseManagerBridgeID),
		},
	}

	deleted, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm-read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 confirmation context", err.Error())
	}
	if len(fakeClient.workRequestRequests) != 1 {
		t.Fatalf("GetWorkRequest calls = %d, want 1 before delete confirmation", len(fakeClient.workRequestRequests))
	}
	if len(fakeClient.deleteRequests) != 0 {
		t.Fatalf("delete calls = %d, want 0 after completed delete work request", len(fakeClient.deleteRequests))
	}
	requireEnterpriseManagerBridgeOpcRequestID(t, resource, "opc-request-id")
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status deletedAt is set after ambiguous completed delete work request")
	}
}

func TestEnterpriseManagerBridgeDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	resource := newEnterpriseManagerBridgeResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fakeClient := &fakeEnterpriseManagerBridgeOCIClient{
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-create": makeEnterpriseManagerBridgeWorkRequest("wr-create", opsisdk.OperationTypeCreateEnterpriseManagerBridge, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, testEnterpriseManagerBridgeID),
		},
	}

	deleted, err := newEnterpriseManagerBridgeServiceClientWithOCIClient(testLogger(), fakeClient).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while create work request is pending")
	}
	if len(fakeClient.deleteRequests) != 0 {
		t.Fatalf("delete calls = %d, want 0 while create work request is pending", len(fakeClient.deleteRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseCreate || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status async current = %#v, want pending create work request", current)
	}
}

func newEnterpriseManagerBridgeResource() *opsiv1beta1.EnterpriseManagerBridge {
	return &opsiv1beta1.EnterpriseManagerBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "enterprise-manager-bridge",
			Namespace: "default",
			UID:       "uid-enterprise-manager-bridge",
		},
		Spec: opsiv1beta1.EnterpriseManagerBridgeSpec{
			CompartmentId:           testEnterpriseManagerBridgeCompartment,
			DisplayName:             testEnterpriseManagerBridgeName,
			ObjectStorageBucketName: testEnterpriseManagerBridgeBucket,
			Description:             "bridge description",
		},
	}
}

func makeEnterpriseManagerBridge(id string, displayName *string, bucket *string, lifecycle opsisdk.LifecycleStateEnum) opsisdk.EnterpriseManagerBridge {
	return opsisdk.EnterpriseManagerBridge{
		Id:                         common.String(id),
		CompartmentId:              common.String(testEnterpriseManagerBridgeCompartment),
		DisplayName:                displayName,
		ObjectStorageNamespaceName: common.String("object-storage-namespace"),
		ObjectStorageBucketName:    bucket,
		FreeformTags:               map[string]string{},
		DefinedTags:                map[string]map[string]interface{}{},
		LifecycleState:             lifecycle,
		Description:                common.String("bridge description"),
	}
}

func makeEnterpriseManagerBridgeSummary(id string, displayName string, bucket string, lifecycle opsisdk.LifecycleStateEnum) opsisdk.EnterpriseManagerBridgeSummary {
	return opsisdk.EnterpriseManagerBridgeSummary{
		Id:                         common.String(id),
		CompartmentId:              common.String(testEnterpriseManagerBridgeCompartment),
		DisplayName:                common.String(displayName),
		ObjectStorageNamespaceName: common.String("object-storage-namespace"),
		ObjectStorageBucketName:    common.String(bucket),
		FreeformTags:               map[string]string{},
		DefinedTags:                map[string]map[string]interface{}{},
		LifecycleState:             lifecycle,
	}
}

func makeEnterpriseManagerBridgeWorkRequest(
	id string,
	operationType opsisdk.OperationTypeEnum,
	status opsisdk.OperationStatusEnum,
	action opsisdk.ActionTypeEnum,
	resourceID string,
) opsisdk.WorkRequest {
	return opsisdk.WorkRequest{
		OperationType: operationType,
		Status:        status,
		Id:            common.String(id),
		CompartmentId: common.String(testEnterpriseManagerBridgeCompartment),
		Resources: []opsisdk.WorkRequestResource{
			{
				EntityType: common.String("EnterpriseManagerBridge"),
				ActionType: action,
				Identifier: common.String(resourceID),
				EntityUri:  common.String("/enterpriseManagerBridges/" + resourceID),
			},
		},
		PercentComplete: common.Float32(50),
	}
}

func lastConditionType(resource *opsiv1beta1.EnterpriseManagerBridge) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}

func requireEnterpriseManagerBridgeCreateRequest(t *testing.T, requests []opsisdk.CreateEnterpriseManagerBridgeRequest) {
	t.Helper()
	if len(requests) != 1 {
		t.Fatalf("create calls = %d, want 1", len(requests))
	}
	createRequest := requests[0]
	if got := stringValue(createRequest.CompartmentId); got != testEnterpriseManagerBridgeCompartment {
		t.Fatalf("create CompartmentId = %q, want %q", got, testEnterpriseManagerBridgeCompartment)
	}
	if createRequest.OpcRetryToken == nil || *createRequest.OpcRetryToken == "" {
		t.Fatal("create OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireEnterpriseManagerBridgeOpcRequestID(t *testing.T, resource *opsiv1beta1.EnterpriseManagerBridge, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status opcRequestId = %q, want %q", got, want)
	}
}

func requireEnterpriseManagerBridgeOCID(t *testing.T, resource *opsiv1beta1.EnterpriseManagerBridge, want string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status ocid = %q, want %q", got, want)
	}
}

func requireEnterpriseManagerBridgeAsync(
	t *testing.T,
	resource *opsiv1beta1.EnterpriseManagerBridge,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil ||
		current.Phase != phase ||
		current.NormalizedClass != class ||
		current.WorkRequestID != workRequestID {
		t.Fatalf("status async current = %#v, want %s %s work request %s", current, phase, class, workRequestID)
	}
}

func testLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}
