/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alarmsuppression

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	monitoringsdk "github.com/oracle/oci-go-sdk/v65/monitoring"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAlarmSuppressionID = "ocid1.alarmsuppression.oc1..example"
	testAlarmID            = "ocid1.alarm.oc1..example"
	testCompartmentID      = "ocid1.compartment.oc1..example"
	testSuppressFrom       = "2026-01-02T03:04:05Z"
	testSuppressUntil      = "2026-01-02T04:04:05Z"
)

type alarmSuppressionOCIClient interface {
	CreateAlarmSuppression(context.Context, monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error)
	GetAlarmSuppression(context.Context, monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error)
	ListAlarmSuppressions(context.Context, monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error)
	DeleteAlarmSuppression(context.Context, monitoringsdk.DeleteAlarmSuppressionRequest) (monitoringsdk.DeleteAlarmSuppressionResponse, error)
}

type fakeAlarmSuppressionOCIClient struct {
	createFn func(context.Context, monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error)
	getFn    func(context.Context, monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error)
	listFn   func(context.Context, monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error)
	deleteFn func(context.Context, monitoringsdk.DeleteAlarmSuppressionRequest) (monitoringsdk.DeleteAlarmSuppressionResponse, error)
}

func (f *fakeAlarmSuppressionOCIClient) CreateAlarmSuppression(ctx context.Context, req monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return monitoringsdk.CreateAlarmSuppressionResponse{}, nil
}

func (f *fakeAlarmSuppressionOCIClient) GetAlarmSuppression(ctx context.Context, req monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return monitoringsdk.GetAlarmSuppressionResponse{}, nil
}

func (f *fakeAlarmSuppressionOCIClient) ListAlarmSuppressions(ctx context.Context, req monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return monitoringsdk.ListAlarmSuppressionsResponse{}, nil
}

func (f *fakeAlarmSuppressionOCIClient) DeleteAlarmSuppression(ctx context.Context, req monitoringsdk.DeleteAlarmSuppressionRequest) (monitoringsdk.DeleteAlarmSuppressionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return monitoringsdk.DeleteAlarmSuppressionResponse{}, nil
}

func newTestAlarmSuppressionClient(client alarmSuppressionOCIClient) AlarmSuppressionServiceClient {
	if client == nil {
		client = &fakeAlarmSuppressionOCIClient{}
	}

	delegate := defaultAlarmSuppressionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*monitoringv1beta1.AlarmSuppression](generatedruntime.Config[*monitoringv1beta1.AlarmSuppression]{
			Kind:            "AlarmSuppression",
			SDKName:         "AlarmSuppression",
			Log:             loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
			Semantics:       alarmSuppressionRuntimeSemantics(),
			BuildCreateBody: buildAlarmSuppressionCreateBody,
			Identity: generatedruntime.IdentityHooks[*monitoringv1beta1.AlarmSuppression]{
				GuardExistingBeforeCreate: guardAlarmSuppressionExistingBeforeCreate,
			},
			StatusHooks: generatedruntime.StatusHooks[*monitoringv1beta1.AlarmSuppression]{
				MarkTerminating: markAlarmSuppressionTerminating,
			},
			ParityHooks: generatedruntime.ParityHooks[*monitoringv1beta1.AlarmSuppression]{
				NormalizeDesiredState: normalizeAlarmSuppressionDesiredState,
			},
			DeleteHooks: generatedruntime.DeleteHooks[*monitoringv1beta1.AlarmSuppression]{
				ApplyOutcome: applyAlarmSuppressionDeleteOutcome,
			},
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.CreateAlarmSuppressionRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.CreateAlarmSuppression(ctx, *request.(*monitoringsdk.CreateAlarmSuppressionRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateAlarmSuppressionDetails", RequestName: "CreateAlarmSuppressionDetails", Contribution: "body"},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.GetAlarmSuppressionRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.GetAlarmSuppression(ctx, *request.(*monitoringsdk.GetAlarmSuppressionRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "AlarmSuppressionId", RequestName: "alarmSuppressionId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.ListAlarmSuppressionsRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.ListAlarmSuppressions(ctx, *request.(*monitoringsdk.ListAlarmSuppressionsRequest))
				},
				Fields: alarmSuppressionListFields(),
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.DeleteAlarmSuppressionRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.DeleteAlarmSuppression(ctx, *request.(*monitoringsdk.DeleteAlarmSuppressionRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "AlarmSuppressionId", RequestName: "alarmSuppressionId", Contribution: "path", PreferResourceID: true},
				},
			},
		}),
	}
	return wrapAlarmSuppressionCanonicalizingClient(delegate)
}

