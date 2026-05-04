/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

const listingRevisionKind = "ListingRevision"

type listingRevisionOCIClient interface {
	CreateListingRevision(context.Context, marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error)
	GetListingRevision(context.Context, marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error)
	ListListingRevisions(context.Context, marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error)
	UpdateListingRevision(context.Context, marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error)
	DeleteListingRevision(context.Context, marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error)
}

type createListingRevisionRequest struct {
	CreateListingRevisionDetails json.RawMessage `contributesTo:"body"`
	OpcRetryToken                *string         `mandatory:"false" contributesTo:"header" name:"opc-retry-token"`
	OpcRequestId                 *string         `mandatory:"false" contributesTo:"header" name:"opc-request-id"`
	RequestMetadata              common.RequestMetadata
}

type updateListingRevisionRequest struct {
	ListingRevisionId            *string         `mandatory:"true" contributesTo:"path" name:"listingRevisionId"`
	UpdateListingRevisionDetails json.RawMessage `contributesTo:"body"`
	IfMatch                      *string         `mandatory:"false" contributesTo:"header" name:"if-match"`
	OpcRequestId                 *string         `mandatory:"false" contributesTo:"header" name:"opc-request-id"`
	RequestMetadata              common.RequestMetadata
}

type listingRevisionOperationResponse struct {
	ListingRevision map[string]any `presentIn:"body"`
	Etag            *string        `presentIn:"header" name:"etag"`
	OpcRequestId    *string        `presentIn:"header" name:"opc-request-id"`
}

type listingRevisionListResponse struct {
	ListingRevisionCollection listingRevisionListBody `presentIn:"body"`
	OpcRequestId              *string                 `presentIn:"header" name:"opc-request-id"`
	OpcNextPage               *string                 `presentIn:"header" name:"opc-next-page"`
}

type listingRevisionListBody struct {
	Items []map[string]any `json:"items"`
}

type listingRevisionIdentity struct {
	listingID   string
	displayName string
	listingType string
}

type listingRevisionAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e listingRevisionAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e listingRevisionAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerListingRevisionRuntimeHooksMutator(func(manager *ListingRevisionServiceManager, hooks *ListingRevisionRuntimeHooks) {
		client, initErr := newListingRevisionSDKClient(manager)
		applyListingRevisionRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newListingRevisionSDKClient(manager *ListingRevisionServiceManager) (listingRevisionOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", listingRevisionKind)
	}
	client, err := marketplacepublishersdk.NewMarketplacePublisherClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyListingRevisionRuntimeHooks(
	manager *ListingRevisionServiceManager,
	hooks *ListingRevisionRuntimeHooks,
	client listingRevisionOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	applyListingRevisionCoreHooks(hooks)
	applyListingRevisionOperationHooks(hooks, client, initErr)
	applyListingRevisionDeleteHooks(hooks)
	applyListingRevisionGeneratedClientWrapper(manager, hooks)
}

func applyListingRevisionCoreHooks(hooks *ListingRevisionRuntimeHooks) {
	hooks.Semantics = newListingRevisionRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *marketplacepublisherv1beta1.ListingRevision, _ string) (any, error) {
		return buildListingRevisionCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *marketplacepublisherv1beta1.ListingRevision,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildListingRevisionUpdateBody(resource, currentResponse)
	}
	hooks.Identity.Resolve = func(resource *marketplacepublisherv1beta1.ListingRevision) (any, error) {
		return resolveListingRevisionIdentity(resource)
	}
	hooks.Identity.RecordPath = recordListingRevisionPathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardListingRevisionExistingBeforeCreate
	hooks.List.Fields = listingRevisionListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedListingRevisionIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateListingRevisionCreateOnlyDriftForResponse
}

func applyListingRevisionOperationHooks(
	hooks *ListingRevisionRuntimeHooks,
	client listingRevisionOCIClient,
	initErr error,
) {
	hooks.Get.Call = listingRevisionGetCall(client, initErr)
	hooks.List.Call = func(ctx context.Context, request marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
		return listListingRevisionsAllPages(ctx, client, initErr, request)
	}
	hooks.Create.Call = listingRevisionCreateCall(client, initErr)
	hooks.Update.Call = listingRevisionUpdateCall(client, initErr)
	hooks.Delete.Call = listingRevisionDeleteCall(client, initErr)
}

func applyListingRevisionDeleteHooks(hooks *ListingRevisionRuntimeHooks) {
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *marketplacepublisherv1beta1.ListingRevision, currentID string) (any, error) {
		return confirmListingRevisionDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleListingRevisionDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyListingRevisionDeleteOutcome
}

func applyListingRevisionGeneratedClientWrapper(manager *ListingRevisionServiceManager, hooks *ListingRevisionRuntimeHooks) {
	if manager == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(ListingRevisionServiceClient) ListingRevisionServiceClient {
		return defaultListingRevisionServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.ListingRevision](
				buildListingRevisionPolymorphicGeneratedRuntimeConfig(manager, *hooks),
			),
		}
	})
}

func listingRevisionGetCall(
	client listingRevisionOCIClient,
	initErr error,
) func(context.Context, marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
	return func(ctx context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
		if err := validateListingRevisionOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.GetListingRevisionResponse{}, err
		}
		response, err := client.GetListingRevision(ctx, request)
		return response, conservativeListingRevisionNotFoundError(err, "read")
	}
}

func listingRevisionCreateCall(
	client listingRevisionOCIClient,
	initErr error,
) func(context.Context, marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
	return func(ctx context.Context, request marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
		if err := validateListingRevisionOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.CreateListingRevisionResponse{}, err
		}
		return client.CreateListingRevision(ctx, request)
	}
}

