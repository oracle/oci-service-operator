/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odaprivateendpointscanproxy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOdaPrivateEndpointID        = "ocid1.odaprivateendpoint.oc1..parent"
	testChangedOdaPrivateEndpointID = "ocid1.odaprivateendpoint.oc1..changed"
	testScanProxyID                 = "ocid1.odaprivateendpointscanproxy.oc1..scanproxy"
)

type fakeOdaPrivateEndpointScanProxyOCIClient struct {
	createFn func(context.Context, odasdk.CreateOdaPrivateEndpointScanProxyRequest) (odasdk.CreateOdaPrivateEndpointScanProxyResponse, error)
	getFn    func(context.Context, odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error)
	listFn   func(context.Context, odasdk.ListOdaPrivateEndpointScanProxiesRequest) (odasdk.ListOdaPrivateEndpointScanProxiesResponse, error)
	deleteFn func(context.Context, odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error)

	createRequests []odasdk.CreateOdaPrivateEndpointScanProxyRequest
	getRequests    []odasdk.GetOdaPrivateEndpointScanProxyRequest
	listRequests   []odasdk.ListOdaPrivateEndpointScanProxiesRequest
	deleteRequests []odasdk.DeleteOdaPrivateEndpointScanProxyRequest
}

func (f *fakeOdaPrivateEndpointScanProxyOCIClient) CreateOdaPrivateEndpointScanProxy(ctx context.Context, req odasdk.CreateOdaPrivateEndpointScanProxyRequest) (odasdk.CreateOdaPrivateEndpointScanProxyResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return odasdk.CreateOdaPrivateEndpointScanProxyResponse{}, nil
}

func (f *fakeOdaPrivateEndpointScanProxyOCIClient) GetOdaPrivateEndpointScanProxy(ctx context.Context, req odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return odasdk.GetOdaPrivateEndpointScanProxyResponse{}, nil
}

func (f *fakeOdaPrivateEndpointScanProxyOCIClient) ListOdaPrivateEndpointScanProxies(ctx context.Context, req odasdk.ListOdaPrivateEndpointScanProxiesRequest) (odasdk.ListOdaPrivateEndpointScanProxiesResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return odasdk.ListOdaPrivateEndpointScanProxiesResponse{}, nil
}

func (f *fakeOdaPrivateEndpointScanProxyOCIClient) DeleteOdaPrivateEndpointScanProxy(ctx context.Context, req odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return odasdk.DeleteOdaPrivateEndpointScanProxyResponse{}, nil
}

func newTestOdaPrivateEndpointScanProxyClient(fake *fakeOdaPrivateEndpointScanProxyOCIClient) OdaPrivateEndpointScanProxyServiceClient {
	return newOdaPrivateEndpointScanProxyServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
}

func makeTestOdaPrivateEndpointScanProxy() *odav1beta1.OdaPrivateEndpointScanProxy {
	return &odav1beta1.OdaPrivateEndpointScanProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scan-proxy",
			Namespace: "default",
			UID:       types.UID("uid-scan-proxy"),
			Annotations: map[string]string{
				OdaPrivateEndpointIDAnnotation: testOdaPrivateEndpointID,
			},
		},
		Spec: odav1beta1.OdaPrivateEndpointScanProxySpec{
			ScanListenerType: string(odasdk.OdaPrivateEndpointScanProxyScanListenerTypeFqdn),
			Protocol:         string(odasdk.OdaPrivateEndpointScanProxyProtocolTcp),
			ScanListenerInfos: []odav1beta1.OdaPrivateEndpointScanProxyScanListenerInfo{
				{
					ScanListenerFqdn: "scan.example.com",
					ScanListenerPort: 1521,
				},
			},
		},
	}
}

func makeSDKOdaPrivateEndpointScanProxy(id string, state odasdk.OdaPrivateEndpointScanProxyLifecycleStateEnum) odasdk.OdaPrivateEndpointScanProxy {
	created := common.SDKTime{Time: time.Date(2026, 4, 27, 16, 0, 0, 0, time.UTC)}
	return odasdk.OdaPrivateEndpointScanProxy{
		Id:               common.String(id),
		ScanListenerType: odasdk.OdaPrivateEndpointScanProxyScanListenerTypeFqdn,
		Protocol:         odasdk.OdaPrivateEndpointScanProxyProtocolTcp,
		ScanListenerInfos: []odasdk.ScanListenerInfo{
			{
				ScanListenerFqdn: common.String("scan.example.com"),
				ScanListenerPort: common.Int(1521),
			},
		},
		LifecycleState: state,
		TimeCreated:    &created,
	}
}

