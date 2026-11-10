variable "repository_pipeline_component_id" {
  type = string
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_pipeline_component" "api" {
  id      = var.repository_pipeline_component_id
  repo_id = data.gitmoap_repository_by_path.api.repository_id
}
