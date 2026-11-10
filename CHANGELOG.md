## 0.0.2 (November 10, 2026)

DOCUMENTATION:

* An example for every data source, and every resource but the provider connections, whose
  credentials the provider can't set yet. Each example is complete on its own.
* Scenarios: resources that belong together, shown on the page of each one they use. A service
  account whose token goes into a pipeline variable, a catalog service with its environments,
  organization-wide defaults, a protected repository, a team, sprints, a wiki from Markdown files,
  webhooks with generated secrets, a release, custom work item types, single sign-on, and access
  for people.

## 0.0.1 (November 9, 2026)

The first release, generated from the gitmoap API 0.99.0.

FEATURES:

* 61 resources: groups and their settings (merge requests, issues, forks, push rules, approval
  policy), branch and tag rules, repositories and their settings, labels, milestones, iterations
  and cadences, issue blueprints, teams and team members, service accounts and their tokens,
  pipeline and environment variables, webhooks, wikis and wiki pages, branches, tags, releases,
  and the catalog's services, environments and scorecards.
* 162 data sources, covering nearly everything else the API serves to read.
* Import for every resource, by the attributes that identify it.
