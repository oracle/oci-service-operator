/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package credhelper

import "context"

type CredentialClient interface {
	CreateSecret(ctx context.Context, secretName string, secretNamespace string, labels map[string]string,
		data map[string][]byte) (bool, error)

	DeleteSecret(ctx context.Context, secretName string, secretNamespace string) (bool, error)

	GetSecret(ctx context.Context, secretName string, secretNamespace string) (map[string][]byte, error)

	UpdateSecret(ctx context.Context, secretName string, secretNamespace string, labels map[string]string,
		data map[string][]byte) (bool, error)
}
