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

# The reviewers team works on everything in the platform group as Developers.
resource "gitmoap_group_access_grant" "reviewers" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  subject  = { team = gitmoap_team.reviewers.team_id }
  role     = { builtin = "developer" }
  reason   = "Reviews the platform's changes"
}
