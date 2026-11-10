data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# A token for the identity provider to provision people with SCIM.
resource "gitmoap_group_scim_token" "okta" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  name     = "Okta"
}

# The secret is only returned when the token is created: give it to the identity provider.
output "scim_token" {
  value     = gitmoap_group_scim_token.okta.secret
  sensitive = true
}
