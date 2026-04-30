/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package protecteddatabase

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testProtectedDatabaseID            = "ocid1.protecteddatabase.oc1..database"
	testProtectedDatabaseCompartmentID = "ocid1.compartment.oc1..database"
	testProtectedDatabasePolicyID      = "ocid1.protectionpolicy.oc1..policy"
	testProtectedDatabaseSubnetID      = "ocid1.recoveryservicesubnet.oc1..subnet"
	testProtectedDatabaseSubnetID2     = "ocid1.recoveryservicesubnet.oc1..subnet2"
	testProtectedDatabaseDBID          = "ocid1.database.oc1..source"
	testProtectedDatabaseDisplayName   = "protected-db"
	testProtectedDatabaseUniqueName    = "CDB01"
	testProtectedDatabasePassword      = "AAaa11__"
	testProtectedDatabaseNewPassword   = "BBbb22__"
)

type fakeProtectedDatabaseOCIClient struct {
	createFn             func(context.Context, recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error)
	getFn                func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error)
	listFn               func(context.Context, recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error)
	updateFn             func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error)
	changeCompartmentFn  func(context.Context, recoverysdk.ChangeProtectedDatabaseCompartmentRequest) (recoverysdk.ChangeProtectedDatabaseCompartmentResponse, error)
	changeSubscriptionFn func(context.Context, recoverysdk.ChangeProtectedDatabaseSubscriptionRequest) (recoverysdk.ChangeProtectedDatabaseSubscriptionResponse, error)
	deleteFn             func(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error)
	workRequestFn        func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error)
}

func (f *fakeProtectedDatabaseOCIClient) CreateProtectedDatabase(ctx context.Context, req recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return recoverysdk.CreateProtectedDatabaseResponse{}, nil
}

func (f *fakeProtectedDatabaseOCIClient) GetProtectedDatabase(ctx context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return recoverysdk.GetProtectedDatabaseResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "protected database is missing")
}

func (f *fakeProtectedDatabaseOCIClient) ListProtectedDatabases(ctx context.Context, req recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return recoverysdk.ListProtectedDatabasesResponse{}, nil
}

func (f *fakeProtectedDatabaseOCIClient) UpdateProtectedDatabase(ctx context.Context, req recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
}

func (f *fakeProtectedDatabaseOCIClient) ChangeProtectedDatabaseCompartment(ctx context.Context, req recoverysdk.ChangeProtectedDatabaseCompartmentRequest) (recoverysdk.ChangeProtectedDatabaseCompartmentResponse, error) {
	if f.changeCompartmentFn != nil {
		return f.changeCompartmentFn(ctx, req)
	}
	return recoverysdk.ChangeProtectedDatabaseCompartmentResponse{}, nil
}

func (f *fakeProtectedDatabaseOCIClient) ChangeProtectedDatabaseSubscription(ctx context.Context, req recoverysdk.ChangeProtectedDatabaseSubscriptionRequest) (recoverysdk.ChangeProtectedDatabaseSubscriptionResponse, error) {
	if f.changeSubscriptionFn != nil {
		return f.changeSubscriptionFn(ctx, req)
	}
	return recoverysdk.ChangeProtectedDatabaseSubscriptionResponse{}, nil
}

func (f *fakeProtectedDatabaseOCIClient) DeleteProtectedDatabase(ctx context.Context, req recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return recoverysdk.DeleteProtectedDatabaseResponse{}, nil
}

func (f *fakeProtectedDatabaseOCIClient) GetWorkRequest(ctx context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return recoverysdk.GetWorkRequestResponse{}, nil
}

type fakeProtectedDatabaseCredentialClient struct {
	secrets map[string]map[string][]byte
}

func newFakeProtectedDatabaseCredentialClient() *fakeProtectedDatabaseCredentialClient {
	return &fakeProtectedDatabaseCredentialClient{secrets: map[string]map[string][]byte{}}
}

func (f *fakeProtectedDatabaseCredentialClient) CreateSecret(
	_ context.Context,
	name string,
	namespace string,
	_ map[string]string,
	data map[string][]byte,
) (bool, error) {
	key := protectedDatabaseSecretKey(name, namespace)
	if _, ok := f.secrets[key]; ok {
		return false, apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, name)
	}
	f.secrets[key] = cloneProtectedDatabaseBytesMap(data)
	return true, nil
}

