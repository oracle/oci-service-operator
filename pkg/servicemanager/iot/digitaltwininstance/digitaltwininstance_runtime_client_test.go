/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwininstance

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDigitalTwinInstanceID        = "ocid1.digitaltwininstance.oc1..instance"
	testDigitalTwinInstanceDomainID  = "ocid1.iotdomain.oc1..domain"
	testDigitalTwinInstanceAdapterID = "ocid1.digitaltwinadapter.oc1..adapter"
	testDigitalTwinInstanceModelID   = "ocid1.digitaltwinmodel.oc1..model"
	testDigitalTwinInstanceSpecURI   = "dtmi:com:example:Thermostat;1"
	testDigitalTwinInstanceAuthID    = "ocid1.vaultsecret.oc1..auth"
	testDigitalTwinInstanceKey       = "device-001"
	testDigitalTwinInstanceName      = "instance-sample"
)

type fakeDigitalTwinInstanceOCIClient struct {
	createFn func(context.Context, iotsdk.CreateDigitalTwinInstanceRequest) (iotsdk.CreateDigitalTwinInstanceResponse, error)
	getFn    func(context.Context, iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error)
	listFn   func(context.Context, iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error)
	updateFn func(context.Context, iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error)
	deleteFn func(context.Context, iotsdk.DeleteDigitalTwinInstanceRequest) (iotsdk.DeleteDigitalTwinInstanceResponse, error)
}

func (f *fakeDigitalTwinInstanceOCIClient) CreateDigitalTwinInstance(ctx context.Context, req iotsdk.CreateDigitalTwinInstanceRequest) (iotsdk.CreateDigitalTwinInstanceResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return iotsdk.CreateDigitalTwinInstanceResponse{}, nil
}

func (f *fakeDigitalTwinInstanceOCIClient) GetDigitalTwinInstance(ctx context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return iotsdk.GetDigitalTwinInstanceResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "digital twin instance is missing")
}

func (f *fakeDigitalTwinInstanceOCIClient) ListDigitalTwinInstances(ctx context.Context, req iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return iotsdk.ListDigitalTwinInstancesResponse{}, nil
}

func (f *fakeDigitalTwinInstanceOCIClient) UpdateDigitalTwinInstance(ctx context.Context, req iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return iotsdk.UpdateDigitalTwinInstanceResponse{}, nil
}

func (f *fakeDigitalTwinInstanceOCIClient) DeleteDigitalTwinInstance(ctx context.Context, req iotsdk.DeleteDigitalTwinInstanceRequest) (iotsdk.DeleteDigitalTwinInstanceResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return iotsdk.DeleteDigitalTwinInstanceResponse{}, nil
}

