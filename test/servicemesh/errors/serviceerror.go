/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errors

import (
	"fmt"
)

type serviceError struct {
	StatusCode   int
	Code         string
	Message      string
	OpcRequestID string
}

func NewServiceError(statusCode int, code string, message string, opcRequestId string) error {
	return serviceError{
		StatusCode:   statusCode,
		Code:         code,
		Message:      message,
		OpcRequestID: opcRequestId,
	}
}

func (se serviceError) Error() string {
	return fmt.Sprintf("Service error:%s. %s. http status code: %d",
		se.Code, se.Message, se.StatusCode)
}

func (se serviceError) GetHTTPStatusCode() int {
	return se.StatusCode
}

func (se serviceError) GetMessage() string {
	return se.Message
}

func (se serviceError) GetCode() string {
	return se.Code
}

func (se serviceError) GetOpcRequestID() string {
	return se.OpcRequestID
}

type serviceTimeoutError struct {
	error
}

func NewServiceTimeoutError() error {
	return serviceTimeoutError{}
}

func (e serviceTimeoutError) Timeout() bool {
	return true
}

func (e serviceTimeoutError) Temporary() bool {
	return true
}

func (e serviceTimeoutError) Error() string {
	return "Request to service timeout"
}
