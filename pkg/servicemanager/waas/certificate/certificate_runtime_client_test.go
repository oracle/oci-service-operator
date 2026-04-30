/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package certificate

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeCertificateOCIClient struct {
	changeCompartmentFunc func(context.Context, waassdk.ChangeCertificateCompartmentRequest) (waassdk.ChangeCertificateCompartmentResponse, error)
	createFunc            func(context.Context, waassdk.CreateCertificateRequest) (waassdk.CreateCertificateResponse, error)
	getFunc               func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error)
	listFunc              func(context.Context, waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error)
	updateFunc            func(context.Context, waassdk.UpdateCertificateRequest) (waassdk.UpdateCertificateResponse, error)
	deleteFunc            func(context.Context, waassdk.DeleteCertificateRequest) (waassdk.DeleteCertificateResponse, error)

	changeCompartmentRequests []waassdk.ChangeCertificateCompartmentRequest
	createRequests            []waassdk.CreateCertificateRequest
	getRequests               []waassdk.GetCertificateRequest
	listRequests              []waassdk.ListCertificatesRequest
	updateRequests            []waassdk.UpdateCertificateRequest
	deleteRequests            []waassdk.DeleteCertificateRequest
}

func (c *fakeCertificateOCIClient) ChangeCertificateCompartment(
	ctx context.Context,
	request waassdk.ChangeCertificateCompartmentRequest,
) (waassdk.ChangeCertificateCompartmentResponse, error) {
	c.changeCompartmentRequests = append(c.changeCompartmentRequests, request)
	if c.changeCompartmentFunc != nil {
		return c.changeCompartmentFunc(ctx, request)
	}
	return waassdk.ChangeCertificateCompartmentResponse{}, nil
}

func (c *fakeCertificateOCIClient) CreateCertificate(ctx context.Context, request waassdk.CreateCertificateRequest) (waassdk.CreateCertificateResponse, error) {
	c.createRequests = append(c.createRequests, request)
	if c.createFunc != nil {
		return c.createFunc(ctx, request)
	}
	return waassdk.CreateCertificateResponse{}, nil
}

func (c *fakeCertificateOCIClient) GetCertificate(ctx context.Context, request waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
	c.getRequests = append(c.getRequests, request)
	if c.getFunc != nil {
		return c.getFunc(ctx, request)
	}
	return waassdk.GetCertificateResponse{}, nil
}

func (c *fakeCertificateOCIClient) ListCertificates(ctx context.Context, request waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error) {
	c.listRequests = append(c.listRequests, request)
	if c.listFunc != nil {
		return c.listFunc(ctx, request)
	}
	return waassdk.ListCertificatesResponse{}, nil
}

func (c *fakeCertificateOCIClient) UpdateCertificate(ctx context.Context, request waassdk.UpdateCertificateRequest) (waassdk.UpdateCertificateResponse, error) {
	c.updateRequests = append(c.updateRequests, request)
	if c.updateFunc != nil {
		return c.updateFunc(ctx, request)
	}
	return waassdk.UpdateCertificateResponse{}, nil
}

func (c *fakeCertificateOCIClient) DeleteCertificate(ctx context.Context, request waassdk.DeleteCertificateRequest) (waassdk.DeleteCertificateResponse, error) {
	c.deleteRequests = append(c.deleteRequests, request)
	if c.deleteFunc != nil {
		return c.deleteFunc(ctx, request)
	}
	return waassdk.DeleteCertificateResponse{}, nil
}

func TestCertificateCreateRecordsStatusRequestIDAndCreateOnlyFingerprint(t *testing.T) {
	resource := testCertificate()
	created := sdkCertificateFromSpec("ocid1.waascertificate.oc1..created", resource.Spec, waassdk.LifecycleStatesCreating)
	client := &fakeCertificateOCIClient{
		listFunc: func(context.Context, waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error) {
			return waassdk.ListCertificatesResponse{}, nil
		},
		createFunc: func(context.Context, waassdk.CreateCertificateRequest) (waassdk.CreateCertificateResponse, error) {
			return waassdk.CreateCertificateResponse{
				Certificate:  created,
				OpcRequestId: common.String("create-request"),
			}, nil
		},
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			return waassdk.GetCertificateResponse{Certificate: created}, nil
		},
	}

	response, err := newCertificateServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() did not report success")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while certificate lifecycle is CREATING")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateCertificate calls = %d, want 1", len(client.createRequests))
	}
	assertCertificateCreateRequest(t, client.createRequests[0], resource.Spec)
	assertCertificateStatusIdentity(t, resource, "ocid1.waascertificate.oc1..created", "create-request")
	assertCertificateAsyncCurrent(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
	assertCertificateFingerprintRecorded(t, resource)
	assertCertificateStatusDoesNotLeakPrivateKey(t, resource)
}

