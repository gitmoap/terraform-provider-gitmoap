---
page_title: "gitmoap_group_by_path Data Source - gitmoap"
subcategory: ""
description: |-
  Get a group by its full path. Looks a group up by its path, such as acme/platform. Anyone who can see the group may read it: a public group without signing in, an internal one when signed in, and a private one only with access to it. A private group the caller cannot see is not found.
  The token needs group:read.
---

# gitmoap_group_by_path (Data Source)

Get a group by its full path. Looks a group up by its path, such as `acme/platform`. Anyone who can see the group may read it: a public group without signing in, an internal one when signed in, and a private one only with access to it. A private group the caller cannot see is not found.

The token needs `group:read`.

## Example Usage

### Basic Usage

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
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
  group_id          = data.gitmoap_group_by_path.acme.group_id
  repository_name   = "payments"
  description       = "Card payments"
  visibility        = "private"
  pipelines_enabled = true
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

# The owners of the files a merge request changes have to approve it.
resource "gitmoap_repository_approval_rule" "code_owners" {
  repo_id   = local.repo_id
  name      = "Code owners"
  required  = 1
  approvers = [{ code_owners = true }]
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

### A service account for CI, its token handed to the pipelines

A service account gets the access it needs and no more, and its token is narrowed to one
repository. Terraform hands the token to that repository's pipelines, as a secret variable and as
the credentials of a gitmoap provider connection, both write-only: nobody copies it by hand, and
neither keeps it in the state. When the token is replaced, say with
`terraform apply -replace=gitmoap_service_account_token.ci`, both are replaced with it.

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
  access_level = "developer"
}

# The token reads and writes the api repository, nothing else, for 90 days.
resource "gitmoap_service_account_token" "ci" {
  sa_id           = gitmoap_group_service_account.ci.service_account_id
  name            = "ci-api"
  scopes          = ["repo:read", "repo:update"]
  expires_in_days = 90

  resource = {
    scope_type = "repository"
    scope_id   = gitmoap_repository.api.repository_id
  }
}

# Only runs on protected branches and tags receive it.
resource "gitmoap_repository_pipeline_variable" "ci_token" {
  repo_id          = gitmoap_repository.api.repository_id
  name             = "CI_TOKEN"
  value_wo         = gitmoap_service_account_token.ci.token
  value_wo_version = 1
  secret           = true
  protected        = true

  lifecycle {
    replace_triggered_by = [gitmoap_service_account_token.ci]
  }
}

# Pipeline steps that call gitmoap through its provider use the same token.
resource "gitmoap_repository_provider_connection" "gitmoap" {
  repo_id                = gitmoap_repository.api.repository_id
  provider_id            = "gitmoap"
  name                   = "ci"
  auth_mode              = "static_credentials"
  credentials_wo         = jsonencode({ api_token = gitmoap_service_account_token.ci.token })
  credentials_wo_version = 1
  protected              = true

  lifecycle {
    replace_triggered_by = [gitmoap_service_account_token.ci]
  }
}
```

### A service in the catalog, with its environments

The payments service is one folder of the services monorepo: the catalog knows where its code is,
where it deploys, and what each environment gives it. Production only takes deploys from main,
one at a time, approved by a Maintainer other than whoever started them.

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
  group_id        = local.group_id
  slug            = "payments"
  display_name    = "Payments"
  kind            = "service"
  description     = "Takes card payments"
  lifecycle_stage = "production"

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

  # Deploys come from main, one at a time, once a Maintainer other than whoever started them
  # approves.
  protection = {
    allowed_refs          = ["main"]
    requires_approval     = true
    required              = 1
    prevent_self_approval = true
    approvers = [
      { role = { min = "maintainer", at = { group = local.group_id } } },
    ]
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

### A short-lived token for one run

An ephemeral token, minted when Terraform opens it and revoked when it's done, configures a second
gitmoap provider that only reads groups: the rest of the run can't write anything through it, and
the token is never in the plan or the state (Terraform 1.10 and later). The service account it acts
as exists already, since Terraform can't configure a provider with what the same run creates.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# The reader service account, found by its name.
data "gitmoap_group_service_accounts" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

locals {
  reader_id = one([for account in data.gitmoap_group_service_accounts.acme.items : account.service_account_id if account.name == "reader"])
}

ephemeral "gitmoap_service_account_token" "reader" {
  sa_id  = local.reader_id
  name   = "terraform-run"
  scopes = ["group:read"]
}

provider "gitmoap" {
  alias = "reader"
  token = ephemeral.gitmoap_service_account_token.reader.token
}

# Read through the reader's token.
data "gitmoap_group" "acme" {
  provider = gitmoap.reader
  group_id = data.gitmoap_group_by_path.acme.group_id
}

output "group_name" {
  value = data.gitmoap_group.acme.group_name
}
```

