/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package routetable

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	routeTableRequeueDuration      = time.Minute
	routeTableLifecycleStateUpdate = coresdk.RouteTableLifecycleStateEnum("UPDATING")
)

type routeTableOCIClient interface {
	CreateRouteTable(ctx context.Context, request coresdk.CreateRouteTableRequest) (coresdk.CreateRouteTableResponse, error)
	GetRouteTable(ctx context.Context, request coresdk.GetRouteTableRequest) (coresdk.GetRouteTableResponse, error)
	UpdateRouteTable(ctx context.Context, request coresdk.UpdateRouteTableRequest) (coresdk.UpdateRouteTableResponse, error)
	DeleteRouteTable(ctx context.Context, request coresdk.DeleteRouteTableRequest) (coresdk.DeleteRouteTableResponse, error)
}

type routeTableRuntimeClient struct {
	manager *RouteTableServiceManager
	client  routeTableOCIClient
	initErr error
}

type normalizedRouteRule struct {
	networkEntityID string
	destination     string
	destinationType string
	description     string
	routeType       string
}

func init() {
	newRouteTableServiceClient = func(manager *RouteTableServiceManager) RouteTableServiceClient {
		sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &routeTableRuntimeClient{
			manager: manager,
			client:  sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize RouteTable OCI client: %w", err)
		}
		return runtimeClient
	}
}

func (c *routeTableRuntimeClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.RouteTable, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		return c.create(ctx, resource)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isRouteTableNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.create(ctx, resource)
		}
		return c.fail(resource, normalizeRouteTableOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	switch current.LifecycleState {
	case coresdk.RouteTableLifecycleStateProvisioning, routeTableLifecycleStateUpdate, coresdk.RouteTableLifecycleStateTerminating, coresdk.RouteTableLifecycleStateTerminated:
		return c.applyLifecycle(resource, current)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}

	if updateNeeded {
		response, err := c.client.UpdateRouteTable(ctx, updateRequest)
		if err != nil {
			return c.fail(resource, normalizeRouteTableOCIError(err))
		}
		current = response.RouteTable
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, current)
}

func (c *routeTableRuntimeClient) Delete(ctx context.Context, resource *corev1beta1.RouteTable) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	deleteRequest := coresdk.DeleteRouteTableRequest{
		RtId: common.String(trackedID),
	}
	if _, err := c.client.DeleteRouteTable(ctx, deleteRequest); err != nil {
		if isRouteTableNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeRouteTableOCIError(err)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isRouteTableNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, normalizeRouteTableOCIError(err)
	}

	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}
	c.markTerminating(resource, current)
	return false, nil
}

func (c *routeTableRuntimeClient) create(ctx context.Context, resource *corev1beta1.RouteTable) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateRouteTableRequest{
		CreateRouteTableDetails: buildCreateRouteTableDetails(resource.Spec),
	}

	response, err := c.client.CreateRouteTable(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeRouteTableOCIError(err))
	}

	if err := c.projectStatus(resource, response.RouteTable); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.RouteTable)
}

func (c *routeTableRuntimeClient) get(ctx context.Context, ocid string) (coresdk.RouteTable, error) {
	response, err := c.client.GetRouteTable(ctx, coresdk.GetRouteTableRequest{
		RtId: common.String(ocid),
	})
	if err != nil {
		return coresdk.RouteTable{}, err
	}
	return response.RouteTable, nil
}

func (c *routeTableRuntimeClient) buildUpdateRequest(resource *corev1beta1.RouteTable, current coresdk.RouteTable) (coresdk.UpdateRouteTableRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateRouteTableRequest{}, false, fmt.Errorf("current RouteTable does not expose an OCI identifier")
	}

	if err := validateRouteTableCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateRouteTableRequest{}, false, err
	}

	updateDetails := coresdk.UpdateRouteTableDetails{}
	updateNeeded := false

	if resource.Spec.DisplayName != "" && !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		updateDetails.FreeformTags = resource.Spec.FreeformTags
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			updateDetails.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}

	desiredRouteRules := convertSpecRouteRulesToOCI(resource.Spec.RouteRules)
	if !normalizedRouteRulesEqual(current.RouteRules, desiredRouteRules) {
		updateDetails.RouteRules = desiredRouteRules
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateRouteTableRequest{}, false, nil
	}

	return coresdk.UpdateRouteTableRequest{
		RtId:                    current.Id,
		UpdateRouteTableDetails: updateDetails,
	}, true, nil
}

