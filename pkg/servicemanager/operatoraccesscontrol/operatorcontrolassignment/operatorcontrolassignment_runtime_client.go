/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operatorcontrolassignment

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	operatoraccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	operatoraccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/operatoraccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type operatorControlAssignmentOCIClient interface {
	CreateOperatorControlAssignment(context.Context, operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse, error)
	GetOperatorControlAssignment(context.Context, operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error)
	ListOperatorControlAssignments(context.Context, operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error)
	UpdateOperatorControlAssignment(context.Context, operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error)
	DeleteOperatorControlAssignment(context.Context, operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error)
}

type ambiguousOperatorControlAssignmentNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousOperatorControlAssignmentNotFoundError) Error() string {
	return e.message
}

func (e ambiguousOperatorControlAssignmentNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerOperatorControlAssignmentRuntimeHooksMutator(func(_ *OperatorControlAssignmentServiceManager, hooks *OperatorControlAssignmentRuntimeHooks) {
		applyOperatorControlAssignmentRuntimeHooks(hooks)
	})
}

func applyOperatorControlAssignmentRuntimeHooks(hooks *OperatorControlAssignmentRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOperatorControlAssignmentRuntimeSemantics()
	hooks.BuildCreateBody = buildOperatorControlAssignmentCreateBody
	hooks.BuildUpdateBody = buildOperatorControlAssignmentUpdateBody
	hooks.StatusHooks.ClearProjectedStatus = clearOperatorControlAssignmentProjectedStatus
	hooks.StatusHooks.RestoreStatus = restoreOperatorControlAssignmentProjectedStatus
	hooks.List.Fields = operatorControlAssignmentListFields()
	hooks.DeleteHooks.HandleError = handleOperatorControlAssignmentDeleteError
	wrapOperatorControlAssignmentReadAndDeleteCalls(hooks)
	configureOperatorControlAssignmentDeleteConfirmationHooks(hooks)
	wrapOperatorControlAssignmentDeleteConfirmation(hooks)
}

func newOperatorControlAssignmentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client operatorControlAssignmentOCIClient,
) OperatorControlAssignmentServiceClient {
	hooks := newOperatorControlAssignmentRuntimeHooksWithOCIClient(client)
	applyOperatorControlAssignmentRuntimeHooks(&hooks)
	delegate := defaultOperatorControlAssignmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*operatoraccesscontrolv1beta1.OperatorControlAssignment](
			buildOperatorControlAssignmentGeneratedRuntimeConfig(&OperatorControlAssignmentServiceManager{Log: log}, hooks),
		),
	}
	return wrapOperatorControlAssignmentGeneratedClient(hooks, delegate)
}

