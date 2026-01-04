package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "mybestpicture"
	pathkey := CASPathTransformFunc(key)

	expectedFileName := "be17b32c2870b1c0c73b59949db6a3be7814dd23"
	expectedPathName := "be17b/32c28/70b1c/0c73b/59949/db6a3/be781/4dd23"

	if pathkey.PathName != expectedPathName {
		t.Errorf("have %s want %s", pathkey.PathName, expectedPathName)
	}

	if pathkey.FileName != expectedFileName {
		t.Errorf("have %s want %s", pathkey.FileName, expectedFileName)
	}
}

func TestStore(t *testing.T) {
	id := generateID()
	s := newStore()
	defer tearDown(t, s)

	count := 50

	for i := range count {
		key := fmt.Sprintf("foo_%d", i)
		data := []byte("some jpg bytes")

		if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); !ok {
			t.Errorf("expected to have key %s", key)
		}

		_, r, err := s.Read(id, key)

		if err != nil {
			t.Error(err)
		}

		b, err := io.ReadAll(r)

		if string(b) != string(data) {
			t.Errorf("want %s have %s", data, b)
		}

		if err := s.Delete(id, key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); ok {
			t.Errorf("expected to NOT have key %s", key)
		}

	}
}

func newStore() *Store {
	options := StoreOptions{
		PathTransformFunc: CASPathTransformFunc,
	}

	return NewStore(options)
}

func tearDown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
