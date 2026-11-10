data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_catalog_environment" "production" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  slug         = "production"
  display_name = "Production"
  tier         = "production"
  url          = "https://api.example.com"

  # Deploys come from main only, and wait five minutes before they start.
  protection = {
    allowed_refs       = ["main"]
    wait_timer_seconds = 300
  }

  concurrency = {
    exclusive = true
  }
}
