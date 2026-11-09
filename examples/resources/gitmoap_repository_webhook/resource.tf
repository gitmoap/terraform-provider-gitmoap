resource "gitmoap_repository_webhook" "ci" {
  repo_id = gitmoap_repository.api.repository_id
  name    = "ci"
  url     = "https://ci.example.com/hooks/gitmoap"
  secret  = var.webhook_secret
  events  = ["push", "merge_request"]
}
