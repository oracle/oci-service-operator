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
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	certificateTestLoadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	certificateTestName           = "example_certificate"
	certificateTestPublicCert     = "-----BEGIN CERTIFICATE-----\npublic\n-----END CERTIFICATE-----"
	certificateTestCACert         = "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----"
	certificateTestPrivateKey     = "-----BEGIN RSA PRIVATE KEY-----\nprivate\n-----END RSA PRIVATE KEY-----"
)

type fakeCertificateOCIClient struct {
	createRequests         []loadbalancersdk.CreateCertificateRequest
	listRequests           []loadbalancersdk.ListCertificatesRequest
	deleteRequests         []loadbalancersdk.DeleteCertificateRequest
	getWorkRequestRequests []loadbalancersdk.GetWorkRequestRequest

	createFn         func(context.Context, loadbalancersdk.CreateCertificateRequest) (loadbalancersdk.CreateCertificateResponse, error)
	listFn           func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error)
	deleteFn         func(context.Context, loadbalancersdk.DeleteCertificateRequest) (loadbalancersdk.DeleteCertificateResponse, error)
	getWorkRequestFn func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error)
}

func (f *fakeCertificateOCIClient) CreateCertificate(ctx context.Context, request loadbalancersdk.CreateCertificateRequest) (loadbalancersdk.CreateCertificateResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return loadbalancersdk.CreateCertificateResponse{OpcWorkRequestId: common.String("wr-create")}, nil
}

func (f *fakeCertificateOCIClient) ListCertificates(ctx context.Context, request loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return loadbalancersdk.ListCertificatesResponse{}, nil
}

func (f *fakeCertificateOCIClient) DeleteCertificate(ctx context.Context, request loadbalancersdk.DeleteCertificateRequest) (loadbalancersdk.DeleteCertificateResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return loadbalancersdk.DeleteCertificateResponse{OpcWorkRequestId: common.String("wr-delete")}, nil
}

func (f *fakeCertificateOCIClient) GetWorkRequest(ctx context.Context, request loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	return loadbalancersdk.GetWorkRequestResponse{}, nil
}

func TestCertificateRuntimeSemanticsEncodeWorkRequestContract(t *testing.T) {
	semantics := newCertificateRuntimeSemantics()

	if semantics.FormalService != "loadbalancer" || semantics.FormalSlug != "certificate" {
		t.Fatalf("formal identity = %s/%s, want loadbalancer/certificate", semantics.FormalService, semantics.FormalSlug)
	}
	if semantics.Async == nil {
		t.Fatal("async semantics = nil, want workrequest")
	}
	if semantics.Async.Strategy != "workrequest" || semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", semantics.Async)
	}
	if semantics.Async.WorkRequest == nil || strings.Join(semantics.Async.WorkRequest.Phases, ",") != "create,delete" {
		t.Fatalf("work request phases = %#v, want create/delete", semantics.Async.WorkRequest)
	}
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", semantics.FinalizerPolicy)
	}
	if semantics.List == nil || strings.Join(semantics.List.MatchFields, ",") != "loadBalancerId,certificateName" {
		t.Fatalf("list match fields = %#v, want loadBalancerId/certificateName", semantics.List)
	}
	if len(semantics.Unsupported) != 0 {
		t.Fatalf("unsupported semantics = %#v, want none", semantics.Unsupported)
	}
}

