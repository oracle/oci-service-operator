/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStringPresent(t *testing.T) {
	tests := []struct {
		name     string
		strSlice []string
		str      string
		want     bool
	}{
		{
			name:     "slice that contains the string",
			strSlice: []string{"Alice", "Bob"},
			str:      "Alice",
			want:     true,
		},
		{
			name:     "slice that does not contain the string",
			strSlice: []string{"Alice", "Bob"},
			str:      "Charlie",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsStringPresent(tt.strSlice, tt.str))
		})
	}
}
