/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package buildpipeline

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
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

const buildPipelineAmbiguousNotFoundErrorCode = "BuildPipelineAmbiguousNotFound"

var buildPipelineWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
	CreateActionTokens:    []string{string(devopssdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(devopssdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(devopssdk.ActionTypeDeleted)},
}

type buildPipelineOCIClient interface {
	CreateBuildPipeline(context.Context, devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error)
	GetBuildPipeline(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error)
	ListBuildPipelines(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error)
	UpdateBuildPipeline(context.Context, devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error)
	DeleteBuildPipeline(context.Context, devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error)
	GetWorkRequest(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

type buildPipelineDeleteWithoutTrackedIDClient struct {
	BuildPipelineServiceClient
	listBuildPipelines func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error)
	getBuildPipeline   func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error)
	getWorkRequest     func(context.Context, string) (any, error)
}

type buildPipelineDeleteListIdentity struct {
	projectID   string
	displayName string
}

type buildPipelineAmbiguousNotFoundError struct {
	HTTPStatusCode int
	ErrorCode      string
	OpcRequestID   string
	message        string
}

func (e buildPipelineAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e buildPipelineAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.OpcRequestID
}

func init() {
	registerBuildPipelineRuntimeHooksMutator(func(manager *BuildPipelineServiceManager, hooks *BuildPipelineRuntimeHooks) {
		client, initErr := newBuildPipelineSDKClient(manager)
		applyBuildPipelineRuntimeHooks(hooks, client, initErr)
	})
}

func newBuildPipelineSDKClient(manager *BuildPipelineServiceManager) (buildPipelineOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("BuildPipeline service manager is nil")
	}
	client, err := devopssdk.NewDevopsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyBuildPipelineRuntimeHooks(
	hooks *BuildPipelineRuntimeHooks,
	client buildPipelineOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedBuildPipelineRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *devopsv1beta1.BuildPipeline, _ string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("BuildPipeline resource is nil")
		}
		return buildBuildPipelineCreateBody(resource.Spec)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *devopsv1beta1.BuildPipeline,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildBuildPipelineUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = func(_ context.Context, resource *devopsv1beta1.BuildPipeline) (generatedruntime.ExistingBeforeCreateDecision, error) {
		if resource == nil {
			return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("BuildPipeline resource is nil")
		}
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
		}
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	hooks.Get.Call = func(ctx context.Context, request devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
		if initErr != nil {
			return devopssdk.GetBuildPipelineResponse{}, fmt.Errorf("initialize BuildPipeline OCI client: %w", initErr)
		}
		if client == nil {
			return devopssdk.GetBuildPipelineResponse{}, fmt.Errorf("BuildPipeline OCI client is not configured")
		}
		response, err := client.GetBuildPipeline(ctx, request)
		return response, conservativeBuildPipelineNotFoundError(err, "read")
	}
	hooks.List.Fields = buildPipelineListFields()
	hooks.List.Call = func(ctx context.Context, request devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
		return listBuildPipelinesAllPages(ctx, client, initErr, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
		if initErr != nil {
			return devopssdk.DeleteBuildPipelineResponse{}, fmt.Errorf("initialize BuildPipeline OCI client: %w", initErr)
		}
		if client == nil {
			return devopssdk.DeleteBuildPipelineResponse{}, fmt.Errorf("BuildPipeline OCI client is not configured")
		}
		response, err := client.DeleteBuildPipeline(ctx, request)
		return response, conservativeBuildPipelineNotFoundError(err, "delete")
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedBuildPipelineIdentity
	hooks.StatusHooks.ProjectStatus = buildPipelineStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateBuildPipelineCreateOnlyDriftForResponse
	hooks.DeleteHooks.ConfirmRead = buildPipelineDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleBuildPipelineDeleteError
	hooks.Async.Adapter = buildPipelineWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getBuildPipelineWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveBuildPipelineGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveBuildPipelineGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverBuildPipelineIDFromGeneratedWorkRequest
	hooks.Async.Message = buildPipelineGeneratedWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapBuildPipelineDeleteWithoutTrackedID(
		hooks.List.Call,
		hooks.Get.Call,
		hooks.Async.GetWorkRequest,
	))
}

func newBuildPipelineServiceClientWithOCIClient(log loggerutil.OSOKLogger, client buildPipelineOCIClient) BuildPipelineServiceClient {
	manager := &BuildPipelineServiceManager{Log: log}
	hooks := newBuildPipelineRuntimeHooksWithOCIClient(client)
	applyBuildPipelineRuntimeHooks(&hooks, client, nil)
	return wrapBuildPipelineGeneratedClient(
		hooks,
		defaultBuildPipelineServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*devopsv1beta1.BuildPipeline](
				buildBuildPipelineGeneratedRuntimeConfig(manager, hooks),
			),
		},
	)
}

