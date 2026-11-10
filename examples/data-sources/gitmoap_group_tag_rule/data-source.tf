data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_tag_rule" "releases" {
  group_id    = data.gitmoap_group_by_path.platform.group_id
  tag_pattern = "v*"
}

data "gitmoap_group_tag_rule" "acme" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  tag_rule_id = gitmoap_group_tag_rule.releases.tag_rule_id
}
