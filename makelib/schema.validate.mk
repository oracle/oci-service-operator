# Schema validation targets used by Maklib-enabled workflows.

.PHONY: schema.validate
schema.validate: ## Validate OSOK provider coverage against the OCI Go SDK.
	@echo "Running osok-schema-validator unit suite"
	@go test ./pkg/validator/...
