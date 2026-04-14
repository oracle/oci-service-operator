package errortest

import "fmt"

type Operation string

const (
	OperationRead   Operation = "read"
	OperationCreate Operation = "create"
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

type Expectation string

const (
	ExpectationFatal     Expectation = "fatal"
	ExpectationRetryable Expectation = "retryable"
	ExpectationAbsent    Expectation = "absent"
	ExpectationDeleted   Expectation = "deleted"
)

type OperationExpectations struct {
	Read   Expectation
	Create Expectation
	Update Expectation
	Delete Expectation
}

type CommonErrorCase struct {
	HTTPStatusCode int
	ErrorCode      string
	NormalizedType string
	Supported      bool
	Rationale      string
	Expectations   OperationExpectations
}

func (c CommonErrorCase) Name() string {
	return fmt.Sprintf("%d/%s", c.HTTPStatusCode, c.ErrorCode)
}

const CommonErrorMatrixVersion = "oracle-api-errors-2026-04-14"

var CommonErrorMatrix = []CommonErrorCase{
	{HTTPStatusCode: 400, ErrorCode: "CannotParseRequest", NormalizedType: "errorutil.BadRequestOciError", Supported: true, Rationale: "Explicit 400 request-format classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 400, ErrorCode: "InvalidParameter", NormalizedType: "errorutil.BadRequestOciError", Supported: true, Rationale: "Explicit 400 parameter validation classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 400, ErrorCode: "LimitExceeded", NormalizedType: "errorutil.BadRequestOciError", Supported: true, Rationale: "Explicit 400 tenancy-limit classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 400, ErrorCode: "MissingParameter", NormalizedType: "errorutil.BadRequestOciError", Supported: true, Rationale: "Explicit 400 missing-parameter classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 400, ErrorCode: "QuotaExceeded", NormalizedType: "errorutil.BadRequestOciError", Supported: true, Rationale: "Explicit 400 compartment-quota classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 400, ErrorCode: "RelatedResourceNotAuthorizedOrNotFound", NormalizedType: "errorutil.BadRequestOciError", Supported: true, Rationale: "Explicit 400 related-resource classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 401, ErrorCode: "NotAuthenticated", NormalizedType: "errorutil.NotAuthenticatedOciError", Supported: true, Rationale: "Explicit 401 auth classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 402, ErrorCode: "SignUpRequired", NormalizedType: "errorutil.SignUpRequiredOciError", Supported: true, Rationale: "Explicit 402 opt-in classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 403, ErrorCode: "NotAuthorized", NormalizedType: "errorutil.UnauthorizedAndNotFoundOciError", Supported: true, Rationale: "Explicit 403 auth classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 403, ErrorCode: "NotAllowed", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 403 NotAllowed, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 404, ErrorCode: "InvalidParameter", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 404 InvalidParameter, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 404, ErrorCode: "NamespaceNotFound", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 404 NamespaceNotFound, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: fatalEverywhere()},
	{
		HTTPStatusCode: 404,
		ErrorCode:      "NotAuthorizedOrNotFound",
		NormalizedType: "errorutil.UnauthorizedAndNotFoundOciError",
		Supported:      true,
		Rationale:      "Explicit 404 auth-shaped classification already exists in pkg/errorutil and feeds delete/finalizer semantics.",
		Expectations: OperationExpectations{
			Read:   ExpectationFatal,
			Create: ExpectationFatal,
			Update: ExpectationFatal,
			Delete: ExpectationFatal,
		},
	},
	{
		HTTPStatusCode: 404,
		ErrorCode:      "NotFound",
		NormalizedType: "errorutil.NotFoundOciError",
		Supported:      true,
		Rationale:      "Explicit 404 unambiguous not-found classification already exists in pkg/errorutil and feeds delete/finalizer semantics.",
		Expectations: OperationExpectations{
			Read:   ExpectationAbsent,
			Create: ExpectationFatal,
			Update: ExpectationFatal,
			Delete: ExpectationDeleted,
		},
	},
	{HTTPStatusCode: 405, ErrorCode: "MethodNotAllowed", NormalizedType: "errorutil.MethodNotAllowedOciError", Supported: true, Rationale: "Explicit 405 classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{
		HTTPStatusCode: 409,
		ErrorCode:      "ExternalServerIncorrectState",
		NormalizedType: "errorutil.ocierrors",
		Supported:      false,
		Rationale:      "Oracle documents 409 ExternalServerIncorrectState, but pkg/errorutil still falls back to the raw ocierrors type for this row.",
		Expectations: OperationExpectations{
			Read:   ExpectationFatal,
			Create: ExpectationRetryable,
			Update: ExpectationRetryable,
			Delete: ExpectationRetryable,
		},
	},
	{
		HTTPStatusCode: 409,
		ErrorCode:      "IncorrectState",
		NormalizedType: "errorutil.ConflictOciError",
		Supported:      true,
		Rationale:      "Explicit 409 lifecycle-conflict classification already exists in pkg/errorutil.",
		Expectations: OperationExpectations{
			Read:   ExpectationFatal,
			Create: ExpectationRetryable,
			Update: ExpectationRetryable,
			Delete: ExpectationRetryable,
		},
	},
	{HTTPStatusCode: 409, ErrorCode: "InvalidatedRetryToken", NormalizedType: "errorutil.ConflictOciError", Supported: true, Rationale: "Explicit 409 retry-token classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 409, ErrorCode: "NotAuthorizedOrResourceAlreadyExists", NormalizedType: "errorutil.ConflictOciError", Supported: true, Rationale: "Explicit 409 duplicate-or-auth classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 412, ErrorCode: "NoEtagMatch", NormalizedType: "errorutil.NoEtagMatchOciError", Supported: true, Rationale: "Explicit 412 etag classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 413, ErrorCode: "PayloadTooLarge", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 413 PayloadTooLarge, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 422, ErrorCode: "UnprocessableEntity", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 422 UnprocessableEntity, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 429, ErrorCode: "TooManyRequests", NormalizedType: "errorutil.TooManyRequestsOciError", Supported: true, Rationale: "Explicit 429 throttling classification already exists in pkg/errorutil.", Expectations: retryableEverywhere()},
	{HTTPStatusCode: 431, ErrorCode: "RequestHeaderFieldsTooLarge", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 431 RequestHeaderFieldsTooLarge, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 500, ErrorCode: "InternalServerError", NormalizedType: "errorutil.InternalServerErrorOciError", Supported: true, Rationale: "Explicit 500 server-error classification already exists in pkg/errorutil.", Expectations: retryableEverywhere()},
	{HTTPStatusCode: 501, ErrorCode: "MethodNotImplemented", NormalizedType: "errorutil.MethodNotImplementedOciError", Supported: true, Rationale: "Explicit 501 classification already exists in pkg/errorutil.", Expectations: fatalEverywhere()},
	{HTTPStatusCode: 503, ErrorCode: "ExternalServerInvalidResponse", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 503 ExternalServerInvalidResponse, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: retryableEverywhere()},
	{HTTPStatusCode: 503, ErrorCode: "ExternalServerTimeout", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 503 ExternalServerTimeout, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: retryableEverywhere()},
	{HTTPStatusCode: 503, ErrorCode: "ExternalServerUnreachable", NormalizedType: "errorutil.ocierrors", Supported: false, Rationale: "Oracle documents 503 ExternalServerUnreachable, but pkg/errorutil still falls back to the raw ocierrors type for this row.", Expectations: retryableEverywhere()},
	{HTTPStatusCode: 503, ErrorCode: "ServiceUnavailable", NormalizedType: "errorutil.ServiceUnavailableOciError", Supported: true, Rationale: "Explicit 503 service-unavailable classification already exists in pkg/errorutil.", Expectations: retryableEverywhere()},
}

func LookupCommonErrorCase(statusCode int, errorCode string) (CommonErrorCase, bool) {
	for _, candidate := range CommonErrorMatrix {
		if candidate.HTTPStatusCode == statusCode && candidate.ErrorCode == errorCode {
			return candidate, true
		}
	}
	return CommonErrorCase{}, false
}

func fatalEverywhere() OperationExpectations {
	return OperationExpectations{
		Read:   ExpectationFatal,
		Create: ExpectationFatal,
		Update: ExpectationFatal,
		Delete: ExpectationFatal,
	}
}

func retryableEverywhere() OperationExpectations {
	return OperationExpectations{
		Read:   ExpectationRetryable,
		Create: ExpectationRetryable,
		Update: ExpectationRetryable,
		Delete: ExpectationRetryable,
	}
}
