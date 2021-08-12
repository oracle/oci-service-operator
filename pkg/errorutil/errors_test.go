/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadRequestResponseWithInvalidParameter(t *testing.T) {

	var code = "InvalidParameter"
	var statusCode = 400
	var opcRequestId = "12-35-67"
	var message = "Invalid Parameter"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(BadRequestOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "Parameter is invalid or incorrectly formatted", err.(BadRequestOciError).Description)
	assert.Equal(t, 400, err.(BadRequestOciError).HTTPStatusCode)
}

func TestBadRequestResponseWithMissingParameter(t *testing.T) {

	var code = "MissingParameter"
	var statusCode = 400
	var opcRequestId = "12-35-89"
	var message = "Missing Parameter in the body"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(BadRequestOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "Required parameter is missing", err.(BadRequestOciError).Description)
	assert.Equal(t, 400, err.(BadRequestOciError).HTTPStatusCode)
}

func TestNotAuthenticatedResponse(t *testing.T) {

	var code = "NotAuthenticated"
	var statusCode = 401
	var opcRequestId = "12-35-89"
	var message = "Not Authenticated to perform operation"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(NotAuthenticatedOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "The required authentication information was not provided or was incorrect",
		err.(NotAuthenticatedOciError).Description)
	assert.Equal(t, 401, err.(NotAuthenticatedOciError).HTTPStatusCode)
}

func TestSignUpRequiredResponse(t *testing.T) {

	var code = "SignUpRequired"
	var statusCode = 402
	var opcRequestId = "12-35-89-09"
	var message = "Sign up required"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(SignUpRequiredOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "This operation requires opt-in before it may be called",
		err.(SignUpRequiredOciError).Description)
	assert.Equal(t, 402, err.(SignUpRequiredOciError).HTTPStatusCode)
}

func TestUnauthorizedResponse(t *testing.T) {

	var code = "NotAuthorized"
	var statusCode = 403
	var opcRequestId = "12-89-87-98"
	var message = "Not Authorized to perform operation"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(UnauthorizedAndNotFoundOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "You do not have authorization to update one or more of the fields included in this"+
		" request", err.(UnauthorizedAndNotFoundOciError).Description)
	assert.Equal(t, 403, err.(UnauthorizedAndNotFoundOciError).HTTPStatusCode)
}

func TestNotFoundResponse(t *testing.T) {

	var code = "NotFound"
	var statusCode = 404
	var opcRequestId = "76-98-57-03"
	var message = "Resource not Found"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(UnauthorizedAndNotFoundOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "There is no operation supported at the URI path and HTTP method you specified in the request",
		err.(UnauthorizedAndNotFoundOciError).Description)
	assert.Equal(t, 404, err.(UnauthorizedAndNotFoundOciError).HTTPStatusCode)
}

func TestNotAuthorizedOrNotFoundResponse(t *testing.T) {

	var code = "NotAuthorizedOrNotFound"
	var statusCode = 404
	var opcRequestId = "12-35-89-343"
	var message = "Not Authorized to perform action Or Resource Not Found"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(UnauthorizedAndNotFoundOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "A resource specified via the URI (path or query parameters) of the request was not found, "+
		"or you do not have authorization to access that resource", err.(UnauthorizedAndNotFoundOciError).Description)
	assert.Equal(t, 404, err.(UnauthorizedAndNotFoundOciError).HTTPStatusCode)
}

func TestMethodNotAllowedResponse(t *testing.T) {

	var code = "MethodNotAllowed"
	var statusCode = 405
	var opcRequestId = "12-35-8324sd9"
	var message = "Method don't Allow http"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(MethodNotAllowedOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "The target resource does not support the HTTP method",
		err.(MethodNotAllowedOciError).Description)
	assert.Equal(t, 405, err.(MethodNotAllowedOciError).HTTPStatusCode)
}

func TestConflictResponseWhileIncorrectState(t *testing.T) {

	var code = "IncorrectState"
	var statusCode = 409
	var opcRequestId = "12-89-23-234"
	var message = "Requested state conflict with present state"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(ConflictOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "The requested state for the resource conflicts with its current state",
		err.(ConflictOciError).Description)
	assert.Equal(t, 409, err.(ConflictOciError).HTTPStatusCode)
}

func TestConflictResponseWhileNotAuthorizedOrResourceAlreadyExists(t *testing.T) {

	var code = "NotAuthorizedOrResourceAlreadyExists"
	var statusCode = 409
	var opcRequestId = "12-35-89"
	var message = "Not Authorized Or Resource Already Exists"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(ConflictOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "You do not have authorization to perform this request, or the resource you are attempting "+
		"to create already exists", err.(ConflictOciError).Description)
	assert.Equal(t, 409, err.(ConflictOciError).HTTPStatusCode)
}

func TestNoEtagMatchResponse(t *testing.T) {

	var code = "NoEtagMatch"
	var statusCode = 412
	var opcRequestId = "12-35-89"
	var message = "No Etag Match"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(NoEtagMatchOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "The ETag specified in the request does not match the ETag for the resource",
		err.(NoEtagMatchOciError).Description)
	assert.Equal(t, 412, err.(NoEtagMatchOciError).HTTPStatusCode)
}

func TestTooManyRequestsResponse(t *testing.T) {

	var code = "TooManyRequests"
	var statusCode = 429
	var opcRequestId = "12-35-89"
	var message = "Too Many Requests"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(TooManyRequestsOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "You have issued too many requests to the Oracle Cloud Infrastructure APIs in too short "+
		"of an amount of time", err.(TooManyRequestsOciError).Description)
	assert.Equal(t, 429, err.(TooManyRequestsOciError).HTTPStatusCode)
}

func TestInternalServerErrorResponse(t *testing.T) {

	var code = "InternalServerError"
	var statusCode = 500
	var opcRequestId = "12-35-89-93-213"
	var message = "InternalServerError"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(InternalServerErrorOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "An internal server error occurred", err.(InternalServerErrorOciError).Description)
	assert.Equal(t, 500, err.(InternalServerErrorOciError).HTTPStatusCode)
}

func TestMethodNotImplementedResponse(t *testing.T) {

	var code = "MethodNotImplemented"
	var statusCode = 501
	var opcRequestId = "12-35-89"
	var message = "Not Authenticated to perform operation"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(MethodNotImplementedOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "The HTTP request target does not recognize the HTTP method",
		err.(MethodNotImplementedOciError).Description)
	assert.Equal(t, 501, err.(MethodNotImplementedOciError).HTTPStatusCode)
}

func TestServiceUnavailableResponse(t *testing.T) {

	var code = "ServiceUnavailable"
	var statusCode = 503
	var opcRequestId = "12-35-89"
	var message = "Service is Unavailable for now"

	resp, err := NewServiceFailureFromResponse(code, statusCode, opcRequestId, message)
	assert.False(t, resp)
	_, ok := err.(ServiceUnavailableOciError)
	assert.Equal(t, true, ok)
	assert.Equal(t, "The service is currently unavailable", err.(ServiceUnavailableOciError).Description)
	assert.Equal(t, 503, err.(ServiceUnavailableOciError).HTTPStatusCode)
}
