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
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type applicationOCIClient interface {
	CreateApplication(ctx context.Context, request dataflowsdk.CreateApplicationRequest) (dataflowsdk.CreateApplicationResponse, error)
	GetApplication(ctx context.Context, request dataflowsdk.GetApplicationRequest) (dataflowsdk.GetApplicationResponse, error)
	UpdateApplication(ctx context.Context, request dataflowsdk.UpdateApplicationRequest) (dataflowsdk.UpdateApplicationResponse, error)
	DeleteApplication(ctx context.Context, request dataflowsdk.DeleteApplicationRequest) (dataflowsdk.DeleteApplicationResponse, error)
}

var newApplicationOCIClient = func(provider common.ConfigurationProvider) (applicationOCIClient, error) {
	return dataflowsdk.NewDataFlowClientWithConfigurationProvider(provider)
}

type applicationGeneratedRuntimeClient struct {
	delegate ApplicationServiceClient
	client   applicationOCIClient
	initErr  error
	log      loggerutil.OSOKLogger
}

var _ ApplicationServiceClient = (*applicationGeneratedRuntimeClient)(nil)

func init() {
	newApplicationServiceClient = func(manager *ApplicationServiceManager) ApplicationServiceClient {
		sdkClient, err := newApplicationOCIClient(manager.Provider)
		return newApplicationServiceClientWithOCIClient(manager, sdkClient, err)
	}
}

func newApplicationServiceClientWithOCIClient(
	manager *ApplicationServiceManager,
	client applicationOCIClient,
	initErr error,
) ApplicationServiceClient {
	if initErr == nil && client == nil {
		initErr = fmt.Errorf("initialize Application OCI client: client is nil")
	}
	if initErr != nil && !strings.Contains(initErr.Error(), "initialize Application OCI client:") {
		initErr = fmt.Errorf("initialize Application OCI client: %w", initErr)
	}

	return &applicationGeneratedRuntimeClient{
		delegate: newGeneratedApplicationServiceClient(manager, client, initErr),
		client:   client,
		initErr:  initErr,
		log:      manager.Log,
	}
}

func newGeneratedApplicationServiceClient(
	manager *ApplicationServiceManager,
	client applicationOCIClient,
	initErr error,
) ApplicationServiceClient {
	config := generatedruntime.Config[*dataflowv1beta1.Application]{
		Kind:            "Application",
		SDKName:         "Application",
		Log:             manager.Log,
		InitError:       initErr,
		BuildCreateBody: buildGeneratedCreateApplicationBody,
		BuildUpdateBody: buildGeneratedUpdateApplicationBody,
		Semantics: &generatedruntime.Semantics{
			FormalService:     "dataflow",
			FormalSlug:        "application",
			StatusProjection:  "required",
			SecretSideEffects: "none",
			FinalizerPolicy:   "retain-until-confirmed-delete",
			Lifecycle: generatedruntime.LifecycleSemantics{
				ProvisioningStates: []string{},
				UpdatingStates:     []string{},
				ActiveStates:       []string{"ACTIVE", "INACTIVE"},
			},
			Delete: generatedruntime.DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETED"},
				TerminalStates: []string{"NOT_FOUND"},
			},
			Mutation: generatedruntime.MutationSemantics{
				Mutable:       []string{"applicationLogConfig", "archiveUri", "arguments", "className", "configuration", "definedTags", "description", "displayName", "driverShape", "driverShapeConfig", "execute", "executorShape", "executorShapeConfig", "fileUri", "freeformTags", "idleTimeoutInMinutes", "language", "logsBucketUri", "maxDurationInMinutes", "metastoreId", "numExecutors", "parameters", "poolId", "privateEndpointId", "sparkVersion", "warehouseBucketUri"},
				ForceNew:      []string{"compartmentId", "type"},
				ConflictsWith: map[string][]string{},
			},
			Hooks: generatedruntime.HookSet{
				Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
				Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
				Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
			},
			CreateFollowUp: generatedruntime.FollowUpSemantics{},
			UpdateFollowUp: generatedruntime.FollowUpSemantics{},
			DeleteFollowUp: generatedruntime.FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
			},
			AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
			Unsupported:         []generatedruntime.UnsupportedSemantic{},
		},
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &dataflowsdk.CreateApplicationRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.CreateApplication(ctx, *request.(*dataflowsdk.CreateApplicationRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "CreateApplicationDetails", RequestName: "CreateApplicationDetails", Contribution: "body", PreferResourceID: false}},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &dataflowsdk.GetApplicationRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.GetApplication(ctx, *request.(*dataflowsdk.GetApplicationRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "ApplicationId", RequestName: "applicationId", Contribution: "path", PreferResourceID: true}},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &dataflowsdk.UpdateApplicationRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.UpdateApplication(ctx, *request.(*dataflowsdk.UpdateApplicationRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "ApplicationId", RequestName: "applicationId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateApplicationDetails", RequestName: "UpdateApplicationDetails", Contribution: "body", PreferResourceID: false}},
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &dataflowsdk.DeleteApplicationRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return client.DeleteApplication(ctx, *request.(*dataflowsdk.DeleteApplicationRequest))
			},
			Fields: []generatedruntime.RequestField{{FieldName: "ApplicationId", RequestName: "applicationId", Contribution: "path", PreferResourceID: true}},
		},
	}

	return defaultApplicationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*dataflowv1beta1.Application](config),
	}
}

