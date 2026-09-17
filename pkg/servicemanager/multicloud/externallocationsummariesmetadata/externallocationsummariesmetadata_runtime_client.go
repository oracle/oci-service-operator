/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package externallocationsummariesmetadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	multicloudsdk "github.com/oracle/oci-go-sdk/v65/multicloud"
	multicloudv1beta1 "github.com/oracle/oci-service-operator/api/multicloud/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	externalLocationSummariesMetadataKind = "ExternalLocationSummariesMetadata"

	externalLocationSummariesMetadataSubscriptionServiceNameAnnotation = "multicloud.oracle.com/subscription-service-name"
	externalLocationSummariesMetadataCompartmentIDAnnotation           = "multicloud.oracle.com/compartment-id"
	externalLocationSummariesMetadataSubscriptionIDAnnotation          = "multicloud.oracle.com/subscription-id"
	externalLocationSummariesMetadataEntityTypeAnnotation              = "multicloud.oracle.com/entity-type"
	externalLocationSummariesMetadataLimitAnnotation                   = "multicloud.oracle.com/limit"
	externalLocationSummariesMetadataSortOrderAnnotation               = "multicloud.oracle.com/sort-order"
	externalLocationSummariesMetadataSortByAnnotation                  = "multicloud.oracle.com/sort-by"
)

type externalLocationSummariesMetadataOCIClient interface {
	ListExternalLocationSummariesMetadata(
		context.Context,
		multicloudsdk.ListExternalLocationSummariesMetadataRequest,
	) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error)
}

type externalLocationSummariesMetadataListCall func(
	context.Context,
	multicloudsdk.ListExternalLocationSummariesMetadataRequest,
) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error)

type externalLocationSummariesMetadataRuntimeClient struct {
	log     loggerutil.OSOKLogger
	list    externalLocationSummariesMetadataListCall
	initErr error
}

func init() {
	registerExternalLocationSummariesMetadataRuntimeHooksMutator(
		func(manager *ExternalLocationSummariesMetadataServiceManager, hooks *ExternalLocationSummariesMetadataRuntimeHooks) {
			client, initErr := newExternalLocationSummariesMetadataOCIClient(manager)
			applyExternalLocationSummariesMetadataRuntimeHooks(manager, hooks, client, initErr)
		},
	)
}

func newExternalLocationSummariesMetadataOCIClient(
	manager *ExternalLocationSummariesMetadataServiceManager,
) (externalLocationSummariesMetadataOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", externalLocationSummariesMetadataKind)
	}
	return multicloudsdk.NewMetadataClientWithConfigurationProvider(manager.Provider)
}

func applyExternalLocationSummariesMetadataRuntimeHooks(
	manager *ExternalLocationSummariesMetadataServiceManager,
	hooks *ExternalLocationSummariesMetadataRuntimeHooks,
	client externalLocationSummariesMetadataOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	listCall := hooks.List.Call
	if client != nil {
		listCall = client.ListExternalLocationSummariesMetadata
	}
	hooks.List.Call = listExternalLocationSummariesMetadataAllPages(listCall)

	log := loggerutil.OSOKLogger{}
	if manager != nil {
		log = manager.Log
	}
	hooks.WrapGeneratedClient = append(
		hooks.WrapGeneratedClient,
		func(ExternalLocationSummariesMetadataServiceClient) ExternalLocationSummariesMetadataServiceClient {
			return newExternalLocationSummariesMetadataServiceClientWithListCall(log, listCall, initErr)
		},
	)
}

func newExternalLocationSummariesMetadataServiceClientWithListCall(
	log loggerutil.OSOKLogger,
	listCall externalLocationSummariesMetadataListCall,
	initErr error,
) ExternalLocationSummariesMetadataServiceClient {
	return &externalLocationSummariesMetadataRuntimeClient{
		log:     log,
		list:    listExternalLocationSummariesMetadataAllPages(listCall),
		initErr: initErr,
	}
}

func (c *externalLocationSummariesMetadataRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s resource is nil", externalLocationSummariesMetadataKind)
	}
	if c.initErr != nil {
		return c.fail(resource, fmt.Errorf("initialize %s OCI client: %w", externalLocationSummariesMetadataKind, c.initErr))
	}
	if c.list == nil {
		return c.fail(resource, fmt.Errorf("%s list operation is not configured", externalLocationSummariesMetadataKind))
	}

	request, err := externalLocationSummariesMetadataListRequest(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	response, err := c.list(ctx, request)
	if err != nil {
		return c.fail(resource, err)
	}
	if err := generatedruntime.ProjectResponseBodyWithAliases(resource, response, nil); err != nil {
		return c.fail(resource, err)
	}
	return c.markActive(resource, response), nil
}

