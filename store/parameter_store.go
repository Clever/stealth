package store

import (
	"fmt"
	"os"
	"strings"
	"time"

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

// CurrentDeployError occurs when a parameter name has suffix current-deploy.
// Such parameters are private to catapult service and should not be surfaced via interface
type CurrentDeployError struct {
	Identifier string
}

func (e *CurrentDeployError) Error() string {
	return fmt.Sprintf("current-deploy parameter should not be surfaced for parameter %s", e.Identifier)
}

// getOrderedRegions provides guarantees that actions on ParamStore will happen
// within a specific order every time. This is helpful for any errors with inconsistent
// state
func getOrderedRegions() []string {
	return []string{
		"us-west-1",
		"us-west-2",
		"us-east-1",
		"us-east-2",
	}
}

func getAPIClients() map[string]*ssm.SSM {
	return map[string]*ssm.SSM{
		"us-west-1": ssm.New(session.New(&aws.Config{Region: aws.String("us-west-1")})),
		"us-west-2": ssm.New(session.New(&aws.Config{Region: aws.String("us-west-2")})),
		"us-east-1": ssm.New(session.New(&aws.Config{Region: aws.String("us-east-1")})),
		"us-east-2": ssm.New(session.New(&aws.Config{Region: aws.String("us-east-2")})),
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
		return SecretIdentifier{}, &InvalidEnvironmentError{Identifier: name}
	}
	if strings.HasSuffix(name, "current-deploy") {
		return SecretIdentifier{}, &CurrentDeployError{Identifier: name}
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
	return fmt.Sprintf("%s:%d", paramName, convertToSSMVersion(version))
}

// convertFromSSMVersion converts ParamStore 1-indexed version to be 0-indexed
func convertFromSSMVersion(version int) int {
	return version - 1
}

// convertToSSMVersion converts  0-indexed version specifier to be 1-indexed for ParamStore
func convertToSSMVersion(version int) int {
	return version + 1
}

// ParameterStore is a secret store that uses AWS SSM Parameter store
type ParameterStore struct {
	ssmClients        map[string]*ssm.SSM
	maxResultsToQuery int64
}

// Create creates a Secret in the secret store. Version is guaranteed to be zero if no error is returned.
func (s *ParameterStore) Create(id SecretIdentifier, value string) error {
	name := getParamNameFromName(id)
	putParameterInput := &ssm.PutParameterInput{
		Name:      aws.String(name),
		Overwrite: aws.Bool(false), // false since we are creating a new secret
		Type:      aws.String(ssm.ParameterTypeSecureString),
		Value:     aws.String(value),
	}

	_, errors := s.readForAllRegions(getParamNameFromName(id))
	for _, err := range errors {
		// the secret exists in some regions, throw error
		if err == nil {
			return &IdentifierAlreadyExistsError{Identifier: id}
		}
	}

	var abortOperation bool
	var failedRegions []string
	orderedRegions := getOrderedRegions()
	for _, region := range orderedRegions {
		regionClient := s.ssmClients[region]
		_, err := regionClient.PutParameter(putParameterInput)
		// If any region fails, we will retry one more time. If retry fails, this Read operation fails.
		// This guarantee the invariant that the all secret values are consistent across regions.
		if err != nil {
			failedRegions = append(failedRegions, region)
		}
	}

	abortOperation = false
	// lets try one more time for the failed regions
	if len(failedRegions) > 0 {
		for _, region := range failedRegions {
			regionClient := s.ssmClients[region]
			_, err := regionClient.PutParameter(putParameterInput)
			if err != nil {
				abortOperation = true
			}
		}
	}

	// cleanup so that the Read() operation is idempotent
	if abortOperation {
		orderedRegions := getOrderedRegions()
		for _, region := range orderedRegions {
			regionClient := s.ssmClients[region]
			deleteParameterInput := &ssm.DeleteParameterInput{
				Name: aws.String(getParamNameFromName(id)),
			}
			_, err := regionClient.DeleteParameter(deleteParameterInput)
			if err != nil {
				return fmt.Errorf("Error during cleanup of secret creation for (%s). try again. error: %s", region, err)
			}
		}
		return fmt.Errorf("Error creating secret (%s). reverted all PutParameter operations. try creating secret again", id)
	}

	return nil
}

// Read a Secret from the store. Returns the latest version of the secret.
func (s *ParameterStore) Read(id SecretIdentifier) (Secret, error) {
	var resp *ssm.GetParameterOutput
	regionalOutput, regionalErrors := s.readForAllRegions(getParamNameFromName(id))
	orderedRegions := getOrderedRegions()
	for _, region := range orderedRegions {
		err := regionalErrors[region]
		if err != nil {
			if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
				if awsErr.Code() == ssm.ErrCodeParameterNotFound {
					return Secret{}, &IdentifierNotFoundError{Identifier: id, Region: region}
				}
			}
			return Secret{}, fmt.Errorf("ParamStore error: %s. ", err)
		}
	}
	resp = regionalOutput[Region]
	return Secret{*resp.Parameter.Value, SecretMeta{Version: convertFromSSMVersion(int(*resp.Parameter.Version))}}, nil
}

// ReadVersion reads a specific version of a secret from the store.
// Version is 0-indexed
func (s *ParameterStore) ReadVersion(id SecretIdentifier, version int) (Secret, error) {
	var resp *ssm.GetParameterOutput
	regionalOutput, regionalErrors := s.readForAllRegions(getParamNameFromNameAtVersion(id, version))
	orderedRegions := getOrderedRegions()
	for _, region := range orderedRegions {
		err := regionalErrors[region]
		if err != nil {
			if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
				if awsErr.Code() == ssm.ErrCodeParameterNotFound {
					return Secret{}, &IdentifierNotFoundError{Identifier: id, Region: region}
				} else if awsErr.Code() == ssm.ErrCodeParameterVersionNotFound {
					return Secret{}, &VersionNotFoundError{Identifier: id, Version: version}
				}
			}
			return Secret{}, fmt.Errorf("ParamStore error: %s. ", err)
		}
	}
	resp = regionalOutput[Region]
	return Secret{*resp.Parameter.Value, SecretMeta{Version: convertFromSSMVersion(int(*resp.Parameter.Version))}}, nil
}

