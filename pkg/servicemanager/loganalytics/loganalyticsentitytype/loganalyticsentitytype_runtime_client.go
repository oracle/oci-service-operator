/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsentitytype

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	objectstoragesdk "github.com/oracle/oci-go-sdk/v65/objectstorage"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type logAnalyticsEntityTypeOCIClient interface {
	CreateLogAnalyticsEntityType(context.Context, loganalyticssdk.CreateLogAnalyticsEntityTypeRequest) (loganalyticssdk.CreateLogAnalyticsEntityTypeResponse, error)
	GetLogAnalyticsEntityType(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error)
	ListLogAnalyticsEntityTypes(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error)
	UpdateLogAnalyticsEntityType(context.Context, loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest) (loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse, error)
	DeleteLogAnalyticsEntityType(context.Context, loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error)
}

type logAnalyticsEntityTypeNamespaceGetter interface {
	GetNamespace(context.Context, objectstoragesdk.GetNamespaceRequest) (objectstoragesdk.GetNamespaceResponse, error)
}

type logAnalyticsEntityTypeNamespaceResolver struct {
	getter  logAnalyticsEntityTypeNamespaceGetter
	initErr error

	mu        sync.Mutex
	namespace string
}

type logAnalyticsEntityTypeIdentity struct {
	EntityTypeName string
}

type logAnalyticsEntityTypeIdentified struct {
	ID                               string                                    `json:"id,omitempty"`
	Name                             *string                                   `json:"name,omitempty"`
	InternalName                     *string                                   `json:"internalName,omitempty"`
	Category                         *string                                   `json:"category,omitempty"`
	CloudType                        loganalyticssdk.EntityCloudTypeEnum       `json:"cloudType,omitempty"`
	LifecycleState                   loganalyticssdk.EntityLifecycleStatesEnum `json:"lifecycleState,omitempty"`
	Properties                       []loganalyticssdk.EntityTypeProperty      `json:"properties,omitempty"`
	ManagementAgentEligibilityStatus string                                    `json:"managementAgentEligibilityStatus,omitempty"`
}

type ambiguousLogAnalyticsEntityTypeNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousLogAnalyticsEntityTypeNotFoundError) Error() string {
	return e.message
}

func (e ambiguousLogAnalyticsEntityTypeNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerLogAnalyticsEntityTypeRuntimeHooksMutator(func(manager *LogAnalyticsEntityTypeServiceManager, hooks *LogAnalyticsEntityTypeRuntimeHooks) {
		applyLogAnalyticsEntityTypeRuntimeHooks(manager, hooks)
	})
}

func applyLogAnalyticsEntityTypeRuntimeHooks(manager *LogAnalyticsEntityTypeServiceManager, hooks *LogAnalyticsEntityTypeRuntimeHooks) {
	applyLogAnalyticsEntityTypeRuntimeHooksWithNamespaceResolver(manager, hooks, newLogAnalyticsEntityTypeNamespaceResolver(manager))
}

func applyLogAnalyticsEntityTypeRuntimeHooksWithNamespaceResolver(
	manager *LogAnalyticsEntityTypeServiceManager,
	hooks *LogAnalyticsEntityTypeRuntimeHooks,
	namespaceResolver *logAnalyticsEntityTypeNamespaceResolver,
) {
	if hooks == nil {
		return
	}

	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}
	hooks.Semantics = logAnalyticsEntityTypeRuntimeSemantics()
	hooks.BuildCreateBody = buildLogAnalyticsEntityTypeCreateBody
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loganalyticsv1beta1.LogAnalyticsEntityType,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildLogAnalyticsEntityTypeUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = logAnalyticsEntityTypeCreateFields()
	hooks.Get.Fields = logAnalyticsEntityTypeGetFields()
	hooks.List.Fields = logAnalyticsEntityTypeListFields()
	hooks.Update.Fields = logAnalyticsEntityTypeUpdateFields()
	hooks.Delete.Fields = logAnalyticsEntityTypeDeleteFields()
	wrapLogAnalyticsEntityTypeListPages(hooks)
	applyLogAnalyticsEntityTypeNamespaceRuntimeHooks(hooks, namespaceResolver)
	hooks.Identity = generatedruntime.IdentityHooks[*loganalyticsv1beta1.LogAnalyticsEntityType]{
		Resolve:                   resolveLogAnalyticsEntityTypeIdentity,
		RecordTracked:             recordLogAnalyticsEntityTypeTrackedIdentity,
		GuardExistingBeforeCreate: guardLogAnalyticsEntityTypeExistingBeforeCreate,
		LookupExisting:            lookupLogAnalyticsEntityTypeExisting(hooks.List.Call),
	}
	hooks.DeleteHooks.HandleError = handleLogAnalyticsEntityTypeDeleteError
	hooks.DeleteHooks.ConfirmRead = logAnalyticsEntityTypeDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.ApplyOutcome = func(
		resource *loganalyticsv1beta1.LogAnalyticsEntityType,
		response any,
		stage generatedruntime.DeleteConfirmStage,
	) (generatedruntime.DeleteOutcome, error) {
		return applyLogAnalyticsEntityTypeDeleteOutcome(resource, response, stage, log)
	}
	hooks.WrapGeneratedClient = append(
		hooks.WrapGeneratedClient,
		wrapLogAnalyticsEntityTypeDeleteListAbsence(hooks.List.Call, log),
	)
}

