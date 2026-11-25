package crypt

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"gorm.io/gorm/schema"
)

// SerializerName is the name to put in a GORM tag to encrypt a column:
//
//	Secret string `gorm:"serializer:encrypted"`
const SerializerName = "encrypted"

// Serializer encrypts a field on write and decrypts it on read. The value is
// JSON encoded first, so any field type works.
type Serializer struct {
	Cipher Cipher
}

// RegisterSerializer wires the "encrypted" serializer to a cipher in GORM's
// global registry.
func RegisterSerializer(c Cipher) {
	schema.RegisterSerializer(SerializerName, Serializer{Cipher: c})
}

func (s Serializer) Value(_ context.Context, _ *schema.Field, _ reflect.Value, fieldValue any) (any, error) {
	data, err := json.Marshal(fieldValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encode field before encryption: %w", err)
	}
	return s.Cipher.Encrypt(data)
}

func (s Serializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue any) error {
	fieldValue := reflect.New(field.FieldType)

	if payload, ok := toString(dbValue); ok && payload != "" {
		plaintext, err := s.Cipher.Decrypt(payload)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(plaintext, fieldValue.Interface()); err != nil {
			return fmt.Errorf("failed to decode field after decryption: %w", err)
		}
	}

	// A NULL or empty column (row older than the field) leaves the zero value.
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

func toString(dbValue any) (string, bool) {
	switch v := dbValue.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		return "", false
	}
}
