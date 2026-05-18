/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alertpolicyrule

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeAlertPolicyRuleOCIClient struct {
	createRequests         []datasafesdk.CreateAlertPolicyRuleRequest
	getRequests            []datasafesdk.GetAlertPolicyRuleRequest
	listRequests           []datasafesdk.ListAlertPolicyRulesRequest
	updateRequests         []datasafesdk.UpdateAlertPolicyRuleRequest
	deleteRequests         []datasafesdk.DeleteAlertPolicyRuleRequest
	getWorkRequestRequests []datasafesdk.GetWorkRequestRequest

	create         func(context.Context, datasafesdk.CreateAlertPolicyRuleRequest) (datasafesdk.CreateAlertPolicyRuleResponse, error)
	get            func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error)
	list           func(context.Context, datasafesdk.ListAlertPolicyRulesRequest) (datasafesdk.ListAlertPolicyRulesResponse, error)
	update         func(context.Context, datasafesdk.UpdateAlertPolicyRuleRequest) (datasafesdk.UpdateAlertPolicyRuleResponse, error)
	delete         func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error)
	getWorkRequest func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

func (f *fakeAlertPolicyRuleOCIClient) CreateAlertPolicyRule(
	ctx context.Context,
	request datasafesdk.CreateAlertPolicyRuleRequest,
) (datasafesdk.CreateAlertPolicyRuleResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return datasafesdk.CreateAlertPolicyRuleResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakeAlertPolicyRuleOCIClient) GetAlertPolicyRule(
	ctx context.Context,
	request datasafesdk.GetAlertPolicyRuleRequest,
) (datasafesdk.GetAlertPolicyRuleResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return datasafesdk.GetAlertPolicyRuleResponse{}, nil
	}
	return f.get(ctx, request)
}

func (f *fakeAlertPolicyRuleOCIClient) ListAlertPolicyRules(
	ctx context.Context,
	request datasafesdk.ListAlertPolicyRulesRequest,
) (datasafesdk.ListAlertPolicyRulesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return datasafesdk.ListAlertPolicyRulesResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakeAlertPolicyRuleOCIClient) UpdateAlertPolicyRule(
	ctx context.Context,
	request datasafesdk.UpdateAlertPolicyRuleRequest,
) (datasafesdk.UpdateAlertPolicyRuleResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return datasafesdk.UpdateAlertPolicyRuleResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakeAlertPolicyRuleOCIClient) DeleteAlertPolicyRule(
	ctx context.Context,
	request datasafesdk.DeleteAlertPolicyRuleRequest,
) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return datasafesdk.DeleteAlertPolicyRuleResponse{}, nil
	}
	return f.delete(ctx, request)
}

func (f *fakeAlertPolicyRuleOCIClient) GetWorkRequest(
	ctx context.Context,
	request datasafesdk.GetWorkRequestRequest,
) (datasafesdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequest == nil {
		return datasafesdk.GetWorkRequestResponse{}, nil
	}
	return f.getWorkRequest(ctx, request)
}

func TestAlertPolicyRuleCreateRecordsRuleKeyAndRequestID(t *testing.T) {
	resource := testAlertPolicyRule()
	fake := &fakeAlertPolicyRuleOCIClient{
		create: func(_ context.Context, request datasafesdk.CreateAlertPolicyRuleRequest) (datasafesdk.CreateAlertPolicyRuleResponse, error) {
			assertAlertPolicyRuleCreateRequest(t, request)
			return datasafesdk.CreateAlertPolicyRuleResponse{
				AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				OpcRequestId:    common.String("opc-create"),
			}, nil
		},
	}

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	assertAlertPolicyRuleCreateStatus(t, resource)
}

func TestAlertPolicyRuleCreateKeepsPendingWorkRequestWhenReadbackIsActive(t *testing.T) {
	resource := testAlertPolicyRule()
	fake := &fakeAlertPolicyRuleOCIClient{
		create: func(context.Context, datasafesdk.CreateAlertPolicyRuleRequest) (datasafesdk.CreateAlertPolicyRuleResponse, error) {
			return datasafesdk.CreateAlertPolicyRuleResponse{
				AlertPolicyRule:  activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		getWorkRequest: func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			if got, want := alertPolicyRuleString(request.WorkRequestId), "wr-create"; got != want {
				t.Fatalf("GetWorkRequest WorkRequestId = %q, want %q", got, want)
			}
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: alertPolicyRuleWorkRequest("wr-create", datasafesdk.WorkRequestStatusInProgress, datasafesdk.WorkRequestOperationTypeCreateAlertPolicyRule),
			}, nil
		},
	}

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while work request is pending", response)
	}
	assertAlertPolicyRuleAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
}

