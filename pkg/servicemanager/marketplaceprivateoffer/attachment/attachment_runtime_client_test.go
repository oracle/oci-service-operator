/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package attachment

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	marketplaceprivateoffersdk "github.com/oracle/oci-go-sdk/v65/marketplaceprivateoffer"
	marketplaceprivateofferv1beta1 "github.com/oracle/oci-service-operator/api/marketplaceprivateoffer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAttachmentID          = "ocid1.attachment.oc1..runtime"
	testAttachmentOfferID     = "ocid1.offer.oc1..runtime"
	testAttachmentSellerID    = "ocid1.tenancy.oc1..seller"
	testAttachmentBuyerID     = "ocid1.tenancy.oc1..buyer"
	testAttachmentDisplayName = "contract.pdf"
	testAttachmentFileBase64  = "JVBERi0xLjQK"
	testAttachmentFileContent = "%PDF-1.4\n"
	testAttachmentType        = string(marketplaceprivateoffersdk.AttachmentTypeContractTAndC)
)

type fakeAttachmentOCIClient struct {
	createFn func(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error)
	getFn    func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error)
	listFn   func(context.Context, marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error)
	deleteFn func(context.Context, marketplaceprivateoffersdk.DeleteAttachmentRequest) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error)

	createRequests []marketplaceprivateoffersdk.CreateAttachmentRequest
	getRequests    []marketplaceprivateoffersdk.GetAttachmentRequest
	listRequests   []marketplaceprivateoffersdk.ListAttachmentsRequest
	deleteRequests []marketplaceprivateoffersdk.DeleteAttachmentRequest
}

func (f *fakeAttachmentOCIClient) CreateAttachment(
	ctx context.Context,
	req marketplaceprivateoffersdk.CreateAttachmentRequest,
) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return marketplaceprivateoffersdk.CreateAttachmentResponse{}, nil
}

func (f *fakeAttachmentOCIClient) GetAttachment(
	ctx context.Context,
	req marketplaceprivateoffersdk.GetAttachmentRequest,
) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return marketplaceprivateoffersdk.GetAttachmentResponse{}, nil
}

func (f *fakeAttachmentOCIClient) ListAttachments(
	ctx context.Context,
	req marketplaceprivateoffersdk.ListAttachmentsRequest,
) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return marketplaceprivateoffersdk.ListAttachmentsResponse{}, nil
}

func (f *fakeAttachmentOCIClient) DeleteAttachment(
	ctx context.Context,
	req marketplaceprivateoffersdk.DeleteAttachmentRequest,
) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return marketplaceprivateoffersdk.DeleteAttachmentResponse{}, nil
}

func testAttachmentClient(fake *fakeAttachmentOCIClient) AttachmentServiceClient {
	return newAttachmentServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func newAttachmentTestResource() *marketplaceprivateofferv1beta1.Attachment {
	return &marketplaceprivateofferv1beta1.Attachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "attachment",
			Namespace: "default",
			Annotations: map[string]string{
				attachmentOfferIDAnnotation: testAttachmentOfferID,
			},
		},
		Spec: marketplaceprivateofferv1beta1.AttachmentSpec{
			FileBase64Encoded: testAttachmentFileBase64,
			DisplayName:       testAttachmentDisplayName,
			Type:              testAttachmentType,
		},
	}
}

func newExistingAttachmentTestResource(id string) *marketplaceprivateofferv1beta1.Attachment {
	resource := newAttachmentTestResource()
	resource.Status.Id = id
	resource.Status.OfferId = testAttachmentOfferID
	resource.Status.DisplayName = testAttachmentDisplayName
	resource.Status.Type = testAttachmentType
	resource.Status.LifecycleState = string(marketplaceprivateoffersdk.AttachmentLifecycleStateActive)
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	recordAttachmentFileFingerprint(resource)
	return resource
}

func newSDKAttachment(
	id string,
	offerID string,
	displayName string,
	attachmentType marketplaceprivateoffersdk.AttachmentTypeEnum,
	state marketplaceprivateoffersdk.AttachmentLifecycleStateEnum,
) marketplaceprivateoffersdk.Attachment {
	return marketplaceprivateoffersdk.Attachment{
		Id:                  common.String(id),
		SellerCompartmentId: common.String(testAttachmentSellerID),
		OfferId:             common.String(offerID),
		DisplayName:         common.String(displayName),
		Type:                attachmentType,
		LifecycleState:      state,
		FreeformTags:        map[string]string{"scenario": "runtime"},
		DefinedTags:         map[string]map[string]interface{}{"ns": {"key": "value"}},
		BuyerCompartmentId:  common.String(testAttachmentBuyerID),
		MimeType:            common.String("application/pdf"),
	}
}