func newTestDigitalTwinInstanceClient(client digitalTwinInstanceOCIClient) DigitalTwinInstanceServiceClient {
	return newDigitalTwinInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDigitalTwinInstanceResource() *iotv1beta1.DigitalTwinInstance {
	return &iotv1beta1.DigitalTwinInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDigitalTwinInstanceName,
			Namespace: "default",
		},
		Spec: iotv1beta1.DigitalTwinInstanceSpec{
			IotDomainId:             testDigitalTwinInstanceDomainID,
			AuthId:                  testDigitalTwinInstanceAuthID,
			ExternalKey:             testDigitalTwinInstanceKey,
			DisplayName:             testDigitalTwinInstanceName,
			Description:             "initial description",
			DigitalTwinAdapterId:    testDigitalTwinInstanceAdapterID,
			DigitalTwinModelId:      testDigitalTwinInstanceModelID,
			DigitalTwinModelSpecUri: testDigitalTwinInstanceSpecURI,
			FreeformTags:            map[string]string{"env": "test"},
			DefinedTags:             map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedDigitalTwinInstanceResource() *iotv1beta1.DigitalTwinInstance {
	resource := makeDigitalTwinInstanceResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalTwinInstanceID)
	resource.Status.Id = testDigitalTwinInstanceID
	resource.Status.IotDomainId = testDigitalTwinInstanceDomainID
	resource.Status.AuthId = testDigitalTwinInstanceAuthID
	resource.Status.ExternalKey = testDigitalTwinInstanceKey
	resource.Status.DisplayName = testDigitalTwinInstanceName
	resource.Status.DigitalTwinAdapterId = testDigitalTwinInstanceAdapterID
	resource.Status.DigitalTwinModelId = testDigitalTwinInstanceModelID
	resource.Status.DigitalTwinModelSpecUri = testDigitalTwinInstanceSpecURI
	resource.Status.LifecycleState = string(iotsdk.LifecycleStateActive)
	return resource
}

func makeDigitalTwinInstanceRequest(resource *iotv1beta1.DigitalTwinInstance) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKDigitalTwinInstance(
	id string,
	spec iotv1beta1.DigitalTwinInstanceSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinInstance {
	return iotsdk.DigitalTwinInstance{
		Id:                      common.String(id),
		IotDomainId:             common.String(spec.IotDomainId),
		AuthId:                  common.String(spec.AuthId),
		ExternalKey:             common.String(spec.ExternalKey),
		DisplayName:             common.String(spec.DisplayName),
		Description:             common.String(spec.Description),
		DigitalTwinAdapterId:    common.String(spec.DigitalTwinAdapterId),
		DigitalTwinModelId:      common.String(spec.DigitalTwinModelId),
		DigitalTwinModelSpecUri: common.String(spec.DigitalTwinModelSpecUri),
		LifecycleState:          state,
		FreeformTags:            cloneDigitalTwinInstanceStringMap(spec.FreeformTags),
		DefinedTags:             digitalTwinInstanceDefinedTags(spec.DefinedTags),
	}
}

func makeSDKDigitalTwinInstanceSummary(
	id string,
	spec iotv1beta1.DigitalTwinInstanceSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinInstanceSummary {
	return iotsdk.DigitalTwinInstanceSummary{
		Id:                      common.String(id),
		IotDomainId:             common.String(spec.IotDomainId),
		AuthId:                  common.String(spec.AuthId),
		ExternalKey:             common.String(spec.ExternalKey),
		DisplayName:             common.String(spec.DisplayName),
		Description:             common.String(spec.Description),
		DigitalTwinAdapterId:    common.String(spec.DigitalTwinAdapterId),
		DigitalTwinModelId:      common.String(spec.DigitalTwinModelId),
		DigitalTwinModelSpecUri: common.String(spec.DigitalTwinModelSpecUri),
		LifecycleState:          state,
		FreeformTags:            cloneDigitalTwinInstanceStringMap(spec.FreeformTags),
		DefinedTags:             digitalTwinInstanceDefinedTags(spec.DefinedTags),
	}
}

func TestDigitalTwinInstanceCreateOrUpdateBindsExistingInstanceByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinInstanceResource()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		listFn: func(_ context.Context, req iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error) {
			listCalls++
			requireStringPtr(t, "ListDigitalTwinInstancesRequest.IotDomainId", req.IotDomainId, resource.Spec.IotDomainId)
			requireStringPtr(t, "ListDigitalTwinInstancesRequest.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireStringPtr(t, "ListDigitalTwinInstancesRequest.DigitalTwinModelId", req.DigitalTwinModelId, resource.Spec.DigitalTwinModelId)
			requireStringPtr(t, "ListDigitalTwinInstancesRequest.DigitalTwinModelSpecUri", req.DigitalTwinModelSpecUri, resource.Spec.DigitalTwinModelSpecUri)
			if listCalls == 1 {
				if req.Page != nil {
					t.Fatalf("first ListDigitalTwinInstancesRequest.Page = %q, want nil", *req.Page)
				}
				otherSpec := resource.Spec
				otherSpec.ExternalKey = "different-device"
				return iotsdk.ListDigitalTwinInstancesResponse{
					DigitalTwinInstanceCollection: iotsdk.DigitalTwinInstanceCollection{
						Items: []iotsdk.DigitalTwinInstanceSummary{
							makeSDKDigitalTwinInstanceSummary("ocid1.digitaltwininstance.oc1..other", otherSpec, iotsdk.LifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second ListDigitalTwinInstancesRequest.Page", req.Page, "page-2")
			return iotsdk.ListDigitalTwinInstancesResponse{
				DigitalTwinInstanceCollection: iotsdk.DigitalTwinInstanceCollection{
					Items: []iotsdk.DigitalTwinInstanceSummary{
						makeSDKDigitalTwinInstanceSummary(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.GetDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinInstanceRequest) (iotsdk.CreateDigitalTwinInstanceResponse, error) {
			createCalled = true
			return iotsdk.CreateDigitalTwinInstanceResponse{}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinInstanceResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinInstanceRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateDigitalTwinInstance() called for existing instance")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinInstance() called for matching instance")
	}
	if listCalls != 2 {
		t.Fatalf("ListDigitalTwinInstances() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDigitalTwinInstance() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalTwinInstanceID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDigitalTwinInstanceID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinInstanceCreateRecordsPayloadRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinInstanceResource()
	listCalls := 0
	createCalls := 0

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		listFn: func(context.Context, iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error) {
			listCalls++
			return iotsdk.ListDigitalTwinInstancesResponse{}, nil
		},
		createFn: func(_ context.Context, req iotsdk.CreateDigitalTwinInstanceRequest) (iotsdk.CreateDigitalTwinInstanceResponse, error) {
			createCalls++
			requireDigitalTwinInstanceCreateRequest(t, req, resource)
			return iotsdk.CreateDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:        common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.GetDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinInstanceRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 {
		t.Fatalf("ListDigitalTwinInstances() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDigitalTwinInstance() calls = %d, want 1", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalTwinInstanceID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDigitalTwinInstanceID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if got := resource.Status.ExternalKey; got != resource.Spec.ExternalKey {
		t.Fatalf("status.externalKey = %q, want %q", got, resource.Spec.ExternalKey)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinInstanceCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinInstanceResource()
	updateCalled := false

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.GetDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinInstanceResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinInstanceRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinInstance() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinInstanceCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinInstanceResource()
	resource.Spec.AuthId = "ocid1.vaultsecret.oc1..updated"
	resource.Spec.ExternalKey = "device-002"
	resource.Spec.DisplayName = "updated-instance"
	resource.Spec.Description = "updated description"
	resource.Spec.DigitalTwinAdapterId = "ocid1.digitaltwinadapter.oc1..updated"
	resource.Spec.DigitalTwinModelId = "ocid1.digitaltwinmodel.oc1..updated"
	resource.Spec.DigitalTwinModelSpecUri = "dtmi:com:example:Updated;1"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := makeDigitalTwinInstanceResource().Spec
	getCalls := 0
	updateCalls := 0

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			if getCalls == 1 {
				return iotsdk.GetDigitalTwinInstanceResponse{
					DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, currentSpec, iotsdk.LifecycleStateActive),
				}, nil
			}
			return iotsdk.GetDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			requireStringPtr(t, "UpdateDigitalTwinInstanceDetails.AuthId", req.AuthId, resource.Spec.AuthId)
			requireStringPtr(t, "UpdateDigitalTwinInstanceDetails.ExternalKey", req.ExternalKey, resource.Spec.ExternalKey)
			requireStringPtr(t, "UpdateDigitalTwinInstanceDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireStringPtr(t, "UpdateDigitalTwinInstanceDetails.Description", req.Description, resource.Spec.Description)
			requireStringPtr(t, "UpdateDigitalTwinInstanceDetails.DigitalTwinAdapterId", req.DigitalTwinAdapterId, resource.Spec.DigitalTwinAdapterId)
			requireStringPtr(t, "UpdateDigitalTwinInstanceDetails.DigitalTwinModelId", req.DigitalTwinModelId, resource.Spec.DigitalTwinModelId)
			requireStringPtr(t, "UpdateDigitalTwinInstanceDetails.DigitalTwinModelSpecUri", req.DigitalTwinModelSpecUri, resource.Spec.DigitalTwinModelSpecUri)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateDigitalTwinInstanceDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateDigitalTwinInstanceDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return iotsdk.UpdateDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:        common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinInstanceRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDigitalTwinInstance() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetDigitalTwinInstance() calls = %d, want current read and update follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinInstanceCreateOrUpdateRejectsIotDomainDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinInstanceResource()
	resource.Spec.IotDomainId = "ocid1.iotdomain.oc1..different"
	currentSpec := resource.Spec
	currentSpec.IotDomainId = testDigitalTwinInstanceDomainID
	updateCalled := false

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.GetDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, currentSpec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinInstanceRequest) (iotsdk.UpdateDigitalTwinInstanceResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinInstanceResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinInstanceRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinInstance() called after create-only iotDomainId drift")
	}
	if !strings.Contains(err.Error(), "iotDomainId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want iotDomainId force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDigitalTwinInstanceDeleteKeepsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinInstanceResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			if getCalls <= 3 {
				return iotsdk.GetDigitalTwinInstanceResponse{
					DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
				}, nil
			}
			return iotsdk.GetDigitalTwinInstanceResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "digital twin instance is gone")
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinInstanceRequest) (iotsdk.DeleteDigitalTwinInstanceResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.DeleteDigitalTwinInstanceResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinInstance() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireDeletePendingStatus(t, resource)
	requireLastCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinInstance() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestDigitalTwinInstanceDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinInstanceResource()

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.GetDigitalTwinInstanceResponse{
				DigitalTwinInstance: makeSDKDigitalTwinInstance(testDigitalTwinInstanceID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinInstanceRequest) (iotsdk.DeleteDigitalTwinInstanceResponse, error) {
			requireStringPtr(t, "DeleteDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.DeleteDigitalTwinInstanceResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinInstanceDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinInstanceResource()
	deleteCalled := false

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinInstanceRequest) (iotsdk.GetDigitalTwinInstanceResponse, error) {
			requireStringPtr(t, "GetDigitalTwinInstanceRequest.DigitalTwinInstanceId", req.DigitalTwinInstanceId, testDigitalTwinInstanceID)
			return iotsdk.GetDigitalTwinInstanceResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, iotsdk.DeleteDigitalTwinInstanceRequest) (iotsdk.DeleteDigitalTwinInstanceResponse, error) {
			deleteCalled = true
			return iotsdk.DeleteDigitalTwinInstanceResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteDigitalTwinInstance() called after auth-shaped pre-delete confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinInstanceCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinInstanceResource()

	client := newTestDigitalTwinInstanceClient(&fakeDigitalTwinInstanceOCIClient{
		listFn: func(context.Context, iotsdk.ListDigitalTwinInstancesRequest) (iotsdk.ListDigitalTwinInstancesResponse, error) {
			return iotsdk.ListDigitalTwinInstancesResponse{}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinInstanceRequest) (iotsdk.CreateDigitalTwinInstanceResponse, error) {
			return iotsdk.CreateDigitalTwinInstanceResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinInstanceRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireDigitalTwinInstanceCreateRequest(
	t *testing.T,
	req iotsdk.CreateDigitalTwinInstanceRequest,
	resource *iotv1beta1.DigitalTwinInstance,
) {
	t.Helper()
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.IotDomainId", req.IotDomainId, resource.Spec.IotDomainId)
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.AuthId", req.AuthId, resource.Spec.AuthId)
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.ExternalKey", req.ExternalKey, resource.Spec.ExternalKey)
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.Description", req.Description, resource.Spec.Description)
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.DigitalTwinAdapterId", req.DigitalTwinAdapterId, resource.Spec.DigitalTwinAdapterId)
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.DigitalTwinModelId", req.DigitalTwinModelId, resource.Spec.DigitalTwinModelId)
	requireStringPtr(t, "CreateDigitalTwinInstanceDetails.DigitalTwinModelSpecUri", req.DigitalTwinModelSpecUri, resource.Spec.DigitalTwinModelSpecUri)
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateDigitalTwinInstanceDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateDigitalTwinInstanceDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateDigitalTwinInstanceRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireDeletePendingStatus(t *testing.T, resource *iotv1beta1.DigitalTwinInstance) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(iotsdk.LifecycleStateActive) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle delete pending ACTIVE", current)
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

func requireLastCondition(t *testing.T, resource *iotv1beta1.DigitalTwinInstance, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
