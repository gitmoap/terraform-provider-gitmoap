data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

variable "reviewers" {
  description = "Usernames of the people who review changes."
  type        = set(string)
  default     = ["ada", "grace"]
}

data "gitmoap_user_by_username" "reviewers" {
  for_each = var.reviewers

  username = each.value
}

resource "gitmoap_team" "reviewers" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  name        = "Reviewers"
  slug        = "reviewers"
  description = "Reviews changes before they merge"
}

resource "gitmoap_team_member" "reviewers" {
  for_each = var.reviewers

  team_id = gitmoap_team.reviewers.team_id
  user_id = data.gitmoap_user_by_username.reviewers[each.key].user_id
  role    = "maintainer"
}
