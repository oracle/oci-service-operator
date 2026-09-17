/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package copyobjectrequest

import (
	"context"
	"strings"

	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	registerCopyObjectRequestRuntimeHooksMutator(func(_ *CopyObjectRequestServiceManager, hooks *CopyObjectRequestRuntimeHooks) {
		hooks.StatusHooks.ApplyLifecycle = applyCopyObjectRequestLifecycle
		hooks.BuildUpdateBody = buildCopyObjectRequestUpdateBody
	})
}

func buildCopyObjectRequestUpdateBody(_ context.Context, resource *dataintegrationv1beta1.CopyObjectRequest, _ string, currentResponse any) (any, bool, error) {
	desired := strings.TrimSpace(resource.Spec.Status)
	if desired == "" {
		return nil, false, nil
	}
	current := ""
	switch response := currentResponse.(type) {
	case dataintegrationsdk.GetCopyObjectRequestResponse:
		current = string(response.CopyObjectRequest.CopyMetadataObjectRequestStatus)
	case *dataintegrationsdk.GetCopyObjectRequestResponse:
		if response != nil {
			current = string(response.CopyObjectRequest.CopyMetadataObjectRequestStatus)
		}
	}
	if strings.EqualFold(desired, current) || strings.EqualFold(desired, "TERMINATING") && strings.EqualFold(current, "TERMINATED") {
		return nil, false, nil
	}
	return dataintegrationsdk.UpdateCopyObjectRequestDetails{Status: dataintegrationsdk.UpdateCopyObjectRequestDetailsStatusEnum(desired)}, true, nil
}

func applyCopyObjectRequestLifecycle(resource *dataintegrationv1beta1.CopyObjectRequest, _ any) (servicemanager.OSOKResponse, error) {
	state := resource.Status.CopyMetadataObjectRequestStatus
	condition := shared.Active
	conditionStatus := corev1.ConditionTrue
	requeue := false
	successful := true
	switch state {
	case "IN_PROGRESS", "QUEUED":
		condition, requeue = shared.Provisioning, true
	case "TERMINATING":
		condition, requeue = shared.Updating, true
	case "FAILED":
		condition, conditionStatus, successful = shared.Failed, corev1.ConditionFalse, false
	case "SUCCESSFUL", "TERMINATED", "":
	default:
		condition, conditionStatus, successful = shared.Failed, corev1.ConditionFalse, false
	}
	now := metav1.Now()
	resource.Status.OsokStatus.Reason = string(condition)
	resource.Status.OsokStatus.Message = "OCI copy object request is " + state
	resource.Status.OsokStatus.UpdatedAt = &now
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, condition, conditionStatus, "", resource.Status.OsokStatus.Message, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{IsSuccessful: successful, ShouldRequeue: requeue}, nil
}
