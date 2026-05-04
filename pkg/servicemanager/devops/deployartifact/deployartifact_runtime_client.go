/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package deployartifact

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const (
	deployArtifactKind                       = "DeployArtifact"
	deployArtifactAmbiguousNotFoundErrorCode = "DeployArtifactAmbiguousNotFound"
)

var deployArtifactWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(devopssdk.OperationStatusAccepted),
		string(devopssdk.OperationStatusInProgress),
		string(devopssdk.OperationStatusWaiting),
		string(devopssdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(devopssdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(devopssdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(devopssdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(devopssdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(devopssdk.OperationTypeCreateDeployArtifact)},
	UpdateActionTokens:    []string{string(devopssdk.OperationTypeUpdateDeployArtifact)},
	DeleteActionTokens:    []string{string(devopssdk.OperationTypeDeleteDeployArtifact)},
}

type deployArtifactOCIClient interface {
	CreateDeployArtifact(context.Context, devopssdk.CreateDeployArtifactRequest) (devopssdk.CreateDeployArtifactResponse, error)
	GetDeployArtifact(context.Context, devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error)
	ListDeployArtifacts(context.Context, devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error)
	UpdateDeployArtifact(context.Context, devopssdk.UpdateDeployArtifactRequest) (devopssdk.UpdateDeployArtifactResponse, error)
	DeleteDeployArtifact(context.Context, devopssdk.DeleteDeployArtifactRequest) (devopssdk.DeleteDeployArtifactResponse, error)
	GetWorkRequest(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

type deployArtifactAmbiguousNotFoundError struct {
	HTTPStatusCode int
	ErrorCode      string
	OpcRequestID   string
	message        string
}

func (e deployArtifactAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e deployArtifactAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.OpcRequestID
}

type deployArtifactAmbiguousDeleteConfirmResponse struct {
	DeployArtifact devopssdk.DeployArtifact `presentIn:"body"`
	err            error
}

func init() {
	registerDeployArtifactRuntimeHooksMutator(func(manager *DeployArtifactServiceManager, hooks *DeployArtifactRuntimeHooks) {
		client, initErr := newDeployArtifactSDKClient(manager)
		applyDeployArtifactRuntimeHooks(hooks, client, initErr)
	})
}

func newDeployArtifactSDKClient(manager *DeployArtifactServiceManager) (deployArtifactOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", deployArtifactKind)
	}
	client, err := devopssdk.NewDevopsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyDeployArtifactRuntimeHooks(
	hooks *DeployArtifactRuntimeHooks,
	client deployArtifactOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newDeployArtifactRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *devopsv1beta1.DeployArtifact, _ string) (any, error) {
		return buildDeployArtifactCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *devopsv1beta1.DeployArtifact,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDeployArtifactUpdateBody(resource, currentResponse)
	}
	hooks.Get.Call = func(ctx context.Context, request devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
		if initErr != nil {
			return devopssdk.GetDeployArtifactResponse{}, fmt.Errorf("initialize %s OCI client: %w", deployArtifactKind, initErr)
		}
		if client == nil {
			return devopssdk.GetDeployArtifactResponse{}, fmt.Errorf("%s OCI client is not configured", deployArtifactKind)
		}
		response, err := client.GetDeployArtifact(ctx, request)
		return response, conservativeDeployArtifactNotFoundError(err, "read")
	}
	hooks.List.Fields = deployArtifactListFields()
	hooks.List.Call = func(ctx context.Context, request devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error) {
		return listDeployArtifactsAllPages(ctx, client, initErr, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request devopssdk.DeleteDeployArtifactRequest) (devopssdk.DeleteDeployArtifactResponse, error) {
		if initErr != nil {
			return devopssdk.DeleteDeployArtifactResponse{}, fmt.Errorf("initialize %s OCI client: %w", deployArtifactKind, initErr)
		}
		if client == nil {
			return devopssdk.DeleteDeployArtifactResponse{}, fmt.Errorf("%s OCI client is not configured", deployArtifactKind)
		}
		response, err := client.DeleteDeployArtifact(ctx, request)
		return response, conservativeDeployArtifactNotFoundError(err, "delete")
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedDeployArtifactIdentity
	hooks.StatusHooks.ProjectStatus = deployArtifactStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDeployArtifactCreateOnlyDriftForResponse
	hooks.DeleteHooks.ConfirmRead = deployArtifactDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleDeployArtifactDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDeployArtifactDeleteOutcome
	hooks.Async.Adapter = deployArtifactWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getDeployArtifactWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveDeployArtifactWorkRequestAction
	hooks.Async.ResolvePhase = resolveDeployArtifactWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverDeployArtifactIDFromWorkRequest
	hooks.Async.Message = deployArtifactWorkRequestMessage
}

func newDeployArtifactServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client deployArtifactOCIClient,
) DeployArtifactServiceClient {
	hooks := newDeployArtifactRuntimeHooksWithOCIClient(client)
	applyDeployArtifactRuntimeHooks(&hooks, client, nil)
	manager := &DeployArtifactServiceManager{Log: log}
	delegate := defaultDeployArtifactServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*devopsv1beta1.DeployArtifact](
			buildDeployArtifactGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDeployArtifactGeneratedClient(hooks, delegate)
}

func newDeployArtifactRuntimeHooksWithOCIClient(client deployArtifactOCIClient) DeployArtifactRuntimeHooks {
	return DeployArtifactRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*devopsv1beta1.DeployArtifact]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*devopsv1beta1.DeployArtifact]{},
		StatusHooks:     generatedruntime.StatusHooks[*devopsv1beta1.DeployArtifact]{},
		ParityHooks:     generatedruntime.ParityHooks[*devopsv1beta1.DeployArtifact]{},
		Async:           generatedruntime.AsyncHooks[*devopsv1beta1.DeployArtifact]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*devopsv1beta1.DeployArtifact]{},
		Create: runtimeOperationHooks[devopssdk.CreateDeployArtifactRequest, devopssdk.CreateDeployArtifactResponse]{
			Fields: deployArtifactCreateFields(),
			Call: func(ctx context.Context, request devopssdk.CreateDeployArtifactRequest) (devopssdk.CreateDeployArtifactResponse, error) {
				return client.CreateDeployArtifact(ctx, request)
			},
		},
		Get: runtimeOperationHooks[devopssdk.GetDeployArtifactRequest, devopssdk.GetDeployArtifactResponse]{
			Fields: deployArtifactGetFields(),
			Call: func(ctx context.Context, request devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error) {
				return client.GetDeployArtifact(ctx, request)
			},
		},
		List: runtimeOperationHooks[devopssdk.ListDeployArtifactsRequest, devopssdk.ListDeployArtifactsResponse]{
			Fields: deployArtifactListFields(),
			Call: func(ctx context.Context, request devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error) {
				return client.ListDeployArtifacts(ctx, request)
			},
		},
		Update: runtimeOperationHooks[devopssdk.UpdateDeployArtifactRequest, devopssdk.UpdateDeployArtifactResponse]{
			Fields: deployArtifactUpdateFields(),
			Call: func(ctx context.Context, request devopssdk.UpdateDeployArtifactRequest) (devopssdk.UpdateDeployArtifactResponse, error) {
				return client.UpdateDeployArtifact(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[devopssdk.DeleteDeployArtifactRequest, devopssdk.DeleteDeployArtifactResponse]{
			Fields: deployArtifactDeleteFields(),
			Call: func(ctx context.Context, request devopssdk.DeleteDeployArtifactRequest) (devopssdk.DeleteDeployArtifactResponse, error) {
				return client.DeleteDeployArtifact(ctx, request)
			},
		},
		WrapGeneratedClient: []func(DeployArtifactServiceClient) DeployArtifactServiceClient{},
	}
}

func newDeployArtifactRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "devops",
		FormalSlug:    "deployartifact",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(devopssdk.DeployArtifactLifecycleStateCreating)},
			UpdatingStates:     []string{string(devopssdk.DeployArtifactLifecycleStateUpdating)},
			ActiveStates:       []string{string(devopssdk.DeployArtifactLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(devopssdk.DeployArtifactLifecycleStateDeleting)},
			TerminalStates: []string{string(devopssdk.DeployArtifactLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"projectId",
				"displayName",
				"deployArtifactType",
				"argumentSubstitutionMode",
				"deployArtifactSource.deployArtifactSourceType",
				"deployArtifactSource.repositoryId",
				"deployArtifactSource.deployArtifactPath",
				"deployArtifactSource.deployArtifactVersion",
				"deployArtifactSource.chartUrl",
				"deployArtifactSource.imageUri",
				"deployArtifactSource.imageDigest",
				"deployArtifactSource.base64EncodedContent",
				"deployArtifactSource.helmArtifactSourceType",
				"deployArtifactSource.helmVerificationKeySource.verificationKeySourceType",
				"deployArtifactSource.helmVerificationKeySource.currentPublicKey",
				"deployArtifactSource.helmVerificationKeySource.previousPublicKey",
				"deployArtifactSource.helmVerificationKeySource.vaultSecretId",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"deployArtifactType",
				"deployArtifactSource",
				"argumentSubstitutionMode",
				"description",
				"displayName",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"projectId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetDeployArtifact",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetDeployArtifact",
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

func deployArtifactCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDeployArtifactDetails", RequestName: "CreateDeployArtifactDetails", Contribution: "body"},
	}
}

func deployArtifactGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DeployArtifactId", RequestName: "deployArtifactId", Contribution: "path", PreferResourceID: true},
	}
}

func deployArtifactListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "query", LookupPaths: []string{"status.projectId", "spec.projectId", "projectId"}},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "compartmentId"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", LookupPaths: []string{"status.lifecycleState", "lifecycleState"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func deployArtifactUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DeployArtifactId", RequestName: "deployArtifactId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateDeployArtifactDetails", RequestName: "UpdateDeployArtifactDetails", Contribution: "body"},
	}
}

func deployArtifactDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DeployArtifactId", RequestName: "deployArtifactId", Contribution: "path", PreferResourceID: true},
	}
}

func buildDeployArtifactCreateBody(resource *devopsv1beta1.DeployArtifact) (devopssdk.CreateDeployArtifactDetails, error) {
	if resource == nil {
		return devopssdk.CreateDeployArtifactDetails{}, fmt.Errorf("%s resource is nil", deployArtifactKind)
	}
	if err := validateDeployArtifactRequiredSpec(resource.Spec); err != nil {
		return devopssdk.CreateDeployArtifactDetails{}, err
	}

	source, err := deployArtifactSourceFromSpec(resource.Spec.DeployArtifactSource)
	if err != nil {
		return devopssdk.CreateDeployArtifactDetails{}, err
	}
	body := devopssdk.CreateDeployArtifactDetails{
		DeployArtifactType:       devopssdk.DeployArtifactDeployArtifactTypeEnum(strings.TrimSpace(resource.Spec.DeployArtifactType)),
		DeployArtifactSource:     source,
		ArgumentSubstitutionMode: devopssdk.DeployArtifactArgumentSubstitutionModeEnum(strings.TrimSpace(resource.Spec.ArgumentSubstitutionMode)),
		ProjectId:                common.String(strings.TrimSpace(resource.Spec.ProjectId)),
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		body.Description = common.String(description)
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		body.DisplayName = common.String(displayName)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = deployArtifactCloneStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = deployArtifactDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildDeployArtifactUpdateBody(
	resource *devopsv1beta1.DeployArtifact,
	currentResponse any,
) (devopssdk.UpdateDeployArtifactDetails, bool, error) {
	if resource == nil {
		return devopssdk.UpdateDeployArtifactDetails{}, false, fmt.Errorf("%s resource is nil", deployArtifactKind)
	}
	if err := validateDeployArtifactRequiredSpec(resource.Spec); err != nil {
		return devopssdk.UpdateDeployArtifactDetails{}, false, err
	}

	current, ok := deployArtifactFromResponse(currentResponse)
	if !ok {
		return devopssdk.UpdateDeployArtifactDetails{}, false, fmt.Errorf("current %s response does not expose a body", deployArtifactKind)
	}
	if err := validateDeployArtifactCreateOnlyDrift(resource.Spec, current); err != nil {
		return devopssdk.UpdateDeployArtifactDetails{}, false, err
	}

	desiredSource, err := deployArtifactSourceFromSpec(resource.Spec.DeployArtifactSource)
	if err != nil {
		return devopssdk.UpdateDeployArtifactDetails{}, false, err
	}

	return deployArtifactUpdateDetailsFromCurrent(resource.Spec, current, desiredSource)
}

func deployArtifactUpdateDetailsFromCurrent(
	spec devopsv1beta1.DeployArtifactSpec,
	current devopssdk.DeployArtifact,
	desiredSource devopssdk.DeployArtifactSource,
) (devopssdk.UpdateDeployArtifactDetails, bool, error) {
	details := devopssdk.UpdateDeployArtifactDetails{}
	updateNeeded := false
	if desired := devopssdk.DeployArtifactDeployArtifactTypeEnum(strings.TrimSpace(spec.DeployArtifactType)); desired != current.DeployArtifactType {
		details.DeployArtifactType = desired
		updateNeeded = true
	}
	if !deployArtifactJSONEqual(desiredSource, current.DeployArtifactSource) {
		details.DeployArtifactSource = desiredSource
		updateNeeded = true
	}
	if desired := devopssdk.DeployArtifactArgumentSubstitutionModeEnum(strings.TrimSpace(spec.ArgumentSubstitutionMode)); desired != current.ArgumentSubstitutionMode {
		details.ArgumentSubstitutionMode = desired
		updateNeeded = true
	}
	if desired, ok := deployArtifactDesiredStringUpdate(spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := deployArtifactDesiredStringUpdate(spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := deployArtifactDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := deployArtifactDesiredDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func validateDeployArtifactRequiredSpec(spec devopsv1beta1.DeployArtifactSpec) error {
	var missing []string
	if strings.TrimSpace(spec.ProjectId) == "" {
		missing = append(missing, "projectId")
	}
	deployArtifactType := strings.TrimSpace(spec.DeployArtifactType)
	argumentSubstitutionMode := strings.TrimSpace(spec.ArgumentSubstitutionMode)
	if deployArtifactType == "" {
		missing = append(missing, "deployArtifactType")
	} else if _, ok := devopssdk.GetMappingDeployArtifactDeployArtifactTypeEnum(deployArtifactType); !ok {
		return fmt.Errorf("%s spec.deployArtifactType %q is not supported", deployArtifactKind, spec.DeployArtifactType)
	}
	if argumentSubstitutionMode == "" {
		missing = append(missing, "argumentSubstitutionMode")
	} else if _, ok := devopssdk.GetMappingDeployArtifactArgumentSubstitutionModeEnum(argumentSubstitutionMode); !ok {
		return fmt.Errorf("%s spec.argumentSubstitutionMode %q is not supported", deployArtifactKind, spec.ArgumentSubstitutionMode)
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s spec is missing required field(s): %s", deployArtifactKind, strings.Join(missing, ", "))
	}
	return nil
}

func deployArtifactSourceFromSpec(spec devopsv1beta1.DeployArtifactSource) (devopssdk.DeployArtifactSource, error) {
	payload, err := deployArtifactSourcePayload(spec)
	if err != nil {
		return nil, err
	}
	return deployArtifactSourceFromPayload(payload)
}

func deployArtifactSourcePayload(spec devopsv1beta1.DeployArtifactSource) (map[string]any, error) {
	payload, err := deployArtifactJSONDataPayload(spec.JsonData, "spec.deployArtifactSource.jsonData")
	if err != nil {
		return nil, err
	}
	setStringPayload(payload, "deployArtifactSourceType", normalizeDeployArtifactSourceType(spec.DeployArtifactSourceType))
	setStringPayload(payload, "repositoryId", spec.RepositoryId)
	setStringPayload(payload, "deployArtifactPath", spec.DeployArtifactPath)
	setStringPayload(payload, "deployArtifactVersion", spec.DeployArtifactVersion)
	setStringPayload(payload, "chartUrl", spec.ChartUrl)
	setStringPayload(payload, "imageUri", spec.ImageUri)
	setStringPayload(payload, "imageDigest", spec.ImageDigest)
	setStringPayload(payload, "base64EncodedContent", spec.Base64EncodedContent)
	setStringPayload(payload, "helmArtifactSourceType", normalizeHelmArtifactSourceType(spec.HelmArtifactSourceType))
	if verificationPayload, ok, err := deployArtifactVerificationKeySourcePayload(spec.HelmVerificationKeySource); err != nil {
		return nil, err
	} else if ok {
		payload["helmVerificationKeySource"] = verificationPayload
	}
	if sourceType := deployArtifactSourceTypeFromPayload(payload); sourceType != "" {
		payload["deployArtifactSourceType"] = sourceType
	} else {
		return nil, fmt.Errorf("%s spec.deployArtifactSource.deployArtifactSourceType is required", deployArtifactKind)
	}
	if err := validateDeployArtifactSourcePayload(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func deployArtifactSourceFromPayload(payload map[string]any) (devopssdk.DeployArtifactSource, error) {
	sourceType := deployArtifactSourceTypeFromPayload(payload)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	switch sourceType {
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeGenericArtifact):
		var source devopssdk.GenericDeployArtifactSource
		err = json.Unmarshal(body, &source)
		return source, err
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeOcir):
		var source devopssdk.OcirDeployArtifactSource
		err = json.Unmarshal(body, &source)
		return source, err
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeInline):
		var source devopssdk.InlineDeployArtifactSource
		err = json.Unmarshal(body, &source)
		return source, err
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeHelmChart):
		var source devopssdk.HelmRepositoryDeployArtifactSource
		err = json.Unmarshal(body, &source)
		return source, err
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeHelmCommandSpec):
		var source devopssdk.HelmCommandSpecArtifactSource
		err = json.Unmarshal(body, &source)
		return source, err
	default:
		return nil, fmt.Errorf("%s deployArtifactSourceType %q is not supported", deployArtifactKind, sourceType)
	}
}

func deployArtifactVerificationKeySourcePayload(
	spec devopsv1beta1.DeployArtifactSourceHelmVerificationKeySource,
) (map[string]any, bool, error) {
	payload, err := deployArtifactJSONDataPayload(spec.JsonData, "spec.deployArtifactSource.helmVerificationKeySource.jsonData")
	if err != nil {
		return nil, false, err
	}
	setStringPayload(payload, "verificationKeySourceType", normalizeVerificationKeySourceType(spec.VerificationKeySourceType))
	setStringPayload(payload, "currentPublicKey", spec.CurrentPublicKey)
	setStringPayload(payload, "previousPublicKey", spec.PreviousPublicKey)
	setStringPayload(payload, "vaultSecretId", spec.VaultSecretId)
	if len(payload) == 0 {
		return nil, false, nil
	}
	sourceType := verificationKeySourceTypeFromPayload(payload)
	if sourceType == "" {
		return nil, false, fmt.Errorf("%s helmVerificationKeySource.verificationKeySourceType is required", deployArtifactKind)
	}
	payload["verificationKeySourceType"] = sourceType
	if err := validateVerificationKeySourcePayload(payload); err != nil {
		return nil, false, err
	}
	return payload, true, nil
}

func validateDeployArtifactSourcePayload(payload map[string]any) error {
	sourceType := deployArtifactSourceTypeFromPayload(payload)
	required := []string{"deployArtifactSourceType"}
	switch sourceType {
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeGenericArtifact):
		required = append(required, "repositoryId", "deployArtifactPath", "deployArtifactVersion")
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeOcir):
		required = append(required, "imageUri")
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeInline):
		required = append(required, "base64EncodedContent")
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeHelmChart):
		required = append(required, "chartUrl", "deployArtifactVersion")
	case string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeHelmCommandSpec):
		required = append(required, "base64EncodedContent", "helmArtifactSourceType")
	default:
		return fmt.Errorf("%s deployArtifactSourceType %q is not supported", deployArtifactKind, sourceType)
	}
	return requirePayloadStrings("spec.deployArtifactSource", payload, required...)
}

