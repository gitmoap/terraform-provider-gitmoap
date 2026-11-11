data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_service_account" "ci" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  name         = "ci"
  access_level = "developer"
}

# The token is kept in the state, sensitive, for what has to use it after the run, like a pipeline.
# For a token used only during a run, use the ephemeral gitmoap_service_account_token, which never is.
resource "gitmoap_service_account_token" "ci" {
  sa_id           = gitmoap_group_service_account.ci.service_account_id
  name            = "ci"
  scopes          = ["repo:read", "package:read"]
  expires_in_days = 90
}
