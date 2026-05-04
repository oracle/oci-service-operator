/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package productlicense

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	licensemanagersdk "github.com/oracle/oci-go-sdk/v65/licensemanager"
	licensemanagerv1beta1 "github.com/oracle/oci-service-operator/api/licensemanager/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const productLicenseRequeueDuration = time.Minute

type productLicenseOCIClient interface {
	CreateProductLicense(context.Context, licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error)
	GetProductLicense(context.Context, licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error)
	ListProductLicenses(context.Context, licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error)
	UpdateProductLicense(context.Context, licensemanagersdk.UpdateProductLicenseRequest) (licensemanagersdk.UpdateProductLicenseResponse, error)
	DeleteProductLicense(context.Context, licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error)
}

type productLicenseRuntimeClient struct {
	hooks   ProductLicenseRuntimeHooks
	initErr error
	log     loggerutil.OSOKLogger
}

type productLicenseAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e productLicenseAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e productLicenseAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerProductLicenseRuntimeHooksMutator(func(manager *ProductLicenseServiceManager, hooks *ProductLicenseRuntimeHooks) {
		client, initErr := newProductLicenseSDKClient(manager)
		applyProductLicenseRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newProductLicenseSDKClient(manager *ProductLicenseServiceManager) (productLicenseOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ProductLicense service manager is nil")
	}
	client, err := licensemanagersdk.NewLicenseManagerClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyProductLicenseRuntimeHooks(
	manager *ProductLicenseServiceManager,
	hooks *ProductLicenseRuntimeHooks,
	client productLicenseOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newProductLicenseRuntimeSemantics()
	hooks.BuildCreateBody = buildProductLicenseCreateBody
	hooks.BuildUpdateBody = buildProductLicenseUpdateBody
	hooks.List.Fields = productLicenseListFields()
	hooks.StatusHooks.ProjectStatus = projectProductLicenseStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateProductLicenseCreateOnlyDriftForResponse
	hooks.Create.Call = func(ctx context.Context, request licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error) {
		if err := requireProductLicenseOCIClient(client, initErr); err != nil {
			return licensemanagersdk.CreateProductLicenseResponse{}, err
		}
		return client.CreateProductLicense(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		if err := requireProductLicenseOCIClient(client, initErr); err != nil {
			return licensemanagersdk.GetProductLicenseResponse{}, err
		}
		return client.GetProductLicense(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error) {
		return listProductLicensePages(ctx, client, initErr, request)
	}
	hooks.Update.Call = func(ctx context.Context, request licensemanagersdk.UpdateProductLicenseRequest) (licensemanagersdk.UpdateProductLicenseResponse, error) {
		if err := requireProductLicenseOCIClient(client, initErr); err != nil {
			return licensemanagersdk.UpdateProductLicenseResponse{}, err
		}
		return client.UpdateProductLicense(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
		if err := requireProductLicenseOCIClient(client, initErr); err != nil {
			return licensemanagersdk.DeleteProductLicenseResponse{}, err
		}
		return client.DeleteProductLicense(ctx, request)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(ProductLicenseServiceClient) ProductLicenseServiceClient {
		var log loggerutil.OSOKLogger
		if manager != nil {
			log = manager.Log
		}
		return &productLicenseRuntimeClient{
			hooks:   *hooks,
			initErr: initErr,
			log:     log,
		}
	})
}

func requireProductLicenseOCIClient(client productLicenseOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize ProductLicense OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("ProductLicense OCI client is not configured")
	}
	return nil
}

func newProductLicenseRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "licensemanager",
		FormalSlug:        "productlicense",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(licensemanagersdk.LifeCycleStateActive),
				string(licensemanagersdk.LifeCycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(licensemanagersdk.LifeCycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"images", "freeformTags", "definedTags"},
			Mutable:         []string{"images", "freeformTags", "definedTags"},
			ForceNew:        []string{"compartmentId", "displayName", "isVendorOracle", "licenseUnit", "vendorName"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ProductLicense", Action: "CreateProductLicense"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ProductLicense", Action: "UpdateProductLicense"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ProductLicense", Action: "DeleteProductLicense"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func productLicenseListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "IsCompartmentIdInSubtree", RequestName: "isCompartmentIdInSubtree", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func (c *productLicenseRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.failCreateOrUpdate(resource, c.initErr)
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ProductLicense resource is nil")
	}
	if err := validateProductLicenseRequiredSpec(resource.Spec); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	current, found, err := c.resolveCurrentProductLicense(ctx, resource)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	namespace := resource.Namespace
	if strings.TrimSpace(namespace) == "" {
		namespace = req.Namespace
	}
	if found {
		return c.reconcileProductLicense(ctx, resource, namespace, current)
	}
	return c.createProductLicense(ctx, resource, namespace)
}

func (c *productLicenseRuntimeClient) Delete(ctx context.Context, resource *licensemanagerv1beta1.ProductLicense) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}
	if resource == nil {
		return false, fmt.Errorf("ProductLicense resource is nil")
	}

	currentID := productLicenseRecordedID(resource)
	if currentID == "" {
		markProductLicenseDeleted(resource, "OCI resource identifier is not recorded", c.log)
		return true, nil
	}

	if productLicenseDeleteInProgress(resource) {
		return c.confirmProductLicenseDelete(ctx, resource, currentID)
	}

	current, err := c.getProductLicense(ctx, currentID)
	if err != nil {
		return handleProductLicensePreDeleteReadError(resource, err, c.log)
	}
	projectProductLicenseStatus(resource, current)
	if productLicenseTerminalDeleted(current) {
		markProductLicenseDeleted(resource, "OCI resource already deleted", c.log)
		return true, nil
	}

	response, err := c.hooks.Delete.Call(ctx, licensemanagersdk.DeleteProductLicenseRequest{
		ProductLicenseId: common.String(currentID),
	})
	if err != nil {
		return handleProductLicenseDeleteCallError(resource, err, c.log)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if workRequestID := stringValue(response.OpcWorkRequestId); workRequestID != "" {
		markProductLicenseDeleteWorkRequest(resource, workRequestID, c.log)
	}
	return c.confirmProductLicenseDelete(ctx, resource, currentID)
}

func (c *productLicenseRuntimeClient) resolveCurrentProductLicense(
	ctx context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
) (licensemanagersdk.ProductLicense, bool, error) {
	if currentID := productLicenseRecordedID(resource); currentID != "" {
		current, found, err := c.getResolvableProductLicense(ctx, currentID)
		if err != nil || found {
			return current, found, err
		}
	}
	return c.resolveListedProductLicense(ctx, resource)
}

func (c *productLicenseRuntimeClient) resolveListedProductLicense(
	ctx context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
) (licensemanagersdk.ProductLicense, bool, error) {
	summary, found, err := c.lookupProductLicenseBySpec(ctx, resource)
	if err != nil || !found {
		return licensemanagersdk.ProductLicense{}, false, err
	}
	resolvedID := productLicenseSummaryID(summary)
	if resolvedID == "" {
		return licensemanagersdk.ProductLicense{}, false, fmt.Errorf("ProductLicense list lookup could not resolve a resource OCID")
	}
	return c.getResolvableProductLicense(ctx, resolvedID)
}

func (c *productLicenseRuntimeClient) getResolvableProductLicense(
	ctx context.Context,
	currentID string,
) (licensemanagersdk.ProductLicense, bool, error) {
	current, err := c.getProductLicense(ctx, currentID)
	if err != nil {
		if productLicenseUnambiguousNotFound(err) {
			return licensemanagersdk.ProductLicense{}, false, nil
		}
		return licensemanagersdk.ProductLicense{}, false, err
	}
	if productLicenseTerminalDeleted(current) {
		return licensemanagersdk.ProductLicense{}, false, nil
	}
	return current, true, nil
}

func (c *productLicenseRuntimeClient) lookupProductLicenseBySpec(
	ctx context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
) (licensemanagersdk.ProductLicenseSummary, bool, error) {
	response, err := c.hooks.List.Call(ctx, licensemanagersdk.ListProductLicensesRequest{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
	})
	if err != nil {
		return licensemanagersdk.ProductLicenseSummary{}, false, err
	}

	matches, drift := collectProductLicenseLookupCandidates(response.Items, resource.Spec)
	return selectProductLicenseLookupMatch(matches, drift, resource.Spec)
}

func collectProductLicenseLookupCandidates(
	items []licensemanagersdk.ProductLicenseSummary,
	spec licensemanagerv1beta1.ProductLicenseSpec,
) ([]licensemanagersdk.ProductLicenseSummary, []string) {
	var matches []licensemanagersdk.ProductLicenseSummary
	var drift []string
	for _, item := range items {
		if !productLicenseSummaryBindKeyMatchesSpec(item, spec) {
			continue
		}
		if err := validateProductLicenseCreateOnlyDriftForSummary(spec, item); err != nil {
			drift = append(drift, err.Error())
			continue
		}
		matches = append(matches, item)
	}
	return matches, drift
}

func selectProductLicenseLookupMatch(
	matches []licensemanagersdk.ProductLicenseSummary,
	drift []string,
	spec licensemanagerv1beta1.ProductLicenseSpec,
) (licensemanagersdk.ProductLicenseSummary, bool, error) {
	switch {
	case len(matches) == 1:
		return matches[0], true, nil
	case len(matches) > 1:
		return licensemanagersdk.ProductLicenseSummary{}, false, fmt.Errorf("ProductLicense list response returned multiple matching resources for compartmentId %q and displayName %q", strings.TrimSpace(spec.CompartmentId), strings.TrimSpace(spec.DisplayName))
	case len(drift) > 0:
		return licensemanagersdk.ProductLicenseSummary{}, false, fmt.Errorf("ProductLicense create-or-bind found existing resource with create-only field drift for compartmentId %q and displayName %q: %s", strings.TrimSpace(spec.CompartmentId), strings.TrimSpace(spec.DisplayName), strings.Join(drift, "; "))
	default:
		return licensemanagersdk.ProductLicenseSummary{}, false, nil
	}
}

func productLicenseSummaryBindKeyMatchesSpec(
	item licensemanagersdk.ProductLicenseSummary,
	spec licensemanagerv1beta1.ProductLicenseSpec,
) bool {
	if productLicenseSummaryTerminalDeleted(item) {
		return false
	}
	return strings.TrimSpace(stringValue(item.CompartmentId)) == strings.TrimSpace(spec.CompartmentId) &&
		strings.TrimSpace(stringValue(item.DisplayName)) == strings.TrimSpace(spec.DisplayName)
}

func validateProductLicenseCreateOnlyDriftForSummary(
	spec licensemanagerv1beta1.ProductLicenseSpec,
	item licensemanagersdk.ProductLicenseSummary,
) error {
	resource := &licensemanagerv1beta1.ProductLicense{Spec: spec}
	return validateProductLicenseCreateOnlyDrift(resource, productLicenseFromSummary(item))
}

func (c *productLicenseRuntimeClient) reconcileProductLicense(
	ctx context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
	namespace string,
	current licensemanagersdk.ProductLicense,
) (servicemanager.OSOKResponse, error) {
	projectProductLicenseStatus(resource, current)
	if err := validateProductLicenseCreateOnlyDrift(resource, current); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	body, updateNeeded, err := buildProductLicenseUpdateBody(ctx, resource, namespace, current)
	if err != nil {
		return c.failCreateOrUpdate(resource, fmt.Errorf("build ProductLicense update body: %w", err))
	}
	if !updateNeeded {
		return markProductLicenseSuccess(resource, current, shared.Active, c.log), nil
	}
	details, ok := body.(licensemanagersdk.UpdateProductLicenseDetails)
	if !ok {
		return c.failCreateOrUpdate(resource, fmt.Errorf("ProductLicense update body has unexpected type %T", body))
	}
	currentID := productLicenseID(current)
	if currentID == "" {
		currentID = productLicenseRecordedID(resource)
	}
	if currentID == "" {
		return c.failCreateOrUpdate(resource, fmt.Errorf("ProductLicense update could not resolve a resource OCID"))
	}
	response, err := c.hooks.Update.Call(ctx, licensemanagersdk.UpdateProductLicenseRequest{
		ProductLicenseId:            common.String(currentID),
		UpdateProductLicenseDetails: details,
	})
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	updated := response.ProductLicense
	if refreshed, err := c.getProductLicense(ctx, currentID); err == nil {
		updated = refreshed
	} else if !productLicenseUnambiguousNotFound(err) {
		return c.failCreateOrUpdate(resource, err)
	}
	projectProductLicenseStatus(resource, updated)
	return markProductLicenseSuccess(resource, updated, shared.Updating, c.log), nil
}

func (c *productLicenseRuntimeClient) createProductLicense(
	ctx context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
	namespace string,
) (servicemanager.OSOKResponse, error) {
	body, err := buildProductLicenseCreateBody(ctx, resource, namespace)
	if err != nil {
		return c.failCreateOrUpdate(resource, fmt.Errorf("build ProductLicense create body: %w", err))
	}
	details, ok := body.(licensemanagersdk.CreateProductLicenseDetails)
	if !ok {
		return c.failCreateOrUpdate(resource, fmt.Errorf("ProductLicense create body has unexpected type %T", body))
	}
	response, err := c.hooks.Create.Call(ctx, licensemanagersdk.CreateProductLicenseRequest{
		CreateProductLicenseDetails: details,
		OpcRetryToken:               productLicenseRetryToken(resource, namespace),
	})
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	created := response.ProductLicense
	projectProductLicenseStatus(resource, created)
	if workRequestID := stringValue(response.OpcWorkRequestId); workRequestID != "" {
		markProductLicenseCreateWorkRequest(resource, workRequestID, c.log)
	}
	if currentID := productLicenseID(created); currentID != "" {
		if refreshed, err := c.getProductLicense(ctx, currentID); err == nil {
			created = refreshed
		} else if !productLicenseUnambiguousNotFound(err) {
			return c.failCreateOrUpdate(resource, err)
		}
	}
	projectProductLicenseStatus(resource, created)
	return markProductLicenseSuccess(resource, created, shared.Provisioning, c.log), nil
}

func (c *productLicenseRuntimeClient) confirmProductLicenseDelete(
	ctx context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
	currentID string,
) (bool, error) {
	confirmed, err := c.getProductLicense(ctx, currentID)
	if err != nil {
		return handleProductLicensePostDeleteReadError(resource, err, c.log)
	}
	projectProductLicenseStatus(resource, confirmed)
	if productLicenseTerminalDeleted(confirmed) {
		markProductLicenseDeleted(resource, "OCI resource deleted", c.log)
		return true, nil
	}
	markProductLicenseTerminating(resource, "OCI resource delete is in progress", c.log)
	return false, nil
}

func (c *productLicenseRuntimeClient) getProductLicense(ctx context.Context, currentID string) (licensemanagersdk.ProductLicense, error) {
	response, err := c.hooks.Get.Call(ctx, licensemanagersdk.GetProductLicenseRequest{
		ProductLicenseId: common.String(strings.TrimSpace(currentID)),
	})
	if err != nil {
		return licensemanagersdk.ProductLicense{}, normalizeProductLicenseNotFoundError(err, "read")
	}
	return response.ProductLicense, nil
}

func (c *productLicenseRuntimeClient) failCreateOrUpdate(
	resource *licensemanagerv1beta1.ProductLicense,
	err error,
) (servicemanager.OSOKResponse, error) {
	markProductLicenseFailure(resource, err, c.log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func buildProductLicenseCreateBody(
	_ context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ProductLicense resource is nil")
	}
	if err := validateProductLicenseRequiredSpec(resource.Spec); err != nil {
		return nil, err
	}
	images, err := productLicenseImageDetailsFromSpec(resource.Spec.Images)
	if err != nil {
		return nil, err
	}

	details := licensemanagersdk.CreateProductLicenseDetails{
		CompartmentId:  common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		IsVendorOracle: common.Bool(resource.Spec.IsVendorOracle),
		DisplayName:    common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		LicenseUnit:    licensemanagersdk.LicenseUnitEnum(strings.TrimSpace(resource.Spec.LicenseUnit)),
	}
	if vendorName := strings.TrimSpace(resource.Spec.VendorName); vendorName != "" {
		details.VendorName = common.String(vendorName)
	}
	if resource.Spec.Images != nil {
		details.Images = images
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildProductLicenseUpdateBody(
	_ context.Context,
	resource *licensemanagerv1beta1.ProductLicense,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return licensemanagersdk.UpdateProductLicenseDetails{}, false, fmt.Errorf("ProductLicense resource is nil")
	}
	current, ok := productLicenseFromResponse(currentResponse)
	if !ok {
		return licensemanagersdk.UpdateProductLicenseDetails{}, false, fmt.Errorf("current ProductLicense response does not expose a ProductLicense body")
	}
	if err := validateProductLicenseCreateOnlyDrift(resource, current); err != nil {
		return licensemanagersdk.UpdateProductLicenseDetails{}, false, err
	}
	return desiredProductLicenseUpdateDetails(resource.Spec, current)
}

func desiredProductLicenseUpdateDetails(
	spec licensemanagerv1beta1.ProductLicenseSpec,
	current licensemanagersdk.ProductLicense,
) (licensemanagersdk.UpdateProductLicenseDetails, bool, error) {
	details := licensemanagersdk.UpdateProductLicenseDetails{
		Images: productLicenseImageDetailsFromResponse(current.Images),
	}
	imagesChanged, err := applyProductLicenseUpdateImages(&details, spec.Images, current.Images)
	if err != nil {
		return licensemanagersdk.UpdateProductLicenseDetails{}, false, err
	}
	tagsChanged := applyProductLicenseUpdateTags(&details, spec, current)
	updateNeeded := imagesChanged || tagsChanged
	return details, updateNeeded, nil
}

func applyProductLicenseUpdateImages(
	details *licensemanagersdk.UpdateProductLicenseDetails,
	desired []licensemanagerv1beta1.ProductLicenseImage,
	current []licensemanagersdk.ImageResponse,
) (bool, error) {
	if desired == nil {
		return false, nil
	}
	desiredImages, err := productLicenseImageDetailsFromSpec(desired)
	if err != nil {
		return false, err
	}
	details.Images = desiredImages
	return !productLicenseImagesEqual(desired, current), nil
}

func applyProductLicenseUpdateTags(
	details *licensemanagersdk.UpdateProductLicenseDetails,
	spec licensemanagerv1beta1.ProductLicenseSpec,
	current licensemanagersdk.ProductLicense,
) bool {
	freeformChanged := applyProductLicenseFreeformTags(details, spec.FreeformTags, current.FreeformTags)
	definedChanged := applyProductLicenseDefinedTags(details, spec.DefinedTags, current.DefinedTags)
	return freeformChanged || definedChanged
}

func applyProductLicenseFreeformTags(
	details *licensemanagersdk.UpdateProductLicenseDetails,
	desired map[string]string,
	current map[string]string,
) bool {
	if desired == nil || maps.Equal(desired, current) {
		return false
	}
	details.FreeformTags = maps.Clone(desired)
	return true
}

func applyProductLicenseDefinedTags(
	details *licensemanagersdk.UpdateProductLicenseDetails,
	desired map[string]shared.MapValue,
	current map[string]map[string]interface{},
) bool {
	if desired == nil {
		return false
	}
	ociDefinedTags := *util.ConvertToOciDefinedTags(&desired)
	if productLicenseDefinedTagsEqual(ociDefinedTags, current) {
		return false
	}
	details.DefinedTags = ociDefinedTags
	return true
}

func validateProductLicenseRequiredSpec(spec licensemanagerv1beta1.ProductLicenseSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.LicenseUnit) == "" {
		missing = append(missing, "licenseUnit")
	}
	if len(missing) > 0 {
		return fmt.Errorf("ProductLicense spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if _, ok := licensemanagersdk.GetMappingLicenseUnitEnum(strings.TrimSpace(spec.LicenseUnit)); !ok {
		return fmt.Errorf("ProductLicense spec.licenseUnit %q is not supported", strings.TrimSpace(spec.LicenseUnit))
	}
	return nil
}

func validateProductLicenseCreateOnlyDriftForResponse(
	resource *licensemanagerv1beta1.ProductLicense,
	currentResponse any,
) error {
	current, ok := productLicenseFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current ProductLicense response does not expose a ProductLicense body")
	}
	return validateProductLicenseCreateOnlyDrift(resource, current)
}

func validateProductLicenseCreateOnlyDrift(
	resource *licensemanagerv1beta1.ProductLicense,
	current licensemanagersdk.ProductLicense,
) error {
	if resource == nil {
		return fmt.Errorf("ProductLicense resource is nil")
	}
	spec := resource.Spec
	var drift []string
	drift = appendStringPtrDrift(drift, "compartmentId", spec.CompartmentId, current.CompartmentId)
	drift = appendStringPtrDrift(drift, "displayName", spec.DisplayName, current.DisplayName)
	drift = appendEnumDrift(drift, "licenseUnit", spec.LicenseUnit, string(current.LicenseUnit))
	if current.IsVendorOracle != nil && spec.IsVendorOracle != *current.IsVendorOracle {
		drift = append(drift, "isVendorOracle")
	}
	drift = appendExplicitStringPtrDrift(drift, "vendorName", spec.VendorName, current.VendorName)
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("ProductLicense create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func appendStringPtrDrift(drift []string, fieldName string, desired string, current *string) []string {
	if current == nil {
		return drift
	}
	if strings.TrimSpace(desired) != strings.TrimSpace(*current) {
		return append(drift, fieldName)
	}
	return drift
}

func appendEnumDrift(drift []string, fieldName string, desired string, current string) []string {
	if strings.TrimSpace(current) == "" {
		return drift
	}
	if strings.TrimSpace(desired) != strings.TrimSpace(current) {
		return append(drift, fieldName)
	}
	return drift
}

func appendExplicitStringPtrDrift(drift []string, fieldName string, desired string, current *string) []string {
	if strings.TrimSpace(desired) == "" {
		return drift
	}
	if current == nil || strings.TrimSpace(desired) != strings.TrimSpace(*current) {
		return append(drift, fieldName)
	}
	return drift
}

func listProductLicensePages(
	ctx context.Context,
	client productLicenseOCIClient,
	initErr error,
	request licensemanagersdk.ListProductLicensesRequest,
) (licensemanagersdk.ListProductLicensesResponse, error) {
	if err := requireProductLicenseOCIClient(client, initErr); err != nil {
		return licensemanagersdk.ListProductLicensesResponse{}, err
	}

	seenPages := map[string]struct{}{}
	var combined licensemanagersdk.ListProductLicensesResponse
	for {
		response, err := client.ListProductLicenses(ctx, request)
		if err != nil {
			return licensemanagersdk.ListProductLicensesResponse{}, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := strings.TrimSpace(stringValue(response.OpcNextPage))
		if nextPage == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return licensemanagersdk.ListProductLicensesResponse{}, fmt.Errorf("ProductLicense list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func productLicenseFromResponse(response any) (licensemanagersdk.ProductLicense, bool) {
	if current, ok := productLicenseFromBody(response); ok {
		return current, true
	}
	if current, ok := productLicenseFromSummaryResponse(response); ok {
		return current, true
	}
	return productLicenseFromOperationResponse(response)
}

func productLicenseFromBody(response any) (licensemanagersdk.ProductLicense, bool) {
	switch current := response.(type) {
	case licensemanagersdk.ProductLicense:
		return current, true
	case *licensemanagersdk.ProductLicense:
		if current == nil {
			return licensemanagersdk.ProductLicense{}, false
		}
		return *current, true
	default:
		return licensemanagersdk.ProductLicense{}, false
	}
}

func productLicenseFromSummaryResponse(response any) (licensemanagersdk.ProductLicense, bool) {
	switch current := response.(type) {
	case licensemanagersdk.ProductLicenseSummary:
		return productLicenseFromSummary(current), true
	case *licensemanagersdk.ProductLicenseSummary:
		if current == nil {
			return licensemanagersdk.ProductLicense{}, false
		}
		return productLicenseFromSummary(*current), true
	default:
		return licensemanagersdk.ProductLicense{}, false
	}
}

func productLicenseFromOperationResponse(response any) (licensemanagersdk.ProductLicense, bool) {
	switch current := response.(type) {
	case licensemanagersdk.CreateProductLicenseResponse:
		return current.ProductLicense, true
	case *licensemanagersdk.CreateProductLicenseResponse:
		if current == nil {
			return licensemanagersdk.ProductLicense{}, false
		}
		return current.ProductLicense, true
	case licensemanagersdk.GetProductLicenseResponse:
		return current.ProductLicense, true
	case *licensemanagersdk.GetProductLicenseResponse:
		if current == nil {
			return licensemanagersdk.ProductLicense{}, false
		}
		return current.ProductLicense, true
	case licensemanagersdk.UpdateProductLicenseResponse:
		return current.ProductLicense, true
	case *licensemanagersdk.UpdateProductLicenseResponse:
		if current == nil {
			return licensemanagersdk.ProductLicense{}, false
		}
		return current.ProductLicense, true
	default:
		return licensemanagersdk.ProductLicense{}, false
	}
}

func productLicenseFromSummary(summary licensemanagersdk.ProductLicenseSummary) licensemanagersdk.ProductLicense {
	return licensemanagersdk.ProductLicense{
		Id:                          summary.Id,
		CompartmentId:               summary.CompartmentId,
		Status:                      summary.Status,
		LicenseUnit:                 summary.LicenseUnit,
		IsVendorOracle:              summary.IsVendorOracle,
		DisplayName:                 summary.DisplayName,
		StatusDescription:           summary.StatusDescription,
		TotalActiveLicenseUnitCount: summary.TotalActiveLicenseUnitCount,
		LifecycleState:              summary.LifecycleState,
		TotalLicenseUnitsConsumed:   summary.TotalLicenseUnitsConsumed,
		TotalLicenseRecordCount:     summary.TotalLicenseRecordCount,
		ActiveLicenseRecordCount:    summary.ActiveLicenseRecordCount,
		IsOverSubscribed:            summary.IsOverSubscribed,
		IsUnlimited:                 summary.IsUnlimited,
		VendorName:                  summary.VendorName,
		TimeCreated:                 summary.TimeCreated,
		TimeUpdated:                 summary.TimeUpdated,
		Images:                      summary.Images,
		FreeformTags:                summary.FreeformTags,
		DefinedTags:                 summary.DefinedTags,
		SystemTags:                  summary.SystemTags,
	}
}

func projectProductLicenseStatusFromResponse(resource *licensemanagerv1beta1.ProductLicense, response any) error {
	current, ok := productLicenseFromResponse(response)
	if !ok {
		return nil
	}
	projectProductLicenseStatus(resource, current)
	return nil
}

func projectProductLicenseStatus(
	resource *licensemanagerv1beta1.ProductLicense,
	current licensemanagersdk.ProductLicense,
) {
	if resource == nil {
		return
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = licensemanagerv1beta1.ProductLicenseStatus{
		OsokStatus:                  osokStatus,
		Id:                          productLicenseID(current),
		CompartmentId:               stringValue(current.CompartmentId),
		Status:                      string(current.Status),
		LicenseUnit:                 string(current.LicenseUnit),
		IsVendorOracle:              boolValue(current.IsVendorOracle),
		DisplayName:                 stringValue(current.DisplayName),
		StatusDescription:           stringValue(current.StatusDescription),
		TotalActiveLicenseUnitCount: intValue(current.TotalActiveLicenseUnitCount),
		LifecycleState:              string(current.LifecycleState),
		TotalLicenseUnitsConsumed:   float64Value(current.TotalLicenseUnitsConsumed),
		TotalLicenseRecordCount:     intValue(current.TotalLicenseRecordCount),
		ActiveLicenseRecordCount:    intValue(current.ActiveLicenseRecordCount),
		IsOverSubscribed:            boolValue(current.IsOverSubscribed),
		IsUnlimited:                 boolValue(current.IsUnlimited),
		VendorName:                  stringValue(current.VendorName),
		TimeCreated:                 sdkTimeString(current.TimeCreated),
		TimeUpdated:                 sdkTimeString(current.TimeUpdated),
		Images:                      productLicenseStatusImages(current.Images),
		FreeformTags:                cloneStringMap(current.FreeformTags),
		DefinedTags:                 productLicenseStatusTags(current.DefinedTags),
		SystemTags:                  productLicenseStatusTags(current.SystemTags),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func markProductLicenseSuccess(
	resource *licensemanagerv1beta1.ProductLicense,
	current licensemanagersdk.ProductLicense,
	fallback shared.OSOKConditionType,
	log loggerutil.OSOKLogger,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}

	condition := productLicenseCondition(current, fallback)
	message := productLicenseConditionMessage(current, condition)
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active {
		servicemanager.ClearAsyncOperation(status)
	}
	conditionStatus := corev1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = corev1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating,
		RequeueDuration: productLicenseRequeueDuration,
	}
}

func markProductLicenseFailure(resource *licensemanagerv1beta1.ProductLicense, err error, log loggerutil.OSOKLogger) {
	if resource == nil || err == nil {
		return
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), log)
}

func markProductLicenseDeleted(resource *licensemanagerv1beta1.ProductLicense, message string, log loggerutil.OSOKLogger) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = strings.TrimSpace(message)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, log)
}

func markProductLicenseTerminating(resource *licensemanagerv1beta1.ProductLicense, message string, log loggerutil.OSOKLogger) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
		UpdatedAt:       &now,
	}
	servicemanager.ApplyAsyncOperation(status, current, log)
}

func markProductLicenseDeleteWorkRequest(resource *licensemanagerv1beta1.ProductLicense, workRequestID string, log loggerutil.OSOKLogger) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         "OCI delete is in progress",
		UpdatedAt:       &now,
	}, log)
}

func markProductLicenseCreateWorkRequest(resource *licensemanagerv1beta1.ProductLicense, workRequestID string, log loggerutil.OSOKLogger) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         "OCI create is in progress",
		UpdatedAt:       &now,
	}, log)
}

