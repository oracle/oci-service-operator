/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package distributeddatabase

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	distributeddatabasesdk "github.com/oracle/oci-go-sdk/v65/distributeddatabase"
	distributeddatabasev1beta1 "github.com/oracle/oci-service-operator/api/distributeddatabase/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type distributedDatabaseRequestBodyBuilder interface {
	HTTPRequest(
		method string,
		path string,
		binaryRequestBody *common.OCIReadSeekCloser,
		extraHeaders map[string]string,
	) (http.Request, error)
}

func TestApplyDistributedDatabaseRuntimeHooksOverridesGeneratedDefaults(t *testing.T) {
	t.Parallel()

	hooks := newDistributedDatabaseDefaultRuntimeHooks(distributeddatabasesdk.DistributedDbServiceClient{})
	applyDistributedDatabaseRuntimeHooks(&DistributedDatabaseServiceManager{}, &hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if got, want := hooks.Semantics.Lifecycle.ProvisioningStates, []string{"CREATING"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hooks.Semantics.Lifecycle.ProvisioningStates = %#v, want %#v", got, want)
	}
	if got, want := hooks.Semantics.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hooks.Semantics.Lifecycle.ActiveStates = %#v, want %#v", got, want)
	}
	if got, want := hooks.Semantics.List.MatchFields, []string{"compartmentId", "displayName", "prefix", "dbDeploymentType", "lifecycleState"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hooks.Semantics.List.MatchFields = %#v, want %#v", got, want)
	}
	if got, want := hooks.Get.Fields, []generatedruntime.RequestField{
		{FieldName: "DistributedDatabaseId", RequestName: "distributedDatabaseId", Contribution: "path", PreferResourceID: true},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hooks.Get.Fields = %#v, want %#v", got, want)
	}
	if got, want := hooks.List.Fields, []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "DbDeploymentType", RequestName: "dbDeploymentType", Contribution: "query"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hooks.List.Fields = %#v, want %#v", got, want)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want explicit create body builder")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want explicit update body builder")
	}
	if hooks.TrackedRecreate.ClearTrackedIdentity == nil {
		t.Fatal("hooks.TrackedRecreate.ClearTrackedIdentity = nil, want tracked identity cleanup")
	}
}

func TestBuildDistributedDatabaseCreateBodyNormalizesExistingClusterAndPreservesFalseBooleans(t *testing.T) {
	t.Parallel()

	resource := newTestDistributedDatabaseResource()
	resource.Spec.ShardDetails[0].Source = "EXISTING_CLUSTER"

	details, err := buildDistributedDatabaseCreateBody(context.Background(), nil, resource, "default")
	if err != nil {
		t.Fatalf("buildDistributedDatabaseCreateBody() error = %v", err)
	}

	body := distributedDatabaseSerializedRequestBody(
		t,
		distributeddatabasesdk.CreateDistributedDatabaseRequest{
			CreateDistributedDatabaseDetails: details,
		},
		http.MethodPost,
		"/distributedDatabases",
	)

	for _, want := range []string{
		`"source":"EXADB_XS"`,
		`"source":"NEW_VAULT_AND_CLUSTER"`,
		`"isDiagnosticsEventsEnabled":false`,
		`"isHealthMonitoringEnabled":false`,
		`"isIncidentLogsEnabled":false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
	if strings.Contains(body, `"jsonData"`) {
		t.Fatalf("request body %s unexpectedly contains jsonData", body)
	}
}

func TestBuildDistributedDatabaseUpdateBodySupportsClearingOptionalFields(t *testing.T) {
	t.Parallel()

	currentSpec := newTestDistributedDatabaseResource().Spec
	resource := newTestDistributedDatabaseResource()
	resource.Spec.DisplayName = "ddb-runtime-renamed"
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}

	current := observedDistributedDatabaseFromSpec(
		"ocid1.distributeddatabase.oc1..existing",
		currentSpec,
		distributeddatabasesdk.DistributedDatabaseLifecycleStateActive,
	)

	details, updateNeeded, err := buildDistributedDatabaseUpdateBody(resource, current)
	if err != nil {
		t.Fatalf("buildDistributedDatabaseUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildDistributedDatabaseUpdateBody() updateNeeded = false, want explicit mutable-field update")
	}
	if details.DisplayName == nil || *details.DisplayName != "ddb-runtime-renamed" {
		t.Fatalf("DisplayName = %#v, want renamed display name", details.DisplayName)
	}
	if details.FreeformTags == nil || len(details.FreeformTags) != 0 {
		t.Fatalf("FreeformTags = %#v, want explicit empty map clear", details.FreeformTags)
	}
	if details.DefinedTags == nil || len(details.DefinedTags) != 0 {
		t.Fatalf("DefinedTags = %#v, want explicit empty map clear", details.DefinedTags)
	}

	body := distributedDatabaseSerializedRequestBody(
		t,
		distributeddatabasesdk.UpdateDistributedDatabaseRequest{
			DistributedDatabaseId:            common.String("ocid1.distributeddatabase.oc1..existing"),
			UpdateDistributedDatabaseDetails: details,
		},
		http.MethodPut,
		"/distributedDatabases/ocid1.distributeddatabase.oc1..existing",
	)

	for _, want := range []string{
		`"displayName":"ddb-runtime-renamed"`,
		`"freeformTags":{}`,
		`"definedTags":{}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body %s does not contain %s", body, want)
		}
	}
}

