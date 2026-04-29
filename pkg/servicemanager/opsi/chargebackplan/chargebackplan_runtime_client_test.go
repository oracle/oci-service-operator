/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package chargebackplan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeChargebackPlanOCI struct {
	create func(context.Context, opsisdk.CreateChargebackPlanRequest) (opsisdk.CreateChargebackPlanResponse, error)
	get    func(context.Context, opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error)
	list   func(context.Context, opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error)
	update func(context.Context, opsisdk.UpdateChargebackPlanRequest) (opsisdk.UpdateChargebackPlanResponse, error)
	delete func(context.Context, opsisdk.DeleteChargebackPlanRequest) (opsisdk.DeleteChargebackPlanResponse, error)
}

func (f *fakeChargebackPlanOCI) CreateChargebackPlan(ctx context.Context, request opsisdk.CreateChargebackPlanRequest) (opsisdk.CreateChargebackPlanResponse, error) {
	if f.create == nil {
		return opsisdk.CreateChargebackPlanResponse{}, fmt.Errorf("unexpected CreateChargebackPlan call")
	}
	return f.create(ctx, request)
}

func (f *fakeChargebackPlanOCI) GetChargebackPlan(ctx context.Context, request opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
	if f.get == nil {
		return opsisdk.GetChargebackPlanResponse{}, fmt.Errorf("unexpected GetChargebackPlan call")
	}
	return f.get(ctx, request)
}

func (f *fakeChargebackPlanOCI) ListChargebackPlans(ctx context.Context, request opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error) {
	if f.list == nil {
		return opsisdk.ListChargebackPlansResponse{}, fmt.Errorf("unexpected ListChargebackPlans call")
	}
	return f.list(ctx, request)
}

func (f *fakeChargebackPlanOCI) UpdateChargebackPlan(ctx context.Context, request opsisdk.UpdateChargebackPlanRequest) (opsisdk.UpdateChargebackPlanResponse, error) {
	if f.update == nil {
		return opsisdk.UpdateChargebackPlanResponse{}, fmt.Errorf("unexpected UpdateChargebackPlan call")
	}
	return f.update(ctx, request)
}

func (f *fakeChargebackPlanOCI) DeleteChargebackPlan(ctx context.Context, request opsisdk.DeleteChargebackPlanRequest) (opsisdk.DeleteChargebackPlanResponse, error) {
	if f.delete == nil {
		return opsisdk.DeleteChargebackPlanResponse{}, fmt.Errorf("unexpected DeleteChargebackPlan call")
	}
	return f.delete(ctx, request)
}

func TestBuildChargebackPlanCreateBodyUsesExadataDetails(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Spec.PlanCustomItems = []opsiv1beta1.ChargebackPlanPlanCustomItem{{
		Name:           "infrastructureCost",
		Value:          "1200",
		IsCustomizable: false,
	}}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	resource.Spec.FreeformTags = map[string]string{"env": "test"}

	body, err := buildChargebackPlanCreateBody(resource)
	if err != nil {
		t.Fatalf("buildChargebackPlanCreateBody() error = %v", err)
	}
	details, ok := body.(opsisdk.CreateChargebackPlanExadataDetails)
	if !ok {
		t.Fatalf("create body type = %T, want opsi.CreateChargebackPlanExadataDetails", body)
	}
	if got, want := chargebackPlanString(details.CompartmentId), resource.Spec.CompartmentId; got != want {
		t.Fatalf("create body compartmentId = %q, want %q", got, want)
	}
	if len(details.PlanCustomItems) != 1 || details.PlanCustomItems[0].IsCustomizable == nil || *details.PlanCustomItems[0].IsCustomizable {
		t.Fatalf("create body planCustomItems[0].isCustomizable = %#v, want explicit false", details.PlanCustomItems)
	}
	payload, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal create body: %v", err)
	}
	if strings.Contains(string(payload), "jsonData") {
		t.Fatalf("create body unexpectedly includes jsonData: %s", payload)
	}
	if !strings.Contains(string(payload), `"entitySource":"CHARGEBACK_EXADATA"`) {
		t.Fatalf("create body = %s, want CHARGEBACK_EXADATA discriminator", payload)
	}
}

