package generator

import (
	"testing"

	"github.com/oracle/oci-service-operator/internal/ocisdk"
)

func TestSharedSchemaType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  ocisdk.Field
		want   string
		wantOK bool
	}{
		{
			name:   "defined tags keep existing shared map type",
			field:  ocisdk.Field{Type: "map[string]map[string]interface{}"},
			want:   "map[string]shared.MapValue",
			wantOK: true,
		},
		{
			name:   "plain json object maps use shared json values",
			field:  ocisdk.Field{Type: "map[string]interface{}"},
			want:   "map[string]shared.JSONValue",
			wantOK: true,
		},
		{
			name:   "wrapped json object maps keep collection shape",
			field:  ocisdk.Field{Type: "[]map[string]interface{}"},
			want:   "[]map[string]shared.JSONValue",
			wantOK: true,
		},
		{
			name:   "non json map types are ignored",
			field:  ocisdk.Field{Type: "map[string]string"},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := sharedSchemaType(tt.field)
			if ok != tt.wantOK {
				t.Fatalf("sharedSchemaType() ok = %t, want %t", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("sharedSchemaType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeInterfaceFamilyFieldsKeepsImplementationOnlyFieldsOptional(t *testing.T) {
	t.Parallel()

	family := ocisdk.InterfaceFamily{
		Base: ocisdk.Struct{
			Name: "createwidgetbase",
			Fields: []ocisdk.Field{
				{
					Name:      "CompartmentId",
					Type:      "string",
					JSONName:  "compartmentId",
					Mandatory: true,
				},
				{
					Name:      "Source",
					Type:      "string",
					JSONName:  "source",
					Mandatory: false,
				},
			},
		},
		Implementations: []ocisdk.Struct{
			{
				Name: "CreateWidgetCloneDetails",
				Fields: []ocisdk.Field{
					{
						Name:      "CompartmentId",
						Type:      "string",
						JSONName:  "compartmentId",
						Mandatory: true,
					},
					{
						Name:      "Source",
						Type:      "string",
						JSONName:  "source",
						Mandatory: true,
					},
					{
						Name:      "SourceId",
						Type:      "string",
						JSONName:  "sourceId",
						Mandatory: true,
					},
				},
			},
			{
				Name: "CreateWidgetDisasterRecoveryDetails",
				Fields: []ocisdk.Field{
					{
						Name:      "CompartmentId",
						Type:      "string",
						JSONName:  "compartmentId",
						Mandatory: true,
					},
					{
						Name:      "SourceId",
						Type:      "string",
						JSONName:  "sourceId",
						Mandatory: true,
					},
					{
						Name:      "RecoveryType",
						Type:      "string",
						JSONName:  "recoveryType",
						Mandatory: true,
					},
				},
			},
		},
	}

	merged := mergeInterfaceFamilyFields(family, true)

	compartmentID := findMergedInterfaceField(t, merged, "CompartmentId")
	if !compartmentID.Mandatory {
		t.Fatal("CompartmentId should remain mandatory when it comes from the base polymorphic shape")
	}

	source := findMergedInterfaceField(t, merged, "Source")
	if source.Mandatory {
		t.Fatal("Source should remain optional when only implementation variants mark it mandatory")
	}

	sourceID := findMergedInterfaceField(t, merged, "SourceId")
	if sourceID.Mandatory {
		t.Fatal("SourceId should be optional because it only exists on specific polymorphic variants")
	}

	recoveryType := findMergedInterfaceField(t, merged, "RecoveryType")
	if recoveryType.Mandatory {
		t.Fatal("RecoveryType should be optional because it only exists on a specific polymorphic variant")
	}
}

func TestRecursiveNestedStructFallsBackToSharedJSONValue(t *testing.T) {
	t.Parallel()

	synthesizer := &fieldSynthesizer{
		resourceKind:        "Thing",
		helperIndex:         make(map[string]int),
		normalizedTypeNames: make(map[string]string),
	}

	field, ok := synthesizer.buildGeneratedField(
		ocisdk.Field{
			Name:     "Root",
			Type:     "ThingRootDetails",
			JSONName: "root",
			Kind:     ocisdk.FieldKindStruct,
			NestedFields: []ocisdk.Field{
				{
					Name:     "Children",
					Type:     "[]ThingRootDetails",
					JSONName: "children",
					Kind:     ocisdk.FieldKindStruct,
					NestedFields: []ocisdk.Field{
						{
							Name:           "Name",
							Type:           "string",
							JSONName:       "name",
							RenderableType: "string",
						},
					},
				},
			},
		},
		fieldRenderingOptions{scope: fieldScopeSpec},
		[]string{"Root"},
		[]string{"Root"},
		[]string{"CreateThingDetails"},
	)
	if !ok {
		t.Fatal("buildGeneratedField() ok = false")
	}
	if field.Type != "ThingRoot" {
		t.Fatalf("field.Type = %q, want %q", field.Type, "ThingRoot")
	}
	if len(synthesizer.helperTypes) != 1 {
		t.Fatalf("helperTypes length = %d, want 1", len(synthesizer.helperTypes))
	}

	helper := synthesizer.helperTypes[0]
	if helper.Name != "ThingRoot" {
		t.Fatalf("helper.Name = %q, want %q", helper.Name, "ThingRoot")
	}
	if len(helper.Fields) != 1 {
		t.Fatalf("helper.Fields length = %d, want 1", len(helper.Fields))
	}
	if helper.Fields[0].Name != "Children" {
		t.Fatalf("helper field name = %q, want %q", helper.Fields[0].Name, "Children")
	}
	if helper.Fields[0].Type != "[]shared.JSONValue" {
		t.Fatalf("recursive helper field type = %q, want %q", helper.Fields[0].Type, "[]shared.JSONValue")
	}
}

func TestSensitiveObservedStateFieldsAreExcluded(t *testing.T) {
	t.Parallel()

	synthesizer := &fieldSynthesizer{
		resourceKind:        "Thing",
		helperIndex:         make(map[string]int),
		normalizedTypeNames: make(map[string]string),
	}

	if _, ok := synthesizer.buildGeneratedField(
		ocisdk.Field{
			Name:           "Password",
			Type:           "string",
			JSONName:       "password",
			RenderableType: "string",
		},
		fieldRenderingOptions{scope: fieldScopeStatus},
		[]string{"Password"},
		[]string{"Password"},
		[]string{"Thing"},
	); ok {
		t.Fatal("buildGeneratedField() included a sensitive observed-state field")
	}

	credentialsField := ocisdk.Field{
		Name:     "Credentials",
		Type:     "ThingCredentials",
		JSONName: "credentials",
		Kind:     ocisdk.FieldKindStruct,
		NestedFields: []ocisdk.Field{
			{
				Name:           "Username",
				Type:           "string",
				JSONName:       "username",
				RenderableType: "string",
			},
			{
				Name:           "Password",
				Type:           "string",
				JSONName:       "password",
				RenderableType: "string",
			},
		},
	}
	if _, ok := synthesizer.buildGeneratedField(
		credentialsField,
		fieldRenderingOptions{scope: fieldScopeSpec},
		[]string{"Credentials"},
		[]string{"Credentials"},
		[]string{"Thing"},
	); !ok {
		t.Fatal("buildGeneratedField() ok = false for spec wrapper")
	}
	specHelper := findHelperType(t, synthesizer.helperTypes, "ThingCredentials")
	assertFieldNamesPresent(t, specHelper.Name+" fields", specHelper.Fields, "Username", "Password")

	renderedHelperField, ok := synthesizer.buildGeneratedField(
		ocisdk.Field{
			Name:           "Credentials",
			Type:           "ThingCredentials",
			JSONName:       "credentials",
			RenderableType: "ThingCredentials",
		},
		fieldRenderingOptions{scope: fieldScopeStatus},
		[]string{"Credentials"},
		[]string{"Credentials"},
		[]string{"Thing"},
	)
	if !ok {
		t.Fatal("buildGeneratedField() ok = false for rendered helper reference")
	}
	if renderedHelperField.Type != "shared.JSONValue" {
		t.Fatalf("rendered helper status field type = %q, want %q", renderedHelperField.Type, "shared.JSONValue")
	}

	field, ok := synthesizer.buildGeneratedField(
		credentialsField,
		fieldRenderingOptions{scope: fieldScopeStatus},
		[]string{"Credentials"},
		[]string{"Credentials"},
		[]string{"Thing"},
	)
	if !ok {
		t.Fatal("buildGeneratedField() ok = false for non-sensitive wrapper")
	}
	if field.Type != "ThingCredentialsObservedState" {
		t.Fatalf("field.Type = %q, want %q", field.Type, "ThingCredentialsObservedState")
	}
	helper := findHelperType(t, synthesizer.helperTypes, "ThingCredentialsObservedState")
	assertFieldNamesPresent(t, helper.Name+" fields", helper.Fields, "Username")
	assertFieldNamesAbsent(t, helper.Name+" fields", helper.Fields, "Password")
}

func findMergedInterfaceField(t *testing.T, fields []ocisdk.Field, name string) ocisdk.Field {
	t.Helper()

	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}

	t.Fatalf("field %q not found in %#v", name, fields)
	return ocisdk.Field{}
}
