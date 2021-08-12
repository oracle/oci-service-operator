/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kubesecret

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type KubeSecretClient struct {
	Client  client.Client
	Log     loggerutil.OSOKLogger
	Metrics *metrics.Metrics
}

func New(client client.Client, logger loggerutil.OSOKLogger, metrics *metrics.Metrics) *KubeSecretClient {
	return &KubeSecretClient{
		Client:  client,
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

	currentSecret := &v1.Secret{}
	err := c.Client.Get(ctx, types.NamespacedName{Name: newSecret.Name, Namespace: newSecret.Namespace}, currentSecret)
	if err == nil {
		c.Log.InfoLog("Secret already exists with provided details, Not creating a new Secret",
			"newSecret.Namespace", newSecret.Namespace, "newSecret.Name", newSecret.Name)
		return false, errors.NewAlreadyExists(v1.Resource("secret"), secretName)
	}

	if errors.IsNotFound(err) {
		c.Log.InfoLog("Secret does not exist, Creating a new Secret", "newSecret.Namespace", newSecret.Namespace, "newSecret.Name", newSecret.Name)
		if err = c.Client.Create(ctx, newSecret); err != nil {
			return false, err
		}
		c.Metrics.AddSecretCountMetrics("kubesecretclient", "New Secret got created", secretName, secretNamespace)
		c.Log.InfoLog("Secret Created successfully", "Secret Name", newSecret.Name)
		return true, nil
	} else {
		return false, err
	}
}

func (c *KubeSecretClient) DeleteSecret(ctx context.Context, secretName string, secretNamespace string) (bool, error) {
	existingSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}
	err := c.Client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, existingSecret)
	if err != nil {
		c.Log.ErrorLog(err, "error getting Kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	err = c.Client.Delete(ctx, existingSecret)
	if err != nil {
		c.Log.ErrorLog(err, "error deleting Kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	c.Log.InfoLog("Secret deleted successfully", "Secret Name", secretName, "Secret Namespace", secretNamespace)
	return true, nil
}

func (c *KubeSecretClient) GetSecret(ctx context.Context, secretName string, secretNamespace string) (map[string][]byte, error) {
	data := map[string][]byte{}

	existingSecret := &v1.Secret{}
	err := c.Client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, existingSecret)
	if err != nil {
		c.Log.ErrorLog(err, "error getting Kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return data, err
	}

	c.Log.InfoLog("Secret retrieved successfully", "Secret Name", existingSecret.Name, "Secret Namespace", existingSecret.Namespace)
	for k, v := range existingSecret.Data {
		data[k] = v
	}
	return data, nil
}

func (c *KubeSecretClient) UpdateSecret(ctx context.Context, secretName string, secretNamespace string, labels map[string]string,
	updatedData map[string][]byte) (bool, error) {
	existingSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
			Labels:    labels,
		},
		Data: updatedData,
	}
	err := c.Client.Update(ctx, existingSecret)
	if err != nil {
		c.Log.ErrorLog(err, "Failed to update kubernetes secret", "Secret Name", secretName, "Secret Namespace", secretNamespace)
		return false, err
	}
	return true, nil
}

/***
This method is used to convert the given secret name into lowercase and validate it against the kubernetes secret naming conventions
https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
*/

func (c *KubeSecretClient) isValidSecretName(ctx context.Context, secretName string) bool {
	validationResults := validation.NameIsDNSSubdomain(secretName, false)
	if validationResults != nil && len(validationResults) > 0 {
		return false
	}
	return true
}
func (c *KubeSecretClient) getValidSecretName(ctx context.Context, secretName string) string {
	secretName = strings.ToLower(secretName)

	regex := regexp.MustCompile("[^a-z0-9-.]+")
	secretName = regex.ReplaceAllString(secretName, "")

	consecutiveCharRegex := regexp.MustCompile("[-.]{2,}")
	secretName = consecutiveCharRegex.ReplaceAllString(secretName, ".")

	/** ToDo
	Add length validation and trim the length to 256 chars, if it is more
	Check the beginning and end characters of the secret name as it should be an alphanumeric char
	*/

	c.Log.InfoLog("Updated secret name is ", "Secret Name", secretName)
	return secretName
}
