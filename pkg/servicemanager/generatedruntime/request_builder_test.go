/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	databasesdk "github.com/oracle/oci-go-sdk/v65/database"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"io"
	"strings"
	"testing"
)

func TestBuildRequestPopulatesAutonomousDatabasePolymorphicCreateBody(t *testing.T) {
	t.Parallel()
	tests := []struct //nolint:gocognit,gocyclo // Table-driven coverage keeps the generated polymorphic create-body variants together.
	{
		name   string
		spec   databasev1beta1.AutonomousDatabaseSpec
		assert func(*testing.T, databasesdk.CreateAutonomousDatabaseBase)
	}{{name: "default create details", spec: databasev1beta1.AutonomousDatabaseSpec{CompartmentId: "ocid1.compartment.oc1..create", DisplayName: "adb-create"}, assert: func(t *testing.T, body databasesdk.CreateAutonomousDatabaseBase) {
		t.Helper()
		details, ok := body.(databasesdk.CreateAutonomousDatabaseDetails)
		if !ok {
			t.Fatalf("create body type = %T, want %T", body, databasesdk.CreateAutonomousDatabaseDetails{})
		}
		if details.CompartmentId == nil || *details.CompartmentId != "ocid1.compartment.oc1..create" {
			t.Fatalf("details.compartmentId = %v, want ocid1.compartment.oc1..create", details.CompartmentId)
		}
		if details.DisplayName == nil || *details.DisplayName != "adb-create" {
			t.Fatalf("details.displayName = %v, want adb-create", details.DisplayName)
		}
	}}, {name: "clone details", spec: databasev1beta1.AutonomousDatabaseSpec{CompartmentId: "ocid1.compartment.oc1..clone", DisplayName: "adb-clone", Source: "DATABASE", SourceId: "ocid1.autonomousdatabase.oc1..source"}, assert: func(t *testing.T, body databasesdk.CreateAutonomousDatabaseBase) {
		t.Helper()
		details, ok := body.(databasesdk.CreateAutonomousDatabaseCloneDetails)
		if !ok {
			t.Fatalf("create body type = %T, want %T", body, databasesdk.CreateAutonomousDatabaseCloneDetails{})
		}
		if details.SourceId == nil || *details.SourceId != "ocid1.autonomousdatabase.oc1..source" {
			t.Fatalf("details.sourceId = %v, want ocid1.autonomousdatabase.oc1..source", details.SourceId)
		}
		if details.DisplayName == nil || *details.DisplayName != "adb-clone" {
			t.Fatalf("details.displayName = %v, want adb-clone", details.DisplayName)
		}
	}}}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := &databasesdk.CreateAutonomousDatabaseRequest{}
			resource := &databasev1beta1.AutonomousDatabase{Spec: tc.spec}
			values, err := lookupValues(resource)
			if err != nil {
				t.Fatalf("lookupValues() error = %v", err)
			}
			err = buildRequest(request, resource, values, "", []RequestField{{FieldName: "CreateAutonomousDatabaseDetails", RequestName: "createAutonomousDatabaseDetails", Contribution: "body"}}, nil, requestBuildOptions{Context: context.Background()}, nil, false)
			if err != nil {
				t.Fatalf("buildRequest() error = %v", err)
			}
			if request.CreateAutonomousDatabaseDetails == nil {
				t.Fatal("buildRequest() should populate CreateAutonomousDatabaseDetails")
			}
			tc.assert(t, request.CreateAutonomousDatabaseDetails)
		})
	}
}