func TestCertificateBindUsesPaginatedListAndDoesNotCreate(t *testing.T) {
	resource := testCertificate()
	bound := sdkCertificateFromSpec("ocid1.waascertificate.oc1..bound", resource.Spec, waassdk.LifecycleStatesActive)
	client := &fakeCertificateOCIClient{
		listFunc: func(_ context.Context, request waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error) {
			if stringValue(request.Page) == "" {
				other := certificateSummary("ocid1.waascertificate.oc1..other", resource.Spec.CompartmentId, "other", waassdk.LifecycleStatesActive)
				return waassdk.ListCertificatesResponse{Items: []waassdk.CertificateSummary{other}, OpcNextPage: common.String("page-2")}, nil
			}
			return waassdk.ListCertificatesResponse{Items: []waassdk.CertificateSummary{certificateSummaryFromCertificate(bound)}}, nil
		},
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			return waassdk.GetCertificateResponse{Certificate: bound}, nil
		},
	}

	response, err := newCertificateServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success without requeue", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateCertificate calls = %d, want 0 when paginated list finds existing certificate", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListCertificates calls = %d, want 2 pages", len(client.listRequests))
	}
	if got, want := stringValue(client.listRequests[1].Page), "page-2"; got != want {
		t.Fatalf("second ListCertificates page = %q, want %q", got, want)
	}
	if got, want := resource.Status.Id, "ocid1.waascertificate.oc1..bound"; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	assertCertificateFingerprintRecorded(t, resource)
}

func TestCertificateNoopReconcileObservesExistingResource(t *testing.T) {
	resource := trackedTestCertificate("ocid1.waascertificate.oc1..existing")
	current := sdkCertificateFromSpec(resource.Status.Id, resource.Spec, waassdk.LifecycleStatesActive)
	client := &fakeCertificateOCIClient{
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			return waassdk.GetCertificateResponse{Certificate: current}, nil
		},
	}

	response, err := newCertificateServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success without requeue", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateCertificate calls = %d, want 0 for no-op reconcile", len(client.updateRequests))
	}
	assertCertificateFingerprintRecorded(t, resource)
}

func TestCertificateMutableUpdateSendsMinimalUpdateBody(t *testing.T) {
	resource := trackedTestCertificate("ocid1.waascertificate.oc1..existing")
	resource.Spec.DisplayName = "renamed-cert"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "waas"}}

	oldSpec := resource.Spec
	oldSpec.DisplayName = "sample-cert"
	oldSpec.FreeformTags = map[string]string{"env": "dev"}
	oldSpec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "old"}}
	current := sdkCertificateFromSpec(resource.Status.Id, oldSpec, waassdk.LifecycleStatesActive)
	updated := sdkCertificateFromSpec(resource.Status.Id, resource.Spec, waassdk.LifecycleStatesActive)
	getCalls := 0
	client := &fakeCertificateOCIClient{
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			getCalls++
			if getCalls == 1 {
				return waassdk.GetCertificateResponse{Certificate: current}, nil
			}
			return waassdk.GetCertificateResponse{Certificate: updated}, nil
		},
		updateFunc: func(context.Context, waassdk.UpdateCertificateRequest) (waassdk.UpdateCertificateResponse, error) {
			return waassdk.UpdateCertificateResponse{
				Certificate:  updated,
				OpcRequestId: common.String("update-request"),
			}, nil
		},
	}

	response, err := newCertificateServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success without requeue", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateCertificate calls = %d, want 1", len(client.updateRequests))
	}
	assertCertificateMutableUpdateRequest(t, client.updateRequests[0], resource)
	if got, want := resource.Status.OsokStatus.OpcRequestID, "update-request"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	assertCertificateFingerprintRecorded(t, resource)
}

