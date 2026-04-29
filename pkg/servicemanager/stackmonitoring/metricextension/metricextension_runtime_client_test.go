/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package metricextension

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testMetricExtensionID            = "ocid1.metricextension.oc1..metricextension"
	testDeletedMetricExtensionID     = "ocid1.metricextension.oc1..deleted"
	testMetricExtensionCompartmentID = "ocid1.compartment.oc1..metricextension"
	testMetricExtensionResourceType  = "oci_oracle_database"
)

type erroringMetricExtensionConfigProvider struct {
	calls int
}

func (p *erroringMetricExtensionConfigProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	p.calls++
	return nil, errors.New("metric extension provider invalid")
}

func (p *erroringMetricExtensionConfigProvider) KeyID() (string, error) {
	p.calls++
	return "", errors.New("metric extension provider invalid")
}

func (p *erroringMetricExtensionConfigProvider) TenancyOCID() (string, error) {
	p.calls++
	return "", errors.New("metric extension provider invalid")
}

func (p *erroringMetricExtensionConfigProvider) UserOCID() (string, error) {
	p.calls++
	return "", errors.New("metric extension provider invalid")
}

func (p *erroringMetricExtensionConfigProvider) KeyFingerprint() (string, error) {
	p.calls++
	return "", errors.New("metric extension provider invalid")
}

func (p *erroringMetricExtensionConfigProvider) Region() (string, error) {
	p.calls++
	return "", errors.New("metric extension provider invalid")
}

func (p *erroringMetricExtensionConfigProvider) AuthType() (common.AuthConfig, error) {
	return common.AuthConfig{}, nil
}

func TestMetricExtensionRuntimeHooksConfigured(t *testing.T) {
	hooks := newMetricExtensionDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	applyMetricExtensionRuntimeHooks(nil, &hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want bool-preserving create builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want bool-preserving update builder")
	}
	if hooks.DeleteHooks.ConfirmRead == nil {
		t.Fatal("hooks.DeleteHooks.ConfirmRead = nil, want conservative delete confirm read")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
	if hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("hooks.DeleteHooks.ApplyOutcome = nil, want auth-shaped confirm-read guard")
	}

	body, err := hooks.BuildCreateBody(context.Background(), testMetricExtensionResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	values, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want map[string]any", body)
	}
	assertMetricExtensionCreateBodyMetricList(t, values)
	assertMetricExtensionCreateBodyQuery(t, values)
}

func assertMetricExtensionCreateBodyMetricList(t *testing.T, values map[string]any) {
	t.Helper()

	metricList, ok := values["metricList"].([]any)
	if !ok || len(metricList) != 1 {
		t.Fatalf("BuildCreateBody() metricList = %#v, want one decoded metric", values["metricList"])
	}
	metric, ok := metricList[0].(map[string]any)
	if !ok {
		t.Fatalf("BuildCreateBody() metricList[0] = %T, want map[string]any", metricList[0])
	}
	if got, ok := metric["isDimension"]; !ok || got != false {
		t.Fatalf("BuildCreateBody() metricList[0].isDimension = %#v, present = %t, want explicit false", got, ok)
	}
	if got, ok := metric["isHidden"]; !ok || got != false {
		t.Fatalf("BuildCreateBody() metricList[0].isHidden = %#v, present = %t, want explicit false", got, ok)
	}
}

func assertMetricExtensionCreateBodyQuery(t *testing.T, values map[string]any) {
	t.Helper()

	query, ok := values["queryProperties"].(map[string]any)
	if !ok {
		t.Fatalf("BuildCreateBody() queryProperties = %T, want map[string]any", values["queryProperties"])
	}
	if got, ok := query["isMetricServiceEnabled"]; !ok || got != false {
		t.Fatalf("BuildCreateBody() queryProperties.isMetricServiceEnabled = %#v, present = %t, want explicit false", got, ok)
	}
}

func TestMetricExtensionCreateOrUpdatePreservesGeneratedOCIInitErrorWhenWrapped(t *testing.T) {
	resource := testMetricExtensionResource()
	provider := &erroringMetricExtensionConfigProvider{}
	client := newMetricExtensionServiceClient(&MetricExtensionServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: logr.Discard()},
	})
	callsAfterInit := provider.calls

	response, err := client.CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	assertMetricExtensionInitError(t, err)
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
	assertMetricExtensionProviderCalls(t, provider, callsAfterInit)
}

