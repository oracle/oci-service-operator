/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c ServiceClient[T]) getReadOperation() *Operation {
	if c.config.Read.Get != nil {
		return c.config.Read.Get
	}
	return c.config.Get
}

func (c ServiceClient[T]) listReadOperation() *Operation {
	if c.config.Read.List != nil {
		return c.config.Read.List
	}
	return c.config.List
}

func (c ServiceClient[T]) hasReadableOperation() bool {
	return c.getReadOperation() != nil || c.listReadOperation() != nil
}

func (c ServiceClient[T]) hasDeleteConfirmRead() bool {
	return c.config.DeleteHooks.ConfirmRead != nil || c.hasReadableOperation()
}

func (c ServiceClient[T]) prepareIdentity(resource T) (any, error) {
	if c.config.Identity.Resolve == nil {
		return nil, nil
	}

	identity, err := c.config.Identity.Resolve(resource)
	if err != nil {
		return nil, err
	}
	if c.config.Identity.RecordPath != nil {
		c.config.Identity.RecordPath(resource, identity)
	}
	return identity, nil
}

func (c ServiceClient[T]) lookupExistingByIdentity(ctx context.Context, resource T, identity any) (any, error) {
	if c.config.Identity.LookupExisting == nil || identity == nil {
		return nil, nil
	}

	response, err := c.config.Identity.LookupExisting(ctx, resource, identity)
	switch {
	case err == nil:
		return response, nil
	case errors.Is(err, errResourceNotFound), isReadNotFound(err):
		return nil, nil
	default:
		return nil, err
	}
}

func (c ServiceClient[T]) guardExistingBeforeCreate(ctx context.Context, resource T) (ExistingBeforeCreateDecision, error) {
	if c.config.Identity.GuardExistingBeforeCreate == nil {
		return ExistingBeforeCreateDecisionAllow, nil
	}

	decision, err := c.config.Identity.GuardExistingBeforeCreate(ctx, resource)
	if err != nil {
		return ExistingBeforeCreateDecisionFail, err
	}
	if decision == "" {
		decision = ExistingBeforeCreateDecisionAllow
	}

	switch decision {
	case ExistingBeforeCreateDecisionAllow, ExistingBeforeCreateDecisionSkip:
		return decision, nil
	case ExistingBeforeCreateDecisionFail:
		return decision, fmt.Errorf("%s identity guard rejected pre-create reuse", c.config.Kind)
	default:
		return "", fmt.Errorf("%s identity guard returned unsupported pre-create decision %q", c.config.Kind, decision)
	}
}

func (c ServiceClient[T]) recordTrackedIdentity(resource T, identity any, resourceID string) {
	if c.config.Identity.RecordTracked == nil {
		return
	}
	c.config.Identity.RecordTracked(resource, identity, resourceID)
}

func (c ServiceClient[T]) clearTrackedIdentity(resource T) {
	if c.config.TrackedRecreate.ClearTrackedIdentity == nil {
		return
	}
	c.config.TrackedRecreate.ClearTrackedIdentity(resource)
}

func (c ServiceClient[T]) normalizeDesiredState(resource T, currentResponse any) {
	if c.config.ParityHooks.NormalizeDesiredState == nil {
		return
	}
	c.config.ParityHooks.NormalizeDesiredState(resource, currentResponse)
}

func (c ServiceClient[T]) clearProjectedStatus(resource T) (any, bool) {
	if c.config.StatusHooks.ClearProjectedStatus == nil {
		return nil, false
	}
	return c.config.StatusHooks.ClearProjectedStatus(resource), true
}

func (c ServiceClient[T]) restoreStatusAfterFailure(resource T, baseline any) {
	if c.config.StatusHooks.RestoreStatus == nil {
		return
	}
	if c.config.StatusHooks.ShouldRestoreOnFailure != nil && !c.config.StatusHooks.ShouldRestoreOnFailure(resource, baseline) {
		return
	}
	c.config.StatusHooks.RestoreStatus(resource, baseline)
}

func (c ServiceClient[T]) applyExistingResourceHooks(
	ctx context.Context,
	resource T,
	state createOrUpdateState,
	namespace string,
) (servicemanager.OSOKResponse, error, bool) {
	if state.currentID == "" || state.liveResponse == nil {
		return servicemanager.OSOKResponse{}, nil, false
	}
	if c.shouldObserveCurrentLifecycle(state.liveResponse) {
		if response, handled, err := c.applyStatusHooksObservation(resource, state.liveResponse, state.identity); handled {
			return response, err, true
		}
	}
	return c.handleParityHooks(ctx, resource, state, namespace)
}

