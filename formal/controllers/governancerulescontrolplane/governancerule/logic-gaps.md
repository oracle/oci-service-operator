---
schemaVersion: 1
surface: repo-authored-semantics
service: governancerulescontrolplane
slug: governancerule
gaps:
  - category: seed-corpus
    status: open
    stopCondition: "Close when Terraform provider resource documentation exists for GovernanceRule, or the generator accepts reviewed SDK/runtime mutability evidence without a Terraform documentation page."
---

# Logic Gaps

The vendored SDK, production service manager, and package-local explicit mock
integration test define the supported request, response, work-request, status,
and delete behavior. The formal row remains intentionally unbound from
`formalSpec` because the pinned Terraform documentation corpus has no
GovernanceRule resource page for deterministic mutability generation.
