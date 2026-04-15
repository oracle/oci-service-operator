package errortest

import (
	"fmt"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type GeneratedRuntimePlainMutationResult struct {
	Response     servicemanager.OSOKResponse
	Err          error
	StatusReason string
}

type GeneratedRuntimePlainReadResult struct {
	Err     error
	Missing bool
}

type GeneratedRuntimePlainDeleteResult struct {
	Deleted      bool
	Err          error
	StatusReason string
}

func RunGeneratedRuntimePlainCreateMatrix(
	t *testing.T,
	run func(*testing.T, CommonErrorCase) GeneratedRuntimePlainMutationResult,
) {
	t.Helper()
	runGeneratedRuntimePlainMutationMatrix(t, OperationCreate, run)
}

func RunGeneratedRuntimePlainUpdateMatrix(
	t *testing.T,
	run func(*testing.T, CommonErrorCase) GeneratedRuntimePlainMutationResult,
) {
	t.Helper()
	runGeneratedRuntimePlainMutationMatrix(t, OperationUpdate, run)
}

func RunGeneratedRuntimePlainReadMatrix(
	t *testing.T,
	run func(*testing.T, CommonErrorCase) GeneratedRuntimePlainReadResult,
) {
	t.Helper()

	for _, candidate := range CommonErrorMatrix {
		candidate := candidate
		t.Run(candidate.Name(), func(t *testing.T) {
			t.Parallel()

			result := run(t, candidate)
			assertGeneratedRuntimePlainReadResult(t, candidate, result)
		})
	}
}

func RunGeneratedRuntimePlainDeleteMatrix(
	t *testing.T,
	run func(*testing.T, CommonErrorCase) GeneratedRuntimePlainDeleteResult,
) {
	t.Helper()

	for _, candidate := range CommonErrorMatrix {
		candidate := candidate
		t.Run(candidate.Name(), func(t *testing.T) {
			t.Parallel()

			result := run(t, candidate)
			assertGeneratedRuntimePlainDeleteResult(t, candidate, result)
		})
	}
}

func GeneratedRuntimePlainDeleteRequiresConfirmRead(candidate CommonErrorCase) bool {
	return candidate.HTTPStatusCode == 409 &&
		(candidate.ErrorCode == "IncorrectState" || candidate.ErrorCode == "ExternalServerIncorrectState")
}

func runGeneratedRuntimePlainMutationMatrix(
	t *testing.T,
	operation Operation,
	run func(*testing.T, CommonErrorCase) GeneratedRuntimePlainMutationResult,
) {
	t.Helper()

	for _, candidate := range CommonErrorMatrix {
		candidate := candidate
		t.Run(candidate.Name(), func(t *testing.T) {
			t.Parallel()

			result := run(t, candidate)
			assertGeneratedRuntimePlainMutationResult(t, operation, candidate, result)
		})
	}
}

func assertGeneratedRuntimePlainMutationResult(
	t *testing.T,
	operation Operation,
	candidate CommonErrorCase,
	result GeneratedRuntimePlainMutationResult,
) {
	t.Helper()

	expectation := expectationForOperation(candidate, operation)
	switch expectation {
	case ExpectationFatal, ExpectationRetryable:
		if result.Err == nil {
			t.Fatalf("%s/%s unexpectedly succeeded", operation, candidate.Name())
		}
		AssertErrorType(t, result.Err, candidate.NormalizedType)
		if result.Response.IsSuccessful {
			t.Fatalf("%s/%s response = %+v, want failure", operation, candidate.Name(), result.Response)
		}
		if result.StatusReason != string(shared.Failed) {
			t.Fatalf("%s/%s status reason = %q, want %q", operation, candidate.Name(), result.StatusReason, shared.Failed)
		}
	default:
		t.Fatalf("%s/%s expectation = %q, want error result", operation, candidate.Name(), expectation)
	}
}

func assertGeneratedRuntimePlainReadResult(
	t *testing.T,
	candidate CommonErrorCase,
	result GeneratedRuntimePlainReadResult,
) {
	t.Helper()

	expectation := generatedRuntimePlainReadExpectation(candidate)
	switch expectation {
	case ExpectationAbsent:
		if result.Err != nil {
			t.Fatalf("read/%s error = %v, want absent result", candidate.Name(), result.Err)
		}
		if !result.Missing {
			t.Fatalf("read/%s missing = false, want true", candidate.Name())
		}
	case ExpectationFatal, ExpectationRetryable:
		if result.Err == nil {
			t.Fatalf("read/%s unexpectedly succeeded", candidate.Name())
		}
		AssertErrorType(t, result.Err, candidate.NormalizedType)
		if result.Missing {
			t.Fatalf("read/%s missing = true, want classified error", candidate.Name())
		}
	default:
		t.Fatalf("read/%s expectation = %q, want absent or error", candidate.Name(), expectation)
	}
}

func assertGeneratedRuntimePlainDeleteResult(
	t *testing.T,
	candidate CommonErrorCase,
	result GeneratedRuntimePlainDeleteResult,
) {
	t.Helper()

	expectation := generatedRuntimePlainDeleteExpectation(candidate)
	switch expectation {
	case ExpectationDeleted:
		if result.Err != nil {
			t.Fatalf("delete/%s error = %v, want deleted outcome", candidate.Name(), result.Err)
		}
		if !result.Deleted {
			t.Fatalf("delete/%s deleted = false, want true", candidate.Name())
		}
		if result.StatusReason != string(shared.Terminating) {
			t.Fatalf("delete/%s status reason = %q, want %q", candidate.Name(), result.StatusReason, shared.Terminating)
		}
	case ExpectationRetryable:
		if GeneratedRuntimePlainDeleteRequiresConfirmRead(candidate) {
			if result.Err != nil {
				AssertErrorType(t, result.Err, candidate.NormalizedType)
				if result.Deleted {
					t.Fatalf("delete/%s deleted = true, want retryable in-progress result", candidate.Name())
				}
				return
			}
			if result.StatusReason != string(shared.Terminating) {
				t.Fatalf("delete/%s status reason = %q, want %q", candidate.Name(), result.StatusReason, shared.Terminating)
			}
			return
		}

		if result.Err == nil {
			t.Fatalf("delete/%s unexpectedly succeeded", candidate.Name())
		}
		AssertErrorType(t, result.Err, candidate.NormalizedType)
		if result.Deleted {
			t.Fatalf("delete/%s deleted = true, want retryable error result", candidate.Name())
		}
	case ExpectationFatal:
		if result.Err == nil {
			t.Fatalf("delete/%s unexpectedly succeeded", candidate.Name())
		}
		AssertErrorType(t, result.Err, candidate.NormalizedType)
		if result.Deleted {
			t.Fatalf("delete/%s deleted = true, want fatal error result", candidate.Name())
		}
	default:
		t.Fatalf("delete/%s expectation = %q, want deleted or error", candidate.Name(), expectation)
	}
}

func expectationForOperation(candidate CommonErrorCase, operation Operation) Expectation {
	switch operation {
	case OperationRead:
		return candidate.Expectations.Read
	case OperationCreate:
		return candidate.Expectations.Create
	case OperationUpdate:
		return candidate.Expectations.Update
	case OperationDelete:
		return candidate.Expectations.Delete
	default:
		panic(fmt.Sprintf("unsupported operation %q", operation))
	}
}

func generatedRuntimePlainReadExpectation(candidate CommonErrorCase) Expectation {
	if candidate.ErrorCode == "NotFound" {
		return ExpectationAbsent
	}
	return candidate.Expectations.Read
}

func generatedRuntimePlainDeleteExpectation(candidate CommonErrorCase) Expectation {
	switch candidate.ErrorCode {
	case "NotFound", "NotAuthorizedOrNotFound":
		return ExpectationDeleted
	default:
		return candidate.Expectations.Delete
	}
}