func TestClearTrackedDistributedDatabaseIdentity(t *testing.T) {
	t.Parallel()

	resource := newTestDistributedDatabaseResource()
	resource.Status.Id = "ocid1.distributeddatabase.oc1..existing"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.distributeddatabase.oc1..existing")

	clearTrackedDistributedDatabaseIdentity(resource)

	if resource.Status.Id != "" {
		t.Fatalf("resource.Status.Id = %q, want empty string", resource.Status.Id)
	}
	if resource.Status.OsokStatus.Ocid != "" {
		t.Fatalf("resource.Status.OsokStatus.Ocid = %q, want empty string", resource.Status.OsokStatus.Ocid)
	}
}

func newTestDistributedDatabaseResource() *distributeddatabasev1beta1.DistributedDatabase {
	return &distributeddatabasev1beta1.DistributedDatabase{
		Spec: distributeddatabasev1beta1.DistributedDatabaseSpec{
			CompartmentId:   "ocid1.compartment.oc1..exampleuniqueID",
			DisplayName:     "ddb-runtime",
			DatabaseVersion: "23ai",
			Prefix:          "ddb123",
			PrivateEndpointIds: []string{
				"ocid1.distributeddatabaseprivateendpoint.oc1..privateendpoint",
			},
			ShardingMethod:   "USER",
			CharacterSet:     "AL32UTF8",
			NcharacterSet:    "AL16UTF16",
			ListenerPort:     1521,
			OnsPortLocal:     6234,
			OnsPortRemote:    6235,
			DbDeploymentType: "EXADB_XS",
			ShardDetails: []distributeddatabasev1beta1.DistributedDatabaseShardDetail{
				{
					Source:        "EXADB_XS",
					AdminPassword: "ShardPassword#123",
					VmClusterId:   "ocid1.vmcluster.oc1..shard",
					ShardSpace:    "PRIMARY",
					PeerVmClusterIds: []string{
						"ocid1.vmcluster.oc1..peer",
					},
				},
			},
			CatalogDetails: []distributeddatabasev1beta1.DistributedDatabaseCatalogDetail{
				{
					Source:             "NEW_VAULT_AND_CLUSTER",
					AdminPassword:      "CatalogPassword#123",
					AvailabilityDomain: "Uocm:PHX-AD-1",
					DbStorageVaultDetails: distributeddatabasev1beta1.DistributedDatabaseCatalogDetailDbStorageVaultDetails{
						HighCapacityDatabaseStorage: 128,
					},
					VmClusterDetails: distributeddatabasev1beta1.DistributedDatabaseCatalogDetailVmClusterDetails{
						SubnetId:                   "ocid1.subnet.oc1..catalog",
						BackupSubnetId:             "ocid1.subnet.oc1..catalogbackup",
						EnabledECpuCount:           16,
						SshPublicKeys:              []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcatalog"},
						IsDiagnosticsEventsEnabled: false,
						IsHealthMonitoringEnabled:  false,
						IsIncidentLogsEnabled:      false,
						BackupNetworkNsgIds:        []string{"ocid1.networksecuritygroup.oc1..backup"},
						NsgIds:                     []string{"ocid1.networksecuritygroup.oc1..catalog"},
						PrivateZoneId:              "ocid1.dnszone.oc1..catalog",
						TotalECpuCount:             32,
						VmFileSystemStorageSize:    2048,
					},
					ShardSpace: "CATALOG",
				},
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

func observedDistributedDatabaseFromSpec(
	id string,
	spec distributeddatabasev1beta1.DistributedDatabaseSpec,
	state distributeddatabasesdk.DistributedDatabaseLifecycleStateEnum,
) distributeddatabasesdk.DistributedDatabase {
	now := &common.SDKTime{Time: time.Unix(1713240000, 0).UTC()}
	return distributeddatabasesdk.DistributedDatabase{
		Id:                 common.String(id),
		CompartmentId:      common.String(spec.CompartmentId),
		DisplayName:        common.String(spec.DisplayName),
		TimeCreated:        now,
		TimeUpdated:        now,
		DatabaseVersion:    common.String(spec.DatabaseVersion),
		LifecycleState:     state,
		LifecycleDetails:   common.String("ready"),
		Prefix:             common.String(spec.Prefix),
		PrivateEndpointIds: append([]string(nil), spec.PrivateEndpointIds...),
		ShardingMethod:     distributeddatabasesdk.DistributedDatabaseShardingMethodEnum(spec.ShardingMethod),
		CharacterSet:       common.String(spec.CharacterSet),
		NcharacterSet:      common.String(spec.NcharacterSet),
		ListenerPort:       common.Int(spec.ListenerPort),
		OnsPortLocal:       common.Int(spec.OnsPortLocal),
		OnsPortRemote:      common.Int(spec.OnsPortRemote),
		DbDeploymentType:   distributeddatabasesdk.DistributedDatabaseDbDeploymentTypeEnum(spec.DbDeploymentType),
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

func distributedDatabaseSerializedRequestBody(
	t *testing.T,
	request distributedDatabaseRequestBodyBuilder,
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
