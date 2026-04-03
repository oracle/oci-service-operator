/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dataflowsdk "github.com/oracle/oci-go-sdk/v65/dataflow"
	dataflowv1beta1 "github.com/oracle/oci-service-operator/api/dataflow/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const applicationRequeueDuration = time.Minute

type applicationOCIClient interface {
	CreateApplication(ctx context.Context, request dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error)
	GetApplication(ctx context.Context, request dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error)
	UpdateApplication(ctx context.Context, request dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error)
	DeleteApplication(ctx context.Context, request dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error)
}

type applicationRuntimeClient struct {
	manager *ApplicationServiceManager
	client  applicationOCIClient
	initErr error
}

func init() {
	newApplicationServiceClient = func(manager *ApplicationServiceManager) ApplicationServiceClient {
		sdkClient, err := dataflowsdk.NewDataFlowClientWithConfigurationProvider(manager.Provider)
		runtimeClient := &applicationRuntimeClient{
			manager: manager,
			client:  sdkClient,
		}
		if err != nil {
			runtimeClient.initErr = fmt.Errorf("initialize Application OCI client: %w", err)
		}
		return runtimeClient
	}
}

func (c *applicationRuntimeClient) CreateOrUpdate(ctx context.Context, resource *dataflowv1beta1.Application, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		return c.create(ctx, resource)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isApplicationReadNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.create(ctx, resource)
		}
		return c.fail(resource, normalizeApplicationOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	if current.LifecycleState == dataflowsdk.ApplicationLifecycleStateDeleted {
		return c.applyLifecycle(resource, current)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}

	if updateNeeded {
		response, err := c.client.UpdateApplication(ctx, updateRequest)
		if err != nil {
			return c.fail(resource, normalizeApplicationOCIError(err))
		}
		current = response.Application
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, current)
}

func (c *applicationRuntimeClient) Delete(ctx context.Context, resource *dataflowv1beta1.Application) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if trackedID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}

	deleteRequest := dataflowsdk.DeleteApplicationRequest{
		ApplicationId: common.String(trackedID),
	}
	if _, err := c.client.DeleteApplication(ctx, deleteRequest); err != nil {
		if isApplicationDeleteNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeApplicationOCIError(err)
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isApplicationDeleteNotFoundOCI(err) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, normalizeApplicationOCIError(err)
	}

	if err := c.projectStatus(resource, current); err != nil {
		return false, err
	}
	c.markTerminating(resource, current)
	return false, nil
}

func (c *applicationRuntimeClient) create(ctx context.Context, resource *dataflowv1beta1.Application) (servicemanager.OSOKResponse, error) {
	createDetails, err := buildCreateApplicationDetails(resource.Spec)
	if err != nil {
		return c.fail(resource, err)
	}

	response, err := c.client.CreateApplication(ctx, dataflowsdk.CreateApplicationRequest{
		CreateApplicationDetails: createDetails,
	})
	if err != nil {
		return c.fail(resource, normalizeApplicationOCIError(err))
	}

	if err := c.projectStatus(resource, response.Application); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.Application)
}

func (c *applicationRuntimeClient) get(ctx context.Context, ocid string) (dataflowsdk.Application, error) {
	response, err := c.client.GetApplication(ctx, dataflowsdk.GetApplicationRequest{
		ApplicationId: common.String(ocid),
	})
	if err != nil {
		return dataflowsdk.Application{}, err
	}
	return response.Application, nil
}

