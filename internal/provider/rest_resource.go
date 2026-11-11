package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// restResource manages a resource through the operations in its resourceAPI. Every resource uses
// it, with its generated schema; what one resource needs on top goes in its resourceOverride.
type restResource struct {
	name     string
	api      resourceAPI
	schema   func(context.Context) schema.Schema
	override resourceOverride
	client   *Client
}

var (
	_ resource.ResourceWithConfigure   = (*restResource)(nil)
	_ resource.ResourceWithImportState = (*restResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*restResource)(nil)
)

func newRestResource(name string, schemaFunc func(context.Context) schema.Schema) func() resource.Resource {
	return func() resource.Resource {
		return &restResource{name: name, api: resourceAPIs[name], schema: schemaFunc, override: resourceOverrides[name]}
	}
}

func (r *restResource) fields() fields {
	return fields{renames: r.api.Renames, transforms: r.api.Transforms}
}

func (r *restResource) typeName() string {
	return "gitmoap_" + r.name
}

func (r *restResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.name
}

func (r *restResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := r.schema(ctx)
	if err := addWriteOnly(&s, r.api.WriteOnly); err != nil {
		resp.Diagnostics.AddError("Invalid schema for "+r.typeName(), err.Error())
	}
	if r.override.Schema != nil {
		r.override.Schema(&s)
	}
	resp.Schema = s
}

func (r *restResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

// ModifyPlan keeps what can't change from becoming unknown, and forces replacement when an
// attribute only create can set changes. One left out of the configuration keeps the value it has,
// since an update couldn't change it: otherwise any other change would leave it unknown, and
// replace the object.
func (r *restResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return // creating or destroying
	}
	planned := req.Plan.Raw
	// What never changes once the object exists keeps its value rather than becoming unknown with
	// any other change, which would also replace whatever refers to it, like a group's group_id.
	for _, name := range append([]string{"created_at", "created_by"}, r.api.Identity...) {
		configured, _ := attributeAt(req.Config.Raw, name)
		value, exists := attributeAt(planned, name)
		if exists && configured.IsNull() && !value.IsKnown() {
			prior, _ := attributeAt(req.State.Raw, name)
			planned = withAttribute(planned, name, prior)
		}
	}
	for _, name := range r.api.Replace {
		if slices.Contains(r.override.Updatable, name) {
			continue
		}
		prior, _ := attributeAt(req.State.Raw, name)
		if configured, _ := attributeAt(req.Config.Raw, name); configured.IsNull() {
			planned = withAttribute(planned, name, prior)
			continue
		}
		value, _ := attributeAt(planned, name)
		if value.IsKnown() && !value.IsFullyKnown() {
			// Its computed parts are unknown only because something else changes: they can't
			// change without replacing it either.
			value = fillUnknown(value, prior)
			planned = withAttribute(planned, name, value)
		}
		if !value.Equal(prior) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root(name))
		}
	}
	resp.Plan.Raw = planned
}

func (r *restResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	planned := req.Plan.Raw
	values := withWriteOnly(planned, req.Config.Raw, r.api.WriteOnly)
	call, err := r.newCall("create", r.api.Create, planned, values)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't create "+r.typeName(), err.Error())
		return
	}
	call.plan = planned
	created, err := r.send(ctx, call)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't create "+r.typeName(), err.Error())
		return
	}
	state, diags := r.settle(ctx, "create", planned, created)
	resp.Diagnostics.Append(diags...)
	resp.State.Raw = state
	if diags.HasError() || !r.setsUpdateOnly(planned) {
		return
	}

	// What only an update takes, it sets now.
	call, err = r.newCall("update", *r.api.Update, state, values)
	if err != nil {
		resp.Diagnostics.AddError(r.typeName()+" was created, but setting "+strings.Join(r.api.UpdateOnly, ", ")+" failed", err.Error())
		return
	}
	call.plan, call.state = planned, state
	updated, err := r.send(ctx, call)
	if err != nil {
		resp.Diagnostics.AddError(r.typeName()+" was created, but setting "+strings.Join(r.api.UpdateOnly, ", ")+" failed", err.Error())
		return
	}
	state, diags = r.settle(ctx, "update", planned, updated)
	resp.Diagnostics.Append(diags...)
	resp.State.Raw = state
}

// setsUpdateOnly reports whether the plan sets an attribute only the update operation takes.
func (r *restResource) setsUpdateOnly(planned tftypes.Value) bool {
	if r.api.Update == nil {
		return false
	}
	for _, name := range r.api.UpdateOnly {
		if value, ok := attributeAt(planned, name); ok && value.IsKnown() && !value.IsNull() {
			return true
		}
	}
	return false
}

