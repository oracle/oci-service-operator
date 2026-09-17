/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package genericartifactcontentbypath

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	genericartifactscontentsdk "github.com/oracle/oci-go-sdk/v65/genericartifactscontent"
	genericartifactscontentv1beta1 "github.com/oracle/oci-service-operator/api/genericartifactscontent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCompartmentID = "ocid1.compartment.oc1..compartment"
	testRepositoryID  = "ocid1.repository.oc1..repository"
	testArtifactPath  = "project01/my-web-app/artifact-abc"
	testVersion       = "1.0.0"
	testArtifactID    = "ocid1.genericartifact.oc1..artifact"
	testContent       = "artifact bytes"
)

func TestGenericArtifactContentByPathCreateUploadsSecretAndRecordsStatus(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	fake := &fakeGenericArtifactContentByPathOCIClient{
		getResults: []genericArtifactContentByPathGetResult{
			{err: errortest.NewServiceError(404, "NotFound", "not found")},
			{response: genericArtifactContentByPathGetResponse("etag-readback", testContent)},
		},
		putResults: []genericArtifactContentByPathPutResult{{
			response: genericArtifactContentByPathPutResponse("etag-put", "opc-put", sdkGenericArtifactContentByPath(testArtifactID)),
		}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/artifact-content": {genericArtifactContentByPathDefaultContentKey: []byte(testContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testGenericArtifactContentByPathRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.putRequests) != 1 {
		t.Fatalf("PutGenericArtifactContentByPath calls = %d, want 1", len(fake.putRequests))
	}
	requireGenericArtifactContentByPathPutRequest(t, fake.putRequests[0], "")
	if got := string(fake.putBodies[0]); got != testContent {
		t.Fatalf("put content = %q, want %q", got, testContent)
	}
	requireGenericArtifactContentByPathStatus(t, resource, testArtifactID, testCompartmentID, testRepositoryID, testArtifactPath, testVersion)
	if got := resource.Status.Etag; got != "etag-readback" {
		t.Fatalf("status.etag = %q, want readback etag", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-put" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-put")
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Active)
}

func TestGenericArtifactContentByPathRecognizesServiceMetadataNotFound(t *testing.T) {
	for _, code := range []string{"GENERIC_ARTIFACT_METADATA_NOT_FOUND", "ARTIFACT_NOT_AVAILABLE"} {
		err := errortest.NewServiceError(404, code, "artifact not found")
		if !isGenericArtifactContentByPathUnambiguousNotFound(err) {
			t.Fatalf("%s was not classified as an unambiguous absence", code)
		}
	}
}

func TestGenericArtifactContentByPathNoOpBindsExistingMatchingContent(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	fake := &fakeGenericArtifactContentByPathOCIClient{
		getResults: []genericArtifactContentByPathGetResult{{
			response: genericArtifactContentByPathGetResponse("etag-existing", testContent),
		}},
		listResults: []genericArtifactContentByPathListResult{
			{
				response: genericArtifactContentByPathListResponse("next-page", sdkGenericArtifactContentByPathSummary("ocid1.genericartifact.oc1..other", "other/path", testVersion, artifactssdk.GenericArtifactLifecycleStateAvailable)),
			},
			{
				response: genericArtifactContentByPathListResponse("", sdkGenericArtifactContentByPathSummary(testArtifactID, testArtifactPath, testVersion, artifactssdk.GenericArtifactLifecycleStateAvailable)),
			},
		},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/artifact-content": {genericArtifactContentByPathDefaultContentKey: []byte(testContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testGenericArtifactContentByPathRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.putRequests) != 0 {
		t.Fatalf("PutGenericArtifactContentByPath calls = %d, want no duplicate upload", len(fake.putRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListGenericArtifacts calls = %d, want pagination across two pages", len(fake.listRequests))
	}
	if got := stringValue(fake.listRequests[1].Page); got != "next-page" {
		t.Fatalf("second list page = %q, want next-page", got)
	}
	requireGenericArtifactContentByPathStatus(t, resource, testArtifactID, testCompartmentID, testRepositoryID, testArtifactPath, testVersion)
	if got := resource.Status.Etag; got != "etag-existing" {
		t.Fatalf("status.etag = %q, want %q", got, "etag-existing")
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Active)
}

func TestGenericArtifactContentByPathUpdateUsesIfMatchWhenContentDiffers(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	resource.Status.CompartmentId = testCompartmentID
	resource.Status.RepositoryId = testRepositoryID
	resource.Status.ArtifactPath = testArtifactPath
	resource.Status.Version = testVersion
	resource.Status.Id = testArtifactID
	resource.Status.OsokStatus.Ocid = shared.OCID(testArtifactID)

	fake := &fakeGenericArtifactContentByPathOCIClient{
		getResults: []genericArtifactContentByPathGetResult{
			{response: genericArtifactContentByPathGetResponse("etag-old", "old bytes")},
			{response: genericArtifactContentByPathGetResponse("etag-new", "new bytes")},
		},
		putResults: []genericArtifactContentByPathPutResult{{
			response: genericArtifactContentByPathPutResponse("etag-put", "opc-update", sdkGenericArtifactContentByPath(testArtifactID)),
		}},
		listResults: []genericArtifactContentByPathListResult{{
			response: genericArtifactContentByPathListResponse("", sdkGenericArtifactContentByPathSummary(testArtifactID, testArtifactPath, testVersion, artifactssdk.GenericArtifactLifecycleStateAvailable)),
		}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/artifact-content": {genericArtifactContentByPathDefaultContentKey: []byte("new bytes")},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testGenericArtifactContentByPathRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful without requeue", response)
	}
	if len(fake.putRequests) != 1 {
		t.Fatalf("PutGenericArtifactContentByPath calls = %d, want 1", len(fake.putRequests))
	}
	if got := stringValue(fake.putRequests[0].IfMatch); got != "etag-old" {
		t.Fatalf("put ifMatch = %q, want %q", got, "etag-old")
	}
	if got := string(fake.putBodies[0]); got != "new bytes" {
		t.Fatalf("put content = %q, want new bytes", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want update request id", got)
	}
	if got := resource.Status.Etag; got != "etag-new" {
		t.Fatalf("status.etag = %q, want readback etag", got)
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Active)
}

func TestGenericArtifactContentByPathRejectsImmutableIdentityDriftBeforeOCI(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	resource.Status.RepositoryId = "ocid1.repository.oc1..old"
	fake := &fakeGenericArtifactContentByPathOCIClient{}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, testGenericArtifactContentByPathRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable drift error")
	}
	if !strings.Contains(err.Error(), "spec.repositoryId is immutable") {
		t.Fatalf("CreateOrUpdate() error = %q, want repositoryId immutable drift", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if len(fake.getRequests)+len(fake.putRequests) != 0 {
		t.Fatalf("OCI calls get/put = %d/%d, want none", len(fake.getRequests), len(fake.putRequests))
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Failed)
}

func TestGenericArtifactContentByPathCreateRequiresContentWhenMissing(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	resource.Spec.Content.SecretName = ""
	fake := &fakeGenericArtifactContentByPathOCIClient{
		getResults: []genericArtifactContentByPathGetResult{{err: errortest.NewServiceError(404, "NotFound", "not found")}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	response, err := client.CreateOrUpdate(context.Background(), resource, testGenericArtifactContentByPathRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing content error")
	}
	if !strings.Contains(err.Error(), "spec.content.secretName is required") {
		t.Fatalf("CreateOrUpdate() error = %q, want missing content message", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if len(fake.putRequests) != 0 {
		t.Fatalf("PutGenericArtifactContentByPath calls = %d, want none", len(fake.putRequests))
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Failed)
}

func TestGenericArtifactContentByPathRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	ociErr := errortest.NewServiceError(500, "InternalError", "service unavailable")
	ociErr.OpcRequestID = "opc-error"
	fake := &fakeGenericArtifactContentByPathOCIClient{
		getResults: []genericArtifactContentByPathGetResult{{err: errortest.NewServiceError(404, "NotFound", "not found")}},
		putResults: []genericArtifactContentByPathPutResult{{err: ociErr}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{
		secrets: map[string]map[string][]byte{
			"default/artifact-content": {genericArtifactContentByPathDefaultContentKey: []byte(testContent)},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testGenericArtifactContentByPathRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-error" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-error")
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Failed)
}

func TestGenericArtifactContentByPathDeleteConfirmsAbsentContent(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	resource.Status.Etag = "etag-existing"
	fake := &fakeGenericArtifactContentByPathOCIClient{
		deleteResults: []genericArtifactContentByPathDeleteResult{{response: genericArtifactContentByPathDeleteResponse("opc-delete")}},
		getResults:    []genericArtifactContentByPathGetResult{{err: errortest.NewServiceError(404, "NotFound", "not found")}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteGenericArtifactByPath calls = %d, want 1", len(fake.deleteRequests))
	}
	deleteRequest := fake.deleteRequests[0]
	if got := stringValue(deleteRequest.RepositoryId); got != testRepositoryID {
		t.Fatalf("delete repositoryId = %q, want %q", got, testRepositoryID)
	}
	if got := stringValue(deleteRequest.ArtifactPath); got != testArtifactPath {
		t.Fatalf("delete artifactPath = %q, want %q", got, testArtifactPath)
	}
	if got := stringValue(deleteRequest.Version); got != testVersion {
		t.Fatalf("delete version = %q, want %q", got, testVersion)
	}
	if got := stringValue(deleteRequest.IfMatch); got != "etag-existing" {
		t.Fatalf("delete ifMatch = %q, want etag-existing", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

func TestGenericArtifactContentByPathDeleteRetainsFinalizerWhenContentStillExists(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	fake := &fakeGenericArtifactContentByPathOCIClient{
		deleteResults: []genericArtifactContentByPathDeleteResult{{response: genericArtifactContentByPathDeleteResponse("opc-delete")}},
		getResults:    []genericArtifactContentByPathGetResult{{response: genericArtifactContentByPathGetResponse("etag-existing", testContent)}},
		listResults: []genericArtifactContentByPathListResult{{
			response: genericArtifactContentByPathListResponse("", sdkGenericArtifactContentByPathSummary(testArtifactID, testArtifactPath, testVersion, artifactssdk.GenericArtifactLifecycleStateAvailable)),
		}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	result, err := client.DeleteWithResult(context.Background(), resource)
	if err != nil {
		t.Fatalf("DeleteWithResult() error = %v", err)
	}
	if result.Deleted {
		t.Fatal("DeleteWithResult() deleted = true, want finalizer retained")
	}
	if result.RequeueDuration != genericArtifactContentByPathDeleteRequeue {
		t.Fatalf("DeleteWithResult() requeue = %s, want %s", result.RequeueDuration, genericArtifactContentByPathDeleteRequeue)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "delete accepted; retaining finalizer while content remains readable") {
		t.Fatalf("status.message = %q, want pending delete message", resource.Status.OsokStatus.Message)
	}
	requireGenericArtifactContentByPathStatus(t, resource, testArtifactID, testCompartmentID, testRepositoryID, testArtifactPath, testVersion)
	requireGenericArtifactContentByPathCondition(t, resource, shared.Terminating)
}

func TestGenericArtifactContentByPathDeleteConflictRereadsDeletingArtifact(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	conflictErr := errortest.NewServiceError(409, "Conflict", "delete is already in progress")
	conflictErr.OpcRequestID = "opc-conflict"
	fake := &fakeGenericArtifactContentByPathOCIClient{
		deleteResults: []genericArtifactContentByPathDeleteResult{{err: conflictErr}},
		getResults:    []genericArtifactContentByPathGetResult{{response: genericArtifactContentByPathGetResponse("etag-existing", testContent)}},
		listResults: []genericArtifactContentByPathListResult{{
			response: genericArtifactContentByPathListResponse("", sdkGenericArtifactContentByPathSummary(testArtifactID, testArtifactPath, testVersion, artifactssdk.GenericArtifactLifecycleStateDeleting)),
		}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	result, err := client.DeleteWithResult(context.Background(), resource)
	if err != nil {
		t.Fatalf("DeleteWithResult() error = %v", err)
	}
	if result.Deleted {
		t.Fatal("DeleteWithResult() deleted = true, want finalizer retained while artifact is DELETING")
	}
	if result.RequeueDuration != genericArtifactContentByPathDeleteRequeue {
		t.Fatalf("DeleteWithResult() requeue = %s, want %s", result.RequeueDuration, genericArtifactContentByPathDeleteRequeue)
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteGenericArtifactByPath calls = %d, want 1", len(fake.deleteRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetGenericArtifactContentByPath calls = %d, want 1 conflict confirmation read", len(fake.getRequests))
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListGenericArtifacts calls = %d, want 1 conflict confirmation list", len(fake.listRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-conflict" {
		t.Fatalf("status.opcRequestId = %q, want opc-conflict", got)
	}
	if got := resource.Status.LifecycleState; got != string(artifactssdk.GenericArtifactLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "retryable conflict") {
		t.Fatalf("status.message = %q, want retryable conflict message", resource.Status.OsokStatus.Message)
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Terminating)
}

func TestGenericArtifactContentByPathDeleteConflictConfirmsAbsentContent(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	conflictErr := errortest.NewServiceError(409, "IncorrectState", "delete state conflict")
	conflictErr.OpcRequestID = "opc-conflict"
	fake := &fakeGenericArtifactContentByPathOCIClient{
		deleteResults: []genericArtifactContentByPathDeleteResult{{err: conflictErr}},
		getResults:    []genericArtifactContentByPathGetResult{{err: errortest.NewServiceError(404, "NotFound", "not found")}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after conflict confirms content is absent")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteGenericArtifactByPath calls = %d, want 1", len(fake.deleteRequests))
	}
	if len(fake.listRequests) != 0 {
		t.Fatalf("ListGenericArtifacts calls = %d, want none when content is absent", len(fake.listRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-conflict" {
		t.Fatalf("status.opcRequestId = %q, want opc-conflict", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Terminating)
}

func TestGenericArtifactContentByPathDeleteConflictConfirmsDeletedLifecycle(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	conflictErr := errortest.NewServiceError(409, "IncorrectState", "delete state conflict")
	conflictErr.OpcRequestID = "opc-conflict"
	fake := &fakeGenericArtifactContentByPathOCIClient{
		deleteResults: []genericArtifactContentByPathDeleteResult{{err: conflictErr}},
		getResults:    []genericArtifactContentByPathGetResult{{response: genericArtifactContentByPathGetResponse("etag-existing", testContent)}},
		listResults: []genericArtifactContentByPathListResult{{
			response: genericArtifactContentByPathListResponse("", sdkGenericArtifactContentByPathSummary(testArtifactID, testArtifactPath, testVersion, artifactssdk.GenericArtifactLifecycleStateDeleted)),
		}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after conflict confirms DELETED lifecycle")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteGenericArtifactByPath calls = %d, want 1", len(fake.deleteRequests))
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListGenericArtifacts calls = %d, want 1 lifecycle confirmation list", len(fake.listRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-conflict" {
		t.Fatalf("status.opcRequestId = %q, want opc-conflict", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Terminating)
}

func TestGenericArtifactContentByPathDeleteSkipsDuplicateCallWhenStatusDeleting(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	resource.Status.LifecycleState = string(artifactssdk.GenericArtifactLifecycleStateDeleting)
	fake := &fakeGenericArtifactContentByPathOCIClient{
		getResults: []genericArtifactContentByPathGetResult{{response: genericArtifactContentByPathGetResponse("etag-existing", testContent)}},
		listResults: []genericArtifactContentByPathListResult{{
			response: genericArtifactContentByPathListResponse("", sdkGenericArtifactContentByPathSummary(testArtifactID, testArtifactPath, testVersion, artifactssdk.GenericArtifactLifecycleStateDeleting)),
		}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	result, err := client.DeleteWithResult(context.Background(), resource)
	if err != nil {
		t.Fatalf("DeleteWithResult() error = %v", err)
	}
	if result.Deleted {
		t.Fatal("DeleteWithResult() deleted = true, want finalizer retained while artifact is DELETING")
	}
	if result.RequeueDuration != genericArtifactContentByPathDeleteRequeue {
		t.Fatalf("DeleteWithResult() requeue = %s, want %s", result.RequeueDuration, genericArtifactContentByPathDeleteRequeue)
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteGenericArtifactByPath calls = %d, want no duplicate delete while status is DELETING", len(fake.deleteRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetGenericArtifactContentByPath calls = %d, want 1 confirmation read", len(fake.getRequests))
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListGenericArtifacts calls = %d, want 1 confirmation list", len(fake.listRequests))
	}
	if got := resource.Status.LifecycleState; got != string(artifactssdk.GenericArtifactLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "already in progress") {
		t.Fatalf("status.message = %q, want already in progress message", resource.Status.OsokStatus.Message)
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Terminating)
}

func TestGenericArtifactContentByPathDeleteRetainsFinalizerOnAuthShapedNotFound(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	fake := &fakeGenericArtifactContentByPathOCIClient{
		deleteResults: []genericArtifactContentByPathDeleteResult{{err: errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "not authorized or not found")}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for ambiguous auth-shaped not found")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want nil while retaining finalizer")
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Terminating)
}

func TestGenericArtifactContentByPathDeleteRetainsFinalizerOnAuthShapedReadback(t *testing.T) {
	resource := newGenericArtifactContentByPathResource()
	fake := &fakeGenericArtifactContentByPathOCIClient{
		deleteResults: []genericArtifactContentByPathDeleteResult{{response: genericArtifactContentByPathDeleteResponse("opc-delete")}},
		getResults:    []genericArtifactContentByPathGetResult{{err: errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "not authorized or not found")}},
	}
	client := newTestGenericArtifactContentByPathClient(fake, fakeGenericArtifactContentByPathCredentialClient{})

	result, err := client.DeleteWithResult(context.Background(), resource)
	if err != nil {
		t.Fatalf("DeleteWithResult() error = %v", err)
	}
	if result.Deleted {
		t.Fatal("DeleteWithResult() deleted = true, want finalizer retained for auth-shaped readback")
	}
	if result.RequeueDuration != genericArtifactContentByPathDeleteRequeue {
		t.Fatalf("DeleteWithResult() requeue = %s, want %s", result.RequeueDuration, genericArtifactContentByPathDeleteRequeue)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want nil while retaining finalizer")
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "delete readback returned NotAuthorizedOrNotFound") {
		t.Fatalf("status.message = %q, want auth-shaped readback message", resource.Status.OsokStatus.Message)
	}
	requireGenericArtifactContentByPathCondition(t, resource, shared.Terminating)
}

func newTestGenericArtifactContentByPathClient(
	ociClient *fakeGenericArtifactContentByPathOCIClient,
	credentialClient credhelper.CredentialClient,
) *genericArtifactContentByPathRuntimeClient {
	return &genericArtifactContentByPathRuntimeClient{
		contentClient:    ociClient,
		artifactClient:   ociClient,
		credentialClient: credentialClient,
		log:              loggerutil.OSOKLogger{},
	}
}

func newGenericArtifactContentByPathResource() *genericartifactscontentv1beta1.GenericArtifactContentByPath {
	return &genericartifactscontentv1beta1.GenericArtifactContentByPath{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "artifact-content",
			Namespace: "default",
		},
		Spec: genericartifactscontentv1beta1.GenericArtifactContentByPathSpec{
			CompartmentId: testCompartmentID,
			RepositoryId:  testRepositoryID,
			ArtifactPath:  testArtifactPath,
			Version:       testVersion,
			Content: shared.SecretSource{
				SecretName: "artifact-content",
			},
			ContentKey: genericArtifactContentByPathDefaultContentKey,
		},
	}
}

func testGenericArtifactContentByPathRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: "artifact-content", Namespace: "default"}}
}

func genericArtifactContentByPathGetResponse(etag string, content string) genericartifactscontentsdk.GetGenericArtifactContentByPathResponse {
	return genericartifactscontentsdk.GetGenericArtifactContentByPathResponse{
		Content: io.NopCloser(strings.NewReader(content)),
		Etag:    common.String(etag),
	}
}

func genericArtifactContentByPathPutResponse(
	etag string,
	opcRequestID string,
	artifact genericartifactscontentsdk.GenericArtifact,
) genericartifactscontentsdk.PutGenericArtifactContentByPathResponse {
	return genericartifactscontentsdk.PutGenericArtifactContentByPathResponse{
		GenericArtifact: artifact,
		Etag:            common.String(etag),
		OpcRequestId:    common.String(opcRequestID),
	}
}

func genericArtifactContentByPathDeleteResponse(opcRequestID string) artifactssdk.DeleteGenericArtifactByPathResponse {
	return artifactssdk.DeleteGenericArtifactByPathResponse{
		OpcRequestId: common.String(opcRequestID),
	}
}

func genericArtifactContentByPathListResponse(nextPage string, items ...artifactssdk.GenericArtifactSummary) artifactssdk.ListGenericArtifactsResponse {
	response := artifactssdk.ListGenericArtifactsResponse{
		GenericArtifactCollection: artifactssdk.GenericArtifactCollection{Items: items},
	}
	if nextPage != "" {
		response.OpcNextPage = common.String(nextPage)
	}
	return response
}

func sdkGenericArtifactContentByPath(id string) genericartifactscontentsdk.GenericArtifact {
	size := int64(len(testContent))
	return genericartifactscontentsdk.GenericArtifact{
		Id:             common.String(id),
		DisplayName:    common.String(testArtifactPath + ":" + testVersion),
		CompartmentId:  common.String(testCompartmentID),
		RepositoryId:   common.String(testRepositoryID),
		ArtifactPath:   common.String(testArtifactPath),
		Version:        common.String(testVersion),
		Sha256:         common.String("sha256"),
		SizeInBytes:    &size,
		LifecycleState: genericartifactscontentsdk.GenericArtifactLifecycleStateAvailable,
	}
}

func sdkGenericArtifactContentByPathSummary(
	id string,
	artifactPath string,
	version string,
	lifecycleState artifactssdk.GenericArtifactLifecycleStateEnum,
) artifactssdk.GenericArtifactSummary {
	size := int64(len(testContent))
	return artifactssdk.GenericArtifactSummary{
		Id:             common.String(id),
		DisplayName:    common.String(artifactPath + ":" + version),
		CompartmentId:  common.String(testCompartmentID),
		RepositoryId:   common.String(testRepositoryID),
		ArtifactPath:   common.String(artifactPath),
		Version:        common.String(version),
		Sha256:         common.String("sha256"),
		SizeInBytes:    &size,
		LifecycleState: lifecycleState,
	}
}

func requireGenericArtifactContentByPathStatus(
	t *testing.T,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	id string,
	compartmentID string,
	repositoryID string,
	artifactPath string,
	version string,
) {
	t.Helper()
	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if got := resource.Status.CompartmentId; got != compartmentID {
		t.Fatalf("status.compartmentId = %q, want %q", got, compartmentID)
	}
	if got := resource.Status.RepositoryId; got != repositoryID {
		t.Fatalf("status.repositoryId = %q, want %q", got, repositoryID)
	}
	if got := resource.Status.ArtifactPath; got != artifactPath {
		t.Fatalf("status.artifactPath = %q, want %q", got, artifactPath)
	}
	if got := resource.Status.Version; got != version {
		t.Fatalf("status.version = %q, want %q", got, version)
	}
	if got := resource.Status.LifecycleState; got != string(genericartifactscontentsdk.GenericArtifactLifecycleStateAvailable) {
		t.Fatalf("status.lifecycleState = %q, want AVAILABLE", got)
	}
}

func requireGenericArtifactContentByPathCondition(
	t *testing.T,
	resource *genericartifactscontentv1beta1.GenericArtifactContentByPath,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = nil, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %s, want %s", got, want)
	}
}

func requireGenericArtifactContentByPathPutRequest(
	t *testing.T,
	request genericartifactscontentsdk.PutGenericArtifactContentByPathRequest,
	ifMatch string,
) {
	t.Helper()
	if got := stringValue(request.RepositoryId); got != testRepositoryID {
		t.Fatalf("put repositoryId = %q, want %q", got, testRepositoryID)
	}
	if got := stringValue(request.ArtifactPath); got != testArtifactPath {
		t.Fatalf("put artifactPath = %q, want %q", got, testArtifactPath)
	}
	if got := stringValue(request.Version); got != testVersion {
		t.Fatalf("put version = %q, want %q", got, testVersion)
	}
	if got := stringValue(request.IfMatch); got != ifMatch {
		t.Fatalf("put ifMatch = %q, want %q", got, ifMatch)
	}
}

type fakeGenericArtifactContentByPathOCIClient struct {
	getRequests    []genericartifactscontentsdk.GetGenericArtifactContentByPathRequest
	getResults     []genericArtifactContentByPathGetResult
	putRequests    []genericartifactscontentsdk.PutGenericArtifactContentByPathRequest
	putBodies      [][]byte
	putResults     []genericArtifactContentByPathPutResult
	deleteRequests []artifactssdk.DeleteGenericArtifactByPathRequest
	deleteResults  []genericArtifactContentByPathDeleteResult
	listRequests   []artifactssdk.ListGenericArtifactsRequest
	listResults    []genericArtifactContentByPathListResult
}

type genericArtifactContentByPathGetResult struct {
	response genericartifactscontentsdk.GetGenericArtifactContentByPathResponse
	err      error
}

type genericArtifactContentByPathPutResult struct {
	response genericartifactscontentsdk.PutGenericArtifactContentByPathResponse
	err      error
}

type genericArtifactContentByPathDeleteResult struct {
	response artifactssdk.DeleteGenericArtifactByPathResponse
	err      error
}

type genericArtifactContentByPathListResult struct {
	response artifactssdk.ListGenericArtifactsResponse
	err      error
}

func (f *fakeGenericArtifactContentByPathOCIClient) GetGenericArtifactContentByPath(
	_ context.Context,
	request genericartifactscontentsdk.GetGenericArtifactContentByPathRequest,
) (genericartifactscontentsdk.GetGenericArtifactContentByPathResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) == 0 {
		return genericartifactscontentsdk.GetGenericArtifactContentByPathResponse{}, fmt.Errorf("unexpected GetGenericArtifactContentByPath call")
	}
	result := f.getResults[0]
	f.getResults = f.getResults[1:]
	return result.response, result.err
}

func (f *fakeGenericArtifactContentByPathOCIClient) PutGenericArtifactContentByPath(
	_ context.Context,
	request genericartifactscontentsdk.PutGenericArtifactContentByPathRequest,
) (genericartifactscontentsdk.PutGenericArtifactContentByPathResponse, error) {
	f.putRequests = append(f.putRequests, request)
	if request.GenericArtifactContentBody != nil {
		content, err := io.ReadAll(request.GenericArtifactContentBody)
		if err != nil {
			return genericartifactscontentsdk.PutGenericArtifactContentByPathResponse{}, err
		}
		f.putBodies = append(f.putBodies, append([]byte(nil), content...))
	} else {
		f.putBodies = append(f.putBodies, nil)
	}
	if len(f.putResults) == 0 {
		return genericartifactscontentsdk.PutGenericArtifactContentByPathResponse{}, fmt.Errorf("unexpected PutGenericArtifactContentByPath call")
	}
	result := f.putResults[0]
	f.putResults = f.putResults[1:]
	return result.response, result.err
}

func (f *fakeGenericArtifactContentByPathOCIClient) DeleteGenericArtifactByPath(
	_ context.Context,
	request artifactssdk.DeleteGenericArtifactByPathRequest,
) (artifactssdk.DeleteGenericArtifactByPathResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if len(f.deleteResults) == 0 {
		return artifactssdk.DeleteGenericArtifactByPathResponse{}, fmt.Errorf("unexpected DeleteGenericArtifactByPath call")
	}
	result := f.deleteResults[0]
	f.deleteResults = f.deleteResults[1:]
	return result.response, result.err
}

func (f *fakeGenericArtifactContentByPathOCIClient) ListGenericArtifacts(
	_ context.Context,
	request artifactssdk.ListGenericArtifactsRequest,
) (artifactssdk.ListGenericArtifactsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResults) == 0 {
		return artifactssdk.ListGenericArtifactsResponse{}, fmt.Errorf("unexpected ListGenericArtifacts call")
	}
	result := f.listResults[0]
	f.listResults = f.listResults[1:]
	return result.response, result.err
}

type fakeGenericArtifactContentByPathCredentialClient struct {
	secrets map[string]map[string][]byte
}

func (f fakeGenericArtifactContentByPathCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return false, nil
}

func (f fakeGenericArtifactContentByPathCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return false, nil
}

func (f fakeGenericArtifactContentByPathCredentialClient) GetSecret(_ context.Context, name string, namespace string) (map[string][]byte, error) {
	if secret, ok := f.secrets[namespace+"/"+name]; ok {
		copied := make(map[string][]byte, len(secret))
		for key, value := range secret {
			copied[key] = bytes.Clone(value)
		}
		return copied, nil
	}
	return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
}

func (f fakeGenericArtifactContentByPathCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return false, nil
}

var _ GenericArtifactContentByPathServiceClient = (*genericArtifactContentByPathRuntimeClient)(nil)
var _ interface {
	DeleteWithResult(context.Context, *genericartifactscontentv1beta1.GenericArtifactContentByPath) (servicemanager.OSOKDeleteResult, error)
} = (*genericArtifactContentByPathRuntimeClient)(nil)
var _ genericArtifactContentByPathContentClient = (*fakeGenericArtifactContentByPathOCIClient)(nil)
var _ genericArtifactContentByPathArtifactClient = (*fakeGenericArtifactContentByPathOCIClient)(nil)
