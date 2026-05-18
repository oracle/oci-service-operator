/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package onpremconnector

import (
	"context"
	"fmt"
	"reflect"
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

const onPremConnectorKind = "OnPremConnector"

type onPremConnectorOCIClient interface {
	CreateOnPremConnector(context.Context, datasafesdk.CreateOnPremConnectorRequest) (datasafesdk.CreateOnPremConnectorResponse, error)
	GetOnPremConnector(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error)
	ListOnPremConnectors(context.Context, datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error)
	UpdateOnPremConnector(context.Context, datasafesdk.UpdateOnPremConnectorRequest) (datasafesdk.UpdateOnPremConnectorResponse, error)
	DeleteOnPremConnector(context.Context, datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error)
}

type onPremConnectorIdentity struct {
	compartmentID string
	displayName   string
}

type onPremConnectorRuntimeClient struct {
	delegate OnPremConnectorServiceClient
	get      func(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error)
	list     func(context.Context, datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error)
}

type onPremConnectorAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e onPremConnectorAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e onPremConnectorAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerOnPremConnectorRuntimeHooksMutator(func(_ *OnPremConnectorServiceManager, hooks *OnPremConnectorRuntimeHooks) {
		applyOnPremConnectorRuntimeHooks(hooks)
	})
}

func applyOnPremConnectorRuntimeHooks(hooks *OnPremConnectorRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = onPremConnectorRuntimeSemantics()
	hooks.BuildCreateBody = buildOnPremConnectorCreateBody
	hooks.BuildUpdateBody = buildOnPremConnectorUpdateBody
	hooks.Identity.Resolve = resolveOnPremConnectorIdentity
	hooks.Identity.RecordPath = recordOnPremConnectorPathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardOnPremConnectorExistingBeforeCreate
	hooks.Create.Fields = onPremConnectorCreateFields()
	hooks.Get.Fields = onPremConnectorGetFields()
	hooks.List.Fields = onPremConnectorListFields()
	hooks.Update.Fields = onPremConnectorUpdateFields()
	hooks.Delete.Fields = onPremConnectorDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listOnPremConnectorsAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleOnPremConnectorDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyOnPremConnectorDeleteOutcome
	hooks.StatusHooks.ProjectStatus = projectOnPremConnectorStatus
	hooks.StatusHooks.MarkTerminating = markOnPremConnectorTerminating
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OnPremConnectorServiceClient) OnPremConnectorServiceClient {
		return onPremConnectorRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
}

func newOnPremConnectorServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client onPremConnectorOCIClient,
) OnPremConnectorServiceClient {
	hooks := newOnPremConnectorRuntimeHooksWithOCIClient(client)
	applyOnPremConnectorRuntimeHooks(&hooks)
	manager := &OnPremConnectorServiceManager{Log: log}
	delegate := defaultOnPremConnectorServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.OnPremConnector](
			buildOnPremConnectorGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapOnPremConnectorGeneratedClient(hooks, delegate)
}

