package util

import (
	"sort"
	"testing"

	"github.com/Clever/stealth/store"
	"github.com/stretchr/testify/assert"
)

func TestFindDupes(t *testing.T) {
	id1 := store.GetRandomTestSecretIdentifier()
	id2 := store.GetRandomTestSecretIdentifier()
	id3 := store.GetRandomTestSecretIdentifier()
	envs := []store.Environment{store.CITestEnvironment}
	for name, s := range store.Stores() {
		defer s.Delete(id1)
		defer s.Delete(id2)
		defer s.Delete(id3)
		t.Logf("---- %s ----\n", name)
		t.Log("creating some duplicate secrets")
		data1 := "bar1"
		err := s.Create(id1, data1)
		err = s.Create(id2, data1)
		data2 := "bar2"
		err = s.Create(id3, data2)
		t.Log("should be able to find dupes for either duplicate value")
		dupes, err := FindDupes(s, id1, envs)
		assert.NoError(t, err)
		expectedIds := []store.SecretIdentifier{id1, id2}
		sort.Sort(store.ByIDString(expectedIds))
		assert.Equal(t, dupes, expectedIds)
		dupes, err = FindDupes(s, id2, envs)
		assert.NoError(t, err)
		assert.Equal(t, dupes, expectedIds)
		t.Log("a secret with no duplicate should only return itself")
		dupes, err = FindDupes(s, id3, envs)
		assert.NoError(t, err)
		assert.Equal(t, dupes, []store.SecretIdentifier{id3})
	}
}
