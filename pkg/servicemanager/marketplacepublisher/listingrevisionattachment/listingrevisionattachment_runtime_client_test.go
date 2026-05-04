/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevisionattachment

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testListingRevisionAttachmentID          = "ocid1.listingrevisionattachment.oc1..test"
	testOtherListingRevisionAttachmentID     = "ocid1.listingrevisionattachment.oc1..other"
	testListingRevisionID                    = "ocid1.listingrevision.oc1..test"
	testListingRevisionAttachmentName        = "listing-revision-attachment"
	testListingRevisionAttachmentCompartment = "ocid1.compartment.oc1..test"
)

type fakeListingRevisionAttachmentOCIClient struct {
	createFn func(context.Context, marketplacepublishersdk.CreateListingRevisionAttachmentRequest) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error)
	getFn    func(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error)
	listFn   func(context.Context, marketplacepublishersdk.ListListingRevisionAttachmentsRequest) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error)
	updateFn func(context.Context, marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error)
	deleteFn func(context.Context, marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error)
}

func (f *fakeListingRevisionAttachmentOCIClient) CreateListingRevisionAttachment(
	ctx context.Context,
	req marketplacepublishersdk.CreateListingRevisionAttachmentRequest,
) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return marketplacepublishersdk.CreateListingRevisionAttachmentResponse{}, nil
}

func (f *fakeListingRevisionAttachmentOCIClient) GetListingRevisionAttachment(
	ctx context.Context,
	req marketplacepublishersdk.GetListingRevisionAttachmentRequest,
) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return marketplacepublishersdk.GetListingRevisionAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "attachment is missing")
}

func (f *fakeListingRevisionAttachmentOCIClient) ListListingRevisionAttachments(
	ctx context.Context,
	req marketplacepublishersdk.ListListingRevisionAttachmentsRequest,
) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return marketplacepublishersdk.ListListingRevisionAttachmentsResponse{}, nil
}

func (f *fakeListingRevisionAttachmentOCIClient) UpdateListingRevisionAttachment(
	ctx context.Context,
	req marketplacepublishersdk.UpdateListingRevisionAttachmentRequest,
) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return marketplacepublishersdk.UpdateListingRevisionAttachmentResponse{}, nil
}

func (f *fakeListingRevisionAttachmentOCIClient) DeleteListingRevisionAttachment(
	ctx context.Context,
	req marketplacepublishersdk.DeleteListingRevisionAttachmentRequest,
) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{}, nil
}

func newTestListingRevisionAttachmentClient(
	client listingRevisionAttachmentOCIClient,
) ListingRevisionAttachmentServiceClient {
	return newListingRevisionAttachmentServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeListingRevisionAttachmentResource() *marketplacepublisherv1beta1.ListingRevisionAttachment {
	return &marketplacepublisherv1beta1.ListingRevisionAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testListingRevisionAttachmentName,
			Namespace: "default",
		},
		Spec: marketplacepublisherv1beta1.ListingRevisionAttachmentSpec{
			DisplayName:       testListingRevisionAttachmentName,
			ListingRevisionId: testListingRevisionID,
			AttachmentType:    string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeSupportedServices),
			ServiceName:       "implementation services",
			Type:              string(marketplacepublishersdk.SupportedServiceAttachmentTypeImplementationService),
			Url:               "https://example.com/service",
			Description:       "initial description",
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeTrackedListingRevisionAttachmentResource() *marketplacepublisherv1beta1.ListingRevisionAttachment {
	resource := makeListingRevisionAttachmentResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionAttachmentID)
	resource.Status.Id = testListingRevisionAttachmentID
	resource.Status.ListingRevisionId = testListingRevisionID
	resource.Status.DisplayName = testListingRevisionAttachmentName
	resource.Status.AttachmentType = string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeSupportedServices)
	return resource
}

func makeTrackedListingRevisionAttachmentResourceWithoutCreateIdentity() *marketplacepublisherv1beta1.ListingRevisionAttachment {
	resource := makeTrackedListingRevisionAttachmentResource()
	resource.Status.Id = ""
	resource.Spec.AttachmentType = ""
	resource.Spec.JsonData = ""
	return resource
}

