data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# Release branches change only through reviewed merge requests, with the code owners' approval.
resource "gitmoap_repository_branch_rule" "release" {
  repo_id                     = data.gitmoap_repository_by_path.api.repository_id
  branch_pattern              = "release/*"
  allowed_to_push             = "maintainer"
  allowed_to_merge            = "maintainer"
  allow_force_push            = false
  require_code_owner_approval = true
}
