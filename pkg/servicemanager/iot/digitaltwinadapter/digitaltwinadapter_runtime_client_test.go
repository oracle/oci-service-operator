/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package digitaltwinadapter

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDigitalTwinAdapterID       = "ocid1.digitaltwinadapter.oc1..adapter"
	testDigitalTwinAdapterDomainID = "ocid1.iotdomain.oc1..domain"
	testDigitalTwinAdapterModelID  = "ocid1.digitaltwinmodel.oc1..model"
	testDigitalTwinAdapterSpecURI  = "dtmi:com:example:Thermostat;1"
	testDigitalTwinAdapterName     = "adapter-sample"
)

type fakeDigitalTwinAdapterOCIClient struct {
	createFn func(context.Context, iotsdk.CreateDigitalTwinAdapterRequest) (iotsdk.CreateDigitalTwinAdapterResponse, error)
	getFn    func(context.Context, iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error)
	listFn   func(context.Context, iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error)
	updateFn func(context.Context, iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error)
	deleteFn func(context.Context, iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error)
}

func (f *fakeDigitalTwinAdapterOCIClient) CreateDigitalTwinAdapter(ctx context.Context, req iotsdk.CreateDigitalTwinAdapterRequest) (iotsdk.CreateDigitalTwinAdapterResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return iotsdk.CreateDigitalTwinAdapterResponse{}, nil
}

func (f *fakeDigitalTwinAdapterOCIClient) GetDigitalTwinAdapter(ctx context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return iotsdk.GetDigitalTwinAdapterResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "digital twin adapter is missing")
}

func (f *fakeDigitalTwinAdapterOCIClient) ListDigitalTwinAdapters(ctx context.Context, req iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return iotsdk.ListDigitalTwinAdaptersResponse{}, nil
}

func (f *fakeDigitalTwinAdapterOCIClient) UpdateDigitalTwinAdapter(ctx context.Context, req iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return iotsdk.UpdateDigitalTwinAdapterResponse{}, nil
}

func (f *fakeDigitalTwinAdapterOCIClient) DeleteDigitalTwinAdapter(ctx context.Context, req iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return iotsdk.DeleteDigitalTwinAdapterResponse{}, nil
}

