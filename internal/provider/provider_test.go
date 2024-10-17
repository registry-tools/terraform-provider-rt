package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProvider provider.Provider = New("test")()

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"rt": providerserver.NewProtocol6WithError(testAccProvider),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("REGISTRY_TOOLS_HOSTNAME") == "" {
		t.Fatal("REGISTRY_TOOLS_HOSTNAME must be set for acceptance tests")
	}

	if os.Getenv("TESTING_GITHUB_TOKEN") == "" {
		t.Fatal("TESTING_GITHUB_TOKEN must be set for acceptance tests")
	}
}
