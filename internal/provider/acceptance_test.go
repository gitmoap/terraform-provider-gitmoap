package provider

// Acceptance tests create and delete real objects in gitmoap. They run only with TF_ACC set:
//
//	TF_ACC=1 GITMOAP_TOKEN=... GITMOAP_TEST_GROUP=gitmoap-showcase go test ./internal/provider -run TestAcc
//
// GITMOAP_TEST_GROUP is the full path of a group the token is at least a Maintainer of. What they
// create is named tfacc-<random>, and deleted at the end even when a step fails.

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

var testProviders = map[string]func() (tfprotov6.ProviderServer, error){
	"gitmoap": providerserver.NewProtocol6WithError(New("test")()),
}

func preCheck(t *testing.T) {
	for _, name := range []string{tokenEnv, "GITMOAP_TEST_GROUP"} {
		if os.Getenv(name) == "" {
			t.Fatalf("%s has to be set for acceptance tests", name)
		}
	}
}

func testCase(t *testing.T, steps ...resource.TestStep) resource.TestCase {
	return resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testProviders,
		CheckDestroy:             checkDestroyed,
		Steps:                    steps,
	}
}

// groupConfig reads the test group, as data.gitmoap_group_by_path.test.
func groupConfig(config string) string {
	return fmt.Sprintf("data \"gitmoap_group_by_path\" \"test\" {\n  full_path = %q\n}\n", os.Getenv("GITMOAP_TEST_GROUP")) + config
}

func expect(address, attribute string, value knownvalue.Check) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(address, tfjsonpath.New(attribute), value)
}

func expectAction(address string, action plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, action)}}
}

// importStep imports address by its identity attributes and checks it reads back the same state,
// but for the attributes in ignore.
func importStep(address, typeName string, ignore ...string) resource.TestStep {
	identity := resourceAPIs[strings.TrimPrefix(typeName, "gitmoap_")].Identity
	return resource.TestStep{
		ResourceName: address,
		ImportState:  true,
		ImportStateIdFunc: func(s *terraform.State) (string, error) {
			attributes := s.RootModule().Resources[address].Primary.Attributes
			parts := make([]string, len(identity))
			for index, attribute := range identity {
				parts[index] = attributes[attribute]
			}
			return strings.Join(parts, "/"), nil
		},
		ImportStateVerify:                    true,
		ImportStateVerifyIdentifierAttribute: identity[len(identity)-1],
		ImportStateVerifyIgnore:              ignore,
	}
}

func TestAccDataSources(t *testing.T) {
	resource.Test(t, testCase(t, resource.TestStep{
		Config: groupConfig(`
data "gitmoap_group" "test" {
  group_id = data.gitmoap_group_by_path.test.group_id
}

data "gitmoap_group_members" "test" {
  group_id = data.gitmoap_group_by_path.test.group_id
}

data "gitmoap_templates" "all" {}
`),
		ConfigStateChecks: []statecheck.StateCheck{
			expect("data.gitmoap_group_by_path.test", "full_path", knownvalue.StringExact(os.Getenv("GITMOAP_TEST_GROUP"))),
			statecheck.CompareValuePairs(
				"data.gitmoap_group.test", tfjsonpath.New("group_id"),
				"data.gitmoap_group_by_path.test", tfjsonpath.New("group_id"),
				compareEqual{},
			),
			expect("data.gitmoap_group_members.test", "items", knownvalue.NotNull()),
			expect("data.gitmoap_templates.all", "items", knownvalue.NotNull()),
		},
	}))
}

