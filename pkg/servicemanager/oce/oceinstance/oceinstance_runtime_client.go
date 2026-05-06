/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package oceinstance

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocesdk "github.com/oracle/oci-go-sdk/v65/oce"
	ocev1beta1 "github.com/oracle/oci-service-operator/api/oce/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const oceInstanceKind = "OceInstance"

var oceInstanceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(ocesdk.WorkRequestStatusAccepted),
		string(ocesdk.WorkRequestStatusInProgress),
		string(ocesdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(ocesdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(ocesdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(ocesdk.WorkRequestStatusCanceled)},
	CreateActionTokens:    []string{string(ocesdk.WorkRequestOperationTypeCreateOceInstance)},
	UpdateActionTokens:    []string{string(ocesdk.WorkRequestOperationTypeUpdateOceInstance)},
	DeleteActionTokens:    []string{string(ocesdk.WorkRequestOperationTypeDeleteOceInstance)},
}

type oceInstanceOCIClient interface {
	CreateOceInstance(context.Context, ocesdk.CreateOceInstanceRequest) (ocesdk.CreateOceInstanceResponse, error)
	GetOceInstance(context.Context, ocesdk.GetOceInstanceRequest) (ocesdk.GetOceInstanceResponse, error)
	ListOceInstances(context.Context, ocesdk.ListOceInstancesRequest) (ocesdk.ListOceInstancesResponse, error)
	UpdateOceInstance(context.Context, ocesdk.UpdateOceInstanceRequest) (ocesdk.UpdateOceInstanceResponse, error)
	DeleteOceInstance(context.Context, ocesdk.DeleteOceInstanceRequest) (ocesdk.DeleteOceInstanceResponse, error)
	GetWorkRequest(context.Context, ocesdk.GetWorkRequestRequest) (ocesdk.GetWorkRequestResponse, error)
}

type oceInstanceListCall func(context.Context, ocesdk.ListOceInstancesRequest) (ocesdk.ListOceInstancesResponse, error)

func init() {
	registerOceInstanceRuntimeHooksMutator(func(manager *OceInstanceServiceManager, hooks *OceInstanceRuntimeHooks) {
		client, initErr := newOceInstanceOCIClient(manager)
		applyOceInstanceRuntimeHooks(hooks, client, initErr)
	})
}

func newOceInstanceOCIClient(manager *OceInstanceServiceManager) (oceInstanceOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", oceInstanceKind)
	}
	client, err := ocesdk.NewOceInstanceClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOceInstanceRuntimeHooks(
	hooks *OceInstanceRuntimeHooks,
	client oceInstanceOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedOceInstanceRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *ocev1beta1.OceInstance, _ string) (any, error) {
		return buildOceInstanceCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *ocev1beta1.OceInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildOceInstanceUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardOceInstanceExistingBeforeCreate
	hooks.ParityHooks.NormalizeDesiredState = normalizeOceInstanceDesiredState
	hooks.List.Fields = oceInstanceListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listOceInstancesAllPages(hooks.List.Call)
	}
	hooks.Async.Adapter = oceInstanceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getOceInstanceWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveOceInstanceWorkRequestAction
	hooks.Async.ResolvePhase = resolveOceInstanceWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverOceInstanceIDFromWorkRequest
	hooks.Async.Message = oceInstanceWorkRequestMessage
}

func newOceInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client oceInstanceOCIClient,
) OceInstanceServiceClient {
	return defaultOceInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*ocev1beta1.OceInstance](
			newOceInstanceRuntimeConfig(log, client),
		),
	}
}

func newOceInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	client oceInstanceOCIClient,
) generatedruntime.Config[*ocev1beta1.OceInstance] {
	hooks := newOceInstanceRuntimeHooksWithOCIClient(client)
	applyOceInstanceRuntimeHooks(&hooks, client, nil)
	return buildOceInstanceGeneratedRuntimeConfig(&OceInstanceServiceManager{Log: log}, hooks)
}

