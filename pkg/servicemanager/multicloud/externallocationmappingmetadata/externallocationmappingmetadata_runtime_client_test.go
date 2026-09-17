/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package externallocationmappingmetadata

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	multicloudsdk "github.com/oracle/oci-go-sdk/v65/multicloud"
	multicloudv1beta1 "github.com/oracle/oci-service-operator/api/multicloud/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testExternalLocationMappingMetadataCompartmentID  = "ocid1.compartment.oc1..example"
	testExternalLocationMappingMetadataSubscriptionID = "ocid1.multicloudsubscription.oc1..example"
	testExternalLocationMappingMetadataProviderError  = "external location mapping metadata provider invalid"
)

type erroringExternalLocationMappingMetadataConfigProvider struct {
	calls int
}

func (p *erroringExternalLocationMappingMetadataConfigProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	p.calls++
	return nil, errors.New(testExternalLocationMappingMetadataProviderError)
}

func (p *erroringExternalLocationMappingMetadataConfigProvider) KeyID() (string, error) {
	p.calls++
	return "", errors.New(testExternalLocationMappingMetadataProviderError)
}

func (p *erroringExternalLocationMappingMetadataConfigProvider) TenancyOCID() (string, error) {
	p.calls++
	return "", errors.New(testExternalLocationMappingMetadataProviderError)
}

func (p *erroringExternalLocationMappingMetadataConfigProvider) UserOCID() (string, error) {
	p.calls++
	return "", errors.New(testExternalLocationMappingMetadataProviderError)
}

func (p *erroringExternalLocationMappingMetadataConfigProvider) KeyFingerprint() (string, error) {
	p.calls++
	return "", errors.New(testExternalLocationMappingMetadataProviderError)
}

func (p *erroringExternalLocationMappingMetadataConfigProvider) Region() (string, error) {
	p.calls++
	return "", errors.New(testExternalLocationMappingMetadataProviderError)
}

func (p *erroringExternalLocationMappingMetadataConfigProvider) AuthType() (common.AuthConfig, error) {
	return common.AuthConfig{}, nil
}

