package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// outboundPolicyOverride serves an organization's outbound policy. Changing it answers with the
// policy inside what the change would refuse, {"policy": {...}, "would_refuse": [...]}, where
// reading it answers with the policy alone. And the policy lists every service with its mode, where
// the configuration may set the mode of some: the others keep theirs, and stay out of the state.
var outboundPolicyOverride = resourceOverride{
	KeyedLists: map[string]string{"services": "service"},
	Schema: func(s *schema.Schema) {
		services, ok := s.Attributes["services"].(schema.ListNestedAttribute)
		if !ok {
			return
		}
		const note = " The state holds only the services listed here."
		services.Description += note
		services.MarkdownDescription += note
		s.Attributes["services"] = services
	},
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		if policy, ok := object["policy"].(map[string]any); ok && operation != "read" {
			replaceWith(object, policy)
		}
		return nil
	},
}
