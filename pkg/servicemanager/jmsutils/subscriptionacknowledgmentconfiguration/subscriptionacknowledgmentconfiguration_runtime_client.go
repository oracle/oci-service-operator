/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscriptionacknowledgmentconfiguration

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	jmsutilssdk "github.com/oracle/oci-go-sdk/v65/jmsutils"
)

func init() {
	registerSubscriptionAcknowledgmentConfigurationRuntimeHooksMutator(func(
		manager *SubscriptionAcknowledgmentConfigurationServiceManager,
		hooks *SubscriptionAcknowledgmentConfigurationRuntimeHooks,
	) {
		if manager == nil || manager.Provider == nil {
			return
		}
		injectSubscriptionAcknowledgmentConfigurationTenancy(hooks, manager.Provider.TenancyOCID)
	})
}

func injectSubscriptionAcknowledgmentConfigurationTenancy(
	hooks *SubscriptionAcknowledgmentConfigurationRuntimeHooks,
	resolveTenancy func() (string, error),
) {
	if hooks == nil || resolveTenancy == nil {
		return
	}

	get := hooks.Get.Call
	hooks.Get.Call = func(ctx context.Context, request jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationRequest) (jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse, error) {
		tenancyID, err := requiredSubscriptionAcknowledgmentConfigurationTenancy(resolveTenancy)
		if err != nil {
			return jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse{}, err
		}
		request.CompartmentId = common.String(tenancyID)
		return get(ctx, request)
	}

	update := hooks.Update.Call
	hooks.Update.Call = func(ctx context.Context, request jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationRequest) (jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse, error) {
		tenancyID, err := requiredSubscriptionAcknowledgmentConfigurationTenancy(resolveTenancy)
		if err != nil {
			return jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse{}, err
		}
		request.CompartmentId = common.String(tenancyID)
		return update(ctx, request)
	}
}

func requiredSubscriptionAcknowledgmentConfigurationTenancy(resolve func() (string, error)) (string, error) {
	tenancyID, err := resolve()
	if err != nil {
		return "", fmt.Errorf("resolve SubscriptionAcknowledgmentConfiguration tenancy: %w", err)
	}
	if strings.TrimSpace(tenancyID) == "" {
		return "", fmt.Errorf("resolve SubscriptionAcknowledgmentConfiguration tenancy: empty tenancy OCID")
	}
	return tenancyID, nil
}
