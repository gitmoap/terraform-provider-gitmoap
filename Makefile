.PHONY: generate build test testacc install

# Regenerate everything from openapi.json. Overrides are hand-written and kept.
generate:
	python3 scripts/generate_generator_config.py
	python3 scripts/generate_provider.py

build:
	go build ./...

test:
	go test ./...

# Creates and deletes real objects: needs GITMOAP_TOKEN, and GITMOAP_TEST_GROUP, the full path of a
# group the token is at least a Maintainer of.
testacc:
	TF_ACC=1 go test ./internal/provider -run TestAcc -v -timeout 20m

install:
	go install .
