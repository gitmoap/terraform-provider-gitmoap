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