func TestBuildRequestPopulatesLaunchInstancePolymorphicSourceDetails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		spec   corev1beta1.InstanceSpec
		assert func(*testing.T, coresdk.LaunchInstanceDetails)
	}{{name: "image source details", spec: corev1beta1.InstanceSpec{CompartmentId: "ocid1.compartment.oc1..instance", AvailabilityDomain: "AD-1", SubnetId: "ocid1.subnet.oc1..instance", DisplayName: "instance-sample", Shape: "VM.Standard.E4.Flex", SourceDetails: corev1beta1.InstanceSourceDetails{SourceType: "image", ImageId: "ocid1.image.oc1..image", BootVolumeSizeInGBs: 50}}, assert: func(t *testing.T, details coresdk.LaunchInstanceDetails) {
		t.Helper()
		if details.CompartmentId == nil || *details.CompartmentId != "ocid1.compartment.oc1..instance" {
			t.Fatalf("details.compartmentId = %v, want ocid1.compartment.oc1..instance", details.CompartmentId)
		}
		if details.AvailabilityDomain == nil || *details.AvailabilityDomain != "AD-1" {
			t.Fatalf("details.availabilityDomain = %v, want AD-1", details.AvailabilityDomain)
		}
		source, ok := details.SourceDetails.(coresdk.InstanceSourceViaImageDetails)
		if !ok {
			t.Fatalf("details.sourceDetails type = %T, want %T", details.SourceDetails, coresdk.InstanceSourceViaImageDetails{})
		}
		if source.ImageId == nil || *source.ImageId != "ocid1.image.oc1..image" {
			t.Fatalf("image source imageId = %v, want ocid1.image.oc1..image", source.ImageId)
		}
		if source.BootVolumeSizeInGBs == nil || *source.BootVolumeSizeInGBs != 50 {
			t.Fatalf("image source bootVolumeSizeInGBs = %v, want 50", source.BootVolumeSizeInGBs)
		}
	}}, {name: "boot volume source details", spec: corev1beta1.InstanceSpec{CompartmentId: "ocid1.compartment.oc1..boot", AvailabilityDomain: "AD-2", Shape: "VM.Standard.E4.Flex", SourceDetails: corev1beta1.InstanceSourceDetails{SourceType: "bootVolume", BootVolumeId: "ocid1.bootvolume.oc1..boot"}}, assert: func(t *testing.T, details coresdk.LaunchInstanceDetails) {
		t.Helper()
		source, ok := details.SourceDetails.(coresdk.InstanceSourceViaBootVolumeDetails)
		if !ok {
			t.Fatalf("details.sourceDetails type = %T, want %T", details.SourceDetails, coresdk.InstanceSourceViaBootVolumeDetails{})
		}
		if source.BootVolumeId == nil || *source.BootVolumeId != "ocid1.bootvolume.oc1..boot" {
			t.Fatalf("boot volume source bootVolumeId = %v, want ocid1.bootvolume.oc1..boot", source.BootVolumeId)
		}
	}}}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := &coresdk.LaunchInstanceRequest{}
			resource := &corev1beta1.Instance{Spec: tc.spec}
			values, err := lookupValues(resource)
			if err != nil {
				t.Fatalf("lookupValues() error = %v", err)
			}
			err = buildRequest(request, resource, values, "", []RequestField{{FieldName: "LaunchInstanceDetails", RequestName: "LaunchInstanceDetails", Contribution: "body"}}, nil, requestBuildOptions{Context: context.Background()}, nil, false)
			if err != nil {
				t.Fatalf("buildRequest() error = %v", err)
			}
			tc.assert(t, request.LaunchInstanceDetails)
		})
	}
}

