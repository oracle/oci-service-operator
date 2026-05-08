/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package organization

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	tenantmanagercontrolplanesdk "github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane"
	tenantmanagercontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/tenantmanagercontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOrganizationID          = "ocid1.organization.oc1..example"
	testOtherOrganizationID     = "ocid1.organization.oc1..other"
	testOrganizationCompartment = "ocid1.tenancy.oc1..example"
	testOldSubscriptionID       = "ocid1.subscription.oc1..old"
	testNewSubscriptionID       = "ocid1.subscription.oc1..new"
)

type fakeOrganizationRuntimeOCIClient struct {
	getFn         func(context.Context, tenantmanagercontrolplanesdk.GetOrganizationRequest) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error)
	listFn        func(context.Context, tenantmanagercontrolplanesdk.ListOrganizationsRequest) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error)
	updateFn      func(context.Context, tenantmanagercontrolplanesdk.UpdateOrganizationRequest) (tenantmanagercontrolplanesdk.UpdateOrganizationResponse, error)
	workRequestFn func(context.Context, tenantmanagercontrolplanesdk.GetWorkRequestRequest) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error)

	getRequests         []tenantmanagercontrolplanesdk.GetOrganizationRequest
	listRequests        []tenantmanagercontrolplanesdk.ListOrganizationsRequest
	updateRequests      []tenantmanagercontrolplanesdk.UpdateOrganizationRequest
	workRequestRequests []tenantmanagercontrolplanesdk.GetWorkRequestRequest
}

func (f *fakeOrganizationRuntimeOCIClient) GetOrganization(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.GetOrganizationRequest,
) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.GetOrganizationResponse{}, nil
}

func (f *fakeOrganizationRuntimeOCIClient) ListOrganizations(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.ListOrganizationsRequest,
) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.ListOrganizationsResponse{}, nil
}

func (f *fakeOrganizationRuntimeOCIClient) UpdateOrganization(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.UpdateOrganizationRequest,
) (tenantmanagercontrolplanesdk.UpdateOrganizationResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.UpdateOrganizationResponse{}, nil
}

func (f *fakeOrganizationRuntimeOCIClient) GetWorkRequest(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.GetWorkRequestRequest,
) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, req)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.GetWorkRequestResponse{}, nil
}

func TestOrganizationRuntimeConfigHasNoCreateOrDeleteOperation(t *testing.T) {
	t.Parallel()

	config := buildOrganizationGeneratedRuntimeConfig(
		&OrganizationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}},
		func() OrganizationRuntimeHooks {
			hooks := newOrganizationRuntimeHooksWithClients(&fakeOrganizationRuntimeOCIClient{})
			applyOrganizationRuntimeHooks(&hooks, &fakeOrganizationRuntimeOCIClient{}, &fakeOrganizationRuntimeOCIClient{}, nil, nil)
			return hooks
		}(),
	)
	if config.Create != nil {
		t.Fatal("config.Create != nil, want bind-existing Organization runtime without Create")
	}
	if config.Delete != nil {
		t.Fatal("config.Delete != nil, want CR-local unbind without Delete")
	}
	if config.Semantics == nil || config.Semantics.Async == nil || len(config.Semantics.Async.WorkRequest.Phases) != 1 || config.Semantics.Async.WorkRequest.Phases[0] != "update" {
		t.Fatalf("Semantics.Async = %#v, want update-only workrequest semantics", config.Semantics.Async)
	}
	if config.Semantics.FinalizerPolicy != "none" {
		t.Fatalf("FinalizerPolicy = %q, want %q", config.Semantics.FinalizerPolicy, "none")
	}
}

func TestOrganizationCreateOrUpdateRejectsMissingBindingIdentity(t *testing.T) {
	t.Parallel()

	resource := newOrganizationResource()
	resource.Spec = tenantmanagercontrolplanev1beta1.OrganizationSpec{}

	response, err := newTestOrganizationClient(&fakeOrganizationRuntimeOCIClient{}).CreateOrUpdate(
		context.Background(),
		resource,
		requestForOrganization(resource),
	)
	if err == nil || err.Error() != "Organization bind-existing flow requires spec.organizationId or spec.compartmentId" {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit bind-existing validation", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful validation failure", response)
	}
}

