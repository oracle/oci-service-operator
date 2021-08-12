/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MySqlDbSystemSpec defines the desired state of MySqlDbSystem
type MySqlDbSystemSpec struct {
	MySqlDbSystemId OCID           `json:"id,omitempty"`
	CompartmentId   OCID           `json:"compartmentId,omitempty"`
	ShapeName       string         `json:"shapeName,omitempty"`
	SubnetId        OCID           `json:"subnetId,omitempty"`
	AdminUsername   UsernameSource `json:"adminUsername,omitempty"`
	AdminPassword   PasswordSource `json:"adminPassword,omitempty"`
	DisplayName     string         `json:"displayName,omitempty"`
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=400
	Description        string `json:"description,omitempty"`
	AvailabilityDomain string `json:"availabilityDomain,omitempty"`
	// +kubebuilder:default:=FAULT-DOMAIN-1
	FaultDomain  string `json:"faultDomain,omitempty"`
	MysqlVersion string `json:"mysqlVersion,omitempty"`
	// +kubebuilder:validation:Minimum:=50
	// +kubebuilder:validation:Maximum:=65536
	DataStorageSizeInGBs int    `json:"dataStorageSizeInGBs,omitempty"`
	HostnameLabel        string `json:"hostnameLabel,omitempty"`
	IpAddress            string `json:"ipAddress,omitempty"`
	IsHighlyAvailable    bool   `json:"isHighlyAvailable,omitempty"`
	// +kubebuilder:validation:Minimum:=1024
	// +kubebuilder:validation:Maximum:=65535
	Port int `json:"port,omitempty"`
	// +kubebuilder:validation:Minimum:=1024
	// +kubebuilder:validation:Maximum:=65535
	PortX           int                         `json:"portX,omitempty"`
	ConfigurationId CreateConfigurationDetails  `json:"configuration,omitempty"`
	BackupPolicy    CreateBackupPolicyDetails   `json:"backupPolicy,omitempty"`
	Source          CreateDbSystemSourceDetails `json:"source,omitempty"`
	Maintenance     CreateMaintenanceDetails    `json:"maintenance,omitempty"`
	TagResources    `json:",inline,omitempty"`
}

// CreateDbSystemSourceDetails Parameters detailing how to provision the initial data of the system.
type CreateDbSystemSourceDetails struct {
	JsonData   string `json:"jsonData,omitempty"`
	SourceType string `json:"sourceType,omitempty"`
}

// CreateMaintenanceDetails The Maintenance Policy for the DB System.
type CreateMaintenanceDetails struct {

	// The start of the 2 hour maintenance window.
	// This string is of the format: "{day-of-week} {time-of-day}".
	// "{day-of-week}" is a case-insensitive string like "mon", "tue", &c.
	// "{time-of-day}" is the "Time" portion of an RFC3339-formatted timestamp. Any second or sub-second time data will be truncated to zero.
	WindowStartTime string `json:"windowStartTime,omitempty"`
}

type CreateBackupPolicyDetails struct {

	// Specifies if automatic backups are enabled.
	IsEnabled bool `json:"isEnabled,omitempty"`

	// The start of a 30-minute window of time in which daily, automated backups occur.
	// This should be in the format of the "Time" portion of an RFC3339-formatted timestamp. Any second or sub-second time data will be truncated to zero.
	// At some point in the window, the system may incur a brief service disruption as the backup is performed.
	WindowStartTime string `json:"windowStartTime,omitempty"`

	// Number of days to retain an automatic backup.
	RetentionInDays int `json:"retentionInDays,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// CreateConfigurationDetails The Configuration for the DB System.
type CreateConfigurationDetails struct {
	// Configuration Id
	Id OCID `json:"id,omitempty"`
}

// MySqlDbSystemStatus defines the observed state of MySqlDbSystem
type MySqlDbSystemStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the MySqlDbSystem",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the MySqlDbSystem",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// MySqlDbSystem is the Schema for the mysqldbsystems API
type MySqlDbSystem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MySqlDbSystemSpec   `json:"spec,omitempty"`
	Status MySqlDbSystemStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MySqlDbSystemList contains a list of MySqlDbSystem
type MySqlDbSystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MySqlDbSystem `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MySqlDbSystem{}, &MySqlDbSystemList{})
}