func validateVerificationKeySourcePayload(payload map[string]any) error {
	sourceType := verificationKeySourceTypeFromPayload(payload)
	switch sourceType {
	case string(devopssdk.VerificationKeySourceVerificationKeySourceTypeInlinePublicKey):
		return requirePayloadStrings("spec.deployArtifactSource.helmVerificationKeySource", payload, "verificationKeySourceType", "currentPublicKey")
	case string(devopssdk.VerificationKeySourceVerificationKeySourceTypeVaultSecret):
		return requirePayloadStrings("spec.deployArtifactSource.helmVerificationKeySource", payload, "verificationKeySourceType", "vaultSecretId")
	case string(devopssdk.VerificationKeySourceVerificationKeySourceTypeNone):
		return requirePayloadStrings("spec.deployArtifactSource.helmVerificationKeySource", payload, "verificationKeySourceType")
	default:
		return fmt.Errorf("%s helmVerificationKeySource.verificationKeySourceType %q is not supported", deployArtifactKind, sourceType)
	}
}

func requirePayloadStrings(prefix string, payload map[string]any, fields ...string) error {
	var missing []string
	for _, field := range fields {
		if strings.TrimSpace(stringFromPayload(payload, field)) == "" {
			missing = append(missing, prefix+"."+field)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", deployArtifactKind, strings.Join(missing, ", "))
}

func deployArtifactJSONDataPayload(raw string, field string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("parse %s: %w", field, err)
	}
	if payload == nil {
		return nil, fmt.Errorf("%s must be a JSON object", field)
	}
	return payload, nil
}

func deployArtifactSourceTypeFromPayload(payload map[string]any) string {
	if sourceType := normalizeDeployArtifactSourceType(stringFromPayload(payload, "deployArtifactSourceType")); sourceType != "" {
		return sourceType
	}
	switch {
	case strings.TrimSpace(stringFromPayload(payload, "repositoryId")) != "":
		return string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeGenericArtifact)
	case strings.TrimSpace(stringFromPayload(payload, "imageUri")) != "":
		return string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeOcir)
	case strings.TrimSpace(stringFromPayload(payload, "chartUrl")) != "":
		return string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeHelmChart)
	case strings.TrimSpace(stringFromPayload(payload, "helmArtifactSourceType")) != "":
		return string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeHelmCommandSpec)
	case strings.TrimSpace(stringFromPayload(payload, "base64EncodedContent")) != "":
		return string(devopssdk.DeployArtifactSourceDeployArtifactSourceTypeInline)
	default:
		return ""
	}
}

