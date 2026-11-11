data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

variable "slack_bot_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

# Pipelines in the group post to Slack. The credentials take the shape the provider's auth_schema
# describes, and are sent write-only: they never reach Terraform's plan or state.
resource "gitmoap_group_provider_connection" "slack" {
  group_id               = data.gitmoap_group_by_path.platform.group_id
  provider_id            = "slack"
  name                   = "releases"
  auth_mode              = "static_credentials"
  credentials_wo         = jsonencode({ bot_token = var.slack_bot_token })
  credentials_wo_version = 1
}
