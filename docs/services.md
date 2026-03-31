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

All checked-in services currently use generated controller, service-manager, and
registration rollout. The remaining explicit checked-in seams are limited to the
companion adapters for the historical published resources
`AutonomousDatabases`, `MySqlDbSystem`, and `Stream`, plus the explicit
`database/AutonomousDatabases` webhook path.

Use the generated examples under `config/samples/` as the starting point for CR
manifests. For rollout details, see `docs/api-generator-contract.md`.
