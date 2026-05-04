/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package artifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
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
	artifactKind                       = "Artifact"
	artifactAmbiguousNotFoundErrorCode = "ArtifactAmbiguousNotFound"
)

var artifactWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(marketplacepublishersdk.OperationStatusAccepted),
		string(marketplacepublishersdk.OperationStatusInProgress),
		string(marketplacepublishersdk.OperationStatusWaiting),
		string(marketplacepublishersdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(marketplacepublishersdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(marketplacepublishersdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(marketplacepublishersdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(marketplacepublishersdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(marketplacepublishersdk.OperationTypeCreateArtifact)},
	UpdateActionTokens:    []string{string(marketplacepublishersdk.OperationTypeUpdateArtifact)},
	DeleteActionTokens:    []string{string(marketplacepublishersdk.OperationTypeDeleteArtifact)},
}

type artifactOCIClient interface {
	CreateArtifact(context.Context, marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error)
	GetArtifact(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error)
	ListArtifacts(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error)
	UpdateArtifact(context.Context, marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error)
	DeleteArtifact(context.Context, marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error)
	GetWorkRequest(context.Context, marketplacepublishersdk.GetWorkRequestRequest) (marketplacepublishersdk.GetWorkRequestResponse, error)
}

type artifactIdentity struct {
	compartmentID string
	displayName   string
	artifactType  string
}

type artifactStatusProjection struct {
	JsonData                  string                                             `json:"jsonData,omitempty"`
	StatusNotes               string                                             `json:"statusNotes,omitempty"`
	FreeformTags              map[string]string                                  `json:"freeformTags,omitempty"`
	DefinedTags               map[string]shared.MapValue                         `json:"definedTags,omitempty"`
	SystemTags                map[string]shared.MapValue                         `json:"systemTags,omitempty"`
	Id                        string                                             `json:"id,omitempty"`
	DisplayName               string                                             `json:"displayName,omitempty"`
	SDKStatus                 string                                             `json:"sdkStatus,omitempty"`
	LifecycleState            string                                             `json:"lifecycleState,omitempty"`
	TimeCreated               string                                             `json:"timeCreated,omitempty"`
	CompartmentId             string                                             `json:"compartmentId,omitempty"`
	PublisherId               string                                             `json:"publisherId,omitempty"`
	TimeUpdated               string                                             `json:"timeUpdated,omitempty"`
	ArtifactType              string                                             `json:"artifactType,omitempty"`
	MachineImage              marketplacepublisherv1beta1.ArtifactMachineImage   `json:"machineImage,omitempty"`
	Stack                     marketplacepublisherv1beta1.ArtifactStack          `json:"stack,omitempty"`
	ContainerImage            marketplacepublisherv1beta1.ArtifactContainerImage `json:"containerImage,omitempty"`
	HelmChart                 marketplacepublisherv1beta1.ArtifactHelmChart      `json:"helmChart,omitempty"`
	ContainerImageArtifactIds []string                                           `json:"containerImageArtifactIds,omitempty"`
}

type artifactProjectedResponse struct {
	Artifact     artifactStatusProjection `presentIn:"body"`
	OpcRequestId *string                  `presentIn:"header" name:"opc-request-id"`
}

type artifactAmbiguousDeleteConfirmResponse struct {
	Artifact artifactStatusProjection `presentIn:"body"`
	err      error
}

type artifactProjectedCollection struct {
	Items []artifactStatusProjection `json:"items,omitempty"`
}

type artifactProjectedListResponse struct {
	ArtifactCollection artifactProjectedCollection `presentIn:"body"`
	OpcRequestId       *string                     `presentIn:"header" name:"opc-request-id"`
	OpcNextPage        *string                     `presentIn:"header" name:"opc-next-page"`
}

type artifactTypedReadback struct {
	artifactType              string
	nested                    any
	containerImageArtifactIDs []string
}

type artifactAmbiguousNotFoundError struct {
	HTTPStatusCode int
	ErrorCode      string
	OpcRequestID   string
	message        string
}

type artifactResourceContextKey struct{}

type artifactResourceContextClient struct {
	delegate ArtifactServiceClient
}

type artifactDeleteWithoutTrackedIDClient struct {
	ArtifactServiceClient
	listArtifacts  func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error)
	getArtifact    func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error)
	getWorkRequest func(context.Context, string) (any, error)
}

func (e artifactAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e artifactAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.OpcRequestID
}

func init() {
	registerArtifactRuntimeHooksMutator(func(manager *ArtifactServiceManager, hooks *ArtifactRuntimeHooks) {
		client, initErr := newArtifactSDKClient(manager)
		applyArtifactRuntimeHooks(hooks, client, initErr)
	})
}

func newArtifactSDKClient(manager *ArtifactServiceManager) (artifactOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", artifactKind)
	}
	client, err := marketplacepublishersdk.NewMarketplacePublisherClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyArtifactRuntimeHooks(
	hooks *ArtifactRuntimeHooks,
	client artifactOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedArtifactRuntimeSemantics()
	applyArtifactIdentityHooks(hooks)
	applyArtifactCreateHooks(hooks, client, initErr)
	applyArtifactReadHooks(hooks, client, initErr)
	applyArtifactUpdateHooks(hooks, client, initErr)
	applyArtifactDeleteHooks(hooks, client, initErr)
	applyArtifactStatusAndAsyncHooks(hooks, client, initErr)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapArtifactResourceContextClient)
}

func applyArtifactIdentityHooks(hooks *ArtifactRuntimeHooks) {
	hooks.Identity.Resolve = func(resource *marketplacepublisherv1beta1.Artifact) (any, error) {
		return resolveArtifactIdentity(resource)
	}
	hooks.Identity.RecordPath = recordArtifactPathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardArtifactExistingBeforeCreate
}

func applyArtifactCreateHooks(
	hooks *ArtifactRuntimeHooks,
	client artifactOCIClient,
	initErr error,
) {
	hooks.Create.Fields = artifactCreateRequestFields()
	hooks.Create.Call = func(ctx context.Context, request marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error) {
		if err := requireArtifactOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.CreateArtifactResponse{}, err
		}
		resource, err := artifactResourceFromContext(ctx)
		if err != nil {
			return marketplacepublishersdk.CreateArtifactResponse{}, err
		}
		request.CreateArtifactDetails, err = artifactCreateDetailsFromSpec(resource.Spec)
		if err != nil {
			return marketplacepublishersdk.CreateArtifactResponse{}, err
		}
		return client.CreateArtifact(ctx, request)
	}
}

func applyArtifactReadHooks(
	hooks *ArtifactRuntimeHooks,
	client artifactOCIClient,
	initErr error,
) {
	hooks.Get.Call = func(ctx context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
		if err := requireArtifactOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.GetArtifactResponse{}, err
		}
		response, err := client.GetArtifact(ctx, request)
		return response, conservativeArtifactNotFoundError(err, "read")
	}
	hooks.List.Fields = artifactListFields()
	hooks.List.Call = func(ctx context.Context, request marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
		return listArtifactsAllPages(ctx, client, initErr, request)
	}
	hooks.Read.Get = artifactProjectedGetReadOperation(hooks.Get)
	hooks.Read.List = artifactProjectedListReadOperation(hooks.List)
}

func applyArtifactUpdateHooks(
	hooks *ArtifactRuntimeHooks,
	client artifactOCIClient,
	initErr error,
) {
	hooks.Update.Fields = artifactUpdateRequestFields()
	hooks.Update.Call = func(ctx context.Context, request marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error) {
		if err := requireArtifactOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.UpdateArtifactResponse{}, err
		}
		resource, err := artifactResourceFromContext(ctx)
		if err != nil {
			return marketplacepublishersdk.UpdateArtifactResponse{}, err
		}
		body, err := artifactUpdateDetailsForRequest(resource, request)
		if err != nil {
			return marketplacepublishersdk.UpdateArtifactResponse{}, err
		}
		request.UpdateArtifactDetails = body
		return client.UpdateArtifact(ctx, request)
	}
}

func applyArtifactDeleteHooks(
	hooks *ArtifactRuntimeHooks,
	client artifactOCIClient,
	initErr error,
) {
	hooks.Delete.Call = func(ctx context.Context, request marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
		if err := requireArtifactOCIClient(client, initErr); err != nil {
			return marketplacepublishersdk.DeleteArtifactResponse{}, err
		}
		response, err := client.DeleteArtifact(ctx, request)
		return response, conservativeArtifactNotFoundError(err, "delete")
	}
	hooks.DeleteHooks.ConfirmRead = artifactDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleArtifactDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyArtifactDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapArtifactDeleteWithoutTrackedID(
		hooks.List.Call,
		hooks.Get.Call,
		func(ctx context.Context, workRequestID string) (any, error) {
			return getArtifactWorkRequest(ctx, client, initErr, workRequestID)
		},
	))
}

func applyArtifactStatusAndAsyncHooks(
	hooks *ArtifactRuntimeHooks,
	client artifactOCIClient,
	initErr error,
) {
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedArtifactIdentity
	hooks.StatusHooks.ProjectStatus = artifactStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateArtifactCreateOnlyDriftForResponse
	hooks.Async.Adapter = artifactWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getArtifactWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveArtifactWorkRequestAction
	hooks.Async.ResolvePhase = resolveArtifactWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverArtifactIDFromWorkRequest
	hooks.Async.Message = artifactWorkRequestMessage
}

