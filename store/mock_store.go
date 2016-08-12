package store

// MockStore is a mocked secret store, for testing
type MockStore struct{}

// Create (no-op) mocks creating a secret
func (s *MockStore) Create(key string, value SecretData) error {
	return nil
}

// Read (no-op) mocks reading a secret
func (s *MockStore) Read(key string) (Secret, error) {
	return Secret{}, nil
}

// Update (no-op) mocks updating a secret
func (s *MockStore) Update(key string, value SecretData) (Secret, error) {
	return Secret{}, nil
}

// History (no-op) mocks retrieving historical versions of a secret
func (s *MockStore) History(key string) ([]SecretMeta, error) {
	return []SecretMeta{}, nil
}

// NewMockStore creates a mock secret store, with all no-op methods.
func NewMockStore() SecretStore {
	return &MockStore{}
}
