package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Starting Go Alpine SaaS Starter...")

	// TODO: Load configuration (Phase 0)
	// TODO: Set up logger (Phase 0)
	// TODO: Establish database connection (Phase 1)
	// TODO: Set up router (Phase 4)
	// TODO: Define handlers (Phase 4)
	// TODO: Set up server (Phase 4)

	// Placeholder listener
	addr := ":4000"
	log.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}
