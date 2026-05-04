/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package discoveryjob

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
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

const (
	discoveryJobKind                         = "DiscoveryJob"
	discoveryJobCreateOnlyStatePrefix        = "DiscoveryJob create-only fields: "
	discoveryJobCreateOnlyPropagationField   = "shouldPropagateTagsToDiscoveredResources"
	discoveryJobCreateOnlyCredentialsField   = "discoveryDetails.credentialsGeneration"
	discoveryJobCreateOnlyCredentialsNone    = "none"
	discoveryJobLegacyCreateOnlyStatusMarker = "discoveryjob-create-only-fingerprint:"
)

type discoveryJobOCIClient interface {
	CreateDiscoveryJob(context.Context, stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error)
	GetDiscoveryJob(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error)
	ListDiscoveryJobs(context.Context, stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error)
	DeleteDiscoveryJob(context.Context, stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error)
}

type discoveryJobIdentity struct {
	compartmentID string
	resourceName  string
	resourceType  string
	discoveryType string
	license       string
}

type discoveryJobRuntimeClient struct {
	delegate DiscoveryJobServiceClient
	get      func(context.Context, stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error)
	list     func(context.Context, stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error)
}

type discoveryJobStatusProjection struct {
	Id               string                                              `json:"id,omitempty"`
	CompartmentId    string                                              `json:"compartmentId,omitempty"`
	DiscoveryType    string                                              `json:"discoveryType,omitempty"`
	Status           string                                              `json:"sdkStatus,omitempty"`
	StatusMessage    string                                              `json:"statusMessage,omitempty"`
	TenantId         string                                              `json:"tenantId,omitempty"`
	UserId           string                                              `json:"userId,omitempty"`
	DiscoveryClient  string                                              `json:"discoveryClient,omitempty"`
	DiscoveryDetails stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails `json:"discoveryDetails,omitempty"`
	TimeUpdated      string                                              `json:"timeUpdated,omitempty"`
	LifecycleState   string                                              `json:"lifecycleState,omitempty"`
	FreeformTags     map[string]string                                   `json:"freeformTags,omitempty"`
	DefinedTags      map[string]shared.MapValue                          `json:"definedTags,omitempty"`
	SystemTags       map[string]shared.MapValue                          `json:"systemTags,omitempty"`
	ResourceType     string                                              `json:"resourceType,omitempty"`
	ResourceName     string                                              `json:"resourceName,omitempty"`
	License          string                                              `json:"license,omitempty"`
}

type discoveryJobCreateOnlyState struct {
	shouldPropagateTagsToDiscoveredResources bool
	credentialsTracked                       bool
	credentialsPresent                       bool
	credentialsGeneration                    int64
}

type discoveryJobProjectedResponse struct {
	DiscoveryJob discoveryJobStatusProjection `presentIn:"body"`
	OpcRequestId *string                      `presentIn:"header" name:"opc-request-id"`
}

type discoveryJobProjectedCollection struct {
	Items []discoveryJobStatusProjection `json:"items,omitempty"`
}

type discoveryJobProjectedListResponse struct {
	DiscoveryJobCollection discoveryJobProjectedCollection `presentIn:"body"`
	OpcRequestId           *string                         `presentIn:"header" name:"opc-request-id"`
	OpcNextPage            *string                         `presentIn:"header" name:"opc-next-page"`
}

type discoveryJobAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e discoveryJobAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e discoveryJobAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerDiscoveryJobRuntimeHooksMutator(func(_ *DiscoveryJobServiceManager, hooks *DiscoveryJobRuntimeHooks) {
		applyDiscoveryJobRuntimeHooks(hooks)
	})
}

func applyDiscoveryJobRuntimeHooks(hooks *DiscoveryJobRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = discoveryJobRuntimeSemantics()
	hooks.BuildCreateBody = buildDiscoveryJobCreateBody
	hooks.Identity.Resolve = resolveDiscoveryJobIdentity
	hooks.Identity.RecordPath = recordDiscoveryJobPathIdentity
	hooks.Create.Fields = discoveryJobCreateFields()
	hooks.Get.Fields = discoveryJobGetFields()
	hooks.List.Fields = discoveryJobListFields()
	hooks.Delete.Fields = discoveryJobDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listDiscoveryJobsAllPages(hooks.List.Call)
	}
	installDiscoveryJobProjectedReadOperations(hooks)
	hooks.DeleteHooks.HandleError = handleDiscoveryJobDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyDiscoveryJobDeleteOutcome
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDiscoveryJobCreateOnlyDrift
	hooks.StatusHooks.ProjectStatus = projectDiscoveryJobStatus
	hooks.StatusHooks.MarkTerminating = markDiscoveryJobTerminating
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DiscoveryJobServiceClient) DiscoveryJobServiceClient {
		return discoveryJobRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
			list:     hooks.List.Call,
		}
	})
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DiscoveryJobServiceClient) DiscoveryJobServiceClient {
		return discoveryJobCreateOnlyTrackingClient{delegate: delegate}
	})
}

func newDiscoveryJobServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client discoveryJobOCIClient,
) DiscoveryJobServiceClient {
	hooks := newDiscoveryJobRuntimeHooksWithOCIClient(client)
	applyDiscoveryJobRuntimeHooks(&hooks)
	manager := &DiscoveryJobServiceManager{Log: log}
	delegate := defaultDiscoveryJobServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.DiscoveryJob](
			buildDiscoveryJobGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDiscoveryJobGeneratedClient(hooks, delegate)
}

func newDiscoveryJobRuntimeHooksWithOCIClient(client discoveryJobOCIClient) DiscoveryJobRuntimeHooks {
	return DiscoveryJobRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*stackmonitoringv1beta1.DiscoveryJob]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*stackmonitoringv1beta1.DiscoveryJob]{},
		StatusHooks:     generatedruntime.StatusHooks[*stackmonitoringv1beta1.DiscoveryJob]{},
		ParityHooks:     generatedruntime.ParityHooks[*stackmonitoringv1beta1.DiscoveryJob]{},
		Async:           generatedruntime.AsyncHooks[*stackmonitoringv1beta1.DiscoveryJob]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*stackmonitoringv1beta1.DiscoveryJob]{},
		Create: runtimeOperationHooks[stackmonitoringsdk.CreateDiscoveryJobRequest, stackmonitoringsdk.CreateDiscoveryJobResponse]{
			Fields: discoveryJobCreateFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.CreateDiscoveryJobRequest) (stackmonitoringsdk.CreateDiscoveryJobResponse, error) {
				if client == nil {
					return stackmonitoringsdk.CreateDiscoveryJobResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.CreateDiscoveryJob(ctx, request)
			},
		},
		Get: runtimeOperationHooks[stackmonitoringsdk.GetDiscoveryJobRequest, stackmonitoringsdk.GetDiscoveryJobResponse]{
			Fields: discoveryJobGetFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.GetDiscoveryJobRequest) (stackmonitoringsdk.GetDiscoveryJobResponse, error) {
				if client == nil {
					return stackmonitoringsdk.GetDiscoveryJobResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.GetDiscoveryJob(ctx, request)
			},
		},
		List: runtimeOperationHooks[stackmonitoringsdk.ListDiscoveryJobsRequest, stackmonitoringsdk.ListDiscoveryJobsResponse]{
			Fields: discoveryJobListFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error) {
				if client == nil {
					return stackmonitoringsdk.ListDiscoveryJobsResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.ListDiscoveryJobs(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[stackmonitoringsdk.DeleteDiscoveryJobRequest, stackmonitoringsdk.DeleteDiscoveryJobResponse]{
			Fields: discoveryJobDeleteFields(),
			Call: func(ctx context.Context, request stackmonitoringsdk.DeleteDiscoveryJobRequest) (stackmonitoringsdk.DeleteDiscoveryJobResponse, error) {
				if client == nil {
					return stackmonitoringsdk.DeleteDiscoveryJobResponse{}, fmt.Errorf("DiscoveryJob OCI client is nil")
				}
				return client.DeleteDiscoveryJob(ctx, request)
			},
		},
		WrapGeneratedClient: []func(DiscoveryJobServiceClient) DiscoveryJobServiceClient{},
	}
}

func discoveryJobRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "stackmonitoring",
		FormalSlug:        "discoveryjob",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{
				string(stackmonitoringsdk.LifecycleStateCreating),
				string(stackmonitoringsdk.DiscoveryJobStatusCreated),
				string(stackmonitoringsdk.DiscoveryJobStatusInprogress),
			},
			UpdatingStates: []string{string(stackmonitoringsdk.LifecycleStateUpdating)},
			ActiveStates: []string{
				string(stackmonitoringsdk.LifecycleStateActive),
				string(stackmonitoringsdk.DiscoveryJobStatusSuccess),
				string(stackmonitoringsdk.DiscoveryJobStatusInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(stackmonitoringsdk.LifecycleStateDeleting),
			},
			TerminalStates: []string{
				string(stackmonitoringsdk.LifecycleStateDeleted),
				string(stackmonitoringsdk.DiscoveryJobStatusDeleted),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"resourceName",
				"resourceType",
				"discoveryType",
				"license",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew: []string{
				"compartmentId",
				"discoveryDetails.agentId",
				"discoveryDetails.resourceType",
				"discoveryDetails.resourceName",
				"discoveryDetails.properties",
				"discoveryDetails.license",
				"discoveryDetails.credentials",
				"discoveryDetails.tags",
				"discoveryType",
				"discoveryClient",
				"shouldPropagateTagsToDiscoveredResources",
				"freeformTags",
				"definedTags",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: discoveryJobKind, Action: "CreateDiscoveryJob"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: discoveryJobKind, Action: "DeleteDiscoveryJob"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: discoveryJobKind, Action: "GetDiscoveryJob"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: discoveryJobKind, Action: "GetDiscoveryJob"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func discoveryJobCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDiscoveryJobDetails", RequestName: "CreateDiscoveryJobDetails", Contribution: "body"},
	}
}

func discoveryJobGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DiscoveryJobId", RequestName: "discoveryJobId", Contribution: "path", PreferResourceID: true},
	}
}

func discoveryJobListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "Name",
			RequestName:  "name",
			Contribution: "query",
			LookupPaths: []string{
				"status.resourceName",
				"spec.discoveryDetails.resourceName",
				"discoveryDetails.resourceName",
				"resourceName",
			},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func discoveryJobDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DiscoveryJobId", RequestName: "discoveryJobId", Contribution: "path", PreferResourceID: true},
	}
}