func (c artifactResourceContextClient) CreateOrUpdate(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(context.WithValue(ctx, artifactResourceContextKey{}, resource), resource, req)
}

func (c artifactResourceContextClient) Delete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
) (bool, error) {
	deleteResource := artifactDeleteResourceForSpecValidation(resource)
	deleted, err := c.delegate.Delete(ctx, deleteResource)
	if deleteResource != resource && resource != nil && deleteResource != nil {
		resource.Status = deleteResource.Status
	}
	return deleted, err
}

func wrapArtifactResourceContextClient(delegate ArtifactServiceClient) ArtifactServiceClient {
	return artifactResourceContextClient{delegate: delegate}
}

func artifactDeleteResourceForSpecValidation(
	resource *marketplacepublisherv1beta1.Artifact,
) *marketplacepublisherv1beta1.Artifact {
	if resource == nil {
		return nil
	}
	if _, err := resolveArtifactIdentity(resource); err == nil {
		return resource
	}
	if !artifactCanDeleteWithRecordedIdentity(resource) {
		return resource
	}

	deleteResource := resource.DeepCopy()
	deleteResource.Spec = artifactSpecForDeleteIdentity(resource)
	return deleteResource
}

func artifactCanDeleteWithRecordedIdentity(resource *marketplacepublisherv1beta1.Artifact) bool {
	if artifactTrackedID(resource) != "" || artifactHasCurrentWorkRequest(resource) {
		return true
	}
	_, ok := artifactRecordedPathIdentity(resource)
	return ok
}

func artifactSpecForDeleteIdentity(
	resource *marketplacepublisherv1beta1.Artifact,
) marketplacepublisherv1beta1.ArtifactSpec {
	spec := marketplacepublisherv1beta1.ArtifactSpec{
		CompartmentId: strings.TrimSpace(resource.Spec.CompartmentId),
		DisplayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}
	if identity, ok := artifactRecordedPathIdentity(resource); ok {
		if spec.CompartmentId == "" {
			spec.CompartmentId = identity.compartmentID
		}
		if spec.DisplayName == "" {
			spec.DisplayName = identity.displayName
		}
		spec.ArtifactType = identity.artifactType
	}
	return spec
}

func artifactResourceFromContext(ctx context.Context) (*marketplacepublisherv1beta1.Artifact, error) {
	resource, ok := ctx.Value(artifactResourceContextKey{}).(*marketplacepublisherv1beta1.Artifact)
	if !ok || resource == nil {
		return nil, fmt.Errorf("%s resource is missing from runtime context", artifactKind)
	}
	return resource, nil
}

func requireArtifactOCIClient(client artifactOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize %s OCI client: %w", artifactKind, initErr)
	}
	if client == nil {
		return fmt.Errorf("%s OCI client is not configured", artifactKind)
	}
	return nil
}

func newArtifactServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client artifactOCIClient,
) ArtifactServiceClient {
	hooks := newArtifactRuntimeHooksWithOCIClient(client)
	applyArtifactRuntimeHooks(&hooks, client, nil)
	manager := &ArtifactServiceManager{Log: log}
	return wrapArtifactGeneratedClient(
		hooks,
		defaultArtifactServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.Artifact](
				buildArtifactGeneratedRuntimeConfig(manager, hooks),
			),
		},
	)
}

func newArtifactRuntimeHooksWithOCIClient(client artifactOCIClient) ArtifactRuntimeHooks {
	return ArtifactRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*marketplacepublisherv1beta1.Artifact]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*marketplacepublisherv1beta1.Artifact]{},
		StatusHooks:     generatedruntime.StatusHooks[*marketplacepublisherv1beta1.Artifact]{},
		ParityHooks:     generatedruntime.ParityHooks[*marketplacepublisherv1beta1.Artifact]{},
		Async:           generatedruntime.AsyncHooks[*marketplacepublisherv1beta1.Artifact]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*marketplacepublisherv1beta1.Artifact]{},
		Create: runtimeOperationHooks[marketplacepublishersdk.CreateArtifactRequest, marketplacepublishersdk.CreateArtifactResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateArtifactDetails", RequestName: "CreateArtifactDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request marketplacepublishersdk.CreateArtifactRequest) (marketplacepublishersdk.CreateArtifactResponse, error) {
				return client.CreateArtifact(ctx, request)
			},
		},
		Get: runtimeOperationHooks[marketplacepublishersdk.GetArtifactRequest, marketplacepublishersdk.GetArtifactResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ArtifactId", RequestName: "artifactId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error) {
				return client.GetArtifact(ctx, request)
			},
		},
		List: runtimeOperationHooks[marketplacepublishersdk.ListArtifactsRequest, marketplacepublishersdk.ListArtifactsResponse]{
			Fields: artifactListFields(),
			Call: func(ctx context.Context, request marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error) {
				return client.ListArtifacts(ctx, request)
			},
		},
		Update: runtimeOperationHooks[marketplacepublishersdk.UpdateArtifactRequest, marketplacepublishersdk.UpdateArtifactResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ArtifactId", RequestName: "artifactId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateArtifactDetails", RequestName: "UpdateArtifactDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request marketplacepublishersdk.UpdateArtifactRequest) (marketplacepublishersdk.UpdateArtifactResponse, error) {
				return client.UpdateArtifact(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[marketplacepublishersdk.DeleteArtifactRequest, marketplacepublishersdk.DeleteArtifactResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ArtifactId", RequestName: "artifactId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request marketplacepublishersdk.DeleteArtifactRequest) (marketplacepublishersdk.DeleteArtifactResponse, error) {
				return client.DeleteArtifact(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ArtifactServiceClient) ArtifactServiceClient{},
	}
}

func artifactProjectedGetReadOperation(
	get runtimeOperationHooks[marketplacepublishersdk.GetArtifactRequest, marketplacepublishersdk.GetArtifactResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &marketplacepublishersdk.GetArtifactRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), get.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := get.Call(ctx, *request.(*marketplacepublishersdk.GetArtifactRequest))
			if err != nil {
				return nil, err
			}
			return artifactProjectedResponseFromGet(response), nil
		},
	}
}

func artifactProjectedListReadOperation(
	list runtimeOperationHooks[marketplacepublishersdk.ListArtifactsRequest, marketplacepublishersdk.ListArtifactsResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &marketplacepublishersdk.ListArtifactsRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), list.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := list.Call(ctx, *request.(*marketplacepublishersdk.ListArtifactsRequest))
			if err != nil {
				return nil, err
			}
			return artifactProjectedListResponseFromSDK(response), nil
		},
	}
}

func reviewedArtifactRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "marketplacepublisher",
		FormalSlug:    "artifact",
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
			ProvisioningStates: []string{string(marketplacepublishersdk.ArtifactLifecycleStateCreating)},
			UpdatingStates:     []string{string(marketplacepublishersdk.ArtifactLifecycleStateUpdating)},
			ActiveStates:       []string{string(marketplacepublishersdk.ArtifactLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(marketplacepublishersdk.ArtifactLifecycleStateDeleting)},
			TerminalStates: []string{string(marketplacepublishersdk.ArtifactLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "artifactType", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"freeformTags",
				"definedTags",
				"helmChart",
				"containerImageArtifactIds",
				"stack",
				"containerImage",
				"machineImage",
			},
			ForceNew:      []string{"compartmentId", "artifactType"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetArtifact",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetArtifact",
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

func artifactListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", LookupPaths: []string{"status.lifecycleState", "lifecycleState"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func artifactCreateRequestFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpcRetryToken", RequestName: "opcRetryToken", Contribution: "header"},
	}
}

func artifactUpdateRequestFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ArtifactId", RequestName: "artifactId", Contribution: "path", PreferResourceID: true},
	}
}

func artifactCreateDetailsFromSpec(
	spec marketplacepublisherv1beta1.ArtifactSpec,
) (marketplacepublishersdk.CreateArtifactDetails, error) {
	payload, artifactType, err := artifactPayloadFromSpec(spec, false)
	if err != nil {
		return nil, err
	}
	if artifactType == "" {
		return nil, fmt.Errorf("%s spec.artifactType is required", artifactKind)
	}
	if err := validateArtifactCreatePayload(payload, artifactType); err != nil {
		return nil, err
	}
	return artifactCreateDetailsFromPayload(payload, artifactType)
}

func artifactUpdateDetailsForRequest(
	resource *marketplacepublisherv1beta1.Artifact,
	request marketplacepublishersdk.UpdateArtifactRequest,
) (marketplacepublishersdk.UpdateArtifactDetails, error) {
	artifactID := strings.TrimSpace(artifactString(request.ArtifactId))
	if artifactID == "" {
		return nil, fmt.Errorf("%s update requires artifactId", artifactKind)
	}
	return artifactUpdateDetailsFromDesired(resource)
}

func artifactUpdateDetailsFromDesired(
	resource *marketplacepublisherv1beta1.Artifact,
) (marketplacepublishersdk.UpdateArtifactDetails, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", artifactKind)
	}
	payload, artifactType, err := artifactPayloadFromSpec(resource.Spec, true)
	if err != nil {
		return nil, err
	}
	if artifactType == "" {
		artifactType = strings.TrimSpace(resource.Status.ArtifactType)
		payload["artifactType"] = artifactType
	}
	if artifactType == "" {
		return nil, fmt.Errorf("%s update cannot resolve artifactType", artifactKind)
	}
	if err := validateArtifactUpdatePayload(payload, artifactType); err != nil {
		return nil, err
	}
	delete(payload, "compartmentId")
	return artifactUpdateDetailsFromPayload(payload, artifactType)
}

