/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package shared

import "bytes"

var nullJSONLiteral = []byte("null")

// JSONValue preserves arbitrary JSON values inside generated CRD fields.
// +kubebuilder:pruning:PreserveUnknownFields
type JSONValue struct {
	Raw []byte `json:"-" protobuf:"bytes,1,opt,name=raw"`
}

func (_ JSONValue) OpenAPISchemaType() []string {
	return nil
}

func (_ JSONValue) OpenAPISchemaFormat() string {
	return ""
}

func (j JSONValue) MarshalJSON() ([]byte, error) {
	if len(j.Raw) > 0 {
		return j.Raw, nil
	}
	return nullJSONLiteral, nil
}

func (j *JSONValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, nullJSONLiteral) {
		j.Raw = nil
		return nil
	}
	j.Raw = append(j.Raw[:0], data...)
	return nil
}