func TestAlertPolicyRuleBindUsesPaginatedListBeforeCreate(t *testing.T) {
	resource := testAlertPolicyRule()
	fake := &fakeAlertPolicyRuleOCIClient{
		list: func(_ context.Context, request datasafesdk.ListAlertPolicyRulesRequest) (datasafesdk.ListAlertPolicyRulesResponse, error) {
			switch page := alertPolicyRuleString(request.Page); page {
			case "":
				return datasafesdk.ListAlertPolicyRulesResponse{
					AlertPolicyRuleCollection: datasafesdk.AlertPolicyRuleCollection{
						Items: []datasafesdk.AlertPolicyRuleSummary{
							alertPolicyRuleSummary("other", "severity == 'LOW'", "", ""),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return datasafesdk.ListAlertPolicyRulesResponse{
					AlertPolicyRuleCollection: datasafesdk.AlertPolicyRuleCollection{
						Items: []datasafesdk.AlertPolicyRuleSummary{
							alertPolicyRuleSummary("rule-1", "severity == 'HIGH'", "initial", "high severity"),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected list page = %q", page)
				return datasafesdk.ListAlertPolicyRulesResponse{}, nil
			}
		},
		create: func(context.Context, datasafesdk.CreateAlertPolicyRuleRequest) (datasafesdk.CreateAlertPolicyRuleResponse, error) {
			t.Fatal("CreateAlertPolicyRule called for existing rule")
			return datasafesdk.CreateAlertPolicyRuleResponse{}, nil
		},
	}

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got, want := len(fake.listRequests), 2; got != want {
		t.Fatalf("ListAlertPolicyRules calls = %d, want %d", got, want)
	}
	if got := len(fake.createRequests); got != 0 {
		t.Fatalf("CreateAlertPolicyRule calls = %d, want 0", got)
	}
	if got, want := resource.Status.Key, "rule-1"; got != want {
		t.Fatalf("status.key = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleNoOpReconcileDoesNotUpdate(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(_ context.Context, request datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			if got, want := alertPolicyRuleString(request.RuleKey), "rule-1"; got != want {
				t.Fatalf("GetAlertPolicyRule RuleKey = %q, want %q", got, want)
			}
			return datasafesdk.GetAlertPolicyRuleResponse{
				AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
			}, nil
		},
		update: func(context.Context, datasafesdk.UpdateAlertPolicyRuleRequest) (datasafesdk.UpdateAlertPolicyRuleResponse, error) {
			t.Fatal("UpdateAlertPolicyRule called for matching desired state")
			return datasafesdk.UpdateAlertPolicyRuleResponse{}, nil
		},
	}

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateAlertPolicyRule calls = %d, want 0", got)
	}
}

func TestAlertPolicyRulePendingCreateWorkRequestBlocksFollowUpUpdate(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	resource.Spec.Description = "changed while create still pending"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		RawStatus:       string(datasafesdk.WorkRequestStatusInProgress),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			return datasafesdk.GetAlertPolicyRuleResponse{
				AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
			}, nil
		},
		getWorkRequest: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: alertPolicyRuleWorkRequest("wr-create", datasafesdk.WorkRequestStatusInProgress, datasafesdk.WorkRequestOperationTypeCreateAlertPolicyRule),
			}, nil
		},
		update: func(context.Context, datasafesdk.UpdateAlertPolicyRuleRequest) (datasafesdk.UpdateAlertPolicyRuleResponse, error) {
			t.Fatal("UpdateAlertPolicyRule called while create work request is pending")
			return datasafesdk.UpdateAlertPolicyRuleResponse{}, nil
		},
	}

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while create is pending", response)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("UpdateAlertPolicyRule calls = %d, want 0", got)
	}
	assertAlertPolicyRuleAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
}

