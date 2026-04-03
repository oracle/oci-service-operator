# Services

OSOK now has two source-of-truth layers that both describe "what ships", but
they answer different questions:

- `internal/generator/config/services.yaml` governs generator ownership,
  rollout posture, and the default-active generated surface.
- `packages/` plus `.github/workflows/publish-service-packages.yml` govern the
  package-local controller and bundle images that get published.

## Default-Active Generated Surface

The current default-active generated surface from
`internal/generator/config/services.yaml` is:

- whole-service: `containerengine`
- whole-service: `core`
- whole-service: `dataflow`
- whole-service: `functions`
- whole-service: `mysql`
- whole-service: `nosql`
- whole-service: `psql`
- whole-service: `queue`
- whole-service: `vault`
- focused-kind: `containerinstances/ContainerInstance`
- focused-kind: `database/AutonomousDatabase`
- focused-kind: `identity/Compartment`
- focused-kind: `objectstorage/Bucket`
- focused-kind: `opensearch/OpensearchCluster`
- focused-kind: `redis/RedisCluster`
- focused-kind: `streaming/Stream`

Local packaging and published bundles also include `core-network`, which is a
package split derived from selected `core` networking kinds:

- `Drg`
- `InternetGateway`
- `NatGateway`
- `NetworkSecurityGroup`
- `RouteTable`
- `SecurityList`
- `ServiceGateway`
- `Subnet`
- `Vcn`

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

## Published Subpackage Bundles

The GitHub workflow `.github/workflows/publish-service-packages.yml` uses
`packages/*` as the publication inventory, not `services.yaml` alone. Its
default `subpackages=all` batch currently publishes:

- `apigateway`
- `containerengine`
- `containerinstances`
- `core-network`
- `database`
- `dataflow`
- `functions`
- `identity`
- `mysql`
- `nosql`
- `objectstorage`
- `opensearch`
- `psql`
- `queue`
- `redis`
- `streaming`
- `vault`

Those published images follow the naming pattern:

```text
ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-<GROUP>-bundle:v2.0.0-alpha
```

Important publication notes:

- `core` exists under `packages/core`, but the workflow intentionally excludes
  it from the default `subpackages=all` batch.
- `apigateway` is published from `packages/apigateway` even though it is not
  part of the current default-active generator surface.
- `database`, `identity`, `objectstorage`, `opensearch`, `redis`, and
  `streaming` publish focused bundles whose runtime scope is narrower than the
  full OCI service name might suggest.

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
