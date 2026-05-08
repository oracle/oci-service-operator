/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package enrichmentjob

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaidatasdk "github.com/oracle/oci-go-sdk/v65/generativeaidata"
	generativeaidatav1beta1 "github.com/oracle/oci-service-operator/api/generativeaidata/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeEnrichmentJobOCIClient struct {
	generateEnrichmentJobFn func(context.Context, generativeaidatasdk.GenerateEnrichmentJobRequest) (generativeaidatasdk.GenerateEnrichmentJobResponse, error)
	getEnrichmentJobFn      func(context.Context, generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error)
	listEnrichmentJobsFn    func(context.Context, generativeaidatasdk.ListEnrichmentJobsRequest) (generativeaidatasdk.ListEnrichmentJobsResponse, error)
	cancelEnrichmentJobFn   func(context.Context, generativeaidatasdk.CancelEnrichmentJobRequest) (generativeaidatasdk.CancelEnrichmentJobResponse, error)
}

func (f *fakeEnrichmentJobOCIClient) GenerateEnrichmentJob(
	ctx context.Context,
	req generativeaidatasdk.GenerateEnrichmentJobRequest,
) (generativeaidatasdk.GenerateEnrichmentJobResponse, error) {
	if f.generateEnrichmentJobFn != nil {
		return f.generateEnrichmentJobFn(ctx, req)
	}
	return generativeaidatasdk.GenerateEnrichmentJobResponse{}, nil
}

func (f *fakeEnrichmentJobOCIClient) GetEnrichmentJob(
	ctx context.Context,
	req generativeaidatasdk.GetEnrichmentJobRequest,
) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
	if f.getEnrichmentJobFn != nil {
		return f.getEnrichmentJobFn(ctx, req)
	}
	return generativeaidatasdk.GetEnrichmentJobResponse{}, nil
}

func (f *fakeEnrichmentJobOCIClient) ListEnrichmentJobs(
	ctx context.Context,
	req generativeaidatasdk.ListEnrichmentJobsRequest,
) (generativeaidatasdk.ListEnrichmentJobsResponse, error) {
	if f.listEnrichmentJobsFn != nil {
		return f.listEnrichmentJobsFn(ctx, req)
	}
	return generativeaidatasdk.ListEnrichmentJobsResponse{}, nil
}

func (f *fakeEnrichmentJobOCIClient) CancelEnrichmentJob(
	ctx context.Context,
	req generativeaidatasdk.CancelEnrichmentJobRequest,
) (generativeaidatasdk.CancelEnrichmentJobResponse, error) {
	if f.cancelEnrichmentJobFn != nil {
		return f.cancelEnrichmentJobFn(ctx, req)
	}
	return generativeaidatasdk.CancelEnrichmentJobResponse{}, nil
}

