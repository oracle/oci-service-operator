/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type clusterGeneratedParityClient struct {
	manager  *ClusterServiceManager
	delegate ClusterServiceClient
}

func init() {
	generatedFactory := newClusterServiceClient
	newClusterServiceClient = func(manager *ClusterServiceManager) ClusterServiceClient {
		return &clusterGeneratedParityClient{
			manager:  manager,
			delegate: generatedFactory(manager),
		}
	}
}

func (c *clusterGeneratedParityClient) CreateOrUpdate(
	ctx context.Context,
	resource *ocvpv1beta1.Cluster,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return c.fail(resource, fmt.Errorf("cluster generatedruntime delegate is not configured"))
	}
	if clusterIdentityResolutionRequiresDisplayName(resource) {
		return c.fail(resource, fmt.Errorf("Cluster spec.displayName is required when no OCI identifier is recorded"))
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *clusterGeneratedParityClient) Delete(ctx context.Context, resource *ocvpv1beta1.Cluster) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("cluster generatedruntime delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *clusterGeneratedParityClient) fail(resource *ocvpv1beta1.Cluster, err error) (servicemanager.OSOKResponse, error) {
	if resource != nil {
		status := &resource.Status.OsokStatus
		status.Message = err.Error()
		status.Reason = string(shared.Failed)
		updatedAt := metav1.NewTime(time.Now())
		status.UpdatedAt = &updatedAt

		log := loggerutil.OSOKLogger{}
		if c.manager != nil {
			log = c.manager.Log
		}
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
			resource.Status.OsokStatus,
			shared.Failed,
			v1.ConditionFalse,
			"",
			err.Error(),
			log,
		)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func clusterIdentityResolutionRequiresDisplayName(resource *ocvpv1beta1.Cluster) bool {
	if resource == nil {
		return false
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return false
	}
	return currentClusterID(resource) == ""
}

func currentClusterID(resource *ocvpv1beta1.Cluster) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}
