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

The only remaining checked-in legacy companion adapter is `streaming/Stream`.
`database/AutonomousDatabase` no longer keeps the old compatibility adapter or
manual webhook path, and `mysql/DbSystem` now runs through the generated v2
controller and service-manager surface with secret-backed credential handling.

Use the generated examples under `config/samples/` as the starting point for CR
manifests. For rollout details, see `docs/api-generator-contract.md`. For the
current v2 service walkthroughs, see `docs/adb.md` and `docs/mysql.md`.
