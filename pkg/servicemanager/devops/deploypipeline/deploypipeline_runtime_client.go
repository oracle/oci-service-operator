/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package deploypipeline

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

const deployPipelineKind = "DeployPipeline"

var deployPipelineWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
	CreateActionTokens:    []string{string(devopssdk.OperationTypeCreateDeployPipeline)},
	UpdateActionTokens:    []string{string(devopssdk.OperationTypeUpdateDeployPipeline)},
	DeleteActionTokens:    []string{string(devopssdk.OperationTypeDeleteDeployPipeline)},
}

type deployPipelineOCIClient interface {
	CreateDeployPipeline(context.Context, devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error)
	GetDeployPipeline(context.Context, devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error)
	ListDeployPipelines(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error)
	UpdateDeployPipeline(context.Context, devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error)
	DeleteDeployPipeline(context.Context, devopssdk.DeleteDeployPipelineRequest) (devopssdk.DeleteDeployPipelineResponse, error)
	GetWorkRequest(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

type ambiguousDeployPipelineNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousDeployPipelineNotFoundError) Error() string {
	return e.message
}

func (e ambiguousDeployPipelineNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

type deployPipelineAmbiguousDeleteConfirmResponse struct {
	DeployPipeline devopssdk.DeployPipeline `presentIn:"body"`
	err            error
}

type deployPipelineDeleteWithoutTrackedIDClient struct {
	DeployPipelineServiceClient
	listDeployPipelines func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error)
}

func init() {
	registerDeployPipelineRuntimeHooksMutator(func(manager *DeployPipelineServiceManager, hooks *DeployPipelineRuntimeHooks) {
		client, initErr := newDeployPipelineSDKClient(manager)
		applyDeployPipelineRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newDeployPipelineSDKClient(manager *DeployPipelineServiceManager) (deployPipelineOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", deployPipelineKind)
	}
	client, err := devopssdk.NewDevopsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyDeployPipelineRuntimeHooks(
	_ *DeployPipelineServiceManager,
	hooks *DeployPipelineRuntimeHooks,
	client deployPipelineOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedDeployPipelineRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *devopsv1beta1.DeployPipeline, _ string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("%s resource is nil", deployPipelineKind)
		}
		return buildDeployPipelineCreateBody(resource.Spec)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *devopsv1beta1.DeployPipeline,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDeployPipelineUpdateBody(resource, currentResponse)
	}
	hooks.Get.Call = func(ctx context.Context, request devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
		if initErr != nil {
			return devopssdk.GetDeployPipelineResponse{}, fmt.Errorf("initialize %s OCI client: %w", deployPipelineKind, initErr)
		}
		if client == nil {
			return devopssdk.GetDeployPipelineResponse{}, fmt.Errorf("%s OCI client is not configured", deployPipelineKind)
		}
		response, err := client.GetDeployPipeline(ctx, request)
		return response, conservativeDeployPipelineNotFoundError(err, "read")
	}
	hooks.List.Fields = deployPipelineListFields()
	hooks.List.Call = func(ctx context.Context, request devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error) {
		return listDeployPipelinesAllPages(ctx, client, initErr, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request devopssdk.DeleteDeployPipelineRequest) (devopssdk.DeleteDeployPipelineResponse, error) {
		if initErr != nil {
			return devopssdk.DeleteDeployPipelineResponse{}, fmt.Errorf("initialize %s OCI client: %w", deployPipelineKind, initErr)
		}
		if client == nil {
			return devopssdk.DeleteDeployPipelineResponse{}, fmt.Errorf("%s OCI client is not configured", deployPipelineKind)
		}
		response, err := client.DeleteDeployPipeline(ctx, request)
		return response, conservativeDeployPipelineNotFoundError(err, "delete")
	}
	hooks.Identity.GuardExistingBeforeCreate = func(_ context.Context, resource *devopsv1beta1.DeployPipeline) (generatedruntime.ExistingBeforeCreateDecision, error) {
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedDeployPipelineIdentity
	hooks.StatusHooks.ProjectStatus = deployPipelineStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDeployPipelineCreateOnlyDriftForResponse
	hooks.DeleteHooks.ConfirmRead = deployPipelineDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleDeployPipelineDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDeployPipelineDeleteOutcome
	hooks.Async.Adapter = deployPipelineWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getDeployPipelineWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveDeployPipelineWorkRequestAction
	hooks.Async.ResolvePhase = resolveDeployPipelineWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverDeployPipelineIDFromWorkRequest
	hooks.Async.Message = deployPipelineWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapDeployPipelineDeleteWithoutTrackedID(hooks.List.Call))
}

func newDeployPipelineServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client deployPipelineOCIClient,
) DeployPipelineServiceClient {
	manager := &DeployPipelineServiceManager{Log: log}
	hooks := newDeployPipelineRuntimeHooksWithOCIClient(client)
	applyDeployPipelineRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultDeployPipelineServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*devopsv1beta1.DeployPipeline](
			buildDeployPipelineGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDeployPipelineGeneratedClient(hooks, delegate)
}

func newDeployPipelineRuntimeHooksWithOCIClient(client deployPipelineOCIClient) DeployPipelineRuntimeHooks {
	return DeployPipelineRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*devopsv1beta1.DeployPipeline]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*devopsv1beta1.DeployPipeline]{},
		StatusHooks:     generatedruntime.StatusHooks[*devopsv1beta1.DeployPipeline]{},
		ParityHooks:     generatedruntime.ParityHooks[*devopsv1beta1.DeployPipeline]{},
		Async:           generatedruntime.AsyncHooks[*devopsv1beta1.DeployPipeline]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*devopsv1beta1.DeployPipeline]{},
		Create: runtimeOperationHooks[devopssdk.CreateDeployPipelineRequest, devopssdk.CreateDeployPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateDeployPipelineDetails", RequestName: "CreateDeployPipelineDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request devopssdk.CreateDeployPipelineRequest) (devopssdk.CreateDeployPipelineResponse, error) {
				return client.CreateDeployPipeline(ctx, request)
			},
		},
		Get: runtimeOperationHooks[devopssdk.GetDeployPipelineRequest, devopssdk.GetDeployPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DeployPipelineId", RequestName: "deployPipelineId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error) {
				return client.GetDeployPipeline(ctx, request)
			},
		},
		List: runtimeOperationHooks[devopssdk.ListDeployPipelinesRequest, devopssdk.ListDeployPipelinesResponse]{
			Fields: deployPipelineListFields(),
			Call: func(ctx context.Context, request devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error) {
				return client.ListDeployPipelines(ctx, request)
			},
		},
		Update: runtimeOperationHooks[devopssdk.UpdateDeployPipelineRequest, devopssdk.UpdateDeployPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DeployPipelineId", RequestName: "deployPipelineId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateDeployPipelineDetails", RequestName: "UpdateDeployPipelineDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request devopssdk.UpdateDeployPipelineRequest) (devopssdk.UpdateDeployPipelineResponse, error) {
				return client.UpdateDeployPipeline(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[devopssdk.DeleteDeployPipelineRequest, devopssdk.DeleteDeployPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "DeployPipelineId", RequestName: "deployPipelineId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request devopssdk.DeleteDeployPipelineRequest) (devopssdk.DeleteDeployPipelineResponse, error) {
				return client.DeleteDeployPipeline(ctx, request)
			},
		},
		WrapGeneratedClient: []func(DeployPipelineServiceClient) DeployPipelineServiceClient{},
	}
}

func reviewedDeployPipelineRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "devops",
		FormalSlug:    "deploypipeline",
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
			ProvisioningStates: []string{string(devopssdk.DeployPipelineLifecycleStateCreating)},
			UpdatingStates:     []string{string(devopssdk.DeployPipelineLifecycleStateUpdating)},
			ActiveStates: []string{
				string(devopssdk.DeployPipelineLifecycleStateActive),
				string(devopssdk.DeployPipelineLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(devopssdk.DeployPipelineLifecycleStateDeleting)},
			TerminalStates: []string{string(devopssdk.DeployPipelineLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"projectId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"description",
				"displayName",
				"deployPipelineParameters",
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
			Strategy: "GetWorkRequest -> GetDeployPipeline",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetDeployPipeline",
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

func deployPipelineListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "query", LookupPaths: []string{"status.projectId", "spec.projectId", "projectId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
	}
}

func buildDeployPipelineCreateBody(spec devopsv1beta1.DeployPipelineSpec) (devopssdk.CreateDeployPipelineDetails, error) {
	if err := validateDeployPipelineSpec(spec); err != nil {
		return devopssdk.CreateDeployPipelineDetails{}, err
	}

	body := devopssdk.CreateDeployPipelineDetails{
		ProjectId: common.String(strings.TrimSpace(spec.ProjectId)),
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	if strings.TrimSpace(spec.DisplayName) != "" {
		body.DisplayName = common.String(spec.DisplayName)
	}
	if parameters := deployPipelineParameterCollectionForCreate(spec.DeployPipelineParameters); parameters != nil {
		body.DeployPipelineParameters = parameters
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneDeployPipelineStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	return body, nil
}

func buildDeployPipelineUpdateBody(
	resource *devopsv1beta1.DeployPipeline,
	currentResponse any,
) (devopssdk.UpdateDeployPipelineDetails, bool, error) {
	if resource == nil {
		return devopssdk.UpdateDeployPipelineDetails{}, false, fmt.Errorf("%s resource is nil", deployPipelineKind)
	}
	if err := validateDeployPipelineSpec(resource.Spec); err != nil {
		return devopssdk.UpdateDeployPipelineDetails{}, false, err
	}

	current, ok := deployPipelineFromResponse(currentResponse)
	if !ok {
		return devopssdk.UpdateDeployPipelineDetails{}, false, fmt.Errorf("current %s response does not expose a %s body", deployPipelineKind, deployPipelineKind)
	}
	if err := validateDeployPipelineCreateOnlyDrift(resource.Spec, current); err != nil {
		return devopssdk.UpdateDeployPipelineDetails{}, false, err
	}

	updateDetails, updateNeeded := deployPipelineUpdateDetails(resource.Spec, current)
	if !updateNeeded {
		return devopssdk.UpdateDeployPipelineDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func deployPipelineUpdateDetails(
	spec devopsv1beta1.DeployPipelineSpec,
	current devopssdk.DeployPipeline,
) (devopssdk.UpdateDeployPipelineDetails, bool) {
	updateDetails := devopssdk.UpdateDeployPipelineDetails{}
	updateNeeded := false

	if !deployPipelineStringPtrEqual(current.Description, spec.Description) {
		updateDetails.Description = common.String(spec.Description)
		updateNeeded = true
	}
	if !deployPipelineStringPtrEqual(current.DisplayName, spec.DisplayName) {
		updateDetails.DisplayName = common.String(spec.DisplayName)
		updateNeeded = true
	}
	if !deployPipelineParameterCollectionsEqual(current.DeployPipelineParameters, spec.DeployPipelineParameters) {
		updateDetails.DeployPipelineParameters = deployPipelineParameterCollectionForUpdate(spec.DeployPipelineParameters)
		updateNeeded = true
	}

	desiredFreeformTags := desiredDeployPipelineFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredDeployPipelineDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	return updateDetails, updateNeeded
}

func validateDeployPipelineSpec(spec devopsv1beta1.DeployPipelineSpec) error {
	var missing []string
	if strings.TrimSpace(spec.ProjectId) == "" {
		missing = append(missing, "projectId")
	}
	for i, item := range spec.DeployPipelineParameters.Items {
		if strings.TrimSpace(item.Name) == "" {
			missing = append(missing, fmt.Sprintf("deployPipelineParameters.items[%d].name", i))
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", deployPipelineKind, strings.Join(missing, ", "))
}

func validateDeployPipelineCreateOnlyDriftForResponse(resource *devopsv1beta1.DeployPipeline, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", deployPipelineKind)
	}
	current, ok := deployPipelineFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current %s response does not expose a %s body", deployPipelineKind, deployPipelineKind)
	}
	return validateDeployPipelineCreateOnlyDrift(resource.Spec, current)
}

func validateDeployPipelineCreateOnlyDrift(spec devopsv1beta1.DeployPipelineSpec, current devopssdk.DeployPipeline) error {
	if deployPipelineStringPtrEqual(current.ProjectId, spec.ProjectId) {
		return nil
	}
	return fmt.Errorf("%s create-only field drift is not supported: projectId", deployPipelineKind)
}

func deployPipelineStatusFromResponse(resource *devopsv1beta1.DeployPipeline, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", deployPipelineKind)
	}
	current, ok := deployPipelineFromResponse(response)
	if !ok {
		return nil
	}
	deployPipelineStatus(resource, current)
	return nil
}

func deployPipelineStatus(resource *devopsv1beta1.DeployPipeline, current devopssdk.DeployPipeline) {
	resource.Status = devopsv1beta1.DeployPipelineStatus{
		OsokStatus:                 resource.Status.OsokStatus,
		Id:                         deployPipelineStringValue(current.Id),
		ProjectId:                  deployPipelineStringValue(current.ProjectId),
		CompartmentId:              deployPipelineStringValue(current.CompartmentId),
		DeployPipelineParameters:   statusDeployPipelineParameters(current.DeployPipelineParameters),
		Description:                deployPipelineStringValue(current.Description),
		DisplayName:                deployPipelineStringValue(current.DisplayName),
		DeployPipelineArtifacts:    statusDeployPipelineArtifacts(current.DeployPipelineArtifacts),
		DeployPipelineEnvironments: statusDeployPipelineEnvironments(current.DeployPipelineEnvironments),
		TimeCreated:                deployPipelineSDKTimeString(current.TimeCreated),
		TimeUpdated:                deployPipelineSDKTimeString(current.TimeUpdated),
		LifecycleState:             string(current.LifecycleState),
		LifecycleDetails:           deployPipelineStringValue(current.LifecycleDetails),
		FreeformTags:               cloneDeployPipelineStringMap(current.FreeformTags),
		DefinedTags:                convertDeployPipelineOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:                 convertDeployPipelineOCIToStatusDefinedTags(current.SystemTags),
	}
}

func clearTrackedDeployPipelineIdentity(resource *devopsv1beta1.DeployPipeline) {
	if resource == nil {
		return
	}
	resource.Status = devopsv1beta1.DeployPipelineStatus{}
}

func deployPipelineParameterCollectionForCreate(parameters devopsv1beta1.DeployPipelineParameters) *devopssdk.DeployPipelineParameterCollection {
	if len(parameters.Items) == 0 {
		return nil
	}
	return deployPipelineParameterCollection(parameters)
}

func deployPipelineParameterCollectionForUpdate(parameters devopsv1beta1.DeployPipelineParameters) *devopssdk.DeployPipelineParameterCollection {
	return deployPipelineParameterCollection(parameters)
}

func deployPipelineParameterCollection(parameters devopsv1beta1.DeployPipelineParameters) *devopssdk.DeployPipelineParameterCollection {
	items := make([]devopssdk.DeployPipelineParameter, 0, len(parameters.Items))
	for _, item := range parameters.Items {
		converted := devopssdk.DeployPipelineParameter{Name: common.String(strings.TrimSpace(item.Name))}
		if item.DefaultValue != "" {
			converted.DefaultValue = common.String(item.DefaultValue)
		}
		if item.Description != "" {
			converted.Description = common.String(item.Description)
		}
		items = append(items, converted)
	}
	return &devopssdk.DeployPipelineParameterCollection{Items: items}
}

func deployPipelineParameterCollectionsEqual(
	current *devopssdk.DeployPipelineParameterCollection,
	desired devopsv1beta1.DeployPipelineParameters,
) bool {
	return reflect.DeepEqual(statusDeployPipelineParameters(current).Items, desired.Items)
}

func statusDeployPipelineParameters(current *devopssdk.DeployPipelineParameterCollection) devopsv1beta1.DeployPipelineParameters {
	if current == nil || len(current.Items) == 0 {
		return devopsv1beta1.DeployPipelineParameters{}
	}
	items := make([]devopsv1beta1.DeployPipelineParametersItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, devopsv1beta1.DeployPipelineParametersItem{
			Name:         deployPipelineStringValue(item.Name),
			DefaultValue: deployPipelineStringValue(item.DefaultValue),
			Description:  deployPipelineStringValue(item.Description),
		})
	}
	return devopsv1beta1.DeployPipelineParameters{Items: items}
}

func statusDeployPipelineArtifacts(current *devopssdk.DeployPipelineArtifactCollection) devopsv1beta1.DeployPipelineArtifacts {
	if current == nil || len(current.Items) == 0 {
		return devopsv1beta1.DeployPipelineArtifacts{}
	}
	items := make([]devopsv1beta1.DeployPipelineArtifactsItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, devopsv1beta1.DeployPipelineArtifactsItem{
			DeployArtifactId:     deployPipelineStringValue(item.DeployArtifactId),
			DeployPipelineStages: statusDeployPipelineArtifactStages(item.DeployPipelineStages),
			DisplayName:          deployPipelineStringValue(item.DisplayName),
		})
	}
	return devopsv1beta1.DeployPipelineArtifacts{Items: items}
}

func statusDeployPipelineArtifactStages(current *devopssdk.DeployPipelineStageCollection) devopsv1beta1.DeployPipelineArtifactsItemDeployPipelineStages {
	if current == nil || len(current.Items) == 0 {
		return devopsv1beta1.DeployPipelineArtifactsItemDeployPipelineStages{}
	}
	items := make([]devopsv1beta1.DeployPipelineArtifactsItemDeployPipelineStagesItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, devopsv1beta1.DeployPipelineArtifactsItemDeployPipelineStagesItem{
			DeployStageId: deployPipelineStringValue(item.DeployStageId),
			DisplayName:   deployPipelineStringValue(item.DisplayName),
		})
	}
	return devopsv1beta1.DeployPipelineArtifactsItemDeployPipelineStages{Items: items}
}

func statusDeployPipelineEnvironments(current *devopssdk.DeployPipelineEnvironmentCollection) devopsv1beta1.DeployPipelineEnvironments {
	if current == nil || len(current.Items) == 0 {
		return devopsv1beta1.DeployPipelineEnvironments{}
	}
	items := make([]devopsv1beta1.DeployPipelineEnvironmentsItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, devopsv1beta1.DeployPipelineEnvironmentsItem{
			DeployEnvironmentId:  deployPipelineStringValue(item.DeployEnvironmentId),
			DeployPipelineStages: statusDeployPipelineEnvironmentStages(item.DeployPipelineStages),
			DisplayName:          deployPipelineStringValue(item.DisplayName),
		})
	}
	return devopsv1beta1.DeployPipelineEnvironments{Items: items}
}

func statusDeployPipelineEnvironmentStages(current *devopssdk.DeployPipelineStageCollection) devopsv1beta1.DeployPipelineEnvironmentsItemDeployPipelineStages {
	if current == nil || len(current.Items) == 0 {
		return devopsv1beta1.DeployPipelineEnvironmentsItemDeployPipelineStages{}
	}
	items := make([]devopsv1beta1.DeployPipelineEnvironmentsItemDeployPipelineStagesItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, devopsv1beta1.DeployPipelineEnvironmentsItemDeployPipelineStagesItem{
			DeployStageId: deployPipelineStringValue(item.DeployStageId),
			DisplayName:   deployPipelineStringValue(item.DisplayName),
		})
	}
	return devopsv1beta1.DeployPipelineEnvironmentsItemDeployPipelineStages{Items: items}
}

func desiredDeployPipelineFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneDeployPipelineStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredDeployPipelineDefinedTagsForUpdate(
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

func getDeployPipelineWorkRequest(
	ctx context.Context,
	client deployPipelineOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", deployPipelineKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", deployPipelineKind)
	}

	response, err := client.GetWorkRequest(ctx, devopssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveDeployPipelineWorkRequestAction(workRequest any) (string, error) {
	current, err := deployPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveDeployPipelineWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := deployPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case devopssdk.OperationTypeCreateDeployPipeline:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case devopssdk.OperationTypeUpdateDeployPipeline:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case devopssdk.OperationTypeDeleteDeployPipeline:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverDeployPipelineIDFromWorkRequest(
	resource *devopsv1beta1.DeployPipeline,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if resourceID := deployPipelineTrackedID(resource); resourceID != "" {
		return resourceID, nil
	}
	current, err := deployPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if id, ok := resolveDeployPipelineIDFromResources(current.Resources, deployPipelineWorkRequestActionForPhase(phase), true); ok {
		return id, nil
	}
	if id, ok := resolveDeployPipelineIDFromResources(current.Resources, deployPipelineWorkRequestActionForPhase(phase), false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose a %s identifier", deployPipelineKind, deployPipelineStringValue(current.Id), deployPipelineKind)
}

func deployPipelineWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) devopssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return devopssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return devopssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return devopssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func deployPipelineWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := deployPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", deployPipelineKind, phase, deployPipelineStringValue(current.Id), current.Status)
}

func deployPipelineWorkRequestFromAny(workRequest any) (devopssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case devopssdk.WorkRequest:
		return current, nil
	case *devopssdk.WorkRequest:
		if current == nil {
			return devopssdk.WorkRequest{}, fmt.Errorf("%s work request is nil", deployPipelineKind)
		}
		return *current, nil
	default:
		return devopssdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", deployPipelineKind, workRequest)
	}
}

