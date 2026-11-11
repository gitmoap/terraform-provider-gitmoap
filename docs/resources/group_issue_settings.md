---
page_title: "gitmoap_group_issue_settings Resource - gitmoap"
subcategory: ""
description: |-
  Changes what the group stores and returns it: only the fields you send change, and a field sent as null is cleared. The group's values reach the repositories and groups beneath it except where one overrides with its own value, and with override_group_settings on, groups further up no longer reach them. A status set needs one unarchived open status marked as the default and at least one unarchived done status, and an existing status cannot change category. Maintainers and above may do this; an empty body is refused.
  The token needs issue_settings:manage and issue_settings:read.
---

# gitmoap_group_issue_settings (Resource)

Changes what the group stores and returns it: only the fields you send change, and a field sent as `null` is cleared. The group's values reach the repositories and groups beneath it except where one overrides with its own value, and with `override_group_settings` on, groups further up no longer reach them. A status set needs one unarchived open status marked as the default and at least one unarchived done status, and an existing status cannot change category. Maintainers and above may do this; an empty body is refused.

The token needs `issue_settings:manage` and `issue_settings:read`.

## Example Usage

### Basic Usage

```terraform
data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_issue_settings" "platform" {
  group_id                = data.gitmoap_group_by_path.platform.group_id
  default_issue_type      = "task"
  close_on_commit_keyword = true
  todo_policy             = "index"

  # Work items follow their merge requests: in review when one opens, done when it merges, and back
  # where they were when it's closed without merging.
  on_mr_opened = { to = "in_review" }
  on_mr_merged = { to = "done" }
  on_mr_closed = { value = "restore" }
}
```

### Custom work item types

A blueprint is a work item type of the organization's own, built on a base type. Each of its
facets becomes an acceptance criterion on every work item of that type, and a required one has
to be met before the item closes.

The settings also move work items along with their merge requests.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_issue_blueprint" "api_feature" {
  group_id     = data.gitmoap_group_by_path.acme.group_id
  blueprint_id = "api-feature"
  name         = "API feature"
  base_type    = "story"
  description  = "A change clients of the API can use"

  facets = [
    {
      key      = "docs"
      label    = "Documented"
      category = "documentation"
      hint     = "The OpenAPI spec describes the change"
    },
    {
      key      = "observability"
      label    = "Observable"
      category = "operations"
      hint     = "Dashboards and alerts cover the new endpoints"
      required = false
    },
  ]
}

# Work items follow their merge requests: in review when one opens, done when it merges, and back
# where they were when it's closed without merging.
resource "gitmoap_group_issue_settings" "acme" {
  group_id           = data.gitmoap_group_by_path.acme.group_id
  default_issue_type = "story"
  on_mr_opened       = { to = "in_review" }
  on_mr_merged       = { to = "done" }
  on_mr_closed       = { value = "restore" }
}
```

### Defaults for every repository in an organization

Settings on a group apply to every repository beneath it, so an organization sets its defaults
once. An `enforced` setting can't be relaxed further down; the others can be overridden by a
subgroup or a repository.

```terraform
data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

locals {
  group_id = data.gitmoap_group_by_path.acme.group_id
}

# Merge requests tidy up their branch, and authors may squash.
resource "gitmoap_group_mr_settings" "acme" {
  group_id                     = local.group_id
  merge_method                 = "merge_commit"
  squash_option                = "allow"
  delete_source_branch_default = true
}

# Nobody approves their own change, and a new push asks for a fresh review.
resource "gitmoap_group_approval_policy" "acme" {
  group_id                = local.group_id
  author_can_approve      = false
  committers_can_approve  = true
  last_pusher_can_approve = false
  reset_on_push           = true
  enforcement             = "enforced"
}

# main changes through merge requests: Developers merge, only Maintainers push directly.
resource "gitmoap_group_branch_rule" "main" {
  group_id         = local.group_id
  branch_pattern   = "main"
  allowed_to_push  = "maintainer"
  allowed_to_merge = "developer"
  allow_force_push = false
}

# Only Maintainers cut release tags.
resource "gitmoap_group_tag_rule" "releases" {
  group_id          = local.group_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}

# Large files stay out of git history.
resource "gitmoap_group_push_rules" "acme" {
  group_id         = local.group_id
  max_file_size_mb = 50
}

# Forks stay inside the organization.
resource "gitmoap_group_fork_settings" "acme" {
  group_id         = local.group_id
  forking          = "organization"
  forking_enforced = true
}