func (c ServiceClient[T]) handleParityHooks(
	ctx context.Context,
	resource T,
	state createOrUpdateState,
	namespace string,
) (servicemanager.OSOKResponse, error, bool) {
	if c.config.ParityHooks.RequiresParityHandling == nil || !c.config.ParityHooks.RequiresParityHandling(resource, state.liveResponse) {
		return servicemanager.OSOKResponse{}, nil, false
	}

	shouldUpdate, err := c.shouldInvokeUpdate(ctx, resource, namespace, state.liveResponse)
	if err != nil {
		return servicemanager.OSOKResponse{}, err, true
	}
	if shouldUpdate {
		if c.config.ParityHooks.ApplyParityUpdate == nil {
			return servicemanager.OSOKResponse{}, fmt.Errorf("%s parity hooks require ApplyParityUpdate when RequiresParityHandling returns true", c.config.Kind), true
		}
		response, err := c.config.ParityHooks.ApplyParityUpdate(ctx, resource, state.liveResponse)
		return response, err, true
	}

	if response, handled, err := c.applyStatusHooksObservation(resource, state.liveResponse, state.identity); handled {
		return response, err, true
	}

	response, err := c.observeExistingResource(ctx, resource, state.currentID, state.liveResponse, state.identity)
	return response, err, true
}

func (c ServiceClient[T]) applyStatusHooksObservation(
	resource T,
	response any,
	identity any,
) (servicemanager.OSOKResponse, bool, error) {
	if c.config.StatusHooks.ProjectStatus == nil && c.config.StatusHooks.ApplyLifecycle == nil {
		return servicemanager.OSOKResponse{}, false, nil
	}
	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return servicemanager.OSOKResponse{}, true, err
	}
	if responseID := responseID(response); responseID != "" && c.config.Identity.RecordTracked != nil {
		c.recordTrackedIdentity(resource, identity, responseID)
	}
	if c.config.StatusHooks.ApplyLifecycle != nil {
		projected, err := c.config.StatusHooks.ApplyLifecycle(resource, response)
		return projected, true, err
	}
	projected, err := c.applySuccessWithIdentity(resource, response, shared.Active, identity)
	return projected, true, err
}

func (c ServiceClient[T]) projectStatusWithHooks(resource T, response any) error {
	if c.config.StatusHooks.ProjectStatus != nil {
		return c.config.StatusHooks.ProjectStatus(resource, response)
	}
	return mergeResponseIntoStatus(resource, response)
}

func (c ServiceClient[T]) markDeletedWithHooks(resource T, message string) {
	if c.config.StatusHooks.MarkDeleted != nil {
		c.config.StatusHooks.MarkDeleted(resource, message)
		if status, err := osokStatus(resource); err == nil {
			c.clearAsyncOperation(status, resource)
		}
		return
	}
	c.markDeleted(resource, message)
}

func (c ServiceClient[T]) markTerminatingWithHooks(resource T, response any) error {
	if c.config.StatusHooks.MarkTerminating != nil {
		c.config.StatusHooks.MarkTerminating(resource, response)
		if c.generatedWorkRequestAsyncEnabled() {
			status, err := osokStatus(resource)
			if err != nil {
				return err
			}
			if status.Async.Current == nil || status.Async.Current.Source == shared.OSOKAsyncSourceWorkRequest {
				message := strings.TrimSpace(status.Message)
				if message == "" {
					message = "OCI resource delete is in progress"
				}
				now := metav1.Now()
				_ = c.applyAsyncOperation(status, resource, &shared.OSOKAsyncOperation{
					Source:          shared.OSOKAsyncSourceLifecycle,
					Phase:           shared.OSOKAsyncPhaseDelete,
					NormalizedClass: shared.OSOKAsyncClassPending,
					Message:         message,
					UpdatedAt:       &now,
				})
			}
		}
		return nil
	}
	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return nil
}

func (c ServiceClient[T]) confirmDeleteRead(ctx context.Context, resource T, currentID string) (any, error) {
	if c.config.DeleteHooks.ConfirmRead != nil {
		response, err := c.config.DeleteHooks.ConfirmRead(ctx, resource, currentID)
		if err != nil {
			return nil, c.handleDeleteError(resource, err)
		}
		return response, nil
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		return nil, c.handleDeleteError(resource, err)
	}
	return response, nil
}

func (c ServiceClient[T]) handleDeleteError(resource T, err error) error {
	if err == nil {
		return nil
	}
	if c.config.DeleteHooks.HandleError != nil {
		if handledErr := c.config.DeleteHooks.HandleError(resource, err); handledErr != nil {
			return handledErr
		}
	}
	return err
}

func (c ServiceClient[T]) applyDeleteOutcomeHooks(resource T, response any, stage DeleteConfirmStage) (DeleteOutcome, error) {
	if c.config.DeleteHooks.ApplyOutcome == nil {
		return DeleteOutcome{}, nil
	}
	return c.config.DeleteHooks.ApplyOutcome(resource, response, stage)
}
