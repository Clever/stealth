package store

import (
	"fmt"
	"strings"
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

// Secret is the unit the secret store
type Secret struct {
	// Data is the actual secret value
	Data string `json:"data"`
	// Meta is the information about the secret
	Meta SecretMeta `json:"meta"`
}

// Environment is an Enum to access different Stealth stores
type Environment int

const (
	// ProductionEnvironment is an index for prod
	ProductionEnvironment Environment = iota
	// DevelopmentEnvironment is an index for dev
	DevelopmentEnvironment
	// DroneTestEnvironment is an index for drone-test
	DroneTestEnvironment
)

// SecretIdentifier is a lookup key for a secret, including the production flag, the service name, and the specific key
type SecretIdentifier struct {
	Environment  Environment
	Service, Key string
}

// EnvironmentString returns the environment used for the secret identifier, as a string
func (id SecretIdentifier) EnvironmentString() string {
	if id.Environment == ProductionEnvironment {
		return "production"
	} else if id.Environment == DevelopmentEnvironment {
		return "development"
	} else if id.Environment == DroneTestEnvironment {
		return "drone-test"
	}
	// Error
	return ""
}

// isValidEnvironmentInt checks if an int is among our supported environments.
// Our environments are represented as an enum
func isValidEnvironmentInt(env Environment) bool {
	for _, val := range []Environment{ProductionEnvironment, DevelopmentEnvironment, DroneTestEnvironment} {
		if env == val {
			return true
		}
	}
	return false
}

// environmentStringToInt converts a string like "production" into the corresponding environment int.
// Our environments are represented as an Enum.
func environmentStringToInt(s string) (Environment, error) {
	if s == "production" {
		return ProductionEnvironment, nil
	} else if s == "development" {
		return DevelopmentEnvironment, nil
	} else if s == "drone-test" {
		return DroneTestEnvironment, nil
	}
	return -1, fmt.Errorf("invalid environment: %s", s)
}

// String() returns the key used for the secret identifier
func (id SecretIdentifier) String() string {
	return fmt.Sprintf("%s.%s.%s", id.EnvironmentString(), id.Service, id.Key)
}

// stringToSecretIdentifier() returns the key used for the secret identifier
func stringToSecretIdentifier(s string) (SecretIdentifier, error) {
	parts := strings.SplitN(s, ".", 4)
	if len(parts) != 3 {
		return SecretIdentifier{}, fmt.Errorf("unable to create SecretIdentifier from string -- couldn't split: %s", s)
	}
	env, err := environmentStringToInt(parts[0])
	if err != nil {
		return SecretIdentifier{}, fmt.Errorf("unable to create SecretIdentifier from string -- invalid environment: %s", s)
	}
	return SecretIdentifier{env, parts[1], parts[2]}, nil
}

// SecretStore is the CRUD-like interface for Secrets
type SecretStore interface {
	// Creates a Secret in the secret store. Version is guaranteed to be zero if no error is returned.
	Create(id SecretIdentifier, value string) error

	// Read a Secret from the store. Returns the lastest version of the secret.
	Read(id SecretIdentifier) (Secret, error)

	// ReadVersion reads a specific version of a secret from the store.
	// Version is 0-indexed
	ReadVersion(id SecretIdentifier, version int) (Secret, error)

	// Updates a Secret from the store and increments version number.
	Update(id SecretIdentifier, value string) (Secret, error)

	// List gets secrets within a namespace (env/service)>
	List(env Environment, service string) ([]SecretIdentifier, error)

	// History gets history for a secret, returning all versions from the store.
	History(id SecretIdentifier) ([]SecretMeta, error)
}

// IdentifierNotFoundError occurs when a secret identifier cannot be found (during Read, History, Update)
type IdentifierNotFoundError struct {
	Identifier SecretIdentifier
}

func (e *IdentifierNotFoundError) Error() string {
	return fmt.Sprintf("Identifier not found: %s", e.Identifier)
}

// InvalidIdentifierError occurs when a malformed identifier argument is given to a SecretStore method
type InvalidIdentifierError struct {
	Identifier SecretIdentifier
}

func (e *InvalidIdentifierError) Error() string {
	return fmt.Sprintf("The given identifier is invalid: %s", e.Identifier)
}

// IdentifierAlreadyExistsError occurs when Create is called and an identifier already exists
type IdentifierAlreadyExistsError struct {
	Identifier SecretIdentifier
}

func (e *IdentifierAlreadyExistsError) Error() string {
	return fmt.Sprintf("The identifier already exists: %s", e.Identifier)
}

// VersionNotFoundError occurs when a secret version cannot be found (during ReadVersion)
type VersionNotFoundError struct {
	Identifier SecretIdentifier
	Version    int
}

func (e *VersionNotFoundError) Error() string {
	return fmt.Sprintf("Version %d not found for identifier: %s", e.Version, e.Identifier)
}

// AuthenticationError occurs when the given credentials fail to access the secret store
type AuthenticationError struct{}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Unable to authenticate with the store using the given credentials")
}

// AuthorizationError occurs when a user lacks sufficient access to interact with a Secret (read-only? read/write?)
type AuthorizationError struct {
	Identifier SecretIdentifier
}

func (e *AuthorizationError) Error() string {
	return fmt.Sprintf("Unauthorized to access secret with identifier: %s", e.Identifier)
}

// ByIDString allows sorting SecretIdentifiers by Key
type ByIDString []SecretIdentifier

func (s ByIDString) Len() int {
	return len(s)
}
func (s ByIDString) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByIDString) Less(i, j int) bool {
	return s[i].String() < s[j].String()
}
