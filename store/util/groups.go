package util

import (
	"fmt"
	"github.com/Clever/stealth/store"
	"time"
)

// FindGroups groups all secrets that match each other based on the first encountered identifier.
func FindGroups(s store.SecretStore, envs []store.Environment, groupsFile string) ([][]store.SecretIdentifier, []error) {
	if groupsFile != "" {
		return findGroupsFromFile(s, envs, groupsFile)
	}
	return findGroupsDefault(s, envs)
}

func findGroupsDefault(s store.SecretStore, envs []store.Environment) ([][]store.SecretIdentifier, []error) {
	groupsWithData := map[string][]store.SecretIdentifier{}
	groups := [][]store.SecretIdentifier{}
	errors := []error{}

	// load all secrets into groups by data
	for _, e := range envs {
		ids, err := s.ListAll(e)
		if err != nil {
			fmt.Printf("error listing secrets for env %s\n", e)
			errors = append(errors, err)
		}
		for _, id := range ids {
			// Try not to overwhelm rate limits
			time.Sleep(100 * time.Millisecond)
			current, err := s.Read(id)
			if err != nil {
				// add to errors
				fmt.Printf("error reading secret %+v\n", id)
				errors = append(errors, err)
			} else {
				_, ok := groupsWithData[current.Data]
				if ok {
					groupsWithData[current.Data] = append(groupsWithData[current.Data], id)
				} else {
					groupsWithData[current.Data] = []store.SecretIdentifier{id}
				}
			}
		}
	}
	// convert to not include data in output
	for _, group := range groupsWithData {
		groups = append(groups, group)
	}
	return groups, nil
}

func findGroupsFromFile(s store.SecretStore, envs []store.Environment, groupsFile string) ([][]store.SecretIdentifier, []error) {
	groupsWithData := map[string][]store.SecretIdentifier{}
	groups := [][]store.SecretIdentifier{}
	errors := []error{}

	// load secrets from apps we care about into groups by data
	var secretKeyStrings []string
	bytes, err := ioutil.ReadFile(groupsFile)
	if err != nil {
		return [][]store.SecretIdentifier{}, []error{err}
	}
	err = json.Unmarshal(bytes, &secretKeyStrings)
	if err != nil {
		return [][]store.SecretIdentifier{}, []error{err}
	}
	secretIDs := []SecretIdentifier{}

	for _, secret := range secretKeyStrings {
		secretID, err := store.StringToSecretID(secret)
		if err != nil {
			fmt.Printf("error listing secret %+v\n", secret)
			errors = append(errors, err)
		}
		secretIDs = append(secretIDs, secretID)
	}
	for _, id := range secretIDs {
		// Try not to overwhelm rate limits
		time.Sleep(100 * time.Millisecond)
		current, err := s.Read(id)
		if err != nil {
			// add to errors
			fmt.Printf("error reading secret %+v\n", id)
			errors = append(errors, err)
		} else {
			_, ok := groupsWithData[current.Data]
			if !ok {
				groupsWithData[current.Data] = []store.SecretIdentifier{}
			}
		}
	}

	// add additional secrets from other apps
	for _, e := range envs {
		allIDs, err := s.ListAll(e)
		if err != nil {
			fmt.Printf("error listing secrets for env %s\n", e)
			errors = append(errors, err)
		}
		for _, anyID := range allIDs {
			// Try not to overwhelm rate limits
			time.Sleep(100 * time.Millisecond)
			current, err := s.Read(anyID)
			if err != nil {
				// add to errors
				fmt.Printf("error reading secret %+v\n", anyID)
				errors = append(errors, err)
			} else {
				_, ok := groupsWithData[current.Data]
				if ok {
					groupsWithData[current.Data] = append(groupsWithData[current.Data], anyID)
				}
			}
		}
	}

	// convert to not include data in output
	for _, group := range groupsWithData {
		groups = append(groups, group)
	}
	return groups, nil
}
