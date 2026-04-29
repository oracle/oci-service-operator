/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package topic

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	onssdk "github.com/oracle/oci-go-sdk/v65/ons"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

type topicOCIClient interface {
	CreateTopic(context.Context, onssdk.CreateTopicRequest) (onssdk.CreateTopicResponse, error)
	GetTopic(context.Context, onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error)
	ListTopics(context.Context, onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error)
	UpdateTopic(context.Context, onssdk.UpdateTopicRequest) (onssdk.UpdateTopicResponse, error)
	DeleteTopic(context.Context, onssdk.DeleteTopicRequest) (onssdk.DeleteTopicResponse, error)
}

type topicResourceBody struct {
	Id             *string                           `json:"id,omitempty"`
	TopicId        *string                           `json:"topicId,omitempty"`
	Name           *string                           `json:"name,omitempty"`
	CompartmentId  *string                           `json:"compartmentId,omitempty"`
	LifecycleState string                            `json:"lifecycleState,omitempty"`
	TimeCreated    *common.SDKTime                   `json:"timeCreated,omitempty"`
	ApiEndpoint    *string                           `json:"apiEndpoint,omitempty"`
	ShortTopicId   *string                           `json:"shortTopicId,omitempty"`
	Description    *string                           `json:"description,omitempty"`
	Etag           *string                           `json:"etag,omitempty"`
	FreeformTags   map[string]string                 `json:"freeformTags,omitempty"`
	DefinedTags    map[string]map[string]interface{} `json:"definedTags,omitempty"`
}

type topicOperationResponse struct {
	Topic        topicResourceBody `presentIn:"body"`
	OpcRequestId *string           `presentIn:"header" name:"opc-request-id"`
	Etag         *string           `presentIn:"header" name:"etag"`
}

type topicListBody struct {
	Items []topicResourceBody `json:"items"`
}

type topicListResponse struct {
	TopicCollection topicListBody `presentIn:"body"`
	OpcNextPage     *string       `presentIn:"header" name:"opc-next-page"`
	OpcRequestId    *string       `presentIn:"header" name:"opc-request-id"`
}

type topicAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e topicAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e topicAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerTopicRuntimeHooksMutator(func(manager *TopicServiceManager, hooks *TopicRuntimeHooks) {
		applyTopicRuntimeHooks(hooks)
		configuredHooks := *hooks
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate TopicServiceClient) TopicServiceClient {
			return newTopicServiceClientFromHooks(manager, configuredHooks, topicGeneratedDelegateInitError(delegate))
		})
	})
}

func applyTopicRuntimeHooks(hooks *TopicRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedTopicRuntimeSemantics()
	hooks.BuildCreateBody = buildTopicCreateBody
	hooks.BuildUpdateBody = buildTopicUpdateBody
	hooks.Create.Fields = topicCreateFields()
	hooks.Get.Fields = topicGetFields()
	hooks.List.Fields = topicListFields()
	hooks.Update.Fields = topicUpdateFields()
	hooks.Delete.Fields = topicDeleteFields()
	hooks.DeleteHooks.HandleError = handleTopicDeleteError
	if hooks.List.Call != nil {
		hooks.List.Call = listTopicsAllPages(hooks.List.Call)
	}
}

func newTopicServiceClientWithOCIClient(log loggerutil.OSOKLogger, client topicOCIClient) TopicServiceClient {
	manager := &TopicServiceManager{Log: log}
	hooks := newTopicRuntimeHooksWithOCIClient(client)
	applyTopicRuntimeHooks(&hooks)
	return newTopicServiceClientFromHooks(manager, hooks, nil)
}

func newTopicServiceClientFromHooks(
	manager *TopicServiceManager,
	hooks TopicRuntimeHooks,
	initError error,
) TopicServiceClient {
	return defaultTopicServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*onsv1beta1.Topic](
			newTopicRuntimeConfig(manager, hooks, initError),
		),
	}
}

