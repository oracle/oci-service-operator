/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package approvaltemplate

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	lockboxsdk "github.com/oracle/oci-go-sdk/v65/lockbox"
	lockboxv1beta1 "github.com/oracle/oci-service-operator/api/lockbox/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type approvalTemplateOCIClient interface {
	CreateApprovalTemplate(context.Context, lockboxsdk.CreateApprovalTemplateRequest) (lockboxsdk.CreateApprovalTemplateResponse, error)
	GetApprovalTemplate(context.Context, lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error)
	ListApprovalTemplates(context.Context, lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error)
	UpdateApprovalTemplate(context.Context, lockboxsdk.UpdateApprovalTemplateRequest) (lockboxsdk.UpdateApprovalTemplateResponse, error)
	DeleteApprovalTemplate(context.Context, lockboxsdk.DeleteApprovalTemplateRequest) (lockboxsdk.DeleteApprovalTemplateResponse, error)
}

type approvalTemplateRuntimeClient struct {
	delegate ApprovalTemplateServiceClient
	hooks    ApprovalTemplateRuntimeHooks
}

var _ ApprovalTemplateServiceClient = (*approvalTemplateRuntimeClient)(nil)

func init() {
	registerApprovalTemplateRuntimeHooksMutator(func(manager *ApprovalTemplateServiceManager, hooks *ApprovalTemplateRuntimeHooks) {
		applyApprovalTemplateRuntimeHooks(manager, hooks)
	})
}

func applyApprovalTemplateRuntimeHooks(_ *ApprovalTemplateServiceManager, hooks *ApprovalTemplateRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newApprovalTemplateRuntimeSemantics()
	hooks.BuildCreateBody = buildApprovalTemplateCreateBody
	hooks.BuildUpdateBody = buildApprovalTemplateUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardApprovalTemplateExistingBeforeCreate
	hooks.List.Fields = approvalTemplateListFields()
	hooks.DeleteHooks.HandleError = handleApprovalTemplateDeleteError
	wrapApprovalTemplateReadAndDeleteCalls(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ApprovalTemplateServiceClient) ApprovalTemplateServiceClient {
		return &approvalTemplateRuntimeClient{delegate: delegate, hooks: *hooks}
	})
}

func newApprovalTemplateServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client approvalTemplateOCIClient,
) ApprovalTemplateServiceClient {
	hooks := newApprovalTemplateRuntimeHooksWithOCIClient(client)
	applyApprovalTemplateRuntimeHooks(&ApprovalTemplateServiceManager{Log: log}, &hooks)
	manager := &ApprovalTemplateServiceManager{Log: log}
	delegate := defaultApprovalTemplateServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*lockboxv1beta1.ApprovalTemplate](
			buildApprovalTemplateGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapApprovalTemplateGeneratedClient(hooks, delegate)
}

