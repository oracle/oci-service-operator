/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package organization

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	tenantmanagercontrolplanesdk "github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane"
	tenantmanagercontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/tenantmanagercontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

var organizationWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(tenantmanagercontrolplanesdk.OperationStatusAccepted),
		string(tenantmanagercontrolplanesdk.OperationStatusInProgress),
		string(tenantmanagercontrolplanesdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(tenantmanagercontrolplanesdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(tenantmanagercontrolplanesdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(tenantmanagercontrolplanesdk.OperationStatusCanceled)},
	UpdateActionTokens:    []string{string(tenantmanagercontrolplanesdk.ActionTypeUpdated)},
}

type organizationOCIClient interface {
	GetOrganization(context.Context, tenantmanagercontrolplanesdk.GetOrganizationRequest) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error)
	ListOrganizations(context.Context, tenantmanagercontrolplanesdk.ListOrganizationsRequest) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error)
	UpdateOrganization(context.Context, tenantmanagercontrolplanesdk.UpdateOrganizationRequest) (tenantmanagercontrolplanesdk.UpdateOrganizationResponse, error)
}

type organizationWorkRequestClient interface {
	GetWorkRequest(context.Context, tenantmanagercontrolplanesdk.GetWorkRequestRequest) (tenantmanagercontrolplanesdk.GetWorkRequestResponse, error)
}

type organizationRuntimeClient struct {
	delegate OrganizationServiceClient
	get      func(context.Context, tenantmanagercontrolplanesdk.GetOrganizationRequest) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error)
}

type organizationListCall func(context.Context, tenantmanagercontrolplanesdk.ListOrganizationsRequest) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error)

func init() {
	registerOrganizationRuntimeHooksMutator(func(manager *OrganizationServiceManager, hooks *OrganizationRuntimeHooks) {
		organizationClient, workRequestClient, organizationInitErr, workRequestInitErr := newOrganizationRuntimeClients(manager)
		applyOrganizationRuntimeHooks(hooks, organizationClient, workRequestClient, organizationInitErr, workRequestInitErr)
	})
}

func newOrganizationRuntimeClients(
	manager *OrganizationServiceManager,
) (organizationOCIClient, organizationWorkRequestClient, error, error) {
	if manager == nil {
		err := fmt.Errorf("Organization service manager is nil")
		return nil, nil, err, err
	}

	organizationClient, organizationErr := tenantmanagercontrolplanesdk.NewOrganizationClientWithConfigurationProvider(manager.Provider)
	workRequestClient, workRequestErr := tenantmanagercontrolplanesdk.NewWorkRequestClientWithConfigurationProvider(manager.Provider)
	if organizationErr != nil {
		organizationClient = tenantmanagercontrolplanesdk.OrganizationClient{}
	}
	if workRequestErr != nil {
		workRequestClient = tenantmanagercontrolplanesdk.WorkRequestClient{}
	}
	return organizationClient, workRequestClient, organizationErr, workRequestErr
}

func applyOrganizationRuntimeHooks(
	hooks *OrganizationRuntimeHooks,
	organizationClient organizationOCIClient,
	workRequestClient organizationWorkRequestClient,
	organizationInitErr error,
	workRequestInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Get.Call = func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.GetOrganizationRequest,
	) (tenantmanagercontrolplanesdk.GetOrganizationResponse, error) {
		if err := organizationRuntimeClientReady(organizationClient, organizationInitErr); err != nil {
			return tenantmanagercontrolplanesdk.GetOrganizationResponse{}, err
		}
		return organizationClient.GetOrganization(ctx, request)
	}
	hooks.List.Fields = organizationListFields()
	hooks.List.Call = wrapOrganizationListCall(func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.ListOrganizationsRequest,
	) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error) {
		if err := organizationRuntimeClientReady(organizationClient, organizationInitErr); err != nil {
			return tenantmanagercontrolplanesdk.ListOrganizationsResponse{}, err
		}
		return organizationClient.ListOrganizations(ctx, request)
	})
	hooks.Update.Call = func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.UpdateOrganizationRequest,
	) (tenantmanagercontrolplanesdk.UpdateOrganizationResponse, error) {
		if err := organizationRuntimeClientReady(organizationClient, organizationInitErr); err != nil {
			return tenantmanagercontrolplanesdk.UpdateOrganizationResponse{}, err
		}
		return organizationClient.UpdateOrganization(ctx, request)
	}
	hooks.Async.Adapter = organizationWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getOrganizationWorkRequest(ctx, workRequestClient, workRequestInitErr, workRequestID)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OrganizationServiceClient) OrganizationServiceClient {
		return organizationRuntimeClient{
			delegate: delegate,
			get:      hooks.Get.Call,
		}
	})
}

func newOrganizationServiceClientWithClients(
	log loggerutil.OSOKLogger,
	organizationClient organizationOCIClient,
	workRequestClient organizationWorkRequestClient,
) OrganizationServiceClient {
	hooks := newOrganizationRuntimeHooksWithClients(organizationClient)
	applyOrganizationRuntimeHooks(&hooks, organizationClient, workRequestClient, nil, nil)
	delegate := defaultOrganizationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*tenantmanagercontrolplanev1beta1.Organization](
			buildOrganizationGeneratedRuntimeConfig(&OrganizationServiceManager{Log: log}, hooks),
		),
	}
	return wrapOrganizationGeneratedClient(hooks, delegate)
}

