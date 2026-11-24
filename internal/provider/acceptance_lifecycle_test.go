package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// A repository archived and unarchived in place, one archived as it's created, and both deleted
// for good: checkDestroyed looks for them among the pending deletions.
func TestAccRepositoryArchive(t *testing.T) {
	const address, born = "gitmoap_repository.test", "gitmoap_repository.born_archived"
	name := "tfacc-" + acctest.RandString(8)
	config := func(description string, archived, purge bool) string {
		return subgroupConfig(name, fmt.Sprintf(`
resource "gitmoap_repository" "test" {
  group_id         = gitmoap_group.test.group_id
  repository_name  = "app"
  description      = %q
  visibility       = "private"
  listed           = false
  archived         = %t
  purge_on_destroy = %t
}

# Archived after the update that sets what only an update takes.
resource "gitmoap_repository" "born_archived" {
  group_id          = gitmoap_group.test.group_id
  repository_name   = "old"
  visibility        = "private"
  pipelines_enabled = false
  archived          = true
  purge_on_destroy  = true
}
`, description, archived, purge))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config("Created by the provider's acceptance tests", false, false),
			ConfigStateChecks: []statecheck.StateCheck{
				expect(address, "archived", knownvalue.Bool(false)),
				expect(address, "archived_at", knownvalue.Null()),
				expect(address, "listed", knownvalue.Bool(false)),
				expect(born, "archived", knownvalue.Bool(true)),
				expect(born, "archived_at", knownvalue.NotNull()),
				expect(born, "pipelines_enabled", knownvalue.Bool(false)),
			},
		},
		resource.TestStep{
			Config:           config("Created by the provider's acceptance tests", true, false),
			ConfigPlanChecks: expectAction(address, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{
				expect(address, "archived", knownvalue.Bool(true)),
				expect(address, "archived_at", knownvalue.NotNull()),
			},
		},
		// Only purge_on_destroy changes: nothing goes to the archived repository, which would refuse it.
		resource.TestStep{
			Config:            config("Created by the provider's acceptance tests", true, true),
			ConfigPlanChecks:  expectAction(address, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{expect(address, "purge_on_destroy", knownvalue.Bool(true))},
		},
		// Unarchived before the description changes, which an archived repository refuses.
		resource.TestStep{
			Config: config("Updated by the provider's acceptance tests", false, true),
			ConfigStateChecks: []statecheck.StateCheck{
				expect(address, "archived", knownvalue.Bool(false)),
				expect(address, "description", knownvalue.StringExact("Updated by the provider's acceptance tests")),
			},
		},
		importStep(address, "gitmoap_repository", "purge_on_destroy"),
	))
}

// List filters take several values, and each has a _not twin that leaves values out. The services
// are external, so they need no repository.
func TestAccDataSourceFilters(t *testing.T) {
	name := "tfacc-" + acctest.RandString(8)
	resource.Test(t, testCase(t, resource.TestStep{
		Config: subgroupConfig(name, `
resource "gitmoap_catalog_service" "api" {
  group_id = gitmoap_group.test.group_id
  slug     = "api"
  kind     = "service"
  external = true
}

resource "gitmoap_catalog_service" "sdk" {
  group_id = gitmoap_group.test.group_id
  slug     = "sdk"
  kind     = "library"
  external = true
}

resource "gitmoap_catalog_service" "docs" {
  group_id = gitmoap_group.test.group_id
  slug     = "docs"
  kind     = "website"
  external = true
}

data "gitmoap_group_catalog_services" "code" {
  group_id   = gitmoap_group.test.group_id
  kind       = ["service", "library"]
  depends_on = [gitmoap_catalog_service.api, gitmoap_catalog_service.sdk, gitmoap_catalog_service.docs]
}

data "gitmoap_group_catalog_services" "not_services" {
  group_id   = gitmoap_group.test.group_id
  kind_not   = ["service"]
  depends_on = [gitmoap_catalog_service.api, gitmoap_catalog_service.sdk, gitmoap_catalog_service.docs]
}
`),
		Check: resource.ComposeTestCheckFunc(
			expectKinds("data.gitmoap_group_catalog_services.code", "library", "service"),
			expectKinds("data.gitmoap_group_catalog_services.not_services", "library", "website"),
		),
	}))
}

