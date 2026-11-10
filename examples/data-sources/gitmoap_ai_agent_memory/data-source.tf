variable "agent_id" {
  type = string
}

data "gitmoap_ai_agent_memory" "example" {
  agent_id = var.agent_id
}