func (c *applicationRuntimeClient) buildUpdateRequest(resource *dataflowv1beta1.Application, current dataflowsdk.Application) (dataflowsdk.UpdateApplicationRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return dataflowsdk.UpdateApplicationRequest{}, false, fmt.Errorf("current Application does not expose an OCI identifier")
	}

	if err := validateApplicationCreateOnlyDrift(resource.Spec, current); err != nil {
		return dataflowsdk.UpdateApplicationRequest{}, false, err
	}

	updateDetails := dataflowsdk.UpdateApplicationDetails{}
	updateNeeded := false

	if resource.Spec.Execute != "" && !stringPtrEqual(current.Execute, resource.Spec.Execute) {
		updateDetails.Execute = common.String(resource.Spec.Execute)
		updateNeeded = true
	}

	if resource.Spec.Execute == "" {
		language, err := applicationLanguageFromSpec(resource.Spec.Language)
		if err != nil {
			return dataflowsdk.UpdateApplicationRequest{}, false, err
		}
		if current.Language != language {
			updateDetails.Language = language
			updateNeeded = true
		}

		if resource.Spec.ClassName != "" && !stringPtrEqual(current.ClassName, resource.Spec.ClassName) {
			updateDetails.ClassName = common.String(resource.Spec.ClassName)
			updateNeeded = true
		}
		if resource.Spec.FileUri != "" && !stringPtrEqual(current.FileUri, resource.Spec.FileUri) {
			updateDetails.FileUri = common.String(resource.Spec.FileUri)
			updateNeeded = true
		}
		if len(resource.Spec.Arguments) > 0 && !reflect.DeepEqual(current.Arguments, resource.Spec.Arguments) {
			updateDetails.Arguments = cloneStringSlice(resource.Spec.Arguments)
			updateNeeded = true
		}
		if len(resource.Spec.Configuration) > 0 && !reflect.DeepEqual(current.Configuration, resource.Spec.Configuration) {
			updateDetails.Configuration = cloneStringMap(resource.Spec.Configuration)
			updateNeeded = true
		}
		if len(resource.Spec.Parameters) > 0 {
			parameters := convertApplicationParameters(resource.Spec.Parameters)
			if !reflect.DeepEqual(current.Parameters, parameters) {
				updateDetails.Parameters = parameters
				updateNeeded = true
			}
		}
	}

	if resource.Spec.SparkVersion != "" && !stringPtrEqual(current.SparkVersion, resource.Spec.SparkVersion) {
		updateDetails.SparkVersion = common.String(resource.Spec.SparkVersion)
		updateNeeded = true
	}
	if resource.Spec.DisplayName != "" && !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.DriverShape != "" && !stringPtrEqual(current.DriverShape, resource.Spec.DriverShape) {
		updateDetails.DriverShape = common.String(resource.Spec.DriverShape)
		updateNeeded = true
	}
	if resource.Spec.ExecutorShape != "" && !stringPtrEqual(current.ExecutorShape, resource.Spec.ExecutorShape) {
		updateDetails.ExecutorShape = common.String(resource.Spec.ExecutorShape)
		updateNeeded = true
	}
	if resource.Spec.NumExecutors != 0 && !intPtrEqual(current.NumExecutors, resource.Spec.NumExecutors) {
		updateDetails.NumExecutors = common.Int(resource.Spec.NumExecutors)
		updateNeeded = true
	}
	if resource.Spec.ArchiveUri != "" && !stringPtrEqual(current.ArchiveUri, resource.Spec.ArchiveUri) {
		updateDetails.ArchiveUri = common.String(resource.Spec.ArchiveUri)
		updateNeeded = true
	}
	if logConfig := convertApplicationLogConfig(resource.Spec.ApplicationLogConfig); logConfig != nil && !reflect.DeepEqual(current.ApplicationLogConfig, logConfig) {
		updateDetails.ApplicationLogConfig = logConfig
		updateNeeded = true
	}
	if resource.Spec.Description != "" && !stringPtrEqual(current.Description, resource.Spec.Description) {
		updateDetails.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}
	if driverShapeConfig := convertShapeConfig(resource.Spec.DriverShapeConfig); driverShapeConfig != nil && !reflect.DeepEqual(current.DriverShapeConfig, driverShapeConfig) {
		updateDetails.DriverShapeConfig = driverShapeConfig
		updateNeeded = true
	}
	if executorShapeConfig := convertExecutorShapeConfig(resource.Spec.ExecutorShapeConfig); executorShapeConfig != nil && !reflect.DeepEqual(current.ExecutorShapeConfig, executorShapeConfig) {
		updateDetails.ExecutorShapeConfig = executorShapeConfig
		updateNeeded = true
	}
	if len(resource.Spec.FreeformTags) > 0 && !reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		updateDetails.FreeformTags = cloneStringMap(resource.Spec.FreeformTags)
		updateNeeded = true
	}
	if len(resource.Spec.DefinedTags) > 0 {
		definedTags := *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, definedTags) {
			updateDetails.DefinedTags = definedTags
			updateNeeded = true
		}
	}
	if resource.Spec.LogsBucketUri != "" && !stringPtrEqual(current.LogsBucketUri, resource.Spec.LogsBucketUri) {
		updateDetails.LogsBucketUri = common.String(resource.Spec.LogsBucketUri)
		updateNeeded = true
	}
	if resource.Spec.MetastoreId != "" && !stringPtrEqual(current.MetastoreId, resource.Spec.MetastoreId) {
		updateDetails.MetastoreId = common.String(resource.Spec.MetastoreId)
		updateNeeded = true
	}
	if resource.Spec.PoolId != "" && !stringPtrEqual(current.PoolId, resource.Spec.PoolId) {
		updateDetails.PoolId = common.String(resource.Spec.PoolId)
		updateNeeded = true
	}
	if resource.Spec.PrivateEndpointId != "" && !stringPtrEqual(current.PrivateEndpointId, resource.Spec.PrivateEndpointId) {
		updateDetails.PrivateEndpointId = common.String(resource.Spec.PrivateEndpointId)
		updateNeeded = true
	}
	if resource.Spec.WarehouseBucketUri != "" && !stringPtrEqual(current.WarehouseBucketUri, resource.Spec.WarehouseBucketUri) {
		updateDetails.WarehouseBucketUri = common.String(resource.Spec.WarehouseBucketUri)
		updateNeeded = true
	}
	if resource.Spec.MaxDurationInMinutes != 0 && !int64PtrEqual(current.MaxDurationInMinutes, resource.Spec.MaxDurationInMinutes) {
		updateDetails.MaxDurationInMinutes = common.Int64(resource.Spec.MaxDurationInMinutes)
		updateNeeded = true
	}
	if resource.Spec.IdleTimeoutInMinutes != 0 && !int64PtrEqual(current.IdleTimeoutInMinutes, resource.Spec.IdleTimeoutInMinutes) {
		updateDetails.IdleTimeoutInMinutes = common.Int64(resource.Spec.IdleTimeoutInMinutes)
		updateNeeded = true
	}

	if !updateNeeded {
		return dataflowsdk.UpdateApplicationRequest{}, false, nil
	}

	return dataflowsdk.UpdateApplicationRequest{
		ApplicationId:            current.Id,
		UpdateApplicationDetails: updateDetails,
	}, true, nil
}

