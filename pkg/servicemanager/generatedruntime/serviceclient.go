/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const defaultRequeueDuration = time.Minute

var errResourceNotFound = errors.New("generated runtime resource not found")

var (
	passwordSourceType = reflect.TypeOf(shared.PasswordSource{})
	usernameSourceType = reflect.TypeOf(shared.UsernameSource{})
)

type Operation struct {
	NewRequest func() any
	Call       func(context.Context, any) (any, error)
	Fields     []RequestField
}

type RequestField struct {
	FieldName        string
	RequestName      string
	Contribution     string
	PreferResourceID bool
}

type Hook struct {
	Helper     string
	EntityType string
	Action     string
}

type FollowUpSemantics struct {
	Strategy string
	Hooks    []Hook
}

type HookSet struct {
	Create []Hook
	Update []Hook
	Delete []Hook
}

type LifecycleSemantics struct {
	ProvisioningStates []string
	UpdatingStates     []string
	ActiveStates       []string
}

type DeleteSemantics struct {
	Policy         string
	PendingStates  []string
	TerminalStates []string
}

type ListSemantics struct {
	ResponseItemsField string
	MatchFields        []string
}

type MutationSemantics struct {
	Mutable       []string
	ForceNew      []string
	ConflictsWith map[string][]string
}

type AuxiliaryOperation struct {
	Phase            string
	MethodName       string
	RequestTypeName  string
	ResponseTypeName string
}

type UnsupportedSemantic struct {
	Category      string
	StopCondition string
}

type Semantics struct {
	FormalService       string
	FormalSlug          string
	StatusProjection    string
	SecretSideEffects   string
	FinalizerPolicy     string
	Lifecycle           LifecycleSemantics
	Delete              DeleteSemantics
	List                *ListSemantics
	Mutation            MutationSemantics
	Hooks               HookSet
	CreateFollowUp      FollowUpSemantics
	UpdateFollowUp      FollowUpSemantics
	DeleteFollowUp      FollowUpSemantics
	AuxiliaryOperations []AuxiliaryOperation
	Unsupported         []UnsupportedSemantic
}

type Config[T any] struct {
	Kind             string
	SDKName          string
	Log              loggerutil.OSOKLogger
	CredentialClient credhelper.CredentialClient
	InitError        error
	Semantics        *Semantics

	Create *Operation
	Get    *Operation
	List   *Operation
	Update *Operation
	Delete *Operation
}

type ServiceClient[T any] struct {
	config Config[T]
}

func NewServiceClient[T any](cfg Config[T]) ServiceClient[T] {
	if err := validateFormalSemantics(cfg.Kind, cfg.Semantics); err != nil {
		cfg.InitError = errors.Join(cfg.InitError, err)
	}
	return ServiceClient[T]{config: cfg}
}

func failedResponse(err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c ServiceClient[T]) CreateOrUpdate(ctx context.Context, resource T, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	namespace := resourceNamespace(resource, req.Namespace)
	if response, err := c.preflightCreateOrUpdate(resource); err != nil {
		return response, err
	}
	currentID := c.currentID(resource)
	if err := c.validateMutationPolicy(resource, currentID != ""); err != nil {
		return c.markedFailureResponse(resource, err)
	}
	if currentID != "" {
		return c.updateExistingResource(ctx, resource, currentID, namespace)
	}
	return c.createNewResource(ctx, resource, namespace)
}

func (c ServiceClient[T]) Delete(ctx context.Context, resource T) (bool, error) {
	if err := c.preflightDelete(resource); err != nil {
		return false, err
	}
	if c.config.Semantics != nil {
		return c.deleteWithSemantics(ctx, resource)
	}
	return c.deleteWithoutSemantics(ctx, resource)
}

func (c ServiceClient[T]) preflightCreateOrUpdate(resource T) (servicemanager.OSOKResponse, error) {
	if c.config.InitError != nil {
		return c.markedFailureResponse(resource, c.config.InitError)
	}
	if _, err := resourceStruct(resource); err != nil {
		return failedResponse(err)
	}
	return servicemanager.OSOKResponse{}, nil
}

func (c ServiceClient[T]) markedFailureResponse(resource T, err error) (servicemanager.OSOKResponse, error) {
	return failedResponse(c.markFailure(resource, err))
}

func (c ServiceClient[T]) requestBuildOptions(ctx context.Context, namespace string) requestBuildOptions {
	return requestBuildOptions{
		Context:          ctx,
		CredentialClient: c.config.CredentialClient,
		Namespace:        namespace,
	}
}

func (c ServiceClient[T]) updateExistingResource(ctx context.Context, resource T, currentID string, namespace string) (servicemanager.OSOKResponse, error) {
	if c.config.Update == nil {
		return c.readExistingResource(ctx, resource, currentID)
	}
	response, err := c.invoke(ctx, c.config.Update, resource, currentID, c.requestBuildOptions(ctx, namespace))
	if err != nil {
		return c.markedFailureResponse(resource, err)
	}
	response, err = c.followUpAfterWrite(ctx, resource, currentID, response, "update")
	if err != nil {
		return c.markedFailureResponse(resource, err)
	}
	return c.applySuccess(resource, response, shared.Updating)
}

func (c ServiceClient[T]) createNewResource(ctx context.Context, resource T, namespace string) (servicemanager.OSOKResponse, error) {
	if c.config.Create == nil {
		return c.readExistingResource(ctx, resource, "")
	}
	response, err := c.invoke(ctx, c.config.Create, resource, "", c.requestBuildOptions(ctx, namespace))
	if err != nil {
		return c.markedFailureResponse(resource, err)
	}
	followUp, err := c.followUpAfterWrite(ctx, resource, responseID(response), response, "create")
	if err != nil {
		return c.markedFailureResponse(resource, err)
	}
	return c.applySuccess(resource, followUp, shared.Provisioning)
}

func (c ServiceClient[T]) readExistingResource(ctx context.Context, resource T, currentID string) (servicemanager.OSOKResponse, error) {
	response, err := c.readResource(ctx, resource, currentID)
	if err != nil {
		return c.markedFailureResponse(resource, err)
	}
	return c.applySuccess(resource, response, shared.Active)
}

func (c ServiceClient[T]) preflightDelete(resource T) error {
	if c.config.InitError != nil {
		return c.config.InitError
	}
	_, err := resourceStruct(resource)
	return err
}

