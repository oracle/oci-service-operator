/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package databaseinsight

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDatabaseInsightID            = "ocid1.databaseinsight.oc1..insight"
	testDatabaseInsightCompartmentID = "ocid1.compartment.oc1..test"
	testDatabaseInsightDatabaseID    = "ocid1.autonomousdatabase.oc1..db"
	testDatabaseInsightName          = "database-insight-sample"
	testDatabaseInsightPrivateEPID   = "ocid1.opsiprivateendpoint.oc1..test"
)

type fakeDatabaseInsightOCIClient struct {
	createFn func(context.Context, opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error)
	getFn    func(context.Context, opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error)
	listFn   func(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error)
	updateFn func(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error)
	deleteFn func(context.Context, opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error)
}

func (f *fakeDatabaseInsightOCIClient) CreateDatabaseInsight(ctx context.Context, req opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return opsisdk.CreateDatabaseInsightResponse{}, nil
}

func (f *fakeDatabaseInsightOCIClient) GetDatabaseInsight(ctx context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return opsisdk.GetDatabaseInsightResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "database insight is missing")
}

func (f *fakeDatabaseInsightOCIClient) ListDatabaseInsights(ctx context.Context, req opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return opsisdk.ListDatabaseInsightsResponse{}, nil
}

func (f *fakeDatabaseInsightOCIClient) UpdateDatabaseInsight(ctx context.Context, req opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return opsisdk.UpdateDatabaseInsightResponse{}, nil
}

func (f *fakeDatabaseInsightOCIClient) DeleteDatabaseInsight(ctx context.Context, req opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return opsisdk.DeleteDatabaseInsightResponse{}, nil
}

