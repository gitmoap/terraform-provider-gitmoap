data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

data "gitmoap_group_package" "acme" {
  format       = "npm"
  group_id     = data.gitmoap_group_by_path.acme.group_id
  package_name = "@acme/ui"
}
