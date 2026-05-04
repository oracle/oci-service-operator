/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package occcapacityrequest

import (
	"context"
	"fmt"

	capacitymanagementv1beta1 "github.com/oracle/oci-service-operator/api/capacitymanagement/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"k8s.io/apimachinery/pkg/runtime"
)

type occCapacityRequestDeleteResultClient interface {
	DeleteWithResult(context.Context, *capacitymanagementv1beta1.OccCapacityRequest) (servicemanager.OSOKDeleteResult, error)
}

var _ servicemanager.OSOKDeleteResultProvider = (*OccCapacityRequestServiceManager)(nil)

func (c *OccCapacityRequestServiceManager) DeleteWithResult(
	ctx context.Context,
	obj runtime.Object,
) (servicemanager.OSOKDeleteResult, error) {
	resource, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKDeleteResult{}, err
	}

	client, ok := c.client.(occCapacityRequestDeleteResultClient)
	if !ok {
		deleted, err := c.client.Delete(ctx, resource)
		return servicemanager.OSOKDeleteResult{Deleted: deleted}, err
	}

	result, err := client.DeleteWithResult(ctx, resource)
	if err != nil {
		return servicemanager.OSOKDeleteResult{}, fmt.Errorf("delete OccCapacityRequest with result: %w", err)
	}
	return result, nil
}
