/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitylist

func init() {
	// SecurityList intentionally stays on the explicit parity runtime rather than
	// the generatedruntime baseline until nested rule status clearing can be
	// preserved without broad generic-runtime changes.
	newSecurityListServiceClient = func(manager *SecurityListServiceManager) SecurityListServiceClient {
		return newExplicitSecurityListServiceClient(manager)
	}
}
