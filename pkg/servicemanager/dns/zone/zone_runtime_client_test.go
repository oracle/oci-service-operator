/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package zone

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	dnssdk "github.com/oracle/oci-go-sdk/v65/dns"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type zoneCreateResult struct {
	response dnssdk.CreateZoneResponse
	err      error
}

type zoneGetResult struct {
	response dnssdk.GetZoneResponse
	err      error
}

type zoneListResult struct {
	response dnssdk.ListZonesResponse
	err      error
}

type zoneUpdateResult struct {
	response dnssdk.UpdateZoneResponse
	err      error
}

type zoneDeleteResult struct {
	response dnssdk.DeleteZoneResponse
	err      error
}

type fakeZoneOCIClient struct {
	createRequests []dnssdk.CreateZoneRequest
	getRequests    []dnssdk.GetZoneRequest
	listRequests   []dnssdk.ListZonesRequest
	updateRequests []dnssdk.UpdateZoneRequest
	deleteRequests []dnssdk.DeleteZoneRequest

	createResults []zoneCreateResult
	getResults    []zoneGetResult
	listResults   []zoneListResult
	updateResults []zoneUpdateResult
	deleteResults []zoneDeleteResult
}

func (f *fakeZoneOCIClient) CreateZone(_ context.Context, request dnssdk.CreateZoneRequest) (dnssdk.CreateZoneResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if len(f.createResults) == 0 {
		return dnssdk.CreateZoneResponse{}, fmt.Errorf("unexpected CreateZone call")
	}
	next := f.createResults[0]
	f.createResults = f.createResults[1:]
	return next.response, next.err
}

func (f *fakeZoneOCIClient) GetZone(_ context.Context, request dnssdk.GetZoneRequest) (dnssdk.GetZoneResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		return dnssdk.GetZoneResponse{}, fmt.Errorf("unexpected GetZone call")
	}
	next := f.getResults[0]
	f.getResults = f.getResults[1:]
	return next.response, next.err
}

func (f *fakeZoneOCIClient) ListZones(_ context.Context, request dnssdk.ListZonesRequest) (dnssdk.ListZonesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		return dnssdk.ListZonesResponse{}, fmt.Errorf("unexpected ListZones call")
	}
	next := f.listResults[0]
	f.listResults = f.listResults[1:]
	return next.response, next.err
}

func (f *fakeZoneOCIClient) UpdateZone(_ context.Context, request dnssdk.UpdateZoneRequest) (dnssdk.UpdateZoneResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if len(f.updateResults) == 0 {
		return dnssdk.UpdateZoneResponse{}, fmt.Errorf("unexpected UpdateZone call")
	}
	next := f.updateResults[0]
	f.updateResults = f.updateResults[1:]
	return next.response, next.err
}

func (f *fakeZoneOCIClient) DeleteZone(_ context.Context, request dnssdk.DeleteZoneRequest) (dnssdk.DeleteZoneResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if len(f.deleteResults) == 0 {
		return dnssdk.DeleteZoneResponse{}, fmt.Errorf("unexpected DeleteZone call")
	}
	next := f.deleteResults[0]
	f.deleteResults = f.deleteResults[1:]
	return next.response, next.err
}

func TestZoneServiceClientCreatesWithReviewedBodyAndTracksRequestID(t *testing.T) {
	resource := testZoneResource()
	resource.Spec.FreeformTags = map[string]string{"owner": "dns"}
	resource.Spec.ExternalMasters = []dnsv1beta1.ZoneExternalMaster{{Address: "192.0.2.10", Port: 53}}

	fake := &fakeZoneOCIClient{
		listResults: []zoneListResult{{response: dnssdk.ListZonesResponse{}}},
		createResults: []zoneCreateResult{{
			response: dnssdk.CreateZoneResponse{
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
				Zone:             testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateCreating),
			},
		}},
		getResults: []zoneGetResult{{
			response: dnssdk.GetZoneResponse{
				Zone: testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive),
			},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateZone calls = %d, want 1", len(fake.createRequests))
	}
	body, ok := fake.createRequests[0].CreateZoneDetails.(dnssdk.CreateZoneDetails)
	if !ok {
		t.Fatalf("CreateZoneDetails = %T, want dnssdk.CreateZoneDetails", fake.createRequests[0].CreateZoneDetails)
	}
	assertReviewedZoneCreateBody(t, body, resource)
	if fake.createRequests[0].OpcRetryToken == nil {
		t.Fatalf("create request OpcRetryToken = nil, want deterministic retry token")
	}
	assertCreatedZoneStatus(t, resource)
	assertLatestCondition(t, resource, shared.Active)
}

func TestZoneServiceClientCreatesWithDynectSpecAndRecordsFingerprint(t *testing.T) {
	runZoneCreateWithDynectInputsAndRecordsFingerprint(t, func(resource *dnsv1beta1.Zone) {
		configureZoneDynectSpec(resource, "customer", "user", "password")
	})
}

func TestZoneServiceClientCreatesWithDynectJSONDataAndRecordsFingerprint(t *testing.T) {
	runZoneCreateWithDynectInputsAndRecordsFingerprint(t, func(resource *dnsv1beta1.Zone) {
		configureZoneDynectJSONData(resource, "customer", "user", "password")
	})
}

func runZoneCreateWithDynectInputsAndRecordsFingerprint(t *testing.T, configure func(*dnsv1beta1.Zone)) {
	t.Helper()
	resource := testZoneResource()
	configure(resource)
	wantFingerprint, ok, err := zoneDynectCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		t.Fatalf("zoneDynectCreateOnlyFingerprint() error = %v", err)
	}
	if !ok {
		t.Fatal("zoneDynectCreateOnlyFingerprint() ok = false, want true")
	}
	fake := &fakeZoneOCIClient{
		listResults: []zoneListResult{{response: dnssdk.ListZonesResponse{}}},
		createResults: []zoneCreateResult{{
			response: dnssdk.CreateZoneResponse{
				OpcRequestId: common.String("opc-create"),
				Zone:         testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive),
			},
		}},
		getResults: []zoneGetResult{{
			response: dnssdk.GetZoneResponse{
				Zone: testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive),
			},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateZone calls = %d, want 1", len(fake.createRequests))
	}
	assertZoneAppliedDynectFingerprint(t, resource, wantFingerprint)
	assertLatestCondition(t, resource, shared.Active)
}