func productLicenseCondition(
	current licensemanagersdk.ProductLicense,
	fallback shared.OSOKConditionType,
) shared.OSOKConditionType {
	switch current.LifecycleState {
	case licensemanagersdk.LifeCycleStateDeleted:
		return shared.Terminating
	case licensemanagersdk.LifeCycleStateActive, licensemanagersdk.LifeCycleStateInactive:
		return shared.Active
	case "":
		return fallback
	default:
		return shared.Failed
	}
}

func productLicenseConditionMessage(
	current licensemanagersdk.ProductLicense,
	condition shared.OSOKConditionType,
) string {
	if displayName := stringValue(current.DisplayName); displayName != "" {
		return displayName
	}
	switch condition {
	case shared.Terminating:
		return "OCI resource delete is in progress"
	case shared.Active:
		return "OCI resource is active"
	case shared.Failed:
		return "OCI resource state is not modeled"
	default:
		return "OCI resource state was observed"
	}
}

func handleProductLicensePreDeleteReadError(
	resource *licensemanagerv1beta1.ProductLicense,
	err error,
	log loggerutil.OSOKLogger,
) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markProductLicenseDeleted(resource, "OCI resource no longer exists", log)
		return true, nil
	case classification.IsAuthShapedNotFound(), isProductLicenseAmbiguousNotFound(err):
		ambiguous := productLicenseAmbiguousNotFound("pre-delete read", err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		return false, ambiguous
	default:
		return false, err
	}
}