func listingRevisionUpdateCall(
	client listingRevisionOCIClient,
	initErr error,
) func(context.Context, marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
	return func(ctx context.Context, request marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
		if err := validateListingRevisionOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.UpdateListingRevisionResponse{}, err
		}
		return client.UpdateListingRevision(ctx, request)
	}
}

func listingRevisionDeleteCall(
	client listingRevisionOCIClient,
	initErr error,
) func(context.Context, marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
	return func(ctx context.Context, request marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
		if err := validateListingRevisionOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.DeleteListingRevisionResponse{}, err
		}
		response, err := client.DeleteListingRevision(ctx, request)
		return response, conservativeListingRevisionNotFoundError(err, "delete")
	}
}

func validateListingRevisionOCIClient(client listingRevisionOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize %s OCI client: %w", listingRevisionKind, initErr)
	}
	if client == nil {
		return fmt.Errorf("%s OCI client is not configured", listingRevisionKind)
	}
	return nil
}

func buildListingRevisionPolymorphicGeneratedRuntimeConfig(
	manager *ListingRevisionServiceManager,
	hooks ListingRevisionRuntimeHooks,
) generatedruntime.Config[*marketplacepublisherv1beta1.ListingRevision] {
	return generatedruntime.Config[*marketplacepublisherv1beta1.ListingRevision]{
		Kind:            listingRevisionKind,
		SDKName:         listingRevisionKind,
		Log:             manager.Log,
		Semantics:       hooks.Semantics,
		Identity:        hooks.Identity,
		Read:            hooks.Read,
		TrackedRecreate: hooks.TrackedRecreate,
		StatusHooks:     hooks.StatusHooks,
		ParityHooks:     hooks.ParityHooks,
		Async:           hooks.Async,
		DeleteHooks:     hooks.DeleteHooks,
		BuildCreateBody: hooks.BuildCreateBody,
		BuildUpdateBody: hooks.BuildUpdateBody,
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &createListingRevisionRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Create.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				typed := request.(*createListingRevisionRequest)
				details, err := listingRevisionCreateDetailsFromRaw(typed.CreateListingRevisionDetails)
				if err != nil {
					return nil, err
				}
				response, err := hooks.Create.Call(ctx, marketplacepublishersdk.CreateListingRevisionRequest{
					CreateListingRevisionDetails: details,
					OpcRetryToken:                typed.OpcRetryToken,
					OpcRequestId:                 typed.OpcRequestId,
					RequestMetadata:              typed.RequestMetadata,
				})
				if err != nil {
					return nil, err
				}
				return adaptListingRevisionOperationResponse(response.ListingRevision, response.OpcRequestId, response.Etag), nil
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacepublishersdk.GetListingRevisionRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Get.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*marketplacepublishersdk.GetListingRevisionRequest))
				if err != nil {
					return nil, err
				}
				return adaptListingRevisionOperationResponse(response.ListingRevision, response.OpcRequestId, response.Etag), nil
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacepublishersdk.ListListingRevisionsRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.List.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*marketplacepublishersdk.ListListingRevisionsRequest))
				if err != nil {
					return nil, err
				}
				return adaptListingRevisionListResponse(response), nil
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &updateListingRevisionRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Update.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				typed := request.(*updateListingRevisionRequest)
				details, err := listingRevisionUpdateDetailsFromRaw(typed.UpdateListingRevisionDetails)
				if err != nil {
					return nil, err
				}
				response, err := hooks.Update.Call(ctx, marketplacepublishersdk.UpdateListingRevisionRequest{
					ListingRevisionId:            typed.ListingRevisionId,
					UpdateListingRevisionDetails: details,
					IfMatch:                      typed.IfMatch,
					OpcRequestId:                 typed.OpcRequestId,
					RequestMetadata:              typed.RequestMetadata,
				})
				if err != nil {
					return nil, err
				}
				return adaptListingRevisionOperationResponse(response.ListingRevision, response.OpcRequestId, response.Etag), nil
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacepublishersdk.DeleteListingRevisionRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Delete.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				return hooks.Delete.Call(ctx, *request.(*marketplacepublishersdk.DeleteListingRevisionRequest))
			},
		},
	}
}

func newListingRevisionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client listingRevisionOCIClient,
) ListingRevisionServiceClient {
	manager := &ListingRevisionServiceManager{Log: log}
	hooks := newListingRevisionRuntimeHooksWithOCIClient(client)
	applyListingRevisionRuntimeHooks(manager, &hooks, client, nil)
	return wrapListingRevisionGeneratedClient(
		hooks,
		defaultListingRevisionServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.ListingRevision](
				buildListingRevisionPolymorphicGeneratedRuntimeConfig(manager, hooks),
			),
		},
	)
}

