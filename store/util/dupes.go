package util

import (
	"github.com/Clever/stealth/store"
	"time"
)

// FindDupes finds all secrets that match a secret with a specified identifier, and optionally
// replace that value with a new value
func FindDupes(s store.SecretStore, id store.SecretIdentifier, envs []store.Environment) ([]store.SecretIdentifier, error) {
	secret, err := s.Read(id)
	if err != nil {
		return []store.SecretIdentifier{}, err
	}
	var dupes []store.SecretIdentifier
	for _, e := range envs {
		ids, err := s.ListAll(e)
		if err != nil {
			return []store.SecretIdentifier{}, err
		}
		for _, id := range ids {
			// Try not to overwhelm rate limits
			time.Sleep(100 * time.Millisecond)
			newSecret, err := s.Read(id)
			if err != nil {
				return []store.SecretIdentifier{}, err
			}
			if newSecret.Data == secret.Data {
				dupes = append(dupes, id)
			}
		}
	}
	return dupes, nil
}
