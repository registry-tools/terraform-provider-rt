// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/registry-tools/rt-sdk"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &RegistryToolsProvider{}
var _ provider.ProviderWithFunctions = &RegistryToolsProvider{}

// RegistryToolsProvider defines the provider implementation.
type RegistryToolsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version          string
	testHostOverride string
}

// RegistryToolsProviderModel describes the provider data model.
type RegistryToolsProviderModel struct {
	Hostname types.String `tfsdk:"hostname"`
}

func (p *RegistryToolsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "rt"
	resp.Version = p.version
}

func (p *RegistryToolsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The registry tools hostname. Defaults to registrytools.cloud",
				Optional:            true,
			},
		},
	}
}

func (p *RegistryToolsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RegistryToolsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	hostname := data.Hostname.ValueString()
	if hostname == "" {
		hostname = "registrytools.cloud"
	}

	if p.testHostOverride != "" {
		hostname = p.testHostOverride
	}

	clientID := os.Getenv("REGISTRY_TOOLS_CLIENT_ID")
	if clientID == "" {
		resp.Diagnostics.AddError("Missing Client ID", "The REGISTRY_TOOLS_CLIENT_ID environment variable must be set.")
		return
	}

	clientSecret := os.Getenv("REGISTRY_TOOLS_CLIENT_SECRET")
	if clientSecret == "" {
		resp.Diagnostics.AddError("Missing Client Secret", "The REGISTRY_TOOLS_CLIENT_SECRET environment variable must be set.")
		return
	}

	client, err := sdk.NewSDK(hostname, clientID, clientSecret)
	if err != nil {
		resp.Diagnostics.AddError("Failed to init", fmt.Sprintf("Could not initialize registry tools client: %v", err))
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RegistryToolsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNamespaceResource,
		NewTerraformTokenResource,
		NewVCSConnectorResource,
	}
}

func (p *RegistryToolsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *RegistryToolsProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string, hostOverride string) func() provider.Provider {
	return func() provider.Provider {
		return &RegistryToolsProvider{
			version:          version,
			testHostOverride: hostOverride,
		}
	}
}