func newLogAnalyticsEntityTypeNamespaceResolver(manager *LogAnalyticsEntityTypeServiceManager) *logAnalyticsEntityTypeNamespaceResolver {
	resolver := &logAnalyticsEntityTypeNamespaceResolver{}
	if manager == nil {
		resolver.initErr = fmt.Errorf("LogAnalyticsEntityType namespace resolver requires service manager")
		return resolver
	}
	sdkClient, err := objectstoragesdk.NewObjectStorageClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		resolver.initErr = fmt.Errorf("initialize Object Storage namespace client for LogAnalyticsEntityType: %w", err)
		return resolver
	}
	resolver.getter = sdkClient
	return resolver
}

func newLogAnalyticsEntityTypeNamespaceResolverWithGetter(
	getter logAnalyticsEntityTypeNamespaceGetter,
) *logAnalyticsEntityTypeNamespaceResolver {
	return &logAnalyticsEntityTypeNamespaceResolver{getter: getter}
}

func (r *logAnalyticsEntityTypeNamespaceResolver) resolve(ctx context.Context) (string, error) {
	if r == nil {
		return "", fmt.Errorf("LogAnalyticsEntityType namespace resolver is not configured")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.namespace != "" {
		return r.namespace, nil
	}
	if r.initErr != nil {
		return "", r.initErr
	}
	if r.getter == nil {
		return "", fmt.Errorf("LogAnalyticsEntityType namespace resolver is not configured")
	}

	response, err := r.getter.GetNamespace(ctx, objectstoragesdk.GetNamespaceRequest{})
	if err != nil {
		return "", fmt.Errorf("lookup LogAnalyticsEntityType namespace: %w", err)
	}
	namespace := strings.TrimSpace(derefLogAnalyticsEntityTypeString(response.Value))
	if namespace == "" {
		return "", fmt.Errorf("lookup LogAnalyticsEntityType namespace: OCI returned empty namespace")
	}
	r.namespace = namespace
	return namespace, nil
}

func applyLogAnalyticsEntityTypeNamespaceRuntimeHooks(
	hooks *LogAnalyticsEntityTypeRuntimeHooks,
	resolver *logAnalyticsEntityTypeNamespaceResolver,
) {
	if hooks == nil {
		return
	}

	hooks.Create.Call = withLogAnalyticsEntityTypeNamespace(hooks.Create.Call, resolver, func(
		request *loganalyticssdk.CreateLogAnalyticsEntityTypeRequest,
		namespace *string,
	) {
		request.NamespaceName = namespace
	})
	hooks.Get.Call = withLogAnalyticsEntityTypeNamespace(hooks.Get.Call, resolver, func(
		request *loganalyticssdk.GetLogAnalyticsEntityTypeRequest,
		namespace *string,
	) {
		request.NamespaceName = namespace
	})
	hooks.List.Call = withLogAnalyticsEntityTypeNamespace(hooks.List.Call, resolver, func(
		request *loganalyticssdk.ListLogAnalyticsEntityTypesRequest,
		namespace *string,
	) {
		request.NamespaceName = namespace
	})
	hooks.Update.Call = withLogAnalyticsEntityTypeNamespace(hooks.Update.Call, resolver, func(
		request *loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest,
		namespace *string,
	) {
		request.NamespaceName = namespace
	})
	hooks.Delete.Call = withLogAnalyticsEntityTypeNamespace(hooks.Delete.Call, resolver, func(
		request *loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest,
		namespace *string,
	) {
		request.NamespaceName = namespace
	})
}

func withLogAnalyticsEntityTypeNamespace[Req any, Resp any](
	call func(context.Context, Req) (Resp, error),
	resolver *logAnalyticsEntityTypeNamespaceResolver,
	setNamespace func(*Req, *string),
) func(context.Context, Req) (Resp, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request Req) (Resp, error) {
		namespace, err := resolver.resolve(ctx)
		if err != nil {
			var zero Resp
			return zero, err
		}
		setNamespace(&request, logAnalyticsEntityTypeString(namespace))
		return call(ctx, request)
	}
}

func logAnalyticsEntityTypeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "loganalytics",
		FormalSlug:        "loganalyticsentitytype",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(loganalyticssdk.EntityLifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(loganalyticssdk.EntityLifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"category", "properties"},
			ForceNew:      []string{"name"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func logAnalyticsEntityTypeCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateLogAnalyticsEntityTypeDetails", RequestName: "CreateLogAnalyticsEntityTypeDetails", Contribution: "body"},
	}
}

func logAnalyticsEntityTypeGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "EntityTypeName",
			RequestName:      "entityTypeName",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.internalName", "internalName", "spec.name", "name"},
		},
	}
}

func logAnalyticsEntityTypeListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"spec.name", "name", "status.name"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func logAnalyticsEntityTypeUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "EntityTypeName",
			RequestName:      "entityTypeName",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.internalName", "internalName", "spec.name", "name"},
		},
		{FieldName: "UpdateLogAnalyticsEntityTypeDetails", RequestName: "UpdateLogAnalyticsEntityTypeDetails", Contribution: "body"},
	}
}

func logAnalyticsEntityTypeDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "EntityTypeName",
			RequestName:      "entityTypeName",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.internalName", "internalName", "spec.name", "name"},
		},
	}
}

func buildLogAnalyticsEntityTypeCreateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("LogAnalyticsEntityType resource is nil")
	}
	if strings.TrimSpace(resource.Spec.Name) == "" {
		return nil, fmt.Errorf("LogAnalyticsEntityType spec.name is required")
	}

	body := loganalyticssdk.CreateLogAnalyticsEntityTypeDetails{
		Name: logAnalyticsEntityTypeString(resource.Spec.Name),
	}
	if strings.TrimSpace(resource.Spec.Category) != "" {
		body.Category = logAnalyticsEntityTypeString(resource.Spec.Category)
	}
	if resource.Spec.Properties != nil {
		body.Properties = logAnalyticsEntityTypePropertiesFromSpec(resource.Spec.Properties)
	}
	return body, nil
}

func buildLogAnalyticsEntityTypeUpdateBody(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	currentResponse any,
) (loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails, bool, error) {
	if resource == nil {
		return loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails{}, false, fmt.Errorf("LogAnalyticsEntityType resource is nil")
	}
	current, hasProperties, ok := logAnalyticsEntityTypeFromResponse(currentResponse)
	if !ok {
		return loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails{}, false, fmt.Errorf("current LogAnalyticsEntityType response does not expose an entity type body")
	}
	if err := validateLogAnalyticsEntityTypeName(resource, current); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails{}, false, err
	}

	details := loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails{}
	updateNeeded := applyLogAnalyticsEntityTypeCategoryUpdate(resource, current, &details)
	updateNeeded = applyLogAnalyticsEntityTypePropertiesUpdate(resource, current, hasProperties, &details) || updateNeeded

	return details, updateNeeded, nil
}

func validateLogAnalyticsEntityTypeName(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	current loganalyticssdk.LogAnalyticsEntityType,
) error {
	currentName := strings.TrimSpace(derefLogAnalyticsEntityTypeString(current.Name))
	specName := strings.TrimSpace(resource.Spec.Name)
	if currentName == "" || specName == "" || currentName == specName {
		return nil
	}
	return fmt.Errorf("LogAnalyticsEntityType name is immutable and cannot change from %q to %q", currentName, specName)
}

func applyLogAnalyticsEntityTypeCategoryUpdate(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	current loganalyticssdk.LogAnalyticsEntityType,
	details *loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails,
) bool {
	if strings.TrimSpace(resource.Spec.Category) == "" || stringsEqual(resource.Spec.Category, current.Category) {
		return false
	}
	details.Category = logAnalyticsEntityTypeString(resource.Spec.Category)
	return true
}

