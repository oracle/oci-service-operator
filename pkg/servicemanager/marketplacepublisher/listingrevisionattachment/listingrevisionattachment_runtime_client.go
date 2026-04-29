/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevisionattachment

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
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const listingRevisionAttachmentDeletePendingMessage = "OCI ListingRevisionAttachment delete is in progress"

type listingRevisionAttachmentOCIClient interface {
	CreateListingRevisionAttachment(context.Context, marketplacepublishersdk.CreateListingRevisionAttachmentRequest) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error)
	GetListingRevisionAttachment(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error)
	ListListingRevisionAttachments(context.Context, marketplacepublishersdk.ListListingRevisionAttachmentsRequest) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error)
	UpdateListingRevisionAttachment(context.Context, marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error)
	DeleteListingRevisionAttachment(context.Context, marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error)
}

type createListingRevisionAttachmentRequest struct {
	CreateListingRevisionAttachmentDetails map[string]any `contributesTo:"body"`
	OpcRetryToken                          *string        `mandatory:"false" contributesTo:"header" name:"opc-retry-token"`
	OpcRequestId                           *string        `mandatory:"false" contributesTo:"header" name:"opc-request-id"`
	RequestMetadata                        common.RequestMetadata
}

type updateListingRevisionAttachmentRequest struct {
	ListingRevisionAttachmentId            *string        `mandatory:"true" contributesTo:"path" name:"listingRevisionAttachmentId"`
	UpdateListingRevisionAttachmentDetails map[string]any `contributesTo:"body"`
	IfMatch                                *string        `mandatory:"false" contributesTo:"header" name:"if-match"`
	OpcRequestId                           *string        `mandatory:"false" contributesTo:"header" name:"opc-request-id"`
	RequestMetadata                        common.RequestMetadata
}

type listingRevisionAttachmentIdentity struct {
	ListingRevisionId string
	DisplayName       string
	AttachmentType    string
}

type listingRevisionAttachmentBody struct {
	Id                string                            `json:"id,omitempty"`
	CompartmentId     string                            `json:"compartmentId,omitempty"`
	ListingRevisionId string                            `json:"listingRevisionId,omitempty"`
	DisplayName       string                            `json:"displayName,omitempty"`
	Description       string                            `json:"description,omitempty"`
	FreeformTags      map[string]string                 `json:"freeformTags,omitempty"`
	DefinedTags       map[string]map[string]interface{} `json:"definedTags,omitempty"`
	SystemTags        map[string]map[string]interface{} `json:"systemTags,omitempty"`
	AttachmentType    string                            `json:"attachmentType,omitempty"`
	LifecycleState    string                            `json:"lifecycleState,omitempty"`
	ContentUrl        string                            `json:"contentUrl,omitempty"`
	MimeType          string                            `json:"mimeType,omitempty"`
	DocumentCategory  string                            `json:"documentCategory,omitempty"`
	DocumentName      string                            `json:"documentName,omitempty"`
	TemplateCode      string                            `json:"templateCode,omitempty"`
	ServiceName       string                            `json:"serviceName,omitempty"`
	Url               string                            `json:"url,omitempty"`
	Type              string                            `json:"type,omitempty"`
	CustomerName      string                            `json:"customerName,omitempty"`
	ProductCodes      []string                          `json:"productCodes,omitempty"`
}

type listingRevisionAttachmentAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type listingRevisionAttachmentCreateBodyBuilder func(map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error)
type listingRevisionAttachmentUpdateBodyBuilder func(map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error)
type listingRevisionAttachmentBodyValueFunc func(listingRevisionAttachmentBody) (any, bool)

var listingRevisionAttachmentCreateBodyBuilders = map[string]listingRevisionAttachmentCreateBodyBuilder{
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeRelatedDocument):       createRelatedDocumentAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeScreenshot):            createScreenshotAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeVideo):                 createVideoAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeReviewSupportDocument): createReviewSupportDocumentAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeCustomerSuccess):       createCustomerSuccessAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeSupportedServices):     createSupportedServiceAttachmentBody,
}

var listingRevisionAttachmentUpdateBodyBuilders = map[string]listingRevisionAttachmentUpdateBodyBuilder{
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeRelatedDocument):       updateRelatedDocumentAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeScreenshot):            updateScreenshotAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeVideo):                 updateVideoAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeReviewSupportDocument): updateReviewSupportDocumentAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeCustomerSuccess):       updateCustomerSuccessAttachmentBody,
	string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeSupportedServices):     updateSupportedServiceAttachmentBody,
}

