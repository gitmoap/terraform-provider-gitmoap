data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_service_account" "ci" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  name         = "ci"
  access_level = "developer"
}

resource "gitmoap_service_account_token" "ci" {
  sa_id           = gitmoap_group_service_account.ci.service_account_id
  name            = "ci"
  scopes          = ["repo:read", "package:read"]
  expires_in_days = 90
}
