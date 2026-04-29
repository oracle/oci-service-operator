/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"testing"
)

func TestServiceClientRejectsOpenFormalGapsAtInit(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Unsupported: []UnsupportedSemantic{{Category: "legacy-adapter", StopCondition: "keep manual adapter until gaps close"}}}})
	if _, err := client.CreateOrUpdate(context.Background(), &fakeResource{}, ctrl.Request{}); err == nil || !strings.Contains(err.Error(), "open formal gap legacy-adapter") {
		t.Fatalf("CreateOrUpdate() error = %v, want init failure for open formal gap", err)
	}
}

func TestValidateFormalSemanticsRejectsWorkRequestHelperWithoutExplicitAsyncContract(t *testing.T) {
	t.Parallel()
	err := validateFormalSemantics("RedisCluster", &Semantics{Delete: DeleteSemantics{Policy: "best-effort"}, CreateFollowUp: FollowUpSemantics{Strategy: "read-after-write", Hooks: []Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"}}}})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want explicit async helper failure")
	}
	if !strings.Contains(err.Error(), `workrequest helper requires explicit async strategy "workrequest"`) {
		t.Fatalf("validateFormalSemantics() error = %v, want helper/strategy failure", err)
	}
}

func TestValidateFormalSemanticsRejectsLifecycleAsyncWithWorkRequestHelper(t *testing.T) {
	t.Parallel()
	err := validateFormalSemantics("OpensearchCluster", &Semantics{Async: &AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"}, Delete: DeleteSemantics{Policy: "best-effort"}, CreateFollowUp: FollowUpSemantics{Strategy: "read-after-write", Hooks: []Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling"}}}})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want explicit async helper failure")
	}
	if !strings.Contains(err.Error(), `workrequest helper requires explicit async strategy "workrequest"`) {
		t.Fatalf("validateFormalSemantics() error = %v, want helper/strategy failure", err)
	}
}

func TestValidateFormalSemanticsRejectsInvalidAsyncMetadataEnums(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		semantics *Semantics
		wantErr   string
	}{{name: "unknown strategy", semantics: &Semantics{Async: &AsyncSemantics{Strategy: "eventual", Runtime: "generatedruntime", FormalClassification: "lifecycle"}, Delete: DeleteSemantics{Policy: "best-effort"}}, wantErr: `async.strategy "eventual" must be one of "lifecycle", "workrequest", or "none"`}, {name: "unknown formal classification", semantics: &Semantics{Async: &AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "eventual"}, Delete: DeleteSemantics{Policy: "best-effort"}}, wantErr: `async.formalClassification "eventual" must be one of "lifecycle", "workrequest", or "none"`}, {name: "unknown workrequest source", semantics: &Semantics{Async: &AsyncSemantics{Strategy: "workrequest", Runtime: "handwritten", FormalClassification: "workrequest", WorkRequest: &WorkRequestSemantics{Source: "custom-source", Phases: []string{"create"}}}, Delete: DeleteSemantics{Policy: "best-effort"}}, wantErr: `async.workRequest.source "custom-source" must be one of "service-sdk", "workrequests-service", or "provider-helper"`}, {name: "invalid workrequest phase", semantics: &Semantics{Async: &AsyncSemantics{Strategy: "workrequest", Runtime: "handwritten", FormalClassification: "workrequest", WorkRequest: &WorkRequestSemantics{Source: "service-sdk", Phases: []string{"reconcile"}}}, Delete: DeleteSemantics{Policy: "best-effort"}}, wantErr: `async.workRequest.phases[0] "reconcile" must be one of "create", "update", or "delete"`}, {name: "duplicate workrequest phase", semantics: &Semantics{Async: &AsyncSemantics{Strategy: "workrequest", Runtime: "handwritten", FormalClassification: "workrequest", WorkRequest: &WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create", "create"}}}, Delete: DeleteSemantics{Policy: "best-effort"}}, wantErr: `async.workRequest.phases contains duplicate phase "create"`}, {name: "blank legacy bridge field", semantics: &Semantics{Async: &AsyncSemantics{Strategy: "workrequest", Runtime: "handwritten", FormalClassification: "workrequest", WorkRequest: &WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create"}, LegacyFieldBridge: &WorkRequestLegacyFieldBridge{Update: "   "}}}, Delete: DeleteSemantics{Policy: "best-effort"}}, wantErr: `async.workRequest.legacyFieldBridge.update must not be blank`}}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateFormalSemantics("Thing", test.semantics)
			if err == nil {
				t.Fatal("validateFormalSemantics() error = nil, want invalid async metadata failure")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateFormalSemantics() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestValidateFormalSemanticsRejectsExplicitHandwrittenAsyncRuntime(t *testing.T) {
	t.Parallel()
	err := validateFormalSemantics("Queue", &Semantics{Async: &AsyncSemantics{Strategy: "workrequest", Runtime: "handwritten", FormalClassification: "workrequest", WorkRequest: &WorkRequestSemantics{Source: "service-sdk", Phases: []string{"create", "delete"}}}, Delete: DeleteSemantics{Policy: "best-effort"}})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want handwritten runtime failure")
	}
	if !strings.Contains(err.Error(), `generatedruntime cannot honor explicit async runtime "handwritten"`) {
		t.Fatalf("validateFormalSemantics() error = %v, want handwritten-runtime detail", err)
	}
}

func TestValidateFormalSemanticsBlocksAuxiliaryOperations(t *testing.T) {
	t.Parallel()
	err := validateFormalSemantics("Vcn", &Semantics{Delete: DeleteSemantics{Policy: "best-effort"}, AuxiliaryOperations: []AuxiliaryOperation{{Phase: "list", MethodName: "ListVcns"}}})
	if err == nil {
		t.Fatal("validateFormalSemantics() error = nil, want auxiliary-operation failure")
	}
	if !strings.Contains(err.Error(), "unsupported list auxiliary operation ListVcns") {
		t.Fatalf("validateFormalSemantics() error = %v, want auxiliary-operation detail", err)
	}
}

func TestValidateFormalSemanticsAllowsGeneratedRuntimeWorkRequestRuntime(t *testing.T) {
	t.Parallel()

	err := validateFormalSemantics("Queue", &Semantics{
		Async: &AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		Delete: DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{"DELETED"},
		},
	})
	if err != nil {
		t.Fatalf("validateFormalSemantics() error = %v, want nil", err)
	}
}

func TestValidateGeneratedWorkRequestAsyncHooksRequiresGetWorkRequest(t *testing.T) {
	t.Parallel()

	err := validateGeneratedWorkRequestAsyncHooks(Config[*fakeResource]{
		Kind: "Queue",
		Semantics: &Semantics{
			Async: &AsyncSemantics{
				Strategy:             "workrequest",
				Runtime:              "generatedruntime",
				FormalClassification: "workrequest",
				WorkRequest: &WorkRequestSemantics{
					Source: "service-sdk",
					Phases: []string{"create", "update", "delete"},
				},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				TerminalStates: []string{"DELETED"},
			},
		},
	})
	if err == nil {
		t.Fatal("validateGeneratedWorkRequestAsyncHooks() error = nil, want missing GetWorkRequest failure")
	}
	if !strings.Contains(err.Error(), "Async.GetWorkRequest") {
		t.Fatalf("validateGeneratedWorkRequestAsyncHooks() error = %v, want GetWorkRequest detail", err)
	}
}
