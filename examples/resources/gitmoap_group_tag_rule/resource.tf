data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# Only Maintainers create release tags.
resource "gitmoap_group_tag_rule" "releases" {
  group_id          = data.gitmoap_group_by_path.platform.group_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}
