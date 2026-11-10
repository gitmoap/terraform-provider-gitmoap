data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_wiki_pages" "api" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
}
