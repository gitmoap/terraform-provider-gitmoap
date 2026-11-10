data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

data "gitmoap_repository_by_path" "payments" {
  full_path = "acme/platform/payments"
}

data "gitmoap_user_by_username" "people" {
  for_each = toset(["ada", "grace"])

  username = each.value
}

# Ada works across the platform group.
resource "gitmoap_group_member" "ada" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  user_id      = data.gitmoap_user_by_username.people["ada"].user_id
  access_level = "developer"
}

# Grace maintains the payments repository only.
resource "gitmoap_repository_member" "grace" {
  repo_id      = data.gitmoap_repository_by_path.payments.repository_id
  user_id      = data.gitmoap_user_by_username.people["grace"].user_id
  access_level = "maintainer"
}
