data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_catalog_service" "api" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "api"
  external = true
}

data "gitmoap_catalog_service_deployments" "example" {
  service_id = gitmoap_catalog_service.api.service_id
}
