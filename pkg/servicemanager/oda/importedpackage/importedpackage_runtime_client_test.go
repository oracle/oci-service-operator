/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package importedpackage

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type importedPackageOCIClient interface {
	CreateImportedPackage(context.Context, odasdk.CreateImportedPackageRequest) (odasdk.CreateImportedPackageResponse, error)
	GetImportedPackage(context.Context, odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error)
	ListImportedPackages(context.Context, odasdk.ListImportedPackagesRequest) (odasdk.ListImportedPackagesResponse, error)
	UpdateImportedPackage(context.Context, odasdk.UpdateImportedPackageRequest) (odasdk.UpdateImportedPackageResponse, error)
	DeleteImportedPackage(context.Context, odasdk.DeleteImportedPackageRequest) (odasdk.DeleteImportedPackageResponse, error)
}

type fakeImportedPackageOCIClient struct {
	createFn func(context.Context, odasdk.CreateImportedPackageRequest) (odasdk.CreateImportedPackageResponse, error)
	getFn    func(context.Context, odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error)
	listFn   func(context.Context, odasdk.ListImportedPackagesRequest) (odasdk.ListImportedPackagesResponse, error)
	updateFn func(context.Context, odasdk.UpdateImportedPackageRequest) (odasdk.UpdateImportedPackageResponse, error)
	deleteFn func(context.Context, odasdk.DeleteImportedPackageRequest) (odasdk.DeleteImportedPackageResponse, error)
}

func (f *fakeImportedPackageOCIClient) CreateImportedPackage(ctx context.Context, req odasdk.CreateImportedPackageRequest) (odasdk.CreateImportedPackageResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return odasdk.CreateImportedPackageResponse{}, nil
}

func (f *fakeImportedPackageOCIClient) GetImportedPackage(ctx context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return odasdk.GetImportedPackageResponse{}, nil
}

func (f *fakeImportedPackageOCIClient) ListImportedPackages(ctx context.Context, req odasdk.ListImportedPackagesRequest) (odasdk.ListImportedPackagesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return odasdk.ListImportedPackagesResponse{}, nil
}

func (f *fakeImportedPackageOCIClient) UpdateImportedPackage(ctx context.Context, req odasdk.UpdateImportedPackageRequest) (odasdk.UpdateImportedPackageResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return odasdk.UpdateImportedPackageResponse{}, nil
}

func (f *fakeImportedPackageOCIClient) DeleteImportedPackage(ctx context.Context, req odasdk.DeleteImportedPackageRequest) (odasdk.DeleteImportedPackageResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return odasdk.DeleteImportedPackageResponse{}, nil
}

func TestImportedPackageRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := newImportedPackageRuntimeSemantics()
	if got == nil {
		t.Fatal("newImportedPackageRuntimeSemantics() = nil")
	}
	if got.FormalService != "oda" {
		t.Fatalf("FormalService = %q, want oda", got.FormalService)
	}
	if got.FormalSlug != "importedpackage" {
		t.Fatalf("FormalSlug = %q, want importedpackage", got.FormalSlug)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}

	assertImportedPackageStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"OPERATION_PENDING"})
	assertImportedPackageStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"READY"})
	assertImportedPackageStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"OPERATION_PENDING"})
	assertImportedPackageStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertImportedPackageStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"odaInstanceId", "currentPackageId"})
	assertImportedPackageStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"currentPackageId", "parameterValues", "freeformTags", "definedTags"})
}

func TestImportedPackageServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	const odaInstanceID = "ocid1.odainstance.oc1..create"
	const packageID = "ocid1.odapackage.oc1..create"
	resource := makeImportedPackageResource(odaInstanceID, packageID)
	resource.Spec.ParameterValues = map[string]string{"threshold": "3"}
	resource.Spec.FreeformTags = map[string]string{"managed-by": "osok"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"key": "value"}}

	getCalls := 0
	var createRequest odasdk.CreateImportedPackageRequest
	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
		listFn: func(_ context.Context, req odasdk.ListImportedPackagesRequest) (odasdk.ListImportedPackagesResponse, error) {
			requireImportedPackageStringPtr(t, "list odaInstanceId", req.OdaInstanceId, odaInstanceID)
			if req.Name != nil {
				t.Fatalf("list name = %q, want nil", *req.Name)
			}
			return odasdk.ListImportedPackagesResponse{}, nil
		},
		createFn: func(_ context.Context, req odasdk.CreateImportedPackageRequest) (odasdk.CreateImportedPackageResponse, error) {
			createRequest = req
			return odasdk.CreateImportedPackageResponse{
				ImportedPackage:  makeSDKImportedPackage(odaInstanceID, packageID, resource, odasdk.ImportedPackageStatusOperationPending),
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
			getCalls++
			requireImportedPackageStringPtr(t, "get odaInstanceId", req.OdaInstanceId, odaInstanceID)
			requireImportedPackageStringPtr(t, "get packageId", req.PackageId, packageID)
			return odasdk.GetImportedPackageResponse{
				ImportedPackage: makeSDKImportedPackage(odaInstanceID, packageID, resource, odasdk.ImportedPackageStatusReady),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after follow-up read reports READY")
	}
	if getCalls != 1 {
		t.Fatalf("GetImportedPackage() calls = %d, want 1 follow-up read", getCalls)
	}
	requireImportedPackageStringPtr(t, "create odaInstanceId", createRequest.OdaInstanceId, odaInstanceID)
	requireImportedPackageStringPtr(t, "create currentPackageId", createRequest.CreateImportedPackageDetails.CurrentPackageId, packageID)
	if !reflect.DeepEqual(createRequest.CreateImportedPackageDetails.ParameterValues, resource.Spec.ParameterValues) {
		t.Fatalf("create parameterValues = %#v, want %#v", createRequest.CreateImportedPackageDetails.ParameterValues, resource.Spec.ParameterValues)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != packageID {
		t.Fatalf("status.ocid = %q, want %q", got, packageID)
	}
	if got := resource.Status.OdaInstanceId; got != odaInstanceID {
		t.Fatalf("status.odaInstanceId = %q, want %q", got, odaInstanceID)
	}
	if got := resource.Status.Status; got != string(odasdk.ImportedPackageStatusReady) {
		t.Fatalf("status.sdkStatus = %q, want READY", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestImportedPackageServiceClientBindsExistingWithoutCreate(t *testing.T) {
	t.Parallel()

	const odaInstanceID = "ocid1.odainstance.oc1..bind"
	const packageID = "ocid1.odapackage.oc1..bind"
	resource := makeImportedPackageResource(odaInstanceID, packageID)
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
		listFn: func(_ context.Context, req odasdk.ListImportedPackagesRequest) (odasdk.ListImportedPackagesResponse, error) {
			requireImportedPackageStringPtr(t, "list odaInstanceId", req.OdaInstanceId, odaInstanceID)
			return odasdk.ListImportedPackagesResponse{
				Items: []odasdk.ImportedPackageSummary{
					makeSDKImportedPackageSummary(odaInstanceID, packageID, resource, odasdk.ImportedPackageStatusReady),
				},
			}, nil
		},
		getFn: func(_ context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
			getCalls++
			requireImportedPackageStringPtr(t, "get packageId", req.PackageId, packageID)
			return odasdk.GetImportedPackageResponse{
				ImportedPackage: makeSDKImportedPackage(odaInstanceID, packageID, resource, odasdk.ImportedPackageStatusReady),
			}, nil
		},
		createFn: func(context.Context, odasdk.CreateImportedPackageRequest) (odasdk.CreateImportedPackageResponse, error) {
			createCalled = true
			return odasdk.CreateImportedPackageResponse{}, nil
		},
		updateFn: func(context.Context, odasdk.UpdateImportedPackageRequest) (odasdk.UpdateImportedPackageResponse, error) {
			updateCalled = true
			return odasdk.UpdateImportedPackageResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if createCalled {
		t.Fatal("CreateImportedPackage() should not be called when list finds a reusable match")
	}
	if updateCalled {
		t.Fatal("UpdateImportedPackage() should not be called when mutable state already matches")
	}
	if getCalls != 1 {
		t.Fatalf("GetImportedPackage() calls = %d, want 1 live assessment read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != packageID {
		t.Fatalf("status.ocid = %q, want %q", got, packageID)
	}
}

func TestImportedPackageServiceClientUpdatesSupportedMutableDrift(t *testing.T) {
	t.Parallel()

	const odaInstanceID = "ocid1.odainstance.oc1..update"
	const oldPackageID = "ocid1.odapackage.oc1..old"
	const newPackageID = "ocid1.odapackage.oc1..new"
	resource := makeImportedPackageResource(odaInstanceID, newPackageID)
	resource.Status.OsokStatus.Ocid = shared.OCID(oldPackageID)
	resource.Status.OdaInstanceId = odaInstanceID
	resource.Status.CurrentPackageId = oldPackageID
	resource.Spec.ParameterValues = map[string]string{"mode": "new"}
	resource.Spec.FreeformTags = map[string]string{"managed-by": "osok"}

	getCalls := 0
	var updateRequest odasdk.UpdateImportedPackageRequest
	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
		getFn: func(_ context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
			getCalls++
			requireImportedPackageStringPtr(t, "get odaInstanceId", req.OdaInstanceId, odaInstanceID)
			if getCalls == 1 {
				requireImportedPackageStringPtr(t, "get packageId", req.PackageId, oldPackageID)
				currentResource := resource.DeepCopy()
				currentResource.Spec.CurrentPackageId = oldPackageID
				currentResource.Spec.ParameterValues = map[string]string{"mode": "old"}
				return odasdk.GetImportedPackageResponse{
					ImportedPackage: makeSDKImportedPackage(odaInstanceID, oldPackageID, currentResource, odasdk.ImportedPackageStatusReady),
				}, nil
			}
			requireImportedPackageStringPtr(t, "get packageId", req.PackageId, oldPackageID)
			return odasdk.GetImportedPackageResponse{
				ImportedPackage: makeSDKImportedPackage(odaInstanceID, newPackageID, resource, odasdk.ImportedPackageStatusReady),
			}, nil
		},
		updateFn: func(_ context.Context, req odasdk.UpdateImportedPackageRequest) (odasdk.UpdateImportedPackageResponse, error) {
			updateRequest = req
			return odasdk.UpdateImportedPackageResponse{
				ImportedPackage: makeSDKImportedPackage(odaInstanceID, newPackageID, resource, odasdk.ImportedPackageStatusOperationPending),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after update follow-up read reports READY")
	}
	requireImportedPackageStringPtr(t, "update odaInstanceId", updateRequest.OdaInstanceId, odaInstanceID)
	requireImportedPackageStringPtr(t, "update packageId", updateRequest.PackageId, oldPackageID)
	requireImportedPackageStringPtr(t, "update currentPackageId", updateRequest.UpdateImportedPackageDetails.CurrentPackageId, newPackageID)
	if !reflect.DeepEqual(updateRequest.UpdateImportedPackageDetails.ParameterValues, resource.Spec.ParameterValues) {
		t.Fatalf("update parameterValues = %#v, want %#v", updateRequest.UpdateImportedPackageDetails.ParameterValues, resource.Spec.ParameterValues)
	}
	if getCalls != 2 {
		t.Fatalf("GetImportedPackage() calls = %d, want live assessment plus update follow-up", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != newPackageID {
		t.Fatalf("status.ocid = %q, want %q", got, newPackageID)
	}
	if got := resource.Status.CurrentPackageId; got != newPackageID {
		t.Fatalf("status.currentPackageId = %q, want %q", got, newPackageID)
	}
}

func TestImportedPackageServiceClientRejectsOdaInstanceDriftBeforeMutation(t *testing.T) {
	t.Parallel()

	const oldOdaInstanceID = "ocid1.odainstance.oc1..old"
	const newOdaInstanceID = "ocid1.odainstance.oc1..new"
	const packageID = "ocid1.odapackage.oc1..drift"
	resource := makeImportedPackageResource(newOdaInstanceID, packageID)
	resource.Status.OdaInstanceId = oldOdaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(packageID)
	getCalled := false
	updateCalled := false

	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
		getFn: func(context.Context, odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
			getCalled = true
			return odasdk.GetImportedPackageResponse{}, nil
		},
		updateFn: func(context.Context, odasdk.UpdateImportedPackageRequest) (odasdk.UpdateImportedPackageResponse, error) {
			updateCalled = true
			return odasdk.UpdateImportedPackageResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "cannot change odaInstanceId") {
		t.Fatalf("CreateOrUpdate() error = %v, want odaInstanceId drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if getCalled {
		t.Fatal("GetImportedPackage() should not be called after identity drift rejection")
	}
	if updateCalled {
		t.Fatal("UpdateImportedPackage() should not be called after identity drift rejection")
	}
}

func TestImportedPackageCreateOrUpdateRequiresOdaInstanceIdentity(t *testing.T) {
	t.Parallel()

	resource := makeImportedPackageResource("", "ocid1.odapackage.oc1..missing-oda")
	resource.Annotations = nil
	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), importedPackageOdaInstanceIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing odaInstanceId annotation error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
}

func TestImportedPackageCreateOrUpdateClassifiesLifecycleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		state          odasdk.ImportedPackageStatusEnum
		wantSuccessful bool
		wantRequeue    bool
		wantReason     shared.OSOKConditionType
	}{
		{
			name:           "ready",
			state:          odasdk.ImportedPackageStatusReady,
			wantSuccessful: true,
			wantRequeue:    false,
			wantReason:     shared.Active,
		},
		{
			name:           "operation-pending",
			state:          odasdk.ImportedPackageStatusOperationPending,
			wantSuccessful: true,
			wantRequeue:    true,
			wantReason:     shared.Provisioning,
		},
		{
			name:           "failed",
			state:          odasdk.ImportedPackageStatusFailed,
			wantSuccessful: false,
			wantRequeue:    false,
			wantReason:     shared.Failed,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const odaInstanceID = "ocid1.odainstance.oc1..lifecycle"
			const packageID = "ocid1.odapackage.oc1..lifecycle"
			resource := makeImportedPackageResource(odaInstanceID, packageID)
			resource.Status.OdaInstanceId = odaInstanceID
			resource.Status.OsokStatus.Ocid = shared.OCID(packageID)

			client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
				getFn: func(_ context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
					requireImportedPackageStringPtr(t, "get packageId", req.PackageId, packageID)
					return odasdk.GetImportedPackageResponse{
						ImportedPackage: makeSDKImportedPackage(odaInstanceID, packageID, resource, tc.state),
					}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil && tc.wantSuccessful {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccessful {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t; err=%v", response.IsSuccessful, tc.wantSuccessful, err)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if got := resource.Status.OsokStatus.Reason; got != string(tc.wantReason) {
				t.Fatalf("status.reason = %q, want %q", got, tc.wantReason)
			}
			if got := resource.Status.Status; got != string(tc.state) {
				t.Fatalf("status.sdkStatus = %q, want %q", got, tc.state)
			}
		})
	}
}

func TestImportedPackageDeleteWaitsForPendingConfirmation(t *testing.T) {
	t.Parallel()

	const odaInstanceID = "ocid1.odainstance.oc1..delete"
	const packageID = "ocid1.odapackage.oc1..delete"
	resource := makeImportedPackageResource(odaInstanceID, packageID)
	resource.Status.OdaInstanceId = odaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(packageID)
	getCalls := 0
	deleteCalls := 0

	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
		getFn: func(_ context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
			getCalls++
			requireImportedPackageStringPtr(t, "get odaInstanceId", req.OdaInstanceId, odaInstanceID)
			requireImportedPackageStringPtr(t, "get packageId", req.PackageId, packageID)
			state := odasdk.ImportedPackageStatusReady
			if getCalls > 1 {
				state = odasdk.ImportedPackageStatusOperationPending
			}
			return odasdk.GetImportedPackageResponse{
				ImportedPackage: makeSDKImportedPackage(odaInstanceID, packageID, resource, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req odasdk.DeleteImportedPackageRequest) (odasdk.DeleteImportedPackageResponse, error) {
			deleteCalls++
			requireImportedPackageStringPtr(t, "delete odaInstanceId", req.OdaInstanceId, odaInstanceID)
			requireImportedPackageStringPtr(t, "delete packageId", req.PackageId, packageID)
			return odasdk.DeleteImportedPackageResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while OCI reports OPERATION_PENDING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteImportedPackage() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.Status; got != string(odasdk.ImportedPackageStatusOperationPending) {
		t.Fatalf("status.sdkStatus = %q, want OPERATION_PENDING", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil while delete confirmation is pending")
	}
}

func TestImportedPackageDeleteKeepsFinalizerWhenReadbackRemainsReady(t *testing.T) {
	t.Parallel()

	const odaInstanceID = "ocid1.odainstance.oc1..delete-ready"
	const packageID = "ocid1.odapackage.oc1..delete-ready"
	resource := makeImportedPackageResource(odaInstanceID, packageID)
	resource.Status.OdaInstanceId = odaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(packageID)
	getCalls := 0

	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
		getFn: func(_ context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
			getCalls++
			requireImportedPackageStringPtr(t, "get packageId", req.PackageId, packageID)
			return odasdk.GetImportedPackageResponse{
				ImportedPackage: makeSDKImportedPackage(odaInstanceID, packageID, resource, odasdk.ImportedPackageStatusReady),
			}, nil
		},
		deleteFn: func(_ context.Context, req odasdk.DeleteImportedPackageRequest) (odasdk.DeleteImportedPackageResponse, error) {
			requireImportedPackageStringPtr(t, "delete packageId", req.PackageId, packageID)
			return odasdk.DeleteImportedPackageResponse{OpcRequestId: common.String("opc-delete-ready")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while OCI readback still reports READY")
	}
	if getCalls != 2 {
		t.Fatalf("GetImportedPackage() calls = %d, want pre-delete and confirm-delete reads", getCalls)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil while READY readback is still visible")
	}
}

func TestImportedPackageDeleteConfirmsReadNotFound(t *testing.T) {
	t.Parallel()

	const odaInstanceID = "ocid1.odainstance.oc1..delete-gone"
	const packageID = "ocid1.odapackage.oc1..delete-gone"
	resource := makeImportedPackageResource(odaInstanceID, packageID)
	resource.Status.OdaInstanceId = odaInstanceID
	resource.Status.OsokStatus.Ocid = shared.OCID(packageID)
	getCalls := 0

	client := newTestImportedPackageClient(&fakeImportedPackageOCIClient{
		getFn: func(_ context.Context, req odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
			getCalls++
			requireImportedPackageStringPtr(t, "get packageId", req.PackageId, packageID)
			if getCalls > 1 {
				return odasdk.GetImportedPackageResponse{}, errortest.NewServiceError(404, "NotFound", "ImportedPackage deleted")
			}
			return odasdk.GetImportedPackageResponse{
				ImportedPackage: makeSDKImportedPackage(odaInstanceID, packageID, resource, odasdk.ImportedPackageStatusReady),
			}, nil
		},
		deleteFn: func(_ context.Context, req odasdk.DeleteImportedPackageRequest) (odasdk.DeleteImportedPackageResponse, error) {
			requireImportedPackageStringPtr(t, "delete packageId", req.PackageId, packageID)
			return odasdk.DeleteImportedPackageResponse{OpcRequestId: common.String("opc-delete-2")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after confirm read reports NotFound")
	}
	if getCalls != 2 {
		t.Fatalf("GetImportedPackage() calls = %d, want pre-delete and confirm-delete reads", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
}

func newTestImportedPackageClient(client importedPackageOCIClient) ImportedPackageServiceClient {
	if client == nil {
		client = &fakeImportedPackageOCIClient{}
	}
	hooks := newImportedPackageRuntimeHooksWithOCIClient(client)
	manager := &ImportedPackageServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}}
	applyImportedPackageRuntimeHooks(manager, &hooks)
	return newImportedPackageAdaptedGeneratedClient(manager, hooks)
}

func newImportedPackageRuntimeHooksWithOCIClient(client importedPackageOCIClient) ImportedPackageRuntimeHooks {
	return ImportedPackageRuntimeHooks{
		Semantics: newImportedPackageRuntimeSemantics(),
		Identity:  generatedruntime.IdentityHooks[*odav1beta1.ImportedPackage]{},
		Create: runtimeOperationHooks[odasdk.CreateImportedPackageRequest, odasdk.CreateImportedPackageResponse]{
			Fields: importedPackageCreateFields(),
			Call: func(ctx context.Context, request odasdk.CreateImportedPackageRequest) (odasdk.CreateImportedPackageResponse, error) {
				return client.CreateImportedPackage(ctx, request)
			},
		},
		Get: runtimeOperationHooks[odasdk.GetImportedPackageRequest, odasdk.GetImportedPackageResponse]{
			Fields: importedPackageGetFields(),
			Call: func(ctx context.Context, request odasdk.GetImportedPackageRequest) (odasdk.GetImportedPackageResponse, error) {
				return client.GetImportedPackage(ctx, request)
			},
		},
		List: runtimeOperationHooks[odasdk.ListImportedPackagesRequest, odasdk.ListImportedPackagesResponse]{
			Fields: importedPackageListFields(),
			Call: func(ctx context.Context, request odasdk.ListImportedPackagesRequest) (odasdk.ListImportedPackagesResponse, error) {
				return client.ListImportedPackages(ctx, request)
			},
		},
		Update: runtimeOperationHooks[odasdk.UpdateImportedPackageRequest, odasdk.UpdateImportedPackageResponse]{
			Fields: importedPackageUpdateFields(),
			Call: func(ctx context.Context, request odasdk.UpdateImportedPackageRequest) (odasdk.UpdateImportedPackageResponse, error) {
				return client.UpdateImportedPackage(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[odasdk.DeleteImportedPackageRequest, odasdk.DeleteImportedPackageResponse]{
			Fields: importedPackageDeleteFields(),
			Call: func(ctx context.Context, request odasdk.DeleteImportedPackageRequest) (odasdk.DeleteImportedPackageResponse, error) {
				return client.DeleteImportedPackage(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ImportedPackageServiceClient) ImportedPackageServiceClient{},
	}
}

func makeImportedPackageResource(odaInstanceID string, packageID string) *odav1beta1.ImportedPackage {
	resource := &odav1beta1.ImportedPackage{
		Spec: odav1beta1.ImportedPackageSpec{
			CurrentPackageId: packageID,
		},
	}
	if odaInstanceID != "" {
		resource.Annotations = map[string]string{importedPackageOdaInstanceIDAnnotation: odaInstanceID}
	}
	return resource
}

func makeSDKImportedPackage(odaInstanceID string, packageID string, resource *odav1beta1.ImportedPackage, state odasdk.ImportedPackageStatusEnum) odasdk.ImportedPackage {
	return odasdk.ImportedPackage{
		OdaInstanceId:    common.String(odaInstanceID),
		CurrentPackageId: common.String(packageID),
		Name:             common.String("osok-imported-package"),
		DisplayName:      common.String("OSOK Imported Package"),
		Version:          common.String("1.0.0"),
		Status:           state,
		StatusMessage:    common.String("state " + string(state)),
		ParameterValues:  cloneImportedPackageStringMap(resource.Spec.ParameterValues),
		FreeformTags:     cloneImportedPackageStringMap(resource.Spec.FreeformTags),
		DefinedTags:      mapValueToOCIDefinedTags(resource.Spec.DefinedTags),
	}
}

func makeSDKImportedPackageSummary(odaInstanceID string, packageID string, resource *odav1beta1.ImportedPackage, state odasdk.ImportedPackageStatusEnum) odasdk.ImportedPackageSummary {
	return odasdk.ImportedPackageSummary{
		OdaInstanceId:    common.String(odaInstanceID),
		CurrentPackageId: common.String(packageID),
		Name:             common.String("osok-imported-package"),
		DisplayName:      common.String("OSOK Imported Package"),
		Version:          common.String("1.0.0"),
		Status:           state,
		FreeformTags:     cloneImportedPackageStringMap(resource.Spec.FreeformTags),
		DefinedTags:      mapValueToOCIDefinedTags(resource.Spec.DefinedTags),
	}
}

func mapValueToOCIDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, tags := range source {
		converted[namespace] = make(map[string]interface{}, len(tags))
		for key, value := range tags {
			converted[namespace][key] = value
		}
	}
	return converted
}

func requireImportedPackageStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func assertImportedPackageStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}
