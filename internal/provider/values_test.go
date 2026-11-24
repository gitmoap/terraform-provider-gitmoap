package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A list filter is sent as its parameter repeated, which the API reads as any of them: a filter
// that couldn't be sent used to be left out without a word, and the data source read everything.
func TestQueryValues(t *testing.T) {
	list := tftypes.List{ElementType: tftypes.String}
	object := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"tier": list, "search": tftypes.String, "orphaned": tftypes.Bool, "none": list, "empty": list,
	}}
	value := tftypes.NewValue(object, map[string]tftypes.Value{
		"tier":     tftypes.NewValue(list, []tftypes.Value{tftypes.NewValue(tftypes.String, "production"), tftypes.NewValue(tftypes.String, "staging")}),
		"search":   tftypes.NewValue(tftypes.String, "api"),
		"orphaned": tftypes.NewValue(tftypes.Bool, true),
		"none":     tftypes.NewValue(list, nil),
		"empty":    tftypes.NewValue(list, []tftypes.Value{}),
	})
	for attribute, want := range map[string][]string{
		"tier":     {"production", "staging"},
		"search":   {"api"},
		"orphaned": {"true"},
		"none":     nil,
		"empty":    {},
	} {
		got, err := queryValues(value, attribute)
		if err != nil {
			t.Errorf("%s: %v", attribute, err)
			continue
		}
		if len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("%s: got %q, want %q", attribute, got, want)
		}
	}
	if _, err := queryValues(value, "missing"); err == nil {
		t.Error("missing: want an error")
	}
}

// 0.1 described a work item status transition table as {"table": {...}}, which no response ever
// held: the API sends the map alone. A state written then decodes with the map type, as the
// framework reads a state at the same schema version, and the next read replaces it.
func TestIssueSettingsTransitionsStateFrom01(t *testing.T) {
	for _, name := range []string{"group_issue_settings", "repository_issue_settings"} {
		var r resource.Resource
		for _, factory := range resources() {
			candidate := factory()
			meta := &resource.MetadataResponse{}
			candidate.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "gitmoap"}, meta)
			if meta.TypeName == "gitmoap_"+name {
				r = candidate
			}
		}
		if r == nil {
			t.Fatalf("no resource gitmoap_%s", name)
		}
		resp := &resource.SchemaResponse{}
		r.Schema(context.Background(), resource.SchemaRequest{}, resp)
		typ := resp.Schema.Type().TerraformType(context.Background())
		for _, state := range []string{`{"transitions": {"table": null}}`, `{"transitions": null}`} {
			value, err := tftypes.ValueFromJSONWithOpts([]byte(state), typ, tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true})
			if err != nil {
				t.Errorf("%s: a 0.1 state %s doesn't decode: %v", name, state, err)
				continue
			}
			converted, err := fromJSON(typ, map[string]any{"transitions": map[string]any{"done": []any{"todo"}}}, value, "", fields{})
			if err != nil {
				t.Errorf("%s: reading transitions after %s: %v", name, state, err)
				continue
			}
			transitions, _ := attributeAt(converted, "transitions")
			got, _, _ := toJSON(transitions, "transitions", fields{})
			if !reflect.DeepEqual(got, map[string]any{"done": []any{"todo"}}) {
				t.Errorf("%s: transitions read as %v", name, got)
			}
		}
	}
}