func TestBuildChargebackPlanCreateBodySupportsJsonData(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Spec.JsonData = `{"entitySource":"CHARGEBACK_EXADATA","compartmentId":"ocid1.compartment.oc1..aaaa","planName":"example-plan","planType":"EQUAL_ALLOCATION","planCustomItems":[{"name":"statistic","value":"AVG","isCustomizable":false}]}`

	body, err := buildChargebackPlanCreateBody(resource)
	if err != nil {
		t.Fatalf("buildChargebackPlanCreateBody() error = %v", err)
	}
	details, ok := body.(opsisdk.CreateChargebackPlanExadataDetails)
	if !ok {
		t.Fatalf("create body type = %T, want opsi.CreateChargebackPlanExadataDetails", body)
	}
	if got := chargebackPlanString(details.CompartmentId); got != resource.Spec.CompartmentId {
		t.Fatalf("create jsonData compartmentId = %q", got)
	}
	if len(details.PlanCustomItems) != 1 || details.PlanCustomItems[0].IsCustomizable == nil || *details.PlanCustomItems[0].IsCustomizable {
		t.Fatalf("create jsonData planCustomItems[0].isCustomizable = %#v, want explicit false", details.PlanCustomItems)
	}
}

func TestBuildChargebackPlanCreateBodyRejectsJsonDataIdentityConflict(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Spec.JsonData = `{"entitySource":"CHARGEBACK_EXADATA","compartmentId":"ocid1.compartment.oc1..other","planName":"example-plan","planType":"EQUAL_ALLOCATION"}`

	_, err := buildChargebackPlanCreateBody(resource)
	if err == nil || !strings.Contains(err.Error(), "jsonData conflicts with spec field(s): compartmentId") {
		t.Fatalf("buildChargebackPlanCreateBody() error = %v, want compartmentId conflict", err)
	}
}

