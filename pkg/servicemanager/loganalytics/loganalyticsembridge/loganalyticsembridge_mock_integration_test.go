/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package loganalyticsembridge

import (
	"context"
	"fmt"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

func TestMockIntegrationLogAnalyticsEmBridgeCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := newLogAnalyticsEmBridgeResource()
	ocimock.InitializeResource(resource, "mock-log-analytics-em-bridge")
	resource.Annotations = map[string]string{logAnalyticsEmBridgeNamespaceAnnotation: "mocknamespace"}
	updatedSpec := resource.Spec
	updatedSpec.BucketName = "em-bridge-bucket-v2"
	updatedSpec.Description = "updated bridge"
	createRequest := ocimock.MustJSONFixture[loganalyticssdk.CreateLogAnalyticsEmBridgeDetails](t, `{
  "displayName":"em-bridge-sample","compartmentId":"ocid1.compartment.oc1..bridge",
  "emEntitiesCompartmentId":"ocid1.compartment.oc1..entities","bucketName":"em-bridge-bucket",
  "description":"desired bridge","freeformTags":{"env":"dev"},"definedTags":{"Operations":{"CostCenter":"42"}}
}`)
	createdState := newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, resource.Spec.BucketName, loganalyticssdk.EmBridgeLifecycleStatesActive)
	updateRequest := ocimock.MustJSONFixture[loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails](t, `{
  "description":"updated bridge","bucketName":"em-bridge-bucket-v2"
}`)
	updatedState := newSDKLogAnalyticsEmBridge(testLogAnalyticsEmBridgeID, updatedSpec.BucketName, loganalyticssdk.EmBridgeLifecycleStatesActive)
	updatedState.Description = common.String(updatedSpec.Description)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[loganalyticssdk.LogAnalyticsEmBridge, loganalyticssdk.CreateLogAnalyticsEmBridgeDetails, loganalyticssdk.UpdateLogAnalyticsEmBridgeDetails]{
		CollectionPath: "/20200601/namespaces/mocknamespace/logAnalyticsEmBridges", ItemPath: "/20200601/namespaces/mocknamespace/logAnalyticsEmBridges/" + testLogAnalyticsEmBridgeID,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://loganalytics.mock.invalid", BasePath: "20200601", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	provider := common.NewRawConfigurationProvider("ocid1.tenancy.oc1..mock", "ocid1.user.oc1..mock", "us-ashburn-1", "00:00:00", "unused", nil)
	client := newLogAnalyticsEmBridgeServiceClientWithOCIClient(loggerutil.OSOKLogger{}, provider, loganalyticssdk.LogAnalyticsClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loganalyticsv1beta1.LogAnalyticsEmBridge]{
		Resource: resource, Client: client, CreateContext: func(ctx context.Context) context.Context { return ctx },
		ValidateCreated: func(current *loganalyticsv1beta1.LogAnalyticsEmBridge) error {
			if current.Status.Id != testLogAnalyticsEmBridgeID || current.Status.BucketName != current.Spec.BucketName || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created LogAnalyticsEmBridge status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loganalyticsv1beta1.LogAnalyticsEmBridge) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loganalyticsv1beta1.LogAnalyticsEmBridge) error {
			if current.Status.BucketName != current.Spec.BucketName || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated LogAnalyticsEmBridge status = %+v", current.Status)
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
