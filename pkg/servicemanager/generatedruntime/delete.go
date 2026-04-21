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

	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func (c ServiceClient[T]) Delete(ctx context.Context, resource T) (bool, error) {
	if err := c.validateDeleteRequest(resource); err != nil {
		return false, err
	}
	if c.config.Semantics != nil {
		return c.deleteWithSemantics(ctx, resource)
	}
	if c.config.Delete == nil {
		c.markDeleted(resource, "OCI delete is not supported for this generated resource")
		return true, nil
	}

	currentID := c.currentID(resource)
	if currentID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	if deleted, err := c.invokeDeleteOperation(ctx, resource, currentID); deleted || err != nil {
		return deleted, err
	}
	return c.confirmDeleteWithoutSemantics(ctx, resource, currentID)
}

func (c ServiceClient[T]) validateDeleteRequest(resource T) error {
	if c.config.InitError != nil {
		return c.config.InitError
	}
	_, err := resourceStruct(resource)
	return err
}

func (c ServiceClient[T]) invokeDeleteOperation(ctx context.Context, resource T, currentID string) (bool, error) {
	response, err := c.invoke(ctx, c.config.Delete, resource, currentID, requestBuildOptions{})
	if err != nil {
		if isDeleteNotFound(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}
	c.seedOpeningRequestID(resource, response)
	c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseDelete)
	return false, nil
}

func (c ServiceClient[T]) confirmDeleteWithoutSemantics(ctx context.Context, resource T, currentID string) (bool, error) {
	if c.config.Get == nil && c.config.List == nil {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}

	_ = mergeResponseIntoStatus(resource, response)
	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return false, nil
}

func (c ServiceClient[T]) followUpAfterWrite(ctx context.Context, resource T, preferredID string, response any, phase string) (any, error) {
	if !c.requiresWriteFollowUp(phase) {
		return response, nil
	}
	if c.config.Get == nil && c.config.List == nil {
		if c.config.Semantics != nil {
			return nil, fmt.Errorf("%s formal semantics require %s follow-up without a readable OCI operation", c.config.Kind, phase)
		}
		return response, nil
	}

	refreshed, err := c.readResource(ctx, resource, preferredID, phaseReadPhase(phase))
	if err == nil {
		return refreshed, nil
	}
	if phase == "create" && errors.Is(err, errResourceNotFound) {
		return response, nil
	}
	return nil, err
}

func (c ServiceClient[T]) requiresWriteFollowUp(phase string) bool {
	if c.config.Semantics == nil {
		return c.config.Get != nil || c.config.List != nil
	}

	switch phase {
	case "create":
		return c.config.Semantics.CreateFollowUp.Strategy == "read-after-write"
	case "update":
		return c.config.Semantics.UpdateFollowUp.Strategy == "read-after-write"
	default:
		return false
	}
}

func phaseReadPhase(phase string) readPhase {
	switch phase {
	case "create":
		return readPhaseCreate
	case "update":
		return readPhaseUpdate
	case "delete":
		return readPhaseDelete
	default:
		return readPhaseObserve
	}
}

func (c ServiceClient[T]) deleteWithSemantics(ctx context.Context, resource T) (bool, error) {
	semantics, err := c.semanticDeleteConfig()
	if err != nil {
		return false, err
	}

	currentID, err := c.resolveDeleteID(ctx, resource)
	if err != nil {
		if errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}
	if deleted, err, handled := c.confirmDeleteIfAlreadyPending(ctx, resource, currentID, semantics); handled {
		return deleted, err
	}
	deleted, err := c.invokeDeleteOperation(ctx, resource, currentID)
	if deleted {
		return true, nil
	}
	if err != nil && !c.shouldConfirmDeleteAfterError(err) {
		return false, err
	}
	return c.confirmDeleteWithSemantics(ctx, resource, currentID, semantics)
}

