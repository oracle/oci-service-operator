/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementstation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testManagementStationID            = "ocid1.managementstation.oc1..station"
	testManagementStationCompartmentID = "ocid1.compartment.oc1..station"
)

func TestManagementStationRuntimeHooksConfigured(t *testing.T) {
	hooks := newManagementStationDefaultRuntimeHooks(osmanagementhubsdk.ManagementStationClient{})
	applyManagementStationRuntimeHooks(nil, &hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want bool-preserving create builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update builder")
	}
	if hooks.DeleteHooks.ConfirmRead == nil {
		t.Fatal("hooks.DeleteHooks.ConfirmRead = nil, want conservative delete confirm read")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
	if hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("hooks.DeleteHooks.ApplyOutcome = nil, want auth-shaped confirm read guard")
	}

	body, err := hooks.BuildCreateBody(context.Background(), testManagementStationResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(osmanagementhubsdk.CreateManagementStationDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want osmanagementhub.CreateManagementStationDetails", body)
	}
	assertManagementStationBoolPointerFalse(t, "BuildCreateBody() isAutoConfigEnabled", details.IsAutoConfigEnabled)
	assertManagementStationBoolPointerFalse(t, "BuildCreateBody() proxy.isEnabled", details.Proxy.IsEnabled)
	assertManagementStationBoolPointerFalse(t, "BuildCreateBody() mirror.isSslverifyEnabled", details.Mirror.IsSslverifyEnabled)
}

func TestManagementStationCreateRecordsIdentityAndRequestID(t *testing.T) {
	resource := testManagementStationResource()
	station := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	fake := &fakeManagementStationOCIClient{
		listManagementStations: func(context.Context, osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error) {
			return osmanagementhubsdk.ListManagementStationsResponse{}, nil
		},
		createManagementStation: func(_ context.Context, request osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error) {
			assertManagementStationCreateRequest(t, request, resource)
			return osmanagementhubsdk.CreateManagementStationResponse{
				ManagementStation: station,
				OpcRequestId:      common.String("opc-create"),
			}, nil
		},
		getManagementStation: func(_ context.Context, request osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			requireManagementStationStringPtr(t, "GetManagementStation() id", request.ManagementStationId, testManagementStationID)
			return osmanagementhubsdk.GetManagementStationResponse{ManagementStation: station}, nil
		},
	}

	response, err := newTestManagementStationClient(fake).CreateOrUpdate(context.Background(), resource, testManagementStationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulManagementStationResponse(t, response)
	assertManagementStationCallCount(t, "CreateManagementStation()", fake.createCalls, 1)
	assertManagementStationRecordedID(t, resource, testManagementStationID)
	assertManagementStationOpcRequestID(t, resource, "opc-create")
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after ACTIVE follow-up", resource.Status.OsokStatus.Async.Current)
	}
	if got := resource.Status.SystemTags["orcl-cloud"]["free-tier-retained"]; got != "true" {
		t.Fatalf("status.systemTags free-tier-retained = %q, want normalized string true", got)
	}
}

func TestManagementStationCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := testManagementStationResource()
	station := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	var listPages []string
	fake := &fakeManagementStationOCIClient{
		listManagementStations: func(_ context.Context, request osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error) {
			listPages = append(listPages, managementStationStringValue(request.Page))
			if request.Page == nil {
				return osmanagementhubsdk.ListManagementStationsResponse{
					ManagementStationCollection: osmanagementhubsdk.ManagementStationCollection{
						Items: []osmanagementhubsdk.ManagementStationSummary{
							sdkManagementStationSummaryFromResource(resource, "ocid1.managementstation.oc1..other", "other-station", resource.Spec.Hostname),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return osmanagementhubsdk.ListManagementStationsResponse{
				ManagementStationCollection: osmanagementhubsdk.ManagementStationCollection{
					Items: []osmanagementhubsdk.ManagementStationSummary{
						sdkManagementStationSummaryFromResource(resource, testManagementStationID, resource.Spec.DisplayName, resource.Spec.Hostname),
					},
				},
			}, nil
		},
		getManagementStation: func(_ context.Context, request osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			requireManagementStationStringPtr(t, "GetManagementStation() id", request.ManagementStationId, testManagementStationID)
			return osmanagementhubsdk.GetManagementStationResponse{ManagementStation: station}, nil
		},
	}

	response, err := newTestManagementStationClient(fake).CreateOrUpdate(context.Background(), resource, testManagementStationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulManagementStationResponse(t, response)
	assertManagementStationCallCount(t, "CreateManagementStation()", fake.createCalls, 0)
	if got := strings.Join(listPages, ","); got != ",page-2" {
		t.Fatalf("ListManagementStations() pages = %q, want \",page-2\"", got)
	}
	assertManagementStationRecordedID(t, resource, testManagementStationID)
}

func TestManagementStationCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := testManagementStationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementStationID)
	station := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	fake := &fakeManagementStationOCIClient{
		getManagementStation: func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			return osmanagementhubsdk.GetManagementStationResponse{ManagementStation: station}, nil
		},
	}

	response, err := newTestManagementStationClient(fake).CreateOrUpdate(context.Background(), resource, testManagementStationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulManagementStationResponse(t, response)
	assertManagementStationCallCount(t, "UpdateManagementStation()", fake.updateCalls, 0)
}

func TestManagementStationMutableUpdateUsesUpdatePath(t *testing.T) {
	resource := testManagementStationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementStationID)
	current := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	current.Description = common.String("old description")
	current.Proxy.IsEnabled = common.Bool(true)
	updated := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	getResponses := []osmanagementhubsdk.GetManagementStationResponse{
		{ManagementStation: current},
		{ManagementStation: updated},
	}
	fake := &fakeManagementStationOCIClient{
		getManagementStation: managementStationGetResponses(t, &getResponses),
		updateManagementStation: func(_ context.Context, request osmanagementhubsdk.UpdateManagementStationRequest) (osmanagementhubsdk.UpdateManagementStationResponse, error) {
			assertManagementStationUpdateRequest(t, request, resource)
			return osmanagementhubsdk.UpdateManagementStationResponse{
				ManagementStation: updated,
				OpcRequestId:      common.String("opc-update"),
			}, nil
		},
	}

	response, err := newTestManagementStationClient(fake).CreateOrUpdate(context.Background(), resource, testManagementStationRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertSuccessfulManagementStationResponse(t, response)
	assertManagementStationCallCount(t, "UpdateManagementStation()", fake.updateCalls, 1)
	assertManagementStationOpcRequestID(t, resource, "opc-update")
}

func TestManagementStationRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := testManagementStationResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementStationID)
	current := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	current.CompartmentId = common.String(testManagementStationCompartmentID)
	fake := &fakeManagementStationOCIClient{
		getManagementStation: func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			return osmanagementhubsdk.GetManagementStationResponse{ManagementStation: current}, nil
		},
	}

	response, err := newTestManagementStationClient(fake).CreateOrUpdate(context.Background(), resource, testManagementStationRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId context", err.Error())
	}
	assertManagementStationCallCount(t, "UpdateManagementStation()", fake.updateCalls, 0)
}

func TestManagementStationDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := testManagementStationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementStationID)
	active := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	notFoundErr := errortest.NewServiceError(404, errorutil.NotFound, "management station not found")
	getCalls := 0
	fake := &fakeManagementStationOCIClient{
		getManagementStation: func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			getCalls++
			if getCalls == 1 {
				return osmanagementhubsdk.GetManagementStationResponse{ManagementStation: active}, nil
			}
			return osmanagementhubsdk.GetManagementStationResponse{}, notFoundErr
		},
		deleteManagementStation: func(context.Context, osmanagementhubsdk.DeleteManagementStationRequest) (osmanagementhubsdk.DeleteManagementStationResponse, error) {
			return osmanagementhubsdk.DeleteManagementStationResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestManagementStationClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found confirmation")
	}
	assertManagementStationCallCount(t, "DeleteManagementStation()", fake.deleteCalls, 1)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
	assertManagementStationOpcRequestID(t, resource, notFoundErr.GetOpcRequestID())
}

func TestManagementStationDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := testManagementStationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementStationID)
	active := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	deleting := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateDeleting)
	getResponses := []osmanagementhubsdk.GetManagementStationResponse{
		{ManagementStation: active},
		{ManagementStation: deleting},
	}
	fake := &fakeManagementStationOCIClient{
		getManagementStation: managementStationGetResponses(t, &getResponses),
		deleteManagementStation: func(context.Context, osmanagementhubsdk.DeleteManagementStationRequest) (osmanagementhubsdk.DeleteManagementStationResponse, error) {
			return osmanagementhubsdk.DeleteManagementStationResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestManagementStationClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while lifecycle is DELETING")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle delete tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current.phase = %q, want delete", got)
	}
}

