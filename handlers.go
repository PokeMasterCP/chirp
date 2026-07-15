package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
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

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		helperJSONError(w, "Forbidden outside of dev", http.StatusForbidden)
		return
	}
	cfg.fileServerHits.Store(0)
	log.Println("reset file server hits")
	cfg.db.DeleteAllUsers(req.Context())
	log.Println("deleted all users")

	helperRespondWithJSON(struct{}{}, w, http.StatusOK)
}

func handlerValidateChirp(w http.ResponseWriter, req *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	type validResponse struct {
		CleanedBody string `json:"cleaned_body"`
	}

	params, err := helperReadJSON[chirp](req.Body)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to unmarshal json request: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	if len(params.Body) > 140 {
		helperJSONError(w, "Chirp is too long", http.StatusBadRequest)
		return
	}

	cleanedChirp := helperCleanOutput(params.Body)
	response := validResponse{CleanedBody: cleanedChirp}
	helperRespondWithJSON(response, w, http.StatusOK)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	params, err := helperReadJSON[parameters](req.Body)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to unmarshal json request: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	newUser, err := cfg.db.CreateUser(req.Context(), params.Email)
	if err != nil {
		errorMessage := fmt.Sprintf("error creating user: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}
	log.Printf("created user: %s", newUser)

	user := User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}

	helperRespondWithJSON(user, w, http.StatusCreated)
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

func helperJSONError(w http.ResponseWriter, msg string, statusCode int) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	payload := errorResponse{Error: msg}
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

func helperCleanOutput(body string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	inputWords := strings.Fields(body)
	cleanWords := []string{}

	for _, word := range inputWords {
		normalized := strings.ToLower(word)
		if slices.Contains(badWords, normalized) {
			word = "****"
		}
		cleanWords = append(cleanWords, word)
	}

	return strings.Join(cleanWords, " ")
}