func TestMetricExtensionDeletePreservesGeneratedOCIInitErrorWhenWrapped(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	provider := &erroringMetricExtensionConfigProvider{}
	client := newMetricExtensionServiceClient(&MetricExtensionServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: logr.Discard()},
	})
	callsAfterInit := provider.calls

	deleted, err := client.Delete(context.Background(), resource)
	assertMetricExtensionInitError(t, err)
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	assertMetricExtensionProviderCalls(t, provider, callsAfterInit)
}

func TestMetricExtensionCreateRecordsIdentityAndRequestID(t *testing.T) {
	resource := testMetricExtensionResource()
	created := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{
		listMetricExtensions: func(context.Context, stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
			return stackmonitoringsdk.ListMetricExtensionsResponse{}, nil
		},
		createMetricExtension: func(_ context.Context, request stackmonitoringsdk.CreateMetricExtensionRequest) (stackmonitoringsdk.CreateMetricExtensionResponse, error) {
			assertMetricExtensionCreateRequest(t, request)
			return stackmonitoringsdk.CreateMetricExtensionResponse{
				MetricExtension: created,
				OpcRequestId:    common.String("opc-create"),
			}, nil
		},
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{
				MetricExtension: created,
				OpcRequestId:    common.String("opc-get"),
			}, nil
		},
	}

	response, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMetricExtensionSuccessfulResponse(t, response)
	assertMetricExtensionCallCount(t, "CreateMetricExtension()", fake.createCalls, 1)
	assertMetricExtensionCallCount(t, "GetMetricExtension()", fake.getCalls, 1)
	assertMetricExtensionRecordedID(t, resource, testMetricExtensionID)
	assertMetricExtensionOpcRequestID(t, resource, "opc-create")
}

func TestMetricExtensionCreateRejectsQueryPropertiesJSONData(t *testing.T) {
	resource := testMetricExtensionResource()
	resource.Spec.QueryProperties.JsonData = `{"collectionMethod":"JMX","managedBeanQuery":"com.example:type=Server"}`
	fake := &fakeMetricExtensionOCIClient{
		createMetricExtension: func(context.Context, stackmonitoringsdk.CreateMetricExtensionRequest) (stackmonitoringsdk.CreateMetricExtensionResponse, error) {
			t.Fatal("CreateMetricExtension() called with unsupported queryProperties.jsonData")
			return stackmonitoringsdk.CreateMetricExtensionResponse{}, nil
		},
	}

	response, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want queryProperties.jsonData rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if !strings.Contains(err.Error(), "spec.queryProperties.jsonData") {
		t.Fatalf("CreateOrUpdate() error = %q, want queryProperties.jsonData context", err.Error())
	}
	assertMetricExtensionCallCount(t, "CreateMetricExtension()", fake.createCalls, 0)
}

func TestMetricExtensionBindUsesLaterListPage(t *testing.T) {
	resource := testMetricExtensionResource()
	live := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{
		listMetricExtensions: func(_ context.Context, request stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
			switch {
			case request.Page == nil:
				return stackmonitoringsdk.ListMetricExtensionsResponse{
					MetricExtensionCollection: stackmonitoringsdk.MetricExtensionCollection{
						Items: []stackmonitoringsdk.MetricExtensionSummary{sdkMetricExtensionSummary("ocid1.metricextension.oc1..other", "other", testMetricExtensionResourceType)},
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			case *request.Page == "next-page":
				return stackmonitoringsdk.ListMetricExtensionsResponse{
					MetricExtensionCollection: stackmonitoringsdk.MetricExtensionCollection{
						Items: []stackmonitoringsdk.MetricExtensionSummary{sdkMetricExtensionSummary(testMetricExtensionID, resource.Spec.Name, resource.Spec.ResourceType)},
					},
				}, nil
			default:
				t.Fatalf("ListMetricExtensions() page = %q, want empty or next-page", *request.Page)
				return stackmonitoringsdk.ListMetricExtensionsResponse{}, nil
			}
		},
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: live}, nil
		},
	}

	response, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMetricExtensionSuccessfulResponse(t, response)
	assertMetricExtensionCallCount(t, "ListMetricExtensions()", fake.listCalls, 2)
	assertMetricExtensionCallCount(t, "CreateMetricExtension()", fake.createCalls, 0)
	assertMetricExtensionRecordedID(t, resource, testMetricExtensionID)
}

