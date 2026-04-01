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

type capturingDeleteClient struct {
	ctrlclient.Client
	deleteCalls         int
	deletePreconditions *metav1.Preconditions
}

func (c *capturingDeleteClient) Delete(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.DeleteOption) error {
	c.deleteCalls++
	deleteOptions := (&ctrlclient.DeleteOptions{}).ApplyOptions(opts)
	if deleteOptions.Preconditions != nil {
		copied := deleteOptions.Preconditions.DeepCopy()
		c.deletePreconditions = copied
	}
	return nil
}

func requireDeletePreconditions(t *testing.T, got *metav1.Preconditions, want *corev1.Secret) {
	t.Helper()

	if got == nil {
		t.Fatal("expected delete preconditions to be set")
	}
	if got.UID == nil || *got.UID != want.UID {
		t.Fatalf("delete UID precondition = %v, want %q", got.UID, want.UID)
	}
	if got.ResourceVersion == nil || *got.ResourceVersion != want.ResourceVersion {
		t.Fatalf("delete resourceVersion precondition = %v, want %q", got.ResourceVersion, want.ResourceVersion)
	}
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

func TestGetSecretRecordUsesConfiguredReader(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
			UID:       "secret-uid",
			Labels:    map[string]string{"managed-by": "osok"},
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

	record, err := client.GetSecretRecord(context.Background(), existingSecret.Name, existingSecret.Namespace)
	if err != nil {
		t.Fatalf("get secret record: %v", err)
	}
	if !reflect.DeepEqual(existingSecret.Data, record.Data) {
		t.Fatalf("unexpected secret data: got %v want %v", record.Data, existingSecret.Data)
	}
	if !reflect.DeepEqual(existingSecret.Labels, record.Labels) {
		t.Fatalf("unexpected secret labels: got %v want %v", record.Labels, existingSecret.Labels)
	}
	if record.UID != existingSecret.UID {
		t.Fatalf("unexpected secret UID: got %q want %q", record.UID, existingSecret.UID)
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

func TestUpdateSecretIfCurrentRejectsReplacedSecret(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	originalSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
			UID:       "secret-uid-1",
			Labels:    map[string]string{"managed-by": "osok"},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	replacementSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      originalSecret.Name,
			Namespace: originalSecret.Namespace,
			UID:       "secret-uid-2",
			Labels:    map[string]string{"managed-by": "other"},
		},
		Data: map[string][]byte{"key": []byte("replacement")},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(originalSecret.DeepCopy()).Build()
	client := NewWithReader(
		baseClient,
		baseClient,
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		newTestMetrics(),
	)

	current, err := client.GetSecretRecord(context.Background(), originalSecret.Name, originalSecret.Namespace)
	if err != nil {
		t.Fatalf("get secret record: %v", err)
	}
	if err := baseClient.Delete(context.Background(), originalSecret.DeepCopy()); err != nil {
		t.Fatalf("delete original secret: %v", err)
	}
	if err := baseClient.Create(context.Background(), replacementSecret.DeepCopy()); err != nil {
		t.Fatalf("create replacement secret: %v", err)
	}

	updated, err := client.UpdateSecretIfCurrent(
		context.Background(),
		originalSecret.Name,
		originalSecret.Namespace,
		current,
		map[string]string{"managed-by": "osok"},
		map[string][]byte{"key": []byte("updated")},
	)
	if updated {
		t.Fatal("expected guarded update to reject a replaced Secret")
	}
	if !apierrors.IsConflict(err) {
		t.Fatalf("expected conflict after Secret replacement, got %v", err)
	}

	storedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: originalSecret.Name, Namespace: originalSecret.Namespace}, storedSecret); err != nil {
		t.Fatalf("load replacement secret: %v", err)
	}
	if storedSecret.UID != replacementSecret.UID {
		t.Fatalf("replacement secret UID = %q, want %q", storedSecret.UID, replacementSecret.UID)
	}
	if !reflect.DeepEqual(storedSecret.Data, replacementSecret.Data) {
		t.Fatalf("replacement secret data mutated: got %v want %v", storedSecret.Data, replacementSecret.Data)
	}
	if !reflect.DeepEqual(storedSecret.Labels, replacementSecret.Labels) {
		t.Fatalf("replacement secret labels mutated: got %v want %v", storedSecret.Labels, replacementSecret.Labels)
	}
}

