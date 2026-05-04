/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package offer

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	offersdk "github.com/oracle/oci-go-sdk/v65/marketplaceprivateoffer"
	offerv1beta1 "github.com/oracle/oci-service-operator/api/marketplaceprivateoffer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type offerOCIClient interface {
	CreateOffer(context.Context, offersdk.CreateOfferRequest) (offersdk.CreateOfferResponse, error)
	GetOffer(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error)
	GetOfferInternalDetail(context.Context, offersdk.GetOfferInternalDetailRequest) (offersdk.GetOfferInternalDetailResponse, error)
	ListOffers(context.Context, offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error)
	UpdateOffer(context.Context, offersdk.UpdateOfferRequest) (offersdk.UpdateOfferResponse, error)
	DeleteOffer(context.Context, offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error)
}

type ambiguousOfferNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousOfferNotFoundError) Error() string {
	return e.message
}

func (e ambiguousOfferNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerOfferRuntimeHooksMutator(func(manager *OfferServiceManager, hooks *OfferRuntimeHooks) {
		client, initErr := newOfferSDKClient(manager)
		applyOfferRuntimeHooks(hooks, client, initErr)
	})
}

func newOfferSDKClient(manager *OfferServiceManager) (offerOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("offer service manager is nil")
	}
	client, err := offersdk.NewOfferClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOfferRuntimeHooks(hooks *OfferRuntimeHooks, client offerOCIClient, initErr error) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOfferRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *offerv1beta1.Offer, _ string) (any, error) {
		return buildOfferCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(ctx context.Context, resource *offerv1beta1.Offer, _ string, currentResponse any) (any, bool, error) {
		return buildOfferUpdateBody(ctx, resource, currentResponse, client, initErr)
	}
	applyOfferOperationHooks(hooks, client, initErr)
	hooks.List.Fields = offerListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedOfferIdentity
	hooks.StatusHooks.ProjectStatus = offerStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateOfferCreateOnlyDrift
	hooks.DeleteHooks.ConfirmRead = confirmOfferDeleteRead(client, initErr)
	hooks.DeleteHooks.HandleError = rejectOfferAuthShapedNotFound
	hooks.DeleteHooks.ApplyOutcome = applyOfferDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OfferServiceClient) OfferServiceClient {
		return offerDeleteFallbackClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	})
}