func newOperatorControlAssignmentRuntimeHooksWithOCIClient(client operatorControlAssignmentOCIClient) OperatorControlAssignmentRuntimeHooks {
	return OperatorControlAssignmentRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*operatoraccesscontrolv1beta1.OperatorControlAssignment]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*operatoraccesscontrolv1beta1.OperatorControlAssignment]{},
		StatusHooks:     generatedruntime.StatusHooks[*operatoraccesscontrolv1beta1.OperatorControlAssignment]{},
		ParityHooks:     generatedruntime.ParityHooks[*operatoraccesscontrolv1beta1.OperatorControlAssignment]{},
		Async:           generatedruntime.AsyncHooks[*operatoraccesscontrolv1beta1.OperatorControlAssignment]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*operatoraccesscontrolv1beta1.OperatorControlAssignment]{},
		Create: runtimeOperationHooks[operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest, operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateOperatorControlAssignmentDetails", RequestName: "CreateOperatorControlAssignmentDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse, error) {
				return client.CreateOperatorControlAssignment(ctx, request)
			},
		},
		Get: runtimeOperationHooks[operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest, operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperatorControlAssignmentId", RequestName: "operatorControlAssignmentId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
				return client.GetOperatorControlAssignment(ctx, request)
			},
		},
		List: runtimeOperationHooks[operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest, operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse]{
			Fields: operatorControlAssignmentListFields(),
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error) {
				return client.ListOperatorControlAssignments(ctx, request)
			},
		},
		Update: runtimeOperationHooks[operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest, operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperatorControlAssignmentId", RequestName: "operatorControlAssignmentId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateOperatorControlAssignmentDetails", RequestName: "UpdateOperatorControlAssignmentDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error) {
				return client.UpdateOperatorControlAssignment(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest, operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperatorControlAssignmentId", RequestName: "operatorControlAssignmentId", Contribution: "path", PreferResourceID: true}, {FieldName: "Description", RequestName: "description", Contribution: "query"}},
			Call: func(ctx context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error) {
				return client.DeleteOperatorControlAssignment(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OperatorControlAssignmentServiceClient) OperatorControlAssignmentServiceClient{},
	}
}

func newOperatorControlAssignmentRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "operatoraccesscontrol",
		FormalSlug:    "operatorcontrolassignment",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesCreated)},
			UpdatingStates:     []string{string(operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesUpdating)},
			ActiveStates:       []string{string(operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesDeleting)},
			TerminalStates: []string{string(operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"operatorControlId",
				"resourceId",
				"resourceName",
				"resourceType",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"isEnforcedAlways",
				"timeAssignmentFrom",
				"timeAssignmentTo",
				"comment",
				"isLogForwarded",
				"remoteSyslogServerAddress",
				"remoteSyslogServerPort",
				"remoteSyslogServerCACert",
				"isHypervisorLogForwarded",
				"isAutoApproveDuringMaintenance",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"operatorControlId",
				"resourceId",
				"resourceName",
				"resourceType",
				"resourceCompartmentId",
				"compartmentId",
			},
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

func operatorControlAssignmentListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "ResourceName", RequestName: "resourceName", Contribution: "query", LookupPaths: []string{"status.resourceName", "spec.resourceName", "resourceName"}},
		{FieldName: "ResourceType", RequestName: "resourceType", Contribution: "query", LookupPaths: []string{"status.resourceType", "spec.resourceType", "resourceType"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildOperatorControlAssignmentCreateBody(
	_ context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("operatorControlAssignment resource is nil")
	}
	if err := validateOperatorControlAssignmentSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	resourceType, err := operatorControlAssignmentResourceType(spec.ResourceType)
	if err != nil {
		return nil, err
	}
	body := operatoraccesscontrolsdk.CreateOperatorControlAssignmentDetails{
		OperatorControlId:     common.String(strings.TrimSpace(spec.OperatorControlId)),
		ResourceId:            common.String(strings.TrimSpace(spec.ResourceId)),
		ResourceName:          common.String(strings.TrimSpace(spec.ResourceName)),
		ResourceType:          resourceType,
		ResourceCompartmentId: common.String(strings.TrimSpace(spec.ResourceCompartmentId)),
		CompartmentId:         common.String(strings.TrimSpace(spec.CompartmentId)),
		IsEnforcedAlways:      common.Bool(spec.IsEnforcedAlways),
	}
	if err := applyOperatorControlAssignmentCreateOptionalFields(&body, spec); err != nil {
		return nil, err
	}
	return body, nil
}

func applyOperatorControlAssignmentCreateOptionalFields(
	body *operatoraccesscontrolsdk.CreateOperatorControlAssignmentDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec,
) error {
	from, err := optionalOperatorControlAssignmentSDKTime(spec.TimeAssignmentFrom, "timeAssignmentFrom")
	if err != nil {
		return err
	}
	body.TimeAssignmentFrom = from
	to, err := optionalOperatorControlAssignmentSDKTime(spec.TimeAssignmentTo, "timeAssignmentTo")
	if err != nil {
		return err
	}
	body.TimeAssignmentTo = to

	body.Comment = optionalOperatorControlAssignmentString(spec.Comment)
	if spec.IsLogForwarded {
		body.IsLogForwarded = common.Bool(true)
	}
	body.RemoteSyslogServerAddress = optionalOperatorControlAssignmentString(spec.RemoteSyslogServerAddress)
	if spec.RemoteSyslogServerPort != 0 {
		body.RemoteSyslogServerPort = common.Int(spec.RemoteSyslogServerPort)
	}
	body.RemoteSyslogServerCACert = optionalOperatorControlAssignmentString(spec.RemoteSyslogServerCACert)
	if spec.IsHypervisorLogForwarded {
		body.IsHypervisorLogForwarded = common.Bool(true)
	}
	if spec.IsAutoApproveDuringMaintenance {
		body.IsAutoApproveDuringMaintenance = common.Bool(true)
	}
	body.FreeformTags = cloneOperatorControlAssignmentStringMap(spec.FreeformTags)
	body.DefinedTags = operatorControlAssignmentDefinedTagsFromSpec(spec.DefinedTags)
	return nil
}

func buildOperatorControlAssignmentUpdateBody(
	_ context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("operatorControlAssignment resource is nil")
	}
	current, ok := operatorControlAssignmentFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current OperatorControlAssignment response does not expose an OperatorControlAssignment body")
	}
	if err := validateOperatorControlAssignmentSpec(resource.Spec); err != nil {
		return nil, false, err
	}

	body := operatoraccesscontrolsdk.UpdateOperatorControlAssignmentDetails{}
	updateNeeded, err := applyOperatorControlAssignmentUpdateFields(&body, resource.Spec, current)
	if err != nil {
		return nil, false, err
	}
	if updateNeeded {
		body.IsEnforcedAlways = common.Bool(resource.Spec.IsEnforcedAlways)
	}
	return body, updateNeeded, nil
}

