/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package log

import (
	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockLogClient(sdkClient loggingsdk.LoggingManagementClient) LogServiceClient {
	manager := &LogServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newLogDefaultRuntimeHooks(sdkClient)
	applyLogRuntimeHooks(&hooks)
	delegate := defaultLogServiceClient{ServiceClient: generatedruntime.NewServiceClient[*loggingv1beta1.Log](buildLogGeneratedRuntimeConfig(manager, hooks))}
	return wrapLogGeneratedClient(hooks, delegate)
}
