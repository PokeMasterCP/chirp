package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func main() {
	PORT := "8080"
	mux := http.NewServeMux()
	cfg := apiConfig{}

	mux.HandleFunc("GET /api/healthz", handlerHealth)
	mux.HandleFunc("GET /admin/metrics", cfg.handlerNumHits)
	mux.HandleFunc("POST /admin/reset", cfg.handlerResetHits)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))

	s := &http.Server{
		Addr:    ":" + PORT,
		Handler: mux,
	}

	fmt.Printf("starting server on %s...\n", PORT)
	log.Fatal(s.ListenAndServe())
}
