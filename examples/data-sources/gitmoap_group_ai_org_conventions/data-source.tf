data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_ai_org_conventions" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
}
