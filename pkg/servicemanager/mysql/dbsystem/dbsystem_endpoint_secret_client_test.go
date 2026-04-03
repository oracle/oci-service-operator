/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeDbSystemEndpointCredentialClient struct {
	createSecretFn          func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	deleteSecretFn          func(context.Context, string, string) (bool, error)
	deleteSecretIfCurrentFn func(context.Context, string, string, credhelper.SecretRecord) (bool, error)
	getSecretFn             func(context.Context, string, string) (map[string][]byte, error)
	getSecretRecordFn       func(context.Context, string, string) (credhelper.SecretRecord, error)
	updateSecretFn          func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	updateSecretIfCurrentFn func(context.Context, string, string, credhelper.SecretRecord, map[string]string, map[string][]byte) (bool, error)
	createCalled            bool
	deleteCalled            bool
	getCalled               bool
	updateCalled            bool
}

func (f *fakeDbSystemEndpointCredentialClient) CreateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

func (f *fakeDbSystemEndpointCredentialClient) DeleteSecret(ctx context.Context, name string, namespace string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *fakeDbSystemEndpointCredentialClient) DeleteSecretIfCurrent(
	ctx context.Context,
	name string,
	namespace string,
	current credhelper.SecretRecord,
) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretIfCurrentFn != nil {
		return f.deleteSecretIfCurrentFn(ctx, name, namespace, current)
	}
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *fakeDbSystemEndpointCredentialClient) GetSecret(ctx context.Context, name string, namespace string) (map[string][]byte, error) {
	f.getCalled = true
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, namespace)
	}
	return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
}

func (f *fakeDbSystemEndpointCredentialClient) GetSecretRecord(ctx context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
	f.getCalled = true
	if f.getSecretRecordFn != nil {
		return f.getSecretRecordFn(ctx, name, namespace)
	}
	return credhelper.SecretRecord{}, apierrors.NewNotFound(v1.Resource("secret"), name)
}

func (f *fakeDbSystemEndpointCredentialClient) UpdateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.updateCalled = true
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

func (f *fakeDbSystemEndpointCredentialClient) UpdateSecretIfCurrent(
	ctx context.Context,
	name string,
	namespace string,
	current credhelper.SecretRecord,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.updateCalled = true
	if f.updateSecretIfCurrentFn != nil {
		return f.updateSecretIfCurrentFn(ctx, name, namespace, current, labels, data)
	}
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

type fakeDbSystemServiceClient struct {
	createOrUpdateFn func(context.Context, *mysqlv1beta1.DbSystem, ctrl.Request) (servicemanager.OSOKResponse, error)
	deleteFn         func(context.Context, *mysqlv1beta1.DbSystem) (bool, error)
}

func (f fakeDbSystemServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *mysqlv1beta1.DbSystem,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, resource, req)
	}
	return servicemanager.OSOKResponse{}, nil
}

func (f fakeDbSystemServiceClient) Delete(ctx context.Context, resource *mysqlv1beta1.DbSystem) (bool, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, resource)
	}
	return false, nil
}

