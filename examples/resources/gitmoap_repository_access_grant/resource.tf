data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_role" "reviewer" {
  group_id  = data.gitmoap_group_by_path.acme.group_id
  slug      = "reviewer"
  name      = "Reviewer"
  base_role = "reporter"
}

data "gitmoap_user_by_username" "grace" {
  username = "grace"
}

# Grace reviews the api repository, with a custom role, until the migration is done.
resource "gitmoap_repository_access_grant" "grace" {
  repo_id    = data.gitmoap_repository_by_path.api.repository_id
  subject    = { principal = data.gitmoap_user_by_username.grace.user_id }
  role       = { custom = gitmoap_role.reviewer.role_id }
  expires_at = "2027-03-31T00:00:00Z"
  reason     = "Reviewing the v2 migration"
}
