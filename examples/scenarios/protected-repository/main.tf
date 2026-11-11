data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_repository" "payments" {
  group_id          = data.gitmoap_group_by_path.acme.group_id
  repository_name   = "payments"
  description       = "Card payments"
  visibility        = "private"
  pipelines_enabled = true
}

locals {
  repo_id = gitmoap_repository.payments.repository_id
}

# main changes only through merge requests the code owners approve.
resource "gitmoap_repository_branch_rule" "main" {
  repo_id                     = local.repo_id
  branch_pattern              = "main"
  allowed_to_push             = "maintainer"
  allowed_to_merge            = "developer"
  allow_force_push            = false
  require_code_owner_approval = true
}

resource "gitmoap_repository_tag_rule" "releases" {
  repo_id           = local.repo_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}

# Neither the author nor anyone who pushed to the branch can approve it.
resource "gitmoap_repository_approval_policy" "payments" {
  repo_id                 = local.repo_id
  author_can_approve      = false
  committers_can_approve  = false
  last_pusher_can_approve = false
  reset_on_push           = true
}

# The owners of the files a merge request changes have to approve it.
resource "gitmoap_repository_approval_rule" "code_owners" {
  repo_id   = local.repo_id
  name      = "Code owners"
  required  = 1
  approvers = [{ code_owners = true }]
}

# Every merge is squashed, and only once its pipeline passed.
resource "gitmoap_repository_mr_settings" "payments" {
  repo_id                  = local.repo_id
  override_group_settings  = true
  squash_option            = "require"
  require_pipeline_success = true
}

# Every commit names the work item it's for.
resource "gitmoap_repository_push_rules" "payments" {
  repo_id                 = local.repo_id
  max_file_size_mb        = 5
  require_issue_reference = true
}
