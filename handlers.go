package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/ireoluwa12345/chirpy/internal/auth"
	"github.com/ireoluwa12345/chirpy/internal/database"
	_ "github.com/lib/pq"
)

const (
	accessTokenExpiry string = "86400s"
	// refreshTokenExpiry is created in hours
	refreshTokenExpiry int = 60 * 24
)

func (cfg *apiConfig) fileServerHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<html>
  <body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.hits.Load())
}

func (cfg *apiConfig) fileServerReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%d", cfg.hits.Swap(0))
}

func validateChirp(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	jsonData := struct {
		Body string `json:"body"`
	}{}
	err := jsonDecoder.Decode(&jsonData)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Something went wrong"}`))
		return
	}

	if len(jsonData.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Chirp is too long"}`))
		return
	}

	forbiddenWords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	bodyArray := strings.Split(jsonData.Body, " ")

	for i := 0; i < len(bodyArray); i++ {
		if ok := forbiddenWords[strings.ToLower(bodyArray[i])]; ok {
			bodyArray[i] = "****"
		}
	}

	result := strings.Join(bodyArray, " ")

	resp, _ := json.Marshal(map[string]string{
		"cleaned_body": result,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var params struct {
		Email    string `json:"email" validate:"required"`
		Password string `json:"password" validate:"required"`
	}
	err := decoder.Decode(&params)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
		return
	}

	validate := validator.New()
	err = validate.Struct(params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}

	id := uuid.New()

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}
	params.Password = hashedPassword

	user, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{ID: id, Email: params.Email, Password: hashedPassword})

	resp, err := json.Marshal(map[string]interface{}{
		"id":         user.ID,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
		"email":      user.Email,
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't marshal json}`))
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleLoginUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := decoder.Decode(&params)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
		return
	}

	user, err := cfg.db.GetUserByEmail(context.Background(), params.Email)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "No user found"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error fetching user by email:", err)
		w.Write([]byte(`{"error": "couldn't get user}`))
		return
	}

	authenticated, err := auth.VerifyPassword(params.Password, user.Password)

	if err != nil || !authenticated {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "email or password is incorrect"}`))
		return
	}

	expiresIn, err := time.ParseDuration(accessTokenExpiry)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	jwtToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiresIn)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	refreshToken, err := auth.MakeRefreshToken()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	expiresAt := time.Now().Add(time.Duration(refreshTokenExpiry) * time.Hour)

	storedRefreshToken, err := cfg.db.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"id":            user.ID,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"email":         user.Email,
		"token":         jwtToken,
		"refresh_token": storedRefreshToken.Token,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleCreateChirp(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	param := struct {
		Body string `json:"body"`
	}{}

	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting bearer token: %v", err)))
		return
	}

	user_id, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting validating token: %v", err)))
		return
	}

	err = decoder.Decode(&param)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
		return
	}

	chirp, err := cfg.db.CreateChirp(context.Background(), database.CreateChirpParams{
		ID:     uuid.New(),
		UserID: user_id,
		Body:   param.Body,
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error creating chirps:", err)
		w.Write([]byte(`{"error": "couldn't create chirp}`))
		return
	}

	resp, err := json.Marshal(chirp)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't marshal json}`))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps(context.Background())

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error fetching chirps:", err)
		w.Write([]byte(`{"error": "couldn't get chirps}`))
		return
	}

	resp, err := json.Marshal(chirps)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't marshal json}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleGetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)

	fmt.Println(chirpIDString)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid UUID}`))
		return
	}

	chirp, err := cfg.db.GetChirpByID(context.Background(), chirpID)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "chirp not found"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error fetching chirp by ID:", err)
		w.Write([]byte(`{"error": "couldn't get chirp}`))
		return
	}

	resp, err := json.Marshal(chirp)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't marshal json}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting bearer token: %v", err)))
		return
	}

	refreshToken, err := cfg.db.CheckRefreshToken(context.Background(), bearerToken)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	expiresIn, err := time.ParseDuration(accessTokenExpiry)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	accessToken, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtSecret, expiresIn)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"token": accessToken,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (cfg *apiConfig) handleRevoke(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting bearer token: %v", err)))
		return
	}

	err = cfg.db.RevokeRefreshToken(context.Background(), bearerToken)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) HandleUpdateUsers(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user_id, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting validating token: %v", err)))
		return
	}

	var params struct {
		Email    string
		Password string
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.db.UpdateUser(context.Background(), database.UpdateUserParams{
		Email:    params.Email,
		Password: hashedPassword,
		ID:       user_id,
	})

	resp, _ := json.Marshal(map[string]interface{}{
		"id":         user.ID,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
		"email":      user.Email,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (cfg *apiConfig) HandleDeleteChirps(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid UUID"}`))
		return
	}

	userID := r.Context().Value("user_id").(uuid.UUID)

	chirp, err := cfg.db.GetChirpByID(context.Background(), chirpID)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "chirp not found"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error fetching chirp by ID:", err)
		w.Write([]byte(`{"error": "couldn't get chirp"}`))
		return
	}

	if chirp.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "you can only delete your own chirps"}`))
		return
	}

	err = cfg.db.DeleteChirp(context.Background(), chirpID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error deleting chirp:", err)
		w.Write([]byte(`{"error": "couldn't delete chirp"}`))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