func TestAccGroupLabel(t *testing.T) {
	const address = "gitmoap_group_label.test"
	key := "tfacc-" + acctest.RandString(8)
	config := func(key, color, description string) string {
		return groupConfig(fmt.Sprintf(`
resource "gitmoap_group_label" "test" {
  group_id    = data.gitmoap_group_by_path.test.group_id
  key         = %q
  color       = %q
  description = %q
}
`, key, color, description))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config(key, "#d73a4a", "Created by the provider's acceptance tests"),
			ConfigStateChecks: []statecheck.StateCheck{
				expect(address, "key", knownvalue.StringExact(key)),
				expect(address, "color", knownvalue.StringExact("#d73a4a")),
				expect(address, "created_at", knownvalue.NotNull()),
				expect(address, "scope_type", knownvalue.NotNull()),
			},
		},
		resource.TestStep{
			Config:            config(key, "#0e8a16", "Updated by the provider's acceptance tests"),
			ConfigPlanChecks:  expectAction(address, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{expect(address, "color", knownvalue.StringExact("#0e8a16"))},
		},
		// labelOverride: a new key renames the label in place.
		resource.TestStep{
			Config:            config(key+"-renamed", "#0e8a16", "Updated by the provider's acceptance tests"),
			ConfigPlanChecks:  expectAction(address, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{expect(address, "key", knownvalue.StringExact(key+"-renamed"))},
		},
		importStep(address, "gitmoap_group_label"),
	))
}

func TestAccGroupMilestone(t *testing.T) {
	const address = "gitmoap_group_milestone.test"
	title := "tfacc-" + acctest.RandString(8)
	config := func(title, dueOn string) string {
		return groupConfig(fmt.Sprintf(`
resource "gitmoap_group_milestone" "test" {
  group_id    = data.gitmoap_group_by_path.test.group_id
  title       = %q
  description = "Created by the provider's acceptance tests"
  due_on      = %q
}
`, title, dueOn))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config(title, "2027-01-31"),
			ConfigStateChecks: []statecheck.StateCheck{
				expect(address, "milestone_id", knownvalue.NotNull()),
				expect(address, "due_on", knownvalue.StringExact("2027-01-31")),
				expect(address, "closed", knownvalue.Bool(false)),
			},
		},
		resource.TestStep{
			Config:            config(title+"-moved", "2027-02-28"),
			ConfigPlanChecks:  expectAction(address, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{expect(address, "due_on", knownvalue.StringExact("2027-02-28"))},
		},
		importStep(address, "gitmoap_group_milestone"),
	))
}

func TestAccGroupIteration(t *testing.T) {
	const address = "gitmoap_group_iteration.test"
	title := "tfacc-" + acctest.RandString(8)
	config := func(title, start, due string) string {
		return groupConfig(fmt.Sprintf(`
resource "gitmoap_group_iteration" "test" {
  group_id   = data.gitmoap_group_by_path.test.group_id
  title      = %q
  start_date = %q
  due_date   = %q
}
`, title, start, due))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config(title, "2027-03-01", "2027-03-14"),
			ConfigStateChecks: []statecheck.StateCheck{
				expect(address, "iteration_id", knownvalue.NotNull()),
				expect(address, "start_date", knownvalue.StringExact("2027-03-01")),
			},
		},
		resource.TestStep{
			Config:            config(title, "2027-03-01", "2027-03-21"),
			ConfigPlanChecks:  expectAction(address, plancheck.ResourceActionUpdate),
			ConfigStateChecks: []statecheck.StateCheck{expect(address, "due_date", knownvalue.StringExact("2027-03-21"))},
		},
		importStep(address, "gitmoap_group_iteration"),
	))
}

// checkDestroyed checks the API no longer has what the test created, reading it the way the
// resource does.
func checkDestroyed(s *terraform.State) error {
	endpoint := firstSet(os.Getenv(endpointEnv), defaultEndpoint)
	client := NewClient(endpoint, os.Getenv(tokenEnv), "terraform-provider-gitmoap/test")
	for address, state := range s.RootModule().Resources {
		api, ok := resourceAPIs[strings.TrimPrefix(state.Type, "gitmoap_")]
		if !ok || strings.HasPrefix(address, "data.") {
			continue
		}
		attributes := state.Primary.Attributes
		path := pathParameter.ReplaceAllStringFunc(api.Read.Path, func(segment string) string {
			return url.PathEscape(attributes[api.Params[segment[1:len(segment)-1]]])
		})
		response, err := client.Get(context.Background(), path, nil, api.Read.Pages)
		if IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if api.Match == nil {
			return fmt.Errorf("%s still exists", address)
		}
		want := map[string]string{}
		for field, attribute := range api.Match {
			want[field] = attributes[attribute]
		}
		items, _ := listItems(response, api.Read.Items)
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			if override := resourceOverrides[strings.TrimPrefix(state.Type, "gitmoap_")]; item != nil && override.Response != nil {
				if err := override.Response(context.Background(), "read", item); err != nil {
					return err
				}
			}
			if item != nil && matchesItem(item, want) {
				return fmt.Errorf("%s still exists", address)
			}
		}
	}
	return nil
}

