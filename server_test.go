package main

import (
	"bytes"
	"fmt"
	"log"
	"testing"
	"time"
)

func TestSystem(t *testing.T) {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", "")
	s3 := makeServer(":6000", ":3000", ":4000")

	go func() { log.Fatal(s1.Start()) }()
	time.Sleep(500 * time.Millisecond)
	go func() { log.Fatal(s2.Start()) }()

	time.Sleep(2 * time.Second)

	go func() { log.Fatal(s3.Start()) }()
	time.Sleep(2 * time.Second)
	fmt.Println("s3 peers:", len(s3.peers))

	for i := range 5 {
		key := fmt.Sprintf("picture_%d.png", i)
		data := bytes.NewReader([]byte("my big data file here!"))
		s3.Store(key, data)

		time.Sleep(time.Millisecond * 500)

		// if err := s3.store.Delete(s3.ID, key); err != nil {
		// 	log.Fatal(err)
		// }

		// r, err := s3.Get(key)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// b, err := io.ReadAll(r)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// fmt.Println(string(b))

		if err := s3.Delete(key); err != nil {
			log.Fatal(err)
		}
	}
}