func TestMetricExtensionCreateIgnoresDeletedListFallbackMatch(t *testing.T) {
	resource := testMetricExtensionResource()
	created := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	deletedSummary := sdkMetricExtensionSummary(testDeletedMetricExtensionID, resource.Spec.Name, resource.Spec.ResourceType)
	deletedSummary.LifecycleState = stackmonitoringsdk.MetricExtensionLifeCycleStatesDeleted
	fake := &fakeMetricExtensionOCIClient{
		listMetricExtensions: func(context.Context, stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
			return stackmonitoringsdk.ListMetricExtensionsResponse{
				MetricExtensionCollection: stackmonitoringsdk.MetricExtensionCollection{
					Items: []stackmonitoringsdk.MetricExtensionSummary{deletedSummary},
				},
			}, nil
		},
		createMetricExtension: func(context.Context, stackmonitoringsdk.CreateMetricExtensionRequest) (stackmonitoringsdk.CreateMetricExtensionResponse, error) {
			return stackmonitoringsdk.CreateMetricExtensionResponse{MetricExtension: created}, nil
		},
		getMetricExtension: func(_ context.Context, request stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			if request.MetricExtensionId != nil && *request.MetricExtensionId == testDeletedMetricExtensionID {
				t.Fatal("GetMetricExtension() called for DELETED list fallback match")
			}
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: created}, nil
		},
		updateMetricExtension: func(context.Context, stackmonitoringsdk.UpdateMetricExtensionRequest) (stackmonitoringsdk.UpdateMetricExtensionResponse, error) {
			t.Fatal("UpdateMetricExtension() called for DELETED list fallback match")
			return stackmonitoringsdk.UpdateMetricExtensionResponse{}, nil
		},
	}

	response, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMetricExtensionSuccessfulResponse(t, response)
	assertMetricExtensionCallCount(t, "ListMetricExtensions()", fake.listCalls, 1)
	assertMetricExtensionCallCount(t, "CreateMetricExtension()", fake.createCalls, 1)
	assertMetricExtensionCallCount(t, "UpdateMetricExtension()", fake.updateCalls, 0)
	assertMetricExtensionRecordedID(t, resource, testMetricExtensionID)
}

func TestMetricExtensionNoOpReconcileDoesNotUpdate(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	live := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: live}, nil
		},
	}

	response, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMetricExtensionSuccessfulResponse(t, response)
	assertMetricExtensionCallCount(t, "UpdateMetricExtension()", fake.updateCalls, 0)
}

func TestMetricExtensionRecordedIDDeletedReadbackCreatesReplacement(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testDeletedMetricExtensionID)
	deletedCurrent := sdkMetricExtensionFromResource(resource, testDeletedMetricExtensionID)
	deletedCurrent.LifecycleState = stackmonitoringsdk.MetricExtensionLifeCycleStatesDeleted
	created := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{
		listMetricExtensions: func(context.Context, stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
			return stackmonitoringsdk.ListMetricExtensionsResponse{}, nil
		},
		getMetricExtension: func(_ context.Context, request stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			if request.MetricExtensionId == nil {
				t.Fatal("GetMetricExtension() metricExtensionId = nil")
			}
			switch *request.MetricExtensionId {
			case testDeletedMetricExtensionID:
				return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: deletedCurrent}, nil
			case testMetricExtensionID:
				return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: created}, nil
			default:
				t.Fatalf("GetMetricExtension() metricExtensionId = %q, want deleted or replacement ID", *request.MetricExtensionId)
				return stackmonitoringsdk.GetMetricExtensionResponse{}, nil
			}
		},
		createMetricExtension: func(context.Context, stackmonitoringsdk.CreateMetricExtensionRequest) (stackmonitoringsdk.CreateMetricExtensionResponse, error) {
			return stackmonitoringsdk.CreateMetricExtensionResponse{MetricExtension: created}, nil
		},
		updateMetricExtension: func(context.Context, stackmonitoringsdk.UpdateMetricExtensionRequest) (stackmonitoringsdk.UpdateMetricExtensionResponse, error) {
			t.Fatal("UpdateMetricExtension() called for DELETED recorded-ID readback")
			return stackmonitoringsdk.UpdateMetricExtensionResponse{}, nil
		},
	}

	response, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMetricExtensionSuccessfulResponse(t, response)
	assertMetricExtensionCallCount(t, "GetMetricExtension()", fake.getCalls, 2)
	assertMetricExtensionCallCount(t, "ListMetricExtensions()", fake.listCalls, 1)
	assertMetricExtensionCallCount(t, "CreateMetricExtension()", fake.createCalls, 1)
	assertMetricExtensionCallCount(t, "UpdateMetricExtension()", fake.updateCalls, 0)
	assertMetricExtensionRecordedID(t, resource, testMetricExtensionID)
}

