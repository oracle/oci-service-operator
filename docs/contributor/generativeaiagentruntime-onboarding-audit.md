# Generative AI Agent Runtime Onboarding Audit

This audit is the `US-156` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/generativeaiagentruntime` before
`services.yaml` publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `generativeaiagentruntime` package in the
  module cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/generativeaiagentruntime` only
  because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/generativeaiagentruntime` so
  `go mod vendor` keeps the package in the branch-local inputs.

## SDK Audit

### `Session`

- Full CRUD family is present: `CreateSession`, `GetSession`, `UpdateSession`,
  and `DeleteSession`.
- There is no `ListSessions` family in this package.
- `CreateSessionRequest` requires `agentEndpointId` in the request path plus
  `CreateSessionDetails` in the body.
- `GetSessionRequest`, `UpdateSessionRequest`, and `DeleteSessionRequest`
  each require both `agentEndpointId` and `sessionId` in the request path.
- `CreateSessionDetails` and `UpdateSessionDetails` expose only
  `displayName` and `description`.
- `CreateSessionResponse`, `GetSessionResponse`, and
  `UpdateSessionResponse` return `Session`.
- `DeleteSessionResponse` returns headers only; it does not return a `Session`
  body or a work-request ID.
- `Session` exposes `id`, `timeCreated`, `displayName`, `description`,
  `welcomeMessage`, and `timeUpdated`.
- The model exposes no lifecycle enum, and the package exposes no service-local
  work-request helper family.

### Auxiliary Families

- The package also exposes `Chat`, `RetrieveMetadata`, and a broad set of
  trace, citation, tool-input, tool-output, and retrieval metadata shapes for
  live agent interactions.
- Those conversational and retrieval helper families should stay unpublished
  initially while the first `Session` rollout lands.

## Generator Implications For `US-160`

- `Session` is the planned first published kind for `US-160`.
- Recommended `formalSpec` is `session`.
- Recommended async classification is `none`.
- `Session` looks viable as a synchronous-style controller-backed rollout only
  if the contract keeps identity explicit as `agentEndpointId + sessionId`.
  There is no list surface to support opportunistic bind-by-search.
- The required risk callout is explicit here: the repo already publishes
  `bastion/Session`, so `US-160` must keep docs, formal rows, package indexes,
  and registrations explicitly service-qualified.
- `Chat` is adjacent but out of scope. `US-160` should not broaden the first
  rollout into conversational execution, retrieval metadata, or tool traces.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for
  `generativeaiagentruntime/Session` in the accessible local provider/docs
  layout.
- `US-160` should treat provider-backed imports as absent or unconfirmed and
  keep the published surface explicitly group-qualified.
