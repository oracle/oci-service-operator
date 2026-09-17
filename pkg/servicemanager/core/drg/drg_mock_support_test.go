/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package drg

import (
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockDrgClient(sdkClient coresdk.VirtualNetworkClient) DrgServiceClient {
	manager := &DrgServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDrgDefaultRuntimeHooks(sdkClient)
	applyDrgRuntimeHooks(manager, &hooks)
	appendDrgCreateFallbackRuntimeWrapper(manager, &hooks)
	delegate := defaultDrgServiceClient{ServiceClient: generatedruntime.NewServiceClient[*corev1beta1.Drg](buildDrgGeneratedRuntimeConfig(manager, hooks))}
	return wrapDrgGeneratedClient(hooks, delegate)
}
