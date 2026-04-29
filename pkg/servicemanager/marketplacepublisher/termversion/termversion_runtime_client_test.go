/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package termversion

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testTermID          = "ocid1.term.oc1..term"
	testTermVersionID   = "ocid1.termversion.oc1..version"
	testCompartmentID   = "ocid1.compartment.oc1..compartment"
	testTermDisplayName = "publisher-term-v1"
	testTermContent     = "terms and conditions"
)

func TestTermVersionRuntimeSemanticsDocumentResourceLocalInputs(t *testing.T) {
	semantics := newTermVersionRuntimeSemantics()

	requireTermVersionSemanticsTarget(t, semantics)
	requireTermVersionSemanticsMutation(t, semantics)
	requireTermVersionSemanticsUnsupported(t, semantics)
}

func requireTermVersionSemanticsTarget(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	if semantics.FormalService != "marketplacepublisher" || semantics.FormalSlug != "termversion" {
		t.Fatalf("semantics formal target = %s/%s, want marketplacepublisher/termversion", semantics.FormalService, semantics.FormalSlug)
	}
	if semantics.Async == nil ||
		semantics.Async.Strategy != "lifecycle" ||
		semantics.Async.Runtime != "handwritten" ||
		semantics.Async.FormalClassification != "lifecycle" {
		t.Fatalf("semantics async = %#v, want handwritten lifecycle", semantics.Async)
	}
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", semantics.FinalizerPolicy)
	}
}

func requireTermVersionSemanticsMutation(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	requireTermVersionStrings(t, "mutable fields", semantics.Mutation.Mutable, "displayName", "definedTags", "freeformTags", "contentSecret", "contentSecretKey")
	requireTermVersionStrings(t, "forceNew fields", semantics.Mutation.ForceNew, "termId")
}

func requireTermVersionSemanticsUnsupported(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()
	if len(semantics.Unsupported) != 1 ||
		!strings.Contains(semantics.Unsupported[0].StopCondition, termVersionTermIDAnnotation) ||
		!strings.Contains(semantics.Unsupported[0].StopCondition, termVersionContentSecretAnnotation) {
		t.Fatalf("unsupported semantics = %#v, want annotation stop condition", semantics.Unsupported)
	}
}

func TestTermVersionCreateRequiresTermAnnotationBeforeOCICalls(t *testing.T) {
	resource := newTermVersionResource()
	resource.Annotations = map[string]string{
		termVersionContentSecretAnnotation: "term-content",
	}
	fake := &fakeTermVersionOCIClient{}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte("terms")},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate error = nil, want missing term annotation error")
	}
	if !strings.Contains(err.Error(), termVersionTermIDAnnotation) {
		t.Fatalf("CreateOrUpdate error = %q, want term annotation name", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	if len(fake.createRequests)+len(fake.listRequests)+len(fake.getRequests) != 0 {
		t.Fatalf("OCI calls create/list/get = %d/%d/%d, want none", len(fake.createRequests), len(fake.listRequests), len(fake.getRequests))
	}
	requireTermVersionCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestTermVersionCreateUploadsSecretAndRecordsStatus(t *testing.T) {
	resource := newTermVersionResource()
	resource.UID = types.UID("uid-termversion")
	resource.Spec.DisplayName = ""
	fake := &fakeTermVersionOCIClient{
		listResults: []termVersionListResult{{response: marketplacepublishersdk.ListTermVersionsResponse{}}},
		createResults: []termVersionCreateResult{{
			response: marketplacepublishersdk.CreateTermVersionResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, "termversion-sample", marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-create"),
			},
		}},
		getResults: []termVersionGetResult{{
			response: marketplacepublishersdk.GetTermVersionResponse{
				TermVersion: sdkTermVersion(testTermVersionID, testTermID, "termversion-sample", marketplacepublishersdk.TermVersionStatusAvailable),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte(testTermContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateTermVersion calls = %d, want 1", len(fake.createRequests))
	}
	createRequest := fake.createRequests[0]
	if got := termVersionString(createRequest.TermId); got != testTermID {
		t.Fatalf("create termId = %q, want %q", got, testTermID)
	}
	if got := termVersionString(createRequest.DisplayName); got != "termversion-sample" {
		t.Fatalf("create displayName = %q, want metadata.name fallback", got)
	}
	if got := termVersionString(createRequest.OpcRetryToken); got != "uid-termversion" {
		t.Fatalf("create opcRetryToken = %q, want uid-derived token", got)
	}
	if got := readTermVersionCreateContent(t, createRequest.CreateTermVersionContent); got != testTermContent {
		t.Fatalf("create content = %q, want secret content", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, "termversion-sample")
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, testTermContent)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want create request id", got)
	}
	requireTermVersionCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestTermVersionCreateWithTagsAppliesRemainingUpdateDrift(t *testing.T) {
	resource := newTermVersionResource()
	resource.Spec.FreeformTags = map[string]string{"owner": "osok"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"key": "value"}}
	fake := &fakeTermVersionOCIClient{
		listResults: []termVersionListResult{{response: marketplacepublishersdk.ListTermVersionsResponse{}}},
		createResults: []termVersionCreateResult{{
			response: marketplacepublishersdk.CreateTermVersionResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-create"),
			},
		}},
		getResults: []termVersionGetResult{
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
				},
			},
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersionWithTags(
						testTermVersionID,
						testTermID,
						testTermDisplayName,
						map[string]string{"owner": "osok"},
						map[string]map[string]interface{}{"ns": {"key": "value"}},
					),
				},
			},
		},
		updateResults: []termVersionUpdateResult{{
			response: marketplacepublishersdk.UpdateTermVersionResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-tags-after-create"),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte(testTermContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateTermVersion calls = %d, want 1", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateTermVersion calls = %d, want 1 for post-create tags", len(fake.updateRequests))
	}
	requireTermVersionUpdateTags(t, fake.updateRequests[0], map[string]string{"owner": "osok"}, map[string]map[string]interface{}{"ns": {"key": "value"}})
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-tags-after-create" {
		t.Fatalf("status.opcRequestId = %q, want post-create tag update request id", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, testTermDisplayName)
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, testTermContent)
}

