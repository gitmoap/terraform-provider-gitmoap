terraform {
  required_providers {
    gitmoap = {
      source  = "gitmoap/gitmoap"
      version = "~> 0.0.1"
    }
  }
}

# The token is read from GITMOAP_TOKEN when it isn't set here.
provider "gitmoap" {}
