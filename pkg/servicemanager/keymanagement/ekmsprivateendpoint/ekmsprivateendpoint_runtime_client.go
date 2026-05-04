/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ekmsprivateendpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	keymanagementsdk "github.com/oracle/oci-go-sdk/v65/keymanagement"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
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

type ekmsPrivateEndpointOCIClient interface {
	GetEkmsPrivateEndpoint(context.Context, keymanagementsdk.GetEkmsPrivateEndpointRequest) (keymanagementsdk.GetEkmsPrivateEndpointResponse, error)
	ListEkmsPrivateEndpoints(context.Context, keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error)
	DeleteEkmsPrivateEndpoint(context.Context, keymanagementsdk.DeleteEkmsPrivateEndpointRequest) (keymanagementsdk.DeleteEkmsPrivateEndpointResponse, error)
}

type ekmsPrivateEndpointListFunc func(context.Context, keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error)

type ekmsPrivateEndpointRuntimeClient struct {
	delegate EkmsPrivateEndpointServiceClient
	sdk      ekmsPrivateEndpointOCIClient
	log      loggerutil.OSOKLogger
	initErr  error
}

func init() {
	registerEkmsPrivateEndpointRuntimeHooksMutator(func(manager *EkmsPrivateEndpointServiceManager, hooks *EkmsPrivateEndpointRuntimeHooks) {
		applyEkmsPrivateEndpointRuntimeHooks(manager, hooks)
		appendEkmsPrivateEndpointRuntimeWrapper(manager, hooks)
	})
}

func applyEkmsPrivateEndpointRuntimeHooks(_ *EkmsPrivateEndpointServiceManager, hooks *EkmsPrivateEndpointRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newEkmsPrivateEndpointRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint, namespace string) (any, error) {
		return buildEkmsPrivateEndpointCreateDetails(ctx, resource, namespace)
	}

	listCall := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request keymanagementsdk.ListEkmsPrivateEndpointsRequest) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error) {
		return listEkmsPrivateEndpointsAllPages(ctx, request, listCall)
	}
}

func appendEkmsPrivateEndpointRuntimeWrapper(manager *EkmsPrivateEndpointServiceManager, hooks *EkmsPrivateEndpointRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate EkmsPrivateEndpointServiceClient) EkmsPrivateEndpointServiceClient {
		return newEkmsPrivateEndpointRuntimeClient(manager, delegate)
	})
}

func newEkmsPrivateEndpointRuntimeClient(manager *EkmsPrivateEndpointServiceManager, delegate EkmsPrivateEndpointServiceClient) *ekmsPrivateEndpointRuntimeClient {
	runtimeClient := &ekmsPrivateEndpointRuntimeClient{delegate: delegate}
	if manager == nil {
		return runtimeClient
	}

	runtimeClient.log = manager.Log
	if manager.Provider == nil {
		return runtimeClient
	}
	sdkClient, err := keymanagementsdk.NewEkmClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		runtimeClient.initErr = fmt.Errorf("initialize EkmsPrivateEndpoint OCI client: %w", err)
		return runtimeClient
	}
	runtimeClient.sdk = sdkClient
	return runtimeClient
}

func newEkmsPrivateEndpointRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "subnetId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"definedTags", "displayName", "freeformTags"},
			Mutable:         []string{"definedTags", "displayName", "freeformTags"},
			ForceNew:        []string{"caBundle", "compartmentId", "externalKeyManagerIp", "port", "subnetId"},
			ConflictsWith:   map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
	}
}

func buildEkmsPrivateEndpointCreateDetails(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint, namespace string) (keymanagementsdk.CreateEkmsPrivateEndpointDetails, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return keymanagementsdk.CreateEkmsPrivateEndpointDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return keymanagementsdk.CreateEkmsPrivateEndpointDetails{}, fmt.Errorf("marshal resolved EkmsPrivateEndpoint spec: %w", err)
	}

	var details keymanagementsdk.CreateEkmsPrivateEndpointDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return keymanagementsdk.CreateEkmsPrivateEndpointDetails{}, fmt.Errorf("decode EkmsPrivateEndpoint create body: %w", err)
	}
	return details, nil
}

