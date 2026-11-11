data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_issue_blueprint" "api_feature" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  blueprint_id = "api-feature"
  name         = "API feature"
  base_type    = "story"
  description  = "A change clients of the API can use"

  facets = [
    {
      key      = "docs"
      label    = "Documented"
      category = "documentation"
      hint     = "The OpenAPI spec describes the change"
    },
    {
      key      = "observability"
      label    = "Observable"
      category = "operations"
      hint     = "Dashboards and alerts cover the new endpoints"
      required = false
    },
  ]
}

# Work items follow their merge requests: in review when one opens, done when it merges, and back
# where they were when it's closed without merging.
resource "gitmoap_group_issue_settings" "acme" {
  group_id           = data.gitmoap_group_by_path.acme.group_id
  default_issue_type = "story"
  on_mr_opened       = { to = "in_review" }
  on_mr_merged       = { to = "done" }
  on_mr_closed       = { value = "restore" }
}