func buildDiscoveryJobCreateBody(_ context.Context, resource *stackmonitoringv1beta1.DiscoveryJob, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", discoveryJobKind)
	}
	details := stackmonitoringsdk.CreateDiscoveryJobDetails{
		CompartmentId:    common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DiscoveryDetails: discoveryJobDetailsFromSpec(resource.Spec.DiscoveryDetails),
		DiscoveryType:    stackmonitoringsdk.CreateDiscoveryJobDetailsDiscoveryTypeEnum(strings.TrimSpace(resource.Spec.DiscoveryType)),
		DiscoveryClient:  discoveryJobOptionalString(resource.Spec.DiscoveryClient),
		ShouldPropagateTagsToDiscoveredResources: common.Bool(
			resource.Spec.ShouldPropagateTagsToDiscoveredResources,
		),
		FreeformTags: discoveryJobStringMap(resource.Spec.FreeformTags),
		DefinedTags:  discoveryJobDefinedTags(resource.Spec.DefinedTags),
	}
	return details, nil
}

func discoveryJobDetailsFromSpec(spec stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails) *stackmonitoringsdk.DiscoveryDetails {
	return &stackmonitoringsdk.DiscoveryDetails{
		AgentId:      common.String(strings.TrimSpace(spec.AgentId)),
		ResourceType: stackmonitoringsdk.DiscoveryDetailsResourceTypeEnum(strings.TrimSpace(spec.ResourceType)),
		ResourceName: common.String(strings.TrimSpace(spec.ResourceName)),
		Properties:   discoveryJobPropertyDetails(spec.Properties),
		License:      stackmonitoringsdk.LicenseTypeEnum(strings.TrimSpace(spec.License)),
		Credentials:  discoveryJobCredentialCollection(spec.Credentials),
		Tags:         discoveryJobPropertyDetailsIfSet(spec.Tags),
	}
}

func discoveryJobPropertyDetails(spec stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsProperties) *stackmonitoringsdk.PropertyDetails {
	return &stackmonitoringsdk.PropertyDetails{PropertiesMap: discoveryJobStringMap(spec.PropertiesMap)}
}

func discoveryJobPropertyDetailsIfSet(spec stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsTags) *stackmonitoringsdk.PropertyDetails {
	if spec.PropertiesMap == nil {
		return nil
	}
	return &stackmonitoringsdk.PropertyDetails{PropertiesMap: discoveryJobStringMap(spec.PropertiesMap)}
}

func discoveryJobCredentialCollection(spec stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentials) *stackmonitoringsdk.CredentialCollection {
	if len(spec.Items) == 0 {
		return nil
	}
	items := make([]stackmonitoringsdk.CredentialDetails, 0, len(spec.Items))
	for _, item := range spec.Items {
		items = append(items, stackmonitoringsdk.CredentialDetails{
			CredentialName: common.String(strings.TrimSpace(item.CredentialName)),
			CredentialType: common.String(strings.TrimSpace(item.CredentialType)),
			Properties: &stackmonitoringsdk.PropertyDetails{
				PropertiesMap: discoveryJobStringMap(item.Properties.PropertiesMap),
			},
		})
	}
	return &stackmonitoringsdk.CredentialCollection{Items: items}
}

func resolveDiscoveryJobIdentity(resource *stackmonitoringv1beta1.DiscoveryJob) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", discoveryJobKind)
	}
	return discoveryJobIdentityFromResource(resource), nil
}

func discoveryJobIdentityFromResource(resource *stackmonitoringv1beta1.DiscoveryJob) discoveryJobIdentity {
	if resource == nil {
		return discoveryJobIdentity{}
	}
	return discoveryJobIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		resourceName:  strings.TrimSpace(resource.Spec.DiscoveryDetails.ResourceName),
		resourceType:  strings.TrimSpace(resource.Spec.DiscoveryDetails.ResourceType),
		discoveryType: strings.TrimSpace(resource.Spec.DiscoveryType),
		license:       strings.TrimSpace(resource.Spec.DiscoveryDetails.License),
	}
}

func validateTrackedDiscoveryJobIdentity(resource *stackmonitoringv1beta1.DiscoveryJob, identity discoveryJobIdentity) error {
	if resource == nil {
		return nil
	}
	checks := []struct {
		field   string
		tracked string
		desired string
	}{
		{field: "compartmentId", tracked: resource.Status.CompartmentId, desired: identity.compartmentID},
		{field: "resourceName", tracked: resource.Status.ResourceName, desired: identity.resourceName},
		{field: "resourceType", tracked: resource.Status.ResourceType, desired: identity.resourceType},
		{field: "discoveryType", tracked: resource.Status.DiscoveryType, desired: identity.discoveryType},
		{field: "license", tracked: resource.Status.License, desired: identity.license},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.tracked) == "" || strings.TrimSpace(check.desired) == "" {
			continue
		}
		if strings.TrimSpace(check.tracked) != strings.TrimSpace(check.desired) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", discoveryJobKind, check.field)
		}
	}
	return nil
}

