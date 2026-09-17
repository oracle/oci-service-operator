/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var errResourceNotFound = errors.New("resource not found")

type commandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type kubectlClient struct {
	binary     string
	kubeconfig string
	runner     commandRunner
}

func newKubectlClient(binary, kubeconfig string, runner commandRunner) *kubectlClient {
	if binary == "" {
		binary = "kubectl"
	}
	if runner == nil {
		runner = execCommandRunner{}
	}
	return &kubectlClient{binary: binary, kubeconfig: kubeconfig, runner: runner}
}

func (c *kubectlClient) apply(ctx context.Context, path, namespace string) ([]byte, error) {
	return c.run(ctx, "apply", "-n", namespace, "-f", path)
}

func (c *kubectlClient) deleteFile(ctx context.Context, path, namespace string, timeout time.Duration) ([]byte, error) {
	return c.run(ctx, "delete", "-n", namespace, "-f", path, "--ignore-not-found=true", "--wait=true", "--timeout="+timeout.String())
}

func (c *kubectlClient) get(ctx context.Context, resource resourceRef) (*unstructured.Unstructured, error) {
	output, err := c.run(ctx, "get", resource.kubectlResource(), resource.Name, "-n", resource.Namespace, "-o", "json")
	if err != nil {
		if strings.Contains(strings.ToLower(string(output)), "notfound") || strings.Contains(strings.ToLower(string(output)), "not found") {
			return nil, errResourceNotFound
		}
		return nil, err
	}
	object := &unstructured.Unstructured{}
	if err := object.UnmarshalJSON(output); err != nil {
		return nil, fmt.Errorf("decode kubectl get output: %w", err)
	}
	return object, nil
}

func (c *kubectlClient) deleteResource(ctx context.Context, resource resourceRef) ([]byte, error) {
	return c.run(ctx, "delete", resource.kubectlResource(), resource.Name, "-n", resource.Namespace, "--wait=false", "--ignore-not-found=true")
}

func (c *kubectlClient) run(ctx context.Context, args ...string) ([]byte, error) {
	if c.kubeconfig != "" {
		args = append([]string{"--kubeconfig", c.kubeconfig}, args...)
	}
	output, err := c.runner.Run(ctx, c.binary, args...)
	if err != nil {
		return output, fmt.Errorf("%s %s failed: %w: %s", c.binary, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type resourceRef struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (r resourceRef) kubectlResource() string {
	if r.Group == "" {
		return strings.ToLower(r.Kind)
	}
	return strings.ToLower(r.Kind) + "." + r.Group
}