func TestZoneServiceClientPreservesCreateWorkRequestIDWhileLifecyclePending(t *testing.T) {
	resource := testZoneResource()

	fake := &fakeZoneOCIClient{
		listResults: []zoneListResult{{response: dnssdk.ListZonesResponse{}}},
		createResults: []zoneCreateResult{{
			response: dnssdk.CreateZoneResponse{
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
				Zone:             testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateCreating),
			},
		}},
		getResults: []zoneGetResult{{
			response: dnssdk.GetZoneResponse{
				Zone: testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateCreating),
			},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	assertCurrentWorkRequestID(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create")
	assertLatestCondition(t, resource, shared.Provisioning)
}

func TestZoneServiceClientBindsExistingAcrossListPages(t *testing.T) {
	resource := testZoneResource()
	fake := &fakeZoneOCIClient{
		listResults: []zoneListResult{
			{response: dnssdk.ListZonesResponse{OpcNextPage: common.String("page-2")}},
			{response: dnssdk.ListZonesResponse{Items: []dnssdk.ZoneSummary{testZoneSummary("ocid1.zone.oc1..existing", resource.Spec.Name, dnssdk.ZoneSummaryLifecycleStateActive)}}},
		},
		getResults: []zoneGetResult{{
			response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..existing", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateZone calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListZones calls = %d, want 2", len(fake.listRequests))
	}
	if got := stringPtrValue(fake.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second ListZones page = %q, want page-2", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.zone.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing OCID", got)
	}
	assertLatestCondition(t, resource, shared.Active)
}

func TestZoneServiceClientNoopsWithReadableConfiguredCreateOnlyInputs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tracked   bool
		configure func(*dnsv1beta1.Zone)
	}{
		{
			name:    "tracked jsonData NONE",
			tracked: true,
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.JsonData = `{
					"migrationSource":"NONE",
					"name":"example.com",
					"compartmentId":"ocid1.compartment.oc1..test",
					"zoneType":"PRIMARY",
					"scope":"GLOBAL",
					"resolutionMode":"STATIC",
					"dnssecState":"DISABLED"
				}`
			},
		},
		{
			name:    "bound jsonData NONE",
			tracked: false,
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.JsonData = `{
					"migrationSource":"NONE",
					"name":"example.com",
					"compartmentId":"ocid1.compartment.oc1..test",
					"zoneType":"PRIMARY",
					"scope":"GLOBAL",
					"resolutionMode":"STATIC",
					"dnssecState":"DISABLED"
				}`
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testZoneResource()
			if tc.tracked {
				resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
			}
			tc.configure(resource)
			current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)

			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
			}
			if !tc.tracked {
				fake.listResults = []zoneListResult{{
					response: dnssdk.ListZonesResponse{
						Items: []dnssdk.ZoneSummary{testZoneSummary("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneSummaryLifecycleStateActive)},
					},
				}}
			}

			response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			assertZoneNoopResponse(t, response)
			assertNoZoneMutations(t, fake)
			if !tc.tracked && len(fake.listRequests) != 1 {
				t.Fatalf("ListZones calls = %d, want 1", len(fake.listRequests))
			}
			assertLatestCondition(t, resource, shared.Active)
		})
	}
}

func TestZoneServiceClientNoopsWithAppliedDynectCreateOnlyInputs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*dnsv1beta1.Zone)
	}{
		{
			name: "tracked migrationSource DYNECT",
			configure: func(resource *dnsv1beta1.Zone) {
				configureZoneDynectSpec(resource, "customer", "user", "password")
			},
		},
		{
			name: "tracked jsonData DYNECT",
			configure: func(resource *dnsv1beta1.Zone) {
				configureZoneDynectJSONData(resource, "customer", "user", "password")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testTrackedZoneResourceWithStatus()
			tc.configure(resource)
			wantFingerprint := seedZoneAppliedDynectFingerprint(t, resource)
			current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)

			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
			}

			response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			assertZoneNoopResponse(t, response)
			assertNoZoneCreateUpdateOrList(t, fake)
			assertZoneAppliedDynectFingerprint(t, resource, wantFingerprint)
			assertLatestCondition(t, resource, shared.Active)
		})
	}
}

func TestZoneServiceClientBindsExistingWithDynectInputsAndRecordsFingerprint(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*dnsv1beta1.Zone)
	}{
		{
			name: "migrationSource DYNECT",
			configure: func(resource *dnsv1beta1.Zone) {
				configureZoneDynectSpec(resource, "customer", "user", "password")
			},
		},
		{
			name: "jsonData DYNECT",
			configure: func(resource *dnsv1beta1.Zone) {
				configureZoneDynectJSONData(resource, "customer", "user", "password")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testZoneResource()
			tc.configure(resource)
			wantFingerprint, ok, err := zoneDynectCreateOnlyFingerprint(resource.Spec)
			if err != nil {
				t.Fatalf("zoneDynectCreateOnlyFingerprint() error = %v", err)
			}
			if !ok {
				t.Fatal("zoneDynectCreateOnlyFingerprint() ok = false, want true")
			}
			fake := &fakeZoneOCIClient{
				listResults: []zoneListResult{{
					response: dnssdk.ListZonesResponse{
						Items: []dnssdk.ZoneSummary{
							testZoneSummary("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneSummaryLifecycleStateActive),
						},
					},
				}},
				getResults: []zoneGetResult{{
					response: dnssdk.GetZoneResponse{
						Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive),
					},
				}},
			}

			response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			assertZoneNoopResponse(t, response)
			assertNoZoneMutations(t, fake)
			if len(fake.listRequests) != 1 {
				t.Fatalf("ListZones calls = %d, want 1", len(fake.listRequests))
			}
			assertZoneAppliedDynectFingerprint(t, resource, wantFingerprint)
			assertLatestCondition(t, resource, shared.Active)
		})
	}
}