func TestManagementStationDeleteWithoutRecordedIDResolvesExistingIdentity(t *testing.T) {
	resource := testManagementStationResource()
	station := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	getCalls := 0
	fake := &fakeManagementStationOCIClient{
		listManagementStations: func(context.Context, osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error) {
			return osmanagementhubsdk.ListManagementStationsResponse{
				ManagementStationCollection: osmanagementhubsdk.ManagementStationCollection{
					Items: []osmanagementhubsdk.ManagementStationSummary{
						sdkManagementStationSummaryFromResource(resource, testManagementStationID, resource.Spec.DisplayName, resource.Spec.Hostname),
					},
				},
			}, nil
		},
		getManagementStation: func(_ context.Context, request osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			requireManagementStationStringPtr(t, "GetManagementStation() id", request.ManagementStationId, testManagementStationID)
			getCalls++
			if getCalls == 1 {
				return osmanagementhubsdk.GetManagementStationResponse{ManagementStation: station}, nil
			}
			return osmanagementhubsdk.GetManagementStationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "management station not found")
		},
		deleteManagementStation: func(_ context.Context, request osmanagementhubsdk.DeleteManagementStationRequest) (osmanagementhubsdk.DeleteManagementStationResponse, error) {
			requireManagementStationStringPtr(t, "DeleteManagementStation() id", request.ManagementStationId, testManagementStationID)
			return osmanagementhubsdk.DeleteManagementStationResponse{}, nil
		},
	}

	deleted, err := newTestManagementStationClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after resolved identity delete confirmation")
	}
	assertManagementStationRecordedID(t, resource, testManagementStationID)
	assertManagementStationCallCount(t, "DeleteManagementStation()", fake.deleteCalls, 1)
}

func TestManagementStationDeleteStopsBeforeDeleteOnAuthShapedConfirmRead(t *testing.T) {
	resource := testManagementStationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementStationID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeManagementStationOCIClient{
		getManagementStation: func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			return osmanagementhubsdk.GetManagementStationResponse{}, authErr
		},
	}

	deleted, err := newTestManagementStationClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	assertManagementStationCallCount(t, "DeleteManagementStation()", fake.deleteCalls, 0)
	if got := resource.Status.OsokStatus.OpcRequestID; got != authErr.GetOpcRequestID() {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, authErr.GetOpcRequestID())
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound context", err.Error())
	}
}

func TestManagementStationDeleteTreatsAuthShapedDeleteErrorAsAmbiguous(t *testing.T) {
	resource := testManagementStationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagementStationID)
	active := sdkManagementStationFromResource(resource, testManagementStationID, osmanagementhubsdk.ManagementStationLifecycleStateActive)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeManagementStationOCIClient{
		getManagementStation: func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
			return osmanagementhubsdk.GetManagementStationResponse{ManagementStation: active}, nil
		},
		deleteManagementStation: func(context.Context, osmanagementhubsdk.DeleteManagementStationRequest) (osmanagementhubsdk.DeleteManagementStationResponse, error) {
			return osmanagementhubsdk.DeleteManagementStationResponse{}, authErr
		},
	}

	deleted, err := newTestManagementStationClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped delete rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	assertManagementStationCallCount(t, "DeleteManagementStation()", fake.deleteCalls, 1)
	if got := resource.Status.OsokStatus.OpcRequestID; got != authErr.GetOpcRequestID() {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, authErr.GetOpcRequestID())
	}
}

