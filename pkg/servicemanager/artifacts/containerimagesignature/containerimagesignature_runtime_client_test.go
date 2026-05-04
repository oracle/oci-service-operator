/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerimagesignature

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	artifactssdk "github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testContainerImageSignatureID           = "ocid1.containerimagesignature.oc1..signature"
	testContainerImageSignatureCompartment  = "ocid1.compartment.oc1..signature"
	testContainerImageSignatureImage        = "ocid1.containerimage.oc1..signature"
	testContainerImageSignatureKMSKey       = "ocid1.key.oc1..signature"
	testContainerImageSignatureKMSKeyVer    = "ocid1.keyversion.oc1..signature"
	testContainerImageSignatureMessage      = "signed-message"
	testContainerImageSignatureSignature    = "signature-payload"
	testContainerImageSignatureAlgorithm    = "SHA_256_RSA_PKCS_PSS"
	testContainerImageSignatureDisplayName  = "keyversion::SHA_256_RSA_PKCS_PSS::signature"
	testContainerImageSignatureCreatedBy    = "ocid1.user.oc1..signature"
	testContainerImageSignatureOtherID      = "ocid1.containerimagesignature.oc1..other"
	testContainerImageSignatureUpdateOpcID  = "opc-update-signature"
	testContainerImageSignatureCreateOpcID  = "opc-create-signature"
	testContainerImageSignatureDeleteOpcID  = "opc-delete-signature"
	testContainerImageSignatureConflictCode = "Conflict"
)

type fakeContainerImageSignatureOCIClient struct {
	createFunc func(context.Context, artifactssdk.CreateContainerImageSignatureRequest) (artifactssdk.CreateContainerImageSignatureResponse, error)
	getFunc    func(context.Context, artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error)
	listFunc   func(context.Context, artifactssdk.ListContainerImageSignaturesRequest) (artifactssdk.ListContainerImageSignaturesResponse, error)
	updateFunc func(context.Context, artifactssdk.UpdateContainerImageSignatureRequest) (artifactssdk.UpdateContainerImageSignatureResponse, error)
	deleteFunc func(context.Context, artifactssdk.DeleteContainerImageSignatureRequest) (artifactssdk.DeleteContainerImageSignatureResponse, error)

	createRequests []artifactssdk.CreateContainerImageSignatureRequest
	getRequests    []artifactssdk.GetContainerImageSignatureRequest
	listRequests   []artifactssdk.ListContainerImageSignaturesRequest
	updateRequests []artifactssdk.UpdateContainerImageSignatureRequest
	deleteRequests []artifactssdk.DeleteContainerImageSignatureRequest
}

func (f *fakeContainerImageSignatureOCIClient) CreateContainerImageSignature(ctx context.Context, request artifactssdk.CreateContainerImageSignatureRequest) (artifactssdk.CreateContainerImageSignatureResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return artifactssdk.CreateContainerImageSignatureResponse{}, nil
}

func (f *fakeContainerImageSignatureOCIClient) GetContainerImageSignature(ctx context.Context, request artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return artifactssdk.GetContainerImageSignatureResponse{}, nil
}

func (f *fakeContainerImageSignatureOCIClient) ListContainerImageSignatures(ctx context.Context, request artifactssdk.ListContainerImageSignaturesRequest) (artifactssdk.ListContainerImageSignaturesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return artifactssdk.ListContainerImageSignaturesResponse{}, nil
}

func (f *fakeContainerImageSignatureOCIClient) UpdateContainerImageSignature(ctx context.Context, request artifactssdk.UpdateContainerImageSignatureRequest) (artifactssdk.UpdateContainerImageSignatureResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return artifactssdk.UpdateContainerImageSignatureResponse{}, nil
}

func (f *fakeContainerImageSignatureOCIClient) DeleteContainerImageSignature(ctx context.Context, request artifactssdk.DeleteContainerImageSignatureRequest) (artifactssdk.DeleteContainerImageSignatureResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return artifactssdk.DeleteContainerImageSignatureResponse{}, nil
}

func TestContainerImageSignatureRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	semantics := newContainerImageSignatureRuntimeSemantics()

	if semantics.FormalService != "artifacts" || semantics.FormalSlug != "containerimagesignature" {
		t.Fatalf("formal identity = %s/%s, want artifacts/containerimagesignature", semantics.FormalService, semantics.FormalSlug)
	}
	if semantics.Async == nil || semantics.Async.Strategy != "lifecycle" || semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", semantics.Async)
	}
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", semantics.FinalizerPolicy)
	}
	assertStringSliceContains(t, "Lifecycle.ActiveStates", semantics.Lifecycle.ActiveStates, "AVAILABLE")
	assertStringSliceContains(t, "Delete.PendingStates", semantics.Delete.PendingStates, "DELETING")
	assertStringSliceContains(t, "Delete.TerminalStates", semantics.Delete.TerminalStates, "DELETED")
	assertStringSliceContains(t, "Mutation.Mutable", semantics.Mutation.Mutable, "freeformTags")
	assertStringSliceContains(t, "Mutation.Mutable", semantics.Mutation.Mutable, "definedTags")
	assertStringSliceContains(t, "Mutation.ForceNew", semantics.Mutation.ForceNew, "kmsKeyId")
	if semantics.List == nil {
		t.Fatal("list semantics = nil")
	}
	assertStringSliceContains(t, "List.MatchFields", semantics.List.MatchFields, "message")
	assertStringSliceContains(t, "List.MatchFields", semantics.List.MatchFields, "signature")
}