func listEkmsPrivateEndpointsAllPages(
	ctx context.Context,
	request keymanagementsdk.ListEkmsPrivateEndpointsRequest,
	list ekmsPrivateEndpointListFunc,
) (keymanagementsdk.ListEkmsPrivateEndpointsResponse, error) {
	if list == nil {
		return keymanagementsdk.ListEkmsPrivateEndpointsResponse{}, fmt.Errorf("EkmsPrivateEndpoint list operation is not configured")
	}

	var combined keymanagementsdk.ListEkmsPrivateEndpointsResponse
	seenPages := map[string]bool{}
	nextPage := request.Page
	for {
		pageRequest := request
		pageRequest.Page = nextPage
		if page := stringValue(pageRequest.Page); page != "" {
			if seenPages[page] {
				return combined, fmt.Errorf("EkmsPrivateEndpoint list pagination repeated page token %q", page)
			}
			seenPages[page] = true
		}

		response, err := list(ctx, pageRequest)
		if err != nil {
			return combined, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		if stringValue(response.OpcNextPage) == "" {
			return combined, nil
		}
		nextPage = response.OpcNextPage
	}
}

func (c *ekmsPrivateEndpointRuntimeClient) CreateOrUpdate(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("EkmsPrivateEndpoint delegate is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *ekmsPrivateEndpointRuntimeClient) Delete(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}
	if c.sdk == nil {
		return false, fmt.Errorf("EkmsPrivateEndpoint OCI client is not configured")
	}

	current, found, err := c.resolveEkmsPrivateEndpoint(ctx, resource)
	if err != nil {
		return false, c.markEkmsPrivateEndpointDeleteError(resource, err)
	}
	if !found {
		c.markEkmsPrivateEndpointDeleted(resource, "OCI EkmsPrivateEndpoint no longer exists")
		return true, nil
	}

	return c.deleteResolvedEkmsPrivateEndpoint(ctx, resource, current)
}

func (c *ekmsPrivateEndpointRuntimeClient) deleteResolvedEkmsPrivateEndpoint(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint, current keymanagementsdk.EkmsPrivateEndpoint) (bool, error) {
	c.syncEkmsPrivateEndpointStatus(resource, current)
	switch lifecycleState := strings.ToUpper(string(current.LifecycleState)); lifecycleState {
	case "DELETED":
		c.markEkmsPrivateEndpointDeleted(resource, "OCI EkmsPrivateEndpoint deleted")
		return true, nil
	case "DELETING":
		c.markEkmsPrivateEndpointCondition(resource, shared.Terminating, ekmsPrivateEndpointLifecycleMessage(lifecycleState))
		return false, nil
	}

	currentID := stringValue(current.Id)
	if currentID == "" {
		return false, fmt.Errorf("EkmsPrivateEndpoint identity is not recorded")
	}

	response, err := c.sdk.DeleteEkmsPrivateEndpoint(ctx, keymanagementsdk.DeleteEkmsPrivateEndpointRequest{
		EkmsPrivateEndpointId: common.String(currentID),
	})
	if err != nil {
		return c.handleEkmsPrivateEndpointDeleteRequestError(ctx, resource, currentID, err)
	}

	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	c.markEkmsPrivateEndpointWorkRequestPending(resource, response.OpcWorkRequestId, "OCI EkmsPrivateEndpoint delete is in progress")
	return c.confirmEkmsPrivateEndpointDelete(ctx, resource, currentID)
}

func (c *ekmsPrivateEndpointRuntimeClient) handleEkmsPrivateEndpointDeleteRequestError(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint, currentID string, err error) (bool, error) {
	if isEkmsPrivateEndpointUnambiguousNotFound(err) {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		c.markEkmsPrivateEndpointDeleted(resource, "OCI EkmsPrivateEndpoint deleted")
		return true, nil
	}
	if errorutil.ClassifyDeleteError(err).IsConflict() {
		return c.confirmEkmsPrivateEndpointDelete(ctx, resource, currentID)
	}
	return false, c.markEkmsPrivateEndpointDeleteError(resource, err)
}

func (c *ekmsPrivateEndpointRuntimeClient) resolveEkmsPrivateEndpoint(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint) (keymanagementsdk.EkmsPrivateEndpoint, bool, error) {
	currentID := currentEkmsPrivateEndpointID(resource)
	if currentID != "" {
		current, found, err := c.getEkmsPrivateEndpoint(ctx, currentID)
		if err != nil || found {
			return current, found, err
		}
		return keymanagementsdk.EkmsPrivateEndpoint{}, false, nil
	}
	return c.lookupEkmsPrivateEndpointBySpec(ctx, resource)
}

func (c *ekmsPrivateEndpointRuntimeClient) getEkmsPrivateEndpoint(ctx context.Context, endpointID string) (keymanagementsdk.EkmsPrivateEndpoint, bool, error) {
	response, err := c.sdk.GetEkmsPrivateEndpoint(ctx, keymanagementsdk.GetEkmsPrivateEndpointRequest{
		EkmsPrivateEndpointId: common.String(endpointID),
	})
	if err != nil {
		if isEkmsPrivateEndpointUnambiguousNotFound(err) {
			return keymanagementsdk.EkmsPrivateEndpoint{}, false, nil
		}
		return keymanagementsdk.EkmsPrivateEndpoint{}, false, normalizeEkmsPrivateEndpointOCIError(err)
	}
	return response.EkmsPrivateEndpoint, true, nil
}

func (c *ekmsPrivateEndpointRuntimeClient) lookupEkmsPrivateEndpointBySpec(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint) (keymanagementsdk.EkmsPrivateEndpoint, bool, error) {
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	subnetID := strings.TrimSpace(resource.Spec.SubnetId)
	if compartmentID == "" || displayName == "" || subnetID == "" {
		return keymanagementsdk.EkmsPrivateEndpoint{}, false, nil
	}

	response, err := listEkmsPrivateEndpointsAllPages(ctx, keymanagementsdk.ListEkmsPrivateEndpointsRequest{
		CompartmentId: common.String(compartmentID),
	}, c.sdk.ListEkmsPrivateEndpoints)
	if err != nil {
		return keymanagementsdk.EkmsPrivateEndpoint{}, false, normalizeEkmsPrivateEndpointOCIError(err)
	}

	matchedIDs := matchingEkmsPrivateEndpointIDs(response.Items, displayName, subnetID)

	switch len(matchedIDs) {
	case 0:
		return keymanagementsdk.EkmsPrivateEndpoint{}, false, nil
	case 1:
		return c.getEkmsPrivateEndpoint(ctx, matchedIDs[0])
	default:
		return keymanagementsdk.EkmsPrivateEndpoint{}, false, fmt.Errorf("EkmsPrivateEndpoint list returned multiple matching resources for displayName %q", displayName)
	}
}

func matchingEkmsPrivateEndpointIDs(items []keymanagementsdk.EkmsPrivateEndpointSummary, displayName string, subnetID string) []string {
	var matchedIDs []string
	for _, item := range items {
		if !ekmsPrivateEndpointSummaryMatches(item, displayName, subnetID) {
			continue
		}
		if id := strings.TrimSpace(stringValue(item.Id)); id != "" {
			matchedIDs = append(matchedIDs, id)
		}
	}
	return matchedIDs
}

func ekmsPrivateEndpointSummaryMatches(item keymanagementsdk.EkmsPrivateEndpointSummary, displayName string, subnetID string) bool {
	if strings.EqualFold(string(item.LifecycleState), "DELETED") {
		return false
	}
	return strings.TrimSpace(stringValue(item.DisplayName)) == displayName &&
		strings.TrimSpace(stringValue(item.SubnetId)) == subnetID
}

func (c *ekmsPrivateEndpointRuntimeClient) confirmEkmsPrivateEndpointDelete(ctx context.Context, resource *keymanagementv1beta1.EkmsPrivateEndpoint, endpointID string) (bool, error) {
	current, found, err := c.getEkmsPrivateEndpoint(ctx, endpointID)
	if err != nil {
		return false, c.markEkmsPrivateEndpointDeleteError(resource, err)
	}
	if !found {
		c.markEkmsPrivateEndpointDeleted(resource, "OCI EkmsPrivateEndpoint deleted")
		return true, nil
	}

	c.syncEkmsPrivateEndpointStatus(resource, current)
	switch lifecycleState := strings.ToUpper(string(current.LifecycleState)); lifecycleState {
	case "DELETED":
		c.markEkmsPrivateEndpointDeleted(resource, "OCI EkmsPrivateEndpoint deleted")
		return true, nil
	case "DELETING":
		c.markEkmsPrivateEndpointCondition(resource, shared.Terminating, ekmsPrivateEndpointLifecycleMessage(lifecycleState))
		return false, nil
	default:
		c.markEkmsPrivateEndpointCondition(resource, shared.Terminating, "OCI EkmsPrivateEndpoint delete is in progress")
		return false, nil
	}
}

func currentEkmsPrivateEndpointID(resource *keymanagementv1beta1.EkmsPrivateEndpoint) string {
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func isEkmsPrivateEndpointUnambiguousNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func normalizeEkmsPrivateEndpointOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, ok := err.(common.ServiceError); ok {
		if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
			return normalized
		}
	}
	return err
}

func normalizeEkmsPrivateEndpointDeleteError(err error) error {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsAuthShapedNotFound() {
		return fmt.Errorf(
			"EkmsPrivateEndpoint delete returned authorization-shaped not-found (status %s code %s); finalizer retained until OCI deletion is unambiguous",
			classification.HTTPStatusCodeString(),
			classification.ErrorCodeString(),
		)
	}
	return normalizeEkmsPrivateEndpointOCIError(err)
}

func (c *ekmsPrivateEndpointRuntimeClient) markEkmsPrivateEndpointDeleteError(resource *keymanagementv1beta1.EkmsPrivateEndpoint, err error) error {
	normalized := normalizeEkmsPrivateEndpointDeleteError(err)
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	servicemanager.RecordErrorOpcRequestID(status, normalized)
	status.Message = normalized.Error()
	status.Reason = string(shared.Failed)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", normalized.Error(), c.log)
	return normalized
}

func (c *ekmsPrivateEndpointRuntimeClient) syncEkmsPrivateEndpointStatus(resource *keymanagementv1beta1.EkmsPrivateEndpoint, endpoint keymanagementsdk.EkmsPrivateEndpoint) {
	resource.Status.Id = stringValue(endpoint.Id)
	resource.Status.CompartmentId = stringValue(endpoint.CompartmentId)
	resource.Status.SubnetId = stringValue(endpoint.SubnetId)
	resource.Status.DisplayName = stringValue(endpoint.DisplayName)
	resource.Status.TimeCreated = sdkTimeString(endpoint.TimeCreated)
	resource.Status.LifecycleState = string(endpoint.LifecycleState)
	resource.Status.ExternalKeyManagerIp = stringValue(endpoint.ExternalKeyManagerIp)
	resource.Status.TimeUpdated = sdkTimeString(endpoint.TimeUpdated)
	resource.Status.FreeformTags = cloneStringMap(endpoint.FreeformTags)
	resource.Status.DefinedTags = convertDefinedTags(endpoint.DefinedTags)
	resource.Status.LifecycleDetails = stringValue(endpoint.LifecycleDetails)
	resource.Status.Port = intValue(endpoint.Port)
	resource.Status.CaBundle = stringValue(endpoint.CaBundle)
	resource.Status.PrivateEndpointIp = stringValue(endpoint.PrivateEndpointIp)
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func (c *ekmsPrivateEndpointRuntimeClient) markEkmsPrivateEndpointDeleted(resource *keymanagementv1beta1.EkmsPrivateEndpoint, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.DeletedAt = &now
	if status.Ocid == "" && resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *ekmsPrivateEndpointRuntimeClient) markEkmsPrivateEndpointCondition(resource *keymanagementv1beta1.EkmsPrivateEndpoint, conditionType shared.OSOKConditionType, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(conditionType)
	status.UpdatedAt = &now
	if status.Ocid == "" && resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.CreatedAt == nil && resource.Status.Id != "" {
		status.CreatedAt = &now
	}
	if conditionType == shared.Terminating {
		servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       strings.ToUpper(resource.Status.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}, c.log)
		return
	}
	conditionStatus := v1.ConditionTrue
	if conditionType == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, conditionType, conditionStatus, "", message, c.log)
}

