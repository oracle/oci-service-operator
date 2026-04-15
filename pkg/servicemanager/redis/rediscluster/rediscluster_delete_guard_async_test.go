package rediscluster

import (
	"context"
	"testing"

	redissdk "github.com/oracle/oci-go-sdk/v65/redis"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestRedisDeleteGuardTreatsConflictAsDeletedWhenFollowUpReadMissing(t *testing.T) {
	t.Parallel()

	delegate := &fakeRedisDeleteGuardDelegate{
		deleteErr: errorutil.ConflictOciError{
			HTTPStatusCode: 409,
			Description:    "delete conflict",
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
			return nil, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "redis cluster not found")
		},
	}

	deleted, err := client.Delete(context.Background(), testRedisDeleteGuardResource())
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil once follow-up read confirms cluster missing", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success when conflict follow-up confirms the Redis cluster is gone")
	}
	if delegate.deleteCalls != 1 {
		t.Fatalf("delegate Delete() calls = %d, want 1", delegate.deleteCalls)
	}
	if loadCalls != 2 {
		t.Fatalf("loadRedisCluster() calls = %d, want 2", loadCalls)
	}
}

func TestRedisDeleteGuardPreservesConflictWhenFollowUpReadFailsServerSide(t *testing.T) {
	t.Parallel()

	delegate := &fakeRedisDeleteGuardDelegate{
		deleteErr: errorutil.ConflictOciError{
			HTTPStatusCode: 409,
			Description:    "delete conflict",
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
			return nil, errortest.NewServiceError(500, "InternalServerError", "redis readback failed")
		},
	}

	deleted, err := client.Delete(context.Background(), testRedisDeleteGuardResource())
	if deleted {
		t.Fatal("Delete() should keep the finalizer when the delete-guard readback still fails")
	}
	errortest.AssertErrorType(t, err, "errorutil.ConflictOciError")
	if delegate.deleteCalls != 1 {
		t.Fatalf("delegate Delete() calls = %d, want 1", delegate.deleteCalls)
	}
	if loadCalls != 2 {
		t.Fatalf("loadRedisCluster() calls = %d, want 2", loadCalls)
	}
}
