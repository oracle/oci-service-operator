package errortest

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
)

type FakeServiceError struct {
	StatusCode   int
	Code         string
	Message      string
	OpcRequestID string
}

func NewServiceError(statusCode int, code string, message string) FakeServiceError {
	if message == "" {
		message = fmt.Sprintf("%d %s", statusCode, code)
	}
	return FakeServiceError{
		StatusCode:   statusCode,
		Code:         code,
		Message:      message,
		OpcRequestID: "opc-request-id",
	}
}

func NewServiceErrorFromCase(candidate CommonErrorCase) FakeServiceError {
	return NewServiceError(candidate.HTTPStatusCode, candidate.ErrorCode, candidate.Name())
}

func (e FakeServiceError) Error() string {
	return e.Message
}

func (e FakeServiceError) GetHTTPStatusCode() int {
	return e.StatusCode
}

func (e FakeServiceError) GetMessage() string {
	return e.Message
}

func (e FakeServiceError) GetCode() string {
	return e.Code
}

func (e FakeServiceError) GetOpcRequestID() string {
	return e.OpcRequestID
}

func TypeName(err error) string {
	if err == nil {
		return ""
	}
	return reflect.TypeOf(err).String()
}

func AssertErrorType(t *testing.T, err error, want string) {
	t.Helper()
	if got := TypeName(err); got != want {
		t.Fatalf("error type = %q, want %q", got, want)
	}
}

var _ common.ServiceError = FakeServiceError{}
