package validator

import (
	"fmt"
	"net/url"
	"strings"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/go-playground/validator/v10"
)

func ValidatePage(sl validator.StructLevel) {
	page := sl.Current().Interface().(commonTypes.Page)

	if page.ContentType == "" {
		sl.ReportError(page.ContentType, "ContentType", "ContentType", "required", fmt.Sprintf("%s", page.ContentType))
		return
	}

	if page.Type == "" {
		sl.ReportError(page.Type, "Type", "Type", "required", fmt.Sprintf("%s", page.Type))
		return
	}

	switch page.Type {
	case commonTypes.PageTypeBasic:
		_, err := url.Parse(page.Path)
		if err != nil || !strings.HasPrefix(page.Path, "/") {
			sl.ReportError(page.Path, "Path", "Path", "invalid path", fmt.Sprintf("%s", page.Path))
			return
		}
	case commonTypes.PageTypeBasicHost:
		path := page.Path
		if !strings.HasPrefix(path, "//") {
			path = "//" + path
		}
		u, err := url.Parse(path)
		if err != nil || u.Host == "" || u.Path == "" {
			sl.ReportError(page.Path, "Path", "Path", "invalid path", fmt.Sprintf("%s", page.Path))
			return
		}
	}
}
