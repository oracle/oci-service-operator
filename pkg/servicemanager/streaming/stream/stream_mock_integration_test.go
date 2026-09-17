/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package stream

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	streamingsdk "github.com/oracle/oci-go-sdk/v65/streaming"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockStreamID = "ocid1.stream.oc1..mock"
const mockStreamUpdatedEndpoint = "https://messages-updated.mock.invalid"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/streaming/stream and formal/imports/streaming/stream.json
//   - endpoint Secret runtime: stream_endpoint_secret_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/streaming
func TestMockIntegrationStreamLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &streamingv1beta1.Stream{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-stream", Namespace: "default", UID: types.UID("mock-stream-uid")},
		Spec: streamingv1beta1.StreamSpec{
			Name:             "mock-stream",
			Partitions:       1,
			CompartmentId:    "ocid1.compartment.oc1..mock",
			RetentionInHours: 24,
			FreeformTags:     map[string]string{"osok-mock": "create"},
		},
	}
	secretRecord := credhelper.SecretRecord{}
	secretExists := false
	secretCreates := 0
	secretUpdates := 0
	secretDeletes := 0
	credentials := &fakeCredentialClient{}
	credentials.getSecretRecordFn = func(_ context.Context, name, namespace string) (credhelper.SecretRecord, error) {
		if name != resource.Name || namespace != resource.Namespace {
			return credhelper.SecretRecord{}, fmt.Errorf("read endpoint Secret %s/%s, want %s/%s", namespace, name, resource.Namespace, resource.Name)
		}
		if !secretExists {
			return credhelper.SecretRecord{}, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
		}
		return cloneMockStreamSecretRecord(secretRecord), nil
	}
	credentials.createSecretFn = func(_ context.Context, name, namespace string, labels map[string]string, data map[string][]byte) (bool, error) {
		if secretExists {
			return false, apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, name)
		}
		if name != resource.Name || namespace != resource.Namespace {
			return false, fmt.Errorf("create endpoint Secret %s/%s, want %s/%s", namespace, name, resource.Namespace, resource.Name)
		}
		secretRecord = credhelper.SecretRecord{
			UID:    types.UID("mock-stream-endpoint-secret-uid"),
			Labels: cloneMockStreamStringMap(labels),
			Data:   cloneMockStreamByteMap(data),
		}
		secretExists = true
		secretCreates++
		return true, nil
	}
	credentials.updateSecretIfCurrentFn = func(_ context.Context, name, namespace string, current credhelper.SecretRecord, labels map[string]string, data map[string][]byte) (bool, error) {
		if !secretExists {
			return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
		}
		if name != resource.Name || namespace != resource.Namespace || current.UID != secretRecord.UID {
			return false, fmt.Errorf("guarded endpoint Secret update did not target the current %s/%s record", resource.Namespace, resource.Name)
		}
		if labels != nil {
			secretRecord.Labels = cloneMockStreamStringMap(labels)
		}
		secretRecord.Data = cloneMockStreamByteMap(data)
		secretUpdates++
		return true, nil
	}
	credentials.deleteSecretIfCurrentFn = func(_ context.Context, name, namespace string, current credhelper.SecretRecord) (bool, error) {
		if !secretExists {
			return false, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
		}
		if name != resource.Name || namespace != resource.Namespace || current.UID != secretRecord.UID {
			return false, fmt.Errorf("guarded endpoint Secret delete did not target the current %s/%s record", resource.Namespace, resource.Name)
		}
		secretExists = false
		secretDeletes++
		return true, nil
	}
	responder, err := newStreamMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://streaming.mock.invalid", BasePath: "20180418", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Stream OCI mock: %v", err)
		}
	})

	sdkClient := streamingsdk.StreamAdminClient{BaseClient: session.BaseClient()}
	client, err := newMockStreamClient(sdkClient, credentials)
	if err != nil {
		t.Fatal(err)
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*streamingv1beta1.Stream]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *streamingv1beta1.Stream) error {
			if current.Status.Id != mockStreamID ||
				current.Status.Name != "mock-stream" ||
				current.Status.MessagesEndpoint != "https://messages.mock.invalid" ||
				current.Status.LifecycleState != string(streamingsdk.StreamLifecycleStateActive) {
				return fmt.Errorf("created Stream status = %+v", current.Status)
			}
			return validateMockStreamEndpointSecret(secretExists, secretRecord, current, "https://messages.mock.invalid")
		},
		Mutate: func(current *streamingv1beta1.Stream) {
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *streamingv1beta1.Stream) error {
			if current.Status.FreeformTags["osok-mock"] != "update" ||
				current.Status.MessagesEndpoint != mockStreamUpdatedEndpoint ||
				current.Status.LifecycleState != string(streamingsdk.StreamLifecycleStateActive) {
				return fmt.Errorf("updated Stream status = %+v", current.Status)
			}
			return validateMockStreamEndpointSecret(secretExists, secretRecord, current, mockStreamUpdatedEndpoint)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if secretExists || secretCreates != 1 || secretUpdates != 1 || secretDeletes != 1 {
		t.Fatalf("endpoint Secret lifecycle = exists:%t creates:%d updates:%d deletes:%d, want false/1/1/1", secretExists, secretCreates, secretUpdates, secretDeletes)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newStreamMockResponder(resource *streamingv1beta1.Stream) (*ocimock.CRUDResponder[streamingsdk.Stream], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createReadObserved := false
	updateReadObserved := false
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[streamingsdk.Stream]{
		CollectionPath:         "/20180418/streams",
		ItemPath:               "/20180418/streams/" + mockStreamID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (streamingsdk.Stream, ocimock.Response, error) {
			var details streamingsdk.CreateStreamDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return streamingsdk.Stream{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return streamingsdk.Stream{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name ||
				details.Partitions == nil || *details.Partitions != resource.Spec.Partitions ||
				details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.RetentionInHours == nil || *details.RetentionInHours != resource.Spec.RetentionInHours {
				return streamingsdk.Stream{}, ocimock.Response{}, fmt.Errorf("unexpected CreateStream details: %+v", details)
			}
			state := streamingsdk.Stream{
				Name:             details.Name,
				Id:               common.String(mockStreamID),
				Partitions:       details.Partitions,
				RetentionInHours: details.RetentionInHours,
				CompartmentId:    details.CompartmentId,
				StreamPoolId:     common.String("ocid1.streampool.oc1..mock"),
				LifecycleState:   streamingsdk.StreamLifecycleStateCreating,
				TimeCreated:      &createdAt,
				MessagesEndpoint: common.String("https://messages.mock.invalid"),
				FreeformTags:     details.FreeformTags,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state streamingsdk.Stream) (streamingsdk.Stream, ocimock.Response, error) {
			switch state.LifecycleState {
			case streamingsdk.StreamLifecycleStateCreating:
				if createReadObserved {
					state.LifecycleState = streamingsdk.StreamLifecycleStateActive
				} else {
					createReadObserved = true
				}
			case streamingsdk.StreamLifecycleStateUpdating:
				if updateReadObserved {
					state.LifecycleState = streamingsdk.StreamLifecycleStateActive
				} else {
					updateReadObserved = true
				}
			case streamingsdk.StreamLifecycleStateDeleting:
				if deleteReadObserved {
					state.LifecycleState = streamingsdk.StreamLifecycleStateDeleted
				} else {
					deleteReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state streamingsdk.Stream) (streamingsdk.Stream, ocimock.Response, error) {
			var details streamingsdk.UpdateStreamDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return streamingsdk.Stream{}, ocimock.Response{}, err
			}
			if details.FreeformTags["osok-mock"] != "update" || details.StreamPoolId != nil {
				return streamingsdk.Stream{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateStream details: %+v", details)
			}
			state.FreeformTags = details.FreeformTags
			state.MessagesEndpoint = common.String(mockStreamUpdatedEndpoint)
			state.LifecycleState = streamingsdk.StreamLifecycleStateUpdating
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state streamingsdk.Stream) (streamingsdk.Stream, ocimock.Response, error) {
			state.LifecycleState = streamingsdk.StreamLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}

func newMockStreamClient(
	sdkClient streamingsdk.StreamAdminClient,
	credentials *fakeCredentialClient,
) (StreamServiceClient, error) {
	manager := &StreamServiceManager{
		CredentialClient: credentials,
		Log:              loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
	}
	hooks := newStreamRuntimeHooks(manager, sdkClient)
	delegate := defaultStreamServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*streamingv1beta1.Stream](
			buildStreamGeneratedRuntimeConfig(manager, hooks),
		),
	}
	client := wrapStreamGeneratedClient(hooks, delegate)
	wrapped, ok := client.(streamEndpointSecretClient)
	if !ok {
		return nil, fmt.Errorf("production Stream client = %T, want streamEndpointSecretClient", client)
	}
	wrapped.loadStream = func(ctx context.Context, streamID shared.OCID) (*streamingsdk.Stream, error) {
		response, err := sdkClient.GetStream(ctx, streamingsdk.GetStreamRequest{StreamId: common.String(string(streamID))})
		if err != nil {
			return nil, err
		}
		return &response.Stream, nil
	}
	return wrapped, nil
}

func validateMockStreamEndpointSecret(
	exists bool,
	record credhelper.SecretRecord,
	resource *streamingv1beta1.Stream,
	wantEndpoint string,
) error {
	if !exists {
		return fmt.Errorf("active Stream did not create its endpoint Secret")
	}
	if got := record.Labels[streamEndpointSecretOwnerUIDLabel]; got != string(resource.UID) {
		return fmt.Errorf("endpoint Secret owner UID = %q, want %q", got, resource.UID)
	}
	if got := string(record.Data["endpoint"]); got != wantEndpoint {
		return fmt.Errorf("endpoint Secret endpoint = %q, want %q", got, wantEndpoint)
	}
	return nil
}

func cloneMockStreamSecretRecord(source credhelper.SecretRecord) credhelper.SecretRecord {
	return credhelper.SecretRecord{
		UID:    source.UID,
		Labels: cloneMockStreamStringMap(source.Labels),
		Data:   cloneMockStreamByteMap(source.Data),
	}
}

func cloneMockStreamStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneMockStreamByteMap(source map[string][]byte) map[string][]byte {
	if source == nil {
		return nil
	}
	cloned := make(map[string][]byte, len(source))
	for key, value := range source {
		cloned[key] = append([]byte(nil), value...)
	}
	return cloned
}
