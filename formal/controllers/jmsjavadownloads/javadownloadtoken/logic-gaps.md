---
schemaVersion: 1
surface: repo-authored-semantics
service: jmsjavadownloads
slug: javadownloadtoken
gaps: []
---

# Logic Gaps

`jmsjavadownloads/JavaDownloadToken` is the first published resource for the
service-scoped Java download rollout and must remain distinct from the broader
`jms` fleet surface.

## Current runtime path

- `JavaDownloadToken` keeps the generated controller, service-manager shell,
  and registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/jmsjavadownloads/javadownloadtoken/` rather than the
  generated baseline alone.
- Create, update, and delete are work-request-backed. The reviewed runtime
  resumes all three operations through `GetWorkRequest`, normalizes
  `ACCEPTED`, `IN_PROGRESS`, `WAITING`, `CANCELING`, `SUCCEEDED`, `FAILED`,
  `CANCELED`, and `NEEDS_ATTENTION` into the shared async tracker, and rereads
  the token through `GetJavaDownloadToken` before settling success.
- Create returns a `JavaDownloadToken` body plus `opc-work-request-id`, while
  update and delete return work-request headers only. The runtime therefore
  captures the tracked token OCID from the create body or work-request
  resources and treats delete confirmation as required, not best-effort.
- Sensitive output handling is explicit. The token `value` authorizes
  downloads, so this rollout intentionally excludes it from generated observed
  state, published status, generated docs, and generic bind/parity matching.
  No Kubernetes Secret side effects are emitted in this story.
- Bind resolution remains bounded. Existing-before-create lookup is limited to
  exact `compartmentId` plus `displayName` matching and does not use the
  sensitive `value` filter even though the SDK exposes it on
  `ListJavaDownloadTokens`.
- Mutable drift is limited to `displayName`, `description`, `isDefault`,
  `timeExpires`, `licenseType`, `freeformTags`, and `definedTags`.
  `compartmentId` and `javaVersion` remain replacement-only drift.
- The reviewed secret-handling contract for this story is explicit: OSOK does
  not persist the one-time token value after create. Operators must preserve
  that sensitive output through an out-of-band reviewed path instead of
  expecting it to reappear in CR status or generated docs.
