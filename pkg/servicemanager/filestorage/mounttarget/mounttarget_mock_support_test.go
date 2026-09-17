/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package mounttarget

import (
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockMountTargetClient(sdkClient filestoragesdk.FileStorageClient) MountTargetServiceClient {
	manager := &MountTargetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newMountTargetRuntimeHooks(manager, sdkClient)
	delegate := defaultMountTargetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*filestoragev1beta1.MountTarget](buildMountTargetGeneratedRuntimeConfig(manager, hooks))}
	return wrapMountTargetGeneratedClient(hooks, delegate)
}
