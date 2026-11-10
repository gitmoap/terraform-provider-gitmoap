variable "triage_agent_id" {
  type = string
}

# The memory page an AI agent reads at the start of every session.
resource "gitmoap_ai_agent_memory" "triage" {
  agent_id = var.triage_agent_id
  note     = "Start triage from the on-call runbook"
  body     = <<-EOT
    # Triage
    - Label every new issue within a day.
    - Page the on-call engineer for anything labelled `security`.
  EOT
}
