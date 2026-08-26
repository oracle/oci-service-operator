/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func TestManagerDeploymentTemplateSatisfiesRestrictedPodSecurity(t *testing.T) {
	t.Parallel()

	for _, dedicatedServiceAccount := range []bool{false, true} {
		content, err := renderManagerDeploymentFile(dedicatedServiceAccount)
		if err != nil {
			t.Fatalf("renderManagerDeploymentFile(%t) error = %v", dedicatedServiceAccount, err)
		}

		assertRestrictedManagerDeployment(t, "manager deployment template", content)
	}
}

func TestCheckedInManagerDeploymentsSatisfyRestrictedPodSecurity(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("..", "..", "config", "manager", "*", "manager.yaml"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	paths = append(paths, filepath.Join("..", "..", "config", "manager", "manager.yaml"))
	slices.Sort(paths)

	for _, path := range paths {
		path := path
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", path, err)
			}
			assertRestrictedManagerDeployment(t, path, string(content))
		})
	}
}

func assertRestrictedManagerDeployment(t *testing.T, source string, content string) {
	t.Helper()

	decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(content), 4096)
	var deployment *appsv1.Deployment
	for {
		var candidate appsv1.Deployment
		err := decoder.Decode(&candidate)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode %s: %v", source, err)
		}
		if candidate.Kind == "Deployment" {
			deployment = &candidate
			break
		}
	}
	if deployment == nil {
		t.Fatalf("%s did not contain a Deployment", source)
	}

	podSpec := deployment.Spec.Template.Spec
	if podSpec.SecurityContext == nil {
		t.Fatalf("%s does not define a pod security context", source)
	}
	if podSpec.SecurityContext.RunAsNonRoot == nil || !*podSpec.SecurityContext.RunAsNonRoot {
		t.Errorf("%s does not require runAsNonRoot", source)
	}
	if podSpec.SecurityContext.RunAsUser == nil || *podSpec.SecurityContext.RunAsUser != 65532 {
		t.Errorf("%s runAsUser = %v, want 65532", source, podSpec.SecurityContext.RunAsUser)
	}
	if podSpec.SecurityContext.SeccompProfile == nil || podSpec.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Errorf("%s does not use the RuntimeDefault seccomp profile", source)
	}

	for _, volume := range podSpec.Volumes {
		if volume.HostPath != nil {
			t.Errorf("%s volume %q uses forbidden hostPath %q", source, volume.Name, volume.HostPath.Path)
		}
	}

	if len(podSpec.Containers) == 0 {
		t.Fatalf("%s does not define a manager container", source)
	}
	for _, container := range podSpec.Containers {
		securityContext := container.SecurityContext
		if securityContext == nil {
			t.Errorf("%s container %q does not define a security context", source, container.Name)
			continue
		}
		if securityContext.AllowPrivilegeEscalation == nil || *securityContext.AllowPrivilegeEscalation {
			t.Errorf("%s container %q does not disable privilege escalation", source, container.Name)
		}
		if securityContext.Capabilities == nil || !slices.Contains(securityContext.Capabilities.Drop, corev1.Capability("ALL")) {
			t.Errorf("%s container %q does not drop all capabilities", source, container.Name)
		}
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == "/etc/pki" {
				t.Errorf("%s container %q masks the image trust store with volume %q", source, container.Name, mount.Name)
			}
		}
	}
}