func (c *ekmsPrivateEndpointRuntimeClient) markEkmsPrivateEndpointWorkRequestPending(resource *keymanagementv1beta1.EkmsPrivateEndpoint, workRequestID *string, message string) {
	if stringValue(workRequestID) == "" {
		return
	}

	now := metav1.Now()
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   stringValue(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}, c.log)
}

func ekmsPrivateEndpointLifecycleMessage(lifecycleState string) string {
	switch strings.ToUpper(lifecycleState) {
	case "CREATING":
		return "OCI EkmsPrivateEndpoint provisioning is in progress"
	case "ACTIVE":
		return "OCI EkmsPrivateEndpoint is active"
	case "DELETING":
		return "OCI EkmsPrivateEndpoint delete is in progress"
	case "DELETED":
		return "OCI EkmsPrivateEndpoint deleted"
	case "FAILED":
		return "OCI EkmsPrivateEndpoint lifecycle state is failed"
	default:
		return fmt.Sprintf("EkmsPrivateEndpoint lifecycle state %q is not modeled", lifecycleState)
	}
}

func convertDefinedTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		innerConverted := make(shared.MapValue, len(values))
		for key, value := range values {
			innerConverted[key] = fmt.Sprint(value)
		}
		converted[namespace] = innerConverted
	}
	return converted
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02T15:04:05.999999999Z07:00")
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
