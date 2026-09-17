# Service-Manager Mock Integration

Mock integration tests exercise a typed OSOK custom resource through its
production service manager and the real OCI Go SDK while replacing only the
SDK HTTP dispatcher. They are faster and broader than live OCI E2E, and deeper
than package tests that replace the SDK client itself.

## Contract and ownership

Each `*_mock_integration_test.go` owns its resource contract explicitly. It
declares:

- the typed custom resource and initial spec;
- the supported create, read, update, and delete operations;
- typed OCI request details and response states;
- the spec mutation used for an update;
- assertions after create/read and update/read;
- resource-specific behavior such as lifecycle requeues, work requests,
  generated Secrets, or special deletion rules.

The shared code under `internal/integration/ocimock` is deliberately
mechanical. It sequences declared operations, retries reconciles when the
service manager requests a requeue, routes SDK HTTP calls to the configured
mock, verifies that every declared operation occurred, invokes package-owned
assertions, and returns a stage-specific error. It does not infer fields,
operations, mutations, or assertions from formal metadata, CR reflection, or
SDK reflection.

The complete path under test is:

```text
typed CR spec
  -> production service manager
  -> real OCI SDK HTTP request
  -> package-owned typed request assertion
  -> package-owned OCI-compatible response state
  -> real OCI SDK typed response
  -> production status projection
  -> package-owned typed CR status assertion
```

## Authoring evidence

Use these sources to author and review a scenario:

1. The vendored OCI Go SDK establishes request and response types, required
   fields, enums, polymorphic discriminators, and HTTP serialization.
2. Repo-authored formal metadata establishes OSOK lifecycle, mutation,
   identity, follow-up, and deletion intent. Provider facts are pinned by
   `formal/sources.lock` to `terraform-provider-oci`.
3. The pinned Terraform provider is supporting evidence for request mapping,
   mutable versus force-new fields, waiters, and response flattening.
4. OCI API documentation and focused live E2E, when available, confirm service
   behavior that cannot be established from local contracts alone.

Formal metadata is authoring provenance, but mock integration tests do not load
it at runtime. Every request, response, mutation, and assertion required by the
scenario is checked into the package-local Go test. A later SDK or lifecycle
change therefore requires a deliberate edit to the typed test contract and a
reviewable diff.

The SDK remains authoritative for SDK wire shapes. Terraform behavior should
agree with formal imports and OSOK-owned semantics, but it does not replace the
typed SDK contract.

## Writing a scenario

Keep each scenario beside its service manager so it can use package-private
runtime seams without exporting production APIs for tests.

1. Construct the typed CR and give it stable Kubernetes identity with
   `ocimock.InitializeResource`.
2. Declare typed create/update request details and typed OCI states. For a
   polymorphic request, assert the discriminator explicitly with
   `ValidateDiscriminatedJSONRequest` and compare the remaining concrete SDK
   type.
3. Build a stateful transport with `NewExplicitCRUDResponder`. Declare only
   operations the resource actually supports and add explicit auxiliary routes
   for resource-specific calls.
4. Open the SDK session and construct the production service-manager client.
5. Run `RunLifecycle` with package-owned mutation and status callbacks.
6. Close the session so responder verification proves every declared
   operation and auxiliary route occurred.

Do not make a fixture pass by dropping a meaningful field. Resolve a mismatch
against the service manager, SDK, formal contract, provider behavior, and live
evidence. Production fixes must follow normal ownership: edit generator or
formal source of truth and regenerate generated outputs; edit handwritten
runtime only when the behavior is resource-owned.

## Running the suite

Run all mock integration tests and their source-derived coverage audit with:

```bash
make mockintegrationtest
```

The suite discovers and runs every package-local mock integration test. Its
inventory separately reports immediate, lifecycle-polled, composite-path,
asynchronous, and not-yet-classified resources so unsupported behavior is not
silently treated as ordinary synchronous CRUD.

The broader credential-free integration surface remains:

```bash
make integrationtest
```

Live E2E remains the final proof that OCI accepts the request and provisions
the intended resource.
