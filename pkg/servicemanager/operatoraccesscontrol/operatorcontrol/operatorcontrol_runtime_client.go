/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operatorcontrol

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	operatoraccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	operatoraccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/operatoraccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	operatorControlDeletePendingMessage      = "OCI OperatorControl delete is in progress"
	operatorControlPendingWriteDeleteMessage = "OCI OperatorControl create or update is still in progress; retaining finalizer"
)

type operatorControlOCIClient interface {
	CreateOperatorControl(context.Context, operatoraccesscontrolsdk.CreateOperatorControlRequest) (operatoraccesscontrolsdk.CreateOperatorControlResponse, error)
	GetOperatorControl(context.Context, operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error)
	ListOperatorControls(context.Context, operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error)
	UpdateOperatorControl(context.Context, operatoraccesscontrolsdk.UpdateOperatorControlRequest) (operatoraccesscontrolsdk.UpdateOperatorControlResponse, error)
	DeleteOperatorControl(context.Context, operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error)
}

type operatorControlRuntimeClient struct {
	delegate OperatorControlServiceClient
	hooks    OperatorControlRuntimeHooks
}

func init() {
	registerOperatorControlRuntimeHooksMutator(func(_ *OperatorControlServiceManager, hooks *OperatorControlRuntimeHooks) {
		applyOperatorControlRuntimeHooks(hooks)
	})
}

func applyOperatorControlRuntimeHooks(hooks *OperatorControlRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOperatorControlRuntimeSemantics()
	hooks.BuildCreateBody = buildOperatorControlCreateBody
	hooks.BuildUpdateBody = buildOperatorControlUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardOperatorControlExistingBeforeCreate
	hooks.List.Fields = operatorControlListFields()
	hooks.ParityHooks.NormalizeDesiredState = normalizeOperatorControlDesiredState
	hooks.Delete.Fields = operatorControlDeleteFields()
	wrapOperatorControlReadAndDeleteCalls(hooks)
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *operatoraccesscontrolv1beta1.OperatorControl, currentID string) (any, error) {
		return confirmOperatorControlDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleOperatorControlDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyOperatorControlDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OperatorControlServiceClient) OperatorControlServiceClient {
		return &operatorControlRuntimeClient{delegate: delegate, hooks: *hooks}
	})
}

func newOperatorControlServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client operatorControlOCIClient,
) OperatorControlServiceClient {
	hooks := newOperatorControlRuntimeHooksWithOCIClient(client)
	applyOperatorControlRuntimeHooks(&hooks)
	manager := &OperatorControlServiceManager{Log: log}
	delegate := defaultOperatorControlServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*operatoraccesscontrolv1beta1.OperatorControl](
			buildOperatorControlGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapOperatorControlGeneratedClient(hooks, delegate)
}