func artifactPayloadFromSpec(
	spec marketplacepublisherv1beta1.ArtifactSpec,
	forUpdate bool,
) (map[string]any, string, error) {
	payload, err := artifactBasePayloadFromSpec(spec)
	if err != nil {
		return nil, "", err
	}
	artifactType, err := artifactPayloadTypeAndDetails(payload, spec)
	if err != nil {
		return nil, "", err
	}
	if !forUpdate && strings.TrimSpace(artifactStringPayload(payload, "compartmentId")) == "" {
		return nil, "", fmt.Errorf("%s spec is missing required field(s): compartmentId", artifactKind)
	}
	return payload, artifactType, nil
}

func artifactBasePayloadFromSpec(spec marketplacepublisherv1beta1.ArtifactSpec) (map[string]any, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		return nil, fmt.Errorf("%s spec.jsonData is not supported; use the typed Artifact spec fields", artifactKind)
	}
	payload := map[string]any{}
	if compartmentID := strings.TrimSpace(spec.CompartmentId); compartmentID != "" {
		payload["compartmentId"] = compartmentID
	}
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" {
		payload["displayName"] = displayName
	}
	if spec.FreeformTags != nil {
		payload["freeformTags"] = artifactCloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		payload["definedTags"] = artifactDefinedTagsFromSpec(spec.DefinedTags)
	}
	return payload, nil
}

func artifactPayloadTypeAndDetails(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ArtifactSpec,
) (string, error) {
	artifactType, err := resolveArtifactType(spec, payload)
	if err != nil {
		return "", err
	}
	if artifactType != "" {
		payload["artifactType"] = artifactType
		if err := applyArtifactTypedFields(payload, spec, artifactType); err != nil {
			return "", err
		}
	}
	return artifactType, nil
}

func resolveArtifactType(
	spec marketplacepublisherv1beta1.ArtifactSpec,
	payload map[string]any,
) (string, error) {
	rawType := strings.TrimSpace(spec.ArtifactType)
	if rawType == "" {
		rawType = artifactStringPayload(payload, "artifactType")
	}
	if rawType != "" {
		artifactType, err := normalizeArtifactType(rawType)
		if err != nil {
			return "", err
		}
		if err := validateArtifactTypeFieldConflicts(spec, artifactType); err != nil {
			return "", err
		}
		return artifactType, nil
	}
	inferredTypes := artifactSpecifiedTypes(spec)
	switch len(inferredTypes) {
	case 0:
		return "", nil
	case 1:
		return inferredTypes[0], nil
	default:
		return "", fmt.Errorf("%s spec cannot infer artifactType from multiple artifact detail blocks: %s", artifactKind, strings.Join(inferredTypes, ", "))
	}
}

func validateArtifactTypeFieldConflicts(
	spec marketplacepublisherv1beta1.ArtifactSpec,
	artifactType string,
) error {
	var conflicts []string
	for _, specifiedType := range artifactSpecifiedTypes(spec) {
		if specifiedType != artifactType {
			conflicts = append(conflicts, artifactTypeFieldName(specifiedType))
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec.artifactType %q conflicts with %s", artifactKind, artifactType, strings.Join(conflicts, ", "))
}

func artifactSpecifiedTypes(spec marketplacepublisherv1beta1.ArtifactSpec) []string {
	var types []string
	if artifactContainerImageSpecified(spec.ContainerImage) {
		types = append(types, string(marketplacepublishersdk.ArtifactTypeEnumContainerImage))
	}
	if artifactHelmChartSpecified(spec) {
		types = append(types, string(marketplacepublishersdk.ArtifactTypeEnumHelmChart))
	}
	if artifactStackSpecified(spec.Stack) {
		types = append(types, string(marketplacepublishersdk.ArtifactTypeEnumStack))
	}
	if artifactMachineImageSpecified(spec.MachineImage) {
		types = append(types, string(marketplacepublishersdk.ArtifactTypeEnumMachineImage))
	}
	return types
}

func artifactContainerImageSpecified(spec marketplacepublisherv1beta1.ArtifactContainerImage) bool {
	return strings.TrimSpace(spec.SourceRegistryId) != "" ||
		strings.TrimSpace(spec.SourceRegistryUrl) != ""
}

func artifactHelmChartSpecified(spec marketplacepublisherv1beta1.ArtifactSpec) bool {
	return strings.TrimSpace(spec.HelmChart.SourceRegistryId) != "" ||
		strings.TrimSpace(spec.HelmChart.SourceRegistryUrl) != "" ||
		spec.HelmChart.SupportedKubernetesVersions != nil ||
		spec.ContainerImageArtifactIds != nil
}

func artifactStackSpecified(spec marketplacepublisherv1beta1.ArtifactStack) bool {
	return strings.TrimSpace(spec.SourceStackId) != "" ||
		spec.ImageListingIds != nil
}

func artifactMachineImageSpecified(spec marketplacepublisherv1beta1.ArtifactMachineImage) bool {
	return strings.TrimSpace(spec.SourceImageId) != "" ||
		strings.TrimSpace(spec.Username) != "" ||
		spec.ImageShapeCompatibilityEntries != nil
}

func artifactTypeFieldName(artifactType string) string {
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		return "containerImage"
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		return "helmChart"
	case string(marketplacepublishersdk.ArtifactTypeEnumStack):
		return "stack"
	case string(marketplacepublishersdk.ArtifactTypeEnumMachineImage):
		return "machineImage"
	default:
		return artifactType
	}
}

func applyArtifactTypedFields(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ArtifactSpec,
	artifactType string,
) error {
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		return applyArtifactContainerImagePayload(payload, spec.ContainerImage)
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		return applyArtifactHelmChartPayload(payload, spec)
	case string(marketplacepublishersdk.ArtifactTypeEnumStack):
		return applyArtifactStackPayload(payload, spec.Stack)
	case string(marketplacepublishersdk.ArtifactTypeEnumMachineImage):
		return applyArtifactMachineImagePayload(payload, spec.MachineImage)
	default:
		return fmt.Errorf("%s spec.artifactType %q is not supported", artifactKind, artifactType)
	}
}

func applyArtifactContainerImagePayload(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ArtifactContainerImage,
) error {
	if !artifactContainerImageSpecified(spec) {
		return nil
	}
	containerImage, err := artifactObjectPayload(payload, "containerImage")
	if err != nil {
		return err
	}
	setArtifactStringPayload(containerImage, "sourceRegistryId", spec.SourceRegistryId)
	setArtifactStringPayload(containerImage, "sourceRegistryUrl", spec.SourceRegistryUrl)
	payload["containerImage"] = containerImage
	return nil
}

func applyArtifactHelmChartPayload(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ArtifactSpec,
) error {
	if !artifactHelmChartSpecified(spec) {
		return nil
	}
	helmChart, err := artifactObjectPayload(payload, "helmChart")
	if err != nil {
		return err
	}
	setArtifactStringPayload(helmChart, "sourceRegistryId", spec.HelmChart.SourceRegistryId)
	setArtifactStringPayload(helmChart, "sourceRegistryUrl", spec.HelmChart.SourceRegistryUrl)
	if spec.HelmChart.SupportedKubernetesVersions != nil {
		helmChart["supportedKubernetesVersions"] = append([]string(nil), spec.HelmChart.SupportedKubernetesVersions...)
	}
	if spec.ContainerImageArtifactIds != nil {
		payload["containerImageArtifactIds"] = append([]string(nil), spec.ContainerImageArtifactIds...)
	}
	payload["helmChart"] = helmChart
	return nil
}

func applyArtifactStackPayload(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ArtifactStack,
) error {
	if !artifactStackSpecified(spec) {
		return nil
	}
	stack, err := artifactObjectPayload(payload, "stack")
	if err != nil {
		return err
	}
	setArtifactStringPayload(stack, "sourceStackId", spec.SourceStackId)
	if spec.ImageListingIds != nil {
		stack["imageListingIds"] = append([]string(nil), spec.ImageListingIds...)
	}
	payload["stack"] = stack
	return nil
}

func applyArtifactMachineImagePayload(
	payload map[string]any,
	spec marketplacepublisherv1beta1.ArtifactMachineImage,
) error {
	if !artifactMachineImageSpecified(spec) {
		return nil
	}
	machineImage, err := artifactObjectPayload(payload, "machineImage")
	if err != nil {
		return err
	}
	setArtifactStringPayload(machineImage, "sourceImageId", spec.SourceImageId)
	if artifactMachineImageSpecified(spec) {
		machineImage["isSnapshotAllowed"] = spec.IsSnapshotAllowed
	}
	if spec.ImageShapeCompatibilityEntries != nil {
		machineImage["imageShapeCompatibilityEntries"] = artifactImageShapeCompatibilityPayloads(spec.ImageShapeCompatibilityEntries)
	}
	setArtifactStringPayload(machineImage, "username", spec.Username)
	payload["machineImage"] = machineImage
	return nil
}

