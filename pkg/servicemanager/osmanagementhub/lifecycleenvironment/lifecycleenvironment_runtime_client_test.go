/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lifecycleenvironment

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLifecycleEnvironmentID = "ocid1.lifecycleenvironment.oc1..example"
	testCompartmentID          = "ocid1.compartment.oc1..example"
)

type fakeLifecycleEnvironmentOCIClient struct {
	createFn func(context.Context, osmanagementhubsdk.CreateLifecycleEnvironmentRequest) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error)
	getFn    func(context.Context, osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error)
	listFn   func(context.Context, osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error)
	updateFn func(context.Context, osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error)
	deleteFn func(context.Context, osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error)

	createRequests []osmanagementhubsdk.CreateLifecycleEnvironmentRequest
	getRequests    []osmanagementhubsdk.GetLifecycleEnvironmentRequest
	listRequests   []osmanagementhubsdk.ListLifecycleEnvironmentsRequest
	updateRequests []osmanagementhubsdk.UpdateLifecycleEnvironmentRequest
	deleteRequests []osmanagementhubsdk.DeleteLifecycleEnvironmentRequest
}

func (f *fakeLifecycleEnvironmentOCIClient) CreateLifecycleEnvironment(
	ctx context.Context,
	request osmanagementhubsdk.CreateLifecycleEnvironmentRequest,
) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return osmanagementhubsdk.CreateLifecycleEnvironmentResponse{}, nil
}

func (f *fakeLifecycleEnvironmentOCIClient) GetLifecycleEnvironment(
	ctx context.Context,
	request osmanagementhubsdk.GetLifecycleEnvironmentRequest,
) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return osmanagementhubsdk.GetLifecycleEnvironmentResponse{}, nil
}

func (f *fakeLifecycleEnvironmentOCIClient) ListLifecycleEnvironments(
	ctx context.Context,
	request osmanagementhubsdk.ListLifecycleEnvironmentsRequest,
) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{}, nil
}

func (f *fakeLifecycleEnvironmentOCIClient) UpdateLifecycleEnvironment(
	ctx context.Context,
	request osmanagementhubsdk.UpdateLifecycleEnvironmentRequest,
) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return osmanagementhubsdk.UpdateLifecycleEnvironmentResponse{}, nil
}

func (f *fakeLifecycleEnvironmentOCIClient) DeleteLifecycleEnvironment(
	ctx context.Context,
	request osmanagementhubsdk.DeleteLifecycleEnvironmentRequest,
) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{}, nil
}

