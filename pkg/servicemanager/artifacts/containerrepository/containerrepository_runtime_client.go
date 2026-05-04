/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerrepository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

type containerRepositoryOCIClient interface {
	CreateContainerRepository(context.Context, artifactssdk.CreateContainerRepositoryRequest) (artifactssdk.CreateContainerRepositoryResponse, error)
	GetContainerRepository(context.Context, artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error)
	ListContainerRepositories(context.Context, artifactssdk.ListContainerRepositoriesRequest) (artifactssdk.ListContainerRepositoriesResponse, error)
	UpdateContainerRepository(context.Context, artifactssdk.UpdateContainerRepositoryRequest) (artifactssdk.UpdateContainerRepositoryResponse, error)
	DeleteContainerRepository(context.Context, artifactssdk.DeleteContainerRepositoryRequest) (artifactssdk.DeleteContainerRepositoryResponse, error)
}

type ambiguousContainerRepositoryNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousContainerRepositoryNotFoundError) Error() string {
	return e.message
}

func (e ambiguousContainerRepositoryNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerContainerRepositoryRuntimeHooksMutator(func(_ *ContainerRepositoryServiceManager, hooks *ContainerRepositoryRuntimeHooks) {
		applyContainerRepositoryRuntimeHooks(hooks)
	})
}

func applyContainerRepositoryRuntimeHooks(hooks *ContainerRepositoryRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedContainerRepositoryRuntimeSemantics()
	hooks.BuildCreateBody = buildContainerRepositoryCreateBody
	hooks.BuildUpdateBody = buildContainerRepositoryUpdateBody
	hooks.List.Fields = containerRepositoryListFields()
	wrapContainerRepositoryReadAndDeleteCalls(hooks)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedContainerRepositoryIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateContainerRepositoryCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleContainerRepositoryDeleteError
}

func wrapContainerRepositoryReadAndDeleteCalls(hooks *ContainerRepositoryRuntimeHooks) {
	getCall := hooks.Get.Call
	if getCall != nil {
		hooks.Get.Call = func(ctx context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativeContainerRepositoryNotFoundError(err, "read")
		}
	}

	listCall := hooks.List.Call
	if listCall != nil {
		hooks.List.Call = func(ctx context.Context, request artifactssdk.ListContainerRepositoriesRequest) (artifactssdk.ListContainerRepositoriesResponse, error) {
			return listContainerRepositoriesAllPages(ctx, listCall, request)
		}
	}

	deleteCall := hooks.Delete.Call
	if deleteCall != nil {
		hooks.Delete.Call = func(ctx context.Context, request artifactssdk.DeleteContainerRepositoryRequest) (artifactssdk.DeleteContainerRepositoryResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeContainerRepositoryNotFoundError(err, "delete")
		}
	}
}

func newContainerRepositoryServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client containerRepositoryOCIClient,
) ContainerRepositoryServiceClient {
	hooks := newContainerRepositoryRuntimeHooksWithOCIClient(client)
	applyContainerRepositoryRuntimeHooks(&hooks)
	manager := &ContainerRepositoryServiceManager{Log: log}
	return defaultContainerRepositoryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*artifactsv1beta1.ContainerRepository](
			buildContainerRepositoryGeneratedRuntimeConfig(manager, hooks),
		),
	}
}