func newOceInstanceRuntimeHooksWithOCIClient(client oceInstanceOCIClient) OceInstanceRuntimeHooks {
	return OceInstanceRuntimeHooks{
		Semantics:       newOceInstanceRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*ocev1beta1.OceInstance]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*ocev1beta1.OceInstance]{},
		StatusHooks:     generatedruntime.StatusHooks[*ocev1beta1.OceInstance]{},
		ParityHooks:     generatedruntime.ParityHooks[*ocev1beta1.OceInstance]{},
		Async:           generatedruntime.AsyncHooks[*ocev1beta1.OceInstance]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*ocev1beta1.OceInstance]{},
		Create: runtimeOperationHooks[ocesdk.CreateOceInstanceRequest, ocesdk.CreateOceInstanceResponse]{
			Fields: oceInstanceCreateFields(),
			Call: func(ctx context.Context, request ocesdk.CreateOceInstanceRequest) (ocesdk.CreateOceInstanceResponse, error) {
				return client.CreateOceInstance(ctx, request)
			},
		},
		Get: runtimeOperationHooks[ocesdk.GetOceInstanceRequest, ocesdk.GetOceInstanceResponse]{
			Fields: oceInstanceGetFields(),
			Call: func(ctx context.Context, request ocesdk.GetOceInstanceRequest) (ocesdk.GetOceInstanceResponse, error) {
				return client.GetOceInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[ocesdk.ListOceInstancesRequest, ocesdk.ListOceInstancesResponse]{
			Fields: oceInstanceListFields(),
			Call: func(ctx context.Context, request ocesdk.ListOceInstancesRequest) (ocesdk.ListOceInstancesResponse, error) {
				return client.ListOceInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[ocesdk.UpdateOceInstanceRequest, ocesdk.UpdateOceInstanceResponse]{
			Fields: oceInstanceUpdateFields(),
			Call: func(ctx context.Context, request ocesdk.UpdateOceInstanceRequest) (ocesdk.UpdateOceInstanceResponse, error) {
				return client.UpdateOceInstance(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[ocesdk.DeleteOceInstanceRequest, ocesdk.DeleteOceInstanceResponse]{
			Fields: oceInstanceDeleteFields(),
			Call: func(ctx context.Context, request ocesdk.DeleteOceInstanceRequest) (ocesdk.DeleteOceInstanceResponse, error) {
				return client.DeleteOceInstance(ctx, request)
			},
		},
	}
}

func reviewedOceInstanceRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newOceInstanceRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{
		ProvisioningStates: []string{string(ocesdk.LifecycleStateCreating)},
		UpdatingStates:     []string{string(ocesdk.LifecycleStateUpdating)},
		ActiveStates:       []string{string(ocesdk.LifecycleStateActive)},
	}
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "name", "tenancyId"},
	}
	semantics.Mutation = generatedruntime.MutationSemantics{
		Mutable: []string{
			"addOnFeatures",
			"definedTags",
			"description",
			"drRegion",
			"freeformTags",
			"instanceLicenseType",
			"instanceUsageType",
			"wafPrimaryDomain",
		},
		ForceNew: []string{
			"adminEmail",
			"compartmentId",
			"identityStripe.serviceName",
			"identityStripe.tenancy",
			"instanceAccessType",
			"name",
			"objectStorageNamespace",
			"tenancyId",
			"tenancyName",
			"upgradeSchedule",
		},
		ConflictsWith: map[string][]string{},
	}
	semantics.Hooks = generatedruntime.HookSet{
		Create: []generatedruntime.Hook{
			{Helper: "tfresource.CreateResource", EntityType: "", Action: ""},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "oceinstance", Action: "CREATED"},
		},
		Update: []generatedruntime.Hook{
			{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "oceinstance", Action: "UPDATED"},
		},
		Delete: []generatedruntime.Hook{
			{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "oceinstance", Action: "DELETED"},
		},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetOceInstance",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Create...),
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetOceInstance",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Update...),
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetOceInstance/ListOceInstances confirm-delete",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Delete...),
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func oceInstanceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateOceInstanceDetails", RequestName: "CreateOceInstanceDetails", Contribution: "body"},
	}
}

func oceInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OceInstanceId", RequestName: "oceInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func oceInstanceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "TenancyId",
			RequestName:  "tenancyId",
			Contribution: "query",
			LookupPaths:  []string{"status.tenancyId", "spec.tenancyId", "tenancyId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.name", "spec.name", "name"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func oceInstanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OceInstanceId", RequestName: "oceInstanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateOceInstanceDetails", RequestName: "UpdateOceInstanceDetails", Contribution: "body"},
	}
}

func oceInstanceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OceInstanceId", RequestName: "oceInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func guardOceInstanceExistingBeforeCreate(
	_ context.Context,
	resource *ocev1beta1.OceInstance,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", oceInstanceKind)
	}
	if strings.TrimSpace(resource.Spec.Name) == "" ||
		strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.TenancyId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func normalizeOceInstanceDesiredState(resource *ocev1beta1.OceInstance, currentResponse any) {
	if resource == nil || currentResponse == nil {
		return
	}
	resource.Spec.IdcsAccessToken = ""
	resource.Spec.LifecycleDetails = ""
}

func buildOceInstanceCreateBody(resource *ocev1beta1.OceInstance) (ocesdk.CreateOceInstanceDetails, error) {
	if resource == nil {
		return ocesdk.CreateOceInstanceDetails{}, fmt.Errorf("%s resource is nil", oceInstanceKind)
	}

	spec := resource.Spec
	for field, value := range map[string]string{
		"compartmentId":          spec.CompartmentId,
		"name":                   spec.Name,
		"tenancyId":              spec.TenancyId,
		"idcsAccessToken":        spec.IdcsAccessToken,
		"tenancyName":            spec.TenancyName,
		"objectStorageNamespace": spec.ObjectStorageNamespace,
		"adminEmail":             spec.AdminEmail,
	} {
		if strings.TrimSpace(value) == "" {
			return ocesdk.CreateOceInstanceDetails{}, fmt.Errorf("%s spec.%s is required", oceInstanceKind, field)
		}
	}

	details := ocesdk.CreateOceInstanceDetails{
		CompartmentId:          common.String(spec.CompartmentId),
		Name:                   common.String(spec.Name),
		TenancyId:              common.String(spec.TenancyId),
		IdcsAccessToken:        common.String(spec.IdcsAccessToken),
		TenancyName:            common.String(spec.TenancyName),
		ObjectStorageNamespace: common.String(spec.ObjectStorageNamespace),
		AdminEmail:             common.String(spec.AdminEmail),
	}
	if spec.Description != "" {
		details.Description = common.String(spec.Description)
	}
	if identityStripe, err := oceInstanceIdentityStripeFromSpec(spec.IdentityStripe); err != nil {
		return ocesdk.CreateOceInstanceDetails{}, err
	} else if identityStripe != nil {
		details.IdentityStripe = identityStripe
	}
	if usageType, ok, err := oceInstanceCreateUsageType(spec.InstanceUsageType); err != nil {
		return ocesdk.CreateOceInstanceDetails{}, err
	} else if ok {
		details.InstanceUsageType = usageType
	}
	if len(spec.AddOnFeatures) > 0 {
		details.AddOnFeatures = slices.Clone(spec.AddOnFeatures)
	}
	if upgradeSchedule, ok, err := oceInstanceUpgradeSchedule(spec.UpgradeSchedule); err != nil {
		return ocesdk.CreateOceInstanceDetails{}, err
	} else if ok {
		details.UpgradeSchedule = upgradeSchedule
	}
	if spec.WafPrimaryDomain != "" {
		details.WafPrimaryDomain = common.String(spec.WafPrimaryDomain)
	}
	if accessType, ok, err := oceInstanceCreateAccessType(spec.InstanceAccessType); err != nil {
		return ocesdk.CreateOceInstanceDetails{}, err
	} else if ok {
		details.InstanceAccessType = accessType
	}
	if licenseType, ok, err := oceInstanceLicenseType(spec.InstanceLicenseType); err != nil {
		return ocesdk.CreateOceInstanceDetails{}, err
	} else if ok {
		details.InstanceLicenseType = licenseType
	}
	if spec.DrRegion != "" {
		details.DrRegion = common.String(spec.DrRegion)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = oceInstanceDefinedTagsFromSpec(spec.DefinedTags)
	}

	return details, nil
}

