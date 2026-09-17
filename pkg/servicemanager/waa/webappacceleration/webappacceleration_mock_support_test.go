/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package webappacceleration

import waasdk "github.com/oracle/oci-go-sdk/v65/waa"

type mockWebAppAccelerationOCIClient struct {
	waasdk.WaaClient
	waasdk.WorkRequestClient
}
