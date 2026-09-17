/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package privilegedapicontrol

import apiaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/apiaccesscontrol"

type mockPrivilegedApiControlClient struct {
	apiaccesscontrolsdk.PrivilegedApiControlClient
	apiaccesscontrolsdk.PrivilegedApiWorkRequestClient
}
