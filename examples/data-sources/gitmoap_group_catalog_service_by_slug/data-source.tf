data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_catalog_service_by_slug" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  slug     = "api"
}
