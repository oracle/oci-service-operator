/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privateapplication

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	servicecatalogsdk "github.com/oracle/oci-go-sdk/v65/servicecatalog"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
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

type privateApplicationOCIClient interface {
	privateApplicationPackageOCIClient

	CreatePrivateApplication(context.Context, servicecatalogsdk.CreatePrivateApplicationRequest) (servicecatalogsdk.CreatePrivateApplicationResponse, error)
	GetPrivateApplication(context.Context, servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error)
	ListPrivateApplications(context.Context, servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error)
	UpdatePrivateApplication(context.Context, servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error)
	DeletePrivateApplication(context.Context, servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error)
}

type privateApplicationPackageOCIClient interface {
	GetPrivateApplication(context.Context, servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error)
	ListPrivateApplications(context.Context, servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error)
	ListPrivateApplicationPackages(context.Context, servicecatalogsdk.ListPrivateApplicationPackagesRequest) (servicecatalogsdk.ListPrivateApplicationPackagesResponse, error)
	GetPrivateApplicationPackageActionDownloadConfig(context.Context, servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigRequest) (servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigResponse, error)
	GetPrivateApplicationActionDownloadLogo(context.Context, servicecatalogsdk.GetPrivateApplicationActionDownloadLogoRequest) (servicecatalogsdk.GetPrivateApplicationActionDownloadLogoResponse, error)
}

type ambiguousPrivateApplicationNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousPrivateApplicationNotFoundError) Error() string {
	return e.message
}

func (e ambiguousPrivateApplicationNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerPrivateApplicationRuntimeHooksMutator(func(manager *PrivateApplicationServiceManager, hooks *PrivateApplicationRuntimeHooks) {
		packageClient, initErr := privateApplicationPackageClientForManager(manager)
		applyPrivateApplicationRuntimeHooks(hooks, packageClient, initErr, privateApplicationManagerLog(manager))
	})
}

func applyPrivateApplicationRuntimeHooks(
	hooks *PrivateApplicationRuntimeHooks,
	packageClient privateApplicationPackageOCIClient,
	packageClientInitErr error,
	log loggerutil.OSOKLogger,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = privateApplicationRuntimeSemantics()
	hooks.BuildCreateBody = buildPrivateApplicationCreateBody
	hooks.BuildUpdateBody = func(ctx context.Context, resource *servicecatalogv1beta1.PrivateApplication, namespace string, currentResponse any) (any, bool, error) {
		return buildPrivateApplicationUpdateBody(ctx, resource, namespace, currentResponse, packageClient, packageClientInitErr)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardPrivateApplicationExistingBeforeCreate
	hooks.List.Fields = privateApplicationListFields()
	wrapPrivateApplicationReadAndDeleteCalls(hooks)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedPrivateApplicationIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validatePrivateApplicationCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handlePrivateApplicationDeleteError
	wrapPrivateApplicationPackageDriftGuard(hooks, packageClient, packageClientInitErr, log)
	wrapPrivateApplicationDeleteConfirmation(hooks, log)
}

func privateApplicationPackageClientForManager(
	manager *PrivateApplicationServiceManager,
) (privateApplicationPackageOCIClient, error) {
	if manager == nil {
		return nil, nil
	}
	client, err := servicecatalogsdk.NewServiceCatalogClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize PrivateApplication package OCI client: %w", err)
	}
	return client, nil
}

func privateApplicationManagerLog(manager *PrivateApplicationServiceManager) loggerutil.OSOKLogger {
	if manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return manager.Log
}

func newPrivateApplicationServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client privateApplicationOCIClient,
) PrivateApplicationServiceClient {
	hooks := newPrivateApplicationRuntimeHooksWithOCIClient(client)
	applyPrivateApplicationRuntimeHooks(&hooks, client, nil, log)
	manager := &PrivateApplicationServiceManager{Log: log}
	delegate := defaultPrivateApplicationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*servicecatalogv1beta1.PrivateApplication](
			buildPrivateApplicationGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapPrivateApplicationGeneratedClient(hooks, delegate)
}

