/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicegateway

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

const serviceGatewayRequeueDuration = time.Minute

type serviceGatewayOCIClient interface {
	CreateServiceGateway(ctx context.Context, request coresdk.CreateServiceGatewayRequest) (coresdk.CreateServiceGatewayResponse, error)
	GetServiceGateway(ctx context.Context, request coresdk.GetServiceGatewayRequest) (coresdk.GetServiceGatewayResponse, error)
	UpdateServiceGateway(ctx context.Context, request coresdk.UpdateServiceGatewayRequest) (coresdk.UpdateServiceGatewayResponse, error)
	DeleteServiceGateway(ctx context.Context, request coresdk.DeleteServiceGatewayRequest) (coresdk.DeleteServiceGatewayResponse, error)
}

type serviceGatewayRuntimeClient struct {
	manager *ServiceGatewayServiceManager
	client  serviceGatewayOCIClient
	initErr error
}

type normalizedServiceGatewayService struct {
	serviceID string
}

func init() {
	newServiceGatewayServiceClient = func(manager *ServiceGatewayServiceManager) ServiceGatewayServiceClient {
		sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &serviceGatewayRuntimeClient{
			manager: manager,
			client:  sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize ServiceGateway OCI client: %w", err)
		}
		return runtimeClient
	}
}

func (c *serviceGatewayRuntimeClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.ServiceGateway, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		return c.create(ctx, resource)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isServiceGatewayNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.create(ctx, resource)
		}
		return c.fail(resource, normalizeServiceGatewayOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	switch current.LifecycleState {
	case coresdk.ServiceGatewayLifecycleStateProvisioning, coresdk.ServiceGatewayLifecycleStateTerminating, coresdk.ServiceGatewayLifecycleStateTerminated:
		return c.applyLifecycle(resource, current)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}

	if updateNeeded {
		response, err := c.client.UpdateServiceGateway(ctx, updateRequest)
		if err != nil {
			return c.fail(resource, normalizeServiceGatewayOCIError(err))
		}
		current = response.ServiceGateway
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, current)
}

func (c *serviceGatewayRuntimeClient) Delete(ctx context.Context, resource *corev1beta1.ServiceGateway) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	deleteRequest := coresdk.DeleteServiceGatewayRequest{
		ServiceGatewayId: common.String(trackedID),
	}
	if _, err := c.client.DeleteServiceGateway(ctx, deleteRequest); err != nil {
		if isServiceGatewayNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeServiceGatewayOCIError(err)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isServiceGatewayNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, normalizeServiceGatewayOCIError(err)
	}

	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}
	c.markTerminating(resource, current)
	return false, nil
}

func (c *serviceGatewayRuntimeClient) create(ctx context.Context, resource *corev1beta1.ServiceGateway) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateServiceGatewayRequest{
		CreateServiceGatewayDetails: buildCreateServiceGatewayDetails(resource.Spec),
	}

	response, err := c.client.CreateServiceGateway(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeServiceGatewayOCIError(err))
	}

	if err := c.projectStatus(resource, response.ServiceGateway); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.ServiceGateway)
}

func (c *serviceGatewayRuntimeClient) get(ctx context.Context, ocid string) (coresdk.ServiceGateway, error) {
	response, err := c.client.GetServiceGateway(ctx, coresdk.GetServiceGatewayRequest{
		ServiceGatewayId: common.String(ocid),
	})
	if err != nil {
		return coresdk.ServiceGateway{}, err
	}
	return response.ServiceGateway, nil
}

func (c *serviceGatewayRuntimeClient) buildUpdateRequest(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) (coresdk.UpdateServiceGatewayRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateServiceGatewayRequest{}, false, fmt.Errorf("current ServiceGateway does not expose an OCI identifier")
	}

	if err := validateServiceGatewayCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateServiceGatewayRequest{}, false, err
	}

	updateDetails := coresdk.UpdateServiceGatewayDetails{}
	updateNeeded := false

	if !boolPtrEqual(current.BlockTraffic, resource.Spec.BlockTraffic) {
		updateDetails.BlockTraffic = common.Bool(resource.Spec.BlockTraffic)
		updateNeeded = true
	}
	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	desiredFreeformTags := desiredFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}
	desiredDefinedTags := desiredDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}
	if !stringPtrEqual(current.RouteTableId, resource.Spec.RouteTableId) {
		updateDetails.RouteTableId = common.String(resource.Spec.RouteTableId)
		updateNeeded = true
	}

	desiredServices := convertSpecServicesToOCI(resource.Spec.Services)
	if !normalizedServicesEqual(current.Services, desiredServices) {
		updateDetails.Services = desiredServices
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateServiceGatewayRequest{}, false, nil
	}

	return coresdk.UpdateServiceGatewayRequest{
		ServiceGatewayId:            current.Id,
		UpdateServiceGatewayDetails: updateDetails,
	}, true, nil
}

