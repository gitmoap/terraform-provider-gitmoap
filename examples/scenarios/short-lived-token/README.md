# A short-lived token for one run

An ephemeral token, minted when Terraform opens it and revoked when it's done, configures a second
gitmoap provider that only reads groups: the rest of the run can't write anything through it, and
the token is never in the plan or the state (Terraform 1.10 and later). The service account it acts
as exists already, since Terraform can't configure a provider with what the same run creates.
