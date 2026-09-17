/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package webappaccelerationpolicy

import waasdk "github.com/oracle/oci-go-sdk/v65/waa"

type mockWebAppAccelerationPolicyOCIClient struct {
	waasdk.WaaClient
	waasdk.WorkRequestClient
}
