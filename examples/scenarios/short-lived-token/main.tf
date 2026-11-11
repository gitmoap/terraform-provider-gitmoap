data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# The reader service account, found by its name.
data "gitmoap_group_service_accounts" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

locals {
  reader_id = one([for account in data.gitmoap_group_service_accounts.acme.items : account.service_account_id if account.name == "reader"])
}

ephemeral "gitmoap_service_account_token" "reader" {
  sa_id  = local.reader_id
  name   = "terraform-run"
  scopes = ["group:read"]
}

provider "gitmoap" {
  alias = "reader"
  token = ephemeral.gitmoap_service_account_token.reader.token
}

# Read through the reader's token.
data "gitmoap_group" "acme" {
  provider = gitmoap.reader
  group_id = data.gitmoap_group_by_path.acme.group_id
}

output "group_name" {
  value = data.gitmoap_group.acme.group_name
}