func verificationKeySourceTypeFromPayload(payload map[string]any) string {
	return normalizeVerificationKeySourceType(stringFromPayload(payload, "verificationKeySourceType"))
}

func normalizeDeployArtifactSourceType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if enum, ok := devopssdk.GetMappingDeployArtifactSourceDeployArtifactSourceTypeEnum(value); ok {
		return string(enum)
	}
	return strings.ToUpper(value)
}

func normalizeVerificationKeySourceType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if enum, ok := devopssdk.GetMappingVerificationKeySourceVerificationKeySourceTypeEnum(value); ok {
		return string(enum)
	}
	return strings.ToUpper(value)
}

func normalizeHelmArtifactSourceType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if enum, ok := devopssdk.GetMappingHelmCommandSpecArtifactSourceHelmArtifactSourceTypeEnum(value); ok {
		return string(enum)
	}
	return strings.ToUpper(value)
}

func setStringPayload(payload map[string]any, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		payload[key] = value
	}
}

func stringFromPayload(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func validateDeployArtifactCreateOnlyDriftForResponse(resource *devopsv1beta1.DeployArtifact, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", deployArtifactKind)
	}
	current, ok := deployArtifactFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current %s response does not expose a body", deployArtifactKind)
	}
	return validateDeployArtifactCreateOnlyDrift(resource.Spec, current)
}

