/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package transcriptionjob

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/aispeech"
	"github.com/oracle/oci-go-sdk/v65/common"
	aispeechv1beta1 "github.com/oracle/oci-service-operator/api/aispeech/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeTranscriptionJobOCIClient struct {
	createTranscriptionJobFn func(context.Context, aispeech.CreateTranscriptionJobRequest) (aispeech.CreateTranscriptionJobResponse, error)
	getTranscriptionJobFn    func(context.Context, aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error)
	listTranscriptionJobsFn  func(context.Context, aispeech.ListTranscriptionJobsRequest) (aispeech.ListTranscriptionJobsResponse, error)
	updateTranscriptionJobFn func(context.Context, aispeech.UpdateTranscriptionJobRequest) (aispeech.UpdateTranscriptionJobResponse, error)
	deleteTranscriptionJobFn func(context.Context, aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error)
}

func (f *fakeTranscriptionJobOCIClient) CreateTranscriptionJob(ctx context.Context, req aispeech.CreateTranscriptionJobRequest) (aispeech.CreateTranscriptionJobResponse, error) {
	if f.createTranscriptionJobFn != nil {
		return f.createTranscriptionJobFn(ctx, req)
	}
	return aispeech.CreateTranscriptionJobResponse{}, nil
}

func (f *fakeTranscriptionJobOCIClient) GetTranscriptionJob(ctx context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
	if f.getTranscriptionJobFn != nil {
		return f.getTranscriptionJobFn(ctx, req)
	}
	return aispeech.GetTranscriptionJobResponse{}, nil
}

func (f *fakeTranscriptionJobOCIClient) ListTranscriptionJobs(ctx context.Context, req aispeech.ListTranscriptionJobsRequest) (aispeech.ListTranscriptionJobsResponse, error) {
	if f.listTranscriptionJobsFn != nil {
		return f.listTranscriptionJobsFn(ctx, req)
	}
	return aispeech.ListTranscriptionJobsResponse{}, nil
}

func (f *fakeTranscriptionJobOCIClient) UpdateTranscriptionJob(ctx context.Context, req aispeech.UpdateTranscriptionJobRequest) (aispeech.UpdateTranscriptionJobResponse, error) {
	if f.updateTranscriptionJobFn != nil {
		return f.updateTranscriptionJobFn(ctx, req)
	}
	return aispeech.UpdateTranscriptionJobResponse{}, nil
}

func (f *fakeTranscriptionJobOCIClient) DeleteTranscriptionJob(ctx context.Context, req aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error) {
	if f.deleteTranscriptionJobFn != nil {
		return f.deleteTranscriptionJobFn(ctx, req)
	}
	return aispeech.DeleteTranscriptionJobResponse{}, nil
}

