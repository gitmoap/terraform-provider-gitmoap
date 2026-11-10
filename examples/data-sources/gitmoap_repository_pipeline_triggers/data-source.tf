variable "pipeline_id" {
  type = string
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_pipeline_triggers" "api" {
  pipeline_id = var.pipeline_id
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
}
