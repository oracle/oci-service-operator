/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package exadatainsight

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeExadataInsightOCIClient struct {
	createFn         func(context.Context, opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error)
	getFn            func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error)
	listFn           func(context.Context, opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error)
	updateFn         func(context.Context, opsisdk.UpdateExadataInsightRequest) (opsisdk.UpdateExadataInsightResponse, error)
	deleteFn         func(context.Context, opsisdk.DeleteExadataInsightRequest) (opsisdk.DeleteExadataInsightResponse, error)
	getWorkRequestFn func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)

	createRequests         []opsisdk.CreateExadataInsightRequest
	getRequests            []opsisdk.GetExadataInsightRequest
	listRequests           []opsisdk.ListExadataInsightsRequest
	updateRequests         []opsisdk.UpdateExadataInsightRequest
	deleteRequests         []opsisdk.DeleteExadataInsightRequest
	getWorkRequestRequests []opsisdk.GetWorkRequestRequest
}

func (f *fakeExadataInsightOCIClient) CreateExadataInsight(
	ctx context.Context,
	request opsisdk.CreateExadataInsightRequest,
) (opsisdk.CreateExadataInsightResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn == nil {
		return opsisdk.CreateExadataInsightResponse{}, fmt.Errorf("unexpected create exadata insight")
	}
	return f.createFn(ctx, request)
}

func (f *fakeExadataInsightOCIClient) GetExadataInsight(
	ctx context.Context,
	request opsisdk.GetExadataInsightRequest,
) (opsisdk.GetExadataInsightResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn == nil {
		return opsisdk.GetExadataInsightResponse{}, fmt.Errorf("unexpected get exadata insight")
	}
	return f.getFn(ctx, request)
}

func (f *fakeExadataInsightOCIClient) ListExadataInsights(
	ctx context.Context,
	request opsisdk.ListExadataInsightsRequest,
) (opsisdk.ListExadataInsightsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn == nil {
		return opsisdk.ListExadataInsightsResponse{}, nil
	}
	return f.listFn(ctx, request)
}

func (f *fakeExadataInsightOCIClient) UpdateExadataInsight(
	ctx context.Context,
	request opsisdk.UpdateExadataInsightRequest,
) (opsisdk.UpdateExadataInsightResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn == nil {
		return opsisdk.UpdateExadataInsightResponse{}, fmt.Errorf("unexpected update exadata insight")
	}
	return f.updateFn(ctx, request)
}

func (f *fakeExadataInsightOCIClient) DeleteExadataInsight(
	ctx context.Context,
	request opsisdk.DeleteExadataInsightRequest,
) (opsisdk.DeleteExadataInsightResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn == nil {
		return opsisdk.DeleteExadataInsightResponse{}, fmt.Errorf("unexpected delete exadata insight")
	}
	return f.deleteFn(ctx, request)
}

func (f *fakeExadataInsightOCIClient) GetWorkRequest(
	ctx context.Context,
	request opsisdk.GetWorkRequestRequest,
) (opsisdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequestFn == nil {
		return opsisdk.GetWorkRequestResponse{}, fmt.Errorf("unexpected GetWorkRequest")
	}
	return f.getWorkRequestFn(ctx, request)
}

func TestExadataInsightBuildCreateBodyUsesPolymorphicDetailsAndPreservesFalse(t *testing.T) {
	resource := newEMExadataInsight()
	resource.Spec.IsAutoSyncEnabled = false

	details, err := buildExadataInsightCreateDetails(context.Background(), &ExadataInsightServiceManager{}, resource, "default")
	if err != nil {
		t.Fatalf("buildExadataInsightCreateDetails() error = %v", err)
	}
	emDetails, ok := details.(opsisdk.CreateEmManagedExternalExadataInsightDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateEmManagedExternalExadataInsightDetails", details)
	}
	if emDetails.IsAutoSyncEnabled == nil {
		t.Fatal("IsAutoSyncEnabled = nil, want explicit false pointer")
	}
	if *emDetails.IsAutoSyncEnabled {
		t.Fatal("IsAutoSyncEnabled = true, want false")
	}
	if got := stringValue(emDetails.EnterpriseManagerEntityIdentifier); got != "em-entity" {
		t.Fatalf("EnterpriseManagerEntityIdentifier = %q, want em-entity", got)
	}
}

func TestExadataInsightBuildPECreateBodyNormalizesMemberDatabaseDetails(t *testing.T) {
	resource := newPEExadataInsight()

	details, err := buildExadataInsightCreateDetails(context.Background(), &ExadataInsightServiceManager{}, resource, "default")
	if err != nil {
		t.Fatalf("buildExadataInsightCreateDetails() error = %v", err)
	}
	peDetails, ok := details.(opsisdk.CreatePeComanagedExadataInsightDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreatePeComanagedExadataInsightDetails", details)
	}
	if len(peDetails.MemberVmClusterDetails) != 1 {
		t.Fatalf("MemberVmClusterDetails length = %d, want 1", len(peDetails.MemberVmClusterDetails))
	}
	member := peDetails.MemberVmClusterDetails[0]
	if got := stringValue(member.OpsiPrivateEndpointId); got != "ocid.pe.vmcluster" {
		t.Fatalf("memberVmClusterDetails[0].opsiPrivateEndpointId = %q, want ocid.pe.vmcluster", got)
	}
	if got := stringValue(member.DbmPrivateEndpointId); got != "ocid.dbm.vmcluster" {
		t.Fatalf("memberVmClusterDetails[0].dbmPrivateEndpointId = %q, want ocid.dbm.vmcluster", got)
	}
	if len(member.MemberDatabaseDetails) != 1 {
		t.Fatalf("MemberDatabaseDetails length = %d, want 1", len(member.MemberDatabaseDetails))
	}
	assertPEMemberDatabaseDetails(t, member.MemberDatabaseDetails[0])
}