func TestCertificateCompartmentDriftUsesChangeCompartment(t *testing.T) {
	resource := trackedTestCertificate("ocid1.waascertificate.oc1..existing")
	currentSpec := resource.Spec
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..moved"
	current := sdkCertificateFromSpec(resource.Status.Id, currentSpec, waassdk.LifecycleStatesActive)
	updateCalled := false
	client := &fakeCertificateOCIClient{
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			return waassdk.GetCertificateResponse{Certificate: current}, nil
		},
		changeCompartmentFunc: func(context.Context, waassdk.ChangeCertificateCompartmentRequest) (waassdk.ChangeCertificateCompartmentResponse, error) {
			return waassdk.ChangeCertificateCompartmentResponse{OpcRequestId: common.String("move-request")}, nil
		},
		updateFunc: func(context.Context, waassdk.UpdateCertificateRequest) (waassdk.UpdateCertificateResponse, error) {
			updateCalled = true
			return waassdk.UpdateCertificateResponse{}, nil
		},
	}

	response, err := newCertificateServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue for compartment move", response)
	}
	if updateCalled {
		t.Fatal("UpdateCertificate called for compartment move")
	}
	if len(client.changeCompartmentRequests) != 1 {
		t.Fatalf("ChangeCertificateCompartment calls = %d, want 1", len(client.changeCompartmentRequests))
	}
	assertCertificateCompartmentMoveRequest(t, client.changeCompartmentRequests[0], resource)
	if got, want := resource.Status.OsokStatus.OpcRequestID, "move-request"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	assertCertificateAsyncCurrent(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
	assertCertificateFingerprintRecorded(t, resource)
}

func TestCertificatePrivateKeyDriftIsRejectedBeforeUpdate(t *testing.T) {
	resource := trackedTestCertificate("ocid1.waascertificate.oc1..existing")
	resource.Spec.PrivateKeyData = "rotated-private-key"
	currentSpec := testCertificate().Spec
	current := sdkCertificateFromSpec(resource.Status.Id, currentSpec, waassdk.LifecycleStatesActive)
	client := &fakeCertificateOCIClient{
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			return waassdk.GetCertificateResponse{Certificate: current}, nil
		},
	}

	response, err := newCertificateServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if !strings.Contains(err.Error(), "create-only fields change") {
		t.Fatalf("CreateOrUpdate() error = %q, want create-only drift message", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateCertificate calls = %d, want 0 after drift rejection", len(client.updateRequests))
	}
}

func TestCertificateDeleteKeepsFinalizerUntilLifecycleIsDeleted(t *testing.T) {
	tests := []struct {
		name        string
		confirmed   waassdk.LifecycleStatesEnum
		wantDeleted bool
	}{
		{
			name:        "deleting",
			confirmed:   waassdk.LifecycleStatesDeleting,
			wantDeleted: false,
		},
		{
			name:        "deleted",
			confirmed:   waassdk.LifecycleStatesDeleted,
			wantDeleted: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCertificateDeleteConfirmationCase(t, tc.confirmed, tc.wantDeleted)
		})
	}
}

func TestCertificateDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := trackedTestCertificate("ocid1.waascertificate.oc1..delete")
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	client := &fakeCertificateOCIClient{
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			return waassdk.GetCertificateResponse{}, authErr
		},
	}

	deleted, err := newCertificateServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous 404")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteCertificate calls = %d, want 0 after auth-shaped confirm-read", len(client.deleteRequests))
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 message", err.Error())
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestCertificateCreateErrorRecordsOpcRequestID(t *testing.T) {
	resource := testCertificate()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	client := &fakeCertificateOCIClient{
		listFunc: func(context.Context, waassdk.ListCertificatesRequest) (waassdk.ListCertificatesResponse, error) {
			return waassdk.ListCertificatesResponse{}, nil
		},
		createFunc: func(context.Context, waassdk.CreateCertificateRequest) (waassdk.CreateCertificateResponse, error) {
			return waassdk.CreateCertificateResponse{}, createErr
		},
	}

	response, err := newCertificateServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func runCertificateDeleteConfirmationCase(
	t *testing.T,
	confirmedState waassdk.LifecycleStatesEnum,
	wantDeleted bool,
) {
	t.Helper()
	resource := trackedTestCertificate("ocid1.waascertificate.oc1..delete")
	active := sdkCertificateFromSpec(resource.Status.Id, resource.Spec, waassdk.LifecycleStatesActive)
	confirmed := sdkCertificateFromSpec(resource.Status.Id, resource.Spec, confirmedState)
	getCalls := 0
	client := &fakeCertificateOCIClient{
		getFunc: func(context.Context, waassdk.GetCertificateRequest) (waassdk.GetCertificateResponse, error) {
			getCalls++
			if getCalls < 3 {
				return waassdk.GetCertificateResponse{Certificate: active}, nil
			}
			return waassdk.GetCertificateResponse{Certificate: confirmed}, nil
		},
		deleteFunc: func(context.Context, waassdk.DeleteCertificateRequest) (waassdk.DeleteCertificateResponse, error) {
			return waassdk.DeleteCertificateResponse{OpcRequestId: common.String("delete-request")}, nil
		},
	}

	deleted, err := newCertificateServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted != wantDeleted {
		t.Fatalf("Delete() deleted = %t, want %t", deleted, wantDeleted)
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteCertificate calls = %d, want 1", len(client.deleteRequests))
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "delete-request"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if !wantDeleted {
		assertCertificateAsyncCurrent(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
		return
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want timestamp after confirmed delete")
	}
}

func assertCertificateCreateRequest(
	t *testing.T,
	request waassdk.CreateCertificateRequest,
	spec waasv1beta1.CertificateSpec,
) {
	t.Helper()
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateCertificate request did not include a deterministic retry token")
	}
	if got, want := stringValue(request.CompartmentId), spec.CompartmentId; got != want {
		t.Fatalf("CreateCertificate compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(request.CertificateData), spec.CertificateData; got != want {
		t.Fatalf("CreateCertificate certificateData = %q, want %q", got, want)
	}
	if got, want := stringValue(request.PrivateKeyData), spec.PrivateKeyData; got != want {
		t.Fatalf("CreateCertificate privateKeyData = %q, want %q", got, want)
	}
	if request.IsTrustVerificationDisabled == nil || !*request.IsTrustVerificationDisabled {
		t.Fatal("CreateCertificate did not preserve explicit trust-verification disablement")
	}
}

func assertCertificateMutableUpdateRequest(
	t *testing.T,
	update waassdk.UpdateCertificateRequest,
	resource *waasv1beta1.Certificate,
) {
	t.Helper()
	if got, want := stringValue(update.CertificateId), resource.Status.Id; got != want {
		t.Fatalf("UpdateCertificate certificateId = %q, want %q", got, want)
	}
	if got, want := stringValue(update.DisplayName), resource.Spec.DisplayName; got != want {
		t.Fatalf("UpdateCertificate displayName = %q, want %q", got, want)
	}
	if got, want := update.FreeformTags["env"], "prod"; got != want {
		t.Fatalf("UpdateCertificate freeformTags[env] = %q, want %q", got, want)
	}
	if got, want := update.DefinedTags["ops"]["owner"], interface{}("waas"); got != want {
		t.Fatalf("UpdateCertificate definedTags[ops][owner] = %#v, want %#v", got, want)
	}
}

func assertCertificateCompartmentMoveRequest(
	t *testing.T,
	request waassdk.ChangeCertificateCompartmentRequest,
	resource *waasv1beta1.Certificate,
) {
	t.Helper()
	if got, want := stringValue(request.CertificateId), resource.Status.Id; got != want {
		t.Fatalf("ChangeCertificateCompartment certificateId = %q, want %q", got, want)
	}
	if got, want := stringValue(request.CompartmentId), resource.Spec.CompartmentId; got != want {
		t.Fatalf("ChangeCertificateCompartment compartmentId = %q, want %q", got, want)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("ChangeCertificateCompartment request did not include a deterministic retry token")
	}
}

func assertCertificateStatusIdentity(t *testing.T, resource *waasv1beta1.Certificate, id string, opcRequestID string) {
	t.Helper()
	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != opcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, opcRequestID)
	}
}

func assertCertificateAsyncCurrent(
	t *testing.T,
	resource *waasv1beta1.Certificate,
	source shared.OSOKAsyncSource,
	phase shared.OSOKAsyncPhase,
	normalizedClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("async current = nil, want source %q phase %q class %q", source, phase, normalizedClass)
	}
	if current.Source != source || current.Phase != phase || current.NormalizedClass != normalizedClass {
		t.Fatalf("async current = %#v, want source %q phase %q class %q", current, source, phase, normalizedClass)
	}
}