func (f *fakeProtectedDatabaseCredentialClient) DeleteSecret(_ context.Context, name string, namespace string) (bool, error) {
	key := protectedDatabaseSecretKey(name, namespace)
	if _, ok := f.secrets[key]; !ok {
		return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	delete(f.secrets, key)
	return true, nil
}

func (f *fakeProtectedDatabaseCredentialClient) GetSecret(
	_ context.Context,
	name string,
	namespace string,
) (map[string][]byte, error) {
	data, ok := f.secrets[protectedDatabaseSecretKey(name, namespace)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	return cloneProtectedDatabaseBytesMap(data), nil
}

func (f *fakeProtectedDatabaseCredentialClient) UpdateSecret(
	_ context.Context,
	name string,
	namespace string,
	_ map[string]string,
	data map[string][]byte,
) (bool, error) {
	key := protectedDatabaseSecretKey(name, namespace)
	if _, ok := f.secrets[key]; !ok {
		return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	f.secrets[key] = cloneProtectedDatabaseBytesMap(data)
	return true, nil
}

func protectedDatabaseSecretKey(name string, namespace string) string {
	return namespace + "/" + name
}

func cloneProtectedDatabaseBytesMap(source map[string][]byte) map[string][]byte {
	if source == nil {
		return nil
	}
	clone := make(map[string][]byte, len(source))
	for key, value := range source {
		valueClone := make([]byte, len(value))
		copy(valueClone, value)
		clone[key] = valueClone
	}
	return clone
}

func newTestProtectedDatabaseClient(client protectedDatabaseOCIClient) ProtectedDatabaseServiceClient {
	return newTestProtectedDatabaseClientWithCredentialClient(client, newFakeProtectedDatabaseCredentialClient())
}

func newTestProtectedDatabaseClientWithCredentialClient(
	client protectedDatabaseOCIClient,
	credentialClient *fakeProtectedDatabaseCredentialClient,
) ProtectedDatabaseServiceClient {
	return newProtectedDatabaseServiceClientWithOCIClientAndCredentialClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
		credentialClient,
	)
}

func makeProtectedDatabaseResource() *recoveryv1beta1.ProtectedDatabase {
	return &recoveryv1beta1.ProtectedDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProtectedDatabaseDisplayName,
			Namespace: "default",
		},
		Spec: recoveryv1beta1.ProtectedDatabaseSpec{
			DisplayName:        testProtectedDatabaseDisplayName,
			DbUniqueName:       testProtectedDatabaseUniqueName,
			Password:           testProtectedDatabasePassword,
			ProtectionPolicyId: testProtectedDatabasePolicyID,
			RecoveryServiceSubnets: []recoveryv1beta1.ProtectedDatabaseRecoveryServiceSubnet{
				{RecoveryServiceSubnetId: testProtectedDatabaseSubnetID},
			},
			CompartmentId:     testProtectedDatabaseCompartmentID,
			DatabaseSize:      string(recoverysdk.DatabaseSizesM),
			DatabaseId:        testProtectedDatabaseDBID,
			ChangeRate:        0.25,
			CompressionRatio:  1.5,
			IsRedoLogsShipped: true,
			SubscriptionId:    "subscription-1",
			FreeformTags:      map[string]string{"env": "test"},
			DefinedTags:       map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedProtectedDatabaseResource() *recoveryv1beta1.ProtectedDatabase {
	resource := makeProtectedDatabaseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectedDatabaseID)
	resource.Status.Id = testProtectedDatabaseID
	resource.Status.CompartmentId = testProtectedDatabaseCompartmentID
	resource.Status.DbUniqueName = testProtectedDatabaseUniqueName
	resource.Status.DisplayName = testProtectedDatabaseDisplayName
	resource.Status.ProtectionPolicyId = testProtectedDatabasePolicyID
	resource.Status.RecoveryServiceSubnets = resource.Spec.RecoveryServiceSubnets
	resource.Status.DatabaseSize = string(recoverysdk.DatabaseSizesM)
	resource.Status.DatabaseId = testProtectedDatabaseDBID
	resource.Status.ChangeRate = 0.25
	resource.Status.CompressionRatio = 1.5
	resource.Status.IsRedoLogsShipped = true
	resource.Status.SubscriptionId = "subscription-1"
	resource.Status.LifecycleState = string(recoverysdk.LifecycleStateActive)
	return resource
}

func markProtectedDatabaseCreatedWithLegacyPasswordHash(resource *recoveryv1beta1.ProtectedDatabase) {
	now := metav1.Now()
	resource.Status.OsokStatus.CreatedAt = &now
	resource.Status.OsokStatus.Message = protectedDatabaseLegacyPasswordHashKey + protectedDatabasePasswordHash(resource.Spec.Password)
}

func makeProtectedDatabaseRequest(resource *recoveryv1beta1.ProtectedDatabase) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKProtectedDatabase(
	id string,
	spec recoveryv1beta1.ProtectedDatabaseSpec,
	state recoverysdk.LifecycleStateEnum,
) recoverysdk.ProtectedDatabase {
	return recoverysdk.ProtectedDatabase{
		Id:                     common.String(id),
		CompartmentId:          common.String(spec.CompartmentId),
		DbUniqueName:           common.String(spec.DbUniqueName),
		VpcUserName:            common.String("vpc-user"),
		DatabaseSize:           recoverysdk.DatabaseSizesEnum(spec.DatabaseSize),
		ProtectionPolicyId:     common.String(spec.ProtectionPolicyId),
		RecoveryServiceSubnets: makeSDKProtectedDatabaseSubnetDetails(spec.RecoveryServiceSubnets),
		DisplayName:            common.String(spec.DisplayName),
		DatabaseId:             common.String(spec.DatabaseId),
		ChangeRate:             common.Float64(spec.ChangeRate),
		CompressionRatio:       common.Float64(spec.CompressionRatio),
		IsRedoLogsShipped:      common.Bool(spec.IsRedoLogsShipped),
		LifecycleState:         state,
		SubscriptionId:         common.String(spec.SubscriptionId),
		FreeformTags:           cloneProtectedDatabaseStringMap(spec.FreeformTags),
		DefinedTags:            protectedDatabaseDefinedTags(spec.DefinedTags),
	}
}

func makeSDKProtectedDatabaseSummary(
	id string,
	spec recoveryv1beta1.ProtectedDatabaseSpec,
	state recoverysdk.LifecycleStateEnum,
) recoverysdk.ProtectedDatabaseSummary {
	return recoverysdk.ProtectedDatabaseSummary{
		Id:                     common.String(id),
		CompartmentId:          common.String(spec.CompartmentId),
		DbUniqueName:           common.String(spec.DbUniqueName),
		VpcUserName:            common.String("vpc-user"),
		DatabaseSize:           recoverysdk.DatabaseSizesEnum(spec.DatabaseSize),
		ProtectionPolicyId:     common.String(spec.ProtectionPolicyId),
		RecoveryServiceSubnets: makeSDKProtectedDatabaseSubnetDetails(spec.RecoveryServiceSubnets),
		DisplayName:            common.String(spec.DisplayName),
		DatabaseId:             common.String(spec.DatabaseId),
		LifecycleState:         state,
		SubscriptionId:         common.String(spec.SubscriptionId),
		FreeformTags:           cloneProtectedDatabaseStringMap(spec.FreeformTags),
		DefinedTags:            protectedDatabaseDefinedTags(spec.DefinedTags),
	}
}

func makeSDKProtectedDatabaseSubnetDetails(
	subnets []recoveryv1beta1.ProtectedDatabaseRecoveryServiceSubnet,
) []recoverysdk.RecoveryServiceSubnetDetails {
	result := make([]recoverysdk.RecoveryServiceSubnetDetails, 0, len(subnets))
	for _, subnet := range subnets {
		result = append(result, recoverysdk.RecoveryServiceSubnetDetails{
			RecoveryServiceSubnetId: common.String(subnet.RecoveryServiceSubnetId),
			LifecycleState:          recoverysdk.LifecycleStateActive,
		})
	}
	return result
}

func makeProtectedDatabaseWorkRequest(
	id string,
	operation recoverysdk.OperationTypeEnum,
	status recoverysdk.OperationStatusEnum,
	action recoverysdk.ActionTypeEnum,
	resourceID string,
) recoverysdk.WorkRequest {
	return recoverysdk.WorkRequest{
		OperationType: operation,
		Status:        status,
		Id:            common.String(id),
		CompartmentId: common.String(testProtectedDatabaseCompartmentID),
		Resources: []recoverysdk.WorkRequestResource{
			{
				EntityType: common.String("protectedDatabase"),
				ActionType: action,
				Identifier: common.String(resourceID),
				EntityUri:  common.String("/protectedDatabases/" + resourceID),
			},
		},
		PercentComplete: common.Float32(100),
		TimeAccepted:    &common.SDKTime{Time: metav1.Now().Time},
	}
}

func TestProtectedDatabaseCreateOrUpdateBindsExistingDatabaseByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeProtectedDatabaseResource()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		listFn: protectedDatabasePagedBindListFn(t, resource, &listCalls),
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error) {
			createCalled = true
			return recoverysdk.CreateProtectedDatabaseResponse{}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateProtectedDatabase() called for existing database")
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called for matching database")
	}
	if listCalls != 2 {
		t.Fatalf("ListProtectedDatabases() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProtectedDatabaseID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProtectedDatabaseID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func protectedDatabasePagedBindListFn(
	t *testing.T,
	resource *recoveryv1beta1.ProtectedDatabase,
	listCalls *int,
) func(context.Context, recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error) {
	t.Helper()
	return func(_ context.Context, req recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error) {
		*listCalls++
		requireStringPtr(t, "ListProtectedDatabasesRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
		if req.DisplayName != nil {
			t.Fatalf("ListProtectedDatabasesRequest.DisplayName = %q, want nil to bind renamed databases by dbUniqueName", *req.DisplayName)
		}
		if *listCalls == 1 {
			return firstProtectedDatabaseBindListPage(t, req, resource)
		}
		requireStringPtr(t, "second ListProtectedDatabasesRequest.Page", req.Page, "page-2")
		return recoverysdk.ListProtectedDatabasesResponse{
			ProtectedDatabaseCollection: recoverysdk.ProtectedDatabaseCollection{
				Items: []recoverysdk.ProtectedDatabaseSummary{
					makeSDKProtectedDatabaseSummary(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
				},
			},
		}, nil
	}
}

func firstProtectedDatabaseBindListPage(
	t *testing.T,
	req recoverysdk.ListProtectedDatabasesRequest,
	resource *recoveryv1beta1.ProtectedDatabase,
) (recoverysdk.ListProtectedDatabasesResponse, error) {
	t.Helper()
	if req.Page != nil {
		t.Fatalf("first ListProtectedDatabasesRequest.Page = %q, want nil", *req.Page)
	}
	otherSpec := resource.Spec
	otherSpec.DbUniqueName = "OTHERDB"
	return recoverysdk.ListProtectedDatabasesResponse{
		ProtectedDatabaseCollection: recoverysdk.ProtectedDatabaseCollection{
			Items: []recoverysdk.ProtectedDatabaseSummary{
				makeSDKProtectedDatabaseSummary("ocid1.protecteddatabase.oc1..other", otherSpec, recoverysdk.LifecycleStateActive),
			},
		},
		OpcNextPage: common.String("page-2"),
	}, nil
}

func TestProtectedDatabaseCreateStartsWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeProtectedDatabaseResource()
	createCalls := 0
	workRequestCalls := 0
	getCalls := 0
	credentialClient := newFakeProtectedDatabaseCredentialClient()

	client := newTestProtectedDatabaseClientWithCredentialClient(&fakeProtectedDatabaseOCIClient{
		listFn: func(context.Context, recoverysdk.ListProtectedDatabasesRequest) (recoverysdk.ListProtectedDatabasesResponse, error) {
			return recoverysdk.ListProtectedDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, req recoverysdk.CreateProtectedDatabaseRequest) (recoverysdk.CreateProtectedDatabaseResponse, error) {
			createCalls++
			requireProtectedDatabaseCreateRequest(t, req, resource)
			if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
				t.Fatal("CreateProtectedDatabaseRequest.OpcRetryToken is empty")
			}
			return recoverysdk.CreateProtectedDatabaseResponse{
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-create")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-create",
					recoverysdk.OperationTypeCreateProtectedDatabase,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeCreated,
					testProtectedDatabaseID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
	}, credentialClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalls != 1 {
		t.Fatalf("CreateProtectedDatabase() calls = %d, want 1", createCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProtectedDatabaseID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProtectedDatabaseID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireProtectedDatabaseSecretPasswordHash(t, credentialClient, resource, testProtectedDatabasePassword)
	requireProtectedDatabaseStatusHasNoPasswordMaterial(t, resource)
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	markProtectedDatabaseCreatedWithLegacyPasswordHash(resource)
	updateCalled := false
	credentialClient := newFakeProtectedDatabaseCredentialClient()

	client := newTestProtectedDatabaseClientWithCredentialClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	}, credentialClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called for matching readback")
	}
	requireProtectedDatabaseSecretPasswordHash(t, credentialClient, resource, testProtectedDatabasePassword)
	requireProtectedDatabaseStatusHasNoPasswordMaterial(t, resource)
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateOmittedTagsAreUnmanaged(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil
	currentSpec := resource.Spec
	currentSpec.FreeformTags = map[string]string{"owner": "external"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "external"}}
	updateCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, currentSpec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called when spec omits freeformTags and definedTags")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateIgnoresOmittedOptionalCreateOnlyReadback(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	resource.Spec.DatabaseId = ""
	resource.Spec.ChangeRate = 0
	resource.Spec.CompressionRatio = 0
	currentSpec := resource.Spec
	currentSpec.DatabaseId = testProtectedDatabaseDBID
	currentSpec.ChangeRate = 0.25
	currentSpec.CompressionRatio = 1.5
	updateCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, currentSpec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called when optional create-only fields are omitted")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	resource.Spec.DisplayName = "updated-protected-db"
	resource.Spec.DatabaseSize = string(recoverysdk.DatabaseSizesL)
	resource.Spec.DatabaseSizeInGBs = 2048
	resource.Spec.ProtectionPolicyId = "ocid1.protectionpolicy.oc1..updated"
	resource.Spec.RecoveryServiceSubnets = []recoveryv1beta1.ProtectedDatabaseRecoveryServiceSubnet{
		{RecoveryServiceSubnetId: testProtectedDatabaseSubnetID2},
	}
	resource.Spec.IsRedoLogsShipped = false
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := makeProtectedDatabaseResource().Spec
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn:         protectedDatabaseMutableReadFn(t, resource, currentSpec, &getCalls),
		updateFn:      protectedDatabaseMutableUpdateFn(t, resource, &updateCalls),
		workRequestFn: protectedDatabaseUpdateWorkRequestFn(t, &workRequestCalls),
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateProtectedDatabase() calls = %d, want 1", updateCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want current read and work-request follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateChangesCompartmentWithChangeOperation(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..moved"
	currentSpec := makeProtectedDatabaseResource().Spec
	getCalls := 0
	changeCalls := 0
	updateCalled := false
	workRequestCalls := 0

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			if getCalls == 1 {
				return recoverysdk.GetProtectedDatabaseResponse{
					ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, currentSpec, recoverysdk.LifecycleStateActive),
				}, nil
			}
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		changeCompartmentFn: func(_ context.Context, req recoverysdk.ChangeProtectedDatabaseCompartmentRequest) (recoverysdk.ChangeProtectedDatabaseCompartmentResponse, error) {
			changeCalls++
			requireStringPtr(t, "ChangeProtectedDatabaseCompartmentRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			requireStringPtr(t, "ChangeProtectedDatabaseCompartmentDetails.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			return recoverysdk.ChangeProtectedDatabaseCompartmentResponse{
				OpcWorkRequestId: common.String("wr-compartment"),
				OpcRequestId:     common.String("opc-compartment"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-compartment")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-compartment",
					recoverysdk.OperationTypeMoveProtectedDatabase,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeUpdated,
					testProtectedDatabaseID,
				),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if changeCalls != 1 {
		t.Fatalf("ChangeProtectedDatabaseCompartment() calls = %d, want 1", changeCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called for compartment change")
	}
	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-compartment" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-compartment", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateChangesSubscriptionWithChangeOperation(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	resource.Spec.SubscriptionId = "subscription-2"
	currentSpec := makeProtectedDatabaseResource().Spec
	getCalls := 0
	changeCalls := 0
	updateCalled := false
	workRequestCalls := 0

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			if getCalls == 1 {
				return recoverysdk.GetProtectedDatabaseResponse{
					ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, currentSpec, recoverysdk.LifecycleStateActive),
				}, nil
			}
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		changeSubscriptionFn: func(_ context.Context, req recoverysdk.ChangeProtectedDatabaseSubscriptionRequest) (recoverysdk.ChangeProtectedDatabaseSubscriptionResponse, error) {
			changeCalls++
			requireStringPtr(t, "ChangeProtectedDatabaseSubscriptionRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			requireStringPtr(t, "ChangeProtectedDatabaseSubscriptionDetails.SubscriptionId", req.SubscriptionId, resource.Spec.SubscriptionId)
			if req.IsDefault != nil {
				t.Fatalf("ChangeProtectedDatabaseSubscriptionDetails.IsDefault = %v, want nil", *req.IsDefault)
			}
			return recoverysdk.ChangeProtectedDatabaseSubscriptionResponse{
				OpcWorkRequestId: common.String("wr-subscription"),
				OpcRequestId:     common.String("opc-subscription"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-subscription")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-subscription",
					recoverysdk.OperationTypeUpdateProtectedDatabase,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeUpdated,
					testProtectedDatabaseID,
				),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if changeCalls != 1 {
		t.Fatalf("ChangeProtectedDatabaseSubscription() calls = %d, want 1", changeCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called for subscription change")
	}
	if got := resource.Status.SubscriptionId; got != resource.Spec.SubscriptionId {
		t.Fatalf("status.subscriptionId = %q, want %q", got, resource.Spec.SubscriptionId)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-subscription" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-subscription", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateUpdatesPasswordOnlyDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	markProtectedDatabaseCreatedWithLegacyPasswordHash(resource)
	resource.Spec.Password = testProtectedDatabaseNewPassword
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0
	credentialClient := newFakeProtectedDatabaseCredentialClient()

	client := newTestProtectedDatabaseClientWithCredentialClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalls++
			requireProtectedDatabasePasswordOnlyUpdateRequest(t, req, resource.Spec.Password)
			return recoverysdk.UpdateProtectedDatabaseResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: protectedDatabaseUpdateWorkRequestFn(t, &workRequestCalls),
	}, credentialClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateProtectedDatabase() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want current read and work-request follow-up", getCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	requireProtectedDatabaseSecretPasswordHash(t, credentialClient, resource, testProtectedDatabaseNewPassword)
	requireProtectedDatabaseStatusHasNoPasswordMaterial(t, resource)
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateKeepsOldPasswordHashUntilWorkRequestSucceeds(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	markProtectedDatabaseCreatedWithLegacyPasswordHash(resource)
	resource.Spec.Password = testProtectedDatabaseNewPassword
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0
	credentialClient := newFakeProtectedDatabaseCredentialClient()

	client := newTestProtectedDatabaseClientWithCredentialClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, makeProtectedDatabaseResource().Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalls++
			requireProtectedDatabasePasswordOnlyUpdateRequest(t, req, resource.Spec.Password)
			return recoverysdk.UpdateProtectedDatabaseResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-update")
			status := recoverysdk.OperationStatusInProgress
			if workRequestCalls == 2 {
				status = recoverysdk.OperationStatusFailed
			}
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-update",
					recoverysdk.OperationTypeUpdateProtectedDatabase,
					status,
					recoverysdk.ActionTypeUpdated,
					testProtectedDatabaseID,
				),
			}, nil
		},
	}, credentialClient)

	requireProtectedDatabasePendingPasswordUpdate(t, client, credentialClient, resource, &updateCalls, "first", 1)
	requireProtectedDatabaseFailedPasswordUpdate(t, client, credentialClient, resource, &updateCalls)
	requireProtectedDatabasePendingPasswordUpdate(t, client, credentialClient, resource, &updateCalls, "third", 2)
	if getCalls != 2 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want current read for initial update and retry", getCalls)
	}
	if workRequestCalls != 3 {
		t.Fatalf("GetWorkRequest() calls = %d, want pending, failed, and retried pending checks", workRequestCalls)
	}
}

func requireProtectedDatabasePendingPasswordUpdate(
	t *testing.T,
	client ProtectedDatabaseServiceClient,
	credentialClient *fakeProtectedDatabaseCredentialClient,
	resource *recoveryv1beta1.ProtectedDatabase,
	updateCalls *int,
	label string,
	wantUpdateCalls int,
) {
	t.Helper()
	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("%s CreateOrUpdate() error = %v", label, err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("%s CreateOrUpdate() response = %#v, want successful requeue while update work request is pending", label, response)
	}
	if *updateCalls != wantUpdateCalls {
		t.Fatalf("UpdateProtectedDatabase() calls after %s reconcile = %d, want %d", label, *updateCalls, wantUpdateCalls)
	}
	requireProtectedDatabaseCurrentAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update", shared.OSOKAsyncClassPending)
	requireProtectedDatabaseSecretPasswordHash(t, credentialClient, resource, testProtectedDatabasePassword)
	requireProtectedDatabaseStatusHasNoPasswordMaterial(t, resource)
}

func requireProtectedDatabaseFailedPasswordUpdate(
	t *testing.T,
	client ProtectedDatabaseServiceClient,
	credentialClient *fakeProtectedDatabaseCredentialClient,
	resource *recoveryv1beta1.ProtectedDatabase,
	updateCalls *int,
) {
	t.Helper()
	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err == nil {
		t.Fatal("second CreateOrUpdate() error = nil, want failed work request error")
	}
	if response.IsSuccessful {
		t.Fatalf("second CreateOrUpdate() response = %#v, want unsuccessful response after failed work request", response)
	}
	if *updateCalls != 1 {
		t.Fatalf("UpdateProtectedDatabase() calls after failed work request = %d, want 1 before retry", *updateCalls)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want cleared failed password update tracker before retry", resource.Status.OsokStatus.Async.Current)
	}
	requireProtectedDatabaseSecretPasswordHash(t, credentialClient, resource, testProtectedDatabasePassword)
	requireProtectedDatabaseStatusHasNoPasswordMaterial(t, resource)
	requireLastCondition(t, resource, shared.Failed)
}

func TestProtectedDatabaseCreateOrUpdateUpdatesPasswordWithMutableDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	markProtectedDatabaseCreatedWithLegacyPasswordHash(resource)
	resource.Spec.Password = testProtectedDatabaseNewPassword
	resource.Spec.DisplayName = "updated-protected-db"
	currentSpec := makeProtectedDatabaseResource().Spec
	getCalls := 0
	updateCalls := 0
	workRequestCalls := 0
	credentialClient := newFakeProtectedDatabaseCredentialClient()

	client := newTestProtectedDatabaseClientWithCredentialClient(&fakeProtectedDatabaseOCIClient{
		getFn:         protectedDatabaseMutableReadFn(t, resource, currentSpec, &getCalls),
		workRequestFn: protectedDatabaseUpdateWorkRequestFn(t, &workRequestCalls),
		updateFn: func(_ context.Context, req recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			requireStringPtr(t, "UpdateProtectedDatabaseDetails.Password", req.Password, testProtectedDatabaseNewPassword)
			requireStringPtr(t, "UpdateProtectedDatabaseDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			return recoverysdk.UpdateProtectedDatabaseResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
	}, credentialClient)

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateProtectedDatabase() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want current read and work-request follow-up", getCalls)
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	requireProtectedDatabaseSecretPasswordHash(t, credentialClient, resource, testProtectedDatabaseNewPassword)
	requireProtectedDatabaseStatusHasNoPasswordMaterial(t, resource)
	requireLastCondition(t, resource, shared.Active)
}

func TestProtectedDatabaseCreateOrUpdateRejectsMissingPasswordSecretBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	now := metav1.Now()
	resource.Status.OsokStatus.CreatedAt = &now
	resource.Spec.DisplayName = "updated-protected-db"
	updateCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, makeProtectedDatabaseResource().Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing password tracking secret rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called after missing password tracking secret rejection")
	}
	if !strings.Contains(err.Error(), "password tracking secret is missing") {
		t.Fatalf("CreateOrUpdate() error = %v, want missing password tracking secret rejection", err)
	}
	requireProtectedDatabaseStatusHasNoPasswordMaterial(t, resource)
	requireLastCondition(t, resource, shared.Failed)
}

func protectedDatabaseMutableReadFn(
	t *testing.T,
	resource *recoveryv1beta1.ProtectedDatabase,
	currentSpec recoveryv1beta1.ProtectedDatabaseSpec,
	getCalls *int,
) func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
	t.Helper()
	return func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
		*getCalls++
		requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
		if *getCalls == 1 {
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, currentSpec, recoverysdk.LifecycleStateActive),
			}, nil
		}
		return recoverysdk.GetProtectedDatabaseResponse{
			ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
		}, nil
	}
}

func protectedDatabaseMutableUpdateFn(
	t *testing.T,
	resource *recoveryv1beta1.ProtectedDatabase,
	updateCalls *int,
) func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
	t.Helper()
	return func(_ context.Context, req recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
		*updateCalls++
		requireProtectedDatabaseMutableUpdateRequest(t, req, resource)
		return recoverysdk.UpdateProtectedDatabaseResponse{
			OpcWorkRequestId: common.String("wr-update"),
			OpcRequestId:     common.String("opc-update"),
		}, nil
	}
}

func protectedDatabaseUpdateWorkRequestFn(
	t *testing.T,
	workRequestCalls *int,
) func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
		*workRequestCalls++
		requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-update")
		return recoverysdk.GetWorkRequestResponse{
			WorkRequest: makeProtectedDatabaseWorkRequest(
				"wr-update",
				recoverysdk.OperationTypeUpdateProtectedDatabase,
				recoverysdk.OperationStatusSucceeded,
				recoverysdk.ActionTypeUpdated,
				testProtectedDatabaseID,
			),
		}, nil
	}
}

