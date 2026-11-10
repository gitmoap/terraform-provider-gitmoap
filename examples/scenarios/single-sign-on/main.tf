data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

variable "oidc_client_secret" {
  description = "The client secret of the gitmoap app in Okta."
  type        = string
  sensitive   = true
}

locals {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

# Proven with the DNS record the domain's attributes describe.
resource "gitmoap_group_sso_domain" "acme" {
  group_id = local.group_id
  domain   = "acme.com"
}

resource "gitmoap_group_sso_connection" "okta" {
  group_id = local.group_id
  protocol = "oidc"

  oidc = {
    issuer_url    = "https://acme.okta.com"
    client_id     = "0oa1b2c3d4e5f6g7h8i9"
    client_secret = var.oidc_client_secret
  }

  attribute_mapping = {
    email    = "email"
    username = "preferred_username"
    groups   = "groups"
  }

  auto_provision = true
  enforced       = true

  depends_on = [gitmoap_group_sso_domain.acme]
}

# Okta creates and removes accounts through SCIM with this token.
resource "gitmoap_group_scim_token" "okta" {
  group_id = local.group_id
  name     = "Okta"
}

output "scim_token" {
  value     = gitmoap_group_scim_token.okta.secret
  sensitive = true
}

# A contractor from outside acme.com, until the contract ends.
resource "gitmoap_group_sso_invite" "contractor" {
  group_id   = local.group_id
  email      = "ada@contractor.example.com"
  reason     = "Payments migration"
  expires_at = "2027-03-31T00:00:00Z"
}