func applyOperatorControlAssignmentUpdateFields(
	body *operatoraccesscontrolsdk.UpdateOperatorControlAssignmentDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec,
	current operatoraccesscontrolsdk.OperatorControlAssignment,
) (bool, error) {
	updateNeeded := current.IsEnforcedAlways == nil || *current.IsEnforcedAlways != spec.IsEnforcedAlways

	timeUpdated, err := applyOperatorControlAssignmentTimeUpdateFields(body, spec, current)
	if err != nil {
		return false, err
	}
	optionalUpdated := applyOperatorControlAssignmentOptionalUpdateFields(body, spec, current)
	tagsUpdated := applyOperatorControlAssignmentTagUpdates(body, spec, current)
	return updateNeeded || timeUpdated || optionalUpdated || tagsUpdated, nil
}

func applyOperatorControlAssignmentTimeUpdateFields(
	body *operatoraccesscontrolsdk.UpdateOperatorControlAssignmentDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec,
	current operatoraccesscontrolsdk.OperatorControlAssignment,
) (bool, error) {
	updateNeeded := false
	from, err := optionalOperatorControlAssignmentSDKTime(spec.TimeAssignmentFrom, "timeAssignmentFrom")
	if err != nil {
		return false, err
	}
	if !operatorControlAssignmentTimeEqual(current.TimeAssignmentFrom, from) {
		body.TimeAssignmentFrom = from
		updateNeeded = true
	}
	to, err := optionalOperatorControlAssignmentSDKTime(spec.TimeAssignmentTo, "timeAssignmentTo")
	if err != nil {
		return false, err
	}
	if !operatorControlAssignmentTimeEqual(current.TimeAssignmentTo, to) {
		body.TimeAssignmentTo = to
		updateNeeded = true
	}
	return updateNeeded, nil
}

func applyOperatorControlAssignmentOptionalUpdateFields(
	body *operatoraccesscontrolsdk.UpdateOperatorControlAssignmentDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec,
	current operatoraccesscontrolsdk.OperatorControlAssignment,
) bool {
	updateNeeded := applyOperatorControlAssignmentOptionalStringUpdate(&body.Comment, current.Comment, spec.Comment)
	if operatorControlAssignmentOptionalBoolDrift(current.IsLogForwarded, spec.IsLogForwarded) {
		body.IsLogForwarded = common.Bool(spec.IsLogForwarded)
		updateNeeded = true
	}
	if applyOperatorControlAssignmentOptionalStringUpdate(&body.RemoteSyslogServerAddress, current.RemoteSyslogServerAddress, spec.RemoteSyslogServerAddress) {
		updateNeeded = true
	}
	if operatorControlAssignmentOptionalIntDrift(current.RemoteSyslogServerPort, spec.RemoteSyslogServerPort) {
		body.RemoteSyslogServerPort = common.Int(spec.RemoteSyslogServerPort)
		updateNeeded = true
	}
	if applyOperatorControlAssignmentOptionalStringUpdate(&body.RemoteSyslogServerCACert, current.RemoteSyslogServerCACert, spec.RemoteSyslogServerCACert) {
		updateNeeded = true
	}
	if operatorControlAssignmentOptionalBoolDrift(current.IsHypervisorLogForwarded, spec.IsHypervisorLogForwarded) {
		body.IsHypervisorLogForwarded = common.Bool(spec.IsHypervisorLogForwarded)
		updateNeeded = true
	}
	if operatorControlAssignmentOptionalBoolDrift(current.IsAutoApproveDuringMaintenance, spec.IsAutoApproveDuringMaintenance) {
		body.IsAutoApproveDuringMaintenance = common.Bool(spec.IsAutoApproveDuringMaintenance)
		updateNeeded = true
	}
	return updateNeeded
}

