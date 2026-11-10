data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_catalog_service" "payments" {
  group_id     = data.gitmoap_group_by_path.platform.group_id
  slug         = "payments"
  display_name = "Payments"
  kind         = "service"
  description  = "Takes card payments"

  # The service's code: everything under services/payments in the api repository.
  repo_paths = [{
    repository_id = data.gitmoap_repository_by_path.api.repository_id
    path_glob     = "services/payments/**"
  }]

  links = [{
    link_type = "runbook"
    url       = "https://runbooks.example.com/payments"
  }]
}
