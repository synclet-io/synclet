package secretservice

type baseErr string

func (e baseErr) Error() string { return string(e) }

const (
	ErrDecryptionFailed baseErr = "decryption failed: invalid key or corrupted data"
)
