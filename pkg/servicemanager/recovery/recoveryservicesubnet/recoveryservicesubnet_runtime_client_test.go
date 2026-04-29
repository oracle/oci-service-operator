/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package recoveryservicesubnet

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testRecoveryServiceSubnetID          = "ocid1.recoveryservicesubnet.oc1..rss"
	testRecoveryServiceSubnetCompartment = "ocid1.compartment.oc1..rss"
	testRecoveryServiceSubnetVCN         = "ocid1.vcn.oc1..rss"
	testRecoveryServiceSubnetSubnet      = "ocid1.subnet.oc1..rss"
	testRecoveryServiceSubnetName        = "rss-sample"
)

type fakeRecoveryServiceSubnetOCIClient struct {
	changeCompartmentFn func(context.Context, recoverysdk.ChangeRecoveryServiceSubnetCompartmentRequest) (recoverysdk.ChangeRecoveryServiceSubnetCompartmentResponse, error)
	createFn            func(context.Context, recoverysdk.CreateRecoveryServiceSubnetRequest) (recoverysdk.CreateRecoveryServiceSubnetResponse, error)
	getFn               func(context.Context, recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error)
	listFn              func(context.Context, recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error)
	updateFn            func(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error)
	deleteFn            func(context.Context, recoverysdk.DeleteRecoveryServiceSubnetRequest) (recoverysdk.DeleteRecoveryServiceSubnetResponse, error)
	getWorkRequestFn    func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error)
}

func (f *fakeRecoveryServiceSubnetOCIClient) ChangeRecoveryServiceSubnetCompartment(
	ctx context.Context,
	request recoverysdk.ChangeRecoveryServiceSubnetCompartmentRequest,
) (recoverysdk.ChangeRecoveryServiceSubnetCompartmentResponse, error) {
	if f.changeCompartmentFn != nil {
		return f.changeCompartmentFn(ctx, request)
	}
	return recoverysdk.ChangeRecoveryServiceSubnetCompartmentResponse{}, nil
}

func (f *fakeRecoveryServiceSubnetOCIClient) CreateRecoveryServiceSubnet(
	ctx context.Context,
	request recoverysdk.CreateRecoveryServiceSubnetRequest,
) (recoverysdk.CreateRecoveryServiceSubnetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return recoverysdk.CreateRecoveryServiceSubnetResponse{}, nil
}

func (f *fakeRecoveryServiceSubnetOCIClient) GetRecoveryServiceSubnet(
	ctx context.Context,
	request recoverysdk.GetRecoveryServiceSubnetRequest,
) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return recoverysdk.GetRecoveryServiceSubnetResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "recovery service subnet is missing")
}

func (f *fakeRecoveryServiceSubnetOCIClient) ListRecoveryServiceSubnets(
	ctx context.Context,
	request recoverysdk.ListRecoveryServiceSubnetsRequest,
) (recoverysdk.ListRecoveryServiceSubnetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return recoverysdk.ListRecoveryServiceSubnetsResponse{}, nil
}

func (f *fakeRecoveryServiceSubnetOCIClient) UpdateRecoveryServiceSubnet(
	ctx context.Context,
	request recoverysdk.UpdateRecoveryServiceSubnetRequest,
) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return recoverysdk.UpdateRecoveryServiceSubnetResponse{}, nil
}

func (f *fakeRecoveryServiceSubnetOCIClient) DeleteRecoveryServiceSubnet(
	ctx context.Context,
	request recoverysdk.DeleteRecoveryServiceSubnetRequest,
) (recoverysdk.DeleteRecoveryServiceSubnetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return recoverysdk.DeleteRecoveryServiceSubnetResponse{}, nil
}

func (f *fakeRecoveryServiceSubnetOCIClient) GetWorkRequest(
	ctx context.Context,
	request recoverysdk.GetWorkRequestRequest,
) (recoverysdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	return recoverysdk.GetWorkRequestResponse{}, nil
}

