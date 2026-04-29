/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementstation

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
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

type managementStationOCIClient interface {
	CreateManagementStation(context.Context, osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error)
	GetManagementStation(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error)
	ListManagementStations(context.Context, osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error)
	UpdateManagementStation(context.Context, osmanagementhubsdk.UpdateManagementStationRequest) (osmanagementhubsdk.UpdateManagementStationResponse, error)
	DeleteManagementStation(context.Context, osmanagementhubsdk.DeleteManagementStationRequest) (osmanagementhubsdk.DeleteManagementStationResponse, error)
}

type managementStationListCall func(context.Context, osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error)

type managementStationAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e managementStationAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e managementStationAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type managementStationAuthShapedConfirmRead struct {
	err error
}

func (e managementStationAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("ManagementStation delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e managementStationAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func init() {
	registerManagementStationRuntimeHooksMutator(func(manager *ManagementStationServiceManager, hooks *ManagementStationRuntimeHooks) {
		applyManagementStationRuntimeHooks(manager, hooks)
	})
}

func applyManagementStationRuntimeHooks(manager *ManagementStationServiceManager, hooks *ManagementStationRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedManagementStationRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *osmanagementhubv1beta1.ManagementStation, _ string) (any, error) {
		return buildManagementStationCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *osmanagementhubv1beta1.ManagementStation,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildManagementStationUpdateBody(resource, currentResponse)
	}
	hooks.Get.Call = normalizeManagementStationGetCall(hooks.Get.Call)
	hooks.List.Fields = managementStationListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listManagementStationsAllPages(hooks.List.Call)
	}
	hooks.Update.Call = normalizeManagementStationUpdateCall(hooks.Update.Call)
	hooks.Create.Call = normalizeManagementStationCreateCall(hooks.Create.Call)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedManagementStationIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateManagementStationCreateOnlyDriftForResponse
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *osmanagementhubv1beta1.ManagementStation, currentID string) (any, error) {
		return confirmManagementStationDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleManagementStationDeleteError
	hooks.DeleteHooks.ApplyOutcome = handleManagementStationDeleteConfirmReadOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ManagementStationServiceClient) ManagementStationServiceClient {
		runtimeClient := managementStationDeleteFallbackClient{
			delegate: delegate,
			list:     hooks.List.Call,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

type managementStationDeleteFallbackClient struct {
	delegate ManagementStationServiceClient
	list     managementStationListCall
	log      loggerutil.OSOKLogger
}

func (c managementStationDeleteFallbackClient) CreateOrUpdate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagementStation,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c managementStationDeleteFallbackClient) Delete(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ManagementStation,
) (bool, error) {
	if !managementStationDeleteFallbackEnabled(resource, c.list) {
		return c.delegate.Delete(ctx, resource)
	}

	summary, found, err := resolveManagementStationDeleteFallbackSummary(ctx, c.list, resource)
	if err != nil {
		return false, err
	}
	if !found {
		markManagementStationDeleted(resource, "OCI resource no longer exists", c.log)
		return true, nil
	}

	currentID := managementStationStringValue(summary.Id)
	if currentID == "" {
		return false, fmt.Errorf("ManagementStation delete fallback could not resolve a resource OCID")
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
	resource.Status.Id = currentID
	return c.delegate.Delete(ctx, resource)
}

func managementStationDeleteFallbackEnabled(
	resource *osmanagementhubv1beta1.ManagementStation,
	list managementStationListCall,
) bool {
	return resource != nil &&
		managementStationRecordedID(resource) == "" &&
		list != nil
}

func resolveManagementStationDeleteFallbackSummary(
	ctx context.Context,
	list managementStationListCall,
	resource *osmanagementhubv1beta1.ManagementStation,
) (osmanagementhubsdk.ManagementStationSummary, bool, error) {
	response, err := list(ctx, osmanagementhubsdk.ListManagementStationsRequest{
		CompartmentId: managementStationStringPointer(resource.Spec.CompartmentId),
		DisplayName:   managementStationStringPointer(resource.Spec.DisplayName),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return osmanagementhubsdk.ManagementStationSummary{}, false, managementStationAuthShapedConfirmRead{err: err}
		}
		return osmanagementhubsdk.ManagementStationSummary{}, false, err
	}

	matches := matchingManagementStationSummaries(response.Items, resource.Spec)
	switch len(matches) {
	case 0:
		return osmanagementhubsdk.ManagementStationSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return osmanagementhubsdk.ManagementStationSummary{}, false, fmt.Errorf("ManagementStation delete fallback found multiple matching resources for compartmentId %q, displayName %q, and hostname %q", resource.Spec.CompartmentId, resource.Spec.DisplayName, resource.Spec.Hostname)
	}
}

func reviewedManagementStationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "osmanagementhub",
		FormalSlug:        "managementstation",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(osmanagementhubsdk.ManagementStationLifecycleStateCreating)},
			UpdatingStates:     []string{string(osmanagementhubsdk.ManagementStationLifecycleStateUpdating)},
			ActiveStates:       []string{string(osmanagementhubsdk.ManagementStationLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(osmanagementhubsdk.ManagementStationLifecycleStateDeleting)},
			TerminalStates: []string{string(osmanagementhubsdk.ManagementStationLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "hostname"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"hostname",
				"isAutoConfigEnabled",
				"proxy",
				"mirror",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ManagementStation", Action: "CreateManagementStation"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ManagementStation", Action: "UpdateManagementStation"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ManagementStation", Action: "DeleteManagementStation"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ManagementStation", Action: "GetManagementStation"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ManagementStation", Action: "GetManagementStation"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ManagementStation", Action: "GetManagementStation"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func managementStationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func buildManagementStationCreateBody(resource *osmanagementhubv1beta1.ManagementStation) (osmanagementhubsdk.CreateManagementStationDetails, error) {
	if resource == nil {
		return osmanagementhubsdk.CreateManagementStationDetails{}, fmt.Errorf("ManagementStation resource is nil")
	}
	if err := validateManagementStationSpec(resource.Spec); err != nil {
		return osmanagementhubsdk.CreateManagementStationDetails{}, err
	}

	spec := resource.Spec
	body := osmanagementhubsdk.CreateManagementStationDetails{
		CompartmentId:       common.String(strings.TrimSpace(spec.CompartmentId)),
		DisplayName:         common.String(strings.TrimSpace(spec.DisplayName)),
		Hostname:            common.String(strings.TrimSpace(spec.Hostname)),
		Proxy:               createManagementStationProxy(spec.Proxy),
		Mirror:              createManagementStationMirror(spec.Mirror),
		IsAutoConfigEnabled: common.Bool(spec.IsAutoConfigEnabled),
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneManagementStationStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = managementStationDefinedTags(spec.DefinedTags)
	}
	return body, nil
}

func buildManagementStationUpdateBody(
	resource *osmanagementhubv1beta1.ManagementStation,
	currentResponse any,
) (osmanagementhubsdk.UpdateManagementStationDetails, bool, error) {
	current, err := managementStationUpdateCurrent(resource, currentResponse)
	if err != nil {
		return osmanagementhubsdk.UpdateManagementStationDetails{}, false, err
	}

	details := osmanagementhubsdk.UpdateManagementStationDetails{}
	spec := resource.Spec
	updateNeeded := applyManagementStationScalarUpdates(&details, spec, current)
	updateNeeded = applyManagementStationConfigUpdates(&details, spec, current) || updateNeeded
	updateNeeded = applyManagementStationTagUpdates(&details, spec, current) || updateNeeded

	if !updateNeeded {
		return osmanagementhubsdk.UpdateManagementStationDetails{}, false, nil
	}
	return details, true, nil
}

func managementStationUpdateCurrent(
	resource *osmanagementhubv1beta1.ManagementStation,
	currentResponse any,
) (osmanagementhubsdk.ManagementStation, error) {
	if resource == nil {
		return osmanagementhubsdk.ManagementStation{}, fmt.Errorf("ManagementStation resource is nil")
	}
	if err := validateManagementStationSpec(resource.Spec); err != nil {
		return osmanagementhubsdk.ManagementStation{}, err
	}

	current, ok := managementStationFromResponse(currentResponse)
	if !ok {
		return osmanagementhubsdk.ManagementStation{}, fmt.Errorf("current ManagementStation response does not expose a ManagementStation body")
	}
	if err := validateManagementStationCreateOnlyDrift(resource.Spec, current); err != nil {
		return osmanagementhubsdk.ManagementStation{}, err
	}
	return current, nil
}

func applyManagementStationScalarUpdates(
	details *osmanagementhubsdk.UpdateManagementStationDetails,
	spec osmanagementhubv1beta1.ManagementStationSpec,
	current osmanagementhubsdk.ManagementStation,
) bool {
	updateNeeded := false

	if !managementStationStringPtrEqual(current.DisplayName, spec.DisplayName) {
		details.DisplayName = common.String(spec.DisplayName)
		updateNeeded = true
	}
	if !managementStationStringPtrEqual(current.Description, spec.Description) {
		details.Description = common.String(spec.Description)
		updateNeeded = true
	}
	if !managementStationStringPtrEqual(current.Hostname, spec.Hostname) {
		details.Hostname = common.String(spec.Hostname)
		updateNeeded = true
	}
	if !managementStationBoolPtrEqual(current.IsAutoConfigEnabled, spec.IsAutoConfigEnabled) {
		details.IsAutoConfigEnabled = common.Bool(spec.IsAutoConfigEnabled)
		updateNeeded = true
	}
	return updateNeeded
}

func applyManagementStationConfigUpdates(
	details *osmanagementhubsdk.UpdateManagementStationDetails,
	spec osmanagementhubv1beta1.ManagementStationSpec,
	current osmanagementhubsdk.ManagementStation,
) bool {
	updateNeeded := false

	if !managementStationProxyEqual(current.Proxy, spec.Proxy) {
		details.Proxy = updateManagementStationProxy(spec.Proxy)
		updateNeeded = true
	}
	if !managementStationMirrorEqual(current.Mirror, spec.Mirror) {
		details.Mirror = updateManagementStationMirror(spec.Mirror)
		updateNeeded = true
	}
	return updateNeeded
}

func applyManagementStationTagUpdates(
	details *osmanagementhubsdk.UpdateManagementStationDetails,
	spec osmanagementhubv1beta1.ManagementStationSpec,
	current osmanagementhubsdk.ManagementStation,
) bool {
	updateNeeded := false

	desiredFreeformTags := desiredManagementStationFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		details.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredManagementStationDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		details.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	return updateNeeded
}

func validateManagementStationSpec(spec osmanagementhubv1beta1.ManagementStationSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.Hostname) == "" {
		missing = append(missing, "hostname")
	}
	if strings.TrimSpace(spec.Mirror.Directory) == "" {
		missing = append(missing, "mirror.directory")
	}
	if strings.TrimSpace(spec.Mirror.Port) == "" {
		missing = append(missing, "mirror.port")
	}
	if strings.TrimSpace(spec.Mirror.Sslport) == "" {
		missing = append(missing, "mirror.sslport")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("ManagementStation spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func createManagementStationProxy(proxy osmanagementhubv1beta1.ManagementStationProxy) *osmanagementhubsdk.CreateProxyConfigurationDetails {
	details := &osmanagementhubsdk.CreateProxyConfigurationDetails{
		IsEnabled: common.Bool(proxy.IsEnabled),
	}
	if len(proxy.Hosts) > 0 {
		details.Hosts = append([]string(nil), proxy.Hosts...)
	}
	if strings.TrimSpace(proxy.Port) != "" {
		details.Port = common.String(proxy.Port)
	}
	if strings.TrimSpace(proxy.Forward) != "" {
		details.Forward = common.String(proxy.Forward)
	}
	return details
}

func updateManagementStationProxy(proxy osmanagementhubv1beta1.ManagementStationProxy) *osmanagementhubsdk.UpdateProxyConfigurationDetails {
	details := &osmanagementhubsdk.UpdateProxyConfigurationDetails{
		IsEnabled: common.Bool(proxy.IsEnabled),
	}
	if len(proxy.Hosts) > 0 {
		details.Hosts = append([]string(nil), proxy.Hosts...)
	}
	if strings.TrimSpace(proxy.Port) != "" {
		details.Port = common.String(proxy.Port)
	}
	if strings.TrimSpace(proxy.Forward) != "" {
		details.Forward = common.String(proxy.Forward)
	}
	return details
}

func createManagementStationMirror(mirror osmanagementhubv1beta1.ManagementStationMirror) *osmanagementhubsdk.CreateMirrorConfigurationDetails {
	details := &osmanagementhubsdk.CreateMirrorConfigurationDetails{
		Directory:          common.String(strings.TrimSpace(mirror.Directory)),
		Port:               common.String(strings.TrimSpace(mirror.Port)),
		Sslport:            common.String(strings.TrimSpace(mirror.Sslport)),
		IsSslverifyEnabled: common.Bool(mirror.IsSslverifyEnabled),
	}
	if strings.TrimSpace(mirror.Sslcert) != "" {
		details.Sslcert = common.String(mirror.Sslcert)
	}
	return details
}

func updateManagementStationMirror(mirror osmanagementhubv1beta1.ManagementStationMirror) *osmanagementhubsdk.UpdateMirrorConfigurationDetails {
	details := &osmanagementhubsdk.UpdateMirrorConfigurationDetails{
		Directory:          common.String(strings.TrimSpace(mirror.Directory)),
		Port:               common.String(strings.TrimSpace(mirror.Port)),
		Sslport:            common.String(strings.TrimSpace(mirror.Sslport)),
		IsSslverifyEnabled: common.Bool(mirror.IsSslverifyEnabled),
	}
	if strings.TrimSpace(mirror.Sslcert) != "" {
		details.Sslcert = common.String(mirror.Sslcert)
	}
	return details
}

func managementStationProxyEqual(current *osmanagementhubsdk.ProxyConfiguration, desired osmanagementhubv1beta1.ManagementStationProxy) bool {
	if current == nil {
		return !desired.IsEnabled && len(desired.Hosts) == 0 && strings.TrimSpace(desired.Port) == "" && strings.TrimSpace(desired.Forward) == ""
	}
	return managementStationBoolPtrEqual(current.IsEnabled, desired.IsEnabled) &&
		reflect.DeepEqual(current.Hosts, desired.Hosts) &&
		managementStationStringPtrEqual(current.Port, desired.Port) &&
		managementStationStringPtrEqual(current.Forward, desired.Forward)
}

func managementStationMirrorEqual(current *osmanagementhubsdk.MirrorConfiguration, desired osmanagementhubv1beta1.ManagementStationMirror) bool {
	if current == nil {
		return strings.TrimSpace(desired.Directory) == "" &&
			strings.TrimSpace(desired.Port) == "" &&
			strings.TrimSpace(desired.Sslport) == "" &&
			strings.TrimSpace(desired.Sslcert) == "" &&
			!desired.IsSslverifyEnabled
	}
	return managementStationStringPtrEqual(current.Directory, desired.Directory) &&
		managementStationStringPtrEqual(current.Port, desired.Port) &&
		managementStationStringPtrEqual(current.Sslport, desired.Sslport) &&
		managementStationStringPtrEqual(current.Sslcert, desired.Sslcert) &&
		managementStationBoolPtrEqual(current.IsSslverifyEnabled, desired.IsSslverifyEnabled)
}

func validateManagementStationCreateOnlyDriftForResponse(resource *osmanagementhubv1beta1.ManagementStation, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("ManagementStation resource is nil")
	}
	current, ok := managementStationFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current ManagementStation response does not expose a ManagementStation body")
	}
	return validateManagementStationCreateOnlyDrift(resource.Spec, current)
}

func validateManagementStationCreateOnlyDrift(
	spec osmanagementhubv1beta1.ManagementStationSpec,
	current osmanagementhubsdk.ManagementStation,
) error {
	if managementStationStringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		return nil
	}
	return fmt.Errorf("ManagementStation create-only field drift is not supported: compartmentId")
}

func confirmManagementStationDeleteRead(
	ctx context.Context,
	hooks *ManagementStationRuntimeHooks,
	resource *osmanagementhubv1beta1.ManagementStation,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm ManagementStation delete: runtime hooks are nil")
	}
	if currentID = strings.TrimSpace(currentID); currentID != "" {
		return confirmManagementStationDeleteReadByID(ctx, hooks, currentID)
	}
	return confirmManagementStationDeleteReadByIdentity(ctx, hooks, resource)
}

func confirmManagementStationDeleteReadByID(
	ctx context.Context,
	hooks *ManagementStationRuntimeHooks,
	currentID string,
) (any, error) {
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("confirm ManagementStation delete: get hook is not configured")
	}
	response, err := hooks.Get.Call(ctx, osmanagementhubsdk.GetManagementStationRequest{
		ManagementStationId: managementStationStringPointer(currentID),
	})
	return managementStationDeleteConfirmReadResponse(response, err)
}

