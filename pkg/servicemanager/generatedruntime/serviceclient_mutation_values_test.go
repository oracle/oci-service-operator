package generatedruntime

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestMutationValuesMergesStatusOnlyFieldsWithLiveResponse(t *testing.T) {
	t.Parallel()

	resource := &mysqlv1beta1.DbSystem{
		Status: mysqlv1beta1.DbSystemStatus{
			DisplayName: "status-name",
			Source:      mysqlv1beta1.DbSystemSourceObservedState{},
			AdminUsername: shared.UsernameSource{
				Secret: shared.SecretSource{SecretName: "admin-secret"},
			},
			AdminPassword: shared.PasswordSource{
				Secret: shared.SecretSource{SecretName: "admin-secret"},
			},
		},
	}

	_, currentValues, err := mutationValues(resource, mysqlsdk.GetDbSystemResponse{
		DbSystem: mysqlsdk.DbSystem{
			Id:          common.String("ocid1.mysqldbsystem.oc1..created"),
			DisplayName: common.String("live-name"),
		},
	})
	if err != nil {
		t.Fatalf("mutationValues() error = %v", err)
	}

	if got, ok := lookupValueByPath(currentValues, "displayName"); !ok || got != "live-name" {
		t.Fatalf("currentValues.displayName = %#v, want live response value", got)
	}

	if got, ok := lookupValueByPath(currentValues, "adminUsername.secret.secretName"); !ok || got != "admin-secret" {
		t.Fatalf("currentValues.adminUsername.secret.secretName = %#v, want status-only secret source preserved", got)
	}
	if got, ok := lookupValueByPath(currentValues, "adminPassword.secret.secretName"); !ok || got != "admin-secret" {
		t.Fatalf("currentValues.adminPassword.secret.secretName = %#v, want status-only secret source preserved", got)
	}
}

func TestValidateForceNewFieldsIgnoresEmptyNestedSpecObjects(t *testing.T) {
	t.Parallel()

	client := ServiceClient[*mysqlv1beta1.DbSystem]{
		config: Config[*mysqlv1beta1.DbSystem]{
			Kind: "DbSystem",
			Semantics: &Semantics{
				Mutation: MutationSemantics{
					ForceNew: []string{"source"},
				},
			},
		},
	}

	err := client.validateForceNewFields(
		map[string]any{"source": map[string]any{}},
		map[string]any{"source": map[string]any{"sourceType": "BACKUP"}},
	)
	if err != nil {
		t.Fatalf("validateForceNewFields() error = %v, want empty spec object ignored", err)
	}
}