func TestManagementStationCreateRecordsOCIErrorRequestID(t *testing.T) {
	resource := testManagementStationResource()
	createErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	fake := &fakeManagementStationOCIClient{
		listManagementStations: func(context.Context, osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error) {
			return osmanagementhubsdk.ListManagementStationsResponse{}, nil
		},
		createManagementStation: func(context.Context, osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error) {
			return osmanagementhubsdk.CreateManagementStationResponse{}, createErr
		},
	}

	response, err := newTestManagementStationClient(fake).CreateOrUpdate(context.Background(), resource, testManagementStationRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != createErr.GetOpcRequestID() {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, createErr.GetOpcRequestID())
	}
}

type fakeManagementStationOCIClient struct {
	createManagementStation func(context.Context, osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error)
	getManagementStation    func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error)
	listManagementStations  func(context.Context, osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error)
	updateManagementStation func(context.Context, osmanagementhubsdk.UpdateManagementStationRequest) (osmanagementhubsdk.UpdateManagementStationResponse, error)
	deleteManagementStation func(context.Context, osmanagementhubsdk.DeleteManagementStationRequest) (osmanagementhubsdk.DeleteManagementStationResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeManagementStationOCIClient) CreateManagementStation(
	ctx context.Context,
	request osmanagementhubsdk.CreateManagementStationRequest,
) (osmanagementhubsdk.CreateManagementStationResponse, error) {
	f.createCalls++
	if f.createManagementStation == nil {
		return osmanagementhubsdk.CreateManagementStationResponse{}, fmt.Errorf("unexpected CreateManagementStation call")
	}
	return f.createManagementStation(ctx, request)
}

func (f *fakeManagementStationOCIClient) GetManagementStation(
	ctx context.Context,
	request osmanagementhubsdk.GetManagementStationRequest,
) (osmanagementhubsdk.GetManagementStationResponse, error) {
	f.getCalls++
	if f.getManagementStation == nil {
		return osmanagementhubsdk.GetManagementStationResponse{}, fmt.Errorf("unexpected GetManagementStation call")
	}
	return f.getManagementStation(ctx, request)
}

func (f *fakeManagementStationOCIClient) ListManagementStations(
	ctx context.Context,
	request osmanagementhubsdk.ListManagementStationsRequest,
) (osmanagementhubsdk.ListManagementStationsResponse, error) {
	f.listCalls++
	if f.listManagementStations == nil {
		return osmanagementhubsdk.ListManagementStationsResponse{}, fmt.Errorf("unexpected ListManagementStations call")
	}
	return f.listManagementStations(ctx, request)
}

func (f *fakeManagementStationOCIClient) UpdateManagementStation(
	ctx context.Context,
	request osmanagementhubsdk.UpdateManagementStationRequest,
) (osmanagementhubsdk.UpdateManagementStationResponse, error) {
	f.updateCalls++
	if f.updateManagementStation == nil {
		return osmanagementhubsdk.UpdateManagementStationResponse{}, fmt.Errorf("unexpected UpdateManagementStation call")
	}
	return f.updateManagementStation(ctx, request)
}

func (f *fakeManagementStationOCIClient) DeleteManagementStation(
	ctx context.Context,
	request osmanagementhubsdk.DeleteManagementStationRequest,
) (osmanagementhubsdk.DeleteManagementStationResponse, error) {
	f.deleteCalls++
	if f.deleteManagementStation == nil {
		return osmanagementhubsdk.DeleteManagementStationResponse{}, fmt.Errorf("unexpected DeleteManagementStation call")
	}
	return f.deleteManagementStation(ctx, request)
}

func newTestManagementStationClient(client *fakeManagementStationOCIClient) ManagementStationServiceClient {
	manager := &ManagementStationServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newManagementStationRuntimeHooksWithOCIClient(client)
	applyManagementStationRuntimeHooks(manager, &hooks)
	delegate := defaultManagementStationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.ManagementStation](
			buildManagementStationGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapManagementStationGeneratedClient(hooks, delegate)
}

