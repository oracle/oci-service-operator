/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevisionpackage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type listingRevisionPackageOCIClient interface {
	CreateListingRevisionPackage(context.Context, marketplacepublishersdk.CreateListingRevisionPackageRequest) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error)
	GetListingRevisionPackage(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error)
	ListListingRevisionPackages(context.Context, marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error)
	UpdateListingRevisionPackage(context.Context, marketplacepublishersdk.UpdateListingRevisionPackageRequest) (marketplacepublishersdk.UpdateListingRevisionPackageResponse, error)
	DeleteListingRevisionPackage(context.Context, marketplacepublishersdk.DeleteListingRevisionPackageRequest) (marketplacepublishersdk.DeleteListingRevisionPackageResponse, error)
}

type listingRevisionPackageRuntimeClient struct {
	delegate ListingRevisionPackageServiceClient
	get      func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error)
}

var _ ListingRevisionPackageServiceClient = (*listingRevisionPackageRuntimeClient)(nil)

type ambiguousListingRevisionPackageNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousListingRevisionPackageNotFoundError) Error() string {
	return e.message
}

func (e ambiguousListingRevisionPackageNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type listingRevisionPackageStatusProjection struct {
	JsonData                    string                                                                `json:"jsonData,omitempty"`
	Id                          string                                                                `json:"id,omitempty"`
	Description                 string                                                                `json:"description,omitempty"`
	ExtendedMetadata            map[string]string                                                     `json:"extendedMetadata,omitempty"`
	FreeformTags                map[string]string                                                     `json:"freeformTags,omitempty"`
	DefinedTags                 map[string]shared.MapValue                                            `json:"definedTags,omitempty"`
	SystemTags                  map[string]shared.MapValue                                            `json:"systemTags,omitempty"`
	DisplayName                 string                                                                `json:"displayName,omitempty"`
	ListingRevisionId           string                                                                `json:"listingRevisionId,omitempty"`
	CompartmentId               string                                                                `json:"compartmentId,omitempty"`
	ArtifactId                  string                                                                `json:"artifactId,omitempty"`
	TermId                      string                                                                `json:"termId,omitempty"`
	PackageVersion              string                                                                `json:"packageVersion,omitempty"`
	LifecycleState              string                                                                `json:"lifecycleState,omitempty"`
	Status                      string                                                                `json:"sdkStatus,omitempty"`
	AreSecurityUpgradesProvided bool                                                                  `json:"areSecurityUpgradesProvided,omitempty"`
	IsDefault                   bool                                                                  `json:"isDefault,omitempty"`
	TimeCreated                 string                                                                `json:"timeCreated,omitempty"`
	TimeUpdated                 string                                                                `json:"timeUpdated,omitempty"`
	PackageType                 string                                                                `json:"packageType,omitempty"`
	MachineImageDetails         marketplacepublisherv1beta1.ListingRevisionPackageMachineImageDetails `json:"machineImageDetails,omitempty"`
}

type listingRevisionPackageProjectedSDK struct {
	listingRevisionPackageStatusProjection
}

type listingRevisionPackageProjectedResponse struct {
	ListingRevisionPackage listingRevisionPackageStatusProjection `presentIn:"body"`
	OpcRequestId           *string                                `presentIn:"header" name:"opc-request-id"`
}

type listingRevisionPackageProjectedCollection struct {
	Items []listingRevisionPackageStatusProjection `json:"items,omitempty"`
}

type listingRevisionPackageProjectedListResponse struct {
	ListingRevisionPackageCollection listingRevisionPackageProjectedCollection `presentIn:"body"`
	OpcRequestId                     *string                                   `presentIn:"header" name:"opc-request-id"`
	OpcNextPage                      *string                                   `presentIn:"header" name:"opc-next-page"`
}

func init() {
	registerListingRevisionPackageRuntimeHooksMutator(func(_ *ListingRevisionPackageServiceManager, hooks *ListingRevisionPackageRuntimeHooks) {
		applyListingRevisionPackageRuntimeHooks(hooks)
	})
}

func applyListingRevisionPackageRuntimeHooks(hooks *ListingRevisionPackageRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedListingRevisionPackageRuntimeSemantics()
	hooks.BuildCreateBody = buildListingRevisionPackageCreateBody
	hooks.BuildUpdateBody = buildListingRevisionPackageUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardListingRevisionPackageExistingBeforeCreate
	hooks.List.Fields = listingRevisionPackageListFields()
	wrapListingRevisionPackageWriteCalls(hooks)
	wrapListingRevisionPackageReadAndDeleteCalls(hooks)
	installListingRevisionPackageProjectedReadOperations(hooks)
	hooks.StatusHooks.ProjectStatus = projectListingRevisionPackageStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateListingRevisionPackageCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleListingRevisionPackageDeleteError
	wrapListingRevisionPackageDeleteConfirmation(hooks)
}

func newListingRevisionPackageServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client listingRevisionPackageOCIClient,
) ListingRevisionPackageServiceClient {
	hooks := newListingRevisionPackageRuntimeHooksWithOCIClient(client)
	applyListingRevisionPackageRuntimeHooks(&hooks)
	delegate := defaultListingRevisionPackageServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.ListingRevisionPackage](
			buildListingRevisionPackageGeneratedRuntimeConfig(&ListingRevisionPackageServiceManager{Log: log}, hooks),
		),
	}
	return wrapListingRevisionPackageGeneratedClient(hooks, delegate)
}

