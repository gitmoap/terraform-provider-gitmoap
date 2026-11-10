variable "webhook_secret" {
  type      = string
  sensitive = true
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_webhook" "ci" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  name    = "ci"
  url     = "https://ci.example.com/hooks/gitmoap"
  secret  = var.webhook_secret
  events  = ["push", "merge_request"]
}
