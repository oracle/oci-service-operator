/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package maskingpolicy

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
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

const maskingPolicyKind = "MaskingPolicy"

type maskingPolicyOCIClient interface {
	CreateMaskingPolicy(context.Context, datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error)
	GetMaskingPolicy(context.Context, datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error)
	ListMaskingPolicies(context.Context, datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error)
	UpdateMaskingPolicy(context.Context, datasafesdk.UpdateMaskingPolicyRequest) (datasafesdk.UpdateMaskingPolicyResponse, error)
	DeleteMaskingPolicy(context.Context, datasafesdk.DeleteMaskingPolicyRequest) (datasafesdk.DeleteMaskingPolicyResponse, error)
}

type maskingPolicyIdentity struct {
	compartmentID        string
	displayName          string
	columnSource         string
	targetID             string
	sensitiveDataModelID string
}

type maskingPolicyRuntimeClient struct {
	delegate MaskingPolicyServiceClient
	get      func(context.Context, datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error)
	list     func(context.Context, datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error)
}

type maskingPolicyAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e maskingPolicyAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e maskingPolicyAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerMaskingPolicyRuntimeHooksMutator(func(_ *MaskingPolicyServiceManager, hooks *MaskingPolicyRuntimeHooks) {
		applyMaskingPolicyRuntimeHooks(hooks)
	})
}

func applyMaskingPolicyRuntimeHooks(hooks *MaskingPolicyRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = maskingPolicyRuntimeSemantics()
	hooks.BuildCreateBody = buildMaskingPolicyCreateBody
	hooks.BuildUpdateBody = buildMaskingPolicyUpdateBody
	hooks.Identity.Resolve = resolveMaskingPolicyIdentity
	hooks.Identity.RecordPath = recordMaskingPolicyPathIdentity
	hooks.Create.Fields = maskingPolicyCreateFields()
	hooks.Get.Fields = maskingPolicyGetFields()
	hooks.List.Fields = maskingPolicyListFields()
	hooks.Update.Fields = maskingPolicyUpdateFields()
	hooks.Delete.Fields = maskingPolicyDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listMaskingPoliciesAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleMaskingPolicyDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyMaskingPolicyDeleteOutcome
	hooks.StatusHooks.ProjectStatus = projectMaskingPolicyStatus
	hooks.StatusHooks.MarkTerminating = markMaskingPolicyTerminating
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MaskingPolicyServiceClient) MaskingPolicyServiceClient {
		return maskingPolicyRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newMaskingPolicyServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client maskingPolicyOCIClient,
) MaskingPolicyServiceClient {
	hooks := newMaskingPolicyRuntimeHooksWithOCIClient(client)
	applyMaskingPolicyRuntimeHooks(&hooks)
	manager := &MaskingPolicyServiceManager{Log: log}
	delegate := defaultMaskingPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.MaskingPolicy](
			buildMaskingPolicyGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapMaskingPolicyGeneratedClient(hooks, delegate)
}

