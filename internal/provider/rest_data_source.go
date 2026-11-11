package provider

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// restDataSource reads a data source through the operation in its dataSourceAPI. Every data source
// uses it, with its generated schema; what one data source needs on top goes in its
// dataSourceOverride.
type restDataSource struct {
	name     string
	api      dataSourceAPI
	schema   func(context.Context) schema.Schema
	override dataSourceOverride
	client   *Client
}

var _ datasource.DataSourceWithConfigure = (*restDataSource)(nil)

func newRestDataSource(name string, schemaFunc func(context.Context) schema.Schema) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &restDataSource{name: name, api: dataSourceAPIs[name], schema: schemaFunc, override: dataSourceOverrides[name]}
	}
}

func (d *restDataSource) fields() fields {
	return fields{renames: d.api.Renames, transforms: d.api.Transforms}
}

func (d *restDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.name
}

func (d *restDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := d.schema(ctx)
	if d.override.Schema != nil {
		d.override.Schema(&s)
	}
	resp.Schema = s
}

func (d *restDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *restDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	fail := func(err error) { resp.Diagnostics.AddError("Couldn't read gitmoap_"+d.name, err.Error()) }
	if d.client == nil {
		resp.Diagnostics.AddError("Couldn't read gitmoap_"+d.name, "The provider isn't configured.")
		return
	}
	config := req.Config.Raw
	path, err := fillPath(d.api.Read.Path, d.api.Params, config)
	if err != nil {
		fail(err)
		return
	}
	query := url.Values{}
	for param, attribute := range d.api.Read.Query {
		if text, err := attributeText(config, attribute); err == nil {
			query.Set(param, text) // the rest aren't set
		}
	}
	response, err := d.client.Get(ctx, path, query, d.api.Read.Pages)
	if err != nil {
		fail(err)
		return
	}
	if d.api.Root != "" {
		response = map[string]any{d.api.Root: response}
	}
	object, ok := response.(map[string]any)
	if !ok {
		fail(mismatch("", "an object", response))
		return
	}
	if d.override.Response != nil {
		if err := d.override.Response(ctx, object); err != nil {
			fail(err)
			return
		}
	}
	state, err := fromJSON(config.Type(), object, config, "", d.fields())
	if err != nil {
		fail(err)
		return
	}
	resp.State.Raw = unknownToNull(state)
}
