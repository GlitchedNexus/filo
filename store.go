package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "ggnetwork"

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

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		FileName: key,
	}
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.FileName)
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")

	if len(paths) == 0 {
		return ""
	}

	return paths[0]
}

// Root:
//   - contains the folde rname of the root, containing all the files/folders
//     of the system.
type StoreOptions struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOptions
}

func NewStore(options StoreOptions) *Store {

	if options.PathTransformFunc == nil {
		options.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(options.Root) == 0 {
		options.Root = defaultRootFolderName
	}

	return &Store{
		StoreOptions: options,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)

	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)

	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("delted [%s] from disk", pathKey.FileName)
	}()

	firstPathNamewithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNamewithRoot)

}

// FIX: Instead of copying directly to a reader we first copy
// this into a buffer. Maybe just return the File from readStream?
func (s *Store) Read(key string) (int64, io.Reader, error) {
	n, f, err := s.readStream(key)

	if err != nil {
		return n, nil, err
	}

	defer f.Close()

	buf := new(bytes.Buffer)

	_, err = io.Copy(buf, f)

	return n, buf, nil
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	// fi, err := os.Stat(fullPathWithRoot)

	// if err != nil {
	// 	return 0, nil, err
	// }

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	pathKey := s.PathTransformFunc(key)

	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.PathName)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return 0, err
	}

	fullPath := pathKey.FullPath()
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, fullPath)

	f, err := os.Create(fullPathWithRoot)

	if err != nil {
		return 0, err
	}

	n, err := io.Copy(f, r)

	if err != nil {
		return 0, err
	}

	return n, nil
}
