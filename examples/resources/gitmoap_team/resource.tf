data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# Teams belong to the organization's top-level group.
resource "gitmoap_team" "reviewers" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  name        = "Reviewers"
  slug        = "reviewers"
  description = "Reviews changes to the platform"
}
