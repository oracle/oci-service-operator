/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subnet

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

const subnetRequeueDuration = time.Minute

type subnetOCIClient interface {
	CreateSubnet(ctx context.Context, request coresdk.CreateSubnetRequest) (coresdk.CreateSubnetResponse, error)
	GetSubnet(ctx context.Context, request coresdk.GetSubnetRequest) (coresdk.GetSubnetResponse, error)
	UpdateSubnet(ctx context.Context, request coresdk.UpdateSubnetRequest) (coresdk.UpdateSubnetResponse, error)
	DeleteSubnet(ctx context.Context, request coresdk.DeleteSubnetRequest) (coresdk.DeleteSubnetResponse, error)
}

type subnetRuntimeClient struct {
	manager *SubnetServiceManager
	client  subnetOCIClient
	initErr error
}

func init() {
	newSubnetServiceClient = func(manager *SubnetServiceManager) SubnetServiceClient {
		sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &subnetRuntimeClient{
			manager: manager,
			client:  sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize Subnet OCI client: %w", err)
		}
		return runtimeClient
	}
}

func (c *subnetRuntimeClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.Subnet, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		return c.create(ctx, resource)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isSubnetNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.create(ctx, resource)
		}
		return c.fail(resource, normalizeSubnetOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	switch current.LifecycleState {
	case coresdk.SubnetLifecycleStateProvisioning, coresdk.SubnetLifecycleStateUpdating, coresdk.SubnetLifecycleStateTerminating:
		return c.applyLifecycle(resource, current)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}

	if updateNeeded {
		response, err := c.client.UpdateSubnet(ctx, updateRequest)
		if err != nil {
			return c.fail(resource, normalizeSubnetOCIError(err))
		}
		current = response.Subnet
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, current)
}

func (c *subnetRuntimeClient) Delete(ctx context.Context, resource *corev1beta1.Subnet) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	deleteRequest := coresdk.DeleteSubnetRequest{
		SubnetId: common.String(trackedID),
	}
	if _, err := c.client.DeleteSubnet(ctx, deleteRequest); err != nil {
		if isSubnetNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeSubnetOCIError(err)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isSubnetNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, normalizeSubnetOCIError(err)
	}

	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}
	switch current.LifecycleState {
	case coresdk.SubnetLifecycleStateTerminated:
		c.markTerminating(resource, current)
		return false, nil
	case coresdk.SubnetLifecycleStateTerminating:
		c.markTerminating(resource, current)
		return false, nil
	default:
		c.markTerminating(resource, current)
		return false, nil
	}
}

func (c *subnetRuntimeClient) create(ctx context.Context, resource *corev1beta1.Subnet) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateSubnetRequest{
		CreateSubnetDetails: buildCreateSubnetDetails(resource.Spec),
	}

	response, err := c.client.CreateSubnet(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeSubnetOCIError(err))
	}

	if err := c.projectStatus(resource, response.Subnet); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.Subnet)
}

func (c *subnetRuntimeClient) get(ctx context.Context, ocid string) (coresdk.Subnet, error) {
	response, err := c.client.GetSubnet(ctx, coresdk.GetSubnetRequest{
		SubnetId: common.String(ocid),
	})
	if err != nil {
		return coresdk.Subnet{}, err
	}
	return response.Subnet, nil
}

