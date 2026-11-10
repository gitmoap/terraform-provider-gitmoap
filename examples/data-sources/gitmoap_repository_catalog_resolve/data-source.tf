data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_catalog_resolve" "api" {
  path    = ["services/payments/handler.go"]
  repo_id = data.gitmoap_repository_by_path.api.repository_id
}
