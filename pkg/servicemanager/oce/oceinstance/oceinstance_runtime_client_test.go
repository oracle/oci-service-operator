/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package oceinstance

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocesdk "github.com/oracle/oci-go-sdk/v65/oce"
	ocev1beta1 "github.com/oracle/oci-service-operator/api/oce/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeOceInstanceOCIClient struct {
	createFn      func(context.Context, ocesdk.CreateOceInstanceRequest) (ocesdk.CreateOceInstanceResponse, error)
	getFn         func(context.Context, ocesdk.GetOceInstanceRequest) (ocesdk.GetOceInstanceResponse, error)
	listFn        func(context.Context, ocesdk.ListOceInstancesRequest) (ocesdk.ListOceInstancesResponse, error)
	updateFn      func(context.Context, ocesdk.UpdateOceInstanceRequest) (ocesdk.UpdateOceInstanceResponse, error)
	deleteFn      func(context.Context, ocesdk.DeleteOceInstanceRequest) (ocesdk.DeleteOceInstanceResponse, error)
	workRequestFn func(context.Context, ocesdk.GetWorkRequestRequest) (ocesdk.GetWorkRequestResponse, error)

	createRequests      []ocesdk.CreateOceInstanceRequest
	getRequests         []ocesdk.GetOceInstanceRequest
	listRequests        []ocesdk.ListOceInstancesRequest
	updateRequests      []ocesdk.UpdateOceInstanceRequest
	deleteRequests      []ocesdk.DeleteOceInstanceRequest
	workRequestRequests []ocesdk.GetWorkRequestRequest
}

func (f *fakeOceInstanceOCIClient) CreateOceInstance(
	ctx context.Context,
	request ocesdk.CreateOceInstanceRequest,
) (ocesdk.CreateOceInstanceResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return ocesdk.CreateOceInstanceResponse{}, nil
}

func (f *fakeOceInstanceOCIClient) GetOceInstance(
	ctx context.Context,
	request ocesdk.GetOceInstanceRequest,
) (ocesdk.GetOceInstanceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return ocesdk.GetOceInstanceResponse{}, nil
}

func (f *fakeOceInstanceOCIClient) ListOceInstances(
	ctx context.Context,
	request ocesdk.ListOceInstancesRequest,
) (ocesdk.ListOceInstancesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return ocesdk.ListOceInstancesResponse{}, nil
}

func (f *fakeOceInstanceOCIClient) UpdateOceInstance(
	ctx context.Context,
	request ocesdk.UpdateOceInstanceRequest,
) (ocesdk.UpdateOceInstanceResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return ocesdk.UpdateOceInstanceResponse{}, nil
}

func (f *fakeOceInstanceOCIClient) DeleteOceInstance(
	ctx context.Context,
	request ocesdk.DeleteOceInstanceRequest,
) (ocesdk.DeleteOceInstanceResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return ocesdk.DeleteOceInstanceResponse{}, nil
}

func (f *fakeOceInstanceOCIClient) GetWorkRequest(
	ctx context.Context,
	request ocesdk.GetWorkRequestRequest,
) (ocesdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return ocesdk.GetWorkRequestResponse{}, nil
}

