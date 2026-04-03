/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rediscluster

import (
	"context"
	"testing"

	redissdk "github.com/oracle/oci-go-sdk/v65/redis"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestRedisDeleteGuardSkipsDeleteWhileClusterCreating(t *testing.T) {
	t.Parallel()

	delegate := &fakeRedisDeleteGuardDelegate{}
	client := redisDeleteGuardClient{
		delegate: delegate,
		loadRedisCluster: func(context.Context, shared.OCID) (*redissdk.RedisCluster, error) {
			return &redissdk.RedisCluster{LifecycleState: redissdk.RedisClusterLifecycleStateCreating}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), testRedisDeleteGuardResource())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while lifecycle is CREATING")
	}
	if delegate.deleteCalls != 0 {
		t.Fatalf("delegate Delete() calls = %d, want 0", delegate.deleteCalls)
	}
}

func TestRedisDeleteGuardSkipsDeleteWhileClusterDeleting(t *testing.T) {
	t.Parallel()

	delegate := &fakeRedisDeleteGuardDelegate{}
	client := redisDeleteGuardClient{
		delegate: delegate,
		loadRedisCluster: func(context.Context, shared.OCID) (*redissdk.RedisCluster, error) {
			return &redissdk.RedisCluster{LifecycleState: redissdk.RedisClusterLifecycleStateDeleting}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), testRedisDeleteGuardResource())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while lifecycle is DELETING")
	}
	if delegate.deleteCalls != 0 {
		t.Fatalf("delegate Delete() calls = %d, want 0", delegate.deleteCalls)
	}
}

func TestRedisDeleteGuardReturnsDeletedWhenClusterMissing(t *testing.T) {
	t.Parallel()

	delegate := &fakeRedisDeleteGuardDelegate{}
	client := redisDeleteGuardClient{
		delegate: delegate,
		loadRedisCluster: func(context.Context, shared.OCID) (*redissdk.RedisCluster, error) {
			return nil, fakeRedisDeleteGuardServiceError{code: "NotAuthorizedOrNotFound", statusCode: 404}
		},
	}

	deleted, err := client.Delete(context.Background(), testRedisDeleteGuardResource())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should succeed when the live Redis cluster is already gone")
	}
	if delegate.deleteCalls != 0 {
		t.Fatalf("delegate Delete() calls = %d, want 0", delegate.deleteCalls)
	}
}

func TestRedisDeleteGuardDelegatesDeleteWhenClusterActive(t *testing.T) {
	t.Parallel()

	delegate := &fakeRedisDeleteGuardDelegate{}
	client := redisDeleteGuardClient{
		delegate: delegate,
		loadRedisCluster: func(context.Context, shared.OCID) (*redissdk.RedisCluster, error) {
			return &redissdk.RedisCluster{LifecycleState: redissdk.RedisClusterLifecycleStateActive}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), testRedisDeleteGuardResource())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should defer finalizer removal until the delegate confirms deletion")
	}
	if delegate.deleteCalls != 1 {
		t.Fatalf("delegate Delete() calls = %d, want 1", delegate.deleteCalls)
	}
}

func TestRedisDeleteGuardTreatsConflictAsRetryWhileClusterTransitions(t *testing.T) {
	t.Parallel()

	delegate := &fakeRedisDeleteGuardDelegate{
		deleteErr: errorutil.ConflictOciError{
			HTTPStatusCode: 409,
			Description:    "The requested state for the resource conflicts with its current state",
		},
	}
	loadCalls := 0
	client := redisDeleteGuardClient{
		delegate: delegate,
		loadRedisCluster: func(context.Context, shared.OCID) (*redissdk.RedisCluster, error) {
			loadCalls++
			if loadCalls == 1 {
				return &redissdk.RedisCluster{LifecycleState: redissdk.RedisClusterLifecycleStateActive}, nil
			}
			return &redissdk.RedisCluster{LifecycleState: redissdk.RedisClusterLifecycleStateCreating}, nil
		},
	}

	deleted, err := client.Delete(context.Background(), testRedisDeleteGuardResource())
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting after a conflict when the live cluster is still CREATING")
	}
	if delegate.deleteCalls != 1 {
		t.Fatalf("delegate Delete() calls = %d, want 1", delegate.deleteCalls)
	}
	if loadCalls != 2 {
		t.Fatalf("loadRedisCluster() calls = %d, want 2", loadCalls)
	}
}

type fakeRedisDeleteGuardDelegate struct {
	deleteCalls int
	deleteOK    bool
	deleteErr   error
}

func (f *fakeRedisDeleteGuardDelegate) CreateOrUpdate(context.Context, *redisv1beta1.RedisCluster, ctrl.Request) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{}, nil
}

func (f *fakeRedisDeleteGuardDelegate) Delete(context.Context, *redisv1beta1.RedisCluster) (bool, error) {
	f.deleteCalls++
	return f.deleteOK, f.deleteErr
}

type fakeRedisDeleteGuardServiceError struct {
	code       string
	message    string
	statusCode int
	opcID      string
}

func (f fakeRedisDeleteGuardServiceError) Error() string           { return f.message }
func (f fakeRedisDeleteGuardServiceError) GetCode() string         { return f.code }
func (f fakeRedisDeleteGuardServiceError) GetMessage() string      { return f.message }
func (f fakeRedisDeleteGuardServiceError) GetHTTPStatusCode() int  { return f.statusCode }
func (f fakeRedisDeleteGuardServiceError) GetOpcRequestID() string { return f.opcID }

func testRedisDeleteGuardResource() *redisv1beta1.RedisCluster {
	return &redisv1beta1.RedisCluster{
		Status: redisv1beta1.RedisClusterStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: "ocid1.rediscluster.oc1..test",
			},
		},
	}
}
