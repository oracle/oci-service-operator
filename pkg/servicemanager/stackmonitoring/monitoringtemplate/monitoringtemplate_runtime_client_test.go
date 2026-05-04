/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoringtemplate

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testMonitoringTemplateID            = "ocid1.stackmonitoringtemplate.oc1..template"
	testMonitoringTemplateCompartmentID = "ocid1.compartment.oc1..template"
)

func TestMonitoringTemplateRuntimeHooksConfigured(t *testing.T) {
	hooks := newMonitoringTemplateDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	applyMonitoringTemplateRuntimeHooks(nil, &hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Read.Get", ok: hooks.Read.Get != nil},
		{name: "Read.List", ok: hooks.Read.List != nil},
		{name: "StatusHooks.ProjectStatus", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "DeleteHooks.ApplyOutcome", ok: hooks.DeleteHooks.ApplyOutcome != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}

	body, err := hooks.BuildCreateBody(context.Background(), testMonitoringTemplate(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(stackmonitoringsdk.CreateMonitoringTemplateDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateMonitoringTemplateDetails", body)
	}
	if details.IsAlarmsEnabled == nil || *details.IsAlarmsEnabled {
		t.Fatalf("BuildCreateBody() IsAlarmsEnabled = %#v, want explicit false pointer", details.IsAlarmsEnabled)
	}
	if details.IsSplitNotificationEnabled == nil || *details.IsSplitNotificationEnabled {
		t.Fatalf("BuildCreateBody() IsSplitNotificationEnabled = %#v, want explicit false pointer", details.IsSplitNotificationEnabled)
	}
}

func TestMonitoringTemplateCreateRecordsIdentityStatusAndRequestID(t *testing.T) {
	resource := testMonitoringTemplate()
	template := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	fake := &fakeMonitoringTemplateOCIClient{
		createMonitoringTemplate: func(_ context.Context, request stackmonitoringsdk.CreateMonitoringTemplateRequest) (stackmonitoringsdk.CreateMonitoringTemplateResponse, error) {
			assertMonitoringTemplateCreateRequest(t, request)
			return stackmonitoringsdk.CreateMonitoringTemplateResponse{
				MonitoringTemplate: template,
				OpcRequestId:       common.String("opc-create"),
			}, nil
		},
		getMonitoringTemplate: func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.GetMonitoringTemplateResponse{MonitoringTemplate: template}, nil
		},
	}

	response, err := newTestMonitoringTemplateClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoringTemplateRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMonitoringTemplateCallCount(t, "CreateMonitoringTemplate()", fake.createCalls, 1)
	assertMonitoringTemplateRecordedID(t, resource, testMonitoringTemplateID)
	assertMonitoringTemplateOpcRequestID(t, resource, "opc-create")
	if got := resource.Status.Status; got != string(stackmonitoringsdk.MonitoringTemplateLifeCycleDetailsApplied) {
		t.Fatalf("status.sdkStatus = %q, want APPLIED", got)
	}
}

func TestMonitoringTemplateCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := testMonitoringTemplate()
	template := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	var pages []string
	fake := &fakeMonitoringTemplateOCIClient{
		listMonitoringTemplates: func(_ context.Context, request stackmonitoringsdk.ListMonitoringTemplatesRequest) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error) {
			pages = append(pages, stringValue(request.Page))
			if request.Page == nil {
				return stackmonitoringsdk.ListMonitoringTemplatesResponse{
					MonitoringTemplateCollection: stackmonitoringsdk.MonitoringTemplateCollection{
						Items: []stackmonitoringsdk.MonitoringTemplateSummary{
							sdkMonitoringTemplateSummary(resource, "ocid1.stackmonitoringtemplate.oc1..deleted", stackmonitoringsdk.MonitoringTemplateLifeCycleStatesDeleted),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return stackmonitoringsdk.ListMonitoringTemplatesResponse{
				MonitoringTemplateCollection: stackmonitoringsdk.MonitoringTemplateCollection{
					Items: []stackmonitoringsdk.MonitoringTemplateSummary{
						sdkMonitoringTemplateSummary(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive),
					},
				},
			}, nil
		},
		getMonitoringTemplate: func(_ context.Context, request stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			if got := stringValue(request.MonitoringTemplateId); got != testMonitoringTemplateID {
				t.Fatalf("GetMonitoringTemplate() MonitoringTemplateId = %q, want %q", got, testMonitoringTemplateID)
			}
			return stackmonitoringsdk.GetMonitoringTemplateResponse{MonitoringTemplate: template}, nil
		},
	}

	response, err := newTestMonitoringTemplateClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoringTemplateRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMonitoringTemplateCallCount(t, "CreateMonitoringTemplate()", fake.createCalls, 0)
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListMonitoringTemplates() pages = %q, want \",page-2\"", got)
	}
	assertMonitoringTemplateRecordedID(t, resource, testMonitoringTemplateID)
}

func TestMonitoringTemplateCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := testMonitoringTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoringTemplateID)
	template := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	fake := &fakeMonitoringTemplateOCIClient{
		getMonitoringTemplate: func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.GetMonitoringTemplateResponse{MonitoringTemplate: template}, nil
		},
		updateMonitoringTemplate: func(context.Context, stackmonitoringsdk.UpdateMonitoringTemplateRequest) (stackmonitoringsdk.UpdateMonitoringTemplateResponse, error) {
			t.Fatal("UpdateMonitoringTemplate() called during no-op reconcile")
			return stackmonitoringsdk.UpdateMonitoringTemplateResponse{}, nil
		},
	}

	response, err := newTestMonitoringTemplateClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoringTemplateRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
}

