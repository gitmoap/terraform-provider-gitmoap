package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// chatSubscriptionOverride serves group and repository channel subscriptions. An update pauses or
// resumes one with paused, which a read answers as its state, "active" or "paused": the provider
// adds paused, read back from the state. Creating takes no paused, so a subscription created
// paused is paused by an update right after.
var chatSubscriptionOverride = resourceOverride{
	UpdateOnly: []string{"paused"},
	Schema: func(s *schema.Schema) {
		const description = "Whether the channel stops hearing events for now, keeping the subscription. gitmoap pauses a subscription by itself when it can't post to the channel any more: paused_reason says why."
		s.Attributes["paused"] = schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Description:         description,
			MarkdownDescription: "Whether the channel stops hearing events for now, keeping the subscription. gitmoap pauses a subscription by itself when it can't post to the channel any more: `paused_reason` says why.",
		}
	},
	Request: func(ctx context.Context, call *apiCall) error {
		if call.Operation != "update" {
			return nil
		}
		if paused, ok := call.Planned("paused"); ok {
			call.Body["paused"] = paused
		}
		return nil
	},
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		if state, ok := object["state"].(string); ok {
			object["paused"] = state == "paused"
		}
		return nil
	},
}
