package provider

import (
	"context"
	"maps"
)

// forkSettingsOverride serves group and repository fork settings. The API takes each setting as its
// value, plus <setting>_enforced, but returns it with where it came from:
// {"forking": {"value": "organization", "source": "default", "enforced": false}}.
var forkSettingsOverride = resourceOverride{
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		for name, field := range maps.Clone(object) {
			setting, ok := field.(map[string]any)
			if !ok {
				continue
			}
			value, ok := setting["value"]
			if !ok {
				continue
			}
			object[name] = value
			if enforced, ok := setting["enforced"]; ok {
				object[name+"_enforced"] = enforced
			}
		}
		return nil
	},
}
