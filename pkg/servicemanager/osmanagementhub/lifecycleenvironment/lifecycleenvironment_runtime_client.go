/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lifecycleenvironment

import (
	"context"
	"errors"
	"fmt"
	"maps"
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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type lifecycleEnvironmentOCIClient interface {
	CreateLifecycleEnvironment(context.Context, osmanagementhubsdk.CreateLifecycleEnvironmentRequest) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error)
	GetLifecycleEnvironment(context.Context, osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error)
	ListLifecycleEnvironments(context.Context, osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error)
	UpdateLifecycleEnvironment(context.Context, osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error)
	DeleteLifecycleEnvironment(context.Context, osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error)
}

type lifecycleEnvironmentRuntimeClient struct {
	delegate LifecycleEnvironmentServiceClient
	hooks    LifecycleEnvironmentRuntimeHooks
}

type lifecycleEnvironmentDeleteIdentity struct {
	compartmentID string
	displayName   string
	archType      string
	osFamily      string
	vendorName    string
	location      string
}

type lifecycleEnvironmentAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e lifecycleEnvironmentAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e lifecycleEnvironmentAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerLifecycleEnvironmentRuntimeHooksMutator(func(_ *LifecycleEnvironmentServiceManager, hooks *LifecycleEnvironmentRuntimeHooks) {
		applyLifecycleEnvironmentRuntimeHooks(hooks)
	})
}

func applyLifecycleEnvironmentRuntimeHooks(hooks *LifecycleEnvironmentRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newLifecycleEnvironmentRuntimeSemantics()
	hooks.BuildCreateBody = buildLifecycleEnvironmentCreateBody
	hooks.BuildUpdateBody = buildLifecycleEnvironmentUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardLifecycleEnvironmentExistingBeforeCreate
	hooks.List.Fields = lifecycleEnvironmentListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedLifecycleEnvironmentIdentity
	hooks.ParityHooks.NormalizeDesiredState = normalizeLifecycleEnvironmentDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateLifecycleEnvironmentCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleLifecycleEnvironmentDeleteError
	wrapLifecycleEnvironmentReadListAndDeleteCalls(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LifecycleEnvironmentServiceClient) LifecycleEnvironmentServiceClient {
		return &lifecycleEnvironmentRuntimeClient{delegate: delegate, hooks: *hooks}
	})
}

func newLifecycleEnvironmentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client lifecycleEnvironmentOCIClient,
) LifecycleEnvironmentServiceClient {
	hooks := newLifecycleEnvironmentRuntimeHooksWithOCIClient(client)
	applyLifecycleEnvironmentRuntimeHooks(&hooks)
	manager := &LifecycleEnvironmentServiceManager{Log: log}
	delegate := defaultLifecycleEnvironmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.LifecycleEnvironment](
			buildLifecycleEnvironmentGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapLifecycleEnvironmentGeneratedClient(hooks, delegate)
}

