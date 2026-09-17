/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package externallocationmappingmetadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	multicloudsdk "github.com/oracle/oci-go-sdk/v65/multicloud"
	multicloudv1beta1 "github.com/oracle/oci-service-operator/api/multicloud/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	externalLocationMappingMetadataAnnotationPrefix = "multicloud.oracle.com/"

	externalLocationMappingMetadataCompartmentIDAnnotation       = externalLocationMappingMetadataAnnotationPrefix + "compartment-id"
	externalLocationMappingMetadataSubscriptionIDAnnotation      = externalLocationMappingMetadataAnnotationPrefix + "subscription-id"
	externalLocationMappingMetadataSubscriptionServiceAnnotation = externalLocationMappingMetadataAnnotationPrefix + "subscription-service-name"
	externalLocationMappingMetadataCSPRegionAnnotation           = externalLocationMappingMetadataAnnotationPrefix + "csp-region"
	externalLocationMappingMetadataCSPPhysicalAZAnnotation       = externalLocationMappingMetadataAnnotationPrefix + "csp-physical-az"
	externalLocationMappingMetadataOCIRegionAnnotation           = externalLocationMappingMetadataAnnotationPrefix + "oci-region"
	externalLocationMappingMetadataOCIPhysicalADAnnotation       = externalLocationMappingMetadataAnnotationPrefix + "oci-physical-ad"
	externalLocationMappingMetadataOCILogicalADAnnotation        = externalLocationMappingMetadataAnnotationPrefix + "oci-logical-ad"
	externalLocationMappingMetadataSyntheticIDPrefix             = "multicloud/externalLocationMappingMetadata/"
	externalLocationMappingMetadataSyntheticIDVersion            = "v1/"
	externalLocationMappingMetadataActiveMessage                 = "observed OCI external location mapping metadata"
	externalLocationMappingMetadataReadOnlyDeleteAcceptedMessage = "ExternalLocationMappingMetadata is read-only OCI metadata; no OCI delete operation is available"
)

type listExternalLocationMappingMetadataFunc func(context.Context, multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error)

type externalLocationMappingMetadataRuntimeClient struct {
	list    listExternalLocationMappingMetadataFunc
	initErr error
	log     loggerutil.OSOKLogger
}

type externalLocationMappingMetadataQuery struct {
	compartmentID             string
	subscriptionID            string
	subscriptionServiceNames  []multicloudsdk.SubscriptionTypeEnum
	subscriptionServiceLabels []string
	selector                  externalLocationMappingMetadataSelector
}

type externalLocationMappingMetadataSelector struct {
	cspRegion     string
	cspPhysicalAZ string
	ociRegion     string
	ociPhysicalAD string
	ociLogicalAD  string
}

type externalLocationMappingMetadataIdentity struct {
	compartmentID             string
	subscriptionID            string
	subscriptionServiceLabels []string
	cspRegion                 string
	cspPhysicalAZ             string
	serviceName               string
	ociRegion                 string
	ociPhysicalAD             string
	ociLogicalAD              string
}

func init() {
	registerExternalLocationMappingMetadataRuntimeHooksMutator(func(manager *ExternalLocationMappingMetadataServiceManager, hooks *ExternalLocationMappingMetadataRuntimeHooks) {
		list := hooks.List.Call
		log := manager.Log
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ExternalLocationMappingMetadataServiceClient) ExternalLocationMappingMetadataServiceClient {
			return externalLocationMappingMetadataRuntimeClient{
				list:    list,
				initErr: externalLocationMappingMetadataGeneratedDelegateInitError(delegate),
				log:     log,
			}
		})
	})
}

func (c externalLocationMappingMetadataRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *multicloudv1beta1.ExternalLocationMappingMetadata,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}
	if c.list == nil {
		return c.fail(resource, fmt.Errorf("ExternalLocationMappingMetadata list operation is not configured"))
	}

	query, err := externalLocationMappingMetadataQueryFromAnnotations(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	items, opcRequestID, err := c.listAll(ctx, query)
	if err != nil {
		return c.fail(resource, err)
	}

	item, err := selectExternalLocationMappingMetadata(items, query)
	if err != nil {
		return c.fail(resource, err)
	}

	identity := query.identityFor(item)
	if err := validateExternalLocationMappingMetadataTrackedIdentity(resource, identity); err != nil {
		return c.fail(resource, err)
	}
	if err := generatedruntime.ProjectResponseBodyWithAliases(resource, item, nil); err != nil {
		return c.fail(resource, err)
	}

	markExternalLocationMappingMetadataActive(resource, identity, opcRequestID, c.log)
	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func (c externalLocationMappingMetadataRuntimeClient) Delete(_ context.Context, resource *multicloudv1beta1.ExternalLocationMappingMetadata) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}
	markExternalLocationMappingMetadataDeleted(resource, c.log)
	return true, nil
}

