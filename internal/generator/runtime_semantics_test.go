/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"testing"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func TestBuildRuntimeSemanticsHonorsRepoAuthoredUpdateOperationSubset(t *testing.T) {
	t.Parallel()

	formalModel := newThingFormalModelWithUpdateSubset([]string{"UpdateThing"})
	runtime := &RuntimeModel{
		Update: &RuntimeOperationModel{MethodName: "UpdateThing"},
	}
	if err := validateRuntimeUpdateOperationSubset(formalModel, runtime); err != nil {
		t.Fatalf("validateRuntimeUpdateOperationSubset() error = %v", err)
	}

	semantics := buildRuntimeSemanticsModel(formalModel, runtime)
	if semantics == nil {
		t.Fatal("buildRuntimeSemanticsModel() = nil")
	}
	if len(semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want excluded ChangeThingCompartment to be absent", semantics.AuxiliaryOperations)
	}
}

func TestValidateRuntimeUpdateOperationSubsetRejectsExcludedPrimary(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		subset []string
	}{
		{name: "auxiliary only", subset: []string{"ChangeThingCompartment"}},
		{name: "explicit empty", subset: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateRuntimeUpdateOperationSubset(
				newThingFormalModelWithUpdateSubset(tc.subset),
				&RuntimeModel{Update: &RuntimeOperationModel{MethodName: "UpdateThing"}},
			)
			if err == nil || err.Error() != `repo-authored update-operation subset must include primary runtime update operation "UpdateThing"` {
				t.Fatalf("validateRuntimeUpdateOperationSubset() error = %v, want excluded-primary failure", err)
			}
		})
	}
}

func newThingFormalModelWithUpdateSubset(subset []string) *FormalModel {
	return &FormalModel{
		Binding: formal.ControllerBinding{
			Import: formal.ImportModel{
				Operations: formal.Operations{
					Update: []formal.OperationBinding{
						{
							Operation:    "ChangeThingCompartment",
							RequestType:  "ChangeThingCompartmentRequest",
							ResponseType: "ChangeThingCompartmentResponse",
						},
						{
							Operation:    "UpdateThing",
							RequestType:  "UpdateThingRequest",
							ResponseType: "UpdateThingResponse",
						},
					},
				},
			},
		},
		RuntimeLifecycle: &formal.RuntimeLifecycleSpec{
			RepoAuthored: &formal.RuntimeLifecycleRepoAuthoredSemantics{
				Operations: &formal.RuntimeLifecycleOperationSemantics{
					Update: subset,
				},
			},
		},
	}
}