var listingRevisionAttachmentBodyValueFuncs = map[string]listingRevisionAttachmentBodyValueFunc{
	"displayName": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.DisplayName)
	},
	"description": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.Description)
	},
	"freeformTags": func(body listingRevisionAttachmentBody) (any, bool) {
		return body.FreeformTags, body.FreeformTags != nil
	},
	"definedTags": func(body listingRevisionAttachmentBody) (any, bool) {
		return body.DefinedTags, body.DefinedTags != nil
	},
	"documentCategory": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.DocumentCategory)
	},
	"documentName": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.DocumentName)
	},
	"templateCode": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.TemplateCode)
	},
	"serviceName": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.ServiceName)
	},
	"url": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.Url)
	},
	"type": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.Type)
	},
	"customerName": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.CustomerName)
	},
	"productCodes": func(body listingRevisionAttachmentBody) (any, bool) {
		return body.ProductCodes, body.ProductCodes != nil
	},
	"videoAttachmentDetails.contentUrl": func(body listingRevisionAttachmentBody) (any, bool) {
		return nonEmptyStringValue(body.ContentUrl)
	},
}

func (e listingRevisionAttachmentAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e listingRevisionAttachmentAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerListingRevisionAttachmentRuntimeHooksMutator(func(manager *ListingRevisionAttachmentServiceManager, hooks *ListingRevisionAttachmentRuntimeHooks) {
		applyListingRevisionAttachmentRuntimeHooksForManager(manager, hooks)
	})
}

func applyListingRevisionAttachmentRuntimeHooks(hooks *ListingRevisionAttachmentRuntimeHooks) {
	applyListingRevisionAttachmentRuntimeHookSettings(hooks)
	wrapListingRevisionAttachmentDeleteConfirmation(hooks)
}

func applyListingRevisionAttachmentRuntimeHooksForManager(
	manager *ListingRevisionAttachmentServiceManager,
	hooks *ListingRevisionAttachmentRuntimeHooks,
) {
	applyListingRevisionAttachmentRuntimeHookSettings(hooks)
	appendListingRevisionAttachmentConcreteClientWrapper(manager, hooks)
	wrapListingRevisionAttachmentDeleteConfirmation(hooks)
}

func applyListingRevisionAttachmentRuntimeHookSettings(hooks *ListingRevisionAttachmentRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = listingRevisionAttachmentRuntimeSemantics()
	hooks.BuildCreateBody = buildListingRevisionAttachmentCreateBody
	hooks.BuildUpdateBody = buildListingRevisionAttachmentUpdateBody
	hooks.Identity.Resolve = resolveListingRevisionAttachmentIdentity
	hooks.Identity.RecordPath = recordListingRevisionAttachmentIdentity
	hooks.List.Fields = listingRevisionAttachmentListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listListingRevisionAttachmentsAllPages(hooks.List.Call)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateListingRevisionAttachmentCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleListingRevisionAttachmentDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyListingRevisionAttachmentDeleteOutcome
	hooks.StatusHooks.MarkTerminating = markListingRevisionAttachmentTerminating
}

func appendListingRevisionAttachmentConcreteClientWrapper(
	manager *ListingRevisionAttachmentServiceManager,
	hooks *ListingRevisionAttachmentRuntimeHooks,
) {
	if manager == nil || hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(ListingRevisionAttachmentServiceClient) ListingRevisionAttachmentServiceClient {
		sdkClient, err := marketplacepublishersdk.NewMarketplacePublisherClientWithConfigurationProvider(manager.Provider)
		customHooks := newListingRevisionAttachmentDefaultRuntimeHooks(sdkClient)
		applyListingRevisionAttachmentRuntimeHookSettings(&customHooks)
		config := buildListingRevisionAttachmentConcreteGeneratedRuntimeConfig(manager, customHooks)
		if err != nil {
			config.InitError = fmt.Errorf("initialize ListingRevisionAttachment OCI client: %w", err)
		}
		return defaultListingRevisionAttachmentServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.ListingRevisionAttachment](config),
		}
	})
}

func newListingRevisionAttachmentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client listingRevisionAttachmentOCIClient,
) ListingRevisionAttachmentServiceClient {
	manager := &ListingRevisionAttachmentServiceManager{Log: log}
	hooks := newListingRevisionAttachmentRuntimeHooksWithOCIClient(client)
	applyListingRevisionAttachmentRuntimeHookSettings(&hooks)
	delegate := defaultListingRevisionAttachmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.ListingRevisionAttachment](
			buildListingRevisionAttachmentConcreteGeneratedRuntimeConfig(manager, hooks),
		),
	}
	wrapListingRevisionAttachmentDeleteConfirmation(&hooks)
	return wrapListingRevisionAttachmentGeneratedClient(hooks, delegate)
}