func artifactImageShapeCompatibilityPayloads(
	entries []marketplacepublisherv1beta1.ArtifactMachineImageImageShapeCompatibilityEntry,
) []map[string]any {
	payloads := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		payload := map[string]any{}
		setArtifactStringPayload(payload, "shape", entry.Shape)
		if constraints := artifactMemoryConstraintsPayload(entry.MemoryConstraints); len(constraints) != 0 {
			payload["memoryConstraints"] = constraints
		}
		if constraints := artifactOCPUConstraintsPayload(entry.OcpuConstraints); len(constraints) != 0 {
			payload["ocpuConstraints"] = constraints
		}
		payloads = append(payloads, payload)
	}
	return payloads
}

func artifactMemoryConstraintsPayload(
	constraints marketplacepublisherv1beta1.ArtifactMachineImageImageShapeCompatibilityEntryMemoryConstraints,
) map[string]any {
	payload := map[string]any{}
	if constraints.MinInGBs != 0 {
		payload["minInGBs"] = constraints.MinInGBs
	}
	if constraints.MaxInGBs != 0 {
		payload["maxInGBs"] = constraints.MaxInGBs
	}
	return payload
}

func artifactOCPUConstraintsPayload(
	constraints marketplacepublisherv1beta1.ArtifactMachineImageImageShapeCompatibilityEntryOcpuConstraints,
) map[string]any {
	payload := map[string]any{}
	if constraints.Min != 0 {
		payload["min"] = constraints.Min
	}
	if constraints.Max != 0 {
		payload["max"] = constraints.Max
	}
	return payload
}

func validateArtifactCreatePayload(payload map[string]any, artifactType string) error {
	var missing []string
	if strings.TrimSpace(artifactStringPayload(payload, "compartmentId")) == "" {
		missing = append(missing, "compartmentId")
	}
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		missing = append(missing, artifactMissingObjectStrings(payload, "containerImage", "sourceRegistryId", "sourceRegistryUrl")...)
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		missing = append(missing, artifactMissingObjectStrings(payload, "helmChart", "sourceRegistryId", "sourceRegistryUrl")...)
	case string(marketplacepublishersdk.ArtifactTypeEnumStack):
		missing = append(missing, artifactMissingObjectStrings(payload, "stack", "sourceStackId")...)
	case string(marketplacepublishersdk.ArtifactTypeEnumMachineImage):
		missing = append(missing, artifactMissingObjectStrings(payload, "machineImage", "sourceImageId")...)
		if _, ok := artifactNestedValue(payload, "machineImage", "isSnapshotAllowed"); !ok {
			missing = append(missing, "machineImage.isSnapshotAllowed")
		}
		if entries, ok := artifactNestedValue(payload, "machineImage", "imageShapeCompatibilityEntries"); !ok || len(artifactSliceValue(entries)) == 0 {
			missing = append(missing, "machineImage.imageShapeCompatibilityEntries")
		}
	default:
		missing = append(missing, "artifactType")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", artifactKind, strings.Join(missing, ", "))
}

func artifactMissingObjectStrings(payload map[string]any, objectName string, fieldNames ...string) []string {
	var missing []string
	object, ok := artifactNestedObject(payload, objectName)
	if !ok {
		for _, fieldName := range fieldNames {
			missing = append(missing, objectName+"."+fieldName)
		}
		return missing
	}
	for _, fieldName := range fieldNames {
		if strings.TrimSpace(artifactStringPayload(object, fieldName)) == "" {
			missing = append(missing, objectName+"."+fieldName)
		}
	}
	return missing
}

func artifactCreateDetailsFromPayload(
	payload map[string]any,
	artifactType string,
) (marketplacepublishersdk.CreateArtifactDetails, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		var details marketplacepublishersdk.CreateContainerImageArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s container image create details: %w", artifactKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		var details marketplacepublishersdk.CreateKubernetesImageArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s helm chart create details: %w", artifactKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ArtifactTypeEnumStack):
		var details marketplacepublishersdk.CreateStackArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s stack create details: %w", artifactKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ArtifactTypeEnumMachineImage):
		var details marketplacepublishersdk.CreateMachineImageArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s machine image create details: %w", artifactKind, err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("%s spec.artifactType %q is not supported", artifactKind, artifactType)
	}
}

func validateArtifactUpdatePayload(payload map[string]any, artifactType string) error {
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		if _, ok := payload["containerImage"]; ok {
			missing := artifactMissingObjectStrings(payload, "containerImage", "sourceRegistryId", "sourceRegistryUrl")
			if len(missing) != 0 {
				return fmt.Errorf("%s spec is missing required field(s): %s", artifactKind, strings.Join(missing, ", "))
			}
		}
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		if _, ok := payload["helmChart"]; ok {
			missing := artifactMissingObjectStrings(payload, "helmChart", "sourceRegistryId", "sourceRegistryUrl")
			if len(missing) != 0 {
				return fmt.Errorf("%s spec is missing required field(s): %s", artifactKind, strings.Join(missing, ", "))
			}
		}
	default:
	}
	return nil
}

func artifactUpdateDetailsFromPayload(
	payload map[string]any,
	artifactType string,
) (marketplacepublishersdk.UpdateArtifactDetails, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		var details marketplacepublishersdk.UpdateContainerImageArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s container image update details: %w", artifactKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		var details marketplacepublishersdk.UpdateKubernetesImageArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s helm chart update details: %w", artifactKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ArtifactTypeEnumStack):
		var details marketplacepublishersdk.UpdateStackArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s stack update details: %w", artifactKind, err)
		}
		return details, nil
	case string(marketplacepublishersdk.ArtifactTypeEnumMachineImage):
		var details marketplacepublishersdk.UpdateMachineImageArtifactDetails
		if err := json.Unmarshal(data, &details); err != nil {
			return nil, fmt.Errorf("decode %s machine image update details: %w", artifactKind, err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("%s spec.artifactType %q is not supported", artifactKind, artifactType)
	}
}

func validateArtifactCreateOnlyDriftForResponse(
	resource *marketplacepublisherv1beta1.Artifact,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", artifactKind)
	}
	current, ok := artifactFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current %s response does not expose an artifact body", artifactKind)
	}
	return validateArtifactCreateOnlyDrift(resource.Spec, current)
}

func validateArtifactCreateOnlyDrift(
	spec marketplacepublisherv1beta1.ArtifactSpec,
	current marketplacepublishersdk.Artifact,
) error {
	desiredPayload, desiredType, err := artifactPayloadFromSpec(spec, true)
	if err != nil {
		return err
	}
	var drift []string
	if desiredCompartmentID := strings.TrimSpace(artifactStringPayload(desiredPayload, "compartmentId")); desiredCompartmentID != "" &&
		!artifactStringPtrEqual(current.GetCompartmentId(), desiredCompartmentID) {
		drift = append(drift, "compartmentId")
	}
	if currentType := artifactTypeFromModel(current); desiredType != "" && currentType != "" && desiredType != currentType {
		drift = append(drift, "artifactType")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("%s create-only field drift is not supported: %s", artifactKind, strings.Join(drift, ", "))
}

func artifactProjectedResponseFromGet(
	response marketplacepublishersdk.GetArtifactResponse,
) artifactProjectedResponse {
	projection, _ := artifactProjectionFromArtifact(response.Artifact)
	return artifactProjectedResponse{
		Artifact:     projection,
		OpcRequestId: response.OpcRequestId,
	}
}

func artifactProjectedListResponseFromSDK(
	response marketplacepublishersdk.ListArtifactsResponse,
) artifactProjectedListResponse {
	projected := artifactProjectedListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		projected.ArtifactCollection.Items = append(projected.ArtifactCollection.Items, artifactProjectionFromSummary(item))
	}
	return projected
}

func artifactProjectionFromArtifact(current marketplacepublishersdk.Artifact) (artifactStatusProjection, bool) {
	if current == nil {
		return artifactStatusProjection{}, false
	}
	projection := artifactStatusProjection{
		StatusNotes:    artifactString(current.GetStatusNotes()),
		FreeformTags:   artifactCloneStringMap(current.GetFreeformTags()),
		DefinedTags:    artifactStatusDefinedTags(current.GetDefinedTags()),
		SystemTags:     artifactStatusDefinedTags(current.GetSystemTags()),
		Id:             artifactString(current.GetId()),
		DisplayName:    artifactString(current.GetDisplayName()),
		SDKStatus:      string(current.GetStatus()),
		LifecycleState: string(current.GetLifecycleState()),
		TimeCreated:    artifactSDKTimeString(current.GetTimeCreated()),
		CompartmentId:  artifactString(current.GetCompartmentId()),
		PublisherId:    artifactString(current.GetPublisherId()),
		TimeUpdated:    artifactSDKTimeString(current.GetTimeUpdated()),
		ArtifactType:   artifactTypeFromModel(current),
	}
	_ = populateArtifactProjectionTypedStatus(&projection, current)
	return projection, true
}

func artifactProjectionFromSummary(current marketplacepublishersdk.ArtifactSummary) artifactStatusProjection {
	return artifactStatusProjection{
		FreeformTags:   artifactCloneStringMap(current.FreeformTags),
		DefinedTags:    artifactStatusDefinedTags(current.DefinedTags),
		SystemTags:     artifactStatusDefinedTags(current.SystemTags),
		Id:             artifactString(current.Id),
		DisplayName:    artifactString(current.DisplayName),
		SDKStatus:      string(current.Status),
		LifecycleState: string(current.LifecycleState),
		TimeCreated:    artifactSDKTimeString(current.TimeCreated),
		CompartmentId:  artifactString(current.CompartmentId),
		TimeUpdated:    artifactSDKTimeString(current.TimeUpdated),
		ArtifactType:   string(current.ArtifactType),
	}
}