func TestZoneServiceClientNoopsWhenExternalTransferPortsDefaultTo53(t *testing.T) {
	for _, tc := range []struct {
		name             string
		configureSpec    func(*dnsv1beta1.Zone)
		configureCurrent func(*dnssdk.Zone)
	}{
		{
			name: "external masters",
			configureSpec: func(resource *dnsv1beta1.Zone) {
				resource.Spec.ExternalMasters = []dnsv1beta1.ZoneExternalMaster{{
					Address:   "192.0.2.10",
					TsigKeyId: "ocid1.tsigkey.oc1..test",
				}}
			},
			configureCurrent: func(current *dnssdk.Zone) {
				current.ExternalMasters = []dnssdk.ExternalMaster{{
					Address:   common.String("192.0.2.10"),
					Port:      common.Int(53),
					TsigKeyId: common.String("ocid1.tsigkey.oc1..test"),
				}}
			},
		},
		{
			name: "external downstreams",
			configureSpec: func(resource *dnsv1beta1.Zone) {
				resource.Spec.ExternalDownstreams = []dnsv1beta1.ZoneExternalDownstream{{
					Address:   "192.0.2.20",
					TsigKeyId: "ocid1.tsigkey.oc1..test",
				}}
			},
			configureCurrent: func(current *dnssdk.Zone) {
				current.ExternalDownstreams = []dnssdk.ExternalDownstream{{
					Address:   common.String("192.0.2.20"),
					Port:      common.Int(53),
					TsigKeyId: common.String("ocid1.tsigkey.oc1..test"),
				}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testZoneResource()
			resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
			tc.configureSpec(resource)
			current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)
			tc.configureCurrent(&current)

			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
			}

			response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			assertZoneNoopResponse(t, response)
			assertNoZoneMutations(t, fake)
			assertLatestCondition(t, resource, shared.Active)
		})
	}
}

func assertZoneNoopResponse(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = true, want false")
	}
}

func assertNoZoneMutations(t *testing.T, fake *fakeZoneOCIClient) {
	t.Helper()
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateZone calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateZone calls = %d, want 0", len(fake.updateRequests))
	}
}

func assertNoZoneCreateUpdateOrList(t *testing.T, fake *fakeZoneOCIClient) {
	t.Helper()
	assertNoZoneMutations(t, fake)
	if len(fake.listRequests) != 0 {
		t.Fatalf("ListZones calls = %d, want 0", len(fake.listRequests))
	}
}

func TestZoneServiceClientUpdatesMutableFields(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
	resource.Spec.ResolutionMode = string(dnssdk.ZoneResolutionModeTransparent)
	resource.Spec.FreeformTags = map[string]string{"owner": "updated"}

	current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)
	current.ResolutionMode = dnssdk.ZoneResolutionModeStatic
	current.FreeformTags = map[string]string{"owner": "old"}
	updated := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)
	updated.ResolutionMode = dnssdk.ZoneResolutionModeTransparent
	updated.FreeformTags = map[string]string{"owner": "updated"}

	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{
			{response: dnssdk.GetZoneResponse{Zone: current}},
			{response: dnssdk.GetZoneResponse{Zone: updated}},
		},
		updateResults: []zoneUpdateResult{{
			response: dnssdk.UpdateZoneResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
				Zone:             updated,
			},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateZone calls = %d, want 1", len(fake.updateRequests))
	}
	details := fake.updateRequests[0].UpdateZoneDetails
	if details.ResolutionMode != dnssdk.ZoneResolutionModeTransparent {
		t.Fatalf("update body resolutionMode = %q, want TRANSPARENT", details.ResolutionMode)
	}
	if got := details.FreeformTags["owner"]; got != "updated" {
		t.Fatalf("update body freeformTags[owner] = %q, want updated", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", got)
	}
	assertLatestCondition(t, resource, shared.Active)
}

func TestZoneServiceClientPreservesCurrentCollectionsDuringResolutionModeUpdate(t *testing.T) {
	resource := testTrackedZoneResource()
	resource.Spec.ResolutionMode = string(dnssdk.ZoneResolutionModeTransparent)

	current := testZoneWithCurrentCollections(resource)
	updated := current
	updated.ResolutionMode = dnssdk.ZoneResolutionModeTransparent
	fake := newZoneScalarUpdateFake(current, updated)

	details := runZoneScalarUpdate(t, resource, fake)
	if details.ResolutionMode != dnssdk.ZoneResolutionModeTransparent {
		t.Fatalf("update body resolutionMode = %q, want %q", details.ResolutionMode, dnssdk.ZoneResolutionModeTransparent)
	}
	if details.DnssecState != "" {
		t.Fatalf("update body dnssecState = %q, want empty", details.DnssecState)
	}
	assertZoneUpdatePreservedCurrentCollections(t, details, current)
}

func TestZoneServiceClientPreservesCurrentCollectionsDuringDnssecStateUpdate(t *testing.T) {
	resource := testTrackedZoneResource()
	resource.Spec.DnssecState = string(dnssdk.ZoneDnssecStateEnabled)

	current := testZoneWithCurrentCollections(resource)
	updated := current
	updated.DnssecState = dnssdk.ZoneDnssecStateEnabled
	fake := newZoneScalarUpdateFake(current, updated)

	details := runZoneScalarUpdate(t, resource, fake)
	if details.ResolutionMode != "" {
		t.Fatalf("update body resolutionMode = %q, want empty", details.ResolutionMode)
	}
	if details.DnssecState != dnssdk.ZoneDnssecStateEnabled {
		t.Fatalf("update body dnssecState = %q, want %q", details.DnssecState, dnssdk.ZoneDnssecStateEnabled)
	}
	assertZoneUpdatePreservedCurrentCollections(t, details, current)
}

