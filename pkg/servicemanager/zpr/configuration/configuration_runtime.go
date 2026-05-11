/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package configuration

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	zprsdk "github.com/oracle/oci-go-sdk/v65/zpr"
	zprv1beta1 "github.com/oracle/oci-service-operator/api/zpr/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

var configurationWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(zprsdk.WorkRequestStatusAccepted),
		string(zprsdk.WorkRequestStatusInProgress),
		string(zprsdk.WorkRequestStatusWaiting),
		string(zprsdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(zprsdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(zprsdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(zprsdk.WorkRequestStatusCanceled)},
	AttentionStatusTokens: []string{string(zprsdk.WorkRequestStatusNeedsAttention)},
	CreateActionTokens:    []string{string(zprsdk.ActionTypeCreated)},
}

type configurationOCIClient interface {
	CreateConfiguration(context.Context, zprsdk.CreateConfigurationRequest) (zprsdk.CreateConfigurationResponse, error)
	GetConfiguration(context.Context, zprsdk.GetConfigurationRequest) (zprsdk.GetConfigurationResponse, error)
	GetZprConfigurationWorkRequest(context.Context, zprsdk.GetZprConfigurationWorkRequestRequest) (zprsdk.GetZprConfigurationWorkRequestResponse, error)
}

type configurationIdentity struct {
	compartmentID string
}

type configurationRuntimeClient struct {
	delegate ConfigurationServiceClient
	client   configurationOCIClient
	initErr  error
}

var _ ConfigurationServiceClient = (*configurationRuntimeClient)(nil)

func init() {
	registerConfigurationRuntimeHooksMutator(func(manager *ConfigurationServiceManager, hooks *ConfigurationRuntimeHooks) {
		client, initErr := newConfigurationSDKClient(manager)
		applyConfigurationRuntimeHooks(hooks, client, initErr)
	})
}

func newConfigurationSDKClient(manager *ConfigurationServiceManager) (configurationOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Configuration service manager is nil")
	}
	client, err := zprsdk.NewZprClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyConfigurationRuntimeHooks(
	hooks *ConfigurationRuntimeHooks,
	client configurationOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	runtimeClient := &configurationRuntimeClient{
		client:  client,
		initErr: initErr,
	}

	hooks.BuildCreateBody = func(_ context.Context, resource *zprv1beta1.Configuration, _ string) (any, error) {
		return buildCreateConfigurationDetails(resource)
	}
	hooks.Get.Fields = configurationGetFields()
	hooks.Identity.Resolve = resolveConfigurationIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardConfigurationExistingBeforeCreate
	hooks.Identity.LookupExisting = func(ctx context.Context, _ *zprv1beta1.Configuration, identity any) (any, error) {
		return lookupExistingConfiguration(ctx, client, initErr, identity)
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedConfigurationIdentity
	hooks.StatusHooks.ProjectStatus = projectConfigurationStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateConfigurationCreateOnlyDriftForResponse
	hooks.Async.Adapter = configurationWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getConfigurationWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveConfigurationGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveConfigurationGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverConfigurationIDFromGeneratedWorkRequest
	hooks.Async.Message = configurationGeneratedWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ConfigurationServiceClient) ConfigurationServiceClient {
		wrapped := *runtimeClient
		wrapped.delegate = delegate
		return &wrapped
	})
}

func configurationGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
	}
}

func buildCreateConfigurationDetails(resource *zprv1beta1.Configuration) (zprsdk.CreateConfigurationDetails, error) {
	if resource == nil {
		return zprsdk.CreateConfigurationDetails{}, fmt.Errorf("Configuration resource is nil")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return zprsdk.CreateConfigurationDetails{}, fmt.Errorf("Configuration spec.compartmentId is required")
	}

	details := zprsdk.CreateConfigurationDetails{
		CompartmentId: common.String(compartmentID),
	}
	if zprStatus := strings.TrimSpace(resource.Spec.ZprStatus); zprStatus != "" {
		details.ZprStatus = zprsdk.ConfigurationZprStatusEnum(zprStatus)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneConfigurationStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		converted := util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if converted != nil {
			details.DefinedTags = *converted
		}
	}
	return details, nil
}

