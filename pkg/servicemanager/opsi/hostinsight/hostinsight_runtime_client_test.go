/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package hostinsight

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testHostInsightID      = "ocid1.hostinsight.oc1..host"
	testHostInsightOtherID = "ocid1.hostinsight.oc1..other"
	testHostInsightCompID  = "ocid1.compartment.oc1..compartment"
	testHostInsightCompute = "ocid1.instance.oc1..compute"
	testHostInsightAgent   = "ocid1.managementagent.oc1..agent"
)

type fakeHostInsightOCIClient struct {
	createRequests      []opsisdk.CreateHostInsightRequest
	getRequests         []opsisdk.GetHostInsightRequest
	listRequests        []opsisdk.ListHostInsightsRequest
	updateRequests      []opsisdk.UpdateHostInsightRequest
	deleteRequests      []opsisdk.DeleteHostInsightRequest
	workRequestRequests []opsisdk.GetWorkRequestRequest

	createFn      func(context.Context, opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error)
	getFn         func(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error)
	listFn        func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error)
	updateFn      func(context.Context, opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error)
	deleteFn      func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error)
	workRequestFn func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeHostInsightOCIClient) CreateHostInsight(
	ctx context.Context,
	request opsisdk.CreateHostInsightRequest,
) (opsisdk.CreateHostInsightResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return opsisdk.CreateHostInsightResponse{}, nil
}

func (f *fakeHostInsightOCIClient) GetHostInsight(
	ctx context.Context,
	request opsisdk.GetHostInsightRequest,
) (opsisdk.GetHostInsightResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return opsisdk.GetHostInsightResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "host insight not found")
}

func (f *fakeHostInsightOCIClient) ListHostInsights(
	ctx context.Context,
	request opsisdk.ListHostInsightsRequest,
) (opsisdk.ListHostInsightsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return opsisdk.ListHostInsightsResponse{}, nil
}

func (f *fakeHostInsightOCIClient) UpdateHostInsight(
	ctx context.Context,
	request opsisdk.UpdateHostInsightRequest,
) (opsisdk.UpdateHostInsightResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return opsisdk.UpdateHostInsightResponse{}, nil
}

func (f *fakeHostInsightOCIClient) DeleteHostInsight(
	ctx context.Context,
	request opsisdk.DeleteHostInsightRequest,
) (opsisdk.DeleteHostInsightResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return opsisdk.DeleteHostInsightResponse{}, nil
}

func (f *fakeHostInsightOCIClient) GetWorkRequest(
	ctx context.Context,
	request opsisdk.GetWorkRequestRequest,
) (opsisdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return opsisdk.GetWorkRequestResponse{}, nil
}