func TestTermVersionContentFingerprintSurvivesFailedReadback(t *testing.T) {
	resource := newTermVersionResource()
	readErr := errortest.NewServiceError(500, "InternalError", "temporary readback failure")
	fake := &fakeTermVersionOCIClient{
		listResults: []termVersionListResult{{response: marketplacepublishersdk.ListTermVersionsResponse{}}},
		createResults: []termVersionCreateResult{{
			response: marketplacepublishersdk.CreateTermVersionResponse{
				TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
			},
		}},
		getResults: []termVersionGetResult{
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
				},
			},
			{err: readErr},
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
				},
			},
		},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte(testTermContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("initial CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("initial response.IsSuccessful = false, want true")
	}
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, testTermContent)

	response, err = client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err == nil {
		t.Fatal("failed readback CreateOrUpdate error = nil, want error")
	}
	if response.IsSuccessful {
		t.Fatalf("failed readback response.IsSuccessful = true, want false")
	}
	requireTermVersionCondition(t, resource, shared.Failed, v1.ConditionFalse)
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, testTermContent)

	response, err = client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("unchanged no-op CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("unchanged no-op response.IsSuccessful = false, want true")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateTermVersion calls = %d, want only initial create", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateTermVersion calls = %d, want 0 after unchanged no-op", len(fake.updateRequests))
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, testTermDisplayName)
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, testTermContent)
}

func TestTermVersionBindExistingViaPaginatedListBeforeCreate(t *testing.T) {
	resource := newTermVersionResource()
	fake := paginatedBindTermVersionFake()
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte(testTermContent)},
		},
	})

	requireTermVersionInitialBind(t, client, resource, fake)
	requireTermVersionBoundNoOp(t, client, resource, fake)
}

func paginatedBindTermVersionFake() *fakeTermVersionOCIClient {
	return &fakeTermVersionOCIClient{
		listResults: []termVersionListResult{
			{response: termVersionListResponse(
				[]marketplacepublishersdk.TermVersionSummary{sdkTermVersionSummary("ocid1.termversion.oc1..other", testTermID, "other")},
				"page-2",
			)},
			{response: termVersionListResponse(
				[]marketplacepublishersdk.TermVersionSummary{sdkTermVersionSummary(testTermVersionID, testTermID, testTermDisplayName)},
				"",
			)},
		},
		getResults: []termVersionGetResult{
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable))},
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable))},
		},
	}
}

