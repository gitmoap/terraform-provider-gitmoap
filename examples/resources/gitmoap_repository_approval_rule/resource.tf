data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# The owners of the files a merge request changes have to approve it.
resource "gitmoap_repository_approval_rule" "code_owners" {
  repo_id   = data.gitmoap_repository_by_path.api.repository_id
  name      = "Code owners"
  required  = 1
  approvers = [{ code_owners = true }]
}