func testEnrichmentJobClient(fake *fakeEnrichmentJobOCIClient) EnrichmentJobServiceClient {
	return newEnrichmentJobServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeEnrichmentJobResource() *generativeaidatav1beta1.EnrichmentJob {
	return &generativeaidatav1beta1.EnrichmentJob{
		Spec: generativeaidatav1beta1.EnrichmentJobSpec{
			SemanticStoreId:   "ocid1.semanticstore.oc1..example",
			CompartmentId:     "ocid1.compartment.oc1..example",
			EnrichmentJobType: string(generativeaidatasdk.EnrichmentJobTypeFullBuild),
			EnrichmentJobConfiguration: generativeaidatav1beta1.EnrichmentJobConfiguration{
				EnrichmentJobType: string(generativeaidatasdk.EnrichmentJobTypeFullBuild),
				SchemaName:        "SALES",
			},
			DisplayName: "job-alpha",
			Description: "desired description",
			FreeformTags: map[string]string{
				"env": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKEnrichmentJob(
	id string,
	semanticStoreID string,
	displayName string,
	description string,
	lifecycleState generativeaidatasdk.LifecycleStateEnum,
	lifecycleDetails string,
) generativeaidatasdk.EnrichmentJob {
	timeAccepted := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	job := generativeaidatasdk.EnrichmentJob{
		Id:                common.String(id),
		SemanticStoreId:   common.String(semanticStoreID),
		EnrichmentJobType: generativeaidatasdk.EnrichmentJobTypeFullBuild,
		EnrichmentJobConfiguration: generativeaidatasdk.FullBuildEnrichmentJobConfiguration{
			SchemaName: common.String("SALES"),
		},
		TimeAccepted:     &timeAccepted,
		LifecycleDetails: common.String(lifecycleDetails),
		LifecycleState:   lifecycleState,
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	if displayName != "" {
		job.DisplayName = common.String(displayName)
	}
	if description != "" {
		job.Description = common.String(description)
	}
	return job
}

func TestEnrichmentJobCreateSkipsLookupWithoutDisplayNameAndProjectsLookupStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.enrichmentjob.oc1..created"

	resource := makeEnrichmentJobResource()
	resource.Spec.DisplayName = ""

	client := testEnrichmentJobClient(&fakeEnrichmentJobOCIClient{
		listEnrichmentJobsFn: func(_ context.Context, _ generativeaidatasdk.ListEnrichmentJobsRequest) (generativeaidatasdk.ListEnrichmentJobsResponse, error) {
			t.Fatal("ListEnrichmentJobs() should not run when spec.displayName is empty")
			return generativeaidatasdk.ListEnrichmentJobsResponse{}, nil
		},
		generateEnrichmentJobFn: func(_ context.Context, req generativeaidatasdk.GenerateEnrichmentJobRequest) (generativeaidatasdk.GenerateEnrichmentJobResponse, error) {
			if req.SemanticStoreId == nil || *req.SemanticStoreId != resource.Spec.SemanticStoreId {
				t.Fatalf("generate semanticStoreId = %v, want %q", req.SemanticStoreId, resource.Spec.SemanticStoreId)
			}
			if req.EnrichmentJobType != generativeaidatasdk.EnrichmentJobTypeFullBuild {
				t.Fatalf("generate enrichmentJobType = %q, want FULL_BUILD", req.EnrichmentJobType)
			}
			cfg, ok := req.EnrichmentJobConfiguration.(generativeaidatasdk.FullBuildEnrichmentJobConfiguration)
			if !ok {
				t.Fatalf("generate enrichmentJobConfiguration = %T, want FullBuildEnrichmentJobConfiguration", req.EnrichmentJobConfiguration)
			}
			if cfg.SchemaName == nil || *cfg.SchemaName != resource.Spec.EnrichmentJobConfiguration.SchemaName {
				t.Fatalf("generate schemaName = %v, want %q", cfg.SchemaName, resource.Spec.EnrichmentJobConfiguration.SchemaName)
			}
			return generativeaidatasdk.GenerateEnrichmentJobResponse{
				EnrichmentJob: makeSDKEnrichmentJob(
					createdID,
					resource.Spec.SemanticStoreId,
					"",
					resource.Spec.Description,
					generativeaidatasdk.LifecycleStateAccepted,
					"accepted",
				),
			}, nil
		},
		getEnrichmentJobFn: func(_ context.Context, req generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
			if req.SemanticStoreId == nil || *req.SemanticStoreId != resource.Spec.SemanticStoreId {
				t.Fatalf("get semanticStoreId = %v, want %q", req.SemanticStoreId, resource.Spec.SemanticStoreId)
			}
			if req.EnrichmentJobId == nil || *req.EnrichmentJobId != createdID {
				t.Fatalf("get enrichmentJobId = %v, want %q", req.EnrichmentJobId, createdID)
			}
			return generativeaidatasdk.GetEnrichmentJobResponse{
				EnrichmentJob: makeSDKEnrichmentJob(
					createdID,
					resource.Spec.SemanticStoreId,
					"",
					resource.Spec.Description,
					generativeaidatasdk.LifecycleStateSucceeded,
					"",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful response", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue after SUCCEEDED follow-up", response)
	}
	if resource.Status.Id != createdID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, createdID)
	}
	if resource.Status.SemanticStoreId != resource.Spec.SemanticStoreId {
		t.Fatalf("status.semanticStoreId = %q, want %q", resource.Status.SemanticStoreId, resource.Spec.SemanticStoreId)
	}
	if resource.Status.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", resource.Status.CompartmentId, resource.Spec.CompartmentId)
	}
}

func TestEnrichmentJobCreateWithDisplayNameRejectsDuplicateLookupMatches(t *testing.T) {
	t.Parallel()

	resource := makeEnrichmentJobResource()

	client := testEnrichmentJobClient(&fakeEnrichmentJobOCIClient{
		listEnrichmentJobsFn: func(_ context.Context, req generativeaidatasdk.ListEnrichmentJobsRequest) (generativeaidatasdk.ListEnrichmentJobsResponse, error) {
			if req.SemanticStoreId == nil || *req.SemanticStoreId != resource.Spec.SemanticStoreId {
				t.Fatalf("list semanticStoreId = %v, want %q", req.SemanticStoreId, resource.Spec.SemanticStoreId)
			}
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("list compartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("list displayName = %v, want %q", req.DisplayName, resource.Spec.DisplayName)
			}
			return generativeaidatasdk.ListEnrichmentJobsResponse{
				EnrichmentJobCollection: generativeaidatasdk.EnrichmentJobCollection{
					Items: []generativeaidatasdk.EnrichmentJobSummary{
						{
							Id:              common.String("ocid1.enrichmentjob.oc1..one"),
							SemanticStoreId: common.String(resource.Spec.SemanticStoreId),
							DisplayName:     common.String(resource.Spec.DisplayName),
						},
						{
							Id:              common.String("ocid1.enrichmentjob.oc1..two"),
							SemanticStoreId: common.String(resource.Spec.SemanticStoreId),
							DisplayName:     common.String(resource.Spec.DisplayName),
						},
					},
				},
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "multiple exact displayName matches") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate displayName match failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful duplicate-match response", response)
	}
}

func TestEnrichmentJobCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.enrichmentjob.oc1..existing"

	resource := makeEnrichmentJobResource()
	resource.Status.Id = existingID
	resource.Status.SemanticStoreId = resource.Spec.SemanticStoreId
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Spec.Description = "new description"

	client := testEnrichmentJobClient(&fakeEnrichmentJobOCIClient{
		getEnrichmentJobFn: func(_ context.Context, req generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
			if req.EnrichmentJobId == nil || *req.EnrichmentJobId != existingID {
				t.Fatalf("get enrichmentJobId = %v, want %q", req.EnrichmentJobId, existingID)
			}
			return generativeaidatasdk.GetEnrichmentJobResponse{
				EnrichmentJob: makeSDKEnrichmentJob(
					existingID,
					resource.Spec.SemanticStoreId,
					resource.Spec.DisplayName,
					"old description",
					generativeaidatasdk.LifecycleStateSucceeded,
					"",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create-only drift for description") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift response", response)
	}
}

func TestEnrichmentJobCreateOrUpdateRejectsCompartmentMoveAcrossConsecutiveReconciles(t *testing.T) {
	t.Parallel()

	const (
		existingID          = "ocid1.enrichmentjob.oc1..existing"
		originalCompartment = "ocid1.compartment.oc1..original"
		newCompartment      = "ocid1.compartment.oc1..moved"
	)

	resource := makeEnrichmentJobResource()
	resource.Spec.CompartmentId = newCompartment
	resource.Status.Id = existingID
	resource.Status.SemanticStoreId = resource.Spec.SemanticStoreId
	resource.Status.CompartmentId = originalCompartment
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	getCalls := 0
	client := testEnrichmentJobClient(&fakeEnrichmentJobOCIClient{
		getEnrichmentJobFn: func(_ context.Context, req generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
			getCalls++
			if req.EnrichmentJobId == nil || *req.EnrichmentJobId != existingID {
				t.Fatalf("get enrichmentJobId = %v, want %q", req.EnrichmentJobId, existingID)
			}
			return generativeaidatasdk.GetEnrichmentJobResponse{
				EnrichmentJob: makeSDKEnrichmentJob(
					existingID,
					resource.Spec.SemanticStoreId,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					generativeaidatasdk.LifecycleStateSucceeded,
					"",
				),
			}, nil
		},
	})

	for attempt := 1; attempt <= 2; attempt++ {
		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if err == nil || !strings.Contains(err.Error(), "compartmentId") {
			t.Fatalf("CreateOrUpdate() attempt %d error = %v, want compartmentId drift failure", attempt, err)
		}
		if response.IsSuccessful {
			t.Fatalf("CreateOrUpdate() attempt %d response = %#v, want unsuccessful drift response", attempt, response)
		}
		if resource.Status.CompartmentId != originalCompartment {
			t.Fatalf("status.compartmentId after attempt %d = %q, want preserved bound compartment %q", attempt, resource.Status.CompartmentId, originalCompartment)
		}
	}
	if getCalls != 2 {
		t.Fatalf("GetEnrichmentJob() calls = %d, want 2", getCalls)
	}
}

func TestEnrichmentJobDeleteCancelsRunningJobAndWaitsForTerminalState(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.enrichmentjob.oc1..running"

	resource := makeEnrichmentJobResource()
	resource.Status.Id = existingID
	resource.Status.SemanticStoreId = resource.Spec.SemanticStoreId
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.LifecycleState = string(generativeaidatasdk.LifecycleStateInProgress)
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	var cancelCalls int
	var getCalls int

	client := testEnrichmentJobClient(&fakeEnrichmentJobOCIClient{
		getEnrichmentJobFn: func(_ context.Context, req generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
			getCalls++
			if req.SemanticStoreId == nil || *req.SemanticStoreId != resource.Spec.SemanticStoreId {
				t.Fatalf("get semanticStoreId = %v, want %q", req.SemanticStoreId, resource.Spec.SemanticStoreId)
			}
			if req.EnrichmentJobId == nil || *req.EnrichmentJobId != existingID {
				t.Fatalf("get enrichmentJobId = %v, want %q", req.EnrichmentJobId, existingID)
			}
			switch getCalls {
			case 1:
				return generativeaidatasdk.GetEnrichmentJobResponse{
					EnrichmentJob: makeSDKEnrichmentJob(
						existingID,
						resource.Spec.SemanticStoreId,
						resource.Spec.DisplayName,
						resource.Spec.Description,
						generativeaidatasdk.LifecycleStateInProgress,
						"running",
					),
				}, nil
			case 2:
				return generativeaidatasdk.GetEnrichmentJobResponse{
					EnrichmentJob: makeSDKEnrichmentJob(
						existingID,
						resource.Spec.SemanticStoreId,
						resource.Spec.DisplayName,
						resource.Spec.Description,
						generativeaidatasdk.LifecycleStateAccepted,
						"cancel requested",
					),
				}, nil
			default:
				return generativeaidatasdk.GetEnrichmentJobResponse{
					EnrichmentJob: makeSDKEnrichmentJob(
						existingID,
						resource.Spec.SemanticStoreId,
						resource.Spec.DisplayName,
						resource.Spec.Description,
						generativeaidatasdk.LifecycleStateCanceled,
						"canceled",
					),
				}, nil
			}
		},
		cancelEnrichmentJobFn: func(_ context.Context, req generativeaidatasdk.CancelEnrichmentJobRequest) (generativeaidatasdk.CancelEnrichmentJobResponse, error) {
			cancelCalls++
			if req.SemanticStoreId == nil || *req.SemanticStoreId != resource.Spec.SemanticStoreId {
				t.Fatalf("cancel semanticStoreId = %v, want %q", req.SemanticStoreId, resource.Spec.SemanticStoreId)
			}
			if req.EnrichmentJobId == nil || *req.EnrichmentJobId != existingID {
				t.Fatalf("cancel enrichmentJobId = %v, want %q", req.EnrichmentJobId, existingID)
			}
			return generativeaidatasdk.CancelEnrichmentJobResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while cancellation is pending")
	}
	if cancelCalls != 1 {
		t.Fatalf("CancelEnrichmentJob() calls = %d, want 1", cancelCalls)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want delete tracker")
	}
	if current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.phase = %q, want delete", current.Phase)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want finalizer release after CANCELED confirmation")
	}
	if cancelCalls != 1 {
		t.Fatalf("CancelEnrichmentJob() calls after terminal confirmation = %d, want 1", cancelCalls)
	}
}

func TestEnrichmentJobDeleteSkipsCancelWhenJobAlreadySucceeded(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.enrichmentjob.oc1..done"

	resource := makeEnrichmentJobResource()
	resource.Status.Id = existingID
	resource.Status.SemanticStoreId = resource.Spec.SemanticStoreId
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.LifecycleState = string(generativeaidatasdk.LifecycleStateSucceeded)
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	client := testEnrichmentJobClient(&fakeEnrichmentJobOCIClient{
		getEnrichmentJobFn: func(_ context.Context, req generativeaidatasdk.GetEnrichmentJobRequest) (generativeaidatasdk.GetEnrichmentJobResponse, error) {
			if req.EnrichmentJobId == nil || *req.EnrichmentJobId != existingID {
				t.Fatalf("get enrichmentJobId = %v, want %q", req.EnrichmentJobId, existingID)
			}
			return generativeaidatasdk.GetEnrichmentJobResponse{
				EnrichmentJob: makeSDKEnrichmentJob(
					existingID,
					resource.Spec.SemanticStoreId,
					resource.Spec.DisplayName,
					resource.Spec.Description,
					generativeaidatasdk.LifecycleStateSucceeded,
					"",
				),
			}, nil
		},
		cancelEnrichmentJobFn: func(_ context.Context, _ generativeaidatasdk.CancelEnrichmentJobRequest) (generativeaidatasdk.CancelEnrichmentJobResponse, error) {
			t.Fatal("CancelEnrichmentJob() should not run for a terminal SUCCEEDED job")
			return generativeaidatasdk.CancelEnrichmentJobResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want terminal SUCCEEDED job to release the finalizer")
	}
}