func requireTermVersionInitialBind(
	t *testing.T,
	client TermVersionServiceClient,
	resource *marketplacepublisherv1beta1.TermVersion,
	fake *fakeTermVersionOCIClient,
) {
	t.Helper()
	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateTermVersion calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListTermVersions calls = %d, want 2", len(fake.listRequests))
	}
	if got := termVersionString(fake.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, testTermDisplayName)
	requireTermVersionCondition(t, resource, shared.Active, v1.ConditionTrue)
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, testTermContent)
}

func requireTermVersionBoundNoOp(
	t *testing.T,
	client TermVersionServiceClient,
	resource *marketplacepublisherv1beta1.TermVersion,
	fake *fakeTermVersionOCIClient,
) {
	t.Helper()
	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("bound no-op CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("bound no-op response = %#v, want successful without requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateTermVersion calls after bound no-op = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateTermVersion calls after bound no-op = %d, want 0", len(fake.updateRequests))
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("GetTermVersion calls after bound no-op = %d, want 2", len(fake.getRequests))
	}
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, testTermContent)
}

func TestTermVersionNoOpReconcileUsesGetAndSkipsUpdate(t *testing.T) {
	resource := newTermVersionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{{
			response: marketplacepublishersdk.GetTermVersionResponse{
				TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateTermVersion calls = %d, want 0", len(fake.updateRequests))
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, testTermDisplayName)
}

func TestTermVersionMutableUpdateSendsChangedFields(t *testing.T) {
	resource := newTermVersionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	resource.Spec.DisplayName = "updated-display-name"
	resource.Spec.FreeformTags = map[string]string{"owner": "osok"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"key": "value"}}
	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
				},
			},
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersionWithTags(
						testTermVersionID,
						testTermID,
						"updated-display-name",
						map[string]string{"owner": "osok"},
						map[string]map[string]interface{}{"ns": {"key": "value"}},
					),
				},
			},
		},
		updateResults: []termVersionUpdateResult{{
			response: marketplacepublishersdk.UpdateTermVersionResponse{
				TermVersion:  sdkTermVersionWithTags(testTermVersionID, testTermID, "updated-display-name", nil, nil),
				OpcRequestId: common.String("opc-update"),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateTermVersion calls = %d, want 1", len(fake.updateRequests))
	}
	details := fake.updateRequests[0].UpdateTermVersionDetails
	if got := termVersionString(details.DisplayName); got != "updated-display-name" {
		t.Fatalf("update displayName = %q, want updated-display-name", got)
	}
	if !reflect.DeepEqual(details.FreeformTags, map[string]string{"owner": "osok"}) {
		t.Fatalf("update freeformTags = %#v, want owner tag", details.FreeformTags)
	}
	wantDefinedTags := map[string]map[string]interface{}{"ns": {"key": "value"}}
	if !reflect.DeepEqual(details.DefinedTags, wantDefinedTags) {
		t.Fatalf("update definedTags = %#v, want %#v", details.DefinedTags, wantDefinedTags)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want update request id", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, "updated-display-name")
}

func TestTermVersionMetadataNameFallbackSendsDisplayNameUpdate(t *testing.T) {
	resource := newTermVersionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	resource.Spec.DisplayName = ""
	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersion(testTermVersionID, testTermID, "oci-readback-display-name", marketplacepublishersdk.TermVersionStatusAvailable),
				},
			},
			{
				response: marketplacepublishersdk.GetTermVersionResponse{
					TermVersion: sdkTermVersion(testTermVersionID, testTermID, resource.Name, marketplacepublishersdk.TermVersionStatusAvailable),
				},
			},
		},
		updateResults: []termVersionUpdateResult{{
			response: marketplacepublishersdk.UpdateTermVersionResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, resource.Name, marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-display-name-fallback-update"),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateTermVersion calls = %d, want 1", len(fake.updateRequests))
	}
	if got := termVersionString(fake.updateRequests[0].DisplayName); got != resource.Name {
		t.Fatalf("update displayName = %q, want metadata.name fallback %q", got, resource.Name)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-display-name-fallback-update" {
		t.Fatalf("status.opcRequestId = %q, want update request id", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, resource.Name)
}

