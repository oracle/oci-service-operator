/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package externallocationdetailsmetadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
	externalLocationDetailsMetadataSubscriptionIDAnnotation          = "multicloud.oracle.com/subscription-id"
	externalLocationDetailsMetadataSubscriptionServiceNameAnnotation = "multicloud.oracle.com/subscription-service-name"
	externalLocationDetailsMetadataEntityTypeAnnotation              = "multicloud.oracle.com/entity-type"
	externalLocationDetailsMetadataCompartmentIDAnnotation           = "multicloud.oracle.com/compartment-id"
	externalLocationDetailsMetadataLinkedCompartmentIDAnnotation     = "multicloud.oracle.com/linked-compartment-id"
	externalLocationDetailsMetadataExternalLocationAnnotation        = "multicloud.oracle.com/external-location"
	externalLocationDetailsMetadataLogicalZoneAnnotation             = "multicloud.oracle.com/logical-zone"
	externalLocationDetailsMetadataClusterPlacementGroupIDAnnotation = "multicloud.oracle.com/cluster-placement-group-id"
	externalLocationDetailsMetadataSortOrderAnnotation               = "multicloud.oracle.com/sort-order"
	externalLocationDetailsMetadataSortByAnnotation                  = "multicloud.oracle.com/sort-by"
	externalLocationDetailsMetadataOCIRegionAnnotation               = "multicloud.oracle.com/oci-region"
	externalLocationDetailsMetadataOCIPhysicalADAnnotation           = "multicloud.oracle.com/oci-physical-ad"
	externalLocationDetailsMetadataOCILogicalADAnnotation            = "multicloud.oracle.com/oci-logical-ad"
	externalLocationDetailsMetadataCspRegionAnnotation               = "multicloud.oracle.com/csp-region"
	externalLocationDetailsMetadataCspPhysicalAZAnnotation           = "multicloud.oracle.com/csp-physical-az"
	externalLocationDetailsMetadataCspLogicalAZAnnotation            = "multicloud.oracle.com/csp-logical-az"

	externalLocationDetailsMetadataObservedMessage = "Observed ExternalLocationDetailsMetadata from OCI metadata list"
	externalLocationDetailsMetadataDeletedMessage  = "ExternalLocationDetailsMetadata has no managed OCI resource; delete is confirmed without an OCI request"
)

func init() {
	registerExternalLocationDetailsMetadataRuntimeHooksMutator(func(manager *ExternalLocationDetailsMetadataServiceManager, hooks *ExternalLocationDetailsMetadataRuntimeHooks) {
		applyExternalLocationDetailsMetadataRuntimeHooks(manager, hooks)
	})
}

type externalLocationDetailsMetadataListFunc func(context.Context, multicloudsdk.ListExternalLocationDetailsMetadataRequest) (multicloudsdk.ListExternalLocationDetailsMetadataResponse, error)

type externalLocationDetailsMetadataRuntimeClient struct {
	log     loggerutil.OSOKLogger
	list    externalLocationDetailsMetadataListFunc
	initErr error
}

var _ ExternalLocationDetailsMetadataServiceClient = (*externalLocationDetailsMetadataRuntimeClient)(nil)

func applyExternalLocationDetailsMetadataRuntimeHooks(manager *ExternalLocationDetailsMetadataServiceManager, hooks *ExternalLocationDetailsMetadataRuntimeHooks) {
	if hooks == nil {
		return
	}

	var log loggerutil.OSOKLogger
	if manager != nil {
		log = manager.Log
	}
	list := hooks.List.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ExternalLocationDetailsMetadataServiceClient) ExternalLocationDetailsMetadataServiceClient {
		return &externalLocationDetailsMetadataRuntimeClient{
			log:     log,
			list:    list,
			initErr: externalLocationDetailsMetadataGeneratedDelegateInitError(delegate),
		}
	})
}

func (c *externalLocationDetailsMetadataRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *multicloudv1beta1.ExternalLocationDetailsMetadata,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.generatedInitError(); err != nil {
		if resource != nil {
			markExternalLocationDetailsMetadataFailed(resource, err, "", c.logger())
		}
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, errors.New("ExternalLocationDetailsMetadata resource is nil")
	}

	selector, err := externalLocationDetailsMetadataSelectorFromAnnotations(resource)
	if err != nil {
		markExternalLocationDetailsMetadataFailed(resource, err, "", c.logger())
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	items, opcRequestID, err := c.listExternalLocationDetailsMetadata(ctx, selector)
	if err != nil {
		markExternalLocationDetailsMetadataFailed(resource, err, opcRequestID, c.logger())
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	selected, err := selector.selectOne(items)
	if err != nil {
		markExternalLocationDetailsMetadataFailed(resource, err, opcRequestID, c.logger())
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	identity := selector.identity(selected)
	if tracked := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); tracked != "" && tracked != identity {
		err := fmt.Errorf("ExternalLocationDetailsMetadata annotations select identity %q but status.status.ocid records %q; replacement is required for selector drift", identity, tracked)
		markExternalLocationDetailsMetadataFailed(resource, err, opcRequestID, c.logger())
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if err := generatedruntime.ProjectResponseBodyWithAliases(resource, selected, nil); err != nil {
		markExternalLocationDetailsMetadataFailed(resource, err, opcRequestID, c.logger())
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	markExternalLocationDetailsMetadataObserved(resource, identity, opcRequestID, c.logger())
	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func (c *externalLocationDetailsMetadataRuntimeClient) Delete(
	_ context.Context,
	resource *multicloudv1beta1.ExternalLocationDetailsMetadata,
) (bool, error) {
	if err := c.generatedInitError(); err != nil {
		return false, err
	}
	if resource == nil {
		return false, errors.New("ExternalLocationDetailsMetadata resource is nil")
	}

	markExternalLocationDetailsMetadataDeleted(resource, c.logger())
	return true, nil
}

func (c *externalLocationDetailsMetadataRuntimeClient) listExternalLocationDetailsMetadata(
	ctx context.Context,
	selector externalLocationDetailsMetadataSelector,
) ([]multicloudsdk.ExternalLocationsMetadatumSummary, string, error) {
	if c == nil || c.list == nil {
		return nil, "", errors.New("ExternalLocationDetailsMetadata list client is not configured")
	}

	request := selector.listRequest()
	var items []multicloudsdk.ExternalLocationsMetadatumSummary
	var opcRequestID string
	seenPages := map[string]struct{}{}
	for {
		response, err := c.list(ctx, request)
		if err != nil {
			return nil, externalLocationDetailsMetadataListErrorRequestID(response, err, opcRequestID), err
		}
		if requestID := stringValue(response.OpcRequestId); requestID != "" {
			opcRequestID = requestID
		}
		items = append(items, response.Items...)
		nextPage := stringValue(response.OpcNextPage)
		if nextPage == "" {
			return items, opcRequestID, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return nil, opcRequestID, fmt.Errorf("ExternalLocationDetailsMetadata list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = common.String(nextPage)
	}
}

func externalLocationDetailsMetadataListErrorRequestID(
	response multicloudsdk.ListExternalLocationDetailsMetadataResponse,
	err error,
	fallback string,
) string {
	if requestID := stringValue(response.OpcRequestId); requestID != "" {
		return requestID
	}
	if requestID := servicemanager.ErrorOpcRequestID(err); requestID != "" {
		return requestID
	}
	return fallback
}

func (c *externalLocationDetailsMetadataRuntimeClient) logger() loggerutil.OSOKLogger {
	if c == nil {
		return loggerutil.OSOKLogger{}
	}
	return c.log
}

func (c *externalLocationDetailsMetadataRuntimeClient) generatedInitError() error {
	if c == nil {
		return nil
	}
	return c.initErr
}

func externalLocationDetailsMetadataGeneratedDelegateInitError(delegate ExternalLocationDetailsMetadataServiceClient) error {
	if delegate == nil {
		return nil
	}

	var resource *multicloudv1beta1.ExternalLocationDetailsMetadata
	_, err := delegate.Delete(context.Background(), resource)
	if err == nil || isExternalLocationDetailsMetadataNilResourceProbeError(err) {
		return nil
	}
	return err
}

func isExternalLocationDetailsMetadataNilResourceProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "resource is nil") || strings.Contains(message, "expected pointer resource")
}

type externalLocationDetailsMetadataSelector struct {
	subscriptionID          string
	subscriptionServiceName string
	entityType              string
	compartmentID           string
	linkedCompartmentID     string
	externalLocation        string
	logicalZone             string
	clusterPlacementGroupID string
	sortOrder               string
	sortBy                  string
	ociRegion               string
	ociPhysicalAD           string
	ociLogicalAD            string
	cspRegion               string
	cspPhysicalAZ           string
	cspLogicalAZ            string
}

func externalLocationDetailsMetadataSelectorFromAnnotations(resource *multicloudv1beta1.ExternalLocationDetailsMetadata) (externalLocationDetailsMetadataSelector, error) {
	annotations := resource.GetAnnotations()
	selector := externalLocationDetailsMetadataSelector{
		subscriptionID:          annotationValue(annotations, externalLocationDetailsMetadataSubscriptionIDAnnotation),
		subscriptionServiceName: annotationValue(annotations, externalLocationDetailsMetadataSubscriptionServiceNameAnnotation),
		entityType:              annotationValue(annotations, externalLocationDetailsMetadataEntityTypeAnnotation),
		compartmentID:           annotationValue(annotations, externalLocationDetailsMetadataCompartmentIDAnnotation),
		linkedCompartmentID:     annotationValue(annotations, externalLocationDetailsMetadataLinkedCompartmentIDAnnotation),
		externalLocation:        annotationValue(annotations, externalLocationDetailsMetadataExternalLocationAnnotation),
		logicalZone:             annotationValue(annotations, externalLocationDetailsMetadataLogicalZoneAnnotation),
		clusterPlacementGroupID: annotationValue(annotations, externalLocationDetailsMetadataClusterPlacementGroupIDAnnotation),
		sortOrder:               annotationValue(annotations, externalLocationDetailsMetadataSortOrderAnnotation),
		sortBy:                  annotationValue(annotations, externalLocationDetailsMetadataSortByAnnotation),
		ociRegion:               annotationValue(annotations, externalLocationDetailsMetadataOCIRegionAnnotation),
		ociPhysicalAD:           annotationValue(annotations, externalLocationDetailsMetadataOCIPhysicalADAnnotation),
		ociLogicalAD:            annotationValue(annotations, externalLocationDetailsMetadataOCILogicalADAnnotation),
		cspRegion:               annotationValue(annotations, externalLocationDetailsMetadataCspRegionAnnotation),
		cspPhysicalAZ:           annotationValue(annotations, externalLocationDetailsMetadataCspPhysicalAZAnnotation),
		cspLogicalAZ:            annotationValue(annotations, externalLocationDetailsMetadataCspLogicalAZAnnotation),
	}
	return selector, selector.validate()
}

func (s externalLocationDetailsMetadataSelector) validate() error {
	if s.subscriptionID == "" {
		return fmt.Errorf("ExternalLocationDetailsMetadata annotation %q is required because the generated spec is empty and ListExternalLocationDetailsMetadata requires subscriptionId", externalLocationDetailsMetadataSubscriptionIDAnnotation)
	}
	if s.subscriptionServiceName == "" {
		return fmt.Errorf("ExternalLocationDetailsMetadata annotation %q is required because the generated spec is empty and ListExternalLocationDetailsMetadata requires subscriptionServiceName", externalLocationDetailsMetadataSubscriptionServiceNameAnnotation)
	}
	if _, ok := multicloudsdk.GetMappingListExternalLocationDetailsMetadataSubscriptionServiceNameEnum(s.subscriptionServiceName); !ok {
		return fmt.Errorf("ExternalLocationDetailsMetadata annotation %q has unsupported subscriptionServiceName %q", externalLocationDetailsMetadataSubscriptionServiceNameAnnotation, s.subscriptionServiceName)
	}
	if s.entityType != "" {
		if _, ok := multicloudsdk.GetMappingListExternalLocationDetailsMetadataEntityTypeEnum(s.entityType); !ok {
			return fmt.Errorf("ExternalLocationDetailsMetadata annotation %q has unsupported entityType %q", externalLocationDetailsMetadataEntityTypeAnnotation, s.entityType)
		}
	}
	if s.sortOrder != "" {
		if _, ok := multicloudsdk.GetMappingListExternalLocationDetailsMetadataSortOrderEnum(s.sortOrder); !ok {
			return fmt.Errorf("ExternalLocationDetailsMetadata annotation %q has unsupported sortOrder %q", externalLocationDetailsMetadataSortOrderAnnotation, s.sortOrder)
		}
	}
	if s.sortBy != "" {
		if _, ok := multicloudsdk.GetMappingListExternalLocationDetailsMetadataSortByEnum(s.sortBy); !ok {
			return fmt.Errorf("ExternalLocationDetailsMetadata annotation %q has unsupported sortBy %q", externalLocationDetailsMetadataSortByAnnotation, s.sortBy)
		}
	}
	return nil
}

func (s externalLocationDetailsMetadataSelector) listRequest() multicloudsdk.ListExternalLocationDetailsMetadataRequest {
	subscriptionServiceName, _ := multicloudsdk.GetMappingListExternalLocationDetailsMetadataSubscriptionServiceNameEnum(s.subscriptionServiceName)
	request := multicloudsdk.ListExternalLocationDetailsMetadataRequest{
		SubscriptionId:          common.String(s.subscriptionID),
		SubscriptionServiceName: subscriptionServiceName,
	}
	if s.entityType != "" {
		request.EntityType, _ = multicloudsdk.GetMappingListExternalLocationDetailsMetadataEntityTypeEnum(s.entityType)
	}
	if s.compartmentID != "" {
		request.CompartmentId = common.String(s.compartmentID)
	}
	if s.linkedCompartmentID != "" {
		request.LinkedCompartmentId = common.String(s.linkedCompartmentID)
	}
	if s.externalLocation != "" {
		request.ExternalLocation = common.String(s.externalLocation)
	}
	if s.logicalZone != "" {
		request.LogicalZone = common.String(s.logicalZone)
	}
	if s.clusterPlacementGroupID != "" {
		request.ClusterPlacementGroupId = common.String(s.clusterPlacementGroupID)
	}
	if s.sortOrder != "" {
		request.SortOrder, _ = multicloudsdk.GetMappingListExternalLocationDetailsMetadataSortOrderEnum(s.sortOrder)
	}
	if s.sortBy != "" {
		request.SortBy, _ = multicloudsdk.GetMappingListExternalLocationDetailsMetadataSortByEnum(s.sortBy)
	}
	return request
}

func (s externalLocationDetailsMetadataSelector) selectOne(items []multicloudsdk.ExternalLocationsMetadatumSummary) (multicloudsdk.ExternalLocationsMetadatumSummary, error) {
	matches := make([]multicloudsdk.ExternalLocationsMetadatumSummary, 0, 1)
	for _, item := range items {
		if s.matches(item) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return multicloudsdk.ExternalLocationsMetadatumSummary{}, errors.New("ExternalLocationDetailsMetadata list returned no item matching the configured annotations")
	case 1:
		return matches[0], nil
	default:
		return multicloudsdk.ExternalLocationsMetadatumSummary{}, fmt.Errorf("ExternalLocationDetailsMetadata annotations matched %d items; add a more specific selector annotation", len(matches))
	}
}

func (s externalLocationDetailsMetadataSelector) matches(item multicloudsdk.ExternalLocationsMetadatumSummary) bool {
	externalLocation := item.ExternalLocation
	return optionalEqual(s.externalLocation, stringValue(externalLocationCspRegion(externalLocation))) &&
		optionalEqual(s.logicalZone, stringValue(item.OciLogicalAd), stringValue(externalLocationCspLogicalAZ(externalLocation))) &&
		optionalEqual(s.clusterPlacementGroupID, stringValue(item.ClusterPlacementGroupId), stringValue(item.CpgId)) &&
		optionalEqual(s.ociRegion, stringValue(item.OciRegion)) &&
		optionalEqual(s.ociPhysicalAD, stringValue(item.OciPhysicalAd)) &&
		optionalEqual(s.ociLogicalAD, stringValue(item.OciLogicalAd)) &&
		optionalEqual(s.cspRegion, stringValue(externalLocationCspRegion(externalLocation))) &&
		optionalEqual(s.cspPhysicalAZ, stringValue(externalLocationCspPhysicalAZ(externalLocation))) &&
		optionalEqual(s.cspLogicalAZ, stringValue(externalLocationCspLogicalAZ(externalLocation)))
}

func (s externalLocationDetailsMetadataSelector) identity(item multicloudsdk.ExternalLocationsMetadatumSummary) string {
	externalLocation := item.ExternalLocation
	parts := []string{
		s.subscriptionID,
		normalizeIdentityPart(s.subscriptionServiceName),
		normalizeIdentityPart(s.entityType),
		s.compartmentID,
		s.linkedCompartmentID,
		stringValue(item.OciRegion),
		stringValue(item.OciPhysicalAd),
		stringValue(item.OciLogicalAd),
		stringValue(item.ClusterPlacementGroupId),
		stringValue(item.CpgId),
		stringValue(externalLocationCspRegion(externalLocation)),
		stringValue(externalLocationCspPhysicalAZ(externalLocation)),
		stringValue(externalLocationCspLogicalAZ(externalLocation)),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "externallocationdetailsmetadata:" + hex.EncodeToString(sum[:16])
}

func markExternalLocationDetailsMetadataFailed(
	resource *multicloudv1beta1.ExternalLocationDetailsMetadata,
	err error,
	opcRequestID string,
	log loggerutil.OSOKLogger,
) {
	status := &resource.Status.OsokStatus
	servicemanager.ClearAsyncOperation(status)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if opcRequestID != "" {
		servicemanager.SetOpcRequestID(status, opcRequestID)
	} else if requestID := servicemanager.ErrorOpcRequestID(err); requestID != "" {
		servicemanager.SetOpcRequestID(status, requestID)
	} else {
		status.OpcRequestID = ""
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), log)
}

func markExternalLocationDetailsMetadataObserved(
	resource *multicloudv1beta1.ExternalLocationDetailsMetadata,
	identity string,
	opcRequestID string,
	log loggerutil.OSOKLogger,
) {
	status := &resource.Status.OsokStatus
	status.Ocid = shared.OCID(identity)
	servicemanager.ClearAsyncOperation(status)
	servicemanager.SetOpcRequestID(status, opcRequestID)
	status.Message = externalLocationDetailsMetadataObservedMessage
	status.Reason = string(shared.Active)
	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", externalLocationDetailsMetadataObservedMessage, log)
}

func markExternalLocationDetailsMetadataDeleted(
	resource *multicloudv1beta1.ExternalLocationDetailsMetadata,
	log loggerutil.OSOKLogger,
) {
	status := &resource.Status.OsokStatus
	status.Ocid = ""
	servicemanager.ClearAsyncOperation(status)
	status.OpcRequestID = ""
	status.Message = externalLocationDetailsMetadataDeletedMessage
	status.Reason = string(shared.Terminating)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.DeletedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", externalLocationDetailsMetadataDeletedMessage, log)
}

func annotationValue(annotations map[string]string, key string) string {
	return strings.TrimSpace(annotations[key])
}

func optionalEqual(want string, candidates ...string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}
	for _, candidate := range candidates {
		if strings.EqualFold(want, strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func normalizeIdentityPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func externalLocationCspRegion(value *multicloudsdk.ExternalLocationDetail) *string {
	if value == nil {
		return nil
	}
	return value.CspRegion
}

func externalLocationCspPhysicalAZ(value *multicloudsdk.ExternalLocationDetail) *string {
	if value == nil {
		return nil
	}
	return value.CspPhysicalAz
}

func externalLocationCspLogicalAZ(value *multicloudsdk.ExternalLocationDetail) *string {
	if value == nil {
		return nil
	}
	return value.CspLogicalAz
}
