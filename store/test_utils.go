package store

import (
	"math/rand"
	"os"
	"time"
)

func isCI() bool {
	return os.Getenv("CI") == "true"
}

// Stores returns all implemented SecretStores memory and param
func Stores() map[string]SecretStore {
	var stores = make(map[string]SecretStore)
	stores["Memory"] = NewMemoryStore()
	// don't test in CI environment, since it would require a role assumption we
	// don't want to support
	if !isCI() {
		// maxResultsToQuery = 5 so that we test the pagination logic of the List command, in the ci-test env
		stores["Paramstore"] = NewParameterStore(5, "ci-test", false)
	}
	return stores
}

// GetRandomTestSecretIdentifier returns a random key in the ci-test environment
func GetRandomTestSecretIdentifier() SecretIdentifier {
	return SecretIdentifier{Environment: CITestEnvironment, Service: "test" + randSeq(2), Key: randSeq(10)}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
