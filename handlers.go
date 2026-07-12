package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func handlerHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerNumHits(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	msg := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load())
	w.Write([]byte(msg))
}

func (cfg *apiConfig) handlerResetHits(w http.ResponseWriter, req *http.Request) {
	cfg.fileServerHits.Store(0)
}

func handlerValidateChirp(w http.ResponseWriter, req *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	type errorResponse struct {
		Error string `json:"error"`
	}

	type validResponse struct {
		Valid bool `json:"valid"`
	}

	params, err := helperReadJSON[chirp](req.Body)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to unmarshal json request: %s", err)
		log.Println(errorMessage)
		response := errorResponse{Error: errorMessage}
		helperRespondWithJSON(response, w, http.StatusInternalServerError)
		return
	}

	if len(params.Body) > 140 {
		response := errorResponse{Error: "Chirp is too long"}
		helperRespondWithJSON(response, w, http.StatusBadRequest)
		return
	}

	response := validResponse{Valid: true}
	helperRespondWithJSON(response, w, http.StatusOK)
}

func helperReadJSON[T any](req io.Reader) (T, error) {
	decoder := json.NewDecoder(req)
	var params T

	err := decoder.Decode(&params)
	if err != nil {
		return params, err
	}
	return params, nil
}

func helperRespondWithJSON[T any](payload T, w http.ResponseWriter, statusCode int) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to marshal response: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}
