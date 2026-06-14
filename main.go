package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	PORT := "8080"
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(".")))
	s := &http.Server{
		Addr:    ":" + PORT,
		Handler: mux,
	}

	fmt.Printf("starting server on %s...\n", PORT)
	log.Fatal(s.ListenAndServe())
}
