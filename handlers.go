package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/PokeMasterCP/chirp/internal/auth"
	"github.com/PokeMasterCP/chirp/internal/database"
	"github.com/google/uuid"
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

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, req *http.Request) {
	chirpRequest, err := helperReadJSON[Chirp](req.Body)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to unmarshal json request: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		helperJSONError(w, "Authorization header not provided", http.StatusUnauthorized)
		return
	}

	loggedInUserID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("invalid JWT: %v", err)
		helperJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if len(chirpRequest.Body) > 140 {
		helperJSONError(w, "Chirp is too long", http.StatusBadRequest)
		return
	}

	cleanedText := helperCleanChirp(chirpRequest.Body)

	queryParams := database.CreateChirpParams{
		Body:   cleanedText,
		UserID: loggedInUserID,
	}

	chirpRecord, err := cfg.db.CreateChirp(req.Context(), queryParams)
	if err != nil {
		errorMessage := fmt.Sprintf("error creating chirp: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	response := Chirp{
		ID:        chirpRecord.ID,
		CreatedAt: chirpRecord.CreatedAt,
		UpdatedAt: chirpRecord.UpdatedAt,
		Body:      chirpRecord.Body,
		UserID:    chirpRecord.UserID,
	}

	helperRespondWithJSON(response, w, http.StatusCreated)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params, err := helperReadJSON[parameters](req.Body)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to unmarshal json request: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	if params.Password == "" {
		helperJSONError(w, "password is required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to hash password: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	newUserParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}

	newUser, err := cfg.db.CreateUser(req.Context(), newUserParams)
	if err != nil {
		errorMessage := fmt.Sprintf("error creating user: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}
	log.Printf("created user: %s", newUser)

	helperRespondWithJSON(User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}, w, http.StatusCreated)
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.db.GetAllChirps(req.Context())
	if err != nil {
		errorMessage := fmt.Sprintf("failed to query chirps db: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	apiChirps := []Chirp{}
	for _, dbChirp := range chirps {
		apiChirps = append(apiChirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		})
	}

	helperRespondWithJSON(apiChirps, w, http.StatusOK)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, req *http.Request) {
	chirpID := req.PathValue("chirpID")

	uuid, err := uuid.Parse(chirpID)
	if err != nil {
		errorMessage := "invalid UUID"
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusBadRequest)
		return
	}

	chirp, err := cfg.db.GetChirp(req.Context(), uuid)
	if err != nil {
		helperJSONError(w, "chirp not found", http.StatusNotFound)
		return
	}

	response := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	helperRespondWithJSON(response, w, http.StatusOK)
}

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		ExpiresIn int    `json:"expires_in"`
	}

	params, err := helperReadJSON[parameters](req.Body)
	if err != nil {
		errorMessage := fmt.Sprintf("failed to unmarshal json request: %s", err)
		log.Println(errorMessage)
		helperJSONError(w, errorMessage, http.StatusInternalServerError)
		return
	}

	if params.ExpiresIn <= 0 || params.ExpiresIn > 3600 {
		params.ExpiresIn = 3600
	}

	user, err := cfg.db.GetUserByEmail(req.Context(), params.Email)
	if err != nil {
		log.Printf("error getting user: %s", err)
		helperJSONError(w, "incorrect email or password", http.StatusUnauthorized)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		log.Printf("error checking password: %s", err)
		helperJSONError(w, "incorrect email or password", http.StatusUnauthorized)
		return
	}

	if !match {
		helperJSONError(w, "incorrect email or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Duration(params.ExpiresIn)*time.Second)
	if err != nil {
		log.Printf("error generating JWT: %s", err)
		helperJSONError(w, "failed to generate JWT", http.StatusInternalServerError)
		return
	}

	type loginResponse struct {
		User
		Token string `json:"token"`
	}

	response := loginResponse{
		User: User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		},
		Token: token,
	}

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

func helperCleanChirp(body string) string {
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