func newOperatorControlRuntimeHooksWithOCIClient(client operatorControlOCIClient) OperatorControlRuntimeHooks {
	return OperatorControlRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*operatoraccesscontrolv1beta1.OperatorControl]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*operatoraccesscontrolv1beta1.OperatorControl]{},
		StatusHooks:     generatedruntime.StatusHooks[*operatoraccesscontrolv1beta1.OperatorControl]{},
		ParityHooks:     generatedruntime.ParityHooks[*operatoraccesscontrolv1beta1.OperatorControl]{},
		Async:           generatedruntime.AsyncHooks[*operatoraccesscontrolv1beta1.OperatorControl]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*operatoraccesscontrolv1beta1.OperatorControl]{},
		Create: runtimeOperationHooks[operatoraccesscontrolsdk.CreateOperatorControlRequest, operatoraccesscontrolsdk.CreateOperatorControlResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateOperatorControlDetails", RequestName: "CreateOperatorControlDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.CreateOperatorControlRequest) (operatoraccesscontrolsdk.CreateOperatorControlResponse, error) {
				if client == nil {
					return operatoraccesscontrolsdk.CreateOperatorControlResponse{}, fmt.Errorf("operator control OCI client is nil")
				}
				return client.CreateOperatorControl(ctx, request)
			},
		},
		Get: runtimeOperationHooks[operatoraccesscontrolsdk.GetOperatorControlRequest, operatoraccesscontrolsdk.GetOperatorControlResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperatorControlId", RequestName: "operatorControlId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
				if client == nil {
					return operatoraccesscontrolsdk.GetOperatorControlResponse{}, fmt.Errorf("operator control OCI client is nil")
				}
				return client.GetOperatorControl(ctx, request)
			},
		},
		List: runtimeOperationHooks[operatoraccesscontrolsdk.ListOperatorControlsRequest, operatoraccesscontrolsdk.ListOperatorControlsResponse]{
			Fields: operatorControlListFields(),
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
				if client == nil {
					return operatoraccesscontrolsdk.ListOperatorControlsResponse{}, fmt.Errorf("operator control OCI client is nil")
				}
				return client.ListOperatorControls(ctx, request)
			},
		},
		Update: runtimeOperationHooks[operatoraccesscontrolsdk.UpdateOperatorControlRequest, operatoraccesscontrolsdk.UpdateOperatorControlResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "OperatorControlId", RequestName: "operatorControlId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateOperatorControlDetails", RequestName: "UpdateOperatorControlDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.UpdateOperatorControlRequest) (operatoraccesscontrolsdk.UpdateOperatorControlResponse, error) {
				if client == nil {
					return operatoraccesscontrolsdk.UpdateOperatorControlResponse{}, fmt.Errorf("operator control OCI client is nil")
				}
				return client.UpdateOperatorControl(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[operatoraccesscontrolsdk.DeleteOperatorControlRequest, operatoraccesscontrolsdk.DeleteOperatorControlResponse]{
			Fields: operatorControlDeleteFields(),
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error) {
				if client == nil {
					return operatoraccesscontrolsdk.DeleteOperatorControlResponse{}, fmt.Errorf("operator control OCI client is nil")
				}
				return client.DeleteOperatorControl(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OperatorControlServiceClient) OperatorControlServiceClient{},
	}
}

func newOperatorControlRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "operatoraccesscontrol",
		FormalSlug:    "operatorcontrol",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{
				string(operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
				string(operatoraccesscontrolsdk.OperatorControlLifecycleStatesAssigned),
				string(operatoraccesscontrolsdk.OperatorControlLifecycleStatesUnassigned),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(operatoraccesscontrolsdk.OperatorControlLifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "operatorControlName", "resourceType", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"operatorControlName",
				"approverGroupsList",
				"isFullyPreApproved",
				"description",
				"approversList",
				"preApprovedOpActionList",
				"emailIdList",
				"numberOfApprovers",
				"systemMessage",
				"freeformTags",
				"definedTags",
			},
			Mutable: []string{
				"operatorControlName",
				"approverGroupsList",
				"isFullyPreApproved",
				"description",
				"approversList",
				"preApprovedOpActionList",
				"emailIdList",
				"numberOfApprovers",
				"systemMessage",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId", "resourceType"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "OperatorControl", Action: "CreateOperatorControl"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "OperatorControl", Action: "UpdateOperatorControl"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "OperatorControl", Action: "DeleteOperatorControl"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func operatorControlListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.operatorControlName", "spec.operatorControlName", "operatorControlName"}},
		{FieldName: "ResourceType", RequestName: "resourceType", Contribution: "query", LookupPaths: []string{"status.resourceType", "spec.resourceType", "resourceType"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func operatorControlDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OperatorControlId", RequestName: "operatorControlId", Contribution: "path", PreferResourceID: true},
	}
}

func guardOperatorControlExistingBeforeCreate(
	_ context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("operator control resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.OperatorControlName) == "" ||
		strings.TrimSpace(resource.Spec.ResourceType) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if err := normalizeOperatorControlResourceType(resource); err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func normalizeOperatorControlDesiredState(resource *operatoraccesscontrolv1beta1.OperatorControl, _ any) {
	_ = normalizeOperatorControlResourceType(resource)
}

func normalizeOperatorControlResourceType(resource *operatoraccesscontrolv1beta1.OperatorControl) error {
	if resource == nil {
		return nil
	}
	resourceType, err := operatorControlResourceType(resource.Spec.ResourceType)
	if err != nil {
		return err
	}
	resource.Spec.ResourceType = string(resourceType)
	return nil
}

func buildOperatorControlCreateBody(
	_ context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	_ string,
) (any, error) {
	if resource == nil {
		return operatoraccesscontrolsdk.CreateOperatorControlDetails{}, fmt.Errorf("operator control resource is nil")
	}
	if err := validateOperatorControlSpec(resource.Spec); err != nil {
		return operatoraccesscontrolsdk.CreateOperatorControlDetails{}, err
	}
	resourceType, err := operatorControlResourceType(resource.Spec.ResourceType)
	if err != nil {
		return operatoraccesscontrolsdk.CreateOperatorControlDetails{}, err
	}

	details := operatoraccesscontrolsdk.CreateOperatorControlDetails{
		OperatorControlName: common.String(strings.TrimSpace(resource.Spec.OperatorControlName)),
		ApproverGroupsList:  slices.Clone(resource.Spec.ApproverGroupsList),
		IsFullyPreApproved:  common.Bool(resource.Spec.IsFullyPreApproved),
		ResourceType:        resourceType,
		CompartmentId:       common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
	}
	applyOperatorControlCreateOptionalFields(&details, resource.Spec)
	return details, nil
}

func applyOperatorControlCreateOptionalFields(
	details *operatoraccesscontrolsdk.CreateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	if value := strings.TrimSpace(spec.Description); value != "" {
		details.Description = common.String(value)
	}
	if spec.ApproversList != nil {
		details.ApproversList = slices.Clone(spec.ApproversList)
	}
	if spec.PreApprovedOpActionList != nil {
		details.PreApprovedOpActionList = slices.Clone(spec.PreApprovedOpActionList)
	}
	if spec.NumberOfApprovers != 0 {
		details.NumberOfApprovers = common.Int(spec.NumberOfApprovers)
	}
	if spec.EmailIdList != nil {
		details.EmailIdList = slices.Clone(spec.EmailIdList)
	}
	if value := strings.TrimSpace(spec.SystemMessage); value != "" {
		details.SystemMessage = common.String(value)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = operatorControlDefinedTags(spec.DefinedTags)
	}
}

func buildOperatorControlUpdateBody(
	_ context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return operatoraccesscontrolsdk.UpdateOperatorControlDetails{}, false, fmt.Errorf("operator control resource is nil")
	}
	if err := validateOperatorControlSpec(resource.Spec); err != nil {
		return operatoraccesscontrolsdk.UpdateOperatorControlDetails{}, false, err
	}
	current, ok := operatorControlFromResponse(currentResponse)
	if !ok {
		return operatoraccesscontrolsdk.UpdateOperatorControlDetails{}, false, fmt.Errorf("current operator control response does not expose an operator control body")
	}
	if err := validateOperatorControlCreateOnlyDrift(resource.Spec, current); err != nil {
		return operatoraccesscontrolsdk.UpdateOperatorControlDetails{}, false, err
	}

	details := operatoraccesscontrolsdk.UpdateOperatorControlDetails{
		OperatorControlName: common.String(strings.TrimSpace(resource.Spec.OperatorControlName)),
		ApproverGroupsList:  slices.Clone(resource.Spec.ApproverGroupsList),
		IsFullyPreApproved:  common.Bool(resource.Spec.IsFullyPreApproved),
	}
	updateNeeded := applyOperatorControlRequiredFieldUpdates(resource.Spec, current)
	updateNeeded = applyOperatorControlOptionalFieldUpdates(&details, resource.Spec, current) || updateNeeded
	return details, updateNeeded, nil
}

func applyOperatorControlRequiredFieldUpdates(
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
	current operatoraccesscontrolsdk.OperatorControl,
) bool {
	return strings.TrimSpace(spec.OperatorControlName) != stringPtrValue(current.OperatorControlName) ||
		!slices.Equal(spec.ApproverGroupsList, current.ApproverGroupsList) ||
		boolPtrValue(current.IsFullyPreApproved) != spec.IsFullyPreApproved
}

func applyOperatorControlOptionalFieldUpdates(
	details *operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
	current operatoraccesscontrolsdk.OperatorControl,
) bool {
	updateNeeded := false
	updateNeeded = applyOperatorControlOptionalString(&details.Description, spec.Description, current.Description) || updateNeeded
	updateNeeded = applyOperatorControlOptionalString(&details.SystemMessage, spec.SystemMessage, current.SystemMessage) || updateNeeded
	updateNeeded = applyOperatorControlOptionalStringSlice(&details.ApproversList, spec.ApproversList, current.ApproversList) || updateNeeded
	updateNeeded = applyOperatorControlOptionalStringSlice(&details.PreApprovedOpActionList, spec.PreApprovedOpActionList, current.PreApprovedOpActionList) || updateNeeded
	updateNeeded = applyOperatorControlOptionalStringSlice(&details.EmailIdList, spec.EmailIdList, current.EmailIdList) || updateNeeded
	updateNeeded = applyOperatorControlOptionalInt(&details.NumberOfApprovers, spec.NumberOfApprovers, current.NumberOfApprovers) || updateNeeded
	updateNeeded = applyOperatorControlFreeformTagsUpdate(details, spec, current) || updateNeeded
	updateNeeded = applyOperatorControlDefinedTagsUpdate(details, spec, current) || updateNeeded
	return updateNeeded
}

func applyOperatorControlOptionalString(target **string, desired string, current *string) bool {
	desired = strings.TrimSpace(desired)
	if desired == "" || desired == stringPtrValue(current) {
		return false
	}
	*target = common.String(desired)
	return true
}

func applyOperatorControlOptionalStringSlice(target *[]string, desired []string, current []string) bool {
	if desired == nil || slices.Equal(desired, current) {
		return false
	}
	*target = slices.Clone(desired)
	return true
}

func applyOperatorControlOptionalInt(target **int, desired int, current *int) bool {
	if desired == 0 || intPtrValue(current) == desired {
		return false
	}
	*target = common.Int(desired)
	return true
}

func applyOperatorControlFreeformTagsUpdate(
	details *operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
	current operatoraccesscontrolsdk.OperatorControl,
) bool {
	if spec.FreeformTags == nil || reflect.DeepEqual(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	details.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func applyOperatorControlDefinedTagsUpdate(
	details *operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
	current operatoraccesscontrolsdk.OperatorControl,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := operatorControlDefinedTags(spec.DefinedTags)
	if reflect.DeepEqual(desired, current.DefinedTags) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func validateOperatorControlSpec(spec operatoraccesscontrolv1beta1.OperatorControlSpec) error {
	switch {
	case strings.TrimSpace(spec.OperatorControlName) == "":
		return fmt.Errorf("operator control spec is invalid: operatorControlName is required")
	case spec.ApproverGroupsList == nil:
		return fmt.Errorf("operator control spec is invalid: approverGroupsList is required")
	case strings.TrimSpace(spec.CompartmentId) == "":
		return fmt.Errorf("operator control spec is invalid: compartmentId is required")
	}
	_, err := operatorControlResourceType(spec.ResourceType)
	return err
}

func validateOperatorControlCreateOnlyDrift(
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
	current operatoraccesscontrolsdk.OperatorControl,
) error {
	resourceType, err := operatorControlResourceType(spec.ResourceType)
	if err != nil {
		return err
	}
	var drift []string
	if strings.TrimSpace(spec.CompartmentId) != stringPtrValue(current.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if string(resourceType) != strings.TrimSpace(string(current.ResourceType)) {
		drift = append(drift, "resourceType")
	}
	if len(drift) != 0 {
		return fmt.Errorf("operator control create-only field drift is not supported: %s", strings.Join(drift, ", "))
	}
	return nil
}

func operatorControlResourceType(value string) (operatoraccesscontrolsdk.ResourceTypesEnum, error) {
	value = strings.TrimSpace(value)
	resourceType, ok := operatoraccesscontrolsdk.GetMappingResourceTypesEnum(value)
	if !ok {
		return "", fmt.Errorf("operator control spec is invalid: unsupported resourceType %q", value)
	}
	return resourceType, nil
}

func wrapOperatorControlReadAndDeleteCalls(hooks *OperatorControlRuntimeHooks) {
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeOperatorControlNotFoundError(err, "read")
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
			return listOperatorControlPages(ctx, call, request)
		}
	}
	if hooks.Delete.Call != nil {
		call := hooks.Delete.Call
		hooks.Delete.Call = func(ctx context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error) {
			response, err := call(ctx, request)
			return response, conservativeOperatorControlNotFoundError(err, "delete")
		}
	}
}

func listOperatorControlPages(
	ctx context.Context,
	call func(context.Context, operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error),
	request operatoraccesscontrolsdk.ListOperatorControlsRequest,
) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
	var combined operatoraccesscontrolsdk.ListOperatorControlsResponse
	seenPages := map[string]bool{}
	for {
		if err := recordOperatorControlListPage(request.Page, seenPages); err != nil {
			return operatoraccesscontrolsdk.ListOperatorControlsResponse{}, err
		}
		response, err := call(ctx, request)
		if err != nil {
			return operatoraccesscontrolsdk.ListOperatorControlsResponse{}, conservativeOperatorControlNotFoundError(err, "list")
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

func recordOperatorControlListPage(pageToken *string, seenPages map[string]bool) error {
	page := stringPtrValue(pageToken)
	if page == "" {
		return nil
	}
	if seenPages[page] {
		return fmt.Errorf("operator control list pagination repeated page token %q", page)
	}
	seenPages[page] = true
	return nil
}

func handleOperatorControlDeleteError(resource *operatoraccesscontrolv1beta1.OperatorControl, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return conservativeOperatorControlNotFoundError(err, "delete")
}

func confirmOperatorControlDeleteRead(
	ctx context.Context,
	hooks *OperatorControlRuntimeHooks,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("operator control delete confirmation requires runtime hooks")
	}
	currentID = strings.TrimSpace(currentID)
	if currentID == "" {
		currentID = currentOperatorControlID(resource)
	}
	if currentID == "" {
		return nil, fmt.Errorf("operator control delete confirmation requires a tracked operator control OCID")
	}
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("operator control delete confirmation requires a readable OCI operation")
	}
	response, err := hooks.Get.Call(ctx, operatoraccesscontrolsdk.GetOperatorControlRequest{OperatorControlId: common.String(currentID)})
	return operatorControlDeleteConfirmReadResponse(response, err)
}

func operatorControlDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if isOperatorControlAmbiguousNotFound(err) || errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return operatorControlAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

func applyOperatorControlDeleteOutcome(
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if outcome, err, handled := applyOperatorControlAuthShapedConfirmReadOutcome(resource, response); handled {
		return outcome, err
	}
	current, ok := operatorControlFromResponse(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	switch current.LifecycleState {
	case operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated,
		operatoraccesscontrolsdk.OperatorControlLifecycleStatesAssigned,
		operatoraccesscontrolsdk.OperatorControlLifecycleStatesUnassigned:
		if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !operatorControlDeleteAlreadyPending(resource) {
			return generatedruntime.DeleteOutcome{}, nil
		}
		if stage != generatedruntime.DeleteConfirmStageAfterRequest && stage != generatedruntime.DeleteConfirmStageAlreadyPending {
			return generatedruntime.DeleteOutcome{}, nil
		}
		markOperatorControlDeletePending(resource, current)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
}

func applyOperatorControlAuthShapedConfirmReadOutcome(
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	response any,
) (generatedruntime.DeleteOutcome, error, bool) {
	switch typed := response.(type) {
	case operatorControlAuthShapedConfirmRead:
		recordOperatorControlConfirmReadRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed, true
	case *operatorControlAuthShapedConfirmRead:
		if typed != nil {
			recordOperatorControlConfirmReadRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed, true
		}
	}
	return generatedruntime.DeleteOutcome{}, nil, false
}

func recordOperatorControlConfirmReadRequestID(
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	confirmRead operatorControlAuthShapedConfirmRead,
) {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, confirmRead)
	}
}

func (c *operatorControlRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("operator control runtime client is not configured")
	}
	if err := normalizeOperatorControlResourceType(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *operatorControlRuntimeClient) Delete(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("operator control runtime client is not configured")
	}
	if hasPendingOperatorControlWrite(resource) {
		markOperatorControlTerminating(resource, operatorControlPendingWriteDeleteMessage)
		return false, nil
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *operatorControlRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
) error {
	currentID := currentOperatorControlID(resource)
	if currentID == "" || c.hooks.Get.Call == nil {
		return nil
	}
	_, err := c.hooks.Get.Call(ctx, operatoraccesscontrolsdk.GetOperatorControlRequest{OperatorControlId: common.String(currentID)})
	if err == nil || (!isOperatorControlAmbiguousNotFound(err) && !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()) {
		return nil
	}
	err = conservativeOperatorControlNotFoundError(err, "delete confirmation")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("operator control delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
}

func hasPendingOperatorControlWrite(resource *operatoraccesscontrolv1beta1.OperatorControl) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func operatorControlDeleteAlreadyPending(resource *operatoraccesscontrolv1beta1.OperatorControl) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markOperatorControlDeletePending(
	resource *operatoraccesscontrolv1beta1.OperatorControl,
	current operatoraccesscontrolsdk.OperatorControl,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = operatorControlDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       string(current.LifecycleState),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         operatorControlDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", operatorControlDeletePendingMessage, loggerutil.OSOKLogger{})
}

func markOperatorControlTerminating(resource *operatoraccesscontrolv1beta1.OperatorControl, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func currentOperatorControlID(resource *operatoraccesscontrolv1beta1.OperatorControl) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

type operatorControlAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type operatorControlAuthShapedConfirmRead struct {
	err error
}

func (e operatorControlAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e operatorControlAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func (e operatorControlAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("operator control delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", e.err)
}

func (e operatorControlAuthShapedConfirmRead) Unwrap() error {
	return e.err
}

func (e operatorControlAuthShapedConfirmRead) GetOpcRequestID() string {
	return servicemanager.ErrorOpcRequestID(e.err)
}

func conservativeOperatorControlNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if isOperatorControlAmbiguousNotFound(err) {
		return err
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return operatorControlAmbiguousNotFoundError{
		message:      fmt.Sprintf("operator control %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error()),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func isOperatorControlAmbiguousNotFound(err error) bool {
	var ambiguous operatorControlAmbiguousNotFoundError
	return errors.As(err, &ambiguous)
}

func operatorControlFromResponse(response any) (operatoraccesscontrolsdk.OperatorControl, bool) {
	if current, ok := operatorControlFromDirectResponse(response); ok {
		return current, true
	}
	return operatorControlFromOperationResponse(response)
}

func operatorControlFromDirectResponse(response any) (operatoraccesscontrolsdk.OperatorControl, bool) {
	switch current := response.(type) {
	case operatoraccesscontrolsdk.OperatorControl:
		return current, true
	case *operatoraccesscontrolsdk.OperatorControl:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControl{}, false
		}
		return *current, true
	case operatoraccesscontrolsdk.OperatorControlSummary:
		return operatorControlFromSummary(current), true
	case *operatoraccesscontrolsdk.OperatorControlSummary:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControl{}, false
		}
		return operatorControlFromSummary(*current), true
	default:
		return operatoraccesscontrolsdk.OperatorControl{}, false
	}
}

func operatorControlFromOperationResponse(response any) (operatoraccesscontrolsdk.OperatorControl, bool) {
	switch current := response.(type) {
	case operatoraccesscontrolsdk.CreateOperatorControlResponse:
		return current.OperatorControl, true
	case *operatoraccesscontrolsdk.CreateOperatorControlResponse:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControl{}, false
		}
		return current.OperatorControl, true
	case operatoraccesscontrolsdk.GetOperatorControlResponse:
		return current.OperatorControl, true
	case *operatoraccesscontrolsdk.GetOperatorControlResponse:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControl{}, false
		}
		return current.OperatorControl, true
	case operatoraccesscontrolsdk.UpdateOperatorControlResponse:
		return current.OperatorControl, true
	case *operatoraccesscontrolsdk.UpdateOperatorControlResponse:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControl{}, false
		}
		return current.OperatorControl, true
	default:
		return operatoraccesscontrolsdk.OperatorControl{}, false
	}
}

func operatorControlFromSummary(summary operatoraccesscontrolsdk.OperatorControlSummary) operatoraccesscontrolsdk.OperatorControl {
	return operatoraccesscontrolsdk.OperatorControl{
		Id:                  summary.Id,
		OperatorControlName: summary.OperatorControlName,
		CompartmentId:       summary.CompartmentId,
		IsFullyPreApproved:  summary.IsFullyPreApproved,
		ResourceType:        summary.ResourceType,
		LifecycleState:      summary.LifecycleState,
		TimeOfCreation:      summary.TimeOfCreation,
		TimeOfModification:  summary.TimeOfModification,
		NumberOfApprovers:   summary.NumberOfApprovers,
		TimeOfDeletion:      summary.TimeOfDeletion,
		FreeformTags:        summary.FreeformTags,
		DefinedTags:         summary.DefinedTags,
	}
}

func operatorControlDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func intPtrValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

var _ OperatorControlServiceClient = (*operatorControlRuntimeClient)(nil)
var _ error = operatorControlAmbiguousNotFoundError{}
var _ error = operatorControlAuthShapedConfirmRead{}
var _ interface{ GetOpcRequestID() string } = operatorControlAmbiguousNotFoundError{}
var _ interface{ GetOpcRequestID() string } = operatorControlAuthShapedConfirmRead{}
