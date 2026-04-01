/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vcn

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

const vcnRequeueDuration = time.Minute

type vcnOCIClient interface {
	CreateVcn(ctx context.Context, request coresdk.CreateVcnRequest) (coresdk.CreateVcnResponse, error)
	GetVcn(ctx context.Context, request coresdk.GetVcnRequest) (coresdk.GetVcnResponse, error)
	UpdateVcn(ctx context.Context, request coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error)
	DeleteVcn(ctx context.Context, request coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error)
}

type vcnRuntimeClient struct {
	manager *VcnServiceManager
	client  vcnOCIClient
	initErr error
}

func init() {
	newVcnServiceClient = func(manager *VcnServiceManager) VcnServiceClient {
		sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &vcnRuntimeClient{
			manager: manager,
			client:  sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize Vcn OCI client: %w", err)
		}
		return runtimeClient
	}
}

func (c *vcnRuntimeClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.Vcn, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		return c.create(ctx, resource)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.create(ctx, resource)
		}
		return c.fail(resource, normalizeOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	switch current.LifecycleState {
	case coresdk.VcnLifecycleStateTerminated:
		c.clearTrackedIdentity(resource)
		return c.create(ctx, resource)
	case coresdk.VcnLifecycleStateProvisioning, coresdk.VcnLifecycleStateUpdating, coresdk.VcnLifecycleStateTerminating:
		return c.applyLifecycle(resource, current)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}

	if updateNeeded {
		response, err := c.client.UpdateVcn(ctx, updateRequest)
		if err != nil {
			return c.fail(resource, normalizeOCIError(err))
		}
		current = response.Vcn
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, current)
}

func (c *vcnRuntimeClient) Delete(ctx context.Context, resource *corev1beta1.Vcn) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	deleteRequest := coresdk.DeleteVcnRequest{
		VcnId: common.String(trackedID),
	}
	if _, err := c.client.DeleteVcn(ctx, deleteRequest); err != nil {
		if isNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeOCIError(err)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, normalizeOCIError(err)
	}

	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}
	switch current.LifecycleState {
	case coresdk.VcnLifecycleStateTerminated, coresdk.VcnLifecycleStateTerminating:
		c.markTerminating(resource, current)
		return false, nil
	default:
		c.markTerminating(resource, current)
		return false, nil
	}
}

func (c *vcnRuntimeClient) create(ctx context.Context, resource *corev1beta1.Vcn) (servicemanager.OSOKResponse, error) {
	if err := validateCreateInputs(resource.Spec); err != nil {
		return c.fail(resource, err)
	}

	request := coresdk.CreateVcnRequest{
		CreateVcnDetails: buildCreateVcnDetails(resource.Spec),
	}

	response, err := c.client.CreateVcn(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeOCIError(err))
	}

	if err := c.projectStatus(resource, response.Vcn); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.Vcn)
}

func (c *vcnRuntimeClient) get(ctx context.Context, ocid string) (coresdk.Vcn, error) {
	response, err := c.client.GetVcn(ctx, coresdk.GetVcnRequest{
		VcnId: common.String(ocid),
	})
	if err != nil {
		return coresdk.Vcn{}, err
	}
	return response.Vcn, nil
}

func (c *vcnRuntimeClient) buildUpdateRequest(resource *corev1beta1.Vcn, current coresdk.Vcn) (coresdk.UpdateVcnRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateVcnRequest{}, false, fmt.Errorf("current Vcn does not expose an OCI identifier")
	}

	if err := validateCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateVcnRequest{}, false, err
	}

	updateDetails := coresdk.UpdateVcnDetails{}
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

	if !updateNeeded {
		return coresdk.UpdateVcnRequest{}, false, nil
	}

	return coresdk.UpdateVcnRequest{
		VcnId:            current.Id,
		UpdateVcnDetails: updateDetails,
	}, true, nil
}

