/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultControllerRateLimiter(t *testing.T) {
	limiter := DefaultControllerRateLimiter()

	assert.Equal(t, 5*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 10*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 20*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 40*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 80*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 160*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 320*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 640*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 1280*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 2560*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 5120*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 10000*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 10000*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 10000*time.Millisecond, limiter.When("one"))

	assert.Equal(t, 5*time.Millisecond, limiter.When("two"))
	assert.Equal(t, 10*time.Millisecond, limiter.When("two"))
}
