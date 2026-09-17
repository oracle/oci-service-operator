/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package announcementsubscription

import (
	"context"
	"fmt"
	announcementsservicesdk "github.com/oracle/oci-go-sdk/v65/announcementsservice"
	announcementsservicev1beta1 "github.com/oracle/oci-service-operator/api/announcementsservice/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationAnnouncementSubscriptionEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &announcementsservicev1beta1.AnnouncementSubscription{}
	ocimock.InitializeResource(resource, "mock-announcementsubscription")
	resource.Spec = ocimock.MustJSONFixture[announcementsservicev1beta1.AnnouncementSubscriptionSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK recorded announcement subscription",
  "displayName": "osok-mock-announcement-subscription",
  "freeformTags": {
    "osok-mock": "create"
  },
  "onsTopicId": "\u003cocid:2\u003e",
  "preferredTimeZone": "UTC"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded announcement subscription updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "preferredTimeZone": "America/Chicago"
}`)
	createRequest := ocimock.MustJSONFixture[announcementsservicesdk.CreateAnnouncementSubscriptionDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK recorded announcement subscription",
  "displayName": "osok-mock-announcement-subscription",
  "freeformTags": {
    "osok-mock": "create"
  },
  "onsTopicId": "\u003cocid:2\u003e",
  "preferredTimeZone": "UTC"
}`)
	createdState := ocimock.MustOCIResponseFixture[announcementsservicesdk.AnnouncementSubscription](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T22:26:17.284Z"
    }
  },
  "description": "OSOK recorded announcement subscription",
  "displayName": "osok-mock-announcement-subscription",
  "filterGroups": {},
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "onsTopicId": "<ocid:2>",
  "preferredLanguage": null,
  "preferredTimeZone": "UTC",
  "systemTags": {},
  "timeCreated": "2026-09-02T22:26:17.409Z",
  "timeUpdated": "2026-09-02T22:26:17.409Z"
}`)
	updateRequest := ocimock.MustJSONFixture[announcementsservicesdk.UpdateAnnouncementSubscriptionDetails](t, `{
  "description": "OSOK recorded announcement subscription updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "preferredTimeZone": "America/Chicago"
}`)
	updatedState := ocimock.MustOCIResponseFixture[announcementsservicesdk.AnnouncementSubscription](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T22:26:17.284Z"
    }
  },
  "description": "OSOK recorded announcement subscription updated",
  "displayName": "osok-mock-announcement-subscription",
  "filterGroups": {},
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "onsTopicId": "<ocid:2>",
  "preferredLanguage": null,
  "preferredTimeZone": "America/Chicago",
  "systemTags": {},
  "timeCreated": "2026-09-02T22:26:17.409Z",
  "timeUpdated": "2026-09-02T22:26:18.116Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[announcementsservicesdk.AnnouncementSubscription](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T22:26:17.284Z"
    }
  },
  "description": "OSOK recorded announcement subscription updated",
  "displayName": "osok-mock-announcement-subscription",
  "filterGroups": {},
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "onsTopicId": "<ocid:2>",
  "preferredLanguage": null,
  "preferredTimeZone": "America/Chicago",
  "systemTags": {},
  "timeCreated": "2026-09-02T22:26:17.409Z",
  "timeUpdated": "2026-09-02T22:26:18.916Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		announcementsservicesdk.AnnouncementSubscription,
		announcementsservicesdk.CreateAnnouncementSubscriptionDetails,
		announcementsservicesdk.UpdateAnnouncementSubscriptionDetails,
	]{
		CollectionPath:    "/20180904/announcementSubscriptions",
		ItemPath:          "/20180904/announcementSubscriptions/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ announcementsservicesdk.CreateAnnouncementSubscriptionDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ announcementsservicesdk.AnnouncementSubscription) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20180904", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close AnnouncementSubscription OCI mock: %v", err)
		}
	})
	sdkClient := announcementsservicesdk.AnnouncementSubscriptionClient{BaseClient: session.BaseClient()}
	manager := &AnnouncementSubscriptionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newAnnouncementSubscriptionRuntimeHooks(manager, sdkClient)
	client := wrapAnnouncementSubscriptionGeneratedClient(hooks, defaultAnnouncementSubscriptionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*announcementsservicev1beta1.AnnouncementSubscription](buildAnnouncementSubscriptionGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*announcementsservicev1beta1.AnnouncementSubscription]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *announcementsservicev1beta1.AnnouncementSubscription) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.OnsTopicId, current.Spec.OnsTopicId) ||
				!reflect.DeepEqual(current.Status.PreferredTimeZone, current.Spec.PreferredTimeZone) {
				return fmt.Errorf("created AnnouncementSubscription status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *announcementsservicev1beta1.AnnouncementSubscription) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *announcementsservicev1beta1.AnnouncementSubscription) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.PreferredTimeZone, current.Spec.PreferredTimeZone) {
				return fmt.Errorf("updated AnnouncementSubscription status = %+v", current.Status)
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