func requireProtectedDatabaseMutableUpdateRequest(
	t *testing.T,
	req recoverysdk.UpdateProtectedDatabaseRequest,
	resource *recoveryv1beta1.ProtectedDatabase,
) {
	t.Helper()
	requireStringPtr(t, "UpdateProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
	if req.Password != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.Password = set, want nil when password hash matches")
	}
	requireStringPtr(t, "UpdateProtectedDatabaseDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	if req.DatabaseSize != recoverysdk.DatabaseSizesL {
		t.Fatalf("UpdateProtectedDatabaseDetails.DatabaseSize = %q, want L", req.DatabaseSize)
	}
	requireIntPtr(t, "UpdateProtectedDatabaseDetails.DatabaseSizeInGBs", req.DatabaseSizeInGBs, 2048)
	requireStringPtr(t, "UpdateProtectedDatabaseDetails.ProtectionPolicyId", req.ProtectionPolicyId, resource.Spec.ProtectionPolicyId)
	if got := len(req.RecoveryServiceSubnets); got != 1 {
		t.Fatalf("UpdateProtectedDatabaseDetails.RecoveryServiceSubnets length = %d, want 1", got)
	}
	requireStringPtr(t, "UpdateProtectedDatabaseDetails.RecoveryServiceSubnets[0]", req.RecoveryServiceSubnets[0].RecoveryServiceSubnetId, testProtectedDatabaseSubnetID2)
	requireBoolPtr(t, "UpdateProtectedDatabaseDetails.IsRedoLogsShipped", req.IsRedoLogsShipped, false)
	if got := req.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateProtectedDatabaseDetails.FreeformTags[env] = %q, want prod", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
		t.Fatalf("UpdateProtectedDatabaseDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
	}
}

