/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package services

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

// ServiceMeshClient provides simple CRUD APIs and process requests to controlplane for all
// kinds of resources.
type ServiceMeshClient interface {
	GetMesh(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error)
	CreateMesh(ctx context.Context, mesh *sdk.Mesh, opcRetryToken *string) (*sdk.Mesh, error)
	UpdateMesh(ctx context.Context, mesh *sdk.Mesh) error
	DeleteMesh(ctx context.Context, meshId *api.OCID) error
	ChangeMeshCompartment(ctx context.Context, meshId *api.OCID, compartmentId *api.OCID) error
	GetVirtualService(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error)
	CreateVirtualService(ctx context.Context, virtualService *sdk.VirtualService, opcRetryToken *string) (*sdk.VirtualService, error)
	UpdateVirtualService(ctx context.Context, virtualService *sdk.VirtualService) error
	DeleteVirtualService(ctx context.Context, virtualServiceId *api.OCID) error
	ChangeVirtualServiceCompartment(ctx context.Context, virtualServiceId *api.OCID, compartmentId *api.OCID) error
	GetVirtualDeployment(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error)
	CreateVirtualDeployment(ctx context.Context, vd *sdk.VirtualDeployment, opcRetryToken *string) (*sdk.VirtualDeployment, error)
	UpdateVirtualDeployment(ctx context.Context, vd *sdk.VirtualDeployment) error
	DeleteVirtualDeployment(ctx context.Context, vd *api.OCID) error
	ChangeVirtualDeploymentCompartment(ctx context.Context, vd *api.OCID, compartmentId *api.OCID) error
	GetVirtualServiceRouteTable(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error)
	CreateVirtualServiceRouteTable(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable, opcRetryToken *string) (*sdk.VirtualServiceRouteTable, error)
	UpdateVirtualServiceRouteTable(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable) error
	DeleteVirtualServiceRouteTable(ctx context.Context, vsrtId *api.OCID) error
	ChangeVirtualServiceRouteTableCompartment(ctx context.Context, vsrtId *api.OCID, compartmentId *api.OCID) error
	GetAccessPolicy(ctx context.Context, accessPolicyId *api.OCID) (*sdk.AccessPolicy, error)
	CreateAccessPolicy(ctx context.Context, accessPolicy *sdk.AccessPolicy, opcRetryToken *string) (*sdk.AccessPolicy, error)
	UpdateAccessPolicy(ctx context.Context, accessPolicy *sdk.AccessPolicy) error
	DeleteAccessPolicy(ctx context.Context, accessPolicyId *api.OCID) error
	ChangeAccessPolicyCompartment(ctx context.Context, accessPolicyId *api.OCID, compartmentId *api.OCID) error
	GetIngressGateway(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error)
	CreateIngressGateway(ctx context.Context, ingressGateway *sdk.IngressGateway, opcRetryToken *string) (*sdk.IngressGateway, error)
	UpdateIngressGateway(ctx context.Context, ingressGateway *sdk.IngressGateway) error
	DeleteIngressGateway(ctx context.Context, ingressGatewayId *api.OCID) error
	ChangeIngressGatewayCompartment(ctx context.Context, ingressGatewayId *api.OCID, compartmentId *api.OCID) error
	GetIngressGatewayRouteTable(ctx context.Context, igrtId *api.OCID) (*sdk.IngressGatewayRouteTable, error)
	CreateIngressGatewayRouteTable(ctx context.Context, igrt *sdk.IngressGatewayRouteTable, opcRetryToken *string) (*sdk.IngressGatewayRouteTable, error)
	UpdateIngressGatewayRouteTable(ctx context.Context, igrt *sdk.IngressGatewayRouteTable) error
	DeleteIngressGatewayRouteTable(ctx context.Context, igrtId *api.OCID) error
	ChangeIngressGatewayRouteTableCompartment(ctx context.Context, igrtId *api.OCID, compartmentId *api.OCID) error
	GetProxyDetails(ctx context.Context) (*string, error)
	SetClientHost(cpEndpoint string)
}

type defaultServiceMeshClient struct {
	client sdk.ServiceMeshClient
	log    loggerutil.OSOKLogger
}