func (c ServiceClient[T]) deleteWithoutSemantics(ctx context.Context, resource T) (bool, error) {
	if c.config.Delete == nil {
		c.markDeleted(resource, "OCI delete is not supported for this generated resource")
		return true, nil
	}
	currentID, deleted := c.deleteIDOrMarkMissing(resource)
	if deleted {
		return true, nil
	}
	if deleted, err := c.invokeDelete(ctx, resource, currentID); deleted || err != nil {
		return deleted, err
	}
	return c.confirmDeleteProgress(ctx, resource, currentID)
}

func (c ServiceClient[T]) deleteIDOrMarkMissing(resource T) (string, bool) {
	currentID := c.currentID(resource)
	if currentID != "" {
		return currentID, false
	}
	c.markDeleted(resource, "OCI resource identifier is not recorded")
	return "", true
}

func (c ServiceClient[T]) invokeDelete(ctx context.Context, resource T, currentID string) (bool, error) {
	if _, err := c.invoke(ctx, c.config.Delete, resource, currentID, requestBuildOptions{}); err != nil {
		if isNotFound(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (c ServiceClient[T]) confirmDeleteProgress(ctx context.Context, resource T, currentID string) (bool, error) {
	if !c.canReadResource() {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}
	response, deleted, err := c.readDeleteConfirmation(ctx, resource, currentID)
	if deleted || err != nil {
		return deleted, err
	}
	_ = mergeResponseIntoStatus(resource, response)
	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return false, nil
}

func (c ServiceClient[T]) followUpAfterWrite(ctx context.Context, resource T, preferredID string, response any, phase string) (any, error) {
	if !c.requiresWriteFollowUp(phase) {
		return response, nil
	}
	if c.config.Get == nil && c.config.List == nil {
		if c.config.Semantics != nil {
			return nil, fmt.Errorf("%s formal semantics require %s follow-up without a readable OCI operation", c.config.Kind, phase)
		}
		return response, nil
	}

	refreshed, err := c.readResource(ctx, resource, preferredID)
	if err == nil {
		return refreshed, nil
	}
	if phase == "create" && errors.Is(err, errResourceNotFound) {
		return response, nil
	}
	return nil, err
}

func (c ServiceClient[T]) requiresWriteFollowUp(phase string) bool {
	if c.config.Semantics == nil {
		return c.config.Get != nil || c.config.List != nil
	}

	switch phase {
	case "create":
		return c.config.Semantics.CreateFollowUp.Strategy == "read-after-write"
	case "update":
		return c.config.Semantics.UpdateFollowUp.Strategy == "read-after-write"
	default:
		return false
	}
}

func (c ServiceClient[T]) deleteWithSemantics(ctx context.Context, resource T) (bool, error) {
	semantics, err := c.formalDeleteSemantics()
	if err != nil {
		return false, err
	}
	currentID, deleted, err := c.resolveDeleteIDOrMarkMissing(ctx, resource)
	if deleted || err != nil {
		return deleted, err
	}
	if deleted, err := c.invokeDelete(ctx, resource, currentID); deleted || err != nil {
		return deleted, err
	}
	return c.confirmDeleteWithSemantics(ctx, resource, currentID, semantics)
}

func (c ServiceClient[T]) resolveDeleteID(ctx context.Context, resource T) (string, error) {
	currentID := c.currentID(resource)
	if currentID != "" {
		return currentID, nil
	}

	if c.config.Get == nil && c.config.List == nil {
		return "", errResourceNotFound
	}

	response, err := c.readResource(ctx, resource, "")
	if err != nil {
		return "", err
	}
	currentID = responseID(response)
	if currentID == "" {
		return "", fmt.Errorf("%s delete confirmation could not resolve a resource OCID", c.config.Kind)
	}
	_ = mergeResponseIntoStatus(resource, response)
	return currentID, nil
}

func (c ServiceClient[T]) validateMutationPolicy(resource T, existing bool) error {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil
	}
	specValues, statusValues, err := mutationPolicyValues(resource)
	if err != nil {
		return err
	}
	if err := c.validateMutationConflicts(specValues); err != nil {
		return err
	}
	if !existing {
		return nil
	}
	return c.validateForceNewFields(specValues, statusValues)
}

func (c ServiceClient[T]) formalDeleteSemantics() (*Semantics, error) {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil, fmt.Errorf("%s formal semantics are not configured", c.config.Kind)
	}
	if c.config.Delete == nil || semantics.Delete.Policy == "not-supported" {
		return nil, fmt.Errorf("%s formal semantics mark delete confirmation as %q", c.config.Kind, semantics.Delete.Policy)
	}
	return semantics, nil
}

func (c ServiceClient[T]) resolveDeleteIDOrMarkMissing(ctx context.Context, resource T) (string, bool, error) {
	currentID, err := c.resolveDeleteID(ctx, resource)
	if err != nil {
		if errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return "", true, nil
		}
		return "", false, err
	}
	return currentID, false, nil
}

func (c ServiceClient[T]) confirmDeleteWithSemantics(ctx context.Context, resource T, currentID string, semantics *Semantics) (bool, error) {
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}
	if !c.canReadResource() {
		return false, fmt.Errorf("%s formal delete confirmation requires a readable OCI operation", c.config.Kind)
	}
	response, deleted, err := c.readDeleteConfirmation(ctx, resource, currentID)
	if deleted || err != nil {
		return deleted, err
	}
	_ = mergeResponseIntoStatus(resource, response)
	return c.evaluateDeleteLifecycle(resource, semantics, responseLifecycleState(response))
}

func (c ServiceClient[T]) canReadResource() bool {
	return c.config.Get != nil || c.config.List != nil
}

func (c ServiceClient[T]) readDeleteConfirmation(ctx context.Context, resource T, currentID string) (any, bool, error) {
	response, err := c.readResource(ctx, resource, currentID)
	if err != nil {
		if isNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource deleted")
			return nil, true, nil
		}
		return nil, false, err
	}
	return response, false, nil
}

func (c ServiceClient[T]) evaluateDeleteLifecycle(resource T, semantics *Semantics, lifecycleState string) (bool, error) {
	lifecycleState = strings.ToUpper(lifecycleState)
	switch semantics.Delete.Policy {
	case "best-effort":
		return c.evaluateBestEffortDelete(resource, semantics, lifecycleState)
	case "required":
		return c.evaluateRequiredDelete(resource, semantics, lifecycleState)
	default:
		return false, fmt.Errorf("%s formal delete confirmation policy %q is not supported", c.config.Kind, semantics.Delete.Policy)
	}
}

func (c ServiceClient[T]) evaluateBestEffortDelete(resource T, semantics *Semantics, lifecycleState string) (bool, error) {
	if lifecycleState == "" ||
		containsString(semantics.Delete.PendingStates, lifecycleState) ||
		containsString(semantics.Delete.TerminalStates, lifecycleState) {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}
	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return false, nil
}