func TestExadataInsightBuildPECreateBodyRejectsMemberDatabaseWithoutEndpoint(t *testing.T) {
	resource := newPEExadataInsight()
	resource.Spec.JsonData = ""

	_, err := buildExadataInsightCreateDetails(context.Background(), &ExadataInsightServiceManager{}, resource, "default")
	if err == nil {
		t.Fatal("buildExadataInsightCreateDetails() error = nil, want missing endpoint rejection")
	}
	if !strings.Contains(err.Error(), "opsiPrivateEndpointId or dbmPrivateEndpointId") {
		t.Fatalf("buildExadataInsightCreateDetails() error = %v, want endpoint rejection", err)
	}
}

func TestExadataInsightBuildPECreateBodySupportsMemberAutonomousDetails(t *testing.T) {
	resource := newPEExadataInsight()
	resource.Spec.JsonData = `{
		"memberVmClusterDetails":[{
			"opsiPrivateEndpointId":"ocid.pe.vmcluster",
			"dbmPrivateEndpointId":"ocid.dbm.vmcluster",
			"memberDatabaseDetails":[{
				"opsiPrivateEndpointId":"ocid.pe.database",
				"dbmPrivateEndpointId":"ocid.dbm.database"
			}],
			"memberAutonomousDetails":[{
				"isAdvancedFeaturesEnabled":true,
				"opsiPrivateEndpointId":"ocid.pe.autonomous"
			}]
		}]
	}`
	resource.Spec.MemberVmClusterDetails[0].MemberAutonomousDetails = []opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetail{
		newExadataInsightMemberAutonomousDetail(),
	}

	details, err := buildExadataInsightCreateDetails(context.Background(), &ExadataInsightServiceManager{}, resource, "default")
	if err != nil {
		t.Fatalf("buildExadataInsightCreateDetails() error = %v", err)
	}
	peDetails, ok := details.(opsisdk.CreatePeComanagedExadataInsightDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreatePeComanagedExadataInsightDetails", details)
	}
	if len(peDetails.MemberVmClusterDetails) != 1 {
		t.Fatalf("MemberVmClusterDetails length = %d, want 1", len(peDetails.MemberVmClusterDetails))
	}
	autonomousDetails := peDetails.MemberVmClusterDetails[0].MemberAutonomousDetails
	if len(autonomousDetails) != 1 {
		t.Fatalf("MemberAutonomousDetails length = %d, want 1", len(autonomousDetails))
	}
	assertPEMemberAutonomousDetails(t, autonomousDetails[0])
}

func TestExadataInsightBuildPECreateBodyRejectsMemberAutonomousWithoutAdvancedOverlay(t *testing.T) {
	resource := newPEExadataInsight()
	resource.Spec.MemberVmClusterDetails[0].MemberAutonomousDetails = []opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetail{
		newExadataInsightMemberAutonomousDetail(),
	}

	_, err := buildExadataInsightCreateDetails(context.Background(), &ExadataInsightServiceManager{}, resource, "default")
	if err == nil {
		t.Fatal("buildExadataInsightCreateDetails() error = nil, want missing advanced-features overlay rejection")
	}
	if !strings.Contains(err.Error(), "memberAutonomousDetails[0].isAdvancedFeaturesEnabled is required") {
		t.Fatalf("buildExadataInsightCreateDetails() error = %v, want missing advanced-features overlay rejection", err)
	}
}