func (c ServiceClient[T]) confirmDeleteIfAlreadyPending(ctx context.Context, resource T, currentID string, semantics *Semantics) (bool, error, bool) {
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		return false, nil, false
	}
	if c.config.Get == nil && c.config.List == nil {
		return false, nil, false
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil, true
		}
		return false, nil, false
	}

	lifecycleState := strings.ToUpper(responseLifecycleState(response))
	if !containsString(semantics.Delete.PendingStates, lifecycleState) &&
		!containsString(semantics.Delete.TerminalStates, lifecycleState) {
		return false, nil, false
	}

	_ = mergeResponseIntoStatus(resource, response)
	deleted, err := c.applyDeletePolicy(resource, response, semantics)
	return deleted, err, true
}

func (c ServiceClient[T]) shouldConfirmDeleteAfterError(err error) bool {
	if err == nil || !isRetryableDeleteConflict(err) {
		return false
	}
	if c.config.Semantics == nil || c.config.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		return false
	}
	return c.config.Get != nil || c.config.List != nil
}

func (c ServiceClient[T]) semanticDeleteConfig() (*Semantics, error) {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil, fmt.Errorf("%s formal semantics are not configured", c.config.Kind)
	}
	if c.config.Delete == nil || semantics.Delete.Policy == "not-supported" {
		return nil, fmt.Errorf("%s formal semantics mark delete confirmation as %q", c.config.Kind, semantics.Delete.Policy)
	}
	return semantics, nil
}

func (c ServiceClient[T]) confirmDeleteWithSemantics(ctx context.Context, resource T, currentID string, semantics *Semantics) (bool, error) {
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}
	if c.config.Get == nil && c.config.List == nil {
		return false, fmt.Errorf("%s formal delete confirmation requires a readable OCI operation", c.config.Kind)
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.recordErrorRequestID(resource, err)
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}

	_ = mergeResponseIntoStatus(resource, response)
	return c.applyDeletePolicy(resource, response, semantics)
}

func (c ServiceClient[T]) applyDeletePolicy(resource T, response any, semantics *Semantics) (bool, error) {
	lifecycleState := strings.ToUpper(responseLifecycleState(response))
	switch semantics.Delete.Policy {
	case "best-effort":
		return c.bestEffortDeleteOutcome(resource, lifecycleState, semantics)
	case "required":
		return c.requiredDeleteOutcome(resource, lifecycleState, semantics)
	default:
		return false, fmt.Errorf("%s formal delete confirmation policy %q is not supported", c.config.Kind, semantics.Delete.Policy)
	}
}

func (c ServiceClient[T]) bestEffortDeleteOutcome(resource T, lifecycleState string, semantics *Semantics) (bool, error) {
	if lifecycleState == "" ||
		containsString(semantics.Delete.PendingStates, lifecycleState) ||
		containsString(semantics.Delete.TerminalStates, lifecycleState) {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}

	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return false, nil
}

func (c ServiceClient[T]) requiredDeleteOutcome(resource T, lifecycleState string, semantics *Semantics) (bool, error) {
	switch {
	case containsString(semantics.Delete.TerminalStates, lifecycleState):
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	case lifecycleState == "" || containsString(semantics.Delete.PendingStates, lifecycleState):
		c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
		return false, nil
	default:
		return false, fmt.Errorf("%s delete confirmation returned unexpected lifecycle state %q", c.config.Kind, lifecycleState)
	}
}

func (c ServiceClient[T]) resolveDeleteID(ctx context.Context, resource T) (string, error) {
	currentID := c.currentID(resource)
	if currentID != "" {
		return currentID, nil
	}

	if c.config.Get == nil && c.config.List == nil {
		return "", errResourceNotFound
	}

	response, err := c.readResource(ctx, resource, "", readPhaseDelete)
	if err != nil {
		return "", err
	}
	currentID = responseID(response)
	if currentID == "" {
		return "", fmt.Errorf("%s delete confirmation could not resolve a resource OCID", c.config.Kind)
	}
	_ = mergeResponseIntoStatus(resource, response)
	return currentID, nil
}