func buildListingRevisionAttachmentConcreteGeneratedRuntimeConfig(
	manager *ListingRevisionAttachmentServiceManager,
	hooks ListingRevisionAttachmentRuntimeHooks,
) generatedruntime.Config[*marketplacepublisherv1beta1.ListingRevisionAttachment] {
	config := buildListingRevisionAttachmentGeneratedRuntimeConfig(manager, hooks)
	config.Create = &generatedruntime.Operation{
		NewRequest: func() any { return &createListingRevisionAttachmentRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), hooks.Create.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			typed := request.(*createListingRevisionAttachmentRequest)
			body, err := createListingRevisionAttachmentBody(typed.CreateListingRevisionAttachmentDetails)
			if err != nil {
				return nil, err
			}
			return hooks.Create.Call(ctx, marketplacepublishersdk.CreateListingRevisionAttachmentRequest{
				CreateListingRevisionAttachmentDetails: body,
				OpcRetryToken:                          typed.OpcRetryToken,
				OpcRequestId:                           typed.OpcRequestId,
				RequestMetadata:                        typed.RequestMetadata,
			})
		},
	}
	config.Update = &generatedruntime.Operation{
		NewRequest: func() any { return &updateListingRevisionAttachmentRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), hooks.Update.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			typed := request.(*updateListingRevisionAttachmentRequest)
			body, err := updateListingRevisionAttachmentBody(typed.UpdateListingRevisionAttachmentDetails)
			if err != nil {
				return nil, err
			}
			return hooks.Update.Call(ctx, marketplacepublishersdk.UpdateListingRevisionAttachmentRequest{
				ListingRevisionAttachmentId:            typed.ListingRevisionAttachmentId,
				UpdateListingRevisionAttachmentDetails: body,
				IfMatch:                                typed.IfMatch,
				OpcRequestId:                           typed.OpcRequestId,
				RequestMetadata:                        typed.RequestMetadata,
			})
		},
	}
	return config
}

func newListingRevisionAttachmentRuntimeHooksWithOCIClient(client listingRevisionAttachmentOCIClient) ListingRevisionAttachmentRuntimeHooks {
	hooks := newListingRevisionAttachmentDefaultRuntimeHooks(marketplacepublishersdk.MarketplacePublisherClient{})
	hooks.Create.Call = func(ctx context.Context, request marketplacepublishersdk.CreateListingRevisionAttachmentRequest) (marketplacepublishersdk.CreateListingRevisionAttachmentResponse, error) {
		if client == nil {
			return marketplacepublishersdk.CreateListingRevisionAttachmentResponse{}, fmt.Errorf("listing revision attachment OCI client is not configured")
		}
		return client.CreateListingRevisionAttachment(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error) {
		if client == nil {
			return marketplacepublishersdk.GetListingRevisionAttachmentResponse{}, fmt.Errorf("listing revision attachment OCI client is not configured")
		}
		return client.GetListingRevisionAttachment(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request marketplacepublishersdk.ListListingRevisionAttachmentsRequest) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error) {
		if client == nil {
			return marketplacepublishersdk.ListListingRevisionAttachmentsResponse{}, fmt.Errorf("listing revision attachment OCI client is not configured")
		}
		return client.ListListingRevisionAttachments(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request marketplacepublishersdk.UpdateListingRevisionAttachmentRequest) (marketplacepublishersdk.UpdateListingRevisionAttachmentResponse, error) {
		if client == nil {
			return marketplacepublishersdk.UpdateListingRevisionAttachmentResponse{}, fmt.Errorf("listing revision attachment OCI client is not configured")
		}
		return client.UpdateListingRevisionAttachment(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request marketplacepublishersdk.DeleteListingRevisionAttachmentRequest) (marketplacepublishersdk.DeleteListingRevisionAttachmentResponse, error) {
		if client == nil {
			return marketplacepublishersdk.DeleteListingRevisionAttachmentResponse{}, fmt.Errorf("listing revision attachment OCI client is not configured")
		}
		return client.DeleteListingRevisionAttachment(ctx, request)
	}
	return hooks
}

func listingRevisionAttachmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplacepublisher",
		FormalSlug:    "listingrevisionattachment",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateActive),
				string(marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"listingRevisionId", "displayName", "attachmentType", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
				"documentCategory",
				"documentName",
				"templateCode",
				"serviceName",
				"url",
				"type",
				"customerName",
				"productCodes",
			},
			ForceNew: []string{
				"listingRevisionId",
				"attachmentType",
				"videoAttachmentDetails.contentUrl",
				"jsonData",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ListingRevisionAttachment", Action: "CreateListingRevisionAttachment"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ListingRevisionAttachment", Action: "UpdateListingRevisionAttachment"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ListingRevisionAttachment", Action: "DeleteListingRevisionAttachment"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ListingRevisionAttachment", Action: "GetListingRevisionAttachment"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ListingRevisionAttachment", Action: "GetListingRevisionAttachment"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ListingRevisionAttachment", Action: "GetListingRevisionAttachment"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func listingRevisionAttachmentListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ListingRevisionId",
			RequestName:  "listingRevisionId",
			Contribution: "query",
			LookupPaths:  []string{"status.listingRevisionId", "spec.listingRevisionId", "listingRevisionId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "metadataName", "displayName"},
		},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "compartmentId"},
		},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func resolveListingRevisionAttachmentIdentity(resource *marketplacepublisherv1beta1.ListingRevisionAttachment) (any, error) {
	payload, err := listingRevisionAttachmentPayload(resource, true)
	if err != nil {
		return nil, err
	}
	allowStatusFallback := trackedListingRevisionAttachmentID(resource) != ""
	listingRevisionID, err := listingRevisionAttachmentIdentityString(
		payload,
		"listingRevisionId",
		resource.Status.ListingRevisionId,
		allowStatusFallback,
		"listing revision attachment spec is missing required field listingRevisionId",
	)
	if err != nil {
		return nil, err
	}
	attachmentType, err := listingRevisionAttachmentIdentityAttachmentType(payload, resource.Status.AttachmentType, allowStatusFallback)
	if err != nil {
		return nil, err
	}
	displayName, err := listingRevisionAttachmentIdentityString(
		payload,
		"displayName",
		resource.Status.DisplayName,
		allowStatusFallback,
		"listing revision attachment identity requires displayName or metadata.name",
	)
	if err != nil {
		return nil, err
	}
	return listingRevisionAttachmentIdentity{
		ListingRevisionId: strings.TrimSpace(listingRevisionID),
		DisplayName:       strings.TrimSpace(displayName),
		AttachmentType:    attachmentType,
	}, nil
}

func recordListingRevisionAttachmentIdentity(
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	identity any,
) {
	if resource == nil {
		return
	}
	typed, ok := identity.(listingRevisionAttachmentIdentity)
	if !ok {
		return
	}
	if typed.ListingRevisionId != "" {
		resource.Status.ListingRevisionId = typed.ListingRevisionId
	}
	if typed.DisplayName != "" {
		resource.Status.DisplayName = typed.DisplayName
	}
	if typed.AttachmentType != "" {
		resource.Status.AttachmentType = typed.AttachmentType
	}
}

func listingRevisionAttachmentIdentityString(
	payload map[string]any,
	field string,
	statusValue string,
	allowEmpty bool,
	missingMessage string,
) (string, error) {
	if value, ok := payloadString(payload, field); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value), nil
	}
	if value := strings.TrimSpace(statusValue); value != "" {
		return value, nil
	}
	if allowEmpty {
		return "", nil
	}
	return "", errors.New(missingMessage)
}

