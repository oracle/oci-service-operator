/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicecatalog

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	servicecatalogsdk "github.com/oracle/oci-go-sdk/v65/servicecatalog"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type serviceCatalogOCIClient interface {
	CreateServiceCatalog(context.Context, servicecatalogsdk.CreateServiceCatalogRequest) (servicecatalogsdk.CreateServiceCatalogResponse, error)
	GetServiceCatalog(context.Context, servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error)
	ListServiceCatalogs(context.Context, servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error)
	UpdateServiceCatalog(context.Context, servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error)
	DeleteServiceCatalog(context.Context, servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error)
}

type ambiguousServiceCatalogNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousServiceCatalogNotFoundError) Error() string {
	return e.message
}

func (e ambiguousServiceCatalogNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerServiceCatalogRuntimeHooksMutator(func(_ *ServiceCatalogServiceManager, hooks *ServiceCatalogRuntimeHooks) {
		applyServiceCatalogRuntimeHooks(hooks)
	})
}

func applyServiceCatalogRuntimeHooks(hooks *ServiceCatalogRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedServiceCatalogRuntimeSemantics()
	hooks.BuildCreateBody = buildServiceCatalogCreateBody
	hooks.BuildUpdateBody = buildServiceCatalogUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardServiceCatalogExistingBeforeCreate
	hooks.List.Fields = serviceCatalogListFields()
	wrapServiceCatalogWriteCalls(hooks)
	wrapServiceCatalogReadAndDeleteCalls(hooks)
	wrapServiceCatalogDeleteConfirmation(hooks)
	installServiceCatalogProjectedReadOperations(hooks)
	hooks.StatusHooks.ProjectStatus = projectServiceCatalogStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateServiceCatalogCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleServiceCatalogDeleteError
}

func wrapServiceCatalogWriteCalls(hooks *ServiceCatalogRuntimeHooks) {
	createCall := hooks.Create.Call
	if createCall != nil {
		hooks.Create.Call = func(ctx context.Context, request servicecatalogsdk.CreateServiceCatalogRequest) (servicecatalogsdk.CreateServiceCatalogResponse, error) {
			response, err := createCall(ctx, request)
			if err == nil {
				response.Status = ""
			}
			return response, err
		}
	}

	updateCall := hooks.Update.Call
	if updateCall != nil {
		hooks.Update.Call = func(ctx context.Context, request servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error) {
			response, err := updateCall(ctx, request)
			if err == nil {
				response.Status = ""
			}
			return response, err
		}
	}
}

func wrapServiceCatalogReadAndDeleteCalls(hooks *ServiceCatalogRuntimeHooks) {
	getCall := hooks.Get.Call
	if getCall != nil {
		hooks.Get.Call = func(ctx context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
			response, err := getCall(ctx, request)
			return response, conservativeServiceCatalogNotFoundError(err, "read")
		}
	}

	listCall := hooks.List.Call
	if listCall != nil {
		hooks.List.Call = func(ctx context.Context, request servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error) {
			return listServiceCatalogsAllPages(ctx, listCall, request)
		}
	}

	deleteCall := hooks.Delete.Call
	if deleteCall != nil {
		hooks.Delete.Call = func(ctx context.Context, request servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
			response, err := deleteCall(ctx, request)
			return response, conservativeServiceCatalogNotFoundError(err, "delete")
		}
	}
}

func installServiceCatalogProjectedReadOperations(hooks *ServiceCatalogRuntimeHooks) {
	if hooks.Get.Call != nil {
		getFields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
		hooks.Read.Get = &generatedruntime.Operation{
			NewRequest: func() any { return &servicecatalogsdk.GetServiceCatalogRequest{} },
			Fields:     getFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.Get.Call(ctx, *request.(*servicecatalogsdk.GetServiceCatalogRequest))
				if err != nil {
					return nil, err
				}
				return serviceCatalogProjectedResponseFromSDK(response.ServiceCatalog, response.OpcRequestId), nil
			},
		}
	}

	if hooks.List.Call != nil {
		listFields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
		hooks.Read.List = &generatedruntime.Operation{
			NewRequest: func() any { return &servicecatalogsdk.ListServiceCatalogsRequest{} },
			Fields:     listFields,
			Call: func(ctx context.Context, request any) (any, error) {
				response, err := hooks.List.Call(ctx, *request.(*servicecatalogsdk.ListServiceCatalogsRequest))
				if err != nil {
					return nil, err
				}
				return serviceCatalogProjectedListResponseFromSDK(response), nil
			},
		}
	}
}

func newServiceCatalogServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client serviceCatalogOCIClient,
) ServiceCatalogServiceClient {
	hooks := newServiceCatalogRuntimeHooksWithOCIClient(client)
	applyServiceCatalogRuntimeHooks(&hooks)
	delegate := defaultServiceCatalogServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*servicecatalogv1beta1.ServiceCatalog](
			buildServiceCatalogGeneratedRuntimeConfig(&ServiceCatalogServiceManager{Log: log}, hooks),
		),
	}
	return wrapServiceCatalogGeneratedClient(hooks, delegate)
}

func newServiceCatalogRuntimeHooksWithOCIClient(client serviceCatalogOCIClient) ServiceCatalogRuntimeHooks {
	return ServiceCatalogRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*servicecatalogv1beta1.ServiceCatalog]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*servicecatalogv1beta1.ServiceCatalog]{},
		StatusHooks:     generatedruntime.StatusHooks[*servicecatalogv1beta1.ServiceCatalog]{},
		ParityHooks:     generatedruntime.ParityHooks[*servicecatalogv1beta1.ServiceCatalog]{},
		Async:           generatedruntime.AsyncHooks[*servicecatalogv1beta1.ServiceCatalog]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*servicecatalogv1beta1.ServiceCatalog]{},
		Create: runtimeOperationHooks[servicecatalogsdk.CreateServiceCatalogRequest, servicecatalogsdk.CreateServiceCatalogResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateServiceCatalogDetails", RequestName: "CreateServiceCatalogDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request servicecatalogsdk.CreateServiceCatalogRequest) (servicecatalogsdk.CreateServiceCatalogResponse, error) {
				return client.CreateServiceCatalog(ctx, request)
			},
		},
		Get: runtimeOperationHooks[servicecatalogsdk.GetServiceCatalogRequest, servicecatalogsdk.GetServiceCatalogResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ServiceCatalogId", RequestName: "serviceCatalogId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error) {
				return client.GetServiceCatalog(ctx, request)
			},
		},
		List: runtimeOperationHooks[servicecatalogsdk.ListServiceCatalogsRequest, servicecatalogsdk.ListServiceCatalogsResponse]{
			Fields: serviceCatalogListFields(),
			Call: func(ctx context.Context, request servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error) {
				return client.ListServiceCatalogs(ctx, request)
			},
		},
		Update: runtimeOperationHooks[servicecatalogsdk.UpdateServiceCatalogRequest, servicecatalogsdk.UpdateServiceCatalogResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ServiceCatalogId", RequestName: "serviceCatalogId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateServiceCatalogDetails", RequestName: "UpdateServiceCatalogDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request servicecatalogsdk.UpdateServiceCatalogRequest) (servicecatalogsdk.UpdateServiceCatalogResponse, error) {
				return client.UpdateServiceCatalog(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[servicecatalogsdk.DeleteServiceCatalogRequest, servicecatalogsdk.DeleteServiceCatalogResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ServiceCatalogId", RequestName: "serviceCatalogId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request servicecatalogsdk.DeleteServiceCatalogRequest) (servicecatalogsdk.DeleteServiceCatalogResponse, error) {
				return client.DeleteServiceCatalog(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ServiceCatalogServiceClient) ServiceCatalogServiceClient{},
	}
}

type serviceCatalogStatusProjection struct {
	Id             string                     `json:"id,omitempty"`
	CompartmentId  string                     `json:"compartmentId,omitempty"`
	DisplayName    string                     `json:"displayName,omitempty"`
	LifecycleState string                     `json:"lifecycleState,omitempty"`
	TimeCreated    string                     `json:"timeCreated,omitempty"`
	Status         string                     `json:"sdkStatus,omitempty"`
	TimeUpdated    string                     `json:"timeUpdated,omitempty"`
	DefinedTags    map[string]shared.MapValue `json:"definedTags,omitempty"`
	FreeformTags   map[string]string          `json:"freeformTags,omitempty"`
	SystemTags     map[string]shared.MapValue `json:"systemTags,omitempty"`
}

type serviceCatalogProjectedResponse struct {
	ServiceCatalog serviceCatalogStatusProjection `presentIn:"body"`
	OpcRequestId   *string                        `presentIn:"header" name:"opc-request-id"`
}

type serviceCatalogProjectedCollection struct {
	Items []serviceCatalogStatusProjection `json:"items,omitempty"`
}

type serviceCatalogProjectedListResponse struct {
	ServiceCatalogCollection serviceCatalogProjectedCollection `presentIn:"body"`
	OpcRequestId             *string                           `presentIn:"header" name:"opc-request-id"`
	OpcNextPage              *string                           `presentIn:"header" name:"opc-next-page"`
}

func serviceCatalogProjectedResponseFromSDK(
	current servicecatalogsdk.ServiceCatalog,
	opcRequestID *string,
) serviceCatalogProjectedResponse {
	return serviceCatalogProjectedResponse{
		ServiceCatalog: serviceCatalogStatusProjectionFromSDK(current),
		OpcRequestId:   opcRequestID,
	}
}

func serviceCatalogProjectedListResponseFromSDK(
	response servicecatalogsdk.ListServiceCatalogsResponse,
) serviceCatalogProjectedListResponse {
	projected := serviceCatalogProjectedListResponse{
		OpcRequestId: response.OpcRequestId,
		OpcNextPage:  response.OpcNextPage,
	}
	for _, item := range response.Items {
		projected.ServiceCatalogCollection.Items = append(
			projected.ServiceCatalogCollection.Items,
			serviceCatalogStatusProjectionFromSDK(serviceCatalogFromSummary(item)),
		)
	}
	return projected
}

func serviceCatalogStatusProjectionFromSDK(current servicecatalogsdk.ServiceCatalog) serviceCatalogStatusProjection {
	return serviceCatalogStatusProjection{
		Id:             stringValue(current.Id),
		CompartmentId:  stringValue(current.CompartmentId),
		DisplayName:    stringValue(current.DisplayName),
		LifecycleState: string(current.LifecycleState),
		TimeCreated:    sdkTimeString(current.TimeCreated),
		Status:         string(current.Status),
		TimeUpdated:    sdkTimeString(current.TimeUpdated),
		DefinedTags:    serviceCatalogDefinedTagsStatusMap(current.DefinedTags),
		FreeformTags:   cloneServiceCatalogStringMap(current.FreeformTags),
		SystemTags:     serviceCatalogDefinedTagsStatusMap(current.SystemTags),
	}
}

func projectServiceCatalogStatus(resource *servicecatalogv1beta1.ServiceCatalog, response any) error {
	if resource == nil {
		return fmt.Errorf("ServiceCatalog resource is nil")
	}
	projected, ok := serviceCatalogProjectionFromResponse(response)
	if !ok {
		return nil
	}

	resource.Status = servicecatalogv1beta1.ServiceCatalogStatus{
		OsokStatus:     resource.Status.OsokStatus,
		Id:             projected.Id,
		CompartmentId:  projected.CompartmentId,
		DisplayName:    projected.DisplayName,
		LifecycleState: projected.LifecycleState,
		TimeCreated:    projected.TimeCreated,
		Status:         projected.Status,
		TimeUpdated:    projected.TimeUpdated,
		DefinedTags:    cloneServiceCatalogStatusDefinedTags(projected.DefinedTags),
		FreeformTags:   cloneServiceCatalogStringMap(projected.FreeformTags),
		SystemTags:     cloneServiceCatalogStatusDefinedTags(projected.SystemTags),
	}
	return nil
}

func serviceCatalogProjectionFromResponse(response any) (serviceCatalogStatusProjection, bool) {
	switch current := response.(type) {
	case serviceCatalogProjectedResponse:
		return current.ServiceCatalog, true
	case *serviceCatalogProjectedResponse:
		if current == nil {
			return serviceCatalogStatusProjection{}, false
		}
		return current.ServiceCatalog, true
	case serviceCatalogStatusProjection:
		return current, true
	case *serviceCatalogStatusProjection:
		if current == nil {
			return serviceCatalogStatusProjection{}, false
		}
		return *current, true
	default:
		if catalog, ok := serviceCatalogFromResponse(response); ok {
			return serviceCatalogStatusProjectionFromSDK(catalog), true
		}
		return serviceCatalogStatusProjection{}, false
	}
}

func reviewedServiceCatalogRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "servicecatalog",
		FormalSlug:    "servicecatalog",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(servicecatalogsdk.ServiceCatalogLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "best-effort",
			TerminalStates: []string{string(servicecatalogsdk.ServiceCatalogLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "status", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId"},
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

func serviceCatalogListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "ServiceCatalogId", RequestName: "serviceCatalogId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildServiceCatalogCreateBody(
	_ context.Context,
	resource *servicecatalogv1beta1.ServiceCatalog,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ServiceCatalog resource is nil")
	}
	if err := validateServiceCatalogSpec(resource.Spec); err != nil {
		return nil, err
	}

	body := servicecatalogsdk.CreateServiceCatalogDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
	}
	if status, ok, err := serviceCatalogSpecStatus(resource.Spec.Status); err != nil {
		return nil, err
	} else if ok {
		body.Status = status
	}
	if resource.Spec.FreeformTags != nil {
		body.FreeformTags = cloneServiceCatalogStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	return body, nil
}

func buildServiceCatalogUpdateBody(
	_ context.Context,
	resource *servicecatalogv1beta1.ServiceCatalog,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return servicecatalogsdk.UpdateServiceCatalogDetails{}, false, fmt.Errorf("ServiceCatalog resource is nil")
	}
	if err := validateServiceCatalogSpec(resource.Spec); err != nil {
		return servicecatalogsdk.UpdateServiceCatalogDetails{}, false, err
	}

	current, ok := serviceCatalogFromResponse(currentResponse)
	if !ok {
		return servicecatalogsdk.UpdateServiceCatalogDetails{}, false, fmt.Errorf("current ServiceCatalog response does not expose a ServiceCatalog body")
	}
	if err := validateServiceCatalogCreateOnlyDrift(resource.Spec, current); err != nil {
		return servicecatalogsdk.UpdateServiceCatalogDetails{}, false, err
	}

	updateDetails := servicecatalogsdk.UpdateServiceCatalogDetails{
		DisplayName: common.String(strings.TrimSpace(resource.Spec.DisplayName)),
		Status:      currentServiceCatalogStatusForUpdate(current.Status),
	}
	updateNeeded := !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName)
	statusUpdated, err := applyServiceCatalogStatusUpdate(&updateDetails, resource.Spec, current)
	if err != nil {
		return servicecatalogsdk.UpdateServiceCatalogDetails{}, false, err
	}
	updateNeeded = statusUpdated || updateNeeded
	updateNeeded = applyServiceCatalogTagUpdates(&updateDetails, resource.Spec, current) || updateNeeded

	if !updateNeeded {
		return servicecatalogsdk.UpdateServiceCatalogDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func applyServiceCatalogStatusUpdate(
	updateDetails *servicecatalogsdk.UpdateServiceCatalogDetails,
	spec servicecatalogv1beta1.ServiceCatalogSpec,
	current servicecatalogsdk.ServiceCatalog,
) (bool, error) {
	desired, ok, err := serviceCatalogSpecStatus(spec.Status)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	updateDetails.Status = desired
	return current.Status != desired, nil
}

func applyServiceCatalogTagUpdates(
	updateDetails *servicecatalogsdk.UpdateServiceCatalogDetails,
	spec servicecatalogv1beta1.ServiceCatalogSpec,
	current servicecatalogsdk.ServiceCatalog,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil {
		desiredFreeformTags := cloneServiceCatalogStringMap(spec.FreeformTags)
		if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
			updateDetails.FreeformTags = desiredFreeformTags
			updateNeeded = true
		}
	}
	if spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			updateDetails.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	return updateNeeded
}

func validateServiceCatalogSpec(spec servicecatalogv1beta1.ServiceCatalogSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if len(missing) != 0 {
		return fmt.Errorf("ServiceCatalog spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	if _, _, err := serviceCatalogSpecStatus(spec.Status); err != nil {
		return err
	}
	return nil
}

func guardServiceCatalogExistingBeforeCreate(_ context.Context, resource *servicecatalogv1beta1.ServiceCatalog) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ServiceCatalog resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateServiceCatalogCreateOnlyDriftForResponse(
	resource *servicecatalogv1beta1.ServiceCatalog,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("ServiceCatalog resource is nil")
	}
	current, ok := serviceCatalogFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current ServiceCatalog response does not expose a ServiceCatalog body")
	}
	return validateServiceCatalogCreateOnlyDrift(resource.Spec, current)
}

