variable "webhook_secret" {
  type      = string
  sensitive = true
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_webhook" "ci" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  url     = "https://ci.example.com/hooks/gitmoap"
  secret  = var.webhook_secret
  events  = ["push"]
}

data "gitmoap_repository_webhook" "api" {
  repo_id    = data.gitmoap_repository_by_path.api.repository_id
  webhook_id = gitmoap_repository_webhook.ci.webhook_id
}
