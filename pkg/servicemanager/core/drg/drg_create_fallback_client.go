/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drg

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	generatedFactory := newDrgServiceClient
	newDrgServiceClient = func(manager *DrgServiceManager) DrgServiceClient {
		return newDrgCreateFallbackClient(manager, generatedFactory(manager))
	}
}

type drgCreateFallbackClient struct {
	delegate DrgServiceClient
	log      loggerutil.OSOKLogger
}

var _ DrgServiceClient = drgCreateFallbackClient{}

func newDrgCreateFallbackClient(manager *DrgServiceManager, delegate DrgServiceClient) DrgServiceClient {
	return drgCreateFallbackClient{
		delegate: delegate,
		log:      manager.Log,
	}
}

func (c drgCreateFallbackClient) CreateOrUpdate(
	ctx context.Context,
	resource *corev1beta1.Drg,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("drg create fallback delegate is not configured")
	}

	hadTrackedID := drgCurrentID(resource) != ""
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil || hadTrackedID {
		return response, err
	}

	recoveredResponse, recovered := recoverDrgCreateObservation(resource, c.log)
	if !recovered {
		return response, err
	}

	c.log.InfoLog("Treating DRG create as successful after post-create read failure",
		"name", req.Name,
		"namespace", req.Namespace,
		"ocid", string(resource.Status.OsokStatus.Ocid),
		"lifecycleState", resource.Status.LifecycleState)
	return recoveredResponse, nil
}

func (c drgCreateFallbackClient) Delete(ctx context.Context, resource *corev1beta1.Drg) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("drg create fallback delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func recoverDrgCreateObservation(resource *corev1beta1.Drg, log loggerutil.OSOKLogger) (servicemanager.OSOKResponse, bool) {
	if resource == nil {
		return servicemanager.OSOKResponse{}, false
	}

	resourceID := drgCurrentID(resource)
	lifecycleState := strings.ToUpper(strings.TrimSpace(resource.Status.LifecycleState))
	if resourceID == "" || lifecycleState == "" {
		return servicemanager.OSOKResponse{}, false
	}

	condition, shouldRequeue, ok := drgRecoveryCondition(lifecycleState)
	if !ok {
		return servicemanager.OSOKResponse{}, false
	}

	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.Ocid = shared.OCID(resourceID)
	if status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now

	message := fmt.Sprintf("Drg %s is %s", drgDisplayName(resource), resource.Status.LifecycleState)
	status.Message = message
	status.Reason = string(condition)

	conditionStatus := corev1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = corev1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: time.Minute,
	}, true
}

func drgRecoveryCondition(lifecycleState string) (shared.OSOKConditionType, bool, bool) {
	switch {
	case strings.Contains(lifecycleState, "AVAIL"),
		strings.Contains(lifecycleState, "ACTIVE"):
		return shared.Active, false, true
	case strings.Contains(lifecycleState, "PROVISION"),
		strings.Contains(lifecycleState, "CREATE"),
		strings.Contains(lifecycleState, "PENDING"),
		strings.Contains(lifecycleState, "IN_PROGRESS"),
		strings.Contains(lifecycleState, "ACCEPT"),
		strings.Contains(lifecycleState, "START"):
		return shared.Provisioning, true, true
	default:
		return "", false, false
	}
}

func drgDisplayName(resource *corev1beta1.Drg) string {
	if resource == nil {
		return ""
	}
	if displayName := strings.TrimSpace(resource.Status.DisplayName); displayName != "" {
		return displayName
	}
	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		return displayName
	}
	return resource.Name
}

func drgCurrentID(resource *corev1beta1.Drg) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}
