package cdn

import (
	"fmt"
	"strings"
)

// FieldKind drives both the widget the admin UI renders and the validation the
// server applies.
type FieldKind string

const (
	FieldKindText       FieldKind = "TEXT"
	FieldKindSecret     FieldKind = "SECRET"
	FieldKindBool       FieldKind = "BOOL"
	FieldKindInt        FieldKind = "INT"
	FieldKindSelect     FieldKind = "SELECT"
	FieldKindStringList FieldKind = "STRING_LIST"
)

// FieldHosts is the conventional name of the field listing the hostnames a
// configuration serves. It is not part of Descriptor because not every provider
// needs it - CloudFront derives its hosts from the distribution - but a provider
// addressing its cache by URL declares it under this name, and the planner is
// built from it.
const FieldHosts = "hosts"

// Option is one choice of a FieldKindSelect field.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Field describes one configuration input of a provider.
type Field struct {
	Name        string    `json:"name"`
	Label       string    `json:"label"`
	Kind        FieldKind `json:"kind"`
	Required    bool      `json:"required"`
	Help        string    `json:"help,omitempty"`
	Placeholder string    `json:"placeholder,omitempty"`
	Options     []Option  `json:"options,omitempty"`
	Default     any       `json:"default,omitempty"`
}

// IsSecret reports whether the value must be stored encrypted and never returned
// to a client.
func (f Field) IsSecret() bool {
	return f.Kind == FieldKindSecret
}

// Descriptor is everything the rest of the application needs to know about a
// provider without importing it: how to configure it, and what it can do.
type Descriptor struct {
	Type         string       `json:"type"`
	Label        string       `json:"label"`
	Doc          string       `json:"doc,omitempty"`
	Fields       []Field      `json:"fields"`
	Capabilities Capabilities `json:"capabilities"`
}

// Field returns the field with this name.
func (d Descriptor) Field(name string) (Field, bool) {
	for _, field := range d.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return Field{}, false
}

// Validate checks a submitted configuration: no unknown field, every required
// field filled, and every value consistent with its kind.
//
// It returns all the problems at once rather than the first one, because the
// caller renders them next to the fields of a form.
func (d Descriptor) Validate(values Values) error {
	var problems []string

	for name := range values {
		if _, ok := d.Field(name); !ok {
			problems = append(problems, fmt.Sprintf("unknown field %q", name))
		}
	}

	for _, field := range d.Fields {
		if problem := field.validate(values); problem != "" {
			problems = append(problems, problem)
		}
	}

	if len(problems) == 0 {
		return nil
	}
	// Sorted by field order then by the loop above, so the message is stable
	// between two identical submissions.
	return fmt.Errorf("invalid %s configuration: %s", d.Type, strings.Join(problems, ", "))
}

func (f Field) validate(values Values) string {
	if !values.Has(f.Name) {
		if f.Required {
			return fmt.Sprintf("%s is required", f.Name)
		}
		return ""
	}

	switch f.Kind {
	case FieldKindText, FieldKindSecret:
		if f.Required && strings.TrimSpace(values.String(f.Name)) == "" {
			return fmt.Sprintf("%s is required", f.Name)
		}
	case FieldKindStringList:
		if f.Required && len(values.StringList(f.Name)) == 0 {
			return fmt.Sprintf("%s is required", f.Name)
		}
	case FieldKindSelect:
		value := values.String(f.Name)
		if value == "" {
			if f.Required {
				return fmt.Sprintf("%s is required", f.Name)
			}
			return ""
		}
		for _, option := range f.Options {
			if option.Value == value {
				return ""
			}
		}
		return fmt.Sprintf("%s must be one of %s", f.Name, strings.Join(optionValues(f.Options), ", "))
	}

	return ""
}

func optionValues(options []Option) []string {
	values := make([]string, 0, len(options))
	for _, option := range options {
		values = append(values, option.Value)
	}
	return values
}

// Split separates a submitted configuration into what is stored in clear and what
// is stored encrypted. Splitting on the descriptor rather than on a naming
// convention is what keeps a provider free to name its fields as it likes.
func (d Descriptor) Split(values Values) (settings, secrets Values) {
	settings, secrets = Values{}, Values{}

	for _, field := range d.Fields {
		if !values.Has(field.Name) {
			continue
		}
		if field.IsSecret() {
			secrets[field.Name] = values[field.Name]
			continue
		}
		settings[field.Name] = values[field.Name]
	}

	return settings, secrets
}

// MergeSecrets applies the update rule for secrets: a secret left out of a
// submission, or submitted empty, keeps its stored value. A configuration form
// never receives the current secret, so it cannot send it back.
func (d Descriptor) MergeSecrets(stored, incoming Values) Values {
	merged := incoming.Clone()

	for _, field := range d.Fields {
		if !field.IsSecret() {
			continue
		}
		if strings.TrimSpace(merged.String(field.Name)) == "" && stored.Has(field.Name) {
			merged[field.Name] = stored[field.Name]
		}
	}

	return merged
}

// SecretNames lists the secret fields that hold a value, which is all a client
// ever learns about them.
func (d Descriptor) SecretNames(secrets Values) []string {
	names := make([]string, 0, len(secrets))

	for _, field := range d.Fields {
		if field.IsSecret() && strings.TrimSpace(secrets.String(field.Name)) != "" {
			names = append(names, field.Name)
		}
	}

	return names
}

// ApplyDefaults fills in the fields left out of a submission that declare a
// default, so a provider reads its configuration without repeating them.
func (d Descriptor) ApplyDefaults(values Values) Values {
	filled := values.Clone()

	for _, field := range d.Fields {
		if field.Default != nil && !filled.Has(field.Name) {
			filled[field.Name] = field.Default
		}
	}

	return filled
}
