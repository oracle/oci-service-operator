/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func TestGetName(t *testing.T) {
	type args struct {
		specName *v1beta1.Name
		metadata *metav1.ObjectMeta
	}
	tests := []struct {
		name string
		args args
		want *string
	}{
		{
			name: "normal case",
			args: args{
				specName: getSpecName("my-spec-name"),
				metadata: &metav1.ObjectMeta{
					Name:      "my-name",
					Namespace: "my-namespace",
				},
			},
			want: String("my-spec-name"),
		},
		{
			name: "spec name is empty",
			args: args{
				specName: getSpecName(""),
				metadata: &metav1.ObjectMeta{
					Name:      "my-name",
					Namespace: "my-namespace",
				},
			},
			want: String("my-namespace/my-name"),
		},
		{
			name: "spec name is null",
			args: args{
				specName: nil,
				metadata: &metav1.ObjectMeta{
					Name:      "my-name",
					Namespace: "my-namespace",
				},
			},
			want: String("my-namespace/my-name"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := GetSpecName(tt.args.specName, tt.args.metadata)
			assert.Equal(t, tt.want, name)
		})
	}
}

func getSpecName(name v1beta1.Name) *v1beta1.Name {
	return &name
}

func TestApiName(t *testing.T) {
	apiName := v1beta1.Name("cr-name")
	tests := []struct {
		name     string
		testName string
		want     *v1beta1.Name
	}{
		{
			name:     "get api Name",
			testName: "cr-name",
			want:     &apiName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApiName(tt.testName)
			assert.Equal(t, *tt.want, *got)
		})
	}
}
