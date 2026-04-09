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