func TestChargebackPlanCreateOrUpdateCreatesAndRecordsStatus(t *testing.T) {
	resource := baseChargebackPlan()
	fake := &fakeChargebackPlanOCI{}
	fake.create = func(_ context.Context, request opsisdk.CreateChargebackPlanRequest) (opsisdk.CreateChargebackPlanResponse, error) {
		details, ok := request.CreateChargebackPlanDetails.(opsisdk.CreateChargebackPlanExadataDetails)
		if !ok {
			t.Fatalf("CreateChargebackPlan details type = %T, want Exadata details", request.CreateChargebackPlanDetails)
		}
		if got := chargebackPlanString(details.PlanName); got != resource.Spec.PlanName {
			t.Fatalf("CreateChargebackPlan planName = %q, want %q", got, resource.Spec.PlanName)
		}
		if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
			t.Fatal("CreateChargebackPlan opc retry token is empty")
		}
		return opsisdk.CreateChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..created", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateCreating),
			OpcRequestId:   common.String("opc-create-1"),
		}, nil
	}
	fake.list = func(context.Context, opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error) {
		return opsisdk.ListChargebackPlansResponse{}, nil
	}
	fake.get = func(_ context.Context, request opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		if got := chargebackPlanString(request.ChargebackplanId); got != "ocid1.chargebackplan.oc1..created" {
			t.Fatalf("GetChargebackPlan id = %q, want created id", got)
		}
		return opsisdk.GetChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..created", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive),
			OpcRequestId:   common.String("opc-get-1"),
		}, nil
	}

	response, err := newChargebackPlanTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertChargebackPlanStatusID(t, resource, "ocid1.chargebackplan.oc1..created")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := resource.Status.LifecycleState; got != string(opsisdk.LifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestChargebackPlanBindsExistingFromPaginatedList(t *testing.T) {
	resource := baseChargebackPlan()
	listCalls := 0
	fake := &fakeChargebackPlanOCI{}
	fake.create = func(context.Context, opsisdk.CreateChargebackPlanRequest) (opsisdk.CreateChargebackPlanResponse, error) {
		t.Fatal("CreateChargebackPlan should not be called when list finds an existing plan")
		return opsisdk.CreateChargebackPlanResponse{}, nil
	}
	fake.list = func(_ context.Context, request opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error) {
		listCalls++
		if got := chargebackPlanString(request.CompartmentId); got != resource.Spec.CompartmentId {
			t.Fatalf("ListChargebackPlans compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
		}
		switch listCalls {
		case 1:
			if request.Page != nil {
				t.Fatalf("first ListChargebackPlans page = %q, want nil", chargebackPlanString(request.Page))
			}
			return opsisdk.ListChargebackPlansResponse{
				ChargebackPlanCollection: opsisdk.ChargebackPlanCollection{Items: []opsisdk.ChargebackPlanSummary{
					chargebackPlanSummary("ocid1.chargebackplan.oc1..other", "other-plan", resource.Spec.PlanType, opsisdk.LifecycleStateActive),
				}},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			if got := chargebackPlanString(request.Page); got != "page-2" {
				t.Fatalf("second ListChargebackPlans page = %q, want page-2", got)
			}
			return opsisdk.ListChargebackPlansResponse{
				ChargebackPlanCollection: opsisdk.ChargebackPlanCollection{Items: []opsisdk.ChargebackPlanSummary{
					chargebackPlanSummary("ocid1.chargebackplan.oc1..existing", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive),
				}},
			}, nil
		default:
			t.Fatalf("ListChargebackPlans call count = %d, want 2", listCalls)
			return opsisdk.ListChargebackPlansResponse{}, nil
		}
	}
	fake.get = func(_ context.Context, request opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		if got := chargebackPlanString(request.ChargebackplanId); got != "ocid1.chargebackplan.oc1..existing" {
			t.Fatalf("GetChargebackPlan id = %q, want existing id", got)
		}
		return opsisdk.GetChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..existing", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive),
		}, nil
	}

	response, err := newChargebackPlanTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if listCalls != 2 {
		t.Fatalf("ListChargebackPlans calls = %d, want 2", listCalls)
	}
	assertChargebackPlanStatusID(t, resource, "ocid1.chargebackplan.oc1..existing")
}

func TestChargebackPlanLookupIgnoresDeletingAndDeletedMatches(t *testing.T) {
	resource := baseChargebackPlan()
	createCalled := false
	fake := &fakeChargebackPlanOCI{}
	fake.list = func(context.Context, opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error) {
		return opsisdk.ListChargebackPlansResponse{
			ChargebackPlanCollection: opsisdk.ChargebackPlanCollection{Items: []opsisdk.ChargebackPlanSummary{
				chargebackPlanSummary("ocid1.chargebackplan.oc1..deleting", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateDeleting),
				chargebackPlanSummary("ocid1.chargebackplan.oc1..deleted", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateDeleted),
			}},
		}, nil
	}
	fake.create = func(context.Context, opsisdk.CreateChargebackPlanRequest) (opsisdk.CreateChargebackPlanResponse, error) {
		createCalled = true
		return opsisdk.CreateChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..created", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateCreating),
		}, nil
	}
	fake.get = func(_ context.Context, request opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		if got := chargebackPlanString(request.ChargebackplanId); got != "ocid1.chargebackplan.oc1..created" {
			t.Fatalf("GetChargebackPlan id = %q, want created id", got)
		}
		return opsisdk.GetChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..created", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive),
		}, nil
	}

	response, err := newChargebackPlanTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !createCalled {
		t.Fatal("CreateChargebackPlan was not called after terminal list matches were ignored")
	}
	assertChargebackPlanStatusID(t, resource, "ocid1.chargebackplan.oc1..created")
}

func TestChargebackPlanNoOpReconcileDoesNotUpdate(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.chargebackplan.oc1..tracked")
	resource.Status.Id = "ocid1.chargebackplan.oc1..tracked"
	fake := &fakeChargebackPlanOCI{}
	fake.get = func(_ context.Context, request opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		if got := chargebackPlanString(request.ChargebackplanId); got != "ocid1.chargebackplan.oc1..tracked" {
			t.Fatalf("GetChargebackPlan id = %q, want tracked id", got)
		}
		return opsisdk.GetChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..tracked", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive),
		}, nil
	}
	fake.update = func(context.Context, opsisdk.UpdateChargebackPlanRequest) (opsisdk.UpdateChargebackPlanResponse, error) {
		t.Fatal("UpdateChargebackPlan should not be called for matching readback")
		return opsisdk.UpdateChargebackPlanResponse{}, nil
	}

	response, err := newChargebackPlanTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
}