func newLifecycleEnvironmentRuntimeHooksWithOCIClient(client lifecycleEnvironmentOCIClient) LifecycleEnvironmentRuntimeHooks {
	return LifecycleEnvironmentRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*osmanagementhubv1beta1.LifecycleEnvironment]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*osmanagementhubv1beta1.LifecycleEnvironment]{},
		StatusHooks:     generatedruntime.StatusHooks[*osmanagementhubv1beta1.LifecycleEnvironment]{},
		ParityHooks:     generatedruntime.ParityHooks[*osmanagementhubv1beta1.LifecycleEnvironment]{},
		Async:           generatedruntime.AsyncHooks[*osmanagementhubv1beta1.LifecycleEnvironment]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*osmanagementhubv1beta1.LifecycleEnvironment]{},
		Create: runtimeOperationHooks[osmanagementhubsdk.CreateLifecycleEnvironmentRequest, osmanagementhubsdk.CreateLifecycleEnvironmentResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateLifecycleEnvironmentDetails", RequestName: "CreateLifecycleEnvironmentDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.CreateLifecycleEnvironmentRequest) (osmanagementhubsdk.CreateLifecycleEnvironmentResponse, error) {
				if client == nil {
					return osmanagementhubsdk.CreateLifecycleEnvironmentResponse{}, fmt.Errorf("lifecycle environment OCI client is nil")
				}
				return client.CreateLifecycleEnvironment(ctx, request)
			},
		},
		Get: runtimeOperationHooks[osmanagementhubsdk.GetLifecycleEnvironmentRequest, osmanagementhubsdk.GetLifecycleEnvironmentResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LifecycleEnvironmentId", RequestName: "lifecycleEnvironmentId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
				if client == nil {
					return osmanagementhubsdk.GetLifecycleEnvironmentResponse{}, fmt.Errorf("lifecycle environment OCI client is nil")
				}
				return client.GetLifecycleEnvironment(ctx, request)
			},
		},
		List: runtimeOperationHooks[osmanagementhubsdk.ListLifecycleEnvironmentsRequest, osmanagementhubsdk.ListLifecycleEnvironmentsResponse]{
			Fields: lifecycleEnvironmentListFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
				if client == nil {
					return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{}, fmt.Errorf("lifecycle environment OCI client is nil")
				}
				return client.ListLifecycleEnvironments(ctx, request)
			},
		},
		Update: runtimeOperationHooks[osmanagementhubsdk.UpdateLifecycleEnvironmentRequest, osmanagementhubsdk.UpdateLifecycleEnvironmentResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "LifecycleEnvironmentId", RequestName: "lifecycleEnvironmentId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateLifecycleEnvironmentDetails", RequestName: "UpdateLifecycleEnvironmentDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request osmanagementhubsdk.UpdateLifecycleEnvironmentRequest) (osmanagementhubsdk.UpdateLifecycleEnvironmentResponse, error) {
				if client == nil {
					return osmanagementhubsdk.UpdateLifecycleEnvironmentResponse{}, fmt.Errorf("lifecycle environment OCI client is nil")
				}
				return client.UpdateLifecycleEnvironment(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[osmanagementhubsdk.DeleteLifecycleEnvironmentRequest, osmanagementhubsdk.DeleteLifecycleEnvironmentResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LifecycleEnvironmentId", RequestName: "lifecycleEnvironmentId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
				if client == nil {
					return osmanagementhubsdk.DeleteLifecycleEnvironmentResponse{}, fmt.Errorf("lifecycle environment OCI client is nil")
				}
				return client.DeleteLifecycleEnvironment(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LifecycleEnvironmentServiceClient) LifecycleEnvironmentServiceClient{},
	}
}

func newLifecycleEnvironmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "osmanagementhub",
		FormalSlug:        "lifecycleenvironment",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateCreating)},
			UpdatingStates:     []string{string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateUpdating)},
			ActiveStates:       []string{string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateCreating),
				string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateUpdating),
				string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateDeleting),
			},
			TerminalStates: []string{string(osmanagementhubsdk.LifecycleEnvironmentLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"archType",
				"osFamily",
				"vendorName",
				"location",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "description", "stages", "freeformTags", "definedTags"},
			Mutable:         []string{"displayName", "description", "stages", "freeformTags", "definedTags"},
			ForceNew:        []string{"compartmentId", "archType", "osFamily", "vendorName", "location"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "LifecycleEnvironment", Action: "CreateLifecycleEnvironment"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "LifecycleEnvironment", Action: "UpdateLifecycleEnvironment"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "LifecycleEnvironment", Action: "DeleteLifecycleEnvironment"}},
		},
		CreateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func lifecycleEnvironmentListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "LifecycleEnvironmentId", RequestName: "lifecycleEnvironmentId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func guardLifecycleEnvironmentExistingBeforeCreate(
	_ context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("lifecycle environment resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildLifecycleEnvironmentCreateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("lifecycle environment resource is nil")
	}
	if err := validateLifecycleEnvironmentSpec(resource.Spec); err != nil {
		return nil, err
	}

	details := osmanagementhubsdk.CreateLifecycleEnvironmentDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		Stages:        createLifecycleStageDetails(resource.Spec.Stages),
		ArchType:      osmanagementhubsdk.ArchTypeEnum(strings.TrimSpace(resource.Spec.ArchType)),
		OsFamily:      osmanagementhubsdk.OsFamilyEnum(strings.TrimSpace(resource.Spec.OsFamily)),
		VendorName:    osmanagementhubsdk.VendorNameEnum(strings.TrimSpace(resource.Spec.VendorName)),
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		details.Description = common.String(description)
	}
	details.Location = osmanagementhubsdk.ManagedInstanceLocationEnum(desiredLifecycleEnvironmentLocation(resource.Spec))
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildLifecycleEnvironmentUpdateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	_ string,
	currentResponse any,
) (any, bool, error) {
	current, err := lifecycleEnvironmentCurrentForUpdate(resource, currentResponse)
	if err != nil {
		return osmanagementhubsdk.UpdateLifecycleEnvironmentDetails{}, false, err
	}
	details, updateNeeded, err := lifecycleEnvironmentUpdateDetails(resource.Spec, current)
	if err != nil {
		return osmanagementhubsdk.UpdateLifecycleEnvironmentDetails{}, false, err
	}
	return details, updateNeeded, nil
}

