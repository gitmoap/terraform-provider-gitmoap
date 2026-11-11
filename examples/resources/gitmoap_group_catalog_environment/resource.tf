data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_catalog_environment" "production" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  slug         = "production"
  display_name = "Production"
  tier         = "production"
  url          = "https://api.example.com"

  # Deploys come from main only, once a Maintainer other than whoever started them approves.
  protection = {
    allowed_refs          = ["main"]
    requires_approval     = true
    required              = 1
    prevent_self_approval = true
    approvers = [
      { role = { min = "maintainer", at = { group = data.gitmoap_group_by_path.platform.group_id } } },
    ]
  }

  concurrency = {
    exclusive = true
  }
}
