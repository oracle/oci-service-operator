/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package credhelper

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

type SecretRecord struct {
	UID    types.UID
	Labels map[string]string
	Data   map[string][]byte
}

type CredentialClient interface {
	CreateSecret(ctx context.Context, secretName string, secretNamespace string, labels map[string]string,
		data map[string][]byte) (bool, error)

	DeleteSecret(ctx context.Context, secretName string, secretNamespace string) (bool, error)

	GetSecret(ctx context.Context, secretName string, secretNamespace string) (map[string][]byte, error)

	UpdateSecret(ctx context.Context, secretName string, secretNamespace string, labels map[string]string,
		data map[string][]byte) (bool, error)
}

type SecretRecordReader interface {
	GetSecretRecord(ctx context.Context, secretName string, secretNamespace string) (SecretRecord, error)
}

type GuardedSecretMutator interface {
	DeleteSecretIfCurrent(ctx context.Context, secretName string, secretNamespace string, current SecretRecord) (bool, error)

	UpdateSecretIfCurrent(ctx context.Context, secretName string, secretNamespace string, current SecretRecord, labels map[string]string,
		data map[string][]byte) (bool, error)
}
