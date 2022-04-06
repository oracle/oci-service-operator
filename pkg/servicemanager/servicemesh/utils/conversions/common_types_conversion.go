/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"errors"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func String(s string) *string {
	return &s
}

func Integer(i int) *int {
	return &i
}

func Bool(b bool) *bool {
	return &b
}

func OCID(ocid string) *api.OCID {
	return (*api.OCID)(&ocid)
}

func ApiName(name string) *v1beta1.Name {
	return (*v1beta1.Name)(&name)
}

func DeRefString(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

func Port(port int) *v1beta1.Port {
	p := v1beta1.Port(port)
	return &p
}

func PortToInt(port *v1beta1.Port) *int {
	if port == nil {
		return nil
	}
	i := int(*port)
	return &i
}

func ConvertCrdDefinedTagsToSdkDefinedTags(crdObj *map[string]api.MapValue, sdkObj *map[string]map[string]interface{}) {
	for k1, v1 := range *crdObj {
		values := map[string]interface{}{}
		for k2, v2 := range v1 {
			values[k2] = v2
		}
		(*sdkObj)[k1] = values
	}
}

func convertCrdAccessLoggingToSdkAccessLogging(crdAccessLogging *v1beta1.AccessLogging) *sdk.AccessLoggingConfiguration {
	if crdAccessLogging == nil {
		return nil
	}
	return &sdk.AccessLoggingConfiguration{
		IsEnabled: &crdAccessLogging.IsEnabled,
	}
}

// GetSpecName returns the spec name of the resource
// Returns spec.Name if present in the crd, else returns the metadata name of the resource
// appended to the namespace of the resource
func GetSpecName(specName *v1beta1.Name, metadata *metav1.ObjectMeta) *string {
	if specName != nil {
		strSpecName := string(*specName)
		if len(strSpecName) > 0 {
			return &strSpecName
		}
	}
	name := metadata.GetNamespace() + "/" + metadata.GetName()
	return &name
}

func ConvertCrdMtlsModeEnum(crdEnum v1beta1.MutualTransportLayerSecurityModeEnum) (sdk.MutualTransportLayerSecurityModeEnum, error) {
	switch crdEnum {
	case v1beta1.MutualTransportLayerSecurityModeDisabled:
		return sdk.MutualTransportLayerSecurityModeDisabled, nil
	case v1beta1.MutualTransportLayerSecurityModePermissive:
		return sdk.MutualTransportLayerSecurityModePermissive, nil
	case v1beta1.MutualTransportLayerSecurityModeStrict:
		return sdk.MutualTransportLayerSecurityModeStrict, nil
	default:
		return "", errors.New("unknown MTLS mode type")
	}
}

func ConvertSdkMtlsModeEnum(sdkEnum sdk.MutualTransportLayerSecurityModeEnum) (v1beta1.MutualTransportLayerSecurityModeEnum, error) {
	switch sdkEnum {
	case sdk.MutualTransportLayerSecurityModeDisabled:
		return v1beta1.MutualTransportLayerSecurityModeDisabled, nil
	case sdk.MutualTransportLayerSecurityModePermissive:
		return v1beta1.MutualTransportLayerSecurityModePermissive, nil
	case sdk.MutualTransportLayerSecurityModeStrict:
		return v1beta1.MutualTransportLayerSecurityModeStrict, nil
	default:
		return "", errors.New("unknown MTLS mode type")
	}
}