func TestReviewedOceInstanceRuntimeSemanticsEncodesReviewedContract(t *testing.T) {
	t.Parallel()

	got := reviewedOceInstanceRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedOceInstanceRuntimeSemantics() = nil")
	}

	if got.FormalService != "oce" {
		t.Fatalf("FormalService = %q, want oce", got.FormalService)
	}
	if got.FormalSlug != "oceinstance" {
		t.Fatalf("FormalSlug = %q, want oceinstance", got.FormalSlug)
	}
	if got.Async == nil || got.Async.WorkRequest == nil {
		t.Fatalf("Async = %#v, want service-sdk workrequest semantics", got.Async)
	}
	assertOceInstanceStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertOceInstanceStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertOceInstanceStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertOceInstanceStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertOceInstanceStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "tenancyId"})
	assertOceInstanceStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"addOnFeatures",
		"definedTags",
		"description",
		"drRegion",
		"freeformTags",
		"instanceLicenseType",
		"instanceUsageType",
		"wafPrimaryDomain",
	})
	assertOceInstanceStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{
		"adminEmail",
		"compartmentId",
		"identityStripe.serviceName",
		"identityStripe.tenancy",
		"instanceAccessType",
		"name",
		"objectStorageNamespace",
		"tenancyId",
		"tenancyName",
		"upgradeSchedule",
	})
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetOceInstance" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetOceInstance", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetOceInstance" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetOceInstance", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetOceInstance/ListOceInstances confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}

	resource := newOceInstanceTestResource()
	resource.Spec.IdcsAccessToken = "updated-token"
	resource.Spec.LifecycleDetails = "IGNORED"
	normalizeOceInstanceDesiredState(resource, makeSDKOceInstance("ocid1.oceinstance.oc1..existing", resource, ocesdk.LifecycleStateActive))
	if resource.Spec.IdcsAccessToken != "" {
		t.Fatalf("normalizeOceInstanceDesiredState() idcsAccessToken = %q, want cleared create-time input", resource.Spec.IdcsAccessToken)
	}
	if resource.Spec.LifecycleDetails != "" {
		t.Fatalf("normalizeOceInstanceDesiredState() lifecycleDetails = %q, want cleared status-only field", resource.Spec.LifecycleDetails)
	}
}

func TestGuardOceInstanceExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := newOceInstanceTestResource()
	resource.Spec.Name = ""

	decision, err := guardOceInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardOceInstanceExistingBeforeCreate(empty name) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardOceInstanceExistingBeforeCreate(empty name) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.Name = "oce-runtime-test"
	decision, err = guardOceInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardOceInstanceExistingBeforeCreate(non-empty name) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardOceInstanceExistingBeforeCreate(non-empty name) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildOceInstanceCreateBodyOmitsEmptyOptionalFields(t *testing.T) {
	t.Parallel()

	resource := newOceInstanceTestResource()
	resource.Spec.InstanceUsageType = "PRIMARY"
	resource.Spec.InstanceAccessType = "PRIVATE"
	resource.Spec.InstanceLicenseType = "NEW"
	resource.Spec.UpgradeSchedule = "DELAYED_UPGRADE"

	details, err := buildOceInstanceCreateBody(resource)
	if err != nil {
		t.Fatalf("buildOceInstanceCreateBody() error = %v", err)
	}

	requireOceInstanceStringPtr(t, "details.name", details.Name, resource.Spec.Name)
	requireOceInstanceStringPtr(t, "details.compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	if details.IdentityStripe != nil {
		t.Fatalf("details.IdentityStripe = %#v, want nil when spec.identityStripe is empty", details.IdentityStripe)
	}
	if details.InstanceUsageType != ocesdk.CreateOceInstanceDetailsInstanceUsageTypePrimary {
		t.Fatalf("details.InstanceUsageType = %q, want PRIMARY", details.InstanceUsageType)
	}
	if details.InstanceAccessType != ocesdk.CreateOceInstanceDetailsInstanceAccessTypePrivate {
		t.Fatalf("details.InstanceAccessType = %q, want PRIVATE", details.InstanceAccessType)
	}
	if details.InstanceLicenseType != ocesdk.LicenseTypeNew {
		t.Fatalf("details.InstanceLicenseType = %q, want NEW", details.InstanceLicenseType)
	}
	if details.UpgradeSchedule != ocesdk.OceInstanceUpgradeScheduleDelayedUpgrade {
		t.Fatalf("details.UpgradeSchedule = %q, want DELAYED_UPGRADE", details.UpgradeSchedule)
	}
}