func TestMetricExtensionUpdateRejectsQueryPropertiesJSONDataAfterReadback(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	current := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	resource.Spec.QueryProperties.JsonData = `{"collectionMethod":"JMX","managedBeanQuery":"com.example:type=Server"}`
	fake := &fakeMetricExtensionOCIClient{
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: current}, nil
		},
		updateMetricExtension: func(context.Context, stackmonitoringsdk.UpdateMetricExtensionRequest) (stackmonitoringsdk.UpdateMetricExtensionResponse, error) {
			t.Fatal("UpdateMetricExtension() called with unsupported queryProperties.jsonData")
			return stackmonitoringsdk.UpdateMetricExtensionResponse{}, nil
		},
	}

	_, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want queryProperties.jsonData rejection")
	}
	if !strings.Contains(err.Error(), "spec.queryProperties.jsonData") {
		t.Fatalf("CreateOrUpdate() error = %q, want queryProperties.jsonData context", err.Error())
	}
	assertMetricExtensionCallCount(t, "GetMetricExtension()", fake.getCalls, 1)
	assertMetricExtensionCallCount(t, "UpdateMetricExtension()", fake.updateCalls, 0)
}

func TestMetricExtensionMutableUpdateShapesBody(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	resource.Spec.DisplayName = "metric extension updated"
	resource.Spec.QueryProperties.IsMetricServiceEnabled = false
	current := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	current.DisplayName = common.String("metric extension original")
	current.QueryProperties = stackmonitoringsdk.JmxQueryProperties{
		ManagedBeanQuery:       common.String(resource.Spec.QueryProperties.ManagedBeanQuery),
		JmxAttributes:          common.String(resource.Spec.QueryProperties.JmxAttributes),
		IdentityMetric:         common.String(resource.Spec.QueryProperties.IdentityMetric),
		AutoRowPrefix:          common.String(resource.Spec.QueryProperties.AutoRowPrefix),
		IsMetricServiceEnabled: common.Bool(true),
	}
	updated := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{}
	fake.getMetricExtension = func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
		if fake.getCalls == 1 {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: current}, nil
		}
		return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: updated}, nil
	}
	fake.updateMetricExtension = func(_ context.Context, request stackmonitoringsdk.UpdateMetricExtensionRequest) (stackmonitoringsdk.UpdateMetricExtensionResponse, error) {
		assertMetricExtensionUpdateRequest(t, request)
		return stackmonitoringsdk.UpdateMetricExtensionResponse{
			MetricExtension: updated,
			OpcRequestId:    common.String("opc-update"),
		}, nil
	}

	response, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMetricExtensionSuccessfulResponse(t, response)
	assertMetricExtensionCallCount(t, "UpdateMetricExtension()", fake.updateCalls, 1)
	assertMetricExtensionOpcRequestID(t, resource, "opc-update")
}

func TestMetricExtensionImmutableDriftRejectedBeforeUpdate(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	current := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	current.ResourceType = common.String("old_resource_type")
	fake := &fakeMetricExtensionOCIClient{
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: current}, nil
		},
	}

	_, err := newTestMetricExtensionClient(fake).CreateOrUpdate(context.Background(), resource, testMetricExtensionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want force-new drift rejection")
	}
	if !strings.Contains(err.Error(), "require replacement when resourceType changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want resourceType force-new context", err.Error())
	}
	assertMetricExtensionCallCount(t, "UpdateMetricExtension()", fake.updateCalls, 0)
}

func TestMetricExtensionDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	active := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: active}, nil
		},
		deleteMetricExtension: func(context.Context, stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error) {
			return stackmonitoringsdk.DeleteMetricExtensionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestMetricExtensionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback remains active")
	}
	assertMetricExtensionCallCount(t, "DeleteMetricExtension()", fake.deleteCalls, 1)
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete pending", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMetricExtensionDeleteSkipsDeleteWhenPreReadAlreadyDeleted(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	deletedCurrent := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	deletedCurrent.LifecycleState = stackmonitoringsdk.MetricExtensionLifeCycleStatesDeleted
	fake := &fakeMetricExtensionOCIClient{
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: deletedCurrent}, nil
		},
		deleteMetricExtension: func(context.Context, stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error) {
			t.Fatal("DeleteMetricExtension() called for terminal deleted readback")
			return stackmonitoringsdk.DeleteMetricExtensionResponse{}, nil
		},
	}

	deleted, err := newTestMetricExtensionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after terminal deleted readback")
	}
	assertMetricExtensionCallCount(t, "GetMetricExtension()", fake.getCalls, 1)
	assertMetricExtensionCallCount(t, "DeleteMetricExtension()", fake.deleteCalls, 0)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deleted timestamp")
	}
}

func TestMetricExtensionDeleteReleasesAfterNotFoundConfirmation(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	active := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{}
	fake.getMetricExtension = func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
		if fake.getCalls == 1 {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: active}, nil
		}
		return stackmonitoringsdk.GetMetricExtensionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "metric extension deleted")
	}
	fake.deleteMetricExtension = func(context.Context, stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error) {
		return stackmonitoringsdk.DeleteMetricExtensionResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestMetricExtensionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after unambiguous NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deleted timestamp")
	}
}

func TestMetricExtensionDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	fake := &fakeMetricExtensionOCIClient{
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{}, authErr
		},
	}

	deleted, err := newTestMetricExtensionClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm-read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained on auth-shaped confirm read")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound context", err.Error())
	}
	assertMetricExtensionCallCount(t, "DeleteMetricExtension()", fake.deleteCalls, 0)
	assertMetricExtensionOpcRequestID(t, resource, authErr.GetOpcRequestID())
}

func TestMetricExtensionDeleteRejectsAuthShapedDeleteError(t *testing.T) {
	resource := testMetricExtensionResource()
	recordMetricExtensionID(resource, testMetricExtensionID)
	active := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
	fake := &fakeMetricExtensionOCIClient{
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: active}, nil
		},
		deleteMetricExtension: func(context.Context, stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error) {
			return stackmonitoringsdk.DeleteMetricExtensionResponse{}, authErr
		},
	}

	deleted, err := newTestMetricExtensionClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped delete rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained on auth-shaped delete error")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound context", err.Error())
	}
	assertMetricExtensionCallCount(t, "DeleteMetricExtension()", fake.deleteCalls, 1)
	assertMetricExtensionOpcRequestID(t, resource, authErr.GetOpcRequestID())
}

func TestMetricExtensionDeleteFallbackResolvesMissingRecordedID(t *testing.T) {
	resource := testMetricExtensionResource()
	active := sdkMetricExtensionFromResource(resource, testMetricExtensionID)
	fake := &fakeMetricExtensionOCIClient{
		listMetricExtensions: func(_ context.Context, request stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
			if request.Page == nil {
				return stackmonitoringsdk.ListMetricExtensionsResponse{
					MetricExtensionCollection: stackmonitoringsdk.MetricExtensionCollection{
						Items: []stackmonitoringsdk.MetricExtensionSummary{sdkMetricExtensionSummary("ocid1.metricextension.oc1..other", "other", testMetricExtensionResourceType)},
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			}
			return stackmonitoringsdk.ListMetricExtensionsResponse{
				MetricExtensionCollection: stackmonitoringsdk.MetricExtensionCollection{
					Items: []stackmonitoringsdk.MetricExtensionSummary{sdkMetricExtensionSummary(testMetricExtensionID, resource.Spec.Name, resource.Spec.ResourceType)},
				},
			}, nil
		},
		getMetricExtension: func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
			return stackmonitoringsdk.GetMetricExtensionResponse{MetricExtension: active}, nil
		},
		deleteMetricExtension: func(context.Context, stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error) {
			return stackmonitoringsdk.DeleteMetricExtensionResponse{}, nil
		},
	}

	deleted, err := newTestMetricExtensionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want retained after fallback resolves live active resource")
	}
	assertMetricExtensionCallCount(t, "ListMetricExtensions()", fake.listCalls, 2)
	assertMetricExtensionRecordedID(t, resource, testMetricExtensionID)
}