func (c ServiceClient[T]) evaluateRequiredDelete(resource T, semantics *Semantics, lifecycleState string) (bool, error) {
	switch {
	case containsString(semantics.Delete.TerminalStates, lifecycleState):
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	case lifecycleState == "" || containsString(semantics.Delete.PendingStates, lifecycleState):
		c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
		return false, nil
	default:
		return false, fmt.Errorf("%s delete confirmation returned unexpected lifecycle state %q", c.config.Kind, lifecycleState)
	}
}

func mutationPolicyValues(resource any) (map[string]any, map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, nil, err
	}
	return jsonMap(fieldInterface(resourceValue, "Spec")), jsonMap(fieldInterface(resourceValue, "Status")), nil
}

func (c ServiceClient[T]) validateMutationConflicts(specValues map[string]any) error {
	for field, conflicts := range c.config.Semantics.Mutation.ConflictsWith {
		if _, ok := lookupMeaningfulValue(specValues, field); !ok {
			continue
		}
		if conflict := firstPresentConflict(specValues, conflicts); conflict != "" {
			return fmt.Errorf("%s formal semantics forbid setting %s with %s", c.config.Kind, field, conflict)
		}
	}
	return nil
}

func firstPresentConflict(specValues map[string]any, conflicts []string) string {
	for _, conflict := range conflicts {
		if _, ok := lookupMeaningfulValue(specValues, conflict); ok {
			return conflict
		}
	}
	return ""
}

func (c ServiceClient[T]) validateForceNewFields(specValues map[string]any, statusValues map[string]any) error {
	for _, field := range c.config.Semantics.Mutation.ForceNew {
		specValue, specOK := lookupValueByPath(specValues, field)
		statusValue, statusOK := lookupValueByPath(statusValues, field)
		if !specOK || !statusOK {
			continue
		}
		if !valuesEqual(specValue, statusValue) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", c.config.Kind, field)
		}
	}
	return nil
}

func (c ServiceClient[T]) readResource(ctx context.Context, resource T, preferredID string) (any, error) {
	if c.config.Get != nil {
		response, err := c.invoke(ctx, c.config.Get, resource, preferredID, requestBuildOptions{})
		if err == nil {
			return response, nil
		}
		if !isNotFound(err) || c.config.List == nil {
			return nil, err
		}
	}

	if c.config.List == nil {
		return nil, fmt.Errorf("%s generated runtime has no readable OCI operation", c.config.Kind)
	}

	response, err := c.invoke(ctx, c.config.List, resource, preferredID, requestBuildOptions{})
	if err != nil {
		return nil, err
	}

	body, ok := responseBody(response)
	if !ok {
		return nil, fmt.Errorf("%s list response did not expose a body payload", c.config.Kind)
	}

	item, err := c.selectListItem(body, resource, preferredID)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (c ServiceClient[T]) invoke(ctx context.Context, op *Operation, resource T, preferredID string, options requestBuildOptions) (any, error) {
	if op == nil {
		return nil, fmt.Errorf("%s generated runtime does not define this OCI operation", c.config.Kind)
	}
	if op.NewRequest == nil || op.Call == nil {
		return nil, fmt.Errorf("%s generated runtime OCI operation is incomplete", c.config.Kind)
	}

	request := op.NewRequest()
	if request == nil {
		return nil, fmt.Errorf("%s generated runtime did not create an OCI request value", c.config.Kind)
	}
	if err := buildRequest(request, resource, preferredID, op.Fields, c.idFieldAliases(), options); err != nil {
		return nil, fmt.Errorf("build %s OCI request: %w", c.config.Kind, err)
	}

	response, err := op.Call(ctx, request)
	if err != nil {
		return nil, normalizeOCIError(err)
	}
	return response, nil
}

func (c ServiceClient[T]) applySuccess(resource T, response any, fallback shared.OSOKConditionType) (servicemanager.OSOKResponse, error) {
	if err := mergeResponseIntoStatus(resource, response); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	status, err := osokStatus(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	resourceID := responseID(response)
	if resourceID == "" {
		resourceID = c.currentID(resource)
	}
	if resourceID != "" {
		status.Ocid = shared.OCID(resourceID)
	}

	conditionType, shouldRequeue, message := c.classifyLifecycle(response, fallback)
	status.Message = message
	status.Reason = string(conditionType)
	now := metav1.Now()
	if resourceID != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, conditionType, v1.ConditionTrue, "", message, c.config.Log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    conditionType != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: defaultRequeueDuration,
	}, nil
}

func (c ServiceClient[T]) markFailure(resource T, err error) error {
	status, statusErr := osokStatus(resource)
	if statusErr != nil {
		return err
	}
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.config.Log)
	return err
}

func (c ServiceClient[T]) markDeleted(resource T, message string) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	if message != "" {
		status.Message = message
	}
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.config.Log)
}

func (c ServiceClient[T]) markCondition(resource T, condition shared.OSOKConditionType, message string) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, v1.ConditionTrue, "", message, c.config.Log)
}

func (c ServiceClient[T]) currentID(resource T) string {
	status, err := osokStatus(resource)
	if err == nil && status.Ocid != "" {
		return string(status.Ocid)
	}

	values, err := lookupValues(resource)
	if err != nil {
		return ""
	}
	return firstNonEmpty(values, "id", "ocid")
}

func (c ServiceClient[T]) idFieldAliases() []string {
	aliases := []string{"id", "ocid"}
	for _, name := range []string{c.config.Kind, c.config.SDKName} {
		if strings.TrimSpace(name) == "" {
			continue
		}
		aliases = appendUniqueStrings(aliases, lowerCamel(name)+"Id")
	}
	return aliases
}

type requestBuildOptions struct {
	Context          context.Context
	CredentialClient credhelper.CredentialClient
	Namespace        string
}

func buildRequest(request any, resource any, preferredID string, fields []RequestField, idAliases []string, options requestBuildOptions) error {
	requestValue := reflect.ValueOf(request)
	if !requestValue.IsValid() || requestValue.Kind() != reflect.Pointer || requestValue.IsNil() {
		return fmt.Errorf("expected pointer OCI request, got %T", request)
	}

	requestStruct := requestValue.Elem()
	if requestStruct.Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to OCI request struct, got %T", request)
	}

	values, err := lookupValues(resource)
	if err != nil {
		return err
	}

	var resolvedSpec any
	if requestNeedsResolvedSpec(fields, requestStruct.Type()) {
		resolvedSpec, err = resolvedSpecValue(resource, options)
		if err != nil {
			return err
		}
	}

	if len(fields) > 0 {
		return buildExplicitRequest(requestStruct, resource, values, preferredID, fields, resolvedSpec)
	}

	return buildHeuristicRequest(requestStruct, requestStruct.Type(), values, preferredID, idAliases, resolvedSpec)
}

