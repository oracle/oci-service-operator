package application

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	dataflowsdk "github.com/oracle/oci-go-sdk/v65/dataflow"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestApplicationPlainGeneratedRuntimeCreateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainCreateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		manager := newTestManager(&fakeApplicationOCIClient{
			createFn: func(_ context.Context, _ dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error) {
				return dataflowsdk.CreateApplicationResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})

		resource := makeSpecApplication()
		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestApplicationPlainGeneratedRuntimeUpdateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainUpdateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		manager := newTestManager(&fakeApplicationOCIClient{
			getFn: func(_ context.Context, _ dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
				return dataflowsdk.GetApplicationResponse{
					Application: makeSDKApplication("ocid1.dataflowapplication.oc1..matrix", "matrix-application", dataflowsdk.ApplicationLifecycleStateActive),
				}, nil
			},
			updateFn: func(_ context.Context, _ dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error) {
				return dataflowsdk.UpdateApplicationResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})

		resource := makeSpecApplication()
		resource.Status.Id = "ocid1.dataflowapplication.oc1..matrix"
		resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..matrix")
		resource.Spec.DisplayName = "updated-application"

		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestApplicationPlainGeneratedRuntimeDeleteErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainDeleteMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainDeleteResult {
		manager := newTestManager(&fakeApplicationOCIClient{
			deleteFn: func(_ context.Context, _ dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error) {
				return dataflowsdk.DeleteApplicationResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
			getFn: func(_ context.Context, req dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
				if req.ApplicationId == nil {
					t.Fatal("get applicationId = nil, want tracked Application ID")
				}
				return dataflowsdk.GetApplicationResponse{
					Application: makeSDKApplication(*req.ApplicationId, "matrix-application", dataflowsdk.ApplicationLifecycleStateActive),
				}, nil
			},
		})

		resource := makeSpecApplication()
		resource.Status.Id = "ocid1.dataflowapplication.oc1..matrix"
		resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..matrix")

		deleted, err := manager.Delete(context.Background(), resource)
		return errortest.GeneratedRuntimePlainDeleteResult{
			Deleted:      deleted,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestApplicationDeleteRetryableConflictMayResolveDeletedAfterConfirmRead(t *testing.T) {
	t.Parallel()

	candidate, ok := errortest.LookupCommonErrorCase(409, "IncorrectState")
	if !ok {
		t.Fatal("LookupCommonErrorCase(409, IncorrectState) = missing, want matrix row")
	}

	getCalls := 0
	manager := newTestManager(&fakeApplicationOCIClient{
		deleteFn: func(_ context.Context, _ dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error) {
			return dataflowsdk.DeleteApplicationResponse{}, errortest.NewServiceErrorFromCase(candidate)
		},
		getFn: func(_ context.Context, req dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error) {
			getCalls++
			if req.ApplicationId == nil {
				t.Fatal("get applicationId = nil, want tracked Application ID")
			}
			if getCalls == 1 {
				return dataflowsdk.GetApplicationResponse{
					Application: makeSDKApplication(*req.ApplicationId, "matrix-application", dataflowsdk.ApplicationLifecycleStateActive),
				}, nil
			}
			return dataflowsdk.GetApplicationResponse{}, errortest.NewServiceError(404, "NotFound", "delete confirm reread reports disappearance")
		},
	})

	resource := makeSpecApplication()
	resource.Status.Id = "ocid1.dataflowapplication.oc1..matrix"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.dataflowapplication.oc1..matrix")

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v, want deleted outcome after confirm reread reports NotFound", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after confirm reread reports NotFound")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want timestamp after confirm reread reports NotFound")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
}

func TestApplicationDeleteUsesMissingTrackedIDShortCircuit(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	manager := newTestManager(&fakeApplicationOCIClient{
		deleteFn: func(_ context.Context, _ dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error) {
			deleteCalled = true
			return dataflowsdk.DeleteApplicationResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	resource := makeSpecApplication()
	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v, want missing-ID short circuit", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want missing-ID short circuit")
	}
	if deleteCalled {
		t.Fatal("Delete() should not call OCI delete when no tracked ID is recorded")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want terminal timestamp")
	}
}
