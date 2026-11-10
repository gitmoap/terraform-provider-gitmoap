variable "provider_id" {
  type = string
}

data "gitmoap_provider" "example" {
  id = var.provider_id
}