# New work items are tasks, and a commit saying "Fixes #12" closes #12.
resource "gitmoap_group_issue_settings" "acme" {
  group_id                = local.group_id
  default_issue_type      = "task"
  close_on_commit_keyword = true
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `group_id` (String) Group ID

### Optional

- `anchor_drift_action` (String) What happens when an anchored region of code changes beyond recognition.
- `auto_close_on_verified` (Boolean)
- `auto_reopen_on_orphaned_close` (Boolean)
- `auto_reopen_on_regression` (Boolean)
- `binding_change_resets_evidence` (Boolean)
- `blocked_by_blocks_close` (Boolean)
- `close_on_commit_keyword` (Boolean)
- `closing_branches` (List of String)
- `default_assignee_from_codeowners` (Boolean)
- `default_issue_type` (String) The kind of work item. The type decides where an item may live (a group or a repository), which
types it may have as children, and which fields it accepts.
- `on_branch_created` (Attributes) Where an automation moves a work item when its event happens: nowhere (`disabled`), back to
where it was (`restore`), or into a status category (`{"to": "in_progress"}`).

Set `value` to `disabled`, `restore`, or set the other fields. (see [below for nested schema](#nestedatt--on_branch_created))
- `on_mr_closed` (Attributes) Where an automation moves a work item when its event happens: nowhere (`disabled`), back to
where it was (`restore`), or into a status category (`{"to": "in_progress"}`).

Set `value` to `disabled`, `restore`, or set the other fields. (see [below for nested schema](#nestedatt--on_mr_closed))
- `on_mr_merged` (Attributes) Where an automation moves a work item when its event happens: nowhere (`disabled`), back to
where it was (`restore`), or into a status category (`{"to": "in_progress"}`).

Set `value` to `disabled`, `restore`, or set the other fields. (see [below for nested schema](#nestedatt--on_mr_merged))
- `on_mr_opened` (Attributes) Where an automation moves a work item when its event happens: nowhere (`disabled`), back to
where it was (`restore`), or into a status category (`{"to": "in_progress"}`).

Set `value` to `disabled`, `restore`, or set the other fields. (see [below for nested schema](#nestedatt--on_mr_opened))
- `override_group_settings` (Boolean)
- `public_issue_filing` (Boolean)
- `require_verified_criteria_to_close` (Boolean)
- `statuses` (Attributes List) (see [below for nested schema](#nestedatt--statuses))
- `todo_ignore_paths` (List of String)
- `todo_policy` (String) What a bare `// TODO:` in a push becomes.
- `todo_promote_type` (String) The kind of work item. The type decides where an item may live (a group or a repository), which
types it may have as children, and which fields it accepts.
- `transitions` (Attributes) Which status categories a work item may move to from each category, as a map from a category to
the categories it may move to. A category not listed may move anywhere. (see [below for nested schema](#nestedatt--transitions))

<a id="nestedatt--on_branch_created"></a>
### Nested Schema for `on_branch_created`

Optional:

- `to` (String) The fixed category behind a status. A status also has a *name*, which the team owns and may
rename freely; the category is the part the platform reasons about. A team can call a status
"Ready for QA" and the platform still knows it is `in_review` — so a board can place it in a
column, a merge can move it, and a progress rollup can count it as not done.
- `value` (String) One of `disabled`, `restore`, instead of the other fields.


<a id="nestedatt--on_mr_closed"></a>
### Nested Schema for `on_mr_closed`

Optional:

- `to` (String) The fixed category behind a status. A status also has a *name*, which the team owns and may
rename freely; the category is the part the platform reasons about. A team can call a status
"Ready for QA" and the platform still knows it is `in_review` — so a board can place it in a
column, a merge can move it, and a progress rollup can count it as not done.
- `value` (String) One of `disabled`, `restore`, instead of the other fields.


<a id="nestedatt--on_mr_merged"></a>
### Nested Schema for `on_mr_merged`

Optional:

- `to` (String) The fixed category behind a status. A status also has a *name*, which the team owns and may
rename freely; the category is the part the platform reasons about. A team can call a status
"Ready for QA" and the platform still knows it is `in_review` — so a board can place it in a
column, a merge can move it, and a progress rollup can count it as not done.
- `value` (String) One of `disabled`, `restore`, instead of the other fields.


<a id="nestedatt--on_mr_opened"></a>
### Nested Schema for `on_mr_opened`

Optional:

- `to` (String) The fixed category behind a status. A status also has a *name*, which the team owns and may
rename freely; the category is the part the platform reasons about. A team can call a status
"Ready for QA" and the platform still knows it is `in_review` — so a board can place it in a
column, a merge can move it, and a progress rollup can count it as not done.
- `value` (String) One of `disabled`, `restore`, instead of the other fields.


<a id="nestedatt--statuses"></a>
### Nested Schema for `statuses`

Optional:

- `archived` (Boolean)
- `category` (String) The fixed category behind a status. A status also has a *name*, which the team owns and may
rename freely; the category is the part the platform reasons about. A team can call a status
"Ready for QA" and the platform still knows it is `in_review` — so a board can place it in a
column, a merge can move it, and a progress rollup can count it as not done.
- `color` (String)
- `is_default` (Boolean) The status new issues land in. Exactly one status in a set carries it, and
it must be open.
- `name` (String)
- `status_id` (String)


<a id="nestedatt--transitions"></a>
### Nested Schema for `transitions`

Optional:

- `table` (String) JSON: set it with `jsonencode()`.

## Import

Import is supported using the following syntax:

The [`terraform import` command](https://developer.hashicorp.com/terraform/cli/commands/import) can be used, for example:

```shell
# Import by its group_id.
terraform import gitmoap_group_issue_settings.example "<group_id>"
```