func buildExplicitRequest(requestStruct reflect.Value, resource any, values map[string]any, preferredID string, fields []RequestField, resolvedSpec any) error {
	for _, field := range fields {
		fieldValue := requestStruct.FieldByName(field.FieldName)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			continue
		}

		switch field.Contribution {
		case "header", "binary":
			continue
		case "body":
			if err := assignField(fieldValue, resolvedSpec); err != nil {
				return fmt.Errorf("set body field %s: %w", field.FieldName, err)
			}
			continue
		}

		rawValue, ok := explicitRequestValue(values, field, preferredID)
		if !ok {
			continue
		}
		if err := assignField(fieldValue, rawValue); err != nil {
			return fmt.Errorf("set request field %s: %w", field.FieldName, err)
		}
	}

	return nil
}

func buildHeuristicRequest(
	requestStruct reflect.Value,
	requestType reflect.Type,
	values map[string]any,
	preferredID string,
	idAliases []string,
	resolvedSpec any,
) error {
	for i := 0; i < requestStruct.NumField(); i++ {
		fieldValue := requestStruct.Field(i)
		fieldType := requestType.Field(i)
		if !shouldPopulateHeuristicField(fieldValue, fieldType) {
			continue
		}
		handled, err := assignHeuristicBodyField(fieldValue, fieldType, resolvedSpec)
		if err != nil {
			return err
		}
		if handled {
			continue
		}
		if err := assignHeuristicRequestField(fieldValue, fieldType, values, preferredID, idAliases); err != nil {
			return err
		}
	}

	return nil
}

func shouldPopulateHeuristicField(fieldValue reflect.Value, fieldType reflect.StructField) bool {
	if !fieldValue.CanSet() || fieldType.Name == "RequestMetadata" {
		return false
	}
	contribution := fieldType.Tag.Get("contributesTo")
	return contribution != "header" && contribution != "binary"
}

func assignHeuristicBodyField(fieldValue reflect.Value, fieldType reflect.StructField, resolvedSpec any) (bool, error) {
	if fieldType.Tag.Get("contributesTo") != "body" {
		return false, nil
	}
	if err := assignField(fieldValue, resolvedSpec); err != nil {
		return false, fmt.Errorf("set body field %s: %w", fieldType.Name, err)
	}
	return true, nil
}

func assignHeuristicRequestField(
	fieldValue reflect.Value,
	fieldType reflect.StructField,
	values map[string]any,
	preferredID string,
	idAliases []string,
) error {
	rawValue, ok := heuristicRequestValue(values, fieldType, preferredID, idAliases)
	if !ok {
		return nil
	}
	if err := assignField(fieldValue, rawValue); err != nil {
		return fmt.Errorf("set request field %s: %w", fieldType.Name, err)
	}
	return nil
}

func heuristicRequestValue(values map[string]any, fieldType reflect.StructField, preferredID string, idAliases []string) (any, bool) {
	lookupKey := heuristicLookupKey(fieldType)
	if rawValue, ok := values[lookupKey]; ok {
		return rawValue, true
	}
	return heuristicFallbackValue(values, lookupKey, preferredID, idAliases)
}

func heuristicLookupKey(fieldType reflect.StructField) string {
	if lookupKey := fieldType.Tag.Get("name"); lookupKey != "" {
		return lookupKey
	}
	if lookupKey := fieldJSONName(fieldType); lookupKey != "" {
		return lookupKey
	}
	return lowerCamel(fieldType.Name)
}

func heuristicFallbackValue(values map[string]any, lookupKey string, preferredID string, idAliases []string) (any, bool) {
	if preferredID != "" && containsString(idAliases, lookupKey) {
		return preferredID, true
	}
	switch lookupKey {
	case "name":
		return lookupValueByPaths(values, "metadataName")
	case "namespaceName":
		return lookupValueByPaths(values, "namespaceName", "namespace")
	default:
		return nil, false
	}
}

func explicitRequestValue(values map[string]any, field RequestField, preferredID string) (any, bool) {
	if field.PreferResourceID {
		if preferredID != "" {
			return preferredID, true
		}
		if currentID, ok := lookupValueByPaths(values, "id", "ocid"); ok {
			return currentID, true
		}
	}

	lookupKey := strings.TrimSpace(field.RequestName)
	if lookupKey == "" {
		lookupKey = lowerCamel(field.FieldName)
	}

	if rawValue, ok := lookupValueByPaths(values, lookupKey); ok {
		return rawValue, true
	}
	if lookupKey == "name" {
		return lookupValueByPaths(values, "metadataName")
	}
	if lookupKey == "namespaceName" {
		if value, ok := lookupValueByPaths(values, "namespaceName"); ok {
			return value, true
		}
		return lookupValueByPaths(values, "namespace")
	}

	return nil, false
}

func lookupValues(resource any) (map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, err
	}

	values := make(map[string]any)
	mergeJSONMap(values, fieldInterface(resourceValue, "Spec"))
	mergeJSONMap(values, fieldInterface(resourceValue, "Status"))
	if statusField, ok := fieldValue(resourceValue, "Status"); ok {
		mergeJSONMap(values, fieldInterface(statusField, "OsokStatus"))
	}

	if metadataName := lookupMetadataString(resourceValue, "Name"); metadataName != "" {
		if _, exists := values["name"]; !exists {
			values["name"] = metadataName
		}
		values["metadataName"] = metadataName
	}
	if namespaceName := lookupMetadataString(resourceValue, "Namespace"); namespaceName != "" {
		if _, exists := values["namespaceName"]; !exists {
			values["namespaceName"] = namespaceName
		}
		if _, exists := values["namespace"]; !exists {
			values["namespace"] = namespaceName
		}
	}

	return values, nil
}

func mergeJSONMap(dst map[string]any, source any) {
	if source == nil {
		return
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return
	}
	for key, value := range decoded {
		if _, exists := dst[key]; exists {
			continue
		}
		dst[key] = value
	}
}

func specValue(resource any) any {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil
	}
	return fieldInterface(resourceValue, "Spec")
}

func requestNeedsResolvedSpec(fields []RequestField, requestType reflect.Type) bool {
	if len(fields) > 0 {
		for _, field := range fields {
			if field.Contribution == "body" {
				return true
			}
		}
		return false
	}

	for i := 0; i < requestType.NumField(); i++ {
		if requestType.Field(i).Tag.Get("contributesTo") == "body" {
			return true
		}
	}
	return false
}

