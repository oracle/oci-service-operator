/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

// Package detaileddescriptionruntime contains the common path identity for the
// application and DIS application detailed-description singleton resources.
package detaileddescriptionruntime

import (
	"fmt"
	"strings"
)

type Identity struct {
	WorkspaceID    string
	ApplicationKey string
	Collection     string
}

func Resolve(workspaceID string, applicationKey string, collection string) (Identity, error) {
	identity := Identity{WorkspaceID: strings.TrimSpace(workspaceID), ApplicationKey: strings.TrimSpace(applicationKey), Collection: strings.TrimSpace(collection)}
	if identity.WorkspaceID == "" {
		return Identity{}, fmt.Errorf("resolve detailed-description identity: workspaceId is empty")
	}
	if identity.ApplicationKey == "" {
		return Identity{}, fmt.Errorf("resolve detailed-description identity: applicationKey is empty")
	}
	return identity, nil
}

func SyntheticID(identity Identity) string {
	return identity.WorkspaceID + "/" + identity.Collection + "/" + identity.ApplicationKey + "/detailedDescription"
}
