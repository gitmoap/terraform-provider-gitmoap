data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_issue_blueprint" "spike" {
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
  blueprint_id = "spike"
  name         = "Research spike"
  base_type    = "spike"
  description  = "A time-boxed investigation that ends in a decision"
}
