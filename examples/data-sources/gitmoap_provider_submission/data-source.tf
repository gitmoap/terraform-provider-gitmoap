variable "submission_id" {
  type = string
}

data "gitmoap_provider_submission" "example" {
  submission_id = var.submission_id
}