func newMaskingPolicyRuntimeHooksWithOCIClient(client maskingPolicyOCIClient) MaskingPolicyRuntimeHooks {
	return MaskingPolicyRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.MaskingPolicy]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.MaskingPolicy]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.MaskingPolicy]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.MaskingPolicy]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.MaskingPolicy]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.MaskingPolicy]{},
		Create: runtimeOperationHooks[datasafesdk.CreateMaskingPolicyRequest, datasafesdk.CreateMaskingPolicyResponse]{
			Fields: maskingPolicyCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateMaskingPolicyRequest) (datasafesdk.CreateMaskingPolicyResponse, error) {
				if client == nil {
					return datasafesdk.CreateMaskingPolicyResponse{}, fmt.Errorf("%s OCI client is nil", maskingPolicyKind)
				}
				return client.CreateMaskingPolicy(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetMaskingPolicyRequest, datasafesdk.GetMaskingPolicyResponse]{
			Fields: maskingPolicyGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetMaskingPolicyRequest) (datasafesdk.GetMaskingPolicyResponse, error) {
				if client == nil {
					return datasafesdk.GetMaskingPolicyResponse{}, fmt.Errorf("%s OCI client is nil", maskingPolicyKind)
				}
				return client.GetMaskingPolicy(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListMaskingPoliciesRequest, datasafesdk.ListMaskingPoliciesResponse]{
			Fields: maskingPolicyListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error) {
				if client == nil {
					return datasafesdk.ListMaskingPoliciesResponse{}, fmt.Errorf("%s OCI client is nil", maskingPolicyKind)
				}
				return client.ListMaskingPolicies(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datasafesdk.UpdateMaskingPolicyRequest, datasafesdk.UpdateMaskingPolicyResponse]{
			Fields: maskingPolicyUpdateFields(),
			Call: func(ctx context.Context, request datasafesdk.UpdateMaskingPolicyRequest) (datasafesdk.UpdateMaskingPolicyResponse, error) {
				if client == nil {
					return datasafesdk.UpdateMaskingPolicyResponse{}, fmt.Errorf("%s OCI client is nil", maskingPolicyKind)
				}
				return client.UpdateMaskingPolicy(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteMaskingPolicyRequest, datasafesdk.DeleteMaskingPolicyResponse]{
			Fields: maskingPolicyDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteMaskingPolicyRequest) (datasafesdk.DeleteMaskingPolicyResponse, error) {
				if client == nil {
					return datasafesdk.DeleteMaskingPolicyResponse{}, fmt.Errorf("%s OCI client is nil", maskingPolicyKind)
				}
				return client.DeleteMaskingPolicy(ctx, request)
			},
		},
		WrapGeneratedClient: []func(MaskingPolicyServiceClient) MaskingPolicyServiceClient{},
	}
}

func maskingPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "maskingpolicy",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.MaskingLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.MaskingLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.MaskingLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(datasafesdk.MaskingLifecycleStateDeleting),
			},
			TerminalStates: []string{
				string(datasafesdk.MaskingLifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"columnSource.columnSource",
				"columnSource.targetId",
				"columnSource.sensitiveDataModelId",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"isDropTempTablesEnabled",
				"isRedoLoggingEnabled",
				"isRefreshStatsEnabled",
				"parallelDegree",
				"recompile",
				"preMaskingScript",
				"postMaskingScript",
				"columnSource",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: maskingPolicyKind, Action: "CreateMaskingPolicy"}},
			Update: []generatedruntime.Hook{{Helper: "generatedruntime.Update", EntityType: maskingPolicyKind, Action: "UpdateMaskingPolicy"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: maskingPolicyKind, Action: "DeleteMaskingPolicy"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func maskingPolicyCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateMaskingPolicyDetails", RequestName: "CreateMaskingPolicyDetails", Contribution: "body"},
	}
}

func maskingPolicyGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MaskingPolicyId", RequestName: "maskingPolicyId", Contribution: "path", PreferResourceID: true},
	}
}

func maskingPolicyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{
			FieldName:    "SensitiveDataModelId",
			RequestName:  "sensitiveDataModelId",
			Contribution: "query",
			LookupPaths: []string{
				"status.columnSource.sensitiveDataModelId",
				"spec.columnSource.sensitiveDataModelId",
				"columnSource.sensitiveDataModelId",
			},
		},
		{
			FieldName:    "TargetId",
			RequestName:  "targetId",
			Contribution: "query",
			LookupPaths: []string{
				"status.columnSource.targetId",
				"spec.columnSource.targetId",
				"columnSource.targetId",
			},
		},
		{FieldName: "MaskingPolicyId", RequestName: "maskingPolicyId", Contribution: "query", PreferResourceID: true},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "TimeCreatedGreaterThanOrEqualTo", RequestName: "timeCreatedGreaterThanOrEqualTo", Contribution: "query"},
		{FieldName: "TimeCreatedLessThan", RequestName: "timeCreatedLessThan", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
	}
}

func maskingPolicyUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MaskingPolicyId", RequestName: "maskingPolicyId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateMaskingPolicyDetails", RequestName: "UpdateMaskingPolicyDetails", Contribution: "body"},
	}
}

func maskingPolicyDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MaskingPolicyId", RequestName: "maskingPolicyId", Contribution: "path", PreferResourceID: true},
	}
}

