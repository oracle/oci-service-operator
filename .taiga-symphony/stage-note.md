Fixed the two open mysql/generated-runtime review regressions in the shared service client.

Changes:
- tightened pre-create list binding so generated-runtime reuse only happens when lookup has an identifying criterion such as `id`, `name`, or `displayName`; broad scope-only matches like `compartmentId` alone now fall through to create
- normalized generated-runtime mutation path matching so the mysql `dataStorageSizeInGBs` status/spec field stays covered by the generated mutable path `dataStorageSizeInGb`
- added shared generated-runtime regression coverage for both behaviors

Validation:
- `GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/gomod GOPATH=$(pwd)/.cache/gopath GOSUMDB=off GOFLAGS=-mod=mod go test ./pkg/servicemanager/generatedruntime`
- `GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/gomod GOPATH=$(pwd)/.cache/gopath GOSUMDB=off GOFLAGS=-mod=mod go test ./pkg/servicemanager/mysql/dbsystem`
- `GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/gomod GOPATH=$(pwd)/.cache/gopath GOSUMDB=off GOFLAGS=-mod=mod make test`
