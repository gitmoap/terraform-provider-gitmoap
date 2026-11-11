package provider

import (
	"context"
	"net/url"

	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// What a resource or data source needs beyond the shared implementation, by name. Each lives in
// its own override_*.go file. TestOverrides fails when one no longer matches what's generated.
var (
	resourceOverrides = map[string]resourceOverride{
		"group_access_grant":         accessGrantOverride,
		"group_approval_policy":      approvalPolicyOverride,
		"group_fork_settings":        forkSettingsOverride,
		"group_iteration":            groupIterationOverride,
		"group_label":                labelOverride,
		"group_wiki_page":            wikiPageOverride,
		"repository_access_grant":    accessGrantOverride,
		"repository_approval_policy": approvalPolicyOverride,
		"repository_fork_settings":   forkSettingsOverride,
		"repository_label":           labelOverride,
		"repository_wiki_page":       wikiPageOverride,
		"service_account_token":      serviceAccountTokenOverride,
		"team":                       teamOverride,
		"team_member":                teamMemberOverride,
	}
	dataSourceOverrides = map[string]dataSourceOverride{}
)

// resourceOverride changes how restResource handles one resource. Every field is optional.
type resourceOverride struct {
	// Updatable lists attributes the update operation can change though its body doesn't take
	// them, so they shouldn't force replacement. Request then has to send them.
	Updatable []string
	// Schema changes the schema, after the shared implementation has.
	Schema func(s *schema.Schema)
	// Request changes a call before it's sent.
	Request func(ctx context.Context, call *apiCall) error
	// Send wraps sending a call, after Request: to lock, read something first, or retry.
	Send func(ctx context.Context, call *apiCall, send func() (any, error)) (any, error)
	// Response changes what the API returned before it goes into the state: the object a create,
	// update or read returned, or for a list read, each item before it's matched.
	Response func(ctx context.Context, operation string, object map[string]any) error
}

// dataSourceOverride changes how restDataSource handles one data source. Every field is optional.
type dataSourceOverride struct {
	Schema   func(s *dsschema.Schema)
	Response func(ctx context.Context, object map[string]any) error
}

// apiCall is a request the shared implementation is about to send.
type apiCall struct {
	Operation string // create, read, update or delete
	Method    string
	Path      string // with its parameters filled in
	Query     url.Values
	Body      map[string]any // the JSON body of a create or update
	Client    *Client        // for an override that has to call the API too

	plan  tftypes.Value // what Terraform plans, on create and update
	state tftypes.Value // the state before the call, on read, update and delete
}

// Planned returns the planned value of an attribute, as JSON would carry it, and whether it has one.
func (c *apiCall) Planned(path string) (any, bool) {
	return jsonAt(c.plan, path)
}

// Prior returns the value an attribute had before the call, as JSON would carry it, and whether it
// had one.
func (c *apiCall) Prior(path string) (any, bool) {
	return jsonAt(c.state, path)
}

func jsonAt(value tftypes.Value, path string) (any, bool) {
	attribute, ok := attributeAt(value, path)
	if !ok {
		return nil, false
	}
	converted, set, err := toJSON(attribute, path, fields{})
	return converted, set && err == nil
}
