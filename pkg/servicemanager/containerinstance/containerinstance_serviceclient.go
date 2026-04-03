/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerinstancessdk "github.com/oracle/oci-go-sdk/v65/containerinstances"
	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// ContainerInstanceClientInterface defines the OCI operations used by ContainerInstanceServiceManager.
type ContainerInstanceClientInterface interface {
	CreateContainerInstance(ctx context.Context, request containerinstancessdk.CreateContainerInstanceRequest) (containerinstancessdk.CreateContainerInstanceResponse, error)
	GetContainerInstance(ctx context.Context, request containerinstancessdk.GetContainerInstanceRequest) (containerinstancessdk.GetContainerInstanceResponse, error)
	ListContainerInstances(ctx context.Context, request containerinstancessdk.ListContainerInstancesRequest) (containerinstancessdk.ListContainerInstancesResponse, error)
	UpdateContainerInstance(ctx context.Context, request containerinstancessdk.UpdateContainerInstanceRequest) (containerinstancessdk.UpdateContainerInstanceResponse, error)
	DeleteContainerInstance(ctx context.Context, request containerinstancessdk.DeleteContainerInstanceRequest) (containerinstancessdk.DeleteContainerInstanceResponse, error)
}

func getContainerInstanceClient(provider common.ConfigurationProvider) (containerinstancessdk.ContainerInstanceClient, error) {
	return containerinstancessdk.NewContainerInstanceClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *ContainerInstanceServiceManager) getOCIClient() (ContainerInstanceClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getContainerInstanceClient(c.Provider)
}

// CreateContainerInstance calls the OCI API to create a new container instance.
func (c *ContainerInstanceServiceManager) CreateContainerInstance(ctx context.Context,
	ci containerinstancesv1beta1.ContainerInstance) (containerinstancessdk.CreateContainerInstanceResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return containerinstancessdk.CreateContainerInstanceResponse{}, err
	}

	containers, err := buildCreateContainers(ci.Spec.Containers)
	if err != nil {
		return containerinstancessdk.CreateContainerInstanceResponse{}, err
	}
	vnics := buildCreateVnics(ci.Spec.Vnics)
	volumes, err := buildCreateVolumes(ci.Spec.Volumes)
	if err != nil {
		return containerinstancessdk.CreateContainerInstanceResponse{}, err
	}
	imagePullSecrets, err := buildImagePullSecrets(ci.Spec.ImagePullSecrets)
	if err != nil {
		return containerinstancessdk.CreateContainerInstanceResponse{}, err
	}

	shapeConfig := &containerinstancessdk.CreateContainerInstanceShapeConfigDetails{
		Ocpus: common.Float32(ci.Spec.ShapeConfig.Ocpus),
	}
	if ci.Spec.ShapeConfig.MemoryInGBs != 0 {
		shapeConfig.MemoryInGBs = common.Float32(ci.Spec.ShapeConfig.MemoryInGBs)
	}

	details := containerinstancessdk.CreateContainerInstanceDetails{
		CompartmentId:      common.String(ci.Spec.CompartmentId),
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
		Shape:              common.String(ci.Spec.Shape),
		ShapeConfig:        shapeConfig,
		Containers:         containers,
		Vnics:              vnics,
	}
	if ci.Spec.DisplayName != "" {
		details.DisplayName = common.String(ci.Spec.DisplayName)
	}
	if ci.Spec.FaultDomain != "" {
		details.FaultDomain = common.String(ci.Spec.FaultDomain)
	}
	if len(volumes) > 0 {
		details.Volumes = volumes
	}
	if dnsConfig := buildDNSConfig(ci.Spec.DnsConfig); dnsConfig != nil {
		details.DnsConfig = dnsConfig
	}
	if ci.Spec.GracefulShutdownTimeoutInSeconds != 0 {
		details.GracefulShutdownTimeoutInSeconds = common.Int64(ci.Spec.GracefulShutdownTimeoutInSeconds)
	}
	if len(imagePullSecrets) > 0 {
		details.ImagePullSecrets = imagePullSecrets
	}
	if ci.Spec.ContainerRestartPolicy != "" {
		details.ContainerRestartPolicy = containerinstancessdk.ContainerInstanceContainerRestartPolicyEnum(ci.Spec.ContainerRestartPolicy)
	}
	if ci.Spec.FreeformTags != nil {
		details.FreeformTags = ci.Spec.FreeformTags
	}
	if ci.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&ci.Spec.DefinedTags)
	}

	req := containerinstancessdk.CreateContainerInstanceRequest{
		CreateContainerInstanceDetails: details,
	}
	return client.CreateContainerInstance(ctx, req)
}

