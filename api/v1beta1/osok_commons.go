/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OSOKConditionType string

type MapValue map[string]string

// +kubebuilder:validation:MaxLength=255
// +kubebuilder:validation:MinLength=1
type OCID string

const (
	Provisioning OSOKConditionType = "Provisioning"
	Active       OSOKConditionType = "Active"
	Failed       OSOKConditionType = "Failed"
	Terminating  OSOKConditionType = "Terminating"
	Updating     OSOKConditionType = "Updating"
)

type OSOKCondition struct {
	Type               OSOKConditionType  `json:"type"`
	Status             v1.ConditionStatus `json:"status"`
	LastTransitionTime *metav1.Time       `json:"lastTransitionTime,omitempty"`
	Message            string             `json:"message,omitempty"`
	Reason             string             `json:"reason,omitempty"`
}

type OSOKStatus struct {
	Conditions  []OSOKCondition `json:"conditions,omitempty"`
	Ocid        OCID            `json:"ocid,omitempty"`
	Message     string          `json:"message,omitempty"`
	Reason      string          `json:"reason,omitempty"`
	CreatedAt   *metav1.Time    `json:"createdAt,omitempty"`
	UpdatedAt   *metav1.Time    `json:"updatedAt,omitempty"`
	RequestedAt *metav1.Time    `json:"requestedAt,omitempty"`
	DeletedAt   *metav1.Time    `json:"deletedAt,omitempty"`
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