func TestMonitoringTemplateMutableUpdatePreservesFalseAndClearsTags(t *testing.T) {
	resource := testMonitoringTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoringTemplateID)
	resource.Spec.FreeformTags = map[string]string{}
	current := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	current.IsAlarmsEnabled = common.Bool(true)
	current.FreeformTags = map[string]string{"owner": "old"}
	updated := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	getResponses := []stackmonitoringsdk.GetMonitoringTemplateResponse{
		{MonitoringTemplate: current},
		{MonitoringTemplate: updated},
	}
	fake := &fakeMonitoringTemplateOCIClient{
		getMonitoringTemplate: getMonitoringTemplateResponses(t, &getResponses),
		updateMonitoringTemplate: func(_ context.Context, request stackmonitoringsdk.UpdateMonitoringTemplateRequest) (stackmonitoringsdk.UpdateMonitoringTemplateResponse, error) {
			assertMonitoringTemplateUpdateRequest(t, request)
			return stackmonitoringsdk.UpdateMonitoringTemplateResponse{
				MonitoringTemplate: updated,
				OpcRequestId:       common.String("opc-update"),
			}, nil
		},
	}

	response, err := newTestMonitoringTemplateClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoringTemplateRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertMonitoringTemplateCallCount(t, "UpdateMonitoringTemplate()", fake.updateCalls, 1)
	assertMonitoringTemplateOpcRequestID(t, resource, "opc-update")
}

func TestMonitoringTemplateImmutableDriftRejectedBeforeUpdate(t *testing.T) {
	resource := testMonitoringTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoringTemplateID)
	current := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	current.CompartmentId = common.String("ocid1.compartment.oc1..different")
	fake := &fakeMonitoringTemplateOCIClient{
		getMonitoringTemplate: func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.GetMonitoringTemplateResponse{MonitoringTemplate: current}, nil
		},
		updateMonitoringTemplate: func(context.Context, stackmonitoringsdk.UpdateMonitoringTemplateRequest) (stackmonitoringsdk.UpdateMonitoringTemplateResponse, error) {
			t.Fatal("UpdateMonitoringTemplate() called despite compartmentId drift")
			return stackmonitoringsdk.UpdateMonitoringTemplateResponse{}, nil
		},
	}

	_, err := newTestMonitoringTemplateClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoringTemplateRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift detail", err)
	}
}

