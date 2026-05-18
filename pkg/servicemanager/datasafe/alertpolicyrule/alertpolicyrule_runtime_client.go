/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alertpolicyrule

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	alertPolicyRuleAlertPolicyIDAnnotation = "datasafe.oracle.com/alert-policy-id"
	alertPolicyRuleTrackedIDSeparator      = "/"
	alertPolicyRuleRequeueDuration         = time.Minute
)

type alertPolicyRuleOCIClient interface {
	CreateAlertPolicyRule(context.Context, datasafesdk.CreateAlertPolicyRuleRequest) (datasafesdk.CreateAlertPolicyRuleResponse, error)
	GetAlertPolicyRule(context.Context, datasafesdk.GetAlertPolicyRuleRequest) (datasafesdk.GetAlertPolicyRuleResponse, error)
	ListAlertPolicyRules(context.Context, datasafesdk.ListAlertPolicyRulesRequest) (datasafesdk.ListAlertPolicyRulesResponse, error)
	UpdateAlertPolicyRule(context.Context, datasafesdk.UpdateAlertPolicyRuleRequest) (datasafesdk.UpdateAlertPolicyRuleResponse, error)
	DeleteAlertPolicyRule(context.Context, datasafesdk.DeleteAlertPolicyRuleRequest) (datasafesdk.DeleteAlertPolicyRuleResponse, error)
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

type alertPolicyRuleIdentity struct {
	alertPolicyID string
	ruleKey       string
}

type alertPolicyRuleRuntimeClient struct {
	client  alertPolicyRuleOCIClient
	initErr error
	log     loggerutil.OSOKLogger
}

var alertPolicyRuleWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(datasafesdk.WorkRequestStatusAccepted),
		string(datasafesdk.WorkRequestStatusInProgress),
		string(datasafesdk.WorkRequestStatusCanceling),
		string(datasafesdk.WorkRequestStatusSuspending),
	},
	SucceededStatusTokens: []string{string(datasafesdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(datasafesdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(datasafesdk.WorkRequestStatusCanceled)},
	AttentionStatusTokens: []string{string(datasafesdk.WorkRequestStatusSuspended)},
	CreateActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeCreateAlertPolicyRule)},
	UpdateActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeUpdateAlertPolicyRule)},
	DeleteActionTokens:    []string{string(datasafesdk.WorkRequestOperationTypeDeleteAlertPolicyRule)},
}

func init() {
	registerAlertPolicyRuleRuntimeHooksMutator(func(manager *AlertPolicyRuleServiceManager, hooks *AlertPolicyRuleRuntimeHooks) {
		if hooks == nil {
			return
		}
		client, err := newAlertPolicyRuleOCIClient(manager)
		log := alertPolicyRuleManagerLog(manager)
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(AlertPolicyRuleServiceClient) AlertPolicyRuleServiceClient {
			return alertPolicyRuleRuntimeClient{
				client:  client,
				initErr: err,
				log:     log,
			}
		})
	})
}

func newAlertPolicyRuleOCIClient(manager *AlertPolicyRuleServiceManager) (alertPolicyRuleOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("initialize AlertPolicyRule OCI client: service manager is nil")
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize AlertPolicyRule OCI client: %w", err)
	}
	return client, nil
}

func alertPolicyRuleManagerLog(manager *AlertPolicyRuleServiceManager) loggerutil.OSOKLogger {
	if manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return manager.Log
}

func newAlertPolicyRuleServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client alertPolicyRuleOCIClient,
) AlertPolicyRuleServiceClient {
	return alertPolicyRuleRuntimeClient{
		client: client,
		log:    log,
	}
}

func (c alertPolicyRuleRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if err := c.validate(resource); err != nil {
		return c.fail(resource, err)
	}
	if err := validateAlertPolicyRuleSpec(resource.Spec); err != nil {
		return c.fail(resource, err)
	}

	identity, err := resolveAlertPolicyRuleIdentity(resource)
	if err != nil {
		return c.fail(resource, err)
	}

	if identity.ruleKey != "" {
		current, err := c.get(ctx, identity)
		switch {
		case err == nil:
			return c.reconcileExisting(ctx, resource, identity, current)
		case isUnambiguousAlertPolicyRuleNotFound(err):
			clearAlertPolicyRuleTrackedIdentity(resource)
			identity.ruleKey = ""
		default:
			return c.fail(resource, err)
		}
	}

	current, found, err := c.lookupExisting(ctx, resource, identity.alertPolicyID)
	if err != nil {
		return c.fail(resource, err)
	}
	if found {
		identity.ruleKey = alertPolicyRuleString(current.Key)
		return c.reconcileExisting(ctx, resource, identity, current)
	}

	return c.create(ctx, resource, identity)
}