func TestLifecycleEnvironmentCreateOrUpdateCreatesAndRecordsStatus(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.listFn = func(_ context.Context, request osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
		if got, want := stringPointerValue(request.CompartmentId), testCompartmentID; got != want {
			t.Fatalf("ListLifecycleEnvironments().CompartmentId = %q, want %q", got, want)
		}
		if len(request.DisplayName) != 0 {
			t.Fatalf("ListLifecycleEnvironments().DisplayName = %v, want unset because SDK expects []string", request.DisplayName)
		}
		return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{}, nil
	}
	fake.createFn = func(_ context.Context, request osmanagementhubsdk.CreateLifecycleEnvironmentRequest) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error) {
		assertLifecycleEnvironmentCreateRequest(t, request, resource.Spec)
		return osmanagementhubsdk.CreateLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateCreating),
			OpcRequestId:         common.String("opc-create"),
		}, nil
	}
	fake.getFn = func(_ context.Context, request osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		if got, want := stringPointerValue(request.LifecycleEnvironmentId), testLifecycleEnvironmentID; got != want {
			t.Fatalf("GetLifecycleEnvironment().LifecycleEnvironmentId = %q, want %q", got, want)
		}
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
			OpcRequestId:         common.String("opc-get"),
		}, nil
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = false, want true")
	}
	if got, want := len(fake.createRequests), 1; got != want {
		t.Fatalf("CreateLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.Ocid, shared.OCID(testLifecycleEnvironmentID); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.Id, testLifecycleEnvironmentID; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentCreateOrUpdateBindsLaterPageMatch(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	pageToken := "page-2"
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.listFn = func(_ context.Context, request osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
		if request.Page == nil {
			return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{OpcNextPage: common.String(pageToken)}, nil
		}
		if got := stringPointerValue(request.Page); got != pageToken {
			t.Fatalf("ListLifecycleEnvironments().Page = %q, want %q", got, pageToken)
		}
		return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{
			LifecycleEnvironmentCollection: osmanagementhubsdk.LifecycleEnvironmentCollection{
				Items: []osmanagementhubsdk.LifecycleEnvironmentSummary{
					sdkLifecycleEnvironmentSummary(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
				},
			},
		}, nil
	}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
		}, nil
	}
	fake.createFn = func(context.Context, osmanagementhubsdk.CreateLifecycleEnvironmentRequest) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error) {
		t.Fatal("CreateLifecycleEnvironment() called, want bind to later-page list match")
		return osmanagementhubsdk.CreateLifecycleEnvironmentResponse{}, nil
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = false, want true")
	}
	if got, want := len(fake.listRequests), 2; got != want {
		t.Fatalf("ListLifecycleEnvironments() calls = %d, want %d", got, want)
	}
	if got, want := len(fake.createRequests), 0; got != want {
		t.Fatalf("CreateLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.Ocid, shared.OCID(testLifecycleEnvironmentID); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentCreateOrUpdateOmittedLocationDoesNotBindNonDefaultLocations(t *testing.T) {
	for _, location := range []osmanagementhubsdk.ManagedInstanceLocationEnum{
		osmanagementhubsdk.ManagedInstanceLocationOciCompute,
		osmanagementhubsdk.ManagedInstanceLocationAzure,
		osmanagementhubsdk.ManagedInstanceLocationEc2,
		osmanagementhubsdk.ManagedInstanceLocationGcp,
	} {
		t.Run(string(location), func(t *testing.T) {
			resource := newTestLifecycleEnvironment()
			resource.Spec.Location = ""
			liveSpec := resource.Spec
			liveSpec.Location = string(location)
			fake := &fakeLifecycleEnvironmentOCIClient{}
			fake.listFn = func(context.Context, osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
				return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{
					LifecycleEnvironmentCollection: osmanagementhubsdk.LifecycleEnvironmentCollection{
						Items: []osmanagementhubsdk.LifecycleEnvironmentSummary{
							sdkLifecycleEnvironmentSummary(testLifecycleEnvironmentID+".remote", liveSpec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
						},
					},
				}, nil
			}
			fake.createFn = func(_ context.Context, request osmanagementhubsdk.CreateLifecycleEnvironmentRequest) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error) {
				assertLifecycleEnvironmentCreateRequest(t, request, resource.Spec)
				return osmanagementhubsdk.CreateLifecycleEnvironmentResponse{
					LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateCreating),
					OpcRequestId:         common.String("opc-create"),
				}, nil
			}
			fake.getFn = func(context.Context, osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
				return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
					LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
				}, nil
			}

			response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatal("CreateOrUpdate() response.IsSuccessful = false, want true")
			}
			if got, want := len(fake.createRequests), 1; got != want {
				t.Fatalf("CreateLifecycleEnvironment() calls = %d, want %d", got, want)
			}
			if got, want := resource.Status.OsokStatus.Ocid, shared.OCID(testLifecycleEnvironmentID); got != want {
				t.Fatalf("status.status.ocid = %q, want new default-location resource %q", got, want)
			}
		})
	}
}

func TestLifecycleEnvironmentCreateOrUpdateNoopDoesNotUpdate(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
		}, nil
	}
	fake.updateFn = func(context.Context, osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error) {
		t.Fatal("UpdateLifecycleEnvironment() called, want no-op observe")
		return osmanagementhubsdk.UpdateLifecycleEnvironmentResponse{}, nil
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = false, want true")
	}
	if got, want := len(fake.updateRequests), 0; got != want {
		t.Fatalf("UpdateLifecycleEnvironment() calls = %d, want %d", got, want)
	}
}