// GetContainerInstance retrieves a container instance by OCID.
func (c *ContainerInstanceServiceManager) GetContainerInstance(ctx context.Context, ciID shared.OCID,
	retryPolicy *common.RetryPolicy) (*containerinstancessdk.ContainerInstance, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := containerinstancessdk.GetContainerInstanceRequest{
		ContainerInstanceId: common.String(string(ciID)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetContainerInstance(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.ContainerInstance, nil
}

// GetContainerInstanceOcid looks up an existing container instance by display name.
func (c *ContainerInstanceServiceManager) GetContainerInstanceOcid(ctx context.Context,
	ci containerinstancesv1beta1.ContainerInstance) (*shared.OCID, error) {
	if strings.TrimSpace(ci.Spec.DisplayName) == "" {
		return nil, nil
	}

	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := containerinstancessdk.ListContainerInstancesRequest{
		CompartmentId:      common.String(ci.Spec.CompartmentId),
		DisplayName:        common.String(ci.Spec.DisplayName),
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
		Limit:              common.Int(50),
	}

	resp, err := client.ListContainerInstances(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing ContainerInstances")
		return nil, err
	}

	for _, item := range resp.Items {
		switch item.LifecycleState {
		case containerinstancessdk.ContainerInstanceLifecycleStateActive,
			containerinstancessdk.ContainerInstanceLifecycleStateCreating,
			containerinstancessdk.ContainerInstanceLifecycleStateUpdating,
			containerinstancessdk.ContainerInstanceLifecycleStateInactive:
			if item.Id != nil {
				c.Log.DebugLog(fmt.Sprintf("ContainerInstance %s exists with OCID %s", ci.Spec.DisplayName, *item.Id))
				ocid := shared.OCID(*item.Id)
				return &ocid, nil
			}
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ContainerInstance %s does not exist", ci.Spec.DisplayName))
	return nil, nil
}

// UpdateContainerInstance updates an existing container instance when supported mutable fields drift.
func (c *ContainerInstanceServiceManager) UpdateContainerInstance(ctx context.Context,
	ci *containerinstancesv1beta1.ContainerInstance, existing *containerinstancessdk.ContainerInstance) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}
	if err := validateImmutableContainerInstanceFields(ci, existing); err != nil {
		return err
	}

	updateDetails := containerinstancessdk.UpdateContainerInstanceDetails{}
	updateNeeded := false

	if ci.Spec.DisplayName != "" && safeString(existing.DisplayName) != ci.Spec.DisplayName {
		updateDetails.DisplayName = common.String(ci.Spec.DisplayName)
		updateNeeded = true
	}
	if ci.Spec.FreeformTags != nil && !reflect.DeepEqual(existing.FreeformTags, ci.Spec.FreeformTags) {
		updateDetails.FreeformTags = ci.Spec.FreeformTags
		updateNeeded = true
	}
	if ci.Spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&ci.Spec.DefinedTags)
		if !reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
			updateDetails.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	if !updateNeeded {
		return nil
	}

	targetID := safeString(existing.Id)
	if targetID == "" {
		targetID = string(ci.Status.OsokStatus.Ocid)
	}

	req := containerinstancessdk.UpdateContainerInstanceRequest{
		ContainerInstanceId:            common.String(targetID),
		UpdateContainerInstanceDetails: updateDetails,
	}
	_, err = client.UpdateContainerInstance(ctx, req)
	return err
}

// DeleteContainerInstance deletes the container instance for the given OCID.
func (c *ContainerInstanceServiceManager) DeleteContainerInstance(ctx context.Context, ciID shared.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := containerinstancessdk.DeleteContainerInstanceRequest{
		ContainerInstanceId: common.String(string(ciID)),
	}
	_, err = client.DeleteContainerInstance(ctx, req)
	return err
}

func buildCreateContainers(specContainers []containerinstancesv1beta1.ContainerInstanceContainer) ([]containerinstancessdk.CreateContainerDetails, error) {
	containers := make([]containerinstancessdk.CreateContainerDetails, 0, len(specContainers))
	for _, ctr := range specContainers {
		if strings.TrimSpace(ctr.ImageUrl) == "" {
			return nil, fmt.Errorf("container imageUrl is required")
		}

		cd := containerinstancessdk.CreateContainerDetails{
			ImageUrl: common.String(ctr.ImageUrl),
		}
		if ctr.DisplayName != "" {
			cd.DisplayName = common.String(ctr.DisplayName)
		}
		if len(ctr.Command) > 0 {
			cd.Command = append([]string(nil), ctr.Command...)
		}
		if len(ctr.Arguments) > 0 {
			cd.Arguments = append([]string(nil), ctr.Arguments...)
		}
		if ctr.WorkingDirectory != "" {
			cd.WorkingDirectory = common.String(ctr.WorkingDirectory)
		}
		if len(ctr.EnvironmentVariables) > 0 {
			cd.EnvironmentVariables = ctr.EnvironmentVariables
		}
		if len(ctr.VolumeMounts) > 0 {
			mounts := make([]containerinstancessdk.CreateVolumeMountDetails, 0, len(ctr.VolumeMounts))
			for _, mount := range ctr.VolumeMounts {
				detail := containerinstancessdk.CreateVolumeMountDetails{
					MountPath:  common.String(mount.MountPath),
					VolumeName: common.String(mount.VolumeName),
				}
				if mount.SubPath != "" {
					detail.SubPath = common.String(mount.SubPath)
				}
				if mount.IsReadOnly {
					detail.IsReadOnly = common.Bool(true)
				}
				if mount.Partition != 0 {
					detail.Partition = common.Int(mount.Partition)
				}
				mounts = append(mounts, detail)
			}
			cd.VolumeMounts = mounts
		}
		if ctr.IsResourcePrincipalDisabled {
			cd.IsResourcePrincipalDisabled = common.Bool(true)
		}
		if resourceConfig := buildContainerResourceConfig(ctr.ResourceConfig); resourceConfig != nil {
			cd.ResourceConfig = resourceConfig
		}
		healthChecks, err := buildHealthChecks(ctr.HealthChecks)
		if err != nil {
			return nil, err
		}
		if len(healthChecks) > 0 {
			cd.HealthChecks = healthChecks
		}
		securityContext, err := buildSecurityContext(ctr.SecurityContext)
		if err != nil {
			return nil, err
		}
		if securityContext != nil {
			cd.SecurityContext = securityContext
		}
		if len(ctr.FreeformTags) > 0 {
			cd.FreeformTags = ctr.FreeformTags
		}
		if ctr.DefinedTags != nil {
			cd.DefinedTags = *util.ConvertToOciDefinedTags(&ctr.DefinedTags)
		}
		containers = append(containers, cd)
	}
	return containers, nil
}

func buildContainerResourceConfig(spec containerinstancesv1beta1.ContainerInstanceContainerResourceConfig) *containerinstancessdk.CreateContainerResourceConfigDetails {
	if spec.VcpusLimit == 0 && spec.MemoryLimitInGBs == 0 {
		return nil
	}

	config := &containerinstancessdk.CreateContainerResourceConfigDetails{}
	if spec.VcpusLimit != 0 {
		config.VcpusLimit = common.Float32(spec.VcpusLimit)
	}
	if spec.MemoryLimitInGBs != 0 {
		config.MemoryLimitInGBs = common.Float32(spec.MemoryLimitInGBs)
	}
	return config
}

func buildHealthChecks(specChecks []containerinstancesv1beta1.ContainerInstanceContainerHealthCheck) ([]containerinstancessdk.CreateContainerHealthCheckDetails, error) {
	healthChecks := make([]containerinstancessdk.CreateContainerHealthCheckDetails, 0, len(specChecks))
	for _, check := range specChecks {
		checkType := strings.ToUpper(strings.TrimSpace(check.HealthCheckType))
		if checkType == "" {
			switch {
			case len(check.Command) > 0:
				checkType = "COMMAND"
			case check.Path != "" || len(check.Headers) > 0:
				checkType = "HTTP"
			case check.Port != 0:
				checkType = "TCP"
			default:
				return nil, fmt.Errorf("container healthCheckType is required")
			}
		}

		base := healthCheckBase(check)
		switch checkType {
		case "COMMAND":
			healthChecks = append(healthChecks, containerinstancessdk.CreateContainerCommandHealthCheckDetails{
				Command:               append([]string(nil), check.Command...),
				Name:                  base.Name,
				InitialDelayInSeconds: base.InitialDelayInSeconds,
				IntervalInSeconds:     base.IntervalInSeconds,
				FailureThreshold:      base.FailureThreshold,
				SuccessThreshold:      base.SuccessThreshold,
				TimeoutInSeconds:      base.TimeoutInSeconds,
				FailureAction:         base.FailureAction,
			})
		case "HTTP":
			headers := make([]containerinstancessdk.HealthCheckHttpHeader, 0, len(check.Headers))
			for _, header := range check.Headers {
				headers = append(headers, containerinstancessdk.HealthCheckHttpHeader{
					Name:  common.String(header.Name),
					Value: common.String(header.Value),
				})
			}
			healthChecks = append(healthChecks, containerinstancessdk.CreateContainerHttpHealthCheckDetails{
				Path:                  common.String(check.Path),
				Port:                  common.Int(check.Port),
				Name:                  base.Name,
				InitialDelayInSeconds: base.InitialDelayInSeconds,
				IntervalInSeconds:     base.IntervalInSeconds,
				FailureThreshold:      base.FailureThreshold,
				SuccessThreshold:      base.SuccessThreshold,
				TimeoutInSeconds:      base.TimeoutInSeconds,
				Headers:               headers,
				FailureAction:         base.FailureAction,
			})
		case "TCP":
			healthChecks = append(healthChecks, containerinstancessdk.CreateContainerTcpHealthCheckDetails{
				Port:                  common.Int(check.Port),
				Name:                  base.Name,
				InitialDelayInSeconds: base.InitialDelayInSeconds,
				IntervalInSeconds:     base.IntervalInSeconds,
				FailureThreshold:      base.FailureThreshold,
				SuccessThreshold:      base.SuccessThreshold,
				TimeoutInSeconds:      base.TimeoutInSeconds,
				FailureAction:         base.FailureAction,
			})
		default:
			return nil, fmt.Errorf("unsupported container healthCheckType %q", checkType)
		}
	}
	return healthChecks, nil
}

type containerHealthCheckBase struct {
	Name                  *string
	InitialDelayInSeconds *int
	IntervalInSeconds     *int
	FailureThreshold      *int
	SuccessThreshold      *int
	TimeoutInSeconds      *int
	FailureAction         containerinstancessdk.ContainerHealthCheckFailureActionEnum
}

func healthCheckBase(check containerinstancesv1beta1.ContainerInstanceContainerHealthCheck) containerHealthCheckBase {
	base := containerHealthCheckBase{}
	if check.Name != "" {
		base.Name = common.String(check.Name)
	}
	if check.InitialDelayInSeconds != 0 {
		base.InitialDelayInSeconds = common.Int(check.InitialDelayInSeconds)
	}
	if check.IntervalInSeconds != 0 {
		base.IntervalInSeconds = common.Int(check.IntervalInSeconds)
	}
	if check.FailureThreshold != 0 {
		base.FailureThreshold = common.Int(check.FailureThreshold)
	}
	if check.SuccessThreshold != 0 {
		base.SuccessThreshold = common.Int(check.SuccessThreshold)
	}
	if check.TimeoutInSeconds != 0 {
		base.TimeoutInSeconds = common.Int(check.TimeoutInSeconds)
	}
	if check.FailureAction != "" {
		base.FailureAction = containerinstancessdk.ContainerHealthCheckFailureActionEnum(check.FailureAction)
	}
	return base
}

func buildSecurityContext(spec containerinstancesv1beta1.ContainerInstanceContainerSecurityContext) (containerinstancessdk.CreateSecurityContextDetails, error) {
	hasValues := spec.RunAsUser != 0 ||
		spec.RunAsGroup != 0 ||
		spec.IsNonRootUserCheckEnabled ||
		spec.IsRootFileSystemReadonly ||
		len(spec.Capabilities.AddCapabilities) > 0 ||
		len(spec.Capabilities.DropCapabilities) > 0
	if !hasValues && strings.TrimSpace(spec.SecurityContextType) == "" {
		return nil, nil
	}

	securityContextType := strings.ToUpper(strings.TrimSpace(spec.SecurityContextType))
	if securityContextType != "" && securityContextType != "LINUX" {
		return nil, fmt.Errorf("unsupported securityContextType %q", spec.SecurityContextType)
	}

	details := containerinstancessdk.CreateLinuxSecurityContextDetails{}
	if spec.RunAsUser != 0 {
		details.RunAsUser = common.Int(spec.RunAsUser)
	}
	if spec.RunAsGroup != 0 {
		details.RunAsGroup = common.Int(spec.RunAsGroup)
	}
	if spec.IsNonRootUserCheckEnabled {
		details.IsNonRootUserCheckEnabled = common.Bool(true)
	}
	if spec.IsRootFileSystemReadonly {
		details.IsRootFileSystemReadonly = common.Bool(true)
	}
	if capabilities := buildCapabilities(spec.Capabilities); capabilities != nil {
		details.Capabilities = capabilities
	}
	return details, nil
}

func buildCapabilities(spec containerinstancesv1beta1.ContainerInstanceContainerSecurityContextCapabilities) *containerinstancessdk.ContainerCapabilities {
	if len(spec.AddCapabilities) == 0 && len(spec.DropCapabilities) == 0 {
		return nil
	}

	capabilities := &containerinstancessdk.ContainerCapabilities{}
	if len(spec.AddCapabilities) > 0 {
		capabilities.AddCapabilities = make([]containerinstancessdk.ContainerCapabilityTypeEnum, 0, len(spec.AddCapabilities))
		for _, capability := range spec.AddCapabilities {
			capabilities.AddCapabilities = append(capabilities.AddCapabilities, containerinstancessdk.ContainerCapabilityTypeEnum(capability))
		}
	}
	if len(spec.DropCapabilities) > 0 {
		capabilities.DropCapabilities = make([]containerinstancessdk.ContainerCapabilityTypeEnum, 0, len(spec.DropCapabilities))
		for _, capability := range spec.DropCapabilities {
			capabilities.DropCapabilities = append(capabilities.DropCapabilities, containerinstancessdk.ContainerCapabilityTypeEnum(capability))
		}
	}
	return capabilities
}

func buildCreateVnics(specVnics []containerinstancesv1beta1.ContainerInstanceVnic) []containerinstancessdk.CreateContainerVnicDetails {
	vnics := make([]containerinstancessdk.CreateContainerVnicDetails, 0, len(specVnics))
	for _, vnic := range specVnics {
		detail := containerinstancessdk.CreateContainerVnicDetails{
			SubnetId: common.String(vnic.SubnetId),
		}
		if vnic.DisplayName != "" {
			detail.DisplayName = common.String(vnic.DisplayName)
		}
		if vnic.HostnameLabel != "" {
			detail.HostnameLabel = common.String(vnic.HostnameLabel)
		}
		if vnic.IsPublicIpAssigned {
			detail.IsPublicIpAssigned = common.Bool(true)
		}
		if vnic.SkipSourceDestCheck {
			detail.SkipSourceDestCheck = common.Bool(true)
		}
		if len(vnic.NsgIds) > 0 {
			detail.NsgIds = append([]string(nil), vnic.NsgIds...)
		}
		if vnic.PrivateIp != "" {
			detail.PrivateIp = common.String(vnic.PrivateIp)
		}
		if len(vnic.FreeformTags) > 0 {
			detail.FreeformTags = vnic.FreeformTags
		}
		if vnic.DefinedTags != nil {
			detail.DefinedTags = *util.ConvertToOciDefinedTags(&vnic.DefinedTags)
		}
		vnics = append(vnics, detail)
	}
	return vnics
}

func buildCreateVolumes(specVolumes []containerinstancesv1beta1.ContainerInstanceVolume) ([]containerinstancessdk.CreateContainerVolumeDetails, error) {
	volumes := make([]containerinstancessdk.CreateContainerVolumeDetails, 0, len(specVolumes))
	for _, volume := range specVolumes {
		volumeType := strings.ToUpper(strings.TrimSpace(volume.VolumeType))
		if volumeType == "" {
			if len(volume.Configs) > 0 {
				volumeType = "CONFIGFILE"
			} else {
				volumeType = "EMPTYDIR"
			}
		}

		switch volumeType {
		case "CONFIGFILE":
			configs := make([]containerinstancessdk.ContainerConfigFile, 0, len(volume.Configs))
			for _, config := range volume.Configs {
				data, err := base64.StdEncoding.DecodeString(config.Data)
				if err != nil {
					return nil, fmt.Errorf("decode volume config %q: %w", config.FileName, err)
				}
				entry := containerinstancessdk.ContainerConfigFile{
					FileName: common.String(config.FileName),
					Data:     data,
				}
				if config.Path != "" {
					entry.Path = common.String(config.Path)
				}
				configs = append(configs, entry)
			}
			volumes = append(volumes, containerinstancessdk.CreateContainerConfigFileVolumeDetails{
				Name:    common.String(volume.Name),
				Configs: configs,
			})
		case "EMPTYDIR":
			emptyDir := containerinstancessdk.CreateContainerEmptyDirVolumeDetails{
				Name: common.String(volume.Name),
			}
			if volume.BackingStore != "" {
				emptyDir.BackingStore = containerinstancessdk.ContainerEmptyDirVolumeBackingStoreEnum(volume.BackingStore)
			}
			volumes = append(volumes, emptyDir)
		default:
			return nil, fmt.Errorf("unsupported volumeType %q", volume.VolumeType)
		}
	}
	return volumes, nil
}

func buildDNSConfig(spec containerinstancesv1beta1.ContainerInstanceDnsConfig) *containerinstancessdk.CreateContainerDnsConfigDetails {
	if len(spec.Nameservers) == 0 && len(spec.Searches) == 0 && len(spec.Options) == 0 {
		return nil
	}
	return &containerinstancessdk.CreateContainerDnsConfigDetails{
		Nameservers: append([]string(nil), spec.Nameservers...),
		Searches:    append([]string(nil), spec.Searches...),
		Options:     append([]string(nil), spec.Options...),
	}
}

func buildImagePullSecrets(specSecrets []containerinstancesv1beta1.ContainerInstanceImagePullSecret) ([]containerinstancessdk.CreateImagePullSecretDetails, error) {
	secrets := make([]containerinstancessdk.CreateImagePullSecretDetails, 0, len(specSecrets))
	for _, secret := range specSecrets {
		secretType := strings.ToUpper(strings.TrimSpace(secret.SecretType))
		switch {
		case secretType == "VAULT" || (secretType == "" && strings.TrimSpace(secret.SecretId) != ""):
			if strings.TrimSpace(secret.SecretId) == "" {
				return nil, fmt.Errorf("imagePullSecrets.secretId is required for VAULT secrets")
			}
			secrets = append(secrets, containerinstancessdk.CreateVaultImagePullSecretDetails{
				RegistryEndpoint: common.String(secret.RegistryEndpoint),
				SecretId:         common.String(secret.SecretId),
			})
		case secretType == "", secretType == "BASIC":
			if strings.TrimSpace(secret.Username) == "" || strings.TrimSpace(secret.Password) == "" {
				return nil, fmt.Errorf("imagePullSecrets.username and imagePullSecrets.password are required for BASIC secrets")
			}
			secrets = append(secrets, containerinstancessdk.CreateBasicImagePullSecretDetails{
				RegistryEndpoint: common.String(secret.RegistryEndpoint),
				Username:         common.String(secret.Username),
				Password:         common.String(secret.Password),
			})
		default:
			return nil, fmt.Errorf("unsupported imagePullSecrets.secretType %q", secret.SecretType)
		}
	}
	return secrets, nil
}

func validateImmutableContainerInstanceFields(ci *containerinstancesv1beta1.ContainerInstance, existing *containerinstancessdk.ContainerInstance) error {
	switch {
	case safeString(existing.CompartmentId) != "" && safeString(existing.CompartmentId) != ci.Spec.CompartmentId:
		return fmt.Errorf("compartmentId cannot be updated in place")
	case safeString(existing.AvailabilityDomain) != "" && safeString(existing.AvailabilityDomain) != ci.Spec.AvailabilityDomain:
		return fmt.Errorf("availabilityDomain cannot be updated in place")
	case safeString(existing.Shape) != "" && safeString(existing.Shape) != ci.Spec.Shape:
		return fmt.Errorf("shape cannot be updated in place")
	case safeString(existing.FaultDomain) != "" && ci.Spec.FaultDomain != "" && safeString(existing.FaultDomain) != ci.Spec.FaultDomain:
		return fmt.Errorf("faultDomain cannot be updated in place")
	case existing.GracefulShutdownTimeoutInSeconds != nil &&
		ci.Spec.GracefulShutdownTimeoutInSeconds != 0 &&
		*existing.GracefulShutdownTimeoutInSeconds != ci.Spec.GracefulShutdownTimeoutInSeconds:
		return fmt.Errorf("gracefulShutdownTimeoutInSeconds cannot be updated in place")
	case existing.ContainerRestartPolicy != "" &&
		ci.Spec.ContainerRestartPolicy != "" &&
		string(existing.ContainerRestartPolicy) != ci.Spec.ContainerRestartPolicy:
		return fmt.Errorf("containerRestartPolicy cannot be updated in place")
	}

	if existing.ShapeConfig != nil {
		if existing.ShapeConfig.Ocpus != nil && ci.Spec.ShapeConfig.Ocpus != 0 && *existing.ShapeConfig.Ocpus != ci.Spec.ShapeConfig.Ocpus {
			return fmt.Errorf("shapeConfig.ocpus cannot be updated in place")
		}
		if existing.ShapeConfig.MemoryInGBs != nil &&
			ci.Spec.ShapeConfig.MemoryInGBs != 0 &&
			*existing.ShapeConfig.MemoryInGBs != ci.Spec.ShapeConfig.MemoryInGBs {
			return fmt.Errorf("shapeConfig.memoryInGBs cannot be updated in place")
		}
	}

	return nil
}