func makeListingRevisionAttachmentRequest(resource *marketplacepublisherv1beta1.ListingRevisionAttachment) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKSupportedAttachment(
	id string,
	spec marketplacepublisherv1beta1.ListingRevisionAttachmentSpec,
	state marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateEnum,
) marketplacepublishersdk.SupportedServiceAttachment {
	return marketplacepublishersdk.SupportedServiceAttachment{
		Id:                common.String(id),
		CompartmentId:     common.String(testListingRevisionAttachmentCompartment),
		ListingRevisionId: common.String(spec.ListingRevisionId),
		DisplayName:       common.String(spec.DisplayName),
		ServiceName:       common.String(spec.ServiceName),
		Description:       common.String(spec.Description),
		Url:               common.String(spec.Url),
		Type:              marketplacepublishersdk.SupportedServiceAttachmentTypeEnum(spec.Type),
		LifecycleState:    state,
		FreeformTags:      cloneListingRevisionAttachmentStringMap(spec.FreeformTags),
		DefinedTags:       listingRevisionAttachmentDefinedTags(spec.DefinedTags),
	}
}

func makeSDKSupportedAttachmentSummary(
	id string,
	spec marketplacepublisherv1beta1.ListingRevisionAttachmentSpec,
	state marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateEnum,
) marketplacepublishersdk.ListingRevisionAttachmentSummary {
	return marketplacepublishersdk.ListingRevisionAttachmentSummary{
		Id:                common.String(id),
		ListingRevisionId: common.String(spec.ListingRevisionId),
		CompartmentId:     common.String(testListingRevisionAttachmentCompartment),
		DisplayName:       common.String(spec.DisplayName),
		AttachmentType:    marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeSupportedServices,
		ContentUrl:        common.String("https://example.com/content"),
		LifecycleState:    state,
		FreeformTags:      cloneListingRevisionAttachmentStringMap(spec.FreeformTags),
		DefinedTags:       listingRevisionAttachmentDefinedTags(spec.DefinedTags),
	}
}

func makeSDKVideoAttachment(
	id string,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	contentURL string,
) marketplacepublishersdk.VideoAttachment {
	return marketplacepublishersdk.VideoAttachment{
		Id:                common.String(id),
		CompartmentId:     common.String(testListingRevisionAttachmentCompartment),
		ListingRevisionId: common.String(resource.Spec.ListingRevisionId),
		DisplayName:       common.String(resource.Spec.DisplayName),
		ContentUrl:        common.String(contentURL),
		Description:       common.String(resource.Spec.Description),
		LifecycleState:    marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive,
	}
}

func TestListingRevisionAttachmentRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newListingRevisionAttachmentRuntimeHooksWithOCIClient(&fakeListingRevisionAttachmentOCIClient{})
	applyListingRevisionAttachmentRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("Semantics = nil, want reviewed resource-local semantics")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("BuildCreateBody = nil, want polymorphic create body shaping")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("BuildUpdateBody = nil, want polymorphic update body shaping")
	}
	if hooks.Identity.Resolve == nil {
		t.Fatal("Identity.Resolve = nil, want stable pre-create identity")
	}
	if hooks.DeleteHooks.HandleError == nil || hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("DeleteHooks missing conservative delete handling")
	}
}