func TestMonitoringTemplateDeleteRetainsFinalizerWhileResourceStillReadable(t *testing.T) {
	resource := testMonitoringTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoringTemplateID)
	active := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	getResponses := []stackmonitoringsdk.GetMonitoringTemplateResponse{
		{MonitoringTemplate: active},
		{MonitoringTemplate: active},
		{MonitoringTemplate: active},
	}
	fake := &fakeMonitoringTemplateOCIClient{
		getMonitoringTemplate: getMonitoringTemplateResponses(t, &getResponses),
		deleteMonitoringTemplate: func(_ context.Context, request stackmonitoringsdk.DeleteMonitoringTemplateRequest) (stackmonitoringsdk.DeleteMonitoringTemplateResponse, error) {
			if got := stringValue(request.MonitoringTemplateId); got != testMonitoringTemplateID {
				t.Fatalf("DeleteMonitoringTemplate() MonitoringTemplateId = %q, want %q", got, testMonitoringTemplateID)
			}
			return stackmonitoringsdk.DeleteMonitoringTemplateResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestMonitoringTemplateClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI resource is still readable")
	}
	assertMonitoringTemplateCallCount(t, "DeleteMonitoringTemplate()", fake.deleteCalls, 1)
	assertMonitoringTemplateOpcRequestID(t, resource, "opc-delete")
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want pending delete lifecycle tracker")
	}
}

func TestMonitoringTemplateDeleteAlreadyPendingConfirmsWithoutSecondDelete(t *testing.T) {
	resource := testMonitoringTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoringTemplateID)
	active := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	fake := &fakeMonitoringTemplateOCIClient{
		getMonitoringTemplate: func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.GetMonitoringTemplateResponse{MonitoringTemplate: active}, nil
		},
		deleteMonitoringTemplate: func(context.Context, stackmonitoringsdk.DeleteMonitoringTemplateRequest) (stackmonitoringsdk.DeleteMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.DeleteMonitoringTemplateResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}
	client := newTestMonitoringTemplateClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("first Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("first Delete() deleted = true, want false while OCI readback remains ACTIVE")
	}
	assertMonitoringTemplateCallCount(t, "DeleteMonitoringTemplate()", fake.deleteCalls, 1)
	if !monitoringTemplateDeleteAlreadyPending(resource) {
		t.Fatalf("status.status.async.current = %#v, want pending delete tracker", resource.Status.OsokStatus.Async.Current)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("second Delete() deleted = true, want false while OCI readback remains ACTIVE")
	}
	assertMonitoringTemplateCallCount(t, "DeleteMonitoringTemplate()", fake.deleteCalls, 1)
	if !monitoringTemplateDeleteAlreadyPending(resource) {
		t.Fatalf("status.status.async.current = %#v, want pending delete tracker after retry", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMonitoringTemplateDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := testMonitoringTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoringTemplateID)
	active := sdkMonitoringTemplate(resource, testMonitoringTemplateID, stackmonitoringsdk.MonitoringTemplateLifeCycleStatesActive)
	notFound := errortest.NewServiceError(404, errorutil.NotFound, "not found")
	getCalls := 0
	fake := &fakeMonitoringTemplateOCIClient{
		getMonitoringTemplate: func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			getCalls++
			if getCalls <= 2 {
				return stackmonitoringsdk.GetMonitoringTemplateResponse{MonitoringTemplate: active}, nil
			}
			return stackmonitoringsdk.GetMonitoringTemplateResponse{}, notFound
		},
		deleteMonitoringTemplate: func(context.Context, stackmonitoringsdk.DeleteMonitoringTemplateRequest) (stackmonitoringsdk.DeleteMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.DeleteMonitoringTemplateResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestMonitoringTemplateClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
}

func TestMonitoringTemplateDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := testMonitoringTemplate()
	resource.Status.OsokStatus.Ocid = shared.OCID(testMonitoringTemplateID)
	authNotFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authNotFound.OpcRequestID = "opc-delete-auth"
	fake := &fakeMonitoringTemplateOCIClient{
		getMonitoringTemplate: func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.GetMonitoringTemplateResponse{}, authNotFound
		},
		deleteMonitoringTemplate: func(context.Context, stackmonitoringsdk.DeleteMonitoringTemplateRequest) (stackmonitoringsdk.DeleteMonitoringTemplateResponse, error) {
			t.Fatal("DeleteMonitoringTemplate() called after auth-shaped pre-delete read")
			return stackmonitoringsdk.DeleteMonitoringTemplateResponse{}, nil
		},
	}

	deleted, err := newTestMonitoringTemplateClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	assertMonitoringTemplateOpcRequestID(t, resource, "opc-delete-auth")
	assertMonitoringTemplateCallCount(t, "DeleteMonitoringTemplate()", fake.deleteCalls, 0)
}

func TestMonitoringTemplateCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := testMonitoringTemplate()
	createErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	createErr.OpcRequestID = "opc-create-error"
	fake := &fakeMonitoringTemplateOCIClient{
		createMonitoringTemplate: func(context.Context, stackmonitoringsdk.CreateMonitoringTemplateRequest) (stackmonitoringsdk.CreateMonitoringTemplateResponse, error) {
			return stackmonitoringsdk.CreateMonitoringTemplateResponse{}, createErr
		},
	}

	_, err := newTestMonitoringTemplateClient(fake).CreateOrUpdate(context.Background(), resource, testMonitoringTemplateRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	assertMonitoringTemplateOpcRequestID(t, resource, "opc-create-error")
}

func newTestMonitoringTemplateClient(fake *fakeMonitoringTemplateOCIClient) MonitoringTemplateServiceClient {
	hooks := newMonitoringTemplateDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	hooks.Create.Call = fake.CreateMonitoringTemplate
	hooks.Get.Call = fake.GetMonitoringTemplate
	hooks.List.Call = fake.ListMonitoringTemplates
	hooks.Update.Call = fake.UpdateMonitoringTemplate
	hooks.Delete.Call = fake.DeleteMonitoringTemplate
	applyMonitoringTemplateRuntimeHooks(&MonitoringTemplateServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}, &hooks)
	manager := &MonitoringTemplateServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	delegate := defaultMonitoringTemplateServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*stackmonitoringv1beta1.MonitoringTemplate](
			buildMonitoringTemplateGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapMonitoringTemplateGeneratedClient(hooks, delegate)
}

type fakeMonitoringTemplateOCIClient struct {
	createMonitoringTemplate func(context.Context, stackmonitoringsdk.CreateMonitoringTemplateRequest) (stackmonitoringsdk.CreateMonitoringTemplateResponse, error)
	getMonitoringTemplate    func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error)
	listMonitoringTemplates  func(context.Context, stackmonitoringsdk.ListMonitoringTemplatesRequest) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error)
	updateMonitoringTemplate func(context.Context, stackmonitoringsdk.UpdateMonitoringTemplateRequest) (stackmonitoringsdk.UpdateMonitoringTemplateResponse, error)
	deleteMonitoringTemplate func(context.Context, stackmonitoringsdk.DeleteMonitoringTemplateRequest) (stackmonitoringsdk.DeleteMonitoringTemplateResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeMonitoringTemplateOCIClient) CreateMonitoringTemplate(
	ctx context.Context,
	request stackmonitoringsdk.CreateMonitoringTemplateRequest,
) (stackmonitoringsdk.CreateMonitoringTemplateResponse, error) {
	f.createCalls++
	if f.createMonitoringTemplate == nil {
		return stackmonitoringsdk.CreateMonitoringTemplateResponse{}, nil
	}
	return f.createMonitoringTemplate(ctx, request)
}

func (f *fakeMonitoringTemplateOCIClient) GetMonitoringTemplate(
	ctx context.Context,
	request stackmonitoringsdk.GetMonitoringTemplateRequest,
) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
	f.getCalls++
	if f.getMonitoringTemplate == nil {
		return stackmonitoringsdk.GetMonitoringTemplateResponse{}, nil
	}
	return f.getMonitoringTemplate(ctx, request)
}

