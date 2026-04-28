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
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
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
	ListVcns(ctx context.Context, request coresdk.ListVcnsRequest) (coresdk.ListVcnsResponse, error)
	UpdateVcn(ctx context.Context, request coresdk.UpdateVcnRequest) (coresdk.UpdateVcnResponse, error)
	DeleteVcn(ctx context.Context, request coresdk.DeleteVcnRequest) (coresdk.DeleteVcnResponse, error)
}

type vcnGeneratedParityClient struct {
	manager  *VcnServiceManager
	delegate VcnServiceClient
	client   vcnOCIClient
	initErr  error
}

func init() {
	registerVcnRuntimeHooksMutator(func(manager *VcnServiceManager, hooks *VcnRuntimeHooks) {
		applyVcnRuntimeHooks(manager, hooks, nil)
	})
}

func applyVcnRuntimeHooks(
	manager *VcnServiceManager,
	hooks *VcnRuntimeHooks,
	client vcnOCIClient,
) {
	if hooks == nil {
		return
	}

	if hooks.Semantics != nil {
		semantics := *hooks.Semantics
		mutation := semantics.Mutation
		mutation.ForceNew = nil
		mutation.Mutable = appendUniqueVcnMutationPaths(mutation.Mutable, "securityAttributes", "isZprOnly")
		semantics.Mutation = mutation
		hooks.Semantics = &semantics
	}

	hooks.BuildUpdateBody = buildVcnUpdateBody
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedVcnIdentity
	hooks.ParityHooks.NormalizeDesiredState = func(resource *corev1beta1.Vcn, currentResponse any) {
		current, ok := vcnFromResponse(currentResponse)
		if !ok {
			return
		}
		normalizeEquivalentVcnCreateOnlyLists(resource, current)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = func(resource *corev1beta1.Vcn, currentResponse any) error {
		current, ok := vcnFromResponse(currentResponse)
		if !ok {
			return fmt.Errorf("unexpected Vcn current response type %T", currentResponse)
		}
		return validateCreateOnlyDrift(resource.Spec, current)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate VcnServiceClient) VcnServiceClient {
		return newVcnTrackedRecreateClient(manager, delegate, client)
	})
}

func newVcnTrackedRecreateClient(
	manager *VcnServiceManager,
	delegate VcnServiceClient,
	client vcnOCIClient,
) VcnServiceClient {
	runtimeClient := &vcnGeneratedParityClient{
		manager:  manager,
		delegate: delegate,
		client:   client,
	}
	if runtimeClient.client != nil {
		return runtimeClient
	}

	sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
	runtimeClient.client = sdkClient
	if err != nil {
		runtimeClient.initErr = fmt.Errorf("initialize Vcn OCI client: %w", err)
	}
	return runtimeClient
}

func (c *vcnGeneratedParityClient) CreateOrUpdate(
	ctx context.Context,
	resource *corev1beta1.Vcn,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("vcn parity delegate is not configured")
	}

	trackedID := currentVcnID(resource)
	explicitRecreate := false
	if trackedID != "" {
		if c.initErr != nil {
			return c.fail(resource, c.initErr)
		}

		current, err := c.get(ctx, trackedID)
		if err != nil {
			if isReadNotFoundOCI(err) {
				clearTrackedVcnIdentity(resource)
				explicitRecreate = true
			} else {
				return c.fail(resource, normalizeOCIError(err))
			}
		} else if current.LifecycleState == coresdk.VcnLifecycleStateTerminated {
			clearTrackedVcnIdentity(resource)
			explicitRecreate = true
		}
	}

	previousStatus := resource.Status
	clearVcnProjectedStatus(resource)

	delegateCtx := ctx
	if explicitRecreate {
		delegateCtx = generatedruntime.WithSkipExistingBeforeCreate(delegateCtx)
	}

	response, err := c.delegate.CreateOrUpdate(delegateCtx, resource, req)
	if err != nil {
		restoreVcnStatus(resource, previousStatus)
	}
	return response, err
}

func (c *vcnGeneratedParityClient) Delete(ctx context.Context, resource *corev1beta1.Vcn) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("vcn parity delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *vcnGeneratedParityClient) get(ctx context.Context, ocid string) (coresdk.Vcn, error) {
	response, err := c.client.GetVcn(ctx, coresdk.GetVcnRequest{
		VcnId: common.String(ocid),
	})
	if err != nil {
		return coresdk.Vcn{}, err
	}
	return response.Vcn, nil
}