func TestExadataInsightBuildMACSCreateBodySupportsMemberDetails(t *testing.T) {
	resource := newMACSExadataInsight()

	details, err := buildExadataInsightCreateDetails(context.Background(), &ExadataInsightServiceManager{}, resource, "default")
	if err != nil {
		t.Fatalf("buildExadataInsightCreateDetails() error = %v", err)
	}
	macsDetails, ok := details.(opsisdk.CreateMacsManagedCloudExadataInsightDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateMacsManagedCloudExadataInsightDetails", details)
	}
	if len(macsDetails.MemberVmClusterDetails) != 1 {
		t.Fatalf("MemberVmClusterDetails length = %d, want 1", len(macsDetails.MemberVmClusterDetails))
	}
	cluster := macsDetails.MemberVmClusterDetails[0]
	if len(cluster.MemberDatabaseDetails) != 1 {
		t.Fatalf("MemberDatabaseDetails length = %d, want 1", len(cluster.MemberDatabaseDetails))
	}
	if got := cluster.MemberDatabaseDetails[0].DeploymentType; got != opsisdk.CreateMacsManagedCloudDatabaseInsightDetailsDeploymentTypeVirtualMachine {
		t.Fatalf("memberDatabaseDetails[0].deploymentType = %q, want VIRTUAL_MACHINE", got)
	}
	if _, ok := cluster.MemberDatabaseDetails[0].ConnectionCredentialDetails.(opsisdk.CredentialByVault); !ok {
		t.Fatalf("memberDatabaseDetails[0].connectionCredentialDetails type = %T, want CredentialByVault", cluster.MemberDatabaseDetails[0].ConnectionCredentialDetails)
	}
	if len(cluster.MemberAutonomousDetails) != 1 {
		t.Fatalf("MemberAutonomousDetails length = %d, want 1", len(cluster.MemberAutonomousDetails))
	}
	if got := cluster.MemberAutonomousDetails[0].DeploymentType; got != opsisdk.CreateMacsManagedAutonomousDatabaseInsightDetailsDeploymentTypeExacc {
		t.Fatalf("memberAutonomousDetails[0].deploymentType = %q, want EXACC", got)
	}
	if _, ok := cluster.MemberAutonomousDetails[0].ConnectionCredentialDetails.(opsisdk.CredentialByVault); !ok {
		t.Fatalf("memberAutonomousDetails[0].connectionCredentialDetails type = %T, want CredentialByVault", cluster.MemberAutonomousDetails[0].ConnectionCredentialDetails)
	}
}

func TestExadataInsightProjectStatusMapsSDKStatus(t *testing.T) {
	resource := newEMExadataInsight()
	resource.Status.OsokStatus.Message = "preserve"

	err := projectExadataInsightStatus(resource, opsisdk.GetExadataInsightResponse{
		ExadataInsight: newEMInsight("ocid.exadata", opsisdk.ExadataInsightLifecycleStateActive),
	})
	if err != nil {
		t.Fatalf("projectExadataInsightStatus() error = %v", err)
	}
	if got := resource.Status.Status; got != string(opsisdk.ResourceStatusEnabled) {
		t.Fatalf("status.sdkStatus = %q, want ENABLED", got)
	}
	if got := resource.Status.OsokStatus.Message; got != "preserve" {
		t.Fatalf("status.status.message = %q, want preserve", got)
	}
	if got := resource.Status.Id; got != "ocid.exadata" {
		t.Fatalf("status.id = %q, want ocid.exadata", got)
	}
}

func TestExadataInsightServiceClientBindsExistingFromPagedList(t *testing.T) {
	resource := newEMExadataInsight()
	fake := &fakeExadataInsightOCIClient{}
	fake.listFn = pagedExadataInsightListFn()
	fake.getFn = getExadataInsightByIDFn(t, "ocid.exadata")

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, namespacedRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertExadataInsightPagedBind(t, resource, fake, response)
}

func TestExadataInsightServiceClientStartsCreateWorkRequest(t *testing.T) {
	resource := newEMExadataInsight()
	fake := &fakeExadataInsightOCIClient{}
	fake.createFn = func(_ context.Context, request opsisdk.CreateExadataInsightRequest) (opsisdk.CreateExadataInsightResponse, error) {
		if _, ok := request.CreateExadataInsightDetails.(opsisdk.CreateEmManagedExternalExadataInsightDetails); !ok {
			t.Fatalf("create body type = %T, want EM create details", request.CreateExadataInsightDetails)
		}
		return opsisdk.CreateExadataInsightResponse{
			ExadataInsight:   newEMInsight("ocid.exadata", opsisdk.ExadataInsightLifecycleStateCreating),
			OpcRequestId:     common.String("opc-create"),
			OpcWorkRequestId: common.String("wr-create"),
		}, nil
	}
	fake.getWorkRequestFn = workRequestFn(opsisdk.OperationTypeCreateExadataInsight, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeCreated, "wr-create", "ocid.exadata")

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, namespacedRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateExadataInsight called %d times, want 1", len(fake.createRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.WorkRequestID != "wr-create" || current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("async.current = %#v, want create work request wr-create", current)
	}
}

