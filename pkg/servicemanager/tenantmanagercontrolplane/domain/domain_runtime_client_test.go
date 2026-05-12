/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package domain

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
	testDomainID            = "ocid1.domain.oc1..example"
	testOtherDomainID       = "ocid1.domain.oc1..other"
	testDomainCompartment   = "ocid1.tenancy.oc1..example"
	testOtherCompartment    = "ocid1.tenancy.oc1..other"
	testDomainName          = "example.com"
	testOtherDomainName     = "other.example.com"
	testOwnerID             = "ocid1.tenancy.oc1..owner"
	testSubscriptionEmail   = "admin@example.com"
	testOtherSubscription   = "new-admin@example.com"
	testDomainWorkRequestID = "wr-domain-1"
)

type fakeDomainRuntimeOCIClient struct {
	createFn      func(context.Context, tenantmanagercontrolplanesdk.CreateDomainRequest) (tenantmanagercontrolplanesdk.CreateDomainResponse, error)
	getFn         func(context.Context, tenantmanagercontrolplanesdk.GetDomainRequest) (tenantmanagercontrolplanesdk.GetDomainResponse, error)
	listFn        func(context.Context, tenantmanagercontrolplanesdk.ListDomainsRequest) (tenantmanagercontrolplanesdk.ListDomainsResponse, error)
	updateFn      func(context.Context, tenantmanagercontrolplanesdk.UpdateDomainRequest) (tenantmanagercontrolplanesdk.UpdateDomainResponse, error)
	deleteFn      func(context.Context, tenantmanagercontrolplanesdk.DeleteDomainRequest) (tenantmanagercontrolplanesdk.DeleteDomainResponse, error)
	workRequestFn func(context.Context, tenantmanagercontrolplanesdk.GetWorkRequestRequest) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error)

	createRequests      []tenantmanagercontrolplanesdk.CreateDomainRequest
	getRequests         []tenantmanagercontrolplanesdk.GetDomainRequest
	listRequests        []tenantmanagercontrolplanesdk.ListDomainsRequest
	updateRequests      []tenantmanagercontrolplanesdk.UpdateDomainRequest
	deleteRequests      []tenantmanagercontrolplanesdk.DeleteDomainRequest
	workRequestRequests []tenantmanagercontrolplanesdk.GetWorkRequestRequest
}

func (f *fakeDomainRuntimeOCIClient) CreateDomain(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.CreateDomainRequest,
) (tenantmanagercontrolplanesdk.CreateDomainResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.CreateDomainResponse{}, nil
}

func (f *fakeDomainRuntimeOCIClient) GetDomain(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.GetDomainRequest,
) (tenantmanagercontrolplanesdk.GetDomainResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.GetDomainResponse{}, nil
}

func (f *fakeDomainRuntimeOCIClient) ListDomains(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.ListDomainsRequest,
) (tenantmanagercontrolplanesdk.ListDomainsResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.ListDomainsResponse{}, nil
}

func (f *fakeDomainRuntimeOCIClient) UpdateDomain(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.UpdateDomainRequest,
) (tenantmanagercontrolplanesdk.UpdateDomainResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.UpdateDomainResponse{}, nil
}

func (f *fakeDomainRuntimeOCIClient) DeleteDomain(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.DeleteDomainRequest,
) (tenantmanagercontrolplanesdk.DeleteDomainResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.DeleteDomainResponse{}, nil
}

func (f *fakeDomainRuntimeOCIClient) GetWorkRequest(
	ctx context.Context,
	req tenantmanagercontrolplanesdk.GetWorkRequestRequest,
) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, req)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return tenantmanagercontrolplanesdk.GetWorkRequestResponse{}, nil
}