func testTranscriptionJobClient(fake *fakeTranscriptionJobOCIClient) TranscriptionJobServiceClient {
	return newTranscriptionJobServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeTranscriptionJobResource() *aispeechv1beta1.TranscriptionJob {
	return &aispeechv1beta1.TranscriptionJob{
		Spec: aispeechv1beta1.TranscriptionJobSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "job-alpha",
			Description:   "desired description",
			InputLocation: aispeechv1beta1.TranscriptionJobInputLocation{
				ObjectLocation: aispeechv1beta1.TranscriptionJobInputLocationObjectLocation{
					NamespaceName: "namespace",
					BucketName:    "input-bucket",
					ObjectNames:   []string{"audio.wav"},
				},
			},
			OutputLocation: aispeechv1beta1.TranscriptionJobOutputLocation{
				NamespaceName: "namespace",
				BucketName:    "output-bucket",
				Prefix:        "transcripts/",
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKTranscriptionJob(
	id string,
	compartmentID string,
	displayName string,
	description string,
	lifecycleState aispeech.TranscriptionJobLifecycleStateEnum,
	lifecycleDetails string,
) aispeech.TranscriptionJob {
	job := aispeech.TranscriptionJob{
		Id:            common.String(id),
		CompartmentId: common.String(compartmentID),
		InputLocation: aispeech.ObjectListInlineInputLocation{
			ObjectLocations: []aispeech.ObjectLocation{
				{
					NamespaceName: common.String("namespace"),
					BucketName:    common.String("input-bucket"),
					ObjectNames:   []string{"audio.wav"},
				},
			},
		},
		OutputLocation: &aispeech.OutputLocation{
			NamespaceName: common.String("namespace"),
			BucketName:    common.String("output-bucket"),
			Prefix:        common.String("transcripts/"),
		},
		DisplayName:    common.String(displayName),
		LifecycleState: lifecycleState,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	if description != "" {
		job.Description = common.String(description)
	}
	if lifecycleDetails != "" {
		job.LifecycleDetails = common.String(lifecycleDetails)
	}
	return job
}

func withSDKTranscriptionJobAdditionalSettings(job aispeech.TranscriptionJob, additionalSettings map[string]string) aispeech.TranscriptionJob {
	if len(additionalSettings) == 0 {
		return job
	}

	settingsCopy := make(map[string]string, len(additionalSettings))
	for key, value := range additionalSettings {
		settingsCopy[key] = value
	}

	job.ModelDetails = &aispeech.TranscriptionModelDetails{
		TranscriptionSettings: &aispeech.TranscriptionSettings{
			AdditionalSettings: settingsCopy,
		},
	}
	return job
}

func requireAsyncCurrent(
	t *testing.T,
	resource *aispeechv1beta1.TranscriptionJob,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	rawStatus string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want populated tracker")
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
}

func requireTrailingCondition(t *testing.T, resource *aispeechv1beta1.TranscriptionJob, condition shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want trailing condition")
	}
	if got := conditions[len(conditions)-1].Type; got != condition {
		t.Fatalf("last condition = %q, want %q", got, condition)
	}
}

func TestTranscriptionJobCreateCarriesAdditionalSettingsAndProjectsStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.transcriptionjob.oc1..created"

	resource := makeTranscriptionJobResource()
	resource.Spec.ModelDetails = aispeechv1beta1.TranscriptionJobModelDetails{
		TranscriptionSettings: aispeechv1beta1.TranscriptionJobModelDetailsTranscriptionSettings{
			AdditionalSettings: map[string]string{
				"normalization_mode": "smart",
				"vocabulary":         "medical",
			},
		},
	}

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		listTranscriptionJobsFn: func(_ context.Context, req aispeech.ListTranscriptionJobsRequest) (aispeech.ListTranscriptionJobsResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("list compartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("list displayName = %v, want %q", req.DisplayName, resource.Spec.DisplayName)
			}
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil before create", req.Id)
			}
			return aispeech.ListTranscriptionJobsResponse{}, nil
		},
		createTranscriptionJobFn: func(_ context.Context, req aispeech.CreateTranscriptionJobRequest) (aispeech.CreateTranscriptionJobResponse, error) {
			if req.ModelDetails == nil {
				t.Fatal("create request modelDetails = nil, want additionalSettings carried into create body")
			}
			if req.ModelDetails.TranscriptionSettings == nil {
				t.Fatal("create request transcriptionSettings = nil, want additionalSettings carried into create body")
			}
			if !reflect.DeepEqual(
				req.ModelDetails.TranscriptionSettings.AdditionalSettings,
				resource.Spec.ModelDetails.TranscriptionSettings.AdditionalSettings,
			) {
				t.Fatalf(
					"create request additionalSettings = %#v, want %#v",
					req.ModelDetails.TranscriptionSettings.AdditionalSettings,
					resource.Spec.ModelDetails.TranscriptionSettings.AdditionalSettings,
				)
			}
			return aispeech.CreateTranscriptionJobResponse{
				TranscriptionJob: withSDKTranscriptionJobAdditionalSettings(
					makeSDKTranscriptionJob(
						createdID,
						resource.Spec.CompartmentId,
						resource.Spec.DisplayName,
						resource.Spec.Description,
						aispeech.TranscriptionJobLifecycleStateAccepted,
						"",
					),
					resource.Spec.ModelDetails.TranscriptionSettings.AdditionalSettings,
				),
			}, nil
		},
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != createdID {
				t.Fatalf("get transcriptionJobId = %v, want %q", req.TranscriptionJobId, createdID)
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: withSDKTranscriptionJobAdditionalSettings(
					makeSDKTranscriptionJob(
						createdID,
						resource.Spec.CompartmentId,
						resource.Spec.DisplayName,
						resource.Spec.Description,
						aispeech.TranscriptionJobLifecycleStateSucceeded,
						"",
					),
					resource.Spec.ModelDetails.TranscriptionSettings.AdditionalSettings,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after SUCCEEDED follow-up")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after SUCCEEDED follow-up")
	}
	if resource.Status.Id != createdID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.status.ocid = %q, want %q", got, createdID)
	}
	if !reflect.DeepEqual(
		resource.Status.ModelDetails.TranscriptionSettings.AdditionalSettings,
		resource.Spec.ModelDetails.TranscriptionSettings.AdditionalSettings,
	) {
		t.Fatalf(
			"status additionalSettings = %#v, want %#v",
			resource.Status.ModelDetails.TranscriptionSettings.AdditionalSettings,
			resource.Spec.ModelDetails.TranscriptionSettings.AdditionalSettings,
		)
	}
}

func TestTranscriptionJobCreateOrUpdateTreatsFailedAsFailed(t *testing.T) {
	t.Parallel()

	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..example",
					"job-alpha",
					"desired description",
					aispeech.TranscriptionJobLifecycleStateFailed,
					"job processing failed",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when lifecycle is FAILED")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when lifecycle is FAILED")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Failed)
	}
	if resource.Status.OsokStatus.Message != "job processing failed" {
		t.Fatalf("status message = %q, want lifecycle details", resource.Status.OsokStatus.Message)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassFailed, "FAILED")
	requireTrailingCondition(t, resource, shared.Failed)
}

