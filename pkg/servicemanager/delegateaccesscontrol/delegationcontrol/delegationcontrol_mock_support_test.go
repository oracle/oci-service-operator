/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package delegationcontrol

import delegateaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol"

type mockDelegationControlClient struct {
	delegateaccesscontrolsdk.DelegateAccessControlClient
	delegateaccesscontrolsdk.WorkRequestClient
}
