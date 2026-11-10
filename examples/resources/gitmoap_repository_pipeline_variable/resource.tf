variable "deploy_key" {
  type      = string
  sensitive = true
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_pipeline_variable" "log_level" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  name    = "LOG_LEVEL"
  value   = "info"
}

# Only runs on protected branches and tags receive it.
resource "gitmoap_repository_pipeline_variable" "deploy_key" {
  repo_id   = data.gitmoap_repository_by_path.api.repository_id
  name      = "DEPLOY_KEY"
  value     = var.deploy_key
  secret    = true
  protected = true
}
