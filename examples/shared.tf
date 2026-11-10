# Definitions for the objects examples refer to. When a resource or data source example refers to
# one it doesn't define, scripts/generate_docs.py adds its definition here to the top of the
# example, with whatever that one refers to in turn, so every example stands on its own. Order
# matters: a definition comes after those it refers to.

data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_user_by_username" "ada" {
  username = "ada"
}

resource "gitmoap_team" "reviewers" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  name     = "Reviewers"
  slug     = "reviewers"
}

resource "gitmoap_role" "reviewer" {
  group_id  = data.gitmoap_group_by_path.acme.group_id
  slug      = "reviewer"
  name      = "Reviewer"
  base_role = "reporter"
}

resource "gitmoap_group_service_account" "ci" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  name         = "ci"
  access_level = "developer"
}

resource "gitmoap_group_branch_rule" "main" {
  group_id       = data.gitmoap_group_by_path.platform.group_id
  branch_pattern = "main"
}

resource "gitmoap_group_tag_rule" "releases" {
  group_id    = data.gitmoap_group_by_path.platform.group_id
  tag_pattern = "v*"
}

resource "gitmoap_group_milestone" "launch" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  title    = "Public launch"
}

resource "gitmoap_group_iteration" "hardening" {
  group_id   = data.gitmoap_group_by_path.platform.group_id
  title      = "Hardening week"
  start_date = "2027-06-28"
  due_date   = "2027-07-02"
}

resource "gitmoap_group_iteration_cadence" "sprints" {
  group_id         = data.gitmoap_group_by_path.platform.group_id
  title            = "Sprint"
  first_start_date = "2027-01-04"
}

resource "gitmoap_group_webhook" "chat" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  url      = "https://chat.example.com/hooks/gitmoap"
  secret   = var.webhook_secret
  events   = ["merge_request"]
}

resource "gitmoap_group_catalog_environment" "staging" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "staging"
  tier     = "staging"
}

resource "gitmoap_group_catalog_environment" "production" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "production"
  tier     = "production"
}

resource "gitmoap_catalog_service" "api" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "api"
  external = true
}

resource "gitmoap_catalog_scorecard" "production_ready" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "production-ready"
}

resource "gitmoap_repository_branch_rule" "main" {
  repo_id        = data.gitmoap_repository_by_path.api.repository_id
  branch_pattern = "main"
}

resource "gitmoap_repository_tag_rule" "releases" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  tag_pattern = "v*"
}

resource "gitmoap_repository_milestone" "v2" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  title   = "v2.0"
}

resource "gitmoap_repository_webhook" "ci" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  url     = "https://ci.example.com/hooks/gitmoap"
  secret  = var.webhook_secret
  events  = ["push"]
}

resource "gitmoap_repository_tag" "v1" {
  repo_id  = data.gitmoap_repository_by_path.api.repository_id
  tag_name = "v1.0.0"
  target   = "main"
  message  = "Version 1.0.0"
}

resource "gitmoap_repository_release" "v1" {
  repo_id  = data.gitmoap_repository_by_path.api.repository_id
  tag_name = gitmoap_repository_tag.v1.tag_name
  name     = "1.0.0"
}

resource "gitmoap_repository_release_asset" "cli" {
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
  tag_name     = gitmoap_repository_release.v1.tag_name
  name         = "api-cli-linux-amd64.tar.gz"
  content_type = "application/gzip"
  size         = 5242880
}
