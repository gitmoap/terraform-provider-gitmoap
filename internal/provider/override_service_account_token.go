package provider

import (
	"context"
	"maps"
)

// serviceAccountTokenOverride: the API reads a token back from a list whose items are the
// access_token object that creating it returned, next to its secret. The secret only comes back
// once, so it keeps the value create returned, and so does expires_in_days, which the list holds
// as a date.
var serviceAccountTokenOverride = resourceOverride{
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		if operation != "read" {
			return nil
		}
		item := maps.Clone(object)
		clear(object)
		object["access_token"] = item
		object["name"] = item["name"]
		object["scopes"] = item["scopes"]
		if scope, ok := item["resource_scope_id"]; ok && scope != nil {
			object["resource"] = map[string]any{"scope_id": scope, "scope_type": item["resource_scope_type"]}
		}
		return nil
	},
}