func applyOperatorControlAssignmentTagUpdates(
	body *operatoraccesscontrolsdk.UpdateOperatorControlAssignmentDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec,
	current operatoraccesscontrolsdk.OperatorControlAssignment,
) bool {
	updateNeeded := false
	if desired, ok := operatorControlAssignmentDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		body.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := operatorControlAssignmentDesiredDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		body.DefinedTags = desired
		updateNeeded = true
	}
	return updateNeeded
}

func clearOperatorControlAssignmentProjectedStatus(resource *operatoraccesscontrolv1beta1.OperatorControlAssignment) any {
	if resource == nil {
		return nil
	}
	baseline := resource.Status
	resource.Status = operatoraccesscontrolv1beta1.OperatorControlAssignmentStatus{
		OsokStatus: baseline.OsokStatus,
	}
	return baseline
}

func restoreOperatorControlAssignmentProjectedStatus(resource *operatoraccesscontrolv1beta1.OperatorControlAssignment, baseline any) {
	if resource == nil {
		return
	}
	status, ok := baseline.(operatoraccesscontrolv1beta1.OperatorControlAssignmentStatus)
	if !ok {
		return
	}
	status.OsokStatus = resource.Status.OsokStatus
	resource.Status = status
}

func wrapOperatorControlAssignmentReadAndDeleteCalls(hooks *OperatorControlAssignmentRuntimeHooks) {
	getCall := hooks.Get.Call
	if getCall != nil {
		hooks.Get.Call = func(ctx context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativeOperatorControlAssignmentNotFoundError(err, "read")
		}
	}

	listCall := hooks.List.Call
	if listCall != nil {
		hooks.List.Call = func(ctx context.Context, request operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error) {
			return listOperatorControlAssignmentPages(ctx, listCall, request)
		}
	}

	deleteCall := hooks.Delete.Call
	if deleteCall != nil {
		hooks.Delete.Call = func(ctx context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeOperatorControlAssignmentNotFoundError(err, "delete")
		}
	}
}