func newListingRevisionPackageRuntimeHooksWithOCIClient(client listingRevisionPackageOCIClient) ListingRevisionPackageRuntimeHooks {
	hooks := newListingRevisionPackageDefaultRuntimeHooks(marketplacepublishersdk.MarketplacePublisherClient{})
	hooks.Create.Call = func(ctx context.Context, request marketplacepublishersdk.CreateListingRevisionPackageRequest) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error) {
		if client == nil {
			return marketplacepublishersdk.CreateListingRevisionPackageResponse{}, fmt.Errorf("ListingRevisionPackage OCI client is not configured")
		}
		return client.CreateListingRevisionPackage(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		if client == nil {
			return marketplacepublishersdk.GetListingRevisionPackageResponse{}, fmt.Errorf("ListingRevisionPackage OCI client is not configured")
		}
		return client.GetListingRevisionPackage(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
		if client == nil {
			return marketplacepublishersdk.ListListingRevisionPackagesResponse{}, fmt.Errorf("ListingRevisionPackage OCI client is not configured")
		}
		return client.ListListingRevisionPackages(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request marketplacepublishersdk.UpdateListingRevisionPackageRequest) (marketplacepublishersdk.UpdateListingRevisionPackageResponse, error) {
		if client == nil {
			return marketplacepublishersdk.UpdateListingRevisionPackageResponse{}, fmt.Errorf("ListingRevisionPackage OCI client is not configured")
		}
		return client.UpdateListingRevisionPackage(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request marketplacepublishersdk.DeleteListingRevisionPackageRequest) (marketplacepublishersdk.DeleteListingRevisionPackageResponse, error) {
		if client == nil {
			return marketplacepublishersdk.DeleteListingRevisionPackageResponse{}, fmt.Errorf("ListingRevisionPackage OCI client is not configured")
		}
		return client.DeleteListingRevisionPackage(ctx, request)
	}
	return hooks
}

func reviewedListingRevisionPackageRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplacepublisher",
		FormalSlug:    "listingrevisionpackage",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateCreating)},
			UpdatingStates:     []string{string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateUpdating)},
			ActiveStates: []string{
				string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
				string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateCreating),
				string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateUpdating),
				string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateDeleting),
			},
			TerminalStates: []string{string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"listingRevisionId", "packageVersion", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"packageVersion",
				"displayName",
				"description",
				"artifactId",
				"termId",
				"isDefault",
				"areSecurityUpgradesProvided",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"listingRevisionId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ListingRevisionPackage", Action: "CreateListingRevisionPackage"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ListingRevisionPackage", Action: "UpdateListingRevisionPackage"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ListingRevisionPackage", Action: "DeleteListingRevisionPackage"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ListingRevisionPackage", Action: "GetListingRevisionPackage"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ListingRevisionPackage", Action: "GetListingRevisionPackage"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ListingRevisionPackage", Action: "GetListingRevisionPackage"}},
		},
	}
}

func listingRevisionPackageListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "ListingRevisionId",
			RequestName:  "listingRevisionId",
			Contribution: "query",
			LookupPaths:  []string{"status.listingRevisionId", "spec.listingRevisionId", "listingRevisionId"},
		},
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "compartmentId"},
		},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func buildListingRevisionPackageCreateBody(
	_ context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ListingRevisionPackage resource is nil")
	}
	if err := validateListingRevisionPackageSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := marketplacepublishersdk.CreateListingRevisionPackageDetails{
		ListingRevisionId:           common.String(strings.TrimSpace(resource.Spec.ListingRevisionId)),
		PackageVersion:              common.String(strings.TrimSpace(resource.Spec.PackageVersion)),
		ArtifactId:                  common.String(strings.TrimSpace(resource.Spec.ArtifactId)),
		TermId:                      common.String(strings.TrimSpace(resource.Spec.TermId)),
		AreSecurityUpgradesProvided: common.Bool(resource.Spec.AreSecurityUpgradesProvided),
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		body.DisplayName = common.String(displayName)
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		body.Description = common.String(description)
	}
	if resource.Spec.IsDefault {
		body.IsDefault = common.Bool(true)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildListingRevisionPackageUpdateBody(
	_ context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return marketplacepublishersdk.UpdateListingRevisionPackageDetails{}, false, fmt.Errorf("ListingRevisionPackage resource is nil")
	}
	if err := validateListingRevisionPackageSpec(resource.Spec); err != nil {
		return marketplacepublishersdk.UpdateListingRevisionPackageDetails{}, false, err
	}
	current, ok := listingRevisionPackageProjectionFromResponse(currentResponse)
	if !ok {
		return marketplacepublishersdk.UpdateListingRevisionPackageDetails{}, false, fmt.Errorf("current ListingRevisionPackage response does not expose a ListingRevisionPackage body")
	}
	if err := validateListingRevisionPackageCreateOnlyDrift(resource.Spec, current); err != nil {
		return marketplacepublishersdk.UpdateListingRevisionPackageDetails{}, false, err
	}

	details := marketplacepublishersdk.UpdateListingRevisionPackageDetails{}
	updateNeeded := applyListingRevisionPackageTextUpdates(&details, resource.Spec, current)
	if applyListingRevisionPackageReferenceUpdates(&details, resource.Spec, current) {
		updateNeeded = true
	}
	if applyListingRevisionPackageFlagUpdates(&details, resource.Spec, current) {
		updateNeeded = true
	}
	if applyListingRevisionPackageTagUpdates(&details, resource.Spec, current) {
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func validateListingRevisionPackageSpec(spec marketplacepublisherv1beta1.ListingRevisionPackageSpec) error {
	var missing []string
	if strings.TrimSpace(spec.ListingRevisionId) == "" {
		missing = append(missing, "listingRevisionId")
	}
	if strings.TrimSpace(spec.PackageVersion) == "" {
		missing = append(missing, "packageVersion")
	}
	if strings.TrimSpace(spec.ArtifactId) == "" {
		missing = append(missing, "artifactId")
	}
	if strings.TrimSpace(spec.TermId) == "" {
		missing = append(missing, "termId")
	}
	if len(missing) != 0 {
		return fmt.Errorf("ListingRevisionPackage spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func guardListingRevisionPackageExistingBeforeCreate(
	_ context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ListingRevisionPackage resource is nil")
	}
	if strings.TrimSpace(resource.Spec.ListingRevisionId) == "" || strings.TrimSpace(resource.Spec.PackageVersion) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateListingRevisionPackageCreateOnlyDriftForResponse(
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("ListingRevisionPackage resource is nil")
	}
	current, ok := listingRevisionPackageProjectionFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current ListingRevisionPackage response does not expose a ListingRevisionPackage body")
	}
	return validateListingRevisionPackageCreateOnlyDrift(resource.Spec, current)
}

func validateListingRevisionPackageCreateOnlyDrift(
	spec marketplacepublisherv1beta1.ListingRevisionPackageSpec,
	current listingRevisionPackageStatusProjection,
) error {
	if strings.TrimSpace(spec.ListingRevisionId) == strings.TrimSpace(current.ListingRevisionId) {
		return nil
	}
	return fmt.Errorf("ListingRevisionPackage create-only field drift is not supported: listingRevisionId")
}

func applyListingRevisionPackageTextUpdates(
	details *marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	spec marketplacepublisherv1beta1.ListingRevisionPackageSpec,
	current listingRevisionPackageStatusProjection,
) bool {
	packageVersion := applyListingRevisionPackageStringUpdate(&details.PackageVersion, spec.PackageVersion, current.PackageVersion)
	displayName := applyListingRevisionPackageOptionalStringUpdate(&details.DisplayName, spec.DisplayName, current.DisplayName)
	description := applyListingRevisionPackageOptionalStringUpdate(&details.Description, spec.Description, current.Description)
	return packageVersion || displayName || description
}

func applyListingRevisionPackageReferenceUpdates(
	details *marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	spec marketplacepublisherv1beta1.ListingRevisionPackageSpec,
	current listingRevisionPackageStatusProjection,
) bool {
	artifactID := applyListingRevisionPackageStringUpdate(&details.ArtifactId, spec.ArtifactId, current.ArtifactId)
	termID := applyListingRevisionPackageStringUpdate(&details.TermId, spec.TermId, current.TermId)
	return artifactID || termID
}

func applyListingRevisionPackageFlagUpdates(
	details *marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	spec marketplacepublisherv1beta1.ListingRevisionPackageSpec,
	current listingRevisionPackageStatusProjection,
) bool {
	securityUpgrades := applyListingRevisionPackageRequiredBoolUpdate(
		&details.AreSecurityUpgradesProvided,
		spec.AreSecurityUpgradesProvided,
		current.AreSecurityUpgradesProvided,
	)
	defaultPackage := applyListingRevisionPackageOptionalBoolUpdate(&details.IsDefault, spec.IsDefault, current.IsDefault)
	return securityUpgrades || defaultPackage
}

func applyListingRevisionPackageTagUpdates(
	details *marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	spec marketplacepublisherv1beta1.ListingRevisionPackageSpec,
	current listingRevisionPackageStatusProjection,
) bool {
	freeformTags := applyListingRevisionPackageFreeformTagsUpdate(details, spec, current)
	definedTags := applyListingRevisionPackageDefinedTagsUpdate(details, spec, current)
	return freeformTags || definedTags
}

func applyListingRevisionPackageStringUpdate(target **string, desired string, current string) bool {
	desired = strings.TrimSpace(desired)
	if desired == "" || desired == strings.TrimSpace(current) {
		return false
	}
	*target = common.String(desired)
	return true
}

func applyListingRevisionPackageOptionalStringUpdate(target **string, desired string, current string) bool {
	desired = strings.TrimSpace(desired)
	if desired == "" || desired == strings.TrimSpace(current) {
		return false
	}
	*target = common.String(desired)
	return true
}

func applyListingRevisionPackageRequiredBoolUpdate(target **bool, desired bool, current bool) bool {
	if desired == current {
		return false
	}
	*target = common.Bool(desired)
	return true
}

func applyListingRevisionPackageOptionalBoolUpdate(target **bool, desired bool, current bool) bool {
	if !desired && !current {
		return false
	}
	if desired == current {
		return false
	}
	*target = common.Bool(desired)
	return true
}

func applyListingRevisionPackageFreeformTagsUpdate(
	details *marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	spec marketplacepublisherv1beta1.ListingRevisionPackageSpec,
	current listingRevisionPackageStatusProjection,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	details.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func applyListingRevisionPackageDefinedTagsUpdate(
	details *marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	spec marketplacepublisherv1beta1.ListingRevisionPackageSpec,
	current listingRevisionPackageStatusProjection,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	if reflect.DeepEqual(desired, listingRevisionPackageStatusDefinedTagsToOCI(current.DefinedTags)) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func wrapListingRevisionPackageWriteCalls(hooks *ListingRevisionPackageRuntimeHooks) {
	if hooks.Create.Call != nil {
		createCall := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request marketplacepublishersdk.CreateListingRevisionPackageRequest) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error) {
			response, err := createCall(ctx, request)
			if err == nil {
				response.ListingRevisionPackage = listingRevisionPackageProjectedSDKFromSDK(response.ListingRevisionPackage)
			}
			return response, err
		}
	}
	if hooks.Update.Call != nil {
		updateCall := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request marketplacepublishersdk.UpdateListingRevisionPackageRequest) (marketplacepublishersdk.UpdateListingRevisionPackageResponse, error) {
			response, err := updateCall(ctx, request)
			if err == nil {
				response.ListingRevisionPackage = listingRevisionPackageProjectedSDKFromSDK(response.ListingRevisionPackage)
			}
			return response, err
		}
	}
}

func wrapListingRevisionPackageReadAndDeleteCalls(hooks *ListingRevisionPackageRuntimeHooks) {
	if hooks.Get.Call != nil {
		getCall := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativeListingRevisionPackageNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		listCall := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
			return listListingRevisionPackagesAllPages(ctx, listCall, request)
		}
	}
	if hooks.Delete.Call != nil {
		deleteCall := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request marketplacepublishersdk.DeleteListingRevisionPackageRequest) (marketplacepublishersdk.DeleteListingRevisionPackageResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeListingRevisionPackageNotFoundError(err, "delete")
		}
	}
}

