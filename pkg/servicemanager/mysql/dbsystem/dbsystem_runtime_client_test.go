/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeDbSystemAvailabilityDomainClient struct {
	response               identity.ListAvailabilityDomainsResponse
	err                    error
	getCompartmentResponse identity.GetCompartmentResponse
	getCompartmentErr      error
	listRequests           []identity.ListAvailabilityDomainsRequest
	getCompartmentRequests []identity.GetCompartmentRequest
}

func (f *fakeDbSystemAvailabilityDomainClient) ListAvailabilityDomains(_ context.Context, request identity.ListAvailabilityDomainsRequest) (identity.ListAvailabilityDomainsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	return f.response, f.err
}

func (f *fakeDbSystemAvailabilityDomainClient) GetCompartment(_ context.Context, request identity.GetCompartmentRequest) (identity.GetCompartmentResponse, error) {
	f.getCompartmentRequests = append(f.getCompartmentRequests, request)
	return f.getCompartmentResponse, f.getCompartmentErr
}

func TestBuildDbSystemCreateDetailsPreservesExplicitAvailabilityDomainAlias(t *testing.T) {
	restoreClient := newDbSystemAvailabilityDomainClient
	newDbSystemAvailabilityDomainClient = func(common.ConfigurationProvider) (dbSystemAvailabilityDomainClient, error) {
		return &fakeDbSystemAvailabilityDomainClient{
			response: identity.ListAvailabilityDomainsResponse{
				Items: []identity.AvailabilityDomain{
					{Name: common.String("ypKW:US-ASHBURN-AD-1")},
					{Name: common.String("ypKW:US-ASHBURN-AD-2")},
				},
			},
			getCompartmentResponse: identity.GetCompartmentResponse{
				Compartment: identity.Compartment{
					CompartmentId: common.String("ocid1.tenancy.oc1..example"),
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		newDbSystemAvailabilityDomainClient = restoreClient
	})

	privateKey := testRSAKey(t)
	details, err := buildDbSystemCreateDetails(
		context.Background(),
		testConfigurationProvider{privateKey: privateKey},
		&fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("adminuser"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
		&mysqlv1beta1.DbSystem{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
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
		},
		"default",
	)
	if err != nil {
		t.Fatalf("buildDbSystemCreateDetails() error = %v", err)
	}
	if details.AvailabilityDomain == nil || *details.AvailabilityDomain != "qqZb:US-ASHBURN-AD-1" {
		t.Fatalf("AvailabilityDomain = %v, want explicit alias preserved", details.AvailabilityDomain)
	}
	if details.AdminUsername == nil || *details.AdminUsername != "adminuser" {
		t.Fatalf("AdminUsername = %v, want resolved username", details.AdminUsername)
	}
	if details.AdminPassword == nil || *details.AdminPassword != "ChangeMe123!!" {
		t.Fatalf("AdminPassword = %v, want resolved password", details.AdminPassword)
	}

	payload, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("json.Marshal(CreateDbSystemDetails) error = %v", err)
	}
	for _, field := range []string{"backupPolicy", "maintenance", "secureConnections", "deletionPolicy", "source"} {
		if strings.Contains(string(payload), "\""+field+"\":{") {
			t.Fatalf("CreateDbSystemDetails JSON = %s, want %q to stay nil instead of a populated object", string(payload), field)
		}
	}
}

func TestBuildDbSystemCreateDetailsRejectsUnknownAvailabilityDomain(t *testing.T) {
	restoreClient := newDbSystemAvailabilityDomainClient
	newDbSystemAvailabilityDomainClient = func(common.ConfigurationProvider) (dbSystemAvailabilityDomainClient, error) {
		return &fakeDbSystemAvailabilityDomainClient{
			response: identity.ListAvailabilityDomainsResponse{
				Items: []identity.AvailabilityDomain{
					{Name: common.String("ypKW:US-ASHBURN-AD-1")},
				},
			},
			getCompartmentResponse: identity.GetCompartmentResponse{
				Compartment: identity.Compartment{
					CompartmentId: common.String("ocid1.tenancy.oc1..example"),
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		newDbSystemAvailabilityDomainClient = restoreClient
	})

	privateKey := testRSAKey(t)
	_, err := buildDbSystemCreateDetails(
		context.Background(),
		testConfigurationProvider{privateKey: privateKey},
		&fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("adminuser"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
		&mysqlv1beta1.DbSystem{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: mysqlv1beta1.DbSystemSpec{
				CompartmentId:      "ocid1.compartment.oc1..example",
				DisplayName:        "mysql-dbsystem-sample",
				ShapeName:          "MySQL.VM.Standard.E3.1.8GB",
				SubnetId:           "ocid1.subnet.oc1..example",
				AvailabilityDomain: "qqZb:US-ASHBURN-AD-9",
				AdminUsername:      usernameSecretSource("admin-secret"),
				AdminPassword:      passwordSecretSource("admin-secret"),
			},
		},
		"default",
	)
	if err == nil {
		t.Fatal("buildDbSystemCreateDetails() error = nil, want explicit availability domain mismatch")
	}
	if !strings.Contains(err.Error(), "availabilityDomain") {
		t.Fatalf("buildDbSystemCreateDetails() error = %v, want availabilityDomain context", err)
	}
}

func TestBuildDbSystemCreateDetailsPreservesStandaloneFalse(t *testing.T) {
	t.Parallel()

	details, err := buildDbSystemCreateDetails(
		context.Background(),
		nil,
		&fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("admin"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
		&mysqlv1beta1.DbSystem{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: mysqlv1beta1.DbSystemSpec{
				CompartmentId:        "ocid1.compartment.oc1..example",
				DisplayName:          "mysql-dbsystem-sample",
				ShapeName:            "MySQL.VM.Standard.E3.1.8GB",
				SubnetId:             "ocid1.subnet.oc1..example",
				IsHighlyAvailable:    false,
				AdminUsername:        usernameSecretSource("admin-secret"),
				AdminPassword:        passwordSecretSource("admin-secret"),
				Description:          "OSOK mysql DbSystem e2e sample",
				DataStorageSizeInGBs: 100,
				Port:                 3306,
				PortX:                33060,
			},
		},
		"default",
	)
	if err != nil {
		t.Fatalf("buildDbSystemCreateDetails() error = %v", err)
	}
	if details.IsHighlyAvailable == nil {
		t.Fatal("IsHighlyAvailable = nil, want explicit false pointer")
	}
	if *details.IsHighlyAvailable {
		t.Fatalf("IsHighlyAvailable = %t, want false", *details.IsHighlyAvailable)
	}
}

func TestBuildDbSystemCreateDetailsExpandsSuffixOnlyAvailabilityDomainAgainstCompartmentTenancy(t *testing.T) {
	fakeClient := &fakeDbSystemAvailabilityDomainClient{
		response: identity.ListAvailabilityDomainsResponse{
			Items: []identity.AvailabilityDomain{
				{Name: common.String("qqZb:US-ASHBURN-AD-1")},
				{Name: common.String("qqZb:US-ASHBURN-AD-2")},
			},
		},
		getCompartmentResponse: identity.GetCompartmentResponse{
			Compartment: identity.Compartment{
				CompartmentId: common.String("ocid1.tenancy.oc1..target"),
			},
		},
	}

	restoreClient := newDbSystemAvailabilityDomainClient
	newDbSystemAvailabilityDomainClient = func(common.ConfigurationProvider) (dbSystemAvailabilityDomainClient, error) {
		return fakeClient, nil
	}
	t.Cleanup(func() {
		newDbSystemAvailabilityDomainClient = restoreClient
	})

	privateKey := testRSAKey(t)
	details, err := buildDbSystemCreateDetails(
		context.Background(),
		testConfigurationProvider{privateKey: privateKey},
		&fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("adminuser"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
		&mysqlv1beta1.DbSystem{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: mysqlv1beta1.DbSystemSpec{
				CompartmentId:        "ocid1.compartment.oc1..target",
				DisplayName:          "mysql-dbsystem-sample",
				ShapeName:            "MySQL.2",
				SubnetId:             "ocid1.subnet.oc1..example",
				AvailabilityDomain:   "US-ASHBURN-AD-1",
				AdminUsername:        usernameSecretSource("admin-secret"),
				AdminPassword:        passwordSecretSource("admin-secret"),
				DataStorageSizeInGBs: 50,
			},
		},
		"default",
	)
	if err != nil {
		t.Fatalf("buildDbSystemCreateDetails() error = %v", err)
	}
	if details.AvailabilityDomain == nil || *details.AvailabilityDomain != "qqZb:US-ASHBURN-AD-1" {
		t.Fatalf("AvailabilityDomain = %v, want suffix-only input expanded against target-compartment tenancy", details.AvailabilityDomain)
	}
	if len(fakeClient.getCompartmentRequests) != 1 {
		t.Fatalf("GetCompartment() calls = %d, want 1", len(fakeClient.getCompartmentRequests))
	}
	if len(fakeClient.listRequests) != 1 {
		t.Fatalf("ListAvailabilityDomains() calls = %d, want 1", len(fakeClient.listRequests))
	}
	if fakeClient.listRequests[0].CompartmentId == nil || *fakeClient.listRequests[0].CompartmentId != "ocid1.tenancy.oc1..target" {
		t.Fatalf("ListAvailabilityDomains() compartment = %v, want target tenancy", fakeClient.listRequests[0].CompartmentId)
	}
}

func TestBuildDbSystemCreateDetailsOmitsAPIServerDefaultedEmptyBlocks(t *testing.T) {
	t.Parallel()

	details, err := buildDbSystemCreateDetails(
		context.Background(),
		nil,
		&fakeCredentialClient{
			secrets: map[string]map[string][]byte{
				"admin-secret": {
					"username": []byte("admin"),
					"password": []byte("ChangeMe123!!"),
				},
			},
		},
		&mysqlv1beta1.DbSystem{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: mysqlv1beta1.DbSystemSpec{
				CompartmentId:        "ocid1.compartment.oc1..example",
				DisplayName:          "mysql-dbsystem-sample",
				ShapeName:            "MySQL.VM.Standard.E3.1.8GB",
				SubnetId:             "ocid1.subnet.oc1..example",
				ConfigurationId:      "ocid1.mysqlconfiguration.oc1..example",
				AdminUsername:        usernameSecretSource("admin-secret"),
				AdminPassword:        passwordSecretSource("admin-secret"),
				DataStorageSizeInGBs: 100,
				BackupPolicy:         mysqlv1beta1.DbSystemBackupPolicy{PitrPolicy: mysqlv1beta1.DbSystemBackupPolicyPitrPolicy{}},
				DeletionPolicy:       mysqlv1beta1.DbSystemDeletionPolicy{},
				Maintenance:          mysqlv1beta1.DbSystemMaintenance{},
				SecureConnections:    mysqlv1beta1.DbSystemSecureConnections{},
				Source:               mysqlv1beta1.DbSystemSource{},
			},
		},
		"default",
	)
	if err != nil {
		t.Fatalf("buildDbSystemCreateDetails() error = %v", err)
	}
	if details.BackupPolicy != nil {
		t.Fatalf("BackupPolicy = %#v, want nil", details.BackupPolicy)
	}
	if details.Source != nil {
		t.Fatalf("Source = %#v, want nil", details.Source)
	}
	if details.Maintenance != nil {
		t.Fatalf("Maintenance = %#v, want nil", details.Maintenance)
	}
	if details.DeletionPolicy != nil {
		t.Fatalf("DeletionPolicy = %#v, want nil", details.DeletionPolicy)
	}
	if details.SecureConnections != nil {
		t.Fatalf("SecureConnections = %#v, want nil", details.SecureConnections)
	}
}

func TestBuildDbSystemCreateDetailsMatchesLegacyStandaloneProjection(t *testing.T) {
	restoreClient := newDbSystemAvailabilityDomainClient
	newDbSystemAvailabilityDomainClient = func(common.ConfigurationProvider) (dbSystemAvailabilityDomainClient, error) {
		return &fakeDbSystemAvailabilityDomainClient{
			response: identity.ListAvailabilityDomainsResponse{
				Items: []identity.AvailabilityDomain{
					{Name: common.String("ypKW:US-ASHBURN-AD-1")},
				},
			},
			getCompartmentResponse: identity.GetCompartmentResponse{
				Compartment: identity.Compartment{
					CompartmentId: common.String("ocid1.tenancy.oc1..example"),
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		newDbSystemAvailabilityDomainClient = restoreClient
	})

	privateKey := testRSAKey(t)
	resource := &mysqlv1beta1.DbSystem{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: mysqlv1beta1.DbSystemSpec{
			CompartmentId:        "ocid1.compartment.oc1..example",
			DisplayName:          "mysql-dbsystem-sample",
			ShapeName:            "MySQL.VM.Standard.E3.1.8GB",
			SubnetId:             "ocid1.subnet.oc1..example",
			ConfigurationId:      "ocid1.mysqlconfiguration.oc1..example",
			AvailabilityDomain:   "qqZb:US-ASHBURN-AD-1",
			IsHighlyAvailable:    false,
			AdminUsername:        usernameSecretSource("admin-secret"),
			AdminPassword:        passwordSecretSource("admin-secret"),
			Description:          "OSOK mysql DbSystem e2e sample",
			DataStorageSizeInGBs: 100,
			Port:                 3306,
			PortX:                33060,
		},
	}
	credentialClient := &fakeCredentialClient{
		secrets: map[string]map[string][]byte{
			"admin-secret": {
				"username": []byte("admin"),
				"password": []byte("ChangeMe123!!"),
			},
		},
	}

	current, err := buildDbSystemCreateDetails(
		context.Background(),
		testConfigurationProvider{privateKey: privateKey},
		credentialClient,
		resource,
		"default",
	)
	if err != nil {
		t.Fatalf("buildDbSystemCreateDetails() error = %v", err)
	}

	legacy := mysqlsdk.CreateDbSystemDetails{
		CompartmentId:        common.String(resource.Spec.CompartmentId),
		ShapeName:            common.String(resource.Spec.ShapeName),
		SubnetId:             common.String(resource.Spec.SubnetId),
		DisplayName:          common.String(resource.Spec.DisplayName),
		Description:          common.String(resource.Spec.Description),
		IsHighlyAvailable:    common.Bool(resource.Spec.IsHighlyAvailable),
		AvailabilityDomain:   common.String(resource.Spec.AvailabilityDomain),
		ConfigurationId:      common.String(resource.Spec.ConfigurationId),
		NsgIds:               []string{},
		AdminUsername:        common.String("admin"),
		AdminPassword:        common.String("ChangeMe123!!"),
		DataStorageSizeInGBs: common.Int(resource.Spec.DataStorageSizeInGBs),
		Port:                 common.Int(resource.Spec.Port),
		PortX:                common.Int(resource.Spec.PortX),
		CustomerContacts:     []mysqlsdk.CustomerContact{},
	}

	if !reflect.DeepEqual(current, legacy) {
		currentJSON, _ := json.Marshal(current)
		legacyJSON, _ := json.Marshal(legacy)
		t.Fatalf("current mysql create body != legacy standalone projection\ncurrent=%s\nlegacy=%s", currentJSON, legacyJSON)
	}
}

func TestNewDbSystemServiceClientWrapsRuntimeWithEndpointSecretClient(t *testing.T) {
	privateKey := testRSAKey(t)
	manager := &DbSystemServiceManager{
		Provider:         testConfigurationProvider{privateKey: privateKey},
		CredentialClient: &fakeDbSystemEndpointCredentialClient{},
	}

	client := newDbSystemServiceClient(manager)

	wrapped, ok := client.(dbSystemEndpointSecretClient)
	if !ok {
		t.Fatalf("newDbSystemServiceClient() type = %T, want dbSystemEndpointSecretClient", client)
	}
	if _, ok := wrapped.delegate.(defaultDbSystemServiceClient); !ok {
		t.Fatalf("wrapped delegate type = %T, want defaultDbSystemServiceClient", wrapped.delegate)
	}
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 1024)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	return privateKey
}