func newTestDatabaseInsightClient(client databaseInsightOCIClient) DatabaseInsightServiceClient {
	return newDatabaseInsightServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDatabaseInsightResource() *opsiv1beta1.DatabaseInsight {
	return &opsiv1beta1.DatabaseInsight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDatabaseInsightName,
			Namespace: "default",
		},
		Spec: opsiv1beta1.DatabaseInsightSpec{
			CompartmentId:        testDatabaseInsightCompartmentID,
			EntitySource:         "AUTONOMOUS_DATABASE",
			DatabaseId:           testDatabaseInsightDatabaseID,
			DatabaseResourceType: "AUTONOMOUS_DATABASE",
			FreeformTags:         map[string]string{"env": "test"},
			DefinedTags:          map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedDatabaseInsightResource() *opsiv1beta1.DatabaseInsight {
	resource := makeDatabaseInsightResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDatabaseInsightID)
	resource.Status.Id = testDatabaseInsightID
	resource.Status.CompartmentId = testDatabaseInsightCompartmentID
	resource.Status.EntitySource = resource.Spec.EntitySource
	resource.Status.DatabaseId = testDatabaseInsightDatabaseID
	resource.Status.DatabaseResourceType = resource.Spec.DatabaseResourceType
	resource.Status.Status = string(opsisdk.ResourceStatusEnabled)
	resource.Status.LifecycleState = string(opsisdk.LifecycleStateActive)
	return resource
}

func makePeComanagedDatabaseInsightResource() *opsiv1beta1.DatabaseInsight {
	resource := makeDatabaseInsightResource()
	resource.Spec.EntitySource = string(databaseInsightEntitySourcePeComanaged)
	resource.Spec.DatabaseResourceType = "EXTERNAL_PDB"
	resource.Spec.DeploymentType = string(opsisdk.CreatePeComanagedDatabaseInsightDetailsDeploymentTypeVirtualMachine)
	resource.Spec.ServiceName = "sales_pdb"
	resource.Spec.OpsiPrivateEndpointId = testDatabaseInsightPrivateEPID
	resource.Spec.ConnectionDetails = opsiv1beta1.DatabaseInsightConnectionDetails{
		HostName:    "10.0.0.42",
		Protocol:    string(opsisdk.PeComanagedDatabaseConnectionDetailsProtocolTcps),
		Port:        1522,
		ServiceName: "sales_pdb",
	}
	resource.Spec.CredentialDetails = opsiv1beta1.DatabaseInsightCredentialDetails{
		CredentialType:   string(opsisdk.CredentialDetailsCredentialTypeVault),
		UserName:         "admin",
		PasswordSecretId: "ocid1.vaultsecret.oc1..password",
		Role:             string(opsisdk.CredentialByVaultRoleNormal),
	}
	return resource
}

func makeDatabaseInsightRequest(resource *opsiv1beta1.DatabaseInsight) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKDatabaseInsight(
	id string,
	spec opsiv1beta1.DatabaseInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.DatabaseInsight {
	return opsisdk.AutonomousDatabaseInsight{
		Id:                        common.String(id),
		CompartmentId:             common.String(spec.CompartmentId),
		FreeformTags:              cloneDatabaseInsightStringMap(spec.FreeformTags),
		DefinedTags:               databaseInsightDefinedTags(spec.DefinedTags),
		DatabaseId:                common.String(spec.DatabaseId),
		DatabaseName:              common.String("test-db"),
		DatabaseResourceType:      common.String(spec.DatabaseResourceType),
		IsAdvancedFeaturesEnabled: common.Bool(spec.IsAdvancedFeaturesEnabled),
		Status:                    opsisdk.ResourceStatusEnabled,
		LifecycleState:            state,
	}
}

func makeSDKPeComanagedDatabaseInsight(
	id string,
	spec opsiv1beta1.DatabaseInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.DatabaseInsight {
	return opsisdk.PeComanagedDatabaseInsight{
		Id:                    common.String(id),
		CompartmentId:         common.String(spec.CompartmentId),
		FreeformTags:          cloneDatabaseInsightStringMap(spec.FreeformTags),
		DefinedTags:           databaseInsightDefinedTags(spec.DefinedTags),
		DatabaseId:            common.String(spec.DatabaseId),
		DatabaseName:          common.String("test-db"),
		DatabaseResourceType:  common.String(spec.DatabaseResourceType),
		OpsiPrivateEndpointId: common.String(spec.OpsiPrivateEndpointId),
		Status:                opsisdk.ResourceStatusEnabled,
		LifecycleState:        state,
	}
}

func makeSDKDatabaseInsightSummary(
	id string,
	spec opsiv1beta1.DatabaseInsightSpec,
	state opsisdk.LifecycleStateEnum,
) opsisdk.DatabaseInsightSummary {
	return opsisdk.AutonomousDatabaseInsightSummary{
		Id:                        common.String(id),
		CompartmentId:             common.String(spec.CompartmentId),
		DatabaseId:                common.String(spec.DatabaseId),
		DatabaseName:              common.String("test-db"),
		DatabaseResourceType:      common.String(spec.DatabaseResourceType),
		IsAdvancedFeaturesEnabled: common.Bool(spec.IsAdvancedFeaturesEnabled),
		FreeformTags:              cloneDatabaseInsightStringMap(spec.FreeformTags),
		DefinedTags:               databaseInsightDefinedTags(spec.DefinedTags),
		Status:                    opsisdk.ResourceStatusEnabled,
		LifecycleState:            state,
	}
}

func TestDatabaseInsightCreateOrUpdateBindsExistingByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeDatabaseInsightResource()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		listFn: databaseInsightPagedListFn(t, resource, &listCalls),
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
			createCalled = true
			return opsisdk.CreateDatabaseInsightResponse{}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
			updateCalled = true
			return opsisdk.UpdateDatabaseInsightResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateDatabaseInsight() called for existing insight")
	}
	if updateCalled {
		t.Fatal("UpdateDatabaseInsight() called for matching insight")
	}
	if listCalls != 2 {
		t.Fatalf("ListDatabaseInsights() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDatabaseInsight() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDatabaseInsightID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDatabaseInsightID)
	}
	if got := resource.Status.Status; got != string(opsisdk.ResourceStatusEnabled) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, opsisdk.ResourceStatusEnabled)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDatabaseInsightCreateRecordsPolymorphicPayloadRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeDatabaseInsightResource()
	listCalls := 0
	createCalls := 0

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		listFn: func(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
			listCalls++
			return opsisdk.ListDatabaseInsightsResponse{}, nil
		},
		createFn: func(_ context.Context, req opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
			createCalls++
			requireDatabaseInsightCreateRequest(t, req, resource)
			return opsisdk.CreateDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
				OpcRequestId:    common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 {
		t.Fatalf("ListDatabaseInsights() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDatabaseInsight() calls = %d, want 1", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDatabaseInsightID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDatabaseInsightID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if got := resource.Status.Status; got != string(opsisdk.ResourceStatusEnabled) {
		t.Fatalf("status.sdkStatus = %q, want %q", got, opsisdk.ResourceStatusEnabled)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDatabaseInsightCreateShapesPeComanagedTypedConnectionDetails(t *testing.T) {
	t.Parallel()

	resource := makePeComanagedDatabaseInsightResource()
	createCalls := 0

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		listFn: func(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
			return opsisdk.ListDatabaseInsightsResponse{}, nil
		},
		createFn: func(_ context.Context, req opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
			createCalls++
			details, ok := req.CreateDatabaseInsightDetails.(opsisdk.CreatePeComanagedDatabaseInsightDetails)
			if !ok {
				t.Fatalf("CreateDatabaseInsightDetails = %T, want opsi.CreatePeComanagedDatabaseInsightDetails", req.CreateDatabaseInsightDetails)
			}
			requireStringPtr(t, "CreatePeComanagedDatabaseInsightDetails.CompartmentId", details.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "CreatePeComanagedDatabaseInsightDetails.DatabaseId", details.DatabaseId, resource.Spec.DatabaseId)
			requireStringPtr(t, "CreatePeComanagedDatabaseInsightDetails.DatabaseResourceType", details.DatabaseResourceType, resource.Spec.DatabaseResourceType)
			requireStringPtr(t, "CreatePeComanagedDatabaseInsightDetails.OpsiPrivateEndpointId", details.OpsiPrivateEndpointId, resource.Spec.OpsiPrivateEndpointId)
			requireStringPtr(t, "CreatePeComanagedDatabaseInsightDetails.ServiceName", details.ServiceName, resource.Spec.ServiceName)
			if details.ConnectionDetails == nil {
				t.Fatal("CreatePeComanagedDatabaseInsightDetails.ConnectionDetails = nil, want PE connection details")
			}
			if got := details.ConnectionDetails.Protocol; got != opsisdk.PeComanagedDatabaseConnectionDetailsProtocolTcps {
				t.Fatalf("CreatePeComanagedDatabaseInsightDetails.ConnectionDetails.Protocol = %q, want %q", got, opsisdk.PeComanagedDatabaseConnectionDetailsProtocolTcps)
			}
			requireStringPtr(t, "CreatePeComanagedDatabaseInsightDetails.ConnectionDetails.ServiceName", details.ConnectionDetails.ServiceName, resource.Spec.ConnectionDetails.ServiceName)
			if len(details.ConnectionDetails.Hosts) != 1 {
				t.Fatalf("CreatePeComanagedDatabaseInsightDetails.ConnectionDetails.Hosts = %#v, want one host", details.ConnectionDetails.Hosts)
			}
			requireStringPtr(t, "CreatePeComanagedDatabaseInsightDetails.ConnectionDetails.Hosts[0].HostIp", details.ConnectionDetails.Hosts[0].HostIp, resource.Spec.ConnectionDetails.HostName)
			requireIntPtr(t, "CreatePeComanagedDatabaseInsightDetails.ConnectionDetails.Hosts[0].Port", details.ConnectionDetails.Hosts[0].Port, resource.Spec.ConnectionDetails.Port)
			return opsisdk.CreateDatabaseInsightResponse{
				DatabaseInsight: makeSDKPeComanagedDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
				OpcRequestId:    common.String("opc-create-pe"),
			}, nil
		},
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKPeComanagedDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalls != 1 {
		t.Fatalf("CreateDatabaseInsight() calls = %d, want 1", createCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-pe" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-pe", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDatabaseInsightCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()
	updateCalled := false

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
			updateCalled = true
			return opsisdk.UpdateDatabaseInsightResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateDatabaseInsight() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDatabaseInsightCreateOrUpdateUpdatesMutableTags(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := makeDatabaseInsightResource().Spec
	getCalls := 0
	updateCalls := 0

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			if getCalls == 1 {
				return opsisdk.GetDatabaseInsightResponse{
					DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, currentSpec, opsisdk.LifecycleStateActive),
				}, nil
			}
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			details, ok := req.UpdateDatabaseInsightDetails.(opsisdk.UpdateAutonomousDatabaseInsightDetails)
			if !ok {
				t.Fatalf("UpdateDatabaseInsightDetails = %T, want opsi.UpdateAutonomousDatabaseInsightDetails", req.UpdateDatabaseInsightDetails)
			}
			if got := details.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateDatabaseInsightDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := details.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateDatabaseInsightDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return opsisdk.UpdateDatabaseInsightResponse{
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDatabaseInsight() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetDatabaseInsight() calls = %d, want current read and update follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDatabaseInsightCreateOrUpdateRejectsDatabaseIDDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()
	resource.Spec.DatabaseId = "ocid1.autonomousdatabase.oc1..different"
	currentSpec := resource.Spec
	currentSpec.DatabaseId = testDatabaseInsightDatabaseID
	updateCalled := false

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, currentSpec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
			updateCalled = true
			return opsisdk.UpdateDatabaseInsightResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDatabaseInsight() called after create-only databaseId drift")
	}
	if !strings.Contains(err.Error(), "databaseId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want databaseId force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDatabaseInsightCreateOrUpdateRejectsOmittedCreateOnlySystemTagsBeforeNoop(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()
	resource.Spec.SystemTags = map[string]shared.MapValue{
		"orcl-cloud": {"free-tier-retained": "true"},
	}
	updateCalled := false

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
			updateCalled = true
			return opsisdk.UpdateDatabaseInsightResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want omitted create-only systemTags rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDatabaseInsight() called after omitted create-only systemTags drift")
	}
	if !strings.Contains(err.Error(), "systemTags changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want systemTags force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDatabaseInsightCreateOrUpdateRejectsAdvancedFeaturesFalseDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()
	resource.Spec.IsAdvancedFeaturesEnabled = false
	currentSpec := resource.Spec
	currentSpec.IsAdvancedFeaturesEnabled = true
	updateCalled := false

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, currentSpec, opsisdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
			updateCalled = true
			return opsisdk.UpdateDatabaseInsightResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only advanced features drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDatabaseInsight() called after create-only advanced features drift")
	}
	if !strings.Contains(err.Error(), "isAdvancedFeaturesEnabled changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want isAdvancedFeaturesEnabled force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDatabaseInsightConnectionDetailsForEntitySourceReturnsNilForEmptyDetails(t *testing.T) {
	t.Parallel()

	if got := databaseInsightConnectionDetailsForEntitySource(nil, nil, opsiv1beta1.DatabaseInsightConnectionDetails{}); got != nil {
		t.Fatalf("databaseInsightConnectionDetailsForEntitySource() = %T, want nil", got)
	}
	peDefaults := map[string]any{"entitySource": string(databaseInsightEntitySourcePeComanaged)}
	if got := databaseInsightConnectionDetailsForEntitySource(peDefaults, nil, opsiv1beta1.DatabaseInsightConnectionDetails{}); got != nil {
		t.Fatalf("databaseInsightConnectionDetailsForEntitySource(PE_COMANAGED_DATABASE) = %T, want nil", got)
	}
}

func TestDatabaseInsightRequestBodySideChannelUsesOperationContext(t *testing.T) {
	t.Parallel()

	resource := makeDatabaseInsightResource()
	firstCtx, secondCtx := newDatabaseInsightRequestBodyContexts(t)

	requireDatabaseInsightCreateBodyContextIsolation(t, resource, firstCtx, secondCtx)
	requireDatabaseInsightUpdateBodyContextIsolation(t, resource, firstCtx, secondCtx)
}

func newDatabaseInsightRequestBodyContexts(t *testing.T) (context.Context, context.Context) {
	t.Helper()

	firstCtx := withDatabaseInsightRequestBodyToken(context.Background())
	secondCtx := withDatabaseInsightRequestBodyToken(context.Background())
	t.Cleanup(func() {
		clearDatabaseInsightRequestBodies(firstCtx)
		clearDatabaseInsightRequestBodies(secondCtx)
	})
	return firstCtx, secondCtx
}

func requireDatabaseInsightCreateBodyContextIsolation(
	t *testing.T,
	resource *opsiv1beta1.DatabaseInsight,
	firstCtx context.Context,
	secondCtx context.Context,
) {
	t.Helper()

	firstCreate := opsisdk.CreateAutonomousDatabaseInsightDetails{
		CompartmentId: common.String("first-compartment"),
	}
	secondCreate := opsisdk.CreatePeComanagedDatabaseInsightDetails{
		CompartmentId: common.String("second-compartment"),
	}
	if err := stashDatabaseInsightCreateBody(firstCtx, resource, firstCreate); err != nil {
		t.Fatalf("stashDatabaseInsightCreateBody(first) error = %v", err)
	}
	if err := stashDatabaseInsightCreateBody(secondCtx, resource, secondCreate); err != nil {
		t.Fatalf("stashDatabaseInsightCreateBody(second) error = %v", err)
	}
	gotFirstCreate, err := takeDatabaseInsightCreateBody(firstCtx, common.String("same-retry-token"))
	if err != nil {
		t.Fatalf("takeDatabaseInsightCreateBody(first) error = %v", err)
	}
	if _, ok := gotFirstCreate.(opsisdk.CreateAutonomousDatabaseInsightDetails); !ok {
		t.Fatalf("first create body = %T, want opsi.CreateAutonomousDatabaseInsightDetails", gotFirstCreate)
	}
	gotSecondCreate, err := takeDatabaseInsightCreateBody(secondCtx, common.String("same-retry-token"))
	if err != nil {
		t.Fatalf("takeDatabaseInsightCreateBody(second) error = %v", err)
	}
	if _, ok := gotSecondCreate.(opsisdk.CreatePeComanagedDatabaseInsightDetails); !ok {
		t.Fatalf("second create body = %T, want opsi.CreatePeComanagedDatabaseInsightDetails", gotSecondCreate)
	}
}

func requireDatabaseInsightUpdateBodyContextIsolation(
	t *testing.T,
	resource *opsiv1beta1.DatabaseInsight,
	firstCtx context.Context,
	secondCtx context.Context,
) {
	t.Helper()

	firstUpdate := opsisdk.UpdateAutonomousDatabaseInsightDetails{
		FreeformTags: map[string]string{"case": "first"},
	}
	secondUpdate := opsisdk.UpdatePeComanagedDatabaseInsightDetails{
		FreeformTags: map[string]string{"case": "second"},
	}
	current := map[string]any{"id": testDatabaseInsightID}
	if err := stashDatabaseInsightUpdateBody(firstCtx, resource, current, firstUpdate); err != nil {
		t.Fatalf("stashDatabaseInsightUpdateBody(first) error = %v", err)
	}
	if err := stashDatabaseInsightUpdateBody(secondCtx, resource, current, secondUpdate); err != nil {
		t.Fatalf("stashDatabaseInsightUpdateBody(second) error = %v", err)
	}
	gotFirstUpdate, err := takeDatabaseInsightUpdateBody(firstCtx, common.String(testDatabaseInsightID))
	if err != nil {
		t.Fatalf("takeDatabaseInsightUpdateBody(first) error = %v", err)
	}
	if _, ok := gotFirstUpdate.(opsisdk.UpdateAutonomousDatabaseInsightDetails); !ok {
		t.Fatalf("first update body = %T, want opsi.UpdateAutonomousDatabaseInsightDetails", gotFirstUpdate)
	}
	gotSecondUpdate, err := takeDatabaseInsightUpdateBody(secondCtx, common.String(testDatabaseInsightID))
	if err != nil {
		t.Fatalf("takeDatabaseInsightUpdateBody(second) error = %v", err)
	}
	if _, ok := gotSecondUpdate.(opsisdk.UpdatePeComanagedDatabaseInsightDetails); !ok {
		t.Fatalf("second update body = %T, want opsi.UpdatePeComanagedDatabaseInsightDetails", gotSecondUpdate)
	}
}

func TestDatabaseInsightCredentialDetailsInfersNamedCredentialWhenTypeOmitted(t *testing.T) {
	t.Parallel()

	details := databaseInsightCredentialDetailsFromTypedFields(opsiv1beta1.DatabaseInsightCredentialDetails{
		CredentialSourceName: "agent-wallet",
		NamedCredentialId:    "ocid1.managementagentnamedcredential.oc1..named",
	})
	named, ok := details.(opsisdk.CredentialByNamedCredentials)
	if !ok {
		t.Fatalf("databaseInsightCredentialDetailsFromTypedFields() = %T, want opsi.CredentialByNamedCredentials", details)
	}
	requireStringPtr(t, "CredentialByNamedCredentials.CredentialSourceName", named.CredentialSourceName, "agent-wallet")
	requireStringPtr(t, "CredentialByNamedCredentials.NamedCredentialId", named.NamedCredentialId, "ocid1.managementagentnamedcredential.oc1..named")
}

func TestDatabaseInsightDeleteKeepsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			if getCalls <= 3 {
				return opsisdk.GetDatabaseInsightResponse{
					DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
				}, nil
			}
			return opsisdk.GetDatabaseInsightResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "database insight is gone")
		},
		deleteFn: func(_ context.Context, req opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.DeleteDatabaseInsightResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDatabaseInsight() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireDeletePendingStatus(t, resource)
	requireLastCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDatabaseInsight() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestDatabaseInsightDeleteRetainsFinalizerForPendingWriteLifecycle(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		state     opsisdk.LifecycleStateEnum
		wantPhase shared.OSOKAsyncPhase
	}{
		{name: "creating", state: opsisdk.LifecycleStateCreating, wantPhase: shared.OSOKAsyncPhaseCreate},
		{name: "updating", state: opsisdk.LifecycleStateUpdating, wantPhase: shared.OSOKAsyncPhaseUpdate},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runDatabaseInsightPendingWriteDeleteCase(t, tt.state, tt.wantPhase)
		})
	}
}

func runDatabaseInsightPendingWriteDeleteCase(
	t *testing.T,
	state opsisdk.LifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	resource := makeTrackedDatabaseInsightResource()
	getCalls := 0
	deleteCalled := false
	client := newPendingWriteDeleteDatabaseInsightClient(t, resource, state, &getCalls, &deleteCalled)

	deleted, err := client.Delete(context.Background(), resource)
	requireDatabaseInsightPendingWriteDeleteResult(
		t,
		resource,
		deleted,
		err,
		getCalls,
		deleteCalled,
		state,
		wantPhase,
	)
}

func newPendingWriteDeleteDatabaseInsightClient(
	t *testing.T,
	resource *opsiv1beta1.DatabaseInsight,
	state opsisdk.LifecycleStateEnum,
	getCalls *int,
	deleteCalled *bool,
) DatabaseInsightServiceClient {
	t.Helper()

	return newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			(*getCalls)++
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, state),
			}, nil
		},
		deleteFn: func(context.Context, opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error) {
			*deleteCalled = true
			return opsisdk.DeleteDatabaseInsightResponse{}, nil
		},
	})
}

