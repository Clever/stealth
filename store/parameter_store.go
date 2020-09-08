package store

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func init() {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-west-1"
	}
	Region = region
}

func getAPIClients() map[string]*ssm.SSM {
	return map[string]*ssm.SSM{
		"us-west-1": ssm.New(session.New(&aws.Config{Region: aws.String("us-west-1")})),
		"us-west-2": ssm.New(session.New(&aws.Config{Region: aws.String("us-west-2")})),
		"us-east-1": ssm.New(session.New(&aws.Config{Region: aws.String("us-east-1")})),
		"us-east-2": ssm.New(session.New(&aws.Config{Region: aws.String("us-east-2")})),
	}
}


// ParameterStore is a secret store that uses AWS SSM Parameter store 
type ParameterStore struct {
	ssmClients map[string]*ssm.SSM
}


// Creates a Secret in the secret store. Version is guaranteed to be zero if no error is returned.
func (s *ParameterStore) Create(id SecretIdentifier, value string) error {
	return nil
}

// Read a Secret from the store. Returns the lastest version of the secret.
func (s *ParameterStore) Read(id SecretIdentifier) (Secret, error) {
	return nil, nil
}

// ReadVersion reads a specific version of a secret from the store.
// Version is 0-indexed
func (s *ParameterStore) ReadVersion(id SecretIdentifier, version int) (Secret, error) {
	return nil, nil
}

// Updates a Secret from the store and increments version number.
func (s *ParameterStore) Update(id SecretIdentifier, value string) (Secret, error) {
	return nil, nil
}

// List gets secrets within a namespace (env/service)>
func (s *ParameterStore) List(env Environment, service string) ([]SecretIdentifier, error) {
	return nil, nil
}

// ListAll gets all secrets within a environment (env)>
func (s *ParameterStore) ListAll(env Environment) ([]SecretIdentifier, error) {
	return nil, nil
}

// History gets history for a secret, returning all versions from the store.
func (s *ParameterStore) History(id SecretIdentifier) ([]SecretMeta, error) {
	return nil, nil
}

// Delete deletes all versions of a secret
func (s *ParameterStore) Delete(id SecretIdentifier) error {
	return nil
}

// NewParameterStore creates a secret store that points at ParameterStore
func NewParameterStore() *ParameterStore {
	return &ParameterStore{
		ssmClients: getAPIClients(),
	}
}