func newBuildPipelineRuntimeHooksWithOCIClient(client buildPipelineOCIClient) BuildPipelineRuntimeHooks {
	return BuildPipelineRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*devopsv1beta1.BuildPipeline]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*devopsv1beta1.BuildPipeline]{},
		StatusHooks:     generatedruntime.StatusHooks[*devopsv1beta1.BuildPipeline]{},
		ParityHooks:     generatedruntime.ParityHooks[*devopsv1beta1.BuildPipeline]{},
		Async:           generatedruntime.AsyncHooks[*devopsv1beta1.BuildPipeline]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*devopsv1beta1.BuildPipeline]{},
		Create: runtimeOperationHooks[devopssdk.CreateBuildPipelineRequest, devopssdk.CreateBuildPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateBuildPipelineDetails", RequestName: "CreateBuildPipelineDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request devopssdk.CreateBuildPipelineRequest) (devopssdk.CreateBuildPipelineResponse, error) {
				return client.CreateBuildPipeline(ctx, request)
			},
		},
		Get: runtimeOperationHooks[devopssdk.GetBuildPipelineRequest, devopssdk.GetBuildPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "BuildPipelineId", RequestName: "buildPipelineId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error) {
				return client.GetBuildPipeline(ctx, request)
			},
		},
		List: runtimeOperationHooks[devopssdk.ListBuildPipelinesRequest, devopssdk.ListBuildPipelinesResponse]{
			Fields: buildPipelineListFields(),
			Call: func(ctx context.Context, request devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error) {
				return client.ListBuildPipelines(ctx, request)
			},
		},
		Update: runtimeOperationHooks[devopssdk.UpdateBuildPipelineRequest, devopssdk.UpdateBuildPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "BuildPipelineId", RequestName: "buildPipelineId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateBuildPipelineDetails", RequestName: "UpdateBuildPipelineDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request devopssdk.UpdateBuildPipelineRequest) (devopssdk.UpdateBuildPipelineResponse, error) {
				return client.UpdateBuildPipeline(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[devopssdk.DeleteBuildPipelineRequest, devopssdk.DeleteBuildPipelineResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "BuildPipelineId", RequestName: "buildPipelineId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request devopssdk.DeleteBuildPipelineRequest) (devopssdk.DeleteBuildPipelineResponse, error) {
				return client.DeleteBuildPipeline(ctx, request)
			},
		},
		WrapGeneratedClient: []func(BuildPipelineServiceClient) BuildPipelineServiceClient{},
	}
}

func reviewedBuildPipelineRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "devops",
		FormalSlug:    "buildpipeline",
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
			ProvisioningStates: []string{string(devopssdk.BuildPipelineLifecycleStateCreating)},
			UpdatingStates:     []string{string(devopssdk.BuildPipelineLifecycleStateUpdating)},
			ActiveStates:       []string{string(devopssdk.BuildPipelineLifecycleStateActive), string(devopssdk.BuildPipelineLifecycleStateInactive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(devopssdk.BuildPipelineLifecycleStateDeleting)},
			TerminalStates: []string{string(devopssdk.BuildPipelineLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"projectId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"description",
				"displayName",
				"buildPipelineParameters",
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

func buildPipelineListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ProjectId", RequestName: "projectId", Contribution: "query", LookupPaths: []string{"status.projectId", "spec.projectId", "projectId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func buildBuildPipelineCreateBody(spec devopsv1beta1.BuildPipelineSpec) (devopssdk.CreateBuildPipelineDetails, error) {
	if err := validateBuildPipelineSpec(spec); err != nil {
		return devopssdk.CreateBuildPipelineDetails{}, err
	}

	body := devopssdk.CreateBuildPipelineDetails{
		ProjectId: common.String(strings.TrimSpace(spec.ProjectId)),
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	if strings.TrimSpace(spec.DisplayName) != "" {
		body.DisplayName = common.String(spec.DisplayName)
	}
	if parameters := buildPipelineParametersFromSpec(spec.BuildPipelineParameters); parameters != nil {
		body.BuildPipelineParameters = parameters
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	return body, nil
}

func buildBuildPipelineUpdateBody(
	resource *devopsv1beta1.BuildPipeline,
	currentResponse any,
) (devopssdk.UpdateBuildPipelineDetails, bool, error) {
	if resource == nil {
		return devopssdk.UpdateBuildPipelineDetails{}, false, fmt.Errorf("BuildPipeline resource is nil")
	}
	if err := validateBuildPipelineSpec(resource.Spec); err != nil {
		return devopssdk.UpdateBuildPipelineDetails{}, false, err
	}

	current, ok := buildPipelineFromResponse(currentResponse)
	if !ok {
		return devopssdk.UpdateBuildPipelineDetails{}, false, fmt.Errorf("current BuildPipeline response does not expose a BuildPipeline body")
	}
	if err := validateBuildPipelineCreateOnlyDrift(resource.Spec, current); err != nil {
		return devopssdk.UpdateBuildPipelineDetails{}, false, err
	}

	updateDetails := devopssdk.UpdateBuildPipelineDetails{}
	updateNeeded := false

	updateNeeded = applyBuildPipelineStringUpdates(&updateDetails, current, resource.Spec) || updateNeeded
	updateNeeded = applyBuildPipelineParameterUpdate(&updateDetails, current, resource.Spec) || updateNeeded
	updateNeeded = applyBuildPipelineTagUpdates(&updateDetails, current, resource.Spec) || updateNeeded

	if !updateNeeded {
		return devopssdk.UpdateBuildPipelineDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func applyBuildPipelineStringUpdates(
	updateDetails *devopssdk.UpdateBuildPipelineDetails,
	current devopssdk.BuildPipeline,
	spec devopsv1beta1.BuildPipelineSpec,
) bool {
	updateNeeded := false
	if desired, ok := desiredBuildPipelineStringForUpdate(spec.Description, current.Description); ok {
		updateDetails.Description = desired
		updateNeeded = true
	}
	if desired, ok := desiredBuildPipelineStringForUpdate(spec.DisplayName, current.DisplayName); ok {
		updateDetails.DisplayName = desired
		updateNeeded = true
	}
	return updateNeeded
}

func applyBuildPipelineParameterUpdate(
	updateDetails *devopssdk.UpdateBuildPipelineDetails,
	current devopssdk.BuildPipeline,
	spec devopsv1beta1.BuildPipelineSpec,
) bool {
	if spec.BuildPipelineParameters.Items == nil ||
		buildPipelineParametersEqual(current.BuildPipelineParameters, spec.BuildPipelineParameters) {
		return false
	}

	updateDetails.BuildPipelineParameters = buildPipelineParametersFromSpec(spec.BuildPipelineParameters)
	if updateDetails.BuildPipelineParameters == nil {
		updateDetails.BuildPipelineParameters = &devopssdk.BuildPipelineParameterCollection{}
	}
	return true
}

func applyBuildPipelineTagUpdates(
	updateDetails *devopssdk.UpdateBuildPipelineDetails,
	current devopssdk.BuildPipeline,
	spec devopsv1beta1.BuildPipelineSpec,
) bool {
	updateNeeded := false
	if desiredFreeformTags, ok := desiredBuildPipelineFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags); ok {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	if desiredDefinedTags, ok := desiredBuildPipelineDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags); ok {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}
	return updateNeeded
}

func validateBuildPipelineSpec(spec devopsv1beta1.BuildPipelineSpec) error {
	var missing []string
	if strings.TrimSpace(spec.ProjectId) == "" {
		missing = append(missing, "projectId")
	}
	for index, parameter := range spec.BuildPipelineParameters.Items {
		if strings.TrimSpace(parameter.Name) == "" {
			missing = append(missing, fmt.Sprintf("buildPipelineParameters.items[%d].name", index))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("BuildPipeline spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if duplicate := duplicateBuildPipelineParameterName(spec.BuildPipelineParameters); duplicate != "" {
		return fmt.Errorf("BuildPipeline spec has duplicate buildPipelineParameters item name %q", duplicate)
	}
	return nil
}

func buildPipelineParametersFromSpec(params devopsv1beta1.BuildPipelineParameters) *devopssdk.BuildPipelineParameterCollection {
	if params.Items == nil {
		return nil
	}
	items := make([]devopssdk.BuildPipelineParameter, 0, len(params.Items))
	for _, item := range params.Items {
		parameter := devopssdk.BuildPipelineParameter{
			Name:         common.String(strings.TrimSpace(item.Name)),
			DefaultValue: common.String(item.DefaultValue),
		}
		if strings.TrimSpace(item.Description) != "" {
			parameter.Description = common.String(item.Description)
		}
		items = append(items, parameter)
	}
	return &devopssdk.BuildPipelineParameterCollection{Items: items}
}

type comparableBuildPipelineParameter struct {
	name         string
	defaultValue string
	description  string
}

func buildPipelineParametersEqual(
	current *devopssdk.BuildPipelineParameterCollection,
	desired devopsv1beta1.BuildPipelineParameters,
) bool {
	return reflect.DeepEqual(
		normalizedCurrentBuildPipelineParameters(current),
		normalizedDesiredBuildPipelineParameters(desired),
	)
}

func normalizedDesiredBuildPipelineParameters(params devopsv1beta1.BuildPipelineParameters) []comparableBuildPipelineParameter {
	items := make([]comparableBuildPipelineParameter, 0, len(params.Items))
	for _, item := range params.Items {
		items = append(items, comparableBuildPipelineParameter{
			name:         strings.TrimSpace(item.Name),
			defaultValue: item.DefaultValue,
			description:  item.Description,
		})
	}
	sortBuildPipelineParameters(items)
	return items
}

func normalizedCurrentBuildPipelineParameters(params *devopssdk.BuildPipelineParameterCollection) []comparableBuildPipelineParameter {
	if params == nil {
		return nil
	}
	items := make([]comparableBuildPipelineParameter, 0, len(params.Items))
	for _, item := range params.Items {
		items = append(items, comparableBuildPipelineParameter{
			name:         stringValue(item.Name),
			defaultValue: stringValue(item.DefaultValue),
			description:  stringValue(item.Description),
		})
	}
	sortBuildPipelineParameters(items)
	return items
}

func sortBuildPipelineParameters(items []comparableBuildPipelineParameter) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].name != items[j].name {
			return items[i].name < items[j].name
		}
		if items[i].defaultValue != items[j].defaultValue {
			return items[i].defaultValue < items[j].defaultValue
		}
		return items[i].description < items[j].description
	})
}

func duplicateBuildPipelineParameterName(params devopsv1beta1.BuildPipelineParameters) string {
	seen := map[string]struct{}{}
	for _, item := range params.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			return name
		}
		seen[name] = struct{}{}
	}
	return ""
}

func validateBuildPipelineCreateOnlyDriftForResponse(resource *devopsv1beta1.BuildPipeline, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("BuildPipeline resource is nil")
	}
	current, ok := buildPipelineFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current BuildPipeline response does not expose a BuildPipeline body")
	}
	return validateBuildPipelineCreateOnlyDrift(resource.Spec, current)
}

