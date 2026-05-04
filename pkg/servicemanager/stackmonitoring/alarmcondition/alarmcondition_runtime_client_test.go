/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alarmcondition

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAlarmConditionID                  = "ocid1.alarmcondition.oc1..alarm"
	testOtherAlarmConditionID             = "ocid1.alarmcondition.oc1..other"
	testAlarmConditionMonitoringTemplate  = "ocid1.monitoringtemplate.oc1..template"
	testOtherAlarmConditionMonitoringTmpl = "ocid1.monitoringtemplate.oc1..other"
)

type fakeAlarmConditionOCIClient struct {
	createFn func(context.Context, stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error)
	getFn    func(context.Context, stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error)
	listFn   func(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error)
	updateFn func(context.Context, stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error)
	deleteFn func(context.Context, stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error)
}

func (f *fakeAlarmConditionOCIClient) CreateAlarmCondition(
	ctx context.Context,
	request stackmonitoringsdk.CreateAlarmConditionRequest,
) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return stackmonitoringsdk.CreateAlarmConditionResponse{}, nil
}

func (f *fakeAlarmConditionOCIClient) GetAlarmCondition(
	ctx context.Context,
	request stackmonitoringsdk.GetAlarmConditionRequest,
) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return stackmonitoringsdk.GetAlarmConditionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "alarm condition not found")
}

func (f *fakeAlarmConditionOCIClient) ListAlarmConditions(
	ctx context.Context,
	request stackmonitoringsdk.ListAlarmConditionsRequest,
) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return stackmonitoringsdk.ListAlarmConditionsResponse{}, nil
}

func (f *fakeAlarmConditionOCIClient) UpdateAlarmCondition(
	ctx context.Context,
	request stackmonitoringsdk.UpdateAlarmConditionRequest,
) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return stackmonitoringsdk.UpdateAlarmConditionResponse{}, nil
}

func (f *fakeAlarmConditionOCIClient) DeleteAlarmCondition(
	ctx context.Context,
	request stackmonitoringsdk.DeleteAlarmConditionRequest,
) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return stackmonitoringsdk.DeleteAlarmConditionResponse{}, nil
}

func testAlarmConditionClient(fake *fakeAlarmConditionOCIClient) AlarmConditionServiceClient {
	return newAlarmConditionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeAlarmConditionResource() *stackmonitoringv1beta1.AlarmCondition {
	return &stackmonitoringv1beta1.AlarmCondition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alarm-condition",
			Namespace: "default",
			UID:       types.UID("alarm-condition-uid"),
			Annotations: map[string]string{
				alarmConditionMonitoringTemplateIDAnnotation: testAlarmConditionMonitoringTemplate,
			},
		},
		Spec: stackmonitoringv1beta1.AlarmConditionSpec{
			Namespace:     "oracle_oci_database",
			ResourceType:  "ocid1.stackmonitoringresourcetype.oc1..db",
			MetricName:    "CpuUtilization",
			ConditionType: "FIXED",
			Conditions: []stackmonitoringv1beta1.AlarmConditionCondition{{
				Severity:         "CRITICAL",
				Query:            "CpuUtilization[1m].mean() > 90",
				Body:             "CPU too high",
				ShouldAppendNote: false,
				ShouldAppendUrl:  true,
				TriggerDelay:     "PT5M",
			}},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeAlarmConditionRequest(resource *stackmonitoringv1beta1.AlarmCondition) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func makeSDKAlarmCondition(
	id string,
	templateID string,
	spec stackmonitoringv1beta1.AlarmConditionSpec,
	state stackmonitoringsdk.AlarmConditionLifeCycleStatesEnum,
) stackmonitoringsdk.AlarmCondition {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)}
	conditions, _ := alarmConditionConditionsFromSpec(spec.Conditions)
	return stackmonitoringsdk.AlarmCondition{
		Id:                   common.String(id),
		MonitoringTemplateId: common.String(templateID),
		Namespace:            common.String(spec.Namespace),
		ResourceType:         common.String(spec.ResourceType),
		MetricName:           common.String(spec.MetricName),
		ConditionType:        stackmonitoringsdk.ConditionTypeEnum(spec.ConditionType),
		Conditions:           conditions,
		Status:               stackmonitoringsdk.AlarmConditionLifeCycleDetailsApplied,
		LifecycleState:       state,
		CompositeType:        optionalTestString(spec.CompositeType),
		TimeCreated:          &created,
		TimeUpdated:          &updated,
		FreeformTags:         cloneAlarmConditionStringMap(spec.FreeformTags),
		DefinedTags:          makeAlarmConditionDefinedTags(spec.DefinedTags),
	}
}