func TestDeleteSecretIfCurrentRejectsReplacedSecret(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	originalSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
			UID:       "secret-uid-1",
			Labels:    map[string]string{"managed-by": "osok"},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	replacementSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      originalSecret.Name,
			Namespace: originalSecret.Namespace,
			UID:       "secret-uid-2",
			Labels:    map[string]string{"managed-by": "other"},
		},
		Data: map[string][]byte{"key": []byte("replacement")},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(originalSecret.DeepCopy()).Build()
	client := NewWithReader(
		baseClient,
		baseClient,
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		newTestMetrics(),
	)

	current, err := client.GetSecretRecord(context.Background(), originalSecret.Name, originalSecret.Namespace)
	if err != nil {
		t.Fatalf("get secret record: %v", err)
	}
	if err := baseClient.Delete(context.Background(), originalSecret.DeepCopy()); err != nil {
		t.Fatalf("delete original secret: %v", err)
	}
	if err := baseClient.Create(context.Background(), replacementSecret.DeepCopy()); err != nil {
		t.Fatalf("create replacement secret: %v", err)
	}

	deleted, err := client.DeleteSecretIfCurrent(context.Background(), originalSecret.Name, originalSecret.Namespace, current)
	if deleted {
		t.Fatal("expected guarded delete to reject a replaced Secret")
	}
	if !apierrors.IsConflict(err) {
		t.Fatalf("expected conflict after Secret replacement, got %v", err)
	}

	storedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: originalSecret.Name, Namespace: originalSecret.Namespace}, storedSecret); err != nil {
		t.Fatalf("load replacement secret: %v", err)
	}
	if storedSecret.UID != replacementSecret.UID {
		t.Fatalf("replacement secret UID = %q, want %q", storedSecret.UID, replacementSecret.UID)
	}
}

func TestUpdateSecretIfCurrentRejectsSameUIDStateChange(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	originalSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
			UID:       "secret-uid-1",
			Labels:    map[string]string{"managed-by": "osok"},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(originalSecret.DeepCopy()).Build()
	client := NewWithReader(
		baseClient,
		baseClient,
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		newTestMetrics(),
	)

	current, err := client.GetSecretRecord(context.Background(), originalSecret.Name, originalSecret.Namespace)
	if err != nil {
		t.Fatalf("get secret record: %v", err)
	}

	mutatedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: originalSecret.Name, Namespace: originalSecret.Namespace}, mutatedSecret); err != nil {
		t.Fatalf("load secret to mutate: %v", err)
	}
	mutatedSecret.Labels = map[string]string{"managed-by": "other"}
	mutatedSecret.Data = map[string][]byte{"key": []byte("rotated")}
	if err := baseClient.Update(context.Background(), mutatedSecret); err != nil {
		t.Fatalf("mutate secret in place: %v", err)
	}

	updated, err := client.UpdateSecretIfCurrent(
		context.Background(),
		originalSecret.Name,
		originalSecret.Namespace,
		current,
		map[string]string{"managed-by": "osok"},
		map[string][]byte{"key": []byte("updated")},
	)
	if updated {
		t.Fatal("expected guarded update to reject same-UID state changes")
	}
	if !apierrors.IsConflict(err) {
		t.Fatalf("expected conflict after same-UID state change, got %v", err)
	}

	storedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: originalSecret.Name, Namespace: originalSecret.Namespace}, storedSecret); err != nil {
		t.Fatalf("load mutated secret: %v", err)
	}
	if storedSecret.UID != originalSecret.UID {
		t.Fatalf("mutated secret UID = %q, want %q", storedSecret.UID, originalSecret.UID)
	}
	if !reflect.DeepEqual(storedSecret.Data, mutatedSecret.Data) {
		t.Fatalf("mutated secret data overwritten: got %v want %v", storedSecret.Data, mutatedSecret.Data)
	}
	if !reflect.DeepEqual(storedSecret.Labels, mutatedSecret.Labels) {
		t.Fatalf("mutated secret labels overwritten: got %v want %v", storedSecret.Labels, mutatedSecret.Labels)
	}
}

