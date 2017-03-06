package store

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/Clever/unicreds"
	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
	"github.com/aws/aws-sdk-go/aws"
)

// UnicredsStore is a secret store pointing at a prod and dev unicreds (https://github.com/Clever/unicreds) backend
type UnicredsStore struct {
	Environments map[Environment]UnicredsConfig
}

// Region is default region for unicreds config
var Region = "us-west-1"

// UnicredsConfig stores the configuration for a unicreds KMS and DynamoDB
type UnicredsConfig struct {
	UnicredsPath  *string
	UnicredsAlias string
}

const prodKey, devKey, ciTestKey = "alias/stealth-key", "alias/stealth-key-dev", "alias/stealth-key-drone-test"

var prodPath, devPath, ciTestPath = "stealth", "stealth-dev", "stealth-drone-test"

// Production is the production unicreds config
var Production = UnicredsConfig{UnicredsPath: &prodPath, UnicredsAlias: prodKey}

// Development is the dev unicreds config
var Development = UnicredsConfig{UnicredsPath: &devPath, UnicredsAlias: devKey}

// CITest is the ci-test unicreds config
var CITest = UnicredsConfig{UnicredsPath: &ciTestPath, UnicredsAlias: ciTestKey}

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
	return Secret{secret.Secret, SecretMeta{Version: version, Created: time.Unix(secret.CreatedAt, 0)}}, nil
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
	return Secret{secret.Secret, SecretMeta{Version: version, Created: time.Unix(secret.CreatedAt, 0)}}, nil
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
	return Secret{value, SecretMeta{Version: secret.Meta.Version + 1, Created: secret.Meta.Created}}, nil
}

// List gets all secrets in a namespace
func (s *UnicredsStore) List(env Environment, service string) ([]SecretIdentifier, error) {
	secrets, err := s.ListAll(env)
	if err != nil {
		return []SecretIdentifier{}, err
	}
	results := []SecretIdentifier{}
	for _, id := range secrets {
		if id.Service == service {
			results = append(results, id)
		}
	}
	return results, nil
}

// ListAll gets all secrets in an environment.
// Note that this is a Unicreds-only function - not part of the SecretStore interface.
func (s *UnicredsStore) ListAll(env Environment) ([]SecretIdentifier, error) {
	// validate environment; avoids a panic looking up Unicreds path below
	if !isValidEnvironmentInt(env) {
		return []SecretIdentifier{}, fmt.Errorf("env %d is invalid", env)
	}

	// create a mockId, so we can get the unicreds store path
	mockID := SecretIdentifier{env, "###", "###"}
	secrets, err := unicreds.ListSecrets(s.path(mockID), false)
	if err != nil {
		return []SecretIdentifier{}, err
	}

	results := []SecretIdentifier{}
	for _, s := range secrets {
		id, err := stringToSecretIdentifier(s.Name)
		if err != nil {
			return []SecretIdentifier{}, err
		}
		results = append(results, id)
	}
	sort.Sort(ByIDString(results))
	return results, nil
}

// Delete deletes all versions of a secret from the secret store.
// Note that this is a Unicreds-only function - not part of the SecretStore interface.
func (s *UnicredsStore) Delete(id SecretIdentifier) error {
	return unicreds.DeleteSecret(s.path(id), id.String())
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
			secretMetas[version] = SecretMeta{Version: version, Created: time.Unix(secret.CreatedAt, 0)}
		}
	}
	if versions == 0 {
		return []SecretMeta{}, &IdentifierNotFoundError{Identifier: id}
	}
	return secretMetas[:versions], nil
}

// NewUnicredsStore creates a secret store that points at DynamoDB and KMS AWS resources
func NewUnicredsStore() *UnicredsStore {
	log.SetHandler(json.New(os.Stderr))
	unicreds.SetAwsConfig(aws.String(Region), nil)
	unicreds.SetDynamoDBConfig(&aws.Config{Region: aws.String(Region), MaxRetries: aws.Int(5)})
	environments := make(map[Environment]UnicredsConfig)
	environments[ProductionEnvironment] = Production
	environments[DevelopmentEnvironment] = Development
	environments[CITestEnvironment] = CITest
	return &UnicredsStore{Environments: environments}
}