func buildCreateRouteTableDetails(spec corev1beta1.RouteTableSpec) coresdk.CreateRouteTableDetails {
	createDetails := coresdk.CreateRouteTableDetails{
		CompartmentId: common.String(spec.CompartmentId),
		RouteRules:    convertSpecRouteRulesToOCI(spec.RouteRules),
		VcnId:         common.String(spec.VcnId),
	}

	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.DisplayName != "" {
		createDetails.DisplayName = common.String(spec.DisplayName)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = spec.FreeformTags
	}

	return createDetails
}

func validateRouteTableCreateOnlyDrift(spec corev1beta1.RouteTableSpec, current coresdk.RouteTable) error {
	var unsupported []string

	if !stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if !stringCreateOnlyMatches(current.VcnId, spec.VcnId) {
		unsupported = append(unsupported, "vcnId")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("RouteTable create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func convertSpecRouteRulesToOCI(rules []corev1beta1.RouteTableRouteRule) []coresdk.RouteRule {
	converted := make([]coresdk.RouteRule, 0, len(rules))
	for _, rule := range rules {
		convertedRule := coresdk.RouteRule{
			NetworkEntityId: common.String(rule.NetworkEntityId),
		}
		if rule.CidrBlock != "" {
			convertedRule.CidrBlock = common.String(rule.CidrBlock)
		}
		if rule.Destination != "" {
			convertedRule.Destination = common.String(rule.Destination)
		}
		if rule.DestinationType != "" {
			convertedRule.DestinationType = coresdk.RouteRuleDestinationTypeEnum(strings.TrimSpace(rule.DestinationType))
		}
		if rule.Description != "" {
			convertedRule.Description = common.String(rule.Description)
		}
		if rule.RouteType != "" {
			convertedRule.RouteType = coresdk.RouteRuleRouteTypeEnum(strings.TrimSpace(rule.RouteType))
		}
		converted = append(converted, convertedRule)
	}
	return converted
}

func convertOCIRouteRulesToStatus(rules []coresdk.RouteRule) []corev1beta1.RouteTableRouteRule {
	converted := make([]corev1beta1.RouteTableRouteRule, 0, len(rules))
	for _, rule := range rules {
		converted = append(converted, corev1beta1.RouteTableRouteRule{
			NetworkEntityId: stringValue(rule.NetworkEntityId),
			CidrBlock:       stringValue(rule.CidrBlock),
			Destination:     stringValue(rule.Destination),
			DestinationType: string(rule.DestinationType),
			Description:     stringValue(rule.Description),
			RouteType:       string(rule.RouteType),
		})
	}
	return converted
}

func normalizedRouteRulesEqual(current []coresdk.RouteRule, desired []coresdk.RouteRule) bool {
	return reflect.DeepEqual(
		normalizeObservedRouteRules(current, desired),
		normalizeRouteRules(desired),
	)
}

func normalizeObservedRouteRules(current []coresdk.RouteRule, desired []coresdk.RouteRule) []normalizedRouteRule {
	desiredNormalized := normalizeRouteRules(desired)
	desiredSet := make(map[normalizedRouteRule]int, len(desiredNormalized))
	for _, rule := range desiredNormalized {
		desiredSet[rule]++
	}

	filtered := make([]coresdk.RouteRule, 0, len(current))
	for _, rule := range current {
		normalized := normalizeRouteRule(rule)
		if normalized.routeType == string(coresdk.RouteRuleRouteTypeLocal) && desiredSet[normalized] == 0 {
			continue
		}
		filtered = append(filtered, rule)
	}

	return normalizeRouteRules(filtered)
}

func normalizeRouteRules(rules []coresdk.RouteRule) []normalizedRouteRule {
	normalized := make([]normalizedRouteRule, 0, len(rules))
	for _, rule := range rules {
		normalized = append(normalized, normalizeRouteRule(rule))
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalizedRouteRuleKey(normalized[i]) < normalizedRouteRuleKey(normalized[j])
	})
	return normalized
}

func normalizeRouteRule(rule coresdk.RouteRule) normalizedRouteRule {
	routeType := strings.ToUpper(strings.TrimSpace(string(rule.RouteType)))
	if routeType == "" {
		routeType = string(coresdk.RouteRuleRouteTypeStatic)
	}

	destination := strings.TrimSpace(stringValue(rule.Destination))
	if destination == "" {
		destination = strings.TrimSpace(stringValue(rule.CidrBlock))
	}

	destinationType := strings.ToUpper(strings.TrimSpace(string(rule.DestinationType)))
	if destinationType == "" && strings.TrimSpace(stringValue(rule.CidrBlock)) != "" {
		destinationType = string(coresdk.RouteRuleDestinationTypeCidrBlock)
	}

	return normalizedRouteRule{
		networkEntityID: strings.TrimSpace(stringValue(rule.NetworkEntityId)),
		destination:     destination,
		destinationType: destinationType,
		description:     strings.TrimSpace(stringValue(rule.Description)),
		routeType:       routeType,
	}
}

func normalizedRouteRuleKey(rule normalizedRouteRule) string {
	return strings.Join([]string{
		rule.networkEntityID,
		rule.destination,
		rule.destinationType,
		rule.description,
		rule.routeType,
	}, "\x00")
}

func (c *routeTableRuntimeClient) applyLifecycle(resource *corev1beta1.RouteTable, current coresdk.RouteTable) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	now := time.Now()
	if status.CreatedAt == nil && current.Id != nil && strings.TrimSpace(*current.Id) != "" {
		createdAt := metav1Time(now)
		status.CreatedAt = &createdAt
	}
	updatedAt := metav1Time(now)
	status.UpdatedAt = &updatedAt
	if current.Id != nil {
		status.Ocid = shared.OCID(*current.Id)
	}

	message := routeTableLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.RouteTableLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.RouteTableLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: routeTableRequeueDuration}, nil
	case routeTableLifecycleStateUpdate:
		status.Reason = string(shared.Updating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Updating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: routeTableRequeueDuration}, nil
	case coresdk.RouteTableLifecycleStateTerminating, coresdk.RouteTableLifecycleStateTerminated:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: routeTableRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("RouteTable lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *routeTableRuntimeClient) fail(resource *corev1beta1.RouteTable, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *routeTableRuntimeClient) markDeleted(resource *corev1beta1.RouteTable, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *routeTableRuntimeClient) clearTrackedIdentity(resource *corev1beta1.RouteTable) {
	resource.Status = corev1beta1.RouteTableStatus{}
}