func TestChargebackPlanMutableUpdateShapesRequestAndRefreshesStatus(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.chargebackplan.oc1..tracked")
	resource.Status.Id = "ocid1.chargebackplan.oc1..tracked"
	resource.Spec.PlanName = "updated-plan"
	resource.Spec.PlanDescription = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
	resource.Spec.PlanCustomItems = []opsiv1beta1.ChargebackPlanPlanCustomItem{{
		Name:           "percentile",
		Value:          "P95",
		IsCustomizable: false,
	}}

	getCalls := 0
	updateCalled := false
	fake := &fakeChargebackPlanOCI{
		get:    chargebackPlanMutableUpdateGet(t, resource, &getCalls),
		update: chargebackPlanMutableUpdateCall(t, resource, &updateCalled),
	}

	response, err := newChargebackPlanTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !updateCalled {
		t.Fatal("UpdateChargebackPlan was not called")
	}
	if got := resource.Status.PlanName; got != resource.Spec.PlanName {
		t.Fatalf("status.planName = %q, want %q", got, resource.Spec.PlanName)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func chargebackPlanMutableUpdateGet(
	t *testing.T,
	resource *opsiv1beta1.ChargebackPlan,
	getCalls *int,
) func(context.Context, opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
	t.Helper()
	return func(_ context.Context, request opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		*getCalls = *getCalls + 1
		if got := chargebackPlanString(request.ChargebackplanId); got != "ocid1.chargebackplan.oc1..tracked" {
			t.Fatalf("GetChargebackPlan id = %q, want tracked id", got)
		}
		if *getCalls == 1 {
			return opsisdk.GetChargebackPlanResponse{ChargebackPlan: chargebackPlanMutableCurrent(resource)}, nil
		}
		return opsisdk.GetChargebackPlanResponse{ChargebackPlan: chargebackPlanMutableUpdated(resource)}, nil
	}
}

func chargebackPlanMutableCurrent(resource *opsiv1beta1.ChargebackPlan) opsisdk.ChargebackPlan {
	current := chargebackPlanSDK("ocid1.chargebackplan.oc1..tracked", "old-plan", resource.Spec.PlanType, opsisdk.LifecycleStateActive)
	current.PlanDescription = common.String("old description")
	current.FreeformTags = map[string]string{"env": "dev"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
	current.PlanCustomItems = []opsisdk.CreatePlanCustomItemDetails{{
		Name:           common.String("percentile"),
		Value:          common.String("P90"),
		IsCustomizable: common.Bool(true),
	}}
	return current
}

func chargebackPlanMutableUpdated(resource *opsiv1beta1.ChargebackPlan) opsisdk.ChargebackPlan {
	updated := chargebackPlanSDK("ocid1.chargebackplan.oc1..tracked", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive)
	updated.PlanDescription = common.String(resource.Spec.PlanDescription)
	updated.FreeformTags = map[string]string{"env": "prod"}
	updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
	updated.PlanCustomItems = chargebackPlanCustomItems(resource.Spec.PlanCustomItems)
	return updated
}

func chargebackPlanMutableUpdateCall(
	t *testing.T,
	resource *opsiv1beta1.ChargebackPlan,
	updateCalled *bool,
) func(context.Context, opsisdk.UpdateChargebackPlanRequest) (opsisdk.UpdateChargebackPlanResponse, error) {
	t.Helper()
	return func(_ context.Context, request opsisdk.UpdateChargebackPlanRequest) (opsisdk.UpdateChargebackPlanResponse, error) {
		*updateCalled = true
		assertChargebackPlanMutableUpdateRequest(t, request, resource)
		return opsisdk.UpdateChargebackPlanResponse{OpcRequestId: common.String("opc-update-1")}, nil
	}
}

func assertChargebackPlanMutableUpdateRequest(
	t *testing.T,
	request opsisdk.UpdateChargebackPlanRequest,
	resource *opsiv1beta1.ChargebackPlan,
) {
	t.Helper()
	if got := chargebackPlanString(request.ChargebackplanId); got != "ocid1.chargebackplan.oc1..tracked" {
		t.Fatalf("UpdateChargebackPlan id = %q, want tracked id", got)
	}
	details := request.UpdateChargebackPlanDetails
	if got := chargebackPlanString(details.PlanName); got != resource.Spec.PlanName {
		t.Fatalf("update planName = %q, want %q", got, resource.Spec.PlanName)
	}
	if got := chargebackPlanString(details.PlanDescription); got != resource.Spec.PlanDescription {
		t.Fatalf("update planDescription = %q, want %q", got, resource.Spec.PlanDescription)
	}
	if len(details.PlanCustomItems) != 1 || details.PlanCustomItems[0].IsCustomizable == nil || *details.PlanCustomItems[0].IsCustomizable {
		t.Fatalf("update planCustomItems[0].isCustomizable = %#v, want explicit false", details.PlanCustomItems)
	}
}

func TestChargebackPlanImmutableDriftRejectedBeforeUpdate(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.chargebackplan.oc1..tracked")
	resource.Status.Id = "ocid1.chargebackplan.oc1..tracked"
	resource.Spec.PlanType = "UNUSED_ALLOCATION"
	fake := &fakeChargebackPlanOCI{}
	fake.get = func(context.Context, opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		return opsisdk.GetChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..tracked", resource.Spec.PlanName, "EQUAL_ALLOCATION", opsisdk.LifecycleStateActive),
		}, nil
	}
	fake.update = func(context.Context, opsisdk.UpdateChargebackPlanRequest) (opsisdk.UpdateChargebackPlanResponse, error) {
		t.Fatal("UpdateChargebackPlan should not be called for create-only planType drift")
		return opsisdk.UpdateChargebackPlanResponse{}, nil
	}

	_, err := newChargebackPlanTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "planType changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want planType replacement error", err)
	}
}