func TestOrganizationCreateOrUpdateRejectsTrackedOrganizationIDDrift(t *testing.T) {
	t.Parallel()

	resource := trackedOrganizationResource()
	resource.Spec.OrganizationId = testOtherOrganizationID

	fake := &fakeOrganizationRuntimeOCIClient{}
	response, err := newTestOrganizationClient(fake).CreateOrUpdate(context.Background(), resource, requestForOrganization(resource))
	if err == nil || err.Error() != "Organization formal semantics require replacement when organizationId changes" {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-only organizationId drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	if len(fake.getRequests) != 0 || len(fake.listRequests) != 0 || len(fake.updateRequests) != 0 {
		t.Fatalf("unexpected OCI calls: get=%d list=%d update=%d", len(fake.getRequests), len(fake.listRequests), len(fake.updateRequests))
	}
}

func TestOrganizationCreateOrUpdateDirectBindDoesNotFallbackToList(t *testing.T) {
	t.Parallel()

	resource := newOrganizationResource()
	resource.Spec.OrganizationId = testOrganizationID
	resource.Spec.CompartmentId = testOrganizationCompartment

	fake := &fakeOrganizationRuntimeOCIClient{
		getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetOrganizationRequest) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error) {
			requireStringPtr(t, "GetOrganizationRequest.OrganizationId", req.OrganizationId, testOrganizationID)
			return tenantmanagercontrolplanesdk.GetOrganizationResponse{}, stubServiceError{statusCode: 404, code: "NotFound"}
		},
		listFn: func(context.Context, tenantmanagercontrolplanesdk.ListOrganizationsRequest) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error) {
			t.Fatal("ListOrganizations() called, want explicit organizationId bind failure without list fallback")
			return tenantmanagercontrolplanesdk.ListOrganizationsResponse{}, nil
		},
	}

	response, err := newTestOrganizationClient(fake).CreateOrUpdate(context.Background(), resource, requestForOrganization(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want explicit organizationId lookup failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful direct-bind failure", response)
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("List calls = %d, want 0 after explicit organizationId miss", len(fake.listRequests))
	}
}