type compareEqual struct{}

func (compareEqual) CompareValues(values ...any) error {
	if len(values) == 2 && fmt.Sprint(values[0]) == fmt.Sprint(values[1]) {
		return nil
	}
	return fmt.Errorf("expected equal values, got %v", values)
}

// subgroupConfig adds a throwaway subgroup of the test group, as gitmoap_group.test, for tests that
// change group-wide settings.
func subgroupConfig(name, config string) string {
	return groupConfig(fmt.Sprintf(`
resource "gitmoap_group" "test" {
  group_name      = %q
  parent_group_id = data.gitmoap_group_by_path.test.group_id
  description     = "Created by the provider's acceptance tests"
  visibility      = "private"
}
`, name) + config)
}

func TestAccGroupResources(t *testing.T) {
	name := "tfacc-" + acctest.RandString(8)
	config := func(size int, squash, page, secret string) string {
		return subgroupConfig(name, fmt.Sprintf(`
resource "gitmoap_group_push_rules" "test" {
  group_id         = gitmoap_group.test.group_id
  max_file_size_mb = %d
}

resource "gitmoap_group_mr_settings" "test" {
  group_id      = gitmoap_group.test.group_id
  squash_option = %q
}

resource "gitmoap_group_fork_settings" "test" {
  group_id = gitmoap_group.test.group_id
  forking  = "organization"
}

resource "gitmoap_group_approval_policy" "test" {
  group_id                = gitmoap_group.test.group_id
  author_can_approve      = false
  committers_can_approve  = true
  last_pusher_can_approve = false
  reset_on_push           = true
}

resource "gitmoap_group_branch_rule" "test" {
  group_id         = gitmoap_group.test.group_id
  branch_pattern   = "main"
  allowed_to_push  = "maintainer"
  allowed_to_merge = "developer"
}

resource "gitmoap_group_wiki" "test" {
  group_id = gitmoap_group.test.group_id
  enabled  = true
  source   = "dedicated"
}

resource "gitmoap_group_wiki_page" "home" {
  group_id   = gitmoap_group.test.group_id
  path       = "home"
  title      = "Home"
  content    = %q
  depends_on = [gitmoap_group_wiki.test]
}

resource "gitmoap_group_wiki_page" "about" {
  group_id   = gitmoap_group.test.group_id
  path       = "about"
  title      = "About"
  content    = %q
  depends_on = [gitmoap_group_wiki.test]
}

resource "gitmoap_group_pipeline_variable" "secret" {
  group_id = gitmoap_group.test.group_id
  name     = "TFACC_TOKEN"
  value    = %q
  secret   = true
}

resource "gitmoap_group_issue_blueprint" "test" {
  group_id     = gitmoap_group.test.group_id
  blueprint_id = "incident"
  name         = "Incident"
  base_type    = "incident"
}

resource "gitmoap_group_service_account" "test" {
  group_id     = gitmoap_group.test.group_id
  name         = "tfacc-bot"
  access_level = "reporter"
}

resource "gitmoap_service_account_token" "test" {
  sa_id  = gitmoap_group_service_account.test.service_account_id
  name   = "tfacc"
  scopes = ["repo:read"]
}
`, size, squash, page, page, secret))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config(50, "allow", "# Home", "first-secret"),
			ConfigStateChecks: []statecheck.StateCheck{
				expect("gitmoap_group_fork_settings.test", "forking", knownvalue.StringExact("organization")),
				expect("gitmoap_group_approval_policy.test", "reset_on_push", knownvalue.Bool(true)),
				expect("gitmoap_group_wiki_page.home", "content", knownvalue.StringExact("# Home")),
				expect("gitmoap_service_account_token.test", "token", knownvalue.NotNull()),
			},
		},
		// Changing the group's own attributes must not replace what's in it.
		resource.TestStep{
			Config: strings.Replace(config(25, "require", "# Home, updated", "second-secret"), "Created by", "Updated by", 1),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("gitmoap_group.test", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_group_push_rules.test", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_group_wiki_page.home", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_group_wiki_page.about", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_group_pipeline_variable.secret", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_service_account_token.test", plancheck.ResourceActionNoop),
			}},
			ConfigStateChecks: []statecheck.StateCheck{expect("gitmoap_group_push_rules.test", "max_file_size_mb", knownvalue.Int64Exact(25))},
		},
		importStep("gitmoap_group_fork_settings.test", "gitmoap_group_fork_settings"),
		importStep("gitmoap_group_approval_policy.test", "gitmoap_group_approval_policy"),
		importStep("gitmoap_group_branch_rule.test", "gitmoap_group_branch_rule"),
		// Right after an update, the API still serves the page's previous table of contents.
		importStep("gitmoap_group_wiki_page.home", "gitmoap_group_wiki_page", "toc"),
		// Creating a service account returns its creator's avatar_url, and reading it doesn't.
		importStep("gitmoap_group_service_account.test", "gitmoap_group_service_account", "created_by.avatar_url"),
	))
}

