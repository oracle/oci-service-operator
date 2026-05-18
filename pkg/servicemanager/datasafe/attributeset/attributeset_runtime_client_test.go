/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package attributeset

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestAttributeSetCreateStartsWorkRequestAndRecordsIdentity(t *testing.T) {
	resource := newTestAttributeSet()
	client := newCreateAttributeSetClient(t, resource)

	response, err := newAttributeSetServiceClientForTest(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireCreateOrUpdateRequeue(t, response)
	requireAttributeSetCreateStarted(t, client, resource)
}

func newCreateAttributeSetClient(t *testing.T, resource *datasafev1beta1.AttributeSet) *fakeAttributeSetOCIClient {
	t.Helper()
	return &fakeAttributeSetOCIClient{
		listAttributeSets: func(_ context.Context, _ datasafesdk.ListAttributeSetsRequest) (datasafesdk.ListAttributeSetsResponse, error) {
			return datasafesdk.ListAttributeSetsResponse{}, nil
		},
		createAttributeSet: func(_ context.Context, _ datasafesdk.CreateAttributeSetRequest) (datasafesdk.CreateAttributeSetResponse, error) {
			return datasafesdk.CreateAttributeSetResponse{
				AttributeSet: datasafesdk.AttributeSet{
					Id:                 common.String("ocid1.attributeset.oc1..created"),
					CompartmentId:      common.String(resource.Spec.CompartmentId),
					DisplayName:        common.String(resource.Spec.DisplayName),
					AttributeSetType:   datasafesdk.AttributeSetAttributeSetTypeEnum(resource.Spec.AttributeSetType),
					AttributeSetValues: []string{"10.0.0.0/24"},
					LifecycleState:     datasafesdk.AttributeSetLifecycleStateCreating,
				},
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		getWorkRequest: requireAttributeSetWorkRequest(t, "wr-create", datasafesdk.WorkRequestOperationTypeCreateAttributeSet),
	}
}

func requireAttributeSetCreateStarted(t *testing.T, client *fakeAttributeSetOCIClient, resource *datasafev1beta1.AttributeSet) {
	t.Helper()
	requireRequestCount(t, "CreateAttributeSet", len(client.createRequests), 1)
	createRequest := client.createRequests[0]
	if got, want := stringValue(createRequest.DisplayName), resource.Spec.DisplayName; got != want {
		t.Errorf("create displayName = %q, want %q", got, want)
	}
	if got, want := createRequest.AttributeSetValues, []string{"10.0.0.0/24"}; !equalStringSlices(got, want) {
		t.Errorf("create attributeSetValues = %v, want %v", got, want)
	}
	if createRequest.OpcRetryToken == nil {
		t.Error("create opcRetryToken = nil, want deterministic retry token")
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.attributeset.oc1..created"; got != want {
		t.Errorf("status.status.ocid = %q, want %q", got, want)
	}
	requireAttributeSetAsync(t, &resource.Status.OsokStatus, "wr-create", shared.OSOKAsyncPhaseCreate)
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create"; got != want {
		t.Errorf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestAttributeSetBindUsesPaginatedUserDefinedList(t *testing.T) {
	resource := newTestAttributeSet()
	client := newPaginatedBindAttributeSetClient(t, resource)

	response, err := newAttributeSetServiceClientForTest(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireCreateOrUpdateNoRequeue(t, response)
	requireRequestCount(t, "ListAttributeSets", len(client.listRequests), 2)
	if got, want := resource.Status.Id, "ocid1.attributeset.oc1..bound"; got != want {
		t.Errorf("status.id = %q, want %q", got, want)
	}
	if got, want := resource.Status.AttributeSetValues, []string{"10.0.0.0/24"}; !equalStringSlices(got, want) {
		t.Errorf("status.attributeSetValues = %v, want %v", got, want)
	}
	requireRequestCount(t, "CreateAttributeSet", len(client.createRequests), 0)
}

func newPaginatedBindAttributeSetClient(t *testing.T, resource *datasafev1beta1.AttributeSet) *fakeAttributeSetOCIClient {
	t.Helper()
	return &fakeAttributeSetOCIClient{
		listAttributeSets: listPaginatedUserDefinedAttributeSets(t, resource),
		getAttributeSet:   getBoundAttributeSet(t, resource),
		createAttributeSet: func(context.Context, datasafesdk.CreateAttributeSetRequest) (datasafesdk.CreateAttributeSetResponse, error) {
			return datasafesdk.CreateAttributeSetResponse{}, fmt.Errorf("CreateAttributeSet should not be called for bind")
		},
	}
}

func listPaginatedUserDefinedAttributeSets(
	t *testing.T,
	resource *datasafev1beta1.AttributeSet,
) func(context.Context, datasafesdk.ListAttributeSetsRequest) (datasafesdk.ListAttributeSetsResponse, error) {
	t.Helper()
	return func(_ context.Context, request datasafesdk.ListAttributeSetsRequest) (datasafesdk.ListAttributeSetsResponse, error) {
		if request.IsUserDefined == nil || !*request.IsUserDefined {
			t.Fatalf("ListAttributeSets isUserDefined = %v, want true", request.IsUserDefined)
		}
		if request.Page == nil {
			return datasafesdk.ListAttributeSetsResponse{OpcNextPage: common.String("page-2")}, nil
		}
		if got, want := stringValue(request.Page), "page-2"; got != want {
			t.Fatalf("ListAttributeSets page = %q, want %q", got, want)
		}
		return datasafesdk.ListAttributeSetsResponse{
			AttributeSetCollection: datasafesdk.AttributeSetCollection{
				Items: []datasafesdk.AttributeSetSummary{boundAttributeSetSummary(resource)},
			},
		}, nil
	}
}

func boundAttributeSetSummary(resource *datasafev1beta1.AttributeSet) datasafesdk.AttributeSetSummary {
	return datasafesdk.AttributeSetSummary{
		Id:               common.String("ocid1.attributeset.oc1..bound"),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      common.String(resource.Spec.DisplayName),
		AttributeSetType: datasafesdk.AttributeSetAttributeSetTypeEnum(resource.Spec.AttributeSetType),
		LifecycleState:   datasafesdk.AttributeSetLifecycleStateActive,
		IsUserDefined:    common.Bool(true),
	}
}

func getBoundAttributeSet(
	t *testing.T,
	resource *datasafev1beta1.AttributeSet,
) func(context.Context, datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
	t.Helper()
	return func(_ context.Context, request datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
		if got, want := stringValue(request.AttributeSetId), "ocid1.attributeset.oc1..bound"; got != want {
			t.Fatalf("GetAttributeSet attributeSetId = %q, want %q", got, want)
		}
		return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..bound", resource.Spec)}, nil
	}
}

func TestAttributeSetMutableUpdateStartsWorkRequest(t *testing.T) {
	resource := newExistingTestAttributeSet()
	resource.Spec.DisplayName = "changed"
	resource.Spec.AttributeSetValues = []string{"10.0.1.0/24"}

	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, request datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			if got, want := stringValue(request.AttributeSetId), "ocid1.attributeset.oc1..existing"; got != want {
				t.Fatalf("GetAttributeSet attributeSetId = %q, want %q", got, want)
			}
			currentSpec := resource.Spec
			currentSpec.DisplayName = "old"
			currentSpec.AttributeSetValues = []string{"10.0.0.0/24"}
			return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..existing", currentSpec)}, nil
		},
		updateAttributeSet: func(_ context.Context, _ datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error) {
			return datasafesdk.UpdateAttributeSetResponse{OpcWorkRequestId: common.String("wr-update"), OpcRequestId: common.String("opc-update")}, nil
		},
		getWorkRequest: func(_ context.Context, _ datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return attributeSetWorkRequestResponse("wr-update", datasafesdk.WorkRequestOperationTypeUpdateAttributeSet, datasafesdk.WorkRequestStatusInProgress), nil
		},
	}

	response, err := newAttributeSetServiceClientForTest(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireCreateOrUpdateRequeue(t, response)
	requireRequestCount(t, "UpdateAttributeSet", len(client.updateRequests), 1)
	updateRequest := client.updateRequests[0]
	if got, want := stringValue(updateRequest.DisplayName), "changed"; got != want {
		t.Errorf("update displayName = %q, want %q", got, want)
	}
	if got, want := updateRequest.AttributeSetValues, []string{"10.0.1.0/24"}; !equalStringSlices(got, want) {
		t.Errorf("update attributeSetValues = %v, want %v", got, want)
	}
	if got, want := resource.Status.OsokStatus.Async.Current.WorkRequestID, "wr-update"; got != want {
		t.Errorf("status async workRequestId = %q, want %q", got, want)
	}
}

func TestAttributeSetOmittedDescriptionDoesNotClearCurrentDescription(t *testing.T) {
	resource := newExistingTestAttributeSet()
	resource.Spec.Description = ""
	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, _ datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			currentSpec := resource.Spec
			currentSpec.Description = "previous description"
			return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..existing", currentSpec)}, nil
		},
		updateAttributeSet: func(context.Context, datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error) {
			return datasafesdk.UpdateAttributeSetResponse{}, fmt.Errorf("UpdateAttributeSet should not be called when omitted description is the only diff")
		},
	}

	response, err := newAttributeSetServiceClientForTest(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireCreateOrUpdateNoRequeue(t, response)
	requireRequestCount(t, "UpdateAttributeSet", len(client.updateRequests), 0)
	if got, want := resource.Status.Description, "previous description"; got != want {
		t.Errorf("status.description = %q, want observed description %q", got, want)
	}
}

func TestAttributeSetNoOpReconcileOnlyObservesCurrentState(t *testing.T) {
	resource := newExistingTestAttributeSet()

	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, request datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			if got, want := stringValue(request.AttributeSetId), "ocid1.attributeset.oc1..existing"; got != want {
				t.Fatalf("GetAttributeSet attributeSetId = %q, want %q", got, want)
			}
			return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..existing", resource.Spec)}, nil
		},
		createAttributeSet: func(context.Context, datasafesdk.CreateAttributeSetRequest) (datasafesdk.CreateAttributeSetResponse, error) {
			return datasafesdk.CreateAttributeSetResponse{}, fmt.Errorf("CreateAttributeSet should not be called for no-op reconcile")
		},
		updateAttributeSet: func(context.Context, datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error) {
			return datasafesdk.UpdateAttributeSetResponse{}, fmt.Errorf("UpdateAttributeSet should not be called for no-op reconcile")
		},
	}

	response, err := newAttributeSetServiceClientForTest(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no requeue", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateAttributeSet calls = %d, want 0", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateAttributeSet calls = %d, want 0", len(client.updateRequests))
	}
}

func TestAttributeSetRejectsCreateOnlyAttributeSetTypeDrift(t *testing.T) {
	resource := newExistingTestAttributeSet()
	resource.Spec.AttributeSetType = string(datasafesdk.AttributeSetAttributeSetTypeOsUser)

	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, _ datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			currentSpec := resource.Spec
			currentSpec.AttributeSetType = string(datasafesdk.AttributeSetAttributeSetTypeIpAddress)
			return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..existing", currentSpec)}, nil
		},
		updateAttributeSet: func(context.Context, datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error) {
			return datasafesdk.UpdateAttributeSetResponse{}, fmt.Errorf("UpdateAttributeSet should not be called for create-only drift")
		},
	}

	_, err := newAttributeSetServiceClientForTest(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "attributeSetType") {
		t.Fatalf("CreateOrUpdate() error = %v, want attributeSetType drift detail", err)
	}
	requireRequestCount(t, "UpdateAttributeSet", len(client.updateRequests), 0)
}