func resolveConfigurationIdentity(resource *zprv1beta1.Configuration) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("Configuration resource is nil")
	}
	return configurationIdentity{
		compartmentID: currentConfigurationCompartmentID(resource),
	}, nil
}

func guardConfigurationExistingBeforeCreate(
	_ context.Context,
	resource *zprv1beta1.Configuration,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Configuration resource is nil")
	}
	if currentConfigurationCompartmentID(resource) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Configuration spec.compartmentId is required")
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupExistingConfiguration(
	ctx context.Context,
	client configurationOCIClient,
	initErr error,
	identity any,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Configuration OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Configuration OCI client is not configured")
	}

	resolved, ok := identity.(configurationIdentity)
	if !ok {
		return nil, fmt.Errorf("unexpected Configuration identity type %T", identity)
	}
	if strings.TrimSpace(resolved.compartmentID) == "" {
		return nil, nil
	}

	response, err := client.GetConfiguration(ctx, zprsdk.GetConfigurationRequest{
		CompartmentId: common.String(strings.TrimSpace(resolved.compartmentID)),
	})
	if err != nil {
		return nil, err
	}
	if response.Configuration.LifecycleState == zprsdk.ConfigurationLifecycleStateDeleted {
		return nil, nil
	}
	return response, nil
}

func clearTrackedConfigurationIdentity(resource *zprv1beta1.Configuration) {
	if resource == nil {
		return
	}
	resource.Status = zprv1beta1.ConfigurationStatus{}
}

func projectConfigurationStatus(resource *zprv1beta1.Configuration, response any) error {
	if resource == nil {
		return fmt.Errorf("Configuration resource is nil")
	}

	current, ok := configurationFromResponse(response)
	if !ok {
		return nil
	}

	resource.Status = zprv1beta1.ConfigurationStatus{
		OsokStatus:       resource.Status.OsokStatus,
		Id:               stringValue(current.Id),
		CompartmentId:    stringValue(current.CompartmentId),
		ZprStatus:        string(current.ZprStatus),
		TimeCreated:      sdkTimeString(current.TimeCreated),
		TimeUpdated:      sdkTimeString(current.TimeUpdated),
		LifecycleState:   string(current.LifecycleState),
		LifecycleDetails: stringValue(current.LifecycleDetails),
		FreeformTags:     cloneConfigurationStringMap(current.FreeformTags),
		DefinedTags:      convertConfigurationOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:       convertConfigurationOCIToStatusDefinedTags(current.SystemTags),
	}
	return nil
}

func configurationFromResponse(response any) (zprsdk.Configuration, bool) {
	switch current := response.(type) {
	case zprsdk.GetConfigurationResponse:
		return current.Configuration, true
	case *zprsdk.GetConfigurationResponse:
		if current == nil {
			return zprsdk.Configuration{}, false
		}
		return current.Configuration, true
	case zprsdk.Configuration:
		return current, true
	case *zprsdk.Configuration:
		if current == nil {
			return zprsdk.Configuration{}, false
		}
		return *current, true
	default:
		return zprsdk.Configuration{}, false
	}
}

func validateConfigurationCreateOnlyDriftForResponse(resource *zprv1beta1.Configuration, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("Configuration resource is nil")
	}

	current, ok := configurationFromResponse(currentResponse)
	if !ok {
		return nil
	}

	if desiredCompartment := strings.TrimSpace(resource.Spec.CompartmentId); desiredCompartment != "" {
		if currentCompartment := stringValue(current.CompartmentId); currentCompartment != "" && currentCompartment != desiredCompartment {
			return fmt.Errorf("Configuration formal semantics require replacement when compartmentId changes")
		}
	}

	if desiredStatus := strings.TrimSpace(resource.Spec.ZprStatus); desiredStatus != "" && !strings.EqualFold(desiredStatus, string(current.ZprStatus)) {
		return fmt.Errorf("Configuration formal semantics reject unsupported update drift for zprStatus")
	}
	if !configurationFreeformTagsMatch(resource.Spec.FreeformTags, current.FreeformTags) {
		return fmt.Errorf("Configuration formal semantics reject unsupported update drift for freeformTags")
	}
	if !configurationDefinedTagsMatch(resource.Spec.DefinedTags, current.DefinedTags) {
		return fmt.Errorf("Configuration formal semantics reject unsupported update drift for definedTags")
	}
	return nil
}