func buildGeneratedCreateApplicationBody(
	_ context.Context,
	resource *dataflowv1beta1.Application,
	_ string,
) (any, error) {
	if resource == nil {
		return dataflowsdk.CreateApplicationDetails{}, fmt.Errorf("Application resource is nil")
	}
	return buildCreateApplicationDetails(resource.Spec)
}

func buildGeneratedUpdateApplicationBody(
	_ context.Context,
	resource *dataflowv1beta1.Application,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return dataflowsdk.UpdateApplicationDetails{}, false, fmt.Errorf("Application resource is nil")
	}

	current, err := applicationFromRuntimeResponse(currentResponse)
	if err != nil {
		return nil, false, err
	}
	return buildUpdateApplicationDetails(resource.Spec, current)
}

func applicationFromRuntimeResponse(currentResponse any) (dataflowsdk.Application, error) {
	switch current := currentResponse.(type) {
	case dataflowsdk.Application:
		return current, nil
	case *dataflowsdk.Application:
		if current == nil {
			return dataflowsdk.Application{}, fmt.Errorf("current Application response is nil")
		}
		return *current, nil
	case dataflowsdk.GetApplicationResponse:
		return current.Application, nil
	case *dataflowsdk.GetApplicationResponse:
		if current == nil {
			return dataflowsdk.Application{}, fmt.Errorf("current Application response is nil")
		}
		return current.Application, nil
	case dataflowsdk.UpdateApplicationResponse:
		return current.Application, nil
	case *dataflowsdk.UpdateApplicationResponse:
		if current == nil {
			return dataflowsdk.Application{}, fmt.Errorf("current Application response is nil")
		}
		return current.Application, nil
	default:
		return dataflowsdk.Application{}, fmt.Errorf("unexpected current Application response type %T", currentResponse)
	}
}

func (c *applicationGeneratedRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *dataflowv1beta1.Application,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return applicationFail(resource, c.log, c.initErr)
	}

	hadTrackedID := currentApplicationID(resource) != ""
	if hadTrackedID {
		current, resetIdentity, err := c.preflightTrackedApplication(ctx, resource)
		if err != nil {
			return applicationFail(resource, c.log, err)
		}
		if resetIdentity {
			hadTrackedID = false
		} else if current != nil {
			if err := projectApplicationStatus(resource, *current); err != nil {
				return applicationFail(resource, c.log, err)
			}
			if err := validateApplicationCreateOnlyDrift(resource.Spec, *current); err != nil {
				return applicationFail(resource, c.log, err)
			}
			if err := validateApplicationUpdateSpec(resource.Spec); err != nil {
				return applicationFail(resource, c.log, err)
			}
		}
	}

	response, err := c.reconcileWithDelegate(ctx, resource, req)
	switch {
	case err == nil && hadTrackedID &&
		strings.EqualFold(resource.Status.LifecycleState, string(dataflowsdk.ApplicationLifecycleStateDeleted)):
		return c.retryCreate(ctx, resource, req)
	case err != nil && hadTrackedID && isApplicationReadNotFoundOCI(err):
		return c.retryCreate(ctx, resource, req)
	default:
		return response, err
	}
}

