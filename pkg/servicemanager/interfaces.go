/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/api/v1beta1"
)

type OSOKResponse struct {
	IsSuccessful    bool
	ShouldRequeue   bool
	RequeueDuration time.Duration
}

type OSOKServiceManager interface {
	CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (OSOKResponse, error)

	Delete(ctx context.Context, obj runtime.Object) (bool, error)

	GetCrdStatus(obj runtime.Object) (*v1beta1.OSOKStatus, error)
}
