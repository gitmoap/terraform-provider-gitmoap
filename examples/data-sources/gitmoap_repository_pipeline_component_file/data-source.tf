data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# The go-build component as its files hold it on the default branch, with what they compute and
# every diagnostic.
data "gitmoap_repository_pipeline_component_file" "go_build" {
  repo_id        = data.gitmoap_repository_by_path.api.repository_id
  component_name = "go-build"
}
