/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"fmt"
)

type OciErrors struct {
	HTTPStatusCode int    `json:"http-status-code,omitempty"`
	ErrorCode      string `json:"error-code,omitempty"`
	OpcRequestID   string `json:"opc-request-id,omitempty"`
	Description    string `json:"description,omitempty"`
}

func (bd OciErrors) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		bd.Description, bd.HTTPStatusCode, bd.OpcRequestID)
}

// For 400 errors
type BadRequestOciError OciErrors

func (bd BadRequestOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		bd.Description, bd.HTTPStatusCode, bd.OpcRequestID)
}

// For 401 errors
type NotAuthenticatedOciError OciErrors

func (na NotAuthenticatedOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		na.Description, na.HTTPStatusCode, na.OpcRequestID)
}

// For 402 errors
type SignUpRequiredOciError OciErrors

func (sr SignUpRequiredOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		sr.Description, sr.HTTPStatusCode, sr.OpcRequestID)
}

// For 403 and 404 errors
type UnauthorizedAndNotFoundOciError OciErrors

func (ua UnauthorizedAndNotFoundOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		ua.Description, ua.HTTPStatusCode, ua.OpcRequestID)
}

//For 405 errors
type MethodNotAllowedOciError OciErrors

func (mna MethodNotAllowedOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		mna.Description, mna.HTTPStatusCode, mna.OpcRequestID)
}

//For 409 errors
type ConflictOciError OciErrors

func (c ConflictOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		c.Description, c.HTTPStatusCode, c.OpcRequestID)
}

//For 412 errors
type NoEtagMatchOciError OciErrors

func (nem NoEtagMatchOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		nem.Description, nem.HTTPStatusCode, nem.OpcRequestID)
}

// For 429 Error
type TooManyRequestsOciError OciErrors

func (tmr TooManyRequestsOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		tmr.Description, tmr.HTTPStatusCode, tmr.OpcRequestID)
}

//For 500 Error
type InternalServerErrorOciError OciErrors

func (nf InternalServerErrorOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		nf.Description, nf.HTTPStatusCode, nf.OpcRequestID)
}

//For 501 Error
type MethodNotImplementedOciError OciErrors

func (mni MethodNotImplementedOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		mni.Description, mni.HTTPStatusCode, mni.OpcRequestID)
}

//For 503 Error
type ServiceUnavailableOciError OciErrors

func (su ServiceUnavailableOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		su.Description, su.HTTPStatusCode, su.OpcRequestID)
}

const (
	CannotParseRequest                     string = "CannotParseRequest"
	InvalidParameters                      string = "InvalidParameters"
	InvalidParameter                       string = "InvalidParameter"
	LimitExceeded                          string = "LimitExceeded"
	MissingParameters                      string = "MissingParameters"
	MissingParameter                       string = "MissingParameter"
	QuotaExceeded                          string = "QuotaExceeded"
	RelatedResourceNotAuthorizedOrNotFound string = "RelatedResourceNotAuthorizedOrNotFound"
	NotAuthenticated                       string = "NotAuthenticated"
	SignUpRequired                         string = "SignUpRequired"
	NotAuthorizedOrNotFound                string = "NotAuthorizedOrNotFound"
	NotFound                               string = "NotFound"
	MethodNotAllowed                       string = "MethodNotAllowed"
	IncorrectState                         string = "IncorrectState"
	InvalidatedRetryToken                  string = "InvalidatedRetryToken"
	NotAuthorizedOrResourceAlreadyExists   string = "NotAuthorizedOrResourceAlreadyExists"
	NotAuthorized                          string = "NotAuthorized"
	NoEtagMatch                            string = "NoEtagMatch"
	TooManyRequests                        string = "TooManyRequests"
	InternalServerError                    string = "InternalServerError"
	MethodNotImplemented                   string = "MethodNotImplemented"
	ServiceUnavailable                     string = "ServiceUnavailable"
)

// For 400 errors
func BadRequestResponse(err error, description string) error {

	br := BadRequestOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}
	return br
}

//For 401 errros
func NotAuthenticatedResponse(err error, description string) error {

	na := NotAuthenticatedOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}
	return na
}

// For 402 errors
func SignUpRequiredResponse(err error, description string) error {

	sur := SignUpRequiredOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}
	return sur
}

// For 403 and 404 errors
func UnauthorizedAndNotFoundResponse(err error, description string) error {

	uanf := UnauthorizedAndNotFoundOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}

	return uanf
}

//For 405 errors
func MethodNotAllowedResponse(err error, description string) error {

	mna := MethodNotAllowedOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}

	return mna
}

//For 409 errors
func ConflictResponse(err error, description string) error {

	c := ConflictOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}

	return c
}

//For 412 errors
func NoEtagMatchResponse(err error, description string) error {

	nem := NoEtagMatchOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}

	return nem
}

// For 429 Error
func TooManyRequestsResponse(err error, description string) error {

	tmr := TooManyRequestsOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}
	return tmr
}

//For 500 error
func InternalServerErrorResponse(err error, description string) error {

	ise := InternalServerErrorOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}
	return ise
}

//For 501 error
func MethodNotImplementedResponse(err error, description string) error {

	mni := MethodNotImplementedOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}
	return mni
}

//For 503 error
func ServiceUnavailableResponse(err error, description string) error {

	su := ServiceUnavailableOciError{
		ErrorCode:      err.(ocierrors).ErrorCode,
		HTTPStatusCode: err.(ocierrors).HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.(ocierrors).OpcRequestID,
	}
	return su
}
