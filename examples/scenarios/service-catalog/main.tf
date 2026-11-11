data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

locals {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

# A monorepo, with a folder for each service.
resource "gitmoap_repository" "services" {
  group_id        = local.group_id
  repository_name = "services"
  visibility      = "private"
}

resource "gitmoap_catalog_service" "payments" {
  group_id        = local.group_id
  slug            = "payments"
  display_name    = "Payments"
  kind            = "service"
  description     = "Takes card payments"
  lifecycle_stage = "production"

  repo_paths = [{
    repository_id = gitmoap_repository.services.repository_id
    path_glob     = "payments/**"
  }]

  links = [
    { link_type = "runbook", url = "https://runbooks.example.com/payments" },
    { link_type = "dashboard", url = "https://grafana.example.com/d/payments" },
  ]
}

resource "gitmoap_group_catalog_environment" "staging" {
  group_id     = local.group_id
  slug         = "staging"
  display_name = "Staging"
  tier         = "staging"
}

resource "gitmoap_group_catalog_environment" "production" {
  group_id     = local.group_id
  slug         = "production"
  display_name = "Production"
  tier         = "production"

  # Deploys come from main, one at a time, once a Maintainer other than whoever started them
  # approves.
  protection = {
    allowed_refs          = ["main"]
    requires_approval     = true
    required              = 1
    prevent_self_approval = true
    approvers = [
      { role = { min = "maintainer", at = { group = local.group_id } } },
    ]
  }

  concurrency = {
    exclusive = true
  }
}

# The card processor's URL, set for each environment.
locals {
  environments = {
    staging    = gitmoap_group_catalog_environment.staging.environment_id
    production = gitmoap_group_catalog_environment.production.environment_id
  }
  card_api_urls = {
    staging    = "https://sandbox.cards.example.com"
    production = "https://cards.example.com"
  }
}

resource "gitmoap_catalog_environment_variable" "card_api_url" {
  for_each = local.card_api_urls

  environment_id = local.environments[each.key]
  name           = "CARD_API_URL"
  value          = each.value
}

# How services in production measure up.
resource "gitmoap_catalog_scorecard" "production_ready" {
  group_id = local.group_id
  slug     = "production-ready"
  name     = "Production ready"

  applies_to = {
    kinds      = ["service"]
    lifecycles = ["production"]
  }
}
