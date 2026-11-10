variable "channel_id" {
  type = string
}

data "gitmoap_channel_members" "example" {
  channel_id = var.channel_id
}
