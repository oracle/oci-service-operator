/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package webappacceleration

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
func TestMockIntegrationWebAppAccelerationEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &waav1beta1.WebAppAcceleration{}
	ocimock.InitializeResource(resource, "mock-webappacceleration")
	resource.Spec = ocimock.MustJSONFixture[waav1beta1.WebAppAccelerationSpec](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waa-acceleration-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "loadBalancerId": "\u003cocid:3\u003e",
  "webAppAccelerationPolicyId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-waa-acceleration-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[waasdk.CreateWebAppAccelerationLoadBalancerDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waa-acceleration-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "loadBalancerId": "\u003cocid:3\u003e",
  "webAppAccelerationPolicyId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T03:06:40.769Z"
    }
  },
  "displayName": "osok-mock-waa-acceleration-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:3>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T03:06:40.824Z",
  "timeUpdated": "2026-09-04T03:07:03.352Z",
  "webAppAccelerationPolicyId": "<ocid:2>"
}`)
	createdReadStates := []waasdk.WebAppAccelerationLoadBalancer{
		ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T03:06:40.769Z"
    }
  },
  "displayName": "osok-mock-waa-acceleration-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:3>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T03:06:40.824Z",
  "timeUpdated": "2026-09-04T03:07:03.352Z",
  "webAppAccelerationPolicyId": "<ocid:2>"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[waasdk.UpdateWebAppAccelerationDetails](t, `{
  "displayName": "osok-mock-waa-acceleration-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T03:06:40.769Z"
    }
  },
  "displayName": "osok-mock-waa-acceleration-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:3>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T03:06:40.824Z",
  "timeUpdated": "2026-09-04T03:07:12.763Z",
  "webAppAccelerationPolicyId": "<ocid:2>"
}`)
	updatedReadStates := []waasdk.WebAppAccelerationLoadBalancer{
		ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T03:06:40.769Z"
    }
  },
  "displayName": "osok-mock-waa-acceleration-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:3>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T03:06:40.824Z",
  "timeUpdated": "2026-09-04T03:07:12.763Z",
  "webAppAccelerationPolicyId": "<ocid:2>"
}`),
	}
	deletedReadStates := []waasdk.WebAppAccelerationLoadBalancer{
		ocimock.MustOCIResponseFixture[waasdk.WebAppAccelerationLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T03:06:40.769Z"
    }
  },
  "displayName": "osok-mock-waa-acceleration-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "loadBalancerId": "<ocid:3>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-04T03:06:40.824Z",
  "timeUpdated": "2026-09-04T03:07:32.084Z",
  "webAppAccelerationPolicyId": "<ocid:2>"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		waasdk.WebAppAccelerationLoadBalancer,
		waasdk.CreateWebAppAccelerationLoadBalancerDetails,
		waasdk.UpdateWebAppAccelerationDetails,
	]{
		CollectionPath:    "/20211230/webAppAccelerations",
		ItemPath:          "/20211230/webAppAccelerations/<ocid:4>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
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
		ValidateCreateRaw: func(request ocimock.Request) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "backendType", "LOAD_BALANCER", createRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ waasdk.WebAppAccelerationLoadBalancer) error {
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
			t.Errorf("close WebAppAcceleration OCI mock: %v", err)
		}
	})
	baseClient := session.BaseClient()
	sdkClient := mockWebAppAccelerationOCIClient{
		WaaClient:         waasdk.WaaClient{BaseClient: baseClient},
		WorkRequestClient: waasdk.WorkRequestClient{BaseClient: baseClient},
	}
	client := newWebAppAccelerationServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*waav1beta1.WebAppAcceleration]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *waav1beta1.WebAppAcceleration) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.BackendType, current.Spec.BackendType) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.LoadBalancerId, current.Spec.LoadBalancerId) ||
				!reflect.DeepEqual(current.Status.WebAppAccelerationPolicyId, current.Spec.WebAppAccelerationPolicyId) {
				return fmt.Errorf("created WebAppAcceleration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *waav1beta1.WebAppAcceleration) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *waav1beta1.WebAppAcceleration) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated WebAppAcceleration status = %+v", current.Status)
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
