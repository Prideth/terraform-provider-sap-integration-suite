.PHONY: build test unit-test hygiene testacc testacc-all testacc-metadata acceptance-test accplan \
	api-discovery api-metadata-diff api-metadata-refresh api-discovery-report \
	fmt vet lint tidy docs clean

BINARY := terraform-provider-sap-integration-suite

# Directory with official specifications downloaded from api.sap.com for the
# REST APIs discovery cannot fetch itself (see CONTRIBUTING.md).
SPEC_DIR ?=
SPEC_FLAG := $(if $(SPEC_DIR),-spec-dir $(SPEC_DIR))

build:
	go build -o $(BINARY) .

test: unit-test

# Unit tests, contract checks against the committed snapshots and the
# discovery classification. Needs no credentials.
unit-test:
	go test -race -cover ./...

# Acceptance tests of the capabilities enabled with
# SAP_INTEGRATION_SUITE_ACC_<CAPABILITY>=1 (see CONTRIBUTING.md).
testacc:
	TF_ACC=1 go test -v -timeout 120m ./...

acceptance-test: testacc

# Every capability whose credentials are set; never destructive tests.
testacc-all:
	TF_ACC=1 SAP_INTEGRATION_SUITE_ACC_ALL=1 go run ./cmd/accplan
	TF_ACC=1 SAP_INTEGRATION_SUITE_ACC_ALL=1 go test -v -timeout 120m ./...

# Live $metadata checks: reachability, contract, delta against the snapshots.
testacc-metadata:
	TF_ACC=1 SAP_INTEGRATION_SUITE_ACC_METADATA=1 go test -v -run '^TestAccMetadata$$' ./internal/apidiscovery/

# What an acceptance run would do with the current environment.
accplan:
	go run ./cmd/accplan

# Compare the live contracts with the snapshots; fail only on breaking
# changes to what the provider uses.
api-metadata-diff:
	go run ./cmd/apidiscovery -fail-on provider-breaking $(SPEC_FLAG)

# Fail on any difference from the snapshots, additive ones included: how new
# SAP entities are found.
api-discovery:
	go run ./cmd/apidiscovery -fail-on additive $(SPEC_FLAG)

# Write the snapshots whose contract changed and regenerate the report.
# Review and commit the result yourself.
api-metadata-refresh:
	go run ./cmd/apidiscovery -update -fail-on none $(SPEC_FLAG)
	go run ./cmd/apidiscovery -offline -report docs/api-discovery-report.md

api-discovery-report:
	go run ./cmd/apidiscovery -offline -report docs/api-discovery-report.md

# Tool attribution, author and commit message rules (CONTRIBUTING.md).
hygiene:
	go run ./cmd/repohygiene -commits origin/master..HEAD

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
	cd tools && go mod tidy

docs:
	cd tools && go build -o ../.bin/tfplugindocs github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	./.bin/tfplugindocs generate --provider-name sapintegrationsuite
	go run ./cmd/gendocs
	go run ./cmd/gendocs -readme
	go run ./cmd/apidiscovery -offline -report docs/api-discovery-report.md

clean:
	rm -f $(BINARY)
	rm -rf .bin dist
