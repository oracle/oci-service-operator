/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package iotdomain

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
	testIotDomainID          = "ocid1.iotdomain.oc1..domain"
	testIotDomainGroupID     = "ocid1.iotdomaingroup.oc1..group"
	testIotDomainCompartment = "ocid1.compartment.oc1..compartment"
	testIotDomainName        = "domain-sample"
)

type fakeIotDomainOCIClient struct {
	createFn      func(context.Context, iotsdk.CreateIotDomainRequest) (iotsdk.CreateIotDomainResponse, error)
	getFn         func(context.Context, iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error)
	listFn        func(context.Context, iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error)
	updateFn      func(context.Context, iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error)
	deleteFn      func(context.Context, iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error)
	workRequestFn func(context.Context, iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error)
}

func (f *fakeIotDomainOCIClient) CreateIotDomain(ctx context.Context, req iotsdk.CreateIotDomainRequest) (iotsdk.CreateIotDomainResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return iotsdk.CreateIotDomainResponse{}, nil
}

func (f *fakeIotDomainOCIClient) GetIotDomain(ctx context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return iotsdk.GetIotDomainResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "IotDomain is missing")
}

func (f *fakeIotDomainOCIClient) ListIotDomains(ctx context.Context, req iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return iotsdk.ListIotDomainsResponse{}, nil
}

func (f *fakeIotDomainOCIClient) UpdateIotDomain(ctx context.Context, req iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return iotsdk.UpdateIotDomainResponse{}, nil
}

func (f *fakeIotDomainOCIClient) DeleteIotDomain(ctx context.Context, req iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return iotsdk.DeleteIotDomainResponse{}, nil
}

func (f *fakeIotDomainOCIClient) GetWorkRequest(ctx context.Context, req iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return iotsdk.GetWorkRequestResponse{}, nil
}

func newTestIotDomainClient(client iotDomainOCIClient) IotDomainServiceClient {
	return newIotDomainServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeIotDomainResource() *iotv1beta1.IotDomain {
	return &iotv1beta1.IotDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIotDomainName,
			Namespace: "default",
		},
		Spec: iotv1beta1.IotDomainSpec{
			IotDomainGroupId: testIotDomainGroupID,
			CompartmentId:    testIotDomainCompartment,
			DisplayName:      testIotDomainName,
			Description:      "initial description",
			FreeformTags:     map[string]string{"env": "test"},
			DefinedTags:      map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedIotDomainResource() *iotv1beta1.IotDomain {
	resource := makeIotDomainResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testIotDomainID)
	resource.Status.Id = testIotDomainID
	resource.Status.IotDomainGroupId = testIotDomainGroupID
	resource.Status.CompartmentId = testIotDomainCompartment
	resource.Status.DisplayName = testIotDomainName
	resource.Status.LifecycleState = string(iotsdk.IotDomainLifecycleStateActive)
	return resource
}

func makeIotDomainRequest(resource *iotv1beta1.IotDomain) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKIotDomain(
	id string,
	spec iotv1beta1.IotDomainSpec,
	state iotsdk.IotDomainLifecycleStateEnum,
) iotsdk.IotDomain {
	return iotsdk.IotDomain{
		Id:               common.String(id),
		IotDomainGroupId: common.String(spec.IotDomainGroupId),
		CompartmentId:    common.String(spec.CompartmentId),
		DisplayName:      common.String(spec.DisplayName),
		Description:      common.String(spec.Description),
		LifecycleState:   state,
		FreeformTags:     cloneIotDomainStringMap(spec.FreeformTags),
		DefinedTags:      iotDomainDefinedTags(spec.DefinedTags),
	}
}

func makeSDKIotDomainSummary(
	id string,
	spec iotv1beta1.IotDomainSpec,
	state iotsdk.IotDomainLifecycleStateEnum,
) iotsdk.IotDomainSummary {
	return iotsdk.IotDomainSummary{
		Id:               common.String(id),
		IotDomainGroupId: common.String(spec.IotDomainGroupId),
		CompartmentId:    common.String(spec.CompartmentId),
		DisplayName:      common.String(spec.DisplayName),
		Description:      common.String(spec.Description),
		LifecycleState:   state,
		FreeformTags:     cloneIotDomainStringMap(spec.FreeformTags),
		DefinedTags:      iotDomainDefinedTags(spec.DefinedTags),
	}
}

func makeIotDomainWorkRequest(
	id string,
	operation iotsdk.OperationTypeEnum,
	status iotsdk.OperationStatusEnum,
	resourceID string,
) iotsdk.WorkRequest {
	workRequest := iotsdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operation,
		Status:          status,
		PercentComplete: common.Float32(100),
	}
	if resourceID != "" {
		workRequest.Resources = []iotsdk.WorkRequestResource{
			{
				EntityType: common.String("IotDomain"),
				ActionType: iotDomainActionForOperation(operation),
				Identifier: common.String(resourceID),
			},
		}
	}
	return workRequest
}

