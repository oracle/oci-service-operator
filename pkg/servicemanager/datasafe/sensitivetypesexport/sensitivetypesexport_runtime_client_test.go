/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivetypesexport

import (
	"reflect"
	"testing"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
)

func TestSensitiveTypesExportRuntimeHooksConfigureWorkRequestLifecycle(t *testing.T) {
	hooks := newSensitiveTypesExportDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applySensitiveTypesExportRuntimeHooks(&hooks, nil, nil)

	if hooks.Semantics == nil || hooks.Semantics.Async == nil || hooks.Semantics.Async.WorkRequest == nil {
		t.Fatal("SensitiveTypesExport work-request semantics were not configured")
	}
	if got, want := hooks.Semantics.Async.WorkRequest.Phases, []string{"create", "update"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("work-request phases = %v, want %v", got, want)
	}
	if got, want := hooks.Semantics.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mutable fields = %v, want %v", got, want)
	}
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("Async.GetWorkRequest was not configured")
	}
}
