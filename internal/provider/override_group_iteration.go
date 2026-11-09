package provider

import (
	"context"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// groupIterationOverride: creating or updating an iteration returns it bare, but reading it returns
// {"iteration": {...}, "rollup": {...}}. The generator kept both shapes, so the schema has the
// iteration's fields twice: at the root, and in an iteration attribute. This keeps the root ones,
// and unwraps the read to fill them.
var groupIterationOverride = resourceOverride{
	Schema: func(s *schema.Schema) {
		delete(s.Attributes, "iteration")
	},
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		iteration, ok := object["iteration"].(map[string]any)
		if operation != "read" || !ok {
			return nil
		}
		delete(object, "iteration")
		maps.Copy(object, iteration)
		return nil
	},
}
