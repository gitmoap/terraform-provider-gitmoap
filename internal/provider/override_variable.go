package provider

import "github.com/hashicorp/terraform-plugin-framework/resource/schema"

// variableOverride serves pipeline and environment variables. secret only says whether the value
// is one: the value itself has to stay out of plans. The API never returns a secret value, so
// the state keeps the one in the configuration.
var variableOverride = resourceOverride{
	Schema: func(s *schema.Schema) {
		value := s.Attributes["value"].(schema.StringAttribute)
		value.Sensitive = true
		s.Attributes["value"] = value
	},
}
