# Terraform provider for gitmoap

Manage [gitmoap](https://gitmoap.com) as code: groups and the defaults they set for every
repository beneath them, repositories, labels, milestones and sprints, teams, pipeline variables,
webhooks and wikis. Data sources read nearly everything else the API serves.

```terraform
terraform {
  required_providers {
    gitmoap = {
      source  = "gitmoap/gitmoap"
      version = "~> 0.0.1"
    }
  }
}

provider "gitmoap" {} # reads the token from GITMOAP_TOKEN

data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.acme.group_id
  repository_name = "api"
  visibility      = "private"
}
```

The documentation for every resource and data source is on the
[Terraform Registry](https://registry.terraform.io/providers/gitmoap/gitmoap/latest/docs), and in
[`docs/`](docs).

## How it's built

The provider is generated from gitmoap's OpenAPI spec, `openapi.json`, so it follows the API as it
grows. `make generate` runs the whole chain:

1. `scripts/generate_generator_config.py` works out the resources and data sources from the
   spec's paths, writes `generator_config.yml`, and runs
   [tfplugingen-openapi](https://github.com/hashicorp/terraform-plugin-codegen-openapi) on it. It
   corrects what that generator gets wrong, and writes `provider_code_spec.json`, and
   `provider_api.json`: the operations behind each resource and data source.
2. `scripts/generate_provider.py` generates each schema with
   [tfplugingen-framework](https://github.com/hashicorp/terraform-plugin-codegen-framework), into
   `internal/provider/resource_*` and `datasource_*`, and the Go tables the provider runs on.
3. [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs) writes `docs/` from the
   schemas, `examples/` and `templates/`.

Before the generator runs, the copy of the spec it reads is rewritten where Terraform has no type
for the API's: a union of objects becomes one object with the fields of every form, a variant
holding an empty object becomes a bool, a string-or-object union gets a `value` field, and a
free-form object becomes a JSON string (`scripts/unions.py`). The JSON sent and received doesn't
change; where a rewrite needs it, the provider translates.

Every resource runs on one shared implementation, `internal/provider/rest_resource.go`, and every
data source on `rest_data_source.go`. It also gives every secret argument a write-only twin
(`write_only.go`), and sets what only an update takes with an update right after creating. What one
resource needs on top, because the API handles it differently, goes in an override: see
`internal/provider/overrides.go`. The ephemeral resources are written by hand.

Some corrections apply only to known cases, listed at the top of
`scripts/generate_generator_config.py`. When the spec grows a new case of the same kind, the script
says so, with the line to add, and exits with 1. The generated code and documentation are
committed, and CI fails when they're out of date.

## Development

You need Go (see `go.mod`), Python 3.10 or later with PyYAML, and Terraform. The code generators
and tfplugindocs are pinned in `go.mod` and run with `go tool`.

| Command | What it does |
| --- | --- |
| `make spec` | Copies `openapi.json` from a gitmoap checkout, `../gitmoap` unless `GITMOAP_REPO` says otherwise |
| `make generate` | Regenerates the code and the documentation from `openapi.json` |
| `make docs` | Regenerates only the documentation, after changing an example |
| `make examples` | Checks every example is valid on its own |
| `make test` | Unit tests, including a check of every schema |
| `make testacc` | Acceptance tests, against the real API |
| `make install` | Installs the provider into `$(go env GOPATH)/bin` |

To try a build in a Terraform configuration, point Terraform at it with a development override, in
`~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/gitmoap/gitmoap" = "/path/to/go/bin"
  }
  direct {}
}
```

### Examples

The documentation's examples live in `examples/`:

- `resources/<type>/resource.tf` and `data-sources/<type>/data-source.tf`: each resource's and
  data source's own example. A data source without one gets a generated example, to edit freely.
- `scenarios/<name>/`: resources that belong together, like a service account whose token goes
  into a pipeline variable. `README.md` gives the scenario's title and what it shows, `main.tf` the
  configuration. Each one appears on the page of every resource and data source it uses.

Every example stands on its own. When one refers to an object it doesn't define, `make docs` adds
the object's definition from `examples/shared.tf`, and `make examples` checks each example alone,
against the provider built from here. `TestAccScenarios` applies every scenario for real; a new
scenario needs an entry in its `scenarioTests`.

### Acceptance tests

They create and delete real objects, named `tfacc-<random>`, and clean up even when a step fails.
The tests that change group-wide settings, and the scenarios, work in a throwaway subgroup of the
test group. The release scenario pushes a commit, and runs only where gitmoap's command-line tools
are installed.

```sh
export GITMOAP_TOKEN=gmp_...                  # an Owner of the test group
export GITMOAP_TEST_GROUP=gitmoap-showcase    # a top-level group's full path
export GITMOAP_TEST_USER_ID=...               # the token's user, for the team test
make testacc
```

The API's CDN blocks a client that sends too many requests at once for a minute or two. The
provider waits and retries when that happens, and `make testacc` keeps Terraform's parallelism low.

## Releasing

1. Add the version's notes to `CHANGELOG.md`.
2. Tag the commit and push the tag: `git tag v0.0.2 && git push origin v0.0.2`.

The release workflow builds the provider for every platform with GoReleaser, signs the checksums,
and publishes a GitHub release, which the Terraform Registry picks up.

It needs, once: the `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets in the GitHub repository, with the
key's public half added to the Terraform Registry, and the provider published on the registry from
this repository. The acceptance workflow needs the `GITMOAP_TOKEN` secret, and the
`GITMOAP_TEST_GROUP` and `GITMOAP_TEST_USER_ID` variables.

## License

[MPL-2.0](LICENSE)
