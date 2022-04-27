/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import (
	"context"
	"fmt"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"

	"github.com/pkg/errors"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type StreamServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
}

func NewStreamServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger, metrics *metrics.Metrics) *StreamServiceManager {
	return &StreamServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
		Metrics:          metrics,
	}
}

func (c *StreamServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	streamObject, err := c.convert(obj)

	// if error happen while object conversion
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var streamInstance *streaming.Stream

	if strings.TrimSpace(string(streamObject.Spec.StreamId)) == "" {

		if streamObject.Spec.Name == "" {
			return servicemanager.OSOKResponse{IsSuccessful: false}, errors.New("Can't able to create the stream")
		}

		// check for whether same name stream exists or not in ACTIVE, UPDATING OR CREATING Phase
		streamOcid, err := c.GetStreamOCID(ctx, *streamObject, "CREATE")
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting Stream using Id")
			c.Metrics.AddCRFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
				"Failed to get the Stream", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if streamOcid != nil {

			streamInstance, err = c.GetStream(ctx, *streamOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting Stream")
				c.Metrics.AddCRFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
					"Error while getting Stream", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			if isValidUpdate(*streamObject, *streamInstance) {
				if err = c.UpdateStream(ctx, streamObject); err != nil {
					c.Log.ErrorLog(err, "Error while updating Stream")
					c.Metrics.AddCRFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
						"Error while updating Stream", req.Name, req.Namespace)
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
					ociv1beta1.Updating, v1.ConditionTrue, "", "Stream Update success", c.Log)
				c.Log.InfoLog(fmt.Sprintf("Stream %s is updated successfully", *streamInstance.Name))
			}
		} else {

			//creating the fresh request for creating the Streams
			resp, err := c.CreateStream(ctx, *streamObject)
			if err != nil {
				streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Invalid Parameter Error")
				c.Metrics.AddCRFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
					"Invalid Parameter Error", req.Name, req.Namespace)
				_, err := errorutil.OciErrorTypeResponse(err)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Assertion error for BadRequestOciError")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				} else {
					streamObject.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
			}

			//create the stream then retry to become it active
			c.Log.InfoLog(fmt.Sprintf("Stream %s is getting Provisioned", streamObject.Spec.Name))
			streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "Stream is getting Provisioned", c.Log)
			retryPolicy := c.getStreamRetryPolicy(9)
			streamInstance, err = c.GetStream(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "Error while getting the stream", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Error while getting Stream")
				c.Metrics.AddCRFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
					"Error while getting Stream", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

		}

	} else {
		// stream already exists update the configuration or modify the changes
		// Bind CRD with an existing Stream.
		streamInstance, err = c.GetStream(ctx, streamObject.Spec.StreamId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting Stream")
			c.Metrics.AddCRFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
				"Error while getting Stream", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if isValidUpdate(*streamObject, *streamInstance) {
			if err = c.UpdateStream(ctx, streamObject); err != nil {
				c.Log.ErrorLog(err, "Error while updating Stream")
				c.Metrics.AddCRFaultMetrics(c.Metrics.ServiceName,
					"Error while updating Stream", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
				ociv1beta1.Updating, v1.ConditionTrue, "", "Stream Update success", c.Log)
			c.Log.InfoLog(fmt.Sprintf("Stream %s is updated successfully", *streamInstance.Name))
		} else {
			streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
				ociv1beta1.Active, v1.ConditionTrue, "", "Stream Bound success", c.Log)
			streamObject.Status.OsokStatus.Ocid = ociv1beta1.OCID(*streamInstance.Id)
			now := metav1.NewTime(time.Now())
			streamObject.Status.OsokStatus.CreatedAt = &now
			c.Log.InfoLog(fmt.Sprintf("Stream %s is bounded successfully", *streamInstance.Name))
		}

	}

	streamObject.Status.OsokStatus.Ocid = ociv1beta1.OCID(*streamInstance.Id)
	if streamObject.Status.OsokStatus.CreatedAt != nil {
		now := metav1.NewTime(time.Now())
		streamObject.Status.OsokStatus.CreatedAt = &now
	}

	if streamInstance.LifecycleState == "FAILED" {
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("Stream %s creation Failed", *streamInstance.Name), c.Log)
		c.Metrics.AddCRFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind, "Failed to Create the Stream",
			req.Name, req.Namespace)
		c.Log.InfoLog(fmt.Sprintf("Stream %s creation Failed", *streamInstance.Name))
	} else {
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("Stream %s is Active", *streamInstance.Name), c.Log)
		c.Log.InfoLog(fmt.Sprintf("Stream %s is Active", *streamInstance.Name))
		c.Metrics.AddCRSuccessMetrics(obj.GetObjectKind().GroupVersionKind().Kind, "Stream in Active state",
			req.Name, req.Namespace)
		_, err := c.addToSecret(ctx, streamObject.Namespace, streamObject.Name, *streamInstance)
		if err != nil {
			c.Log.InfoLog(fmt.Sprintf("Secret creation got failed"))
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func isValidUpdate(streamObject ociv1beta1.Stream, streamInstance streaming.Stream) bool {
	definedTagUpdated := false
	if streamObject.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&streamObject.Spec.DefinedTags); !reflect.DeepEqual(streamInstance.DefinedTags, defTag) {
			definedTagUpdated = true
		}
	}

	return streamObject.Spec.StreamPoolId != "" && string(streamObject.Spec.StreamPoolId) != *streamInstance.StreamPoolId ||
		streamObject.Spec.FreeFormTags != nil && !reflect.DeepEqual(streamObject.Spec.FreeFormTags, streamInstance.FreeformTags) ||
		definedTagUpdated
}

