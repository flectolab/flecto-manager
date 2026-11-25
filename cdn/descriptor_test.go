package cdn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testDescriptor covers every field kind.
func testDescriptor() Descriptor {
	return Descriptor{
		Type:  "test",
		Label: "Test",
		Fields: []Field{
			{Name: "zone_id", Label: "Zone", Kind: FieldKindText, Required: true},
			{Name: "api_token", Label: "Token", Kind: FieldKindSecret, Required: true},
			{Name: "hosts", Label: "Hosts", Kind: FieldKindStringList, Required: true},
			{Name: "note", Label: "Note", Kind: FieldKindText},
			{Name: "verbose", Label: "Verbose", Kind: FieldKindBool, Default: false},
			{Name: "timeout", Label: "Timeout", Kind: FieldKindInt, Default: 10},
			{Name: "mode", Label: "Mode", Kind: FieldKindSelect, Options: []Option{{Value: "fast"}, {Value: "safe"}}},
		},
		Capabilities: Capabilities{URL: true},
	}
}

func validValues() Values {
	return Values{
		"zone_id":   "abc123",
		"api_token": "s3cret",
		"hosts":     []string{"www.example.com"},
	}
}

func TestField_IsSecret(t *testing.T) {
	assert.True(t, Field{Kind: FieldKindSecret}.IsSecret())
	assert.False(t, Field{Kind: FieldKindText}.IsSecret())
}

func TestDescriptor_Field(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		got, ok := testDescriptor().Field("zone_id")

		assert.True(t, ok)
		assert.Equal(t, "zone_id", got.Name)
	})

	t.Run("failWithUnknownField", func(t *testing.T) {
		_, ok := testDescriptor().Field("nope")

		assert.False(t, ok)
	})
}

func TestDescriptor_Validate(t *testing.T) {
	tests := []struct {
		name       string
		values     Values
		wantErr    assert.ErrorAssertionFunc
		wantErrMsg string
	}{
		{
			name:    "success",
			values:  validValues(),
			wantErr: assert.NoError,
		},
		{
			name: "successWithOptionalFields",
			values: Values{
				"zone_id":   "abc123",
				"api_token": "s3cret",
				"hosts":     []string{"www.example.com"},
				"note":      "",
				"verbose":   true,
				"timeout":   30,
				"mode":      "safe",
			},
			wantErr: assert.NoError,
		},
		{
			name:       "failWithMissingRequiredField",
			values:     Values{"api_token": "s3cret", "hosts": []string{"a.com"}},
			wantErr:    assert.Error,
			wantErrMsg: "zone_id is required",
		},
		{
			name:       "failWithBlankRequiredField",
			values:     Values{"zone_id": "   ", "api_token": "s3cret", "hosts": []string{"a.com"}},
			wantErr:    assert.Error,
			wantErrMsg: "zone_id is required",
		},
		{
			name:       "failWithEmptyRequiredList",
			values:     Values{"zone_id": "abc", "api_token": "s3cret", "hosts": []string{}},
			wantErr:    assert.Error,
			wantErrMsg: "hosts is required",
		},
		{
			name:       "failWithUnknownField",
			values:     Values{"zone_id": "abc", "api_token": "s3cret", "hosts": []string{"a.com"}, "nope": "x"},
			wantErr:    assert.Error,
			wantErrMsg: `unknown field "nope"`,
		},
		{
			name:       "failWithInvalidSelectValue",
			values:     Values{"zone_id": "abc", "api_token": "s3cret", "hosts": []string{"a.com"}, "mode": "turbo"},
			wantErr:    assert.Error,
			wantErrMsg: "mode must be one of fast, safe",
		},
		{
			name:    "successWithEmptyOptionalSelect",
			values:  Values{"zone_id": "abc", "api_token": "s3cret", "hosts": []string{"a.com"}, "mode": ""},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testDescriptor().Validate(tt.values)

			tt.wantErr(t, err)
			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestDescriptor_Split(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		settings, secrets := testDescriptor().Split(validValues())

		assert.Equal(t, Values{"zone_id": "abc123", "hosts": []string{"www.example.com"}}, settings)
		assert.Equal(t, Values{"api_token": "s3cret"}, secrets)
	})

	t.Run("successIgnoresUnknownValues", func(t *testing.T) {
		values := validValues()
		values["nope"] = "x"

		settings, secrets := testDescriptor().Split(values)

		assert.NotContains(t, settings, "nope")
		assert.NotContains(t, secrets, "nope")
	})

	t.Run("successWithoutValues", func(t *testing.T) {
		settings, secrets := testDescriptor().Split(Values{})

		assert.Equal(t, Values{}, settings)
		assert.Equal(t, Values{}, secrets)
	})
}

func TestDescriptor_MergeSecrets(t *testing.T) {
	stored := Values{"api_token": "stored-token"}

	tests := []struct {
		name     string
		incoming Values
		want     string
	}{
		{
			name:     "successKeepsStoredWhenAbsent",
			incoming: Values{"zone_id": "abc"},
			want:     "stored-token",
		},
		{
			name:     "successKeepsStoredWhenBlank",
			incoming: Values{"api_token": "   "},
			want:     "stored-token",
		},
		{
			name:     "successReplacesWhenProvided",
			incoming: Values{"api_token": "new-token"},
			want:     "new-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testDescriptor().MergeSecrets(stored, tt.incoming)

			assert.Equal(t, tt.want, got.String("api_token"))
		})
	}

	t.Run("successWithoutStoredSecret", func(t *testing.T) {
		got := testDescriptor().MergeSecrets(Values{}, Values{"zone_id": "abc"})

		assert.False(t, got.Has("api_token"))
	})

	t.Run("successDoesNotMutateIncoming", func(t *testing.T) {
		incoming := Values{"zone_id": "abc"}

		testDescriptor().MergeSecrets(stored, incoming)

		assert.False(t, incoming.Has("api_token"))
	})
}

