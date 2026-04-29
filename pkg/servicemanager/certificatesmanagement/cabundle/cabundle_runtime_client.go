/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cabundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"

	certificatesmanagementsdk "github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/common"
	certificatesmanagementv1beta1 "github.com/oracle/oci-service-operator/api/certificatesmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

const caBundlePEMFingerprintTag = "osokCaBundlePemSha256"

type caBundleOCIClient interface {
	CreateCaBundle(context.Context, certificatesmanagementsdk.CreateCaBundleRequest) (certificatesmanagementsdk.CreateCaBundleResponse, error)
	GetCaBundle(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error)
	ListCaBundles(context.Context, certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error)
	UpdateCaBundle(context.Context, certificatesmanagementsdk.UpdateCaBundleRequest) (certificatesmanagementsdk.UpdateCaBundleResponse, error)
	DeleteCaBundle(context.Context, certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error)
}

type ambiguousCaBundleNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousCaBundleNotFoundError) Error() string {
	return e.message
}

func (e ambiguousCaBundleNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerCaBundleRuntimeHooksMutator(func(_ *CaBundleServiceManager, hooks *CaBundleRuntimeHooks) {
		applyCaBundleRuntimeHooks(hooks)
	})
}

func applyCaBundleRuntimeHooks(hooks *CaBundleRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedCaBundleRuntimeSemantics()
	hooks.BuildCreateBody = buildCaBundleCreateBody
	hooks.BuildUpdateBody = buildCaBundleUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardCaBundleExistingBeforeCreate
	hooks.List.Fields = caBundleListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedCaBundleIdentity
	hooks.StatusHooks.ProjectStatus = projectCaBundleStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateCaBundleCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleCaBundleDeleteError
	wrapCaBundleReadAndDeleteCalls(hooks)
	wrapCaBundleDeleteConfirmation(hooks)
}

func newCaBundleServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client caBundleOCIClient,
) CaBundleServiceClient {
	hooks := newCaBundleRuntimeHooksWithOCIClient(client)
	applyCaBundleRuntimeHooks(&hooks)
	manager := &CaBundleServiceManager{Log: log}
	delegate := defaultCaBundleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*certificatesmanagementv1beta1.CaBundle](
			buildCaBundleGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapCaBundleGeneratedClient(hooks, delegate)
}

func newCaBundleRuntimeHooksWithOCIClient(client caBundleOCIClient) CaBundleRuntimeHooks {
	return CaBundleRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*certificatesmanagementv1beta1.CaBundle]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*certificatesmanagementv1beta1.CaBundle]{},
		StatusHooks:     generatedruntime.StatusHooks[*certificatesmanagementv1beta1.CaBundle]{},
		ParityHooks:     generatedruntime.ParityHooks[*certificatesmanagementv1beta1.CaBundle]{},
		Async:           generatedruntime.AsyncHooks[*certificatesmanagementv1beta1.CaBundle]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*certificatesmanagementv1beta1.CaBundle]{},
		Create: runtimeOperationHooks[certificatesmanagementsdk.CreateCaBundleRequest, certificatesmanagementsdk.CreateCaBundleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateCaBundleDetails", RequestName: "CreateCaBundleDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request certificatesmanagementsdk.CreateCaBundleRequest) (certificatesmanagementsdk.CreateCaBundleResponse, error) {
				if client == nil {
					return certificatesmanagementsdk.CreateCaBundleResponse{}, fmt.Errorf("CaBundle OCI client is nil")
				}
				return client.CreateCaBundle(ctx, request)
			},
		},
		Get: runtimeOperationHooks[certificatesmanagementsdk.GetCaBundleRequest, certificatesmanagementsdk.GetCaBundleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CaBundleId", RequestName: "caBundleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
				if client == nil {
					return certificatesmanagementsdk.GetCaBundleResponse{}, fmt.Errorf("CaBundle OCI client is nil")
				}
				return client.GetCaBundle(ctx, request)
			},
		},
		List: runtimeOperationHooks[certificatesmanagementsdk.ListCaBundlesRequest, certificatesmanagementsdk.ListCaBundlesResponse]{
			Fields: caBundleListFields(),
			Call: func(ctx context.Context, request certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error) {
				if client == nil {
					return certificatesmanagementsdk.ListCaBundlesResponse{}, fmt.Errorf("CaBundle OCI client is nil")
				}
				return client.ListCaBundles(ctx, request)
			},
		},
		Update: runtimeOperationHooks[certificatesmanagementsdk.UpdateCaBundleRequest, certificatesmanagementsdk.UpdateCaBundleResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CaBundleId", RequestName: "caBundleId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateCaBundleDetails", RequestName: "UpdateCaBundleDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request certificatesmanagementsdk.UpdateCaBundleRequest) (certificatesmanagementsdk.UpdateCaBundleResponse, error) {
				if client == nil {
					return certificatesmanagementsdk.UpdateCaBundleResponse{}, fmt.Errorf("CaBundle OCI client is nil")
				}
				return client.UpdateCaBundle(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[certificatesmanagementsdk.DeleteCaBundleRequest, certificatesmanagementsdk.DeleteCaBundleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CaBundleId", RequestName: "caBundleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error) {
				if client == nil {
					return certificatesmanagementsdk.DeleteCaBundleResponse{}, fmt.Errorf("CaBundle OCI client is nil")
				}
				return client.DeleteCaBundle(ctx, request)
			},
		},
		WrapGeneratedClient: []func(CaBundleServiceClient) CaBundleServiceClient{},
	}
}

func reviewedCaBundleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "certificatesmanagement",
		FormalSlug:    "cabundle",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(certificatesmanagementsdk.CaBundleLifecycleStateCreating)},
			UpdatingStates:     []string{string(certificatesmanagementsdk.CaBundleLifecycleStateUpdating)},
			ActiveStates:       []string{string(certificatesmanagementsdk.CaBundleLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(certificatesmanagementsdk.CaBundleLifecycleStateDeleting)},
			TerminalStates: []string{string(certificatesmanagementsdk.CaBundleLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"caBundlePem", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "name"},
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

func caBundleListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "name"}},
		{FieldName: "CaBundleId", RequestName: "caBundleId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func guardCaBundleExistingBeforeCreate(
	_ context.Context,
	resource *certificatesmanagementv1beta1.CaBundle,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("CaBundle resource is nil")
	}
	normalizeCaBundleSpec(resource)
	if resource.Spec.CompartmentId == "" || resource.Spec.Name == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildCaBundleCreateBody(
	_ context.Context,
	resource *certificatesmanagementv1beta1.CaBundle,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("CaBundle resource is nil")
	}
	normalizeCaBundleSpec(resource)
	if err := validateCaBundleSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := certificatesmanagementsdk.CreateCaBundleDetails{
		Name:          common.String(resource.Spec.Name),
		CompartmentId: common.String(resource.Spec.CompartmentId),
		CaBundlePem:   common.String(resource.Spec.CaBundlePem),
	}
	if resource.Spec.Description != "" {
		body.Description = common.String(resource.Spec.Description)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = caBundleFreeformTagsFromSpec(resource.Spec.FreeformTags, resource.Spec.CaBundlePem)
	}
	if body.FreeformTags == nil {
		body.FreeformTags = caBundleFreeformTagsFromSpec(nil, resource.Spec.CaBundlePem)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = caBundleDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildCaBundleUpdateBody(
	_ context.Context,
	resource *certificatesmanagementv1beta1.CaBundle,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return certificatesmanagementsdk.UpdateCaBundleDetails{}, false, fmt.Errorf("CaBundle resource is nil")
	}
	normalizeCaBundleSpec(resource)
	if err := validateCaBundleSpec(resource.Spec); err != nil {
		return certificatesmanagementsdk.UpdateCaBundleDetails{}, false, err
	}
	current, ok := caBundleFromResponse(currentResponse)
	if !ok {
		return certificatesmanagementsdk.UpdateCaBundleDetails{}, false, fmt.Errorf("current CaBundle response does not expose a CaBundle body")
	}
	if err := validateCaBundleCreateOnlyDrift(resource.Spec, current); err != nil {
		return certificatesmanagementsdk.UpdateCaBundleDetails{}, false, err
	}

	details := certificatesmanagementsdk.UpdateCaBundleDetails{}
	updateNeeded := false
	if !caBundlePEMFingerprintMatches(current.FreeformTags, resource.Spec.CaBundlePem) {
		details.CaBundlePem = common.String(resource.Spec.CaBundlePem)
		updateNeeded = true
	}
	if !caBundleStringPtrEqual(current.Description, resource.Spec.Description) {
		details.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}
	updateNeeded = applyCaBundleFreeformTagsUpdate(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyCaBundleDefinedTagsUpdate(&details, resource.Spec, current) || updateNeeded
	if !updateNeeded {
		return certificatesmanagementsdk.UpdateCaBundleDetails{}, false, nil
	}
	return details, true, nil
}

func applyCaBundleFreeformTagsUpdate(
	details *certificatesmanagementsdk.UpdateCaBundleDetails,
	spec certificatesmanagementv1beta1.CaBundleSpec,
	current certificatesmanagementsdk.CaBundle,
) bool {
	desired := desiredCaBundleFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags, spec.CaBundlePem)
	if reflect.DeepEqual(current.FreeformTags, desired) {
		return false
	}
	details.FreeformTags = desired
	return true
}

