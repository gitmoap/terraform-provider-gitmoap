# A service account for CI, its token handed to the pipelines

A service account gets the access it needs and no more, and its token is narrowed to one
repository. Terraform hands the token to that repository's pipelines, as a secret variable and as
the credentials of a gitmoap provider connection, both write-only: nobody copies it by hand, and
neither keeps it in the state. When the token is replaced, say with
`terraform apply -replace=gitmoap_service_account_token.ci`, both are replaced with it.
