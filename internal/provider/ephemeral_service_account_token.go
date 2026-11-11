package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ephemeral.EphemeralResourceWithConfigure = (*ephemeralServiceAccountToken)(nil)
	_ ephemeral.EphemeralResourceWithClose     = (*ephemeralServiceAccountToken)(nil)
)

// ephemeralServiceAccountToken mints a token when Terraform opens it, and revokes it when Terraform
// is done with it, so the token is never in the plan or the state.
type ephemeralServiceAccountToken struct {
	client *Client
}

func newEphemeralServiceAccountToken() ephemeral.EphemeralResource {
	return &ephemeralServiceAccountToken{}
}

type ephemeralTokenModel struct {
	SaID          types.String `tfsdk:"sa_id"`
	Name          types.String `tfsdk:"name"`
	Scopes        types.List   `tfsdk:"scopes"`
	ExpiresInDays types.Int64  `tfsdk:"expires_in_days"`
	Resource      types.Object `tfsdk:"resource"`
	Token         types.String `tfsdk:"token"`
	TokenID       types.String `tfsdk:"token_id"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
}

// What Close needs to revoke the token, kept by Terraform between the two calls.
type ephemeralTokenPrivate struct {
	SaID    string `json:"sa_id"`
	TokenID string `json:"token_id"`
}

func (e *ephemeralServiceAccountToken) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_token"
}

func (e *ephemeralServiceAccountToken) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A short-lived token for a service account, minted when Terraform opens it and revoked when " +
			"Terraform is done with it, so it's never in the plan or the state (Terraform 1.10 and later). Use it where a " +
			"token is needed for the length of a run: to configure a provider, or a write-only argument read during it. " +
			"A token that has to outlive the run, like one a pipeline uses, is `gitmoap_service_account_token`.",
		Attributes: map[string]schema.Attribute{
			"sa_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The service account the token acts as.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "What the token is for, as its list of tokens shows it.",
			},
			"scopes": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The scopes the token carries, such as `repo:read`.",
			},
			"expires_in_days": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Days until the token expires, should revoking it fail. Defaults to 1.",
			},
			"resource": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Restricts the token to one group subtree or repository.",
				Attributes: map[string]schema.Attribute{
					"scope_type": schema.StringAttribute{Required: true, MarkdownDescription: "`group` or `repository`."},
					"scope_id":   schema.StringAttribute{Required: true, MarkdownDescription: "The group's or repository's id."},
				},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The token.",
			},
			"token_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The token's id.",
			},
			"expires_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the token expires, if Terraform doesn't revoke it first.",
			},
		},
	}
}

func (e *ephemeralServiceAccountToken) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	e.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (e *ephemeralServiceAccountToken) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data ephemeralTokenModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if e.client == nil {
		resp.Diagnostics.AddError("Couldn't mint a token", "The provider isn't configured.")
		return
	}
	var scopes []string
	resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	days := int64(1)
	if !data.ExpiresInDays.IsNull() {
		days = data.ExpiresInDays.ValueInt64()
	}
	body := map[string]any{"name": data.Name.ValueString(), "scopes": scopes, "expires_in_days": days}
	if !data.Resource.IsNull() {
		scope := data.Resource.Attributes()
		body["resource"] = map[string]any{
			"scope_type": scope["scope_type"].(types.String).ValueString(),
			"scope_id":   scope["scope_id"].(types.String).ValueString(),
		}
	}

	var created struct {
		Token       string `json:"token"`
		AccessToken struct {
			ID        string  `json:"access_token_id"`
			ExpiresAt *string `json:"expires_at"`
		} `json:"access_token"`
	}
	path := "/service-accounts/" + url.PathEscape(data.SaID.ValueString()) + "/tokens"
	if err := e.client.Do(ctx, http.MethodPost, path, nil, body, &created); err != nil {
		resp.Diagnostics.AddError("Couldn't mint a token", err.Error())
		return
	}

	data.Token = types.StringValue(created.Token)
	data.TokenID = types.StringValue(created.AccessToken.ID)
	data.ExpiresAt = types.StringPointerValue(created.AccessToken.ExpiresAt)
	if data.ExpiresInDays.IsNull() {
		data.ExpiresInDays = types.Int64Value(days)
	}
	if data.Resource.IsNull() {
		data.Resource = types.ObjectNull(map[string]attr.Type{"scope_type": types.StringType, "scope_id": types.StringType})
	}
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)

	private, err := json.Marshal(ephemeralTokenPrivate{SaID: data.SaID.ValueString(), TokenID: created.AccessToken.ID})
	if err != nil {
		resp.Diagnostics.AddError("Couldn't keep the token's id", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "token", private)...)
}

func (e *ephemeralServiceAccountToken) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	stored, diags := req.Private.GetKey(ctx, "token")
	resp.Diagnostics.Append(diags...)
	if stored == nil || diags.HasError() {
		return
	}
	var private ephemeralTokenPrivate
	if err := json.Unmarshal(stored, &private); err != nil {
		resp.Diagnostics.AddError("Couldn't revoke the token", err.Error())
		return
	}
	if e.client == nil {
		resp.Diagnostics.AddError("Couldn't revoke the token", "The provider isn't configured.")
		return
	}
	path := fmt.Sprintf("/service-accounts/%s/tokens/%s", url.PathEscape(private.SaID), url.PathEscape(private.TokenID))
	if err := e.client.Do(ctx, http.MethodDelete, path, nil, nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Couldn't revoke the token", err.Error())
	}
}