func confirmManagementStationDeleteReadByIdentity(
	ctx context.Context,
	hooks *ManagementStationRuntimeHooks,
	resource *osmanagementhubv1beta1.ManagementStation,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("confirm ManagementStation delete: resource is nil")
	}
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm ManagementStation delete: list hook is not configured")
	}

	response, err := hooks.List.Call(ctx, osmanagementhubsdk.ListManagementStationsRequest{
		CompartmentId: managementStationStringPointer(resource.Spec.CompartmentId),
		DisplayName:   managementStationStringPointer(resource.Spec.DisplayName),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return managementStationAuthShapedConfirmRead{err: err}, nil
		}
		return nil, err
	}

	matches := matchingManagementStationSummaries(response.Items, resource.Spec)
	switch len(matches) {
	case 0:
		return nil, managementStationNotFoundError("ManagementStation delete confirmation did not find a matching OCI management station")
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("ManagementStation list response returned multiple matching resources for compartmentId %q, displayName %q, and hostname %q", resource.Spec.CompartmentId, resource.Spec.DisplayName, resource.Spec.Hostname)
	}
}

func managementStationDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return managementStationAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

func handleManagementStationDeleteError(resource *osmanagementhubv1beta1.ManagementStation, err error) error {
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
	return managementStationAmbiguousNotFoundError{
		message:      "ManagementStation delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func handleManagementStationDeleteConfirmReadOutcome(
	resource *osmanagementhubv1beta1.ManagementStation,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case managementStationAuthShapedConfirmRead:
		recordManagementStationConfirmReadRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *managementStationAuthShapedConfirmRead:
		if typed != nil {
			recordManagementStationConfirmReadRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func recordManagementStationConfirmReadRequestID(
	resource *osmanagementhubv1beta1.ManagementStation,
	err managementStationAuthShapedConfirmRead,
) {
	if resource == nil {
		return
	}
	servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, err.GetOpcRequestID())
}

func matchingManagementStationSummaries(
	items []osmanagementhubsdk.ManagementStationSummary,
	spec osmanagementhubv1beta1.ManagementStationSpec,
) []osmanagementhubsdk.ManagementStationSummary {
	matches := make([]osmanagementhubsdk.ManagementStationSummary, 0, len(items))
	for _, item := range items {
		if managementStationSummaryMatchesSpec(item, spec) {
			matches = append(matches, item)
		}
	}
	return matches
}

func managementStationSummaryMatchesSpec(
	summary osmanagementhubsdk.ManagementStationSummary,
	spec osmanagementhubv1beta1.ManagementStationSpec,
) bool {
	return managementStationStringValue(summary.CompartmentId) == spec.CompartmentId &&
		managementStationStringValue(summary.DisplayName) == spec.DisplayName &&
		managementStationStringValue(summary.Hostname) == spec.Hostname
}

func managementStationFromResponse(response any) (osmanagementhubsdk.ManagementStation, bool) {
	if current, ok := managementStationFromCreateResponse(response); ok {
		return current, true
	}
	if current, ok := managementStationFromGetResponse(response); ok {
		return current, true
	}
	if current, ok := managementStationFromUpdateResponse(response); ok {
		return current, true
	}
	if current, ok := managementStationFromEntityResponse(response); ok {
		return current, true
	}
	return osmanagementhubsdk.ManagementStation{}, false
}

func managementStationFromCreateResponse(response any) (osmanagementhubsdk.ManagementStation, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.CreateManagementStationResponse:
		return normalizeManagementStation(current.ManagementStation), true
	case *osmanagementhubsdk.CreateManagementStationResponse:
		if current == nil {
			return osmanagementhubsdk.ManagementStation{}, false
		}
		return normalizeManagementStation(current.ManagementStation), true
	default:
		return osmanagementhubsdk.ManagementStation{}, false
	}
}

func managementStationFromGetResponse(response any) (osmanagementhubsdk.ManagementStation, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.GetManagementStationResponse:
		return normalizeManagementStation(current.ManagementStation), true
	case *osmanagementhubsdk.GetManagementStationResponse:
		if current == nil {
			return osmanagementhubsdk.ManagementStation{}, false
		}
		return normalizeManagementStation(current.ManagementStation), true
	default:
		return osmanagementhubsdk.ManagementStation{}, false
	}
}

