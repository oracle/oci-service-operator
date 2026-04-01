/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kubesecret

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeSecretClient struct {
	Client  client.Client
	Reader  client.Reader
	Log     loggerutil.OSOKLogger
	Metrics *metrics.Metrics
}

var _ credhelper.SecretRecordReader = (*KubeSecretClient)(nil)
var _ credhelper.GuardedSecretMutator = (*KubeSecretClient)(nil)

func New(kubeClient client.Client, logger loggerutil.OSOKLogger, metrics *metrics.Metrics) *KubeSecretClient {
	return NewWithReader(kubeClient, kubeClient, logger, metrics)
}

func NewWithReader(kubeClient client.Client, reader client.Reader, logger loggerutil.OSOKLogger, metrics *metrics.Metrics) *KubeSecretClient {
	if reader == nil {
		reader = kubeClient
	}
	return &KubeSecretClient{
		Client:  kubeClient,
		Reader:  reader,
		Log:     logger,
		Metrics: metrics,
	}
}

func (c *KubeSecretClient) CreateSecret(ctx context.Context, secretName string, secretNamespace string,
	labels map[string]string, data map[string][]byte) (bool, error) {
	newSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
			Labels:    labels,
		},
		Data: data,
	}

	c.Log.InfoLog("Creating Kubernetes Secret", "newSecret.Namespace", newSecret.Namespace, "newSecret.Name", newSecret.Name)
	if err := c.Client.Create(ctx, newSecret); err != nil {
		if errors.IsAlreadyExists(err) {
			c.Log.InfoLog("Secret already exists with provided details, Not creating a new Secret",
				"newSecret.Namespace", newSecret.Namespace, "newSecret.Name", newSecret.Name)
		}
		return false, err
	}
	c.Metrics.AddSecretCountMetrics(ctx, "kubesecretclient", "New Secret got created", secretName, secretNamespace)
	c.Log.InfoLog("Secret Created successfully", "Secret Name", newSecret.Name)
	return true, nil
}

