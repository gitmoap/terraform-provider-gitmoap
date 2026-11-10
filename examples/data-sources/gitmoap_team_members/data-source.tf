data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_team" "reviewers" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  name     = "Reviewers"
  slug     = "reviewers"
}

data "gitmoap_team_members" "example" {
  team_id = gitmoap_team.reviewers.team_id
}
