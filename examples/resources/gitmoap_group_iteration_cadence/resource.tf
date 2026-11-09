# Two-week sprints starting on Mondays, keeping the next two created.
resource "gitmoap_group_iteration_cadence" "sprints" {
  group_id              = gitmoap_group.platform.group_id
  title                 = "Sprint"
  first_start_date      = "2027-01-04"
  start_day             = "monday"
  duration_weeks        = 2
  iterations_in_advance = 2
}