func (c *KubeSecretClient) DeleteSecret(ctx context.Context, secretName string, secretNamespace string) (bool, error) {
	existingSecret, err := c.getSecretObject(ctx, secretName, secretNamespace)
	if err != nil {
		c.Log.ErrorLog(err, "error getting Kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	err = c.deleteSecretObject(ctx, existingSecret)
	if err != nil {
		c.Log.ErrorLog(err, "error deleting Kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	c.Log.InfoLog("Secret deleted successfully", "Secret Name", secretName, "Secret Namespace", secretNamespace)
	return true, nil
}

func (c *KubeSecretClient) DeleteSecretIfCurrent(
	ctx context.Context,
	secretName string,
	secretNamespace string,
	current credhelper.SecretRecord,
) (bool, error) {
	existingSecret, err := c.getSecretObject(ctx, secretName, secretNamespace)
	if err != nil {
		c.Log.ErrorLog(err, "error getting Kubernetes secret before guarded delete", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	if err := validateSecretIdentity(secretName, secretNamespace, current, existingSecret); err != nil {
		c.Log.ErrorLog(err, "guarded delete rejected because the Kubernetes secret changed", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	if err := c.deleteSecretObject(ctx, existingSecret); err != nil {
		c.Log.ErrorLog(err, "error deleting Kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	c.Log.InfoLog("Secret deleted successfully after guarded identity check", "Secret Name", secretName, "Secret Namespace", secretNamespace)
	return true, nil
}

func (c *KubeSecretClient) GetSecret(ctx context.Context, secretName string, secretNamespace string) (map[string][]byte, error) {
	record, err := c.GetSecretRecord(ctx, secretName, secretNamespace)
	if err != nil {
		return map[string][]byte{}, err
	}
	return record.Data, nil
}

func (c *KubeSecretClient) GetSecretRecord(ctx context.Context, secretName string, secretNamespace string) (credhelper.SecretRecord, error) {
	existingSecret, err := c.getSecretObject(ctx, secretName, secretNamespace)
	if err != nil {
		c.Log.ErrorLog(err, "error getting Kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return credhelper.SecretRecord{}, err
	}

	c.Log.InfoLog("Secret retrieved successfully", "Secret Name", existingSecret.Name, "Secret Namespace", existingSecret.Namespace)
	return credhelper.SecretRecord{
		UID:    existingSecret.UID,
		Labels: cloneLabels(existingSecret.Labels),
		Data:   cloneSecretData(existingSecret.Data),
	}, nil
}

func (c *KubeSecretClient) UpdateSecret(ctx context.Context, secretName string, secretNamespace string, labels map[string]string,
	updatedData map[string][]byte) (bool, error) {
	existingSecret, err := c.getSecretObject(ctx, secretName, secretNamespace)
	if err != nil {
		c.Log.ErrorLog(err, "Failed to get kubernetes secret before update", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	err = c.updateSecretObject(ctx, existingSecret, secretName, secretNamespace, labels, updatedData)
	if err != nil {
		c.Log.ErrorLog(err, "Failed to update kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	c.Log.InfoLog("Secret updated successfully", "Secret Name", secretName, "Secret Namespace", secretNamespace)
	return true, nil
}

func (c *KubeSecretClient) UpdateSecretIfCurrent(
	ctx context.Context,
	secretName string,
	secretNamespace string,
	current credhelper.SecretRecord,
	labels map[string]string,
	updatedData map[string][]byte,
) (bool, error) {
	existingSecret, err := c.getSecretObject(ctx, secretName, secretNamespace)
	if err != nil {
		c.Log.ErrorLog(err, "Failed to get kubernetes secret before guarded update", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	if err := validateSecretIdentity(secretName, secretNamespace, current, existingSecret); err != nil {
		c.Log.ErrorLog(err, "guarded update rejected because the Kubernetes secret changed", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	err = c.updateSecretObject(ctx, existingSecret, secretName, secretNamespace, labels, updatedData)
	if err != nil {
		c.Log.ErrorLog(err, "Failed to update kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	c.Log.InfoLog("Secret updated successfully after guarded identity check", "Secret Name", secretName, "Secret Namespace", secretNamespace)
	return true, nil
}

func (c *KubeSecretClient) updateSecretObject(
	ctx context.Context,
	existingSecret *v1.Secret,
	secretName string,
	secretNamespace string,
	labels map[string]string,
	updatedData map[string][]byte,
) error {
	if labels != nil {
		existingSecret.Labels = labels
	}
	existingSecret.Data = updatedData
	return c.Client.Update(ctx, existingSecret)
}

func (c *KubeSecretClient) deleteSecretObject(ctx context.Context, existingSecret *v1.Secret) error {
	uid := existingSecret.UID
	return c.Client.Delete(ctx, existingSecret, client.Preconditions{UID: &uid})
}

func (c *KubeSecretClient) reader() client.Reader {
	if c.Reader != nil {
		return c.Reader
	}
	return c.Client
}

func (c *KubeSecretClient) getSecretObject(ctx context.Context, secretName string, secretNamespace string) (*v1.Secret, error) {
	existingSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}
	if err := c.reader().Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, existingSecret); err != nil {
		return nil, err
	}
	return existingSecret, nil
}

func cloneLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}

func cloneSecretData(data map[string][]byte) map[string][]byte {
	if data == nil {
		return nil
	}
	cloned := make(map[string][]byte, len(data))
	for key, value := range data {
		cloned[key] = append([]byte(nil), value...)
	}
	return cloned
}

func validateSecretIdentity(
	secretName string,
	secretNamespace string,
	current credhelper.SecretRecord,
	existingSecret *v1.Secret,
) error {
	expectedUID := strings.TrimSpace(string(current.UID))
	if expectedUID == "" {
		return fmt.Errorf("guarded secret mutation requires the previously read Secret UID for %s/%s", secretNamespace, secretName)
	}
	if existingSecret == nil {
		return fmt.Errorf("guarded secret mutation requires the current Kubernetes Secret for %s/%s", secretNamespace, secretName)
	}
	actualUID := strings.TrimSpace(string(existingSecret.UID))
	if actualUID != expectedUID {
		return errors.NewConflict(
			v1.Resource("secret"),
			secretName,
			fmt.Errorf("secret %s/%s changed UID from %q to %q", secretNamespace, secretName, expectedUID, actualUID),
		)
	}
	return nil
}
