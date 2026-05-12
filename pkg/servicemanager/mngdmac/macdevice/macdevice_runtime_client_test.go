/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package macdevice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	mngdmacsdk "github.com/oracle/oci-go-sdk/v65/mngdmac"
	mngdmacv1beta1 "github.com/oracle/oci-service-operator/api/mngdmac/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testMacDeviceID    = "example-mac-device-uuid"
	testMacOrderID     = "ocid1.macorder.oc1..exampleuniqueID"
	testMacCompartment = "ocid1.compartment.oc1..exampleuniqueID"
	testMacSerial      = "C02TESTMAC001"
	testMacIP          = "10.0.0.25"
)

type fakeMacDeviceOCIClient struct {
	getFn       func(context.Context, mngdmacsdk.GetMacDeviceRequest) (mngdmacsdk.GetMacDeviceResponse, error)
	listFn      func(context.Context, mngdmacsdk.ListMacDevicesRequest) (mngdmacsdk.ListMacDevicesResponse, error)
	terminateFn func(context.Context, mngdmacsdk.TerminateMacDeviceRequest) (mngdmacsdk.TerminateMacDeviceResponse, error)

	getRequests       []mngdmacsdk.GetMacDeviceRequest
	listRequests      []mngdmacsdk.ListMacDevicesRequest
	terminateRequests []mngdmacsdk.TerminateMacDeviceRequest
}

func (f *fakeMacDeviceOCIClient) GetMacDevice(
	ctx context.Context,
	request mngdmacsdk.GetMacDeviceRequest,
) (mngdmacsdk.GetMacDeviceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return mngdmacsdk.GetMacDeviceResponse{}, nil
}

func (f *fakeMacDeviceOCIClient) ListMacDevices(
	ctx context.Context,
	request mngdmacsdk.ListMacDevicesRequest,
) (mngdmacsdk.ListMacDevicesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return mngdmacsdk.ListMacDevicesResponse{}, nil
}

func (f *fakeMacDeviceOCIClient) TerminateMacDevice(
	ctx context.Context,
	request mngdmacsdk.TerminateMacDeviceRequest,
) (mngdmacsdk.TerminateMacDeviceResponse, error) {
	f.terminateRequests = append(f.terminateRequests, request)
	if f.terminateFn != nil {
		return f.terminateFn(ctx, request)
	}
	return mngdmacsdk.TerminateMacDeviceResponse{}, nil
}

type fakeMacDeviceWorkRequestClient struct {
	getFn func(context.Context, mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error)

	getRequests []mngdmacsdk.GetWorkRequestRequest
}

func (f *fakeMacDeviceWorkRequestClient) GetWorkRequest(
	ctx context.Context,
	request mngdmacsdk.GetWorkRequestRequest,
) (mngdmacsdk.GetWorkRequestResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return mngdmacsdk.GetWorkRequestResponse{}, nil
}

func TestMacDeviceRuntimeConfigHasNoCreateOrUpdateAndTracksDeleteWorkRequests(t *testing.T) {
	t.Parallel()

	hooks := newMacDeviceRuntimeHooksWithClients(&fakeMacDeviceOCIClient{})
	applyMacDeviceRuntimeHooks(&hooks, &fakeMacDeviceWorkRequestClient{}, nil)
	config := buildMacDeviceGeneratedRuntimeConfig(
		&MacDeviceServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}},
		hooks,
	)
	if config.Create != nil {
		t.Fatal("config.Create != nil, want bind-existing MacDevice runtime without Create")
	}
	if config.Update != nil {
		t.Fatal("config.Update != nil, want terminate-only MacDevice runtime without Update")
	}
	if config.Delete == nil {
		t.Fatal("config.Delete = nil, want TerminateMacDevice-backed delete")
	}
	if config.Async.GetWorkRequest == nil {
		t.Fatal("config.Async.GetWorkRequest = nil, want terminate work request tracking")
	}
	if config.Semantics == nil || config.Semantics.Async == nil {
		t.Fatalf("config.Semantics.Async = %#v, want delete-only workrequest semantics", config.Semantics)
	}
	if got := config.Semantics.Async.WorkRequest.Phases; len(got) != 1 || got[0] != "delete" {
		t.Fatalf("config.Semantics.Async.WorkRequest.Phases = %#v, want [delete]", got)
	}
	if len(config.List.Fields) != 4 {
		t.Fatalf("config.List.Fields = %#v, want focused macOrderId/id pagination fields", config.List.Fields)
	}
	if config.List.Fields[0].PreferResourceID {
		t.Fatal("config.List.Fields[0].PreferResourceID = true, want explicit macOrderId lookup")
	}
	if !config.List.Fields[1].PreferResourceID {
		t.Fatal("config.List.Fields[1].PreferResourceID = false, want list id query to reuse tracked/spec macDeviceId")
	}
}

