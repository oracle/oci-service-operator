/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package nodepool

import (
	containerenginesdk "github.com/oracle/oci-go-sdk/v65/containerengine"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockNodePoolClient(sdkClient containerenginesdk.ContainerEngineClient) NodePoolServiceClient {
	manager := &NodePoolServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newNodePoolDefaultRuntimeHooks(sdkClient)
	applyNodePoolRuntimeHooks(&hooks)
	delegate := defaultNodePoolServiceClient{ServiceClient: generatedruntime.NewServiceClient[*containerenginev1beta1.NodePool](buildNodePoolGeneratedRuntimeConfig(manager, hooks))}
	return wrapNodePoolGeneratedClient(hooks, delegate)
}
