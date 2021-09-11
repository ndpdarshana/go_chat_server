package main

import (
	"log"
	"os"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Fatalf("service account key not provided")
	}

	s := newServer(string(args[1]))
	s.start(":8080")
}
