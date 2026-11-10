data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_milestone" "v2" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  title       = "v2.0"
  description = "The second version of the API"
  due_on      = "2027-09-30"
}