func (c *applicationGeneratedRuntimeClient) preflightTrackedApplication(
	ctx context.Context,
	resource *dataflowv1beta1.Application,
) (*dataflowsdk.Application, bool, error) {
	trackedID := currentApplicationID(resource)
	if trackedID == "" {
		return nil, false, nil
	}

	current, err := getApplication(ctx, c.client, trackedID)
	if err != nil {
		if isApplicationReadNotFoundOCI(err) {
			clearApplicationTrackedIdentity(resource)
			return nil, true, nil
		}
		return nil, false, normalizeApplicationOCIError(err)
	}

	if current.LifecycleState == dataflowsdk.ApplicationLifecycleStateDeleted {
		clearApplicationTrackedIdentity(resource)
		return nil, true, nil
	}

	return &current, false, nil
}

func (c *applicationGeneratedRuntimeClient) reconcileWithDelegate(
	ctx context.Context,
	resource *dataflowv1beta1.Application,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	prepared, err := prepareApplicationForDelegate(resource)
	if err != nil {
		return applicationFail(resource, c.log, err)
	}

	response, err := c.delegate.CreateOrUpdate(ctx, prepared, req)
	resource.Status = prepared.Status
	return response, err
}

func (c *applicationGeneratedRuntimeClient) retryCreate(
	ctx context.Context,
	resource *dataflowv1beta1.Application,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	clearApplicationTrackedIdentity(resource)
	return c.reconcileWithDelegate(ctx, resource, req)
}

func prepareApplicationForDelegate(
	resource *dataflowv1beta1.Application,
) (*dataflowv1beta1.Application, error) {
	if resource == nil {
		return nil, fmt.Errorf("Application resource is nil")
	}

	prepared := resource.DeepCopy()
	if prepared == nil {
		return nil, fmt.Errorf("Application resource is nil")
	}
	return prepared, nil
}

func validateApplicationUpdateSpec(spec dataflowv1beta1.ApplicationSpec) error {
	if strings.TrimSpace(spec.Execute) != "" {
		return nil
	}

	_, err := applicationLanguageFromSpec(spec.Language)
	return err
}

func (c *applicationGeneratedRuntimeClient) Delete(
	ctx context.Context,
	resource *dataflowv1beta1.Application,
) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := currentApplicationID(resource)
	if trackedID == "" {
		markApplicationDeleted(resource, c.log, "OCI resource identifier is not recorded")
		return true, nil
	}

	deleteRequest := dataflowsdk.DeleteApplicationRequest{
		ApplicationId: common.String(trackedID),
	}
	if _, err := c.client.DeleteApplication(ctx, deleteRequest); err != nil {
		if isApplicationDeleteNotFoundOCI(err) {
			markApplicationDeleted(resource, c.log, "OCI resource no longer exists")
			return true, nil
		}
		return false, normalizeApplicationOCIError(err)
	}

	current, err := getApplication(ctx, c.client, trackedID)
	if err != nil {
		if isApplicationDeleteNotFoundOCI(err) {
			markApplicationDeleted(resource, c.log, "OCI resource deleted")
			return true, nil
		}
		return false, normalizeApplicationOCIError(err)
	}

	if err := projectApplicationStatus(resource, current); err != nil {
		return false, err
	}
	markApplicationTerminating(resource, c.log, current)
	return false, nil
}

