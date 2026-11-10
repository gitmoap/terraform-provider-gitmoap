# Single sign-on through an identity provider

The organization signs people in through Okta over OIDC: it claims its domain, connects the
identity provider, and gives it a SCIM token to create and remove accounts. Someone from outside
the domain gets an invite. Once `enforced`, everyone from the domain has to sign in through Okta,
so check the connection works first.
