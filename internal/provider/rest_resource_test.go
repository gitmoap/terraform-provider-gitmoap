package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// An update sends what the plan sets, and fills what it leaves unknown from the state — except an
// attribute only an update takes, and an empty collection, which the API can refuse as they were:
// a provider connection's environment_bound false beside environment_ids [] is a 400.
func TestUpdateLeavesOutUpdateOnlyAndEmptyAttributes(t *testing.T) {
	ids := tftypes.List{ElementType: tftypes.String}
	object := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String, "protected": tftypes.Bool, "bound": tftypes.Bool, "ids": ids, "note": tftypes.String,
	}}
	r := &restResource{name: "thing", api: resourceAPI{
		Params:     map[string]string{"id": "id"},
		Update:     &operation{Method: "PUT", Path: "/things/{id}", Body: map[string]string{"protected": "protected", "bound": "bound", "ids": "ids", "note": "note"}},
		UpdateOnly: []string{"bound"},
	}}
	state := tftypes.NewValue(object, map[string]tftypes.Value{
		"id":        tftypes.NewValue(tftypes.String, "c1"),
		"protected": tftypes.NewValue(tftypes.Bool, false),
		"bound":     tftypes.NewValue(tftypes.Bool, false),
		"ids":       tftypes.NewValue(ids, []tftypes.Value{}),
		"note":      tftypes.NewValue(tftypes.String, "kept"),
	})
	plan := tftypes.NewValue(object, map[string]tftypes.Value{
		"id":        tftypes.NewValue(tftypes.String, "c1"),
		"protected": tftypes.NewValue(tftypes.Bool, true),
		"bound":     tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		"ids":       tftypes.NewValue(ids, tftypes.UnknownValue),
		"note":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	call, err := r.newCall("update", *r.api.Update, state, plan)
	if err != nil {
		t.Fatal(err)
	}
	if call.Path != "/things/c1" {
		t.Errorf("path %q", call.Path)
	}
	if got := call.Body["protected"]; got != true {
		t.Errorf("protected: the plan sets it, so it is sent; got %v", got)
	}
	if got := call.Body["note"]; got != "kept" {
		t.Errorf("note: unknown in the plan, so the state's value is sent back; got %v", got)
	}
	for _, field := range []string{"bound", "ids"} {
		if got, sent := call.Body[field]; sent {
			t.Errorf("%s: sent as %v, want left out", field, got)
		}
	}
}

// The same attributes, set in the plan, are sent: only what the plan leaves unknown is left out.
func TestUpdateSendsUpdateOnlyAndEmptyAttributesThePlanSets(t *testing.T) {
	ids := tftypes.List{ElementType: tftypes.String}
	object := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String, "bound": tftypes.Bool, "ids": ids}}
	r := &restResource{name: "thing", api: resourceAPI{
		Params:     map[string]string{"id": "id"},
		Update:     &operation{Method: "PUT", Path: "/things/{id}", Body: map[string]string{"bound": "bound", "ids": "ids"}},
		UpdateOnly: []string{"bound"},
	}}
	value := tftypes.NewValue(object, map[string]tftypes.Value{
		"id":    tftypes.NewValue(tftypes.String, "c1"),
		"bound": tftypes.NewValue(tftypes.Bool, true),
		"ids":   tftypes.NewValue(ids, []tftypes.Value{tftypes.NewValue(tftypes.String, "env-1")}),
	})
	call, err := r.newCall("update", *r.api.Update, value, value)
	if err != nil {
		t.Fatal(err)
	}
	if call.Body["bound"] != true {
		t.Errorf("bound: set in the plan, so sent; body %v", call.Body)
	}
	if _, sent := call.Body["ids"]; !sent {
		t.Errorf("ids: set in the plan, so sent; body %v", call.Body)
	}
}

func TestIsEmptyCollection(t *testing.T) {
	list := tftypes.List{ElementType: tftypes.String}
	mapType := tftypes.Map{ElementType: tftypes.String}
	for _, c := range []struct {
		name  string
		value tftypes.Value
		want  bool
	}{
		{"empty list", tftypes.NewValue(list, []tftypes.Value{}), true},
		{"list", tftypes.NewValue(list, []tftypes.Value{tftypes.NewValue(tftypes.String, "a")}), false},
		{"empty map", tftypes.NewValue(mapType, map[string]tftypes.Value{}), true},
		{"null list", tftypes.NewValue(list, nil), false},
		{"unknown list", tftypes.NewValue(list, tftypes.UnknownValue), false},
		{"string", tftypes.NewValue(tftypes.String, ""), false},
	} {
		if got := isEmptyCollection(c.value); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