func TestTranscriptionJobCreateOrUpdateTreatsCancelingAsTerminating(t *testing.T) {
	t.Parallel()

	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..example",
					"job-alpha",
					"desired description",
					aispeech.TranscriptionJobLifecycleStateCanceling,
					"delete is settling",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should stay successful while lifecycle is CANCELING")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while lifecycle is CANCELING")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "CANCELING")
	requireTrailingCondition(t, resource, shared.Terminating)
}

func TestTranscriptionJobCreateOrUpdateTreatsCanceledAsCanceledFailure(t *testing.T) {
	t.Parallel()

	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..example",
					"job-alpha",
					"desired description",
					aispeech.TranscriptionJobLifecycleStateCanceled,
					"job was canceled",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when lifecycle is CANCELED")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when lifecycle is CANCELED")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Failed)
	}
	if resource.Status.OsokStatus.Message != "job was canceled" {
		t.Fatalf("status message = %q, want lifecycle details", resource.Status.OsokStatus.Message)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassCanceled, "CANCELED")
	requireTrailingCondition(t, resource, shared.Failed)
}

func TestTranscriptionJobCreateOrUpdateRejectsCompartmentMove(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..observed",
					"job-alpha",
					"desired description",
					aispeech.TranscriptionJobLifecycleStateSucceeded,
					"",
				),
			}, nil
		},
		updateTranscriptionJobFn: func(_ context.Context, _ aispeech.UpdateTranscriptionJobRequest) (aispeech.UpdateTranscriptionJobResponse, error) {
			updateCalls++
			return aispeech.UpdateTranscriptionJobResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want replacement validation error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when compartment drift requires replacement")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateTranscriptionJob() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "TranscriptionJob formal semantics require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want replacement message", err)
	}
}

func TestTranscriptionJobCreateOrUpdateRejectsAdditionalSettingsDrift(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"
	resource.Spec.ModelDetails = aispeechv1beta1.TranscriptionJobModelDetails{
		TranscriptionSettings: aispeechv1beta1.TranscriptionJobModelDetailsTranscriptionSettings{
			AdditionalSettings: map[string]string{
				"normalization_mode": "smart",
			},
		},
	}

	liveAdditionalSettings := map[string]string{
		"normalization_mode": "basic",
	}

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: withSDKTranscriptionJobAdditionalSettings(
					makeSDKTranscriptionJob(
						"ocid1.transcriptionjob.oc1..existing",
						"ocid1.compartment.oc1..example",
						"job-alpha",
						"desired description",
						aispeech.TranscriptionJobLifecycleStateSucceeded,
						"",
					),
					liveAdditionalSettings,
				),
			}, nil
		},
		updateTranscriptionJobFn: func(_ context.Context, _ aispeech.UpdateTranscriptionJobRequest) (aispeech.UpdateTranscriptionJobResponse, error) {
			updateCalls++
			return aispeech.UpdateTranscriptionJobResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported additionalSettings drift")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure when additionalSettings drift is create-only")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateTranscriptionJob() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "modelDetails.transcriptionSettings.additionalSettings") {
		t.Fatalf("CreateOrUpdate() error = %q, want additionalSettings drift path", err)
	}
	if !reflect.DeepEqual(resource.Status.ModelDetails.TranscriptionSettings.AdditionalSettings, liveAdditionalSettings) {
		t.Fatalf(
			"status additionalSettings = %#v, want live OCI status %#v",
			resource.Status.ModelDetails.TranscriptionSettings.AdditionalSettings,
			liveAdditionalSettings,
		)
	}
}

