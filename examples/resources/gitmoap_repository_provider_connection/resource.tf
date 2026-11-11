data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# Deploys assume a role in AWS through OIDC federation: no secret is stored anywhere.
resource "gitmoap_repository_provider_connection" "aws" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  provider_id = "aws"
  name        = "production"
  auth_mode   = "oidc_federation"
  protected   = true

  federation_target = {
    cloud    = "aws"
    role_arn = "arn:aws:iam::123456789012:role/gitmoap-deploy"
    region   = "eu-west-1"
  }
}
