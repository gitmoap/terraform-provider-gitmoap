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

# New work items are stories unless their author picks another type.
resource "gitmoap_group_issue_settings" "acme" {
  group_id           = data.gitmoap_group_by_path.acme.group_id
  default_issue_type = "story"
}
