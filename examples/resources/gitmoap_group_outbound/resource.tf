data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# The organization's AI workspaces reach only what a rule allows, and its pipelines report what
# they reach that no rule allows before that's enforced too. The platform's package registries stay
# reachable without a rule. Only the services listed here are managed: any other keeps its mode.
resource "gitmoap_group_outbound" "acme" {
  group_id            = data.gitmoap_group_by_path.acme.group_id
  platform_registries = true

  services = [
    {
      service = "ai_workspaces"
      mode    = "enforce"
    },
    {
      service = "pipelines"
      mode    = "report"
    },
  ]
}
