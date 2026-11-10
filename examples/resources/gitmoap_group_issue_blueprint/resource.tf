data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_issue_blueprint" "incident" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  blueprint_id = "incident"
  name         = "Incident"
  base_type    = "incident"
  description  = "Something broke in production"
}
