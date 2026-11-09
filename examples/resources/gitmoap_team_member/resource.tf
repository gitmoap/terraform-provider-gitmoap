data "gitmoap_user_by_username" "ada" {
  username = "ada"
}

resource "gitmoap_team_member" "ada" {
  team_id = gitmoap_team.reviewers.team_id
  user_id = data.gitmoap_user_by_username.ada.user_id
  role    = "maintainer"
}