func managementStationFromUpdateResponse(response any) (osmanagementhubsdk.ManagementStation, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.UpdateManagementStationResponse:
		return normalizeManagementStation(current.ManagementStation), true
	case *osmanagementhubsdk.UpdateManagementStationResponse:
		if current == nil {
			return osmanagementhubsdk.ManagementStation{}, false
		}
		return normalizeManagementStation(current.ManagementStation), true
	default:
		return osmanagementhubsdk.ManagementStation{}, false
	}
}

func managementStationFromEntityResponse(response any) (osmanagementhubsdk.ManagementStation, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.ManagementStation:
		return normalizeManagementStation(current), true
	case *osmanagementhubsdk.ManagementStation:
		if current == nil {
			return osmanagementhubsdk.ManagementStation{}, false
		}
		return normalizeManagementStation(*current), true
	case osmanagementhubsdk.ManagementStationSummary:
		return managementStationFromSummary(current), true
	case *osmanagementhubsdk.ManagementStationSummary:
		if current == nil {
			return osmanagementhubsdk.ManagementStation{}, false
		}
		return managementStationFromSummary(*current), true
	default:
		return osmanagementhubsdk.ManagementStation{}, false
	}
}

func managementStationFromSummary(summary osmanagementhubsdk.ManagementStationSummary) osmanagementhubsdk.ManagementStation {
	return normalizeManagementStation(osmanagementhubsdk.ManagementStation{
		Id:                summary.Id,
		CompartmentId:     summary.CompartmentId,
		DisplayName:       summary.DisplayName,
		Hostname:          summary.Hostname,
		ManagedInstanceId: summary.ManagedInstanceId,
		ProfileId:         summary.ProfileId,
		ScheduledJobId:    summary.ScheduledJobId,
		Description:       summary.Description,
		OverallState:      summary.OverallState,
		OverallPercentage: summary.OverallPercentage,
		MirrorCapacity:    summary.MirrorCapacity,
		LifecycleState:    summary.LifecycleState,
		Location:          summary.Location,
		FreeformTags:      summary.FreeformTags,
		DefinedTags:       summary.DefinedTags,
		SystemTags:        summary.SystemTags,
	})
}