func TestAlertPolicyRuleMutableUpdateRefreshesStatus(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	resource.Spec.Expression = "severity == 'CRITICAL'"
	resource.Spec.Description = "updated"
	resource.Spec.DisplayName = "critical severity"
	fake := newMutableUpdateAlertPolicyRuleFake(t)

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while update is pending", response)
	}
	assertAlertPolicyRuleUpdatePendingStatus(t, resource)
}

func TestAlertPolicyRuleMutableUpdateCanClearDescriptionAndDisplayName(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	resource.Spec.Description = ""
	resource.Spec.DisplayName = ""
	fake := newClearMutableAlertPolicyRuleFake(t)

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue after clear", response)
	}
	assertAlertPolicyRuleClearUpdateStatus(t, resource)
}

func TestAlertPolicyRuleUpdateKeepsPendingWorkRequestWhenReadbackIsActive(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	resource.Spec.Description = "updated"
	getCalls := 0
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAlertPolicyRuleResponse{
					AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				}, nil
			}
			return datasafesdk.GetAlertPolicyRuleResponse{
				AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "updated", "high severity"),
			}, nil
		},
		update: func(context.Context, datasafesdk.UpdateAlertPolicyRuleRequest) (datasafesdk.UpdateAlertPolicyRuleResponse, error) {
			return datasafesdk.UpdateAlertPolicyRuleResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			}, nil
		},
		getWorkRequest: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: alertPolicyRuleWorkRequest("wr-update", datasafesdk.WorkRequestStatusInProgress, datasafesdk.WorkRequestOperationTypeUpdateAlertPolicyRule),
			}, nil
		},
	}

	response, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while update work request is pending", response)
	}
	assertAlertPolicyRuleAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update", shared.OSOKAsyncClassPending)
}

func TestAlertPolicyRuleRejectsParentAnnotationDrift(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	resource.Annotations[alertPolicyRuleAlertPolicyIDAnnotation] = "policy-2"
	fake := &fakeAlertPolicyRuleOCIClient{}

	_, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want parent drift error")
	}
	if !strings.Contains(err.Error(), "create-only parent annotation") {
		t.Fatalf("CreateOrUpdate() error = %q, want parent drift error", err)
	}
	if len(fake.getRequests)+len(fake.listRequests)+len(fake.createRequests)+len(fake.updateRequests) != 0 {
		t.Fatalf("OCI calls = get:%d list:%d create:%d update:%d, want none", len(fake.getRequests), len(fake.listRequests), len(fake.createRequests), len(fake.updateRequests))
	}
	if got, want := lastAlertPolicyRuleCondition(resource), shared.Failed; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleDeleteRetainsFinalizerWhileReadbackDeletes(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	getCalls := 0
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(_ context.Context, _ datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAlertPolicyRuleResponse{
					AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				}, nil
			}
			return datasafesdk.GetAlertPolicyRuleResponse{
				AlertPolicyRule: datasafesdk.AlertPolicyRule{
					Key:            common.String("rule-1"),
					Expression:     common.String("severity == 'HIGH'"),
					LifecycleState: datasafesdk.AlertPolicyRuleLifecycleStateDeleting,
				},
			}, nil
		},
		delete: func(_ context.Context, request datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			if got, want := alertPolicyRuleString(request.AlertPolicyId), "policy-1"; got != want {
				t.Fatalf("DeleteAlertPolicyRule AlertPolicyId = %q, want %q", got, want)
			}
			if got, want := alertPolicyRuleString(request.RuleKey), "rule-1"; got != want {
				t.Fatalf("DeleteAlertPolicyRule RuleKey = %q, want %q", got, want)
			}
			return datasafesdk.DeleteAlertPolicyRuleResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
		getWorkRequest: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: alertPolicyRuleWorkRequest("wr-delete", datasafesdk.WorkRequestStatusInProgress, datasafesdk.WorkRequestOperationTypeDeleteAlertPolicyRule),
			}, nil
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI reports DELETING")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async current = %#v, want pending delete", current)
	}
}

