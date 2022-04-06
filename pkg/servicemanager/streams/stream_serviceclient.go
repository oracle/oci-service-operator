/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
	"github.com/pkg/errors"
	"reflect"
	"strings"
)

func getStreamClient(provider common.ConfigurationProvider) streaming.StreamAdminClient {
	streamClient, _ := streaming.NewStreamAdminClientWithConfigurationProvider(provider)
	return streamClient
}

func (c *StreamServiceManager) CreateStream(ctx context.Context, stream ociv1beta1.Stream) (streaming.CreateStreamResponse, error) {

	streamClient := getStreamClient(c.Provider)
	c.Log.DebugLog("Creating Stream ", "name", stream.Spec.Name)

	createStreamDetails := streaming.CreateStreamDetails{
		Name:       common.String(stream.Spec.Name),
		Partitions: common.Int(stream.Spec.Partitions),
	}

	if stream.Spec.StreamPoolId != "" {
		createStreamDetails.StreamPoolId = common.String(string(stream.Spec.StreamPoolId))
	}

	if stream.Spec.CompartmentId != "" {
		createStreamDetails.CompartmentId = common.String(string(stream.Spec.CompartmentId))
	}

	if stream.Spec.RetentionInHours > 0 {
		createStreamDetails.RetentionInHours = common.Int(stream.Spec.RetentionInHours)
	}

	createStreamRequest := streaming.CreateStreamRequest{
		CreateStreamDetails: createStreamDetails,
	}

	return streamClient.CreateStream(ctx, createStreamRequest)
}

func (c *StreamServiceManager) GetStreamOcid(ctx context.Context, stream ociv1beta1.Stream) (*ociv1beta1.OCID, error) {

	streamClient := getStreamClient(c.Provider)
	listStreamsRequest := streaming.ListStreamsRequest{
		Name: common.String(stream.Spec.Name),
	}

	if string(stream.Spec.StreamPoolId) != "" {
		listStreamsRequest.StreamPoolId = common.String(string(stream.Spec.StreamPoolId))
	}

	if string(stream.Spec.CompartmentId) != "" {
		listStreamsRequest.CompartmentId = common.String(string(stream.Spec.CompartmentId))
	}

	listStreamsResponse, err := streamClient.ListStreams(ctx, listStreamsRequest)

	if err != nil {
		c.Log.ErrorLog(err, "Error while listing Stream")
		return nil, err
	}

	return c.GetCreateOrUpdateStream(listStreamsResponse, stream)
}

func (c *StreamServiceManager) DeleteStream(ctx context.Context, stream ociv1beta1.Stream) (streaming.DeleteStreamResponse, error) {
	streamClient := getStreamClient(c.Provider)
	c.Log.InfoLog("Deleting Stream ", "name", stream.Spec.Name)

	deleteStreamRequest := streaming.DeleteStreamRequest{
		StreamId: common.String(string(stream.Spec.StreamId)),
	}

	return streamClient.DeleteStream(ctx, deleteStreamRequest)
}

func (c *StreamServiceManager) GetStream(ctx context.Context, streamId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*streaming.Stream, error) {

	streamClient := getStreamClient(c.Provider)

	getStreamRequest := streaming.GetStreamRequest{
		StreamId: common.String(string(streamId)),
	}

	if retryPolicy != nil {
		getStreamRequest.RequestMetadata.RetryPolicy = retryPolicy
	}

	response, err := streamClient.GetStream(ctx, getStreamRequest)
	if err != nil {
		return nil, err
	}

	return &response.Stream, nil
}

