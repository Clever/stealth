package store

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/pkg/errors"
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
		// "us-west-2": ssm.New(session.New(&aws.Config{Region: aws.String("us-west-2")})),
		// "us-east-1": ssm.New(session.New(&aws.Config{Region: aws.String("us-east-1")})),
		// "us-east-2": ssm.New(session.New(&aws.Config{Region: aws.String("us-east-2")})),
	}
}

// getNamespace converts an env, app to a namespace to be used for parameterstore
func getNamespace(env string, app string) string {
	if app == "" {
		return fmt.Sprintf("/%s", env)
	}
	return fmt.Sprintf("/%s/%s", env, app)
}

// getSecretIDFromParamName converts from /development/oauth/foo-bar to SecretIdentifier development.oauth.foo-bar
func getSecretIDFromParamName(name string) (SecretIdentifier, error) {
	parts := strings.Split(name, "/")
	env, err := environmentStringToInt(parts[1])
	if err != nil {
		return SecretIdentifier{}, err
	}
	return SecretIdentifier{Environment: env, Service: parts[2], Key: parts[3]}, nil
}

// getParamNameFromName converts from development.oauth.foo-bar to /development/oauth/foo-bar
// because we want namespaces for parameter naming.
func getParamNameFromName(id SecretIdentifier) string {
	return fmt.Sprintf("%s/%s", getNamespace(id.EnvironmentString(), id.Service), id.Key)
}

// getParamNameFromNameAtVersion constructs AWS SSM paramname with version
func getParamNameFromNameAtVersion(id SecretIdentifier, version int) string {
	paramName := getParamNameFromName(id)
	// parameterStore is 1-indexed, hence we bump the version number from the SecretStore
	return fmt.Sprintf("%s:%d", paramName, version+1)
}

// ParameterStore is a secret store that uses AWS SSM Parameter store
type ParameterStore struct {
	ssmClients map[string]*ssm.SSM
}

// Creates a Secret in the secret store. Version is guaranteed to be zero if no error is returned.
func (s *ParameterStore) Create(id SecretIdentifier, value string) error {
	name := getParamNameFromName(id)
	putParameterInput := &ssm.PutParameterInput{
		Name:      aws.String(name),
		Overwrite: aws.Bool(false), // false since we are creating a new secret
		Type:      aws.String("SecureString"),
		Value:     aws.String(value),
	}
	var failedRegions []string
	var succeededRegions []string
	exists := false
	for region, regionClient := range s.ssmClients {
		_, err := regionClient.PutParameter(putParameterInput)
		// If any region fails, this operation fails.
		// This guarantee the invariant that the all secret values are consistent across regions.
		if err != nil {
			if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
				if awsErr.Code() == ssm.ErrCodeParameterAlreadyExists {
					exists = true
				}
			}
			failedRegions = append(failedRegions, region)
		} else {
			succeededRegions = append(succeededRegions, region)
		}
	}
	if len(failedRegions) > 0 {
		for _, region := range succeededRegions {
			regionClient := s.ssmClients[region]
			deleteParameterInput := &ssm.DeleteParameterInput{
				Name: aws.String(getParamNameFromName(id)),
			}
			_, err := regionClient.DeleteParameter(deleteParameterInput)
			if err != nil {
				return fmt.Errorf("Error creating secret for (%s). try again. Error: %s", region, err)
			}
		}
		if exists {
			return &IdentifierAlreadyExistsError{Identifier: id}
		}
		return fmt.Errorf("error creating secret for (%s). try again", strings.Join(failedRegions, ", "))
	}
	return nil
}

// Read a Secret from the store. Returns the lastest version of the secret.
func (s *ParameterStore) Read(id SecretIdentifier) (Secret, error) {
	paramName := getParamNameFromName(id)
	getParameterInput := &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(true),
	}

	var resp *ssm.GetParameterOutput
	var err error
	for _, regionClient := range s.ssmClients {
		resp, err = regionClient.GetParameter(getParameterInput)
		if err != nil {
			if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
				if awsErr.Code() == ssm.ErrCodeParameterNotFound {
					return Secret{}, &IdentifierNotFoundError{Identifier: id}
				}
			}
			return Secret{}, fmt.Errorf("ParamStore error: %s. ", err)
		}
	}
	return Secret{*resp.Parameter.Value, SecretMeta{Version: int(*resp.Parameter.Version)}}, nil
}

// ReadVersion reads a specific version of a secret from the store.
// Version is 0-indexed
func (s *ParameterStore) ReadVersion(id SecretIdentifier, version int) (Secret, error) {
	paramName := getParamNameFromNameAtVersion(id, version)
	getParameterInput := &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(true),
	}
	apiClient := s.ssmClients[Region]
	resp, err := apiClient.GetParameter(getParameterInput)
	if err != nil {
		if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
			if awsErr.Code() == ssm.ErrCodeParameterNotFound {
				return Secret{}, &IdentifierNotFoundError{Identifier: id}
			}
		}
		return Secret{}, fmt.Errorf("ParamStore error: %s. ", err)
	}
	return Secret{*resp.Parameter.Value, SecretMeta{Version: int(*resp.Parameter.Version)}}, nil
}

// Updates a Secret from the store and increments version number.
func (s *ParameterStore) Update(id SecretIdentifier, value string) (Secret, error) {
	return Secret{}, nil
}

// List gets secrets within a namespace (env/service)>
func (s *ParameterStore) List(env Environment, service string) ([]SecretIdentifier, error) {
	id := SecretIdentifier{env, service, ""}
	namespace := getNamespace(id.EnvironmentString(), service)
	apiClient := s.ssmClients[Region]

	describeParametersByPathInput := &ssm.DescribeParametersInput{
		ParameterFilters: []*ssm.ParameterStringFilter{
			&ssm.ParameterStringFilter{
				Key:    aws.String("Path"),
				Option: aws.String("Recursive"),
				Values: []*string{aws.String(namespace)},
			},
		},
	}
	results := []SecretIdentifier{}
	resp, err := apiClient.DescribeParameters(describeParametersByPathInput)
	if err != nil {
		return results, err
	}
	// fmt.Printf("params[%d] for namespace=%s \n ", len(resp.Parameters), namespace)
	for _, result := range resp.Parameters {
		ident, err := getSecretIDFromParamName(*result.Name)
		if err != nil {
			return results, err
		}
		results = append(results, ident)
	}
	return results, nil
}

// ListAll gets all secrets within a environment (env)>
func (s *ParameterStore) ListAll(env Environment) ([]SecretIdentifier, error) {
	return s.List(env, "")
}

// History gets history for a secret, returning all versions from the store.
func (s *ParameterStore) History(id SecretIdentifier) ([]SecretMeta, error) {
	return nil, nil
}

// Delete deletes all versions of a secret
func (s *ParameterStore) Delete(id SecretIdentifier) error {
	deleteParameterInput := &ssm.DeleteParameterInput{
		Name: aws.String(getParamNameFromName(id)),
	}
	for _, regionClient := range s.ssmClients {
		_, err := regionClient.DeleteParameter(deleteParameterInput)
		// If any region fails, add to the return list of errors and continue.
		if err != nil {
			return err
		}
	}
	return nil
}

// NewParameterStore creates a secret store that points at ParameterStore
func NewParameterStore() *ParameterStore {
	return &ParameterStore{
		ssmClients: getAPIClients(),
	}
}
