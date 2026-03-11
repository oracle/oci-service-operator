/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kubesecret

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestKubeSecretClient(t *testing.T, objs ...ctrlclient.Object) *KubeSecretClient {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 scheme: %v", err)
	}

	logger := loggerutil.OSOKLogger{Logger: logr.Discard()}
	return New(
		fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		logger,
		&metrics.Metrics{
			Name:        "oci",
			ServiceName: "KubeSecretTest",
			Logger:      logger,
		},
	)
}

func TestCreateUpdateDeleteSecret(t *testing.T) {
	client := newTestKubeSecretClient(t)
	ctx := context.Background()

	secretName := "test-secret"
	secretNamespace := "default"
	labels := map[string]string{"label_def": "default_label"}
	data := map[string][]byte{
		"secret": []byte("test"),
		"data":   []byte("default"),
	}

	created, err := client.CreateSecret(ctx, secretName, secretNamespace, labels, data)
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}
	if !created {
		t.Fatalf("expected secret to be created")
	}

	retrieved, err := client.GetSecret(ctx, secretName, secretNamespace)
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if !reflect.DeepEqual(data, retrieved) {
		t.Fatalf("unexpected secret data after create: got %v want %v", retrieved, data)
	}

	updatedData := map[string][]byte{
		"secret": []byte("updated_test"),
		"data":   []byte("updated_default"),
		"value":  []byte("updated value"),
	}
	updated, err := client.UpdateSecret(ctx, secretName, secretNamespace, labels, updatedData)
	if err != nil {
		t.Fatalf("update secret: %v", err)
	}
	if !updated {
		t.Fatalf("expected secret to be updated")
	}

	retrieved, err = client.GetSecret(ctx, secretName, secretNamespace)
	if err != nil {
		t.Fatalf("get updated secret: %v", err)
	}
	if !reflect.DeepEqual(updatedData, retrieved) {
		t.Fatalf("unexpected secret data after update: got %v want %v", retrieved, updatedData)
	}

	deleted, err := client.DeleteSecret(ctx, secretName, secretNamespace)
	if err != nil {
		t.Fatalf("delete secret: %v", err)
	}
	if !deleted {
		t.Fatalf("expected secret to be deleted")
	}

	_, err = client.GetSecret(ctx, secretName, secretNamespace)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestCreateSecretAlreadyExists(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	client := newTestKubeSecretClient(t, existingSecret)

	created, err := client.CreateSecret(
		context.Background(),
		existingSecret.Name,
		existingSecret.Namespace,
		nil,
		map[string][]byte{"key": []byte("new-value")},
	)
	if created {
		t.Fatalf("expected create to report secret already exists")
	}
	if !apierrors.IsAlreadyExists(err) {
		t.Fatalf("expected already exists error, got %v", err)
	}
}