func externalLocationMappingMetadataGeneratedDelegateInitError(delegate ExternalLocationMappingMetadataServiceClient) error {
	if delegate == nil {
		return nil
	}

	var resource *multicloudv1beta1.ExternalLocationMappingMetadata
	_, err := delegate.Delete(context.Background(), resource)
	if err == nil || isExternalLocationMappingMetadataNilResourceProbeError(err) {
		return nil
	}
	return err
}

func isExternalLocationMappingMetadataNilResourceProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "resource is nil") || strings.Contains(message, "expected pointer resource")
}

func (c externalLocationMappingMetadataRuntimeClient) listAll(
	ctx context.Context,
	query externalLocationMappingMetadataQuery,
) ([]multicloudsdk.ExternalLocationMappingMetadatumSummary, string, error) {
	var items []multicloudsdk.ExternalLocationMappingMetadatumSummary
	var page *string
	var lastOpcRequestID string
	seenPages := map[string]struct{}{}

	for {
		request := multicloudsdk.ListExternalLocationMappingMetadataRequest{
			CompartmentId:           common.String(query.compartmentID),
			SubscriptionServiceName: append([]multicloudsdk.SubscriptionTypeEnum(nil), query.subscriptionServiceNames...),
			Page:                    page,
		}
		if query.subscriptionID != "" {
			request.SubscriptionId = common.String(query.subscriptionID)
		}

		response, err := c.list(ctx, request)
		if err != nil {
			return nil, "", err
		}
		if response.OpcRequestId != nil && strings.TrimSpace(*response.OpcRequestId) != "" {
			lastOpcRequestID = strings.TrimSpace(*response.OpcRequestId)
		}
		items = append(items, response.Items...)
		nextPage := strings.TrimSpace(stringValue(response.OpcNextPage))
		if nextPage == "" {
			return items, lastOpcRequestID, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return nil, "", fmt.Errorf("ExternalLocationMappingMetadata list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		page = common.String(nextPage)
	}
}

func externalLocationMappingMetadataQueryFromAnnotations(resource *multicloudv1beta1.ExternalLocationMappingMetadata) (externalLocationMappingMetadataQuery, error) {
	if resource == nil {
		return externalLocationMappingMetadataQuery{}, fmt.Errorf("ExternalLocationMappingMetadata resource is nil")
	}

	query := externalLocationMappingMetadataQuery{
		compartmentID:  externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataCompartmentIDAnnotation),
		subscriptionID: externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataSubscriptionIDAnnotation),
		selector: externalLocationMappingMetadataSelector{
			cspRegion:     externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataCSPRegionAnnotation),
			cspPhysicalAZ: externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataCSPPhysicalAZAnnotation),
			ociRegion:     externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataOCIRegionAnnotation),
			ociPhysicalAD: externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataOCIPhysicalADAnnotation),
			ociLogicalAD:  externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataOCILogicalADAnnotation),
		},
	}
	if query.compartmentID == "" {
		return externalLocationMappingMetadataQuery{}, fmt.Errorf("ExternalLocationMappingMetadata requires metadata annotation %q because the generated spec does not expose compartmentId", externalLocationMappingMetadataCompartmentIDAnnotation)
	}

	rawServiceNames := externalLocationMappingMetadataAnnotation(resource, externalLocationMappingMetadataSubscriptionServiceAnnotation)
	serviceNames, serviceLabels, err := parseExternalLocationMappingMetadataSubscriptionServiceNames(rawServiceNames)
	if err != nil {
		return externalLocationMappingMetadataQuery{}, err
	}
	query.subscriptionServiceNames = serviceNames
	query.subscriptionServiceLabels = serviceLabels
	return query, nil
}

