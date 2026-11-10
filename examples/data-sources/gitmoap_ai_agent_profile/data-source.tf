variable "agent_id" {
  type = string
}

data "gitmoap_ai_agent_profile" "example" {
  agent_id = var.agent_id
}
