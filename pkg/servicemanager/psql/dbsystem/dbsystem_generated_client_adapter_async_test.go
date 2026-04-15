package dbsystem

import (
	"context"
	"strings"
	"testing"

	psqlsdk "github.com/oracle/oci-go-sdk/v65/psql"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestManualDbSystemServiceClientUpdateFollowUpErrorsRemainExplicit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		followUpErr   error
		wantSubstring string
	}{
		{
			name:          "auth shaped not found",
			followUpErr:   errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "dbsystem not found"),
			wantSubstring: "dbsystem not found",
		},
		{
			name:          "internal server error",
			followUpErr:   errortest.NewServiceError(500, "InternalServerError", "update readback failed"),
			wantSubstring: "update readback failed",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			getCalls := 0
			client := manualDbSystemServiceClient{
				sdk: fakeDbSystemOCIClient{
					listDbSystems: func(_ context.Context, _ psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error) {
						return psqlsdk.ListDbSystemsResponse{
							DbSystemCollection: psqlsdk.DbSystemCollection{
								Items: []psqlsdk.DbSystemSummary{
									sdkDbSystemSummary("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateActive),
								},
							},
						}, nil
					},
					getDbSystem: func(_ context.Context, _ psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error) {
						getCalls++
						if getCalls == 1 {
							current := sdkDbSystem("ocid1.dbsystem.oc1..existing", "sample-db", psqlsdk.DbSystemLifecycleStateActive)
							current.Description = commonString("old description")
							return psqlsdk.GetDbSystemResponse{DbSystem: current}, nil
						}
						return psqlsdk.GetDbSystemResponse{}, test.followUpErr
					},
					updateDbSystem: func(_ context.Context, _ psqlsdk.UpdateDbSystemRequest) (psqlsdk.UpdateDbSystemResponse, error) {
						return psqlsdk.UpdateDbSystemResponse{}, nil
					},
				},
				log: discardDbSystemLogger(),
			}

			resource := testDbSystemResource()
			resource.Spec.Description = "new description"

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want update follow-up failure")
			}
			if !strings.Contains(err.Error(), test.wantSubstring) {
				t.Fatalf("CreateOrUpdate() error = %v, want substring %q", err, test.wantSubstring)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful follow-up failure", response)
			}
			if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Failed {
				t.Fatalf("status conditions = %#v, want trailing Failed condition", resource.Status.OsokStatus.Conditions)
			}
		})
	}
}

func commonString(value string) *string {
	return &value
}
