variable "webhook_secret" {
  type      = string
  sensitive = true
}

data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_webhook" "chat" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  url      = "https://chat.example.com/hooks/gitmoap"
  secret   = var.webhook_secret
  events   = ["merge_request"]
}

data "gitmoap_group_webhook" "acme" {
  group_id   = data.gitmoap_group_by_path.acme.group_id
  webhook_id = gitmoap_group_webhook.chat.webhook_id
}