func listingRevisionAttachmentIdentityAttachmentType(
	payload map[string]any,
	statusValue string,
	allowEmpty bool,
) (string, error) {
	if value, ok := payloadString(payload, "attachmentType"); ok && strings.TrimSpace(value) != "" {
		return normalizeListingRevisionAttachmentType(value)
	}
	if value := strings.TrimSpace(statusValue); value != "" {
		return normalizeListingRevisionAttachmentType(value)
	}
	if allowEmpty {
		return "", nil
	}
	return "", errors.New("listing revision attachment spec is missing required field attachmentType")
}

func buildListingRevisionAttachmentCreateBody(
	_ context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	_ string,
) (any, error) {
	payload, err := listingRevisionAttachmentPayload(resource, true)
	if err != nil {
		return nil, err
	}
	return createListingRevisionAttachmentBody(payload)
}

func buildListingRevisionAttachmentUpdateBody(
	_ context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	_ string,
	currentResponse any,
) (any, bool, error) {
	current, ok := listingRevisionAttachmentBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current ListingRevisionAttachment response does not expose a body")
	}
	payload, err := listingRevisionAttachmentPayload(resource, false)
	if err != nil {
		return nil, false, err
	}
	if err := completeListingRevisionAttachmentUpdatePayload(payload, current); err != nil {
		return nil, false, err
	}
	body, err := updateListingRevisionAttachmentBody(payload)
	if err != nil {
		return nil, false, err
	}
	return body, listingRevisionAttachmentUpdateNeeded(payload, current), nil
}

func listingRevisionAttachmentPayload(
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	defaultDisplayName bool,
) (map[string]any, error) {
	if resource == nil {
		return nil, fmt.Errorf("listing revision attachment resource is nil")
	}
	payload, err := baseListingRevisionAttachmentPayload(resource.Spec.JsonData)
	if err != nil {
		return nil, err
	}
	if err := overlayListingRevisionAttachmentSpec(payload, resource, defaultDisplayName); err != nil {
		return nil, err
	}
	return normalizeListingRevisionAttachmentPayload(payload)
}

func baseListingRevisionAttachmentPayload(rawJSON string) (map[string]any, error) {
	payload := map[string]any{}
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON == "" {
		return payload, nil
	}
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return nil, fmt.Errorf("decode ListingRevisionAttachment jsonData: %w", err)
	}
	return payload, nil
}

func overlayListingRevisionAttachmentSpec(
	payload map[string]any,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	defaultDisplayName bool,
) error {
	if err := overlayListingRevisionAttachmentIdentityFields(payload, resource, defaultDisplayName); err != nil {
		return err
	}
	overlayListingRevisionAttachmentCommonFields(payload, resource.Spec)
	overlayListingRevisionAttachmentTypedFields(payload, resource.Spec)
	return nil
}