func newListingRevisionRuntimeHooksWithOCIClient(client listingRevisionOCIClient) ListingRevisionRuntimeHooks {
	return ListingRevisionRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*marketplacepublisherv1beta1.ListingRevision]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*marketplacepublisherv1beta1.ListingRevision]{},
		StatusHooks:     generatedruntime.StatusHooks[*marketplacepublisherv1beta1.ListingRevision]{},
		ParityHooks:     generatedruntime.ParityHooks[*marketplacepublisherv1beta1.ListingRevision]{},
		Async:           generatedruntime.AsyncHooks[*marketplacepublisherv1beta1.ListingRevision]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*marketplacepublisherv1beta1.ListingRevision]{},
		Create: runtimeOperationHooks[marketplacepublishersdk.CreateListingRevisionRequest, marketplacepublishersdk.CreateListingRevisionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateListingRevisionDetails", RequestName: "CreateListingRevisionDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
				return client.CreateListingRevision(ctx, request)
			},
		},
		Get: runtimeOperationHooks[marketplacepublishersdk.GetListingRevisionRequest, marketplacepublishersdk.GetListingRevisionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ListingRevisionId", RequestName: "listingRevisionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
				return client.GetListingRevision(ctx, request)
			},
		},
		List: runtimeOperationHooks[marketplacepublishersdk.ListListingRevisionsRequest, marketplacepublishersdk.ListListingRevisionsResponse]{
			Fields: listingRevisionListFields(),
			Call: func(ctx context.Context, request marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
				return client.ListListingRevisions(ctx, request)
			},
		},
		Update: runtimeOperationHooks[marketplacepublishersdk.UpdateListingRevisionRequest, marketplacepublishersdk.UpdateListingRevisionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ListingRevisionId", RequestName: "listingRevisionId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateListingRevisionDetails", RequestName: "UpdateListingRevisionDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
				return client.UpdateListingRevision(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[marketplacepublishersdk.DeleteListingRevisionRequest, marketplacepublishersdk.DeleteListingRevisionResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ListingRevisionId", RequestName: "listingRevisionId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
				return client.DeleteListingRevision(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ListingRevisionServiceClient) ListingRevisionServiceClient{},
	}
}

func newListingRevisionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplacepublisher",
		FormalSlug:    "listingrevision",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(marketplacepublishersdk.ListingRevisionLifecycleStateCreating)},
			UpdatingStates:     []string{string(marketplacepublishersdk.ListingRevisionLifecycleStateUpdating)},
			ActiveStates:       []string{string(marketplacepublishersdk.ListingRevisionLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(marketplacepublishersdk.ListingRevisionLifecycleStateDeleting)},
			TerminalStates: []string{string(marketplacepublishersdk.ListingRevisionLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"listingId", "displayName", "listingType", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       listingRevisionMutableFields(),
			ForceNew:      []string{"listingId", "listingType"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func listingRevisionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ListingId", RequestName: "listingId", Contribution: "query", LookupPaths: []string{"status.listingId", "spec.listingId", "listingId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", LookupPaths: []string{"status.lifecycleState", "lifecycleState"}},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "compartmentId"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func listingRevisionMutableFields() []string {
	return []string{
		"jsonData",
		"displayName",
		"headline",
		"tagline",
		"keywords",
		"shortDescription",
		"usageInformation",
		"longDescription",
		"contentLanguage",
		"supportedlanguages",
		"supportContacts",
		"supportLinks",
		"status",
		"freeformTags",
		"definedTags",
		"products",
		"versionDetails",
		"systemRequirements",
		"pricingPlans",
		"vanityUrl",
		"recommendedServiceProviderListingIds",
		"availabilityAndPricingPolicy",
		"isRoverExportable",
		"pricingType",
		"productCodes",
		"industries",
		"contactUs",
		"trainedProfessionals",
		"geoLocations",
		"demoUrl",
		"selfPacedTrainingUrl",
		"downloadInfo",
	}
}

func buildListingRevisionCreateBody(resource *marketplacepublisherv1beta1.ListingRevision) (map[string]any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", listingRevisionKind)
	}
	payload, err := listingRevisionPayloadFromSpec(resource.Spec, "")
	if err != nil {
		return nil, err
	}
	if err := validateListingRevisionCreatePayload(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func buildListingRevisionUpdateBody(
	resource *marketplacepublisherv1beta1.ListingRevision,
	currentResponse any,
) (map[string]any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", listingRevisionKind)
	}
	current, ok := listingRevisionMapFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a listing revision body", listingRevisionKind)
	}
	if err := validateListingRevisionCreateOnlyDrift(resource.Spec, current); err != nil {
		return nil, false, err
	}
	currentType := strings.TrimSpace(listingRevisionStringValue(current["listingType"]))
	payload, err := listingRevisionPayloadFromSpec(resource.Spec, currentType)
	if err != nil {
		return nil, false, err
	}
	if err := validateListingRevisionUpdatePayload(payload); err != nil {
		return nil, false, err
	}

	updatePayload, updateNeeded := buildListingRevisionUpdatePayload(payload, current)
	if !updateNeeded {
		return nil, false, nil
	}
	return updatePayload, true, nil
}

func buildListingRevisionUpdatePayload(payload map[string]any, current map[string]any) (map[string]any, bool) {
	updatePayload := map[string]any{"listingType": payload["listingType"]}
	updateNeeded := false
	for _, field := range listingRevisionUpdatePayloadFieldsForType(listingRevisionStringValue(payload["listingType"])) {
		desired, ok := lookupListingRevisionMapValue(payload, field)
		if !ok {
			continue
		}
		currentValue, currentOK := lookupListingRevisionMapValue(current, field)
		if !currentOK && listingRevisionMissingCurrentValueMatchesDefault(field, desired) {
			continue
		}
		updatePayload[field] = desired
		if !currentOK || !listingRevisionValuesEqualForField(field, desired, currentValue) {
			updateNeeded = true
		}
	}
	return updatePayload, updateNeeded
}

func listingRevisionMissingCurrentValueMatchesDefault(field string, desired any) bool {
	if field != "isRoverExportable" {
		return false
	}
	value, ok := desired.(bool)
	return ok && !value
}

func listingRevisionUpdatePayloadFieldsForType(listingType string) []string {
	fields := listingRevisionCommonUpdatePayloadFields()
	switch listingType {
	case string(marketplacepublishersdk.ListingTypeService):
		return append(fields, listingRevisionServiceUpdatePayloadFields()...)
	case string(marketplacepublishersdk.ListingTypeOciApplication):
		return append(fields, listingRevisionOciUpdatePayloadFields()...)
	case string(marketplacepublishersdk.ListingTypeLeadGeneration):
		return append(fields, listingRevisionLeadGenUpdatePayloadFields()...)
	default:
		return fields
	}
}

func listingRevisionCommonUpdatePayloadFields() []string {
	return []string{
		"displayName",
		"headline",
		"tagline",
		"keywords",
		"shortDescription",
		"usageInformation",
		"longDescription",
		"contentLanguage",
		"supportedlanguages",
		"supportContacts",
		"supportLinks",
		"freeformTags",
		"definedTags",
	}
}

func listingRevisionServiceUpdatePayloadFields() []string {
	return []string{
		"contactUs",
		"productCodes",
		"industries",
		"trainedProfessionals",
		"vanityUrl",
		"geoLocations",
	}
}

func listingRevisionOciUpdatePayloadFields() []string {
	return []string{
		"products",
		"versionDetails",
		"systemRequirements",
		"pricingPlans",
		"vanityUrl",
		"recommendedServiceProviderListingIds",
		"availabilityAndPricingPolicy",
		"isRoverExportable",
		"pricingType",
	}
}

func listingRevisionLeadGenUpdatePayloadFields() []string {
	return []string{
		"products",
		"versionDetails",
		"systemRequirements",
		"pricingPlans",
		"vanityUrl",
		"recommendedServiceProviderListingIds",
		"pricingType",
		"demoUrl",
		"selfPacedTrainingUrl",
		"downloadInfo",
	}
}

func listingRevisionPayloadFromSpec(
	spec marketplacepublisherv1beta1.ListingRevisionSpec,
	listingTypeFallback string,
) (map[string]any, error) {
	payload := map[string]any{}
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		decoded, err := listingRevisionRawMap(raw)
		if err != nil {
			return nil, err
		}
		payload = decoded
	}

	specValues, err := listingRevisionSpecMap(spec)
	if err != nil {
		return nil, err
	}
	if err := validateListingRevisionIdentityOverrides(specValues, payload); err != nil {
		return nil, err
	}
	mergeListingRevisionDefaults(payload, specValues)
	if _, ok := lookupListingRevisionMapValue(payload, "listingType"); !ok && strings.TrimSpace(listingTypeFallback) != "" {
		payload["listingType"] = strings.TrimSpace(listingTypeFallback)
	}
	if err := normalizeListingRevisionBody(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func listingRevisionSpecMap(spec marketplacepublisherv1beta1.ListingRevisionSpec) (map[string]any, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal %s spec: %w", listingRevisionKind, err)
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode %s spec: %w", listingRevisionKind, err)
	}
	delete(values, "jsonData")
	pruned, ok := pruneListingRevisionSpecValue(values)
	if !ok {
		return map[string]any{}, nil
	}
	typed, ok := pruned.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode %s spec produced %T, want object", listingRevisionKind, pruned)
	}
	overlayListingRevisionBoolSpecFields(typed, spec)
	return typed, nil
}

func overlayListingRevisionBoolSpecFields(values map[string]any, spec marketplacepublisherv1beta1.ListingRevisionSpec) {
	values["isRoverExportable"] = spec.IsRoverExportable
}

func listingRevisionRawMap(raw string) (map[string]any, error) {
	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode %s jsonData: %w", listingRevisionKind, err)
	}
	if values == nil {
		return map[string]any{}, nil
	}
	return values, nil
}