func normalizeEquivalentVcnCreateOnlyLists(resource *corev1beta1.Vcn, current coresdk.Vcn) {
	if resource == nil {
		return
	}

	if normalizedStringSlicesEqual(current.CidrBlocks, resource.Spec.CidrBlocks) {
		resource.Spec.CidrBlocks = append([]string(nil), current.CidrBlocks...)
	}

	if normalizedStringSlicesEqual(current.Ipv6PrivateCidrBlocks, resource.Spec.Ipv6PrivateCidrBlocks) {
		resource.Spec.Ipv6PrivateCidrBlocks = append([]string(nil), current.Ipv6PrivateCidrBlocks...)
	}

	if normalizedStringSlicesEqual(current.Byoipv6CidrBlocks, desiredByoipv6Blocks(resource.Spec.Byoipv6CidrDetails)) {
		resource.Spec.Byoipv6CidrDetails = reorderByoipv6Details(resource.Spec.Byoipv6CidrDetails, current.Byoipv6CidrBlocks)
	}
}

func buildVcnUpdateBody(
	_ context.Context,
	resource *corev1beta1.Vcn,
	_ string,
	currentResponse any,
) (any, bool, error) {
	current, ok := vcnFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("unexpected Vcn current response type %T", currentResponse)
	}

	details := coresdk.UpdateVcnDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}

	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := convertSharedMapValuesToOCI(resource.Spec.DefinedTags)
		if !nestedMapEqual(current.DefinedTags, desiredDefinedTags) {
			details.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}

	if resource.Spec.FreeformTags != nil {
		desiredFreeformTags := copyStringMap(resource.Spec.FreeformTags)
		if !stringMapEqual(current.FreeformTags, desiredFreeformTags) {
			details.FreeformTags = desiredFreeformTags
			updateNeeded = true
		}
	}

	if resource.Spec.SecurityAttributes != nil {
		desiredSecurityAttributes := convertSharedMapValuesToOCI(resource.Spec.SecurityAttributes)
		if !nestedMapEqual(current.SecurityAttributes, desiredSecurityAttributes) {
			details.SecurityAttributes = desiredSecurityAttributes
			updateNeeded = true
		}
	}

	if boolPtrValue(current.IsZprOnly) != resource.Spec.IsZprOnly {
		details.IsZprOnly = common.Bool(resource.Spec.IsZprOnly)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil, false, nil
	}
	return details, true, nil
}

func clearVcnProjectedStatus(resource *corev1beta1.Vcn) {
	if resource == nil {
		return
	}

	resource.Status = corev1beta1.VcnStatus{
		OsokStatus: resource.Status.OsokStatus,
		Id:         resource.Status.Id,
	}
}

func restoreVcnStatus(resource *corev1beta1.Vcn, previous corev1beta1.VcnStatus) {
	if resource == nil {
		return
	}

	failedStatus := resource.Status.OsokStatus
	resource.Status = previous
	resource.Status.OsokStatus = failedStatus
}

func (c *vcnGeneratedParityClient) fail(resource *corev1beta1.Vcn, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1.NewTime(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func clearTrackedVcnIdentity(resource *corev1beta1.Vcn) {
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func vcnFromResponse(response any) (coresdk.Vcn, bool) {
	switch typed := response.(type) {
	case coresdk.Vcn:
		return typed, true
	case coresdk.CreateVcnResponse:
		return typed.Vcn, true
	case coresdk.GetVcnResponse:
		return typed.Vcn, true
	case coresdk.UpdateVcnResponse:
		return typed.Vcn, true
	default:
		return coresdk.Vcn{}, false
	}
}

func currentVcnID(resource *corev1beta1.Vcn) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func vcnLifecycleIsRetryable(state coresdk.VcnLifecycleStateEnum) bool {
	switch state {
	case coresdk.VcnLifecycleStateProvisioning,
		coresdk.VcnLifecycleStateUpdating,
		coresdk.VcnLifecycleStateTerminating:
		return true
	default:
		return false
	}
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

func reorderByoipv6Details(details []corev1beta1.VcnByoipv6CidrDetail, currentBlocks []string) []corev1beta1.VcnByoipv6CidrDetail {
	if len(details) == 0 {
		return nil
	}

	ordered := make([]corev1beta1.VcnByoipv6CidrDetail, 0, len(details))
	used := make([]bool, len(details))
	for _, currentBlock := range currentBlocks {
		for index, detail := range details {
			if used[index] || strings.TrimSpace(detail.Ipv6CidrBlock) != strings.TrimSpace(currentBlock) {
				continue
			}
			ordered = append(ordered, detail)
			used[index] = true
			break
		}
	}

	for index, detail := range details {
		if used[index] {
			continue
		}
		ordered = append(ordered, detail)
	}
	return ordered
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

func isReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func isDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}

func convertSharedMapValuesToOCI(values map[string]shared.MapValue) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}

	converted := make(map[string]map[string]interface{}, len(values))
	for outerKey, outerValue := range values {
		innerMap := make(map[string]interface{}, len(outerValue))
		for innerKey, innerValue := range outerValue {
			innerMap[innerKey] = innerValue
		}
		converted[outerKey] = innerMap
	}
	return converted
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}

	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func stringMapEqual(left map[string]string, right map[string]string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func nestedMapEqual(left map[string]map[string]interface{}, right map[string]map[string]interface{}) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func appendUniqueVcnMutationPaths(existing []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(extras))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range extras {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}