func installListingRevisionPackageProjectedReadOperations(hooks *ListingRevisionPackageRuntimeHooks) {
	if hooks.Get.Call != nil {
		getFields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
		hooks.Read.Get = &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacepublishersdk.GetListingRevisionPackageRequest{} },
			Fields:     getFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*marketplacepublishersdk.GetListingRevisionPackageRequest))
				if err != nil {
					return nil, err
				}
				projected, ok := listingRevisionPackageProjectionFromSDK(response.ListingRevisionPackage)
				if !ok {
					return nil, fmt.Errorf("ListingRevisionPackage read response does not expose a package body")
				}
				return listingRevisionPackageProjectedResponse{
					ListingRevisionPackage: projected,
					OpcRequestId:           response.OpcRequestId,
				}, nil
			},
		}
	}
	if hooks.List.Call != nil {
		listFields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
		hooks.Read.List = &generatedruntime.Operation{
			NewRequest: func() any { return &marketplacepublishersdk.ListListingRevisionPackagesRequest{} },
			Fields:     listFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*marketplacepublishersdk.ListListingRevisionPackagesRequest))
				if err != nil {
					return nil, err
				}
				return listingRevisionPackageProjectedListResponseFromSDK(response), nil
			},
		}
	}
}

func listListingRevisionPackagesAllPages(
	ctx context.Context,
	list func(context.Context, marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error),
	request marketplacepublishersdk.ListListingRevisionPackagesRequest,
) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
	var combined marketplacepublishersdk.ListListingRevisionPackagesResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return marketplacepublishersdk.ListListingRevisionPackagesResponse{}, conservativeListingRevisionPackageNotFoundError(err, "list")
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

