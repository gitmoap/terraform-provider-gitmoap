data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_branch_rule" "main" {
  repo_id        = data.gitmoap_repository_by_path.api.repository_id
  branch_pattern = "main"
}

data "gitmoap_repository_branch_rule" "api" {
  branch_rule_id = gitmoap_repository_branch_rule.main.branch_rule_id
  repo_id        = data.gitmoap_repository_by_path.api.repository_id
}
