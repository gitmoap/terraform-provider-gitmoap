---
page_title: "gitmoap_repository Resource - gitmoap"
subcategory: ""
description: |-
  Creates an empty repository in a group, with main as its default branch. It needs Maintainer on the group, and you become the repository's Owner in the same act. Leave the visibility out and it inherits the group's; set it, and it can be no wider than the group's. The path must not be taken by another repository or a group.
  The token needs repo:create, repo:read, repo:update and repo:delete.
---

# gitmoap_repository (Resource)

Creates an empty repository in a group, with `main` as its default branch. It needs Maintainer on the group, and you become the repository's Owner in the same act. Leave the visibility out and it inherits the group's; set it, and it can be no wider than the group's. The path must not be taken by another repository or a group.

The token needs `repo:create`, `repo:read`, `repo:update` and `repo:delete`.

## Example Usage

### Basic Usage

```terraform
data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.platform.group_id
  repository_name = "api"
  description     = "The public API"
  visibility      = "private"
}
```

### A repository stricter than its group

A repository can tighten what it inherits. Branch rules, tag rules, push rules and the approval
policy add to the group's; merge request settings only take over from the group's with
`override_group_settings = true`.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_repository" "payments" {
  group_id        = data.gitmoap_group_by_path.acme.group_id
  repository_name = "payments"
  description     = "Card payments"
  visibility      = "private"
}

locals {
  repo_id = gitmoap_repository.payments.repository_id
}

# main changes only through merge requests the code owners approve.
resource "gitmoap_repository_branch_rule" "main" {
  repo_id                     = local.repo_id
  branch_pattern              = "main"
  allowed_to_push             = "maintainer"
  allowed_to_merge            = "developer"
  allow_force_push            = false
  require_code_owner_approval = true
}

resource "gitmoap_repository_tag_rule" "releases" {
  repo_id           = local.repo_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}

# Neither the author nor anyone who pushed to the branch can approve it.
resource "gitmoap_repository_approval_policy" "payments" {
  repo_id                 = local.repo_id
  author_can_approve      = false
  committers_can_approve  = false
  last_pusher_can_approve = false
  reset_on_push           = true
}

# Every merge is squashed, and only once its pipeline passed.
resource "gitmoap_repository_mr_settings" "payments" {
  repo_id                  = local.repo_id
  override_group_settings  = true
  squash_option            = "require"
  require_pipeline_success = true
}

# Every commit names the work item it's for.
resource "gitmoap_repository_push_rules" "payments" {
  repo_id                 = local.repo_id
  max_file_size_mb        = 5
  require_issue_reference = true
}
```

### A service account for CI, its token stored for the pipelines

A service account gets the access it needs and no more. Its token is narrowed further, to read
one repository, and handed to that repository's pipelines as a secret variable: nobody copies it
by hand. To rotate it, run `terraform apply -replace=gitmoap_service_account_token.ci`.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.acme.group_id
  repository_name = "api"
  visibility      = "private"
}

resource "gitmoap_group_service_account" "ci" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  name         = "ci"
  access_level = "reporter"
}

# The token reads the api repository, nothing else, for 90 days.
resource "gitmoap_service_account_token" "ci" {
  sa_id           = gitmoap_group_service_account.ci.service_account_id
  name            = "ci-api"
  scopes          = ["repo:read"]
  expires_in_days = 90

  resource = {
    scope_type = "repository"
    scope_id   = gitmoap_repository.api.repository_id
  }
}

# Only runs on protected branches and tags receive it, and the value is never shown.
resource "gitmoap_repository_pipeline_variable" "ci_token" {
  repo_id   = gitmoap_repository.api.repository_id
  name      = "CI_TOKEN"
  value     = gitmoap_service_account_token.ci.token
  secret    = true
  protected = true
}
```

### A service in the catalog, with its environments

The payments service is one folder of the services monorepo: the catalog knows where its code is,
where it deploys, and what each environment gives it. Production only takes deploys from main,
one at a time.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

