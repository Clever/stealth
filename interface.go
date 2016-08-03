package main

import (
	"fmt"
	"time"
)

// Credentials needed to authenticate with secrets backend, such as a token
type Credentials map[string]string

// SecretMeta is metadata to manage a secret
type SecretMeta struct {
	Created    time.Time `json:"created"`
	Expiration time.Time `json:"expiration"`
	Version    int       `json:"version"`
	// TODO: Add other useful metadata?
}

// SecretData is the private data encapsulated by a Secret
type SecretData string

// Secret is the unit the secret store
type Secret struct {
	// Data is a key-val mapping. (secret1=foo,secret2=bar,...)
	Data SecretData `json:"data"`
	// Meta is
	Meta SecretMeta `json:"meta"`
}

// SecretStore is the CRUD-like interface for Secrets
type SecretStore interface {
	// Creates a Secret in the secret store. Version is guaranteed to be zero if no error is returned.
	Create(key string, value SecretData) error

	// Read a Secret from the store
	Read(key string) (Secret, error)

	// ReadVersion reads a specific version of a secret from the store
	// Version is 0-indexed
	// If version < 0, means “latest” version
	// ReadVersion(key string, version int)

	// Updates a Secret from the store and increments version number.
	Update(key string, value SecretData) (Secret, error)

	// History gets history for a secret, returning all versions from the store
	History(key string) ([]Secret, error)

	// Revoke a Secret from the store. History will still be available, but a Read operation will error.
	Revoke(key string) error
}

// KeyNotFoundError occurs when a key cannot be found (during Read, History, Update or Revoke)
type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string { return fmt.Sprintf("Key not found: %s", e.Key) }

// KeyRevokedError occurs when a key was revoked, and no later Write operations have occured
type KeyRevokedError struct {
	Key string
}

func (e *KeyRevokedError) Error() string { return fmt.Sprintf("Key was revoked: %s", e.Key) }

// InvalidKeyError occurs when a malformed key argument is given to a SecretStore method
type InvalidKeyError struct {
	Key string
}

func (e *InvalidKeyError) Error() string { return fmt.Sprintf("The given key is invalid: %s", e.Key) }

// KeyAlreadyExistsError occurs when Create is called and a key already exists
type KeyAlreadyExistsError struct {
	Key string
}

func (e *KeyAlreadyExistsError) Error() string {
	return fmt.Sprintf("The key already exists: %s", e.Key)
}

// AuthenticationError occurs when the given credentials fail to access the secret store
type AuthenticationError struct{}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Unable to authenticate with the store using the given credentials")
}

// AuthorizationError occurs when a user lacks sufficient access to interact with a Secret (read-only? read/write?)
type AuthorizationError struct {
	Key string
}

func (e *AuthorizationError) Error() string {
	return fmt.Sprintf("Unauthorized to access secret with key: %s", e.Key)
}
