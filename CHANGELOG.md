## 0.1.0 (November 11, 2026)

FEATURES:

* New resources: `gitmoap_group_approval_rule` and `gitmoap_repository_approval_rule`, and
  `gitmoap_group_access_grant` and `gitmoap_repository_access_grant`, which give a built-in or custom
  role to a person, a service account or a team.
* New ephemeral resource: `gitmoap_service_account_token`, a token minted for one Terraform run and
  revoked after it, never stored.
* Write-only arguments: every secret argument has a `<name>_wo` twin, kept out of the plan and the
  state, and a `<name>_wo_version` to send it again: webhook secrets, pipeline and environment
  variable values, provider connection credentials, and an SSO connection's client secret.
* Fields the API gives several forms can be set: approvers, on approval rules and environment
  protection; issue automations; an OIDC federation target; a grant's subject and role; an SSO
  connection's default role; scorecard check rules; issue blueprint facet bindings.
* Free-form fields are JSON strings, set with `jsonencode()`: provider connection credentials and
  config overrides, catalog annotations, scorecard check config, issue status transitions.
* Provider connections can be created, with static credentials or OIDC federation.
* Fields only an update takes can be set: a milestone's `closed`, a repository's
  `pipelines_enabled` and `catalog_enabled`, an environment's or blueprint's `archived`, a provider
  connection's `environment_bound`.
* `gitmoap_catalog_service` has `lifecycle_stage`, and `gitmoap_provider_listing` `provider_info`:
  Terraform reserves the API's names for them.

BUG FIXES:

* A computed part of an attribute only creating can set, like a grant's `subject`, no longer forces
  replacement when something else changes.

## 0.0.2 (November 10, 2026)

DOCUMENTATION:

* An example for every data source, and every resource but the provider connections, whose
  credentials the provider can't set yet. Each example is complete on its own.
* Scenarios: resources that belong together, shown on the page of each one they use. A service
  account whose token goes into a pipeline variable, a catalog service with its environments,
  organization-wide defaults, a protected repository, a team, sprints, a wiki from Markdown files,
  webhooks with generated secrets, a release, custom work item types, single sign-on, and access
  for people.

## 0.0.1 (November 9, 2026)

The first release, generated from the gitmoap API 0.99.0.

FEATURES:

* 61 resources: groups and their settings (merge requests, issues, forks, push rules, approval
  policy), branch and tag rules, repositories and their settings, labels, milestones, iterations
  and cadences, issue blueprints, teams and team members, service accounts and their tokens,
  pipeline and environment variables, webhooks, wikis and wiki pages, branches, tags, releases,
  and the catalog's services, environments and scorecards.
* 162 data sources, covering nearly everything else the API serves to read.
* Import for every resource, by the attributes that identify it.
