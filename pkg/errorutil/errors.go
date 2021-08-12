/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"fmt"
	"github.com/oracle/oci-go-sdk/v41/common"
)

type ocierrors struct {
	HTTPStatusCode int
	ErrorCode      string `json:"code,omitempty"`
	Description    string `json:"message,omitempty"`
	OpcRequestID   string `json:"opc-request-id"`
}

func (o ocierrors) Error() string {
	return fmt.Sprintf("Service error:%s. %s. http status code: %d. Opc request id: %s",
		o.ErrorCode, o.Description, o.HTTPStatusCode, o.OpcRequestID)
}

func OciErrorTypeResponse(err error) (bool, error) {
	return NewServiceFailureFromResponse(err.(common.ServiceError).GetCode(),
		err.(common.ServiceError).GetHTTPStatusCode(), err.(common.ServiceError).GetOpcRequestID(),
		err.(common.ServiceError).GetMessage())
}

func NewServiceFailureFromResponse(code string, statusCode int, opcRequestId string, message string) (bool, error) {

	se := ocierrors{
		ErrorCode:      code,
		HTTPStatusCode: statusCode,
		OpcRequestID:   opcRequestId,
		Description:    message,
	}

	switch status := se.HTTPStatusCode; {
	case status >= 400 && status <= 499:
		return check4xxFailures(se)
	case status >= 500 && status <= 599:
		return check5xxFailures(se)
	default:
		return true, nil
	}

}

// Return the specific 4xx error Object
func check4xxFailures(se ocierrors) (bool, error) {

	switch statusCode := se.HTTPStatusCode; {

	case statusCode == 400:
		return specific400Error(se)
	case statusCode == 401:
		return specific401Error(se)
	case statusCode == 402:
		return specific402Error(se)
	case statusCode == 403:
		return specific403Error(se)
	case statusCode == 404:
		return specific404Error(se)
	case statusCode == 405:
		return specific405Error(se)
	case statusCode == 409:
		return specific409Error(se)
	case statusCode == 412:
		return specific412Error(se)
	case statusCode == 429:
		return specific429Error(se)
	default:
		return false, se
	}
}

// Return the specific 5xx error Object
func check5xxFailures(se ocierrors) (bool, error) {
	switch statusCode := se.HTTPStatusCode; {
	case statusCode == 500:
		return specific500Error(se)
	case statusCode == 501:
		return specific501Error(se)
	case statusCode == 503:
		return specific503Error(se)
	default:
		return false, se
	}
}

//Return specific 400 error. Their are different types of 400 error ex. Invalid-Parameter or Missing-Parameter etc.
func specific400Error(se ocierrors) (bool, error) {

	switch errorCode := se.ErrorCode; {

	case errorCode == CannotParseRequest:
		return false, BadRequestResponse(se, "The request is incorrectly formatted")
	case errorCode == InvalidParameter || errorCode == InvalidParameters:
		return false, BadRequestResponse(se, "Parameter is invalid or incorrectly formatted")
	case errorCode == MissingParameter || errorCode == MissingParameters:
		return false, BadRequestResponse(se, "Required parameter is missing")
	case errorCode == LimitExceeded:
		return false, BadRequestResponse(se, "Fulfilling this request exceeds the Oracle-defined "+
			"limit for this tenancy for this resource type")
	case errorCode == QuotaExceeded:
		return false, BadRequestResponse(se, "Fulfilling this request exceeds the "+
			"administrator-defined quota for this compartment for this resource")
	case errorCode == RelatedResourceNotAuthorizedOrNotFound:
		return false, BadRequestResponse(se, "A resource specified in the body of the request was "+
			"not found, or you do not have authorization to access that resource")
	default:
		return false, BadRequestResponse(se, "Bad Request")
	}
}

func specific401Error(se ocierrors) (bool, error) {

	if se.ErrorCode == NotAuthenticated {
		return false, NotAuthenticatedResponse(se, "The required authentication information was not "+
			"provided or was incorrect")
	}

	return false, se
}

func specific402Error(se ocierrors) (bool, error) {
	if se.ErrorCode == SignUpRequired {
		return false, SignUpRequiredResponse(se, "This operation requires opt-in before it may be called")
	}
	return false, se
}

func specific403Error(se ocierrors) (bool, error) {

	if se.ErrorCode == NotAuthorized {
		return false, UnauthorizedAndNotFoundResponse(se, "You do not have authorization to update one "+
			"or more of the fields included in this request")
	}
	return false, se
}

func specific404Error(se ocierrors) (bool, error) {

	if se.ErrorCode == NotFound {
		return false, UnauthorizedAndNotFoundResponse(se, "There is no operation supported at the "+
			"URI path and HTTP method you specified in the request")
	} else if se.ErrorCode == NotAuthorizedOrNotFound {
		return false, UnauthorizedAndNotFoundResponse(se, "A resource specified via the URI (path or "+
			"query parameters) of the request was not found, or you do not have authorization to access that resource")
	}

	return false, se
}

func specific405Error(se ocierrors) (bool, error) {
	if se.ErrorCode == MethodNotAllowed {
		return false, MethodNotAllowedResponse(se, "The target resource does not support the HTTP method")
	}
	return false, se
}

func specific409Error(se ocierrors) (bool, error) {

	switch errorCode := se.ErrorCode; {
	case errorCode == IncorrectState:
		return false, ConflictResponse(se, "The requested state for the resource conflicts with "+
			"its current state")
	case errorCode == InvalidatedRetryToken:
		return false, ConflictResponse(se, "The provided retry token was used in an earlier request "+
			"that resulted in a system update, but a subsequent operation invalidated "+
			"the token")
	case errorCode == NotAuthorizedOrResourceAlreadyExists:

		return false, ConflictResponse(se, "You do not have authorization to perform this request, or "+
			"the resource you are attempting to create already exists")
	default:
		return false, se
	}
}

func specific412Error(se ocierrors) (bool, error) {
	if se.ErrorCode == NoEtagMatch {
		return false, NoEtagMatchResponse(se, "The ETag specified in the request does not match the ETag for "+
			"the resource")
	}
	return false, se
}

func specific429Error(se ocierrors) (bool, error) {
	if se.ErrorCode == TooManyRequests {
		return false, TooManyRequestsResponse(se, "You have issued too many requests to the Oracle Cloud "+
			"Infrastructure APIs in too short of an amount of time")
	}
	return false, se
}

func specific500Error(se ocierrors) (bool, error) {

	if se.ErrorCode == InternalServerError {
		return false, InternalServerErrorResponse(se, "An internal server error occurred")
	}
	return false, se
}

func specific501Error(se ocierrors) (bool, error) {

	if se.ErrorCode == MethodNotImplemented {
		return false, MethodNotImplementedResponse(se, "The HTTP request target does not recognize the HTTP method")
	}
	return false, se
}

func specific503Error(se ocierrors) (bool, error) {

	if se.ErrorCode == ServiceUnavailable {
		return false, ServiceUnavailableResponse(se, "The service is currently unavailable")
	}

	return false, se
}