func TestAttributeSetCompartmentMoveUsesChangeCompartmentWorkRequest(t *testing.T) {
	resource := newExistingTestAttributeSet()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, _ datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			currentSpec := resource.Spec
			currentSpec.CompartmentId = "ocid1.compartment.oc1..old"
			return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..existing", currentSpec)}, nil
		},
		changeAttributeSetCompartment: func(_ context.Context, _ datasafesdk.ChangeAttributeSetCompartmentRequest) (datasafesdk.ChangeAttributeSetCompartmentResponse, error) {
			return datasafesdk.ChangeAttributeSetCompartmentResponse{OpcWorkRequestId: common.String("wr-move"), OpcRequestId: common.String("opc-move")}, nil
		},
	}

	response, err := newAttributeSetServiceClientForTest(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireCreateOrUpdateRequeue(t, response)
	requireRequestCount(t, "ChangeAttributeSetCompartment", len(client.changeCompartmentRequests), 1)
	moveRequest := client.changeCompartmentRequests[0]
	if got, want := stringValue(moveRequest.AttributeSetId), "ocid1.attributeset.oc1..existing"; got != want {
		t.Errorf("move attributeSetId = %q, want %q", got, want)
	}
	if got, want := stringValue(moveRequest.CompartmentId), "ocid1.compartment.oc1..new"; got != want {
		t.Errorf("move compartmentId = %q, want %q", got, want)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.WorkRequestID != "wr-move" || current.RawOperationType != string(datasafesdk.WorkRequestOperationTypeChangeAttributeSetCompartment) {
		t.Fatalf("status async current = %+v, want move work request", current)
	}
}

func TestAttributeSetDeleteWaitsForPendingCreateOrUpdateWorkRequest(t *testing.T) {
	cases := []struct {
		name          string
		phase         shared.OSOKAsyncPhase
		workRequestID string
		operation     datasafesdk.WorkRequestOperationTypeEnum
	}{
		{
			name:          "create work request",
			phase:         shared.OSOKAsyncPhaseCreate,
			workRequestID: "wr-create",
			operation:     datasafesdk.WorkRequestOperationTypeCreateAttributeSet,
		},
		{
			name:          "update work request",
			phase:         shared.OSOKAsyncPhaseUpdate,
			workRequestID: "wr-update",
			operation:     datasafesdk.WorkRequestOperationTypeUpdateAttributeSet,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resource := newExistingTestAttributeSet()
			resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceWorkRequest,
				Phase:           tt.phase,
				WorkRequestID:   tt.workRequestID,
				NormalizedClass: shared.OSOKAsyncClassPending,
			}

			client := &fakeAttributeSetOCIClient{
				getAttributeSet: func(context.Context, datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
					t.Fatal("GetAttributeSet should not be called while write work request is still pending")
					return datasafesdk.GetAttributeSetResponse{}, nil
				},
				deleteAttributeSet: func(context.Context, datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error) {
					t.Fatal("DeleteAttributeSet should not be called while write work request is still pending")
					return datasafesdk.DeleteAttributeSetResponse{}, nil
				},
				getWorkRequest: func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
					if got := stringValue(request.WorkRequestId); got != tt.workRequestID {
						t.Fatalf("GetWorkRequest workRequestId = %q, want %q", got, tt.workRequestID)
					}
					return attributeSetWorkRequestResponse(tt.workRequestID, tt.operation, datasafesdk.WorkRequestStatusInProgress), nil
				},
			}

			deleted, err := newAttributeSetServiceClientForTest(client).Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want finalizer retained while write work request is pending")
			}
			requireRequestCount(t, "GetWorkRequest", len(client.getWorkRequestRequests), 1)
			requireRequestCount(t, "GetAttributeSet", len(client.getRequests), 0)
			requireRequestCount(t, "DeleteAttributeSet", len(client.deleteRequests), 0)
			requireAttributeSetAsync(t, &resource.Status.OsokStatus, tt.workRequestID, tt.phase)
		})
	}
}