//nolint:gocognit,gocyclo // This end-to-end create case verifies request shaping and status projection together.
func TestContainerImageSignatureCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newContainerImageSignatureResource()
	current := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	fake := &fakeContainerImageSignatureOCIClient{
		createFunc: func(_ context.Context, request artifactssdk.CreateContainerImageSignatureRequest) (artifactssdk.CreateContainerImageSignatureResponse, error) {
			details := request.CreateContainerImageSignatureDetails
			if got := stringPtrValue(details.CompartmentId); got != testContainerImageSignatureCompartment {
				t.Fatalf("create compartmentId = %q, want %q", got, testContainerImageSignatureCompartment)
			}
			if got := stringPtrValue(details.ImageId); got != testContainerImageSignatureImage {
				t.Fatalf("create imageId = %q, want %q", got, testContainerImageSignatureImage)
			}
			if got := stringPtrValue(details.KmsKeyId); got != testContainerImageSignatureKMSKey {
				t.Fatalf("create kmsKeyId = %q, want %q", got, testContainerImageSignatureKMSKey)
			}
			if got := stringPtrValue(details.Message); got != testContainerImageSignatureMessage {
				t.Fatalf("create message = %q, want configured message", got)
			}
			if got := string(details.SigningAlgorithm); got != testContainerImageSignatureAlgorithm {
				t.Fatalf("create signingAlgorithm = %q, want %q", got, testContainerImageSignatureAlgorithm)
			}
			if got := details.FreeformTags["env"]; got != "test" {
				t.Fatalf("create freeform env tag = %q, want test", got)
			}
			if got := details.DefinedTags["Operations"]["CostCenter"]; got != "42" {
				t.Fatalf("create defined tag = %#v, want 42", got)
			}
			return artifactssdk.CreateContainerImageSignatureResponse{
				ContainerImageSignature: current,
				OpcRequestId:            common.String(testContainerImageSignatureCreateOpcID),
			}, nil
		},
		getFunc: func(_ context.Context, request artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			if got := stringPtrValue(request.ImageSignatureId); got != testContainerImageSignatureID {
				t.Fatalf("get imageSignatureId = %q, want %q", got, testContainerImageSignatureID)
			}
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: current}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is not successful")
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(fake.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testContainerImageSignatureID {
		t.Fatalf("status ocid = %q, want %q", got, testContainerImageSignatureID)
	}
	if resource.Status.Id != testContainerImageSignatureID {
		t.Fatalf("status id = %q, want %q", resource.Status.Id, testContainerImageSignatureID)
	}
	if resource.Status.LifecycleState != "AVAILABLE" {
		t.Fatalf("status lifecycleState = %q, want AVAILABLE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != testContainerImageSignatureCreateOpcID {
		t.Fatalf("status opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testContainerImageSignatureCreateOpcID)
	}
}

//nolint:gocyclo // This case keeps the paginated bind request and response assertions together.
func TestContainerImageSignatureCreateOrUpdateBindsFromPaginatedList(t *testing.T) {
	resource := newContainerImageSignatureResource()
	matching := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	fake := &fakeContainerImageSignatureOCIClient{
		listFunc: func(_ context.Context, request artifactssdk.ListContainerImageSignaturesRequest) (artifactssdk.ListContainerImageSignaturesResponse, error) {
			if got := stringPtrValue(request.CompartmentId); got != testContainerImageSignatureCompartment {
				t.Fatalf("list compartmentId = %q, want %q", got, testContainerImageSignatureCompartment)
			}
			if got := string(request.SigningAlgorithm); got != testContainerImageSignatureAlgorithm {
				t.Fatalf("list signingAlgorithm = %q, want %q", got, testContainerImageSignatureAlgorithm)
			}
			if request.Page == nil {
				return artifactssdk.ListContainerImageSignaturesResponse{
					ContainerImageSignatureCollection: artifactssdk.ContainerImageSignatureCollection{
						Items: []artifactssdk.ContainerImageSignatureSummary{
							newSDKContainerImageSignatureSummary(testContainerImageSignatureOtherID, "other-message"),
						},
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			}
			return artifactssdk.ListContainerImageSignaturesResponse{
				ContainerImageSignatureCollection: artifactssdk.ContainerImageSignatureCollection{
					Items: []artifactssdk.ContainerImageSignatureSummary{containerImageSignatureSummaryFromSDK(matching)},
				},
			}, nil
		},
		getFunc: func(_ context.Context, request artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			if got := stringPtrValue(request.ImageSignatureId); got != testContainerImageSignatureID {
				t.Fatalf("get imageSignatureId = %q, want %q", got, testContainerImageSignatureID)
			}
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: matching}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is not successful")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for list bind", len(fake.createRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 paginated requests", len(fake.listRequests))
	}
	if fake.listRequests[0].Page != nil {
		t.Fatalf("first list page = %q, want nil", stringPtrValue(fake.listRequests[0].Page))
	}
	if got := stringPtrValue(fake.listRequests[1].Page); got != "next-page" {
		t.Fatalf("second list page = %q, want next-page", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testContainerImageSignatureID {
		t.Fatalf("status ocid = %q, want %q", got, testContainerImageSignatureID)
	}
}

func TestContainerImageSignatureCreateOrUpdateBindsWithLowerCaseSigningAlgorithm(t *testing.T) {
	resource := newContainerImageSignatureResource()
	originalAlgorithm := strings.ToLower(testContainerImageSignatureAlgorithm)
	resource.Spec.SigningAlgorithm = originalAlgorithm
	matching := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	fake := &fakeContainerImageSignatureOCIClient{
		listFunc: func(_ context.Context, request artifactssdk.ListContainerImageSignaturesRequest) (artifactssdk.ListContainerImageSignaturesResponse, error) {
			if got := string(request.SigningAlgorithm); got != testContainerImageSignatureAlgorithm {
				t.Fatalf("list signingAlgorithm = %q, want canonical %q", got, testContainerImageSignatureAlgorithm)
			}
			return artifactssdk.ListContainerImageSignaturesResponse{
				ContainerImageSignatureCollection: artifactssdk.ContainerImageSignatureCollection{
					Items: []artifactssdk.ContainerImageSignatureSummary{containerImageSignatureSummaryFromSDK(matching)},
				},
			}, nil
		},
		getFunc: func(_ context.Context, request artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			if got := stringPtrValue(request.ImageSignatureId); got != testContainerImageSignatureID {
				t.Fatalf("get imageSignatureId = %q, want %q", got, testContainerImageSignatureID)
			}
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: matching}, nil
		},
		createFunc: func(context.Context, artifactssdk.CreateContainerImageSignatureRequest) (artifactssdk.CreateContainerImageSignatureResponse, error) {
			t.Fatal("CreateContainerImageSignature should not be called when lower-case signingAlgorithm matches an existing signature")
			return artifactssdk.CreateContainerImageSignatureResponse{}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is not successful")
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for canonical list bind", len(fake.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testContainerImageSignatureID {
		t.Fatalf("status ocid = %q, want %q", got, testContainerImageSignatureID)
	}
	if resource.Spec.SigningAlgorithm != originalAlgorithm {
		t.Fatalf("spec signingAlgorithm = %q, want original %q restored", resource.Spec.SigningAlgorithm, originalAlgorithm)
	}
}

func TestContainerImageSignatureCreateOrUpdateRejectsDuplicateMatchesAcrossPages(t *testing.T) {
	resource := newContainerImageSignatureResource()
	first := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	second := newSDKContainerImageSignature("ocid1.containerimagesignature.oc1..duplicate", artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	fake := &fakeContainerImageSignatureOCIClient{
		listFunc: func(_ context.Context, request artifactssdk.ListContainerImageSignaturesRequest) (artifactssdk.ListContainerImageSignaturesResponse, error) {
			if request.Page == nil {
				return artifactssdk.ListContainerImageSignaturesResponse{
					ContainerImageSignatureCollection: artifactssdk.ContainerImageSignatureCollection{
						Items: []artifactssdk.ContainerImageSignatureSummary{containerImageSignatureSummaryFromSDK(first)},
					},
					OpcNextPage: common.String("next-page"),
				}, nil
			}
			return artifactssdk.ListContainerImageSignaturesResponse{
				ContainerImageSignatureCollection: artifactssdk.ContainerImageSignatureCollection{
					Items: []artifactssdk.ContainerImageSignatureSummary{containerImageSignatureSummaryFromSDK(second)},
				},
			}, nil
		},
		createFunc: func(context.Context, artifactssdk.CreateContainerImageSignatureRequest) (artifactssdk.CreateContainerImageSignatureResponse, error) {
			t.Fatal("CreateContainerImageSignature should not be called when list returns duplicate matches")
			return artifactssdk.CreateContainerImageSignatureResponse{}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate list match error", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is successful, want failure")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 paginated requests", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(fake.createRequests))
	}
}

func TestContainerImageSignatureCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	resource := newContainerImageSignatureResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerImageSignatureID)
	current := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	fake := &fakeContainerImageSignatureOCIClient{
		getFunc: func(_ context.Context, _ artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: current}, nil
		},
		updateFunc: func(context.Context, artifactssdk.UpdateContainerImageSignatureRequest) (artifactssdk.UpdateContainerImageSignatureResponse, error) {
			t.Fatal("UpdateContainerImageSignature should not be called when mutable state matches")
			return artifactssdk.UpdateContainerImageSignatureResponse{}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is not successful")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestContainerImageSignatureCreateOrUpdateSkipsUpdateWithLowerCaseSigningAlgorithm(t *testing.T) {
	resource := newContainerImageSignatureResource()
	originalAlgorithm := strings.ToLower(testContainerImageSignatureAlgorithm)
	resource.Spec.SigningAlgorithm = originalAlgorithm
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerImageSignatureID)
	current := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	fake := &fakeContainerImageSignatureOCIClient{
		getFunc: func(_ context.Context, _ artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: current}, nil
		},
		updateFunc: func(context.Context, artifactssdk.UpdateContainerImageSignatureRequest) (artifactssdk.UpdateContainerImageSignatureResponse, error) {
			t.Fatal("UpdateContainerImageSignature should not be called when only signingAlgorithm casing differs")
			return artifactssdk.UpdateContainerImageSignatureResponse{}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is not successful")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	if resource.Spec.SigningAlgorithm != originalAlgorithm {
		t.Fatalf("spec signingAlgorithm = %q, want original %q restored", resource.Spec.SigningAlgorithm, originalAlgorithm)
	}
}

func TestContainerImageSignatureCreateOrUpdateUpdatesMutableTags(t *testing.T) {
	resource := newContainerImageSignatureResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerImageSignatureID)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
	current := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	updated := current
	updated.FreeformTags = map[string]string{"env": "prod"}
	updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
	getCount := 0
	fake := &fakeContainerImageSignatureOCIClient{
		getFunc: func(_ context.Context, _ artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			getCount++
			if getCount == 1 {
				return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: current}, nil
			}
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: updated}, nil
		},
		updateFunc: func(_ context.Context, request artifactssdk.UpdateContainerImageSignatureRequest) (artifactssdk.UpdateContainerImageSignatureResponse, error) {
			if got := stringPtrValue(request.ImageSignatureId); got != testContainerImageSignatureID {
				t.Fatalf("update imageSignatureId = %q, want %q", got, testContainerImageSignatureID)
			}
			if !reflect.DeepEqual(request.FreeformTags, map[string]string{"env": "prod"}) {
				t.Fatalf("update freeformTags = %#v, want env=prod", request.FreeformTags)
			}
			if got := request.DefinedTags["Operations"]["CostCenter"]; got != "84" {
				t.Fatalf("update defined tag = %#v, want 84", got)
			}
			return artifactssdk.UpdateContainerImageSignatureResponse{
				ContainerImageSignature: updated,
				OpcRequestId:            common.String(testContainerImageSignatureUpdateOpcID),
			}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is not successful")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	if resource.Status.OsokStatus.OpcRequestID != testContainerImageSignatureUpdateOpcID {
		t.Fatalf("status opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testContainerImageSignatureUpdateOpcID)
	}
}

func TestContainerImageSignatureBuildUpdateBodyPreservesExplicitEmptyTagClears(t *testing.T) {
	resource := newContainerImageSignatureResource()
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	current := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)

	body, ok, err := buildContainerImageSignatureUpdateBody(resource, artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: current})
	if err != nil {
		t.Fatalf("buildContainerImageSignatureUpdateBody() error = %v", err)
	}
	if !ok {
		t.Fatal("buildContainerImageSignatureUpdateBody() ok = false, want tag clear update")
	}
	if body.FreeformTags == nil || len(body.FreeformTags) != 0 {
		t.Fatalf("freeformTags = %#v, want non-nil empty map", body.FreeformTags)
	}
	if body.DefinedTags == nil || len(body.DefinedTags) != 0 {
		t.Fatalf("definedTags = %#v, want non-nil empty map", body.DefinedTags)
	}
}

func TestContainerImageSignatureCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newContainerImageSignatureResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerImageSignatureID)
	resource.Spec.KmsKeyId = "ocid1.key.oc1..changed"
	current := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	fake := &fakeContainerImageSignatureOCIClient{
		getFunc: func(_ context.Context, _ artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: current}, nil
		},
		updateFunc: func(context.Context, artifactssdk.UpdateContainerImageSignatureRequest) (artifactssdk.UpdateContainerImageSignatureResponse, error) {
			t.Fatal("UpdateContainerImageSignature should not be called for create-only drift")
			return artifactssdk.UpdateContainerImageSignatureResponse{}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "kmsKeyId") {
		t.Fatalf("CreateOrUpdate() error = %v, want kmsKeyId force-new error", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is successful, want failure")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestContainerImageSignatureCreateOrUpdateRecordsErrorOpcRequestID(t *testing.T) {
	resource := newContainerImageSignatureResource()
	fake := &fakeContainerImageSignatureOCIClient{
		createFunc: func(context.Context, artifactssdk.CreateContainerImageSignatureRequest) (artifactssdk.CreateContainerImageSignatureResponse, error) {
			err := errortest.NewServiceError(409, testContainerImageSignatureConflictCode, "create conflict")
			err.OpcRequestID = "opc-create-error"
			return artifactssdk.CreateContainerImageSignatureResponse{}, err
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create conflict")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response is successful, want failure")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-error" {
		t.Fatalf("status opcRequestId = %q, want opc-create-error", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestContainerImageSignatureDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := newContainerImageSignatureResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerImageSignatureID)
	available := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	deleting := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateDeleting)
	getCount := 0
	fake := &fakeContainerImageSignatureOCIClient{
		getFunc: func(_ context.Context, _ artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			getCount++
			if getCount == 1 {
				return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: available}, nil
			}
			return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: deleting}, nil
		},
		deleteFunc: func(_ context.Context, request artifactssdk.DeleteContainerImageSignatureRequest) (artifactssdk.DeleteContainerImageSignatureResponse, error) {
			if got := stringPtrValue(request.ImageSignatureId); got != testContainerImageSignatureID {
				t.Fatalf("delete imageSignatureId = %q, want %q", got, testContainerImageSignatureID)
			}
			return artifactssdk.DeleteContainerImageSignatureResponse{OpcRequestId: common.String(testContainerImageSignatureDeleteOpcID)}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI reports DELETING")
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want Terminating", resource.Status.OsokStatus.Reason)
	}
	if resource.Status.OsokStatus.OpcRequestID != testContainerImageSignatureDeleteOpcID {
		t.Fatalf("status opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, testContainerImageSignatureDeleteOpcID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status deletedAt is set while delete is still pending")
	}
}

func TestContainerImageSignatureDeleteConfirmsDeletedOnUnambiguousNotFound(t *testing.T) {
	resource := newContainerImageSignatureResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerImageSignatureID)
	available := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	getCount := 0
	fake := &fakeContainerImageSignatureOCIClient{
		getFunc: func(_ context.Context, _ artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			getCount++
			if getCount == 1 {
				return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: available}, nil
			}
			return artifactssdk.GetContainerImageSignatureResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFunc: func(context.Context, artifactssdk.DeleteContainerImageSignatureRequest) (artifactssdk.DeleteContainerImageSignatureResponse, error) {
			return artifactssdk.DeleteContainerImageSignatureResponse{OpcRequestId: common.String(testContainerImageSignatureDeleteOpcID)}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status deletedAt = nil, want deletion timestamp")
	}
}

func TestContainerImageSignatureDeleteKeepsAuthShapedNotFoundFatal(t *testing.T) {
	resource := newContainerImageSignatureResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testContainerImageSignatureID)
	available := newSDKContainerImageSignature(testContainerImageSignatureID, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	getCount := 0
	fake := &fakeContainerImageSignatureOCIClient{
		getFunc: func(_ context.Context, _ artifactssdk.GetContainerImageSignatureRequest) (artifactssdk.GetContainerImageSignatureResponse, error) {
			getCount++
			if getCount == 1 {
				return artifactssdk.GetContainerImageSignatureResponse{ContainerImageSignature: available}, nil
			}
			return artifactssdk.GetContainerImageSignatureResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFunc: func(context.Context, artifactssdk.DeleteContainerImageSignatureRequest) (artifactssdk.DeleteContainerImageSignatureResponse, error) {
			return artifactssdk.DeleteContainerImageSignatureResponse{OpcRequestId: common.String(testContainerImageSignatureDeleteOpcID)}, nil
		},
	}
	client := newTestContainerImageSignatureServiceClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous not-found") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not found")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status deletedAt is set for auth-shaped not found")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status opcRequestId = %q, want auth error request id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func newTestContainerImageSignatureServiceClient(fake *fakeContainerImageSignatureOCIClient) ContainerImageSignatureServiceClient {
	hooks := ContainerImageSignatureRuntimeHooks{
		Create: runtimeOperationHooks[artifactssdk.CreateContainerImageSignatureRequest, artifactssdk.CreateContainerImageSignatureResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateContainerImageSignatureDetails", RequestName: "CreateContainerImageSignatureDetails", Contribution: "body"}},
			Call:   fake.CreateContainerImageSignature,
		},
		Get: runtimeOperationHooks[artifactssdk.GetContainerImageSignatureRequest, artifactssdk.GetContainerImageSignatureResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ImageSignatureId", RequestName: "imageSignatureId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.GetContainerImageSignature,
		},
		List: runtimeOperationHooks[artifactssdk.ListContainerImageSignaturesRequest, artifactssdk.ListContainerImageSignaturesResponse]{
			Fields: containerImageSignatureListFields(),
			Call:   fake.ListContainerImageSignatures,
		},
		Update: runtimeOperationHooks[artifactssdk.UpdateContainerImageSignatureRequest, artifactssdk.UpdateContainerImageSignatureResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ImageSignatureId", RequestName: "imageSignatureId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateContainerImageSignatureDetails", RequestName: "UpdateContainerImageSignatureDetails", Contribution: "body"},
			},
			Call: fake.UpdateContainerImageSignature,
		},
		Delete: runtimeOperationHooks[artifactssdk.DeleteContainerImageSignatureRequest, artifactssdk.DeleteContainerImageSignatureResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ImageSignatureId", RequestName: "imageSignatureId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.DeleteContainerImageSignature,
		},
	}
	applyContainerImageSignatureRuntimeHooks(&hooks)
	delegate := defaultContainerImageSignatureServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*artifactsv1beta1.ContainerImageSignature](
			buildContainerImageSignatureGeneratedRuntimeConfig(&ContainerImageSignatureServiceManager{}, hooks),
		),
	}
	return wrapContainerImageSignatureGeneratedClient(hooks, delegate)
}