func validateBuildPipelineCreateOnlyDrift(spec devopsv1beta1.BuildPipelineSpec, current devopssdk.BuildPipeline) error {
	if !stringPtrEqual(current.ProjectId, spec.ProjectId) {
		return fmt.Errorf("BuildPipeline create-only field drift is not supported: projectId")
	}
	return nil
}

func buildPipelineStatusFromResponse(resource *devopsv1beta1.BuildPipeline, response any) error {
	if resource == nil {
		return fmt.Errorf("BuildPipeline resource is nil")
	}
	current, ok := buildPipelineFromResponse(response)
	if !ok {
		return nil
	}
	buildPipelineStatus(resource, current)
	return nil
}

func buildPipelineStatus(resource *devopsv1beta1.BuildPipeline, current devopssdk.BuildPipeline) {
	resource.Status = devopsv1beta1.BuildPipelineStatus{
		OsokStatus:              resource.Status.OsokStatus,
		Id:                      stringValue(current.Id),
		CompartmentId:           stringValue(current.CompartmentId),
		ProjectId:               stringValue(current.ProjectId),
		Description:             stringValue(current.Description),
		DisplayName:             stringValue(current.DisplayName),
		TimeCreated:             sdkTimeString(current.TimeCreated),
		TimeUpdated:             sdkTimeString(current.TimeUpdated),
		LifecycleState:          string(current.LifecycleState),
		LifecycleDetails:        stringValue(current.LifecycleDetails),
		BuildPipelineParameters: statusBuildPipelineParameters(current.BuildPipelineParameters),
		FreeformTags:            cloneStringMap(current.FreeformTags),
		DefinedTags:             convertOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:              convertOCIToStatusDefinedTags(current.SystemTags),
	}
}

func statusBuildPipelineParameters(params *devopssdk.BuildPipelineParameterCollection) devopsv1beta1.BuildPipelineParameters {
	if params == nil {
		return devopsv1beta1.BuildPipelineParameters{}
	}
	items := make([]devopsv1beta1.BuildPipelineParametersItem, 0, len(params.Items))
	for _, item := range params.Items {
		items = append(items, devopsv1beta1.BuildPipelineParametersItem{
			Name:         stringValue(item.Name),
			DefaultValue: stringValue(item.DefaultValue),
			Description:  stringValue(item.Description),
		})
	}
	return devopsv1beta1.BuildPipelineParameters{Items: items}
}

func clearTrackedBuildPipelineIdentity(resource *devopsv1beta1.BuildPipeline) {
	if resource == nil {
		return
	}
	resource.Status = devopsv1beta1.BuildPipelineStatus{}
}

func desiredBuildPipelineFreeformTagsForUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	desired := cloneStringMap(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func desiredBuildPipelineDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := *util.ConvertToOciDefinedTags(&spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func desiredBuildPipelineStringForUpdate(spec string, current *string) (*string, bool) {
	trimmedSpec := strings.TrimSpace(spec)
	if trimmedSpec == "" || trimmedSpec == strings.TrimSpace(stringValue(current)) {
		return nil, false
	}
	return common.String(spec), true
}

func wrapBuildPipelineDeleteWithoutTrackedID(
	listBuildPipelines func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error),
	getBuildPipeline func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error),
	getWorkRequest func(context.Context, string) (any, error),
) func(BuildPipelineServiceClient) BuildPipelineServiceClient {
	return func(delegate BuildPipelineServiceClient) BuildPipelineServiceClient {
		return buildPipelineDeleteWithoutTrackedIDClient{
			BuildPipelineServiceClient: delegate,
			listBuildPipelines:         listBuildPipelines,
			getBuildPipeline:           getBuildPipeline,
			getWorkRequest:             getWorkRequest,
		}
	}
}

