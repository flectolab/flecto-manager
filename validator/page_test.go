package validator

import (
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidatePage(t *testing.T) {
	validate := validator.New()
	validate.RegisterStructValidation(ValidatePage, commonTypes.Page{})
	tests := []struct {
		name    string
		page    *commonTypes.Page
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "successWithBasic",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/robots.txt",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithBasicHost",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasicHost,
				Path:        "example.com/robots.txt",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithBasicHostAndPort",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasicHost,
				Path:        "example.com:80/robots.txt",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.NoError,
		},
		{
			name: "successWithXML",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/sitemap.xml",
				Content:     "<?xml version=\"1.0\"?><urlset/>",
				ContentType: commonTypes.PageContentTypeXML,
			},
			wantErr: assert.NoError,
		},
		{
			name: "failedContentTypeEmpty",
			page: &commonTypes.Page{
				Type:    commonTypes.PageTypeBasic,
				Path:    "/robots.txt",
				Content: "User-agent: *",
			},
			wantErr: assert.Error,
		},
		{
			name: "failedTypeEmpty",
			page: &commonTypes.Page{
				Path:        "/robots.txt",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedPathInvalidWithBasic",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "robots.txt",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedPathInvalidWithBasicHost",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasicHost,
				Path:        "robots.txt",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.Error,
		},
		{
			name: "failedPathInvalidWithBasicHostNoPath",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasicHost,
				Path:        "example.com",
				Content:     "User-agent: *",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.Error,
		},
		{
			name: "successWithEmptyContent",
			page: &commonTypes.Page{
				Type:        commonTypes.PageTypeBasic,
				Path:        "/robots.txt",
				Content:     "",
				ContentType: commonTypes.PageContentTypeTextPlain,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.page)
			tt.wantErr(t, err, "Page is not valid")
		})
	}
}