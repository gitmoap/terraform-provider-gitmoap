data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_service_account" "deployer" {
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
  name         = "deployer"
  access_level = "developer"
}
