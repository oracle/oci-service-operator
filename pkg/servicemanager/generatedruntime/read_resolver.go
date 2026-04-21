/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func (c ServiceClient[T]) prepareCreateOrUpdateState(ctx context.Context, resource T) (createOrUpdateState, error) {
	currentID, existingResponse, resolvedBeforeCreate, err := c.resolveCurrentResource(ctx, resource)
	if err != nil {
		return createOrUpdateState{}, err
	}

	currentID, liveResponse, err := c.loadLiveMutationResponse(ctx, resource, currentID, existingResponse, resolvedBeforeCreate)
	if err != nil {
		return createOrUpdateState{}, err
	}

	return createOrUpdateState{currentID: currentID, liveResponse: liveResponse}, nil
}

func (c ServiceClient[T]) resolveCurrentResource(ctx context.Context, resource T) (string, any, bool, error) {
	currentID := c.currentID(resource)
	existingResponse, trackedIDStale, err := c.resolveExistingBeforeCreate(ctx, resource)
	if err != nil {
		return "", nil, false, err
	}

	currentID, resolvedBeforeCreate := c.resolveTrackedCurrentID(resource, currentID, existingResponse, trackedIDStale)
	return currentID, existingResponse, resolvedBeforeCreate, nil
}

func (c ServiceClient[T]) resolveTrackedCurrentID(resource T, currentID string, existingResponse any, trackedIDStale bool) (string, bool) {
	originalCurrentID := currentID
	if existingResponse != nil {
		resolvedID := responseID(existingResponse)
		if currentID == "" && resolvedID != "" {
			return resolvedID, true
		}
		if trackedIDStale && resolvedID != "" && resolvedID != originalCurrentID {
			return resolvedID, true
		}
	}
	if trackedIDStale {
		return "", false
	}
	return currentID, false
}

func (c ServiceClient[T]) trackedStatusIDCanBeClearedAfterGetNotFound(resource T, preferredID string) bool {
	if preferredID == "" || !c.usesStatusOnlyCurrentID(resource, preferredID) || c.config.Get == nil {
		return false
	}

	values, err := lookupValues(resource)
	if err != nil {
		return false
	}

	if len(c.config.Get.Fields) > 0 {
		for _, field := range c.config.Get.Fields {
			if !requestFieldRequiresResourceID(field, c.idFieldAliases()) {
				continue
			}
			if _, ok := explicitRequestValue(values, field, preferredID); ok {
				return true
			}
		}
		return false
	}

	requestStruct, ok := operationRequestStruct(c.config.Get.NewRequest)
	if !ok {
		return false
	}
	for i := 0; i < requestStruct.NumField(); i++ {
		fieldType, inspect := heuristicGetField(requestStruct, i)
		if !inspect {
			continue
		}
		if containsString(c.idFieldAliases(), requestLookupKey(fieldType)) {
			return true
		}
	}
	return false
}

func (c ServiceClient[T]) readResourceForExistingBeforeCreate(ctx context.Context, resource T) (any, bool, error) {
	state, err := c.prepareReadResourceState(resource, "")
	if err != nil {
		return nil, false, err
	}

	state, response, handled, trackedIDStale, err := c.readResourceWithGetForExistingBeforeCreate(ctx, resource, state)
	if handled {
		return response, trackedIDStale, err
	}

	response, err = c.readResourceWithList(ctx, resource, state, readPhaseCreate)
	return response, trackedIDStale, err
}

func (c ServiceClient[T]) readResourceWithGetForExistingBeforeCreate(ctx context.Context, resource T, state readResourceState) (readResourceState, any, bool, bool, error) {
	if c.config.Get == nil || !c.canInvokeGet(resource, state.readID) {
		return state, nil, false, false, nil
	}

	trackedIDStale := c.trackedStatusIDCanBeClearedAfterGetNotFound(resource, state.readID)
	response, err := c.invoke(ctx, c.config.Get, resource, state.readID, requestBuildOptions{})
	if err == nil {
		return state, response, true, false, nil
	}
	if !isReadNotFound(err) || c.config.List == nil {
		return state, nil, true, false, err
	}

	return c.fallbackReadResourceState(resource, state, readPhaseCreate), nil, false, trackedIDStale, nil
}

