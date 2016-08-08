package main

// mHistory has all versions of a secret, and its revocation status
type mHistory struct {
	// Secrets contains all versions of a secret
	Secrets []Secret
}

// TODO: Grab a lock whenever manipulating any key

// MemoryStore is an in-memory secret store, for testing
type MemoryStore struct {
	history map[string]mHistory
}

// Create creates a secret in the store
func (s *MemoryStore) Create(key string, value SecretData) error {
	var (
		history mHistory
		ok      bool
	)

	// Initialize secret if does not exist
	if history, ok = s.history[key]; ok {
		return &KeyAlreadyExistsError{Key: key}
	}

	// Append newest version
	history.Secrets = []Secret{Secret{Data: value}}

	// Save
	s.history[key] = history

	return nil
}

// Read a secret from the store
func (s *MemoryStore) Read(key string) (Secret, error) {
	if history, ok := s.history[key]; ok {
		return history.Secrets[len(history.Secrets)-1], nil
	}
	return Secret{}, &KeyNotFoundError{Key: key}
}

// Update updates a secret in the secret store
func (s *MemoryStore) Update(key string, value SecretData) (Secret, error) {
	var (
		history mHistory
		ok      bool
	)

	// Return error if secret does not exist
	if history, ok = s.history[key]; !ok {
		return Secret{}, &KeyNotFoundError{Key: key}
	}

	// Append newest version
	history.Secrets = append(s.history[key].Secrets, Secret{Data: value})

	// Save
	s.history[key] = history

	return Secret{Data: value}, nil
}

// History gets all historical versions of a secret
func (s *MemoryStore) History(key string) ([]Secret, error) {
	if history, ok := s.history[key]; ok {
		return history.Secrets, nil
	}
	return []Secret{}, &KeyNotFoundError{Key: key}
}

// NewMemoryStore creates an in-memory secret store
func NewMemoryStore() SecretStore {
	return &MemoryStore{
		history: map[string]mHistory{},
	}
}