func buildOceInstanceUpdateBody(
	resource *ocev1beta1.OceInstance,
	currentResponse any,
) (ocesdk.UpdateOceInstanceDetails, bool, error) {
	if resource == nil {
		return ocesdk.UpdateOceInstanceDetails{}, false, fmt.Errorf("%s resource is nil", oceInstanceKind)
	}

	current, err := oceInstanceRuntimeBody(currentResponse)
	if err != nil {
		return ocesdk.UpdateOceInstanceDetails{}, false, err
	}

	details := ocesdk.UpdateOceInstanceDetails{}
	updateNeeded := false

	if desired, ok := oceInstanceDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := oceInstanceDesiredStringUpdate(resource.Spec.WafPrimaryDomain, current.WafPrimaryDomain); ok {
		details.WafPrimaryDomain = desired
		updateNeeded = true
	}
	if desired, ok, err := oceInstanceDesiredLicenseTypeUpdate(resource.Spec.InstanceLicenseType, current.InstanceLicenseType); err != nil {
		return ocesdk.UpdateOceInstanceDetails{}, false, err
	} else if ok {
		details.InstanceLicenseType = desired
		updateNeeded = true
	}
	if desired, ok, err := oceInstanceDesiredUsageTypeUpdate(resource.Spec.InstanceUsageType, current.InstanceUsageType); err != nil {
		return ocesdk.UpdateOceInstanceDetails{}, false, err
	} else if ok {
		details.InstanceUsageType = desired
		updateNeeded = true
	}
	if desired, ok := oceInstanceDesiredAddOnFeaturesUpdate(resource.Spec.AddOnFeatures, current.AddOnFeatures); ok {
		details.AddOnFeatures = desired
		updateNeeded = true
	}
	if desired, ok := oceInstanceDesiredStringUpdate(resource.Spec.DrRegion, current.DrRegion); ok {
		details.DrRegion = desired
		updateNeeded = true
	}
	if desired, ok := oceInstanceDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := oceInstanceDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func oceInstanceRuntimeBody(currentResponse any) (ocesdk.OceInstance, error) {
	switch current := currentResponse.(type) {
	case ocesdk.OceInstance:
		return current, nil
	case *ocesdk.OceInstance:
		if current == nil {
			return ocesdk.OceInstance{}, fmt.Errorf("current %s response is nil", oceInstanceKind)
		}
		return *current, nil
	case ocesdk.OceInstanceSummary:
		return ocesdk.OceInstance{
			Id:                     current.Id,
			Guid:                   current.Guid,
			CompartmentId:          current.CompartmentId,
			Name:                   current.Name,
			TenancyId:              current.TenancyId,
			IdcsTenancy:            current.IdcsTenancy,
			TenancyName:            current.TenancyName,
			ObjectStorageNamespace: current.ObjectStorageNamespace,
			AdminEmail:             current.AdminEmail,
			Description:            current.Description,
			InstanceUsageType:      ocesdk.OceInstanceInstanceUsageTypeEnum(current.InstanceUsageType),
			AddOnFeatures:          slices.Clone(current.AddOnFeatures),
			UpgradeSchedule:        current.UpgradeSchedule,
			WafPrimaryDomain:       current.WafPrimaryDomain,
			InstanceAccessType:     ocesdk.OceInstanceInstanceAccessTypeEnum(current.InstanceAccessType),
			InstanceLicenseType:    current.InstanceLicenseType,
			TimeCreated:            current.TimeCreated,
			TimeUpdated:            current.TimeUpdated,
			LifecycleState:         current.LifecycleState,
			LifecycleDetails:       current.LifecycleDetails,
			DrRegion:               current.DrRegion,
			StateMessage:           current.StateMessage,
			Service:                current.Service,
			FreeformTags:           maps.Clone(current.FreeformTags),
			DefinedTags:            current.DefinedTags,
			SystemTags:             current.SystemTags,
		}, nil
	case *ocesdk.OceInstanceSummary:
		if current == nil {
			return ocesdk.OceInstance{}, fmt.Errorf("current %s response is nil", oceInstanceKind)
		}
		return oceInstanceRuntimeBody(*current)
	case ocesdk.GetOceInstanceResponse:
		return current.OceInstance, nil
	case *ocesdk.GetOceInstanceResponse:
		if current == nil {
			return ocesdk.OceInstance{}, fmt.Errorf("current %s response is nil", oceInstanceKind)
		}
		return current.OceInstance, nil
	default:
		return ocesdk.OceInstance{}, fmt.Errorf("unexpected current %s response type %T", oceInstanceKind, currentResponse)
	}
}

func oceInstanceDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func oceInstanceDesiredAddOnFeaturesUpdate(spec []string, current []string) ([]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if slices.Equal(spec, current) {
		return nil, false
	}
	return slices.Clone(spec), true
}

func oceInstanceDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func oceInstanceDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := oceInstanceDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func oceInstanceDesiredLicenseTypeUpdate(
	spec string,
	current ocesdk.LicenseTypeEnum,
) (ocesdk.LicenseTypeEnum, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == string(current) {
		return "", false, nil
	}
	licenseType, ok := ocesdk.GetMappingLicenseTypeEnum(spec)
	if !ok {
		return "", false, fmt.Errorf("%s spec.instanceLicenseType %q is not supported", oceInstanceKind, spec)
	}
	return licenseType, true, nil
}

func oceInstanceDesiredUsageTypeUpdate(
	spec string,
	current ocesdk.OceInstanceInstanceUsageTypeEnum,
) (ocesdk.UpdateOceInstanceDetailsInstanceUsageTypeEnum, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == string(current) {
		return "", false, nil
	}
	usageType, ok := ocesdk.GetMappingUpdateOceInstanceDetailsInstanceUsageTypeEnum(spec)
	if !ok {
		return "", false, fmt.Errorf("%s spec.instanceUsageType %q is not supported", oceInstanceKind, spec)
	}
	return usageType, true, nil
}

func oceInstanceIdentityStripeFromSpec(
	spec ocev1beta1.OceInstanceIdentityStripe,
) (*ocesdk.IdentityStripeDetails, error) {
	serviceName := strings.TrimSpace(spec.ServiceName)
	tenancy := strings.TrimSpace(spec.Tenancy)
	switch {
	case serviceName == "" && tenancy == "":
		return nil, nil
	case serviceName == "" || tenancy == "":
		return nil, fmt.Errorf("%s spec.identityStripe requires both serviceName and tenancy", oceInstanceKind)
	default:
		return &ocesdk.IdentityStripeDetails{
			ServiceName: common.String(serviceName),
			Tenancy:     common.String(tenancy),
		}, nil
	}
}

func oceInstanceCreateUsageType(spec string) (ocesdk.CreateOceInstanceDetailsInstanceUsageTypeEnum, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", false, nil
	}
	usageType, ok := ocesdk.GetMappingCreateOceInstanceDetailsInstanceUsageTypeEnum(spec)
	if !ok {
		return "", false, fmt.Errorf("%s spec.instanceUsageType %q is not supported", oceInstanceKind, spec)
	}
	return usageType, true, nil
}

