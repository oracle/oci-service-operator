/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscriptionacknowledgmentconfiguration

import (
	"context"
	"errors"
	"strings"
	"testing"

	jmsutilssdk "github.com/oracle/oci-go-sdk/v65/jmsutils"
)

func TestInjectSubscriptionAcknowledgmentConfigurationTenancy(t *testing.T) {
	hooks := SubscriptionAcknowledgmentConfigurationRuntimeHooks{
		Get: runtimeOperationHooks[jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationRequest, jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse]{
			Call: func(_ context.Context, request jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationRequest) (jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse, error) {
				if request.CompartmentId == nil || *request.CompartmentId != "ocid1.tenancy.oc1..test" {
					t.Fatalf("get compartmentId = %v", request.CompartmentId)
				}
				return jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse{}, nil
			},
		},
		Update: runtimeOperationHooks[jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationRequest, jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse]{
			Call: func(_ context.Context, request jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationRequest) (jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse, error) {
				if request.CompartmentId == nil || *request.CompartmentId != "ocid1.tenancy.oc1..test" {
					t.Fatalf("update compartmentId = %v", request.CompartmentId)
				}
				return jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse{}, nil
			},
		},
	}
	injectSubscriptionAcknowledgmentConfigurationTenancy(&hooks, func() (string, error) {
		return "ocid1.tenancy.oc1..test", nil
	})
	if _, err := hooks.Get.Call(context.Background(), jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := hooks.Update.Call(context.Background(), jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestInjectSubscriptionAcknowledgmentConfigurationTenancyReportsResolutionFailure(t *testing.T) {
	hooks := SubscriptionAcknowledgmentConfigurationRuntimeHooks{
		Get: runtimeOperationHooks[jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationRequest, jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse]{
			Call: func(context.Context, jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationRequest) (jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse, error) {
				t.Fatal("OCI get must not run when tenancy resolution fails")
				return jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationResponse{}, nil
			},
		},
		Update: runtimeOperationHooks[jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationRequest, jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse]{
			Call: func(context.Context, jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationRequest) (jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse, error) {
				return jmsutilssdk.UpdateSubscriptionAcknowledgmentConfigurationResponse{}, nil
			},
		},
	}
	injectSubscriptionAcknowledgmentConfigurationTenancy(&hooks, func() (string, error) {
		return "", errors.New("provider unavailable")
	})
	_, err := hooks.Get.Call(context.Background(), jmsutilssdk.GetSubscriptionAcknowledgmentConfigurationRequest{})
	if err == nil || !strings.Contains(err.Error(), "resolve SubscriptionAcknowledgmentConfiguration tenancy") {
		t.Fatalf("get error = %v", err)
	}
}
