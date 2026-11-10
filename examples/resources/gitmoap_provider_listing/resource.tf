# How a pipeline provider you publish appears in the marketplace.
resource "gitmoap_provider_listing" "deploy" {
  provider_id    = "acme-deploy"
  categories     = ["deploy", "infrastructure"]
  keywords       = ["kubernetes", "helm"]
  homepage_url   = "https://acme.example.com/deploy"
  repository_url = "https://gitmoap.com/acme/deploy-provider"
  readme         = <<-EOT
    # acme-deploy
    Deploys a Helm chart to a Kubernetes cluster.
  EOT
}