func makeSDKOdaPrivateEndpointScanProxySummary(id string, state odasdk.OdaPrivateEndpointScanProxyLifecycleStateEnum) odasdk.OdaPrivateEndpointScanProxySummary {
	current := makeSDKOdaPrivateEndpointScanProxy(id, state)
	return odasdk.OdaPrivateEndpointScanProxySummary{
		Id:                current.Id,
		ScanListenerType:  current.ScanListenerType,
		Protocol:          current.Protocol,
		ScanListenerInfos: current.ScanListenerInfos,
		LifecycleState:    current.LifecycleState,
		TimeCreated:       current.TimeCreated,
	}
}

func TestOdaPrivateEndpointScanProxyCreateOrUpdateCreatesWithAnnotatedParent(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		listFn: func(_ context.Context, req odasdk.ListOdaPrivateEndpointScanProxiesRequest) (odasdk.ListOdaPrivateEndpointScanProxiesResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			return odasdk.ListOdaPrivateEndpointScanProxiesResponse{}, nil
		},
		createFn: func(_ context.Context, req odasdk.CreateOdaPrivateEndpointScanProxyRequest) (odasdk.CreateOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			if req.ScanListenerType != odasdk.OdaPrivateEndpointScanProxyScanListenerTypeFqdn {
				t.Fatalf("create scanListenerType = %q, want FQDN", req.ScanListenerType)
			}
			if req.Protocol != odasdk.OdaPrivateEndpointScanProxyProtocolTcp {
				t.Fatalf("create protocol = %q, want TCP", req.Protocol)
			}
			if got := len(req.ScanListenerInfos); got != 1 {
				t.Fatalf("create scanListenerInfos length = %d, want 1", got)
			}
			requireStringPtr(t, req.ScanListenerInfos[0].ScanListenerFqdn, "scan.example.com")
			requireIntPtr(t, req.ScanListenerInfos[0].ScanListenerPort, 1521)
			requireStringPtr(t, req.OpcRetryToken, string(resource.UID))
			return odasdk.CreateOdaPrivateEndpointScanProxyResponse{
				OdaPrivateEndpointScanProxy: makeSDKOdaPrivateEndpointScanProxy(testScanProxyID, odasdk.OdaPrivateEndpointScanProxyLifecycleStateActive),
				OpcRequestId:                common.String("opc-create"),
				OpcWorkRequestId:            common.String("wr-create"),
			}, nil
		},
	}

	response, err := newTestOdaPrivateEndpointScanProxyClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if resource.Status.Id != testScanProxyID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testScanProxyID)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(testScanProxyID) {
		t.Fatalf("status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testScanProxyID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestOdaPrivateEndpointScanProxyCreateOrUpdateBindsExistingFromList(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		listFn: func(_ context.Context, req odasdk.ListOdaPrivateEndpointScanProxiesRequest) (odasdk.ListOdaPrivateEndpointScanProxiesResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			return odasdk.ListOdaPrivateEndpointScanProxiesResponse{
				OdaPrivateEndpointScanProxyCollection: odasdk.OdaPrivateEndpointScanProxyCollection{
					Items: []odasdk.OdaPrivateEndpointScanProxySummary{
						makeSDKOdaPrivateEndpointScanProxySummary(testScanProxyID, odasdk.OdaPrivateEndpointScanProxyLifecycleStateActive),
					},
				},
			}, nil
		},
		createFn: func(context.Context, odasdk.CreateOdaPrivateEndpointScanProxyRequest) (odasdk.CreateOdaPrivateEndpointScanProxyResponse, error) {
			t.Fatal("CreateOdaPrivateEndpointScanProxy should not be called when list finds a matching scan proxy")
			return odasdk.CreateOdaPrivateEndpointScanProxyResponse{}, nil
		},
	}

	response, err := newTestOdaPrivateEndpointScanProxyClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful without requeue", response)
	}
	if resource.Status.Id != testScanProxyID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testScanProxyID)
	}
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestOdaPrivateEndpointScanProxyCreateOrUpdateObservesExistingActive(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	resource.Status.Id = testScanProxyID
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		getFn: func(_ context.Context, req odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			requireStringPtr(t, req.OdaPrivateEndpointScanProxyId, testScanProxyID)
			return odasdk.GetOdaPrivateEndpointScanProxyResponse{
				OdaPrivateEndpointScanProxy: makeSDKOdaPrivateEndpointScanProxy(testScanProxyID, odasdk.OdaPrivateEndpointScanProxyLifecycleStateActive),
			}, nil
		},
	}

	response, err := newTestOdaPrivateEndpointScanProxyClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful without requeue", response)
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestOdaPrivateEndpointScanProxyCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	resource.Status.Id = testScanProxyID
	current := makeSDKOdaPrivateEndpointScanProxy(testScanProxyID, odasdk.OdaPrivateEndpointScanProxyLifecycleStateActive)
	current.ScanListenerInfos[0].ScanListenerPort = common.Int(1522)
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		getFn: func(context.Context, odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
			return odasdk.GetOdaPrivateEndpointScanProxyResponse{OdaPrivateEndpointScanProxy: current}, nil
		},
	}

	response, err := newTestOdaPrivateEndpointScanProxyClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil || !strings.Contains(err.Error(), "create-only") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only drift error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
}

