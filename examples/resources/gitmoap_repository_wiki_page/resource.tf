data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_wiki" "api" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  enabled = true
  source  = "dedicated"
}

resource "gitmoap_repository_wiki_page" "runbook" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  path    = "runbook"
  title   = "Runbook"
  message = "Add the runbook"
  content = <<-EOT
    # Runbook
    ## The API is slow
    Check the database's connection pool first.
  EOT

  depends_on = [gitmoap_repository_wiki.api]
}
