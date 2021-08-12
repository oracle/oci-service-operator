/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

type UserAuthConfig struct {
	Region      string `json:"region"`
	Tenancy     string `json:"tenancy"`
	User        string `json:"user"`
	PrivateKey  string `json:"key"`
	Fingerprint string `json:"fingerprint"`
	Passphrase  string `json:"passphrase"`
}