func overlayListingRevisionAttachmentIdentityFields(
	payload map[string]any,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	defaultDisplayName bool,
) error {
	spec := resource.Spec
	if err := setPayloadIdentityString(payload, "listingRevisionId", spec.ListingRevisionId); err != nil {
		return err
	}
	if err := overlayListingRevisionAttachmentType(payload, spec.AttachmentType); err != nil {
		return err
	}
	return setPayloadIdentityString(payload, "displayName", listingRevisionAttachmentDisplayName(payload, resource, defaultDisplayName))
}

func overlayListingRevisionAttachmentType(payload map[string]any, attachmentType string) error {
	if strings.TrimSpace(attachmentType) == "" {
		return nil
	}
	normalized, err := normalizeListingRevisionAttachmentType(attachmentType)
	if err != nil {
		return err
	}
	return setPayloadIdentityString(payload, "attachmentType", normalized)
}

func listingRevisionAttachmentDisplayName(
	payload map[string]any,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	defaultDisplayName bool,
) string {
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if displayName != "" || !defaultDisplayName {
		return displayName
	}
	if current, ok := payloadString(payload, "displayName"); ok && strings.TrimSpace(current) != "" {
		return displayName
	}
	return strings.TrimSpace(resource.Name)
}

func overlayListingRevisionAttachmentCommonFields(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ListingRevisionAttachmentSpec,
) {
	setPayloadString(payload, "description", spec.Description)
	if spec.FreeformTags != nil {
		payload["freeformTags"] = cloneListingRevisionAttachmentStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		payload["definedTags"] = listingRevisionAttachmentDefinedTags(spec.DefinedTags)
	}
}

func overlayListingRevisionAttachmentTypedFields(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ListingRevisionAttachmentSpec,
) {
	setPayloadString(payload, "serviceName", spec.ServiceName)
	setPayloadString(payload, "url", spec.Url)
	setPayloadString(payload, "type", spec.Type)
	setPayloadString(payload, "customerName", spec.CustomerName)
	if spec.ProductCodes != nil {
		payload["productCodes"] = append([]string(nil), spec.ProductCodes...)
	}
	setPayloadString(payload, "documentName", spec.DocumentName)
	setPayloadString(payload, "templateCode", spec.TemplateCode)
	setPayloadString(payload, "documentCategory", spec.DocumentCategory)
	if contentURL := strings.TrimSpace(spec.VideoAttachmentDetails.ContentUrl); contentURL != "" {
		setNestedPayloadString(payload, []string{"videoAttachmentDetails", "contentUrl"}, contentURL)
	}
}

func normalizeListingRevisionAttachmentPayload(payload map[string]any) (map[string]any, error) {
	if value, ok := payloadString(payload, "attachmentType"); ok && strings.TrimSpace(value) != "" {
		attachmentType, err := normalizeListingRevisionAttachmentType(value)
		if err != nil {
			return nil, err
		}
		payload["attachmentType"] = attachmentType
	}
	if value, ok := payloadString(payload, "documentCategory"); ok && strings.TrimSpace(value) != "" {
		category, err := normalizeRelatedDocumentCategory(value)
		if err != nil {
			return nil, err
		}
		payload["documentCategory"] = category
	}
	if value, ok := payloadString(payload, "type"); ok && strings.TrimSpace(value) != "" {
		serviceType, err := normalizeSupportedServiceType(value)
		if err != nil {
			return nil, err
		}
		payload["type"] = serviceType
	}
	return payload, nil
}

func createListingRevisionAttachmentBody(payload map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "listingRevisionId"); err != nil {
		return nil, err
	}
	attachmentType, err := payloadAttachmentType(payload)
	if err != nil {
		return nil, err
	}
	builder, ok := listingRevisionAttachmentCreateBodyBuilders[attachmentType]
	if !ok {
		return nil, fmt.Errorf("unsupported ListingRevisionAttachment attachmentType %q", attachmentType)
	}
	return builder(payload)
}

func updateListingRevisionAttachmentBody(payload map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error) {
	attachmentType, err := payloadAttachmentType(payload)
	if err != nil {
		return nil, err
	}
	builder, ok := listingRevisionAttachmentUpdateBodyBuilders[attachmentType]
	if !ok {
		return nil, fmt.Errorf("unsupported ListingRevisionAttachment attachmentType %q", attachmentType)
	}
	return builder(payload)
}

