data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_team" "reviewers" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  name     = "Reviewers"
  slug     = "reviewers"
}

# Two approvals before anything merges into main: from the reviewers team, or from anyone who's at
# least a Maintainer of the group.
resource "gitmoap_group_approval_rule" "reviewed" {
  group_id            = data.gitmoap_group_by_path.platform.group_id
  name                = "Reviewed"
  required            = 2
  applies_to_branches = "main"

  approvers = [
    { team = gitmoap_team.reviewers.team_id },
    { role = { min = "maintainer", at = { group = data.gitmoap_group_by_path.platform.group_id } } },
  ]
}