func newOnPremConnectorRuntimeHooksWithOCIClient(client onPremConnectorOCIClient) OnPremConnectorRuntimeHooks {
	return OnPremConnectorRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.OnPremConnector]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.OnPremConnector]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.OnPremConnector]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.OnPremConnector]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.OnPremConnector]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.OnPremConnector]{},
		Create: runtimeOperationHooks[datasafesdk.CreateOnPremConnectorRequest, datasafesdk.CreateOnPremConnectorResponse]{
			Fields: onPremConnectorCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateOnPremConnectorRequest) (datasafesdk.CreateOnPremConnectorResponse, error) {
				if client == nil {
					return datasafesdk.CreateOnPremConnectorResponse{}, fmt.Errorf("%s OCI client is nil", onPremConnectorKind)
				}
				return client.CreateOnPremConnector(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetOnPremConnectorRequest, datasafesdk.GetOnPremConnectorResponse]{
			Fields: onPremConnectorGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
				if client == nil {
					return datasafesdk.GetOnPremConnectorResponse{}, fmt.Errorf("%s OCI client is nil", onPremConnectorKind)
				}
				return client.GetOnPremConnector(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListOnPremConnectorsRequest, datasafesdk.ListOnPremConnectorsResponse]{
			Fields: onPremConnectorListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error) {
				if client == nil {
					return datasafesdk.ListOnPremConnectorsResponse{}, fmt.Errorf("%s OCI client is nil", onPremConnectorKind)
				}
				return client.ListOnPremConnectors(ctx, request)
			},
		},
		Update: runtimeOperationHooks[datasafesdk.UpdateOnPremConnectorRequest, datasafesdk.UpdateOnPremConnectorResponse]{
			Fields: onPremConnectorUpdateFields(),
			Call: func(ctx context.Context, request datasafesdk.UpdateOnPremConnectorRequest) (datasafesdk.UpdateOnPremConnectorResponse, error) {
				if client == nil {
					return datasafesdk.UpdateOnPremConnectorResponse{}, fmt.Errorf("%s OCI client is nil", onPremConnectorKind)
				}
				return client.UpdateOnPremConnector(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteOnPremConnectorRequest, datasafesdk.DeleteOnPremConnectorResponse]{
			Fields: onPremConnectorDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
				if client == nil {
					return datasafesdk.DeleteOnPremConnectorResponse{}, fmt.Errorf("%s OCI client is nil", onPremConnectorKind)
				}
				return client.DeleteOnPremConnector(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OnPremConnectorServiceClient) OnPremConnectorServiceClient{},
	}
}

func onPremConnectorRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "onpremconnector",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.OnPremConnectorLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.OnPremConnectorLifecycleStateUpdating)},
			ActiveStates: []string{
				string(datasafesdk.OnPremConnectorLifecycleStateActive),
				string(datasafesdk.OnPremConnectorLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:        "required",
			PendingStates: []string{string(datasafesdk.OnPremConnectorLifecycleStateDeleting)},
			TerminalStates: []string{
				string(datasafesdk.OnPremConnectorLifecycleStateDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: onPremConnectorKind, Action: "CreateOnPremConnector"}},
			Update: []generatedruntime.Hook{{Helper: "generatedruntime.Update", EntityType: onPremConnectorKind, Action: "UpdateOnPremConnector"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: onPremConnectorKind, Action: "DeleteOnPremConnector"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func onPremConnectorCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateOnPremConnectorDetails", RequestName: "CreateOnPremConnectorDetails", Contribution: "body"},
	}
}

func onPremConnectorGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OnPremConnectorId", RequestName: "onPremConnectorId", Contribution: "path", PreferResourceID: true},
	}
}

func onPremConnectorListFields() []generatedruntime.RequestField {
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
		{FieldName: "OnPremConnectorId", RequestName: "onPremConnectorId", Contribution: "query", PreferResourceID: true},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
	}
}

func onPremConnectorUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OnPremConnectorId", RequestName: "onPremConnectorId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateOnPremConnectorDetails", RequestName: "UpdateOnPremConnectorDetails", Contribution: "body"},
	}
}

func onPremConnectorDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OnPremConnectorId", RequestName: "onPremConnectorId", Contribution: "path", PreferResourceID: true},
	}
}

func buildOnPremConnectorCreateBody(_ context.Context, resource *datasafev1beta1.OnPremConnector, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", onPremConnectorKind)
	}
	return datasafesdk.CreateOnPremConnectorDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   onPremConnectorOptionalString(resource.Spec.DisplayName),
		Description:   onPremConnectorOptionalString(resource.Spec.Description),
		FreeformTags:  onPremConnectorStringMap(resource.Spec.FreeformTags),
		DefinedTags:   onPremConnectorDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func buildOnPremConnectorUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.OnPremConnector,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return datasafesdk.UpdateOnPremConnectorDetails{}, false, fmt.Errorf("%s resource is nil", onPremConnectorKind)
	}
	current, ok := onPremConnectorStatusProjectionFromResponse(currentResponse)
	if !ok {
		current = onPremConnectorStatusProjectionFromResource(resource)
	}

	details := datasafesdk.UpdateOnPremConnectorDetails{}
	updateNeeded := false
	updateNeeded = applyOnPremConnectorStringUpdates(&details, resource.Spec, current) || updateNeeded
	updateNeeded = applyOnPremConnectorTagUpdates(&details, resource.Spec, current) || updateNeeded
	if !updateNeeded {
		return datasafesdk.UpdateOnPremConnectorDetails{}, false, nil
	}
	return details, true, nil
}