func TestDomainCreateOrUpdateBindsExistingDomainByPagedList(t *testing.T) {
	t.Parallel()

	resource := newDomainResource()
	resource.Spec.Status = string(tenantmanagercontrolplanesdk.DomainStatusActive)
	resource.Spec.LifecycleState = string(tenantmanagercontrolplanesdk.ListDomainsLifecycleStateActive)

	fake := &fakeDomainRuntimeOCIClient{
		listFn: func(_ context.Context, req tenantmanagercontrolplanesdk.ListDomainsRequest) (tenantmanagercontrolplanesdk.ListDomainsResponse, error) {
			requireStringPtr(t, "ListDomainsRequest.CompartmentId", req.CompartmentId, testDomainCompartment)
			requireStringPtr(t, "ListDomainsRequest.Name", req.Name, testDomainName)
			if req.Status != tenantmanagercontrolplanesdk.DomainStatusActive {
				t.Fatalf("ListDomainsRequest.Status = %q, want %q", req.Status, tenantmanagercontrolplanesdk.DomainStatusActive)
			}
			if req.LifecycleState != tenantmanagercontrolplanesdk.ListDomainsLifecycleStateActive {
				t.Fatalf("ListDomainsRequest.LifecycleState = %q, want %q", req.LifecycleState, tenantmanagercontrolplanesdk.ListDomainsLifecycleStateActive)
			}
			switch page := stringPtrValue(req.Page); page {
			case "":
				return tenantmanagercontrolplanesdk.ListDomainsResponse{
					DomainCollection: tenantmanagercontrolplanesdk.DomainCollection{},
					OpcNextPage:      common.String("page-2"),
				}, nil
			case "page-2":
				return tenantmanagercontrolplanesdk.ListDomainsResponse{
					DomainCollection: tenantmanagercontrolplanesdk.DomainCollection{
						Items: []tenantmanagercontrolplanesdk.DomainSummary{
							activeDomainSummarySDK(testDomainID),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", page)
				return tenantmanagercontrolplanesdk.ListDomainsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetDomainRequest) (tenantmanagercontrolplanesdk.GetDomainResponse, error) {
			requireStringPtr(t, "GetDomainRequest.DomainId", req.DomainId, testDomainID)
			return tenantmanagercontrolplanesdk.GetDomainResponse{
				Domain: activeDomainSDK(testDomainID, tenantmanagercontrolplanesdk.DomainStatusActive, nil),
			}, nil
		},
	}

	response, err := newTestDomainClient(fake).CreateOrUpdate(context.Background(), resource, requestForDomain(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful active bind", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListDomains calls = %d, want 2 paged lookups", len(fake.listRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetDomain calls = %d, want 1 live read after list bind", len(fake.getRequests))
	}
	if len(fake.createRequests) != 0 || len(fake.updateRequests) != 0 || len(fake.deleteRequests) != 0 || len(fake.workRequestRequests) != 0 {
		t.Fatalf(
			"unexpected OCI calls: create=%d update=%d delete=%d workRequest=%d",
			len(fake.createRequests),
			len(fake.updateRequests),
			len(fake.deleteRequests),
			len(fake.workRequestRequests),
		)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDomainID {
		t.Fatalf("status.ocid = %q, want %q", got, testDomainID)
	}
	if got := resource.Status.CompartmentId; got != testDomainCompartment {
		t.Fatalf("status.compartmentId = %q, want %q", got, testDomainCompartment)
	}
	if got := resource.Status.SubscriptionEmail; got != testSubscriptionEmail {
		t.Fatalf("status.subscriptionEmail = %q, want %q", got, testSubscriptionEmail)
	}
}

func TestDomainCreateOrUpdateRejectsTrackedCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		mutate    func(*tenantmanagercontrolplanev1beta1.Domain)
		wantError string
	}{
		{
			name: "compartmentId",
			mutate: func(resource *tenantmanagercontrolplanev1beta1.Domain) {
				resource.Spec.CompartmentId = testOtherCompartment
			},
			wantError: "Domain formal semantics require replacement when compartmentId changes",
		},
		{
			name: "subscriptionEmail",
			mutate: func(resource *tenantmanagercontrolplanev1beta1.Domain) {
				resource.Spec.SubscriptionEmail = testOtherSubscription
			},
			wantError: "Domain formal semantics require replacement when subscriptionEmail changes",
		},
		{
			name: "isGovernanceEnabled",
			mutate: func(resource *tenantmanagercontrolplanev1beta1.Domain) {
				resource.Spec.IsGovernanceEnabled = true
			},
			wantError: "Domain formal semantics require replacement when isGovernanceEnabled changes",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := trackedDomainResource()
			tc.mutate(resource)

			fake := &fakeDomainRuntimeOCIClient{
				getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetDomainRequest) (tenantmanagercontrolplanesdk.GetDomainResponse, error) {
					requireStringPtr(t, "GetDomainRequest.DomainId", req.DomainId, testDomainID)
					return tenantmanagercontrolplanesdk.GetDomainResponse{
						Domain: activeDomainSDK(testDomainID, tenantmanagercontrolplanesdk.DomainStatusActive, nil),
					}, nil
				},
			}

			response, err := newTestDomainClient(fake).CreateOrUpdate(context.Background(), resource, requestForDomain(resource))
			if err == nil || err.Error() != tc.wantError {
				t.Fatalf("CreateOrUpdate() error = %v, want %q", err, tc.wantError)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
			}
			if len(fake.updateRequests) != 0 {
				t.Fatalf("UpdateDomain calls = %d, want 0 after create-only drift rejection", len(fake.updateRequests))
			}
		})
	}
}

func TestDomainCreateOrUpdateStartsAndResumesCreateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newDomainResource()
	workRequestCalls := 0
	fake := &fakeDomainRuntimeOCIClient{
		listFn: func(_ context.Context, req tenantmanagercontrolplanesdk.ListDomainsRequest) (tenantmanagercontrolplanesdk.ListDomainsResponse, error) {
			requireStringPtr(t, "ListDomainsRequest.CompartmentId", req.CompartmentId, testDomainCompartment)
			requireStringPtr(t, "ListDomainsRequest.Name", req.Name, testDomainName)
			return tenantmanagercontrolplanesdk.ListDomainsResponse{
				DomainCollection: tenantmanagercontrolplanesdk.DomainCollection{},
			}, nil
		},
		createFn: func(_ context.Context, req tenantmanagercontrolplanesdk.CreateDomainRequest) (tenantmanagercontrolplanesdk.CreateDomainResponse, error) {
			requireStringPtr(t, "CreateDomainRequest.CompartmentId", req.CompartmentId, testDomainCompartment)
			requireStringPtr(t, "CreateDomainRequest.DomainName", req.DomainName, testDomainName)
			requireStringPtr(t, "CreateDomainRequest.SubscriptionEmail", req.SubscriptionEmail, testSubscriptionEmail)
			if req.IsGovernanceEnabled != nil {
				t.Fatalf("CreateDomainRequest.IsGovernanceEnabled = %v, want nil for default false", *req.IsGovernanceEnabled)
			}
			return tenantmanagercontrolplanesdk.CreateDomainResponse{
				Domain:           activeDomainSDK(testDomainID, tenantmanagercontrolplanesdk.DomainStatusPending, nil),
				OpcWorkRequestId: common.String(testDomainWorkRequestID),
			}, nil
		},
		workRequestFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetWorkRequestRequest) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, testDomainWorkRequestID)
			workRequestCalls++
			status := tenantmanagercontrolplanesdk.OperationStatusInProgress
			action := tenantmanagercontrolplanesdk.ActionTypeInProgress
			if workRequestCalls == 2 {
				status = tenantmanagercontrolplanesdk.OperationStatusSucceeded
				action = tenantmanagercontrolplanesdk.ActionTypeCreated
			}
			return tenantmanagercontrolplanesdk.GetWorkRequestResponse{
				WorkRequest: domainWorkRequest(testDomainWorkRequestID, status, action, testDomainID),
			}, nil
		},
		getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetDomainRequest) (tenantmanagercontrolplanesdk.GetDomainResponse, error) {
			requireStringPtr(t, "GetDomainRequest.DomainId", req.DomainId, testDomainID)
			return tenantmanagercontrolplanesdk.GetDomainResponse{
				Domain: activeDomainSDK(testDomainID, tenantmanagercontrolplanesdk.DomainStatusActive, map[string]string{"env": "prod"}),
			}, nil
		},
	}

	client := newTestDomainClient(fake)

	firstResponse, err := client.CreateOrUpdate(context.Background(), resource, requestForDomain(resource))
	if err != nil {
		t.Fatalf("first CreateOrUpdate() error = %v", err)
	}
	if !firstResponse.IsSuccessful || !firstResponse.ShouldRequeue {
		t.Fatalf("first CreateOrUpdate() response = %#v, want pending create work request requeue", firstResponse)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want tracked create work request")
	}
	if got := resource.Status.OsokStatus.Async.Current.WorkRequestID; got != testDomainWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", got, testDomainWorkRequestID)
	}
	if got := resource.Status.OsokStatus.Async.Current.RawOperationType; got != string(tenantmanagercontrolplanesdk.OperationTypeRegisterDomain) {
		t.Fatalf("status.async.current.rawOperationType = %q, want %q", got, tenantmanagercontrolplanesdk.OperationTypeRegisterDomain)
	}
	if got := resource.Status.CompartmentId; got != testDomainCompartment {
		t.Fatalf("status.compartmentId = %q, want %q after create starts", got, testDomainCompartment)
	}

	secondResponse, err := client.CreateOrUpdate(context.Background(), resource, requestForDomain(resource))
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !secondResponse.IsSuccessful || secondResponse.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want settled active state", secondResponse)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateDomain calls = %d, want 1", len(fake.createRequests))
	}
	if len(fake.workRequestRequests) != 2 {
		t.Fatalf("GetWorkRequest calls = %d, want 2", len(fake.workRequestRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetDomain calls = %d, want 1 after work request success", len(fake.getRequests))
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after successful create follow-up", resource.Status.OsokStatus.Async.Current)
	}
	if got := resource.Status.FreeformTags["env"]; got != "prod" {
		t.Fatalf("status.freeformTags[env] = %q, want %q", got, "prod")
	}
}

func TestDomainCreateOrUpdateProjectsUpdateResponseDirectly(t *testing.T) {
	t.Parallel()

	resource := trackedDomainResource()
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	fake := &fakeDomainRuntimeOCIClient{
		getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetDomainRequest) (tenantmanagercontrolplanesdk.GetDomainResponse, error) {
			requireStringPtr(t, "GetDomainRequest.DomainId", req.DomainId, testDomainID)
			return tenantmanagercontrolplanesdk.GetDomainResponse{
				Domain: activeDomainSDK(testDomainID, tenantmanagercontrolplanesdk.DomainStatusActive, map[string]string{"env": "dev"}),
			}, nil
		},
		updateFn: func(_ context.Context, req tenantmanagercontrolplanesdk.UpdateDomainRequest) (tenantmanagercontrolplanesdk.UpdateDomainResponse, error) {
			requireStringPtr(t, "UpdateDomainRequest.DomainId", req.DomainId, testDomainID)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateDomainRequest.FreeformTags[env] = %q, want %q", got, "prod")
			}
			return tenantmanagercontrolplanesdk.UpdateDomainResponse{
				Domain: activeDomainSDK(testDomainID, tenantmanagercontrolplanesdk.DomainStatusActive, map[string]string{"env": "prod"}),
			}, nil
		},
	}

	response, err := newTestDomainClient(fake).CreateOrUpdate(context.Background(), resource, requestForDomain(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want synchronous update success", response)
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetDomain calls = %d, want 1 pre-update read", len(fake.getRequests))
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateDomain calls = %d, want 1", len(fake.updateRequests))
	}
	if len(fake.workRequestRequests) != 0 {
		t.Fatalf("GetWorkRequest calls = %d, want 0 for direct-body update", len(fake.workRequestRequests))
	}
	if got := resource.Status.FreeformTags["env"]; got != "prod" {
		t.Fatalf("status.freeformTags[env] = %q, want %q", got, "prod")
	}
}

