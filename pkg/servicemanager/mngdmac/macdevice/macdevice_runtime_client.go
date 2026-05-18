/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package macdevice

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	mngdmacsdk "github.com/oracle/oci-go-sdk/v65/mngdmac"
	mngdmacv1beta1 "github.com/oracle/oci-service-operator/api/mngdmac/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const macDeviceKind = "MacDevice"

type macDeviceOCIClient interface {
	GetMacDevice(context.Context, mngdmacsdk.GetMacDeviceRequest) (mngdmacsdk.GetMacDeviceResponse, error)
	ListMacDevices(context.Context, mngdmacsdk.ListMacDevicesRequest) (mngdmacsdk.ListMacDevicesResponse, error)
	TerminateMacDevice(context.Context, mngdmacsdk.TerminateMacDeviceRequest) (mngdmacsdk.TerminateMacDeviceResponse, error)
}

type macDeviceWorkRequestClient interface {
	GetWorkRequest(context.Context, mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error)
}

type macDeviceListCall func(context.Context, mngdmacsdk.ListMacDevicesRequest) (mngdmacsdk.ListMacDevicesResponse, error)

type macDeviceRuntimeClient struct {
	delegate MacDeviceServiceClient
}

var macDeviceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(mngdmacsdk.OperationStatusAccepted),
		string(mngdmacsdk.OperationStatusInProgress),
		string(mngdmacsdk.OperationStatusWaiting),
		string(mngdmacsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(mngdmacsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(mngdmacsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(mngdmacsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(mngdmacsdk.OperationStatusNeedsAttention)},
	DeleteActionTokens: []string{
		string(mngdmacsdk.OperationTypeDeleteMacDevice),
		string(mngdmacsdk.ActionTypeDeleted),
	},
}

func init() {
	registerMacDeviceRuntimeHooksMutator(func(manager *MacDeviceServiceManager, hooks *MacDeviceRuntimeHooks) {
		_, workRequestClient, _, workRequestInitErr := newMacDeviceRuntimeClients(manager)
		applyMacDeviceRuntimeHooks(hooks, workRequestClient, workRequestInitErr)
	})
}

func newMacDeviceRuntimeClients(
	manager *MacDeviceServiceManager,
) (macDeviceOCIClient, macDeviceWorkRequestClient, error, error) {
	if manager == nil {
		err := fmt.Errorf("%s service manager is nil", macDeviceKind)
		return nil, nil, err, err
	}

	deviceClient, deviceErr := mngdmacsdk.NewMacDeviceClientWithConfigurationProvider(manager.Provider)
	workRequestClient, workRequestErr := mngdmacsdk.NewMacOrderClientWithConfigurationProvider(manager.Provider)
	if deviceErr != nil {
		deviceClient = mngdmacsdk.MacDeviceClient{}
	}
	if workRequestErr != nil {
		workRequestClient = mngdmacsdk.MacOrderClient{}
	}
	return deviceClient, workRequestClient, deviceErr, workRequestErr
}

func newMacDeviceRuntimeHooksWithClients(deviceClient macDeviceOCIClient) MacDeviceRuntimeHooks {
	return MacDeviceRuntimeHooks{
		Semantics:       newMacDeviceRuntimeSemantics(),
		Identity:        generatedruntime.IdentityHooks[*mngdmacv1beta1.MacDevice]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*mngdmacv1beta1.MacDevice]{},
		StatusHooks:     generatedruntime.StatusHooks[*mngdmacv1beta1.MacDevice]{},
		ParityHooks:     generatedruntime.ParityHooks[*mngdmacv1beta1.MacDevice]{},
		Async:           generatedruntime.AsyncHooks[*mngdmacv1beta1.MacDevice]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*mngdmacv1beta1.MacDevice]{},
		Get: runtimeOperationHooks[mngdmacsdk.GetMacDeviceRequest, mngdmacsdk.GetMacDeviceResponse]{
			Fields: macDeviceGetFields(),
			Call: func(ctx context.Context, request mngdmacsdk.GetMacDeviceRequest) (mngdmacsdk.GetMacDeviceResponse, error) {
				if deviceClient == nil {
					return mngdmacsdk.GetMacDeviceResponse{}, fmt.Errorf("%s OCI client is not configured", macDeviceKind)
				}
				return deviceClient.GetMacDevice(ctx, request)
			},
		},
		List: runtimeOperationHooks[mngdmacsdk.ListMacDevicesRequest, mngdmacsdk.ListMacDevicesResponse]{
			Fields: macDeviceListFields(),
			Call: func(ctx context.Context, request mngdmacsdk.ListMacDevicesRequest) (mngdmacsdk.ListMacDevicesResponse, error) {
				if deviceClient == nil {
					return mngdmacsdk.ListMacDevicesResponse{}, fmt.Errorf("%s OCI client is not configured", macDeviceKind)
				}
				return deviceClient.ListMacDevices(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[mngdmacsdk.TerminateMacDeviceRequest, mngdmacsdk.TerminateMacDeviceResponse]{
			Fields: macDeviceDeleteFields(),
			Call: func(ctx context.Context, request mngdmacsdk.TerminateMacDeviceRequest) (mngdmacsdk.TerminateMacDeviceResponse, error) {
				if deviceClient == nil {
					return mngdmacsdk.TerminateMacDeviceResponse{}, fmt.Errorf("%s OCI client is not configured", macDeviceKind)
				}
				return deviceClient.TerminateMacDevice(ctx, request)
			},
		},
		WrapGeneratedClient: []func(MacDeviceServiceClient) MacDeviceServiceClient{},
	}
}

func newMacDeviceServiceClientWithClients(
	log loggerutil.OSOKLogger,
	deviceClient macDeviceOCIClient,
	workRequestClient macDeviceWorkRequestClient,
) MacDeviceServiceClient {
	hooks := newMacDeviceRuntimeHooksWithClients(deviceClient)
	applyMacDeviceRuntimeHooks(&hooks, workRequestClient, nil)
	delegate := defaultMacDeviceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*mngdmacv1beta1.MacDevice](
			buildMacDeviceGeneratedRuntimeConfig(&MacDeviceServiceManager{Log: log}, hooks),
		),
	}
	return wrapMacDeviceGeneratedClient(hooks, delegate)
}

func applyMacDeviceRuntimeHooks(
	hooks *MacDeviceRuntimeHooks,
	workRequestClient macDeviceWorkRequestClient,
	workRequestInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Get.Fields = macDeviceGetFields()
	hooks.List.Fields = macDeviceListFields()
	hooks.Delete.Fields = macDeviceDeleteFields()
	hooks.List.Call = listMacDevicesAllPages(hooks.List.Call)
	hooks.Async.Adapter = macDeviceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getMacDeviceWorkRequest(ctx, workRequestClient, workRequestInitErr, workRequestID)
	}
	hooks.Async.RecoverResourceID = recoverMacDeviceIDFromWorkRequest
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate MacDeviceServiceClient) MacDeviceServiceClient {
		return macDeviceRuntimeClient{delegate: delegate}
	})
}

func macDeviceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "MacDeviceId",
			RequestName:      "macDeviceId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "spec.macDeviceId", "macDeviceId"},
		},
		{
			FieldName:    "MacOrderId",
			RequestName:  "macOrderId",
			Contribution: "path",
			LookupPaths:  []string{"status.macOrderId", "spec.macOrderId", "macOrderId"},
		},
	}
}

func macDeviceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "MacOrderId",
			RequestName:  "macOrderId",
			Contribution: "path",
			LookupPaths:  []string{"status.macOrderId", "spec.macOrderId", "macOrderId"},
		},
		{
			FieldName:        "Id",
			RequestName:      "id",
			Contribution:     "query",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "spec.macDeviceId", "macDeviceId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func macDeviceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:        "MacDeviceId",
			RequestName:      "macDeviceId",
			Contribution:     "path",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "spec.macDeviceId", "macDeviceId"},
		},
		{
			FieldName:    "MacOrderId",
			RequestName:  "macOrderId",
			Contribution: "path",
			LookupPaths:  []string{"status.macOrderId", "spec.macOrderId", "macOrderId"},
		},
	}
}

func listMacDevicesAllPages(call macDeviceListCall) macDeviceListCall {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request mngdmacsdk.ListMacDevicesRequest) (mngdmacsdk.ListMacDevicesResponse, error) {
		var combined mngdmacsdk.ListMacDevicesResponse
		seenPages := map[string]struct{}{}
		for {
			pageToken := ""
			if request.Page != nil {
				pageToken = strings.TrimSpace(*request.Page)
			}
			if _, seen := seenPages[pageToken]; seen {
				return mngdmacsdk.ListMacDevicesResponse{}, fmt.Errorf("%s list pagination repeated page %q", macDeviceKind, pageToken)
			}
			seenPages[pageToken] = struct{}{}

			response, err := call(ctx, request)
			if err != nil {
				return mngdmacsdk.ListMacDevicesResponse{}, err
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
}

func getMacDeviceWorkRequest(
	ctx context.Context,
	client macDeviceWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s work request client: %w", macDeviceKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s work request client is not configured", macDeviceKind)
	}

	response, err := client.GetWorkRequest(ctx, mngdmacsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func recoverMacDeviceIDFromWorkRequest(
	resource *mngdmacv1beta1.MacDevice,
	workRequest any,
	_ shared.OSOKAsyncPhase,
) (string, error) {
	if id := trackedMacDeviceID(resource); id != "" {
		return id, nil
	}
	for _, item := range macDeviceWorkRequestResources(workRequest) {
		if !strings.EqualFold(strings.TrimSpace(stringPtrValue(item.EntityType)), macDeviceKind) {
			continue
		}
		if id := strings.TrimSpace(stringPtrValue(item.Identifier)); id != "" {
			return id, nil
		}
	}
	if resource == nil {
		return "", nil
	}
	return strings.TrimSpace(resource.Spec.MacDeviceId), nil
}

func macDeviceWorkRequestResources(workRequest any) []mngdmacsdk.WorkRequestResource {
	switch typed := workRequest.(type) {
	case mngdmacsdk.WorkRequest:
		return typed.Resources
	case *mngdmacsdk.WorkRequest:
		if typed != nil {
			return typed.Resources
		}
	}
	return nil
}

func (c macDeviceRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *mngdmacv1beta1.MacDevice,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s generated runtime delegate is not configured", macDeviceKind)
	}
	if err := validateMacDeviceBinding(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if err := validateTrackedMacDeviceDrift(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c macDeviceRuntimeClient) Delete(ctx context.Context, resource *mngdmacv1beta1.MacDevice) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("%s generated runtime delegate is not configured", macDeviceKind)
	}
	return c.delegate.Delete(ctx, resource)
}

func validateMacDeviceBinding(resource *mngdmacv1beta1.MacDevice) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", macDeviceKind)
	}
	if strings.TrimSpace(resource.Spec.MacOrderId) == "" || strings.TrimSpace(resource.Spec.MacDeviceId) == "" {
		return fmt.Errorf("%s bind-existing flow requires spec.macOrderId plus spec.macDeviceId", macDeviceKind)
	}
	return nil
}

func validateTrackedMacDeviceDrift(resource *mngdmacv1beta1.MacDevice) error {
	if resource == nil {
		return nil
	}
	if trackedID := trackedMacDeviceID(resource); trackedID != "" && strings.TrimSpace(resource.Spec.MacDeviceId) != "" && trackedID != strings.TrimSpace(resource.Spec.MacDeviceId) {
		return fmt.Errorf("%s formal semantics require replacement when macDeviceId changes", macDeviceKind)
	}
	if trackedOrderID := strings.TrimSpace(resource.Status.MacOrderId); trackedOrderID != "" && strings.TrimSpace(resource.Spec.MacOrderId) != "" && trackedOrderID != strings.TrimSpace(resource.Spec.MacOrderId) {
		return fmt.Errorf("%s formal semantics require replacement when macOrderId changes", macDeviceKind)
	}
	return nil
}

func trackedMacDeviceID(resource *mngdmacv1beta1.MacDevice) string {
	if resource == nil {
		return ""
	}
	if value := strings.TrimSpace(resource.Status.Id); value != "" {
		return value
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
