/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package opensearchcluster

import (
	opensearchsdk "github.com/oracle/oci-go-sdk/v65/opensearch"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockOpensearchClusterName = "osok-mock-opensearch-v1"

func newMockOpensearchClusterClient(sdkClient opensearchsdk.OpensearchClusterClient) OpensearchClusterServiceClient {
	manager := &OpensearchClusterServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newOpensearchClusterDefaultRuntimeHooks(sdkClient)
	applyOpensearchClusterRuntimeHooks(manager, &hooks)
	delegate := defaultOpensearchClusterServiceClient{ServiceClient: generatedruntime.NewServiceClient[*opensearchv1beta1.OpensearchCluster](buildOpensearchClusterGeneratedRuntimeConfig(manager, hooks))}
	return wrapOpensearchClusterGeneratedClient(hooks, delegate)
}