func requireProtectedDatabasePasswordOnlyUpdateRequest(
	t *testing.T,
	req recoverysdk.UpdateProtectedDatabaseRequest,
	wantPassword string,
) {
	t.Helper()
	requireStringPtr(t, "UpdateProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
	requireStringPtr(t, "UpdateProtectedDatabaseDetails.Password", req.Password, wantPassword)
	if req.DisplayName != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.DisplayName = %q, want nil", *req.DisplayName)
	}
	if req.DatabaseSize != "" {
		t.Fatalf("UpdateProtectedDatabaseDetails.DatabaseSize = %q, want empty", req.DatabaseSize)
	}
	if req.DatabaseSizeInGBs != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.DatabaseSizeInGBs = %d, want nil", *req.DatabaseSizeInGBs)
	}
	if req.ProtectionPolicyId != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.ProtectionPolicyId = %q, want nil", *req.ProtectionPolicyId)
	}
	if req.RecoveryServiceSubnets != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.RecoveryServiceSubnets = %#v, want nil", req.RecoveryServiceSubnets)
	}
	if req.IsRedoLogsShipped != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.IsRedoLogsShipped = %v, want nil", *req.IsRedoLogsShipped)
	}
	if req.FreeformTags != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.FreeformTags = %#v, want nil", req.FreeformTags)
	}
	if req.DefinedTags != nil {
		t.Fatalf("UpdateProtectedDatabaseDetails.DefinedTags = %#v, want nil", req.DefinedTags)
	}
}

func TestProtectedDatabaseCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	resource.Spec.DbUniqueName = "CDB02"
	currentSpec := makeProtectedDatabaseResource().Spec
	updateCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, currentSpec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
			updateCalled = true
			return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateProtectedDatabase() called after create-only dbUniqueName drift")
	}
	if !strings.Contains(err.Error(), "dbUniqueName") {
		t.Fatalf("CreateOrUpdate() error = %v, want dbUniqueName force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestProtectedDatabaseCreateOrUpdateRejectsExplicitOptionalCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mutateSpec  func(*recoveryv1beta1.ProtectedDatabaseSpec)
		wantMessage string
	}{
		{
			name: "databaseId",
			mutateSpec: func(spec *recoveryv1beta1.ProtectedDatabaseSpec) {
				spec.DatabaseId = "ocid1.database.oc1..replacement"
			},
			wantMessage: "databaseId",
		},
		{
			name: "changeRate",
			mutateSpec: func(spec *recoveryv1beta1.ProtectedDatabaseSpec) {
				spec.ChangeRate = 0.75
			},
			wantMessage: "changeRate",
		},
		{
			name: "compressionRatio",
			mutateSpec: func(spec *recoveryv1beta1.ProtectedDatabaseSpec) {
				spec.CompressionRatio = 2.5
			},
			wantMessage: "compressionRatio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedProtectedDatabaseResource()
			currentSpec := resource.Spec
			tt.mutateSpec(&resource.Spec)
			updateCalled := false

			client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
				getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
					requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
					return recoverysdk.GetProtectedDatabaseResponse{
						ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, currentSpec, recoverysdk.LifecycleStateActive),
					}, nil
				},
				updateFn: func(context.Context, recoverysdk.UpdateProtectedDatabaseRequest) (recoverysdk.UpdateProtectedDatabaseResponse, error) {
					updateCalled = true
					return recoverysdk.UpdateProtectedDatabaseResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, makeProtectedDatabaseRequest(resource))
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want optional create-only drift rejection")
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() successful = true, want false")
			}
			if updateCalled {
				t.Fatalf("UpdateProtectedDatabase() called after create-only %s drift", tt.wantMessage)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("CreateOrUpdate() error = %v, want %s force-new rejection", err, tt.wantMessage)
			}
			requireLastCondition(t, resource, shared.Failed)
		})
	}
}