func parseExternalLocationMappingMetadataSubscriptionServiceNames(raw string) ([]multicloudsdk.SubscriptionTypeEnum, []string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil, nil
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	seen := map[string]struct{}{}
	var values []multicloudsdk.SubscriptionTypeEnum
	var labels []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, ok := multicloudsdk.GetMappingSubscriptionTypeEnum(part)
		if !ok {
			return nil, nil, fmt.Errorf("ExternalLocationMappingMetadata annotation %q has unsupported value %q; supported values: %s",
				externalLocationMappingMetadataSubscriptionServiceAnnotation,
				part,
				strings.Join(multicloudsdk.GetSubscriptionTypeEnumStringValues(), ", "))
		}
		label := string(value)
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		values = append(values, value)
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return values, labels, nil
}

func externalLocationMappingMetadataAnnotation(resource *multicloudv1beta1.ExternalLocationMappingMetadata, key string) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Annotations[key])
}

func selectExternalLocationMappingMetadata(
	items []multicloudsdk.ExternalLocationMappingMetadatumSummary,
	query externalLocationMappingMetadataQuery,
) (multicloudsdk.ExternalLocationMappingMetadatumSummary, error) {
	if len(items) == 0 {
		return multicloudsdk.ExternalLocationMappingMetadatumSummary{}, fmt.Errorf("ExternalLocationMappingMetadata list returned no metadata mappings")
	}

	if !query.selector.hasSelectors() {
		if len(items) == 1 {
			return items[0], nil
		}
		return multicloudsdk.ExternalLocationMappingMetadatumSummary{}, fmt.Errorf("ExternalLocationMappingMetadata list returned %d mappings; add one of %q, %q, %q, %q, or %q to select exactly one mapping",
			len(items),
			externalLocationMappingMetadataCSPRegionAnnotation,
			externalLocationMappingMetadataCSPPhysicalAZAnnotation,
			externalLocationMappingMetadataOCIRegionAnnotation,
			externalLocationMappingMetadataOCIPhysicalADAnnotation,
			externalLocationMappingMetadataOCILogicalADAnnotation)
	}

	var matches []multicloudsdk.ExternalLocationMappingMetadatumSummary
	for _, item := range items {
		if query.selector.matches(item) && externalLocationMappingMetadataServiceNamesAllow(query.subscriptionServiceLabels, item) {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return multicloudsdk.ExternalLocationMappingMetadatumSummary{}, fmt.Errorf("ExternalLocationMappingMetadata list did not return a mapping matching the configured annotations")
	default:
		return multicloudsdk.ExternalLocationMappingMetadatumSummary{}, fmt.Errorf("ExternalLocationMappingMetadata list returned multiple mappings matching the configured annotations")
	}
}

func (s externalLocationMappingMetadataSelector) hasSelectors() bool {
	return s.cspRegion != "" ||
		s.cspPhysicalAZ != "" ||
		s.ociRegion != "" ||
		s.ociPhysicalAD != "" ||
		s.ociLogicalAD != ""
}

func (s externalLocationMappingMetadataSelector) matches(item multicloudsdk.ExternalLocationMappingMetadatumSummary) bool {
	for _, value := range s.matchValues(item) {
		if value.want != "" && value.want != value.got {
			return false
		}
	}
	return true
}

func (s externalLocationMappingMetadataSelector) matchValues(item multicloudsdk.ExternalLocationMappingMetadatumSummary) []struct {
	want string
	got  string
} {
	externalLocation := item.ExternalLocation
	var cspRegion, cspPhysicalAZ string
	if externalLocation != nil {
		cspRegion = stringValue(externalLocation.CspRegion)
		cspPhysicalAZ = stringValue(externalLocation.CspPhysicalAz)
	}
	return []struct {
		want string
		got  string
	}{
		{want: s.cspRegion, got: cspRegion},
		{want: s.cspPhysicalAZ, got: cspPhysicalAZ},
		{want: s.ociRegion, got: stringValue(item.OciRegion)},
		{want: s.ociPhysicalAD, got: stringValue(item.OciPhysicalAd)},
		{want: s.ociLogicalAD, got: stringValue(item.OciLogicalAd)},
	}
}

func externalLocationMappingMetadataServiceNamesAllow(allowed []string, item multicloudsdk.ExternalLocationMappingMetadatumSummary) bool {
	if len(allowed) == 0 || item.ExternalLocation == nil || item.ExternalLocation.ServiceName == "" {
		return true
	}
	itemServiceName := string(item.ExternalLocation.ServiceName)
	for _, serviceName := range allowed {
		if serviceName == itemServiceName {
			return true
		}
	}
	return false
}

func (q externalLocationMappingMetadataQuery) identityFor(item multicloudsdk.ExternalLocationMappingMetadatumSummary) externalLocationMappingMetadataIdentity {
	identity := externalLocationMappingMetadataIdentity{
		compartmentID:             q.compartmentID,
		subscriptionID:            q.subscriptionID,
		subscriptionServiceLabels: append([]string(nil), q.subscriptionServiceLabels...),
		ociRegion:                 stringValue(item.OciRegion),
		ociPhysicalAD:             stringValue(item.OciPhysicalAd),
		ociLogicalAD:              stringValue(item.OciLogicalAd),
	}
	if item.ExternalLocation != nil {
		identity.cspRegion = stringValue(item.ExternalLocation.CspRegion)
		identity.cspPhysicalAZ = stringValue(item.ExternalLocation.CspPhysicalAz)
		identity.serviceName = string(item.ExternalLocation.ServiceName)
	}
	return identity
}

func validateExternalLocationMappingMetadataTrackedIdentity(
	resource *multicloudv1beta1.ExternalLocationMappingMetadata,
	identity externalLocationMappingMetadataIdentity,
) error {
	if resource == nil {
		return nil
	}
	currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	if currentID == "" || currentID == string(identity.syntheticID()) {
		return nil
	}
	return fmt.Errorf("ExternalLocationMappingMetadata identity drift is not supported for read-only OCI metadata; delete and recreate the Kubernetes resource to select a different mapping")
}

func markExternalLocationMappingMetadataActive(
	resource *multicloudv1beta1.ExternalLocationMappingMetadata,
	identity externalLocationMappingMetadataIdentity,
	opcRequestID string,
	log loggerutil.OSOKLogger,
) {
	if resource == nil {
		return
	}

	status := &resource.Status.OsokStatus
	status.Ocid = identity.syntheticID()
	servicemanager.SetOpcRequestID(status, opcRequestID)
	now := metav1.Now()
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = externalLocationMappingMetadataActiveMessage + ": " + identity.displayName()
	status.Reason = string(shared.Active)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Active, corev1.ConditionTrue, "", status.Message, log)
}

