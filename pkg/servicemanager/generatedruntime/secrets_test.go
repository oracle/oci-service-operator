/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestServiceClientCreateOrUpdateResolvesSecretBackedBodyFields(t *testing.T) {
	t.Parallel()
	var createRequest fakeCreateThingWithSecretRequest
	credClient := &fakeCredentialClient{secrets: map[string]map[string][]byte{"admin-user": {"username": []byte("dbadmin")}, "admin-password": {"password": []byte("SuperSecret123")}}}
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", CredentialClient: credClient, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingWithSecretRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		createRequest = *request.(*fakeCreateThingWithSecretRequest)
		return fakeCreateThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..create", DisplayName: "created-name", LifecycleState: "ACTIVE"}}, nil
	}}})
	resource := &fakeResource{Namespace: "database-system", Spec: fakeSpec{DisplayName: "created-name", AdminUsername: shared.UsernameSource{Secret: shared.SecretSource{SecretName: "admin-user"}}, AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-password"}}}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireStringEqual(t, "create request displayName", createRequest.DisplayName, "created-name")
	requireStringEqual(t, "create request adminUsername", createRequest.AdminUsername, "dbadmin")
	requireStringEqual(t, "create request adminPassword", createRequest.AdminPassword, "SuperSecret123")
	requireStringEqual(t, "GetSecret() names", strings.Join(credClient.getCalls, ","), "admin-user,admin-password")
	for _, namespace := range credClient.namespaces {
		requireStringEqual(t, "GetSecret() namespace", namespace, "database-system")
	}
	requireStringEqual(t, "status.adminUsername.secret.secretName", resource.Status.AdminUsername.Secret.SecretName, "admin-user")
	requireStringEqual(t, "status.adminPassword.secret.secretName", resource.Status.AdminPassword.Secret.SecretName, "admin-password")
}

func TestServiceClientCreateOrUpdateFailsWhenSecretKeyIsMissing(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", CredentialClient: &fakeCredentialClient{secrets: map[string]map[string][]byte{"admin-password": {"not-password": []byte("missing")}}}, Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingWithSecretRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeCreateThingResponse{}, nil
	}}})
	resource := &fakeResource{Namespace: "database-system", Spec: fakeSpec{AdminPassword: shared.PasswordSource{Secret: shared.SecretSource{SecretName: "admin-password"}}}}
	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), `password key in secret "admin-password" is not found`) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing password key failure", err)
	}
}

func TestGeneratedCredentialSourceFieldsOmitZeroJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    any
		unwanted []string
	}{{name: "autonomous database spec", value: databasev1beta1.AutonomousDatabaseSpec{CompartmentId: "ocid1.compartment.oc1..adb", DisplayName: "adb-sample"}, unwanted: []string{`"adminPassword"`}}, {name: "mysql dbsystem spec", value: mysqlv1beta1.DbSystemSpec{CompartmentId: "ocid1.compartment.oc1..mysql", ShapeName: "MySQL.VM.Standard.E4.1.8GB", SubnetId: "ocid1.subnet.oc1..mysql"}, unwanted: []string{`"adminUsername"`, `"adminPassword"`}}, {name: "mysql dbsystem status", value: mysqlv1beta1.DbSystemStatus{}, unwanted: []string{`"adminUsername"`, `"adminPassword"`}}}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			for _, token := range tc.unwanted {
				if strings.Contains(string(payload), token) {
					t.Fatalf("json.Marshal() = %s, unexpected token %s", payload, token)
				}
			}
		})
	}
}