func buildCreateApplicationDetails(spec dataflowv1beta1.ApplicationSpec) (dataflowsdk.CreateApplicationDetails, error) {
	language, err := applicationLanguageFromSpec(spec.Language)
	if err != nil {
		return dataflowsdk.CreateApplicationDetails{}, err
	}

	createDetails := dataflowsdk.CreateApplicationDetails{
		CompartmentId: common.String(spec.CompartmentId),
		DisplayName:   common.String(spec.DisplayName),
		DriverShape:   common.String(spec.DriverShape),
		ExecutorShape: common.String(spec.ExecutorShape),
		Language:      language,
		NumExecutors:  common.Int(spec.NumExecutors),
		SparkVersion:  common.String(spec.SparkVersion),
	}

	if spec.Type != "" {
		applicationType, err := applicationTypeFromSpec(spec.Type)
		if err != nil {
			return dataflowsdk.CreateApplicationDetails{}, err
		}
		createDetails.Type = applicationType
	}
	if spec.ArchiveUri != "" {
		createDetails.ArchiveUri = common.String(spec.ArchiveUri)
	}
	if len(spec.Arguments) > 0 && spec.Execute == "" {
		createDetails.Arguments = cloneStringSlice(spec.Arguments)
	}
	if logConfig := convertApplicationLogConfig(spec.ApplicationLogConfig); logConfig != nil {
		createDetails.ApplicationLogConfig = logConfig
	}
	if spec.ClassName != "" && spec.Execute == "" {
		createDetails.ClassName = common.String(spec.ClassName)
	}
	if len(spec.Configuration) > 0 && spec.Execute == "" {
		createDetails.Configuration = cloneStringMap(spec.Configuration)
	}
	if len(spec.DefinedTags) > 0 {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.Description != "" {
		createDetails.Description = common.String(spec.Description)
	}
	if driverShapeConfig := convertShapeConfig(spec.DriverShapeConfig); driverShapeConfig != nil {
		createDetails.DriverShapeConfig = driverShapeConfig
	}
	if spec.Execute != "" {
		createDetails.Execute = common.String(spec.Execute)
	}
	if executorShapeConfig := convertExecutorShapeConfig(spec.ExecutorShapeConfig); executorShapeConfig != nil {
		createDetails.ExecutorShapeConfig = executorShapeConfig
	}
	if spec.FileUri != "" && spec.Execute == "" {
		createDetails.FileUri = common.String(spec.FileUri)
	}
	if len(spec.FreeformTags) > 0 {
		createDetails.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.LogsBucketUri != "" {
		createDetails.LogsBucketUri = common.String(spec.LogsBucketUri)
	}
	if spec.MetastoreId != "" {
		createDetails.MetastoreId = common.String(spec.MetastoreId)
	}
	if len(spec.Parameters) > 0 && spec.Execute == "" {
		createDetails.Parameters = convertApplicationParameters(spec.Parameters)
	}
	if spec.PoolId != "" {
		createDetails.PoolId = common.String(spec.PoolId)
	}
	if spec.PrivateEndpointId != "" {
		createDetails.PrivateEndpointId = common.String(spec.PrivateEndpointId)
	}
	if spec.WarehouseBucketUri != "" {
		createDetails.WarehouseBucketUri = common.String(spec.WarehouseBucketUri)
	}
	if spec.MaxDurationInMinutes != 0 {
		createDetails.MaxDurationInMinutes = common.Int64(spec.MaxDurationInMinutes)
	}
	if spec.IdleTimeoutInMinutes != 0 {
		createDetails.IdleTimeoutInMinutes = common.Int64(spec.IdleTimeoutInMinutes)
	}

	return createDetails, nil
}

func validateApplicationCreateOnlyDrift(spec dataflowv1beta1.ApplicationSpec, current dataflowsdk.Application) error {
	var unsupported []string

	if !stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if spec.Type != "" {
		applicationType, err := applicationTypeFromSpec(spec.Type)
		if err != nil {
			return err
		}
		if current.Type != applicationType {
			unsupported = append(unsupported, "type")
		}
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("Application create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func (c *applicationRuntimeClient) applyLifecycle(resource *dataflowv1beta1.Application, current dataflowsdk.Application) (servicemanager.OSOKResponse, error) {
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

	message := applicationLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case dataflowsdk.ApplicationLifecycleStateActive, dataflowsdk.ApplicationLifecycleStateInactive:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case dataflowsdk.ApplicationLifecycleStateDeleted:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: applicationRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("Application lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *applicationRuntimeClient) fail(resource *dataflowv1beta1.Application, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *applicationRuntimeClient) markDeleted(resource *dataflowv1beta1.Application, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *applicationRuntimeClient) clearTrackedIdentity(resource *dataflowv1beta1.Application) {
	resource.Status = dataflowv1beta1.ApplicationStatus{}
}

func (c *applicationRuntimeClient) markTerminating(resource *dataflowv1beta1.Application, current dataflowsdk.Application) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = applicationLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *applicationRuntimeClient) projectStatus(resource *dataflowv1beta1.Application, current dataflowsdk.Application) error {
	resource.Status = dataflowv1beta1.ApplicationStatus{
		OsokStatus:           resource.Status.OsokStatus,
		CompartmentId:        stringValue(current.CompartmentId),
		DisplayName:          stringValue(current.DisplayName),
		DriverShape:          stringValue(current.DriverShape),
		ExecutorShape:        stringValue(current.ExecutorShape),
		FileUri:              stringValue(current.FileUri),
		Id:                   stringValue(current.Id),
		Language:             string(current.Language),
		LifecycleState:       string(current.LifecycleState),
		NumExecutors:         intValue(current.NumExecutors),
		OwnerPrincipalId:     stringValue(current.OwnerPrincipalId),
		SparkVersion:         stringValue(current.SparkVersion),
		TimeCreated:          sdkTimeString(current.TimeCreated),
		TimeUpdated:          sdkTimeString(current.TimeUpdated),
		ApplicationLogConfig: statusApplicationLogConfig(current.ApplicationLogConfig),
		ArchiveUri:           stringValue(current.ArchiveUri),
		Arguments:            cloneStringSlice(current.Arguments),
		ClassName:            stringValue(current.ClassName),
		Configuration:        cloneStringMap(current.Configuration),
		DefinedTags:          convertOCIToStatusDefinedTags(current.DefinedTags),
		Description:          stringValue(current.Description),
		DriverShapeConfig:    statusShapeConfig(current.DriverShapeConfig),
		Execute:              stringValue(current.Execute),
		ExecutorShapeConfig:  statusExecutorShapeConfig(current.ExecutorShapeConfig),
		FreeformTags:         cloneStringMap(current.FreeformTags),
		LogsBucketUri:        stringValue(current.LogsBucketUri),
		MetastoreId:          stringValue(current.MetastoreId),
		OwnerUserName:        stringValue(current.OwnerUserName),
		Parameters:           statusApplicationParameters(current.Parameters),
		PoolId:               stringValue(current.PoolId),
		PrivateEndpointId:    stringValue(current.PrivateEndpointId),
		Type:                 string(current.Type),
		WarehouseBucketUri:   stringValue(current.WarehouseBucketUri),
		MaxDurationInMinutes: int64Value(current.MaxDurationInMinutes),
		IdleTimeoutInMinutes: int64Value(current.IdleTimeoutInMinutes),
	}
	return nil
}

func applicationLanguageFromSpec(value string) (dataflowsdk.ApplicationLanguageEnum, error) {
	enumValue, ok := dataflowsdk.GetMappingApplicationLanguageEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported Application language %q", value)
	}
	return enumValue, nil
}

func applicationTypeFromSpec(value string) (dataflowsdk.ApplicationTypeEnum, error) {
	enumValue, ok := dataflowsdk.GetMappingApplicationTypeEnum(strings.TrimSpace(value))
	if !ok {
		return "", fmt.Errorf("unsupported Application type %q", value)
	}
	return enumValue, nil
}

func convertApplicationLogConfig(spec dataflowv1beta1.ApplicationLogConfig) *dataflowsdk.ApplicationLogConfig {
	if strings.TrimSpace(spec.LogGroupId) == "" && strings.TrimSpace(spec.LogId) == "" {
		return nil
	}
	return &dataflowsdk.ApplicationLogConfig{
		LogGroupId: common.String(spec.LogGroupId),
		LogId:      common.String(spec.LogId),
	}
}

func convertShapeConfig(spec dataflowv1beta1.ApplicationDriverShapeConfig) *dataflowsdk.ShapeConfig {
	if spec.Ocpus == 0 && spec.MemoryInGBs == 0 {
		return nil
	}
	shapeConfig := &dataflowsdk.ShapeConfig{}
	if spec.Ocpus != 0 {
		shapeConfig.Ocpus = common.Float32(spec.Ocpus)
	}
	if spec.MemoryInGBs != 0 {
		shapeConfig.MemoryInGBs = common.Float32(spec.MemoryInGBs)
	}
	return shapeConfig
}

func convertExecutorShapeConfig(spec dataflowv1beta1.ApplicationExecutorShapeConfig) *dataflowsdk.ShapeConfig {
	if spec.Ocpus == 0 && spec.MemoryInGBs == 0 {
		return nil
	}
	shapeConfig := &dataflowsdk.ShapeConfig{}
	if spec.Ocpus != 0 {
		shapeConfig.Ocpus = common.Float32(spec.Ocpus)
	}
	if spec.MemoryInGBs != 0 {
		shapeConfig.MemoryInGBs = common.Float32(spec.MemoryInGBs)
	}
	return shapeConfig
}

func convertApplicationParameters(spec []dataflowv1beta1.ApplicationParameter) []dataflowsdk.ApplicationParameter {
	if len(spec) == 0 {
		return nil
	}
	parameters := make([]dataflowsdk.ApplicationParameter, 0, len(spec))
	for _, parameter := range spec {
		parameters = append(parameters, dataflowsdk.ApplicationParameter{
			Name:  common.String(parameter.Name),
			Value: common.String(parameter.Value),
		})
	}
	return parameters
}

func statusApplicationLogConfig(config *dataflowsdk.ApplicationLogConfig) dataflowv1beta1.ApplicationLogConfig {
	if config == nil {
		return dataflowv1beta1.ApplicationLogConfig{}
	}
	return dataflowv1beta1.ApplicationLogConfig{
		LogGroupId: stringValue(config.LogGroupId),
		LogId:      stringValue(config.LogId),
	}
}

func statusShapeConfig(config *dataflowsdk.ShapeConfig) dataflowv1beta1.ApplicationDriverShapeConfig {
	if config == nil {
		return dataflowv1beta1.ApplicationDriverShapeConfig{}
	}
	return dataflowv1beta1.ApplicationDriverShapeConfig{
		Ocpus:       float32Value(config.Ocpus),
		MemoryInGBs: float32Value(config.MemoryInGBs),
	}
}

func statusExecutorShapeConfig(config *dataflowsdk.ShapeConfig) dataflowv1beta1.ApplicationExecutorShapeConfig {
	if config == nil {
		return dataflowv1beta1.ApplicationExecutorShapeConfig{}
	}
	return dataflowv1beta1.ApplicationExecutorShapeConfig{
		Ocpus:       float32Value(config.Ocpus),
		MemoryInGBs: float32Value(config.MemoryInGBs),
	}
}

func statusApplicationParameters(parameters []dataflowsdk.ApplicationParameter) []dataflowv1beta1.ApplicationParameter {
	if len(parameters) == 0 {
		return nil
	}
	statusParameters := make([]dataflowv1beta1.ApplicationParameter, 0, len(parameters))
	for _, parameter := range parameters {
		statusParameters = append(statusParameters, dataflowv1beta1.ApplicationParameter{
			Name:  stringValue(parameter.Name),
			Value: stringValue(parameter.Value),
		})
	}
	return statusParameters
}

func applicationLifecycleMessage(current dataflowsdk.Application) string {
	name := strings.TrimSpace(stringValue(current.DisplayName))
	if name == "" {
		name = strings.TrimSpace(stringValue(current.Id))
	}
	if name == "" {
		name = "Application"
	}
	return fmt.Sprintf("Application %s is %s", name, current.LifecycleState)
}

func normalizeApplicationOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isApplicationReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func isApplicationDeleteNotFoundOCI(err error) bool {
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

func intPtrEqual(actual *int, expected int) bool {
	if actual == nil {
		return expected == 0
	}
	return *actual == expected
}

func int64PtrEqual(actual *int64, expected int64) bool {
	if actual == nil {
		return expected == 0
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

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func float32Value(value *float32) float32 {
	if value == nil {
		return 0
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	cloned := make([]string, len(input))
	copy(cloned, input)
	return cloned
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
