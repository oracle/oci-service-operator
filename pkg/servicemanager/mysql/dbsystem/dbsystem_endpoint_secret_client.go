/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const dbSystemEndpointSecretOwnerUIDLabel = "mysql.oracle.com/dbsystem-uid"

func init() {
	registerDbSystemRuntimeHooksMutator(func(manager *DbSystemServiceManager, hooks *DbSystemRuntimeHooks) {
		appendDbSystemEndpointSecretRuntimeWrapper(manager, hooks)
	})
}

type dbSystemEndpointSecretRecordReader interface {
	GetSecretRecord(context.Context, string, string) (credhelper.SecretRecord, error)
}

type dbSystemEndpointSecretClient struct {
	delegate             DbSystemServiceClient
	credentialClient     credhelper.CredentialClient
	secretRecordReader   dbSystemEndpointSecretRecordReader
	guardedSecretMutator credhelper.GuardedSecretMutator
}

var _ DbSystemServiceClient = dbSystemEndpointSecretClient{}

func appendDbSystemEndpointSecretRuntimeWrapper(manager *DbSystemServiceManager, hooks *DbSystemRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DbSystemServiceClient) DbSystemServiceClient {
		return newDbSystemEndpointSecretClient(manager, delegate)
	})
}

func newDbSystemEndpointSecretClient(manager *DbSystemServiceManager, delegate DbSystemServiceClient) DbSystemServiceClient {
	client := dbSystemEndpointSecretClient{
		delegate:         delegate,
		credentialClient: manager.CredentialClient,
	}
	if recordReader, ok := manager.CredentialClient.(dbSystemEndpointSecretRecordReader); ok {
		client.secretRecordReader = recordReader
	}
	if guardedMutator, ok := manager.CredentialClient.(credhelper.GuardedSecretMutator); ok {
		client.guardedSecretMutator = guardedMutator
	}
	return client
}

func (c dbSystemEndpointSecretClient) CreateOrUpdate(
	ctx context.Context,
	resource *mysqlv1beta1.DbSystem,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !dbSystemReadyForEndpointSecret(resource) {
		return response, err
	}

	if err := c.syncEndpointSecret(ctx, resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return response, nil
}

func (c dbSystemEndpointSecretClient) Delete(ctx context.Context, resource *mysqlv1beta1.DbSystem) (bool, error) {
	deleted, err := c.delegate.Delete(ctx, resource)
	if err != nil || !deleted {
		return deleted, err
	}

	if err := c.deleteEndpointSecret(ctx, resource); err != nil {
		return deleted, err
	}
	return deleted, nil
}

func dbSystemReadyForEndpointSecret(resource *mysqlv1beta1.DbSystem) bool {
	if resource == nil {
		return false
	}
	if strings.EqualFold(resource.Status.LifecycleState, "ACTIVE") {
		return true
	}
	if resource.Status.OsokStatus.Reason == string(shared.Active) {
		return true
	}

	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return false
	}
	return conditions[len(conditions)-1].Type == shared.Active
}

