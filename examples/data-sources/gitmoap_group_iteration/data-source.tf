data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_iteration" "hardening" {
  group_id   = data.gitmoap_group_by_path.platform.group_id
  title      = "Hardening week"
  start_date = "2027-06-28"
  due_date   = "2027-07-02"
}

data "gitmoap_group_iteration" "acme" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  iteration_id = gitmoap_group_iteration.hardening.iteration_id
}