func normalizeManagementStationCreateCall(
	call func(context.Context, osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error),
) func(context.Context, osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request osmanagementhubsdk.CreateManagementStationRequest) (osmanagementhubsdk.CreateManagementStationResponse, error) {
		response, err := call(ctx, request)
		response.ManagementStation = normalizeManagementStation(response.ManagementStation)
		return response, err
	}
}

func normalizeManagementStationGetCall(
	call func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error),
) func(context.Context, osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request osmanagementhubsdk.GetManagementStationRequest) (osmanagementhubsdk.GetManagementStationResponse, error) {
		response, err := call(ctx, request)
		response.ManagementStation = normalizeManagementStation(response.ManagementStation)
		return response, err
	}
}

func normalizeManagementStationUpdateCall(
	call func(context.Context, osmanagementhubsdk.UpdateManagementStationRequest) (osmanagementhubsdk.UpdateManagementStationResponse, error),
) func(context.Context, osmanagementhubsdk.UpdateManagementStationRequest) (osmanagementhubsdk.UpdateManagementStationResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request osmanagementhubsdk.UpdateManagementStationRequest) (osmanagementhubsdk.UpdateManagementStationResponse, error) {
		response, err := call(ctx, request)
		response.ManagementStation = normalizeManagementStation(response.ManagementStation)
		return response, err
	}
}