func lifecycleEnvironmentCurrentForUpdate(
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	currentResponse any,
) (osmanagementhubsdk.LifecycleEnvironment, error) {
	if resource == nil {
		return osmanagementhubsdk.LifecycleEnvironment{}, fmt.Errorf("lifecycle environment resource is nil")
	}
	if err := validateLifecycleEnvironmentSpec(resource.Spec); err != nil {
		return osmanagementhubsdk.LifecycleEnvironment{}, err
	}
	current, ok := lifecycleEnvironmentFromResponse(currentResponse)
	if !ok {
		return osmanagementhubsdk.LifecycleEnvironment{}, fmt.Errorf("current lifecycle environment response does not expose a lifecycle environment body")
	}
	if err := validateLifecycleEnvironmentCreateOnlyDrift(resource.Spec, current); err != nil {
		return osmanagementhubsdk.LifecycleEnvironment{}, err
	}
	return current, nil
}

func lifecycleEnvironmentUpdateDetails(
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	current osmanagementhubsdk.LifecycleEnvironment,
) (osmanagementhubsdk.UpdateLifecycleEnvironmentDetails, bool, error) {
	details := osmanagementhubsdk.UpdateLifecycleEnvironmentDetails{}
	updateNeeded := false
	updateNeeded = applyLifecycleEnvironmentDisplayNameUpdate(&details, spec, current) || updateNeeded
	updateNeeded = applyLifecycleEnvironmentDescriptionUpdate(&details, spec, current) || updateNeeded
	updateNeeded = applyLifecycleEnvironmentFreeformTagsUpdate(&details, spec, current) || updateNeeded
	updateNeeded = applyLifecycleEnvironmentDefinedTagsUpdate(&details, spec, current) || updateNeeded
	stages, stagesUpdated, err := lifecycleEnvironmentStageUpdates(spec.Stages, current.Stages)
	if err != nil {
		return osmanagementhubsdk.UpdateLifecycleEnvironmentDetails{}, false, err
	}
	if stagesUpdated {
		details.Stages = stages
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func validateLifecycleEnvironmentSpec(spec osmanagementhubv1beta1.LifecycleEnvironmentSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if len(spec.Stages) == 0 {
		missing = append(missing, "stages")
	}
	if strings.TrimSpace(spec.ArchType) == "" {
		missing = append(missing, "archType")
	}
	if strings.TrimSpace(spec.OsFamily) == "" {
		missing = append(missing, "osFamily")
	}
	if strings.TrimSpace(spec.VendorName) == "" {
		missing = append(missing, "vendorName")
	}
	if len(missing) != 0 {
		return fmt.Errorf("lifecycle environment spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return validateLifecycleEnvironmentStageSpec(spec.Stages)
}

func validateLifecycleEnvironmentStageSpec(stages []osmanagementhubv1beta1.LifecycleEnvironmentStage) error {
	seenRanks := make(map[int]struct{}, len(stages))
	for index, stage := range stages {
		if strings.TrimSpace(stage.DisplayName) == "" {
			return fmt.Errorf("lifecycle environment spec stage %d is missing required field displayName", index)
		}
		if _, ok := seenRanks[stage.Rank]; ok {
			return fmt.Errorf("lifecycle environment spec has duplicate stage rank %d", stage.Rank)
		}
		seenRanks[stage.Rank] = struct{}{}
	}
	return nil
}

func createLifecycleStageDetails(stages []osmanagementhubv1beta1.LifecycleEnvironmentStage) []osmanagementhubsdk.CreateLifecycleStageDetails {
	details := make([]osmanagementhubsdk.CreateLifecycleStageDetails, 0, len(stages))
	for _, stage := range stages {
		next := osmanagementhubsdk.CreateLifecycleStageDetails{
			DisplayName: common.String(strings.TrimSpace(stage.DisplayName)),
			Rank:        common.Int(stage.Rank),
		}
		if stage.FreeformTags != nil {
			next.FreeformTags = maps.Clone(stage.FreeformTags)
		}
		if stage.DefinedTags != nil {
			next.DefinedTags = *util.ConvertToOciDefinedTags(&stage.DefinedTags)
		}
		details = append(details, next)
	}
	return details
}

func applyLifecycleEnvironmentDisplayNameUpdate(
	details *osmanagementhubsdk.UpdateLifecycleEnvironmentDetails,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	current osmanagementhubsdk.LifecycleEnvironment,
) bool {
	desired := strings.TrimSpace(spec.DisplayName)
	if desired == "" || desired == stringPointerValue(current.DisplayName) {
		return false
	}
	details.DisplayName = common.String(desired)
	return true
}

func applyLifecycleEnvironmentDescriptionUpdate(
	details *osmanagementhubsdk.UpdateLifecycleEnvironmentDetails,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	current osmanagementhubsdk.LifecycleEnvironment,
) bool {
	desired := strings.TrimSpace(spec.Description)
	if desired == stringPointerValue(current.Description) {
		return false
	}
	details.Description = common.String(desired)
	return true
}

func applyLifecycleEnvironmentFreeformTagsUpdate(
	details *osmanagementhubsdk.UpdateLifecycleEnvironmentDetails,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	current osmanagementhubsdk.LifecycleEnvironment,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	details.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func applyLifecycleEnvironmentDefinedTagsUpdate(
	details *osmanagementhubsdk.UpdateLifecycleEnvironmentDetails,
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	current osmanagementhubsdk.LifecycleEnvironment,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	if reflect.DeepEqual(desired, current.DefinedTags) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func lifecycleEnvironmentStageUpdates(
	specStages []osmanagementhubv1beta1.LifecycleEnvironmentStage,
	currentStages []osmanagementhubsdk.LifecycleStage,
) ([]osmanagementhubsdk.UpdateLifecycleStageDetails, bool, error) {
	currentByRank, err := lifecycleEnvironmentStagesByRank(currentStages)
	if err != nil {
		return nil, false, err
	}
	updates := make([]osmanagementhubsdk.UpdateLifecycleStageDetails, 0, len(specStages))
	for _, desired := range specStages {
		current := currentByRank[desired.Rank]
		update, updated, err := lifecycleEnvironmentStageUpdate(desired, current)
		if err != nil {
			return nil, false, err
		}
		if updated {
			updates = append(updates, update)
		}
	}
	return updates, len(updates) != 0, nil
}

func lifecycleEnvironmentStageUpdate(
	desired osmanagementhubv1beta1.LifecycleEnvironmentStage,
	current osmanagementhubsdk.LifecycleStage,
) (osmanagementhubsdk.UpdateLifecycleStageDetails, bool, error) {
	update := osmanagementhubsdk.UpdateLifecycleStageDetails{}
	updated := false
	if displayName := strings.TrimSpace(desired.DisplayName); displayName != "" && displayName != stringPointerValue(current.DisplayName) {
		update.DisplayName = common.String(displayName)
		updated = true
	}
	if desired.FreeformTags != nil && !maps.Equal(desired.FreeformTags, current.FreeformTags) {
		update.FreeformTags = maps.Clone(desired.FreeformTags)
		updated = true
	}
	if desired.DefinedTags != nil {
		definedTags := *util.ConvertToOciDefinedTags(&desired.DefinedTags)
		if !reflect.DeepEqual(definedTags, current.DefinedTags) {
			update.DefinedTags = definedTags
			updated = true
		}
	}
	if !updated {
		return osmanagementhubsdk.UpdateLifecycleStageDetails{}, false, nil
	}
	stageID := stringPointerValue(current.Id)
	if stageID == "" {
		return osmanagementhubsdk.UpdateLifecycleStageDetails{}, false, fmt.Errorf("lifecycle environment stage rank %d cannot be updated because OCI readback omitted its id", desired.Rank)
	}
	update.Id = common.String(stageID)
	return update, true, nil
}

func lifecycleEnvironmentStagesByRank(stages []osmanagementhubsdk.LifecycleStage) (map[int]osmanagementhubsdk.LifecycleStage, error) {
	byRank := make(map[int]osmanagementhubsdk.LifecycleStage, len(stages))
	for _, stage := range stages {
		if stage.Rank == nil {
			return nil, fmt.Errorf("lifecycle environment readback contains a stage without rank")
		}
		rank := *stage.Rank
		if _, ok := byRank[rank]; ok {
			return nil, fmt.Errorf("lifecycle environment readback contains duplicate stage rank %d", rank)
		}
		byRank[rank] = stage
	}
	return byRank, nil
}

func validateLifecycleEnvironmentCreateOnlyDriftForResponse(
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("lifecycle environment resource is nil")
	}
	current, ok := lifecycleEnvironmentFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current lifecycle environment response does not expose a lifecycle environment body")
	}
	return validateLifecycleEnvironmentCreateOnlyDrift(resource.Spec, current)
}

func validateLifecycleEnvironmentCreateOnlyDrift(
	spec osmanagementhubv1beta1.LifecycleEnvironmentSpec,
	current osmanagementhubsdk.LifecycleEnvironment,
) error {
	var drift []string
	if strings.TrimSpace(spec.CompartmentId) != stringPointerValue(current.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if strings.TrimSpace(spec.ArchType) != strings.TrimSpace(string(current.ArchType)) {
		drift = append(drift, "archType")
	}
	if strings.TrimSpace(spec.OsFamily) != strings.TrimSpace(string(current.OsFamily)) {
		drift = append(drift, "osFamily")
	}
	if strings.TrimSpace(spec.VendorName) != strings.TrimSpace(string(current.VendorName)) {
		drift = append(drift, "vendorName")
	}
	if desiredLocation := desiredLifecycleEnvironmentLocation(spec); desiredLocation != strings.TrimSpace(string(current.Location)) {
		drift = append(drift, "location")
	}
	if stageDrift := lifecycleEnvironmentCreateOnlyStageDrift(spec.Stages, current.Stages); stageDrift != "" {
		drift = append(drift, stageDrift)
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("lifecycle environment create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func lifecycleEnvironmentCreateOnlyStageDrift(
	specStages []osmanagementhubv1beta1.LifecycleEnvironmentStage,
	currentStages []osmanagementhubsdk.LifecycleStage,
) string {
	if len(specStages) != len(currentStages) {
		return "stages"
	}
	currentByRank, err := lifecycleEnvironmentStagesByRank(currentStages)
	if err != nil {
		return "stages.rank"
	}
	for _, desired := range specStages {
		if _, ok := currentByRank[desired.Rank]; !ok {
			return "stages.rank"
		}
	}
	return ""
}

func clearTrackedLifecycleEnvironmentIdentity(resource *osmanagementhubv1beta1.LifecycleEnvironment) {
	if resource == nil {
		return
	}
	resource.Status = osmanagementhubv1beta1.LifecycleEnvironmentStatus{}
}

func normalizeLifecycleEnvironmentDesiredState(
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	_ any,
) {
	normalizeLifecycleEnvironmentSpecDefaults(resource)
}

func normalizeLifecycleEnvironmentSpecDefaults(resource *osmanagementhubv1beta1.LifecycleEnvironment) {
	if resource == nil {
		return
	}
	if strings.TrimSpace(resource.Spec.Location) == "" {
		resource.Spec.Location = string(osmanagementhubsdk.ManagedInstanceLocationOnPremise)
	}
}

func desiredLifecycleEnvironmentLocation(spec osmanagementhubv1beta1.LifecycleEnvironmentSpec) string {
	if location := strings.TrimSpace(spec.Location); location != "" {
		return location
	}
	return string(osmanagementhubsdk.ManagedInstanceLocationOnPremise)
}

func (c *lifecycleEnvironmentRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("lifecycle environment runtime client is not configured")
	}
	normalizeLifecycleEnvironmentSpecDefaults(resource)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *lifecycleEnvironmentRuntimeClient) Delete(
	ctx context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("lifecycle environment runtime client is not configured")
	}
	normalizeLifecycleEnvironmentSpecDefaults(resource)
	deleteID, found, err := c.resolveLifecycleEnvironmentDeleteID(ctx, resource)
	if err != nil {
		return false, err
	}
	if !found {
		markLifecycleEnvironmentDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource, deleteID); err != nil {
		return false, err
	}
	recordResolvedLifecycleEnvironmentID(resource, deleteID)
	return c.delegate.Delete(ctx, resource)
}

func (c *lifecycleEnvironmentRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	deleteID string,
) error {
	currentID := strings.TrimSpace(deleteID)
	if currentID == "" {
		currentID = currentLifecycleEnvironmentID(resource)
	}
	if currentID == "" || c.hooks.Get.Call == nil {
		return nil
	}
	_, err := c.hooks.Get.Call(ctx, osmanagementhubsdk.GetLifecycleEnvironmentRequest{
		LifecycleEnvironmentId: common.String(currentID),
	})
	if err == nil || (!isLifecycleEnvironmentAmbiguousNotFound(err) && !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()) {
		return nil
	}
	err = conservativeLifecycleEnvironmentNotFoundError(err, "delete confirmation")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("lifecycle environment delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
}

func (c *lifecycleEnvironmentRuntimeClient) resolveLifecycleEnvironmentDeleteID(
	ctx context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
) (string, bool, error) {
	if currentID := currentLifecycleEnvironmentID(resource); currentID != "" {
		return currentID, true, nil
	}
	summary, found, err := c.resolveLifecycleEnvironmentDeleteSummaryByList(ctx, resource)
	if err != nil || !found {
		return "", found, err
	}
	resolvedID := stringPointerValue(summary.Id)
	if resolvedID == "" {
		return "", false, fmt.Errorf("lifecycle environment delete confirmation could not resolve a resource OCID")
	}
	return resolvedID, true, nil
}

func (c *lifecycleEnvironmentRuntimeClient) resolveLifecycleEnvironmentDeleteSummaryByList(
	ctx context.Context,
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
) (osmanagementhubsdk.LifecycleEnvironmentSummary, bool, error) {
	if c.hooks.List.Call == nil {
		return osmanagementhubsdk.LifecycleEnvironmentSummary{}, false, nil
	}
	identity, err := lifecycleEnvironmentDeleteIdentityForList(resource)
	if err != nil {
		return osmanagementhubsdk.LifecycleEnvironmentSummary{}, false, err
	}
	response, err := c.hooks.List.Call(ctx, osmanagementhubsdk.ListLifecycleEnvironmentsRequest{
		CompartmentId: common.String(identity.compartmentID),
	})
	if err != nil {
		return osmanagementhubsdk.LifecycleEnvironmentSummary{}, false, err
	}

	matches := make([]osmanagementhubsdk.LifecycleEnvironmentSummary, 0, 1)
	for _, item := range response.Items {
		if lifecycleEnvironmentSummaryMatchesDeleteIdentity(item, identity) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return osmanagementhubsdk.LifecycleEnvironmentSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return osmanagementhubsdk.LifecycleEnvironmentSummary{}, false, fmt.Errorf(
			"lifecycle environment delete confirmation found multiple matches for compartmentId %q, displayName %q",
			identity.compartmentID,
			identity.displayName,
		)
	}
}

func lifecycleEnvironmentDeleteIdentityForList(
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
) (lifecycleEnvironmentDeleteIdentity, error) {
	if resource == nil {
		return lifecycleEnvironmentDeleteIdentity{}, fmt.Errorf("lifecycle environment delete confirmation requires a resource")
	}
	identity := lifecycleEnvironmentDeleteIdentity{
		compartmentID: firstLifecycleEnvironmentValue(resource.Status.CompartmentId, resource.Spec.CompartmentId),
		displayName:   firstLifecycleEnvironmentValue(resource.Status.DisplayName, resource.Spec.DisplayName),
		archType:      firstLifecycleEnvironmentValue(resource.Status.ArchType, resource.Spec.ArchType),
		osFamily:      firstLifecycleEnvironmentValue(resource.Status.OsFamily, resource.Spec.OsFamily),
		vendorName:    firstLifecycleEnvironmentValue(resource.Status.VendorName, resource.Spec.VendorName),
		location:      firstLifecycleEnvironmentValue(resource.Status.Location, desiredLifecycleEnvironmentLocation(resource.Spec)),
	}
	var missing []string
	if identity.compartmentID == "" {
		missing = append(missing, "compartmentId")
	}
	if identity.displayName == "" {
		missing = append(missing, "displayName")
	}
	if identity.archType == "" {
		missing = append(missing, "archType")
	}
	if identity.osFamily == "" {
		missing = append(missing, "osFamily")
	}
	if identity.vendorName == "" {
		missing = append(missing, "vendorName")
	}
	if identity.location == "" {
		missing = append(missing, "location")
	}
	if len(missing) != 0 {
		return lifecycleEnvironmentDeleteIdentity{}, fmt.Errorf("lifecycle environment delete confirmation missing identity field(s): %s", strings.Join(missing, ", "))
	}
	return identity, nil
}

func lifecycleEnvironmentSummaryMatchesDeleteIdentity(
	summary osmanagementhubsdk.LifecycleEnvironmentSummary,
	identity lifecycleEnvironmentDeleteIdentity,
) bool {
	return stringPointerValue(summary.CompartmentId) == identity.compartmentID &&
		stringPointerValue(summary.DisplayName) == identity.displayName &&
		strings.TrimSpace(string(summary.ArchType)) == identity.archType &&
		strings.TrimSpace(string(summary.OsFamily)) == identity.osFamily &&
		strings.TrimSpace(string(summary.VendorName)) == identity.vendorName &&
		lifecycleEnvironmentSummaryLocation(summary) == identity.location
}

func lifecycleEnvironmentSummaryLocation(summary osmanagementhubsdk.LifecycleEnvironmentSummary) string {
	if location := strings.TrimSpace(string(summary.Location)); location != "" {
		return location
	}
	return string(osmanagementhubsdk.ManagedInstanceLocationOnPremise)
}

func firstLifecycleEnvironmentValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func recordResolvedLifecycleEnvironmentID(resource *osmanagementhubv1beta1.LifecycleEnvironment, id string) {
	if resource == nil || strings.TrimSpace(id) == "" {
		return
	}
	resource.Status.Id = strings.TrimSpace(id)
	resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(id))
}

func markLifecycleEnvironmentDeleted(resource *osmanagementhubv1beta1.LifecycleEnvironment, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = strings.TrimSpace(message)
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", strings.TrimSpace(message), loggerutil.OSOKLogger{})
}

func currentLifecycleEnvironmentID(resource *osmanagementhubv1beta1.LifecycleEnvironment) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func wrapLifecycleEnvironmentReadListAndDeleteCalls(hooks *LifecycleEnvironmentRuntimeHooks) {
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request osmanagementhubsdk.GetLifecycleEnvironmentRequest) (osmanagementhubsdk.GetLifecycleEnvironmentResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeLifecycleEnvironmentNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
			return listLifecycleEnvironmentPages(ctx, call, request)
		}
	}
	if hooks.Delete.Call != nil {
		call := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request osmanagementhubsdk.DeleteLifecycleEnvironmentRequest) (osmanagementhubsdk.DeleteLifecycleEnvironmentResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeLifecycleEnvironmentNotFoundError(err, "delete")
		}
	}
}

func listLifecycleEnvironmentPages(
	ctx context.Context,
	call func(context.Context, osmanagementhubsdk.ListLifecycleEnvironmentsRequest) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error),
	request osmanagementhubsdk.ListLifecycleEnvironmentsRequest,
) (osmanagementhubsdk.ListLifecycleEnvironmentsResponse, error) {
	var combined osmanagementhubsdk.ListLifecycleEnvironmentsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return osmanagementhubsdk.ListLifecycleEnvironmentsResponse{}, conservativeLifecycleEnvironmentNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleLifecycleEnvironmentDeleteError(
	resource *osmanagementhubv1beta1.LifecycleEnvironment,
	err error,
) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return conservativeLifecycleEnvironmentNotFoundError(err, "delete")
}

func conservativeLifecycleEnvironmentNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if isLifecycleEnvironmentAmbiguousNotFound(err) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return lifecycleEnvironmentAmbiguousNotFoundError{
		message:      fmt.Sprintf("lifecycle environment %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func isLifecycleEnvironmentAmbiguousNotFound(err error) bool {
	var ambiguous lifecycleEnvironmentAmbiguousNotFoundError
	return errors.As(err, &ambiguous)
}

func lifecycleEnvironmentFromResponse(response any) (osmanagementhubsdk.LifecycleEnvironment, bool) {
	if current, ok := lifecycleEnvironmentFromMutationResponse(response); ok {
		return current, true
	}
	if current, ok := lifecycleEnvironmentFromReadResponse(response); ok {
		return current, true
	}
	return lifecycleEnvironmentFromSummaryResponse(response)
}

func lifecycleEnvironmentFromMutationResponse(response any) (osmanagementhubsdk.LifecycleEnvironment, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.CreateLifecycleEnvironmentResponse:
		return current.LifecycleEnvironment, true
	case *osmanagementhubsdk.CreateLifecycleEnvironmentResponse:
		if current == nil {
			return osmanagementhubsdk.LifecycleEnvironment{}, false
		}
		return current.LifecycleEnvironment, true
	case osmanagementhubsdk.UpdateLifecycleEnvironmentResponse:
		return current.LifecycleEnvironment, true
	case *osmanagementhubsdk.UpdateLifecycleEnvironmentResponse:
		if current == nil {
			return osmanagementhubsdk.LifecycleEnvironment{}, false
		}
		return current.LifecycleEnvironment, true
	default:
		return osmanagementhubsdk.LifecycleEnvironment{}, false
	}
}

func lifecycleEnvironmentFromReadResponse(response any) (osmanagementhubsdk.LifecycleEnvironment, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.GetLifecycleEnvironmentResponse:
		return current.LifecycleEnvironment, true
	case *osmanagementhubsdk.GetLifecycleEnvironmentResponse:
		if current == nil {
			return osmanagementhubsdk.LifecycleEnvironment{}, false
		}
		return current.LifecycleEnvironment, true
	case osmanagementhubsdk.LifecycleEnvironment:
		return current, true
	case *osmanagementhubsdk.LifecycleEnvironment:
		if current == nil {
			return osmanagementhubsdk.LifecycleEnvironment{}, false
		}
		return *current, true
	default:
		return osmanagementhubsdk.LifecycleEnvironment{}, false
	}
}