func (c *externalLocationSummariesMetadataRuntimeClient) Delete(
	_ context.Context,
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", externalLocationSummariesMetadataKind)
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = "ExternalLocationSummariesMetadata is read-only metadata; no OCI delete is required"
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		status.Message,
		c.log,
	)
	return true, nil
}

func (c *externalLocationSummariesMetadataRuntimeClient) markActive(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
	response multicloudsdk.ListExternalLocationSummariesMetadataResponse,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	servicemanager.RecordResponseOpcRequestID(status, response)

	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = fmt.Sprintf("observed %d ExternalLocationSummariesMetadata item(s)", len(response.Items))
	status.Reason = string(shared.Active)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", status.Message, c.log)
	return servicemanager.OSOKResponse{IsSuccessful: true}
}

func (c *externalLocationSummariesMetadataRuntimeClient) fail(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
	err error,
) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)

	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func externalLocationSummariesMetadataListRequest(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
) (multicloudsdk.ListExternalLocationSummariesMetadataRequest, error) {
	if resource == nil {
		return multicloudsdk.ListExternalLocationSummariesMetadataRequest{}, fmt.Errorf("%s resource is nil", externalLocationSummariesMetadataKind)
	}

	subscriptionServiceName, err := externalLocationSummariesMetadataSubscriptionServiceName(resource)
	if err != nil {
		return multicloudsdk.ListExternalLocationSummariesMetadataRequest{}, err
	}
	compartmentID := externalLocationSummariesMetadataRequiredAnnotation(
		resource,
		externalLocationSummariesMetadataCompartmentIDAnnotation,
	)
	if compartmentID == "" {
		return multicloudsdk.ListExternalLocationSummariesMetadataRequest{}, fmt.Errorf(
			"%s requires metadata annotation %q because the OCI list request has a mandatory compartmentId query field and the CRD has no spec field for it",
			externalLocationSummariesMetadataKind,
			externalLocationSummariesMetadataCompartmentIDAnnotation,
		)
	}

	request := multicloudsdk.ListExternalLocationSummariesMetadataRequest{
		SubscriptionServiceName: subscriptionServiceName,
		CompartmentId:           common.String(compartmentID),
	}
	if subscriptionID := externalLocationSummariesMetadataAnnotation(resource, externalLocationSummariesMetadataSubscriptionIDAnnotation); subscriptionID != "" {
		request.SubscriptionId = common.String(subscriptionID)
	}
	if request.EntityType, err = externalLocationSummariesMetadataEntityType(resource); err != nil {
		return multicloudsdk.ListExternalLocationSummariesMetadataRequest{}, err
	}
	if request.Limit, err = externalLocationSummariesMetadataLimit(resource); err != nil {
		return multicloudsdk.ListExternalLocationSummariesMetadataRequest{}, err
	}
	if request.SortOrder, err = externalLocationSummariesMetadataSortOrder(resource); err != nil {
		return multicloudsdk.ListExternalLocationSummariesMetadataRequest{}, err
	}
	if request.SortBy, err = externalLocationSummariesMetadataSortBy(resource); err != nil {
		return multicloudsdk.ListExternalLocationSummariesMetadataRequest{}, err
	}
	return request, nil
}

func externalLocationSummariesMetadataSubscriptionServiceName(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
) (multicloudsdk.ListExternalLocationSummariesMetadataSubscriptionServiceNameEnum, error) {
	value := externalLocationSummariesMetadataRequiredAnnotation(
		resource,
		externalLocationSummariesMetadataSubscriptionServiceNameAnnotation,
	)
	if value == "" {
		return "", fmt.Errorf(
			"%s requires metadata annotation %q because the OCI list request has a mandatory subscriptionServiceName query field and the CRD has no spec field for it",
			externalLocationSummariesMetadataKind,
			externalLocationSummariesMetadataSubscriptionServiceNameAnnotation,
		)
	}
	enumValue, ok := multicloudsdk.GetMappingListExternalLocationSummariesMetadataSubscriptionServiceNameEnum(value)
	if !ok {
		return "", fmt.Errorf(
			"%s annotation %q has unsupported value %q; supported values: %s",
			externalLocationSummariesMetadataKind,
			externalLocationSummariesMetadataSubscriptionServiceNameAnnotation,
			value,
			strings.Join(multicloudsdk.GetListExternalLocationSummariesMetadataSubscriptionServiceNameEnumStringValues(), ","),
		)
	}
	return enumValue, nil
}

