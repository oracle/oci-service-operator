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

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 scheme: %v", err)
	}
	return scheme
}

func newTestMetrics() *metrics.Metrics {
	logger := loggerutil.OSOKLogger{Logger: logr.Discard()}
	return &metrics.Metrics{
		Name:        "oci",
		ServiceName: "KubeSecretTest",
		Logger:      logger,
	}
}

func newTestKubeSecretClient(t *testing.T, objs ...ctrlclient.Object) *KubeSecretClient {
	t.Helper()

	scheme := newTestScheme(t)

	logger := loggerutil.OSOKLogger{Logger: logr.Discard()}
	return New(
		fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		logger,
		newTestMetrics(),
	)
}

const (
	testSecretName      = "test-secret"
	testSecretNamespace = "default"
)

func expectSecretData(t *testing.T, client *KubeSecretClient, want map[string][]byte) {
	t.Helper()

	retrieved, err := client.GetSecret(context.Background(), testSecretName, testSecretNamespace)
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if !reflect.DeepEqual(want, retrieved) {
		t.Fatalf("unexpected secret data: got %v want %v", retrieved, want)
	}
}

func mustCreateSecret(t *testing.T, client *KubeSecretClient, labels map[string]string, data map[string][]byte) {
	t.Helper()

	created, err := client.CreateSecret(context.Background(), testSecretName, testSecretNamespace, labels, data)
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}
	if !created {
		t.Fatal("expected secret to be created")
	}
}

func mustUpdateSecret(t *testing.T, client *KubeSecretClient, labels map[string]string, data map[string][]byte) {
	t.Helper()

	updated, err := client.UpdateSecret(context.Background(), testSecretName, testSecretNamespace, labels, data)
	if err != nil {
		t.Fatalf("update secret: %v", err)
	}
	if !updated {
		t.Fatal("expected secret to be updated")
	}
}

func mustDeleteSecret(t *testing.T, client *KubeSecretClient) {
	t.Helper()

	deleted, err := client.DeleteSecret(context.Background(), testSecretName, testSecretNamespace)
	if err != nil {
		t.Fatalf("delete secret: %v", err)
	}
	if !deleted {
		t.Fatal("expected secret to be deleted")
	}
}

func expectSecretNotFound(t *testing.T, client *KubeSecretClient) {
	t.Helper()

	_, err := client.GetSecret(context.Background(), testSecretName, testSecretNamespace)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

type staleGetClient struct {
	ctrlclient.Client
	getCalls int
	getErr   error
}

func (c *staleGetClient) Get(ctx context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
	c.getCalls++
	return c.getErr
}

func TestCreateUpdateDeleteSecret(t *testing.T) {
	client := newTestKubeSecretClient(t)
	labels := map[string]string{"label_def": "default_label"}
	data := map[string][]byte{
		"secret": []byte("test"),
		"data":   []byte("default"),
	}

	mustCreateSecret(t, client, labels, data)
	expectSecretData(t, client, data)

	updatedData := map[string][]byte{
		"secret": []byte("updated_test"),
		"data":   []byte("updated_default"),
		"value":  []byte("updated value"),
	}
	mustUpdateSecret(t, client, labels, updatedData)
	expectSecretData(t, client, updatedData)
	mustDeleteSecret(t, client)
	expectSecretNotFound(t, client)
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

func TestGetSecretUsesConfiguredReader(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret.DeepCopy()).Build()
	cachedClient := &staleGetClient{
		Client: baseClient,
		getErr: apierrors.NewNotFound(corev1.Resource("secret"), existingSecret.Name),
	}
	client := NewWithReader(
		cachedClient,
		baseClient,
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		newTestMetrics(),
	)

	data, err := client.GetSecret(context.Background(), existingSecret.Name, existingSecret.Namespace)
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if !reflect.DeepEqual(existingSecret.Data, data) {
		t.Fatalf("unexpected secret data: got %v want %v", data, existingSecret.Data)
	}
	if cachedClient.getCalls != 0 {
		t.Fatalf("expected reads to use the configured Reader, got %d cached Client.Get calls", cachedClient.getCalls)
	}
}

func TestUpdateSecretUsesConfiguredReader(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
			Labels:    map[string]string{"existing": "label"},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret.DeepCopy()).Build()
	cachedClient := &staleGetClient{
		Client: baseClient,
		getErr: apierrors.NewNotFound(corev1.Resource("secret"), existingSecret.Name),
	}
	client := NewWithReader(
		cachedClient,
		baseClient,
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		newTestMetrics(),
	)

	updatedLabels := map[string]string{"updated": "label"}
	updatedData := map[string][]byte{"key": []byte("new-value")}
	updated, err := client.UpdateSecret(context.Background(), existingSecret.Name, existingSecret.Namespace, updatedLabels, updatedData)
	if err != nil {
		t.Fatalf("update secret: %v", err)
	}
	if !updated {
		t.Fatal("expected secret to be updated")
	}
	if cachedClient.getCalls != 0 {
		t.Fatalf("expected update to use the configured Reader, got %d cached Client.Get calls", cachedClient.getCalls)
	}

	storedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: existingSecret.Name, Namespace: existingSecret.Namespace}, storedSecret); err != nil {
		t.Fatalf("load updated secret: %v", err)
	}
	if !reflect.DeepEqual(updatedData, storedSecret.Data) {
		t.Fatalf("unexpected updated data: got %v want %v", storedSecret.Data, updatedData)
	}
	if !reflect.DeepEqual(updatedLabels, storedSecret.Labels) {
		t.Fatalf("unexpected updated labels: got %v want %v", storedSecret.Labels, updatedLabels)
	}
}