func TestLifecycleEnvironmentCreateOrUpdateClearsDescription(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Spec.Description = ""
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID

	currentSpec := newTestLifecycleEnvironment().Spec
	cleared := sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive)
	getCalls := 0
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		getCalls++
		if getCalls == 1 {
			return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
				LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, currentSpec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
			}, nil
		}
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{LifecycleEnvironment: cleared}, nil
	}
	fake.updateFn = func(_ context.Context, request osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error) {
		assertLifecycleEnvironmentClearDescriptionRequest(t, request)
		return osmanagementhubsdk.UpdateLifecycleEnvironmentResponse{
			LifecycleEnvironment: cleared,
			OpcRequestId:         common.String("opc-clear-description"),
		}, nil
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = false, want true")
	}
	if got, want := len(fake.updateRequests), 1; got != want {
		t.Fatalf("UpdateLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-clear-description"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Spec.DisplayName = "lifecycle-renamed"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	resource.Spec.Stages[0].DisplayName = "dev-renamed"
	resource.Spec.Stages[0].FreeformTags = map[string]string{"stage": "renamed"}
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID

	currentSpec := newTestLifecycleEnvironment().Spec
	updated := sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive)
	getCalls := 0
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		getCalls++
		if getCalls == 1 {
			return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
				LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, currentSpec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
			}, nil
		}
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{LifecycleEnvironment: updated}, nil
	}
	fake.updateFn = func(_ context.Context, request osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error) {
		assertLifecycleEnvironmentUpdateRequest(t, request, resource.Spec)
		return osmanagementhubsdk.UpdateLifecycleEnvironmentResponse{
			LifecycleEnvironment: updated,
			OpcRequestId:         common.String("opc-update"),
		}, nil
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = false, want true")
	}
	if got, want := len(fake.updateRequests), 1; got != want {
		t.Fatalf("UpdateLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID
	currentSpec := resource.Spec
	currentSpec.ArchType = string(osmanagementhubsdk.ArchTypeAarch64)
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, currentSpec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
		}, nil
	}
	fake.updateFn = func(context.Context, osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error) {
		t.Fatal("UpdateLifecycleEnvironment() called, want create-only drift rejection")
		return osmanagementhubsdk.UpdateLifecycleEnvironmentResponse{}, nil
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err == nil || !strings.Contains(err.Error(), "archType") {
		t.Fatalf("CreateOrUpdate() error = %v, want archType create-only drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = true, want false")
	}
	if got, want := len(fake.updateRequests), 0; got != want {
		t.Fatalf("UpdateLifecycleEnvironment() calls = %d, want %d", got, want)
	}
}

func TestLifecycleEnvironmentCreateOrUpdateRejectsOmittedLocationDrift(t *testing.T) {
	for _, location := range []osmanagementhubsdk.ManagedInstanceLocationEnum{
		osmanagementhubsdk.ManagedInstanceLocationOciCompute,
		osmanagementhubsdk.ManagedInstanceLocationAzure,
		osmanagementhubsdk.ManagedInstanceLocationEc2,
		osmanagementhubsdk.ManagedInstanceLocationGcp,
	} {
		t.Run(string(location), func(t *testing.T) {
			resource := newTestLifecycleEnvironment()
			resource.Spec.Location = ""
			resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
			resource.Status.Id = testLifecycleEnvironmentID
			currentSpec := resource.Spec
			currentSpec.Location = string(location)
			fake := &fakeLifecycleEnvironmentOCIClient{}
			fake.getFn = func(context.Context, osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
				return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
					LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, currentSpec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
				}, nil
			}
			fake.updateFn = func(context.Context, osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error) {
				t.Fatal("UpdateLifecycleEnvironment() called, want omitted location create-only drift rejection")
				return osmanagementhubsdk.UpdateLifecycleEnvironmentResponse{}, nil
			}

			response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
			if err == nil || !strings.Contains(err.Error(), "location") {
				t.Fatalf("CreateOrUpdate() error = %v, want location create-only drift", err)
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() response.IsSuccessful = true, want false")
			}
			if got, want := len(fake.updateRequests), 0; got != want {
				t.Fatalf("UpdateLifecycleEnvironment() calls = %d, want %d", got, want)
			}
		})
	}
}

func TestLifecycleEnvironmentCreateOrUpdateRejectsStageRankDrift(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Spec.Stages[0].Rank = 10
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID
	currentSpec := newTestLifecycleEnvironment().Spec
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, currentSpec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
		}, nil
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err == nil || !strings.Contains(err.Error(), "stages.rank") {
		t.Fatalf("CreateOrUpdate() error = %v, want stages.rank create-only drift", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = true, want false")
	}
}