func TestTermVersionTermIDDriftRejectedBeforeUpdate(t *testing.T) {
	resource := newTermVersionResource()
	resource.Annotations[termVersionTermIDAnnotation] = "ocid1.term.oc1..replacement"
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	fake := &fakeTermVersionOCIClient{}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate error = nil, want create-only term drift error")
	}
	if !strings.Contains(err.Error(), "create-only parent term") {
		t.Fatalf("CreateOrUpdate error = %q, want create-only term drift", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = true, want false")
	}
	if len(fake.getRequests)+len(fake.updateRequests) != 0 {
		t.Fatalf("OCI calls get/update = %d/%d, want none", len(fake.getRequests), len(fake.updateRequests))
	}
	requireTermVersionCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestTermVersionContentSecretChangeWithSameBytesUpdatesFingerprintWithoutOCIContentUpdate(t *testing.T) {
	resource := trackedTermVersionResourceWithContentFingerprint()
	resource.Annotations[termVersionContentSecretAnnotation] = "replacement-content"

	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{{
			response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable)),
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/replacement-content": {termVersionDefaultContentSecretKey: []byte(testTermContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if len(fake.updateContentRequests) != 0 {
		t.Fatalf("UpdateTermVersionContent calls = %d, want 0 for same content bytes", len(fake.updateContentRequests))
	}
	requireTermVersionContentFingerprint(t, resource, "replacement-content", termVersionDefaultContentSecretKey, testTermContent)
}

