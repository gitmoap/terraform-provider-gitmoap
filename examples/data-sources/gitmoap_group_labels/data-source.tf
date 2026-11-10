data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

data "gitmoap_group_labels" "platform" {
  group_id = data.gitmoap_group_by_path.platform.group_id
}
