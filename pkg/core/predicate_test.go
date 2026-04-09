/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package core

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestReconcilePredicateAllowsGenerationChange(t *testing.T) {
	t.Parallel()

	pred := ReconcilePredicate()
	oldObj := &metav1.PartialObjectMetadata{}
	oldObj.SetGeneration(1)
	newObj := &metav1.PartialObjectMetadata{}
	newObj.SetGeneration(2)

	if !pred.Update(event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}) {
		t.Fatal("Update() should allow generation changes")
	}
}

func TestReconcilePredicateAllowsDeleteIntentUpdate(t *testing.T) {
	t.Parallel()

	pred := ReconcilePredicate()
	oldObj := &metav1.PartialObjectMetadata{}
	newObj := &metav1.PartialObjectMetadata{}
	now := metav1.Now()
	newObj.SetDeletionTimestamp(&now)

	if !pred.Update(event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}) {
		t.Fatal("Update() should allow the first delete-intent update")
	}
}

func TestReconcilePredicateRejectsMetadataOnlyUpdate(t *testing.T) {
	t.Parallel()

	pred := ReconcilePredicate()
	oldObj := &metav1.PartialObjectMetadata{}
	oldObj.SetGeneration(1)
	newObj := &metav1.PartialObjectMetadata{}
	newObj.SetGeneration(1)
	newObj.SetAnnotations(map[string]string{"example": "value"})

	if pred.Update(event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}) {
		t.Fatal("Update() should reject metadata-only updates without delete intent")
	}
}
