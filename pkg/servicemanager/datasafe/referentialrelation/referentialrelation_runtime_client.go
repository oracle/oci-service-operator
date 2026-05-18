/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package referentialrelation

import (
	"context"
	"fmt"
	"reflect"
	"strings"

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

const (
	referentialRelationKind                           = "ReferentialRelation"
	referentialRelationSensitiveDataModelIDAnnotation = "datasafe.oracle.com/sensitive-data-model-id"
)

type referentialRelationOCIClient interface {
	CreateReferentialRelation(context.Context, datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error)
	GetReferentialRelation(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error)
	ListReferentialRelations(context.Context, datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error)
	DeleteReferentialRelation(context.Context, datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error)
}

type referentialRelationListCall func(context.Context, datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error)
type referentialRelationGetCall func(context.Context, datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error)

type referentialRelationIdentity struct {
	sensitiveDataModelID string
	key                  string
}

type referentialRelationRuntimeClient struct {
	delegate ReferentialRelationServiceClient
}

type referentialRelationReadModel struct {
	Id                   string                                    `json:"id,omitempty"`
	Key                  string                                    `json:"key,omitempty"`
	LifecycleState       string                                    `json:"lifecycleState,omitempty"`
	SensitiveDataModelId string                                    `json:"sensitiveDataModelId,omitempty"`
	RelationType         string                                    `json:"relationType,omitempty"`
	Parent               datasafev1beta1.ReferentialRelationParent `json:"parent,omitempty"`
	Child                datasafev1beta1.ReferentialRelationChild  `json:"child,omitempty"`
	IsSensitive          bool                                      `json:"isSensitive,omitempty"`
}

type referentialRelationReadCollection struct {
	Items []referentialRelationReadModel `json:"items"`
}

type referentialRelationAuthShapedNotFound struct {
	err error
}

func (e referentialRelationAuthShapedNotFound) Error() string {
	return fmt.Sprintf("datasafe ReferentialRelation delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e referentialRelationAuthShapedNotFound) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func (e referentialRelationAuthShapedNotFound) Unwrap() error {
	return e.err
}

type referentialRelationAuthShapedConfirmRead struct {
	err error
}

func (e referentialRelationAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("datasafe ReferentialRelation delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e referentialRelationAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func (e referentialRelationAuthShapedConfirmRead) Unwrap() error {
	return e.err
}

type referentialRelationSpecNotFound struct {
	sensitiveDataModelID string
}

func (e referentialRelationSpecNotFound) Error() string {
	return fmt.Sprintf("datasafe ReferentialRelation matching spec was not found in sensitive data model %q", e.sensitiveDataModelID)
}

func (e referentialRelationSpecNotFound) Is(target error) bool {
	return target != nil && target.Error() == "generated runtime resource not found"
}

func init() {
	registerReferentialRelationRuntimeHooksMutator(func(_ *ReferentialRelationServiceManager, hooks *ReferentialRelationRuntimeHooks) {
		applyReferentialRelationRuntimeHooks(hooks)
	})
}

func applyReferentialRelationRuntimeHooks(hooks *ReferentialRelationRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = referentialRelationRuntimeSemantics()
	hooks.BuildCreateBody = buildReferentialRelationCreateBody
	hooks.Identity.Resolve = resolveReferentialRelationIdentity
	hooks.Identity.RecordPath = recordReferentialRelationPathIdentity
	hooks.Identity.RecordTracked = recordReferentialRelationTrackedIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateReferentialRelationCreateOnlyDrift
	hooks.Create.Fields = referentialRelationCreateFields()
	hooks.Get.Fields = referentialRelationGetFields()
	hooks.List.Fields = referentialRelationListFields()
	hooks.Delete.Fields = referentialRelationDeleteFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listReferentialRelationsAllPages(hooks.List.Call)
	}
	hooks.Identity.LookupExisting = referentialRelationLookupExisting(hooks.List.Call)
	hooks.Read.Get = referentialRelationReadGetOperation(hooks.Get.Call)
	hooks.Read.List = referentialRelationReadListOperation(hooks.List.Call)
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *datasafev1beta1.ReferentialRelation, currentID string) (any, error) {
		return confirmReferentialRelationDeleteRead(ctx, resource, currentID, hooks.Get.Call, hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleReferentialRelationDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyReferentialRelationDeleteOutcome
	hooks.StatusHooks.ProjectStatus = projectReferentialRelationStatus
	hooks.StatusHooks.MarkTerminating = markReferentialRelationTerminating
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ReferentialRelationServiceClient) ReferentialRelationServiceClient {
		return referentialRelationRuntimeClient{delegate: delegate}
	})
}

func newReferentialRelationServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client referentialRelationOCIClient,
) ReferentialRelationServiceClient {
	hooks := newReferentialRelationRuntimeHooksWithOCIClient(client)
	applyReferentialRelationRuntimeHooks(&hooks)
	manager := &ReferentialRelationServiceManager{Log: log}
	delegate := defaultReferentialRelationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.ReferentialRelation](
			buildReferentialRelationGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapReferentialRelationGeneratedClient(hooks, delegate)
}