func (c *StreamServiceManager) UpdateStream(ctx context.Context, stream *ociv1beta1.Stream) error {

	streamClient := getStreamClient(c.Provider)

	existingStream, err := c.GetStream(ctx, stream.Spec.StreamId, nil)
	if err != nil {
		return err
	}

	if stream.Spec.Partitions <= 0 || stream.Spec.Partitions != *existingStream.Partitions {
		return errors.New("Partitions can't be updated")
	}

	if stream.Spec.RetentionInHours <= 23 || stream.Spec.RetentionInHours != *existingStream.RetentionInHours {
		return errors.New("RetentionsHours can't be updated")
	}

	updateStreamDetails := streaming.UpdateStreamDetails{}
	updateNeeded := false

	if stream.Spec.StreamPoolId != "" && string(stream.Spec.StreamPoolId) != *existingStream.StreamPoolId {
		updateStreamDetails.StreamPoolId = common.String(strings.TrimSpace(string(stream.Spec.StreamPoolId)))
		updateNeeded = true
	}

	if stream.Spec.FreeFormTags != nil && !reflect.DeepEqual(existingStream.FreeformTags, stream.Spec.FreeFormTags) {
		updateStreamDetails.FreeformTags = stream.Spec.FreeFormTags
		updateNeeded = true
	}

	if stream.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&stream.Spec.DefinedTags); !reflect.DeepEqual(existingStream.DefinedTags, defTag) {
			updateStreamDetails.DefinedTags = defTag
			updateNeeded = true
		}
	}

	if updateNeeded {
		updateAutonomousDatabaseRequest := streaming.UpdateStreamRequest{
			StreamId:            common.String(string(stream.Spec.StreamId)),
			UpdateStreamDetails: updateStreamDetails,
		}

		if _, err := streamClient.UpdateStream(ctx, updateAutonomousDatabaseRequest); err != nil {
			return err
		}
	}

	return nil

}

func (c *StreamServiceManager) GetStreamOCID(ctx context.Context, stream ociv1beta1.Stream, status string) (*ociv1beta1.OCID, error) {

	if status == "CREATE" {
		listResponse, err := c.GetListOfStreams(ctx, stream)

		if err != nil {
			return nil, err
		}

		return c.GetCreateOrUpdateStream(listResponse, stream)
	} else {

		listResponse, err := c.GetListOfStreams(ctx, stream)

		if err != nil {
			return nil, err
		}

		return c.GetFailedOrDeleteStream(listResponse, stream)
	}
}

func (c *StreamServiceManager) GetListOfStreams(ctx context.Context, stream ociv1beta1.Stream) (streaming.ListStreamsResponse, error) {

	streamClient := getStreamClient(c.Provider)
	listStreamsRequest := streaming.ListStreamsRequest{
		Name:  common.String(stream.Spec.Name),
		Limit: common.Int(1),
	}

	if string(stream.Spec.StreamPoolId) != "" {
		listStreamsRequest.StreamPoolId = common.String(string(stream.Spec.StreamPoolId))
	}

	if string(stream.Spec.CompartmentId) != "" {
		listStreamsRequest.CompartmentId = common.String(string(stream.Spec.CompartmentId))
	}

	listStreamsResponse, err := streamClient.ListStreams(ctx, listStreamsRequest)

	if err != nil {
		c.Log.ErrorLog(err, "Error while listing Stream")
		return listStreamsResponse, err
	}

	return listStreamsResponse, nil
}

func (c *StreamServiceManager) GetFailedOrDeleteStream(listStreamsResponse streaming.ListStreamsResponse, stream ociv1beta1.Stream) (*ociv1beta1.OCID, error) {

	if len(listStreamsResponse.Items) > 0 {
		status := listStreamsResponse.Items[0].LifecycleState
		if status == "DELETED" || status == "DELETING" || status == "FAILED" {

			c.Log.DebugLog(fmt.Sprintf("Stream %s exists in GetFailedOrDeletingStream", stream.Spec.Name))

			return (*ociv1beta1.OCID)(listStreamsResponse.Items[0].Id), nil
		}
	}
	c.Log.DebugLog(fmt.Sprintf("Stream %s does not exist.", stream.Spec.Name))
	return nil, nil
}

func (c *StreamServiceManager) GetCreateOrUpdateStream(listStreamsResponse streaming.ListStreamsResponse, stream ociv1beta1.Stream) (*ociv1beta1.OCID, error) {

	if len(listStreamsResponse.Items) > 0 {
		c.Log.DebugLog(fmt.Sprintf(
			"Number of streams with same name %d ",
			len(listStreamsResponse.Items),
		))
		for entry := 0; entry < len(listStreamsResponse.Items); entry++ {
			status := listStreamsResponse.Items[entry].LifecycleState
			if status == "ACTIVE" || status == "CREATING" || status == "UPDATING" {

				c.Log.DebugLog(fmt.Sprintf("Stream %s exists.", stream.Spec.Name))

				return (*ociv1beta1.OCID)(listStreamsResponse.Items[entry].Id), nil
			}
		}

	}
	c.Log.DebugLog(fmt.Sprintf("Stream %s does not exist.", stream.Spec.Name))
	return nil, nil
}
