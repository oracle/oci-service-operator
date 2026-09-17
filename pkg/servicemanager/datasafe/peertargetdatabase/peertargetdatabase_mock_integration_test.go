/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package peertargetdatabase

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func TestMockIntegrationPeerTargetDatabaseCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := newTestPeerTargetDatabase()
	ocimock.InitializeResource(resource, "mock-peer-target-database")
	resource.Annotations[peerTargetDatabaseTargetDatabaseIDAnnotation] = "<ocid:1>"
	updatedSpec := resource.Spec
	updatedSpec.Description = "updated description"
	createdState := newSDKPeerTargetDatabaseWithDescription(7, resource.Spec.DisplayName, resource.Spec.Description)
	updatedState := newSDKPeerTargetDatabaseWithDescription(7, updatedSpec.DisplayName, updatedSpec.Description)
	deletedState := newSDKPeerTargetDatabaseWithDescription(7, updatedSpec.DisplayName, updatedSpec.Description, datasafesdk.TargetDatabaseLifecycleStateDeleted)

	responder, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[datasafesdk.PeerTargetDatabase]{
		CollectionPath: "/20181201/targetDatabases/<ocid:1>/peerTargetDatabases", ItemPath: "/20181201/targetDatabases/<ocid:1>/peerTargetDatabases/7",
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  false, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		List: func(_ ocimock.Request, present bool, state datasafesdk.PeerTargetDatabase) (ocimock.Response, error) {
			items := []datasafesdk.PeerTargetDatabaseSummary{}
			if present {
				items = append(items, newSDKPeerTargetDatabaseSummary(7, *state.DisplayName, resource.Spec.DataguardAssociationId))
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": items})
		},
		Create: func(request ocimock.Request) (datasafesdk.PeerTargetDatabase, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero datasafesdk.PeerTargetDatabase
				return zero, ocimock.Response{}, err
			}
			var details datasafesdk.CreatePeerTargetDatabaseDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return datasafesdk.PeerTargetDatabase{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || details.Description == nil || *details.Description != resource.Spec.Description || details.DatabaseDetails == nil {
				return datasafesdk.PeerTargetDatabase{}, ocimock.Response{}, fmt.Errorf("CreatePeerTargetDatabase details = %+v", details)
			}
			response, err := ocimock.JSONResponse(http.StatusCreated, createdState)
			return createdState, response, err
		},
		Read: func(_ ocimock.Request, state datasafesdk.PeerTargetDatabase) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, _ datasafesdk.PeerTargetDatabase) (datasafesdk.PeerTargetDatabase, ocimock.Response, error) {
			var details datasafesdk.UpdatePeerTargetDatabaseDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return datasafesdk.PeerTargetDatabase{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != updatedSpec.Description {
				return datasafesdk.PeerTargetDatabase{}, ocimock.Response{}, fmt.Errorf("UpdatePeerTargetDatabase details = %+v", details)
			}
			response, err := ocimock.JSONResponse(http.StatusOK, updatedState)
			return updatedState, response, err
		},
		DeleteTransition: func(_ ocimock.Request, _ datasafesdk.PeerTargetDatabase) (datasafesdk.PeerTargetDatabase, ocimock.Response, error) {
			return deletedState, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newPeerTargetDatabaseServiceClientWithOCIClient(datasafesdk.DataSafeClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.PeerTargetDatabase]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.PeerTargetDatabase) error {
			if current.Status.Key != 7 || current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("created PeerTargetDatabase status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.PeerTargetDatabase) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.PeerTargetDatabase) error {
			if current.Status.Key != 7 || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated PeerTargetDatabase status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}