func TestCertificateCreateOrUpdateCreatesWithAnnotatedLoadBalancer(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		createFn: func(_ context.Context, request loadbalancersdk.CreateCertificateRequest) (loadbalancersdk.CreateCertificateResponse, error) {
			if got := stringPtrValue(request.LoadBalancerId); got != certificateTestLoadBalancerID {
				t.Fatalf("create loadBalancerId = %q, want %q", got, certificateTestLoadBalancerID)
			}
			if got := stringPtrValue(request.CertificateName); got != certificateTestName {
				t.Fatalf("create certificateName = %q, want %q", got, certificateTestName)
			}
			if got := stringPtrValue(request.PublicCertificate); got != certificateTestPublicCert {
				t.Fatalf("create publicCertificate = %q, want %q", got, certificateTestPublicCert)
			}
			if got := stringPtrValue(request.CaCertificate); got != certificateTestCACert {
				t.Fatalf("create caCertificate = %q, want %q", got, certificateTestCACert)
			}
			if got := stringPtrValue(request.PrivateKey); got != certificateTestPrivateKey {
				t.Fatalf("create privateKey = %q, want configured private key", got)
			}
			return loadbalancersdk.CreateCertificateResponse{
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want 1 before create", len(fake.listRequests))
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("opc request ID = %q, want opc-create", got)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	requireCondition(t, resource, shared.Provisioning, v1.ConditionTrue)
	if got := string(resource.Status.OsokStatus.Ocid); !strings.HasPrefix(got, certificateSyntheticIDPrefix) {
		t.Fatalf("status ocid = %q, want synthetic certificate id", got)
	}
}

func TestCertificateCreateOrUpdateBindsExistingFromList(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{
				Items: []loadbalancersdk.Certificate{sdkCertificate(certificateTestName, certificateTestPublicCert, certificateTestCACert)},
			}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful observe path", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind path", len(fake.createRequests))
	}
	if resource.Status.CertificateName != certificateTestName {
		t.Fatalf("status.certificateName = %q, want %q", resource.Status.CertificateName, certificateTestName)
	}
	if resource.Status.PublicCertificate != certificateTestPublicCert {
		t.Fatalf("status.publicCertificate = %q, want projected certificate", resource.Status.PublicCertificate)
	}
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
	if got := string(resource.Status.OsokStatus.Ocid); !strings.HasPrefix(got, certificateSyntheticIDPrefix) {
		t.Fatalf("status ocid = %q, want synthetic certificate id", got)
	}
}

func TestCertificateCreateWorkRequestPendingRequeues(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-create", "CreateCertificate", loadbalancersdk.WorkRequestLifecycleStateInProgress), nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseCreate,
		WorkRequestID:    "wr-create",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "CreateCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want pending requeue", response)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	requireCondition(t, resource, shared.Provisioning, v1.ConditionTrue)
	if len(fake.getWorkRequestRequests) != 1 {
		t.Fatalf("get work request calls = %d, want 1", len(fake.getWorkRequestRequests))
	}
}

func TestCertificateCreateWorkRequestSucceededProjectsActiveAfterReadback(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-create", "CreateCertificate", loadbalancersdk.WorkRequestLifecycleStateSucceeded), nil
		},
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{
				Items: []loadbalancersdk.Certificate{sdkCertificate(certificateTestName, certificateTestPublicCert, certificateTestCACert)},
			}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseCreate,
		WorkRequestID:    "wr-create",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "CreateCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want converged active", response)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
	requireCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestCertificateCreateWorkRequestSucceededListMissKeepsRequeue(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-create", "CreateCertificate", loadbalancersdk.WorkRequestLifecycleStateSucceeded), nil
		},
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseCreate,
		WorkRequestID:    "wr-create",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "CreateCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while list misses", response)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.Async.Current.RawStatus; got != string(loadbalancersdk.WorkRequestLifecycleStateSucceeded) {
		t.Fatalf("async raw status = %q, want SUCCEEDED breadcrumb", got)
	}
	requireCondition(t, resource, shared.Provisioning, v1.ConditionTrue)
	if resource.Status.OsokStatus.Reason == string(shared.Active) {
		t.Fatal("status reason = Active, want pending readback state")
	}
}