locals {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

# A monorepo, with a folder for each service.
resource "gitmoap_repository" "services" {
  group_id        = local.group_id
  repository_name = "services"
  visibility      = "private"
}

resource "gitmoap_catalog_service" "payments" {
  group_id     = local.group_id
  slug         = "payments"
  display_name = "Payments"
  kind         = "service"
  description  = "Takes card payments"

  repo_paths = [{
    repository_id = gitmoap_repository.services.repository_id
    path_glob     = "payments/**"
  }]

  links = [
    { link_type = "runbook", url = "https://runbooks.example.com/payments" },
    { link_type = "dashboard", url = "https://grafana.example.com/d/payments" },
  ]
}

resource "gitmoap_group_catalog_environment" "staging" {
  group_id     = local.group_id
  slug         = "staging"
  display_name = "Staging"
  tier         = "staging"
}

resource "gitmoap_group_catalog_environment" "production" {
  group_id     = local.group_id
  slug         = "production"
  display_name = "Production"
  tier         = "production"

  # Deploys come from main, one at a time.
  protection = {
    allowed_refs = ["main"]
  }

  concurrency = {
    exclusive = true
  }
}

# The card processor's URL, set for each environment.
locals {
  environments = {
    staging    = gitmoap_group_catalog_environment.staging.environment_id
    production = gitmoap_group_catalog_environment.production.environment_id
  }
  card_api_urls = {
    staging    = "https://sandbox.cards.example.com"
    production = "https://cards.example.com"
  }
}

resource "gitmoap_catalog_environment_variable" "card_api_url" {
  for_each = local.card_api_urls

  environment_id = local.environments[each.key]
  name           = "CARD_API_URL"
  value          = each.value
}

# How services in production measure up.
resource "gitmoap_catalog_scorecard" "production_ready" {
  group_id = local.group_id
  slug     = "production-ready"
  name     = "Production ready"

  applies_to = {
    kinds      = ["service"]
    lifecycles = ["production"]
  }
}
```

### Webhooks with generated secrets

Each endpoint checks deliveries against a secret, generated here with the
[random provider](https://registry.terraform.io/providers/hashicorp/random/latest/docs) rather than
written down. The secrets end up in Terraform's state, so keep the state somewhere safe.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.acme.group_id
  repository_name = "api"
  visibility      = "private"
}

resource "random_password" "webhook" {
  for_each = toset(["ci", "chat"])

  length  = 32
  special = false
}

# The CI server builds every push, merge request and tag of the api repository.
resource "gitmoap_repository_webhook" "ci" {
  repo_id = gitmoap_repository.api.repository_id
  name    = "ci"
  url     = "https://ci.example.com/hooks/gitmoap"
  secret  = random_password.webhook["ci"].result
  events  = ["push", "merge_request", "tag_push"]
}

# The team's chat hears about merge requests in every repository of the group.
resource "gitmoap_group_webhook" "chat" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  name     = "chat"
  url      = "https://chat.example.com/hooks/gitmoap"
  secret   = random_password.webhook["chat"].result
  events   = ["merge_request"]
}

# To configure on the receiving end.
output "webhook_secrets" {
  value     = { for name, secret in random_password.webhook : name => secret.result }
  sensitive = true
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `group_id` (String)
- `repository_name` (String)

### Optional

- `avatar` (String)
- `description` (String)
- `display_name` (String)
- `visibility` (String)

### Read-Only

- `ancestor_group_ids` (List of String)
- `catalog_enabled` (Boolean) Whether the service catalog reads this repository's catalog file. A new fork starts with
this off: the file names its upstream's services.
- `created_at` (String)
- `created_by` (String)
- `default_branch` (String)
- `empty` (Boolean) `true` while the repository has no branch and no tag — from creation until the first push,
or after every branch and tag has been deleted. There is then no file tree to show: a tree
listing, its commit column or a file read answers `409`, so show how to push to the
repository instead of asking for one.
- `fork` (Attributes) This repository is a fork: what it was forked from, as far as you can see, and how far its tracked branch has drifted as of the last refresh. `get_repo_fork` has the full relationship. (see [below for nested schema](#nestedatt--fork))
- `forks_count` (Number) How many forks of this repository every one of its readers can see — public forks of a
public repository, public and internal forks of an internal one. Absent on a private
repository, because no fork of it is visible to all of its readers.
- `full_path` (String)
- `pipelines_enabled` (Boolean) Whether pushes, merge requests and schedules start pipeline runs here. A new fork starts
with this off, so it runs nothing until one of its Maintainers turns it on.
- `remote_url` (String) The address `git clone` and `git remote add` take for this repository:
`gitmoap://<host>/<path>` when this deployment knows its public host, which clones on a
machine never configured for it, and `gitmoap://<path>` otherwise, which works once the CLI
is signed in. Print this rather than assembling one — it is the only clone address every
surface agrees on.
- `repository_id` (String) Repository ID
- `size_bytes` (Number) Cached on-disk git object size in bytes, refreshed after each push.
Rolls up to the namespace (root group) storage quota.
- `stars` (Number)
- `updated_at` (String)

<a id="nestedatt--fork"></a>
### Nested Schema for `fork`

Read-Only:

- `ahead` (Number) Commits its tracked branch has that the upstream branch lacks, as of the last refresh.
- `behind` (Number) Commits the upstream branch has that its tracked branch lacks, as of the last refresh.
- `kind` (String) `bound`: a fork of private code, or a private fix fork — only people who can read the repositories it is bound to can read it, and it never leaves its upstream's organization. `independent`: a fork of a public or internal repository, whose audience is its own within its visibility ceiling.
- `purpose` (String) `fork`: an ordinary fork. `private_fix`: a private fork in which a security fix to its upstream is prepared out of view and landed in one step.
- `relationship` (String) `linked`: the fork can sync from its upstream and propose to it. `detached`: its Owner ended that relationship; it no longer syncs or proposes, and a bound fork stays bound.
- `state` (String) `copying`: the upstream's branches and tags are still being copied, and the fork serves nothing yet. `ready`: the copy is complete. `failed`: the copy stopped for good; `failure` says why.
- `upstream` (Attributes) The repository a fork was created from, as far as you can see it: `visible: false` names nothing, because the upstream is a repository you cannot see. (see [below for nested schema](#nestedatt--fork--upstream))
- `visibility_ceiling` (String)

<a id="nestedatt--fork--upstream"></a>
### Nested Schema for `fork.upstream`

Read-Only:

- `deleted` (Boolean) `true` once the upstream has been deleted; the fork stays, and says so.
- `full_path` (String)
- `repository_id` (String)
- `visible` (Boolean) Whether you can see the upstream. When `false`, nothing else about it is named.

## Import

Import is supported using the following syntax:

The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```shell
# Import by its repository_id.
terraform import gitmoap_repository.example "<repository_id>"
```