func applyOnPremConnectorStringUpdates(
	details *datasafesdk.UpdateOnPremConnectorDetails,
	spec datasafev1beta1.OnPremConnectorSpec,
	current onPremConnectorStatusProjection,
) bool {
	updateNeeded := false
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" && displayName != current.DisplayName {
		details.DisplayName = common.String(displayName)
		updateNeeded = true
	}
	if description := strings.TrimSpace(spec.Description); description != "" && description != current.Description {
		details.Description = common.String(description)
		updateNeeded = true
	}
	return updateNeeded
}

func applyOnPremConnectorTagUpdates(
	details *datasafesdk.UpdateOnPremConnectorDetails,
	spec datasafev1beta1.OnPremConnectorSpec,
	current onPremConnectorStatusProjection,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil && !reflect.DeepEqual(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = onPremConnectorStringMap(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil && !reflect.DeepEqual(spec.DefinedTags, current.DefinedTags) {
		details.DefinedTags = onPremConnectorDefinedTags(spec.DefinedTags)
		updateNeeded = true
	}
	return updateNeeded
}

func resolveOnPremConnectorIdentity(resource *datasafev1beta1.OnPremConnector) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", onPremConnectorKind)
	}
	return onPremConnectorIdentityFromSpec(resource), nil
}

func onPremConnectorIdentityFromSpec(resource *datasafev1beta1.OnPremConnector) onPremConnectorIdentity {
	if resource == nil {
		return onPremConnectorIdentity{}
	}
	return onPremConnectorIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}
}

func onPremConnectorIdentityForDelete(resource *datasafev1beta1.OnPremConnector) onPremConnectorIdentity {
	if resource == nil {
		return onPremConnectorIdentity{}
	}
	return onPremConnectorIdentity{
		compartmentID: firstOnPremConnectorString(resource.Status.CompartmentId, resource.Spec.CompartmentId),
		displayName:   firstOnPremConnectorString(resource.Status.DisplayName, resource.Spec.DisplayName),
	}
}

func recordOnPremConnectorPathIdentity(resource *datasafev1beta1.OnPremConnector, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(onPremConnectorIdentity)
	if !ok {
		return
	}
	if typed.compartmentID != "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
}

