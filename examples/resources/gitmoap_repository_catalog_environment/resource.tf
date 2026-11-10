data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# An environment only this repository deploys to.
resource "gitmoap_repository_catalog_environment" "preview" {
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
  slug         = "preview"
  display_name = "Preview"
  tier         = "testing"
}
