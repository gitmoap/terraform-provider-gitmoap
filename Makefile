.PHONY: generate docs examples spec build test testacc install

# Regenerate everything from openapi.json: the provider code, then its documentation. Overrides
# and examples are hand-written and kept.
generate:
	python3 scripts/generate_generator_config.py
	python3 scripts/generate_provider.py
	$(MAKE) docs

# Completes the examples and writes the scenario templates, then renders docs/.
docs:
	python3 scripts/generate_docs.py
	go tool tfplugindocs generate --provider-name gitmoap

# Checks every example is valid on its own, against the provider built from here.
examples:
	python3 scripts/validate_examples.py

# Take the OpenAPI spec from a gitmoap checkout, by default the one next to this repository.
GITMOAP_REPO ?= ../gitmoap
spec:
	cp $(GITMOAP_REPO)/openapi.json openapi.json

build:
	go build ./...

test:
	go test ./...

# Creates and deletes real objects: needs GITMOAP_TOKEN, and GITMOAP_TEST_GROUP, the full path of a
# group the token is an Owner of. GITMOAP_TEST_USER_ID, the token's user, adds the team test.
# Parallelism stays low: the API's CDN blocks a client that sends too much at once.
testacc:
	TF_ACC=1 TF_CLI_ARGS_apply="-parallelism=3" TF_CLI_ARGS_plan="-parallelism=3" \
		go test ./internal/provider -run TestAcc -v -timeout 30m

install:
	go install .
