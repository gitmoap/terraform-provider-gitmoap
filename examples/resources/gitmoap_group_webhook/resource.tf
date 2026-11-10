variable "webhook_secret" {
  type      = string
  sensitive = true
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# Every push and merge request in the group's repositories, sent to the team's chat.
resource "gitmoap_group_webhook" "chat" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  name     = "chat"
  url      = "https://chat.example.com/hooks/gitmoap"
  secret   = var.webhook_secret
  events   = ["push", "merge_request"]
}