func (p artifactStatusProjection) GetId() *string {
	return artifactOptionalString(p.Id)
}

func (p artifactStatusProjection) GetDisplayName() *string {
	return artifactOptionalString(p.DisplayName)
}

func (p artifactStatusProjection) GetStatus() marketplacepublishersdk.ArtifactStatusEnum {
	return marketplacepublishersdk.ArtifactStatusEnum(p.SDKStatus)
}

func (p artifactStatusProjection) GetLifecycleState() marketplacepublishersdk.ArtifactLifecycleStateEnum {
	return marketplacepublishersdk.ArtifactLifecycleStateEnum(p.LifecycleState)
}

func (p artifactStatusProjection) GetTimeCreated() *common.SDKTime {
	return nil
}

func (p artifactStatusProjection) GetCompartmentId() *string {
	return artifactOptionalString(p.CompartmentId)
}

func (p artifactStatusProjection) GetPublisherId() *string {
	return artifactOptionalString(p.PublisherId)
}

func (p artifactStatusProjection) GetTimeUpdated() *common.SDKTime {
	return nil
}

func (p artifactStatusProjection) GetStatusNotes() *string {
	return artifactOptionalString(p.StatusNotes)
}

func (p artifactStatusProjection) GetFreeformTags() map[string]string {
	return artifactCloneStringMap(p.FreeformTags)
}

func (p artifactStatusProjection) GetDefinedTags() map[string]map[string]interface{} {
	return artifactDefinedTagsFromSpec(p.DefinedTags)
}

func (p artifactStatusProjection) GetSystemTags() map[string]map[string]interface{} {
	return artifactDefinedTagsFromSpec(p.SystemTags)
}

func populateArtifactProjectionTypedStatus(
	projection *artifactStatusProjection,
	current marketplacepublishersdk.Artifact,
) error {
	readback, ok := artifactTypedReadbackFromModel(current)
	if !ok {
		return nil
	}
	projection.ContainerImageArtifactIds = append([]string(nil), readback.containerImageArtifactIDs...)
	return artifactProjectNestedStatus(readback.nested, artifactProjectionNestedStatusTarget(projection, readback.artifactType))
}

func artifactStatusFromResponse(resource *marketplacepublisherv1beta1.Artifact, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", artifactKind)
	}
	switch current := response.(type) {
	case artifactProjectedResponse:
		artifactStatusFromProjection(resource, current.Artifact)
		return nil
	case *artifactProjectedResponse:
		if current != nil {
			artifactStatusFromProjection(resource, current.Artifact)
		}
		return nil
	case artifactStatusProjection:
		artifactStatusFromProjection(resource, current)
		return nil
	case *artifactStatusProjection:
		if current != nil {
			artifactStatusFromProjection(resource, *current)
		}
		return nil
	}
	if current, ok := artifactFromResponse(response); ok {
		return artifactStatusFromArtifact(resource, current)
	}
	if summary, ok := artifactSummaryFromResponse(response); ok {
		artifactStatusFromSummary(resource, summary)
	}
	return nil
}