func applyOfferOperationHooks(hooks *OfferRuntimeHooks, client offerOCIClient, initErr error) {
	hooks.Create.Call = func(ctx context.Context, request offersdk.CreateOfferRequest) (offersdk.CreateOfferResponse, error) {
		if err := requireOfferOCIClient(client, initErr); err != nil {
			return offersdk.CreateOfferResponse{}, err
		}
		return client.CreateOffer(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		if err := requireOfferOCIClient(client, initErr); err != nil {
			return offersdk.GetOfferResponse{}, err
		}
		return client.GetOffer(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error) {
		return listOfferPages(ctx, client, initErr, request)
	}
	hooks.Update.Call = func(ctx context.Context, request offersdk.UpdateOfferRequest) (offersdk.UpdateOfferResponse, error) {
		if err := requireOfferOCIClient(client, initErr); err != nil {
			return offersdk.UpdateOfferResponse{}, err
		}
		return client.UpdateOffer(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
		if err := requireOfferOCIClient(client, initErr); err != nil {
			return offersdk.DeleteOfferResponse{}, err
		}
		return client.DeleteOffer(ctx, request)
	}
}

func requireOfferOCIClient(client offerOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize offer OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("offer OCI client is not configured")
	}
	return nil
}

func newOfferRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "marketplaceprivateoffer",
		FormalSlug:        "offer",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(offersdk.OfferLifecycleStateCreating)},
			UpdatingStates:     []string{string(offersdk.OfferLifecycleStateUpdating)},
			ActiveStates:       []string{string(offersdk.OfferLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(offersdk.OfferLifecycleStateDeleting)},
			TerminalStates: []string{string(offersdk.OfferLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"sellerCompartmentId", "buyerCompartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"buyerCompartmentId",
				"buyerInformation",
				"customFields",
				"definedTags",
				"description",
				"displayName",
				"duration",
				"freeformTags",
				"internalNotes",
				"pricing",
				"resourceBundles",
				"sellerInformation",
				"timeAcceptBy",
				"timeStartDate",
			},
			ForceNew:      []string{"sellerCompartmentId"},
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

func offerListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SellerCompartmentId",
			RequestName:  "sellerCompartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.sellerCompartmentId", "spec.sellerCompartmentId", "sellerCompartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func listOfferPages(
	ctx context.Context,
	client offerOCIClient,
	initErr error,
	request offersdk.ListOffersRequest,
) (offersdk.ListOffersResponse, error) {
	if err := requireOfferOCIClient(client, initErr); err != nil {
		return offersdk.ListOffersResponse{}, err
	}

	seenPages := map[string]struct{}{}
	var combined offersdk.ListOffersResponse
	for {
		response, err := client.ListOffers(ctx, request)
		if err != nil {
			return offersdk.ListOffersResponse{}, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := offerStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return offersdk.ListOffersResponse{}, fmt.Errorf("offer list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func buildOfferCreateBody(resource *offerv1beta1.Offer) (offersdk.CreateOfferDetails, error) {
	if resource == nil {
		return offersdk.CreateOfferDetails{}, fmt.Errorf("offer resource is nil")
	}
	if err := validateOfferSpec(resource.Spec); err != nil {
		return offersdk.CreateOfferDetails{}, err
	}

	spec := resource.Spec
	body := offersdk.CreateOfferDetails{
		DisplayName:         offerString(spec.DisplayName),
		SellerCompartmentId: offerString(spec.SellerCompartmentId),
	}
	if err := applyOfferMutableFieldsForCreate(&body, spec); err != nil {
		return offersdk.CreateOfferDetails{}, err
	}
	return body, nil
}

func buildOfferUpdateBody(
	ctx context.Context,
	resource *offerv1beta1.Offer,
	currentResponse any,
	client offerOCIClient,
	initErr error,
) (offersdk.UpdateOfferDetails, bool, error) {
	if resource == nil {
		return offersdk.UpdateOfferDetails{}, false, fmt.Errorf("offer resource is nil")
	}
	if err := validateOfferSpec(resource.Spec); err != nil {
		return offersdk.UpdateOfferDetails{}, false, err
	}

	current, err := offerRuntimeBody(currentResponse)
	if err != nil {
		return offersdk.UpdateOfferDetails{}, false, err
	}
	if err := validateOfferCreateOnlyDriftForCurrent(resource.Spec, current); err != nil {
		return offersdk.UpdateOfferDetails{}, false, err
	}

	details, updateNeeded, err := offerExternalUpdateDetails(resource.Spec, current)
	if err != nil {
		return offersdk.UpdateOfferDetails{}, false, err
	}

	internalDetails, hasInternalDetails, err := offerInternalDetailsForUpdate(ctx, resource, current, client, initErr)
	if err != nil {
		return offersdk.UpdateOfferDetails{}, false, err
	}
	offerApplyInternalUpdateDetails(&details, &updateNeeded, resource.Spec, internalDetails, hasInternalDetails)

	return details, updateNeeded, nil
}

type offerUpdateBuilder struct {
	details      offersdk.UpdateOfferDetails
	updateNeeded bool
}

func offerExternalUpdateDetails(spec offerv1beta1.OfferSpec, current offersdk.Offer) (offersdk.UpdateOfferDetails, bool, error) {
	builder := offerUpdateBuilder{}
	builder.setString(spec.DisplayName, current.DisplayName, func(value *string) { builder.details.DisplayName = value })
	builder.setString(spec.BuyerCompartmentId, current.BuyerCompartmentId, func(value *string) { builder.details.BuyerCompartmentId = value })
	builder.setString(spec.Description, current.Description, func(value *string) { builder.details.Description = value })
	if err := builder.setTime("timeStartDate", spec.TimeStartDate, current.TimeStartDate, func(value *common.SDKTime) {
		builder.details.TimeStartDate = value
	}); err != nil {
		return offersdk.UpdateOfferDetails{}, false, err
	}
	builder.setString(spec.Duration, current.Duration, func(value *string) { builder.details.Duration = value })
	if err := builder.setTime("timeAcceptBy", spec.TimeAcceptBy, current.TimeAcceptBy, func(value *common.SDKTime) {
		builder.details.TimeAcceptBy = value
	}); err != nil {
		return offersdk.UpdateOfferDetails{}, false, err
	}
	builder.setPricing(spec.Pricing, current.Pricing)
	builder.setBuyerInformation(spec.BuyerInformation, current.BuyerInformation)
	builder.setSellerInformation(spec.SellerInformation, current.SellerInformation)
	builder.setResourceBundles(spec.ResourceBundles, current.ResourceBundles)
	builder.setFreeformTags(spec.FreeformTags, current.FreeformTags)
	builder.setDefinedTags(spec.DefinedTags, current.DefinedTags)
	return builder.details, builder.updateNeeded, nil
}

func (b *offerUpdateBuilder) setString(spec string, current *string, assign func(*string)) {
	if desired, ok := offerDesiredStringUpdate(spec, current); ok {
		assign(desired)
		b.updateNeeded = true
	}
}

func (b *offerUpdateBuilder) setTime(fieldName string, spec string, current *common.SDKTime, assign func(*common.SDKTime)) error {
	desired, ok, err := offerDesiredTimeUpdate(spec, current)
	if err != nil {
		return fmt.Errorf("parse %s: %w", fieldName, err)
	}
	if ok {
		assign(desired)
		b.updateNeeded = true
	}
	return nil
}

func (b *offerUpdateBuilder) setPricing(spec offerv1beta1.OfferPricing, current *offersdk.Pricing) {
	if desired, ok := offerDesiredPricingUpdate(spec, current); ok {
		b.details.Pricing = desired
		b.updateNeeded = true
	}
}

func (b *offerUpdateBuilder) setBuyerInformation(spec offerv1beta1.OfferBuyerInformation, current *offersdk.BuyerInformation) {
	if desired, ok := offerDesiredBuyerInformationUpdate(spec, current); ok {
		b.details.BuyerInformation = desired
		b.updateNeeded = true
	}
}

func (b *offerUpdateBuilder) setSellerInformation(spec offerv1beta1.OfferSellerInformation, current *offersdk.SellerInformation) {
	if desired, ok := offerDesiredSellerInformationUpdate(spec, current); ok {
		b.details.SellerInformation = desired
		b.updateNeeded = true
	}
}

func (b *offerUpdateBuilder) setResourceBundles(spec []offerv1beta1.OfferResourceBundle, current []offersdk.ResourceBundle) {
	if desired, ok := offerDesiredResourceBundlesUpdate(spec, current); ok {
		b.details.ResourceBundles = desired
		b.updateNeeded = true
	}
}

func (b *offerUpdateBuilder) setFreeformTags(spec map[string]string, current map[string]string) {
	if desired, ok := offerDesiredFreeformTagsUpdate(spec, current); ok {
		b.details.FreeformTags = desired
		b.updateNeeded = true
	}
}

func (b *offerUpdateBuilder) setDefinedTags(spec map[string]shared.MapValue, current map[string]map[string]interface{}) {
	if desired, ok := offerDesiredDefinedTagsUpdate(spec, current); ok {
		b.details.DefinedTags = desired
		b.updateNeeded = true
	}
}

func offerApplyInternalUpdateDetails(
	details *offersdk.UpdateOfferDetails,
	updateNeeded *bool,
	spec offerv1beta1.OfferSpec,
	internalDetails offersdk.OfferInternalDetail,
	hasInternalDetails bool,
) {
	if desired, ok := offerDesiredStringUpdate(spec.InternalNotes, internalDetails.InternalNotes); ok {
		details.InternalNotes = desired
		*updateNeeded = true
	}
	if hasInternalDetails {
		if desired, ok := offerDesiredCustomFieldsUpdate(spec.CustomFields, internalDetails.CustomFields); ok {
			details.CustomFields = desired
			*updateNeeded = true
		}
	}
}

func applyOfferMutableFieldsForCreate(body *offersdk.CreateOfferDetails, spec offerv1beta1.OfferSpec) error {
	builder := offerCreateBodyBuilder{body: body}
	builder.setString(spec.BuyerCompartmentId, func(value *string) { body.BuyerCompartmentId = value })
	builder.setString(spec.Description, func(value *string) { body.Description = value })
	builder.setString(spec.InternalNotes, func(value *string) { body.InternalNotes = value })
	if err := builder.setTime("timeStartDate", spec.TimeStartDate, func(value *common.SDKTime) { body.TimeStartDate = value }); err != nil {
		return err
	}
	builder.setString(spec.Duration, func(value *string) { body.Duration = value })
	if err := builder.setTime("timeAcceptBy", spec.TimeAcceptBy, func(value *common.SDKTime) { body.TimeAcceptBy = value }); err != nil {
		return err
	}
	builder.setPricing(spec.Pricing)
	builder.setBuyerInformation(spec.BuyerInformation)
	builder.setSellerInformation(spec.SellerInformation)
	builder.setResourceBundles(spec.ResourceBundles)
	builder.setCustomFields(spec.CustomFields)
	builder.setFreeformTags(spec.FreeformTags)
	builder.setDefinedTags(spec.DefinedTags)
	return nil
}

type offerCreateBodyBuilder struct {
	body *offersdk.CreateOfferDetails
}

func (b offerCreateBodyBuilder) setString(spec string, assign func(*string)) {
	if value := offerOptionalString(spec); value != nil {
		assign(value)
	}
}

func (b offerCreateBodyBuilder) setTime(fieldName string, spec string, assign func(*common.SDKTime)) error {
	value, err := offerOptionalSDKTime(spec)
	if err != nil {
		return fmt.Errorf("parse %s: %w", fieldName, err)
	}
	if value != nil {
		assign(value)
	}
	return nil
}

func (b offerCreateBodyBuilder) setPricing(spec offerv1beta1.OfferPricing) {
	if value := offerPricing(spec); value != nil {
		b.body.Pricing = value
	}
}

func (b offerCreateBodyBuilder) setBuyerInformation(spec offerv1beta1.OfferBuyerInformation) {
	if value := offerBuyerInformation(spec); value != nil {
		b.body.BuyerInformation = value
	}
}

func (b offerCreateBodyBuilder) setSellerInformation(spec offerv1beta1.OfferSellerInformation) {
	if value := offerSellerInformation(spec); value != nil {
		b.body.SellerInformation = value
	}
}

func (b offerCreateBodyBuilder) setResourceBundles(spec []offerv1beta1.OfferResourceBundle) {
	if value := offerResourceBundles(spec); value != nil {
		b.body.ResourceBundles = value
	}
}

func (b offerCreateBodyBuilder) setCustomFields(spec []offerv1beta1.OfferCustomField) {
	if value := offerCustomFields(spec); value != nil {
		b.body.CustomFields = value
	}
}

func (b offerCreateBodyBuilder) setFreeformTags(spec map[string]string) {
	if spec != nil {
		b.body.FreeformTags = maps.Clone(spec)
	}
}

func (b offerCreateBodyBuilder) setDefinedTags(spec map[string]shared.MapValue) {
	if spec != nil {
		b.body.DefinedTags = offerDefinedTagsFromSpec(spec)
	}
}

func validateOfferSpec(spec offerv1beta1.OfferSpec) error {
	var missing []string
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.SellerCompartmentId) == "" {
		missing = append(missing, "sellerCompartmentId")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("offer spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func validateOfferCreateOnlyDrift(resource *offerv1beta1.Offer, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("offer resource is nil")
	}
	current, err := offerRuntimeBody(currentResponse)
	if err != nil {
		return err
	}
	return validateOfferCreateOnlyDriftForCurrent(resource.Spec, current)
}

func validateOfferCreateOnlyDriftForCurrent(spec offerv1beta1.OfferSpec, current offersdk.Offer) error {
	if spec.SellerCompartmentId != "" && offerStringValue(current.SellerCompartmentId) != "" &&
		spec.SellerCompartmentId != offerStringValue(current.SellerCompartmentId) {
		return fmt.Errorf("offer create-only field drift is not supported: sellerCompartmentId")
	}
	return nil
}

func offerInternalDetailsForUpdate(
	ctx context.Context,
	resource *offerv1beta1.Offer,
	current offersdk.Offer,
	client offerOCIClient,
	initErr error,
) (offersdk.OfferInternalDetail, bool, error) {
	if !offerSpecNeedsInternalRead(resource.Spec) {
		return offersdk.OfferInternalDetail{}, false, nil
	}
	offerID := offerStringValue(current.Id)
	if offerID == "" {
		offerID = currentOfferID(resource)
	}
	if offerID == "" {
		return offersdk.OfferInternalDetail{}, false, fmt.Errorf("offer internal detail read requires a resource OCID")
	}
	if err := requireOfferOCIClient(client, initErr); err != nil {
		return offersdk.OfferInternalDetail{}, false, err
	}
	response, err := client.GetOfferInternalDetail(ctx, offersdk.GetOfferInternalDetailRequest{
		OfferId: offerString(offerID),
	})
	if err != nil {
		return offersdk.OfferInternalDetail{}, false, err
	}
	return response.OfferInternalDetail, true, nil
}

func offerSpecNeedsInternalRead(spec offerv1beta1.OfferSpec) bool {
	return strings.TrimSpace(spec.InternalNotes) != "" || spec.CustomFields != nil
}

func offerStatusFromResponse(resource *offerv1beta1.Offer, response any) error {
	if resource == nil {
		return fmt.Errorf("offer resource is nil")
	}
	current, err := offerRuntimeBody(response)
	if err != nil {
		return nil
	}
	projectOfferStatus(resource, current)
	return nil
}

func projectOfferStatus(resource *offerv1beta1.Offer, current offersdk.Offer) {
	osokStatus := resource.Status.OsokStatus
	resource.Status = offerv1beta1.OfferStatus{
		OsokStatus:          osokStatus,
		Id:                  offerStringValue(current.Id),
		DisplayName:         offerStringValue(current.DisplayName),
		SellerCompartmentId: offerStringValue(current.SellerCompartmentId),
		TimeCreated:         offerSDKTimeString(current.TimeCreated),
		LifecycleState:      string(current.LifecycleState),
		FreeformTags:        maps.Clone(current.FreeformTags),
		DefinedTags:         offerStatusDefinedTagsFromSDK(current.DefinedTags),
		BuyerCompartmentId:  offerStringValue(current.BuyerCompartmentId),
		Description:         offerStringValue(current.Description),
		TimeStartDate:       offerSDKTimeString(current.TimeStartDate),
		Duration:            offerStringValue(current.Duration),
		TimeUpdated:         offerSDKTimeString(current.TimeUpdated),
		TimeAcceptBy:        offerSDKTimeString(current.TimeAcceptBy),
		TimeAccepted:        offerSDKTimeString(current.TimeAccepted),
		TimeOfferEnd:        offerSDKTimeString(current.TimeOfferEnd),
		LifecycleDetails:    offerStringValue(current.LifecycleDetails),
		OfferStatus:         string(current.OfferStatus),
		PublisherSummary:    offerStatusPublisherSummary(current.PublisherSummary),
		Pricing:             offerStatusPricing(current.Pricing),
		BuyerInformation:    offerStatusBuyerInformation(current.BuyerInformation),
		SellerInformation:   offerStatusSellerInformation(current.SellerInformation),
		ResourceBundles:     offerStatusResourceBundles(current.ResourceBundles),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func clearTrackedOfferIdentity(resource *offerv1beta1.Offer) {
	if resource == nil {
		return
	}
	resource.Status = offerv1beta1.OfferStatus{}
}

func confirmOfferDeleteRead(client offerOCIClient, initErr error) func(context.Context, *offerv1beta1.Offer, string) (any, error) {
	return func(ctx context.Context, resource *offerv1beta1.Offer, currentID string) (any, error) {
		offerID := strings.TrimSpace(currentID)
		if offerID == "" {
			offerID = currentOfferID(resource)
		}
		if offerID == "" {
			return confirmOfferDeleteReadByList(ctx, client, initErr, resource)
		}
		return confirmOfferDeleteReadByID(ctx, client, initErr, offerID)
	}
}

func confirmOfferDeleteReadByID(
	ctx context.Context,
	client offerOCIClient,
	initErr error,
	offerID string,
) (any, error) {
	if err := requireOfferOCIClient(client, initErr); err != nil {
		return nil, err
	}
	response, err := client.GetOffer(ctx, offersdk.GetOfferRequest{OfferId: offerString(offerID)})
	if err == nil {
		return response, nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil, err
	}
	return ambiguousOfferNotFound("delete confirmation", err), nil
}

type offerDeleteIdentity struct {
	sellerCompartmentID string
	displayName         string
	buyerCompartmentID  string
}

type offerDeleteFallbackClient struct {
	delegate OfferServiceClient
	client   offerOCIClient
	initErr  error
}

func (c offerDeleteFallbackClient) CreateOrUpdate(
	ctx context.Context,
	resource *offerv1beta1.Offer,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("offer generated runtime delegate is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c offerDeleteFallbackClient) Delete(ctx context.Context, resource *offerv1beta1.Offer) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("offer generated runtime delegate is not configured")
	}
	if resource == nil || currentOfferID(resource) != "" {
		return c.delegate.Delete(ctx, resource)
	}

	summary, found, err := resolveOfferDeleteSummaryByList(ctx, c.client, c.initErr, resource)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, rejectOfferAuthShapedNotFound(resource, err)
	}
	if !found {
		markOfferDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if offerStringValue(summary.Id) == "" {
		return false, fmt.Errorf("offer delete confirmation could not resolve a resource OCID")
	}

	projectOfferStatus(resource, offerFromSummary(summary))
	return c.delegate.Delete(ctx, resource)
}

func confirmOfferDeleteReadByList(
	ctx context.Context,
	client offerOCIClient,
	initErr error,
	resource *offerv1beta1.Offer,
) (any, error) {
	summary, found, err := resolveOfferDeleteSummaryByList(ctx, client, initErr, resource)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errorutil.NotFoundOciError{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			Description:    "Offer delete confirmation did not find a matching OCI offer",
		}
	}
	return summary, nil
}

func resolveOfferDeleteSummaryByList(
	ctx context.Context,
	client offerOCIClient,
	initErr error,
	resource *offerv1beta1.Offer,
) (offersdk.OfferSummary, bool, error) {
	identity, err := offerDeleteIdentityForList(resource)
	if err != nil {
		return offersdk.OfferSummary{}, false, err
	}
	response, err := listOfferPages(ctx, client, initErr, offersdk.ListOffersRequest{
		SellerCompartmentId: offerString(identity.sellerCompartmentID),
		DisplayName:         offerString(identity.displayName),
	})
	if err != nil {
		return offersdk.OfferSummary{}, false, err
	}

	matches := make([]offersdk.OfferSummary, 0, 1)
	for _, item := range response.Items {
		if offerSummaryMatchesDeleteIdentity(item, identity) {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 0:
		return offersdk.OfferSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return offersdk.OfferSummary{}, false, fmt.Errorf(
			"offer delete confirmation found multiple matches for sellerCompartmentId %q, displayName %q, buyerCompartmentId %q",
			identity.sellerCompartmentID,
			identity.displayName,
			identity.buyerCompartmentID,
		)
	}
}

func offerDeleteIdentityForList(resource *offerv1beta1.Offer) (offerDeleteIdentity, error) {
	if resource == nil {
		return offerDeleteIdentity{}, fmt.Errorf("offer delete confirmation requires a resource")
	}
	identity := offerDeleteIdentity{
		sellerCompartmentID: strings.TrimSpace(resource.Status.SellerCompartmentId),
		displayName:         strings.TrimSpace(resource.Status.DisplayName),
		buyerCompartmentID:  strings.TrimSpace(resource.Status.BuyerCompartmentId),
	}
	if identity.sellerCompartmentID == "" {
		identity.sellerCompartmentID = strings.TrimSpace(resource.Spec.SellerCompartmentId)
	}
	if identity.displayName == "" {
		identity.displayName = strings.TrimSpace(resource.Spec.DisplayName)
	}
	if identity.buyerCompartmentID == "" {
		identity.buyerCompartmentID = strings.TrimSpace(resource.Spec.BuyerCompartmentId)
	}
	var missing []string
	if identity.sellerCompartmentID == "" {
		missing = append(missing, "sellerCompartmentId")
	}
	if identity.displayName == "" {
		missing = append(missing, "displayName")
	}
	if len(missing) != 0 {
		return offerDeleteIdentity{}, fmt.Errorf("offer delete confirmation missing identity field(s): %s", strings.Join(missing, ", "))
	}
	return identity, nil
}

func offerSummaryMatchesDeleteIdentity(summary offersdk.OfferSummary, identity offerDeleteIdentity) bool {
	if offerStringValue(summary.SellerCompartmentId) != identity.sellerCompartmentID {
		return false
	}
	if offerStringValue(summary.DisplayName) != identity.displayName {
		return false
	}
	if identity.buyerCompartmentID == "" {
		return true
	}
	return offerStringValue(summary.BuyerCompartmentId) == identity.buyerCompartmentID
}

func markOfferDeleted(resource *offerv1beta1.Offer, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func applyOfferDeleteOutcome(resource *offerv1beta1.Offer, response any, stage generatedruntime.DeleteConfirmStage) (generatedruntime.DeleteOutcome, error) {
	if ambiguous, ok := response.(ambiguousOfferNotFoundError); ok {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		}
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
	}

	current, err := offerRuntimeBody(response)
	if err != nil {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage != generatedruntime.DeleteConfirmStageAfterRequest {
		return generatedruntime.DeleteOutcome{}, nil
	}
	switch current.LifecycleState {
	case offersdk.OfferLifecycleStateCreating,
		offersdk.OfferLifecycleStateUpdating,
		offersdk.OfferLifecycleStateActive:
		markOfferTerminating(resource, string(current.LifecycleState), "OCI resource delete is in progress")
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
}

func markOfferTerminating(resource *offerv1beta1.Offer, rawState string, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       rawState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func rejectOfferAuthShapedNotFound(resource *offerv1beta1.Offer, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return ambiguousOfferNotFound("delete path", err)
}

func ambiguousOfferNotFound(operation string, err error) ambiguousOfferNotFoundError {
	message := fmt.Sprintf("offer %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %s", operation, err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousOfferNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousOfferNotFoundError{message: message}
}

func offerRuntimeBody(currentResponse any) (offersdk.Offer, error) {
	switch current := currentResponse.(type) {
	case offersdk.Offer:
		return current, nil
	case *offersdk.Offer:
		return offerDereferenceRuntimeBody(current)
	case offersdk.OfferSummary:
		return offerFromSummary(current), nil
	case *offersdk.OfferSummary:
		summary, err := offerDereferenceRuntimeBody(current)
		if err != nil {
			return offersdk.Offer{}, err
		}
		return offerFromSummary(summary), nil
	default:
		return offerRuntimeResponseBody(currentResponse)
	}
}

func offerRuntimeResponseBody(currentResponse any) (offersdk.Offer, error) {
	switch current := currentResponse.(type) {
	case offersdk.CreateOfferResponse:
		return current.Offer, nil
	case *offersdk.CreateOfferResponse:
		response, err := offerDereferenceRuntimeBody(current)
		if err != nil {
			return offersdk.Offer{}, err
		}
		return response.Offer, nil
	case offersdk.GetOfferResponse:
		return current.Offer, nil
	case *offersdk.GetOfferResponse:
		response, err := offerDereferenceRuntimeBody(current)
		if err != nil {
			return offersdk.Offer{}, err
		}
		return response.Offer, nil
	case offersdk.UpdateOfferResponse:
		return current.Offer, nil
	case *offersdk.UpdateOfferResponse:
		response, err := offerDereferenceRuntimeBody(current)
		if err != nil {
			return offersdk.Offer{}, err
		}
		return response.Offer, nil
	default:
		return offersdk.Offer{}, fmt.Errorf("unexpected current offer response type %T", currentResponse)
	}
}

func offerDereferenceRuntimeBody[T any](current *T) (T, error) {
	if current == nil {
		var zero T
		return zero, fmt.Errorf("current Offer response is nil")
	}
	return *current, nil
}

func offerFromSummary(summary offersdk.OfferSummary) offersdk.Offer {
	return offersdk.Offer{
		Id:                  summary.Id,
		DisplayName:         summary.DisplayName,
		SellerCompartmentId: summary.SellerCompartmentId,
		TimeCreated:         summary.TimeCreated,
		LifecycleState:      summary.LifecycleState,
		FreeformTags:        summary.FreeformTags,
		DefinedTags:         summary.DefinedTags,
		BuyerCompartmentId:  summary.BuyerCompartmentId,
		TimeUpdated:         summary.TimeUpdated,
		TimeAcceptBy:        summary.TimeAcceptBy,
		TimeAccepted:        summary.TimeAccepted,
		TimeStartDate:       summary.TimeStartDate,
		TimeOfferEnd:        summary.TimeOfferEnd,
		LifecycleDetails:    summary.LifecycleDetails,
		OfferStatus:         summary.OfferStatus,
		BuyerInformation:    summary.BuyerInformation,
		SellerInformation:   summary.SellerInformation,
		Pricing:             summary.Pricing,
	}
}

func currentOfferID(resource *offerv1beta1.Offer) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func offerDesiredStringUpdate(spec string, current *string) (*string, bool) {
	if strings.TrimSpace(spec) == "" {
		return nil, false
	}
	if spec == offerStringValue(current) {
		return nil, false
	}
	return common.String(spec), true
}

func offerDesiredTimeUpdate(spec string, current *common.SDKTime) (*common.SDKTime, bool, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, false, nil
	}
	desired, err := offerOptionalSDKTime(spec)
	if err != nil {
		return nil, false, err
	}
	if current != nil && desired != nil && current.Equal(desired.Time) {
		return nil, false, nil
	}
	return desired, true, nil
}

func offerDesiredPricingUpdate(spec offerv1beta1.OfferPricing, current *offersdk.Pricing) (*offersdk.Pricing, bool) {
	desired := offerPricing(spec)
	if desired == nil || reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func offerDesiredBuyerInformationUpdate(spec offerv1beta1.OfferBuyerInformation, current *offersdk.BuyerInformation) (*offersdk.BuyerInformation, bool) {
	desired := offerBuyerInformation(spec)
	if desired == nil || reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func offerDesiredSellerInformationUpdate(spec offerv1beta1.OfferSellerInformation, current *offersdk.SellerInformation) (*offersdk.SellerInformation, bool) {
	desired := offerSellerInformation(spec)
	if desired == nil || reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func offerDesiredResourceBundlesUpdate(spec []offerv1beta1.OfferResourceBundle, current []offersdk.ResourceBundle) ([]offersdk.ResourceBundle, bool) {
	if spec == nil {
		return nil, false
	}
	desired := offerResourceBundles(spec)
	if desired == nil {
		desired = []offersdk.ResourceBundle{}
	}
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func offerDesiredCustomFieldsUpdate(spec []offerv1beta1.OfferCustomField, current []offersdk.CustomField) ([]offersdk.CustomField, bool) {
	if spec == nil {
		return nil, false
	}
	desired := offerCustomFields(spec)
	if desired == nil {
		desired = []offersdk.CustomField{}
	}
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func offerDesiredFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if reflect.DeepEqual(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func offerDesiredDefinedTagsUpdate(spec map[string]shared.MapValue, current map[string]map[string]interface{}) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := offerDefinedTagsFromSpec(spec)
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func offerPricing(spec offerv1beta1.OfferPricing) *offersdk.Pricing {
	if strings.TrimSpace(spec.CurrencyType) == "" && spec.TotalAmount == 0 && strings.TrimSpace(spec.BillingCycle) == "" {
		return nil
	}
	pricing := &offersdk.Pricing{}
	if value := offerOptionalString(spec.CurrencyType); value != nil {
		pricing.CurrencyType = value
	}
	if spec.TotalAmount != 0 {
		pricing.TotalAmount = common.Int64(spec.TotalAmount)
	}
	if strings.TrimSpace(spec.BillingCycle) != "" {
		pricing.BillingCycle = offersdk.PricingBillingCycleEnum(spec.BillingCycle)
	}
	return pricing
}

func offerBuyerInformation(spec offerv1beta1.OfferBuyerInformation) *offersdk.BuyerInformation {
	primary := offerBuyerPrimaryContact(spec.PrimaryContact)
	contacts := offerBuyerAdditionalContacts(spec.AdditionalContacts)
	if strings.TrimSpace(spec.CompanyName) == "" && strings.TrimSpace(spec.NoteToBuyer) == "" && primary == nil && contacts == nil {
		return nil
	}
	info := &offersdk.BuyerInformation{
		PrimaryContact:     primary,
		AdditionalContacts: contacts,
	}
	if value := offerOptionalString(spec.CompanyName); value != nil {
		info.CompanyName = value
	}
	if value := offerOptionalString(spec.NoteToBuyer); value != nil {
		info.NoteToBuyer = value
	}
	return info
}

func offerSellerInformation(spec offerv1beta1.OfferSellerInformation) *offersdk.SellerInformation {
	primary := offerSellerPrimaryContact(spec.PrimaryContact)
	contacts := offerSellerAdditionalContacts(spec.AdditionalContacts)
	if primary == nil && contacts == nil {
		return nil
	}
	return &offersdk.SellerInformation{
		PrimaryContact:     primary,
		AdditionalContacts: contacts,
	}
}

func offerBuyerPrimaryContact(spec offerv1beta1.OfferBuyerInformationPrimaryContact) *offersdk.Contact {
	return offerContact(spec.FirstName, spec.LastName, spec.Email)
}

func offerBuyerAdditionalContacts(spec []offerv1beta1.OfferBuyerInformationAdditionalContact) []offersdk.Contact {
	if spec == nil {
		return nil
	}
	contacts := make([]offersdk.Contact, 0, len(spec))
	for _, item := range spec {
		if contact := offerContact(item.FirstName, item.LastName, item.Email); contact != nil {
			contacts = append(contacts, *contact)
		}
	}
	return contacts
}

func offerSellerPrimaryContact(spec offerv1beta1.OfferSellerInformationPrimaryContact) *offersdk.Contact {
	return offerContact(spec.FirstName, spec.LastName, spec.Email)
}

func offerSellerAdditionalContacts(spec []offerv1beta1.OfferSellerInformationAdditionalContact) []offersdk.Contact {
	if spec == nil {
		return nil
	}
	contacts := make([]offersdk.Contact, 0, len(spec))
	for _, item := range spec {
		if contact := offerContact(item.FirstName, item.LastName, item.Email); contact != nil {
			contacts = append(contacts, *contact)
		}
	}
	return contacts
}

func offerContact(firstName string, lastName string, email string) *offersdk.Contact {
	if strings.TrimSpace(firstName) == "" && strings.TrimSpace(lastName) == "" && strings.TrimSpace(email) == "" {
		return nil
	}
	contact := &offersdk.Contact{}
	if value := offerOptionalString(firstName); value != nil {
		contact.FirstName = value
	}
	if value := offerOptionalString(lastName); value != nil {
		contact.LastName = value
	}
	if value := offerOptionalString(email); value != nil {
		contact.Email = value
	}
	return contact
}

func offerResourceBundles(spec []offerv1beta1.OfferResourceBundle) []offersdk.ResourceBundle {
	if spec == nil {
		return nil
	}
	bundles := make([]offersdk.ResourceBundle, 0, len(spec))
	for _, item := range spec {
		if strings.TrimSpace(item.Type) == "" && item.Quantity == 0 && strings.TrimSpace(item.UnitOfMeasurement) == "" && len(item.ResourceIds) == 0 {
			continue
		}
		bundle := offersdk.ResourceBundle{
			ResourceIds: append([]string(nil), item.ResourceIds...),
		}
		if strings.TrimSpace(item.Type) != "" {
			bundle.Type = offersdk.ResourceBundleTypeEnum(item.Type)
		}
		if item.Quantity != 0 {
			bundle.Quantity = common.Int64(item.Quantity)
		}
		if strings.TrimSpace(item.UnitOfMeasurement) != "" {
			bundle.UnitOfMeasurement = offersdk.ResourceBundleUnitOfMeasurementEnum(item.UnitOfMeasurement)
		}
		bundles = append(bundles, bundle)
	}
	return bundles
}

func offerCustomFields(spec []offerv1beta1.OfferCustomField) []offersdk.CustomField {
	if spec == nil {
		return nil
	}
	fields := make([]offersdk.CustomField, 0, len(spec))
	for _, item := range spec {
		if strings.TrimSpace(item.Key) == "" && strings.TrimSpace(item.Value) == "" {
			continue
		}
		field := offersdk.CustomField{}
		if value := offerOptionalString(item.Key); value != nil {
			field.Key = value
		}
		if value := offerOptionalString(item.Value); value != nil {
			field.Value = value
		}
		fields = append(fields, field)
	}
	return fields
}

func offerStatusPricing(current *offersdk.Pricing) offerv1beta1.OfferPricing {
	if current == nil {
		return offerv1beta1.OfferPricing{}
	}
	return offerv1beta1.OfferPricing{
		CurrencyType: offerStringValue(current.CurrencyType),
		TotalAmount:  offerInt64Value(current.TotalAmount),
		BillingCycle: string(current.BillingCycle),
	}
}

func offerStatusBuyerInformation(current *offersdk.BuyerInformation) offerv1beta1.OfferBuyerInformation {
	if current == nil {
		return offerv1beta1.OfferBuyerInformation{}
	}
	return offerv1beta1.OfferBuyerInformation{
		CompanyName:        offerStringValue(current.CompanyName),
		NoteToBuyer:        offerStringValue(current.NoteToBuyer),
		PrimaryContact:     offerStatusBuyerPrimaryContact(current.PrimaryContact),
		AdditionalContacts: offerStatusBuyerAdditionalContacts(current.AdditionalContacts),
	}
}

func offerStatusSellerInformation(current *offersdk.SellerInformation) offerv1beta1.OfferSellerInformation {
	if current == nil {
		return offerv1beta1.OfferSellerInformation{}
	}
	return offerv1beta1.OfferSellerInformation{
		PrimaryContact:     offerStatusSellerPrimaryContact(current.PrimaryContact),
		AdditionalContacts: offerStatusSellerAdditionalContacts(current.AdditionalContacts),
	}
}

func offerStatusBuyerPrimaryContact(current *offersdk.Contact) offerv1beta1.OfferBuyerInformationPrimaryContact {
	firstName, lastName, email := offerStatusContactValues(current)
	return offerv1beta1.OfferBuyerInformationPrimaryContact{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}
}

func offerStatusBuyerAdditionalContacts(current []offersdk.Contact) []offerv1beta1.OfferBuyerInformationAdditionalContact {
	if current == nil {
		return nil
	}
	contacts := make([]offerv1beta1.OfferBuyerInformationAdditionalContact, 0, len(current))
	for _, item := range current {
		firstName, lastName, email := offerStatusContactValues(&item)
		contacts = append(contacts, offerv1beta1.OfferBuyerInformationAdditionalContact{
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		})
	}
	return contacts
}

func offerStatusSellerPrimaryContact(current *offersdk.Contact) offerv1beta1.OfferSellerInformationPrimaryContact {
	firstName, lastName, email := offerStatusContactValues(current)
	return offerv1beta1.OfferSellerInformationPrimaryContact{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}
}

func offerStatusSellerAdditionalContacts(current []offersdk.Contact) []offerv1beta1.OfferSellerInformationAdditionalContact {
	if current == nil {
		return nil
	}
	contacts := make([]offerv1beta1.OfferSellerInformationAdditionalContact, 0, len(current))
	for _, item := range current {
		firstName, lastName, email := offerStatusContactValues(&item)
		contacts = append(contacts, offerv1beta1.OfferSellerInformationAdditionalContact{
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		})
	}
	return contacts
}

func offerStatusContactValues(current *offersdk.Contact) (string, string, string) {
	if current == nil {
		return "", "", ""
	}
	return offerStringValue(current.FirstName), offerStringValue(current.LastName), offerStringValue(current.Email)
}

func offerStatusResourceBundles(current []offersdk.ResourceBundle) []offerv1beta1.OfferResourceBundle {
	if current == nil {
		return nil
	}
	bundles := make([]offerv1beta1.OfferResourceBundle, 0, len(current))
	for _, item := range current {
		bundles = append(bundles, offerv1beta1.OfferResourceBundle{
			Type:              string(item.Type),
			Quantity:          offerInt64Value(item.Quantity),
			UnitOfMeasurement: string(item.UnitOfMeasurement),
			ResourceIds:       append([]string(nil), item.ResourceIds...),
		})
	}
	return bundles
}

func offerStatusPublisherSummary(current *offersdk.PublisherSummary) offerv1beta1.OfferPublisherSummary {
	if current == nil {
		return offerv1beta1.OfferPublisherSummary{}
	}
	return offerv1beta1.OfferPublisherSummary{
		Id:                offerStringValue(current.Id),
		CompartmentId:     offerStringValue(current.CompartmentId),
		RegistryNamespace: offerStringValue(current.RegistryNamespace),
		DisplayName:       offerStringValue(current.DisplayName),
		ContactEmail:      offerStringValue(current.ContactEmail),
		ContactPhone:      offerStringValue(current.ContactPhone),
		PublisherType:     string(current.PublisherType),
		TimeCreated:       offerSDKTimeString(current.TimeCreated),
		TimeUpdated:       offerSDKTimeString(current.TimeUpdated),
		LegacyId:          offerStringValue(current.LegacyId),
		Description:       offerStringValue(current.Description),
		YearFounded:       offerInt64Value(current.YearFounded),
		WebsiteUrl:        offerStringValue(current.WebsiteUrl),
		HqAddress:         offerStringValue(current.HqAddress),
		Logo:              offerStatusPublisherLogo(current.Logo),
		FacebookUrl:       offerStringValue(current.FacebookUrl),
		TwitterUrl:        offerStringValue(current.TwitterUrl),
		LinkedinUrl:       offerStringValue(current.LinkedinUrl),
		FreeformTags:      maps.Clone(current.FreeformTags),
		DefinedTags:       offerStatusDefinedTagsFromSDK(current.DefinedTags),
		SystemTags:        offerStatusDefinedTagsFromSDK(current.SystemTags),
	}
}

func offerStatusPublisherLogo(current *offersdk.UploadData) offerv1beta1.OfferPublisherSummaryLogo {
	if current == nil {
		return offerv1beta1.OfferPublisherSummaryLogo{}
	}
	return offerv1beta1.OfferPublisherSummaryLogo{
		Name:       offerStringValue(current.Name),
		ContentUrl: offerStringValue(current.ContentUrl),
		MimeType:   offerStringValue(current.MimeType),
	}
}

func offerDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func offerStatusDefinedTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(shared.MapValue, len(values))
		for key, value := range values {
			if stringValue, ok := value.(string); ok {
				converted[namespace][key] = stringValue
				continue
			}
			converted[namespace][key] = fmt.Sprint(value)
		}
	}
	return converted
}

func offerOptionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func offerString(value string) *string {
	return common.String(strings.TrimSpace(value))
}

func offerStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func offerInt64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func offerOptionalSDKTime(value string) (*common.SDKTime, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, err
	}
	return &common.SDKTime{Time: parsed}, nil
}

func offerSDKTimeString(value *common.SDKTime) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