func testTrackedZoneResource() *dnsv1beta1.Zone {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
	return resource
}

func testTrackedZoneResourceWithStatus() *dnsv1beta1.Zone {
	resource := testTrackedZoneResource()
	resource.Status.Id = "ocid1.zone.oc1..tracked"
	resource.Status.Name = resource.Spec.Name
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.ZoneType = resource.Spec.ZoneType
	resource.Status.Scope = resource.Spec.Scope
	resource.Status.ResolutionMode = resource.Spec.ResolutionMode
	resource.Status.DnssecState = resource.Spec.DnssecState
	resource.Status.FreeformTags = map[string]string{}
	resource.Status.DefinedTags = map[string]shared.MapValue{}
	resource.Status.ExternalMasters = []dnsv1beta1.ZoneExternalMaster{}
	resource.Status.ExternalDownstreams = []dnsv1beta1.ZoneExternalDownstream{}
	return resource
}

func testZoneWithCurrentCollections(resource *dnsv1beta1.Zone) dnssdk.Zone {
	current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)
	current.FreeformTags = map[string]string{"owner": "current"}
	current.DefinedTags = map[string]map[string]interface{}{
		"Operations": {"CostCenter": "42"},
	}
	current.ExternalMasters = []dnssdk.ExternalMaster{{
		Address:   common.String("192.0.2.10"),
		Port:      common.Int(53),
		TsigKeyId: common.String("ocid1.tsigkey.oc1..master"),
	}}
	current.ExternalDownstreams = []dnssdk.ExternalDownstream{{
		Address:   common.String("192.0.2.20"),
		Port:      common.Int(5300),
		TsigKeyId: common.String("ocid1.tsigkey.oc1..downstream"),
	}}
	return current
}

func newZoneScalarUpdateFake(current, updated dnssdk.Zone) *fakeZoneOCIClient {
	return &fakeZoneOCIClient{
		getResults: []zoneGetResult{
			{response: dnssdk.GetZoneResponse{Zone: current}},
			{response: dnssdk.GetZoneResponse{Zone: updated}},
		},
		updateResults: []zoneUpdateResult{{
			response: dnssdk.UpdateZoneResponse{
				OpcRequestId: common.String("opc-update"),
				Zone:         updated,
			},
		}},
	}
}

func runZoneScalarUpdate(
	t *testing.T,
	resource *dnsv1beta1.Zone,
	fake *fakeZoneOCIClient,
) dnssdk.UpdateZoneDetails {
	t.Helper()
	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateZone calls = %d, want 1", len(fake.updateRequests))
	}
	assertLatestCondition(t, resource, shared.Active)
	return fake.updateRequests[0].UpdateZoneDetails
}

func assertZoneUpdatePreservedCurrentCollections(
	t *testing.T,
	details dnssdk.UpdateZoneDetails,
	current dnssdk.Zone,
) {
	t.Helper()
	if !reflect.DeepEqual(details.FreeformTags, current.FreeformTags) {
		t.Fatalf("update body freeformTags = %#v, want current %#v", details.FreeformTags, current.FreeformTags)
	}
	if !reflect.DeepEqual(details.DefinedTags, current.DefinedTags) {
		t.Fatalf("update body definedTags = %#v, want current %#v", details.DefinedTags, current.DefinedTags)
	}
	if !reflect.DeepEqual(details.ExternalMasters, current.ExternalMasters) {
		t.Fatalf("update body externalMasters = %#v, want current %#v", details.ExternalMasters, current.ExternalMasters)
	}
	if !reflect.DeepEqual(details.ExternalDownstreams, current.ExternalDownstreams) {
		t.Fatalf("update body externalDownstreams = %#v, want current %#v", details.ExternalDownstreams, current.ExternalDownstreams)
	}
	assertZoneUpdateCollectionsDoNotEncodeNull(t, details)
}

func assertZoneUpdateCollectionsDoNotEncodeNull(t *testing.T, details dnssdk.UpdateZoneDetails) {
	t.Helper()
	encoded, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal update body: %v", err)
	}
	for _, disallowed := range []string{
		`"freeformTags":null`,
		`"definedTags":null`,
		`"externalMasters":null`,
		`"externalDownstreams":null`,
	} {
		if strings.Contains(string(encoded), disallowed) {
			t.Fatalf("update body encoded as %s, want current collection value instead of %s", encoded, disallowed)
		}
	}
}

func TestZoneServiceClientPreservesUpdateWorkRequestIDWhileLifecyclePending(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
	resource.Spec.ResolutionMode = string(dnssdk.ZoneResolutionModeTransparent)

	current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)
	current.ResolutionMode = dnssdk.ZoneResolutionModeStatic
	updated := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateUpdating)
	updated.ResolutionMode = dnssdk.ZoneResolutionModeTransparent

	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{
			{response: dnssdk.GetZoneResponse{Zone: current}},
			{response: dnssdk.GetZoneResponse{Zone: updated}},
		},
		updateResults: []zoneUpdateResult{{
			response: dnssdk.UpdateZoneResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
				Zone:             updated,
			},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateZone calls = %d, want 1", len(fake.updateRequests))
	}
	assertCurrentWorkRequestID(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update")
	assertLatestCondition(t, resource, shared.Updating)
}

