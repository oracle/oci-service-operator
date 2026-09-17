---
schemaVersion: 1
surface: repo-authored-semantics
service: apigateway
slug: deployment
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- The public `ApiGatewayDeployment` kind is preserved as a stable alias of the
  SDK `Deployment` resource family.
- Create, update, and delete use the service SDK work-request contract. Status
  projects the complete Deployment read model, including the deployed API
  specification, and the shared async tracker.
- `gatewayId` and `pathPrefix` are create-only. Display name, specification,
  and tags are mutable; compartment movement remains an explicit auxiliary
  operation rather than part of the normal update body.
- The compatibility top-level `routes` field is normalized to
  `specification.routes`. Supplying both forms with different routes is
  rejected before an OCI request is sent.
- Delete keeps the finalizer until the work request and follow-up read prove
  the Deployment is gone.