func TestDomainDeleteRequeuesUntilReadConfirmsGone(t *testing.T) {
	t.Parallel()

	resource := trackedDomainResource()
	getCalls := 0
	fake := &fakeDomainRuntimeOCIClient{
		getFn: func(_ context.Context, req tenantmanagercontrolplanesdk.GetDomainRequest) (tenantmanagercontrolplanesdk.GetDomainResponse, error) {
			requireStringPtr(t, "GetDomainRequest.DomainId", req.DomainId, testDomainID)
			getCalls++
			switch getCalls {
			case 1, 2, 3:
				return tenantmanagercontrolplanesdk.GetDomainResponse{
					Domain: activeDomainSDK(testDomainID, tenantmanagercontrolplanesdk.DomainStatusActive, nil),
				}, nil
			case 4:
				return tenantmanagercontrolplanesdk.GetDomainResponse{}, stubServiceError{statusCode: 404, code: "NotFound"}
			default:
				t.Fatalf("unexpected GetDomain call %d", getCalls)
				return tenantmanagercontrolplanesdk.GetDomainResponse{}, nil
			}
		},
		deleteFn: func(_ context.Context, req tenantmanagercontrolplanesdk.DeleteDomainRequest) (tenantmanagercontrolplanesdk.DeleteDomainResponse, error) {
			requireStringPtr(t, "DeleteDomainRequest.DomainId", req.DomainId, testDomainID)
			return tenantmanagercontrolplanesdk.DeleteDomainResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	}

	client := newTestDomainClient(fake)

	firstDeleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("first Delete() error = %v", err)
	}
	if firstDeleted {
		t.Fatal("first Delete() = true, want in-progress delete")
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want lifecycle-backed pending delete", resource.Status.OsokStatus.Async.Current)
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteDomain calls after first pass = %d, want 1", len(fake.deleteRequests))
	}

	secondDeleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if secondDeleted {
		t.Fatal("second Delete() = true, want continued pending delete")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteDomain calls after second pass = %d, want no repeat delete while confirmation is pending", len(fake.deleteRequests))
	}

	thirdDeleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("third Delete() error = %v", err)
	}
	if !thirdDeleted {
		t.Fatal("third Delete() = false, want delete confirmed by not found")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteDomain calls after final confirmation = %d, want 1 total", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after delete confirmation", resource.Status.OsokStatus.Async.Current)
	}
}

