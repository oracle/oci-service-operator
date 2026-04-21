/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package model

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

type modelDeletedLifecycleReadGuardContextKey struct{}

func init() {
	registerModelRuntimeHooksMutator(func(_ *ModelServiceManager, hooks *ModelRuntimeHooks) {
		applyModelRuntimeHooks(hooks)
	})
}

func applyModelRuntimeHooks(hooks *ModelRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Identity.GuardExistingBeforeCreate = guardModelExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearModelIdentity
	hooks.Read.Get = modelDeletedLifecycleReadOperation(hooks.Get)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ModelServiceClient) ModelServiceClient {
		return modelDeletedLifecycleGuardClient{delegate: delegate}
	})
}

func guardModelExistingBeforeCreate(
	_ context.Context,
	resource *generativeaiv1beta1.Model,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func modelDeletedLifecycleReadOperation(
	get runtimeOperationHooks[generativeaisdk.GetModelRequest, generativeaisdk.GetModelResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &generativeaisdk.GetModelRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), get.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := get.Call(ctx, *request.(*generativeaisdk.GetModelRequest))
			if err != nil {
				return nil, err
			}
			if modelDeletedLifecycleReadGuardEnabled(ctx) && response.Model.LifecycleState == generativeaisdk.ModelLifecycleStateDeleted {
				return nil, errorutil.NotFoundOciError{
					HTTPStatusCode: 404,
					ErrorCode:      errorutil.NotFound,
					Description:    "Model lifecycle state DELETED is treated as not found during stale tracked identity recovery",
				}
			}
			return response, nil
		},
	}
}

func withModelDeletedLifecycleReadGuard(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, modelDeletedLifecycleReadGuardContextKey{}, true)
}

func modelDeletedLifecycleReadGuardEnabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(modelDeletedLifecycleReadGuardContextKey{}).(bool)
	return enabled
}

type modelDeletedLifecycleGuardClient struct {
	delegate ModelServiceClient
}

func (c modelDeletedLifecycleGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaiv1beta1.Model,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Model generated delegate is not configured")
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Model resource must not be nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}
	return c.delegate.CreateOrUpdate(withModelDeletedLifecycleReadGuard(ctx), resource, req)
}

func (c modelDeletedLifecycleGuardClient) Delete(
	ctx context.Context,
	resource *generativeaiv1beta1.Model,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("Model generated delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func clearModelIdentity(resource *generativeaiv1beta1.Model) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}
