package provider

import "context"

// approvalPolicyOverride serves group and repository approval policies. The API takes the policy's
// fields flat, but returns them in policy, next to the effective policy and the rules:
// {"policy": {"author_can_approve": false, ...}, "effective": {...}, "rules": [...]}.
var approvalPolicyOverride = resourceOverride{
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		policy, _ := object["policy"].(map[string]any)
		for name, value := range policy {
			if _, taken := object[name]; !taken {
				object[name] = value
			}
		}
		return nil
	},
}
