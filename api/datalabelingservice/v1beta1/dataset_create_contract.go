/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

// DatasetCreateSourceDetails defines the supported create-time source shape for
// the published Dataset runtime.
type DatasetCreateSourceDetails struct {
	// SourceType must remain OBJECT_STORAGE for the published Dataset rollout.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=OBJECT_STORAGE
	SourceType string `json:"sourceType"`
	// The namespace of the bucket that contains the dataset data source.
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
	// The object storage bucket that contains the dataset data source.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// A common path prefix shared by the objects that make up the dataset.
	// +kubebuilder:validation:Optional
	Prefix string `json:"prefix,omitempty"`
}

// DatasetCreateTextFileTypeMetadata defines the supported create-time text file
// metadata shape for the published Dataset runtime.
type DatasetCreateTextFileTypeMetadata struct {
	// FormatType must remain DELIMITED for the published Dataset rollout.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=DELIMITED
	FormatType string `json:"formatType"`
	// The index of a selected column. This is a zero-based index.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	ColumnIndex *int `json:"columnIndex"`
	// The name of a selected column.
	// +kubebuilder:validation:Optional
	ColumnName string `json:"columnName,omitempty"`
	// A column delimiter
	// +kubebuilder:validation:Optional
	ColumnDelimiter string `json:"columnDelimiter,omitempty"`
	// A line delimiter.
	// +kubebuilder:validation:Optional
	LineDelimiter string `json:"lineDelimiter,omitempty"`
	// An escape character.
	// +kubebuilder:validation:Optional
	EscapeCharacter string `json:"escapeCharacter,omitempty"`
}

// DatasetCreateFormatDetails defines the supported create-time dataset format
// shape for the published Dataset runtime.
type DatasetCreateFormatDetails struct {
	// FormatType selects the supported published dataset format variant.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=DOCUMENT;IMAGE;TEXT
	FormatType string `json:"formatType"`
	// TextFileTypeMetadata must be set when FormatType is TEXT.
	// +kubebuilder:validation:Optional
	TextFileTypeMetadata *DatasetCreateTextFileTypeMetadata `json:"textFileTypeMetadata,omitempty"`
}

// DatasetCreateImportMetadataPath defines the supported create-time import
// metadata path shape for the published Dataset runtime.
type DatasetCreateImportMetadataPath struct {
	// SourceType must remain OBJECT_STORAGE for the published Dataset rollout.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=OBJECT_STORAGE
	SourceType string `json:"sourceType"`
	// Bucket namespace name
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
	// Bucket name
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`
	// Path for the metadata file.
	// +kubebuilder:validation:Required
	Path string `json:"path"`
}

// DatasetCreateInitialImportDatasetConfiguration defines the supported
// create-time import configuration shape for the published Dataset runtime.
type DatasetCreateInitialImportDatasetConfiguration struct {
	// +kubebuilder:validation:Required
	ImportFormat DatasetInitialImportDatasetConfigurationImportFormat `json:"importFormat"`
	// +kubebuilder:validation:Required
	ImportMetadataPath DatasetCreateImportMetadataPath `json:"importMetadataPath"`
}
