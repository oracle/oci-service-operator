/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package serviceenvironment

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	servicemanagerproxysdk "github.com/oracle/oci-go-sdk/v65/servicemanagerproxy"
	servicemanagerproxyv1beta1 "github.com/oracle/oci-service-operator/api/servicemanagerproxy/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testServiceEnvironmentID = "example-service-environment-id"
	testCompartmentID        = "ocid1.compartment.oc1..exampleuniqueID"
)

type fakeServiceEnvironmentOCIClient struct {
	getFn  func(context.Context, servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error)
	listFn func(context.Context, servicemanagerproxysdk.ListServiceEnvironmentsRequest) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error)

	getRequests  []servicemanagerproxysdk.GetServiceEnvironmentRequest
	listRequests []servicemanagerproxysdk.ListServiceEnvironmentsRequest
}

func (f *fakeServiceEnvironmentOCIClient) GetServiceEnvironment(
	ctx context.Context,
	request servicemanagerproxysdk.GetServiceEnvironmentRequest,
) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return servicemanagerproxysdk.GetServiceEnvironmentResponse{}, nil
}

func (f *fakeServiceEnvironmentOCIClient) ListServiceEnvironments(
	ctx context.Context,
	request servicemanagerproxysdk.ListServiceEnvironmentsRequest,
) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return servicemanagerproxysdk.ListServiceEnvironmentsResponse{}, nil
}

func TestServiceEnvironmentCreateOrUpdateProjectsStableObservedState(t *testing.T) {
	t.Parallel()

	resource := serviceEnvironmentResource()
	client := &fakeServiceEnvironmentOCIClient{
		getFn: func(_ context.Context, request servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error) {
			requireStringPtr(t, "get serviceEnvironmentId", request.ServiceEnvironmentId, testServiceEnvironmentID)
			requireStringPtr(t, "get compartmentId", request.CompartmentId, testCompartmentID)
			return servicemanagerproxysdk.GetServiceEnvironmentResponse{
				ServiceEnvironment: sdkServiceEnvironment(servicemanagerproxysdk.ServiceEntitlementRegistrationStatusDisabled),
			}, nil
		},
		listFn: func(_ context.Context, request servicemanagerproxysdk.ListServiceEnvironmentsRequest) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error) {
			t.Fatalf("ListServiceEnvironments(%#v) should not be called when GetServiceEnvironment succeeds", request)
			return servicemanagerproxysdk.ListServiceEnvironmentsResponse{}, nil
		},
	}

	response, err := newServiceEnvironmentServiceClientWithOCIClient(testServiceEnvironmentLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want stable success without requeue", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testServiceEnvironmentID {
		t.Fatalf("status.ocid = %q, want %q", got, testServiceEnvironmentID)
	}
	if got := resource.Status.Status; got != string(servicemanagerproxysdk.ServiceEntitlementRegistrationStatusDisabled) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, servicemanagerproxysdk.ServiceEntitlementRegistrationStatusDisabled)
	}
	if got := resource.Status.OsokStatus.Reason; got != "Active" {
		t.Fatalf("status.reason = %q, want Active", got)
	}
	if got := lastConditionType(t, resource); got != "Active" {
		t.Fatalf("last condition type = %q, want Active", got)
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "DISABLED") {
		t.Fatalf("status.message = %q, want DISABLED detail", got)
	}
	if resource.Status.OsokStatus.CreatedAt == nil {
		t.Fatal("status.createdAt = nil, want timestamp after successful bind")
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("GetServiceEnvironment() calls = %d, want 1", len(client.getRequests))
	}
	if len(client.listRequests) != 0 {
		t.Fatalf("ListServiceEnvironments() calls = %d, want 0", len(client.listRequests))
	}
}