// expectKinds checks a catalog service list holds services of exactly these kinds, one of each.
func expectKinds(address string, kinds ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		attributes := s.RootModule().Resources[address].Primary.Attributes
		count, _ := strconv.Atoi(attributes["items.#"])
		found := map[string]int{}
		for index := range count {
			found[attributes[fmt.Sprintf("items.%d.service.kind", index)]]++
		}
		if len(found) != len(kinds) || count != len(kinds) {
			return fmt.Errorf("%s: got kinds %v, want one each of %v", address, found, kinds)
		}
		for _, kind := range kinds {
			if found[kind] != 1 {
				return fmt.Errorf("%s: got kinds %v, want one each of %v", address, found, kinds)
			}
		}
		return nil
	}
}

// A mirror of a small public repository, whose interval changes in place. A sync runs as soon as
// it's set, so what the last one did isn't compared on import.
func TestAccRepositoryMirror(t *testing.T) {
	const address = "gitmoap_repository_mirror.test"
	name := "tfacc-" + acctest.RandString(8)
	config := func(interval int) string {
		return subgroupConfig(name, fmt.Sprintf(`
resource "gitmoap_repository" "test" {
  group_id         = gitmoap_group.test.group_id
  repository_name  = "hello-world"
  visibility       = "private"
  purge_on_destroy = true
}

resource "gitmoap_repository_mirror" "test" {
  repo_id          = gitmoap_repository.test.repository_id
  source_url       = "https://github.com/octocat/Hello-World.git"
  interval_minutes = %d
  run_pipelines    = false
}
`, interval))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config(60),
			ConfigStateChecks: []statecheck.StateCheck{
				expect(address, "source_url", knownvalue.StringExact("https://github.com/octocat/Hello-World.git")),
				expect(address, "interval_minutes", knownvalue.Int64Exact(60)),
				expect(address, "run_pipelines", knownvalue.Bool(false)),
				expect(address, "next_sync_at", knownvalue.NotNull()),
			},
		},
		resource.TestStep{
			Config:            config(120),
			ConfigPlanChecks:  expectAction(address, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{expect(address, "interval_minutes", knownvalue.Int64Exact(120))},
		},
		importStep(address, "gitmoap_repository_mirror", "last_sync_at", "last_sync_error", "last_sync_result", "next_sync_at", "source_ahead", "paused_reason"),
	))
}

// The outbound policy is the organization's, so this runs on the test group itself, which has to be
// a top-level group. It sets the pipelines' mode and the registries' switch to what they are, so
// the organization's code reaches what it did, and allows a name nobody uses, on a port nobody does.
func TestAccOutbound(t *testing.T) {
	const policy, rule = "gitmoap_group_outbound.test", "gitmoap_group_outbound_rule.test"
	host := "tfacc-" + acctest.RandString(8) + ".example.com"
	config := func(note string, port int) string {
		return groupConfig(fmt.Sprintf(`
data "gitmoap_group_outbound" "current" {
  group_id = data.gitmoap_group_by_path.test.group_id
}

resource "gitmoap_group_outbound" "test" {
  group_id            = data.gitmoap_group_by_path.test.group_id
  platform_registries = data.gitmoap_group_outbound.current.platform_registries
  services = [
    for service in data.gitmoap_group_outbound.current.services : {
      service = service.service
      mode    = service.mode
    } if service.service == "pipelines"
  ]
}

resource "gitmoap_group_outbound_rule" "test" {
  group_id = data.gitmoap_group_by_path.test.group_id
  kind     = "host"
  value    = %q
  ports    = [%d]
  services = ["pipelines"]
  note     = %q
}
`, host, port, note))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config("Created by the provider's acceptance tests", 8443),
			ConfigStateChecks: []statecheck.StateCheck{
				// Only the service the configuration names: the policy lists every one.
				expect(policy, "services", knownvalue.ListSizeExact(1)),
				expect(policy, "version", knownvalue.NotNull()),
				expect(rule, "rule_id", knownvalue.NotNull()),
				expect(rule, "value", knownvalue.StringExact(host)),
			},
		},
		resource.TestStep{
			Config:            config("Updated by the provider's acceptance tests", 9443),
			ConfigPlanChecks:  expectAction(rule, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{expect(rule, "ports", knownvalue.ListExact([]knownvalue.Check{knownvalue.Int64Exact(9443)}))},
		},
		importStep(rule, "gitmoap_group_outbound_rule", "last_admitted_at"),
		// Imported, the policy holds every service: the configuration names one.
		importStep(policy, "gitmoap_group_outbound", "services"),
	))
}