func newContainerRepositoryRuntimeHooksWithOCIClient(client containerRepositoryOCIClient) ContainerRepositoryRuntimeHooks {
	return ContainerRepositoryRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*artifactsv1beta1.ContainerRepository]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*artifactsv1beta1.ContainerRepository]{},
		StatusHooks:     generatedruntime.StatusHooks[*artifactsv1beta1.ContainerRepository]{},
		ParityHooks:     generatedruntime.ParityHooks[*artifactsv1beta1.ContainerRepository]{},
		Async:           generatedruntime.AsyncHooks[*artifactsv1beta1.ContainerRepository]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*artifactsv1beta1.ContainerRepository]{},
		Create: runtimeOperationHooks[artifactssdk.CreateContainerRepositoryRequest, artifactssdk.CreateContainerRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateContainerRepositoryDetails", RequestName: "CreateContainerRepositoryDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request artifactssdk.CreateContainerRepositoryRequest) (artifactssdk.CreateContainerRepositoryResponse, error) {
				return client.CreateContainerRepository(ctx, request)
			},
		},
		Get: runtimeOperationHooks[artifactssdk.GetContainerRepositoryRequest, artifactssdk.GetContainerRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "RepositoryId", RequestName: "repositoryId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request artifactssdk.GetContainerRepositoryRequest) (artifactssdk.GetContainerRepositoryResponse, error) {
				return client.GetContainerRepository(ctx, request)
			},
		},
		List: runtimeOperationHooks[artifactssdk.ListContainerRepositoriesRequest, artifactssdk.ListContainerRepositoriesResponse]{
			Fields: containerRepositoryListFields(),
			Call: func(ctx context.Context, request artifactssdk.ListContainerRepositoriesRequest) (artifactssdk.ListContainerRepositoriesResponse, error) {
				return client.ListContainerRepositories(ctx, request)
			},
		},
		Update: runtimeOperationHooks[artifactssdk.UpdateContainerRepositoryRequest, artifactssdk.UpdateContainerRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "RepositoryId", RequestName: "repositoryId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateContainerRepositoryDetails", RequestName: "UpdateContainerRepositoryDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request artifactssdk.UpdateContainerRepositoryRequest) (artifactssdk.UpdateContainerRepositoryResponse, error) {
				return client.UpdateContainerRepository(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[artifactssdk.DeleteContainerRepositoryRequest, artifactssdk.DeleteContainerRepositoryResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "RepositoryId", RequestName: "repositoryId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request artifactssdk.DeleteContainerRepositoryRequest) (artifactssdk.DeleteContainerRepositoryResponse, error) {
				return client.DeleteContainerRepository(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ContainerRepositoryServiceClient) ContainerRepositoryServiceClient{},
	}
}

func reviewedContainerRepositoryRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "artifacts",
		FormalSlug:    "containerrepository",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(artifactssdk.ContainerRepositoryLifecycleStateAvailable)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(artifactssdk.ContainerRepositoryLifecycleStateDeleting)},
			TerminalStates: []string{string(artifactssdk.ContainerRepositoryLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"isImmutable", "isPublic", "readme", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "displayName"},
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

func containerRepositoryListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "RepositoryId", RequestName: "repositoryId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildContainerRepositoryCreateBody(
	_ context.Context,
	resource *artifactsv1beta1.ContainerRepository,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ContainerRepository resource is nil")
	}
	if err := validateContainerRepositorySpec(resource.Spec); err != nil {
		return nil, err
	}

	body := artifactssdk.CreateContainerRepositoryDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		IsImmutable:   common.Bool(resource.Spec.IsImmutable),
		IsPublic:      common.Bool(resource.Spec.IsPublic),
	}
	if readmeSpecified(resource.Spec.Readme) {
		readme, err := sdkContainerRepositoryReadme(resource.Spec.Readme)
		if err != nil {
			return nil, err
		}
		body.Readme = readme
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneContainerRepositoryStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildContainerRepositoryUpdateBody(
	_ context.Context,
	resource *artifactsv1beta1.ContainerRepository,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return artifactssdk.UpdateContainerRepositoryDetails{}, false, fmt.Errorf("ContainerRepository resource is nil")
	}
	if err := validateContainerRepositorySpec(resource.Spec); err != nil {
		return artifactssdk.UpdateContainerRepositoryDetails{}, false, err
	}

	current, ok := containerRepositoryFromResponse(currentResponse)
	if !ok {
		return artifactssdk.UpdateContainerRepositoryDetails{}, false, fmt.Errorf("current ContainerRepository response does not expose a ContainerRepository body")
	}
	if err := validateContainerRepositoryCreateOnlyDrift(resource.Spec, current); err != nil {
		return artifactssdk.UpdateContainerRepositoryDetails{}, false, err
	}

	updateDetails := artifactssdk.UpdateContainerRepositoryDetails{}
	updateNeeded := false

	updateNeeded = applyContainerRepositoryBooleanUpdates(&updateDetails, resource.Spec, current) || updateNeeded
	readmeUpdated, err := applyContainerRepositoryReadmeUpdate(&updateDetails, resource.Spec, current)
	if err != nil {
		return artifactssdk.UpdateContainerRepositoryDetails{}, false, err
	}
	updateNeeded = readmeUpdated || updateNeeded
	updateNeeded = applyContainerRepositoryTagUpdates(&updateDetails, resource.Spec, current) || updateNeeded

	if !updateNeeded {
		return artifactssdk.UpdateContainerRepositoryDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func applyContainerRepositoryBooleanUpdates(
	updateDetails *artifactssdk.UpdateContainerRepositoryDetails,
	spec artifactsv1beta1.ContainerRepositorySpec,
	current artifactssdk.ContainerRepository,
) bool {
	updateNeeded := false
	if !boolPtrEqual(current.IsImmutable, spec.IsImmutable) {
		updateDetails.IsImmutable = common.Bool(spec.IsImmutable)
		updateNeeded = true
	}
	if !boolPtrEqual(current.IsPublic, spec.IsPublic) {
		updateDetails.IsPublic = common.Bool(spec.IsPublic)
		updateNeeded = true
	}
	return updateNeeded
}

func applyContainerRepositoryReadmeUpdate(
	updateDetails *artifactssdk.UpdateContainerRepositoryDetails,
	spec artifactsv1beta1.ContainerRepositorySpec,
	current artifactssdk.ContainerRepository,
) (bool, error) {
	if !readmeSpecified(spec.Readme) || containerRepositoryReadmeEqual(current.Readme, spec.Readme) {
		return false, nil
	}
	readme, err := sdkContainerRepositoryReadme(spec.Readme)
	if err != nil {
		return false, err
	}
	updateDetails.Readme = readme
	return true, nil
}

func applyContainerRepositoryTagUpdates(
	updateDetails *artifactssdk.UpdateContainerRepositoryDetails,
	spec artifactsv1beta1.ContainerRepositorySpec,
	current artifactssdk.ContainerRepository,
) bool {
	updateNeeded := false
	desiredFreeformTags := desiredContainerRepositoryFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredContainerRepositoryDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}
	return updateNeeded
}

func validateContainerRepositorySpec(spec artifactsv1beta1.ContainerRepositorySpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if readmePartiallySpecified(spec.Readme) && !readmeSpecified(spec.Readme) {
		missing = append(missing, "readme.content", "readme.format")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("ContainerRepository spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func validateContainerRepositoryCreateOnlyDriftForResponse(
	resource *artifactsv1beta1.ContainerRepository,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("ContainerRepository resource is nil")
	}
	current, ok := containerRepositoryFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current ContainerRepository response does not expose a ContainerRepository body")
	}
	return validateContainerRepositoryCreateOnlyDrift(resource.Spec, current)
}

func validateContainerRepositoryCreateOnlyDrift(
	spec artifactsv1beta1.ContainerRepositorySpec,
	current artifactssdk.ContainerRepository,
) error {
	var drift []string
	if !stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !stringPtrEqual(current.DisplayName, spec.DisplayName) {
		drift = append(drift, "displayName")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("ContainerRepository create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func listContainerRepositoriesAllPages(
	ctx context.Context,
	list func(context.Context, artifactssdk.ListContainerRepositoriesRequest) (artifactssdk.ListContainerRepositoriesResponse, error),
	request artifactssdk.ListContainerRepositoriesRequest,
) (artifactssdk.ListContainerRepositoriesResponse, error) {
	var combined artifactssdk.ListContainerRepositoriesResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return artifactssdk.ListContainerRepositoriesResponse{}, conservativeContainerRepositoryNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.LayerCount = response.LayerCount
		combined.LayersSizeInBytes = response.LayersSizeInBytes
		combined.ImageCount = response.ImageCount
		combined.RemainingItemsCount = response.RemainingItemsCount
		combined.RepositoryCount = response.RepositoryCount
		for _, item := range response.Items {
			if item.LifecycleState == artifactssdk.ContainerRepositoryLifecycleStateDeleted {
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

func handleContainerRepositoryDeleteError(resource *artifactsv1beta1.ContainerRepository, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeContainerRepositoryNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("ContainerRepository %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousContainerRepositoryNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousContainerRepositoryNotFoundError{message: message}
}

func clearTrackedContainerRepositoryIdentity(resource *artifactsv1beta1.ContainerRepository) {
	if resource == nil {
		return
	}
	resource.Status = artifactsv1beta1.ContainerRepositoryStatus{}
}

func containerRepositoryFromResponse(response any) (artifactssdk.ContainerRepository, bool) {
	if current, ok := containerRepositoryFromWriteResponse(response); ok {
		return current, true
	}
	if current, ok := containerRepositoryFromReadResponse(response); ok {
		return current, true
	}
	return containerRepositoryFromListItem(response)
}

func containerRepositoryFromWriteResponse(response any) (artifactssdk.ContainerRepository, bool) {
	switch current := response.(type) {
	case artifactssdk.CreateContainerRepositoryResponse:
		return current.ContainerRepository, true
	case *artifactssdk.CreateContainerRepositoryResponse:
		if current == nil {
			return artifactssdk.ContainerRepository{}, false
		}
		return current.ContainerRepository, true
	case artifactssdk.UpdateContainerRepositoryResponse:
		return current.ContainerRepository, true
	case *artifactssdk.UpdateContainerRepositoryResponse:
		if current == nil {
			return artifactssdk.ContainerRepository{}, false
		}
		return current.ContainerRepository, true
	default:
		return artifactssdk.ContainerRepository{}, false
	}
}

func containerRepositoryFromReadResponse(response any) (artifactssdk.ContainerRepository, bool) {
	switch current := response.(type) {
	case artifactssdk.GetContainerRepositoryResponse:
		return current.ContainerRepository, true
	case *artifactssdk.GetContainerRepositoryResponse:
		if current == nil {
			return artifactssdk.ContainerRepository{}, false
		}
		return current.ContainerRepository, true
	case artifactssdk.ContainerRepository:
		return current, true
	case *artifactssdk.ContainerRepository:
		if current == nil {
			return artifactssdk.ContainerRepository{}, false
		}
		return *current, true
	default:
		return artifactssdk.ContainerRepository{}, false
	}
}

func containerRepositoryFromListItem(response any) (artifactssdk.ContainerRepository, bool) {
	switch current := response.(type) {
	case artifactssdk.ContainerRepositorySummary:
		return containerRepositoryFromSummary(current), true
	case *artifactssdk.ContainerRepositorySummary:
		if current == nil {
			return artifactssdk.ContainerRepository{}, false
		}
		return containerRepositoryFromSummary(*current), true
	default:
		return artifactssdk.ContainerRepository{}, false
	}
}

func containerRepositoryFromSummary(summary artifactssdk.ContainerRepositorySummary) artifactssdk.ContainerRepository {
	return artifactssdk.ContainerRepository{
		CompartmentId:     summary.CompartmentId,
		DisplayName:       summary.DisplayName,
		Id:                summary.Id,
		ImageCount:        summary.ImageCount,
		IsPublic:          summary.IsPublic,
		LayerCount:        summary.LayerCount,
		LayersSizeInBytes: summary.LayersSizeInBytes,
		LifecycleState:    summary.LifecycleState,
		TimeCreated:       summary.TimeCreated,
		BillableSizeInGBs: summary.BillableSizeInGBs,
		Namespace:         summary.Namespace,
		FreeformTags:      cloneContainerRepositoryStringMap(summary.FreeformTags),
		DefinedTags:       cloneContainerRepositoryDefinedTags(summary.DefinedTags),
		SystemTags:        cloneContainerRepositoryDefinedTags(summary.SystemTags),
	}
}

func sdkContainerRepositoryReadme(
	readme artifactsv1beta1.ContainerRepositoryReadme,
) (*artifactssdk.ContainerRepositoryReadme, error) {
	format, err := containerRepositoryReadmeFormat(readme.Format)
	if err != nil {
		return nil, err
	}
	return &artifactssdk.ContainerRepositoryReadme{
		Content: common.String(readme.Content),
		Format:  format,
	}, nil
}

func readmeSpecified(readme artifactsv1beta1.ContainerRepositoryReadme) bool {
	return strings.TrimSpace(readme.Content) != "" && strings.TrimSpace(readme.Format) != ""
}

func readmePartiallySpecified(readme artifactsv1beta1.ContainerRepositoryReadme) bool {
	return strings.TrimSpace(readme.Content) != "" || strings.TrimSpace(readme.Format) != ""
}

func containerRepositoryReadmeEqual(current *artifactssdk.ContainerRepositoryReadme, desired artifactsv1beta1.ContainerRepositoryReadme) bool {
	if !readmeSpecified(desired) {
		return true
	}
	if current == nil {
		return false
	}
	format, err := containerRepositoryReadmeFormat(desired.Format)
	if err != nil {
		return false
	}
	return stringValue(current.Content) == desired.Content && current.Format == format
}

func containerRepositoryReadmeFormat(value string) (artifactssdk.ContainerRepositoryReadmeFormatEnum, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "text/markdown", "text_markdown", "text-markdown", "markdown":
		return artifactssdk.ContainerRepositoryReadmeFormatMarkdown, nil
	case "text/plain", "text_plain", "text-plain", "plain":
		return artifactssdk.ContainerRepositoryReadmeFormatPlain, nil
	default:
		return "", fmt.Errorf("unsupported ContainerRepository readme format %q", value)
	}
}

func desiredContainerRepositoryFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneContainerRepositoryStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredContainerRepositoryDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func cloneContainerRepositoryStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneContainerRepositoryDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
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

func boolPtrEqual(current *bool, desired bool) bool {
	return current != nil && *current == desired
}

func stringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(stringValue(current)) == strings.TrimSpace(desired)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