func applyLogAnalyticsEntityTypePropertiesUpdate(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	current loganalyticssdk.LogAnalyticsEntityType,
	hasProperties bool,
	details *loganalyticssdk.UpdateLogAnalyticsEntityTypeDetails,
) bool {
	if resource.Spec.Properties == nil || !hasProperties {
		return false
	}
	desired := logAnalyticsEntityTypePropertiesFromSpec(resource.Spec.Properties)
	if reflect.DeepEqual(desired, current.Properties) {
		return false
	}
	details.Properties = desired
	return true
}

func resolveLogAnalyticsEntityTypeIdentity(resource *loganalyticsv1beta1.LogAnalyticsEntityType) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("LogAnalyticsEntityType resource is nil")
	}
	entityTypeName := logAnalyticsEntityTypeTrackedName(resource)
	if entityTypeName == "" {
		entityTypeName = strings.TrimSpace(resource.Spec.Name)
	}
	return logAnalyticsEntityTypeIdentity{EntityTypeName: entityTypeName}, nil
}

func guardLogAnalyticsEntityTypeExistingBeforeCreate(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("LogAnalyticsEntityType resource is nil")
	}
	if strings.TrimSpace(resource.Spec.Name) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupLogAnalyticsEntityTypeExisting(
	listLogAnalyticsEntityTypes func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error),
) func(context.Context, *loganalyticsv1beta1.LogAnalyticsEntityType, any) (any, error) {
	if listLogAnalyticsEntityTypes == nil {
		return nil
	}
	return func(
		ctx context.Context,
		resource *loganalyticsv1beta1.LogAnalyticsEntityType,
		identity any,
	) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("LogAnalyticsEntityType resource is nil")
		}
		name := strings.TrimSpace(resource.Spec.Name)
		if name == "" {
			name = logAnalyticsEntityTypeIdentityName(identity)
		}
		if name == "" {
			return nil, nil
		}
		summary, found, err := listLogAnalyticsEntityTypeByName(ctx, listLogAnalyticsEntityTypes, name)
		if err != nil || !found {
			return nil, err
		}
		recordLogAnalyticsEntityTypeSummary(resource, summary)
		return logAnalyticsEntityTypeIdentifiedFromSummary(summary), nil
	}
}

func recordLogAnalyticsEntityTypeTrackedIdentity(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	identity any,
	resourceID string,
) {
	if resource == nil {
		return
	}
	trackedName := strings.TrimSpace(resourceID)
	if trackedName == "" {
		trackedName = logAnalyticsEntityTypeStatusInternalName(resource)
	}
	if trackedName == "" {
		trackedName = logAnalyticsEntityTypeIdentityName(identity)
	}
	if trackedName != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(trackedName)
	}
}

func logAnalyticsEntityTypeTrackedName(resource *loganalyticsv1beta1.LogAnalyticsEntityType) string {
	if resource == nil {
		return ""
	}
	if internalName := strings.TrimSpace(resource.Status.InternalName); internalName != "" {
		return internalName
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		if specName := strings.TrimSpace(resource.Spec.Name); specName != "" && ocid == specName {
			return ""
		}
		return ocid
	}
	return ""
}

func logAnalyticsEntityTypeStatusInternalName(resource *loganalyticsv1beta1.LogAnalyticsEntityType) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Status.InternalName)
}

func logAnalyticsEntityTypeIdentityName(identity any) string {
	switch value := identity.(type) {
	case logAnalyticsEntityTypeIdentity:
		return strings.TrimSpace(value.EntityTypeName)
	case *logAnalyticsEntityTypeIdentity:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(value.EntityTypeName)
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func wrapLogAnalyticsEntityTypeListPages(hooks *LogAnalyticsEntityTypeRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
		return listLogAnalyticsEntityTypePages(ctx, call, request)
	}
}

func listLogAnalyticsEntityTypePages(
	ctx context.Context,
	call func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error),
	request loganalyticssdk.ListLogAnalyticsEntityTypesRequest,
) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
	var combined loganalyticssdk.ListLogAnalyticsEntityTypesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func listLogAnalyticsEntityTypeByName(
	ctx context.Context,
	call func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error),
	name string,
) (loganalyticssdk.LogAnalyticsEntityTypeSummary, bool, error) {
	if call == nil {
		return loganalyticssdk.LogAnalyticsEntityTypeSummary{}, false, nil
	}
	response, err := call(ctx, loganalyticssdk.ListLogAnalyticsEntityTypesRequest{
		Name: logAnalyticsEntityTypeString(name),
	})
	if err != nil {
		return loganalyticssdk.LogAnalyticsEntityTypeSummary{}, false, err
	}
	return selectLogAnalyticsEntityTypeSummary(response.Items, name)
}

