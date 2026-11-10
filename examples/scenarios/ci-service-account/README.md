# A service account for CI, its token stored for the pipelines

A service account gets the access it needs and no more. Its token is narrowed further, to read
one repository, and handed to that repository's pipelines as a secret variable: nobody copies it
by hand. To rotate it, run `terraform apply -replace=gitmoap_service_account_token.ci`.
