data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_role" "reviewer" {
  group_id  = data.gitmoap_group_by_path.acme.group_id
  slug      = "reviewer"
  name      = "Reviewer"
  base_role = "reporter"
}

data "gitmoap_role" "example" {
  role_id = gitmoap_role.reviewer.role_id
}
