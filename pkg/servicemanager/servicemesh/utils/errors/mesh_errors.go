/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RequeueOnError is to used when a request needs to be requeued immediately due to an error.
type RequeueOnError struct {
	err error
}

func NewRequeueOnError(err error) *RequeueOnError {
	return &RequeueOnError{
		err: err,
	}
}

func NewRequeueAfter(duration time.Duration) *RequeueAfterError {
	return &RequeueAfterError{
		err:      nil,
		duration: duration,
	}
}

// RequeueAfterError is used when a request needs to be requeued after some time due to an error.
type RequeueAfterError struct {
	err      error
	duration time.Duration
}

func NewRequeueAfterError(err error, duration time.Duration) *RequeueAfterError {
	return &RequeueAfterError{
		err:      err,
		duration: duration,
	}
}

func (e *RequeueAfterError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

var _ error = &RequeueAfterError{}

func (e *RequeueOnError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

var _ error = &RequeueOnError{}

func GetOsokResponseByHandlingReconcileError(err error) (servicemanager.OSOKResponse, error) {
	var doNotRequeue *DoNotRequeueError
	if errors.As(err, &doNotRequeue) {
		return servicemanager.OSOKResponse{ShouldRequeue: false}, err
	}

	// Requeue the request right away with rate limiting if there are errors. A nil error will reset the backoff.
	var requeueOnError *RequeueOnError
	if errors.As(err, &requeueOnError) {
		return servicemanager.OSOKResponse{ShouldRequeue: true}, nil
	}

	// Requeue a Request if there is an error and continue processing items with exponential backoff
	return servicemanager.OSOKResponse{ShouldRequeue: true}, err
}

func GetValidationErrorMessage(object client.Object, reason string) string {
	return fmt.Sprintf("Failed to create Resource for Kind: %s, Name: %s, Namespace: %s, Error: %s", object.GetObjectKind().GroupVersionKind().Kind, object.GetName(), object.GetNamespace(), reason)
}

func ResponseStatusText(err common.ServiceError) string {
	// For connection errors, httpResponse is nil
	if err.GetHTTPStatusCode() == 0 {
		return string(commons.ConnectionError)
	}
	return strcase.ToCamel(http.StatusText(err.GetHTTPStatusCode()))
}

// GetConditionStatus returns the state of the condition based on the error returned from Control plane
func GetConditionStatus(err common.ServiceError) metav1.ConditionStatus {
	if err.GetHTTPStatusCode() != 404 {
		return metav1.ConditionUnknown
	}
	return metav1.ConditionFalse
}

// GetMeshConfiguredConditionStatus returns the state of the MeshConfigured condition based on the error returned from Control plane
func GetMeshConfiguredConditionStatus(err common.ServiceError) metav1.ConditionStatus {
	if err.GetHTTPStatusCode() >= 500 && err.GetHTTPStatusCode() <= 599 {
		return metav1.ConditionUnknown
	}
	return metav1.ConditionFalse
}

// GetErrorMessage returns the error message along with the opcRequestId in it
func GetErrorMessage(err common.ServiceError) string {
	return fmt.Sprintf("%s (opc-request-id: %s )", err.GetMessage(), err.GetOpcRequestID())
}

func IsDeleted(ctx context.Context, err error, log loggerutil.OSOKLogger) error {
	if err == nil {
		return nil
	}

	if serviceError, ok := err.(common.ServiceError); ok {
		if serviceError.GetHTTPStatusCode() == http.StatusNotFound {
			log.ErrorLogWithFixedMessage(ctx, err, "Entity not found. Maybe it was already deleted.")
			return nil
		}
	}
	log.ErrorLogWithFixedMessage(ctx, err, "Failed to delete entity from control plane")
	return err
}

type DoNotRequeueError struct {
	err error
}

func NewDoNotRequeueError(err error) *DoNotRequeueError {
	return &DoNotRequeueError{
		err: err,
	}
}

func (e *DoNotRequeueError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

var _ error = &DoNotRequeueError{}

func HandleErrorAndRequeue(ctx context.Context, err error, log loggerutil.OSOKLogger) (ctrl.Result, error) {
	// Do not requeue the request if there is no error or error is of type DoNotRequeue
	var doNotRequeue *DoNotRequeueError
	if err == nil || errors.As(err, &doNotRequeue) {
		return ctrl.Result{}, err
	}

	// Requeue the request right away with rate limiting if there are errors. A nil error will reset the backoff.
	var requeueOnError *RequeueOnError
	if errors.As(err, &requeueOnError) {
		log.InfoLogWithFixedMessage(ctx, "requeue due to error", "error", requeueOnError.err.Error())
		return ctrl.Result{Requeue: true}, nil
	}

	// Requeue (Pending) the request after a specific duration. A single item will not be processed multiple times
	// concurrently, and if an item is added multiple times before it can be processed, it will only be processed once.
	var requeueAfterErr *RequeueAfterError
	if errors.As(err, &requeueAfterErr) {
		if requeueAfterErr.err == nil {
			log.InfoLogWithFixedMessage(ctx, "requeue after for periodical syncing", "duration", requeueAfterErr.duration.String())
			return ctrl.Result{RequeueAfter: requeueAfterErr.duration}, nil
		}
		log.InfoLogWithFixedMessage(ctx, "requeue after due to error", "duration", requeueAfterErr.duration.String(), "error", requeueAfterErr.err.Error())
		return ctrl.Result{RequeueAfter: requeueAfterErr.duration}, nil
	}

	// Requeue a Request if there is an error and continue processing items with exponential backoff
	log.InfoLogWithFixedMessage(ctx, "request failed due to error: ", "error", err.Error())
	return ctrl.Result{}, err
}

func IsNetworkErrorOrInternalError(err error) bool {
	if common.IsNetworkError(err) {
		return true
	}
	if serviceError, ok := err.(common.ServiceError); ok {
		if serviceError.GetHTTPStatusCode() >= 500 && serviceError.GetHTTPStatusCode() <= 599 {
			return true
		}
	}
	return false
}