func oceInstanceCreateAccessType(spec string) (ocesdk.CreateOceInstanceDetailsInstanceAccessTypeEnum, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", false, nil
	}
	accessType, ok := ocesdk.GetMappingCreateOceInstanceDetailsInstanceAccessTypeEnum(spec)
	if !ok {
		return "", false, fmt.Errorf("%s spec.instanceAccessType %q is not supported", oceInstanceKind, spec)
	}
	return accessType, true, nil
}

func oceInstanceUpgradeSchedule(spec string) (ocesdk.OceInstanceUpgradeScheduleEnum, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", false, nil
	}
	upgradeSchedule, ok := ocesdk.GetMappingOceInstanceUpgradeScheduleEnum(spec)
	if !ok {
		return "", false, fmt.Errorf("%s spec.upgradeSchedule %q is not supported", oceInstanceKind, spec)
	}
	return upgradeSchedule, true, nil
}

func oceInstanceLicenseType(spec string) (ocesdk.LicenseTypeEnum, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", false, nil
	}
	licenseType, ok := ocesdk.GetMappingLicenseTypeEnum(spec)
	if !ok {
		return "", false, fmt.Errorf("%s spec.instanceLicenseType %q is not supported", oceInstanceKind, spec)
	}
	return licenseType, true, nil
}

func oceInstanceDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		converted[namespace] = child
	}
	return converted
}