### A team, with its members found by username

Teams belong to an organization's top-level group. Their members are separate resources, so
adding or removing one person changes only that membership. Whoever creates a team becomes one
of its maintainers.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

variable "reviewers" {
  description = "Usernames of the people who review changes."
  type        = set(string)
  default     = ["ada", "grace"]
}

data "gitmoap_user_by_username" "reviewers" {
  for_each = var.reviewers

  username = each.value
}

resource "gitmoap_team" "reviewers" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  name        = "Reviewers"
  slug        = "reviewers"
  description = "Reviews changes before they merge"
}

resource "gitmoap_team_member" "reviewers" {
  for_each = var.reviewers

  team_id = gitmoap_team.reviewers.team_id
  user_id = data.gitmoap_user_by_username.reviewers[each.key].user_id
  role    = "maintainer"
}
```

### A wiki written in Markdown files

The wiki's pages live next to the configuration, one Markdown file each, and each page takes its
title from its file's first heading. Adding a file adds a page; deleting one deletes it.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_wiki" "acme" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  enabled      = true
  source       = "dedicated"
  default_page = "home"
}

resource "gitmoap_group_wiki_page" "pages" {
  for_each = fileset("${path.module}/wiki", "*.md")

  group_id = data.gitmoap_group_by_path.acme.group_id
  path     = trimsuffix(each.value, ".md")
  title    = trimprefix(split("\n", file("${path.module}/wiki/${each.value}"))[0], "# ")
  content  = file("${path.module}/wiki/${each.value}")

  depends_on = [gitmoap_group_wiki.acme]
}
```

### Access for people, by username

People are found by their username, and given access where they need it: to a group and
everything beneath it, or to one repository. Access to a repository adds to what its group gives.

```terraform
data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

data "gitmoap_repository_by_path" "payments" {
  full_path = "acme/platform/payments"
}

data "gitmoap_user_by_username" "people" {
  for_each = toset(["ada", "grace"])

  username = each.value
}

# Ada works across the platform group.
resource "gitmoap_group_member" "ada" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  user_id      = data.gitmoap_user_by_username.people["ada"].user_id
  access_level = "developer"
}

# Grace maintains the payments repository only.
resource "gitmoap_repository_member" "grace" {
  repo_id      = data.gitmoap_repository_by_path.payments.repository_id
  user_id      = data.gitmoap_user_by_username.people["grace"].user_id
  access_level = "maintainer"
}
```

### Custom work item types

A blueprint is a work item type of the organization's own, built on a base type. Each of its
facets becomes an acceptance criterion on every work item of that type, and a required one has
to be met before the item closes.

The settings also move work items along with their merge requests.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_issue_blueprint" "api_feature" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  blueprint_id = "api-feature"
  name         = "API feature"
  base_type    = "story"
  description  = "A change clients of the API can use"

  facets = [
    {
      key      = "docs"
      label    = "Documented"
      category = "documentation"
      hint     = "The OpenAPI spec describes the change"
    },
    {
      key      = "observability"
      label    = "Observable"
      category = "operations"
      hint     = "Dashboards and alerts cover the new endpoints"
      required = false
    },
  ]
}

