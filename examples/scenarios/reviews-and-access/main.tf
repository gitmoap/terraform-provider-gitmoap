data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group" "payments" {
  group_name      = "payments"
  parent_group_id = data.gitmoap_group_by_path.acme.group_id
  visibility      = "private"
}

# Teams belong to the organization's top-level group.
resource "gitmoap_team" "reviewers" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  name        = "Payments reviewers"
  slug        = "payments-reviewers"
  description = "Reviews changes to payments"
}

resource "gitmoap_group_access_grant" "reviewers" {
  group_id = gitmoap_group.payments.group_id
  subject  = { team = gitmoap_team.reviewers.team_id }
  role     = { builtin = "developer" }
  reason   = "Reviews changes to payments"
}

resource "gitmoap_group_approval_rule" "reviewed" {
  group_id            = gitmoap_group.payments.group_id
  name                = "Reviewed by payments"
  required            = 1
  applies_to_branches = "main"
  approvers           = [{ team = gitmoap_team.reviewers.team_id }]
}