func markExternalLocationMappingMetadataDeleted(resource *multicloudv1beta1.ExternalLocationMappingMetadata, log loggerutil.OSOKLogger) {
	if resource == nil {
		return
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = externalLocationMappingMetadataReadOnlyDeleteAcceptedMessage
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, log)
}

func (c externalLocationMappingMetadataRuntimeClient) fail(
	resource *multicloudv1beta1.ExternalLocationMappingMetadata,
	err error,
) (servicemanager.OSOKResponse, error) {
	markExternalLocationMappingMetadataFailure(resource, err, c.log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func markExternalLocationMappingMetadataFailure(
	resource *multicloudv1beta1.ExternalLocationMappingMetadata,
	err error,
	log loggerutil.OSOKLogger,
) {
	if resource == nil || err == nil {
		return
	}

	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), log)
}

func (identity externalLocationMappingMetadataIdentity) syntheticID() shared.OCID {
	return shared.OCID(externalLocationMappingMetadataSyntheticIDPrefix + externalLocationMappingMetadataSyntheticIDVersion + identity.fingerprint())
}

func (identity externalLocationMappingMetadataIdentity) fingerprint() string {
	parts := []string{
		identity.compartmentID,
		identity.subscriptionID,
		strings.Join(identity.subscriptionServiceLabels, ","),
		identity.cspRegion,
		identity.cspPhysicalAZ,
		identity.serviceName,
		identity.ociRegion,
		identity.ociPhysicalAD,
		identity.ociLogicalAD,
	}
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(strings.TrimSpace(part)))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (identity externalLocationMappingMetadataIdentity) displayName() string {
	parts := []string{
		"cspRegion=" + identity.cspRegion,
		"cspPhysicalAz=" + identity.cspPhysicalAZ,
		"ociRegion=" + identity.ociRegion,
		"ociPhysicalAd=" + identity.ociPhysicalAD,
		"ociLogicalAd=" + identity.ociLogicalAD,
	}
	if identity.serviceName != "" {
		parts = append(parts, "serviceName="+identity.serviceName)
	}
	return strings.Join(parts, ", ")
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