func (c buildPipelineDeleteWithoutTrackedIDClient) Delete(
	ctx context.Context,
	resource *devopsv1beta1.BuildPipeline,
) (bool, error) {
	if workRequestID, phase := currentBuildPipelineWriteWorkRequest(resource); workRequestID != "" {
		return c.resumeWriteWorkRequestBeforeDelete(ctx, resource, workRequestID, phase)
	}

	if buildPipelineTrackedID(resource) != "" {
		return c.BuildPipelineServiceClient.Delete(ctx, resource)
	}

	response, found, err := buildPipelineDeleteResolutionByList(ctx, resource, c.listBuildPipelines)
	if err != nil {
		return false, err
	}
	if !found {
		markBuildPipelineDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if buildPipelineID := stringValue(response.Id); buildPipelineID != "" {
		resource.Status.Id = buildPipelineID
		resource.Status.OsokStatus.Ocid = shared.OCID(buildPipelineID)
	}
	return c.BuildPipelineServiceClient.Delete(ctx, resource)
}

func (c buildPipelineDeleteWithoutTrackedIDClient) resumeWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *devopsv1beta1.BuildPipeline,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	if c.getWorkRequest == nil {
		return false, fmt.Errorf("BuildPipeline work request polling is not configured")
	}

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	current, err := buildBuildPipelineAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return false, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("BuildPipeline %s work request %s is still in progress; waiting before delete", phase, workRequestID)
		markBuildPipelineWorkRequestOperation(resource, current, shared.OSOKAsyncClassPending, message)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.deleteAfterSucceededWriteWorkRequest(ctx, resource, workRequest, current, workRequestID)
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("BuildPipeline %s work request %s finished with status %s before delete", phase, workRequestID, current.RawStatus)
		markBuildPipelineWorkRequestOperation(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return false, err
	default:
		err := fmt.Errorf("BuildPipeline %s work request %s projected unsupported async class %s before delete", phase, workRequestID, current.NormalizedClass)
		markBuildPipelineWorkRequestOperation(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return false, err
	}
}

func (c buildPipelineDeleteWithoutTrackedIDClient) deleteAfterSucceededWriteWorkRequest(
	ctx context.Context,
	resource *devopsv1beta1.BuildPipeline,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID, err := recoverBuildPipelineIDFromGeneratedWorkRequest(resource, workRequest, current.Phase)
	if err != nil && buildPipelineTrackedID(resource) == "" {
		markBuildPipelineWorkRequestOperation(resource, current, shared.OSOKAsyncClassFailed, err.Error())
		return false, err
	}
	if strings.TrimSpace(resourceID) == "" {
		resourceID = buildPipelineTrackedID(resource)
	}
	response, err := c.readBuildPipelineAfterWrite(ctx, strings.TrimSpace(resourceID))
	if err != nil {
		if isBuildPipelineReadNotFound(err) {
			markBuildPipelineWriteReadbackPending(resource, current, workRequestID, resourceID)
			return false, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	buildPipelineStatus(resource, response.BuildPipeline)
	if id := strings.TrimSpace(stringValue(response.Id)); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return c.BuildPipelineServiceClient.Delete(ctx, resource)
}

func (c buildPipelineDeleteWithoutTrackedIDClient) readBuildPipelineAfterWrite(
	ctx context.Context,
	buildPipelineID string,
) (devopssdk.GetBuildPipelineResponse, error) {
	if strings.TrimSpace(buildPipelineID) == "" {
		return devopssdk.GetBuildPipelineResponse{}, fmt.Errorf("BuildPipeline succeeded write work request did not expose a build pipeline identifier")
	}
	if c.getBuildPipeline == nil {
		return devopssdk.GetBuildPipelineResponse{}, fmt.Errorf("BuildPipeline readback is not configured")
	}
	return c.getBuildPipeline(ctx, devopssdk.GetBuildPipelineRequest{
		BuildPipelineId: common.String(strings.TrimSpace(buildPipelineID)),
	})
}

func currentBuildPipelineWriteWorkRequest(resource *devopsv1beta1.BuildPipeline) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return strings.TrimSpace(current.WorkRequestID), current.Phase
	default:
		return "", ""
	}
}

func buildBuildPipelineAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := buildPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	action, err := resolveBuildPipelineWorkRequestAction(current)
	if err != nil {
		return nil, err
	}
	derivedPhase, ok := buildPipelineWorkRequestPhaseFromOperationType(current.OperationType)
	if ok {
		if phase != "" && phase != derivedPhase {
			return nil, fmt.Errorf("BuildPipeline work request %s exposes phase %q while delete expected %q", stringValue(current.Id), derivedPhase, phase)
		}
		phase = derivedPhase
	}
	operation, err := servicemanager.BuildWorkRequestAsyncOperation(status, buildPipelineWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        action,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    stringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    phase,
	})
	if err != nil {
		return nil, err
	}
	if message := strings.TrimSpace(buildPipelineGeneratedWorkRequestMessage(operation.Phase, current)); message != "" {
		operation.Message = message
	}
	return operation, nil
}

func markBuildPipelineWorkRequestOperation(
	resource *devopsv1beta1.BuildPipeline,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) {
	if resource == nil || current == nil {
		return
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	now := metav1.Now()
	next.UpdatedAt = &now
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, loggerutil.OSOKLogger{})
}

func markBuildPipelineWriteReadbackPending(
	resource *devopsv1beta1.BuildPipeline,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
	resourceID string,
) {
	if strings.TrimSpace(resourceID) != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	message := fmt.Sprintf(
		"BuildPipeline %s work request %s succeeded; waiting for BuildPipeline %s to become readable",
		current.Phase,
		strings.TrimSpace(workRequestID),
		strings.TrimSpace(resourceID),
	)
	markBuildPipelineWorkRequestOperation(resource, current, shared.OSOKAsyncClassPending, message)
}

func isBuildPipelineReadNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func buildPipelineTrackedID(resource *devopsv1beta1.BuildPipeline) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func buildPipelineDeleteConfirmRead(
	getBuildPipeline func(context.Context, devopssdk.GetBuildPipelineRequest) (devopssdk.GetBuildPipelineResponse, error),
	listBuildPipelines func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error),
) func(context.Context, *devopsv1beta1.BuildPipeline, string) (any, error) {
	return func(ctx context.Context, resource *devopsv1beta1.BuildPipeline, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID == "" {
			response, found, err := buildPipelineDeleteResolutionByList(ctx, resource, listBuildPipelines)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("BuildPipeline delete confirmation found no build pipeline matching projectId %q and displayName %q", strings.TrimSpace(resource.Spec.ProjectId), strings.TrimSpace(resource.Spec.DisplayName))
			}
			return response, nil
		}
		if getBuildPipeline == nil {
			return nil, fmt.Errorf("BuildPipeline delete confirmation requires a readable OCI operation")
		}
		return getBuildPipeline(ctx, devopssdk.GetBuildPipelineRequest{BuildPipelineId: common.String(currentID)})
	}
}

func buildPipelineDeleteResolutionByList(
	ctx context.Context,
	resource *devopsv1beta1.BuildPipeline,
	listBuildPipelines func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error),
) (devopssdk.BuildPipeline, bool, error) {
	identity, ok, err := buildPipelineDeleteListIdentityFor(resource, listBuildPipelines)
	if err != nil {
		return devopssdk.BuildPipeline{}, false, err
	}
	if !ok {
		return devopssdk.BuildPipeline{}, false, nil
	}

	matches, err := buildPipelineDeleteListMatches(ctx, identity, listBuildPipelines)
	if err != nil {
		return devopssdk.BuildPipeline{}, false, err
	}
	return buildPipelineDeleteResolutionFromMatches(identity, matches)
}

func buildPipelineDeleteListIdentityFor(
	resource *devopsv1beta1.BuildPipeline,
	listBuildPipelines func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error),
) (buildPipelineDeleteListIdentity, bool, error) {
	if resource == nil {
		return buildPipelineDeleteListIdentity{}, false, fmt.Errorf("BuildPipeline resource is nil")
	}

	identity := buildPipelineDeleteListIdentity{
		projectID:   strings.TrimSpace(resource.Spec.ProjectId),
		displayName: strings.TrimSpace(resource.Spec.DisplayName),
	}
	if identity.projectID == "" {
		return buildPipelineDeleteListIdentity{}, false, fmt.Errorf("BuildPipeline delete confirmation requires projectId when no build pipeline OCID is tracked")
	}
	if identity.displayName == "" {
		return identity, false, nil
	}
	if listBuildPipelines == nil {
		return buildPipelineDeleteListIdentity{}, false, fmt.Errorf("BuildPipeline delete confirmation requires a tracked build pipeline OCID")
	}
	return identity, true, nil
}