func configurationFreeformTagsMatch(spec map[string]string, current map[string]string) bool {
	if spec == nil {
		return true
	}
	if len(spec) == 0 && len(current) == 0 {
		return true
	}
	return reflect.DeepEqual(spec, current)
}

func configurationDefinedTagsMatch(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) bool {
	if spec == nil {
		return true
	}
	converted := util.ConvertToOciDefinedTags(&spec)
	if converted == nil {
		return len(current) == 0
	}
	if len(*converted) == 0 && len(current) == 0 {
		return true
	}
	return reflect.DeepEqual(*converted, current)
}

func getConfigurationWorkRequest(
	ctx context.Context,
	client configurationOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Configuration OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Configuration OCI client is not configured")
	}

	response, err := client.GetZprConfigurationWorkRequest(ctx, zprsdk.GetZprConfigurationWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveConfigurationGeneratedWorkRequestAction(workRequest any) (string, error) {
	zprWorkRequest, err := configurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveConfigurationWorkRequestAction(zprWorkRequest)
}

func resolveConfigurationGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	zprWorkRequest, err := configurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := configurationWorkRequestPhaseFromOperationType(zprWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverConfigurationIDFromGeneratedWorkRequest(
	_ *zprv1beta1.Configuration,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	zprWorkRequest, err := configurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveConfigurationIDFromWorkRequest(zprWorkRequest, configurationWorkRequestActionForPhase(phase))
}

func configurationGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	zprWorkRequest, err := configurationWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Configuration %s work request %s is %s", phase, stringValue(zprWorkRequest.Id), zprWorkRequest.Status)
}

func configurationWorkRequestFromAny(workRequest any) (zprsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case zprsdk.WorkRequest:
		return current, nil
	case *zprsdk.WorkRequest:
		if current == nil {
			return zprsdk.WorkRequest{}, fmt.Errorf("Configuration work request is nil")
		}
		return *current, nil
	default:
		return zprsdk.WorkRequest{}, fmt.Errorf("unexpected Configuration work request type %T", workRequest)
	}
}

func configurationWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) zprsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return zprsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return zprsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return zprsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveConfigurationIDFromWorkRequest(workRequest zprsdk.WorkRequest, action zprsdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveConfigurationIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveConfigurationIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("Configuration work request %s does not expose a Configuration identifier", stringValue(workRequest.Id))
}

func resolveConfigurationIDFromResources(
	resources []zprsdk.WorkRequestResource,
	action zprsdk.ActionTypeEnum,
	preferConfigurationOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferConfigurationOnly && !isConfigurationWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func resolveConfigurationWorkRequestAction(workRequest zprsdk.WorkRequest) (string, error) {
	var action string

	for _, resource := range workRequest.Resources {
		if !isConfigurationWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf(
				"Configuration work request %s exposes conflicting Configuration action types %q and %q",
				stringValue(workRequest.Id),
				action,
				candidate,
			)
		}
	}

	return action, nil
}

func configurationWorkRequestPhaseFromOperationType(operationType zprsdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case zprsdk.OperationTypeCreateZprConfiguration:
		return shared.OSOKAsyncPhaseCreate, true
	case zprsdk.OperationTypeUpdateZprConfiguration:
		return shared.OSOKAsyncPhaseUpdate, true
	case zprsdk.OperationTypeDeleteZprConfiguration:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func isConfigurationWorkRequestResource(resource zprsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "configuration", "configurations", "zprconfiguration", "zpr_configuration":
		return true
	}
	if strings.Contains(entityType, "configuration") {
		return true
	}

	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/configuration")
}

