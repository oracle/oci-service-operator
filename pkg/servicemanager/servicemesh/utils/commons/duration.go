/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import "time"

const (
	// Delay duration parameters for rate limiter exponential back off
	MaxDelay           = 10 * time.Second
	MaxControllerDelay = 10 * time.Minute
	BaseDelay          = 5 * time.Millisecond

	PollInterval = time.Second * 5

	// Requeue request to sync resources from k8s to controlplane every 60 mins
	RequeueSyncDuration = time.Minute * 60

	PollControlPlaneEndpointInterval = time.Minute * 10

	ControlPlaneEndpointSleepInterval = time.Second * 30
)
