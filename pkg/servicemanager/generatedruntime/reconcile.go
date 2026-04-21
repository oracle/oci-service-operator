/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (c ServiceClient[T]) CreateOrUpdate(ctx context.Context, resource T, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if response, err, handled := c.validateCreateOrUpdateRequest(resource); handled {
		return response, err
	}

	namespace := resourceNamespace(resource, req.Namespace)
	state, err := c.prepareCreateOrUpdateState(ctx, resource)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if err := c.validateMutationPolicy(resource, state.currentID != "", state.liveResponse); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if state.currentID != "" {
		return c.reconcileExistingResource(ctx, resource, state, namespace)
	}
	return c.createOrReadResource(ctx, resource, namespace)
}

func (c ServiceClient[T]) validateCreateOrUpdateRequest(resource T) (servicemanager.OSOKResponse, error, bool) {
	if c.config.InitError != nil {
		response, err := c.failCreateOrUpdate(resource, c.config.InitError)
		return response, err, true
	}
	if _, err := resourceStruct(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err, true
	}
	return servicemanager.OSOKResponse{}, nil, false
}

func (c ServiceClient[T]) reconcileExistingResource(ctx context.Context, resource T, state createOrUpdateState, namespace string) (servicemanager.OSOKResponse, error) {
	shouldUpdate, err := c.shouldInvokeUpdate(ctx, resource, namespace, state.liveResponse)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if shouldUpdate {
		return c.updateExistingResource(ctx, resource, state.currentID, namespace, state.liveResponse)
	}
	return c.observeExistingResource(ctx, resource, state.currentID, state.liveResponse)
}

func (c ServiceClient[T]) updateExistingResource(ctx context.Context, resource T, currentID string, namespace string, currentResponse any) (servicemanager.OSOKResponse, error) {
	options := c.requestBuildOptions(ctx, namespace)
	options.CurrentResponse = currentResponse

	response, err := c.invoke(ctx, c.config.Update, resource, currentID, options)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	c.seedOpeningRequestID(resource, response)
	c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseUpdate)

	response, err = c.followUpAfterWrite(ctx, resource, currentID, response, "update")
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccess(resource, response, shared.Updating)
}

func (c ServiceClient[T]) observeExistingResource(ctx context.Context, resource T, currentID string, liveResponse any) (servicemanager.OSOKResponse, error) {
	response := liveResponse
	if response == nil && (c.config.Get != nil || c.config.List != nil) {
		var err error
		response, err = c.readResource(ctx, resource, currentID, readPhaseObserve)
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
	}
	return c.applySuccess(resource, response, shared.Active)
}

func (c ServiceClient[T]) createOrReadResource(ctx context.Context, resource T, namespace string) (servicemanager.OSOKResponse, error) {
	if c.config.Create != nil {
		response, err := c.invoke(ctx, c.config.Create, resource, "", c.requestBuildOptions(ctx, namespace))
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		c.seedOpeningRequestID(resource, response)
		c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseCreate)

		followUp, err := c.followUpAfterWrite(ctx, resource, responseID(response), response, "create")
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		return c.applySuccess(resource, followUp, shared.Provisioning)
	}

	response, err := c.readResource(ctx, resource, "", readPhaseObserve)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccess(resource, response, shared.Active)
}

func (c ServiceClient[T]) failCreateOrUpdate(resource T, err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
}