func newTestDomainClient(fake *fakeDomainRuntimeOCIClient) DomainServiceClient {
	tlog := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return newDomainServiceClientWithClients(tlog, fake, fake)
}

func newDomainResource() *tenantmanagercontrolplanev1beta1.Domain {
	return &tenantmanagercontrolplanev1beta1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "domain-sample",
			Namespace: "default",
		},
		Spec: tenantmanagercontrolplanev1beta1.DomainSpec{
			CompartmentId:       testDomainCompartment,
			DomainName:          testDomainName,
			SubscriptionEmail:   testSubscriptionEmail,
			IsGovernanceEnabled: false,
		},
	}
}

func trackedDomainResource() *tenantmanagercontrolplanev1beta1.Domain {
	resource := newDomainResource()
	resource.Status.Id = testDomainID
	resource.Status.DomainName = testDomainName
	resource.Status.OwnerId = testOwnerID
	resource.Status.CompartmentId = testDomainCompartment
	resource.Status.SubscriptionEmail = testSubscriptionEmail
	resource.Status.IsGovernanceEnabled = false
	resource.Status.LifecycleState = string(tenantmanagercontrolplanesdk.DomainLifecycleStateActive)
	resource.Status.Status = string(tenantmanagercontrolplanesdk.DomainStatusActive)
	resource.Status.OsokStatus.Ocid = shared.OCID(testDomainID)
	return resource
}