func newHostInsightTestClient(fake *fakeHostInsightOCIClient) HostInsightServiceClient {
	return newHostInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func newHostInsightResource() *opsiv1beta1.HostInsight {
	return &opsiv1beta1.HostInsight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host-insight",
			Namespace: "default",
			UID:       types.UID("host-insight-uid"),
		},
		Spec: opsiv1beta1.HostInsightSpec{
			CompartmentId: testHostInsightCompID,
			ComputeId:     testHostInsightCompute,
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func newExternalHostInsightResource() *opsiv1beta1.HostInsight {
	resource := newHostInsightResource()
	resource.Spec.EntitySource = hostInsightEntitySourceMacsManagedExternalHost
	resource.Spec.ComputeId = ""
	resource.Spec.ManagementAgentId = testHostInsightAgent
	return resource
}

func newExistingHostInsightResource(id string) *opsiv1beta1.HostInsight {
	resource := newHostInsightResource()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	return resource
}

func sdkHostInsightFromSpec(
	id string,
	spec opsiv1beta1.HostInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.MacsManagedCloudHostInsight {
	return opsisdk.MacsManagedCloudHostInsight{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		HostName:          common.String("host-alpha"),
		FreeformTags:      hostInsightCloneStringMap(spec.FreeformTags),
		DefinedTags:       hostInsightDefinedTagsFromSpec(spec.DefinedTags),
		ComputeId:         common.String(spec.ComputeId),
		ManagementAgentId: common.String(testHostInsightAgent),
		Status:            opsisdk.ResourceStatusEnabled,
		LifecycleState:    state,
	}
}

func sdkExternalHostInsightFromSpec(
	id string,
	spec opsiv1beta1.HostInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.MacsManagedExternalHostInsight {
	return opsisdk.MacsManagedExternalHostInsight{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		HostName:          common.String("host-alpha"),
		ManagementAgentId: common.String(spec.ManagementAgentId),
		FreeformTags:      hostInsightCloneStringMap(spec.FreeformTags),
		DefinedTags:       hostInsightDefinedTagsFromSpec(spec.DefinedTags),
		Status:            opsisdk.ResourceStatusEnabled,
		LifecycleState:    state,
	}
}

func sdkHostInsightSummaryFromSpec(
	id string,
	spec opsiv1beta1.HostInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.MacsManagedCloudHostInsightSummary {
	return opsisdk.MacsManagedCloudHostInsightSummary{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		HostName:          common.String("host-alpha"),
		ComputeId:         common.String(spec.ComputeId),
		ManagementAgentId: common.String(testHostInsightAgent),
		FreeformTags:      hostInsightCloneStringMap(spec.FreeformTags),
		DefinedTags:       hostInsightDefinedTagsFromSpec(spec.DefinedTags),
		Status:            opsisdk.ResourceStatusEnabled,
		LifecycleState:    state,
	}
}

func sdkExternalHostInsightSummaryFromSpec(
	id string,
	spec opsiv1beta1.HostInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.MacsManagedExternalHostInsightSummary {
	return opsisdk.MacsManagedExternalHostInsightSummary{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		HostName:          common.String("host-alpha"),
		ManagementAgentId: common.String(spec.ManagementAgentId),
		FreeformTags:      hostInsightCloneStringMap(spec.FreeformTags),
		DefinedTags:       hostInsightDefinedTagsFromSpec(spec.DefinedTags),
		Status:            opsisdk.ResourceStatusEnabled,
		LifecycleState:    state,
	}
}

func sdkCloudDatabaseHostInsightSummaryFromSpec(
	id string,
	spec opsiv1beta1.HostInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.MacsManagedCloudDatabaseHostInsightSummary {
	return opsisdk.MacsManagedCloudDatabaseHostInsightSummary{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		HostName:          common.String("cloud-db-host"),
		ManagementAgentId: common.String(spec.ManagementAgentId),
		FreeformTags:      hostInsightCloneStringMap(spec.FreeformTags),
		DefinedTags:       hostInsightDefinedTagsFromSpec(spec.DefinedTags),
		Status:            opsisdk.ResourceStatusEnabled,
		LifecycleState:    state,
	}
}

func hostInsightWorkRequest(
	id string,
	operation opsisdk.OperationTypeEnum,
	status opsisdk.OperationStatusEnum,
	action opsisdk.ActionTypeEnum,
	hostInsightID string,
) opsisdk.WorkRequest {
	percentComplete := float32(50)
	workRequest := opsisdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if hostInsightID != "" {
		workRequest.Resources = []opsisdk.WorkRequestResource{
			{
				EntityType: common.String("HostInsight"),
				ActionType: action,
				Identifier: common.String(hostInsightID),
				EntityUri:  common.String("/20200630/hostInsights/" + hostInsightID),
			},
		}
	}
	return workRequest
}

func TestHostInsightRuntimeSemanticsEncodesWorkRequestAndDeleteContracts(t *testing.T) {
	t.Parallel()

	got := newHostInsightRuntimeSemantics()
	if got.FormalService != "opsi" || got.FormalSlug != "hostinsight" {
		t.Fatalf("formal identity = %s/%s, want opsi/hostinsight", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", got.Async)
	}
	assertHostInsightStrings(t, "work request phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v follow-up %#v, want required work request confirmation", got.Delete, got.DeleteFollowUp)
	}
	assertHostInsightStrings(t, "mutable fields", got.Mutation.Mutable, []string{"definedTags", "freeformTags"})
	assertHostInsightStrings(t, "force-new fields", got.Mutation.ForceNew, []string{
		"compartmentId",
		"entitySource",
		"computeId",
		"managementAgentId",
		"enterpriseManagerIdentifier",
		"enterpriseManagerBridgeId",
		"enterpriseManagerEntityIdentifier",
		"exadataInsightId",
	})
}

func TestHostInsightCreateUsesPolymorphicBodyAndWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newHostInsightResource()
	workRequestStatus := opsisdk.OperationStatusInProgress
	createCalls := 0
	getCalls := 0
	var createRequest opsisdk.CreateHostInsightRequest

	client := newHostInsightTestClient(&fakeHostInsightOCIClient{
		createFn: func(_ context.Context, request opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
			createCalls++
			createRequest = request
			return opsisdk.CreateHostInsightResponse{
				HostInsight:      sdkHostInsightFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireHostInsightStringPtr(t, "workRequestId", request.WorkRequestId, "wr-create")
			action := opsisdk.ActionTypeInProgress
			if workRequestStatus == opsisdk.OperationStatusSucceeded {
				action = opsisdk.ActionTypeCreated
			}
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: hostInsightWorkRequest("wr-create", opsisdk.OperationTypeCreateHostInsight, workRequestStatus, action, testHostInsightID),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			getCalls++
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkHostInsightFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireHostInsightNoError(t, "CreateOrUpdate()", err)
	requireHostInsightRequeueResponse(t, response, true, "while work request is pending")
	requireHostInsightCreateRequest(t, createRequest, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireHostInsightAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)

	workRequestStatus = opsisdk.OperationStatusSucceeded
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireHostInsightNoError(t, "CreateOrUpdate() after work request success", err)
	requireHostInsightRequeueResponse(t, response, false, "after completed create work request")
	requireHostInsightCallCount(t, "CreateHostInsight()", createCalls, 1)
	requireHostInsightCallCount(t, "GetHostInsight()", getCalls, 1)
	requireHostInsightIdentity(t, resource, testHostInsightID)
	if got := resource.Status.Status; got != string(opsisdk.ResourceStatusEnabled) {
		t.Fatalf("status.sdkStatus = %q, want %s", got, opsisdk.ResourceStatusEnabled)
	}
	requireHostInsightCondition(t, resource, shared.Active)
	requireHostInsightNoCurrentAsync(t, resource)
}

func TestHostInsightCreateRejectsJsonDataBeforeOCI(t *testing.T) {
	t.Parallel()

	resource := newHostInsightResource()
	resource.Spec.JsonData = `{"entitySource":"MACS_MANAGED_CLOUD_HOST","computeId":"ocid1.instance.oc1..fromjson"}`
	fake := &fakeHostInsightOCIClient{
		listFn: func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			t.Fatal("ListHostInsights() should not be called when spec.jsonData is rejected")
			return opsisdk.ListHostInsightsResponse{}, nil
		},
		createFn: func(context.Context, opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
			t.Fatal("CreateHostInsight() should not be called when spec.jsonData is rejected")
			return opsisdk.CreateHostInsightResponse{}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error) {
			t.Fatal("UpdateHostInsight() should not be called when spec.jsonData is rejected")
			return opsisdk.UpdateHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want spec.jsonData rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "spec.jsonData") {
		t.Fatalf("CreateOrUpdate() error = %v, want spec.jsonData detail", err)
	}
	requireHostInsightCallCount(t, "ListHostInsights()", len(fake.listRequests), 0)
	requireHostInsightCallCount(t, "CreateHostInsight()", len(fake.createRequests), 0)
	requireHostInsightCallCount(t, "UpdateHostInsight()", len(fake.updateRequests), 0)
}

func TestHostInsightCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := newHostInsightResource()
	otherSpec := resource.Spec
	otherSpec.ComputeId = "ocid1.instance.oc1..other"
	fake := &fakeHostInsightOCIClient{
		listFn: func(_ context.Context, request opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			requireHostInsightStringPtr(t, "list compartmentId", request.CompartmentId, testHostInsightCompID)
			switch hostInsightString(request.Page) {
			case "":
				return opsisdk.ListHostInsightsResponse{
					HostInsightSummaryCollection: opsisdk.HostInsightSummaryCollection{Items: []opsisdk.HostInsightSummary{
						sdkHostInsightSummaryFromSpec(testHostInsightOtherID, otherSpec, opsisdk.LifecycleStateActive),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return opsisdk.ListHostInsightsResponse{
					HostInsightSummaryCollection: opsisdk.HostInsightSummaryCollection{Items: []opsisdk.HostInsightSummary{
						sdkHostInsightSummaryFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive),
					}},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", hostInsightString(request.Page))
				return opsisdk.ListHostInsightsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkHostInsightFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
			t.Fatal("CreateHostInsight() should not be called when list resolves an existing HostInsight")
			return opsisdk.CreateHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireHostInsightNoError(t, "CreateOrUpdate()", err)
	requireHostInsightRequeueResponse(t, response, false, "after binding existing HostInsight")
	requireHostInsightCallCount(t, "ListHostInsights()", len(fake.listRequests), 2)
	requireHostInsightCallCount(t, "CreateHostInsight()", len(fake.createRequests), 0)
	requireHostInsightIdentity(t, resource, testHostInsightID)
}

func TestHostInsightCreateOrUpdateSkipsCrossSubtypeSummaryBeforeBindingExternalHost(t *testing.T) {
	t.Parallel()

	resource := newExternalHostInsightResource()
	fake := &fakeHostInsightOCIClient{
		listFn: func(_ context.Context, request opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			requireHostInsightStringPtr(t, "list compartmentId", request.CompartmentId, testHostInsightCompID)
			switch hostInsightString(request.Page) {
			case "":
				return opsisdk.ListHostInsightsResponse{
					HostInsightSummaryCollection: opsisdk.HostInsightSummaryCollection{Items: []opsisdk.HostInsightSummary{
						sdkCloudDatabaseHostInsightSummaryFromSpec(testHostInsightOtherID, resource.Spec, opsisdk.LifecycleStateActive),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return opsisdk.ListHostInsightsResponse{
					HostInsightSummaryCollection: opsisdk.HostInsightSummaryCollection{Items: []opsisdk.HostInsightSummary{
						sdkExternalHostInsightSummaryFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive),
					}},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", hostInsightString(request.Page))
				return opsisdk.ListHostInsightsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkExternalHostInsightFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
			t.Fatal("CreateHostInsight() should not be called when the intended external HostInsight exists")
			return opsisdk.CreateHostInsightResponse{}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error) {
			t.Fatal("UpdateHostInsight() should not be called for a cross-subtype bind candidate")
			return opsisdk.UpdateHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireHostInsightNoError(t, "CreateOrUpdate()", err)
	requireHostInsightRequeueResponse(t, response, false, "after binding external HostInsight")
	requireHostInsightCallCount(t, "ListHostInsights()", len(fake.listRequests), 2)
	requireHostInsightCallCount(t, "GetHostInsight()", len(fake.getRequests), 1)
	requireHostInsightCallCount(t, "CreateHostInsight()", len(fake.createRequests), 0)
	requireHostInsightCallCount(t, "UpdateHostInsight()", len(fake.updateRequests), 0)
	requireHostInsightIdentity(t, resource, testHostInsightID)
}

func TestHostInsightCreateOrUpdateSkipsUpdateWhenCurrentMatches(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	client := newHostInsightTestClient(&fakeHostInsightOCIClient{
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkHostInsightFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error) {
			t.Fatal("UpdateHostInsight() should not be called when desired and observed state match")
			return opsisdk.UpdateHostInsightResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireHostInsightNoError(t, "CreateOrUpdate()", err)
	requireHostInsightRequeueResponse(t, response, false, "for no-op reconcile")
}

func TestHostInsightCreateOrUpdateUpdatesMutableTagsAndCompletesWorkRequest(t *testing.T) {
	t.Parallel()

	original := newHostInsightResource()
	resource := newExistingHostInsightResource(testHostInsightID)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	getCalls := 0
	var updateRequest opsisdk.UpdateHostInsightRequest
	client := newHostInsightTestClient(&fakeHostInsightOCIClient{
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			getCalls++
			spec := original.Spec
			if getCalls > 1 {
				spec = resource.Spec
			}
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkHostInsightFromSpec(testHostInsightID, spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error) {
			updateRequest = request
			return opsisdk.UpdateHostInsightResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireHostInsightStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update")
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: hostInsightWorkRequest("wr-update", opsisdk.OperationTypeUpdateHostInsight, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeUpdated, testHostInsightID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireHostInsightNoError(t, "CreateOrUpdate()", err)
	requireHostInsightRequeueResponse(t, response, false, "after completed update work request")
	requireHostInsightStringPtr(t, "update hostInsightId", updateRequest.HostInsightId, testHostInsightID)
	details, ok := updateRequest.UpdateHostInsightDetails.(opsisdk.UpdateMacsManagedCloudHostInsightDetails)
	if !ok {
		t.Fatalf("update details type = %T, want UpdateMacsManagedCloudHostInsightDetails", updateRequest.UpdateHostInsightDetails)
	}
	if got := details.FreeformTags["env"]; got != "prod" {
		t.Fatalf("update freeformTags[env] = %q, want prod", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireHostInsightCondition(t, resource, shared.Active)
}

func TestHostInsightCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	original := newHostInsightResource()
	resource := newExistingHostInsightResource(testHostInsightID)
	resource.Spec.ComputeId = "ocid1.instance.oc1..different"
	fake := &fakeHostInsightOCIClient{
		getFn: func(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkHostInsightFromSpec(testHostInsightID, original.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateHostInsightRequest) (opsisdk.UpdateHostInsightResponse, error) {
			t.Fatal("UpdateHostInsight() should not be called when computeId drifts")
			return opsisdk.UpdateHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want computeId drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "computeId") {
		t.Fatalf("CreateOrUpdate() error = %v, want computeId detail", err)
	}
	requireHostInsightCallCount(t, "UpdateHostInsight()", len(fake.updateRequests), 0)
}

func TestHostInsightDeleteKeepsFinalizerWhileWorkRequestIsPending(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	client := newHostInsightTestClient(&fakeHostInsightOCIClient{
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkHostInsightFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "delete hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.DeleteHostInsightResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireHostInsightStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: hostInsightWorkRequest("wr-delete", opsisdk.OperationTypeDeleteHostInsight, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeInProgress, testHostInsightID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	requireHostInsightNoError(t, "Delete()", err)
	if deleted {
		t.Fatal("Delete() deleted = true, want false while work request is pending")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireHostInsightAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set while delete work request is still pending")
	}
}

func TestHostInsightDeleteTreatsPreReadAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	fake := &fakeHostInsightOCIClient{
		getFn: func(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			return opsisdk.GetHostInsightResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous pre-read")
		},
		deleteFn: func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			t.Fatal("DeleteHostInsight() should not be called after ambiguous pre-delete read")
			return opsisdk.DeleteHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-read to stay fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 0)
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Delete() error = %v, want ambiguous detail", err)
	}
}

func TestHostInsightDeleteTrackedStatusIDIgnoresUnsupportedJsonData(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	resource.Status.OsokStatus.Ocid = ""
	resource.Spec.ComputeId = ""
	resource.Spec.JsonData = `{"entitySource":"MACS_MANAGED_CLOUD_HOST","computeId":"ocid1.instance.oc1..fromjson"}`
	observedSpec := newHostInsightResource().Spec
	fake := &fakeHostInsightOCIClient{
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: sdkHostInsightFromSpec(testHostInsightID, observedSpec, opsisdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "delete hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.DeleteHostInsightResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		listFn: func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			t.Fatal("ListHostInsights() should not be called when status.id records the delete target")
			return opsisdk.ListHostInsightsResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireHostInsightNoError(t, "Delete()", err)
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	requireHostInsightCallCount(t, "GetHostInsight()", len(fake.getRequests), 1)
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 1)
	requireHostInsightCallCount(t, "ListHostInsights()", len(fake.listRequests), 0)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireHostInsightAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	requireHostInsightCondition(t, resource, shared.Terminating)
}

func TestHostInsightDeleteTrackedOCIDResumesWorkRequestWithUnsupportedJsonData(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	resource.Spec.JsonData = `{"entitySource":"MACS_MANAGED_CLOUD_HOST","computeId":"ocid1.instance.oc1..fromjson"}`
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeHostInsightOCIClient{
		workRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireHostInsightStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: hostInsightWorkRequest("wr-delete", opsisdk.OperationTypeDeleteHostInsight, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testHostInsightID),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			t.Fatal("DeleteHostInsight() should not be called while resuming the tracked delete work request")
			return opsisdk.DeleteHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireHostInsightNoError(t, "Delete()", err)
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after completed work request and NotFound readback")
	}
	requireHostInsightCallCount(t, "GetWorkRequest()", len(fake.workRequestRequests), 1)
	requireHostInsightCallCount(t, "GetHostInsight()", len(fake.getRequests), 1)
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want timestamp after confirmed deletion")
	}
	requireHostInsightNoCurrentAsync(t, resource)
}

func TestHostInsightDeleteNoTrackedIDUnsupportedJsonDataMarksDeleted(t *testing.T) {
	t.Parallel()

	resource := newHostInsightResource()
	resource.Spec.JsonData = `{"entitySource":"MACS_MANAGED_CLOUD_HOST","computeId":"ocid1.instance.oc1..fromjson"}`
	fake := &fakeHostInsightOCIClient{
		getFn: func(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			t.Fatal("GetHostInsight() should not be called when unsupported spec.jsonData has no tracked OCID")
			return opsisdk.GetHostInsightResponse{}, nil
		},
		listFn: func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			t.Fatal("ListHostInsights() should not be called when unsupported spec.jsonData has no tracked OCID")
			return opsisdk.ListHostInsightsResponse{}, nil
		},
		deleteFn: func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			t.Fatal("DeleteHostInsight() should not be called without a tracked OCID")
			return opsisdk.DeleteHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireHostInsightNoError(t, "Delete()", err)
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when unsupported spec.jsonData has no tracked OCID")
	}
	requireHostInsightCallCount(t, "GetHostInsight()", len(fake.getRequests), 0)
	requireHostInsightCallCount(t, "ListHostInsights()", len(fake.listRequests), 0)
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want timestamp after no-target cleanup")
	}
	requireHostInsightCondition(t, resource, shared.Terminating)
}

func TestHostInsightDeleteNoTrackedIDListMissMarksDeleted(t *testing.T) {
	t.Parallel()

	resource := newHostInsightResource()
	otherSpec := resource.Spec
	otherSpec.ComputeId = "ocid1.instance.oc1..other"
	fake := &fakeHostInsightOCIClient{
		listFn: func(_ context.Context, request opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			requireHostInsightStringPtr(t, "list compartmentId", request.CompartmentId, testHostInsightCompID)
			switch hostInsightString(request.Page) {
			case "":
				return opsisdk.ListHostInsightsResponse{
					HostInsightSummaryCollection: opsisdk.HostInsightSummaryCollection{Items: []opsisdk.HostInsightSummary{
						sdkHostInsightSummaryFromSpec(testHostInsightOtherID, otherSpec, opsisdk.LifecycleStateActive),
					}},
					OpcNextPage:  common.String("page-2"),
					OpcRequestId: common.String("opc-list"),
				}, nil
			case "page-2":
				return opsisdk.ListHostInsightsResponse{
					HostInsightSummaryCollection: opsisdk.HostInsightSummaryCollection{Items: []opsisdk.HostInsightSummary{}},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", hostInsightString(request.Page))
				return opsisdk.ListHostInsightsResponse{}, nil
			}
		},
		getFn: func(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			t.Fatal("GetHostInsight() should not be called when no tracked OCID exists")
			return opsisdk.GetHostInsightResponse{}, nil
		},
		deleteFn: func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			t.Fatal("DeleteHostInsight() should not be called when list miss confirms deletion")
			return opsisdk.DeleteHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireHostInsightNoError(t, "Delete()", err)
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after paginated list miss")
	}
	requireHostInsightCallCount(t, "ListHostInsights()", len(fake.listRequests), 2)
	requireHostInsightCallCount(t, "GetHostInsight()", len(fake.getRequests), 0)
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want timestamp after list miss confirms deletion")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-list" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-list", got)
	}
	requireHostInsightCondition(t, resource, shared.Terminating)
}

func TestHostInsightDeleteTreatsSucceededWorkRequestAuthShapedReadbackAsAmbiguous(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeHostInsightOCIClient{
		workRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireHostInsightStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: hostInsightWorkRequest("wr-delete", opsisdk.OperationTypeDeleteHostInsight, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testHostInsightID),
			}, nil
		},
		getFn: func(context.Context, opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			return opsisdk.GetHostInsightResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous readback")
		},
		deleteFn: func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			t.Fatal("DeleteHostInsight() should not be called while resuming a delete work request")
			return opsisdk.DeleteHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous readback to stay fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 0)
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Delete() error = %v, want ambiguous detail", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set after ambiguous delete readback")
	}
}

func TestHostInsightDeleteConfirmsSucceededWorkRequestWithUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeHostInsightOCIClient{
		workRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireHostInsightStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: hostInsightWorkRequest("wr-delete", opsisdk.OperationTypeDeleteHostInsight, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testHostInsightID),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		listFn: func(context.Context, opsisdk.ListHostInsightsRequest) (opsisdk.ListHostInsightsResponse, error) {
			t.Fatal("ListHostInsights() should not be called after unambiguous NotFound confirms delete")
			return opsisdk.ListHostInsightsResponse{}, nil
		},
		deleteFn: func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			t.Fatal("DeleteHostInsight() should not be called while resuming a completed delete work request")
			return opsisdk.DeleteHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireHostInsightNoError(t, "Delete()", err)
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after completed work request and NotFound readback")
	}
	requireHostInsightCallCount(t, "GetWorkRequest()", len(fake.workRequestRequests), 1)
	requireHostInsightCallCount(t, "GetHostInsight()", len(fake.getRequests), 1)
	requireHostInsightCallCount(t, "ListHostInsights()", len(fake.listRequests), 0)
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want timestamp after confirmed deletion")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after confirmed deletion", resource.Status.OsokStatus.Async.Current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireHostInsightCondition(t, resource, shared.Terminating)
}

func TestHostInsightDeleteConfirmsSucceededWorkRequestWithTerminatedSDKStatus(t *testing.T) {
	t.Parallel()

	resource := newExistingHostInsightResource(testHostInsightID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	terminated := sdkHostInsightFromSpec(testHostInsightID, resource.Spec, opsisdk.LifecycleStateActive)
	terminated.Status = opsisdk.ResourceStatusTerminated
	fake := &fakeHostInsightOCIClient{
		workRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireHostInsightStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: hostInsightWorkRequest("wr-delete", opsisdk.OperationTypeDeleteHostInsight, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, testHostInsightID),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetHostInsightRequest) (opsisdk.GetHostInsightResponse, error) {
			requireHostInsightStringPtr(t, "get hostInsightId", request.HostInsightId, testHostInsightID)
			return opsisdk.GetHostInsightResponse{
				HostInsight: hostInsightWithSDKStatusAlias(terminated),
			}, nil
		},
		deleteFn: func(context.Context, opsisdk.DeleteHostInsightRequest) (opsisdk.DeleteHostInsightResponse, error) {
			t.Fatal("DeleteHostInsight() should not be called after TERMINATED sdkStatus confirms deletion")
			return opsisdk.DeleteHostInsightResponse{}, nil
		},
	}
	client := newHostInsightTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	requireHostInsightNoError(t, "Delete()", err)
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after completed work request and TERMINATED sdkStatus readback")
	}
	requireHostInsightCallCount(t, "GetWorkRequest()", len(fake.workRequestRequests), 1)
	requireHostInsightCallCount(t, "GetHostInsight()", len(fake.getRequests), 1)
	requireHostInsightCallCount(t, "DeleteHostInsight()", len(fake.deleteRequests), 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want timestamp after TERMINATED sdkStatus readback")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after TERMINATED sdkStatus readback", resource.Status.OsokStatus.Async.Current)
	}
	requireHostInsightCondition(t, resource, shared.Terminating)
}

func TestHostInsightCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := newHostInsightResource()
	client := newHostInsightTestClient(&fakeHostInsightOCIClient{
		createFn: func(context.Context, opsisdk.CreateHostInsightRequest) (opsisdk.CreateHostInsightResponse, error) {
			return opsisdk.CreateHostInsightResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "conflict")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireHostInsightCondition(t, resource, shared.Failed)
}

func requireHostInsightNoError(t *testing.T, label string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s error = %v", label, err)
	}
}

func requireHostInsightRequeueResponse(
	t *testing.T,
	response servicemanager.OSOKResponse,
	shouldRequeue bool,
	context string,
) {
	t.Helper()

	if !response.IsSuccessful || response.ShouldRequeue != shouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue=%t %s", response, shouldRequeue, context)
	}
}

func requireHostInsightCreateRequest(
	t *testing.T,
	request opsisdk.CreateHostInsightRequest,
	resource *opsiv1beta1.HostInsight,
) {
	t.Helper()

	details, ok := request.CreateHostInsightDetails.(opsisdk.CreateMacsManagedCloudHostInsightDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateMacsManagedCloudHostInsightDetails", request.CreateHostInsightDetails)
	}
	requireHostInsightStringPtr(t, "create compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireHostInsightStringPtr(t, "create computeId", details.ComputeId, resource.Spec.ComputeId)
	if got := details.FreeformTags["env"]; got != "dev" {
		t.Fatalf("create freeformTags[env] = %q, want dev", got)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) != string(resource.UID) {
		t.Fatalf("create retry token = %v, want resource UID", request.OpcRetryToken)
	}
}

func requireHostInsightIdentity(t *testing.T, resource *opsiv1beta1.HostInsight, want string) {
	t.Helper()

	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func requireHostInsightAsync(
	t *testing.T,
	resource *opsiv1beta1.HostInsight,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Phase != phase || current.WorkRequestID != workRequestID || current.NormalizedClass != class {
		t.Fatalf("status.async.current = %#v, want phase=%q workRequestID=%q class=%q", current, phase, workRequestID, class)
	}
}

func requireHostInsightNoCurrentAsync(t *testing.T, resource *opsiv1beta1.HostInsight) {
	t.Helper()

	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after completed work request", resource.Status.OsokStatus.Async.Current)
	}
}

func requireHostInsightCondition(
	t *testing.T,
	resource *opsiv1beta1.HostInsight,
	want shared.OSOKConditionType,
) {
	t.Helper()

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions is empty, want %q", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireHostInsightStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if strings.TrimSpace(*got) != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireHostInsightCallCount(t *testing.T, label string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", label, got, want)
	}
}

func assertHostInsightStrings(t *testing.T, label string, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d (%v)", label, len(got), len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q (got %v)", label, i, got[i], want[i], got)
		}
	}
}