func (f *fakeMonitoringTemplateOCIClient) ListMonitoringTemplates(
	ctx context.Context,
	request stackmonitoringsdk.ListMonitoringTemplatesRequest,
) (stackmonitoringsdk.ListMonitoringTemplatesResponse, error) {
	f.listCalls++
	if f.listMonitoringTemplates == nil {
		return stackmonitoringsdk.ListMonitoringTemplatesResponse{}, nil
	}
	return f.listMonitoringTemplates(ctx, request)
}

func (f *fakeMonitoringTemplateOCIClient) UpdateMonitoringTemplate(
	ctx context.Context,
	request stackmonitoringsdk.UpdateMonitoringTemplateRequest,
) (stackmonitoringsdk.UpdateMonitoringTemplateResponse, error) {
	f.updateCalls++
	if f.updateMonitoringTemplate == nil {
		return stackmonitoringsdk.UpdateMonitoringTemplateResponse{}, nil
	}
	return f.updateMonitoringTemplate(ctx, request)
}

func (f *fakeMonitoringTemplateOCIClient) DeleteMonitoringTemplate(
	ctx context.Context,
	request stackmonitoringsdk.DeleteMonitoringTemplateRequest,
) (stackmonitoringsdk.DeleteMonitoringTemplateResponse, error) {
	f.deleteCalls++
	if f.deleteMonitoringTemplate == nil {
		return stackmonitoringsdk.DeleteMonitoringTemplateResponse{}, nil
	}
	return f.deleteMonitoringTemplate(ctx, request)
}

func getMonitoringTemplateResponses(
	t *testing.T,
	responses *[]stackmonitoringsdk.GetMonitoringTemplateResponse,
) func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
	t.Helper()

	return func(context.Context, stackmonitoringsdk.GetMonitoringTemplateRequest) (stackmonitoringsdk.GetMonitoringTemplateResponse, error) {
		if len(*responses) == 0 {
			t.Fatal("GetMonitoringTemplate() called more times than expected")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func testMonitoringTemplate() *stackmonitoringv1beta1.MonitoringTemplate {
	return &stackmonitoringv1beta1.MonitoringTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring-template",
			Namespace: "default",
			UID:       types.UID("monitoring-template-uid"),
		},
		Spec: stackmonitoringv1beta1.MonitoringTemplateSpec{
			DisplayName:                "template",
			CompartmentId:              testMonitoringTemplateCompartmentID,
			Destinations:               []string{"ocid1.onstopic.oc1..topic"},
			Members:                    []stackmonitoringv1beta1.MonitoringTemplateMember{{Id: "ocid1.stackmonitoringresource.oc1..resource", Type: string(stackmonitoringsdk.MemberReferenceTypeResourceInstance)}},
			Description:                "managed by osok",
			IsAlarmsEnabled:            false,
			IsSplitNotificationEnabled: false,
			RepeatNotificationDuration: "PT4H",
			MessageFormat:              string(stackmonitoringsdk.MessageFormatRaw),
			FreeformTags:               map[string]string{"owner": "osok"},
		},
	}
}

func testMonitoringTemplateRequest(resource *stackmonitoringv1beta1.MonitoringTemplate) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func sdkMonitoringTemplate(
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	id string,
	lifecycle stackmonitoringsdk.MonitoringTemplateLifeCycleStatesEnum,
) stackmonitoringsdk.MonitoringTemplate {
	members, _ := monitoringTemplateMembers(resource.Spec.Members)
	return stackmonitoringsdk.MonitoringTemplate{
		Id:                          common.String(id),
		DisplayName:                 common.String(resource.Spec.DisplayName),
		TenantId:                    common.String("ocid1.tenancy.oc1..template"),
		CompartmentId:               common.String(resource.Spec.CompartmentId),
		Status:                      stackmonitoringsdk.MonitoringTemplateLifeCycleDetailsApplied,
		LifecycleState:              lifecycle,
		Destinations:                cloneMonitoringTemplateStringSlice(resource.Spec.Destinations),
		Members:                     members,
		TotalAlarmConditions:        common.Float32(2),
		TotalAppliedAlarmConditions: common.Float32(2),
		Description:                 common.String(resource.Spec.Description),
		IsAlarmsEnabled:             common.Bool(resource.Spec.IsAlarmsEnabled),
		IsSplitNotificationEnabled:  common.Bool(resource.Spec.IsSplitNotificationEnabled),
		RepeatNotificationDuration:  common.String(resource.Spec.RepeatNotificationDuration),
		MessageFormat:               stackmonitoringsdk.MessageFormatEnum(resource.Spec.MessageFormat),
		FreeformTags:                cloneMonitoringTemplateStringMap(resource.Spec.FreeformTags),
	}
}

