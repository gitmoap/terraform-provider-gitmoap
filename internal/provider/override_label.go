package provider

import "context"

// labelOverride serves group and repository labels. Their update path names a label by its current
// key, and the update body renames it with new_key, so changing key updates the label in place:
// without this, it would delete the label and create another.
var labelOverride = resourceOverride{
	Updatable: []string{"key"},
	Request: func(ctx context.Context, call *apiCall) error {
		if call.Operation != "update" {
			return nil
		}
		planned, _ := call.Planned("key")
		prior, _ := call.Prior("key")
		if planned != prior {
			call.Body["new_key"] = planned
		}
		return nil
	},
}
