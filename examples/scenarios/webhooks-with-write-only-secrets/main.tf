data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

variable "webhook_secrets" {
  description = "The secret each endpoint checks deliveries against, by endpoint: ci and chat."
  type        = map(string)
  sensitive   = true
  ephemeral   = true
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.acme.group_id
  repository_name = "api"
  visibility      = "private"
}

# The CI server builds every push, merge request and tag of the api repository.
resource "gitmoap_repository_webhook" "ci" {
  repo_id           = gitmoap_repository.api.repository_id
  name              = "ci"
  url               = "https://ci.example.com/hooks/gitmoap"
  secret_wo         = var.webhook_secrets["ci"]
  secret_wo_version = 1
  events            = ["push", "merge_request", "tag_push"]
}

# The team's chat hears about merge requests in every repository of the group.
resource "gitmoap_group_webhook" "chat" {
  group_id          = data.gitmoap_group_by_path.acme.group_id
  name              = "chat"
  url               = "https://chat.example.com/hooks/gitmoap"
  secret_wo         = var.webhook_secrets["chat"]
  secret_wo_version = 1
  events            = ["merge_request"]
}
