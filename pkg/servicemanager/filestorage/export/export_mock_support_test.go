/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package export

import (
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockExportClient(sdkClient filestoragesdk.FileStorageClient) ExportServiceClient {
	manager := &ExportServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newExportRuntimeHooks(manager, sdkClient)
	delegate := defaultExportServiceClient{ServiceClient: generatedruntime.NewServiceClient[*filestoragev1beta1.Export](buildExportGeneratedRuntimeConfig(manager, hooks))}
	return wrapExportGeneratedClient(hooks, delegate)
}
