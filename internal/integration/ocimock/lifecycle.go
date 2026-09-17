/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const defaultMaxReconciles = 20

// LifecycleClient is the production service-manager seam exercised by a mock
// OCI lifecycle.
type LifecycleClient[T any] interface {
	CreateOrUpdate(context.Context, T, ctrl.Request) (servicemanager.OSOKResponse, error)
	Delete(context.Context, T) (bool, error)
}

// LifecycleScenario keeps synchronous CRUD orchestration shared while leaving
// typed resource construction, mutation, and status assertions package-local.
type LifecycleScenario[T any] struct {
	Resource        T
	Client          LifecycleClient[T]
	CreateContext   func(context.Context) context.Context
	Mutate          func(T)
	ValidateCreated func(T) error
	ValidateUpdated func(T) error
	ValidateStable  func(T) error
	// RequireAsyncPending declares phases that must expose a non-nil shared
	// status.async.current during at least one requeue and clear it on success.
	RequireAsyncPending []Operation
	RetryError          func(error) bool
	RetryDeleteError    func(error) bool
	MaxReconciles       int
}

// RunLifecycle drives create/read, update/read, and confirmed delete without
// sleeping or contacting OCI.
func RunLifecycle[T any](ctx context.Context, scenario LifecycleScenario[T]) error {
	if scenario.Client == nil {
		return fmt.Errorf("OCI mock lifecycle client is required")
	}
	if scenario.ValidateCreated == nil {
		return fmt.Errorf("OCI mock lifecycle created-state validator is required")
	}
	if scenario.Mutate == nil && scenario.ValidateUpdated != nil {
		return fmt.Errorf("OCI mock lifecycle update validator requires a mutation")
	}
	if scenario.Mutate != nil && scenario.ValidateUpdated == nil {
		return fmt.Errorf("OCI mock lifecycle mutation requires an updated-state validator")
	}
	maxReconciles := scenario.MaxReconciles
	if maxReconciles <= 0 {
		maxReconciles = defaultMaxReconciles
	}
	createContext := ctx
	if scenario.CreateContext != nil {
		createContext = scenario.CreateContext(ctx)
	}
	request := lifecycleRequest(scenario.Resource)
	if err := converge(createContext, scenario.Resource, scenario.Client, request, OperationCreate, requiresOperation(scenario.RequireAsyncPending, OperationCreate), maxReconciles, scenario.RetryError); err != nil {
		return fmt.Errorf("create/read lifecycle: %w", err)
	}
	if scenario.ValidateCreated != nil {
		if err := scenario.ValidateCreated(scenario.Resource); err != nil {
			return fmt.Errorf("validate created resource: %w", err)
		}
	}
	if scenario.Mutate != nil {
		scenario.Mutate(scenario.Resource)
		if err := converge(ctx, scenario.Resource, scenario.Client, request, OperationUpdate, requiresOperation(scenario.RequireAsyncPending, OperationUpdate), maxReconciles, scenario.RetryError); err != nil {
			return fmt.Errorf("update/read lifecycle: %w", err)
		}
		if scenario.ValidateUpdated != nil {
			if err := scenario.ValidateUpdated(scenario.Resource); err != nil {
				return fmt.Errorf("validate updated resource: %w", err)
			}
		}
	}
	if scenario.ValidateStable != nil {
		response, err := scenario.Client.CreateOrUpdate(ctx, scenario.Resource, request)
		if err != nil {
			return fmt.Errorf("stable lifecycle: %w", err)
		}
		if !response.IsSuccessful {
			return fmt.Errorf("stable lifecycle was unsuccessful: %+v", response)
		}
		if response.ShouldRequeue {
			return fmt.Errorf("stable lifecycle unexpectedly requested requeue: %+v", response)
		}
		if err := scenario.ValidateStable(scenario.Resource); err != nil {
			return fmt.Errorf("validate stable resource: %w", err)
		}
	}
	requireDeleteAsync := requiresOperation(scenario.RequireAsyncPending, OperationDelete)
	sawDeleteAsync := false
	for attempt := 1; attempt <= maxReconciles; attempt++ {
		deleted, err := scenario.Client.Delete(ctx, scenario.Resource)
		if err != nil {
			if scenario.RetryDeleteError != nil && scenario.RetryDeleteError(err) {
				continue
			}
			return fmt.Errorf("delete lifecycle attempt %d: %w", attempt, err)
		}
		if requireDeleteAsync {
			pending, pendingErr := asyncCurrentPresent(scenario.Resource)
			if pendingErr != nil {
				return fmt.Errorf("delete lifecycle attempt %d async status: %w", attempt, pendingErr)
			}
			if pending {
				sawDeleteAsync = true
			}
		}
		if deleted {
			if requireDeleteAsync && !sawDeleteAsync {
				return fmt.Errorf("delete lifecycle did not expose status.async.current")
			}
			if requireDeleteAsync {
				pending, pendingErr := asyncCurrentPresent(scenario.Resource)
				if pendingErr != nil {
					return fmt.Errorf("delete lifecycle terminal async status: %w", pendingErr)
				}
				if pending {
					return fmt.Errorf("delete lifecycle left status.async.current set")
				}
			}
			return nil
		}
	}
	return fmt.Errorf("delete lifecycle did not converge after %d attempts", maxReconciles)
}

func lifecycleRequest(resource any) ctrl.Request {
	object, ok := resource.(metav1.Object)
	if !ok {
		return ctrl.Request{}
	}
	return ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}}
}

func converge[T any](ctx context.Context, resource T, client LifecycleClient[T], request ctrl.Request, phase Operation, requireAsyncPending bool, maxReconciles int, retryError func(error) bool) error {
	sawAsyncPending := false
	for attempt := 1; attempt <= maxReconciles; attempt++ {
		response, err := client.CreateOrUpdate(ctx, resource, request)
		if err != nil {
			if retryError != nil && retryError(err) {
				continue
			}
			return fmt.Errorf("reconcile attempt %d: %w", attempt, err)
		}
		if !response.IsSuccessful {
			return fmt.Errorf("reconcile attempt %d was unsuccessful: %+v", attempt, response)
		}
		if requireAsyncPending {
			pending, pendingErr := asyncCurrentPresent(resource)
			if pendingErr != nil {
				return fmt.Errorf("reconcile attempt %d %s async status: %w", attempt, phase, pendingErr)
			}
			if pending {
				sawAsyncPending = true
			}
		}
		if !response.ShouldRequeue {
			if requireAsyncPending && !sawAsyncPending {
				return fmt.Errorf("%s lifecycle did not expose status.async.current", phase)
			}
			if requireAsyncPending {
				pending, pendingErr := asyncCurrentPresent(resource)
				if pendingErr != nil {
					return fmt.Errorf("%s lifecycle terminal async status: %w", phase, pendingErr)
				}
				if pending {
					return fmt.Errorf("%s lifecycle left status.async.current set", phase)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("reconcile did not converge after %d attempts", maxReconciles)
}

func requiresOperation(operations []Operation, want Operation) bool {
	for _, operation := range operations {
		if operation == want {
			return true
		}
	}
	return false
}

func asyncCurrentPresent(resource any) (bool, error) {
	payload, err := json.Marshal(resource)
	if err != nil {
		return false, err
	}
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return false, err
	}
	status, _ := object["status"].(map[string]any)
	sharedStatus, _ := status["status"].(map[string]any)
	async, _ := sharedStatus["async"].(map[string]any)
	current, exists := async["current"]
	return exists && current != nil, nil
}
