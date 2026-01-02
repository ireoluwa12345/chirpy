package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/ireoluwa12345/chirpy/internal/database"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	hits atomic.Int32
	db   *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.hits.Add(1)
		next.ServeHTTP(w, r)
	})
}

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
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := decoder.Decode(&params)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
	}

	id := uuid.New()

	user, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{ID: id, Email: params.Email, Password: params.Password})

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

func (cfg *apiConfig) HandleCreateChirp(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	param := struct {
		Body    string    `json:"body"`
		User_id uuid.UUID `json:"user_id"`
	}{}

	err := decoder.Decode(&param)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
		return
	}

	chirp, err := cfg.db.CreateChirp(context.Background(), database.CreateChirpParams{
		ID:     uuid.New(),
		UserID: param.User_id,
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
