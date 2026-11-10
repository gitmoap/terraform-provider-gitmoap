data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_user_by_username" "ada" {
  username = "ada"
}

data "gitmoap_group_member" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  user_id  = data.gitmoap_user_by_username.ada.user_id
}
