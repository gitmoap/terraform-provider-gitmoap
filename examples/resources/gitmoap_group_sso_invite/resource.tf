data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# Let someone from outside the organization's domains sign in, until the contract ends.
resource "gitmoap_group_sso_invite" "contractor" {
  group_id   = data.gitmoap_group_by_path.acme.group_id
  email      = "ada@contractor.example.com"
  reason     = "Payments migration"
  expires_at = "2027-03-31T00:00:00Z"
}
