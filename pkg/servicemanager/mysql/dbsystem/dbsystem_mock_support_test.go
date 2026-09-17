/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dbsystem

import (
	"context"
	"fmt"

	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockDbSystemName = "osok-mock-mysql-v1"

func newMockDbSystemClient(sdkClient mysqlsdk.DbSystemClient, credentials credhelper.CredentialClient) DbSystemServiceClient {
	manager := &DbSystemServiceManager{
		CredentialClient: credentials,
		Log:              loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
	}
	hooks := newDbSystemDefaultRuntimeHooks(sdkClient)
	applyDbSystemRuntimeHooks(manager, &hooks)
	appendDbSystemEndpointSecretRuntimeWrapper(manager, &hooks)
	delegate := defaultDbSystemServiceClient{ServiceClient: generatedruntime.NewServiceClient[*mysqlv1beta1.DbSystem](buildDbSystemGeneratedRuntimeConfig(manager, hooks))}
	return wrapDbSystemGeneratedClient(hooks, delegate)
}

type mockDbSystemCredentialClient struct {
	secrets map[string]map[string][]byte
	records map[string]credhelper.SecretRecord
}

var _ credhelper.CredentialClient = (*mockDbSystemCredentialClient)(nil)

func (c *mockDbSystemCredentialClient) CreateSecret(_ context.Context, name string, _ string, labels map[string]string, data map[string][]byte) (bool, error) {
	if c.records == nil {
		c.records = map[string]credhelper.SecretRecord{}
	}
	if _, exists := c.records[name]; exists {
		return false, apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, name)
	}
	c.records[name] = credhelper.SecretRecord{
		UID:    types.UID("mock-endpoint-secret-uid"),
		Labels: cloneMockDbSystemStringMap(labels),
		Data:   cloneMockDbSystemByteMap(data),
	}
	return true, nil
}

func (c *mockDbSystemCredentialClient) DeleteSecret(_ context.Context, name string, _ string) (bool, error) {
	if _, exists := c.records[name]; !exists {
		return false, nil
	}
	delete(c.records, name)
	return true, nil
}

func (c *mockDbSystemCredentialClient) GetSecret(_ context.Context, name string, namespace string) (map[string][]byte, error) {
	secret, ok := c.secrets[name]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
	}
	return secret, nil
}

func (c *mockDbSystemCredentialClient) UpdateSecret(_ context.Context, name string, _ string, labels map[string]string, data map[string][]byte) (bool, error) {
	record, exists := c.records[name]
	if !exists {
		return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	if labels != nil {
		record.Labels = cloneMockDbSystemStringMap(labels)
	}
	record.Data = cloneMockDbSystemByteMap(data)
	c.records[name] = record
	return true, nil
}

func (c *mockDbSystemCredentialClient) GetSecretRecord(_ context.Context, name string, _ string) (credhelper.SecretRecord, error) {
	record, exists := c.records[name]
	if !exists {
		return credhelper.SecretRecord{}, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	record.Labels = cloneMockDbSystemStringMap(record.Labels)
	record.Data = cloneMockDbSystemByteMap(record.Data)
	return record, nil
}

func (c *mockDbSystemCredentialClient) UpdateSecretIfCurrent(_ context.Context, name string, _ string, current credhelper.SecretRecord, labels map[string]string, data map[string][]byte) (bool, error) {
	record, exists := c.records[name]
	if !exists {
		return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	if record.UID != current.UID {
		return false, fmt.Errorf("endpoint Secret %s changed before guarded update", name)
	}
	if labels != nil {
		record.Labels = cloneMockDbSystemStringMap(labels)
	}
	record.Data = cloneMockDbSystemByteMap(data)
	c.records[name] = record
	return true, nil
}

func (c *mockDbSystemCredentialClient) DeleteSecretIfCurrent(_ context.Context, name string, _ string, current credhelper.SecretRecord) (bool, error) {
	record, exists := c.records[name]
	if !exists {
		return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	if record.UID != current.UID {
		return false, fmt.Errorf("endpoint Secret %s changed before guarded delete", name)
	}
	delete(c.records, name)
	return true, nil
}

func (c *mockDbSystemCredentialClient) hasEndpointSecret(name string) bool {
	_, exists := c.records[name]
	return exists
}

func cloneMockDbSystemStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneMockDbSystemByteMap(source map[string][]byte) map[string][]byte {
	if source == nil {
		return nil
	}
	cloned := make(map[string][]byte, len(source))
	for key, value := range source {
		cloned[key] = append([]byte(nil), value...)
	}
	return cloned
}

func mockDbSystemUsernameSource(name string) shared.UsernameSource {
	return shared.UsernameSource{Secret: shared.SecretSource{SecretName: name}}
}

func mockDbSystemPasswordSource(name string) shared.PasswordSource {
	return shared.PasswordSource{Secret: shared.SecretSource{SecretName: name}}
}
