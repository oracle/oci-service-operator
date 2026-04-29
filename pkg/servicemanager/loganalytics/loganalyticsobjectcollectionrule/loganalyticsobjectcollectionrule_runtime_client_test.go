package loganalyticsobjectcollectionrule

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testObjectCollectionRuleID            = "ocid1.loganalyticsobjectcollectionrule.oc1..rule"
	testObjectCollectionRuleOtherID       = "ocid1.loganalyticsobjectcollectionrule.oc1..other"
	testObjectCollectionRuleCompartmentID = "ocid1.compartment.oc1..loganalytics"
	testObjectCollectionRuleLogGroupID    = "ocid1.loganalyticsloggroup.oc1..group"
	testObjectCollectionRuleNamespace     = "tenantns"
	testObjectCollectionRuleBucket        = "logs-bucket"
	testObjectCollectionRuleName          = "object-collection-rule"
	testObjectCollectionRuleSourceName    = "linux_syslog"
)

type fakeLogAnalyticsObjectCollectionRuleOCIClient struct {
	createFn func(context.Context, loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse, error)
	getFn    func(context.Context, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error)
	listFn   func(context.Context, loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error)
	updateFn func(context.Context, loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse, error)
	deleteFn func(context.Context, loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error)
}

func (f *fakeLogAnalyticsObjectCollectionRuleOCIClient) CreateLogAnalyticsObjectCollectionRule(
	ctx context.Context,
	request loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest,
) (loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse{}, nil
}

func (f *fakeLogAnalyticsObjectCollectionRuleOCIClient) GetLogAnalyticsObjectCollectionRule(
	ctx context.Context,
	request loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest,
) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{}, nil
}

func (f *fakeLogAnalyticsObjectCollectionRuleOCIClient) ListLogAnalyticsObjectCollectionRules(
	ctx context.Context,
	request loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest,
) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{}, nil
}

func (f *fakeLogAnalyticsObjectCollectionRuleOCIClient) UpdateLogAnalyticsObjectCollectionRule(
	ctx context.Context,
	request loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest,
) (loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse{}, nil
}

func (f *fakeLogAnalyticsObjectCollectionRuleOCIClient) DeleteLogAnalyticsObjectCollectionRule(
	ctx context.Context,
	request loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest,
) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse{}, nil
}

func testLogAnalyticsObjectCollectionRuleClient(fake *fakeLogAnalyticsObjectCollectionRuleOCIClient) LogAnalyticsObjectCollectionRuleServiceClient {
	return newLogAnalyticsObjectCollectionRuleServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func makeLogAnalyticsObjectCollectionRuleResource() *loganalyticsv1beta1.LogAnalyticsObjectCollectionRule {
	return &loganalyticsv1beta1.LogAnalyticsObjectCollectionRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "object-rule",
			Namespace: "default",
			UID:       types.UID("object-rule-uid"),
		},
		Spec: loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec{
			Name:                      testObjectCollectionRuleName,
			CompartmentId:             testObjectCollectionRuleCompartmentID,
			OsNamespace:               testObjectCollectionRuleNamespace,
			OsBucketName:              testObjectCollectionRuleBucket,
			LogGroupId:                testObjectCollectionRuleLogGroupID,
			Description:               "collect bucket logs",
			CollectionType:            string(loganalyticssdk.ObjectCollectionRuleCollectionTypesHistoric),
			PollSince:                 "BEGINNING",
			PollTill:                  "CURRENT_TIME",
			LogSourceName:             testObjectCollectionRuleSourceName,
			IsEnabled:                 true,
			ObjectNameFilters:         []string{"*.log"},
			LogType:                   string(loganalyticssdk.LogTypesLog),
			IsForceHistoricCollection: false,
			DefinedTags:               map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			FreeformTags:              map[string]string{"env": "dev"},
		},
	}
}

