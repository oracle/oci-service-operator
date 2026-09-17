/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package table

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	nosqlsdk "github.com/oracle/oci-go-sdk/v65/nosql"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockTableID = "ocid1.nosqltable.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/nosql/table and formal/imports/nosql/table.json
//   - resource runtime: table_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/nosql
func TestMockIntegrationTableLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &nosqlv1beta1.Table{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-table", Namespace: "default", UID: types.UID("mock-table-uid")},
		Spec: nosqlv1beta1.TableSpec{
			Name:          "mock_table",
			CompartmentId: "ocid1.compartment.oc1..mock",
			DdlStatement:  "CREATE TABLE mock_table (id INTEGER, PRIMARY KEY(id))",
			TableLimits: nosqlv1beta1.TableLimits{
				MaxReadUnits:    10,
				MaxWriteUnits:   10,
				MaxStorageInGBs: 1,
				CapacityMode:    "PROVISIONED",
			},
			FreeformTags: map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newTableMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://nosql.mock.invalid", BasePath: "20190828", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Table OCI mock: %v", err)
		}
	})

	client := newTableServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		nosqlsdk.NosqlClient{BaseClient: session.BaseClient()},
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*nosqlv1beta1.Table]{
		Resource: resource,
		Client:   client,
		ValidateCreated: func(current *nosqlv1beta1.Table) error {
			if current.Status.Id != mockTableID ||
				current.Status.Name != "mock_table" ||
				current.Status.TableLimits.MaxReadUnits != 10 ||
				current.Status.LifecycleState != string(nosqlsdk.TableLifecycleStateActive) {
				return fmt.Errorf("created Table status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *nosqlv1beta1.Table) {
			current.Spec.TableLimits.MaxReadUnits = 20
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *nosqlv1beta1.Table) error {
			if current.Status.TableLimits.MaxReadUnits != 20 ||
				current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.LifecycleState != string(nosqlsdk.TableLifecycleStateActive) {
				return fmt.Errorf("updated Table status = %+v", current.Status)
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

type tablePathAliasResponder struct {
	delegate *ocimock.CRUDResponder[nosqlsdk.Table]
	alias    string
	itemPath string
}

func (r tablePathAliasResponder) Respond(request ocimock.Request) (ocimock.Response, error) {
	if request.URL != nil && request.URL.Path == r.alias {
		clonedURL := new(url.URL)
		*clonedURL = *request.URL
		clonedURL.Path = r.itemPath
		request.URL = clonedURL
	}
	return r.delegate.Respond(request)
}

func (r tablePathAliasResponder) Verify() error {
	return r.delegate.Verify()
}

func newTableMockResponder(resource *nosqlv1beta1.Table) (ocimock.Responder, error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createReadObserved := false
	updateReadObserved := false
	deleteReadObserved := false
	itemPath := "/20190828/tables/" + mockTableID
	delegate, err := ocimock.NewCRUDResponder(ocimock.CRUDOptions[nosqlsdk.Table]{
		CollectionPath:         "/20190828/tables",
		ItemPath:               itemPath,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		List: func(request ocimock.Request, present bool, state nosqlsdk.Table) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("compartmentId") != resource.Spec.CompartmentId ||
				query.Get("name") != resource.Spec.Name ||
				query.Get("lifecycleState") != "ALL" {
				return ocimock.Response{}, fmt.Errorf("unexpected ListTables query: %s", request.URL.RawQuery)
			}
			items := []nosqlsdk.Table{}
			if present {
				items = append(items, state)
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": items})
		},
		Create: func(request ocimock.Request) (nosqlsdk.Table, ocimock.Response, error) {
			var details nosqlsdk.CreateTableDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return nosqlsdk.Table{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return nosqlsdk.Table{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DdlStatement == nil || *details.DdlStatement != resource.Spec.DdlStatement ||
				details.TableLimits == nil || details.TableLimits.MaxReadUnits == nil ||
				*details.TableLimits.MaxReadUnits != 10 {
				return nosqlsdk.Table{}, ocimock.Response{}, fmt.Errorf("unexpected CreateTable details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return nosqlsdk.Table{}, ocimock.Response{}, fmt.Errorf("CreateTable opc-retry-token is empty")
			}
			state := nosqlsdk.Table{
				Id:                common.String(mockTableID),
				CompartmentId:     details.CompartmentId,
				Name:              details.Name,
				TimeCreated:       &createdAt,
				TimeUpdated:       &createdAt,
				TableLimits:       details.TableLimits,
				LifecycleState:    nosqlsdk.TableLifecycleStateCreating,
				IsAutoReclaimable: details.IsAutoReclaimable,
				DdlStatement:      details.DdlStatement,
				FreeformTags:      details.FreeformTags,
			}
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-table-create"}}
			return state, response, nil
		},
		ReadTransition: func(_ ocimock.Request, state nosqlsdk.Table) (nosqlsdk.Table, ocimock.Response, error) {
			switch state.LifecycleState {
			case nosqlsdk.TableLifecycleStateCreating:
				if createReadObserved {
					state.LifecycleState = nosqlsdk.TableLifecycleStateActive
				} else {
					createReadObserved = true
				}
			case nosqlsdk.TableLifecycleStateUpdating:
				if updateReadObserved {
					state.LifecycleState = nosqlsdk.TableLifecycleStateActive
				} else {
					updateReadObserved = true
				}
			case nosqlsdk.TableLifecycleStateDeleting:
				if deleteReadObserved {
					response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
					return state, response, err
				}
				deleteReadObserved = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state nosqlsdk.Table) (nosqlsdk.Table, ocimock.Response, error) {
			var details nosqlsdk.UpdateTableDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return nosqlsdk.Table{}, ocimock.Response{}, err
			}
			switch {
			case details.TableLimits != nil:
				if details.TableLimits.MaxReadUnits == nil || *details.TableLimits.MaxReadUnits != 20 ||
					details.FreeformTags != nil {
					return nosqlsdk.Table{}, ocimock.Response{}, fmt.Errorf("unexpected table-limit update: %+v", details)
				}
				state.TableLimits = details.TableLimits
			case details.FreeformTags != nil:
				if details.FreeformTags["osok-mock"] != "update" {
					return nosqlsdk.Table{}, ocimock.Response{}, fmt.Errorf("unexpected tag update: %+v", details)
				}
				state.FreeformTags = details.FreeformTags
			default:
				return nosqlsdk.Table{}, ocimock.Response{}, fmt.Errorf("empty UpdateTable details")
			}
			state.LifecycleState = nosqlsdk.TableLifecycleStateUpdating
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-table-update"}}
			return state, response, nil
		},
		DeleteTransition: func(request ocimock.Request, state nosqlsdk.Table) (nosqlsdk.Table, ocimock.Response, error) {
			if request.URL.Query().Get("isIfExists") != "true" {
				return nosqlsdk.Table{}, ocimock.Response{}, fmt.Errorf("DeleteTable omitted isIfExists=true")
			}
			state.LifecycleState = nosqlsdk.TableLifecycleStateDeleting
			response := ocimock.EmptyResponse(http.StatusAccepted)
			response.Header = http.Header{"Opc-Work-Request-Id": []string{"ocid1.workrequest.oc1..mock-table-delete"}}
			return state, response, nil
		},
	})
	if err != nil {
		return nil, err
	}
	return tablePathAliasResponder{
		delegate: delegate,
		alias:    "/20190828/tables/" + resource.Spec.Name,
		itemPath: itemPath,
	}, nil
}
