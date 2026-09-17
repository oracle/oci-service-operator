/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sddc

import (
	ocvpsdk "github.com/oracle/oci-go-sdk/v65/ocvp"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockSddcName = "osok-mock-sddc"

func newMockSddcManager(sdkClient ocvpsdk.SddcClient) *SddcServiceManager {
	manager := &SddcServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSddcDefaultRuntimeHooks(sdkClient)
	applySddcRuntimeHooks(&hooks)
	delegate := defaultSddcServiceClient{ServiceClient: generatedruntime.NewServiceClient[*ocvpv1beta1.Sddc](buildSddcGeneratedRuntimeConfig(manager, hooks))}
	return manager.WithClient(wrapSddcGeneratedClient(hooks, delegate))
}
