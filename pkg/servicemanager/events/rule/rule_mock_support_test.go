/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package rule

import (
	eventssdk "github.com/oracle/oci-go-sdk/v65/events"
	eventsv1beta1 "github.com/oracle/oci-service-operator/api/events/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newMockRuleClient(sdkClient eventssdk.EventsClient) RuleServiceClient {
	manager := &RuleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newRuleRuntimeHooks(manager, sdkClient)
	delegate := defaultRuleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*eventsv1beta1.Rule](buildRuleGeneratedRuntimeConfig(manager, hooks))}
	return wrapRuleGeneratedClient(hooks, delegate)
}
