# Webhooks with generated secrets

Each endpoint checks deliveries against a secret, generated here with the
[random provider](https://registry.terraform.io/providers/hashicorp/random/latest/docs) rather than
written down. The secrets end up in Terraform's state, so keep the state somewhere safe.
