variable "group_provider_connection_id" {
  type = string
}

data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_provider_connection" "acme" {
  group_id = data.gitmoap_group_by_path.acme.group_id
  id       = var.group_provider_connection_id
}
