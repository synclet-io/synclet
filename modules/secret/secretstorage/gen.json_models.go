package secretstorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	time "time"

	uuid "github.com/google/uuid"

	secretservice "github.com/synclet-io/synclet/modules/secret/secretservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonSecret struct {
	ID             uuid.UUID        `json:"id"`
	EncryptedValue sliceValue[byte] `json:"encrypted_value"`
	Salt           sliceValue[byte] `json:"salt"`
	Nonce          sliceValue[byte] `json:"nonce"`
	KeyVersion     int              `json:"key_version"`
	OwnerType      string           `json:"owner_type"`
	OwnerID        uuid.UUID        `json:"owner_id"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

func (s *jsonSecret) Scan(value any) error {
	return json.Unmarshal(value.([]byte), s)
}

func (s jsonSecret) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func convertSecretToJsonModel(src *secretservice.Secret) (*jsonSecret, error) {
	result := &jsonSecret{}
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

func convertSecretFromJsonModel(src *jsonSecret) (*secretservice.Secret, error) {
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