func artifactStatusFromProjection(
	resource *marketplacepublisherv1beta1.Artifact,
	current artifactStatusProjection,
) {
	status := resource.Status.OsokStatus
	resource.Status = marketplacepublisherv1beta1.ArtifactStatus{
		OsokStatus:                status,
		JsonData:                  current.JsonData,
		StatusNotes:               current.StatusNotes,
		FreeformTags:              artifactCloneStringMap(current.FreeformTags),
		DefinedTags:               cloneArtifactStatusDefinedTags(current.DefinedTags),
		SystemTags:                cloneArtifactStatusDefinedTags(current.SystemTags),
		Id:                        current.Id,
		DisplayName:               current.DisplayName,
		Status:                    current.SDKStatus,
		LifecycleState:            current.LifecycleState,
		TimeCreated:               current.TimeCreated,
		CompartmentId:             current.CompartmentId,
		PublisherId:               current.PublisherId,
		TimeUpdated:               current.TimeUpdated,
		ArtifactType:              current.ArtifactType,
		MachineImage:              current.MachineImage,
		Stack:                     current.Stack,
		ContainerImage:            current.ContainerImage,
		HelmChart:                 current.HelmChart,
		ContainerImageArtifactIds: append([]string(nil), current.ContainerImageArtifactIds...),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func artifactStatusFromArtifact(
	resource *marketplacepublisherv1beta1.Artifact,
	current marketplacepublishersdk.Artifact,
) error {
	status := resource.Status.OsokStatus
	next := marketplacepublisherv1beta1.ArtifactStatus{
		OsokStatus:     status,
		StatusNotes:    artifactString(current.GetStatusNotes()),
		FreeformTags:   artifactCloneStringMap(current.GetFreeformTags()),
		DefinedTags:    artifactStatusDefinedTags(current.GetDefinedTags()),
		SystemTags:     artifactStatusDefinedTags(current.GetSystemTags()),
		Id:             artifactString(current.GetId()),
		DisplayName:    artifactString(current.GetDisplayName()),
		Status:         string(current.GetStatus()),
		LifecycleState: string(current.GetLifecycleState()),
		TimeCreated:    artifactSDKTimeString(current.GetTimeCreated()),
		CompartmentId:  artifactString(current.GetCompartmentId()),
		PublisherId:    artifactString(current.GetPublisherId()),
		TimeUpdated:    artifactSDKTimeString(current.GetTimeUpdated()),
		ArtifactType:   artifactTypeFromModel(current),
	}
	if err := populateArtifactTypedStatus(&next, current); err != nil {
		return err
	}
	resource.Status = next
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func artifactSummaryFromResponse(response any) (marketplacepublishersdk.ArtifactSummary, bool) {
	switch current := response.(type) {
	case marketplacepublishersdk.ArtifactSummary:
		return current, true
	case *marketplacepublishersdk.ArtifactSummary:
		if current == nil {
			return marketplacepublishersdk.ArtifactSummary{}, false
		}
		return *current, true
	default:
		return marketplacepublishersdk.ArtifactSummary{}, false
	}
}

func artifactStatusFromSummary(
	resource *marketplacepublisherv1beta1.Artifact,
	current marketplacepublishersdk.ArtifactSummary,
) {
	status := resource.Status.OsokStatus
	resource.Status = marketplacepublisherv1beta1.ArtifactStatus{
		OsokStatus:     status,
		FreeformTags:   artifactCloneStringMap(current.FreeformTags),
		DefinedTags:    artifactStatusDefinedTags(current.DefinedTags),
		SystemTags:     artifactStatusDefinedTags(current.SystemTags),
		Id:             artifactString(current.Id),
		DisplayName:    artifactString(current.DisplayName),
		Status:         string(current.Status),
		LifecycleState: string(current.LifecycleState),
		TimeCreated:    artifactSDKTimeString(current.TimeCreated),
		CompartmentId:  artifactString(current.CompartmentId),
		TimeUpdated:    artifactSDKTimeString(current.TimeUpdated),
		ArtifactType:   string(current.ArtifactType),
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func populateArtifactTypedStatus(
	status *marketplacepublisherv1beta1.ArtifactStatus,
	current marketplacepublishersdk.Artifact,
) error {
	readback, ok := artifactTypedReadbackFromModel(current)
	if !ok {
		return nil
	}
	status.ContainerImageArtifactIds = append([]string(nil), readback.containerImageArtifactIDs...)
	return artifactProjectNestedStatus(readback.nested, artifactStatusNestedStatusTarget(status, readback.artifactType))
}

func artifactTypedReadbackFromModel(current marketplacepublishersdk.Artifact) (artifactTypedReadback, bool) {
	switch typed := current.(type) {
	case marketplacepublishersdk.ContainerImageArtifact:
		return artifactContainerImageReadback(typed.ContainerImage)
	case *marketplacepublishersdk.ContainerImageArtifact:
		return artifactContainerImageReadbackFromPointer(typed)
	case marketplacepublishersdk.KubernetesImageArtifact:
		return artifactHelmChartReadback(typed.HelmChart, typed.ContainerImageArtifactIds)
	case *marketplacepublishersdk.KubernetesImageArtifact:
		return artifactHelmChartReadbackFromPointer(typed)
	case marketplacepublishersdk.StackArtifact:
		return artifactStackReadback(typed.Stack)
	case *marketplacepublishersdk.StackArtifact:
		return artifactStackReadbackFromPointer(typed)
	case marketplacepublishersdk.MachineImageArtifact:
		return artifactMachineImageReadback(typed.MachineImage)
	case *marketplacepublishersdk.MachineImageArtifact:
		return artifactMachineImageReadbackFromPointer(typed)
	default:
		return artifactTypedReadback{}, false
	}
}

func artifactContainerImageReadback(nested any) (artifactTypedReadback, bool) {
	return artifactTypedReadback{
		artifactType: string(marketplacepublishersdk.ArtifactTypeEnumContainerImage),
		nested:       nested,
	}, true
}

func artifactContainerImageReadbackFromPointer(
	current *marketplacepublishersdk.ContainerImageArtifact,
) (artifactTypedReadback, bool) {
	if current == nil {
		return artifactTypedReadback{}, false
	}
	return artifactContainerImageReadback(current.ContainerImage)
}

func artifactHelmChartReadback(nested any, artifactIDs []string) (artifactTypedReadback, bool) {
	return artifactTypedReadback{
		artifactType:              string(marketplacepublishersdk.ArtifactTypeEnumHelmChart),
		nested:                    nested,
		containerImageArtifactIDs: append([]string(nil), artifactIDs...),
	}, true
}

func artifactHelmChartReadbackFromPointer(
	current *marketplacepublishersdk.KubernetesImageArtifact,
) (artifactTypedReadback, bool) {
	if current == nil {
		return artifactTypedReadback{}, false
	}
	return artifactHelmChartReadback(current.HelmChart, current.ContainerImageArtifactIds)
}

func artifactStackReadback(nested any) (artifactTypedReadback, bool) {
	return artifactTypedReadback{
		artifactType: string(marketplacepublishersdk.ArtifactTypeEnumStack),
		nested:       nested,
	}, true
}

func artifactStackReadbackFromPointer(
	current *marketplacepublishersdk.StackArtifact,
) (artifactTypedReadback, bool) {
	if current == nil {
		return artifactTypedReadback{}, false
	}
	return artifactStackReadback(current.Stack)
}

func artifactMachineImageReadback(nested any) (artifactTypedReadback, bool) {
	return artifactTypedReadback{
		artifactType: string(marketplacepublishersdk.ArtifactTypeEnumMachineImage),
		nested:       nested,
	}, true
}

func artifactMachineImageReadbackFromPointer(
	current *marketplacepublishersdk.MachineImageArtifact,
) (artifactTypedReadback, bool) {
	if current == nil {
		return artifactTypedReadback{}, false
	}
	return artifactMachineImageReadback(current.MachineImage)
}

func artifactProjectionNestedStatusTarget(
	projection *artifactStatusProjection,
	artifactType string,
) any {
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		return &projection.ContainerImage
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		return &projection.HelmChart
	case string(marketplacepublishersdk.ArtifactTypeEnumStack):
		return &projection.Stack
	case string(marketplacepublishersdk.ArtifactTypeEnumMachineImage):
		return &projection.MachineImage
	default:
		return nil
	}
}

func artifactStatusNestedStatusTarget(
	status *marketplacepublisherv1beta1.ArtifactStatus,
	artifactType string,
) any {
	switch artifactType {
	case string(marketplacepublishersdk.ArtifactTypeEnumContainerImage):
		return &status.ContainerImage
	case string(marketplacepublishersdk.ArtifactTypeEnumHelmChart):
		return &status.HelmChart
	case string(marketplacepublishersdk.ArtifactTypeEnumStack):
		return &status.Stack
	case string(marketplacepublishersdk.ArtifactTypeEnumMachineImage):
		return &status.MachineImage
	default:
		return nil
	}
}

func artifactProjectNestedStatus(source any, target any) error {
	if source == nil {
		return nil
	}
	if target == nil {
		return nil
	}
	data, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("marshal %s nested readback: %w", artifactKind, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("project %s nested readback into status: %w", artifactKind, err)
	}
	return nil
}

func resolveArtifactIdentity(resource *marketplacepublisherv1beta1.Artifact) (artifactIdentity, error) {
	if resource == nil {
		return artifactIdentity{}, fmt.Errorf("%s resource is nil", artifactKind)
	}
	payload, artifactType, err := artifactPayloadFromSpec(resource.Spec, true)
	if err != nil {
		return artifactIdentity{}, err
	}
	return artifactIdentity{
		compartmentID: strings.TrimSpace(artifactStringPayload(payload, "compartmentId")),
		displayName:   strings.TrimSpace(artifactStringPayload(payload, "displayName")),
		artifactType:  artifactType,
	}, nil
}

func recordArtifactPathIdentity(resource *marketplacepublisherv1beta1.Artifact, identity any) {
	if resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	typed, ok := identity.(artifactIdentity)
	if !ok {
		return
	}
	resource.Status.CompartmentId = typed.compartmentID
	resource.Status.DisplayName = typed.displayName
	resource.Status.ArtifactType = typed.artifactType
}

func guardArtifactExistingBeforeCreate(
	_ context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveArtifactIdentity(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	if identity.compartmentID == "" || identity.displayName == "" || identity.artifactType == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func wrapArtifactDeleteWithoutTrackedID(
	listArtifacts func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error),
	getArtifact func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error),
	getWorkRequest func(context.Context, string) (any, error),
) func(ArtifactServiceClient) ArtifactServiceClient {
	return func(delegate ArtifactServiceClient) ArtifactServiceClient {
		return artifactDeleteWithoutTrackedIDClient{
			ArtifactServiceClient: delegate,
			listArtifacts:         listArtifacts,
			getArtifact:           getArtifact,
			getWorkRequest:        getWorkRequest,
		}
	}
}

func (c artifactDeleteWithoutTrackedIDClient) Delete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
) (bool, error) {
	if workRequestID, phase := currentArtifactWriteWorkRequest(resource); workRequestID != "" {
		return c.resumeWriteWorkRequestBeforeDelete(ctx, resource, workRequestID, phase)
	}

	if artifactTrackedID(resource) != "" || artifactHasCurrentWorkRequest(resource) {
		return c.ArtifactServiceClient.Delete(ctx, resource)
	}

	response, found, err := artifactDeleteResolutionByList(ctx, resource, c.listArtifacts)
	if err != nil {
		return false, err
	}
	if !found {
		markArtifactDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if err := artifactStatusFromResponse(resource, response); err != nil {
		return false, err
	}
	return c.ArtifactServiceClient.Delete(ctx, resource)
}

func (c artifactDeleteWithoutTrackedIDClient) resumeWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	if c.getWorkRequest == nil {
		return false, fmt.Errorf("%s work request polling is not configured", artifactKind)
	}

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	current, err := buildArtifactWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return false, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("%s %s work request %s is still in progress; waiting before delete", artifactKind, phase, workRequestID)
		markArtifactWriteWorkRequestForDelete(resource, current, shared.OSOKAsyncClassPending, message)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.deleteAfterSucceededWriteWorkRequest(ctx, resource, workRequest, current, workRequestID)
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("%s %s work request %s finished with status %s before delete", artifactKind, phase, workRequestID, current.RawStatus)
		return false, failArtifactWriteWorkRequestForDelete(resource, current, err)
	default:
		err := fmt.Errorf("%s %s work request %s projected unsupported async class %s before delete", artifactKind, phase, workRequestID, current.NormalizedClass)
		return false, failArtifactWriteWorkRequestForDelete(resource, current, err)
	}
}

func (c artifactDeleteWithoutTrackedIDClient) deleteAfterSucceededWriteWorkRequest(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID := artifactTrackedID(resource)
	if resourceID == "" {
		recoveredID, err := recoverArtifactIDFromWorkRequest(resource, workRequest, current.Phase)
		if err != nil {
			return false, failArtifactWriteWorkRequestForDelete(resource, current, err)
		}
		resourceID = strings.TrimSpace(recoveredID)
	}
	if resourceID == "" {
		err := fmt.Errorf("%s %s work request %s did not expose an %s identifier", artifactKind, current.Phase, workRequestID, artifactKind)
		return false, failArtifactWriteWorkRequestForDelete(resource, current, err)
	}
	if c.getArtifact == nil {
		return false, failArtifactWriteWorkRequestForDelete(resource, current, fmt.Errorf("%s readback is not configured", artifactKind))
	}

	response, err := c.getArtifact(ctx, marketplacepublishersdk.GetArtifactRequest{ArtifactId: common.String(resourceID)})
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		if classification.IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, fmt.Errorf("%s write work request readback returned ambiguous 404 NotAuthorizedOrNotFound before delete: %w", artifactKind, err)
		}
		if classification.IsUnambiguousNotFound() {
			markArtifactWriteReadbackPending(resource, current, workRequestID, resourceID)
			return false, nil
		}
		return false, failArtifactWriteWorkRequestForDelete(resource, current, err)
	}
	if err := artifactStatusFromResponse(resource, response); err != nil {
		return false, err
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return c.ArtifactServiceClient.Delete(ctx, resource)
}

func artifactTrackedID(resource *marketplacepublisherv1beta1.Artifact) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentArtifactWriteWorkRequest(resource *marketplacepublisherv1beta1.Artifact) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		if workRequestID := strings.TrimSpace(current.WorkRequestID); workRequestID != "" {
			return workRequestID, current.Phase
		}
	}
	return "", ""
}

func artifactHasCurrentWorkRequest(resource *marketplacepublisherv1beta1.Artifact) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Source == shared.OSOKAsyncSourceWorkRequest && strings.TrimSpace(current.WorkRequestID) != ""
}

