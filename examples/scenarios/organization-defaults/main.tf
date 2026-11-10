data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

locals {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

# Merge requests tidy up their branch, and authors may squash.
resource "gitmoap_group_mr_settings" "acme" {
  group_id                     = local.group_id
  merge_method                 = "merge_commit"
  squash_option                = "allow"
  delete_source_branch_default = true
}

# Nobody approves their own change, and a new push asks for a fresh review.
resource "gitmoap_group_approval_policy" "acme" {
  group_id                = local.group_id
  author_can_approve      = false
  committers_can_approve  = true
  last_pusher_can_approve = false
  reset_on_push           = true
  enforcement             = "enforced"
}

# main changes through merge requests: Developers merge, only Maintainers push directly.
resource "gitmoap_group_branch_rule" "main" {
  group_id         = local.group_id
  branch_pattern   = "main"
  allowed_to_push  = "maintainer"
  allowed_to_merge = "developer"
  allow_force_push = false
}

# Only Maintainers cut release tags.
resource "gitmoap_group_tag_rule" "releases" {
  group_id          = local.group_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}

# Large files stay out of git history.
resource "gitmoap_group_push_rules" "acme" {
  group_id         = local.group_id
  max_file_size_mb = 50
}

# Forks stay inside the organization.
resource "gitmoap_group_fork_settings" "acme" {
  group_id         = local.group_id
  forking          = "organization"
  forking_enforced = true
}

# New work items are tasks, and a commit saying "Fixes #12" closes #12.
resource "gitmoap_group_issue_settings" "acme" {
  group_id                = local.group_id
  default_issue_type      = "task"
  close_on_commit_keyword = true
}