func (c *routeTableRuntimeClient) markTerminating(resource *corev1beta1.RouteTable, current coresdk.RouteTable) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = routeTableLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *routeTableRuntimeClient) projectStatus(resource *corev1beta1.RouteTable, current coresdk.RouteTable) error {
	resource.Status = corev1beta1.RouteTableStatus{
		OsokStatus:     resource.Status.OsokStatus,
		CompartmentId:  stringValue(current.CompartmentId),
		Id:             stringValue(current.Id),
		LifecycleState: string(current.LifecycleState),
		RouteRules:     convertOCIRouteRulesToStatus(current.RouteRules),
		VcnId:          stringValue(current.VcnId),
		DefinedTags:    convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:    stringValue(current.DisplayName),
		FreeformTags:   cloneStringMap(current.FreeformTags),
		TimeCreated:    sdkTimeString(current.TimeCreated),
	}
	return nil
}

func routeTableLifecycleMessage(current coresdk.RouteTable) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "RouteTable"
	}
	return fmt.Sprintf("RouteTable %s is %s", name, current.LifecycleState)
}

func normalizeRouteTableOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isRouteTableNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func stringCreateOnlyMatches(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func metav1Time(t time.Time) metav1.Time {
	return metav1.NewTime(t)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		convertedValues := make(shared.MapValue, len(values))
		for key, value := range values {
			convertedValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedValues
	}
	return converted
}