func (c alertPolicyRuleRuntimeClient) Delete(ctx context.Context, resource *datasafev1beta1.AlertPolicyRule) (bool, error) {
	if err := c.validate(resource); err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}

	identity, err := resolveAlertPolicyRuleDeleteIdentity(resource)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	if identity.alertPolicyID == "" && identity.ruleKey == "" {
		markAlertPolicyRuleDeleted(resource, c.log, "OCI AlertPolicyRule identity is not recorded")
		return true, nil
	}
	if workRequestID, _, ok := currentAlertPolicyRulePendingWorkRequest(resource, shared.OSOKAsyncPhaseDelete); ok {
		return c.resumeDeleteWorkRequest(ctx, resource, identity, workRequestID)
	}
	if identity.ruleKey == "" {
		return c.deleteByDesiredMatch(ctx, resource, identity.alertPolicyID)
	}

	return c.deleteKnownRule(ctx, resource, identity)
}

func (c alertPolicyRuleRuntimeClient) deleteByDesiredMatch(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	alertPolicyID string,
) (bool, error) {
	current, found, err := c.lookupExisting(ctx, resource, alertPolicyID)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	if !found {
		markAlertPolicyRuleDeleted(resource, c.log, "OCI AlertPolicyRule no longer exists")
		return true, nil
	}

	identity := alertPolicyRuleIdentity{
		alertPolicyID: alertPolicyID,
		ruleKey:       alertPolicyRuleString(current.Key),
	}
	return c.deleteCurrentRule(ctx, resource, identity, current)
}

func (c alertPolicyRuleRuntimeClient) deleteKnownRule(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
) (bool, error) {
	current, err := c.get(ctx, identity)
	switch {
	case err == nil:
	case isUnambiguousAlertPolicyRuleNotFound(err):
		markAlertPolicyRuleDeleted(resource, c.log, "OCI AlertPolicyRule no longer exists")
		return true, nil
	default:
		_, failErr := c.fail(resource, err)
		return false, failErr
	}

	return c.deleteCurrentRule(ctx, resource, identity, current)
}

