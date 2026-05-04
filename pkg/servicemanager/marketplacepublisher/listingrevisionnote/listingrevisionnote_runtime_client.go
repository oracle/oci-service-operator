/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevisionnote

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerListingRevisionNoteRuntimeHooksMutator(func(_ *ListingRevisionNoteServiceManager, hooks *ListingRevisionNoteRuntimeHooks) {
		applyListingRevisionNoteRuntimeHooks(hooks)
	})
}

func applyListingRevisionNoteRuntimeHooks(hooks *ListingRevisionNoteRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newListingRevisionNoteRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *marketplacepublisherv1beta1.ListingRevisionNote, _ string) (any, error) {
		return buildListingRevisionNoteCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *marketplacepublisherv1beta1.ListingRevisionNote, _ string, currentResponse any) (any, bool, error) {
		return buildListingRevisionNoteUpdateBody(resource, currentResponse)
	}
	hooks.List.Fields = listingRevisionNoteListFields()
	if hooks.List.Call != nil {
		listCall := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request marketplacepublishersdk.ListListingRevisionNotesRequest) (marketplacepublishersdk.ListListingRevisionNotesResponse, error) {
			response, err := listListingRevisionNotesAllPages(ctx, request, listCall)
			if err != nil {
				return response, err
			}
			response.Items = publisherListingRevisionNoteSummaries(response.Items)
			return response, nil
		}
	}
	hooks.DeleteHooks.HandleError = rejectListingRevisionNoteAuthShapedNotFound
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapListingRevisionNotePreDeleteAuthGuard(hooks.Get.Call))
}

func newListingRevisionNoteRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "marketplacepublisher",
		FormalSlug:        "listingrevisionnote",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(marketplacepublishersdk.ListingRevisionNoteLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "best-effort",
			TerminalStates: []string{
				string(marketplacepublishersdk.ListingRevisionNoteLifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"listingRevisionId",
				"noteDetails",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"freeformTags",
			},
			ForceNew: []string{
				"listingRevisionId",
				"noteDetails",
			},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ListingRevisionNote", Action: "CreateListingRevisionNote"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ListingRevisionNote", Action: "UpdateListingRevisionNote"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ListingRevisionNote", Action: "DeleteListingRevisionNote"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ListingRevisionNote", Action: "GetListingRevisionNote"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ListingRevisionNote", Action: "GetListingRevisionNote"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ListingRevisionNote", Action: "GetListingRevisionNote"}},
		},
	}
}

func listingRevisionNoteListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ListingRevisionId", RequestName: "listingRevisionId", Contribution: "query"},
	}
}

