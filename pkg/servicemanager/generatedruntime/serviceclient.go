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
}

type Config[T any] struct {
	Kind      string
	SDKName   string
	Log       loggerutil.OSOKLogger
	InitError error

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
	if currentID != "" {
		if c.config.Update != nil {
			if _, err := c.invoke(ctx, c.config.Update, resource, currentID); err != nil {
				return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
			}
		}

		response, err := c.readResource(ctx, resource, currentID)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}
		return c.applySuccess(resource, response, shared.Updating)
	}

	if c.config.Create != nil {
		response, err := c.invoke(ctx, c.config.Create, resource, "")
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}

		followUp := response
		responseID := responseID(response)
		if c.config.Get != nil || c.config.List != nil {
			if refreshed, err := c.readResource(ctx, resource, responseID); err == nil {
				followUp = refreshed
			} else if !errors.Is(err, errResourceNotFound) {
				return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
			}
		}
		return c.applySuccess(resource, followUp, shared.Provisioning)
	}

	response, err := c.readResource(ctx, resource, "")
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

func (c ServiceClient[T]) readResource(ctx context.Context, resource T, preferredID string) (any, error) {
	if c.config.Get != nil {
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
	if err := buildRequest(request, resource, preferredID, c.idFieldAliases()); err != nil {
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

	conditionType, shouldRequeue, message := classifyLifecycle(response, fallback)
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
	return lookupString(values, "id")
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

func buildRequest(request any, resource any, preferredID string, idAliases []string) error {
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

	requestType := requestStruct.Type()
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

func classifyLifecycle(response any, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool, string) {
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
	items, err := listItems(body)
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

func listItems(body any) ([]any, error) {
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
