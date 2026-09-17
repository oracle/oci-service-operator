/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package project

import (
	datasciencesdk "github.com/oracle/oci-go-sdk/v65/datascience"
	datasciencev1beta1 "github.com/oracle/oci-service-operator/api/datascience/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockDataScienceProjectClient(sdkClient datasciencesdk.DataScienceClient) ProjectServiceClient {
	manager := &ProjectServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newProjectRuntimeHooks(manager, sdkClient)
	delegate := defaultProjectServiceClient{ServiceClient: generatedruntime.NewServiceClient[*datasciencev1beta1.Project](buildProjectGeneratedRuntimeConfig(manager, hooks))}
	return wrapProjectGeneratedClient(hooks, delegate)
}
