package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultEndpoint = "https://api.gitmoap.com"
	endpointEnv     = "GITMOAP_ENDPOINT"
	tokenEnv        = "GITMOAP_TOKEN"
)

var _ provider.Provider = (*gitmoapProvider)(nil)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &gitmoapProvider{version: version}
	}
}

type gitmoapProvider struct {
	version string
}

type gitmoapProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *gitmoapProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gitmoap"
	resp.Version = p.version
}

func (p *gitmoapProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages gitmoap groups, repositories and their settings.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "Base URL of the gitmoap API. Defaults to the " + endpointEnv + " environment variable, then " + defaultEndpoint + ".",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "A gitmoap access token: personal (gmp_…) or service account (gms_…). Defaults to the " + tokenEnv + " environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *gitmoapProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config gitmoapProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for name, value := range map[string]types.String{"endpoint": config.Endpoint, "token": config.Token} {
		if value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Unknown "+name, "The provider's "+name+" has to be known before apply.")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := firstSet(config.Endpoint.ValueString(), os.Getenv(endpointEnv), defaultEndpoint)
	token := firstSet(config.Token.ValueString(), os.Getenv(tokenEnv))
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing gitmoap token", "Set token in the provider block, or the "+tokenEnv+" environment variable.")
		return
	}

	client := NewClient(endpoint, token, "terraform-provider-gitmoap/"+p.version)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *gitmoapProvider) Resources(ctx context.Context) []func() resource.Resource {
	return resources()
}

func (p *gitmoapProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return dataSources()
}

func firstSet(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