func createRelatedDocumentAttachmentBody(payload map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "documentCategory"); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.CreateRelatedDocumentAttachmentDetails
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func createScreenshotAttachmentBody(payload map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error) {
	var body marketplacepublishersdk.CreateScreenShotAttachmentDetails
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func createVideoAttachmentBody(payload map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error) {
	if err := requireNestedPayloadString(payload, []string{"videoAttachmentDetails", "contentUrl"}); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.CreateVideoAttachmentDetails
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func createReviewSupportDocumentAttachmentBody(payload map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "documentName"); err != nil {
		return nil, err
	}
	if err := requirePayloadString(payload, "templateCode"); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.CreateReviewSupportDocumentAttachment
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func createCustomerSuccessAttachmentBody(payload map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "customerName"); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.CreateCustomerSuccessAttachment
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func createSupportedServiceAttachmentBody(payload map[string]any) (marketplacepublishersdk.CreateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "serviceName"); err != nil {
		return nil, err
	}
	if err := requirePayloadString(payload, "type"); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.CreateSupportedServiceAttachment
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func updateRelatedDocumentAttachmentBody(payload map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error) {
	var body marketplacepublishersdk.UpdateRelatedDocumentAttachmentDetails
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func updateScreenshotAttachmentBody(payload map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error) {
	var body marketplacepublishersdk.UpdateScreenShotAttachmentDetails
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func updateVideoAttachmentBody(payload map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error) {
	var body marketplacepublishersdk.UpdateVideoAttachmentDetails
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func updateReviewSupportDocumentAttachmentBody(payload map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "documentName"); err != nil {
		return nil, err
	}
	if err := requirePayloadString(payload, "templateCode"); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.UpdateReviewSupportDocumentAttachment
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func updateCustomerSuccessAttachmentBody(payload map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "customerName"); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.UpdateCustomerSuccessAttachment
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func updateSupportedServiceAttachmentBody(payload map[string]any) (marketplacepublishersdk.UpdateListingRevisionAttachmentDetails, error) {
	if err := requirePayloadString(payload, "serviceName"); err != nil {
		return nil, err
	}
	if err := requirePayloadString(payload, "type"); err != nil {
		return nil, err
	}
	var body marketplacepublishersdk.UpdateSupportedServiceAttachment
	return body, decodeListingRevisionAttachmentPayload(payload, &body)
}

func decodeListingRevisionAttachmentPayload(payload map[string]any, body any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ListingRevisionAttachment payload: %w", err)
	}
	if err := json.Unmarshal(encoded, body); err != nil {
		return fmt.Errorf("decode ListingRevisionAttachment payload: %w", err)
	}
	if validator, ok := body.(interface {
		ValidateEnumValue() (bool, error)
	}); ok {
		if _, err := validator.ValidateEnumValue(); err != nil {
			return err
		}
	}
	return nil
}

func completeListingRevisionAttachmentUpdatePayload(
	payload map[string]any,
	current listingRevisionAttachmentBody,
) error {
	attachmentType, err := payloadAttachmentTypeWithFallback(payload, current.AttachmentType)
	if err != nil {
		return err
	}
	if current.AttachmentType != "" && attachmentType != current.AttachmentType {
		return fmt.Errorf("ListingRevisionAttachment formal semantics require replacement when attachmentType changes")
	}
	payload["attachmentType"] = attachmentType
	switch attachmentType {
	case string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeReviewSupportDocument):
		setPayloadStringIfMissing(payload, "documentName", current.DocumentName)
		setPayloadStringIfMissing(payload, "templateCode", current.TemplateCode)
	case string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeCustomerSuccess):
		setPayloadStringIfMissing(payload, "customerName", current.CustomerName)
	case string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeSupportedServices):
		setPayloadStringIfMissing(payload, "serviceName", current.ServiceName)
		setPayloadStringIfMissing(payload, "type", current.Type)
	}
	return nil
}

func listingRevisionAttachmentUpdateNeeded(payload map[string]any, current listingRevisionAttachmentBody) bool {
	for _, field := range listingRevisionAttachmentMutableFieldsForType(current.AttachmentType) {
		desired, ok := payloadValue(payload, field)
		if !ok {
			continue
		}
		observed, _ := current.value(field)
		if !reflect.DeepEqual(normalizeComparableValue(desired), normalizeComparableValue(observed)) {
			return true
		}
	}
	return false
}

func listingRevisionAttachmentMutableFieldsForType(attachmentType string) []string {
	fields := []string{"displayName", "description", "freeformTags", "definedTags"}
	switch attachmentType {
	case string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeRelatedDocument):
		fields = append(fields, "documentCategory")
	case string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeReviewSupportDocument):
		fields = append(fields, "documentName", "templateCode")
	case string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeCustomerSuccess):
		fields = append(fields, "customerName", "url", "productCodes")
	case string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeSupportedServices):
		fields = append(fields, "serviceName", "url", "type")
	}
	return fields
}

