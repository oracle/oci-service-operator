package errortest

import (
	"fmt"
	"strings"
	"testing"
)

// AsyncFollowUpMatrixCase captures one focused OCI error expectation for an
// async follow-up path such as read-after-write, confirm-delete, or manual
// work-request polling.
type AsyncFollowUpMatrixCase struct {
	Candidate          CommonErrorCase
	WantDeleted        bool
	WantSuccessful     bool
	WantRequeue        bool
	WantErrorType      string
	WantErrorSubstring string
}

// AsyncFollowUpResult is the normalized assertion surface used by async
// follow-up family tests across generatedruntime and handwritten runtimes.
type AsyncFollowUpResult struct {
	Err        error
	Deleted    bool
	Successful bool
	Requeue    bool
}

func MustCommonErrorCase(t *testing.T, statusCode int, errorCode string) CommonErrorCase {
	t.Helper()

	candidate, ok := LookupCommonErrorCase(statusCode, errorCode)
	if !ok {
		t.Fatalf("LookupCommonErrorCase(%d, %q) returned no entry", statusCode, errorCode)
	}
	return candidate
}

func ReviewedRegistrationForFamily(
	t *testing.T,
	service string,
	kind string,
	wantFamily APIErrorCoverageFamily,
) APIErrorCoverageRegistration {
	t.Helper()

	key := resourceKey(service, kind)
	registration, ok := ReviewedAPIErrorCoverageRegistry.Registrations[key]
	if !ok {
		t.Fatalf("reviewed registration %q was not found", key)
	}
	if registration.Family != wantFamily {
		t.Fatalf("%s family = %q, want %q", key, registration.Family, wantFamily)
	}
	return registration
}

func RunAsyncFollowUpMatrix(
	t *testing.T,
	cases []AsyncFollowUpMatrixCase,
	invoke func(t *testing.T, candidate CommonErrorCase) AsyncFollowUpResult,
) {
	t.Helper()

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.Candidate.Name(), func(t *testing.T) {
			result := invoke(t, testCase.Candidate)

			wantErr := testCase.WantErrorType != "" || testCase.WantErrorSubstring != ""
			if wantErr && result.Err == nil {
				t.Fatalf(
					"%s err = nil, want error type %q substring %q",
					testCase.Candidate.Name(),
					testCase.WantErrorType,
					testCase.WantErrorSubstring,
				)
			}
			if !wantErr && result.Err != nil {
				t.Fatalf("%s err = %v, want nil", testCase.Candidate.Name(), result.Err)
			}
			if testCase.WantErrorType != "" {
				AssertErrorType(t, result.Err, testCase.WantErrorType)
			}
			if testCase.WantErrorSubstring != "" && !strings.Contains(result.Err.Error(), testCase.WantErrorSubstring) {
				t.Fatalf(
					"%s err = %q, want substring %q",
					testCase.Candidate.Name(),
					result.Err.Error(),
					testCase.WantErrorSubstring,
				)
			}
			if result.Deleted != testCase.WantDeleted {
				t.Fatalf("%s deleted = %t, want %t", testCase.Candidate.Name(), result.Deleted, testCase.WantDeleted)
			}
			if result.Successful != testCase.WantSuccessful {
				t.Fatalf("%s successful = %t, want %t", testCase.Candidate.Name(), result.Successful, testCase.WantSuccessful)
			}
			if result.Requeue != testCase.WantRequeue {
				t.Fatalf("%s requeue = %t, want %t", testCase.Candidate.Name(), result.Requeue, testCase.WantRequeue)
			}
		})
	}
}

func FocusedAsyncFollowUpCases(t *testing.T) map[string]CommonErrorCase {
	t.Helper()

	return map[string]CommonErrorCase{
		"notfound":    MustCommonErrorCase(t, 404, "NotFound"),
		"auth404":     MustCommonErrorCase(t, 404, "NotAuthorizedOrNotFound"),
		"conflict":    MustCommonErrorCase(t, 409, "IncorrectState"),
		"internal":    MustCommonErrorCase(t, 500, "InternalServerError"),
		"unavailable": MustCommonErrorCase(t, 503, "ServiceUnavailable"),
	}
}

func DescribeReviewedRegistration(registration APIErrorCoverageRegistration) string {
	return fmt.Sprintf(
		"%s/%s family=%s delete=%s conflict=%s deviation=%q",
		registration.Resource.Service,
		registration.Resource.Kind,
		registration.Family,
		registration.DeleteNotFoundSemantics,
		registration.RetryableConflictSemantics,
		registration.Deviation,
	)
}
