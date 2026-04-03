/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/stretchr/testify/assert"
)

func TestClassifyDeleteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		err                  error
		wantHTTPStatusCode   int
		wantErrorCode        string
		wantNormalizedType   string
		wantStatusCodeString string
		wantErrorCodeString  string
		wantTypeString       string
	}{
		{
			name: "raw service not found",
			err: deleteTestServiceError{
				statusCode: 404,
				code:       NotFound,
				message:    "resource not found",
			},
			wantHTTPStatusCode:   404,
			wantErrorCode:        NotFound,
			wantNormalizedType:   "errorutil.NotFoundOciError",
			wantStatusCodeString: "404",
			wantErrorCodeString:  NotFound,
			wantTypeString:       "errorutil.NotFoundOciError",
		},
		{
			name: "raw service auth shaped 404",
			err: deleteTestServiceError{
				statusCode: 404,
				code:       NotAuthorizedOrNotFound,
				message:    "not authorized or not found",
			},
			wantHTTPStatusCode:   404,
			wantErrorCode:        NotAuthorizedOrNotFound,
			wantNormalizedType:   "errorutil.UnauthorizedAndNotFoundOciError",
			wantStatusCodeString: "404",
			wantErrorCodeString:  NotAuthorizedOrNotFound,
			wantTypeString:       "errorutil.UnauthorizedAndNotFoundOciError",
		},
		{
			name: "normalized not found",
			err: NotFoundOciError{
				HTTPStatusCode: 404,
				ErrorCode:      NotFound,
				Description:    "normalized not found",
				OpcRequestID:   "opc-request-id",
			},
			wantHTTPStatusCode:   404,
			wantErrorCode:        NotFound,
			wantNormalizedType:   "errorutil.NotFoundOciError",
			wantStatusCodeString: "404",
			wantErrorCodeString:  NotFound,
			wantTypeString:       "errorutil.NotFoundOciError",
		},
		{
			name: "normalized auth shaped 404",
			err: UnauthorizedAndNotFoundOciError{
				HTTPStatusCode: 404,
				ErrorCode:      NotAuthorizedOrNotFound,
				Description:    "normalized auth shaped 404",
				OpcRequestID:   "opc-request-id",
			},
			wantHTTPStatusCode:   404,
			wantErrorCode:        NotAuthorizedOrNotFound,
			wantNormalizedType:   "errorutil.UnauthorizedAndNotFoundOciError",
			wantStatusCodeString: "404",
			wantErrorCodeString:  NotAuthorizedOrNotFound,
			wantTypeString:       "errorutil.UnauthorizedAndNotFoundOciError",
		},
		{
			name: "raw conflict",
			err: deleteTestServiceError{
				statusCode: 409,
				code:       IncorrectState,
				message:    "conflict",
			},
			wantHTTPStatusCode:   409,
			wantErrorCode:        IncorrectState,
			wantNormalizedType:   "errorutil.ConflictOciError",
			wantStatusCodeString: "409",
			wantErrorCodeString:  IncorrectState,
			wantTypeString:       "errorutil.ConflictOciError",
		},
		{
			name: "normalized conflict",
			err: ConflictOciError{
				HTTPStatusCode: 409,
				ErrorCode:      IncorrectState,
				Description:    "normalized conflict",
				OpcRequestID:   "opc-request-id",
			},
			wantHTTPStatusCode:   409,
			wantErrorCode:        IncorrectState,
			wantNormalizedType:   "errorutil.ConflictOciError",
			wantStatusCodeString: "409",
			wantErrorCodeString:  IncorrectState,
			wantTypeString:       "errorutil.ConflictOciError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			classification := ClassifyDeleteError(tt.err)

			assert.Equal(t, tt.wantHTTPStatusCode, classification.HTTPStatusCode)
			assert.Equal(t, tt.wantErrorCode, classification.ErrorCode)
			assert.Equal(t, tt.wantNormalizedType, classification.NormalizedType)
			assert.Equal(t, tt.wantStatusCodeString, classification.HTTPStatusCodeString())
			assert.Equal(t, tt.wantErrorCodeString, classification.ErrorCodeString())
			assert.Equal(t, tt.wantTypeString, classification.NormalizedTypeString())
		})
	}
}

type deleteTestServiceError struct {
	statusCode int
	code       string
	message    string
}

func (e deleteTestServiceError) Error() string {
	return e.message
}

func (e deleteTestServiceError) GetHTTPStatusCode() int {
	return e.statusCode
}

func (e deleteTestServiceError) GetMessage() string {
	return e.message
}

func (e deleteTestServiceError) GetCode() string {
	return e.code
}

func (e deleteTestServiceError) GetOpcRequestID() string {
	return "opc-request-id"
}

var _ common.ServiceError = deleteTestServiceError{}
