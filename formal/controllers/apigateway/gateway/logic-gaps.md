---
schemaVersion: 1
surface: repo-authored-semantics
service: apigateway
slug: gateway
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- The public `ApiGateway` kind is preserved as a stable alias of the SDK
  `Gateway` resource family.
- Create, update, and delete use the service SDK work-request contract. Status
  projects the complete Gateway read model and the shared async tracker.
- `endpointType`, `subnetId`, `ipMode`, and IP-address configuration are
  create-only. Gateway display name, certificate, response-cache settings,
  CA bundles, network security groups, and tags are mutable.
- The package-local endpoint companion writes the observed hostname to a
  same-name Secret only after ACTIVE and removes only a Secret carrying the
  current Gateway kind/name ownership metadata.
- Delete keeps the finalizer until the work request and follow-up read prove
  the Gateway is gone.