func (r *restResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	object, found, err := r.fetch(ctx, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read "+r.typeName(), err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state, err := fromJSON(req.State.Raw.Type(), object, req.State.Raw, "", r.fields())
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read "+r.typeName(), err.Error())
		return
	}
	resp.State.Raw = unknownToNull(state)
}

func (r *restResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	planned := req.Plan.Raw
	if r.api.Update == nil {
		// Everything the API takes forces replacement, so only computed values can differ.
		resp.State.Raw = unknownToNull(fillUnknown(planned, req.State.Raw))
		return
	}
	// The path takes the identity the object has now: an update can change it.
	call, err := r.newCall("update", *r.api.Update, req.State.Raw, withWriteOnly(planned, req.Config.Raw, r.api.WriteOnly))
	if err != nil {
		resp.Diagnostics.AddError("Couldn't update "+r.typeName(), err.Error())
		return
	}
	call.plan, call.state = planned, req.State.Raw
	updated, err := r.send(ctx, call)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't update "+r.typeName(), err.Error())
		return
	}
	state, diags := r.settle(ctx, "update", planned, updated)
	resp.Diagnostics.Append(diags...)
	resp.State.Raw = state
}

func (r *restResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.api.Delete == nil {
		resp.Diagnostics.AddWarning(
			r.typeName()+" can't be deleted through the API",
			"Terraform no longer manages it, but it's still in gitmoap.",
		)
		return
	}
	call, err := r.newCall("delete", *r.api.Delete, req.State.Raw, tftypes.Value{})
	if err != nil {
		resp.Diagnostics.AddError("Couldn't delete "+r.typeName(), err.Error())
		return
	}
	call.state = req.State.Raw
	if _, err := r.send(ctx, call); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Couldn't delete "+r.typeName(), err.Error())
	}
}

// ImportState takes the identity attributes joined with slashes, like group_id/key for a label.
// The last one may hold slashes itself.
func (r *restResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", len(r.api.Identity))
	if len(r.api.Identity) == 0 || len(parts) != len(r.api.Identity) || slices.Contains(parts, "") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got %q.", strings.Join(r.api.Identity, "/"), req.ID))
		return
	}
	for index, attribute := range r.api.Identity {
		at := attributePath(attribute)
		attributeType, diags := resp.State.Schema.TypeAtPath(ctx, at)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		switch {
		case attributeType.Equal(types.StringType):
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, at, parts[index])...)
		case attributeType.Equal(types.Int64Type):
			number, err := strconv.ParseInt(parts[index], 10, 64)
			if err != nil {
				resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("%s has to be a whole number, got %q.", attribute, parts[index]))
				return
			}
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, at, number)...)
		default:
			resp.Diagnostics.AddError("Can't import "+r.typeName(), fmt.Sprintf("%s is a %s, which import can't set.", attribute, attributeType))
		}
	}
}

// settle works out the state after a create or update: what the plan set, and the rest from the
// API. It reads the object back, because what an operation returns often holds less than a read.
func (r *restResource) settle(ctx context.Context, operation string, planned tftypes.Value, response any) (tftypes.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	state := planned
	if object, ok := response.(map[string]any); ok {
		if r.override.Response != nil {
			if err := r.override.Response(ctx, operation, object); err != nil {
				diags.AddError("Couldn't "+operation+" "+r.typeName(), err.Error())
				return unknownToNull(state), diags
			}
		}
		converted, err := fromJSON(planned.Type(), object, planned, "", r.fields())
		if err != nil {
			diags.AddError("Couldn't "+operation+" "+r.typeName(), err.Error())
			return unknownToNull(state), diags
		}
		state = fillUnknown(planned, converted)
	}

	object, found, err := r.fetch(ctx, state)
	switch {
	case err != nil:
		diags.AddError(r.typeName()+" was "+operation+"d, but reading it back failed", err.Error())
	case !found:
		diags.AddError(r.typeName()+" was "+operation+"d, but reading it back didn't find it", "The API may need a moment, or the provider is looking in the wrong place.")
	default:
		converted, err := fromJSON(planned.Type(), object, state, "", r.fields())
		if err != nil {
			diags.AddError(r.typeName()+" was "+operation+"d, but reading it back failed", err.Error())
			break
		}
		state = fillUnknown(planned, converted)
	}
	return unknownToNull(state), diags
}

