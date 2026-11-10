data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_catalog_scorecard" "production_ready" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "production-ready"
}

data "gitmoap_catalog_scorecard" "example" {
  scorecard_id = gitmoap_catalog_scorecard.production_ready.scorecard_id
}