func listOperatorControlAssignmentPages(
	ctx context.Context,
	list func(context.Context, operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error),
	request operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest,
) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error) {
	var combined operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse
	seenPages := map[string]bool{}
	for {
		response, err := list(ctx, request)
		if err != nil {
			return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{}, conservativeOperatorControlAssignmentNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.Items = append(combined.Items, response.Items...)

		nextPage := strings.TrimSpace(operatorControlAssignmentStringValue(response.OpcNextPage))
		if nextPage == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		if seenPages[nextPage] {
			return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{}, fmt.Errorf("operatorControlAssignment list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = true
		combined.OpcNextPage = response.OpcNextPage
		request.Page = response.OpcNextPage
	}
}

func handleOperatorControlAssignmentDeleteError(resource *operatoraccesscontrolv1beta1.OperatorControlAssignment, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func configureOperatorControlAssignmentDeleteConfirmationHooks(hooks *OperatorControlAssignmentRuntimeHooks) {
	if hooks == nil {
		return
	}
	getOperatorControlAssignment := hooks.Get.Call
	listOperatorControlAssignments := hooks.List.Call
	if getOperatorControlAssignment != nil || listOperatorControlAssignments != nil {
		hooks.DeleteHooks.ConfirmRead = func(
			ctx context.Context,
			resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
			currentID string,
		) (any, error) {
			return confirmOperatorControlAssignmentDeleteRead(
				ctx,
				resource,
				currentID,
				getOperatorControlAssignment,
				listOperatorControlAssignments,
			)
		}
	}
	hooks.DeleteHooks.ApplyOutcome = applyOperatorControlAssignmentDeleteConfirmOutcome
}

func confirmOperatorControlAssignmentDeleteRead(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	currentID string,
	getOperatorControlAssignment func(context.Context, operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error),
	listOperatorControlAssignments func(context.Context, operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error),
) (any, error) {
	assignmentID := strings.TrimSpace(currentID)
	if assignmentID == "" {
		assignmentID = trackedOperatorControlAssignmentID(resource)
	}
	if assignmentID != "" && getOperatorControlAssignment != nil {
		response, err := getOperatorControlAssignment(ctx, operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest{
			OperatorControlAssignmentId: common.String(assignmentID),
		})
		return operatorControlAssignmentDeleteConfirmReadResponse(response, err)
	}
	return confirmOperatorControlAssignmentDeleteReadByList(ctx, resource, listOperatorControlAssignments)
}

func confirmOperatorControlAssignmentDeleteReadByList(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	listOperatorControlAssignments func(context.Context, operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error),
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("operatorControlAssignment delete confirmation resource is nil")
	}
	if listOperatorControlAssignments == nil {
		return nil, fmt.Errorf("operatorControlAssignment delete confirmation has no readable OCI operation")
	}
	response, err := listOperatorControlAssignments(ctx, operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		ResourceName:  optionalOperatorControlAssignmentString(resource.Spec.ResourceName),
		ResourceType:  optionalOperatorControlAssignmentString(resource.Spec.ResourceType),
	})
	converted, err := operatorControlAssignmentDeleteConfirmReadResponse(response, err)
	if err != nil {
		return nil, err
	}
	if _, ok := converted.(operatorControlAssignmentDeleteAuthShapedConfirmRead); ok {
		return converted, nil
	}

	listResponse, ok := converted.(operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse)
	if !ok {
		return nil, fmt.Errorf("operatorControlAssignment delete confirmation list returned %T", converted)
	}
	return singleOperatorControlAssignmentDeleteMatch(resource, listResponse)
}

func singleOperatorControlAssignmentDeleteMatch(
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	response operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse,
) (any, error) {
	var matches []operatoraccesscontrolsdk.OperatorControlAssignmentSummary
	for _, item := range response.Items {
		if operatorControlAssignmentSummaryMatchesResource(item, resource) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, errorutil.NotFoundOciError{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			OpcRequestID:   operatorControlAssignmentStringValue(response.OpcRequestId),
			Description:    "OperatorControlAssignment delete confirmation did not find a matching assignment",
		}
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("operatorControlAssignment delete confirmation found %d matching assignments", len(matches))
	}
}

func operatorControlAssignmentSummaryMatchesResource(
	summary operatoraccesscontrolsdk.OperatorControlAssignmentSummary,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
) bool {
	if resource == nil {
		return false
	}
	spec := resource.Spec
	return operatorControlAssignmentStringPtrEqual(summary.OperatorControlId, spec.OperatorControlId) &&
		operatorControlAssignmentStringPtrEqual(summary.ResourceId, spec.ResourceId) &&
		operatorControlAssignmentStringPtrEqual(summary.ResourceName, spec.ResourceName) &&
		operatorControlAssignmentStringPtrEqual(summary.CompartmentId, spec.CompartmentId) &&
		strings.TrimSpace(string(summary.ResourceType)) == strings.TrimSpace(spec.ResourceType)
}

func operatorControlAssignmentDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if isAmbiguousOperatorControlAssignmentNotFound(err) {
		return operatorControlAssignmentDeleteAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

type operatorControlAssignmentDeleteAuthShapedConfirmRead struct {
	err error
}

func (e operatorControlAssignmentDeleteAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("operatorControlAssignment delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e operatorControlAssignmentDeleteAuthShapedConfirmRead) GetOpcRequestID() string {
	return servicemanager.ErrorOpcRequestID(e.err)
}

func applyOperatorControlAssignmentDeleteConfirmOutcome(
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case operatorControlAssignmentDeleteAuthShapedConfirmRead:
		return operatorControlAssignmentAuthShapedConfirmReadOutcome(resource, typed)
	case *operatorControlAssignmentDeleteAuthShapedConfirmRead:
		if typed != nil {
			return operatorControlAssignmentAuthShapedConfirmReadOutcome(resource, *typed)
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func operatorControlAssignmentAuthShapedConfirmReadOutcome(
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	err operatorControlAssignmentDeleteAuthShapedConfirmRead,
) (generatedruntime.DeleteOutcome, error) {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, err
}

func wrapOperatorControlAssignmentDeleteConfirmation(hooks *OperatorControlAssignmentRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getOperatorControlAssignment := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OperatorControlAssignmentServiceClient) OperatorControlAssignmentServiceClient {
		return operatorControlAssignmentDeleteConfirmationClient{
			delegate:                     delegate,
			getOperatorControlAssignment: getOperatorControlAssignment,
		}
	})
}

type operatorControlAssignmentDeleteConfirmationClient struct {
	delegate                     OperatorControlAssignmentServiceClient
	getOperatorControlAssignment func(context.Context, operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error)
}

func (c operatorControlAssignmentDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c operatorControlAssignmentDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c operatorControlAssignmentDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
) error {
	if c.getOperatorControlAssignment == nil || resource == nil {
		return nil
	}
	assignmentID := trackedOperatorControlAssignmentID(resource)
	if assignmentID == "" {
		return nil
	}
	_, err := c.getOperatorControlAssignment(ctx, operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest{
		OperatorControlAssignmentId: common.String(assignmentID),
	})
	if err == nil || !isAmbiguousOperatorControlAssignmentNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("operatorControlAssignment delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedOperatorControlAssignmentID(resource *operatoraccesscontrolv1beta1.OperatorControlAssignment) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func conservativeOperatorControlAssignmentNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !isAmbiguousOperatorControlAssignmentNotFound(err) {
		return err
	}
	message := fmt.Sprintf("operatorControlAssignment %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	return ambiguousOperatorControlAssignmentNotFoundError{message: message, opcRequestID: errorutil.OpcRequestID(err)}
}

func isAmbiguousOperatorControlAssignmentNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousOperatorControlAssignmentNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func validateOperatorControlAssignmentSpec(spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec) error {
	var missing []string
	if strings.TrimSpace(spec.OperatorControlId) == "" {
		missing = append(missing, "operatorControlId")
	}
	if strings.TrimSpace(spec.ResourceId) == "" {
		missing = append(missing, "resourceId")
	}
	if strings.TrimSpace(spec.ResourceName) == "" {
		missing = append(missing, "resourceName")
	}
	if strings.TrimSpace(spec.ResourceType) == "" {
		missing = append(missing, "resourceType")
	}
	if strings.TrimSpace(spec.ResourceCompartmentId) == "" {
		missing = append(missing, "resourceCompartmentId")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if len(missing) != 0 {
		return fmt.Errorf("operatorControlAssignment spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	_, err := operatorControlAssignmentResourceType(spec.ResourceType)
	return err
}

func operatorControlAssignmentResourceType(value string) (operatoraccesscontrolsdk.ResourceTypesEnum, error) {
	resourceType, ok := operatoraccesscontrolsdk.GetMappingResourceTypesEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported operatorControlAssignment resourceType %q", value)
	}
	return resourceType, nil
}

func optionalOperatorControlAssignmentSDKTime(value string, fieldName string) (*common.SDKTime, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", fieldName, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func optionalOperatorControlAssignmentString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func applyOperatorControlAssignmentOptionalStringUpdate(target **string, current *string, desired string) bool {
	desired = strings.TrimSpace(desired)
	if operatorControlAssignmentStringPtrEqual(current, desired) {
		return false
	}
	*target = optionalOperatorControlAssignmentString(desired)
	return true
}

func operatorControlAssignmentOptionalBoolDrift(current *bool, desired bool) bool {
	if desired {
		return current == nil || !*current
	}
	return current != nil && *current
}

func operatorControlAssignmentTimeEqual(current *common.SDKTime, desired *common.SDKTime) bool {
	switch {
	case current == nil && desired == nil:
		return true
	case current == nil || desired == nil:
		return false
	default:
		return current.Equal(desired.Time)
	}
}

func operatorControlAssignmentOptionalIntDrift(current *int, desired int) bool {
	if current == nil {
		return desired != 0
	}
	return *current != desired
}

func operatorControlAssignmentStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(operatorControlAssignmentStringValue(current)) == strings.TrimSpace(desired)
}

func operatorControlAssignmentStringMapsEqual(current map[string]string, desired map[string]string) bool {
	if len(current) == 0 && len(desired) == 0 {
		return true
	}
	return reflect.DeepEqual(current, desired)
}

func operatorControlAssignmentDefinedTagsEqual(current map[string]map[string]interface{}, desired map[string]map[string]interface{}) bool {
	if len(current) == 0 && len(desired) == 0 {
		return true
	}
	return reflect.DeepEqual(current, desired)
}

func cloneOperatorControlAssignmentStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func operatorControlAssignmentDesiredFreeformTagsUpdate(desired map[string]string, current map[string]string) (map[string]string, bool) {
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if operatorControlAssignmentStringMapsEqual(current, desired) {
		return nil, false
	}
	if len(desired) == 0 {
		return map[string]string{}, true
	}
	return cloneOperatorControlAssignmentStringMap(desired), true
}

func operatorControlAssignmentDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&source)
}

func operatorControlAssignmentDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	desired := operatorControlAssignmentDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if operatorControlAssignmentDefinedTagsEqual(current, desired) {
		return nil, false
	}
	if len(desired) == 0 {
		return map[string]map[string]interface{}{}, true
	}
	return desired, true
}

func operatorControlAssignmentFromResponse(response any) (operatoraccesscontrolsdk.OperatorControlAssignment, bool) {
	if current, ok := operatorControlAssignmentFromWriteResponse(response); ok {
		return current, true
	}
	if current, ok := operatorControlAssignmentFromReadResponse(response); ok {
		return current, true
	}
	return operatorControlAssignmentFromListItem(response)
}

func operatorControlAssignmentFromWriteResponse(response any) (operatoraccesscontrolsdk.OperatorControlAssignment, bool) {
	switch current := response.(type) {
	case operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse:
		return current.OperatorControlAssignment, true
	case *operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
		}
		return current.OperatorControlAssignment, true
	case operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse:
		return current.OperatorControlAssignment, true
	case *operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
		}
		return current.OperatorControlAssignment, true
	default:
		return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
	}
}

func operatorControlAssignmentFromReadResponse(response any) (operatoraccesscontrolsdk.OperatorControlAssignment, bool) {
	switch current := response.(type) {
	case operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse:
		return current.OperatorControlAssignment, true
	case *operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
		}
		return current.OperatorControlAssignment, true
	case operatoraccesscontrolsdk.OperatorControlAssignment:
		return current, true
	case *operatoraccesscontrolsdk.OperatorControlAssignment:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
		}
		return *current, true
	default:
		return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
	}
}

