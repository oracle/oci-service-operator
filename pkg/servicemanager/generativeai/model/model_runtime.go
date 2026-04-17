/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package model

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

type modelOCIClient interface {
	GetModel(context.Context, generativeaisdk.GetModelRequest) (generativeaisdk.GetModelResponse, error)
}

type modelGeneratedParityClient struct {
	delegate ModelServiceClient
	client   modelOCIClient
	initErr  error
}

func init() {
	generatedFactory := newModelServiceClient
	newModelServiceClient = func(manager *ModelServiceManager) ModelServiceClient {
		delegate := generatedFactory(manager)
		sdkClient, err := generativeaisdk.NewGenerativeAiClientWithConfigurationProvider(manager.Provider)
		parityClient := &modelGeneratedParityClient{
			delegate: delegate,
			client:   sdkClient,
		}
		if err != nil {
			parityClient.initErr = fmt.Errorf("initialize Model OCI client: %w", err)
		}
		return parityClient
	}
}

func (c *modelGeneratedParityClient) CreateOrUpdate(
	ctx context.Context,
	resource *generativeaiv1beta1.Model,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Model generated delegate is not configured")
	}
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Model resource must not be nil")
	}

	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return c.delegate.CreateOrUpdate(ctx, resource, req)
	}

	trackedID := currentModelID(resource)
	if trackedID == "" {
		return c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
	}

	if c.initErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.initErr
	}

	current, err := c.get(ctx, trackedID)
	if err != nil {
		if isModelReadNotFoundOCI(err) {
			clearModelIdentity(resource)
			return c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
		}
		return servicemanager.OSOKResponse{IsSuccessful: false}, normalizeModelOCIError(err)
	}
	if current.LifecycleState == generativeaisdk.ModelLifecycleStateDeleted {
		clearModelIdentity(resource)
		return c.delegate.CreateOrUpdate(generatedruntime.WithSkipExistingBeforeCreate(ctx), resource, req)
	}

	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *modelGeneratedParityClient) Delete(
	ctx context.Context,
	resource *generativeaiv1beta1.Model,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("Model generated delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *modelGeneratedParityClient) get(ctx context.Context, ocid string) (generativeaisdk.Model, error) {
	response, err := c.client.GetModel(ctx, generativeaisdk.GetModelRequest{
		ModelId: common.String(ocid),
	})
	if err != nil {
		return generativeaisdk.Model{}, err
	}
	return response.Model, nil
}

func currentModelID(resource *generativeaiv1beta1.Model) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func clearModelIdentity(resource *generativeaiv1beta1.Model) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}

func normalizeModelOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isModelReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}