func recordDiscoveryJobPathIdentity(resource *stackmonitoringv1beta1.DiscoveryJob, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(discoveryJobIdentity)
	if !ok {
		return
	}
	if typed.compartmentID != "" {
		resource.Status.CompartmentId = typed.compartmentID
	}
	if typed.resourceName != "" {
		resource.Status.ResourceName = typed.resourceName
	}
	if typed.resourceType != "" {
		resource.Status.ResourceType = typed.resourceType
	}
	if typed.discoveryType != "" {
		resource.Status.DiscoveryType = typed.discoveryType
	}
	if typed.license != "" {
		resource.Status.License = typed.license
	}
}

func listDiscoveryJobsAllPages(
	call func(context.Context, stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error),
) func(context.Context, stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error) {
	return func(ctx context.Context, request stackmonitoringsdk.ListDiscoveryJobsRequest) (stackmonitoringsdk.ListDiscoveryJobsResponse, error) {
		var combined stackmonitoringsdk.ListDiscoveryJobsResponse
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

func installDiscoveryJobProjectedReadOperations(hooks *DiscoveryJobRuntimeHooks) {
	if hooks.Get.Call != nil {
		getFields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
		hooks.Read.Get = &generatedruntime.Operation{
			NewRequest: func() any { return &stackmonitoringsdk.GetDiscoveryJobRequest{} },
			Fields:     getFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*stackmonitoringsdk.GetDiscoveryJobRequest))
				if err != nil {
					return nil, err
				}
				return discoveryJobProjectedResponseFromSDK(response.DiscoveryJob, response.OpcRequestId), nil
			},
		}
	}
	if hooks.List.Call != nil {
		listFields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
		hooks.Read.List = &generatedruntime.Operation{
			NewRequest: func() any { return &stackmonitoringsdk.ListDiscoveryJobsRequest{} },
			Fields:     listFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*stackmonitoringsdk.ListDiscoveryJobsRequest))
				if err != nil {
					return nil, err
				}
				return discoveryJobProjectedListResponseFromSDK(response), nil
			},
		}
	}
}

func handleDiscoveryJobDeleteError(resource *stackmonitoringv1beta1.DiscoveryJob, err error) error {
	if err == nil {
		return nil
	}
	if ambiguous := discoveryJobAmbiguousDeleteError(resource, err, "delete path"); ambiguous != nil {
		return ambiguous
	}
	return err
}

func applyDiscoveryJobDeleteOutcome(
	resource *stackmonitoringv1beta1.DiscoveryJob,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	state := discoveryJobLifecycleState(response)
	if state == "" {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if discoveryJobTerminalDeleteState(state) {
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: true}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest {
		markDiscoveryJobTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func markDiscoveryJobTerminating(resource *stackmonitoringv1beta1.DiscoveryJob, _ any) {
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

func discoveryJobLifecycleState(response any) string {
	if projected, ok := discoveryJobProjectionFromProjectedResponse(response); ok {
		return discoveryJobProjectionLifecycleState(projected)
	}
	if current, ok := discoveryJobFromResponse(response); ok {
		return discoveryJobState(current.LifecycleState, current.Status)
	}
	if summary, ok := discoveryJobSummaryFromResponse(response); ok {
		return discoveryJobSummaryState(summary.LifecycleState, summary.Status)
	}
	return ""
}

func discoveryJobProjectionLifecycleState(current discoveryJobStatusProjection) string {
	if current.LifecycleState != "" {
		return strings.ToUpper(current.LifecycleState)
	}
	return strings.ToUpper(current.Status)
}

func discoveryJobState(lifecycleState stackmonitoringsdk.LifecycleStateEnum, status stackmonitoringsdk.DiscoveryJobStatusEnum) string {
	if lifecycleState != "" {
		return strings.ToUpper(string(lifecycleState))
	}
	return strings.ToUpper(string(status))
}

func discoveryJobSummaryState(lifecycleState stackmonitoringsdk.LifecycleStateEnum, status stackmonitoringsdk.DiscoveryJobSummaryStatusEnum) string {
	if lifecycleState != "" {
		return strings.ToUpper(string(lifecycleState))
	}
	return strings.ToUpper(string(status))
}

func discoveryJobTerminalDeleteState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case string(stackmonitoringsdk.LifecycleStateDeleted):
		return true
	default:
		return false
	}
}

func projectDiscoveryJobStatus(resource *stackmonitoringv1beta1.DiscoveryJob, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", discoveryJobKind)
	}
	projected, ok := discoveryJobProjectionFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = stackmonitoringv1beta1.DiscoveryJobStatus{
		OsokStatus:       osokStatus,
		Id:               projected.Id,
		CompartmentId:    projected.CompartmentId,
		DiscoveryType:    projected.DiscoveryType,
		Status:           projected.Status,
		StatusMessage:    projected.StatusMessage,
		TenantId:         projected.TenantId,
		UserId:           projected.UserId,
		DiscoveryClient:  projected.DiscoveryClient,
		DiscoveryDetails: projected.DiscoveryDetails,
		TimeUpdated:      projected.TimeUpdated,
		LifecycleState:   projected.LifecycleState,
		FreeformTags:     discoveryJobStringMap(projected.FreeformTags),
		DefinedTags:      discoveryJobCloneSharedTags(projected.DefinedTags),
		SystemTags:       discoveryJobCloneSharedTags(projected.SystemTags),
		ResourceType:     projected.ResourceType,
		ResourceName:     projected.ResourceName,
		License:          projected.License,
	}
	if projected.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(projected.Id)
	}
	return nil
}