func resolveDeployPipelineIDFromResources(
	resources []devopssdk.WorkRequestResource,
	action devopssdk.ActionTypeEnum,
	preferDeployPipelineOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferDeployPipelineOnly && !isDeployPipelineWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(deployPipelineStringValue(resource.Identifier))
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

func isDeployPipelineWorkRequestResource(resource devopssdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(deployPipelineStringValue(resource.EntityType)))
	normalizedEntityType := strings.NewReplacer("_", "", "-", "", " ", "").Replace(entityType)
	if strings.Contains(normalizedEntityType, "deploypipeline") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(deployPipelineStringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/deploypipelines/")
}

func listDeployPipelinesAllPages(
	ctx context.Context,
	client deployPipelineOCIClient,
	initErr error,
	request devopssdk.ListDeployPipelinesRequest,
) (devopssdk.ListDeployPipelinesResponse, error) {
	if initErr != nil {
		return devopssdk.ListDeployPipelinesResponse{}, fmt.Errorf("initialize %s OCI client: %w", deployPipelineKind, initErr)
	}
	if client == nil {
		return devopssdk.ListDeployPipelinesResponse{}, fmt.Errorf("%s OCI client is not configured", deployPipelineKind)
	}

	var combined devopssdk.ListDeployPipelinesResponse
	for {
		response, err := client.ListDeployPipelines(ctx, request)
		if err != nil {
			return devopssdk.ListDeployPipelinesResponse{}, conservativeDeployPipelineNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == devopssdk.DeployPipelineLifecycleStateDeleted {
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

func deployPipelineDeleteConfirmRead(
	getDeployPipeline func(context.Context, devopssdk.GetDeployPipelineRequest) (devopssdk.GetDeployPipelineResponse, error),
	listDeployPipelines func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error),
) func(context.Context, *devopsv1beta1.DeployPipeline, string) (any, error) {
	return func(ctx context.Context, resource *devopsv1beta1.DeployPipeline, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID == "" {
			return deployPipelineDeleteConfirmReadByList(ctx, resource, listDeployPipelines)
		}
		if getDeployPipeline == nil {
			return nil, fmt.Errorf("%s delete confirmation requires a readable OCI operation", deployPipelineKind)
		}

		response, err := getDeployPipeline(ctx, devopssdk.GetDeployPipelineRequest{
			DeployPipelineId: common.String(currentID),
		})
		if err == nil {
			return response, nil
		}
		if !isDeployPipelineAmbiguousNotFound(err) {
			return nil, err
		}
		handledErr := handleDeployPipelineDeleteError(resource, err)
		if handledErr == nil {
			handledErr = err
		}
		return deployPipelineAmbiguousDeleteConfirmResponse{
			DeployPipeline: devopssdk.DeployPipeline{
				Id:             common.String(currentID),
				LifecycleState: devopssdk.DeployPipelineLifecycleStateActive,
			},
			err: handledErr,
		}, nil
	}
}

func deployPipelineDeleteConfirmReadByList(
	ctx context.Context,
	resource *devopsv1beta1.DeployPipeline,
	listDeployPipelines func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error),
) (any, error) {
	response, found, err := deployPipelineDeleteResolutionByList(ctx, resource, listDeployPipelines)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%s delete confirmation found no pipeline matching projectId %q and displayName %q", deployPipelineKind, strings.TrimSpace(resource.Spec.ProjectId), strings.TrimSpace(resource.Spec.DisplayName))
	}
	return response, nil
}

func deployPipelineDeleteResolutionByList(
	ctx context.Context,
	resource *devopsv1beta1.DeployPipeline,
	listDeployPipelines func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error),
) (devopssdk.GetDeployPipelineResponse, bool, error) {
	if resource == nil {
		return devopssdk.GetDeployPipelineResponse{}, false, fmt.Errorf("%s resource is nil", deployPipelineKind)
	}
	projectID := strings.TrimSpace(resource.Spec.ProjectId)
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if projectID == "" || displayName == "" {
		return devopssdk.GetDeployPipelineResponse{}, false, fmt.Errorf("%s delete confirmation requires projectId and displayName when no pipeline OCID is tracked", deployPipelineKind)
	}
	if listDeployPipelines == nil {
		return devopssdk.GetDeployPipelineResponse{}, false, fmt.Errorf("%s delete confirmation requires a list OCI operation", deployPipelineKind)
	}

	response, err := listDeployPipelines(ctx, devopssdk.ListDeployPipelinesRequest{
		ProjectId:   common.String(projectID),
		DisplayName: common.String(displayName),
	})
	if err != nil {
		return devopssdk.GetDeployPipelineResponse{}, false, err
	}
	var matches []devopssdk.DeployPipelineSummary
	for _, item := range response.Items {
		if deployPipelineSummaryMatchesDeleteIdentity(item, projectID, displayName) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return devopssdk.GetDeployPipelineResponse{OpcRequestId: response.OpcRequestId}, false, nil
	case 1:
		return devopssdk.GetDeployPipelineResponse{
			DeployPipeline: deployPipelineFromSummary(matches[0]),
			OpcRequestId:   response.OpcRequestId,
		}, true, nil
	default:
		return devopssdk.GetDeployPipelineResponse{}, false, fmt.Errorf("%s delete confirmation found %d pipelines matching projectId %q and displayName %q", deployPipelineKind, len(matches), projectID, displayName)
	}
}

func deployPipelineSummaryMatchesDeleteIdentity(item devopssdk.DeployPipelineSummary, projectID string, displayName string) bool {
	return deployPipelineStringValue(item.ProjectId) == projectID &&
		deployPipelineStringValue(item.DisplayName) == displayName
}

func wrapDeployPipelineDeleteWithoutTrackedID(
	listDeployPipelines func(context.Context, devopssdk.ListDeployPipelinesRequest) (devopssdk.ListDeployPipelinesResponse, error),
) func(DeployPipelineServiceClient) DeployPipelineServiceClient {
	return func(delegate DeployPipelineServiceClient) DeployPipelineServiceClient {
		return deployPipelineDeleteWithoutTrackedIDClient{
			DeployPipelineServiceClient: delegate,
			listDeployPipelines:         listDeployPipelines,
		}
	}
}

func (c deployPipelineDeleteWithoutTrackedIDClient) Delete(
	ctx context.Context,
	resource *devopsv1beta1.DeployPipeline,
) (bool, error) {
	if deployPipelineTrackedID(resource) != "" {
		return c.DeployPipelineServiceClient.Delete(ctx, resource)
	}
	response, found, err := deployPipelineDeleteResolutionByList(ctx, resource, c.listDeployPipelines)
	if err != nil {
		return false, err
	}
	if !found {
		markDeployPipelineDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if deployPipelineID := deployPipelineStringValue(response.Id); deployPipelineID != "" {
		resource.Status.Id = deployPipelineID
		resource.Status.OsokStatus.Ocid = shared.OCID(deployPipelineID)
	}
	return c.DeployPipelineServiceClient.Delete(ctx, resource)
}

func handleDeployPipelineDeleteError(resource *devopsv1beta1.DeployPipeline, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func applyDeployPipelineDeleteOutcome(
	_ *devopsv1beta1.DeployPipeline,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if err, ok := deployPipelineAmbiguousDeleteConfirmError(response); ok {
		return generatedruntime.DeleteOutcome{Handled: true}, err
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markDeployPipelineDeleted(resource *devopsv1beta1.DeployPipeline, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func deployPipelineFromResponse(response any) (devopssdk.DeployPipeline, bool) {
	if current, ok := deployPipelineFromDirectResponse(response); ok {
		return current, true
	}
	return deployPipelineFromOperationResponse(response)
}

func deployPipelineFromOperationResponse(response any) (devopssdk.DeployPipeline, bool) {
	switch current := response.(type) {
	case devopssdk.CreateDeployPipelineResponse:
		return current.DeployPipeline, true
	case *devopssdk.CreateDeployPipelineResponse:
		if current == nil {
			return devopssdk.DeployPipeline{}, false
		}
		return current.DeployPipeline, true
	case devopssdk.GetDeployPipelineResponse:
		return current.DeployPipeline, true
	case *devopssdk.GetDeployPipelineResponse:
		if current == nil {
			return devopssdk.DeployPipeline{}, false
		}
		return current.DeployPipeline, true
	case devopssdk.UpdateDeployPipelineResponse:
		return current.DeployPipeline, true
	case *devopssdk.UpdateDeployPipelineResponse:
		if current == nil {
			return devopssdk.DeployPipeline{}, false
		}
		return current.DeployPipeline, true
	default:
		return devopssdk.DeployPipeline{}, false
	}
}

func deployPipelineFromDirectResponse(response any) (devopssdk.DeployPipeline, bool) {
	switch current := response.(type) {
	case devopssdk.DeployPipeline:
		return current, true
	case *devopssdk.DeployPipeline:
		if current == nil {
			return devopssdk.DeployPipeline{}, false
		}
		return *current, true
	case devopssdk.DeployPipelineSummary:
		return deployPipelineFromSummary(current), true
	case *devopssdk.DeployPipelineSummary:
		if current == nil {
			return devopssdk.DeployPipeline{}, false
		}
		return deployPipelineFromSummary(*current), true
	case deployPipelineAmbiguousDeleteConfirmResponse:
		return current.DeployPipeline, true
	case *deployPipelineAmbiguousDeleteConfirmResponse:
		if current == nil {
			return devopssdk.DeployPipeline{}, false
		}
		return current.DeployPipeline, true
	default:
		return devopssdk.DeployPipeline{}, false
	}
}

func deployPipelineFromSummary(summary devopssdk.DeployPipelineSummary) devopssdk.DeployPipeline {
	return devopssdk.DeployPipeline{
		Id:                       summary.Id,
		ProjectId:                summary.ProjectId,
		CompartmentId:            summary.CompartmentId,
		DeployPipelineParameters: summary.DeployPipelineParameters,
		Description:              summary.Description,
		DisplayName:              summary.DisplayName,
		TimeCreated:              summary.TimeCreated,
		TimeUpdated:              summary.TimeUpdated,
		LifecycleState:           summary.LifecycleState,
		LifecycleDetails:         summary.LifecycleDetails,
		FreeformTags:             cloneDeployPipelineStringMap(summary.FreeformTags),
		DefinedTags:              cloneDeployPipelineDefinedTags(summary.DefinedTags),
		SystemTags:               cloneDeployPipelineDefinedTags(summary.SystemTags),
	}
}

func deployPipelineTrackedID(resource *devopsv1beta1.DeployPipeline) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func conservativeDeployPipelineNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", deployPipelineKind, strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousDeployPipelineNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousDeployPipelineNotFoundError{message: message}
}

func isDeployPipelineAmbiguousNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousDeployPipelineNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func deployPipelineAmbiguousDeleteConfirmError(response any) (error, bool) {
	switch current := response.(type) {
	case deployPipelineAmbiguousDeleteConfirmResponse:
		return current.err, true
	case *deployPipelineAmbiguousDeleteConfirmResponse:
		if current == nil {
			return nil, false
		}
		return current.err, true
	default:
		return nil, false
	}
}

func deployPipelineStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func deployPipelineStringPtrEqual(actual *string, expected string) bool {
	return strings.TrimSpace(deployPipelineStringValue(actual)) == strings.TrimSpace(expected)
}

func deployPipelineSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func cloneDeployPipelineStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneDeployPipelineDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
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

func convertDeployPipelineOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
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
