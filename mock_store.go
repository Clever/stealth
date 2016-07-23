package main

// MockStore is a mocked secret store, for testing
type MockStore struct{}

// Write (no-op) mocks writing a secret
func (s *MockStore) Write(key string, value SecretData) error {
	return nil
}

// Read (no-op) mocks reading a secret
func (s *MockStore) Read(key string) (Secret, error) {
	return Secret{}, nil
}

// History (no-op) mocks retrieving historical versions of a secret
func (s *MockStore) History(key string) ([]Secret, error) {
	return []Secret{}, nil
}

// Revoke (no-op) mocks revoking a secret
func (s *MockStore) Revoke(key string) error {
	return nil
}

// NewMockStore creates a mock secret store, with all no-op methods.
func NewMockStore() SecretStore {
	return &MockStore{}
}
