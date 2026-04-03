/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const queueEndpointSecretOwnerUIDLabel = "queue.oracle.com/queue-uid"

func init() {
	runtimeFactory := newQueueServiceClient
	newQueueServiceClient = func(manager *QueueServiceManager) QueueServiceClient {
		return newQueueEndpointSecretClient(manager, runtimeFactory(manager))
	}
}

type queueEndpointSecretRecordReader interface {
	GetSecretRecord(context.Context, string, string) (credhelper.SecretRecord, error)
}

type queueEndpointSecretClient struct {
	delegate             QueueServiceClient
	credentialClient     credhelper.CredentialClient
	secretRecordReader   queueEndpointSecretRecordReader
	guardedSecretMutator credhelper.GuardedSecretMutator
}

var _ QueueServiceClient = queueEndpointSecretClient{}

func newQueueEndpointSecretClient(manager *QueueServiceManager, delegate QueueServiceClient) QueueServiceClient {
	client := queueEndpointSecretClient{
		delegate:         delegate,
		credentialClient: manager.CredentialClient,
	}
	if recordReader, ok := manager.CredentialClient.(queueEndpointSecretRecordReader); ok {
		client.secretRecordReader = recordReader
	}
	if guardedMutator, ok := manager.CredentialClient.(credhelper.GuardedSecretMutator); ok {
		client.guardedSecretMutator = guardedMutator
	}
	return client
}

func (c queueEndpointSecretClient) CreateOrUpdate(
	ctx context.Context,
	resource *queuev1beta1.Queue,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !queueReadyForEndpointSecret(resource) {
		return response, err
	}

	if err := c.syncEndpointSecret(ctx, resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return response, nil
}

func (c queueEndpointSecretClient) Delete(ctx context.Context, resource *queuev1beta1.Queue) (bool, error) {
	deleted, err := c.delegate.Delete(ctx, resource)
	if err != nil || !deleted {
		return deleted, err
	}

	if err := c.deleteEndpointSecret(ctx, resource); err != nil {
		return deleted, err
	}
	return deleted, nil
}

func queueReadyForEndpointSecret(resource *queuev1beta1.Queue) bool {
	if resource == nil || strings.TrimSpace(resource.Status.MessagesEndpoint) == "" {
		return false
	}
	if resource.Status.OsokStatus.Reason == string(shared.Active) {
		return true
	}
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return false
	}
	return conditions[len(conditions)-1].Type == shared.Active
}