func TestBuildRequestOmitsUnsetGeneratedAdminCredentialSources(t *testing.T) {
	t.Parallel()
	mysqlRequest := &mysqlsdk.CreateDbSystemRequest{}
	mysqlResource := &mysqlv1beta1.DbSystem{Spec: mysqlv1beta1.DbSystemSpec{CompartmentId: "ocid1.compartment.oc1..mysql", ShapeName: "MySQL.VM.Standard.E4.1.8GB", SubnetId: "ocid1.subnet.oc1..mysql"}}
	mysqlValues, err := lookupValues(mysqlResource)
	if err != nil {
		t.Fatalf("lookupValues(mysql) error = %v", err)
	}
	if err := buildRequest(mysqlRequest, mysqlResource, mysqlValues, "", nil, nil, requestBuildOptions{Context: context.Background()}, nil, false); err != nil {
		t.Fatalf("buildRequest(mysql) error = %v", err)
	}
	if mysqlRequest.AdminUsername != nil {
		t.Fatalf("mysql adminUsername = %v, want nil when secret source is omitted", mysqlRequest.AdminUsername)
	}
	if mysqlRequest.AdminPassword != nil {
		t.Fatalf("mysql adminPassword = %v, want nil when secret source is omitted", mysqlRequest.AdminPassword)
	}
	adbRequest := &databasesdk.CreateAutonomousDatabaseRequest{}
	adbResource := &databasev1beta1.AutonomousDatabase{Spec: databasev1beta1.AutonomousDatabaseSpec{CompartmentId: "ocid1.compartment.oc1..adb", DisplayName: "adb-sample"}}
	adbValues, err := lookupValues(adbResource)
	if err != nil {
		t.Fatalf("lookupValues(adb) error = %v", err)
	}
	if err := buildRequest(adbRequest, adbResource, adbValues, "", []RequestField{{FieldName: "CreateAutonomousDatabaseDetails", RequestName: "createAutonomousDatabaseDetails", Contribution: "body"}}, nil, requestBuildOptions{Context: context.Background()}, nil, false); err != nil {
		t.Fatalf("buildRequest(adb) error = %v", err)
	}
	if adbRequest.CreateAutonomousDatabaseDetails == nil {
		t.Fatal("buildRequest(adb) should populate CreateAutonomousDatabaseDetails")
	}
	adbDetails, ok := adbRequest.CreateAutonomousDatabaseDetails.(databasesdk.CreateAutonomousDatabaseDetails)
	if !ok {
		t.Fatalf("create body type = %T, want %T", adbRequest.CreateAutonomousDatabaseDetails, databasesdk.CreateAutonomousDatabaseDetails{})
	}
	if adbDetails.AdminPassword != nil {
		t.Fatalf("autonomous database adminPassword = %v, want nil when secret source is omitted", adbDetails.AdminPassword)
	}
}

func TestBuildRequestOmitsZeroValueAutonomousDatabaseNestedStructs(t *testing.T) {
	t.Parallel()
	adbRequest := &databasesdk.CreateAutonomousDatabaseRequest{}
	adbResource := &databasev1beta1.AutonomousDatabase{Spec: databasev1beta1.AutonomousDatabaseSpec{CompartmentId: "ocid1.compartment.oc1..adb", DisplayName: "adb-sample", DbName: "adbsample", DbWorkload: "OLTP", IsDedicated: false, DbVersion: "19c", DataStorageSizeInTBs: 1, CpuCoreCount: 1, LicenseModel: "LICENSE_INCLUDED", IsAutoScalingEnabled: true}}
	adbValues, err := lookupValues(adbResource)
	if err != nil {
		t.Fatalf("lookupValues(adb) error = %v", err)
	}
	if err := buildRequest(adbRequest, adbResource, adbValues, "", []RequestField{{FieldName: "CreateAutonomousDatabaseDetails", RequestName: "createAutonomousDatabaseDetails", Contribution: "body"}}, nil, requestBuildOptions{Context: context.Background()}, nil, false); err != nil {
		t.Fatalf("buildRequest(adb) error = %v", err)
	}
	if adbRequest.CreateAutonomousDatabaseDetails == nil {
		t.Fatal("buildRequest(adb) should populate CreateAutonomousDatabaseDetails")
	}
	httpRequest, err := adbRequest.HTTPRequest("POST", "/autonomousDatabases", nil, nil)
	if err != nil {
		t.Fatalf("HTTPRequest(adb create body) error = %v", err)
	}
	payload, err := io.ReadAll(httpRequest.Body)
	if err != nil {
		t.Fatalf("io.ReadAll(adb create body) error = %v", err)
	}
	body := string(payload)
	for _, unwanted := range []string{`"resourcePoolSummary"`, `"longTermBackupSchedule"`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("adb create body = %s, unexpected token %s", body, unwanted)
		}
	}
	for _, wanted := range []string{`"dbVersion":"19c"`, `"cpuCoreCount":1`, `"dataStorageSizeInTBs":1`, `"isAutoScalingEnabled":true`} {
		if !strings.Contains(body, wanted) {
			t.Fatalf("adb create body = %s, missing token %s", body, wanted)
		}
	}
}