func TestTermVersionContentKeyChangeWithSameBytesCanAccompanyMetadataUpdate(t *testing.T) {
	resource := trackedTermVersionResourceWithContentFingerprint()
	resource.Annotations[termVersionContentSecretKeyAnnotation] = "replacement-key"
	resource.Spec.DisplayName = "updated-display-name"
	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable))},
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, "updated-display-name", marketplacepublishersdk.TermVersionStatusAvailable))},
		},
		updateResults: []termVersionUpdateResult{{
			response: marketplacepublishersdk.UpdateTermVersionResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, "updated-display-name", marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-metadata-update"),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {"replacement-key": []byte(testTermContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateTermVersion calls = %d, want 1", len(fake.updateRequests))
	}
	if len(fake.updateContentRequests) != 0 {
		t.Fatalf("UpdateTermVersionContent calls = %d, want 0 for same content bytes", len(fake.updateContentRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-metadata-update" {
		t.Fatalf("status.opcRequestId = %q, want metadata update request id", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, "updated-display-name")
	requireTermVersionContentFingerprint(t, resource, "term-content", "replacement-key", testTermContent)
}

func TestTermVersionContentBytesDriftUpdatesContentAndProjectsReadback(t *testing.T) {
	resource := trackedTermVersionResourceWithContentFingerprint()
	updatedContent := "updated terms and conditions"
	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable))},
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable))},
		},
		updateContentResults: []termVersionUpdateContentResult{{
			response: marketplacepublishersdk.UpdateTermVersionContentResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, "response-display-name", marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-content-update"),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte(updatedContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if len(fake.updateContentRequests) != 1 {
		t.Fatalf("UpdateTermVersionContent calls = %d, want 1", len(fake.updateContentRequests))
	}
	updateRequest := fake.updateContentRequests[0]
	if got := termVersionString(updateRequest.TermVersionId); got != testTermVersionID {
		t.Fatalf("content update termVersionId = %q, want %q", got, testTermVersionID)
	}
	if got := readTermVersionContent(t, updateRequest.UpdateTermVersionContent); got != updatedContent {
		t.Fatalf("content update body = %q, want %q", got, updatedContent)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateTermVersion calls = %d, want 0 when content update is selected", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-content-update" {
		t.Fatalf("status.opcRequestId = %q, want content update request id", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, testTermDisplayName)
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, updatedContent)
}

func TestTermVersionContentUpdateWithTagsAppliesRemainingUpdateDrift(t *testing.T) {
	resource := trackedTermVersionResourceWithContentFingerprint()
	resource.Spec.FreeformTags = map[string]string{"owner": "osok"}
	updatedContent := "updated terms and conditions"
	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable))},
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable))},
			{response: termVersionGetResponse(sdkTermVersionWithTags(testTermVersionID, testTermID, testTermDisplayName, map[string]string{"owner": "osok"}, nil))},
		},
		updateContentResults: []termVersionUpdateContentResult{{
			response: marketplacepublishersdk.UpdateTermVersionContentResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-content-update"),
			},
		}},
		updateResults: []termVersionUpdateResult{{
			response: marketplacepublishersdk.UpdateTermVersionResponse{
				TermVersion:  sdkTermVersionWithTags(testTermVersionID, testTermID, testTermDisplayName, map[string]string{"owner": "osok"}, nil),
				OpcRequestId: common.String("opc-tags-after-content"),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte(updatedContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateContentRequests) != 1 {
		t.Fatalf("UpdateTermVersionContent calls = %d, want 1", len(fake.updateContentRequests))
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateTermVersion calls = %d, want 1 for tags after content update", len(fake.updateRequests))
	}
	requireTermVersionUpdateTags(t, fake.updateRequests[0], map[string]string{"owner": "osok"}, nil)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-tags-after-content" {
		t.Fatalf("status.opcRequestId = %q, want post-content tag update request id", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, testTermDisplayName)
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, updatedContent)
}

func TestTermVersionContentUpdateUsesMetadataNameFallbackDisplayName(t *testing.T) {
	resource := trackedTermVersionResourceWithContentFingerprint()
	resource.Spec.DisplayName = ""
	updatedContent := "updated terms and conditions"
	fake := &fakeTermVersionOCIClient{
		getResults: []termVersionGetResult{
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, "oci-readback-display-name", marketplacepublishersdk.TermVersionStatusAvailable))},
			{response: termVersionGetResponse(sdkTermVersion(testTermVersionID, testTermID, resource.Name, marketplacepublishersdk.TermVersionStatusAvailable))},
		},
		updateContentResults: []termVersionUpdateContentResult{{
			response: marketplacepublishersdk.UpdateTermVersionContentResponse{
				TermVersion:  sdkTermVersion(testTermVersionID, testTermID, resource.Name, marketplacepublishersdk.TermVersionStatusAvailable),
				OpcRequestId: common.String("opc-content-display-name-fallback"),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/term-content": {termVersionDefaultContentSecretKey: []byte(updatedContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testTermVersionRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if len(fake.updateContentRequests) != 1 {
		t.Fatalf("UpdateTermVersionContent calls = %d, want 1", len(fake.updateContentRequests))
	}
	if got := termVersionString(fake.updateContentRequests[0].DisplayName); got != resource.Name {
		t.Fatalf("content update displayName = %q, want metadata.name fallback %q", got, resource.Name)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateTermVersion calls = %d, want 0 for content update path", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-content-display-name-fallback" {
		t.Fatalf("status.opcRequestId = %q, want content update request id", got)
	}
	requireTermVersionStatus(t, resource, testTermVersionID, testTermID, resource.Name)
	requireTermVersionContentFingerprint(t, resource, "term-content", termVersionDefaultContentSecretKey, updatedContent)
}

func TestTermVersionDeleteWaitsUntilReadbackConfirmsDeleted(t *testing.T) {
	resource := newTermVersionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	fake := &fakeTermVersionOCIClient{
		deleteResults: []termVersionDeleteResult{{response: marketplacepublishersdk.DeleteTermVersionResponse{OpcRequestId: common.String("opc-delete")}}},
		getResults: []termVersionGetResult{{
			response: marketplacepublishersdk.GetTermVersionResponse{
				TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusAvailable),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if deleted {
		t.Fatal("Delete deleted = true, want false while readback is not terminal")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want delete request id", got)
	}
	requireTermVersionCondition(t, resource, shared.Terminating, v1.ConditionTrue)
	requireTermVersionAsync(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
}

func TestTermVersionDeleteCompletesOnDeletedReadback(t *testing.T) {
	resource := newTermVersionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	fake := &fakeTermVersionOCIClient{
		deleteResults: []termVersionDeleteResult{{response: marketplacepublishersdk.DeleteTermVersionResponse{OpcRequestId: common.String("opc-delete")}}},
		getResults: []termVersionGetResult{{
			response: marketplacepublishersdk.GetTermVersionResponse{
				TermVersion: sdkTermVersion(testTermVersionID, testTermID, testTermDisplayName, marketplacepublishersdk.TermVersionStatusDeleted),
			},
		}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete deleted = false, want true after deleted status readback")
	}
	requireTermVersionCondition(t, resource, shared.Terminating, v1.ConditionTrue)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("async.current = %#v, want nil after confirmed delete", resource.Status.OsokStatus.Async.Current)
	}
}

func TestTermVersionDeleteKeepsAuthShapedNotFoundConservative(t *testing.T) {
	resource := newTermVersionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-404"
	fake := &fakeTermVersionOCIClient{
		deleteResults: []termVersionDeleteResult{{err: authErr}},
	}
	client := newTestTermVersionClient(fake, fakeTermVersionCredentialClient{})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete error = nil, want auth-shaped not-found error")
	}
	if deleted {
		t.Fatal("Delete deleted = true, want false for auth-shaped not-found")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-auth-404" {
		t.Fatalf("status.opcRequestId = %q, want auth-shaped error request id", got)
	}
	requireTermVersionCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

type fakeTermVersionOCIClient struct {
	createRequests        []marketplacepublishersdk.CreateTermVersionRequest
	createResults         []termVersionCreateResult
	getRequests           []marketplacepublishersdk.GetTermVersionRequest
	getResults            []termVersionGetResult
	listRequests          []marketplacepublishersdk.ListTermVersionsRequest
	listResults           []termVersionListResult
	updateRequests        []marketplacepublishersdk.UpdateTermVersionRequest
	updateResults         []termVersionUpdateResult
	updateContentRequests []marketplacepublishersdk.UpdateTermVersionContentRequest
	updateContentResults  []termVersionUpdateContentResult
	deleteRequests        []marketplacepublishersdk.DeleteTermVersionRequest
	deleteResults         []termVersionDeleteResult
}

type termVersionCreateResult struct {
	response marketplacepublishersdk.CreateTermVersionResponse
	err      error
}

type termVersionGetResult struct {
	response marketplacepublishersdk.GetTermVersionResponse
	err      error
}

type termVersionListResult struct {
	response marketplacepublishersdk.ListTermVersionsResponse
	err      error
}

type termVersionUpdateResult struct {
	response marketplacepublishersdk.UpdateTermVersionResponse
	err      error
}

type termVersionUpdateContentResult struct {
	response marketplacepublishersdk.UpdateTermVersionContentResponse
	err      error
}

type termVersionDeleteResult struct {
	response marketplacepublishersdk.DeleteTermVersionResponse
	err      error
}

func (f *fakeTermVersionOCIClient) CreateTermVersion(_ context.Context, request marketplacepublishersdk.CreateTermVersionRequest) (marketplacepublishersdk.CreateTermVersionResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if len(f.createResults) == 0 {
		return marketplacepublishersdk.CreateTermVersionResponse{}, fmt.Errorf("unexpected CreateTermVersion call")
	}
	result := f.createResults[0]
	f.createResults = f.createResults[1:]
	return result.response, result.err
}

func (f *fakeTermVersionOCIClient) GetTermVersion(_ context.Context, request marketplacepublishersdk.GetTermVersionRequest) (marketplacepublishersdk.GetTermVersionResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		return marketplacepublishersdk.GetTermVersionResponse{}, fmt.Errorf("unexpected GetTermVersion call")
	}
	result := f.getResults[0]
	f.getResults = f.getResults[1:]
	return result.response, result.err
}

func (f *fakeTermVersionOCIClient) ListTermVersions(_ context.Context, request marketplacepublishersdk.ListTermVersionsRequest) (marketplacepublishersdk.ListTermVersionsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		return marketplacepublishersdk.ListTermVersionsResponse{}, fmt.Errorf("unexpected ListTermVersions call")
	}
	result := f.listResults[0]
	f.listResults = f.listResults[1:]
	return result.response, result.err
}

func (f *fakeTermVersionOCIClient) UpdateTermVersion(_ context.Context, request marketplacepublishersdk.UpdateTermVersionRequest) (marketplacepublishersdk.UpdateTermVersionResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if len(f.updateResults) == 0 {
		return marketplacepublishersdk.UpdateTermVersionResponse{}, fmt.Errorf("unexpected UpdateTermVersion call")
	}
	result := f.updateResults[0]
	f.updateResults = f.updateResults[1:]
	return result.response, result.err
}

func (f *fakeTermVersionOCIClient) UpdateTermVersionContent(_ context.Context, request marketplacepublishersdk.UpdateTermVersionContentRequest) (marketplacepublishersdk.UpdateTermVersionContentResponse, error) {
	f.updateContentRequests = append(f.updateContentRequests, request)
	if len(f.updateContentResults) == 0 {
		return marketplacepublishersdk.UpdateTermVersionContentResponse{}, fmt.Errorf("unexpected UpdateTermVersionContent call")
	}
	result := f.updateContentResults[0]
	f.updateContentResults = f.updateContentResults[1:]
	return result.response, result.err
}

func (f *fakeTermVersionOCIClient) DeleteTermVersion(_ context.Context, request marketplacepublishersdk.DeleteTermVersionRequest) (marketplacepublishersdk.DeleteTermVersionResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if len(f.deleteResults) == 0 {
		return marketplacepublishersdk.DeleteTermVersionResponse{}, fmt.Errorf("unexpected DeleteTermVersion call")
	}
	result := f.deleteResults[0]
	f.deleteResults = f.deleteResults[1:]
	return result.response, result.err
}

type fakeTermVersionCredentialClient struct {
	secrets map[string]map[string][]byte
	err     error
}

func (f fakeTermVersionCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return false, nil
}

func (f fakeTermVersionCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return false, nil
}

func (f fakeTermVersionCredentialClient) GetSecret(_ context.Context, name string, namespace string) (map[string][]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.secrets == nil {
		return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
	}
	secret, ok := f.secrets[namespace+"/"+name]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
	}
	return secret, nil
}

func (f fakeTermVersionCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return false, nil
}

var _ credhelper.CredentialClient = fakeTermVersionCredentialClient{}

func newTestTermVersionClient(client termVersionOCIClient, credentials credhelper.CredentialClient) TermVersionServiceClient {
	return newTermVersionServiceClientWithOCIClient(client, credentials, loggerutil.OSOKLogger{})
}

func newTermVersionResource() *marketplacepublisherv1beta1.TermVersion {
	return &marketplacepublisherv1beta1.TermVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "termversion-sample",
			Namespace: "default",
			Annotations: map[string]string{
				termVersionTermIDAnnotation:        testTermID,
				termVersionContentSecretAnnotation: "term-content",
			},
		},
		Spec: marketplacepublisherv1beta1.TermVersionSpec{
			DisplayName: testTermDisplayName,
		},
	}
}

func trackedTermVersionResourceWithContentFingerprint() *marketplacepublisherv1beta1.TermVersion {
	resource := newTermVersionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTermVersionID)
	resource.Status.Id = testTermVersionID
	resource.Status.TermId = testTermID
	recordTermVersionContentFingerprint(resource, termVersionIdentity{
		contentSecret: "term-content",
		contentKey:    termVersionDefaultContentSecretKey,
		contentHash:   termVersionContentFingerprint([]byte(testTermContent)),
		contentSourceHash: termVersionContentSourceFingerprint(termVersionIdentity{
			contentSecret: "term-content",
			contentKey:    termVersionDefaultContentSecretKey,
		}),
	})
	return resource
}