func selectLogAnalyticsEntityTypeSummary(
	items []loganalyticssdk.LogAnalyticsEntityTypeSummary,
	name string,
) (loganalyticssdk.LogAnalyticsEntityTypeSummary, bool, error) {
	var matched *loganalyticssdk.LogAnalyticsEntityTypeSummary
	for i := range items {
		if !stringsEqual(name, items[i].Name) {
			continue
		}
		if matched != nil {
			return loganalyticssdk.LogAnalyticsEntityTypeSummary{}, false, fmt.Errorf("LogAnalyticsEntityType list response returned multiple matches for name %q", name)
		}
		matched = &items[i]
	}
	if matched == nil {
		return loganalyticssdk.LogAnalyticsEntityTypeSummary{}, false, nil
	}
	return *matched, true, nil
}

func recordLogAnalyticsEntityTypeSummary(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	summary loganalyticssdk.LogAnalyticsEntityTypeSummary,
) {
	if resource == nil {
		return
	}
	if internalName := strings.TrimSpace(derefLogAnalyticsEntityTypeString(summary.InternalName)); internalName != "" {
		resource.Status.InternalName = internalName
		resource.Status.OsokStatus.Ocid = shared.OCID(internalName)
	}
}

func handleLogAnalyticsEntityTypeDeleteError(resource *loganalyticsv1beta1.LogAnalyticsEntityType, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return ambiguousLogAnalyticsEntityTypeError("delete", err)
	}
	return err
}

func logAnalyticsEntityTypeDeleteConfirmRead(
	getLogAnalyticsEntityType func(context.Context, loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error),
	listLogAnalyticsEntityTypes func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error),
) func(context.Context, *loganalyticsv1beta1.LogAnalyticsEntityType, string) (any, error) {
	if getLogAnalyticsEntityType == nil && listLogAnalyticsEntityTypes == nil {
		return nil
	}
	return func(
		ctx context.Context,
		resource *loganalyticsv1beta1.LogAnalyticsEntityType,
		currentID string,
	) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("LogAnalyticsEntityType resource is nil")
		}
		entityTypeName := strings.TrimSpace(currentID)
		if entityTypeName == "" {
			entityTypeName = logAnalyticsEntityTypeTrackedName(resource)
		}
		if entityTypeName == "" {
			return logAnalyticsEntityTypeDeleteConfirmList(ctx, resource, listLogAnalyticsEntityTypes)
		}
		if getLogAnalyticsEntityType == nil {
			return nil, fmt.Errorf("LogAnalyticsEntityType get operation is required for delete confirmation of %q", entityTypeName)
		}
		request := loganalyticssdk.GetLogAnalyticsEntityTypeRequest{
			EntityTypeName: logAnalyticsEntityTypeString(entityTypeName),
		}
		response, err := getLogAnalyticsEntityType(ctx, request)
		if err != nil {
			if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
				ambiguous := ambiguousLogAnalyticsEntityTypeError("delete confirmation", err)
				servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
				return nil, ambiguous
			}
			return nil, err
		}
		return response, nil
	}
}

func logAnalyticsEntityTypeDeleteConfirmList(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	listLogAnalyticsEntityTypes func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error),
) (any, error) {
	name := strings.TrimSpace(resource.Spec.Name)
	if name == "" {
		return nil, fmt.Errorf("LogAnalyticsEntityType spec.name is required for untracked delete confirmation")
	}
	summary, found, err := listLogAnalyticsEntityTypeByName(ctx, listLogAnalyticsEntityTypes, name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errorutil.NotFoundOciError{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			Description:    fmt.Sprintf("LogAnalyticsEntityType %q no longer exists", name),
		}
	}
	recordLogAnalyticsEntityTypeSummary(resource, summary)
	return logAnalyticsEntityTypeIdentifiedFromSummary(summary), nil
}

type logAnalyticsEntityTypeDeleteListAbsenceClient struct {
	delegate LogAnalyticsEntityTypeServiceClient
	list     func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error)
	log      loggerutil.OSOKLogger
}

func wrapLogAnalyticsEntityTypeDeleteListAbsence(
	listLogAnalyticsEntityTypes func(context.Context, loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error),
	log loggerutil.OSOKLogger,
) func(LogAnalyticsEntityTypeServiceClient) LogAnalyticsEntityTypeServiceClient {
	return func(delegate LogAnalyticsEntityTypeServiceClient) LogAnalyticsEntityTypeServiceClient {
		return logAnalyticsEntityTypeDeleteListAbsenceClient{
			delegate: delegate,
			list:     listLogAnalyticsEntityTypes,
			log:      log,
		}
	}
}