func newTestDigitalTwinAdapterClient(client digitalTwinAdapterOCIClient) DigitalTwinAdapterServiceClient {
	return newDigitalTwinAdapterServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDigitalTwinAdapterResource() *iotv1beta1.DigitalTwinAdapter {
	return &iotv1beta1.DigitalTwinAdapter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDigitalTwinAdapterName,
			Namespace: "default",
		},
		Spec: iotv1beta1.DigitalTwinAdapterSpec{
			IotDomainId:             testDigitalTwinAdapterDomainID,
			DigitalTwinModelId:      testDigitalTwinAdapterModelID,
			DigitalTwinModelSpecUri: testDigitalTwinAdapterSpecURI,
			DisplayName:             testDigitalTwinAdapterName,
			Description:             "initial description",
			InboundEnvelope: iotv1beta1.DigitalTwinAdapterInboundEnvelope{
				ReferenceEndpoint: "device/temperature",
				ReferencePayload: iotv1beta1.DigitalTwinAdapterInboundEnvelopeReferencePayload{
					DataFormat: "JSON",
					Data: map[string]shared.JSONValue{
						"temperature": jsonValue("72"),
					},
				},
				EnvelopeMapping: iotv1beta1.DigitalTwinAdapterInboundEnvelopeEnvelopeMapping{
					TimeObserved: "$.time",
				},
			},
			InboundRoutes: []iotv1beta1.DigitalTwinAdapterInboundRoute{
				{
					Condition: "true",
					ReferencePayload: iotv1beta1.DigitalTwinAdapterInboundRouteReferencePayload{
						JsonData: `{"humidity":55}`,
					},
					PayloadMapping: map[string]string{"humidity": "$.humidity"},
					Description:    "default route",
				},
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedDigitalTwinAdapterResource() *iotv1beta1.DigitalTwinAdapter {
	resource := makeDigitalTwinAdapterResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDigitalTwinAdapterID)
	resource.Status.Id = testDigitalTwinAdapterID
	resource.Status.IotDomainId = testDigitalTwinAdapterDomainID
	resource.Status.DigitalTwinModelId = testDigitalTwinAdapterModelID
	resource.Status.DigitalTwinModelSpecUri = testDigitalTwinAdapterSpecURI
	resource.Status.DisplayName = testDigitalTwinAdapterName
	resource.Status.LifecycleState = string(iotsdk.LifecycleStateActive)
	return resource
}

func makeDigitalTwinAdapterRequest(resource *iotv1beta1.DigitalTwinAdapter) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKDigitalTwinAdapter(
	id string,
	spec iotv1beta1.DigitalTwinAdapterSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinAdapter {
	envelope, _, _ := digitalTwinAdapterInboundEnvelope(spec.InboundEnvelope)
	routes, _, _ := digitalTwinAdapterInboundRoutes(spec.InboundRoutes)
	return iotsdk.DigitalTwinAdapter{
		Id:                      common.String(id),
		IotDomainId:             common.String(spec.IotDomainId),
		DigitalTwinModelId:      common.String(spec.DigitalTwinModelId),
		DigitalTwinModelSpecUri: common.String(spec.DigitalTwinModelSpecUri),
		DisplayName:             common.String(spec.DisplayName),
		Description:             common.String(spec.Description),
		InboundEnvelope:         envelope,
		InboundRoutes:           routes,
		LifecycleState:          state,
		FreeformTags:            cloneDigitalTwinAdapterStringMap(spec.FreeformTags),
		DefinedTags:             digitalTwinAdapterDefinedTags(spec.DefinedTags),
	}
}

func makeSDKDigitalTwinAdapterSummary(
	id string,
	spec iotv1beta1.DigitalTwinAdapterSpec,
	state iotsdk.LifecycleStateEnum,
) iotsdk.DigitalTwinAdapterSummary {
	return iotsdk.DigitalTwinAdapterSummary{
		Id:                      common.String(id),
		IotDomainId:             common.String(spec.IotDomainId),
		DigitalTwinModelId:      common.String(spec.DigitalTwinModelId),
		DigitalTwinModelSpecUri: common.String(spec.DigitalTwinModelSpecUri),
		DisplayName:             common.String(spec.DisplayName),
		Description:             common.String(spec.Description),
		LifecycleState:          state,
		FreeformTags:            cloneDigitalTwinAdapterStringMap(spec.FreeformTags),
		DefinedTags:             digitalTwinAdapterDefinedTags(spec.DefinedTags),
	}
}

func TestDigitalTwinAdapterCreateOrUpdateBindsExistingAdapterByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinAdapterResource()
	createCalled := false
	updateCalled := false
	listCalls := 0
	getCalls := 0

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		listFn: func(_ context.Context, req iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error) {
			listCalls++
			requireStringPtr(t, "ListDigitalTwinAdaptersRequest.IotDomainId", req.IotDomainId, resource.Spec.IotDomainId)
			requireStringPtr(t, "ListDigitalTwinAdaptersRequest.DigitalTwinModelId", req.DigitalTwinModelId, resource.Spec.DigitalTwinModelId)
			requireStringPtr(t, "ListDigitalTwinAdaptersRequest.DigitalTwinModelSpecUri", req.DigitalTwinModelSpecUri, resource.Spec.DigitalTwinModelSpecUri)
			if listCalls == 1 {
				if req.Page != nil {
					t.Fatalf("first ListDigitalTwinAdaptersRequest.Page = %q, want nil", *req.Page)
				}
				otherSpec := resource.Spec
				otherSpec.DisplayName = "other-adapter"
				return iotsdk.ListDigitalTwinAdaptersResponse{
					DigitalTwinAdapterCollection: iotsdk.DigitalTwinAdapterCollection{
						Items: []iotsdk.DigitalTwinAdapterSummary{
							makeSDKDigitalTwinAdapterSummary("ocid1.digitaltwinadapter.oc1..other", otherSpec, iotsdk.LifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second ListDigitalTwinAdaptersRequest.Page", req.Page, "page-2")
			return iotsdk.ListDigitalTwinAdaptersResponse{
				DigitalTwinAdapterCollection: iotsdk.DigitalTwinAdapterCollection{
					Items: []iotsdk.DigitalTwinAdapterSummary{
						makeSDKDigitalTwinAdapterSummary(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.GetDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinAdapterRequest) (iotsdk.CreateDigitalTwinAdapterResponse, error) {
			createCalled = true
			return iotsdk.CreateDigitalTwinAdapterResponse{}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinAdapterResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinAdapterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateDigitalTwinAdapter() called for existing adapter")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinAdapter() called for matching adapter")
	}
	if listCalls != 2 {
		t.Fatalf("ListDigitalTwinAdapters() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetDigitalTwinAdapter() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalTwinAdapterID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDigitalTwinAdapterID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinAdapterCreateRecordsTypedPayloadRetryTokenRequestIDAndStatus(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinAdapterResource()
	listCalls := 0
	createCalls := 0

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		listFn: func(context.Context, iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error) {
			listCalls++
			return iotsdk.ListDigitalTwinAdaptersResponse{}, nil
		},
		createFn: func(_ context.Context, req iotsdk.CreateDigitalTwinAdapterRequest) (iotsdk.CreateDigitalTwinAdapterResponse, error) {
			createCalls++
			requireDigitalTwinAdapterCreateRequest(t, req, resource)
			return iotsdk.CreateDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:       common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.GetDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinAdapterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if listCalls != 1 {
		t.Fatalf("ListDigitalTwinAdapters() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateDigitalTwinAdapter() calls = %d, want 1", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDigitalTwinAdapterID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDigitalTwinAdapterID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if got := resource.Status.InboundEnvelope.ReferencePayload.DataFormat; got != "JSON" {
		t.Fatalf("status.inboundEnvelope.referencePayload.dataFormat = %q, want JSON", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinAdapterCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinAdapterResource()
	updateCalled := false

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.GetDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinAdapterResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinAdapterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinAdapter() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinAdapterCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinAdapterResource()
	resource.Spec.DisplayName = "updated-adapter"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := resource.Spec
	currentSpec.DisplayName = testDigitalTwinAdapterName
	currentSpec.Description = "initial description"
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	getCalls := 0
	updateCalls := 0

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			if getCalls == 1 {
				return iotsdk.GetDigitalTwinAdapterResponse{
					DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, currentSpec, iotsdk.LifecycleStateActive),
				}, nil
			}
			return iotsdk.GetDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			requireStringPtr(t, "UpdateDigitalTwinAdapterDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireStringPtr(t, "UpdateDigitalTwinAdapterDetails.Description", req.Description, resource.Spec.Description)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateDigitalTwinAdapterDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateDigitalTwinAdapterDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return iotsdk.UpdateDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
				OpcRequestId:       common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinAdapterRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateDigitalTwinAdapter() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetDigitalTwinAdapter() calls = %d, want current read and update follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestDigitalTwinAdapterCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinAdapterResource()
	resource.Spec.IotDomainId = "ocid1.iotdomain.oc1..different"
	currentSpec := resource.Spec
	currentSpec.IotDomainId = testDigitalTwinAdapterDomainID
	updateCalled := false

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.GetDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, currentSpec, iotsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error) {
			updateCalled = true
			return iotsdk.UpdateDigitalTwinAdapterResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinAdapterRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateDigitalTwinAdapter() called after create-only iotDomainId drift")
	}
	if !strings.Contains(err.Error(), "iotDomainId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want iotDomainId force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestDigitalTwinAdapterCreateOrUpdateRejectsOmittedOptionalCreateOnlyFieldsBeforeUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		omitField func(*iotv1beta1.DigitalTwinAdapter)
		wantError string
	}{
		{
			name: "digitalTwinModelId",
			omitField: func(resource *iotv1beta1.DigitalTwinAdapter) {
				resource.Spec.DigitalTwinModelId = ""
			},
			wantError: "digitalTwinModelId changes",
		},
		{
			name: "digitalTwinModelSpecUri",
			omitField: func(resource *iotv1beta1.DigitalTwinAdapter) {
				resource.Spec.DigitalTwinModelSpecUri = ""
			},
			wantError: "digitalTwinModelSpecUri changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedDigitalTwinAdapterResource()
			currentSpec := resource.Spec
			resource.Spec.DisplayName = "updated-adapter"
			tt.omitField(resource)
			updateCalled := false

			client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
				getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
					requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
					return iotsdk.GetDigitalTwinAdapterResponse{
						DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, currentSpec, iotsdk.LifecycleStateActive),
					}, nil
				},
				updateFn: func(context.Context, iotsdk.UpdateDigitalTwinAdapterRequest) (iotsdk.UpdateDigitalTwinAdapterResponse, error) {
					updateCalled = true
					return iotsdk.UpdateDigitalTwinAdapterResponse{}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinAdapterRequest(resource))
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want optional create-only drift rejection")
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() successful = true, want false")
			}
			if updateCalled {
				t.Fatalf("UpdateDigitalTwinAdapter() called after omitted %s drift", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("CreateOrUpdate() error = %v, want %q", err, tt.wantError)
			}
			requireLastCondition(t, resource, shared.Failed)
		})
	}
}

func TestDigitalTwinAdapterDeleteKeepsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinAdapterResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			switch getCalls {
			case 1, 2, 3:
				return iotsdk.GetDigitalTwinAdapterResponse{
					DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
				}, nil
			default:
				return iotsdk.GetDigitalTwinAdapterResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "digital twin adapter is gone")
			}
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.DeleteDigitalTwinAdapterResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinAdapter() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireDeletePendingStatus(t, resource)
	requireLastCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinAdapter() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestDigitalTwinAdapterDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinAdapterResource()

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.GetDigitalTwinAdapterResponse{
				DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error) {
			requireStringPtr(t, "DeleteDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.DeleteDigitalTwinAdapterResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinAdapterDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinAdapterResource()
	deleteCalled := false

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.GetDigitalTwinAdapterResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error) {
			deleteCalled = true
			return iotsdk.DeleteDigitalTwinAdapterResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteDigitalTwinAdapter() called after auth-shaped pre-delete confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinAdapterDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedDigitalTwinAdapterResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetDigitalTwinAdapterRequest) (iotsdk.GetDigitalTwinAdapterResponse, error) {
			getCalls++
			requireStringPtr(t, "GetDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			if getCalls < 3 {
				return iotsdk.GetDigitalTwinAdapterResponse{
					DigitalTwinAdapter: makeSDKDigitalTwinAdapter(testDigitalTwinAdapterID, resource.Spec, iotsdk.LifecycleStateActive),
				}, nil
			}
			return iotsdk.GetDigitalTwinAdapterResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteDigitalTwinAdapterRequest) (iotsdk.DeleteDigitalTwinAdapterResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteDigitalTwinAdapterRequest.DigitalTwinAdapterId", req.DigitalTwinAdapterId, testDigitalTwinAdapterID)
			return iotsdk.DeleteDigitalTwinAdapterResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped post-delete confirm read")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteDigitalTwinAdapter() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDigitalTwinAdapterCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinAdapterResource()

	client := newTestDigitalTwinAdapterClient(&fakeDigitalTwinAdapterOCIClient{
		listFn: func(context.Context, iotsdk.ListDigitalTwinAdaptersRequest) (iotsdk.ListDigitalTwinAdaptersResponse, error) {
			return iotsdk.ListDigitalTwinAdaptersResponse{}, nil
		},
		createFn: func(context.Context, iotsdk.CreateDigitalTwinAdapterRequest) (iotsdk.CreateDigitalTwinAdapterResponse, error) {
			return iotsdk.CreateDigitalTwinAdapterResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeDigitalTwinAdapterRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireDigitalTwinAdapterCreateRequest(
	t *testing.T,
	req iotsdk.CreateDigitalTwinAdapterRequest,
	resource *iotv1beta1.DigitalTwinAdapter,
) {
	t.Helper()
	requireStringPtr(t, "CreateDigitalTwinAdapterDetails.IotDomainId", req.IotDomainId, resource.Spec.IotDomainId)
	requireStringPtr(t, "CreateDigitalTwinAdapterDetails.DigitalTwinModelId", req.DigitalTwinModelId, resource.Spec.DigitalTwinModelId)
	requireStringPtr(t, "CreateDigitalTwinAdapterDetails.DigitalTwinModelSpecUri", req.DigitalTwinModelSpecUri, resource.Spec.DigitalTwinModelSpecUri)
	requireStringPtr(t, "CreateDigitalTwinAdapterDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateDigitalTwinAdapterDetails.Description", req.Description, resource.Spec.Description)
	requireStringPtr(t, "CreateDigitalTwinAdapterDetails.InboundEnvelope.ReferenceEndpoint", req.InboundEnvelope.ReferenceEndpoint, "device/temperature")
	if _, ok := req.InboundEnvelope.ReferencePayload.(iotsdk.DigitalTwinAdapterJsonPayload); !ok {
		t.Fatalf("CreateDigitalTwinAdapterDetails.InboundEnvelope.ReferencePayload = %T, want DigitalTwinAdapterJsonPayload", req.InboundEnvelope.ReferencePayload)
	}
	if len(req.InboundRoutes) != 1 {
		t.Fatalf("CreateDigitalTwinAdapterDetails.InboundRoutes length = %d, want 1", len(req.InboundRoutes))
	}
	if _, ok := req.InboundRoutes[0].ReferencePayload.(iotsdk.DigitalTwinAdapterJsonPayload); !ok {
		t.Fatalf("CreateDigitalTwinAdapterDetails.InboundRoutes[0].ReferencePayload = %T, want DigitalTwinAdapterJsonPayload", req.InboundRoutes[0].ReferencePayload)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateDigitalTwinAdapterRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireDeletePendingStatus(t *testing.T, resource *iotv1beta1.DigitalTwinAdapter) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(iotsdk.LifecycleStateActive) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle delete pending ACTIVE", current)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireLastCondition(t *testing.T, resource *iotv1beta1.DigitalTwinAdapter, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}

func jsonValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}
