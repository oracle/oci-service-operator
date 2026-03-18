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
