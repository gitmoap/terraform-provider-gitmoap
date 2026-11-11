data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_service_account" "ci" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  name         = "ci"
  access_level = "developer"
}

# A token that only reads groups, for a provider configuration used during this run. Terraform
# revokes it when it's done, and never stores it.
ephemeral "gitmoap_service_account_token" "reader" {
  sa_id  = gitmoap_group_service_account.ci.service_account_id
  name   = "terraform-run"
  scopes = ["group:read"]
}

provider "gitmoap" {
  alias = "reader"
  token = ephemeral.gitmoap_service_account_token.reader.token
}

data "gitmoap_group_by_path" "acme_as_reader" {
  provider  = gitmoap.reader
  full_path = "acme"
}