func newTestRecoveryServiceSubnetClient(client recoveryServiceSubnetOCIClient) RecoveryServiceSubnetServiceClient {
	return newRecoveryServiceSubnetServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}, client)
}

func makeRecoveryServiceSubnetResource() *recoveryv1beta1.RecoveryServiceSubnet {
	return &recoveryv1beta1.RecoveryServiceSubnet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRecoveryServiceSubnetName,
			Namespace: "default",
		},
		Spec: recoveryv1beta1.RecoveryServiceSubnetSpec{
			DisplayName:   testRecoveryServiceSubnetName,
			VcnId:         testRecoveryServiceSubnetVCN,
			CompartmentId: testRecoveryServiceSubnetCompartment,
			SubnetId:      testRecoveryServiceSubnetSubnet,
			Subnets:       []string{testRecoveryServiceSubnetSubnet},
			NsgIds:        []string{"ocid1.nsg.oc1..initial"},
			FreeformTags:  map[string]string{"env": "test"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeRecoveryServiceSubnetRequest(resource *recoveryv1beta1.RecoveryServiceSubnet) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKRecoveryServiceSubnet(
	id string,
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	state recoverysdk.LifecycleStateEnum,
) recoverysdk.RecoveryServiceSubnet {
	return recoverysdk.RecoveryServiceSubnet{
		Id:               common.String(id),
		CompartmentId:    common.String(spec.CompartmentId),
		VcnId:            common.String(spec.VcnId),
		SubnetId:         common.String(spec.SubnetId),
		DisplayName:      common.String(spec.DisplayName),
		Subnets:          append([]string(nil), spec.Subnets...),
		NsgIds:           append([]string(nil), spec.NsgIds...),
		LifecycleState:   state,
		LifecycleDetails: common.String(string(state)),
		FreeformTags:     cloneRecoveryServiceSubnetStringMap(spec.FreeformTags),
		DefinedTags:      recoveryServiceSubnetDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func makeSDKRecoveryServiceSubnetSummary(
	id string,
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	state recoverysdk.LifecycleStateEnum,
) recoverysdk.RecoveryServiceSubnetSummary {
	return recoverysdk.RecoveryServiceSubnetSummary{
		Id:               common.String(id),
		CompartmentId:    common.String(spec.CompartmentId),
		VcnId:            common.String(spec.VcnId),
		SubnetId:         common.String(spec.SubnetId),
		DisplayName:      common.String(spec.DisplayName),
		Subnets:          append([]string(nil), spec.Subnets...),
		NsgIds:           append([]string(nil), spec.NsgIds...),
		LifecycleState:   state,
		LifecycleDetails: common.String(string(state)),
		FreeformTags:     cloneRecoveryServiceSubnetStringMap(spec.FreeformTags),
		DefinedTags:      recoveryServiceSubnetDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func makeRecoveryServiceSubnetWorkRequest(
	id string,
	status recoverysdk.OperationStatusEnum,
	operationType recoverysdk.OperationTypeEnum,
	action recoverysdk.ActionTypeEnum,
) recoverysdk.WorkRequest {
	return recoverysdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operationType,
		Resources: []recoverysdk.WorkRequestResource{
			{
				EntityType: common.String(recoveryServiceSubnetWorkRequestEntityType),
				ActionType: action,
				Identifier: common.String(testRecoveryServiceSubnetID),
				EntityUri:  common.String("/recoveryServiceSubnets/" + testRecoveryServiceSubnetID),
			},
		},
	}
}

func recoveryServiceSubnetPagedVCNListFn(
	t *testing.T,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	calls *int,
) func(context.Context, recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error) {
	t.Helper()
	return func(_ context.Context, request recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error) {
		*calls = *calls + 1
		requireStringPtr(t, "ListRecoveryServiceSubnetsRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		requireStringPtr(t, "ListRecoveryServiceSubnetsRequest.VcnId", request.VcnId, resource.Spec.VcnId)
		if request.DisplayName != nil {
			t.Fatalf("ListRecoveryServiceSubnetsRequest.DisplayName = %q, want nil so VCN identity can bind before display name update", *request.DisplayName)
		}
		if *calls == 1 {
			if request.Page != nil {
				t.Fatalf("first ListRecoveryServiceSubnetsRequest.Page = %q, want nil", *request.Page)
			}
			other := resource.Spec
			other.VcnId = "ocid1.vcn.oc1..other"
			return recoverysdk.ListRecoveryServiceSubnetsResponse{
				RecoveryServiceSubnetCollection: recoverysdk.RecoveryServiceSubnetCollection{
					Items: []recoverysdk.RecoveryServiceSubnetSummary{
						makeSDKRecoveryServiceSubnetSummary("ocid1.recoveryservicesubnet.oc1..other", other, recoverysdk.LifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		requireStringPtr(t, "second ListRecoveryServiceSubnetsRequest.Page", request.Page, "page-2")
		return recoverysdk.ListRecoveryServiceSubnetsResponse{
			RecoveryServiceSubnetCollection: recoverysdk.RecoveryServiceSubnetCollection{
				Items: []recoverysdk.RecoveryServiceSubnetSummary{
					makeSDKRecoveryServiceSubnetSummary(testRecoveryServiceSubnetID, resource.Spec, recoverysdk.LifecycleStateActive),
				},
			},
		}, nil
	}
}

func recoveryServiceSubnetUpdateGetFn(
	t *testing.T,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	currentSpec recoveryv1beta1.RecoveryServiceSubnetSpec,
	calls *int,
) func(context.Context, recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
	t.Helper()
	return func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
		*calls = *calls + 1
		requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)

		spec := resource.Spec
		if *calls == 1 {
			spec = currentSpec
		}
		return recoverysdk.GetRecoveryServiceSubnetResponse{
			RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, spec, recoverysdk.LifecycleStateActive),
		}, nil
	}
}

func recoveryServiceSubnetMutableUpdateFn(
	t *testing.T,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	calls *int,
) func(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
	t.Helper()
	return func(_ context.Context, request recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
		*calls = *calls + 1
		requireStringPtr(t, "UpdateRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
		requireStringPtr(t, "UpdateRecoveryServiceSubnetDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
		requireStringSlice(t, "UpdateRecoveryServiceSubnetDetails.Subnets", request.Subnets, resource.Spec.Subnets)
		requireEmptyStringSlice(t, "UpdateRecoveryServiceSubnetDetails.NsgIds", request.NsgIds)
		requireEmptyStringMap(t, "UpdateRecoveryServiceSubnetDetails.FreeformTags", request.FreeformTags)
		requireEmptyDefinedTags(t, "UpdateRecoveryServiceSubnetDetails.DefinedTags", request.DefinedTags)
		return recoverysdk.UpdateRecoveryServiceSubnetResponse{
			OpcWorkRequestId: common.String("wr-update"),
			OpcRequestId:     common.String("opc-update"),
		}, nil
	}
}

func recoveryServiceSubnetUpdateWorkRequestFn(
	t *testing.T,
	calls *int,
) func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, request recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
		*calls = *calls + 1
		requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-update")
		return recoverysdk.GetWorkRequestResponse{
			WorkRequest: makeRecoveryServiceSubnetWorkRequest(
				"wr-update",
				recoverysdk.OperationStatusSucceeded,
				recoverysdk.OperationTypeUpdateRecoveryServiceSubnet,
				recoverysdk.ActionTypeUpdated,
			),
		}, nil
	}
}

func TestRecoveryServiceSubnetCreateOrUpdateBindsExistingByPagedVCNList(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	createCalled := false
	updateCalled := false
	getCalls := 0
	listCalls := 0

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		listFn: recoveryServiceSubnetPagedVCNListFn(t, resource, &listCalls),
		getFn: func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			return recoverysdk.GetRecoveryServiceSubnetResponse{
				RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, recoverysdk.CreateRecoveryServiceSubnetRequest) (recoverysdk.CreateRecoveryServiceSubnetResponse, error) {
			createCalled = true
			return recoverysdk.CreateRecoveryServiceSubnetResponse{}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateRecoveryServiceSubnetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateRecoveryServiceSubnet() called for existing recovery service subnet")
	}
	if updateCalled {
		t.Fatal("UpdateRecoveryServiceSubnet() called for matching recovery service subnet")
	}
	if listCalls != 2 {
		t.Fatalf("ListRecoveryServiceSubnets() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetRecoveryServiceSubnet() calls = %d, want 1 live read after bind", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testRecoveryServiceSubnetID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testRecoveryServiceSubnetID)
	}
	requireLastRecoveryServiceSubnetCondition(t, resource, shared.Active)
}

func TestRecoveryServiceSubnetCreateStartsWorkRequestAndRecordsRetryToken(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	createCalls := 0
	listCalls := 0
	workRequestCalls := 0

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		listFn: func(_ context.Context, request recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListRecoveryServiceSubnetsRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListRecoveryServiceSubnetsRequest.VcnId", request.VcnId, resource.Spec.VcnId)
			return recoverysdk.ListRecoveryServiceSubnetsResponse{}, nil
		},
		createFn: func(_ context.Context, request recoverysdk.CreateRecoveryServiceSubnetRequest) (recoverysdk.CreateRecoveryServiceSubnetResponse, error) {
			createCalls++
			requireRecoveryServiceSubnetCreateRequest(t, request, resource)
			if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
				t.Fatal("CreateRecoveryServiceSubnetRequest.OpcRetryToken is empty, want deterministic retry token")
			}
			return recoverysdk.CreateRecoveryServiceSubnetResponse{
				RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, resource.Spec, recoverysdk.LifecycleStateCreating),
				OpcWorkRequestId:      common.String("wr-create"),
				OpcRequestId:          common.String("opc-create"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-create")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeRecoveryServiceSubnetWorkRequest(
					"wr-create",
					recoverysdk.OperationStatusInProgress,
					recoverysdk.OperationTypeCreateRecoveryServiceSubnet,
					recoverysdk.ActionTypeCreated,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = false, want true for pending work request")
	}
	if createCalls != 1 {
		t.Fatalf("CreateRecoveryServiceSubnet() calls = %d, want 1", createCalls)
	}
	if listCalls != 1 {
		t.Fatalf("ListRecoveryServiceSubnets() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireRecoveryServiceSubnetAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
}

func TestRecoveryServiceSubnetCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	getCalls := 0
	updateCalled := false

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getFn: func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			return recoverysdk.GetRecoveryServiceSubnetResponse{
				RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateRecoveryServiceSubnetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateRecoveryServiceSubnet() called for matching observed state")
	}
	if getCalls != 1 {
		t.Fatalf("GetRecoveryServiceSubnet() calls = %d, want 1", getCalls)
	}
	requireLastRecoveryServiceSubnetCondition(t, resource, shared.Active)
}

func TestRecoveryServiceSubnetUpdateMutableFieldsAndRefreshesObservedState(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	currentSpec := resource.Spec
	resource.Spec.DisplayName = "rss-updated"
	resource.Spec.Subnets = []string{"ocid1.subnet.oc1..updated"}
	resource.Spec.NsgIds = []string{}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getFn:            recoveryServiceSubnetUpdateGetFn(t, resource, currentSpec, &getCalls),
		updateFn:         recoveryServiceSubnetMutableUpdateFn(t, resource, &updateCalls),
		getWorkRequestFn: recoveryServiceSubnetUpdateWorkRequestFn(t, &workRequestCalls),
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = true, want false after succeeded update work request")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateRecoveryServiceSubnet() calls = %d, want 1", updateCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetRecoveryServiceSubnet() calls = %d, want current read plus refresh after work request", getCalls)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastRecoveryServiceSubnetCondition(t, resource, shared.Active)
}

func TestRecoveryServiceSubnetCompartmentDriftUsesMoveWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	currentSpec := resource.Spec
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	getCalls := 0
	changeCalls := 0
	updateCalled := false
	workRequestCalled := false

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getFn: func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			return recoverysdk.GetRecoveryServiceSubnetResponse{
				RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, currentSpec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		changeCompartmentFn: func(_ context.Context, request recoverysdk.ChangeRecoveryServiceSubnetCompartmentRequest) (recoverysdk.ChangeRecoveryServiceSubnetCompartmentResponse, error) {
			changeCalls++
			requireStringPtr(t, "ChangeRecoveryServiceSubnetCompartmentRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			requireStringPtr(t, "ChangeRecoveryServiceSubnetCompartmentDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			return recoverysdk.ChangeRecoveryServiceSubnetCompartmentResponse{
				OpcWorkRequestId: common.String("wr-move"),
				OpcRequestId:     common.String("opc-move"),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateRecoveryServiceSubnetResponse{}, nil
		},
		getWorkRequestFn: func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalled = true
			return recoverysdk.GetWorkRequestResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = false, want true while compartment move work request is pending")
	}
	if getCalls != 1 {
		t.Fatalf("GetRecoveryServiceSubnet() calls = %d, want 1 current read", getCalls)
	}
	if changeCalls != 1 {
		t.Fatalf("ChangeRecoveryServiceSubnetCompartment() calls = %d, want 1", changeCalls)
	}
	if updateCalled {
		t.Fatal("UpdateRecoveryServiceSubnet() called for compartment move")
	}
	if workRequestCalled {
		t.Fatal("GetWorkRequest() called before the next reconcile resumes the move work request")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-move" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-move", got)
	}
	requireRecoveryServiceSubnetAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-move", shared.OSOKAsyncClassPending)
}

func TestRecoveryServiceSubnetMoveWorkRequestRefreshesCompartment(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   "wr-move",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	getCalls := 0
	workRequestCalls := 0

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getWorkRequestFn: func(_ context.Context, request recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-move")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeRecoveryServiceSubnetWorkRequest(
					"wr-move",
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.OperationTypeMoveRecoveryServiceSubnet,
					recoverysdk.ActionTypeUpdated,
				),
			}, nil
		},
		getFn: func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			return recoverysdk.GetRecoveryServiceSubnetResponse{
				RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = true, want false after move work request succeeds")
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetRecoveryServiceSubnet() calls = %d, want 1 refresh read", getCalls)
	}
	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want cleared tracker", resource.Status.OsokStatus.Async.Current)
	}
	requireLastRecoveryServiceSubnetCondition(t, resource, shared.Active)
}

func TestRecoveryServiceSubnetNoopsWhenSubnetListsReadBackReordered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		updateSpec       func(*recoveryv1beta1.RecoveryServiceSubnet)
		updateReadback   func(*recoverysdk.RecoveryServiceSubnet)
		requireReadback  func(*testing.T, *recoveryv1beta1.RecoveryServiceSubnet)
		unexpectedUpdate string
	}{
		{
			name: "subnets",
			updateSpec: func(resource *recoveryv1beta1.RecoveryServiceSubnet) {
				resource.Spec.Subnets = []string{"ocid1.subnet.oc1..a", "ocid1.subnet.oc1..b"}
			},
			updateReadback: func(current *recoverysdk.RecoveryServiceSubnet) {
				current.Subnets = []string{"ocid1.subnet.oc1..b", "ocid1.subnet.oc1..a"}
			},
			requireReadback: func(t *testing.T, resource *recoveryv1beta1.RecoveryServiceSubnet) {
				t.Helper()
				requireStringSlice(t, "status.subnets", resource.Status.Subnets, []string{"ocid1.subnet.oc1..b", "ocid1.subnet.oc1..a"})
			},
			unexpectedUpdate: "UpdateRecoveryServiceSubnet() called for reordered subnets",
		},
		{
			name: "nsgIds",
			updateSpec: func(resource *recoveryv1beta1.RecoveryServiceSubnet) {
				resource.Spec.NsgIds = []string{"ocid1.nsg.oc1..a", "ocid1.nsg.oc1..b"}
			},
			updateReadback: func(current *recoverysdk.RecoveryServiceSubnet) {
				current.NsgIds = []string{"ocid1.nsg.oc1..b", "ocid1.nsg.oc1..a"}
			},
			requireReadback: func(t *testing.T, resource *recoveryv1beta1.RecoveryServiceSubnet) {
				t.Helper()
				requireStringSlice(t, "status.nsgIds", resource.Status.NsgIds, []string{"ocid1.nsg.oc1..b", "ocid1.nsg.oc1..a"})
			},
			unexpectedUpdate: "UpdateRecoveryServiceSubnet() called for reordered nsgIds",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			resource := makeRecoveryServiceSubnetResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
			test.updateSpec(resource)
			getCalls := 0
			updateCalled := false

			client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
				getFn: func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
					getCalls++
					requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
					current := makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, resource.Spec, recoverysdk.LifecycleStateActive)
					test.updateReadback(&current)
					return recoverysdk.GetRecoveryServiceSubnetResponse{RecoveryServiceSubnet: current}, nil
				},
				updateFn: func(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
					updateCalled = true
					return recoverysdk.UpdateRecoveryServiceSubnetResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatal("CreateOrUpdate() successful = false, want true")
			}
			if updateCalled {
				t.Fatal(test.unexpectedUpdate)
			}
			if getCalls != 1 {
				t.Fatalf("GetRecoveryServiceSubnet() calls = %d, want 1", getCalls)
			}
			test.requireReadback(t, resource)
			requireLastRecoveryServiceSubnetCondition(t, resource, shared.Active)
		})
	}
}

func TestRecoveryServiceSubnetCreateOnlyDriftRejectsImmutableUpdate(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	currentSpec := resource.Spec
	resource.Spec.VcnId = "ocid1.vcn.oc1..changed"
	updateCalled := false

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getFn: func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			return recoverysdk.GetRecoveryServiceSubnetResponse{
				RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, currentSpec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateRecoveryServiceSubnetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeRecoveryServiceSubnetRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateRecoveryServiceSubnet() called after create-only drift")
	}
	if !strings.Contains(err.Error(), "vcnId") {
		t.Fatalf("CreateOrUpdate() error = %v, want vcnId drift detail", err)
	}
}

func TestRecoveryServiceSubnetDeleteStartsWorkRequestAndKeepsFinalizer(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	getCalls := 0
	deleteCalls := 0
	workRequestCalls := 0

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getFn: func(_ context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			return recoverysdk.GetRecoveryServiceSubnetResponse{
				RecoveryServiceSubnet: makeSDKRecoveryServiceSubnet(testRecoveryServiceSubnetID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request recoverysdk.DeleteRecoveryServiceSubnetRequest) (recoverysdk.DeleteRecoveryServiceSubnetResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteRecoveryServiceSubnetRequest.RecoveryServiceSubnetId", request.RecoveryServiceSubnetId, testRecoveryServiceSubnetID)
			return recoverysdk.DeleteRecoveryServiceSubnetResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-delete")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeRecoveryServiceSubnetWorkRequest(
					"wr-delete",
					recoverysdk.OperationStatusInProgress,
					recoverysdk.OperationTypeDeleteRecoveryServiceSubnet,
					recoverysdk.ActionTypeDeleted,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if getCalls != 2 {
		t.Fatalf("GetRecoveryServiceSubnet() calls = %d, want preflight plus generatedruntime confirm read", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteRecoveryServiceSubnet() calls = %d, want 1", deleteCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set while delete work request is pending")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireRecoveryServiceSubnetAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
}

func TestRecoveryServiceSubnetDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	deleteCalled := false

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getFn: func(context.Context, recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			return recoverysdk.GetRecoveryServiceSubnetResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
		deleteFn: func(context.Context, recoverysdk.DeleteRecoveryServiceSubnetRequest) (recoverysdk.DeleteRecoveryServiceSubnetResponse, error) {
			deleteCalled = true
			return recoverysdk.DeleteRecoveryServiceSubnetResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found failure")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous not-found")
	}
	if deleteCalled {
		t.Fatal("DeleteRecoveryServiceSubnet() called after ambiguous confirm read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set after ambiguous confirm read")
	}
}

func TestRecoveryServiceSubnetDeleteWorkRequestSucceededRejectsAuthShapedReadback(t *testing.T) {
	t.Parallel()

	resource := makeRecoveryServiceSubnetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRecoveryServiceSubnetID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	deleteCalled := false

	client := newTestRecoveryServiceSubnetClient(&fakeRecoveryServiceSubnetOCIClient{
		getWorkRequestFn: func(_ context.Context, request recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-delete")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeRecoveryServiceSubnetWorkRequest(
					"wr-delete",
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.OperationTypeDeleteRecoveryServiceSubnet,
					recoverysdk.ActionTypeDeleted,
				),
			}, nil
		},
		getFn: func(context.Context, recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
			return recoverysdk.GetRecoveryServiceSubnetResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
		deleteFn: func(context.Context, recoverysdk.DeleteRecoveryServiceSubnetRequest) (recoverysdk.DeleteRecoveryServiceSubnetResponse, error) {
			deleteCalled = true
			return recoverysdk.DeleteRecoveryServiceSubnetResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous readback failure")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous post-work-request readback")
	}
	if deleteCalled {
		t.Fatal("DeleteRecoveryServiceSubnet() called while resuming an existing delete work request")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set after ambiguous post-work-request readback")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func requireRecoveryServiceSubnetCreateRequest(
	t *testing.T,
	request recoverysdk.CreateRecoveryServiceSubnetRequest,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
) {
	t.Helper()
	requireStringPtr(t, "CreateRecoveryServiceSubnetDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateRecoveryServiceSubnetDetails.VcnId", request.VcnId, resource.Spec.VcnId)
	requireStringPtr(t, "CreateRecoveryServiceSubnetDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateRecoveryServiceSubnetDetails.SubnetId", request.SubnetId, resource.Spec.SubnetId)
	requireStringSlice(t, "CreateRecoveryServiceSubnetDetails.Subnets", request.Subnets, resource.Spec.Subnets)
	requireStringSlice(t, "CreateRecoveryServiceSubnetDetails.NsgIds", request.NsgIds, resource.Spec.NsgIds)
}

func requireRecoveryServiceSubnetAsync(
	t *testing.T,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want tracked async operation")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}

func requireLastRecoveryServiceSubnetCondition(
	t *testing.T,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	want shared.OSOKConditionType,
) {
	t.Helper()
	if len(resource.Status.OsokStatus.Conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	got := resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
	if got != want {
		t.Fatalf("last status condition = %q, want %q", got, want)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireStringSlice(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}

func requireEmptyStringSlice(t *testing.T, name string, got []string) {
	t.Helper()
	if got == nil || len(got) != 0 {
		t.Fatalf("%s = %#v, want explicit empty slice", name, got)
	}
}

func requireEmptyStringMap(t *testing.T, name string, got map[string]string) {
	t.Helper()
	if got == nil || len(got) != 0 {
		t.Fatalf("%s = %#v, want explicit empty map", name, got)
	}
}

func requireEmptyDefinedTags(t *testing.T, name string, got map[string]map[string]interface{}) {
	t.Helper()
	if got == nil || len(got) != 0 {
		t.Fatalf("%s = %#v, want explicit empty map", name, got)
	}
}
