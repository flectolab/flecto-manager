package cdn

import (
	"fmt"
	"strings"
)

// FieldKind drives both the admin UI widget and the server-side validation.
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
// configuration serves, declared by providers that purge by URL.
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

// IsSecret reports whether the value is stored encrypted and never returned to a client.
func (f Field) IsSecret() bool {
	return f.Kind == FieldKindSecret
}

// Descriptor describes how to configure a provider and what it can do.
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

// Validate checks a submitted configuration and reports all problems at once.
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

// Split separates a configuration into clear settings and encrypted secrets.
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

// MergeSecrets keeps the stored value of a secret left out or submitted empty,
// since the form never receives the current secret.
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

// SecretNames lists the secret fields that hold a value.
func (d Descriptor) SecretNames(secrets Values) []string {
	names := make([]string, 0, len(secrets))

	for _, field := range d.Fields {
		if field.IsSecret() && strings.TrimSpace(secrets.String(field.Name)) != "" {
			names = append(names, field.Name)
		}
	}

	return names
}

// ApplyDefaults fills in missing fields that declare a default.
func (d Descriptor) ApplyDefaults(values Values) Values {
	filled := values.Clone()

	for _, field := range d.Fields {
		if field.Default != nil && !filled.Has(field.Name) {
			filled[field.Name] = field.Default
		}
	}

	return filled
}