func makeSDKLogAnalyticsObjectCollectionRule(
	id string,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	state loganalyticssdk.ObjectCollectionRuleLifecycleStatesEnum,
) loganalyticssdk.LogAnalyticsObjectCollectionRule {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)}
	return loganalyticssdk.LogAnalyticsObjectCollectionRule{
		Id:                        common.String(id),
		Name:                      common.String(spec.Name),
		CompartmentId:             common.String(spec.CompartmentId),
		OsNamespace:               common.String(spec.OsNamespace),
		OsBucketName:              common.String(spec.OsBucketName),
		CollectionType:            loganalyticssdk.ObjectCollectionRuleCollectionTypesEnum(spec.CollectionType),
		PollSince:                 common.String(spec.PollSince),
		LogGroupId:                common.String(spec.LogGroupId),
		LogSourceName:             common.String(spec.LogSourceName),
		LifecycleState:            state,
		TimeCreated:               &created,
		TimeUpdated:               &updated,
		IsEnabled:                 common.Bool(spec.IsEnabled),
		Description:               common.String(spec.Description),
		PollTill:                  common.String(spec.PollTill),
		EntityId:                  optionalTestString(spec.EntityId),
		CharEncoding:              optionalTestString(spec.CharEncoding),
		Timezone:                  optionalTestString(spec.Timezone),
		LogSet:                    optionalTestString(spec.LogSet),
		LogSetKey:                 loganalyticssdk.LogSetKeyTypesEnum(spec.LogSetKey),
		LogSetExtRegex:            optionalTestString(spec.LogSetExtRegex),
		Overrides:                 propertyOverridesFromSpec(spec.Overrides),
		ObjectNameFilters:         copyStrings(spec.ObjectNameFilters),
		LogType:                   loganalyticssdk.LogTypesEnum(spec.LogType),
		IsForceHistoricCollection: common.Bool(spec.IsForceHistoricCollection),
		StreamId:                  optionalTestString(spec.StreamId),
		StreamCursorType:          loganalyticssdk.StreamCursorTypesEnum(spec.StreamCursorType),
		StreamCursorTime:          optionalTestSDKTime(spec.StreamCursorTime),
		DefinedTags:               definedTagsFromSpec(spec.DefinedTags),
		FreeformTags:              spec.FreeformTags,
	}
}

func makeSDKLogAnalyticsObjectCollectionRuleSummary(
	id string,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
	state loganalyticssdk.ObjectCollectionRuleLifecycleStatesEnum,
) loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)}
	return loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary{
		Id:                common.String(id),
		Name:              common.String(spec.Name),
		CompartmentId:     common.String(spec.CompartmentId),
		OsNamespace:       common.String(spec.OsNamespace),
		OsBucketName:      common.String(spec.OsBucketName),
		CollectionType:    loganalyticssdk.ObjectCollectionRuleCollectionTypesEnum(spec.CollectionType),
		LifecycleState:    state,
		TimeCreated:       &created,
		TimeUpdated:       &updated,
		IsEnabled:         common.Bool(spec.IsEnabled),
		Description:       common.String(spec.Description),
		ObjectNameFilters: copyStrings(spec.ObjectNameFilters),
		LogType:           loganalyticssdk.LogTypesEnum(spec.LogType),
		DefinedTags:       definedTagsFromSpec(spec.DefinedTags),
		FreeformTags:      spec.FreeformTags,
	}
}

