package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/go-playground/validator/v10"
)

func containsScheme(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "://")
}

func hasSchemePrefix(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "//")
}

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
		if containsScheme(redirect.Source) || strings.HasPrefix(redirect.Source, "//") {
			sl.ReportError(redirect.Source, "Source", "Source", "source must not contain scheme", fmt.Sprintf("%s", redirect.Source))
			return
		}
		if !hasSchemePrefix(redirect.Target) {
			sl.ReportError(redirect.Target, "Target", "Target", "target must contain scheme", fmt.Sprintf("%s", redirect.Target))
			return
		}
		u, err := url.Parse("//" + redirect.Source)
		if err != nil || u.Host == "" || u.Path == "" {
			sl.ReportError(redirect.Source, "Source", "Source", "invalid path", fmt.Sprintf("%s", redirect.Source))
			return
		}
	case commonTypes.RedirectTypeRegex:
		_, err := regexp.Compile(redirect.Source)
		if err != nil {
			sl.ReportError(redirect.Source, "Source", "Source", "invalid regex", fmt.Sprintf("%s", redirect.Source))
			return
		}
	case commonTypes.RedirectTypeRegexHost:
		if containsScheme(redirect.Source) {
			sl.ReportError(redirect.Source, "Source", "Source", "source must not contain scheme", fmt.Sprintf("%s", redirect.Source))
			return
		}
		if !hasSchemePrefix(redirect.Target) {
			sl.ReportError(redirect.Target, "Target", "Target", "target must contain scheme", fmt.Sprintf("%s", redirect.Target))
			return
		}
		_, err := regexp.Compile(redirect.Source)
		if err != nil {
			sl.ReportError(redirect.Source, "Source", "Source", "invalid regex", fmt.Sprintf("%s", redirect.Source))
			return
		}
	}

}