func operatorControlAssignmentFromListItem(response any) (operatoraccesscontrolsdk.OperatorControlAssignment, bool) {
	switch current := response.(type) {
	case operatoraccesscontrolsdk.OperatorControlAssignmentSummary:
		return operatorControlAssignmentFromSummary(current), true
	case *operatoraccesscontrolsdk.OperatorControlAssignmentSummary:
		if current == nil {
			return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
		}
		return operatorControlAssignmentFromSummary(*current), true
	default:
		return operatoraccesscontrolsdk.OperatorControlAssignment{}, false
	}
}

func operatorControlAssignmentFromSummary(summary operatoraccesscontrolsdk.OperatorControlAssignmentSummary) operatoraccesscontrolsdk.OperatorControlAssignment {
	return operatoraccesscontrolsdk.OperatorControlAssignment{
		Id:                        summary.Id,
		OperatorControlId:         summary.OperatorControlId,
		ResourceId:                summary.ResourceId,
		ResourceName:              summary.ResourceName,
		CompartmentId:             summary.CompartmentId,
		ResourceType:              summary.ResourceType,
		TimeAssignmentFrom:        summary.TimeAssignmentFrom,
		TimeAssignmentTo:          summary.TimeAssignmentTo,
		IsEnforcedAlways:          summary.IsEnforcedAlways,
		LifecycleState:            summary.LifecycleState,
		LifecycleDetails:          summary.LifecycleDetails,
		TimeOfAssignment:          summary.TimeOfAssignment,
		IsLogForwarded:            summary.IsLogForwarded,
		RemoteSyslogServerAddress: summary.RemoteSyslogServerAddress,
		RemoteSyslogServerPort:    summary.RemoteSyslogServerPort,
		IsHypervisorLogForwarded:  summary.IsHypervisorLogForwarded,
		OpControlName:             summary.OpControlName,
		ErrorCode:                 summary.ErrorCode,
		ErrorMessage:              summary.ErrorMessage,
		FreeformTags:              cloneOperatorControlAssignmentStringMap(summary.FreeformTags),
		DefinedTags:               cloneOperatorControlAssignmentDefinedTags(summary.DefinedTags),
	}
}

func cloneOperatorControlAssignmentDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func operatorControlAssignmentStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
