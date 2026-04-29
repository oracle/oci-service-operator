/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevisionnote

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testListingRevisionNoteID          = "ocid1.listingrevisionnote.oc1..note"
	testListingRevisionNoteOtherID     = "ocid1.listingrevisionnote.oc1..other"
	testListingRevisionNoteRevisionID  = "ocid1.listingrevision.oc1..revision"
	testListingRevisionNoteCompartment = "ocid1.compartment.oc1..listing"
	testListingRevisionNoteDetails     = "review note"
	testListingRevisionNoteCreateOpcID = "opc-create-note"
	testListingRevisionNoteUpdateOpcID = "opc-update-note"
	testListingRevisionNoteDeleteOpcID = "opc-delete-note"
)

type fakeListingRevisionNoteOCIClient struct {
	createFunc func(context.Context, marketplacepublishersdk.CreateListingRevisionNoteRequest) (marketplacepublishersdk.CreateListingRevisionNoteResponse, error)
	getFunc    func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error)
	listFunc   func(context.Context, marketplacepublishersdk.ListListingRevisionNotesRequest) (marketplacepublishersdk.ListListingRevisionNotesResponse, error)
	updateFunc func(context.Context, marketplacepublishersdk.UpdateListingRevisionNoteRequest) (marketplacepublishersdk.UpdateListingRevisionNoteResponse, error)
	deleteFunc func(context.Context, marketplacepublishersdk.DeleteListingRevisionNoteRequest) (marketplacepublishersdk.DeleteListingRevisionNoteResponse, error)

	createRequests []marketplacepublishersdk.CreateListingRevisionNoteRequest
	getRequests    []marketplacepublishersdk.GetListingRevisionNoteRequest
	listRequests   []marketplacepublishersdk.ListListingRevisionNotesRequest
	updateRequests []marketplacepublishersdk.UpdateListingRevisionNoteRequest
	deleteRequests []marketplacepublishersdk.DeleteListingRevisionNoteRequest
}

func (f *fakeListingRevisionNoteOCIClient) CreateListingRevisionNote(
	ctx context.Context,
	request marketplacepublishersdk.CreateListingRevisionNoteRequest,
) (marketplacepublishersdk.CreateListingRevisionNoteResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc == nil {
		return marketplacepublishersdk.CreateListingRevisionNoteResponse{}, fmt.Errorf("unexpected CreateListingRevisionNote call")
	}
	return f.createFunc(ctx, request)
}

func (f *fakeListingRevisionNoteOCIClient) GetListingRevisionNote(
	ctx context.Context,
	request marketplacepublishersdk.GetListingRevisionNoteRequest,
) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc == nil {
		return marketplacepublishersdk.GetListingRevisionNoteResponse{}, fmt.Errorf("unexpected GetListingRevisionNote call")
	}
	return f.getFunc(ctx, request)
}

func (f *fakeListingRevisionNoteOCIClient) ListListingRevisionNotes(
	ctx context.Context,
	request marketplacepublishersdk.ListListingRevisionNotesRequest,
) (marketplacepublishersdk.ListListingRevisionNotesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc == nil {
		return marketplacepublishersdk.ListListingRevisionNotesResponse{}, fmt.Errorf("unexpected ListListingRevisionNotes call")
	}
	return f.listFunc(ctx, request)
}

func (f *fakeListingRevisionNoteOCIClient) UpdateListingRevisionNote(
	ctx context.Context,
	request marketplacepublishersdk.UpdateListingRevisionNoteRequest,
) (marketplacepublishersdk.UpdateListingRevisionNoteResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc == nil {
		return marketplacepublishersdk.UpdateListingRevisionNoteResponse{}, fmt.Errorf("unexpected UpdateListingRevisionNote call")
	}
	return f.updateFunc(ctx, request)
}

func (f *fakeListingRevisionNoteOCIClient) DeleteListingRevisionNote(
	ctx context.Context,
	request marketplacepublishersdk.DeleteListingRevisionNoteRequest,
) (marketplacepublishersdk.DeleteListingRevisionNoteResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc == nil {
		return marketplacepublishersdk.DeleteListingRevisionNoteResponse{}, fmt.Errorf("unexpected DeleteListingRevisionNote call")
	}
	return f.deleteFunc(ctx, request)
}

func TestListingRevisionNoteRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := ListingRevisionNoteRuntimeHooks{}
	applyListingRevisionNoteRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed runtime semantics")
	}
	if hooks.Semantics.List == nil || len(hooks.Semantics.List.MatchFields) != 2 {
		t.Fatalf("hooks.Semantics.List = %#v, want listingRevisionId/noteDetails matching", hooks.Semantics.List)
	}
	if got, want := hooks.Semantics.Delete.Policy, "best-effort"; got != want {
		t.Fatalf("hooks.Semantics.Delete.Policy = %q, want %q", got, want)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want typed create body builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want typed update body builder")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("hooks.WrapGeneratedClient is empty, want pre-delete auth guard")
	}
}

func TestListingRevisionNoteCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	created := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	fake := &fakeListingRevisionNoteOCIClient{
		listFunc: func(context.Context, marketplacepublishersdk.ListListingRevisionNotesRequest) (marketplacepublishersdk.ListListingRevisionNotesResponse, error) {
			return marketplacepublishersdk.ListListingRevisionNotesResponse{}, nil
		},
		createFunc: func(_ context.Context, request marketplacepublishersdk.CreateListingRevisionNoteRequest) (marketplacepublishersdk.CreateListingRevisionNoteResponse, error) {
			assertListingRevisionNoteCreateRequest(t, request)
			return marketplacepublishersdk.CreateListingRevisionNoteResponse{
				ListingRevisionNote: created,
				OpcRequestId:        common.String(testListingRevisionNoteCreateOpcID),
			}, nil
		},
		getFunc: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			if got := stringValue(request.ListingRevisionNoteId); got != testListingRevisionNoteID {
				t.Fatalf("GetListingRevisionNoteRequest.ListingRevisionNoteId = %q, want %q", got, testListingRevisionNoteID)
			}
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: created}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate().ShouldRequeue = true, want false after ACTIVE readback")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateListingRevisionNote calls = %d, want 1", len(fake.createRequests))
	}
	assertListingRevisionNoteStatus(t, resource, testListingRevisionNoteID, testListingRevisionNoteCreateOpcID)
}

func TestListingRevisionNoteCreateOrUpdateBindsExistingFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	existing := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	administrator := newSDKListingRevisionNote(testListingRevisionNoteOtherID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	administrator.NoteSource = marketplacepublishersdk.ListingRevisionNoteNoteSourceAdministrator
	other := newSDKListingRevisionNote(testListingRevisionNoteOtherID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	other.NoteDetails = common.String("different note")
	fake := &fakeListingRevisionNoteOCIClient{
		listFunc: func(_ context.Context, request marketplacepublishersdk.ListListingRevisionNotesRequest) (marketplacepublishersdk.ListListingRevisionNotesResponse, error) {
			if got := stringValue(request.ListingRevisionId); got != testListingRevisionNoteRevisionID {
				t.Fatalf("ListListingRevisionNotesRequest.ListingRevisionId = %q, want %q", got, testListingRevisionNoteRevisionID)
			}
			if request.Page == nil {
				return marketplacepublishersdk.ListListingRevisionNotesResponse{
					ListingRevisionNoteCollection: marketplacepublishersdk.ListingRevisionNoteCollection{
						Items: []marketplacepublishersdk.ListingRevisionNoteSummary{
							listingRevisionNoteSummaryFromSDK(administrator),
							listingRevisionNoteSummaryFromSDK(other),
						},
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			}
			if got := stringValue(request.Page); got != "next-page" {
				t.Fatalf("ListListingRevisionNotesRequest.Page = %q, want next-page", got)
			}
			return marketplacepublishersdk.ListListingRevisionNotesResponse{
				ListingRevisionNoteCollection: marketplacepublishersdk.ListingRevisionNoteCollection{
					Items: []marketplacepublishersdk.ListingRevisionNoteSummary{listingRevisionNoteSummaryFromSDK(existing)},
				},
			}, nil
		},
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: existing}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListListingRevisionNotes calls = %d, want 2 pages", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateListingRevisionNote calls = %d, want 0 for bind", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateListingRevisionNote calls = %d, want 0 for matching bind", len(fake.updateRequests))
	}
	assertListingRevisionNoteStatus(t, resource, testListingRevisionNoteID, "")
}

func TestListingRevisionNoteCreateOrUpdateSkipsAdministratorListMatchBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	administrator := newSDKListingRevisionNote(testListingRevisionNoteOtherID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	administrator.NoteSource = marketplacepublishersdk.ListingRevisionNoteNoteSourceAdministrator
	created := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	fake := &fakeListingRevisionNoteOCIClient{
		listFunc: func(context.Context, marketplacepublishersdk.ListListingRevisionNotesRequest) (marketplacepublishersdk.ListListingRevisionNotesResponse, error) {
			return marketplacepublishersdk.ListListingRevisionNotesResponse{
				ListingRevisionNoteCollection: marketplacepublishersdk.ListingRevisionNoteCollection{
					Items: []marketplacepublishersdk.ListingRevisionNoteSummary{listingRevisionNoteSummaryFromSDK(administrator)},
				},
			}, nil
		},
		createFunc: func(_ context.Context, request marketplacepublishersdk.CreateListingRevisionNoteRequest) (marketplacepublishersdk.CreateListingRevisionNoteResponse, error) {
			assertListingRevisionNoteCreateRequest(t, request)
			return marketplacepublishersdk.CreateListingRevisionNoteResponse{
				ListingRevisionNote: created,
				OpcRequestId:        common.String(testListingRevisionNoteCreateOpcID),
			}, nil
		},
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: created}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateListingRevisionNote calls = %d, want 1 after skipping ADMINISTRATOR note", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateListingRevisionNote calls = %d, want 0 after create", len(fake.updateRequests))
	}
	assertListingRevisionNoteStatus(t, resource, testListingRevisionNoteID, testListingRevisionNoteCreateOpcID)
}

func TestListingRevisionNoteCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionNoteID)
	current := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	fake := &fakeListingRevisionNoteOCIClient{
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: current}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateListingRevisionNote calls = %d, want 0 for no-op reconcile", len(fake.updateRequests))
	}
	assertListingRevisionNoteStatus(t, resource, testListingRevisionNoteID, "")
}

func TestListingRevisionNoteCreateOrUpdateUpdatesMutableTags(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionNoteID)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	current := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	updated := current
	updated.FreeformTags = map[string]string{"env": "prod"}
	getCount := 0
	fake := &fakeListingRevisionNoteOCIClient{
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			getCount++
			if getCount == 1 {
				return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: current}, nil
			}
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: updated}, nil
		},
		updateFunc: func(_ context.Context, request marketplacepublishersdk.UpdateListingRevisionNoteRequest) (marketplacepublishersdk.UpdateListingRevisionNoteResponse, error) {
			if got := stringValue(request.ListingRevisionNoteId); got != testListingRevisionNoteID {
				t.Fatalf("UpdateListingRevisionNoteRequest.ListingRevisionNoteId = %q, want %q", got, testListingRevisionNoteID)
			}
			if got := request.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateListingRevisionNoteDetails.FreeformTags[env] = %q, want prod", got)
			}
			return marketplacepublishersdk.UpdateListingRevisionNoteResponse{
				ListingRevisionNote: updated,
				OpcRequestId:        common.String(testListingRevisionNoteUpdateOpcID),
			}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateListingRevisionNote calls = %d, want 1", len(fake.updateRequests))
	}
	assertListingRevisionNoteStatus(t, resource, testListingRevisionNoteID, testListingRevisionNoteUpdateOpcID)
}

func TestListingRevisionNoteCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionNoteID)
	resource.Spec.NoteDetails = "changed note"
	current := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	fake := &fakeListingRevisionNoteOCIClient{
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: current}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "noteDetails") {
		t.Fatalf("CreateOrUpdate() error = %v, want noteDetails create-only drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false for create-only drift")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateListingRevisionNote calls = %d, want 0 after pre-OCI drift rejection", len(fake.updateRequests))
	}
}

func TestListingRevisionNoteCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	fake := &fakeListingRevisionNoteOCIClient{
		listFunc: func(context.Context, marketplacepublishersdk.ListListingRevisionNotesRequest) (marketplacepublishersdk.ListListingRevisionNotesResponse, error) {
			return marketplacepublishersdk.ListListingRevisionNotesResponse{}, nil
		},
		createFunc: func(context.Context, marketplacepublishersdk.CreateListingRevisionNoteRequest) (marketplacepublishersdk.CreateListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.CreateListingRevisionNoteResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false for create error")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	if got, want := resource.Status.OsokStatus.Reason, string(shared.Failed); got != want {
		t.Fatalf("status.reason = %q, want %q", got, want)
	}
}

func TestListingRevisionNoteDeleteConfirmsDeletedLifecycle(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionNoteID)
	active := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	deletedNote := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateDeleted)
	getCount := 0
	fake := &fakeListingRevisionNoteOCIClient{
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			getCount++
			if getCount < 3 {
				return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: active}, nil
			}
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: deletedNote}, nil
		},
		deleteFunc: func(context.Context, marketplacepublishersdk.DeleteListingRevisionNoteRequest) (marketplacepublishersdk.DeleteListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.DeleteListingRevisionNoteResponse{OpcRequestId: common.String(testListingRevisionNoteDeleteOpcID)}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED lifecycle readback")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteListingRevisionNote calls = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != testListingRevisionNoteDeleteOpcID {
		t.Fatalf("status.opcRequestId = %q, want %q", got, testListingRevisionNoteDeleteOpcID)
	}
}

