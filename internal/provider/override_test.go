package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// An archived repository takes no change to its settings: an update whose body holds what the
// repository has already is skipped rather than refused.
func TestUnchanged(t *testing.T) {
	object := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"description": tftypes.String, "stars": tftypes.Number, "purge_on_destroy": tftypes.Bool}}
	state := tftypes.NewValue(object, map[string]tftypes.Value{
		"description":      tftypes.NewValue(tftypes.String, "API"),
		"stars":            tftypes.NewValue(tftypes.Number, 3),
		"purge_on_destroy": tftypes.NewValue(tftypes.Bool, nil),
	})
	call := &apiCall{state: state, Body: map[string]any{"description": "API", "stars": json.Number("3")}}
	if !unchanged(call) {
		t.Errorf("the body holds what the state does: want unchanged")
	}
	call.Body["description"] = "Payments API"
	if unchanged(call) {
		t.Errorf("a new description: want changed")
	}
	call.Body = map[string]any{"purge_on_destroy": true}
	if unchanged(call) {
		t.Errorf("a field the state has no value for: want changed")
	}
}

// A chat subscription's paused is read from its state, and an update sends it.
func TestChatSubscriptionPaused(t *testing.T) {
	for state, want := range map[string]bool{"paused": true, "active": false} {
		object := map[string]any{"state": state}
		if err := chatSubscriptionOverride.Response(nil, "read", object); err != nil || object["paused"] != want {
			t.Errorf("state %s: paused %v, want %v", state, object["paused"], want)
		}
	}
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"paused": tftypes.Bool}}
	call := &apiCall{Operation: "update", Body: map[string]any{}, plan: tftypes.NewValue(typ, map[string]tftypes.Value{"paused": tftypes.NewValue(tftypes.Bool, true)})}
	if err := chatSubscriptionOverride.Request(nil, call); err != nil || call.Body["paused"] != true {
		t.Errorf("update body %v, want paused true", call.Body)
	}
}

// Changing the outbound policy answers with the policy inside what the change would refuse.
func TestOutboundPolicyUnwrapsChanges(t *testing.T) {
	object := map[string]any{"policy": map[string]any{"version": json.Number("4"), "platform_registries": true}, "would_refuse": []any{}}
	if err := outboundPolicyOverride.Response(nil, "update", object); err != nil {
		t.Fatal(err)
	}
	if object["version"] != json.Number("4") || object["would_refuse"] != nil {
		t.Errorf("got %v, want the policy alone", object)
	}
}