func TestAttributeSetDeleteStartsAfterWriteWorkRequestSucceeds(t *testing.T) {
	resource := newExistingTestAttributeSet()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, _ datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..existing", resource.Spec)}, nil
		},
		deleteAttributeSet: func(_ context.Context, _ datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error) {
			return datasafesdk.DeleteAttributeSetResponse{OpcWorkRequestId: common.String("wr-delete"), OpcRequestId: common.String("opc-delete")}, nil
		},
		getWorkRequest: func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			switch got := stringValue(request.WorkRequestId); got {
			case "wr-create":
				return attributeSetWorkRequestResponse(got, datasafesdk.WorkRequestOperationTypeCreateAttributeSet, datasafesdk.WorkRequestStatusSucceeded), nil
			case "wr-delete":
				return attributeSetWorkRequestResponse(got, datasafesdk.WorkRequestOperationTypeDeleteAttributeSet, datasafesdk.WorkRequestStatusInProgress), nil
			default:
				t.Fatalf("GetWorkRequest workRequestId = %q, want wr-create or wr-delete", got)
				return datasafesdk.GetWorkRequestResponse{}, nil
			}
		},
	}

	deleted, err := newAttributeSetServiceClientForTest(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while delete work request is pending")
	}
	requireRequestCount(t, "GetWorkRequest", len(client.getWorkRequestRequests), 2)
	requireRequestCount(t, "GetAttributeSet", len(client.getRequests), 1)
	requireRequestCount(t, "DeleteAttributeSet", len(client.deleteRequests), 1)
	requireAttributeSetAsync(t, &resource.Status.OsokStatus, "wr-delete", shared.OSOKAsyncPhaseDelete)
}

