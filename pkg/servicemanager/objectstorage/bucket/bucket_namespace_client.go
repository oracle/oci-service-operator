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
	newBucketServiceClient = func(manager *BucketServiceManager) BucketServiceClient {
		sdkClient, err := objectstoragesdk.NewObjectStorageClientWithConfigurationProvider(manager.Provider)
		config := generatedruntime.Config[*objectstoragev1beta1.Bucket]{
			Kind:    "Bucket",
			SDKName: "Bucket",
			Log:     manager.Log,
			Semantics: &generatedruntime.Semantics{
				FormalService:     "objectstorage",
				FormalSlug:        "objectstoragebucket",
				StatusProjection:  "required",
				SecretSideEffects: "none",
				FinalizerPolicy:   "retain-until-confirmed-delete",
				Lifecycle: generatedruntime.LifecycleSemantics{
					ProvisioningStates: []string{},
					UpdatingStates:     []string{},
					ActiveStates:       []string{"ACTIVE"},
				},
				Delete: generatedruntime.DeleteSemantics{
					Policy:         "required",
					PendingStates:  []string{},
					TerminalStates: []string{"DELETED"},
				},
				List: &generatedruntime.ListSemantics{
					ResponseItemsField: "Items",
					MatchFields:        []string{"compartmentId", "name", "namespace"},
				},
				Mutation: generatedruntime.MutationSemantics{
					UpdateCandidate: []string{"autoTiering", "compartmentId", "definedTags", "freeformTags", "kmsKeyId", "metadata", "name", "namespace", "objectEventsEnabled", "publicAccessType", "versioning"},
					Mutable:         []string{"autoTiering", "compartmentId", "definedTags", "freeformTags", "kmsKeyId", "metadata", "name", "namespace", "objectEventsEnabled", "publicAccessType", "versioning"},
					ForceNew:        []string{"storageTier"},
					ConflictsWith:   map[string][]string{},
				},
				Hooks: generatedruntime.HookSet{
					Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
					Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
					Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
				},
				CreateFollowUp: generatedruntime.FollowUpSemantics{
					Strategy: "read-after-write",
					Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
				},
				UpdateFollowUp: generatedruntime.FollowUpSemantics{
					Strategy: "read-after-write",
					Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
				},
				DeleteFollowUp: generatedruntime.FollowUpSemantics{
					Strategy: "confirm-delete",
					Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
				},
				AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
				Unsupported:         []generatedruntime.UnsupportedSemantic{},
			},
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.CreateBucketRequest{} },
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					{FieldName: "CreateBucketDetails", Contribution: "body"},
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.CreateBucket(ctx, *request.(*objectstoragesdk.CreateBucketRequest))
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.GetBucketRequest{} },
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					bucketNameField(),
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.GetBucket(ctx, *request.(*objectstoragesdk.GetBucketRequest))
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.ListBucketsRequest{} },
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.ListBuckets(ctx, *request.(*objectstoragesdk.ListBucketsRequest))
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.UpdateBucketRequest{} },
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					bucketNameField(),
					{FieldName: "UpdateBucketDetails", Contribution: "body"},
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.UpdateBucket(ctx, *request.(*objectstoragesdk.UpdateBucketRequest))
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &objectstoragesdk.DeleteBucketRequest{} },
				Fields: []generatedruntime.RequestField{
					bucketNamespaceField(),
					bucketNameField(),
				},
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.DeleteBucket(ctx, *request.(*objectstoragesdk.DeleteBucketRequest))
				},
			},
		}
		if err != nil {
			config.InitError = fmt.Errorf("initialize Bucket OCI client: %w", err)
		}
		delegate := defaultBucketServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*objectstoragev1beta1.Bucket](config),
		}

		return &namespaceResolvingBucketServiceClient{
			delegate:        delegate,
			namespaceGetter: sdkClient,
		}
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
