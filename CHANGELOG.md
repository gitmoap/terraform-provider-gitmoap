## 0.2.0 (November 24, 2026)

Generated from the gitmoap API 0.116.0.

BREAKING CHANGES:

* A list data source's filters take several values, and match any of them: `tier = "production"`
  is now `tier = ["production"]`. Each has a `<name>_not` twin that leaves values out. They are
  `gitmoap_catalog_scorecard_results`' `state`, `check_key` and `service_id`;
  `gitmoap_catalog_service_deployments`' `environment_id` and `status`; `gitmoap_group_access`' and
  `gitmoap_repository_access`' `subject_kind`; `gitmoap_group_ai_org_gates`' `state` and
  `session_id`; `gitmoap_group_ai_org_sessions`' `agent_id`, `state`, `squad` and
  `ritual_run_id`; `gitmoap_group_catalog_deployments`' `environment_id` and `tier`;
  `gitmoap_group_catalog_environments`' and `gitmoap_repository_catalog_environments`' `tier`;
  `gitmoap_group_catalog_services`' and `gitmoap_repository_catalog_services`' `kind`,
  `lifecycle_stage` and `source`; `gitmoap_group_iterations`' `state` and `cadence_id`;
  `gitmoap_group_packages`' `format`; `gitmoap_provider_capabilities`' `stage_type`;
  `gitmoap_providers`' `category` and `tier`; `gitmoap_repository_pipeline_component_versions`'
  `lifecycle_state`; and `gitmoap_templates`' `kind`.
* An issue settings' `transitions` is a map from a status category to the categories a work item
  may move to from it, `{ done = ["todo"] }`, as the API sends it. `transitions.table` never held
  the table: the API refused it, and reading left it empty. A state from 0.1 reads as it is.

NOTES:

* Destroying a `gitmoap_repository` or `gitmoap_group` no longer deletes it at once: the API now
  keeps it for seven days, restorable by an Owner, and it holds its path until then, so creating
  another under the same name fails. Set `purge_on_destroy` to delete it for good.

FEATURES:

* New resources: `gitmoap_group_outbound`, an organization's outbound policy, and
  `gitmoap_group_outbound_rule`, a destination its code may reach; `gitmoap_repository_mirror`, a
  repository that follows one on another host, with a write-only `token_wo`; and
  `gitmoap_group_chat_subscription` and `gitmoap_repository_chat_subscription`, a Slack channel
  that hears a group's or a repository's events, which `paused` pauses.
* New data sources: `gitmoap_group_outbound`, `gitmoap_group_chat_apps`,
  `gitmoap_group_chat_apps_slack_conversations`, `gitmoap_group_chat_subscriptions`,
  `gitmoap_repository_chat_subscriptions`, `gitmoap_group_imports`, `gitmoap_repository_import`,
  `gitmoap_group_imported_identities`, `gitmoap_group_imported_identities_suggestions`,
  `gitmoap_repository_mirror`, `gitmoap_repository_pipeline_component_file` and
  `gitmoap_repository_pipeline_component_files`.
* `gitmoap_repository` has `archived`, which archives and unarchives it in place, `listed`, and
  `purge_on_destroy`; and reads `archived_at`, `mirrored_from` and `last_pushed_at`.
  `gitmoap_group` has `purge_on_destroy`.
* Who did something, like a `created_by` or an `author`, has `imported`: who they are on another
  host, when an import brought what they wrote there.
* A pipeline's steps and components have `condition`, as JSON; `gitmoap_repository_pipeline_file`
  has `source_map`; a provider version has `auth_schema` and `launcher_requires`; and an AI
  session says who woke it.

BUG FIXES:

* A data source's list argument sent as a query parameter, like
  `gitmoap_repository_catalog_resolve`'s `path`, was left out of the request without a word, so the
  data source read as if it weren't set. It's sent now, one parameter for each value.
* An update no longer sends back an attribute only an update takes, or an empty list, that the
  configuration leaves out. A provider connection's update sent `environment_bound` false beside
  `environment_ids` [], which the API refuses, so every change to a connection failed — rotating its
  write-only credentials included.

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
