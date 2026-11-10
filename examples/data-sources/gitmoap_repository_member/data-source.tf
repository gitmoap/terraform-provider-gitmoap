data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_user_by_username" "ada" {
  username = "ada"
}

data "gitmoap_repository_member" "api" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  user_id = data.gitmoap_user_by_username.ada.user_id
}
