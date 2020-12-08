package store

import (
	"math/rand"
	"time"
)

// Stores returns all implemented SecretStores
func Stores() map[string]SecretStore {
	var stores = make(map[string]SecretStore)
	stores["Memory"] = NewMemoryStore()
	stores["Unicreds"] = NewUnicredsStore()
	// maxResultsToQuery = 5 so that we test the pagination logic of the List command
	stores["Paramstore"] = NewParameterStore(5)
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
