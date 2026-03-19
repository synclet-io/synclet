package secretservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

type SecretField byte

const (
	SecretFieldID SecretField = iota + 1
	SecretFieldEncryptedValue
	SecretFieldSalt
	SecretFieldNonce
	SecretFieldKeyVersion
	SecretFieldOwnerType
	SecretFieldOwnerID
	SecretFieldCreatedAt
	SecretFieldUpdatedAt
)

type SecretFilter struct {
	ID        filter.Filter[uuid.UUID]
	OwnerType filter.Filter[string]
	OwnerID   filter.Filter[uuid.UUID]
	Or        []*SecretFilter
	And       []*SecretFilter
}
type SecretOrder order.Order[SecretField]

type Secret struct {
	ID             uuid.UUID
	EncryptedValue []byte
	Salt           []byte
	Nonce          []byte
	KeyVersion     int
	OwnerType      string
	OwnerID        uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// user code 'Secret methods'
// end user code 'Secret methods'

func (s *Secret) Copy() Secret {
	var result Secret
	result.ID = s.ID
	tmp := make([]byte, 0, len(s.EncryptedValue))
	for _, i := range s.EncryptedValue {
		var itemCopy byte
		itemCopy = i
		tmp = append(tmp, itemCopy)
	}
	result.EncryptedValue = tmp
	tmp1 := make([]byte, 0, len(s.Salt))
	for _, i1 := range s.Salt {
		var itemCopy1 byte
		itemCopy1 = i1
		tmp1 = append(tmp1, itemCopy1)
	}
	result.Salt = tmp1
	tmp2 := make([]byte, 0, len(s.Nonce))
	for _, i2 := range s.Nonce {
		var itemCopy2 byte
		itemCopy2 = i2
		tmp2 = append(tmp2, itemCopy2)
	}
	result.Nonce = tmp2
	result.KeyVersion = s.KeyVersion
	result.OwnerType = s.OwnerType
	result.OwnerID = s.OwnerID
	result.CreatedAt = s.CreatedAt
	result.UpdatedAt = s.UpdatedAt

	return result
}
func (s *Secret) Equals(to *Secret) bool {
	if (s == nil) != (to == nil) {
		return false
	}
	if s == nil && to == nil {
		return true
	}
	if s.ID != to.ID {
		return false
	}
	if len(s.EncryptedValue) != len(to.EncryptedValue) {
		return false
	}
	for idx := range s.EncryptedValue {
		if s.EncryptedValue[idx] != to.EncryptedValue[idx] {
			return false
		}
	}
	if len(s.Salt) != len(to.Salt) {
		return false
	}
	for idx1 := range s.Salt {
		if s.Salt[idx1] != to.Salt[idx1] {
			return false
		}
	}
	if len(s.Nonce) != len(to.Nonce) {
		return false
	}
	for idx2 := range s.Nonce {
		if s.Nonce[idx2] != to.Nonce[idx2] {
			return false
		}
	}
	if s.KeyVersion != to.KeyVersion {
		return false
	}
	if s.OwnerType != to.OwnerType {
		return false
	}
	if s.OwnerID != to.OwnerID {
		return false
	}
	if s.CreatedAt != to.CreatedAt {
		return false
	}
	if s.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}