func TestMacDeviceCreateOrUpdateRejectsMissingBindingIdentity(t *testing.T) {
	t.Parallel()

	resource := &mngdmacv1beta1.MacDevice{}
	response, err := newTestMacDeviceClient(&fakeMacDeviceOCIClient{}, &fakeMacDeviceWorkRequestClient{}).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err == nil || err.Error() != "MacDevice bind-existing flow requires spec.macOrderId plus spec.macDeviceId" {
		t.Fatalf("CreateOrUpdate() error = %v, want explicit bind-existing validation", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful validation failure", response)
	}
}

func TestMacDeviceCreateOrUpdateBindsExistingDeviceByExplicitIDs(t *testing.T) {
	t.Parallel()

	resource := newTestMacDeviceResource()
	deviceClient := &fakeMacDeviceOCIClient{
		getFn: func(_ context.Context, request mngdmacsdk.GetMacDeviceRequest) (mngdmacsdk.GetMacDeviceResponse, error) {
			requireStringPtr(t, "GetMacDeviceRequest.MacDeviceId", request.MacDeviceId, testMacDeviceID)
			requireStringPtr(t, "GetMacDeviceRequest.MacOrderId", request.MacOrderId, testMacOrderID)
			return mngdmacsdk.GetMacDeviceResponse{
				MacDevice: makeSDKMacDevice(mngdmacsdk.MacDeviceLifecycleStateActive),
			}, nil
		},
		listFn: func(context.Context, mngdmacsdk.ListMacDevicesRequest) (mngdmacsdk.ListMacDevicesResponse, error) {
			t.Fatal("ListMacDevices() should not run when direct GetMacDevice bind succeeds")
			return mngdmacsdk.ListMacDevicesResponse{}, nil
		},
	}

	response, err := newTestMacDeviceClient(deviceClient, &fakeMacDeviceWorkRequestClient{}).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want stable success without requeue", response)
	}
	if got := resource.Status.Id; got != testMacDeviceID {
		t.Fatalf("status.id = %q, want %q", got, testMacDeviceID)
	}
	if got := resource.Status.MacOrderId; got != testMacOrderID {
		t.Fatalf("status.macOrderId = %q, want %q", got, testMacOrderID)
	}
	if got := resource.Status.LifecycleState; got != string(mngdmacsdk.MacDeviceLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testMacDeviceID {
		t.Fatalf("status.ocid = %q, want %q", got, testMacDeviceID)
	}
	if resource.Status.OsokStatus.CreatedAt == nil {
		t.Fatal("status.createdAt = nil, want timestamp after successful bind")
	}
	if len(deviceClient.getRequests) != 1 {
		t.Fatalf("GetMacDevice() calls = %d, want 1", len(deviceClient.getRequests))
	}
	if len(deviceClient.listRequests) != 0 {
		t.Fatalf("ListMacDevices() calls = %d, want 0", len(deviceClient.listRequests))
	}
}

func TestMacDeviceCreateOrUpdateRejectsTrackedDeviceIDDift(t *testing.T) {
	t.Parallel()

	resource := newTestMacDeviceResource()
	resource.Status.Id = "example-mac-device-old"
	resource.Status.MacOrderId = testMacOrderID
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	resource.Spec.MacDeviceId = testMacDeviceID

	response, err := newTestMacDeviceClient(&fakeMacDeviceOCIClient{}, &fakeMacDeviceWorkRequestClient{}).CreateOrUpdate(
		context.Background(),
		resource,
		ctrl.Request{},
	)
	if err == nil || err.Error() != "MacDevice formal semantics require replacement when macDeviceId changes" {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-only macDeviceId drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
}

func TestMacDeviceDeleteUsesTerminateWorkRequestAndConfirmsDeletion(t *testing.T) {
	t.Parallel()

	const workRequestID = "wr-delete-macdevice"

	resource := newTestMacDeviceResource()
	resource.Status.Id = testMacDeviceID
	resource.Status.MacOrderId = testMacOrderID
	resource.Status.OsokStatus.Ocid = shared.OCID(testMacDeviceID)

	getCalls := 0
	deviceClient := &fakeMacDeviceOCIClient{
		getFn: func(_ context.Context, request mngdmacsdk.GetMacDeviceRequest) (mngdmacsdk.GetMacDeviceResponse, error) {
			getCalls++
			requireStringPtr(t, "GetMacDeviceRequest.MacDeviceId", request.MacDeviceId, testMacDeviceID)
			requireStringPtr(t, "GetMacDeviceRequest.MacOrderId", request.MacOrderId, testMacOrderID)
			state := mngdmacsdk.MacDeviceLifecycleStateActive
			if getCalls > 1 {
				state = mngdmacsdk.MacDeviceLifecycleStateDeleted
			}
			return mngdmacsdk.GetMacDeviceResponse{
				MacDevice: makeSDKMacDevice(state),
			}, nil
		},
		terminateFn: func(_ context.Context, request mngdmacsdk.TerminateMacDeviceRequest) (mngdmacsdk.TerminateMacDeviceResponse, error) {
			requireStringPtr(t, "TerminateMacDeviceRequest.MacDeviceId", request.MacDeviceId, testMacDeviceID)
			requireStringPtr(t, "TerminateMacDeviceRequest.MacOrderId", request.MacOrderId, testMacOrderID)
			return mngdmacsdk.TerminateMacDeviceResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
	}
	workRequestClient := &fakeMacDeviceWorkRequestClient{
		getFn: func(_ context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, workRequestID)
			return mngdmacsdk.GetWorkRequestResponse{
				WorkRequest: makeMacDeviceWorkRequest(workRequestID, mngdmacsdk.OperationStatusSucceeded, mngdmacsdk.ActionTypeDeleted, testMacDeviceID),
			}, nil
		},
	}

	deleted, err := newTestMacDeviceClient(deviceClient, workRequestClient).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after terminate work request confirmation")
	}
	if len(deviceClient.terminateRequests) != 1 {
		t.Fatalf("TerminateMacDevice() calls = %d, want 1", len(deviceClient.terminateRequests))
	}
	if len(workRequestClient.getRequests) != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", len(workRequestClient.getRequests))
	}
	if getCalls != 2 {
		t.Fatalf("GetMacDevice() calls = %d, want 2 (pre-delete check + post-workrequest confirm)", getCalls)
	}
	if got := resource.Status.LifecycleState; got != string(mngdmacsdk.MacDeviceLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want DELETED", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after delete", resource.Status.OsokStatus.Async.Current)
	}
}