func lifecycleEnvironmentFromSummaryResponse(response any) (osmanagementhubsdk.LifecycleEnvironment, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.LifecycleEnvironmentSummary:
		return lifecycleEnvironmentFromSummary(current), true
	case *osmanagementhubsdk.LifecycleEnvironmentSummary:
		if current == nil {
			return osmanagementhubsdk.LifecycleEnvironment{}, false
		}
		return lifecycleEnvironmentFromSummary(*current), true
	default:
		return osmanagementhubsdk.LifecycleEnvironment{}, false
	}
}

func lifecycleEnvironmentFromSummary(summary osmanagementhubsdk.LifecycleEnvironmentSummary) osmanagementhubsdk.LifecycleEnvironment {
	return osmanagementhubsdk.LifecycleEnvironment{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		DisplayName:    summary.DisplayName,
		Description:    summary.Description,
		Stages:         lifecycleStagesFromSummary(summary.Stages),
		ArchType:       summary.ArchType,
		OsFamily:       summary.OsFamily,
		VendorName:     summary.VendorName,
		LifecycleState: summary.LifecycleState,
		Location:       summary.Location,
		TimeCreated:    summary.TimeCreated,
		TimeModified:   summary.TimeModified,
		FreeformTags:   maps.Clone(summary.FreeformTags),
		DefinedTags:    cloneDefinedTags(summary.DefinedTags),
		SystemTags:     cloneDefinedTags(summary.SystemTags),
	}
}