func artifactDeleteResolutionByList(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
	list func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error),
) (artifactStatusProjection, bool, error) {
	if list == nil {
		return artifactStatusProjection{}, false, fmt.Errorf("confirm %s delete: list hook is not configured", artifactKind)
	}
	request, identity, ok, err := artifactDeleteListRequest(resource)
	if err != nil {
		return artifactStatusProjection{}, false, err
	}
	if !ok {
		return artifactStatusProjection{}, false, fmt.Errorf("confirm %s delete: artifact identity is not recorded", artifactKind)
	}
	response, err := list(ctx, request)
	if err != nil {
		return artifactStatusProjection{}, false, recordArtifactDeleteConfirmError(resource, err)
	}

	var matches []artifactStatusProjection
	for _, item := range response.Items {
		if artifactSummaryMatchesIdentity(identity, item) {
			matches = append(matches, artifactProjectionFromSummary(item))
		}
	}
	switch len(matches) {
	case 0:
		return artifactStatusProjection{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return artifactStatusProjection{}, false, fmt.Errorf(
			"confirm %s delete: found %d artifacts matching compartmentId %q, displayName %q, and artifactType %q",
			artifactKind,
			len(matches),
			identity.compartmentID,
			identity.displayName,
			identity.artifactType,
		)
	}
}

func markArtifactDeleted(resource *marketplacepublisherv1beta1.Artifact, message string) {
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

func markArtifactWriteReadbackPending(
	resource *marketplacepublisherv1beta1.Artifact,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
	resourceID string,
) {
	if resourceID := strings.TrimSpace(resourceID); resourceID != "" {
		resource.Status.Id = resourceID
		resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	}
	message := fmt.Sprintf(
		"%s %s work request %s succeeded; waiting for %s %s to become readable before delete",
		artifactKind,
		current.Phase,
		strings.TrimSpace(workRequestID),
		artifactKind,
		strings.TrimSpace(resourceID),
	)
	markArtifactWriteWorkRequestForDelete(resource, current, shared.OSOKAsyncClassPending, message)
}

func markArtifactWriteWorkRequestForDelete(
	resource *marketplacepublisherv1beta1.Artifact,
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
	next.UpdatedAt = nil
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, loggerutil.OSOKLogger{})
}

func failArtifactWriteWorkRequestForDelete(
	resource *marketplacepublisherv1beta1.Artifact,
	current *shared.OSOKAsyncOperation,
	err error,
) error {
	if resource == nil || err == nil {
		return err
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if current == nil {
		return err
	}
	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}
	markArtifactWriteWorkRequestForDelete(resource, current, class, err.Error())
	return err
}

func listArtifactsAllPages(
	ctx context.Context,
	client artifactOCIClient,
	initErr error,
	request marketplacepublishersdk.ListArtifactsRequest,
) (marketplacepublishersdk.ListArtifactsResponse, error) {
	if initErr != nil {
		return marketplacepublishersdk.ListArtifactsResponse{}, fmt.Errorf("initialize %s OCI client: %w", artifactKind, initErr)
	}
	if client == nil {
		return marketplacepublishersdk.ListArtifactsResponse{}, fmt.Errorf("%s OCI client is not configured", artifactKind)
	}

	var combined marketplacepublishersdk.ListArtifactsResponse
	for {
		response, err := client.ListArtifacts(ctx, request)
		if err != nil {
			return marketplacepublishersdk.ListArtifactsResponse{}, conservativeArtifactNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == marketplacepublishersdk.ArtifactLifecycleStateDeleted {
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

func artifactDeleteConfirmRead(
	get func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error),
	list func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error),
) func(context.Context, *marketplacepublisherv1beta1.Artifact, string) (any, error) {
	return func(ctx context.Context, resource *marketplacepublisherv1beta1.Artifact, currentID string) (any, error) {
		if artifactID := strings.TrimSpace(currentID); artifactID != "" {
			return confirmArtifactDeleteByID(ctx, resource, get, artifactID)
		}
		return confirmArtifactDeleteByList(ctx, resource, list)
	}
}

func confirmArtifactDeleteByID(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
	get func(context.Context, marketplacepublishersdk.GetArtifactRequest) (marketplacepublishersdk.GetArtifactResponse, error),
	artifactID string,
) (any, error) {
	if get == nil {
		return nil, fmt.Errorf("confirm %s delete: get hook is not configured", artifactKind)
	}
	response, err := get(ctx, marketplacepublishersdk.GetArtifactRequest{ArtifactId: common.String(artifactID)})
	if err != nil {
		if !isArtifactAmbiguousDeleteConfirmError(err) {
			return nil, recordArtifactDeleteConfirmError(resource, err)
		}
		handledErr := handleArtifactDeleteError(resource, err)
		if handledErr == nil {
			handledErr = err
		}
		return artifactAmbiguousDeleteConfirmResponse{
			Artifact: artifactStatusProjection{
				Id:             artifactID,
				LifecycleState: string(marketplacepublishersdk.ArtifactLifecycleStateActive),
			},
			err: handledErr,
		}, nil
	}
	return artifactProjectedResponseFromGet(response), nil
}

func confirmArtifactDeleteByList(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
	list func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error),
) (any, error) {
	if list == nil {
		return nil, fmt.Errorf("confirm %s delete: list hook is not configured", artifactKind)
	}
	request, identity, ok, err := artifactDeleteListRequest(resource)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("confirm %s delete: artifact identity is not recorded", artifactKind)
	}
	return confirmArtifactDeleteListResponse(ctx, resource, list, request, identity)
}

func confirmArtifactDeleteListResponse(
	ctx context.Context,
	resource *marketplacepublisherv1beta1.Artifact,
	list func(context.Context, marketplacepublishersdk.ListArtifactsRequest) (marketplacepublishersdk.ListArtifactsResponse, error),
	request marketplacepublishersdk.ListArtifactsRequest,
	identity artifactIdentity,
) (any, error) {
	response, err := list(ctx, request)
	if err != nil {
		return nil, recordArtifactDeleteConfirmError(resource, err)
	}
	for _, item := range response.Items {
		if artifactSummaryMatchesIdentity(identity, item) {
			return artifactProjectionFromSummary(item), nil
		}
	}
	return nil, artifactDeleteConfirmNotFoundError()
}

func artifactDeleteConfirmNotFoundError() error {
	return errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "Artifact delete confirmation did not find a matching OCI artifact",
	}
}

func artifactDeleteListRequest(resource *marketplacepublisherv1beta1.Artifact) (
	marketplacepublishersdk.ListArtifactsRequest,
	artifactIdentity,
	bool,
	error,
) {
	identity, ok, err := artifactDeleteIdentity(resource)
	if err != nil {
		return marketplacepublishersdk.ListArtifactsRequest{}, artifactIdentity{}, false, err
	}
	request := marketplacepublishersdk.ListArtifactsRequest{}
	if identity.compartmentID != "" {
		request.CompartmentId = common.String(identity.compartmentID)
	}
	if identity.displayName != "" {
		request.DisplayName = common.String(identity.displayName)
	}
	return request, identity, ok && request.CompartmentId != nil && request.DisplayName != nil && identity.artifactType != "", nil
}

func artifactDeleteIdentity(resource *marketplacepublisherv1beta1.Artifact) (artifactIdentity, bool, error) {
	identity, err := resolveArtifactIdentity(resource)
	if err != nil {
		recorded, ok := artifactRecordedPathIdentity(resource)
		if ok {
			return recorded, true, nil
		}
		return artifactIdentity{}, false, err
	}

	recorded, recordedOK := artifactRecordedPathIdentity(resource)
	if recordedOK {
		if identity.compartmentID == "" {
			identity.compartmentID = recorded.compartmentID
		}
		if identity.displayName == "" {
			identity.displayName = recorded.displayName
		}
		if identity.artifactType == "" {
			identity.artifactType = recorded.artifactType
		}
	}
	return identity, artifactIdentityComplete(identity), nil
}

func artifactRecordedPathIdentity(resource *marketplacepublisherv1beta1.Artifact) (artifactIdentity, bool) {
	if resource == nil {
		return artifactIdentity{}, false
	}
	identity := artifactIdentity{
		compartmentID: strings.TrimSpace(resource.Status.CompartmentId),
		displayName:   strings.TrimSpace(resource.Status.DisplayName),
		artifactType:  strings.TrimSpace(resource.Status.ArtifactType),
	}
	if identity.artifactType != "" {
		artifactType, err := normalizeArtifactType(identity.artifactType)
		if err != nil {
			return artifactIdentity{}, false
		}
		identity.artifactType = artifactType
	}
	return identity, artifactIdentityComplete(identity)
}

func artifactIdentityComplete(identity artifactIdentity) bool {
	return identity.compartmentID != "" && identity.displayName != "" && identity.artifactType != ""
}

func artifactSummaryMatchesIdentity(identity artifactIdentity, summary marketplacepublishersdk.ArtifactSummary) bool {
	if summary.LifecycleState == marketplacepublishersdk.ArtifactLifecycleStateDeleted {
		return false
	}
	if identity.compartmentID != "" && !artifactStringPtrEqual(summary.CompartmentId, identity.compartmentID) {
		return false
	}
	if identity.displayName == "" || !artifactStringPtrEqual(summary.DisplayName, identity.displayName) {
		return false
	}
	if identity.artifactType != "" && string(summary.ArtifactType) != identity.artifactType {
		return false
	}
	return strings.TrimSpace(artifactString(summary.Id)) != ""
}

