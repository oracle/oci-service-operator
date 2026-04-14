/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Package shared contains reusable schema types shared across OSOK CRDs.
// +kubebuilder:object:generate=true
package shared

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OSOKConditionType string

type MapValue map[string]string

// +kubebuilder:validation:MaxLength=255
// +kubebuilder:validation:MinLength=1
type OCID string

// +kubebuilder:validation:Enum=none;lifecycle;workrequest
type OSOKAsyncSource string

// +kubebuilder:validation:Enum=create;update;delete
type OSOKAsyncPhase string

// +kubebuilder:validation:Enum=pending;succeeded;failed;canceled;attention;unknown
type OSOKAsyncNormalizedClass string

const (
	Provisioning OSOKConditionType = "Provisioning"
	Active       OSOKConditionType = "Active"
	Failed       OSOKConditionType = "Failed"
	Terminating  OSOKConditionType = "Terminating"
	Updating     OSOKConditionType = "Updating"
)

const (
	OSOKAsyncSourceNone        OSOKAsyncSource = "none"
	OSOKAsyncSourceLifecycle   OSOKAsyncSource = "lifecycle"
	OSOKAsyncSourceWorkRequest OSOKAsyncSource = "workrequest"
)

const (
	OSOKAsyncPhaseCreate OSOKAsyncPhase = "create"
	OSOKAsyncPhaseUpdate OSOKAsyncPhase = "update"
	OSOKAsyncPhaseDelete OSOKAsyncPhase = "delete"
)

const (
	OSOKAsyncClassPending   OSOKAsyncNormalizedClass = "pending"
	OSOKAsyncClassSucceeded OSOKAsyncNormalizedClass = "succeeded"
	OSOKAsyncClassFailed    OSOKAsyncNormalizedClass = "failed"
	OSOKAsyncClassCanceled  OSOKAsyncNormalizedClass = "canceled"
	OSOKAsyncClassAttention OSOKAsyncNormalizedClass = "attention"
	OSOKAsyncClassUnknown   OSOKAsyncNormalizedClass = "unknown"
)

type OSOKCondition struct {
	Type               OSOKConditionType  `json:"type"`
	Status             v1.ConditionStatus `json:"status"`
	LastTransitionTime *metav1.Time       `json:"lastTransitionTime,omitempty"`
	Message            string             `json:"message,omitempty"`
	Reason             string             `json:"reason,omitempty"`
}

type OSOKAsyncOperation struct {
	Source           OSOKAsyncSource          `json:"source"`
	Phase            OSOKAsyncPhase           `json:"phase"`
	WorkRequestID    string                   `json:"workRequestId,omitempty"`
	RawStatus        string                   `json:"rawStatus,omitempty"`
	RawOperationType string                   `json:"rawOperationType,omitempty"`
	NormalizedClass  OSOKAsyncNormalizedClass `json:"normalizedClass"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	PercentComplete *float32     `json:"percentComplete,omitempty"`
	Message         string       `json:"message,omitempty"`
	UpdatedAt       *metav1.Time `json:"updatedAt"`
}

type OSOKAsyncTracker struct {
	Current *OSOKAsyncOperation `json:"current,omitempty"`
}

type OSOKStatus struct {
	Conditions []OSOKCondition `json:"conditions,omitempty"`
	// Async is the canonical controller-owned async contract. Resource-local
	// legacy work-request fields may remain as compatibility mirrors while
	// follow-on migrations land, but new async state should project here first.
	Async       OSOKAsyncTracker `json:"async,omitempty"`
	Ocid        OCID             `json:"ocid,omitempty"`
	Message     string           `json:"message,omitempty"`
	Reason      string           `json:"reason,omitempty"`
	CreatedAt   *metav1.Time     `json:"createdAt,omitempty"`
	UpdatedAt   *metav1.Time     `json:"updatedAt,omitempty"`
	RequestedAt *metav1.Time     `json:"requestedAt,omitempty"`
	DeletedAt   *metav1.Time     `json:"deletedAt,omitempty"`
}

type TagResources struct {
	FreeFormTags map[string]string   `json:"freeformTags,omitempty"`
	DefinedTags  map[string]MapValue `json:"definedTags,omitempty"`
}

type SecretSource struct {
	SecretName string `json:"secretName,omitempty"`
}

type UsernameSource struct {
	Secret SecretSource `json:"secret,omitempty"`
}

type PasswordSource struct {
	Secret SecretSource `json:"secret,omitempty"`
}
