package store

import (
	"fmt"
	"sort"
)

// mHistory has all versions of a secret, and its revocation status
type mHistory struct {
	// Secrets contains all versions of a secret
	Secrets []Secret
}

// MemoryStore is an in-memory secret store, for testing
type MemoryStore struct {
	history map[SecretIdentifier]mHistory
}

// Create creates a secret in the store
func (s *MemoryStore) Create(id SecretIdentifier, value string) error {
	var (
		history mHistory
		ok      bool
	)

	// Initialize secret if does not exist
	if history, ok = s.history[id]; ok {
		return &IdentifierAlreadyExistsError{Identifier: id}
	}

	// Append newest version
	history.Secrets = []Secret{Secret{Data: value}}

	// Save
	s.history[id] = history

	return nil
}

// Read a secret from the store
func (s *MemoryStore) Read(id SecretIdentifier) (Secret, error) {
	if history, ok := s.history[id]; ok {
		return history.Secrets[len(history.Secrets)-1], nil
	}
	return Secret{}, &IdentifierNotFoundError{Identifier: id, Region: ""}
}

// ReadVersion reads a version of a secret
func (s *MemoryStore) ReadVersion(id SecretIdentifier, version int) (Secret, error) {
	if history, ok := s.history[id]; ok {
		if len(history.Secrets) > version && version >= 0 {
			return history.Secrets[version], nil
		}
		return Secret{}, &VersionNotFoundError{Version: version, Identifier: id}
	}
	return Secret{}, &IdentifierNotFoundError{Identifier: id, Region: ""}
}

// Update updates a secret in the secret store
func (s *MemoryStore) Update(id SecretIdentifier, value string) (Secret, error) {
	var (
		history mHistory
		ok      bool
	)

	// Return error if secret does not exist
	if history, ok = s.history[id]; !ok {
		return Secret{}, &IdentifierNotFoundError{Identifier: id, Region: ""}
	}

	// Append newest version
	version := len(history.Secrets)
	history.Secrets = append(s.history[id].Secrets, Secret{Data: value, Meta: SecretMeta{Version: version}})

	// Save
	s.history[id] = history

	return Secret{Data: value}, nil
}

// List gets all secret identifiers within a namespace
func (s *MemoryStore) List(env Environment, service string) ([]SecretIdentifier, error) {
	ids, err := s.ListAll(env)
	if err != nil {
		return []SecretIdentifier{}, err
	}
	results := []SecretIdentifier{}
	for _, id := range ids {
		if id.Environment == env && id.Service == service {
			results = append(results, id)
		}
	}
	return results, nil
}

// ListAll gets all secret identifiers within an environment
func (s *MemoryStore) ListAll(env Environment) ([]SecretIdentifier, error) {
	// validate environment; avoids a panic looking up Unicreds path below
	if !isValidEnvironmentInt(env) {
		return []SecretIdentifier{}, fmt.Errorf("env %d is invalid", env)
	}

	results := []SecretIdentifier{}
	for id := range s.history {
		results = append(results, id)
	}
	sort.Sort(ByIDString(results))
	return results, nil
}

// History gets all historical versions of a secret
func (s *MemoryStore) History(id SecretIdentifier) ([]SecretMeta, error) {
	if history, ok := s.history[id]; ok {
		secrets := make([]SecretMeta, len(history.Secrets))
		for index, secret := range history.Secrets {
			secrets[index] = secret.Meta
		}
		return secrets, nil
	}
	return []SecretMeta{}, &IdentifierNotFoundError{Identifier: id, Region: ""}
}

// Delete deletes all versions of a secret
func (s *MemoryStore) Delete(id SecretIdentifier) error {
	if _, ok := s.history[id]; ok {
		delete(s.history, id)
		return nil
	}
	return &IdentifierNotFoundError{Identifier: id, Region: ""}
}

// NewMemoryStore creates an in-memory secret store
func NewMemoryStore() SecretStore {
	return &MemoryStore{
		history: map[SecretIdentifier]mHistory{},
	}
}