func TestAccRepositoryResources(t *testing.T) {
	name := "tfacc-" + acctest.RandString(8)
	config := func(description, key, level string) string {
		return subgroupConfig(name, fmt.Sprintf(`
resource "gitmoap_repository" "test" {
  group_id        = gitmoap_group.test.group_id
  repository_name = "app"
  description     = %q
  visibility      = "private"
}

resource "gitmoap_repository_mr_settings" "test" {
  repo_id                 = gitmoap_repository.test.repository_id
  override_group_settings = true
  squash_option           = "encourage"
}

resource "gitmoap_repository_issue_settings" "test" {
  repo_id                 = gitmoap_repository.test.repository_id
  override_group_settings = true
  default_issue_type      = "bug"
}

resource "gitmoap_repository_label" "test" {
  repo_id = gitmoap_repository.test.repository_id
  key     = %q
  color   = "#d73a4a"
}

resource "gitmoap_repository_pipeline_variable" "test" {
  repo_id = gitmoap_repository.test.repository_id
  name    = "LOG_LEVEL"
  value   = %q
}
`, description, key, level))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config("Created by the provider's acceptance tests", "bug", "debug"),
			ConfigStateChecks: []statecheck.StateCheck{
				expect("gitmoap_repository.test", "repository_id", knownvalue.NotNull()),
				expect("gitmoap_repository_issue_settings.test", "default_issue_type", knownvalue.StringExact("bug")),
			},
		},
		resource.TestStep{
			Config: config("Updated by the provider's acceptance tests", "kind-bug", "info"),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("gitmoap_repository.test", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_repository_label.test", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_repository_mr_settings.test", plancheck.ResourceActionNoop),
			}},
		},
		importStep("gitmoap_repository.test", "gitmoap_repository"),
		importStep("gitmoap_repository_issue_settings.test", "gitmoap_repository_issue_settings"),
		// Only creating a variable returns who created it and when: the list it's read from doesn't.
		importStep("gitmoap_repository_pipeline_variable.test", "gitmoap_repository_pipeline_variable", "created_at", "created_by", "updated_at"),
	))
}