func TestLifecycleEnvironmentDeleteRetainsFinalizerForDeletingReadback(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID
	getCalls := 0
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		getCalls++
		state := osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive
		if getCalls == 3 {
			state = osmanagementhubsdk.LifecycleEnvironmentLifecycleStateDeleting
		}
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, state),
		}, nil
	}
	fake.deleteFn = func(_ context.Context, _ osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := testLifecycleEnvironmentClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI readback is DELETING")
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("DeleteLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete projection")
	}
	if got, want := resource.Status.LifecycleState, string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateDeleting); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentDeleteDoesNotCallDeleteWhileReadbackIsPendingWrite(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state osmanagementhubsdk.LifecycleEnvironmentLifecycleStateEnum
	}{
		{name: "creating", state: osmanagementhubsdk.LifecycleEnvironmentLifecycleStateCreating},
		{name: "updating", state: osmanagementhubsdk.LifecycleEnvironmentLifecycleStateUpdating},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := newTestLifecycleEnvironment()
			resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
			resource.Status.Id = testLifecycleEnvironmentID
			fake := &fakeLifecycleEnvironmentOCIClient{}
			fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
				return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
					LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, tc.state),
				}, nil
			}
			fake.deleteFn = func(context.Context, osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
				t.Fatal("DeleteLifecycleEnvironment() called, want finalizer retained until pending write state clears")
				return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{}, nil
			}

			deleted, err := testLifecycleEnvironmentClient(fake).Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want false while OCI readback is pending write")
			}
			if got, want := len(fake.deleteRequests), 0; got != want {
				t.Fatalf("DeleteLifecycleEnvironment() calls = %d, want %d", got, want)
			}
			assertLifecycleEnvironmentPendingDeleteStatus(t, resource, string(tc.state))
		})
	}
}

