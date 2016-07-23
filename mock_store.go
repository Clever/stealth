package main

// MockStore is a mocked secret store, for testing
type MockStore struct{}

func (s *MockStore) Write(key string, value SecretData) error {
	return nil
}

func (s *MockStore) Read(key string) (Secret, error) {
	return Secret{}, nil
}

func (s *MockStore) History(key string) ([]Secret, error) {
	return []Secret{}, nil
}

func (s *MockStore) Revoke(key string) error {
	return nil
}

func NewMockStore() SecretStore {
	return &MockStore{}
}