func resolvedSpecValue(resource any, options requestBuildOptions) (any, error) {
	raw := specValue(resource)
	if raw == nil {
		return nil, nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal spec value: %w", err)
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode spec value: %w", err)
	}

	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, err
	}
	specField, ok := fieldValue(resourceValue, "Spec")
	if !ok {
		return nil, fmt.Errorf("resource %T does not expose Spec", resource)
	}

	resolved, _, err := rewriteSecretSources(specField, decoded, options)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func indirectValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, value.IsValid()
}

func rewriteSecretSources(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	value, ok := indirectValue(value)
	if !ok {
		return nil, false, nil
	}
	if rewritten, include, handled, err := rewriteSharedSecretSource(value, options); handled {
		return rewritten, include, err
	}
	switch value.Kind() {
	case reflect.Struct:
		return rewriteSecretStruct(value, decoded, options)
	case reflect.Slice, reflect.Array:
		return rewriteSecretSlice(value, decoded, options)
	case reflect.Map:
		return rewriteSecretMap(value, decoded, options)
	default:
		return decoded, true, nil
	}
}

func rewriteSharedSecretSource(value reflect.Value, options requestBuildOptions) (any, bool, bool, error) {
	switch value.Type() {
	case passwordSourceType:
		rewritten, include, err := resolveSecretSourceValue(options.Context, options.CredentialClient, options.Namespace, value.FieldByName("Secret"), "SecretName", "password")
		return rewritten, include, true, err
	case usernameSourceType:
		rewritten, include, err := resolveSecretSourceValue(options.Context, options.CredentialClient, options.Namespace, value.FieldByName("Secret"), "SecretName", "username")
		return rewritten, include, true, err
	default:
		return nil, false, false, nil
	}
}

func rewriteSecretStruct(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	decodedMap := decodedMapValue(decoded)
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		if fieldType.Anonymous && embeddedJSONField(fieldType) {
			var err error
			decodedMap, err = rewriteEmbeddedSecretField(value.Field(i), decodedMap, options)
			if err != nil {
				return nil, false, err
			}
			continue
		}
		if err := rewriteNamedSecretField(decodedMap, value.Field(i), fieldType, options); err != nil {
			return nil, false, err
		}
	}
	return decodedMap, true, nil
}

func decodedMapValue(decoded any) map[string]any {
	decodedMap, ok := decoded.(map[string]any)
	if !ok || decodedMap == nil {
		return map[string]any{}
	}
	return decodedMap
}

func rewriteEmbeddedSecretField(fieldValue reflect.Value, decodedMap map[string]any, options requestBuildOptions) (map[string]any, error) {
	rewritten, _, err := rewriteSecretSources(fieldValue, decodedMap, options)
	if err != nil {
		return nil, err
	}
	if nestedMap, ok := rewritten.(map[string]any); ok {
		return nestedMap, nil
	}
	return decodedMap, nil
}

func rewriteNamedSecretField(decodedMap map[string]any, fieldValue reflect.Value, fieldType reflect.StructField, options requestBuildOptions) error {
	jsonName, skip := fieldJSONTagName(fieldType)
	if skip {
		return nil
	}
	childDecoded, exists := decodedMap[jsonName]
	rewritten, include, err := rewriteSecretSources(fieldValue, childDecoded, options)
	if err != nil {
		return err
	}
	if include {
		decodedMap[jsonName] = rewritten
		return nil
	}
	if exists {
		delete(decodedMap, jsonName)
	}
	return nil
}

func rewriteSecretSlice(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	decodedSlice, ok := decoded.([]any)
	if !ok {
		return decoded, true, nil
	}
	for i := 0; i < value.Len() && i < len(decodedSlice); i++ {
		rewritten, include, err := rewriteSecretSources(value.Index(i), decodedSlice[i], options)
		if err != nil {
			return nil, false, err
		}
		if include {
			decodedSlice[i] = rewritten
		}
	}
	return decodedSlice, true, nil
}

func rewriteSecretMap(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	if value.Type().Key().Kind() != reflect.String {
		return decoded, true, nil
	}
	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		return decoded, true, nil
	}
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		childDecoded, exists := decodedMap[key]
		rewritten, include, err := rewriteSecretSources(iter.Value(), childDecoded, options)
		if err != nil {
			return nil, false, err
		}
		if include {
			decodedMap[key] = rewritten
			continue
		}
		if exists {
			delete(decodedMap, key)
		}
	}
	return decodedMap, true, nil
}

func resolveSecretSourceValue(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	namespace string,
	secretField reflect.Value,
	nameField string,
	dataKey string,
) (any, bool, error) {
	if !secretField.IsValid() {
		return nil, false, nil
	}
	secretNameField := secretField.FieldByName(nameField)
	if !secretNameField.IsValid() || secretNameField.Kind() != reflect.String {
		return nil, false, nil
	}

	secretName := strings.TrimSpace(secretNameField.String())
	if secretName == "" {
		return nil, false, nil
	}
	if credentialClient == nil {
		return nil, false, fmt.Errorf("resolve %s secret %q: credential client is nil", dataKey, secretName)
	}
	if strings.TrimSpace(namespace) == "" {
		return nil, false, fmt.Errorf("resolve %s secret %q: namespace is empty", dataKey, secretName)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	secretData, err := credentialClient.GetSecret(ctx, secretName, namespace)
	if err != nil {
		return nil, false, fmt.Errorf("get %s secret %q: %w", dataKey, secretName, err)
	}
	rawValue, ok := secretData[dataKey]
	if !ok {
		return nil, false, fmt.Errorf("%s key in secret %q is not found", dataKey, secretName)
	}
	return string(rawValue), true, nil
}

func fieldJSONTagName(field reflect.StructField) (string, bool) {
	name := field.Tag.Get("json")
	if name == "-" {
		return "", true
	}
	if strings.TrimSpace(name) == "" {
		return lowerCamel(field.Name), false
	}
	parts := strings.Split(name, ",")
	if len(parts) == 0 {
		return lowerCamel(field.Name), false
	}
	if parts[0] == "" {
		return lowerCamel(field.Name), false
	}
	return parts[0], false
}

func embeddedJSONField(field reflect.StructField) bool {
	if !field.Anonymous {
		return false
	}
	parts := strings.Split(field.Tag.Get("json"), ",")
	return len(parts) == 0 || parts[0] == ""
}

func responseBody(response any) (any, bool) {
	if response == nil {
		return nil, false
	}
	value, ok := indirectValue(reflect.ValueOf(response))
	if !ok {
		return nil, false
	}
	if value.Kind() != reflect.Struct {
		return response, true
	}
	if !strings.HasSuffix(value.Type().Name(), "Response") {
		return value.Interface(), true
	}
	return responseStructBody(value)
}

func responseStructBody(value reflect.Value) (any, bool) {
	var fallback reflect.Value
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		fieldValue := value.Field(i)
		if fieldType.Tag.Get("presentIn") == "body" {
			return responseFieldValue(fieldValue)
		}
		if isResponseMetadataField(fieldType) {
			continue
		}
		if !fallback.IsValid() {
			fallback = fieldValue
		}
	}
	if !fallback.IsValid() {
		return nil, false
	}
	return fallback.Interface(), true
}

