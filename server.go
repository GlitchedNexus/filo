package main

import (
	"fmt"
	"log"

	"github.com/GlitchedNexus/filo/p2p"
)

type FileServerOptions struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOptions

	store  *Store
	quitch chan struct{}
}

func NewFileServer(options FileServerOptions) *FileServer {

	storeOptions := StoreOptions{
		Root:              options.StorageRoot,
		PathTransformFunc: options.PathTransformFunc,
	}

	return &FileServer{
		FileServerOptions: options,
		store:             NewStore(storeOptions),
		quitch:            make(chan struct{}),
	}
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action.")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Println(msg)
		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.loop()

	return nil
}