func newApprovalTemplateRuntimeHooksWithOCIClient(client approvalTemplateOCIClient) ApprovalTemplateRuntimeHooks {
	return ApprovalTemplateRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*lockboxv1beta1.ApprovalTemplate]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*lockboxv1beta1.ApprovalTemplate]{},
		StatusHooks:     generatedruntime.StatusHooks[*lockboxv1beta1.ApprovalTemplate]{},
		ParityHooks:     generatedruntime.ParityHooks[*lockboxv1beta1.ApprovalTemplate]{},
		Async:           generatedruntime.AsyncHooks[*lockboxv1beta1.ApprovalTemplate]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*lockboxv1beta1.ApprovalTemplate]{},
		Create: runtimeOperationHooks[lockboxsdk.CreateApprovalTemplateRequest, lockboxsdk.CreateApprovalTemplateResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateApprovalTemplateDetails", RequestName: "CreateApprovalTemplateDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request lockboxsdk.CreateApprovalTemplateRequest) (lockboxsdk.CreateApprovalTemplateResponse, error) {
				if client == nil {
					return lockboxsdk.CreateApprovalTemplateResponse{}, fmt.Errorf("ApprovalTemplate OCI client is nil")
				}
				return client.CreateApprovalTemplate(ctx, request)
			},
		},
		Get: runtimeOperationHooks[lockboxsdk.GetApprovalTemplateRequest, lockboxsdk.GetApprovalTemplateResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ApprovalTemplateId", RequestName: "approvalTemplateId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
				if client == nil {
					return lockboxsdk.GetApprovalTemplateResponse{}, fmt.Errorf("ApprovalTemplate OCI client is nil")
				}
				return client.GetApprovalTemplate(ctx, request)
			},
		},
		List: runtimeOperationHooks[lockboxsdk.ListApprovalTemplatesRequest, lockboxsdk.ListApprovalTemplatesResponse]{
			Fields: approvalTemplateListFields(),
			Call: func(ctx context.Context, request lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error) {
				if client == nil {
					return lockboxsdk.ListApprovalTemplatesResponse{}, fmt.Errorf("ApprovalTemplate OCI client is nil")
				}
				return client.ListApprovalTemplates(ctx, request)
			},
		},
		Update: runtimeOperationHooks[lockboxsdk.UpdateApprovalTemplateRequest, lockboxsdk.UpdateApprovalTemplateResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ApprovalTemplateId", RequestName: "approvalTemplateId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateApprovalTemplateDetails", RequestName: "UpdateApprovalTemplateDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request lockboxsdk.UpdateApprovalTemplateRequest) (lockboxsdk.UpdateApprovalTemplateResponse, error) {
				if client == nil {
					return lockboxsdk.UpdateApprovalTemplateResponse{}, fmt.Errorf("ApprovalTemplate OCI client is nil")
				}
				return client.UpdateApprovalTemplate(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[lockboxsdk.DeleteApprovalTemplateRequest, lockboxsdk.DeleteApprovalTemplateResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ApprovalTemplateId", RequestName: "approvalTemplateId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request lockboxsdk.DeleteApprovalTemplateRequest) (lockboxsdk.DeleteApprovalTemplateResponse, error) {
				if client == nil {
					return lockboxsdk.DeleteApprovalTemplateResponse{}, fmt.Errorf("ApprovalTemplate OCI client is nil")
				}
				return client.DeleteApprovalTemplate(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ApprovalTemplateServiceClient) ApprovalTemplateServiceClient{},
	}
}

func newApprovalTemplateRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "lockbox",
		FormalSlug:        "approvaltemplate",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(lockboxsdk.ApprovalTemplateLifecycleStateCreating)},
			UpdatingStates:     []string{string(lockboxsdk.ApprovalTemplateLifecycleStateUpdating)},
			ActiveStates:       []string{string(lockboxsdk.ApprovalTemplateLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(lockboxsdk.ApprovalTemplateLifecycleStateCreating),
				string(lockboxsdk.ApprovalTemplateLifecycleStateUpdating),
				string(lockboxsdk.ApprovalTemplateLifecycleStateDeleting),
			},
			TerminalStates: []string{string(lockboxsdk.ApprovalTemplateLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "approverLevels", "autoApprovalState", "freeformTags", "definedTags"},
			Mutable:         []string{"displayName", "approverLevels", "autoApprovalState", "freeformTags", "definedTags"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "ApprovalTemplate", Action: "CreateApprovalTemplate"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "ApprovalTemplate", Action: "UpdateApprovalTemplate"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "ApprovalTemplate", Action: "DeleteApprovalTemplate"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func approvalTemplateListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardApprovalTemplateExistingBeforeCreate(
	_ context.Context,
	resource *lockboxv1beta1.ApprovalTemplate,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ApprovalTemplate resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildApprovalTemplateCreateBody(
	_ context.Context,
	resource *lockboxv1beta1.ApprovalTemplate,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ApprovalTemplate resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("ApprovalTemplate spec is invalid: compartmentId is required")
	}

	details := lockboxsdk.CreateApprovalTemplateDetails{
		CompartmentId: stringPointer(strings.TrimSpace(resource.Spec.CompartmentId)),
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		details.DisplayName = stringPointer(displayName)
	}
	approverLevels, specified, err := approvalTemplateApproverLevelsFromSpec(resource.Spec.ApproverLevels)
	if err != nil {
		return nil, err
	}
	if specified {
		details.ApproverLevels = approverLevels
	}
	if autoApprovalState := strings.TrimSpace(resource.Spec.AutoApprovalState); autoApprovalState != "" {
		details.AutoApprovalState = lockboxsdk.LockboxAutoApprovalStateEnum(autoApprovalState)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildApprovalTemplateUpdateBody(
	_ context.Context,
	resource *lockboxv1beta1.ApprovalTemplate,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return lockboxsdk.UpdateApprovalTemplateDetails{}, false, fmt.Errorf("ApprovalTemplate resource is nil")
	}
	current, ok := approvalTemplateFromResponse(currentResponse)
	if !ok {
		return lockboxsdk.UpdateApprovalTemplateDetails{}, false, fmt.Errorf("current ApprovalTemplate response does not expose an ApprovalTemplate body")
	}
	if err := validateApprovalTemplateCreateOnlyDrift(resource.Spec, current); err != nil {
		return lockboxsdk.UpdateApprovalTemplateDetails{}, false, err
	}

	details := lockboxsdk.UpdateApprovalTemplateDetails{}
	updateNeeded := false
	updateNeeded = applyApprovalTemplateDisplayNameUpdate(&details, resource.Spec, current) || updateNeeded
	approverUpdated, err := applyApprovalTemplateApproverLevelsUpdate(&details, resource.Spec, current)
	if err != nil {
		return lockboxsdk.UpdateApprovalTemplateDetails{}, false, err
	}
	updateNeeded = approverUpdated || updateNeeded
	updateNeeded = applyApprovalTemplateAutoApprovalStateUpdate(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyApprovalTemplateFreeformTagsUpdate(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyApprovalTemplateDefinedTagsUpdate(&details, resource.Spec, current) || updateNeeded
	return details, updateNeeded, nil
}

func applyApprovalTemplateDisplayNameUpdate(
	details *lockboxsdk.UpdateApprovalTemplateDetails,
	spec lockboxv1beta1.ApprovalTemplateSpec,
	current lockboxsdk.ApprovalTemplate,
) bool {
	desired := strings.TrimSpace(spec.DisplayName)
	if desired == "" || desired == stringPointerValue(current.DisplayName) {
		return false
	}
	details.DisplayName = stringPointer(desired)
	return true
}

func applyApprovalTemplateApproverLevelsUpdate(
	details *lockboxsdk.UpdateApprovalTemplateDetails,
	spec lockboxv1beta1.ApprovalTemplateSpec,
	current lockboxsdk.ApprovalTemplate,
) (bool, error) {
	desired, specified, err := approvalTemplateApproverLevelsFromSpec(spec.ApproverLevels)
	if err != nil {
		return false, err
	}
	if !specified || approvalTemplateApproverLevelsEqual(desired, current.ApproverLevels) {
		return false, nil
	}
	details.ApproverLevels = desired
	return true, nil
}

func applyApprovalTemplateAutoApprovalStateUpdate(
	details *lockboxsdk.UpdateApprovalTemplateDetails,
	spec lockboxv1beta1.ApprovalTemplateSpec,
	current lockboxsdk.ApprovalTemplate,
) bool {
	desired := strings.TrimSpace(spec.AutoApprovalState)
	if desired == "" || desired == string(current.AutoApprovalState) {
		return false
	}
	details.AutoApprovalState = lockboxsdk.LockboxAutoApprovalStateEnum(desired)
	return true
}

func applyApprovalTemplateFreeformTagsUpdate(
	details *lockboxsdk.UpdateApprovalTemplateDetails,
	spec lockboxv1beta1.ApprovalTemplateSpec,
	current lockboxsdk.ApprovalTemplate,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	details.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func applyApprovalTemplateDefinedTagsUpdate(
	details *lockboxsdk.UpdateApprovalTemplateDetails,
	spec lockboxv1beta1.ApprovalTemplateSpec,
	current lockboxsdk.ApprovalTemplate,
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

func validateApprovalTemplateCreateOnlyDrift(
	spec lockboxv1beta1.ApprovalTemplateSpec,
	current lockboxsdk.ApprovalTemplate,
) error {
	if strings.TrimSpace(spec.CompartmentId) == stringPointerValue(current.CompartmentId) {
		return nil
	}
	return fmt.Errorf("ApprovalTemplate create-only field drift is not supported: compartmentId")
}

func approvalTemplateApproverLevelsFromSpec(
	spec lockboxv1beta1.ApprovalTemplateApproverLevels,
) (*lockboxsdk.ApproverLevels, bool, error) {
	specified := approvalTemplateSpecifiedApproverLevels(spec)
	if !specified.any() {
		return nil, false, nil
	}
	if err := specified.validate(); err != nil {
		return nil, false, err
	}

	level1, err := approvalTemplateApproverInfoFromLevel1("level1", spec.Level1)
	if err != nil {
		return nil, false, err
	}
	levels := &lockboxsdk.ApproverLevels{Level1: level1}
	if err := applyApprovalTemplateOptionalApproverLevels(levels, spec, specified); err != nil {
		return nil, false, err
	}
	return levels, true, nil
}

type specifiedApprovalTemplateApproverLevels struct {
	level1 bool
	level2 bool
	level3 bool
}

func approvalTemplateSpecifiedApproverLevels(
	spec lockboxv1beta1.ApprovalTemplateApproverLevels,
) specifiedApprovalTemplateApproverLevels {
	return specifiedApprovalTemplateApproverLevels{
		level1: approvalTemplateApproverLevel1Specified(spec.Level1),
		level2: approvalTemplateApproverLevel2Specified(spec.Level2),
		level3: approvalTemplateApproverLevel3Specified(spec.Level3),
	}
}

func (s specifiedApprovalTemplateApproverLevels) any() bool {
	return s.level1 || s.level2 || s.level3
}

func (s specifiedApprovalTemplateApproverLevels) validate() error {
	if !s.level1 {
		return fmt.Errorf("ApprovalTemplate spec is invalid: approverLevels.level1 is required when approver levels are set")
	}
	if s.level3 && !s.level2 {
		return fmt.Errorf("ApprovalTemplate spec is invalid: approverLevels.level2 is required when level3 is set")
	}
	return nil
}

func applyApprovalTemplateOptionalApproverLevels(
	levels *lockboxsdk.ApproverLevels,
	spec lockboxv1beta1.ApprovalTemplateApproverLevels,
	specified specifiedApprovalTemplateApproverLevels,
) error {
	var err error
	if specified.level2 {
		levels.Level2, err = approvalTemplateApproverInfoFromLevel2("level2", spec.Level2)
		if err != nil {
			return err
		}
	}
	if specified.level3 {
		levels.Level3, err = approvalTemplateApproverInfoFromLevel3("level3", spec.Level3)
		if err != nil {
			return err
		}
	}
	return nil
}

func approvalTemplateApproverInfoFromLevel1(
	name string,
	level lockboxv1beta1.ApprovalTemplateApproverLevelsLevel1,
) (*lockboxsdk.ApproverInfo, error) {
	return approvalTemplateApproverInfo(name, level.ApproverType, level.ApproverId, level.DomainId)
}

func approvalTemplateApproverInfoFromLevel2(
	name string,
	level lockboxv1beta1.ApprovalTemplateApproverLevelsLevel2,
) (*lockboxsdk.ApproverInfo, error) {
	return approvalTemplateApproverInfo(name, level.ApproverType, level.ApproverId, level.DomainId)
}

func approvalTemplateApproverInfoFromLevel3(
	name string,
	level lockboxv1beta1.ApprovalTemplateApproverLevelsLevel3,
) (*lockboxsdk.ApproverInfo, error) {
	return approvalTemplateApproverInfo(name, level.ApproverType, level.ApproverId, level.DomainId)
}

func approvalTemplateApproverInfo(name string, approverType string, approverID string, domainID string) (*lockboxsdk.ApproverInfo, error) {
	approverType = strings.TrimSpace(approverType)
	approverID = strings.TrimSpace(approverID)
	if approverType == "" || approverID == "" {
		return nil, fmt.Errorf("ApprovalTemplate spec is invalid: approverLevels.%s.approverType and approverId are required", name)
	}
	info := &lockboxsdk.ApproverInfo{
		ApproverType: lockboxsdk.ApproverTypeEnum(approverType),
		ApproverId:   stringPointer(approverID),
	}
	if domainID = strings.TrimSpace(domainID); domainID != "" {
		info.DomainId = stringPointer(domainID)
	}
	return info, nil
}

func approvalTemplateApproverLevel1Specified(level lockboxv1beta1.ApprovalTemplateApproverLevelsLevel1) bool {
	return strings.TrimSpace(level.ApproverType) != "" ||
		strings.TrimSpace(level.ApproverId) != "" ||
		strings.TrimSpace(level.DomainId) != ""
}

func approvalTemplateApproverLevel2Specified(level lockboxv1beta1.ApprovalTemplateApproverLevelsLevel2) bool {
	return strings.TrimSpace(level.ApproverType) != "" ||
		strings.TrimSpace(level.ApproverId) != "" ||
		strings.TrimSpace(level.DomainId) != ""
}

func approvalTemplateApproverLevel3Specified(level lockboxv1beta1.ApprovalTemplateApproverLevelsLevel3) bool {
	return strings.TrimSpace(level.ApproverType) != "" ||
		strings.TrimSpace(level.ApproverId) != "" ||
		strings.TrimSpace(level.DomainId) != ""
}

type comparableApproverLevels struct {
	Level1 comparableApproverInfo
	Level2 comparableApproverInfo
	Level3 comparableApproverInfo
}

type comparableApproverInfo struct {
	Specified    bool
	ApproverType string
	ApproverID   string
	DomainID     string
}

func approvalTemplateApproverLevelsEqual(desired *lockboxsdk.ApproverLevels, current *lockboxsdk.ApproverLevels) bool {
	return comparableApprovalTemplateApproverLevels(desired) == comparableApprovalTemplateApproverLevels(current)
}

func comparableApprovalTemplateApproverLevels(levels *lockboxsdk.ApproverLevels) comparableApproverLevels {
	if levels == nil {
		return comparableApproverLevels{}
	}
	return comparableApproverLevels{
		Level1: comparableApprovalTemplateApproverInfo(levels.Level1),
		Level2: comparableApprovalTemplateApproverInfo(levels.Level2),
		Level3: comparableApprovalTemplateApproverInfo(levels.Level3),
	}
}

func comparableApprovalTemplateApproverInfo(info *lockboxsdk.ApproverInfo) comparableApproverInfo {
	if info == nil {
		return comparableApproverInfo{}
	}
	return comparableApproverInfo{
		Specified:    true,
		ApproverType: strings.TrimSpace(string(info.ApproverType)),
		ApproverID:   stringPointerValue(info.ApproverId),
		DomainID:     stringPointerValue(info.DomainId),
	}
}

func (c *approvalTemplateRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *lockboxv1beta1.ApprovalTemplate,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ApprovalTemplate runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *approvalTemplateRuntimeClient) Delete(ctx context.Context, resource *lockboxv1beta1.ApprovalTemplate) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("ApprovalTemplate runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *approvalTemplateRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *lockboxv1beta1.ApprovalTemplate,
) error {
	currentID := currentApprovalTemplateID(resource)
	if currentID == "" || c.hooks.Get.Call == nil {
		return nil
	}
	_, err := c.hooks.Get.Call(ctx, lockboxsdk.GetApprovalTemplateRequest{ApprovalTemplateId: stringPointer(currentID)})
	if err == nil || (!isApprovalTemplateAmbiguousNotFound(err) && !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()) {
		return nil
	}
	err = conservativeApprovalTemplateNotFoundError(err, "delete confirmation")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("ApprovalTemplate delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
}

func currentApprovalTemplateID(resource *lockboxv1beta1.ApprovalTemplate) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func wrapApprovalTemplateReadAndDeleteCalls(hooks *ApprovalTemplateRuntimeHooks) {
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request lockboxsdk.GetApprovalTemplateRequest) (lockboxsdk.GetApprovalTemplateResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeApprovalTemplateNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error) {
			return listApprovalTemplatePages(ctx, call, request)
		}
	}
	if hooks.Delete.Call != nil {
		call := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request lockboxsdk.DeleteApprovalTemplateRequest) (lockboxsdk.DeleteApprovalTemplateResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeApprovalTemplateNotFoundError(err, "delete")
		}
	}
}

func listApprovalTemplatePages(
	ctx context.Context,
	call func(context.Context, lockboxsdk.ListApprovalTemplatesRequest) (lockboxsdk.ListApprovalTemplatesResponse, error),
	request lockboxsdk.ListApprovalTemplatesRequest,
) (lockboxsdk.ListApprovalTemplatesResponse, error) {
	var combined lockboxsdk.ListApprovalTemplatesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return lockboxsdk.ListApprovalTemplatesResponse{}, conservativeApprovalTemplateNotFoundError(err, "list")
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

func handleApprovalTemplateDeleteError(resource *lockboxv1beta1.ApprovalTemplate, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return conservativeApprovalTemplateNotFoundError(err, "delete")
}

type approvalTemplateAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e approvalTemplateAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e approvalTemplateAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func conservativeApprovalTemplateNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if isApprovalTemplateAmbiguousNotFound(err) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return approvalTemplateAmbiguousNotFoundError{
		message:      fmt.Sprintf("ApprovalTemplate %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func isApprovalTemplateAmbiguousNotFound(err error) bool {
	var ambiguous approvalTemplateAmbiguousNotFoundError
	return errors.As(err, &ambiguous)
}

func approvalTemplateFromResponse(response any) (lockboxsdk.ApprovalTemplate, bool) {
	if current, ok := approvalTemplateFromDirectResponse(response); ok {
		return current, true
	}
	return approvalTemplateFromOperationResponse(response)
}

func approvalTemplateFromDirectResponse(response any) (lockboxsdk.ApprovalTemplate, bool) {
	switch current := response.(type) {
	case lockboxsdk.ApprovalTemplate:
		return current, true
	case *lockboxsdk.ApprovalTemplate:
		if current == nil {
			return lockboxsdk.ApprovalTemplate{}, false
		}
		return *current, true
	case lockboxsdk.ApprovalTemplateSummary:
		return approvalTemplateFromSummary(current), true
	case *lockboxsdk.ApprovalTemplateSummary:
		if current == nil {
			return lockboxsdk.ApprovalTemplate{}, false
		}
		return approvalTemplateFromSummary(*current), true
	default:
		return lockboxsdk.ApprovalTemplate{}, false
	}
}

func approvalTemplateFromOperationResponse(response any) (lockboxsdk.ApprovalTemplate, bool) {
	switch current := response.(type) {
	case lockboxsdk.CreateApprovalTemplateResponse:
		return current.ApprovalTemplate, true
	case *lockboxsdk.CreateApprovalTemplateResponse:
		if current == nil {
			return lockboxsdk.ApprovalTemplate{}, false
		}
		return current.ApprovalTemplate, true
	case lockboxsdk.GetApprovalTemplateResponse:
		return current.ApprovalTemplate, true
	case *lockboxsdk.GetApprovalTemplateResponse:
		if current == nil {
			return lockboxsdk.ApprovalTemplate{}, false
		}
		return current.ApprovalTemplate, true
	case lockboxsdk.UpdateApprovalTemplateResponse:
		return current.ApprovalTemplate, true
	case *lockboxsdk.UpdateApprovalTemplateResponse:
		if current == nil {
			return lockboxsdk.ApprovalTemplate{}, false
		}
		return current.ApprovalTemplate, true
	default:
		return lockboxsdk.ApprovalTemplate{}, false
	}
}

func approvalTemplateFromSummary(summary lockboxsdk.ApprovalTemplateSummary) lockboxsdk.ApprovalTemplate {
	return lockboxsdk.ApprovalTemplate(summary)
}

func stringPointer(value string) *string {
	return &value
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ error = approvalTemplateAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = approvalTemplateAmbiguousNotFoundError{}