func responseFieldValue(fieldValue reflect.Value) (any, bool) {
	if fieldValue.Kind() != reflect.Pointer {
		return fieldValue.Interface(), true
	}
	if fieldValue.IsNil() {
		return nil, false
	}
	return fieldValue.Interface(), true
}

func isResponseMetadataField(fieldType reflect.StructField) bool {
	return fieldType.Name == "RawResponse" || strings.HasPrefix(fieldType.Name, "Opc") || fieldType.Name == "Etag"
}

func mergeResponseIntoStatus(resource any, response any) error {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return nil
	}

	statusValue, err := statusStruct(resource)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal OCI response body: %w", err)
	}
	if err := json.Unmarshal(payload, statusValue.Addr().Interface()); err != nil {
		return fmt.Errorf("project OCI response body into status: %w", err)
	}
	return nil
}

func responseID(response any) string {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return ""
	}
	values := jsonMap(body)
	return firstNonEmpty(values, "id", "ocid")
}

func (c ServiceClient[T]) classifyLifecycle(response any, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool, string) {
	if c.config.Semantics == nil {
		return classifyLifecycleHeuristics(response, fallback)
	}
	return classifyLifecycleSemantics(response, fallback, c.config.Semantics)
}

func classifyLifecycleSemantics(response any, fallback shared.OSOKConditionType, semantics *Semantics) (shared.OSOKConditionType, bool, string) {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return fallback, shouldRequeueForCondition(fallback), defaultConditionMessage(fallback)
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return fallback, shouldRequeueForCondition(fallback), message
	case containsString(semantics.Lifecycle.ProvisioningStates, lifecycleState):
		return shared.Provisioning, true, message
	case containsString(semantics.Lifecycle.UpdatingStates, lifecycleState):
		return shared.Updating, true, message
	case containsString(semantics.Delete.PendingStates, lifecycleState):
		return shared.Terminating, true, message
	case containsString(semantics.Lifecycle.ActiveStates, lifecycleState),
		containsString(semantics.Delete.TerminalStates, lifecycleState):
		return shared.Active, false, message
	default:
		return shared.Failed, false, fmt.Sprintf("formal lifecycle state %q is not modeled: %s", lifecycleState, message)
	}
}

func classifyLifecycleHeuristics(response any, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool, string) {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return fallback, shouldRequeueForCondition(fallback), defaultConditionMessage(fallback)
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return fallback, shouldRequeueForCondition(fallback), message
	case strings.Contains(lifecycleState, "FAIL"),
		strings.Contains(lifecycleState, "ERROR"),
		strings.Contains(lifecycleState, "NEEDS_ATTENTION"),
		strings.Contains(lifecycleState, "INOPERABLE"):
		return shared.Failed, false, message
	case strings.Contains(lifecycleState, "DELETE"),
		strings.Contains(lifecycleState, "TERMINAT"):
		return shared.Terminating, true, message
	case strings.Contains(lifecycleState, "UPDAT"),
		strings.Contains(lifecycleState, "MODIFY"),
		strings.Contains(lifecycleState, "PATCH"):
		return shared.Updating, true, message
	case strings.Contains(lifecycleState, "CREATE"),
		strings.Contains(lifecycleState, "PROVISION"),
		strings.Contains(lifecycleState, "PENDING"),
		strings.Contains(lifecycleState, "IN_PROGRESS"),
		strings.Contains(lifecycleState, "ACCEPT"),
		strings.Contains(lifecycleState, "START"):
		return shared.Provisioning, true, message
	default:
		return shared.Active, false, message
	}
}

func shouldRequeueForCondition(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating
}

func defaultConditionMessage(condition shared.OSOKConditionType) string {
	switch condition {
	case shared.Provisioning:
		return "OCI resource provisioning is in progress"
	case shared.Updating:
		return "OCI resource update is in progress"
	case shared.Terminating:
		return "OCI resource delete is in progress"
	case shared.Failed:
		return "OCI resource reconcile failed"
	default:
		return "OCI resource is active"
	}
}

func validateFormalSemantics(kind string, semantics *Semantics) error {
	if semantics == nil {
		return nil
	}
	problems := formalSemanticsProblems(semantics)
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s formal semantics blocked: %s", kind, strings.Join(problems, "; "))
}

func formalSemanticsProblems(semantics *Semantics) []string {
	problems := unsupportedSemanticProblems(semantics.Unsupported)
	problems = append(problems, auxiliaryOperationProblems(semantics.AuxiliaryOperations)...)
	problems = append(problems, followUpHelperProblems(semantics)...)
	if semantics.List != nil && strings.TrimSpace(semantics.List.ResponseItemsField) == "" {
		problems = append(problems, "list semantics require responseItemsField")
	}
	if semantics.Delete.Policy == "required" && len(semantics.Delete.TerminalStates) == 0 {
		problems = append(problems, "required delete semantics need terminal states")
	}
	return problems
}

func unsupportedSemanticProblems(unsupported []UnsupportedSemantic) []string {
	problems := make([]string, 0, len(unsupported))
	for _, gap := range unsupported {
		problems = append(problems, fmt.Sprintf("open formal gap %s: %s", gap.Category, gap.StopCondition))
	}
	return problems
}

func auxiliaryOperationProblems(operations []AuxiliaryOperation) []string {
	problems := make([]string, 0, len(operations))
	for _, operation := range operations {
		problems = append(problems, fmt.Sprintf("unsupported %s auxiliary operation %s", operation.Phase, operation.MethodName))
	}
	return problems
}

func followUpHelperProblems(semantics *Semantics) []string {
	var problems []string
	for phase, followUp := range map[string]FollowUpSemantics{
		"create": semantics.CreateFollowUp,
		"update": semantics.UpdateFollowUp,
		"delete": semantics.DeleteFollowUp,
	} {
		problems = append(problems, unsupportedFollowUpHelpers(phase, followUp)...)
	}
	return problems
}