func buildPipelineDeleteListMatches(
	ctx context.Context,
	identity buildPipelineDeleteListIdentity,
	listBuildPipelines func(context.Context, devopssdk.ListBuildPipelinesRequest) (devopssdk.ListBuildPipelinesResponse, error),
) ([]devopssdk.BuildPipelineSummary, error) {
	var matches []devopssdk.BuildPipelineSummary
	var page *string
	for {
		response, err := listBuildPipelines(ctx, devopssdk.ListBuildPipelinesRequest{
			ProjectId:   common.String(identity.projectID),
			DisplayName: common.String(identity.displayName),
			Page:        page,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range response.Items {
			if buildPipelineSummaryMatchesDeleteIdentity(item, identity) {
				matches = append(matches, item)
			}
		}
		page = nextPage(response.OpcNextPage)
		if page == nil {
			return matches, nil
		}
	}
}

func buildPipelineSummaryMatchesDeleteIdentity(
	item devopssdk.BuildPipelineSummary,
	identity buildPipelineDeleteListIdentity,
) bool {
	return stringValue(item.ProjectId) == identity.projectID &&
		stringValue(item.DisplayName) == identity.displayName
}

func buildPipelineDeleteResolutionFromMatches(
	identity buildPipelineDeleteListIdentity,
	matches []devopssdk.BuildPipelineSummary,
) (devopssdk.BuildPipeline, bool, error) {
	switch len(matches) {
	case 0:
		return devopssdk.BuildPipeline{}, false, nil
	case 1:
		return buildPipelineFromSummary(matches[0]), true, nil
	default:
		return devopssdk.BuildPipeline{}, false, fmt.Errorf("BuildPipeline delete confirmation found %d build pipelines matching projectId %q and displayName %q", len(matches), identity.projectID, identity.displayName)
	}
}

func markBuildPipelineDeleted(resource *devopsv1beta1.BuildPipeline, message string) {
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

func handleBuildPipelineDeleteError(resource *devopsv1beta1.BuildPipeline, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return buildPipelineAmbiguousNotFoundError{
		HTTPStatusCode: classification.HTTPStatusCode,
		ErrorCode:      buildPipelineAmbiguousNotFoundErrorCode,
		OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
		message: fmt.Sprintf(
			"BuildPipeline delete returned ambiguous not-found response (HTTP %s, code %s); retaining finalizer until OCI deletion is confirmed",
			classification.HTTPStatusCodeString(),
			classification.ErrorCodeString(),
		),
	}
}

func conservativeBuildPipelineNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("BuildPipeline %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return buildPipelineAmbiguousNotFoundError{
			HTTPStatusCode: serviceErr.GetHTTPStatusCode(),
			ErrorCode:      buildPipelineAmbiguousNotFoundErrorCode,
			OpcRequestID:   serviceErr.GetOpcRequestID(),
			message:        message,
		}
	}
	return buildPipelineAmbiguousNotFoundError{ErrorCode: buildPipelineAmbiguousNotFoundErrorCode, message: message}
}

func getBuildPipelineWorkRequest(
	ctx context.Context,
	client buildPipelineOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize BuildPipeline OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("BuildPipeline OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, devopssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveBuildPipelineGeneratedWorkRequestAction(workRequest any) (string, error) {
	buildPipelineWorkRequest, err := buildPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveBuildPipelineWorkRequestAction(buildPipelineWorkRequest)
}

func resolveBuildPipelineGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	buildPipelineWorkRequest, err := buildPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := buildPipelineWorkRequestPhaseFromOperationType(buildPipelineWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverBuildPipelineIDFromGeneratedWorkRequest(
	_ *devopsv1beta1.BuildPipeline,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	buildPipelineWorkRequest, err := buildPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveBuildPipelineIDFromWorkRequest(buildPipelineWorkRequest, buildPipelineWorkRequestActionForPhase(phase))
}

func resolveBuildPipelineWorkRequestAction(workRequest devopssdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isBuildPipelineWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || strings.EqualFold(candidate, string(devopssdk.ActionTypeInProgress)) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("BuildPipeline work request %s exposes conflicting BuildPipeline action types %q and %q", stringValue(workRequest.Id), action, candidate)
		}
	}
	return action, nil
}

func buildPipelineWorkRequestPhaseFromOperationType(operationType devopssdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case devopssdk.OperationTypeCreateBuildPipeline:
		return shared.OSOKAsyncPhaseCreate, true
	case devopssdk.OperationTypeUpdateBuildPipeline:
		return shared.OSOKAsyncPhaseUpdate, true
	case devopssdk.OperationTypeDeleteBuildPipeline:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func buildPipelineWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) devopssdk.ActionTypeEnum {
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

func buildPipelineGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	buildPipelineWorkRequest, err := buildPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("BuildPipeline %s work request %s is %s", phase, stringValue(buildPipelineWorkRequest.Id), buildPipelineWorkRequest.Status)
}

func buildPipelineWorkRequestFromAny(workRequest any) (devopssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case devopssdk.WorkRequest:
		return current, nil
	case *devopssdk.WorkRequest:
		if current == nil {
			return devopssdk.WorkRequest{}, fmt.Errorf("BuildPipeline work request is nil")
		}
		return *current, nil
	default:
		return devopssdk.WorkRequest{}, fmt.Errorf("unexpected BuildPipeline work request type %T", workRequest)
	}
}

func resolveBuildPipelineIDFromWorkRequest(workRequest devopssdk.WorkRequest, action devopssdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveBuildPipelineIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveBuildPipelineIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("BuildPipeline work request %s does not expose a BuildPipeline identifier", stringValue(workRequest.Id))
}