func TestOrganizationCreateOrUpdateBindsExistingOrganizationByPagedList(t *testing.T) {
	t.Parallel()

	resource := newOrganizationResource()
	resource.Spec.OrganizationId = ""
	resource.Spec.CompartmentId = testOrganizationCompartment

	fake := &fakeOrganizationRuntimeOCIClient{
		listFn: func(_ context.Context, req tenantmanagercontrolplanesdk.ListOrganizationsRequest) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error) {
			requireStringPtr(t, "ListOrganizationsRequest.CompartmentId", req.CompartmentId, testOrganizationCompartment)
			switch page := stringPtrValue(req.Page); page {
			case "":
				return tenantmanagercontrolplanesdk.ListOrganizationsResponse{
					OrganizationCollection: tenantmanagercontrolplanesdk.OrganizationCollection{},
					OpcNextPage:            common.String("page-2"),
				}, nil
			case "page-2":
				return tenantmanagercontrolplanesdk.ListOrganizationsResponse{
					OrganizationCollection: tenantmanagercontrolplanesdk.OrganizationCollection{
						Items: []tenantmanagercontrolplanesdk.OrganizationSummary{
							organizationSummary(testOrganizationID, testOldSubscriptionID),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", page)
				return tenantmanagercontrolplanesdk.ListOrganizationsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetOrganizationRequest) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error) {
			requireStringPtr(t, "GetOrganizationRequest.OrganizationId", req.OrganizationId, testOrganizationID)
			return tenantmanagercontrolplanesdk.GetOrganizationResponse{
				Organization: activeOrganizationSDK(testOrganizationID, testOldSubscriptionID),
			}, nil
		},
	}

	response, err := newTestOrganizationClient(fake).CreateOrUpdate(context.Background(), resource, requestForOrganization(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind without requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("List calls = %d, want 2 paged lookups", len(fake.listRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("Get calls = %d, want 1 live read after list bind", len(fake.getRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want no update on matching bind", len(fake.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testOrganizationID {
		t.Fatalf("status.ocid = %q, want %q", got, testOrganizationID)
	}
}

func TestOrganizationCreateOrUpdateResumesUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := trackedOrganizationResource()
	resource.Spec.OrganizationId = ""
	resource.Spec.DefaultUcmSubscriptionId = testNewSubscriptionID

	workRequestCalls := 0
	fake := &fakeOrganizationRuntimeOCIClient{
		getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetOrganizationRequest) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error) {
			requireStringPtr(t, "GetOrganizationRequest.OrganizationId", req.OrganizationId, testOrganizationID)
			if workRequestCalls == 0 {
				return tenantmanagercontrolplanesdk.GetOrganizationResponse{
					Organization: activeOrganizationSDK(testOrganizationID, testOldSubscriptionID),
				}, nil
			}
			return tenantmanagercontrolplanesdk.GetOrganizationResponse{
				Organization: activeOrganizationSDK(testOrganizationID, testNewSubscriptionID),
			}, nil
		},
		updateFn: func(_ context.Context, req tenantmanagercontrolplanesdk.UpdateOrganizationRequest) (tenantmanagercontrolplanesdk.UpdateOrganizationResponse, error) {
			requireStringPtr(t, "UpdateOrganizationRequest.OrganizationId", req.OrganizationId, testOrganizationID)
			requireStringPtr(t, "UpdateOrganizationRequest.DefaultUcmSubscriptionId", req.DefaultUcmSubscriptionId, testNewSubscriptionID)
			return tenantmanagercontrolplanesdk.UpdateOrganizationResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetWorkRequestRequest) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-update-1")
			workRequestCalls++
			status := tenantmanagercontrolplanesdk.OperationStatusInProgress
			action := tenantmanagercontrolplanesdk.ActionTypeInProgress
			if workRequestCalls == 2 {
				status = tenantmanagercontrolplanesdk.OperationStatusSucceeded
				action = tenantmanagercontrolplanesdk.ActionTypeUpdated
			}
			return tenantmanagercontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: organizationWorkRequest("wr-update-1", status, action, testOrganizationID),
			}, nil
		},
	}

	client := newTestOrganizationClient(fake)

	firstResponse, err := client.CreateOrUpdate(context.Background(), resource, requestForOrganization(resource))
	if err != nil {
		t.Fatalf("first CreateOrUpdate() error = %v", err)
	}
	if !firstResponse.IsSuccessful || !firstResponse.ShouldRequeue {
		t.Fatalf("first CreateOrUpdate() response = %#v, want pending work request requeue", firstResponse)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want tracked update work request")
	}
	if got := resource.Status.OsokStatus.Async.Current.WorkRequestID; got != "wr-update-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", got, "wr-update-1")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls after first pass = %d, want 1", len(fake.updateRequests))
	}

	secondResponse, err := client.CreateOrUpdate(context.Background(), resource, requestForOrganization(resource))
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !secondResponse.IsSuccessful || secondResponse.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want settled active state", secondResponse)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("Update calls after resume = %d, want 1", len(fake.updateRequests))
	}
	if len(fake.workRequestRequests) != 2 {
		t.Fatalf("GetWorkRequest calls = %d, want 2", len(fake.workRequestRequests))
	}
	if got := resource.Status.DefaultUcmSubscriptionId; got != testNewSubscriptionID {
		t.Fatalf("status.defaultUcmSubscriptionId = %q, want %q", got, testNewSubscriptionID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after successful update follow-up", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOrganizationDeleteIsCRLocalUnbind(t *testing.T) {
	t.Parallel()

	resource := trackedOrganizationResource()
	fake := &fakeOrganizationRuntimeOCIClient{}

	deleted, err := newTestOrganizationClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true for CR-local unbind")
	}
	if len(fake.getRequests) != 0 || len(fake.listRequests) != 0 || len(fake.updateRequests) != 0 || len(fake.workRequestRequests) != 0 {
		t.Fatalf("unexpected OCI calls during delete: get=%d list=%d update=%d workRequest=%d", len(fake.getRequests), len(fake.listRequests), len(fake.updateRequests), len(fake.workRequestRequests))
	}
}