func newTestMacDeviceClient(
	deviceClient macDeviceOCIClient,
	workRequestClient macDeviceWorkRequestClient,
) MacDeviceServiceClient {
	return newMacDeviceServiceClientWithClients(loggerutil.OSOKLogger{Logger: logr.Discard()}, deviceClient, workRequestClient)
}

func newTestMacDeviceResource() *mngdmacv1beta1.MacDevice {
	return &mngdmacv1beta1.MacDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "macdevice-sample",
			Namespace: "default",
		},
		Spec: mngdmacv1beta1.MacDeviceSpec{
			MacOrderId:  testMacOrderID,
			MacDeviceId: testMacDeviceID,
		},
	}
}

func makeSDKMacDevice(state mngdmacsdk.MacDeviceLifecycleStateEnum) mngdmacsdk.MacDevice {
	timeCreated := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	timeUpdated := common.SDKTime{Time: time.Unix(60, 0).UTC()}
	timeDecom := common.SDKTime{Time: time.Unix(120, 0).UTC()}
	return mngdmacsdk.MacDevice{
		Id:             common.String(testMacDeviceID),
		CompartmentId:  common.String(testMacCompartment),
		MacOrderId:     common.String(testMacOrderID),
		SerialNumber:   common.String(testMacSerial),
		IpAddress:      common.String(testMacIP),
		LifecycleState: state,
		Shape:          mngdmacsdk.MacOrderShapeM4ProMacMini64gb2tb,
		TimeCreated:    &timeCreated,
		TimeUpdated:    &timeUpdated,
		IsMarkedDecom:  common.Bool(state == mngdmacsdk.MacDeviceLifecycleStateDeleted),
		TimeDecom:      &timeDecom,
	}
}

func makeMacDeviceWorkRequest(
	workRequestID string,
	status mngdmacsdk.OperationStatusEnum,
	action mngdmacsdk.ActionTypeEnum,
	identifier string,
) mngdmacsdk.WorkRequest {
	resources := []mngdmacsdk.WorkRequestResource{}
	if identifier != "" {
		resources = append(resources, mngdmacsdk.WorkRequestResource{
			EntityType: common.String(macDeviceKind),
			ActionType: action,
			Identifier: common.String(identifier),
		})
	}
	timeAccepted := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(100)
	return mngdmacsdk.WorkRequest{
		Id:              common.String(workRequestID),
		OperationType:   mngdmacsdk.OperationTypeDeleteMacDevice,
		Status:          status,
		CompartmentId:   common.String(testMacCompartment),
		Resources:       resources,
		PercentComplete: &percentComplete,
		TimeAccepted:    &timeAccepted,
	}
}

func requireStringPtr(t *testing.T, field string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if got := strings.TrimSpace(*value); got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}
