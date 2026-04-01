/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package stream

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	streamingsdk "github.com/oracle/oci-go-sdk/v65/streaming"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const streamEndpointSecretOwnerUIDLabel = "streaming.oracle.com/stream-uid"

func init() {
	generatedFactory := newStreamServiceClient
	newStreamServiceClient = func(manager *StreamServiceManager) StreamServiceClient {
		return newStreamEndpointSecretClient(manager, generatedFactory(manager))
	}
}

type streamEndpointSecretRecordReader interface {
	GetSecretRecord(context.Context, string, string) (credhelper.SecretRecord, error)
}

type streamEndpointSecretClient struct {
	delegate           StreamServiceClient
	credentialClient   credhelper.CredentialClient
	secretRecordReader streamEndpointSecretRecordReader
	loadStream         func(context.Context, shared.OCID) (*streamingsdk.Stream, error)
}

var _ StreamServiceClient = streamEndpointSecretClient{}

func newStreamEndpointSecretClient(manager *StreamServiceManager, delegate StreamServiceClient) StreamServiceClient {
	client := streamEndpointSecretClient{
		delegate:         delegate,
		credentialClient: manager.CredentialClient,
		loadStream: func(ctx context.Context, streamID shared.OCID) (*streamingsdk.Stream, error) {
			sdkClient, err := streamingsdk.NewStreamAdminClientWithConfigurationProvider(manager.Provider)
			if err != nil {
				return nil, fmt.Errorf("initialize Stream endpoint secret OCI client: %w", err)
			}

			response, err := sdkClient.GetStream(ctx, streamingsdk.GetStreamRequest{
				StreamId: common.String(string(streamID)),
			})
			if err != nil {
				return nil, err
			}
			return &response.Stream, nil
		},
	}
	if recordReader, ok := manager.CredentialClient.(streamEndpointSecretRecordReader); ok {
		client.secretRecordReader = recordReader
	}
	return client
}

func (c streamEndpointSecretClient) CreateOrUpdate(
	ctx context.Context,
	resource *streamingv1beta1.Stream,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !streamReadyForEndpointSecret(resource) {
		return response, err
	}

	if err := c.syncEndpointSecret(ctx, resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return response, nil
}

func (c streamEndpointSecretClient) Delete(
	ctx context.Context,
	resource *streamingv1beta1.Stream,
) (bool, error) {
	deleted, err := c.delegate.Delete(ctx, resource)
	if err != nil || !deleted {
		return deleted, err
	}

	if err := c.deleteEndpointSecret(ctx, resource); err != nil {
		return deleted, err
	}
	return deleted, nil
}

func streamReadyForEndpointSecret(resource *streamingv1beta1.Stream) bool {
	if resource == nil {
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

func (c streamEndpointSecretClient) syncEndpointSecret(ctx context.Context, resource *streamingv1beta1.Stream) error {
	if c.credentialClient == nil {
		return fmt.Errorf("stream endpoint secret credential client is not configured")
	}
	if c.secretRecordReader == nil {
		return fmt.Errorf("stream endpoint secret ownership checks require secret metadata reads")
	}

	streamID := resource.Status.OsokStatus.Ocid
	if strings.TrimSpace(string(streamID)) == "" {
		return fmt.Errorf("stream endpoint secret sync requires a tracked stream OCID")
	}
	ownerLabels, err := streamEndpointSecretLabels(resource)
	if err != nil {
		return err
	}

	stream, err := c.loadStream(ctx, streamID)
	if err != nil {
		return err
	}

	data, err := streamEndpointSecretData(*stream)
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

func (c streamEndpointSecretClient) createEndpointSecret(
	ctx context.Context,
	resource *streamingv1beta1.Stream,
	ownerLabels map[string]string,
	data map[string][]byte,
) error {
	_, err := c.credentialClient.CreateSecret(ctx, resource.Name, resource.Namespace, ownerLabels, data)
	switch {
	case err == nil:
		return nil
	case apierrors.IsAlreadyExists(err):
		// Manager-backed clients can observe a stale NotFound on the cached read while the direct create
		// already sees the Secret. Re-read and converge so repeat ACTIVE reconciles stay idempotent.
		currentRecord, rereadErr := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
		if rereadErr != nil {
			return rereadErr
		}
		return c.syncExistingEndpointSecret(ctx, resource, currentRecord, data)
	default:
		return err
	}
}

func (c streamEndpointSecretClient) syncExistingEndpointSecret(
	ctx context.Context,
	resource *streamingv1beta1.Stream,
	currentRecord credhelper.SecretRecord,
	desiredData map[string][]byte,
) error {
	owned, err := streamOwnsEndpointSecret(resource, currentRecord.Labels)
	if err != nil {
		return err
	}
	if !owned {
		return fmt.Errorf(
			"stream endpoint secret %s/%s is not owned by Stream UID %q",
			resource.Namespace,
			resource.Name,
			resource.UID,
		)
	}
	if reflect.DeepEqual(currentRecord.Data, desiredData) {
		return nil
	}

	_, err = c.credentialClient.UpdateSecret(ctx, resource.Name, resource.Namespace, nil, desiredData)
	return err
}

func streamEndpointSecretData(stream streamingsdk.Stream) (map[string][]byte, error) {
	if stream.MessagesEndpoint == nil || strings.TrimSpace(*stream.MessagesEndpoint) == "" {
		return nil, fmt.Errorf("stream messagesEndpoint is not available")
	}

	return map[string][]byte{
		"endpoint": []byte(*stream.MessagesEndpoint),
	}, nil
}

func (c streamEndpointSecretClient) deleteEndpointSecret(ctx context.Context, resource *streamingv1beta1.Stream) error {
	if c.credentialClient == nil {
		return fmt.Errorf("stream endpoint secret credential client is not configured")
	}
	if c.secretRecordReader == nil {
		return nil
	}

	record, err := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
	switch {
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		return err
	}

	owned, err := streamOwnsEndpointSecret(resource, record.Labels)
	if err != nil {
		return err
	}
	if !owned {
		return nil
	}

	_, err = c.credentialClient.DeleteSecret(ctx, resource.Name, resource.Namespace)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func streamEndpointSecretLabels(resource *streamingv1beta1.Stream) (map[string]string, error) {
	ownerUID, err := streamEndpointSecretOwnerUID(resource)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		streamEndpointSecretOwnerUIDLabel: ownerUID,
	}, nil
}

func streamOwnsEndpointSecret(resource *streamingv1beta1.Stream, labels map[string]string) (bool, error) {
	ownerUID, err := streamEndpointSecretOwnerUID(resource)
	if err != nil {
		return false, err
	}
	return labels[streamEndpointSecretOwnerUIDLabel] == ownerUID, nil
}

func streamEndpointSecretOwnerUID(resource *streamingv1beta1.Stream) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("stream endpoint secret ownership requires a Stream resource")
	}
	ownerUID := strings.TrimSpace(string(resource.UID))
	if ownerUID == "" {
		return "", fmt.Errorf("stream endpoint secret ownership requires a Stream UID")
	}
	return ownerUID, nil
}