func TestZoneServiceClientSkipsUpdateWhileLifecycleBlocksMutation(t *testing.T) {
	for _, tc := range []struct {
		name          string
		state         dnssdk.ZoneLifecycleStateEnum
		wantCondition shared.OSOKConditionType
		wantRequeue   bool
		wantSuccess   bool
	}{
		{
			name:          "creating",
			state:         dnssdk.ZoneLifecycleStateCreating,
			wantCondition: shared.Provisioning,
			wantRequeue:   true,
			wantSuccess:   true,
		},
		{
			name:          "updating",
			state:         dnssdk.ZoneLifecycleStateUpdating,
			wantCondition: shared.Updating,
			wantRequeue:   true,
			wantSuccess:   true,
		},
		{
			name:          "deleting",
			state:         dnssdk.ZoneLifecycleStateDeleting,
			wantCondition: shared.Terminating,
			wantRequeue:   true,
			wantSuccess:   true,
		},
		{
			name:          "failed",
			state:         dnssdk.ZoneLifecycleStateFailed,
			wantCondition: shared.Failed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testZoneResource()
			resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
			resource.Spec.ResolutionMode = string(dnssdk.ZoneResolutionModeTransparent)

			current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, tc.state)
			current.ResolutionMode = dnssdk.ZoneResolutionModeStatic
			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
			}

			response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccess {
				t.Fatalf("CreateOrUpdate() successful = %t, want %t", response.IsSuccessful, tc.wantSuccess)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if len(fake.updateRequests) != 0 {
				t.Fatalf("UpdateZone calls = %d, want 0", len(fake.updateRequests))
			}
			assertLatestCondition(t, resource, tc.wantCondition)
		})
	}
}

func TestZoneServiceClientTreatsDeletedReadbackAsTerminalDelete(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
	resource.Spec.ResolutionMode = string(dnssdk.ZoneResolutionModeTransparent)

	current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateDeleted)
	current.ResolutionMode = dnssdk.ZoneResolutionModeStatic
	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateZone calls = %d, want 0 for DELETED readback", len(fake.updateRequests))
	}
	assertCurrentLifecycleAsync(
		t,
		resource,
		shared.OSOKAsyncPhaseDelete,
		string(dnssdk.ZoneLifecycleStateDeleted),
		shared.OSOKAsyncClassSucceeded,
	)
	assertLatestCondition(t, resource, shared.Terminating)
}

func TestZoneServiceClientRejectsImmutableNameDriftBeforeUpdate(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
	resource.Spec.Name = "renamed.example.com"
	current := testZoneSDK("ocid1.zone.oc1..tracked", "example.com", dnssdk.ZoneLifecycleStateActive)

	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
	}

	_, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "require replacement when name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want name replacement rejection", err)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateZone calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestZoneServiceClientRejectsTrackedReadMissWithImmutableLookupDrift(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*dnsv1beta1.Zone)
		wantField string
	}{
		{
			name: "compartmentId changes",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
			},
			wantField: "compartmentId",
		},
		{
			name: "scope changes",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.Scope = string(dnssdk.ScopePrivate)
				resource.Spec.ViewId = "ocid1.dnsview.oc1..replacement"
			},
			wantField: "scope",
		},
		{
			name: "private viewId changes",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.Scope = string(dnssdk.ScopePrivate)
				resource.Spec.ViewId = "ocid1.dnsview.oc1..replacement"
				resource.Status.Scope = string(dnssdk.ScopePrivate)
				resource.Status.ViewId = "ocid1.dnsview.oc1..tracked"
			},
			wantField: "viewId",
		},
		{
			name: "private viewId omitted",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.Scope = string(dnssdk.ScopePrivate)
				resource.Spec.ViewId = ""
				resource.Status.Scope = string(dnssdk.ScopePrivate)
				resource.Status.ViewId = "ocid1.dnsview.oc1..tracked"
			},
			wantField: "viewId",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testTrackedZoneResourceWithStatus()
			tc.configure(resource)
			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{
					err: errortest.NewServiceError(404, errorutil.NotFound, "zone not found"),
				}},
				listResults: []zoneListResult{{response: dnssdk.ListZonesResponse{}}},
				createResults: []zoneCreateResult{{
					response: dnssdk.CreateZoneResponse{
						Zone: testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive),
					},
				}},
			}

			_, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			if err == nil || !strings.Contains(err.Error(), "require replacement") || !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want %s replacement rejection", err, tc.wantField)
			}
			if len(fake.listRequests) != 0 {
				t.Fatalf("ListZones calls = %d, want 0 for tracked read miss", len(fake.listRequests))
			}
			if len(fake.createRequests) != 0 {
				t.Fatalf("CreateZone calls = %d, want 0 for tracked read miss", len(fake.createRequests))
			}
		})
	}
}

func TestZoneServiceClientRejectsTrackedReadMissWithoutStatusDriftConservatively(t *testing.T) {
	resource := testTrackedZoneResourceWithStatus()
	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotFound, "zone not found"),
		}},
		listResults: []zoneListResult{{response: dnssdk.ListZonesResponse{}}},
		createResults: []zoneCreateResult{{
			response: dnssdk.CreateZoneResponse{
				Zone: testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive),
			},
		}},
	}

	_, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "refusing list/create fallback") {
		t.Fatalf("CreateOrUpdate() error = %v, want conservative tracked read miss error", err)
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("ListZones calls = %d, want 0 for tracked read miss", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateZone calls = %d, want 0 for tracked read miss", len(fake.createRequests))
	}
}

func TestZoneServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mutate    func(*dnsv1beta1.Zone)
		wantField string
	}{
		{
			name: "jsonData",
			mutate: func(resource *dnsv1beta1.Zone) {
				resource.Spec.JsonData = `{"migrationSource":"NONE","name":"renamed.example.com","compartmentId":"ocid1.compartment.oc1..test"}`
			},
			wantField: "jsonData",
		},
		{
			name: "jsonData externalMasters",
			mutate: func(resource *dnsv1beta1.Zone) {
				resource.Spec.JsonData = `{
					"migrationSource":"NONE",
					"name":"example.com",
					"compartmentId":"ocid1.compartment.oc1..test",
					"zoneType":"PRIMARY",
					"scope":"GLOBAL",
					"externalMasters":[{"address":"192.0.2.10"}]
				}`
			},
			wantField: "jsonData",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testZoneResource()
			resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
			tc.mutate(resource)
			current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)

			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
			}

			_, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			if err == nil || !strings.Contains(err.Error(), "require replacement") || !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want %s replacement rejection", err, tc.wantField)
			}
			if len(fake.updateRequests) != 0 {
				t.Fatalf("UpdateZone calls = %d, want 0", len(fake.updateRequests))
			}
		})
	}
}