func newPrivateApplicationRuntimeHooksWithOCIClient(client privateApplicationOCIClient) PrivateApplicationRuntimeHooks {
	return PrivateApplicationRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*servicecatalogv1beta1.PrivateApplication]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*servicecatalogv1beta1.PrivateApplication]{},
		StatusHooks:     generatedruntime.StatusHooks[*servicecatalogv1beta1.PrivateApplication]{},
		ParityHooks:     generatedruntime.ParityHooks[*servicecatalogv1beta1.PrivateApplication]{},
		Async:           generatedruntime.AsyncHooks[*servicecatalogv1beta1.PrivateApplication]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*servicecatalogv1beta1.PrivateApplication]{},
		Create: runtimeOperationHooks[servicecatalogsdk.CreatePrivateApplicationRequest, servicecatalogsdk.CreatePrivateApplicationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreatePrivateApplicationDetails", RequestName: "CreatePrivateApplicationDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request servicecatalogsdk.CreatePrivateApplicationRequest) (servicecatalogsdk.CreatePrivateApplicationResponse, error) {
				return client.CreatePrivateApplication(ctx, request)
			},
		},
		Get: runtimeOperationHooks[servicecatalogsdk.GetPrivateApplicationRequest, servicecatalogsdk.GetPrivateApplicationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "PrivateApplicationId", RequestName: "privateApplicationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
				return client.GetPrivateApplication(ctx, request)
			},
		},
		List: runtimeOperationHooks[servicecatalogsdk.ListPrivateApplicationsRequest, servicecatalogsdk.ListPrivateApplicationsResponse]{
			Fields: privateApplicationListFields(),
			Call: func(ctx context.Context, request servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error) {
				return client.ListPrivateApplications(ctx, request)
			},
		},
		Update: runtimeOperationHooks[servicecatalogsdk.UpdatePrivateApplicationRequest, servicecatalogsdk.UpdatePrivateApplicationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "PrivateApplicationId", RequestName: "privateApplicationId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdatePrivateApplicationDetails", RequestName: "UpdatePrivateApplicationDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request servicecatalogsdk.UpdatePrivateApplicationRequest) (servicecatalogsdk.UpdatePrivateApplicationResponse, error) {
				return client.UpdatePrivateApplication(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[servicecatalogsdk.DeletePrivateApplicationRequest, servicecatalogsdk.DeletePrivateApplicationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "PrivateApplicationId", RequestName: "privateApplicationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
				return client.DeletePrivateApplication(ctx, request)
			},
		},
		WrapGeneratedClient: []func(PrivateApplicationServiceClient) PrivateApplicationServiceClient{},
	}
}

func privateApplicationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "servicecatalog",
		FormalSlug:    "privateapplication",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(servicecatalogsdk.PrivateApplicationLifecycleStateCreating)},
			UpdatingStates:     []string{string(servicecatalogsdk.PrivateApplicationLifecycleStateUpdating)},
			ActiveStates:       []string{string(servicecatalogsdk.PrivateApplicationLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(servicecatalogsdk.PrivateApplicationLifecycleStateDeleting)},
			TerminalStates: []string{string(servicecatalogsdk.PrivateApplicationLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "shortDescription", "longDescription", "logoFileBase64Encoded", "definedTags", "freeformTags"},
			Mutable:         []string{"displayName", "shortDescription", "longDescription", "logoFileBase64Encoded", "definedTags", "freeformTags"},
			ForceNew:        []string{"compartmentId", "packageDetails"},
			ConflictsWith:   map[string][]string{},
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

func privateApplicationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "PrivateApplicationId", RequestName: "privateApplicationId", Contribution: "query", PreferResourceID: true, LookupPaths: []string{"status.id", "status.ocid"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildPrivateApplicationCreateBody(
	_ context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("PrivateApplication resource is nil")
	}
	if err := validatePrivateApplicationSpec(resource.Spec); err != nil {
		return nil, err
	}

	packageDetails, err := sdkPrivateApplicationPackage(resource.Spec.PackageDetails)
	if err != nil {
		return nil, err
	}

	body := servicecatalogsdk.CreatePrivateApplicationDetails{
		CompartmentId:    common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:      common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		ShortDescription: common.String(strings.TrimSpace(resource.Spec.ShortDescription)),
		PackageDetails:   packageDetails,
	}
	if value := strings.TrimSpace(resource.Spec.LongDescription); value != "" {
		body.LongDescription = common.String(value)
	}
	if value := strings.TrimSpace(resource.Spec.LogoFileBase64Encoded); value != "" {
		body.LogoFileBase64Encoded = common.String(value)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = clonePrivateApplicationStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildPrivateApplicationUpdateBody(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	_ string,
	currentResponse any,
	logoClient privateApplicationPackageOCIClient,
	logoClientInitErr error,
) (any, bool, error) {
	if resource == nil {
		return servicecatalogsdk.UpdatePrivateApplicationDetails{}, false, fmt.Errorf("PrivateApplication resource is nil")
	}
	if err := validatePrivateApplicationSpec(resource.Spec); err != nil {
		return servicecatalogsdk.UpdatePrivateApplicationDetails{}, false, err
	}

	current, ok := privateApplicationFromResponse(currentResponse)
	if !ok {
		return servicecatalogsdk.UpdatePrivateApplicationDetails{}, false, fmt.Errorf("current PrivateApplication response does not expose a PrivateApplication body")
	}
	if err := validatePrivateApplicationCreateOnlyDrift(resource.Spec, current); err != nil {
		return servicecatalogsdk.UpdatePrivateApplicationDetails{}, false, err
	}

	updateDetails := servicecatalogsdk.UpdatePrivateApplicationDetails{}
	updateNeeded := false
	updateNeeded = applyPrivateApplicationStringUpdates(&updateDetails, resource.Spec, current) || updateNeeded
	logoUpdateNeeded, err := applyPrivateApplicationLogoUpdate(ctx, &updateDetails, resource, current, logoClient, logoClientInitErr)
	if err != nil {
		return servicecatalogsdk.UpdatePrivateApplicationDetails{}, false, err
	}
	updateNeeded = logoUpdateNeeded || updateNeeded
	updateNeeded = applyPrivateApplicationTagUpdates(&updateDetails, resource.Spec, current) || updateNeeded

	if !updateNeeded {
		return servicecatalogsdk.UpdatePrivateApplicationDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func applyPrivateApplicationStringUpdates(
	updateDetails *servicecatalogsdk.UpdatePrivateApplicationDetails,
	spec servicecatalogv1beta1.PrivateApplicationSpec,
	current servicecatalogsdk.PrivateApplication,
) bool {
	updateNeeded := false
	if !stringPtrEqual(current.DisplayName, spec.DisplayName) {
		updateDetails.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
		updateNeeded = true
	}
	if !stringPtrEqual(current.ShortDescription, spec.ShortDescription) {
		updateDetails.ShortDescription = common.String(strings.TrimSpace(spec.ShortDescription))
		updateNeeded = true
	}
	if desired := strings.TrimSpace(spec.LongDescription); desired != "" && !stringPtrEqual(current.LongDescription, desired) {
		updateDetails.LongDescription = common.String(desired)
		updateNeeded = true
	}
	return updateNeeded
}

func applyPrivateApplicationLogoUpdate(
	ctx context.Context,
	updateDetails *servicecatalogsdk.UpdatePrivateApplicationDetails,
	resource *servicecatalogv1beta1.PrivateApplication,
	current servicecatalogsdk.PrivateApplication,
	logoClient privateApplicationPackageOCIClient,
	logoClientInitErr error,
) (bool, error) {
	desired := strings.TrimSpace(resource.Spec.LogoFileBase64Encoded)
	if desired == "" {
		return false, nil
	}
	if current.Logo != nil {
		matches, err := currentPrivateApplicationLogoMatches(ctx, resource, current, desired, logoClient, logoClientInitErr)
		if err != nil {
			return false, err
		}
		if matches {
			return false, nil
		}
	}
	updateDetails.LogoFileBase64Encoded = common.String(desired)
	return true, nil
}

func currentPrivateApplicationLogoMatches(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	current servicecatalogsdk.PrivateApplication,
	desiredLogo string,
	logoClient privateApplicationPackageOCIClient,
	logoClientInitErr error,
) (bool, error) {
	if logoClientInitErr != nil {
		return false, logoClientInitErr
	}
	if logoClient == nil {
		return false, fmt.Errorf("PrivateApplication logo drift validation requires a logo OCI client")
	}
	privateApplicationID := strings.TrimSpace(stringPtrValue(current.Id))
	if privateApplicationID == "" {
		return false, fmt.Errorf("PrivateApplication logo drift validation could not resolve the current private application ID")
	}

	response, err := logoClient.GetPrivateApplicationActionDownloadLogo(ctx, servicecatalogsdk.GetPrivateApplicationActionDownloadLogoRequest{
		PrivateApplicationId: common.String(privateApplicationID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, fmt.Errorf("download PrivateApplication logo for drift validation: %w", err)
	}
	if response.Content == nil {
		return false, fmt.Errorf("download PrivateApplication logo for drift validation: empty response body")
	}
	defer func() {
		_ = response.Content.Close()
	}()

	currentLogo, err := io.ReadAll(response.Content)
	if err != nil {
		return false, fmt.Errorf("read PrivateApplication logo for drift validation: %w", err)
	}
	return privateApplicationBase64PayloadMatches(currentLogo, desiredLogo), nil
}

func applyPrivateApplicationTagUpdates(
	updateDetails *servicecatalogsdk.UpdatePrivateApplicationDetails,
	spec servicecatalogv1beta1.PrivateApplicationSpec,
	current servicecatalogsdk.PrivateApplication,
) bool {
	updateNeeded := false
	desiredFreeformTags := desiredPrivateApplicationFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredPrivateApplicationDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}
	return updateNeeded
}

func validatePrivateApplicationSpec(spec servicecatalogv1beta1.PrivateApplicationSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.ShortDescription) == "" {
		missing = append(missing, "shortDescription")
	}
	if _, err := privateApplicationPackagePayloadFromSpec(spec.PackageDetails); err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("PrivateApplication spec is missing required field(s): %s", strings.Join(missing, ", "))
}

type privateApplicationPackagePayload struct {
	PackageType          string
	Version              string
	ZipFileBase64Encoded string
}

func privateApplicationPackagePayloadFromSpec(
	spec servicecatalogv1beta1.PrivateApplicationPackageDetails,
) (privateApplicationPackagePayload, error) {
	payload, err := privateApplicationPackagePayloadFromJSONData(spec.JsonData)
	if err != nil {
		return privateApplicationPackagePayload{}, err
	}
	payload.applySpecFallbacks(spec)
	if strings.TrimSpace(payload.PackageType) == "" {
		payload.PackageType = string(servicecatalogsdk.PackageTypeEnumStack)
	}

	if err := payload.normalizePackageType(); err != nil {
		return privateApplicationPackagePayload{}, err
	}

	return payload.normalizedRequiredPayload()
}

func privateApplicationPackagePayloadFromJSONData(raw string) (privateApplicationPackagePayload, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return privateApplicationPackagePayload{}, nil
	}

	values := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return privateApplicationPackagePayload{}, fmt.Errorf("decode PrivateApplication packageDetails.jsonData: %w", err)
	}
	return privateApplicationPackagePayload{
		PackageType:          stringFromMap(values, "packageType"),
		Version:              stringFromMap(values, "version"),
		ZipFileBase64Encoded: stringFromMap(values, "zipFileBase64Encoded"),
	}, nil
}

func (p *privateApplicationPackagePayload) applySpecFallbacks(
	spec servicecatalogv1beta1.PrivateApplicationPackageDetails,
) {
	if p.PackageType == "" {
		p.PackageType = spec.PackageType
	}
	if p.Version == "" {
		p.Version = spec.Version
	}
	if p.ZipFileBase64Encoded == "" {
		p.ZipFileBase64Encoded = spec.ZipFileBase64Encoded
	}
}

func (p *privateApplicationPackagePayload) normalizePackageType() error {
	packageType, ok := servicecatalogsdk.GetMappingPackageTypeEnumEnum(p.PackageType)
	if !ok {
		return fmt.Errorf("unsupported PrivateApplication packageDetails.packageType %q", p.PackageType)
	}
	if packageType != servicecatalogsdk.PackageTypeEnumStack {
		return fmt.Errorf("PrivateApplication packageDetails.packageType %q is not supported by the OCI create-private-application SDK shape", packageType)
	}
	p.PackageType = string(packageType)
	return nil
}

func (p privateApplicationPackagePayload) normalizedRequiredPayload() (privateApplicationPackagePayload, error) {
	var missing []string
	if strings.TrimSpace(p.Version) == "" {
		missing = append(missing, "packageDetails.version")
	}
	if strings.TrimSpace(p.ZipFileBase64Encoded) == "" {
		missing = append(missing, "packageDetails.zipFileBase64Encoded")
	}
	if len(missing) > 0 {
		return privateApplicationPackagePayload{}, fmt.Errorf("PrivateApplication spec is missing required field(s): %s", strings.Join(missing, ", "))
	}

	p.Version = strings.TrimSpace(p.Version)
	p.ZipFileBase64Encoded = strings.TrimSpace(p.ZipFileBase64Encoded)
	return p, nil
}

func sdkPrivateApplicationPackage(
	spec servicecatalogv1beta1.PrivateApplicationPackageDetails,
) (servicecatalogsdk.CreatePrivateApplicationPackage, error) {
	payload, err := privateApplicationPackagePayloadFromSpec(spec)
	if err != nil {
		return nil, err
	}
	return servicecatalogsdk.CreatePrivateApplicationStackPackage{
		Version:              common.String(payload.Version),
		ZipFileBase64Encoded: common.String(payload.ZipFileBase64Encoded),
	}, nil
}

func validatePrivateApplicationCreateOnlyDriftForResponse(
	resource *servicecatalogv1beta1.PrivateApplication,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("PrivateApplication resource is nil")
	}
	current, ok := privateApplicationFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current PrivateApplication response does not expose a PrivateApplication body")
	}
	return validatePrivateApplicationCreateOnlyDrift(resource.Spec, current)
}

func validatePrivateApplicationCreateOnlyDrift(
	spec servicecatalogv1beta1.PrivateApplicationSpec,
	current servicecatalogsdk.PrivateApplication,
) error {
	var drift []string
	if !stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	desiredPackage, err := privateApplicationPackagePayloadFromSpec(spec.PackageDetails)
	if err != nil {
		return err
	}
	if current.PackageType != "" && string(current.PackageType) != desiredPackage.PackageType {
		drift = append(drift, "packageDetails.packageType")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("PrivateApplication create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func guardPrivateApplicationExistingBeforeCreate(
	_ context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("PrivateApplication resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func wrapPrivateApplicationReadAndDeleteCalls(hooks *PrivateApplicationRuntimeHooks) {
	getCall := hooks.Get.Call
	if getCall != nil {
		hooks.Get.Call = func(ctx context.Context, request servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativePrivateApplicationNotFoundError(err, "read")
		}
	}

	listCall := hooks.List.Call
	if listCall != nil {
		hooks.List.Call = func(ctx context.Context, request servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error) {
			return listPrivateApplicationsAllPages(ctx, listCall, request)
		}
	}

	deleteCall := hooks.Delete.Call
	if deleteCall != nil {
		hooks.Delete.Call = func(ctx context.Context, request servicecatalogsdk.DeletePrivateApplicationRequest) (servicecatalogsdk.DeletePrivateApplicationResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativePrivateApplicationNotFoundError(err, "delete")
		}
	}
}

func listPrivateApplicationsAllPages(
	ctx context.Context,
	list func(context.Context, servicecatalogsdk.ListPrivateApplicationsRequest) (servicecatalogsdk.ListPrivateApplicationsResponse, error),
	request servicecatalogsdk.ListPrivateApplicationsRequest,
) (servicecatalogsdk.ListPrivateApplicationsResponse, error) {
	var combined servicecatalogsdk.ListPrivateApplicationsResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return servicecatalogsdk.ListPrivateApplicationsResponse{}, conservativePrivateApplicationNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == servicecatalogsdk.PrivateApplicationLifecycleStateDeleted {
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

func handlePrivateApplicationDeleteError(resource *servicecatalogv1beta1.PrivateApplication, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func wrapPrivateApplicationPackageDriftGuard(
	hooks *PrivateApplicationRuntimeHooks,
	packageClient privateApplicationPackageOCIClient,
	initErr error,
	log loggerutil.OSOKLogger,
) {
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate PrivateApplicationServiceClient) PrivateApplicationServiceClient {
		return privateApplicationPackageDriftGuardClient{
			delegate:      delegate,
			packageClient: packageClient,
			initErr:       initErr,
			log:           log,
		}
	})
}

type privateApplicationPackageDriftGuardClient struct {
	delegate      PrivateApplicationServiceClient
	packageClient privateApplicationPackageOCIClient
	initErr       error
	log           loggerutil.OSOKLogger
}

func (c privateApplicationPackageDriftGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if shouldDefer, err := c.validatePackageDriftBeforeDelegate(ctx, resource); shouldDefer || err != nil {
		if err != nil {
			markPrivateApplicationFailure(resource, err, c.log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c privateApplicationPackageDriftGuardClient) validatePackageDriftBeforeDelegate(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	if shouldDefer, err := c.shouldDeferPackageDriftValidation(resource); shouldDefer || err != nil {
		return shouldDefer, err
	}
	if shouldDefer, err := c.validateUntrackedBindPackage(ctx, resource); shouldDefer || err != nil {
		return shouldDefer, err
	}
	return c.validateTrackedPackageWithPendingRetry(ctx, resource)
}

func (c privateApplicationPackageDriftGuardClient) validateTrackedPackageWithPendingRetry(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	if err := c.validateTrackedPackage(ctx, resource); err != nil {
		if shouldDefer, deferErr := c.shouldDeferPackageDriftValidationAfterError(ctx, resource); shouldDefer || deferErr != nil {
			if deferErr != nil {
				return false, deferErr
			}
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (c privateApplicationPackageDriftGuardClient) Delete(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c privateApplicationPackageDriftGuardClient) validateTrackedPackage(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) error {
	privateApplicationID := trackedPrivateApplicationID(resource)
	if privateApplicationID == "" {
		return nil
	}
	return c.validatePackageForID(ctx, resource, privateApplicationID)
}

func (c privateApplicationPackageDriftGuardClient) validateUntrackedBindPackage(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("PrivateApplication resource is nil")
	}
	if trackedPrivateApplicationID(resource) != "" {
		return false, nil
	}

	current, found, err := c.findExistingPrivateApplicationForBind(ctx, resource)
	if err != nil || !found {
		return false, err
	}
	if privateApplicationLifecycleDefersPackageDriftValidation(current.LifecycleState) {
		return true, nil
	}

	privateApplicationID := strings.TrimSpace(stringPtrValue(current.Id))
	if privateApplicationID == "" {
		return false, fmt.Errorf("PrivateApplication package drift validation could not resolve the current private application ID")
	}
	return false, c.validatePackageForID(ctx, resource, privateApplicationID)
}

func (c privateApplicationPackageDriftGuardClient) findExistingPrivateApplicationForBind(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (servicecatalogsdk.PrivateApplication, bool, error) {
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return servicecatalogsdk.PrivateApplication{}, false, nil
	}
	if c.initErr != nil {
		return servicecatalogsdk.PrivateApplication{}, false, c.initErr
	}
	if c.packageClient == nil {
		return servicecatalogsdk.PrivateApplication{}, false, fmt.Errorf("PrivateApplication package drift validation requires a package OCI client")
	}

	matches, err := c.listExistingPrivateApplicationBindMatches(ctx, resource)
	if err != nil {
		return servicecatalogsdk.PrivateApplication{}, false, err
	}
	return c.resolveExistingPrivateApplicationBindMatch(ctx, resource, matches)
}

func (c privateApplicationPackageDriftGuardClient) listExistingPrivateApplicationBindMatches(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) ([]servicecatalogsdk.PrivateApplicationSummary, error) {
	response, err := listPrivateApplicationsAllPages(ctx, c.packageClient.ListPrivateApplications, servicecatalogsdk.ListPrivateApplicationsRequest{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return nil, fmt.Errorf("list PrivateApplications before package drift validation: %w", err)
	}
	return matchingPrivateApplicationSummariesForBind(response.Items, resource.Spec), nil
}

func (c privateApplicationPackageDriftGuardClient) resolveExistingPrivateApplicationBindMatch(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	matches []servicecatalogsdk.PrivateApplicationSummary,
) (servicecatalogsdk.PrivateApplication, bool, error) {
	switch len(matches) {
	case 0:
		return servicecatalogsdk.PrivateApplication{}, false, nil
	case 1:
	default:
		return servicecatalogsdk.PrivateApplication{}, false, fmt.Errorf("PrivateApplication package drift validation found multiple matching private applications")
	}

	privateApplicationID := strings.TrimSpace(stringPtrValue(matches[0].Id))
	if privateApplicationID == "" {
		return servicecatalogsdk.PrivateApplication{}, false, fmt.Errorf("PrivateApplication package drift validation could not resolve the current private application ID")
	}
	current, found, err := c.readPrivateApplicationForPackageValidation(ctx, resource, privateApplicationID)
	if err != nil || !found {
		return servicecatalogsdk.PrivateApplication{}, false, err
	}
	if current.Id == nil {
		current.Id = common.String(privateApplicationID)
	}
	return current, true, nil
}

func matchingPrivateApplicationSummariesForBind(
	items []servicecatalogsdk.PrivateApplicationSummary,
	spec servicecatalogv1beta1.PrivateApplicationSpec,
) []servicecatalogsdk.PrivateApplicationSummary {
	var matches []servicecatalogsdk.PrivateApplicationSummary
	for _, item := range items {
		if !stringPtrEqual(item.CompartmentId, spec.CompartmentId) {
			continue
		}
		if !stringPtrEqual(item.DisplayName, spec.DisplayName) {
			continue
		}
		matches = append(matches, item)
	}
	return matches
}

func (c privateApplicationPackageDriftGuardClient) readPrivateApplicationForPackageValidation(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	privateApplicationID string,
) (servicecatalogsdk.PrivateApplication, bool, error) {
	response, err := c.packageClient.GetPrivateApplication(ctx, servicecatalogsdk.GetPrivateApplicationRequest{
		PrivateApplicationId: common.String(privateApplicationID),
	})
	if err == nil {
		return response.PrivateApplication, true, nil
	}
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return servicecatalogsdk.PrivateApplication{}, false, nil
	}
	err = conservativePrivateApplicationNotFoundError(err, "read")
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return servicecatalogsdk.PrivateApplication{}, false, fmt.Errorf("read PrivateApplication before package drift validation: %w", err)
}

func (c privateApplicationPackageDriftGuardClient) validatePackageForID(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	privateApplicationID string,
) error {
	if c.initErr != nil {
		return c.initErr
	}
	if c.packageClient == nil {
		return fmt.Errorf("PrivateApplication package drift validation requires a package OCI client")
	}
	desired, err := privateApplicationPackagePayloadFromSpec(resource.Spec.PackageDetails)
	if err != nil {
		return err
	}
	currentPackage, err := c.findPackageVersion(ctx, resource, privateApplicationID, desired)
	if err != nil {
		return err
	}
	return c.validatePackageConfigPayload(ctx, resource, currentPackage, desired)
}

func (c privateApplicationPackageDriftGuardClient) shouldDeferPackageDriftValidation(
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	if privateApplicationHasActiveCreateOrUpdate(resource) ||
		privateApplicationStatusLifecycleDefersPackageDriftValidation(resource) {
		return true, nil
	}
	return false, nil
}

func (c privateApplicationPackageDriftGuardClient) shouldDeferPackageDriftValidationAfterError(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	privateApplicationID := trackedPrivateApplicationID(resource)
	if privateApplicationID == "" || c.packageClient == nil {
		return false, nil
	}

	response, err := c.packageClient.GetPrivateApplication(ctx, servicecatalogsdk.GetPrivateApplicationRequest{
		PrivateApplicationId: common.String(privateApplicationID),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return true, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, fmt.Errorf("read PrivateApplication before package drift validation: %w", err)
	}
	return privateApplicationLifecycleDefersPackageDriftValidation(response.LifecycleState), nil
}

func privateApplicationStatusLifecycleDefersPackageDriftValidation(resource *servicecatalogv1beta1.PrivateApplication) bool {
	if resource == nil {
		return false
	}
	return privateApplicationLifecycleDefersPackageDriftValidation(
		servicecatalogsdk.PrivateApplicationLifecycleStateEnum(strings.TrimSpace(resource.Status.LifecycleState)),
	)
}

func privateApplicationLifecycleDefersPackageDriftValidation(state servicecatalogsdk.PrivateApplicationLifecycleStateEnum) bool {
	switch state {
	case servicecatalogsdk.PrivateApplicationLifecycleStateCreating,
		servicecatalogsdk.PrivateApplicationLifecycleStateUpdating,
		servicecatalogsdk.PrivateApplicationLifecycleStateDeleting:
		return true
	default:
		return false
	}
}

func (c privateApplicationPackageDriftGuardClient) findPackageVersion(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	privateApplicationID string,
	desired privateApplicationPackagePayload,
) (servicecatalogsdk.PrivateApplicationPackageSummary, error) {
	request := servicecatalogsdk.ListPrivateApplicationPackagesRequest{
		PrivateApplicationId: common.String(privateApplicationID),
		PackageType:          []servicecatalogsdk.PackageTypeEnumEnum{servicecatalogsdk.PackageTypeEnumStack},
	}
	for {
		response, err := c.packageClient.ListPrivateApplicationPackages(ctx, request)
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return servicecatalogsdk.PrivateApplicationPackageSummary{}, fmt.Errorf("list PrivateApplication packages for drift validation: %w", err)
		}
		for _, current := range response.Items {
			if privateApplicationPackageMatches(current, privateApplicationID, desired) {
				return current, nil
			}
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}

	return servicecatalogsdk.PrivateApplicationPackageSummary{}, fmt.Errorf(
		"PrivateApplication create-only field drift is not supported: packageDetails.version",
	)
}

func privateApplicationPackageMatches(
	current servicecatalogsdk.PrivateApplicationPackageSummary,
	privateApplicationID string,
	desired privateApplicationPackagePayload,
) bool {
	if current.PrivateApplicationId != nil && strings.TrimSpace(*current.PrivateApplicationId) != privateApplicationID {
		return false
	}
	return stringPtrEqual(current.Version, desired.Version) &&
		strings.EqualFold(strings.TrimSpace(string(current.PackageType)), desired.PackageType)
}

func (c privateApplicationPackageDriftGuardClient) validatePackageConfigPayload(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	current servicecatalogsdk.PrivateApplicationPackageSummary,
	desired privateApplicationPackagePayload,
) error {
	packageID := ""
	if current.Id != nil {
		packageID = strings.TrimSpace(*current.Id)
	}
	if packageID == "" {
		return fmt.Errorf("PrivateApplication package drift validation could not resolve the current package ID")
	}

	response, err := c.packageClient.GetPrivateApplicationPackageActionDownloadConfig(
		ctx,
		servicecatalogsdk.GetPrivateApplicationPackageActionDownloadConfigRequest{
			PrivateApplicationPackageId: common.String(packageID),
		},
	)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return fmt.Errorf("download PrivateApplication package config for drift validation: %w", err)
	}
	if response.Content == nil {
		return fmt.Errorf("download PrivateApplication package config for drift validation: empty response body")
	}
	defer func() {
		_ = response.Content.Close()
	}()

	currentPayload, err := io.ReadAll(response.Content)
	if err != nil {
		return fmt.Errorf("read PrivateApplication package config for drift validation: %w", err)
	}
	if !privateApplicationPackageConfigMatches(currentPayload, desired.ZipFileBase64Encoded) {
		return fmt.Errorf("PrivateApplication create-only field drift is not supported: packageDetails.zipFileBase64Encoded")
	}
	return nil
}

func privateApplicationPackageConfigMatches(currentPayload []byte, desiredZip string) bool {
	return privateApplicationBase64PayloadMatches(currentPayload, desiredZip)
}

func privateApplicationBase64PayloadMatches(currentPayload []byte, desiredPayload string) bool {
	current := strings.TrimSpace(string(currentPayload))
	desired := strings.TrimSpace(desiredPayload)
	if current == desired {
		return true
	}
	return strings.TrimSpace(base64.StdEncoding.EncodeToString(currentPayload)) == desired
}

func markPrivateApplicationFailure(
	resource *servicecatalogv1beta1.PrivateApplication,
	err error,
	log loggerutil.OSOKLogger,
) {
	if resource == nil || err == nil {
		return
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), log)
}

func wrapPrivateApplicationDeleteConfirmation(hooks *PrivateApplicationRuntimeHooks, log loggerutil.OSOKLogger) {
	if hooks.Get.Call == nil {
		return
	}
	getPrivateApplication := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate PrivateApplicationServiceClient) PrivateApplicationServiceClient {
		return privateApplicationDeleteConfirmationClient{
			delegate:              delegate,
			getPrivateApplication: getPrivateApplication,
			log:                   log,
		}
	})
}

type privateApplicationDeleteConfirmationClient struct {
	delegate              PrivateApplicationServiceClient
	getPrivateApplication func(context.Context, servicecatalogsdk.GetPrivateApplicationRequest) (servicecatalogsdk.GetPrivateApplicationResponse, error)
	log                   loggerutil.OSOKLogger
}

func (c privateApplicationDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c privateApplicationDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	if blocked, err := c.rejectUnsafeDeleteStart(ctx, resource); blocked || err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c privateApplicationDeleteConfirmationClient) rejectUnsafeDeleteStart(
	ctx context.Context,
	resource *servicecatalogv1beta1.PrivateApplication,
) (bool, error) {
	if privateApplicationHasActiveCreateOrUpdate(resource) {
		return true, nil
	}
	if c.getPrivateApplication == nil || resource == nil {
		return false, nil
	}
	privateApplicationID := trackedPrivateApplicationID(resource)
	if privateApplicationID == "" {
		return false, nil
	}
	response, err := c.getPrivateApplication(ctx, servicecatalogsdk.GetPrivateApplicationRequest{
		PrivateApplicationId: common.String(privateApplicationID),
	})
	if err == nil {
		current := response.PrivateApplication
		projectPrivateApplicationDeleteReadback(resource, current, c.log)
		if privateApplicationLifecycleBlocksDelete(current.LifecycleState) {
			return true, nil
		}
		return false, nil
	}
	if !isAmbiguousPrivateApplicationNotFound(err) {
		return false, nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return false, fmt.Errorf("PrivateApplication delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func privateApplicationHasActiveCreateOrUpdate(resource *servicecatalogv1beta1.PrivateApplication) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
	default:
		return false
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassSucceeded, shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled:
		return false
	default:
		return true
	}
}

func privateApplicationLifecycleBlocksDelete(state servicecatalogsdk.PrivateApplicationLifecycleStateEnum) bool {
	switch state {
	case servicecatalogsdk.PrivateApplicationLifecycleStateCreating,
		servicecatalogsdk.PrivateApplicationLifecycleStateUpdating:
		return true
	default:
		return false
	}
}

func projectPrivateApplicationDeleteReadback(
	resource *servicecatalogv1beta1.PrivateApplication,
	current servicecatalogsdk.PrivateApplication,
	log loggerutil.OSOKLogger,
) {
	if resource == nil {
		return
	}
	if id := strings.TrimSpace(stringPtrValue(current.Id)); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	resource.Status.LifecycleState = strings.TrimSpace(string(current.LifecycleState))
	if state := strings.TrimSpace(resource.Status.LifecycleState); state != "" {
		message := fmt.Sprintf("OCI PrivateApplication is %s; waiting before delete", state)
		currentAsync := servicemanager.NewLifecycleAsyncOperation(
			&resource.Status.OsokStatus,
			state,
			message,
			"",
		)
		if currentAsync != nil {
			servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, log)
		}
	}
}

func conservativePrivateApplicationNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("PrivateApplication %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousPrivateApplicationNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousPrivateApplicationNotFoundError{message: message}
}

func isAmbiguousPrivateApplicationNotFound(err error) bool {
	var ambiguous ambiguousPrivateApplicationNotFoundError
	return errors.As(err, &ambiguous)
}

func clearTrackedPrivateApplicationIdentity(resource *servicecatalogv1beta1.PrivateApplication) {
	if resource == nil {
		return
	}
	resource.Status = servicecatalogv1beta1.PrivateApplicationStatus{}
}

func trackedPrivateApplicationID(resource *servicecatalogv1beta1.PrivateApplication) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func privateApplicationFromResponse(response any) (servicecatalogsdk.PrivateApplication, bool) {
	if current, ok := privateApplicationFromWriteResponse(response); ok {
		return current, true
	}
	if current, ok := privateApplicationFromReadResponse(response); ok {
		return current, true
	}
	return privateApplicationFromListItem(response)
}

func privateApplicationFromWriteResponse(response any) (servicecatalogsdk.PrivateApplication, bool) {
	switch current := response.(type) {
	case servicecatalogsdk.CreatePrivateApplicationResponse:
		return current.PrivateApplication, true
	case *servicecatalogsdk.CreatePrivateApplicationResponse:
		if current == nil {
			return servicecatalogsdk.PrivateApplication{}, false
		}
		return current.PrivateApplication, true
	case servicecatalogsdk.UpdatePrivateApplicationResponse:
		return current.PrivateApplication, true
	case *servicecatalogsdk.UpdatePrivateApplicationResponse:
		if current == nil {
			return servicecatalogsdk.PrivateApplication{}, false
		}
		return current.PrivateApplication, true
	default:
		return servicecatalogsdk.PrivateApplication{}, false
	}
}

func privateApplicationFromReadResponse(response any) (servicecatalogsdk.PrivateApplication, bool) {
	switch current := response.(type) {
	case servicecatalogsdk.GetPrivateApplicationResponse:
		return current.PrivateApplication, true
	case *servicecatalogsdk.GetPrivateApplicationResponse:
		if current == nil {
			return servicecatalogsdk.PrivateApplication{}, false
		}
		return current.PrivateApplication, true
	case servicecatalogsdk.PrivateApplication:
		return current, true
	case *servicecatalogsdk.PrivateApplication:
		if current == nil {
			return servicecatalogsdk.PrivateApplication{}, false
		}
		return *current, true
	default:
		return servicecatalogsdk.PrivateApplication{}, false
	}
}

func privateApplicationFromListItem(response any) (servicecatalogsdk.PrivateApplication, bool) {
	switch current := response.(type) {
	case servicecatalogsdk.PrivateApplicationSummary:
		return privateApplicationFromSummary(current), true
	case *servicecatalogsdk.PrivateApplicationSummary:
		if current == nil {
			return servicecatalogsdk.PrivateApplication{}, false
		}
		return privateApplicationFromSummary(*current), true
	default:
		return servicecatalogsdk.PrivateApplication{}, false
	}
}

func privateApplicationFromSummary(summary servicecatalogsdk.PrivateApplicationSummary) servicecatalogsdk.PrivateApplication {
	return servicecatalogsdk.PrivateApplication{
		LifecycleState:   summary.LifecycleState,
		CompartmentId:    summary.CompartmentId,
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		PackageType:      summary.PackageType,
		TimeCreated:      summary.TimeCreated,
		ShortDescription: summary.ShortDescription,
		Logo:             summary.Logo,
		DefinedTags:      clonePrivateApplicationDefinedTags(summary.DefinedTags),
		FreeformTags:     clonePrivateApplicationStringMap(summary.FreeformTags),
		SystemTags:       clonePrivateApplicationDefinedTags(summary.SystemTags),
	}
}

func desiredPrivateApplicationFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return clonePrivateApplicationStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredPrivateApplicationDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func clonePrivateApplicationStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func clonePrivateApplicationDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		cloned[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			cloned[namespace][key] = value
		}
	}
	return cloned
}

func stringPtrEqual(value *string, desired string) bool {
	if value == nil {
		return strings.TrimSpace(desired) == ""
	}
	return strings.TrimSpace(*value) == strings.TrimSpace(desired)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringFromMap(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
