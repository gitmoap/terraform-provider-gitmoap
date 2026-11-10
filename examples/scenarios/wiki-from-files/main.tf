data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_wiki" "acme" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  enabled      = true
  source       = "dedicated"
  default_page = "home"
}

resource "gitmoap_group_wiki_page" "pages" {
  for_each = fileset("${path.module}/wiki", "*.md")

  group_id = data.gitmoap_group_by_path.acme.group_id
  path     = trimsuffix(each.value, ".md")
  title    = trimprefix(split("\n", file("${path.module}/wiki/${each.value}"))[0], "# ")
  content  = file("${path.module}/wiki/${each.value}")

  depends_on = [gitmoap_group_wiki.acme]
}