func TestCertificateCreateWorkRequestFailedMarksFailed(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-create", "CreateCertificate", loadbalancersdk.WorkRequestLifecycleStateFailed), nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseCreate,
		WorkRequestID:    "wr-create",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "CreateCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want failed work request error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassFailed)
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestCertificateCreateOrUpdateRejectsCreateOnlyDriftBeforeMutation(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{
				Items: []loadbalancersdk.Certificate{sdkCertificate(certificateTestName, "old-public", certificateTestCACert)},
			}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Spec.PublicCertificate = "new-public"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if !strings.Contains(err.Error(), "create-only") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want no mutation after drift", len(fake.createRequests))
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestCertificateCreateOrUpdateRequiresParentAnnotation(t *testing.T) {
	client := newCertificateServiceClientWithOCIClient(&fakeCertificateOCIClient{}, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Annotations = nil

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing annotation error")
	}
	if !strings.Contains(err.Error(), CertificateLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want annotation name", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestCertificateCreateOrUpdateListAuthShaped404IsFatal(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want auth-shaped list error")
	}
	if classification := errorutil.ClassifyDeleteError(err); classification.ErrorCode != errorutil.NotAuthorizedOrNotFound {
		t.Fatalf("CreateOrUpdate() error code = %q, want %s", classification.ErrorCode, errorutil.NotAuthorizedOrNotFound)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want no create after auth-shaped list error", len(fake.createRequests))
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestCertificateDeleteStartsWorkRequestAndRetainsFinalizer(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{
				Items: []loadbalancersdk.Certificate{sdkCertificate(certificateTestName, certificateTestPublicCert, certificateTestCACert)},
			}, nil
		},
		deleteFn: func(context.Context, loadbalancersdk.DeleteCertificateRequest) (loadbalancersdk.DeleteCertificateResponse, error) {
			return loadbalancersdk.DeleteCertificateResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while work request is pending")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if got := stringPtrValue(fake.deleteRequests[0].LoadBalancerId); got != certificateTestLoadBalancerID {
		t.Fatalf("delete loadBalancerId = %q, want %q", got, certificateTestLoadBalancerID)
	}
	if got := stringPtrValue(fake.deleteRequests[0].CertificateName); got != certificateTestName {
		t.Fatalf("delete certificateName = %q, want %q", got, certificateTestName)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("opc request ID = %q, want opc-delete", got)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
}

func TestCertificateDeleteAuthShaped404FromDeleteIsFatal(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{
				Items: []loadbalancersdk.Certificate{sdkCertificate(certificateTestName, certificateTestPublicCert, certificateTestCACert)},
			}, nil
		},
		deleteFn: func(context.Context, loadbalancersdk.DeleteCertificateRequest) (loadbalancersdk.DeleteCertificateResponse, error) {
			return loadbalancersdk.DeleteCertificateResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped delete error")
	}
	if classification := errorutil.ClassifyDeleteError(err); classification.ErrorCode != errorutil.NotAuthorizedOrNotFound {
		t.Fatalf("Delete() error code = %q, want %s", classification.ErrorCode, errorutil.NotAuthorizedOrNotFound)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped delete error")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	requireCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestCertificateDeleteWithCreateWorkRequestPendingRetainsFinalizer(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-create", "CreateCertificate", loadbalancersdk.WorkRequestLifecycleStateInProgress), nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseCreate,
		WorkRequestID:    "wr-create",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "CreateCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while create is pending")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want none before create completes", len(fake.deleteRequests))
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	requireCondition(t, resource, shared.Provisioning, v1.ConditionTrue)
}

func TestCertificateDeleteWithCreateWorkRequestSucceededStartsDeleteAfterReadback(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-create", "CreateCertificate", loadbalancersdk.WorkRequestLifecycleStateSucceeded), nil
		},
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{
				Items: []loadbalancersdk.Certificate{sdkCertificate(certificateTestName, certificateTestPublicCert, certificateTestCACert)},
			}, nil
		},
		deleteFn: func(context.Context, loadbalancersdk.DeleteCertificateRequest) (loadbalancersdk.DeleteCertificateResponse, error) {
			return loadbalancersdk.DeleteCertificateResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseCreate,
		WorkRequestID:    "wr-create",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "CreateCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while delete work request is pending")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("list requests = %d, want one create readback", len(fake.listRequests))
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want one delete after create readback", len(fake.deleteRequests))
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
}

func TestCertificateDeleteWorkRequestSucceededReleasesAfterListMiss(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-delete", "DeleteCertificate", loadbalancersdk.WorkRequestLifecycleStateSucceeded), nil
		},
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    "wr-delete",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "DeleteCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after successful work request and missing list item")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want set")
	}
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
}

