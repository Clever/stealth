package store

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Other possible tests
// - keys shouldn't be case sensitive
// - should fail if key contains invalid chars / format

func TestCreateRead(t *testing.T) {
	id := SecretIdentifier{Environment: "env", Service: "service", Key: "foo"}
	assert.Equal(t, id.String(), "env.service.foo")
	assert.Equal(t, fmt.Sprintf("%s", id), "env.service.foo")
	store := NewMemoryStore()

	t.Log("no secrets exist, to begin")
	_, err := store.Read(id)
	assert.Error(t, err)
	assert.Equal(t, err, &IdentifierNotFoundError{id})

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
	assert.Equal(t, err, &IdentifierAlreadyExistsError{id})
}

func TestUpdateHistory(t *testing.T) {
	id := SecretIdentifier{Environment: "env", Service: "service", Key: "foo"}
	store := NewMemoryStore()

	t.Log("no secrets exist, to begin")
	_, err := store.Read(id)
	assert.Error(t, err)
	assert.Equal(t, err, &IdentifierNotFoundError{id})
	_, err = store.History(id)
	assert.Error(t, err)
	assert.Equal(t, err, &IdentifierNotFoundError{id})
	data1 := "bar"
	_, err = store.Update(id, data1)
	assert.Error(t, err)
	assert.Equal(t, err, &IdentifierNotFoundError{id})

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