func newSDKAttachmentSummary(
	id string,
	offerID string,
	displayName string,
	attachmentType marketplaceprivateoffersdk.AttachmentTypeEnum,
	state marketplaceprivateoffersdk.AttachmentLifecycleStateEnum,
) marketplaceprivateoffersdk.AttachmentSummary {
	return marketplaceprivateoffersdk.AttachmentSummary{
		Id:             common.String(id),
		OfferId:        common.String(offerID),
		DisplayName:    common.String(displayName),
		Type:           attachmentType,
		LifecycleState: state,
		FreeformTags:   map[string]string{"scenario": "runtime"},
		DefinedTags:    map[string]map[string]interface{}{"ns": {"key": "value"}},
		MimeType:       common.String("application/pdf"),
	}
}

func attachmentCreateSuccessFn(
	t *testing.T,
	opcRequestID string,
) func(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
	t.Helper()
	return func(_ context.Context, req marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
		t.Helper()
		requireAttachmentStringPtr(t, "create offerId", req.OfferId, testAttachmentOfferID)
		requireAttachmentStringPtr(t, "create displayName", req.DisplayName, testAttachmentDisplayName)
		if got, want := string(req.FileBase64Encoded), testAttachmentFileContent; got != want {
			t.Fatalf("create file bytes = %q, want decoded content %q", got, want)
		}
		if req.Type != marketplaceprivateoffersdk.AttachmentTypeContractTAndC {
			t.Fatalf("create type = %q, want %q", req.Type, marketplaceprivateoffersdk.AttachmentTypeContractTAndC)
		}
		if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
			t.Fatal("create opcRetryToken is empty, want deterministic retry token")
		}
		return marketplaceprivateoffersdk.CreateAttachmentResponse{
			Attachment:   newSDKAttachment(testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateCreating),
			OpcRequestId: common.String(opcRequestID),
		}, nil
	}
}

func attachmentGetActiveFn(
	t *testing.T,
	attachmentID string,
) func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
	t.Helper()
	return func(_ context.Context, req marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
		t.Helper()
		requireAttachmentStringPtr(t, "get offerId", req.OfferId, testAttachmentOfferID)
		requireAttachmentStringPtr(t, "get attachmentId", req.AttachmentId, attachmentID)
		return marketplaceprivateoffersdk.GetAttachmentResponse{
			Attachment: newSDKAttachment(attachmentID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
		}, nil
	}
}

func TestAttachmentCreateOrUpdateCreatesAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	fake := &fakeAttachmentOCIClient{}
	fake.listFn = func(context.Context, marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
		return marketplaceprivateoffersdk.ListAttachmentsResponse{}, nil
	}
	fake.createFn = attachmentCreateSuccessFn(t, "opc-create-1")
	fake.getFn = attachmentGetActiveFn(t, testAttachmentID)

	resource := newAttachmentTestResource()
	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireAttachmentCreateSuccess(t, response, err, resource, fake, "opc-create-1")
}

func requireAttachmentCreateSuccess(
	t *testing.T,
	response servicemanager.OSOKResponse,
	err error,
	resource *marketplaceprivateofferv1beta1.Attachment,
	fake *fakeAttachmentOCIClient,
	wantOpcRequestID string,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE readback")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListAttachments() calls = %d, want 1", len(fake.listRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetAttachment() calls = %d, want 1", len(fake.getRequests))
	}
	requireAttachmentStatus(t, resource, testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, testAttachmentType, "ACTIVE")
	if got := resource.Status.OsokStatus.OpcRequestID; got != wantOpcRequestID {
		t.Fatalf("status.opcRequestId = %q, want %q", got, wantOpcRequestID)
	}
	requireAttachmentFileFingerprint(t, resource, testAttachmentFileBase64)
}

func TestAttachmentCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.attachment.oc1..existing"
	var pages []string

	fake := &fakeAttachmentOCIClient{}
	fake.listFn = func(_ context.Context, req marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
		pages = append(pages, attachmentString(req.Page))
		requireAttachmentStringPtr(t, "list offerId", req.OfferId, testAttachmentOfferID)
		requireAttachmentStringPtr(t, "list displayName", req.DisplayName, testAttachmentDisplayName)
		switch attachmentString(req.Page) {
		case "":
			return marketplaceprivateoffersdk.ListAttachmentsResponse{
				AttachmentCollection: marketplaceprivateoffersdk.AttachmentCollection{
					Items: []marketplaceprivateoffersdk.AttachmentSummary{
						newSDKAttachmentSummary("ocid1.attachment.oc1..other", testAttachmentOfferID, "other.pdf", marketplaceprivateoffersdk.AttachmentTypeQuote, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return marketplaceprivateoffersdk.ListAttachmentsResponse{
				AttachmentCollection: marketplaceprivateoffersdk.AttachmentCollection{
					Items: []marketplaceprivateoffersdk.AttachmentSummary{
						newSDKAttachmentSummary(existingID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page = %q", attachmentString(req.Page))
			return marketplaceprivateoffersdk.ListAttachmentsResponse{}, nil
		}
	}
	fake.createFn = func(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
		t.Fatal("CreateAttachment() should not run when list finds existing Attachment")
		return marketplaceprivateoffersdk.CreateAttachmentResponse{}, nil
	}
	fake.getFn = func(_ context.Context, req marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
		requireAttachmentStringPtr(t, "get offerId", req.OfferId, testAttachmentOfferID)
		requireAttachmentStringPtr(t, "get attachmentId", req.AttachmentId, existingID)
		return marketplaceprivateoffersdk.GetAttachmentResponse{
			Attachment: newSDKAttachment(existingID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
		}, nil
	}

	resource := newAttachmentTestResource()
	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if got, want := strings.Join(pages, ","), ",page-2"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateAttachment() calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetAttachment() calls = %d, want 1 after bind", len(fake.getRequests))
	}
	requireAttachmentStatus(t, resource, existingID, testAttachmentOfferID, testAttachmentDisplayName, testAttachmentType, "ACTIVE")
	requireAttachmentFileFingerprint(t, resource, testAttachmentFileBase64)
}

func TestAttachmentCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	fake := &fakeAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
			requireAttachmentStringPtr(t, "get offerId", req.OfferId, testAttachmentOfferID)
			requireAttachmentStringPtr(t, "get attachmentId", req.AttachmentId, testAttachmentID)
			return marketplaceprivateoffersdk.GetAttachmentResponse{
				Attachment: newSDKAttachment(testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
			}, nil
		},
	}

	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateAttachment() calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("ListAttachments() calls = %d, want 0", len(fake.listRequests))
	}
	requireAttachmentStatus(t, resource, testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, testAttachmentType, "ACTIVE")
	requireAttachmentFileFingerprint(t, resource, testAttachmentFileBase64)
}

func TestAttachmentCreateOrUpdateNoopsWithLowercaseSupportedType(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	resource.Spec.Type = "contract_t_and_c"
	fake := &fakeAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
			requireAttachmentStringPtr(t, "get offerId", req.OfferId, testAttachmentOfferID)
			requireAttachmentStringPtr(t, "get attachmentId", req.AttachmentId, testAttachmentID)
			return marketplaceprivateoffersdk.GetAttachmentResponse{
				Attachment: newSDKAttachment(testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
			}, nil
		},
	}

	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateAttachment() calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("ListAttachments() calls = %d, want 0", len(fake.listRequests))
	}
	requireAttachmentStatus(t, resource, testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, testAttachmentType, "ACTIVE")
	requireAttachmentFileFingerprint(t, resource, testAttachmentFileBase64)
}

func TestAttachmentRejectsFileDriftWithoutOCIUpdate(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	recorded, ok := attachmentRecordedFileFingerprint(resource)
	if !ok {
		t.Fatal("test setup did not record file fingerprint")
	}
	resource.Spec.FileBase64Encoded = "cmVwbGFjZW1lbnQK"

	fake := &fakeAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
			requireAttachmentStringPtr(t, "get offerId", req.OfferId, testAttachmentOfferID)
			requireAttachmentStringPtr(t, "get attachmentId", req.AttachmentId, testAttachmentID)
			return marketplaceprivateoffersdk.GetAttachmentResponse{
				Attachment: newSDKAttachment(testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
			}, nil
		},
	}

	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only file drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for file drift")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateAttachment() calls = %d, want 0", len(fake.createRequests))
	}
	if !strings.Contains(err.Error(), "fileBase64Encoded") {
		t.Fatalf("CreateOrUpdate() error = %v, want fileBase64Encoded drift context", err)
	}
	if got, ok := attachmentRecordedFileFingerprint(resource); !ok || got != recorded {
		t.Fatalf("recorded fingerprint = %q, %t, want preserved %q", got, ok, recorded)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestAttachmentRejectsMissingFileFingerprintForTrackedResource(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	resource.Status.OsokStatus.Message = "status from older controller"
	resource.Spec.FileBase64Encoded = "cmVwbGFjZW1lbnQK"

	fake := &fakeAttachmentOCIClient{
		getFn: func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
			t.Fatal("GetAttachment() should not run when tracked Attachment lacks file fingerprint")
			return marketplaceprivateoffersdk.GetAttachmentResponse{}, nil
		},
		createFn: func(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
			t.Fatal("CreateAttachment() should not run when tracked Attachment lacks file fingerprint")
			return marketplaceprivateoffersdk.CreateAttachmentResponse{}, nil
		},
	}

	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing file fingerprint rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for missing file fingerprint")
	}
	if len(fake.getRequests) != 0 || len(fake.createRequests) != 0 || len(fake.listRequests) != 0 {
		t.Fatalf("OCI calls = get %d list %d create %d, want none", len(fake.getRequests), len(fake.listRequests), len(fake.createRequests))
	}
	if !strings.Contains(err.Error(), "fileBase64Encoded") || !strings.Contains(err.Error(), attachmentFileSHA256Key) {
		t.Fatalf("CreateOrUpdate() error = %v, want file fingerprint context", err)
	}
	if got, ok := attachmentRecordedFileFingerprint(resource); ok {
		t.Fatalf("recorded fingerprint = %q, want none because missing status marker must not be re-baselined", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestAttachmentRejectsClearedDisplayNameForTrackedResource(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	resource.Spec.DisplayName = " "

	fake := &fakeAttachmentOCIClient{
		getFn: func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
			t.Fatal("GetAttachment() should not run when tracked Attachment has cleared displayName")
			return marketplaceprivateoffersdk.GetAttachmentResponse{}, nil
		},
		listFn: func(context.Context, marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
			t.Fatal("ListAttachments() should not run when tracked Attachment has cleared displayName")
			return marketplaceprivateoffersdk.ListAttachmentsResponse{}, nil
		},
		createFn: func(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
			t.Fatal("CreateAttachment() should not run when tracked Attachment has cleared displayName")
			return marketplaceprivateoffersdk.CreateAttachmentResponse{}, nil
		},
	}

	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want required displayName rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for cleared displayName")
	}
	if len(fake.getRequests) != 0 || len(fake.listRequests) != 0 || len(fake.createRequests) != 0 {
		t.Fatalf("OCI calls = get %d list %d create %d, want none", len(fake.getRequests), len(fake.listRequests), len(fake.createRequests))
	}
	if !strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName context", err)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
	requireAttachmentFileFingerprint(t, resource, testAttachmentFileBase64)
}

