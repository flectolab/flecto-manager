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
//
// It is registered once at boot by RegisterSerializer.
const SerializerName = "encrypted"

// Serializer encrypts a field on its way to the database and decrypts it on the
// way back. The value is JSON encoded first, so it works for any field type, not
// just strings.
type Serializer struct {
	Cipher Cipher
}

// RegisterSerializer wires the "encrypted" serializer to a cipher. GORM keeps
// serializers in a global registry, so this is called once, at startup, right
// after the cipher is built.
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

	// A NULL or empty column leaves the zero value in place: a row written before
	// the field existed must not fail to load.
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
