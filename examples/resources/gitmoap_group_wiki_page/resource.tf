resource "gitmoap_group_wiki" "platform" {
  group_id = gitmoap_group.platform.group_id
  enabled  = true
  source   = "dedicated"
}

resource "gitmoap_group_wiki_page" "home" {
  group_id   = gitmoap_group.platform.group_id
  path       = "home"
  title      = "Platform"
  content    = file("${path.module}/wiki/home.md")
  depends_on = [gitmoap_group_wiki.platform]
}
