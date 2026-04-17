/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dedicatedaicluster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type dedicatedAiClusterOCIClient interface {
	GetDedicatedAiCluster(context.Context, generativeaisdk.GetDedicatedAiClusterRequest) (generativeaisdk.GetDedicatedAiClusterResponse, error)
}

type dedicatedAiClusterGeneratedParityClient struct {
	delegate DedicatedAiClusterServiceClient
	client   dedicatedAiClusterOCIClient
	initErr  error
}

func init() {
	generatedFactory := newDedicatedAiClusterServiceClient
	newDedicatedAiClusterServiceClient = func(manager *DedicatedAiClusterServiceManager) DedicatedAiClusterServiceClient {
		delegate := generatedFactory(manager)
		sdkClient, err := generativeaisdk.NewGenerativeAiClientWithConfigurationProvider(manager.Provider)
		parityClient := &dedicatedAiClusterGeneratedParityClient{
			delegate: delegate,
			client:   sdkClient,
		}
		if err != nil {
			parityClient.initErr = fmt.Errorf("initialize DedicatedAiCluster OCI client: %w", err)
		}
		return parityClient
	}
}

func (c *dedicatedAiClusterGeneratedParityClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaiv1beta1.DedicatedAiCluster,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DedicatedAiCluster generated delegate is not configured")
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("DedicatedAiCluster resource must not be nil")
	}

	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}

	trackedID := currentDedicatedAiClusterID(resource)
	if trackedID == "" {
		return c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
	}

	if c.initErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.initErr
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isDedicatedAiClusterReadNotFoundOCI(err) {
			clearDedicatedAiClusterIdentity(resource)
			return c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
		}
		return servicemanager.OSOKResponse{IsSuccessful: false}, normalizeDedicatedAiClusterOCIError(err)
	}
	if current.LifecycleState == generativeaisdk.DedicatedAiClusterLifecycleStateDeleted {
		clearDedicatedAiClusterIdentity(resource)
		return c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
	}

	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *dedicatedAiClusterGeneratedParityClient) Delete(
	ctx context.Context,
	resource *generativeaiv1beta1.DedicatedAiCluster,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("DedicatedAiCluster generated delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *dedicatedAiClusterGeneratedParityClient) get(
	ctx context.Context,
	ocid string,
) (generativeaisdk.DedicatedAiCluster, error) {
	response, err := c.client.GetDedicatedAiCluster(ctx, generativeaisdk.GetDedicatedAiClusterRequest{
		DedicatedAiClusterId: common.String(ocid),
	})
	if err != nil {
		return generativeaisdk.DedicatedAiCluster{}, err
	}
	return response.DedicatedAiCluster, nil
}

func currentDedicatedAiClusterID(resource *generativeaiv1beta1.DedicatedAiCluster) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func clearDedicatedAiClusterIdentity(resource *generativeaiv1beta1.DedicatedAiCluster) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}

func normalizeDedicatedAiClusterOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isDedicatedAiClusterReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}
