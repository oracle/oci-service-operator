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
	limiter := DefaultControllerRateLimiter(1 * time.Minute)

	assert.Equal(t, 500*time.Millisecond, limiter.When("one"))
	assert.Equal(t, 1*time.Second, limiter.When("one"))
	assert.Equal(t, 2*time.Second, limiter.When("one"))
	assert.Equal(t, 4*time.Second, limiter.When("one"))
	assert.Equal(t, 8*time.Second, limiter.When("one"))
	assert.Equal(t, 16*time.Second, limiter.When("one"))
	assert.Equal(t, 32*time.Second, limiter.When("one"))
	assert.Equal(t, 1*time.Minute, limiter.When("one"))
	assert.Equal(t, 1*time.Minute, limiter.When("one"))
	assert.Equal(t, 1*time.Minute, limiter.When("one"))
	assert.Equal(t, 1*time.Minute, limiter.When("one"))

	assert.Equal(t, 500*time.Millisecond, limiter.When("two"))
	assert.Equal(t, 1*time.Second, limiter.When("two"))
}