func TestLifecycleEnvironmentDeleteWithOmittedLocationDoesNotDeleteNonDefaultListMatch(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Spec.Location = ""
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.Id = ""
	liveSpec := resource.Spec
	liveSpec.Location = string(osmanagementhubsdk.ManagedInstanceLocationOciCompute)
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(context.Context, osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		t.Fatal("GetLifecycleEnvironment() called, want list fallback without a tracked id")
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{}, nil
	}
	fake.listFn = func(_ context.Context, request osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
		if got, want := stringPointerValue(request.CompartmentId), testCompartmentID; got != want {
			t.Fatalf("ListLifecycleEnvironments().CompartmentId = %q, want %q", got, want)
		}
		return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{
			LifecycleEnvironmentCollection: osmanagementhubsdk.LifecycleEnvironmentCollection{
				Items: []osmanagementhubsdk.LifecycleEnvironmentSummary{
					sdkLifecycleEnvironmentSummary(testLifecycleEnvironmentID+".remote", liveSpec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
				},
			},
		}, nil
	}
	fake.deleteFn = func(context.Context, osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
		t.Fatal("DeleteLifecycleEnvironment() called, want no delete for non-default location list match")
		return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{}, nil
	}

	deleted, err := testLifecycleEnvironmentClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when no matching default-location resource exists")
	}
	if got, want := len(fake.listRequests), 1; got != want {
		t.Fatalf("ListLifecycleEnvironments() calls = %d, want %d", got, want)
	}
	if got, want := len(fake.deleteRequests), 0; got != want {
		t.Fatalf("DeleteLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := resource.Spec.Location, string(osmanagementhubsdk.ManagedInstanceLocationOnPremise); got != want {
		t.Fatalf("spec.location = %q, want defaulted %q", got, want)
	}
}

func TestLifecycleEnvironmentDeleteRejectsAuthShapedListFallbackPreRead(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Status.OsokStatus.Ocid = ""
	resource.Status.Id = ""
	fake := newLifecycleEnvironmentListFallbackAuthDeleteClient(t, resource)

	deleted, err := testLifecycleEnvironmentClient(fake).Delete(context.Background(), resource)
	assertLifecycleEnvironmentListFallbackAuthDeleteBlocked(t, resource, fake, deleted, err)
}

func newLifecycleEnvironmentListFallbackAuthDeleteClient(
	t *testing.T,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
) *fakeLifecycleEnvironmentOCIClient {
	t.Helper()
	return &fakeLifecycleEnvironmentOCIClient{
		listFn:   lifecycleEnvironmentListMatchFunc(t, resource),
		getFn:    lifecycleEnvironmentAuthPreReadFunc(t),
		deleteFn: failLifecycleEnvironmentDeleteFunc(t),
	}
}

func lifecycleEnvironmentListMatchFunc(
	t *testing.T,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
) func(context.Context, osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
	t.Helper()
	return func(_ context.Context, request osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
		if got, want := stringPointerValue(request.CompartmentId), testCompartmentID; got != want {
			t.Fatalf("ListLifecycleEnvironments().CompartmentId = %q, want %q", got, want)
		}
		return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{
			LifecycleEnvironmentCollection: osmanagementhubsdk.LifecycleEnvironmentCollection{
				Items: []osmanagementhubsdk.LifecycleEnvironmentSummary{
					sdkLifecycleEnvironmentSummary(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
				},
			},
		}, nil
	}
}

func lifecycleEnvironmentAuthPreReadFunc(
	t *testing.T,
) func(context.Context, osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
	t.Helper()
	return func(_ context.Context, request osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		if got, want := stringPointerValue(request.LifecycleEnvironmentId), testLifecycleEnvironmentID; got != want {
			t.Fatalf("GetLifecycleEnvironment().LifecycleEnvironmentId = %q, want %q", got, want)
		}
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
}

func failLifecycleEnvironmentDeleteFunc(
	t *testing.T,
) func(context.Context, osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
	t.Helper()
	return func(context.Context, osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
		t.Fatal("DeleteLifecycleEnvironment() called, want auth-shaped pre-read rejection after list fallback")
		return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{}, nil
	}
}

func assertLifecycleEnvironmentListFallbackAuthDeleteBlocked(
	t *testing.T,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	fake *fakeLifecycleEnvironmentOCIClient,
	deleted bool,
	err error,
) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped list-fallback pre-read failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped list-fallback pre-read")
	}
	if got, want := len(fake.listRequests), 1; got != want {
		t.Fatalf("ListLifecycleEnvironments() calls = %d, want %d", got, want)
	}
	if got, want := len(fake.getRequests), 1; got != want {
		t.Fatalf("GetLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := len(fake.deleteRequests), 0; got != want {
		t.Fatalf("DeleteLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.Ocid != "" || resource.Status.Id != "" {
		t.Fatalf("tracked id = %q/%q, want empty after ambiguous list-fallback pre-read", resource.Status.OsokStatus.Ocid, resource.Status.Id)
	}
}

func TestLifecycleEnvironmentDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID
	getCalls := 0
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		getCalls++
		if getCalls == 3 {
			return osmanagementhubsdk.GetLifecycleEnvironmentResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "lifecycle environment deleted")
		}
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
		}, nil
	}
	fake.deleteFn = func(_ context.Context, _ osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := testLifecycleEnvironmentClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete completion timestamp")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentDeleteRejectsAuthShapedPreRead(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
	fake.deleteFn = func(context.Context, osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
		t.Fatal("DeleteLifecycleEnvironment() called, want auth-shaped pre-read rejection")
		return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{}, nil
	}

	deleted, err := testLifecycleEnvironmentClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped pre-read failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-read")
	}
	if got, want := len(fake.deleteRequests), 0; got != want {
		t.Fatalf("DeleteLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLifecycleEnvironmentID)
	resource.Status.Id = testLifecycleEnvironmentID
	getCalls := 0
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.getFn = func(_ context.Context, _ osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
		getCalls++
		if getCalls == 3 {
			return osmanagementhubsdk.GetLifecycleEnvironmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		}
		return osmanagementhubsdk.GetLifecycleEnvironmentResponse{
			LifecycleEnvironment: sdkLifecycleEnvironment(testLifecycleEnvironmentID, resource.Spec, osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive),
		}, nil
	}
	fake.deleteFn = func(_ context.Context, _ osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := testLifecycleEnvironmentClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped post-delete confirmation failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped post-delete confirmation")
	}
	if got, want := len(fake.deleteRequests), 1; got != want {
		t.Fatalf("DeleteLifecycleEnvironment() calls = %d, want %d", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after ambiguous confirmation", resource.Status.OsokStatus.DeletedAt)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestLifecycleEnvironmentCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := newTestLifecycleEnvironment()
	fake := &fakeLifecycleEnvironmentOCIClient{}
	fake.listFn = func(context.Context, osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
		return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{}, nil
	}
	fake.createFn = func(context.Context, osmanagementhubsdk.CreateLifecycleEnvironmentRequest) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error) {
		return osmanagementhubsdk.CreateLifecycleEnvironmentResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	}

	response, err := testLifecycleEnvironmentClient(fake).CreateOrUpdate(context.Background(), resource, testLifecycleEnvironmentRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = true, want false")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "internal server error") {
		t.Fatalf("status.status.message = %q, want OCI error message", resource.Status.OsokStatus.Message)
	}
}

func testLifecycleEnvironmentClient(fake *fakeLifecycleEnvironmentOCIClient) LifecycleEnvironmentServiceClient {
	return newLifecycleEnvironmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func testLifecycleEnvironmentRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "lifecycle-env"}}
}