func TestChargebackPlanJsonDataDriftRejectedBeforeUpdate(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Spec.JsonData = `{"entitySource":"CHARGEBACK_EXADATA","compartmentId":"ocid1.compartment.oc1..aaaa","planName":"example-plan","planType":"EQUAL_ALLOCATION","planDescription":"json desired"}`
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.chargebackplan.oc1..tracked")
	resource.Status.Id = "ocid1.chargebackplan.oc1..tracked"
	fake := &fakeChargebackPlanOCI{}
	fake.get = func(context.Context, opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		current := chargebackPlanSDK("ocid1.chargebackplan.oc1..tracked", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive)
		current.PlanDescription = common.String("live description")
		return opsisdk.GetChargebackPlanResponse{ChargebackPlan: current}, nil
	}
	fake.update = func(context.Context, opsisdk.UpdateChargebackPlanRequest) (opsisdk.UpdateChargebackPlanResponse, error) {
		t.Fatal("UpdateChargebackPlan should not be called for jsonData drift")
		return opsisdk.UpdateChargebackPlanResponse{}, nil
	}

	_, err := newChargebackPlanTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "jsonData.planDescription") {
		t.Fatalf("CreateOrUpdate() error = %v, want jsonData planDescription drift", err)
	}
}

func TestChargebackPlanDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.chargebackplan.oc1..tracked")
	resource.Status.Id = "ocid1.chargebackplan.oc1..tracked"
	getCalls := 0
	fake := &fakeChargebackPlanOCI{}
	fake.delete = func(_ context.Context, request opsisdk.DeleteChargebackPlanRequest) (opsisdk.DeleteChargebackPlanResponse, error) {
		if got := chargebackPlanString(request.ChargebackplanId); got != "ocid1.chargebackplan.oc1..tracked" {
			t.Fatalf("DeleteChargebackPlan id = %q, want tracked id", got)
		}
		return opsisdk.DeleteChargebackPlanResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	fake.get = func(context.Context, opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		getCalls++
		if getCalls == 1 {
			return opsisdk.GetChargebackPlanResponse{
				ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..tracked", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateActive),
			}, nil
		}
		return opsisdk.GetChargebackPlanResponse{
			ChargebackPlan: chargebackPlanSDK("ocid1.chargebackplan.oc1..tracked", resource.Spec.PlanName, resource.Spec.PlanType, opsisdk.LifecycleStateDeleting),
		}, nil
	}

	deleted, err := newChargebackPlanTestClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycleState is DELETING")
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete phase", resource.Status.OsokStatus.Async.Current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func TestChargebackPlanDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.chargebackplan.oc1..tracked")
	resource.Status.Id = "ocid1.chargebackplan.oc1..tracked"
	fake := &fakeChargebackPlanOCI{}
	fake.delete = func(context.Context, opsisdk.DeleteChargebackPlanRequest) (opsisdk.DeleteChargebackPlanResponse, error) {
		return opsisdk.DeleteChargebackPlanResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	fake.get = func(context.Context, opsisdk.GetChargebackPlanRequest) (opsisdk.GetChargebackPlanResponse, error) {
		return opsisdk.GetChargebackPlanResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
	}
	fake.list = func(context.Context, opsisdk.ListChargebackPlansRequest) (opsisdk.ListChargebackPlansResponse, error) {
		return opsisdk.ListChargebackPlansResponse{}, nil
	}

	deleted, err := newChargebackPlanTestClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
}

func TestChargebackPlanDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := baseChargebackPlan()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.chargebackplan.oc1..tracked")
	resource.Status.Id = "ocid1.chargebackplan.oc1..tracked"
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-1"
	fake := &fakeChargebackPlanOCI{}
	fake.delete = func(context.Context, opsisdk.DeleteChargebackPlanRequest) (opsisdk.DeleteChargebackPlanResponse, error) {
		return opsisdk.DeleteChargebackPlanResponse{}, authErr
	}

	deleted, err := newChargebackPlanTestClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not found")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-auth-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-auth-1", got)
	}
}

func newChargebackPlanTestClient(fake *fakeChargebackPlanOCI) ChargebackPlanServiceClient {
	hooks := ChargebackPlanRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.ChargebackPlan]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.ChargebackPlan]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.ChargebackPlan]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.ChargebackPlan]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.ChargebackPlan]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.ChargebackPlan]{},
		Create: runtimeOperationHooks[opsisdk.CreateChargebackPlanRequest, opsisdk.CreateChargebackPlanResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateChargebackPlanDetails", RequestName: "CreateChargebackPlanDetails", Contribution: "body"}},
			Call:   fake.CreateChargebackPlan,
		},
		Get: runtimeOperationHooks[opsisdk.GetChargebackPlanRequest, opsisdk.GetChargebackPlanResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ChargebackplanId", RequestName: "chargebackplanId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.GetChargebackPlan,
		},
		List: runtimeOperationHooks[opsisdk.ListChargebackPlansRequest, opsisdk.ListChargebackPlansResponse]{
			Fields: chargebackPlanListFields(),
			Call:   fake.ListChargebackPlans,
		},
		Update: runtimeOperationHooks[opsisdk.UpdateChargebackPlanRequest, opsisdk.UpdateChargebackPlanResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ChargebackplanId", RequestName: "chargebackplanId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateChargebackPlanDetails", RequestName: "UpdateChargebackPlanDetails", Contribution: "body"},
			},
			Call: fake.UpdateChargebackPlan,
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteChargebackPlanRequest, opsisdk.DeleteChargebackPlanResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ChargebackplanId", RequestName: "chargebackplanId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.DeleteChargebackPlan,
		},
	}
	applyChargebackPlanRuntimeHooks(&hooks)
	manager := &ChargebackPlanServiceManager{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}
	delegate := defaultChargebackPlanServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.ChargebackPlan](
			buildChargebackPlanGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapChargebackPlanGeneratedClient(hooks, delegate)
}

func baseChargebackPlan() *opsiv1beta1.ChargebackPlan {
	return &opsiv1beta1.ChargebackPlan{
		Spec: opsiv1beta1.ChargebackPlanSpec{
			CompartmentId: "ocid1.compartment.oc1..aaaa",
			PlanName:      "example-plan",
			PlanType:      "EQUAL_ALLOCATION",
			EntitySource:  string(opsisdk.ChargebackPlanEntitySourceChargebackExadata),
		},
	}
}

func chargebackPlanSDK(id string, planName string, planType string, state opsisdk.LifecycleStateEnum) opsisdk.ChargebackPlan {
	return opsisdk.ChargebackPlan{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..aaaa"),
		PlanName:       common.String(planName),
		PlanType:       common.String(planType),
		LifecycleState: state,
		EntitySource:   opsisdk.ChargebackPlanEntitySourceChargebackExadata,
	}
}

func chargebackPlanSummary(id string, planName string, planType string, state opsisdk.LifecycleStateEnum) opsisdk.ChargebackPlanSummary {
	return opsisdk.ChargebackPlanSummary{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..aaaa"),
		PlanName:       common.String(planName),
		PlanType:       common.String(planType),
		EntitySource:   opsisdk.ChargebackPlanEntitySourceChargebackExadata,
		LifecycleState: state,
	}
}

func assertChargebackPlanStatusID(t *testing.T, resource *opsiv1beta1.ChargebackPlan, want string) {
	t.Helper()
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}