func (c *subnetRuntimeClient) buildUpdateRequest(resource *corev1beta1.Subnet, current coresdk.Subnet) (coresdk.UpdateSubnetRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateSubnetRequest{}, false, fmt.Errorf("current Subnet does not expose an OCI identifier")
	}

	if err := validateSubnetCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateSubnetRequest{}, false, err
	}

	updateDetails := coresdk.UpdateSubnetDetails{}
	updateNeeded := false

	if resource.Spec.CidrBlock != "" && !stringPtrEqual(current.CidrBlock, resource.Spec.CidrBlock) {
		updateDetails.CidrBlock = common.String(resource.Spec.CidrBlock)
		updateNeeded = true
	}
	if resource.Spec.DhcpOptionsId != "" && !stringPtrEqual(current.DhcpOptionsId, resource.Spec.DhcpOptionsId) {
		updateDetails.DhcpOptionsId = common.String(resource.Spec.DhcpOptionsId)
		updateNeeded = true
	}
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
	if resource.Spec.RouteTableId != "" && !stringPtrEqual(current.RouteTableId, resource.Spec.RouteTableId) {
		updateDetails.RouteTableId = common.String(resource.Spec.RouteTableId)
		updateNeeded = true
	}
	if resource.Spec.SecurityListIds != nil && !normalizedStringSlicesEqual(current.SecurityListIds, resource.Spec.SecurityListIds) {
		updateDetails.SecurityListIds = normalizeStringSlice(resource.Spec.SecurityListIds)
		updateNeeded = true
	}
	if resource.Spec.Ipv6CidrBlock != "" && !stringPtrEqual(current.Ipv6CidrBlock, resource.Spec.Ipv6CidrBlock) {
		updateDetails.Ipv6CidrBlock = common.String(resource.Spec.Ipv6CidrBlock)
		updateNeeded = true
	}
	if resource.Spec.Ipv6CidrBlocks != nil && !normalizedStringSlicesEqual(current.Ipv6CidrBlocks, resource.Spec.Ipv6CidrBlocks) {
		updateDetails.Ipv6CidrBlocks = normalizeStringSlice(resource.Spec.Ipv6CidrBlocks)
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateSubnetRequest{}, false, nil
	}

	return coresdk.UpdateSubnetRequest{
		SubnetId:            current.Id,
		UpdateSubnetDetails: updateDetails,
	}, true, nil
}

