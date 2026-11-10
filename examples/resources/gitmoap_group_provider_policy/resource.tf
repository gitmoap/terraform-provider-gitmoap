data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# Which marketplace providers the organization's pipelines may run.
resource "gitmoap_group_provider_policy" "acme" {
  group_id              = data.gitmoap_group_by_path.acme.group_id
  accept_verified       = true
  accept_community      = "verified_only"
  require_changelog_ack = true
}
