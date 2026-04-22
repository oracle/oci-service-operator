/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package query

import (
	"context"
	"strings"

	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type synchronousQueryServiceClient struct {
	delegate QueryServiceClient
	log      loggerutil.OSOKLogger
}

func init() {
	registerQueryRuntimeHooksMutator(func(manager *QueryServiceManager, hooks *QueryRuntimeHooks) {
		appendSynchronousQueryRuntimeWrapper(manager, hooks)
	})
}

func appendSynchronousQueryRuntimeWrapper(manager *QueryServiceManager, hooks *QueryRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate QueryServiceClient) QueryServiceClient {
		return &synchronousQueryServiceClient{
			delegate: delegate,
			log:      manager.Log,
		}
	})
}

func (c *synchronousQueryServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *usageapiv1beta1.Query,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !response.ShouldRequeue || resource == nil {
		return response, err
	}

	status := &resource.Status.OsokStatus
	if status.Async.Current != nil {
		return response, err
	}
	if status.Reason != string(shared.Provisioning) && status.Reason != string(shared.Updating) {
		return response, err
	}

	now := metav1.Now()
	servicemanager.ClearAsyncOperation(status)
	status.Reason = string(shared.Active)
	status.UpdatedAt = &now
	if strings.TrimSpace(status.Message) == "" {
		status.Message = resource.Status.QueryDefinition.DisplayName
		if strings.TrimSpace(status.Message) == "" {
			status.Message = resource.Spec.QueryDefinition.DisplayName
		}
	}
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Active,
		corev1.ConditionTrue,
		"",
		status.Message,
		c.log,
	)

	response.ShouldRequeue = false
	response.RequeueDuration = 0
	return response, nil
}

func (c *synchronousQueryServiceClient) Delete(ctx context.Context, resource *usageapiv1beta1.Query) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
