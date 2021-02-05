package store

import (
	"fmt"
	"log"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Other possible tests
// - keys shouldn't be case sensitive
// - should fail if key contains invalid chars / format (must be [a-z0-9-])

// TestMain runs deleteSecretsFromStores before running the tests
func TestMain(m *testing.M) {
	deleteSecretsFromStores()
	code := m.Run()
	os.Exit(code)
}

// deleteSecretsFromStores deletes all secrets from all stores
func deleteSecretsFromStores() {
	log.Println("Deleting secrets from all stores...")
	for name, store := range Stores() {
		ids, err := store.ListAll(CITestEnvironment)
		if err != nil {
			errMsg := fmt.Errorf("Unable to get secrets from %s. error: %s", name, err)
			fmt.Println(errMsg)
		} else if len(ids) > 0 {
			for _, id := range ids {
				store.Delete(id)
			}
			log.Printf("All secrets from %s successfully deleted", name)
		} else {
			log.Printf("No secrets in %s", name)
		}
	}
	log.Println("End of deleting secrets from all stores")
}

func TestIdentifer(t *testing.T) {
	id := SecretIdentifier{Environment: CITestEnvironment, Service: "service", Key: "foo"}
	assert.Equal(t, id.String(), "ci-test.service.foo")
	assert.Equal(t, fmt.Sprintf("%s", id), "ci-test.service.foo")
}

func TestStringToSecretIdentifier(t *testing.T) {
	t.Log("works for all valid environments")
	for _, env := range []Environment{CITestEnvironment, DevelopmentEnvironment, ProductionEnvironment} {
		id := SecretIdentifier{Environment: env, Service: "service", Key: "foo"}
		idFromString, err := stringToSecretIdentifier(id.String())
		assert.NoError(t, err)
		assert.Equal(t, idFromString, id)
	}

	// TODO: consider creating SecretIdentifier's in a NewSecretIdentifier constructor and put validation there
	t.Log("errors on invalid environment")
	id := SecretIdentifier{Environment: -1, Service: "service", Key: "foo"}
	_, err := stringToSecretIdentifier(id.String())
	assert.Error(t, err)

	id = SecretIdentifier{Environment: CITestEnvironment, Service: "service", Key: "foo.bar"}
	idString := id.String()
	t.Log(fmt.Sprintf("errors on '.' in key name: %s", idString))
	_, err = stringToSecretIdentifier(idString)
	assert.Error(t, err)
}

func TestCreateRead(t *testing.T) {
	id := GetRandomTestSecretIdentifier()
	for name, store := range Stores() {
		defer store.Delete(id)
		region := Region
		if name == "Memory" {
			region = ""
		}
		t.Logf("---- %s ----\n", name)
		t.Log("no secrets exist, to begin")
		_, err := store.Read(id)
		assert.Error(t, err)
		assert.Equal(t, err, &IdentifierNotFoundError{Identifier: id, Region: region})

		t.Log("write a secret")
		data := "bar"
		err = store.Create(id, data)
		assert.NoError(t, err)

		t.Log("we should now be able to read it")
		secret, err := store.Read(id)
		assert.NoError(t, err)
		assert.Equal(t, secret.Data, data)

		t.Log("creating the secret again fails")
		err = store.Create(id, data)
		assert.Error(t, err)
		assert.Equal(t, err, &IdentifierAlreadyExistsError{Identifier: id})
	}
}

func TestCreateList(t *testing.T) {
	// service 1 has 2 identifiers
	s1id1 := GetRandomTestSecretIdentifier()
	test1 := s1id1.Service
	s1id2 := GetRandomTestSecretIdentifier()
	s1id2.Service = test1

	// service 2 has 1 identifier
	s2id1 := GetRandomTestSecretIdentifier()
	test2 := s2id1.Service

	data := "foo"
	for name, store := range Stores() {
		defer store.Delete(s1id1)
		defer store.Delete(s1id2)
		defer store.Delete(s2id1)
		t.Logf("---- %s ----\n", name)

		t.Log("errors if given invalid environment")
		_, err := store.List(-1, test1)
		assert.Error(t, err)

		t.Log("no secrets exist, to begin")
		ids, err := store.List(CITestEnvironment, test1)
		assert.NoError(t, err)
		assert.Equal(t, len(ids), 0)

		t.Log("write 1st secret for service 1")
		err = store.Create(s1id1, data)
		assert.NoError(t, err)

		t.Log("we should now be able to list 1 secret id")
		ids, err = store.List(CITestEnvironment, test1)
		assert.NoError(t, err)
		assert.Equal(t, len(ids), 1)
		assert.Equal(t, ids, []SecretIdentifier{s1id1})

		t.Log("write 2nd secret for service 1")
		err = store.Create(s1id2, data)
		assert.NoError(t, err)

		t.Log("write 1st secret for service 2")
		err = store.Create(s2id1, data)
		assert.NoError(t, err)

		// so that the Create calls go through
		time.Sleep(1 * time.Second)

		t.Log("we should now be able to list 2 secret ids for service 1")
		ids, err = store.List(CITestEnvironment, test1)
		assert.NoError(t, err)
		expectedIds := []SecretIdentifier{s1id1, s1id2}
		sort.Sort(ByIDString(expectedIds))
		assert.Equal(t, ids, expectedIds)

		t.Log("we should now be able to list 1 secret id for service 2")
		ids, err = store.List(CITestEnvironment, test2)
		assert.NoError(t, err)
		assert.Equal(t, ids, []SecretIdentifier{s2id1})

		t.Log("we should now be able to list all secrets ids for service 1 and 2")
		ids, err = store.ListAll(CITestEnvironment)
		assert.NoError(t, err)
		expectedIds = []SecretIdentifier{s1id1, s1id2, s2id1}
		sort.Sort(ByIDString(expectedIds))
		assert.Equal(t, ids, expectedIds)
	}
}

func TestCreateListMultipleTimes(t *testing.T) {
	data := "foo"
	limit := 10
	for name, store := range Stores() {
		t.Logf("---- %s ----\n", name)
		testServiceName := GetRandomTestSecretIdentifier().Service

		for i := 1; i <= limit; i++ {
			newID := GetRandomTestSecretIdentifier()
			newID.Service = testServiceName
			t.Log(fmt.Sprintf("write secret #%d %s for service", i, newID.String()))
			err := store.Create(newID, data)
			assert.NoError(t, err)

			ids, err := store.List(CITestEnvironment, testServiceName)
			assert.NoError(t, err)
			assert.True(t, len(ids) <= i)
			defer store.Delete(newID)
		}

		t.Log("we should now be able to list secrets and match the count")
		ids, err := store.List(CITestEnvironment, testServiceName)
		assert.NoError(t, err)
		assert.Equal(t, len(ids), limit)
	}
	return
}

func TestUpdateHistory(t *testing.T) {
	id := GetRandomTestSecretIdentifier()
	for name, store := range Stores() {
		defer store.Delete(id)
		region := Region
		if name == "Memory" {
			region = ""
		}
		t.Logf("---- %s ----\n", name)
		t.Log("no secrets exist, to begin")
		_, err := store.Read(id)
		assert.Error(t, err)
		assert.Equal(t, err, &IdentifierNotFoundError{Identifier: id, Region: region})
		_, err = store.History(id)
		assert.Error(t, err)
		assert.Equal(t, err, &IdentifierNotFoundError{Identifier: id, Region: region})
		data1 := "bar"
		_, err = store.Update(id, data1)
		assert.Error(t, err)
		assert.Equal(t, err, &IdentifierNotFoundError{Identifier: id, Region: region})

		t.Log("STEP 1: write a secret")
		err = store.Create(id, data1)
		assert.NoError(t, err)

		t.Log("we should now see one version in History")
		hist1, err := store.History(id)
		assert.NoError(t, err)
		assert.Equal(t, len(hist1), 1)
		assert.Equal(t, hist1[0].Version, 0)

		t.Log("Read should return the most recent secret")
		read1, err := store.Read(id)
		assert.NoError(t, err)
		assert.Equal(t, read1.Data, data1)

		t.Log("STEP 2: overwrite the secret")
		data2 := "bibimbap"
		_, err = store.Update(id, data2)
		assert.NoError(t, err)

		t.Log("we should now see two versions in History")
		hist2, err := store.History(id)
		assert.NoError(t, err)
		assert.Equal(t, len(hist2), 2)
		assert.Equal(t, hist2[0].Version, 0)
		assert.Equal(t, hist2[1].Version, 1)

		t.Log("Read should return the most recent secret")
		read2, err := store.Read(id)
		assert.NoError(t, err)
		assert.Equal(t, read2.Data, data2)
		t.Log("we should now be able to read the previous version")
		readVersion, err := store.ReadVersion(id, 0)
		assert.NoError(t, err)
		assert.Equal(t, readVersion.Data, data1)

		t.Log("we should now be able to read the current version")
		readVersion, err = store.ReadVersion(id, 1)
		assert.NoError(t, err)
		assert.Equal(t, readVersion.Data, data2)

		t.Log("we should not be able to read an non-existant version")
		_, err = store.ReadVersion(id, 2)
		assert.Error(t, err)
		assert.Equal(t, err, &VersionNotFoundError{Identifier: id, Version: 2})

		t.Log("we should not be able to read a version less than zero")
		readVersion, err = store.ReadVersion(id, -1)
		assert.Error(t, err)
		assert.Equal(t, err, &VersionNotFoundError{Identifier: id, Version: -1})
	}
}

func TestDelete(t *testing.T) {
	id := GetRandomTestSecretIdentifier()
	for name, store := range Stores() {
		defer store.Delete(id)
		region := Region
		if name == "Memory" {
			region = ""
		}
		t.Logf("---- %s ----\n", name)
		t.Log("creating secret")
		data1 := "bar"
		err := store.Create(id, data1)
		assert.NoError(t, err)
		t.Log("deleting secret")
		err = store.Delete(id)
		assert.NoError(t, err)

		t.Log("we should not be able to read")
		_, err = store.Read(id)
		assert.Error(t, err)
		assert.Equal(t, err, &IdentifierNotFoundError{Identifier: id, Region: region})

		t.Log("we should see no history")
		_, err = store.History(id)
		assert.Error(t, err)
		assert.Equal(t, err, &IdentifierNotFoundError{Identifier: id, Region: region})
	}
}
