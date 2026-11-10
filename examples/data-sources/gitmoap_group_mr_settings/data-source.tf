data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_mr_settings" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
}
