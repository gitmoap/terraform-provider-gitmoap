data "gitmoap_repository_by_path" "api" {
  full_path = "acme/api"
}

locals {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
}

resource "gitmoap_repository_branch" "release" {
  repo_id     = local.repo_id
  branch_name = "release/1.0"
  from_branch = "main"
}

resource "gitmoap_repository_tag_rule" "releases" {
  repo_id           = local.repo_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}

resource "gitmoap_repository_tag" "v1_0_0" {
  repo_id  = local.repo_id
  tag_name = "v1.0.0"
  target   = gitmoap_repository_branch.release.branch_name
  message  = "Version 1.0.0"

  depends_on = [gitmoap_repository_tag_rule.releases]
}

resource "gitmoap_repository_release" "v1_0_0" {
  repo_id  = local.repo_id
  tag_name = gitmoap_repository_tag.v1_0_0.tag_name
  name     = "1.0.0"
  body     = file("${path.module}/notes/1.0.0.md")
}
