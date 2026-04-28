/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drg

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type stubDrgServiceClient struct {
	createOrUpdate func(context.Context, *corev1beta1.Drg, ctrl.Request) (servicemanager.OSOKResponse, error)
	deleteFn       func(context.Context, *corev1beta1.Drg) (bool, error)
}

func (s stubDrgServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *corev1beta1.Drg,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if s.createOrUpdate != nil {
		return s.createOrUpdate(ctx, resource, req)
	}
	return servicemanager.OSOKResponse{}, nil
}

func (s stubDrgServiceClient) Delete(ctx context.Context, resource *corev1beta1.Drg) (bool, error) {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, resource)
	}
	return false, nil
}

func TestDrgCreateFallbackClient_RecoversProvisioningAfterCreateReadFailure(t *testing.T) {
	resource := &corev1beta1.Drg{
		Spec: corev1beta1.DrgSpec{DisplayName: "test-drg"},
	}
	delegateErr := errors.New("Service error:InternalError. Internal Error. http status code: 500")
	client := drgCreateFallbackClient{
		delegate: stubDrgServiceClient{
			createOrUpdate: func(_ context.Context, resource *corev1beta1.Drg, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
				resource.Status.Id = "ocid1.drg.oc1..create"
				resource.Status.DisplayName = "test-drg"
				resource.Status.LifecycleState = "PROVISIONING"
				resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
					resource.Status.OsokStatus,
					shared.Failed,
					"False",
					"",
					delegateErr.Error(),
					loggerutil.OSOKLogger{Logger: logr.Discard()},
				)
				resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
				resource.Status.OsokStatus.Message = delegateErr.Error()
				resource.Status.OsokStatus.Reason = string(shared.Failed)
				return servicemanager.OSOKResponse{IsSuccessful: false}, delegateErr
			},
		},
		log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.True(t, response.ShouldRequeue)
	assert.Equal(t, time.Minute, response.RequeueDuration)
	assert.Equal(t, "ocid1.drg.oc1..create", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, string(shared.Provisioning), resource.Status.OsokStatus.Reason)
	assert.Equal(t, shared.Provisioning, resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type)
	assert.Contains(t, resource.Status.OsokStatus.Message, "PROVISIONING")
}

func TestDrgCreateFallbackClient_RecoversAvailableAfterCreateReadFailure(t *testing.T) {
	resource := &corev1beta1.Drg{
		Spec: corev1beta1.DrgSpec{DisplayName: "test-drg"},
	}
	delegateErr := errors.New("Service error:InternalError. Internal Error. http status code: 500")
	client := drgCreateFallbackClient{
		delegate: stubDrgServiceClient{
			createOrUpdate: func(_ context.Context, resource *corev1beta1.Drg, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
				resource.Status.Id = "ocid1.drg.oc1..create"
				resource.Status.DisplayName = "test-drg"
				resource.Status.LifecycleState = "AVAILABLE"
				resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, delegateErr
			},
		},
		log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, string(shared.Active), resource.Status.OsokStatus.Reason)
	assert.Equal(t, shared.Active, resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type)
}

func TestDrgCreateFallbackClient_PreservesErrorWithoutCreatedResource(t *testing.T) {
	resource := &corev1beta1.Drg{
		Spec: corev1beta1.DrgSpec{DisplayName: "test-drg"},
	}
	delegateErr := errors.New("create failed")
	client := drgCreateFallbackClient{
		delegate: stubDrgServiceClient{
			createOrUpdate: func(_ context.Context, _ *corev1beta1.Drg, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
				return servicemanager.OSOKResponse{IsSuccessful: false}, delegateErr
			},
		},
		log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.ErrorIs(t, err, delegateErr)
	assert.False(t, response.IsSuccessful)
}

func TestDrgCreateFallbackClient_PreservesErrorWhenTrackedIDAlreadyExists(t *testing.T) {
	resource := &corev1beta1.Drg{
		Spec: corev1beta1.DrgSpec{DisplayName: "test-drg"},
		Status: corev1beta1.DrgStatus{
			Id:             "ocid1.drg.oc1..existing",
			DisplayName:    "test-drg",
			LifecycleState: "AVAILABLE",
		},
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)

	delegateErr := errors.New("update read failed")
	client := drgCreateFallbackClient{
		delegate: stubDrgServiceClient{
			createOrUpdate: func(_ context.Context, resource *corev1beta1.Drg, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
				resource.Status.LifecycleState = "PROVISIONING"
				resource.Status.OsokStatus.Message = delegateErr.Error()
				resource.Status.OsokStatus.Reason = string(shared.Failed)
				return servicemanager.OSOKResponse{IsSuccessful: false}, delegateErr
			},
		},
		log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.ErrorIs(t, err, delegateErr)
	assert.False(t, response.IsSuccessful)
	assert.Equal(t, "ocid1.drg.oc1..existing", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "PROVISIONING", resource.Status.LifecycleState)
}