func TestZoneServiceClientRejectsDynectCreateOnlyDriftBeforeNoopOrUpdate(t *testing.T) {
	for _, tc := range []struct {
		name             string
		configure        func(*dnsv1beta1.Zone)
		configureCurrent func(*dnssdk.Zone)
		wantFields       []string
	}{
		{
			name: "migrationSource DYNECT before noop",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.MigrationSource = string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect)
				resource.Spec.DynectMigrationDetails = dnsv1beta1.ZoneDynectMigrationDetails{
					CustomerName: "changed-customer",
					Username:     "changed-user",
					Password:     "changed-password",
				}
			},
			wantFields: []string{"migrationSource", "dynectMigrationDetails"},
		},
		{
			name: "jsonData DYNECT before noop",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.JsonData = `{
					"migrationSource":"DYNECT",
					"name":"example.com",
					"compartmentId":"ocid1.compartment.oc1..test",
					"dynectMigrationDetails":{
						"customerName":"changed-customer",
						"username":"changed-user",
						"password":"changed-password"
					}
				}`
			},
			wantFields: []string{"jsonData.migrationSource", "jsonData.dynectMigrationDetails"},
		},
		{
			name: "jsonData DYNECT before mutable update",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.FreeformTags = map[string]string{"owner": "updated"}
				resource.Spec.JsonData = `{
					"migrationSource":"DYNECT",
					"name":"example.com",
					"compartmentId":"ocid1.compartment.oc1..test",
					"dynectMigrationDetails":{
						"customerName":"changed-customer",
						"username":"changed-user",
						"password":"changed-password"
					}
				}`
			},
			configureCurrent: func(current *dnssdk.Zone) {
				current.FreeformTags = map[string]string{"owner": "old"}
			},
			wantFields: []string{"jsonData.migrationSource", "jsonData.dynectMigrationDetails"},
		},
		{
			name: "dynectMigrationDetails without discriminator",
			configure: func(resource *dnsv1beta1.Zone) {
				resource.Spec.DynectMigrationDetails = dnsv1beta1.ZoneDynectMigrationDetails{
					CustomerName: "changed-customer",
					Username:     "changed-user",
					Password:     "changed-password",
				}
			},
			wantFields: []string{"dynectMigrationDetails"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testTrackedZoneResourceWithStatus()
			tc.configure(resource)
			current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)
			if tc.configureCurrent != nil {
				tc.configureCurrent(&current)
			}
			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
			}

			response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			assertZoneCreateOnlyDriftRejected(t, response, err, tc.wantFields)
			assertNoZoneCreateUpdateOrList(t, fake)
		})
	}
}

func TestZoneServiceClientRejectsEditedAppliedDynectInputsBeforeNoopOrUpdate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*dnsv1beta1.Zone, string, string, string)
	}{
		{
			name:      "migrationSource DYNECT",
			configure: configureZoneDynectSpec,
		},
		{
			name:      "jsonData DYNECT",
			configure: configureZoneDynectJSONData,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := testTrackedZoneResourceWithStatus()
			tc.configure(resource, "original-customer", "original-user", "original-password")
			seedZoneAppliedDynectFingerprint(t, resource)
			tc.configure(resource, "changed-customer", "changed-user", "changed-password")
			current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)

			fake := &fakeZoneOCIClient{
				getResults: []zoneGetResult{{response: dnssdk.GetZoneResponse{Zone: current}}},
			}

			response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
			assertZoneCreateOnlyDriftRejected(t, response, err, []string{"migrationSource", "dynectMigrationDetails"})
			assertNoZoneCreateUpdateOrList(t, fake)
		})
	}
}

func TestZoneServiceClientRejectsDynectCreateOnlyDriftBeforeTrackedReadMissFallback(t *testing.T) {
	resource := testTrackedZoneResourceWithStatus()
	resource.Spec.MigrationSource = string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect)
	resource.Spec.DynectMigrationDetails = dnsv1beta1.ZoneDynectMigrationDetails{
		CustomerName: "changed-customer",
		Username:     "changed-user",
		Password:     "changed-password",
	}
	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotFound, "zone not found"),
		}},
		listResults: []zoneListResult{{response: dnssdk.ListZonesResponse{}}},
		createResults: []zoneCreateResult{{
			response: dnssdk.CreateZoneResponse{
				Zone: testZoneSDK("ocid1.zone.oc1..created", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive),
			},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	assertZoneCreateOnlyDriftRejected(t, response, err, []string{"migrationSource", "dynectMigrationDetails"})
	assertNoZoneCreateUpdateOrList(t, fake)
}

func assertZoneCreateOnlyDriftRejected(
	t *testing.T,
	response servicemanager.OSOKResponse,
	err error,
	wantFields []string,
) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "require replacement") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only replacement rejection", err)
	}
	for _, want := range wantFields {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("CreateOrUpdate() error = %v, want field %q", err, want)
		}
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = true, want false")
	}
}

func TestZoneLifecycleRequeueUsesNonZeroDuration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		state     dnssdk.ZoneLifecycleStateEnum
		condition shared.OSOKConditionType
		phase     shared.OSOKAsyncPhase
	}{
		{
			name:      "creating",
			state:     dnssdk.ZoneLifecycleStateCreating,
			condition: shared.Provisioning,
			phase:     shared.OSOKAsyncPhaseCreate,
		},
		{
			name:      "updating",
			state:     dnssdk.ZoneLifecycleStateUpdating,
			condition: shared.Updating,
			phase:     shared.OSOKAsyncPhaseUpdate,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertZoneLifecycleRequeue(t, tc.state, tc.condition, tc.phase)
		})
	}
}