// fetch reads the object state identifies. found is false when the API doesn't have it any more.
func (r *restResource) fetch(ctx context.Context, state tftypes.Value) (object map[string]any, found bool, err error) {
	call, err := r.newCall("read", r.api.Read, state, tftypes.Value{})
	if err != nil {
		return nil, false, err
	}
	call.state = state
	response, err := r.send(ctx, call)
	if IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if r.api.Match != nil {
		object, err = r.findItem(ctx, response, state)
		return object, object != nil, err
	}
	if object, found = response.(map[string]any); !found {
		return nil, false, mismatch("", "an object", response)
	}
	if r.override.Response != nil {
		if err := r.override.Response(ctx, "read", object); err != nil {
			return nil, false, err
		}
	}
	return object, true, nil
}

// findItem returns the item of a list response that's the object state identifies, or nil. The
// Response override sees each item before it's matched.
func (r *restResource) findItem(ctx context.Context, response any, state tftypes.Value) (map[string]any, error) {
	items, ok := listItems(response, r.api.Read.Items)
	if !ok {
		return nil, mismatch("", "a list", response)
	}
	want := map[string]string{}
	for field, attribute := range r.api.Match {
		text, err := attributeText(state, attribute)
		if err != nil {
			return nil, err
		}
		want[field] = text
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		if r.override.Response != nil {
			if err := r.override.Response(ctx, "read", item); err != nil {
				return nil, err
			}
		}
		if matchesItem(item, want) {
			return item, nil
		}
	}
	return nil, nil
}

// newCall prepares a call: its path parameters come from identity, its body from values. On an
// update, identity is the prior state, and what values leave unknown it takes from there: an
// unconfigured field keeps its value anyway, and some fields have to be sent back, like a version.
func (r *restResource) newCall(name string, op operation, identity, values tftypes.Value) (*apiCall, error) {
	path, err := fillPath(op.Path, r.api.Params, identity)
	if err != nil {
		return nil, err
	}
	call := &apiCall{Operation: name, Method: op.Method, Path: path, Query: url.Values{}}
	if name == "create" || name == "update" {
		call.Body = map[string]any{}
		for attribute, field := range op.Body {
			value, _ := attributeAt(values, attribute)
			if name == "update" && !value.IsKnown() {
				// An attribute only an update takes is a change to make, not a value to keep:
				// left out of the configuration, it isn't sent at all. Sent back as it was, it can
				// contradict a field beside it — a provider connection's environment_bound false
				// beside its environment_ids is refused.
				if slices.Contains(r.api.UpdateOnly, attribute) {
					continue
				}
				value, _ = attributeAt(identity, attribute)
				// Leaving an empty collection out keeps it empty as surely as sending it back,
				// and an API may refuse the empty form outright, as environment_ids does.
				if isEmptyCollection(value) {
					continue
				}
			}
			converted, ok, err := toJSON(value, attribute, r.fields())
			if err != nil {
				return nil, err
			}
			if ok {
				call.Body[field] = converted
			}
		}
	}
	return call, nil
}

func (r *restResource) send(ctx context.Context, call *apiCall) (any, error) {
	if r.client == nil {
		return nil, fmt.Errorf("the provider isn't configured")
	}
	if r.override.Request != nil {
		if err := r.override.Request(ctx, call); err != nil {
			return nil, err
		}
	}
	call.Client = r.client
	send := func() (any, error) {
		if call.Operation == "read" {
			return r.client.Get(ctx, call.Path, call.Query, r.api.Read.Pages)
		}
		var body any
		if call.Body != nil {
			body = call.Body
		}
		var out any
		err := r.client.Do(ctx, call.Method, call.Path, call.Query, body, &out)
		return out, err
	}
	if r.override.Send != nil {
		return r.override.Send(ctx, call, send)
	}
	return send()
}

var pathParameter = regexp.MustCompile(`\{([^}]+)\}`)

// fillPath fills in a path's parameters from the attributes behind them.
func fillPath(template string, params map[string]string, values tftypes.Value) (string, error) {
	var err error
	path := pathParameter.ReplaceAllStringFunc(template, func(segment string) string {
		name := segment[1 : len(segment)-1]
		attribute, ok := params[name]
		if !ok {
			err = fmt.Errorf("no attribute holds path parameter %s", name)
			return segment
		}
		text, textErr := attributeText(values, attribute)
		if textErr != nil {
			err = fmt.Errorf("path parameter %s: %w", name, textErr)
			return segment
		}
		return url.PathEscape(text)
	})
	return path, err
}

func attributePath(dotted string) path.Path {
	names := strings.Split(dotted, ".")
	at := path.Root(names[0])
	for _, name := range names[1:] {
		at = at.AtName(name)
	}
	return at
}
