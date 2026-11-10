variable "channel_id" {
  type = string
}

data "gitmoap_channel" "example" {
  channel_id = var.channel_id
}