func buildMaskingPolicyCreateBody(_ context.Context, resource *datasafev1beta1.MaskingPolicy, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", maskingPolicyKind)
	}
	columnSource, err := maskingPolicyCreateColumnSource(resource.Spec.ColumnSource)
	if err != nil {
		return nil, err
	}
	recompile, err := maskingPolicyRecompileEnum(resource.Spec.Recompile)
	if err != nil {
		return nil, err
	}
	return datasafesdk.CreateMaskingPolicyDetails{
		CompartmentId:           common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		ColumnSource:            columnSource,
		DisplayName:             maskingPolicyOptionalString(resource.Spec.DisplayName),
		Description:             maskingPolicyOptionalString(resource.Spec.Description),
		IsDropTempTablesEnabled: common.Bool(resource.Spec.IsDropTempTablesEnabled),
		IsRedoLoggingEnabled:    common.Bool(resource.Spec.IsRedoLoggingEnabled),
		IsRefreshStatsEnabled:   common.Bool(resource.Spec.IsRefreshStatsEnabled),
		ParallelDegree:          maskingPolicyOptionalString(resource.Spec.ParallelDegree),
		Recompile:               recompile,
		PreMaskingScript:        maskingPolicyOptionalString(resource.Spec.PreMaskingScript),
		PostMaskingScript:       maskingPolicyOptionalString(resource.Spec.PostMaskingScript),
		FreeformTags:            maskingPolicyStringMap(resource.Spec.FreeformTags),
		DefinedTags:             maskingPolicyDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func buildMaskingPolicyUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.MaskingPolicy,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", maskingPolicyKind)
	}
	details, err := maskingPolicyUpdateDetails(resource)
	if err != nil {
		return nil, false, err
	}
	current, ok := maskingPolicyStatusProjectionFromResponse(currentResponse)
	if !ok {
		return details, true, nil
	}
	return details, maskingPolicyUpdateNeeded(resource, current), nil
}

func maskingPolicyUpdateDetails(resource *datasafev1beta1.MaskingPolicy) (datasafesdk.UpdateMaskingPolicyDetails, error) {
	columnSource, err := maskingPolicyUpdateColumnSource(resource.Spec.ColumnSource)
	if err != nil {
		return datasafesdk.UpdateMaskingPolicyDetails{}, err
	}
	recompile, err := maskingPolicyRecompileEnum(resource.Spec.Recompile)
	if err != nil {
		return datasafesdk.UpdateMaskingPolicyDetails{}, err
	}
	return datasafesdk.UpdateMaskingPolicyDetails{
		DisplayName:             maskingPolicyOptionalString(resource.Spec.DisplayName),
		Description:             maskingPolicyOptionalString(resource.Spec.Description),
		IsDropTempTablesEnabled: common.Bool(resource.Spec.IsDropTempTablesEnabled),
		IsRedoLoggingEnabled:    common.Bool(resource.Spec.IsRedoLoggingEnabled),
		IsRefreshStatsEnabled:   common.Bool(resource.Spec.IsRefreshStatsEnabled),
		ParallelDegree:          maskingPolicyOptionalString(resource.Spec.ParallelDegree),
		Recompile:               recompile,
		PreMaskingScript:        maskingPolicyOptionalString(resource.Spec.PreMaskingScript),
		PostMaskingScript:       maskingPolicyOptionalString(resource.Spec.PostMaskingScript),
		ColumnSource:            columnSource,
		FreeformTags:            maskingPolicyStringMap(resource.Spec.FreeformTags),
		DefinedTags:             maskingPolicyDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func maskingPolicyUpdateNeeded(resource *datasafev1beta1.MaskingPolicy, current maskingPolicyStatusProjection) bool {
	spec := resource.Spec
	return maskingPolicyScalarUpdateNeeded(spec, current) ||
		maskingPolicyBooleanUpdateNeeded(spec, current) ||
		!maskingPolicyColumnSourceEqual(maskingPolicyAPIColumnSource(spec.ColumnSource), current.ColumnSource) ||
		maskingPolicyTagUpdateNeeded(spec, current)
}

func maskingPolicyScalarUpdateNeeded(
	spec datasafev1beta1.MaskingPolicySpec,
	current maskingPolicyStatusProjection,
) bool {
	checks := []struct {
		desired string
		current string
	}{
		{desired: spec.DisplayName, current: current.DisplayName},
		{desired: spec.Description, current: current.Description},
		{desired: spec.ParallelDegree, current: current.ParallelDegree},
		{desired: maskingPolicyCanonicalRecompile(spec.Recompile), current: maskingPolicyCanonicalRecompile(current.Recompile)},
		{desired: spec.PreMaskingScript, current: current.PreMaskingScript},
		{desired: spec.PostMaskingScript, current: current.PostMaskingScript},
	}
	for _, check := range checks {
		if maskingPolicyMeaningfulStringChanged(check.desired, check.current) {
			return true
		}
	}
	return false
}

func maskingPolicyBooleanUpdateNeeded(
	spec datasafev1beta1.MaskingPolicySpec,
	current maskingPolicyStatusProjection,
) bool {
	return spec.IsDropTempTablesEnabled != current.IsDropTempTablesEnabled ||
		spec.IsRedoLoggingEnabled != current.IsRedoLoggingEnabled ||
		spec.IsRefreshStatsEnabled != current.IsRefreshStatsEnabled
}

func maskingPolicyTagUpdateNeeded(
	spec datasafev1beta1.MaskingPolicySpec,
	current maskingPolicyStatusProjection,
) bool {
	return spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) ||
		spec.DefinedTags != nil && !maskingPolicySharedTagsEqual(spec.DefinedTags, current.DefinedTags)
}

func resolveMaskingPolicyIdentity(resource *datasafev1beta1.MaskingPolicy) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", maskingPolicyKind)
	}
	return maskingPolicyIdentityFromResource(resource)
}

