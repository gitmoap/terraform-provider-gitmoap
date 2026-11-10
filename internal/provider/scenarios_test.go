package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const scenariosDir = "../../examples/scenarios"

// How TestAccScenarios applies each scenario in examples/scenarios. A new scenario needs an entry.
var scenarioTests = map[string]scenarioTest{
	"access-for-people":               {skip: "it gives access to other people's accounts"},
	"ci-service-account":              {},
	"issue-types":                     {},
	"organization-defaults":           {},
	"protected-repository":            {},
	"release":                         {repository: "api"},
	"service-catalog":                 {},
	"single-sign-on":                  {skip: "it would change how everyone in the test group signs in"},
	"sprints-and-milestones":          {},
	"team-of-people":                  {topLevel: true, reviewers: true},
	"webhooks-with-generated-secrets": {},
	"wiki-from-files":                 {},
}

type scenarioTest struct {
	skip       string
	topLevel   bool   // it needs the organization's top-level group, as teams do
	repository string // a repository it reads, which the test creates with a commit in it
	reviewers  bool   // its reviewers variable takes the token's own username
}

var resourceBlock = regexp.MustCompile(`(?m)^resource "(\w+)" "(\w+)"`)

// scenarioText returns the Terraform files of a scenario, together.
func scenarioText(t *testing.T, name string) string {
	files, err := filepath.Glob(filepath.Join(scenariosDir, name, "*.tf"))
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		text.Write(content)
	}
	return text.String()
}

var acmePath = regexp.MustCompile(`full_path(\s*)=(\s*)"acme(/[^"]*)?"`)

// TestAccScenarios applies every scenario in the documentation, so its examples are known to work:
// then a plan has to find nothing to change, and it's destroyed. Each runs in a throwaway subgroup
// of the test group, standing in for the acme organization. It drives Terraform directly, since
// the testing framework can't read a state that uses for_each, as scenarios do.
func TestAccScenarios(t *testing.T) {
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, entry := range entries {
		found[entry.Name()] = true
		if _, ok := scenarioTests[entry.Name()]; !ok {
			t.Errorf("examples/scenarios/%s has no entry in scenarioTests: add one", entry.Name())
		}
	}
	for name := range scenarioTests {
		if !found[name] {
			t.Errorf("scenarioTests has %s, which isn't in examples/scenarios any more: remove it", name)
		}
	}
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("applying the scenarios is an acceptance test: set %s", resource.EnvTfAcc)
	}
	preCheck(t)
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("applying the scenarios needs terraform on PATH")
	}
	cliConfig := buildProvider(t)

	for _, entry := range entries {
		name, test := entry.Name(), scenarioTests[entry.Name()]
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skipf("not applied: %s", test.skip)
			}
			runScenario(t, terraform, cliConfig, name, test)
		})
	}
}