func TestAttributeSetDeleteWaitsForPendingLifecycleReadback(t *testing.T) {
	cases := []struct {
		name  string
		state datasafesdk.AttributeSetLifecycleStateEnum
		phase shared.OSOKAsyncPhase
	}{
		{
			name:  "creating",
			state: datasafesdk.AttributeSetLifecycleStateCreating,
			phase: shared.OSOKAsyncPhaseCreate,
		},
		{
			name:  "updating",
			state: datasafesdk.AttributeSetLifecycleStateUpdating,
			phase: shared.OSOKAsyncPhaseUpdate,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resource := newExistingTestAttributeSet()
			current := activeAttributeSet("ocid1.attributeset.oc1..existing", resource.Spec)
			current.LifecycleState = tt.state
			client := &fakeAttributeSetOCIClient{
				getAttributeSet: func(_ context.Context, _ datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
					return datasafesdk.GetAttributeSetResponse{AttributeSet: current}, nil
				},
				deleteAttributeSet: func(context.Context, datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error) {
					t.Fatal("DeleteAttributeSet should not be called while readback lifecycle is pending")
					return datasafesdk.DeleteAttributeSetResponse{}, nil
				},
			}

			deleted, err := newAttributeSetServiceClientForTest(client).Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want finalizer retained while lifecycle is pending")
			}
			requireRequestCount(t, "GetAttributeSet", len(client.getRequests), 1)
			requireRequestCount(t, "DeleteAttributeSet", len(client.deleteRequests), 0)
			if got, want := resource.Status.LifecycleState, string(tt.state); got != want {
				t.Fatalf("status.lifecycleState = %q, want %q", got, want)
			}
			requireAttributeSetLifecycleAsync(t, &resource.Status.OsokStatus, tt.phase)
		})
	}
}

