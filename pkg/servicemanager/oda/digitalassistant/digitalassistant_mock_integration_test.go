/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package digitalassistant

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: explicit NEW subtype selected from the CR discriminator,
// vendored SDK, production runtime, formal lifecycle, and sanitized evidence.
func TestMockIntegrationDigitalAssistantCRUD(t *testing.T) {
	t.Parallel()
	resource := &odav1beta1.DigitalAssistant{}
	ocimock.InitializeResource(resource, "mock-digitalassistant")
	resource.SetAnnotations(map[string]string{digitalAssistantOdaInstanceIDAnnotation: "<ocid:1>"})
	resource.Spec = ocimock.MustJSONFixture[odav1beta1.DigitalAssistantSpec](t, `{
		"category":"dev",
		"definedTags":{},
		"description":"test digital assistant",
		"displayName":"Test Assistant",
		"kind":"NEW",
		"multilingualMode":"NATIVE",
		"name":"TestAssistant",
		"nativeLanguageTags":["en"],
		"platformVersion":"22.04",
		"primaryLanguageTag":"en",
		"version":"1.0"
	}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"digital assistant updated"}`)

	createRequest := ocimock.MustJSONFixture[odasdk.CreateNewDigitalAssistantDetails](t, `{
		"category":"dev",
		"definedTags":{},
		"description":"test digital assistant",
		"displayName":"Test Assistant",
		"multilingualMode":"NATIVE",
		"name":"TestAssistant",
		"nativeLanguageTags":["en"],
		"platformVersion":"22.04",
		"primaryLanguageTag":"en",
		"version":"1.0"
	}`)
	updateRequest := ocimock.MustJSONFixture[odasdk.UpdateDigitalAssistantDetails](t, `{"description":"digital assistant updated"}`)
	createdState := ocimock.MustOCIResponseFixture[odasdk.DigitalAssistant](t, `{
		"category":"dev",
		"description":"test digital assistant",
		"displayName":"Test Assistant",
		"id":"<ocid:2>",
		"kind":"NEW",
		"lifecycleState":"ACTIVE",
		"multilingualMode":"NATIVE",
		"name":"TestAssistant",
		"nativeLanguageTags":["en"],
		"platformVersion":"22.04",
		"primaryLanguageTag":"en",
		"version":"1.0"
	}`)
	updatedState := ocimock.MustOCIResponseFixture[odasdk.DigitalAssistant](t, `{
		"category":"dev",
		"description":"digital assistant updated",
		"displayName":"Test Assistant",
		"id":"<ocid:2>",
		"kind":"NEW",
		"lifecycleState":"ACTIVE",
		"multilingualMode":"NATIVE",
		"name":"TestAssistant",
		"nativeLanguageTags":["en"],
		"platformVersion":"22.04",
		"primaryLanguageTag":"en",
		"version":"1.0"
	}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[odasdk.WorkRequest](t, `{
		"compartmentId":null,
		"id":"<ocid:3>",
		"odaInstanceId":"<ocid:1>",
		"percentComplete":100,
		"requestAction":"CREATE_DIGITAL_ASSISTANT",
		"resourceId":"<ocid:2>",
		"resources":[{
			"resourceAction":"CREATE",
			"resourceId":"<ocid:2>",
			"resourceType":"digitalassistant",
			"resourceUri":null,
			"status":"SUCCEEDED",
			"statusMessage":null
		}],
		"status":"SUCCEEDED",
		"statusMessage":null,
		"timeAccepted":null,
		"timeFinished":null,
		"timeStarted":null
	}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[odasdk.DigitalAssistant, odasdk.CreateNewDigitalAssistantDetails, odasdk.UpdateDigitalAssistantDetails]{
		CollectionPath: "/20190506/odaInstances/<ocid:1>/digitalAssistants",
		ItemPath:       "/20190506/odaInstances/<ocid:1>/digitalAssistants/<ocid:2>",
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:   &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: http.StatusAccepted, UpdateStatus: http.StatusOK, DeleteStatus: http.StatusNoContent, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "kind", "NEW", createRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20190506/workRequests/<ocid:3>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host: "https://digitalassistant-api.us-ashburn-1.oci.oraclecloud.com", BasePath: "20190506", Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	base := session.BaseClient()
	client := newDigitalAssistantServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		digitalAssistantManagementClient{
			management: odasdk.ManagementClient{BaseClient: base},
			work:       odasdk.OdaClient{BaseClient: base},
		},
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.DigitalAssistant]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.DigitalAssistant) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DigitalAssistant status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.DigitalAssistant) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.DigitalAssistant) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DigitalAssistant status = %+v", current.Status)
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
