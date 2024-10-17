package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdk "github.com/registry-tools/rt-sdk"
	"github.com/registry-tools/rt-sdk/generated/api"
	"github.com/registry-tools/rt-sdk/generated/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VCSConnectorResource{}
var _ resource.ResourceWithImportState = &VCSConnectorResource{}

func NewVCSConnectorResource() resource.Resource {
	return &VCSConnectorResource{}
}

// VCSConnectorResource defines the resource implementation.
type VCSConnectorResource struct {
	client sdk.SDK
}

// VCSConnectorResourceModel describes the resource data model.
type VCSConnectorResourceModel struct {
	ID          types.String `tfsdk:"id"`
	GitHub      types.Object `tfsdk:"github"`
	Description types.String `tfsdk:"description"`
}

func (r *VCSConnectorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vcs_connector"
}

func (r *VCSConnectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"description": schema.StringAttribute{
				Optional: true,
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
			"github": schema.ObjectAttribute{
				AttributeTypes: map[string]attr.Type{
					"token": types.StringType,
				},
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *VCSConnectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VCSConnectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "VCS Connectors cannot be updated. This is a bug in the provider.")
}

func (r *VCSConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VCSConnectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	token := data.GitHub.Attributes()["token"].(types.String).ValueString()

	newGitHubConnector := api.NewVcsConnectorsPostRequestBody_githubConnector()
	newGitHubConnector.SetDescription(data.Description.ValueStringPointer())
	newGitHubConnector.SetToken(&token)

	newVCSConnectorBody := api.NewVcsConnectorsPostRequestBody()
	newVCSConnectorBody.SetGithubConnector(newGitHubConnector)

	vcsConnector, err := r.client.Api().VcsConnectors().PostAsVcsConnectorsPostResponse(ctx, newVCSConnectorBody, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	r.responseToModel(vcsConnector.GetData(), &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VCSConnectorResource) responseToModel(response models.VCSConnectorable, model *VCSConnectorResourceModel) {
	model.ID = types.StringPointerValue(response.GetId())

	description := response.GetDescription()
	if *description != "" {
		model.Description = types.StringPointerValue(description)
	}
}

func (r *VCSConnectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Reads only from state
	var data VCSConnectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
}

func (r *VCSConnectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VCSConnectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Api().VcsConnectors().ByVcsConnectorId(data.ID.ValueString()).Delete(ctx, nil)
	if err != nil {
		if IsNotFoundError(err) {
			req.State.RemoveResource(ctx)
			return
		}
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}
}

func (r *VCSConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
