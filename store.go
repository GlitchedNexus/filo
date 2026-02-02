package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const defaultRootFolderName = "ggnetwork"

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	FileName string
}

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
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

type StoreOptions struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOptions
	s3Client *s3.Client
}

func NewStore(options StoreOptions) *Store {
	if options.PathTransformFunc == nil {
		options.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(options.Root) == 0 {
		options.Root = defaultRootFolderName
	}

	cfg, _ := config.LoadDefaultConfig(context.TODO())
	return &Store{
		StoreOptions: options, // Changed from StoreOpts: opts
		s3Client:     s3.NewFromConfig(cfg),
	}
}

// MoveToS3 transfers a local file to S3 cold storage and deletes the local copy.
func (s *Store) MoveToS3(id string, key string) error {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return err
	}
	defer file.Close()

	bucketName := os.Getenv("S3_BUCKET_NAME")
	s3Key := fmt.Sprintf("%s/%s", id, key)

	_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &s3Key,
		Body:   file,
	})
	
	if err != nil {
		return err
	}

	return os.Remove(fullPathWithRoot)
}

// Read tries to fetch from local storage first, then falls back to S3.
func (s *Store) Read(id string, key string) (int64, io.ReadCloser, error) {
	// 1. Try local storage
	size, r, err := s.readStream(id, key)
	if err == nil {
		return size, r, nil
	}

	// 2. If not on disk, check S3
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("[%s] not found locally, attempting to thaw from S3...", key)
		
		s3Key := fmt.Sprintf("%s/%s", id, key)
		output, err := s.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
			Key:    aws.String(s3Key),
		})
		if err != nil {
			return 0, nil, fmt.Errorf("file not found locally or in S3: %w", err)
		}

		return *output.ContentLength, output.Body, nil
	}

	return 0, nil, err
}

func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)

	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.FileName)
	}()

	firstPathNamewithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNamewithRoot)

}

func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)

	if err != nil {
		return 0, err
	}

	n, err := copyDecrypt(encKey, r, f)
	return int64(n), nil
}

func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)

	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.PathName)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPath := pathKey.FullPath()
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, fullPath)

	return os.Create(fullPathWithRoot)
}

func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)

	if err != nil {
		return 0, err
	}

	return io.Copy(f, r)
}

func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())
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