func getOceInstanceWorkRequest(
	ctx context.Context,
	client oceInstanceOCIClient,
	initErr error,
	workRequestID string,
) (ocesdk.WorkRequest, error) {
	if initErr != nil {
		return ocesdk.WorkRequest{}, initErr
	}
	if client == nil {
		return ocesdk.WorkRequest{}, fmt.Errorf("%s work request client is not configured", oceInstanceKind)
	}
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return ocesdk.WorkRequest{}, fmt.Errorf("%s work request ID is required", oceInstanceKind)
	}
	response, err := client.GetWorkRequest(ctx, ocesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return ocesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func resolveOceInstanceWorkRequestAction(workRequest any) (string, error) {
	current, err := oceInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveOceInstanceWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := oceInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case ocesdk.WorkRequestOperationTypeCreateOceInstance:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case ocesdk.WorkRequestOperationTypeUpdateOceInstance:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case ocesdk.WorkRequestOperationTypeDeleteOceInstance:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverOceInstanceIDFromWorkRequest(
	_ *ocev1beta1.OceInstance,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := oceInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return oceInstanceIDFromWorkRequest(current, oceInstanceActionForPhase(phase))
}

func oceInstanceWorkRequestFromAny(workRequest any) (ocesdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case ocesdk.WorkRequest:
		return current, nil
	case *ocesdk.WorkRequest:
		if current == nil {
			return ocesdk.WorkRequest{}, fmt.Errorf("%s work request is nil", oceInstanceKind)
		}
		return *current, nil
	default:
		return ocesdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", oceInstanceKind, workRequest)
	}
}

func oceInstanceActionForPhase(phase shared.OSOKAsyncPhase) ocesdk.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return ocesdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return ocesdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return ocesdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func oceInstanceIDFromWorkRequest(
	workRequest ocesdk.WorkRequest,
	action ocesdk.WorkRequestResourceActionTypeEnum,
) (string, error) {
	if id, ok := oceInstanceIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := oceInstanceIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose an OceInstance identifier", oceInstanceKind, oceInstanceStringValue(workRequest.Id))
}

func oceInstanceIDFromResources(
	resources []ocesdk.WorkRequestResource,
	action ocesdk.WorkRequestResourceActionTypeEnum,
	preferResourceOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferResourceOnly && !isOceInstanceWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(oceInstanceStringValue(resource.Identifier))
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

func isOceInstanceWorkRequestResource(resource ocesdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(oceInstanceStringValue(resource.EntityType)))
	switch entityType {
	case "oceinstance", "oceinstances", "oce_instance", "oce_instances":
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(oceInstanceStringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/oceinstances/") || strings.Contains(entityURI, "oceinstance")
}

func oceInstanceWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := oceInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", oceInstanceKind, phase, oceInstanceStringValue(current.Id), current.Status)
}

func listOceInstancesAllPages(call oceInstanceListCall) oceInstanceListCall {
	return func(ctx context.Context, request ocesdk.ListOceInstancesRequest) (ocesdk.ListOceInstancesResponse, error) {
		if call == nil {
			return ocesdk.ListOceInstancesResponse{}, fmt.Errorf("%s list operation is not configured", oceInstanceKind)
		}
		if !oceInstanceHasBoundedListRequest(request) {
			return ocesdk.ListOceInstancesResponse{}, nil
		}

		accumulator := newOceInstanceListAccumulator()
		for {
			response, err := call(ctx, request)
			if err != nil {
				return ocesdk.ListOceInstancesResponse{}, err
			}
			accumulator.append(response)

			nextPage := oceInstanceStringValue(response.OpcNextPage)
			if nextPage == "" {
				return accumulator.response, nil
			}
			if err := accumulator.advance(&request, nextPage); err != nil {
				return ocesdk.ListOceInstancesResponse{}, err
			}
		}
	}
}

func oceInstanceHasBoundedListRequest(request ocesdk.ListOceInstancesRequest) bool {
	return strings.TrimSpace(oceInstanceStringValue(request.CompartmentId)) != "" &&
		strings.TrimSpace(oceInstanceStringValue(request.TenancyId)) != "" &&
		strings.TrimSpace(oceInstanceStringValue(request.DisplayName)) != ""
}

type oceInstanceListAccumulator struct {
	response  ocesdk.ListOceInstancesResponse
	seenPages map[string]struct{}
}

func newOceInstanceListAccumulator() oceInstanceListAccumulator {
	return oceInstanceListAccumulator{seenPages: map[string]struct{}{}}
}

func (a *oceInstanceListAccumulator) append(response ocesdk.ListOceInstancesResponse) {
	if a.response.RawResponse == nil {
		a.response.RawResponse = response.RawResponse
	}
	if a.response.OpcRequestId == nil {
		a.response.OpcRequestId = response.OpcRequestId
	}
	a.response.OpcNextPage = response.OpcNextPage
	a.response.Items = append(a.response.Items, response.Items...)
}

func (a *oceInstanceListAccumulator) advance(request *ocesdk.ListOceInstancesRequest, nextPage string) error {
	if _, ok := a.seenPages[nextPage]; ok {
		return fmt.Errorf("%s list pagination repeated page token %q", oceInstanceKind, nextPage)
	}
	a.seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	return nil
}

func oceInstanceStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