func validateListingRevisionAttachmentCreateOnlyDrift(
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	currentResponse any,
) error {
	current, ok := listingRevisionAttachmentBodyFromResponse(currentResponse)
	if !ok {
		return nil
	}
	payload, err := listingRevisionAttachmentPayload(resource, false)
	if err != nil {
		return err
	}
	if err := rejectListingRevisionAttachmentStringDrift("listingRevisionId", payload, current.ListingRevisionId); err != nil {
		return err
	}
	if err := rejectListingRevisionAttachmentStringDrift("attachmentType", payload, current.AttachmentType); err != nil {
		return err
	}
	if current.AttachmentType == string(marketplacepublishersdk.ListingRevisionAttachmentAttachmentTypeVideo) {
		return rejectListingRevisionAttachmentVideoURLDrift(payload, current.ContentUrl)
	}
	return nil
}

func rejectListingRevisionAttachmentStringDrift(
	field string,
	payload map[string]any,
	current string,
) error {
	desired, ok := payloadString(payload, field)
	if !ok || strings.TrimSpace(desired) == "" || strings.TrimSpace(current) == "" {
		return nil
	}
	if strings.TrimSpace(desired) != strings.TrimSpace(current) {
		return fmt.Errorf("ListingRevisionAttachment formal semantics require replacement when %s changes", field)
	}
	return nil
}

func rejectListingRevisionAttachmentVideoURLDrift(payload map[string]any, currentURL string) error {
	currentURL = strings.TrimSpace(currentURL)
	if currentURL == "" {
		return nil
	}
	desired, ok := nestedPayloadString(payload, []string{"videoAttachmentDetails", "contentUrl"})
	if !ok || strings.TrimSpace(desired) == "" {
		return fmt.Errorf("ListingRevisionAttachment formal semantics require replacement when videoAttachmentDetails.contentUrl changes")
	}
	if strings.TrimSpace(desired) != currentURL {
		return fmt.Errorf("ListingRevisionAttachment formal semantics require replacement when videoAttachmentDetails.contentUrl changes")
	}
	return nil
}