func validateListingRevisionIdentityOverrides(specValues map[string]any, rawValues map[string]any) error {
	if len(rawValues) == 0 {
		return nil
	}
	var conflicts []string
	for _, field := range []string{"listingId", "listingType"} {
		specValue, specOK := lookupListingRevisionMapValue(specValues, field)
		rawValue, rawOK := lookupListingRevisionMapValue(rawValues, field)
		if specOK && rawOK && !listingRevisionValuesEqualForField(field, specValue, rawValue) {
			conflicts = append(conflicts, field)
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("%s jsonData identity conflicts with spec field(s): %s", listingRevisionKind, strings.Join(conflicts, ", "))
}

func mergeListingRevisionDefaults(payload map[string]any, defaults map[string]any) {
	for key, value := range defaults {
		if _, exists := lookupListingRevisionMapValue(payload, key); exists {
			continue
		}
		payload[key] = value
	}
}

func pruneListingRevisionSpecValue(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		pruned := make(map[string]any, len(typed))
		for key, item := range typed {
			value, ok := pruneListingRevisionSpecValue(item)
			if !ok {
				continue
			}
			pruned[key] = value
		}
		return pruned, len(pruned) > 0
	case []any:
		pruned := make([]any, 0, len(typed))
		for _, item := range typed {
			value, ok := pruneListingRevisionSpecValue(item)
			if !ok {
				continue
			}
			pruned = append(pruned, value)
		}
		return pruned, len(pruned) > 0
	case string:
		value := strings.TrimSpace(typed)
		return value, value != ""
	case float64:
		return typed, true
	case bool:
		return typed, true
	default:
		return typed, typed != nil
	}
}

func normalizeListingRevisionBody(payload map[string]any) error {
	listingType, err := normalizeListingRevisionTypeValue(payload["listingType"])
	if err != nil {
		return err
	}
	if listingType != "" {
		payload["listingType"] = listingType
	}
	if listingType == string(marketplacepublishersdk.ListingTypeLeadGeneration) {
		normalizeLeadGenListingRevisionPricingPlans(payload)
	}
	return nil
}

func normalizeListingRevisionTypeValue(value any) (string, error) {
	listingType := strings.TrimSpace(listingRevisionStringValue(value))
	if listingType == "" {
		return "", nil
	}
	if mapped, ok := marketplacepublishersdk.GetMappingListingTypeEnum(listingType); ok {
		return string(mapped), nil
	}
	return "", fmt.Errorf("unsupported %s listingType %q", listingRevisionKind, listingType)
}

func normalizeLeadGenListingRevisionPricingPlans(payload map[string]any) {
	value, ok := lookupListingRevisionMapValue(payload, "pricingPlans")
	if !ok {
		return
	}
	if _, ok := value.(string); ok {
		return
	}
	payload["pricingPlans"] = listingRevisionCompactJSON(value)
}

func validateListingRevisionCreatePayload(payload map[string]any) error {
	required := []string{"listingId", "headline", "listingType"}
	switch listingRevisionStringValue(payload["listingType"]) {
	case string(marketplacepublishersdk.ListingTypeService):
		required = append(required, "productCodes", "industries")
	case string(marketplacepublishersdk.ListingTypeOciApplication), string(marketplacepublishersdk.ListingTypeLeadGeneration):
		required = append(required, "products", "pricingType")
	default:
	}
	return validateListingRevisionRequiredPayloadFields(payload, "create", required...)
}

func validateListingRevisionUpdatePayload(payload map[string]any) error {
	return validateListingRevisionRequiredPayloadFields(payload, "update", "listingType")
}

func validateListingRevisionRequiredPayloadFields(payload map[string]any, operation string, fields ...string) error {
	var missing []string
	for _, field := range fields {
		value, ok := lookupListingRevisionMapValue(payload, field)
		if !ok || !listingRevisionPayloadValueMeaningful(value) {
			missing = append(missing, field)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s %s body is missing required field(s): %s", listingRevisionKind, operation, strings.Join(missing, ", "))
}

func listingRevisionPayloadValueMeaningful(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case []string:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func listingRevisionCreateDetailsFromRaw(raw json.RawMessage) (marketplacepublishersdk.CreateListingRevisionDetails, error) {
	payload, err := listingRevisionBodyMapFromRaw(raw)
	if err != nil {
		return nil, err
	}
	if err := validateListingRevisionCreatePayload(payload); err != nil {
		return nil, err
	}
	raw, err = json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal %s create body: %w", listingRevisionKind, err)
	}
	switch listingRevisionStringValue(payload["listingType"]) {
	case string(marketplacepublishersdk.ListingTypeService):
		var details marketplacepublishersdk.CreateServiceListingRevisionDetails
		if err := json.Unmarshal(raw, &details); err != nil {
			return nil, fmt.Errorf("decode SERVICE %s create body: %w", listingRevisionKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ListingTypeOciApplication):
		var details marketplacepublishersdk.CreateOciListingRevisionDetails
		if err := json.Unmarshal(raw, &details); err != nil {
			return nil, fmt.Errorf("decode OCI_APPLICATION %s create body: %w", listingRevisionKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ListingTypeLeadGeneration):
		var details marketplacepublishersdk.CreateLeadGenListingRevisionDetails
		if err := json.Unmarshal(raw, &details); err != nil {
			return nil, fmt.Errorf("decode LEAD_GENERATION %s create body: %w", listingRevisionKind, err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("unsupported %s listingType %q", listingRevisionKind, listingRevisionStringValue(payload["listingType"]))
	}
}

func listingRevisionUpdateDetailsFromRaw(raw json.RawMessage) (marketplacepublishersdk.UpdateListingRevisionDetails, error) {
	payload, err := listingRevisionBodyMapFromRaw(raw)
	if err != nil {
		return nil, err
	}
	if err := validateListingRevisionUpdatePayload(payload); err != nil {
		return nil, err
	}
	raw, err = json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal %s update body: %w", listingRevisionKind, err)
	}
	switch listingRevisionStringValue(payload["listingType"]) {
	case string(marketplacepublishersdk.ListingTypeService):
		var details marketplacepublishersdk.UpdateServiceListingRevisionDetails
		if err := json.Unmarshal(raw, &details); err != nil {
			return nil, fmt.Errorf("decode SERVICE %s update body: %w", listingRevisionKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ListingTypeOciApplication):
		var details marketplacepublishersdk.UpdateOciListingRevisionDetails
		if err := json.Unmarshal(raw, &details); err != nil {
			return nil, fmt.Errorf("decode OCI_APPLICATION %s update body: %w", listingRevisionKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ListingTypeLeadGeneration):
		var details marketplacepublishersdk.UpdateLeadGenListingRevisionDetails
		if err := json.Unmarshal(raw, &details); err != nil {
			return nil, fmt.Errorf("decode LEAD_GENERATION %s update body: %w", listingRevisionKind, err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("unsupported %s listingType %q", listingRevisionKind, listingRevisionStringValue(payload["listingType"]))
	}
}

func listingRevisionBodyMapFromRaw(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return nil, fmt.Errorf("%s request body is empty", listingRevisionKind)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode %s request body: %w", listingRevisionKind, err)
	}
	if err := normalizeListingRevisionBody(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func adaptListingRevisionOperationResponse(
	revision marketplacepublishersdk.ListingRevision,
	requestID *string,
	etag *string,
) listingRevisionOperationResponse {
	body, _ := listingRevisionBodyMapFromSDK(revision)
	return listingRevisionOperationResponse{
		ListingRevision: body,
		Etag:            etag,
		OpcRequestId:    requestID,
	}
}

func adaptListingRevisionListResponse(response marketplacepublishersdk.ListListingRevisionsResponse) listingRevisionListResponse {
	adapted := listingRevisionListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		body, ok := listingRevisionBodyMapFromSDK(item)
		if !ok || strings.EqualFold(listingRevisionStringValue(body["lifecycleState"]), string(marketplacepublishersdk.ListingRevisionLifecycleStateDeleted)) {
			continue
		}
		adapted.ListingRevisionCollection.Items = append(adapted.ListingRevisionCollection.Items, body)
	}
	return adapted
}

func listingRevisionBodyMapFromSDK(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, false
	}
	return normalizeListingRevisionResponseBody(body, payload), true
}

func normalizeListingRevisionResponseBody(body map[string]any, raw []byte) map[string]any {
	if body == nil {
		body = map[string]any{}
	}
	if status, ok := lookupListingRevisionMapValue(body, "status"); ok {
		body["sdkStatus"] = status
		delete(body, "status")
	}
	if pricingPlans, ok := lookupListingRevisionMapValue(body, "pricingPlans"); ok {
		if _, isString := pricingPlans.(string); !isString {
			body["pricingPlans"] = listingRevisionCompactJSON(pricingPlans)
		}
	}
	if len(raw) > 0 {
		body["jsonData"] = string(raw)
	}
	return body
}

func listingRevisionMapFromResponse(response any) (map[string]any, bool) {
	switch current := response.(type) {
	case listingRevisionOperationResponse:
		return listingRevisionOriginalMap(current.ListingRevision)
	case *listingRevisionOperationResponse:
		if current == nil {
			return nil, false
		}
		return listingRevisionOriginalMap(current.ListingRevision)
	case listingRevisionListResponse:
		return nil, false
	case map[string]any:
		return listingRevisionOriginalMap(current)
	default:
		if body, ok := listingRevisionBodyMapFromSDK(current); ok {
			return listingRevisionOriginalMap(body)
		}
		return nil, false
	}
}

func listingRevisionOriginalMap(body map[string]any) (map[string]any, bool) {
	if body == nil {
		return nil, false
	}
	if original, ok := listingRevisionOriginalRawMap(body); ok {
		return original, true
	}
	return cloneListingRevisionMap(body), true
}

func listingRevisionOriginalRawMap(body map[string]any) (map[string]any, bool) {
	raw := strings.TrimSpace(listingRevisionStringValue(body["jsonData"]))
	if raw == "" {
		return nil, false
	}
	var original map[string]any
	if err := json.Unmarshal([]byte(raw), &original); err != nil || original == nil {
		return nil, false
	}
	if sdkStatus, ok := lookupListingRevisionMapValue(original, "status"); ok {
		original["sdkStatus"] = sdkStatus
	}
	if _, ok := lookupListingRevisionMapValue(original, "pricingPlans"); !ok {
		copyListingRevisionPricingPlans(original, body)
	}
	return original, true
}

func copyListingRevisionPricingPlans(target map[string]any, source map[string]any) {
	pricingPlans, ok := lookupListingRevisionMapValue(source, "pricingPlans")
	if !ok {
		return
	}
	target["pricingPlans"] = pricingPlans
}

func cloneListingRevisionMap(body map[string]any) map[string]any {
	cloned := make(map[string]any, len(body))
	for key, value := range body {
		cloned[key] = value
	}
	return cloned
}

func resolveListingRevisionIdentity(resource *marketplacepublisherv1beta1.ListingRevision) (listingRevisionIdentity, error) {
	if resource == nil {
		return listingRevisionIdentity{}, fmt.Errorf("%s resource is nil", listingRevisionKind)
	}
	statusIdentity := listingRevisionIdentityFromStatus(resource.Status)
	if strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" || statusIdentity.hasLookupIdentity() {
		specIdentity, err := listingRevisionIdentityFromSpec(resource.Spec)
		if err != nil {
			return statusIdentity, nil
		}
		return mergeListingRevisionIdentities(statusIdentity, specIdentity), nil
	}
	return listingRevisionIdentityFromSpec(resource.Spec)
}

func listingRevisionIdentityFromSpec(spec marketplacepublisherv1beta1.ListingRevisionSpec) (listingRevisionIdentity, error) {
	body, err := listingRevisionPayloadFromSpec(spec, "")
	if err != nil {
		return listingRevisionIdentity{}, err
	}
	return listingRevisionIdentityFromMap(body), nil
}

func listingRevisionIdentityFromStatus(status marketplacepublisherv1beta1.ListingRevisionStatus) listingRevisionIdentity {
	return listingRevisionIdentity{
		listingID:   strings.TrimSpace(status.ListingId),
		displayName: strings.TrimSpace(status.DisplayName),
		listingType: strings.TrimSpace(status.ListingType),
	}
}

func listingRevisionIdentityFromMap(body map[string]any) listingRevisionIdentity {
	return listingRevisionIdentity{
		listingID:   strings.TrimSpace(listingRevisionStringValue(body["listingId"])),
		displayName: strings.TrimSpace(listingRevisionStringValue(body["displayName"])),
		listingType: strings.TrimSpace(listingRevisionStringValue(body["listingType"])),
	}
}

func mergeListingRevisionIdentities(primary listingRevisionIdentity, fallback listingRevisionIdentity) listingRevisionIdentity {
	if primary.listingID == "" {
		primary.listingID = fallback.listingID
	}
	if primary.displayName == "" {
		primary.displayName = fallback.displayName
	}
	if primary.listingType == "" {
		primary.listingType = fallback.listingType
	}
	return primary
}

func (i listingRevisionIdentity) hasLookupIdentity() bool {
	return i.listingID != "" && (i.displayName != "" || i.listingType != "")
}

func recordListingRevisionPathIdentity(resource *marketplacepublisherv1beta1.ListingRevision, identity any) {
	if resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	typed, ok := identity.(listingRevisionIdentity)
	if !ok {
		return
	}
	resource.Status.ListingId = typed.listingID
	resource.Status.DisplayName = typed.displayName
	resource.Status.ListingType = typed.listingType
}

func guardListingRevisionExistingBeforeCreate(
	_ context.Context,
	resource *marketplacepublisherv1beta1.ListingRevision,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveListingRevisionIdentity(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	if identity.listingID == "" || (identity.displayName == "" && identity.listingType == "") {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateListingRevisionCreateOnlyDriftForResponse(
	resource *marketplacepublisherv1beta1.ListingRevision,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", listingRevisionKind)
	}
	current, ok := listingRevisionMapFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current %s response does not expose a listing revision body", listingRevisionKind)
	}
	return validateListingRevisionCreateOnlyDrift(resource.Spec, current)
}

func validateListingRevisionCreateOnlyDrift(
	spec marketplacepublisherv1beta1.ListingRevisionSpec,
	current map[string]any,
) error {
	desired, err := listingRevisionPayloadFromSpec(spec, listingRevisionStringValue(current["listingType"]))
	if err != nil {
		return err
	}
	if err := validateListingRevisionStatusUpdateDrift(desired, current); err != nil {
		return err
	}
	var drift []string
	for _, field := range []string{"listingId", "listingType"} {
		desiredValue, desiredOK := lookupListingRevisionMapValue(desired, field)
		currentValue, currentOK := lookupListingRevisionMapValue(current, field)
		if desiredOK && currentOK && !listingRevisionValuesEqualForField(field, desiredValue, currentValue) {
			drift = append(drift, field)
		}
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("%s create-only field drift is not supported: %s", listingRevisionKind, strings.Join(drift, ", "))
}

func validateListingRevisionStatusUpdateDrift(desired map[string]any, current map[string]any) error {
	desiredStatus, ok := lookupListingRevisionMapValue(desired, "status")
	if !ok || !listingRevisionPayloadValueMeaningful(desiredStatus) {
		return nil
	}
	currentStatus, ok := lookupListingRevisionCurrentStatus(current)
	if ok && listingRevisionValuesEqualForField("status", desiredStatus, currentStatus) {
		return nil
	}
	return fmt.Errorf(
		"%s status update is not supported; current OCI status %q does not match desired spec.status %q",
		listingRevisionKind,
		listingRevisionStringValue(currentStatus),
		listingRevisionStringValue(desiredStatus),
	)
}

func lookupListingRevisionCurrentStatus(current map[string]any) (any, bool) {
	if status, ok := lookupListingRevisionMapValue(current, "sdkStatus"); ok {
		return status, true
	}
	return lookupListingRevisionMapValue(current, "status")
}

func listListingRevisionsAllPages(
	ctx context.Context,
	client listingRevisionOCIClient,
	initErr error,
	request marketplacepublishersdk.ListListingRevisionsRequest,
) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
	if initErr != nil {
		return marketplacepublishersdk.ListListingRevisionsResponse{}, fmt.Errorf("initialize %s OCI client: %w", listingRevisionKind, initErr)
	}
	if client == nil {
		return marketplacepublishersdk.ListListingRevisionsResponse{}, fmt.Errorf("%s OCI client is not configured", listingRevisionKind)
	}
	var combined marketplacepublishersdk.ListListingRevisionsResponse
	for {
		response, err := client.ListListingRevisions(ctx, request)
		if err != nil {
			return marketplacepublishersdk.ListListingRevisionsResponse{}, conservativeListingRevisionNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == marketplacepublishersdk.ListingRevisionLifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func confirmListingRevisionDeleteRead(
	ctx context.Context,
	hooks *ListingRevisionRuntimeHooks,
	resource *marketplacepublisherv1beta1.ListingRevision,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm %s delete: runtime hooks are nil", listingRevisionKind)
	}
	if listingRevisionID := strings.TrimSpace(currentID); listingRevisionID != "" {
		return confirmListingRevisionDeleteReadByID(ctx, hooks, listingRevisionID)
	}
	return confirmListingRevisionDeleteReadByList(ctx, hooks, resource)
}

func confirmListingRevisionDeleteReadByID(
	ctx context.Context,
	hooks *ListingRevisionRuntimeHooks,
	listingRevisionID string,
) (any, error) {
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("confirm %s delete: get hook is not configured", listingRevisionKind)
	}
	response, err := hooks.Get.Call(ctx, marketplacepublishersdk.GetListingRevisionRequest{ListingRevisionId: common.String(listingRevisionID)})
	if err != nil {
		return listingRevisionDeleteConfirmReadResponse(nil, err)
	}
	return adaptListingRevisionOperationResponse(response.ListingRevision, response.OpcRequestId, response.Etag), nil
}

func confirmListingRevisionDeleteReadByList(
	ctx context.Context,
	hooks *ListingRevisionRuntimeHooks,
	resource *marketplacepublisherv1beta1.ListingRevision,
) (any, error) {
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm %s delete: list hook is not configured", listingRevisionKind)
	}
	request, identity, ok, err := listingRevisionDeleteListRequest(resource)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("confirm %s delete: listing revision identity is not recorded", listingRevisionKind)
	}
	response, err := hooks.List.Call(ctx, request)
	if err != nil {
		return listingRevisionDeleteConfirmReadResponse(nil, err)
	}
	matches := []map[string]any{}
	for _, item := range response.Items {
		body, bodyOK := listingRevisionBodyMapFromSDK(item)
		if !bodyOK {
			continue
		}
		if listingRevisionMatchesIdentity(identity, body) {
			matches = append(matches, body)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
	default:
		return nil, fmt.Errorf(
			"%s delete confirmation found %d listing revisions matching listingId %q, displayName %q, and listingType %q",
			listingRevisionKind,
			len(matches),
			identity.listingID,
			identity.displayName,
			identity.listingType,
		)
	}
	return nil, errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "ListingRevision delete confirmation did not find a matching OCI listing revision",
	}
}

func listingRevisionDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	convertedErr := conservativeListingRevisionNotFoundError(err, "delete confirmation")
	if ambiguous, ok := asAmbiguousListingRevisionNotFoundError(convertedErr); ok {
		return listingRevisionAmbiguousNotFoundError{
			message:      fmt.Sprintf("%s delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound: %s", listingRevisionKind, err.Error()),
			opcRequestID: ambiguous.opcRequestID,
		}, nil
	}
	return nil, convertedErr
}

func listingRevisionDeleteListRequest(resource *marketplacepublisherv1beta1.ListingRevision) (
	marketplacepublishersdk.ListListingRevisionsRequest,
	listingRevisionIdentity,
	bool,
	error,
) {
	identity, err := resolveListingRevisionIdentity(resource)
	if err != nil {
		return marketplacepublishersdk.ListListingRevisionsRequest{}, listingRevisionIdentity{}, false, err
	}
	request := marketplacepublishersdk.ListListingRevisionsRequest{}
	if identity.listingID != "" {
		request.ListingId = common.String(identity.listingID)
	}
	if identity.displayName != "" {
		request.DisplayName = common.String(identity.displayName)
	}
	return request, identity, request.ListingId != nil && (request.DisplayName != nil || identity.listingType != ""), nil
}

func listingRevisionMatchesIdentity(identity listingRevisionIdentity, body map[string]any) bool {
	if body == nil || strings.EqualFold(listingRevisionStringValue(body["lifecycleState"]), string(marketplacepublishersdk.ListingRevisionLifecycleStateDeleted)) {
		return false
	}
	if identity.listingID != "" && !listingRevisionValuesEqualForField("listingId", identity.listingID, body["listingId"]) {
		return false
	}
	if identity.displayName != "" && !listingRevisionValuesEqualForField("displayName", identity.displayName, body["displayName"]) {
		return false
	}
	if identity.listingType != "" && !listingRevisionValuesEqualForField("listingType", identity.listingType, body["listingType"]) {
		return false
	}
	return strings.TrimSpace(listingRevisionStringValue(body["id"])) != ""
}

func handleListingRevisionDeleteError(resource *marketplacepublisherv1beta1.ListingRevision, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func applyListingRevisionDeleteOutcome(
	resource *marketplacepublisherv1beta1.ListingRevision,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	ambiguous, ok := response.(listingRevisionAmbiguousNotFoundError)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
	}
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
}

func conservativeListingRevisionNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", listingRevisionKind, strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return listingRevisionAmbiguousNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return listingRevisionAmbiguousNotFoundError{message: message}
}

func asAmbiguousListingRevisionNotFoundError(err error) (listingRevisionAmbiguousNotFoundError, bool) {
	var ambiguous listingRevisionAmbiguousNotFoundError
	if errors.As(err, &ambiguous) {
		return ambiguous, true
	}
	return listingRevisionAmbiguousNotFoundError{}, false
}

func clearTrackedListingRevisionIdentity(resource *marketplacepublisherv1beta1.ListingRevision) {
	if resource == nil {
		return
	}
	resource.Status = marketplacepublisherv1beta1.ListingRevisionStatus{}
}

func lookupListingRevisionMapValue(values map[string]any, field string) (any, bool) {
	if values == nil {
		return nil, false
	}
	if value, ok := values[field]; ok {
		return value, true
	}
	normalized := normalizeListingRevisionPathSegment(field)
	for key, value := range values {
		if normalizeListingRevisionPathSegment(key) == normalized {
			return value, true
		}
	}
	return nil, false
}

func normalizeListingRevisionPathSegment(segment string) string {
	segment = strings.ToLower(strings.TrimSpace(segment))
	return strings.ReplaceAll(segment, "_", "")
}

func listingRevisionValuesEqualForField(field string, left any, right any) bool {
	if field == "pricingPlans" {
		left = normalizeListingRevisionPricingPlansComparable(left)
		right = normalizeListingRevisionPricingPlansComparable(right)
	}
	if field == "listingType" {
		leftType, leftErr := normalizeListingRevisionTypeValue(left)
		rightType, rightErr := normalizeListingRevisionTypeValue(right)
		if leftErr == nil && rightErr == nil {
			return leftType == rightType
		}
	}
	return listingRevisionValuesEqual(left, right)
}

func normalizeListingRevisionPricingPlansComparable(value any) any {
	if raw := strings.TrimSpace(listingRevisionStringValue(value)); raw != "" {
		var decoded any
		if json.Unmarshal([]byte(raw), &decoded) == nil {
			return decoded
		}
	}
	return value
}

func listingRevisionValuesEqual(left any, right any) bool {
	left, leftOK := normalizeListingRevisionComparableValue(left)
	right, rightOK := normalizeListingRevisionComparableValue(right)
	if !leftOK || !rightOK {
		return leftOK == rightOK
	}
	if reflect.DeepEqual(left, right) {
		return true
	}
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func normalizeListingRevisionComparableValue(value any) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, false
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return nil, false
		}
		return value, true
	case []any:
		return normalizeListingRevisionComparableSlice(typed)
	case map[string]any:
		return normalizeListingRevisionComparableMap(typed)
	default:
		return typed, true
	}
}

func normalizeListingRevisionComparableSlice(values []any) ([]any, bool) {
	normalized := make([]any, 0, len(values))
	for _, item := range values {
		value, ok := normalizeListingRevisionComparableValue(item)
		if !ok {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized, len(normalized) > 0
}

func normalizeListingRevisionComparableMap(values map[string]any) (map[string]any, bool) {
	normalized := make(map[string]any, len(values))
	for key, item := range values {
		value, ok := normalizeListingRevisionComparableValue(item)
		if !ok {
			continue
		}
		normalized[key] = value
	}
	return normalized, len(normalized) > 0
}

func listingRevisionStringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case *string:
		if typed == nil {
			return ""
		}
		return *typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func listingRevisionCompactJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(payload)
}
