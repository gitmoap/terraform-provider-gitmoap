data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_iteration_cadence" "sprints" {
  group_id         = data.gitmoap_group_by_path.platform.group_id
  title            = "Sprint"
  first_start_date = "2027-01-04"
}

data "gitmoap_group_iteration_cadence" "acme" {
  cadence_id = gitmoap_group_iteration_cadence.sprints.cadence_id
  group_id   = data.gitmoap_group_by_path.acme.group_id
}
