/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"errors"
	"testing"

	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDbSystemServiceClient struct {
	createOrUpdateFn func(context.Context, *mysqlv1beta1.DbSystem, ctrl.Request) (servicemanager.OSOKResponse, error)
	deleteFn         func(context.Context, *mysqlv1beta1.DbSystem) (bool, error)
}

func (f *fakeDbSystemServiceClient) CreateOrUpdate(ctx context.Context, resource *mysqlv1beta1.DbSystem, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, resource, req)
	}
	return servicemanager.OSOKResponse{}, nil
}

func (f *fakeDbSystemServiceClient) Delete(ctx context.Context, resource *mysqlv1beta1.DbSystem) (bool, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, resource)
	}
	return true, nil
}

func newTestDbSystemServiceManager(client DbSystemServiceClient) *DbSystemServiceManager {
	return (&DbSystemServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}).WithClient(client)
}

func TestNewDbSystemServiceManagerWithDepsUsesClientFactory(t *testing.T) {
	sentinel := &fakeDbSystemServiceClient{}
	previousFactory := newDbSystemServiceClient
	t.Cleanup(func() {
		newDbSystemServiceClient = previousFactory
	})

	newDbSystemServiceClient = func(*DbSystemServiceManager) DbSystemServiceClient {
		return sentinel
	}

	manager := NewDbSystemServiceManagerWithDeps(servicemanager.RuntimeDeps{})

	assert.Same(t, sentinel, manager.client)
}

func TestDbSystemServiceManagerCreateOrUpdateDelegatesToClient(t *testing.T) {
	resource := &mysqlv1beta1.DbSystem{}
	req := ctrl.Request{}
	expectedResponse := servicemanager.OSOKResponse{
		IsSuccessful:  true,
		ShouldRequeue: true,
	}
	expectedErr := errors.New("create failed")

	manager := newTestDbSystemServiceManager(&fakeDbSystemServiceClient{
		createOrUpdateFn: func(_ context.Context, got *mysqlv1beta1.DbSystem, gotReq ctrl.Request) (servicemanager.OSOKResponse, error) {
			assert.Same(t, resource, got)
			assert.Equal(t, req, gotReq)
			return expectedResponse, expectedErr
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, req)

	assert.Equal(t, expectedResponse, response)
	assert.ErrorIs(t, err, expectedErr)
}

func TestDbSystemServiceManagerDeleteDelegatesToClient(t *testing.T) {
	resource := &mysqlv1beta1.DbSystem{}
	expectedErr := errors.New("delete failed")

	manager := newTestDbSystemServiceManager(&fakeDbSystemServiceClient{
		deleteFn: func(_ context.Context, got *mysqlv1beta1.DbSystem) (bool, error) {
			assert.Same(t, resource, got)
			return false, expectedErr
		},
	})

	done, err := manager.Delete(context.Background(), resource)

	assert.False(t, done)
	assert.ErrorIs(t, err, expectedErr)
}

func TestDbSystemServiceManagerGetCrdStatusReturnsUnderlyingStatus(t *testing.T) {
	resource := &mysqlv1beta1.DbSystem{}
	resource.Status.OsokStatus = shared.OSOKStatus{
		Ocid:    "ocid1.dbsystem.oc1..example",
		Message: "ready",
	}

	manager := newTestDbSystemServiceManager(nil)

	status, err := manager.GetCrdStatus(resource)

	assert.NoError(t, err)
	assert.Equal(t, &resource.Status.OsokStatus, status)
}

func TestDbSystemServiceManagerRejectsWrongTypes(t *testing.T) {
	manager := newTestDbSystemServiceManager(&fakeDbSystemServiceClient{
		createOrUpdateFn: func(context.Context, *mysqlv1beta1.DbSystem, ctrl.Request) (servicemanager.OSOKResponse, error) {
			t.Fatal("CreateOrUpdate should not be called for wrong types")
			return servicemanager.OSOKResponse{}, nil
		},
		deleteFn: func(context.Context, *mysqlv1beta1.DbSystem) (bool, error) {
			t.Fatal("Delete should not be called for wrong types")
			return false, nil
		},
	})

	wrongObject := &databasev1beta1.AutonomousDatabase{}

	response, createErr := manager.CreateOrUpdate(context.Background(), wrongObject, ctrl.Request{})
	done, deleteErr := manager.Delete(context.Background(), wrongObject)
	status, statusErr := manager.GetCrdStatus(wrongObject)

	assert.False(t, response.IsSuccessful)
	assert.ErrorContains(t, createErr, "expected *mysqlv1beta1.DbSystem")
	assert.False(t, done)
	assert.ErrorContains(t, deleteErr, "expected *mysqlv1beta1.DbSystem")
	assert.Nil(t, status)
	assert.ErrorContains(t, statusErr, "expected *mysqlv1beta1.DbSystem")
}
