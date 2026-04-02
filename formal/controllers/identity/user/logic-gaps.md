---
schemaVersion: 1
surface: repo-authored-semantics
service: identity
slug: user
gaps: []
---

# Logic Gaps

## Current runtime path

- `User` is no longer part of the checked-in default-active generated surface.
  When contributors regenerate the identity backlog explicitly, this preserved
  row still maps to the generated `UserServiceManager` and generated runtime
  client directly; there is no handwritten adapter seam.
- The generated runtime builds OCI requests from the CR spec, reads back the OCI resource with `GetUser`, and projects the full response into status.

## Repo-authored semantics

- OSOK status projection is part of the repo-authored contract, not a provider-fact import: the generated runtime merges the OCI response into `status`, stamps `status.status.ocid`, and updates OSOK lifecycle conditions.
- Delete confirmation is also repo-authored. The generated runtime keeps the finalizer until `GetUser` or list fallback confirms that the user is gone.
- No additional secret, webhook, endpoint, or legacy-adapter semantics are
  required for the explicit backlog `User` path, which keeps this preserved row
  as a high-fidelity generated-runtime reference even though blank/default-
  active runs no longer check it in.
