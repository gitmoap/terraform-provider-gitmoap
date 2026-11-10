variable "agent_id" {
  type = string
}

data "gitmoap_ai_agent" "example" {
  agent_id = var.agent_id
}
