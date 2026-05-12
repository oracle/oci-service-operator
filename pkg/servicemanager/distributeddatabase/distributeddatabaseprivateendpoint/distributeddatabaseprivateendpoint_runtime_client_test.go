/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package distributeddatabaseprivateendpoint

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	distributeddatabasesdk "github.com/oracle/oci-go-sdk/v65/distributeddatabase"
	distributeddatabasev1beta1 "github.com/oracle/oci-service-operator/api/distributeddatabase/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type distributedDatabasePrivateEndpointRequestBodyBuilder interface {
	HTTPRequest(
		method string,
		path string,
		binaryRequestBody *common.OCIReadSeekCloser,
		extraHeaders map[string]string,
	) (http.Request, error)
}

func TestApplyDistributedDatabasePrivateEndpointRuntimeHooksOverridesGeneratedDefaults(t *testing.T) {
	t.Parallel()

	hooks := newDistributedDatabasePrivateEndpointDefaultRuntimeHooks(distributeddatabasesdk.DistributedDbPrivateEndpointServiceClient{})
	applyDistributedDatabasePrivateEndpointRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if got, want := hooks.Semantics.List.MatchFields, []string{"compartmentId", "displayName"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want %#v", got, want)
	}
	if got, want := hooks.List.Fields, reviewedDistributedDatabasePrivateEndpointListFields(); !reflect.DeepEqual(got, want) {
		t.Fatalf("hooks.List.Fields = %#v, want %#v", got, want)
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want explicit update builder")
	}
	if hooks.TrackedRecreate.ClearTrackedIdentity == nil {
		t.Fatal("hooks.TrackedRecreate.ClearTrackedIdentity = nil, want tracked identity cleanup")
	}
}

func TestBuildDistributedDatabasePrivateEndpointUpdateBodySupportsClearingOptionalFields(t *testing.T) {
	t.Parallel()

	currentSpec := newTestDistributedDatabasePrivateEndpointResource().Spec
	resource := newTestDistributedDatabasePrivateEndpointResource()
	resource.Spec.Description = ""
	resource.Spec.NsgIds = []string{}
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := observedDistributedDatabasePrivateEndpointFromSpec(
		"ocid1.distributeddatabaseprivateendpoint.oc1..existing",
		currentSpec,
		distributeddatabasesdk.DistributedDatabasePrivateEndpointLifecycleStateActive,
	)

	details, updateNeeded, err := buildDistributedDatabasePrivateEndpointUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildDistributedDatabasePrivateEndpointUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildDistributedDatabasePrivateEndpointUpdateBody() updateNeeded = false, want explicit clear intent")
	}
	if details.Description == nil || *details.Description != "" {
		t.Fatalf("Description = %#v, want explicit empty string clear", details.Description)
	}
	if details.NsgIds == nil || len(details.NsgIds) != 0 {
		t.Fatalf("NsgIds = %#v, want explicit empty slice clear", details.NsgIds)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("DefinedTags = %#v, want explicit empty map clear", details.DefinedTags)
	}

	body := distributedDatabasePrivateEndpointSerializedRequestBody(
		t,
		distributeddatabasesdk.UpdateDistributedDatabasePrivateEndpointRequest{
			DistributedDatabasePrivateEndpointId:            common.String("ocid1.distributeddatabaseprivateendpoint.oc1..existing"),
			UpdateDistributedDatabasePrivateEndpointDetails: details,
		},
		http.MethodPut,
		"/distributedDatabasePrivateEndpoints/ocid1.distributeddatabaseprivateendpoint.oc1..existing",
	)

	for _, want := range []string{
		`"description":""`,
		`"nsgIds":[]`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestClearTrackedDistributedDatabasePrivateEndpointIdentity(t *testing.T) {
	t.Parallel()

	resource := newTestDistributedDatabasePrivateEndpointResource()
	resource.Status.Id = "ocid1.distributeddatabaseprivateendpoint.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.distributeddatabaseprivateendpoint.oc1..existing")

	clearTrackedDistributedDatabasePrivateEndpointIdentity(resource)

	if resource.Status.Id != "" {
		t.Fatalf("resource.Status.Id = %q, want empty string", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Ocid != "" {
		t.Fatalf("resource.Status.OsokStatus.Ocid = %q, want empty string", resource.Status.OsokStatus.Ocid)
	}
}

func newTestDistributedDatabasePrivateEndpointResource() *distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint {
	return &distributeddatabasev1beta1.DistributedDatabasePrivateEndpoint{
		Spec: distributeddatabasev1beta1.DistributedDatabasePrivateEndpointSpec{
			CompartmentId: "ocid1.compartment.oc1..exampleuniqueID",
			SubnetId:      "ocid1.subnet.oc1..exampleuniqueID",
			DisplayName:   "ddb-private-endpoint",
			Description:   "private endpoint for distributed database",
			NsgIds: []string{
				"ocid1.networksecuritygroup.oc1..nsg1",
				"ocid1.networksecuritygroup.oc1..nsg2",
			},
			FreeformTags: map[string]string{
				"managed-by": "oci-service-operator",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func observedDistributedDatabasePrivateEndpointFromSpec(
	id string,
	spec distributeddatabasev1beta1.DistributedDatabasePrivateEndpointSpec,
	state distributeddatabasesdk.DistributedDatabasePrivateEndpointLifecycleStateEnum,
) distributeddatabasesdk.DistributedDatabasePrivateEndpoint {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	return distributeddatabasesdk.DistributedDatabasePrivateEndpoint{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		SubnetId:       common.String(spec.SubnetId),
		VcnId:          common.String("ocid1.vcn.oc1..exampleuniqueID"),
		DisplayName:    common.String(spec.DisplayName),
		TimeCreated:    now,
		TimeUpdated:    now,
		LifecycleState: state,
		Description:    common.String(spec.Description),
		PrivateIp:      common.String("10.0.0.10"),
		NsgIds:         append([]string(nil), spec.NsgIds...),
		LifecycleDetails: common.String(
			"ready",
		),
		ProxyComputeInstanceId: common.String("ocid1.instance.oc1..proxy"),
		FreeformTags: map[string]string{
			"managed-by": "oci-service-operator",
		},
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {"CostCenter": "42"},
		},
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {"free-tier-retained": "true"},
		},
	}
}

func distributedDatabasePrivateEndpointSerializedRequestBody(
	t *testing.T,
	request distributedDatabasePrivateEndpointRequestBodyBuilder,
	method string,
	path string,
) string {
	t.Helper()

	httpRequest, err := request.HTTPRequest(method, path, nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest() error = %v", err)
	}

	body, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("ReadAll(request body) error = %v", err)
	}
	return string(body)
}
