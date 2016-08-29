package store

import (
	"fmt"
	"github.com/Clever/unicreds"
	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
	"os"
	"strconv"
)

// UnicredsStore is a secret store pointing at a prod and dev unicreds (https://github.com/Clever/unicreds) backend
type UnicredsStore struct {
	Production  *UnicredsConfig
	Development *UnicredsConfig
}

// Region is default region for unicreds config
var Region = "us-west-1"

// ProdPath is default prod DynamoDB path
var ProdPath = "stealth"

// ProdKey is default prod KMS key
const ProdKey = "alias/stealth-key"

// DevPath is default dev DynamoDB path
var DevPath = "stealth-dev"

// DevKey is default dev KMS key
const DevKey = "alias/stealth-key-dev"

// UnicredsConfig stores the configuration for a unicreds KMS and DynamoDB
type UnicredsConfig struct {
	UnicredsPath  *string
	UnicredsAlias string
}

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
	if id.Production {
		return s.Production.UnicredsPath
	}
	return s.Development.UnicredsPath
}

func (s *UnicredsStore) alias(id SecretIdentifier) string {
	if id.Production {
		return s.Production.UnicredsAlias
	}
	return s.Development.UnicredsAlias
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
	development := UnicredsConfig{UnicredsPath: &DevPath, UnicredsAlias: DevKey}
	production := UnicredsConfig{UnicredsPath: &ProdPath, UnicredsAlias: ProdKey}
	return &UnicredsStore{Production: &production, Development: &development}
}