func TestDeleteSecretIfCurrentRejectsSameUIDStateChange(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	originalSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
			UID:       "secret-uid-1",
			Labels:    map[string]string{"managed-by": "osok"},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(originalSecret.DeepCopy()).Build()
	client := NewWithReader(
		baseClient,
		baseClient,
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		newTestMetrics(),
	)

	current, err := client.GetSecretRecord(context.Background(), originalSecret.Name, originalSecret.Namespace)
	if err != nil {
		t.Fatalf("get secret record: %v", err)
	}

	mutatedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: originalSecret.Name, Namespace: originalSecret.Namespace}, mutatedSecret); err != nil {
		t.Fatalf("load secret to mutate: %v", err)
	}
	mutatedSecret.Labels = map[string]string{"managed-by": "other"}
	mutatedSecret.Data = map[string][]byte{"key": []byte("rotated")}
	if err := baseClient.Update(context.Background(), mutatedSecret); err != nil {
		t.Fatalf("mutate secret in place: %v", err)
	}

	deleted, err := client.DeleteSecretIfCurrent(context.Background(), originalSecret.Name, originalSecret.Namespace, current)
	if deleted {
		t.Fatal("expected guarded delete to reject same-UID state changes")
	}
	if !apierrors.IsConflict(err) {
		t.Fatalf("expected conflict after same-UID state change, got %v", err)
	}

	storedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: originalSecret.Name, Namespace: originalSecret.Namespace}, storedSecret); err != nil {
		t.Fatalf("load mutated secret: %v", err)
	}
	if storedSecret.UID != originalSecret.UID {
		t.Fatalf("mutated secret UID = %q, want %q", storedSecret.UID, originalSecret.UID)
	}
	if !reflect.DeepEqual(storedSecret.Data, mutatedSecret.Data) {
		t.Fatalf("mutated secret data deleted or overwritten: got %v want %v", storedSecret.Data, mutatedSecret.Data)
	}
	if !reflect.DeepEqual(storedSecret.Labels, mutatedSecret.Labels) {
		t.Fatalf("mutated secret labels deleted or overwritten: got %v want %v", storedSecret.Labels, mutatedSecret.Labels)
	}
}

func TestDeleteSecretIfCurrentUsesResourceVersionPrecondition(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
			UID:       "secret-uid-1",
			Labels:    map[string]string{"managed-by": "osok"},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret.DeepCopy()).Build()
	deleteClient := &capturingDeleteClient{Client: baseClient}
	client := NewWithReader(
		deleteClient,
		baseClient,
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		newTestMetrics(),
	)

	current, err := client.GetSecretRecord(context.Background(), existingSecret.Name, existingSecret.Namespace)
	if err != nil {
		t.Fatalf("get secret record: %v", err)
	}

	refreshedSecret := &corev1.Secret{}
	if err := baseClient.Get(context.Background(), ctrlclient.ObjectKey{Name: existingSecret.Name, Namespace: existingSecret.Namespace}, refreshedSecret); err != nil {
		t.Fatalf("load refreshed secret: %v", err)
	}

	deleted, err := client.DeleteSecretIfCurrent(context.Background(), existingSecret.Name, existingSecret.Namespace, current)
	if err != nil {
		t.Fatalf("guarded delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected guarded delete to report success")
	}
	if deleteClient.deleteCalls != 1 {
		t.Fatalf("Delete() calls = %d, want 1", deleteClient.deleteCalls)
	}
	requireDeletePreconditions(t, deleteClient.deletePreconditions, refreshedSecret)
}