func newReferentialRelationRuntimeHooksWithOCIClient(client referentialRelationOCIClient) ReferentialRelationRuntimeHooks {
	return ReferentialRelationRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.ReferentialRelation]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.ReferentialRelation]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.ReferentialRelation]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.ReferentialRelation]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.ReferentialRelation]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.ReferentialRelation]{},
		Create: runtimeOperationHooks[datasafesdk.CreateReferentialRelationRequest, datasafesdk.CreateReferentialRelationResponse]{
			Fields: referentialRelationCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateReferentialRelationRequest) (datasafesdk.CreateReferentialRelationResponse, error) {
				if client == nil {
					return datasafesdk.CreateReferentialRelationResponse{}, fmt.Errorf("%s OCI client is nil", referentialRelationKind)
				}
				return client.CreateReferentialRelation(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetReferentialRelationRequest, datasafesdk.GetReferentialRelationResponse]{
			Fields: referentialRelationGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetReferentialRelationRequest) (datasafesdk.GetReferentialRelationResponse, error) {
				if client == nil {
					return datasafesdk.GetReferentialRelationResponse{}, fmt.Errorf("%s OCI client is nil", referentialRelationKind)
				}
				return client.GetReferentialRelation(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListReferentialRelationsRequest, datasafesdk.ListReferentialRelationsResponse]{
			Fields: referentialRelationListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
				if client == nil {
					return datasafesdk.ListReferentialRelationsResponse{}, fmt.Errorf("%s OCI client is nil", referentialRelationKind)
				}
				return client.ListReferentialRelations(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteReferentialRelationRequest, datasafesdk.DeleteReferentialRelationResponse]{
			Fields: referentialRelationDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteReferentialRelationRequest) (datasafesdk.DeleteReferentialRelationResponse, error) {
				if client == nil {
					return datasafesdk.DeleteReferentialRelationResponse{}, fmt.Errorf("%s OCI client is nil", referentialRelationKind)
				}
				return client.DeleteReferentialRelation(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ReferentialRelationServiceClient) ReferentialRelationServiceClient{},
	}
}

func referentialRelationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "referentialrelation",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.ReferentialRelationLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.ReferentialRelationLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.ReferentialRelationLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.ReferentialRelationLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"sensitiveDataModelId",
				"relationType",
				"parent",
				"child",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew: []string{
				"sensitiveDataModelId",
				"relationType",
				"parent",
				"child",
				"isSensitive",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "generatedruntime.Create", EntityType: referentialRelationKind, Action: "CreateReferentialRelation"}},
			Delete: []generatedruntime.Hook{{Helper: "generatedruntime.Delete", EntityType: referentialRelationKind, Action: "DeleteReferentialRelation"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func referentialRelationCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SensitiveDataModelId",
			RequestName:  "sensitiveDataModelId",
			Contribution: "path",
			LookupPaths:  referentialRelationSensitiveDataModelLookupPaths(),
		},
		{FieldName: "CreateReferentialRelationDetails", RequestName: "CreateReferentialRelationDetails", Contribution: "body"},
	}
}

func referentialRelationGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SensitiveDataModelId",
			RequestName:  "sensitiveDataModelId",
			Contribution: "path",
			LookupPaths:  referentialRelationSensitiveDataModelLookupPaths(),
		},
		{
			FieldName:        "ReferentialRelationKey",
			RequestName:      "referentialRelationKey",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.key", "key"},
		},
	}
}

func referentialRelationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "SensitiveDataModelId",
			RequestName:  "sensitiveDataModelId",
			Contribution: "path",
			LookupPaths:  referentialRelationSensitiveDataModelLookupPaths(),
		},
		{FieldName: "ColumnName", RequestName: "columnName", Contribution: "query", LookupPaths: []string{"status.child.columnGroup", "spec.child.columnGroup", "child.columnGroup"}},
		{FieldName: "IsSensitive", RequestName: "isSensitive", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func referentialRelationDeleteFields() []generatedruntime.RequestField {
	return referentialRelationGetFields()
}

func referentialRelationSensitiveDataModelLookupPaths() []string {
	return []string{"status.sensitiveDataModelId", "sensitiveDataModelId"}
}

func buildReferentialRelationCreateBody(_ context.Context, resource *datasafev1beta1.ReferentialRelation, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", referentialRelationKind)
	}
	parent, err := referentialRelationColumnsInfoFromParent(resource.Spec.Parent)
	if err != nil {
		return nil, fmt.Errorf("parent: %w", err)
	}
	child, err := referentialRelationColumnsInfoFromChild(resource.Spec.Child)
	if err != nil {
		return nil, fmt.Errorf("child: %w", err)
	}
	return datasafesdk.CreateReferentialRelationDetails{
		RelationType: datasafesdk.CreateReferentialRelationDetailsRelationTypeEnum(strings.TrimSpace(resource.Spec.RelationType)),
		Parent:       &parent,
		Child:        &child,
		IsSensitive:  common.Bool(resource.Spec.IsSensitive),
	}, nil
}

func resolveReferentialRelationIdentity(resource *datasafev1beta1.ReferentialRelation) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", referentialRelationKind)
	}
	parentID := referentialRelationSensitiveDataModelID(resource)
	if parentID == "" {
		return nil, fmt.Errorf("%s requires annotation %q or status.sensitiveDataModelId to address the parent sensitive data model", referentialRelationKind, referentialRelationSensitiveDataModelIDAnnotation)
	}
	return referentialRelationIdentity{
		sensitiveDataModelID: parentID,
		key:                  referentialRelationTrackedKey(resource),
	}, nil
}

func recordReferentialRelationPathIdentity(resource *datasafev1beta1.ReferentialRelation, identity any) {
	if resource == nil {
		return
	}
	typed, ok := identity.(referentialRelationIdentity)
	if !ok {
		return
	}
	if typed.sensitiveDataModelID != "" {
		resource.Status.SensitiveDataModelId = typed.sensitiveDataModelID
	}
	if typed.key != "" {
		resource.Status.Key = typed.key
		resource.Status.OsokStatus.Ocid = shared.OCID(typed.key)
	}
}

func recordReferentialRelationTrackedIdentity(resource *datasafev1beta1.ReferentialRelation, identity any, resourceID string) {
	if resource == nil {
		return
	}
	if typed, ok := identity.(referentialRelationIdentity); ok && typed.sensitiveDataModelID != "" {
		resource.Status.SensitiveDataModelId = typed.sensitiveDataModelID
	}
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		resourceID = referentialRelationTrackedKey(resource)
	}
	if resourceID == "" {
		return
	}
	resource.Status.Key = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
}

func referentialRelationLookupExisting(list referentialRelationListCall) func(context.Context, *datasafev1beta1.ReferentialRelation, any) (any, error) {
	if list == nil {
		return nil
	}
	return func(ctx context.Context, resource *datasafev1beta1.ReferentialRelation, identity any) (any, error) {
		if resource == nil {
			return nil, nil
		}
		typed, ok := identity.(referentialRelationIdentity)
		if !ok || typed.sensitiveDataModelID == "" {
			return nil, nil
		}
		model, found, err := findReferentialRelationBySpec(ctx, list, resource, typed.sensitiveDataModelID)
		if err != nil {
			return nil, err
		}
		if found {
			return model, nil
		}
		return nil, nil
	}
}

func findReferentialRelationBySpec(
	ctx context.Context,
	list referentialRelationListCall,
	resource *datasafev1beta1.ReferentialRelation,
	parentID string,
) (referentialRelationReadModel, bool, error) {
	if list == nil {
		return referentialRelationReadModel{}, false, fmt.Errorf("list %s hook is not configured", referentialRelationKind)
	}
	response, err := list(ctx, datasafesdk.ListReferentialRelationsRequest{
		SensitiveDataModelId: common.String(parentID),
		ColumnName:           referentialRelationStringSlice(resource.Spec.Child.ColumnGroup),
		IsSensitive:          common.Bool(resource.Spec.IsSensitive),
	})
	if err != nil {
		return referentialRelationReadModel{}, false, err
	}
	for _, item := range response.Items {
		model := referentialRelationReadModelFromSummary(item)
		if referentialRelationReadModelMatchesResource(model, resource, parentID) {
			return model, true, nil
		}
	}
	return referentialRelationReadModel{}, false, nil
}