func guardOnPremConnectorExistingBeforeCreate(_ context.Context, resource *datasafev1beta1.OnPremConnector) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity := onPremConnectorIdentityFromSpec(resource)
	if identity.compartmentID == "" || identity.displayName == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func listOnPremConnectorsAllPages(
	call func(context.Context, datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error),
) func(context.Context, datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error) {
	return func(ctx context.Context, request datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error) {
		var combined datasafesdk.ListOnPremConnectorsResponse
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

func handleOnPremConnectorDeleteError(resource *datasafev1beta1.OnPremConnector, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := onPremConnectorAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func applyOnPremConnectorDeleteOutcome(
	resource *datasafev1beta1.OnPremConnector,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := onPremConnectorLifecycleState(response)
	if state == "" {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if onPremConnectorTerminalDeleteState(state) {
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && onPremConnectorPendingWriteState(state) {
		markOnPremConnectorPendingDelete(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest {
		markOnPremConnectorTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markOnPremConnectorTerminating(resource *datasafev1beta1.OnPremConnector, _ any) {
	markOnPremConnectorDeletePending(resource, "", "")
}

func markOnPremConnectorPendingDelete(resource *datasafev1beta1.OnPremConnector, response any) {
	state := onPremConnectorLifecycleState(response)
	message := "OCI resource has a pending write lifecycle state; waiting before delete"
	if state != "" {
		message = fmt.Sprintf("OCI resource is %s; waiting before delete", state)
	}
	markOnPremConnectorDeletePending(resource, message, state)
}

func markOnPremConnectorDeletePending(resource *datasafev1beta1.OnPremConnector, message string, rawStatus string) {
	if resource == nil {
		return
	}
	if strings.TrimSpace(message) == "" {
		message = "OCI resource delete is in progress"
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       strings.TrimSpace(rawStatus),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}, loggerutil.OSOKLogger{})
}

func onPremConnectorLifecycleState(response any) string {
	if current, ok := onPremConnectorFromResponse(response); ok {
		return strings.ToUpper(string(current.LifecycleState))
	}
	if summary, ok := onPremConnectorSummaryFromResponse(response); ok {
		return strings.ToUpper(string(summary.LifecycleState))
	}
	return ""
}

func onPremConnectorTerminalDeleteState(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), string(datasafesdk.OnPremConnectorLifecycleStateDeleted))
}

func onPremConnectorPendingWriteState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case string(datasafesdk.OnPremConnectorLifecycleStateCreating),
		string(datasafesdk.OnPremConnectorLifecycleStateUpdating):
		return true
	default:
		return false
	}
}

func projectOnPremConnectorStatus(resource *datasafev1beta1.OnPremConnector, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", onPremConnectorKind)
	}
	projected, ok := onPremConnectorStatusProjectionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.OnPremConnectorStatus{
		OsokStatus:       osokStatus,
		Id:               projected.Id,
		DisplayName:      projected.DisplayName,
		CompartmentId:    projected.CompartmentId,
		TimeCreated:      projected.TimeCreated,
		LifecycleState:   projected.LifecycleState,
		Description:      projected.Description,
		LifecycleDetails: projected.LifecycleDetails,
		FreeformTags:     onPremConnectorStringMap(projected.FreeformTags),
		DefinedTags:      onPremConnectorCloneSharedTags(projected.DefinedTags),
		SystemTags:       onPremConnectorCloneSharedTags(projected.SystemTags),
		AvailableVersion: projected.AvailableVersion,
		CreatedVersion:   projected.CreatedVersion,
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

type onPremConnectorStatusProjection struct {
	Id               string
	DisplayName      string
	CompartmentId    string
	TimeCreated      string
	LifecycleState   string
	Description      string
	LifecycleDetails string
	FreeformTags     map[string]string
	DefinedTags      map[string]shared.MapValue
	SystemTags       map[string]shared.MapValue
	AvailableVersion string
	CreatedVersion   string
}

func onPremConnectorStatusProjectionFromResponse(response any) (onPremConnectorStatusProjection, bool) {
	if current, ok := onPremConnectorFromResponse(response); ok {
		return onPremConnectorStatusProjectionFromSDK(current), true
	}
	if summary, ok := onPremConnectorSummaryFromResponse(response); ok {
		return onPremConnectorStatusProjectionFromSummary(summary), true
	}
	return onPremConnectorStatusProjection{}, false
}

func onPremConnectorStatusProjectionFromResource(resource *datasafev1beta1.OnPremConnector) onPremConnectorStatusProjection {
	if resource == nil {
		return onPremConnectorStatusProjection{}
	}
	return onPremConnectorStatusProjection{
		Id:               strings.TrimSpace(resource.Status.Id),
		DisplayName:      strings.TrimSpace(resource.Status.DisplayName),
		CompartmentId:    strings.TrimSpace(resource.Status.CompartmentId),
		TimeCreated:      resource.Status.TimeCreated,
		LifecycleState:   strings.TrimSpace(resource.Status.LifecycleState),
		Description:      strings.TrimSpace(resource.Status.Description),
		LifecycleDetails: strings.TrimSpace(resource.Status.LifecycleDetails),
		FreeformTags:     onPremConnectorStringMap(resource.Status.FreeformTags),
		DefinedTags:      onPremConnectorCloneSharedTags(resource.Status.DefinedTags),
		SystemTags:       onPremConnectorCloneSharedTags(resource.Status.SystemTags),
		AvailableVersion: strings.TrimSpace(resource.Status.AvailableVersion),
		CreatedVersion:   strings.TrimSpace(resource.Status.CreatedVersion),
	}
}

func onPremConnectorFromResponse(response any) (datasafesdk.OnPremConnector, bool) {
	switch current := response.(type) {
	case datasafesdk.GetOnPremConnectorResponse:
		return current.OnPremConnector, true
	case *datasafesdk.GetOnPremConnectorResponse:
		if current == nil {
			return datasafesdk.OnPremConnector{}, false
		}
		return current.OnPremConnector, true
	case datasafesdk.CreateOnPremConnectorResponse:
		return current.OnPremConnector, true
	case *datasafesdk.CreateOnPremConnectorResponse:
		if current == nil {
			return datasafesdk.OnPremConnector{}, false
		}
		return current.OnPremConnector, true
	case datasafesdk.OnPremConnector:
		return current, true
	case *datasafesdk.OnPremConnector:
		if current == nil {
			return datasafesdk.OnPremConnector{}, false
		}
		return *current, true
	default:
		return datasafesdk.OnPremConnector{}, false
	}
}

func onPremConnectorSummaryFromResponse(response any) (datasafesdk.OnPremConnectorSummary, bool) {
	switch current := response.(type) {
	case datasafesdk.OnPremConnectorSummary:
		return current, true
	case *datasafesdk.OnPremConnectorSummary:
		if current == nil {
			return datasafesdk.OnPremConnectorSummary{}, false
		}
		return *current, true
	default:
		return datasafesdk.OnPremConnectorSummary{}, false
	}
}

func onPremConnectorStatusProjectionFromSDK(current datasafesdk.OnPremConnector) onPremConnectorStatusProjection {
	return onPremConnectorStatusProjection{
		Id:               onPremConnectorStringValue(current.Id),
		DisplayName:      onPremConnectorStringValue(current.DisplayName),
		CompartmentId:    onPremConnectorStringValue(current.CompartmentId),
		TimeCreated:      onPremConnectorSDKTimeString(current.TimeCreated),
		LifecycleState:   string(current.LifecycleState),
		Description:      onPremConnectorStringValue(current.Description),
		LifecycleDetails: onPremConnectorStringValue(current.LifecycleDetails),
		FreeformTags:     onPremConnectorStringMap(current.FreeformTags),
		DefinedTags:      onPremConnectorSharedTags(current.DefinedTags),
		SystemTags:       onPremConnectorSharedTags(current.SystemTags),
		AvailableVersion: onPremConnectorStringValue(current.AvailableVersion),
		CreatedVersion:   onPremConnectorStringValue(current.CreatedVersion),
	}
}

func onPremConnectorStatusProjectionFromSummary(current datasafesdk.OnPremConnectorSummary) onPremConnectorStatusProjection {
	return onPremConnectorStatusProjection{
		Id:               onPremConnectorStringValue(current.Id),
		DisplayName:      onPremConnectorStringValue(current.DisplayName),
		CompartmentId:    onPremConnectorStringValue(current.CompartmentId),
		TimeCreated:      onPremConnectorSDKTimeString(current.TimeCreated),
		LifecycleState:   string(current.LifecycleState),
		Description:      onPremConnectorStringValue(current.Description),
		LifecycleDetails: onPremConnectorStringValue(current.LifecycleDetails),
		FreeformTags:     onPremConnectorStringMap(current.FreeformTags),
		DefinedTags:      onPremConnectorSharedTags(current.DefinedTags),
		SystemTags:       onPremConnectorSharedTags(current.SystemTags),
		CreatedVersion:   onPremConnectorStringValue(current.CreatedVersion),
	}
}

func (c onPremConnectorRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.OnPremConnector,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", onPremConnectorKind)
	}
	if err := validateOnPremConnectorCreateOrUpdateIdentity(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, markOnPremConnectorFailed(resource, err)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func validateOnPremConnectorCreateOrUpdateIdentity(resource *datasafev1beta1.OnPremConnector) error {
	if trackedOnPremConnectorID(resource) == "" || resource == nil {
		return nil
	}
	trackedCompartmentID := strings.TrimSpace(resource.Status.CompartmentId)
	desiredCompartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if trackedCompartmentID == "" || desiredCompartmentID == "" || trackedCompartmentID == desiredCompartmentID {
		return nil
	}
	return fmt.Errorf("%s formal semantics require replacement when compartmentId changes", onPremConnectorKind)
}

func markOnPremConnectorFailed(resource *datasafev1beta1.OnPremConnector, err error) error {
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

func (c onPremConnectorRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.OnPremConnector) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", onPremConnectorKind)
	}
	if resource != nil && trackedOnPremConnectorID(resource) == "" && onPremConnectorIdentityForDelete(resource).displayName == "" {
		markOnPremConnectorDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	handled, err := c.guardOnPremConnectorPreDeleteRead(ctx, resource)
	if err != nil {
		return false, err
	}
	if handled {
		return resource.Status.OsokStatus.DeletedAt != nil, nil
	}
	return c.delegate.Delete(ctx, resource)
}

func (c onPremConnectorRuntimeClient) guardOnPremConnectorPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.OnPremConnector,
) (bool, error) {
	if resource == nil {
		return false, nil
	}
	currentID := trackedOnPremConnectorID(resource)
	if currentID != "" {
		return c.guardOnPremConnectorPreDeleteGet(ctx, resource, currentID)
	}
	return false, c.rejectAuthShapedList(ctx, resource)
}

func (c onPremConnectorRuntimeClient) guardOnPremConnectorPreDeleteGet(
	ctx context.Context,
	resource *datasafev1beta1.OnPremConnector,
	currentID string,
) (bool, error) {
	if c.get == nil {
		return false, nil
	}
	response, err := c.get(ctx, datasafesdk.GetOnPremConnectorRequest{OnPremConnectorId: common.String(currentID)})
	if ambiguous := onPremConnectorAmbiguousDeleteError(resource, err, "pre-delete get"); ambiguous != nil {
		return false, ambiguous
	}
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markOnPremConnectorDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if err != nil {
		return false, markOnPremConnectorPreDeleteReadFailed(resource, err, "pre-delete get")
	}
	if !onPremConnectorPendingWriteState(onPremConnectorLifecycleState(response)) {
		return false, nil
	}
	if err := projectOnPremConnectorStatus(resource, response); err != nil {
		return false, err
	}
	markOnPremConnectorPendingDelete(resource, response)
	return true, nil
}

func (c onPremConnectorRuntimeClient) rejectAuthShapedList(
	ctx context.Context,
	resource *datasafev1beta1.OnPremConnector,
) error {
	if c.list == nil {
		return nil
	}
	identity := onPremConnectorIdentityForDelete(resource)
	if identity.compartmentID == "" || identity.displayName == "" {
		return nil
	}
	_, err := c.list(ctx, datasafesdk.ListOnPremConnectorsRequest{
		CompartmentId: common.String(identity.compartmentID),
		DisplayName:   onPremConnectorOptionalString(identity.displayName),
	})
	if ambiguous := onPremConnectorAmbiguousDeleteError(resource, err, "pre-delete list"); ambiguous != nil {
		return ambiguous
	}
	if err != nil {
		return markOnPremConnectorPreDeleteReadFailed(resource, err, "pre-delete list")
	}
	return nil
}

func markOnPremConnectorPreDeleteReadFailed(
	resource *datasafev1beta1.OnPremConnector,
	err error,
	operation string,
) error {
	if err == nil {
		return nil
	}
	return markOnPremConnectorFailed(
		resource,
		fmt.Errorf("%s %s failed; refusing to call delete: %w", onPremConnectorKind, operation, err),
	)
}

func onPremConnectorAmbiguousDeleteError(
	resource *datasafev1beta1.OnPremConnector,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return onPremConnectorAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", onPremConnectorKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func markOnPremConnectorDeleted(resource *datasafev1beta1.OnPremConnector, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func trackedOnPremConnectorID(resource *datasafev1beta1.OnPremConnector) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func onPremConnectorOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func onPremConnectorStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func onPremConnectorSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func onPremConnectorStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func onPremConnectorSharedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
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

func onPremConnectorCloneSharedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
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

func onPremConnectorDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
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

func firstOnPremConnectorString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var _ interface{ GetOpcRequestID() string } = onPremConnectorAmbiguousNotFoundError{}
