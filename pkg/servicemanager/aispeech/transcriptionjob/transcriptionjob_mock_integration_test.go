/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package transcriptionjob

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	aispeechsdk "github.com/oracle/oci-go-sdk/v65/aispeech"
	"github.com/oracle/oci-go-sdk/v65/common"
	aispeechv1beta1 "github.com/oracle/oci-service-operator/api/aispeech/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockTranscriptionJobID = "ocid1.aispeechtranscriptionjob.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal job contract, resource-local delete semantics, and vendored OCI SDK.
func TestMockIntegrationTranscriptionJobLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &aispeechv1beta1.TranscriptionJob{ObjectMeta: metav1.ObjectMeta{Name: "mock-transcription-job", Namespace: "default", UID: types.UID("mock-transcription-job-uid")}, Spec: aispeechv1beta1.TranscriptionJobSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-transcription-job", Description: "mock create",
		InputLocation:  aispeechv1beta1.TranscriptionJobInputLocation{LocationType: "OBJECT_LIST_INLINE_INPUT_LOCATION", ObjectLocations: []aispeechv1beta1.TranscriptionJobInputLocationObjectLocation{{NamespaceName: "mocknamespace", BucketName: "mock-bucket", ObjectNames: []string{"audio.wav"}}}},
		OutputLocation: aispeechv1beta1.TranscriptionJobOutputLocation{NamespaceName: "mocknamespace", BucketName: "mock-bucket", Prefix: "transcripts/"},
		ModelDetails:   aispeechv1beta1.TranscriptionJobModelDetails{Domain: "GENERIC", LanguageCode: "en-US"}, FreeformTags: map[string]string{"osok-mock": "create"},
	}}
	responder, err := newTranscriptionJobMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://speech.aiservice.mock.invalid", BasePath: "20220101", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newTranscriptionJobServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, aispeechsdk.AIServiceSpeechClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*aispeechv1beta1.TranscriptionJob]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *aispeechv1beta1.TranscriptionJob) error {
			if current.Status.Id != mockTranscriptionJobID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(aispeechsdk.TranscriptionJobLifecycleStateSucceeded) {
				return fmt.Errorf("created TranscriptionJob status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *aispeechv1beta1.TranscriptionJob) {
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *aispeechv1beta1.TranscriptionJob) error {
			if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated TranscriptionJob status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newTranscriptionJobMockResponder(resource *aispeechv1beta1.TranscriptionJob) (*ocimock.CRUDResponder[aispeechsdk.TranscriptionJob], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	createReads := 0
	deleteReads := 0
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[aispeechsdk.TranscriptionJob]{
		CollectionPath: "/20220101/transcriptionJobs", ItemPath: "/20220101/transcriptionJobs/" + mockTranscriptionJobID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (aispeechsdk.TranscriptionJob, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero aispeechsdk.TranscriptionJob
				return zero, ocimock.Response{}, err
			}
			var details aispeechsdk.CreateTranscriptionJobDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return aispeechsdk.TranscriptionJob{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return aispeechsdk.TranscriptionJob{}, ocimock.Response{}, err
			}
			input, ok := details.InputLocation.(aispeechsdk.ObjectListInlineInputLocation)
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || !ok || len(input.ObjectLocations) != 1 || details.OutputLocation == nil || details.OutputLocation.Prefix == nil || *details.OutputLocation.Prefix != "transcripts/" {
				return aispeechsdk.TranscriptionJob{}, ocimock.Response{}, fmt.Errorf("unexpected CreateTranscriptionJob details: %+v", details)
			}
			state := aispeechsdk.TranscriptionJob{Id: common.String(mockTranscriptionJobID), CompartmentId: details.CompartmentId, ModelDetails: details.ModelDetails,
				InputLocation: details.InputLocation, OutputLocation: details.OutputLocation, DisplayName: details.DisplayName, Description: details.Description,
				TimeAccepted: &now, TotalTasks: common.Int(1), OutstandingTasks: common.Int(1), SuccessfulTasks: common.Int(0), PercentComplete: common.Int(0),
				CreatedBy: common.String("ocid1.user.oc1..mock"), LifecycleState: aispeechsdk.TranscriptionJobLifecycleStateAccepted, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state aispeechsdk.TranscriptionJob) (aispeechsdk.TranscriptionJob, ocimock.Response, error) {
			if state.LifecycleState == aispeechsdk.TranscriptionJobLifecycleStateAccepted {
				createReads++
				if createReads == 2 {
					state.LifecycleState, state.TimeStarted, state.PercentComplete = aispeechsdk.TranscriptionJobLifecycleStateInProgress, &now, common.Int(50)
				}
			} else if state.LifecycleState == aispeechsdk.TranscriptionJobLifecycleStateInProgress {
				state.LifecycleState, state.TimeFinished, state.OutstandingTasks, state.SuccessfulTasks, state.PercentComplete = aispeechsdk.TranscriptionJobLifecycleStateSucceeded, &now, common.Int(0), common.Int(1), common.Int(100)
			} else if state.LifecycleState == aispeechsdk.TranscriptionJobLifecycleStateCanceling {
				deleteReads++
				if deleteReads > 1 {
					state.LifecycleState = aispeechsdk.TranscriptionJobLifecycleStateCanceled
				}
			} else if state.LifecycleState == aispeechsdk.TranscriptionJobLifecycleStateCanceled {
				response, err := ocimock.JSONResponse(http.StatusNotFound, map[string]string{"code": "NotAuthorizedOrNotFound"})
				return state, response, err
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state aispeechsdk.TranscriptionJob) (aispeechsdk.TranscriptionJob, ocimock.Response, error) {
			var details aispeechsdk.UpdateTranscriptionJobDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return aispeechsdk.TranscriptionJob{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.FreeformTags["osok-mock"] != "update" || details.DisplayName != nil {
				return aispeechsdk.TranscriptionJob{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateTranscriptionJob details: %+v", details)
			}
			state.Description, state.FreeformTags = details.Description, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state aispeechsdk.TranscriptionJob) (aispeechsdk.TranscriptionJob, ocimock.Response, error) {
			state.LifecycleState = aispeechsdk.TranscriptionJobLifecycleStateCanceling
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
