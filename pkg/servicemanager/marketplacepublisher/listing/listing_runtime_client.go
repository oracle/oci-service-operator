/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listing

import (
	"context"
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
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type listingOCIClient interface {
	CreateListing(context.Context, marketplacepublishersdk.CreateListingRequest) (marketplacepublishersdk.CreateListingResponse, error)
	GetListing(context.Context, marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error)
	ListListings(context.Context, marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error)
	UpdateListing(context.Context, marketplacepublishersdk.UpdateListingRequest) (marketplacepublishersdk.UpdateListingResponse, error)
	DeleteListing(context.Context, marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error)
}

type listingListCall func(context.Context, marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error)

type listingAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e listingAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e listingAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerListingRuntimeHooksMutator(func(_ *ListingServiceManager, hooks *ListingRuntimeHooks) {
		applyListingRuntimeHooks(hooks)
	})
}

func applyListingRuntimeHooks(hooks *ListingRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedListingRuntimeSemantics()
	hooks.BuildCreateBody = buildListingCreateBody
	hooks.BuildUpdateBody = buildListingUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardListingExistingBeforeCreate
	hooks.List.Fields = listingListFields()
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateListingCreateOnlyDriftForResponse
	hooks.DeleteHooks.ApplyOutcome = applyListingDeleteOutcome
	hooks.DeleteHooks.HandleError = handleListingDeleteError
	if hooks.List.Call != nil {
		hooks.List.Call = listListingsAllPages(hooks.List.Call)
	}
	wrapListingDeleteConfirmation(hooks)
}

func newListingServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client listingOCIClient,
) ListingServiceClient {
	hooks := newListingRuntimeHooksWithOCIClient(client)
	applyListingRuntimeHooks(&hooks)
	delegate := defaultListingServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.Listing](
			buildListingGeneratedRuntimeConfig(&ListingServiceManager{Log: log}, hooks),
		),
	}
	return wrapListingGeneratedClient(hooks, delegate)
}