func discoveryJobProjectedResponseFromSDK(
	current stackmonitoringsdk.DiscoveryJob,
	opcRequestID *string,
) discoveryJobProjectedResponse {
	return discoveryJobProjectedResponse{
		DiscoveryJob: discoveryJobStatusProjectionFromSDK(current),
		OpcRequestId: opcRequestID,
	}
}

func discoveryJobProjectedListResponseFromSDK(
	response stackmonitoringsdk.ListDiscoveryJobsResponse,
) discoveryJobProjectedListResponse {
	projected := discoveryJobProjectedListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		projected.DiscoveryJobCollection.Items = append(projected.DiscoveryJobCollection.Items, discoveryJobStatusProjectionFromSummary(item))
	}
	return projected
}

func discoveryJobStatusProjectionFromSDK(current stackmonitoringsdk.DiscoveryJob) discoveryJobStatusProjection {
	projection := discoveryJobStatusProjection{
		Id:               discoveryJobStringValue(current.Id),
		CompartmentId:    discoveryJobStringValue(current.CompartmentId),
		DiscoveryType:    string(current.DiscoveryType),
		Status:           string(current.Status),
		StatusMessage:    discoveryJobStringValue(current.StatusMessage),
		TenantId:         discoveryJobStringValue(current.TenantId),
		UserId:           discoveryJobStringValue(current.UserId),
		DiscoveryClient:  discoveryJobStringValue(current.DiscoveryClient),
		DiscoveryDetails: discoveryJobAPIDiscoveryDetails(current.DiscoveryDetails),
		TimeUpdated:      discoveryJobSDKTimeString(current.TimeUpdated),
		LifecycleState:   string(current.LifecycleState),
		FreeformTags:     discoveryJobStringMap(current.FreeformTags),
		DefinedTags:      discoveryJobSharedTags(current.DefinedTags),
		SystemTags:       discoveryJobSharedTags(current.SystemTags),
	}
	projection.ResourceType = projection.DiscoveryDetails.ResourceType
	projection.ResourceName = projection.DiscoveryDetails.ResourceName
	projection.License = projection.DiscoveryDetails.License
	if projection.LifecycleState == "" {
		projection.LifecycleState = projection.Status
	}
	return projection
}

func discoveryJobStatusProjectionFromSummary(current stackmonitoringsdk.DiscoveryJobSummary) discoveryJobStatusProjection {
	projection := discoveryJobStatusProjection{
		Id:             discoveryJobStringValue(current.Id),
		CompartmentId:  discoveryJobStringValue(current.CompartmentId),
		DiscoveryType:  string(current.DiscoveryType),
		Status:         string(current.Status),
		StatusMessage:  discoveryJobStringValue(current.StatusMessage),
		TenantId:       discoveryJobStringValue(current.TenantId),
		UserId:         discoveryJobStringValue(current.UserId),
		TimeUpdated:    discoveryJobSDKTimeString(current.TimeUpdated),
		LifecycleState: string(current.LifecycleState),
		FreeformTags:   discoveryJobStringMap(current.FreeformTags),
		DefinedTags:    discoveryJobSharedTags(current.DefinedTags),
		SystemTags:     discoveryJobSharedTags(current.SystemTags),
		ResourceType:   string(current.ResourceType),
		ResourceName:   discoveryJobStringValue(current.ResourceName),
		License:        string(current.License),
	}
	projection.DiscoveryDetails.ResourceType = projection.ResourceType
	projection.DiscoveryDetails.ResourceName = projection.ResourceName
	projection.DiscoveryDetails.License = projection.License
	if projection.LifecycleState == "" {
		projection.LifecycleState = projection.Status
	}
	return projection
}

func discoveryJobProjectionFromResponse(response any) (discoveryJobStatusProjection, bool) {
	if projected, ok := discoveryJobProjectionFromProjectedResponse(response); ok {
		return projected, true
	}
	if current, ok := discoveryJobFromResponse(response); ok {
		return discoveryJobStatusProjectionFromSDK(current), true
	}
	if summary, ok := discoveryJobSummaryFromResponse(response); ok {
		return discoveryJobStatusProjectionFromSummary(summary), true
	}
	return discoveryJobStatusProjection{}, false
}

func discoveryJobProjectionFromProjectedResponse(response any) (discoveryJobStatusProjection, bool) {
	switch current := response.(type) {
	case discoveryJobProjectedResponse:
		return current.DiscoveryJob, true
	case *discoveryJobProjectedResponse:
		if current == nil {
			return discoveryJobStatusProjection{}, false
		}
		return current.DiscoveryJob, true
	case discoveryJobStatusProjection:
		return current, true
	case *discoveryJobStatusProjection:
		if current == nil {
			return discoveryJobStatusProjection{}, false
		}
		return *current, true
	default:
		return discoveryJobStatusProjection{}, false
	}
}