func requestForDomain(resource *tenantmanagercontrolplanev1beta1.Domain) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      resource.Name,
			Namespace: resource.Namespace,
		},
	}
}

func activeDomainSDK(
	id string,
	status tenantmanagercontrolplanesdk.DomainStatusEnum,
	freeformTags map[string]string,
) tenantmanagercontrolplanesdk.Domain {
	return tenantmanagercontrolplanesdk.Domain{
		Id:             common.String(id),
		DomainName:     common.String(testDomainName),
		OwnerId:        common.String(testOwnerID),
		LifecycleState: tenantmanagercontrolplanesdk.DomainLifecycleStateActive,
		Status:         status,
		TxtRecord:      common.String("txt-value"),
		FreeformTags:   cloneStringMap(freeformTags),
	}
}

func activeDomainSummarySDK(id string) tenantmanagercontrolplanesdk.DomainSummary {
	return tenantmanagercontrolplanesdk.DomainSummary{
		Id:             common.String(id),
		DomainName:     common.String(testDomainName),
		OwnerId:        common.String(testOwnerID),
		LifecycleState: tenantmanagercontrolplanesdk.DomainLifecycleStateActive,
		Status:         tenantmanagercontrolplanesdk.DomainStatusActive,
		TxtRecord:      common.String("txt-value"),
	}
}

func domainWorkRequest(
	workRequestID string,
	status tenantmanagercontrolplanesdk.OperationStatusEnum,
	action tenantmanagercontrolplanesdk.ActionTypeEnum,
	resourceID string,
) tenantmanagercontrolplanesdk.WorkRequest {
	return tenantmanagercontrolplanesdk.WorkRequest{
		OperationType:   tenantmanagercontrolplanesdk.OperationTypeRegisterDomain,
		Status:          status,
		Id:              common.String(workRequestID),
		CompartmentId:   common.String(testDomainCompartment),
		Resources:       []tenantmanagercontrolplanesdk.WorkRequestResource{{EntityType: common.String("Domain"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: common.Float32(50),
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
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
