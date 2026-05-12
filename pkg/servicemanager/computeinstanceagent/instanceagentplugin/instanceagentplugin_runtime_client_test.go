/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package instanceagentplugin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	computeinstanceagentsdk "github.com/oracle/oci-go-sdk/v65/computeinstanceagent"
	computeinstanceagentv1beta1 "github.com/oracle/oci-service-operator/api/computeinstanceagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testInstanceAgentID = "ocid1.instance.oc1..exampleuniqueID"
	testCompartmentID   = "ocid1.compartment.oc1..exampleuniqueID"
	testPluginName      = "Compute Instance Monitoring"
)

type fakeInstanceAgentPluginOCIClient struct {
	getFn  func(context.Context, computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error)
	listFn func(context.Context, computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error)

	getRequests  []computeinstanceagentsdk.GetInstanceAgentPluginRequest
	listRequests []computeinstanceagentsdk.ListInstanceAgentPluginsRequest
}

func (f *fakeInstanceAgentPluginOCIClient) GetInstanceAgentPlugin(
	ctx context.Context,
	request computeinstanceagentsdk.GetInstanceAgentPluginRequest,
) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return computeinstanceagentsdk.GetInstanceAgentPluginResponse{}, nil
}

func (f *fakeInstanceAgentPluginOCIClient) ListInstanceAgentPlugins(
	ctx context.Context,
	request computeinstanceagentsdk.ListInstanceAgentPluginsRequest,
) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return computeinstanceagentsdk.ListInstanceAgentPluginsResponse{}, nil
}