func discoveryJobFromResponse(response any) (stackmonitoringsdk.DiscoveryJob, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.GetDiscoveryJobResponse:
		return current.DiscoveryJob, true
	case *stackmonitoringsdk.GetDiscoveryJobResponse:
		if current == nil {
			return stackmonitoringsdk.DiscoveryJob{}, false
		}
		return current.DiscoveryJob, true
	case stackmonitoringsdk.CreateDiscoveryJobResponse:
		return current.DiscoveryJob, true
	case *stackmonitoringsdk.CreateDiscoveryJobResponse:
		if current == nil {
			return stackmonitoringsdk.DiscoveryJob{}, false
		}
		return current.DiscoveryJob, true
	case stackmonitoringsdk.DiscoveryJob:
		return current, true
	case *stackmonitoringsdk.DiscoveryJob:
		if current == nil {
			return stackmonitoringsdk.DiscoveryJob{}, false
		}
		return *current, true
	default:
		return stackmonitoringsdk.DiscoveryJob{}, false
	}
}

func discoveryJobSummaryFromResponse(response any) (stackmonitoringsdk.DiscoveryJobSummary, bool) {
	switch current := response.(type) {
	case stackmonitoringsdk.DiscoveryJobSummary:
		return current, true
	case *stackmonitoringsdk.DiscoveryJobSummary:
		if current == nil {
			return stackmonitoringsdk.DiscoveryJobSummary{}, false
		}
		return *current, true
	default:
		return stackmonitoringsdk.DiscoveryJobSummary{}, false
	}
}

func (c discoveryJobRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.DiscoveryJob,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DiscoveryJob runtime client is not configured")
	}
	if err := validateDiscoveryJobCreateOrUpdateIdentity(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, markDiscoveryJobFailed(resource, err)
	}
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	normalizeDiscoveryJobStatus(resource)
	return response, err
}

func validateDiscoveryJobCreateOrUpdateIdentity(resource *stackmonitoringv1beta1.DiscoveryJob) error {
	if trackedDiscoveryJobID(resource) == "" {
		return nil
	}
	return validateTrackedDiscoveryJobIdentity(resource, discoveryJobIdentityFromResource(resource))
}