func runScenario(t *testing.T, terraform, cliConfig, name string, test scenarioTest) {
	if test.repository != "" {
		if _, err := exec.LookPath("git-remote-gitmoap"); err != nil {
			t.Skip("needs git-remote-gitmoap, from gitmoap's command-line tools, to push a commit")
		}
	}
	ctx := context.Background()
	directory := t.TempDir()
	tf, err := tfexec.NewTerraform(directory, terraform)
	if err != nil {
		t.Fatal(err)
	}
	// Only what Terraform and the provider need: none of the caller's other Terraform settings.
	env := map[string]string{"TF_CLI_CONFIG_FILE": cliConfig}
	for _, variable := range os.Environ() {
		name, value, _ := strings.Cut(variable, "=")
		if slices.Contains([]string{"PATH", "HOME", "TMPDIR", "TF_PLUGIN_CACHE_DIR"}, name) || strings.HasPrefix(name, "GITMOAP_") {
			env[name] = value
		}
	}
	if err := tf.SetEnv(env); err != nil {
		t.Fatal(err)
	}

	planFile := filepath.Join(t.TempDir(), "scenario.tfplan")
	applyOptions := []tfexec.ApplyOption{tfexec.Parallelism(3)}
	planOptions := []tfexec.PlanOption{tfexec.Parallelism(3), tfexec.Out(planFile)}
	destroyOptions := []tfexec.DestroyOption{tfexec.Parallelism(3)}
	if test.reviewers {
		reviewers := tfexec.Var(fmt.Sprintf("reviewers=[%q]", tokenUsername(t)))
		applyOptions = append(applyOptions, reviewers)
		planOptions = append(planOptions, reviewers)
		destroyOptions = append(destroyOptions, reviewers)
	}

	// Whatever happens, destroy what was created.
	defer func() {
		if err := tf.Destroy(ctx, destroyOptions...); err != nil {
			t.Errorf("destroying the scenario, there may be dangling resources: %s", err)
		}
	}()

	acme, dependsOn := os.Getenv("GITMOAP_TEST_GROUP"), ""
	if !test.topLevel {
		subgroup := "tfacc-" + acctest.RandString(8)
		acme += "/" + subgroup
		dependsOn = "gitmoap_group.tfacc"
		setup := groupConfig(fmt.Sprintf(`
resource "gitmoap_group" "tfacc" {
  group_name      = %q
  parent_group_id = data.gitmoap_group_by_path.test.group_id
  description     = "Created by the provider's acceptance tests"
  visibility      = "private"
}
`, subgroup))
		if test.repository != "" {
			dependsOn += ", gitmoap_repository.tfacc"
			setup += fmt.Sprintf(`
resource "gitmoap_repository" "tfacc" {
  group_id        = gitmoap_group.tfacc.group_id
  repository_name = %q
  visibility      = "private"
}
`, test.repository)
		}
		writeFile(t, filepath.Join(directory, "tfacc_setup.tf"), setup)
		if err := tf.Init(ctx); err != nil {
			t.Fatalf("terraform init: %s", err)
		}
		if err := tf.Apply(ctx, tfexec.Parallelism(3)); err != nil {
			t.Fatalf("creating the subgroup to run in: %s", err)
		}
		if test.repository != "" {
			pushCommit(t, acme+"/"+test.repository)
		}
	}

	copyScenario(t, name, directory, acme, dependsOn)
	if err := tf.Init(ctx); err != nil {
		t.Fatalf("terraform init: %s", err)
	}
	if err := tf.Apply(ctx, applyOptions...); err != nil {
		t.Fatalf("applying the scenario: %s", err)
	}
	// Every resource block made something: a for_each that found nothing would pass unseen.
	state, err := tf.Show(ctx)
	if err != nil {
		t.Fatalf("reading the state: %s", err)
	}
	created := map[string]bool{}
	if state.Values != nil {
		for _, instance := range state.Values.RootModule.Resources {
			created[instance.Type+"."+instance.Name] = true
		}
	}
	for _, block := range resourceBlock.FindAllStringSubmatch(scenarioText(t, name), -1) {
		if !created[block[1]+"."+block[2]] {
			t.Errorf("%s.%s created nothing", block[1], block[2])
		}
	}

	changes, err := tf.Plan(ctx, planOptions...)
	if err != nil {
		t.Fatalf("planning again: %s", err)
	}
	if changes {
		plan, _ := tf.ShowPlanFileRaw(ctx, planFile)
		t.Errorf("after applying the scenario, a plan still finds changes:\n%s", plan)
	}
}

// copyScenario copies a scenario into directory, with acme standing for the group at path, which
// it looks up once dependsOn exists.
func copyScenario(t *testing.T, name, directory, path, dependsOn string) {
	source := filepath.Join(scenariosDir, name)
	err := filepath.WalkDir(source, func(file string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(source, file)
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if strings.HasSuffix(file, ".tf") {
			replacement := `full_path${1}=${2}"` + path + `${3}"`
			if dependsOn != "" {
				replacement += "\n  depends_on = [" + dependsOn + "]"
			}
			content = acmePath.ReplaceAll(content, []byte(replacement))
		}
		writeFile(t, filepath.Join(directory, relative), string(content))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// buildProvider builds the provider from this repository, and returns a Terraform CLI configuration
// that uses it in place of the published one.
func buildProvider(t *testing.T) string {
	directory := t.TempDir()
	build := exec.Command("go", "build", "-o", filepath.Join(directory, "terraform-provider-gitmoap"), ".")
	build.Dir = "../.."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the provider: %s\n%s", err, output)
	}
	cliConfig := filepath.Join(directory, "terraformrc")
	writeFile(t, cliConfig, fmt.Sprintf(`provider_installation {
  dev_overrides {
    "registry.terraform.io/hashicorp/gitmoap" = %q
  }
  direct {}
}
`, directory))
	return cliConfig
}

func writeFile(t *testing.T, path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// tokenUsername returns the username of the user the test token belongs to.
func tokenUsername(t *testing.T) string {
	var user struct {
		Username string `json:"username"`
	}
	client := NewClient(firstSet(os.Getenv(endpointEnv), defaultEndpoint), os.Getenv(tokenEnv), "terraform-provider-gitmoap/test")
	if err := client.Do(context.Background(), "GET", "/user", nil, nil, &user); err != nil {
		t.Fatalf("finding the token's user: %s", err)
	}
	return user.Username
}

// pushCommit pushes a first commit to main of the repository at path.
func pushCommit(t *testing.T, path string) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "README.md"), "# "+filepath.Base(path)+"\n")
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "README.md"},
		{"-c", "user.name=Terraform provider tests", "-c", "user.email=tfacc@example.com", "commit", "-q", "-m", "First commit"},
		{"push", "-q", "gitmoap://" + path, "main"},
	} {
		command := exec.Command("git", args...)
		command.Dir = directory
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %s\n%s", args[0], err, output)
		}
	}
}