# Work items follow their merge requests: in review when one opens, done when it merges, and back
# where they were when it's closed without merging.
resource "gitmoap_group_issue_settings" "acme" {
  group_id           = data.gitmoap_group_by_path.acme.group_id
  default_issue_type = "story"
  on_mr_opened       = { to = "in_review" }
  on_mr_merged       = { to = "done" }
  on_mr_closed       = { value = "restore" }
}
```

### Defaults for every repository in an organization

Settings on a group apply to every repository beneath it, so an organization sets its defaults
once. An `enforced` setting can't be relaxed further down; the others can be overridden by a
subgroup or a repository.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

locals {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

# Merge requests tidy up their branch, and authors may squash.
resource "gitmoap_group_mr_settings" "acme" {
  group_id                     = local.group_id
  merge_method                 = "merge_commit"
  squash_option                = "allow"
  delete_source_branch_default = true
}

# Nobody approves their own change, and a new push asks for a fresh review.
resource "gitmoap_group_approval_policy" "acme" {
  group_id                = local.group_id
  author_can_approve      = false
  committers_can_approve  = true
  last_pusher_can_approve = false
  reset_on_push           = true
  enforcement             = "enforced"
}

# main changes through merge requests: Developers merge, only Maintainers push directly.
resource "gitmoap_group_branch_rule" "main" {
  group_id         = local.group_id
  branch_pattern   = "main"
  allowed_to_push  = "maintainer"
  allowed_to_merge = "developer"
  allow_force_push = false
}

# Only Maintainers cut release tags.
resource "gitmoap_group_tag_rule" "releases" {
  group_id          = local.group_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}

# Large files stay out of git history.
resource "gitmoap_group_push_rules" "acme" {
  group_id         = local.group_id
  max_file_size_mb = 50
}

# Forks stay inside the organization.
resource "gitmoap_group_fork_settings" "acme" {
  group_id         = local.group_id
  forking          = "organization"
  forking_enforced = true
}

# New work items are tasks, and a commit saying "Fixes #12" closes #12.
resource "gitmoap_group_issue_settings" "acme" {
  group_id                = local.group_id
  default_issue_type      = "task"
  close_on_commit_keyword = true
}
```

### Reviews, by a team given access

A team of reviewers is given access to the payments group through a grant, and an approval rule
asks for one of their approvals before anything merges into main in any of the group's
repositories.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group" "payments" {
  group_name      = "payments"
  parent_group_id = data.gitmoap_group_by_path.acme.group_id
  visibility      = "private"
}

# Teams belong to the organization's top-level group.
resource "gitmoap_team" "reviewers" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  name        = "Payments reviewers"
  slug        = "payments-reviewers"
  description = "Reviews changes to payments"
}

resource "gitmoap_group_access_grant" "reviewers" {
  group_id = gitmoap_group.payments.group_id
  subject  = { team = gitmoap_team.reviewers.team_id }
  role     = { builtin = "developer" }
  reason   = "Reviews changes to payments"
}

resource "gitmoap_group_approval_rule" "reviewed" {
  group_id            = gitmoap_group.payments.group_id
  name                = "Reviewed by payments"
  required            = 1
  applies_to_branches = "main"
  approvers           = [{ team = gitmoap_team.reviewers.team_id }]
}
```

### Single sign-on through an identity provider

The organization signs people in through Okta over OIDC: it claims its domain, connects the
identity provider, and gives it a SCIM token to create and remove accounts. Someone from outside
the domain gets an invite. Once `enforced`, everyone from the domain has to sign in through Okta,
so check the connection works first.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

variable "oidc_client_secret" {
  description = "The client secret of the gitmoap app in Okta."
  type        = string
  sensitive   = true
  ephemeral   = true
}

locals {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

# Proven with the DNS record the domain's attributes describe.
resource "gitmoap_group_sso_domain" "acme" {
  group_id = local.group_id
  domain   = "acme.com"
}

resource "gitmoap_group_sso_connection" "okta" {
  group_id = local.group_id
  protocol = "oidc"

  oidc = {
    issuer_url               = "https://acme.okta.com"
    client_id                = "0oa1b2c3d4e5f6g7h8i9"
    client_secret_wo         = var.oidc_client_secret
    client_secret_wo_version = 1
  }

  attribute_mapping = {
    email    = "email"
    username = "preferred_username"
    groups   = "groups"
  }

  auto_provision = true
  enforced       = true

  depends_on = [gitmoap_group_sso_domain.acme]
}

# Okta creates and removes accounts through SCIM with this token.
resource "gitmoap_group_scim_token" "okta" {
  group_id = local.group_id
  name     = "Okta"
}

output "scim_token" {
  value     = gitmoap_group_scim_token.okta.secret
  sensitive = true
}

# A contractor from outside acme.com, until the contract ends.
resource "gitmoap_group_sso_invite" "contractor" {
  group_id   = local.group_id
  email      = "ada@contractor.example.com"
  reason     = "Payments migration"
  expires_at = "2027-03-31T00:00:00Z"
}
```

