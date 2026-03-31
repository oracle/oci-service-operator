Implemented the follow-up fix for the generated `AutonomousDatabase` v2 schema regression.

- Narrowed polymorphic field synthesis so implementation-only required fields are relaxed only when flattening spec interface families; status and other non-spec scopes keep the strict required-field behavior.
- Added generator regressions covering the `AutonomousDatabase` clone/disaster-recovery source fields and the spec-only merge behavior.
- Regenerated the checked-in database `AutonomousDatabase` API and CRD/package manifests so clone and disaster-recovery variant fields are optional again instead of being required for every resource.
- Validation passed: `make lint`
- Validation passed: `make formal-verify`
- Validation passed: `make test`
- Coverage: `unittests.percent=12.0`
