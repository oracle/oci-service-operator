package errortest

import (
	"fmt"
	"testing"
)

// ManualRuntimeClassifierContract captures the reviewed read/delete classifier
// behavior for a manual runtime family member.
type ManualRuntimeClassifierContract struct {
	Name                         string
	DeleteAuthShaped404AsDeleted bool
}

func NewManualRuntimeClassifierContract(name string, deleteAuthShaped404AsDeleted bool) ManualRuntimeClassifierContract {
	return ManualRuntimeClassifierContract{
		Name:                         name,
		DeleteAuthShaped404AsDeleted: deleteAuthShaped404AsDeleted,
	}
}

func ManualRuntimeClassifierContractFromReviewedRegistration(service string, kind string) (ManualRuntimeClassifierContract, error) {
	key := resourceKey(service, kind)
	registration, ok := ReviewedAPIErrorCoverageRegistry.Registrations[key]
	if !ok {
		return ManualRuntimeClassifierContract{}, fmt.Errorf("reviewed registration %s is not defined", key)
	}
	if registration.Family != APIErrorCoverageFamilyManualRuntime {
		return ManualRuntimeClassifierContract{}, fmt.Errorf("reviewed registration %s family = %q, want %q", key, registration.Family, APIErrorCoverageFamilyManualRuntime)
	}

	return ManualRuntimeClassifierContract{
		Name:                         key,
		DeleteAuthShaped404AsDeleted: registration.DeleteNotFoundSemantics == deleteNotFoundGeneratedRuntime || registration.DeleteNotFoundSemantics == deleteNotFoundManualRuntime,
	}, nil
}

func RunManualRuntimeClassifierContract(
	t *testing.T,
	contract ManualRuntimeClassifierContract,
	readNotFound func(error) bool,
	deleteNotFound func(error) bool,
) {
	t.Helper()

	if readNotFound == nil {
		t.Fatalf("%s read classifier is nil", contract.Name)
	}
	if deleteNotFound == nil {
		t.Fatalf("%s delete classifier is nil", contract.Name)
	}

	for _, candidate := range CommonErrorMatrix {
		candidate := candidate
		t.Run(candidate.Name(), func(t *testing.T) {
			err := NewServiceErrorFromCase(candidate)

			wantReadNotFound := candidate.Expectations.Read == ExpectationAbsent
			wantDeleteNotFound := candidate.Expectations.Delete == ExpectationDeleted
			if contract.DeleteAuthShaped404AsDeleted &&
				candidate.HTTPStatusCode == 404 &&
				candidate.ErrorCode == "NotAuthorizedOrNotFound" {
				wantDeleteNotFound = true
			}

			if got := readNotFound(err); got != wantReadNotFound {
				t.Fatalf("%s read classifier(%s) = %t, want %t", contract.Name, candidate.Name(), got, wantReadNotFound)
			}
			if got := deleteNotFound(err); got != wantDeleteNotFound {
				t.Fatalf("%s delete classifier(%s) = %t, want %t", contract.Name, candidate.Name(), got, wantDeleteNotFound)
			}
		})
	}
}