func maskingPolicyIdentityFromResource(resource *datasafev1beta1.MaskingPolicy) (maskingPolicyIdentity, error) {
	if resource == nil {
		return maskingPolicyIdentity{}, nil
	}
	columnSource, err := maskingPolicyAPIColumnSourceValidated(resource.Spec.ColumnSource)
	if err != nil {
		return maskingPolicyIdentity{}, err
	}
	return maskingPolicyIdentity{
		compartmentID:        strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:          strings.TrimSpace(resource.Spec.DisplayName),
		columnSource:         columnSource.ColumnSource,
		targetID:             columnSource.TargetId,
		sensitiveDataModelID: columnSource.SensitiveDataModelId,
	}, nil
}

func recordMaskingPolicyPathIdentity(resource *datasafev1beta1.MaskingPolicy, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(maskingPolicyIdentity)
	if !ok {
		return
	}
	if typed.compartmentID != "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
	if typed.displayName != "" {
		resource.Status.DisplayName = typed.displayName
	}
	resource.Status.ColumnSource = datasafev1beta1.MaskingPolicyColumnSource{
		ColumnSource:         typed.columnSource,
		TargetId:             typed.targetID,
		SensitiveDataModelId: typed.sensitiveDataModelID,
	}
}

func listMaskingPoliciesAllPages(
	call func(context.Context, datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error),
) func(context.Context, datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListMaskingPoliciesRequest) (datasafesdk.ListMaskingPoliciesResponse, error) {
		var combined datasafesdk.ListMaskingPoliciesResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			combined.RawResponse = response.RawResponse
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleMaskingPolicyDeleteError(resource *datasafev1beta1.MaskingPolicy, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := maskingPolicyAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func applyMaskingPolicyDeleteOutcome(
	resource *datasafev1beta1.MaskingPolicy,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := maskingPolicyLifecycleState(response)
	if state == "" {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if maskingPolicyTerminalDeleteState(state) {
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest {
		markMaskingPolicyTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markMaskingPolicyTerminating(resource *datasafev1beta1.MaskingPolicy, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = "OCI resource delete is in progress"
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         status.Message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, loggerutil.OSOKLogger{})
}

func maskingPolicyLifecycleState(response any) string {
	if current, ok := maskingPolicyFromResponse(response); ok {
		return strings.ToUpper(string(current.LifecycleState))
	}
	if summary, ok := maskingPolicySummaryFromResponse(response); ok {
		return strings.ToUpper(string(summary.LifecycleState))
	}
	return ""
}

func maskingPolicyTerminalDeleteState(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), string(datasafesdk.MaskingLifecycleStateDeleted))
}

func projectMaskingPolicyStatus(resource *datasafev1beta1.MaskingPolicy, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", maskingPolicyKind)
	}
	projected, ok := maskingPolicyStatusProjectionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.MaskingPolicyStatus{
		OsokStatus:                   osokStatus,
		Id:                           projected.Id,
		CompartmentId:                projected.CompartmentId,
		DisplayName:                  projected.DisplayName,
		TimeCreated:                  projected.TimeCreated,
		LifecycleState:               projected.LifecycleState,
		TimeUpdated:                  projected.TimeUpdated,
		IsDropTempTablesEnabled:      projected.IsDropTempTablesEnabled,
		IsRedoLoggingEnabled:         projected.IsRedoLoggingEnabled,
		IsRefreshStatsEnabled:        projected.IsRefreshStatsEnabled,
		ParallelDegree:               projected.ParallelDegree,
		Recompile:                    projected.Recompile,
		Description:                  projected.Description,
		PreMaskingScript:             projected.PreMaskingScript,
		PostMaskingScript:            projected.PostMaskingScript,
		ColumnSource:                 projected.ColumnSource,
		AreTargetCredentialsRequired: projected.AreTargetCredentialsRequired,
		FreeformTags:                 maskingPolicyStringMap(projected.FreeformTags),
		DefinedTags:                  maskingPolicyCloneSharedTags(projected.DefinedTags),
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

type maskingPolicyStatusProjection struct {
	Id                           string
	CompartmentId                string
	DisplayName                  string
	TimeCreated                  string
	LifecycleState               string
	TimeUpdated                  string
	IsDropTempTablesEnabled      bool
	IsRedoLoggingEnabled         bool
	IsRefreshStatsEnabled        bool
	ParallelDegree               string
	Recompile                    string
	Description                  string
	PreMaskingScript             string
	PostMaskingScript            string
	ColumnSource                 datasafev1beta1.MaskingPolicyColumnSource
	AreTargetCredentialsRequired bool
	FreeformTags                 map[string]string
	DefinedTags                  map[string]shared.MapValue
}

func maskingPolicyStatusProjectionFromResponse(response any) (maskingPolicyStatusProjection, bool) {
	if current, ok := maskingPolicyFromResponse(response); ok {
		return maskingPolicyStatusProjectionFromSDK(current), true
	}
	if summary, ok := maskingPolicySummaryFromResponse(response); ok {
		return maskingPolicyStatusProjectionFromSummary(summary), true
	}
	return maskingPolicyStatusProjection{}, false
}

func maskingPolicyFromResponse(response any) (datasafesdk.MaskingPolicy, bool) {
	switch current := response.(type) {
	case datasafesdk.GetMaskingPolicyResponse:
		return current.MaskingPolicy, true
	case *datasafesdk.GetMaskingPolicyResponse:
		if current == nil {
			return datasafesdk.MaskingPolicy{}, false
		}
		return current.MaskingPolicy, true
	case datasafesdk.CreateMaskingPolicyResponse:
		return current.MaskingPolicy, true
	case *datasafesdk.CreateMaskingPolicyResponse:
		if current == nil {
			return datasafesdk.MaskingPolicy{}, false
		}
		return current.MaskingPolicy, true
	case datasafesdk.MaskingPolicy:
		return current, true
	case *datasafesdk.MaskingPolicy:
		if current == nil {
			return datasafesdk.MaskingPolicy{}, false
		}
		return *current, true
	default:
		return datasafesdk.MaskingPolicy{}, false
	}
}

func maskingPolicySummaryFromResponse(response any) (datasafesdk.MaskingPolicySummary, bool) {
	switch current := response.(type) {
	case datasafesdk.MaskingPolicySummary:
		return current, true
	case *datasafesdk.MaskingPolicySummary:
		if current == nil {
			return datasafesdk.MaskingPolicySummary{}, false
		}
		return *current, true
	default:
		return datasafesdk.MaskingPolicySummary{}, false
	}
}

func maskingPolicyStatusProjectionFromSDK(current datasafesdk.MaskingPolicy) maskingPolicyStatusProjection {
	return maskingPolicyStatusProjection{
		Id:                           maskingPolicyStringValue(current.Id),
		CompartmentId:                maskingPolicyStringValue(current.CompartmentId),
		DisplayName:                  maskingPolicyStringValue(current.DisplayName),
		TimeCreated:                  maskingPolicySDKTimeString(current.TimeCreated),
		LifecycleState:               string(current.LifecycleState),
		TimeUpdated:                  maskingPolicySDKTimeString(current.TimeUpdated),
		IsDropTempTablesEnabled:      maskingPolicyBoolValue(current.IsDropTempTablesEnabled),
		IsRedoLoggingEnabled:         maskingPolicyBoolValue(current.IsRedoLoggingEnabled),
		IsRefreshStatsEnabled:        maskingPolicyBoolValue(current.IsRefreshStatsEnabled),
		ParallelDegree:               maskingPolicyStringValue(current.ParallelDegree),
		Recompile:                    string(current.Recompile),
		Description:                  maskingPolicyStringValue(current.Description),
		PreMaskingScript:             maskingPolicyStringValue(current.PreMaskingScript),
		PostMaskingScript:            maskingPolicyStringValue(current.PostMaskingScript),
		ColumnSource:                 maskingPolicyColumnSourceFromSDK(current.ColumnSource),
		AreTargetCredentialsRequired: maskingPolicyBoolValue(current.AreTargetCredentialsRequired),
		FreeformTags:                 maskingPolicyStringMap(current.FreeformTags),
		DefinedTags:                  maskingPolicySharedTags(current.DefinedTags),
	}
}

func maskingPolicyStatusProjectionFromSummary(current datasafesdk.MaskingPolicySummary) maskingPolicyStatusProjection {
	return maskingPolicyStatusProjection{
		Id:             maskingPolicyStringValue(current.Id),
		CompartmentId:  maskingPolicyStringValue(current.CompartmentId),
		DisplayName:    maskingPolicyStringValue(current.DisplayName),
		TimeCreated:    maskingPolicySDKTimeString(current.TimeCreated),
		LifecycleState: string(current.LifecycleState),
		TimeUpdated:    maskingPolicySDKTimeString(current.TimeUpdated),
		Description:    maskingPolicyStringValue(current.Description),
		ColumnSource:   maskingPolicyColumnSourceFromSDK(current.ColumnSource),
		FreeformTags:   maskingPolicyStringMap(current.FreeformTags),
		DefinedTags:    maskingPolicySharedTags(current.DefinedTags),
	}
}

func (c maskingPolicyRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.MaskingPolicy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", maskingPolicyKind)
	}
	if _, err := maskingPolicyIdentityFromResource(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, markMaskingPolicyFailed(resource, err)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func markMaskingPolicyFailed(resource *datasafev1beta1.MaskingPolicy, err error) error {
	if resource == nil || err == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		status.Async.Current = &current
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), loggerutil.OSOKLogger{})
	return err
}

func (c maskingPolicyRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.MaskingPolicy) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", maskingPolicyKind)
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c maskingPolicyRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.MaskingPolicy,
) error {
	if resource == nil {
		return nil
	}
	currentID := trackedMaskingPolicyID(resource)
	if currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedList(ctx, resource)
}

func (c maskingPolicyRuntimeClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *datasafev1beta1.MaskingPolicy,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	_, err := c.get(ctx, datasafesdk.GetMaskingPolicyRequest{MaskingPolicyId: common.String(currentID)})
	return maskingPolicyAmbiguousDeleteError(resource, err, "pre-delete get")
}

func (c maskingPolicyRuntimeClient) rejectAuthShapedList(
	ctx context.Context,
	resource *datasafev1beta1.MaskingPolicy,
) error {
	if c.list == nil {
		return nil
	}
	identity, err := maskingPolicyIdentityFromResource(resource)
	if err != nil {
		return err
	}
	if identity.compartmentID == "" {
		return nil
	}
	_, err = c.list(ctx, datasafesdk.ListMaskingPoliciesRequest{
		CompartmentId:        common.String(identity.compartmentID),
		DisplayName:          maskingPolicyOptionalString(identity.displayName),
		SensitiveDataModelId: maskingPolicyOptionalString(identity.sensitiveDataModelID),
		TargetId:             maskingPolicyOptionalString(identity.targetID),
	})
	return maskingPolicyAmbiguousDeleteError(resource, err, "pre-delete list")
}

func maskingPolicyAmbiguousDeleteError(
	resource *datasafev1beta1.MaskingPolicy,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return maskingPolicyAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", maskingPolicyKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func trackedMaskingPolicyID(resource *datasafev1beta1.MaskingPolicy) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func maskingPolicyCreateColumnSource(source datasafev1beta1.MaskingPolicyColumnSource) (datasafesdk.CreateColumnSourceDetails, error) {
	apiSource, err := maskingPolicyAPIColumnSourceValidated(source)
	if err != nil {
		return nil, err
	}
	switch apiSource.ColumnSource {
	case string(datasafesdk.CreateColumnSourceDetailsColumnSourceTarget):
		return datasafesdk.CreateColumnSourceFromTargetDetails{TargetId: common.String(apiSource.TargetId)}, nil
	case string(datasafesdk.CreateColumnSourceDetailsColumnSourceSensitiveDataModel):
		return datasafesdk.CreateColumnSourceFromSdmDetails{SensitiveDataModelId: common.String(apiSource.SensitiveDataModelId)}, nil
	default:
		return nil, fmt.Errorf("%s columnSource %q is not supported", maskingPolicyKind, apiSource.ColumnSource)
	}
}

func maskingPolicyUpdateColumnSource(source datasafev1beta1.MaskingPolicyColumnSource) (datasafesdk.UpdateColumnSourceDetails, error) {
	apiSource, err := maskingPolicyAPIColumnSourceValidated(source)
	if err != nil {
		return nil, err
	}
	switch apiSource.ColumnSource {
	case string(datasafesdk.UpdateColumnSourceDetailsColumnSourceTarget):
		return datasafesdk.UpdateColumnSourceTargetDetails{TargetId: common.String(apiSource.TargetId)}, nil
	case string(datasafesdk.UpdateColumnSourceDetailsColumnSourceSensitiveDataModel):
		return datasafesdk.UpdateColumnSourceSdmDetails{SensitiveDataModelId: common.String(apiSource.SensitiveDataModelId)}, nil
	default:
		return nil, fmt.Errorf("%s columnSource %q is not supported", maskingPolicyKind, apiSource.ColumnSource)
	}
}

func maskingPolicyAPIColumnSourceValidated(source datasafev1beta1.MaskingPolicyColumnSource) (datasafev1beta1.MaskingPolicyColumnSource, error) {
	normalized := maskingPolicyAPIColumnSource(source)
	switch normalized.ColumnSource {
	case string(datasafesdk.ColumnSourceDetailsColumnSourceTarget):
		if normalized.TargetId == "" {
			return normalized, fmt.Errorf("%s columnSource.targetId is required when columnSource is TARGET", maskingPolicyKind)
		}
		if normalized.SensitiveDataModelId != "" {
			return normalized, fmt.Errorf("%s columnSource.sensitiveDataModelId cannot be set when columnSource is TARGET", maskingPolicyKind)
		}
	case string(datasafesdk.ColumnSourceDetailsColumnSourceSensitiveDataModel):
		if normalized.SensitiveDataModelId == "" {
			return normalized, fmt.Errorf("%s columnSource.sensitiveDataModelId is required when columnSource is SENSITIVE_DATA_MODEL", maskingPolicyKind)
		}
		if normalized.TargetId != "" {
			return normalized, fmt.Errorf("%s columnSource.targetId cannot be set when columnSource is SENSITIVE_DATA_MODEL", maskingPolicyKind)
		}
	default:
		return normalized, fmt.Errorf("%s columnSource must be TARGET or SENSITIVE_DATA_MODEL", maskingPolicyKind)
	}
	return normalized, nil
}

func maskingPolicyAPIColumnSource(source datasafev1beta1.MaskingPolicyColumnSource) datasafev1beta1.MaskingPolicyColumnSource {
	normalized := datasafev1beta1.MaskingPolicyColumnSource{
		ColumnSource:         strings.ToUpper(strings.TrimSpace(source.ColumnSource)),
		TargetId:             strings.TrimSpace(source.TargetId),
		SensitiveDataModelId: strings.TrimSpace(source.SensitiveDataModelId),
	}
	switch {
	case normalized.ColumnSource == "" && normalized.TargetId != "":
		normalized.ColumnSource = string(datasafesdk.ColumnSourceDetailsColumnSourceTarget)
	case normalized.ColumnSource == "" && normalized.SensitiveDataModelId != "":
		normalized.ColumnSource = string(datasafesdk.ColumnSourceDetailsColumnSourceSensitiveDataModel)
	}
	return normalized
}

func maskingPolicyColumnSourceFromSDK(source datasafesdk.ColumnSourceDetails) datasafev1beta1.MaskingPolicyColumnSource {
	switch typed := source.(type) {
	case datasafesdk.ColumnSourceFromTargetDetails:
		return datasafev1beta1.MaskingPolicyColumnSource{
			ColumnSource: string(datasafesdk.ColumnSourceDetailsColumnSourceTarget),
			TargetId:     maskingPolicyStringValue(typed.TargetId),
		}
	case *datasafesdk.ColumnSourceFromTargetDetails:
		if typed == nil {
			return datasafev1beta1.MaskingPolicyColumnSource{}
		}
		return datasafev1beta1.MaskingPolicyColumnSource{
			ColumnSource: string(datasafesdk.ColumnSourceDetailsColumnSourceTarget),
			TargetId:     maskingPolicyStringValue(typed.TargetId),
		}
	case datasafesdk.ColumnSourceFromSdmDetails:
		return datasafev1beta1.MaskingPolicyColumnSource{
			ColumnSource:         string(datasafesdk.ColumnSourceDetailsColumnSourceSensitiveDataModel),
			SensitiveDataModelId: maskingPolicyStringValue(typed.SensitiveDataModelId),
		}
	case *datasafesdk.ColumnSourceFromSdmDetails:
		if typed == nil {
			return datasafev1beta1.MaskingPolicyColumnSource{}
		}
		return datasafev1beta1.MaskingPolicyColumnSource{
			ColumnSource:         string(datasafesdk.ColumnSourceDetailsColumnSourceSensitiveDataModel),
			SensitiveDataModelId: maskingPolicyStringValue(typed.SensitiveDataModelId),
		}
	default:
		return datasafev1beta1.MaskingPolicyColumnSource{}
	}
}

func maskingPolicyColumnSourceEqual(left datasafev1beta1.MaskingPolicyColumnSource, right datasafev1beta1.MaskingPolicyColumnSource) bool {
	left = maskingPolicyAPIColumnSource(left)
	right = maskingPolicyAPIColumnSource(right)
	return left.ColumnSource == right.ColumnSource &&
		left.TargetId == right.TargetId &&
		left.SensitiveDataModelId == right.SensitiveDataModelId
}

func maskingPolicyMeaningfulStringChanged(desired string, current string) bool {
	desired = strings.TrimSpace(desired)
	if desired == "" {
		return false
	}
	return desired != strings.TrimSpace(current)
}

func maskingPolicyRecompileEnum(value string) (datasafesdk.MaskingPolicyRecompileEnum, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	recompile, ok := datasafesdk.GetMappingMaskingPolicyRecompileEnum(value)
	if !ok {
		return "", fmt.Errorf("%s recompile %q is not supported; supported values are: %s", maskingPolicyKind, value, strings.Join(datasafesdk.GetMaskingPolicyRecompileEnumStringValues(), ", "))
	}
	return recompile, nil
}

func maskingPolicyCanonicalRecompile(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	recompile, ok := datasafesdk.GetMappingMaskingPolicyRecompileEnum(value)
	if !ok {
		return value
	}
	return string(recompile)
}

func maskingPolicyOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func maskingPolicyStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func maskingPolicyBoolValue(value *bool) bool {
	return value != nil && *value
}

func maskingPolicySDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func maskingPolicyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func maskingPolicySharedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		converted[namespace] = make(shared.MapValue, len(values))
		for key, value := range values {
			converted[namespace][key] = fmt.Sprint(value)
		}
	}
	return converted
}

func maskingPolicyCloneSharedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		cloned[namespace] = make(shared.MapValue, len(values))
		for key, value := range values {
			cloned[namespace][key] = value
		}
	}
	return cloned
}

func maskingPolicyDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

func maskingPolicySharedTagsEqual(left map[string]shared.MapValue, right map[string]shared.MapValue) bool {
	if len(left) != len(right) {
		return false
	}
	for namespace, leftValues := range left {
		rightValues, ok := right[namespace]
		if !ok || !maps.Equal(leftValues, rightValues) {
			return false
		}
	}
	return true
}

var _ interface{ GetOpcRequestID() string } = maskingPolicyAmbiguousNotFoundError{}