func (c *configurationRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *zprv1beta1.Configuration,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Configuration generated runtime delegate is not configured")
	}

	if response, err, handled := c.handleStaleTrackedIdentity(ctx, resource, req); handled {
		return response, err
	}

	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *configurationRuntimeClient) Delete(_ context.Context, _ *zprv1beta1.Configuration) (bool, error) {
	return true, nil
}

func (c *configurationRuntimeClient) handleStaleTrackedIdentity(
	ctx context.Context,
	resource *zprv1beta1.Configuration,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error, bool) {
	if resource == nil || c.client == nil || c.initErr != nil {
		return servicemanager.OSOKResponse{}, nil, false
	}

	if currentConfigurationID(resource) == "" {
		return servicemanager.OSOKResponse{}, nil, false
	}

	trackedCompartmentID := trackedConfigurationCompartmentID(resource)
	if trackedCompartmentID == "" {
		return servicemanager.OSOKResponse{}, nil, false
	}

	current, err := c.get(ctx, trackedCompartmentID)
	if err != nil {
		if !servicemanager.IsNotFoundServiceError(err) {
			return servicemanager.OSOKResponse{}, nil, false
		}
		if desiredCompartmentID := strings.TrimSpace(resource.Spec.CompartmentId); desiredCompartmentID != "" && desiredCompartmentID != trackedCompartmentID {
			return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Configuration formal semantics require replacement when compartmentId changes"), true
		}
		clearTrackedConfigurationIdentity(resource)
		response, err := c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
		return response, err, true
	}
	if current.LifecycleState != zprsdk.ConfigurationLifecycleStateDeleted {
		return servicemanager.OSOKResponse{}, nil, false
	}

	clearTrackedConfigurationIdentity(resource)
	response, err := c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
	return response, err, true
}

func (c *configurationRuntimeClient) get(
	ctx context.Context,
	compartmentID string,
) (zprsdk.Configuration, error) {
	if c.initErr != nil {
		return zprsdk.Configuration{}, fmt.Errorf("initialize Configuration OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return zprsdk.Configuration{}, fmt.Errorf("Configuration OCI client is not configured")
	}
	if strings.TrimSpace(compartmentID) == "" {
		return zprsdk.Configuration{}, fmt.Errorf("Configuration get requires a compartmentId")
	}

	response, err := c.client.GetConfiguration(ctx, zprsdk.GetConfigurationRequest{
		CompartmentId: common.String(strings.TrimSpace(compartmentID)),
	})
	if err != nil {
		return zprsdk.Configuration{}, err
	}
	return response.Configuration, nil
}

func currentConfigurationID(resource *zprv1beta1.Configuration) string {
	if resource == nil {
		return ""
	}
	if trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); trackedID != "" {
		return trackedID
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentConfigurationCompartmentID(resource *zprv1beta1.Configuration) string {
	if resource == nil {
		return ""
	}
	if compartmentID := strings.TrimSpace(resource.Spec.CompartmentId); compartmentID != "" {
		return compartmentID
	}
	return strings.TrimSpace(resource.Status.CompartmentId)
}

func trackedConfigurationCompartmentID(resource *zprv1beta1.Configuration) string {
	if resource == nil {
		return ""
	}
	if compartmentID := strings.TrimSpace(resource.Status.CompartmentId); compartmentID != "" {
		return compartmentID
	}
	return strings.TrimSpace(resource.Spec.CompartmentId)
}

func cloneConfigurationStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func convertConfigurationOCIToStatusDefinedTags(values map[string]map[string]interface{}) map[string]shared.MapValue {
	if values == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(values))
	for namespace, entries := range values {
		convertedEntries := make(shared.MapValue, len(entries))
		for key, value := range entries {
			convertedEntries[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedEntries
	}
	return converted
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}