func TestDescriptor_SecretNames(t *testing.T) {
	t.Run("successWithFilledSecret", func(t *testing.T) {
		assert.Equal(t, []string{"api_token"}, testDescriptor().SecretNames(Values{"api_token": "s3cret"}))
	})

	t.Run("successWithBlankSecret", func(t *testing.T) {
		assert.Empty(t, testDescriptor().SecretNames(Values{"api_token": "  "}))
	})

	t.Run("successWithoutSecret", func(t *testing.T) {
		assert.Empty(t, testDescriptor().SecretNames(Values{}))
	})
}

func TestDescriptor_ApplyDefaults(t *testing.T) {
	t.Run("successFillsMissingFields", func(t *testing.T) {
		got := testDescriptor().ApplyDefaults(validValues())

		assert.Equal(t, false, got["verbose"])
		assert.Equal(t, 10, got["timeout"])
	})

	t.Run("successKeepsProvidedValues", func(t *testing.T) {
		values := validValues()
		values["timeout"] = 30

		got := testDescriptor().ApplyDefaults(values)

		assert.Equal(t, 30, got.Int("timeout"))
	})

	t.Run("successDoesNotMutateInput", func(t *testing.T) {
		values := validValues()

		testDescriptor().ApplyDefaults(values)

		assert.False(t, values.Has("timeout"))
	})
}

func multiSecretDescriptor() Descriptor {
	return Descriptor{
		Type:  "multi",
		Label: "Multi",
		Fields: []Field{
			{Name: "endpoint", Label: "Endpoint", Kind: FieldKindText, Required: true},
			{Name: "access_key", Label: "Access key", Kind: FieldKindSecret, Required: true},
			{Name: "secret_key", Label: "Secret key", Kind: FieldKindSecret, Required: true},
		},
	}
}

func TestDescriptor_MultipleSecrets(t *testing.T) {
	submitted := Values{
		"endpoint":   "https://api.example.com",
		"access_key": "AKIA",
		"secret_key": "s3cret",
	}

	t.Run("successSplitsEverySecret", func(t *testing.T) {
		settings, secrets := multiSecretDescriptor().Split(submitted)

		assert.Equal(t, Values{"endpoint": "https://api.example.com"}, settings)
		assert.Equal(t, Values{"access_key": "AKIA", "secret_key": "s3cret"}, secrets)
	})

	t.Run("successMergesEachSecretIndependently", func(t *testing.T) {
		stored := Values{"access_key": "stored-key", "secret_key": "stored-secret"}

		// Only one secret is rotated; the other must survive.
		merged := multiSecretDescriptor().MergeSecrets(stored, Values{
			"endpoint":   "https://api.example.com",
			"secret_key": "rotated",
		})

		assert.Equal(t, "stored-key", merged.String("access_key"))
		assert.Equal(t, "rotated", merged.String("secret_key"))
	})

	t.Run("successNamesEverySetSecret", func(t *testing.T) {
		names := multiSecretDescriptor().SecretNames(Values{"access_key": "AKIA", "secret_key": "s3cret"})

		assert.Equal(t, []string{"access_key", "secret_key"}, names)
	})

	t.Run("successNamesOnlyTheSecretsThatHoldAValue", func(t *testing.T) {
		names := multiSecretDescriptor().SecretNames(Values{"access_key": "AKIA"})

		assert.Equal(t, []string{"access_key"}, names)
	})

	t.Run("failWhenOneSecretIsMissing", func(t *testing.T) {
		err := multiSecretDescriptor().Validate(Values{
			"endpoint":   "https://api.example.com",
			"access_key": "AKIA",
		})

		assert.ErrorContains(t, err, "secret_key is required")
	})
}
