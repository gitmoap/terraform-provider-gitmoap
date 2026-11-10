data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_tag_rule" "releases" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  tag_pattern = "v*"
}

data "gitmoap_repository_tag_rule" "api" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  tag_rule_id = gitmoap_repository_tag_rule.releases.tag_rule_id
}
