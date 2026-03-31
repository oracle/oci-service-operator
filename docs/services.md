# Services

OSOK's checked-in generator config currently publishes these API groups:

- `artifacts`
- `certificates`
- `certificatesmanagement`
- `containerengine`
- `core`
- `database`
- `dns`
- `events`
- `functions`
- `identity`
- `keymanagement`
- `limits`
- `loadbalancer`
- `logging`
- `monitoring`
- `mysql`
- `networkloadbalancer`
- `nosql`
- `objectstorage`
- `ons`
- `psql`
- `queue`
- `secrets`
- `streaming`
- `vault`
- `workrequests`

Every listed API group currently uses the `controller-backed` package profile
and generated group registration rollout, but generated controller and
service-manager coverage is still resource-scoped. The published kinds that
intentionally keep `controller.strategy: none` and
`serviceManager.strategy: none` in
`internal/generator/config/services.yaml` are `keymanagement/Key`,
`keymanagement/KeyVersion`, `keymanagement/ReplicationStatus`,
`keymanagement/WrappingKey`, `streaming/Cursor`, `streaming/Group`,
`streaming/GroupCursor`, and `streaming/Message`.

The remaining explicit checked-in seams are limited to the companion adapters
for the historical published resources `AutonomousDatabases`, `MySqlDbSystem`,
and `Stream`, plus the explicit `database/AutonomousDatabases` webhook path.

Use the generated examples under `config/samples/` as the starting point for CR
manifests. For rollout details, see `docs/api-generator-contract.md`.