func buildCreateSubnetDetails(spec corev1beta1.SubnetSpec) coresdk.CreateSubnetDetails {
	createDetails := coresdk.CreateSubnetDetails{
		CidrBlock:     common.String(spec.CidrBlock),
		CompartmentId: common.String(spec.CompartmentId),
		VcnId:         common.String(spec.VcnId),
	}

	if spec.AvailabilityDomain != "" {
		createDetails.AvailabilityDomain = common.String(spec.AvailabilityDomain)
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.DhcpOptionsId != "" {
		createDetails.DhcpOptionsId = common.String(spec.DhcpOptionsId)
	}
	if spec.DisplayName != "" {
		createDetails.DisplayName = common.String(spec.DisplayName)
	}
	if spec.DnsLabel != "" {
		createDetails.DnsLabel = common.String(spec.DnsLabel)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = spec.FreeformTags
	}
	if spec.Ipv6CidrBlock != "" {
		createDetails.Ipv6CidrBlock = common.String(spec.Ipv6CidrBlock)
	}
	if len(spec.Ipv6CidrBlocks) > 0 {
		createDetails.Ipv6CidrBlocks = normalizeStringSlice(spec.Ipv6CidrBlocks)
	}
	if spec.ProhibitInternetIngress {
		createDetails.ProhibitInternetIngress = common.Bool(true)
	}
	if spec.ProhibitPublicIpOnVnic {
		createDetails.ProhibitPublicIpOnVnic = common.Bool(true)
	}
	if spec.RouteTableId != "" {
		createDetails.RouteTableId = common.String(spec.RouteTableId)
	}
	if len(spec.SecurityListIds) > 0 {
		createDetails.SecurityListIds = normalizeStringSlice(spec.SecurityListIds)
	}

	return createDetails
}

func validateSubnetCreateOnlyDrift(spec corev1beta1.SubnetSpec, current coresdk.Subnet) error {
	var unsupported []string

	if !stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if !stringCreateOnlyMatches(current.VcnId, spec.VcnId) {
		unsupported = append(unsupported, "vcnId")
	}
	if !stringCreateOnlyMatches(current.AvailabilityDomain, spec.AvailabilityDomain) {
		unsupported = append(unsupported, "availabilityDomain")
	}
	if !stringCreateOnlyMatches(current.DnsLabel, spec.DnsLabel) {
		unsupported = append(unsupported, "dnsLabel")
	}
	if !boolCreateOnlyMatches(current.ProhibitInternetIngress, spec.ProhibitInternetIngress) {
		unsupported = append(unsupported, "prohibitInternetIngress")
	}
	if !boolCreateOnlyMatches(current.ProhibitPublicIpOnVnic, spec.ProhibitPublicIpOnVnic) {
		unsupported = append(unsupported, "prohibitPublicIpOnVnic")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("Subnet create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func normalizedStringSlicesEqual(left []string, right []string) bool {
	return reflect.DeepEqual(normalizeStringSlice(left), normalizeStringSlice(right))
}

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func (c *subnetRuntimeClient) applyLifecycle(resource *corev1beta1.Subnet, current coresdk.Subnet) (servicemanager.OSOKResponse, error) {
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

	message := subnetLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.SubnetLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.SubnetLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: subnetRequeueDuration}, nil
	case coresdk.SubnetLifecycleStateUpdating:
		status.Reason = string(shared.Updating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Updating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: subnetRequeueDuration}, nil
	case coresdk.SubnetLifecycleStateTerminating:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: subnetRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("Subnet lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *subnetRuntimeClient) fail(resource *corev1beta1.Subnet, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *subnetRuntimeClient) markDeleted(resource *corev1beta1.Subnet, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *subnetRuntimeClient) clearTrackedIdentity(resource *corev1beta1.Subnet) {
	resource.Status = corev1beta1.SubnetStatus{}
}

func (c *subnetRuntimeClient) markTerminating(resource *corev1beta1.Subnet, current coresdk.Subnet) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = subnetLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *subnetRuntimeClient) projectStatus(resource *corev1beta1.Subnet, current coresdk.Subnet) error {
	resource.Status = corev1beta1.SubnetStatus{
		OsokStatus:              resource.Status.OsokStatus,
		CidrBlock:               stringValue(current.CidrBlock),
		CompartmentId:           stringValue(current.CompartmentId),
		Id:                      stringValue(current.Id),
		LifecycleState:          string(current.LifecycleState),
		RouteTableId:            stringValue(current.RouteTableId),
		VcnId:                   stringValue(current.VcnId),
		VirtualRouterIp:         stringValue(current.VirtualRouterIp),
		VirtualRouterMac:        stringValue(current.VirtualRouterMac),
		AvailabilityDomain:      stringValue(current.AvailabilityDomain),
		DefinedTags:             convertOCIToStatusDefinedTags(current.DefinedTags),
		DhcpOptionsId:           stringValue(current.DhcpOptionsId),
		DisplayName:             stringValue(current.DisplayName),
		DnsLabel:                stringValue(current.DnsLabel),
		FreeformTags:            cloneStringMap(current.FreeformTags),
		Ipv6CidrBlock:           stringValue(current.Ipv6CidrBlock),
		Ipv6CidrBlocks:          append([]string(nil), current.Ipv6CidrBlocks...),
		Ipv6VirtualRouterIp:     stringValue(current.Ipv6VirtualRouterIp),
		ProhibitInternetIngress: boolValue(current.ProhibitInternetIngress),
		ProhibitPublicIpOnVnic:  boolValue(current.ProhibitPublicIpOnVnic),
		SecurityListIds:         append([]string(nil), current.SecurityListIds...),
		SubnetDomainName:        stringValue(current.SubnetDomainName),
		TimeCreated:             sdkTimeString(current.TimeCreated),
	}
	return nil
}

func subnetLifecycleMessage(current coresdk.Subnet) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "Subnet"
	}
	return fmt.Sprintf("Subnet %s is %s", name, current.LifecycleState)
}

func normalizeSubnetOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isSubnetNotFoundOCI(err error) bool {
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

func boolPtrEqual(actual *bool, expected bool) bool {
	if actual == nil {
		return !expected
	}
	return *actual == expected
}

func boolCreateOnlyMatches(actual *bool, expected bool) bool {
	return boolValue(actual) == expected
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

func boolValue(value *bool) bool {
	if value == nil {
		return false
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
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(input) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if len(values) == 0 {
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	if len(converted) == 0 {
		return nil
	}
	return converted
}