func TestDbSystemEndpointSecretClientCreatesSecretWhenActive(t *testing.T) {
	t.Parallel()

	resource := newActiveDbSystemResource()
	credClient := &fakeDbSystemEndpointCredentialClient{
		getSecretRecordFn: func(_ context.Context, name, namespace string) (credhelper.SecretRecord, error) {
			requireDbSystemSecretTarget(t, "GetSecretRecord", name, namespace)
			return credhelper.SecretRecord{}, apierrors.NewNotFound(v1.Resource("secret"), name)
		},
		createSecretFn: func(_ context.Context, name, namespace string, labels map[string]string, data map[string][]byte) (bool, error) {
			requireDbSystemSecretTarget(t, "CreateSecret", name, namespace)
			requireDbSystemOwnedSecretLabels(t, labels, string(resource.UID))
			requireDbSystemSecretData(t, data)
			return true, nil
		},
	}
	client := dbSystemEndpointSecretClient{
		delegate: fakeDbSystemServiceClient{createOrUpdateFn: func(context.Context, *mysqlv1beta1.DbSystem, ctrl.Request) (servicemanager.OSOKResponse, error) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want success", response)
	}
	if !credClient.getCalled || !credClient.createCalled {
		t.Fatalf("credential client calls = get:%t create:%t, want both true", credClient.getCalled, credClient.createCalled)
	}
	if credClient.updateCalled {
		t.Fatal("UpdateSecretIfCurrent() should not be called when the endpoint secret does not exist")
	}
	if credClient.deleteCalled {
		t.Fatal("DeleteSecretIfCurrent() should not be called during create/update")
	}
}

func TestDbSystemEndpointSecretClientSkipsSecretSyncBeforeActive(t *testing.T) {
	t.Parallel()

	resource := &mysqlv1beta1.DbSystem{}
	resource.Name = "test-dbsystem"
	resource.Namespace = "default"
	resource.UID = "dbsystem-uid"
	resource.Status.OsokStatus.Reason = string(shared.Provisioning)
	resource.Status.OsokStatus.Conditions = []shared.OSOKCondition{{Type: shared.Provisioning, Status: v1.ConditionTrue}}

	credClient := &fakeDbSystemEndpointCredentialClient{}
	client := dbSystemEndpointSecretClient{
		delegate: fakeDbSystemServiceClient{createOrUpdateFn: func(context.Context, *mysqlv1beta1.DbSystem, ctrl.Request) (servicemanager.OSOKResponse, error) {
			return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true}, nil
		}},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful provisioning requeue", response)
	}
	if credClient.getCalled || credClient.createCalled || credClient.updateCalled || credClient.deleteCalled {
		t.Fatalf("credential calls = get:%t create:%t update:%t delete:%t, want no secret activity before ACTIVE", credClient.getCalled, credClient.createCalled, credClient.updateCalled, credClient.deleteCalled)
	}
}

func TestDbSystemEndpointSecretClientDeletesOwnedSecretAfterDelete(t *testing.T) {
	t.Parallel()

	resource := newActiveDbSystemResource()
	desiredData, err := dbSystemEndpointSecretData(resource)
	if err != nil {
		t.Fatalf("dbSystemEndpointSecretData() error = %v", err)
	}

	credClient := &fakeDbSystemEndpointCredentialClient{
		getSecretRecordFn: func(_ context.Context, name, namespace string) (credhelper.SecretRecord, error) {
			requireDbSystemSecretTarget(t, "GetSecretRecord", name, namespace)
			return credhelper.SecretRecord{
				UID: types.UID("endpoint-secret-uid"),
				Labels: map[string]string{
					dbSystemEndpointSecretOwnerUIDLabel: string(resource.UID),
				},
				Data: desiredData,
			}, nil
		},
		deleteSecretIfCurrentFn: func(_ context.Context, name, namespace string, current credhelper.SecretRecord) (bool, error) {
			requireDbSystemSecretTarget(t, "DeleteSecretIfCurrent", name, namespace)
			if got := current.Labels[dbSystemEndpointSecretOwnerUIDLabel]; got != string(resource.UID) {
				t.Fatalf("guarded delete owner label = %q, want %q", got, resource.UID)
			}
			return true, nil
		},
	}
	client := dbSystemEndpointSecretClient{
		delegate:             fakeDbSystemServiceClient{deleteFn: func(context.Context, *mysqlv1beta1.DbSystem) (bool, error) { return true, nil }},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true")
	}
	if !credClient.getCalled || !credClient.deleteCalled {
		t.Fatalf("credential client calls = get:%t delete:%t, want both true", credClient.getCalled, credClient.deleteCalled)
	}
	if credClient.createCalled || credClient.updateCalled {
		t.Fatalf("credential client calls = create:%t update:%t, want both false", credClient.createCalled, credClient.updateCalled)
	}
}

func newActiveDbSystemResource() *mysqlv1beta1.DbSystem {
	resource := &mysqlv1beta1.DbSystem{}
	resource.Name = "test-dbsystem"
	resource.Namespace = "default"
	resource.UID = "dbsystem-uid"
	resource.Status.LifecycleState = "ACTIVE"
	resource.Status.IpAddress = "10.0.0.10"
	resource.Status.HostnameLabel = "mysql-host"
	resource.Status.AvailabilityDomain = "ypKW:US-ASHBURN-AD-1"
	resource.Status.FaultDomain = "FAULT-DOMAIN-1"
	resource.Status.Port = 3306
	resource.Status.PortX = 33060
	resource.Status.Endpoints = []mysqlv1beta1.DbSystemEndpoint{
		{
			IpAddress: "10.0.0.10",
			Port:      3306,
			PortX:     33060,
			Hostname:  "mysql-host.example.internal",
			Modes:     []string{"READ", "WRITE"},
			Status:    "ACTIVE",
		},
	}
	resource.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..example"
	resource.Status.OsokStatus.Reason = string(shared.Active)
	resource.Status.OsokStatus.Conditions = []shared.OSOKCondition{{Type: shared.Active, Status: v1.ConditionTrue}}
	return resource
}

