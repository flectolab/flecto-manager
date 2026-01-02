package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/go-playground/validator/v10"
)

func ValidateRedirect(sl validator.StructLevel) {
	redirect := sl.Current().Interface().(commonTypes.Redirect)
	if redirect.Status == "" {
		sl.ReportError(redirect.Status, "Status", "Status", "required", fmt.Sprintf("%s", redirect.Status))
		return
	}

	if redirect.Target == "" {
		sl.ReportError(redirect.Target, "Target", "Target", "required", fmt.Sprintf("%s", redirect.Target))
		return
	}

	if redirect.Type == "" {
		sl.ReportError(redirect.Type, "Type", "Type", "required", fmt.Sprintf("%s", redirect.Type))
		return
	}

	switch redirect.Type {
	case commonTypes.RedirectTypeBasic:
		_, err := url.Parse(redirect.Source)
		if err != nil || !strings.HasPrefix(redirect.Source, "/") {
			sl.ReportError(redirect.Source, "Source", "Source", "invalid path", fmt.Sprintf("%s", redirect.Source))
			return
		}
	case commonTypes.RedirectTypeBasicHost:
		source := redirect.Source
		if !strings.HasPrefix(source, "//") {
			source = "//" + source
		}
		u, err := url.Parse(source)
		if err != nil || u.Host == "" || u.Path == "" {
			sl.ReportError(redirect.Source, "Source", "Source", "invalid path", fmt.Sprintf("%s", redirect.Source))
			return
		}
	case commonTypes.RedirectTypeRegex, commonTypes.RedirectTypeRegexHost:
		_, err := regexp.Compile(redirect.Source)
		if err != nil {
			sl.ReportError(redirect.Source, "Source", "Source", "invalid regex", fmt.Sprintf("%s", redirect.Source))
			return
		}
	}

}
