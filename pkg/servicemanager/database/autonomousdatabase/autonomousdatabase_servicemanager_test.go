//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package autonomousdatabase

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

type fakeAutonomousDatabaseServiceClient struct {
	createOrUpdateFn func(context.Context, *databasev1beta1.AutonomousDatabase, ctrl.Request) (servicemanager.OSOKResponse, error)
	deleteFn         func(context.Context, *databasev1beta1.AutonomousDatabase) (bool, error)
}

func (f *fakeAutonomousDatabaseServiceClient) CreateOrUpdate(ctx context.Context, resource *databasev1beta1.AutonomousDatabase, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, resource, req)
	}
	return servicemanager.OSOKResponse{}, nil
}

func (f *fakeAutonomousDatabaseServiceClient) Delete(ctx context.Context, resource *databasev1beta1.AutonomousDatabase) (bool, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, resource)
	}
	return true, nil
}

func newTestAutonomousDatabaseServiceManager(client AutonomousDatabaseServiceClient) *AutonomousDatabaseServiceManager {
	return (&AutonomousDatabaseServiceManager{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}).WithClient(client)
}

func TestNewAutonomousDatabaseServiceManagerWithDepsUsesClientFactory(t *testing.T) {
	sentinel := &fakeAutonomousDatabaseServiceClient{}
	previousFactory := newAutonomousDatabaseServiceClient
	t.Cleanup(func() {
		newAutonomousDatabaseServiceClient = previousFactory
	})

	newAutonomousDatabaseServiceClient = func(*AutonomousDatabaseServiceManager) AutonomousDatabaseServiceClient {
		return sentinel
	}

	manager := NewAutonomousDatabaseServiceManagerWithDeps(servicemanager.RuntimeDeps{})

	assert.Same(t, sentinel, manager.client)
}

func TestAutonomousDatabaseServiceManagerCreateOrUpdateDelegatesToClient(t *testing.T) {
	resource := &databasev1beta1.AutonomousDatabase{}
	req := ctrl.Request{}
	expectedResponse := servicemanager.OSOKResponse{
		IsSuccessful:  true,
		ShouldRequeue: true,
	}
	expectedErr := errors.New("create failed")

	manager := newTestAutonomousDatabaseServiceManager(&fakeAutonomousDatabaseServiceClient{
		createOrUpdateFn: func(_ context.Context, got *databasev1beta1.AutonomousDatabase, gotReq ctrl.Request) (servicemanager.OSOKResponse, error) {
			assert.Same(t, resource, got)
			assert.Equal(t, req, gotReq)
			return expectedResponse, expectedErr
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, req)

	assert.Equal(t, expectedResponse, response)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAutonomousDatabaseServiceManagerDeleteDelegatesToClient(t *testing.T) {
	resource := &databasev1beta1.AutonomousDatabase{}
	expectedErr := errors.New("delete failed")

	manager := newTestAutonomousDatabaseServiceManager(&fakeAutonomousDatabaseServiceClient{
		deleteFn: func(_ context.Context, got *databasev1beta1.AutonomousDatabase) (bool, error) {
			assert.Same(t, resource, got)
			return false, expectedErr
		},
	})

	done, err := manager.Delete(context.Background(), resource)

	assert.False(t, done)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAutonomousDatabaseServiceManagerGetCrdStatusReturnsUnderlyingStatus(t *testing.T) {
	resource := &databasev1beta1.AutonomousDatabase{}
	resource.Status.OsokStatus = shared.OSOKStatus{
		Ocid:    "ocid1.autonomousdatabase.oc1..example",
		Message: "ready",
	}

	manager := newTestAutonomousDatabaseServiceManager(nil)

	status, err := manager.GetCrdStatus(resource)

	assert.NoError(t, err)
	assert.Equal(t, &resource.Status.OsokStatus, status)
}

func TestAutonomousDatabaseServiceManagerRejectsWrongTypes(t *testing.T) {
	manager := newTestAutonomousDatabaseServiceManager(&fakeAutonomousDatabaseServiceClient{
		createOrUpdateFn: func(context.Context, *databasev1beta1.AutonomousDatabase, ctrl.Request) (servicemanager.OSOKResponse, error) {
			t.Fatal("CreateOrUpdate should not be called for wrong types")
			return servicemanager.OSOKResponse{}, nil
		},
		deleteFn: func(context.Context, *databasev1beta1.AutonomousDatabase) (bool, error) {
			t.Fatal("Delete should not be called for wrong types")
			return false, nil
		},
	})

	wrongObject := &mysqlv1beta1.DbSystem{}

	response, createErr := manager.CreateOrUpdate(context.Background(), wrongObject, ctrl.Request{})
	done, deleteErr := manager.Delete(context.Background(), wrongObject)
	status, statusErr := manager.GetCrdStatus(wrongObject)

	assert.False(t, response.IsSuccessful)
	assert.ErrorContains(t, createErr, "expected *databasev1beta1.AutonomousDatabase")
	assert.False(t, done)
	assert.ErrorContains(t, deleteErr, "expected *databasev1beta1.AutonomousDatabase")
	assert.Nil(t, status)
	assert.ErrorContains(t, statusErr, "expected *databasev1beta1.AutonomousDatabase")
}