func newTestLifecycleEnvironment() *osmanagementhubv1beta1.LifecycleEnvironment {
	return &osmanagementhubv1beta1.LifecycleEnvironment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lifecycle-env",
			Namespace: "default",
			UID:       types.UID("uid-lifecycle-env"),
		},
		Spec: osmanagementhubv1beta1.LifecycleEnvironmentSpec{
			CompartmentId: testCompartmentID,
			DisplayName:   "lifecycle-env",
			Stages: []osmanagementhubv1beta1.LifecycleEnvironmentStage{
				{
					DisplayName:  "dev",
					Rank:         1,
					FreeformTags: map[string]string{"stage": "dev"},
					DefinedTags:  map[string]shared.MapValue{"Operations": {"Stage": "dev"}},
				},
				{DisplayName: "prod", Rank: 2},
			},
			ArchType:     string(osmanagementhubsdk.ArchTypeX8664),
			OsFamily:     string(osmanagementhubsdk.OsFamilyOracleLinux8),
			VendorName:   string(osmanagementhubsdk.VendorNameOracle),
			Description:  "desired description",
			Location:     string(osmanagementhubsdk.ManagedInstanceLocationOnPremise),
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func assertLifecycleEnvironmentCreateRequest(
	t *testing.T,
	request osmanagementhubsdk.CreateLifecycleEnvironmentRequest,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
) {
	t.Helper()
	details := request.CreateLifecycleEnvironmentDetails
	if got, want := stringPointerValue(details.CompartmentId), spec.CompartmentId; got != want {
		t.Fatalf("CreateLifecycleEnvironment().CompartmentId = %q, want %q", got, want)
	}
	if got, want := stringPointerValue(details.DisplayName), spec.DisplayName; got != want {
		t.Fatalf("CreateLifecycleEnvironment().DisplayName = %q, want %q", got, want)
	}
	if got, want := string(details.Location), desiredLifecycleEnvironmentLocation(spec); got != want {
		t.Fatalf("CreateLifecycleEnvironment().Location = %q, want %q", got, want)
	}
	if got, want := len(details.Stages), len(spec.Stages); got != want {
		t.Fatalf("CreateLifecycleEnvironment().Stages length = %d, want %d", got, want)
	}
	if got, want := *details.Stages[0].Rank, spec.Stages[0].Rank; got != want {
		t.Fatalf("CreateLifecycleEnvironment().Stages[0].Rank = %d, want %d", got, want)
	}
	if got, want := stringPointerValue(details.Stages[0].DisplayName), spec.Stages[0].DisplayName; got != want {
		t.Fatalf("CreateLifecycleEnvironment().Stages[0].DisplayName = %q, want %q", got, want)
	}
	if got, want := details.Stages[0].FreeformTags["stage"], "dev"; got != want {
		t.Fatalf("CreateLifecycleEnvironment().Stages[0].FreeformTags[stage] = %q, want %q", got, want)
	}
	if got, want := request.OpcRetryToken == nil, false; got != want {
		t.Fatalf("CreateLifecycleEnvironment().OpcRetryToken nil = %t, want %t", got, want)
	}
}

func assertLifecycleEnvironmentUpdateRequest(
	t *testing.T,
	request osmanagementhubsdk.UpdateLifecycleEnvironmentRequest,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
) {
	t.Helper()
	if got, want := stringPointerValue(request.LifecycleEnvironmentId), testLifecycleEnvironmentID; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().LifecycleEnvironmentId = %q, want %q", got, want)
	}
	assertLifecycleEnvironmentUpdateDetails(t, request.UpdateLifecycleEnvironmentDetails, spec)
}