func TestBuildOceInstanceUpdateBodyPreservesClearToEmpty(t *testing.T) {
	t.Parallel()

	resource := newOceInstanceTestResource()
	resource.Spec.Description = ""
	resource.Spec.WafPrimaryDomain = ""
	resource.Spec.DrRegion = ""
	resource.Spec.AddOnFeatures = []string{}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := makeSDKOceInstance("ocid1.oceinstance.oc1..existing", newOceInstanceTestResource(), ocesdk.LifecycleStateActive)

	details, updateNeeded, err := buildOceInstanceUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildOceInstanceUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildOceInstanceUpdateBody() updateNeeded = false, want true")
	}
	requireOceInstanceStringPtr(t, "details.description", details.Description, "")
	requireOceInstanceStringPtr(t, "details.wafPrimaryDomain", details.WafPrimaryDomain, "")
	requireOceInstanceStringPtr(t, "details.drRegion", details.DrRegion, "")
	if len(details.AddOnFeatures) != 0 {
		t.Fatalf("details.AddOnFeatures = %#v, want empty slice clear", details.AddOnFeatures)
	}
	if len(details.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map clear", details.FreeformTags)
	}
	if len(details.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map clear", details.DefinedTags)
	}
	if details.InstanceLicenseType != "" {
		t.Fatalf("details.InstanceLicenseType = %q, want enum omitted for empty desired value", details.InstanceLicenseType)
	}
}

func TestOceInstanceCreateOrUpdateReusesMatchingListResultAcrossPages(t *testing.T) {
	t.Parallel()

	resource := newOceInstanceTestResource()
	client := &fakeOceInstanceOCIClient{}
	client.listFn = func(_ context.Context, request ocesdk.ListOceInstancesRequest) (ocesdk.ListOceInstancesResponse, error) {
		switch oceInstanceStringValue(request.Page) {
		case "":
			requireOceInstanceStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireOceInstanceStringPtr(t, "list tenancyId", request.TenancyId, resource.Spec.TenancyId)
			requireOceInstanceStringPtr(t, "list displayName", request.DisplayName, resource.Spec.Name)
			return ocesdk.ListOceInstancesResponse{
				Items: []ocesdk.OceInstanceSummary{
					makeSDKOceInstanceSummary("ocid1.oceinstance.oc1..failed", resource, ocesdk.LifecycleStateFailed),
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return ocesdk.ListOceInstancesResponse{
				Items: []ocesdk.OceInstanceSummary{
					makeSDKOceInstanceSummary("ocid1.oceinstance.oc1..bound", resource, ocesdk.LifecycleStateActive),
				},
			}, nil
		default:
			t.Fatalf("unexpected list page %q", oceInstanceStringValue(request.Page))
			return ocesdk.ListOceInstancesResponse{}, nil
		}
	}
	client.getFn = func(_ context.Context, request ocesdk.GetOceInstanceRequest) (ocesdk.GetOceInstanceResponse, error) {
		requireOceInstanceStringPtr(t, "get oceInstanceId", request.OceInstanceId, "ocid1.oceinstance.oc1..bound")
		return ocesdk.GetOceInstanceResponse{
			OceInstance: makeSDKOceInstance("ocid1.oceinstance.oc1..bound", resource, ocesdk.LifecycleStateActive),
		}, nil
	}

	response, err := newTestOceInstanceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateOceInstance() calls = %d, want 0 after reusable match", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListOceInstances() calls = %d, want 2 across pagination", len(client.listRequests))
	}
	if got := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); got != "ocid1.oceinstance.oc1..bound" {
		t.Fatalf("status.ocid = %q, want reusable OCID", got)
	}
}