func TestAttachmentRejectsDisplayNameDriftWithoutOCIUpdate(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	resource.Spec.DisplayName = "replacement.pdf"

	fake := &fakeAttachmentOCIClient{
		getFn: func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
			return marketplaceprivateoffersdk.GetAttachmentResponse{
				Attachment: newSDKAttachment(testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, marketplaceprivateoffersdk.AttachmentLifecycleStateActive),
			}, nil
		},
	}

	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only displayName drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for displayName drift")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateAttachment() calls = %d, want 0", len(fake.createRequests))
	}
	if !strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName drift context", err)
	}
}

func TestAttachmentRequiresOfferAnnotationBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := newAttachmentTestResource()
	resource.Annotations = nil
	fake := &fakeAttachmentOCIClient{
		createFn: func(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
			t.Fatal("CreateAttachment() should not run without parent offer annotation")
			return marketplaceprivateoffersdk.CreateAttachmentResponse{}, nil
		},
	}

	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing offer annotation error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if !strings.Contains(err.Error(), attachmentOfferIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want %s context", err, attachmentOfferIDAnnotation)
	}
	if len(fake.createRequests) != 0 || len(fake.listRequests) != 0 {
		t.Fatalf("OCI calls made before annotation validation: list=%d create=%d", len(fake.listRequests), len(fake.createRequests))
	}
}

