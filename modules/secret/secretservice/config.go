package secretservice

// Config holds secret service configuration passed from the app layer.
type Config struct {
	MasterKey   []byte
	PreviousKey []byte
	KeyVersion  int
}
