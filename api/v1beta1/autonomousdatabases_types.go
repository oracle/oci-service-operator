/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AutonomousDatabasesSpec defines the desired state of AutonomousDatabases
type AutonomousDatabasesSpec struct {
	AdbId                OCID           `json:"id,omitempty"`
	CompartmentId        OCID           `json:"compartmentId,omitempty"`
	DisplayName          string         `json:"displayName,omitempty"`
	DbName               string         `json:"dbName,omitempty"`
	DbWorkload           string         `json:"dbWorkload,omitempty"`
	IsDedicated          bool           `json:"isDedicated,omitempty"`
	DbVersion            string         `json:"dbVersion,omitempty"`
	DataStorageSizeInTBs int            `json:"dataStorageSizeInTBs,omitempty"`
	CpuCoreCount         int            `json:"cpuCoreCount,omitempty"`
	AdminPassword        PasswordSource `json:"adminPassword,omitempty"`
	IsAutoScalingEnabled bool           `json:"isAutoScalingEnabled,omitempty"`
	IsFreeTier           bool           `json:"isFreeTier,omitempty"`
	LicenseModel         string         `json:"licenseModel,omitempty"`
	TagResources         `json:",inline"`
	Wallet               AutonomousDatabaseWallet `json:"wallet,omitempty"`
}

type AutonomousDatabaseWallet struct {
	WalletName     string         `json:"walletName,omitempty"`
	WalletPassword PasswordSource `json:"walletPassword,omitempty"`
}

// AutonomousDatabasesStatus defines the observed state of AutonomousDatabases
type AutonomousDatabasesStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="DbWorkload",type="string",JSONPath=".spec.dbWorkload",priority=0
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the AutonomousDatabases",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the AutonomousDatabases",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// AutonomousDatabases is the Schema for the autonomousdatabases API
type AutonomousDatabases struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutonomousDatabasesSpec   `json:"spec,omitempty"`
	Status AutonomousDatabasesStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AutonomousDatabasesList contains a list of AutonomousDatabases
type AutonomousDatabasesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AutonomousDatabases `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AutonomousDatabases{}, &AutonomousDatabasesList{})
}