func (c logAnalyticsEntityTypeDeleteListAbsenceClient) CreateOrUpdate(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	request ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, request)
}

func (c logAnalyticsEntityTypeDeleteListAbsenceClient) Delete(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
) (bool, error) {
	if resource == nil || logAnalyticsEntityTypeTrackedName(resource) != "" || c.list == nil {
		return c.delegate.Delete(ctx, resource)
	}

	name := strings.TrimSpace(resource.Spec.Name)
	if name == "" {
		return c.delegate.Delete(ctx, resource)
	}
	summary, found, err := listLogAnalyticsEntityTypeByName(ctx, c.list, name)
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			ambiguous := ambiguousLogAnalyticsEntityTypeError("delete confirmation", err)
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
			return false, ambiguous
		}
		return false, err
	}
	if !found {
		markLogAnalyticsEntityTypeDeleted(resource, fmt.Sprintf("LogAnalyticsEntityType %q no longer exists", name), c.log)
		return true, nil
	}
	recordLogAnalyticsEntityTypeSummary(resource, summary)
	return c.delegate.Delete(ctx, resource)
}

func applyLogAnalyticsEntityTypeDeleteOutcome(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	response any,
	stage generatedruntime.DeleteConfirmStage,
	log loggerutil.OSOKLogger,
) (generatedruntime.DeleteOutcome, error) {
	if resource == nil {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if !strings.EqualFold(logAnalyticsEntityTypeLifecycleState(response), string(loganalyticssdk.EntityLifecycleStatesActive)) {
		return generatedruntime.DeleteOutcome{}, nil
	}

	switch stage {
	case generatedruntime.DeleteConfirmStageAfterRequest:
		markLogAnalyticsEntityTypeTerminating(resource, "OCI LogAnalyticsEntityType delete is in progress", log)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	case generatedruntime.DeleteConfirmStageAlreadyPending:
		if logAnalyticsEntityTypeDeleteAlreadyPending(resource) {
			markLogAnalyticsEntityTypeTerminating(resource, "OCI LogAnalyticsEntityType delete is still in progress", log)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func logAnalyticsEntityTypeDeleteAlreadyPending(resource *loganalyticsv1beta1.LogAnalyticsEntityType) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Phase == shared.OSOKAsyncPhaseDelete && current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markLogAnalyticsEntityTypeTerminating(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	message string,
	log loggerutil.OSOKLogger,
) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, log)
}

func markLogAnalyticsEntityTypeDeleted(
	resource *loganalyticsv1beta1.LogAnalyticsEntityType,
	message string,
	log loggerutil.OSOKLogger,
) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, log)
}

func ambiguousLogAnalyticsEntityTypeError(operation string, err error) error {
	return ambiguousLogAnalyticsEntityTypeNotFoundError{
		message:      fmt.Sprintf("LogAnalyticsEntityType %s returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", operation, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func logAnalyticsEntityTypeFromResponse(response any) (loganalyticssdk.LogAnalyticsEntityType, bool, bool) {
	if entityType, hasProperties, ok := logAnalyticsEntityTypeFromBody(response); ok {
		return entityType, hasProperties, true
	}
	return logAnalyticsEntityTypeFromGetResponse(response)
}

func logAnalyticsEntityTypeFromBody(response any) (loganalyticssdk.LogAnalyticsEntityType, bool, bool) {
	switch current := response.(type) {
	case loganalyticssdk.LogAnalyticsEntityType:
		return current, true, true
	case *loganalyticssdk.LogAnalyticsEntityType:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntityType{}, false, false
		}
		return *current, true, true
	case loganalyticssdk.LogAnalyticsEntityTypeSummary:
		return logAnalyticsEntityTypeFromSummary(current), false, true
	case *loganalyticssdk.LogAnalyticsEntityTypeSummary:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntityType{}, false, false
		}
		return logAnalyticsEntityTypeFromSummary(*current), false, true
	case logAnalyticsEntityTypeIdentified:
		return logAnalyticsEntityTypeFromIdentified(current), current.Properties != nil, true
	case *logAnalyticsEntityTypeIdentified:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntityType{}, false, false
		}
		return logAnalyticsEntityTypeFromIdentified(*current), current.Properties != nil, true
	default:
		return loganalyticssdk.LogAnalyticsEntityType{}, false, false
	}
}

