package queue

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	queuesdk "github.com/oracle/oci-go-sdk/v65/queue"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestQueueRuntimeWorkRequestPollErrorsUseMatrix(t *testing.T) {
	t.Parallel()

	registration := errortest.ReviewedRegistrationForFamily(
		t,
		"queue",
		"Queue",
		errortest.APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
	)
	if !strings.Contains(registration.Deviation, "work-request") {
		t.Fatalf("reviewed registration = %s, want explicit work-request note", errortest.DescribeReviewedRegistration(registration))
	}

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantErrorType: focused["notfound"].NormalizedType},
		{Candidate: focused["auth404"], WantErrorType: focused["auth404"].NormalizedType},
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	t.Run("create", func(t *testing.T) {
		errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
			manager := newQueueTestManager(&fakeQueueOCIClient{
				createFn: func(_ context.Context, _ queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error) {
					return queuesdk.CreateQueueResponse{
						OpcWorkRequestId: common.String("wr-create-matrix"),
					}, nil
				},
				getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
					return queuesdk.GetWorkRequestResponse{}, errortest.NewServiceErrorFromCase(candidate)
				},
			})

			response, err := manager.CreateOrUpdate(context.Background(), makeSpecQueue(), ctrl.Request{})
			return errortest.AsyncFollowUpResult{
				Err:        err,
				Successful: response.IsSuccessful,
				Requeue:    response.ShouldRequeue,
			}
		})
	})

	t.Run("update", func(t *testing.T) {
		errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
			manager := newQueueTestManager(&fakeQueueOCIClient{
				getFn: func(_ context.Context, _ queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
					return queuesdk.GetQueueResponse{
						Queue: makeSDKQueue("ocid1.queue.oc1..existing", "old-name", queuesdk.QueueLifecycleStateActive),
					}, nil
				},
				updateFn: func(_ context.Context, _ queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
					return queuesdk.UpdateQueueResponse{
						OpcWorkRequestId: common.String("wr-update-matrix"),
					}, nil
				},
				getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
					return queuesdk.GetWorkRequestResponse{}, errortest.NewServiceErrorFromCase(candidate)
				},
			})

			resource := makeSpecQueue()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
			resource.Spec.DisplayName = "queue-sample"

			response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			return errortest.AsyncFollowUpResult{
				Err:        err,
				Successful: response.IsSuccessful,
				Requeue:    response.ShouldRequeue,
			}
		})
	})
}

func TestQueueRuntimeCreateFollowUpReadAfterSucceededWorkRequestUsesMatrix(t *testing.T) {
	t.Parallel()

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantSuccessful: true, WantRequeue: true},
		{Candidate: focused["auth404"], WantErrorType: focused["auth404"].NormalizedType},
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		manager := newQueueTestManager(&fakeQueueOCIClient{
			createFn: func(_ context.Context, _ queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error) {
				return queuesdk.CreateQueueResponse{
					OpcWorkRequestId: common.String("wr-create-followup"),
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
				return queuesdk.GetWorkRequestResponse{
					WorkRequest: makeWorkRequest(
						"wr-create-followup",
						queuesdk.OperationStatusSucceeded,
						queuesdk.ActionTypeCreated,
						"ocid1.queue.oc1..created",
					),
				}, nil
			},
			getFn: func(_ context.Context, _ queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
				return queuesdk.GetQueueResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})

		resource := makeSpecQueue()
		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if err == nil {
			if got := resource.Status.CreateWorkRequestId; got != "wr-create-followup" {
				t.Fatalf("status.createWorkRequestId = %q, want wr-create-followup while create follow-up keeps waiting", got)
			}
			if !strings.Contains(resource.Status.OsokStatus.Message, "waiting for Queue ocid1.queue.oc1..created") {
				t.Fatalf("status.message = %q, want follow-up waiting message", resource.Status.OsokStatus.Message)
			}
		}
		return errortest.AsyncFollowUpResult{
			Err:        err,
			Successful: response.IsSuccessful,
			Requeue:    response.ShouldRequeue,
		}
	})
}

