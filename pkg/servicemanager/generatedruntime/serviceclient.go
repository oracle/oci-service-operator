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
	"sort"
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
	InitError        error
	Semantics        *Semantics
	CredentialClient credhelper.CredentialClient

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

//nolint:gocognit,gocyclo // Reconcile orchestration branches across create, update, and read-only generated-runtime flows.
func (c ServiceClient[T]) CreateOrUpdate(ctx context.Context, resource T, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.config.InitError != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, c.config.InitError)
	}
	if _, err := resourceStruct(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	currentID := c.currentID(resource)
	if err := c.validateMutationPolicy(resource, currentID != ""); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	if currentID != "" {
		shouldUpdate, err := c.shouldUpdateExistingResource(resource)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}
		if shouldUpdate {
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

		if c.config.Get == nil && c.config.List == nil {
			c.markCondition(resource, shared.Active, defaultConditionMessage(shared.Active))
			return servicemanager.OSOKResponse{
				IsSuccessful:    true,
				ShouldRequeue:   false,
				RequeueDuration: defaultRequeueDuration,
			}, nil
		}

		response, err := c.readResource(ctx, resource, currentID)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}
		return c.applySuccess(resource, response, shared.Active)
	}

	if c.shouldBindBeforeCreate() {
		response, err := c.bindBeforeCreate(ctx, resource)
		switch {
		case err == nil:
			return c.applySuccess(resource, response, shared.Active)
		case !errors.Is(err, errResourceNotFound):
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}
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

	response, err := c.readResource(ctx, resource, "")
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	return c.applySuccess(resource, response, shared.Active)
}

//nolint:gocyclo // Delete must coordinate semantic and legacy fallback paths in one entrypoint.
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

	response, err := c.readResource(ctx, resource, currentID)
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

//nolint:gocognit,gocyclo // Semantic delete handling encodes policy-specific confirmation behavior and lifecycle mapping.
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

	response, err := c.readResource(ctx, resource, currentID)
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

//nolint:gocognit,gocyclo // Mutation validation combines conflicts and force-new comparisons over dotted JSON paths.
func (c ServiceClient[T]) validateMutationPolicy(resource T, existing bool) error {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil
	}

	specValues, statusValues, err := c.mutationValues(resource)
	if err != nil {
		return err
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
		statusValue, statusOK := lookupValueByPath(statusValues, field)
		if !specOK || !statusOK {
			continue
		}
		if !valuesEqual(specValue, statusValue) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", c.config.Kind, field)
		}
	}
	if c.config.Update == nil {
		return nil
	}

	unsupportedPaths := unsupportedUpdateDriftPaths(specValues, statusValues, semantics.Mutation)
	if len(unsupportedPaths) == 0 {
		return nil
	}
	return fmt.Errorf("%s formal semantics reject unsupported update drift for %s", c.config.Kind, strings.Join(unsupportedPaths, ", "))
}

func (c ServiceClient[T]) mutationValues(resource T) (map[string]any, map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, nil, err
	}
	return jsonMap(fieldInterface(resourceValue, "Spec")), jsonMap(fieldInterface(resourceValue, "Status")), nil
}

func (c ServiceClient[T]) shouldUpdateExistingResource(resource T) (bool, error) {
	if c.config.Update == nil {
		return false, nil
	}
	if c.config.Semantics == nil {
		return true, nil
	}
	return c.hasMutableDrift(resource)
}

func (c ServiceClient[T]) hasMutableDrift(resource T) (bool, error) {
	semantics := c.config.Semantics
	if semantics == nil {
		return false, nil
	}

	specValues, statusValues, err := c.mutationValues(resource)
	if err != nil {
		return false, err
	}
	for _, field := range semantics.Mutation.Mutable {
		specValue, specOK := lookupValueByPath(specValues, field)
		statusValue, statusOK := lookupValueByPath(statusValues, field)
		if !specOK || !statusOK {
			continue
		}
		if !valuesEqual(specValue, statusValue) {
			return true, nil
		}
	}
	return false, nil
}