func TestExadataInsightServiceClientUpdatesMutableFields(t *testing.T) {
	resource := newEMExadataInsight()
	setTrackedExadataInsight(resource, "ocid.exadata")
	if err := recordExadataInsightCreateOnlyFingerprint(resource); err != nil {
		t.Fatalf("record fingerprint: %v", err)
	}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ops": {"tier": "gold"}}
	resource.Spec.IsAutoSyncEnabled = false

	fake := &fakeExadataInsightOCIClient{}
	fake.getFn = func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		return opsisdk.GetExadataInsightResponse{
			ExadataInsight: newEMInsightWithTags("ocid.exadata", map[string]string{"env": "test"}, map[string]map[string]interface{}{"ops": {"tier": "silver"}}, true),
		}, nil
	}
	fake.updateFn = func(_ context.Context, request opsisdk.UpdateExadataInsightRequest) (opsisdk.UpdateExadataInsightResponse, error) {
		update, ok := request.UpdateExadataInsightDetails.(opsisdk.UpdateEmManagedExternalExadataInsightDetails)
		if !ok {
			t.Fatalf("update body type = %T, want EM update details", request.UpdateExadataInsightDetails)
		}
		if got := update.FreeformTags["env"]; got != "prod" {
			t.Fatalf("update freeform env = %q, want prod", got)
		}
		if update.IsAutoSyncEnabled == nil || *update.IsAutoSyncEnabled {
			t.Fatalf("update IsAutoSyncEnabled = %#v, want explicit false", update.IsAutoSyncEnabled)
		}
		return opsisdk.UpdateExadataInsightResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	fake.getWorkRequestFn = workRequestFn(opsisdk.OperationTypeUpdateExadataInsight, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeUpdated, "wr-update", "ocid.exadata")

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, namespacedRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateExadataInsight called %d times, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestExadataInsightServiceClientClearsExplicitEmptyFreeformTags(t *testing.T) {
	resource := newTrackedEMExadataInsight(t)
	resource.Spec.FreeformTags = map[string]string{}

	fake := &fakeExadataInsightOCIClient{}
	fake.getFn = getExadataInsightByIDFn(t, "ocid.exadata")
	fake.updateFn = func(_ context.Context, request opsisdk.UpdateExadataInsightRequest) (opsisdk.UpdateExadataInsightResponse, error) {
		update, ok := request.UpdateExadataInsightDetails.(opsisdk.UpdateEmManagedExternalExadataInsightDetails)
		if !ok {
			t.Fatalf("update body type = %T, want EM update details", request.UpdateExadataInsightDetails)
		}
		if update.FreeformTags == nil {
			t.Fatal("update freeformTags = nil, want explicit empty map")
		}
		if len(update.FreeformTags) != 0 {
			t.Fatalf("update freeformTags length = %d, want 0", len(update.FreeformTags))
		}
		if update.DefinedTags != nil {
			t.Fatalf("update definedTags = %#v, want nil", update.DefinedTags)
		}
		return opsisdk.UpdateExadataInsightResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	fake.getWorkRequestFn = workRequestFn(opsisdk.OperationTypeUpdateExadataInsight, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeUpdated, "wr-update", "ocid.exadata")

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, namespacedRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateExadataInsight called %d times, want 1", len(fake.updateRequests))
	}
}

func TestExadataInsightServiceClientClearsExplicitEmptyDefinedTags(t *testing.T) {
	resource := newTrackedEMExadataInsight(t)
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	fake := &fakeExadataInsightOCIClient{}
	fake.getFn = getExadataInsightByIDFn(t, "ocid.exadata")
	fake.updateFn = func(_ context.Context, request opsisdk.UpdateExadataInsightRequest) (opsisdk.UpdateExadataInsightResponse, error) {
		update, ok := request.UpdateExadataInsightDetails.(opsisdk.UpdateEmManagedExternalExadataInsightDetails)
		if !ok {
			t.Fatalf("update body type = %T, want EM update details", request.UpdateExadataInsightDetails)
		}
		if update.FreeformTags != nil {
			t.Fatalf("update freeformTags = %#v, want nil", update.FreeformTags)
		}
		if update.DefinedTags == nil {
			t.Fatal("update definedTags = nil, want explicit empty map")
		}
		if len(update.DefinedTags) != 0 {
			t.Fatalf("update definedTags length = %d, want 0", len(update.DefinedTags))
		}
		return opsisdk.UpdateExadataInsightResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	fake.getWorkRequestFn = workRequestFn(opsisdk.OperationTypeUpdateExadataInsight, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeUpdated, "wr-update", "ocid.exadata")

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, namespacedRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateExadataInsight called %d times, want 1", len(fake.updateRequests))
	}
}

func TestExadataInsightServiceClientNoopObserveDoesNotUpdate(t *testing.T) {
	resource := newEMExadataInsight()
	setTrackedExadataInsight(resource, "ocid.exadata")
	if err := recordExadataInsightCreateOnlyFingerprint(resource); err != nil {
		t.Fatalf("record fingerprint: %v", err)
	}

	fake := &fakeExadataInsightOCIClient{}
	fake.getFn = func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		return opsisdk.GetExadataInsightResponse{
			ExadataInsight: newEMInsight("ocid.exadata", opsisdk.ExadataInsightLifecycleStateActive),
		}, nil
	}

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, namespacedRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateExadataInsight called %d times, want 0", len(fake.updateRequests))
	}
}

func TestExadataInsightServiceClientRejectsUnexposedCreateOnlyDrift(t *testing.T) {
	resource := newEMExadataInsight()
	setTrackedExadataInsight(resource, "ocid.exadata")
	if err := recordExadataInsightCreateOnlyFingerprint(resource); err != nil {
		t.Fatalf("record fingerprint: %v", err)
	}
	resource.Spec.MemberEntityDetails = []opsiv1beta1.ExadataInsightMemberEntityDetail{{
		EnterpriseManagerEntityIdentifier: "member-1",
		CompartmentId:                     "member-compartment",
	}}

	fake := &fakeExadataInsightOCIClient{}
	fake.getFn = func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		return opsisdk.GetExadataInsightResponse{
			ExadataInsight: newEMInsight("ocid.exadata", opsisdk.ExadataInsightLifecycleStateActive),
		}, nil
	}

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	_, err := client.CreateOrUpdate(context.Background(), resource, namespacedRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "create-only fields change") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only fields change", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateExadataInsight called %d times, want 0", len(fake.updateRequests))
	}
}

func TestExadataInsightDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := newEMExadataInsight()
	setTrackedExadataInsight(resource, "ocid.exadata")
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeExadataInsightOCIClient{
		getFn: func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
			return opsisdk.GetExadataInsightResponse{}, authErr
		},
	}

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirmation context", err)
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteExadataInsight called %d times, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestExadataInsightDeletePendingWorkRequestSkipsAuthShapedPrecheck(t *testing.T) {
	resource := newEMExadataInsight()
	setTrackedExadataInsight(resource, "ocid.exadata")
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}

	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeExadataInsightOCIClient{}
	fake.getFn = func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		return opsisdk.GetExadataInsightResponse{}, authErr
	}
	fake.getWorkRequestFn = workRequestFn(opsisdk.OperationTypeDeleteExadataInsight, opsisdk.OperationStatusInProgress, opsisdk.ActionTypeDeleted, "wr-delete", "ocid.exadata")

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if len(fake.getWorkRequestRequests) != 1 {
		t.Fatalf("GetWorkRequest called %d times, want 1", len(fake.getWorkRequestRequests))
	}
	if len(fake.getRequests) != 0 {
		t.Fatalf("GetExadataInsight called %d times, want 0 before pending work request observation finishes", len(fake.getRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.WorkRequestID != "wr-delete" || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want pending delete work request wr-delete", current)
	}
}

func TestExadataInsightDeleteWorkRequestSucceededAuthShapedReadStaysFatal(t *testing.T) {
	resource := newEMExadataInsight()
	setTrackedExadataInsight(resource, "ocid.exadata")
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}

	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeExadataInsightOCIClient{}
	fake.getFn = func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		return opsisdk.GetExadataInsightResponse{}, authErr
	}
	fake.getWorkRequestFn = workRequestFn(opsisdk.OperationTypeDeleteExadataInsight, opsisdk.OperationStatusSucceeded, opsisdk.ActionTypeDeleted, "wr-delete", "ocid.exadata")

	client := newExadataInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous read after work request")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteExadataInsight called %d times, want 0", len(fake.deleteRequests))
	}
	if len(fake.getWorkRequestRequests) != 1 {
		t.Fatalf("GetWorkRequest called %d times, want 1", len(fake.getWorkRequestRequests))
	}
}

func newEMExadataInsight() *opsiv1beta1.ExadataInsight {
	return &opsiv1beta1.ExadataInsight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exadata",
			Namespace: "default",
		},
		Spec: opsiv1beta1.ExadataInsightSpec{
			CompartmentId:                     "compartment",
			EntitySource:                      string(opsisdk.ExadataEntitySourceEmManagedExternalExadata),
			EnterpriseManagerIdentifier:       "em",
			EnterpriseManagerBridgeId:         "bridge",
			EnterpriseManagerEntityIdentifier: "em-entity",
			FreeformTags:                      map[string]string{"env": "test"},
			DefinedTags:                       map[string]shared.MapValue{"ops": {"tier": "silver"}},
		},
	}
}

func newTrackedEMExadataInsight(t *testing.T) *opsiv1beta1.ExadataInsight {
	t.Helper()
	resource := newEMExadataInsight()
	setTrackedExadataInsight(resource, "ocid.exadata")
	if err := recordExadataInsightCreateOnlyFingerprint(resource); err != nil {
		t.Fatalf("record fingerprint: %v", err)
	}
	return resource
}

func newPEExadataInsight() *opsiv1beta1.ExadataInsight {
	return &opsiv1beta1.ExadataInsight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pe-exadata",
			Namespace: "default",
		},
		Spec: opsiv1beta1.ExadataInsightSpec{
			CompartmentId:  "compartment",
			EntitySource:   string(opsisdk.ExadataEntitySourcePeComanagedExadata),
			ExadataInfraId: "ocid.exadata.infra",
			JsonData: `{
				"memberVmClusterDetails":[{
					"opsiPrivateEndpointId":"ocid.pe.vmcluster",
					"dbmPrivateEndpointId":"ocid.dbm.vmcluster",
					"memberDatabaseDetails":[{
						"opsiPrivateEndpointId":"ocid.pe.database",
						"dbmPrivateEndpointId":"ocid.dbm.database"
					}]
				}]
			}`,
			MemberVmClusterDetails: []opsiv1beta1.ExadataInsightMemberVmClusterDetail{{
				VmclusterId:   "ocid.vmcluster",
				CompartmentId: "vmcluster-compartment",
				MemberDatabaseDetails: []opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetail{{
					CompartmentId:        "database-compartment",
					DatabaseId:           "ocid.database",
					ManagementAgentId:    "ocid.managementagent",
					DatabaseResourceType: "DATABASE",
					ConnectionDetails: opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionDetails{
						HostName:    "10.0.0.12",
						Protocol:    string(opsisdk.PeComanagedDatabaseConnectionDetailsProtocolTcp),
						Port:        1521,
						ServiceName: "pdb.example",
					},
					ConnectionCredentialDetails: opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberDatabaseDetailConnectionCredentialDetails{
						CredentialType:   string(opsisdk.CredentialDetailsCredentialTypeVault),
						UserName:         "opsi_user",
						PasswordSecretId: "ocid.passwordsecret",
						WalletSecretId:   "ocid.walletsecret",
						Role:             string(opsisdk.CredentialByVaultRoleNormal),
					},
					FreeformTags:   map[string]string{"member": "database"},
					DefinedTags:    map[string]shared.MapValue{"ops": {"tier": "gold"}},
					SystemTags:     map[string]shared.MapValue{"orcl-cloud": {"free-tier-retained": "true"}},
					DeploymentType: string(opsisdk.CreatePeComanagedDatabaseInsightDetailsDeploymentTypeVirtualMachine),
				}},
			}},
		},
	}
}

