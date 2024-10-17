package provider

import (
	"context"
	"fmt"

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
var _ resource.Resource = &TagPublisherResource{}
var _ resource.ResourceWithImportState = &TagPublisherResource{}

func NewTagPublisherResource() resource.Resource {
	return &TagPublisherResource{}
}

// TagPublisherResource defines the resource implementation.
type TagPublisherResource struct {
	client sdk.SDK
}

// TagPublisherResourceModel describes the resource data model.
type TagPublisherResourceModel struct {
	ID              types.String `tfsdk:"id"`
	VCSConnectorID  types.String `tfsdk:"vcs_connector_id"`
	NamespaceID     types.String `tfsdk:"namespace_id"`
	RepoIdentifier  types.String `tfsdk:"repo_identifier"`
	BackfillPattern types.String `tfsdk:"backfill_pattern"`
}

func (r *TagPublisherResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_publisher"
}

func (r *TagPublisherResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_connector_id": schema.StringAttribute{
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
			"repo_identifier": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"backfill_pattern": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *TagPublisherResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TagPublisherResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Tag Publishers cannot be updated. This is a bug in the provider.")
}

func (r *TagPublisherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TagPublisherResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newTagPublisher := api.NewNamespacesItemTagPublishersPostRequestBody_tagPublisher()
	newTagPublisher.SetVcsConnectorId(data.VCSConnectorID.ValueStringPointer())
	newTagPublisher.SetRepo(data.RepoIdentifier.ValueStringPointer())
	newTagPublisher.SetBackfillPattern(data.BackfillPattern.ValueStringPointer())

	newTagPublisherBody := api.NewNamespacesItemTagPublishersPostRequestBody()
	newTagPublisherBody.SetTagPublisher(newTagPublisher)

	tagPublisher, err := r.client.Api().Namespaces().ByNamespaceId(data.NamespaceID.ValueString()).TagPublishers().PostAsTagPublishersPostResponse(ctx, newTagPublisherBody, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	r.responseToModel(tagPublisher.GetData(), &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagPublisherResource) responseToModel(response models.TagPublisherable, model *TagPublisherResourceModel) {
	model.ID = types.StringPointerValue(response.GetId())

	model.RepoIdentifier = types.StringPointerValue(response.GetRepo())
	model.VCSConnectorID = types.StringPointerValue(response.GetVcsConnectorId())
}

func (r *TagPublisherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TagPublisherResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tagPublisher, err := r.client.Api().TagPublishers().ById(data.ID.ValueString()).GetAsTagPublishersGetResponse(ctx, nil)
	if err != nil {
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}

	r.responseToModel(tagPublisher.GetData(), &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagPublisherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TagPublisherResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Api().TagPublishers().ById(data.ID.ValueString()).Delete(ctx, nil)
	if err != nil {
		if IsNotFoundError(err) {
			req.State.RemoveResource(ctx)
			return
		}
		APIErrorsAsDiagnostics(err, &resp.Diagnostics)
		return
	}
}

func (r *TagPublisherResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