func TestListingRevisionNoteDeleteRetainsFinalizerWhileReadbackRemainsActive(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionNoteID)
	active := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	fake := &fakeListingRevisionNoteOCIClient{
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: active}, nil
		},
		deleteFunc: func(context.Context, marketplacepublishersdk.DeleteListingRevisionNoteRequest) (marketplacepublishersdk.DeleteListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.DeleteListingRevisionNoteResponse{OpcRequestId: common.String(testListingRevisionNoteDeleteOpcID)}, nil
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI readback remains ACTIVE")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got, want := resource.Status.OsokStatus.Reason, string(shared.Terminating); got != want {
		t.Fatalf("status.reason = %q, want %q", got, want)
	}
}

func TestListingRevisionNoteDeleteRejectsAuthShapedDeleteNotFound(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionNoteID)
	active := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)
	fake := &fakeListingRevisionNoteOCIClient{
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: active}, nil
		},
		deleteFunc: func(context.Context, marketplacepublishersdk.DeleteListingRevisionNoteRequest) (marketplacepublishersdk.DeleteListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.DeleteListingRevisionNoteResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped not-found error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestListingRevisionNoteDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionNoteID)
	fake := &fakeListingRevisionNoteOCIClient{
		getFunc: func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.GetListingRevisionNoteResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFunc: func(context.Context, marketplacepublishersdk.DeleteListingRevisionNoteRequest) (marketplacepublishersdk.DeleteListingRevisionNoteResponse, error) {
			return marketplacepublishersdk.DeleteListingRevisionNoteResponse{}, fmt.Errorf("delete should not run")
		},
	}
	client := newTestListingRevisionNoteServiceClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative pre-delete auth-shaped error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteListingRevisionNote calls = %d, want 0 after auth-shaped pre-delete read", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped pre-delete read", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestListingRevisionNoteBuildUpdateBodyPreservesExplicitEmptyTagClears(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	current := newSDKListingRevisionNote(testListingRevisionNoteID, marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)

	body, updateNeeded, err := buildListingRevisionNoteUpdateBody(resource, marketplacepublishersdk.GetListingRevisionNoteResponse{ListingRevisionNote: current})
	if err != nil {
		t.Fatalf("buildListingRevisionNoteUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildListingRevisionNoteUpdateBody() updateNeeded = false, want true for explicit empty tag clear")
	}
	if body.FreeformTags == nil || len(body.FreeformTags) != 0 {
		t.Fatalf("UpdateListingRevisionNoteDetails.FreeformTags = %#v, want explicit empty map", body.FreeformTags)
	}
	if body.DefinedTags == nil || len(body.DefinedTags) != 0 {
		t.Fatalf("UpdateListingRevisionNoteDetails.DefinedTags = %#v, want explicit empty map", body.DefinedTags)
	}
}

