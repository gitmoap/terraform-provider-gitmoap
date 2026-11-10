data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

data "gitmoap_user_by_username" "ada" {
  username = "ada"
}

resource "gitmoap_group_member" "ada" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  user_id      = data.gitmoap_user_by_username.ada.user_id
  access_level = "developer"
}