func listManagementStationsAllPages(call managementStationListCall) managementStationListCall {
	return func(ctx context.Context, request osmanagementhubsdk.ListManagementStationsRequest) (osmanagementhubsdk.ListManagementStationsResponse, error) {
		if call == nil {
			return osmanagementhubsdk.ListManagementStationsResponse{}, fmt.Errorf("ManagementStation list operation is not configured")
		}
		return collectManagementStationListPages(ctx, call, request)
	}
}

func collectManagementStationListPages(
	ctx context.Context,
	call managementStationListCall,
	request osmanagementhubsdk.ListManagementStationsRequest,
) (osmanagementhubsdk.ListManagementStationsResponse, error) {
	seenPages := map[string]struct{}{}
	var combined osmanagementhubsdk.ListManagementStationsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return osmanagementhubsdk.ListManagementStationsResponse{}, err
		}
		appendManagementStationListPage(&combined, response)

		nextPage := managementStationStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return osmanagementhubsdk.ListManagementStationsResponse{}, fmt.Errorf("ManagementStation list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = managementStationStringPointer(nextPage)
	}
}

func appendManagementStationListPage(
	combined *osmanagementhubsdk.ListManagementStationsResponse,
	response osmanagementhubsdk.ListManagementStationsResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	for _, item := range response.Items {
		combined.Items = append(combined.Items, normalizeManagementStationSummary(item))
	}
}

