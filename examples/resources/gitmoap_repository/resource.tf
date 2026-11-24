data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.platform.group_id
  repository_name = "api"
  description     = "The public API"
  visibility      = "private"

  # Only an update can turn pipelines on: the provider creates the repository, then updates it.
  pipelines_enabled = true
}

# Kept for reference: read-only, and its pipelines don't run. Set archived to false to work on it
# again.
resource "gitmoap_repository" "legacy_api" {
  group_id        = data.gitmoap_group_by_path.platform.group_id
  repository_name = "legacy-api"
  visibility      = "private"
  archived        = true
}

# Made again for each demo under the same name: destroying it deletes it for good, so the next one
# can take its path at once, rather than in seven days.
resource "gitmoap_repository" "demo" {
  group_id         = data.gitmoap_group_by_path.platform.group_id
  repository_name  = "demo"
  visibility       = "private"
  purge_on_destroy = true
}