func TestLogAnalyticsObjectCollectionRuleCreateUsesNamespaceBodyAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Spec.IsEnabled = false

	var listRequest loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest
	var createRequest loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest
	var getRequest loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		listFn: func(_ context.Context, request loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error) {
			listRequest = request
			return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{}, nil
		},
		createFn: func(_ context.Context, request loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse, error) {
			createRequest = request
			return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			getRequest = request
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}

	requireStringPtr(t, "List namespaceName", listRequest.NamespaceName, testObjectCollectionRuleNamespace)
	requireStringPtr(t, "List compartmentId", listRequest.CompartmentId, testObjectCollectionRuleCompartmentID)
	requireStringPtr(t, "List name", listRequest.Name, testObjectCollectionRuleName)
	requireStringPtr(t, "Create namespaceName", createRequest.NamespaceName, testObjectCollectionRuleNamespace)
	requireStringPtr(t, "Get namespaceName", getRequest.NamespaceName, testObjectCollectionRuleNamespace)
	requireStringPtr(t, "Get rule ID", getRequest.LogAnalyticsObjectCollectionRuleId, testObjectCollectionRuleID)

	details := createRequest.CreateLogAnalyticsObjectCollectionRuleDetails
	requireStringPtr(t, "Create body name", details.Name, testObjectCollectionRuleName)
	requireStringPtr(t, "Create body osNamespace", details.OsNamespace, testObjectCollectionRuleNamespace)
	requireStringPtr(t, "Create body osBucketName", details.OsBucketName, testObjectCollectionRuleBucket)
	if details.IsEnabled == nil || *details.IsEnabled {
		t.Fatalf("Create body IsEnabled = %#v, want explicit false", details.IsEnabled)
	}
	if got := string(details.CollectionType); got != resource.Spec.CollectionType {
		t.Fatalf("Create body CollectionType = %q, want %q", got, resource.Spec.CollectionType)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testObjectCollectionRuleID {
		t.Fatalf("status.ocid = %q, want %q", got, testObjectCollectionRuleID)
	}
	if got := resource.Status.Id; got != testObjectCollectionRuleID {
		t.Fatalf("status.id = %q, want %q", got, testObjectCollectionRuleID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", got)
	}
}

func TestLogAnalyticsObjectCollectionRuleBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	otherSpec := resource.Spec
	otherSpec.Name = "other-rule"
	var listPages []string
	createCalls := 0
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		listFn: func(_ context.Context, request loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error) {
			listPages = append(listPages, stringPtrValue(request.Page))
			if request.Page == nil {
				return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{
					LogAnalyticsObjectCollectionRuleCollection: loganalyticssdk.LogAnalyticsObjectCollectionRuleCollection{
						Items: []loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary{
							makeSDKLogAnalyticsObjectCollectionRuleSummary(testObjectCollectionRuleOtherID, otherSpec, loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{
				LogAnalyticsObjectCollectionRuleCollection: loganalyticssdk.LogAnalyticsObjectCollectionRuleCollection{
					Items: []loganalyticssdk.LogAnalyticsObjectCollectionRuleSummary{
						makeSDKLogAnalyticsObjectCollectionRuleSummary(testObjectCollectionRuleID, resource.Spec, loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive),
					},
				},
			}, nil
		},
		createFn: func(context.Context, loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse, error) {
			createCalls++
			return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse{}, nil
		},
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			requireStringPtr(t, "Get rule ID", request.LogAnalyticsObjectCollectionRuleId, testObjectCollectionRuleID)
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0 after list bind", createCalls)
	}
	if got, want := strings.Join(listPages, ","), ",page-2"; got != want {
		t.Fatalf("List pages = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testObjectCollectionRuleID {
		t.Fatalf("status.ocid = %q, want %q", got, testObjectCollectionRuleID)
	}
}

func TestLogAnalyticsObjectCollectionRuleNoopReconcileSkipsUpdate(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testObjectCollectionRuleID)
	updateCalls := 0
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			requireStringPtr(t, "Get namespaceName", request.NamespaceName, testObjectCollectionRuleNamespace)
			requireStringPtr(t, "Get rule ID", request.LogAnalyticsObjectCollectionRuleId, testObjectCollectionRuleID)
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
		updateFn: func(context.Context, loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse, error) {
			updateCalls++
			return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-requeue observe", response)
	}
	if updateCalls != 0 {
		t.Fatalf("Update calls = %d, want 0 when observed mutable state matches spec", updateCalls)
	}
}

func TestLogAnalyticsObjectCollectionRuleMutableUpdateShapesBody(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testObjectCollectionRuleID)
	resource.Spec.Description = ""
	resource.Spec.LogGroupId = "ocid1.loganalyticsloggroup.oc1..updated"
	resource.Spec.IsEnabled = false
	resource.Spec.LogSetKey = string(loganalyticssdk.LogSetKeyTypesObjectPath)
	resource.Spec.LogSetExtRegex = "apps/([^/]+)/"
	resource.Spec.StreamCursorType = string(loganalyticssdk.StreamCursorTypesLatest)
	resource.Spec.StreamCursorTime = "2026-04-29T15:04:05Z"
	resource.Spec.ObjectNameFilters = []string{"app/*.log"}
	resource.Spec.Overrides = map[string][]loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleOverrides{
		"app": {{MatchType: "contains", MatchValue: "app/", PropertyName: "logSourceName", PropertyValue: "app_source"}},
	}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	var updateRequest loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest
	getCalls := 0
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		getFn: func(_ context.Context, request loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			getCalls++
			requireStringPtr(t, "Get rule ID", request.LogAnalyticsObjectCollectionRuleId, testObjectCollectionRuleID)
			currentSpec := makeLogAnalyticsObjectCollectionRuleResource().Spec
			if getCalls > 1 {
				currentSpec = resource.Spec
			}
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					currentSpec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, request loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse, error) {
			updateRequest = request
			return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}

	requireStringPtr(t, "Update namespaceName", updateRequest.NamespaceName, testObjectCollectionRuleNamespace)
	requireStringPtr(t, "Update rule ID", updateRequest.LogAnalyticsObjectCollectionRuleId, testObjectCollectionRuleID)
	details := updateRequest.UpdateLogAnalyticsObjectCollectionRuleDetails
	requireLogAnalyticsObjectCollectionRuleStringUpdateDetails(t, details, resource.Spec)
	requireLogAnalyticsObjectCollectionRuleEnumUpdateDetails(t, details, resource.Spec)
	requireLogAnalyticsObjectCollectionRuleCollectionUpdateDetails(t, details)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", got)
	}
}

func requireLogAnalyticsObjectCollectionRuleStringUpdateDetails(
	t *testing.T,
	details loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
) {
	t.Helper()

	requireStringPtr(t, "Update logGroupId", details.LogGroupId, spec.LogGroupId)
	if details.Description == nil || *details.Description != "" {
		t.Fatalf("Update description = %#v, want explicit empty string clear", details.Description)
	}
	if details.IsEnabled == nil || *details.IsEnabled {
		t.Fatalf("Update IsEnabled = %#v, want explicit false", details.IsEnabled)
	}
}

func requireLogAnalyticsObjectCollectionRuleEnumUpdateDetails(
	t *testing.T,
	details loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
	spec loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec,
) {
	t.Helper()

	if got := string(details.LogSetKey); got != spec.LogSetKey {
		t.Fatalf("Update LogSetKey = %q, want %q", got, spec.LogSetKey)
	}
	if got := string(details.StreamCursorType); got != spec.StreamCursorType {
		t.Fatalf("Update StreamCursorType = %q, want %q", got, spec.StreamCursorType)
	}
	if details.StreamCursorTime == nil || !details.StreamCursorTime.Equal(time.Date(2026, 4, 29, 15, 4, 5, 0, time.UTC)) {
		t.Fatalf("Update StreamCursorTime = %#v, want parsed RFC3339 time", details.StreamCursorTime)
	}
}

func requireLogAnalyticsObjectCollectionRuleCollectionUpdateDetails(
	t *testing.T,
	details loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleDetails,
) {
	t.Helper()

	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("Update FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if got := details.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("Update DefinedTags Operations.CostCenter = %#v, want 84", got)
	}
	if len(details.ObjectNameFilters) != 1 || details.ObjectNameFilters[0] != "app/*.log" {
		t.Fatalf("Update ObjectNameFilters = %#v, want updated filter", details.ObjectNameFilters)
	}
	if got := details.Overrides["app"][0].PropertyValue; got == nil || *got != "app_source" {
		t.Fatalf("Update Overrides = %#v, want app_source property override", details.Overrides)
	}
}

func TestLogAnalyticsObjectCollectionRuleCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testObjectCollectionRuleID)
	resource.Spec.Name = "renamed-rule"
	updateCalls := 0
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			currentSpec := makeLogAnalyticsObjectCollectionRuleResource().Spec
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					currentSpec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
		updateFn: func(context.Context, loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse, error) {
			updateCalls++
			return loganalyticssdk.UpdateLogAnalyticsObjectCollectionRuleResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("CreateOrUpdate() error = %v, want name create-only drift rejection", err)
	}
	if updateCalls != 0 {
		t.Fatalf("Update calls = %d, want 0 after pre-OCI drift rejection", updateCalls)
	}
}

func TestLogAnalyticsObjectCollectionRuleDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testObjectCollectionRuleID)
	getCalls := 0
	var deleteRequest loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			getCalls++
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error) {
			deleteRequest = request
			return loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback remains ACTIVE")
	}
	if getCalls != 2 {
		t.Fatalf("Get calls = %d, want already-pending read and post-delete confirmation read", getCalls)
	}
	requireStringPtr(t, "Delete namespaceName", deleteRequest.NamespaceName, testObjectCollectionRuleNamespace)
	requireStringPtr(t, "Delete rule ID", deleteRequest.LogAnalyticsObjectCollectionRuleId, testObjectCollectionRuleID)
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete ||
		resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending lifecycle delete", resource.Status.OsokStatus.Async.Current)
	}
	if got := latestConditionType(resource.Status.OsokStatus); got != shared.Terminating {
		t.Fatalf("latest condition = %q, want Terminating", got)
	}
}

func TestLogAnalyticsObjectCollectionRuleDeleteConfirmedAfterDeletedReadback(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testObjectCollectionRuleID)
	getCalls := 0
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			getCalls++
			state := loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive
			if getCalls > 1 {
				state = loganalyticssdk.ObjectCollectionRuleLifecycleStatesDeleted
			}
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(testObjectCollectionRuleID, resource.Spec, state),
			}, nil
		},
		deleteFn: func(context.Context, loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error) {
			return loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer removal after DELETED readback")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestLogAnalyticsObjectCollectionRuleDeleteSkipsRepeatDeleteWhileLifecyclePending(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testObjectCollectionRuleID)
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       string(loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive),
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
	getCalls := 0
	deleteCalls := 0
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			getCalls++
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error) {
			deleteCalls++
			return loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while pending lifecycle delete remains ACTIVE")
	}
	if getCalls != 1 {
		t.Fatalf("Get calls = %d, want one already-pending confirmation read", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete calls = %d, want 0 while status already records a pending lifecycle delete", deleteCalls)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete ||
		resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending lifecycle delete", resource.Status.OsokStatus.Async.Current)
	}
	if got := latestConditionType(resource.Status.OsokStatus); got != shared.Terminating {
		t.Fatalf("latest condition = %q, want Terminating", got)
	}
}

