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

The default active generator surface is also declared in
`internal/generator/config/services.yaml` through each service's `selection`
block. The current first-wave default-active surface is:

- whole-service: `containerengine`
- whole-service: `mysql`
- whole-service: `nosql`
- whole-service: `psql`
- explicit kind: `database/AutonomousDatabase`
- explicit kind: `streaming/Stream`

All other configured services remain inactive by default until future rollout
stories promote them, but they stay available for explicit generator runs.

The only remaining checked-in legacy companion adapter is `streaming/Stream`.
`database/AutonomousDatabase` no longer keeps the old compatibility adapter,
but the published `v1beta1` kind still carries its checked-in webhook
registration while using the generated v2 spec/controller path.
`mysql/DbSystem` now runs through the generated v2 controller and
service-manager surface with secret-backed credential handling.

Use the generated examples under `config/samples/` as the starting point for CR
manifests. For rollout details, see `docs/api-generator-contract.md`. For the
current v2 service walkthroughs, see `docs/adb.md` and `docs/mysql.md`.