func buildCreateVcnDetails(spec corev1beta1.VcnSpec) coresdk.CreateVcnDetails {
	createDetails := coresdk.CreateVcnDetails{
		CompartmentId: common.String(spec.CompartmentId),
	}

	if spec.CidrBlock != "" {
		createDetails.CidrBlock = common.String(spec.CidrBlock)
	}
	if len(spec.CidrBlocks) > 0 {
		createDetails.CidrBlocks = spec.CidrBlocks
	}
	if len(spec.Ipv6PrivateCidrBlocks) > 0 {
		createDetails.Ipv6PrivateCidrBlocks = spec.Ipv6PrivateCidrBlocks
	}
	if spec.IsOracleGuaAllocationEnabled {
		createDetails.IsOracleGuaAllocationEnabled = common.Bool(spec.IsOracleGuaAllocationEnabled)
	}
	if len(spec.Byoipv6CidrDetails) > 0 {
		createDetails.Byoipv6CidrDetails = convertByoipv6CidrDetails(spec.Byoipv6CidrDetails)
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
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
	if spec.IsIpv6Enabled {
		createDetails.IsIpv6Enabled = common.Bool(spec.IsIpv6Enabled)
	}

	return createDetails
}

func convertByoipv6CidrDetails(details []corev1beta1.VcnByoipv6CidrDetail) []coresdk.Byoipv6CidrDetails {
	converted := make([]coresdk.Byoipv6CidrDetails, 0, len(details))
	for _, detail := range details {
		converted = append(converted, coresdk.Byoipv6CidrDetails{
			Byoipv6RangeId: common.String(detail.Byoipv6RangeId),
			Ipv6CidrBlock:  common.String(detail.Ipv6CidrBlock),
		})
	}
	return converted
}

func validateCreateOnlyDrift(spec corev1beta1.VcnSpec, current coresdk.Vcn) error {
	var unsupported []string

	if spec.CompartmentId != "" && !stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if spec.DnsLabel != "" && !stringPtrEqual(current.DnsLabel, spec.DnsLabel) {
		unsupported = append(unsupported, "dnsLabel")
	}
	if spec.CidrBlock != "" && !stringPtrEqual(current.CidrBlock, spec.CidrBlock) {
		unsupported = append(unsupported, "cidrBlock")
	}
	if len(spec.CidrBlocks) > 0 && !normalizedStringSlicesEqual(current.CidrBlocks, spec.CidrBlocks) {
		unsupported = append(unsupported, "cidrBlocks")
	}
	if len(spec.Ipv6PrivateCidrBlocks) > 0 && !normalizedStringSlicesEqual(current.Ipv6PrivateCidrBlocks, spec.Ipv6PrivateCidrBlocks) {
		unsupported = append(unsupported, "ipv6PrivateCidrBlocks")
	}
	if spec.IsIpv6Enabled && len(current.Ipv6CidrBlocks) == 0 && len(current.Ipv6PrivateCidrBlocks) == 0 && len(current.Byoipv6CidrBlocks) == 0 {
		unsupported = append(unsupported, "isIpv6Enabled")
	}
	if spec.IsOracleGuaAllocationEnabled && len(current.Ipv6CidrBlocks) == 0 {
		unsupported = append(unsupported, "isOracleGuaAllocationEnabled")
	}
	if len(spec.Byoipv6CidrDetails) > 0 && !normalizedStringSlicesEqual(current.Byoipv6CidrBlocks, desiredByoipv6Blocks(spec.Byoipv6CidrDetails)) {
		unsupported = append(unsupported, "byoipv6CidrDetails")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("Vcn create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func desiredByoipv6Blocks(details []corev1beta1.VcnByoipv6CidrDetail) []string {
	blocks := make([]string, 0, len(details))
	for _, detail := range details {
		if strings.TrimSpace(detail.Ipv6CidrBlock) != "" {
			blocks = append(blocks, detail.Ipv6CidrBlock)
		}
	}
	return blocks
}

func validateCreateInputs(spec corev1beta1.VcnSpec) error {
	if strings.TrimSpace(spec.CidrBlock) != "" && len(normalizeStringSlice(spec.CidrBlocks)) > 0 {
		return fmt.Errorf("Vcn create input cannot set both cidrBlock and cidrBlocks")
	}
	return nil
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

func (c *vcnRuntimeClient) applyLifecycle(resource *corev1beta1.Vcn, current coresdk.Vcn) (servicemanager.OSOKResponse, error) {
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

	message := lifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.VcnLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.VcnLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: vcnRequeueDuration}, nil
	case coresdk.VcnLifecycleStateUpdating:
		status.Reason = string(shared.Updating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Updating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: vcnRequeueDuration}, nil
	case coresdk.VcnLifecycleStateTerminating:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: vcnRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("Vcn lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *vcnRuntimeClient) fail(resource *corev1beta1.Vcn, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *vcnRuntimeClient) markDeleted(resource *corev1beta1.Vcn, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *vcnRuntimeClient) clearTrackedIdentity(resource *corev1beta1.Vcn) {
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func (c *vcnRuntimeClient) markTerminating(resource *corev1beta1.Vcn, current coresdk.Vcn) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = lifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *vcnRuntimeClient) projectStatus(resource *corev1beta1.Vcn, current coresdk.Vcn) error {
	resource.Status = corev1beta1.VcnStatus{
		OsokStatus:            resource.Status.OsokStatus,
		CidrBlock:             stringValue(current.CidrBlock),
		CidrBlocks:            append([]string(nil), current.CidrBlocks...),
		CompartmentId:         stringValue(current.CompartmentId),
		Id:                    stringValue(current.Id),
		LifecycleState:        string(current.LifecycleState),
		Byoipv6CidrBlocks:     append([]string(nil), current.Byoipv6CidrBlocks...),
		Ipv6PrivateCidrBlocks: append([]string(nil), current.Ipv6PrivateCidrBlocks...),
		DefaultDhcpOptionsId:  stringValue(current.DefaultDhcpOptionsId),
		DefaultRouteTableId:   stringValue(current.DefaultRouteTableId),
		DefaultSecurityListId: stringValue(current.DefaultSecurityListId),
		DefinedTags:           convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:           stringValue(current.DisplayName),
		DnsLabel:              stringValue(current.DnsLabel),
		FreeformTags:          cloneStringMap(current.FreeformTags),
		Ipv6CidrBlocks:        append([]string(nil), current.Ipv6CidrBlocks...),
		TimeCreated:           sdkTimeString(current.TimeCreated),
		VcnDomainName:         stringValue(current.VcnDomainName),
	}
	return nil
}

func lifecycleMessage(current coresdk.Vcn) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "Vcn"
	}
	return fmt.Sprintf("Vcn %s is %s", name, current.LifecycleState)
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

func isNotFoundOCI(err error) bool {
	var unauthorizedAndNotFound errorutil.UnauthorizedAndNotFoundOciError
	if errors.As(err, &unauthorizedAndNotFound) {
		return false
	}

	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		switch serviceErr.GetCode() {
		case "NotFound":
			return true
		case "NotAuthorizedOrNotFound":
			return false
		}
		if serviceErr.GetHTTPStatusCode() == 404 {
			message := strings.ToLower(strings.TrimSpace(serviceErr.GetMessage()))
			if message != "" && strings.Contains(message, "not found") && !strings.Contains(message, "not authorized") {
				return true
			}
		}
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "NotAuthorizedOrNotFound") {
		return false
	}
	return strings.Contains(message, "http status code: 404") &&
		strings.Contains(message, "not found") &&
		!strings.Contains(message, "not authorized")
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
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