func TestZoneDeleteKeepsFinalizerWhileLifecycleDeleting(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"

	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{
			{response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)}},
			{response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)}},
			{response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateDeleting)}},
		},
		deleteResults: []zoneDeleteResult{{
			response: dnssdk.DeleteZoneResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			},
		}},
	}

	deleted, err := newTestZoneServiceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false while lifecycle is DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteZone calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
	assertCurrentWorkRequestID(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete")
	assertLatestCondition(t, resource, shared.Terminating)
}

func TestZoneDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"

	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{
			{response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)}},
			{response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)}},
			{err: errortest.NewServiceError(404, errorutil.NotFound, "zone not found")},
		},
		listResults:   []zoneListResult{{response: dnssdk.ListZonesResponse{}}},
		deleteResults: []zoneDeleteResult{{response: dnssdk.DeleteZoneResponse{OpcRequestId: common.String("opc-delete")}}},
	}

	deleted, err := newTestZoneServiceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatalf("Delete() deleted = false, want true after unambiguous not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatalf("status.deletedAt = nil, want delete confirmation timestamp")
	}
	assertLatestCondition(t, resource, shared.Terminating)
}

func TestZoneDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"

	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{
			{response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)}},
			{response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)}},
		},
		deleteResults: []zoneDeleteResult{{
			err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		}},
	}

	deleted, err := newTestZoneServiceClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped 404 error", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false for auth-shaped 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for ambiguous delete", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestZoneDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-pre-error-1"

	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{{err: serviceErr}},
		deleteResults: []zoneDeleteResult{{
			response: dnssdk.DeleteZoneResponse{OpcRequestId: common.String("unexpected-delete")},
		}},
	}

	deleted, err := newTestZoneServiceClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped confirm-read 404", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false for auth-shaped confirm-read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteZone calls = %d, want 0 after auth-shaped confirm-read", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-pre-error-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-confirm-pre-error-1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for ambiguous confirm-read", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestZoneCreateBodySupportsDynectJSONData(t *testing.T) {
	resource := testZoneResource()
	resource.Spec.JsonData = `{
		"migrationSource":"DYNECT",
		"name":"example.com",
		"compartmentId":"ocid1.compartment.oc1..test",
		"dynectMigrationDetails":{
			"customerName":"customer",
			"username":"user",
			"password":"password"
		}
	}`

	body, err := buildZoneCreateBody(context.Background(), resource, "")
	if err != nil {
		t.Fatalf("buildZoneCreateBody() error = %v", err)
	}
	details, ok := body.(dnssdk.CreateMigratedDynectZoneDetails)
	if !ok {
		t.Fatalf("buildZoneCreateBody() = %T, want dnssdk.CreateMigratedDynectZoneDetails", body)
	}
	if got := stringPtrValue(details.DynectMigrationDetails.CustomerName); got != "customer" {
		t.Fatalf("DynectMigrationDetails.CustomerName = %q, want customer", got)
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal dynect create body: %v", err)
	}
	if strings.Contains(string(encoded), "jsonData") {
		t.Fatalf("dynect create body leaked jsonData: %s", string(encoded))
	}
	if !strings.Contains(string(encoded), `"migrationSource":"DYNECT"`) {
		t.Fatalf("dynect create body = %s, want DYNECT discriminator", string(encoded))
	}
}

func newTestZoneServiceClient(client zoneOCIClient) ZoneServiceClient {
	return newZoneServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		client,
		nil,
	)
}

func testZoneResource() *dnsv1beta1.Zone {
	return &dnsv1beta1.Zone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "zone-sample",
			Namespace: "default",
			UID:       types.UID("uid-zone"),
		},
		Spec: dnsv1beta1.ZoneSpec{
			Name:           "example.com",
			CompartmentId:  "ocid1.compartment.oc1..test",
			ZoneType:       string(dnssdk.CreateZoneDetailsZoneTypePrimary),
			Scope:          string(dnssdk.ScopeGlobal),
			ResolutionMode: string(dnssdk.ZoneResolutionModeStatic),
			DnssecState:    string(dnssdk.ZoneDnssecStateDisabled),
		},
	}
}

func configureZoneDynectSpec(resource *dnsv1beta1.Zone, customerName, username, password string) {
	resource.Spec.JsonData = ""
	resource.Spec.MigrationSource = string(dnssdk.CreateZoneBaseDetailsMigrationSourceDynect)
	resource.Spec.DynectMigrationDetails = dnsv1beta1.ZoneDynectMigrationDetails{
		CustomerName: customerName,
		Username:     username,
		Password:     password,
	}
}

func configureZoneDynectJSONData(resource *dnsv1beta1.Zone, customerName, username, password string) {
	resource.Spec.MigrationSource = ""
	resource.Spec.DynectMigrationDetails = dnsv1beta1.ZoneDynectMigrationDetails{}
	resource.Spec.JsonData = fmt.Sprintf(`{
		"migrationSource":"DYNECT",
		"name":"%s",
		"compartmentId":"%s",
		"dynectMigrationDetails":{
			"customerName":"%s",
			"username":"%s",
			"password":"%s"
		}
	}`, resource.Spec.Name, resource.Spec.CompartmentId, customerName, username, password)
}

func seedZoneAppliedDynectFingerprint(t *testing.T, resource *dnsv1beta1.Zone) string {
	t.Helper()
	fingerprint, ok, err := zoneDynectCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		t.Fatalf("zoneDynectCreateOnlyFingerprint() error = %v", err)
	}
	if !ok {
		t.Fatal("zoneDynectCreateOnlyFingerprint() ok = false, want true")
	}
	resource.Status.OsokStatus.Conditions = append(resource.Status.OsokStatus.Conditions, shared.OSOKCondition{
		Type:   shared.Active,
		Reason: zoneDynectCreateOnlyReasonPrefix + fingerprint,
	})
	return fingerprint
}