func sdkMonitoringTemplateSummary(
	resource *stackmonitoringv1beta1.MonitoringTemplate,
	id string,
	lifecycle stackmonitoringsdk.MonitoringTemplateLifeCycleStatesEnum,
) stackmonitoringsdk.MonitoringTemplateSummary {
	members, _ := monitoringTemplateMembers(resource.Spec.Members)
	return stackmonitoringsdk.MonitoringTemplateSummary{
		Id:                          common.String(id),
		DisplayName:                 common.String(resource.Spec.DisplayName),
		TenantId:                    common.String("ocid1.tenancy.oc1..template"),
		CompartmentId:               common.String(resource.Spec.CompartmentId),
		Status:                      stackmonitoringsdk.MonitoringTemplateLifeCycleDetailsApplied,
		LifecycleState:              lifecycle,
		Destinations:                cloneMonitoringTemplateStringSlice(resource.Spec.Destinations),
		Members:                     members,
		TotalAlarmConditions:        common.Float32(2),
		TotalAppliedAlarmConditions: common.Float32(2),
		Description:                 common.String(resource.Spec.Description),
		FreeformTags:                cloneMonitoringTemplateStringMap(resource.Spec.FreeformTags),
	}
}

func assertMonitoringTemplateCreateRequest(t *testing.T, request stackmonitoringsdk.CreateMonitoringTemplateRequest) {
	t.Helper()

	details := request.CreateMonitoringTemplateDetails
	if got := stringValue(details.CompartmentId); got != testMonitoringTemplateCompartmentID {
		t.Fatalf("CreateMonitoringTemplate() compartmentId = %q, want %q", got, testMonitoringTemplateCompartmentID)
	}
	if details.IsAlarmsEnabled == nil || *details.IsAlarmsEnabled {
		t.Fatalf("CreateMonitoringTemplate() IsAlarmsEnabled = %#v, want explicit false pointer", details.IsAlarmsEnabled)
	}
	if details.IsSplitNotificationEnabled == nil || *details.IsSplitNotificationEnabled {
		t.Fatalf("CreateMonitoringTemplate() IsSplitNotificationEnabled = %#v, want explicit false pointer", details.IsSplitNotificationEnabled)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateMonitoringTemplate() OpcRetryToken is empty, want deterministic retry token")
	}
}

func assertMonitoringTemplateUpdateRequest(t *testing.T, request stackmonitoringsdk.UpdateMonitoringTemplateRequest) {
	t.Helper()

	if got := stringValue(request.MonitoringTemplateId); got != testMonitoringTemplateID {
		t.Fatalf("UpdateMonitoringTemplate() MonitoringTemplateId = %q, want %q", got, testMonitoringTemplateID)
	}
	details := request.UpdateMonitoringTemplateDetails
	if details.IsAlarmsEnabled == nil || *details.IsAlarmsEnabled {
		t.Fatalf("UpdateMonitoringTemplate() IsAlarmsEnabled = %#v, want explicit false pointer", details.IsAlarmsEnabled)
	}
	if details.FreeformTags == nil {
		t.Fatal("UpdateMonitoringTemplate() FreeformTags = nil, want explicit empty map clear")
	}
	if len(details.FreeformTags) != 0 {
		t.Fatalf("UpdateMonitoringTemplate() FreeformTags = %#v, want empty map clear", details.FreeformTags)
	}
}

func assertMonitoringTemplateRecordedID(t *testing.T, resource *stackmonitoringv1beta1.MonitoringTemplate, want string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertMonitoringTemplateOpcRequestID(t *testing.T, resource *stackmonitoringv1beta1.MonitoringTemplate, want string) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertMonitoringTemplateCallCount(t *testing.T, operation string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", operation, got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