// Teams belong to the top-level group, so this one goes in the test group itself.
func TestAccTeam(t *testing.T) {
	slug := "tfacc-" + acctest.RandString(8)
	me := os.Getenv("GITMOAP_TEST_USER_ID")
	if me == "" {
		t.Skip("GITMOAP_TEST_USER_ID, the user the token belongs to, isn't set")
	}
	config := func(description, role string) string {
		return groupConfig(fmt.Sprintf(`
resource "gitmoap_team" "test" {
  group_id    = data.gitmoap_group_by_path.test.group_id
  name        = %q
  slug        = %q
  description = %q
}

resource "gitmoap_team_member" "test" {
  team_id = gitmoap_team.test.team_id
  user_id = %q
  role    = %q
}
`, slug, slug, description, me, role))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{Config: config("Created by the provider's acceptance tests", "maintainer")},
		resource.TestStep{
			Config: config("Updated by the provider's acceptance tests", "member"),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("gitmoap_team.test", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_team_member.test", plancheck.ResourceActionUpdate),
			}},
		},
		importStep("gitmoap_team.test", "gitmoap_team"),
		importStep("gitmoap_team_member.test", "gitmoap_team_member"),
	))
}

// Approval rules and grants are read back from lists their group keeps; a webhook's write-only
// secret is sent again when its version changes, and never kept in the state.
func TestAccRulesGrantsAndWriteOnly(t *testing.T) {
	name := "tfacc-" + acctest.RandString(8)
	config := func(required int, reason, secret string, version int) string {
		return subgroupConfig(name, fmt.Sprintf(`
resource "gitmoap_team" "test" {
  group_id = data.gitmoap_group_by_path.test.group_id
  name     = %[1]q
  slug     = %[1]q
}

resource "gitmoap_group_access_grant" "test" {
  group_id = gitmoap_group.test.group_id
  subject  = { team = gitmoap_team.test.team_id }
  role     = { builtin = "developer" }
  reason   = %[3]q
}

resource "gitmoap_group_approval_rule" "test" {
  group_id  = gitmoap_group.test.group_id
  name      = "Reviewed"
  required  = %[2]d
  approvers = [{ team = gitmoap_team.test.team_id }, { code_owners = true }]
}

resource "gitmoap_group_webhook" "test" {
  group_id          = gitmoap_group.test.group_id
  url               = "https://example.com/gitmoap-webhook"
  secret_wo         = %[4]q
  secret_wo_version = %[5]d
  events            = ["push"]
  active            = false
}
`, name, required, reason, secret, version))
	}
	resource.Test(t, testCase(t,
		resource.TestStep{
			Config: config(1, "Reviews", "first-secret", 1),
			ConfigStateChecks: []statecheck.StateCheck{
				expect("gitmoap_group_access_grant.test", "grant_id", knownvalue.NotNull()),
				expect("gitmoap_group_approval_rule.test", "rule_id", knownvalue.NotNull()),
				expect("gitmoap_group_webhook.test", "secret", knownvalue.Null()),
				expect("gitmoap_group_webhook.test", "secret_wo", knownvalue.Null()),
			},
		},
		resource.TestStep{
			Config: config(2, "Reviews payments", "second-secret", 2),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("gitmoap_group_access_grant.test", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_group_approval_rule.test", plancheck.ResourceActionUpdate),
				plancheck.ExpectResourceAction("gitmoap_group_webhook.test", plancheck.ResourceActionUpdate),
			}},
			ConfigStateChecks: []statecheck.StateCheck{
				expect("gitmoap_group_approval_rule.test", "required", knownvalue.Int64Exact(2)),
				expect("gitmoap_group_webhook.test", "secret_wo", knownvalue.Null()),
			},
		},
		// The access list grants are read back from leaves out where they're made, and when they
		// last changed: only creating and changing one return them.
		importStep("gitmoap_group_access_grant.test", "gitmoap_group_access_grant", "scope", "updated_at"),
		importStep("gitmoap_group_approval_rule.test", "gitmoap_group_approval_rule"),
		importStep("gitmoap_group_webhook.test", "gitmoap_group_webhook", "secret_wo_version"),
	))
}
