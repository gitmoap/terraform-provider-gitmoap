variable "oidc_client_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}

data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_sso_connection" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  protocol = "oidc"

  oidc = {
    issuer_url               = "https://acme.okta.com"
    client_id                = "0oa1b2c3d4e5f6g7h8i9"
    client_secret_wo         = var.oidc_client_secret
    client_secret_wo_version = 1
  }

  # Create an account the first time someone from a verified domain signs in.
  auto_provision = true
}
