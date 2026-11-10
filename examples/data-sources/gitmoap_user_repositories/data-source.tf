data "gitmoap_user_by_username" "ada" {
  username = "ada"
}

data "gitmoap_user_repositories" "example" {
  user_id = data.gitmoap_user_by_username.ada.user_id
}