func NewServiceMeshClient(provider common.ConfigurationProvider, log loggerutil.OSOKLogger, operatorConditionName string) (ServiceMeshClient, error) {
	log.InfoLog("initializing service mesh client control plane")
	client, err := getServiceMeshClient(provider, log)
	if err != nil {
		return nil, err
	}
	client.UserAgent = client.UserAgent + "/OSOK_SERVICEMESH" + "/" + operatorConditionName
	return &defaultServiceMeshClient{
		client: client,
		log:    log,
	}, nil
}

func getServiceMeshClient(provider common.ConfigurationProvider, log loggerutil.OSOKLogger) (sdk.ServiceMeshClient, error) {
	client, err := sdk.NewServiceMeshClientWithConfigurationProvider(provider)
	if err != nil {
		log.ErrorLog(err, "failed to connect to the control plane")
		return sdk.ServiceMeshClient{}, err
	}
	client.SetCustomClientConfiguration(common.CustomClientConfiguration{RetryPolicy: commons.GetDefaultExponentialRetryPolicy()})
	log.InfoLog("initialized service mesh client control plane")
	return client, nil
}

func (c *defaultServiceMeshClient) GetMesh(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
	getMeshRequest := sdk.GetMeshRequest{MeshId: (*string)(meshId)}
	getMeshRequest.RequestMetadata.RetryPolicy = commons.GetServiceMeshRetryPolicy(commons.Mesh)
	response, err := c.client.GetMesh(ctx, getMeshRequest)
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "failed to get mesh from control plane", "meshId", conversions.DeRefString((*string)(meshId)))
		return nil, err
	}
	return &response.Mesh, nil
}

func (c *defaultServiceMeshClient) CreateMesh(ctx context.Context, mesh *sdk.Mesh, opcRetryToken *string) (*sdk.Mesh, error) {
	response, err := c.client.CreateMesh(ctx, sdk.CreateMeshRequest{
		CreateMeshDetails: sdk.CreateMeshDetails{
			CompartmentId:          mesh.CompartmentId,
			DisplayName:            mesh.DisplayName,
			Description:            mesh.Description,
			CertificateAuthorities: mesh.CertificateAuthorities,
			Mtls:                   mesh.Mtls,
			FreeformTags:           mesh.FreeformTags,
			DefinedTags:            mesh.DefinedTags,
		},
		OpcRetryToken: opcRetryToken})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "failed to create mesh in ControlPlane", "meshName", conversions.DeRefString(mesh.DisplayName))
		return nil, err
	}

	return &response.Mesh, nil
}

