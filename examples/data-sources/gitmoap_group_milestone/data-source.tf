data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_milestone" "launch" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  title    = "Public launch"
}

data "gitmoap_group_milestone" "acme" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  milestone_id = gitmoap_group_milestone.launch.milestone_id
}
