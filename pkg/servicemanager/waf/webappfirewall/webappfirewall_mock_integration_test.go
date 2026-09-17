/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package webappfirewall

import (
	"context"
	"fmt"
	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationWebAppFirewallEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &wafv1beta1.WebAppFirewall{}
	ocimock.InitializeResource(resource, "mock-webappfirewall")
	resource.Spec = ocimock.MustJSONFixture[wafv1beta1.WebAppFirewallSpec](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waf-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "loadBalancerId": "\u003cocid:2\u003e",
  "webAppFirewallPolicyId": "\u003cocid:3\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-waf-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[wafsdk.CreateWebAppFirewallLoadBalancerDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-waf-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "loadBalancerId": "\u003cocid:2\u003e",
  "webAppFirewallPolicyId": "\u003cocid:3\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:11:24.587Z"
    }
  },
  "displayName": "osok-mock-waf-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:5>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:11:24.617Z",
  "timeUpdated": "2026-09-01T23:11:41.200Z",
  "webAppFirewallPolicyId": "<ocid:3>"
}`)
	createdReadStates := []wafsdk.WebAppFirewallLoadBalancer{
		ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:11:24.587Z"
    }
  },
  "displayName": "osok-mock-waf-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:5>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:11:24.617Z",
  "timeUpdated": "2026-09-01T23:11:41.200Z",
  "webAppFirewallPolicyId": "<ocid:3>"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[wafsdk.UpdateWebAppFirewallDetails](t, `{
  "displayName": "osok-mock-waf-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:11:24.587Z"
    }
  },
  "displayName": "osok-mock-waf-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:11:24.617Z",
  "timeUpdated": "2026-09-01T23:11:42.629Z",
  "webAppFirewallPolicyId": "<ocid:3>"
}`)
	updatedReadStates := []wafsdk.WebAppFirewallLoadBalancer{
		ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:11:24.587Z"
    }
  },
  "displayName": "osok-mock-waf-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "loadBalancerId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:11:24.617Z",
  "timeUpdated": "2026-09-01T23:11:42.629Z",
  "webAppFirewallPolicyId": "<ocid:3>"
}`),
	}
	deletedReadStates := []wafsdk.WebAppFirewallLoadBalancer{
		ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:11:24.587Z"
    }
  },
  "displayName": "osok-mock-waf-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETING",
  "loadBalancerId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:11:24.617Z",
  "timeUpdated": "2026-09-01T23:11:45.128Z",
  "webAppFirewallPolicyId": "<ocid:3>"
}`),
		ocimock.MustOCIResponseFixture[wafsdk.WebAppFirewallLoadBalancer](t, `{
  "backendType": "LOAD_BALANCER",
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T23:11:24.587Z"
    }
  },
  "displayName": "osok-mock-waf-v1-updated",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:5>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "loadBalancerId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": "2026-09-01T23:11:24.617Z",
  "timeUpdated": "2026-09-01T23:11:59.389Z",
  "webAppFirewallPolicyId": "<ocid:3>"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		wafsdk.WebAppFirewallLoadBalancer,
		wafsdk.CreateWebAppFirewallLoadBalancerDetails,
		wafsdk.UpdateWebAppFirewallDetails,
	]{
		CollectionPath:    "/20210930/webAppFirewalls",
		ItemPath:          "/20210930/webAppFirewalls/<ocid:5>",
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
		ValidateDelete: func(request ocimock.Request, _ wafsdk.WebAppFirewallLoadBalancer) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210930", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close WebAppFirewall OCI mock: %v", err)
		}
	})
	sdkClient := wafsdk.WafClient{BaseClient: session.BaseClient()}
	client := newWebAppFirewallServiceClientWithOCIClient(loggerutil.OSOKLogger{}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*wafv1beta1.WebAppFirewall]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *wafv1beta1.WebAppFirewall) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.BackendType, current.Spec.BackendType) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.LoadBalancerId, current.Spec.LoadBalancerId) ||
				!reflect.DeepEqual(current.Status.WebAppFirewallPolicyId, current.Spec.WebAppFirewallPolicyId) {
				return fmt.Errorf("created WebAppFirewall status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *wafv1beta1.WebAppFirewall) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *wafv1beta1.WebAppFirewall) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated WebAppFirewall status = %+v", current.Status)
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