func TestProtectedDatabaseDeleteKeepsFinalizerUntilLifecycleConfirmsDeleted(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	getCalls := 0

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			state := recoverysdk.LifecycleStateActive
			if getCalls == 3 {
				state = recoverysdk.LifecycleStateDeleteScheduled
			}
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
			requireStringPtr(t, "DeleteProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			return recoverysdk.DeleteProtectedDatabaseResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-delete",
					recoverysdk.OperationTypeDeleteProtectedDatabase,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeDeleted,
					testProtectedDatabaseID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle is DELETE_SCHEDULED")
	}
	if getCalls != 3 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want pre-read, generated confirm read, and work-request follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseDelete || current.RawStatus != string(recoverysdk.LifecycleStateDeleteScheduled) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle delete pending DELETE_SCHEDULED", current)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestProtectedDatabaseDeleteConfirmsNotFoundAfterWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	getCalls := 0

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProtectedDatabaseRequest.ProtectedDatabaseId", req.ProtectedDatabaseId, testProtectedDatabaseID)
			if getCalls == 3 {
				return recoverysdk.GetProtectedDatabaseResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
			}
			return recoverysdk.GetProtectedDatabaseResponse{
				ProtectedDatabase: makeSDKProtectedDatabase(testProtectedDatabaseID, resource.Spec, recoverysdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
			return recoverysdk.DeleteProtectedDatabaseResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		workRequestFn: func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-delete",
					recoverysdk.OperationTypeDeleteProtectedDatabase,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeDeleted,
					testProtectedDatabaseID,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound readback")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestProtectedDatabaseDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	deleteCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			return recoverysdk.GetProtectedDatabaseResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
			deleteCalled = true
			return recoverysdk.DeleteProtectedDatabaseResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if deleteCalled {
		t.Fatal("DeleteProtectedDatabase() called after ambiguous pre-delete read")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestProtectedDatabaseDeleteResumesTrackedWorkRequestBeforeAuthShapedReadback(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	getCalls := 0
	workRequestCalls := 0
	deleteCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			getCalls++
			if workRequestCalls == 0 {
				t.Fatal("GetProtectedDatabase() called before resuming the tracked delete work request")
			}
			return recoverysdk.GetProtectedDatabaseResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-delete")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-delete",
					recoverysdk.OperationTypeDeleteProtectedDatabase,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeDeleted,
					testProtectedDatabaseID,
				),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
			deleteCalled = true
			return recoverysdk.DeleteProtectedDatabaseResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped readback error after resuming delete work request")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetProtectedDatabase() calls = %d, want post-work-request readback only", getCalls)
	}
	if deleteCalled {
		t.Fatal("DeleteProtectedDatabase() called while resuming tracked delete work request")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestProtectedDatabaseDeleteWaitsForPendingCreateWorkRequestWithoutRecordedOCID(t *testing.T) {
	t.Parallel()

	resource := makeProtectedDatabaseResource()
	seedProtectedDatabasePendingWrite(resource, shared.OSOKAsyncPhaseCreate, "wr-create")
	workRequestCalls := 0
	deleteCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			t.Fatal("GetProtectedDatabase() called while create work request is still pending")
			return recoverysdk.GetProtectedDatabaseResponse{}, nil
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-create")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-create",
					recoverysdk.OperationTypeCreateProtectedDatabase,
					recoverysdk.OperationStatusInProgress,
					recoverysdk.ActionTypeCreated,
					testProtectedDatabaseID,
				),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
			deleteCalled = true
			return recoverysdk.DeleteProtectedDatabaseResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while create work request is pending")
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if deleteCalled {
		t.Fatal("DeleteProtectedDatabase() called while create work request is still pending")
	}
	if got := resource.Status.Id; got != "" {
		t.Fatalf("status.id = %q, want empty while create work request is pending", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "" {
		t.Fatalf("status.status.ocid = %q, want empty while create work request is pending", got)
	}
	requireProtectedDatabaseCurrentAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	requireLastCondition(t, resource, shared.Provisioning)
}

func TestProtectedDatabaseDeleteWaitsForPendingUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedProtectedDatabaseResource()
	seedProtectedDatabasePendingWrite(resource, shared.OSOKAsyncPhaseUpdate, "wr-update")
	workRequestCalls := 0
	deleteCalled := false

	client := newTestProtectedDatabaseClient(&fakeProtectedDatabaseOCIClient{
		getFn: func(context.Context, recoverysdk.GetProtectedDatabaseRequest) (recoverysdk.GetProtectedDatabaseResponse, error) {
			t.Fatal("GetProtectedDatabase() called while update work request is still pending")
			return recoverysdk.GetProtectedDatabaseResponse{}, nil
		},
		workRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", req.WorkRequestId, "wr-update")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectedDatabaseWorkRequest(
					"wr-update",
					recoverysdk.OperationTypeUpdateProtectedDatabase,
					recoverysdk.OperationStatusInProgress,
					recoverysdk.ActionTypeUpdated,
					testProtectedDatabaseID,
				),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectedDatabaseRequest) (recoverysdk.DeleteProtectedDatabaseResponse, error) {
			deleteCalled = true
			return recoverysdk.DeleteProtectedDatabaseResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while update work request is pending")
	}
	if workRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", workRequestCalls)
	}
	if deleteCalled {
		t.Fatal("DeleteProtectedDatabase() called while update work request is still pending")
	}
	requireProtectedDatabaseCurrentAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update", shared.OSOKAsyncClassPending)
	requireLastCondition(t, resource, shared.Updating)
}

func seedProtectedDatabasePendingWrite(
	resource *recoveryv1beta1.ProtectedDatabase,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func requireProtectedDatabaseCreateRequest(
	t *testing.T,
	req recoverysdk.CreateProtectedDatabaseRequest,
	resource *recoveryv1beta1.ProtectedDatabase,
) {
	t.Helper()
	requireStringPtr(t, "CreateProtectedDatabaseDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateProtectedDatabaseDetails.DbUniqueName", req.DbUniqueName, resource.Spec.DbUniqueName)
	requireStringPtr(t, "CreateProtectedDatabaseDetails.Password", req.Password, resource.Spec.Password)
	requireStringPtr(t, "CreateProtectedDatabaseDetails.ProtectionPolicyId", req.ProtectionPolicyId, resource.Spec.ProtectionPolicyId)
	requireStringPtr(t, "CreateProtectedDatabaseDetails.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
	if got := req.DatabaseSize; got != recoverysdk.DatabaseSizesM {
		t.Fatalf("CreateProtectedDatabaseDetails.DatabaseSize = %q, want M", got)
	}
	requireStringPtr(t, "CreateProtectedDatabaseDetails.DatabaseId", req.DatabaseId, resource.Spec.DatabaseId)
	requireFloat64Ptr(t, "CreateProtectedDatabaseDetails.ChangeRate", req.ChangeRate, resource.Spec.ChangeRate)
	requireFloat64Ptr(t, "CreateProtectedDatabaseDetails.CompressionRatio", req.CompressionRatio, resource.Spec.CompressionRatio)
	requireBoolPtr(t, "CreateProtectedDatabaseDetails.IsRedoLogsShipped", req.IsRedoLogsShipped, true)
	requireStringPtr(t, "CreateProtectedDatabaseDetails.SubscriptionId", req.SubscriptionId, resource.Spec.SubscriptionId)
	if got := len(req.RecoveryServiceSubnets); got != 1 {
		t.Fatalf("CreateProtectedDatabaseDetails.RecoveryServiceSubnets length = %d, want 1", got)
	}
	requireStringPtr(t, "CreateProtectedDatabaseDetails.RecoveryServiceSubnets[0]", req.RecoveryServiceSubnets[0].RecoveryServiceSubnetId, testProtectedDatabaseSubnetID)
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateProtectedDatabaseDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateProtectedDatabaseDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
}

func requireLastCondition(t *testing.T, resource *recoveryv1beta1.ProtectedDatabase, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want last condition %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}

func requireProtectedDatabaseCurrentAsync(
	t *testing.T,
	resource *recoveryv1beta1.ProtectedDatabase,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.status.async.current = nil, want phase %s work request %s class %s", wantPhase, wantWorkRequestID, wantClass)
	}
	if current.Phase != wantPhase || current.WorkRequestID != wantWorkRequestID || current.NormalizedClass != wantClass {
		t.Fatalf(
			"status.status.async.current = phase %s workRequestID %q class %s, want phase %s workRequestID %q class %s",
			current.Phase,
			current.WorkRequestID,
			current.NormalizedClass,
			wantPhase,
			wantWorkRequestID,
			wantClass,
		)
	}
}

func requireProtectedDatabaseSecretPasswordHash(
	t *testing.T,
	credentialClient *fakeProtectedDatabaseCredentialClient,
	resource *recoveryv1beta1.ProtectedDatabase,
	wantPassword string,
) {
	t.Helper()
	data, err := credentialClient.GetSecret(
		context.Background(),
		protectedDatabasePasswordStateSecretName(resource),
		resource.Namespace,
	)
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if !servicemanager.SecretOwnedBy(data, protectedDatabasePasswordStateSecretKind, protectedDatabasePasswordStateOwnerName(resource)) {
		t.Fatalf("password tracking secret is not owned by ProtectedDatabase %q", resource.Name)
	}
	recorded, ok := protectedDatabasePasswordHashFromBytes(data[protectedDatabasePasswordStateHashKey])
	if !ok {
		t.Fatal("password tracking secret hash missing")
	}
	if want := protectedDatabasePasswordHash(wantPassword); recorded != want {
		t.Fatalf("password tracking secret hash = %q, want %q", recorded, want)
	}
}

func requireProtectedDatabaseStatusHasNoPasswordMaterial(t *testing.T, resource *recoveryv1beta1.ProtectedDatabase) {
	t.Helper()
	message := resource.Status.OsokStatus.Message
	if strings.Contains(message, protectedDatabaseLegacyPasswordHashKey) {
		t.Fatalf("status.status.message contains password fingerprint marker: %q", message)
	}
	for _, password := range []string{testProtectedDatabasePassword, testProtectedDatabaseNewPassword} {
		if strings.Contains(message, password) {
			t.Fatalf("status.status.message contains raw password material: %q", message)
		}
	}
	for _, hash := range []string{
		protectedDatabasePasswordHash(testProtectedDatabasePassword),
		protectedDatabasePasswordHash(testProtectedDatabaseNewPassword),
	} {
		if strings.Contains(message, hash) {
			t.Fatalf("status.status.message contains password-derived hash: %q", message)
		}
	}
}

func requireStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireIntPtr(t *testing.T, label string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", label, *got, want)
	}
}

func requireFloat64Ptr(t *testing.T, label string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %f", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %f, want %f", label, *got, want)
	}
}

func requireBoolPtr(t *testing.T, label string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", label, *got, want)
	}
}