func TestQueueRuntimeUpdateFollowUpReadAfterSucceededWorkRequestUsesMatrix(t *testing.T) {
	t.Parallel()

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantErrorSubstring: "is no longer readable"},
		{Candidate: focused["auth404"], WantErrorType: focused["auth404"].NormalizedType},
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		getCalls := 0
		manager := newQueueTestManager(&fakeQueueOCIClient{
			getFn: func(_ context.Context, _ queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
				getCalls++
				if getCalls == 1 {
					return queuesdk.GetQueueResponse{
						Queue: makeSDKQueue("ocid1.queue.oc1..existing", "old-name", queuesdk.QueueLifecycleStateActive),
					}, nil
				}
				return queuesdk.GetQueueResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
			updateFn: func(_ context.Context, _ queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
				return queuesdk.UpdateQueueResponse{
					OpcWorkRequestId: common.String("wr-update-followup"),
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
				return queuesdk.GetWorkRequestResponse{
					WorkRequest: makeWorkRequest(
						"wr-update-followup",
						queuesdk.OperationStatusSucceeded,
						queuesdk.ActionTypeUpdated,
						"ocid1.queue.oc1..existing",
					),
				}, nil
			},
		})

		resource := makeSpecQueue()
		resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
		resource.Spec.DisplayName = "queue-sample"

		response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.AsyncFollowUpResult{
			Err:        err,
			Successful: response.IsSuccessful,
			Requeue:    response.ShouldRequeue,
		}
	})
}

func TestQueueRuntimeDeleteWorkRequestPoll404UsesQueueConfirmation(t *testing.T) {
	t.Parallel()

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantDeleted: true},
		{Candidate: focused["auth404"], WantDeleted: true},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		manager := newQueueTestManager(&fakeQueueOCIClient{
			getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
				return queuesdk.GetWorkRequestResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
			getFn: func(_ context.Context, _ queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
				return queuesdk.GetQueueResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})

		resource := makeSpecQueue()
		resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
		resource.Status.DeleteWorkRequestId = "wr-delete-missing"

		deleted, err := manager.Delete(context.Background(), resource)
		if err == nil && deleted && resource.Status.OsokStatus.DeletedAt == nil {
			t.Fatal("status.deletedAt should be set after queue delete confirmation succeeds")
		}
		return errortest.AsyncFollowUpResult{
			Err:     err,
			Deleted: deleted,
		}
	})
}

func TestQueueRuntimeDeleteWorkRequestPoll404KeepsWaitingWhileQueueExists(t *testing.T) {
	t.Parallel()

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"]},
		{Candidate: focused["auth404"]},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		manager := newQueueTestManager(&fakeQueueOCIClient{
			getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
				return queuesdk.GetWorkRequestResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
			getFn: func(_ context.Context, _ queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
				return queuesdk.GetQueueResponse{
					Queue: makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateActive),
				}, nil
			},
		})

		resource := makeSpecQueue()
		resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
		resource.Status.DeleteWorkRequestId = "wr-delete-still-visible"

		deleted, err := manager.Delete(context.Background(), resource)
		if err == nil {
			if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
				t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
			}
			if !strings.Contains(resource.Status.OsokStatus.Message, "waiting for Queue ocid1.queue.oc1..existing to disappear") {
				t.Fatalf("status.message = %q, want waiting-to-disappear detail", resource.Status.OsokStatus.Message)
			}
		}
		return errortest.AsyncFollowUpResult{
			Err:     err,
			Deleted: deleted,
		}
	})
}

func TestQueueRuntimeDeleteWorkRequestPollConflictAndServerErrorsUseMatrix(t *testing.T) {
	t.Parallel()

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		manager := newQueueTestManager(&fakeQueueOCIClient{
			getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
				return queuesdk.GetWorkRequestResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})

		resource := makeSpecQueue()
		resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
		resource.Status.DeleteWorkRequestId = "wr-delete-error"

		deleted, err := manager.Delete(context.Background(), resource)
		return errortest.AsyncFollowUpResult{
			Err:     err,
			Deleted: deleted,
		}
	})
}

func TestQueueRuntimeDeleteSucceededWorkRequestConfirmation404UsesMatrix(t *testing.T) {
	t.Parallel()

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantDeleted: true},
		{Candidate: focused["auth404"], WantDeleted: true},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		manager := newQueueTestManager(&fakeQueueOCIClient{
			getWorkRequestFn: func(_ context.Context, _ queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
				return queuesdk.GetWorkRequestResponse{
					WorkRequest: makeWorkRequest(
						"wr-delete-succeeded",
						queuesdk.OperationStatusSucceeded,
						queuesdk.ActionTypeDeleted,
						"ocid1.queue.oc1..existing",
					),
				}, nil
			},
			getFn: func(_ context.Context, _ queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
				return queuesdk.GetQueueResponse{}, errortest.NewServiceErrorFromCase(candidate)
			},
		})

		resource := makeSpecQueue()
		resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
		resource.Status.DeleteWorkRequestId = "wr-delete-succeeded"

		deleted, err := manager.Delete(context.Background(), resource)
		return errortest.AsyncFollowUpResult{
			Err:     err,
			Deleted: deleted,
		}
	})
}
