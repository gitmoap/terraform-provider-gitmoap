variable "gate_id" {
  type = string
}

data "gitmoap_ai_gate" "example" {
  gate_id = var.gate_id
}
