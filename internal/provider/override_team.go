package provider

import "github.com/hashicorp/terraform-plugin-framework/resource/schema"

// teamOverride: creating a team can take its first members, but reading it doesn't return them
// and updating it can't change them, so a members attribute could only replace the team, and
// with it the access the team holds. Membership goes in gitmoap_team_member instead.
var teamOverride = resourceOverride{
	Schema: func(s *schema.Schema) {
		delete(s.Attributes, "members")
	},
}