func listingRevisionPackageProjectedListResponseFromSDK(
	response marketplacepublishersdk.ListListingRevisionPackagesResponse,
) listingRevisionPackageProjectedListResponse {
	projected := listingRevisionPackageProjectedListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		projected.ListingRevisionPackageCollection.Items = append(
			projected.ListingRevisionPackageCollection.Items,
			listingRevisionPackageProjectionFromSummary(item),
		)
	}
	return projected
}

func projectListingRevisionPackageStatus(
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	response any,
) error {
	if resource == nil {
		return fmt.Errorf("ListingRevisionPackage resource is nil")
	}
	projected, ok := listingRevisionPackageProjectionFromResponse(response)
	if !ok {
		return nil
	}
	resource.Status = marketplacepublisherv1beta1.ListingRevisionPackageStatus{
		OsokStatus:                  resource.Status.OsokStatus,
		JsonData:                    projected.JsonData,
		Id:                          projected.Id,
		Description:                 projected.Description,
		ExtendedMetadata:            maps.Clone(projected.ExtendedMetadata),
		FreeformTags:                maps.Clone(projected.FreeformTags),
		DefinedTags:                 cloneListingRevisionPackageStatusDefinedTags(projected.DefinedTags),
		SystemTags:                  cloneListingRevisionPackageStatusDefinedTags(projected.SystemTags),
		DisplayName:                 projected.DisplayName,
		ListingRevisionId:           projected.ListingRevisionId,
		CompartmentId:               projected.CompartmentId,
		ArtifactId:                  projected.ArtifactId,
		TermId:                      projected.TermId,
		PackageVersion:              projected.PackageVersion,
		LifecycleState:              projected.LifecycleState,
		Status:                      projected.Status,
		AreSecurityUpgradesProvided: projected.AreSecurityUpgradesProvided,
		IsDefault:                   projected.IsDefault,
		TimeCreated:                 projected.TimeCreated,
		TimeUpdated:                 projected.TimeUpdated,
		PackageType:                 projected.PackageType,
		MachineImageDetails:         projected.MachineImageDetails,
	}
	return nil
}