func newTestListingRevisionNoteServiceClient(fake *fakeListingRevisionNoteOCIClient) ListingRevisionNoteServiceClient {
	hooks := ListingRevisionNoteRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*marketplacepublisherv1beta1.ListingRevisionNote]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*marketplacepublisherv1beta1.ListingRevisionNote]{},
		StatusHooks:     generatedruntime.StatusHooks[*marketplacepublisherv1beta1.ListingRevisionNote]{},
		ParityHooks:     generatedruntime.ParityHooks[*marketplacepublisherv1beta1.ListingRevisionNote]{},
		Async:           generatedruntime.AsyncHooks[*marketplacepublisherv1beta1.ListingRevisionNote]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*marketplacepublisherv1beta1.ListingRevisionNote]{},
		Create: runtimeOperationHooks[marketplacepublishersdk.CreateListingRevisionNoteRequest, marketplacepublishersdk.CreateListingRevisionNoteResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateListingRevisionNoteDetails", RequestName: "CreateListingRevisionNoteDetails", Contribution: "body"}},
			Call:   fake.CreateListingRevisionNote,
		},
		Get: runtimeOperationHooks[marketplacepublishersdk.GetListingRevisionNoteRequest, marketplacepublishersdk.GetListingRevisionNoteResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ListingRevisionNoteId", RequestName: "listingRevisionNoteId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.GetListingRevisionNote,
		},
		List: runtimeOperationHooks[marketplacepublishersdk.ListListingRevisionNotesRequest, marketplacepublishersdk.ListListingRevisionNotesResponse]{
			Fields: listingRevisionNoteListFields(),
			Call:   fake.ListListingRevisionNotes,
		},
		Update: runtimeOperationHooks[marketplacepublishersdk.UpdateListingRevisionNoteRequest, marketplacepublishersdk.UpdateListingRevisionNoteResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ListingRevisionNoteId", RequestName: "listingRevisionNoteId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateListingRevisionNoteDetails", RequestName: "UpdateListingRevisionNoteDetails", Contribution: "body"},
			},
			Call: fake.UpdateListingRevisionNote,
		},
		Delete: runtimeOperationHooks[marketplacepublishersdk.DeleteListingRevisionNoteRequest, marketplacepublishersdk.DeleteListingRevisionNoteResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ListingRevisionNoteId", RequestName: "listingRevisionNoteId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.DeleteListingRevisionNote,
		},
		WrapGeneratedClient: []func(ListingRevisionNoteServiceClient) ListingRevisionNoteServiceClient{},
	}
	applyListingRevisionNoteRuntimeHooks(&hooks)
	delegate := defaultListingRevisionNoteServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.ListingRevisionNote](
			buildListingRevisionNoteGeneratedRuntimeConfig(&ListingRevisionNoteServiceManager{}, hooks),
		),
	}
	return wrapListingRevisionNoteGeneratedClient(hooks, delegate)
}

func newListingRevisionNoteResource() *marketplacepublisherv1beta1.ListingRevisionNote {
	return &marketplacepublisherv1beta1.ListingRevisionNote{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "listing-revision-note",
			Namespace: "default",
		},
		Spec: marketplacepublisherv1beta1.ListingRevisionNoteSpec{
			ListingRevisionId: testListingRevisionNoteRevisionID,
			NoteDetails:       testListingRevisionNoteDetails,
			FreeformTags:      map[string]string{"env": "test"},
			DefinedTags:       map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func newSDKListingRevisionNote(
	id string,
	state marketplacepublishersdk.ListingRevisionNoteLifecycleStateEnum,
) marketplacepublishersdk.ListingRevisionNote {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.ListingRevisionNote{
		Id:                common.String(id),
		ListingRevisionId: common.String(testListingRevisionNoteRevisionID),
		CompartmentId:     common.String(testListingRevisionNoteCompartment),
		NoteSource:        marketplacepublishersdk.ListingRevisionNoteNoteSourcePublisher,
		NoteDetails:       common.String(testListingRevisionNoteDetails),
		TimeCreated:       &now,
		TimeUpdated:       &now,
		LifecycleState:    state,
		FreeformTags:      map[string]string{"env": "test"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:        map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func listingRevisionNoteSummaryFromSDK(note marketplacepublishersdk.ListingRevisionNote) marketplacepublishersdk.ListingRevisionNoteSummary {
	return marketplacepublishersdk.ListingRevisionNoteSummary(note)
}

func assertListingRevisionNoteCreateRequest(t *testing.T, request marketplacepublishersdk.CreateListingRevisionNoteRequest) {
	t.Helper()
	if got := stringValue(request.ListingRevisionId); got != testListingRevisionNoteRevisionID {
		t.Fatalf("CreateListingRevisionNoteDetails.ListingRevisionId = %q, want %q", got, testListingRevisionNoteRevisionID)
	}
	if got := stringValue(request.NoteDetails); got != testListingRevisionNoteDetails {
		t.Fatalf("CreateListingRevisionNoteDetails.NoteDetails = %q, want %q", got, testListingRevisionNoteDetails)
	}
	if got := request.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateListingRevisionNoteDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateListingRevisionNoteDetails.DefinedTags[Operations][CostCenter] = %#v, want 42", got)
	}
}

func assertListingRevisionNoteStatus(
	t *testing.T,
	resource *marketplacepublisherv1beta1.ListingRevisionNote,
	id string,
	requestID string,
) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	if got := resource.Status.NoteDetails; got != testListingRevisionNoteDetails {
		t.Fatalf("status.noteDetails = %q, want %q", got, testListingRevisionNoteDetails)
	}
	if got := resource.Status.LifecycleState; got != string(marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if requestID != "" {
		if got := resource.Status.OsokStatus.OpcRequestID; got != requestID {
			t.Fatalf("status.status.opcRequestId = %q, want %q", got, requestID)
		}
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