func makeSDKAlarmConditionSummary(
	id string,
	templateID string,
	spec stackmonitoringv1beta1.AlarmConditionSpec,
) stackmonitoringsdk.AlarmConditionSummary {
	current := makeSDKAlarmCondition(id, templateID, spec, stackmonitoringsdk.AlarmConditionLifeCycleStatesActive)
	return stackmonitoringsdk.AlarmConditionSummary{
		Id:                   current.Id,
		MonitoringTemplateId: current.MonitoringTemplateId,
		Namespace:            current.Namespace,
		ResourceType:         current.ResourceType,
		MetricName:           current.MetricName,
		ConditionType:        current.ConditionType,
		Conditions:           current.Conditions,
		Status:               current.Status,
		LifecycleState:       current.LifecycleState,
		CompositeType:        current.CompositeType,
		TimeCreated:          current.TimeCreated,
		TimeUpdated:          current.TimeUpdated,
		FreeformTags:         current.FreeformTags,
		DefinedTags:          current.DefinedTags,
		SystemTags:           current.SystemTags,
	}
}

func optionalTestString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(strings.TrimSpace(value))
}

func cloneAlarmConditionStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func makeAlarmConditionDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func TestAlarmConditionRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := alarmConditionRuntimeSemantics()
	if got == nil {
		t.Fatal("alarmConditionRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	if len(got.Unsupported) != 0 {
		t.Fatalf("unsupported semantics = %#v, want no generatedruntime-blocking gaps", got.Unsupported)
	}
}