func TestOceInstanceCreateOrUpdateTracksWorkRequestAndRecoversID(t *testing.T) {
	t.Parallel()

	resource := newOceInstanceTestResource()
	client := &fakeOceInstanceOCIClient{}
	client.createFn = func(_ context.Context, request ocesdk.CreateOceInstanceRequest) (ocesdk.CreateOceInstanceResponse, error) {
		requireOceInstanceStringPtr(t, "create name", request.CreateOceInstanceDetails.Name, resource.Spec.Name)
		return ocesdk.CreateOceInstanceResponse{
			OpcRequestId:     common.String("opc-create"),
			OpcWorkRequestId: common.String("wr-create"),
		}, nil
	}
	client.workRequestFn = func(_ context.Context, request ocesdk.GetWorkRequestRequest) (ocesdk.GetWorkRequestResponse, error) {
		requireOceInstanceStringPtr(t, "workRequestId", request.WorkRequestId, "wr-create")
		return ocesdk.GetWorkRequestResponse{
			WorkRequest: makeOceInstanceWorkRequest(
				"wr-create",
				ocesdk.WorkRequestOperationTypeCreateOceInstance,
				ocesdk.WorkRequestStatusSucceeded,
				ocesdk.WorkRequestResourceActionTypeCreated,
				"ocid1.oceinstance.oc1..created",
			),
		}, nil
	}
	client.getFn = func(_ context.Context, request ocesdk.GetOceInstanceRequest) (ocesdk.GetOceInstanceResponse, error) {
		requireOceInstanceStringPtr(t, "get oceInstanceId", request.OceInstanceId, "ocid1.oceinstance.oc1..created")
		return ocesdk.GetOceInstanceResponse{
			OceInstance: makeSDKOceInstance("ocid1.oceinstance.oc1..created", resource, ocesdk.LifecycleStateActive),
		}, nil
	}

	response, err := newTestOceInstanceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want settled success", response)
	}
	if got := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); got != "ocid1.oceinstance.oc1..created" {
		t.Fatalf("status.ocid = %q, want recovered create OCID", got)
	}
	if resource.Status.Id != "ocid1.oceinstance.oc1..created" {
		t.Fatalf("status.id = %q, want recovered create ID", resource.Status.Id)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestOceInstanceDeleteTreatsMissingListMatchAsDeleted(t *testing.T) {
	t.Parallel()

	resource := newOceInstanceTestResource()
	client := &fakeOceInstanceOCIClient{
		listFn: func(_ context.Context, request ocesdk.ListOceInstancesRequest) (ocesdk.ListOceInstancesResponse, error) {
			requireOceInstanceStringPtr(t, "delete list displayName", request.DisplayName, resource.Spec.Name)
			requireOceInstanceStringPtr(t, "delete list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireOceInstanceStringPtr(t, "delete list tenancyId", request.TenancyId, resource.Spec.TenancyId)
			return ocesdk.ListOceInstancesResponse{}, nil
		},
	}

	deleted, err := newTestOceInstanceClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after empty list confirmation")
	}
}

func newTestOceInstanceClient(client *fakeOceInstanceOCIClient) OceInstanceServiceClient {
	if client == nil {
		client = &fakeOceInstanceOCIClient{}
	}
	return newOceInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func newOceInstanceTestResource() *ocev1beta1.OceInstance {
	return &ocev1beta1.OceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oce-runtime-test",
			Namespace: "default",
		},
		Spec: ocev1beta1.OceInstanceSpec{
			CompartmentId:          "ocid1.compartment.oc1..oceexample",
			Name:                   "oce-runtime-test",
			TenancyId:              "ocid1.tenancy.oc1..oceexample",
			IdcsAccessToken:        "initial-idcs-token",
			TenancyName:            "oce-tenancy",
			ObjectStorageNamespace: "oce-namespace",
			AdminEmail:             "admin@example.com",
			Description:            "runtime test instance",
			InstanceUsageType:      "PRIMARY",
			AddOnFeatures:          []string{"CONTENT_CAPTURE"},
			UpgradeSchedule:        "UPGRADE_IMMEDIATELY",
			WafPrimaryDomain:       "oce.example.com",
			InstanceAccessType:     "PRIVATE",
			InstanceLicenseType:    "NEW",
			DrRegion:               "us-phoenix-1",
			FreeformTags: map[string]string{
				"environment": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeSDKOceInstance(
	id string,
	resource *ocev1beta1.OceInstance,
	state ocesdk.LifecycleStateEnum,
) ocesdk.OceInstance {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return ocesdk.OceInstance{
		Id:                     common.String(id),
		Guid:                   common.String("guid-1"),
		CompartmentId:          common.String(resource.Spec.CompartmentId),
		Name:                   common.String(resource.Spec.Name),
		TenancyId:              common.String(resource.Spec.TenancyId),
		IdcsTenancy:            common.String("idcs-tenancy"),
		TenancyName:            common.String(resource.Spec.TenancyName),
		ObjectStorageNamespace: common.String(resource.Spec.ObjectStorageNamespace),
		AdminEmail:             common.String(resource.Spec.AdminEmail),
		Description:            common.String(resource.Spec.Description),
		InstanceUsageType:      ocesdk.OceInstanceInstanceUsageTypePrimary,
		AddOnFeatures:          slices.Clone(resource.Spec.AddOnFeatures),
		UpgradeSchedule:        ocesdk.OceInstanceUpgradeScheduleUpgradeImmediately,
		WafPrimaryDomain:       common.String(resource.Spec.WafPrimaryDomain),
		InstanceAccessType:     ocesdk.OceInstanceInstanceAccessTypePrivate,
		InstanceLicenseType:    ocesdk.LicenseTypeNew,
		TimeCreated:            &now,
		TimeUpdated:            &now,
		LifecycleState:         state,
		LifecycleDetails:       "",
		DrRegion:               common.String(resource.Spec.DrRegion),
		StateMessage:           common.String("ready"),
		FreeformTags:           maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:            oceInstanceDefinedTagsFromSpec(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
		Service: map[string]interface{}{
			"IDCS": "value",
		},
	}
}

func makeSDKOceInstanceSummary(
	id string,
	resource *ocev1beta1.OceInstance,
	state ocesdk.LifecycleStateEnum,
) ocesdk.OceInstanceSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return ocesdk.OceInstanceSummary{
		Id:                     common.String(id),
		Guid:                   common.String("guid-1"),
		CompartmentId:          common.String(resource.Spec.CompartmentId),
		Name:                   common.String(resource.Spec.Name),
		TenancyId:              common.String(resource.Spec.TenancyId),
		IdcsTenancy:            common.String("idcs-tenancy"),
		TenancyName:            common.String(resource.Spec.TenancyName),
		ObjectStorageNamespace: common.String(resource.Spec.ObjectStorageNamespace),
		AdminEmail:             common.String(resource.Spec.AdminEmail),
		Description:            common.String(resource.Spec.Description),
		InstanceUsageType:      ocesdk.OceInstanceSummaryInstanceUsageTypePrimary,
		AddOnFeatures:          slices.Clone(resource.Spec.AddOnFeatures),
		UpgradeSchedule:        ocesdk.OceInstanceUpgradeScheduleUpgradeImmediately,
		WafPrimaryDomain:       common.String(resource.Spec.WafPrimaryDomain),
		InstanceAccessType:     ocesdk.OceInstanceSummaryInstanceAccessTypePrivate,
		InstanceLicenseType:    ocesdk.LicenseTypeNew,
		TimeCreated:            &now,
		TimeUpdated:            &now,
		LifecycleState:         state,
		LifecycleDetails:       "",
		DrRegion:               common.String(resource.Spec.DrRegion),
		StateMessage:           common.String("ready"),
		FreeformTags:           maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:            oceInstanceDefinedTagsFromSpec(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeOceInstanceWorkRequest(
	id string,
	operationType ocesdk.WorkRequestOperationTypeEnum,
	status ocesdk.WorkRequestStatusEnum,
	actionType ocesdk.WorkRequestResourceActionTypeEnum,
	resourceID string,
) ocesdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	resources := []ocesdk.WorkRequestResource{}
	if resourceID != "" {
		resources = append(resources, ocesdk.WorkRequestResource{
			EntityType: common.String("OceInstance"),
			ActionType: actionType,
			Identifier: common.String(resourceID),
			EntityUri:  common.String("/oceInstances/" + resourceID),
		})
	}
	return ocesdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..oceexample"),
		Resources:       resources,
		PercentComplete: common.Float32(100),
		TimeAccepted:    &now,
		TimeStarted:     &now,
		TimeFinished:    &now,
	}
}

func requireOceInstanceStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func assertOceInstanceStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
