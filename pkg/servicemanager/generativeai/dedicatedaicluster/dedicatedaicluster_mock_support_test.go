/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dedicatedaicluster

import (
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockDedicatedAiClusterName = "osok-mock-ai-cluster-v1"

func newMockDedicatedAiClusterClient(sdkClient generativeaisdk.GenerativeAiClient) DedicatedAiClusterServiceClient {
	manager := &DedicatedAiClusterServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDedicatedAiClusterDefaultRuntimeHooks(sdkClient)
	applyDedicatedAiClusterRuntimeHooks(&hooks)
	delegate := defaultDedicatedAiClusterServiceClient{ServiceClient: generatedruntime.NewServiceClient[*generativeaiv1beta1.DedicatedAiCluster](buildDedicatedAiClusterGeneratedRuntimeConfig(manager, hooks))}
	return wrapDedicatedAiClusterGeneratedClient(hooks, delegate)
}