func newMACSExadataInsight() *opsiv1beta1.ExadataInsight {
	resource := newPEExadataInsight()
	resource.Name = "macs-exadata"
	resource.Spec.EntitySource = string(opsisdk.ExadataEntitySourceMacsManagedCloudExadata)
	resource.Spec.JsonData = ""
	resource.Spec.MemberVmClusterDetails[0].MemberDatabaseDetails[0].DeploymentType = string(opsisdk.CreateMacsManagedCloudDatabaseInsightDetailsDeploymentTypeVirtualMachine)
	resource.Spec.MemberVmClusterDetails[0].MemberAutonomousDetails = []opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetail{
		newExadataInsightMemberAutonomousDetail(),
	}
	resource.Spec.MemberVmClusterDetails[0].MemberAutonomousDetails[0].DeploymentType = string(opsisdk.CreateMacsManagedAutonomousDatabaseInsightDetailsDeploymentTypeExacc)
	return resource
}

func newExadataInsightMemberAutonomousDetail() opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetail {
	return opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetail{
		CompartmentId:        "autonomous-compartment",
		DatabaseId:           "ocid.autonomousdatabase",
		ManagementAgentId:    "ocid.managementagent",
		DatabaseResourceType: "AUTONOMOUS_DATABASE",
		ConnectionDetails: opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetailConnectionDetails{
			HostName:    "adb.example",
			Protocol:    string(opsisdk.ConnectionDetailsProtocolTcps),
			Port:        1522,
			ServiceName: "adb_low",
		},
		ConnectionCredentialDetails: opsiv1beta1.ExadataInsightMemberVmClusterDetailMemberAutonomousDetailConnectionCredentialDetails{
			CredentialType:   string(opsisdk.CredentialDetailsCredentialTypeVault),
			UserName:         "adb_user",
			PasswordSecretId: "ocid.autonomous.passwordsecret",
			WalletSecretId:   "ocid.autonomous.walletsecret",
			Role:             string(opsisdk.CredentialByVaultRoleNormal),
		},
		FreeformTags:   map[string]string{"member": "autonomous"},
		DefinedTags:    map[string]shared.MapValue{"ops": {"tier": "platinum"}},
		SystemTags:     map[string]shared.MapValue{"orcl-cloud": {"free-tier-retained": "false"}},
		DeploymentType: "AUTONOMOUS",
	}
}

func assertPEMemberDatabaseDetails(t *testing.T, database opsisdk.CreatePeComanagedDatabaseInsightDetails) {
	t.Helper()
	if got := stringValue(database.ServiceName); got != "pdb.example" {
		t.Fatalf("memberDatabaseDetails[0].serviceName = %q, want pdb.example", got)
	}
	if got := stringValue(database.OpsiPrivateEndpointId); got != "ocid.pe.database" {
		t.Fatalf("memberDatabaseDetails[0].opsiPrivateEndpointId = %q, want ocid.pe.database", got)
	}
	if got := stringValue(database.DbmPrivateEndpointId); got != "ocid.dbm.database" {
		t.Fatalf("memberDatabaseDetails[0].dbmPrivateEndpointId = %q, want ocid.dbm.database", got)
	}
	assertPEMemberDatabaseConnectionDetails(t, database.ConnectionDetails)
	assertPEMemberDatabaseCredentialDetails(t, database.CredentialDetails)
}

func assertPEMemberAutonomousDetails(t *testing.T, autonomous opsisdk.CreateAutonomousDatabaseInsightDetails) {
	t.Helper()
	if got := stringValue(autonomous.DatabaseId); got != "ocid.autonomousdatabase" {
		t.Fatalf("memberAutonomousDetails[0].databaseId = %q, want ocid.autonomousdatabase", got)
	}
	if autonomous.IsAdvancedFeaturesEnabled == nil || !*autonomous.IsAdvancedFeaturesEnabled {
		t.Fatalf("memberAutonomousDetails[0].isAdvancedFeaturesEnabled = %#v, want true", autonomous.IsAdvancedFeaturesEnabled)
	}
	if got := stringValue(autonomous.OpsiPrivateEndpointId); got != "ocid.pe.autonomous" {
		t.Fatalf("memberAutonomousDetails[0].opsiPrivateEndpointId = %q, want ocid.pe.autonomous", got)
	}
	assertPEMemberAutonomousConnectionDetails(t, autonomous.ConnectionDetails)
	assertPEMemberAutonomousCredentialDetails(t, autonomous.CredentialDetails)
}