func (c queueEndpointSecretClient) syncEndpointSecret(ctx context.Context, resource *queuev1beta1.Queue) error {
	if c.credentialClient == nil {
		return fmt.Errorf("queue endpoint secret credential client is not configured")
	}
	if c.secretRecordReader == nil {
		return fmt.Errorf("queue endpoint secret ownership checks require secret metadata reads")
	}
	if c.guardedSecretMutator == nil {
		return fmt.Errorf("queue endpoint secret ownership checks require guarded secret mutations")
	}

	ownerLabels, err := queueEndpointSecretLabels(resource)
	if err != nil {
		return err
	}
	data, err := queueEndpointSecretData(resource)
	if err != nil {
		return err
	}

	currentRecord, err := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
	if err == nil {
		return c.syncExistingEndpointSecret(ctx, resource, currentRecord, data)
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	return c.createEndpointSecret(ctx, resource, ownerLabels, data)
}

func (c queueEndpointSecretClient) createEndpointSecret(
	ctx context.Context,
	resource *queuev1beta1.Queue,
	ownerLabels map[string]string,
	data map[string][]byte,
) error {
	_, err := c.credentialClient.CreateSecret(ctx, resource.Name, resource.Namespace, ownerLabels, data)
	switch {
	case err == nil:
		return nil
	case apierrors.IsAlreadyExists(err):
		currentRecord, rereadErr := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
		if rereadErr != nil {
			return rereadErr
		}
		return c.syncExistingEndpointSecret(ctx, resource, currentRecord, data)
	default:
		return err
	}
}

func (c queueEndpointSecretClient) syncExistingEndpointSecret(
	ctx context.Context,
	resource *queuev1beta1.Queue,
	currentRecord credhelper.SecretRecord,
	desiredData map[string][]byte,
) error {
	owned, err := queueOwnsEndpointSecret(resource, currentRecord.Labels)
	if err != nil {
		return err
	}
	if owned {
		if reflect.DeepEqual(currentRecord.Data, desiredData) {
			return nil
		}
		_, err = c.guardedSecretMutator.UpdateSecretIfCurrent(ctx, resource.Name, resource.Namespace, currentRecord, nil, desiredData)
		return err
	}

	adoptionLabels, adoptable, err := queueLegacyEndpointSecretAdoptionLabels(resource, currentRecord, desiredData)
	if err != nil {
		return err
	}
	if !adoptable {
		return fmt.Errorf("queue endpoint secret %s/%s is not owned by Queue UID %q", resource.Namespace, resource.Name, resource.UID)
	}

	_, err = c.guardedSecretMutator.UpdateSecretIfCurrent(ctx, resource.Name, resource.Namespace, currentRecord, adoptionLabels, desiredData)
	return err
}

func queueEndpointSecretData(resource *queuev1beta1.Queue) (map[string][]byte, error) {
	if resource == nil || strings.TrimSpace(resource.Status.MessagesEndpoint) == "" {
		return nil, fmt.Errorf("queue messagesEndpoint is not available")
	}
	return map[string][]byte{
		"endpoint": []byte(resource.Status.MessagesEndpoint),
	}, nil
}

func (c queueEndpointSecretClient) deleteEndpointSecret(ctx context.Context, resource *queuev1beta1.Queue) error {
	if c.credentialClient == nil {
		return fmt.Errorf("queue endpoint secret credential client is not configured")
	}
	if c.secretRecordReader == nil {
		return nil
	}
	if c.guardedSecretMutator == nil {
		return fmt.Errorf("queue endpoint secret ownership checks require guarded secret mutations")
	}

	record, err := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
	switch {
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		return err
	}

	owned, err := queueOwnsEndpointSecret(resource, record.Labels)
	if err != nil {
		return err
	}
	if !owned {
		return nil
	}

	_, err = c.guardedSecretMutator.DeleteSecretIfCurrent(ctx, resource.Name, resource.Namespace, record)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func queueEndpointSecretLabels(resource *queuev1beta1.Queue) (map[string]string, error) {
	ownerUID, err := queueEndpointSecretOwnerUID(resource)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		queueEndpointSecretOwnerUIDLabel: ownerUID,
	}, nil
}

func queueOwnsEndpointSecret(resource *queuev1beta1.Queue, labels map[string]string) (bool, error) {
	ownerUID, err := queueEndpointSecretOwnerUID(resource)
	if err != nil {
		return false, err
	}
	return labels[queueEndpointSecretOwnerUIDLabel] == ownerUID, nil
}

func queueLegacyEndpointSecretAdoptionLabels(
	resource *queuev1beta1.Queue,
	currentRecord credhelper.SecretRecord,
	desiredData map[string][]byte,
) (map[string]string, bool, error) {
	if strings.TrimSpace(currentRecord.Labels[queueEndpointSecretOwnerUIDLabel]) != "" {
		return nil, false, nil
	}
	if !reflect.DeepEqual(currentRecord.Data, desiredData) {
		return nil, false, nil
	}

	ownerLabels, err := queueEndpointSecretLabels(resource)
	if err != nil {
		return nil, false, err
	}
	return mergeQueueEndpointSecretLabels(currentRecord.Labels, ownerLabels), true, nil
}

func mergeQueueEndpointSecretLabels(existing map[string]string, updates map[string]string) map[string]string {
	merged := make(map[string]string, len(existing)+len(updates))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range updates {
		merged[key] = value
	}
	return merged
}

func queueEndpointSecretOwnerUID(resource *queuev1beta1.Queue) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("queue endpoint secret ownership requires a Queue resource")
	}
	ownerUID := strings.TrimSpace(string(resource.UID))
	if ownerUID == "" {
		return "", fmt.Errorf("queue endpoint secret ownership requires a Queue UID")
	}
	return ownerUID, nil
}
