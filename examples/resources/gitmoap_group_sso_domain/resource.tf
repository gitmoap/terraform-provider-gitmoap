data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# Claim a domain, then prove it with the DNS record the domain's attributes describe.
resource "gitmoap_group_sso_domain" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  domain   = "acme.com"
}