func recordArtifactDeleteConfirmError(resource *marketplacepublisherv1beta1.Artifact, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func handleArtifactDeleteError(resource *marketplacepublisherv1beta1.Artifact, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	classification := errorutil.ClassifyDeleteError(err)
	if classification.HTTPStatusCode == 404 && classification.ErrorCode == artifactAmbiguousNotFoundErrorCode {
		return err
	}
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return artifactAmbiguousNotFoundError{
		HTTPStatusCode: classification.HTTPStatusCode,
		ErrorCode:      artifactAmbiguousNotFoundErrorCode,
		OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
		message: fmt.Sprintf(
			"%s delete returned ambiguous not-found response (HTTP %s, code %s); retaining finalizer until OCI deletion is confirmed",
			artifactKind,
			classification.HTTPStatusCodeString(),
			classification.ErrorCodeString(),
		),
	}
}

func applyArtifactDeleteOutcome(
	_ *marketplacepublisherv1beta1.Artifact,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if err, ok := artifactAmbiguousDeleteConfirmError(response); ok {
		if err != nil {
			return generatedruntime.DeleteOutcome{Handled: true}, err
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func artifactAmbiguousDeleteConfirmError(response any) (error, bool) {
	switch typed := response.(type) {
	case artifactAmbiguousDeleteConfirmResponse:
		return typed.err, true
	case *artifactAmbiguousDeleteConfirmResponse:
		if typed == nil {
			return nil, false
		}
		return typed.err, true
	default:
		return nil, false
	}
}

func isArtifactAmbiguousDeleteConfirmError(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsAuthShapedNotFound() {
		return true
	}
	return classification.HTTPStatusCode == 404 && classification.ErrorCode == artifactAmbiguousNotFoundErrorCode
}

func conservativeArtifactNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf(
		"%s %s returned ambiguous not-found response: %s",
		artifactKind,
		strings.TrimSpace(operation),
		err.Error(),
	)
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return artifactAmbiguousNotFoundError{
			HTTPStatusCode: serviceErr.GetHTTPStatusCode(),
			ErrorCode:      artifactAmbiguousNotFoundErrorCode,
			OpcRequestID:   serviceErr.GetOpcRequestID(),
			message:        message,
		}
	}
	return artifactAmbiguousNotFoundError{ErrorCode: artifactAmbiguousNotFoundErrorCode, message: message}
}

func clearTrackedArtifactIdentity(resource *marketplacepublisherv1beta1.Artifact) {
	if resource == nil {
		return
	}
	resource.Status = marketplacepublisherv1beta1.ArtifactStatus{}
}

func getArtifactWorkRequest(
	ctx context.Context,
	client artifactOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", artifactKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", artifactKind)
	}
	response, err := client.GetWorkRequest(ctx, marketplacepublishersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveArtifactWorkRequestAction(workRequest any) (string, error) {
	current, err := artifactWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveArtifactWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := artifactWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case marketplacepublishersdk.OperationTypeCreateArtifact:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case marketplacepublishersdk.OperationTypeUpdateArtifact:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case marketplacepublishersdk.OperationTypeDeleteArtifact:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func buildArtifactWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := artifactWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	action, err := resolveArtifactWorkRequestAction(current)
	if err != nil {
		return nil, err
	}
	derivedPhase, ok, err := resolveArtifactWorkRequestPhase(current)
	if err != nil {
		return nil, err
	}
	if ok {
		if phase != "" && phase != derivedPhase {
			return nil, fmt.Errorf("%s work request %s exposes phase %q while delete expected %q", artifactKind, artifactString(current.Id), derivedPhase, phase)
		}
		phase = derivedPhase
	}
	operation, err := servicemanager.BuildWorkRequestAsyncOperation(status, artifactWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        action,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    artifactString(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    phase,
	})
	if err != nil {
		return nil, err
	}
	if message := strings.TrimSpace(artifactWorkRequestMessage(operation.Phase, current)); message != "" {
		operation.Message = message
	}
	return operation, nil
}

func recoverArtifactIDFromWorkRequest(
	_ *marketplacepublisherv1beta1.Artifact,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := artifactWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if id, ok := resolveArtifactIDFromWorkRequestResources(current.Resources, phase, true); ok {
		return id, nil
	}
	if id, ok := resolveArtifactIDFromWorkRequestResources(current.Resources, phase, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose an artifact identifier", artifactKind, artifactString(current.Id))
}

func resolveArtifactIDFromWorkRequestResources(
	resources []marketplacepublishersdk.WorkRequestResource,
	phase shared.OSOKAsyncPhase,
	requireActionMatch bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if !isArtifactWorkRequestResource(resource) {
			continue
		}
		if requireActionMatch && !artifactWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		id := strings.TrimSpace(artifactString(resource.Identifier))
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

func isArtifactWorkRequestResource(resource marketplacepublishersdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(artifactString(resource.EntityType)))
	if entityType == "artifact" || strings.Contains(entityType, "artifact") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(artifactString(resource.EntityUri)))
	return strings.Contains(entityURI, "/artifacts/")
}

func artifactWorkRequestActionMatchesPhase(action marketplacepublishersdk.ActionTypeEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == marketplacepublishersdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return action == marketplacepublishersdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return action == marketplacepublishersdk.ActionTypeDeleted
	default:
		return false
	}
}

func artifactWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := artifactWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", artifactKind, phase, artifactString(current.Id), current.Status)
}

func artifactWorkRequestFromAny(workRequest any) (marketplacepublishersdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case marketplacepublishersdk.WorkRequest:
		return current, nil
	case *marketplacepublishersdk.WorkRequest:
		if current == nil {
			return marketplacepublishersdk.WorkRequest{}, fmt.Errorf("%s work request is nil", artifactKind)
		}
		return *current, nil
	default:
		return marketplacepublishersdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", artifactKind, workRequest)
	}
}

func artifactFromResponse(response any) (marketplacepublishersdk.Artifact, bool) {
	switch current := response.(type) {
	case marketplacepublishersdk.GetArtifactResponse:
		return artifactFromBody(current.Artifact)
	case *marketplacepublishersdk.GetArtifactResponse:
		return artifactFromGetResponsePointer(current)
	case artifactProjectedResponse:
		return current.Artifact, strings.TrimSpace(current.Artifact.Id) != ""
	case *artifactProjectedResponse:
		return artifactFromProjectedResponsePointer(current)
	case artifactStatusProjection:
		return current, strings.TrimSpace(current.Id) != ""
	case *artifactStatusProjection:
		return artifactFromStatusProjectionPointer(current)
	case marketplacepublishersdk.Artifact:
		return artifactFromBody(current)
	default:
		return nil, false
	}
}

func artifactFromGetResponsePointer(
	current *marketplacepublishersdk.GetArtifactResponse,
) (marketplacepublishersdk.Artifact, bool) {
	if current == nil {
		return nil, false
	}
	return artifactFromBody(current.Artifact)
}

func artifactFromProjectedResponsePointer(
	current *artifactProjectedResponse,
) (marketplacepublishersdk.Artifact, bool) {
	if current == nil {
		return nil, false
	}
	return current.Artifact, strings.TrimSpace(current.Artifact.Id) != ""
}

func artifactFromStatusProjectionPointer(
	current *artifactStatusProjection,
) (marketplacepublishersdk.Artifact, bool) {
	if current == nil {
		return nil, false
	}
	return *current, strings.TrimSpace(current.Id) != ""
}

func artifactFromBody(current marketplacepublishersdk.Artifact) (marketplacepublishersdk.Artifact, bool) {
	if current == nil {
		return nil, false
	}
	return current, true
}

func artifactTypeFromModel(model any) string {
	values, err := artifactJSONMap(model)
	if err != nil {
		return ""
	}
	artifactType, err := normalizeArtifactType(artifactStringPayload(values, "artifactType"))
	if err != nil {
		return ""
	}
	return artifactType
}

func normalizeArtifactType(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	artifactType, ok := marketplacepublishersdk.GetMappingArtifactTypeEnumEnum(raw)
	if !ok {
		return "", fmt.Errorf("%s spec.artifactType %q is not supported", artifactKind, raw)
	}
	return string(artifactType), nil
}

func artifactObjectPayload(payload map[string]any, key string) (map[string]any, error) {
	if current, ok := artifactNestedObject(payload, key); ok {
		return artifactCloneAnyMap(current), nil
	}
	if _, exists := payload[key]; exists {
		return nil, fmt.Errorf("%s spec.%s must be a JSON object", artifactKind, key)
	}
	return map[string]any{}, nil
}

func artifactNestedObject(payload map[string]any, path ...string) (map[string]any, bool) {
	if len(path) == 0 {
		return payload, true
	}
	var current any = payload
	for _, segment := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	object, ok := current.(map[string]any)
	return object, ok
}

func artifactNestedValue(payload map[string]any, path ...string) (any, bool) {
	if len(path) == 0 {
		return payload, true
	}
	var current any = payload
	for _, segment := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func artifactStringPayload(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch current := value.(type) {
	case string:
		return strings.TrimSpace(current)
	default:
		return strings.TrimSpace(fmt.Sprint(current))
	}
}

func setArtifactStringPayload(payload map[string]any, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	payload[key] = value
}

func artifactSliceValue(value any) []any {
	switch current := value.(type) {
	case []any:
		return current
	case []string:
		values := make([]any, 0, len(current))
		for _, item := range current {
			values = append(values, item)
		}
		return values
	case []map[string]any:
		values := make([]any, 0, len(current))
		for _, item := range current {
			values = append(values, item)
		}
		return values
	default:
		return nil
	}
}

func artifactJSONMap(value any) (map[string]any, error) {
	payload := map[string]any{}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func artifactCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func artifactDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&spec)
}

func artifactStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if values == nil {
			converted[namespace] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	return converted
}

func cloneArtifactStatusDefinedTags(input map[string]shared.MapValue) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = value
		}
		cloned[namespace] = tagValues
	}
	return cloned
}

func artifactCloneAnyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func artifactString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func artifactOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func artifactSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func artifactStringPtrEqual(value *string, want string) bool {
	return strings.TrimSpace(artifactString(value)) == strings.TrimSpace(want)
}

var _ interface{ GetOpcRequestID() string } = artifactAmbiguousNotFoundError{}