func TestCertificateDeleteWorkRequestSucceededRetainsFinalizerWhileStillListed(t *testing.T) {
	fake := &fakeCertificateOCIClient{
		getWorkRequestFn: func(context.Context, loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
			return workRequestResponse("wr-delete", "DeleteCertificate", loadbalancersdk.WorkRequestLifecycleStateSucceeded), nil
		},
		listFn: func(context.Context, loadbalancersdk.ListCertificatesRequest) (loadbalancersdk.ListCertificatesResponse, error) {
			return loadbalancersdk.ListCertificatesResponse{
				Items: []loadbalancersdk.Certificate{sdkCertificate(certificateTestName, certificateTestPublicCert, certificateTestCACert)},
			}, nil
		},
	}
	client := newCertificateServiceClientWithOCIClient(fake, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    "wr-delete",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawStatus:        string(loadbalancersdk.WorkRequestLifecycleStateAccepted),
		RawOperationType: "DeleteCertificate",
		UpdatedAt:        &metav1.Time{},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while list still shows certificate")
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassSucceeded)
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
}

func TestCertificateDeleteReleasesNeverBoundResourceWithoutAnnotation(t *testing.T) {
	client := newCertificateServiceClientWithOCIClient(&fakeCertificateOCIClient{}, loggerutil.OSOKLogger{}, nil)
	resource := newCertificateResource()
	resource.Annotations = nil

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for never-bound resource")
	}
	requireCondition(t, resource, shared.Terminating, v1.ConditionTrue)
}

func newCertificateResource() *loadbalancerv1beta1.Certificate {
	return &loadbalancerv1beta1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "default",
			Annotations: map[string]string{
				CertificateLoadBalancerIDAnnotation: certificateTestLoadBalancerID,
			},
		},
		Spec: loadbalancerv1beta1.CertificateSpec{
			CertificateName:   certificateTestName,
			PrivateKey:        certificateTestPrivateKey,
			PublicCertificate: certificateTestPublicCert,
			CaCertificate:     certificateTestCACert,
		},
	}
}

func sdkCertificate(name string, publicCertificate string, caCertificate string) loadbalancersdk.Certificate {
	return loadbalancersdk.Certificate{
		CertificateName:   common.String(name),
		PublicCertificate: common.String(publicCertificate),
		CaCertificate:     common.String(caCertificate),
	}
}

func workRequestResponse(id string, operationType string, state loadbalancersdk.WorkRequestLifecycleStateEnum) loadbalancersdk.GetWorkRequestResponse {
	return loadbalancersdk.GetWorkRequestResponse{
		WorkRequest: loadbalancersdk.WorkRequest{
			Id:             common.String(id),
			Type:           common.String(operationType),
			LifecycleState: state,
			Message:        common.String(operationType + " " + string(state)),
		},
	}
}

func requireCurrentAsync(
	t *testing.T,
	resource *loadbalancerv1beta1.Certificate,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want work request operation")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("async source = %q, want workrequest", current.Source)
	}
	if current.Phase != phase || current.WorkRequestID != workRequestID || current.NormalizedClass != class {
		t.Fatalf("async current = %#v, want phase=%s workRequestID=%s class=%s", current, phase, workRequestID, class)
	}
}

func requireCondition(
	t *testing.T,
	resource *loadbalancerv1beta1.Certificate,
	conditionType shared.OSOKConditionType,
	status v1.ConditionStatus,
) {
	t.Helper()
	for i := len(resource.Status.OsokStatus.Conditions) - 1; i >= 0; i-- {
		condition := resource.Status.OsokStatus.Conditions[i]
		if condition.Type != conditionType {
			continue
		}
		if condition.Status != status {
			t.Fatalf("%s condition status = %s, want %s", conditionType, condition.Status, status)
		}
		return
	}
	t.Fatalf("%s condition not found in %#v", conditionType, resource.Status.OsokStatus.Conditions)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ certificateRuntimeOCIClient = (*fakeCertificateOCIClient)(nil)