func testTermVersionRequest() ctrl.Request {
	return ctrl.Request{}
}

func termVersionGetResponse(termVersion marketplacepublishersdk.TermVersion) marketplacepublishersdk.GetTermVersionResponse {
	return marketplacepublishersdk.GetTermVersionResponse{TermVersion: termVersion}
}

func termVersionListResponse(
	items []marketplacepublishersdk.TermVersionSummary,
	nextPage string,
) marketplacepublishersdk.ListTermVersionsResponse {
	response := marketplacepublishersdk.ListTermVersionsResponse{
		TermVersionCollection: marketplacepublishersdk.TermVersionCollection{Items: items},
	}
	if nextPage != "" {
		response.OpcNextPage = common.String(nextPage)
	}
	return response
}

func sdkTermVersion(
	id string,
	termID string,
	displayName string,
	status marketplacepublishersdk.TermVersionStatusEnum,
) marketplacepublishersdk.TermVersion {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.TermVersion{
		TermId:         common.String(termID),
		TermAuthor:     marketplacepublishersdk.TermAuthorPartner,
		DisplayName:    common.String(displayName),
		Attachment:     &marketplacepublishersdk.TermVersionAttachment{ContentUrl: common.String("https://objectstorage.example/terms.pdf"), MimeType: common.String("application/pdf")},
		Status:         status,
		LifecycleState: marketplacepublishersdk.TermVersionLifecycleStateActive,
		TimeCreated:    &now,
		TimeUpdated:    &now,
		Id:             common.String(id),
		CompartmentId:  common.String(testCompartmentID),
		Author:         marketplacepublishersdk.TermAuthorPartner,
	}
}