func requireDatabaseInsightPendingWriteDeleteResult(
	t *testing.T,
	resource *opsiv1beta1.DatabaseInsight,
	deleted bool,
	err error,
	getCalls int,
	deleteCalled bool,
	state opsisdk.LifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	requireDatabaseInsightPendingWriteDeleteCallResult(t, deleted, err, getCalls, deleteCalled)
	requireDatabaseInsightPendingWriteStatus(t, resource, state, wantPhase)
}

func requireDatabaseInsightPendingWriteDeleteCallResult(
	t *testing.T,
	deleted bool,
	err error,
	getCalls int,
	deleteCalled bool,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while write is pending")
	}
	if deleteCalled {
		t.Fatal("DeleteDatabaseInsight() called while write lifecycle state is still pending")
	}
	if getCalls != 2 {
		t.Fatalf("GetDatabaseInsight() calls = %d, want pre-delete and generated confirmation reads", getCalls)
	}
}

func requireDatabaseInsightPendingWriteStatus(
	t *testing.T,
	resource *opsiv1beta1.DatabaseInsight,
	state opsisdk.LifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending write tracker")
	}
	if !databaseInsightPendingWriteStatusMatches(current, state, wantPhase) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle %s %s", current, wantPhase, state)
	}
	if got := resource.Status.OsokStatus.Message; got != databaseInsightPendingWriteDeleteMessage {
		t.Fatalf("status.status.message = %q, want %q", got, databaseInsightPendingWriteDeleteMessage)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func databaseInsightPendingWriteStatusMatches(
	current *shared.OSOKAsyncOperation,
	state opsisdk.LifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) bool {
	return current.Phase == wantPhase &&
		current.Source == shared.OSOKAsyncSourceLifecycle &&
		current.NormalizedClass == shared.OSOKAsyncClassPending &&
		current.RawStatus == string(state)
}

func TestDatabaseInsightDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{
				DatabaseInsight: makeSDKDatabaseInsight(testDatabaseInsightID, resource.Spec, opsisdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error) {
			requireStringPtr(t, "DeleteDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.DeleteDatabaseInsightResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDatabaseInsightDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDatabaseInsightResource()
	deleteCalled := false

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		getFn: func(_ context.Context, req opsisdk.GetDatabaseInsightRequest) (opsisdk.GetDatabaseInsightResponse, error) {
			requireStringPtr(t, "GetDatabaseInsightRequest.DatabaseInsightId", req.DatabaseInsightId, testDatabaseInsightID)
			return opsisdk.GetDatabaseInsightResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, opsisdk.DeleteDatabaseInsightRequest) (opsisdk.DeleteDatabaseInsightResponse, error) {
			deleteCalled = true
			return opsisdk.DeleteDatabaseInsightResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteDatabaseInsight() called after auth-shaped pre-delete confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDatabaseInsightCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeDatabaseInsightResource()

	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		listFn: func(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
			return opsisdk.ListDatabaseInsightsResponse{}, nil
		},
		createFn: func(context.Context, opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
			return opsisdk.CreateDatabaseInsightResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDatabaseInsightCreateRejectsCredentialJSONErrorsBeforeMutation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*opsiv1beta1.DatabaseInsight)
		want   string
	}{
		{
			name: "credentialDetails invalid JSON",
			mutate: func(resource *opsiv1beta1.DatabaseInsight) {
				resource.Spec.CredentialDetails.JsonData = `{"credentialType":`
			},
			want: "credentialDetails.jsonData: decode DatabaseInsight credential jsonData",
		},
		{
			name: "credentialDetails unsupported credential type",
			mutate: func(resource *opsiv1beta1.DatabaseInsight) {
				resource.Spec.CredentialDetails.JsonData = `{"credentialType":"CREDENTIALS_BY_UNKNOWN"}`
			},
			want: "credentialDetails.jsonData: unsupported DatabaseInsight credentialType",
		},
		{
			name: "connectionCredentialDetails invalid JSON",
			mutate: func(resource *opsiv1beta1.DatabaseInsight) {
				resource.Spec.ConnectionCredentialDetails.JsonData = `{"credentialType":`
			},
			want: "connectionCredentialDetails.jsonData: decode DatabaseInsight credential jsonData",
		},
		{
			name: "connectionCredentialDetails unsupported credential type",
			mutate: func(resource *opsiv1beta1.DatabaseInsight) {
				resource.Spec.ConnectionCredentialDetails.JsonData = `{"credentialType":"CREDENTIALS_BY_UNKNOWN"}`
			},
			want: "connectionCredentialDetails.jsonData: unsupported DatabaseInsight credentialType",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			requireDatabaseInsightCredentialJSONRejected(t, test.mutate, test.want)
		})
	}
}

func requireDatabaseInsightCreateRequest(
	t *testing.T,
	req opsisdk.CreateDatabaseInsightRequest,
	resource *opsiv1beta1.DatabaseInsight,
) {
	t.Helper()
	details, ok := req.CreateDatabaseInsightDetails.(opsisdk.CreateAutonomousDatabaseInsightDetails)
	if !ok {
		t.Fatalf("CreateDatabaseInsightDetails = %T, want opsi.CreateAutonomousDatabaseInsightDetails", req.CreateDatabaseInsightDetails)
	}
	requireStringPtr(t, "CreateDatabaseInsightDetails.CompartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateDatabaseInsightDetails.DatabaseId", details.DatabaseId, resource.Spec.DatabaseId)
	requireStringPtr(t, "CreateDatabaseInsightDetails.DatabaseResourceType", details.DatabaseResourceType, resource.Spec.DatabaseResourceType)
	if details.IsAdvancedFeaturesEnabled == nil || *details.IsAdvancedFeaturesEnabled {
		t.Fatalf("CreateDatabaseInsightDetails.IsAdvancedFeaturesEnabled = %v, want false", details.IsAdvancedFeaturesEnabled)
	}
	if got := details.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateDatabaseInsightDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := details.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateDatabaseInsightDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateDatabaseInsightRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func databaseInsightPagedListFn(
	t *testing.T,
	resource *opsiv1beta1.DatabaseInsight,
	listCalls *int,
) func(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
	t.Helper()
	return func(_ context.Context, req opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
		t.Helper()
		(*listCalls)++
		requireStringPtr(t, "ListDatabaseInsightsRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
		requireStringSlice(t, "ListDatabaseInsightsRequest.DatabaseId", req.DatabaseId, []string{resource.Spec.DatabaseId})
		if req.CompartmentIdInSubtree == nil || *req.CompartmentIdInSubtree {
			t.Fatalf("ListDatabaseInsightsRequest.CompartmentIdInSubtree = %v, want false", req.CompartmentIdInSubtree)
		}
		if *listCalls != 1 {
			requireStringPtr(t, "second ListDatabaseInsightsRequest.Page", req.Page, "page-2")
			return databaseInsightListPage(resource.Spec, testDatabaseInsightID, "")
		}
		if req.Page != nil {
			t.Fatalf("first ListDatabaseInsightsRequest.Page = %q, want nil", *req.Page)
		}
		otherSpec := resource.Spec
		otherSpec.DatabaseId = "ocid1.autonomousdatabase.oc1..other"
		return databaseInsightListPage(otherSpec, "ocid1.databaseinsight.oc1..other", "page-2")
	}
}

func databaseInsightListPage(
	spec opsiv1beta1.DatabaseInsightSpec,
	id string,
	nextPage string,
) (opsisdk.ListDatabaseInsightsResponse, error) {
	response := opsisdk.ListDatabaseInsightsResponse{
		DatabaseInsightsCollection: opsisdk.DatabaseInsightsCollection{
			Items: []opsisdk.DatabaseInsightSummary{
				makeSDKDatabaseInsightSummary(id, spec, opsisdk.LifecycleStateActive),
			},
		},
	}
	if nextPage != "" {
		response.OpcNextPage = common.String(nextPage)
	}
	return response, nil
}

func requireDatabaseInsightCredentialJSONRejected(
	t *testing.T,
	mutate func(*opsiv1beta1.DatabaseInsight),
	want string,
) {
	t.Helper()
	resource := makeDatabaseInsightResource()
	mutate(resource)
	mutationCalled := false
	client := newTestDatabaseInsightClient(&fakeDatabaseInsightOCIClient{
		listFn: func(context.Context, opsisdk.ListDatabaseInsightsRequest) (opsisdk.ListDatabaseInsightsResponse, error) {
			return opsisdk.ListDatabaseInsightsResponse{}, nil
		},
		createFn: func(context.Context, opsisdk.CreateDatabaseInsightRequest) (opsisdk.CreateDatabaseInsightResponse, error) {
			mutationCalled = true
			return opsisdk.CreateDatabaseInsightResponse{}, nil
		},
		updateFn: func(context.Context, opsisdk.UpdateDatabaseInsightRequest) (opsisdk.UpdateDatabaseInsightResponse, error) {
			mutationCalled = true
			return opsisdk.UpdateDatabaseInsightResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDatabaseInsightRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want credential jsonData rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if mutationCalled {
		t.Fatal("mutating OCI call ran after credential jsonData rejection")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("CreateOrUpdate() error = %v, want substring %q", err, want)
	}
}

func requireDeletePendingStatus(t *testing.T, resource *opsiv1beta1.DatabaseInsight) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(opsisdk.LifecycleStateActive) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle delete pending ACTIVE", current)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", name, *got, want)
	}
}

func requireStringSlice(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}

func requireLastCondition(t *testing.T, resource *opsiv1beta1.DatabaseInsight, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
