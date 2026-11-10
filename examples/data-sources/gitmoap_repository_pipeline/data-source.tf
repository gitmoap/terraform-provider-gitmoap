variable "repository_pipeline_id" {
  type = string
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_pipeline" "api" {
  id      = var.repository_pipeline_id
  repo_id = data.gitmoap_repository_by_path.api.repository_id
}
