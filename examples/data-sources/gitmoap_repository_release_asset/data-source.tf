data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_tag" "v1" {
  repo_id  = data.gitmoap_repository_by_path.api.repository_id
  tag_name = "v1.0.0"
  target   = "main"
  message  = "Version 1.0.0"
}

resource "gitmoap_repository_release" "v1" {
  repo_id  = data.gitmoap_repository_by_path.api.repository_id
  tag_name = gitmoap_repository_tag.v1.tag_name
  name     = "1.0.0"
}

resource "gitmoap_repository_release_asset" "cli" {
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
  tag_name     = gitmoap_repository_release.v1.tag_name
  name         = "api-cli-linux-amd64.tar.gz"
  content_type = "application/gzip"
  size         = 5242880
}

data "gitmoap_repository_release_asset" "api" {
  asset_id = gitmoap_repository_release_asset.cli.asset_id
  repo_id  = data.gitmoap_repository_by_path.api.repository_id
  tag_name = "v1.0.0"
}
