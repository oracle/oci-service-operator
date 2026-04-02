package shared

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestJSONValueRoundTrip(t *testing.T) {
	t.Parallel()

	type payload struct {
		Value map[string]JSONValue `json:"value"`
	}

	original := []byte(`{"value":{"count":3,"enabled":true,"labels":["a","b"],"nested":{"name":"row"},"optional":null}}`)

	var decoded payload
	if err := json.Unmarshal(original, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	encoded, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var want any
	if err := json.Unmarshal(original, &want); err != nil {
		t.Fatalf("json.Unmarshal(want) error = %v", err)
	}
	var got any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("json.Unmarshal(got) error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-tripped payload = %#v, want %#v", got, want)
	}
}