func TestInstanceAgentPluginCreateOrUpdateProjectsStableObservedState(t *testing.T) {
	t.Parallel()

	resource := instanceAgentPluginResource()
	client := &fakeInstanceAgentPluginOCIClient{
		getFn: func(_ context.Context, request computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error) {
			requireStringPtr(t, "get instanceagentId", request.InstanceagentId, testInstanceAgentID)
			requireStringPtr(t, "get compartmentId", request.CompartmentId, testCompartmentID)
			requireStringPtr(t, "get pluginName", request.PluginName, testPluginName)
			return computeinstanceagentsdk.GetInstanceAgentPluginResponse{
				InstanceAgentPlugin: sdkInstanceAgentPlugin(
					testPluginName,
					computeinstanceagentsdk.InstanceAgentPluginStatusStopped,
					"plugin deliberately disabled",
				),
			}, nil
		},
		listFn: func(_ context.Context, request computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error) {
			t.Fatalf("ListInstanceAgentPlugins(%#v) should not be called when GetInstanceAgentPlugin succeeds", request)
			return computeinstanceagentsdk.ListInstanceAgentPluginsResponse{}, nil
		},
	}

	response, err := newInstanceAgentPluginServiceClientWithOCIClient(testInstanceAgentPluginLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want stable success without requeue", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != trackedInstanceAgentPluginID(testInstanceAgentID, testCompartmentID, testPluginName) {
		t.Fatalf("status.ocid = %q, want composite tracked identity", got)
	}
	if got := resource.Status.InstanceagentId; got != testInstanceAgentID {
		t.Fatalf("status.instanceagentId = %q, want %q", got, testInstanceAgentID)
	}
	if got := resource.Status.CompartmentId; got != testCompartmentID {
		t.Fatalf("status.compartmentId = %q, want %q", got, testCompartmentID)
	}
	if got := resource.Status.Name; got != testPluginName {
		t.Fatalf("status.name = %q, want %q", got, testPluginName)
	}
	if got := resource.Status.Status; got != string(computeinstanceagentsdk.InstanceAgentPluginStatusStopped) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, computeinstanceagentsdk.InstanceAgentPluginStatusStopped)
	}
	if got := resource.Status.Message; got != "plugin deliberately disabled" {
		t.Fatalf("status.message = %q, want plugin detail", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != "Active" {
		t.Fatalf("status.reason = %q, want Active", got)
	}
	if got := resource.Status.OsokStatus.Message; !strings.Contains(got, "STOPPED") || !strings.Contains(got, "plugin deliberately disabled") {
		t.Fatalf("osok status.message = %q, want status and plugin detail", got)
	}
	if resource.Status.OsokStatus.CreatedAt == nil {
		t.Fatal("status.createdAt = nil, want timestamp after successful bind")
	}
	if got := lastConditionType(t, resource); got != "Active" {
		t.Fatalf("last condition type = %q, want Active", got)
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("GetInstanceAgentPlugin() calls = %d, want 1", len(client.getRequests))
	}
	if len(client.listRequests) != 0 {
		t.Fatalf("ListInstanceAgentPlugins() calls = %d, want 0", len(client.listRequests))
	}
}

func TestInstanceAgentPluginCreateOrUpdateFailsWhenPluginIsMissing(t *testing.T) {
	t.Parallel()

	resource := instanceAgentPluginResource()
	client := &fakeInstanceAgentPluginOCIClient{
		getFn: func(_ context.Context, request computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error) {
			requireStringPtr(t, "get instanceagentId", request.InstanceagentId, testInstanceAgentID)
			requireStringPtr(t, "get compartmentId", request.CompartmentId, testCompartmentID)
			requireStringPtr(t, "get pluginName", request.PluginName, testPluginName)
			return computeinstanceagentsdk.GetInstanceAgentPluginResponse{}, errortest.NewServiceError(404, "NotFound", "missing plugin")
		},
		listFn: func(_ context.Context, request computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error) {
			requireStringPtr(t, "list instanceagentId", request.InstanceagentId, testInstanceAgentID)
			requireStringPtr(t, "list compartmentId", request.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list name", request.Name, testPluginName)
			return computeinstanceagentsdk.ListInstanceAgentPluginsResponse{}, nil
		},
	}

	response, err := newInstanceAgentPluginServiceClientWithOCIClient(testInstanceAgentPluginLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing plugin failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("GetInstanceAgentPlugin() calls = %d, want 1", len(client.getRequests))
	}
	if len(client.listRequests) < 1 {
		t.Fatalf("ListInstanceAgentPlugins() calls = %d, want at least 1 confirmation read", len(client.listRequests))
	}
}

func TestInstanceAgentPluginCreateOrUpdateUsesListForMissingGetConfirmationOnly(t *testing.T) {
	t.Parallel()

	resource := instanceAgentPluginResource()
	client := &fakeInstanceAgentPluginOCIClient{
		getFn: func(_ context.Context, request computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error) {
			requireStringPtr(t, "get instanceagentId", request.InstanceagentId, testInstanceAgentID)
			requireStringPtr(t, "get compartmentId", request.CompartmentId, testCompartmentID)
			requireStringPtr(t, "get pluginName", request.PluginName, testPluginName)
			return computeinstanceagentsdk.GetInstanceAgentPluginResponse{}, errortest.NewServiceError(404, "NotFound", "missing plugin")
		},
		listFn: func(_ context.Context, request computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error) {
			requireStringPtr(t, "list instanceagentId", request.InstanceagentId, testInstanceAgentID)
			requireStringPtr(t, "list compartmentId", request.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list name", request.Name, testPluginName)
			return computeinstanceagentsdk.ListInstanceAgentPluginsResponse{
				Items: []computeinstanceagentsdk.InstanceAgentPluginSummary{
					sdkInstanceAgentPluginSummary(testPluginName, computeinstanceagentsdk.InstanceAgentPluginSummaryStatusRunning),
				},
			}, nil
		},
	}

	response, err := newInstanceAgentPluginServiceClientWithOCIClient(testInstanceAgentPluginLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "still finds a matching plugin") {
		t.Fatalf("CreateOrUpdate() error = %v, want list-confirmed mismatch failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response after get/list mismatch", response)
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("GetInstanceAgentPlugin() calls = %d, want 1", len(client.getRequests))
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("ListInstanceAgentPlugins() calls = %d, want 1", len(client.listRequests))
	}
}

func TestInstanceAgentPluginCreateOrUpdateRejectsTrackedPluginNameDrift(t *testing.T) {
	t.Parallel()

	resource := instanceAgentPluginResource()
	resource.Spec.PluginName = "OS Management Service Agent"
	resource.Status.Name = testPluginName
	resource.Status.InstanceagentId = testInstanceAgentID
	resource.Status.CompartmentId = testCompartmentID
	resource.Status.OsokStatus.Ocid = sharedTrackedIDForTest(testInstanceAgentID, testCompartmentID, testPluginName)

	client := &fakeInstanceAgentPluginOCIClient{
		getFn: func(_ context.Context, request computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error) {
			requireStringPtr(t, "get pluginName", request.PluginName, testPluginName)
			return computeinstanceagentsdk.GetInstanceAgentPluginResponse{
				InstanceAgentPlugin: sdkInstanceAgentPlugin(
					testPluginName,
					computeinstanceagentsdk.InstanceAgentPluginStatusRunning,
					"",
				),
			}, nil
		},
	}

	response, err := newInstanceAgentPluginServiceClientWithOCIClient(testInstanceAgentPluginLogger(), client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "pluginName changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want pluginName replacement error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("GetInstanceAgentPlugin() calls = %d, want 1", len(client.getRequests))
	}
}

func TestInstanceAgentPluginDeleteIsKubernetesLocalOnly(t *testing.T) {
	t.Parallel()

	resource := instanceAgentPluginResource()
	resource.Status.Name = testPluginName
	resource.Status.InstanceagentId = testInstanceAgentID
	resource.Status.CompartmentId = testCompartmentID

	client := &fakeInstanceAgentPluginOCIClient{
		getFn: func(_ context.Context, request computeinstanceagentsdk.GetInstanceAgentPluginRequest) (computeinstanceagentsdk.GetInstanceAgentPluginResponse, error) {
			t.Fatalf("GetInstanceAgentPlugin(%#v) should not be called during Kubernetes-local delete", request)
			return computeinstanceagentsdk.GetInstanceAgentPluginResponse{}, nil
		},
		listFn: func(_ context.Context, request computeinstanceagentsdk.ListInstanceAgentPluginsRequest) (computeinstanceagentsdk.ListInstanceAgentPluginsResponse, error) {
			t.Fatalf("ListInstanceAgentPlugins(%#v) should not be called during Kubernetes-local delete", request)
			return computeinstanceagentsdk.ListInstanceAgentPluginsResponse{}, nil
		},
	}

	deleted, err := newInstanceAgentPluginServiceClientWithOCIClient(testInstanceAgentPluginLogger(), client).Delete(context.Background(), resource)
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

func instanceAgentPluginResource() *computeinstanceagentv1beta1.InstanceAgentPlugin {
	return &computeinstanceagentv1beta1.InstanceAgentPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instanceagentplugin-sample",
			Namespace: "default",
		},
		Spec: computeinstanceagentv1beta1.InstanceAgentPluginSpec{
			InstanceagentId: testInstanceAgentID,
			CompartmentId:   testCompartmentID,
			PluginName:      testPluginName,
		},
	}
}

func sdkInstanceAgentPlugin(
	name string,
	status computeinstanceagentsdk.InstanceAgentPluginStatusEnum,
	message string,
) computeinstanceagentsdk.InstanceAgentPlugin {
	response := computeinstanceagentsdk.InstanceAgentPlugin{
		Name:               common.String(name),
		Status:             status,
		TimeLastUpdatedUtc: &common.SDKTime{Time: time.Date(2026, 5, 12, 18, 0, 0, 0, time.UTC)},
	}
	if strings.TrimSpace(message) != "" {
		response.Message = common.String(message)
	}
	return response
}

func sdkInstanceAgentPluginSummary(
	name string,
	status computeinstanceagentsdk.InstanceAgentPluginSummaryStatusEnum,
) computeinstanceagentsdk.InstanceAgentPluginSummary {
	return computeinstanceagentsdk.InstanceAgentPluginSummary{
		Name:               common.String(name),
		Status:             status,
		TimeLastUpdatedUtc: &common.SDKTime{Time: time.Date(2026, 5, 12, 18, 0, 0, 0, time.UTC)},
	}
}

func testInstanceAgentPluginLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("instanceagentplugin-runtime-test")}
}

func requireStringPtr(t *testing.T, label string, value *string, want string) {
	t.Helper()
	if value == nil || *value != want {
		t.Fatalf("%s = %v, want %q", label, value, want)
	}
}

func lastConditionType(t *testing.T, resource *computeinstanceagentv1beta1.InstanceAgentPlugin) string {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want at least one condition")
	}
	return string(conditions[len(conditions)-1].Type)
}

func trackedInstanceAgentPluginID(instanceAgentID, compartmentID, pluginName string) string {
	return strings.Join([]string{instanceAgentID, compartmentID, pluginName}, "|")
}

func sharedTrackedIDForTest(instanceAgentID, compartmentID, pluginName string) shared.OCID {
	return shared.OCID(trackedInstanceAgentPluginID(instanceAgentID, compartmentID, pluginName))
}