func newContainerImageSignatureResource() *artifactsv1beta1.ContainerImageSignature {
	return &artifactsv1beta1.ContainerImageSignature{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "signature",
			Namespace: "default",
		},
		Spec: artifactsv1beta1.ContainerImageSignatureSpec{
			CompartmentId:    testContainerImageSignatureCompartment,
			ImageId:          testContainerImageSignatureImage,
			KmsKeyId:         testContainerImageSignatureKMSKey,
			KmsKeyVersionId:  testContainerImageSignatureKMSKeyVer,
			Message:          testContainerImageSignatureMessage,
			Signature:        testContainerImageSignatureSignature,
			SigningAlgorithm: testContainerImageSignatureAlgorithm,
			FreeformTags:     map[string]string{"env": "test"},
			DefinedTags:      map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func newSDKContainerImageSignature(id string, state artifactssdk.ContainerImageSignatureLifecycleStateEnum) artifactssdk.ContainerImageSignature {
	created := common.SDKTime{Time: time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)}
	return artifactssdk.ContainerImageSignature{
		CompartmentId:    common.String(testContainerImageSignatureCompartment),
		CreatedBy:        common.String(testContainerImageSignatureCreatedBy),
		DisplayName:      common.String(testContainerImageSignatureDisplayName),
		Id:               common.String(id),
		ImageId:          common.String(testContainerImageSignatureImage),
		KmsKeyId:         common.String(testContainerImageSignatureKMSKey),
		KmsKeyVersionId:  common.String(testContainerImageSignatureKMSKeyVer),
		Message:          common.String(testContainerImageSignatureMessage),
		Signature:        common.String(testContainerImageSignatureSignature),
		SigningAlgorithm: artifactssdk.ContainerImageSignatureSigningAlgorithm256RsaPkcsPss,
		TimeCreated:      &created,
		LifecycleState:   state,
		FreeformTags:     map[string]string{"env": "test"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func newSDKContainerImageSignatureSummary(id string, message string) artifactssdk.ContainerImageSignatureSummary {
	signature := newSDKContainerImageSignature(id, artifactssdk.ContainerImageSignatureLifecycleStateAvailable)
	signature.Message = common.String(message)
	return containerImageSignatureSummaryFromSDK(signature)
}

func containerImageSignatureSummaryFromSDK(signature artifactssdk.ContainerImageSignature) artifactssdk.ContainerImageSignatureSummary {
	return artifactssdk.ContainerImageSignatureSummary{
		CompartmentId:    signature.CompartmentId,
		DisplayName:      signature.DisplayName,
		Id:               signature.Id,
		ImageId:          signature.ImageId,
		KmsKeyId:         signature.KmsKeyId,
		KmsKeyVersionId:  signature.KmsKeyVersionId,
		Message:          signature.Message,
		Signature:        signature.Signature,
		SigningAlgorithm: artifactssdk.ContainerImageSignatureSummarySigningAlgorithmEnum(signature.SigningAlgorithm),
		TimeCreated:      signature.TimeCreated,
		LifecycleState:   signature.LifecycleState,
		FreeformTags:     signature.FreeformTags,
		DefinedTags:      signature.DefinedTags,
		SystemTags:       signature.SystemTags,
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertStringSliceContains(t *testing.T, name string, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%s = %#v, want %q", name, values, want)
}
