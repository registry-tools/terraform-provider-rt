// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTerraformTokenResource(t *testing.T) {
	rand := time.Now().UnixNano()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTerraformTokenResourceConfig(rand, "5m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rt_terraform_token.this", "role", "publisher"),
					resource.TestCheckResourceAttr("rt_terraform_token.this", "expires_in", "5m"),
					resource.TestCheckResourceAttrSet("rt_terraform_token.this", "id"),
					resource.TestCheckResourceAttrSet("rt_terraform_token.this", "expires_at"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTerraformTokenResourceConfig(rand, "10m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rt_terraform_token.this", "expires_in", "10m"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTerraformTokenResourceConfig(rand int64, expiration string) string {
	return fmt.Sprintf(`
resource "rt_namespace" "this" {
  name = "default-%[1]d"
  description = "Test namespace"
}

resource "rt_terraform_token" "this" {
  namespace_id = rt_namespace.this.id
  role         = "publisher"
  expires_in   = "%[2]s"
}
`, rand, expiration)
}