func newTestOrganizationClient(fake *fakeOrganizationRuntimeOCIClient) OrganizationServiceClient {
	tlog := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return newOrganizationServiceClientWithClients(tlog, fake, fake)
}

func newOrganizationResource() *tenantmanagercontrolplanev1beta1.Organization {
	return &tenantmanagercontrolplanev1beta1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "organization-sample",
			Namespace: "default",
		},
		Spec: tenantmanagercontrolplanev1beta1.OrganizationSpec{
			CompartmentId: testOrganizationCompartment,
		},
	}
}

func trackedOrganizationResource() *tenantmanagercontrolplanev1beta1.Organization {
	resource := newOrganizationResource()
	resource.Status.Id = testOrganizationID
	resource.Status.CompartmentId = testOrganizationCompartment
	resource.Status.DefaultUcmSubscriptionId = testOldSubscriptionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testOrganizationID)
	return resource
}

func requestForOrganization(resource *tenantmanagercontrolplanev1beta1.Organization) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      resource.Name,
			Namespace: resource.Namespace,
		},
	}
}

func activeOrganizationSDK(id string, defaultSubscriptionID string) tenantmanagercontrolplanesdk.Organization {
	return tenantmanagercontrolplanesdk.Organization{
		Id:                       common.String(id),
		CompartmentId:            common.String(testOrganizationCompartment),
		DefaultUcmSubscriptionId: common.String(defaultSubscriptionID),
		LifecycleState:           tenantmanagercontrolplanesdk.OrganizationLifecycleStateActive,
		DisplayName:              common.String("organization-sample"),
		ParentName:               common.String("parent-tenancy"),
	}
}

func organizationSummary(id string, defaultSubscriptionID string) tenantmanagercontrolplanesdk.OrganizationSummary {
	return tenantmanagercontrolplanesdk.OrganizationSummary{
		Id:                       common.String(id),
		CompartmentId:            common.String(testOrganizationCompartment),
		DefaultUcmSubscriptionId: common.String(defaultSubscriptionID),
		LifecycleState:           tenantmanagercontrolplanesdk.OrganizationLifecycleStateActive,
		DisplayName:              common.String("organization-sample"),
		ParentName:               common.String("parent-tenancy"),
	}
}

func organizationWorkRequest(
	workRequestID string,
	status tenantmanagercontrolplanesdk.OperationStatusEnum,
	action tenantmanagercontrolplanesdk.ActionTypeEnum,
	resourceID string,
) tenantmanagercontrolplanesdk.WorkRequest {
	return tenantmanagercontrolplanesdk.WorkRequest{
		OperationType:   tenantmanagercontrolplanesdk.OperationTypeAssignDefaultSubscription,
		Status:          status,
		Id:              common.String(workRequestID),
		CompartmentId:   common.String(testOrganizationCompartment),
		Resources:       []tenantmanagercontrolplanesdk.WorkRequestResource{{EntityType: common.String("Organization"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: common.Float32(50),
	}
}

func requireStringPtr(t *testing.T, field string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if got := *value; got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type stubServiceError struct {
	statusCode int
	code       string
}

func (e stubServiceError) Error() string {
	return e.code
}

func (e stubServiceError) GetHTTPStatusCode() int {
	return e.statusCode
}

func (e stubServiceError) GetMessage() string {
	return e.code
}

func (e stubServiceError) GetCode() string {
	return e.code
}

func (e stubServiceError) GetOpcRequestID() string {
	return ""
}

func (e stubServiceError) GetCause() error {
	return nil
}

func (e stubServiceError) GetSuggestions() []string {
	return nil
}

func (e stubServiceError) GetOperationName() string {
	return ""
}

func (e stubServiceError) GetTimestamp() string {
	return ""
}

var _ common.ServiceError = stubServiceError{}
