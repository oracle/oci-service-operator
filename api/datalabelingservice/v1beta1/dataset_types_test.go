package v1beta1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDatasetSpecOmitsZeroOptionalStructs(t *testing.T) {
	payload, err := json.Marshal(DatasetSpec{})
	if err != nil {
		t.Fatalf("Marshal(DatasetSpec{}) error = %v", err)
	}

	body := string(payload)
	for _, omitted := range []string{
		"initialImportDatasetConfiguration",
		"initialRecordGenerationConfiguration",
	} {
		if strings.Contains(body, omitted) {
			t.Fatalf("zero DatasetSpec JSON = %s, unexpectedly contains %q", body, omitted)
		}
	}
}

func TestDatasetStatusOmitsZeroOptionalStructs(t *testing.T) {
	payload, err := json.Marshal(DatasetStatus{})
	if err != nil {
		t.Fatalf("Marshal(DatasetStatus{}) error = %v", err)
	}

	body := string(payload)
	for _, omitted := range []string{
		"initialImportDatasetConfiguration",
		"initialRecordGenerationConfiguration",
	} {
		if strings.Contains(body, omitted) {
			t.Fatalf("zero DatasetStatus JSON = %s, unexpectedly contains %q", body, omitted)
		}
	}
}