func (c ServiceClient[T]) resolveExistingBeforeCreate(ctx context.Context, resource T) (any, bool, error) {
	if skipExistingBeforeCreate(ctx) {
		return nil, false, nil
	}
	if !c.shouldResolveExistingBeforeCreate() {
		return nil, false, nil
	}

	response, trackedIDStale, err := c.readResourceForExistingBeforeCreate(ctx, resource)
	if err == nil {
		return response, trackedIDStale, nil
	}
	if errors.Is(err, errResourceNotFound) {
		return nil, trackedIDStale, nil
	}
	return nil, false, err
}

func (c ServiceClient[T]) loadLiveMutationResponse(ctx context.Context, resource T, currentID string, existingResponse any, resolvedBeforeCreate bool) (string, any, error) {
	liveResponse := existingResponse
	if currentID == "" || !c.requiresLiveMutationAssessment() {
		c.mergeLiveResponseIntoStatus(resource, currentID, liveResponse)
		return currentID, liveResponse, nil
	}

	forceLiveGet := resolvedBeforeCreate && c.config.Get != nil
	if liveResponse == nil || forceLiveGet {
		var err error
		currentID, liveResponse, err = c.readMutationAssessmentResponse(ctx, resource, currentID, forceLiveGet)
		if err != nil {
			return "", nil, err
		}
	}

	c.mergeLiveResponseIntoStatus(resource, currentID, liveResponse)
	return currentID, liveResponse, nil
}

func (c ServiceClient[T]) readMutationAssessmentResponse(ctx context.Context, resource T, currentID string, forceLiveGet bool) (string, any, error) {
	response, err := c.readResourceForMutationValidation(ctx, resource, currentID, forceLiveGet)
	switch {
	case err == nil:
		return currentID, response, nil
	case !errors.Is(err, errResourceNotFound):
		return currentID, nil, err
	case forceLiveGet:
		return "", nil, nil
	default:
		return currentID, nil, nil
	}
}

func (c ServiceClient[T]) mergeLiveResponseIntoStatus(resource T, currentID string, liveResponse any) {
	if currentID != "" && liveResponse != nil {
		_ = mergeResponseIntoStatus(resource, liveResponse)
	}
}

func skipExistingBeforeCreate(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	skip, _ := ctx.Value(skipExistingBeforeCreateContextKey).(bool)
	return skip
}

func (c ServiceClient[T]) shouldResolveExistingBeforeCreate() bool {
	return c.config.Create != nil && c.config.List != nil && c.config.Semantics != nil && c.config.Semantics.List != nil
}

func (c ServiceClient[T]) requiresLiveMutationAssessment() bool {
	return c.config.Semantics != nil &&
		(len(c.config.Semantics.Mutation.ForceNew) > 0 || len(c.config.Semantics.Mutation.Mutable) > 0) &&
		(c.config.Get != nil || c.config.List != nil)
}

func (c ServiceClient[T]) readResource(ctx context.Context, resource T, preferredID string, phase readPhase) (any, error) {
	state, err := c.prepareReadResourceState(resource, preferredID)
	if err != nil {
		return nil, err
	}

	state, response, handled, err := c.readResourceWithGet(ctx, resource, state, phase)
	if handled {
		return response, err
	}
	return c.readResourceWithList(ctx, resource, state, phase)
}

func (c ServiceClient[T]) readResourceForMutationValidation(ctx context.Context, resource T, currentID string, forceLiveGet bool) (any, error) {
	if !forceLiveGet {
		return c.readResource(ctx, resource, currentID, readPhaseUpdate)
	}
	if c.config.Get == nil {
		return nil, fmt.Errorf("%s generated runtime has no OCI Get operation for live mutation validation", c.config.Kind)
	}

	response, err := c.invoke(ctx, c.config.Get, resource, currentID, requestBuildOptions{})
	if err != nil {
		if isReadNotFound(err) {
			return nil, errResourceNotFound
		}
		return nil, err
	}
	return response, nil
}

func (c ServiceClient[T]) prepareReadResourceState(resource T, preferredID string) (readResourceState, error) {
	readID := preferredID
	if readID == "" {
		readID = c.currentID(resource)
	}

	values, err := lookupValues(resource)
	if err != nil {
		return readResourceState{}, err
	}

	return readResourceState{
		values:     values,
		readID:     readID,
		listValues: values,
		listID:     readID,
	}, nil
}

