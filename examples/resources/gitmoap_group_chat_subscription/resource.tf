data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# The Slack channels the organization's Slack app is in, by name.
data "gitmoap_group_chat_apps_slack_conversations" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

locals {
  slack_channels = { for channel in data.gitmoap_group_chat_apps_slack_conversations.acme.items : channel.name => channel.channel_id }
}

# #platform-releases hears the releases and deployments of every repository in the platform group
# and its subgroups.
resource "gitmoap_group_chat_subscription" "releases" {
  group_id   = data.gitmoap_group_by_path.platform.group_id
  channel_id = local.slack_channels["platform-releases"]
  events     = ["releases", "deployments"]
}