func newManagementStationRuntimeHooksWithOCIClient(client managementStationOCIClient) ManagementStationRuntimeHooks {
	return ManagementStationRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*osmanagementhubv1beta1.ManagementStation]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*osmanagementhubv1beta1.ManagementStation]{},
		StatusHooks:     generatedruntime.StatusHooks[*osmanagementhubv1beta1.ManagementStation]{},
		ParityHooks:     generatedruntime.ParityHooks[*osmanagementhubv1beta1.ManagementStation]{},
		Async:           generatedruntime.AsyncHooks[*osmanagementhubv1beta1.ManagementStation]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*osmanagementhubv1beta1.ManagementStation]{},
		Create: runtimeOperationHooks[osmanagementhubsdk.CreateManagementStationRequest, osmanagementhubsdk.CreateManagementStationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateManagementStationDetails", RequestName: "CreateManagementStationDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error) {
				return client.CreateManagementStation(ctx, request)
			},
		},
		Get: runtimeOperationHooks[osmanagementhubsdk.GetManagementStationRequest, osmanagementhubsdk.GetManagementStationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ManagementStationId", RequestName: "managementStationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
				return client.GetManagementStation(ctx, request)
			},
		},
		List: runtimeOperationHooks[osmanagementhubsdk.ListManagementStationsRequest, osmanagementhubsdk.ListManagementStationsResponse]{
			Fields: managementStationListFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error) {
				return client.ListManagementStations(ctx, request)
			},
		},
		Update: runtimeOperationHooks[osmanagementhubsdk.UpdateManagementStationRequest, osmanagementhubsdk.UpdateManagementStationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ManagementStationId", RequestName: "managementStationId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateManagementStationDetails", RequestName: "UpdateManagementStationDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.UpdateManagementStationRequest) (osmanagementhubsdk.UpdateManagementStationResponse, error) {
				return client.UpdateManagementStation(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[osmanagementhubsdk.DeleteManagementStationRequest, osmanagementhubsdk.DeleteManagementStationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ManagementStationId", RequestName: "managementStationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.DeleteManagementStationRequest) (osmanagementhubsdk.DeleteManagementStationResponse, error) {
				return client.DeleteManagementStation(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ManagementStationServiceClient) ManagementStationServiceClient{},
	}
}

func testManagementStationRequest(resource *osmanagementhubv1beta1.ManagementStation) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func testManagementStationResource() *osmanagementhubv1beta1.ManagementStation {
	return &osmanagementhubv1beta1.ManagementStation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "station",
			Namespace: "default",
			UID:       types.UID("station-uid"),
		},
		Spec: osmanagementhubv1beta1.ManagementStationSpec{
			CompartmentId: testManagementStationCompartmentID,
			DisplayName:   "station",
			Hostname:      "station.example.com",
			Proxy: osmanagementhubv1beta1.ManagementStationProxy{
				IsEnabled: false,
				Hosts:     []string{"proxy.internal"},
				Port:      "3128",
				Forward:   "https://updates.example.com",
			},
			Mirror: osmanagementhubv1beta1.ManagementStationMirror{
				Directory:          "/var/lib/osmh",
				Port:               "8080",
				Sslport:            "8443",
				Sslcert:            "/etc/pki/osmh.crt",
				IsSslverifyEnabled: false,
			},
			Description:         "desired description",
			IsAutoConfigEnabled: false,
			FreeformTags:        map[string]string{"owner": "osok"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func sdkManagementStationFromResource(
	resource *osmanagementhubv1beta1.ManagementStation,
	id string,
	state osmanagementhubsdk.ManagementStationLifecycleStateEnum,
) osmanagementhubsdk.ManagementStation {
	spec := resource.Spec
	return osmanagementhubsdk.ManagementStation{
		Id:                     common.String(id),
		CompartmentId:          common.String(spec.CompartmentId),
		DisplayName:            common.String(spec.DisplayName),
		Hostname:               common.String(spec.Hostname),
		Proxy:                  sdkManagementStationProxy(spec.Proxy),
		Mirror:                 sdkManagementStationMirror(spec.Mirror),
		PeerManagementStations: []osmanagementhubsdk.PeerManagementStation{},
		Description:            common.String(spec.Description),
		LifecycleState:         state,
		IsAutoConfigEnabled:    common.Bool(spec.IsAutoConfigEnabled),
		FreeformTags:           cloneManagementStationStringMap(spec.FreeformTags),
		DefinedTags:            managementStationDefinedTags(spec.DefinedTags),
		SystemTags:             map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": true}},
	}
}

func sdkManagementStationSummaryFromResource(
	resource *osmanagementhubv1beta1.ManagementStation,
	id string,
	displayName string,
	hostname string,
) osmanagementhubsdk.ManagementStationSummary {
	spec := resource.Spec
	return osmanagementhubsdk.ManagementStationSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(displayName),
		Hostname:       common.String(hostname),
		Description:    common.String(spec.Description),
		LifecycleState: osmanagementhubsdk.ManagementStationLifecycleStateActive,
		FreeformTags:   cloneManagementStationStringMap(spec.FreeformTags),
		DefinedTags:    managementStationDefinedTags(spec.DefinedTags),
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": true}},
	}
}

func sdkManagementStationProxy(proxy osmanagementhubv1beta1.ManagementStationProxy) *osmanagementhubsdk.ProxyConfiguration {
	return &osmanagementhubsdk.ProxyConfiguration{
		IsEnabled: common.Bool(proxy.IsEnabled),
		Hosts:     append([]string(nil), proxy.Hosts...),
		Port:      common.String(proxy.Port),
		Forward:   common.String(proxy.Forward),
	}
}

func sdkManagementStationMirror(mirror osmanagementhubv1beta1.ManagementStationMirror) *osmanagementhubsdk.MirrorConfiguration {
	return &osmanagementhubsdk.MirrorConfiguration{
		Directory:          common.String(mirror.Directory),
		Port:               common.String(mirror.Port),
		Sslport:            common.String(mirror.Sslport),
		Sslcert:            common.String(mirror.Sslcert),
		IsSslverifyEnabled: common.Bool(mirror.IsSslverifyEnabled),
	}
}

func managementStationGetResponses(
	t *testing.T,
	responses *[]osmanagementhubsdk.GetManagementStationResponse,
) func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
	t.Helper()

	return func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
		if len(*responses) == 0 {
			t.Fatal("GetManagementStation() called more times than expected")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func assertManagementStationCreateRequest(
	t *testing.T,
	request osmanagementhubsdk.CreateManagementStationRequest,
	resource *osmanagementhubv1beta1.ManagementStation,
) {
	t.Helper()

	requireManagementStationStringPtr(t, "CreateManagementStation() compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireManagementStationStringPtr(t, "CreateManagementStation() displayName", request.DisplayName, resource.Spec.DisplayName)
	requireManagementStationStringPtr(t, "CreateManagementStation() hostname", request.Hostname, resource.Spec.Hostname)
	assertManagementStationBoolPointerFalse(t, "CreateManagementStation() isAutoConfigEnabled", request.IsAutoConfigEnabled)
	assertManagementStationBoolPointerFalse(t, "CreateManagementStation() proxy.isEnabled", request.Proxy.IsEnabled)
	assertManagementStationBoolPointerFalse(t, "CreateManagementStation() mirror.isSslverifyEnabled", request.Mirror.IsSslverifyEnabled)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateManagementStation() OpcRetryToken is empty, want deterministic retry token")
	}
}

func assertManagementStationUpdateRequest(
	t *testing.T,
	request osmanagementhubsdk.UpdateManagementStationRequest,
	resource *osmanagementhubv1beta1.ManagementStation,
) {
	t.Helper()

	requireManagementStationStringPtr(t, "UpdateManagementStation() id", request.ManagementStationId, testManagementStationID)
	requireManagementStationStringPtr(t, "UpdateManagementStation() description", request.Description, resource.Spec.Description)
	assertManagementStationBoolPointerFalse(t, "UpdateManagementStation() proxy.isEnabled", request.Proxy.IsEnabled)
}

func assertSuccessfulManagementStationResponse(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
}

func assertManagementStationCallCount(t *testing.T, operation string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", operation, got, want)
	}
}

func assertManagementStationRecordedID(
	t *testing.T,
	resource *osmanagementhubv1beta1.ManagementStation,
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

func assertManagementStationOpcRequestID(
	t *testing.T,
	resource *osmanagementhubv1beta1.ManagementStation,
	want string,
) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertManagementStationBoolPointerFalse(t *testing.T, label string, value *bool) {
	t.Helper()

	if value == nil || *value {
		t.Fatalf("%s = %#v, want explicit false", label, value)
	}
}

func requireManagementStationStringPtr(t *testing.T, label string, value *string, want string) {
	t.Helper()

	if got := managementStationStringValue(value); got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}