func lifecycleStagesFromSummary(stages []osmanagementhubsdk.LifecycleStageSummary) []osmanagementhubsdk.LifecycleStage {
	if stages == nil {
		return nil
	}
	converted := make([]osmanagementhubsdk.LifecycleStage, 0, len(stages))
	for _, stage := range stages {
		converted = append(converted, osmanagementhubsdk.LifecycleStage{
			CompartmentId:          stage.CompartmentId,
			DisplayName:            stage.DisplayName,
			Rank:                   stage.Rank,
			Id:                     stage.Id,
			LifecycleEnvironmentId: stage.LifecycleEnvironmentId,
			OsFamily:               stage.OsFamily,
			ArchType:               stage.ArchType,
			VendorName:             stage.VendorName,
			Location:               stage.Location,
			SoftwareSourceId:       stage.SoftwareSourceId,
			TimeCreated:            stage.TimeCreated,
			TimeModified:           stage.TimeModified,
			LifecycleState:         stage.LifecycleState,
			FreeformTags:           maps.Clone(stage.FreeformTags),
			DefinedTags:            cloneDefinedTags(stage.DefinedTags),
			SystemTags:             cloneDefinedTags(stage.SystemTags),
		})
	}
	return converted
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

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ LifecycleEnvironmentServiceClient = (*lifecycleEnvironmentRuntimeClient)(nil)
var _ error = lifecycleEnvironmentAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = lifecycleEnvironmentAmbiguousNotFoundError{}
