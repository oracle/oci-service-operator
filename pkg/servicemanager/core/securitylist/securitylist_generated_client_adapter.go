/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitylist

func init() {
	registerSecurityListRuntimeHooksMutator(func(manager *SecurityListServiceManager, hooks *SecurityListRuntimeHooks) {
		applySecurityListRuntimeHooks(manager, hooks)
	})
}
