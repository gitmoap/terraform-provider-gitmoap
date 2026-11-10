data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

# The conventions every AI agent in the organization reads at the start of a session.
resource "gitmoap_group_ai_org_conventions" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  note     = "Write down how we work"
  body     = <<-EOT
    # Conventions
    - Every change goes through a merge request.
    - Tests come with the code they test.
  EOT
}