func assertMetricExtensionCreateRequest(t *testing.T, request stackmonitoringsdk.CreateMetricExtensionRequest) {
	t.Helper()

	if request.Name == nil || *request.Name != "metric-extension" {
		t.Fatalf("CreateMetricExtension() name = %#v, want metric-extension", request.Name)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateMetricExtension() opc retry token is empty")
	}
	assertMetricExtensionCreateRequestQuery(t, request)
	assertMetricExtensionCreateRequestMetricList(t, request)
}

func assertMetricExtensionCreateRequestQuery(t *testing.T, request stackmonitoringsdk.CreateMetricExtensionRequest) {
	t.Helper()

	query, ok := request.QueryProperties.(stackmonitoringsdk.JmxQueryProperties)
	if !ok {
		t.Fatalf("CreateMetricExtension() queryProperties = %T, want JmxQueryProperties", request.QueryProperties)
	}
	if query.IsMetricServiceEnabled == nil || *query.IsMetricServiceEnabled {
		t.Fatalf("CreateMetricExtension() queryProperties.isMetricServiceEnabled = %#v, want explicit false", query.IsMetricServiceEnabled)
	}
}

func assertMetricExtensionCreateRequestMetricList(t *testing.T, request stackmonitoringsdk.CreateMetricExtensionRequest) {
	t.Helper()

	if len(request.MetricList) != 1 || request.MetricList[0].IsDimension == nil || *request.MetricList[0].IsDimension {
		t.Fatalf("CreateMetricExtension() metricList = %#v, want explicit false isDimension", request.MetricList)
	}
}

func assertMetricExtensionUpdateRequest(t *testing.T, request stackmonitoringsdk.UpdateMetricExtensionRequest) {
	t.Helper()

	if request.MetricExtensionId == nil || *request.MetricExtensionId != testMetricExtensionID {
		t.Fatalf("UpdateMetricExtension() metricExtensionId = %#v, want %q", request.MetricExtensionId, testMetricExtensionID)
	}
	if request.DisplayName == nil || *request.DisplayName != "metric extension updated" {
		t.Fatalf("UpdateMetricExtension() displayName = %#v, want updated value", request.DisplayName)
	}
	query, ok := request.QueryProperties.(stackmonitoringsdk.JmxUpdateQueryProperties)
	if !ok {
		t.Fatalf("UpdateMetricExtension() queryProperties = %T, want JmxUpdateQueryProperties", request.QueryProperties)
	}
	if query.IsMetricServiceEnabled == nil || *query.IsMetricServiceEnabled {
		t.Fatalf("UpdateMetricExtension() queryProperties.isMetricServiceEnabled = %#v, want explicit false", query.IsMetricServiceEnabled)
	}
}

func testMetricExtensionResource() *stackmonitoringv1beta1.MetricExtension {
	return &stackmonitoringv1beta1.MetricExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metric-extension",
			Namespace: "default",
			UID:       types.UID("metric-extension-uid"),
		},
		Spec: stackmonitoringv1beta1.MetricExtensionSpec{
			Name:                  "metric-extension",
			DisplayName:           "metric extension",
			ResourceType:          testMetricExtensionResourceType,
			CompartmentId:         testMetricExtensionCompartmentID,
			CollectionRecurrences: "FREQ=DAILY;INTERVAL=1",
			MetricList: []stackmonitoringv1beta1.MetricExtensionMetricList{
				{
					Name:              "open_connections",
					DataType:          "NUMBER",
					DisplayName:       "Open connections",
					IsDimension:       false,
					ComputeExpression: "value",
					IsHidden:          false,
					MetricCategory:    "UTILIZATION",
					Unit:              "count",
				},
			},
			QueryProperties: stackmonitoringv1beta1.MetricExtensionQueryProperties{
				CollectionMethod:       "JMX",
				ManagedBeanQuery:       "com.example:type=Server",
				JmxAttributes:          "OpenConnectionsCurrentCount",
				IdentityMetric:         "name",
				AutoRowPrefix:          "row",
				IsMetricServiceEnabled: false,
			},
			Description: "metric extension description",
		},
	}
}