func TestServiceEnvironmentCreateOrUpdateRequeuesProvisioningState(t *testing.T) {
	t.Parallel()

	resource := serviceEnvironmentResource()
	client := &fakeServiceEnvironmentOCIClient{
		getFn: func(_ context.Context, request servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error) {
			requireStringPtr(t, "get serviceEnvironmentId", request.ServiceEnvironmentId, testServiceEnvironmentID)
			requireStringPtr(t, "get compartmentId", request.CompartmentId, testCompartmentID)
			return servicemanagerproxysdk.GetServiceEnvironmentResponse{
				ServiceEnvironment: sdkServiceEnvironment(servicemanagerproxysdk.ServiceEntitlementRegistrationStatusBeginActivation),
			}, nil
		},
	}

	response, err := newServiceEnvironmentServiceClientWithOCIClient(testServiceEnvironmentLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue during BEGIN_ACTIVATION", response)
	}
	if got := resource.Status.OsokStatus.Reason; got != "Provisioning" {
		t.Fatalf("status.reason = %q, want Provisioning", got)
	}
	if got := lastConditionType(t, resource); got != "Provisioning" {
		t.Fatalf("last condition type = %q, want Provisioning", got)
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "BEGIN_ACTIVATION") {
		t.Fatalf("status.message = %q, want BEGIN_ACTIVATION detail", got)
	}
}

func TestServiceEnvironmentCreateOrUpdateMarksFailedTerminalState(t *testing.T) {
	t.Parallel()

	resource := serviceEnvironmentResource()
	client := &fakeServiceEnvironmentOCIClient{
		getFn: func(_ context.Context, request servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error) {
			requireStringPtr(t, "get serviceEnvironmentId", request.ServiceEnvironmentId, testServiceEnvironmentID)
			requireStringPtr(t, "get compartmentId", request.CompartmentId, testCompartmentID)
			return servicemanagerproxysdk.GetServiceEnvironmentResponse{
				ServiceEnvironment: sdkServiceEnvironment(servicemanagerproxysdk.ServiceEntitlementRegistrationStatusFailedActivation),
			}, nil
		},
	}

	response, err := newServiceEnvironmentServiceClientWithOCIClient(testServiceEnvironmentLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want nil with projected failed condition", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want terminal unsuccessful response without requeue", response)
	}
	if got := resource.Status.OsokStatus.Reason; got != "Failed" {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
	if got := lastConditionType(t, resource); got != "Failed" {
		t.Fatalf("last condition type = %q, want Failed", got)
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "FAILED_ACTIVATION") {
		t.Fatalf("status.message = %q, want FAILED_ACTIVATION detail", got)
	}
}

func TestServiceEnvironmentCreateOrUpdateUsesListForMissingGetConfirmationOnly(t *testing.T) {
	t.Parallel()

	resource := serviceEnvironmentResource()
	client := &fakeServiceEnvironmentOCIClient{
		getFn: func(_ context.Context, request servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error) {
			requireStringPtr(t, "get serviceEnvironmentId", request.ServiceEnvironmentId, testServiceEnvironmentID)
			requireStringPtr(t, "get compartmentId", request.CompartmentId, testCompartmentID)
			return servicemanagerproxysdk.GetServiceEnvironmentResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
		},
		listFn: func(_ context.Context, request servicemanagerproxysdk.ListServiceEnvironmentsRequest) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error) {
			requireStringPtr(t, "list serviceEnvironmentId", request.ServiceEnvironmentId, testServiceEnvironmentID)
			requireStringPtr(t, "list compartmentId", request.CompartmentId, testCompartmentID)
			return servicemanagerproxysdk.ListServiceEnvironmentsResponse{
				ServiceEnvironmentCollection: servicemanagerproxysdk.ServiceEnvironmentCollection{
					Items: []servicemanagerproxysdk.ServiceEnvironmentSummary{
						sdkServiceEnvironmentSummary(servicemanagerproxysdk.ServiceEntitlementRegistrationStatusActive),
					},
				},
			}, nil
		},
	}

	response, err := newServiceEnvironmentServiceClientWithOCIClient(testServiceEnvironmentLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "still finds a matching environment") {
		t.Fatalf("CreateOrUpdate() error = %v, want list-confirmed mismatch failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response after get/list mismatch", response)
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("GetServiceEnvironment() calls = %d, want 1", len(client.getRequests))
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("ListServiceEnvironments() calls = %d, want 1", len(client.listRequests))
	}
}

