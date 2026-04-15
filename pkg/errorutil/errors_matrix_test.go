package errorutil

import (
	"testing"

	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
)

func TestCommonErrorMatrixMatchesCurrentNormalization(t *testing.T) {
	t.Parallel()

	for _, candidate := range errortest.CommonErrorMatrix {
		candidate := candidate
		t.Run(candidate.Name(), func(t *testing.T) {
			t.Parallel()

			isSuccess, err := NewServiceFailureFromResponse(
				candidate.ErrorCode,
				candidate.HTTPStatusCode,
				"opc-request-id",
				candidate.Name(),
			)
			if isSuccess {
				t.Fatal("NewServiceFailureFromResponse() = success, want classified error")
			}
			errortest.AssertErrorType(t, err, candidate.NormalizedType)
		})
	}
}

func TestCommonErrorMatrixLookup(t *testing.T) {
	t.Parallel()

	candidate, ok := errortest.LookupCommonErrorCase(404, NotFound)
	if !ok {
		t.Fatal("LookupCommonErrorCase(404, NotFound) = missing, want matrix row")
	}
	if candidate.NormalizedType != "errorutil.NotFoundOciError" {
		t.Fatalf("LookupCommonErrorCase normalized type = %q, want errorutil.NotFoundOciError", candidate.NormalizedType)
	}
	if candidate.Expectations.Delete != errortest.ExpectationDeleted {
		t.Fatalf("LookupCommonErrorCase delete expectation = %q, want %q", candidate.Expectations.Delete, errortest.ExpectationDeleted)
	}
}