func buildCreateServiceGatewayDetails(spec corev1beta1.ServiceGatewaySpec) coresdk.CreateServiceGatewayDetails {
	createDetails := coresdk.CreateServiceGatewayDetails{
		CompartmentId: common.String(spec.CompartmentId),
		Services:      convertSpecServicesToOCI(spec.Services),
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
	if spec.RouteTableId != "" {
		createDetails.RouteTableId = common.String(spec.RouteTableId)
	}

	return createDetails
}

func desiredFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredDefinedTagsForUpdate(spec map[string]shared.MapValue, current map[string]map[string]interface{}) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func validateServiceGatewayCreateOnlyDrift(spec corev1beta1.ServiceGatewaySpec, current coresdk.ServiceGateway) error {
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
	return fmt.Errorf("ServiceGateway create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func convertSpecServicesToOCI(services []corev1beta1.ServiceGatewayService) []coresdk.ServiceIdRequestDetails {
	converted := make([]coresdk.ServiceIdRequestDetails, 0, len(services))
	for _, service := range services {
		converted = append(converted, coresdk.ServiceIdRequestDetails{
			ServiceId: common.String(service.ServiceId),
		})
	}
	return converted
}

func convertOCIServicesToStatus(services []coresdk.ServiceIdResponseDetails) []corev1beta1.ServiceGatewayService {
	converted := make([]corev1beta1.ServiceGatewayService, 0, len(services))
	for _, service := range services {
		converted = append(converted, corev1beta1.ServiceGatewayService{
			ServiceId: stringValue(service.ServiceId),
		})
	}
	return converted
}

func normalizedServicesEqual(current []coresdk.ServiceIdResponseDetails, desired []coresdk.ServiceIdRequestDetails) bool {
	return reflect.DeepEqual(normalizeObservedServices(current), normalizeDesiredServices(desired))
}

func normalizeObservedServices(services []coresdk.ServiceIdResponseDetails) []normalizedServiceGatewayService {
	normalized := make([]normalizedServiceGatewayService, 0, len(services))
	for _, service := range services {
		normalized = append(normalized, normalizedServiceGatewayService{
			serviceID: strings.TrimSpace(stringValue(service.ServiceId)),
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].serviceID < normalized[j].serviceID
	})
	return normalized
}

func normalizeDesiredServices(services []coresdk.ServiceIdRequestDetails) []normalizedServiceGatewayService {
	normalized := make([]normalizedServiceGatewayService, 0, len(services))
	for _, service := range services {
		normalized = append(normalized, normalizedServiceGatewayService{
			serviceID: strings.TrimSpace(stringValue(service.ServiceId)),
		})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].serviceID < normalized[j].serviceID
	})
	return normalized
}

func (c *serviceGatewayRuntimeClient) applyLifecycle(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) (servicemanager.OSOKResponse, error) {
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

	message := serviceGatewayLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.ServiceGatewayLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.ServiceGatewayLifecycleStateProvisioning:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: serviceGatewayRequeueDuration}, nil
	case coresdk.ServiceGatewayLifecycleStateTerminating, coresdk.ServiceGatewayLifecycleStateTerminated:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: serviceGatewayRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("ServiceGateway lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *serviceGatewayRuntimeClient) fail(resource *corev1beta1.ServiceGateway, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *serviceGatewayRuntimeClient) markDeleted(resource *corev1beta1.ServiceGateway, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *serviceGatewayRuntimeClient) clearTrackedIdentity(resource *corev1beta1.ServiceGateway) {
	resource.Status = corev1beta1.ServiceGatewayStatus{}
}

func (c *serviceGatewayRuntimeClient) markTerminating(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = serviceGatewayLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *serviceGatewayRuntimeClient) projectStatus(resource *corev1beta1.ServiceGateway, current coresdk.ServiceGateway) error {
	resource.Status = corev1beta1.ServiceGatewayStatus{
		OsokStatus:     resource.Status.OsokStatus,
		BlockTraffic:   boolValue(current.BlockTraffic),
		CompartmentId:  stringValue(current.CompartmentId),
		Id:             stringValue(current.Id),
		LifecycleState: string(current.LifecycleState),
		Services:       convertOCIServicesToStatus(current.Services),
		VcnId:          stringValue(current.VcnId),
		DefinedTags:    convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:    stringValue(current.DisplayName),
		FreeformTags:   cloneStringMap(current.FreeformTags),
		RouteTableId:   stringValue(current.RouteTableId),
		TimeCreated:    sdkTimeString(current.TimeCreated),
	}
	return nil
}

func serviceGatewayLifecycleMessage(current coresdk.ServiceGateway) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "ServiceGateway"
	}
	return fmt.Sprintf("ServiceGateway %s is %s", name, current.LifecycleState)
}

func normalizeServiceGatewayOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isServiceGatewayNotFoundOCI(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
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
