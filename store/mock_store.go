package store

// MockStore is a mocked secret store, for testing
type MockStore struct{}

// Create (no-op) mocks creating a secret
func (s *MockStore) Create(id SecretIdentifier, value string) error {
	return nil
}

// Read (no-op) mocks reading a secret
func (s *MockStore) Read(id SecretIdentifier) (Secret, error) {
	return Secret{}, nil
}

// ReadVersion (no-op) mocks reading a version of a secret
func (s *MockStore) ReadVersion(id SecretIdentifier, version int) (Secret, error) {
	return Secret{}, nil
}

// Update (no-op) mocks updating a secret
func (s *MockStore) Update(id SecretIdentifier, value string) (Secret, error) {
	return Secret{}, nil
}

// History (no-op) mocks retrieving historical versions of a secret
func (s *MockStore) History(id SecretIdentifier) ([]SecretMeta, error) {
	return []SecretMeta{}, nil
}

// NewMockStore creates a mock secret store, with all no-op methods.
func NewMockStore() SecretStore {
	return &MockStore{}
}
