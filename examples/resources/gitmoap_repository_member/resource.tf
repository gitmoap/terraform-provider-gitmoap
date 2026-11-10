data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_user_by_username" "grace" {
  username = "grace"
}

resource "gitmoap_repository_member" "grace" {
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
  user_id      = data.gitmoap_user_by_username.grace.user_id
  access_level = "maintainer"
}