func newTopicRuntimeConfig(
	manager *TopicServiceManager,
	hooks TopicRuntimeHooks,
	initError error,
) generatedruntime.Config[*onsv1beta1.Topic] {
	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}
	return generatedruntime.Config[*onsv1beta1.Topic]{
		Kind:            "Topic",
		SDKName:         "Topic",
		Log:             log,
		InitError:       initError,
		Semantics:       hooks.Semantics,
		Identity:        hooks.Identity,
		Read:            hooks.Read,
		TrackedRecreate: hooks.TrackedRecreate,
		StatusHooks:     hooks.StatusHooks,
		ParityHooks:     hooks.ParityHooks,
		Async:           hooks.Async,
		DeleteHooks:     hooks.DeleteHooks,
		BuildCreateBody: hooks.BuildCreateBody,
		BuildUpdateBody: hooks.BuildUpdateBody,
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &onssdk.CreateTopicRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Create.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Create.Call(ctx, *request.(*onssdk.CreateTopicRequest))
				if err != nil {
					return nil, err
				}
				return adaptTopicOperationResponse(response.NotificationTopic, response.OpcRequestId, response.Etag), nil
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &onssdk.GetTopicRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Get.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*onssdk.GetTopicRequest))
				if err != nil {
					return nil, err
				}
				return adaptTopicOperationResponse(response.NotificationTopic, response.OpcRequestId, response.Etag), nil
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &onssdk.ListTopicsRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.List.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*onssdk.ListTopicsRequest))
				if err != nil {
					return nil, err
				}
				return adaptTopicListResponse(response), nil
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &onssdk.UpdateTopicRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Update.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Update.Call(ctx, *request.(*onssdk.UpdateTopicRequest))
				if err != nil {
					return nil, err
				}
				return adaptTopicOperationResponse(response.NotificationTopic, response.OpcRequestId, response.Etag), nil
			},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &onssdk.DeleteTopicRequest{} },
			Fields:     append([]generatedruntime.RequestField(nil), hooks.Delete.Fields...),
			Call: func(ctx context.Context, request any) (any, error) {
				return hooks.Delete.Call(ctx, *request.(*onssdk.DeleteTopicRequest))
			},
		},
	}
}

func topicGeneratedDelegateInitError(delegate TopicServiceClient) error {
	if delegate == nil {
		return nil
	}

	// The generated service client records SDK construction failures before
	// WrapGeneratedClient runs. This package replaces the delegate to adapt ONS
	// response shapes, so probe the delegate validation path and carry any
	// early InitError into the replacement config.
	var resource *onsv1beta1.Topic
	_, err := delegate.Delete(context.Background(), resource)
	if err == nil || isTopicNilResourceProbeError(err) {
		return nil
	}
	return err
}

func isTopicNilResourceProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "resource is nil") || strings.Contains(message, "expected pointer resource")
}

func newTopicRuntimeHooksWithOCIClient(client topicOCIClient) TopicRuntimeHooks {
	return TopicRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*onsv1beta1.Topic]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*onsv1beta1.Topic]{},
		StatusHooks:     generatedruntime.StatusHooks[*onsv1beta1.Topic]{},
		ParityHooks:     generatedruntime.ParityHooks[*onsv1beta1.Topic]{},
		Async:           generatedruntime.AsyncHooks[*onsv1beta1.Topic]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*onsv1beta1.Topic]{},
		Create: runtimeOperationHooks[onssdk.CreateTopicRequest, onssdk.CreateTopicResponse]{
			Fields: topicCreateFields(),
			Call: func(ctx context.Context, request onssdk.CreateTopicRequest) (onssdk.CreateTopicResponse, error) {
				if client == nil {
					return onssdk.CreateTopicResponse{}, fmt.Errorf("topic OCI client is nil")
				}
				return client.CreateTopic(ctx, request)
			},
		},
		Get: runtimeOperationHooks[onssdk.GetTopicRequest, onssdk.GetTopicResponse]{
			Fields: topicGetFields(),
			Call: func(ctx context.Context, request onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
				if client == nil {
					return onssdk.GetTopicResponse{}, fmt.Errorf("topic OCI client is nil")
				}
				return client.GetTopic(ctx, request)
			},
		},
		List: runtimeOperationHooks[onssdk.ListTopicsRequest, onssdk.ListTopicsResponse]{
			Fields: topicListFields(),
			Call: func(ctx context.Context, request onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error) {
				if client == nil {
					return onssdk.ListTopicsResponse{}, fmt.Errorf("topic OCI client is nil")
				}
				return client.ListTopics(ctx, request)
			},
		},
		Update: runtimeOperationHooks[onssdk.UpdateTopicRequest, onssdk.UpdateTopicResponse]{
			Fields: topicUpdateFields(),
			Call: func(ctx context.Context, request onssdk.UpdateTopicRequest) (onssdk.UpdateTopicResponse, error) {
				if client == nil {
					return onssdk.UpdateTopicResponse{}, fmt.Errorf("topic OCI client is nil")
				}
				return client.UpdateTopic(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[onssdk.DeleteTopicRequest, onssdk.DeleteTopicResponse]{
			Fields: topicDeleteFields(),
			Call: func(ctx context.Context, request onssdk.DeleteTopicRequest) (onssdk.DeleteTopicResponse, error) {
				if client == nil {
					return onssdk.DeleteTopicResponse{}, fmt.Errorf("topic OCI client is nil")
				}
				return client.DeleteTopic(ctx, request)
			},
		},
		WrapGeneratedClient: []func(TopicServiceClient) TopicServiceClient{},
	}
}

func reviewedTopicRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "ons",
		FormalSlug:    "topic",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(onssdk.NotificationTopicLifecycleStateCreating)},
			ActiveStates:       []string{string(onssdk.NotificationTopicLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(onssdk.NotificationTopicLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"description", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Topic", Action: "CreateTopic"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Topic", Action: "UpdateTopic"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Topic", Action: "DeleteTopic"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Topic", Action: "GetTopic"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Topic", Action: "GetTopic"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Topic", Action: "GetTopic"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func topicCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateTopicDetails", RequestName: "CreateTopicDetails", Contribution: "body"},
	}
}

func topicGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TopicId", RequestName: "topicId", Contribution: "path", PreferResourceID: true},
	}
}

func topicListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "metadataName", "name"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func topicUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TopicId", RequestName: "topicId", Contribution: "path", PreferResourceID: true},
		{FieldName: "TopicAttributesDetails", RequestName: "TopicAttributesDetails", Contribution: "body"},
	}
}

func topicDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "TopicId", RequestName: "topicId", Contribution: "path", PreferResourceID: true},
	}
}

func buildTopicCreateBody(_ context.Context, resource *onsv1beta1.Topic, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("topic resource is nil")
	}
	if err := validateTopicSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := onssdk.CreateTopicDetails{
		Name:          common.String(strings.TrimSpace(resource.Spec.Name)),
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		body.Description = common.String(description)
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneTopicStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = topicDefinedTags(resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildTopicUpdateBody(_ context.Context, resource *onsv1beta1.Topic, _ string, currentResponse any) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("topic resource is nil")
	}
	current, ok := topicBodyFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current Topic response does not expose a Topic body")
	}

	body := onssdk.TopicAttributesDetails{
		Description: desiredTopicDescriptionForUpdate(resource.Spec.Description, current.Description),
	}
	updateNeeded := false
	if desiredDescription := strings.TrimSpace(resource.Spec.Description); desiredDescription != "" && !stringPtrEqual(current.Description, desiredDescription) {
		updateNeeded = true
	}

	if resource.Spec.FreeformTags != nil {
		desired := cloneTopicStringMap(resource.Spec.FreeformTags)
		body.FreeformTags = desired
		if !reflect.DeepEqual(current.FreeformTags, desired) {
			updateNeeded = true
		}
	}
	if resource.Spec.DefinedTags != nil {
		desired := topicDefinedTags(resource.Spec.DefinedTags)
		body.DefinedTags = desired
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			updateNeeded = true
		}
	}
	return body, updateNeeded, nil
}

