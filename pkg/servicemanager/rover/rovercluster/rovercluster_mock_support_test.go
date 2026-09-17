/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package rovercluster

import (
	roversdk "github.com/oracle/oci-go-sdk/v65/rover"
	roverv1beta1 "github.com/oracle/oci-service-operator/api/rover/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockRoverClusterName = "osok-mock-rover-v1"

func newMockRoverClusterManager(sdkClient roversdk.RoverClusterClient) *RoverClusterServiceManager {
	manager := &RoverClusterServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newRoverClusterRuntimeHooks(manager, sdkClient)
	client := defaultRoverClusterServiceClient{ServiceClient: generatedruntime.NewServiceClient[*roverv1beta1.RoverCluster](buildRoverClusterGeneratedRuntimeConfig(manager, hooks))}
	return manager.WithClient(client)
}
