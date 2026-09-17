/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type lifecycleTestResource struct {
	Name    string
	Created bool
	Deleted bool
}

type lifecycleTestClient struct {
	createCalls   int
	updateCalls   int
	stableCalls   int
	deleteCalls   int
	retryDelete   bool
	stableRequeue bool
}

type asyncLifecycleTestResource struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            struct {
		OSOKStatus shared.OSOKStatus `json:"status"`
	} `json:"status,omitempty"`
}

type asyncLifecycleTestClient struct {
	createCalls int
}

func (c *asyncLifecycleTestClient) CreateOrUpdate(_ context.Context, resource *asyncLifecycleTestResource, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	c.createCalls++
	if c.createCalls == 1 {
		resource.Status.OSOKStatus.Async.Current = &shared.OSOKAsyncOperation{WorkRequestID: "work-request-1"}
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true}, nil
	}
	resource.Status.OSOKStatus.Async.Current = nil
	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func (*asyncLifecycleTestClient) Delete(context.Context, *asyncLifecycleTestResource) (bool, error) {
	return true, nil
}

func (c *lifecycleTestClient) CreateOrUpdate(_ context.Context, resource *lifecycleTestResource, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	if !resource.Created {
		c.createCalls++
		resource.Created = true
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	}
	if c.updateCalls == 0 {
		c.updateCalls++
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	}
	c.stableCalls++
	return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: c.stableRequeue}, nil
}

func (c *lifecycleTestClient) Delete(_ context.Context, resource *lifecycleTestResource) (bool, error) {
	c.deleteCalls++
	if c.retryDelete && c.deleteCalls == 1 {
		return false, fmt.Errorf("retry delete")
	}
	resource.Deleted = true
	return true, nil
}

func TestRunLifecycleExecutesTypedCRUDContract(t *testing.T) {
	t.Parallel()

	resource := &lifecycleTestResource{Name: "created"}
	client := &lifecycleTestClient{}
	err := RunLifecycle(context.Background(), LifecycleScenario[*lifecycleTestResource]{
		Resource: resource,
		Client:   client,
		ValidateCreated: func(current *lifecycleTestResource) error {
			if !current.Created || current.Name != "created" {
				return fmt.Errorf("created resource = %+v", current)
			}
			return nil
		},
		Mutate: func(current *lifecycleTestResource) {
			current.Name = "updated"
		},
		ValidateUpdated: func(current *lifecycleTestResource) error {
			if current.Name != "updated" {
				return fmt.Errorf("updated resource = %+v", current)
			}
			return nil
		},
		ValidateStable: func(current *lifecycleTestResource) error {
			if current.Name != "updated" {
				return fmt.Errorf("stable resource = %+v", current)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.createCalls != 1 || client.updateCalls != 1 || client.stableCalls != 1 || client.deleteCalls != 1 || !resource.Deleted {
		t.Fatalf("calls create/update/stable/delete=%d/%d/%d/%d resource=%+v", client.createCalls, client.updateCalls, client.stableCalls, client.deleteCalls, resource)
	}
}

func TestLifecycleRequestUsesKubernetesIdentity(t *testing.T) {
	t.Parallel()

	resource := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{
		Namespace: "test-namespace",
		Name:      "test-name",
	}}
	request := lifecycleRequest(resource)
	if request.Namespace != resource.Namespace || request.Name != resource.Name {
		t.Fatalf("lifecycle request = %s/%s, want %s/%s", request.Namespace, request.Name, resource.Namespace, resource.Name)
	}
}

func TestRunLifecycleRequiresPendingAsyncStateForDeclaredPhase(t *testing.T) {
	t.Parallel()

	resource := &asyncLifecycleTestResource{ObjectMeta: metav1.ObjectMeta{Name: "async", Namespace: "default"}}
	err := RunLifecycle(context.Background(), LifecycleScenario[*asyncLifecycleTestResource]{
		Resource:            resource,
		Client:              &asyncLifecycleTestClient{},
		RequireAsyncPending: []Operation{OperationCreate},
		ValidateCreated:     func(*asyncLifecycleTestResource) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunLifecycleRejectsMissingDeclaredAsyncState(t *testing.T) {
	t.Parallel()

	resource := &asyncLifecycleTestResource{ObjectMeta: metav1.ObjectMeta{Name: "async", Namespace: "default"}}
	client := &asyncLifecycleTestClient{createCalls: 1}
	err := RunLifecycle(context.Background(), LifecycleScenario[*asyncLifecycleTestResource]{
		Resource:            resource,
		Client:              client,
		RequireAsyncPending: []Operation{OperationCreate},
		ValidateCreated:     func(*asyncLifecycleTestResource) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "create lifecycle did not expose status.async.current") {
		t.Fatalf("RunLifecycle() error = %v, want missing async state", err)
	}
}

func TestRunLifecycleRejectsStableRequeue(t *testing.T) {
	t.Parallel()

	resource := &lifecycleTestResource{Name: "created"}
	err := RunLifecycle(context.Background(), LifecycleScenario[*lifecycleTestResource]{
		Resource: resource,
		Client:   &lifecycleTestClient{stableRequeue: true},
		ValidateCreated: func(*lifecycleTestResource) error {
			return nil
		},
		Mutate: func(current *lifecycleTestResource) {
			current.Name = "updated"
		},
		ValidateUpdated: func(*lifecycleTestResource) error {
			return nil
		},
		ValidateStable: func(*lifecycleTestResource) error {
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "stable lifecycle unexpectedly requested requeue") {
		t.Fatalf("RunLifecycle() error = %v, want stable requeue rejection", err)
	}
}

func TestRunLifecycleRejectsMutationWithoutUpdatedStateValidator(t *testing.T) {
	t.Parallel()

	resource := &lifecycleTestResource{Name: "created"}
	err := RunLifecycle(context.Background(), LifecycleScenario[*lifecycleTestResource]{
		Resource: resource,
		Client:   &lifecycleTestClient{},
		ValidateCreated: func(*lifecycleTestResource) error {
			return nil
		},
		Mutate: func(current *lifecycleTestResource) {
			current.Name = "updated"
		},
	})
	if err == nil {
		t.Fatal("RunLifecycle() error = nil, want missing update validator")
	}
}

func TestRunLifecycleRetriesClassifiedDeleteError(t *testing.T) {
	t.Parallel()

	resource := &lifecycleTestResource{Name: "created"}
	client := &lifecycleTestClient{retryDelete: true}
	err := RunLifecycle(context.Background(), LifecycleScenario[*lifecycleTestResource]{
		Resource: resource,
		Client:   client,
		ValidateCreated: func(*lifecycleTestResource) error {
			return nil
		},
		RetryDeleteError: func(err error) bool {
			return err != nil && err.Error() == "retry delete"
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.deleteCalls != 2 || !resource.Deleted {
		t.Fatalf("delete calls=%d resource=%+v", client.deleteCalls, resource)
	}
}
