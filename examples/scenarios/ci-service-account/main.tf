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
