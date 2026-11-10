data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# A Reporter who can also approve merge requests. The access_permissions data source lists the
# permission keys.
resource "gitmoap_role" "reviewer" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  slug        = "reviewer"
  name        = "Reviewer"
  description = "Reads everything, and approves merge requests"
  base_role   = "reporter"
  added       = ["merge_requests.approve"]
}