func (c dbSystemEndpointSecretClient) syncEndpointSecret(ctx context.Context, resource *mysqlv1beta1.DbSystem) error {
	if c.credentialClient == nil {
		return fmt.Errorf("mysql dbsystem endpoint secret credential client is not configured")
	}
	if c.secretRecordReader == nil {
		return fmt.Errorf("mysql dbsystem endpoint secret ownership checks require secret metadata reads")
	}
	if c.guardedSecretMutator == nil {
		return fmt.Errorf("mysql dbsystem endpoint secret ownership checks require guarded secret mutations")
	}

	ownerLabels, err := dbSystemEndpointSecretLabels(resource)
	if err != nil {
		return err
	}
	desiredData, err := dbSystemEndpointSecretData(resource)
	if err != nil {
		return err
	}

	currentRecord, err := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
	if err == nil {
		return c.syncExistingEndpointSecret(ctx, resource, currentRecord, desiredData)
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	return c.createEndpointSecret(ctx, resource, ownerLabels, desiredData)
}

func (c dbSystemEndpointSecretClient) createEndpointSecret(
	ctx context.Context,
	resource *mysqlv1beta1.DbSystem,
	ownerLabels map[string]string,
	data map[string][]byte,
) error {
	_, err := c.credentialClient.CreateSecret(ctx, resource.Name, resource.Namespace, ownerLabels, data)
	switch {
	case err == nil:
		return nil
	case apierrors.IsAlreadyExists(err):
		currentRecord, rereadErr := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
		if rereadErr != nil {
			return rereadErr
		}
		return c.syncExistingEndpointSecret(ctx, resource, currentRecord, data)
	default:
		return err
	}
}

func (c dbSystemEndpointSecretClient) syncExistingEndpointSecret(
	ctx context.Context,
	resource *mysqlv1beta1.DbSystem,
	currentRecord credhelper.SecretRecord,
	desiredData map[string][]byte,
) error {
	owned, err := dbSystemOwnsEndpointSecret(resource, currentRecord.Labels)
	if err != nil {
		return err
	}
	if owned {
		if reflect.DeepEqual(currentRecord.Data, desiredData) {
			return nil
		}
		_, err = c.guardedSecretMutator.UpdateSecretIfCurrent(ctx, resource.Name, resource.Namespace, currentRecord, nil, desiredData)
		return err
	}

	adoptionLabels, adoptable, err := dbSystemLegacyEndpointSecretAdoptionLabels(resource, currentRecord, desiredData)
	if err != nil {
		return err
	}
	if !adoptable {
		return fmt.Errorf(
			"mysql dbsystem endpoint secret %s/%s is not owned by DbSystem UID %q",
			resource.Namespace,
			resource.Name,
			resource.UID,
		)
	}

	_, err = c.guardedSecretMutator.UpdateSecretIfCurrent(ctx, resource.Name, resource.Namespace, currentRecord, adoptionLabels, desiredData)
	return err
}

func dbSystemEndpointSecretData(resource *mysqlv1beta1.DbSystem) (map[string][]byte, error) {
	if resource == nil {
		return nil, fmt.Errorf("mysql dbsystem endpoint secret requires a DbSystem resource")
	}

	privateIP := strings.TrimSpace(resource.Status.IpAddress)
	if privateIP == "" {
		return nil, fmt.Errorf("mysql dbsystem endpoint secret requires status.ipAddress")
	}

	availabilityDomain := strings.TrimSpace(resource.Status.AvailabilityDomain)
	if availabilityDomain == "" {
		availabilityDomain = strings.TrimSpace(resource.Status.CurrentPlacement.AvailabilityDomain)
	}
	if availabilityDomain == "" {
		return nil, fmt.Errorf("mysql dbsystem endpoint secret requires status.availabilityDomain")
	}

	if resource.Status.Port <= 0 {
		return nil, fmt.Errorf("mysql dbsystem endpoint secret requires status.port")
	}
	if resource.Status.PortX <= 0 {
		return nil, fmt.Errorf("mysql dbsystem endpoint secret requires status.portX")
	}

	endpointsJSON, err := json.Marshal(resource.Status.Endpoints)
	if err != nil {
		return nil, fmt.Errorf("encode mysql dbsystem endpoints: %w", err)
	}

	faultDomain := strings.TrimSpace(resource.Status.FaultDomain)
	if faultDomain == "" {
		faultDomain = strings.TrimSpace(resource.Status.CurrentPlacement.FaultDomain)
	}

	return map[string][]byte{
		"PrivateIPAddress":   []byte(privateIP),
		"InternalFQDN":       []byte(strings.TrimSpace(resource.Status.HostnameLabel)),
		"AvailabilityDomain": []byte(availabilityDomain),
		"FaultDomain":        []byte(faultDomain),
		"MySQLPort":          []byte(strconv.Itoa(resource.Status.Port)),
		"MySQLXProtocolPort": []byte(strconv.Itoa(resource.Status.PortX)),
		"Endpoints":          endpointsJSON,
	}, nil
}

func (c dbSystemEndpointSecretClient) deleteEndpointSecret(ctx context.Context, resource *mysqlv1beta1.DbSystem) error {
	if c.credentialClient == nil {
		return fmt.Errorf("mysql dbsystem endpoint secret credential client is not configured")
	}
	if c.secretRecordReader == nil {
		return nil
	}
	if c.guardedSecretMutator == nil {
		return fmt.Errorf("mysql dbsystem endpoint secret ownership checks require guarded secret mutations")
	}

	record, err := c.secretRecordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
	switch {
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		return err
	}

	owned, err := dbSystemOwnsEndpointSecret(resource, record.Labels)
	if err != nil {
		return err
	}
	if !owned {
		return nil
	}

	_, err = c.guardedSecretMutator.DeleteSecretIfCurrent(ctx, resource.Name, resource.Namespace, record)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func dbSystemEndpointSecretLabels(resource *mysqlv1beta1.DbSystem) (map[string]string, error) {
	ownerUID, err := dbSystemEndpointSecretOwnerUID(resource)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		dbSystemEndpointSecretOwnerUIDLabel: ownerUID,
	}, nil
}

func dbSystemOwnsEndpointSecret(resource *mysqlv1beta1.DbSystem, labels map[string]string) (bool, error) {
	ownerUID, err := dbSystemEndpointSecretOwnerUID(resource)
	if err != nil {
		return false, err
	}
	return labels[dbSystemEndpointSecretOwnerUIDLabel] == ownerUID, nil
}

func dbSystemLegacyEndpointSecretAdoptionLabels(
	resource *mysqlv1beta1.DbSystem,
	currentRecord credhelper.SecretRecord,
	desiredData map[string][]byte,
) (map[string]string, bool, error) {
	if strings.TrimSpace(currentRecord.Labels[dbSystemEndpointSecretOwnerUIDLabel]) != "" {
		return nil, false, nil
	}
	if !reflect.DeepEqual(currentRecord.Data, desiredData) {
		return nil, false, nil
	}

	ownerLabels, err := dbSystemEndpointSecretLabels(resource)
	if err != nil {
		return nil, false, err
	}
	return mergeDbSystemEndpointSecretLabels(currentRecord.Labels, ownerLabels), true, nil
}

func mergeDbSystemEndpointSecretLabels(existing map[string]string, updates map[string]string) map[string]string {
	merged := make(map[string]string, len(existing)+len(updates))
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range updates {
		merged[key] = value
	}
	return merged
}

func dbSystemEndpointSecretOwnerUID(resource *mysqlv1beta1.DbSystem) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("mysql dbsystem endpoint secret ownership requires a DbSystem resource")
	}
	ownerUID := strings.TrimSpace(string(resource.UID))
	if ownerUID == "" {
		return "", fmt.Errorf("mysql dbsystem endpoint secret ownership requires a DbSystem UID")
	}
	return ownerUID, nil
}