func validateDeployArtifactCreateOnlyDrift(
	spec devopsv1beta1.DeployArtifactSpec,
	current devopssdk.DeployArtifact,
) error {
	if desired := strings.TrimSpace(spec.ProjectId); desired != "" && desired != deployArtifactString(current.ProjectId) {
		return fmt.Errorf("%s create-only field projectId requires replacement when changed", deployArtifactKind)
	}
	return nil
}

func deployArtifactStatusFromResponse(resource *devopsv1beta1.DeployArtifact, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", deployArtifactKind)
	}
	current, ok := deployArtifactFromResponse(response)
	if !ok {
		return nil
	}
	status := resource.Status.OsokStatus
	resource.Status = devopsv1beta1.DeployArtifactStatus{
		OsokStatus:               status,
		Id:                       deployArtifactString(current.Id),
		ProjectId:                deployArtifactString(current.ProjectId),
		CompartmentId:            deployArtifactString(current.CompartmentId),
		DeployArtifactType:       string(current.DeployArtifactType),
		ArgumentSubstitutionMode: string(current.ArgumentSubstitutionMode),
		DeployArtifactSource:     deployArtifactSourceStatus(current.DeployArtifactSource),
		Description:              deployArtifactString(current.Description),
		DisplayName:              deployArtifactString(current.DisplayName),
		TimeCreated:              deployArtifactSDKTimeString(current.TimeCreated),
		TimeUpdated:              deployArtifactSDKTimeString(current.TimeUpdated),
		LifecycleState:           string(current.LifecycleState),
		LifecycleDetails:         deployArtifactString(current.LifecycleDetails),
		FreeformTags:             deployArtifactCloneStringMap(current.FreeformTags),
		DefinedTags:              deployArtifactStatusDefinedTags(current.DefinedTags),
		SystemTags:               deployArtifactStatusDefinedTags(current.SystemTags),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func clearTrackedDeployArtifactIdentity(resource *devopsv1beta1.DeployArtifact) {
	if resource == nil {
		return
	}
	resource.Status = devopsv1beta1.DeployArtifactStatus{}
}

func deployArtifactSourceStatus(source devopssdk.DeployArtifactSource) devopsv1beta1.DeployArtifactSource {
	if source == nil {
		return devopsv1beta1.DeployArtifactSource{}
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return devopsv1beta1.DeployArtifactSource{}
	}
	var status devopsv1beta1.DeployArtifactSource
	if err := json.Unmarshal(payload, &status); err != nil {
		return devopsv1beta1.DeployArtifactSource{}
	}
	return status
}

func deployArtifactFromResponse(response any) (devopssdk.DeployArtifact, bool) {
	if current, ok := deployArtifactFromResponseValue(response); ok {
		return current, true
	}
	return deployArtifactFromResponsePointer(response)
}

func deployArtifactFromResponseValue(response any) (devopssdk.DeployArtifact, bool) {
	switch current := response.(type) {
	case devopssdk.CreateDeployArtifactResponse:
		return current.DeployArtifact, true
	case devopssdk.GetDeployArtifactResponse:
		return current.DeployArtifact, true
	case devopssdk.UpdateDeployArtifactResponse:
		return current.DeployArtifact, true
	case devopssdk.DeployArtifact:
		return current, true
	case devopssdk.DeployArtifactSummary:
		return deployArtifactFromSummary(current), true
	default:
		return devopssdk.DeployArtifact{}, false
	}
}

func deployArtifactFromResponsePointer(response any) (devopssdk.DeployArtifact, bool) {
	switch current := response.(type) {
	case *devopssdk.CreateDeployArtifactResponse:
		return deployArtifactFromCreateResponsePointer(current)
	case *devopssdk.GetDeployArtifactResponse:
		return deployArtifactFromGetResponsePointer(current)
	case *devopssdk.UpdateDeployArtifactResponse:
		return deployArtifactFromUpdateResponsePointer(current)
	case *devopssdk.DeployArtifact:
		return deployArtifactFromBodyPointer(current)
	case *devopssdk.DeployArtifactSummary:
		return deployArtifactFromSummaryPointer(current)
	default:
		return devopssdk.DeployArtifact{}, false
	}
}

func deployArtifactFromCreateResponsePointer(response *devopssdk.CreateDeployArtifactResponse) (devopssdk.DeployArtifact, bool) {
	if response == nil {
		return devopssdk.DeployArtifact{}, false
	}
	return response.DeployArtifact, true
}

func deployArtifactFromGetResponsePointer(response *devopssdk.GetDeployArtifactResponse) (devopssdk.DeployArtifact, bool) {
	if response == nil {
		return devopssdk.DeployArtifact{}, false
	}
	return response.DeployArtifact, true
}

func deployArtifactFromUpdateResponsePointer(response *devopssdk.UpdateDeployArtifactResponse) (devopssdk.DeployArtifact, bool) {
	if response == nil {
		return devopssdk.DeployArtifact{}, false
	}
	return response.DeployArtifact, true
}

func deployArtifactFromBodyPointer(response *devopssdk.DeployArtifact) (devopssdk.DeployArtifact, bool) {
	if response == nil {
		return devopssdk.DeployArtifact{}, false
	}
	return *response, true
}

func deployArtifactFromSummaryPointer(response *devopssdk.DeployArtifactSummary) (devopssdk.DeployArtifact, bool) {
	if response == nil {
		return devopssdk.DeployArtifact{}, false
	}
	return deployArtifactFromSummary(*response), true
}

func deployArtifactFromSummary(summary devopssdk.DeployArtifactSummary) devopssdk.DeployArtifact {
	return devopssdk.DeployArtifact{
		Id:                       summary.Id,
		ProjectId:                summary.ProjectId,
		CompartmentId:            summary.CompartmentId,
		DeployArtifactType:       summary.DeployArtifactType,
		DeployArtifactSource:     summary.DeployArtifactSource,
		ArgumentSubstitutionMode: summary.ArgumentSubstitutionMode,
		Description:              summary.Description,
		DisplayName:              summary.DisplayName,
		TimeCreated:              summary.TimeCreated,
		TimeUpdated:              summary.TimeUpdated,
		LifecycleState:           summary.LifecycleState,
		LifecycleDetails:         summary.LifecycleDetails,
		FreeformTags:             deployArtifactCloneStringMap(summary.FreeformTags),
		DefinedTags:              deployArtifactCloneDefinedTags(summary.DefinedTags),
		SystemTags:               deployArtifactCloneDefinedTags(summary.SystemTags),
	}
}

func listDeployArtifactsAllPages(
	ctx context.Context,
	client deployArtifactOCIClient,
	initErr error,
	request devopssdk.ListDeployArtifactsRequest,
) (devopssdk.ListDeployArtifactsResponse, error) {
	if initErr != nil {
		return devopssdk.ListDeployArtifactsResponse{}, fmt.Errorf("initialize %s OCI client: %w", deployArtifactKind, initErr)
	}
	if client == nil {
		return devopssdk.ListDeployArtifactsResponse{}, fmt.Errorf("%s OCI client is not configured", deployArtifactKind)
	}

	var combined devopssdk.ListDeployArtifactsResponse
	for {
		response, err := client.ListDeployArtifacts(ctx, request)
		if err != nil {
			return devopssdk.ListDeployArtifactsResponse{}, conservativeDeployArtifactNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		for _, item := range response.Items {
			if item.LifecycleState == devopssdk.DeployArtifactLifecycleStateDeleted {
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

func deployArtifactDeleteConfirmRead(
	getDeployArtifact func(context.Context, devopssdk.GetDeployArtifactRequest) (devopssdk.GetDeployArtifactResponse, error),
	listDeployArtifacts func(context.Context, devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error),
) func(context.Context, *devopsv1beta1.DeployArtifact, string) (any, error) {
	return func(ctx context.Context, resource *devopsv1beta1.DeployArtifact, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID == "" {
			return deployArtifactDeleteConfirmReadByList(ctx, resource, listDeployArtifacts)
		}
		if getDeployArtifact == nil {
			return nil, fmt.Errorf("%s delete confirmation requires a readable OCI operation", deployArtifactKind)
		}
		response, err := getDeployArtifact(ctx, devopssdk.GetDeployArtifactRequest{
			DeployArtifactId: common.String(currentID),
		})
		if err == nil {
			return response, nil
		}
		if !isDeployArtifactAmbiguousNotFound(err) && !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return nil, err
		}
		handledErr := handleDeployArtifactDeleteError(resource, err)
		if handledErr == nil {
			handledErr = err
		}
		return deployArtifactAmbiguousDeleteConfirmResponse{
			DeployArtifact: devopssdk.DeployArtifact{
				Id:             common.String(currentID),
				LifecycleState: devopssdk.DeployArtifactLifecycleStateActive,
			},
			err: handledErr,
		}, nil
	}
}

func deployArtifactDeleteConfirmReadByList(
	ctx context.Context,
	resource *devopsv1beta1.DeployArtifact,
	listDeployArtifacts func(context.Context, devopssdk.ListDeployArtifactsRequest) (devopssdk.ListDeployArtifactsResponse, error),
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", deployArtifactKind)
	}
	if listDeployArtifacts == nil {
		return nil, fmt.Errorf("%s delete confirmation requires a tracked OCI identifier", deployArtifactKind)
	}
	response, err := listDeployArtifacts(ctx, devopssdk.ListDeployArtifactsRequest{
		ProjectId:   common.String(strings.TrimSpace(resource.Spec.ProjectId)),
		DisplayName: deployArtifactOptionalString(resource.Spec.DisplayName),
	})
	if err != nil {
		return nil, err
	}
	matches := deployArtifactMatchingSummaries(resource, response.Items)
	switch len(matches) {
	case 0:
		return nil, errorutil.NotFoundOciError(errorutil.OciErrors{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			OpcRequestID:   deployArtifactString(response.OpcRequestId),
			Description:    "deployment artifact not found during delete confirmation",
		})
	case 1:
		return devopssdk.GetDeployArtifactResponse{
			DeployArtifact: deployArtifactFromSummary(matches[0]),
			OpcRequestId:   response.OpcRequestId,
		}, nil
	default:
		return nil, fmt.Errorf("%s delete confirmation found %d matching deployment artifacts", deployArtifactKind, len(matches))
	}
}

func deployArtifactMatchingSummaries(
	resource *devopsv1beta1.DeployArtifact,
	items []devopssdk.DeployArtifactSummary,
) []devopssdk.DeployArtifactSummary {
	if resource == nil {
		return nil
	}
	desiredSource, err := deployArtifactSourceFromSpec(resource.Spec.DeployArtifactSource)
	if err != nil {
		return nil
	}
	var matches []devopssdk.DeployArtifactSummary
	for _, item := range items {
		if deployArtifactSummaryMatches(resource, desiredSource, item) {
			matches = append(matches, item)
		}
	}
	return matches
}

func deployArtifactSummaryMatches(
	resource *devopsv1beta1.DeployArtifact,
	desiredSource devopssdk.DeployArtifactSource,
	item devopssdk.DeployArtifactSummary,
) bool {
	if item.LifecycleState == devopssdk.DeployArtifactLifecycleStateDeleted {
		return false
	}
	if deployArtifactString(item.ProjectId) != strings.TrimSpace(resource.Spec.ProjectId) {
		return false
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" && deployArtifactString(item.DisplayName) != displayName {
		return false
	}
	if string(item.DeployArtifactType) != strings.TrimSpace(resource.Spec.DeployArtifactType) {
		return false
	}
	if string(item.ArgumentSubstitutionMode) != strings.TrimSpace(resource.Spec.ArgumentSubstitutionMode) {
		return false
	}
	return deployArtifactJSONEqual(desiredSource, item.DeployArtifactSource)
}

func applyDeployArtifactDeleteOutcome(
	_ *devopsv1beta1.DeployArtifact,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if err, ok := deployArtifactAmbiguousDeleteConfirmError(response); ok {
		if err != nil {
			return generatedruntime.DeleteOutcome{Handled: true}, err
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func deployArtifactAmbiguousDeleteConfirmError(response any) (error, bool) {
	switch typed := response.(type) {
	case deployArtifactAmbiguousDeleteConfirmResponse:
		return typed.err, true
	case *deployArtifactAmbiguousDeleteConfirmResponse:
		if typed == nil {
			return nil, false
		}
		return typed.err, true
	default:
		return nil, false
	}
}

func handleDeployArtifactDeleteError(resource *devopsv1beta1.DeployArtifact, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if isDeployArtifactAmbiguousNotFound(err) {
		return err
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return deployArtifactAmbiguousNotFoundError{
		HTTPStatusCode: classification.HTTPStatusCode,
		ErrorCode:      deployArtifactAmbiguousNotFoundErrorCode,
		OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
		message: fmt.Sprintf(
			"%s delete returned ambiguous not-found response (HTTP %s, code %s); retaining finalizer until OCI deletion is confirmed",
			deployArtifactKind,
			classification.HTTPStatusCodeString(),
			classification.ErrorCodeString(),
		),
	}
}

func conservativeDeployArtifactNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return deployArtifactAmbiguousNotFoundError{
		HTTPStatusCode: classification.HTTPStatusCode,
		ErrorCode:      deployArtifactAmbiguousNotFoundErrorCode,
		OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
		message: fmt.Sprintf(
			"%s %s returned ambiguous not-found response (HTTP %s, code %s)",
			deployArtifactKind,
			strings.TrimSpace(operation),
			classification.HTTPStatusCodeString(),
			classification.ErrorCodeString(),
		),
	}
}

func isDeployArtifactAmbiguousNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous deployArtifactAmbiguousNotFoundError
	return errorAs(err, &ambiguous)
}

func getDeployArtifactWorkRequest(
	ctx context.Context,
	client deployArtifactOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", deployArtifactKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", deployArtifactKind)
	}
	response, err := client.GetWorkRequest(ctx, devopssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveDeployArtifactWorkRequestAction(workRequest any) (string, error) {
	current, err := deployArtifactWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveDeployArtifactWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := deployArtifactWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case devopssdk.OperationTypeCreateDeployArtifact:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case devopssdk.OperationTypeUpdateDeployArtifact:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case devopssdk.OperationTypeDeleteDeployArtifact:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverDeployArtifactIDFromWorkRequest(
	_ *devopsv1beta1.DeployArtifact,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := deployArtifactWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if id, ok := resolveDeployArtifactIDFromWorkRequestResources(current.Resources, phase, true); ok {
		return id, nil
	}
	if id, ok := resolveDeployArtifactIDFromWorkRequestResources(current.Resources, phase, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose a deployment artifact identifier", deployArtifactKind, deployArtifactString(current.Id))
}

func resolveDeployArtifactIDFromWorkRequestResources(
	resources []devopssdk.WorkRequestResource,
	phase shared.OSOKAsyncPhase,
	requireActionMatch bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if !isDeployArtifactWorkRequestResource(resource) {
			continue
		}
		if requireActionMatch && !deployArtifactWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		id := strings.TrimSpace(deployArtifactString(resource.Identifier))
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

func isDeployArtifactWorkRequestResource(resource devopssdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(deployArtifactString(resource.EntityType)))
	if strings.Contains(entityType, "deploy") && strings.Contains(entityType, "artifact") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(deployArtifactString(resource.EntityUri)))
	return strings.Contains(entityURI, "/deployartifacts/")
}

func deployArtifactWorkRequestActionMatchesPhase(action devopssdk.ActionTypeEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == devopssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return action == devopssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return action == devopssdk.ActionTypeDeleted
	default:
		return false
	}
}

func deployArtifactWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := deployArtifactWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", deployArtifactKind, phase, deployArtifactString(current.Id), current.Status)
}

func deployArtifactWorkRequestFromAny(workRequest any) (devopssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case devopssdk.WorkRequest:
		return current, nil
	case *devopssdk.WorkRequest:
		if current == nil {
			return devopssdk.WorkRequest{}, fmt.Errorf("%s work request is nil", deployArtifactKind)
		}
		return *current, nil
	default:
		return devopssdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", deployArtifactKind, workRequest)
	}
}

func deployArtifactDesiredStringUpdate(spec string, current *string) (*string, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == deployArtifactString(current) {
		return nil, false
	}
	return common.String(spec), true
}

func deployArtifactDesiredFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if reflect.DeepEqual(spec, current) {
		return nil, false
	}
	return deployArtifactCloneStringMap(spec), true
}

func deployArtifactDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := deployArtifactDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if deployArtifactJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func deployArtifactDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		converted[namespace] = inner
	}
	return converted
}

func deployArtifactStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for key, values := range input {
		if values == nil {
			converted[key] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for innerKey, innerValue := range values {
			tagValues[innerKey] = fmt.Sprint(innerValue)
		}
		converted[key] = tagValues
	}
	return converted
}

func deployArtifactCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func deployArtifactCloneDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
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

func deployArtifactOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func deployArtifactString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func deployArtifactSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func deployArtifactJSONEqual(left any, right any) bool {
	leftValue, leftErr := normalizedJSONValue(left)
	rightValue, rightErr := normalizedJSONValue(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func normalizedJSONValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func errorAs(err error, target any) bool {
	if err == nil || target == nil {
		return false
	}
	switch typed := err.(type) {
	case deployArtifactAmbiguousNotFoundError:
		if out, ok := target.(*deployArtifactAmbiguousNotFoundError); ok {
			*out = typed
			return true
		}
	case *deployArtifactAmbiguousNotFoundError:
		if typed == nil {
			return false
		}
		if out, ok := target.(*deployArtifactAmbiguousNotFoundError); ok {
			*out = *typed
			return true
		}
	}
	return false
}