func assertLifecycleEnvironmentClearDescriptionRequest(
	t *testing.T,
	request osmanagementhubsdk.UpdateLifecycleEnvironmentRequest,
) {
	t.Helper()
	if got, want := stringPointerValue(request.LifecycleEnvironmentId), testLifecycleEnvironmentID; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().LifecycleEnvironmentId = %q, want %q", got, want)
	}
	assertLifecycleEnvironmentClearDescriptionDetails(t, request.UpdateLifecycleEnvironmentDetails)
}

func assertLifecycleEnvironmentClearDescriptionDetails(
	t *testing.T,
	details osmanagementhubsdk.UpdateLifecycleEnvironmentDetails,
) {
	t.Helper()
	if details.Description == nil {
		t.Fatal("UpdateLifecycleEnvironment().Description = nil, want empty string pointer")
	}
	if got := *details.Description; got != "" {
		t.Fatalf("UpdateLifecycleEnvironment().Description = %q, want empty string", got)
	}
	if details.DisplayName != nil {
		t.Fatalf("UpdateLifecycleEnvironment().DisplayName = %q, want unset", stringPointerValue(details.DisplayName))
	}
	if details.FreeformTags != nil {
		t.Fatalf("UpdateLifecycleEnvironment().FreeformTags = %v, want unset", details.FreeformTags)
	}
	if details.DefinedTags != nil {
		t.Fatalf("UpdateLifecycleEnvironment().DefinedTags = %v, want unset", details.DefinedTags)
	}
	if len(details.Stages) != 0 {
		t.Fatalf("UpdateLifecycleEnvironment().Stages length = %d, want 0", len(details.Stages))
	}
}

func assertLifecycleEnvironmentUpdateDetails(
	t *testing.T,
	details osmanagementhubsdk.UpdateLifecycleEnvironmentDetails,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
) {
	t.Helper()
	if got, want := stringPointerValue(details.DisplayName), spec.DisplayName; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().DisplayName = %q, want %q", got, want)
	}
	if got, want := stringPointerValue(details.Description), spec.Description; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().Description = %q, want %q", got, want)
	}
	if got, want := details.FreeformTags["env"], "prod"; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().FreeformTags[env] = %q, want %q", got, want)
	}
	if got, want := details.DefinedTags["Operations"]["CostCenter"], "99"; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().DefinedTags[Operations][CostCenter] = %v, want %q", got, want)
	}
	if got, want := len(details.Stages), 1; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().Stages length = %d, want %d", got, want)
	}
	assertLifecycleEnvironmentStageUpdate(t, details.Stages[0], spec.Stages[0])
}

func assertLifecycleEnvironmentStageUpdate(
	t *testing.T,
	stage osmanagementhubsdk.UpdateLifecycleStageDetails,
	spec osmanagementhubv1beta1.LifecycleEnvironmentStage,
) {
	t.Helper()
	stageID := testLifecycleEnvironmentID + ".stage." + strconv.Itoa(spec.Rank)
	if got, want := stringPointerValue(stage.Id), stageID; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().Stages[0].Id = %q, want %q", got, want)
	}
	if got, want := stringPointerValue(stage.DisplayName), spec.DisplayName; got != want {
		t.Fatalf("UpdateLifecycleEnvironment().Stages[0].DisplayName = %q, want %q", got, want)
	}
}

func assertLifecycleEnvironmentPendingDeleteStatus(
	t *testing.T,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	state string,
) {
	t.Helper()
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete projection")
	}
	if got, want := resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete; got != want {
		t.Fatalf("status.status.async.current.phase = %q, want %q", got, want)
	}
	if got, want := resource.Status.LifecycleState, state; got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
}