func makeAlarmSuppressionResource() *monitoringv1beta1.AlarmSuppression {
	return &monitoringv1beta1.AlarmSuppression{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-alarm-suppression", Namespace: "default"},
		Spec: monitoringv1beta1.AlarmSuppressionSpec{
			AlarmSuppressionTarget: monitoringv1beta1.AlarmSuppressionTarget{AlarmId: testAlarmID},
			DisplayName:            "sample-alarm-suppression",
			Dimensions:             map[string]string{"resourceId": "ocid1.instance.oc1..example"},
			TimeSuppressFrom:       testSuppressFrom,
			TimeSuppressUntil:      testSuppressUntil,
			Description:            "planned maintenance",
			FreeformTags:           map[string]string{"owner": "osok"},
			DefinedTags:            map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKAlarmSuppression(t *testing.T, id string, resource *monitoringv1beta1.AlarmSuppression, lifecycleState monitoringsdk.AlarmSuppressionLifecycleStateEnum) monitoringsdk.AlarmSuppression {
	t.Helper()

	target, err := alarmSuppressionSDKTarget(resource.Spec.AlarmSuppressionTarget)
	if err != nil {
		t.Fatalf("alarmSuppressionSDKTarget() error = %v", err)
	}
	return monitoringsdk.AlarmSuppression{
		Id:                     common.String(id),
		CompartmentId:          common.String(testCompartmentID),
		AlarmSuppressionTarget: target,
		DisplayName:            common.String(resource.Spec.DisplayName),
		Dimensions:             cloneStringMap(resource.Spec.Dimensions),
		TimeSuppressFrom:       mustSDKTime(t, resource.Spec.TimeSuppressFrom),
		TimeSuppressUntil:      mustSDKTime(t, resource.Spec.TimeSuppressUntil),
		LifecycleState:         lifecycleState,
		TimeCreated:            mustSDKTime(t, resource.Spec.TimeSuppressFrom),
		TimeUpdated:            mustSDKTime(t, resource.Spec.TimeSuppressFrom),
		Description:            optionalString(resource.Spec.Description),
		FreeformTags:           cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:            sdkDefinedTags(resource.Spec.DefinedTags),
	}
}

func makeSDKAlarmSuppressionSummary(t *testing.T, id string, resource *monitoringv1beta1.AlarmSuppression, lifecycleState monitoringsdk.AlarmSuppressionLifecycleStateEnum) monitoringsdk.AlarmSuppressionSummary {
	t.Helper()

	target, err := alarmSuppressionSDKTarget(resource.Spec.AlarmSuppressionTarget)
	if err != nil {
		t.Fatalf("alarmSuppressionSDKTarget() error = %v", err)
	}
	return monitoringsdk.AlarmSuppressionSummary{
		Id:                     common.String(id),
		CompartmentId:          common.String(testCompartmentID),
		AlarmSuppressionTarget: target,
		DisplayName:            common.String(resource.Spec.DisplayName),
		Dimensions:             cloneStringMap(resource.Spec.Dimensions),
		TimeSuppressFrom:       mustSDKTime(t, resource.Spec.TimeSuppressFrom),
		TimeSuppressUntil:      mustSDKTime(t, resource.Spec.TimeSuppressUntil),
		LifecycleState:         lifecycleState,
		TimeCreated:            mustSDKTime(t, resource.Spec.TimeSuppressFrom),
		TimeUpdated:            mustSDKTime(t, resource.Spec.TimeSuppressFrom),
		Description:            optionalString(resource.Spec.Description),
		FreeformTags:           cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:            sdkDefinedTags(resource.Spec.DefinedTags),
	}
}

func mustSDKTime(t *testing.T, value string) *common.SDKTime {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return &common.SDKTime{Time: parsed}
}

func TestAlarmSuppressionServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	var createRequest monitoringsdk.CreateAlarmSuppressionRequest
	listCalls := 0
	getCalls := 0

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		listFn: func(_ context.Context, req monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
			listCalls++
			if got := stringValue(req.AlarmId); got != testAlarmID {
				t.Fatalf("ListAlarmSuppressionsRequest.AlarmId = %q, want %q", got, testAlarmID)
			}
			if got := stringValue(req.DisplayName); got != resource.Spec.DisplayName {
				t.Fatalf("ListAlarmSuppressionsRequest.DisplayName = %q, want %q", got, resource.Spec.DisplayName)
			}
			return monitoringsdk.ListAlarmSuppressionsResponse{}, nil
		},
		createFn: func(_ context.Context, req monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
			createRequest = req
			return monitoringsdk.CreateAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			getCalls++
			if got := stringValue(req.AlarmSuppressionId); got != testAlarmSuppressionID {
				t.Fatalf("GetAlarmSuppressionRequest.AlarmSuppressionId = %q, want %q", got, testAlarmSuppressionID)
			}
			return monitoringsdk.GetAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListAlarmSuppressions() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetAlarmSuppression() calls = %d, want 1 follow-up read", getCalls)
	}
	target, ok := createRequest.CreateAlarmSuppressionDetails.AlarmSuppressionTarget.(monitoringsdk.AlarmSuppressionAlarmTarget)
	if !ok {
		t.Fatalf("CreateAlarmSuppressionDetails.AlarmSuppressionTarget = %T, want AlarmSuppressionAlarmTarget", createRequest.CreateAlarmSuppressionDetails.AlarmSuppressionTarget)
	}
	if got := stringValue(target.AlarmId); got != testAlarmID {
		t.Fatalf("CreateAlarmSuppressionDetails.AlarmId = %q, want %q", got, testAlarmID)
	}
	if got := stringValue(createRequest.CreateAlarmSuppressionDetails.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("CreateAlarmSuppressionDetails.DisplayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testAlarmSuppressionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testAlarmSuppressionID)
	}
	if got := resource.Status.Id; got != testAlarmSuppressionID {
		t.Fatalf("status.id = %q, want %q", got, testAlarmSuppressionID)
	}
	if got := resource.Status.LifecycleState; got != string(monitoringsdk.AlarmSuppressionLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestAlarmSuppressionServiceClientCreatesWithJsonDataTarget(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Spec.AlarmSuppressionTarget = monitoringv1beta1.AlarmSuppressionTarget{
		JsonData: `{"targetType":"ALARM","alarmId":"` + testAlarmID + `"}`,
	}
	var createRequest monitoringsdk.CreateAlarmSuppressionRequest
	listCalls := 0

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		listFn: func(_ context.Context, req monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
			listCalls++
			if got := stringValue(req.AlarmId); got != testAlarmID {
				t.Fatalf("ListAlarmSuppressionsRequest.AlarmId = %q, want %q", got, testAlarmID)
			}
			if got := stringValue(req.DisplayName); got != resource.Spec.DisplayName {
				t.Fatalf("ListAlarmSuppressionsRequest.DisplayName = %q, want %q", got, resource.Spec.DisplayName)
			}
			return monitoringsdk.ListAlarmSuppressionsResponse{}, nil
		},
		createFn: func(_ context.Context, req monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
			createRequest = req
			return monitoringsdk.CreateAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
			}, nil
		},
		getFn: func(context.Context, monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			return monitoringsdk.GetAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListAlarmSuppressions() calls = %d, want 1", listCalls)
	}
	target, ok := createRequest.CreateAlarmSuppressionDetails.AlarmSuppressionTarget.(monitoringsdk.AlarmSuppressionAlarmTarget)
	if !ok {
		t.Fatalf("CreateAlarmSuppressionDetails.AlarmSuppressionTarget = %T, want AlarmSuppressionAlarmTarget", createRequest.CreateAlarmSuppressionDetails.AlarmSuppressionTarget)
	}
	if got := stringValue(target.AlarmId); got != testAlarmID {
		t.Fatalf("CreateAlarmSuppressionDetails.AlarmId = %q, want %q", got, testAlarmID)
	}
	if got := resource.Spec.AlarmSuppressionTarget.JsonData; got != "" {
		t.Fatalf("spec.alarmSuppressionTarget.jsonData = %q, want normalized empty helper field", got)
	}
}

func TestAlarmSuppressionServiceClientJsonDataTargetSecondReconcileIsIdempotent(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Spec.AlarmSuppressionTarget = monitoringv1beta1.AlarmSuppressionTarget{
		JsonData: `{"targetType":"ALARM","alarmId":"` + testAlarmID + `"}`,
	}
	createCalls := 0
	getCalls := 0

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		listFn: func(context.Context, monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
			return monitoringsdk.ListAlarmSuppressionsResponse{}, nil
		},
		createFn: func(context.Context, monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
			createCalls++
			return monitoringsdk.CreateAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
			}, nil
		},
		getFn: func(context.Context, monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			getCalls++
			return monitoringsdk.GetAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("first CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("first CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateAlarmSuppression() calls after first reconcile = %d, want 1", createCalls)
	}

	nextResource := makeAlarmSuppressionResource()
	nextResource.Spec.AlarmSuppressionTarget = monitoringv1beta1.AlarmSuppressionTarget{
		JsonData: `{"targetType":"ALARM","alarmId":"` + testAlarmID + `"}`,
	}
	nextResource.Status.OsokStatus.Ocid = shared.OCID(testAlarmSuppressionID)

	response, err = client.CreateOrUpdate(context.Background(), nextResource, ctrl.Request{})
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want successful non-requeue observe", response)
	}
	if createCalls != 1 {
		t.Fatalf("CreateAlarmSuppression() calls after second reconcile = %d, want still 1", createCalls)
	}
	if getCalls < 2 {
		t.Fatalf("GetAlarmSuppression() calls = %d, want create follow-up plus tracked readback", getCalls)
	}
	if got := nextResource.Spec.AlarmSuppressionTarget.JsonData; got != "" {
		t.Fatalf("second reconcile spec.alarmSuppressionTarget.jsonData = %q, want normalized empty helper field", got)
	}
}

func TestAlarmSuppressionServiceClientRejectsJsonDataTargetWithoutAlarmIDBeforeList(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Spec.AlarmSuppressionTarget = monitoringv1beta1.AlarmSuppressionTarget{
		JsonData: `{"targetType":"ALARM"}`,
	}
	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		listFn: func(context.Context, monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
			t.Fatal("ListAlarmSuppressions() should not be called when jsonData lacks alarmId")
			return monitoringsdk.ListAlarmSuppressionsResponse{}, nil
		},
		createFn: func(context.Context, monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
			t.Fatal("CreateAlarmSuppression() should not be called when jsonData lacks alarmId")
			return monitoringsdk.CreateAlarmSuppressionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want alarmId validation error")
	}
	if !strings.Contains(err.Error(), "alarmSuppressionTarget.alarmId is required") {
		t.Fatalf("CreateOrUpdate() error = %v, want alarmId validation error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful validation failure", response)
	}
}

func TestAlarmSuppressionServiceClientRejectsUnsupportedCompartmentTargetBeforeList(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Spec.AlarmSuppressionTarget = monitoringv1beta1.AlarmSuppressionTarget{
		JsonData: `{"targetType":"COMPARTMENT","alarmId":"` + testAlarmID + `"}`,
	}
	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		listFn: func(context.Context, monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
			t.Fatal("ListAlarmSuppressions() should not be called for unsupported COMPARTMENT target")
			return monitoringsdk.ListAlarmSuppressionsResponse{}, nil
		},
		createFn: func(context.Context, monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
			t.Fatal("CreateAlarmSuppression() should not be called for unsupported COMPARTMENT target")
			return monitoringsdk.CreateAlarmSuppressionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported targetType validation error")
	}
	if !strings.Contains(err.Error(), `unsupported alarmSuppressionTarget.targetType "COMPARTMENT"`) {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported COMPARTMENT target error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful validation failure", response)
	}
}

func TestAlarmSuppressionServiceClientBindsExistingByList(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	createCalled := false
	getCalls := 0

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		listFn: func(context.Context, monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
			other := makeAlarmSuppressionResource()
			other.Spec.DisplayName = "different-suppression"
			return monitoringsdk.ListAlarmSuppressionsResponse{
				AlarmSuppressionCollection: monitoringsdk.AlarmSuppressionCollection{
					Items: []monitoringsdk.AlarmSuppressionSummary{
						makeSDKAlarmSuppressionSummary(t, "ocid1.alarmsuppression.oc1..other", other, monitoringsdk.AlarmSuppressionLifecycleStateActive),
						makeSDKAlarmSuppressionSummary(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			getCalls++
			if got := stringValue(req.AlarmSuppressionId); got != testAlarmSuppressionID {
				t.Fatalf("GetAlarmSuppressionRequest.AlarmSuppressionId = %q, want %q", got, testAlarmSuppressionID)
			}
			return monitoringsdk.GetAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
			createCalled = true
			return monitoringsdk.CreateAlarmSuppressionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if createCalled {
		t.Fatal("CreateAlarmSuppression() should not be called when list lookup finds a matching suppression")
	}
	if getCalls != 1 {
		t.Fatalf("GetAlarmSuppression() calls = %d, want 1 live follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testAlarmSuppressionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testAlarmSuppressionID)
	}
}

func TestAlarmSuppressionServiceClientCanonicalizesSuppressTimesForBindAndSecondReconcile(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Spec.TimeSuppressFrom = "2026-01-02T03:04:05.600Z"
	resource.Spec.TimeSuppressUntil = "2026-01-02T04:04:05.600Z"
	current := makeAlarmSuppressionResource()
	current.Spec.TimeSuppressFrom = "2026-01-02T03:04:05.6Z"
	current.Spec.TimeSuppressUntil = "2026-01-02T04:04:05.6Z"

	createCalled := false
	getCalls := 0

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		listFn: func(context.Context, monitoringsdk.ListAlarmSuppressionsRequest) (monitoringsdk.ListAlarmSuppressionsResponse, error) {
			return monitoringsdk.ListAlarmSuppressionsResponse{
				AlarmSuppressionCollection: monitoringsdk.AlarmSuppressionCollection{
					Items: []monitoringsdk.AlarmSuppressionSummary{
						makeSDKAlarmSuppressionSummary(t, testAlarmSuppressionID, current, monitoringsdk.AlarmSuppressionLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(context.Context, monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			getCalls++
			return monitoringsdk.GetAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, current, monitoringsdk.AlarmSuppressionLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, monitoringsdk.CreateAlarmSuppressionRequest) (monitoringsdk.CreateAlarmSuppressionResponse, error) {
			createCalled = true
			return monitoringsdk.CreateAlarmSuppressionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() bind error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() bind response = %#v, want successful non-requeue bind", response)
	}
	if createCalled {
		t.Fatal("CreateAlarmSuppression() should not be called when canonicalized list lookup finds a matching suppression")
	}
	if got := resource.Spec.TimeSuppressFrom; got != current.Spec.TimeSuppressFrom {
		t.Fatalf("spec.timeSuppressFrom after bind = %q, want canonical %q", got, current.Spec.TimeSuppressFrom)
	}
	if got := resource.Spec.TimeSuppressUntil; got != current.Spec.TimeSuppressUntil {
		t.Fatalf("spec.timeSuppressUntil after bind = %q, want canonical %q", got, current.Spec.TimeSuppressUntil)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testAlarmSuppressionID {
		t.Fatalf("status.status.ocid after bind = %q, want %q", got, testAlarmSuppressionID)
	}

	nextResource := makeAlarmSuppressionResource()
	nextResource.Spec.TimeSuppressFrom = "2026-01-02T03:04:05.600Z"
	nextResource.Spec.TimeSuppressUntil = "2026-01-02T04:04:05.600Z"
	nextResource.Status.OsokStatus.Ocid = shared.OCID(testAlarmSuppressionID)

	response, err = client.CreateOrUpdate(context.Background(), nextResource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() second reconcile error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() second reconcile response = %#v, want successful non-requeue observe", response)
	}
	if got := nextResource.Spec.TimeSuppressFrom; got != current.Spec.TimeSuppressFrom {
		t.Fatalf("second reconcile spec.timeSuppressFrom = %q, want canonical %q", got, current.Spec.TimeSuppressFrom)
	}
	if got := nextResource.Spec.TimeSuppressUntil; got != current.Spec.TimeSuppressUntil {
		t.Fatalf("second reconcile spec.timeSuppressUntil = %q, want canonical %q", got, current.Spec.TimeSuppressUntil)
	}
	if getCalls != 2 {
		t.Fatalf("GetAlarmSuppression() calls = %d, want bind follow-up plus tracked readback", getCalls)
	}
}

func TestAlarmSuppressionServiceClientRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmSuppressionID)

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		getFn: func(context.Context, monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			current := makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, monitoringsdk.AlarmSuppressionLifecycleStateActive)
			current.DisplayName = common.String("previous-display-name")
			return monitoringsdk.GetAlarmSuppressionResponse{AlarmSuppression: current}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
	}
	if got := lastConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestAlarmSuppressionServiceClientMapsLifecycleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		lifecycle     monitoringsdk.AlarmSuppressionLifecycleStateEnum
		wantSuccess   bool
		wantRequeue   bool
		wantCondition shared.OSOKConditionType
	}{
		{
			name:          "active",
			lifecycle:     monitoringsdk.AlarmSuppressionLifecycleStateActive,
			wantSuccess:   true,
			wantCondition: shared.Active,
		},
		{
			name:          "intermediate delete",
			lifecycle:     monitoringsdk.AlarmSuppressionLifecycleStateEnum("DELETING"),
			wantSuccess:   true,
			wantRequeue:   true,
			wantCondition: shared.Terminating,
		},
		{
			name:          "unmodeled failure",
			lifecycle:     monitoringsdk.AlarmSuppressionLifecycleStateEnum("FAILED"),
			wantSuccess:   false,
			wantCondition: shared.Failed,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeAlarmSuppressionResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmSuppressionID)
			client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
				getFn: func(context.Context, monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
					return monitoringsdk.GetAlarmSuppressionResponse{
						AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, tt.lifecycle),
					}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tt.wantSuccess || response.ShouldRequeue != tt.wantRequeue {
				t.Fatalf("CreateOrUpdate() response = %#v, want success=%t requeue=%t", response, tt.wantSuccess, tt.wantRequeue)
			}
			if got := lastConditionType(resource); got != tt.wantCondition {
				t.Fatalf("last condition = %q, want %q", got, tt.wantCondition)
			}
		})
	}
}

func TestAlarmSuppressionServiceClientDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmSuppressionID)
	getCalls := 0
	deleteCalls := 0

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			getCalls++
			if got := stringValue(req.AlarmSuppressionId); got != testAlarmSuppressionID {
				t.Fatalf("GetAlarmSuppressionRequest.AlarmSuppressionId = %q, want %q", got, testAlarmSuppressionID)
			}
			lifecycleState := monitoringsdk.AlarmSuppressionLifecycleStateActive
			if getCalls >= 3 {
				lifecycleState = monitoringsdk.AlarmSuppressionLifecycleStateDeleted
			}
			return monitoringsdk.GetAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, lifecycleState),
			}, nil
		},
		deleteFn: func(_ context.Context, req monitoringsdk.DeleteAlarmSuppressionRequest) (monitoringsdk.DeleteAlarmSuppressionResponse, error) {
			deleteCalls++
			if got := stringValue(req.AlarmSuppressionId); got != testAlarmSuppressionID {
				t.Fatalf("DeleteAlarmSuppressionRequest.AlarmSuppressionId = %q, want %q", got, testAlarmSuppressionID)
			}
			return monitoringsdk.DeleteAlarmSuppressionResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAlarmSuppression() calls = %d, want 1", deleteCalls)
	}
	if got := lastConditionType(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete ||
		resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending delete", resource.Status.OsokStatus.Async.Current)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("second Delete() deleted = false, want finalizer release after DELETED confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAlarmSuppression() calls = %d, want still 1 after DELETED confirmation", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp after confirmation")
	}
}

func TestAlarmSuppressionServiceClientDeleteDoesNotReissueWhilePendingActive(t *testing.T) {
	t.Parallel()

	resource := makeAlarmSuppressionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testAlarmSuppressionID)
	getCalls := 0
	deleteCalls := 0

	client := newTestAlarmSuppressionClient(&fakeAlarmSuppressionOCIClient{
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmSuppressionRequest) (monitoringsdk.GetAlarmSuppressionResponse, error) {
			getCalls++
			if got := stringValue(req.AlarmSuppressionId); got != testAlarmSuppressionID {
				t.Fatalf("GetAlarmSuppressionRequest.AlarmSuppressionId = %q, want %q", got, testAlarmSuppressionID)
			}
			lifecycleState := monitoringsdk.AlarmSuppressionLifecycleStateActive
			if getCalls >= 5 {
				lifecycleState = monitoringsdk.AlarmSuppressionLifecycleStateDeleted
			}
			return monitoringsdk.GetAlarmSuppressionResponse{
				AlarmSuppression: makeSDKAlarmSuppression(t, testAlarmSuppressionID, resource, lifecycleState),
			}, nil
		},
		deleteFn: func(_ context.Context, req monitoringsdk.DeleteAlarmSuppressionRequest) (monitoringsdk.DeleteAlarmSuppressionResponse, error) {
			deleteCalls++
			if got := stringValue(req.AlarmSuppressionId); got != testAlarmSuppressionID {
				t.Fatalf("DeleteAlarmSuppressionRequest.AlarmSuppressionId = %q, want %q", got, testAlarmSuppressionID)
			}
			return monitoringsdk.DeleteAlarmSuppressionResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	for attempt := 1; attempt <= 3; attempt++ {
		deleted, err := client.Delete(context.Background(), resource)
		if err != nil {
			t.Fatalf("Delete() attempt %d error = %v", attempt, err)
		}
		if deleted {
			t.Fatalf("Delete() attempt %d deleted = true, want finalizer retained while readback remains ACTIVE", attempt)
		}
		if deleteCalls != 1 {
			t.Fatalf("DeleteAlarmSuppression() calls after attempt %d = %d, want 1", attempt, deleteCalls)
		}
		if resource.Status.OsokStatus.Async.Current == nil ||
			resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete ||
			resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
			t.Fatalf("status.async.current after attempt %d = %#v, want pending delete", attempt, resource.Status.OsokStatus.Async.Current)
		}
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() final attempt error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() final attempt deleted = false, want finalizer release after DELETED confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAlarmSuppression() calls after final attempt = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp after confirmation")
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func lastConditionType(resource *monitoringv1beta1.AlarmSuppression) shared.OSOKConditionType {
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return ""
	}
	return conditions[len(conditions)-1].Type
}