func TestExternalLocationMappingMetadataCreateOrUpdateRequiresCompartmentAnnotation(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	listCalled := false
	client := newTestExternalLocationMappingMetadataClient(func(context.Context, multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error) {
		listCalled = true
		return multicloudsdk.ListExternalLocationMappingMetadataResponse{}, nil
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing compartment annotation error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if listCalled {
		t.Fatal("ListExternalLocationMappingMetadata() should not be called when compartment annotation is missing")
	}
	if !strings.Contains(err.Error(), externalLocationMappingMetadataCompartmentIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %q, want compartment annotation detail", err.Error())
	}
	if got := lastExternalLocationMappingMetadataCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestExternalLocationMappingMetadataCreateOrUpdateObservesMappingFromLaterPage(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Annotations[externalLocationMappingMetadataCompartmentIDAnnotation] = testExternalLocationMappingMetadataCompartmentID
	resource.Annotations[externalLocationMappingMetadataSubscriptionIDAnnotation] = testExternalLocationMappingMetadataSubscriptionID
	resource.Annotations[externalLocationMappingMetadataSubscriptionServiceAnnotation] = "ORACLEDBATAZURE"
	resource.Annotations[externalLocationMappingMetadataCSPRegionAnnotation] = "eastus"
	resource.Annotations[externalLocationMappingMetadataCSPPhysicalAZAnnotation] = "1"
	resource.Annotations[externalLocationMappingMetadataOCIRegionAnnotation] = "us-ashburn-1"

	target := externalLocationMappingMetadataSummary("eastus", "1", "ORACLEDBATAZURE", "us-ashburn-1", "phx-ad-1", "iad-ad-1")
	var requests []multicloudsdk.ListExternalLocationMappingMetadataRequest
	client := newPagedExternalLocationMappingMetadataClient(t, &requests,
		multicloudsdk.ListExternalLocationMappingMetadataResponse{
			ExternalLocationMappingMetadatumSummaryCollection: multicloudsdk.ExternalLocationMappingMetadatumSummaryCollection{
				Items: []multicloudsdk.ExternalLocationMappingMetadatumSummary{
					externalLocationMappingMetadataSummary("westus", "2", "ORACLEDBATAZURE", "us-phoenix-1", "phx-ad-2", "iad-ad-2"),
				},
			},
			OpcRequestId: common.String("opc-list-1"),
			OpcNextPage:  common.String("page-2"),
		},
		multicloudsdk.ListExternalLocationMappingMetadataResponse{
			ExternalLocationMappingMetadatumSummaryCollection: multicloudsdk.ExternalLocationMappingMetadatumSummaryCollection{
				Items: []multicloudsdk.ExternalLocationMappingMetadatumSummary{target},
			},
			OpcRequestId: common.String("opc-list-2"),
		})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireExternalLocationMappingMetadataSuccess(t, response, err)
	requireExternalLocationMappingMetadataPagedRequests(t, requests)
	requireExternalLocationMappingMetadataObservedStatus(t, resource, target)
}

func TestExternalLocationMappingMetadataCreateOrUpdateNoopsForTrackedMapping(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Annotations[externalLocationMappingMetadataCompartmentIDAnnotation] = testExternalLocationMappingMetadataCompartmentID
	resource.Annotations[externalLocationMappingMetadataCSPRegionAnnotation] = "eastus"
	item := externalLocationMappingMetadataSummary("eastus", "1", "ORACLEDBATAZURE", "us-ashburn-1", "phx-ad-1", "iad-ad-1")
	identity := externalLocationMappingMetadataQuery{
		compartmentID: testExternalLocationMappingMetadataCompartmentID,
	}.identityFor(item)
	resource.Status.OsokStatus.Ocid = identity.syntheticID()

	client := newTestExternalLocationMappingMetadataClient(func(context.Context, multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error) {
		return multicloudsdk.ListExternalLocationMappingMetadataResponse{
			ExternalLocationMappingMetadatumSummaryCollection: multicloudsdk.ExternalLocationMappingMetadatumSummaryCollection{
				Items: []multicloudsdk.ExternalLocationMappingMetadatumSummary{item},
			},
			OpcRequestId: common.String("opc-list"),
		}, nil
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op observation", response)
	}
	if got := resource.Status.OsokStatus.Ocid; got != identity.syntheticID() {
		t.Fatalf("status.status.ocid = %q, want existing synthetic id", got)
	}
	if got := lastExternalLocationMappingMetadataCondition(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestExternalLocationMappingMetadataCreateOrUpdateRejectsIdentityDrift(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Annotations[externalLocationMappingMetadataCompartmentIDAnnotation] = testExternalLocationMappingMetadataCompartmentID
	resource.Annotations[externalLocationMappingMetadataCSPRegionAnnotation] = "eastus"
	oldItem := externalLocationMappingMetadataSummary("westus", "2", "ORACLEDBATAZURE", "us-phoenix-1", "phx-ad-2", "iad-ad-2")
	resource.Status.OsokStatus.Ocid = externalLocationMappingMetadataQuery{
		compartmentID: testExternalLocationMappingMetadataCompartmentID,
	}.identityFor(oldItem).syntheticID()

	newItem := externalLocationMappingMetadataSummary("eastus", "1", "ORACLEDBATAZURE", "us-ashburn-1", "phx-ad-1", "iad-ad-1")
	client := newTestExternalLocationMappingMetadataClient(func(context.Context, multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error) {
		return multicloudsdk.ListExternalLocationMappingMetadataResponse{
			ExternalLocationMappingMetadatumSummaryCollection: multicloudsdk.ExternalLocationMappingMetadatumSummaryCollection{
				Items: []multicloudsdk.ExternalLocationMappingMetadatumSummary{newItem},
			},
		}, nil
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want identity drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "identity drift") {
		t.Fatalf("CreateOrUpdate() error = %q, want identity drift detail", err.Error())
	}
	if got := lastExternalLocationMappingMetadataCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestExternalLocationMappingMetadataCreateOrUpdateRejectsAmbiguousList(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Annotations[externalLocationMappingMetadataCompartmentIDAnnotation] = testExternalLocationMappingMetadataCompartmentID
	client := newTestExternalLocationMappingMetadataClient(func(context.Context, multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error) {
		return multicloudsdk.ListExternalLocationMappingMetadataResponse{
			ExternalLocationMappingMetadatumSummaryCollection: multicloudsdk.ExternalLocationMappingMetadatumSummaryCollection{
				Items: []multicloudsdk.ExternalLocationMappingMetadatumSummary{
					externalLocationMappingMetadataSummary("eastus", "1", "ORACLEDBATAZURE", "us-ashburn-1", "phx-ad-1", "iad-ad-1"),
					externalLocationMappingMetadataSummary("westus", "2", "ORACLEDBATAZURE", "us-phoenix-1", "phx-ad-2", "iad-ad-2"),
				},
			},
		}, nil
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want ambiguous list error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "returned 2 mappings") {
		t.Fatalf("CreateOrUpdate() error = %q, want ambiguous mapping count", err.Error())
	}
}

func TestExternalLocationMappingMetadataCreateOrUpdateRejectsRepeatedPageToken(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Annotations[externalLocationMappingMetadataCompartmentIDAnnotation] = testExternalLocationMappingMetadataCompartmentID
	var requests []multicloudsdk.ListExternalLocationMappingMetadataRequest
	client := newPagedExternalLocationMappingMetadataClient(t, &requests,
		multicloudsdk.ListExternalLocationMappingMetadataResponse{
			OpcNextPage: common.String("page-2"),
		},
		multicloudsdk.ListExternalLocationMappingMetadataResponse{
			OpcNextPage: common.String("page-2"),
		})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want repeated page token error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("CreateOrUpdate() error = %q, want repeated page token detail", err.Error())
	}
	if len(requests) != 2 {
		t.Fatalf("ListExternalLocationMappingMetadata() calls = %d, want 2", len(requests))
	}
	requireStringPtr(t, "second request page", requests[1].Page, "page-2")
	if got := lastExternalLocationMappingMetadataCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestExternalLocationMappingMetadataCreateOrUpdatePreservesGeneratedOCIInitError(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Annotations[externalLocationMappingMetadataCompartmentIDAnnotation] = testExternalLocationMappingMetadataCompartmentID
	provider := &erroringExternalLocationMappingMetadataConfigProvider{}
	client := newExternalLocationMappingMetadataServiceClient(&ExternalLocationMappingMetadataServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: logr.Discard()},
	})
	callsAfterInit := provider.calls

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assertExternalLocationMappingMetadataInitError(t, err)
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
	assertExternalLocationMappingMetadataProviderCalls(t, provider, callsAfterInit)
}

func TestExternalLocationMappingMetadataCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Annotations[externalLocationMappingMetadataCompartmentIDAnnotation] = testExternalLocationMappingMetadataCompartmentID
	resource.Annotations[externalLocationMappingMetadataCSPRegionAnnotation] = "eastus"
	client := newTestExternalLocationMappingMetadataClient(func(context.Context, multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error) {
		return multicloudsdk.ListExternalLocationMappingMetadataResponse{}, errortest.NewServiceError(500, "InternalError", "list failed")
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI list error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if got := lastExternalLocationMappingMetadataCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestExternalLocationMappingMetadataDeletePreservesGeneratedOCIInitError(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(externalLocationMappingMetadataSyntheticIDPrefix + externalLocationMappingMetadataSyntheticIDVersion + strings.Repeat("a", 64))
	provider := &erroringExternalLocationMappingMetadataConfigProvider{}
	client := newExternalLocationMappingMetadataServiceClient(&ExternalLocationMappingMetadataServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: logr.Discard()},
	})
	callsAfterInit := provider.calls

	deleted, err := client.Delete(context.Background(), resource)
	assertExternalLocationMappingMetadataInitError(t, err)
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set; delete should stop before read-only finalizer release")
	}
	assertExternalLocationMappingMetadataProviderCalls(t, provider, callsAfterInit)
}

func TestExternalLocationMappingMetadataDeleteMarksReadOnlyMetadataDeleted(t *testing.T) {
	t.Parallel()

	resource := makeExternalLocationMappingMetadataResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(externalLocationMappingMetadataSyntheticIDPrefix + externalLocationMappingMetadataSyntheticIDVersion + strings.Repeat("a", 64))
	client := newTestExternalLocationMappingMetadataClient(func(context.Context, multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error) {
		t.Fatal("ListExternalLocationMappingMetadata() should not be called during delete")
		return multicloudsdk.ListExternalLocationMappingMetadataResponse{}, nil
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for local read-only metadata finalizer release")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	if got := lastExternalLocationMappingMetadataCondition(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "read-only OCI metadata") {
		t.Fatalf("status.status.message = %q, want read-only metadata delete explanation", resource.Status.OsokStatus.Message)
	}
}

func assertExternalLocationMappingMetadataInitError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("operation error = nil, want OCI client initialization error")
	}
	if !strings.Contains(err.Error(), "initialize ExternalLocationMappingMetadata OCI client") {
		t.Fatalf("operation error = %v, want ExternalLocationMappingMetadata OCI client initialization failure", err)
	}
	if !strings.Contains(err.Error(), testExternalLocationMappingMetadataProviderError) {
		t.Fatalf("operation error = %v, want provider failure detail", err)
	}
}

func assertExternalLocationMappingMetadataProviderCalls(
	t *testing.T,
	provider *erroringExternalLocationMappingMetadataConfigProvider,
	want int,
) {
	t.Helper()
	if provider.calls != want {
		t.Fatalf("provider calls after operation = %d, want %d; runtime wrapper should stop at InitError", provider.calls, want)
	}
}

func newTestExternalLocationMappingMetadataClient(list listExternalLocationMappingMetadataFunc) externalLocationMappingMetadataRuntimeClient {
	return externalLocationMappingMetadataRuntimeClient{list: list}
}

func newPagedExternalLocationMappingMetadataClient(
	t *testing.T,
	requests *[]multicloudsdk.ListExternalLocationMappingMetadataRequest,
	responses ...multicloudsdk.ListExternalLocationMappingMetadataResponse,
) externalLocationMappingMetadataRuntimeClient {
	t.Helper()
	return newTestExternalLocationMappingMetadataClient(func(_ context.Context, req multicloudsdk.ListExternalLocationMappingMetadataRequest) (multicloudsdk.ListExternalLocationMappingMetadataResponse, error) {
		*requests = append(*requests, req)
		call := len(*requests) - 1
		if call >= len(responses) {
			t.Fatalf("ListExternalLocationMappingMetadata() call %d exceeded configured responses", call+1)
		}
		return responses[call], nil
	})
}

func requireExternalLocationMappingMetadataSuccess(
	t *testing.T,
	response servicemanager.OSOKResponse,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observation", response)
	}
}

func requireExternalLocationMappingMetadataPagedRequests(
	t *testing.T,
	requests []multicloudsdk.ListExternalLocationMappingMetadataRequest,
) {
	t.Helper()
	if len(requests) != 2 {
		t.Fatalf("ListExternalLocationMappingMetadata() calls = %d, want 2", len(requests))
	}
	requireStringPtr(t, "first request compartmentId", requests[0].CompartmentId, testExternalLocationMappingMetadataCompartmentID)
	requireStringPtr(t, "first request subscriptionId", requests[0].SubscriptionId, testExternalLocationMappingMetadataSubscriptionID)
	requireSingleSubscriptionServiceName(t, requests[0].SubscriptionServiceName)
	requireStringPtr(t, "second request page", requests[1].Page, "page-2")
}

func requireSingleSubscriptionServiceName(t *testing.T, got []multicloudsdk.SubscriptionTypeEnum) {
	t.Helper()
	if len(got) != 1 || got[0] != multicloudsdk.SubscriptionTypeOracledbatazure {
		t.Fatalf("first request SubscriptionServiceName = %#v, want ORACLEDBATAZURE", got)
	}
}

func requireExternalLocationMappingMetadataObservedStatus(
	t *testing.T,
	resource *multicloudv1beta1.ExternalLocationMappingMetadata,
	target multicloudsdk.ExternalLocationMappingMetadatumSummary,
) {
	t.Helper()
	wantID := externalLocationMappingMetadataQuery{
		compartmentID:             testExternalLocationMappingMetadataCompartmentID,
		subscriptionID:            testExternalLocationMappingMetadataSubscriptionID,
		subscriptionServiceLabels: []string{"ORACLEDBATAZURE"},
	}.identityFor(target).syntheticID()
	if got := resource.Status.OsokStatus.Ocid; got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-list-2" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-list-2", got)
	}
	if got := lastExternalLocationMappingMetadataCondition(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "us-ashburn-1") {
		t.Fatalf("status.status.message = %q, want observed OCI region", resource.Status.OsokStatus.Message)
	}
	if resource.Status.OciRegion != "us-ashburn-1" || resource.Status.OciPhysicalAd != "phx-ad-1" {
		t.Fatalf("projected ExternalLocationMappingMetadata status = %+v", resource.Status)
	}
}

func makeExternalLocationMappingMetadataResource() *multicloudv1beta1.ExternalLocationMappingMetadata {
	return &multicloudv1beta1.ExternalLocationMappingMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "mapping-metadata",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	}
}

func externalLocationMappingMetadataSummary(
	cspRegion string,
	cspPhysicalAZ string,
	serviceName string,
	ociRegion string,
	ociPhysicalAD string,
	ociLogicalAD string,
) multicloudsdk.ExternalLocationMappingMetadatumSummary {
	return multicloudsdk.ExternalLocationMappingMetadatumSummary{
		ExternalLocation: &multicloudsdk.ExternalLocation{
			CspRegion:                common.String(cspRegion),
			CspRegionDisplayName:     common.String(cspRegion),
			CspPhysicalAz:            common.String(cspPhysicalAZ),
			CspPhysicalAzDisplayName: common.String(cspPhysicalAZ),
			ServiceName:              multicloudsdk.SubscriptionTypeEnum(serviceName),
		},
		OciRegion:     common.String(ociRegion),
		OciPhysicalAd: common.String(ociPhysicalAD),
		OciLogicalAd:  common.String(ociLogicalAD),
		FreeformTags:  map[string]string{},
		DefinedTags:   map[string]map[string]interface{}{},
	}
}

func lastExternalLocationMappingMetadataCondition(resource *multicloudv1beta1.ExternalLocationMappingMetadata) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
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
