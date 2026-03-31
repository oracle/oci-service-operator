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
	Kind      string
	SDKName   string
	Log       loggerutil.OSOKLogger
	InitError error
	Semantics *Semantics

	Create *Operation
	Get    *Operation
	List   *Operation
	Update *Operation
	Delete *Operation
}

type ServiceClient[T any] struct {
	config Config[T]
}

type readPhase string

const (
	readPhaseObserve readPhase = "observe"
	readPhaseCreate  readPhase = "create"
	readPhaseUpdate  readPhase = "update"
	readPhaseDelete  readPhase = "delete"
)

func NewServiceClient[T any](cfg Config[T]) ServiceClient[T] {
	if err := validateFormalSemantics(cfg.Kind, cfg.Semantics); err != nil {
		cfg.InitError = errors.Join(cfg.InitError, err)
	}
	return ServiceClient[T]{config: cfg}
}

func (c ServiceClient[T]) CreateOrUpdate(ctx context.Context, resource T, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.config.InitError != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, c.config.InitError)
	}
	if _, err := resourceStruct(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	currentID := c.currentID(resource)
	existingResponse, err := c.resolveExistingBeforeCreate(ctx, resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	resolvedExistingBeforeCreate := currentID == "" && existingResponse != nil
	if currentID == "" && existingResponse != nil {
		currentID = responseID(existingResponse)
	}

	liveResponse := existingResponse
	if currentID != "" && c.requiresLiveMutationValidation() {
		forceLiveGet := resolvedExistingBeforeCreate && c.config.Get != nil
		if liveResponse == nil || forceLiveGet {
			response, err := c.readResourceForMutationValidation(ctx, resource, currentID, forceLiveGet)
			if err != nil {
				if !errors.Is(err, errResourceNotFound) {
					return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
				}
				if forceLiveGet {
					currentID = ""
					liveResponse = nil
				}
			} else {
				liveResponse = response
			}
		}
	}
	if currentID != "" && liveResponse != nil {
		_ = mergeResponseIntoStatus(resource, liveResponse)
	}

	if err := c.validateMutationPolicy(resource, currentID != "", liveResponse); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	if currentID != "" {
		if c.config.Update != nil {
			response, err := c.invoke(ctx, c.config.Update, resource, currentID)
			if err != nil {
				return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
			}
			response, err = c.followUpAfterWrite(ctx, resource, currentID, response, "update")
			if err != nil {
				return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
			}
			return c.applySuccess(resource, response, shared.Updating)
		}

		response := liveResponse
		if response == nil {
			response, err = c.readResource(ctx, resource, currentID, readPhaseObserve)
			if err != nil {
				return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
			}
		}
		return c.applySuccess(resource, response, shared.Active)
	}

	if c.config.Create != nil {
		response, err := c.invoke(ctx, c.config.Create, resource, "")
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}

		followUp, err := c.followUpAfterWrite(ctx, resource, responseID(response), response, "create")
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}
		return c.applySuccess(resource, followUp, shared.Provisioning)
	}

	response, err := c.readResource(ctx, resource, "", readPhaseObserve)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	return c.applySuccess(resource, response, shared.Active)
}

