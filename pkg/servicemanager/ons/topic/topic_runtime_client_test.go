/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package topic

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	onssdk "github.com/oracle/oci-go-sdk/v65/ons"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testTopicID          = "ocid1.onstopic.oc1..topic"
	testTopicCompartment = "ocid1.compartment.oc1..topic"
	testTopicName        = "topic-sample"
)

type fakeTopicOCIClient struct {
	createFn func(context.Context, onssdk.CreateTopicRequest) (onssdk.CreateTopicResponse, error)
	getFn    func(context.Context, onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error)
	listFn   func(context.Context, onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error)
	updateFn func(context.Context, onssdk.UpdateTopicRequest) (onssdk.UpdateTopicResponse, error)
	deleteFn func(context.Context, onssdk.DeleteTopicRequest) (onssdk.DeleteTopicResponse, error)
}

type erroringTopicConfigProvider struct{}

func (erroringTopicConfigProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	return nil, errors.New("topic provider invalid")
}

func (erroringTopicConfigProvider) KeyID() (string, error) {
	return "", errors.New("topic provider invalid")
}

func (erroringTopicConfigProvider) TenancyOCID() (string, error) {
	return "", errors.New("topic provider invalid")
}

func (erroringTopicConfigProvider) UserOCID() (string, error) {
	return "", errors.New("topic provider invalid")
}

func (erroringTopicConfigProvider) KeyFingerprint() (string, error) {
	return "", errors.New("topic provider invalid")
}

func (erroringTopicConfigProvider) Region() (string, error) {
	return "", errors.New("topic provider invalid")
}

func (erroringTopicConfigProvider) AuthType() (common.AuthConfig, error) {
	return common.AuthConfig{}, nil
}

func (f *fakeTopicOCIClient) CreateTopic(ctx context.Context, req onssdk.CreateTopicRequest) (onssdk.CreateTopicResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return onssdk.CreateTopicResponse{}, nil
}

func (f *fakeTopicOCIClient) GetTopic(ctx context.Context, req onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return onssdk.GetTopicResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "topic is missing")
}

func (f *fakeTopicOCIClient) ListTopics(ctx context.Context, req onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return onssdk.ListTopicsResponse{}, nil
}

func (f *fakeTopicOCIClient) UpdateTopic(ctx context.Context, req onssdk.UpdateTopicRequest) (onssdk.UpdateTopicResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return onssdk.UpdateTopicResponse{}, nil
}

func (f *fakeTopicOCIClient) DeleteTopic(ctx context.Context, req onssdk.DeleteTopicRequest) (onssdk.DeleteTopicResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return onssdk.DeleteTopicResponse{}, nil
}

func newTestTopicClient(client topicOCIClient) TopicServiceClient {
	return newTopicServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}, client)
}

func makeTopicResource() *onsv1beta1.Topic {
	return &onsv1beta1.Topic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testTopicName,
			Namespace: "default",
		},
		Spec: onsv1beta1.TopicSpec{
			Name:          testTopicName,
			CompartmentId: testTopicCompartment,
			Description:   "initial description",
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeTopicRequest(resource *onsv1beta1.Topic) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKTopic(id string, spec onsv1beta1.TopicSpec, state onssdk.NotificationTopicLifecycleStateEnum) onssdk.NotificationTopic {
	return onssdk.NotificationTopic{
		TopicId:        common.String(id),
		Name:           common.String(spec.Name),
		CompartmentId:  common.String(spec.CompartmentId),
		Description:    common.String(spec.Description),
		LifecycleState: state,
		ApiEndpoint:    common.String("https://ons.example.com"),
		FreeformTags:   cloneTopicStringMap(spec.FreeformTags),
		DefinedTags:    topicDefinedTags(spec.DefinedTags),
	}
}

func makeSDKTopicSummary(id string, spec onsv1beta1.TopicSpec, state onssdk.NotificationTopicSummaryLifecycleStateEnum) onssdk.NotificationTopicSummary {
	return onssdk.NotificationTopicSummary{
		TopicId:        common.String(id),
		Name:           common.String(spec.Name),
		CompartmentId:  common.String(spec.CompartmentId),
		Description:    common.String(spec.Description),
		LifecycleState: state,
		ApiEndpoint:    common.String("https://ons.example.com"),
		FreeformTags:   cloneTopicStringMap(spec.FreeformTags),
		DefinedTags:    topicDefinedTags(spec.DefinedTags),
	}
}

func topicDeleteGetResponse(resource *onsv1beta1.Topic, call int) (onssdk.GetTopicResponse, error) {
	switch call {
	case 1:
		return onssdk.GetTopicResponse{
			NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateActive),
		}, nil
	case 2:
		return onssdk.GetTopicResponse{
			NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateDeleting),
		}, nil
	default:
		return onssdk.GetTopicResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "topic is gone")
	}
}