### Sprints, a milestone, and a hardening week

A cadence keeps two-week sprints created ahead of time. A milestone marks what has to land for a
launch, and an iteration outside the cadence covers the week before it.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_iteration_cadence" "sprints" {
  group_id              = data.gitmoap_group_by_path.acme.group_id
  title                 = "Sprint"
  first_start_date      = "2027-01-04"
  start_day             = "monday"
  duration_weeks        = 2
  iterations_in_advance = 3
}

resource "gitmoap_group_milestone" "launch" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  title       = "Public launch"
  description = "Everything that has to land before launch"
  due_on      = "2027-03-31"
}

# Already behind us. Only an update can close a milestone: the provider creates it, then closes it.
resource "gitmoap_group_milestone" "beta" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  title    = "Private beta"
  due_on   = "2026-12-15"
  closed   = true
}

resource "gitmoap_group_iteration" "hardening" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  title       = "Hardening before launch"
  description = "No new features: fix, test and tidy up."
  start_date  = "2027-03-22"
  due_date    = "2027-03-26"
}
```

### Webhooks whose secrets never reach the state

Each endpoint checks deliveries against a secret. Here the secrets come in through an ephemeral
variable, say from `TF_VAR_webhook_secrets`, and go to gitmoap write-only, so they're in neither
Terraform's plan nor its state (Terraform 1.11 and later). Terraform can't tell when a value it
doesn't keep changes: after changing a secret, change its `secret_wo_version` too.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

variable "webhook_secrets" {
  description = "The secret each endpoint checks deliveries against, by endpoint: ci and chat."
  type        = map(string)
  sensitive   = true
  ephemeral   = true
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.acme.group_id
  repository_name = "api"
  visibility      = "private"
}

# The CI server builds every push, merge request and tag of the api repository.
resource "gitmoap_repository_webhook" "ci" {
  repo_id           = gitmoap_repository.api.repository_id
  name              = "ci"
  url               = "https://ci.example.com/hooks/gitmoap"
  secret_wo         = var.webhook_secrets["ci"]
  secret_wo_version = 1
  events            = ["push", "merge_request", "tag_push"]
}

# The team's chat hears about merge requests in every repository of the group.
resource "gitmoap_group_webhook" "chat" {
  group_id          = data.gitmoap_group_by_path.acme.group_id
  name              = "chat"
  url               = "https://chat.example.com/hooks/gitmoap"
  secret_wo         = var.webhook_secrets["chat"]
  secret_wo_version = 1
  events            = ["merge_request"]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `full_path` (String) Full hierarchical path (e.g. acme/platform)

### Read-Only

- `ancestor_group_ids` (List of String)
- `avatar` (String)
- `created_at` (String)
- `created_by` (String)
- `description` (String)
- `display_name` (String)
- `group_id` (String)
- `group_name` (String)
- `parent_group_id` (String)
- `updated_at` (String)
- `verified_domains` (List of String) The domains the organization has proven it controls, each through a DNS record. Only an
organization — a top-level group — has any; omitted when it has none.
- `visibility` (String)