func validateServiceCatalogCreateOnlyDrift(
	spec servicecatalogv1beta1.ServiceCatalogSpec,
	current servicecatalogsdk.ServiceCatalog,
) error {
	if stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		return nil
	}
	return fmt.Errorf("ServiceCatalog create-only field drift is not supported: compartmentId")
}

func listServiceCatalogsAllPages(
	ctx context.Context,
	list func(context.Context, servicecatalogsdk.ListServiceCatalogsRequest) (servicecatalogsdk.ListServiceCatalogsResponse, error),
	request servicecatalogsdk.ListServiceCatalogsRequest,
) (servicecatalogsdk.ListServiceCatalogsResponse, error) {
	var combined servicecatalogsdk.ListServiceCatalogsResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return servicecatalogsdk.ListServiceCatalogsResponse{}, conservativeServiceCatalogNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == servicecatalogsdk.ServiceCatalogLifecycleStateDeleted {
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

func handleServiceCatalogDeleteError(resource *servicecatalogv1beta1.ServiceCatalog, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func wrapServiceCatalogDeleteConfirmation(hooks *ServiceCatalogRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getServiceCatalog := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ServiceCatalogServiceClient) ServiceCatalogServiceClient {
		return serviceCatalogDeleteConfirmationClient{
			delegate:          delegate,
			getServiceCatalog: getServiceCatalog,
		}
	})
}

type serviceCatalogDeleteConfirmationClient struct {
	delegate          ServiceCatalogServiceClient
	getServiceCatalog func(context.Context, servicecatalogsdk.GetServiceCatalogRequest) (servicecatalogsdk.GetServiceCatalogResponse, error)
}

func (c serviceCatalogDeleteConfirmationClient) CreateOrUpdate(ctx context.Context, resource *servicecatalogv1beta1.ServiceCatalog, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c serviceCatalogDeleteConfirmationClient) Delete(ctx context.Context, resource *servicecatalogv1beta1.ServiceCatalog) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c serviceCatalogDeleteConfirmationClient) rejectAuthShapedConfirmRead(ctx context.Context, resource *servicecatalogv1beta1.ServiceCatalog) error {
	if c.getServiceCatalog == nil || resource == nil {
		return nil
	}
	serviceCatalogID := trackedServiceCatalogID(resource)
	if serviceCatalogID == "" {
		return nil
	}
	_, err := c.getServiceCatalog(ctx, servicecatalogsdk.GetServiceCatalogRequest{ServiceCatalogId: &serviceCatalogID})
	if err == nil {
		return nil
	}
	if !isAmbiguousServiceCatalogNotFound(err) {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("ServiceCatalog delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %w", err)
}

func trackedServiceCatalogID(resource *servicecatalogv1beta1.ServiceCatalog) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func isAmbiguousServiceCatalogNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousServiceCatalogNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func conservativeServiceCatalogNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("ServiceCatalog %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousServiceCatalogNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousServiceCatalogNotFoundError{message: message}
}

func serviceCatalogFromResponse(response any) (servicecatalogsdk.ServiceCatalog, bool) {
	if current, ok := serviceCatalogFromWriteResponse(response); ok {
		return current, true
	}
	if current, ok := serviceCatalogFromReadResponse(response); ok {
		return current, true
	}
	return serviceCatalogFromListItem(response)
}

func serviceCatalogFromWriteResponse(response any) (servicecatalogsdk.ServiceCatalog, bool) {
	switch current := response.(type) {
	case servicecatalogsdk.CreateServiceCatalogResponse:
		return current.ServiceCatalog, true
	case *servicecatalogsdk.CreateServiceCatalogResponse:
		if current == nil {
			return servicecatalogsdk.ServiceCatalog{}, false
		}
		return current.ServiceCatalog, true
	case servicecatalogsdk.UpdateServiceCatalogResponse:
		return current.ServiceCatalog, true
	case *servicecatalogsdk.UpdateServiceCatalogResponse:
		if current == nil {
			return servicecatalogsdk.ServiceCatalog{}, false
		}
		return current.ServiceCatalog, true
	default:
		return servicecatalogsdk.ServiceCatalog{}, false
	}
}

func serviceCatalogFromReadResponse(response any) (servicecatalogsdk.ServiceCatalog, bool) {
	switch current := response.(type) {
	case servicecatalogsdk.GetServiceCatalogResponse:
		return current.ServiceCatalog, true
	case *servicecatalogsdk.GetServiceCatalogResponse:
		if current == nil {
			return servicecatalogsdk.ServiceCatalog{}, false
		}
		return current.ServiceCatalog, true
	case servicecatalogsdk.ServiceCatalog:
		return current, true
	case *servicecatalogsdk.ServiceCatalog:
		if current == nil {
			return servicecatalogsdk.ServiceCatalog{}, false
		}
		return *current, true
	case serviceCatalogProjectedResponse:
		return serviceCatalogFromProjection(current.ServiceCatalog), true
	case *serviceCatalogProjectedResponse:
		if current == nil {
			return servicecatalogsdk.ServiceCatalog{}, false
		}
		return serviceCatalogFromProjection(current.ServiceCatalog), true
	default:
		return servicecatalogsdk.ServiceCatalog{}, false
	}
}

func serviceCatalogFromListItem(response any) (servicecatalogsdk.ServiceCatalog, bool) {
	switch current := response.(type) {
	case servicecatalogsdk.ServiceCatalogSummary:
		return serviceCatalogFromSummary(current), true
	case *servicecatalogsdk.ServiceCatalogSummary:
		if current == nil {
			return servicecatalogsdk.ServiceCatalog{}, false
		}
		return serviceCatalogFromSummary(*current), true
	case serviceCatalogStatusProjection:
		return serviceCatalogFromProjection(current), true
	case *serviceCatalogStatusProjection:
		if current == nil {
			return servicecatalogsdk.ServiceCatalog{}, false
		}
		return serviceCatalogFromProjection(*current), true
	default:
		return servicecatalogsdk.ServiceCatalog{}, false
	}
}

func serviceCatalogFromProjection(projection serviceCatalogStatusProjection) servicecatalogsdk.ServiceCatalog {
	status, _, _ := serviceCatalogSpecStatus(projection.Status)
	return servicecatalogsdk.ServiceCatalog{
		Id:             common.String(projection.Id),
		CompartmentId:  common.String(projection.CompartmentId),
		DisplayName:    common.String(projection.DisplayName),
		LifecycleState: servicecatalogsdk.ServiceCatalogLifecycleStateEnum(projection.LifecycleState),
		Status:         status,
		DefinedTags:    serviceCatalogStatusDefinedTagsToOCI(projection.DefinedTags),
		FreeformTags:   cloneServiceCatalogStringMap(projection.FreeformTags),
		SystemTags:     serviceCatalogStatusDefinedTagsToOCI(projection.SystemTags),
	}
}

func serviceCatalogFromSummary(summary servicecatalogsdk.ServiceCatalogSummary) servicecatalogsdk.ServiceCatalog {
	return servicecatalogsdk.ServiceCatalog{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		DisplayName:    summary.DisplayName,
		LifecycleState: summary.LifecycleState,
		TimeCreated:    summary.TimeCreated,
		Status:         summary.Status,
		DefinedTags:    cloneServiceCatalogDefinedTags(summary.DefinedTags),
		FreeformTags:   cloneServiceCatalogStringMap(summary.FreeformTags),
		SystemTags:     cloneServiceCatalogDefinedTags(summary.SystemTags),
	}
}

func serviceCatalogSpecStatus(value string) (servicecatalogsdk.ServiceCatalogStatusEnumEnum, bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false, nil
	}
	status, ok := servicecatalogsdk.GetMappingServiceCatalogStatusEnumEnum(trimmed)
	if !ok {
		return "", false, fmt.Errorf("unsupported ServiceCatalog status %q", value)
	}
	return status, true, nil
}

func currentServiceCatalogStatusForUpdate(status servicecatalogsdk.ServiceCatalogStatusEnumEnum) servicecatalogsdk.ServiceCatalogStatusEnumEnum {
	if status != "" {
		return status
	}
	return servicecatalogsdk.ServiceCatalogStatusEnumActive
}

func cloneServiceCatalogStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneServiceCatalogDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
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

func serviceCatalogDefinedTagsStatusMap(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
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

func cloneServiceCatalogStatusDefinedTags(source map[string]shared.MapValue) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	cloned := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
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

func serviceCatalogStatusDefinedTagsToOCI(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			converted[namespace] = nil
			continue
		}
		tagValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			tagValues[key] = value
		}
		converted[namespace] = tagValues
	}
	return converted
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
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