func TestServiceEnvironmentDeleteIsKubernetesLocalOnly(t *testing.T) {
	t.Parallel()

	resource := serviceEnvironmentResource()
	resource.Status.Id = testServiceEnvironmentID
	resource.Status.ServiceDefinition.DisplayName = "Retail Order Management"

	client := &fakeServiceEnvironmentOCIClient{
		getFn: func(_ context.Context, request servicemanagerproxysdk.GetServiceEnvironmentRequest) (servicemanagerproxysdk.GetServiceEnvironmentResponse, error) {
			t.Fatalf("GetServiceEnvironment(%#v) should not be called during Kubernetes-local delete", request)
			return servicemanagerproxysdk.GetServiceEnvironmentResponse{}, nil
		},
		listFn: func(_ context.Context, request servicemanagerproxysdk.ListServiceEnvironmentsRequest) (servicemanagerproxysdk.ListServiceEnvironmentsResponse, error) {
			t.Fatalf("ListServiceEnvironments(%#v) should not be called during Kubernetes-local delete", request)
			return servicemanagerproxysdk.ListServiceEnvironmentsResponse{}, nil
		},
	}

	deleted, err := newServiceEnvironmentServiceClientWithOCIClient(testServiceEnvironmentLogger(), client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for local finalizer cleanup")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want timestamp after local delete")
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "released from Kubernetes control") {
		t.Fatalf("status.message = %q, want local delete note", got)
	}
	if got := lastConditionType(t, resource); got != "Terminating" {
		t.Fatalf("last condition type = %q, want Terminating", got)
	}
}

func serviceEnvironmentResource() *servicemanagerproxyv1beta1.ServiceEnvironment {
	return &servicemanagerproxyv1beta1.ServiceEnvironment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "serviceenvironment-sample",
			Namespace: "default",
		},
		Spec: servicemanagerproxyv1beta1.ServiceEnvironmentSpec{
			CompartmentId:        testCompartmentID,
			ServiceEnvironmentId: testServiceEnvironmentID,
		},
	}
}

func sdkServiceEnvironment(
	state servicemanagerproxysdk.ServiceEntitlementRegistrationStatusEnum,
) servicemanagerproxysdk.ServiceEnvironment {
	return servicemanagerproxysdk.ServiceEnvironment{
		Id:             common.String(testServiceEnvironmentID),
		SubscriptionId: common.String("example-subscription-id"),
		Status:         state,
		CompartmentId:  common.String(testCompartmentID),
		ServiceDefinition: &servicemanagerproxysdk.ServiceDefinition{
			Type:             common.String("RGBUOROMS"),
			DisplayName:      common.String("Oracle Retail Order Management Cloud Service"),
			ShortDisplayName: common.String("Retail Order Management"),
		},
		ConsoleUrl: common.String("https://example.oraclecloud.com"),
	}
}

func sdkServiceEnvironmentSummary(
	state servicemanagerproxysdk.ServiceEntitlementRegistrationStatusEnum,
) servicemanagerproxysdk.ServiceEnvironmentSummary {
	return servicemanagerproxysdk.ServiceEnvironmentSummary{
		Id:             common.String(testServiceEnvironmentID),
		SubscriptionId: common.String("example-subscription-id"),
		Status:         state,
		CompartmentId:  common.String(testCompartmentID),
		ServiceDefinition: &servicemanagerproxysdk.ServiceDefinition{
			Type:             common.String("RGBUOROMS"),
			DisplayName:      common.String("Oracle Retail Order Management Cloud Service"),
			ShortDisplayName: common.String("Retail Order Management"),
		},
		ConsoleUrl: common.String("https://example.oraclecloud.com"),
	}
}

func testServiceEnvironmentLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("serviceenvironment-runtime-test")}
}

func requireStringPtr(t *testing.T, label string, value *string, want string) {
	t.Helper()
	if value == nil || *value != want {
		t.Fatalf("%s = %v, want %q", label, value, want)
	}
}

func lastConditionType(t *testing.T, resource *servicemanagerproxyv1beta1.ServiceEnvironment) string {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want at least one condition")
	}
	return string(conditions[len(conditions)-1].Type)
}
