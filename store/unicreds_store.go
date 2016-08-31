package store

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/Clever/unicreds"
	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
)

// UnicredsStore is a secret store pointing at a prod and dev unicreds (https://github.com/Clever/unicreds) backend
type UnicredsStore struct {
	Environments map[int]UnicredsConfig
}

// Region is default region for unicreds config
var Region = "us-west-1"

// UnicredsConfig stores the configuration for a unicreds KMS and DynamoDB
type UnicredsConfig struct {
	UnicredsPath  *string
	UnicredsAlias string
}

const prodKey, devKey, droneTestKey = "alias/stealth-key", "alias/stealth-key-dev", "alias/stealth-key-drone-test"

var prodPath, devPath, droneTestPath = "stealth", "stealth-dev", "stealth-drone-test"

// Production is the production unicreds config
var Production = UnicredsConfig{UnicredsPath: &prodPath, UnicredsAlias: prodKey}

// Development is the dev unicreds config
var Development = UnicredsConfig{UnicredsPath: &devPath, UnicredsAlias: devKey}

// DroneTest is the drone-test unicreds config
var DroneTest = UnicredsConfig{UnicredsPath: &droneTestPath, UnicredsAlias: droneTestKey}

// MalformedVersionError occurs when a secret version is malformed
type MalformedVersionError struct {
	Identifier       SecretIdentifier
	MalformedVersion string
}

func (e *MalformedVersionError) Error() string {
	return fmt.Sprintf("Version string %s for identifier %s is malformed", e.MalformedVersion, e.Identifier)
}

// getEncryptionContext gets the encryption context for a secret identifier
func getEncryptionContext(id SecretIdentifier) *unicreds.EncryptionContextValue {
	context := unicreds.NewEncryptionContextValue()
	context.Set(fmt.Sprintf("service:%s", id.Service))
	return context
}

func (s *UnicredsStore) path(id SecretIdentifier) *string {
	return s.Environments[id.Environment].UnicredsPath
}

func (s *UnicredsStore) alias(id SecretIdentifier) string {
	return s.Environments[id.Environment].UnicredsAlias
}

// Create creates a key in the unicreds store
func (s *UnicredsStore) Create(id SecretIdentifier, value string) error {
	_, err := s.Read(id)
	if err == nil {
		return &IdentifierAlreadyExistsError{Identifier: id}
	}
	err = unicreds.PutSecret(s.path(id), s.alias(id), id.String(), value, unicreds.PaddedInt(0), getEncryptionContext(id))
	return err
}

// Read reads the latest version of the secret
func (s *UnicredsStore) Read(id SecretIdentifier) (Secret, error) {
	secret, err := unicreds.GetHighestVersionSecret(s.path(id), id.String(), getEncryptionContext(id))
	if err != nil {
		return Secret{}, &IdentifierNotFoundError{Identifier: id}
	}
	version, err := strconv.Atoi(secret.Version)
	if err != nil {
		return Secret{}, &MalformedVersionError{Identifier: id, MalformedVersion: secret.Version}
	}
	return Secret{secret.Secret, SecretMeta{Version: version}}, nil
}

// ReadVersion reads a version of a secret
func (s *UnicredsStore) ReadVersion(id SecretIdentifier, version int) (Secret, error) {
	secret, err := unicreds.GetSecret(s.path(id), id.String(), unicreds.PaddedInt(version), getEncryptionContext(id))
	if err != nil {
		_, err = s.Read(id)
		if err != nil {
			return Secret{}, &IdentifierNotFoundError{Identifier: id}
		}
		return Secret{}, &VersionNotFoundError{Identifier: id, Version: version}
	}
	return Secret{secret.Secret, SecretMeta{Version: version}}, nil
}

// Update writes a new version of the key
func (s *UnicredsStore) Update(id SecretIdentifier, value string) (Secret, error) {
	secret, err := s.Read(id)
	if err != nil {
		return Secret{}, err
	}
	nextVersion, err := unicreds.ResolveVersion(s.path(id), id.String(), 0)
	if err != nil {
		return Secret{}, err
	}
	err = unicreds.PutSecret(s.path(id), s.alias(id), id.String(), value, nextVersion, getEncryptionContext(id))
	return Secret{value, SecretMeta{Version: secret.Meta.Version + 1}}, nil
}

// List gets all secrets in a namespace
func (s *UnicredsStore) List(env int, service string) ([]SecretIdentifier, error) {
	// validate environment; avoids a panic looking up Unicreds path below
	if !isValidEnvironmentInt(env) {
		return []SecretIdentifier{}, fmt.Errorf("env %d is invalid", env)
	}

	// create a mockId, so we can get the unicreds store path
	mockId := SecretIdentifier{env, service, "###"}
	secrets, err := unicreds.ListSecrets(s.path(mockId), false)
	if err != nil {
		return []SecretIdentifier{}, err
	}

	results := []SecretIdentifier{}
	for _, s := range secrets {
		id, err := stringToSecretIdentifier(s.Name)
		if err != nil {
			return []SecretIdentifier{}, err
		}
		if id.Environment == env && id.Service == service {
			results = append(results, id)
		}
	}
	sort.Sort(ByIDString(results))
	return results, nil
}

// History returns all versions of a secret
func (s *UnicredsStore) History(id SecretIdentifier) ([]SecretMeta, error) {
	secrets, err := unicreds.ListSecrets(s.path(id), true)
	if err != nil {
		return []SecretMeta{}, err
	}
	var secretMetas []SecretMeta
	secretMetas = make([]SecretMeta, len(secrets))
	versions := 0
	for _, secret := range secrets {
		if id.String() == secret.Name {
			versions++
			version, err := strconv.Atoi(secret.Version)
			if err != nil {
				return []SecretMeta{}, &MalformedVersionError{Identifier: id, MalformedVersion: secret.Version}
			}
			secretMetas[version] = SecretMeta{Version: version}
		}
	}
	if versions == 0 {
		return []SecretMeta{}, &IdentifierNotFoundError{Identifier: id}
	}
	return secretMetas[:versions], nil
}

// NewUnicredsStore creates a secret store that points at DynamoDB and KMS AWS resources
func NewUnicredsStore() SecretStore {
	log.SetHandler(json.New(os.Stderr))
	unicreds.SetAwsConfig(&Region, nil)
	environments := make(map[int]UnicredsConfig)
	environments[ProductionEnvironment] = Production
	environments[DevelopmentEnvironment] = Development
	environments[DroneTestEnvironment] = DroneTest
	return &UnicredsStore{Environments: environments}
}