func listingRevisionPackageProjectionFromResponse(response any) (listingRevisionPackageStatusProjection, bool) {
	if projected, ok := listingRevisionPackageProjectionFromLocalResponse(response); ok {
		return projected, true
	}
	if projected, ok := listingRevisionPackageProjectionFromSDKResponse(response); ok {
		return projected, true
	}
	return listingRevisionPackageProjectionFromSDKResource(response)
}

func listingRevisionPackageProjectionFromLocalResponse(response any) (listingRevisionPackageStatusProjection, bool) {
	switch current := response.(type) {
	case listingRevisionPackageProjectedResponse:
		return current.ListingRevisionPackage, true
	case *listingRevisionPackageProjectedResponse:
		if current == nil {
			return listingRevisionPackageStatusProjection{}, false
		}
		return current.ListingRevisionPackage, true
	case listingRevisionPackageStatusProjection:
		return current, true
	case *listingRevisionPackageStatusProjection:
		if current == nil {
			return listingRevisionPackageStatusProjection{}, false
		}
		return *current, true
	default:
		return listingRevisionPackageStatusProjection{}, false
	}
}

func listingRevisionPackageProjectionFromSDKResponse(response any) (listingRevisionPackageStatusProjection, bool) {
	switch current := response.(type) {
	case marketplacepublishersdk.CreateListingRevisionPackageResponse:
		return listingRevisionPackageProjectionFromSDK(current.ListingRevisionPackage)
	case *marketplacepublishersdk.CreateListingRevisionPackageResponse:
		if current == nil {
			return listingRevisionPackageStatusProjection{}, false
		}
		return listingRevisionPackageProjectionFromSDK(current.ListingRevisionPackage)
	case marketplacepublishersdk.UpdateListingRevisionPackageResponse:
		return listingRevisionPackageProjectionFromSDK(current.ListingRevisionPackage)
	case *marketplacepublishersdk.UpdateListingRevisionPackageResponse:
		if current == nil {
			return listingRevisionPackageStatusProjection{}, false
		}
		return listingRevisionPackageProjectionFromSDK(current.ListingRevisionPackage)
	case marketplacepublishersdk.GetListingRevisionPackageResponse:
		return listingRevisionPackageProjectionFromSDK(current.ListingRevisionPackage)
	case *marketplacepublishersdk.GetListingRevisionPackageResponse:
		if current == nil {
			return listingRevisionPackageStatusProjection{}, false
		}
		return listingRevisionPackageProjectionFromSDK(current.ListingRevisionPackage)
	default:
		return listingRevisionPackageStatusProjection{}, false
	}
}

func listingRevisionPackageProjectionFromSDKResource(response any) (listingRevisionPackageStatusProjection, bool) {
	switch current := response.(type) {
	case marketplacepublishersdk.ListingRevisionPackage:
		return listingRevisionPackageProjectionFromSDK(current)
	case marketplacepublishersdk.ListingRevisionPackageSummary:
		return listingRevisionPackageProjectionFromSummary(current), true
	case *marketplacepublishersdk.ListingRevisionPackageSummary:
		if current == nil {
			return listingRevisionPackageStatusProjection{}, false
		}
		return listingRevisionPackageProjectionFromSummary(*current), true
	default:
		return listingRevisionPackageStatusProjection{}, false
	}
}

