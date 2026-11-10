data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_wiki" "platform" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  enabled      = true
  source       = "dedicated"
  default_page = "home"
}