func (c ServiceClient[T]) Delete(ctx context.Context, resource T) (bool, error) {
	if c.config.InitError != nil {
		return false, c.config.InitError
	}
	if _, err := resourceStruct(resource); err != nil {
		return false, err
	}
	if c.config.Semantics != nil {
		return c.deleteWithSemantics(ctx, resource)
	}
	if c.config.Delete == nil {
		c.markDeleted(resource, "OCI delete is not supported for this generated resource")
		return true, nil
	}

	currentID := c.currentID(resource)
	if currentID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	if _, err := c.invoke(ctx, c.config.Delete, resource, currentID); err != nil {
		if isNotFound(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}

	if c.config.Get == nil && c.config.List == nil {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
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

	refreshed, err := c.readResource(ctx, resource, preferredID, phaseReadPhase(phase))
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

func phaseReadPhase(phase string) readPhase {
	switch phase {
	case "create":
		return readPhaseCreate
	case "update":
		return readPhaseUpdate
	case "delete":
		return readPhaseDelete
	default:
		return readPhaseObserve
	}
}

func (c ServiceClient[T]) deleteWithSemantics(ctx context.Context, resource T) (bool, error) {
	semantics := c.config.Semantics
	if semantics == nil {
		return false, fmt.Errorf("%s formal semantics are not configured", c.config.Kind)
	}
	if c.config.Delete == nil || semantics.Delete.Policy == "not-supported" {
		return false, fmt.Errorf("%s formal semantics mark delete confirmation as %q", c.config.Kind, semantics.Delete.Policy)
	}

	currentID, err := c.resolveDeleteID(ctx, resource)
	if err != nil {
		if errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}

	if _, err := c.invoke(ctx, c.config.Delete, resource, currentID); err != nil {
		if isNotFound(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}

	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}
	if c.config.Get == nil && c.config.List == nil {
		return false, fmt.Errorf("%s formal delete confirmation requires a readable OCI operation", c.config.Kind)
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}
	_ = mergeResponseIntoStatus(resource, response)

	lifecycleState := strings.ToUpper(responseLifecycleState(response))
	switch semantics.Delete.Policy {
	case "best-effort":
		if lifecycleState == "" ||
			containsString(semantics.Delete.PendingStates, lifecycleState) ||
			containsString(semantics.Delete.TerminalStates, lifecycleState) {
			c.markDeleted(resource, "OCI delete request accepted")
			return true, nil
		}
		c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
		return false, nil
	case "required":
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
	default:
		return false, fmt.Errorf("%s formal delete confirmation policy %q is not supported", c.config.Kind, semantics.Delete.Policy)
	}
}

func (c ServiceClient[T]) resolveDeleteID(ctx context.Context, resource T) (string, error) {
	currentID := c.currentID(resource)
	if currentID != "" {
		return currentID, nil
	}

	if c.config.Get == nil && c.config.List == nil {
		return "", errResourceNotFound
	}

	response, err := c.readResource(ctx, resource, "", readPhaseDelete)
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

func (c ServiceClient[T]) resolveExistingBeforeCreate(ctx context.Context, resource T) (any, error) {
	if !c.shouldResolveExistingBeforeCreate() {
		return nil, nil
	}

	response, err := c.readResource(ctx, resource, "", readPhaseCreate)
	if err == nil {
		return response, nil
	}
	if errors.Is(err, errResourceNotFound) {
		return nil, nil
	}
	return nil, err
}

func (c ServiceClient[T]) shouldResolveExistingBeforeCreate() bool {
	return c.config.Create != nil && c.config.List != nil && c.config.Semantics != nil && c.config.Semantics.List != nil
}

func (c ServiceClient[T]) requiresLiveMutationValidation() bool {
	return c.config.Semantics != nil &&
		len(c.config.Semantics.Mutation.ForceNew) > 0 &&
		(c.config.Get != nil || c.config.List != nil)
}

func (c ServiceClient[T]) validateMutationPolicy(resource T, existing bool, currentResponse any) error {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil
	}

	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return err
	}
	specValues := jsonMap(fieldInterface(resourceValue, "Spec"))
	currentValues := jsonMap(fieldInterface(resourceValue, "Status"))
	if body, ok := responseBody(currentResponse); ok && body != nil {
		currentValues = jsonMap(body)
	}

	for field, conflicts := range semantics.Mutation.ConflictsWith {
		if _, ok := lookupMeaningfulValue(specValues, field); !ok {
			continue
		}
		for _, conflict := range conflicts {
			if _, ok := lookupMeaningfulValue(specValues, conflict); ok {
				return fmt.Errorf("%s formal semantics forbid setting %s with %s", c.config.Kind, field, conflict)
			}
		}
	}

	if !existing {
		return nil
	}
	for _, field := range semantics.Mutation.ForceNew {
		specValue, specOK := lookupValueByPath(specValues, field)
		statusValue, statusOK := lookupValueByPath(currentValues, field)
		if !specOK || !statusOK {
			continue
		}
		if !valuesEqual(specValue, statusValue) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", c.config.Kind, field)
		}
	}

	return nil
}

func (c ServiceClient[T]) readResource(ctx context.Context, resource T, preferredID string, phase readPhase) (any, error) {
	readID := preferredID
	if readID == "" {
		readID = c.currentID(resource)
	}

	if c.config.Get != nil && c.canInvokeGet(resource, readID) {
		response, err := c.invoke(ctx, c.config.Get, resource, readID)
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

	response, err := c.invoke(ctx, c.config.List, resource, readID)
	if err != nil {
		return nil, err
	}

	body, ok := responseBody(response)
	if !ok {
		return nil, fmt.Errorf("%s list response did not expose a body payload", c.config.Kind)
	}

	item, err := c.selectListItem(body, resource, readID, phase)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (c ServiceClient[T]) readResourceForMutationValidation(ctx context.Context, resource T, currentID string, forceLiveGet bool) (any, error) {
	if !forceLiveGet {
		return c.readResource(ctx, resource, currentID, readPhaseUpdate)
	}
	if c.config.Get == nil {
		return nil, fmt.Errorf("%s generated runtime has no OCI Get operation for live mutation validation", c.config.Kind)
	}

	response, err := c.invoke(ctx, c.config.Get, resource, currentID)
	if err != nil {
		if isNotFound(err) {
			return nil, errResourceNotFound
		}
		return nil, err
	}
	return response, nil
}

func (c ServiceClient[T]) canInvokeGet(resource T, preferredID string) bool {
	if c.config.Get == nil {
		return false
	}

	values, err := lookupValues(resource)
	if err != nil {
		return true
	}

	if len(c.config.Get.Fields) > 0 {
		for _, field := range c.config.Get.Fields {
			if !requestFieldRequiresResourceID(field, c.idFieldAliases()) {
				continue
			}
			if _, ok := explicitRequestValue(values, field, preferredID); !ok {
				return false
			}
		}
		return true
	}

	request := c.config.Get.NewRequest()
	if request == nil {
		return true
	}

	requestValue := reflect.ValueOf(request)
	if !requestValue.IsValid() || requestValue.Kind() != reflect.Pointer || requestValue.IsNil() {
		return true
	}

	requestStruct := requestValue.Elem()
	if requestStruct.Kind() != reflect.Struct {
		return true
	}

	requestType := requestStruct.Type()
	for i := 0; i < requestStruct.NumField(); i++ {
		fieldValue := requestStruct.Field(i)
		fieldType := requestType.Field(i)
		if !fieldValue.CanSet() || fieldType.Name == "RequestMetadata" {
			continue
		}

		switch fieldType.Tag.Get("contributesTo") {
		case "header", "binary", "body":
			continue
		}

		lookupKey := fieldType.Tag.Get("name")
		if lookupKey == "" {
			lookupKey = fieldJSONName(fieldType)
		}
		if lookupKey == "" {
			lookupKey = lowerCamel(fieldType.Name)
		}
		if !containsString(c.idFieldAliases(), lookupKey) {
			continue
		}
		if preferredID != "" {
			continue
		}
		if _, ok := lookupValueByPaths(values, lookupKey); !ok {
			return false
		}
	}

	return true
}

func (c ServiceClient[T]) invoke(ctx context.Context, op *Operation, resource T, preferredID string) (any, error) {
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
	if err := buildRequest(request, resource, preferredID, op.Fields, c.idFieldAliases()); err != nil {
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
	return firstNonEmpty(values, c.idFieldAliases()...)
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

func buildRequest(request any, resource any, preferredID string, fields []RequestField, idAliases []string) error {
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

	if len(fields) > 0 {
		return buildExplicitRequest(requestStruct, resource, values, preferredID, fields)
	}

	return buildHeuristicRequest(requestStruct, requestStruct.Type(), resource, values, preferredID, idAliases)
}

func buildExplicitRequest(requestStruct reflect.Value, resource any, values map[string]any, preferredID string, fields []RequestField) error {
	for _, field := range fields {
		fieldValue := requestStruct.FieldByName(field.FieldName)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			continue
		}

		switch field.Contribution {
		case "header", "binary":
			continue
		case "body":
			if err := assignField(fieldValue, specValue(resource)); err != nil {
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
	resource any,
	values map[string]any,
	preferredID string,
	idAliases []string,
) error {
	for i := 0; i < requestStruct.NumField(); i++ {
		fieldValue := requestStruct.Field(i)
		fieldType := requestType.Field(i)
		if !fieldValue.CanSet() {
			continue
		}
		if fieldType.Name == "RequestMetadata" {
			continue
		}

		switch fieldType.Tag.Get("contributesTo") {
		case "header", "binary":
			continue
		case "body":
			if err := assignField(fieldValue, specValue(resource)); err != nil {
				return fmt.Errorf("set body field %s: %w", fieldType.Name, err)
			}
			continue
		}

		lookupKey := fieldType.Tag.Get("name")
		if lookupKey == "" {
			lookupKey = fieldJSONName(fieldType)
		}
		if lookupKey == "" {
			lookupKey = lowerCamel(fieldType.Name)
		}

		rawValue, ok := values[lookupKey]
		if !ok && preferredID != "" && containsString(idAliases, lookupKey) {
			rawValue = preferredID
			ok = true
		}
		if !ok && lookupKey == "name" {
			if metadataName, exists := values["metadataName"]; exists {
				rawValue, ok = metadataName, true
			}
		}
		if !ok && lookupKey == "namespaceName" {
			if namespaceName, exists := values["namespaceName"]; exists {
				rawValue, ok = namespaceName, true
			}
		}
		if !ok {
			continue
		}

		if err := assignField(fieldValue, rawValue); err != nil {
			return fmt.Errorf("set request field %s: %w", fieldType.Name, err)
		}
	}

	return nil
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

func requestFieldRequiresResourceID(field RequestField, idAliases []string) bool {
	if field.PreferResourceID {
		return true
	}

	lookupKey := strings.TrimSpace(field.RequestName)
	if lookupKey == "" {
		lookupKey = lowerCamel(field.FieldName)
	}
	return containsString(idAliases, lookupKey)
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

func responseBody(response any) (any, bool) {
	if response == nil {
		return nil, false
	}

	value := reflect.ValueOf(response)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, false
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return response, true
	}

	typ := value.Type()
	if !strings.HasSuffix(typ.Name(), "Response") {
		return value.Interface(), true
	}

	var fallback reflect.Value
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		fieldValue := value.Field(i)
		if fieldType.Tag.Get("presentIn") == "body" {
			if fieldValue.Kind() == reflect.Pointer {
				if fieldValue.IsNil() {
					return nil, false
				}
				return fieldValue.Interface(), true
			}
			return fieldValue.Interface(), true
		}
		if fieldType.Name == "RawResponse" || strings.HasPrefix(fieldType.Name, "Opc") || fieldType.Name == "Etag" {
			continue
		}
		if !fallback.IsValid() {
			fallback = fieldValue
		}
	}

	if fallback.IsValid() {
		return fallback.Interface(), true
	}
	return nil, false
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

	var problems []string
	for _, gap := range semantics.Unsupported {
		problems = append(problems, fmt.Sprintf("open formal gap %s: %s", gap.Category, gap.StopCondition))
	}
	for _, operation := range semantics.AuxiliaryOperations {
		problems = append(problems, fmt.Sprintf("unsupported %s auxiliary operation %s", operation.Phase, operation.MethodName))
	}
	for phase, followUp := range map[string]FollowUpSemantics{
		"create": semantics.CreateFollowUp,
		"update": semantics.UpdateFollowUp,
		"delete": semantics.DeleteFollowUp,
	} {
		for _, hook := range followUp.Hooks {
			if !supportedFormalHelper(hook.Helper) {
				problems = append(problems, fmt.Sprintf("%s helper %q is not supported", phase, hook.Helper))
			}
		}
	}
	if semantics.List != nil && strings.TrimSpace(semantics.List.ResponseItemsField) == "" {
		problems = append(problems, "list semantics require responseItemsField")
	}
	if semantics.Delete.Policy == "required" && len(semantics.Delete.TerminalStates) == 0 {
		problems = append(problems, "required delete semantics need terminal states")
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s formal semantics blocked: %s", kind, strings.Join(problems, "; "))
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

func (c ServiceClient[T]) selectListItem(body any, resource T, preferredID string, phase readPhase) (any, error) {
	responseItemsField := ""
	if c.config.Semantics != nil && c.config.Semantics.List != nil {
		responseItemsField = c.config.Semantics.List.ResponseItemsField
	}
	items, err := listItems(body, responseItemsField)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errResourceNotFound
	}

	criteria, err := lookupValues(resource)
	if err != nil {
		return nil, err
	}
	if preferredID != "" {
		criteria["id"] = preferredID
		criteria["ocid"] = preferredID
	}
	if c.config.Semantics != nil && c.config.Semantics.List != nil {
		return c.selectFormalListItem(items, criteria, preferredID, phase)
	}

	targetID := firstNonEmpty(criteria, "ocid", "id")
	targetName := firstNonEmpty(criteria, "name", "metadataName")
	targetDisplayName := firstNonEmpty(criteria, "displayName")

	var matches []any
	for _, item := range items {
		values := jsonMap(item)
		switch {
		case targetID != "" && targetID == firstNonEmpty(values, "id", "ocid"):
			matches = append(matches, item)
		case targetDisplayName != "" && targetDisplayName == firstNonEmpty(values, "displayName"):
			matches = append(matches, item)
		case targetName != "" && targetName == firstNonEmpty(values, "name"):
			matches = append(matches, item)
		}
	}

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

func (c ServiceClient[T]) selectFormalListItem(items []any, criteria map[string]any, preferredID string, phase readPhase) (any, error) {
	matchFields := []string{}
	if c.config.Semantics != nil && c.config.Semantics.List != nil {
		matchFields = append(matchFields, c.config.Semantics.List.MatchFields...)
	}

	var matches []any
	comparedAny := false
	for _, item := range items {
		values := jsonMap(item)
		if preferredID != "" && preferredID == firstNonEmpty(values, "id", "ocid") {
			matches = append(matches, item)
			continue
		}

		comparedFields := 0
		matched := true
		for _, field := range matchFields {
			expected, ok := lookupMeaningfulValue(criteria, field)
			if !ok {
				continue
			}
			comparedFields++
			actual, ok := lookupMeaningfulValue(values, field)
			if !ok || !valuesEqual(expected, actual) {
				matched = false
				break
			}
		}
		if comparedFields == 0 {
			continue
		}
		comparedAny = true
		if matched {
			matches = append(matches, item)
		}
	}
	matches = c.filterPhaseMatches(matches, phase)

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

func (c ServiceClient[T]) filterPhaseMatches(matches []any, phase readPhase) []any {
	if len(matches) == 0 {
		return nil
	}

	switch phase {
	case readPhaseCreate, readPhaseUpdate, readPhaseObserve:
		filtered := make([]any, 0, len(matches))
		for _, item := range matches {
			if c.allowListItemForReadPhase(item, phase) {
				filtered = append(filtered, item)
			}
		}
		return filtered
	case readPhaseDelete:
		bestPriority := 0
		filtered := make([]any, 0, len(matches))
		for _, item := range matches {
			priority := c.deleteListItemPriority(item)
			if priority > bestPriority {
				bestPriority = priority
				filtered = filtered[:0]
			}
			if priority == bestPriority {
				filtered = append(filtered, item)
			}
		}
		return filtered
	default:
		return matches
	}
}

func (c ServiceClient[T]) allowListItemForReadPhase(item any, phase readPhase) bool {
	switch c.listItemLifecycleCategory(item) {
	case lifecycleCategoryProvisioning, lifecycleCategoryUpdating, lifecycleCategoryActive, lifecycleCategoryEmpty:
		return true
	case lifecycleCategoryUnknown:
		return phase == readPhaseObserve
	default:
		return false
	}
}

func (c ServiceClient[T]) deleteListItemPriority(item any) int {
	switch c.listItemLifecycleCategory(item) {
	case lifecycleCategoryProvisioning, lifecycleCategoryUpdating, lifecycleCategoryActive:
		return 4
	case lifecycleCategoryDeleting, lifecycleCategoryDeleted:
		return 3
	case lifecycleCategoryFailed:
		return 2
	default:
		return 1
	}
}

type lifecycleCategory string

const (
	lifecycleCategoryEmpty        lifecycleCategory = "empty"
	lifecycleCategoryProvisioning lifecycleCategory = "provisioning"
	lifecycleCategoryUpdating     lifecycleCategory = "updating"
	lifecycleCategoryActive       lifecycleCategory = "active"
	lifecycleCategoryDeleting     lifecycleCategory = "deleting"
	lifecycleCategoryDeleted      lifecycleCategory = "deleted"
	lifecycleCategoryFailed       lifecycleCategory = "failed"
	lifecycleCategoryUnknown      lifecycleCategory = "unknown"
)

func (c ServiceClient[T]) listItemLifecycleCategory(item any) lifecycleCategory {
	state := strings.ToUpper(firstNonEmpty(jsonMap(item), "lifecycleState", "status", "state"))
	if state == "" {
		return lifecycleCategoryEmpty
	}

	if c.config.Semantics != nil {
		switch {
		case containsString(c.config.Semantics.Lifecycle.ProvisioningStates, state):
			return lifecycleCategoryProvisioning
		case containsString(c.config.Semantics.Lifecycle.UpdatingStates, state):
			return lifecycleCategoryUpdating
		case containsString(c.config.Semantics.Lifecycle.ActiveStates, state):
			return lifecycleCategoryActive
		case containsString(c.config.Semantics.Delete.PendingStates, state):
			return lifecycleCategoryDeleting
		case containsString(c.config.Semantics.Delete.TerminalStates, state):
			return lifecycleCategoryDeleted
		}
	}

	switch {
	case strings.Contains(state, "FAIL"),
		strings.Contains(state, "ERROR"),
		strings.Contains(state, "NEEDS_ATTENTION"),
		strings.Contains(state, "INOPERABLE"):
		return lifecycleCategoryFailed
	case strings.Contains(state, "DELETED"),
		strings.Contains(state, "TERMINATED"):
		return lifecycleCategoryDeleted
	case strings.Contains(state, "DELETE"),
		strings.Contains(state, "TERMINAT"):
		return lifecycleCategoryDeleting
	case strings.Contains(state, "UPDAT"),
		strings.Contains(state, "MODIFY"),
		strings.Contains(state, "PATCH"):
		return lifecycleCategoryUpdating
	case strings.Contains(state, "CREATE"),
		strings.Contains(state, "PROVISION"),
		strings.Contains(state, "PENDING"),
		strings.Contains(state, "IN_PROGRESS"),
		strings.Contains(state, "ACCEPT"),
		strings.Contains(state, "START"):
		return lifecycleCategoryProvisioning
	default:
		return lifecycleCategoryUnknown
	}
}

func listItems(body any, responseItemsField string) ([]any, error) {
	value := reflect.ValueOf(body)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, errResourceNotFound
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("OCI list body must be a struct, got %T", body)
	}

	if strings.TrimSpace(responseItemsField) != "" {
		itemsField := value.FieldByName(responseItemsField)
		if itemsField.IsValid() && itemsField.Kind() == reflect.Slice {
			return sliceValues(itemsField), nil
		}
		return nil, fmt.Errorf("OCI list body does not expose %s", responseItemsField)
	}

	if itemsField := value.FieldByName("Items"); itemsField.IsValid() && itemsField.Kind() == reflect.Slice {
		return sliceValues(itemsField), nil
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() != reflect.Slice {
			continue
		}
		return sliceValues(field), nil
	}

	return nil, fmt.Errorf("OCI list body does not expose an items slice")
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
		next, ok := lookupMapKey(mapValue, segment)
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func lookupMapKey(values map[string]any, segment string) (any, bool) {
	if value, ok := values[segment]; ok {
		return value, true
	}

	normalized := lowerCamel(segment)
	for key, value := range values {
		if strings.EqualFold(key, segment) || lowerCamel(key) == normalized {
			return value, true
		}
	}
	return nil, false
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
	raw, ok := lookupValueByPath(values, key)
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
		if index > 0 {
			prev := runes[index-1]
			nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower)) {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = current[:0]
			}
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}
	return tokens
}