func TestAttributeSetDeleteRetainsFinalizerForPendingWorkRequestAndAmbiguousReadback(t *testing.T) {
	resource := newExistingTestAttributeSet()
	resource.Status.OsokStatus.Ocid = "ocid1.attributeset.oc1..delete"
	resource.Status.Id = "ocid1.attributeset.oc1..delete"

	getCalls := 0
	workRequestStatuses := []datasafesdk.WorkRequestStatusEnum{
		datasafesdk.WorkRequestStatusInProgress,
		datasafesdk.WorkRequestStatusSucceeded,
	}
	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, _ datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..delete", resource.Spec)}, nil
			}
			return datasafesdk.GetAttributeSetResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteAttributeSet: func(_ context.Context, _ datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error) {
			return datasafesdk.DeleteAttributeSetResponse{OpcWorkRequestId: common.String("wr-delete"), OpcRequestId: common.String("opc-delete")}, nil
		},
		getWorkRequest: func(_ context.Context, _ datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			if len(workRequestStatuses) == 0 {
				return attributeSetWorkRequestResponse("wr-delete", datasafesdk.WorkRequestOperationTypeDeleteAttributeSet, datasafesdk.WorkRequestStatusInProgress), nil
			}
			status := workRequestStatuses[0]
			workRequestStatuses = workRequestStatuses[1:]
			return attributeSetWorkRequestResponse("wr-delete", datasafesdk.WorkRequestOperationTypeDeleteAttributeSet, status), nil
		},
	}
	serviceClient := newAttributeSetServiceClientForTest(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	requireAttributeSetDeletePending(t, resource, client, deleted, err)

	deleted, err = serviceClient.Delete(context.Background(), resource)
	requireAttributeSetDeleteAmbiguous(t, resource, client, deleted, err)
}

