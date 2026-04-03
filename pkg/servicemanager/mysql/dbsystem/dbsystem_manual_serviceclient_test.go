//go:build legacyservicemanager

package dbsystem

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDbSystemRuntimeOCIClient struct {
	listRequest  mysqlsdk.ListDbSystemsRequest
	listResponse mysqlsdk.ListDbSystemsResponse
	listErr      error

	createRequest  mysqlsdk.CreateDbSystemRequest
	createResponse mysqlsdk.CreateDbSystemResponse
	createErr      error
	createCalls    int
}

func (f *fakeDbSystemRuntimeOCIClient) CreateDbSystem(_ context.Context, request mysqlsdk.CreateDbSystemRequest) (mysqlsdk.CreateDbSystemResponse, error) {
	f.createCalls++
	f.createRequest = request
	return f.createResponse, f.createErr
}

func (f *fakeDbSystemRuntimeOCIClient) ListDbSystems(_ context.Context, request mysqlsdk.ListDbSystemsRequest) (mysqlsdk.ListDbSystemsResponse, error) {
	f.listRequest = request
	return f.listResponse, f.listErr
}

type fakeDbSystemDelegate struct {
	createOrUpdateCalls int
	deleteCalls         int
	receivedOcid        string
	response            servicemanager.OSOKResponse
	err                 error
}

func (f *fakeDbSystemDelegate) CreateOrUpdate(_ context.Context, resource *mysqlv1beta1.DbSystem, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	f.createOrUpdateCalls++
	if resource != nil {
		f.receivedOcid = string(resource.Status.OsokStatus.Ocid)
	}
	return f.response, f.err
}

func (f *fakeDbSystemDelegate) Delete(_ context.Context, _ *mysqlv1beta1.DbSystem) (bool, error) {
	f.deleteCalls++
	return true, f.err
}

