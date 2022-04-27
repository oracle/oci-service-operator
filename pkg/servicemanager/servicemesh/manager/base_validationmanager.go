/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package manager

import (
	"context"
	"errors"
	"net/http"

	"k8s.io/client-go/tools/record"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	k8Errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type CustomValidationManager interface {
	ValidateCreateRequest(ctx context.Context, object client.Object) admission.Response
	ValidateUpdateRequest(ctx context.Context, object client.Object, oldObject client.Object) admission.Response
	ValidateDeleteRequest(ctx context.Context, object client.Object) admission.Response
	GetObject() client.Object
}

// client - uses cache client, and has read/write both operations
// reader - uses direct client, but only allows read operations
type BaseValidator struct {
	Client            client.Client
	Reader            client.Reader
	Decoder           *admission.Decoder
	ValidationManager CustomValidationManager
	Log               loggerutil.OSOKLogger
	Metrics           *metrics.Metrics
	Recorder          record.EventRecorder
}

func (c *BaseValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	object := c.ValidationManager.GetObject()

	switch req.Operation {
	case v1.Create:
		err := c.Decoder.Decode(req, object)
		if err != nil {
			c.Log.ErrorLog(err, "Failed to decode create request")
			c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
				"Failed to decode create request", req.Name, req.Namespace)
			c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to decode create request")
			return admission.Errored(http.StatusBadRequest, err)
		}
		admissionResponse := c.ValidationManager.ValidateCreateRequest(ctx, object)
		if !admissionResponse.Allowed {
			c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
				"Failed to validate create request", req.Name, req.Namespace)
			c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to validate create request")
		}
		return admissionResponse
	case v1.Update:
		err := c.Decoder.Decode(req, object)
		if err != nil {
			c.Log.ErrorLog(err, "Failed to update existing request")
			c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
				"Failed to update existing request", req.Name, req.Namespace)
			c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to update existing request")
			return admission.Errored(http.StatusBadRequest, err)
		}
		oldObject := c.ValidationManager.GetObject()
		err = c.Decoder.DecodeRaw(req.OldObject, oldObject)
		if err != nil {
			c.Log.ErrorLog(err, "Failed to decode update request")
			c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
				"Failed to decode update request", req.Name, req.Namespace)
			c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to decode update request")
			return admission.Errored(http.StatusBadRequest, err)
		}
		admissionResponse := c.ValidationManager.ValidateUpdateRequest(ctx, object, oldObject)
		if !admissionResponse.Allowed {
			c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
				"Failed to validate update request", req.Name, req.Namespace)
			c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to validate update request")
		}
		return admissionResponse
	case v1.Delete:
		if err := c.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, object); err != nil {
			if k8Errors.IsNotFound(err) {
				c.Log.ErrorLog(err, "Failed to find the required object")
				c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
					"Failed to find the required object", req.Name, req.Namespace)
				c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to find the required object")
				return admission.Errored(http.StatusBadRequest, errors.New("resource not found"))
			}
			c.Log.ErrorLog(err, "Failed to get required object")
			c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
				"Failed to get required object", req.Name, req.Namespace)
			c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to get required object")
			return admission.Errored(http.StatusBadRequest, err)
		}
		admissionResponse := c.ValidationManager.ValidateDeleteRequest(ctx, object)
		if !admissionResponse.Allowed {
			c.Metrics.AddCRDeleteFaultMetrics(object.GetObjectKind().GroupVersionKind().Kind,
				"Failed to validate delete request", req.Name, req.Namespace)
			c.Recorder.Event(object, corev1.EventTypeWarning, "Failed", "Failed to validate delete request")
		}
		return admissionResponse
	default:
		return admission.Errored(http.StatusBadRequest, errors.New("unacceptable request type"))
	}
}

// InjectDecoder injects the Decoder.
// BaseValidator implements admission.DecoderInjector.
// A Decoder will be automatically injected.
func (c *BaseValidator) InjectDecoder(d *admission.Decoder) error {
	c.Decoder = d
	return nil
}
