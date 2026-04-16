package analyticsinstance

import (
	"context"
	"testing"

	analyticssdk "github.com/oracle/oci-go-sdk/v65/analytics"
	analyticsv1beta1 "github.com/oracle/oci-service-operator/api/analytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestAnalyticsInstancePlainGeneratedRuntimeCreateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainCreateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &analyticssdk.CreateAnalyticsInstanceRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return analyticssdk.CreateAnalyticsInstanceResponse{}, errortest.NewServiceErrorFromCase(candidate)
				},
				Fields: analyticsInstanceCreateFields(),
			},
		})

		resource := newAnalyticsInstanceTestResource()
		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestAnalyticsInstancePlainGeneratedRuntimeUpdateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainUpdateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		const existingID = "ocid1.analyticsinstance.oc1..matrix"

		resource := newExistingAnalyticsInstanceTestResource(existingID)
		resource.Spec.Description = "updated analytics description"

		current := observedAnalyticsInstanceFromSpec(existingID, newAnalyticsInstanceTestResource().Spec, "ACTIVE")

		manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return analyticssdk.GetAnalyticsInstanceResponse{AnalyticsInstance: current}, nil
				},
				Fields: analyticsInstanceGetFields(),
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &analyticssdk.UpdateAnalyticsInstanceRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return analyticssdk.UpdateAnalyticsInstanceResponse{}, errortest.NewServiceErrorFromCase(candidate)
				},
				Fields: analyticsInstanceUpdateFields(),
			},
		})

		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestAnalyticsInstancePlainGeneratedRuntimeDeleteErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainDeleteMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainDeleteResult {
		const existingID = "ocid1.analyticsinstance.oc1..matrix"

		resource := newExistingAnalyticsInstanceTestResource(existingID)
		getCalls := 0

		manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					getCalls++
					lifecycleState := "ACTIVE"
					if getCalls > 1 && errortest.GeneratedRuntimePlainDeleteRequiresConfirmRead(candidate) {
						lifecycleState = "DELETING"
					}
					return analyticssdk.GetAnalyticsInstanceResponse{
						AnalyticsInstance: observedAnalyticsInstanceFromSpec(existingID, resource.Spec, lifecycleState),
					}, nil
				},
				Fields: analyticsInstanceGetFields(),
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &analyticssdk.DeleteAnalyticsInstanceRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return analyticssdk.DeleteAnalyticsInstanceResponse{}, errortest.NewServiceErrorFromCase(candidate)
				},
				Fields: analyticsInstanceDeleteFields(),
			},
		})

		deleted, err := manager.Delete(context.Background(), resource)
		return errortest.GeneratedRuntimePlainDeleteResult{
			Deleted:      deleted,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestAnalyticsInstancePlainGeneratedRuntimeDeleteRetryableConflictMayResolveDeletedAfterConfirmRead(t *testing.T) {
	t.Parallel()

	candidate, ok := errortest.LookupCommonErrorCase(409, "IncorrectState")
	if !ok {
		t.Fatal("LookupCommonErrorCase(409, IncorrectState) = missing, want matrix row")
	}

	const existingID = "ocid1.analyticsinstance.oc1..matrix"

	resource := newExistingAnalyticsInstanceTestResource(existingID)
	getCalls := 0

	manager := newAnalyticsInstanceRuntimeTestManager(generatedruntime.Config[*analyticsv1beta1.AnalyticsInstance]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.GetAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				getCalls++
				if getCalls == 1 {
					return analyticssdk.GetAnalyticsInstanceResponse{
						AnalyticsInstance: observedAnalyticsInstanceFromSpec(existingID, resource.Spec, "ACTIVE"),
					}, nil
				}
				return nil, errortest.NewServiceError(404, "NotFound", "delete confirm reread reports disappearance")
			},
			Fields: analyticsInstanceGetFields(),
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &analyticssdk.DeleteAnalyticsInstanceRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return analyticssdk.DeleteAnalyticsInstanceResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
			Fields: analyticsInstanceDeleteFields(),
		},
	})

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
}
