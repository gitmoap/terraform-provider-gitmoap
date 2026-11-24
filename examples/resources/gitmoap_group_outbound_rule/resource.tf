data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# The api repository's pipelines may call Stripe's API.
resource "gitmoap_group_outbound_rule" "stripe" {
  group_id       = data.gitmoap_group_by_path.acme.group_id
  kind           = "host"
  value          = "api.stripe.com"
  services       = ["pipelines"]
  repository_ids = [data.gitmoap_repository_by_path.api.repository_id]
  note           = "Payment integration tests"
}

# Every name under the company's own domain, from every service and repository, on HTTPS and on
# the internal registry's port.
resource "gitmoap_group_outbound_rule" "internal" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  kind     = "domain"
  value    = "*.acme.example"
  ports    = [443, 5000]
}