func applyCaBundleDefinedTagsUpdate(
	details *certificatesmanagementsdk.UpdateCaBundleDetails,
	spec certificatesmanagementv1beta1.CaBundleSpec,
	current certificatesmanagementsdk.CaBundle,
) bool {
	desired := desiredCaBundleDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags)
	if reflect.DeepEqual(current.DefinedTags, desired) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func validateCaBundleSpec(spec certificatesmanagementv1beta1.CaBundleSpec) error {
	var missing []string
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.CaBundlePem) == "" {
		missing = append(missing, "caBundlePem")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("CaBundle spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func normalizeCaBundleSpec(resource *certificatesmanagementv1beta1.CaBundle) {
	if resource == nil {
		return
	}
	resource.Spec.Name = strings.TrimSpace(resource.Spec.Name)
	resource.Spec.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	resource.Spec.Description = strings.TrimSpace(resource.Spec.Description)
}

func validateCaBundleCreateOnlyDriftForResponse(
	resource *certificatesmanagementv1beta1.CaBundle,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("CaBundle resource is nil")
	}
	current, ok := caBundleFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current CaBundle response does not expose a CaBundle body")
	}
	return validateCaBundleCreateOnlyDrift(resource.Spec, current)
}

func validateCaBundleCreateOnlyDrift(
	spec certificatesmanagementv1beta1.CaBundleSpec,
	current certificatesmanagementsdk.CaBundle,
) error {
	var drift []string
	if !caBundleStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !caBundleStringPtrEqual(current.Name, spec.Name) {
		drift = append(drift, "name")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("CaBundle create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func desiredCaBundleFreeformTagsForUpdate(spec map[string]string, current map[string]string, caBundlePEM string) map[string]string {
	if spec != nil {
		return caBundleFreeformTagsFromSpec(spec, caBundlePEM)
	}
	if current != nil {
		return caBundleFreeformTagsFromSpec(nil, caBundlePEM)
	}
	return caBundleFreeformTagsFromSpec(nil, caBundlePEM)
}

func desiredCaBundleDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return caBundleDefinedTagsFromSpec(spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func wrapCaBundleReadAndDeleteCalls(hooks *CaBundleRuntimeHooks) {
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeCaBundleNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error) {
			return listCaBundlesAllPages(ctx, call, request)
		}
	}
	if hooks.Delete.Call != nil {
		call := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request certificatesmanagementsdk.DeleteCaBundleRequest) (certificatesmanagementsdk.DeleteCaBundleResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeCaBundleNotFoundError(err, "delete")
		}
	}
}