func markDiscoveryJobFailed(resource *stackmonitoringv1beta1.DiscoveryJob, err error) error {
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

func (c discoveryJobRuntimeClient) Delete(ctx context.Context, resource *stackmonitoringv1beta1.DiscoveryJob) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("DiscoveryJob runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	deleted, err := c.delegate.Delete(ctx, resource)
	normalizeDiscoveryJobStatus(resource)
	return deleted, err
}

type discoveryJobCreateOnlyTrackingClient struct {
	delegate DiscoveryJobServiceClient
}

func (c discoveryJobCreateOnlyTrackingClient) CreateOrUpdate(
	ctx context.Context,
	resource *stackmonitoringv1beta1.DiscoveryJob,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DiscoveryJob runtime client is not configured")
	}
	recorded, hasRecorded := discoveryJobRecordedCreateOnlyState(resource)
	desired := discoveryJobCreateOnlyStateFromResource(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	switch {
	case err == nil && response.IsSuccessful && discoveryJobHasTrackedIdentity(resource):
		setDiscoveryJobCreateOnlyState(resource, desired)
	case hasRecorded && discoveryJobHasTrackedIdentity(resource):
		setDiscoveryJobCreateOnlyState(resource, recorded)
	}
	return response, err
}

func (c discoveryJobCreateOnlyTrackingClient) Delete(
	ctx context.Context,
	resource *stackmonitoringv1beta1.DiscoveryJob,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("DiscoveryJob runtime client is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c discoveryJobRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *stackmonitoringv1beta1.DiscoveryJob,
) error {
	if resource == nil {
		return nil
	}
	currentID := trackedDiscoveryJobID(resource)
	if currentID != "" {
		return c.rejectAuthShapedGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedList(ctx, resource)
}

func (c discoveryJobRuntimeClient) rejectAuthShapedGet(
	ctx context.Context,
	resource *stackmonitoringv1beta1.DiscoveryJob,
	currentID string,
) error {
	if c.get == nil {
		return nil
	}
	_, err := c.get(ctx, stackmonitoringsdk.GetDiscoveryJobRequest{DiscoveryJobId: common.String(currentID)})
	return discoveryJobAmbiguousDeleteError(resource, err, "pre-delete get")
}

func (c discoveryJobRuntimeClient) rejectAuthShapedList(
	ctx context.Context,
	resource *stackmonitoringv1beta1.DiscoveryJob,
) error {
	if c.list == nil {
		return nil
	}
	identity := discoveryJobIdentityFromResource(resource)
	if identity.compartmentID == "" || identity.resourceName == "" {
		return nil
	}
	_, err := c.list(ctx, stackmonitoringsdk.ListDiscoveryJobsRequest{
		CompartmentId: common.String(identity.compartmentID),
		Name:          common.String(identity.resourceName),
	})
	return discoveryJobAmbiguousDeleteError(resource, err, "pre-delete list")
}

func discoveryJobAmbiguousDeleteError(
	resource *stackmonitoringv1beta1.DiscoveryJob,
	err error,
	operation string,
) error {
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return discoveryJobAmbiguousNotFoundError{
		message:      fmt.Sprintf("%s %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", discoveryJobKind, operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func normalizeDiscoveryJobStatus(resource *stackmonitoringv1beta1.DiscoveryJob) {
	if resource == nil {
		return
	}
	resource.Status.DiscoveryDetails.Credentials = stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentials{}
	if resource.Status.Id != "" && resource.Status.OsokStatus.Ocid == "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	if details := resource.Status.DiscoveryDetails; details.ResourceName != "" {
		resource.Status.ResourceName = details.ResourceName
		resource.Status.ResourceType = details.ResourceType
		resource.Status.License = details.License
	}
}

func validateDiscoveryJobCreateOnlyDrift(resource *stackmonitoringv1beta1.DiscoveryJob, _ any) error {
	if resource == nil || !discoveryJobHasTrackedIdentity(resource) {
		return nil
	}
	recorded, ok := discoveryJobRecordedCreateOnlyState(resource)
	if !ok {
		return nil
	}
	desired := discoveryJobCreateOnlyStateFromResource(resource)
	if !recorded.credentialsTracked && !desired.credentialsPresent {
		recorded.credentialsTracked = true
	}
	if desired != recorded {
		if desired.credentialsTracked != recorded.credentialsTracked ||
			desired.credentialsPresent != recorded.credentialsPresent ||
			desired.credentialsGeneration != recorded.credentialsGeneration {
			return fmt.Errorf(
				"%s formal semantics require replacement when %s changes; recorded %s, desired %s",
				discoveryJobKind,
				"discoveryDetails.credentials",
				discoveryJobCredentialStateDescription(recorded),
				discoveryJobCredentialStateDescription(desired),
			)
		}
		return fmt.Errorf("%s formal semantics require replacement when create-only fields change, including %s", discoveryJobKind, discoveryJobCreateOnlyPropagationField)
	}
	return nil
}

func recordDiscoveryJobCreateOnlyState(resource *stackmonitoringv1beta1.DiscoveryJob) {
	if resource == nil {
		return
	}
	setDiscoveryJobCreateOnlyState(resource, discoveryJobCreateOnlyStateFromResource(resource))
}

func discoveryJobCreateOnlyStateFromResource(resource *stackmonitoringv1beta1.DiscoveryJob) discoveryJobCreateOnlyState {
	if resource == nil {
		return discoveryJobCreateOnlyState{}
	}
	state := discoveryJobCreateOnlyState{
		shouldPropagateTagsToDiscoveredResources: resource.Spec.ShouldPropagateTagsToDiscoveredResources,
	}
	state.credentialsTracked = true
	if discoveryJobHasCredentials(resource.Spec.DiscoveryDetails.Credentials) {
		state.credentialsPresent = true
		state.credentialsGeneration = resource.Generation
	}
	return state
}

func setDiscoveryJobCreateOnlyState(resource *stackmonitoringv1beta1.DiscoveryJob, state discoveryJobCreateOnlyState) {
	if resource == nil {
		return
	}
	base := stripDiscoveryJobCreateOnlyState(resource.Status.OsokStatus.Message)
	markers := []string{
		discoveryJobCreateOnlyPropagationMarker(state),
		discoveryJobCreateOnlyCredentialsMarker(state),
	}
	if base == "" {
		resource.Status.OsokStatus.Message = strings.Join(markers, "; ")
		return
	}
	resource.Status.OsokStatus.Message = base + "; " + strings.Join(markers, "; ")
}

func discoveryJobRecordedCreateOnlyState(resource *stackmonitoringv1beta1.DiscoveryJob) (discoveryJobCreateOnlyState, bool) {
	if resource == nil {
		return discoveryJobCreateOnlyState{}, false
	}
	for _, state := range []discoveryJobCreateOnlyState{
		{shouldPropagateTagsToDiscoveredResources: true},
		{shouldPropagateTagsToDiscoveredResources: false},
	} {
		if strings.Contains(resource.Status.OsokStatus.Message, discoveryJobCreateOnlyPropagationMarker(state)) {
			state.credentialsGeneration, state.credentialsPresent, state.credentialsTracked = discoveryJobRecordedCredentialsGeneration(resource.Status.OsokStatus.Message)
			return state, true
		}
	}
	return discoveryJobCreateOnlyState{}, false
}

func discoveryJobCreateOnlyPropagationMarker(state discoveryJobCreateOnlyState) string {
	return fmt.Sprintf("%s%s=%t", discoveryJobCreateOnlyStatePrefix, discoveryJobCreateOnlyPropagationField, state.shouldPropagateTagsToDiscoveredResources)
}

func discoveryJobCreateOnlyCredentialsMarker(state discoveryJobCreateOnlyState) string {
	value := discoveryJobCreateOnlyCredentialsNone
	if state.credentialsPresent {
		value = strconv.FormatInt(state.credentialsGeneration, 10)
	}
	return discoveryJobCreateOnlyCredentialsMarkerPrefix() + value
}

func discoveryJobCreateOnlyCredentialsMarkerPrefix() string {
	return fmt.Sprintf("%s%s=", discoveryJobCreateOnlyStatePrefix, discoveryJobCreateOnlyCredentialsField)
}

func discoveryJobRecordedCredentialsGeneration(raw string) (int64, bool, bool) {
	value, ok := discoveryJobRecordedMarkerValue(raw, discoveryJobCreateOnlyCredentialsMarkerPrefix())
	if !ok {
		return 0, false, false
	}
	if value == discoveryJobCreateOnlyCredentialsNone {
		return 0, false, true
	}
	generation, err := strconv.ParseInt(value, 10, 64)
	if err != nil || generation < 0 {
		return 0, false, false
	}
	return generation, true, true
}

func discoveryJobRecordedMarkerValue(raw string, markerPrefix string) (string, bool) {
	index := strings.LastIndex(raw, markerPrefix)
	if index < 0 {
		return "", false
	}
	start := index + len(markerPrefix)
	end := start
	for end < len(raw) && raw[end] != ';' {
		end++
	}
	value := strings.TrimSpace(raw[start:end])
	if value == "" {
		return "", false
	}
	return value, true
}

func stripDiscoveryJobCreateOnlyState(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = stripDiscoveryJobStatusMessageSegment(raw, discoveryJobCreateOnlyPropagationMarker(discoveryJobCreateOnlyState{shouldPropagateTagsToDiscoveredResources: true}))
	raw = stripDiscoveryJobStatusMessageSegment(raw, discoveryJobCreateOnlyPropagationMarker(discoveryJobCreateOnlyState{shouldPropagateTagsToDiscoveredResources: false}))
	raw = stripDiscoveryJobStatusMessageSegmentPrefix(raw, discoveryJobCreateOnlyCredentialsMarkerPrefix())
	return stripDiscoveryJobLegacyCreateOnlyStatusMarker(raw)
}

func stripDiscoveryJobStatusMessageSegment(raw string, segment string) string {
	index := strings.LastIndex(raw, segment)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	suffix := strings.TrimSpace(strings.TrimLeft(raw[index+len(segment):], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func stripDiscoveryJobStatusMessageSegmentPrefix(raw string, segmentPrefix string) string {
	index := strings.LastIndex(raw, segmentPrefix)
	if index < 0 {
		return raw
	}
	end := index + len(segmentPrefix)
	for end < len(raw) && raw[end] != ';' {
		end++
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func stripDiscoveryJobLegacyCreateOnlyStatusMarker(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, discoveryJobLegacyCreateOnlyStatusMarker)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(discoveryJobLegacyCreateOnlyStatusMarker)
	end := start
	for end < len(raw) && discoveryJobIsHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func discoveryJobIsHexDigit(value byte) bool {
	return ('0' <= value && value <= '9') ||
		('a' <= value && value <= 'f') ||
		('A' <= value && value <= 'F')
}

func discoveryJobCredentialStateDescription(state discoveryJobCreateOnlyState) string {
	if !state.credentialsTracked {
		return "untracked"
	}
	if !state.credentialsPresent {
		return "none"
	}
	return "generation " + strconv.FormatInt(state.credentialsGeneration, 10)
}

func discoveryJobHasCredentials(credentials stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentials) bool {
	return len(credentials.Items) > 0
}

func discoveryJobHasTrackedIdentity(resource *stackmonitoringv1beta1.DiscoveryJob) bool {
	return trackedDiscoveryJobID(resource) != ""
}

func trackedDiscoveryJobID(resource *stackmonitoringv1beta1.DiscoveryJob) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

func discoveryJobOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func discoveryJobStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func discoveryJobSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func discoveryJobAPIDiscoveryDetails(details *stackmonitoringsdk.DiscoveryDetails) stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails {
	if details == nil {
		return stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails{}
	}
	return stackmonitoringv1beta1.DiscoveryJobDiscoveryDetails{
		AgentId:      discoveryJobStringValue(details.AgentId),
		ResourceType: string(details.ResourceType),
		ResourceName: discoveryJobStringValue(details.ResourceName),
		Properties: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsProperties{
			PropertiesMap: discoveryJobStringMapFromPropertyDetails(details.Properties),
		},
		License:     string(details.License),
		Credentials: discoveryJobAPICredentials(details.Credentials),
		Tags: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsTags{
			PropertiesMap: discoveryJobStringMapFromPropertyDetails(details.Tags),
		},
	}
}

func discoveryJobStringMapFromPropertyDetails(details *stackmonitoringsdk.PropertyDetails) map[string]string {
	if details == nil {
		return nil
	}
	return discoveryJobStringMap(details.PropertiesMap)
}

func discoveryJobAPICredentials(credentials *stackmonitoringsdk.CredentialCollection) stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentials {
	if credentials == nil || len(credentials.Items) == 0 {
		return stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentials{}
	}
	items := make([]stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentialsItem, 0, len(credentials.Items))
	for _, item := range credentials.Items {
		items = append(items, stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentialsItem{
			CredentialName: discoveryJobStringValue(item.CredentialName),
			CredentialType: discoveryJobStringValue(item.CredentialType),
			Properties: stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentialsItemProperties{
				PropertiesMap: discoveryJobStringMapFromPropertyDetails(item.Properties),
			},
		})
	}
	return stackmonitoringv1beta1.DiscoveryJobDiscoveryDetailsCredentials{Items: items}
}

func discoveryJobStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func discoveryJobSharedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
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

func discoveryJobCloneSharedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
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

func discoveryJobDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
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

var _ interface{ GetOpcRequestID() string } = discoveryJobAmbiguousNotFoundError{}