func (c *StreamServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	streamObject, err := c.convert(obj)

	if err != nil {
		c.Log.ErrorLog(err, "Error while converting the object")
		return true, nil
	}

	if strings.TrimSpace(string(streamObject.Spec.StreamId)) == "" {
		// if name and StreamID both are null or empty then delete is not possible
		if streamObject.Status.OsokStatus.Ocid != "" {
			c.Log.ErrorLog(err, "Deleted is not possible ocid not found")
			return true, nil
		}

		streamObject.Spec.StreamId = streamObject.Status.OsokStatus.Ocid
		streamOcid, err := c.GetStreamOcid(ctx, *streamObject)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting the stream ocid")
			return true, nil
		}
		//Error happened in while getting the StreamOcid
		if streamOcid == nil && err == nil {
			//check whether the given cluster is in failed state or not
			streamOcid, err = c.GetStreamOCID(ctx, *streamObject, "DELETE")

			if err == nil && streamOcid == nil {
				return true, nil
			}

			if err != nil || streamOcid == nil {
				c.Log.ErrorLog(err, "Error while getting the err")
				return true, err
			}
			c.Log.InfoLog(fmt.Sprintf("Stream OCID is %s ", *streamOcid))
			streamObject.Spec.StreamId = *streamOcid
		}

		if streamOcid != nil {
			c.Log.InfoLog(fmt.Sprintf("Stream OCID is %s ", *streamOcid))
			streamObject.Spec.StreamId = *streamOcid
		}
	}
	var streamInstance *streaming.Stream
	_, err = c.DeleteStream(ctx, *streamObject)
	if err != nil {
		c.Log.ErrorLog(err, "Error while Deleting the Stream")
		return true, nil
	}

	streamInstance, err = c.GetStream(ctx, streamObject.Spec.StreamId, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while Getting the Stream")
		c.Log.InfoLog(fmt.Sprintf("Error after calling the GetStream %s ", streamInstance.LifecycleState))
		return true, nil
	}
	if streamInstance.LifecycleState == "DELETED" || streamInstance.LifecycleState == "DELETING" {
		_, err := c.deleteFromSecret(ctx, streamObject.Namespace, streamObject.Name)

		if err != nil {
			c.Log.ErrorLog(err, "Secret deletion failed")
			return true, err
		}
		return true, nil
	}
	return true, nil
}

func (c *StreamServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {

	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *StreamServiceManager) convert(obj runtime.Object) (*ociv1beta1.Stream, error) {
	deepcopy, err := obj.(*ociv1beta1.Stream)
	if !err {
		return nil, fmt.Errorf("failed to convert the type assertion for Stream")
	}
	return deepcopy, nil
}

func (c *StreamServiceManager) getStreamRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(streaming.GetStreamResponse); ok {
			return resp.LifecycleState == "CREATING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

func (c *StreamServiceManager) deleteStreamRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(streaming.GetStreamResponse); ok {
			return resp.LifecycleState == "DELETING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
