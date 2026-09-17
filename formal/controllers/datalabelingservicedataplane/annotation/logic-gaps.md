---
schemaVersion: 1
surface: repo-authored-semantics
service: datalabelingservicedataplane
slug: annotation
gaps: []
---

# Logic Gaps

The vendored SDK, generated service manager, and package-local mock integration
test define the reviewed runtime contract. The generator binds these formal
semantics without producing Terraform-derived mutability or VAP artifacts
because no corresponding provider resource exists.