func resolveBuildPipelineIDFromResources(
	resources []devopssdk.WorkRequestResource,
	action devopssdk.ActionTypeEnum,
	preferBuildPipelineOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferBuildPipelineOnly && !isBuildPipelineWorkRequestResource(resource) {
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

func isBuildPipelineWorkRequestResource(resource devopssdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	if entityType != "" {
		normalized := strings.NewReplacer("_", "", "-", "", " ", "").Replace(entityType)
		if strings.Contains(normalized, "buildpipelinestage") {
			return false
		}
		switch normalized {
		case "buildpipeline", "devopsbuildpipeline":
			return true
		}
		if strings.Contains(normalized, "buildpipeline") && !strings.Contains(normalized, "stage") {
			return true
		}
	}

	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	if entityURI == "" || strings.Contains(entityURI, "buildpipelinestages") || strings.Contains(entityURI, "/stages/") {
		return false
	}
	return strings.Contains(entityURI, "/buildpipelines/")
}

func listBuildPipelinesAllPages(
	ctx context.Context,
	client buildPipelineOCIClient,
	initErr error,
	request devopssdk.ListBuildPipelinesRequest,
) (devopssdk.ListBuildPipelinesResponse, error) {
	if initErr != nil {
		return devopssdk.ListBuildPipelinesResponse{}, fmt.Errorf("initialize BuildPipeline OCI client: %w", initErr)
	}
	if client == nil {
		return devopssdk.ListBuildPipelinesResponse{}, fmt.Errorf("BuildPipeline OCI client is not configured")
	}

	var combined devopssdk.ListBuildPipelinesResponse
	for {
		response, err := client.ListBuildPipelines(ctx, request)
		if err != nil {
			return devopssdk.ListBuildPipelinesResponse{}, conservativeBuildPipelineNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == devopssdk.BuildPipelineLifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		page := nextPage(response.OpcNextPage)
		if page == nil {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = page
		combined.OpcNextPage = page
	}
}

func nextPage(page *string) *string {
	if strings.TrimSpace(stringValue(page)) == "" {
		return nil
	}
	return page
}

func buildPipelineFromResponse(response any) (devopssdk.BuildPipeline, bool) {
	if buildPipeline, ok := buildPipelineFromResponseBody(response); ok {
		return buildPipeline, true
	}
	return buildPipelineFromSummaryResponse(response)
}

func buildPipelineFromResponseBody(response any) (devopssdk.BuildPipeline, bool) {
	switch current := response.(type) {
	case devopssdk.CreateBuildPipelineResponse:
		return current.BuildPipeline, true
	case *devopssdk.CreateBuildPipelineResponse:
		return buildPipelineFromCreateResponsePointer(current)
	case devopssdk.GetBuildPipelineResponse:
		return current.BuildPipeline, true
	case *devopssdk.GetBuildPipelineResponse:
		return buildPipelineFromGetResponsePointer(current)
	case devopssdk.UpdateBuildPipelineResponse:
		return current.BuildPipeline, true
	case *devopssdk.UpdateBuildPipelineResponse:
		return buildPipelineFromUpdateResponsePointer(current)
	case devopssdk.BuildPipeline:
		return current, true
	case *devopssdk.BuildPipeline:
		return buildPipelineFromPointer(current)
	default:
		return devopssdk.BuildPipeline{}, false
	}
}

func buildPipelineFromSummaryResponse(response any) (devopssdk.BuildPipeline, bool) {
	switch current := response.(type) {
	case devopssdk.BuildPipelineSummary:
		return buildPipelineFromSummary(current), true
	case *devopssdk.BuildPipelineSummary:
		if current == nil {
			return devopssdk.BuildPipeline{}, false
		}
		return buildPipelineFromSummary(*current), true
	default:
		return devopssdk.BuildPipeline{}, false
	}
}

func buildPipelineFromCreateResponsePointer(current *devopssdk.CreateBuildPipelineResponse) (devopssdk.BuildPipeline, bool) {
	if current == nil {
		return devopssdk.BuildPipeline{}, false
	}
	return current.BuildPipeline, true
}

func buildPipelineFromGetResponsePointer(current *devopssdk.GetBuildPipelineResponse) (devopssdk.BuildPipeline, bool) {
	if current == nil {
		return devopssdk.BuildPipeline{}, false
	}
	return current.BuildPipeline, true
}

func buildPipelineFromUpdateResponsePointer(current *devopssdk.UpdateBuildPipelineResponse) (devopssdk.BuildPipeline, bool) {
	if current == nil {
		return devopssdk.BuildPipeline{}, false
	}
	return current.BuildPipeline, true
}

func buildPipelineFromPointer(current *devopssdk.BuildPipeline) (devopssdk.BuildPipeline, bool) {
	if current == nil {
		return devopssdk.BuildPipeline{}, false
	}
	return *current, true
}

func buildPipelineFromSummary(summary devopssdk.BuildPipelineSummary) devopssdk.BuildPipeline {
	return devopssdk.BuildPipeline{
		Id:                      summary.Id,
		CompartmentId:           summary.CompartmentId,
		ProjectId:               summary.ProjectId,
		Description:             summary.Description,
		DisplayName:             summary.DisplayName,
		TimeCreated:             summary.TimeCreated,
		TimeUpdated:             summary.TimeUpdated,
		LifecycleState:          summary.LifecycleState,
		LifecycleDetails:        summary.LifecycleDetails,
		BuildPipelineParameters: cloneBuildPipelineParameterCollection(summary.BuildPipelineParameters),
		FreeformTags:            cloneStringMap(summary.FreeformTags),
		DefinedTags:             cloneDefinedTags(summary.DefinedTags),
		SystemTags:              cloneDefinedTags(summary.SystemTags),
	}
}

func cloneBuildPipelineParameterCollection(
	input *devopssdk.BuildPipelineParameterCollection,
) *devopssdk.BuildPipelineParameterCollection {
	if input == nil {
		return nil
	}
	items := make([]devopssdk.BuildPipelineParameter, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, devopssdk.BuildPipelineParameter{
			Name:         cloneStringPtr(item.Name),
			DefaultValue: cloneStringPtr(item.DefaultValue),
			Description:  cloneStringPtr(item.Description),
		})
	}
	return &devopssdk.BuildPipelineParameterCollection{Items: items}
}

func cloneStringPtr(input *string) *string {
	if input == nil {
		return nil
	}
	return common.String(*input)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrEqual(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
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

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
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
