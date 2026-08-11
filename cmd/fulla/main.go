package main

import (
	"log"
	"runtime/debug"

	"github.com/ChancellorCookie/fulla/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: %v\n%s", r, debug.Stack())
		}
	}()
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
