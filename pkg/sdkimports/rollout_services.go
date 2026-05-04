/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Package sdkimports keeps OCI SDK service packages vendored until the generator
// and rollout work import them directly.
package sdkimports

import (
	_ "github.com/oracle/oci-go-sdk/v65/accessgovernancecp"
	_ "github.com/oracle/oci-go-sdk/v65/adm"
	_ "github.com/oracle/oci-go-sdk/v65/aidocument"
	_ "github.com/oracle/oci-go-sdk/v65/ailanguage"
	_ "github.com/oracle/oci-go-sdk/v65/aispeech"
	_ "github.com/oracle/oci-go-sdk/v65/aivision"
	_ "github.com/oracle/oci-go-sdk/v65/analytics"
	_ "github.com/oracle/oci-go-sdk/v65/apiaccesscontrol"
	_ "github.com/oracle/oci-go-sdk/v65/apiplatform"
	_ "github.com/oracle/oci-go-sdk/v65/apmconfig"
	_ "github.com/oracle/oci-go-sdk/v65/apmcontrolplane"
	_ "github.com/oracle/oci-go-sdk/v65/apmsynthetics"
	_ "github.com/oracle/oci-go-sdk/v65/apmtraces"
	_ "github.com/oracle/oci-go-sdk/v65/artifacts"
	_ "github.com/oracle/oci-go-sdk/v65/bds"
	_ "github.com/oracle/oci-go-sdk/v65/budget"
	_ "github.com/oracle/oci-go-sdk/v65/capacitymanagement"
	_ "github.com/oracle/oci-go-sdk/v65/certificates"
	_ "github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	_ "github.com/oracle/oci-go-sdk/v65/clusterplacementgroups"
	_ "github.com/oracle/oci-go-sdk/v65/containerengine"
	_ "github.com/oracle/oci-go-sdk/v65/core"
	_ "github.com/oracle/oci-go-sdk/v65/dashboardservice"
	_ "github.com/oracle/oci-go-sdk/v65/databasemigration"
	_ "github.com/oracle/oci-go-sdk/v65/databasetools"
	_ "github.com/oracle/oci-go-sdk/v65/datalabelingservice"
	_ "github.com/oracle/oci-go-sdk/v65/datascience"
	_ "github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol"
	_ "github.com/oracle/oci-go-sdk/v65/dns"
	_ "github.com/oracle/oci-go-sdk/v65/events"
	_ "github.com/oracle/oci-go-sdk/v65/functions"
	_ "github.com/oracle/oci-go-sdk/v65/keymanagement"
	_ "github.com/oracle/oci-go-sdk/v65/limits"
	_ "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	_ "github.com/oracle/oci-go-sdk/v65/logging"
	_ "github.com/oracle/oci-go-sdk/v65/monitoring"
	_ "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	_ "github.com/oracle/oci-go-sdk/v65/nosql"
	_ "github.com/oracle/oci-go-sdk/v65/objectstorage"
	_ "github.com/oracle/oci-go-sdk/v65/ons"
	_ "github.com/oracle/oci-go-sdk/v65/psql"
	_ "github.com/oracle/oci-go-sdk/v65/queue"
	_ "github.com/oracle/oci-go-sdk/v65/secrets"
	_ "github.com/oracle/oci-go-sdk/v65/workrequests"
)