func sdkMetricExtensionFromResource(
	resource *stackmonitoringv1beta1.MetricExtension,
	id string,
) stackmonitoringsdk.MetricExtension {
	return stackmonitoringsdk.MetricExtension{
		Id:                    common.String(id),
		Name:                  common.String(resource.Spec.Name),
		DisplayName:           common.String(resource.Spec.DisplayName),
		ResourceType:          common.String(resource.Spec.ResourceType),
		CompartmentId:         common.String(resource.Spec.CompartmentId),
		TenantId:              common.String("ocid1.tenancy.oc1..metricextension"),
		CollectionMethod:      common.String(resource.Spec.QueryProperties.CollectionMethod),
		Status:                stackmonitoringsdk.MetricExtensionLifeCycleDetailsDraft,
		CollectionRecurrences: common.String(resource.Spec.CollectionRecurrences),
		MetricList:            sdkMetricListFromResource(resource),
		QueryProperties: stackmonitoringsdk.JmxQueryProperties{
			ManagedBeanQuery:       common.String(resource.Spec.QueryProperties.ManagedBeanQuery),
			JmxAttributes:          common.String(resource.Spec.QueryProperties.JmxAttributes),
			IdentityMetric:         common.String(resource.Spec.QueryProperties.IdentityMetric),
			AutoRowPrefix:          common.String(resource.Spec.QueryProperties.AutoRowPrefix),
			IsMetricServiceEnabled: common.Bool(resource.Spec.QueryProperties.IsMetricServiceEnabled),
		},
		Description:             common.String(resource.Spec.Description),
		LifecycleState:          stackmonitoringsdk.MetricExtensionLifeCycleStatesActive,
		EnabledOnResourcesCount: common.Int(0),
		ResourceUri:             common.String("/metricExtensions/" + id),
	}
}

func sdkMetricListFromResource(resource *stackmonitoringv1beta1.MetricExtension) []stackmonitoringsdk.Metric {
	items := make([]stackmonitoringsdk.Metric, 0, len(resource.Spec.MetricList))
	for _, item := range resource.Spec.MetricList {
		items = append(items, stackmonitoringsdk.Metric{
			Name:              common.String(item.Name),
			DataType:          stackmonitoringsdk.MetricDataTypeEnum(item.DataType),
			DisplayName:       common.String(item.DisplayName),
			IsDimension:       common.Bool(item.IsDimension),
			ComputeExpression: common.String(item.ComputeExpression),
			IsHidden:          common.Bool(item.IsHidden),
			MetricCategory:    stackmonitoringsdk.MetricMetricCategoryEnum(item.MetricCategory),
			Unit:              common.String(item.Unit),
		})
	}
	return items
}

func sdkMetricExtensionSummary(id string, name string, resourceType string) stackmonitoringsdk.MetricExtensionSummary {
	return stackmonitoringsdk.MetricExtensionSummary{
		Id:            common.String(id),
		Name:          common.String(name),
		ResourceType:  common.String(resourceType),
		CompartmentId: common.String(testMetricExtensionCompartmentID),
		Status:        stackmonitoringsdk.MetricExtensionLifeCycleDetailsDraft,
		DisplayName:   common.String("metric extension"),
	}
}

func newTestMetricExtensionClient(fake *fakeMetricExtensionOCIClient) MetricExtensionServiceClient {
	return newMetricExtensionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func testMetricExtensionRequest(resource *stackmonitoringv1beta1.MetricExtension) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func assertMetricExtensionSuccessfulResponse(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatalf("OSOKResponse.IsSuccessful = false, want true")
	}
}