func handleProductLicensePostDeleteReadError(
	resource *licensemanagerv1beta1.ProductLicense,
	err error,
	log loggerutil.OSOKLogger,
) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markProductLicenseDeleted(resource, "OCI resource deleted", log)
		return true, nil
	case classification.IsAuthShapedNotFound(), isProductLicenseAmbiguousNotFound(err):
		ambiguous := productLicenseAmbiguousNotFound("delete confirmation", err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		return false, ambiguous
	default:
		return false, err
	}
}

func handleProductLicenseDeleteCallError(
	resource *licensemanagerv1beta1.ProductLicense,
	err error,
	log loggerutil.OSOKLogger,
) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markProductLicenseDeleted(resource, "OCI resource no longer exists", log)
		return true, nil
	case classification.IsAuthShapedNotFound(), isProductLicenseAmbiguousNotFound(err):
		ambiguous := productLicenseAmbiguousNotFound("delete", err)
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		return false, ambiguous
	default:
		return false, err
	}
}

func normalizeProductLicenseNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if isProductLicenseAmbiguousNotFound(err) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return productLicenseAmbiguousNotFound(operation, err)
}

func productLicenseAmbiguousNotFound(operation string, err error) productLicenseAmbiguousNotFoundError {
	return productLicenseAmbiguousNotFoundError{
		message:      fmt.Sprintf("ProductLicense %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func productLicenseUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func isProductLicenseAmbiguousNotFound(err error) bool {
	var ambiguous productLicenseAmbiguousNotFoundError
	return errors.As(err, &ambiguous)
}

func productLicenseDeleteInProgress(resource *licensemanagerv1beta1.ProductLicense) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete && current.NormalizedClass == shared.OSOKAsyncClassPending
}

func productLicenseRecordedID(resource *licensemanagerv1beta1.ProductLicense) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func productLicenseID(current licensemanagersdk.ProductLicense) string {
	return strings.TrimSpace(stringValue(current.Id))
}

func productLicenseSummaryID(current licensemanagersdk.ProductLicenseSummary) string {
	return strings.TrimSpace(stringValue(current.Id))
}

func productLicenseTerminalDeleted(current licensemanagersdk.ProductLicense) bool {
	return current.LifecycleState == licensemanagersdk.LifeCycleStateDeleted
}

func productLicenseSummaryTerminalDeleted(current licensemanagersdk.ProductLicenseSummary) bool {
	return current.LifecycleState == licensemanagersdk.LifeCycleStateDeleted
}

func productLicenseRetryToken(resource *licensemanagerv1beta1.ProductLicense, namespace string) *string {
	if resource == nil {
		return nil
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return common.String(uid)
	}
	ns := strings.TrimSpace(resource.Namespace)
	if ns == "" {
		ns = strings.TrimSpace(namespace)
	}
	seed := strings.Trim(ns+"/"+strings.TrimSpace(resource.Name), "/")
	if seed == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(seed))
	return common.String(fmt.Sprintf("%x", sum[:16]))
}