func newListingRuntimeHooksWithOCIClient(client listingOCIClient) ListingRuntimeHooks {
	return ListingRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*marketplacepublisherv1beta1.Listing]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*marketplacepublisherv1beta1.Listing]{},
		StatusHooks:     generatedruntime.StatusHooks[*marketplacepublisherv1beta1.Listing]{},
		ParityHooks:     generatedruntime.ParityHooks[*marketplacepublisherv1beta1.Listing]{},
		Async:           generatedruntime.AsyncHooks[*marketplacepublisherv1beta1.Listing]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*marketplacepublisherv1beta1.Listing]{},
		Create: runtimeOperationHooks[marketplacepublishersdk.CreateListingRequest, marketplacepublishersdk.CreateListingResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateListingDetails", RequestName: "CreateListingDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request marketplacepublishersdk.CreateListingRequest) (marketplacepublishersdk.CreateListingResponse, error) {
				if client == nil {
					return marketplacepublishersdk.CreateListingResponse{}, fmt.Errorf("listing OCI client is nil")
				}
				return client.CreateListing(ctx, request)
			},
		},
		Get: runtimeOperationHooks[marketplacepublishersdk.GetListingRequest, marketplacepublishersdk.GetListingResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ListingId", RequestName: "listingId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
				if client == nil {
					return marketplacepublishersdk.GetListingResponse{}, fmt.Errorf("listing OCI client is nil")
				}
				return client.GetListing(ctx, request)
			},
		},
		List: runtimeOperationHooks[marketplacepublishersdk.ListListingsRequest, marketplacepublishersdk.ListListingsResponse]{
			Fields: listingListFields(),
			Call: func(ctx context.Context, request marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error) {
				if client == nil {
					return marketplacepublishersdk.ListListingsResponse{}, fmt.Errorf("listing OCI client is nil")
				}
				return client.ListListings(ctx, request)
			},
		},
		Update: runtimeOperationHooks[marketplacepublishersdk.UpdateListingRequest, marketplacepublishersdk.UpdateListingResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ListingId", RequestName: "listingId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateListingDetails", RequestName: "UpdateListingDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request marketplacepublishersdk.UpdateListingRequest) (marketplacepublishersdk.UpdateListingResponse, error) {
				if client == nil {
					return marketplacepublishersdk.UpdateListingResponse{}, fmt.Errorf("listing OCI client is nil")
				}
				return client.UpdateListing(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[marketplacepublishersdk.DeleteListingRequest, marketplacepublishersdk.DeleteListingResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ListingId", RequestName: "listingId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error) {
				if client == nil {
					return marketplacepublishersdk.DeleteListingResponse{}, fmt.Errorf("listing OCI client is nil")
				}
				return client.DeleteListing(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ListingServiceClient) ListingServiceClient{},
	}
}

func reviewedListingRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplacepublisher",
		FormalSlug:    "listing",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(marketplacepublishersdk.ListingLifecycleStateCreating)},
			UpdatingStates:     []string{string(marketplacepublishersdk.ListingLifecycleStateUpdating)},
			ActiveStates: []string{
				string(marketplacepublishersdk.ListingLifecycleStateActive),
				string(marketplacepublishersdk.ListingLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(marketplacepublishersdk.ListingLifecycleStateDeleting)},
			TerminalStates: []string{string(marketplacepublishersdk.ListingLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "listingType", "packageType"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"name",
				"listingType",
				"packageType",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Listing", Action: "CreateListing"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Listing", Action: "UpdateListing"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Listing", Action: "DeleteListing"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Listing", Action: "GetListing"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Listing", Action: "GetListing"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Listing", Action: "GetListing"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func listingListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "ListingType", RequestName: "listingType", Contribution: "query", LookupPaths: []string{"status.listingType", "spec.listingType", "listingType"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "metadataName", "name"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func buildListingCreateBody(
	_ context.Context,
	resource *marketplacepublisherv1beta1.Listing,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("listing resource is nil")
	}
	if err := validateListingSpec(resource.Spec); err != nil {
		return nil, err
	}

	listingType, _ := listingTypeFromSpec(resource.Spec.ListingType)
	packageType, _ := packageTypeFromSpec(resource.Spec.PackageType)
	body := marketplacepublishersdk.CreateListingDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		Name:          common.String(strings.TrimSpace(resource.Spec.Name)),
		ListingType:   listingType,
		PackageType:   packageType,
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneListingStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = listingDefinedTags(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildListingUpdateBody(
	_ context.Context,
	resource *marketplacepublisherv1beta1.Listing,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return marketplacepublishersdk.UpdateListingDetails{}, false, fmt.Errorf("listing resource is nil")
	}
	if err := validateListingSpec(resource.Spec); err != nil {
		return marketplacepublishersdk.UpdateListingDetails{}, false, err
	}

	current, ok := listingFromResponse(currentResponse)
	if !ok {
		return marketplacepublishersdk.UpdateListingDetails{}, false, fmt.Errorf("current listing response does not expose a listing body")
	}
	if err := validateListingCreateOnlyDrift(resource.Spec, current); err != nil {
		return marketplacepublishersdk.UpdateListingDetails{}, false, err
	}

	updateDetails := marketplacepublishersdk.UpdateListingDetails{}
	updateNeeded := false
	if resource.Spec.FreeformTags != nil {
		desired := cloneListingStringMap(resource.Spec.FreeformTags)
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateDetails.FreeformTags = desired
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := listingDefinedTags(resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateDetails.DefinedTags = desired
			updateNeeded = true
		}
	}
	if !updateNeeded {
		return marketplacepublishersdk.UpdateListingDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func validateListingSpec(spec marketplacepublisherv1beta1.ListingSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(spec.ListingType) == "" {
		missing = append(missing, "listingType")
	}
	if strings.TrimSpace(spec.PackageType) == "" {
		missing = append(missing, "packageType")
	}
	if len(missing) > 0 {
		return fmt.Errorf("listing spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if _, err := listingTypeFromSpec(spec.ListingType); err != nil {
		return err
	}
	if _, err := packageTypeFromSpec(spec.PackageType); err != nil {
		return err
	}
	return nil
}

func listingTypeFromSpec(value string) (marketplacepublishersdk.ListingTypeEnum, error) {
	trimmed := strings.TrimSpace(value)
	if listingType, ok := marketplacepublishersdk.GetMappingListingTypeEnum(trimmed); ok {
		return listingType, nil
	}
	return "", fmt.Errorf("listing spec has unsupported listingType %q", value)
}

func packageTypeFromSpec(value string) (marketplacepublishersdk.PackageTypeEnum, error) {
	trimmed := strings.TrimSpace(value)
	if packageType, ok := marketplacepublishersdk.GetMappingPackageTypeEnum(trimmed); ok {
		return packageType, nil
	}
	return "", fmt.Errorf("listing spec has unsupported packageType %q", value)
}

func guardListingExistingBeforeCreate(
	_ context.Context,
	resource *marketplacepublisherv1beta1.Listing,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("listing resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.Name) == "" ||
		strings.TrimSpace(resource.Spec.ListingType) == "" ||
		strings.TrimSpace(resource.Spec.PackageType) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateListingCreateOnlyDriftForResponse(
	resource *marketplacepublisherv1beta1.Listing,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("listing resource is nil")
	}
	current, ok := listingFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current listing response does not expose a listing body")
	}
	return validateListingCreateOnlyDrift(resource.Spec, current)
}

func validateListingCreateOnlyDrift(
	spec marketplacepublisherv1beta1.ListingSpec,
	current marketplacepublishersdk.Listing,
) error {
	var drift []string
	if !stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !stringPtrEqual(current.Name, spec.Name) {
		drift = append(drift, "name")
	}
	if !strings.EqualFold(string(current.ListingType), strings.TrimSpace(spec.ListingType)) {
		drift = append(drift, "listingType")
	}
	if !strings.EqualFold(string(current.PackageType), strings.TrimSpace(spec.PackageType)) {
		drift = append(drift, "packageType")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("listing create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func listListingsAllPages(call listingListCall) listingListCall {
	return func(ctx context.Context, request marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error) {
		if call == nil {
			return marketplacepublishersdk.ListListingsResponse{}, fmt.Errorf("listing list operation is not configured")
		}

		accumulator := newListingListAccumulator()
		for {
			response, err := call(ctx, request)
			if err != nil {
				return marketplacepublishersdk.ListListingsResponse{}, err
			}
			accumulator.append(response)

			nextPage := stringValue(response.OpcNextPage)
			if nextPage == "" {
				accumulator.response.OpcNextPage = nil
				return accumulator.response, nil
			}
			if err := accumulator.advance(&request, nextPage); err != nil {
				return marketplacepublishersdk.ListListingsResponse{}, err
			}
		}
	}
}

type listingListAccumulator struct {
	response  marketplacepublishersdk.ListListingsResponse
	seenPages map[string]struct{}
}

func newListingListAccumulator() listingListAccumulator {
	return listingListAccumulator{seenPages: map[string]struct{}{}}
}

func (a *listingListAccumulator) append(response marketplacepublishersdk.ListListingsResponse) {
	if a.response.RawResponse == nil {
		a.response.RawResponse = response.RawResponse
	}
	if a.response.OpcRequestId == nil {
		a.response.OpcRequestId = response.OpcRequestId
	}
	a.response.OpcNextPage = response.OpcNextPage
	a.response.Items = append(a.response.Items, response.Items...)
}

func (a *listingListAccumulator) advance(request *marketplacepublishersdk.ListListingsRequest, nextPage string) error {
	if _, ok := a.seenPages[nextPage]; ok {
		return fmt.Errorf("listing list pagination repeated page token %q", nextPage)
	}
	a.seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	return nil
}

func applyListingDeleteOutcome(
	resource *marketplacepublisherv1beta1.Listing,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	current, ok := listingFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	state := strings.ToUpper(string(current.LifecycleState))
	switch stage {
	case generatedruntime.DeleteConfirmStageAlreadyPending:
		if state == string(marketplacepublishersdk.ListingLifecycleStateDeleting) {
			markListingTerminating(resource, state, "OCI resource delete is in progress")
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	case generatedruntime.DeleteConfirmStageAfterRequest:
		switch state {
		case string(marketplacepublishersdk.ListingLifecycleStateCreating),
			string(marketplacepublishersdk.ListingLifecycleStateUpdating),
			string(marketplacepublishersdk.ListingLifecycleStateActive),
			string(marketplacepublishersdk.ListingLifecycleStateInactive),
			string(marketplacepublishersdk.ListingLifecycleStateDeleting):
			markListingTerminating(resource, state, "OCI resource delete is in progress")
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markListingTerminating(resource *marketplacepublisherv1beta1.Listing, rawStatus string, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	_ = servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       strings.TrimSpace(rawStatus),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
		UpdatedAt:       &now,
	}, loggerutil.OSOKLogger{})
}

func handleListingDeleteError(resource *marketplacepublisherv1beta1.Listing, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return listingAmbiguousNotFoundError{
		message:      "listing delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapListingDeleteConfirmation(hooks *ListingRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getListing := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ListingServiceClient) ListingServiceClient {
		return listingDeleteConfirmationClient{
			delegate:   delegate,
			getListing: getListing,
		}
	})
}

type listingDeleteConfirmationClient struct {
	delegate   ListingServiceClient
	getListing func(context.Context, marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error)
}

func (c listingDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Listing,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c listingDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Listing,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c listingDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Listing,
) error {
	if c.getListing == nil || resource == nil {
		return nil
	}
	listingID := trackedListingID(resource)
	if listingID == "" {
		return nil
	}
	_, err := c.getListing(ctx, marketplacepublishersdk.GetListingRequest{ListingId: common.String(listingID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("listing delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to confirm deletion: %w", err)
}

func trackedListingID(resource *marketplacepublisherv1beta1.Listing) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func listingFromResponse(response any) (marketplacepublishersdk.Listing, bool) {
	switch current := response.(type) {
	case marketplacepublishersdk.CreateListingResponse:
		return current.Listing, true
	case marketplacepublishersdk.GetListingResponse:
		return current.Listing, true
	case marketplacepublishersdk.UpdateListingResponse:
		return current.Listing, true
	case marketplacepublishersdk.Listing:
		return current, true
	case marketplacepublishersdk.ListingSummary:
		return listingFromSummary(current), true
	default:
		return listingFromPointerResponse(response)
	}
}

func listingFromPointerResponse(response any) (marketplacepublishersdk.Listing, bool) {
	switch current := response.(type) {
	case *marketplacepublishersdk.CreateListingResponse:
		return listingFromPointer(current, func(response marketplacepublishersdk.CreateListingResponse) marketplacepublishersdk.Listing {
			return response.Listing
		})
	case *marketplacepublishersdk.GetListingResponse:
		return listingFromPointer(current, func(response marketplacepublishersdk.GetListingResponse) marketplacepublishersdk.Listing {
			return response.Listing
		})
	case *marketplacepublishersdk.UpdateListingResponse:
		return listingFromPointer(current, func(response marketplacepublishersdk.UpdateListingResponse) marketplacepublishersdk.Listing {
			return response.Listing
		})
	case *marketplacepublishersdk.Listing:
		return listingFromPointer(current, func(listing marketplacepublishersdk.Listing) marketplacepublishersdk.Listing {
			return listing
		})
	case *marketplacepublishersdk.ListingSummary:
		return listingFromPointer(current, listingFromSummary)
	default:
		return marketplacepublishersdk.Listing{}, false
	}
}

func listingFromPointer[T any](
	current *T,
	convert func(T) marketplacepublishersdk.Listing,
) (marketplacepublishersdk.Listing, bool) {
	if current == nil {
		return marketplacepublishersdk.Listing{}, false
	}
	return convert(*current), true
}

func listingFromSummary(summary marketplacepublishersdk.ListingSummary) marketplacepublishersdk.Listing {
	return marketplacepublishersdk.Listing{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		ListingType:    marketplacepublishersdk.ListingTypeEnum(summary.ListingType),
		Name:           summary.Name,
		PackageType:    summary.PackageType,
		TimeCreated:    summary.TimeCreated,
		TimeUpdated:    summary.TimeUpdated,
		LifecycleState: summary.LifecycleState,
		FreeformTags:   cloneListingStringMap(summary.FreeformTags),
		DefinedTags:    cloneListingDefinedTags(summary.DefinedTags),
		SystemTags:     cloneListingDefinedTags(summary.SystemTags),
	}
}

func listingDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	converted := util.ConvertToOciDefinedTags(&tags)
	if converted == nil {
		return nil
	}
	return *converted
}

func cloneListingStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneListingDefinedTags(in map[string]map[string]interface{}) map[string]map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(in))
	for namespace, values := range in {
		copiedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			copiedValues[key] = value
		}
		out[namespace] = copiedValues
	}
	return out
}

func stringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(stringValue(current)) == strings.TrimSpace(desired)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
