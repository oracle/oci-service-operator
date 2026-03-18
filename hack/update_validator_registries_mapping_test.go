package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSDKMappingsAppliesLegacyStatusOverrides(t *testing.T) {
	t.Parallel()

	got := buildSDKMappings("core", "AllDrgAttachment", []string{"DrgAttachmentInfo"}, false, specTarget{})
	if len(got) != 1 {
		t.Fatalf("len(buildSDKMappings()) = %d, want 1", len(got))
	}
	if got[0].SDKStruct != "core.DrgAttachmentInfo" {
		t.Fatalf("buildSDKMappings()[0].SDKStruct = %q, want %q", got[0].SDKStruct, "core.DrgAttachmentInfo")
	}
	if got[0].APISurface != "status" {
		t.Fatalf("buildSDKMappings()[0].APISurface = %q, want %q", got[0].APISurface, "status")
	}
}

func TestBuildSDKMappingsSupportsExplicitSurfaceAndExclusionOverrides(t *testing.T) {
	t.Parallel()

	coreInstance := buildSDKMappings("core", "Instance", []string{
		"UpdateInstanceDetails",
		"Instance",
		"InstanceSummary",
	}, false, specTarget{})

	coreByStruct := make(map[string]sdkMapping, len(coreInstance))
	for _, mapping := range coreInstance {
		coreByStruct[mapping.SDKStruct] = mapping
	}
	if coreByStruct["core.Instance"].APISurface != "status" {
		t.Fatalf("core.Instance APISurface = %q, want status", coreByStruct["core.Instance"].APISurface)
	}
	if coreByStruct["core.InstanceSummary"].APISurface != "status" {
		t.Fatalf("core.InstanceSummary APISurface = %q, want status", coreByStruct["core.InstanceSummary"].APISurface)
	}
	if coreByStruct["core.UpdateInstanceDetails"].APISurface != "" {
		t.Fatalf("core.UpdateInstanceDetails APISurface = %q, want empty", coreByStruct["core.UpdateInstanceDetails"].APISurface)
	}

	loadBalancerShape := buildSDKMappings("loadbalancer", "Shape", []string{
		"UpdateLoadBalancerShapeDetails",
		"ShapeDetails",
		"LoadBalancerShape",
	}, false, specTarget{})

	shapeByStruct := make(map[string]sdkMapping, len(loadBalancerShape))
	for _, mapping := range loadBalancerShape {
		shapeByStruct[mapping.SDKStruct] = mapping
	}
	if !shapeByStruct["loadbalancer.UpdateLoadBalancerShapeDetails"].Exclude {
		t.Fatal("loadbalancer.UpdateLoadBalancerShapeDetails should be excluded")
	}
	if shapeByStruct["loadbalancer.UpdateLoadBalancerShapeDetails"].Reason == "" {
		t.Fatal("loadbalancer.UpdateLoadBalancerShapeDetails should carry an exclusion reason")
	}
	if shapeByStruct["loadbalancer.ShapeDetails"].Exclude {
		t.Fatal("loadbalancer.ShapeDetails should remain included")
	}
}

func TestParseExistingAPITargetsPreservesSDKMappingMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	registryPath := filepath.Join(dir, "registry.go")
	source := `package apispec

import (
	"reflect"

	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
)

var targets = []Target{
	{
		Name:       "CoreInstance",
		SpecType:   reflect.TypeOf(corev1beta1.InstanceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.Instance",
				APISurface: "status",
			},
			{
				SDKStruct: "core.InstanceSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: summary mapping excluded from desired-state coverage.",
			},
		},
	},
}
`
	if err := os.WriteFile(registryPath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := parseExistingAPITargets(registryPath)
	if err != nil {
		t.Fatalf("parseExistingAPITargets() error = %v", err)
	}

	target, ok := got["core.Instance"]
	if !ok {
		t.Fatalf("parsed targets missing %q", "core.Instance")
	}
	if len(target.SDKMappings) != 2 {
		t.Fatalf("len(target.SDKMappings) = %d, want 2", len(target.SDKMappings))
	}
	if target.SDKMappings[0].APISurface != "status" {
		t.Fatalf("target.SDKMappings[0].APISurface = %q, want status", target.SDKMappings[0].APISurface)
	}
	if !target.SDKMappings[1].Exclude {
		t.Fatal("target.SDKMappings[1].Exclude = false, want true")
	}
}
