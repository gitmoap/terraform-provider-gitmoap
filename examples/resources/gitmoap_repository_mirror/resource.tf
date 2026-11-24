data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

variable "vendor_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "gitmoap_repository" "vendor_sdk" {
  group_id        = data.gitmoap_group_by_path.platform.group_id
  repository_name = "vendor-sdk"
  description     = "A mirror of the vendor's SDK"
  visibility      = "internal"
}

# Follows the vendor's private repository every six hours: its branches and tags move only with
# the source, and nobody pushes to it here. Each change to the mirror makes the account Terraform
# runs as the one its syncs run as. Pipelines stay off: what a sync brings is code nobody here
# reviewed.
resource "gitmoap_repository_mirror" "vendor_sdk" {
  repo_id          = gitmoap_repository.vendor_sdk.repository_id
  source_url       = "https://github.com/vendor/sdk.git"
  token_wo         = var.vendor_token
  token_wo_version = 1
  interval_minutes = 360
  run_pipelines    = false
}