func getApplication(ctx context.Context, client applicationOCIClient, ocid string) (dataflowsdk.Application, error) {
	response, err := client.GetApplication(ctx, dataflowsdk.GetApplicationRequest{
		ApplicationId: common.String(ocid),
	})
	if err != nil {
		return dataflowsdk.Application{}, err
	}
	return response.Application, nil
}

func applicationFail(
	resource *dataflowv1beta1.Application,
	log loggerutil.OSOKLogger,
	err error,
) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func markApplicationDeleted(
	resource *dataflowv1beta1.Application,
	log loggerutil.OSOKLogger,
	message string,
) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, log)
}

func clearApplicationTrackedIdentity(resource *dataflowv1beta1.Application) {
	resource.Status = dataflowv1beta1.ApplicationStatus{}
}

func markApplicationTerminating(
	resource *dataflowv1beta1.Application,
	log loggerutil.OSOKLogger,
	current dataflowsdk.Application,
) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = applicationLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, log)
}

func projectApplicationStatus(resource *dataflowv1beta1.Application, current dataflowsdk.Application) error {
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

func buildUpdateApplicationDetails(
	spec dataflowv1beta1.ApplicationSpec,
	current dataflowsdk.Application,
) (dataflowsdk.UpdateApplicationDetails, bool, error) {
	updateDetails := dataflowsdk.UpdateApplicationDetails{}
	updateNeeded := false

	if spec.Execute != "" && !stringPtrEqual(current.Execute, spec.Execute) {
		updateDetails.Execute = common.String(spec.Execute)
		updateNeeded = true
	}

	if spec.Execute == "" {
		language, err := applicationLanguageFromSpec(spec.Language)
		if err != nil {
			return dataflowsdk.UpdateApplicationDetails{}, false, err
		}
		if current.Language != language {
			updateDetails.Language = language
			updateNeeded = true
		}

		if spec.ClassName != "" && !stringPtrEqual(current.ClassName, spec.ClassName) {
			updateDetails.ClassName = common.String(spec.ClassName)
			updateNeeded = true
		}
		if spec.FileUri != "" && !stringPtrEqual(current.FileUri, spec.FileUri) {
			updateDetails.FileUri = common.String(spec.FileUri)
			updateNeeded = true
		}
		if len(spec.Arguments) > 0 && !reflect.DeepEqual(current.Arguments, spec.Arguments) {
			updateDetails.Arguments = cloneStringSlice(spec.Arguments)
			updateNeeded = true
		}
		if len(spec.Configuration) > 0 && !reflect.DeepEqual(current.Configuration, spec.Configuration) {
			updateDetails.Configuration = cloneStringMap(spec.Configuration)
			updateNeeded = true
		}
		if len(spec.Parameters) > 0 {
			parameters := convertApplicationParameters(spec.Parameters)
			if !reflect.DeepEqual(current.Parameters, parameters) {
				updateDetails.Parameters = parameters
				updateNeeded = true
			}
		}
	}

	if spec.SparkVersion != "" && !stringPtrEqual(current.SparkVersion, spec.SparkVersion) {
		updateDetails.SparkVersion = common.String(spec.SparkVersion)
		updateNeeded = true
	}
	if spec.DisplayName != "" && !stringPtrEqual(current.DisplayName, spec.DisplayName) {
		updateDetails.DisplayName = common.String(spec.DisplayName)
		updateNeeded = true
	}
	if spec.DriverShape != "" && !stringPtrEqual(current.DriverShape, spec.DriverShape) {
		updateDetails.DriverShape = common.String(spec.DriverShape)
		updateNeeded = true
	}
	if spec.ExecutorShape != "" && !stringPtrEqual(current.ExecutorShape, spec.ExecutorShape) {
		updateDetails.ExecutorShape = common.String(spec.ExecutorShape)
		updateNeeded = true
	}
	if spec.NumExecutors != 0 && !intPtrEqual(current.NumExecutors, spec.NumExecutors) {
		updateDetails.NumExecutors = common.Int(spec.NumExecutors)
		updateNeeded = true
	}
	if spec.ArchiveUri != "" && !stringPtrEqual(current.ArchiveUri, spec.ArchiveUri) {
		updateDetails.ArchiveUri = common.String(spec.ArchiveUri)
		updateNeeded = true
	}
	if logConfig := convertApplicationLogConfig(spec.ApplicationLogConfig); logConfig != nil && !reflect.DeepEqual(current.ApplicationLogConfig, logConfig) {
		updateDetails.ApplicationLogConfig = logConfig
		updateNeeded = true
	}
	if spec.Description != "" && !stringPtrEqual(current.Description, spec.Description) {
		updateDetails.Description = common.String(spec.Description)
		updateNeeded = true
	}
	if driverShapeConfig := convertShapeConfig(spec.DriverShapeConfig); driverShapeConfig != nil && !reflect.DeepEqual(current.DriverShapeConfig, driverShapeConfig) {
		updateDetails.DriverShapeConfig = driverShapeConfig
		updateNeeded = true
	}
	if executorShapeConfig := convertExecutorShapeConfig(spec.ExecutorShapeConfig); executorShapeConfig != nil && !reflect.DeepEqual(current.ExecutorShapeConfig, executorShapeConfig) {
		updateDetails.ExecutorShapeConfig = executorShapeConfig
		updateNeeded = true
	}
	if len(spec.FreeformTags) > 0 && !reflect.DeepEqual(current.FreeformTags, spec.FreeformTags) {
		updateDetails.FreeformTags = cloneStringMap(spec.FreeformTags)
		updateNeeded = true
	}
	if len(spec.DefinedTags) > 0 {
		definedTags := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, definedTags) {
			updateDetails.DefinedTags = definedTags
			updateNeeded = true
		}
	}
	if spec.LogsBucketUri != "" && !stringPtrEqual(current.LogsBucketUri, spec.LogsBucketUri) {
		updateDetails.LogsBucketUri = common.String(spec.LogsBucketUri)
		updateNeeded = true
	}
	if spec.MetastoreId != "" && !stringPtrEqual(current.MetastoreId, spec.MetastoreId) {
		updateDetails.MetastoreId = common.String(spec.MetastoreId)
		updateNeeded = true
	}
	if spec.PoolId != "" && !stringPtrEqual(current.PoolId, spec.PoolId) {
		updateDetails.PoolId = common.String(spec.PoolId)
		updateNeeded = true
	}
	if spec.PrivateEndpointId != "" && !stringPtrEqual(current.PrivateEndpointId, spec.PrivateEndpointId) {
		updateDetails.PrivateEndpointId = common.String(spec.PrivateEndpointId)
		updateNeeded = true
	}
	if spec.WarehouseBucketUri != "" && !stringPtrEqual(current.WarehouseBucketUri, spec.WarehouseBucketUri) {
		updateDetails.WarehouseBucketUri = common.String(spec.WarehouseBucketUri)
		updateNeeded = true
	}
	if spec.MaxDurationInMinutes != 0 && !int64PtrEqual(current.MaxDurationInMinutes, spec.MaxDurationInMinutes) {
		updateDetails.MaxDurationInMinutes = common.Int64(spec.MaxDurationInMinutes)
		updateNeeded = true
	}
	if spec.IdleTimeoutInMinutes != 0 && !int64PtrEqual(current.IdleTimeoutInMinutes, spec.IdleTimeoutInMinutes) {
		updateDetails.IdleTimeoutInMinutes = common.Int64(spec.IdleTimeoutInMinutes)
		updateNeeded = true
	}

	return updateDetails, updateNeeded, nil
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

func currentApplicationID(resource *dataflowv1beta1.Application) string {
	if resource == nil {
		return ""
	}

	if trackedID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); trackedID != "" {
		return trackedID
	}
	return strings.TrimSpace(resource.Status.Id)
}

func stringCreateOnlyMatches(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
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
