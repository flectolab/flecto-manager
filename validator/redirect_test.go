package validator

import (
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidateRedirect(t *testing.T) {
	validate := validator.New()
	validate.RegisterStructValidation(ValidateRedirect, commonTypes.Redirect{})
	tests := []struct {
		name     string
		redirect *commonTypes.Redirect
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "successWithBasic",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "example.com/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithBasicHostAndPort",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "example.com:80/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithRegex",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "^/source/[0-9]+$",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithRegex2",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "/source/[0-9]+",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithRegexWithGroup",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "/source/([0-9]+)",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.NoError,
		},
		{
			name: "failedStatusEmpty",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
			},
			wantErr: assert.Error,
		},
		{
			name: "failedTypeEmpty",
			redirect: &commonTypes.Redirect{
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedTargetEmpty",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceInvalidWithBasic",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceInvalidWithBasicHost",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasicHost,
				Source: "source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedSourceInvalidWithRegex",
			redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "source[",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.redirect)
			tt.wantErr(t, err, "Redirect is not valid")
		})
	}
}