// Update updates a Secret from the store and increments version number.
func (s *ParameterStore) Update(id SecretIdentifier, value string) (Secret, error) {
	name := getParamNameFromName(id)
	putParameterInput := &ssm.PutParameterInput{
		Name:      aws.String(name),
		Overwrite: aws.Bool(true), // true since we are updating existing secret
		Type:      aws.String(ssm.ParameterTypeSecureString),
		Value:     aws.String(value),
	}

	var abortOperation bool
	var failedRegions []string
	oldSecretValue, err := s.Read(id)
	if err != nil {
		return Secret{}, err
	}

	orderedRegions := getOrderedRegions()
	for _, region := range orderedRegions {
		regionClient := s.ssmClients[region]
		_, err := regionClient.PutParameter(putParameterInput)
		// If any region fails, we will retry one more time. If retry fails, this Update operation fails.
		// This guarantee the invariant that the all secret values are consistent across regions.
		if err != nil {
			failedRegions = append(failedRegions, region)
		}
	}

	abortOperation = false
	// lets try one more time for the failed regions
	if len(failedRegions) > 0 {
		for _, region := range failedRegions {
			regionClient := s.ssmClients[region]
			_, err := regionClient.PutParameter(putParameterInput)
			if err != nil {
				abortOperation = true
			}
		}
	}

	// cleanup so that Update is idempotent
	if abortOperation {
		orderedRegions := getOrderedRegions()
		for _, region := range orderedRegions {
			regionClient := s.ssmClients[region]
			putParameterInput := &ssm.PutParameterInput{
				Name:      aws.String(name),
				Overwrite: aws.Bool(true), // true since we are reverting the update,
				Type:      aws.String(ssm.ParameterTypeSecureString),
				Value:     aws.String(oldSecretValue.Data),
			}
			_, err := regionClient.PutParameter(putParameterInput)
			if err != nil {
				return Secret{}, fmt.Errorf("error update secret for region(%s). try again. error: %s", region, err)
			}
		}
		return Secret{}, fmt.Errorf("error updating secret for (%s). try again", id)
	}

	return s.Read(id)
}

