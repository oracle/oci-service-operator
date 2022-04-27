/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import (
	"math"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/servicemesh"
)

func GetDefaultExponentialRetryPolicy() *common.RetryPolicy {
	// number of times to do the retry
	attempts := uint(3)
	// retry for all non-200 status code
	retryOnAllNon200ResponseCodes := func(r common.OCIOperationResponse) bool {
		// Response is nil if r.Error is non-nil
		if r.Error != nil {
			return true
		}
		res := r.Response.HTTPResponse()
		defer res.Body.Close()
		return !(199 < res.StatusCode && res.StatusCode < 300)
	}
	return getExponentialBackoffRetryPolicy(attempts, retryOnAllNon200ResponseCodes)
}

func getExponentialBackoffRetryPolicy(n uint, fn func(r common.OCIOperationResponse) bool) *common.RetryPolicy {
	// the duration between each retry operation, you might want to wait longer each time the retry fails
	// this function returns duration as 1s, 2s, 4s, 8s, etc.
	exponentialBackoff := func(r common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(r.AttemptNumber-1))) * time.Second
	}
	policy := common.NewRetryPolicy(n, fn, exponentialBackoff)
	return &policy
}

func GetServiceMeshRetryPolicy(resource MeshResources) *common.RetryPolicy {
	// number of times to do the retry
	attempts := uint(5)
	// retry for all intermediate states
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if response.Error != nil {
			return true
		}
		switch resource {
		case Mesh:
			if resp, ok := response.Response.(servicemesh.GetMeshResponse); ok {
				return resp.LifecycleState == "CREATING" || resp.LifecycleState == "UPDATING" || resp.LifecycleState == "DELETING"
			}
		case VirtualService:
			if resp, ok := response.Response.(servicemesh.GetVirtualServiceResponse); ok {
				return resp.LifecycleState == "CREATING" || resp.LifecycleState == "UPDATING" || resp.LifecycleState == "DELETING"
			}
		case VirtualDeployment:
			if resp, ok := response.Response.(servicemesh.GetVirtualDeploymentResponse); ok {
				return resp.LifecycleState == "CREATING" || resp.LifecycleState == "UPDATING" || resp.LifecycleState == "DELETING"
			}
		case VirtualServiceRouteTable:
			if resp, ok := response.Response.(servicemesh.GetVirtualServiceRouteTableResponse); ok {
				return resp.LifecycleState == "CREATING" || resp.LifecycleState == "UPDATING" || resp.LifecycleState == "DELETING"
			}
		case AccessPolicy:
			if resp, ok := response.Response.(servicemesh.GetAccessPolicyResponse); ok {
				return resp.LifecycleState == "CREATING" || resp.LifecycleState == "UPDATING" || resp.LifecycleState == "DELETING"
			}
		case IngressGateway:
			if resp, ok := response.Response.(servicemesh.GetIngressGatewayResponse); ok {
				return resp.LifecycleState == "CREATING" || resp.LifecycleState == "UPDATING" || resp.LifecycleState == "DELETING"
			}
		case IngressGatewayRouteTable:
			if resp, ok := response.Response.(servicemesh.GetIngressGatewayRouteTableResponse); ok {
				return resp.LifecycleState == "CREATING" || resp.LifecycleState == "UPDATING" || resp.LifecycleState == "DELETING"
			}
		default:
			return false
		}
		return true
	}
	return getExponentialBackoffRetryPolicy(attempts, shouldRetry)
}