func newOrganizationRuntimeHooksWithClients(organizationClient organizationOCIClient) OrganizationRuntimeHooks {
	_ = organizationClient
	return newOrganizationDefaultRuntimeHooks(tenantmanagercontrolplanesdk.OrganizationClient{})
}

func organizationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func wrapOrganizationListCall(call organizationListCall) organizationListCall {
	if call == nil {
		return nil
	}

	return func(
		ctx context.Context,
		request tenantmanagercontrolplanesdk.ListOrganizationsRequest,
	) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error) {
		return listOrganizationPages(ctx, call, request)
	}
}

func listOrganizationPages(
	ctx context.Context,
	call organizationListCall,
	request tenantmanagercontrolplanesdk.ListOrganizationsRequest,
) (tenantmanagercontrolplanesdk.ListOrganizationsResponse, error) {
	if call == nil {
		return tenantmanagercontrolplanesdk.ListOrganizationsResponse{}, fmt.Errorf("Organization list call is not configured")
	}

	var combined tenantmanagercontrolplanesdk.ListOrganizationsResponse
	seenPages := map[string]struct{}{}
	for {
		pageToken := ""
		if request.Page != nil {
			pageToken = strings.TrimSpace(*request.Page)
		}
		if _, seen := seenPages[pageToken]; seen {
			return tenantmanagercontrolplanesdk.ListOrganizationsResponse{}, fmt.Errorf("Organization list pagination repeated page %q", pageToken)
		}
		seenPages[pageToken] = struct{}{}

		response, err := call(ctx, request)
		if err != nil {
			return tenantmanagercontrolplanesdk.ListOrganizationsResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}

		nextPage := strings.TrimSpace(*response.OpcNextPage)
		combined.OpcNextPage = common.String(nextPage)
		request.Page = common.String(nextPage)
	}
}

func getOrganizationWorkRequest(
	ctx context.Context,
	client organizationWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Organization OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Organization work request client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, tenantmanagercontrolplanesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func organizationRuntimeClientReady(client organizationOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize Organization OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("Organization OCI client is not configured")
	}
	return nil
}

func (c organizationRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *tenantmanagercontrolplanev1beta1.Organization,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Organization generated runtime delegate is not configured")
	}
	if err := validateOrganizationBinding(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	hadTrackedIdentity := trackedOrganizationID(resource) != ""
	explicitOrganizationID := strings.TrimSpace(resource.Spec.OrganizationId)
	originalSpecCompartmentID := resource.Spec.CompartmentId
	originalStatusCompartmentID := resource.Status.CompartmentId
	clearedStatusCompartmentID := false
	if !hadTrackedIdentity && explicitOrganizationID != "" {
		if c.get == nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Organization get hook is not configured")
		}
		if _, err := c.get(ctx, tenantmanagercontrolplanesdk.GetOrganizationRequest{
			OrganizationId: common.String(explicitOrganizationID),
		}); err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		resource.Status.OsokStatus.Ocid = shared.OCID(explicitOrganizationID)
		resource.Status.Id = explicitOrganizationID
	}
	if explicitOrganizationID != "" {
		resource.Spec.CompartmentId = ""
		if strings.TrimSpace(resource.Status.CompartmentId) != "" {
			resource.Status.CompartmentId = ""
			clearedStatusCompartmentID = true
		}
		defer func() {
			resource.Spec.CompartmentId = originalSpecCompartmentID
		}()
	}

	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil && clearedStatusCompartmentID {
		resource.Status.CompartmentId = originalStatusCompartmentID
	}
	if err != nil || hadTrackedIdentity || explicitOrganizationID != "" || trackedOrganizationID(resource) == "" {
		return response, err
	}

	// With Create=nil, the first untracked reconcile binds from ListOrganizations.
	// Run one live-ID-backed delegate pass immediately afterwards so status
	// projection and mutable-drift handling use GetOrganization instead of the
	// summary payload from the list response.
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c organizationRuntimeClient) Delete(
	_ context.Context,
	_ *tenantmanagercontrolplanev1beta1.Organization,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("Organization generated runtime delegate is not configured")
	}

	return true, nil
}

func validateOrganizationBinding(resource *tenantmanagercontrolplanev1beta1.Organization) error {
	if resource == nil {
		return fmt.Errorf("Organization resource is nil")
	}

	trackedID := trackedOrganizationID(resource)
	explicitOrganizationID := strings.TrimSpace(resource.Spec.OrganizationId)
	if trackedID != "" && explicitOrganizationID != "" && explicitOrganizationID != trackedID {
		return fmt.Errorf("Organization formal semantics require replacement when organizationId changes")
	}
	if trackedID != "" {
		currentCompartmentID := strings.TrimSpace(resource.Status.CompartmentId)
		desiredCompartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
		if currentCompartmentID != "" && desiredCompartmentID != "" && desiredCompartmentID != currentCompartmentID {
			return fmt.Errorf("Organization formal semantics require replacement when compartmentId changes")
		}
	}
	if trackedID != "" {
		return nil
	}
	if explicitOrganizationID != "" || strings.TrimSpace(resource.Spec.CompartmentId) != "" {
		return nil
	}

	return fmt.Errorf("Organization bind-existing flow requires spec.organizationId or spec.compartmentId")
}

func trackedOrganizationID(resource *tenantmanagercontrolplanev1beta1.Organization) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}
