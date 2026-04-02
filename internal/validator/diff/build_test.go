package diff_test

import (
	"testing"

	"github.com/oracle/oci-service-operator/internal/validator/allowlist"
	"github.com/oracle/oci-service-operator/internal/validator/diff"
	"github.com/oracle/oci-service-operator/internal/validator/provider"
	"github.com/oracle/oci-service-operator/internal/validator/sdk"
)

func TestBuildReportFlagsUnusedField(t *testing.T) {
	sdkStructs := []sdk.SDKStruct{{
		QualifiedName: "database.CreateAutonomousDatabaseDetails",
		Fields:        []sdk.SDKField{{Name: "CompartmentId"}, {Name: "DisplayName"}},
	}}
	providerAnalysis := provider.Analysis{
		Usages: []provider.FieldUsage{{StructType: "database.CreateAutonomousDatabaseDetails", FieldName: "CompartmentId"}},
	}
	allow := allowlist.Allowlist{Structs: map[string]allowlist.Struct{}}

	report := diff.BuildReport(sdkStructs, providerAnalysis, allow)
	if len(report.Structs) != 1 {
		t.Fatalf("expected one struct")
	}
	found := false
	for _, field := range report.Structs[0].Fields {
		if field.FieldName == "DisplayName" && field.Status == diff.FieldStatusUnclassified {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected DisplayName to be marked unclassified")
	}
}
