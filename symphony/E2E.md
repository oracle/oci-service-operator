# E2E Stage Guidance

- This is a no-op verification gate for now.
- Do not add or require end-to-end coverage in this stage unless the story explicitly targets E2E work.
- If no story-specific E2E work is in scope, return `advance` with a short note that E2E is intentionally deferred.
- If the story changes shared runtime integration behavior in a way that clearly needs E2E follow-up, call that out as deferred work instead of inventing a new E2E suite here.