func logAnalyticsEntityTypeFromGetResponse(response any) (loganalyticssdk.LogAnalyticsEntityType, bool, bool) {
	switch current := response.(type) {
	case loganalyticssdk.GetLogAnalyticsEntityTypeResponse:
		return current.LogAnalyticsEntityType, true, true
	case *loganalyticssdk.GetLogAnalyticsEntityTypeResponse:
		if current == nil {
			return loganalyticssdk.LogAnalyticsEntityType{}, false, false
		}
		return current.LogAnalyticsEntityType, true, true
	default:
		return loganalyticssdk.LogAnalyticsEntityType{}, false, false
	}
}

func logAnalyticsEntityTypeIdentifiedFromSummary(summary loganalyticssdk.LogAnalyticsEntityTypeSummary) logAnalyticsEntityTypeIdentified {
	return logAnalyticsEntityTypeIdentified{
		ID:                               strings.TrimSpace(derefLogAnalyticsEntityTypeString(summary.InternalName)),
		Name:                             summary.Name,
		InternalName:                     summary.InternalName,
		Category:                         summary.Category,
		CloudType:                        summary.CloudType,
		LifecycleState:                   summary.LifecycleState,
		ManagementAgentEligibilityStatus: string(summary.ManagementAgentEligibilityStatus),
	}
}

func logAnalyticsEntityTypeFromIdentified(identified logAnalyticsEntityTypeIdentified) loganalyticssdk.LogAnalyticsEntityType {
	return loganalyticssdk.LogAnalyticsEntityType{
		Name:                             identified.Name,
		InternalName:                     identified.InternalName,
		Category:                         identified.Category,
		CloudType:                        identified.CloudType,
		LifecycleState:                   identified.LifecycleState,
		Properties:                       identified.Properties,
		ManagementAgentEligibilityStatus: loganalyticssdk.LogAnalyticsEntityTypeManagementAgentEligibilityStatusEnum(identified.ManagementAgentEligibilityStatus),
	}
}

func logAnalyticsEntityTypeFromSummary(summary loganalyticssdk.LogAnalyticsEntityTypeSummary) loganalyticssdk.LogAnalyticsEntityType {
	return loganalyticssdk.LogAnalyticsEntityType{
		Name:                             summary.Name,
		InternalName:                     summary.InternalName,
		Category:                         summary.Category,
		CloudType:                        summary.CloudType,
		LifecycleState:                   summary.LifecycleState,
		TimeCreated:                      summary.TimeCreated,
		TimeUpdated:                      summary.TimeUpdated,
		ManagementAgentEligibilityStatus: loganalyticssdk.LogAnalyticsEntityTypeManagementAgentEligibilityStatusEnum(summary.ManagementAgentEligibilityStatus),
	}
}

func logAnalyticsEntityTypeLifecycleState(response any) string {
	entityType, _, ok := logAnalyticsEntityTypeFromResponse(response)
	if !ok {
		return ""
	}
	return strings.TrimSpace(string(entityType.LifecycleState))
}

func logAnalyticsEntityTypePropertiesFromSpec(properties []loganalyticsv1beta1.LogAnalyticsEntityTypeProperty) []loganalyticssdk.EntityTypeProperty {
	if properties == nil {
		return nil
	}
	converted := make([]loganalyticssdk.EntityTypeProperty, 0, len(properties))
	for _, property := range properties {
		converted = append(converted, loganalyticssdk.EntityTypeProperty{
			Name:        logAnalyticsEntityTypeString(property.Name),
			Description: optionalLogAnalyticsEntityTypeString(property.Description),
		})
	}
	return converted
}

func logAnalyticsEntityTypeString(value string) *string {
	return &value
}

func optionalLogAnalyticsEntityTypeString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return logAnalyticsEntityTypeString(value)
}

func stringsEqual(desired string, current *string) bool {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	return desired == currentValue
}

func derefLogAnalyticsEntityTypeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newLogAnalyticsEntityTypeServiceClientWithOCIClientAndNamespaceGetter(
	log loggerutil.OSOKLogger,
	client logAnalyticsEntityTypeOCIClient,
	namespaceGetter logAnalyticsEntityTypeNamespaceGetter,
) LogAnalyticsEntityTypeServiceClient {
	hooks := newLogAnalyticsEntityTypeRuntimeHooksWithOCIClient(client)
	applyLogAnalyticsEntityTypeRuntimeHooksWithNamespaceResolver(
		&LogAnalyticsEntityTypeServiceManager{Log: log},
		&hooks,
		newLogAnalyticsEntityTypeNamespaceResolverWithGetter(namespaceGetter),
	)
	delegate := defaultLogAnalyticsEntityTypeServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.LogAnalyticsEntityType](
			buildLogAnalyticsEntityTypeGeneratedRuntimeConfig(&LogAnalyticsEntityTypeServiceManager{Log: log}, hooks),
		),
	}
	return wrapLogAnalyticsEntityTypeGeneratedClient(hooks, delegate)
}

func newLogAnalyticsEntityTypeRuntimeHooksWithOCIClient(client logAnalyticsEntityTypeOCIClient) LogAnalyticsEntityTypeRuntimeHooks {
	return LogAnalyticsEntityTypeRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*loganalyticsv1beta1.LogAnalyticsEntityType]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*loganalyticsv1beta1.LogAnalyticsEntityType]{},
		StatusHooks:     generatedruntime.StatusHooks[*loganalyticsv1beta1.LogAnalyticsEntityType]{},
		ParityHooks:     generatedruntime.ParityHooks[*loganalyticsv1beta1.LogAnalyticsEntityType]{},
		Async:           generatedruntime.AsyncHooks[*loganalyticsv1beta1.LogAnalyticsEntityType]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*loganalyticsv1beta1.LogAnalyticsEntityType]{},
		Create: runtimeOperationHooks[loganalyticssdk.CreateLogAnalyticsEntityTypeRequest, loganalyticssdk.CreateLogAnalyticsEntityTypeResponse]{
			Fields: logAnalyticsEntityTypeCreateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.CreateLogAnalyticsEntityTypeRequest) (loganalyticssdk.CreateLogAnalyticsEntityTypeResponse, error) {
				if client == nil {
					return loganalyticssdk.CreateLogAnalyticsEntityTypeResponse{}, fmt.Errorf("LogAnalyticsEntityType OCI client is nil")
				}
				return client.CreateLogAnalyticsEntityType(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loganalyticssdk.GetLogAnalyticsEntityTypeRequest, loganalyticssdk.GetLogAnalyticsEntityTypeResponse]{
			Fields: logAnalyticsEntityTypeGetFields(),
			Call: func(ctx context.Context, request loganalyticssdk.GetLogAnalyticsEntityTypeRequest) (loganalyticssdk.GetLogAnalyticsEntityTypeResponse, error) {
				if client == nil {
					return loganalyticssdk.GetLogAnalyticsEntityTypeResponse{}, fmt.Errorf("LogAnalyticsEntityType OCI client is nil")
				}
				return client.GetLogAnalyticsEntityType(ctx, request)
			},
		},
		List: runtimeOperationHooks[loganalyticssdk.ListLogAnalyticsEntityTypesRequest, loganalyticssdk.ListLogAnalyticsEntityTypesResponse]{
			Fields: logAnalyticsEntityTypeListFields(),
			Call: func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsEntityTypesRequest) (loganalyticssdk.ListLogAnalyticsEntityTypesResponse, error) {
				if client == nil {
					return loganalyticssdk.ListLogAnalyticsEntityTypesResponse{}, fmt.Errorf("LogAnalyticsEntityType OCI client is nil")
				}
				return client.ListLogAnalyticsEntityTypes(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest, loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse]{
			Fields: logAnalyticsEntityTypeUpdateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.UpdateLogAnalyticsEntityTypeRequest) (loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse, error) {
				if client == nil {
					return loganalyticssdk.UpdateLogAnalyticsEntityTypeResponse{}, fmt.Errorf("LogAnalyticsEntityType OCI client is nil")
				}
				return client.UpdateLogAnalyticsEntityType(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest, loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse]{
			Fields: logAnalyticsEntityTypeDeleteFields(),
			Call: func(ctx context.Context, request loganalyticssdk.DeleteLogAnalyticsEntityTypeRequest) (loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse, error) {
				if client == nil {
					return loganalyticssdk.DeleteLogAnalyticsEntityTypeResponse{}, fmt.Errorf("LogAnalyticsEntityType OCI client is nil")
				}
				return client.DeleteLogAnalyticsEntityType(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LogAnalyticsEntityTypeServiceClient) LogAnalyticsEntityTypeServiceClient{},
	}
}

var _ interface{ GetOpcRequestID() string } = ambiguousLogAnalyticsEntityTypeNotFoundError{}
