package main

// mHistory has all versions of a secret, and its revocation status
type mHistory struct {
	// Secrets contains all versions of a secret
	Secrets []Secret
	// Revoked is whether or not a secret has been revoked
	Revoked bool
}

// TODO: Grab a lock whenever manipulating any key

// MemoryStore is an in-memory secret store, for testing
type MemoryStore struct {
	history map[string]mHistory
}

func (s *MemoryStore) Write(key string, value SecretData) error {
	var (
		history mHistory
		ok      bool
	)

	// Initialize secret if does not exist
	if history, ok = s.history[key]; !ok {
		s.history[key] = mHistory{
			Secrets: []Secret{},
			Revoked: false,
		}
	}

	// Append newest version
	history.Secrets = append(s.history[key].Secrets, Secret{Data: value})
	// Mark as non-revoked
	history.Revoked = false

	// Save
	s.history[key] = history

	return nil
}

func (s *MemoryStore) Read(key string) (Secret, error) {
	if history, ok := s.history[key]; ok {
		if history.Revoked {
			return Secret{}, &KeyRevokedError{Key: key}
		}
		return history.Secrets[len(history.Secrets)-1], nil
	}
	return Secret{}, &KeyNotFoundError{Key: key}
}

func (s *MemoryStore) History(key string) ([]Secret, error) {
	if history, ok := s.history[key]; ok {
		return history.Secrets, nil
	}
	return []Secret{}, &KeyNotFoundError{Key: key}
}

func (s *MemoryStore) Revoke(key string) error {
	if history, ok := s.history[key]; ok {
		history.Revoked = true
		s.history[key] = history
		return nil
	}
	return &KeyNotFoundError{Key: key}
}

func NewMemoryStore() SecretStore {
	return &MemoryStore{
		history: map[string]mHistory{},
	}
}