func unsupportedUpdateDriftPaths(specValues map[string]any, statusValues map[string]any, semantics MutationSemantics) []string {
	diffPaths := comparableDiffPaths(specValues, statusValues, "")
	unsupported := make([]string, 0, len(diffPaths))
	for _, path := range diffPaths {
		switch {
		case pathCoveredByAny(path, semantics.Mutable):
		case pathCoveredByAny(path, semantics.ForceNew):
		default:
			unsupported = appendUniqueStrings(unsupported, path)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}

func (c ServiceClient[T]) bindBeforeCreate(ctx context.Context, resource T) (any, error) {
	response, err := c.invoke(ctx, c.config.List, resource, "")
	if err != nil {
		return nil, err
	}

	body, ok := responseBody(response)
	if !ok {
		return nil, fmt.Errorf("%s list response did not expose a body payload", c.config.Kind)
	}

	item, err := c.selectReusableListItem(body, resource)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (c ServiceClient[T]) readResource(ctx context.Context, resource T, preferredID string) (any, error) {
	if c.config.Get != nil && !c.shouldUseFormalListLookup(preferredID) {
		response, err := c.invoke(ctx, c.config.Get, resource, preferredID)
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

	response, err := c.invoke(ctx, c.config.List, resource, preferredID)
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

func (c ServiceClient[T]) shouldBindBeforeCreate() bool {
	return c.config.Create != nil &&
		c.config.List != nil &&
		c.config.Semantics != nil &&
		c.config.Semantics.List != nil
}

func (c ServiceClient[T]) shouldUseFormalListLookup(preferredID string) bool {
	return preferredID == "" &&
		c.config.Semantics != nil &&
		c.config.Semantics.List != nil &&
		c.config.List != nil
}

func (c ServiceClient[T]) selectReusableListItem(body any, resource T) (any, error) {
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
	return c.selectFormalReusableListItem(items, criteria)
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
	if err := buildRequest(ctx, request, resource, preferredID, op.Fields, c.idFieldAliases(), c.config.CredentialClient); err != nil {
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
	stampSecretSourceStatus(resource)

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

func buildRequest(
	ctx context.Context,
	request any,
	resource any,
	preferredID string,
	fields []RequestField,
	idAliases []string,
	credentialClient credhelper.CredentialClient,
) error {
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
		return buildExplicitRequest(ctx, requestStruct, resource, values, preferredID, fields, credentialClient)
	}

	return buildHeuristicRequest(ctx, requestStruct, requestStruct.Type(), resource, values, preferredID, idAliases, credentialClient)
}

func buildExplicitRequest(
	ctx context.Context,
	requestStruct reflect.Value,
	resource any,
	values map[string]any,
	preferredID string,
	fields []RequestField,
	credentialClient credhelper.CredentialClient,
) error {
	for _, field := range fields {
		fieldValue := requestStruct.FieldByName(field.FieldName)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			continue
		}

		switch field.Contribution {
		case "header", "binary":
			continue
		case "body":
			if err := assignSpecField(ctx, fieldValue, resource, credentialClient); err != nil {
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

//nolint:gocognit,gocyclo // Heuristic request projection must account for body, path, query, metadata, and ID alias cases.
func buildHeuristicRequest(
	ctx context.Context,
	requestStruct reflect.Value,
	requestType reflect.Type,
	resource any,
	values map[string]any,
	preferredID string,
	idAliases []string,
	credentialClient credhelper.CredentialClient,
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
			if err := assignSpecField(ctx, fieldValue, resource, credentialClient); err != nil {
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

func assignSpecField(ctx context.Context, field reflect.Value, resource any, credentialClient credhelper.CredentialClient) error {
	prepared, err := prepareSpecValue(ctx, specValue(resource), field.Type(), resource, credentialClient)
	if err != nil {
		return err
	}
	return assignField(field, prepared)
}

func prepareSpecValue(
	ctx context.Context,
	spec any,
	targetType reflect.Type,
	resource any,
	credentialClient credhelper.CredentialClient,
) (any, error) {
	raw, err := normalizeJSONValue(spec)
	if err != nil {
		return nil, err
	}

	namespace := ""
	values, lookupErr := lookupValues(resource)
	if lookupErr == nil {
		namespace = firstNonEmpty(values, "namespaceName", "namespace")
	}

	return prepareValueForTarget(ctx, raw, targetType, namespace, credentialClient, nil)
}

func normalizeJSONValue(raw any) (any, error) {
	if raw == nil {
		return nil, nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal source value: %w", err)
	}
	var normalized any
	if err := json.Unmarshal(payload, &normalized); err != nil {
		return nil, fmt.Errorf("normalize source value: %w", err)
	}
	return compactNormalizedJSONValue(normalized), nil
}

func compactNormalizedJSONValue(raw any) any {
	switch value := raw.(type) {
	case map[string]any:
		compacted := make(map[string]any, len(value))
		for key, child := range value {
			next := compactNormalizedJSONValue(child)
			if next == nil {
				continue
			}
			compacted[key] = next
		}
		if isEmptySecretSourceMap(compacted) {
			return nil
		}
		return compacted
	case []any:
		compacted := make([]any, len(value))
		for i, child := range value {
			compacted[i] = compactNormalizedJSONValue(child)
		}
		return compacted
	default:
		return raw
	}
}

func isEmptySecretSourceMap(values map[string]any) bool {
	if len(values) != 1 {
		return false
	}
	secretValue, ok := values["secret"]
	if !ok {
		return false
	}
	if secretValue == nil {
		return true
	}
	secretMap, ok := secretValue.(map[string]any)
	if !ok {
		return false
	}
	if len(secretMap) == 0 {
		return true
	}
	secretName, ok := secretMap["secretName"].(string)
	return ok && strings.TrimSpace(secretName) == ""
}

//nolint:gocognit,gocyclo // Recursive projection handles structs, collections, maps, and secret-backed scalar inputs.
func prepareValueForTarget(
	ctx context.Context,
	raw any,
	targetType reflect.Type,
	namespace string,
	credentialClient credhelper.CredentialClient,
	path []string,
) (any, error) {
	if targetType == nil {
		return raw, nil
	}
	for targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}

	switch targetType.Kind() {
	case reflect.Struct:
		rawMap, ok := raw.(map[string]any)
		if !ok {
			return raw, nil
		}
		prepared := make(map[string]any, len(rawMap))
		for key, value := range rawMap {
			prepared[key] = value
		}
		for i := 0; i < targetType.NumField(); i++ {
			fieldType := targetType.Field(i)
			if !fieldType.IsExported() {
				continue
			}
			fieldName := fieldJSONName(fieldType)
			if fieldName == "" {
				continue
			}
			childRaw, ok := rawMap[fieldName]
			if !ok {
				continue
			}
			childPrepared, err := prepareValueForTarget(ctx, childRaw, fieldType.Type, namespace, credentialClient, append(path, fieldName))
			if err != nil {
				return nil, err
			}
			prepared[fieldName] = childPrepared
		}
		return prepared, nil
	case reflect.Slice, reflect.Array:
		items, ok := raw.([]any)
		if !ok {
			return raw, nil
		}
		prepared := make([]any, len(items))
		for index, item := range items {
			next, err := prepareValueForTarget(ctx, item, targetType.Elem(), namespace, credentialClient, path)
			if err != nil {
				return nil, err
			}
			prepared[index] = next
		}
		return prepared, nil
	case reflect.Map:
		values, ok := raw.(map[string]any)
		if !ok {
			return raw, nil
		}
		prepared := make(map[string]any, len(values))
		for key, value := range values {
			next, err := prepareValueForTarget(ctx, value, targetType.Elem(), namespace, credentialClient, path)
			if err != nil {
				return nil, err
			}
			prepared[key] = next
		}
		return prepared, nil
	case reflect.String:
		if resolved, ok, err := maybeResolveSecretInput(ctx, raw, namespace, credentialClient, path); ok || err != nil {
			return resolved, err
		}
	}

	return raw, nil
}

func maybeResolveSecretInput(
	ctx context.Context,
	raw any,
	namespace string,
	credentialClient credhelper.CredentialClient,
	path []string,
) (string, bool, error) {
	secretName, ok := extractSecretName(raw)
	if !ok {
		return "", false, nil
	}
	if secretName == "" {
		return "", true, nil
	}
	if credentialClient == nil {
		return "", false, fmt.Errorf("generated runtime requires a credential client to resolve %s", strings.Join(path, "."))
	}

	secretData, err := credentialClient.GetSecret(ctx, secretName, namespace)
	if err != nil {
		return "", false, err
	}

	key := secretDataKeyForPath(path)
	value, ok := secretData[key]
	if !ok {
		return "", false, fmt.Errorf("secret %q is missing required key %q for %s", secretName, key, strings.Join(path, "."))
	}
	return string(value), true, nil
}

func extractSecretName(raw any) (string, bool) {
	values, ok := raw.(map[string]any)
	if !ok {
		return "", false
	}
	secretValue, ok := values["secret"]
	if !ok {
		return "", false
	}
	secretMap, ok := secretValue.(map[string]any)
	if !ok {
		return "", false
	}
	name, _ := secretMap["secretName"].(string)
	return name, true
}

func secretDataKeyForPath(path []string) string {
	if len(path) == 0 {
		return "password"
	}

	fieldName := strings.ToLower(path[len(path)-1])
	switch fieldName {
	case "walletpassword":
		return "walletPassword"
	}
	if strings.HasSuffix(fieldName, "username") {
		return "username"
	}
	return "password"
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

//nolint:gocognit,gocyclo // OCI responses vary widely, so body extraction handles several structural fallbacks.
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

func stampSecretSourceStatus(resource any) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return
	}

	specField, ok := fieldValue(resourceValue, "Spec")
	if !ok {
		return
	}
	statusField, ok := fieldValue(resourceValue, "Status")
	if !ok {
		return
	}

	copySecretSourceFields(specField, statusField)
}

//nolint:gocyclo // Status stamping recursively walks nested structs to preserve secret-reference mirrors.
func copySecretSourceFields(source reflect.Value, destination reflect.Value) {
	source, destination, ok := secretSourceStructPair(source, destination)
	if !ok {
		return
	}
	for i := 0; i < source.NumField(); i++ {
		copySecretSourceField(source, destination, i)
	}
}

func secretSourceStructPair(source reflect.Value, destination reflect.Value) (reflect.Value, reflect.Value, bool) {
	source = indirectValue(source)
	destination = indirectValue(destination)
	if !source.IsValid() || !destination.IsValid() {
		return reflect.Value{}, reflect.Value{}, false
	}
	if source.Kind() != reflect.Struct || destination.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, false
	}
	return source, destination, true
}

func copySecretSourceField(source reflect.Value, destination reflect.Value, index int) {
	fieldType := source.Type().Field(index)
	if !fieldType.IsExported() {
		return
	}

	sourceField := source.Field(index)
	destinationField, ok := settableFieldByName(destination, fieldType.Name)
	if !ok {
		return
	}
	if copySecretSourceLeaf(sourceField, destinationField) {
		return
	}
	copySecretSourceFields(sourceField, destinationField)
}

func settableFieldByName(value reflect.Value, name string) (reflect.Value, bool) {
	field := value.FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		return reflect.Value{}, false
	}
	return field, true
}

func copySecretSourceLeaf(source reflect.Value, destination reflect.Value) bool {
	if !isSecretSourceType(source.Type()) || destination.Type() != source.Type() {
		return false
	}
	if secretSourceValueIsEmpty(source) {
		destination.Set(reflect.Zero(destination.Type()))
		return true
	}
	destination.Set(source)
	return true
}

func secretSourceValueIsEmpty(value reflect.Value) bool {
	value = indirectValue(value)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return true
	}
	secretField := value.FieldByName("Secret")
	secretField = indirectValue(secretField)
	if !secretField.IsValid() || secretField.Kind() != reflect.Struct {
		return true
	}
	nameField := secretField.FieldByName("SecretName")
	if !nameField.IsValid() || nameField.Kind() != reflect.String {
		return true
	}
	return strings.TrimSpace(nameField.String()) == ""
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func isSecretSourceType(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ == reflect.TypeOf(shared.UsernameSource{}) || typ == reflect.TypeOf(shared.PasswordSource{})
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

//nolint:gocyclo // Formal validation aggregates independent semantic compatibility checks into one report.
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

//nolint:gocyclo // List selection combines preferred-ID, formal semantics, and heuristic matching fallbacks.
func (c ServiceClient[T]) selectListItem(body any, resource T, preferredID string) (any, error) {
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
		return c.selectFormalListItem(items, criteria, preferredID)
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
	default:
		return nil, errResourceNotFound
	}
}

//nolint:gocognit,gocyclo // Formal list matching must evaluate preferred IDs, match fields, and ambiguity handling together.
func (c ServiceClient[T]) selectFormalListItem(items []any, criteria map[string]any, preferredID string) (any, error) {
	return c.selectFormalListItemWithFilter(items, criteria, preferredID, nil)
}

func (c ServiceClient[T]) selectFormalReusableListItem(items []any, criteria map[string]any) (any, error) {
	return c.selectFormalListItemWithFilter(items, criteria, "", c.listItemReusableBeforeCreate)
}

//nolint:gocognit,gocyclo // Formal list matching must evaluate preferred IDs, match fields, lifecycle filters, and ambiguity handling together.
func (c ServiceClient[T]) selectFormalListItemWithFilter(items []any, criteria map[string]any, preferredID string, accept func(map[string]any) bool) (any, error) {
	matchFields := []string{}
	if c.config.Semantics != nil && c.config.Semantics.List != nil {
		matchFields = append(matchFields, c.config.Semantics.List.MatchFields...)
	}

	var matches []any
	comparedReusable := false
	for _, item := range items {
		values := jsonMap(item)
		if preferredID != "" && preferredID == firstNonEmpty(values, "id", "ocid") {
			matches = append(matches, item)
			continue
		}

		comparedFields := 0
		comparedReusableFields := 0
		matched := true
		for _, field := range matchFields {
			expected, ok := lookupMeaningfulValue(criteria, field)
			if !ok {
				continue
			}
			comparedFields++
			if isReusableListMatchField(field) {
				comparedReusableFields++
			}
			actual, ok := lookupMeaningfulValue(values, field)
			if !ok || !valuesEqual(expected, actual) {
				matched = false
				break
			}
		}
		if comparedFields == 0 || comparedReusableFields == 0 {
			continue
		}
		comparedReusable = true
		if matched && (accept == nil || accept(values)) {
			matches = append(matches, item)
		}
	}

	switch {
	case len(matches) == 1:
		return matches[0], nil
	case len(matches) > 1:
		return nil, fmt.Errorf("%s formal list semantics returned multiple matching resources", c.config.Kind)
	case comparedReusable || preferredID != "":
		return nil, errResourceNotFound
	default:
		return nil, errResourceNotFound
	}
}

func (c ServiceClient[T]) listItemReusableBeforeCreate(values map[string]any) bool {
	reusableStates := c.reusableLifecycleStates()
	if len(reusableStates) == 0 {
		return true
	}

	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status", "state"))
	if lifecycleState == "" {
		return false
	}
	return containsString(reusableStates, lifecycleState)
}

func (c ServiceClient[T]) reusableLifecycleStates() []string {
	if c.config.Semantics == nil {
		return nil
	}

	states := appendUniqueStrings([]string{}, c.config.Semantics.Lifecycle.ActiveStates...)
	states = appendUniqueStrings(states, c.config.Semantics.Lifecycle.ProvisioningStates...)
	states = appendUniqueStrings(states, c.config.Semantics.Lifecycle.UpdatingStates...)
	return states
}

//nolint:gocognit,gocyclo // OCI list bodies expose item slices through several schema shapes.
func listItems(body any, responseItemsField string) ([]any, error) {
	value := reflect.ValueOf(body)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, errResourceNotFound
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return nil, fmt.Errorf("OCI list body must be a struct or slice, got %T", body)
	}
	if value.Kind() == reflect.Slice {
		return sliceValues(value), nil
	}
	if value.Kind() != reflect.Struct {
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
	normalized, err := normalizeJSONValue(value)
	if err != nil {
		return nil
	}
	decoded, ok := normalized.(map[string]any)
	if !ok {
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
		next, ok := lookupPathSegment(mapValue, segment)
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func lookupPathSegment(values map[string]any, segment string) (any, bool) {
	if value, ok := values[segment]; ok {
		return value, true
	}

	normalizedSegment := normalizePathSegment(segment)
	for key, value := range values {
		if normalizePathSegment(key) == normalizedSegment {
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

func comparableDiffPaths(specValues map[string]any, statusValues map[string]any, prefix string) []string {
	if specValues == nil || statusValues == nil {
		return nil
	}

	keys := meaningfulSortedKeys(specValues)
	paths := make([]string, 0, len(keys))
	for _, key := range keys {
		paths = append(paths, comparableDiffPathsForKey(specValues, statusValues, prefix, key)...)
	}

	return paths
}

func meaningfulSortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if meaningfulValue(value) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func comparableDiffPathsForKey(specValues map[string]any, statusValues map[string]any, prefix string, key string) []string {
	specValue := specValues[key]
	statusValue, ok := statusValues[key]
	if !ok {
		return nil
	}

	path := key
	if prefix != "" {
		path = prefix + "." + key
	}

	specMap, specIsMap := specValue.(map[string]any)
	statusMap, statusIsMap := statusValue.(map[string]any)
	if specIsMap && statusIsMap {
		return comparableDiffPaths(specMap, statusMap, path)
	}
	if !valuesEqual(specValue, statusValue) {
		return []string{path}
	}
	return nil
}

func pathCoveredByAny(path string, semanticPaths []string) bool {
	for _, semanticPath := range semanticPaths {
		if pathCoveredBy(path, semanticPath) {
			return true
		}
	}
	return false
}

func pathCoveredBy(path string, semanticPath string) bool {
	path = normalizePath(path)
	semanticPath = normalizePath(semanticPath)
	if path == "" || semanticPath == "" {
		return false
	}
	return path == semanticPath ||
		strings.HasPrefix(path, semanticPath+".") ||
		strings.HasPrefix(semanticPath, path+".")
}

func isReusableListMatchField(path string) bool {
	switch normalizePathSegment(lastPathSegment(path)) {
	case "displayname", "id", "metadataname", "name", "ocid":
		return true
	default:
		return false
	}
}

func lastPathSegment(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return path
	}
	return path[lastDot+1:]
}

func normalizePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}

	segments := strings.Split(path, ".")
	for index, segment := range segments {
		segments[index] = normalizePathSegment(segment)
	}
	return strings.Join(segments, ".")
}

func normalizePathSegment(segment string) string {
	segment = strings.ToLower(strings.TrimSpace(segment))
	if strings.HasSuffix(segment, "gbs") {
		return strings.TrimSuffix(segment, "gbs") + "gb"
	}
	return segment
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

//nolint:gocyclo // Camel splitting preserves acronym boundaries and mixed token transitions.
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
