data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_issue_blueprint" "acme" {
  blueprint_id = "incident"
  group_id     = data.gitmoap_group_by_path.acme.group_id
}
