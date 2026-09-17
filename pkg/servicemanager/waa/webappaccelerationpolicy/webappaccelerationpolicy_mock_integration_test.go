/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package webappaccelerationpolicy

import (
	"context"
	"fmt"
	waasdk "github.com/oracle/oci-go-sdk/v65/waa"
	waav1beta1 "github.com/oracle/oci-service-operator/api/waa/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationWebAppAccelerationPolicyEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &waav1beta1.WebAppAccelerationPolicy{}
	ocimock.InitializeResource(resource, "mock-webappaccelerationpolicy")
	resource.Spec = ocimock.MustJSONFixture[waav1beta1.WebAppAccelerationPolicySpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waa-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-waa-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  }
}`)
	createRequest := ocimock.MustJSONFixture[waasdk.CreateWebAppAccelerationPolicyDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waa-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:55:43.178Z"
    }
  },
  "displayName": "osok-mock-waa-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T02:55:43.325Z",
  "timeUpdated": "2026-09-04T02:56:24.523Z"
}`)
	createdReadStates := []waasdk.WebAppAccelerationPolicy{
		ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:55:43.178Z"
    }
  },
  "displayName": "osok-mock-waa-policy-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T02:55:43.325Z",
  "timeUpdated": "2026-09-04T02:56:24.523Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[waasdk.UpdateWebAppAccelerationPolicyDetails](t, `{
  "displayName": "osok-mock-waa-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:55:43.178Z"
    }
  },
  "displayName": "osok-mock-waa-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T02:55:43.325Z",
  "timeUpdated": "2026-09-04T02:56:25.460Z"
}`)
	updatedReadStates := []waasdk.WebAppAccelerationPolicy{
		ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:55:43.178Z"
    }
  },
  "displayName": "osok-mock-waa-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T02:55:43.325Z",
  "timeUpdated": "2026-09-04T02:56:25.460Z"
}`),
	}
	deletedReadStates := []waasdk.WebAppAccelerationPolicy{
		ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:55:43.178Z"
    }
  },
  "displayName": "osok-mock-waa-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETING",
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T02:55:43.325Z",
  "timeUpdated": "2026-09-04T02:56:26.334Z"
}`),
		ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationPolicy](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:55:43.178Z"
    }
  },
  "displayName": "osok-mock-waa-policy-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "responseCachingPolicy": {
    "isResponseHeaderBasedCachingEnabled": true
  },
  "responseCompressionPolicy": {
    "gzipCompression": {
      "isEnabled": true
    }
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T02:55:43.325Z",
  "timeUpdated": "2026-09-04T02:57:00.200Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		waasdk.WebAppAccelerationPolicy,
		waasdk.CreateWebAppAccelerationPolicyDetails,
		waasdk.UpdateWebAppAccelerationPolicyDetails,
	]{
		CollectionPath:    "/20211230/webAppAccelerationPolicies",
		ItemPath:          "/20211230/webAppAccelerationPolicies/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: createdReadStates,
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      202,
		DeleteStatus:      202,
		ValidateCreate: func(request ocimock.Request, _ waasdk.CreateWebAppAccelerationPolicyDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ waasdk.WebAppAccelerationPolicy) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20211230", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close WebAppAccelerationPolicy OCI mock: %v", err)
		}
	})
	baseClient := session.BaseClient()
	sdkClient := mockWebAppAccelerationPolicyOCIClient{
		WaaClient:         waasdk.WaaClient{BaseClient: baseClient},
		WorkRequestClient: waasdk.WorkRequestClient{BaseClient: baseClient},
	}
	client := newWebAppAccelerationPolicyServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*waav1beta1.WebAppAccelerationPolicy]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *waav1beta1.WebAppAccelerationPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ResponseCachingPolicy, current.Spec.ResponseCachingPolicy) ||
				!reflect.DeepEqual(current.Status.ResponseCompressionPolicy, current.Spec.ResponseCompressionPolicy) {
				return fmt.Errorf("created WebAppAccelerationPolicy status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *waav1beta1.WebAppAccelerationPolicy) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *waav1beta1.WebAppAccelerationPolicy) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated WebAppAccelerationPolicy status = %+v", current.Status)
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
