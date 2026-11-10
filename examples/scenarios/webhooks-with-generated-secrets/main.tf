data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.acme.group_id
  repository_name = "api"
  visibility      = "private"
}

resource "random_password" "webhook" {
  for_each = toset(["ci", "chat"])

  length  = 32
  special = false
}

# The CI server builds every push, merge request and tag of the api repository.
resource "gitmoap_repository_webhook" "ci" {
  repo_id = gitmoap_repository.api.repository_id
  name    = "ci"
  url     = "https://ci.example.com/hooks/gitmoap"
  secret  = random_password.webhook["ci"].result
  events  = ["push", "merge_request", "tag_push"]
}

# The team's chat hears about merge requests in every repository of the group.
resource "gitmoap_group_webhook" "chat" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  name     = "chat"
  url      = "https://chat.example.com/hooks/gitmoap"
  secret   = random_password.webhook["chat"].result
  events   = ["merge_request"]
}

# To configure on the receiving end.
output "webhook_secrets" {
  value     = { for name, secret in random_password.webhook : name => secret.result }
  sensitive = true
}