func TestTranscriptionJobDeleteWaitsForNotFoundAfterCanceled(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			getCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			switch getCalls {
			case 1:
				return aispeech.GetTranscriptionJobResponse{
					TranscriptionJob: makeSDKTranscriptionJob(
						"ocid1.transcriptionjob.oc1..existing",
						"ocid1.compartment.oc1..example",
						"job-alpha",
						"desired description",
						aispeech.TranscriptionJobLifecycleStateSucceeded,
						"",
					),
				}, nil
			case 2:
				return aispeech.GetTranscriptionJobResponse{
					TranscriptionJob: makeSDKTranscriptionJob(
						"ocid1.transcriptionjob.oc1..existing",
						"ocid1.compartment.oc1..example",
						"job-alpha",
						"desired description",
						aispeech.TranscriptionJobLifecycleStateCanceled,
						"cleanup is settling",
					),
				}, nil
			default:
				return aispeech.GetTranscriptionJobResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "gone")
			}
		},
		deleteTranscriptionJobFn: func(_ context.Context, req aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error) {
			deleteCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("delete transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.DeleteTranscriptionJobResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while GET still returns CANCELED")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTranscriptionJob() calls = %d, want 1", deleteCalls)
	}
	if resource.Status.LifecycleState != "CANCELED" {
		t.Fatalf("status.lifecycleState = %q, want CANCELED", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "CANCELED")
	requireTrailingCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call should succeed after GET reports not found")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTranscriptionJob() calls = %d after second call, want 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want timestamp after delete confirmation")
	}
}

func TestTranscriptionJobDeleteCallsDeleteForLiveCanceledJobWithoutPendingState(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			getCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..example",
					"job-alpha",
					"desired description",
					aispeech.TranscriptionJobLifecycleStateCanceled,
					"job already canceled before delete",
				),
			}, nil
		},
		deleteTranscriptionJobFn: func(_ context.Context, req aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error) {
			deleteCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("delete transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.DeleteTranscriptionJobResponse{OpcRequestId: common.String("opc-delete-canceled")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should still wait for final not-found confirmation after deleting a canceled job")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTranscriptionJob() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetTranscriptionJob() calls = %d, want 2 (pre-delete read plus confirm read)", getCalls)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "CANCELED")
	requireTrailingCondition(t, resource, shared.Terminating)
}

func TestTranscriptionJobDeleteCallsDeleteAfterCancelingSettlesToCanceled(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			getCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}

			lifecycleState := aispeech.TranscriptionJobLifecycleStateCanceling
			lifecycleDetails := "cancel still settling"
			if getCalls >= 2 {
				lifecycleState = aispeech.TranscriptionJobLifecycleStateCanceled
				lifecycleDetails = "cancel completed"
			}

			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..example",
					"job-alpha",
					"desired description",
					lifecycleState,
					lifecycleDetails,
				),
			}, nil
		},
		deleteTranscriptionJobFn: func(_ context.Context, req aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error) {
			deleteCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("delete transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.DeleteTranscriptionJobResponse{OpcRequestId: common.String("opc-delete-after-cancel")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call should wait while the job is still CANCELING")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteTranscriptionJob() calls after first delete = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want no delete-confirmation marker before delete is attempted", resource.Status.OsokStatus.Async.Current)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "waiting to issue delete") {
		t.Fatalf("status message = %q, want waiting-to-issue-delete breadcrumb", resource.Status.OsokStatus.Message)
	}
	requireTrailingCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() second call should still wait for final not-found confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTranscriptionJob() calls after second delete = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetTranscriptionJob() calls after second delete = %d, want 3", getCalls)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "CANCELED")
	requireTrailingCondition(t, resource, shared.Terminating)
}

