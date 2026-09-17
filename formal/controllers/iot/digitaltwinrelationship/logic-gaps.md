---
schemaVersion: 1
surface: repo-authored-semantics
service: iot
slug: digitaltwinrelationship
gaps: []
---

# Logic Gaps

The pinned Terraform provider registers DigitalTwinRelationship, but `formal-import` cannot resolve its CRUD type. The checked-in runtime lifecycle and vendored OCI SDK remain authoritative until importer support is added.
