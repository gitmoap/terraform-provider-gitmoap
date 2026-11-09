package provider

import (
	"context"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// teamMemberOverride: adding a member returns what the call did, {"dry_run", "gained_access",
// "member"}, and the team's member list holds the member itself, with its user as an object, where
// the rest of the API takes user_id. The state keeps the member, filled the same way either way.
var teamMemberOverride = resourceOverride{
	Schema: func(s *schema.Schema) {
		delete(s.Attributes, "dry_run")
		delete(s.Attributes, "gained_access")
	},
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		if operation == "read" {
			object["member"] = maps.Clone(object)
		}
		if user, ok := object["user"].(map[string]any); ok {
			object["user_id"] = user["id"]
		}
		return nil
	},
}