func listListingRevisionAttachmentsAllPages(
	call func(context.Context, marketplacepublishersdk.ListListingRevisionAttachmentsRequest) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error),
) func(context.Context, marketplacepublishersdk.ListListingRevisionAttachmentsRequest) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error) {
	return func(ctx context.Context, request marketplacepublishersdk.ListListingRevisionAttachmentsRequest) (marketplacepublishersdk.ListListingRevisionAttachmentsResponse, error) {
		var combined marketplacepublishersdk.ListListingRevisionAttachmentsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return marketplacepublishersdk.ListListingRevisionAttachmentsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleListingRevisionAttachmentDeleteError(
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	err error,
) error {
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
	return listingRevisionAttachmentAmbiguousNotFoundError{
		message:      "ListingRevisionAttachment delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func wrapListingRevisionAttachmentDeleteConfirmation(hooks *ListingRevisionAttachmentRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getListingRevisionAttachment := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ListingRevisionAttachmentServiceClient) ListingRevisionAttachmentServiceClient {
		return listingRevisionAttachmentDeleteConfirmationClient{
			delegate:                     delegate,
			getListingRevisionAttachment: getListingRevisionAttachment,
		}
	})
}

type listingRevisionAttachmentDeleteConfirmationClient struct {
	delegate                     ListingRevisionAttachmentServiceClient
	getListingRevisionAttachment func(context.Context, marketplacepublishersdk.GetListingRevisionAttachmentRequest) (marketplacepublishersdk.GetListingRevisionAttachmentResponse, error)
}

func (c listingRevisionAttachmentDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c listingRevisionAttachmentDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c listingRevisionAttachmentDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
) error {
	attachmentID := trackedListingRevisionAttachmentID(resource)
	if c.getListingRevisionAttachment == nil || attachmentID == "" {
		return nil
	}
	_, err := c.getListingRevisionAttachment(ctx, marketplacepublishersdk.GetListingRevisionAttachmentRequest{
		ListingRevisionAttachmentId: common.String(attachmentID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("ListingRevisionAttachment delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func applyListingRevisionAttachmentDeleteOutcome(
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	current, ok := listingRevisionAttachmentBodyFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	state := strings.ToUpper(strings.TrimSpace(current.LifecycleState))
	if state == "" || state == string(marketplacepublishersdk.ListingRevisionAttachmentLifecycleStateDeleted) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !listingRevisionAttachmentDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markListingRevisionAttachmentTerminating(resource, response)
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func listingRevisionAttachmentDeleteAlreadyPending(resource *marketplacepublisherv1beta1.ListingRevisionAttachment) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markListingRevisionAttachmentTerminating(
	resource *marketplacepublisherv1beta1.ListingRevisionAttachment,
	response any,
) {
	if resource == nil {
		return
	}
	current, _ := listingRevisionAttachmentBodyFromResponse(response)
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = listingRevisionAttachmentDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       current.LifecycleState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         listingRevisionAttachmentDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		listingRevisionAttachmentDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func trackedListingRevisionAttachmentID(resource *marketplacepublisherv1beta1.ListingRevisionAttachment) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func listingRevisionAttachmentBodyFromResponse(response any) (listingRevisionAttachmentBody, bool) {
	if body, ok := listingRevisionAttachmentResponseBody(response); ok {
		return decodeListingRevisionAttachmentBody(body)
	}
	return decodeListingRevisionAttachmentBody(response)
}

func listingRevisionAttachmentResponseBody(response any) (any, bool) {
	value := reflect.ValueOf(response)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, false
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil, false
	}
	field := value.FieldByName("ListingRevisionAttachment")
	if !field.IsValid() || !field.CanInterface() {
		return nil, false
	}
	return field.Interface(), true
}

func decodeListingRevisionAttachmentBody(body any) (listingRevisionAttachmentBody, bool) {
	if body == nil {
		return listingRevisionAttachmentBody{}, false
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return listingRevisionAttachmentBody{}, false
	}
	var decoded listingRevisionAttachmentBody
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return listingRevisionAttachmentBody{}, false
	}
	return decoded, true
}

func (b listingRevisionAttachmentBody) value(field string) (any, bool) {
	valueFunc, ok := listingRevisionAttachmentBodyValueFuncs[field]
	if !ok {
		return nil, false
	}
	return valueFunc(b)
}

func nonEmptyStringValue(value string) (any, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}

func payloadAttachmentType(payload map[string]any) (string, error) {
	return payloadAttachmentTypeWithFallback(payload, "")
}

func payloadAttachmentTypeWithFallback(payload map[string]any, fallback string) (string, error) {
	if value, ok := payloadString(payload, "attachmentType"); ok && strings.TrimSpace(value) != "" {
		return normalizeListingRevisionAttachmentType(value)
	}
	if strings.TrimSpace(fallback) != "" {
		return normalizeListingRevisionAttachmentType(fallback)
	}
	return "", fmt.Errorf("listing revision attachment spec is missing required field attachmentType")
}

func normalizeListingRevisionAttachmentType(value string) (string, error) {
	normalized, ok := marketplacepublishersdk.GetMappingListingRevisionAttachmentAttachmentTypeEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported ListingRevisionAttachment attachmentType %q", value)
	}
	return string(normalized), nil
}

func normalizeRelatedDocumentCategory(value string) (string, error) {
	normalized, ok := marketplacepublishersdk.GetMappingRelatedDocumentAttachmentDocumentCategoryEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported ListingRevisionAttachment documentCategory %q", value)
	}
	return string(normalized), nil
}

func normalizeSupportedServiceType(value string) (string, error) {
	normalized, ok := marketplacepublishersdk.GetMappingSupportedServiceAttachmentTypeEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported ListingRevisionAttachment supported service type %q", value)
	}
	return string(normalized), nil
}

func setPayloadIdentityString(payload map[string]any, key string, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if current, ok := payloadString(payload, key); ok && strings.TrimSpace(current) != "" && strings.TrimSpace(current) != value {
		return fmt.Errorf("listing revision attachment jsonData identity conflicts with spec field %s", key)
	}
	payload[key] = value
	return nil
}

func setPayloadString(payload map[string]any, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	payload[key] = value
}

func setPayloadStringIfMissing(payload map[string]any, key string, value string) {
	if _, ok := payloadString(payload, key); ok {
		return
	}
	setPayloadString(payload, key, value)
}

func setNestedPayloadString(payload map[string]any, path []string, value string) {
	value = strings.TrimSpace(value)
	if value == "" || len(path) == 0 {
		return
	}
	current := payload
	for _, segment := range path[:len(path)-1] {
		next, ok := current[segment].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[segment] = next
		}
		current = next
	}
	current[path[len(path)-1]] = value
}

func payloadString(payload map[string]any, key string) (string, bool) {
	value, ok := payload[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(text), true
}

func nestedPayloadString(payload map[string]any, path []string) (string, bool) {
	value, ok := payloadValue(payload, strings.Join(path, "."))
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(text), true
}

func requirePayloadString(payload map[string]any, key string) error {
	value, ok := payloadString(payload, key)
	if !ok || strings.TrimSpace(value) == "" {
		return fmt.Errorf("listing revision attachment spec is missing required field %s", key)
	}
	return nil
}

func requireNestedPayloadString(payload map[string]any, path []string) error {
	value, ok := nestedPayloadString(payload, path)
	if !ok || strings.TrimSpace(value) == "" {
		return fmt.Errorf("listing revision attachment spec is missing required field %s", strings.Join(path, "."))
	}
	return nil
}

func payloadValue(payload map[string]any, path string) (any, bool) {
	segments := strings.Split(path, ".")
	var current any = payload
	for _, segment := range segments {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = currentMap[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func normalizeComparableValue(value any) any {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var normalized any
	if err := json.Unmarshal(payload, &normalized); err != nil {
		return value
	}
	return normalized
}

func cloneListingRevisionAttachmentStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func listingRevisionAttachmentDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		convertedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			convertedValues[key] = value
		}
		converted[namespace] = convertedValues
	}
	return converted
}