func (c alertPolicyRuleRuntimeClient) deleteCurrentRule(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	current datasafesdk.AlertPolicyRule,
) (bool, error) {
	if identity.ruleKey == "" {
		_, failErr := c.fail(resource, fmt.Errorf("AlertPolicyRule list response matched a rule without a rule key"))
		return false, failErr
	}
	if alertPolicyRuleWritePending(current.LifecycleState) {
		c.applyStatus(resource, identity, current, shared.OSOKAsyncPhaseDelete, "")
		return false, nil
	}

	response, err := c.client.DeleteAlertPolicyRule(ctx, datasafesdk.DeleteAlertPolicyRuleRequest{
		AlertPolicyId: common.String(identity.alertPolicyID),
		RuleKey:       common.String(identity.ruleKey),
	})
	if err != nil {
		return c.handleDeleteError(ctx, resource, identity, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if workRequestID := alertPolicyRuleString(response.OpcWorkRequestId); workRequestID != "" {
		return c.resumeDeleteWorkRequest(ctx, resource, identity, workRequestID)
	}

	confirmed, err := c.get(ctx, identity)
	switch {
	case err == nil:
		c.applyDeleteReadback(resource, identity, confirmed, alertPolicyRuleString(response.OpcWorkRequestId))
		return false, nil
	case isUnambiguousAlertPolicyRuleNotFound(err):
		markAlertPolicyRuleDeleted(resource, c.log, "OCI AlertPolicyRule deleted")
		servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
		return true, nil
	default:
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
}

func (c alertPolicyRuleRuntimeClient) validate(resource *datasafev1beta1.AlertPolicyRule) error {
	if c.initErr != nil {
		return c.initErr
	}
	if c.client == nil {
		return fmt.Errorf("AlertPolicyRule OCI client is not configured")
	}
	if resource == nil {
		return fmt.Errorf("AlertPolicyRule resource is nil")
	}
	return nil
}

func (c alertPolicyRuleRuntimeClient) create(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
) (servicemanager.OSOKResponse, error) {
	body, err := buildAlertPolicyRuleCreateBody(resource)
	if err != nil {
		return c.fail(resource, err)
	}
	response, err := c.client.CreateAlertPolicyRule(ctx, datasafesdk.CreateAlertPolicyRuleRequest{
		AlertPolicyId:                common.String(identity.alertPolicyID),
		CreateAlertPolicyRuleDetails: body,
		OpcRetryToken:                alertPolicyRuleRetryToken(resource),
	})
	if err != nil {
		return c.fail(resource, err)
	}

	identity.ruleKey = alertPolicyRuleString(response.Key)
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if workRequestID := alertPolicyRuleString(response.OpcWorkRequestId); workRequestID != "" {
		return c.resumeWriteWorkRequest(ctx, resource, identity, response.AlertPolicyRule, shared.OSOKAsyncPhaseCreate, workRequestID)
	}

	projected := c.applyStatus(resource, identity, response.AlertPolicyRule, shared.OSOKAsyncPhaseCreate, "")
	return projected, nil
}

func (c alertPolicyRuleRuntimeClient) reconcileExisting(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	current datasafesdk.AlertPolicyRule,
) (servicemanager.OSOKResponse, error) {
	if workRequestID, phase, ok := currentAlertPolicyRulePendingWorkRequest(resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate); ok {
		return c.resumeWriteWorkRequest(ctx, resource, identity, current, phase, workRequestID)
	}

	if alertPolicyRuleWritePending(current.LifecycleState) {
		return c.applyStatus(resource, identity, current, phaseForAlertPolicyRuleState(current.LifecycleState, shared.OSOKAsyncPhaseCreate), ""), nil
	}

	body, updateNeeded, err := buildAlertPolicyRuleUpdateBody(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.applyStatus(resource, identity, current, shared.OSOKAsyncPhaseCreate, ""), nil
	}

	response, err := c.client.UpdateAlertPolicyRule(ctx, datasafesdk.UpdateAlertPolicyRuleRequest{
		AlertPolicyId:                common.String(identity.alertPolicyID),
		RuleKey:                      common.String(identity.ruleKey),
		UpdateAlertPolicyRuleDetails: body,
	})
	if err != nil {
		return c.fail(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	refreshed, err := c.get(ctx, identity)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, failErr
	}
	if workRequestID := alertPolicyRuleString(response.OpcWorkRequestId); workRequestID != "" {
		return c.resumeWriteWorkRequest(ctx, resource, identity, refreshed, shared.OSOKAsyncPhaseUpdate, workRequestID)
	}

	projected := c.applyStatus(resource, identity, refreshed, shared.OSOKAsyncPhaseUpdate, "")
	return projected, nil
}

func (c alertPolicyRuleRuntimeClient) get(
	ctx context.Context,
	identity alertPolicyRuleIdentity,
) (datasafesdk.AlertPolicyRule, error) {
	response, err := c.client.GetAlertPolicyRule(ctx, datasafesdk.GetAlertPolicyRuleRequest{
		AlertPolicyId: common.String(identity.alertPolicyID),
		RuleKey:       common.String(identity.ruleKey),
	})
	if err != nil {
		return datasafesdk.AlertPolicyRule{}, normalizeAlertPolicyRuleOCIError(err)
	}
	return response.AlertPolicyRule, nil
}

func (c alertPolicyRuleRuntimeClient) lookupExisting(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	alertPolicyID string,
) (datasafesdk.AlertPolicyRule, bool, error) {
	items, err := c.listAll(ctx, alertPolicyID)
	if err != nil {
		return datasafesdk.AlertPolicyRule{}, false, err
	}

	var matches []datasafesdk.AlertPolicyRuleSummary
	for _, item := range items {
		if alertPolicyRuleSummaryMatchesDesired(resource, item) {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 0:
		return datasafesdk.AlertPolicyRule{}, false, nil
	case 1:
		return alertPolicyRuleFromSummary(matches[0]), true, nil
	default:
		return datasafesdk.AlertPolicyRule{}, false, fmt.Errorf("AlertPolicyRule list response returned multiple matching rules for alertPolicyId %q", alertPolicyID)
	}
}

func (c alertPolicyRuleRuntimeClient) listAll(
	ctx context.Context,
	alertPolicyID string,
) ([]datasafesdk.AlertPolicyRuleSummary, error) {
	request := datasafesdk.ListAlertPolicyRulesRequest{
		AlertPolicyId: common.String(alertPolicyID),
	}
	seenPages := map[string]struct{}{}
	var items []datasafesdk.AlertPolicyRuleSummary

	for {
		response, err := c.client.ListAlertPolicyRules(ctx, request)
		if err != nil {
			return nil, normalizeAlertPolicyRuleOCIError(err)
		}
		items = append(items, response.Items...)

		nextPage := alertPolicyRuleString(response.OpcNextPage)
		if nextPage == "" {
			return items, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return nil, fmt.Errorf("AlertPolicyRule list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
	}
}

func (c alertPolicyRuleRuntimeClient) handleDeleteError(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	err error,
) (bool, error) {
	err = normalizeAlertPolicyRuleOCIError(err)
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	switch {
	case isUnambiguousAlertPolicyRuleNotFound(err):
		markAlertPolicyRuleDeleted(resource, c.log, "OCI AlertPolicyRule no longer exists")
		return true, nil
	case isRetryableAlertPolicyRuleDeleteConflict(err):
		current, readErr := c.get(ctx, identity)
		switch {
		case readErr == nil:
			c.applyDeleteReadback(resource, identity, current, "")
			return false, nil
		case isUnambiguousAlertPolicyRuleNotFound(readErr):
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, readErr)
			markAlertPolicyRuleDeleted(resource, c.log, "OCI AlertPolicyRule deleted")
			return true, nil
		default:
			_, failErr := c.fail(resource, readErr)
			return false, failErr
		}
	default:
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
}

func (c alertPolicyRuleRuntimeClient) getWorkRequest(
	ctx context.Context,
	workRequestID string,
) (datasafesdk.WorkRequest, error) {
	response, err := c.client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return datasafesdk.WorkRequest{}, normalizeAlertPolicyRuleOCIError(err)
	}
	return response.WorkRequest, nil
}

func (c alertPolicyRuleRuntimeClient) resumeWriteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	current datasafesdk.AlertPolicyRule,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) (servicemanager.OSOKResponse, error) {
	projectAlertPolicyRuleStatus(resource, identity, current)

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, err)
	}
	currentAsync, err := alertPolicyRuleWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, phase, workRequestID)
	if err != nil {
		return c.fail(resource, err)
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.markAsyncOperation(resource, currentAsync), nil
	case shared.OSOKAsyncClassSucceeded:
		return applyAlertPolicyRuleLifecycle(resource, c.log, current.LifecycleState, phase, ""), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failAsyncOperation(resource, currentAsync, fmt.Errorf("AlertPolicyRule %s work request %s finished with status %s", phase, workRequestID, workRequest.Status))
	default:
		return c.fail(resource, fmt.Errorf("AlertPolicyRule work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass))
	}
}

func (c alertPolicyRuleRuntimeClient) resumeDeleteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
	currentAsync, err := alertPolicyRuleWorkRequestAsyncOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete, workRequestID)
	if err != nil {
		_, failErr := c.fail(resource, err)
		return false, failErr
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.markAsyncOperation(resource, currentAsync)
		return false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, failErr := c.failAsyncOperation(resource, currentAsync, fmt.Errorf("AlertPolicyRule delete work request %s finished with status %s", workRequestID, workRequest.Status))
		return false, failErr
	case shared.OSOKAsyncClassSucceeded:
		c.markAsyncOperation(resource, currentAsync)
		return c.confirmDeleteAfterWorkRequest(ctx, resource, identity)
	default:
		_, failErr := c.fail(resource, fmt.Errorf("AlertPolicyRule delete work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass))
		return false, failErr
	}
}

func (c alertPolicyRuleRuntimeClient) confirmDeleteAfterWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
) (bool, error) {
	if identity.ruleKey == "" {
		return c.deleteByDesiredMatch(ctx, resource, identity.alertPolicyID)
	}

	current, err := c.get(ctx, identity)
	switch {
	case err == nil:
		projectAlertPolicyRuleStatus(resource, identity, current)
		return false, nil
	case isUnambiguousAlertPolicyRuleNotFound(err):
		markAlertPolicyRuleDeleted(resource, c.log, "OCI AlertPolicyRule deleted")
		return true, nil
	default:
		_, failErr := c.fail(resource, err)
		return false, failErr
	}
}

func (c alertPolicyRuleRuntimeClient) markAsyncOperation(
	resource *datasafev1beta1.AlertPolicyRule,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, c.log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: alertPolicyRuleRequeueDuration,
	}
}

func (c alertPolicyRuleRuntimeClient) failAsyncOperation(
	resource *datasafev1beta1.AlertPolicyRule,
	current *shared.OSOKAsyncOperation,
	err error,
) (servicemanager.OSOKResponse, error) {
	if current == nil {
		return c.fail(resource, err)
	}
	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}
	next := *current
	next.NormalizedClass = class
	if err != nil {
		next.Message = err.Error()
	}
	return c.markAsyncOperation(resource, &next), err
}

func (c alertPolicyRuleRuntimeClient) applyStatus(
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	current datasafesdk.AlertPolicyRule,
	fallbackPhase shared.OSOKAsyncPhase,
	workRequestID string,
) servicemanager.OSOKResponse {
	projectAlertPolicyRuleStatus(resource, identity, current)
	return applyAlertPolicyRuleLifecycle(resource, c.log, current.LifecycleState, fallbackPhase, workRequestID)
}

func (c alertPolicyRuleRuntimeClient) applyDeleteReadback(
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	current datasafesdk.AlertPolicyRule,
	workRequestID string,
) servicemanager.OSOKResponse {
	projectAlertPolicyRuleStatus(resource, identity, current)
	if current.LifecycleState == datasafesdk.AlertPolicyRuleLifecycleStateDeleting {
		return applyAlertPolicyRuleLifecycle(resource, c.log, current.LifecycleState, shared.OSOKAsyncPhaseDelete, workRequestID)
	}
	return markAlertPolicyRuleDeletePending(resource, c.log, workRequestID)
}

func (c alertPolicyRuleRuntimeClient) fail(
	resource *datasafev1beta1.AlertPolicyRule,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil && err != nil {
		markAlertPolicyRuleFailed(resource, c.log, err)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func validateAlertPolicyRuleSpec(spec datasafev1beta1.AlertPolicyRuleSpec) error {
	if strings.TrimSpace(spec.Expression) == "" {
		return fmt.Errorf("AlertPolicyRule spec is missing required field(s): expression")
	}
	return nil
}

func resolveAlertPolicyRuleIdentity(resource *datasafev1beta1.AlertPolicyRule) (alertPolicyRuleIdentity, error) {
	if resource == nil {
		return alertPolicyRuleIdentity{}, fmt.Errorf("AlertPolicyRule resource is nil")
	}

	trackedParent, trackedKey := parseAlertPolicyRuleTrackedID(resource.Status.OsokStatus.Ocid)
	statusKey := strings.TrimSpace(resource.Status.Key)
	if trackedKey != "" && statusKey != "" && trackedKey != statusKey {
		return alertPolicyRuleIdentity{}, fmt.Errorf("AlertPolicyRule tracked rule key %q does not match status.key %q", trackedKey, statusKey)
	}

	annotationParent := alertPolicyRuleAnnotation(resource, alertPolicyRuleAlertPolicyIDAnnotation)
	if trackedParent != "" && annotationParent != "" && trackedParent != annotationParent && resource.DeletionTimestamp.IsZero() {
		return alertPolicyRuleIdentity{}, fmt.Errorf("AlertPolicyRule create-only parent annotation %q changed from %q to %q; create a replacement resource instead", alertPolicyRuleAlertPolicyIDAnnotation, trackedParent, annotationParent)
	}

	identity := alertPolicyRuleIdentity{
		alertPolicyID: firstNonEmptyAlertPolicyRuleString(trackedParent, annotationParent),
		ruleKey:       firstNonEmptyAlertPolicyRuleString(trackedKey, statusKey),
	}
	if identity.alertPolicyID == "" {
		return alertPolicyRuleIdentity{}, fmt.Errorf("AlertPolicyRule requires metadata annotation %q with the parent alert policy OCID because spec.alertPolicyId is not available", alertPolicyRuleAlertPolicyIDAnnotation)
	}
	return identity, nil
}

func resolveAlertPolicyRuleDeleteIdentity(resource *datasafev1beta1.AlertPolicyRule) (alertPolicyRuleIdentity, error) {
	if resource == nil {
		return alertPolicyRuleIdentity{}, fmt.Errorf("AlertPolicyRule resource is nil")
	}

	trackedParent, trackedKey := parseAlertPolicyRuleTrackedID(resource.Status.OsokStatus.Ocid)
	identity := alertPolicyRuleIdentity{
		alertPolicyID: firstNonEmptyAlertPolicyRuleString(trackedParent, alertPolicyRuleAnnotation(resource, alertPolicyRuleAlertPolicyIDAnnotation)),
		ruleKey:       firstNonEmptyAlertPolicyRuleString(trackedKey, resource.Status.Key),
	}
	switch {
	case identity.alertPolicyID == "" && identity.ruleKey == "":
		return alertPolicyRuleIdentity{}, nil
	case identity.alertPolicyID == "":
		return alertPolicyRuleIdentity{}, fmt.Errorf("AlertPolicyRule cannot delete recorded rule key %q without parent alert policy identity", identity.ruleKey)
	default:
		return identity, nil
	}
}

func buildAlertPolicyRuleCreateBody(
	resource *datasafev1beta1.AlertPolicyRule,
) (datasafesdk.CreateAlertPolicyRuleDetails, error) {
	if resource == nil {
		return datasafesdk.CreateAlertPolicyRuleDetails{}, fmt.Errorf("AlertPolicyRule resource is nil")
	}
	if err := validateAlertPolicyRuleSpec(resource.Spec); err != nil {
		return datasafesdk.CreateAlertPolicyRuleDetails{}, err
	}

	body := datasafesdk.CreateAlertPolicyRuleDetails{
		Expression: common.String(strings.TrimSpace(resource.Spec.Expression)),
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		body.Description = common.String(description)
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		body.DisplayName = common.String(displayName)
	}
	return body, nil
}

func buildAlertPolicyRuleUpdateBody(
	resource *datasafev1beta1.AlertPolicyRule,
	current datasafesdk.AlertPolicyRule,
) (datasafesdk.UpdateAlertPolicyRuleDetails, bool, error) {
	if resource == nil {
		return datasafesdk.UpdateAlertPolicyRuleDetails{}, false, fmt.Errorf("AlertPolicyRule resource is nil")
	}
	if err := validateAlertPolicyRuleSpec(resource.Spec); err != nil {
		return datasafesdk.UpdateAlertPolicyRuleDetails{}, false, err
	}

	var updateNeeded bool
	body := datasafesdk.UpdateAlertPolicyRuleDetails{}
	if expression := strings.TrimSpace(resource.Spec.Expression); expression != alertPolicyRuleString(current.Expression) {
		body.Expression = common.String(expression)
		updateNeeded = true
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != alertPolicyRuleString(current.Description) {
		body.Description = common.String(description)
		updateNeeded = true
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != alertPolicyRuleString(current.DisplayName) {
		body.DisplayName = common.String(displayName)
		updateNeeded = true
	}
	return body, updateNeeded, nil
}

func alertPolicyRuleSummaryMatchesDesired(
	resource *datasafev1beta1.AlertPolicyRule,
	summary datasafesdk.AlertPolicyRuleSummary,
) bool {
	if resource == nil {
		return false
	}
	if key := strings.TrimSpace(resource.Status.Key); key != "" {
		return key == alertPolicyRuleString(summary.Key)
	}
	if expression := strings.TrimSpace(resource.Spec.Expression); expression == "" || expression != alertPolicyRuleString(summary.Expression) {
		return false
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" && displayName != alertPolicyRuleString(summary.DisplayName) {
		return false
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" && description != alertPolicyRuleString(summary.Description) {
		return false
	}
	return true
}

func alertPolicyRuleFromSummary(summary datasafesdk.AlertPolicyRuleSummary) datasafesdk.AlertPolicyRule {
	return datasafesdk.AlertPolicyRule(summary)
}

func projectAlertPolicyRuleStatus(
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
	current datasafesdk.AlertPolicyRule,
) {
	if resource == nil {
		return
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = datasafev1beta1.AlertPolicyRuleStatus{
		OsokStatus:     osokStatus,
		Key:            alertPolicyRuleString(current.Key),
		Expression:     alertPolicyRuleString(current.Expression),
		Description:    alertPolicyRuleString(current.Description),
		LifecycleState: string(current.LifecycleState),
		DisplayName:    alertPolicyRuleString(current.DisplayName),
		TimeCreated:    alertPolicyRuleTimeString(current.TimeCreated),
	}
	if identity.ruleKey == "" {
		identity.ruleKey = resource.Status.Key
	}
	recordAlertPolicyRuleTrackedIdentity(resource, identity)
}

func recordAlertPolicyRuleTrackedIdentity(
	resource *datasafev1beta1.AlertPolicyRule,
	identity alertPolicyRuleIdentity,
) {
	if resource == nil || strings.TrimSpace(identity.alertPolicyID) == "" || strings.TrimSpace(identity.ruleKey) == "" {
		return
	}
	resource.Status.Key = strings.TrimSpace(identity.ruleKey)
	resource.Status.OsokStatus.Ocid = formatAlertPolicyRuleTrackedID(identity.alertPolicyID, identity.ruleKey)
	if resource.Status.OsokStatus.CreatedAt == nil {
		now := metav1.Now()
		resource.Status.OsokStatus.CreatedAt = &now
	}
}

func clearAlertPolicyRuleTrackedIdentity(resource *datasafev1beta1.AlertPolicyRule) {
	if resource == nil {
		return
	}
	resource.Status.Key = ""
	resource.Status.OsokStatus.Ocid = ""
}

func applyAlertPolicyRuleLifecycle(
	resource *datasafev1beta1.AlertPolicyRule,
	log loggerutil.OSOKLogger,
	state datasafesdk.AlertPolicyRuleLifecycleStateEnum,
	fallbackPhase shared.OSOKAsyncPhase,
	workRequestID string,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	message := alertPolicyRuleLifecycleMessage(resource, state)
	now := metav1.Now()

	switch state {
	case datasafesdk.AlertPolicyRuleLifecycleStateCreating,
		datasafesdk.AlertPolicyRuleLifecycleStateUpdating,
		datasafesdk.AlertPolicyRuleLifecycleStateDeleting,
		datasafesdk.AlertPolicyRuleLifecycleStateFailed:
		phase := phaseForAlertPolicyRuleState(state, fallbackPhase)
		class := shared.OSOKAsyncClassPending
		if state == datasafesdk.AlertPolicyRuleLifecycleStateFailed {
			class = shared.OSOKAsyncClassFailed
		}
		current := &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           phase,
			WorkRequestID:   strings.TrimSpace(workRequestID),
			RawStatus:       string(state),
			NormalizedClass: class,
			Message:         message,
			UpdatedAt:       &now,
		}
		projection := servicemanager.ApplyAsyncOperation(status, current, log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: alertPolicyRuleRequeueDuration,
		}
	default:
		servicemanager.ClearAsyncOperation(status)
		status.Message = message
		status.Reason = string(shared.Active)
		status.UpdatedAt = &now
		*status = util.UpdateOSOKStatusCondition(*status, shared.Active, corev1.ConditionTrue, "", message, log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	}
}

func phaseForAlertPolicyRuleState(
	state datasafesdk.AlertPolicyRuleLifecycleStateEnum,
	fallback shared.OSOKAsyncPhase,
) shared.OSOKAsyncPhase {
	switch state {
	case datasafesdk.AlertPolicyRuleLifecycleStateUpdating:
		return shared.OSOKAsyncPhaseUpdate
	case datasafesdk.AlertPolicyRuleLifecycleStateDeleting:
		return shared.OSOKAsyncPhaseDelete
	case datasafesdk.AlertPolicyRuleLifecycleStateFailed:
		if fallback != "" {
			return fallback
		}
		return shared.OSOKAsyncPhaseCreate
	default:
		return shared.OSOKAsyncPhaseCreate
	}
}

func alertPolicyRuleWritePending(state datasafesdk.AlertPolicyRuleLifecycleStateEnum) bool {
	switch state {
	case datasafesdk.AlertPolicyRuleLifecycleStateCreating,
		datasafesdk.AlertPolicyRuleLifecycleStateUpdating,
		datasafesdk.AlertPolicyRuleLifecycleStateDeleting:
		return true
	default:
		return false
	}
}

func alertPolicyRuleWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest datasafesdk.WorkRequest,
	fallbackPhase shared.OSOKAsyncPhase,
	fallbackWorkRequestID string,
) (*shared.OSOKAsyncOperation, error) {
	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, alertPolicyRuleWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        string(workRequest.OperationType),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    firstNonEmptyAlertPolicyRuleString(alertPolicyRuleString(workRequest.Id), fallbackWorkRequestID),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	current.Message = alertPolicyRuleWorkRequestMessage(current.Phase, workRequest, current.WorkRequestID)
	return current, nil
}

func currentAlertPolicyRulePendingWorkRequest(
	resource *datasafev1beta1.AlertPolicyRule,
	phases ...shared.OSOKAsyncPhase,
) (string, shared.OSOKAsyncPhase, bool) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", "", false
	}
	current := resource.Status.OsokStatus.Async.Current
	workRequestID := strings.TrimSpace(current.WorkRequestID)
	if workRequestID == "" || current.NormalizedClass != shared.OSOKAsyncClassPending {
		return "", "", false
	}
	if len(phases) == 0 {
		return workRequestID, current.Phase, true
	}
	for _, phase := range phases {
		if current.Phase == phase {
			return workRequestID, current.Phase, true
		}
	}
	return "", "", false
}

func markAlertPolicyRuleDeletePending(
	resource *datasafev1beta1.AlertPolicyRule,
	log loggerutil.OSOKLogger,
	workRequestID string,
) servicemanager.OSOKResponse {
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		RawStatus:       string(datasafesdk.AlertPolicyRuleLifecycleStateDeleting),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         "OCI AlertPolicyRule delete is in progress",
		UpdatedAt:       &now,
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   true,
		RequeueDuration: alertPolicyRuleRequeueDuration,
	}
}

func markAlertPolicyRuleDeleted(
	resource *datasafev1beta1.AlertPolicyRule,
	log loggerutil.OSOKLogger,
	message string,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = strings.TrimSpace(message)
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", status.Message, log)
}

func markAlertPolicyRuleFailed(
	resource *datasafev1beta1.AlertPolicyRule,
	log loggerutil.OSOKLogger,
	err error,
) {
	if resource == nil || err == nil {
		return
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		servicemanager.ApplyAsyncOperation(status, &current, log)
		return
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), log)
}

func alertPolicyRuleLifecycleMessage(
	resource *datasafev1beta1.AlertPolicyRule,
	state datasafesdk.AlertPolicyRuleLifecycleStateEnum,
) string {
	displayName := ""
	if resource != nil {
		displayName = strings.TrimSpace(resource.Status.DisplayName)
	}
	if displayName != "" {
		return displayName
	}
	switch state {
	case datasafesdk.AlertPolicyRuleLifecycleStateCreating:
		return "OCI AlertPolicyRule create is in progress"
	case datasafesdk.AlertPolicyRuleLifecycleStateUpdating:
		return "OCI AlertPolicyRule update is in progress"
	case datasafesdk.AlertPolicyRuleLifecycleStateDeleting:
		return "OCI AlertPolicyRule delete is in progress"
	case datasafesdk.AlertPolicyRuleLifecycleStateFailed:
		return "OCI AlertPolicyRule is failed"
	default:
		return "OCI AlertPolicyRule is active"
	}
}

func alertPolicyRuleWorkRequestMessage(
	phase shared.OSOKAsyncPhase,
	workRequest datasafesdk.WorkRequest,
	workRequestID string,
) string {
	phaseLabel := strings.TrimSpace(string(phase))
	if phaseLabel == "" {
		phaseLabel = "operation"
	}
	status := strings.TrimSpace(string(workRequest.Status))
	if status == "" {
		status = "unknown"
	}
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return fmt.Sprintf("OCI AlertPolicyRule %s work request is %s", phaseLabel, status)
	}
	return fmt.Sprintf("OCI AlertPolicyRule %s work request %s is %s", phaseLabel, workRequestID, status)
}

