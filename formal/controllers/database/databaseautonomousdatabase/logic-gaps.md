---
schemaVersion: 1
surface: repo-authored-semantics
service: database
slug: databaseautonomousdatabase
gaps: []
---

# Logic Gaps

No open logic gaps remain for the promoted `database/AutonomousDatabase`
runtime semantics now that the main service-manager path is
`pkg/servicemanager/database/autonomousdatabase`.

The promoted row still binds the published API kind to generator-owned
lifecycle, delete confirmation, same-namespace secret resolution, and
list-lookup semantics without the legacy adapter or manual webhook seam.
