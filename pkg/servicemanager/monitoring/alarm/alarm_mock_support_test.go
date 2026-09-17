/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package alarm

import (
	monitoringsdk "github.com/oracle/oci-go-sdk/v65/monitoring"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockAlarmClient(sdkClient monitoringsdk.MonitoringClient) AlarmServiceClient {
	manager := &AlarmServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAlarmDefaultRuntimeHooks(sdkClient)
	return defaultAlarmServiceClient{ServiceClient: generatedruntime.NewServiceClient[*monitoringv1beta1.Alarm](buildAlarmGeneratedRuntimeConfig(manager, hooks))}
}
