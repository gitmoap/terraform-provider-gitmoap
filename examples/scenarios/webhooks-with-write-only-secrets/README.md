# Webhooks whose secrets never reach the state

Each endpoint checks deliveries against a secret. Here the secrets come in through an ephemeral
variable, say from `TF_VAR_webhook_secrets`, and go to gitmoap write-only, so they're in neither
Terraform's plan nor its state (Terraform 1.11 and later). Terraform can't tell when a value it
doesn't keep changes: after changing a secret, change its `secret_wo_version` too.
