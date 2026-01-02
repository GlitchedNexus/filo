package main

import (
	"bytes"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "mybestpicture"
	pathkey := CASPathTransformFunc(key)

	expectedOriginalKey := "cf5d4b01c4d9438c22c56c832f83bd3e8c6304f9"
	expectedPathName := "cf5d4/b01c4/d9438/c22c5/6c832/f83bd/3e8c6/304f9"

	if pathkey.PathName != expectedPathName {
		t.Errorf("have %s want %s", pathkey.PathName, expectedPathName)
	}

	if pathkey.FileName != expectedOriginalKey {
		t.Errorf("have %s want %s", pathkey.FileName, expectedOriginalKey)
	}
}

func TestStoreDelete(t *testing.T) {
	options := StoreOptions{
		PathTransformFunc: CASPathTransformFunc,
	}

	s := NewStore(options)
	key := "myspecials"
	data := []byte("some jpg bytes")

	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}

func TestStore(t *testing.T) {
	options := StoreOptions{
		PathTransformFunc: CASPathTransformFunc,
	}

	s := NewStore(options)
	key := "myspecials"
	data := []byte("some jpg bytes")

	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if ok := s.Has(key); !ok {
		t.Errorf("expected to have key %s", key)
	}

	r, err := s.Read(key)

	if err != nil {
		t.Error(err)
	}

	b, err := io.ReadAll(r)

	if string(b) != string(data) {
		t.Errorf("want %s have %s", data, b)
	}

	s.Delete(key)
}