func TestAlarmConditionCreateProjectsStatusAndRetryToken(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	var createRequest stackmonitoringsdk.CreateAlarmConditionRequest
	var getRequest stackmonitoringsdk.GetAlarmConditionRequest
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		createFn: func(_ context.Context, request stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
			createRequest = request
			return stackmonitoringsdk.CreateAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			getRequest = request
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	requireAlarmConditionCreateRequest(t, createRequest, resource)
	requireAlarmConditionGetRequest(t, getRequest, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
	if resource.Status.Id != testAlarmConditionID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testAlarmConditionID)
	}
	if resource.Status.MonitoringTemplateId != testAlarmConditionMonitoringTemplate {
		t.Fatalf("status.monitoringTemplateId = %q, want %q", resource.Status.MonitoringTemplateId, testAlarmConditionMonitoringTemplate)
	}
	if string(resource.Status.OsokStatus.Ocid) != testAlarmConditionID {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testAlarmConditionID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestAlarmConditionCreateUsesProjectedResponseWhenFollowUpReadIsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	getCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		createFn: func(_ context.Context, request stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
			requireAlarmConditionCreateRequest(t, request, resource)
			return stackmonitoringsdk.CreateAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			getCalls++
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "eventual read miss")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	if getCalls != 1 {
		t.Fatalf("GetAlarmCondition calls = %d, want 1", getCalls)
	}
	if resource.Status.Id != testAlarmConditionID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testAlarmConditionID)
	}
	if resource.Status.Status != string(stackmonitoringsdk.AlarmConditionLifeCycleDetailsApplied) {
		t.Fatalf("status.sdkStatus = %q, want %q", resource.Status.Status, stackmonitoringsdk.AlarmConditionLifeCycleDetailsApplied)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestAlarmConditionBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	listCalls := 0
	createCalls := 0
	updateCalls := 0
	var pages []string
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
			listCalls++
			requireAlarmConditionListRequest(t, request, testAlarmConditionMonitoringTemplate)
			pages = append(pages, alarmConditionStringValue(request.Page))
			if listCalls == 1 {
				otherSpec := resource.Spec
				otherSpec.MetricName = "MemoryUtilization"
				return stackmonitoringsdk.ListAlarmConditionsResponse{
					AlarmConditionCollection: stackmonitoringsdk.AlarmConditionCollection{
						Items: []stackmonitoringsdk.AlarmConditionSummary{
							makeSDKAlarmConditionSummary(testOtherAlarmConditionID, testAlarmConditionMonitoringTemplate, otherSpec),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return stackmonitoringsdk.ListAlarmConditionsResponse{
				AlarmConditionCollection: stackmonitoringsdk.AlarmConditionCollection{
					Items: []stackmonitoringsdk.AlarmConditionSummary{
						makeSDKAlarmConditionSummary(testAlarmConditionID, testAlarmConditionMonitoringTemplate, resource.Spec),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
			createCalls++
			return stackmonitoringsdk.CreateAlarmConditionResponse{}, nil
		},
		updateFn: func(context.Context, stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
			updateCalls++
			return stackmonitoringsdk.UpdateAlarmConditionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	if createCalls != 0 {
		t.Fatalf("CreateAlarmCondition calls = %d, want 0 after list bind", createCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateAlarmCondition calls = %d, want 0 after matching readback", updateCalls)
	}
	if got, want := strings.Join(pages, ","), ",page-2"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
}

func TestAlarmConditionBindsExistingWithLowercaseConditionType(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Spec.ConditionType = "fixed"
	canonicalSpec := resource.Spec
	canonicalSpec.ConditionType = string(stackmonitoringsdk.ConditionTypeFixed)
	createCalls := 0
	updateCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
			requireAlarmConditionListRequest(t, request, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.ListAlarmConditionsResponse{
				AlarmConditionCollection: stackmonitoringsdk.AlarmConditionCollection{
					Items: []stackmonitoringsdk.AlarmConditionSummary{
						makeSDKAlarmConditionSummary(testAlarmConditionID, testAlarmConditionMonitoringTemplate, canonicalSpec),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					canonicalSpec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
			createCalls++
			return stackmonitoringsdk.CreateAlarmConditionResponse{}, nil
		},
		updateFn: func(context.Context, stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
			updateCalls++
			return stackmonitoringsdk.UpdateAlarmConditionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	if createCalls != 0 {
		t.Fatalf("CreateAlarmCondition calls = %d, want 0 after case-insensitive list bind", createCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateAlarmCondition calls = %d, want 0 after canonical conditionType readback", updateCalls)
	}
	if resource.Status.Id != testAlarmConditionID {
		t.Fatalf("status.id = %q, want bound %q", resource.Status.Id, testAlarmConditionID)
	}
}

func TestAlarmConditionCreateDoesNotBindCompositeTypeMismatch(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	mismatchedSpec := resource.Spec
	mismatchedSpec.CompositeType = "ocid1.stackmonitoringcompositetype.oc1..ebs"
	listCalls := 0
	createCalls := 0
	updateCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
			listCalls++
			requireAlarmConditionListRequest(t, request, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.ListAlarmConditionsResponse{
				AlarmConditionCollection: stackmonitoringsdk.AlarmConditionCollection{
					Items: []stackmonitoringsdk.AlarmConditionSummary{
						makeSDKAlarmConditionSummary(testOtherAlarmConditionID, testAlarmConditionMonitoringTemplate, mismatchedSpec),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, request stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
			createCalls++
			requireAlarmConditionCreateRequest(t, request, resource)
			return stackmonitoringsdk.CreateAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		updateFn: func(context.Context, stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
			updateCalls++
			return stackmonitoringsdk.UpdateAlarmConditionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	if listCalls != 1 {
		t.Fatalf("ListAlarmConditions calls = %d, want 1", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateAlarmCondition calls = %d, want 1 when list match differs by compositeType", createCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateAlarmCondition calls = %d, want 0 after create", updateCalls)
	}
	if resource.Status.Id != testAlarmConditionID {
		t.Fatalf("status.id = %q, want created %q", resource.Status.Id, testAlarmConditionID)
	}
}

func TestAlarmConditionNoopSkipsUpdate(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Status.Id = testAlarmConditionID
	resource.Status.MonitoringTemplateId = testAlarmConditionMonitoringTemplate
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	updateCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		updateFn: func(context.Context, stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
			updateCalls++
			return stackmonitoringsdk.UpdateAlarmConditionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	if updateCalls != 0 {
		t.Fatalf("UpdateAlarmCondition calls = %d, want 0", updateCalls)
	}
}

func TestAlarmConditionUpdateShapesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Status.Id = testAlarmConditionID
	resource.Status.MonitoringTemplateId = testAlarmConditionMonitoringTemplate
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	currentSpec := resource.Spec
	currentSpec.Conditions = append([]stackmonitoringv1beta1.AlarmConditionCondition(nil), resource.Spec.Conditions...)
	currentSpec.MetricName = "OldMetric"
	currentSpec.Conditions[0].ShouldAppendNote = true
	var updateRequest stackmonitoringsdk.UpdateAlarmConditionRequest
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					currentSpec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, request stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
			updateRequest = request
			return stackmonitoringsdk.UpdateAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	requireAlarmConditionUpdateRequest(t, updateRequest, resource)
	if updateRequest.MetricName == nil || *updateRequest.MetricName != resource.Spec.MetricName {
		t.Fatalf("update metricName = %v, want %q", updateRequest.MetricName, resource.Spec.MetricName)
	}
	if len(updateRequest.Conditions) != 1 || updateRequest.Conditions[0].ShouldAppendNote == nil || *updateRequest.Conditions[0].ShouldAppendNote {
		t.Fatalf("update conditions = %#v, want explicit shouldAppendNote=false", updateRequest.Conditions)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestAlarmConditionUpdateClearsCompositeType(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Status.Id = testAlarmConditionID
	resource.Status.MonitoringTemplateId = testAlarmConditionMonitoringTemplate
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	currentSpec := resource.Spec
	currentSpec.CompositeType = "ocid1.stackmonitoringcompositetype.oc1..ebs"
	var updateRequest stackmonitoringsdk.UpdateAlarmConditionRequest
	updateCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					currentSpec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, request stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
			updateCalls++
			updateRequest = request
			return stackmonitoringsdk.UpdateAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
				OpcRequestId: common.String("opc-clear-composite"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	requireAlarmConditionSuccess(t, response, err)
	if updateCalls != 1 {
		t.Fatalf("UpdateAlarmCondition calls = %d, want 1 for compositeType clear", updateCalls)
	}
	if got := alarmConditionStringValue(updateRequest.AlarmConditionId); got != testAlarmConditionID {
		t.Fatalf("UpdateAlarmCondition alarmConditionId = %q, want %q", got, testAlarmConditionID)
	}
	if got := alarmConditionStringValue(updateRequest.MonitoringTemplateId); got != testAlarmConditionMonitoringTemplate {
		t.Fatalf("UpdateAlarmCondition monitoringTemplateId = %q, want %q", got, testAlarmConditionMonitoringTemplate)
	}
	if updateRequest.CompositeType == nil {
		t.Fatal("UpdateAlarmCondition compositeType = nil, want explicit empty string clear")
	}
	if got := *updateRequest.CompositeType; got != "" {
		t.Fatalf("UpdateAlarmCondition compositeType = %q, want empty string clear", got)
	}
	if updateRequest.MetricName != nil {
		t.Fatalf("UpdateAlarmCondition metricName = %v, want nil for compositeType-only update", updateRequest.MetricName)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-clear-composite" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-clear-composite", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestAlarmConditionCreateOnlyParentDriftIsRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Status.Id = testAlarmConditionID
	resource.Status.MonitoringTemplateId = testAlarmConditionMonitoringTemplate
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	resource.Annotations[alarmConditionMonitoringTemplateIDAnnotation] = testOtherAlarmConditionMonitoringTmpl
	updateCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		updateFn: func(context.Context, stackmonitoringsdk.UpdateAlarmConditionRequest) (stackmonitoringsdk.UpdateAlarmConditionResponse, error) {
			updateCalls++
			return stackmonitoringsdk.UpdateAlarmConditionResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "monitoringTemplateId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want monitoringTemplateId replacement error", err)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateAlarmCondition calls = %d, want 0", updateCalls)
	}
}

func TestAlarmConditionRequiresMonitoringTemplateAnnotationBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Annotations = nil
	createCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		createFn: func(context.Context, stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
			createCalls++
			return stackmonitoringsdk.CreateAlarmConditionResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	if err == nil || !strings.Contains(err.Error(), alarmConditionMonitoringTemplateIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want monitoringTemplate annotation error", err)
	}
	if createCalls != 0 {
		t.Fatalf("CreateAlarmCondition calls = %d, want 0", createCalls)
	}
}

func TestAlarmConditionDeleteWithoutTrackedIDReleasesWithoutMonitoringTemplateAnnotation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		annotations map[string]string
	}{
		{
			name:        "annotation absent",
			annotations: nil,
		},
		{
			name:        "annotation removed",
			annotations: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeAlarmConditionResource()
			resource.Annotations = tt.annotations
			client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
				getFn: func(context.Context, stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
					t.Fatal("GetAlarmCondition should not be called for a never-bound delete")
					return stackmonitoringsdk.GetAlarmConditionResponse{}, nil
				},
				listFn: func(context.Context, stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
					t.Fatal("ListAlarmConditions should not be called for a never-bound delete")
					return stackmonitoringsdk.ListAlarmConditionsResponse{}, nil
				},
				deleteFn: func(context.Context, stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
					t.Fatal("DeleteAlarmCondition should not be called for a never-bound delete")
					return stackmonitoringsdk.DeleteAlarmConditionResponse{}, nil
				},
			})

			deleted, err := client.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if !deleted {
				t.Fatal("Delete() deleted = false, want true for never-bound resource")
			}
			if resource.Status.OsokStatus.DeletedAt == nil {
				t.Fatal("status.status.deletedAt = nil, want finalizer release marker")
			}
		})
	}
}

func TestAlarmConditionDeleteWithoutTrackedIDResolvesParentScopedListMatch(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	listCalls := 0
	deleteCalls := 0
	client := alarmConditionUntrackedDeleteMatchClient(t, resource, &listCalls, &deleteCalls)

	deleted, err := client.Delete(context.Background(), resource)
	requireAlarmConditionUntrackedDeleteMatch(t, resource, deleted, err, listCalls, deleteCalls)
}

func TestAlarmConditionDeleteWithoutTrackedIDMatchesLowercaseConditionType(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Spec.ConditionType = "fixed"
	canonicalSpec := resource.Spec
	canonicalSpec.ConditionType = string(stackmonitoringsdk.ConditionTypeFixed)
	listCalls := 0
	deleteCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
			listCalls++
			requireAlarmConditionListRequest(t, request, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.ListAlarmConditionsResponse{
				AlarmConditionCollection: stackmonitoringsdk.AlarmConditionCollection{
					Items: []stackmonitoringsdk.AlarmConditionSummary{
						makeSDKAlarmConditionSummary(testAlarmConditionID, testAlarmConditionMonitoringTemplate, canonicalSpec),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					canonicalSpec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			deleteCalls++
			requireAlarmConditionDeleteRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.DeleteAlarmConditionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	requireAlarmConditionUntrackedDeleteMatch(t, resource, deleted, err, listCalls, deleteCalls)
}

func alarmConditionUntrackedDeleteMatchClient(
	t *testing.T,
	resource *stackmonitoringv1beta1.AlarmCondition,
	listCalls *int,
	deleteCalls *int,
) AlarmConditionServiceClient {
	t.Helper()
	return testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
			(*listCalls)++
			requireAlarmConditionListRequest(t, request, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.ListAlarmConditionsResponse{
				AlarmConditionCollection: stackmonitoringsdk.AlarmConditionCollection{
					Items: []stackmonitoringsdk.AlarmConditionSummary{
						makeSDKAlarmConditionSummary(testAlarmConditionID, testAlarmConditionMonitoringTemplate, resource.Spec),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			(*deleteCalls)++
			requireAlarmConditionDeleteRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.DeleteAlarmConditionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})
}

func requireAlarmConditionUntrackedDeleteMatch(
	t *testing.T,
	resource *stackmonitoringv1beta1.AlarmCondition,
	deleted bool,
	err error,
	listCalls int,
	deleteCalls int,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until OCI delete is confirmed")
	}
	if listCalls != 1 {
		t.Fatalf("ListAlarmConditions calls = %d, want 1", listCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAlarmCondition calls = %d, want 1 for parent-scoped list match", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil before confirmed delete", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.Id != testAlarmConditionID {
		t.Fatalf("status.id = %q, want resolved list match %q", resource.Status.Id, testAlarmConditionID)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete pending", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestAlarmConditionDeleteWithoutTrackedIDIgnoresCompositeTypeMismatch(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	mismatchedSpec := resource.Spec
	mismatchedSpec.CompositeType = "ocid1.stackmonitoringcompositetype.oc1..ebs"
	listCalls := 0
	deleteCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListAlarmConditionsRequest) (stackmonitoringsdk.ListAlarmConditionsResponse, error) {
			listCalls++
			requireAlarmConditionListRequest(t, request, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.ListAlarmConditionsResponse{
				AlarmConditionCollection: stackmonitoringsdk.AlarmConditionCollection{
					Items: []stackmonitoringsdk.AlarmConditionSummary{
						makeSDKAlarmConditionSummary(testOtherAlarmConditionID, testAlarmConditionMonitoringTemplate, mismatchedSpec),
					},
				},
			}, nil
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			deleteCalls++
			t.Fatal("DeleteAlarmCondition should not be called for a compositeType mismatch")
			return stackmonitoringsdk.DeleteAlarmConditionResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when no matching OCI identity exists")
	}
	if listCalls != 1 {
		t.Fatalf("ListAlarmConditions calls = %d, want 1", listCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteAlarmCondition calls = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want finalizer release marker")
	}
	if resource.Status.Id != "" {
		t.Fatalf("status.id = %q, want empty when mismatched list item is ignored", resource.Status.Id)
	}
}

func TestAlarmConditionDeleteWithTrackedIDStillRequiresParentIdentity(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Annotations = nil
	resource.Status.Id = testAlarmConditionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			t.Fatal("DeleteAlarmCondition should not be called without the tracked parent identity")
			return stackmonitoringsdk.DeleteAlarmConditionResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), alarmConditionMonitoringTemplateIDAnnotation) {
		t.Fatalf("Delete() error = %v, want monitoringTemplate annotation error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for tracked resource with missing parent identity")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil for tracked resource with missing parent identity", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestAlarmConditionDeleteKeepsFinalizerUntilConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Status.Id = testAlarmConditionID
	resource.Status.MonitoringTemplateId = testAlarmConditionMonitoringTemplate
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	deleteCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					stackmonitoringsdk.AlarmConditionLifeCycleStatesActive,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			deleteCalls++
			if got := alarmConditionStringValue(request.AlarmConditionId); got != testAlarmConditionID {
				t.Fatalf("DeleteAlarmCondition alarmConditionId = %q, want %q", got, testAlarmConditionID)
			}
			if got := alarmConditionStringValue(request.MonitoringTemplateId); got != testAlarmConditionMonitoringTemplate {
				t.Fatalf("DeleteAlarmCondition monitoringTemplateId = %q, want %q", got, testAlarmConditionMonitoringTemplate)
			}
			return stackmonitoringsdk.DeleteAlarmConditionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAlarmCondition calls = %d, want 1", deleteCalls)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete pending", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestAlarmConditionDeleteWaitsForWriteLifecycleBeforeDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		state         stackmonitoringsdk.AlarmConditionLifeCycleStatesEnum
		wantPhase     shared.OSOKAsyncPhase
		wantCondition shared.OSOKConditionType
	}{
		{
			name:          "creating",
			state:         stackmonitoringsdk.AlarmConditionLifeCycleStatesCreating,
			wantPhase:     shared.OSOKAsyncPhaseCreate,
			wantCondition: shared.Provisioning,
		},
		{
			name:          "updating",
			state:         stackmonitoringsdk.AlarmConditionLifeCycleStatesUpdating,
			wantPhase:     shared.OSOKAsyncPhaseUpdate,
			wantCondition: shared.Updating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runAlarmConditionPendingWriteDeleteCase(t, tt.state, tt.wantPhase, tt.wantCondition)
		})
	}
}

func runAlarmConditionPendingWriteDeleteCase(
	t *testing.T,
	state stackmonitoringsdk.AlarmConditionLifeCycleStatesEnum,
	wantPhase shared.OSOKAsyncPhase,
	wantCondition shared.OSOKConditionType,
) {
	t.Helper()

	resource := makeAlarmConditionResource()
	resource.Status.Id = testAlarmConditionID
	resource.Status.MonitoringTemplateId = testAlarmConditionMonitoringTemplate
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	getCalls := 0
	deleteCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			getCalls++
			requireAlarmConditionGetRequest(t, request, testAlarmConditionID, testAlarmConditionMonitoringTemplate)
			return stackmonitoringsdk.GetAlarmConditionResponse{
				AlarmCondition: makeSDKAlarmCondition(
					testAlarmConditionID,
					testAlarmConditionMonitoringTemplate,
					resource.Spec,
					state,
				),
			}, nil
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			deleteCalls++
			return stackmonitoringsdk.DeleteAlarmConditionResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while write lifecycle is pending")
	}
	if getCalls != 1 {
		t.Fatalf("GetAlarmCondition calls = %d, want 1", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteAlarmCondition calls = %d, want 0", deleteCalls)
	}
	if got := resource.Status.LifecycleState; got != string(state) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, state)
	}
	requireAlarmConditionPendingWriteAsync(t, resource, wantPhase, state)
	if got := resource.Status.OsokStatus.Reason; got != string(wantCondition) {
		t.Fatalf("status.status.reason = %q, want %q", got, wantCondition)
	}
}

func TestAlarmConditionDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	resource.Status.Id = testAlarmConditionID
	resource.Status.MonitoringTemplateId = testAlarmConditionMonitoringTemplate
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmConditionID)
	deleteCalls := 0
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		getFn: func(context.Context, stackmonitoringsdk.GetAlarmConditionRequest) (stackmonitoringsdk.GetAlarmConditionResponse, error) {
			err := errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "ambiguous")
			err.OpcRequestID = "opc-auth"
			return stackmonitoringsdk.GetAlarmConditionResponse{}, err
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteAlarmConditionRequest) (stackmonitoringsdk.DeleteAlarmConditionResponse, error) {
			deleteCalls++
			return stackmonitoringsdk.DeleteAlarmConditionResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped confirm-read error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteAlarmCondition calls = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-auth" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-auth", resource.Status.OsokStatus.OpcRequestID)
	}
}

func requireAlarmConditionPendingWriteAsync(
	t *testing.T,
	resource *stackmonitoringv1beta1.AlarmCondition,
	phase shared.OSOKAsyncPhase,
	state stackmonitoringsdk.AlarmConditionLifeCycleStatesEnum,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending write lifecycle")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != phase {
		t.Fatalf("status.status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.RawStatus != string(state) {
		t.Fatalf("status.status.async.current.rawStatus = %q, want %q", current.RawStatus, state)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func TestAlarmConditionRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeAlarmConditionResource()
	client := testAlarmConditionClient(&fakeAlarmConditionOCIClient{
		createFn: func(context.Context, stackmonitoringsdk.CreateAlarmConditionRequest) (stackmonitoringsdk.CreateAlarmConditionResponse, error) {
			err := errortest.NewServiceError(500, "InternalError", "failed")
			err.OpcRequestID = "opc-failed-create"
			return stackmonitoringsdk.CreateAlarmConditionResponse{}, err
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeAlarmConditionRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-failed-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-failed-create", resource.Status.OsokStatus.OpcRequestID)
	}
}

func requireAlarmConditionSuccess(t *testing.T, response servicemanager.OSOKResponse, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("operation error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("operation response = %#v, want successful", response)
	}
}

func requireAlarmConditionCreateRequest(
	t *testing.T,
	request stackmonitoringsdk.CreateAlarmConditionRequest,
	resource *stackmonitoringv1beta1.AlarmCondition,
) {
	t.Helper()
	requireAlarmConditionCreateIdentity(t, request, resource)
	requireAlarmConditionCreateConditions(t, request)
}

func requireAlarmConditionCreateIdentity(
	t *testing.T,
	request stackmonitoringsdk.CreateAlarmConditionRequest,
	resource *stackmonitoringv1beta1.AlarmCondition,
) {
	t.Helper()
	if got := alarmConditionStringValue(request.MonitoringTemplateId); got != testAlarmConditionMonitoringTemplate {
		t.Fatalf("CreateAlarmCondition monitoringTemplateId = %q, want %q", got, testAlarmConditionMonitoringTemplate)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateAlarmCondition opc-retry-token is empty")
	}
	if got := alarmConditionStringValue(request.Namespace); got != resource.Spec.Namespace {
		t.Fatalf("CreateAlarmCondition namespace = %q, want %q", got, resource.Spec.Namespace)
	}
	if got := alarmConditionStringValue(request.ResourceType); got != resource.Spec.ResourceType {
		t.Fatalf("CreateAlarmCondition resourceType = %q, want %q", got, resource.Spec.ResourceType)
	}
	if got := request.ConditionType; got != stackmonitoringsdk.ConditionTypeFixed {
		t.Fatalf("CreateAlarmCondition conditionType = %q, want %q", got, stackmonitoringsdk.ConditionTypeFixed)
	}
}

func requireAlarmConditionCreateConditions(t *testing.T, request stackmonitoringsdk.CreateAlarmConditionRequest) {
	t.Helper()
	if len(request.Conditions) != 1 {
		t.Fatalf("CreateAlarmCondition conditions len = %d, want 1", len(request.Conditions))
	}
	if request.Conditions[0].ShouldAppendNote == nil || *request.Conditions[0].ShouldAppendNote {
		t.Fatalf("CreateAlarmCondition shouldAppendNote = %v, want explicit false", request.Conditions[0].ShouldAppendNote)
	}
	if request.Conditions[0].ShouldAppendUrl == nil || !*request.Conditions[0].ShouldAppendUrl {
		t.Fatalf("CreateAlarmCondition shouldAppendUrl = %v, want explicit true", request.Conditions[0].ShouldAppendUrl)
	}
}

func requireAlarmConditionGetRequest(
	t *testing.T,
	request stackmonitoringsdk.GetAlarmConditionRequest,
	alarmConditionID string,
	monitoringTemplateID string,
) {
	t.Helper()
	if got := alarmConditionStringValue(request.AlarmConditionId); got != alarmConditionID {
		t.Fatalf("GetAlarmCondition alarmConditionId = %q, want %q", got, alarmConditionID)
	}
	if got := alarmConditionStringValue(request.MonitoringTemplateId); got != monitoringTemplateID {
		t.Fatalf("GetAlarmCondition monitoringTemplateId = %q, want %q", got, monitoringTemplateID)
	}
}

func requireAlarmConditionDeleteRequest(
	t *testing.T,
	request stackmonitoringsdk.DeleteAlarmConditionRequest,
	alarmConditionID string,
	monitoringTemplateID string,
) {
	t.Helper()
	if got := alarmConditionStringValue(request.AlarmConditionId); got != alarmConditionID {
		t.Fatalf("DeleteAlarmCondition alarmConditionId = %q, want %q", got, alarmConditionID)
	}
	if got := alarmConditionStringValue(request.MonitoringTemplateId); got != monitoringTemplateID {
		t.Fatalf("DeleteAlarmCondition monitoringTemplateId = %q, want %q", got, monitoringTemplateID)
	}
}

func requireAlarmConditionListRequest(
	t *testing.T,
	request stackmonitoringsdk.ListAlarmConditionsRequest,
	monitoringTemplateID string,
) {
	t.Helper()
	if got := alarmConditionStringValue(request.MonitoringTemplateId); got != monitoringTemplateID {
		t.Fatalf("ListAlarmConditions monitoringTemplateId = %q, want %q", got, monitoringTemplateID)
	}
}

func requireAlarmConditionUpdateRequest(
	t *testing.T,
	request stackmonitoringsdk.UpdateAlarmConditionRequest,
	resource *stackmonitoringv1beta1.AlarmCondition,
) {
	t.Helper()
	if got := alarmConditionStringValue(request.AlarmConditionId); got != testAlarmConditionID {
		t.Fatalf("UpdateAlarmCondition alarmConditionId = %q, want %q", got, testAlarmConditionID)
	}
	if got := alarmConditionStringValue(request.MonitoringTemplateId); got != testAlarmConditionMonitoringTemplate {
		t.Fatalf("UpdateAlarmCondition monitoringTemplateId = %q, want %q", got, testAlarmConditionMonitoringTemplate)
	}
	if request.MetricName == nil || *request.MetricName != resource.Spec.MetricName {
		t.Fatalf("UpdateAlarmCondition metricName = %v, want %q", request.MetricName, resource.Spec.MetricName)
	}
}