func sdkTermVersionWithTags(
	id string,
	termID string,
	displayName string,
	freeformTags map[string]string,
	definedTags map[string]map[string]interface{},
) marketplacepublishersdk.TermVersion {
	termVersion := sdkTermVersion(id, termID, displayName, marketplacepublishersdk.TermVersionStatusAvailable)
	termVersion.FreeformTags = freeformTags
	termVersion.DefinedTags = definedTags
	return termVersion
}

func sdkTermVersionSummary(id string, _ string, displayName string) marketplacepublishersdk.TermVersionSummary {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.TermVersionSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(testCompartmentID),
		DisplayName:    common.String(displayName),
		Status:         marketplacepublishersdk.TermVersionStatusAvailable,
		LifecycleState: marketplacepublishersdk.TermVersionLifecycleStateActive,
		TimeCreated:    &now,
		TimeUpdated:    &now,
	}
}

func readTermVersionCreateContent(t *testing.T, reader io.ReadCloser) string {
	t.Helper()
	if reader == nil {
		t.Fatal("create content reader = nil")
	}
	return readTermVersionContent(t, reader)
}

func readTermVersionContent(t *testing.T, reader io.ReadCloser) string {
	t.Helper()
	if reader == nil {
		t.Fatal("content reader = nil")
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read content: %v", err)
	}
	return string(content)
}