// List gets secrets within a namespace (env/service)>
func (s *ParameterStore) List(env Environment, service string) ([]SecretIdentifier, error) {
	id := SecretIdentifier{env, service, ""}
	namespace := getNamespace(id.EnvironmentString(), service)
	apiClient := s.ssmClients[Region]

	// Per https://docs.aws.amazon.com/systems-manager/latest/APIReference/API_DescribeParameters.html
	// DescribeParameters request results are returned on a best-effort basis. Hence, we need to rely on NextToken
	// to fully fetch teh list of parameters and additionally we have to retry multiple times to fully fetch all the parameters.
	// We retry 2 times and return the maximum of the results
	results := []SecretIdentifier{}
	retryCount := 2
	for i := 1; i <= retryCount; i++ {
		resultsPerTry := []SecretIdentifier{}
		hasNextToken := true
		nextTokenStr := ""
		for hasNextToken {
			describeParametersByPathInput := &ssm.DescribeParametersInput{
				ParameterFilters: []*ssm.ParameterStringFilter{
					&ssm.ParameterStringFilter{
						Key:    aws.String("Path"),
						Option: aws.String("Recursive"),
						Values: []*string{aws.String(namespace)},
					},
				},
				MaxResults: aws.Int64(s.maxResultsToQuery),
			}
			if nextTokenStr != "" {
				describeParametersByPathInput.NextToken = aws.String(nextTokenStr)
			}

			resp, err := apiClient.DescribeParameters(describeParametersByPathInput)
			if err != nil {
				return []SecretIdentifier{}, err
			}
			for _, result := range resp.Parameters {
				ident, err := getSecretIDFromParamName(*result.Name)
				if _, ok := err.(*CurrentDeployError); ok {
					// secrets that fail with CurrentDeployError are intended to be read by machines, and not returned for human consumption.
					continue
				}
				resultsPerTry = append(resultsPerTry, ident)
			}
			if resp.NextToken != nil && *resp.NextToken != "" {
				nextTokenStr = *resp.NextToken
			} else {
				hasNextToken = false
			}
			// Try not to overwhelm rate limits
			time.Sleep(100 * time.Millisecond)
		}
		if len(resultsPerTry) >= len(results) {
			results = resultsPerTry
		}
		// retry again in a second
		time.Sleep(1 * time.Second)
	}
	return results, nil
}

// ListAll gets all secrets within a environment (env)>
func (s *ParameterStore) ListAll(env Environment) ([]SecretIdentifier, error) {
	return s.List(env, "")
}

// History gets history for a secret, returning all versions from the store.
func (s *ParameterStore) History(id SecretIdentifier) ([]SecretMeta, error) {
	paramName := getParamNameFromName(id)
	getParamHistoryInput := &ssm.GetParameterHistoryInput{
		Name: aws.String(paramName),
	}
	apiClient := s.ssmClients[Region]
	results := []SecretMeta{}
	resp, err := apiClient.GetParameterHistory(getParamHistoryInput)
	if err != nil {
		if awsErr, ok := errors.Cause(err).(awserr.Error); ok {
			if awsErr.Code() == ssm.ErrCodeParameterNotFound {
				return results, &IdentifierNotFoundError{Identifier: id, Region: Region}
			}
		}
		return results, err
	}
	for _, history := range resp.Parameters {
		results = append(results, SecretMeta{
			Created: *history.LastModifiedDate,
			Version: convertFromSSMVersion(int(*history.Version)),
		})
	}
	return results, nil
}

// Delete deletes all versions of a secret
func (s *ParameterStore) Delete(id SecretIdentifier) error {
	deleteParameterInput := &ssm.DeleteParameterInput{
		Name: aws.String(getParamNameFromName(id)),
	}
	orderedRegions := getOrderedRegions()
	var failedRegions []string
	for _, region := range orderedRegions {
		regionClient := s.ssmClients[region]
		_, err := regionClient.DeleteParameter(deleteParameterInput)
		// If any region fails, add to the return list of errors and continue.
		if err != nil {
			failedRegions = append(failedRegions, region)
		}
	}
	// retry one more time
	if len(failedRegions) > 0 {
		for _, region := range failedRegions {
			regionClient := s.ssmClients[region]
			_, err := regionClient.DeleteParameter(deleteParameterInput)
			// If any region fails now, consider this Delete operation failed and return
			if err != nil {
				return fmt.Errorf("failed to delete secret from region %s. try again", region)
			}
		}
	}
	return nil
}

// NewParameterStore creates a secret store that points at ParameterStore
func NewParameterStore(maxResultsToQuery int64) *ParameterStore {
	return &ParameterStore{
		ssmClients:        getAPIClients(),
		maxResultsToQuery: maxResultsToQuery,
	}
}

// readForAllRegions reads given secret from all AWS regions and return status for the corresponding region.
// If a read for a region fails, the corresponding error is returned
func (s *ParameterStore) readForAllRegions(paramName string) (map[string]*ssm.GetParameterOutput, map[string]error) {
	output := make(map[string]*ssm.GetParameterOutput)
	errors := make(map[string]error)
	getParameterInput := &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(true),
	}
	orderedRegions := getOrderedRegions()
	for _, region := range orderedRegions {
		regionClient := s.ssmClients[region]
		resp, err := regionClient.GetParameter(getParameterInput)
		output[region] = resp
		errors[region] = err
	}
	return output, errors
}
