data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_wiki" "platform" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  enabled  = true
  source   = "dedicated"
}

resource "gitmoap_group_wiki_page" "home" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  path     = "home"
  title    = "Platform"
  content  = <<-EOT
    # Platform
    Start here: how we build, ship and run the platform.
  EOT

  depends_on = [gitmoap_group_wiki.platform]
}