func listingRevisionPackageProjectionFromSDK(
	current marketplacepublishersdk.ListingRevisionPackage,
) (listingRevisionPackageStatusProjection, bool) {
	if current == nil {
		return listingRevisionPackageStatusProjection{}, false
	}
	rawPayload, _ := json.Marshal(current)
	extra := struct {
		PackageType         string                                                                `json:"packageType,omitempty"`
		MachineImageDetails marketplacepublisherv1beta1.ListingRevisionPackageMachineImageDetails `json:"machineImageDetails,omitempty"`
	}{}
	if len(rawPayload) != 0 {
		_ = json.Unmarshal(rawPayload, &extra)
	}

	return listingRevisionPackageStatusProjection{
		JsonData:                    string(rawPayload),
		Id:                          listingRevisionPackageStringValue(current.GetId()),
		Description:                 listingRevisionPackageStringValue(current.GetDescription()),
		ExtendedMetadata:            maps.Clone(current.GetExtendedMetadata()),
		FreeformTags:                maps.Clone(current.GetFreeformTags()),
		DefinedTags:                 listingRevisionPackageDefinedTagsStatusMap(current.GetDefinedTags()),
		SystemTags:                  listingRevisionPackageDefinedTagsStatusMap(current.GetSystemTags()),
		DisplayName:                 listingRevisionPackageStringValue(current.GetDisplayName()),
		ListingRevisionId:           listingRevisionPackageStringValue(current.GetListingRevisionId()),
		CompartmentId:               listingRevisionPackageStringValue(current.GetCompartmentId()),
		ArtifactId:                  listingRevisionPackageStringValue(current.GetArtifactId()),
		TermId:                      listingRevisionPackageStringValue(current.GetTermId()),
		PackageVersion:              listingRevisionPackageStringValue(current.GetPackageVersion()),
		LifecycleState:              string(current.GetLifecycleState()),
		Status:                      string(current.GetStatus()),
		AreSecurityUpgradesProvided: listingRevisionPackageBoolValue(current.GetAreSecurityUpgradesProvided()),
		IsDefault:                   listingRevisionPackageBoolValue(current.GetIsDefault()),
		TimeCreated:                 listingRevisionPackageSDKTimeString(current.GetTimeCreated()),
		TimeUpdated:                 listingRevisionPackageSDKTimeString(current.GetTimeUpdated()),
		PackageType:                 extra.PackageType,
		MachineImageDetails:         extra.MachineImageDetails,
	}, true
}

func listingRevisionPackageProjectedSDKFromSDK(
	current marketplacepublishersdk.ListingRevisionPackage,
) marketplacepublishersdk.ListingRevisionPackage {
	projected, ok := listingRevisionPackageProjectionFromSDK(current)
	if !ok {
		return current
	}
	return listingRevisionPackageProjectedSDK{listingRevisionPackageStatusProjection: projected}
}

func (p listingRevisionPackageProjectedSDK) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.listingRevisionPackageStatusProjection)
}

func (p listingRevisionPackageProjectedSDK) GetDisplayName() *string {
	return listingRevisionPackageOptionalStringPointer(p.DisplayName)
}

func (p listingRevisionPackageProjectedSDK) GetListingRevisionId() *string {
	return listingRevisionPackageOptionalStringPointer(p.ListingRevisionId)
}

func (p listingRevisionPackageProjectedSDK) GetCompartmentId() *string {
	return listingRevisionPackageOptionalStringPointer(p.CompartmentId)
}

func (p listingRevisionPackageProjectedSDK) GetArtifactId() *string {
	return listingRevisionPackageOptionalStringPointer(p.ArtifactId)
}

func (p listingRevisionPackageProjectedSDK) GetTermId() *string {
	return listingRevisionPackageOptionalStringPointer(p.TermId)
}

func (p listingRevisionPackageProjectedSDK) GetPackageVersion() *string {
	return listingRevisionPackageOptionalStringPointer(p.PackageVersion)
}

func (p listingRevisionPackageProjectedSDK) GetLifecycleState() marketplacepublishersdk.ListingRevisionPackageLifecycleStateEnum {
	return marketplacepublishersdk.ListingRevisionPackageLifecycleStateEnum(p.LifecycleState)
}

func (p listingRevisionPackageProjectedSDK) GetStatus() marketplacepublishersdk.ListingRevisionPackageStatusEnum {
	return marketplacepublishersdk.ListingRevisionPackageStatusEnum(p.Status)
}

func (p listingRevisionPackageProjectedSDK) GetAreSecurityUpgradesProvided() *bool {
	return common.Bool(p.AreSecurityUpgradesProvided)
}

func (p listingRevisionPackageProjectedSDK) GetIsDefault() *bool {
	return common.Bool(p.IsDefault)
}

func (p listingRevisionPackageProjectedSDK) GetTimeCreated() *common.SDKTime {
	return nil
}

func (p listingRevisionPackageProjectedSDK) GetTimeUpdated() *common.SDKTime {
	return nil
}

func (p listingRevisionPackageProjectedSDK) GetId() *string {
	return listingRevisionPackageOptionalStringPointer(p.Id)
}

func (p listingRevisionPackageProjectedSDK) GetDescription() *string {
	return listingRevisionPackageOptionalStringPointer(p.Description)
}

func (p listingRevisionPackageProjectedSDK) GetExtendedMetadata() map[string]string {
	return maps.Clone(p.ExtendedMetadata)
}