func sdkLifecycleEnvironment(
	id string,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	state osmanagementhubsdk.LifecycleEnvironmentLifecycleStateEnum,
) osmanagementhubsdk.LifecycleEnvironment {
	return osmanagementhubsdk.LifecycleEnvironment{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		Stages:         sdkLifecycleStages(id, spec.Stages),
		LifecycleState: state,
		OsFamily:       osmanagementhubsdk.OsFamilyEnum(spec.OsFamily),
		ArchType:       osmanagementhubsdk.ArchTypeEnum(spec.ArchType),
		VendorName:     osmanagementhubsdk.VendorNameEnum(spec.VendorName),
		Description:    common.String(spec.Description),
		Location:       osmanagementhubsdk.ManagedInstanceLocationEnum(spec.Location),
		FreeformTags:   cloneStringMap(spec.FreeformTags),
		DefinedTags:    definedTags(spec.DefinedTags),
	}
}

func sdkLifecycleEnvironmentSummary(
	id string,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	state osmanagementhubsdk.LifecycleEnvironmentLifecycleStateEnum,
) osmanagementhubsdk.LifecycleEnvironmentSummary {
	return osmanagementhubsdk.LifecycleEnvironmentSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		Description:    common.String(spec.Description),
		Stages:         sdkLifecycleStageSummaries(id, spec.Stages),
		ArchType:       osmanagementhubsdk.ArchTypeEnum(spec.ArchType),
		OsFamily:       osmanagementhubsdk.OsFamilyEnum(spec.OsFamily),
		VendorName:     osmanagementhubsdk.VendorNameEnum(spec.VendorName),
		LifecycleState: state,
		Location:       osmanagementhubsdk.ManagedInstanceLocationEnum(spec.Location),
		FreeformTags:   cloneStringMap(spec.FreeformTags),
		DefinedTags:    definedTags(spec.DefinedTags),
	}
}

func sdkLifecycleStages(
	lifecycleEnvironmentID string,
	stages []osmanagementhubv1beta1.LifecycleEnvironmentStage,
) []osmanagementhubsdk.LifecycleStage {
	converted := make([]osmanagementhubsdk.LifecycleStage, 0, len(stages))
	for _, stage := range stages {
		stageID := lifecycleEnvironmentID + ".stage." + strconv.Itoa(stage.Rank)
		converted = append(converted, osmanagementhubsdk.LifecycleStage{
			Id:                     common.String(stageID),
			LifecycleEnvironmentId: common.String(lifecycleEnvironmentID),
			DisplayName:            common.String(stage.DisplayName),
			Rank:                   common.Int(stage.Rank),
			CompartmentId:          common.String(testCompartmentID),
			OsFamily:               osmanagementhubsdk.OsFamilyOracleLinux8,
			ArchType:               osmanagementhubsdk.ArchTypeX8664,
			VendorName:             osmanagementhubsdk.VendorNameOracle,
			LifecycleState:         osmanagementhubsdk.LifecycleStageLifecycleStateActive,
			FreeformTags:           cloneStringMap(stage.FreeformTags),
			DefinedTags:            definedTags(stage.DefinedTags),
		})
	}
	return converted
}

func sdkLifecycleStageSummaries(
	lifecycleEnvironmentID string,
	stages []osmanagementhubv1beta1.LifecycleEnvironmentStage,
) []osmanagementhubsdk.LifecycleStageSummary {
	converted := make([]osmanagementhubsdk.LifecycleStageSummary, 0, len(stages))
	for _, stage := range stages {
		stageID := lifecycleEnvironmentID + ".stage." + strconv.Itoa(stage.Rank)
		converted = append(converted, osmanagementhubsdk.LifecycleStageSummary{
			Id:                     common.String(stageID),
			LifecycleEnvironmentId: common.String(lifecycleEnvironmentID),
			DisplayName:            common.String(stage.DisplayName),
			Rank:                   common.Int(stage.Rank),
			CompartmentId:          common.String(testCompartmentID),
			OsFamily:               osmanagementhubsdk.OsFamilyOracleLinux8,
			ArchType:               osmanagementhubsdk.ArchTypeX8664,
			VendorName:             osmanagementhubsdk.VendorNameOracle,
			LifecycleState:         osmanagementhubsdk.LifecycleStageLifecycleStateActive,
			FreeformTags:           cloneStringMap(stage.FreeformTags),
			DefinedTags:            definedTags(stage.DefinedTags),
		})
	}
	return converted
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func definedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(input))
	for namespace, tags := range input {
		converted[namespace] = make(map[string]interface{}, len(tags))
		for key, value := range tags {
			converted[namespace][key] = value
		}
	}
	return converted
}