func TestListingRevisionAttachmentCreateOrUpdateBindsExistingAttachmentByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionAttachmentResource()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		listFn: func(_ context.Context, req marketplacepublishersdk.ListListingRevisionAttachmentsRequest) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListListingRevisionAttachmentsRequest.ListingRevisionId", req.ListingRevisionId, resource.Spec.ListingRevisionId)
			requireStringPtr(t, "ListListingRevisionAttachmentsRequest.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			if listCalls == 1 {
				if req.Page != nil {
					t.Fatalf("first ListListingRevisionAttachmentsRequest.Page = %q, want nil", *req.Page)
				}
				return marketplacepublishersdk.ListListingRevisionAttachmentsResponse{
					ListingRevisionAttachmentCollection: marketplacepublishersdk.ListingRevisionAttachmentCollection{
						Items: []marketplacepublishersdk.ListingRevisionAttachmentSummary{
							makeSDKSupportedAttachmentSummary(testOtherListingRevisionAttachmentID, marketplacepublisherv1beta1.ListingRevisionAttachmentSpec{
								ListingRevisionId: resource.Spec.ListingRevisionId,
								DisplayName:       "other",
							}, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second ListListingRevisionAttachmentsRequest.Page", req.Page, "page-2")
			return marketplacepublishersdk.ListListingRevisionAttachmentsResponse{
				ListingRevisionAttachmentCollection: marketplacepublishersdk.ListingRevisionAttachmentCollection{
					Items: []marketplacepublishersdk.ListingRevisionAttachmentSummary{
						makeSDKSupportedAttachmentSummary(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			getCalls++
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateListingRevisionAttachmentRequest) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error) {
			createCalled = true
			return marketplacepublishersdk.CreateListingRevisionAttachmentResponse{}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error) {
			updateCalled = true
			return marketplacepublishersdk.UpdateListingRevisionAttachmentResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateListingRevisionAttachment() called for existing attachment")
	}
	if updateCalled {
		t.Fatal("UpdateListingRevisionAttachment() called for matching attachment")
	}
	if listCalls != 2 {
		t.Fatalf("ListListingRevisionAttachments() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetListingRevisionAttachment() calls = %d, want 1 live readback", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testListingRevisionAttachmentID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testListingRevisionAttachmentID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestListingRevisionAttachmentCreateBuildsPolymorphicBodyAndRecordsRequestID(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionAttachmentResource()
	resource.Spec.DisplayName = ""
	createCalled := false

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		createFn: func(_ context.Context, req marketplacepublishersdk.CreateListingRevisionAttachmentRequest) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error) {
			createCalled = true
			if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
				t.Fatal("CreateListingRevisionAttachmentRequest.OpcRetryToken = nil/empty, want deterministic retry token")
			}
			body, ok := req.CreateListingRevisionAttachmentDetails.(marketplacepublishersdk.CreateSupportedServiceAttachment)
			if !ok {
				t.Fatalf("CreateListingRevisionAttachmentDetails type = %T, want CreateSupportedServiceAttachment", req.CreateListingRevisionAttachmentDetails)
			}
			requireStringPtr(t, "CreateSupportedServiceAttachment.ListingRevisionId", body.ListingRevisionId, resource.Spec.ListingRevisionId)
			requireStringPtr(t, "CreateSupportedServiceAttachment.DisplayName", body.DisplayName, resource.Name)
			requireStringPtr(t, "CreateSupportedServiceAttachment.ServiceName", body.ServiceName, resource.Spec.ServiceName)
			if body.Type != marketplacepublishersdk.SupportedServiceAttachmentTypeImplementationService {
				t.Fatalf("CreateSupportedServiceAttachment.Type = %q, want IMPLEMENTATION_SERVICE", body.Type)
			}
			createdSpec := resource.Spec
			createdSpec.DisplayName = resource.Name
			return marketplacepublishersdk.CreateListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, createdSpec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
				OpcRequestId:              common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			createdSpec := resource.Spec
			createdSpec.DisplayName = resource.Name
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, createdSpec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !createCalled {
		t.Fatal("CreateListingRevisionAttachment() was not called")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testListingRevisionAttachmentID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testListingRevisionAttachmentID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestListingRevisionAttachmentCreateSupportsJsonDataVideoBody(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionAttachmentResource()
	resource.Spec.AttachmentType = ""
	resource.Spec.DisplayName = ""
	resource.Spec.ServiceName = ""
	resource.Spec.Type = ""
	resource.Spec.JsonData = `{
		"attachmentType":"VIDEO",
		"displayName":"json-video",
		"description":"video attachment",
		"videoAttachmentDetails":{"contentUrl":"https://example.com/video"}
	}`

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		createFn: func(_ context.Context, req marketplacepublishersdk.CreateListingRevisionAttachmentRequest) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error) {
			body, ok := req.CreateListingRevisionAttachmentDetails.(marketplacepublishersdk.CreateVideoAttachmentDetails)
			if !ok {
				t.Fatalf("CreateListingRevisionAttachmentDetails type = %T, want CreateVideoAttachmentDetails", req.CreateListingRevisionAttachmentDetails)
			}
			requireStringPtr(t, "CreateVideoAttachmentDetails.ListingRevisionId", body.ListingRevisionId, resource.Spec.ListingRevisionId)
			requireStringPtr(t, "CreateVideoAttachmentDetails.DisplayName", body.DisplayName, "json-video")
			if body.VideoAttachmentDetails == nil {
				t.Fatal("CreateVideoAttachmentDetails.VideoAttachmentDetails = nil, want content URL")
			}
			requireStringPtr(t, "CreateVideoDetails.ContentUrl", body.VideoAttachmentDetails.ContentUrl, "https://example.com/video")
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal create body: %v", err)
			}
			if strings.Contains(string(encoded), "jsonData") {
				t.Fatalf("create body leaked jsonData helper field: %s", string(encoded))
			}
			return marketplacepublishersdk.CreateListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKVideoAttachment(testListingRevisionAttachmentID, resource, "https://example.com/video"),
			}, nil
		},
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKVideoAttachment(testListingRevisionAttachmentID, resource, "https://example.com/video"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if got := resource.Status.AttachmentType; got != string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeVideo) {
		t.Fatalf("status.attachmentType = %q, want VIDEO", got)
	}
}

func TestListingRevisionAttachmentNoopReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()
	updateCalled := false

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error) {
			updateCalled = true
			return marketplacepublishersdk.UpdateListingRevisionAttachmentResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateListingRevisionAttachment() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestListingRevisionAttachmentMutableUpdateUsesTypedBody(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()
	currentSpec := resource.Spec
	currentSpec.Description = "old description"
	currentSpec.ServiceName = "old service"
	currentSpec.FreeformTags = map[string]string{"env": "old"}
	resource.Spec.Description = "new description"
	resource.Spec.ServiceName = "new service"
	resource.Spec.FreeformTags = map[string]string{"env": "new"}
	updateCalled := false

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			if updateCalled {
				return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
					ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, currentSpec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error) {
			updateCalled = true
			requireStringPtr(t, "UpdateListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			body, ok := req.UpdateListingRevisionAttachmentDetails.(marketplacepublishersdk.UpdateSupportedServiceAttachment)
			if !ok {
				t.Fatalf("UpdateListingRevisionAttachmentDetails type = %T, want UpdateSupportedServiceAttachment", req.UpdateListingRevisionAttachmentDetails)
			}
			requireStringPtr(t, "UpdateSupportedServiceAttachment.Description", body.Description, resource.Spec.Description)
			requireStringPtr(t, "UpdateSupportedServiceAttachment.ServiceName", body.ServiceName, resource.Spec.ServiceName)
			if got := body.FreeformTags["env"]; got != "new" {
				t.Fatalf("UpdateSupportedServiceAttachment.FreeformTags[env] = %q, want new", got)
			}
			return marketplacepublishersdk.UpdateListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
				OpcRequestId:              common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !updateCalled {
		t.Fatal("UpdateListingRevisionAttachment() was not called for mutable drift")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestListingRevisionAttachmentRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()
	currentSpec := resource.Spec
	resource.Spec.ListingRevisionId = "ocid1.listingrevision.oc1..changed"
	updateCalled := false

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, currentSpec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error) {
			updateCalled = true
			return marketplacepublishersdk.UpdateListingRevisionAttachmentResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateListingRevisionAttachment() called after create-only drift")
	}
	if !strings.Contains(err.Error(), "listingRevisionId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want listingRevisionId drift detail", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestListingRevisionAttachmentRejectsVideoContentURLDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()
	resource.Spec.AttachmentType = string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeVideo)
	resource.Spec.VideoAttachmentDetails.ContentUrl = "https://example.com/new-video"
	resource.Status.AttachmentType = string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeVideo)
	updateCalled := false

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKVideoAttachment(testListingRevisionAttachmentID, resource, "https://example.com/old-video"),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error) {
			updateCalled = true
			return marketplacepublishersdk.UpdateListingRevisionAttachmentResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want video content URL drift rejection")
	}
	if updateCalled {
		t.Fatal("UpdateListingRevisionAttachment() called after video content URL drift")
	}
	if !strings.Contains(err.Error(), "videoAttachmentDetails.contentUrl changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want video content URL drift detail", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestListingRevisionAttachmentDeleteRetainsFinalizerUntilConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			getCalls++
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			if getCalls <= 3 {
				return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
					ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "attachment is gone")
		},
		deleteFn: func(_ context.Context, req marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteListingRevisionAttachment() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireDeletePendingStatus(t, resource)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteListingRevisionAttachment() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestListingRevisionAttachmentDeleteUsesTrackedIDWhenCreateIdentityIsCleared(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResourceWithoutCreateIdentity()
	getCalls := 0
	deleteCalls := 0

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			getCalls++
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			if getCalls <= 3 {
				return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
					ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "attachment is gone")
		},
		deleteFn: func(_ context.Context, req marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteListingRevisionAttachment() calls after first delete = %d, want 1", deleteCalls)
	}
	requireDeletePendingStatus(t, resource)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteListingRevisionAttachment() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestListingRevisionAttachmentDeleteKeepsAuthShapedNotFoundConservativeWhenCreateIdentityIsCleared(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResourceWithoutCreateIdentity()
	deleteCalls := 0

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteListingRevisionAttachment() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for ambiguous NotAuthorizedOrNotFound")
	}
}

func TestListingRevisionAttachmentDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
				ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
			return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for ambiguous NotAuthorizedOrNotFound")
	}
}

func TestListingRevisionAttachmentDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()
	getCalls := 0
	deleteCalled := false

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(_ context.Context, req marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			getCalls++
			requireStringPtr(t, "GetListingRevisionAttachmentRequest.ListingRevisionAttachmentId", req.ListingRevisionAttachmentId, testListingRevisionAttachmentID)
			if getCalls <= 2 {
				return marketplacepublishersdk.GetListingRevisionAttachmentResponse{
					ListingRevisionAttachment: makeSDKSupportedAttachment(testListingRevisionAttachmentID, resource.Spec, marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
				}, nil
			}
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization is ambiguous")
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
			deleteCalled = true
			return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped post-delete confirm read")
	}
	if !deleteCalled {
		t.Fatal("DeleteListingRevisionAttachment() was not called before post-delete confirm read")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for post-delete NotAuthorizedOrNotFound")
	}
}

func TestListingRevisionAttachmentDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListingRevisionAttachmentResource()
	deleteCalled := false

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization is ambiguous")
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
			deleteCalled = true
			return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteListingRevisionAttachment() called after auth-shaped pre-delete confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestListingRevisionAttachmentCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionAttachmentResource()

	client := newTestListingRevisionAttachmentClient(&fakeListingRevisionAttachmentOCIClient{
		createFn: func(context.Context, marketplacepublishersdk.CreateListingRevisionAttachmentRequest) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error) {
			return marketplacepublishersdk.CreateListingRevisionAttachmentResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeListingRevisionAttachmentRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireLastCondition(
	t *testing.T,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want last condition %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition type = %q, want %q", got, want)
	}
}

func requireDeletePendingStatus(
	t *testing.T,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending operation")
	}
	if current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("async phase = %q, want delete", current.Phase)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async normalizedClass = %q, want pending", current.NormalizedClass)
	}
	requireLastCondition(t, resource, shared.Terminating)
}
