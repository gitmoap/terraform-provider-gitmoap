variable "repository_provider_connection_id" {
  type = string
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_provider_connection" "api" {
  id      = var.repository_provider_connection_id
  repo_id = data.gitmoap_repository_by_path.api.repository_id
}