func (c ServiceClient[T]) readResourceWithGet(ctx context.Context, resource T, state readResourceState, phase readPhase) (readResourceState, any, bool, error) {
	if c.config.Get == nil || !c.canInvokeGet(resource, state.readID) {
		return state, nil, false, nil
	}

	response, err := c.invoke(ctx, c.config.Get, resource, state.readID, requestBuildOptions{})
	if err == nil {
		return state, response, true, nil
	}
	if !isReadNotFound(err) || c.config.List == nil {
		return state, nil, true, err
	}

	return c.fallbackReadResourceState(resource, state, phase), nil, false, nil
}

func (c ServiceClient[T]) fallbackReadResourceState(resource T, state readResourceState, phase readPhase) readResourceState {
	if phase != readPhaseDelete && c.usesStatusOnlyCurrentID(resource, state.readID) {
		state.listValues = valuesWithoutAliases(state.values, c.idFieldAliases())
		state.listID = ""
	}
	return state
}

func (c ServiceClient[T]) readResourceWithList(ctx context.Context, resource T, state readResourceState, phase readPhase) (any, error) {
	if c.config.List == nil {
		return nil, fmt.Errorf("%s generated runtime has no readable OCI operation", c.config.Kind)
	}

	response, err := c.invokeWithValues(ctx, c.config.List, resource, state.listValues, state.listID, requestBuildOptions{})
	if err != nil {
		return nil, err
	}

	body, ok := responseBody(response)
	if !ok {
		return nil, fmt.Errorf("%s list response did not expose a body payload", c.config.Kind)
	}
	return c.selectListItem(body, state.listValues, state.listID, phase)
}

func (c ServiceClient[T]) canInvokeExplicitGet(values map[string]any, preferredID string) bool {
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

func (c ServiceClient[T]) canInvokeHeuristicGet(values map[string]any, preferredID string) bool {
	requestStruct, ok := operationRequestStruct(c.config.Get.NewRequest)
	if !ok {
		return true
	}

	for i := 0; i < requestStruct.NumField(); i++ {
		fieldType, inspect := heuristicGetField(requestStruct, i)
		if !inspect {
			continue
		}
		if !c.canPopulateHeuristicGetField(values, preferredID, fieldType) {
			return false
		}
	}

	return true
}

func heuristicGetField(requestStruct reflect.Value, index int) (reflect.StructField, bool) {
	fieldValue := requestStruct.Field(index)
	fieldType := requestStruct.Type().Field(index)
	if !fieldValue.CanSet() || fieldType.Name == "RequestMetadata" {
		return reflect.StructField{}, false
	}

	switch fieldType.Tag.Get("contributesTo") {
	case "header", "binary", "body":
		return reflect.StructField{}, false
	default:
		return fieldType, true
	}
}

func (c ServiceClient[T]) canPopulateHeuristicGetField(values map[string]any, preferredID string, fieldType reflect.StructField) bool {
	lookupKey := requestLookupKey(fieldType)
	if !containsString(c.idFieldAliases(), lookupKey) || preferredID != "" {
		return true
	}
	_, ok := lookupValueByPaths(values, lookupKey)
	return ok
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
		return c.canInvokeExplicitGet(values, preferredID)
	}
	return c.canInvokeHeuristicGet(values, preferredID)
}

func (c ServiceClient[T]) currentID(resource T) string {
	return c.statusID(resource)
}

func (c ServiceClient[T]) usesStatusOnlyCurrentID(resource T, currentID string) bool {
	if currentID == "" {
		return false
	}
	return currentID == c.statusID(resource) && c.specID(resource) == ""
}

func (c ServiceClient[T]) statusID(resource T) string {
	status, err := osokStatus(resource)
	if err == nil && status.Ocid != "" {
		return string(status.Ocid)
	}

	statusValue, err := statusStruct(resource)
	if err != nil {
		return ""
	}
	return firstNonEmpty(jsonMap(statusValue.Interface()), c.idFieldAliases()...)
}

func (c ServiceClient[T]) specID(resource T) string {
	return firstNonEmpty(jsonMap(specValue(resource)), c.idFieldAliases()...)
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

func (c ServiceClient[T]) selectListItem(body any, criteria map[string]any, preferredID string, phase readPhase) (any, error) {
	items, err := listItems(body, c.listResponseItemsField())
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errResourceNotFound
	}
	if c.config.Semantics != nil && c.config.Semantics.List != nil {
		return c.selectFormalListItem(items, criteriaWithPreferredID(criteria, preferredID), preferredID, phase)
	}
	return c.selectHeuristicListItem(items, criteriaWithPreferredID(criteria, preferredID))
}

