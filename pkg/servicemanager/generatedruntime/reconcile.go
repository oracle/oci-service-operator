/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"fmt"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (c ServiceClient[T]) CreateOrUpdate(ctx context.Context, resource T, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if response, err, handled := c.validateCreateOrUpdateRequest(resource); handled {
		return response, err
	}

	identity, err := c.prepareIdentity(resource)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if workRequestID, phase := c.currentGeneratedWorkRequest(resource); workRequestID != "" {
		switch phase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			return c.resumeGeneratedWorkRequestCreateOrUpdate(ctx, resource, identity, workRequestID, phase)
		case shared.OSOKAsyncPhaseDelete:
			return c.failCreateOrUpdate(resource, fmt.Errorf("%s delete work request %s is still active during CreateOrUpdate", c.config.Kind, workRequestID))
		}
	}

	namespace := resourceNamespace(resource, req.Namespace)
	state, err := c.prepareCreateOrUpdateState(ctx, resource, identity)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if err := c.validateMutationPolicy(resource, state.currentID != "", state.liveResponse); err != nil {
		if state.restoreSyntheticTrackedID != nil {
			state.restoreSyntheticTrackedID()
		}
		return c.failCreateOrUpdate(resource, err)
	}

	var response servicemanager.OSOKResponse
	if response, err, handled := c.applyExistingResourceHooks(ctx, resource, state, namespace); handled {
		if err != nil {
			if state.restoreSyntheticTrackedID != nil {
				state.restoreSyntheticTrackedID()
			}
			return response, err
		}
		if state.restoreSyntheticTrackedID != nil && c.config.Identity.RecordTracked == nil {
			state.restoreSyntheticTrackedID()
		}
		return response, nil
	}

	statusBaseline, statusCleared := c.clearProjectedStatus(resource)
	if state.currentID != "" {
		response, err = c.reconcileExistingResource(ctx, resource, state, namespace)
	} else {
		response, err = c.createOrReadResource(ctx, resource, namespace, state.identity)
	}
	if err != nil {
		if statusCleared {
			c.restoreStatusAfterFailure(resource, statusBaseline)
		}
		if state.restoreSyntheticTrackedID != nil {
			state.restoreSyntheticTrackedID()
		}
		return response, err
	}
	if state.restoreSyntheticTrackedID != nil && c.config.Identity.RecordTracked == nil {
		state.restoreSyntheticTrackedID()
	}
	return response, nil
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
		return c.updateExistingResource(ctx, resource, state.currentID, namespace, state.liveResponse, state.identity)
	}
	return c.observeExistingResource(ctx, resource, state.currentID, state.liveResponse, state.identity)
}

func (c ServiceClient[T]) updateExistingResource(ctx context.Context, resource T, currentID string, namespace string, currentResponse any, identity any) (servicemanager.OSOKResponse, error) {
	options := c.requestBuildOptions(ctx, namespace)
	options.CurrentResponse = currentResponse

	response, err := c.invoke(ctx, c.config.Update, resource, currentID, options)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	requeueDuration := responseRetryAfterDuration(response)
	c.seedOpeningRequestID(resource, response)
	if c.generatedWorkRequestPhaseEnabled(shared.OSOKAsyncPhaseUpdate) {
		workRequestID, err := c.startGeneratedWorkRequest(resource, response, shared.OSOKAsyncPhaseUpdate, identity)
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		return c.resumeGeneratedWorkRequestCreateOrUpdate(ctx, resource, identity, workRequestID, shared.OSOKAsyncPhaseUpdate)
	}
	c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseUpdate)

	response, err = c.followUpAfterWrite(ctx, resource, currentID, response, "update")
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccessWithIdentityAndRequeue(resource, response, shared.Updating, identity, requeueDuration)
}

func (c ServiceClient[T]) observeExistingResource(ctx context.Context, resource T, currentID string, liveResponse any, identity any) (servicemanager.OSOKResponse, error) {
	response := liveResponse
	if response == nil && c.hasReadableOperation() {
		var err error
		response, err = c.readResource(ctx, resource, currentID, readPhaseObserve)
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
	}
	return c.applySuccessWithIdentity(resource, response, shared.Active, identity)
}

func (c ServiceClient[T]) createOrReadResource(ctx context.Context, resource T, namespace string, identity any) (servicemanager.OSOKResponse, error) {
	if c.config.Create != nil {
		response, err := c.invoke(ctx, c.config.Create, resource, "", c.requestBuildOptions(ctx, namespace))
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		requeueDuration := responseRetryAfterDuration(response)
		c.seedOpeningRequestID(resource, response)
		if c.generatedWorkRequestPhaseEnabled(shared.OSOKAsyncPhaseCreate) {
			workRequestID, err := c.startGeneratedWorkRequest(resource, response, shared.OSOKAsyncPhaseCreate, identity)
			if err != nil {
				return c.failCreateOrUpdate(resource, err)
			}
			return c.resumeGeneratedWorkRequestCreateOrUpdate(ctx, resource, identity, workRequestID, shared.OSOKAsyncPhaseCreate)
		}
		c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseCreate)

		followUp, err := c.followUpAfterWrite(ctx, resource, responseID(response), response, "create")
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		return c.applySuccessWithIdentityAndRequeue(resource, followUp, shared.Provisioning, identity, requeueDuration)
	}

	response, err := c.readResource(ctx, resource, "", readPhaseObserve)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccessWithIdentity(resource, response, shared.Active, identity)
}

func (c ServiceClient[T]) failCreateOrUpdate(resource T, err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
}
