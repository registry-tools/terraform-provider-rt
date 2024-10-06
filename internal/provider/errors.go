package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/registry-tools/rt-sdk/generated/models"
)

func APIErrorsAsDiagnostics(err error, diags *diag.Diagnostics) {
	if modelError, ok := err.(*models.Errors); ok {
		for _, e := range modelError.GetErrors() {
			diags.AddError(*e.GetTitle(), *e.GetDetail())
		}
	} else {
		diags.AddError("Unknown Error", err.Error())
	}
}

func IsNotFoundError(err error) bool {
	if modelError, ok := err.(*models.Errors); ok {
		return modelError.ResponseStatusCode == 404
	}
	return false
}