func productLicenseImageDetailsFromSpec(
	images []licensemanagerv1beta1.ProductLicenseImage,
) ([]licensemanagersdk.ImageDetails, error) {
	if images == nil {
		return nil, nil
	}
	result := make([]licensemanagersdk.ImageDetails, 0, len(images))
	for index, image := range images {
		listingID := strings.TrimSpace(image.ListingId)
		packageVersion := strings.TrimSpace(image.PackageVersion)
		if listingID == "" || packageVersion == "" {
			return nil, fmt.Errorf("ProductLicense spec.images[%d].listingId and packageVersion are required", index)
		}
		result = append(result, licensemanagersdk.ImageDetails{
			ListingId:      common.String(listingID),
			PackageVersion: common.String(packageVersion),
		})
	}
	return result, nil
}

func productLicenseImageDetailsFromResponse(images []licensemanagersdk.ImageResponse) []licensemanagersdk.ImageDetails {
	if len(images) == 0 {
		return nil
	}
	result := make([]licensemanagersdk.ImageDetails, 0, len(images))
	for _, image := range images {
		result = append(result, licensemanagersdk.ImageDetails{
			ListingId:      cloneStringPtr(image.ListingId),
			PackageVersion: cloneStringPtr(image.PackageVersion),
		})
	}
	return result
}

