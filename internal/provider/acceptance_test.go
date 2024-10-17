package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	sdk "github.com/registry-tools/rt-sdk"
)

func TestAccRTProvider(t *testing.T) {
	rand := time.Now().UnixNano()

	githubToken := os.Getenv("TESTING_GITHUB_TOKEN")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTagPublisherDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testInitialConfig(rand, "5m", githubToken),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rt_terraform_token.this", "role", "provisioner"),
					resource.TestCheckResourceAttr("rt_terraform_token.this", "expires_in", "5m"),
					resource.TestCheckResourceAttrSet("rt_terraform_token.this", "id"),
					resource.TestCheckResourceAttrSet("rt_terraform_token.this", "expires_at"),
					resource.TestCheckResourceAttrSet("rt_tag_publisher.this", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testInitialConfig(rand, "10m", githubToken),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rt_terraform_token.this", "expires_in", "10m"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("rt_terraform_token.this", plancheck.ResourceActionReplace),
						plancheck.ExpectKnownValue("rt_terraform_token.this", tfjsonpath.New("expires_in"), knownvalue.StringExact("10m")),
						plancheck.ExpectUnknownValue("rt_terraform_token.this", tfjsonpath.New("expires_at")),
					},
				},
			},
		}})
}

func testInitialConfig(rand int64, expiration string, githubToken string) string {
	return fmt.Sprintf(`
resource "rt_namespace" "this" {
  name = "default-%[1]d"
  description = "Test namespace"
}

resource "rt_terraform_token" "this" {
  namespace_id = rt_namespace.this.id
  role         = "provisioner"
  expires_in   = "%[2]s"
}

resource "rt_vcs_connector" "this" {
	description = "test github connector"
	github = {
		token = "%[3]s"
	}
}

resource "rt_tag_publisher" "this" {
	vcs_connector_id = rt_vcs_connector.this.id
	repo_identifier = "registry-tools/terraform-rt-private-registry"
	namespace_id = rt_namespace.this.id
}
`, rand, expiration, githubToken)
}

func testSDKClientFromENV() (sdk.SDK, error) {
	hostname := os.Getenv("REGISTRY_TOOLS_HOSTNAME")
	clientID := os.Getenv("REGISTRY_TOOLS_CLIENT_ID")
	if clientID == "" {
		return nil, errors.New("The REGISTRY_TOOLS_CLIENT_ID environment variable must be set.")
	}

	clientSecret := os.Getenv("REGISTRY_TOOLS_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, errors.New("The REGISTRY_TOOLS_CLIENT_SECRET environment variable must be set.")
	}

	client, err := sdk.NewSDK(hostname, clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize registry tools client: %w", err)
	}

	return client, nil
}

func testAccCheckTagPublisherDestroy(state *terraform.State) error {
	sdk, err := testSDKClientFromENV()
	if err != nil {
		return err
	}

	tpResource := state.RootModule().Resources["rt_tag_publisher.this"]

	if tpResource != nil {
		tagPublisherID := tpResource.Primary.ID

		_, err = sdk.Api().TagPublishers().ById(tagPublisherID).GetAsTagPublishersGetResponse(context.TODO(), nil)
		if err == nil {
			return fmt.Errorf("Tag Publisher %s still exists", tagPublisherID)
		}
	}

	return nil
}

func init() {
	resource.AddTestSweepers("rt_tag_publisher", &resource.Sweeper{
		Name: "rt_tag_publisher",
		F: func(region string) error {
			// TODO: Sweep github of any leftover webhooks
			return nil
		},
	})
}