func assertPEMemberAutonomousConnectionDetails(t *testing.T, connection *opsisdk.ConnectionDetails) {
	t.Helper()
	if connection == nil {
		t.Fatal("memberAutonomousDetails[0].connectionDetails = nil")
	}
	if got := connection.Protocol; got != opsisdk.ConnectionDetailsProtocolTcps {
		t.Fatalf("connectionDetails.protocol = %q, want TCPS", got)
	}
	if got := stringValue(connection.HostName); got != "adb.example" {
		t.Fatalf("connectionDetails.hostName = %q, want adb.example", got)
	}
	if got := stringValue(connection.ServiceName); got != "adb_low" {
		t.Fatalf("connectionDetails.serviceName = %q, want adb_low", got)
	}
	if connection.Port == nil || *connection.Port != 1522 {
		t.Fatalf("connectionDetails.port = %v, want 1522", connection.Port)
	}
}

func assertPEMemberAutonomousCredentialDetails(t *testing.T, credentials opsisdk.CredentialDetails) {
	t.Helper()
	vault, ok := credentials.(opsisdk.CredentialByVault)
	if !ok {
		t.Fatalf("memberAutonomousDetails[0].credentialDetails type = %T, want CredentialByVault", credentials)
	}
	if got := stringValue(vault.UserName); got != "adb_user" {
		t.Fatalf("credentialDetails.userName = %q, want adb_user", got)
	}
	if got := stringValue(vault.PasswordSecretId); got != "ocid.autonomous.passwordsecret" {
		t.Fatalf("credentialDetails.passwordSecretId = %q, want ocid.autonomous.passwordsecret", got)
	}
	if got := stringValue(vault.WalletSecretId); got != "ocid.autonomous.walletsecret" {
		t.Fatalf("credentialDetails.walletSecretId = %q, want ocid.autonomous.walletsecret", got)
	}
	if got := vault.Role; got != opsisdk.CredentialByVaultRoleNormal {
		t.Fatalf("credentialDetails.role = %q, want NORMAL", got)
	}
}

func assertPEMemberDatabaseConnectionDetails(
	t *testing.T,
	connection *opsisdk.PeComanagedDatabaseConnectionDetails,
) {
	t.Helper()
	if connection == nil {
		t.Fatal("memberDatabaseDetails[0].connectionDetails = nil")
	}
	if got := connection.Protocol; got != opsisdk.PeComanagedDatabaseConnectionDetailsProtocolTcp {
		t.Fatalf("connectionDetails.protocol = %q, want TCP", got)
	}
	if got := stringValue(connection.ServiceName); got != "pdb.example" {
		t.Fatalf("connectionDetails.serviceName = %q, want pdb.example", got)
	}
	if len(connection.Hosts) != 1 {
		t.Fatalf("connectionDetails.hosts length = %d, want 1", len(connection.Hosts))
	}
	host := connection.Hosts[0]
	if got := stringValue(host.HostIp); got != "10.0.0.12" {
		t.Fatalf("connectionDetails.hosts[0].hostIp = %q, want 10.0.0.12", got)
	}
	if host.Port == nil || *host.Port != 1521 {
		t.Fatalf("connectionDetails.hosts[0].port = %v, want 1521", host.Port)
	}
}

func assertPEMemberDatabaseCredentialDetails(t *testing.T, credentials opsisdk.CredentialDetails) {
	t.Helper()
	vault, ok := credentials.(opsisdk.CredentialByVault)
	if !ok {
		t.Fatalf("memberDatabaseDetails[0].credentialDetails type = %T, want CredentialByVault", credentials)
	}
	if got := stringValue(vault.UserName); got != "opsi_user" {
		t.Fatalf("credentialDetails.userName = %q, want opsi_user", got)
	}
	if got := stringValue(vault.PasswordSecretId); got != "ocid.passwordsecret" {
		t.Fatalf("credentialDetails.passwordSecretId = %q, want ocid.passwordsecret", got)
	}
	if got := stringValue(vault.WalletSecretId); got != "ocid.walletsecret" {
		t.Fatalf("credentialDetails.walletSecretId = %q, want ocid.walletsecret", got)
	}
	if got := vault.Role; got != opsisdk.CredentialByVaultRoleNormal {
		t.Fatalf("credentialDetails.role = %q, want NORMAL", got)
	}
}

func setTrackedExadataInsight(resource *opsiv1beta1.ExadataInsight, id string) {
	now := metav1.Now()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.OsokStatus.CreatedAt = &now
}

func namespacedRequest(resource *opsiv1beta1.ExadataInsight) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func pagedExadataInsightListFn() func(context.Context, opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error) {
	return func(_ context.Context, request opsisdk.ListExadataInsightsRequest) (opsisdk.ListExadataInsightsResponse, error) {
		if request.Page != nil {
			return exadataInsightListPage(newEMInsightSummary("ocid.exadata", "em-entity"), nil), nil
		}
		return exadataInsightListPage(newEMInsightSummary("ocid.other", "other-entity"), common.String("page-2")), nil
	}
}

