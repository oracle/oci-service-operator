# Services

`internal/generator/config/services.yaml` still tracks the broader configured
service inventory for rollout planning and explicit backlog generation:

- `artifacts`
- `certificates`
- `certificatesmanagement`
- `containerengine`
- `core`
- `database`
- `dataflow`
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

The checked-in generated APIs, CRDs, samples, package scaffolding, and release
bundle ship the current default-active surface declared through each service's
`selection` block:

- whole-service: `containerengine`
- whole-service: `core`
- whole-service: `dataflow`
- whole-service: `mysql`
- whole-service: `nosql`
- whole-service: `psql`
- whole-service: `queue`
- whole-service: `vault`
- explicit kind: `database/AutonomousDatabase`
- explicit kind: `identity/Compartment`
- explicit kind: `opensearch/OpensearchCluster`
- explicit kind: `redis/RedisCluster`
- explicit kind: `streaming/Stream`

Refresh that shipped surface with:

```sh
go run ./cmd/generator --config internal/generator/config/services.yaml --all --overwrite
```

Use explicit service selection when you need backlog or inactive services that
are still configured in `services.yaml` but are not checked in by default:

```sh
go run ./cmd/generator --config internal/generator/config/services.yaml --service <service> --overwrite
```

Follow explicit backlog generation with `make generate`, `make manifests`, and
`make package-generate GROUP=<group>` when the targeted group's checked-in
package-local install artifacts also need refresh.

Every listed API group currently uses the `controller-backed` package profile
and generated group registration rollout, but generated controller and
service-manager coverage is still resource-scoped. The published kinds that
intentionally keep `controller.strategy: none` and
`serviceManager.strategy: none` in
`internal/generator/config/services.yaml` are `keymanagement/Key`,
`keymanagement/KeyVersion`, `keymanagement/ReplicationStatus`,
`keymanagement/WrappingKey`, `streaming/Cursor`, `streaming/Group`,
`streaming/GroupCursor`, and `streaming/Message`.

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
