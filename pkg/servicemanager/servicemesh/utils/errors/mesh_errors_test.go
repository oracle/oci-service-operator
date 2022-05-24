/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errors

import (
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	meshErrors "github.com/oracle/oci-service-operator/test/servicemesh/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetOsokResponseByHandlingReconcileError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		shouldRequeue bool
		expectedErr   string
	}{
		{
			name:          "Do not requeue on DoNotRequeueError",
			err:           NewDoNotRequeueError(errors.New("Resource in CP is deleted")),
			shouldRequeue: false,
			expectedErr:   "Resource in CP is deleted",
		},
		{
			name:          "Requeue on RequeueOnError",
			err:           NewRequeueOnError(errors.New("Resource in CP is not active")),
			shouldRequeue: true,
		},
		{
			name:          "Requeue on ServiceError - internal error ",
			err:           meshErrors.NewServiceError(500, "InternalServerError", "An internal server error occurred", "12-35-89"),
			shouldRequeue: true,
			expectedErr:   "Service error:InternalServerError. An internal server error occurred. http status code: 500",
		},
		{
			name:          "Do not requeue on ServiceError - customer mistake",
			err:           meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			shouldRequeue: false,
			expectedErr:   "Service error:MissingParameter. Missing Parameter in the body. http status code: 400",
		},
		{
			name:          "Requeue all other errors",
			err:           errors.New("Other errors"),
			shouldRequeue: true,
			expectedErr:   "Other errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := GetOsokResponseByHandlingReconcileError(tt.err)
			assert.Equal(t, tt.shouldRequeue, response.ShouldRequeue)
			if len(tt.expectedErr) != 0 {
				assert.Equal(t, tt.expectedErr, err.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestShouldRequeueServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Requeue on 500 errors",
			err:  meshErrors.NewServiceError(500, "InternalServerError", "An internal server error occurred", "12-35-89"),
			want: true,
		},
		{
			name: "Do not requeue on MissingParameter errors",
			err:  meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			want: false,
		},
		{
			name: "Do not requeue on MissingParameters errors",
			err:  meshErrors.NewServiceError(400, "MissingParameters", "Missing Parameter in the body", "12-35-89"),
			want: false,
		},
		{
			name: "Do not requeue on InvalidParameter errors",
			err:  meshErrors.NewServiceError(400, "InvalidParameter", "Parameter is invalid or incorrectly formatted", "12-35-89"),
			want: false,
		},
		{
			name: "Do not requeue on InvalidParameters errors",
			err:  meshErrors.NewServiceError(400, "InvalidParameters", "Parameter is invalid or incorrectly formatted", "12-35-89"),
			want: false,
		},
		{
			name: "Do not requeue on CannotParseRequest errors",
			err:  meshErrors.NewServiceError(400, "CannotParseRequest", "The request is incorrectly formatted", "12-35-89"),
			want: false,
		},
		{
			name: "Requeue on other 400 errors",
			err:  meshErrors.NewServiceError(400, "RelatedResourceNotAuthorizedOrNotFound", "A resource specified in the body of the request was not found", "12-35-89"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRequeue := shouldRequeueServiceError(tt.err.(common.ServiceError))
			assert.Equal(t, tt.want, shouldRequeue)
		})
	}
}
