/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package processset

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

const processSetKind = "ProcessSet"

type processSetOCIClient interface {
	CreateProcessSet(context.Context, stackmonitoringsdk.CreateProcessSetRequest) (stackmonitoringsdk.CreateProcessSetResponse, error)
	GetProcessSet(context.Context, stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error)
	ListProcessSets(context.Context, stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error)
	UpdateProcessSet(context.Context, stackmonitoringsdk.UpdateProcessSetRequest) (stackmonitoringsdk.UpdateProcessSetResponse, error)
	DeleteProcessSet(context.Context, stackmonitoringsdk.DeleteProcessSetRequest) (stackmonitoringsdk.DeleteProcessSetResponse, error)
}

type processSetIdentity struct {
	compartmentID string
	displayName   string
}

type processSetResourceBody struct {
	Id             *string
	CompartmentId  *string
	LifecycleState string
	DisplayName    *string
	Specification  *stackmonitoringsdk.ProcessSetSpecification
	FreeformTags   map[string]string
	DefinedTags    map[string]map[string]interface{}
}

type processSetAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

type processSetDeletePreflightClient struct {
	delegate ProcessSetServiceClient
	get      func(context.Context, stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error)
}

func (e processSetAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e processSetAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerProcessSetRuntimeHooksMutator(func(_ *ProcessSetServiceManager, hooks *ProcessSetRuntimeHooks) {
		applyProcessSetRuntimeHooks(hooks)
	})
}

func applyProcessSetRuntimeHooks(hooks *ProcessSetRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedProcessSetRuntimeSemantics()
	hooks.BuildCreateBody = buildProcessSetCreateBody
	hooks.BuildUpdateBody = buildProcessSetUpdateBody
	hooks.Identity.Resolve = resolveProcessSetIdentity
	hooks.Create.Fields = processSetCreateFields()
	hooks.Get.Fields = processSetGetFields()
	hooks.List.Fields = processSetListFields()
	hooks.Update.Fields = processSetUpdateFields()
	hooks.Delete.Fields = processSetDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listProcessSetsAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleProcessSetDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ProcessSetServiceClient) ProcessSetServiceClient {
		return processSetDeletePreflightClient{delegate: delegate, get: hooks.Get.Call}
	})
}

func newProcessSetServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client processSetOCIClient,
) ProcessSetServiceClient {
	hooks := newProcessSetRuntimeHooksWithOCIClient(client)
	applyProcessSetRuntimeHooks(&hooks)
	manager := &ProcessSetServiceManager{Log: log}
	delegate := defaultProcessSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.ProcessSet](
			buildProcessSetGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapProcessSetGeneratedClient(hooks, delegate)
}

func newProcessSetRuntimeHooksWithOCIClient(client processSetOCIClient) ProcessSetRuntimeHooks {
	return ProcessSetRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*stackmonitoringv1beta1.ProcessSet]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*stackmonitoringv1beta1.ProcessSet]{},
		StatusHooks:     generatedruntime.StatusHooks[*stackmonitoringv1beta1.ProcessSet]{},
		ParityHooks:     generatedruntime.ParityHooks[*stackmonitoringv1beta1.ProcessSet]{},
		Async:           generatedruntime.AsyncHooks[*stackmonitoringv1beta1.ProcessSet]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*stackmonitoringv1beta1.ProcessSet]{},
		Create: runtimeOperationHooks[stackmonitoringsdk.CreateProcessSetRequest, stackmonitoringsdk.CreateProcessSetResponse]{
			Fields: processSetCreateFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.CreateProcessSetRequest) (stackmonitoringsdk.CreateProcessSetResponse, error) {
				if client == nil {
					return stackmonitoringsdk.CreateProcessSetResponse{}, fmt.Errorf("processSet OCI client is nil")
				}
				return client.CreateProcessSet(ctx, request)
			},
		},
		Get: runtimeOperationHooks[stackmonitoringsdk.GetProcessSetRequest, stackmonitoringsdk.GetProcessSetResponse]{
			Fields: processSetGetFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
				if client == nil {
					return stackmonitoringsdk.GetProcessSetResponse{}, fmt.Errorf("processSet OCI client is nil")
				}
				return client.GetProcessSet(ctx, request)
			},
		},
		List: runtimeOperationHooks[stackmonitoringsdk.ListProcessSetsRequest, stackmonitoringsdk.ListProcessSetsResponse]{
			Fields: processSetListFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error) {
				if client == nil {
					return stackmonitoringsdk.ListProcessSetsResponse{}, fmt.Errorf("processSet OCI client is nil")
				}
				return client.ListProcessSets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[stackmonitoringsdk.UpdateProcessSetRequest, stackmonitoringsdk.UpdateProcessSetResponse]{
			Fields: processSetUpdateFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.UpdateProcessSetRequest) (stackmonitoringsdk.UpdateProcessSetResponse, error) {
				if client == nil {
					return stackmonitoringsdk.UpdateProcessSetResponse{}, fmt.Errorf("processSet OCI client is nil")
				}
				return client.UpdateProcessSet(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[stackmonitoringsdk.DeleteProcessSetRequest, stackmonitoringsdk.DeleteProcessSetResponse]{
			Fields: processSetDeleteFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.DeleteProcessSetRequest) (stackmonitoringsdk.DeleteProcessSetResponse, error) {
				if client == nil {
					return stackmonitoringsdk.DeleteProcessSetResponse{}, fmt.Errorf("processSet OCI client is nil")
				}
				return client.DeleteProcessSet(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ProcessSetServiceClient) ProcessSetServiceClient{},
	}
}

func reviewedProcessSetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "stackmonitoring",
		FormalSlug:        "processset",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(stackmonitoringsdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(stackmonitoringsdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(stackmonitoringsdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(stackmonitoringsdk.LifecycleStateDeleting)},
			TerminalStates: []string{string(stackmonitoringsdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "specification", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: processSetKind, Action: "CreateProcessSet"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: processSetKind, Action: "UpdateProcessSet"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: processSetKind, Action: "DeleteProcessSet"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: processSetKind, Action: "GetProcessSet"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: processSetKind, Action: "GetProcessSet"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: processSetKind, Action: "GetProcessSet"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func processSetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateProcessSetDetails", RequestName: "CreateProcessSetDetails", Contribution: "body"},
	}
}

func processSetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ProcessSetId", RequestName: "processSetId", Contribution: "path", PreferResourceID: true},
	}
}

func processSetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func processSetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ProcessSetId", RequestName: "processSetId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateProcessSetDetails", RequestName: "UpdateProcessSetDetails", Contribution: "body"},
	}
}

func processSetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ProcessSetId", RequestName: "processSetId", Contribution: "path", PreferResourceID: true},
	}
}

func buildProcessSetCreateBody(_ context.Context, resource *stackmonitoringv1beta1.ProcessSet, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("processSet resource is nil")
	}
	if err := validateProcessSetSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := stackmonitoringsdk.CreateProcessSetDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		Specification: processSetSpecification(resource.Spec.Specification),
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneProcessSetStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = processSetDefinedTags(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildProcessSetUpdateBody(
	_ context.Context,
	resource *stackmonitoringv1beta1.ProcessSet,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("processSet resource is nil")
	}
	if err := validateProcessSetSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := processSetBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current processSet response does not expose a processSet body")
	}

	body := stackmonitoringsdk.UpdateProcessSetDetails{
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		Specification: processSetSpecification(resource.Spec.Specification),
	}
	updateNeeded := !processSetStringPtrEqual(current.DisplayName, strings.TrimSpace(resource.Spec.DisplayName)) ||
		!processSetSpecificationsEqual(current.Specification, body.Specification)

	if resource.Spec.FreeformTags != nil {
		desired := cloneProcessSetStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := processSetDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateProcessSetSpec(spec stackmonitoringv1beta1.ProcessSetSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if len(spec.Specification.Items) == 0 {
		missing = append(missing, "specification.items")
	}
	if len(missing) > 0 {
		return fmt.Errorf("processSet spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func resolveProcessSetIdentity(resource *stackmonitoringv1beta1.ProcessSet) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("processSet resource is nil")
	}
	var missing []string
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("processSet identity is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return processSetIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}, nil
}

func processSetSpecification(spec stackmonitoringv1beta1.ProcessSetSpecification) *stackmonitoringsdk.ProcessSetSpecification {
	items := make([]stackmonitoringsdk.ProcessSetSpecificationDetails, 0, len(spec.Items))
	for _, item := range spec.Items {
		details := stackmonitoringsdk.ProcessSetSpecificationDetails{}
		if item.Label != "" {
			details.Label = common.String(item.Label)
		}
		if item.ProcessCommand != "" {
			details.ProcessCommand = common.String(item.ProcessCommand)
		}
		if item.ProcessUser != "" {
			details.ProcessUser = common.String(item.ProcessUser)
		}
		if item.ProcessLineRegexPattern != "" {
			details.ProcessLineRegexPattern = common.String(item.ProcessLineRegexPattern)
		}
		items = append(items, details)
	}
	return &stackmonitoringsdk.ProcessSetSpecification{Items: items}
}

func processSetBodyFromResponse(response any) (processSetResourceBody, bool) {
	if processSet, ok := sdkProcessSetFromResponse(response); ok {
		return processSetBodyFromSDKProcessSet(processSet), true
	}
	if summary, ok := sdkProcessSetSummaryFromResponse(response); ok {
		return processSetBodyFromSDKSummary(summary), true
	}
	return processSetResourceBody{}, false
}

func sdkProcessSetFromResponse(response any) (stackmonitoringsdk.ProcessSet, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.CreateProcessSetResponse:
		return current.ProcessSet, true
	case *stackmonitoringsdk.CreateProcessSetResponse:
		return sdkProcessSetFromCreateResponse(current)
	case stackmonitoringsdk.GetProcessSetResponse:
		return current.ProcessSet, true
	case *stackmonitoringsdk.GetProcessSetResponse:
		return sdkProcessSetFromGetResponse(current)
	case stackmonitoringsdk.UpdateProcessSetResponse:
		return current.ProcessSet, true
	case *stackmonitoringsdk.UpdateProcessSetResponse:
		return sdkProcessSetFromUpdateResponse(current)
	case stackmonitoringsdk.ProcessSet:
		return current, true
	case *stackmonitoringsdk.ProcessSet:
		if current == nil {
			return stackmonitoringsdk.ProcessSet{}, false
		}
		return *current, true
	default:
		return stackmonitoringsdk.ProcessSet{}, false
	}
}

func sdkProcessSetFromCreateResponse(response *stackmonitoringsdk.CreateProcessSetResponse) (stackmonitoringsdk.ProcessSet, bool) {
	if response == nil {
		return stackmonitoringsdk.ProcessSet{}, false
	}
	return response.ProcessSet, true
}

func sdkProcessSetFromGetResponse(response *stackmonitoringsdk.GetProcessSetResponse) (stackmonitoringsdk.ProcessSet, bool) {
	if response == nil {
		return stackmonitoringsdk.ProcessSet{}, false
	}
	return response.ProcessSet, true
}

func sdkProcessSetFromUpdateResponse(response *stackmonitoringsdk.UpdateProcessSetResponse) (stackmonitoringsdk.ProcessSet, bool) {
	if response == nil {
		return stackmonitoringsdk.ProcessSet{}, false
	}
	return response.ProcessSet, true
}

func sdkProcessSetSummaryFromResponse(response any) (stackmonitoringsdk.ProcessSetSummary, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.ProcessSetSummary:
		return current, true
	case *stackmonitoringsdk.ProcessSetSummary:
		if current == nil {
			return stackmonitoringsdk.ProcessSetSummary{}, false
		}
		return *current, true
	default:
		return stackmonitoringsdk.ProcessSetSummary{}, false
	}
}

func processSetBodyFromSDKProcessSet(processSet stackmonitoringsdk.ProcessSet) processSetResourceBody {
	return processSetResourceBody{
		Id:             processSet.Id,
		CompartmentId:  processSet.CompartmentId,
		LifecycleState: string(processSet.LifecycleState),
		DisplayName:    processSet.DisplayName,
		Specification:  processSet.Specification,
		FreeformTags:   cloneProcessSetStringMap(processSet.FreeformTags),
		DefinedTags:    cloneProcessSetDefinedTagMap(processSet.DefinedTags),
	}
}

func processSetBodyFromSDKSummary(summary stackmonitoringsdk.ProcessSetSummary) processSetResourceBody {
	return processSetResourceBody{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		LifecycleState: string(summary.LifecycleState),
		DisplayName:    summary.DisplayName,
		Specification:  summary.Specification,
		FreeformTags:   cloneProcessSetStringMap(summary.FreeformTags),
		DefinedTags:    cloneProcessSetDefinedTagMap(summary.DefinedTags),
	}
}

func listProcessSetsAllPages(
	call func(context.Context, stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error),
) func(context.Context, stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error) {
	return func(ctx context.Context, request stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error) {
		var combined stackmonitoringsdk.ListProcessSetsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return stackmonitoringsdk.ListProcessSetsResponse{}, err
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
}

func handleProcessSetDeleteError(resource *stackmonitoringv1beta1.ProcessSet, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	ambiguous := processSetAmbiguousNotFound("delete", err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, ambiguous.GetOpcRequestID())
	}
	return ambiguous
}

func (c processSetDeletePreflightClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.ProcessSet,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("processSet runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c processSetDeletePreflightClient) Delete(ctx context.Context, resource *stackmonitoringv1beta1.ProcessSet) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("processSet runtime client is not configured")
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c processSetDeletePreflightClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *stackmonitoringv1beta1.ProcessSet,
) error {
	if c.get == nil || resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) == "" {
		return nil
	}

	_, err := c.get(ctx, stackmonitoringsdk.GetProcessSetRequest{
		ProcessSetId: common.String(strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))),
	})
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	ambiguous := processSetAmbiguousNotFound("delete confirmation read", err)
	servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, ambiguous.GetOpcRequestID())
	return ambiguous
}

func processSetAmbiguousNotFound(operation string, err error) processSetAmbiguousNotFoundError {
	message := fmt.Sprintf("processSet %s returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed", strings.TrimSpace(operation))
	if err != nil {
		message = fmt.Sprintf("%s: %s", message, err.Error())
	}
	return processSetAmbiguousNotFoundError{
		message:      message,
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func processSetSpecificationsEqual(got *stackmonitoringsdk.ProcessSetSpecification, want *stackmonitoringsdk.ProcessSetSpecification) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return reflect.DeepEqual(got.Items, want.Items)
}

func cloneProcessSetStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneProcessSetDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func processSetDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func processSetStringPtrEqual(got *string, want string) bool {
	return got != nil && *got == want
}