func requireDbSystemSecretTarget(t *testing.T, action string, name string, namespace string) {
	t.Helper()
	if name != "test-dbsystem" || namespace != "default" {
		t.Fatalf("%s() target = %s/%s, want default/test-dbsystem", action, namespace, name)
	}
}

func requireDbSystemOwnedSecretLabels(t *testing.T, labels map[string]string, wantUID string) {
	t.Helper()
	if got := labels[dbSystemEndpointSecretOwnerUIDLabel]; got != wantUID {
		t.Fatalf("secret owner label = %q, want %q", got, wantUID)
	}
}

func requireDbSystemSecretData(t *testing.T, data map[string][]byte) {
	t.Helper()
	if got := string(data["PrivateIPAddress"]); got != "10.0.0.10" {
		t.Fatalf("PrivateIPAddress = %q, want 10.0.0.10", got)
	}
	if got := string(data["InternalFQDN"]); got != "mysql-host" {
		t.Fatalf("InternalFQDN = %q, want mysql-host", got)
	}
	if got := string(data["AvailabilityDomain"]); got != "ypKW:US-ASHBURN-AD-1" {
		t.Fatalf("AvailabilityDomain = %q, want ypKW:US-ASHBURN-AD-1", got)
	}
	if got := string(data["FaultDomain"]); got != "FAULT-DOMAIN-1" {
		t.Fatalf("FaultDomain = %q, want FAULT-DOMAIN-1", got)
	}
	if got := string(data["MySQLPort"]); got != "3306" {
		t.Fatalf("MySQLPort = %q, want 3306", got)
	}
	if got := string(data["MySQLXProtocolPort"]); got != "33060" {
		t.Fatalf("MySQLXProtocolPort = %q, want 33060", got)
	}

	var endpoints []mysqlv1beta1.DbSystemEndpoint
	if err := json.Unmarshal(data["Endpoints"], &endpoints); err != nil {
		t.Fatalf("Endpoints JSON unmarshal error = %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("len(Endpoints) = %d, want 1", len(endpoints))
	}
	if endpoints[0].Hostname != "mysql-host.example.internal" {
		t.Fatalf("Endpoints[0].Hostname = %q, want mysql-host.example.internal", endpoints[0].Hostname)
	}
}

func TestDbSystemEndpointSecretDataRejectsMissingObservedFields(t *testing.T) {
	t.Parallel()

	resource := &mysqlv1beta1.DbSystem{}
	resource.Name = "test-dbsystem"
	resource.Namespace = "default"
	resource.UID = "dbsystem-uid"

	_, err := dbSystemEndpointSecretData(resource)
	if err == nil {
		t.Fatal("dbSystemEndpointSecretData() error = nil, want missing status field failure")
	}
	if got := err.Error(); got == "" {
		t.Fatal("dbSystemEndpointSecretData() error = empty string, want context")
	}
}

func TestDbSystemEndpointSecretOwnerUIDRequiresUID(t *testing.T) {
	t.Parallel()

	_, err := dbSystemEndpointSecretOwnerUID(&mysqlv1beta1.DbSystem{})
	if err == nil {
		t.Fatal("dbSystemEndpointSecretOwnerUID() error = nil, want missing UID failure")
	}
	if got := err.Error(); got == "" {
		t.Fatal("dbSystemEndpointSecretOwnerUID() error = empty string, want context")
	}
}

func Example_dbSystemEndpointSecretData() {
	resource := newActiveDbSystemResource()
	data, err := dbSystemEndpointSecretData(resource)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s %s\n", data["MySQLPort"], data["PrivateIPAddress"])
	// Output:
	// 3306 10.0.0.10
}
