data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group" "platform" {
  group_name      = "platform"
  parent_group_id = data.gitmoap_group_by_path.acme.group_id
  description     = "Shared services and the tools around them"
  visibility      = "internal"
}
