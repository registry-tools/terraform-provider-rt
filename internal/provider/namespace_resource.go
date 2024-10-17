package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdk "github.com/registry-tools/rt-sdk"
	"github.com/registry-tools/rt-sdk/generated/api"
	"github.com/registry-tools/rt-sdk/generated/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NamespaceResource{}
var _ resource.ResourceWithImportState = &NamespaceResource{}

func NewNamespaceResource() resource.Resource {
	return &NamespaceResource{}
}

// NamespaceResource defines the resource implementation.
type NamespaceResource struct {
	client sdk.SDK
}

// NamespaceResourceModel describes the resource data model.
type NamespaceResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *NamespaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_namespace"
}

func (r *NamespaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *NamespaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NamespaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NamespaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newNamespace := models.NewNamespace()
	newNamespace.SetName(data.Name.ValueStringPointer())

	description := ""
	if !data.Description.IsNull() {
		description = data.Description.ValueString()
	}
	newNamespace.SetDescription(&description)

	newNamespaceBody := api.NewNamespacesPostRequestBody()
	newNamespaceBody.SetNamespace(newNamespace)

	namespace, err := r.client.Api().Namespaces().PostAsNamespacesPostResponse(ctx, newNamespaceBody, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	r.responseToModel(namespace.GetData(), &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NamespaceResource) responseToModel(response models.Namespaceable, model *NamespaceResourceModel) {
	model.ID = types.StringPointerValue(response.GetId())
	model.Name = types.StringPointerValue(response.GetName())
	model.CreatedAt = types.StringValue(response.GetCreatedAt().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(response.GetUpdatedAt().Format(time.RFC3339))

	description := response.GetDescription()
	if *description != "" {
		model.Description = types.StringPointerValue(description)
	}
}

func (r *NamespaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NamespaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	namespace, err := r.client.Api().Namespaces().ByNamespaceId(data.ID.ValueString()).GetAsNamespaceGetResponse(ctx, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	r.responseToModel(namespace.GetData(), &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NamespaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NamespaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateNamespace := models.NewNamespace()
	updateNamespace.SetName(data.Name.ValueStringPointer())

	description := ""
	if !data.Description.IsNull() {
		description = data.Description.ValueString()
	}
	updateNamespace.SetDescription(&description)

	updateNamespaceBody := api.NewNamespacesPostRequestBody()
	updateNamespaceBody.SetNamespace(updateNamespace)

	namespace, err := r.client.Api().Namespaces().ByNamespaceId(data.ID.ValueString()).PatchAsNamespacePatchResponse(ctx, updateNamespaceBody, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	r.responseToModel(namespace.GetData(), &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NamespaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NamespaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Api().Namespaces().ByNamespaceId(data.ID.ValueString()).Delete(ctx, nil)
	if err != nil {
		if IsNotFoundError(err) {
			req.State.RemoveResource(ctx)
			return
		}
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}
}

func (r *NamespaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