func assertMetricExtensionRecordedID(t *testing.T, resource *stackmonitoringv1beta1.MetricExtension, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertMetricExtensionOpcRequestID(t *testing.T, resource *stackmonitoringv1beta1.MetricExtension, want string) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertMetricExtensionCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func assertMetricExtensionInitError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("operation error = nil, want OCI client initialization error")
	}
	if !strings.Contains(err.Error(), "initialize MetricExtension OCI client") {
		t.Fatalf("operation error = %v, want MetricExtension OCI client initialization failure", err)
	}
	if !strings.Contains(err.Error(), "metric extension provider invalid") {
		t.Fatalf("operation error = %v, want provider failure detail", err)
	}
}

func assertMetricExtensionProviderCalls(t *testing.T, provider *erroringMetricExtensionConfigProvider, want int) {
	t.Helper()

	if provider.calls != want {
		t.Fatalf("provider calls after operation = %d, want %d; runtime wrapper should stop at InitError", provider.calls, want)
	}
}

type fakeMetricExtensionOCIClient struct {
	createMetricExtension func(context.Context, stackmonitoringsdk.CreateMetricExtensionRequest) (stackmonitoringsdk.CreateMetricExtensionResponse, error)
	getMetricExtension    func(context.Context, stackmonitoringsdk.GetMetricExtensionRequest) (stackmonitoringsdk.GetMetricExtensionResponse, error)
	listMetricExtensions  func(context.Context, stackmonitoringsdk.ListMetricExtensionsRequest) (stackmonitoringsdk.ListMetricExtensionsResponse, error)
	updateMetricExtension func(context.Context, stackmonitoringsdk.UpdateMetricExtensionRequest) (stackmonitoringsdk.UpdateMetricExtensionResponse, error)
	deleteMetricExtension func(context.Context, stackmonitoringsdk.DeleteMetricExtensionRequest) (stackmonitoringsdk.DeleteMetricExtensionResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeMetricExtensionOCIClient) CreateMetricExtension(
	ctx context.Context,
	request stackmonitoringsdk.CreateMetricExtensionRequest,
) (stackmonitoringsdk.CreateMetricExtensionResponse, error) {
	f.createCalls++
	if f.createMetricExtension == nil {
		return stackmonitoringsdk.CreateMetricExtensionResponse{}, nil
	}
	return f.createMetricExtension(ctx, request)
}

func (f *fakeMetricExtensionOCIClient) GetMetricExtension(
	ctx context.Context,
	request stackmonitoringsdk.GetMetricExtensionRequest,
) (stackmonitoringsdk.GetMetricExtensionResponse, error) {
	f.getCalls++
	if f.getMetricExtension == nil {
		return stackmonitoringsdk.GetMetricExtensionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "metric extension not found")
	}
	return f.getMetricExtension(ctx, request)
}

func (f *fakeMetricExtensionOCIClient) ListMetricExtensions(
	ctx context.Context,
	request stackmonitoringsdk.ListMetricExtensionsRequest,
) (stackmonitoringsdk.ListMetricExtensionsResponse, error) {
	f.listCalls++
	if f.listMetricExtensions == nil {
		return stackmonitoringsdk.ListMetricExtensionsResponse{}, nil
	}
	return f.listMetricExtensions(ctx, request)
}

func (f *fakeMetricExtensionOCIClient) UpdateMetricExtension(
	ctx context.Context,
	request stackmonitoringsdk.UpdateMetricExtensionRequest,
) (stackmonitoringsdk.UpdateMetricExtensionResponse, error) {
	f.updateCalls++
	if f.updateMetricExtension == nil {
		return stackmonitoringsdk.UpdateMetricExtensionResponse{}, nil
	}
	return f.updateMetricExtension(ctx, request)
}

func (f *fakeMetricExtensionOCIClient) DeleteMetricExtension(
	ctx context.Context,
	request stackmonitoringsdk.DeleteMetricExtensionRequest,
) (stackmonitoringsdk.DeleteMetricExtensionResponse, error) {
	f.deleteCalls++
	if f.deleteMetricExtension == nil {
		return stackmonitoringsdk.DeleteMetricExtensionResponse{}, nil
	}
	return f.deleteMetricExtension(ctx, request)
}