func assertCertificateStatusDoesNotLeakPrivateKey(t *testing.T, resource *waasv1beta1.Certificate) {
	t.Helper()
	if strings.Contains(resource.Status.OsokStatus.Message, resource.Spec.PrivateKeyData) {
		t.Fatalf("status message leaked private key material: %q", resource.Status.OsokStatus.Message)
	}
}

func testCertificate() *waasv1beta1.Certificate {
	return &waasv1beta1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-cert",
			Namespace: "default",
		},
		Spec: waasv1beta1.CertificateSpec{
			CompartmentId:               "ocid1.compartment.oc1..example",
			CertificateData:             "certificate-data",
			PrivateKeyData:              "private-key-data",
			DisplayName:                 "sample-cert",
			IsTrustVerificationDisabled: true,
			FreeformTags:                map[string]string{"env": "dev"},
			DefinedTags:                 map[string]shared.MapValue{"ops": {"owner": "team"}},
		},
	}
}

func trackedTestCertificate(id string) *waasv1beta1.Certificate {
	resource := testCertificate()
	resource.Status.Id = id
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.CertificateData = resource.Spec.CertificateData
	resource.Status.IsTrustVerificationDisabled = resource.Spec.IsTrustVerificationDisabled
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	now := metav1.Now()
	resource.Status.OsokStatus.CreatedAt = &now
	recordCertificateCreateOnlyFingerprint(resource)
	return resource
}

func sdkCertificateFromSpec(id string, spec waasv1beta1.CertificateSpec, state waassdk.LifecycleStatesEnum) waassdk.Certificate {
	displayName := spec.DisplayName
	return waassdk.Certificate{
		Id:                          common.String(id),
		CompartmentId:               common.String(spec.CompartmentId),
		DisplayName:                 common.String(displayName),
		SerialNumber:                common.String("1"),
		Version:                     common.Int(1),
		SignatureAlgorithm:          common.String("SHA256-RSA"),
		LifecycleState:              state,
		CertificateData:             common.String(spec.CertificateData),
		IsTrustVerificationDisabled: common.Bool(spec.IsTrustVerificationDisabled),
		FreeformTags:                cloneCertificateStringMap(spec.FreeformTags),
		DefinedTags:                 certificateDefinedTags(spec.DefinedTags),
	}
}

func certificateSummaryFromCertificate(cert waassdk.Certificate) waassdk.CertificateSummary {
	return waassdk.CertificateSummary{
		Id:             cert.Id,
		CompartmentId:  cert.CompartmentId,
		DisplayName:    cert.DisplayName,
		FreeformTags:   cloneCertificateStringMap(cert.FreeformTags),
		DefinedTags:    cloneCertificateDefinedTagMap(cert.DefinedTags),
		LifecycleState: cert.LifecycleState,
		TimeCreated:    cert.TimeCreated,
	}
}

func certificateSummary(id string, compartmentID string, displayName string, state waassdk.LifecycleStatesEnum) waassdk.CertificateSummary {
	return waassdk.CertificateSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
	}
}

func assertCertificateFingerprintRecorded(t *testing.T, resource *waasv1beta1.Certificate) {
	t.Helper()
	fingerprint, ok := certificateRecordedCreateOnlyFingerprint(resource)
	if !ok {
		t.Fatalf("create-only fingerprint missing from status.status.message %q", resource.Status.OsokStatus.Message)
	}
	want, err := certificateCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		t.Fatalf("certificateCreateOnlyFingerprint() error = %v", err)
	}
	if fingerprint != want {
		t.Fatalf("create-only fingerprint = %q, want %q", fingerprint, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