func TestAttributeSetDeleteRejectsAmbiguousReadbackAfterInitialSucceededWorkRequest(t *testing.T) {
	resource := newExistingTestAttributeSet()
	resource.Status.OsokStatus.Ocid = "ocid1.attributeset.oc1..delete"
	resource.Status.Id = "ocid1.attributeset.oc1..delete"

	getCalls := 0
	client := &fakeAttributeSetOCIClient{
		getAttributeSet: func(_ context.Context, _ datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAttributeSetResponse{AttributeSet: activeAttributeSet("ocid1.attributeset.oc1..delete", resource.Spec)}, nil
			}
			return datasafesdk.GetAttributeSetResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteAttributeSet: func(_ context.Context, _ datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error) {
			return datasafesdk.DeleteAttributeSetResponse{OpcWorkRequestId: common.String("wr-delete"), OpcRequestId: common.String("opc-delete")}, nil
		},
		getWorkRequest: func(_ context.Context, _ datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return attributeSetWorkRequestResponse("wr-delete", datasafesdk.WorkRequestOperationTypeDeleteAttributeSet, datasafesdk.WorkRequestStatusSucceeded), nil
		},
	}

	deleted, err := newAttributeSetServiceClientForTest(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound readback rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained after ambiguous readback")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	requireRequestCount(t, "DeleteAttributeSet", len(client.deleteRequests), 1)
	requireRequestCount(t, "GetWorkRequest", len(client.getWorkRequestRequests), 1)
	requireRequestCount(t, "GetAttributeSet", len(client.getRequests), 2)
	requireAttributeSetAsync(t, &resource.Status.OsokStatus, "wr-delete", shared.OSOKAsyncPhaseDelete)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want no confirmed deletion", resource.Status.OsokStatus.DeletedAt)
	}
}

func requireAttributeSetDeletePending(
	t *testing.T,
	resource *datasafev1beta1.AttributeSet,
	client *fakeAttributeSetOCIClient,
	deleted bool,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call deleted = true, want finalizer retained while work request is pending")
	}
	requireRequestCount(t, "DeleteAttributeSet", len(client.deleteRequests), 1)
	requireAttributeSetAsync(t, &resource.Status.OsokStatus, "wr-delete", shared.OSOKAsyncPhaseDelete)
}

func requireAttributeSetDeleteAmbiguous(
	t *testing.T,
	resource *datasafev1beta1.AttributeSet,
	client *fakeAttributeSetOCIClient,
	deleted bool,
	err error,
) {
	t.Helper()
	if err == nil {
		t.Fatal("Delete() second call error = nil, want ambiguous NotAuthorizedOrNotFound readback rejection")
	}
	if deleted {
		t.Fatal("Delete() second call deleted = true, want finalizer retained on ambiguous readback")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() second call error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	requireRequestCount(t, "DeleteAttributeSet after ambiguous readback", len(client.deleteRequests), 1)
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Errorf("status.status.opcRequestId = %q, want %q from ambiguous service error", got, want)
	}
}

func newAttributeSetServiceClientForTest(client attributeSetOCIClient) AttributeSetServiceClient {
	manager := &AttributeSetServiceManager{}
	hooks := newAttributeSetDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyAttributeSetRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultAttributeSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.AttributeSet](
			buildAttributeSetGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapAttributeSetGeneratedClient(hooks, delegate)
}

func requireCreateOrUpdateRequeue(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful requeue", response)
	}
}

func requireCreateOrUpdateNoRequeue(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no requeue", response)
	}
}

func requireRequestCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func requireAttributeSetAsync(t *testing.T, status *shared.OSOKStatus, workRequestID string, phase shared.OSOKAsyncPhase) {
	t.Helper()
	current := status.Async.Current
	if current == nil || current.WorkRequestID != workRequestID || current.Phase != phase {
		t.Fatalf("status async current = %+v, want %s work request %s", current, phase, workRequestID)
	}
}

func requireAttributeSetLifecycleAsync(t *testing.T, status *shared.OSOKStatus, phase shared.OSOKAsyncPhase) {
	t.Helper()
	current := status.Async.Current
	if current == nil {
		t.Fatal("status async current = nil, want lifecycle pending operation")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle || current.Phase != phase || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status async current = %+v, want lifecycle pending %s operation", current, phase)
	}
}

func requireAttributeSetWorkRequest(
	t *testing.T,
	workRequestID string,
	operation datasafesdk.WorkRequestOperationTypeEnum,
) func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
		if got := stringValue(request.WorkRequestId); got != workRequestID {
			t.Fatalf("GetWorkRequest workRequestId = %q, want %q", got, workRequestID)
		}
		return attributeSetWorkRequestResponse(workRequestID, operation, datasafesdk.WorkRequestStatusInProgress), nil
	}
}

func newExistingTestAttributeSet() *datasafev1beta1.AttributeSet {
	resource := newTestAttributeSet()
	resource.Status.OsokStatus.Ocid = "ocid1.attributeset.oc1..existing"
	resource.Status.Id = "ocid1.attributeset.oc1..existing"
	return resource
}

