data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# The wiki's pages come from the docs folder of the repository itself.
resource "gitmoap_repository_wiki" "api" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  enabled     = true
  source      = "in_repo"
  source_path = "docs"
}
