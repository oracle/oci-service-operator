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

func init() {
	generatedFactory := newStreamServiceClient
	newStreamServiceClient = func(manager *StreamServiceManager) StreamServiceClient {
		return newStreamEndpointSecretClient(manager, generatedFactory(manager))
	}
}

type streamEndpointSecretClient struct {
	delegate         StreamServiceClient
	credentialClient credhelper.CredentialClient
	loadStream       func(context.Context, shared.OCID) (*streamingsdk.Stream, error)
}

var _ StreamServiceClient = streamEndpointSecretClient{}

func newStreamEndpointSecretClient(manager *StreamServiceManager, delegate StreamServiceClient) StreamServiceClient {
	return streamEndpointSecretClient{
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

	streamID := resource.Status.OsokStatus.Ocid
	if strings.TrimSpace(string(streamID)) == "" {
		return fmt.Errorf("stream endpoint secret sync requires a tracked stream OCID")
	}

	stream, err := c.loadStream(ctx, streamID)
	if err != nil {
		return err
	}

	data, err := streamEndpointSecretData(*stream)
	if err != nil {
		return err
	}

	currentData, err := c.credentialClient.GetSecret(ctx, resource.Name, resource.Namespace)
	switch {
	case err == nil:
		if reflect.DeepEqual(currentData, data) {
			return nil
		}
		_, err = c.credentialClient.UpdateSecret(ctx, resource.Name, resource.Namespace, nil, data)
		return err
	case apierrors.IsNotFound(err):
		_, err = c.credentialClient.CreateSecret(ctx, resource.Name, resource.Namespace, nil, data)
		return err
	default:
		return err
	}
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

	_, err := c.credentialClient.DeleteSecret(ctx, resource.Name, resource.Namespace)
	return err
}