func validateTopicSpec(spec onsv1beta1.TopicSpec) error {
	var missing []string
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if len(missing) > 0 {
		return fmt.Errorf("topic spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func desiredTopicDescriptionForUpdate(specDescription string, currentDescription *string) *string {
	if description := strings.TrimSpace(specDescription); description != "" {
		return common.String(description)
	}
	if currentDescription != nil {
		return common.String(*currentDescription)
	}
	return common.String("")
}

func listTopicsAllPages(
	call func(context.Context, onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error),
) func(context.Context, onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error) {
	return func(ctx context.Context, request onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error) {
		var combined onssdk.ListTopicsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return onssdk.ListTopicsResponse{}, err
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

func handleTopicDeleteError(resource *onsv1beta1.Topic, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return topicAmbiguousNotFoundError{
		message:      "topic delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func adaptTopicOperationResponse(topic onssdk.NotificationTopic, requestID *string, etag *string) topicOperationResponse {
	return topicOperationResponse{
		Topic:        topicBodyFromSDKTopic(topic),
		OpcRequestId: requestID,
		Etag:         etag,
	}
}

func adaptTopicListResponse(response onssdk.ListTopicsResponse) topicListResponse {
	items := make([]topicResourceBody, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, topicBodyFromSDKTopicSummary(item))
	}
	return topicListResponse{
		TopicCollection: topicListBody{Items: items},
		OpcNextPage:     response.OpcNextPage,
		OpcRequestId:    response.OpcRequestId,
	}
}

func topicBodyFromResponse(response any) (topicResourceBody, bool) {
	if body, ok := topicBodyFromAdaptedResponse(response); ok {
		return body, true
	}
	if body, ok := topicBodyFromSDKResponse(response); ok {
		return body, true
	}
	return topicBodyFromSDKResource(response)
}

func topicBodyFromAdaptedResponse(response any) (topicResourceBody, bool) {
	switch current := response.(type) {
	case topicOperationResponse:
		return current.Topic, true
	case *topicOperationResponse:
		if current == nil {
			return topicResourceBody{}, false
		}
		return current.Topic, true
	case topicResourceBody:
		return current, true
	case *topicResourceBody:
		if current == nil {
			return topicResourceBody{}, false
		}
		return *current, true
	default:
		return topicResourceBody{}, false
	}
}

func topicBodyFromSDKResponse(response any) (topicResourceBody, bool) {
	switch current := response.(type) {
	case onssdk.CreateTopicResponse:
		return topicBodyFromSDKTopic(current.NotificationTopic), true
	case *onssdk.CreateTopicResponse:
		if current == nil {
			return topicResourceBody{}, false
		}
		return topicBodyFromSDKTopic(current.NotificationTopic), true
	case onssdk.GetTopicResponse:
		return topicBodyFromSDKTopic(current.NotificationTopic), true
	case *onssdk.GetTopicResponse:
		if current == nil {
			return topicResourceBody{}, false
		}
		return topicBodyFromSDKTopic(current.NotificationTopic), true
	case onssdk.UpdateTopicResponse:
		return topicBodyFromSDKTopic(current.NotificationTopic), true
	case *onssdk.UpdateTopicResponse:
		if current == nil {
			return topicResourceBody{}, false
		}
		return topicBodyFromSDKTopic(current.NotificationTopic), true
	default:
		return topicResourceBody{}, false
	}
}

func topicBodyFromSDKResource(response any) (topicResourceBody, bool) {
	switch current := response.(type) {
	case onssdk.NotificationTopic:
		return topicBodyFromSDKTopic(current), true
	case *onssdk.NotificationTopic:
		if current == nil {
			return topicResourceBody{}, false
		}
		return topicBodyFromSDKTopic(*current), true
	case onssdk.NotificationTopicSummary:
		return topicBodyFromSDKTopicSummary(current), true
	case *onssdk.NotificationTopicSummary:
		if current == nil {
			return topicResourceBody{}, false
		}
		return topicBodyFromSDKTopicSummary(*current), true
	default:
		return topicResourceBody{}, false
	}
}

func topicBodyFromSDKTopic(topic onssdk.NotificationTopic) topicResourceBody {
	return topicResourceBody{
		Id:             topic.TopicId,
		TopicId:        topic.TopicId,
		Name:           topic.Name,
		CompartmentId:  topic.CompartmentId,
		LifecycleState: string(topic.LifecycleState),
		TimeCreated:    topic.TimeCreated,
		ApiEndpoint:    topic.ApiEndpoint,
		ShortTopicId:   topic.ShortTopicId,
		Description:    topic.Description,
		Etag:           topic.Etag,
		FreeformTags:   cloneTopicStringMap(topic.FreeformTags),
		DefinedTags:    cloneTopicDefinedTagMap(topic.DefinedTags),
	}
}

func topicBodyFromSDKTopicSummary(topic onssdk.NotificationTopicSummary) topicResourceBody {
	return topicResourceBody{
		Id:             topic.TopicId,
		TopicId:        topic.TopicId,
		Name:           topic.Name,
		CompartmentId:  topic.CompartmentId,
		LifecycleState: string(topic.LifecycleState),
		TimeCreated:    topic.TimeCreated,
		ApiEndpoint:    topic.ApiEndpoint,
		ShortTopicId:   topic.ShortTopicId,
		Description:    topic.Description,
		Etag:           topic.Etag,
		FreeformTags:   cloneTopicStringMap(topic.FreeformTags),
		DefinedTags:    cloneTopicDefinedTagMap(topic.DefinedTags),
	}
}

func cloneTopicStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneTopicDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
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

func topicDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func stringPtrEqual(got *string, want string) bool {
	return got != nil && *got == want
}