func iotDomainActionForOperation(operation iotsdk.OperationTypeEnum) iotsdk.ActionTypeEnum {
	switch operation {
	case iotsdk.OperationTypeCreateIotDomain:
		return iotsdk.ActionTypeCreated
	case iotsdk.OperationTypeUpdateIotDomain:
		return iotsdk.ActionTypeUpdated
	case iotsdk.OperationTypeDeleteIotDomain:
		return iotsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func TestIotDomainCreateOrUpdateBindsExistingDomainByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeIotDomainResource()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		listFn: func(_ context.Context, req iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListIotDomainsRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListIotDomainsRequest.IotDomainGroupId", req.IotDomainGroupId, resource.Spec.IotDomainGroupId)
			requireStringPtr(t, "ListIotDomainsRequest.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			if listCalls == 1 {
				if req.Page != nil {
					t.Fatalf("first ListIotDomainsRequest.Page = %q, want nil", *req.Page)
				}
				otherSpec := resource.Spec
				otherSpec.DisplayName = "other-domain"
				return iotsdk.ListIotDomainsResponse{
					IotDomainCollection: iotsdk.IotDomainCollection{
						Items: []iotsdk.IotDomainSummary{
							makeSDKIotDomainSummary("ocid1.iotdomain.oc1..other", otherSpec, iotsdk.IotDomainLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second ListIotDomainsRequest.Page", req.Page, "page-2")
			return iotsdk.ListIotDomainsResponse{
				IotDomainCollection: iotsdk.IotDomainCollection{
					Items: []iotsdk.IotDomainSummary{
						makeSDKIotDomainSummary(testIotDomainID, resource.Spec, iotsdk.IotDomainLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
			getCalls++
			requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			return iotsdk.GetIotDomainResponse{
				IotDomain: makeSDKIotDomain(testIotDomainID, resource.Spec, iotsdk.IotDomainLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, iotsdk.CreateIotDomainRequest) (iotsdk.CreateIotDomainResponse, error) {
			createCalled = true
			return iotsdk.CreateIotDomainResponse{}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error) {
			updateCalled = true
			return iotsdk.UpdateIotDomainResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateIotDomain() called for existing domain")
	}
	if updateCalled {
		t.Fatal("UpdateIotDomain() called for matching domain")
	}
	if listCalls != 2 {
		t.Fatalf("ListIotDomains() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetIotDomain() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testIotDomainID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testIotDomainID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestIotDomainCreateRecordsTypedPayloadWorkRequestRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeIotDomainResource()
	listCalls := 0
	createCalls := 0
	workRequestCalls := 0
	getCalls := 0

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		listFn: func(context.Context, iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error) {
			listCalls++
			return iotsdk.ListIotDomainsResponse{}, nil
		},
		createFn: func(_ context.Context, req iotsdk.CreateIotDomainRequest) (iotsdk.CreateIotDomainResponse, error) {
			createCalls++
			requireIotDomainCreateRequest(t, req, resource)
			return iotsdk.CreateIotDomainResponse{
				IotDomain:        makeSDKIotDomain(testIotDomainID, resource.Spec, iotsdk.IotDomainLifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-create")
			return iotsdk.GetWorkRequestResponse{
				WorkRequest: makeIotDomainWorkRequest("wr-create", iotsdk.OperationTypeCreateIotDomain, iotsdk.OperationStatusSucceeded, testIotDomainID),
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
			getCalls++
			requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			return iotsdk.GetIotDomainResponse{
				IotDomain: makeSDKIotDomain(testIotDomainID, resource.Spec, iotsdk.IotDomainLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 {
		t.Fatalf("ListIotDomains() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 || workRequestCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts create/workRequest/get = %d/%d/%d, want 1/1/1", createCalls, workRequestCalls, getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testIotDomainID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testIotDomainID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after succeeded work request", resource.Status.OsokStatus.Async.Current)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestIotDomainCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainResource()
	updateCalled := false

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
			requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			return iotsdk.GetIotDomainResponse{
				IotDomain: makeSDKIotDomain(testIotDomainID, resource.Spec, iotsdk.IotDomainLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error) {
			updateCalled = true
			return iotsdk.UpdateIotDomainResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateIotDomain() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestIotDomainCreateOrUpdateUpdatesMutableFieldsThroughWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainResource()
	resource.Spec.DisplayName = "updated-domain"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := resource.Spec
	currentSpec.DisplayName = testIotDomainName
	currentSpec.Description = "initial description"
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
			getCalls++
			requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			if getCalls == 1 {
				return iotsdk.GetIotDomainResponse{
					IotDomain: makeSDKIotDomain(testIotDomainID, currentSpec, iotsdk.IotDomainLifecycleStateActive),
				}, nil
			}
			return iotsdk.GetIotDomainResponse{
				IotDomain: makeSDKIotDomain(testIotDomainID, resource.Spec, iotsdk.IotDomainLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			requireStringPtr(t, "UpdateIotDomainDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireStringPtr(t, "UpdateIotDomainDetails.Description", req.Description, resource.Spec.Description)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateIotDomainDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateIotDomainDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return iotsdk.UpdateIotDomainResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-update")
			return iotsdk.GetWorkRequestResponse{
				WorkRequest: makeIotDomainWorkRequest("wr-update", iotsdk.OperationTypeUpdateIotDomain, iotsdk.OperationStatusSucceeded, testIotDomainID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 || workRequestCalls != 1 || getCalls != 2 {
		t.Fatalf("call counts update/workRequest/get = %d/%d/%d, want 1/1/2", updateCalls, workRequestCalls, getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestIotDomainCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mutateSpec func(*iotv1beta1.IotDomain)
		wantError  string
	}{
		{
			name: "changed iotDomainGroupId",
			mutateSpec: func(resource *iotv1beta1.IotDomain) {
				resource.Spec.IotDomainGroupId = "ocid1.iotdomaingroup.oc1..different"
			},
			wantError: "iotDomainGroupId changes",
		},
		{
			name: "omitted compartmentId",
			mutateSpec: func(resource *iotv1beta1.IotDomain) {
				resource.Spec.CompartmentId = ""
			},
			wantError: "compartmentId changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedIotDomainResource()
			currentSpec := resource.Spec
			resource.Spec.DisplayName = "updated-domain"
			tt.mutateSpec(resource)
			updateCalled := false

			client := newTestIotDomainClient(&fakeIotDomainOCIClient{
				getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
					requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
					return iotsdk.GetIotDomainResponse{
						IotDomain: makeSDKIotDomain(testIotDomainID, currentSpec, iotsdk.IotDomainLifecycleStateActive),
					}, nil
				},
				updateFn: func(context.Context, iotsdk.UpdateIotDomainRequest) (iotsdk.UpdateIotDomainResponse, error) {
					updateCalled = true
					return iotsdk.UpdateIotDomainResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainRequest(resource))
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() successful = true, want false")
			}
			if updateCalled {
				t.Fatal("UpdateIotDomain() called after create-only drift")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("CreateOrUpdate() error = %v, want %q", err, tt.wantError)
			}
			requireLastCondition(t, resource, shared.Failed)
		})
	}
}

type iotDomainDeleteWorkRequestFixture struct {
	resource         *iotv1beta1.IotDomain
	client           IotDomainServiceClient
	getCalls         int
	deleteCalls      int
	workRequestCalls int
}

func TestIotDomainDeleteTracksWorkRequestUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	fixture := newIotDomainDeleteWorkRequestFixture(t)
	fixture.requireFirstDeletePending(t)
	fixture.requireSecondDeleteConfirmed(t)
}

func newIotDomainDeleteWorkRequestFixture(t *testing.T) *iotDomainDeleteWorkRequestFixture {
	t.Helper()
	fixture := &iotDomainDeleteWorkRequestFixture{resource: makeTrackedIotDomainResource()}
	fixture.client = newTestIotDomainClient(&fakeIotDomainOCIClient{
		getFn:         fixture.getIotDomainDuringDelete(t),
		deleteFn:      fixture.deleteIotDomain(t),
		workRequestFn: fixture.getDeleteWorkRequest(t),
	})
	return fixture
}

func (f *iotDomainDeleteWorkRequestFixture) getIotDomainDuringDelete(
	t *testing.T,
) func(context.Context, iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
	t.Helper()
	return func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
		f.getCalls++
		requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
		if f.getCalls <= 2 {
			return iotsdk.GetIotDomainResponse{
				IotDomain: makeSDKIotDomain(testIotDomainID, f.resource.Spec, iotsdk.IotDomainLifecycleStateActive),
			}, nil
		}
		return iotsdk.GetIotDomainResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "IotDomain is gone")
	}
}

func (f *iotDomainDeleteWorkRequestFixture) deleteIotDomain(
	t *testing.T,
) func(context.Context, iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error) {
	t.Helper()
	return func(_ context.Context, req iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error) {
		f.deleteCalls++
		requireStringPtr(t, "DeleteIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
		return iotsdk.DeleteIotDomainResponse{
			OpcWorkRequestId: common.String("wr-delete"),
			OpcRequestId:     common.String("opc-delete"),
		}, nil
	}
}

func (f *iotDomainDeleteWorkRequestFixture) getDeleteWorkRequest(
	t *testing.T,
) func(context.Context, iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, req iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error) {
		f.workRequestCalls++
		requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
		return iotsdk.GetWorkRequestResponse{
			WorkRequest: makeIotDomainWorkRequest("wr-delete", iotsdk.OperationTypeDeleteIotDomain, f.deleteWorkRequestStatus(), testIotDomainID),
		}, nil
	}
}

func (f *iotDomainDeleteWorkRequestFixture) deleteWorkRequestStatus() iotsdk.OperationStatusEnum {
	if f.workRequestCalls > 1 {
		return iotsdk.OperationStatusSucceeded
	}
	return iotsdk.OperationStatusInProgress
}

func (f *iotDomainDeleteWorkRequestFixture) requireFirstDeletePending(t *testing.T) {
	t.Helper()
	deleted, err := f.client.Delete(context.Background(), f.resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while work request is pending")
	}
	if f.deleteCalls != 1 {
		t.Fatalf("DeleteIotDomain() calls after first delete = %d, want 1", f.deleteCalls)
	}
	if got := f.resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireWorkRequestStatus(t, f.resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	requireLastCondition(t, f.resource, shared.Terminating)
}

func (f *iotDomainDeleteWorkRequestFixture) requireSecondDeleteConfirmed(t *testing.T) {
	t.Helper()
	deleted, err := f.client.Delete(context.Background(), f.resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after work request and NotFound confirmation")
	}
	if f.deleteCalls != 1 {
		t.Fatalf("DeleteIotDomain() calls after confirmed delete = %d, want still 1", f.deleteCalls)
	}
	if f.resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, f.resource, shared.Terminating)
}

func TestIotDomainDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainResource()

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
			requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			return iotsdk.GetIotDomainResponse{
				IotDomain: makeSDKIotDomain(testIotDomainID, resource.Spec, iotsdk.IotDomainLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error) {
			requireStringPtr(t, "DeleteIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			return iotsdk.DeleteIotDomainResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
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

func TestIotDomainDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainResource()
	deleteCalled := false

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
			requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			return iotsdk.GetIotDomainResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error) {
			deleteCalled = true
			return iotsdk.DeleteIotDomainResponse{}, nil
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
		t.Fatal("DeleteIotDomain() called after auth-shaped pre-delete confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestIotDomainDeleteRejectsAuthShapedSucceededWorkRequestConfirmation(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		workRequestFn: func(_ context.Context, req iotsdk.GetWorkRequestRequest) (iotsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
			return iotsdk.GetWorkRequestResponse{
				WorkRequest: makeIotDomainWorkRequest("wr-delete", iotsdk.OperationTypeDeleteIotDomain, iotsdk.OperationStatusSucceeded, testIotDomainID),
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetIotDomainRequest) (iotsdk.GetIotDomainResponse, error) {
			requireStringPtr(t, "GetIotDomainRequest.IotDomainId", req.IotDomainId, testIotDomainID)
			return iotsdk.GetIotDomainResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, iotsdk.DeleteIotDomainRequest) (iotsdk.DeleteIotDomainResponse, error) {
			t.Fatal("DeleteIotDomain() called for existing delete work request")
			return iotsdk.DeleteIotDomainResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous succeeded work-request confirmation error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped succeeded work-request confirmation")
	}
	if !strings.Contains(err.Error(), "work request wr-delete succeeded") {
		t.Fatalf("Delete() error = %v, want succeeded work-request context", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestIotDomainCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeIotDomainResource()

	client := newTestIotDomainClient(&fakeIotDomainOCIClient{
		listFn: func(context.Context, iotsdk.ListIotDomainsRequest) (iotsdk.ListIotDomainsResponse, error) {
			return iotsdk.ListIotDomainsResponse{}, nil
		},
		createFn: func(context.Context, iotsdk.CreateIotDomainRequest) (iotsdk.CreateIotDomainResponse, error) {
			return iotsdk.CreateIotDomainResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainRequest(resource))
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

func requireIotDomainCreateRequest(
	t *testing.T,
	req iotsdk.CreateIotDomainRequest,
	resource *iotv1beta1.IotDomain,
) {
	t.Helper()
	requireStringPtr(t, "CreateIotDomainDetails.IotDomainGroupId", req.IotDomainGroupId, resource.Spec.IotDomainGroupId)
	requireStringPtr(t, "CreateIotDomainDetails.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateIotDomainDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateIotDomainDetails.Description", req.Description, resource.Spec.Description)
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateIotDomainDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateIotDomainDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateIotDomainRequest.OpcRetryToken is empty, want deterministic retry token")
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

func requireWorkRequestStatus(
	t *testing.T,
	resource *iotv1beta1.IotDomain,
	wantPhase shared.OSOKAsyncPhase,
	wantID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want work request tracker")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != wantPhase ||
		current.WorkRequestID != wantID ||
		current.NormalizedClass != wantClass {
		t.Fatalf("status.status.async.current = %#v, want %s/%s/%s", current, wantPhase, wantID, wantClass)
	}
}

func requireLastCondition(t *testing.T, resource *iotv1beta1.IotDomain, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