func unsupportedFollowUpHelpers(phase string, followUp FollowUpSemantics) []string {
	var problems []string
	for _, hook := range followUp.Hooks {
		if !supportedFormalHelper(hook.Helper) {
			problems = append(problems, fmt.Sprintf("%s helper %q is not supported", phase, hook.Helper))
		}
	}
	return problems
}

func supportedFormalHelper(helper string) bool {
	switch strings.TrimSpace(helper) {
	case "", "tfresource.CreateResource", "tfresource.UpdateResource", "tfresource.DeleteResource", "tfresource.WaitForUpdatedState":
		return true
	default:
		return false
	}
}

func normalizeOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isNotFound(err error) bool {
	if errors.Is(err, errResourceNotFound) {
		return true
	}
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		if serviceErr.GetHTTPStatusCode() == 404 {
			return true
		}
		switch serviceErr.GetCode() {
		case "NotFound", "NotAuthorizedOrNotFound":
			return true
		default:
			return false
		}
	}

	message := err.Error()
	if strings.Contains(message, "http status code: 404") {
		return true
	}
	if strings.Contains(message, "NotFound") || strings.Contains(message, "NotAuthorizedOrNotFound") {
		return true
	}
	return false
}

func (c ServiceClient[T]) selectListItem(body any, resource T, preferredID string) (any, error) {
	items, err := c.listResponseItems(body)
	if err != nil {
		return nil, err
	}
	criteria, err := c.listCriteria(resource, preferredID)
	if err != nil {
		return nil, err
	}
	if c.hasFormalListSemantics() {
		return c.selectFormalListItem(items, criteria, preferredID)
	}
	return c.selectDefaultListItem(items, criteria)
}

func (c ServiceClient[T]) selectFormalListItem(items []any, criteria map[string]any, preferredID string) (any, error) {
	matches, comparedAny := c.formalListMatches(items, criteria, preferredID)
	return c.resolveFormalListMatches(matches, comparedAny, preferredID)
}

func listItems(body any, responseItemsField string) ([]any, error) {
	value, err := listBodyValue(body)
	if err != nil {
		return nil, err
	}
	if items, found, err := namedListItems(value, responseItemsField); found || err != nil {
		return items, err
	}
	if items, ok := sliceFieldValues(value, "Items"); ok {
		return items, nil
	}
	if items, ok := firstSliceFieldValues(value); ok {
		return items, nil
	}
	return nil, fmt.Errorf("OCI list body does not expose an items slice")
}

func (c ServiceClient[T]) listResponseItemsField() string {
	if c.config.Semantics == nil || c.config.Semantics.List == nil {
		return ""
	}
	return c.config.Semantics.List.ResponseItemsField
}

func (c ServiceClient[T]) listResponseItems(body any) ([]any, error) {
	items, err := listItems(body, c.listResponseItemsField())
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errResourceNotFound
	}
	return items, nil
}

func (c ServiceClient[T]) listCriteria(resource T, preferredID string) (map[string]any, error) {
	criteria, err := lookupValues(resource)
	if err != nil {
		return nil, err
	}
	if preferredID != "" {
		criteria["id"] = preferredID
		criteria["ocid"] = preferredID
	}
	return criteria, nil
}

func (c ServiceClient[T]) hasFormalListSemantics() bool {
	return c.config.Semantics != nil && c.config.Semantics.List != nil
}

func (c ServiceClient[T]) selectDefaultListItem(items []any, criteria map[string]any) (any, error) {
	targetID, targetName, targetDisplayName := defaultListTargets(criteria)
	matches := matchingDefaultListItems(items, targetID, targetName, targetDisplayName)
	switch {
	case len(matches) == 1:
		return matches[0], nil
	case len(matches) > 1:
		return nil, fmt.Errorf("%s list response returned multiple matching resources", c.config.Kind)
	case len(items) == 1:
		return items[0], nil
	default:
		return nil, errResourceNotFound
	}
}

func defaultListTargets(criteria map[string]any) (string, string, string) {
	return firstNonEmpty(criteria, "ocid", "id"), firstNonEmpty(criteria, "name", "metadataName"), firstNonEmpty(criteria, "displayName")
}

func matchingDefaultListItems(items []any, targetID string, targetName string, targetDisplayName string) []any {
	var matches []any
	for _, item := range items {
		if defaultListItemMatches(jsonMap(item), targetID, targetName, targetDisplayName) {
			matches = append(matches, item)
		}
	}
	return matches
}

func defaultListItemMatches(values map[string]any, targetID string, targetName string, targetDisplayName string) bool {
	switch {
	case targetID != "" && targetID == firstNonEmpty(values, "id", "ocid"):
		return true
	case targetDisplayName != "" && targetDisplayName == firstNonEmpty(values, "displayName"):
		return true
	case targetName != "" && targetName == firstNonEmpty(values, "name"):
		return true
	default:
		return false
	}
}

func (c ServiceClient[T]) formalListMatchFields() []string {
	if c.config.Semantics == nil || c.config.Semantics.List == nil {
		return nil
	}
	return append([]string(nil), c.config.Semantics.List.MatchFields...)
}

func (c ServiceClient[T]) formalListMatches(items []any, criteria map[string]any, preferredID string) ([]any, bool) {
	matchFields := c.formalListMatchFields()
	var matches []any
	comparedAny := false
	for _, item := range items {
		matched, compared := formalListItemMatch(item, criteria, preferredID, matchFields)
		if !compared {
			continue
		}
		comparedAny = true
		if matched {
			matches = append(matches, item)
		}
	}
	return matches, comparedAny
}

func formalListItemMatch(item any, criteria map[string]any, preferredID string, matchFields []string) (bool, bool) {
	values := jsonMap(item)
	if preferredID != "" && preferredID == firstNonEmpty(values, "id", "ocid") {
		return true, true
	}
	comparedFields := 0
	for _, field := range matchFields {
		expected, ok := lookupMeaningfulValue(criteria, field)
		if !ok {
			continue
		}
		comparedFields++
		actual, ok := lookupMeaningfulValue(values, field)
		if !ok || !valuesEqual(expected, actual) {
			return false, comparedFields > 0
		}
	}
	if comparedFields == 0 {
		return false, false
	}
	return true, true
}

