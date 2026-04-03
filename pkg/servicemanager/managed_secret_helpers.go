/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"context"
	"fmt"
	"reflect"

	"github.com/oracle/oci-service-operator/pkg/credhelper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	ManagedSecretLabelKey     = "oci.oracle.com/osok-managed"
	ManagedSecretLabelValue   = "true"
	ManagedSecretOwnerKindKey = "oci.oracle.com/osok-owner-kind"
	ManagedSecretOwnerNameKey = "oci.oracle.com/osok-owner-name"

	managedSecretDataKey   = "_osok_managed"
	managedSecretOwnerKind = "_osok_owner_kind"
	managedSecretOwnerName = "_osok_owner_name"
)

func ManagedSecretLabels(ownerKind, ownerName string) map[string]string {
	return map[string]string{
		ManagedSecretLabelKey:     ManagedSecretLabelValue,
		ManagedSecretOwnerKindKey: ownerKind,
		ManagedSecretOwnerNameKey: ownerName,
	}
}

func AddManagedSecretData(data map[string][]byte, ownerKind, ownerName string) map[string][]byte {
	managed := make(map[string][]byte, len(data)+3)
	for key, value := range data {
		copyValue := make([]byte, len(value))
		copy(copyValue, value)
		managed[key] = copyValue
	}
	managed[managedSecretDataKey] = []byte(ManagedSecretLabelValue)
	managed[managedSecretOwnerKind] = []byte(ownerKind)
	managed[managedSecretOwnerName] = []byte(ownerName)
	return managed
}

func SecretOwnedBy(data map[string][]byte, ownerKind, ownerName string) bool {
	return string(data[managedSecretDataKey]) == ManagedSecretLabelValue &&
		string(data[managedSecretOwnerKind]) == ownerKind &&
		string(data[managedSecretOwnerName]) == ownerName
}

func SecretMatchesExpectedData(existing, expected map[string][]byte) bool {
	return reflect.DeepEqual(stripManagedSecretData(existing), stripManagedSecretData(expected))
}

func EnsureOwnedSecret(
	ctx context.Context,
	client credhelper.CredentialClient,
	secretName,
	secretNamespace,
	ownerKind,
	ownerName string,
	data map[string][]byte,
) (bool, error) {
	managedData := AddManagedSecretData(data, ownerKind, ownerName)
	labels := ManagedSecretLabels(ownerKind, ownerName)

	ok, err := client.CreateSecret(ctx, secretName, secretNamespace, labels, managedData)
	if err == nil {
		return ok, nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return false, err
	}

	existing, getErr := client.GetSecret(ctx, secretName, secretNamespace)
	if getErr != nil {
		return false, getErr
	}
	if SecretOwnedBy(existing, ownerKind, ownerName) {
		return true, nil
	}

	return false, fmt.Errorf(
		"secret %s/%s already exists and is not owned by %s %s",
		secretNamespace,
		secretName,
		ownerKind,
		ownerName,
	)
}

func DeleteOwnedSecretIfPresent(
	ctx context.Context,
	client credhelper.CredentialClient,
	secretName,
	secretNamespace,
	ownerKind,
	ownerName string,
) (bool, error) {
	existing, err := client.GetSecret(ctx, secretName, secretNamespace)
	if err != nil {
		if IsSecretNotFoundError(err) {
			return true, nil
		}
		return false, err
	}
	if !SecretOwnedBy(existing, ownerKind, ownerName) {
		return true, nil
	}

	_, err = client.DeleteSecret(ctx, secretName, secretNamespace)
	if err != nil && !IsSecretNotFoundError(err) {
		return false, err
	}
	return true, nil
}

func stripManagedSecretData(data map[string][]byte) map[string][]byte {
	stripped := make(map[string][]byte, len(data))
	for key, value := range data {
		switch key {
		case managedSecretDataKey, managedSecretOwnerKind, managedSecretOwnerName:
			continue
		default:
			copyValue := make([]byte, len(value))
			copy(copyValue, value)
			stripped[key] = copyValue
		}
	}
	return stripped
}
