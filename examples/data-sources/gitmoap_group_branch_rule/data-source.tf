data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_branch_rule" "main" {
  group_id       = data.gitmoap_group_by_path.platform.group_id
  branch_pattern = "main"
}

data "gitmoap_group_branch_rule" "acme" {
  branch_rule_id = gitmoap_group_branch_rule.main.branch_rule_id
  group_id       = data.gitmoap_group_by_path.acme.group_id
}
