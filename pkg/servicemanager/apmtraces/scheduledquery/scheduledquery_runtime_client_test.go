package scheduledquery

import (
	"context"
	"strings"
	"testing"

	apmtracessdk "github.com/oracle/oci-go-sdk/v65/apmtraces"
	"github.com/oracle/oci-go-sdk/v65/common"
	apmtracesv1beta1 "github.com/oracle/oci-service-operator/api/apmtraces/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestApplyScheduledQueryRuntimeHooksUsesReviewedListLookup(t *testing.T) {
	t.Parallel()

	hooks := newScheduledQueryDefaultRuntimeHooks(apmtracessdk.ScheduledQueryClient{})
	applyScheduledQueryRuntimeHooks(&hooks)

	if hooks.Semantics == nil || hooks.Semantics.List == nil {
		t.Fatal("hooks.Semantics.List = nil, want reviewed list semantics")
	}
	if len(hooks.Semantics.List.MatchFields) != 1 || hooks.Semantics.List.MatchFields[0] != "scheduledQueryName" {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want [\"scheduledQueryName\"]", hooks.Semantics.List.MatchFields)
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want reviewed guard")
	}
	if len(hooks.List.Fields) != 2 {
		t.Fatalf("hooks.List.Fields length = %d, want 2", len(hooks.List.Fields))
	}
	if hooks.List.Fields[1].FieldName != "DisplayName" {
		t.Fatalf("hooks.List.Fields[1].FieldName = %q, want DisplayName", hooks.List.Fields[1].FieldName)
	}
	if got := hooks.List.Fields[1].LookupPaths; len(got) != 3 || got[0] != "status.scheduledQueryName" || got[1] != "spec.scheduledQueryName" || got[2] != "scheduledQueryName" {
		t.Fatalf("hooks.List.Fields[1].LookupPaths = %#v, want scheduledQueryName lookup paths", got)
	}
}

func TestGuardScheduledQueryExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource *apmtracesv1beta1.ScheduledQuery
		want     generatedruntime.ExistingBeforeCreateDecision
		wantErr  string
	}{
		{
			name:     "nil resource fails",
			resource: nil,
			want:     generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr:  "resource is nil",
		},
		{
			name: "missing domain fails",
			resource: &apmtracesv1beta1.ScheduledQuery{
				Spec: apmtracesv1beta1.ScheduledQuerySpec{
					ScheduledQueryName: "scheduled-query",
				},
			},
			want:    generatedruntime.ExistingBeforeCreateDecisionFail,
			wantErr: "spec.apmDomainId is required",
		},
		{
			name: "missing scheduled query name skips",
			resource: &apmtracesv1beta1.ScheduledQuery{
				Spec: apmtracesv1beta1.ScheduledQuerySpec{
					ApmDomainId: "ocid1.apmdomain.oc1..example",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionSkip,
		},
		{
			name: "domain and name allow lookup",
			resource: &apmtracesv1beta1.ScheduledQuery{
				Spec: apmtracesv1beta1.ScheduledQuerySpec{
					ApmDomainId:        "ocid1.apmdomain.oc1..example",
					ScheduledQueryName: "scheduled-query",
				},
			},
			want: generatedruntime.ExistingBeforeCreateDecisionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := guardScheduledQueryExistingBeforeCreate(context.Background(), tt.resource)
			if got != tt.want {
				t.Fatalf("guardScheduledQueryExistingBeforeCreate() = %q, want %q", got, tt.want)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("guardScheduledQueryExistingBeforeCreate() error = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("guardScheduledQueryExistingBeforeCreate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestScheduledQueryBuildCreateBodyOmitsEmptyProcessingConfiguration(t *testing.T) {
	t.Parallel()

	body, err := scheduledQueryBuildCreateBody(context.Background(), &apmtracesv1beta1.ScheduledQuery{
		Spec: apmtracesv1beta1.ScheduledQuerySpec{
			ScheduledQueryName:                    "scheduled-query",
			ScheduledQueryProcessingType:          "QUERY",
			ScheduledQueryText:                    "SHOW SPANS * BETWEEN now() - 2 HOURS AND now()",
			ScheduledQuerySchedule:                "SCHEDULE STARTING AFTER 2026-05-28T22:00:00Z EVERY 60 MINUTES",
			ScheduledQueryMaximumRuntimeInSeconds: 60,
			ScheduledQueryRetentionPeriodInMs:     60000,
			ScheduledQueryRetentionCriteria:       "UPDATE",
		},
	}, "default")
	if err != nil {
		t.Fatalf("scheduledQueryBuildCreateBody() error = %v", err)
	}
	details, ok := body.(apmtracessdk.CreateScheduledQueryDetails)
	if !ok {
		t.Fatalf("scheduledQueryBuildCreateBody() = %T, want CreateScheduledQueryDetails", body)
	}
	if details.ScheduledQueryProcessingConfiguration != nil {
		t.Fatalf("ScheduledQueryProcessingConfiguration = %#v, want nil for empty generated nested spec", details.ScheduledQueryProcessingConfiguration)
	}
	if details.ScheduledQueryRetentionCriteria != apmtracessdk.ScheduledQueryRetentionCriteriaUpdate {
		t.Fatalf("ScheduledQueryRetentionCriteria = %q, want UPDATE", details.ScheduledQueryRetentionCriteria)
	}
	if details.ScheduledQueryMaximumRuntimeInSeconds == nil || *details.ScheduledQueryMaximumRuntimeInSeconds != 60 {
		t.Fatalf("ScheduledQueryMaximumRuntimeInSeconds = %#v, want 60", details.ScheduledQueryMaximumRuntimeInSeconds)
	}
	if details.ScheduledQueryRetentionPeriodInMs != nil {
		t.Fatalf("ScheduledQueryRetentionPeriodInMs = %#v, want nil when retention criteria is UPDATE", details.ScheduledQueryRetentionPeriodInMs)
	}
}

func TestScheduledQueryBuildUpdateBodyIgnoresEmptyProcessingConfigurationWhenCurrentMatches(t *testing.T) {
	t.Parallel()

	resource := &apmtracesv1beta1.ScheduledQuery{
		Spec: apmtracesv1beta1.ScheduledQuerySpec{
			ScheduledQueryName:                    "scheduled-query",
			ScheduledQueryProcessingType:          "QUERY",
			ScheduledQueryText:                    "SHOW SPANS * BETWEEN now() - 2 HOURS AND now()",
			ScheduledQuerySchedule:                "SCHEDULE STARTING AFTER 2026-05-28T22:00:00Z EVERY 60 MINUTES",
			ScheduledQueryMaximumRuntimeInSeconds: 60,
			ScheduledQueryRetentionPeriodInMs:     60000,
			ScheduledQueryRetentionCriteria:       "UPDATE",
		},
	}
	current := apmtracessdk.GetScheduledQueryResponse{
		ScheduledQuery: apmtracessdk.ScheduledQuery{
			ScheduledQueryName:                    common.String(resource.Spec.ScheduledQueryName),
			ScheduledQueryProcessingType:          apmtracessdk.ScheduledQueryProcessingTypeQuery,
			ScheduledQueryText:                    common.String(resource.Spec.ScheduledQueryText),
			ScheduledQuerySchedule:                common.String(resource.Spec.ScheduledQuerySchedule),
			ScheduledQueryMaximumRuntimeInSeconds: common.Int64(resource.Spec.ScheduledQueryMaximumRuntimeInSeconds),
			ScheduledQueryRetentionPeriodInMs:     common.Int64(resource.Spec.ScheduledQueryRetentionPeriodInMs),
			ScheduledQueryProcessingSubType:       apmtracessdk.ScheduledQueryProcessingSubTypeNone,
			ScheduledQueryProcessingConfiguration: nil,
			ScheduledQueryRetentionCriteria:       apmtracessdk.ScheduledQueryRetentionCriteriaUpdate,
		},
	}

	body, ok, err := scheduledQueryBuildUpdateBody(context.Background(), resource, "default", current)
	if err != nil {
		t.Fatalf("scheduledQueryBuildUpdateBody() error = %v", err)
	}
	if ok {
		t.Fatalf("scheduledQueryBuildUpdateBody() ok = true, body = %#v; want no update", body)
	}
}

func TestScheduledQueryBuildUpdateBodyDetectsMutableDrift(t *testing.T) {
	t.Parallel()

	resource := &apmtracesv1beta1.ScheduledQuery{
		Spec: apmtracesv1beta1.ScheduledQuerySpec{
			ScheduledQueryName:                    "scheduled-query",
			ScheduledQueryProcessingType:          "QUERY",
			ScheduledQueryText:                    "SHOW SPANS * BETWEEN now() - 2 HOURS AND now()",
			ScheduledQuerySchedule:                "SCHEDULE STARTING AFTER 2026-05-28T22:00:00Z EVERY 60 MINUTES",
			ScheduledQueryDescription:             "updated description",
			ScheduledQueryMaximumRuntimeInSeconds: 60,
			ScheduledQueryRetentionPeriodInMs:     60000,
			ScheduledQueryRetentionCriteria:       "UPDATE",
		},
	}
	current := apmtracessdk.GetScheduledQueryResponse{
		ScheduledQuery: apmtracessdk.ScheduledQuery{
			ScheduledQueryName:                    common.String(resource.Spec.ScheduledQueryName),
			ScheduledQueryProcessingType:          apmtracessdk.ScheduledQueryProcessingTypeQuery,
			ScheduledQueryText:                    common.String(resource.Spec.ScheduledQueryText),
			ScheduledQuerySchedule:                common.String(resource.Spec.ScheduledQuerySchedule),
			ScheduledQueryDescription:             common.String("old description"),
			ScheduledQueryMaximumRuntimeInSeconds: common.Int64(resource.Spec.ScheduledQueryMaximumRuntimeInSeconds),
			ScheduledQueryRetentionPeriodInMs:     common.Int64(resource.Spec.ScheduledQueryRetentionPeriodInMs),
			ScheduledQueryRetentionCriteria:       apmtracessdk.ScheduledQueryRetentionCriteriaUpdate,
		},
	}

	body, ok, err := scheduledQueryBuildUpdateBody(context.Background(), resource, "default", current)
	if err != nil {
		t.Fatalf("scheduledQueryBuildUpdateBody() error = %v", err)
	}
	if !ok {
		t.Fatal("scheduledQueryBuildUpdateBody() ok = false, want mutable update")
	}
	details, ok := body.(apmtracessdk.UpdateScheduledQueryDetails)
	if !ok {
		t.Fatalf("scheduledQueryBuildUpdateBody() body = %T, want UpdateScheduledQueryDetails", body)
	}
	if details.ScheduledQueryProcessingConfiguration != nil {
		t.Fatalf("ScheduledQueryProcessingConfiguration = %#v, want nil for empty generated nested spec", details.ScheduledQueryProcessingConfiguration)
	}
	if details.ScheduledQueryDescription == nil || *details.ScheduledQueryDescription != resource.Spec.ScheduledQueryDescription {
		t.Fatalf("ScheduledQueryDescription = %#v, want updated description", details.ScheduledQueryDescription)
	}
	if details.ScheduledQueryText != nil || details.ScheduledQuerySchedule != nil || details.ScheduledQueryMaximumRuntimeInSeconds != nil {
		t.Fatalf("update body included unchanged fields: text=%#v schedule=%#v maximumRuntime=%#v", details.ScheduledQueryText, details.ScheduledQuerySchedule, details.ScheduledQueryMaximumRuntimeInSeconds)
	}
}

func TestNewScheduledQueryServiceClientWithOCIClientUsesNameFilteredLookupAndMirrorsDomain(t *testing.T) {
	t.Parallel()

	resource := &apmtracesv1beta1.ScheduledQuery{
		Spec: apmtracesv1beta1.ScheduledQuerySpec{
			ApmDomainId:                  "ocid1.apmdomain.oc1..example",
			ScheduledQueryName:           "scheduled-query",
			ScheduledQueryProcessingType: "QUERY",
			ScheduledQueryText:           "show spans",
			ScheduledQuerySchedule:       "0 */12 * * *",
		},
	}

	existingID := "ocid1.scheduledquery.oc1..existing"
	var listRequest apmtracessdk.ListScheduledQueriesRequest
	var getRequest apmtracessdk.GetScheduledQueryRequest
	createCalled := false

	client := newScheduledQueryServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubScheduledQueryOCIClient{
			create: func(context.Context, apmtracessdk.CreateScheduledQueryRequest) (apmtracessdk.CreateScheduledQueryResponse, error) {
				createCalled = true
				return apmtracessdk.CreateScheduledQueryResponse{}, nil
			},
			list: func(_ context.Context, req apmtracessdk.ListScheduledQueriesRequest) (apmtracessdk.ListScheduledQueriesResponse, error) {
				listRequest = req
				return apmtracessdk.ListScheduledQueriesResponse{
					ScheduledQueryCollection: apmtracessdk.ScheduledQueryCollection{
						Items: []apmtracessdk.ScheduledQuerySummary{
							{
								Id:                 common.String(existingID),
								ScheduledQueryName: common.String(resource.Spec.ScheduledQueryName),
								LifecycleState:     apmtracessdk.LifecycleStatesActive,
							},
						},
					},
				}, nil
			},
			get: func(_ context.Context, req apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error) {
				getRequest = req
				return apmtracessdk.GetScheduledQueryResponse{
					ScheduledQuery: apmtracessdk.ScheduledQuery{
						Id:                           common.String(existingID),
						ScheduledQueryName:           common.String(resource.Spec.ScheduledQueryName),
						ScheduledQueryProcessingType: apmtracessdk.ScheduledQueryProcessingTypeEnum(resource.Spec.ScheduledQueryProcessingType),
						ScheduledQueryText:           common.String(resource.Spec.ScheduledQueryText),
						ScheduledQuerySchedule:       common.String(resource.Spec.ScheduledQuerySchedule),
						LifecycleState:               apmtracessdk.LifecycleStatesActive,
					},
				}, nil
			},
		},
	)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateOrUpdate() invoked CreateScheduledQuery, want existing-before-create reuse")
	}
	if listRequest.ApmDomainId == nil || *listRequest.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("ListScheduledQueriesRequest.ApmDomainId = %#v, want %q", listRequest.ApmDomainId, resource.Spec.ApmDomainId)
	}
	if listRequest.DisplayName == nil || *listRequest.DisplayName != resource.Spec.ScheduledQueryName {
		t.Fatalf("ListScheduledQueriesRequest.DisplayName = %#v, want %q", listRequest.DisplayName, resource.Spec.ScheduledQueryName)
	}
	if getRequest.ApmDomainId == nil || *getRequest.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("GetScheduledQueryRequest.ApmDomainId = %#v, want %q", getRequest.ApmDomainId, resource.Spec.ApmDomainId)
	}
	if getRequest.ScheduledQueryId == nil || *getRequest.ScheduledQueryId != existingID {
		t.Fatalf("GetScheduledQueryRequest.ScheduledQueryId = %#v, want %q", getRequest.ScheduledQueryId, existingID)
	}
	if resource.Status.ApmDomainId != resource.Spec.ApmDomainId {
		t.Fatalf("Status.ApmDomainId = %q, want mirrored domain %q", resource.Status.ApmDomainId, resource.Spec.ApmDomainId)
	}
	if resource.Status.ScheduledQueryName != resource.Spec.ScheduledQueryName {
		t.Fatalf("Status.ScheduledQueryName = %q, want %q", resource.Status.ScheduledQueryName, resource.Spec.ScheduledQueryName)
	}
}

func TestScheduledQueryDeleteWithoutTrackedIdentitySkipsOCI(t *testing.T) {
	t.Parallel()

	ociCalled := false
	client := newScheduledQueryServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubScheduledQueryOCIClient{
			get: func(context.Context, apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error) {
				ociCalled = true
				return apmtracessdk.GetScheduledQueryResponse{}, nil
			},
			list: func(context.Context, apmtracessdk.ListScheduledQueriesRequest) (apmtracessdk.ListScheduledQueriesResponse, error) {
				ociCalled = true
				return apmtracessdk.ListScheduledQueriesResponse{}, nil
			},
			delete: func(context.Context, apmtracessdk.DeleteScheduledQueryRequest) (apmtracessdk.DeleteScheduledQueryResponse, error) {
				ociCalled = true
				return apmtracessdk.DeleteScheduledQueryResponse{}, nil
			},
		},
	)

	deleted, err := client.Delete(context.Background(), &apmtracesv1beta1.ScheduledQuery{
		Spec: apmtracesv1beta1.ScheduledQuerySpec{
			ApmDomainId:        "ocid1.apmdomain.oc1..example",
			ScheduledQueryName: "scheduled-query",
		},
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for an untracked ScheduledQuery")
	}
	if ociCalled {
		t.Fatal("Delete() called OCI despite no tracked ScheduledQuery identity")
	}
}

func TestScheduledQueryDeleteWithTrackedIdentityDelegates(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	getCalls := 0
	client := newScheduledQueryServiceClientWithOCIClient(
		loggerutil.OSOKLogger{},
		stubScheduledQueryOCIClient{
			get: func(context.Context, apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error) {
				getCalls++
				lifecycleState := apmtracessdk.LifecycleStatesActive
				if getCalls > 1 {
					lifecycleState = apmtracessdk.LifecycleStatesDeleted
				}
				return apmtracessdk.GetScheduledQueryResponse{
					ScheduledQuery: apmtracessdk.ScheduledQuery{
						Id:             common.String("ocid1.scheduledquery.oc1..tracked"),
						LifecycleState: lifecycleState,
					},
				}, nil
			},
			delete: func(context.Context, apmtracessdk.DeleteScheduledQueryRequest) (apmtracessdk.DeleteScheduledQueryResponse, error) {
				deleteCalled = true
				return apmtracessdk.DeleteScheduledQueryResponse{}, nil
			},
		},
	)

	deleted, err := client.Delete(context.Background(), &apmtracesv1beta1.ScheduledQuery{
		Status: apmtracesv1beta1.ScheduledQueryStatus{
			Id: "ocid1.scheduledquery.oc1..tracked",
		},
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after generatedruntime confirms delete")
	}
	if !deleteCalled {
		t.Fatal("Delete() did not delegate OCI delete for a tracked ScheduledQuery")
	}
	if getCalls != 2 {
		t.Fatalf("GetScheduledQuery() calls = %d, want pre-delete and post-delete confirmation reads", getCalls)
	}
}

type stubScheduledQueryOCIClient struct {
	create func(context.Context, apmtracessdk.CreateScheduledQueryRequest) (apmtracessdk.CreateScheduledQueryResponse, error)
	get    func(context.Context, apmtracessdk.GetScheduledQueryRequest) (apmtracessdk.GetScheduledQueryResponse, error)
	list   func(context.Context, apmtracessdk.ListScheduledQueriesRequest) (apmtracessdk.ListScheduledQueriesResponse, error)
	update func(context.Context, apmtracessdk.UpdateScheduledQueryRequest) (apmtracessdk.UpdateScheduledQueryResponse, error)
	delete func(context.Context, apmtracessdk.DeleteScheduledQueryRequest) (apmtracessdk.DeleteScheduledQueryResponse, error)
}

func (s stubScheduledQueryOCIClient) CreateScheduledQuery(
	ctx context.Context,
	req apmtracessdk.CreateScheduledQueryRequest,
) (apmtracessdk.CreateScheduledQueryResponse, error) {
	if s.create == nil {
		return apmtracessdk.CreateScheduledQueryResponse{}, nil
	}
	return s.create(ctx, req)
}

func (s stubScheduledQueryOCIClient) GetScheduledQuery(
	ctx context.Context,
	req apmtracessdk.GetScheduledQueryRequest,
) (apmtracessdk.GetScheduledQueryResponse, error) {
	if s.get == nil {
		return apmtracessdk.GetScheduledQueryResponse{}, nil
	}
	return s.get(ctx, req)
}

func (s stubScheduledQueryOCIClient) ListScheduledQueries(
	ctx context.Context,
	req apmtracessdk.ListScheduledQueriesRequest,
) (apmtracessdk.ListScheduledQueriesResponse, error) {
	if s.list == nil {
		return apmtracessdk.ListScheduledQueriesResponse{}, nil
	}
	return s.list(ctx, req)
}

func (s stubScheduledQueryOCIClient) UpdateScheduledQuery(
	ctx context.Context,
	req apmtracessdk.UpdateScheduledQueryRequest,
) (apmtracessdk.UpdateScheduledQueryResponse, error) {
	if s.update == nil {
		return apmtracessdk.UpdateScheduledQueryResponse{}, nil
	}
	return s.update(ctx, req)
}

func (s stubScheduledQueryOCIClient) DeleteScheduledQuery(
	ctx context.Context,
	req apmtracessdk.DeleteScheduledQueryRequest,
) (apmtracessdk.DeleteScheduledQueryResponse, error) {
	if s.delete == nil {
		return apmtracessdk.DeleteScheduledQueryResponse{}, nil
	}
	return s.delete(ctx, req)
}
