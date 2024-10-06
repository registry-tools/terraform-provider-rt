package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdk "github.com/registry-tools/rt-sdk"
	"github.com/registry-tools/rt-sdk/generated/api"
	"github.com/registry-tools/rt-sdk/generated/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TerraformTokenResource{}

func NewTerraformTokenResource() resource.Resource {
	return &TerraformTokenResource{}
}

// TerraformTokenResource defines the resource implementation.
type TerraformTokenResource struct {
	client sdk.SDK
}

// TerraformTokenResourceModel describes the resource data model.
type TerraformTokenResourceModel struct {
	Role        types.String `tfsdk:"role"`
	Description types.String `tfsdk:"description"`
	Id          types.String `tfsdk:"id"`
	NamespaceID types.String `tfsdk:"namespace_id"`
	ExpiresIn   types.String `tfsdk:"expires_in"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	Token       types.String `tfsdk:"token"`
}

type TerraformTokenPrivateData struct {
	ServiceAccountID      string `json:"service_account_id"`
	AuthenticationTokenID string `json:"authentication_token_id"`
}

type PrivateData interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

func (r *TerraformTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_terraform_token"
}

func (r *TerraformTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"role": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"namespace_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expires_in": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Computed: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Managed by terraform-provider-rt"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func (r *TerraformTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(sdk.SDK)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *TerraformTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TerraformTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newSA := models.NewServiceAccount()
	name := fmt.Sprintf("managed-sa %s token", data.Role.ValueString())
	newSA.SetName(&name)
	newSA.SetRole(data.Role.ValueStringPointer())

	newSABody := api.NewNamespacesItemServiceAccountsPostRequestBody()
	newSABody.SetServiceAccount(newSA)

	sa, err := r.client.Api().Namespaces().ByNamespaceId(data.NamespaceID.ValueString()).ServiceAccounts().PostAsServiceAccountsPostResponse(ctx, newSABody, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	saID := sa.GetData().GetId()

	newAuthItem := api.NewServiceAccountsItemAuthenticationTokensPostRequestBody_authenticationToken()
	newAuthItem.SetDescription(data.Description.ValueStringPointer())
	newAuthItem.SetExpiresAfter(data.ExpiresIn.ValueStringPointer())

	newAuthBody := api.NewServiceAccountsItemAuthenticationTokensPostRequestBody()
	newAuthBody.SetAuthenticationToken(newAuthItem)

	token, err := r.client.Api().ServiceAccounts().ByServiceAccountId(*saID).AuthenticationTokens().PostAsAuthenticationTokensPostResponse(ctx, newAuthBody, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	privateData := TerraformTokenPrivateData{
		ServiceAccountID:      *saID,
		AuthenticationTokenID: *token.GetData().GetId(),
	}

	privateDataBytes, err := json.Marshal(privateData)
	if err != nil {
		resp.Diagnostics.AddError("Internal error storing private data for this resource", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "sa_token_data", privateDataBytes)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.responseToModel(sa.GetData(), token.GetData(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TerraformTokenResource) responseToModel(responseSA models.ServiceAccountable, responseToken models.AuthenticationTokenable, model *TerraformTokenResourceModel) {
	model.Id = types.StringPointerValue(responseToken.GetId())
	model.Description = types.StringPointerValue(responseToken.GetDescription())
	model.Role = types.StringPointerValue(responseSA.GetRole())
	model.ExpiresAt = types.StringPointerValue(responseToken.GetExpiresAt())

	if token := responseToken.GetToken(); token != nil {
		model.Token = types.StringValue(*token)
	}
}

func (r *TerraformTokenResource) privateData(ctx context.Context, private PrivateData) (TerraformTokenPrivateData, error) {
	var privateData TerraformTokenPrivateData
	privateDataBytes, diags := private.GetKey(ctx, "sa_token_data")
	if diags.HasError() {
		return privateData, errors.New("error getting private state")
	}

	err := json.Unmarshal(privateDataBytes, &privateData)
	if err != nil {
		return privateData, fmt.Errorf("failed to unmarshal private data: %w", err)
	}

	return privateData, nil
}

func (r *TerraformTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TerraformTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	privateData, err := r.privateData(ctx, req.Private)
	if err != nil {
		resp.Diagnostics.AddError("Internal error fetching private data for this resource", err.Error())
		return
	}

	sa, err := r.client.Api().ServiceAccounts().ByServiceAccountId(privateData.ServiceAccountID).GetAsServiceAccountGetResponse(ctx, nil)
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	token, err := r.client.Api().AuthenticationTokens().ByTokenId(privateData.AuthenticationTokenID).GetAsTokenGetResponse(ctx, nil)
	if err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	r.responseToModel(sa.GetData(), token.GetData(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TerraformTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Not updatable
}

func (r *TerraformTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TerraformTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	privateData, err := r.privateData(ctx, req.Private)
	if err != nil {
		resp.Diagnostics.AddError("Internal error fetching private data for this resource", err.Error())
		return
	}

	err = r.client.Api().AuthenticationTokens().ByTokenId(privateData.AuthenticationTokenID).Delete(ctx, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	// If we made it this far, remove the resource
	resp.State.RemoveResource(ctx)

	// Best attempt to delete the service account
	err = r.client.Api().ServiceAccounts().ByServiceAccountId(privateData.ServiceAccountID).Delete(ctx, nil)
	if err != nil {
		// Warn about the service account not being deleted
		apiErrors := err.(*models.Errors)
		for _, err := range apiErrors.GetErrors() {
			resp.Diagnostics.AddWarning("Service account resource could not be deleted", fmt.Sprintf("The autenticaton token was deleted, but the associated service account could not be deleted: %s: %s", *err.GetTitle(), *err.GetDetail()))
		}
	}
}