func attachmentGetDeletingOnThirdReadFn(
	t *testing.T,
	fake *fakeAttachmentOCIClient,
) func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
	t.Helper()
	return func(_ context.Context, req marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
		t.Helper()
		requireAttachmentStringPtr(t, "get offerId", req.OfferId, testAttachmentOfferID)
		requireAttachmentStringPtr(t, "get attachmentId", req.AttachmentId, testAttachmentID)
		state := marketplaceprivateoffersdk.AttachmentLifecycleStateActive
		if len(fake.getRequests) == 3 {
			state = marketplaceprivateoffersdk.AttachmentLifecycleStateDeleting
		}
		return marketplaceprivateoffersdk.GetAttachmentResponse{
			Attachment: newSDKAttachment(testAttachmentID, testAttachmentOfferID, testAttachmentDisplayName, marketplaceprivateoffersdk.AttachmentTypeContractTAndC, state),
		}, nil
	}
}

func attachmentDeleteAcceptedFn(
	t *testing.T,
) func(context.Context, marketplaceprivateoffersdk.DeleteAttachmentRequest) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error) {
	t.Helper()
	return func(_ context.Context, req marketplaceprivateoffersdk.DeleteAttachmentRequest) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error) {
		t.Helper()
		requireAttachmentStringPtr(t, "delete offerId", req.OfferId, testAttachmentOfferID)
		requireAttachmentStringPtr(t, "delete attachmentId", req.AttachmentId, testAttachmentID)
		return marketplaceprivateoffersdk.DeleteAttachmentResponse{
			OpcRequestId:     common.String("opc-delete-1"),
			OpcWorkRequestId: common.String("wr-delete-1"),
		}, nil
	}
}

func TestAttachmentDeleteRetainsFinalizerWhileLifecycleDeleting(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	fake := &fakeAttachmentOCIClient{}
	fake.getFn = attachmentGetDeletingOnThirdReadFn(t, fake)
	fake.deleteFn = attachmentDeleteAcceptedFn(t)

	deleted, err := testAttachmentClient(fake).Delete(context.Background(), resource)
	requireAttachmentDeletePending(t, deleted, err, resource, fake)
}

func requireAttachmentDeletePending(
	t *testing.T,
	deleted bool,
	err error,
	resource *marketplaceprivateofferv1beta1.Attachment,
	fake *fakeAttachmentOCIClient,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI lifecycle is DELETING")
	}
	if len(fake.getRequests) != 3 {
		t.Fatalf("GetAttachment() calls = %d, want 3", len(fake.getRequests))
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteAttachment() calls = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete phase", current)
	}
}

func TestAttachmentDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newExistingAttachmentTestResource(testAttachmentID)
	fake := &fakeAttachmentOCIClient{
		getFn: func(context.Context, marketplaceprivateoffersdk.GetAttachmentRequest) (marketplaceprivateoffersdk.GetAttachmentResponse, error) {
			return marketplaceprivateoffersdk.GetAttachmentResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
		deleteFn: func(context.Context, marketplaceprivateoffersdk.DeleteAttachmentRequest) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error) {
			t.Fatal("DeleteAttachment() should not run after ambiguous confirm read")
			return marketplaceprivateoffersdk.DeleteAttachmentResponse{}, nil
		},
	}

	deleted, err := testAttachmentClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteAttachment() calls = %d, want 0", len(fake.deleteRequests))
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 context", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for ambiguous 404", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestAttachmentDeleteConfirmsUntrackedNoMatchByList(t *testing.T) {
	t.Parallel()

	resource := newAttachmentTestResource()
	resource.Annotations = nil
	resource.Status.OfferId = testAttachmentOfferID
	resource.Status.DisplayName = testAttachmentDisplayName
	resource.Status.Type = testAttachmentType

	fake := &fakeAttachmentOCIClient{
		listFn: func(_ context.Context, req marketplaceprivateoffersdk.ListAttachmentsRequest) (marketplaceprivateoffersdk.ListAttachmentsResponse, error) {
			requireAttachmentStringPtr(t, "list offerId", req.OfferId, testAttachmentOfferID)
			requireAttachmentStringPtr(t, "list displayName", req.DisplayName, testAttachmentDisplayName)
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil for untracked delete confirmation", req.Id)
			}
			return marketplaceprivateoffersdk.ListAttachmentsResponse{}, nil
		},
		deleteFn: func(context.Context, marketplaceprivateoffersdk.DeleteAttachmentRequest) (marketplaceprivateoffersdk.DeleteAttachmentResponse, error) {
			t.Fatal("DeleteAttachment() should not run when list confirms no match")
			return marketplaceprivateoffersdk.DeleteAttachmentResponse{}, nil
		},
	}

	deleted, err := testAttachmentClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after list no-match confirmation")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListAttachments() calls = %d, want 1", len(fake.listRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestAttachmentDeleteWithoutTrackedIdentityOrOfferReleasesFinalizer(t *testing.T) {
	t.Parallel()

	resource := newAttachmentTestResource()
	resource.Annotations = nil
	fake := &fakeAttachmentOCIClient{}

	deleted, err := testAttachmentClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want untracked resource released")
	}
	if len(fake.listRequests) != 0 || len(fake.deleteRequests) != 0 {
		t.Fatalf("OCI calls = list %d delete %d, want none", len(fake.listRequests), len(fake.deleteRequests))
	}
}

func TestAttachmentCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	fake := &fakeAttachmentOCIClient{
		createFn: func(context.Context, marketplaceprivateoffersdk.CreateAttachmentRequest) (marketplaceprivateoffersdk.CreateAttachmentResponse, error) {
			return marketplaceprivateoffersdk.CreateAttachmentResponse{}, errortest.NewServiceError(
				500,
				"InternalError",
				"create failed",
			)
		},
	}

	resource := newAttachmentTestResource()
	response, err := testAttachmentClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestApplyAttachmentRuntimeHooksInstallsReviewedBehavior(t *testing.T) {
	t.Parallel()

	hooks := newAttachmentDefaultRuntimeHooks(marketplaceprivateoffersdk.AttachmentClient{})
	applyAttachmentRuntimeHooks(&hooks)
	requireAttachmentRuntimeHooks(t, hooks)

	resource := newAttachmentTestResource()
	body, err := hooks.BuildCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("hooks.BuildCreateBody() error = %v", err)
	}
	requireAttachmentCreateDetails(t, body)
}

func requireAttachmentRuntimeHooks(t *testing.T, hooks AttachmentRuntimeHooks) {
	t.Helper()
	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("hooks.Semantics.FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create builder")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("hooks.ParityHooks.ValidateCreateOnlyDrift = nil, want no-update drift guard")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handler")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("hooks.WrapGeneratedClient is empty, want lifecycle wrapper")
	}
}

func requireAttachmentCreateDetails(t *testing.T, body any) {
	t.Helper()
	details, ok := body.(marketplaceprivateoffersdk.CreateAttachmentDetails)
	if !ok {
		t.Fatalf("hooks.BuildCreateBody() body type = %T, want marketplaceprivateoffer.CreateAttachmentDetails", body)
	}
	requireAttachmentStringPtr(t, "create body displayName", details.DisplayName, testAttachmentDisplayName)
	if got, want := string(details.FileBase64Encoded), testAttachmentFileContent; got != want {
		t.Fatalf("create body file bytes = %q, want %q", got, want)
	}
	if details.Type != marketplaceprivateoffersdk.AttachmentTypeContractTAndC {
		t.Fatalf("create body type = %q, want %q", details.Type, marketplaceprivateoffersdk.AttachmentTypeContractTAndC)
	}
}

func requireAttachmentStatus(
	t *testing.T,
	resource *marketplaceprivateofferv1beta1.Attachment,
	wantID string,
	wantOfferID string,
	wantDisplayName string,
	wantType string,
	wantLifecycleState string,
) {
	t.Helper()
	if got := resource.Status.Id; got != wantID {
		t.Fatalf("status.id = %q, want %q", got, wantID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if got := resource.Status.OfferId; got != wantOfferID {
		t.Fatalf("status.offerId = %q, want %q", got, wantOfferID)
	}
	if got := resource.Status.DisplayName; got != wantDisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, wantDisplayName)
	}
	if got := resource.Status.Type; got != wantType {
		t.Fatalf("status.type = %q, want %q", got, wantType)
	}
	if got := resource.Status.LifecycleState; got != wantLifecycleState {
		t.Fatalf("status.lifecycleState = %q, want %q", got, wantLifecycleState)
	}
}

func requireAttachmentFileFingerprint(
	t *testing.T,
	resource *marketplaceprivateofferv1beta1.Attachment,
	fileBase64Encoded string,
) {
	t.Helper()
	got, ok := attachmentRecordedFileFingerprint(resource)
	if !ok {
		t.Fatalf("file fingerprint missing from status.status.message %q", resource.Status.OsokStatus.Message)
	}
	want, err := attachmentFileFingerprint(fileBase64Encoded)
	if err != nil {
		t.Fatalf("attachmentFileFingerprint() error = %v", err)
	}
	if got != want {
		t.Fatalf("file fingerprint = %q, want %q", got, want)
	}
}

func requireAttachmentStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}