func productLicenseImagesEqual(
	desired []licensemanagerv1beta1.ProductLicenseImage,
	current []licensemanagersdk.ImageResponse,
) bool {
	if len(desired) != len(current) {
		return false
	}
	for index := range desired {
		if strings.TrimSpace(desired[index].ListingId) != strings.TrimSpace(stringValue(current[index].ListingId)) ||
			strings.TrimSpace(desired[index].PackageVersion) != strings.TrimSpace(stringValue(current[index].PackageVersion)) {
			return false
		}
	}
	return true
}

func productLicenseDefinedTagsEqual(
	desired map[string]map[string]interface{},
	current map[string]map[string]interface{},
) bool {
	if len(desired) == 0 && len(current) == 0 {
		return true
	}
	return reflect.DeepEqual(desired, current)
}

func productLicenseStatusImages(images []licensemanagersdk.ImageResponse) []licensemanagerv1beta1.ProductLicenseImage {
	if len(images) == 0 {
		return nil
	}
	result := make([]licensemanagerv1beta1.ProductLicenseImage, 0, len(images))
	for _, image := range images {
		result = append(result, licensemanagerv1beta1.ProductLicenseImage{
			ListingId:      stringValue(image.ListingId),
			PackageVersion: stringValue(image.PackageVersion),
		})
	}
	return result
}

func productLicenseStatusTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(tags) == 0 {
		return nil
	}
	result := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		converted := make(shared.MapValue, len(values))
		for key, value := range values {
			if stringValue, ok := value.(string); ok {
				converted[key] = stringValue
				continue
			}
			converted[key] = fmt.Sprint(value)
		}
		result[namespace] = converted
	}
	return result
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	return maps.Clone(values)
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	return common.String(*value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

var _ ProductLicenseServiceClient = (*productLicenseRuntimeClient)(nil)
var _ error = productLicenseAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = productLicenseAmbiguousNotFoundError{}