func alertPolicyRuleRetryToken(resource *datasafev1beta1.AlertPolicyRule) *string {
	if resource == nil {
		return nil
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return common.String(uid)
	}

	namespace := strings.TrimSpace(resource.Namespace)
	name := strings.TrimSpace(resource.Name)
	if namespace == "" && name == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return common.String(hex.EncodeToString(sum[:]))
}

func formatAlertPolicyRuleTrackedID(alertPolicyID string, ruleKey string) shared.OCID {
	return shared.OCID(url.PathEscape(strings.TrimSpace(alertPolicyID)) + alertPolicyRuleTrackedIDSeparator + url.PathEscape(strings.TrimSpace(ruleKey)))
}

func parseAlertPolicyRuleTrackedID(raw shared.OCID) (string, string) {
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", ""
	}
	parent, key, ok := strings.Cut(value, alertPolicyRuleTrackedIDSeparator)
	if !ok {
		return "", value
	}
	parent, parentErr := url.PathUnescape(parent)
	key, keyErr := url.PathUnescape(key)
	if parentErr != nil || keyErr != nil {
		return "", ""
	}
	return strings.TrimSpace(parent), strings.TrimSpace(key)
}

func alertPolicyRuleAnnotation(resource *datasafev1beta1.AlertPolicyRule, key string) string {
	if resource == nil || len(resource.Annotations) == 0 {
		return ""
	}
	return strings.TrimSpace(resource.Annotations[key])
}

func normalizeAlertPolicyRuleOCIError(err error) error {
	if err == nil {
		return nil
	}
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isUnambiguousAlertPolicyRuleNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func isRetryableAlertPolicyRuleDeleteConflict(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsConflict()
}

func alertPolicyRuleString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func alertPolicyRuleTimeString(value *common.SDKTime) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func firstNonEmptyAlertPolicyRuleString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