func assertZoneAppliedDynectFingerprint(t *testing.T, resource *dnsv1beta1.Zone, want string) {
	t.Helper()
	if got := zoneAppliedDynectFingerprint(resource); got != want {
		t.Fatalf("applied Dynect fingerprint = %q, want %q", got, want)
	}
}

func testZoneRequest(resource *dnsv1beta1.Zone) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func testZoneSDK(id string, name string, state dnssdk.ZoneLifecycleStateEnum) dnssdk.Zone {
	return dnssdk.Zone{
		Name:                common.String(name),
		ZoneType:            dnssdk.ZoneZoneTypePrimary,
		CompartmentId:       common.String("ocid1.compartment.oc1..test"),
		Scope:               dnssdk.ScopeGlobal,
		FreeformTags:        map[string]string{},
		DefinedTags:         map[string]map[string]interface{}{},
		ResolutionMode:      dnssdk.ZoneResolutionModeStatic,
		DnssecState:         dnssdk.ZoneDnssecStateDisabled,
		ExternalMasters:     []dnssdk.ExternalMaster{},
		ExternalDownstreams: []dnssdk.ExternalDownstream{},
		Self:                common.String("https://dns.example/zone/" + id),
		Id:                  common.String(id),
		Version:             common.String("1"),
		Serial:              common.Int64(1),
		LifecycleState:      state,
		IsProtected:         common.Bool(false),
		ViewId:              nil,
	}
}

func testZoneSummary(id string, name string, state dnssdk.ZoneSummaryLifecycleStateEnum) dnssdk.ZoneSummary {
	return dnssdk.ZoneSummary{
		Name:           common.String(name),
		ZoneType:       dnssdk.ZoneSummaryZoneTypePrimary,
		CompartmentId:  common.String("ocid1.compartment.oc1..test"),
		Scope:          dnssdk.ScopeGlobal,
		FreeformTags:   map[string]string{},
		DefinedTags:    map[string]map[string]interface{}{},
		ResolutionMode: dnssdk.ZoneResolutionModeStatic,
		DnssecState:    dnssdk.ZoneDnssecStateDisabled,
		Self:           common.String("https://dns.example/zone/" + id),
		Id:             common.String(id),
		Version:        common.String("1"),
		Serial:         common.Int64(1),
		LifecycleState: state,
		IsProtected:    common.Bool(false),
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertReviewedZoneCreateBody(t *testing.T, body dnssdk.CreateZoneDetails, resource *dnsv1beta1.Zone) {
	t.Helper()
	if got := stringPtrValue(body.Name); got != resource.Spec.Name {
		t.Fatalf("create body name = %q, want %q", got, resource.Spec.Name)
	}
	if got := body.FreeformTags["owner"]; got != "dns" {
		t.Fatalf("create body freeformTags[owner] = %q, want dns", got)
	}
	if len(body.ExternalMasters) != 1 || stringPtrValue(body.ExternalMasters[0].Address) != "192.0.2.10" {
		t.Fatalf("create body externalMasters = %#v, want configured master", body.ExternalMasters)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal create body: %v", err)
	}
	if strings.Contains(string(encoded), "jsonData") {
		t.Fatalf("create body leaked jsonData: %s", string(encoded))
	}
}

func assertCreatedZoneStatus(t *testing.T, resource *dnsv1beta1.Zone) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.zone.oc1..created" {
		t.Fatalf("status.ocid = %q, want created OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", got)
	}
}

func assertZoneLifecycleRequeue(
	t *testing.T,
	state dnssdk.ZoneLifecycleStateEnum,
	condition shared.OSOKConditionType,
	phase shared.OSOKAsyncPhase,
) {
	t.Helper()
	resource := testZoneResource()
	resource.Status.OsokStatus.Ocid = "ocid1.zone.oc1..tracked"
	fake := &fakeZoneOCIClient{
		getResults: []zoneGetResult{{
			response: dnssdk.GetZoneResponse{Zone: testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, state)},
		}},
	}

	response, err := newTestZoneServiceClient(fake).CreateOrUpdate(context.Background(), resource, testZoneRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = false, want true")
	}
	if response.RequeueDuration != zoneRequeueDuration {
		t.Fatalf("CreateOrUpdate() RequeueDuration = %s, want %s", response.RequeueDuration, zoneRequeueDuration)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != phase {
		t.Fatalf("status.async.current = %#v, want phase %s", resource.Status.OsokStatus.Async.Current, phase)
	}
	assertLatestCondition(t, resource, condition)
}

func assertLatestCondition(t *testing.T, resource *dnsv1beta1.Zone, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("latest status condition = %s, want %s; all conditions: %#v", got, want, conditions)
	}
}

func assertCurrentWorkRequestID(t *testing.T, resource *dnsv1beta1.Zone, wantPhase shared.OSOKAsyncPhase, wantID string) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.async.current = nil, want phase %s workRequestId %q", wantPhase, wantID)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantID)
	}
}

func assertCurrentLifecycleAsync(
	t *testing.T,
	resource *dnsv1beta1.Zone,
	wantPhase shared.OSOKAsyncPhase,
	wantRawStatus string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.async.current = nil, want phase %s rawStatus %q class %s", wantPhase, wantRawStatus, wantClass)
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.RawStatus != wantRawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, wantRawStatus)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
}

func TestZoneUpdateBodyLeavesNoopEmpty(t *testing.T) {
	resource := testZoneResource()
	current := testZoneSDK("ocid1.zone.oc1..tracked", resource.Spec.Name, dnssdk.ZoneLifecycleStateActive)

	details, needed, err := buildZoneUpdateBody(resource, dnssdk.GetZoneResponse{Zone: current})
	if err != nil {
		t.Fatalf("buildZoneUpdateBody() error = %v", err)
	}
	if needed {
		t.Fatalf("buildZoneUpdateBody() needed = true, want false")
	}
	if !reflect.DeepEqual(details, dnssdk.UpdateZoneDetails{}) {
		t.Fatalf("buildZoneUpdateBody() details = %#v, want empty", details)
	}
}