func normalizeManagementStation(station osmanagementhubsdk.ManagementStation) osmanagementhubsdk.ManagementStation {
	station.DefinedTags = normalizeManagementStationTags(station.DefinedTags)
	station.SystemTags = normalizeManagementStationTags(station.SystemTags)
	return station
}

func normalizeManagementStationSummary(summary osmanagementhubsdk.ManagementStationSummary) osmanagementhubsdk.ManagementStationSummary {
	summary.DefinedTags = normalizeManagementStationTags(summary.DefinedTags)
	summary.SystemTags = normalizeManagementStationTags(summary.SystemTags)
	return summary
}

func normalizeManagementStationTags(tags map[string]map[string]interface{}) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	normalized := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		if values == nil {
			normalized[namespace] = nil
			continue
		}
		copied := make(map[string]interface{}, len(values))
		for key, value := range values {
			copied[key] = fmt.Sprint(value)
		}
		normalized[namespace] = copied
	}
	return normalized
}

func clearTrackedManagementStationIdentity(resource *osmanagementhubv1beta1.ManagementStation) {
	if resource == nil {
		return
	}
	resource.Status = osmanagementhubv1beta1.ManagementStationStatus{}
}

func markManagementStationDeleted(resource *osmanagementhubv1beta1.ManagementStation, message string, log loggerutil.OSOKLogger) {
	if resource == nil {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, log)
}

func managementStationNotFoundError(message string) error {
	return errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    message,
	}
}

func managementStationRecordedID(resource *osmanagementhubv1beta1.ManagementStation) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func desiredManagementStationFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneManagementStationStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredManagementStationDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return managementStationDefinedTags(spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func managementStationDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func cloneManagementStationStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func managementStationStringPointer(value string) *string {
	return common.String(strings.TrimSpace(value))
}

func managementStationStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func managementStationStringPtrEqual(current *string, desired string) bool {
	return managementStationStringValue(current) == strings.TrimSpace(desired)
}

func managementStationBoolPtrEqual(current *bool, desired bool) bool {
	return current != nil && *current == desired
}
