variable "draft_id" {
  type = string
}

data "gitmoap_ai_profile_draft" "example" {
  draft_id = var.draft_id
}
