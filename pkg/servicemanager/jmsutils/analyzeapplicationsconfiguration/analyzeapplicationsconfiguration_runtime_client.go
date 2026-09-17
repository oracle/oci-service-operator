/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package analyzeapplicationsconfiguration

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	jmsutilssdk "github.com/oracle/oci-go-sdk/v65/jmsutils"
	jmsutilsv1beta1 "github.com/oracle/oci-service-operator/api/jmsutils/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const analyzeApplicationsConfigurationDeleteMessage = "OCI delete is not supported for AnalyzeApplicationsConfiguration; removing Kubernetes finalizer without changing OCI configuration"

type analyzeApplicationsConfigurationOCIClient interface {
	GetAnalyzeApplicationsConfiguration(context.Context, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.GetAnalyzeApplicationsConfigurationResponse, error)
	UpdateAnalyzeApplicationsConfiguration(context.Context, jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationResponse, error)
}

type analyzeApplicationsConfigurationRuntimeClient struct {
	delegate      AnalyzeApplicationsConfigurationServiceClient
	client        analyzeApplicationsConfigurationOCIClient
	compartmentID *string
	initErr       error
	log           loggerutil.OSOKLogger
}

var _ AnalyzeApplicationsConfigurationServiceClient = (*analyzeApplicationsConfigurationRuntimeClient)(nil)

func init() {
	registerAnalyzeApplicationsConfigurationRuntimeHooksMutator(func(manager *AnalyzeApplicationsConfigurationServiceManager, hooks *AnalyzeApplicationsConfigurationRuntimeHooks) {
		client, initErr := newAnalyzeApplicationsConfigurationSDKClient(manager)
		applyAnalyzeApplicationsConfigurationRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newAnalyzeApplicationsConfigurationSDKClient(manager *AnalyzeApplicationsConfigurationServiceManager) (analyzeApplicationsConfigurationOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("AnalyzeApplicationsConfiguration service manager is nil")
	}
	client, err := jmsutilssdk.NewJmsUtilsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAnalyzeApplicationsConfigurationRuntimeHooks(
	manager *AnalyzeApplicationsConfigurationServiceManager,
	hooks *AnalyzeApplicationsConfigurationRuntimeHooks,
	client analyzeApplicationsConfigurationOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	log := loggerutil.OSOKLogger{}
	var compartmentID *string
	if manager != nil {
		log = manager.Log
		if initErr == nil && manager.Provider != nil {
			tenancyID, err := manager.Provider.TenancyOCID()
			if err != nil {
				initErr = fmt.Errorf("resolve AnalyzeApplicationsConfiguration tenancy: %w", err)
			} else if strings.TrimSpace(tenancyID) == "" {
				initErr = fmt.Errorf("resolve AnalyzeApplicationsConfiguration tenancy: empty tenancy OCID")
			} else {
				compartmentID = common.String(tenancyID)
			}
		}
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AnalyzeApplicationsConfigurationServiceClient) AnalyzeApplicationsConfigurationServiceClient {
		return &analyzeApplicationsConfigurationRuntimeClient{
			delegate:      delegate,
			client:        client,
			compartmentID: compartmentID,
			initErr:       initErr,
			log:           log,
		}
	})
}

func (c *analyzeApplicationsConfigurationRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AnalyzeApplicationsConfiguration resource is nil")
	}
	if c.initErr != nil {
		return c.fail(resource, fmt.Errorf("initialize AnalyzeApplicationsConfiguration OCI client: %w", c.initErr))
	}
	if c.client == nil {
		return c.fail(resource, fmt.Errorf("AnalyzeApplicationsConfiguration OCI client is not configured"))
	}

	getResponse, err := c.client.GetAnalyzeApplicationsConfiguration(ctx, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest{
		CompartmentId: c.compartmentID,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	projectAnalyzeApplicationsConfigurationStatus(resource, getResponse.AnalyzeApplicationsConfiguration)

	updateDetails, shouldUpdate := analyzeApplicationsConfigurationUpdateDetails(resource, getResponse.AnalyzeApplicationsConfiguration)
	if !shouldUpdate {
		return c.markActive(resource, "AnalyzeApplicationsConfiguration configuration is current"), nil
	}

	updateRequest := jmsutilssdk.UpdateAnalyzeApplicationsConfigurationRequest{
		UpdateAnalyzeApplicationsConfigurationDetails: updateDetails,
		CompartmentId: c.compartmentID,
	}
	if getResponse.Etag != nil && strings.TrimSpace(*getResponse.Etag) != "" {
		updateRequest.IfMatch = getResponse.Etag
	}

	updateResponse, err := c.client.UpdateAnalyzeApplicationsConfiguration(ctx, updateRequest)
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, updateResponse)
	refreshResponse, err := c.client.GetAnalyzeApplicationsConfiguration(ctx, jmsutilssdk.GetAnalyzeApplicationsConfigurationRequest{
		CompartmentId: c.compartmentID,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	projectAnalyzeApplicationsConfigurationStatus(resource, refreshResponse.AnalyzeApplicationsConfiguration)
	return c.markActive(resource, "AnalyzeApplicationsConfiguration configuration updated"), nil
}

func (c *analyzeApplicationsConfigurationRuntimeClient) Delete(ctx context.Context, resource *jmsutilsv1beta1.AnalyzeApplicationsConfiguration) (bool, error) {
	if resource == nil {
		if c.delegate != nil {
			return c.delegate.Delete(ctx, resource)
		}
		return false, fmt.Errorf("AnalyzeApplicationsConfiguration resource is nil")
	}

	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = analyzeApplicationsConfigurationDeleteMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", analyzeApplicationsConfigurationDeleteMessage, c.log)
	return true, nil
}

func (c *analyzeApplicationsConfigurationRuntimeClient) markActive(
	resource *jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
	message string,
) servicemanager.OSOKResponse {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Active)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", message, c.log)
	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func (c *analyzeApplicationsConfigurationRuntimeClient) fail(
	resource *jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
	err error,
) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		status.Async.Current = &current
	} else {
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func analyzeApplicationsConfigurationUpdateDetails(
	resource *jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
	current jmsutilssdk.AnalyzeApplicationsConfiguration,
) (jmsutilssdk.UpdateAnalyzeApplicationsConfigurationDetails, bool) {
	details := jmsutilssdk.UpdateAnalyzeApplicationsConfigurationDetails{}
	shouldUpdate := false

	if desired := strings.TrimSpace(resource.Spec.NamespaceName); desired != "" && !stringPtrEqual(current.NamespaceName, desired) {
		details.NamespaceName = common.String(desired)
		shouldUpdate = true
	}
	if desired := strings.TrimSpace(resource.Spec.BucketName); desired != "" && !stringPtrEqual(current.BucketName, desired) {
		details.BucketName = common.String(desired)
		shouldUpdate = true
	}

	return details, shouldUpdate
}

func projectAnalyzeApplicationsConfigurationStatus(
	resource *jmsutilsv1beta1.AnalyzeApplicationsConfiguration,
	current jmsutilssdk.AnalyzeApplicationsConfiguration,
) {
	resource.Status.NamespaceName = stringValue(current.NamespaceName)
	resource.Status.BucketName = stringValue(current.BucketName)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(stringValue(current)) == strings.TrimSpace(desired)
}