func TestLogAnalyticsObjectCollectionRuleDeleteKeepsAuthShapedNotFoundConservative(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testObjectCollectionRuleID)
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		getFn: func(context.Context, loganalyticssdk.GetLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse, error) {
			return loganalyticssdk.GetLogAnalyticsObjectCollectionRuleResponse{
				LogAnalyticsObjectCollectionRule: makeSDKLogAnalyticsObjectCollectionRule(
					testObjectCollectionRuleID,
					resource.Spec,
					loganalyticssdk.ObjectCollectionRuleLifecycleStatesActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse, error) {
			return loganalyticssdk.DeleteLogAnalyticsObjectCollectionRuleResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative ambiguous 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained on ambiguous 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want OCI error request ID", got)
	}
}

func TestLogAnalyticsObjectCollectionRuleCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := makeLogAnalyticsObjectCollectionRuleResource()
	client := testLogAnalyticsObjectCollectionRuleClient(&fakeLogAnalyticsObjectCollectionRuleOCIClient{
		listFn: func(context.Context, loganalyticssdk.ListLogAnalyticsObjectCollectionRulesRequest) (loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse, error) {
			return loganalyticssdk.ListLogAnalyticsObjectCollectionRulesResponse{}, nil
		},
		createFn: func(context.Context, loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleRequest) (loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse, error) {
			return loganalyticssdk.CreateLogAnalyticsObjectCollectionRuleResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want OCI error request ID", got)
	}
	if got := latestConditionType(resource.Status.OsokStatus); got != shared.Failed {
		t.Fatalf("latest condition = %q, want Failed", got)
	}
}

func optionalTestString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func optionalTestSDKTime(value string) *common.SDKTime {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &common.SDKTime{Time: parsed}
}

func requireStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", field, *got, want)
	}
}

func latestConditionType(status shared.OSOKStatus) shared.OSOKConditionType {
	if len(status.Conditions) == 0 {
		return ""
	}
	return status.Conditions[len(status.Conditions)-1].Type
}