func (c ServiceClient[T]) listResponseItemsField() string {
	if c.config.Semantics == nil || c.config.Semantics.List == nil {
		return ""
	}
	return c.config.Semantics.List.ResponseItemsField
}

func criteriaWithPreferredID(criteria map[string]any, preferredID string) map[string]any {
	if preferredID == "" {
		return criteria
	}
	cloned := make(map[string]any, len(criteria)+2)
	for key, value := range criteria {
		cloned[key] = value
	}
	cloned["id"] = preferredID
	cloned["ocid"] = preferredID
	return cloned
}

func (c ServiceClient[T]) selectHeuristicListItem(items []any, criteria map[string]any) (any, error) {
	targetID := firstNonEmpty(criteria, "ocid", "id")
	targetName := firstNonEmpty(criteria, "name", "metadataName")
	targetDisplayName := firstNonEmpty(criteria, "displayName")
	matches := heuristicListMatches(items, targetID, targetName, targetDisplayName)

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

func heuristicListMatches(items []any, targetID string, targetName string, targetDisplayName string) []any {
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
	return matches
}

func (c ServiceClient[T]) selectFormalListItem(items []any, criteria map[string]any, preferredID string, phase readPhase) (any, error) {
	matches, comparedAny := c.formalListMatches(items, criteria, preferredID)
	matches = c.filterPhaseMatches(matches, phase)
	return c.resolveFormalListMatch(matches, comparedAny, preferredID)
}

func (c ServiceClient[T]) formalListMatches(items []any, criteria map[string]any, preferredID string) ([]any, bool) {
	matchFields := append([]string(nil), c.config.Semantics.List.MatchFields...)
	var matches []any
	comparedAny := false

	for _, item := range items {
		matched, compared := formalListItemMatch(item, criteria, preferredID, matchFields)
		comparedAny = comparedAny || compared
		if matched {
			matches = append(matches, item)
		}
	}
	return matches, comparedAny
}

func formalListItemMatch(item any, criteria map[string]any, preferredID string, matchFields []string) (bool, bool) {
	values := jsonMap(item)
	if preferredID != "" && preferredID == firstNonEmpty(values, "id", "ocid") {
		return true, false
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
	return comparedFields > 0, comparedFields > 0
}

func (c ServiceClient[T]) resolveFormalListMatch(matches []any, comparedAny bool, preferredID string) (any, error) {
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

	if category, ok := c.formalLifecycleCategory(state); ok {
		return category
	}
	return heuristicLifecycleCategory(state)
}

func (c ServiceClient[T]) formalLifecycleCategory(state string) (lifecycleCategory, bool) {
	if c.config.Semantics == nil {
		return "", false
	}

	switch {
	case containsString(c.config.Semantics.Lifecycle.ProvisioningStates, state):
		return lifecycleCategoryProvisioning, true
	case containsString(c.config.Semantics.Lifecycle.UpdatingStates, state):
		return lifecycleCategoryUpdating, true
	case containsString(c.config.Semantics.Lifecycle.ActiveStates, state):
		return lifecycleCategoryActive, true
	case containsString(c.config.Semantics.Delete.PendingStates, state):
		return lifecycleCategoryDeleting, true
	case containsString(c.config.Semantics.Delete.TerminalStates, state):
		return lifecycleCategoryDeleted, true
	default:
		return "", false
	}
}

func heuristicLifecycleCategory(state string) lifecycleCategory {
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
	value, err := listBodyStruct(body)
	if err != nil {
		return nil, err
	}
	if value.Kind() == reflect.Slice {
		return sliceValues(value), nil
	}

	if items, ok, err := configuredListItems(value, responseItemsField); ok || err != nil {
		return items, err
	}
	if items, ok := structSliceField(value, "Items"); ok {
		return items, nil
	}
	if items, ok := firstSliceListItems(value); ok {
		return items, nil
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

func valuesWithoutAliases(values map[string]any, aliases []string) map[string]any {
	filtered := make(map[string]any, len(values))
	for key, value := range values {
		if matchesAnyAlias(key, aliases) {
			continue
		}
		filtered[key] = value
	}
	return filtered
}

func matchesAnyAlias(key string, aliases []string) bool {
	for _, alias := range aliases {
		if strings.EqualFold(key, alias) || lowerCamel(key) == lowerCamel(alias) {
			return true
		}
	}
	return false
}
