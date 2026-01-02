package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	FileName string
}

// Each key is transformed to some path on disk
func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))

	// To convert a byte array to a slice we can do
	// ex. 20[byte] => []byte => [:]
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5

	sliceLength := len(hashStr) / blockSize

	paths := make([]string, sliceLength)

	for i := range sliceLength {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		FileName: hashStr,
	}
}

var DefaultPathTransformFunc = func(key string) string {
	return key
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.FileName)
}

type StoreOptions struct {
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOptions
}

func NewStore(options StoreOptions) *Store {
	return &Store{
		StoreOptions: options,
	}
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	buf := new(bytes.Buffer)

	_, err = io.Copy(buf, f)

	return buf, nil
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	return os.Open(pathKey.FullPath())
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(pathKey.PathName, os.ModePerm); err != nil {
		return err
	}

	fullPath := pathKey.FullPath()

	f, err := os.Create(fullPath)

	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)

	if err != nil {
		return err
	}

	log.Printf("written (%d) bytes to disk: %s", n, fullPath)

	return nil
}
