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
	"github.com/oracle/oci-service-operator/pkg/shared"
)

func (m *FunctionsFunctionServiceManager) getOCIClient() (FunctionsManagementClientInterface, error) {
	if m.ociClient != nil {
		return m.ociClient, nil
	}
	return getFunctionsManagementClient(m.Provider)
}

func (m *FunctionsFunctionServiceManager) CreateFunction(
	ctx context.Context,
	resource *functionsv1beta1.Function,
) (ocifunctions.CreateFunctionResponse, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return ocifunctions.CreateFunctionResponse{}, err
	}

	details, err := buildFunctionsDetails[ocifunctions.CreateFunctionDetails](ctx, m.CredentialClient, resource)
	if err != nil {
		return ocifunctions.CreateFunctionResponse{}, fmt.Errorf("build Functions Function create details: %w", err)
	}

	return client.CreateFunction(ctx, ocifunctions.CreateFunctionRequest{
		CreateFunctionDetails: details,
	})
}

func (m *FunctionsFunctionServiceManager) GetFunction(
	ctx context.Context,
	functionID shared.OCID,
	retryPolicy *common.RetryPolicy,
) (*ocifunctions.Function, error) {
	if strings.TrimSpace(string(functionID)) == "" {
		return nil, fmt.Errorf("function id is required")
	}

	client, err := m.getOCIClient()
	if err != nil {
		return nil, err
	}

	request := ocifunctions.GetFunctionRequest{
		FunctionId: common.String(string(functionID)),
	}
	if retryPolicy != nil {
		request.RequestMetadata.RetryPolicy = retryPolicy
	}

	response, err := client.GetFunction(ctx, request)
	if err != nil {
		return nil, err
	}
	return &response.Function, nil
}

func (m *FunctionsFunctionServiceManager) FindFunction(
	ctx context.Context,
	resource *functionsv1beta1.Function,
) (*ocifunctions.Function, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return nil, err
	}

	response, err := client.ListFunctions(ctx, ocifunctions.ListFunctionsRequest{
		ApplicationId: common.String(resource.Spec.ApplicationId),
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
		return m.GetFunction(ctx, shared.OCID(*item.Id), nil)
	}

	return nil, nil
}

func buildFunctionUpdateDetails(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *functionsv1beta1.Function,
	existing *ocifunctions.Function,
) (ocifunctions.UpdateFunctionDetails, bool, error) {
	details, err := buildFunctionsDetails[ocifunctions.UpdateFunctionDetails](ctx, credentialClient, resource)
	if err != nil {
		return ocifunctions.UpdateFunctionDetails{}, false, fmt.Errorf("build Functions Function update details: %w", err)
	}

	updateNeeded := trimUnchangedFunctionsDetails(&details, existing)
	return details, updateNeeded, nil
}

func (m *FunctionsFunctionServiceManager) UpdateFunction(
	ctx context.Context,
	resource *functionsv1beta1.Function,
	existing *ocifunctions.Function,
) (*ocifunctions.Function, bool, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return nil, false, err
	}

	details, updateNeeded, err := buildFunctionUpdateDetails(ctx, m.CredentialClient, resource, existing)
	if err != nil {
		return nil, false, err
	}
	if !updateNeeded {
		return existing, false, nil
	}

	functionID := functionStatusID(resource)
	if functionID == "" && existing != nil && existing.Id != nil {
		functionID = shared.OCID(*existing.Id)
	}
	if strings.TrimSpace(string(functionID)) == "" {
		return nil, false, fmt.Errorf("function update requires a tracked function id")
	}

	response, err := client.UpdateFunction(ctx, ocifunctions.UpdateFunctionRequest{
		FunctionId:            common.String(string(functionID)),
		UpdateFunctionDetails: details,
	})
	if err != nil {
		return nil, false, err
	}
	return &response.Function, true, nil
}

func (m *FunctionsFunctionServiceManager) DeleteFunction(ctx context.Context, functionID shared.OCID) error {
	if strings.TrimSpace(string(functionID)) == "" {
		return nil
	}

	client, err := m.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteFunction(ctx, ocifunctions.DeleteFunctionRequest{
		FunctionId: common.String(string(functionID)),
	})
	return err
}
