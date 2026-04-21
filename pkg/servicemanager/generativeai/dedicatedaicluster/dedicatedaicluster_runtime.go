/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dedicatedaicluster

import (
	"context"
	"fmt"
	"strings"

	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type dedicatedAiClusterDeletedLifecycleReadGuardContextKey struct{}

func init() {
	registerDedicatedAiClusterRuntimeHooksMutator(func(_ *DedicatedAiClusterServiceManager, hooks *DedicatedAiClusterRuntimeHooks) {
		applyDedicatedAiClusterRuntimeHooks(hooks)
	})
}

func applyDedicatedAiClusterRuntimeHooks(hooks *DedicatedAiClusterRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Identity.GuardExistingBeforeCreate = guardDedicatedAiClusterExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearDedicatedAiClusterIdentity
	hooks.Read.Get = dedicatedAiClusterDeletedLifecycleReadOperation(hooks.Get)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DedicatedAiClusterServiceClient) DedicatedAiClusterServiceClient {
		return dedicatedAiClusterDeletedLifecycleGuardClient{delegate: delegate}
	})
}

func guardDedicatedAiClusterExistingBeforeCreate(
	_ context.Context,
	resource *generativeaiv1beta1.DedicatedAiCluster,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func dedicatedAiClusterDeletedLifecycleReadOperation(
	get runtimeOperationHooks[generativeaisdk.GetDedicatedAiClusterRequest, generativeaisdk.GetDedicatedAiClusterResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &generativeaisdk.GetDedicatedAiClusterRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), get.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := get.Call(ctx, *request.(*generativeaisdk.GetDedicatedAiClusterRequest))
			if err != nil {
				return nil, err
			}
			if dedicatedAiClusterDeletedLifecycleReadGuardEnabled(ctx) &&
				response.DedicatedAiCluster.LifecycleState == generativeaisdk.DedicatedAiClusterLifecycleStateDeleted {
				return nil, errorutil.NotFoundOciError{
					HTTPStatusCode: 404,
					ErrorCode:      errorutil.NotFound,
					Description:    "DedicatedAiCluster lifecycle state DELETED is treated as not found during stale tracked identity recovery",
				}
			}
			return response, nil
		},
	}
}

func withDedicatedAiClusterDeletedLifecycleReadGuard(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, dedicatedAiClusterDeletedLifecycleReadGuardContextKey{}, true)
}

func dedicatedAiClusterDeletedLifecycleReadGuardEnabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(dedicatedAiClusterDeletedLifecycleReadGuardContextKey{}).(bool)
	return enabled
}

type dedicatedAiClusterDeletedLifecycleGuardClient struct {
	delegate DedicatedAiClusterServiceClient
}

func (c dedicatedAiClusterDeletedLifecycleGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaiv1beta1.DedicatedAiCluster,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DedicatedAiCluster generated delegate is not configured")
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DedicatedAiCluster resource must not be nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}
	return c.delegate.CreateOrUpdate(withDedicatedAiClusterDeletedLifecycleReadGuard(ctx), resource, req)
}

func (c dedicatedAiClusterDeletedLifecycleGuardClient) Delete(
	ctx context.Context,
	resource *generativeaiv1beta1.DedicatedAiCluster,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("DedicatedAiCluster generated delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func clearDedicatedAiClusterIdentity(resource *generativeaiv1beta1.DedicatedAiCluster) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}
