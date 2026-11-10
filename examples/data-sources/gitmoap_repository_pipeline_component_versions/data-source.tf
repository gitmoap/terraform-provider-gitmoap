variable "component_id" {
  type = string
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_pipeline_component_versions" "api" {
  component_id = var.component_id
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
}
