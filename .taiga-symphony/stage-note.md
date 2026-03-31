Implemented the mysql `DbSystem` formal-gap follow-through that review flagged.

- Updated `generatedruntime.ServiceClient` to use formal list lookup before create when no OCI ID is tracked, skip no-op updates when no mutable drift remains, reject comparable update drift outside the formal mutable/force-new surface, and prefer formal list lookup over an empty-ID `Get` on unresolved reads.
- Cleared mysql `dbsystem` repo-authored logic gaps, rerendered the mysql formal activity/sequence/state-machine artifacts, and regenerated the mysql service client so live `Unsupported` semantics are now empty.
- Added shared generated-runtime regressions for bind-before-create, formal delete list lookup, no-op update handling, and unsupported update drift rejection, plus a mysql `DbSystem` constructor-path regression that proves the generated client no longer fails on an open formal-gap init error.

Validation:

- `make formal-verify`
- `go test ./pkg/servicemanager/generatedruntime ./pkg/servicemanager/mysql/dbsystem`
- `make test` (`unittests.percent=12.0`)
- `make lint`