func exadataInsightListPage(
	item opsisdk.ExadataInsightSummary,
	nextPage *string,
) opsisdk.ListExadataInsightsResponse {
	return opsisdk.ListExadataInsightsResponse{
		ExadataInsightSummaryCollection: opsisdk.ExadataInsightSummaryCollection{
			Items: []opsisdk.ExadataInsightSummary{item},
		},
		OpcNextPage: nextPage,
	}
}

func getExadataInsightByIDFn(
	t *testing.T,
	wantID string,
) func(context.Context, opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
	t.Helper()
	return func(_ context.Context, request opsisdk.GetExadataInsightRequest) (opsisdk.GetExadataInsightResponse, error) {
		if got := stringValue(request.ExadataInsightId); got != wantID {
			t.Fatalf("GetExadataInsight id = %q, want %s", got, wantID)
		}
		return opsisdk.GetExadataInsightResponse{
			ExadataInsight: newEMInsight(wantID, opsisdk.ExadataInsightLifecycleStateActive),
		}, nil
	}
}

func assertExadataInsightPagedBind(
	t *testing.T,
	resource *opsiv1beta1.ExadataInsight,
	fake *fakeExadataInsightOCIClient,
	response servicemanager.OSOKResponse,
) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = false")
	}
	assertExadataInsightBindCalls(t, fake)
	assertTrackedExadataInsightID(t, resource, "ocid.exadata")
	if _, ok := exadataInsightRecordedCreateOnlyFingerprint(resource); !ok {
		t.Fatal("create-only fingerprint was not recorded after bind")
	}
}

func assertExadataInsightBindCalls(t *testing.T, fake *fakeExadataInsightOCIClient) {
	t.Helper()
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateExadataInsight called %d times, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListExadataInsights called %d times, want 2", len(fake.listRequests))
	}
	if got := stringValue(fake.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
}

func assertTrackedExadataInsightID(t *testing.T, resource *opsiv1beta1.ExadataInsight, wantID string) {
	t.Helper()
	if got := resource.Status.Id; got != wantID {
		t.Fatalf("status.id = %q, want %s", got, wantID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %s", got, wantID)
	}
}

func newEMInsight(id string, lifecycle opsisdk.ExadataInsightLifecycleStateEnum) opsisdk.EmManagedExternalExadataInsight {
	insight := newEMInsightWithTags(id, map[string]string{"env": "test"}, map[string]map[string]interface{}{"ops": {"tier": "silver"}}, false)
	insight.LifecycleState = lifecycle
	return insight
}

func newEMInsightWithTags(
	id string,
	freeformTags map[string]string,
	definedTags map[string]map[string]interface{},
	isAutoSyncEnabled bool,
) opsisdk.EmManagedExternalExadataInsight {
	return opsisdk.EmManagedExternalExadataInsight{
		Id:                                common.String(id),
		CompartmentId:                     common.String("compartment"),
		ExadataName:                       common.String("em-exadata"),
		FreeformTags:                      freeformTags,
		DefinedTags:                       definedTags,
		EnterpriseManagerIdentifier:       common.String("em"),
		EnterpriseManagerBridgeId:         common.String("bridge"),
		EnterpriseManagerEntityIdentifier: common.String("em-entity"),
		Status:                            opsisdk.ResourceStatusEnabled,
		LifecycleState:                    opsisdk.ExadataInsightLifecycleStateActive,
		IsAutoSyncEnabled:                 common.Bool(isAutoSyncEnabled),
	}
}

func newEMInsightSummary(id string, entityIdentifier string) opsisdk.EmManagedExternalExadataInsightSummary {
	return opsisdk.EmManagedExternalExadataInsightSummary{
		Id:                                common.String(id),
		CompartmentId:                     common.String("compartment"),
		ExadataName:                       common.String("em-exadata"),
		FreeformTags:                      map[string]string{"env": "test"},
		DefinedTags:                       map[string]map[string]interface{}{"ops": {"tier": "silver"}},
		EnterpriseManagerIdentifier:       common.String("em"),
		EnterpriseManagerBridgeId:         common.String("bridge"),
		EnterpriseManagerEntityIdentifier: common.String(entityIdentifier),
		Status:                            opsisdk.ResourceStatusEnabled,
		LifecycleState:                    opsisdk.ExadataInsightLifecycleStateActive,
	}
}

func workRequestFn(
	operationType opsisdk.OperationTypeEnum,
	status opsisdk.OperationStatusEnum,
	action opsisdk.ActionTypeEnum,
	workRequestID string,
	resourceID string,
) func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	return func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		if got := stringValue(request.WorkRequestId); got != workRequestID {
			return opsisdk.GetWorkRequestResponse{}, fmt.Errorf("work request id = %q, want %s", got, workRequestID)
		}
		return opsisdk.GetWorkRequestResponse{
			WorkRequest: opsisdk.WorkRequest{
				Id:            common.String(workRequestID),
				OperationType: operationType,
				Status:        status,
				Resources: []opsisdk.WorkRequestResource{{
					ActionType: action,
					EntityType: common.String("exadataInsight"),
					Identifier: common.String(resourceID),
					EntityUri:  common.String("/exadataInsights/" + resourceID),
				}},
			},
		}, nil
	}
}
