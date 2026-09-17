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

func TestBuildRuntimeSemanticsIncludesSelectedAuxiliaryUpdateOperation(t *testing.T) {
	t.Parallel()

	formalModel := newThingFormalModelWithUpdateSubset([]string{"UpdateThing", "ChangeThingCompartment"})
	runtime := &RuntimeModel{Update: &RuntimeOperationModel{MethodName: "UpdateThing"}}

	semantics := buildRuntimeSemanticsModel(formalModel, runtime)
	if semantics == nil {
		t.Fatal("buildRuntimeSemanticsModel() = nil")
	}
	if len(semantics.AuxiliaryOperations) != 1 {
		t.Fatalf("AuxiliaryOperations = %#v, want one selected update operation", semantics.AuxiliaryOperations)
	}
	got := semantics.AuxiliaryOperations[0]
	if got.Phase != "update" || got.MethodName != "ChangeThingCompartment" {
		t.Fatalf("AuxiliaryOperations[0] = %#v, want selected compartment update", got)
	}
}

func TestBuildRuntimeSemanticsHonorsRepoAuthoredCreateAndDeleteOperationSubsets(t *testing.T) {
	t.Parallel()

	formalModel := newThingFormalModelWithPhaseSubsets(
		[]string{"CreateThing"},
		[]string{"DeleteThing"},
	)
	runtime := &RuntimeModel{
		Create: &RuntimeOperationModel{MethodName: "CreateThing"},
		Delete: &RuntimeOperationModel{MethodName: "DeleteThing"},
	}

	semantics := buildRuntimeSemanticsModel(formalModel, runtime)
	if semantics == nil {
		t.Fatal("buildRuntimeSemanticsModel() = nil")
	}
	if len(semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want provider-only create/delete actions excluded", semantics.AuxiliaryOperations)
	}
}

func TestBuildRuntimeSemanticsAllowsExplicitEmptyDeleteSubsetWithoutImportedPrimary(t *testing.T) {
	t.Parallel()

	formalModel := &FormalModel{
		Binding: formal.ControllerBinding{
			Import: formal.ImportModel{
				Operations: formal.Operations{
					Delete: []formal.OperationBinding{{
						Operation:    "ScheduleThingDeletion",
						RequestType:  "ScheduleThingDeletionRequest",
						ResponseType: "ScheduleThingDeletionResponse",
					}},
				},
			},
		},
		RuntimeLifecycle: &formal.RuntimeLifecycleSpec{
			RepoAuthored: &formal.RuntimeLifecycleRepoAuthoredSemantics{
				Operations: &formal.RuntimeLifecycleOperationSemantics{Delete: []string{}},
			},
		},
	}
	runtime := &RuntimeModel{Delete: &RuntimeOperationModel{MethodName: "DeleteThing"}}

	semantics := buildRuntimeSemanticsModel(formalModel, runtime)
	if semantics == nil {
		t.Fatal("buildRuntimeSemanticsModel() = nil")
	}
	if len(semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want provider-only delete action excluded", semantics.AuxiliaryOperations)
	}
}

func TestValidateRuntimeUpdateOperationSubsetAllowsEmptySubsetWithoutImportedPrimary(t *testing.T) {
	t.Parallel()

	formalModel := &FormalModel{
		Binding: formal.ControllerBinding{
			Import: formal.ImportModel{
				Operations: formal.Operations{
					Update: []formal.OperationBinding{{Operation: "ManageThing"}},
				},
			},
		},
		RuntimeLifecycle: &formal.RuntimeLifecycleSpec{
			RepoAuthored: &formal.RuntimeLifecycleRepoAuthoredSemantics{
				Operations: &formal.RuntimeLifecycleOperationSemantics{Update: []string{}},
			},
		},
	}
	err := validateRuntimeUpdateOperationSubset(
		formalModel,
		&RuntimeModel{Update: &RuntimeOperationModel{MethodName: "UpdateThing"}},
	)
	if err != nil {
		t.Fatalf("validateRuntimeUpdateOperationSubset() error = %v, want provider-only subset exclusion to be valid", err)
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

func newThingFormalModelWithPhaseSubsets(create, delete []string) *FormalModel {
	return &FormalModel{
		Binding: formal.ControllerBinding{
			Import: formal.ImportModel{
				Operations: formal.Operations{
					Create: []formal.OperationBinding{
						{Operation: "ImportThing", RequestType: "ImportThingRequest", ResponseType: "ImportThingResponse"},
						{Operation: "CreateThing", RequestType: "CreateThingRequest", ResponseType: "CreateThingResponse"},
					},
					Delete: []formal.OperationBinding{
						{Operation: "DeleteThing", RequestType: "DeleteThingRequest", ResponseType: "DeleteThingResponse"},
						{Operation: "ScheduleThingDeletion", RequestType: "ScheduleThingDeletionRequest", ResponseType: "ScheduleThingDeletionResponse"},
					},
				},
			},
		},
		RuntimeLifecycle: &formal.RuntimeLifecycleSpec{
			RepoAuthored: &formal.RuntimeLifecycleRepoAuthoredSemantics{
				Operations: &formal.RuntimeLifecycleOperationSemantics{
					Create: create,
					Delete: delete,
				},
			},
		},
	}
}
