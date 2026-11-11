variable "chat_webhook_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# Every push and merge request in the group's repositories, sent to the team's chat. The secret
# is write-only: change secret_wo_version to send a new one.
resource "gitmoap_group_webhook" "chat" {
  group_id          = data.gitmoap_group_by_path.platform.group_id
  name              = "chat"
  url               = "https://chat.example.com/hooks/gitmoap"
  secret_wo         = var.chat_webhook_secret
  secret_wo_version = 1
  events            = ["push", "merge_request"]
}
