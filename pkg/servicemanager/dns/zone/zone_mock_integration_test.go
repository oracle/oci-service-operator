/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package zone

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockZoneID = "ocid1.dns-zone.oc1..mock"

// Contract evidence: recorded private-zone CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationZoneLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &dnsv1beta1.Zone{ObjectMeta: metav1.ObjectMeta{Name: "mock-zone", Namespace: "default", UID: types.UID("mock-zone-uid")}, Spec: dnsv1beta1.ZoneSpec{
		Name: "mock.internal", CompartmentId: "ocid1.compartment.oc1..mock", ViewId: "ocid1.dnsview.oc1..mock", ZoneType: string(dnssdk.CreateZoneDetailsZoneTypePrimary), Scope: string(dnssdk.ScopePrivate), FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newZoneMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dns.mock.invalid", BasePath: "20180115", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Zone OCI mock: %v", err)
		}
	})
	client := newZoneServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, dnssdk.DnsClient{BaseClient: session.BaseClient()}, nil)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dnsv1beta1.Zone]{
		Resource: resource, Client: client,
		ValidateCreated: func(current *dnsv1beta1.Zone) error {
			if current.Status.Id != mockZoneID || current.Status.Name != resource.Spec.Name || current.Status.Scope != string(dnssdk.ScopePrivate) || current.Status.LifecycleState != string(dnssdk.ZoneLifecycleStateActive) {
				return fmt.Errorf("created Zone status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dnsv1beta1.Zone) { current.Spec.FreeformTags = map[string]string{"osok-mock": "update"} },
		ValidateUpdated: func(current *dnsv1beta1.Zone) error {
			if current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Zone status = %+v", current.Status)
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

func newZoneMockResponder(resource *dnsv1beta1.Zone) (*ocimock.CRUDResponder[dnssdk.Zone], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	updateRead := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[dnssdk.Zone]{
		CollectionPath: "/20180115/zones", ItemPath: "/20180115/zones/" + mockZoneID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(request ocimock.Request, present bool, state dnssdk.Zone) (ocimock.Response, error) {
			query := request.URL.Query()
			if query.Get("name") != resource.Spec.Name || query.Get("scope") != resource.Spec.Scope || query.Get("viewId") != resource.Spec.ViewId {
				return ocimock.Response{}, fmt.Errorf("unexpected ListZones query: %s", request.URL.RawQuery)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []dnssdk.Zone{})
			}
			return ocimock.JSONResponse(http.StatusOK, []dnssdk.Zone{state})
		},
		Create: func(request ocimock.Request) (dnssdk.Zone, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero dnssdk.Zone
				return zero, ocimock.Response{}, err
			}
			var envelope struct {
				dnssdk.CreateZoneDetails
				MigrationSource string `json:"migrationSource"`
			}
			if err := ocimock.DecodeJSONRequest(request, &envelope); err != nil {
				return dnssdk.Zone{}, ocimock.Response{}, err
			}
			details := envelope.CreateZoneDetails
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return dnssdk.Zone{}, ocimock.Response{}, err
			}
			if envelope.MigrationSource != "NONE" || details.Name == nil || *details.Name != resource.Spec.Name || details.ViewId == nil || *details.ViewId != resource.Spec.ViewId || details.Scope != dnssdk.ScopePrivate || details.ZoneType != dnssdk.CreateZoneDetailsZoneTypePrimary {
				return dnssdk.Zone{}, ocimock.Response{}, fmt.Errorf("unexpected CreateZone details: %+v", envelope)
			}
			state := dnssdk.Zone{Name: details.Name, ZoneType: dnssdk.ZoneZoneTypePrimary, CompartmentId: details.CompartmentId, Scope: details.Scope,
				FreeformTags: details.FreeformTags, DefinedTags: map[string]map[string]interface{}{}, ResolutionMode: dnssdk.ZoneResolutionModeStatic,
				DnssecState: dnssdk.ZoneDnssecStateDisabled, ExternalMasters: []dnssdk.ExternalMaster{}, ExternalDownstreams: []dnssdk.ExternalDownstream{},
				Self: common.String("https://dns.mock.invalid/zones/" + mockZoneID), Id: common.String(mockZoneID), TimeCreated: &now, Version: common.String("1"), Serial: common.Int64(1),
				LifecycleState: dnssdk.ZoneLifecycleStateActive, IsProtected: common.Bool(false), Nameservers: []dnssdk.Nameserver{}, ViewId: details.ViewId}
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state dnssdk.Zone) (dnssdk.Zone, ocimock.Response, error) {
			if state.LifecycleState == dnssdk.ZoneLifecycleStateUpdating && updateRead {
				state.LifecycleState = dnssdk.ZoneLifecycleStateActive
			} else if state.LifecycleState == dnssdk.ZoneLifecycleStateUpdating {
				updateRead = true
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state dnssdk.Zone) (dnssdk.Zone, ocimock.Response, error) {
			var details dnssdk.UpdateZoneDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return dnssdk.Zone{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" {
				return dnssdk.Zone{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateZone details: %+v", details)
			}
			state.FreeformTags, state.LifecycleState = details.FreeformTags, dnssdk.ZoneLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ dnssdk.Zone) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
