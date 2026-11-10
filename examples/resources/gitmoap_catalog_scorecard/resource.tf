data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_catalog_scorecard" "production_ready" {
  group_id    = data.gitmoap_group_by_path.platform.group_id
  slug        = "production-ready"
  name        = "Production ready"
  description = "What a service needs before it takes production traffic"

  applies_to = {
    kinds      = ["service", "api"]
    lifecycles = ["production"]
  }
}