func newTestAttributeSet() *datasafev1beta1.AttributeSet {
	return &datasafev1beta1.AttributeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "attribute-set",
			Namespace: "default",
			UID:       types.UID("uid-attribute-set"),
		},
		Spec: datasafev1beta1.AttributeSetSpec{
			DisplayName:        "attribute-set",
			CompartmentId:      "ocid1.compartment.oc1..test",
			AttributeSetType:   string(datasafesdk.AttributeSetAttributeSetTypeIpAddress),
			AttributeSetValues: []string{"10.0.0.0/24"},
			Description:        "test attribute set",
			FreeformTags:       map[string]string{"env": "test"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func activeAttributeSet(id string, spec datasafev1beta1.AttributeSetSpec) datasafesdk.AttributeSet {
	return datasafesdk.AttributeSet{
		Id:                 common.String(id),
		CompartmentId:      common.String(spec.CompartmentId),
		DisplayName:        common.String(spec.DisplayName),
		AttributeSetType:   datasafesdk.AttributeSetAttributeSetTypeEnum(spec.AttributeSetType),
		AttributeSetValues: cloneAttributeSetValues(spec.AttributeSetValues),
		Description:        common.String(spec.Description),
		LifecycleState:     datasafesdk.AttributeSetLifecycleStateActive,
		IsUserDefined:      common.Bool(true),
		FreeformTags:       cloneAttributeSetStringMap(spec.FreeformTags),
		DefinedTags:        attributeSetDefinedTags(spec.DefinedTags),
	}
}

func attributeSetWorkRequestResponse(
	id string,
	operation datasafesdk.WorkRequestOperationTypeEnum,
	status datasafesdk.WorkRequestStatusEnum,
) datasafesdk.GetWorkRequestResponse {
	return datasafesdk.GetWorkRequestResponse{
		OpcRequestId: common.String("opc-work-request"),
		WorkRequest: datasafesdk.WorkRequest{
			Id:              common.String(id),
			OperationType:   operation,
			Status:          status,
			CompartmentId:   common.String("ocid1.compartment.oc1..test"),
			PercentComplete: common.Float32(50),
			Resources: []datasafesdk.WorkRequestResource{{
				EntityType: common.String("AttributeSet"),
				Identifier: common.String("ocid1.attributeset.oc1..created"),
				ActionType: datasafesdk.WorkRequestResourceActionTypeInProgress,
			}},
		},
	}
}

type fakeAttributeSetOCIClient struct {
	createAttributeSet            func(context.Context, datasafesdk.CreateAttributeSetRequest) (datasafesdk.CreateAttributeSetResponse, error)
	getAttributeSet               func(context.Context, datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error)
	listAttributeSets             func(context.Context, datasafesdk.ListAttributeSetsRequest) (datasafesdk.ListAttributeSetsResponse, error)
	updateAttributeSet            func(context.Context, datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error)
	deleteAttributeSet            func(context.Context, datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error)
	changeAttributeSetCompartment func(context.Context, datasafesdk.ChangeAttributeSetCompartmentRequest) (datasafesdk.ChangeAttributeSetCompartmentResponse, error)
	getWorkRequest                func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
	createRequests                []datasafesdk.CreateAttributeSetRequest
	getRequests                   []datasafesdk.GetAttributeSetRequest
	listRequests                  []datasafesdk.ListAttributeSetsRequest
	updateRequests                []datasafesdk.UpdateAttributeSetRequest
	deleteRequests                []datasafesdk.DeleteAttributeSetRequest
	changeCompartmentRequests     []datasafesdk.ChangeAttributeSetCompartmentRequest
	getWorkRequestRequests        []datasafesdk.GetWorkRequestRequest
}

func (f *fakeAttributeSetOCIClient) CreateAttributeSet(ctx context.Context, request datasafesdk.CreateAttributeSetRequest) (datasafesdk.CreateAttributeSetResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createAttributeSet == nil {
		return datasafesdk.CreateAttributeSetResponse{}, fmt.Errorf("unexpected CreateAttributeSet call")
	}
	return f.createAttributeSet(ctx, request)
}

func (f *fakeAttributeSetOCIClient) GetAttributeSet(ctx context.Context, request datasafesdk.GetAttributeSetRequest) (datasafesdk.GetAttributeSetResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getAttributeSet == nil {
		return datasafesdk.GetAttributeSetResponse{}, fmt.Errorf("unexpected GetAttributeSet call")
	}
	return f.getAttributeSet(ctx, request)
}

func (f *fakeAttributeSetOCIClient) ListAttributeSets(ctx context.Context, request datasafesdk.ListAttributeSetsRequest) (datasafesdk.ListAttributeSetsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listAttributeSets == nil {
		return datasafesdk.ListAttributeSetsResponse{}, fmt.Errorf("unexpected ListAttributeSets call")
	}
	return f.listAttributeSets(ctx, request)
}

func (f *fakeAttributeSetOCIClient) UpdateAttributeSet(ctx context.Context, request datasafesdk.UpdateAttributeSetRequest) (datasafesdk.UpdateAttributeSetResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateAttributeSet == nil {
		return datasafesdk.UpdateAttributeSetResponse{}, fmt.Errorf("unexpected UpdateAttributeSet call")
	}
	return f.updateAttributeSet(ctx, request)
}

func (f *fakeAttributeSetOCIClient) DeleteAttributeSet(ctx context.Context, request datasafesdk.DeleteAttributeSetRequest) (datasafesdk.DeleteAttributeSetResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteAttributeSet == nil {
		return datasafesdk.DeleteAttributeSetResponse{}, fmt.Errorf("unexpected DeleteAttributeSet call")
	}
	return f.deleteAttributeSet(ctx, request)
}

func (f *fakeAttributeSetOCIClient) ChangeAttributeSetCompartment(ctx context.Context, request datasafesdk.ChangeAttributeSetCompartmentRequest) (datasafesdk.ChangeAttributeSetCompartmentResponse, error) {
	f.changeCompartmentRequests = append(f.changeCompartmentRequests, request)
	if f.changeAttributeSetCompartment == nil {
		return datasafesdk.ChangeAttributeSetCompartmentResponse{}, fmt.Errorf("unexpected ChangeAttributeSetCompartment call")
	}
	return f.changeAttributeSetCompartment(ctx, request)
}

func (f *fakeAttributeSetOCIClient) GetWorkRequest(ctx context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequest == nil {
		return datasafesdk.GetWorkRequestResponse{}, fmt.Errorf("unexpected GetWorkRequest call")
	}
	return f.getWorkRequest(ctx, request)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func equalStringSlices(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

var _ attributeSetOCIClient = (*fakeAttributeSetOCIClient)(nil)