func listReferentialRelationsAllPages(next referentialRelationListCall) referentialRelationListCall {
	return func(ctx context.Context, request datasafesdk.ListReferentialRelationsRequest) (datasafesdk.ListReferentialRelationsResponse, error) {
		var combined datasafesdk.ListReferentialRelationsResponse
		for {
			normalizeReferentialRelationListRequest(&request)
			response, err := next(ctx, request)
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

func normalizeReferentialRelationListRequest(request *datasafesdk.ListReferentialRelationsRequest) {
	if request == nil || request.IsSensitive != nil {
		return
	}
	request.IsSensitive = common.Bool(false)
}

func referentialRelationReadGetOperation(get referentialRelationGetCall) *generatedruntime.Operation {
	if get == nil {
		return nil
	}
	return &generatedruntime.Operation{
		NewRequest: func() any { return &datasafesdk.GetReferentialRelationRequest{} },
		Fields:     referentialRelationGetFields(),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := get(ctx, *request.(*datasafesdk.GetReferentialRelationRequest))
			if err != nil {
				return nil, err
			}
			return referentialRelationReadModelFromSDK(response.ReferentialRelation), nil
		},
	}
}

func referentialRelationReadListOperation(list referentialRelationListCall) *generatedruntime.Operation {
	if list == nil {
		return nil
	}
	return &generatedruntime.Operation{
		NewRequest: func() any { return &datasafesdk.ListReferentialRelationsRequest{} },
		Fields:     referentialRelationListFields(),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := list(ctx, *request.(*datasafesdk.ListReferentialRelationsRequest))
			if err != nil {
				return nil, err
			}
			items := make([]referentialRelationReadModel, 0, len(response.Items))
			for _, item := range response.Items {
				items = append(items, referentialRelationReadModelFromSummary(item))
			}
			return referentialRelationReadCollection{Items: items}, nil
		},
	}
}

func confirmReferentialRelationDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.ReferentialRelation,
	currentID string,
	get referentialRelationGetCall,
	list referentialRelationListCall,
) (any, error) {
	if get == nil {
		return nil, fmt.Errorf("confirm %s delete: get hook is not configured", referentialRelationKind)
	}
	parentID := referentialRelationSensitiveDataModelID(resource)
	if parentID == "" {
		return nil, fmt.Errorf("confirm %s delete: sensitive data model OCID is empty", referentialRelationKind)
	}
	key := strings.TrimSpace(currentID)
	if key == "" {
		key = referentialRelationTrackedKey(resource)
	}
	if key == "" {
		model, found, err := findReferentialRelationBySpec(ctx, list, resource, parentID)
		if err != nil {
			if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
				return nil, referentialRelationAuthShapedConfirmRead{err: err}
			}
			return nil, err
		}
		if !found {
			return nil, referentialRelationSpecNotFound{sensitiveDataModelID: parentID}
		}
		return model, nil
	}

	response, err := get(ctx, datasafesdk.GetReferentialRelationRequest{
		SensitiveDataModelId:   common.String(parentID),
		ReferentialRelationKey: common.String(key),
	})
	if err == nil {
		return referentialRelationReadModelFromSDK(response.ReferentialRelation), nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return referentialRelationAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

func handleReferentialRelationDeleteError(resource *datasafev1beta1.ReferentialRelation, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if ambiguous, ok := referentialRelationConfirmReadError(err); ok {
		return ambiguous
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return referentialRelationAuthShapedNotFound{err: err}
	}
	return err
}

func applyReferentialRelationDeleteOutcome(
	resource *datasafev1beta1.ReferentialRelation,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if ambiguous, ok := referentialRelationConfirmReadError(response); ok {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		}
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest {
		markReferentialRelationTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func referentialRelationConfirmReadError(response any) (referentialRelationAuthShapedConfirmRead, bool) {
	switch typed := response.(type) {
	case referentialRelationAuthShapedConfirmRead:
		return typed, true
	case *referentialRelationAuthShapedConfirmRead:
		if typed != nil {
			return *typed, true
		}
	}
	return referentialRelationAuthShapedConfirmRead{}, false
}

func markReferentialRelationTerminating(resource *datasafev1beta1.ReferentialRelation, _ any) {
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

func projectReferentialRelationStatus(resource *datasafev1beta1.ReferentialRelation, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", referentialRelationKind)
	}
	model, ok := referentialRelationReadModelFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.ReferentialRelationStatus{
		OsokStatus:           osokStatus,
		Key:                  model.Key,
		LifecycleState:       model.LifecycleState,
		SensitiveDataModelId: model.SensitiveDataModelId,
		RelationType:         model.RelationType,
		Parent:               model.Parent,
		Child:                model.Child,
		IsSensitive:          model.IsSensitive,
	}
	if model.Key != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(model.Key)
	}
	return nil
}

func validateReferentialRelationCreateOnlyDrift(resource *datasafev1beta1.ReferentialRelation, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", referentialRelationKind)
	}
	current, ok := referentialRelationReadModelFromResponse(currentResponse)
	if !ok {
		return nil
	}
	desiredParentID := referentialRelationSensitiveDataModelID(resource)
	checks := []struct {
		field   string
		current string
		desired string
	}{
		{field: "sensitiveDataModelId", current: current.SensitiveDataModelId, desired: desiredParentID},
		{field: "relationType", current: current.RelationType, desired: resource.Spec.RelationType},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.current) == "" || strings.TrimSpace(check.desired) == "" {
			continue
		}
		if strings.TrimSpace(check.current) != strings.TrimSpace(check.desired) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", referentialRelationKind, check.field)
		}
	}
	if !referentialRelationParentMatches(current.Parent, resource.Spec.Parent) {
		return fmt.Errorf("%s formal semantics require replacement when parent changes", referentialRelationKind)
	}
	if !referentialRelationChildMatches(current.Child, resource.Spec.Child) {
		return fmt.Errorf("%s formal semantics require replacement when child changes", referentialRelationKind)
	}
	if current.IsSensitive != resource.Spec.IsSensitive {
		return fmt.Errorf("%s formal semantics require replacement when isSensitive changes", referentialRelationKind)
	}
	return nil
}

