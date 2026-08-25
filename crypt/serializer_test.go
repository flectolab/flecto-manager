package crypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// serializerRow exercises the serializer on both a plain string and a structured
// field, since the value is JSON encoded before being sealed.
type serializerRow struct {
	ID     int64             `gorm:"primaryKey;autoIncrement"`
	Secret string            `gorm:"serializer:encrypted"`
	Extra  map[string]string `gorm:"serializer:encrypted"`
	Plain  string
}

func TestRegisterSerializer(t *testing.T) {
	t.Run("successRoundTripsThroughDatabase", func(t *testing.T) {
		db := setupSerializerTestDB(t, newTestCipher(t))

		row := &serializerRow{
			Secret: "cf-api-token",
			Extra:  map[string]string{"zone": "abc123"},
			Plain:  "not-a-secret",
		}
		assert.NoError(t, db.Create(row).Error)

		var got serializerRow
		assert.NoError(t, db.First(&got, row.ID).Error)

		assert.Equal(t, "cf-api-token", got.Secret)
		assert.Equal(t, map[string]string{"zone": "abc123"}, got.Extra)
		assert.Equal(t, "not-a-secret", got.Plain)
	})

	t.Run("successStoresCiphertextInColumn", func(t *testing.T) {
		db := setupSerializerTestDB(t, newTestCipher(t))
		assert.NoError(t, db.Create(&serializerRow{Secret: "cf-api-token"}).Error)

		var stored string
		assert.NoError(t, db.Raw("SELECT secret FROM serializer_rows LIMIT 1").Scan(&stored).Error)

		assert.NotContains(t, stored, "cf-api-token")
		assert.Contains(t, stored, envelopeVersion+".")
	})

	t.Run("successWithEmptyColumn", func(t *testing.T) {
		db := setupSerializerTestDB(t, newTestCipher(t))
		assert.NoError(t, db.Exec("INSERT INTO serializer_rows (secret, extra, plain) VALUES (NULL, '', 'x')").Error)

		var got serializerRow
		assert.NoError(t, db.First(&got).Error)

		// A row written before the column was encrypted must still load.
		assert.Equal(t, "", got.Secret)
		assert.Nil(t, got.Extra)
		assert.Equal(t, "x", got.Plain)
	})

	t.Run("failWithoutKeyOnWrite", func(t *testing.T) {
		db := setupSerializerTestDB(t, NewUnavailable())

		err := db.Create(&serializerRow{Secret: "cf-api-token"}).Error

		assert.ErrorIs(t, err, ErrNoKey)
	})

	t.Run("failWithRetiredKeyOnRead", func(t *testing.T) {
		retired, err := NewAESGCM(testKey(0xaa))
		assert.NoError(t, err)
		payload, err := EncryptString(retired, "cf-api-token")
		assert.NoError(t, err)

		// The column holds a value sealed with a key the current cipher does not
		// know, which is what a key rotated out too early looks like.
		db := setupSerializerTestDB(t, newTestCipher(t))
		assert.NoError(t, db.Exec("INSERT INTO serializer_rows (secret, extra, plain) VALUES (?, '', 'x')", payload).Error)

		var got serializerRow
		err = db.First(&got).Error

		assert.ErrorIs(t, err, ErrUnknownKey)
	})
}

func setupSerializerTestDB(t *testing.T, c Cipher) *gorm.DB {
	t.Helper()

	// The registry is global to GORM, so each case installs the cipher it needs.
	RegisterSerializer(c)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&serializerRow{}))

	return db
}