func (c ServiceClient[T]) resolveFormalListMatches(matches []any, comparedAny bool, preferredID string) (any, error) {
	switch {
	case len(matches) == 1:
		return matches[0], nil
	case len(matches) > 1:
		return nil, fmt.Errorf("%s formal list semantics returned multiple matching resources", c.config.Kind)
	case comparedAny || preferredID != "":
		return nil, errResourceNotFound
	default:
		return nil, fmt.Errorf("%s formal list semantics did not yield any match criteria", c.config.Kind)
	}
}

func listBodyValue(body any) (reflect.Value, error) {
	value, ok := indirectValue(reflect.ValueOf(body))
	if !ok {
		return reflect.Value{}, errResourceNotFound
	}
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("OCI list body must be a struct, got %T", body)
	}
	return value, nil
}

func namedListItems(value reflect.Value, responseItemsField string) ([]any, bool, error) {
	fieldName := strings.TrimSpace(responseItemsField)
	if fieldName == "" {
		return nil, false, nil
	}
	items, ok := sliceFieldValues(value, fieldName)
	if ok {
		return items, true, nil
	}
	return nil, true, fmt.Errorf("OCI list body does not expose %s", responseItemsField)
}

func sliceFieldValues(value reflect.Value, fieldName string) ([]any, bool) {
	itemsField := value.FieldByName(fieldName)
	if !itemsField.IsValid() || itemsField.Kind() != reflect.Slice {
		return nil, false
	}
	return sliceValues(itemsField), true
}

func firstSliceFieldValues(value reflect.Value) ([]any, bool) {
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() != reflect.Slice {
			continue
		}
		return sliceValues(field), true
	}
	return nil, false
}

func sliceValues(value reflect.Value) []any {
	items := make([]any, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		items = append(items, value.Index(i).Interface())
	}
	return items
}

func assignField(field reflect.Value, raw any) error {
	converted, err := convertValue(raw, field.Type())
	if err != nil {
		return err
	}
	field.Set(converted)
	return nil
}

func convertValue(raw any, targetType reflect.Type) (reflect.Value, error) {
	if raw == nil {
		return reflect.Zero(targetType), nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("marshal source value: %w", err)
	}
	converted := reflect.New(targetType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("unmarshal into %s: %w", targetType, err)
	}
	return converted.Elem(), nil
}

func osokStatus(resource any) (*shared.OSOKStatus, error) {
	statusValue, err := statusStruct(resource)
	if err != nil {
		return nil, err
	}

	field := statusValue.FieldByName("OsokStatus")
	if !field.IsValid() || !field.CanAddr() {
		return nil, fmt.Errorf("resource %T does not expose Status.OsokStatus", resource)
	}

	status, ok := field.Addr().Interface().(*shared.OSOKStatus)
	if !ok {
		return nil, fmt.Errorf("resource %T Status.OsokStatus has unexpected type %T", resource, field.Addr().Interface())
	}
	return status, nil
}

func statusStruct(resource any) (reflect.Value, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return reflect.Value{}, err
	}
	field, ok := fieldValue(resourceValue, "Status")
	if !ok {
		return reflect.Value{}, fmt.Errorf("resource %T does not expose Status", resource)
	}
	return field, nil
}

func resourceStruct(resource any) (reflect.Value, error) {
	value := reflect.ValueOf(resource)
	if !value.IsValid() {
		return reflect.Value{}, fmt.Errorf("resource is nil")
	}
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return reflect.Value{}, fmt.Errorf("expected pointer resource, got %T", resource)
	}
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected pointer to struct resource, got %T", resource)
	}
	return value, nil
}

func fieldValue(value reflect.Value, name string) (reflect.Value, bool) {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return reflect.Value{}, false
	}
	return field, true
}

func fieldInterface(value reflect.Value, name string) any {
	field, ok := fieldValue(value, name)
	if !ok || !field.IsValid() {
		return nil
	}
	return field.Interface()
}

func lookupMetadataString(value reflect.Value, fieldName string) string {
	field, ok := fieldValue(value, fieldName)
	if !ok || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

func resourceNamespace(resource any, fallback string) string {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return strings.TrimSpace(fallback)
	}
	namespace := lookupMetadataString(resourceValue, "Namespace")
	if strings.TrimSpace(namespace) != "" {
		return namespace
	}
	return strings.TrimSpace(fallback)
}

func fieldJSONName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func jsonMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}

func responseLifecycleState(response any) string {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return ""
	}
	return firstNonEmpty(jsonMap(body), "lifecycleState", "status")
}

func lookupValueByPaths(values map[string]any, paths ...string) (any, bool) {
	for _, path := range paths {
		if value, ok := lookupValueByPath(values, path); ok {
			return value, true
		}
	}
	return nil, false
}

func lookupMeaningfulValue(values map[string]any, path string) (any, bool) {
	value, ok := lookupValueByPath(values, path)
	if !ok || !meaningfulValue(value) {
		return nil, false
	}
	return value, true
}

func lookupValueByPath(values map[string]any, path string) (any, bool) {
	if values == nil {
		return nil, false
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}

	current := any(values)
	for _, segment := range strings.Split(path, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, false
		}

		mapValue, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := mapValue[segment]
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func meaningfulValue(value any) bool {
	if value == nil {
		return false
	}

	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case []any:
		return len(concrete) > 0
	case map[string]any:
		return len(concrete) > 0
	case bool:
		return concrete
	case float64:
		return concrete != 0
	default:
		return true
	}
}

func valuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := lookupString(values, key); value != "" {
			return value
		}
	}
	return ""
}

func lookupString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return ""
	}
	switch concrete := raw.(type) {
	case string:
		return concrete
	default:
		return fmt.Sprint(concrete)
	}
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func appendUniqueStrings(existing []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(extras))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range extras {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}

func lowerCamel(name string) string {
	tokens := splitCamel(name)
	if len(tokens) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(tokens[0])
	for _, token := range tokens[1:] {
		builder.WriteString(strings.ToUpper(token[:1]))
		builder.WriteString(token[1:])
	}
	return builder.String()
}

func splitCamel(name string) []string {
	if strings.TrimSpace(name) == "" {
		return nil
	}

	var tokens []string
	var current []rune
	runes := []rune(name)
	for index, r := range runes {
		if index > 0 && startsNewCamelToken(runes, index) {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = current[:0]
		}
		current = append(current, r)
	}
	return appendCurrentCamelToken(tokens, current)
}

func startsNewCamelToken(runes []rune, index int) bool {
	prev := runes[index-1]
	nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
	return unicode.IsUpper(runes[index]) &&
		(unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower))
}

func appendCurrentCamelToken(tokens []string, current []rune) []string {
	if len(current) == 0 {
		return tokens
	}
	return append(tokens, strings.ToLower(string(current)))
}
