/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
)

// FunctionsManagementClientInterface defines the OCI operations used by Functions service managers.
type FunctionsManagementClientInterface interface {
	CreateApplication(ctx context.Context, request ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error)
	GetApplication(ctx context.Context, request ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error)
	ListApplications(ctx context.Context, request ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error)
	UpdateApplication(ctx context.Context, request ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error)
	DeleteApplication(ctx context.Context, request ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error)
	CreateFunction(ctx context.Context, request ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error)
	GetFunction(ctx context.Context, request ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error)
	ListFunctions(ctx context.Context, request ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error)
	UpdateFunction(ctx context.Context, request ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error)
	DeleteFunction(ctx context.Context, request ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error)
}

func getFunctionsManagementClient(provider common.ConfigurationProvider) (ocifunctions.FunctionsManagementClient, error) {
	return ocifunctions.NewFunctionsManagementClientWithConfigurationProvider(provider)
}

func (m *FunctionsApplicationServiceManager) getOCIClient() (FunctionsManagementClientInterface, error) {
	if m.ociClient != nil {
		return m.ociClient, nil
	}
	return getFunctionsManagementClient(m.Provider)
}

func (m *FunctionsApplicationServiceManager) CreateApplication(
	ctx context.Context,
	resource *functionsv1beta1.Application,
) (ocifunctions.CreateApplicationResponse, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return ocifunctions.CreateApplicationResponse{}, err
	}

	details, err := buildFunctionsDetails[ocifunctions.CreateApplicationDetails](ctx, m.CredentialClient, resource)
	if err != nil {
		return ocifunctions.CreateApplicationResponse{}, fmt.Errorf("build Functions Application create details: %w", err)
	}

	response, err := client.CreateApplication(ctx, ocifunctions.CreateApplicationRequest{
		CreateApplicationDetails: details,
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return ocifunctions.CreateApplicationResponse{}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return response, nil
}

func (m *FunctionsApplicationServiceManager) GetApplication(
	ctx context.Context,
	applicationID shared.OCID,
	retryPolicy *common.RetryPolicy,
) (*ocifunctions.Application, error) {
	if strings.TrimSpace(string(applicationID)) == "" {
		return nil, fmt.Errorf("application id is required")
	}

	client, err := m.getOCIClient()
	if err != nil {
		return nil, err
	}

	request := ocifunctions.GetApplicationRequest{
		ApplicationId: common.String(string(applicationID)),
	}
	if retryPolicy != nil {
		request.RequestMetadata.RetryPolicy = retryPolicy
	}

	response, err := client.GetApplication(ctx, request)
	if err != nil {
		return nil, err
	}
	return &response.Application, nil
}

func (m *FunctionsApplicationServiceManager) FindApplication(
	ctx context.Context,
	resource *functionsv1beta1.Application,
) (*ocifunctions.Application, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return nil, err
	}

	response, err := client.ListApplications(ctx, ocifunctions.ListApplicationsRequest{
		CompartmentId: common.String(resource.Spec.CompartmentId),
		DisplayName:   common.String(resource.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		return nil, err
	}

	for _, item := range response.Items {
		if item.Id == nil || strings.EqualFold(string(item.LifecycleState), "DELETED") {
			continue
		}
		return m.GetApplication(ctx, shared.OCID(*item.Id), nil)
	}

	return nil, nil
}

func buildApplicationUpdateDetails(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *functionsv1beta1.Application,
	existing *ocifunctions.Application,
) (ocifunctions.UpdateApplicationDetails, bool, error) {
	details, err := buildFunctionsDetails[ocifunctions.UpdateApplicationDetails](ctx, credentialClient, resource)
	if err != nil {
		return ocifunctions.UpdateApplicationDetails{}, false, fmt.Errorf("build Functions Application update details: %w", err)
	}

	updateNeeded := trimUnchangedFunctionsDetails(&details, existing)
	return details, updateNeeded, nil
}

func (m *FunctionsApplicationServiceManager) UpdateApplication(
	ctx context.Context,
	resource *functionsv1beta1.Application,
	existing *ocifunctions.Application,
) (*ocifunctions.Application, bool, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return nil, false, err
	}

	details, updateNeeded, err := buildApplicationUpdateDetails(ctx, m.CredentialClient, resource, existing)
	if err != nil {
		return nil, false, err
	}
	if !updateNeeded {
		return existing, false, nil
	}

	applicationID := applicationStatusID(resource)
	if applicationID == "" && existing != nil && existing.Id != nil {
		applicationID = shared.OCID(*existing.Id)
	}
	if strings.TrimSpace(string(applicationID)) == "" {
		return nil, false, fmt.Errorf("application update requires a tracked application id")
	}

	response, err := client.UpdateApplication(ctx, ocifunctions.UpdateApplicationRequest{
		ApplicationId:            common.String(string(applicationID)),
		UpdateApplicationDetails: details,
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return nil, false, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	return &response.Application, true, nil
}

func (m *FunctionsApplicationServiceManager) DeleteApplication(ctx context.Context, resource *functionsv1beta1.Application, applicationID shared.OCID) error {
	if strings.TrimSpace(string(applicationID)) == "" {
		return nil
	}

	client, err := m.getOCIClient()
	if err != nil {
		return err
	}

	response, err := client.DeleteApplication(ctx, ocifunctions.DeleteApplicationRequest{
		ApplicationId: common.String(string(applicationID)),
	})
	if resource != nil {
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		} else {
			servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
		}
	}
	return err
}