func TestOdaPrivateEndpointScanProxyCreateOrUpdateRejectsParentAnnotationDriftWhenTrackedIDNotFound(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	resource.Status.Id = testScanProxyID
	resource.Annotations[OdaPrivateEndpointIDAnnotation] = testChangedOdaPrivateEndpointID
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		getFn: func(_ context.Context, req odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testChangedOdaPrivateEndpointID)
			requireStringPtr(t, req.OdaPrivateEndpointScanProxyId, testScanProxyID)
			return odasdk.GetOdaPrivateEndpointScanProxyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
		},
		listFn: func(context.Context, odasdk.ListOdaPrivateEndpointScanProxiesRequest) (odasdk.ListOdaPrivateEndpointScanProxiesResponse, error) {
			t.Fatal("ListOdaPrivateEndpointScanProxies should not be called after tracked parent drift")
			return odasdk.ListOdaPrivateEndpointScanProxiesResponse{}, nil
		},
		createFn: func(context.Context, odasdk.CreateOdaPrivateEndpointScanProxyRequest) (odasdk.CreateOdaPrivateEndpointScanProxyResponse, error) {
			t.Fatal("CreateOdaPrivateEndpointScanProxy should not be called after tracked parent drift")
			return odasdk.CreateOdaPrivateEndpointScanProxyResponse{}, nil
		},
	}

	response, err := newTestOdaPrivateEndpointScanProxyClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil || !strings.Contains(err.Error(), "create-only") || !strings.Contains(err.Error(), OdaPrivateEndpointIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want parent annotation create-only drift error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	if resource.Status.Id != testScanProxyID {
		t.Fatalf("status.id = %q, want tracked scan proxy id preserved", resource.Status.Id)
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("list requests = %d, want 0", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
}

func TestOdaPrivateEndpointScanProxyCreateOrUpdateClassifiesLifecycleStates(t *testing.T) {
	tests := []struct {
		name           string
		state          odasdk.OdaPrivateEndpointScanProxyLifecycleStateEnum
		wantSuccessful bool
		wantRequeue    bool
		wantCondition  shared.OSOKConditionType
		wantStatus     v1.ConditionStatus
		wantPhase      shared.OSOKAsyncPhase
	}{
		{
			name:           "creating",
			state:          odasdk.OdaPrivateEndpointScanProxyLifecycleStateCreating,
			wantSuccessful: true,
			wantRequeue:    true,
			wantCondition:  shared.Provisioning,
			wantStatus:     v1.ConditionTrue,
			wantPhase:      shared.OSOKAsyncPhaseCreate,
		},
		{
			name:           "deleting",
			state:          odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting,
			wantSuccessful: true,
			wantRequeue:    true,
			wantCondition:  shared.Terminating,
			wantStatus:     v1.ConditionTrue,
			wantPhase:      shared.OSOKAsyncPhaseDelete,
		},
		{
			name:           "failed",
			state:          odasdk.OdaPrivateEndpointScanProxyLifecycleStateFailed,
			wantSuccessful: false,
			wantRequeue:    false,
			wantCondition:  shared.Failed,
			wantStatus:     v1.ConditionFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := makeTestOdaPrivateEndpointScanProxy()
			resource.Status.Id = testScanProxyID
			fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
				getFn: func(context.Context, odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
					return odasdk.GetOdaPrivateEndpointScanProxyResponse{
						OdaPrivateEndpointScanProxy: makeSDKOdaPrivateEndpointScanProxy(testScanProxyID, tt.state),
					}, nil
				},
			}

			response, err := newTestOdaPrivateEndpointScanProxyClient(fake).CreateOrUpdate(context.Background(), resource, testRequest())
			if err != nil {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tt.wantSuccessful {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t", response.IsSuccessful, tt.wantSuccessful)
			}
			if response.ShouldRequeue != tt.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tt.wantRequeue)
			}
			requireCondition(t, resource, tt.wantCondition, tt.wantStatus)
			if tt.wantPhase != "" {
				if resource.Status.OsokStatus.Async.Current == nil {
					t.Fatal("status.async.current = nil, want lifecycle async operation")
				}
				if resource.Status.OsokStatus.Async.Current.Phase != tt.wantPhase {
					t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, tt.wantPhase)
				}
			}
		})
	}
}

func TestOdaPrivateEndpointScanProxyDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	resource.Status.Id = testScanProxyID
	getCalls := 0
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		getFn: func(_ context.Context, req odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			requireStringPtr(t, req.OdaPrivateEndpointScanProxyId, testScanProxyID)
			getCalls++
			state := odasdk.OdaPrivateEndpointScanProxyLifecycleStateActive
			if getCalls > 1 {
				state = odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleting
			}
			return odasdk.GetOdaPrivateEndpointScanProxyResponse{
				OdaPrivateEndpointScanProxy: makeSDKOdaPrivateEndpointScanProxy(testScanProxyID, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			requireStringPtr(t, req.OdaPrivateEndpointScanProxyId, testScanProxyID)
			return odasdk.DeleteOdaPrivateEndpointScanProxyResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
	}

	deleted, err := newTestOdaPrivateEndpointScanProxyClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want delete lifecycle tracking")
	}
	if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.phase = %q, want delete", resource.Status.OsokStatus.Async.Current.Phase)
	}
	if resource.Status.OsokStatus.Async.Current.WorkRequestID != "wr-delete" {
		t.Fatalf("status.async.current.workRequestId = %q, want wr-delete", resource.Status.OsokStatus.Async.Current.WorkRequestID)
	}
}

func TestOdaPrivateEndpointScanProxyDeleteReleasesOnTerminalLifecycle(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	resource.Status.Id = testScanProxyID
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		getFn: func(context.Context, odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
			return odasdk.GetOdaPrivateEndpointScanProxyResponse{
				OdaPrivateEndpointScanProxy: makeSDKOdaPrivateEndpointScanProxy(testScanProxyID, odasdk.OdaPrivateEndpointScanProxyLifecycleStateDeleted),
			}, nil
		},
		deleteFn: func(context.Context, odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error) {
			t.Fatal("DeleteOdaPrivateEndpointScanProxy should not be called after readback confirms DELETED")
			return odasdk.DeleteOdaPrivateEndpointScanProxyResponse{}, nil
		},
	}

	deleted, err := newTestOdaPrivateEndpointScanProxyClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestOdaPrivateEndpointScanProxyDeleteRetainsFinalizerOnPreDeleteNotFound(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	resource.Status.Id = testScanProxyID
	resource.Annotations[OdaPrivateEndpointIDAnnotation] = testChangedOdaPrivateEndpointID
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		getFn: func(_ context.Context, req odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testChangedOdaPrivateEndpointID)
			requireStringPtr(t, req.OdaPrivateEndpointScanProxyId, testScanProxyID)
			return odasdk.GetOdaPrivateEndpointScanProxyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
		},
		deleteFn: func(context.Context, odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error) {
			t.Fatal("DeleteOdaPrivateEndpointScanProxy should not be called when pre-delete read is ambiguous")
			return odasdk.DeleteOdaPrivateEndpointScanProxyResponse{}, nil
		},
	}

	deleted, err := newTestOdaPrivateEndpointScanProxyClient(fake).Delete(context.Background(), resource)
	if err == nil ||
		!strings.Contains(err.Error(), "delete confirmation") ||
		!strings.Contains(err.Error(), OdaPrivateEndpointIDAnnotation) {
		t.Fatalf("Delete() error = %v, want ambiguous parent annotation delete confirmation error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false so finalizer is retained")
	}
	if resource.Status.Id != testScanProxyID {
		t.Fatalf("status.id = %q, want tracked scan proxy id preserved", resource.Status.Id)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt is set, want nil while delete confirmation is ambiguous")
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(fake.getRequests))
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0", len(fake.deleteRequests))
	}
}

