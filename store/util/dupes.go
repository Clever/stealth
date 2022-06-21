package util

import (
	"log"
	"time"

	"github.com/Clever/stealth/store"
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
		log.Printf("reading from %s\n", e.String())
		ids, err := s.ListAll(e)
		log.Printf("total secrets: %d\n", len(ids))
		if err != nil {
			return []store.SecretIdentifier{}, err
		}
		for num, id := range ids {
			if num%100 == 0 {
				log.Printf("reading %04d/%04d\n", num, len(ids))
			}
			// With Parameter Store, the maximal normal limit is 40 requests per second.
			// 1000 ms / 67 ms ~= 15 secrets per second.
			time.Sleep(67 * time.Millisecond)
			newSecret, err := s.Read(id)
			if err != nil {
				// We assume that any missing secret isn't an issue
				// with the duplicate checking. We'll log instead of erroring.
				log.Printf("error reading secret: %v\n", err)
			}
			if newSecret.Data == secret.Data {
				dupes = append(dupes, id)
			}
		}
	}
	return dupes, nil
}