func (c *defaultServiceMeshClient) UpdateMesh(ctx context.Context, mesh *sdk.Mesh) error {
	_, err := c.client.UpdateMesh(ctx, sdk.UpdateMeshRequest{
		MeshId: mesh.Id,
		UpdateMeshDetails: sdk.UpdateMeshDetails{
			DisplayName:  mesh.DisplayName,
			Description:  mesh.Description,
			Mtls:         mesh.Mtls,
			FreeformTags: mesh.FreeformTags,
			DefinedTags:  mesh.DefinedTags,
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "failed to update mesh in ControlPlane", "meshId", conversions.DeRefString(mesh.Id))
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) DeleteMesh(ctx context.Context, meshId *api.OCID) error {
	_, err := c.client.DeleteMesh(ctx, sdk.DeleteMeshRequest{
		MeshId: (*string)(meshId),
	})
	return meshErrors.IsDeleted(ctx, err, c.log)
}

func (c *defaultServiceMeshClient) ChangeMeshCompartment(ctx context.Context, meshId *api.OCID, compartmentId *api.OCID) error {
	_, err := c.client.ChangeMeshCompartment(ctx, sdk.ChangeMeshCompartmentRequest{
		MeshId: (*string)(meshId),
		ChangeMeshCompartmentDetails: sdk.ChangeMeshCompartmentDetails{
			CompartmentId: (*string)(compartmentId),
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to change mesh compartmentId in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) GetVirtualService(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
	getVirtualServiceRequest := sdk.GetVirtualServiceRequest{VirtualServiceId: (*string)(virtualServiceId)}
	getVirtualServiceRequest.RequestMetadata.RetryPolicy = commons.GetServiceMeshRetryPolicy(commons.VirtualService)
	response, err := c.client.GetVirtualService(ctx, getVirtualServiceRequest)
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to get VirtualService from ControlPlane")
		return nil, err
	}
	return &response.VirtualService, nil
}

func (c *defaultServiceMeshClient) CreateVirtualService(ctx context.Context, virtualService *sdk.VirtualService, opcRetryToken *string) (*sdk.VirtualService, error) {
	createVirtualServiceDetails := sdk.CreateVirtualServiceDetails{
		CompartmentId:        virtualService.CompartmentId,
		Name:                 virtualService.Name,
		Description:          virtualService.Description,
		MeshId:               virtualService.MeshId,
		Hosts:                virtualService.Hosts,
		DefaultRoutingPolicy: virtualService.DefaultRoutingPolicy,
		FreeformTags:         virtualService.FreeformTags,
		DefinedTags:          virtualService.DefinedTags,
	}
	if virtualService.Mtls != nil {
		createVirtualServiceDetails.Mtls = &sdk.CreateMutualTransportLayerSecurityDetails{Mode: virtualService.Mtls.Mode}
	}

	response, err := c.client.CreateVirtualService(ctx, sdk.CreateVirtualServiceRequest{
		CreateVirtualServiceDetails: createVirtualServiceDetails,
		OpcRetryToken:               opcRetryToken,
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create VirtualService in ControlPlane")
		return nil, err
	}

	return &response.VirtualService, nil
}

func (c *defaultServiceMeshClient) UpdateVirtualService(ctx context.Context, virtualService *sdk.VirtualService) error {
	updateVirtualServiceDetails := sdk.UpdateVirtualServiceDetails{
		Description:          virtualService.Description,
		Hosts:                virtualService.Hosts,
		DefaultRoutingPolicy: virtualService.DefaultRoutingPolicy,
		FreeformTags:         virtualService.FreeformTags,
		DefinedTags:          virtualService.DefinedTags,
	}
	if virtualService.Mtls != nil {
		updateVirtualServiceDetails.Mtls = &sdk.CreateMutualTransportLayerSecurityDetails{Mode: virtualService.Mtls.Mode}
	}

	_, err := c.client.UpdateVirtualService(ctx, sdk.UpdateVirtualServiceRequest{
		VirtualServiceId:            virtualService.Id,
		UpdateVirtualServiceDetails: updateVirtualServiceDetails,
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to update VirtualService in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) DeleteVirtualService(ctx context.Context, virtualServiceId *api.OCID) error {
	_, err := c.client.DeleteVirtualService(ctx, sdk.DeleteVirtualServiceRequest{VirtualServiceId: (*string)(virtualServiceId)})
	return meshErrors.IsDeleted(ctx, err, c.log)
}

func (c *defaultServiceMeshClient) ChangeVirtualServiceCompartment(ctx context.Context, virtualServiceId *api.OCID, compartmentId *api.OCID) error {
	_, err := c.client.ChangeVirtualServiceCompartment(ctx, sdk.ChangeVirtualServiceCompartmentRequest{
		VirtualServiceId: (*string)(virtualServiceId),
		ChangeVirtualServiceCompartmentDetails: sdk.ChangeVirtualServiceCompartmentDetails{
			CompartmentId: (*string)(compartmentId),
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to change virtual service compartmentId in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) GetVirtualDeployment(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error) {
	getVirtualDeploymentRequest := sdk.GetVirtualDeploymentRequest{VirtualDeploymentId: (*string)(virtualDeploymentId)}
	getVirtualDeploymentRequest.RequestMetadata.RetryPolicy = commons.GetServiceMeshRetryPolicy(commons.VirtualDeployment)
	response, err := c.client.GetVirtualDeployment(ctx, getVirtualDeploymentRequest)
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to get VirtualDeployment from ControlPlane")
		return nil, err
	}
	return &response.VirtualDeployment, nil
}

func (c *defaultServiceMeshClient) CreateVirtualDeployment(ctx context.Context, vd *sdk.VirtualDeployment, opcRetryToken *string) (*sdk.VirtualDeployment, error) {
	response, err := c.client.CreateVirtualDeployment(ctx, sdk.CreateVirtualDeploymentRequest{
		CreateVirtualDeploymentDetails: sdk.CreateVirtualDeploymentDetails{
			Name:             vd.Name,
			Description:      vd.Description,
			CompartmentId:    vd.CompartmentId,
			VirtualServiceId: vd.VirtualServiceId,
			Listeners:        vd.Listeners,
			ServiceDiscovery: vd.ServiceDiscovery,
			AccessLogging:    vd.AccessLogging,
			DefinedTags:      vd.DefinedTags,
			FreeformTags:     vd.FreeformTags,
		},
		OpcRetryToken: opcRetryToken,
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create VirtualDeployment in ControlPlane")
		return nil, err
	}

	return &response.VirtualDeployment, nil
}

func (c *defaultServiceMeshClient) UpdateVirtualDeployment(ctx context.Context, vd *sdk.VirtualDeployment) error {
	_, err := c.client.UpdateVirtualDeployment(ctx, sdk.UpdateVirtualDeploymentRequest{
		UpdateVirtualDeploymentDetails: sdk.UpdateVirtualDeploymentDetails{
			Description:      vd.Description,
			Listeners:        vd.Listeners,
			ServiceDiscovery: vd.ServiceDiscovery,
			AccessLogging:    vd.AccessLogging,
			DefinedTags:      vd.DefinedTags,
			FreeformTags:     vd.FreeformTags,
		},
		VirtualDeploymentId: vd.Id,
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to update VirtualDeployment in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) DeleteVirtualDeployment(ctx context.Context, vd *api.OCID) error {
	_, err := c.client.DeleteVirtualDeployment(ctx, sdk.DeleteVirtualDeploymentRequest{VirtualDeploymentId: (*string)(vd)})
	return meshErrors.IsDeleted(ctx, err, c.log)
}

func (c *defaultServiceMeshClient) ChangeVirtualDeploymentCompartment(ctx context.Context, vd *api.OCID, compartmentId *api.OCID) error {
	_, err := c.client.ChangeVirtualDeploymentCompartment(ctx, sdk.ChangeVirtualDeploymentCompartmentRequest{
		VirtualDeploymentId: (*string)(vd),
		ChangeVirtualDeploymentCompartmentDetails: sdk.ChangeVirtualDeploymentCompartmentDetails{
			CompartmentId: (*string)(compartmentId),
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to change virtual deployment compartmentId in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) GetVirtualServiceRouteTable(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
	getVirtualServiceRouteTableRequest := sdk.GetVirtualServiceRouteTableRequest{VirtualServiceRouteTableId: (*string)(vsrtId)}
	getVirtualServiceRouteTableRequest.RequestMetadata.RetryPolicy = commons.GetServiceMeshRetryPolicy(commons.VirtualServiceRouteTable)
	response, err := c.client.GetVirtualServiceRouteTable(ctx, getVirtualServiceRouteTableRequest)
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to get virtual service route table from ControlPlane")
		return nil, err
	}
	return &response.VirtualServiceRouteTable, nil
}

func (c *defaultServiceMeshClient) CreateVirtualServiceRouteTable(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable, opcRetryToken *string) (*sdk.VirtualServiceRouteTable, error) {
	response, err := c.client.CreateVirtualServiceRouteTable(ctx, sdk.CreateVirtualServiceRouteTableRequest{
		CreateVirtualServiceRouteTableDetails: sdk.CreateVirtualServiceRouteTableDetails{
			VirtualServiceId: vsrt.VirtualServiceId,
			RouteRules:       vsrt.RouteRules,
			CompartmentId:    vsrt.CompartmentId,
			Name:             vsrt.Name,
			Description:      vsrt.Description,
			FreeformTags:     vsrt.FreeformTags,
			DefinedTags:      vsrt.DefinedTags,
			Priority:         vsrt.Priority,
		},
		OpcRetryToken: opcRetryToken,
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create virtual service route table in ControlPlane")
		return nil, err
	}

	return &response.VirtualServiceRouteTable, nil
}

func (c *defaultServiceMeshClient) UpdateVirtualServiceRouteTable(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable) error {
	_, err := c.client.UpdateVirtualServiceRouteTable(ctx, sdk.UpdateVirtualServiceRouteTableRequest{
		VirtualServiceRouteTableId: vsrt.Id,
		UpdateVirtualServiceRouteTableDetails: sdk.UpdateVirtualServiceRouteTableDetails{
			Description:  vsrt.Description,
			RouteRules:   vsrt.RouteRules,
			FreeformTags: vsrt.FreeformTags,
			DefinedTags:  vsrt.DefinedTags,
		}})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to update virtual service route table in ControlPlane")
		return err
	}
	return nil
}

func (c *defaultServiceMeshClient) DeleteVirtualServiceRouteTable(ctx context.Context, vsrtId *api.OCID) error {
	_, err := c.client.DeleteVirtualServiceRouteTable(ctx, sdk.DeleteVirtualServiceRouteTableRequest{VirtualServiceRouteTableId: (*string)(vsrtId)})
	return meshErrors.IsDeleted(ctx, err, c.log)
}

func (c *defaultServiceMeshClient) ChangeVirtualServiceRouteTableCompartment(ctx context.Context, vsrtId *api.OCID, compartmentId *api.OCID) error {
	_, err := c.client.ChangeVirtualServiceRouteTableCompartment(ctx, sdk.ChangeVirtualServiceRouteTableCompartmentRequest{
		VirtualServiceRouteTableId: (*string)(vsrtId),
		ChangeVirtualServiceRouteTableCompartmentDetails: sdk.ChangeVirtualServiceRouteTableCompartmentDetails{
			CompartmentId: (*string)(compartmentId),
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to change virtual service route table compartmentId in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) GetAccessPolicy(ctx context.Context, accessPolicyId *api.OCID) (*sdk.AccessPolicy, error) {
	getAccessPolicyRequest := sdk.GetAccessPolicyRequest{AccessPolicyId: (*string)(accessPolicyId)}
	getAccessPolicyRequest.RequestMetadata.RetryPolicy = commons.GetServiceMeshRetryPolicy(commons.AccessPolicy)
	response, err := c.client.GetAccessPolicy(ctx, getAccessPolicyRequest)
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to get AccessPolicy from ControlPlane")
		return nil, err
	}
	return &response.AccessPolicy, nil
}

func (c *defaultServiceMeshClient) CreateAccessPolicy(ctx context.Context, accessPolicy *sdk.AccessPolicy, opcRetryToken *string) (*sdk.AccessPolicy, error) {
	response, err := c.client.CreateAccessPolicy(ctx, sdk.CreateAccessPolicyRequest{
		CreateAccessPolicyDetails: sdk.CreateAccessPolicyDetails{
			CompartmentId: accessPolicy.CompartmentId,
			Name:          accessPolicy.Name,
			Description:   accessPolicy.Description,
			MeshId:        accessPolicy.MeshId,
			Rules:         accessPolicy.Rules,
			FreeformTags:  accessPolicy.FreeformTags,
			DefinedTags:   accessPolicy.DefinedTags,
		},
		OpcRetryToken: opcRetryToken,
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create AccessPolicy in ControlPlane")
		return nil, err
	}

	return &response.AccessPolicy, nil
}

func (c *defaultServiceMeshClient) UpdateAccessPolicy(ctx context.Context, accessPolicy *sdk.AccessPolicy) error {
	_, err := c.client.UpdateAccessPolicy(ctx, sdk.UpdateAccessPolicyRequest{
		AccessPolicyId: accessPolicy.Id,
		UpdateAccessPolicyDetails: sdk.UpdateAccessPolicyDetails{
			Description:  accessPolicy.Description,
			Rules:        accessPolicy.Rules,
			FreeformTags: accessPolicy.FreeformTags,
			DefinedTags:  accessPolicy.DefinedTags,
		}})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to update AccessPolicy in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) DeleteAccessPolicy(ctx context.Context, accessPolicyId *api.OCID) error {
	_, err := c.client.DeleteAccessPolicy(ctx, sdk.DeleteAccessPolicyRequest{AccessPolicyId: (*string)(accessPolicyId)})
	return meshErrors.IsDeleted(ctx, err, c.log)
}

func (c *defaultServiceMeshClient) ChangeAccessPolicyCompartment(ctx context.Context, accessPolicyId *api.OCID, compartmentId *api.OCID) error {
	_, err := c.client.ChangeAccessPolicyCompartment(ctx, sdk.ChangeAccessPolicyCompartmentRequest{
		AccessPolicyId:                       (*string)(accessPolicyId),
		ChangeAccessPolicyCompartmentDetails: sdk.ChangeAccessPolicyCompartmentDetails{CompartmentId: (*string)(compartmentId)},
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to change access policy compartmentId in ControlPlane")
		return err
	}
	return nil
}

func (c *defaultServiceMeshClient) GetIngressGateway(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error) {
	getIngressGatewayRequest := sdk.GetIngressGatewayRequest{IngressGatewayId: (*string)(ingressGatewayId)}
	getIngressGatewayRequest.RequestMetadata.RetryPolicy = commons.GetServiceMeshRetryPolicy(commons.IngressGateway)
	response, err := c.client.GetIngressGateway(ctx, getIngressGatewayRequest)
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to get IngressGateway from ControlPlane")
		return nil, err
	}
	return &response.IngressGateway, nil
}

func (c *defaultServiceMeshClient) CreateIngressGateway(ctx context.Context, ingressGateway *sdk.IngressGateway, opcRetryToken *string) (*sdk.IngressGateway, error) {
	response, err := c.client.CreateIngressGateway(ctx, sdk.CreateIngressGatewayRequest{
		CreateIngressGatewayDetails: sdk.CreateIngressGatewayDetails{
			CompartmentId: ingressGateway.CompartmentId,
			Name:          ingressGateway.Name,
			Description:   ingressGateway.Description,
			MeshId:        ingressGateway.MeshId,
			Hosts:         ingressGateway.Hosts,
			AccessLogging: ingressGateway.AccessLogging,
			FreeformTags:  ingressGateway.FreeformTags,
			DefinedTags:   ingressGateway.DefinedTags,
		},
		OpcRetryToken: opcRetryToken,
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create IngressGateway in ControlPlane")
		return nil, err
	}

	return &response.IngressGateway, nil
}

func (c *defaultServiceMeshClient) UpdateIngressGateway(ctx context.Context, ingressGateway *sdk.IngressGateway) error {
	_, err := c.client.UpdateIngressGateway(ctx, sdk.UpdateIngressGatewayRequest{
		IngressGatewayId: ingressGateway.Id,
		UpdateIngressGatewayDetails: sdk.UpdateIngressGatewayDetails{
			Description:   ingressGateway.Description,
			Hosts:         ingressGateway.Hosts,
			AccessLogging: ingressGateway.AccessLogging,
			FreeformTags:  ingressGateway.FreeformTags,
			DefinedTags:   ingressGateway.DefinedTags,
		},
	})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to update IngressGateway in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) DeleteIngressGateway(ctx context.Context, ingressGatewayId *api.OCID) error {
	_, err := c.client.DeleteIngressGateway(ctx, sdk.DeleteIngressGatewayRequest{IngressGatewayId: (*string)(ingressGatewayId)})
	return meshErrors.IsDeleted(ctx, err, c.log)
}

func (c *defaultServiceMeshClient) ChangeIngressGatewayCompartment(ctx context.Context, ingressGatewayId *api.OCID, compartmentId *api.OCID) error {
	_, err := c.client.ChangeIngressGatewayCompartment(ctx, sdk.ChangeIngressGatewayCompartmentRequest{
		IngressGatewayId: (*string)(ingressGatewayId),
		ChangeIngressGatewayCompartmentDetails: sdk.ChangeIngressGatewayCompartmentDetails{
			CompartmentId: (*string)(compartmentId),
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to change IngressGateway compartmentId in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) GetIngressGatewayRouteTable(ctx context.Context, igrtId *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
	getIngressGatewayRouteTableRequest := sdk.GetIngressGatewayRouteTableRequest{IngressGatewayRouteTableId: (*string)(igrtId)}
	getIngressGatewayRouteTableRequest.RequestMetadata.RetryPolicy = commons.GetServiceMeshRetryPolicy(commons.IngressGatewayRouteTable)
	response, err := c.client.GetIngressGatewayRouteTable(ctx, getIngressGatewayRouteTableRequest)
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "failed to get ingressGatewayRouteTable from ControlPlane", "ingressGatewayRouteTableId", igrtId)
		return nil, err
	}
	return &response.IngressGatewayRouteTable, nil
}

func (c *defaultServiceMeshClient) CreateIngressGatewayRouteTable(ctx context.Context, igrt *sdk.IngressGatewayRouteTable, opcRetryToken *string) (*sdk.IngressGatewayRouteTable, error) {
	response, err := c.client.CreateIngressGatewayRouteTable(ctx, sdk.CreateIngressGatewayRouteTableRequest{
		CreateIngressGatewayRouteTableDetails: sdk.CreateIngressGatewayRouteTableDetails{
			IngressGatewayId: igrt.IngressGatewayId,
			RouteRules:       igrt.RouteRules,
			CompartmentId:    igrt.CompartmentId,
			Name:             igrt.Name,
			Description:      igrt.Description,
			FreeformTags:     igrt.FreeformTags,
			DefinedTags:      igrt.DefinedTags,
		},
		OpcRetryToken: opcRetryToken,
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "failed to create ingressGatewayRouteTable in ControlPlane", "ingressGatewayRouteTableName", igrt.Name)
		return nil, err
	}

	return &response.IngressGatewayRouteTable, nil
}

func (c *defaultServiceMeshClient) UpdateIngressGatewayRouteTable(ctx context.Context, igrt *sdk.IngressGatewayRouteTable) error {
	_, err := c.client.UpdateIngressGatewayRouteTable(ctx, sdk.UpdateIngressGatewayRouteTableRequest{
		IngressGatewayRouteTableId: igrt.Id,
		UpdateIngressGatewayRouteTableDetails: sdk.UpdateIngressGatewayRouteTableDetails{
			Description:  igrt.Description,
			RouteRules:   igrt.RouteRules,
			FreeformTags: igrt.FreeformTags,
			DefinedTags:  igrt.DefinedTags,
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "failed to update ingressGatewayRouteTable in ControlPlane", "ingressGatewayRouteTableId", igrt.Id)
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) DeleteIngressGatewayRouteTable(ctx context.Context, igrtId *api.OCID) error {
	_, err := c.client.DeleteIngressGatewayRouteTable(ctx, sdk.DeleteIngressGatewayRouteTableRequest{
		IngressGatewayRouteTableId: (*string)(igrtId),
	})
	return meshErrors.IsDeleted(ctx, err, c.log)
}

func (c *defaultServiceMeshClient) ChangeIngressGatewayRouteTableCompartment(ctx context.Context, igrtId *api.OCID, compartmentId *api.OCID) error {
	_, err := c.client.ChangeIngressGatewayRouteTableCompartment(ctx, sdk.ChangeIngressGatewayRouteTableCompartmentRequest{
		IngressGatewayRouteTableId: (*string)(igrtId),
		ChangeIngressGatewayRouteTableCompartmentDetails: sdk.ChangeIngressGatewayRouteTableCompartmentDetails{
			CompartmentId: (*string)(compartmentId),
		},
	})

	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "Failed to change ingressGatewayRouteTable compartmentId in ControlPlane")
		return err
	}

	return nil
}

func (c *defaultServiceMeshClient) SetClientHost(cpEndpoint string) {
	if c.client.Host != cpEndpoint {
		c.client.Host = cpEndpoint
		c.log.InfoLog("Updated endpoint", "CP_ENDPOINT", cpEndpoint)
	}
}

func (c *defaultServiceMeshClient) GetProxyDetails(ctx context.Context) (*string, error) {
	response, err := c.client.GetProxyDetails(ctx, sdk.GetProxyDetailsRequest{})
	if err != nil {
		c.log.ErrorLogWithFixedMessage(ctx, err, "failed to get proxy details")
		return nil, err
	}
	proxyImage := response.ProxyDetails.ProxyImage
	return proxyImage, nil
}
