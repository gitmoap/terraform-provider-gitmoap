data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_ai_org_squad_memory" "platform" {
  group_id  = data.gitmoap_group_by_path.acme.group_id
  squad_key = "platform"
  note      = "What the platform squad owns"
  body      = <<-EOT
    # Platform squad
    Owns the API gateway, the deploy pipeline and the service catalog.
  EOT
}
