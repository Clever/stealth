package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Other possible tests
// - keys shouldn't be case sensitive
// - should fail if key contains invalid chars / format

func TestCreateRead(t *testing.T) {
	store := NewMemoryStore()

	t.Log("no secrets exist, to begin")
	_, err := store.Read("foo")
	assert.Error(t, err)
	assert.Equal(t, err, &KeyNotFoundError{Key: "foo"})

	t.Log("write a secret")
	data := SecretData("bar")
	err = store.Create("foo", data)
	assert.NoError(t, err)

	t.Log("we should now be able to read it")
	secret, err := store.Read("foo")
	assert.NoError(t, err)
	assert.Equal(t, secret.Data, data)

	t.Log("creating the secret again fails")
	err = store.Create("foo", data)
	assert.Error(t, err)
	assert.Equal(t, err, &KeyAlreadyExistsError{Key: "foo"})
}

func TestUpdateHistory(t *testing.T) {
	store := NewMemoryStore()

	t.Log("no secrets exist, to begin")
	_, err := store.Read("foo")
	assert.Error(t, err)
	assert.Equal(t, err, &KeyNotFoundError{Key: "foo"})
	_, err = store.History("foo")
	assert.Error(t, err)
	assert.Equal(t, err, &KeyNotFoundError{Key: "foo"})
	data1 := SecretData("bar")
	_, err = store.Update("foo", data1)
	assert.Error(t, err)
	assert.Equal(t, err, &KeyNotFoundError{Key: "foo"})

	t.Log("STEP 1: write a secret")
	err = store.Create("foo", data1)
	assert.NoError(t, err)

	t.Log("we should now see one version in History")
	hist1, err := store.History("foo")
	assert.NoError(t, err)
	assert.Equal(t, len(hist1), 1)
	assert.Equal(t, hist1[0].Data, data1)

	t.Log("Read should return the most recent secret")
	read1, err := store.Read("foo")
	assert.NoError(t, err)
	assert.Equal(t, read1.Data, data1)

	t.Log("STEP 2: overwrite the secret")
	data2 := SecretData("bibimbap")
	_, err = store.Update("foo", data2)
	assert.NoError(t, err)

	t.Log("we should now see two versions in History")
	hist2, err := store.History("foo")
	assert.NoError(t, err)
	assert.Equal(t, len(hist2), 2)
	assert.Equal(t, hist2[0].Data, data1)
	assert.Equal(t, hist2[1].Data, data2)

	t.Log("Read should return the most recent secret")
	read2, err := store.Read("foo")
	assert.NoError(t, err)
	assert.Equal(t, read2.Data, data2)
}
