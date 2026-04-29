package budget

import (
	"slices"
	"testing"

	budgetv1beta1 "github.com/oracle/oci-service-operator/api/budget/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func TestBudgetGeneratedRuntimeSemanticsMatchReviewedFormalContract(t *testing.T) {
	semantics := newBudgetRuntimeSemantics()
	if semantics == nil {
		t.Fatal("newBudgetRuntimeSemantics() returned nil")
	}

	if semantics.Async == nil {
		t.Fatal("Budget async semantics = nil, want explicit async none contract")
	}
	if semantics.Async.Strategy != "none" {
		t.Fatalf("Budget async strategy = %q, want %q", semantics.Async.Strategy, "none")
	}
	if semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("Budget async runtime = %q, want %q", semantics.Async.Runtime, "generatedruntime")
	}
	if semantics.Async.FormalClassification != "none" {
		t.Fatalf("Budget async formal classification = %q, want %q", semantics.Async.FormalClassification, "none")
	}
	if !slices.Equal(semantics.Lifecycle.ProvisioningStates, []string{}) {
		t.Fatalf("Budget provisioning states = %v, want none", semantics.Lifecycle.ProvisioningStates)
	}
	if !slices.Equal(semantics.Lifecycle.UpdatingStates, []string{}) {
		t.Fatalf("Budget updating states = %v, want none", semantics.Lifecycle.UpdatingStates)
	}
	if !slices.Equal(semantics.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"}) {
		t.Fatalf("Budget active states = %v, want [ACTIVE INACTIVE]", semantics.Lifecycle.ActiveStates)
	}
	if !slices.Equal(semantics.Delete.PendingStates, []string{}) {
		t.Fatalf("Budget delete pending states = %v, want none", semantics.Delete.PendingStates)
	}
	if !slices.Equal(semantics.Delete.TerminalStates, []string{"NOT_FOUND"}) {
		t.Fatalf("Budget delete terminal states = %v, want [NOT_FOUND]", semantics.Delete.TerminalStates)
	}
	if semantics.List == nil {
		t.Fatal("Budget list semantics = nil, want explicit pre-create lookup match fields")
	}
	if !slices.Equal(semantics.List.MatchFields, []string{"compartmentId", "displayName"}) {
		t.Fatalf("Budget list match fields = %v, want [compartmentId displayName]", semantics.List.MatchFields)
	}
	if semantics.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("Budget create follow-up = %q, want %q", semantics.CreateFollowUp.Strategy, "read-after-write")
	}
	if semantics.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("Budget update follow-up = %q, want %q", semantics.UpdateFollowUp.Strategy, "read-after-write")
	}
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("Budget delete follow-up = %q, want %q", semantics.DeleteFollowUp.Strategy, "confirm-delete")
	}
}

func TestGuardBudgetExistingBeforeCreateSkipsReuseWithoutDisplayName(t *testing.T) {
	resource := &budgetv1beta1.Budget{}
	decision, err := guardBudgetExistingBeforeCreate(nil, resource)
	if err != nil {
		t.Fatalf("guardBudgetExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf(
			"guardBudgetExistingBeforeCreate(empty displayName) = %q, want %q",
			decision,
			generatedruntime.ExistingBeforeCreateDecisionSkip,
		)
	}

	resource.Spec.DisplayName = "budget-a"
	decision, err = guardBudgetExistingBeforeCreate(nil, resource)
	if err != nil {
		t.Fatalf("guardBudgetExistingBeforeCreate(named budget) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf(
			"guardBudgetExistingBeforeCreate(named budget) = %q, want %q",
			decision,
			generatedruntime.ExistingBeforeCreateDecisionAllow,
		)
	}
}
