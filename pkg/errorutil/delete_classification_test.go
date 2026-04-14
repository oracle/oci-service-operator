/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"testing"

	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
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
		wantIsConflict       bool
		wantStatusCodeString string
		wantErrorCodeString  string
		wantTypeString       string
	}{
		{
			name:                 "raw service not found",
			err:                  errortest.NewServiceError(404, NotFound, "resource not found"),
			wantHTTPStatusCode:   404,
			wantErrorCode:        NotFound,
			wantNormalizedType:   "errorutil.NotFoundOciError",
			wantIsConflict:       false,
			wantStatusCodeString: "404",
			wantErrorCodeString:  NotFound,
			wantTypeString:       "errorutil.NotFoundOciError",
		},
		{
			name:                 "raw service auth shaped 404",
			err:                  errortest.NewServiceError(404, NotAuthorizedOrNotFound, "not authorized or not found"),
			wantHTTPStatusCode:   404,
			wantErrorCode:        NotAuthorizedOrNotFound,
			wantNormalizedType:   "errorutil.UnauthorizedAndNotFoundOciError",
			wantIsConflict:       false,
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
			wantIsConflict:       false,
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
			wantIsConflict:       false,
			wantStatusCodeString: "404",
			wantErrorCodeString:  NotAuthorizedOrNotFound,
			wantTypeString:       "errorutil.UnauthorizedAndNotFoundOciError",
		},
		{
			name:                 "raw conflict",
			err:                  errortest.NewServiceError(409, IncorrectState, "conflict"),
			wantHTTPStatusCode:   409,
			wantErrorCode:        IncorrectState,
			wantNormalizedType:   "errorutil.ConflictOciError",
			wantIsConflict:       true,
			wantStatusCodeString: "409",
			wantErrorCodeString:  IncorrectState,
			wantTypeString:       "errorutil.ConflictOciError",
		},
		{
			name:                 "raw external server incorrect state",
			err:                  errortest.NewServiceError(409, "ExternalServerIncorrectState", "external conflict"),
			wantHTTPStatusCode:   409,
			wantErrorCode:        "ExternalServerIncorrectState",
			wantNormalizedType:   "errorutil.ocierrors",
			wantIsConflict:       true,
			wantStatusCodeString: "409",
			wantErrorCodeString:  "ExternalServerIncorrectState",
			wantTypeString:       "errorutil.ocierrors",
		},
		{
			name:                 "raw invalidated retry token",
			err:                  errortest.NewServiceError(409, InvalidatedRetryToken, "retry token invalidated"),
			wantHTTPStatusCode:   409,
			wantErrorCode:        InvalidatedRetryToken,
			wantNormalizedType:   "errorutil.ConflictOciError",
			wantIsConflict:       false,
			wantStatusCodeString: "409",
			wantErrorCodeString:  InvalidatedRetryToken,
			wantTypeString:       "errorutil.ConflictOciError",
		},
		{
			name:                 "raw resource already exists ambiguity",
			err:                  errortest.NewServiceError(409, NotAuthorizedOrResourceAlreadyExists, "already exists"),
			wantHTTPStatusCode:   409,
			wantErrorCode:        NotAuthorizedOrResourceAlreadyExists,
			wantNormalizedType:   "errorutil.ConflictOciError",
			wantIsConflict:       false,
			wantStatusCodeString: "409",
			wantErrorCodeString:  NotAuthorizedOrResourceAlreadyExists,
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
			wantIsConflict:       true,
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
			assert.Equal(t, tt.wantIsConflict, classification.IsConflict())
			assert.Equal(t, tt.wantStatusCodeString, classification.HTTPStatusCodeString())
			assert.Equal(t, tt.wantErrorCodeString, classification.ErrorCodeString())
			assert.Equal(t, tt.wantTypeString, classification.NormalizedTypeString())
		})
	}
}