func TestAlertPolicyRuleDeleteDoesNotReissueWhileWorkRequestPending(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	getCalls := 0
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			return datasafesdk.GetAlertPolicyRuleResponse{
				AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
			}, nil
		},
		delete: func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			return datasafesdk.DeleteAlertPolicyRuleResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
		getWorkRequest: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: alertPolicyRuleWorkRequest("wr-delete", datasafesdk.WorkRequestStatusInProgress, datasafesdk.WorkRequestOperationTypeDeleteAlertPolicyRule),
			}, nil
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("first Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("first Delete() deleted = true, want false while delete work request is pending")
	}
	deleted, err = newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("second Delete() deleted = true, want false while delete work request is pending")
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("DeleteAlertPolicyRule calls = %d, want %d", got, want)
	}
	if got, want := getCalls, 1; got != want {
		t.Fatalf("GetAlertPolicyRule calls = %d, want %d", got, want)
	}
	assertAlertPolicyRuleAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
}

func TestAlertPolicyRuleDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	getCalls := 0
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(_ context.Context, _ datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAlertPolicyRuleResponse{
					AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				}, nil
			}
			return datasafesdk.GetAlertPolicyRuleResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		delete: func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			return datasafesdk.DeleteAlertPolicyRuleResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleDeleteConflictConfirmsUnambiguousNotFound(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	getCalls := 0
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAlertPolicyRuleResponse{
					AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				}, nil
			}
			err := errortest.NewServiceError(404, errorutil.NotFound, "deleted after retryable conflict")
			err.OpcRequestID = "opc-confirm-notfound"
			return datasafesdk.GetAlertPolicyRuleResponse{}, err
		},
		delete: func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			err := errortest.NewServiceError(409, errorutil.IncorrectState, "delete conflict")
			err.OpcRequestID = "opc-delete-conflict"
			return datasafesdk.DeleteAlertPolicyRuleResponse{}, err
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found confirms retryable conflict")
	}
	if got, want := len(fake.getRequests), 2; got != want {
		t.Fatalf("GetAlertPolicyRule calls = %d, want %d", got, want)
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("DeleteAlertPolicyRule calls = %d, want %d", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-confirm-notfound"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleDeleteConflictKeepsFinalizerOnAuthShapedConfirmRead(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	getCalls := 0
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAlertPolicyRuleResponse{
					AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				}, nil
			}
			err := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous confirm read")
			err.OpcRequestID = "opc-auth-confirm"
			return datasafesdk.GetAlertPolicyRuleResponse{}, err
		},
		delete: func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			return datasafesdk.DeleteAlertPolicyRuleResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "delete conflict")
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound confirm read", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous confirm read")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set, want finalizer retained")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-auth-confirm"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleDeleteKeepsFinalizerOnAuthShapedNotFound(t *testing.T) {
	resource := testTrackedAlertPolicyRule("policy-1", "rule-1")
	fake := &fakeAlertPolicyRuleOCIClient{
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			err := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
			err.OpcRequestID = "opc-auth"
			return datasafesdk.GetAlertPolicyRuleResponse{}, err
		},
		delete: func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			t.Fatal("DeleteAlertPolicyRule called after auth-shaped confirm read")
			return datasafesdk.DeleteAlertPolicyRuleResponse{}, nil
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped readback error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous NotAuthorizedOrNotFound")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("DeleteAlertPolicyRule calls = %d, want 0", got)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-auth"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleDeleteWithParentOnlyConfirmsAbsenceByList(t *testing.T) {
	resource := testAlertPolicyRule()
	fake := &fakeAlertPolicyRuleOCIClient{
		list: func(_ context.Context, request datasafesdk.ListAlertPolicyRulesRequest) (datasafesdk.ListAlertPolicyRulesResponse, error) {
			switch page := alertPolicyRuleString(request.Page); page {
			case "":
				return datasafesdk.ListAlertPolicyRulesResponse{
					AlertPolicyRuleCollection: datasafesdk.AlertPolicyRuleCollection{
						Items: []datasafesdk.AlertPolicyRuleSummary{
							alertPolicyRuleSummary("other", "severity == 'LOW'", "", ""),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return datasafesdk.ListAlertPolicyRulesResponse{}, nil
			default:
				t.Fatalf("unexpected list page = %q", page)
				return datasafesdk.ListAlertPolicyRulesResponse{}, nil
			}
		},
		delete: func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			t.Fatal("DeleteAlertPolicyRule called when parent-scoped list found no desired rule")
			return datasafesdk.DeleteAlertPolicyRuleResponse{}, nil
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after parent-scoped list confirms absence")
	}
	if got, want := len(fake.listRequests), 2; got != want {
		t.Fatalf("ListAlertPolicyRules calls = %d, want %d", got, want)
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("DeleteAlertPolicyRule calls = %d, want 0", got)
	}
}

func TestAlertPolicyRuleDeleteWithParentOnlyDeletesSingleListMatch(t *testing.T) {
	resource := testAlertPolicyRule()
	fake := &fakeAlertPolicyRuleOCIClient{
		list: func(context.Context, datasafesdk.ListAlertPolicyRulesRequest) (datasafesdk.ListAlertPolicyRulesResponse, error) {
			return datasafesdk.ListAlertPolicyRulesResponse{
				AlertPolicyRuleCollection: datasafesdk.AlertPolicyRuleCollection{
					Items: []datasafesdk.AlertPolicyRuleSummary{
						alertPolicyRuleSummary("rule-1", "severity == 'HIGH'", "initial", "high severity"),
					},
				},
			}, nil
		},
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			return datasafesdk.GetAlertPolicyRuleResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		delete: func(_ context.Context, request datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			assertAlertPolicyRuleDeleteRequest(t, request)
			return datasafesdk.DeleteAlertPolicyRuleResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after deleting discovered match")
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("DeleteAlertPolicyRule calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleDeleteWithParentOnlyRetainsFinalizerOnAmbiguousList(t *testing.T) {
	resource := testAlertPolicyRule()
	fake := &fakeAlertPolicyRuleOCIClient{
		list: func(context.Context, datasafesdk.ListAlertPolicyRulesRequest) (datasafesdk.ListAlertPolicyRulesResponse, error) {
			return datasafesdk.ListAlertPolicyRulesResponse{
				AlertPolicyRuleCollection: datasafesdk.AlertPolicyRuleCollection{
					Items: []datasafesdk.AlertPolicyRuleSummary{
						alertPolicyRuleSummary("rule-1", "severity == 'HIGH'", "initial", "high severity"),
						alertPolicyRuleSummary("rule-2", "severity == 'HIGH'", "initial", "high severity"),
					},
				},
			}, nil
		},
		delete: func(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error) {
			t.Fatal("DeleteAlertPolicyRule called when parent-scoped list was ambiguous")
			return datasafesdk.DeleteAlertPolicyRuleResponse{}, nil
		},
	}

	deleted, err := newTestAlertPolicyRuleClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous list error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous parent-scoped list")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("DeleteAlertPolicyRule calls = %d, want 0", got)
	}
	if got, want := lastAlertPolicyRuleCondition(resource), shared.Failed; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func TestAlertPolicyRuleCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := testAlertPolicyRule()
	fake := &fakeAlertPolicyRuleOCIClient{
		create: func(context.Context, datasafesdk.CreateAlertPolicyRuleRequest) (datasafesdk.CreateAlertPolicyRuleResponse, error) {
			err := errortest.NewServiceError(500, "InternalError", "create failed")
			err.OpcRequestID = "opc-create-error"
			return datasafesdk.CreateAlertPolicyRuleResponse{}, err
		},
	}

	_, err := newTestAlertPolicyRuleClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create error")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-error"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if got, want := lastAlertPolicyRuleCondition(resource), shared.Failed; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func assertAlertPolicyRuleCreateRequest(t *testing.T, request datasafesdk.CreateAlertPolicyRuleRequest) {
	t.Helper()

	if got, want := alertPolicyRuleString(request.AlertPolicyId), "policy-1"; got != want {
		t.Fatalf("CreateAlertPolicyRule AlertPolicyId = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.Expression), "severity == 'HIGH'"; got != want {
		t.Fatalf("CreateAlertPolicyRule Expression = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.Description), "initial"; got != want {
		t.Fatalf("CreateAlertPolicyRule Description = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.DisplayName), "high severity"; got != want {
		t.Fatalf("CreateAlertPolicyRule DisplayName = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.OpcRetryToken), "alertpolicyrule-uid"; got != want {
		t.Fatalf("CreateAlertPolicyRule OpcRetryToken = %q, want %q", got, want)
	}
}

func assertAlertPolicyRuleCreateStatus(t *testing.T, resource *datasafev1beta1.AlertPolicyRule) {
	t.Helper()

	if got, want := resource.Status.Key, "rule-1"; got != want {
		t.Fatalf("status.key = %q, want %q", got, want)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), string(formatAlertPolicyRuleTrackedID("policy-1", "rule-1")); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if got, want := lastAlertPolicyRuleCondition(resource), shared.Active; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func newMutableUpdateAlertPolicyRuleFake(t *testing.T) *fakeAlertPolicyRuleOCIClient {
	t.Helper()

	getCalls := 0
	return &fakeAlertPolicyRuleOCIClient{
		get: func(_ context.Context, _ datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAlertPolicyRuleResponse{
					AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				}, nil
			}
			return datasafesdk.GetAlertPolicyRuleResponse{
				AlertPolicyRule: datasafesdk.AlertPolicyRule{
					Key:            common.String("rule-1"),
					Expression:     common.String("severity == 'CRITICAL'"),
					Description:    common.String("updated"),
					DisplayName:    common.String("critical severity"),
					LifecycleState: datasafesdk.AlertPolicyRuleLifecycleStateUpdating,
				},
			}, nil
		},
		update: func(_ context.Context, request datasafesdk.UpdateAlertPolicyRuleRequest) (datasafesdk.UpdateAlertPolicyRuleResponse, error) {
			assertAlertPolicyRuleUpdateRequest(t, request)
			return datasafesdk.UpdateAlertPolicyRuleResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			}, nil
		},
		getWorkRequest: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: alertPolicyRuleWorkRequest("wr-update", datasafesdk.WorkRequestStatusInProgress, datasafesdk.WorkRequestOperationTypeUpdateAlertPolicyRule),
			}, nil
		},
	}
}

func newClearMutableAlertPolicyRuleFake(t *testing.T) *fakeAlertPolicyRuleOCIClient {
	t.Helper()

	getCalls := 0
	return &fakeAlertPolicyRuleOCIClient{
		get: func(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetAlertPolicyRuleResponse{
					AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "initial", "high severity"),
				}, nil
			}
			return datasafesdk.GetAlertPolicyRuleResponse{
				AlertPolicyRule: activeAlertPolicyRule("rule-1", "severity == 'HIGH'", "", ""),
			}, nil
		},
		update: func(_ context.Context, request datasafesdk.UpdateAlertPolicyRuleRequest) (datasafesdk.UpdateAlertPolicyRuleResponse, error) {
			assertAlertPolicyRuleClearUpdateRequest(t, request)
			return datasafesdk.UpdateAlertPolicyRuleResponse{OpcRequestId: common.String("opc-update")}, nil
		},
	}
}

func assertAlertPolicyRuleUpdateRequest(t *testing.T, request datasafesdk.UpdateAlertPolicyRuleRequest) {
	t.Helper()

	if got, want := alertPolicyRuleString(request.AlertPolicyId), "policy-1"; got != want {
		t.Fatalf("UpdateAlertPolicyRule AlertPolicyId = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.RuleKey), "rule-1"; got != want {
		t.Fatalf("UpdateAlertPolicyRule RuleKey = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.Expression), "severity == 'CRITICAL'"; got != want {
		t.Fatalf("UpdateAlertPolicyRule Expression = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.Description), "updated"; got != want {
		t.Fatalf("UpdateAlertPolicyRule Description = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.DisplayName), "critical severity"; got != want {
		t.Fatalf("UpdateAlertPolicyRule DisplayName = %q, want %q", got, want)
	}
}

func assertAlertPolicyRuleClearUpdateRequest(t *testing.T, request datasafesdk.UpdateAlertPolicyRuleRequest) {
	t.Helper()

	if request.Description == nil || *request.Description != "" {
		t.Fatalf("UpdateAlertPolicyRule Description = %#v, want explicit empty string clear", request.Description)
	}
	if request.DisplayName == nil || *request.DisplayName != "" {
		t.Fatalf("UpdateAlertPolicyRule DisplayName = %#v, want explicit empty string clear", request.DisplayName)
	}
}

func assertAlertPolicyRuleUpdatePendingStatus(t *testing.T, resource *datasafev1beta1.AlertPolicyRule) {
	t.Helper()

	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending update")
	}
	if got, want := current.Phase, shared.OSOKAsyncPhaseUpdate; got != want {
		t.Fatalf("async phase = %q, want %q", got, want)
	}
	if got, want := current.WorkRequestID, "wr-update"; got != want {
		t.Fatalf("async workRequestId = %q, want %q", got, want)
	}
}

func assertAlertPolicyRuleClearUpdateStatus(t *testing.T, resource *datasafev1beta1.AlertPolicyRule) {
	t.Helper()

	if got, want := resource.Status.Description, ""; got != want {
		t.Fatalf("status.description = %q, want cleared string", got)
	}
	if got, want := resource.Status.DisplayName, ""; got != want {
		t.Fatalf("status.displayName = %q, want cleared string", got)
	}
}

func assertAlertPolicyRuleDeleteRequest(t *testing.T, request datasafesdk.DeleteAlertPolicyRuleRequest) {
	t.Helper()

	if got, want := alertPolicyRuleString(request.AlertPolicyId), "policy-1"; got != want {
		t.Fatalf("DeleteAlertPolicyRule AlertPolicyId = %q, want %q", got, want)
	}
	if got, want := alertPolicyRuleString(request.RuleKey), "rule-1"; got != want {
		t.Fatalf("DeleteAlertPolicyRule RuleKey = %q, want %q", got, want)
	}
}

func newTestAlertPolicyRuleClient(fake *fakeAlertPolicyRuleOCIClient) AlertPolicyRuleServiceClient {
	return newAlertPolicyRuleServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
}

func testAlertPolicyRule() *datasafev1beta1.AlertPolicyRule {
	return &datasafev1beta1.AlertPolicyRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rule",
			UID:  types.UID("alertpolicyrule-uid"),
			Annotations: map[string]string{
				alertPolicyRuleAlertPolicyIDAnnotation: "policy-1",
			},
		},
		Spec: datasafev1beta1.AlertPolicyRuleSpec{
			Expression:  "severity == 'HIGH'",
			Description: "initial",
			DisplayName: "high severity",
		},
	}
}

func testTrackedAlertPolicyRule(alertPolicyID string, ruleKey string) *datasafev1beta1.AlertPolicyRule {
	resource := testAlertPolicyRule()
	resource.Annotations[alertPolicyRuleAlertPolicyIDAnnotation] = alertPolicyID
	resource.Status.Key = ruleKey
	resource.Status.OsokStatus.Ocid = formatAlertPolicyRuleTrackedID(alertPolicyID, ruleKey)
	return resource
}

func activeAlertPolicyRule(key string, expression string, description string, displayName string) datasafesdk.AlertPolicyRule {
	return datasafesdk.AlertPolicyRule{
		Key:            common.String(key),
		Expression:     common.String(expression),
		Description:    common.String(description),
		DisplayName:    common.String(displayName),
		LifecycleState: datasafesdk.AlertPolicyRuleLifecycleStateActive,
	}
}

func alertPolicyRuleSummary(key string, expression string, description string, displayName string) datasafesdk.AlertPolicyRuleSummary {
	return datasafesdk.AlertPolicyRuleSummary{
		Key:            common.String(key),
		Expression:     common.String(expression),
		Description:    common.String(description),
		DisplayName:    common.String(displayName),
		LifecycleState: datasafesdk.AlertPolicyRuleLifecycleStateActive,
	}
}

func alertPolicyRuleWorkRequest(
	id string,
	status datasafesdk.WorkRequestStatusEnum,
	operation datasafesdk.WorkRequestOperationTypeEnum,
) datasafesdk.WorkRequest {
	return datasafesdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operation,
	}
}

func assertAlertPolicyRuleAsync(
	t *testing.T,
	resource *datasafev1beta1.AlertPolicyRule,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want async operation")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("async source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != wantPhase {
		t.Fatalf("async phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("async workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("async normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
}

func lastAlertPolicyRuleCondition(resource *datasafev1beta1.AlertPolicyRule) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}