func (p listingRevisionPackageProjectedSDK) GetFreeformTags() map[string]string {
	return maps.Clone(p.FreeformTags)
}

func (p listingRevisionPackageProjectedSDK) GetDefinedTags() map[string]map[string]interface{} {
	return listingRevisionPackageStatusDefinedTagsToOCI(p.DefinedTags)
}

func (p listingRevisionPackageProjectedSDK) GetSystemTags() map[string]map[string]interface{} {
	return listingRevisionPackageStatusDefinedTagsToOCI(p.SystemTags)
}

func listingRevisionPackageProjectionFromSummary(
	current marketplacepublishersdk.ListingRevisionPackageSummary,
) listingRevisionPackageStatusProjection {
	return listingRevisionPackageStatusProjection{
		Id:                          listingRevisionPackageStringValue(current.Id),
		ListingRevisionId:           listingRevisionPackageStringValue(current.ListingRevisionId),
		CompartmentId:               listingRevisionPackageStringValue(current.CompartmentId),
		DisplayName:                 listingRevisionPackageStringValue(current.DisplayName),
		PackageVersion:              listingRevisionPackageStringValue(current.PackageVersion),
		PackageType:                 string(current.PackageType),
		AreSecurityUpgradesProvided: listingRevisionPackageBoolValue(current.AreSecurityUpgradesProvided),
		LifecycleState:              string(current.LifecycleState),
		Status:                      string(current.Status),
		TimeCreated:                 listingRevisionPackageSDKTimeString(current.TimeCreated),
		TimeUpdated:                 listingRevisionPackageSDKTimeString(current.TimeUpdated),
		FreeformTags:                maps.Clone(current.FreeformTags),
		DefinedTags:                 listingRevisionPackageDefinedTagsStatusMap(current.DefinedTags),
		SystemTags:                  listingRevisionPackageDefinedTagsStatusMap(current.SystemTags),
	}
}

func wrapListingRevisionPackageDeleteConfirmation(hooks *ListingRevisionPackageRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getPackage := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ListingRevisionPackageServiceClient) ListingRevisionPackageServiceClient {
		return &listingRevisionPackageRuntimeClient{delegate: delegate, get: getPackage}
	})
}

func (c *listingRevisionPackageRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ListingRevisionPackage runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *listingRevisionPackageRuntimeClient) Delete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("ListingRevisionPackage runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *listingRevisionPackageRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
) error {
	currentID := currentListingRevisionPackageID(resource)
	if currentID == "" || c.get == nil {
		return nil
	}
	_, err := c.get(ctx, marketplacepublishersdk.GetListingRevisionPackageRequest{ListingRevisionPackageId: common.String(currentID)})
	if err == nil || !isAmbiguousListingRevisionPackageNotFound(err) {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("ListingRevisionPackage delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %w", err)
}

func currentListingRevisionPackageID(resource *marketplacepublisherv1beta1.ListingRevisionPackage) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func handleListingRevisionPackageDeleteError(
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	err error,
) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeListingRevisionPackageNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("ListingRevisionPackage %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousListingRevisionPackageNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousListingRevisionPackageNotFoundError{message: message, opcRequestID: errorutil.OpcRequestID(err)}
}

func isAmbiguousListingRevisionPackageNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousListingRevisionPackageNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func listingRevisionPackageDefinedTagsStatusMap(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		if values == nil {
			converted[namespace] = nil
			continue
		}
		copied := make(shared.MapValue, len(values))
		for key, value := range values {
			if stringValue, ok := value.(string); ok {
				copied[key] = stringValue
				continue
			}
			copied[key] = fmt.Sprint(value)
		}
		converted[namespace] = copied
	}
	return converted
}

func listingRevisionPackageStatusDefinedTagsToOCI(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			converted[namespace] = nil
			continue
		}
		copied := make(map[string]interface{}, len(values))
		for key, value := range values {
			copied[key] = value
		}
		converted[namespace] = copied
	}
	return converted
}

func cloneListingRevisionPackageStatusDefinedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		copied := make(shared.MapValue, len(values))
		for key, value := range values {
			copied[key] = value
		}
		cloned[namespace] = copied
	}
	return cloned
}

func listingRevisionPackageStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func listingRevisionPackageOptionalStringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func listingRevisionPackageBoolValue(value *bool) bool {
	return value != nil && *value
}

func listingRevisionPackageSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