func TestOdaPrivateEndpointScanProxyDeleteReleasesOnConfirmNotFoundAfterDeleteRequest(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	resource.Status.Id = testScanProxyID
	getCalls := 0
	fake := &fakeOdaPrivateEndpointScanProxyOCIClient{
		getFn: func(_ context.Context, req odasdk.GetOdaPrivateEndpointScanProxyRequest) (odasdk.GetOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			requireStringPtr(t, req.OdaPrivateEndpointScanProxyId, testScanProxyID)
			getCalls++
			if getCalls > 1 {
				return odasdk.GetOdaPrivateEndpointScanProxyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
			}
			return odasdk.GetOdaPrivateEndpointScanProxyResponse{
				OdaPrivateEndpointScanProxy: makeSDKOdaPrivateEndpointScanProxy(testScanProxyID, odasdk.OdaPrivateEndpointScanProxyLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req odasdk.DeleteOdaPrivateEndpointScanProxyRequest) (odasdk.DeleteOdaPrivateEndpointScanProxyResponse, error) {
			requireStringPtr(t, req.OdaPrivateEndpointId, testOdaPrivateEndpointID)
			requireStringPtr(t, req.OdaPrivateEndpointScanProxyId, testScanProxyID)
			return odasdk.DeleteOdaPrivateEndpointScanProxyResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestOdaPrivateEndpointScanProxyClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after delete confirmation returns not found")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestOdaPrivateEndpointScanProxyCreateOrUpdateRequiresParentAnnotation(t *testing.T) {
	resource := makeTestOdaPrivateEndpointScanProxy()
	delete(resource.Annotations, OdaPrivateEndpointIDAnnotation)

	response, err := newTestOdaPrivateEndpointScanProxyClient(&fakeOdaPrivateEndpointScanProxyOCIClient{}).
		CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil || !strings.Contains(err.Error(), OdaPrivateEndpointIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing parent annotation error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func requireStringPtr(t *testing.T, actual *string, want string) {
	t.Helper()
	if actual == nil {
		t.Fatalf("string pointer = nil, want %q", want)
	}
	if *actual != want {
		t.Fatalf("string pointer = %q, want %q", *actual, want)
	}
}

func testRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "scan-proxy"}}
}

func requireIntPtr(t *testing.T, actual *int, want int) {
	t.Helper()
	if actual == nil {
		t.Fatalf("int pointer = nil, want %d", want)
	}
	if *actual != want {
		t.Fatalf("int pointer = %d, want %d", *actual, want)
	}
}

func requireCondition(
	t *testing.T,
	resource *odav1beta1.OdaPrivateEndpointScanProxy,
	condition shared.OSOKConditionType,
	status v1.ConditionStatus,
) {
	t.Helper()
	for _, actual := range resource.Status.OsokStatus.Conditions {
		if actual.Type == condition && actual.Status == status {
			return
		}
	}
	t.Fatalf("condition %q with status %q not found in %#v", condition, status, resource.Status.OsokStatus.Conditions)
}
