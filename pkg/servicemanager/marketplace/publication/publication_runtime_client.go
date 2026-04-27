/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package publication

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	marketplacesdk "github.com/oracle/oci-go-sdk/v65/marketplace"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type publicationOCIClient interface {
	CreatePublication(context.Context, marketplacesdk.CreatePublicationRequest) (marketplacesdk.CreatePublicationResponse, error)
	GetPublication(context.Context, marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error)
	ListPublications(context.Context, marketplacesdk.ListPublicationsRequest) (marketplacesdk.ListPublicationsResponse, error)
	UpdatePublication(context.Context, marketplacesdk.UpdatePublicationRequest) (marketplacesdk.UpdatePublicationResponse, error)
	DeletePublication(context.Context, marketplacesdk.DeletePublicationRequest) (marketplacesdk.DeletePublicationResponse, error)
}

type publicationIdentity struct {
	compartmentID string
	listingType   marketplacesdk.ListPublicationsListingTypeEnum
	name          string
}

type publicationRuntimeClient struct {
	delegate PublicationServiceClient
	log      loggerutil.OSOKLogger
}

func init() {
	registerPublicationRuntimeHooksMutator(func(manager *PublicationServiceManager, hooks *PublicationRuntimeHooks) {
		applyPublicationRuntimeHooks(manager, hooks)
		appendPublicationRuntimeWrapper(manager, hooks)
	})
}

func applyPublicationRuntimeHooks(manager *PublicationServiceManager, hooks *PublicationRuntimeHooks) {
	if hooks == nil {
		return
	}

	var credClient credhelper.CredentialClient
	if manager != nil {
		credClient = manager.CredentialClient
	}

	hooks.Semantics = newPublicationRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *marketplacev1beta1.Publication, namespace string) (any, error) {
		return buildPublicationCreateDetails(ctx, resource, credClient, namespace)
	}
	hooks.Identity.Resolve = resolvePublicationIdentity
	hooks.List.Fields = publicationListFields()
	hooks.ParityHooks.ValidateCreateOnlyDrift = validatePublicationCreateOnlyDrift
}

func appendPublicationRuntimeWrapper(manager *PublicationServiceManager, hooks *PublicationRuntimeHooks) {
	if hooks == nil {
		return
	}

	var log loggerutil.OSOKLogger
	if manager != nil {
		log = manager.Log
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate PublicationServiceClient) PublicationServiceClient {
		return &publicationRuntimeClient{delegate: delegate, log: log}
	})
}

func newPublicationServiceClientWithOCIClient(log loggerutil.OSOKLogger, client publicationOCIClient) PublicationServiceClient {
	manager := &PublicationServiceManager{Log: log}
	hooks := newPublicationRuntimeHooksWithOCIClient(client)
	applyPublicationRuntimeHooks(manager, &hooks)
	delegate := defaultPublicationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*marketplacev1beta1.Publication](
			buildPublicationGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapPublicationGeneratedClient(hooks, delegate)
}

func newPublicationRuntimeHooksWithOCIClient(client publicationOCIClient) PublicationRuntimeHooks {
	return PublicationRuntimeHooks{
		Create: runtimeOperationHooks[marketplacesdk.CreatePublicationRequest, marketplacesdk.CreatePublicationResponse]{
			Fields: publicationCreateFields(),
			Call: func(ctx context.Context, request marketplacesdk.CreatePublicationRequest) (marketplacesdk.CreatePublicationResponse, error) {
				return client.CreatePublication(ctx, request)
			},
		},
		Get: runtimeOperationHooks[marketplacesdk.GetPublicationRequest, marketplacesdk.GetPublicationResponse]{
			Fields: publicationGetFields(),
			Call: func(ctx context.Context, request marketplacesdk.GetPublicationRequest) (marketplacesdk.GetPublicationResponse, error) {
				return client.GetPublication(ctx, request)
			},
		},
		List: runtimeOperationHooks[marketplacesdk.ListPublicationsRequest, marketplacesdk.ListPublicationsResponse]{
			Fields: publicationListFields(),
			Call: func(ctx context.Context, request marketplacesdk.ListPublicationsRequest) (marketplacesdk.ListPublicationsResponse, error) {
				return client.ListPublications(ctx, request)
			},
		},
		Update: runtimeOperationHooks[marketplacesdk.UpdatePublicationRequest, marketplacesdk.UpdatePublicationResponse]{
			Fields: publicationUpdateFields(),
			Call: func(ctx context.Context, request marketplacesdk.UpdatePublicationRequest) (marketplacesdk.UpdatePublicationResponse, error) {
				return client.UpdatePublication(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[marketplacesdk.DeletePublicationRequest, marketplacesdk.DeletePublicationResponse]{
			Fields: publicationDeleteFields(),
			Call: func(ctx context.Context, request marketplacesdk.DeletePublicationRequest) (marketplacesdk.DeletePublicationResponse, error) {
				return client.DeletePublication(ctx, request)
			},
		},
	}
}

func newPublicationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplace",
		FormalSlug:    "publication",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(marketplacesdk.PublicationLifecycleStateCreating)},
			UpdatingStates:     []string{},
			ActiveStates:       []string{string(marketplacesdk.PublicationLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(marketplacesdk.PublicationLifecycleStateDeleting)},
			TerminalStates: []string{string(marketplacesdk.PublicationLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"id", "compartmentId", "listingType", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"freeformTags",
				"longDescription",
				"name",
				"shortDescription",
				"supportContacts",
			},
			ForceNew: []string{
				"compartmentId",
				"listingType",
			},
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
	}
}

func publicationCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreatePublicationDetails", RequestName: "CreatePublicationDetails", Contribution: "body"},
	}
}

func publicationGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "PublicationId",
			RequestName:      "publicationId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.ocid"},
		},
	}
}

func publicationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "ListingType",
			RequestName:  "listingType",
			Contribution: "query",
			LookupPaths:  []string{"status.listingType", "spec.listingType", "listingType"},
		},
		{
			FieldName:        "PublicationId",
			RequestName:      "publicationId",
			Contribution:     "query",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.ocid"},
		},
	}
}

func publicationUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "PublicationId",
			RequestName:      "publicationId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.ocid"},
		},
		{FieldName: "UpdatePublicationDetails", RequestName: "UpdatePublicationDetails", Contribution: "body"},
	}
}

func publicationDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "PublicationId",
			RequestName:      "publicationId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.ocid"},
		},
	}
}

func resolvePublicationIdentity(resource *marketplacev1beta1.Publication) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("Publication resource is nil")
	}
	if err := validatePublicationDesiredState(resource); err != nil {
		return nil, err
	}
	return publicationIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		listingType:   marketplacesdk.ListPublicationsListingTypeEnum(strings.TrimSpace(resource.Spec.ListingType)),
		name:          strings.TrimSpace(resource.Spec.Name),
	}, nil
}

func buildPublicationCreateDetails(
	ctx context.Context,
	resource *marketplacev1beta1.Publication,
	credClient credhelper.CredentialClient,
	namespace string,
) (marketplacesdk.CreatePublicationDetails, error) {
	if resource == nil {
		return marketplacesdk.CreatePublicationDetails{}, fmt.Errorf("Publication resource is nil")
	}
	if err := validatePublicationDesiredState(resource); err != nil {
		return marketplacesdk.CreatePublicationDetails{}, err
	}

	resolved, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, credClient, namespace)
	if err != nil {
		return marketplacesdk.CreatePublicationDetails{}, err
	}
	values, err := publicationSpecMap(resolved)
	if err != nil {
		return marketplacesdk.CreatePublicationDetails{}, err
	}
	if err := normalizePublicationPackageDetails(values); err != nil {
		return marketplacesdk.CreatePublicationDetails{}, err
	}

	payload, err := json.Marshal(values)
	if err != nil {
		return marketplacesdk.CreatePublicationDetails{}, fmt.Errorf("marshal Publication create body: %w", err)
	}
	var details marketplacesdk.CreatePublicationDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return marketplacesdk.CreatePublicationDetails{}, fmt.Errorf("decode Publication create body: %w", err)
	}
	if details.PackageDetails == nil {
		return marketplacesdk.CreatePublicationDetails{}, fmt.Errorf("Publication packageDetails must resolve to a supported OCI package payload")
	}
	return details, nil
}

func publicationSpecMap(resolved any) (map[string]any, error) {
	payload, err := json.Marshal(resolved)
	if err != nil {
		return nil, fmt.Errorf("marshal resolved Publication spec: %w", err)
	}
	values := map[string]any{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode resolved Publication spec: %w", err)
	}
	return values, nil
}

func normalizePublicationPackageDetails(values map[string]any) error {
	rawPackageDetails, _ := values["packageDetails"].(map[string]any)
	if rawPackageDetails == nil {
		return fmt.Errorf("Publication packageDetails is required")
	}

	packageDetails := rawPackageDetails
	if rawJSON, _ := rawPackageDetails["jsonData"].(string); strings.TrimSpace(rawJSON) != "" {
		fromJSON := map[string]any{}
		if err := json.Unmarshal([]byte(rawJSON), &fromJSON); err != nil {
			return fmt.Errorf("decode Publication packageDetails.jsonData: %w", err)
		}
		for _, key := range []string{"packageVersion", "operatingSystem", "packageType", "imageId"} {
			if _, exists := fromJSON[key]; exists {
				continue
			}
			if value, exists := rawPackageDetails[key]; exists && meaningfulPublicationPackageValue(value) {
				fromJSON[key] = value
			}
		}
		packageDetails = fromJSON
	}
	delete(packageDetails, "jsonData")

	packageType := strings.ToUpper(strings.TrimSpace(stringValue(packageDetails["packageType"])))
	if packageType == "" {
		packageType = string(marketplacesdk.PackageTypeEnumImage)
		packageDetails["packageType"] = packageType
	}
	if packageType != string(marketplacesdk.PackageTypeEnumImage) {
		return fmt.Errorf("Publication packageDetails.packageType %q is not supported by the OCI create-publication SDK shape", packageType)
	}
	packageDetails["packageType"] = packageType

	values["packageDetails"] = packageDetails
	return nil
}

func meaningfulPublicationPackageValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case map[string]any:
		for _, child := range typed {
			if meaningfulPublicationPackageValue(child) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func validatePublicationDesiredState(resource *marketplacev1beta1.Publication) error {
	if resource == nil {
		return fmt.Errorf("Publication resource is nil")
	}
	listingType := strings.TrimSpace(resource.Spec.ListingType)
	if _, ok := marketplacesdk.GetMappingListingTypeEnum(listingType); !ok {
		return fmt.Errorf("Publication listingType %q is not supported", resource.Spec.ListingType)
	}
	if !resource.Spec.IsAgreementAcknowledged {
		return fmt.Errorf("Publication requires spec.isAgreementAcknowledged to remain true")
	}
	packageType := strings.TrimSpace(resource.Spec.PackageDetails.PackageType)
	if packageType != "" && !strings.EqualFold(packageType, string(marketplacesdk.PackageTypeEnumImage)) {
		return fmt.Errorf("Publication packageDetails.packageType %q is not supported by the OCI create-publication SDK shape", packageType)
	}
	return nil
}

func validatePublicationCreateOnlyDrift(resource *marketplacev1beta1.Publication, currentResponse any) error {
	if resource == nil || currentResponse == nil {
		return nil
	}
	live := publicationSnapshotFromResponse(currentResponse)
	if live.packageType != "" && strings.TrimSpace(resource.Spec.PackageDetails.PackageType) != "" &&
		!strings.EqualFold(resource.Spec.PackageDetails.PackageType, live.packageType) {
		return fmt.Errorf("Publication formal semantics require replacement when packageDetails.packageType changes")
	}
	desiredOS := strings.TrimSpace(resource.Spec.PackageDetails.OperatingSystem.Name)
	if desiredOS != "" && len(live.operatingSystems) > 0 && !containsPublicationOperatingSystem(live.operatingSystems, desiredOS) {
		return fmt.Errorf("Publication formal semantics require replacement when packageDetails.operatingSystem changes")
	}
	return nil
}

type publicationSnapshot struct {
	packageType      string
	operatingSystems []string
}

func publicationSnapshotFromResponse(response any) publicationSnapshot {
	switch typed := response.(type) {
	case marketplacesdk.GetPublicationResponse:
		return publicationSnapshotFromPublication(typed.Publication)
	case marketplacesdk.CreatePublicationResponse:
		return publicationSnapshotFromPublication(typed.Publication)
	case marketplacesdk.UpdatePublicationResponse:
		return publicationSnapshotFromPublication(typed.Publication)
	case marketplacesdk.Publication:
		return publicationSnapshotFromPublication(typed)
	case marketplacesdk.PublicationSummary:
		return publicationSnapshotFromSummary(typed)
	default:
		return publicationSnapshot{}
	}
}

func publicationSnapshotFromPublication(publication marketplacesdk.Publication) publicationSnapshot {
	return publicationSnapshot{
		packageType:      string(publication.PackageType),
		operatingSystems: publicationOperatingSystemNames(publication.SupportedOperatingSystems),
	}
}

func publicationSnapshotFromSummary(summary marketplacesdk.PublicationSummary) publicationSnapshot {
	return publicationSnapshot{
		packageType:      string(summary.PackageType),
		operatingSystems: publicationOperatingSystemNames(summary.SupportedOperatingSystems),
	}
}

func publicationOperatingSystemNames(operatingSystems []marketplacesdk.OperatingSystem) []string {
	names := make([]string, 0, len(operatingSystems))
	for _, operatingSystem := range operatingSystems {
		if operatingSystem.Name == nil {
			continue
		}
		name := strings.TrimSpace(*operatingSystem.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func containsPublicationOperatingSystem(names []string, desired string) bool {
	for _, name := range names {
		if strings.EqualFold(name, desired) {
			return true
		}
	}
	return false
}

func (c *publicationRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacev1beta1.Publication,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Publication delegate is not configured")
	}
	if err := validatePublicationDesiredState(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.fail(resource, err)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *publicationRuntimeClient) Delete(ctx context.Context, resource *marketplacev1beta1.Publication) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("Publication delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *publicationRuntimeClient) fail(resource *marketplacev1beta1.Publication, err error) error {
	if resource == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return err
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

var _ PublicationServiceClient = (*publicationRuntimeClient)(nil)
var _ publicationOCIClient = (*marketplacesdk.MarketplaceClient)(nil)