func listCaBundlesAllPages(
	ctx context.Context,
	call func(context.Context, certificatesmanagementsdk.ListCaBundlesRequest) (certificatesmanagementsdk.ListCaBundlesResponse, error),
	request certificatesmanagementsdk.ListCaBundlesRequest,
) (certificatesmanagementsdk.ListCaBundlesResponse, error) {
	var combined certificatesmanagementsdk.ListCaBundlesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return certificatesmanagementsdk.ListCaBundlesResponse{}, conservativeCaBundleNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == certificatesmanagementsdk.CaBundleLifecycleStateDeleted {
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

func projectCaBundleStatus(resource *certificatesmanagementv1beta1.CaBundle, response any) error {
	if resource == nil {
		return fmt.Errorf("CaBundle resource is nil")
	}
	current, ok := caBundleFromResponse(response)
	if !ok {
		return nil
	}
	resource.Status = certificatesmanagementv1beta1.CaBundleStatus{
		OsokStatus:       resource.Status.OsokStatus,
		Id:               caBundleStringValue(current.Id),
		Name:             caBundleStringValue(current.Name),
		TimeCreated:      caBundleSDKTimeString(current.TimeCreated),
		LifecycleState:   string(current.LifecycleState),
		CompartmentId:    caBundleStringValue(current.CompartmentId),
		Description:      caBundleStringValue(current.Description),
		LifecycleDetails: caBundleStringValue(current.LifecycleDetails),
		FreeformTags:     cloneCaBundleStringMap(current.FreeformTags),
		DefinedTags:      caBundleStatusDefinedTags(current.DefinedTags),
	}
	return nil
}

func clearTrackedCaBundleIdentity(resource *certificatesmanagementv1beta1.CaBundle) {
	if resource == nil {
		return
	}
	resource.Status = certificatesmanagementv1beta1.CaBundleStatus{}
}

func handleCaBundleDeleteError(resource *certificatesmanagementv1beta1.CaBundle, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func wrapCaBundleDeleteConfirmation(hooks *CaBundleRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getCaBundle := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate CaBundleServiceClient) CaBundleServiceClient {
		return caBundleDeleteConfirmationClient{
			delegate:    delegate,
			getCaBundle: getCaBundle,
		}
	})
}

type caBundleDeleteConfirmationClient struct {
	delegate    CaBundleServiceClient
	getCaBundle func(context.Context, certificatesmanagementsdk.GetCaBundleRequest) (certificatesmanagementsdk.GetCaBundleResponse, error)
}

func (c caBundleDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *certificatesmanagementv1beta1.CaBundle,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c caBundleDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *certificatesmanagementv1beta1.CaBundle,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c caBundleDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *certificatesmanagementv1beta1.CaBundle,
) error {
	if c.getCaBundle == nil || resource == nil {
		return nil
	}
	caBundleID := trackedCaBundleID(resource)
	if caBundleID == "" {
		return nil
	}
	_, err := c.getCaBundle(ctx, certificatesmanagementsdk.GetCaBundleRequest{CaBundleId: common.String(caBundleID)})
	if err == nil {
		return nil
	}
	if !isAmbiguousCaBundleNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("CaBundle delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedCaBundleID(resource *certificatesmanagementv1beta1.CaBundle) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func conservativeCaBundleNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !isAmbiguousCaBundleNotFound(err) {
		return err
	}
	return ambiguousCaBundleNotFoundError{
		message:      fmt.Sprintf("CaBundle %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func isAmbiguousCaBundleNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousCaBundleNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func caBundleFromResponse(response any) (certificatesmanagementsdk.CaBundle, bool) {
	if bundle, ok := caBundleFromResourceResponse(response); ok {
		return bundle, true
	}
	return caBundleFromOperationResponse(response)
}

func caBundleFromResourceResponse(response any) (certificatesmanagementsdk.CaBundle, bool) {
	switch current := response.(type) {
	case certificatesmanagementsdk.CaBundle:
		return current, true
	case *certificatesmanagementsdk.CaBundle:
		return caBundleFromOptional(current, func(bundle certificatesmanagementsdk.CaBundle) certificatesmanagementsdk.CaBundle {
			return bundle
		})
	case certificatesmanagementsdk.CaBundleSummary:
		return caBundleFromSummary(current), true
	case *certificatesmanagementsdk.CaBundleSummary:
		return caBundleFromOptional(current, caBundleFromSummary)
	default:
		return certificatesmanagementsdk.CaBundle{}, false
	}
}

func caBundleFromOperationResponse(response any) (certificatesmanagementsdk.CaBundle, bool) {
	switch current := response.(type) {
	case certificatesmanagementsdk.CreateCaBundleResponse:
		return current.CaBundle, true
	case *certificatesmanagementsdk.CreateCaBundleResponse:
		return caBundleFromOptional(current, func(response certificatesmanagementsdk.CreateCaBundleResponse) certificatesmanagementsdk.CaBundle {
			return response.CaBundle
		})
	case certificatesmanagementsdk.GetCaBundleResponse:
		return current.CaBundle, true
	case *certificatesmanagementsdk.GetCaBundleResponse:
		return caBundleFromOptional(current, func(response certificatesmanagementsdk.GetCaBundleResponse) certificatesmanagementsdk.CaBundle {
			return response.CaBundle
		})
	case certificatesmanagementsdk.UpdateCaBundleResponse:
		return current.CaBundle, true
	case *certificatesmanagementsdk.UpdateCaBundleResponse:
		return caBundleFromOptional(current, func(response certificatesmanagementsdk.UpdateCaBundleResponse) certificatesmanagementsdk.CaBundle {
			return response.CaBundle
		})
	default:
		return certificatesmanagementsdk.CaBundle{}, false
	}
}

func caBundleFromOptional[T any](
	current *T,
	convert func(T) certificatesmanagementsdk.CaBundle,
) (certificatesmanagementsdk.CaBundle, bool) {
	if current == nil {
		return certificatesmanagementsdk.CaBundle{}, false
	}
	return convert(*current), true
}

func caBundleFromSummary(summary certificatesmanagementsdk.CaBundleSummary) certificatesmanagementsdk.CaBundle {
	return certificatesmanagementsdk.CaBundle{
		Id:               summary.Id,
		Name:             summary.Name,
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		CompartmentId:    summary.CompartmentId,
		Description:      summary.Description,
		LifecycleDetails: summary.LifecycleDetails,
		FreeformTags:     cloneCaBundleStringMap(summary.FreeformTags),
		DefinedTags:      cloneCaBundleDefinedTags(summary.DefinedTags),
	}
}

func caBundleStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(caBundleStringValue(current)) == strings.TrimSpace(desired)
}

func caBundleStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func caBundleSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func cloneCaBundleStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func caBundleFreeformTagsFromSpec(spec map[string]string, caBundlePEM string) map[string]string {
	tags := cloneCaBundleStringMap(spec)
	if tags == nil {
		tags = map[string]string{}
	}
	tags[caBundlePEMFingerprintTag] = caBundlePEMFingerprint(caBundlePEM)
	return tags
}

func caBundleStatusFreeformTags(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	filtered := make(map[string]string, len(input))
	for key, value := range input {
		if key == caBundlePEMFingerprintTag {
			continue
		}
		filtered[key] = value
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func caBundlePEMFingerprintMatches(tags map[string]string, caBundlePEM string) bool {
	if tags == nil {
		return false
	}
	return strings.TrimSpace(tags[caBundlePEMFingerprintTag]) == caBundlePEMFingerprint(caBundlePEM)
}

func caBundlePEMFingerprint(caBundlePEM string) string {
	sum := sha256.Sum256([]byte(caBundlePEM))
	return hex.EncodeToString(sum[:])
}

func caBundleDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func caBundleStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if values == nil {
			converted[namespace] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	return converted
}

func cloneCaBundleDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, values := range input {
		if values == nil {
			cloned[key] = nil
			continue
		}
		inner := make(map[string]interface{}, len(values))
		for innerKey, innerValue := range values {
			inner[innerKey] = innerValue
		}
		cloned[key] = inner
	}
	return cloned
}

var _ error = ambiguousCaBundleNotFoundError{}
var _ interface{ GetOpcRequestID() string } = ambiguousCaBundleNotFoundError{}