func buildListingRevisionNoteCreateBody(resource *marketplacepublisherv1beta1.ListingRevisionNote) (marketplacepublishersdk.CreateListingRevisionNoteDetails, error) {
	if resource == nil {
		return marketplacepublishersdk.CreateListingRevisionNoteDetails{}, fmt.Errorf("ListingRevisionNote resource is nil")
	}

	details := marketplacepublishersdk.CreateListingRevisionNoteDetails{
		ListingRevisionId: common.String(resource.Spec.ListingRevisionId),
		NoteDetails:       common.String(resource.Spec.NoteDetails),
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneListingRevisionNoteStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = listingRevisionNoteDefinedTags(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildListingRevisionNoteUpdateBody(
	resource *marketplacepublisherv1beta1.ListingRevisionNote,
	currentResponse any,
) (marketplacepublishersdk.UpdateListingRevisionNoteDetails, bool, error) {
	if resource == nil {
		return marketplacepublishersdk.UpdateListingRevisionNoteDetails{}, false, fmt.Errorf("ListingRevisionNote resource is nil")
	}
	current, ok := listingRevisionNoteFromResponse(currentResponse)
	if !ok {
		return marketplacepublishersdk.UpdateListingRevisionNoteDetails{}, false, fmt.Errorf("current ListingRevisionNote response does not expose a ListingRevisionNote body")
	}

	details := marketplacepublishersdk.UpdateListingRevisionNoteDetails{}
	updateNeeded := false
	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		details.FreeformTags = cloneListingRevisionNoteStringMap(resource.Spec.FreeformTags)
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := listingRevisionNoteDefinedTags(resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			details.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	return details, updateNeeded, nil
}

func listingRevisionNoteFromResponse(response any) (marketplacepublishersdk.ListingRevisionNote, bool) {
	switch typed := response.(type) {
	case marketplacepublishersdk.CreateListingRevisionNoteResponse:
		return typed.ListingRevisionNote, true
	case marketplacepublishersdk.GetListingRevisionNoteResponse:
		return typed.ListingRevisionNote, true
	case marketplacepublishersdk.UpdateListingRevisionNoteResponse:
		return typed.ListingRevisionNote, true
	case marketplacepublishersdk.ListingRevisionNote:
		return typed, true
	case marketplacepublishersdk.ListingRevisionNoteSummary:
		return listingRevisionNoteFromSummary(typed), true
	default:
		return marketplacepublishersdk.ListingRevisionNote{}, false
	}
}

func listingRevisionNoteFromSummary(summary marketplacepublishersdk.ListingRevisionNoteSummary) marketplacepublishersdk.ListingRevisionNote {
	return marketplacepublishersdk.ListingRevisionNote(summary)
}

func listListingRevisionNotesAllPages(
	ctx context.Context,
	request marketplacepublishersdk.ListListingRevisionNotesRequest,
	list func(context.Context, marketplacepublishersdk.ListListingRevisionNotesRequest) (marketplacepublishersdk.ListListingRevisionNotesResponse, error),
) (marketplacepublishersdk.ListListingRevisionNotesResponse, error) {
	var combined marketplacepublishersdk.ListListingRevisionNotesResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return combined, err
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func publisherListingRevisionNoteSummaries(items []marketplacepublishersdk.ListingRevisionNoteSummary) []marketplacepublishersdk.ListingRevisionNoteSummary {
	filtered := make([]marketplacepublishersdk.ListingRevisionNoteSummary, 0, len(items))
	for _, item := range items {
		if item.NoteSource == marketplacepublishersdk.ListingRevisionNoteNoteSourcePublisher {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func rejectListingRevisionNoteAuthShapedNotFound(resource *marketplacepublisherv1beta1.ListingRevisionNote, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return fmt.Errorf("ListingRevisionNote delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted")
}

type listingRevisionNotePreDeleteAuthGuard struct {
	delegate ListingRevisionNoteServiceClient
	get      func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error)
}

func wrapListingRevisionNotePreDeleteAuthGuard(
	get func(context.Context, marketplacepublishersdk.GetListingRevisionNoteRequest) (marketplacepublishersdk.GetListingRevisionNoteResponse, error),
) func(ListingRevisionNoteServiceClient) ListingRevisionNoteServiceClient {
	return func(delegate ListingRevisionNoteServiceClient) ListingRevisionNoteServiceClient {
		return listingRevisionNotePreDeleteAuthGuard{
			delegate: delegate,
			get:      get,
		}
	}
}

func (c listingRevisionNotePreDeleteAuthGuard) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionNote,
	request ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, request)
}

func (c listingRevisionNotePreDeleteAuthGuard) Delete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionNote,
) (bool, error) {
	if c.get == nil || resource == nil {
		return c.delegate.Delete(ctx, resource)
	}
	currentID := currentListingRevisionNoteID(resource)
	if currentID == "" {
		return c.delegate.Delete(ctx, resource)
	}

	_, err := c.get(ctx, marketplacepublishersdk.GetListingRevisionNoteRequest{
		ListingRevisionNoteId: common.String(currentID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return c.delegate.Delete(ctx, resource)
	}
	return false, rejectListingRevisionNoteAuthShapedNotFound(resource, err)
}

func currentListingRevisionNoteID(resource *marketplacepublisherv1beta1.ListingRevisionNote) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func cloneListingRevisionNoteStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func listingRevisionNoteDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}
