data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# The Slack channels the organization's Slack app is in, by name.
data "gitmoap_group_chat_apps_slack_conversations" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

locals {
  slack_channels = { for channel in data.gitmoap_group_chat_apps_slack_conversations.acme.items : channel.name => channel.channel_id }
}

# #api-reviews hears the api repository's merge requests and finished pipeline runs, except while
# the team freezes it: paused keeps the subscription, and the channel hears nothing.
resource "gitmoap_repository_chat_subscription" "reviews" {
  repo_id    = data.gitmoap_repository_by_path.api.repository_id
  channel_id = local.slack_channels["api-reviews"]
  events     = ["merge_requests", "pipelines"]
  paused     = false
}
