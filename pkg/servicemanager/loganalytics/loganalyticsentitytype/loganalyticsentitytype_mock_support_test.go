/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package loganalyticsentitytype

import (
	"context"

	objectstoragesdk "github.com/oracle/oci-go-sdk/v65/objectstorage"
)

type mockEntityTypeNamespaceGetter struct{ namespace string }

func (g mockEntityTypeNamespaceGetter) GetNamespace(context.Context, objectstoragesdk.GetNamespaceRequest) (objectstoragesdk.GetNamespaceResponse, error) {
	return objectstoragesdk.GetNamespaceResponse{Value: &g.namespace}, nil
}
