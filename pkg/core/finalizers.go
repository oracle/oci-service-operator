/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package core

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Finalizer interface {
	AddFinalizers(obj client.Object, finalizers ...string)
	RemoveFinalizer(obj client.Object, finalizers ...string)
}

func NewBaseFinalizer(client client.Client, log logr.Logger) Finalizer {
	return &baseFinalizer{
		client: client,
		log:    log,
	}
}

type baseFinalizer struct {
	client client.Client
	log    logr.Logger
}

func (b baseFinalizer) AddFinalizers(obj client.Object, finalizers ...string) {
	for _, finalizer := range finalizers {
		if !HasFinalizer(obj, finalizer) {
			controllerutil.AddFinalizer(obj, finalizer)
		}
	}
	return
}

func (b baseFinalizer) RemoveFinalizer(obj client.Object, finalizers ...string) {
	for _, finalizer := range finalizers {
		if HasFinalizer(obj, finalizer) {
			controllerutil.RemoveFinalizer(obj, finalizer)
		}
	}
	return
}

func HasFinalizer(obj client.Object, finalizer string) bool {
	f := obj.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return true
		}
	}
	return false
}
