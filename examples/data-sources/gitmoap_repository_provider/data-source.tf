data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_provider" "api" {
  provider_id = "acme-deploy"
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
}