func externalLocationSummariesMetadataEntityType(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
) (multicloudsdk.ListExternalLocationSummariesMetadataEntityTypeEnum, error) {
	value := externalLocationSummariesMetadataAnnotation(resource, externalLocationSummariesMetadataEntityTypeAnnotation)
	if value == "" {
		return "", nil
	}
	enumValue, ok := multicloudsdk.GetMappingListExternalLocationSummariesMetadataEntityTypeEnum(value)
	if !ok {
		return "", fmt.Errorf(
			"%s annotation %q has unsupported value %q; supported values: %s",
			externalLocationSummariesMetadataKind,
			externalLocationSummariesMetadataEntityTypeAnnotation,
			value,
			strings.Join(multicloudsdk.GetListExternalLocationSummariesMetadataEntityTypeEnumStringValues(), ","),
		)
	}
	return enumValue, nil
}

func externalLocationSummariesMetadataSortOrder(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
) (multicloudsdk.ListExternalLocationSummariesMetadataSortOrderEnum, error) {
	value := externalLocationSummariesMetadataAnnotation(resource, externalLocationSummariesMetadataSortOrderAnnotation)
	if value == "" {
		return "", nil
	}
	enumValue, ok := multicloudsdk.GetMappingListExternalLocationSummariesMetadataSortOrderEnum(value)
	if !ok {
		return "", fmt.Errorf(
			"%s annotation %q has unsupported value %q; supported values: %s",
			externalLocationSummariesMetadataKind,
			externalLocationSummariesMetadataSortOrderAnnotation,
			value,
			strings.Join(multicloudsdk.GetListExternalLocationSummariesMetadataSortOrderEnumStringValues(), ","),
		)
	}
	return enumValue, nil
}

func externalLocationSummariesMetadataSortBy(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
) (multicloudsdk.ListExternalLocationSummariesMetadataSortByEnum, error) {
	value := externalLocationSummariesMetadataAnnotation(resource, externalLocationSummariesMetadataSortByAnnotation)
	if value == "" {
		return "", nil
	}
	enumValue, ok := multicloudsdk.GetMappingListExternalLocationSummariesMetadataSortByEnum(value)
	if !ok {
		return "", fmt.Errorf(
			"%s annotation %q has unsupported value %q; supported values: %s",
			externalLocationSummariesMetadataKind,
			externalLocationSummariesMetadataSortByAnnotation,
			value,
			strings.Join(multicloudsdk.GetListExternalLocationSummariesMetadataSortByEnumStringValues(), ","),
		)
	}
	return enumValue, nil
}

func externalLocationSummariesMetadataLimit(resource *multicloudv1beta1.ExternalLocationSummariesMetadata) (*int, error) {
	value := externalLocationSummariesMetadataAnnotation(resource, externalLocationSummariesMetadataLimitAnnotation)
	if value == "" {
		return nil, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return nil, fmt.Errorf(
			"%s annotation %q must be a positive integer",
			externalLocationSummariesMetadataKind,
			externalLocationSummariesMetadataLimitAnnotation,
		)
	}
	return common.Int(limit), nil
}

func externalLocationSummariesMetadataRequiredAnnotation(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
	key string,
) string {
	return externalLocationSummariesMetadataAnnotation(resource, key)
}

func externalLocationSummariesMetadataAnnotation(
	resource *multicloudv1beta1.ExternalLocationSummariesMetadata,
	key string,
) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.GetAnnotations()[key])
}

func listExternalLocationSummariesMetadataAllPages(
	call externalLocationSummariesMetadataListCall,
) externalLocationSummariesMetadataListCall {
	if call == nil {
		return nil
	}
	return func(
		ctx context.Context,
		request multicloudsdk.ListExternalLocationSummariesMetadataRequest,
	) (multicloudsdk.ListExternalLocationSummariesMetadataResponse, error) {
		seenPages := map[string]struct{}{}
		var combined multicloudsdk.ListExternalLocationSummariesMetadataResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return multicloudsdk.ListExternalLocationSummariesMetadataResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			if response.OpcRequestId != nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.Items = append(combined.Items, response.Items...)

			nextPage := strings.TrimSpace(stringPtrValue(response.OpcNextPage))
			if nextPage == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return multicloudsdk.ListExternalLocationSummariesMetadataResponse{}, fmt.Errorf(
					"%s list pagination repeated page token %q",
					externalLocationSummariesMetadataKind,
					nextPage,
				)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
