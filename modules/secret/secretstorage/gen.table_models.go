package secretstorage

import (
	time "time"

	uuid "github.com/google/uuid"

	secretservice "github.com/synclet-io/synclet/modules/secret/secretservice"
	// user code 'imports'
	// end user code 'imports'
)

type dbSecret struct {
	ID             uuid.UUID        `gorm:"column:id;"`
	EncryptedValue sliceValue[byte] `gorm:"column:encrypted_value;"`
	Salt           sliceValue[byte] `gorm:"column:salt;"`
	Nonce          sliceValue[byte] `gorm:"column:nonce;"`
	KeyVersion     int              `gorm:"column:key_version;"`
	OwnerType      string           `gorm:"column:owner_type;type:text;"`
	OwnerID        uuid.UUID        `gorm:"column:owner_id;"`
	CreatedAt      time.Time        `gorm:"column:created_at;"`
	UpdatedAt      time.Time        `gorm:"column:updated_at;"`
}

func convertSecretToDB(src *secretservice.Secret) (*dbSecret, error) {
	result := &dbSecret{}
	result.ID = src.ID
	tmp1 := make(sliceValue[byte], 0, len(src.EncryptedValue))
	for _, el := range src.EncryptedValue {
		tmp1 = append(tmp1, el)
	}
	result.EncryptedValue = tmp1
	tmp3 := make(sliceValue[byte], 0, len(src.Salt))
	for _, el := range src.Salt {
		tmp3 = append(tmp3, el)
	}
	result.Salt = tmp3
	tmp5 := make(sliceValue[byte], 0, len(src.Nonce))
	for _, el := range src.Nonce {
		tmp5 = append(tmp5, el)
	}
	result.Nonce = tmp5
	result.KeyVersion = src.KeyVersion
	result.OwnerType = src.OwnerType
	result.OwnerID = src.OwnerID
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()

	return result, nil
}

func convertSecretFromDB(src *dbSecret) (*secretservice.Secret, error) {
	result := &secretservice.Secret{}
	result.ID = src.ID
	tmp13 := make([]byte, 0, len(src.EncryptedValue))
	for _, el := range src.EncryptedValue {

		tmp13 = append(tmp13, el)
	}
	result.EncryptedValue = tmp13
	tmp15 := make([]byte, 0, len(src.Salt))
	for _, el := range src.Salt {

		tmp15 = append(tmp15, el)
	}
	result.Salt = tmp15
	tmp17 := make([]byte, 0, len(src.Nonce))
	for _, el := range src.Nonce {

		tmp17 = append(tmp17, el)
	}
	result.Nonce = tmp17
	result.KeyVersion = src.KeyVersion
	result.OwnerType = src.OwnerType
	result.OwnerID = src.OwnerID
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt

	return result, nil
}
func (a dbSecret) TableName() string {
	return "secret.secrets"
}