func TestBuildDbSystemCreateRequestOmitsZeroValueNestedStructs(t *testing.T) {
	t.Parallel()

	resource := &mysqlv1beta1.DbSystem{
		Spec: mysqlv1beta1.DbSystemSpec{
			CompartmentId:        "ocid1.compartment.oc1..example",
			DisplayName:          "mysql-dbsystem-sample",
			ShapeName:            "MySQL.VM.Standard.E3.1.8GB",
			SubnetId:             "ocid1.subnet.oc1..example",
			AvailabilityDomain:   "qqZb:US-ASHBURN-AD-1",
			AdminUsername:        usernameSecretSource("admin-secret"),
			AdminPassword:        passwordSecretSource("admin-secret"),
			Description:          "OSOK mysql DbSystem e2e sample",
			DataStorageSizeInGBs: 50,
		},
	}

	request, err := buildDbSystemCreateRequest(
		context.Background(),
		&fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("adminuser"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
		resource,
		"default",
	)
	if err != nil {
		t.Fatalf("buildDbSystemCreateRequest() error = %v", err)
	}

	if request.CreateDbSystemDetails.AdminUsername == nil || *request.CreateDbSystemDetails.AdminUsername != "adminuser" {
		t.Fatalf("AdminUsername = %v, want resolved username", request.CreateDbSystemDetails.AdminUsername)
	}
	if request.CreateDbSystemDetails.AdminPassword == nil || *request.CreateDbSystemDetails.AdminPassword != "ChangeMe123!!" {
		t.Fatalf("AdminPassword = %v, want resolved password", request.CreateDbSystemDetails.AdminPassword)
	}

	payload, err := json.Marshal(request.CreateDbSystemDetails)
	if err != nil {
		t.Fatalf("json.Marshal(CreateDbSystemDetails) error = %v", err)
	}

	for _, field := range []string{"backupPolicy", "maintenance", "secureConnections", "deletionPolicy", "source"} {
		if strings.Contains(string(payload), "\""+field+"\":{") {
			t.Fatalf("CreateDbSystemDetails JSON = %s, want %q to stay nil instead of a populated object", string(payload), field)
		}
	}
}

func TestManualDbSystemServiceClientBindsReusableExistingDbSystemBeforeCreate(t *testing.T) {
	t.Parallel()

	ociClient := &fakeDbSystemRuntimeOCIClient{
		listResponse: mysqlsdk.ListDbSystemsResponse{
			Items: []mysqlsdk.DbSystemSummary{
				{
					Id:             common.String("ocid1.mysqldbsystem.oc1..existing"),
					DisplayName:    common.String("mysql-dbsystem-sample"),
					CompartmentId:  common.String("ocid1.compartment.oc1..example"),
					LifecycleState: mysqlsdk.DbSystemLifecycleStateActive,
				},
			},
		},
	}
	delegate := &fakeDbSystemDelegate{
		response: servicemanager.OSOKResponse{IsSuccessful: true},
	}
	client := manualDbSystemServiceClient{
		delegate:  delegate,
		ociClient: ociClient,
	}
	resource := &mysqlv1beta1.DbSystem{
		Spec: mysqlv1beta1.DbSystemSpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "mysql-dbsystem-sample",
			ShapeName:     "MySQL.VM.Standard.E3.1.8GB",
			SubnetId:      "ocid1.subnet.oc1..example",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want success", response)
	}
	if ociClient.createCalls != 0 {
		t.Fatalf("CreateDbSystem() calls = %d, want 0 when reusable match exists", ociClient.createCalls)
	}
	if delegate.createOrUpdateCalls != 1 {
		t.Fatalf("delegate CreateOrUpdate() calls = %d, want 1", delegate.createOrUpdateCalls)
	}
	if delegate.receivedOcid != "ocid1.mysqldbsystem.oc1..existing" {
		t.Fatalf("delegate received ocid = %q, want existing resource id", delegate.receivedOcid)
	}
}

func TestManualDbSystemServiceClientSkipsDeletingMatchAndCreates(t *testing.T) {
	t.Parallel()

	ociClient := &fakeDbSystemRuntimeOCIClient{
		listResponse: mysqlsdk.ListDbSystemsResponse{
			Items: []mysqlsdk.DbSystemSummary{
				{
					Id:             common.String("ocid1.mysqldbsystem.oc1..deleting"),
					DisplayName:    common.String("mysql-dbsystem-sample"),
					CompartmentId:  common.String("ocid1.compartment.oc1..example"),
					LifecycleState: mysqlsdk.DbSystemLifecycleStateDeleting,
				},
			},
		},
		createResponse: mysqlsdk.CreateDbSystemResponse{
			DbSystem: mysqlsdk.DbSystem{
				Id: common.String("ocid1.mysqldbsystem.oc1..created"),
			},
		},
	}
	delegate := &fakeDbSystemDelegate{
		response: servicemanager.OSOKResponse{IsSuccessful: true},
	}
	client := manualDbSystemServiceClient{
		delegate:  delegate,
		ociClient: ociClient,
		credentialClient: &fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("adminuser"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
	}
	resource := &mysqlv1beta1.DbSystem{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: mysqlv1beta1.DbSystemSpec{
			CompartmentId:        "ocid1.compartment.oc1..example",
			DisplayName:          "mysql-dbsystem-sample",
			ShapeName:            "MySQL.VM.Standard.E3.1.8GB",
			SubnetId:             "ocid1.subnet.oc1..example",
			AvailabilityDomain:   "qqZb:US-ASHBURN-AD-1",
			AdminUsername:        usernameSecretSource("admin-secret"),
			AdminPassword:        passwordSecretSource("admin-secret"),
			DataStorageSizeInGBs: 50,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want success", response)
	}
	if ociClient.createCalls != 1 {
		t.Fatalf("CreateDbSystem() calls = %d, want 1 when deleting match should not bind", ociClient.createCalls)
	}
	if delegate.receivedOcid != "ocid1.mysqldbsystem.oc1..created" {
		t.Fatalf("delegate received ocid = %q, want created resource id", delegate.receivedOcid)
	}
}
