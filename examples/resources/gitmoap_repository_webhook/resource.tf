variable "ci_webhook_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# The secret is write-only: change secret_wo_version to send a new one.
resource "gitmoap_repository_webhook" "ci" {
  repo_id           = data.gitmoap_repository_by_path.api.repository_id
  name              = "ci"
  url               = "https://ci.example.com/hooks/gitmoap"
  secret_wo         = var.ci_webhook_secret
  secret_wo_version = 1
  events            = ["push", "merge_request"]
}
