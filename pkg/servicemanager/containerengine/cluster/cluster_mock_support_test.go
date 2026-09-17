/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package cluster

import (
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockClusterClient(sdkClient containerenginesdk.ContainerEngineClient) ClusterServiceClient {
	manager := &ClusterServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newClusterDefaultRuntimeHooks(sdkClient)
	applyClusterRuntimeHooks(manager, &hooks)
	delegate := defaultClusterServiceClient{ServiceClient: generatedruntime.NewServiceClient[*containerenginev1beta1.Cluster](buildClusterGeneratedRuntimeConfig(manager, hooks))}
	return wrapClusterGeneratedClient(hooks, delegate)
}