func (c referentialRelationRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.ReferentialRelation,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", referentialRelationKind)
	}
	if err := validateReferentialRelationCreateOrUpdateIdentity(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, markReferentialRelationFailed(resource, err)
	}
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	syncReferentialRelationTrackedKey(resource)
	return response, err
}

func (c referentialRelationRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.ReferentialRelation) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", referentialRelationKind)
	}
	syncReferentialRelationTrackedKey(resource)
	return c.delegate.Delete(ctx, resource)
}

func validateReferentialRelationCreateOrUpdateIdentity(resource *datasafev1beta1.ReferentialRelation) error {
	if resource == nil {
		return nil
	}
	trackedParentID := strings.TrimSpace(resource.Status.SensitiveDataModelId)
	annotationParentID := referentialRelationAnnotationParentID(resource)
	if trackedParentID != "" && annotationParentID != "" && trackedParentID != annotationParentID {
		return fmt.Errorf("%s formal semantics require replacement when sensitiveDataModelId changes", referentialRelationKind)
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" && resource.Status.Key != "" && ocid != resource.Status.Key {
		return fmt.Errorf("%s status key %q conflicts with tracked status.status.ocid %q", referentialRelationKind, resource.Status.Key, ocid)
	}
	return nil
}

func markReferentialRelationFailed(resource *datasafev1beta1.ReferentialRelation, err error) error {
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

func referentialRelationReadModelFromResponse(response any) (referentialRelationReadModel, bool) {
	switch current := response.(type) {
	case referentialRelationReadModel:
		return current, true
	case *referentialRelationReadModel:
		if current == nil {
			return referentialRelationReadModel{}, false
		}
		return *current, true
	case datasafesdk.GetReferentialRelationResponse:
		return referentialRelationReadModelFromSDK(current.ReferentialRelation), true
	case *datasafesdk.GetReferentialRelationResponse:
		if current == nil {
			return referentialRelationReadModel{}, false
		}
		return referentialRelationReadModelFromSDK(current.ReferentialRelation), true
	default:
		return referentialRelationReadModelFromResourceResponse(response)
	}
}

func referentialRelationReadModelFromResourceResponse(response any) (referentialRelationReadModel, bool) {
	switch current := response.(type) {
	case datasafesdk.ReferentialRelation:
		return referentialRelationReadModelFromSDK(current), true
	case *datasafesdk.ReferentialRelation:
		if current == nil {
			return referentialRelationReadModel{}, false
		}
		return referentialRelationReadModelFromSDK(*current), true
	case datasafesdk.ReferentialRelationSummary:
		return referentialRelationReadModelFromSummary(current), true
	case *datasafesdk.ReferentialRelationSummary:
		if current == nil {
			return referentialRelationReadModel{}, false
		}
		return referentialRelationReadModelFromSummary(*current), true
	default:
		return referentialRelationReadModel{}, false
	}
}

func referentialRelationReadModelFromSDK(current datasafesdk.ReferentialRelation) referentialRelationReadModel {
	key := referentialRelationStringValue(current.Key)
	return referentialRelationReadModel{
		Id:                   key,
		Key:                  key,
		LifecycleState:       string(current.LifecycleState),
		SensitiveDataModelId: referentialRelationStringValue(current.SensitiveDataModelId),
		RelationType:         string(current.RelationType),
		Parent:               referentialRelationParentFromSDK(current.Parent),
		Child:                referentialRelationChildFromSDK(current.Child),
		IsSensitive:          referentialRelationBoolValue(current.IsSensitive),
	}
}

func referentialRelationReadModelFromSummary(current datasafesdk.ReferentialRelationSummary) referentialRelationReadModel {
	key := referentialRelationStringValue(current.Key)
	return referentialRelationReadModel{
		Id:                   key,
		Key:                  key,
		LifecycleState:       string(current.LifecycleState),
		SensitiveDataModelId: referentialRelationStringValue(current.SensitiveDataModelId),
		RelationType:         string(current.RelationType),
		Parent:               referentialRelationParentFromSDK(current.Parent),
		Child:                referentialRelationChildFromSDK(current.Child),
		IsSensitive:          referentialRelationBoolValue(current.IsSensitive),
	}
}

func referentialRelationReadModelMatchesResource(model referentialRelationReadModel, resource *datasafev1beta1.ReferentialRelation, parentID string) bool {
	if resource == nil {
		return false
	}
	return strings.TrimSpace(model.SensitiveDataModelId) == strings.TrimSpace(parentID) &&
		strings.TrimSpace(model.RelationType) == strings.TrimSpace(resource.Spec.RelationType) &&
		referentialRelationParentMatches(model.Parent, resource.Spec.Parent) &&
		referentialRelationChildMatches(model.Child, resource.Spec.Child) &&
		model.IsSensitive == resource.Spec.IsSensitive
}

func referentialRelationParentMatches(current, desired datasafev1beta1.ReferentialRelationParent) bool {
	return reflect.DeepEqual(
		normalizeReferentialRelationParentSensitiveTypes(current),
		normalizeReferentialRelationParentSensitiveTypes(desired),
	)
}

func referentialRelationChildMatches(current, desired datasafev1beta1.ReferentialRelationChild) bool {
	return reflect.DeepEqual(
		normalizeReferentialRelationChildSensitiveTypes(current),
		normalizeReferentialRelationChildSensitiveTypes(desired),
	)
}

func normalizeReferentialRelationParentSensitiveTypes(input datasafev1beta1.ReferentialRelationParent) datasafev1beta1.ReferentialRelationParent {
	if len(input.SensitiveTypeIds) == 0 {
		input.SensitiveTypeIds = nil
	}
	return input
}

func normalizeReferentialRelationChildSensitiveTypes(input datasafev1beta1.ReferentialRelationChild) datasafev1beta1.ReferentialRelationChild {
	if len(input.SensitiveTypeIds) == 0 {
		input.SensitiveTypeIds = nil
	}
	return input
}

func referentialRelationColumnsInfoFromParent(input datasafev1beta1.ReferentialRelationParent) (datasafesdk.ColumnsInfo, error) {
	return referentialRelationColumnsInfo(input.SchemaName, input.ObjectType, input.ObjectName, input.AppName, input.ColumnGroup, input.SensitiveTypeIds)
}

func referentialRelationColumnsInfoFromChild(input datasafev1beta1.ReferentialRelationChild) (datasafesdk.ColumnsInfo, error) {
	return referentialRelationColumnsInfo(input.SchemaName, input.ObjectType, input.ObjectName, input.AppName, input.ColumnGroup, input.SensitiveTypeIds)
}

func referentialRelationColumnsInfo(
	schemaName string,
	objectType string,
	objectName string,
	appName string,
	columnGroup []string,
	sensitiveTypeIDs []string,
) (datasafesdk.ColumnsInfo, error) {
	if strings.TrimSpace(schemaName) == "" {
		return datasafesdk.ColumnsInfo{}, fmt.Errorf("schemaName is required")
	}
	if strings.TrimSpace(objectType) == "" {
		return datasafesdk.ColumnsInfo{}, fmt.Errorf("objectType is required")
	}
	if strings.TrimSpace(objectName) == "" {
		return datasafesdk.ColumnsInfo{}, fmt.Errorf("objectName is required")
	}
	if strings.TrimSpace(appName) == "" {
		return datasafesdk.ColumnsInfo{}, fmt.Errorf("appName is required")
	}
	if len(columnGroup) == 0 {
		return datasafesdk.ColumnsInfo{}, fmt.Errorf("columnGroup is required")
	}
	return datasafesdk.ColumnsInfo{
		SchemaName:       common.String(strings.TrimSpace(schemaName)),
		ObjectType:       datasafesdk.ColumnsInfoObjectTypeEnum(strings.TrimSpace(objectType)),
		ObjectName:       common.String(strings.TrimSpace(objectName)),
		AppName:          common.String(strings.TrimSpace(appName)),
		ColumnGroup:      referentialRelationStringSlice(columnGroup),
		SensitiveTypeIds: referentialRelationStringSlice(sensitiveTypeIDs),
	}, nil
}

func referentialRelationParentFromSDK(input *datasafesdk.ColumnsInfo) datasafev1beta1.ReferentialRelationParent {
	if input == nil {
		return datasafev1beta1.ReferentialRelationParent{}
	}
	return datasafev1beta1.ReferentialRelationParent{
		SchemaName:       referentialRelationStringValue(input.SchemaName),
		ObjectType:       string(input.ObjectType),
		ObjectName:       referentialRelationStringValue(input.ObjectName),
		AppName:          referentialRelationStringValue(input.AppName),
		ColumnGroup:      referentialRelationStringSlice(input.ColumnGroup),
		SensitiveTypeIds: referentialRelationStringSlice(input.SensitiveTypeIds),
	}
}

func referentialRelationChildFromSDK(input *datasafesdk.ColumnsInfo) datasafev1beta1.ReferentialRelationChild {
	if input == nil {
		return datasafev1beta1.ReferentialRelationChild{}
	}
	return datasafev1beta1.ReferentialRelationChild{
		SchemaName:       referentialRelationStringValue(input.SchemaName),
		ObjectType:       string(input.ObjectType),
		ObjectName:       referentialRelationStringValue(input.ObjectName),
		AppName:          referentialRelationStringValue(input.AppName),
		ColumnGroup:      referentialRelationStringSlice(input.ColumnGroup),
		SensitiveTypeIds: referentialRelationStringSlice(input.SensitiveTypeIds),
	}
}

func referentialRelationSensitiveDataModelID(resource *datasafev1beta1.ReferentialRelation) string {
	if resource == nil {
		return ""
	}
	if parentID := strings.TrimSpace(resource.Status.SensitiveDataModelId); parentID != "" {
		return parentID
	}
	return referentialRelationAnnotationParentID(resource)
}

func referentialRelationAnnotationParentID(resource *datasafev1beta1.ReferentialRelation) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Annotations[referentialRelationSensitiveDataModelIDAnnotation])
}

func referentialRelationTrackedKey(resource *datasafev1beta1.ReferentialRelation) string {
	if resource == nil {
		return ""
	}
	if key := strings.TrimSpace(resource.Status.Key); key != "" {
		return key
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func syncReferentialRelationTrackedKey(resource *datasafev1beta1.ReferentialRelation) {
	if resource == nil {
		return
	}
	if resource.Status.Key == "" {
		resource.Status.Key = strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	if resource.Status.OsokStatus.Ocid == "" && resource.Status.Key != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Key)
	}
}

func referentialRelationStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func referentialRelationBoolValue(value *bool) bool {
	return value != nil && *value
}

func referentialRelationStringSlice(source []string) []string {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]string, len(source))
	copy(cloned, source)
	return cloned
}

var _ interface{ GetOpcRequestID() string } = referentialRelationAuthShapedNotFound{}
var _ interface{ GetOpcRequestID() string } = referentialRelationAuthShapedConfirmRead{}
