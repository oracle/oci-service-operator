/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package bucket

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	objectstoragesdk "github.com/oracle/oci-go-sdk/v65/objectstorage"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type bucketNamespaceGetter interface {
	GetNamespace(context.Context, objectstoragesdk.GetNamespaceRequest) (objectstoragesdk.GetNamespaceResponse, error)
}

// Wrap the generated runtime client so bucket requests keep the legacy namespace
// behavior: if the spec omits the Object Storage namespace, resolve it from OCI
// instead of accidentally using the Kubernetes namespace.
type namespaceResolvingBucketServiceClient struct {
	delegate        BucketServiceClient
	namespaceGetter bucketNamespaceGetter
}

func init() {
	registerBucketRuntimeHooksMutator(func(manager *BucketServiceManager, hooks *BucketRuntimeHooks) {
		applyBucketRuntimeHooks(hooks)
		appendBucketNamespaceRuntimeWrapper(manager, hooks)
	})
}

func applyBucketRuntimeHooks(hooks *BucketRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Create.Fields = bucketCreateFields()
	hooks.Get.Fields = bucketGetFields()
	hooks.List.Fields = bucketListFields()
	hooks.Update.Fields = bucketUpdateFields()
	hooks.Delete.Fields = bucketDeleteFields()
}

func appendBucketNamespaceRuntimeWrapper(manager *BucketServiceManager, hooks *BucketRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate BucketServiceClient) BucketServiceClient {
		return newNamespaceResolvingBucketServiceClient(manager, delegate)
	})
}

func newNamespaceResolvingBucketServiceClient(manager *BucketServiceManager, delegate BucketServiceClient) *namespaceResolvingBucketServiceClient {
	client := &namespaceResolvingBucketServiceClient{delegate: delegate}
	if manager == nil {
		return client
	}

	sdkClient, err := objectstoragesdk.NewObjectStorageClientWithConfigurationProvider(manager.Provider)
	if err == nil {
		client.namespaceGetter = sdkClient
	}
	return client
}

func bucketCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		bucketNamespaceField(),
		{FieldName: "CreateBucketDetails", Contribution: "body"},
	}
}

func bucketGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		bucketNamespaceField(),
		bucketNameField(),
	}
}

func bucketListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		bucketNamespaceField(),
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
	}
}

func bucketUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		bucketNamespaceField(),
		bucketNameField(),
		{FieldName: "UpdateBucketDetails", Contribution: "body"},
	}
}

func bucketDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		bucketNamespaceField(),
		bucketNameField(),
	}
}

func bucketNamespaceField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "NamespaceName",
		RequestName:  "namespaceName",
		Contribution: "path",
		LookupPaths:  []string{"status.namespace", "spec.namespace", "namespace"},
	}
}

func bucketNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "BucketName",
		RequestName:  "bucketName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

func (c *namespaceResolvingBucketServiceClient) CreateOrUpdate(ctx context.Context, resource *objectstoragev1beta1.Bucket, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if err := c.ensureNamespace(ctx, resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *namespaceResolvingBucketServiceClient) Delete(ctx context.Context, resource *objectstoragev1beta1.Bucket) (bool, error) {
	if err := c.ensureNamespace(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *namespaceResolvingBucketServiceClient) ensureNamespace(ctx context.Context, resource *objectstoragev1beta1.Bucket) error {
	if resource == nil {
		return fmt.Errorf("bucket resource is nil")
	}
	if strings.TrimSpace(resource.Spec.Namespace) != "" || strings.TrimSpace(resource.Status.Namespace) != "" {
		return nil
	}
	if c.namespaceGetter == nil {
		return nil
	}

	request := objectstoragesdk.GetNamespaceRequest{}
	if compartmentID := strings.TrimSpace(resource.Spec.CompartmentId); compartmentID != "" {
		request.CompartmentId = common.String(compartmentID)
	}

	response, err := c.namespaceGetter.GetNamespace(ctx, request)
	if err != nil {
		return fmt.Errorf("lookup Bucket namespace: %w", err)
	}
	namespace := ""
	if response.Value != nil {
		namespace = strings.TrimSpace(*response.Value)
	}
	if namespace == "" {
		return fmt.Errorf("lookup Bucket namespace: OCI returned empty namespace")
	}

	resource.Status.Namespace = namespace
	return nil
}