func requireTermVersionStatus(t *testing.T, resource *marketplacepublisherv1beta1.TermVersion, id string, termID string, displayName string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	if got := resource.Status.TermId; got != termID {
		t.Fatalf("status.termId = %q, want %q", got, termID)
	}
	if got := resource.Status.DisplayName; got != displayName {
		t.Fatalf("status.displayName = %q, want %q", got, displayName)
	}
	if got := resource.Status.Status; got == "" {
		t.Fatal("status.sdkStatus = empty, want projected OCI status")
	}
}

func requireTermVersionUpdateTags(
	t *testing.T,
	request marketplacepublishersdk.UpdateTermVersionRequest,
	freeformTags map[string]string,
	definedTags map[string]map[string]interface{},
) {
	t.Helper()
	if !reflect.DeepEqual(request.FreeformTags, freeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", request.FreeformTags, freeformTags)
	}
	if !reflect.DeepEqual(request.DefinedTags, definedTags) {
		t.Fatalf("update definedTags = %#v, want %#v", request.DefinedTags, definedTags)
	}
}

func requireTermVersionCondition(
	t *testing.T,
	resource *marketplacepublisherv1beta1.TermVersion,
	conditionType shared.OSOKConditionType,
	status v1.ConditionStatus,
) {
	t.Helper()
	for _, condition := range resource.Status.OsokStatus.Conditions {
		if condition.Type == conditionType && condition.Status == status {
			return
		}
	}
	t.Fatalf("condition %s/%s not found in %#v", conditionType, status, resource.Status.OsokStatus.Conditions)
}

func requireTermVersionAsync(
	t *testing.T,
	resource *marketplacepublisherv1beta1.TermVersion,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("async.current = nil, want delete pending")
	}
	if current.Phase != phase || current.NormalizedClass != class {
		t.Fatalf("async.current = %#v, want phase %s class %s", current, phase, class)
	}
}

func requireTermVersionContentFingerprint(
	t *testing.T,
	resource *marketplacepublisherv1beta1.TermVersion,
	secretName string,
	secretKey string,
	content string,
) {
	t.Helper()
	got, ok := termVersionRecordedContentFingerprint(resource)
	if !ok {
		t.Fatalf("content fingerprint missing from status.status.message %q", resource.Status.OsokStatus.Message)
	}
	want := termVersionContentFingerprint([]byte(content))
	if got != want {
		t.Fatalf("content fingerprint = %q, want %q", got, want)
	}
	gotSource, ok := termVersionRecordedContentSourceFingerprint(resource)
	if !ok {
		t.Fatalf("content source fingerprint missing from status.status.message %q", resource.Status.OsokStatus.Message)
	}
	wantSource := termVersionContentSourceFingerprint(termVersionIdentity{
		contentSecret: secretName,
		contentKey:    secretKey,
	})
	if gotSource != wantSource {
		t.Fatalf("content source fingerprint = %q, want %q", gotSource, wantSource)
	}
}

func requireTermVersionStrings(t *testing.T, label string, values []string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !termVersionTestStringContains(values, want) {
			t.Fatalf("%s = %#v, want %s", label, values, want)
		}
	}
}

func termVersionTestStringContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