func TestTranscriptionJobDeleteConflictRereadsLifecycleState(t *testing.T) {
	t.Parallel()

	getCalls := 0
	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			getCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			if getCalls == 1 {
				return aispeech.GetTranscriptionJobResponse{
					TranscriptionJob: makeSDKTranscriptionJob(
						"ocid1.transcriptionjob.oc1..existing",
						"ocid1.compartment.oc1..example",
						"job-alpha",
						"desired description",
						aispeech.TranscriptionJobLifecycleStateSucceeded,
						"",
					),
				}, nil
			}
			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..example",
					"job-alpha",
					"desired description",
					aispeech.TranscriptionJobLifecycleStateCanceling,
					"delete conflict settled into canceling",
				),
			}, nil
		},
		deleteTranscriptionJobFn: func(_ context.Context, req aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error) {
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("delete transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			return aispeech.DeleteTranscriptionJobResponse{}, errortest.NewServiceError(409, "IncorrectState", "delete is still settling")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting after conflict follow-up returns CANCELING")
	}
	if getCalls != 2 {
		t.Fatalf("GetTranscriptionJob() calls = %d, want pre-delete read plus one follow-up read", getCalls)
	}
	if resource.Status.LifecycleState != "CANCELING" {
		t.Fatalf("status.lifecycleState = %q, want CANCELING", resource.Status.LifecycleState)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "CANCELING")
	requireTrailingCondition(t, resource, shared.Terminating)
}

func TestTranscriptionJobDeleteConflictWithSucceededFollowUpRetriesDelete(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	resource := makeTranscriptionJobResource()
	resource.Status.Id = "ocid1.transcriptionjob.oc1..existing"

	client := testTranscriptionJobClient(&fakeTranscriptionJobOCIClient{
		getTranscriptionJobFn: func(_ context.Context, req aispeech.GetTranscriptionJobRequest) (aispeech.GetTranscriptionJobResponse, error) {
			getCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("get transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}

			lifecycleState := aispeech.TranscriptionJobLifecycleStateSucceeded
			lifecycleDetails := ""
			if getCalls == 4 {
				lifecycleState = aispeech.TranscriptionJobLifecycleStateCanceled
				lifecycleDetails = "delete request accepted"
			}

			return aispeech.GetTranscriptionJobResponse{
				TranscriptionJob: makeSDKTranscriptionJob(
					"ocid1.transcriptionjob.oc1..existing",
					"ocid1.compartment.oc1..example",
					"job-alpha",
					"desired description",
					lifecycleState,
					lifecycleDetails,
				),
			}, nil
		},
		deleteTranscriptionJobFn: func(_ context.Context, req aispeech.DeleteTranscriptionJobRequest) (aispeech.DeleteTranscriptionJobResponse, error) {
			deleteCalls++
			if req.TranscriptionJobId == nil || *req.TranscriptionJobId != "ocid1.transcriptionjob.oc1..existing" {
				t.Fatalf("delete transcriptionJobId = %v, want tracked TranscriptionJob ID", req.TranscriptionJobId)
			}
			if deleteCalls == 1 {
				return aispeech.DeleteTranscriptionJobResponse{}, errortest.NewServiceError(409, "IncorrectState", "delete is still settling")
			}
			return aispeech.DeleteTranscriptionJobResponse{OpcRequestId: common.String("opc-delete-2")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call should retry when conflict follow-up still returns SUCCEEDED")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTranscriptionJob() calls after first delete = %d, want 1", deleteCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetTranscriptionJob() calls after first delete = %d, want 2", getCalls)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared retry state after SUCCEEDED follow-up", resource.Status.OsokStatus.Async.Current)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "retrying delete") {
		t.Fatalf("status message = %q, want retrying delete breadcrumb", resource.Status.OsokStatus.Message)
	}
	requireTrailingCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() second call should wait while follow-up reports CANCELED")
	}
	if deleteCalls != 2 {
		t.Fatalf("DeleteTranscriptionJob() calls after second delete = %d, want 2", deleteCalls)
	}
	if getCalls != 4 {
		t.Fatalf("GetTranscriptionJob() calls after second delete = %d, want 4", getCalls)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "CANCELED")
	requireTrailingCondition(t, resource, shared.Terminating)
}