func TestTopicCreateOrUpdateBindsExistingTopicByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()
	createCalled := false
	updateCalled := false
	getCalls := 0
	listCalls := 0

	client := newTestTopicClient(&fakeTopicOCIClient{
		listFn: func(_ context.Context, req onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListTopicsRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListTopicsRequest.Name", req.Name, resource.Spec.Name)
			if listCalls == 1 {
				if req.Page != nil {
					t.Fatalf("first ListTopicsRequest.Page = %q, want nil", *req.Page)
				}
				return onssdk.ListTopicsResponse{
					Items: []onssdk.NotificationTopicSummary{
						makeSDKTopicSummary("ocid1.onstopic.oc1..other", onsv1beta1.TopicSpec{
							Name:          "other-topic",
							CompartmentId: resource.Spec.CompartmentId,
						}, onssdk.NotificationTopicSummaryLifecycleStateActive),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second ListTopicsRequest.Page", req.Page, "page-2")
			return onssdk.ListTopicsResponse{
				Items: []onssdk.NotificationTopicSummary{
					makeSDKTopicSummary(testTopicID, resource.Spec, onssdk.NotificationTopicSummaryLifecycleStateActive),
				},
			}, nil
		},
		getFn: func(_ context.Context, req onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
			getCalls++
			requireStringPtr(t, "GetTopicRequest.TopicId", req.TopicId, testTopicID)
			return onssdk.GetTopicResponse{
				NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, onssdk.CreateTopicRequest) (onssdk.CreateTopicResponse, error) {
			createCalled = true
			return onssdk.CreateTopicResponse{}, nil
		},
		updateFn: func(context.Context, onssdk.UpdateTopicRequest) (onssdk.UpdateTopicResponse, error) {
			updateCalled = true
			return onssdk.UpdateTopicResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTopicRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateTopic() called for existing topic")
	}
	if updateCalled {
		t.Fatal("UpdateTopic() called for matching topic")
	}
	if listCalls != 2 {
		t.Fatalf("ListTopics() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetTopic() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testTopicID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testTopicID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestTopicCreateRecordsOCIDRetryTokenRequestIDAndLifecycle(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()
	createCalls := 0
	listCalls := 0

	client := newTestTopicClient(&fakeTopicOCIClient{
		listFn: func(_ context.Context, req onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListTopicsRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListTopicsRequest.Name", req.Name, resource.Spec.Name)
			return onssdk.ListTopicsResponse{}, nil
		},
		createFn: func(_ context.Context, req onssdk.CreateTopicRequest) (onssdk.CreateTopicResponse, error) {
			createCalls++
			requireTopicCreateRequest(t, req, resource)
			return onssdk.CreateTopicResponse{
				NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateCreating),
				OpcRequestId:      common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, req onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
			requireStringPtr(t, "GetTopicRequest.TopicId", req.TopicId, testTopicID)
			return onssdk.GetTopicResponse{
				NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateCreating),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTopicRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = false, want true for CREATING topic")
	}
	if createCalls != 1 {
		t.Fatalf("CreateTopic() calls = %d, want 1", createCalls)
	}
	if listCalls != 1 {
		t.Fatalf("ListTopics() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testTopicID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testTopicID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireTopicCreatePendingStatus(t, resource)
}

func TestTopicCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTopicID)
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := resource.Spec
	currentSpec.Description = "initial description"
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	getCalls := 0
	updateCalls := 0

	client := newTestTopicClient(&fakeTopicOCIClient{
		getFn: func(_ context.Context, req onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
			getCalls++
			requireStringPtr(t, "GetTopicRequest.TopicId", req.TopicId, testTopicID)
			if getCalls == 1 {
				return onssdk.GetTopicResponse{
					NotificationTopic: makeSDKTopic(testTopicID, currentSpec, onssdk.NotificationTopicLifecycleStateActive),
				}, nil
			}
			return onssdk.GetTopicResponse{
				NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req onssdk.UpdateTopicRequest) (onssdk.UpdateTopicResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateTopicRequest.TopicId", req.TopicId, testTopicID)
			requireStringPtr(t, "TopicAttributesDetails.Description", req.Description, resource.Spec.Description)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("TopicAttributesDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("TopicAttributesDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return onssdk.UpdateTopicResponse{
				NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateActive),
				OpcRequestId:      common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTopicRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateTopic() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetTopic() calls = %d, want current read and update follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestTopicCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTopicID)
	resource.Spec.Name = "renamed-topic"
	currentSpec := resource.Spec
	currentSpec.Name = testTopicName
	updateCalled := false

	client := newTestTopicClient(&fakeTopicOCIClient{
		getFn: func(_ context.Context, req onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
			requireStringPtr(t, "GetTopicRequest.TopicId", req.TopicId, testTopicID)
			return onssdk.GetTopicResponse{
				NotificationTopic: makeSDKTopic(testTopicID, currentSpec, onssdk.NotificationTopicLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, onssdk.UpdateTopicRequest) (onssdk.UpdateTopicResponse, error) {
			updateCalled = true
			return onssdk.UpdateTopicResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTopicRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateTopic() called after create-only name drift")
	}
	if !strings.Contains(err.Error(), "name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want name force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestTopicDeleteKeepsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTopicID)
	getCalls := 0
	deleteCalls := 0

	client := newTestTopicClient(&fakeTopicOCIClient{
		getFn: func(_ context.Context, req onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
			getCalls++
			requireStringPtr(t, "GetTopicRequest.TopicId", req.TopicId, testTopicID)
			return topicDeleteGetResponse(resource, getCalls)
		},
		deleteFn: func(_ context.Context, req onssdk.DeleteTopicRequest) (onssdk.DeleteTopicResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteTopicRequest.TopicId", req.TopicId, testTopicID)
			return onssdk.DeleteTopicResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while OCI reports DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTopic() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireLastCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteTopic() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestTopicDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testTopicID)

	client := newTestTopicClient(&fakeTopicOCIClient{
		getFn: func(_ context.Context, req onssdk.GetTopicRequest) (onssdk.GetTopicResponse, error) {
			requireStringPtr(t, "GetTopicRequest.TopicId", req.TopicId, testTopicID)
			return onssdk.GetTopicResponse{
				NotificationTopic: makeSDKTopic(testTopicID, resource.Spec, onssdk.NotificationTopicLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req onssdk.DeleteTopicRequest) (onssdk.DeleteTopicResponse, error) {
			requireStringPtr(t, "DeleteTopicRequest.TopicId", req.TopicId, testTopicID)
			return onssdk.DeleteTopicResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
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

func TestTopicCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()

	client := newTestTopicClient(&fakeTopicOCIClient{
		listFn: func(context.Context, onssdk.ListTopicsRequest) (onssdk.ListTopicsResponse, error) {
			return onssdk.ListTopicsResponse{}, nil
		},
		createFn: func(context.Context, onssdk.CreateTopicRequest) (onssdk.CreateTopicResponse, error) {
			return onssdk.CreateTopicResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTopicRequest(resource))
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

func TestTopicServiceClientPreservesGeneratedOCIInitErrorWhenWrapped(t *testing.T) {
	t.Parallel()

	resource := makeTopicResource()
	manager := &TopicServiceManager{
		Provider: erroringTopicConfigProvider{},
		Log:      loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
	client := newTopicServiceClient(manager)

	response, err := client.CreateOrUpdate(context.Background(), resource, makeTopicRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI client initialization error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if !strings.Contains(err.Error(), "initialize Topic OCI client") {
		t.Fatalf("CreateOrUpdate() error = %v, want Topic OCI client initialization failure", err)
	}
	if !strings.Contains(err.Error(), "topic provider invalid") {
		t.Fatalf("CreateOrUpdate() error = %v, want provider failure detail", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireTopicCreateRequest(t *testing.T, req onssdk.CreateTopicRequest, resource *onsv1beta1.Topic) {
	t.Helper()
	requireStringPtr(t, "CreateTopicDetails.Name", req.Name, resource.Spec.Name)
	requireStringPtr(t, "CreateTopicDetails.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateTopicDetails.Description", req.Description, resource.Spec.Description)
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateTopicRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireTopicCreatePendingStatus(t *testing.T, resource *onsv1beta1.Topic) {
	t.Helper()
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle create tracker")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceLifecycle || current.Phase != shared.OSOKAsyncPhaseCreate || current.RawStatus != "CREATING" || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want lifecycle create pending CREATING", current)
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

func requireLastCondition(t *testing.T, resource *onsv1beta1.Topic, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
